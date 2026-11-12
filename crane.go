package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE CRANE OF THE ROADWORKS (idea 304 of the user: "grúa de obra (cable activable: el bloque aplasta)").
//
// A small tower crane ("workcrane") stands on the pavement at one end of every roadworks zone (roadworks.go): only its
// concrete base is solid. Its jib reaches back over the works and a pallet of bricks hangs from the cable, in the air,
// over the place of one of the workers (rwLoadAt). The player can drop it with the use key (the same as for traps and
// bodies): "[E] Soltar la carga" within craneReach of the base (the label of useLabel and the action share the reach).
// The cable gives way, the pallet falls (its shadow grows on the ground), and whoever is right under it (craneHitR) is
// crushed: guests, guards, cops, animals, killers and the players too, like any trap. The dead are the player's with the
// rules of the traps (g.credit, the "traps" statistic, the owner in 2P, no witness of the hand). If nobody is under it
// it falls all the same, with a cloud of dust. Once per crane: the bricks stay on the ground, broken.
// ---------------------------------------------------------------------------

const (
	craneName  = "workcrane"
	craneW     = 104
	craneH     = 112
	craneJib   = 45.0 // x from the mast to the hook
	craneTipH  = 92.0 // height of the hook above the base
	craneLoadH = 40.0 // the bottom of the load is this high above the ground
	craneFall  = 520.0
	craneHitR  = 13.0
	craneReach = 30.0
)

var craneLabelES = pevTexts("Soltar la carga", "¡Cuidado abajo!", "¡Se suelta la carga de la grúa!", "Te ha aplastado la carga de la grúa")

func init() {
	propKinds[craneName] = &PropKind{W: craneW, H: craneH, FW: 14, FH: 5, Vars: 2, Category: "decor", Draw: drawWorkCrane}
}

// drawWorkCrane: v 0 the jib to the left, 1 to the right.
func drawWorkCrane(b *pixBuf, v int) {
	r := func(x, y, w, h int, cc color.RGBA) {
		if v == 1 {
			x = craneW - x - w
		}
		b.rect(x, y, w, h, cc)
	}
	ln := func(x0, y0, x1, y1 int, cc color.RGBA) {
		if v == 1 {
			x0, x1 = craneW-1-x0, craneW-1-x1
		}
		thickLine(b, float64(x0), float64(y0), float64(x1), float64(y1), 1, cc)
	}
	yel, yelD, yelL := c(0xf0c020), c(0x8a6a10), c(0xfae27a)
	conc, concD := c(0x9a9a9e), c(0x5e5e64)
	// the concrete base
	r(44, 104, 16, 8, conc)
	r(44, 110, 16, 2, concD)
	r(44, 104, 16, 1, c(0xc4c4c8))
	// the mast: two rails and a lattice
	r(48, 22, 1, 82, yelD)
	r(55, 22, 1, 82, yelD)
	r(49, 22, 6, 82, yel)
	for y := 24; y < 102; y += 6 {
		r(49, y, 6, 1, yelD)
		ln(49, y+1, 54, y+5, yelD)
	}
	r(49, 22, 1, 82, yelL)
	// the jib (to the hook) and the counter-jib with its weight
	r(6, 14, 43, 1, yelD)
	r(6, 18, 43, 1, yelD)
	r(6, 15, 43, 3, yel)
	for x := 8; x < 48; x += 5 {
		ln(x, 15, x+2, 17, yelD)
	}
	r(56, 14, 18, 1, yelD)
	r(56, 18, 18, 1, yelD)
	r(56, 15, 18, 3, yel)
	r(65, 19, 9, 7, conc)
	r(65, 25, 9, 1, concD)
	// the cabin under the jib, with its window toward the hook
	r(46, 18, 12, 6, yelD)
	r(47, 19, 10, 4, yel)
	r(47, 19, 4, 3, c(0x8ad0f0))
	// the top of the tower and the ties to the ends
	r(51, 3, 3, 12, yel)
	r(51, 3, 1, 12, yelD)
	r(51, 2, 3, 1, c(0xe02a2a)) // the warning light
	ln(51, 4, 8, 14, c(0x3a3a40))
	ln(53, 4, 72, 14, c(0x3a3a40))
	// the trolley at the end of the jib
	r(4, 19, 6, 2, c(0x2a2a30))
	r(6, 21, 2, 1, c(0x2a2a30))
}

func (l *Level) craneProp() *Prop {
	if l == nil {
		return nil
	}
	for _, pr := range l.Props {
		if pr.Name == craneName && !pr.gone {
			return pr
		}
	}
	return nil
}

