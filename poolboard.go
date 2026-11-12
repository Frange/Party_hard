package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// DIVING BOARD (user, request 229: "en donde haya piscinas sí").
//
// The prop: buildLevel (vip.go) calls Level.addDivingBoard() right after the cloth tables. Only levels with POOLS get one
// (CRUCERO NOCTURNO and ROOFTOP PARTY of the campaign, and the pools of the random rooftops); never the island (PropScale). The
// board stands on the deck at the left or the right edge of a pool with the plank over the water. Only the concrete
// pedestal is solid (its footprint, 12x7); the plank is drawn over the water and never blocks anything. A spot is
// accepted only if the pedestal is on dry free floor, the place where the jumper climbs is reachable, the tip and the
// landing point are in the water, nothing else stands on the plank or the landing, and every floor cell that could be
// reached before is still reachable (like the cloth tables: otherwise it is undone).
//
// The event "poolboard" (a "sometimes" party event, partyevents.go): now and then one or two guests walk to the
// board, climb, walk the plank, bounce and jump with a spin, and splash into the pool (drops and a ring of foam). Then
// they swim as any guest who went for a swim. They are plain guests all the time: they can be stabbed on the ladder,
// on the plank or in the air (the body falls where their feet are: the water). The guest's position never enters the
// pedestal (he waits beside it while he is drawn climbing), so whatever stops the jump leaves him on walkable ground
// (the deck or the water). All the dice are partyDice(); g.rng is never moved.
// ---------------------------------------------------------------------------

const (
	pbName   = "divingboard"
	pbHalf   = 32   // half width of the sprite (the pedestal is in its middle)
	pbLen    = 28.0 // from the pedestal centre to the tip of the plank
	pbStep   = 16.0 // from the pedestal centre to the first point of the plank the jumper walks on
	pbTop    = 17.0 // height of the plank's top over the ground line of the jumper (pedestal Y + 1)
	pbLandDX = 20.0 // from the tip to where he enters the water
	pbStand  = 16.0 // from the pedestal centre to the foot of the ladder
	pbQueue  = 31.0 // the second jumper waits here
)

// divingOff: tests switch the board off to look at a bare level.
var divingOff bool

func init() {
	waterOK[pbName] = true // the plank hangs over the water on purpose (overlap.go); the pedestal is checked by hand
	propKinds[pbName] = &PropKind{W: 2 * pbHalf, H: 26, FW: 12, FH: 7, Vars: 2, Category: "furniture", Draw: drawDivingBoard}
}

// drawDivingBoard: var 0 the plank points right (the board stands at the left edge of the pool), var 1 to the left.
func drawDivingBoard(b *pixBuf, v int) {
	W := 2 * pbHalf
	r := func(x, y, w, h int, col color.RGBA) {
		if v%2 == 1 {
			x = W - x - w
		}
		b.rect(x, y, w, h, col)
	}
	steel, steelD, steelL := c(0xb8c0cc), c(0x6c7482), c(0xeef2f8)
	conc, concD := c(0xd8d4c8), c(0x9a968c)
	// the plank: white top with a non-slip line, a blue edge, a rounded tip
	r(30, 9, 32, 2, c(0xf4f6f8))
	r(30, 11, 32, 2, c(0x2f6cc0))
	r(30, 13, 32, 1, c(0x1c4680))
	for x := 34; x < 60; x += 3 {
		r(x, 10, 1, 1, c(0xc8d2de))
	}
	r(62, 10, 1, 3, c(0x2f6cc0))
	r(62, 9, 1, 1, c(0xdfe6ee))
	// the roller under the plank, on the pedestal
	r(38, 14, 3, 2, steelD)
	// the pedestal (the only solid part)
	r(26, 12, 12, 4, conc)
	r(26, 12, 12, 1, c(0xf0ece2))
	r(26, 16, 12, 10, concD)
	r(26, 16, 12, 1, conc)
	r(37, 16, 1, 10, c(0x7a766e))
	// the ladder up the back of the pedestal, with its two curved handrails
	r(22, 12, 1, 14, steelD)
	r(25, 12, 1, 14, steelD)
	for y := 15; y < 26; y += 3 {
		r(22, y, 4, 1, steel)
	}
	for _, x := range []int{22, 25} {
		r(x, 3, 1, 9, steel)
		r(x+1, 2, 3, 1, steel)
		r(x+4, 3, 1, 4, steel)
		r(x, 3, 1, 1, steelL)
	}
}

