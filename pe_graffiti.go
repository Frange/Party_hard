package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE STREET PAINTER (idea 288, approved: "si se puede hacer bien sí, que ponga graffitis bonitos").
//
// A reactive, Minor party event "graffiti" (it never takes one of the places of the frame, like the petting and the
// mechanical bull, so the mean of planned events per party does not move): in SOME parties (gfPartyChance) of the
// levels with a free house wall that faces the street (the facades of the house, the ranch, the disco, the cinema, the
// casino, the birthday house, the 70s disco, the Paris blocks, the bullring and the stadium: gfSegments decides it from
// the walls themselves, never from a list), a guest puts on a cap, a hood and a backpack, walks to the wall, faces it
// and PAINTS a piece of the graffiti library (pe_graffiti_art.go) with a spray can: first the outline, then the colours,
// in 8-15 s, the painter sliding along the wall with the stroke, a cloud of paint at the nozzle. Then he admires it and
// goes on with the party. The piece STAYS on the wall for the rest of the party (drawn over the wall and under every
// character, never solid), at most gfMaxPieces (2-3) per party, never on a door, a picture, a camera, a prop or another
// piece, and never the same design twice in a party (nor in the session until all were used).
//
// The painter is a plain guest all the time (pevMem): he can be killed (he is a witness, he counts for every cap; nobody
// is added). Killed while painting: he drops the can, which drips paint a moment, and the piece stays half done (drawn
// up to the progress it had). Frightened, alarmed or with the police near: he stops and the piece stays as it is. Every
// phase has its own time limit; if no wall or no painter is found, nothing starts. Dice: gfDice() (never g.rng, never
// the frame's dice). -pevent graffiti forces it (progress frozen); -pevent graffiti:muro also leaves two finished pieces
// on the walls at the start (for the captures).
// ---------------------------------------------------------------------------

const (
	gfPartyChance = 0.45 // parties of a level with a free wall that get a painter at all
	gfPaintMin    = 8.0
	gfPaintMax    = 15.0
	gfWalkMax     = 15.0
	gfAdmireT     = 1.6
	gfPainterR    = 360.0 // the painter is a free guest this near the wall
	gfFloorGap    = 7.0   // the painter's feet: this far under the wall's foot
	gfEndGap      = 6.0   // margin to the ends of a wall run (jambs, corners)
	gfPieceGap    = 16.0  // between two pieces
	gfTryMax      = 10
)

var (
	gfLinesStart = pevTexts("Esta pared pide color", "Arte urbano, señores", "Un momento, que me inspiro", "¡Lienzo libre!")
	gfLinesHiss  = pevTexts("*fsssssh*", "*chk chk fsssh*")
	gfLinesEnd   = pevTexts("¡Obra maestra!", "Firmado: el artista", "Esto vale millones", "¡Arte para el pueblo!")
)

var (
	gfRng      *rand.Rand
	gfSeed     int64 // tests: non-zero = deterministic dice
	gfInTests  bool  // tests: let it happen under go test (off by default so the other tests' parties stay as they were)
	gfTestsOff bool  // set by the tests' init (testiso_test.go): the graffiti event is off under go test unless gfInTests
	gfUsed     = map[int]bool{}
)

func gfDice() *rand.Rand {
	if gfRng == nil {
		sd := gfSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x6a7f1b
		}
		gfRng = rand.New(rand.NewSource(sd))
	}
	return gfRng
}

// ---- the walls -------------------------------------------------------------------------------------------------

// gfSeg: a horizontal house wall whose face can take a piece. The piece's bottom sits on the wall's foot (bottom),
// its rows go up to the cornice's lip (top): avail rows.
type gfSeg struct {
	r      Rect    // the wall as drawn (wallFoot)
	x0, x1 float64 // the stretch of the face where a piece may go
	bottom float64 // y of the wall's foot
	avail  int     // rows of face (outline edge included) a piece may use
	avoid  []Rect  // pictures and windows painted on the face
}

// gfFaceTop: the first row of the wall face, as drawThickWall paints it (under the cornice and its lip).
func gfFaceTop(r Rect) float64 {
	in := Rect{r.X + 1, r.Y + 1, r.W - 2, r.H - 2}
	capH := math.Min(4, math.Max(2, math.Round(in.H*0.4)))
	return in.Y + capH + 1
}