// craneUseAt: a place beside the base from where the player reaches it (checked when the works are placed).
func craneUseAt(cr Vec, sd, cs float64) Vec { return Vec{cr.X + cs*16, cr.Y + sd*2} }

// ---- the drop ----------------------------------------------------------------------------------------------

const (
	crReady = iota
	crFalling
	crDown
)

type craneState struct {
	phase int
	t0    float64
	owner int // the player who dropped it (2P: the dead are his), -1 none
	by    *Player
	dead  int // how many it crushed
}

func craneTip(s *rwSite) Vec {
	return Vec{s.crane.X - s.cs*craneJib, s.crane.Y - craneTipH}
}

// craneHeight: the height of the bottom of the load above the ground now.
func (g *Game) craneHeight(run *rwRun) float64 {
	switch run.crane.phase {
	case crReady:
		return craneLoadH
	case crFalling:
		t := g.time - run.crane.t0
		return math.Max(0, craneLoadH-0.5*craneFall*t*t)
	}
	return 0
}

// craneCanDrop: the load still hangs and p can reach the base.
func (g *Game) craneCanDrop(run *rwRun, p *Player) bool {
	if run == nil || run.site.crane == nil || run.crane.phase != crReady || p == nil || p.dead || p.hidden || p.carrying != nil {
		return false
	}
	return distRect(p.pos, run.site.crane.Foot) < craneReach
}

// craneLabel: the text of the crane (items.go useLabel).
func (g *Game) craneLabel(pr *Prop) (string, color.RGBA) {
	run := g.rwActive()
	if run == nil || run.site.crane != pr || !g.craneCanDrop(run, g.p) {
		return "", color.RGBA{}
	}
	return craneLabelES[0], rgb(0xffa040)
}

// craneInteract: the use key beside the crane drops the load (game.go interact).
func (g *Game) craneInteract(p *Player) bool {
	run := g.rwActive()
	if !g.craneCanDrop(run, p) {
		return false
	}
	g.craneDrop(run, p)
	return true
}

func (g *Game) craneDrop(run *rwRun, p *Player) {
	run.crane.phase, run.crane.t0, run.crane.by, run.crane.owner = crFalling, g.time, p, -1
	if g.is2P() && p != nil {
		run.crane.owner = g.playerIndex(p)
	}
	g.snd.Play("squeak")
	g.say(craneLabelES[2])
	// the nearest worker shouts
	L := run.site.load()
	var best *Guest
	for _, w := range run.ws {
		if gu := w.m.gu; !w.gone && !gu.out() && dist(gu.pos, L) < 140 && (best == nil || dist(gu.pos, L) < dist(best.pos, L)) {
			best = gu
		}
	}
	if best != nil {
		best.icon, best.iconT = craneLabelES[1], 1.6
	}
}

func (g *Game) craneUpdate(run *rwRun, dt float64) {
	if run.crane.phase != crFalling || g.craneHeight(run) > 0 {
		return
	}
	run.crane.phase = crDown
	g.craneImpact(run)
}

// craneImpact: the load hits the ground: whoever is right under it is crushed (the player's kill, like a trap).
func (g *Game) craneImpact(run *rwRun) {
	L := run.site.load()
	in := func(v Vec) bool { return math.Abs(v.X-L.X) < craneHitR && math.Abs(v.Y-L.Y) < craneHitR*0.8 }
	g.coopCreditOwner(run.crane.owner, func() {
		g.credit = true
		defer func() { g.credit = false }()
		for _, gu := range g.guests {
			if !gu.out() && in(gu.pos) {
				g.trapKill(gu)
				run.crane.dead++
			}
		}
		for _, gd := range g.guards {
			if !gd.dead && in(gd.pos) {
				g.trapGuard(gd)
			}
		}
		for _, c := range append([]*Cop(nil), g.cops...) {
			if in(c.pos) {
				g.trapCop(c)
			}
		}
		g.trapHitOthers(in)
	})
	g.eachPlayer(func(p *Player) {
		if !p.dead && !p.hidden && math.Abs(p.pos.X-L.X) < craneHitR-2 && math.Abs(p.pos.Y-L.Y) < craneHitR*0.7 {
			g.die(craneLabelES[3])
		}
	})
	for _, gu := range g.guests {
		if !gu.out() && dist(gu.pos, L) < 160 && run.crane.dead > 0 {
			g.alarm(gu, "!?")
		}
	}
	g.snd.Play("thud")
	g.shake = math.Max(g.shake, 0.35)
	r := rwDice()
	for i := 0; i < 26; i++ {
		a := r.Float64() * 2 * math.Pi
		sp := 20 + r.Float64()*40
		l := 0.6 + r.Float64()*0.6
		g.parts = append(g.parts, Particle{pos: Vec{L.X, L.Y - 3}, vel: Vec{math.Cos(a) * sp, math.Sin(a)*sp*0.5 - 12}, life: l, max: l,
			col: color.RGBA{0xb8, 0xac, 0x94, 255}, size: 3, grav: -8})
	}
	g.checkDone()
}

