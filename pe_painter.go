package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "pintor" (idea 273, PINTOR EN ESCALERA): in a few parties with a facade or a high wall (painterIcons), sometimes a
// guest in white overalls walks to a ladder leaning on the wall, climbs it and paints the wall with a brush and a pot (a fresh
// stripe of colour grows on the wall).
//
// The rule of the user: "si ejecutas la acción se cae la escalera y muere, es el único modo de matarle, porque está arriba y
// no puedes ir con el cuchillo por la escalera":
//   - up there the knife does not reach him (stab skips him: pevAloft, one line in game.go);
//   - the ladder is a prop you can use: "[E] Tirar la escalera" (useLabel / pevUse, same 30 px reach as its label); it falls
//     (animation), he falls with it and dies: a body with paint, the pot rolling away, a stain of paint on the floor. It is a
//     kill of the player who pushed (killGuest byPlayer: witnesses, stars and streak as usual). Once per ladder.
//   - while he is not up there (walking to it) or once he has fallen, there is no prompt.
// The ladder is solid only on its real feet (14x4 px), against the wall, and it is only placed where no way of the party is
// cut (pevAddProp). The painter is an ordinary guest for the count (aliveCount, caps, targets) the whole time.
// ---------------------------------------------------------------------------

const (
	pntrLift     = 15.0 // how high his feet are when he paints
	pntrClimb    = 1.4  // seconds to climb
	pntrPaintLen = 70.0 // seconds he paints (then he climbs down and goes back to the party)
	pntrFallLen  = 0.55 // seconds the ladder takes to fall
	pntrReach    = 30.0 // same reach as the label of a prop (items.go useTextProp)
	pntrLadderL  = 36.0 // length of the ladder
)

// painterIcons: the parties with a facade or a high outdoor wall where the ladder fits (checked by TestPmEventsFitSomeLevelsAndStart).
var painterIcons = map[string]bool{"house": true, "cake": true} // (v1.4.0: the ranch became BARCELONA, an open square)

var (
	pntrLines     = pevTexts("Un poco más a la izquierda...", "Esta pared quedará preciosa", "¡Cuidado con el bote!", "Pintura fresca, ¿eh?", "No me mováis la escalera")
	pntrFallLines = pevTexts("¡Uaaah!", "¡La escaleraaa!")
	pntrDownLines = pevTexts("Mañana la segunda mano", "Listo. Ni una gota fuera")
	pntrSeeLines  = pevTexts("¡El pintor se ha caído!", "¡Qué golpe!")
	pntrLabel     = "Tirar la escalera"
	pntrTooHigh   = "Está muy alto: tira la escalera"
)

const (
	pnWalk = iota
	pnClimb
	pnPaint
	pnFall
	pnDown // climbing down at the end
	pnGone
)

type painterState struct {
	m       *pevMem
	ladder  *Prop
	wall    Rect
	base    Vec // the feet of the ladder (prop anchor)
	phase   int
	t0      float64
	lift    float64
	col     color.RGBA
	colIdx  int
	painted float64 // 0..1 of the stripe
	fallAt  float64
	land    Vec
	by      *Player
	next    float64
	toldHi  bool
	dead    bool
	pot     Vec // where the pot ends after rolling
}

func (s *painterState) pevHas(gu *Guest) bool { return s.m != nil && s.m.gu == gu }

// aloft: he is up the ladder (the knife does not reach him).
func (s *painterState) aloft(gu *Guest) bool {
	return s.m != nil && s.m.gu == gu && (s.phase == pnClimb || s.phase == pnPaint || s.phase == pnFall || s.phase == pnDown)
}

var pntrPaints = []color.RGBA{{0x7a, 0xc8, 0xf0, 255}, {0xf4, 0xd8, 0x50, 255}, {0x8a, 0xe0, 0xa8, 255}, {0xf0, 0x9a, 0x7a, 255}}

// decalMoreCols: the colours of the new floor stains (draw.go): 5 the cream of the cake (trapcake.go), 6.. the paints
// (premultiplied, see LECCIONES).
var decalMoreCols = map[int]color.RGBA{5: rgba(0xfff4ec, 225), 6: rgba(0x7ac8f0, 225), 7: rgba(0xf4d850, 225), 8: rgba(0x8ae0a8, 225), 9: rgba(0xf09a7a, 225)}