// gfWallDeco: the pictures (plaster, wood) and windows (tall brick) drawThickWall paints on a face (same rules).
func gfWallDeco(r Rect, style int) []Rect {
	in := Rect{r.X + 1, r.Y + 1, r.W - 2, r.H - 2}
	fy := gfFaceTop(r)
	fh := in.Y + in.H - fy
	var out []Rect
	if (style == WallPlaster || style == WallWood) && fh >= 8 {
		for x := 8.0; x+10 < in.W; x += 36 {
			h := uint32(int(in.X+x)*2654435761) >> 8
			if h%3 != 0 {
				continue
			}
			pw := 6 + float64(h%5)
			out = append(out, Rect{in.X + x - 1, fy + 1, pw + 2, 7})
		}
	}
	if style == WallBrick && r.H > 26 {
		for x := 7.0; x+9 < in.W; x += 26 {
			out = append(out, Rect{in.X + x, fy + 6, 9, 10})
		}
	}
	return out
}

// gfSegments: the walls of a level that face open street floor: a house wall, horizontal, long enough, with a face of
// 9+ rows and walkable outdoor floor all along its foot (the painter's strip).
func gfSegments(l *Level) []gfSeg {
	var out []gfSeg
	if l == nil || l.Def == nil {
		return nil
	}
	for _, w := range l.Def.Walls {
		if !wallThick(w.Style) || w.R.W < w.R.H*2 {
			continue
		}
		r := sr(wallFoot(w))
		if r.Y < 0 || r.W < 40 {
			continue
		}
		bottom := r.Y + r.H
		avail := int(bottom - gfFaceTop(r))
		if avail < 9 {
			continue
		}
		s := gfSeg{r: r, bottom: bottom, avail: avail, avoid: gfWallDeco(r, w.Style)}
		// the longest run of good floor under the wall
		y := bottom + gfFloorGap
		best0, best1, run0 := 0.0, 0.0, -1.0
		for x := r.X + gfEndGap; x <= r.X+r.W-gfEndGap; x += 2 {
			p := Vec{x, y}
			ok := l.Free(p) && !l.SolidP(p.X, p.Y) && l.Outdoor(p) && !l.badGround(p) && !l.InPool(p) &&
				l.Free(Vec{x, y + 6}) && l.Outdoor(Vec{x, y + 6})
			if ok && run0 < 0 {
				run0 = x
			}
			if (!ok || x+2 > r.X+r.W-gfEndGap) && run0 >= 0 {
				end := x
				if !ok {
					end = x - 2
				}
				if end-run0 > best1-best0 {
					best0, best1 = run0, end
				}
				run0 = -1
			}
		}
		if best1-best0 < 40 {
			continue
		}
		s.x0, s.x1 = best0, best1
		out = append(out, s)
	}
	return out
}

// ---- the pieces on the walls (persistent for the party) -----------------------------------------------------------

type gfPiece struct {
	art  *gfArt
	x, y float64 // top-left on screen/world (whole pixels)
	prog float64 // 0..1 painted
}

func (p *gfPiece) rect() Rect { return Rect{p.x, p.y, float64(p.art.w), float64(p.art.h)} }

// gfCan: a can dropped by a killed painter: it lies on the floor and drips paint for a moment.
type gfCan struct {
	pos Vec
	t0  float64
	col color.RGBA
}

type gfLevel struct {
	lvl    *Level
	segs   []gfSeg
	pieces []*gfPiece
	cans   []gfCan
	want   bool // this party has a painter
	max    int  // pieces this party may get (2-3)
	next   float64
	tries  int
	n      int // painters started
}

var gfLv gfLevel

func gfReset(g *Game) {
	r := gfDice()
	gfLv = gfLevel{lvl: g.lvl, segs: gfSegments(g.lvl)}
	gfLv.want = r.Float64() < gfPartyChance
	gfLv.max = 2
	if r.Float64() < 0.35 {
		gfLv.max = 3
	}
	gfLv.next = g.time + 35 + r.Float64()*120
	if skateGraffiti(g.lvl.Def) && len(gfLv.segs) > 0 { // SKATE PARK: 1-3 painters ALWAYS (skategraffiti.go)
		gfLv.want, gfLv.max, gfLv.next = true, 1+r.Intn(3), g.time+skateGfFirst+r.Float64()*skateGfFirstVar
	}
	if pevDebugOn("graffiti") {
		gfLv.want, gfLv.next, gfLv.max = true, g.time+2, 3
		if pevDemo("graffiti") == "muro" { // two finished pieces at the start (captures)
			for i := 0; i < 2; i++ {
				if p := gfPlace(g, r); p != nil {
					p.prog = 1
				}
			}
		}
	}
}

