package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

type Vec struct{ X, Y float64 }

func dist(a, b Vec) float64 { return math.Hypot(a.X-b.X, a.Y-b.Y) }

type Rect struct{ X, Y, W, H float64 }

func (r Rect) Has(x, y float64) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }
func (r Rect) Grow(d float64) Rect   { return Rect{r.X - d, r.Y - d, r.W + 2*d, r.H + 2*d} }
func (r Rect) Hits(o Rect) bool {
	return r.X < o.X+o.W && o.X < r.X+r.W && r.Y < o.Y+o.H && o.Y < r.Y+r.H
}
func (r Rect) Center() Vec { return Vec{r.X + r.W/2, r.Y + r.H/2} }

// distRect is the distance from a point to the closest point of a rectangle.
func distRect(p Vec, r Rect) float64 {
	dx := math.Max(math.Max(r.X-p.X, 0), p.X-(r.X+r.W))
	dy := math.Max(math.Max(r.Y-p.Y, 0), p.Y-(r.Y+r.H))
	return math.Hypot(dx, dy)
}

type Floor struct {
	R     Rect
	Kind  string
	Color uint32
}

// Wall styles
const (
	WallPlaster = iota
	WallBrick
	WallFence
	WallHedge
	WallClub
	WallRoof
	WallWood
	WallRail
)

type Wall struct {
	R     Rect
	Style int
}

type Door struct {
	R     Rect
	Horiz bool
	Style int
}

type PropInst struct {
	Kind  string
	X, Y  float64
	Var   int
	Scale float64 // 0 or 1 = normal size; the island draws its objects smaller (footprint, solids and use points follow)
}

type Perf struct {
	Pos  Vec
	Role string // dj, drum, guitar, sing
	Dir  Dir
}

type Glow struct {
	Pos   Vec
	Col   color.RGBA
	R     float64
	Pulse bool
}

type LevelDef struct {
	Title       string
	Icon        string // menu icon key, campaign levels only (see campaign.go)
	Sky         string // city | sea | none
	Police      string // car | heli
	Transit     string // "sea" or "space": vehicles arrive as boats or ships along CarFrom..CarTo instead of cars
	Pads        []Vec
	Performers  []Perf
	Doors       []Door
	Animals     []AnimalSpawn
	CellSize    float64
	Glows       []Glow
	Points      bool
	BGImg       *ebiten.Image
	Custom      bool
	Name, Intro string
	W, H        float64
	Floors      []Floor
	Walls       []Wall
	Props       []PropInst
	Pools       []Rect
	Swim        bool
	Roam        []Rect
	Dance       []Rect
	Start       Vec
	CarFrom     Vec
	CarTo       Vec
	Guests      int
	Guards      [][]Vec // patrol routes
	Tint        color.RGBA
	Disco       bool
	Strobe      []uint32
	FastTraffic bool      // circuit-style venues: ambient cars spawn much more often
	Highway     bool      // wide multi-lane road: dense traffic and people partying on the asphalt
	Roads       []RoadSeg // one-lane roads painted in the editor
	roadsDrawn  bool
	Tracks      []float64 // railway: the y of each track; trains run along them (see trains.go)
	TrainStop   float64   // x where a stopping train halts, centred
	Lanes       []float64 // highway: the y of every lane (above CarFrom.Y drives left, below drives right)
	Road        Rect      // highway: the asphalt band
	Hull        bool      // draw a ship hull band with portholes around HullRect
	HullRect    Rect
	Shape       func(x, y, grow float64) bool // inside test for ship / station outlines; everything outside is void
	HullName    string
	Day         bool    // no night darkness layer (daylight / dusk)
	LaneOff     float64 // distance of each traffic lane from the road centre line; 0 = default
	Traffic     string  // both | right | left
	ParkX0      float64 // police/ambulance stopping range along the road (0,0 = default)
	ParkX1      float64
	Slow        []Rect // walkable ground that slows the player
	Windows     []Rect // solid, but the player can jump through with E
	Sign        string // a neon wall sign (gradient text with a glow backing), e.g. "PARTY HARDY"
	SignPos     Vec
	PaintBG     func() *ebiten.Image // a background painted by code (built the first time the level is shown)
	RoadCar     string               // vehicle that drives the roads (default: ordinary cars)
	PropScale   float64              // props added to the level when it loads (phones, toilets, items) get this size too
	CharScale   float64              // 0 or 1 = normal; below 1 every person is drawn smaller and walks slower (the island)
}