func init() {
	propKinds["paintladder"] = &PropKind{W: 18, H: 38, FW: 14, FH: 4, Category: "furniture", Draw: drawPaintLadder}
	propKinds["ladderdown"] = &PropKind{W: 50, H: 12, FW: 40, FH: 6, NoSolid: true, Category: "furniture", Draw: drawLadderDown}
	registerPartyEvent(&partyEvent{ID: "pintor", Name: "Pintor en escalera", Group: "wall",
		Prob: 0.20, AtMin: 10, AtMax: 150, Dur: 90,
		Eligible: func(g *Game, t partyTraits) bool { return t.roam && painterIcons[t.icon] },
		Start:    painterStart, Items: painterItems,
		Update: pevGuard(painterUpdate, func(g *Game, a *partyAct) { painterEnd(g, a.D.(*painterState)) })})
}

func drawPaintLadder(b *pixBuf, v int) {
	wood, woodD := c(0xc89858), c(0x7a5a30)
	b.rect(2, 2, 2, 36, woodD)
	b.rect(14, 2, 2, 36, woodD)
	b.rect(3, 2, 1, 36, wood)
	b.rect(15, 2, 1, 36, wood)
	for y := 6; y < 36; y += 6 {
		b.rect(4, y, 10, 2, wood)
		b.rect(4, y+1, 10, 1, woodD)
	}
	b.rect(1, 36, 4, 2, c(0x3a2a18)) // the feet
	b.rect(13, 36, 4, 2, c(0x3a2a18))
}

// drawLadderDown: the ladder lying on the floor after the fall.
func drawLadderDown(b *pixBuf, v int) {
	wood, woodD := c(0xc89858), c(0x7a5a30)
	b.rect(2, 2, 46, 2, woodD)
	b.rect(2, 9, 46, 2, woodD)
	b.rect(2, 3, 46, 1, wood)
	b.rect(2, 10, 46, 1, wood)
	for x := 6; x < 46; x += 6 {
		b.rect(x, 4, 2, 5, wood)
	}
	b.rect(2, 11, 46, 1, c(0x3a2a18))
}

// painterSpot: a place at the foot of a high wall where the ladder fits (nothing under it, open floor in front, far from
// doors, the start and the players). Outdoor walls (a facade) first.
func (g *Game) painterSpot(r *rand.Rand) (Vec, Rect, bool) {
	l := g.lvl
	k := propKinds["paintladder"]
	fitAllFootprints()
	type cand struct {
		base Vec
		wall Rect
	}
	var outs, ins []cand
	for _, w := range l.Walls {
		rr := w.R
		if rr.W < rr.H*2 || rr.W < 48 || rr.H < 12 {
			continue
		}
		switch w.Style {
		case WallBrick, WallPlaster, WallWood, WallClub:
		default:
			continue
		}
		for x := rr.X + 18; x <= rr.X+rr.W-18; x += 12 {
			base := Vec{math.Round(x), math.Round(rr.Y + rr.H + 6)}
			if !g.painterSpotOK(base, k) {
				continue
			}
			if l.Outdoor(Vec{base.X, base.Y + 8}) {
				outs = append(outs, cand{base, rr})
			} else {
				ins = append(ins, cand{base, rr})
			}
		}
	}
	pick := outs
	if len(pick) == 0 {
		pick = ins
	}
	if len(pick) == 0 {
		return Vec{}, Rect{}, false
	}
	c := pick[r.Intn(len(pick))]
	return c.base, c.wall, true
}

func (g *Game) painterSpotOK(base Vec, k *PropKind) bool {
	l := g.lvl
	if base.X < 30 || base.X > l.W-30 || base.Y > l.H-30 || dist(base, l.Def.Start) < 70 || !g.pevPlayerFar(base, 80) {
		return false
	}
	for _, o := range []Vec{{0, 1}, {-7, 1}, {7, 1}, {0, 10}, {-16, 6}, {16, 6}, {0, 18}} {
		q := Vec{base.X + o.X, base.Y + o.Y}
		if !l.Free(q) || l.badGround(q) || l.InPool(q) {
			return false
		}
	}
	for _, d := range l.Def.Doors {
		if d.R.Grow(26).Has(base.X, base.Y) {
			return false
		}
	}
	for _, dr := range l.Def.Dance {
		if dr.Grow(16).Has(base.X, base.Y) {
			return false
		}
	}
	if z := l.vip; z != nil {
		for _, rc := range []Rect{z.in, z.trig, z.gate} {
			if rc.Grow(24).Has(base.X, base.Y) {
				return false
			}
		}
	}
	return !l.propClashes("paintladder", k, base, 0)
}

