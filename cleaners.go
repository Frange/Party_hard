package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// CLEANING CREW (user, request 233: "Limpiadores ok, que vayan de un verde clarito y que se distinga que no son
// militares"). A "sometimes" party event (partyevents.go, id "cleaners"): when a party already has a mess (bodies or
// pools of blood), 1-3 cleaners come in from the edge of the level, pick up the bodies that may be picked up, mop the
// blood and the sick, and leave the way they came.
//
//   - LOOK: mint green overalls (shirt and trousers, 0x9ce8c6 / 0x6ccaa2) with the reflective yellow-striped vest, a mint
//     cap, white rubber boots and a mop; while mopping a yellow bucket stands beside them. The military are olive
//     (0x3a4a2a / 0x2a3a22) with an armour vest and a helmet: nothing in common. No logo, no brand.
//   - THEY ARE GUESTS: each one is a plain *Guest (counted in g.total and the live cap: they only come if guestRoom()
//     has places), so they can be killed like anyone (a normal kill, with its witnesses and points) and, while they
//     are alive, they see like anyone: a cleaner who sees a body nobody has reported warns the police through the
//     usual rules (guestScan: no stars for bodies the player did not kill, starsTarget). When one leaves alive he is
//     `gone` and g.total goes down by one, so BAJAS x/y keeps its x (x = total - alive): picking up bodies never
//     changes the kills counted.
//   - WHICH BODIES: only one that lies there free (not carried, not bagged by the police, not claimed by a cop, not
//     falling, not being taken by the dog) and that the police already know about OR that the player did not kill
//     (an unreported kill of the player is evidence: a cleaner reports it, never hides it). Hidden bodies are not in
//     g.bodies any more; a body next to a dead player (the ambulance, revive.go) is never touched.
//   - WHAT THEY DO: walk to the body, mop around it (the blood of that spot goes), put it in a bag and carry it to
//     their exit, where it is taken away (removeBody). Then the next one; then up to three stains of blood/sick
//     without a body; then out.
//   - SAFETY: whatever takes a cleaner away (a fright, a call to the police) makes him drop the bag where he is; when
//     he is free again the crew takes him back. A cleaner who does not move for cleanStuckT seconds while walking is
//     sent again; if it happens twice he leaves the party at once (vanishes through the exit with a puff). After
//     cleanShiftT seconds the crew goes home.
//
// All the dice are partyDice() (g.rng is never moved). Texts with pevTexts and their English in lang_poolclean.go.
// ---------------------------------------------------------------------------

const (
	cleanMopT   = 2.4   // seconds mopping a spot
	cleanStuckT = 6.0   // seconds without moving while walking: sent again (twice: he leaves)
	cleanShiftT = 110.0 // the crew goes home after this
	cleanLeaveT = 35.0  // seconds to walk out; then he just vanishes at the door
	cleanStains = 3     // stains without a body each cleaner mops at most
)

var (
	cleanMint   = hx(0x9ce8c6)
	cleanMintDk = hx(0x6ccaa2)
	cleanBoots  = hx(0xeef2f0)
	cleanBucket = color.RGBA{0xf0, 0xc8, 0x28, 255}

	cleanArrive = pevTexts("Equipo de limpieza, ¡paso!", "¡Venimos a limpiar!")
	cleanLines  = pevTexts("Aquí no ha pasado nada", "Otro más para la bolsa", "Esto sale con lejía",
		"¡Cuidado, suelo mojado!", "Siempre igual en estas fiestas", "Cobro por horas, eh")
	cleanSay = pevTexts("Llega el equipo de limpieza")
)

// cleanerSpec: a cleaner of the crew (dice r; the face and hair change, the uniform does not).
func cleanerSpec(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Top, s.Bottom = TopVest, BotPants
	s.Shirt, s.Shirt2, s.Pants, s.Shoes = cleanMint, cleanMint, cleanMintDk, cleanBoots
	s.Hat, s.HatCol, s.Accent = HatCap, cleanMint, cleanMintDk
	s.Glasses, s.Mask, s.Cape, s.Susp = GlassNone, color.RGBA{}, color.RGBA{}, color.RGBA{}
	s.Pattern, s.Weapon, s.Ride, s.Censor = 0, 0, 0, false
	s.Body = []int{BodyNormal, BodyNormal, BodySlim, BodyBig}[r.Intn(4)]
	s.Carry, s.CarryCol = HoldBroom, cleanMint
	return s
}

const (
	clGo    = iota // walking to a body or a stain
	clMop          // mopping there
	clCarry        // carrying the bag to the exit
	clLeave        // going home
	clGone         // left (or dead)
)

