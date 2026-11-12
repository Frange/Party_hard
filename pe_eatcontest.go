package main

import (
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "comer" (CONCURSO DE COMER, idea 283): 2-4 guests meet at a long folding table with a checked cloth and a
// chalkboard "CONCURSO DE COMER", and eat burgers and hot dogs against each other while a few others gather round and
// cheer. While they compete they are FAT (pe_eatfat.go: belly, hips, cheeks; only the drawing). They eat in bites (the
// food goes to the mouth, the cheeks puff up while they chew, a number on the cloth counts each one's score); at the end
// the one who ate least chokes and throws up (the vomit stain of the game) and the one who ate most celebrates.
// They are plain guests: they can be killed, they see and report. If one of them dies or runs off, or there is a new body
// near or the police are called, everybody runs away and the contest is over; the fat look always goes back.
//
// WHEN: barbecues, squares, fairs, the festival, the house and the birthday garden; 11 % of those parties,
// between 50 and 240 s, ONCE per party. Big (a crowd gathers).
// Capture: -pevent comer (near the player), comer:show (four at the table, no alarm breaks it), comer:kill (a contestant is stabbed).
// ---------------------------------------------------------------------------

const (
	eatChance  = 0.11
	eatSlotGap = 26.0 // between two contestants (world px)
	eatWalkMax = 16.0
	eatMinLen  = 16.0
	eatMaxLen  = 24.0
	eatEndLen  = 6.0
	eatBiteT   = 0.45 // the food at the mouth
	eatChewT   = 1.05 // cheeks full until here (from the bite)
	eatStack   = 12   // food on each plate at the start
)

// where it fits: the house, the birthday garden, Italy (square), the fair, the festival; and any party with
// a barbecue
var eatIcons = map[string]bool{"house": true, "cake": true, "italy": true, "funfair": true, "festival": true}

var (
	eatSign   = pevTexts("CONCURSO DE COMER")
	eatGo     = pevTexts("¡Que empiece el concurso!", "¡Preparados, listos... a comer!", "¡Hamburguesas y perritos para todos!")
	eatBites  = pevTexts("¡Ñam!", "¡Uno más!", "¡Otro!", "¡Esto no es nada!", "Mmmf...")
	eatCheers = pevTexts("¡Come, come!", "¡Vamos, vamos!", "¡Otro perrito!", "¡No puede más!", "¡Qué máquina!")
	eatWin    = pevTexts("¡CAMPEÓN!", "¡Que traigan más!", "¡Invicto!")
	eatChoke  = pevTexts("¡Glup!", "No puedo más...", "Me... me va a salir...")
	eatRest   = pevTexts("¡Revancha!", "¡Estaba frío!", "Uf, qué lleno...")
)

const (
	ecWalk = iota
	ecEat
	ecEnd
	ecOver
)

type eatPart struct {
	m             *pevMem
	orig, fat     *CharSet
	eaten         int
	rate          float64 // seconds between bites
	nextBite      float64
	biteT         float64 // time of the last bite
	food          int     // 0 burger, 1 hot dog (the one he is eating now)
	arrived       bool
	stature       float64
	skin          color.RGBA
	choked, vomit bool
}

type eatState struct {
	parts   []*eatPart
	crowd   pevCrowd
	spot    Vec // the middle of the front edge of the table
	tableW  float64
	signL   bool // the chalkboard stands on the left of the table
	phase   int
	t0      float64
	eatLen  float64
	winner  *eatPart
	loser   *eatPart
	bodies  int
	police  bool
	fled    bool
	demoK   bool
	cheerAt float64
}

func (s *eatState) pevHas(gu *Guest) bool {
	for _, p := range s.parts {
		if p.m.gu == gu {
			return true
		}
	}
	return s.crowd.pevHas(gu)
}

func init() {
	registerPartyEvent(&partyEvent{ID: "comer", Name: "Concurso de comer", Group: "eat", Big: true,
		Prob: eatChance, AtMin: 50, AtMax: 240, Dur: 40, MaxRuns: 1,
		Eligible: func(g *Game, t partyTraits) bool {
			return (t.bbq && t.outdoor || eatIcons[t.icon]) && t.roam && t.guests >= 20
		},
		Start: eatStart, Items: eatItems,
		Update: pevGuard(eatUpdate, func(g *Game, a *partyAct) { eatEnd(g, a.D.(*eatState)) })})
}

// eatSpotOK: room for the table, the contestants behind it, the chalkboard at one side and the audience in front.
func (g *Game) eatSpotOK(p Vec, w float64) (ok, signL bool) {
	l := g.lvl
	for _, dx := range []float64{-w / 2, -w / 4, 0, w / 4, w / 2} {
		for _, dy := range []float64{-6, 6, 26} {
			q := Vec{p.X + dx, p.Y + dy}
			if !l.Free(q) || l.SolidP(q.X, q.Y) || l.badGround(q) || l.InPool(q) {
				return false, false
			}
		}
	}
	// nothing of the level drawn where the table, the eaters and the chalkboard go (a phone box behind the board)
	clear := func(r Rect) bool {
		for _, pr := range l.Props {
			sc := 1.0
			if pr.scale > 0 {
				sc = pr.scale
			}
			pw, ph := float64(pr.K.W)*sc, float64(pr.K.H)*sc
			if r.Hits(Rect{pr.X - pw/2, pr.Y - ph, pw, ph}) || r.Hits(pr.Foot) {
				return false
			}
		}
		return true
	}
	if !clear(Rect{p.X - w/2 - 10, p.Y - 16, w + 20, 26}) { // the table (the eaters stand in front of what is behind, as anybody)
		return false, false
	}
	free := func(x float64) bool {
		if !clear(Rect{x - 40, p.Y - 50, 80, 58}) {
			return false
		}
		for _, dx := range []float64{-40, 0, 40} { // the board is not painted over a wall
			if q := (Vec{x + dx, p.Y - 50}); l.SolidP(q.X, q.Y) {
				return false
			}
		}
		for _, dx := range []float64{-14, 0, 14} {
			q := Vec{x + dx, p.Y + 2}
			if !l.Free(q) || l.SolidP(q.X, q.Y) || l.badGround(q) || l.InPool(q) {
				return false
			}
		}
		return true
	}
	switch {
	case free(p.X - w/2 - 46):
		return true, true
	case free(p.X + w/2 + 46):
		return true, false
	}
	return false, false
}

// eatCanBe: a guest who may sit at the table (any ordinary guest who is free).
func eatCanBe(g *Game, gu *Guest) bool {
	return peleaCanBe(g, gu) && gu.cs != nil && gu.cs.Spec != nil && !gu.cs.fat && gu.seat == nil && fatRoom(gu.cs)
}

func eatStart(g *Game, a *partyAct) bool {
	r := partyDice()
	n := 2 + r.Intn(3)
	if pevDemo("comer") != "" { // capture (-pevent comer:show, comer:kill): a full table
		n = 4
	}
	w := float64(n)*eatSlotGap + 14
	var spot Vec
	var signL, ok bool
	for try := 0; try < 80 && !ok; try++ {
		var p Vec
		var got bool
		if pevDebugOn("comer") && try < 20 {
			p, got = g.pevNearPlayerSpot(30, false)
		} else {
			p, got = g.pevSpot(r, 30, false)
		}
		if !got {
			continue
		}
		for k := n; k >= 2 && !ok; k-- { // a shorter table where four do not fit
			if ok, signL = g.eatSpotOK(p, float64(k)*eatSlotGap+14); ok {
				spot, n, w = p, k, float64(k)*eatSlotGap+14
			}
		}
	}
	if !ok {
		return false
	}
	var cand []*Guest
	for _, gu := range g.guests {
		if eatCanBe(g, gu) && (dist(gu.pos, spot) < 400 || pevDebugOn("comer")) {
			cand = append(cand, gu)
		}
	}
	if len(cand) < n+2 { // the contestants and somebody to watch them: fewer at the table, or none
		n = len(cand) - 2
	}
	if n < 2 {
		return false
	}
	sort.SliceStable(cand, func(i, j int) bool { return dist(cand[i].pos, spot) < dist(cand[j].pos, spot) })
	s := &eatState{spot: spot, tableW: w, signL: signL, t0: g.time, bodies: len(g.bodies), police: g.policeCalled}
	for _, gu := range cand {
		if len(s.parts) >= n {
			break
		}
		i := len(s.parts)
		pt := Vec{spot.X - float64(n-1)*eatSlotGap/2 + float64(i)*eatSlotGap, spot.Y - 5}
		m := &pevMem{gu: gu, pt: pt}
		if !g.pevSendTo(m, pt) {
			continue
		}
		p := &eatPart{m: m, orig: gu.cs, rate: 1.0 + r.Float64()*0.9, food: r.Intn(2), skin: color.RGBA{0xf0, 0xc0, 0x98, 255}}
		if sp := gu.cs.Spec; sp != nil {
			p.stature = float64(sp.Stature)
			if sp.Skin.A != 0 {
				p.skin = sp.Skin
			}
		}
		p.fat = fatCharSet(gu.cs)
		gu.cs = p.fat
		s.parts = append(s.parts, p)
	}
	if len(s.parts) < 2 {
		eatRestore(s)
		var ms []*pevMem
		for _, p := range s.parts {
			ms = append(ms, p.m)
		}
		pevReleaseAll(ms, r)
		return false
	}
	// the table is as long as the contestants who came
	if k := len(s.parts); k < n {
		for i, p := range s.parts {
			p.m.pt = Vec{spot.X - float64(k-1)*eatSlotGap/2 + float64(i)*eatSlotGap, spot.Y - 5}
			g.pevSendTo(p.m, p.m.pt)
		}
		s.tableW = float64(k)*eatSlotGap + 14
	}
	pevDemoLog("comer", spot)
	a.D = s
	return true
}

// eatRestore: every contestant gets his own look back (dead or alive), if nothing else changed it meanwhile.
func eatRestore(s *eatState) {
	for _, p := range s.parts {
		if p.m.gu.cs == p.fat && p.orig != nil {
			p.m.gu.cs = p.orig
		}
	}
}

func (s *eatState) alarm(g *Game) bool {
	if pevDemo("comer") == "show" { // capture: the random killers of the party do not break it up
		return false
	}
	if g.policeCalled && !s.police {
		return true
	}
	for i := s.bodies; i < len(g.bodies); i++ {
		if dist(g.bodies[i].pos, s.spot) < 120 {
			return true
		}
	}
	return false
}

func (s *eatState) arrived(p *eatPart) bool {
	gu := p.m.gu
	return gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, p.m.pt) < 6
}

func eatUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*eatState)
	r := partyDice()
	age := g.time - s.t0
	if s.phase == ecOver {
		return true
	}
	// capture: a contestant is stabbed in the middle of the contest
	if pevDemo("comer") == "kill" && !s.demoK && s.phase == ecEat && age > 5 && len(s.parts) > 0 {
		s.demoK = true
		g.killGuest(s.parts[0].m.gu, true)
	}
	if s.phase == ecWalk {
		// on the way, the ones who are taken by something else simply drop out
		kept := s.parts[:0]
		for _, p := range s.parts {
			if p.m.mine() {
				kept = append(kept, p)
				continue
			}
			if p.m.gu.cs == p.fat {
				p.m.gu.cs = p.orig
			}
			pevForget(p.m)
		}
		s.parts = kept
		if len(s.parts) < 2 || s.alarm(g) {
			eatFlee(g, s, r)
			return true
		}
	} else if s.phase < ecEnd {
		for _, p := range s.parts {
			if !p.m.mine() {
				eatFlee(g, s, r)
				return true
			}
		}
		if s.alarm(g) {
			eatFlee(g, s, r)
			return true
		}
	} else { // the end: a dead one now only stops the show
		for _, p := range s.parts {
			if p.m.gu.out() && p.m.gu.cs == p.fat {
				p.m.gu.cs = p.orig
			}
		}
		if s.alarm(g) {
			eatFlee(g, s, r)
			return true
		}
	}
	if s.phase >= ecEat {
		s.crowd.update(g, r, nil, 0, nil)
		for _, m := range s.crowd.mem {
			if m.gu.icon == "" && r.Float64() < dt*0.12 {
				m.gu.icon, m.gu.iconT = pevPick(r, eatCheers), 1.5
			}
		}
	}
	switch s.phase {
	case ecWalk:
		all := true
		for _, p := range s.parts {
			if !p.arrived && s.arrived(p) {
				p.arrived = true
				p.m.want = gIdle
			}
			if p.arrived {
				p.m.keep()
				p.m.gu.dir = DirDown
			} else {
				all = false
			}
		}
		if !all && age > eatWalkMax { // the late ones are let go; with two at the table it goes on
			kept := s.parts[:0]
			for _, p := range s.parts {
				if p.arrived {
					kept = append(kept, p)
					continue
				}
				if p.m.gu.cs == p.fat {
					p.m.gu.cs = p.orig
				}
				pevLetGo(p.m)
			}
			s.parts = kept
			if len(s.parts) < 2 {
				eatEnd(g, s)
				return true
			}
			all = true
		}
		if all {
			s.phase, s.t0 = ecEat, g.time
			s.eatLen = eatMinLen + r.Float64()*(eatMaxLen-eatMinLen)
			for _, p := range s.parts {
				p.nextBite = g.time + 0.6 + r.Float64()*0.8
			}
			s.crowd.gather(g, r, Vec{s.spot.X, s.spot.Y + 16}, 4+r.Intn(3), 26, 40, func(gu *Guest) bool { return s.pevHas(gu) })
			if p := s.parts[r.Intn(len(s.parts))]; p.m.gu.icon == "" {
				p.m.gu.icon, p.m.gu.iconT = pevPick(r, eatGo), 2.2
			}
		}
	case ecEat:
		for _, p := range s.parts {
			p.m.keep()
			p.m.gu.dir = DirDown
			if g.time >= p.nextBite {
				p.biteT = g.time
				p.eaten++
				p.food = (p.food + r.Intn(3)/2) % 2 // mostly the same, sometimes the other
				p.nextBite = g.time + p.rate*(0.8+0.4*r.Float64())
				if p.m.gu.icon == "" && r.Float64() < 0.12 {
					p.m.gu.icon, p.m.gu.iconT = pevPick(r, eatBites), 1.2
				}
			}
		}
		if age >= s.eatLen {
			eatFinish(g, s, r)
		}
	case ecEnd:
		for _, p := range s.parts {
			if p.m.mine() {
				p.m.keep()
			}
		}
		if l := s.loser; l != nil && l.m.mine() {
			if !l.vomit && age > 1.8 {
				l.vomit = true
				l.m.pose(AnimVomit)
				l.m.gu.icon, l.m.gu.iconT = "BLEH", 2
			}
			if l.vomit && age > 4.2 && !l.choked {
				l.choked = true // the stain is on the floor
				g.decals = append(g.decals, Decal{pos: Vec{l.m.gu.pos.X + 2, l.m.gu.pos.Y + 3}, kind: 2, r: 8})
				l.m.pose(AnimTired)
			}
		}
		if g.time-s.cheerAt > 1.6 {
			s.cheerAt = g.time
			for _, m := range s.crowd.mem {
				if m.gu.icon == "" && r.Float64() < 0.35 {
					m.gu.icon, m.gu.iconT = pevPick(r, eatCheers), 1.4
				}
			}
		}
		if age > eatEndLen {
			eatEnd(g, s)
			return true
		}
	}
	return false
}