func painterStart(g *Game, a *partyAct) bool {
	r := partyDice()
	demo := pevDemo("pintor")
	var base Vec
	var wall Rect
	ok := false
	if demo != "" { // capture: the wall spot nearest to the player
		base, wall, ok = g.painterSpotNear(g.p.pos)
	} else {
		base, wall, ok = g.painterSpot(r)
	}
	if !ok {
		return false
	}
	reach0 := g.lvl.reachFrom(g.lvl.Def.Start)
	stand := Vec{base.X, base.Y + 10}
	ladder := g.pevAddProp("paintladder", propKinds["paintladder"], base, 0, reach0, func(l *Level) bool { return l.Free(stand) })
	if ladder == nil {
		return false
	}
	// the painter: the nearest free guest (not a star) who can walk there
	var best *Guest
	for _, gu := range g.guests {
		if !g.pevFreeGuest(gu) || gu.name != "" || dist(gu.pos, stand) > 420 {
			continue
		}
		if best == nil || dist(gu.pos, stand) < dist(best.pos, stand) {
			best = gu
		}
	}
	if best == nil {
		g.pevRemoveProp(ladder)
		return false
	}
	m := &pevMem{gu: best, pt: stand}
	if demo != "" { // the capture does not wait for the walk
		best.pos = stand
	}
	if !g.pevSendTo(m, stand) {
		g.pevRemoveProp(ladder)
		return false
	}
	ci := r.Intn(len(pntrPaints))
	st := &painterState{m: m, ladder: ladder, wall: wall, base: base, phase: pnWalk, t0: g.time, col: pntrPaints[ci], colIdx: ci}
	a.D = st
	pevDemoLog("pintor", base)
	return true
}

// painterSpotNear (captures): the spot nearest to p.
func (g *Game) painterSpotNear(p Vec) (Vec, Rect, bool) {
	k := propKinds["paintladder"]
	fitAllFootprints()
	bd := 1e9
	var best Vec
	var wall Rect
	for _, w := range g.lvl.Walls {
		rr := w.R
		if rr.W < rr.H*2 || rr.W < 48 || rr.H < 12 || !(w.Style == WallBrick || w.Style == WallPlaster || w.Style == WallWood || w.Style == WallClub) {
			continue
		}
		for x := rr.X + 18; x <= rr.X+rr.W-18; x += 12 {
			base := Vec{math.Round(x), math.Round(rr.Y + rr.H + 6)}
			l := g.lvl
			okk := l.Free(Vec{base.X, base.Y + 10}) && l.Free(base) && !l.propClashes("paintladder", k, base, 0) && dist(base, l.Def.Start) > 40
			if okk && dist(base, p) < bd {
				bd, best, wall = dist(base, p), base, rr
			}
		}
	}
	return best, wall, bd < 1e9
}

func painterUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*painterState)
	r := partyDice()
	gu := st.m.gu
	age := g.time - st.t0
	switch st.phase {
	case pnWalk:
		if !st.m.mine() {
			painterEnd(g, st)
			return true
		}
		if gu.state == gIdle && gu.timer > 1000 {
			st.phase, st.t0 = pnClimb, g.time
			st.m.hold(gIdle)
		} else if age > 30 {
			painterEnd(g, st)
			return true
		}
	case pnClimb, pnPaint, pnDown:
		if gu.dead { // something else killed him up there (a blast, a killer): the ladder stays
			st.phase = pnGone
			return false
		}
		// up a ladder he cannot run away: whatever frightened him, he stays (and shouts)
		if gu.state != gIdle || gu.timer < 1000 || gu.path != nil {
			if gu.icon == "" {
				gu.icon, gu.iconT = "¡AAAH!", 1.4
			}
		}
		pevForgetHold(gu)
		gu.pos, gu.dir = Vec{st.base.X, st.base.Y + 1}, DirUp
		if st.phase == pnClimb {
			st.lift = pntrLift * math.Min(1, age/pntrClimb)
			if age >= pntrClimb {
				st.phase, st.t0, st.next = pnPaint, g.time, g.time+2
			}
		} else if st.phase == pnPaint {
			st.lift = pntrLift
			st.painted = math.Min(1, age/40)
			if pevDemo("pintor") != "" { // captures: the stripe shows at once
				st.painted = math.Min(1, 0.4+age/8)
			}
			if pevDemo("pintor") == "prompt" && age > 1 && age-1 < 0.05 { // capture: the player next to the ladder (its prompt)
				g.p.pos = g.lvl.NearestFree(Vec{st.base.X + 18, st.base.Y + 8})
			}
			if g.time >= st.next {
				st.next = g.time + 6 + r.Float64()*6
				if gu.icon == "" && g.pevPlayerFar(st.base, 70) {
					gu.icon, gu.iconT = pevPick(r, pntrLines), 2.2
				}
			}
			g.painterTooHigh(st)
			if !g.pevPlayerFar(st.base, 70) && gu.icon != "" { // his lines would cover the prompt of the ladder
				gu.icon, gu.iconT = "", 0
			}
			if age > pntrPaintLen && pevDemo("pintor") == "" {
				st.phase, st.t0 = pnDown, g.time
				gu.icon, gu.iconT = pevPick(r, pntrDownLines), 2
			}
			if pevDemo("pintor") == "caida" && age > 2.5 && st.by == nil { // capture: the player knocks it over
				g.p.pos = g.lvl.NearestFree(Vec{st.base.X + 18, st.base.Y + 8})
				g.painterKnock(st, g.p)
			}
		} else { // climbing down
			st.lift = pntrLift * math.Max(0, 1-age/pntrClimb)
			if age >= pntrClimb {
				gu.pos = Vec{st.base.X, st.base.Y + 10}
				st.phase = pnGone
				pevRelease(st.m, r)
				g.pevRemoveProp(st.ladder)
				return true
			}
		}
	case pnFall:
		u := (g.time - st.fallAt) / pntrFallLen
		st.lift = pntrLift * math.Max(0, 1-u*u)
		if u >= 1 && !st.dead {
			st.dead = true
			g.painterLands(st)
		}
		if g.time-st.fallAt > 25 {
			return true
		}
	case pnGone:
		return true
	}
	return false
}

// pevForgetHold keeps a painter up there: idle, held, no path (whatever another system wanted from him).
func pevForgetHold(gu *Guest) {
	gu.state, gu.timer, gu.path, gu.moving, gu.lookT = gIdle, 9999, nil, false, 5
	gu.dur, gu.hurry = 0, false
}

// painterTooHigh: a player who tries the knife under him is told how it works (once per party).
func (g *Game) painterTooHigh(st *painterState) {
	if st.toldHi {
		return
	}
	for _, p := range g.players() {
		if p != nil && !p.dead && p.stabT > 0 && dist(p.pos, st.m.gu.pos) < 30 {
			st.toldHi = true
			g.say(pntrTooHigh)
			return
		}
	}
}

// painterCanKnock: the ladder can be pushed now (he is up there, alive) and p is close enough.
func (g *Game) painterCanKnock(st *painterState, p *Player) bool {
	if st.ladder == nil || st.ladder.gone || st.by != nil || st.m.gu.out() || p == nil || p.dead || p.carrying != nil {
		return false
	}
	if st.phase != pnClimb && st.phase != pnPaint && st.phase != pnDown {
		return false
	}
	return distRect(p.pos, st.ladder.Foot) < pntrReach
}

// pevUse: E next to the ladder knocks it over.
func (st *painterState) pevUse(g *Game, p *Player) bool {
	if !g.painterCanKnock(st, p) {
		return false
	}
	g.painterKnock(st, p)
	return true
}

func (g *Game) painterKnock(st *painterState, p *Player) {
	r := partyDice()
	st.by, st.phase, st.fallAt = p, pnFall, g.time
	st.ladder.gone = true
	side := 1.0
	if p.pos.X > st.base.X { // it falls away from the one who pushes
		side = -1
	}
	st.land = g.lvl.NearestFree(Vec{st.base.X + side*30, st.base.Y + 12})
	st.pot = g.lvl.NearestFree(Vec{st.base.X + side*48, st.base.Y + 20})
	gu := st.m.gu
	gu.icon, gu.iconT = pevPick(r, pntrFallLines), 1.6
	g.snd.Play("squeak")
}

