package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// VIP ZONE SHAPES, SIZES, SECURITY AND THE MINI VIP PARTY (v1.4.0, owner's feedback: "a VIP zone came out with no
// guards and no VIP; it should not always be the same").
//
//   - The zone is a set of 32x32 cells (z.cells): a rectangle (1-2 rows, 3-6 cells wide), ROUND (3x3 / 4x3 without
//     the corners, an elliptic carpet), OVAL (5x3 / 6x3 without the corners), an L (a top corner block missing, left
//     or right) or a PLATFORM (a raised dais with steps at the gate). The cordon follows the outline of the cells, so
//     the ropes are still the real props viprope_h / viprope_v (overlap and reach tests see them) and the gate is the
//     south side of the bottom cell nearest the middle.
//   - Security: low (2 door guards), normal (+1 inside), high (+2 inside). A MINI VIP PARTY has 2-3 VIPs and always
//     more security: 2 door guards and one inner guard per VIP (at least 2).
//   - NEVER EMPTY: every zone has at least one VIP and the two door guards (spawnVIP; an unknown VIP id falls back to a
//     registry VIP). The empty zone the owner saw came from the party killers: slasher and Exterminator picked the
//     zone staff and the VIP himself as ordinary guards (killer.go nearestSeen / slasherPick). They skip them now.
//   - The style is rolled with its OWN dice (vipStyleDice: g.rng and vipDice are not moved) in vipRoll; the zero style
//     is the classic carpet (tests that build a zone by hand keep it). When the rolled shape does not fit the level,
//     smaller shapes and finally the classic sizes are tried; when nothing fits there is no zone at all (no carpet
//     without people).
// ---------------------------------------------------------------------------

const (
	vsRect = iota
	vsRound
	vsOval
	vsL
	vsPlatform
)

const (
	vipSecLow = iota
	vipSecNormal
	vipSecHigh
)

var vipShapeNames = []string{"rect", "round", "oval", "L", "platform"}

// vipStyle: what the dice asked for. The zero value is the classic carpet (legacy layout).
type vipStyle struct {
	set    bool // false: classic
	shape  int
	cols   int
	rows   int
	flip   bool // L: the missing block on the right
	sec    int
	partyN int      // extra VIPs wanted (0 = a single VIP)
	extra  []string // their ids (filled by vipRoll once the main VIP is known)
}

func (s vipStyle) key() string {
	if !s.set {
		return "classic"
	}
	return fmt.Sprintf("%d/%d/%d/%v/%d/%d", s.shape, s.cols, s.rows, s.flip, s.sec, s.partyN)
}

// vipStyleDecision: the style of the zone of the level being loaded (vipRoll sets it, loadLevel clears it).
var vipStyleDecision vipStyle

var vipStyleRng *rand.Rand

func vipStyleDice() *rand.Rand {
	if vipStyleRng == nil {
		sd := vipSeed*7919 + 17
		if vipSeed == 0 {
			sd = time.Now().UnixNano() ^ 0x5eed
		}
		vipStyleRng = rand.New(rand.NewSource(sd))
	}
	return vipStyleRng
}

// dbgVIPShape: -vipshape rect|round|oval|L|platform|party forces the style of the zone (captures; with -vipdemo).
var dbgVIPShape = flag.String("vipshape", "", "VIP zone shape: rect | round | oval | L | platform | party (debug, with -vipdemo)")

// vipDebugStyle: the style asked by -vipshape (ok false: none).
func vipDebugStyle() (vipStyle, bool) {
	switch *dbgVIPShape {
	case "rect":
		return vipStyle{set: true, shape: vsRect, cols: 6, rows: 2, sec: vipSecHigh}, true
	case "round":
		return vipStyle{set: true, shape: vsRound, cols: 4, rows: 3, sec: vipSecNormal}, true
	case "oval":
		return vipStyle{set: true, shape: vsOval, cols: 6, rows: 3, sec: vipSecNormal}, true
	case "L":
		return vipStyle{set: true, shape: vsL, cols: 5, rows: 3, sec: vipSecNormal}, true
	case "platform":
		return vipStyle{set: true, shape: vsPlatform, cols: 5, rows: 2, sec: vipSecLow}, true
	case "party":
		return vipStyle{set: true, shape: vsOval, cols: 6, rows: 3, sec: vipSecHigh, partyN: 2}, true
	}
	return vipStyle{}, false
}