// eatFinish: the most eaten wins and celebrates, the least eaten chokes (and then throws up).
func eatFinish(g *Game, s *eatState, r *rand.Rand) {
	s.phase, s.t0, s.cheerAt = ecEnd, g.time, g.time
	order := r.Perm(len(s.parts)) // ties are settled by chance
	w, l := s.parts[order[0]], s.parts[order[0]]
	for _, i := range order {
		p := s.parts[i]
		if p.eaten > w.eaten {
			w = p
		}
		if p.eaten < l.eaten {
			l = p
		}
	}
	if l == w {
		for _, i := range order {
			if s.parts[i] != w {
				l = s.parts[i]
				break
			}
		}
	}
	s.winner, s.loser = w, l
	for _, p := range s.parts {
		switch p {
		case w:
			p.m.pose(AnimCheer)
			p.m.gu.icon, p.m.gu.iconT = pevPick(r, eatWin), 2.6
		case l:
			p.m.pose(AnimTired)
			p.m.gu.icon, p.m.gu.iconT = pevPick(r, eatChoke), 1.8
		default:
			p.m.hold(gIdle)
			if r.Float64() < 0.5 {
				p.m.gu.icon, p.m.gu.iconT = pevPick(r, eatRest), 1.8
			}
		}
	}
	for _, m := range s.crowd.mem {
		if m.gu.icon == "" && r.Float64() < 0.5 {
			m.gu.icon, m.gu.iconT = pevPick(r, eatCheers), 1.6
		}
	}
	g.snd.Play("win")
}