// gfLevelOf: the persistent state of this party (reset when the level changes).
func gfLevelOf(g *Game) *gfLevel {
	if gfLv.lvl != g.lvl {
		gfReset(g)
	}
	return &gfLv
}

// gfBlocked: the rectangle (in the face) would cover a door, a picture, a camera, a prop or another piece.
func gfBlocked(g *Game, s gfSeg, rc Rect) bool {
	for _, a := range s.avoid {
		if rc.Grow(2).Hits(a) {
			return true
		}
	}
	for _, d := range g.lvl.Def.Doors {
		if rc.Grow(8).Hits(sr(doorFoot(d))) {
			return true
		}
	}
	for _, c := range g.cams {
		if c != nil && rc.Grow(10).Has(c.mount.X, c.mount.Y) {
			return true
		}
	}
	for _, pr := range g.lvl.Props {
		if pr.K == nil {
			continue
		}
		w, h := float64(pr.K.W), float64(pr.K.H)
		if rc.Grow(3).Hits(Rect{pr.X - w/2, pr.Y - h, w, h + 2}) {
			return true
		}
	}
	for _, p := range gfLv.pieces {
		if rc.Grow(gfPieceGap).Hits(p.rect()) {
			return true
		}
	}
	return false
}

// gfPickArt: a design that fits the rows and was not painted yet in this party (nor, while some are left, in the session).
func gfPickArt(r *rand.Rand, avail int) int {
	var free, fresh []int
	inParty := map[*gfArt]bool{}
	for _, p := range gfLv.pieces {
		inParty[p.art] = true
	}
	for i, a := range gfLib {
		if a.h > avail || inParty[a] {
			continue
		}
		free = append(free, i)
		if !gfUsed[i] {
			fresh = append(fresh, i)
		}
	}
	if len(free) == 0 {
		return -1
	}
	if len(fresh) == 0 { // the session saw them all: start again
		for i := range gfUsed {
			delete(gfUsed, i)
		}
		fresh = free
	}
	return fresh[r.Intn(len(fresh))]
}

// gfPlace finds a free stretch of wall and a design for it and adds the (unpainted) piece. nil = no room.
func gfPlace(g *Game, r *rand.Rand) *gfPiece {
	if len(gfLv.segs) == 0 {
		return nil
	}
	order := r.Perm(len(gfLv.segs))
	for _, si := range order {
		s := gfLv.segs[si]
		ai := gfPickArt(r, s.avail)
		if ai < 0 {
			continue
		}
		a := gfLib[ai]
		span := s.x1 - s.x0 - float64(a.w) - 10 // room for the painter beside the last stroke
		if span < 0 {
			continue
		}
		for k := 0; k < 12; k++ {
			x := math.Round(s.x0 + r.Float64()*span)
			y := s.bottom - float64(a.h)
			rc := Rect{x, y, float64(a.w), float64(a.h)}
			if gfBlocked(g, s, rc) {
				continue
			}
			p := &gfPiece{art: a, x: x, y: y}
			gfLv.pieces = append(gfLv.pieces, p)
			gfUsed[ai] = true
			return p
		}
	}
	return nil
}

func gfDropPiece(p *gfPiece) {
	for i, q := range gfLv.pieces {
		if q == p {
			gfLv.pieces = append(gfLv.pieces[:i], gfLv.pieces[i+1:]...)
			return
		}
	}
}

// ---- the event ---------------------------------------------------------------------------------------------------

func init() {
	registerPartyEvent(&partyEvent{ID: "graffiti", Name: "Grafitero", Group: "graffiti", Reactive: true, Minor: true,
		Prob: gfPartyChance, AtMin: 35, AtMax: 900, Dur: 30,
		Eligible: gfDue, Start: gfStart, Items: gfItems,
		Update: pevGuard(gfUpdate, func(g *Game, a *partyAct) { gfEnd(g, a.D.(*gfState)) })})
}

const (
	gfWalk = iota
	gfPaint
	gfAdmire
)

type gfState struct {
	m     pevMem
	piece *gfPiece
	phase int
	t     float64
	dur   float64
	stand Vec // where he stands when the stroke starts
	said  bool
	hiss  bool
	col   color.RGBA // the main colour of the piece (the can)
}

func (s *gfState) pevHas(gu *Guest) bool { return s.m.gu == gu }