type Prop struct {
	Name    string
	K       *PropKind
	X, Y    float64
	Var     int
	Foot    Rect
	armT    float64
	armed   bool    // a trap that finished loading: it waits like a mine (trapmine.go)
	by      *Player // who armed it
	armWait bool    // loaded but a player is still inside the radius: it becomes armed when nobody of the team is (trapmine.go)
	armedAt float64 // g.time when it became armed (blinking LED)
	used    bool
	on      bool
	techBy  *Tech // a technician is on his way to repair this cut phone
	shake   float64
	Sort    float64
	scale   float64 // see PropInst.Scale (0 = normal)
	wreck   bool
	fireOut bool    // flames put out by the firefighters
	burn    float64 // 0..1: how charred this prop is after a fire (fire.go)
	lit     bool    // it is burning right now
	gone    bool    // picked up / removed
	fade    float64 // 0..1: how see-through this tall prop is while someone stands behind it
}

type Level struct {
	Def       *LevelDef
	iceOn     int8 // 0 = not looked at yet, 1 = the level has ice floor, -1 = it has none (ice.go)
	W, H      float64
	Walls     []Wall
	Props     []*Prop
	LightSrc  []*PropKind // the kind behind each entry of Lights (its own light radius and colour)
	Pools     []Rect
	solidsP   []Rect
	solidsN   []Rect
	sight     []Rect
	POIs      map[string][]Vec
	Lights    []Vec
	cw, ch    int
	blocked   []bool
	blockedG  []bool // the navigation grid of the guests: the skate bowls are closed to them
	bowls     []Rect // the skate bowls: only the player, the kids, the police and the killers may step in
	extraGrow float64
	bg        *ebiten.Image
	graded    bool     // gradeBG applied (needs a running game, so it happens on first draw)
	vip       *vipZone // the VIP zone of this party, if the plan has one (vip.go)
}

const navCell = 8.0

func buildLevelRaw(def *LevelDef) *Level {
	l := &Level{Def: def, W: def.W, H: def.H, Walls: def.Walls, Pools: def.Pools, POIs: map[string][]Vec{}}
	for _, w := range def.Walls {
		wr := wallFoot(w) // house walls are a touch thicker than their design rect (walls_look.go)
		l.solidsP = append(l.solidsP, wr)
		l.solidsN = append(l.solidsN, wr)
		if w.Style != WallFence && w.Style != WallBoards {
			l.sight = append(l.sight, wr)
		}
	}
	for _, in := range def.Props {
		k := propKinds[in.Kind]
		if k == nil {
			panic("unknown prop kind " + in.Kind)
		}
		p := &Prop{Name: in.Kind, K: k, X: in.X, Y: in.Y, Var: in.Var, Foot: Rect{in.X - k.FW/2, in.Y - k.FH, k.FW, k.FH}, Sort: in.Y + k.SortOff}
		if k.hasFoot {
			p.Foot = Rect{in.X + k.FootDX, in.Y + k.FootDY, k.FW, k.FH}
		}
		sc := 1.0
		if in.Scale > 0 && in.Scale != 1 { // a smaller object: its footprint, solids and use points shrink around its anchor
			sc, p.scale = in.Scale, in.Scale
			p.Foot = Rect{in.X + (p.Foot.X-in.X)*sc, in.Y + (p.Foot.Y-in.Y)*sc, p.Foot.W * sc, p.Foot.H * sc}
		}
		l.Props = append(l.Props, p)
		if !k.NoSolid {
			l.solidsP = append(l.solidsP, p.Foot)
			l.solidsN = append(l.solidsN, p.Foot)
		}
		for _, r := range k.Solids {
			sr := Rect{in.X + r.X*sc, in.Y + r.Y*sc, r.W * sc, r.H * sc}
			l.solidsP = append(l.solidsP, sr)
			l.solidsN = append(l.solidsN, sr)
		}
		for _, u := range k.Use {
			l.POIs[k.UseKind] = append(l.POIs[k.UseKind], Vec{in.X + u.X*sc, in.Y + u.Y*sc})
		}
		if k.Light {
			l.Lights = append(l.Lights, Vec{in.X, in.Y - 44})
			l.LightSrc = append(l.LightSrc, k)
		}
	}
	// world bounds
	b := 60.0
	bounds := []Rect{{-b, -b, def.W + 2*b, b}, {-b, def.H, def.W + 2*b, b}, {-b, 0, b, def.H}, {def.W, 0, b, def.H}}
	l.solidsP = append(l.solidsP, bounds...)
	l.solidsN = append(l.solidsN, bounds...)
	l.initNav()
	if l.ensurePhones(2) {
		l.initNav()
	}
	if l.ensureToilets(2) {
		l.initNav()
	}
	if l.ensureItems() {
		l.initNav()
	}
	if l.ensureVIP() { // VIP zone and gold suits (vip.go)
		l.initNav()
	}
	return l
}

