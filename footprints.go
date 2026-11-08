package main

import (
	"image/color"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Footprint trails (footprints.go): on special ground -- sand on the beaches,
// snow at the ski station, mud at the festival, wet floor around the pools --
// everybody who walks leaves little prints (a 2x3 px foot), left foot / right foot,
// pointing the way they go. They fade out linearly in 6-10 s.
//
// Rules: pure decoration. Drawn right after the floor and the ambient layer
// (below props, bodies and actors); no hitboxes, no sort order, no 2P changes.
// Update only runs while the game runs (the help window / pause skip it, so
// prints do not age). A ring buffer of fpMax prints and a fixed table of
// trackers: nothing is allocated per frame. Reset when the level changes.
// footprintsOn=false (PH_FOOTPRINTS=0) switches it all off.
// ---------------------------------------------------------------------------

var footprintsOn = os.Getenv("PH_FOOTPRINTS") != "0"

const (
	fpMax    = 120 // prints alive at most
	fpTracks = 192 // actors followed at most
)

type fpStyle uint8

const (
	fpNone fpStyle = iota
	fpSand
	fpSnow
	fpMud
	fpWet
)

// footprintPlan: the levels with prints (by title, like ambientPlan) and the ground they need.
var footprintPlan = []struct {
	Title string
	Style fpStyle
}{
	{"MIAMI BEACH", fpSand},
	{"PLAYA", fpSand},
	{"ESTACIÓN DE ESQUÍ", fpSnow},
	{"FESTIVAL", fpMud},
	{"CRUCERO NOCTURNO", fpWet},
}

func footprintStyleFor(title string) fpStyle {
	for _, e := range footprintPlan {
		if e.Title == title {
			return e.Style
		}
	}
	return fpNone
}

// fpGround: does the style apply at this point? (the topmost floor decides; wet = beside a pool)
func fpGround(l *Level, st fpStyle, p Vec) bool {
	if st == fpWet {
		for _, r := range l.Pools {
			if r.Has(p.X, p.Y) {
				return false
			}
		}
		for _, r := range l.Pools {
			if r.Grow(12).Has(p.X, p.Y) {
				return true
			}
		}
		return false
	}
	want := "sand"
	switch st {
	case fpSnow:
		want = "snow"
	case fpMud:
		want = "dirt"
	}
	kind := ""
	for _, f := range l.Def.Floors {
		if f.R.Has(p.X, p.Y) {
			kind = f.Kind
		}
	}
	return kind == want
}

type fpPrint struct {
	x, y   float64
	t0     float64
	life   float64
	horiz  bool // the foot points along x
	strong bool // the player's prints: a bit darker
}

type fpTrack struct {
	key  any
	last Vec
	acc  float64
	side bool
}

type fpState struct {
	lvl    *Level
	style  fpStyle
	t      float64
	ring   [fpMax]fpPrint
	head   int // next slot
	n      int // prints stored
	tracks [fpTracks]fpTrack
	nt     int
	hint   int
	seq    int
	drawn  int // prints painted in the last draw (tests read it)
}

var fp fpState

// fpSync: forget everything when the level changed.
func (g *Game) fpSync() bool {
	if g.lvl == nil || g.lvl.Def == nil {
		return false
	}
	if fp.lvl != g.lvl {
		fp.lvl, fp.t, fp.head, fp.n, fp.seq, fp.nt, fp.hint, fp.drawn = g.lvl, 0, 0, 0, 0, 0, 0, 0
		for i := range fp.tracks {
			fp.tracks[i] = fpTrack{}
		}
		fp.style = footprintStyleFor(g.lvl.Def.Title)
	}
	return fp.style != fpNone
}

// updateFootprints follows everybody who walks (called from updatePlay, so it stops with the pause).
func (g *Game) updateFootprints(dt float64) {
	if !footprintsOn || g.mode != ModePlay {
		return
	}
	if !g.fpSync() {
		return
	}
	fp.t += dt
	fp.hint = 0
	if g.is2P() {
		g.fpPlayer(g.first())
		g.fpPlayer(g.p2)
	} else {
		g.fpPlayer(g.p)
	}
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" {
			g.fpStep(gu, gu.pos, false, 12)
		}
	}
	for _, gd := range g.guards {
		g.fpStep(gd, gd.pos, false, 13)
	}
	for _, c := range g.cops {
		g.fpStep(c, c.pos, false, 13)
	}
}

