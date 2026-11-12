package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// MASCOT COSTUMES (idea 252): in SOME parties 1-3 ordinary guests come in a big mascot suit: the dancing BEAR, the
// dancing COW, the DUCK or the DINOSAUR (generic animals, no brand). They are people in a suit, and it shows: the hood
// leaves the guest's own FACE out in front (eyes, mouth, skin are the guest's pixels, untouched), with the ears or horns
// on top and the suit over the whole body.
//
// It reuses the themed costumes of costume.go (idea 213) without duplicating them: the four suits are extra looks of
// costumeLooks (appended here, so the indexes of the themed ones never move) painted by dressFrame like any other look,
// in every view and animation and on the body when they die; the themed parties never pick them (they are not in
// costumeLevels). The dressed CharSet is cached per (base, look) in the event's state.
//
// The event "mascota" is a party event of the frame (partyevents.go) with its own dice (one party in four; see
// fursuitDue), never in STEALTH, the RETO DE HOY or the island. A few seconds into the party it dresses its guests and
// stays for the whole party:
//   - they dance now and then wherever they are (their own step: a sway from side to side with the arms of the dance
//     sheet, drawn through the drawGuest hook of the frame, never touching gu.pos) and say a short line;
//   - they walk a bit slower (Guest.walk asks fursuitSlow: fursuitSpeedK);
//   - killing one gives a few extra points (fursuitBonus, only when the kill is the player's) and bursts confetti and
//     tufts of fur; they are ordinary guests for every cap and goal (nobody is added: guests are dressed).
// ---------------------------------------------------------------------------

const (
	fursuitSpeedK = 0.80 // their walking speed against a plain guest
	fursuitBonus  = 150  // extra points for the player who kills one
	fursuitMin    = 1
	fursuitMax    = 3
)

var (
	fursuitLines = pevTexts("¡Abrazos gratis!", "¡Qué calor hace aquí dentro!", "¡Baila conmigo!", "¡Foto con la mascota!", "No veo nada por los ojos")
	fursuitCalls = map[string][]string{
		"oso":  pevTexts("¡Grrr... de broma!"),
		"vaca": pevTexts("¡Muuu!"),
		"pato": pevTexts("¡Cuac, cuac!"),
		"dino": pevTexts("¡Raaawr!"),
	}
	fursuitKillLine = pevTexts("¡Mascota!")
)

// fursuitKind: the colours and the head of one suit.
type fursuitKind struct {
	id                string
	fur, dark, belly  uint32
	ear, earIn, extra uint32
}

var fursuitKinds = []fursuitKind{
	{id: "oso", fur: 0x9a6232, dark: 0x5a3416, belly: 0xe0b888, ear: 0x9a6232, earIn: 0xe0a0a0},
	{id: "vaca", fur: 0xf2f0ea, dark: 0x9a968e, belly: 0xf4b6c4, ear: 0xf2f0ea, earIn: 0xf4b6c4, extra: 0x1c1a20},
	{id: "pato", fur: 0xf6d63a, dark: 0xb89418, belly: 0xfff0a0, ear: 0xf6d63a, earIn: 0xf08a20, extra: 0xf08a20},
	{id: "dino", fur: 0x4aae4e, dark: 0x2a6e2e, belly: 0xbfe08a, ear: 0xf0a030, earIn: 0xf0a030, extra: 0xf0a030},
}

func init() {
	for _, mk := range fursuitKinds {
		mk := mk
		costumeLooks = append(costumeLooks, costumeLook{id: mk.id, hat: true, face: true,
			paint: func(p *dressPen, d dressGeom, v int) { paintFursuit(p, d, mk) }})
	}
	registerPartyEvent(&partyEvent{ID: "mascota", Name: "Invitados disfrazados de mascota", Group: "mascot", Reactive: true,
		Prob: fursuitChance, AtMin: 2, AtMax: 25, Dur: 1e5,
		Eligible: fursuitDue, Start: fursuitStart, Update: fursuitUpdate, Items: fursuitItems})
}