// ensurePhones makes sure every party has at least n phone booths, spread apart
// on walkable, reachable floor (not on dance floors or in pools).
func (l *Level) ensurePhones(n int) bool {
	a := l.ensureBooth("phone", n, func(p *Prop) bool { return p.K.Phone }, nil)
	if a {
		l.initNav()
	}
	// the yellow technicians' phone is one more booth, apart from the red ones
	var reds []Vec
	for _, p := range l.Props {
		if p.K.Phone {
			reds = append(reds, Vec{p.X, p.Y})
		}
	}
	b := l.ensureBooth("techphone", 1, func(p *Prop) bool { return p.K.TechPhone }, reds)
	return a || b
}

// ensureBooth places props of one kind until the level has n of them (match counts the
// existing ones); seeds are extra spots to keep away from.
func (l *Level) ensureBooth(name string, n int, match func(p *Prop) bool, seeds []Vec, margin ...float64) bool {
	mg := 0.0
	if len(margin) > 0 {
		mg = margin[0]
	}
	k := propKinds[name]
	if k == nil {
		return false
	}
	fitAllFootprints()
	var have []Vec
	for _, p := range l.Props {
		if match(p) {
			have = append(have, Vec{p.X, p.Y})
		}
	}
	near := append(append([]Vec(nil), have...), seeds...)
	added := false
	reach := l.reachFrom(l.Def.Start)
	spots := l.Def.Roam
	for len(have) < n {
		var best Vec
		bd := -1.0
		search := func(mg float64) {
			try := func(p Vec) {
				for _, o := range []Vec{{0, 0}, {16, 0}, {-16, 0}, {0, -14}, {0, 10}, {14, 10}, {-14, 10}} {
					if !l.Free(Vec{p.X + o.X, p.Y + o.Y}) {
						return
					}
				}
				if mg > 0 {
					for _, o := range []Vec{{mg, 0}, {-mg, 0}, {0, mg}, {0, -mg}, {mg, mg}, {-mg, mg}, {mg, -mg}, {-mg, -mg}} {
						if !l.Free(Vec{p.X + o.X, p.Y + o.Y}) {
							return
						}
					}
				}
				if l.badGround(p) || l.InPool(p) || p.X < 40 || p.X > l.W-40 || p.Y < 70 || p.Y > l.H-30 {
					return
				}
				for _, d := range l.Def.Dance {
					if d.Grow(14).Has(p.X, p.Y) {
						return
					}
				}
				if !reach[int(p.Y/navCell)*l.cw+int(p.X/navCell)] {
					return
				}
				if l.propClashes(name, k, p, 0) || l.hidesStage(k, p) {
					return
				}
				md := 1e9
				for _, h := range near {
					md = math.Min(md, dist(h, p))
				}
				if len(near) == 0 {
					md = dist(p, l.Def.Start)
				}
				if md < 26 { // never pile two things on the same spot
					return
				}
				if md > bd {
					best, bd = p, md
				}
			}
			if len(spots) > 0 {
				for _, r := range spots {
					for y := r.Y + 10; y < r.Y+r.H; y += 28 {
						for x := r.X + 10; x < r.X+r.W; x += 28 {
							try(Vec{x, y})
						}
					}
				}
			}
			if bd < 0 {
				for y := 80.0; y < l.H-30; y += 30 {
					for x := 50.0; x < l.W-50; x += 30 {
						try(Vec{x, y})
					}
				}
			}
		}
		search(mg)
		if bd < 0 {
			break
		}
		p := &Prop{Name: name, K: k, X: best.X, Y: best.Y, Foot: Rect{best.X - k.FW/2, best.Y - k.FH, k.FW, k.FH}, Sort: best.Y + k.SortOff}
		if k.hasFoot {
			p.Foot = Rect{best.X + k.FootDX, best.Y + k.FootDY, k.FW, k.FH}
		}
		if sc := l.Def.PropScale; sc > 0 && sc != 1 { // same size as the rest of the level
			p.scale = sc
			p.Foot = Rect{best.X + (p.Foot.X-best.X)*sc, best.Y + (p.Foot.Y-best.Y)*sc, p.Foot.W * sc, p.Foot.H * sc}
		}
		l.Props = append(l.Props, p)
		if !k.NoSolid { // pick-up items are not solid: a solid would stay behind, invisible, once they are taken
			l.solidsP = append(l.solidsP, p.Foot)
			l.solidsN = append(l.solidsN, p.Foot)
		}
		have = append(have, best)
		near = append(near, best)
		added = true
	}
	return added
}