// vipRollStyle: shape, size, security and party, with weights.
func vipRollStyle(r *rand.Rand) vipStyle {
	st := vipStyle{set: true}
	switch w := r.Intn(100); {
	case w < 30:
		st.shape = vsRect
		st.cols, st.rows = 3+r.Intn(4), 1+r.Intn(2)
	case w < 48:
		st.shape = vsRound
		st.cols, st.rows = 3+r.Intn(2), 3
	case w < 63:
		st.shape = vsOval
		st.cols, st.rows = 5+r.Intn(2), 3
	case w < 80:
		st.shape = vsL
		st.cols, st.rows, st.flip = 4+r.Intn(2), 2+r.Intn(2), r.Intn(2) == 0
	default:
		st.shape = vsPlatform
		st.cols, st.rows = 4+r.Intn(3), 2
	}
	switch w := r.Intn(100); {
	case w < 25:
		st.sec = vipSecLow
	case w < 70:
		st.sec = vipSecNormal
	default:
		st.sec = vipSecHigh
	}
	if r.Float64() < 0.25 {
		st.partyN = 1 + r.Intn(2)
		st.sec = vipSecHigh
	}
	return st
}

// vipPickExtra: n more VIPs for a mini party, all different from main (missing ones first, like vipPick).
func vipPickExtra(r *rand.Rand, album map[string]int, main string, n int) []string {
	var left, rest []string
	for _, d := range vipRegistry() {
		if d.id == main {
			continue
		}
		if vipGot(album, d.id) {
			rest = append(rest, d.id)
		} else {
			left = append(left, d.id)
		}
	}
	r.Shuffle(len(left), func(i, j int) { left[i], left[j] = left[j], left[i] })
	r.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	all := append(left, rest...)
	if n > len(all) {
		n = len(all)
	}
	return append([]string(nil), all[:n]...)
}

// ---- cells ------------------------------------------------------------------------------------------------------

type vipCell struct{ c, r int }

// vipShapeCells: the occupied cells of a shape in a cols x rows grid.
func vipShapeCells(shape, cols, rows int, flip bool) []vipCell {
	var out []vipCell
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			corner := (c == 0 || c == cols-1) && (r == 0 || r == rows-1)
			switch shape {
			case vsRound, vsOval:
				if corner {
					continue
				}
			case vsL:
				k := cols / 2
				if r < rows-1 && ((!flip && c < k) || (flip && c >= cols-k)) {
					continue
				}
			}
			out = append(out, vipCell{c, r})
		}
	}
	return out
}

// vipTry: one shape to try on a level.
type vipTry struct {
	legacy     bool
	shape      int
	cols, rows int
	flip       bool
}

// vipTries: the preferred shape first, then smaller ones of the same kind, other shapes, the classic sizes last.
func vipTries(st vipStyle) []vipTry {
	legacy := []vipTry{{true, vsRect, 5, 2, false}, {true, vsRect, 4, 2, false}, {true, vsRect, 5, 1, false}, {true, vsRect, 4, 1, false}}
	if !st.set {
		return legacy
	}
	var out []vipTry
	seen := map[vipTry]bool{}
	add := func(t vipTry) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	add(vipTry{false, st.shape, st.cols, st.rows, st.flip})
	switch st.shape {
	case vsRect:
		for c := st.cols; c >= 3; c-- {
			for r := st.rows; r >= 1; r-- {
				add(vipTry{false, vsRect, c, r, false})
			}
		}
	case vsRound:
		add(vipTry{false, vsRound, 3, 3, false})
	case vsOval:
		add(vipTry{false, vsOval, 5, 3, false})
		add(vipTry{false, vsRound, 4, 3, false})
		add(vipTry{false, vsRound, 3, 3, false})
	case vsL:
		add(vipTry{false, vsL, st.cols, st.rows, !st.flip})
		add(vipTry{false, vsL, 4, 2, st.flip})
		add(vipTry{false, vsL, 4, 2, !st.flip})
	case vsPlatform:
		for c := st.cols; c >= 3; c-- {
			add(vipTry{false, vsPlatform, c, 2, false})
		}
		add(vipTry{false, vsPlatform, 4, 1, false})
	}
	// a different shape of the same feeling before the classic carpet
	add(vipTry{false, vsRound, 3, 3, false})
	add(vipTry{false, vsL, 4, 2, st.flip})
	add(vipTry{false, vsPlatform, 4, 2, false})
	add(vipTry{false, vsRect, 4, 2, false})
	add(vipTry{false, vsRect, 3, 2, false})
	add(vipTry{false, vsRect, 4, 1, false})
	add(vipTry{false, vsRect, 3, 1, false})
	for _, t := range legacy {
		add(t)
	}
	return out
}

// vipCellRects: the world rectangles of the cells of a zone whose bounding box starts at (x, y).
func vipCellRects(x, y float64, cells []vipCell) []Rect {
	out := make([]Rect, 0, len(cells))
	for _, c := range cells {
		out = append(out, Rect{x + vipSeg*float64(c.c), y + vipSeg*float64(c.r), vipSeg, vipSeg})
	}
	return out
}