// The suits are decided once per party with their own dice (fursuitDice, never g.rng nor the frame's): the event is
// "reactive" for the frame (no plan, it does not take one of the three places of a party) and its condition is "this
// party rolled the suits and their moment has come". -pevent mascota forces it.
const fursuitChance = 0.25

var (
	fursuitRng  *rand.Rand
	fursuitSeed int64 // tests: non-zero = deterministic dice
	fursuitLvl  *Level
	fursuitWant bool
	fursuitAt   float64
	fursuitDone bool
)

func fursuitDice() *rand.Rand {
	if fursuitRng == nil {
		sd := fursuitSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x0f0f5017
		}
		fursuitRng = rand.New(rand.NewSource(sd))
	}
	return fursuitRng
}

// fursuitFits: the parties where the suits may come (room to roam and a crowd).
func fursuitFits(t partyTraits) bool { return t.roam && t.guests >= 12 }

// fursuitDue: the condition of the reactive event. The first time it is asked in a party it rolls the dice.
func fursuitDue(g *Game, t partyTraits) bool {
	if g.lvl == nil {
		return false
	}
	if fursuitLvl != g.lvl {
		fursuitLvl, fursuitDone = g.lvl, false
		r := fursuitDice()
		roll, at := r.Float64(), 2+r.Float64()*23 // both always drawn
		fursuitWant = fursuitFits(t) && (roll < fursuitChance || dbgPartyEvent == "mascota")
		fursuitAt = g.time + at
		if dbgPartyEvent == "mascota" {
			fursuitAt = g.time + 3
		}
	}
	return fursuitWant && !fursuitDone && g.time >= fursuitAt
}

// fursuitLook: is this costume look one of the mascot suits? (0 = no costume)
func fursuitLook(look uint8) bool {
	if look == 0 || int(look) >= len(costumeLooks) {
		return false
	}
	id := costumeLooks[look].id
	for _, mk := range fursuitKinds {
		if mk.id == id {
			return true
		}
	}
	return false
}

// fursuitSlow: Guest.walk slows the guests in a suit down (game.go).
func fursuitSlow(gu *Guest) bool { return gu.cs != nil && fursuitLook(gu.cs.dress.look) }

// ---- painting ------------------------------------------------------------------------------------------

// mput: a pixel of the suit, only inside the frame (the pen never has to slide the costume: the suit covers the whole
// body, so a slide would leave a row of the clothes uncovered).
func mput(p *dressPen, x, y int, c uint32) {
	if x >= 0 && y >= 0 && x < p.b.w && y < p.b.h {
		p.set(x, y, c)
	}
}