// propClashes: would a prop of this kind standing at p stand on something already in the level?
// The solid grid is not rebuilt between two placements, so the props themselves are compared.
func (l *Level) propClashes(name string, k *PropKind, p Vec, skip int) bool {
	in := PropInst{Kind: name, X: p.X, Y: p.Y}
	foot, base := instFoot(in, k), instBase(in, k)
	for _, o := range l.Props {
		if pairClash(name, k, foot, base, o.Name, o.K, o.Foot, o.base()) {
			return true
		}
	}
	for _, w := range l.Walls {
		if w.Style != WallVoid && wallClash(name, k, foot, base, w) {
			return true
		}
	}
	return groundClash(l.Def, name, k, foot, p) != ""
}

// reachFrom flood-fills the navigation grid from a point.
func (l *Level) reachFrom(from Vec) []bool {
	seen := make([]bool, l.cw*l.ch)
	sx, sy := int(from.X/navCell), int(from.Y/navCell)
	if sx < 0 || sy < 0 || sx >= l.cw || sy >= l.ch {
		return seen
	}
	q := []int{sy*l.cw + sx}
	seen[q[0]] = true
	for len(q) > 0 {
		c := q[0]
		q = q[1:]
		cx, cy := c%l.cw, c/l.cw
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := cx+d[0], cy+d[1]
			if nx < 0 || ny < 0 || nx >= l.cw || ny >= l.ch {
				continue
			}
			i := ny*l.cw + nx
			if !seen[i] && !l.blocked[i] {
				seen[i] = true
				q = append(q, i)
			}
		}
	}
	return seen
}

func (l *Level) initNav() {
	defer l.initBowlNav()
	l.cw = int(math.Ceil(l.W / navCell))
	l.ch = int(math.Ceil(l.H / navCell))
	l.blocked = make([]bool, l.cw*l.ch)
	grown := make([]Rect, len(l.solidsN))
	for i, r := range l.solidsN {
		grown[i] = r.Grow(4 + l.extraGrow)
	}
	for cy := 0; cy < l.ch; cy++ {
		for cx := 0; cx < l.cw; cx++ {
			px, py := (float64(cx)+0.5)*navCell, (float64(cy)+0.5)*navCell
			for _, r := range grown {
				if r.Has(px, py) {
					l.blocked[cy*l.cw+cx] = true
					break
				}
			}
		}
	}
}

// SolidP: does a player-sized foot box at (x,y) collide?
func (l *Level) SolidP(x, y float64) bool {
	box := Rect{x - 4, y - 3, 8, 5}
	for _, r := range l.solidsP {
		if box.Hits(r) {
			return true
		}
	}
	return false
}