// eatFlee: somebody was killed or frightened: the contestants and the audience run off, the fat look goes back.
func eatFlee(g *Game, s *eatState, r *rand.Rand) {
	s.fled = true
	eatRestore(s)
	for _, p := range s.parts {
		if p.m.mine() {
			pevScare(g, p.m.gu, r)
		} else {
			pevForget(p.m)
		}
	}
	for _, m := range s.crowd.mem {
		if m.mine() {
			pevScare(g, m.gu, r)
		} else {
			pevForget(m)
		}
	}
	s.crowd.mem = nil
	s.phase = ecOver
}

func eatEnd(g *Game, s *eatState) {
	if s.phase == ecOver {
		return
	}
	r := partyDice()
	eatRestore(s)
	for _, p := range s.parts {
		pevLetGo(p.m)
	}
	s.crowd.release(g, r)
	s.phase = ecOver
}

// ---- drawing ----------------------------------------------------------------------------------------

var (
	eatClothA = color.RGBA{0xd8, 0x30, 0x30, 255}
	eatClothB = color.RGBA{0xf6, 0xf2, 0xea, 255}
	eatBoard  = color.RGBA{0x24, 0x40, 0x34, 255}
	eatWood   = color.RGBA{0x8a, 0x5a, 0x30, 255}
	eatWoodDk = color.RGBA{0x5a, 0x3a, 0x1e, 255}
)

