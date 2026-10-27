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
	Kind string
	X, Y float64
	Var  int
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
	Sky         string // city | sea | none
	Police      string // car | heli
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
}

type Prop struct {
	Name  string
	K     *PropKind
	X, Y  float64
	Var   int
	Foot  Rect
	armT  float64
	used  bool
	on    bool
	shake float64
	Sort  float64
	wreck bool
}

type Level struct {
	Def       *LevelDef
	W, H      float64
	Walls     []Wall
	Props     []*Prop
	Pools     []Rect
	solidsP   []Rect
	solidsN   []Rect
	sight     []Rect
	POIs      map[string][]Vec
	Lights    []Vec
	cw, ch    int
	blocked   []bool
	extraGrow float64
	bg        *ebiten.Image
}

const navCell = 8.0

func buildLevel(def *LevelDef) *Level {
	l := &Level{Def: def, W: def.W, H: def.H, Walls: def.Walls, Pools: def.Pools, POIs: map[string][]Vec{}}
	for _, w := range def.Walls {
		l.solidsP = append(l.solidsP, w.R)
		l.solidsN = append(l.solidsN, w.R)
		if w.Style != WallFence {
			l.sight = append(l.sight, w.R)
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
		l.Props = append(l.Props, p)
		if !k.NoSolid {
			l.solidsP = append(l.solidsP, p.Foot)
			l.solidsN = append(l.solidsN, p.Foot)
		}
		for _, u := range k.Use {
			l.POIs[k.UseKind] = append(l.POIs[k.UseKind], Vec{in.X + u.X, in.Y + u.Y})
		}
		if k.Light {
			l.Lights = append(l.Lights, Vec{in.X, in.Y - 44})
		}
	}
	// world bounds
	b := 60.0
	bounds := []Rect{{-b, -b, def.W + 2*b, b}, {-b, def.H, def.W + 2*b, b}, {-b, 0, b, def.H}, {def.W, 0, b, def.H}}
	l.solidsP = append(l.solidsP, bounds...)
	l.solidsN = append(l.solidsN, bounds...)
	l.initNav()
	return l
}

func (l *Level) initNav() {
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
	if l.cellFree(cx, cy) {
		return p
	}
	for r := 1; r < 40; r++ {
		best, bd := Vec{}, math.MaxFloat64
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}
				if l.cellFree(cx+dx, cy+dy) {
					c := cellCenter(cx+dx, cy+dy)
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

// LOS: only opaque walls block sight.
func (l *Level) LOS(a, b Vec) bool {
	n := int(dist(a, b)/4) + 1
	for i := 1; i < n; i++ {
		t := float64(i) / float64(n)
		x, y := a.X+(b.X-a.X)*t, a.Y+(b.Y-a.Y)*t
		for _, r := range l.sight {
			if r.Has(x, y) {
				return false
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

func (l *Level) RandomRoam(rng *rand.Rand) Vec {
	// weight by area
	total := 0.0
	for _, r := range l.Def.Roam {
		total += r.W * r.H
	}
	t := rng.Float64() * total
	for _, r := range l.Def.Roam {
		t -= r.W * r.H
		if t <= 0 {
			return l.RandomIn(rng, r)
		}
	}
	return l.RandomIn(rng, l.Def.Roam[0])
}

func (l *Level) InPool(p Vec) bool {
	for _, r := range l.Pools {
		if r.Has(p.X, p.Y) {
			return true
		}
	}
	return false
}