// vipGateCell: the bottom cell nearest the middle (the gate opens in its south side).
func vipGateCell(cells []vipCell, cols, rows int) vipCell {
	best, bd := vipCell{-1, -1}, 1e9
	for _, c := range cells {
		if c.r != rows-1 {
			continue
		}
		if d := math.Abs(float64(c.c) + 0.5 - float64(cols)/2); d <= bd+1e-9 {
			best, bd = c, d
		}
	}
	return best
}

// vipCordonCells: the rope segments around the outline of the cells, a gap at the gate.
func vipCordonCells(x, y float64, cells []vipCell, gate vipCell) []vipPiece {
	has := map[vipCell]bool{}
	for _, c := range cells {
		has[c] = true
	}
	var out []vipPiece
	for _, c := range cells {
		cx, cy := x+vipSeg*float64(c.c), y+vipSeg*float64(c.r)
		if !has[vipCell{c.c, c.r - 1}] {
			out = append(out, vipPiece{"viprope_h", cx + vipSeg/2, cy})
		}
		if !has[vipCell{c.c, c.r + 1}] && c != gate {
			out = append(out, vipPiece{"viprope_h", cx + vipSeg/2, cy + vipSeg + 2})
		}
		// at a step of the outline the rope of the lower cell runs up to the corner of this side: the whole run of
		// posts of this side stands 4 px higher (pairClash measured: 4 is the least that clears both ropes)
		if !has[vipCell{c.c - 1, c.r}] {
			out = append(out, vipPiece{"viprope_v", cx - 2, cy + vipSeg - vipRunStep(has, c, -1)})
		}
		if !has[vipCell{c.c + 1, c.r}] {
			out = append(out, vipPiece{"viprope_v", cx + vipSeg + 2, cy + vipSeg - vipRunStep(has, c, 1)})
		}
	}
	return out
}

// vipRunStep: 4 when the run of open edges on side sd (-1 left, 1 right) that holds cell c ends in a step.
func vipRunStep(has map[vipCell]bool, c vipCell, sd int) float64 {
	r := c.r
	for has[vipCell{c.c, r + 1}] && !has[vipCell{c.c + sd, r + 1}] {
		r++
	}
	if has[vipCell{c.c + sd, r + 1}] {
		return 4
	}
	return 0
}

// vipShapeLayout: the positions inside a shaped zone. loot first, then the VIPs (at the back), then the inner guards
// (near the gate).
type vipLayout struct {
	loot   Vec // the gold suit prop
	vips   []Vec
	inners []Vec
}

func vipShapeLayout(x, y float64, cells []vipCell, gate vipCell, cols int, wantVIPs, wantInner int) vipLayout {
	type spot struct {
		c   vipCell
		pos Vec
	}
	var spots []spot
	for _, c := range cells {
		spots = append(spots, spot{c, Vec{x + vipSeg*float64(c.c) + 16, y + vipSeg*float64(c.r) + 22}})
	}
	gpos := Vec{x + vipSeg*(float64(gate.c)+0.5), y + vipSeg*float64(gate.r+1)}
	// loot: the cell nearest the middle of the top row(s)
	mid := Vec{x + vipSeg*float64(cols)/2, y}
	sort.SliceStable(spots, func(i, j int) bool { return dist(spots[i].pos, mid) < dist(spots[j].pos, mid) })
	var lay vipLayout
	lc := spots[0]
	lay.loot = Vec{lc.pos.X - 8, lc.pos.Y - 6}
	spots = spots[1:]
	// VIPs: the farthest from the gate (the back of the zone)
	sort.SliceStable(spots, func(i, j int) bool { return dist(spots[i].pos, gpos) > dist(spots[j].pos, gpos) })
	for len(lay.vips) < wantVIPs && len(spots) > 0 {
		lay.vips = append(lay.vips, spots[0].pos)
		spots = spots[1:]
	}
	// inner guards: nearest the gate, the gate cell itself last (the way in stays clear)
	sort.SliceStable(spots, func(i, j int) bool {
		gi, gj := spots[i].c == gate, spots[j].c == gate
		if gi != gj {
			return gj
		}
		return dist(spots[i].pos, gpos) < dist(spots[j].pos, gpos)
	})
	for len(lay.inners) < wantInner && len(spots) > 0 {
		lay.inners = append(lay.inners, spots[0].pos)
		spots = spots[1:]
	}
	return lay
}