// ---- placement (buildLevel) -------------------------------------------------------------------------------

// addDivingBoard puts one board at a pool of a built level. true if it placed one.
func (l *Level) addDivingBoard() bool {
	d := l.Def
	if d == nil || divingOff || vipBuilding || len(l.Pools) == 0 || d.Title == "" || d.PropScale > 0 && d.PropScale != 1 { // not on the island (its things are drawn smaller)
		return false
	}
	if l.divingBoard() != nil {
		return false
	}
	k := propKinds[pbName]
	fitAllFootprints()
	reach0 := l.reachFrom(d.Start)
	for _, pool := range l.Pools {
		if pool.W < pbLen+pbLandDX+30 || pool.H < 40 {
			continue
		}
		mid := math.Round(pool.Y + pool.H/2)
		for i := 0; i < 14; i++ {
			off := float64((i+1)/2) * 6
			if i%2 == 1 {
				off = -off
			}
			y := mid + off
			if y < pool.Y+14 || y > pool.Y+pool.H-6 {
				continue
			}
			for v := 0; v < 2; v++ {
				x := pool.X - 8
				if v == 1 {
					x = pool.X + pool.W + 8
				}
				if l.tryDivingBoard(Vec{x, y}, v, pool, k, reach0) {
					return true
				}
			}
		}
	}
	return false
}

// pbSide: +1 the plank points right, -1 left.
func pbSide(pr *Prop) float64 {
	if pr.Var%2 == 1 {
		return -1
	}
	return 1
}

// The places of a board (world points on the ground line of the jumper, pedestal Y + 1).
func pbStandAt(pr *Prop) Vec { return Vec{pr.X - pbSide(pr)*pbStand, pr.Y + 1} }
func pbQueueAt(pr *Prop) Vec { return Vec{pr.X - pbSide(pr)*pbQueue, pr.Y + 2} }
func pbBoardAt(pr *Prop, u float64) Vec {
	return Vec{pr.X + pbSide(pr)*(pbStep+(pbLen-pbStep)*u), pr.Y + 1}
}
func pbLandAt(pr *Prop) Vec { return Vec{pr.X + pbSide(pr)*(pbLen+pbLandDX), pr.Y + 3} }