// gfDue: the condition of the reactive event: this party has a painter, the time has come, there is room for a piece.
func gfDue(g *Game, t partyTraits) bool {
	if g.lvl == nil {
		return false
	}
	if gfTestsOff && !gfInTests && !pevDebugOn("graffiti") {
		return false
	}
	lv := gfLevelOf(g)
	if !lv.want || len(lv.segs) == 0 || len(lv.pieces) >= lv.max || lv.tries >= gfTryMax || g.time < lv.next {
		return false
	}
	return true
}

// gfMainCol: the most used colour of a design (besides the outline): the colour of its can.
func gfMainCol(a *gfArt) color.RGBA {
	n := map[color.RGBA]int{}
	best, bn := color.RGBA{0xff, 0x40, 0xa0, 255}, 0
	for _, c := range a.pix {
		if c.A == 0 || c == gfInk {
			continue
		}
		n[c]++
		if n[c] > bn {
			best, bn = c, n[c]
		}
	}
	return best
}

func gfStart(g *Game, a *partyAct) bool {
	r := gfDice()
	lv := &gfLv
	lv.next = g.time + 6 // if it cannot start now, ask again in a while
	lv.tries++
	p := gfPlace(g, r)
	if p == nil {
		return false
	}
	stand := Vec{p.x + 6, p.y + float64(p.art.h) + gfFloorGap}
	// the painter: the nearest free guest who can walk there
	var best *Guest
	bd := gfPainterR
	if pevDebugOn("graffiti") {
		bd = 1e9
	}
	for _, gu := range g.guests {
		if !g.pevFreeGuest(gu) || isKid(gu) {
			continue
		}
		if d := dist(gu.pos, stand); d < bd {
			best, bd = gu, d
		}
	}
	if best == nil || g.sackAlarmNear(stand, 140) {
		gfDropPiece(p)
		return false
	}
	s := &gfState{piece: p, stand: stand, col: gfMainCol(p.art), dur: gfPaintMin + r.Float64()*(gfPaintMax-gfPaintMin)}
	s.m.gu = best
	if !g.pevSendTo(&s.m, stand) {
		gfDropPiece(p)
		return false
	}
	a.D = s
	lv.n++
	if pevDebugOn("graffiti") {
		fmt.Printf("PEVDEMO graffiti %s at %.0f %.0f\n", p.art.name, p.x, p.y)
	}
	return true
}

// gfEnd: the painter is let go; a piece with almost no paint is wiped, the rest stays as it is.
func gfEnd(g *Game, s *gfState) {
	if s == nil {
		return
	}
	r := gfDice()
	if s.m.mine() {
		pevRelease(&s.m, r)
	} else {
		pevForget(&s.m)
	}
	if s.piece != nil && s.piece.prog < 0.03 {
		gfDropPiece(s.piece)
	}
	gfLv.next = g.time + 60 + r.Float64()*80
	if skateGraffiti(g.lvl.Def) { // the skate park's crew comes one after another (skategraffiti.go)
		gfLv.next = g.time + skateGfNext + r.Float64()*skateGfNextVar
	}
	if pevDebugOn("graffiti") {
		gfLv.next = g.time + 4
	}
}

// gfStrokeX: the x of the nozzle on the wall at progress p (the outline pass, then the colour pass, left to right).
func (s *gfState) strokeX() float64 {
	p := s.piece
	u := p.prog / 0.4
	if p.prog > 0.4 {
		u = (p.prog - 0.4) / 0.6
	}
	return p.x + 1 + math.Min(1, u)*float64(p.art.w-2)
}

func gfUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*gfState)
	r := gfDice()
	gu := s.m.gu
	if gu.dead && s.phase == gfPaint { // killed with the can in his hand: it falls and drips
		gfLv.cans = append(gfLv.cans, gfCan{pos: Vec{math.Round(gu.pos.X - 6), math.Round(gu.pos.Y + 2)}, t0: g.time, col: s.col})
	}
	if !s.m.mine() || g.sackAlarmNear(s.stand, 140) {
		gfEnd(g, s)
		return true
	}
	s.t += dt
	switch s.phase {
	case gfWalk:
		if gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, s.stand) < 6 {
			s.phase, s.t = gfPaint, 0
			s.m.hold(gIdle)
			gu.dir = DirUp
			if gu.icon == "" {
				gu.icon, gu.iconT = pevPick(r, gfLinesStart), 2
			}
		} else if s.t > gfWalkMax && !(pevDebugOn("graffiti") && s.t < 40) {
			gfEnd(g, s)
			return true
		}
	case gfPaint:
		s.m.keep()
		gu.dir = DirUp
		s.piece.prog = math.Min(1, s.piece.prog+dt/s.dur)
		// he slides along the wall with the stroke, standing just right of it (the strip under the wall is free floor)
		want := math.Min(s.strokeX()+7, s.piece.x+float64(s.piece.art.w)+8)
		if d := want - gu.pos.X; math.Abs(d) > 0.5 {
			step := math.Min(math.Abs(d), 14*dt)
			nx := gu.pos.X + math.Copysign(step, d)
			if g.lvl.Free(Vec{nx, gu.pos.Y}) && !g.lvl.SolidP(nx, gu.pos.Y) {
				gu.pos.X = nx
			}
		}
		if !s.hiss && s.t > s.dur*0.45 && gu.icon == "" {
			s.hiss = true
			gu.icon, gu.iconT = pevPick(r, gfLinesHiss), 1.2
		}
		if s.piece.prog >= 1 {
			s.phase, s.t = gfAdmire, 0
			gu.dir = DirDown
			gu.icon, gu.iconT = pevPick(r, gfLinesEnd), 1.8
		}
	case gfAdmire:
		s.m.keep()
		if s.t >= gfAdmireT {
			gfEnd(g, s)
			return true
		}
	}
	return false
}

// ---- drawing -------------------------------------------------------------------------------------------------------

var (
	gfCapCol   = color.RGBA{0x2a, 0x2e, 0x3a, 255}
	gfCapBrim  = color.RGBA{0xf0, 0x40, 0x60, 255}
	gfHoodCol  = color.RGBA{0x3a, 0x6a, 0x5a, 255}
	gfHoodDark = color.RGBA{0x22, 0x44, 0x3a, 255}
	gfPackCol  = color.RGBA{0xe0, 0x8a, 0x20, 255}
	gfPackDark = color.RGBA{0x9a, 0x5a, 0x10, 255}
	gfCanCol   = color.RGBA{0xd8, 0xdc, 0xe4, 255}
)

// drawGraffiti paints the pieces of this party on their walls (after the floor, under every character) and the cans
// dropped by killed painters. One line in draw.go.
func (g *Game) drawGraffiti(dst *ebiten.Image) {
	if g.lvl == nil || gfLv.lvl != g.lvl || len(gfLv.pieces) == 0 && len(gfLv.cans) == 0 {
		return
	}
	for _, p := range gfLv.pieces {
		gfDrawPiece(g, dst, p)
	}
	plot := g.plotter(dst)
	for _, c := range gfLv.cans {
		x, y := g.pevXY(c.pos)
		gfDrawDroppedCan(plot, x, y, c.col, g.time-c.t0)
	}
}