// paintFursuit: the suit over the body, the hood around the face (left as it is) and the ears on top.
func paintFursuit(p *dressPen, d dressGeom, mk fursuitKind) {
	ey := d.eyeY()
	chin := d.hy + d.hh - 1
	// 1. the suit: every pixel of the body under the chin takes the fur; its outline (a pixel next to the empty
	// background) stays dark, so the shape still reads
	y0 := maxI(d.ty+1, chin+2)
	edge := func(x, y int) bool {
		return !p.solid(x-1, y) || !p.solid(x+1, y) || !p.solid(x, y-1) || !p.solid(x, y+1)
	}
	for y := y0; y < p.b.h; y++ {
		for x := 0; x < p.b.w; x++ {
			if !p.solid(x, y) {
				continue
			}
			c := mk.fur
			switch {
			case edge(x, y):
				c = mk.dark
			case x >= d.tx+d.tw-1 && d.vw != vBack:
				c = shadeHex(mk.fur, 0.88)
			}
			if mk.id == "vaca" && c == mk.fur && ((x/3)+2*(y/3))%4 == 1 {
				c = mk.extra // the black patches of the cow
			}
			mput(p, x, y, c)
		}
	}
	if d.vw == vFront { // the belly patch
		for y := y0 + 1; y < minI(p.b.h, y0+1+minI(6, maxI(2, d.th-4))); y++ {
			for x := d.cx() - 2; x <= d.cx()+2; x++ {
				if p.solid(x, y) && !edge(x, y) {
					mput(p, x, y, mk.belly)
				}
			}
		}
	}
	// 2. the hood: the top of the head (hair) and a rim around the face; the face itself is never painted
	top := maxI(1, d.hy-1)
	for y := top; y <= ey-2; y++ {
		for x := d.l() - 1; x <= d.r()+1; x++ {
			if y == top && (x == d.l()-1 || x == d.r()+1) {
				continue // round corners
			}
			mput(p, x, y, mk.fur)
		}
	}
	switch d.vw {
	case vFront:
		for y := ey - 1; y <= chin+1; y++ {
			mput(p, d.l()-1, y, shadeHex(mk.fur, 0.9))
			mput(p, d.r()+1, y, shadeHex(mk.fur, 0.9))
		}
		for x := d.l(); x <= d.r(); x++ {
			mput(p, x, chin+1, shadeHex(mk.fur, 0.9))
		}
	case vSide: // the face looks to the right: the back of the head is fur
		for y := ey - 1; y <= chin+1; y++ {
			for x := d.l() - 1; x < d.cx(); x++ {
				mput(p, x, y, mk.fur)
			}
			mput(p, d.r()+1, y, shadeHex(mk.fur, 0.9))
		}
	default: // back: all hood
		for y := ey - 1; y <= chin+1; y++ {
			for x := d.l() - 1; x <= d.r()+1; x++ {
				mput(p, x, y, mk.fur)
			}
		}
	}
	// 3. ears, horns, beak, spikes
	ty := maxI(0, top-2)
	switch mk.id {
	case "oso":
		for _, x := range []int{d.l() - 1, d.r()} {
			if d.vw == vSide && x != d.l()-1 {
				continue
			}
			mput(p, x, ty, mk.ear)
			mput(p, x+1, ty, mk.ear)
			mput(p, x, ty+1, mk.ear)
			mput(p, x+1, ty+1, mk.earIn)
		}
	case "vaca":
		mput(p, d.l()-1, ty+1, 0xf0e2b0) // horns
		mput(p, d.l()-2, ty, 0xf0e2b0)
		mput(p, d.r()+1, ty+1, 0xf0e2b0)
		mput(p, d.r()+2, ty, 0xf0e2b0)
		mput(p, d.l()-2, ty+2, mk.earIn) // ears
		mput(p, d.r()+2, ty+2, mk.earIn)
		mput(p, d.cx(), top, mk.extra) // a patch on the hood
		mput(p, d.cx()+1, top+1, mk.extra)
	case "pato":
		mput(p, d.cx(), ty, mk.fur) // the tuft
		mput(p, d.cx()+1, ty+1, mk.fur)
		if d.vw != vBack { // the beak is the visor of the hood, over the eyes
			x0, x1 := d.l(), d.r()
			if d.vw == vSide {
				x0, x1 = d.cx(), d.r()+2
			}
			for x := x0; x <= x1; x++ {
				mput(p, x, ey-2, mk.extra)
			}
		}
	case "dino":
		for k, x := range []int{d.cx() - 2, d.cx(), d.cx() + 2} {
			h := 1 + k%2
			for j := 0; j < h; j++ {
				mput(p, x, top-1-j, mk.extra)
			}
		}
	}
}

// ---- the event -----------------------------------------------------------------------------------------

type fursuitGuest struct {
	gu    *Guest
	kind  int
	paid  bool
	step  int     // its own dance (an animation of the dance sheet)
	next  float64 // next time it may start dancing
	talkT float64
}