type cleanBot struct {
	mem    *pevMem
	cs     *CharSet // with the mop
	csBag  *CharSet // hands free for the bag
	phase  int
	body   *Body // the body he is on his way to / carrying
	carry  bool  // the bag is on his shoulders
	stain  Vec   // the stain he is on his way to (body == nil)
	t      float64
	t0     float64 // game time the phase began (walks)
	still  float64 // seconds without moving while walking
	last   Vec
	resent int
	stains int
	busyAt float64 // game time of his last line
}

type cleanState struct {
	bots []*cleanBot
	exit Vec
	t0   float64
	// kept for the tests
	removed, mopped, vanished int
}

func (s *cleanState) pevHas(gu *Guest) bool {
	for _, b := range s.bots {
		if b.mem.gu == gu && b.phase != clGone {
			return true
		}
	}
	return false
}

func init() {
	registerPartyEvent(&partyEvent{ID: "cleaners", Name: "Equipo de limpieza", Group: "crew", Prob: 0.15, AtMin: 120, AtMax: 330, Dur: 75,
		Cooldown: 200, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && t.icon != "" && !cleanNoIcons[t.icon] },
		Start:    cleanStart, Update: pevGuard(cleanUpdate, cleanEnd), Items: cleanItems})
}

// cleanNoIcons: parties where a cleaning crew makes no sense (space, a ship at sea, the motorway, the train).
var cleanNoIcons = map[string]bool{"station": true, "pirate": true, "highway": true, "train": true, "boat": true}