// gfDrawPiece: the outline is sprayed first (to 40 %), then the colours; a half piece shows only what was painted.
func gfDrawPiece(g *Game, dst *ebiten.Image, p *gfPiece) {
	if p.prog <= 0 {
		return
	}
	full, line := p.art.images()
	x, y := g.wx(p.x), g.wy(p.y)
	c1 := int(math.Round(math.Min(1, p.prog/0.4) * float64(p.art.w)))
	c2 := 0
	if p.prog > 0.4 {
		c2 = int(math.Round(math.Min(1, (p.prog-0.4)/0.6) * float64(p.art.w)))
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	if c1 > c2 {
		dst.DrawImage(line.SubImage(image.Rect(0, 0, c1, p.art.h)).(*ebiten.Image), op)
	}
	if c2 > 0 {
		dst.DrawImage(full.SubImage(image.Rect(0, 0, c2, p.art.h)).(*ebiten.Image), op)
	}
}

// gfDrawDroppedCan: a can lying on the floor; for 2.5 s paint drips from its nozzle into a small puddle that stays.
func gfDrawDroppedCan(plot plotFn, x, y int, col color.RGBA, t float64) {
	pool := int(math.Min(5, 1+t*2))
	plot(x-pool, y+1, pool, 1, col)
	plot(x-pool+1, y+2, pool-1, 1, shade(col, 0.8))
	plot(x, y-1, 5, 3, gfCanCol)
	plot(x, y-1, 5, 1, pevWhite)
	plot(x+1, y, 3, 1, col)
	plot(x-1, y, 1, 1, pevInk) // the nozzle
	if t < 2.5 {
		for i := 0; i < 2; i++ {
			u := math.Mod(t*3+float64(i)*0.5, 1)
			plot(x-1, y+int(u*2), 1, 1, col)
		}
	}
}

// gfPainterKit: cap (worn backwards), hood and backpack on a guest whose feet are at (cx, cy).
func gfPainterKit(plot plotFn, cx, cy, px int, k func(float64) int, dir Dir) {
	top := cy - k(31)
	back := dir == DirUp
	side := 0
	switch dir {
	case DirLeft:
		side = 1
	case DirRight:
		side = -1
	}
	// the cap: a dark crown on the head, the red brim at the back of the head
	plot(cx-3*px, top-px, 7*px, 2*px, gfCapCol)
	plot(cx-2*px, top-2*px, 5*px, px, gfCapCol)
	switch {
	case back:
		plot(cx-2*px, top+px, 5*px, px, gfCapBrim)
	case side != 0:
		plot(cx+side*3*px-px, top, 3*px, px, gfCapBrim)
	default:
		plot(cx-2*px, top-2*px, 5*px, px, gfCapBrim)
	}
	if dir == DirDown {
		return // seen from the front: the hood and the pack are behind him
	}
	// the hood hanging at the nape and the backpack on his back
	hx := cx + side*2*px
	plot(hx-3*px, top+k(8), 7*px, 2*px, gfHoodCol)
	plot(hx-2*px, top+k(8)+2*px, 5*px, px, gfHoodDark)
	bx, bw := cx-3*px+side*3*px, 7*px
	if side != 0 {
		bw = 4 * px
		if side > 0 {
			bx = cx + 2*px
		} else {
			bx = cx - 6*px
		}
	}
	by := cy - k(22)
	plot(bx, by, bw, k(9), gfPackCol)
	plot(bx, by, bw, px, shade(gfPackCol, 1.2))
	plot(bx+px, by+k(4), bw-2*px, px, gfPackDark)
	plot(bx, by+k(9)-px, bw, px, gfPackDark)
}

// gfSprayCan: the can held in a hand at (x, y), the nozzle up-left toward the wall.
func gfSprayCan(plot plotFn, x, y, px int, col color.RGBA) {
	plot(x-px, y-px, 5*px, 7*px, pevInk) // outline, so it reads on any wall
	plot(x, y, 3*px, 5*px, gfCanCol)
	plot(x, y+px, 3*px, 3*px, col)
	plot(x, y+px, px, 3*px, shade(col, 1.3))
	plot(x+px, y-2*px, px, px, pevInk) // the nozzle
}

func gfItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s, ok := a.D.(*gfState)
	if !ok || s.m.gu == nil || s.m.gu.out() || !s.m.mine() {
		return items
	}
	gu := s.m.gu
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	items = append(items, drawItem{gu.pos.Y + 0.1, func() {
		cx, cy := g.pevXY(gu.pos)
		gfPainterKit(plot, cx, cy, px, k, gu.dir)
		if s.phase != gfPaint {
			gfSprayCan(plot, cx+4*px, cy-k(14), px, s.col)
			return
		}
		// the arm reaches the stroke; the can at the nozzle; a cloud of paint on the wall
		p := s.piece
		sx := int(math.Round(g.wx(s.strokeX())))
		row := math.Mod(g.time*7, 1)
		sy := int(math.Round(g.wy(p.y + 2 + row*float64(p.art.h-4))))
		shx, shy := cx-3*px, cy-k(22)
		hx, hy := sx+3*px, sy+2*px
		drawLinePix(plot, shx, shy, hx, hy, 2*px, gfHoodDark)
		drawLinePix(plot, shx, shy, hx, hy, px, gfHoodCol)
		plot(hx+px, hy+2*px, 2*px, 2*px, pevSkinW) // the hand
		gfSprayCan(plot, hx, hy-px, px, s.col)
		for i := 0; i < 10; i++ {
			ang := g.time*9 + float64(i)*0.63
			rr := 2 + math.Mod(g.time*5+float64(i)*0.37, 1)*4
			qx := sx + int(math.Round(math.Cos(ang)*rr))
			qy := sy + int(math.Round(math.Sin(ang)*rr*0.7))
			c := s.col
			al := uint8(190 - i*12)
			plot(qx, qy, px, px, color.RGBA{uint8(int(c.R) * int(al) / 255), uint8(int(c.G) * int(al) / 255), uint8(int(c.B) * int(al) / 255), al})
		}
	}})
	return items
}