func (l *Level) tryDivingBoard(p Vec, v int, pool Rect, k *PropKind, reach0 []bool) bool {
	probe := &Prop{X: p.X, Y: p.Y, Var: v}
	dry := func(q Vec) bool { return l.Free(q) && !l.SolidP(q.X, q.Y) && !l.badGround(q) && !l.InPool(q) }
	for _, o := range []Vec{{0, -1}, {-6, -1}, {6, -1}, {-6, -6}, {6, -6}, {0, -6}} {
		if !dry(Vec{p.X + o.X, p.Y + o.Y}) {
			return false
		}
	}
	stand, queue := pbStandAt(probe), pbQueueAt(probe)
	if !dry(stand) || !dry(queue) || !reach0[l.navIdx(stand)] || !reach0[l.navIdx(queue)] {
		return false
	}
	in := pool.Grow(-8)
	for u := 0.0; u <= 1; u += 0.25 {
		if q := pbBoardAt(probe, u); !l.InPool(q) || !l.Free(q) {
			return false
		}
	}
	land := pbLandAt(probe)
	if !in.Has(land.X, land.Y) || !l.Free(land) {
		return false
	}
	// nothing else on the plank (its drawing) or where he lands
	sd := pbSide(probe)
	plank := Rect{math.Min(p.X, p.X+sd*(pbLen+4)), p.Y - 20, pbLen + 4, 22}
	splash := Rect{land.X - 12, land.Y - 10, 24, 16}
	for _, o := range l.Props {
		bs := o.base()
		if pbOverlap(bs, plank) || pbOverlap(bs, splash) || pbOverlap(o.Foot, plank) {
			return false
		}
	}
	if z := l.vip; z != nil {
		for _, rc := range []Rect{z.in, z.trig, z.gate} {
			if rc.Grow(20).Has(p.X, p.Y) || rc.Grow(10).Has(stand.X, stand.Y) || rc.Grow(10).Has(queue.X, queue.Y) {
				return false
			}
		}
	}
	for _, dr := range l.Def.Dance {
		if dr.Grow(10).Has(p.X, p.Y) || dr.Has(stand.X, stand.Y) {
			return false
		}
	}
	if dist(p, l.Def.Start) < 50 {
		return false
	}
	if l.propClashes(pbName, k, p, 0) {
		return false
	}
	nProps, nP, nN := len(l.Props), len(l.solidsP), len(l.solidsN)
	undo := func() {
		l.Props, l.solidsP, l.solidsN = l.Props[:nProps], l.solidsP[:nP], l.solidsN[:nN]
		l.initNav()
	}
	l.addTableProp(pbName, k, p, v)
	l.initNav()
	reach1 := l.reachFrom(l.Def.Start)
	if !reach1[l.navIdx(stand)] || !reach1[l.navIdx(queue)] || !reach1[l.navIdx(land)] || !l.Free(stand) || !l.Free(queue) {
		undo()
		return false
	}
	for u := 0.0; u <= 1; u += 0.1 { // every point he walks on stays open water once the pedestal is solid
		if q := pbBoardAt(probe, u); !l.Free(q) || l.SolidP(q.X, q.Y) || !reach1[l.navIdx(q)] {
			undo()
			return false
		}
	}
	if len(reach0) == len(reach1) {
		for i := range reach0 {
			if reach0[i] && !reach1[i] && !l.blocked[i] {
				undo()
				return false
			}
		}
	}
	return true
}

// navIdx: the cell of the walking map of a point (clamped to the map).
func (l *Level) navIdx(p Vec) int {
	cx := clampI(int(p.X/navCell), 0, l.cw-1)
	cy := clampI(int(p.Y/navCell), 0, l.ch-1)
	return cy*l.cw + cx
}