// cleanBodyOK: a body a cleaner may pick up (see the header).
func cleanBodyOK(g *Game, b *Body) bool {
	if b == nil || b.carried || b.bagged || b.fallT > 0 || b.claimed != nil || g.bodyHeld(b) || (b.byPlayer && !b.reported) { // an ambulance's body stays (ambulance.go)
		return false
	}
	found := false
	for _, x := range g.bodies {
		if x == b {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	for _, p := range g.players() {
		if p != nil && p.dead && dist(p.pos, b.pos) < 40 { // the ambulance may come for him (revive.go)
			return false
		}
	}
	return true
}

func (s *cleanState) bodyTaken(b *Body, except *cleanBot) bool {
	for _, o := range s.bots {
		if o != except && o.phase != clGone && o.body == b {
			return true
		}
	}
	return false
}

// cleanStainOK: a stain of blood or sick with no body on it.
func cleanStainOK(g *Game, d Decal) bool {
	if d.kind != 0 && d.kind != 2 {
		return false
	}
	for _, b := range g.bodies {
		if dist(b.pos, d.pos) < 20 {
			return false
		}
	}
	return g.lvl.Free(d.pos) && !g.lvl.InPool(d.pos)
}

func cleanMess(g *Game) (bodies, stains int) {
	for _, b := range g.bodies {
		if cleanBodyOK(g, b) {
			bodies++
		}
	}
	for _, d := range g.decals {
		if cleanStainOK(g, d) {
			stains++
		}
	}
	return
}

func cleanStart(g *Game, a *partyAct) bool {
	r := partyDice()
	if dbgPartyEvent == "cleaners" && g.time < 10 {
		cleanDebugMess(g, r)
	}
	bodies, stains := cleanMess(g)
	if bodies == 0 && stains < 2 {
		return false
	}
	n := minI(1+r.Intn(3), g.guestRoom())
	if bodies > 0 {
		n = minI(n, bodies+1)
	}
	if n < 1 {
		return false
	}
	near := g.p.pos
	for _, b := range g.bodies {
		if cleanBodyOK(g, b) {
			near = b.pos
			break
		}
	}
	exit, ok := dogEdge(g, r, near, nil, 0)
	if !ok {
		return false
	}
	s := &cleanState{exit: exit, t0: g.time}
	for i := 0; i < n; i++ {
		spec := cleanerSpec(r)
		bag := spec
		bag.Carry = HoldNone
		pos := g.lvl.NearestFree(Vec{exit.X + float64(i%2)*10 - 5, exit.Y + float64(i/2)*8})
		if !g.lvl.Free(pos) || g.lvl.PathG(pos, near) == nil {
			pos = exit
		}
		gu := &Guest{pos: pos, dir: DirDown, cs: newCharSet(&spec), style: r.Intn(4), off: r.Float64() * 6, state: gIdle, timer: 9999, role: r.Intn(2), keep: true}
		g.guests = append(g.guests, gu)
		g.total++
		b := &cleanBot{mem: &pevMem{gu: gu, want: gIdle}, cs: gu.cs, csBag: newCharSet(&bag), phase: clGo, t0: g.time, last: pos}
		s.bots = append(s.bots, b)
	}
	a.D = s
	for i, b := range s.bots {
		cleanNext(g, s, b)
		if i == 0 {
			b.mem.gu.icon, b.mem.gu.iconT = pevPick(r, cleanArrive), 2.5
			b.busyAt = g.time
		}
	}
	g.say(cleanSay[0])
	return true
}

// cleanDebugMess: -pevent cleaners: two bodies nobody killed and a stain, so the capture has something to clean.
func cleanDebugMess(g *Game, r *rand.Rand) {
	if b, _ := cleanMess(g); b > 0 {
		return
	}
	for i := 0; i < 2; i++ {
		sp, ok := g.pevSpot(r, 14, false)
		if !ok {
			continue
		}
		cs := g.p.cs
		if len(g.guests) > 0 {
			cs = g.guests[r.Intn(len(g.guests))].cs
		}
		g.spawnBody(sp, cs, DirDown)
		nb := g.bodies[len(g.bodies)-1]
		nb.fallT, nb.byPlayer, nb.reported = 0, false, true
	}
	if sp, ok := g.pevSpot(r, 10, false); ok {
		g.decals = append(g.decals, Decal{pos: sp, kind: 0, r: 10})
	}
}

// cleanNext: the bot chooses his next job: the nearest free body, a stain, or home.
func cleanNext(g *Game, s *cleanState, b *cleanBot) {
	gu := b.mem.gu
	b.body, b.carry, b.t, b.t0, b.still, b.last = nil, false, 0, g.time, 0, gu.pos
	if g.time-s.t0 < cleanShiftT {
		var best *Body
		for _, x := range g.bodies {
			if cleanBodyOK(g, x) && !s.bodyTaken(x, b) && (best == nil || dist(gu.pos, x.pos) < dist(gu.pos, best.pos)) {
				best = x
			}
		}
		if best != nil && g.pevSendTo(b.mem, g.lvl.NearestFree(Vec{best.pos.X + 9, best.pos.Y + 2})) {
			b.body, b.phase = best, clGo
			return
		}
		if b.stains < cleanStains {
			bd := math.Inf(1)
			var at Vec
			for _, d := range g.decals {
				if !cleanStainOK(g, d) || s.stainTaken(d.pos, b) {
					continue
				}
				if dd := dist(gu.pos, d.pos); dd < bd {
					bd, at = dd, d.pos
				}
			}
			if !math.IsInf(bd, 1) && g.pevSendTo(b.mem, at) {
				b.stain, b.phase = at, clGo
				return
			}
		}
	}
	b.phase = clLeave
	if !g.pevSendTo(b.mem, s.exit) {
		cleanVanish(g, s, b)
	}
}

func (s *cleanState) stainTaken(p Vec, except *cleanBot) bool {
	for _, o := range s.bots {
		if o != except && o.phase == clGo && o.body == nil && dist(o.stain, p) < 16 {
			return true
		}
	}
	return false
}

// cleanDrop: he lets the bag down where he is (it stays bagged: the police may still collect it).
func cleanDrop(g *Game, b *cleanBot) {
	if b.carry && b.body != nil {
		b.body.carried = false
		b.body.pos = g.lvl.NearestFree(b.mem.gu.pos)
	}
	b.carry = false
}

// cleanVanish: he leaves the party at once (stuck, or the shift is over): gone, and out of the kill target.
func cleanVanish(g *Game, s *cleanState, b *cleanBot) {
	gu := b.mem.gu
	cleanDrop(g, b)
	b.body = nil
	if !gu.out() {
		pevPuff(g, gu.pos, 10, partyDice())
		gu.gone, gu.icon = true, ""
		g.total--
		s.vanished++
		g.checkDone()
	}
	b.phase = clGone
}

func cleanUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*cleanState)
	r := partyDice()
	alive := 0
	for _, b := range s.bots {
		if b.phase == clGone {
			continue
		}
		gu := b.mem.gu
		if gu.out() {
			cleanDrop(g, b)
			b.body, b.phase = nil, clGone
			continue
		}
		alive++
		if !cleanOurs(b) {
			// a fright, a phone call...: the bag goes down; when he is free again he goes back to work
			cleanDrop(g, b)
			if cleanFree(g, b) {
				if b.body != nil && b.body.bagged && !b.body.carried && b.body.claimed == nil && !g.bodyHeld(b.body) && g.pevSendTo(b.mem, b.body.pos) {
					b.phase, b.t0, b.still, b.last = clCarry, g.time, 0, gu.pos // back to the bag he left
					b.carry = false
				} else {
					b.body = nil
					cleanNext(g, s, b)
				}
			}
			continue
		}
		switch b.phase {
		case clGo:
			if b.body != nil && !cleanBodyOK(g, b.body) {
				cleanNext(g, s, b)
				break
			}
			if cleanArrived(b) {
				b.mem.hold(gIdle)
				b.phase, b.t = clMop, 0
				if b.body != nil {
					gu.dir = faceTo(gu.pos, b.body.pos)
				}
				if g.time-b.busyAt > 8 && r.Float64() < 0.4 {
					gu.icon, gu.iconT, b.busyAt = pevPick(r, cleanLines), 2.2, g.time
				}
				break
			}
			cleanWatch(g, s, b, dt, func() { cleanNext(g, s, b) })
		case clMop:
			b.mem.keep()
			b.t += dt
			if b.body != nil && !cleanBodyOK(g, b.body) {
				cleanNext(g, s, b)
				break
			}
			if int(b.t*6)%4 == 0 && r.Float64() < dt*8 {
				at := gu.pos
				if b.body != nil {
					at = b.body.pos
				}
				g.parts = append(g.parts, Particle{pos: Vec{at.X + (r.Float64()-0.5)*14, at.Y - 1}, vel: Vec{(r.Float64() - 0.5) * 16, -10 - r.Float64()*14},
					life: 0.5, max: 0.5, col: color.RGBA{0xe8, 0xfc, 0xff, 255}, size: 2, grav: -6})
			}
			if b.t < cleanMopT {
				break
			}
			spot := b.stain
			if b.body != nil {
				spot = b.body.pos
			}
			s.mopped += cleanMop(g, spot, 22)
			if b.body == nil {
				b.stains++
				cleanNext(g, s, b)
				break
			}
			// into the bag and away
			b.body.bagged, b.body.carried = true, true
			b.carry = true
			b.phase, b.t0, b.still, b.last = clCarry, g.time, 0, gu.pos
			g.snd.Play("thud")
			if !g.pevSendTo(b.mem, s.exit) {
				cleanDrop(g, b)
				cleanVanish(g, s, b)
			}
		case clCarry:
			if !b.carry { // on his way back to the bag he dropped
				if b.body == nil || !b.body.bagged || b.body.carried || b.body.claimed != nil || g.bodyHeld(b.body) || !cleanInBodies(g, b.body) {
					cleanNext(g, s, b)
					break
				}
				if cleanArrived(b) {
					b.body.carried, b.carry = true, true
					b.t0, b.still = g.time, 0
					if !g.pevSendTo(b.mem, s.exit) {
						cleanVanish(g, s, b)
					}
					break
				}
				cleanWatch(g, s, b, dt, func() { cleanVanish(g, s, b) })
				break
			}
			if !cleanInBodies(g, b.body) { // someone else took it (it cannot happen while carried, but be safe)
				b.carry = false
				cleanNext(g, s, b)
				break
			}
			b.body.pos = gu.pos
			if cleanArrived(b) {
				g.removeBody(b.body)
				s.removed++
				pevPuff(g, gu.pos, 8, r)
				b.body, b.carry = nil, false
				cleanNext(g, s, b)
				break
			}
			cleanWatch(g, s, b, dt, func() { cleanVanish(g, s, b) })
		case clLeave:
			if cleanArrived(b) || g.time-b.t0 > cleanLeaveT {
				cleanVanish(g, s, b)
				break
			}
			cleanWatch(g, s, b, dt, func() { cleanVanish(g, s, b) })
		}
	}
	return alive == 0
}