func (g *Game) fpPlayer(p *Player) {
	if p == nil || p.dead || p.hidden {
		return
	}
	g.fpStep(p, p.pos, true, 11)
}

// fpStep: accumulates the distance walked; every ~10-14 px a print.
func (g *Game) fpStep(key any, pos Vec, strong bool, step float64) {
	i := fp.hint
	if i >= fp.nt || fp.tracks[i].key != key {
		i = -1
		for j := 0; j < fp.nt; j++ {
			if fp.tracks[j].key == key {
				i = j
				break
			}
		}
		if i < 0 {
			if fp.nt >= fpTracks {
				return
			}
			i = fp.nt
			fp.nt++
			fp.tracks[i] = fpTrack{key: key, last: pos}
			fp.hint = i + 1
			return
		}
	}
	fp.hint = i + 1
	tr := &fp.tracks[i]
	dx, dy := pos.X-tr.last.X, pos.Y-tr.last.Y
	d := dx*dx + dy*dy
	tr.last = pos
	if d == 0 {
		return
	}
	if d > 24*24 { // jumped (teleport, new spot): no trail
		tr.acc = 0
		return
	}
	tr.acc += math.Sqrt(d)
	jit := 10 + 4*jpHash(fp.seq, i, 7) // 10-14 px between prints
	jit = jit * step / 12
	if tr.acc < jit {
		return
	}
	tr.acc = 0
	if pos.X < 0 || pos.Y < 0 || pos.X >= screenW || pos.Y >= screenH || !fpGround(g.lvl, fp.style, pos) {
		return
	}
	tr.side = !tr.side
	horiz := dx*dx > dy*dy
	off := 2.0
	if tr.side {
		off = -2
	}
	pr := fpPrint{x: pos.X, y: pos.Y, t0: fp.t, life: 6 + 4*jpHash(fp.seq, i, 11), horiz: horiz, strong: strong}
	if horiz {
		pr.y += off
	} else {
		pr.x += off
	}
	fp.seq++
	fp.ring[fp.head] = pr
	fp.head = (fp.head + 1) % fpMax
	if fp.n < fpMax {
		fp.n++
	}
}

var fpOp ebiten.DrawImageOptions

// fpRects: rectangles painted (tests read it).
var fpRects int

func fpRect(dst *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	if c.A == 0 {
		return
	}
	fpRects++
	fpOp.GeoM.Reset()
	fpOp.GeoM.Scale(w, h)
	fpOp.GeoM.Translate(x, y)
	fpOp.ColorScale.Reset()
	fpOp.ColorScale.ScaleWithColor(c)
	dst.DrawImage(px, &fpOp)
}

// drawFootprints: one pass over the ring, right after the floor (below everything else).
func (g *Game) drawFootprints(dst *ebiten.Image) {
	fp.drawn = 0
	if !footprintsOn || px == nil || !g.fpSync() || fp.n == 0 {
		return
	}
	// dark print + a light rim on its lower side (the pushed-up sand / snow) so it reads on any tint
	var r, gg, b, rr, rg, rb uint8
	var a0, ar float64
	switch fp.style {
	case fpSand:
		r, gg, b, a0, rr, rg, rb, ar = 28, 18, 8, 215, 255, 226, 170, 170
	case fpSnow:
		r, gg, b, a0, rr, rg, rb, ar = 40, 62, 120, 215, 255, 255, 255, 170
	case fpMud:
		r, gg, b, a0, rr, rg, rb, ar = 22, 12, 4, 220, 190, 140, 90, 130
	default:
		r, gg, b, a0, rr, rg, rb, ar = 50, 130, 235, 220, 220, 245, 255, 130
	}
	for i := 0; i < fp.n; i++ {
		p := &fp.ring[i]
		k := 1 - (fp.t-p.t0)/p.life
		if k <= 0 {
			continue
		}
		boost := 1.0
		if p.strong {
			boost = 1.4
		}
		x, y := g.wx(p.x), g.wy(p.y)
		w, h := 2.0, 3.0
		if p.horiz {
			w, h = 3, 2
		}
		fpRect(dst, x+1, y+1, w, h, bgCol(rr, rg, rb, ar*k*boost))
		fpRect(dst, x, y, w, h, bgCol(r, gg, b, a0*k*boost))
		if fp.style == fpWet { // a drop beside the foot
			fpRect(dst, x+2, y-1, 1, 1, bgCol(r, gg, b, a0*k*0.7))
		}
		fp.drawn++
	}
}