type fursuitState struct {
	ms   []*fursuitGuest
	sets map[costumeSetKey]*CharSet
	fx   []fursuitPuff
}

// fursuitPuff: confetti and fur flying off a mascot that dies (drawn over it for a moment).
type fursuitPuff struct {
	pos Vec
	t0  float64
	fur color.RGBA
}

// fursuitEligible: an ordinary guest who can wear a suit (no hat or big hairdo: the hood goes on the head).
func (g *Game) fursuitEligible(gu *Guest) bool {
	if gu == nil || gu.cs == nil || gu.cs.Spec == nil || gu.cs.dress.look != 0 || gu.out() {
		return false
	}
	if gu.name != "" || gu.perf != "" || gu.keep || gu.quirk != "" || gu.py != nil || gu.famousOK || gu.vipCard {
		return false
	}
	n := gu.cs.Spec.Name
	return !isKidSlug(n) && !g.isFamous(n) && !tallHead(gu.cs)
}

func fursuitStart(g *Game, a *partyAct) bool {
	fursuitDone = true // one try per party: the reactive event never comes back in it
	r := partyDice()
	var pool []*Guest
	for _, gu := range g.guests {
		if g.fursuitEligible(gu) {
			pool = append(pool, gu)
		}
	}
	n := fursuitMin + r.Intn(fursuitMax-fursuitMin+1)
	if len(pool) < n+4 {
		return false
	}
	s := &fursuitState{sets: map[costumeSetKey]*CharSet{}}
	kinds := r.Perm(len(fursuitKinds))
	pick := r.Perm(len(pool))[:n]
	if dbgPartyEvent == "mascota" && g.p != nil { // -pevent mascota (debug): the ones nearest the player, to be seen at once
		sort.Slice(pool, func(i, j int) bool { return dist(pool[i].pos, g.p.pos) < dist(pool[j].pos, g.p.pos) })
		for i := range pick {
			pick[i] = i
		}
	}
	for i, k := range pick {
		gu := pool[k]
		kind := kinds[i%len(kinds)]
		if i == 0 && r.Float64() < 0.7 { // the bear and the cow are the classic ones
			kind = r.Intn(2)
		}
		li := uint8(costumeLookIdx(fursuitKinds[kind].id))
		key := costumeSetKey{gu.cs, dressKey{li, 0}}
		cs := s.sets[key]
		if cs == nil {
			cs = dressedCharSet(gu.cs, dressKey{li, 0})
			s.sets[key] = cs
		}
		gu.cs = cs
		s.ms = append(s.ms, &fursuitGuest{gu: gu, kind: kind, step: AnimDance0 + r.Intn(12), next: g.time + 2 + r.Float64()*8})
	}
	a.D = s
	fursuitStarted++
	return true
}

// fursuitStarted: parties where the suits came (tests).
var fursuitStarted int

func fursuitUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*fursuitState)
	r := partyDice()
	left := 0 // mascots still at the party
	for _, m := range s.ms {
		gu := m.gu
		if gu.dead && !m.paid {
			m.paid = true
			g.fursuitDies(s, m, r)
			continue
		}
		if gu.out() {
			continue
		}
		left++
		// now and then it dances wherever it is (only when it is free: never inside another event or scene)
		if g.time >= m.next && gu.state == gIdle && gu.timer < 1000 && lifeFree(gu) && !g.pevTaken(gu) && !g.inSelfie(gu) && !g.inContest(gu) {
			gu.state, gu.timer, gu.path, gu.moving, gu.dir = gDance, 5+r.Float64()*5, nil, false, DirDown
			m.next = g.time + 14 + r.Float64()*14
			if gu.icon == "" && r.Float64() < 0.6 {
				lines := fursuitLines
				if r.Float64() < 0.5 {
					lines = fursuitCalls[fursuitKinds[m.kind].id]
				}
				gu.icon, gu.iconT = pevPick(r, lines), 2
			}
		}
	}
	out := s.fx[:0]
	for _, f := range s.fx {
		if g.time-f.t0 < 1.4 {
			out = append(out, f)
		}
	}
	s.fx = out
	// the event ends (and frees its place in the frame) once no mascot is left and the last burst is over; while one is
	// alive it stays, because its dance and its bonus live here
	return left == 0 && len(s.fx) == 0
}