// craneItems: the cable and the load (hanging, falling or broken on the ground) and its shadow.
func (g *Game) craneItems(dst *ebiten.Image, run *rwRun, items []drawItem) []drawItem {
	s := run.site
	if s.crane == nil {
		return items
	}
	L := s.load()
	return append(items, drawItem{L.Y + 0.4, func() { g.craneDraw(dst, run) }})
}

func (g *Game) craneDraw(dst *ebiten.Image, run *rwRun) {
	s := run.site
	L := s.load()
	tip := craneTip(s)
	h := g.craneHeight(run)
	lx, ly := g.wx(L.X), g.wy(L.Y)
	tx, ty := g.wx(tip.X), g.wy(tip.Y)
	if run.crane.phase == crDown {
		g.craneDrawRubble(dst, lx, ly)
		fillRect(dst, tx, ty, 1, 6, rgb(0x2a2a30)) // the broken cable
		return
	}
	// the shadow grows as it comes down
	u := 1 - h/craneLoadH
	rx := math.Round(4 + 5*u)
	filledEllipse(dst, lx, ly, rx, math.Round(rx*0.45), rgba(0x000000, uint8(60+80*u)))
	top := ly - math.Round(h) - 10
	if run.crane.phase == crReady {
		fillRect(dst, tx, ty, 1, top-ty, rgb(0x2a2a30))
		fillRect(dst, tx-1, top-2, 3, 2, rgb(0x5a5a62)) // the hook
	} else {
		fillRect(dst, tx, ty, 1, 6, rgb(0x2a2a30))
	}
	g.craneDrawLoad(dst, lx, top)
}

// craneDrawLoad: a wooden pallet with a stack of red bricks and two straps (top-left at x-7, y).
func (g *Game) craneDrawLoad(dst *ebiten.Image, cx, y float64) {
	x := cx - 7
	brick, mortar, wood, woodD, strap := rgb(0xb8442c), rgb(0xd8c8b0), rgb(0xa87840), rgb(0x6a4820), rgb(0x2a2a34)
	fillRect(dst, x, y, 14, 8, brick)
	for row := 0; row < 4; row++ {
		fillRect(dst, x, y+float64(row*2+1), 14, 1, mortar)
		off := float64(row%2) * 2
		for bx := 2.0 + off; bx < 14; bx += 4 {
			fillRect(dst, x+bx, y+float64(row*2), 1, 1, mortar)
		}
	}
	fillRect(dst, x+4, y, 1, 8, strap)
	fillRect(dst, x+9, y, 1, 8, strap)
	fillRect(dst, x-1, y+8, 16, 2, wood)
	fillRect(dst, x-1, y+9, 16, 1, woodD)
}

func (g *Game) craneDrawRubble(dst *ebiten.Image, cx, cy float64) {
	brick, wood, woodD := rgb(0xb8442c), rgb(0xa87840), rgb(0x6a4820)
	fillRect(dst, cx-9, cy-3, 18, 2, wood)
	fillRect(dst, cx-9, cy-1, 18, 1, woodD)
	fillRect(dst, cx-6, cy-6, 12, 3, brick)
	for _, o := range [][2]float64{{-13, 0}, {11, -2}, {-4, 2}, {7, 3}, {-10, -5}, {13, 2}} {
		fillRect(dst, cx+o[0], cy+o[1], 3, 2, brick)
	}
}

// ---- debug stages (-pevent obras:crane / obras:drop) -------------------------------------------------------

func (g *Game) rwStageDebug(run *rwRun) {
	if run.staged || !rwDebug() || g.p == nil {
		return
	}
	st := rwStage()
	if st != "crane" && st != "drop" {
		run.staged = true
		return
	}
	if g.time < 0.3 {
		return
	}
	cr := run.site.crane
	if cr == nil {
		run.staged = true
		return
	}
	if !run.placed {
		run.placed = true
		g.p.pos = g.lvl.NearestFree(craneUseAt(Vec{cr.X, cr.Y}, run.site.sd, run.site.cs))
	}
	if st == "crane" {
		run.staged = true
	} else if g.time > 2.5 {
		g.craneDrop(run, g.p)
		run.staged = true
	}
}