func pbOverlap(a, b Rect) bool {
	return a.W > 0 && a.H > 0 && b.W > 0 && b.H > 0 && a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// divingBoard: the board of the level (nil = none).
func (l *Level) divingBoard() *Prop {
	if l == nil {
		return nil
	}
	for _, pr := range l.Props {
		if pr.Name == pbName && !pr.gone {
			return pr
		}
	}
	return nil
}

// ---- the event ------------------------------------------------------------------------------------------

var (
	pbCalls  = pevTexts("¡Mirad cómo salto!", "¡Allá voy!", "¡Bomba va!", "¡Al agua, patos!", "¡Que me grabe alguien!")
	pbCheers = pevTexts("¡Un diez!", "¡Uuuuh!", "¡Otra, otra!", "¡Menudo planchazo!", "¡Toma ya!", "¡Qué estilo!")
)

const (
	pbWalk   = iota // walking to the ladder (or to the queue)
	pbWait          // the second one waits his turn
	pbClimb         // up the ladder (drawn; he stays at the foot of it)
	pbPlank         // along the plank
	pbBounce        // two bounces at the tip
	pbAir           // the jump
	pbDone          // in the water (or let go)
)

const (
	pbClimbT  = 0.7
	pbPlankT  = 1.2
	pbBounceT = 0.9
	pbAirT    = 0.75
)

type pbJumper struct {
	mem   *pevMem
	phase int
	t     float64 // seconds in the phase
	t0    float64 // game time he was sent
	spin  bool    // a spin in the air (else arms up)
	// toQueue: on his way to the waiting place (the second jumper)
	toQueue bool
}

type pbRing struct {
	at Vec
	t0 float64
}

type poolBoardState struct {
	board *Prop
	js    []*pbJumper
	rings []pbRing
	// kept for the tests
	splashes int
}

func (s *poolBoardState) pevHas(gu *Guest) bool {
	for _, j := range s.js {
		if j.mem.gu == gu && j.phase != pbDone {
			return true
		}
	}
	return false
}

func init() {
	registerPartyEvent(&partyEvent{ID: "poolboard", Name: "Trampolín", Group: "pool", Prob: 0.30, AtMin: 40, AtMax: 260, Dur: 35,
		Cooldown: 150, MaxRuns: 3,
		Eligible: func(g *Game, t partyTraits) bool {
			if !t.pool {
				return false
			}
			// in the party being played it needs the board itself (a pool where none fits gets no jumps)
			if g != nil && g.lvl != nil && g.levelIdx == t.idx && g.lvl.Def != nil && len(g.lvl.Def.Pools) > 0 {
				return g.lvl.divingBoard() != nil
			}
			return true
		},
		Start: poolBoardStart, Update: pevGuard(poolBoardUpdate, poolBoardEnd), Items: poolBoardItems})
}

func poolBoardStart(g *Game, a *partyAct) bool {
	board := g.lvl.divingBoard()
	if board == nil {
		return false
	}
	r := partyDice()
	stand := pbStandAt(board)
	var cand []*Guest
	for _, gu := range g.guests {
		if !g.pevFreeGuest(gu) || g.lvl.InPool(gu.pos) || dist(gu.pos, stand) > 420 || gu.gait != "" {
			continue
		}
		cand = append(cand, gu)
	}
	if len(cand) == 0 {
		return false
	}
	// the nearest ones (a little shuffled)
	for i := range cand {
		j := i + r.Intn(minI(3, len(cand)-i))
		cand[i], cand[j] = cand[j], cand[i]
	}
	sortGuestsBy(cand, func(gu *Guest) float64 { return dist(gu.pos, stand) + r.Float64()*60 })
	want := 1 + r.Intn(2)
	s := &poolBoardState{board: board}
	for _, gu := range cand {
		if len(s.js) >= want {
			break
		}
		m := &pevMem{gu: gu}
		to := stand
		if len(s.js) == 1 {
			to = pbQueueAt(board)
		}
		if !g.pevSendTo(m, to) {
			continue
		}
		s.js = append(s.js, &pbJumper{mem: m, t0: g.time, spin: r.Float64() < 0.5, toQueue: to != stand})
	}
	if len(s.js) == 0 {
		return false
	}
	gu := s.js[0].mem.gu
	gu.icon, gu.iconT = pevPick(r, pbCalls), 2.5
	a.D = s
	return true
}

// pbOnBoard: a jumper in the phases that are drawn by this event.
func (j *pbJumper) onBoard() bool { return j.phase >= pbClimb && j.phase <= pbAir }

func poolBoardUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*poolBoardState)
	r := partyDice()
	busy := false // somebody is on the ladder, the plank or in the air
	for _, j := range s.js {
		if j.onBoard() {
			busy = true
		}
	}
	for i, j := range s.js {
		gu := j.mem.gu
		if j.phase == pbDone {
			continue
		}
		if gu.out() || !j.mem.mine() {
			pbLetGo(j) // a fright, a murder...: he is on the deck or in the water, both walkable
			continue
		}
		j.t += dt
		switch j.phase {
		case pbWait:
			j.mem.hold(gIdle)
			gu.dir = faceTo(gu.pos, Vec{s.board.X, s.board.Y})
			if !busy && !pbAnyBefore(s, i) {
				j.phase, j.t, j.t0, j.toQueue = pbWalk, 0, g.time, false
				if !g.pevSendTo(j.mem, pbStandAt(s.board)) {
					pbLetGo(j)
				}
			}
		case pbWalk:
			to := pbStandAt(s.board)
			if j.toQueue {
				to = pbQueueAt(s.board)
			}
			arrived := gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, to) < 9
			switch {
			case arrived && j.toQueue:
				j.phase = pbWait
				j.mem.hold(gIdle)
			case arrived && !busy && !pbAnyBefore(s, i):
				j.mem.hold(gIdle)
				j.phase, j.t = pbClimb, 0
				busy = true
			case arrived:
				j.mem.hold(gIdle)
			case g.time-j.t0 > 30: // he could not get there
				pbLetGo(j)
			}
		case pbClimb:
			j.mem.keep()
			gu.pos = pbStandAt(s.board)
			if j.t >= pbClimbT {
				j.phase, j.t = pbPlank, 0
				gu.pos = pbBoardAt(s.board, 0)
			}
		case pbPlank:
			j.mem.keep()
			u := math.Min(1, j.t/pbPlankT)
			gu.pos = pbBoardAt(s.board, u)
			gu.dir = DirRight
			if pbSide(s.board) < 0 {
				gu.dir = DirLeft
			}
			if j.t >= pbPlankT {
				j.phase, j.t = pbBounce, 0
			}
		case pbBounce:
			j.mem.keep()
			gu.pos = pbBoardAt(s.board, 1)
			if j.t >= pbBounceT {
				j.phase, j.t = pbAir, 0
			}
		case pbAir:
			j.mem.keep()
			u := math.Min(1, j.t/pbAirT)
			tip, land := pbBoardAt(s.board, 1), pbLandAt(s.board)
			gu.pos = Vec{tip.X + (land.X-tip.X)*u, tip.Y + (land.Y-tip.Y)*u}
			if j.t >= pbAirT {
				gu.pos = g.lvl.NearestFree(land)
				pbSplash(g, s, gu.pos, r)
				gu.state, gu.timer, gu.path, gu.moving = gSwim, 5+r.Float64()*6, nil, false
				gu.dur, gu.lookT = 0, 0
				j.phase = pbDone
				pbCheer(g, gu, r)
			}
		}
	}
	// splash rings live 0.7 s
	out := s.rings[:0]
	for _, rg := range s.rings {
		if g.time-rg.t0 < 0.7 {
			out = append(out, rg)
		}
	}
	s.rings = out
	for _, j := range s.js {
		if j.phase != pbDone {
			return false
		}
	}
	return len(s.rings) == 0
}