func cleanInBodies(g *Game, b *Body) bool {
	for _, x := range g.bodies {
		if x == b {
			return true
		}
	}
	return false
}

// cleanOurs: the crew still has him (walking where it sent him, or held still).
func cleanOurs(b *cleanBot) bool {
	gu := b.mem.gu
	if gu.out() || gu.inside != nil || gu.vipStun > 0 || gu.lift > 0 {
		return false
	}
	return gu.state == gGoto && gu.next == gIdle && gu.dur >= 9999 || gu.state == gIdle && gu.timer > 1000
}

// cleanFree: he is back to doing nothing in particular (the crew may take him again).
func cleanFree(g *Game, b *cleanBot) bool {
	gu := b.mem.gu
	if !lifeFree(gu) || g.reporterActive(gu) || gu.sawKiller && gu.state != gIdle {
		return false
	}
	switch gu.state {
	case gIdle, gGoto, gDance, gLounge, gDrink, gSwim, gChat, gWaitP, gChatGo:
		return true
	}
	return false
}

// cleanArrived: he stopped at the end of the walk the crew gave him.
func cleanArrived(b *cleanBot) bool {
	gu := b.mem.gu
	return gu.state == gIdle && gu.timer > 1000
}

// cleanWatch: the stuck watch of a walking cleaner (sent again once, then out).
func cleanWatch(g *Game, s *cleanState, b *cleanBot, dt float64, giveUp func()) {
	gu := b.mem.gu
	if dist(gu.pos, b.last) > 3 {
		b.last, b.still = gu.pos, 0
		return
	}
	b.still += dt
	if b.still < cleanStuckT {
		return
	}
	b.still = 0
	b.resent++
	if b.resent >= 2 {
		cleanVanish(g, s, b)
		return
	}
	target := s.exit
	switch {
	case b.phase == clGo && b.body != nil:
		target = g.lvl.NearestFree(Vec{b.body.pos.X + 9, b.body.pos.Y + 2})
	case b.phase == clGo:
		target = b.stain
	case b.phase == clCarry && !b.carry && b.body != nil:
		target = b.body.pos
	}
	if !g.pevSendTo(b.mem, target) {
		giveUp()
	}
}