func (l *Level) cellOf(p Vec) (int, int) {
	return int(math.Floor(p.X / navCell)), int(math.Floor(p.Y / navCell))
}

func (l *Level) cellFree(cx, cy int) bool {
	return cx >= 0 && cy >= 0 && cx < l.cw && cy < l.ch && !l.blocked[cy*l.cw+cx]
}

func (l *Level) Free(p Vec) bool {
	cx, cy := l.cellOf(p)
	return l.cellFree(cx, cy)
}

func cellCenter(cx, cy int) Vec {
	return Vec{(float64(cx) + 0.5) * navCell, (float64(cy) + 0.5) * navCell}
}

func (l *Level) NearestFree(p Vec) Vec {
	cx, cy := l.cellOf(p)
	// the spot itself, but only when the player's collision box fits there too: a "free" cell can
	// still have a wall or a prop foot inside it, and standing in one means being stuck
	if l.cellFree(cx, cy) && !l.SolidP(p.X, p.Y) {
		return p
	}
	for r := 0; r < 40; r++ {
		best, bd := Vec{}, math.MaxFloat64
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}
				if l.cellFree(cx+dx, cy+dy) {
					c := cellCenter(cx+dx, cy+dy)
					if l.SolidP(c.X, c.Y) {
						continue
					}
					if d := dist(c, p); d < bd {
						best, bd = c, d
					}
				}
			}
		}
		if bd < math.MaxFloat64 {
			return best
		}
	}
	return p
}

func (l *Level) clearLine(a, b Vec) bool {
	d := dist(a, b)
	n := int(d/3) + 1
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		if !l.Free(Vec{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}) {
			return false
		}
	}
	return true
}

// Path returns waypoints (excluding the start) from -> to, smoothed.
func (l *Level) Path(from, to Vec) []Vec {
	s := l.NearestFree(from)
	e := l.NearestFree(to)
	sx, sy := l.cellOf(s)
	ex, ey := l.cellOf(e)
	if sx == ex && sy == ey {
		return []Vec{e}
	}
	prev := make([]int32, l.cw*l.ch)
	for i := range prev {
		prev[i] = -2
	}
	start := sy*l.cw + sx
	goal := ey*l.cw + ex
	prev[start] = -1
	queue := []int{start}
	dirs := [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}}
	found := false
	for h := 0; h < len(queue) && !found; h++ {
		cur := queue[h]
		cx, cy := cur%l.cw, cur/l.cw
		for _, d := range dirs {
			nx, ny := cx+d[0], cy+d[1]
			if !l.cellFree(nx, ny) {
				continue
			}
			if d[0] != 0 && d[1] != 0 && (!l.cellFree(cx+d[0], cy) || !l.cellFree(cx, cy+d[1])) {
				continue
			}
			ni := ny*l.cw + nx
			if prev[ni] != -2 {
				continue
			}
			prev[ni] = int32(cur)
			if ni == goal {
				found = true
				break
			}
			queue = append(queue, ni)
		}
	}
	if !found {
		return nil
	}
	var cells []Vec
	for i := goal; i != -1; i = int(prev[i]) {
		cells = append(cells, cellCenter(i%l.cw, i/l.cw))
	}
	for i, j := 0, len(cells)-1; i < j; i, j = i+1, j-1 {
		cells[i], cells[j] = cells[j], cells[i]
	}
	// string pulling
	var out []Vec
	cur := from
	i := 0
	for i < len(cells) {
		j := i
		for k := i + 1; k < len(cells) && k < i+14; k++ {
			if l.clearLine(cur, cells[k]) {
				j = k
			}
		}
		out = append(out, cells[j])
		cur = cells[j]
		i = j + 1
	}
	if len(out) > 0 && l.Free(to) {
		out[len(out)-1] = to
	}
	return out
}

// LOS: only opaque walls block sight (exact segment-vs-rectangle test).
func (l *Level) LOS(a, b Vec) bool {
	for _, r := range l.sight {
		if segHitsRect(a, b, r) {
			return false
		}
	}
	return true
}