// pbAnyBefore: a jumper before i has not jumped yet.
func pbAnyBefore(s *poolBoardState, i int) bool {
	for k := 0; k < i; k++ {
		if s.js[k].phase != pbDone {
			return true
		}
	}
	return false
}

// pbLetGo: the event lets go of him wherever he is (never inside the pedestal: see the header).
func pbLetGo(j *pbJumper) {
	gu := j.mem.gu
	if !gu.out() && j.mem.mine() {
		gu.state, gu.timer, gu.path, gu.moving = gIdle, 0.6, nil, false
		gu.dur, gu.lookT = 0, 0
	} else {
		pevForget(j.mem)
	}
	j.phase = pbDone
}

func poolBoardEnd(g *Game, a *partyAct) {
	if s, ok := a.D.(*poolBoardState); ok {
		for _, j := range s.js {
			if j.phase != pbDone {
				pbLetGo(j)
			}
		}
	}
}

// pbSplash: drops and foam where he enters the water (partyDice: g.rng is never moved).
func pbSplash(g *Game, s *poolBoardState, at Vec, r *rand.Rand) {
	s.splashes++
	s.rings = append(s.rings, pbRing{at: at, t0: g.time})
	g.snd.Play("splash")
	for i := 0; i < 26; i++ {
		a := -math.Pi/2 + (r.Float64()-0.5)*2.2
		sp := 40 + r.Float64()*70
		l := 0.45 + r.Float64()*0.35
		col := color.RGBA{0xa8, 0xdc, 0xf8, 255}
		if i%3 == 0 {
			col = color.RGBA{0xf0, 0xfa, 0xff, 255}
		}
		g.parts = append(g.parts, Particle{pos: Vec{at.X + (r.Float64()-0.5)*8, at.Y - 2}, vel: Vec{math.Cos(a) * sp * 0.6, math.Sin(a) * sp},
			life: l, max: l, col: col, size: 2, grav: 230})
	}
}