// drawBurgerPix: a burger of 7x5 (px 1) with its top-left corner at (x, y).
func drawBurgerPix(plot plotFn, x, y, px int) {
	bun, bunD := color.RGBA{0xe0, 0x9a, 0x44, 255}, color.RGBA{0xb8, 0x74, 0x2c, 255}
	plot(x+px, y, 5*px, px, bun)
	plot(x, y+px, 7*px, px, bun)
	plot(x+2*px, y, px, px, pevWhite) // sesame
	plot(x+4*px, y+px, px, px, pevWhite)
	plot(x, y+2*px, 7*px, px, color.RGBA{0x58, 0xc0, 0x3a, 255}) // lettuce
	plot(x, y+3*px, 7*px, px, color.RGBA{0x6a, 0x36, 0x18, 255}) // the meat
	plot(x+px, y+4*px, 5*px, px, bunD)
}

// drawHotDogPix: a hot dog of 9x4 (px 1) with mustard, its top-left corner at (x, y).
func drawHotDogPix(plot plotFn, x, y, px int) {
	bun := color.RGBA{0xe8, 0xb0, 0x5a, 255}
	saus := color.RGBA{0xb8, 0x46, 0x28, 255}
	mus := color.RGBA{0xf4, 0xd4, 0x2a, 255}
	plot(x+px, y, 7*px, px, bun)
	plot(x, y+px, 9*px, 2*px, saus)
	for i := 1; i < 8; i += 2 {
		plot(x+i*px, y+px+(i/2%2)*px, px, px, mus)
	}
	plot(x+px, y+3*px, 7*px, px, shade(bun, 0.8))
}