func segHitsRect(a, b Vec, r Rect) bool {
	t0, t1 := 0.0, 1.0
	dx, dy := b.X-a.X, b.Y-a.Y
	for _, e := range [4][2]float64{{-dx, a.X - r.X}, {dx, r.X + r.W - a.X}, {-dy, a.Y - r.Y}, {dy, r.Y + r.H - a.Y}} {
		p, q := e[0], e[1]
		if p == 0 {
			if q < 0 {
				return false
			}
			continue
		}
		t := q / p
		if p < 0 {
			if t > t1 {
				return false
			}
			if t > t0 {
				t0 = t
			}
		} else {
			if t < t0 {
				return false
			}
			if t < t1 {
				t1 = t
			}
		}
	}
	return true
}

func (l *Level) RandomIn(rng *rand.Rand, r Rect) Vec {
	for i := 0; i < 30; i++ {
		p := Vec{r.X + rng.Float64()*r.W, r.Y + rng.Float64()*r.H}
		if l.Free(p) {
			return p
		}
	}
	return l.NearestFree(r.Center())
}

// roamTopY: no place to wander to (nor to appear at) has its feet higher than this: a guest walking along the very top
// of the screen is hardly seen (his head is cut by the border). The roaming areas are cut there (v1.4.0); a level whose
// areas are all above it keeps them as they are.
const roamTopY = 64.0

// roamRects: the roaming areas with the top band cut off.
func (l *Level) roamRects() []Rect {
	var out []Rect
	for _, r := range l.Def.Roam {
		if r.Y < roamTopY {
			r.H -= roamTopY - r.Y
			r.Y = roamTopY
		}
		if r.H >= 4 && r.W > 0 {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return l.Def.Roam
	}
	return out
}

func (l *Level) RandomRoam(rng *rand.Rand) Vec {
	rects := l.roamRects()
	// weight by area
	total := 0.0
	for _, r := range rects {
		total += r.W * r.H
	}
	t := rng.Float64() * total
	for _, r := range rects {
		t -= r.W * r.H
		if t <= 0 {
			return l.RandomIn(rng, r)
		}
	}
	return l.RandomIn(rng, rects[0])
}

func (l *Level) InPool(p Vec) bool {
	for _, r := range l.Pools {
		if r.Has(p.X, p.Y) {
			return true
		}
	}
	return false
}

// Outdoor says whether the topmost floor under p is open ground rather than
// the inside of a building.
func (l *Level) Outdoor(p Vec) bool {
	kind := ""
	for _, f := range l.Def.Floors {
		if f.R.Has(p.X, p.Y) {
			kind = f.Kind
		}
	}
	switch kind {
	case "grass", "dirt", "sand", "cobble", "pave", "asphalt", "dash", "pitch", "arena", "boardwalk", "deck", "rooftop", "rooftop2", "snow", "chalk", "edge", "rails", "galleon", "alcala", "guadalajara", "barcelona", "paris", "":
		return true
	}
	return false
}

// badGround: roads, helipads and the like, where nothing may be planted.
func (l *Level) badGround(p Vec) bool {
	d := l.Def
	if d.Highway && d.Road.Has(p.X, p.Y) {
		return true
	}
	for _, f := range d.Floors {
		if spotBadFloors[f.Kind] && f.R.Grow(6).Has(p.X, p.Y) {
			return true
		}
	}
	for _, pd := range d.Pads {
		if math.Hypot(pd.X-p.X, pd.Y-p.Y) < 60 {
			return true
		}
	}
	for _, rs := range d.Roads { // nothing is planted on the roads of the level
		if rs.R.Grow(8).Has(p.X, p.Y) {
			return true
		}
	}
	return false
}

// sc is the size factor of the prop (1 = normal).
func (p *Prop) sc() float64 {
	if p.scale > 0 {
		return p.scale
	}
	return 1
}

// RoadSeg is a one-lane road painted in the editor: traffic drives through the rectangle in Dir.
type RoadSeg struct {
	R   Rect
	Dir int // roadRight, roadLeft, roadUp, roadDown
}

const (
	roadRight = iota
	roadLeft
	roadUp
	roadDown
)

func (r RoadSeg) horizontal() bool { return r.Dir == roadRight || r.Dir == roadLeft }