// fursuitDies: confetti and fur; the player's kill is worth a few extra points.
func (g *Game) fursuitDies(s *fursuitState, m *fursuitGuest, r *rand.Rand) {
	gu := m.gu
	mk := fursuitKinds[m.kind]
	fur := hx(mk.fur)
	s.fx = append(s.fx, fursuitPuff{pos: gu.pos, t0: g.time, fur: fur})
	for i := 0; i < 14; i++ {
		ang := r.Float64() * 2 * math.Pi
		sp := 25 + r.Float64()*35
		l := 0.6 + r.Float64()*0.5
		col := pevNoteCols[i%len(pevNoteCols)]
		if i%3 == 0 {
			col = fur
		}
		g.parts = append(g.parts, Particle{pos: Vec{gu.pos.X, gu.pos.Y - 14}, vel: Vec{math.Cos(ang) * sp, math.Sin(ang)*sp - 30}, life: l, max: l,
			col: col, size: 2, grav: 60})
	}
	byPlayer := false
	for i := len(g.bodies) - 1; i >= 0 && i >= len(g.bodies)-4; i-- {
		if b := g.bodies[i]; b.cs == gu.cs && b.byPlayer {
			byPlayer = true
		}
	}
	if byPlayer {
		g.award(gu.pos, fursuitBonus, fursuitKillLine[0])
	}
}

// drawGuest: the frame's hook in the guests' draw loop. A dancing mascot does its own step: it sways from side to side
// (whole pixels) and bobs, with the arms of its dance. Everything else is the normal drawing (the suit is in the frames).
func (s *fursuitState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	for _, m := range s.ms {
		if m.gu != gu {
			continue
		}
		if gu.state == gDance {
			sway := math.Round(math.Sin(g.time*3.2+gu.off) * 2)
			bob := -math.Abs(math.Round(math.Sin(g.time*6.4+gu.off) * 2))
			dir := DirDown
			if int(g.time*0.8+gu.off)%4 == 3 {
				dir = DirLeft
				if sway > 0 {
					dir = DirRight
				}
			}
			g.drawActor(dst, gu.cs, m.step, g.time*5, Vec{pos.X + sway, pos.Y}, dir, bob, false)
			return true
		}
		if gu.moving && gu.state == gGoto { // a heavy waddle
			anim, ph := g.guestPose(gu)
			g.drawActor(dst, gu.cs, anim, ph, Vec{pos.X + math.Round(math.Sin(g.time*7+gu.off)), pos.Y}, g.gaitDir(gu), 0, false)
			return true
		}
		return false
	}
	return false
}

// fursuitItems: the burst of a mascot that died (a ring of confetti that opens and falls).
func fursuitItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*fursuitState)
	plot := g.plotter(dst)
	px := g.pevPx()
	for _, f := range s.fx {
		f := f
		items = append(items, drawItem{f.pos.Y + 0.5, func() {
			u := (g.time - f.t0) / 1.4
			cx, cy := g.pevXY(f.pos)
			for i := 0; i < 10; i++ {
				ang := float64(i) * 2 * math.Pi / 10
				rad := 6 + u*20
				x := cx + int(math.Round(math.Cos(ang)*rad))
				y := cy - 16 + int(math.Round(math.Sin(ang)*rad*0.6+u*u*18))
				col := pevNoteCols[i%len(pevNoteCols)]
				if i%2 == 0 {
					col = f.fur
				}
				plot(x, y, 2*px, 2*px, col)
			}
		}})
	}
	return items
}