// cleanMop: the stains of blood and sick within rad of p go. It returns how many.
func cleanMop(g *Game, p Vec, rad float64) int {
	n := 0
	out := g.decals[:0]
	for _, d := range g.decals {
		if (d.kind == 0 || d.kind == 2) && dist(d.pos, p) < rad {
			n++
			continue
		}
		out = append(out, d)
	}
	g.decals = out
	return n
}

func cleanEnd(g *Game, a *partyAct) {
	s, ok := a.D.(*cleanState)
	if !ok {
		return
	}
	for _, b := range s.bots {
		if b.phase != clGone {
			cleanVanish(g, s, b)
		}
	}
}

// drawGuest: a cleaner mopping (the mop sways, the bucket beside him) or carrying the bag (pe_fx.go asks).
func (s *cleanState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	var b *cleanBot
	for _, x := range s.bots {
		if x.mem.gu == gu && x.phase != clGone {
			b = x
		}
	}
	if b == nil || gu.out() {
		return false
	}
	switch {
	case b.carry:
		anim, phase := AnimCarry, 0.0
		if gu.moving {
			phase = gu.anim
		}
		g.drawActor(dst, b.csBag, anim, phase, pos, gu.dir, 0, g.lvl.InPool(gu.pos))
		bi := g.as.bag
		if bi != nil {
			w, h := bi.Bounds().Dx(), bi.Bounds().Dy()
			g.drawImg(dst, bi, g.wx(pos.X), g.wy(pos.Y)-32, float64(w)/2, float64(h))
		}
		return true
	case b.phase == clMop && cleanOurs(b):
		sway := math.Round(math.Sin(g.time*9+gu.off) * 1.5)
		side := 1.0
		if gu.dir == DirLeft {
			side = -1
		}
		drawBucketAt(dst, g, Vec{pos.X - side*9, pos.Y})
		g.drawActor(dst, b.cs, AnimIdle, g.time*1.3, Vec{pos.X + sway, pos.Y}, gu.dir, 0, false)
		return true
	}
	return false
}

// drawBucketAt: a yellow mop bucket with a little water (whole pixels).
func drawBucketAt(dst *ebiten.Image, g *Game, at Vec) {
	x, y := math.Round(g.wx(at.X)), math.Round(g.wy(at.Y))
	fillRect(dst, x-4, y-7, 8, 7, cleanBucket)
	fillRect(dst, x-4, y-7, 8, 1, hx(0xfff0a0))
	fillRect(dst, x-3, y-6, 6, 1, hx(0x7cc8e8))
	fillRect(dst, x+3, y-6, 1, 6, hx(0xb89010))
	fillRect(dst, x-4, y-1, 8, 1, hx(0x8a6a08))
	fillRect(dst, x-3, y-10, 1, 3, hx(0x606870)) // the handle
	fillRect(dst, x+2, y-10, 1, 3, hx(0x606870))
	fillRect(dst, x-3, y-10, 6, 1, hx(0x606870))
}

// cleanItems: wet floor shine where they mopped a moment ago is the particles; nothing else to draw here.
func cleanItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem { return items }

// isBot: he is (or was) one of the crew.
func (s *cleanState) isBot(gu *Guest) bool {
	for _, b := range s.bots {
		if b.mem.gu == gu {
			return true
		}
	}
	return false
}