// vipWantStaff: VIPs and inner guards a style asks for.
func vipWantStaff(st vipStyle) (vips, inner int) {
	vips = 1 + st.partyN
	switch st.sec {
	case vipSecLow:
		inner = 0
	case vipSecNormal:
		inner = 1
	default:
		inner = 2
	}
	if st.partyN > 0 {
		inner = maxI(2, vips)
	}
	return
}

// inside: a point in the cells of the zone (a little margin), or in its box for the classic carpet.
func (z *vipZone) inside(x, y float64) bool {
	if len(z.cells) == 0 {
		return z.trig.Has(x, y)
	}
	for _, c := range z.cells {
		if c.Grow(4).Has(x, y) {
			return true
		}
	}
	return false
}

// ---- drawing of the shaped floors ---------------------------------------------------------------------------------

// drawVIPShapeFloor: carpet of a shaped zone (the classic one keeps its own drawing in vip.go).
func (g *Game) drawVIPShapeFloor(dst *ebiten.Image, z *vipZone) {
	rr := func(r Rect, col color.RGBA) {
		fillRect(dst, g.wx(r.X), g.wy(r.Y), r.W*wk, r.H*wk, col)
	}
	dark, mid, light, gold := rgb(0x6a0c1a), rgb(0x9a1628), rgb(0xa81c30), rgb(0xe8c040)
	switch z.shape {
	case vsRound, vsOval:
		cx, cy := z.in.X+z.in.W/2, z.in.Y+z.in.H/2
		ell := func(rx, ry float64, col color.RGBA) {
			for y := -ry; y < ry; y++ {
				t := (y + 0.5) / ry
				hw := math.Round(rx * math.Sqrt(math.Max(0, 1-t*t)))
				if hw > 0 {
					rr(Rect{cx - hw, math.Floor(cy + y), 2 * hw, 1}, col)
				}
			}
		}
		rx, ry := z.in.W/2-2, z.in.H/2-2
		ell(rx, ry, dark)
		ell(rx-3, ry-3, gold)
		ell(rx-4, ry-4, light)
		ell(rx*0.45, ry*0.45, mid) // a darker medallion in the middle
		ell(rx*0.45-1, ry*0.45-1, light)
		return
	case vsPlatform:
		in := z.in
		rr(in, rgb(0x4c4c58))                                            // the dais
		rr(Rect{in.X + 2, in.Y + 2, in.W - 4, in.H - 10}, rgb(0x8e8e9c)) // its top
		rr(Rect{in.X + 2, in.Y + 2, in.W - 4, 1}, rgb(0xc8c8d4))
		rr(Rect{in.X + 2, in.Y + in.H - 8, in.W - 4, 6}, rgb(0x5c5c68)) // the front face
		rr(Rect{in.X + 6, in.Y + 6, in.W - 12, in.H - 18}, mid)         // the carpet on top
		rr(Rect{in.X + 7, in.Y + 7, in.W - 14, in.H - 20}, light)
		for _, e := range []Rect{{in.X + 6, in.Y + 6, in.W - 12, 1}, {in.X + 6, in.Y + in.H - 13, in.W - 12, 1}} {
			rr(e, gold)
		}
		for i := 0; i < 2; i++ { // steps at the gate
			rr(Rect{z.gx - 14 + float64(i)*2, in.Y + in.H - 8 + float64(i)*4, 28 - float64(i)*4, 4}, rgb(0x6e6e7a))
		}
		return
	}
	// rectangles and L: the union of the cells, a border only where the outline is
	for _, c := range z.cells {
		rr(c, light)
	}
	has := func(x, y float64) bool {
		for _, c := range z.cells {
			if c.Has(x, y) {
				return true
			}
		}
		return false
	}
	for _, c := range z.cells {
		if !has(c.X+c.W/2, c.Y-1) {
			rr(Rect{c.X, c.Y, c.W, 3}, dark)
			rr(Rect{c.X, c.Y + 5, c.W, 1}, gold)
		}
		if !has(c.X+c.W/2, c.Y+c.H+1) {
			rr(Rect{c.X, c.Y + c.H - 4, c.W, 1}, gold)
		}
		if !has(c.X-1, c.Y+c.H/2) {
			rr(Rect{c.X, c.Y, 3, c.H}, mid)
			rr(Rect{c.X + 3, c.Y, 1, c.H}, gold)
		}
		if !has(c.X+c.W+1, c.Y+c.H/2) {
			rr(Rect{c.X + c.W - 3, c.Y, 3, c.H}, mid)
			rr(Rect{c.X + c.W - 4, c.Y, 1, c.H}, gold)
		}
	}
}

// shapeName: for tests and logs.
func (z *vipZone) shapeName() string {
	if z.legacy {
		return "classic"
	}
	return vipShapeNames[z.shape]
}