// pbCheer: one or two of the people near the pool cheer (or laugh).
func pbCheer(g *Game, gu *Guest, r *rand.Rand) {
	n := 0
	for _, o := range g.guests {
		if n >= 2 || o == gu || o.out() || o.icon != "" || !g.canWatch(o) || dist(o.pos, gu.pos) > 150 || g.pevTaken(o) {
			continue
		}
		if r.Float64() < 0.5 {
			o.icon, o.iconT = pevPick(r, pbCheers), 2
			n++
		}
	}
}

// drawGuest: the jumper on the ladder, the plank and in the air is drawn here (pe_fx.go asks every running event).
func (s *poolBoardState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	var j *pbJumper
	for _, x := range s.js {
		if x.mem.gu == gu && x.onBoard() {
			j = x
		}
	}
	if j == nil {
		return false
	}
	sc := g.lifeScale()
	b := s.board
	side := pbSide(b)
	dir := DirRight
	if side < 0 {
		dir = DirLeft
	}
	anim, phase := AnimIdle, g.time*1.3
	at, lift := gu.pos, pbTop
	switch j.phase {
	case pbClimb:
		u := math.Min(1, j.t/pbClimbT)
		st, top := pbStandAt(b), pbBoardAt(b, 0)
		at = Vec{st.X + (top.X-st.X)*u, st.Y}
		lift = pbTop * u
		anim, phase, dir = AnimWalk, j.t*8, DirUp
	case pbPlank:
		anim, phase = AnimWalk, j.t*7
	case pbBounce:
		lift = pbTop + math.Abs(math.Sin(j.t/pbBounceT*2*math.Pi))*5
		anim, phase = AnimCrouch, j.t*3
		if math.Sin(j.t/pbBounceT*2*math.Pi) > 0.3 {
			anim = AnimCheer
		}
	case pbAir:
		u := math.Min(1, j.t/pbAirT)
		lift = pbTop*(1-u) + 34*4*u*(1-u)
		anim, phase = AnimCheer, j.t*6
		if j.spin {
			anim, phase = AnimPirouette, u*6
		}
	}
	lift = math.Round(lift)
	g.drawActor(dst, gu.cs, anim, phase, Vec{math.Round(at.X), at.Y}, dir, -lift/sc, false)
	return true
}

// poolBoardItems: the rings of foam on the water after each splash.
func poolBoardItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s, ok := a.D.(*poolBoardState)
	if !ok {
		return items
	}
	for _, rg := range s.rings {
		rg := rg
		items = append(items, drawItem{rg.at.Y - 3, func() { drawFoamRing(dst, g, rg.at, g.time-rg.t0) }})
	}
	return items
}

// drawFoamRing: a ring of foam that widens and thins out (whole pixels).
func drawFoamRing(dst *ebiten.Image, g *Game, at Vec, age float64) {
	u := math.Min(1, age/0.7)
	rx, ry := 5+u*14, 2+u*5
	al := uint8(230 * (1 - u))
	col := color.RGBA{al, al, al, al} // premultiplied white
	cx, cy := g.wx(at.X), g.wy(at.Y)
	for i := 0; i < 28; i++ {
		an := float64(i) / 28 * 2 * math.Pi
		fillRect(dst, math.Round(cx+math.Cos(an)*rx), math.Round(cy+math.Sin(an)*ry), 2, 1, col)
	}
}

// sortGuestsBy: a small stable insertion sort (the lists are short).
func sortGuestsBy(gs []*Guest, key func(*Guest) float64) {
	ks := make([]float64, len(gs))
	for i, gu := range gs {
		ks[i] = key(gu)
	}
	for i := 1; i < len(gs); i++ {
		for j := i; j > 0 && ks[j] < ks[j-1]; j-- {
			ks[j], ks[j-1] = ks[j-1], ks[j]
			gs[j], gs[j-1] = gs[j-1], gs[j]
		}
	}
}