// painterLands: he hits the floor: the player's kill.
func (g *Game) painterLands(st *painterState) {
	gu := st.m.gu
	l := g.lvl
	// the ladder lies on the floor now (not solid any more)
	g.pevRemoveProp(st.ladder)
	k := propKinds["ladderdown"]
	ld := Vec{(st.base.X + st.land.X) / 2, st.base.Y + 8}
	l.Props = append(l.Props, &Prop{Name: "ladderdown", K: k, X: ld.X, Y: ld.Y, Foot: Rect{ld.X - k.FW/2, ld.Y - k.FH, k.FW, k.FH}, Sort: ld.Y})
	g.snd.Play("thud")
	g.shake = math.Max(g.shake, 0.25)
	g.decals = append(g.decals, Decal{pos: Vec{st.land.X, st.land.Y + 2}, kind: 6 + st.colIdx, r: 16})
	if gu.out() {
		return
	}
	gu.pos = st.land
	p := st.by
	if p == nil || p.dead {
		p = g.p
	}
	g.withPlayer(p, func() {
		g.killCause = kcTrap
		g.killGuest(gu, true)
	})
	for _, o := range g.guests {
		if !o.out() && o != gu && dist(o.pos, st.land) < 120 && o.icon == "" && g.canWatch(o) {
			o.icon, o.iconT = pevPick(partyDice(), pntrSeeLines), 1.8
			break
		}
	}
}

func painterEnd(g *Game, st *painterState) {
	if st.phase == pnFall && !st.dead {
		st.dead = true
		g.painterLands(st)
	}
	if st.phase == pnWalk || st.phase == pnClimb || st.phase == pnPaint || st.phase == pnDown {
		gu := st.m.gu
		if !gu.out() && st.phase != pnWalk {
			gu.pos = Vec{st.base.X, st.base.Y + 10}
			gu.state, gu.timer, gu.path = gIdle, 1, nil
		} else {
			pevLetGo(st.m)
		}
		if st.ladder != nil && !st.ladder.gone {
			g.pevRemoveProp(st.ladder)
		}
	}
	st.phase = pnGone
}

// painterLabel: the label of the ladder (items.go useLabel): only while the painter is up and alive.
func (g *Game) painterLabel(pr *Prop) (string, color.RGBA) {
	for _, a := range g.pev.act {
		if st, ok := a.D.(*painterState); ok && st.ladder == pr && st.by == nil && !st.m.gu.out() &&
			(st.phase == pnClimb || st.phase == pnPaint || st.phase == pnDown) {
			return pntrLabel, rgb(0xff8060)
		}
	}
	return "", color.RGBA{}
}

// pevAloft: a guest an event keeps up in the air (the painter on his ladder): the knife does not reach him (game.go stab).
func (g *Game) pevAloft(gu *Guest) bool {
	if !g.partyActive() {
		return false
	}
	for _, a := range g.pev.act {
		if s, ok := a.D.(interface{ aloft(*Guest) bool }); ok && s.aloft(gu) {
			return true
		}
	}
	return false
}

// drawGuest: the painter up the ladder (or falling) is drawn by the event: lifted, in white overalls.
func (st *painterState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if st.m == nil || gu != st.m.gu || st.phase == pnWalk || st.phase == pnGone && !gu.out() {
		return false
	}
	if gu.out() {
		return false
	}
	return true // drawn in painterItems (over the ladder)
}

func painterItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*painterState)
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	gu := st.m.gu
	// the fresh paint on the wall (just in front of the wall, behind everything standing below it)
	if st.painted > 0 {
		w := st.wall
		items = append(items, drawItem{w.Y + w.H - 0.5, func() {
			half := 2 + st.painted*14
			x0, _ := g.pevXY(Vec{st.base.X - half, 0})
			x1, _ := g.pevXY(Vec{st.base.X + half, 0})
			_, y0 := g.pevXY(Vec{0, math.Max(w.Y+3, st.base.Y-6-pntrLift-34)})
			_, y1 := g.pevXY(Vec{0, w.Y + w.H - 2})
			if y1 > y0 {
				plot(x0, y0, x1-x0, y1-y0, st.col)
				plot(x0, y1-px, x1-x0, px, shade(st.col, 0.75))
			}
		}})
	}
	if st.phase == pnFall && !st.dead {
		// the ladder falling: two rails and rungs turning round its feet
		u := math.Min(1, (g.time-st.fallAt)/pntrFallLen)
		ang := u * u * math.Pi / 2
		side := 1.0
		if st.land.X < st.base.X {
			side = -1
		}
		items = append(items, drawItem{st.base.Y + 0.4, func() {
			bx, by := g.pevXY(st.base)
			dx, dy := math.Sin(ang)*side, -math.Cos(ang) // along the ladder
			nx, ny := -dy, dx                            // across it (the two rails)
			L := float64(k(pntrLadderL))
			at := func(along, across float64) (int, int) {
				return bx + int(math.Round(dx*along+nx*across)), by + int(math.Round(dy*along+ny*across))
			}
			for _, off := range []float64{-6, 6} {
				x0, y0 := at(0, off)
				x1, y1 := at(L, off)
				drawLinePix(plot, x0, y0, x1, y1, 2*px, color.RGBA{0x7a, 0x5a, 0x30, 255})
			}
			for i := 1; i < 7; i++ {
				f := L * float64(i) / 7
				x0, y0 := at(f, -5)
				x1, y1 := at(f, 5)
				drawLinePix(plot, x0, y0, x1, y1, px, color.RGBA{0xc8, 0x98, 0x58, 255})
			}
		}})
	}
	// the pot of paint: on the top of the ladder while he paints, rolling away when it falls
	switch {
	case st.phase == pnPaint || st.phase == pnClimb:
		items = append(items, drawItem{st.base.Y + 0.6, func() {
			x, y := g.pevXY(st.base)
			drawPaintPotPix(plot, x+k(7), y-k(pntrLadderL)+k(4), px, st.col)
		}})
	case st.phase == pnFall || st.phase == pnGone && st.by != nil:
		u := math.Min(1, (g.time-st.fallAt)/1.1)
		p := Vec{st.base.X + (st.pot.X-st.base.X)*u, st.base.Y + (st.pot.Y-st.base.Y)*u}
		h := 40 * (1 - u) * (1 - u)
		items = append(items, drawItem{p.Y, func() {
			x, y := g.pevXY(p)
			drawPaintPotPix(plot, x, y-k(h), px, st.col)
		}})
	}
	// the painter himself, up the ladder or falling
	if !gu.out() && st.phase != pnWalk && st.phase != pnGone {
		items = append(items, drawItem{st.base.Y + 1, func() {
			pos := Vec{st.base.X, st.base.Y + 1}
			anim, ph := AnimWave, g.time*3
			dir := DirUp
			switch st.phase {
			case pnClimb, pnDown:
				anim, ph = AnimWalk, g.time*5
			case pnFall:
				u := math.Min(1, (g.time-st.fallAt)/pntrFallLen)
				pos = Vec{st.base.X + (st.land.X-st.base.X)*u, st.base.Y + 1 + (st.land.Y-st.base.Y-1)*u}
				anim, ph, dir = AnimPanic, g.time*8, DirDown
			}
			g.drawActor(dst, gu.cs, anim, ph, pos, dir, -math.Round(st.lift), false)
			cx, cy := g.pevXY(pos)
			cy -= int(math.Round(st.lift))
			drawOverallsPix(plot, cx, cy, px, k)
			if st.phase == pnPaint { // the brush going up and down on the wall
				bob := int(math.Round(math.Sin(g.time*5) * 3))
				plot(cx+k(4), cy-k(36)+bob*px, 2*px, 4*px, color.RGBA{0x8a, 0x5a, 0x30, 255})
				plot(cx+k(4), cy-k(38)+bob*px, 2*px, 2*px, st.col)
			}
		}})
	}
	return items
}

// drawOverallsPix: white overalls with straps and a white cap over a figure whose feet are at (cx, cy) (seen from behind
// or from the front alike).
func drawOverallsPix(plot plotFn, cx, cy, px int, k func(float64) int) {
	white, shadow := color.RGBA{0xf2, 0xf2, 0xee, 255}, color.RGBA{0xc8, 0xc8, 0xc4, 255}
	plot(cx-3*px, cy-k(24), 2*px, k(8), white)
	plot(cx+2*px, cy-k(24), 2*px, k(8), white)
	plot(cx-3*px, cy-k(16), 7*px, k(7), white)
	plot(cx-3*px, cy-k(10), 2*px, k(6), shadow)
	plot(cx+2*px, cy-k(10), 2*px, k(6), shadow)
	plot(cx-3*px, cy-k(32), 7*px, 2*px, white) // the cap
	plot(cx-2*px, cy-k(16), px, px, color.RGBA{0x7a, 0xc8, 0xf0, 255})
}

// drawPaintPotPix: a small pot of paint with a handle, its bottom at (x, y).
func drawPaintPotPix(plot plotFn, x, y, px int, col color.RGBA) {
	plot(x-2*px, y-4*px, 5*px, 4*px, color.RGBA{0xb8, 0xbc, 0xc4, 255})
	plot(x-2*px, y-4*px, 5*px, px, col)
	plot(x-2*px, y-6*px, px, 2*px, pevInk)
	plot(x+2*px, y-6*px, px, 2*px, pevInk)
	plot(x-px, y-6*px, 3*px, px, pevInk)
}