func drawFoodPix(plot plotFn, kind, cx, y, px int) {
	if kind == 0 {
		drawBurgerPix(plot, cx-3*px, y, px)
		return
	}
	drawHotDogPix(plot, cx-4*px, y+px, px)
}

// eatTableRect: the screen rectangle of the table (top of the board to the floor).
func (g *Game) eatTableRect(s *eatState) (x0, top, w, floor int) {
	k := g.pevK()
	cx, by := g.pevXY(Vec{s.spot.X, s.spot.Y + 5})
	_, fy := g.pevXY(Vec{s.spot.X, s.spot.Y - 5})
	w = int(math.Round(s.tableW))
	return cx - w/2, fy - k(10), w, by
}

// eatSignRect: the chalkboard (screen) and its text lines.
func (g *Game) eatSignRect(s *eatState) (x, y, w, h int, lines []string) {
	lines = wrapBubble(tr(eatSign[0]), 64, func(t string) float64 { return g.textW(t, 1) }, 2)
	tw := 0.0
	for _, l := range lines {
		tw = math.Max(tw, math.Round(g.textW(l, 1)))
	}
	w = int(tw) + 14
	h = int(bubBoxBase+bubLineH*float64(len(lines)-1)) + 10
	tx0, _, tw0, floor := g.eatTableRect(s)
	if s.signL {
		x = tx0 - 8 - w
	} else {
		x = tx0 + tw0 + 8
	}
	y = floor - 8 - h
	return
}

func eatItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*eatState)
	if s.phase == ecOver {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	// the table, the plates, the scores and the chalkboard (in front of the contestants)
	items = append(items, drawItem{s.spot.Y + 5, func() {
		x0, top, w, floor := g.eatTableRect(s)
		bh := 4 * px
		plot(x0, top, w, bh, eatClothB)
		plot(x0, top, w, px, color.RGBA{0xff, 0xff, 0xff, 255})
		ch := floor - top - bh
		sq := 3 * px
		for yy := 0; yy < ch; yy += sq {
			for xx := 0; xx < w; xx += sq {
				col := eatClothB
				if (xx/sq+yy/sq)%2 == 0 {
					col = eatClothA
				}
				plot(x0+xx, top+bh+yy, minI(sq, w-xx), minI(sq, ch-yy), col)
			}
		}
		plot(x0, top+bh, w, px, shade(eatClothA, 0.7))
		plot(x0, floor-px, w, px, shade(eatClothA, 0.6))
		g.hudTouch(float64(x0), float64(top), float64(x0+w), float64(floor))
		for _, p := range s.parts {
			cx, _ := g.pevXY(p.m.pt)
			// the plate and what is left on it, low on the board (the belly must show above it)
			plot(cx-9*px, top+2*px, 19*px, px, pevWhite)
			left := minI(2, eatStack-p.eaten)
			for i := 0; i < left; i++ {
				drawFoodPix(plot, (p.food+i)%2, cx-5*px+i*10*px, top-3*px, px)
			}
			// the score on a dark tag on the cloth
			ty := top + bh + 3*px
			digits := []int{p.eaten % 10}
			if p.eaten >= 10 {
				digits = []int{p.eaten / 10 % 10, p.eaten % 10}
			}
			tagW := (len(digits)*4 + 4) * px
			plot(cx-tagW/2, ty, tagW, 9*px, color.RGBA{0x20, 0x1c, 0x28, 255})
			for i, d := range digits {
				drawDigitPix(plot, cx-tagW/2+2*px+i*4*px, ty+2*px, px, d, pevGold)
			}
		}
		// the chalkboard on its easel
		sx, sy, sw, sh, lines := g.eatSignRect(s)
		plot(sx+3, sy+sh-2, 2, floor-(sy+sh)+2, eatWoodDk)
		plot(sx+sw-5, sy+sh-2, 2, floor-(sy+sh)+2, eatWoodDk)
		plot(sx, sy, sw, sh, eatWood)
		plot(sx+2, sy+2, sw-4, sh-4, eatBoard)
		g.hudTouch(float64(sx), float64(sy), float64(sx+sw), float64(floor))
		for i, l := range lines {
			lw := math.Round(g.textW(l, 1))
			g.drawText(dst, l, float64(sx)+math.Round((float64(sw)-lw)/2), float64(sy+4)+float64(i)*bubLineH, rgb(0xfff4c8), 1)
		}
	}})
	// food at the mouth, full cheeks, crumbs, the winner's medal (over each contestant)
	for _, p := range s.parts {
		p := p
		gu := p.m.gu
		if gu.out() || gu.cs != p.fat {
			continue
		}
		items = append(items, drawItem{gu.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(gu.pos)
			hy := cy - k(28-p.stature) // the middle of the face
			since := g.time - p.biteT
			if s.phase == ecEat {
				switch {
				case since < eatBiteT:
					u := since / eatBiteT
					fy := hy + k(9-7*u)
					plot(cx-2*px, fy+2*px, 4*px, 2*px, p.skin) // the hand
					drawFoodPix(plot, p.food, cx, fy-2*px, px)
				case since < eatChewT:
					for _, sd := range []int{-1, 1} { // puffed cheeks
						x := cx + sd*4*px
						if sd < 0 {
							x -= 2 * px
						}
						plot(x, hy, 2*px, 3*px, p.skin)
						plot(x, hy+2*px, 2*px, px, shade(p.skin, 0.85))
						plot(x+px/2, hy+px, px, px, color.RGBA{0xf0, 0x80, 0x80, 255})
					}
					if int((since-eatBiteT)*8)%2 == 0 { // crumbs
						plot(cx-3*px, hy+4*px+int(since*10)%3*px, px, px, color.RGBA{0xe0, 0x9a, 0x44, 255})
						plot(cx+3*px, hy+5*px+int(since*13)%3*px, px, px, color.RGBA{0xe0, 0x9a, 0x44, 255})
					}
				}
			}
			if s.phase == ecEnd && p == s.winner { // a gold medal on the chest
				my := hy + k(9)
				plot(cx-px, hy+k(4), px, k(5), color.RGBA{0x2a, 0x6a, 0xd8, 255})
				plot(cx+px, hy+k(4), px, k(5), color.RGBA{0x2a, 0x6a, 0xd8, 255})
				plot(cx-2*px, my, 5*px, 4*px, pevGold)
				plot(cx-px, my+px, 2*px, px, pevWhite)
				drawStarsPix(plot, cx, hy-k(9), px, g.time)
			}
			if s.phase == ecEnd && p == s.loser && !p.vomit { // choking: green face
				plot(cx-2*px, hy, 5*px, px, color.RGBA{0x7a, 0xc0, 0x50, 255})
			}
		}})
	}
	// the audience claps
	for _, m := range s.crowd.mem {
		gu := m.gu
		if !m.mine() || gu.state != gIdle {
			continue
		}
		items = append(items, drawItem{gu.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(gu.pos)
			drawClapPix(plot, cx, cy-k(34), px, g.time, int(math.Abs(gu.pos.X)))
		}})
	}
	return items
}
