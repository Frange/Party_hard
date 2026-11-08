package main

import (
	"fmt"
	"math"
	"sort"
)

// ---------------------------------------------------------------------------
// Objects must not stand on top of each other. This file defines what "standing
// on" means (footprints, plus the opaque base of the drawn sprite), finds every
// overlap in a level and, for the levels the game generates itself, resolves
// them: the object with less priority is moved to the nearest free spot or,
// failing that, removed (decoration) — see resolveOverlaps.
// ---------------------------------------------------------------------------

// overlayKinds are layered on purpose: they hang over the scenery (arches, bridges, ropes, banners),
// are drawn flat on the ground, or are animated scenery of the island that never blocks anything.
var overlayKinds = map[string]bool{
	"arch": true, "bridge": true, "rope": true, "truss": true, "slide": true, "posters": true, "sign": true,
	"lifebuoy": true, "jp_gate": true, "jp_trex": true, "jp_brachio": true, "jp_raptor": true, "jp_trike": true,
	"jp_fin": true, "jp_boat": true, "hen": true, "weapon": true,
}

// stageHosts may stand on a stage (the performers' platform).
var stageHosts = map[string]bool{"dj": true, "speaker": true, "screen": true, "truss": true, "lamp": true, "plant": true, "jukebox": true}

// waterOK: things that stand in the water on purpose.
var waterOK = map[string]bool{"jp_hadro": true, "jp_boat": true, "jp_fin": true, "jp_brachio": true}

// droppable: big scatter pieces that may disappear when there is no room for them anywhere near.
var droppable = map[string]bool{"booth": true, "bungalow": true, "tent": true, "igloo": true, "lifeguard": true, "carousel": true, "chalet": true}

// roadOK: kinds that may stand on a road or asphalt (vehicles, things that belong to the traffic).
var roadOK = map[string]bool{"car": true, "bridge": true, "arch": true, "jp_jeep": true, "lamp": true, "sign": true, "rope": true, "truss": true}

// priorities: who keeps its place when two things collide (bigger = stays).
func propPriority(name string) int {
	switch name {
	case "stage", "jp_gate", "booth", "bungalow", "stands", "eiffel", "ferris", "carousel", "coaster", "chalet", "crypt", "goal", "goalv",
		"screen", "halfpipe", "skatebowl", "lift", "igloo", "jp_lab", "jp_rotunda", "jp_tower", "jp_control", "jp_lighthouse", "fountain",
		"lifeguard", "liberty", "obelisk", "tent":
		return 4
	case "phone", "techphone", "toilet", "portapotty", "dj", "bar", "punch", "buffet", "bowl", "checkout", "jukebox", "stall",
		"poisonbottle", "disguisebag":
		return 3
	case "tree", "palm", "pine", "deadtree", "bush", "hedge", "plant", "planter", "lamp", "umbrella", "pumpkin", "tombstone",
		"snowman", "balloons", "barrel", "crate", "surfboard", "stool", "cart", "extinguisher", "fusebox", "gastank", "fireworksbox", "cottoncandy", "popcorn":
		return 1
	}
	return 2
}

// propBaseRel returns the ground a prop visibly covers, relative to its anchor: the bounding box of the
// opaque pixels of the lower part of the sprite (from a little above its footprint down), joined with the footprint.
type baseKey struct {
	k *PropKind
	v int
}

var baseCache = map[baseKey]Rect{}

func propFootRel(k *PropKind) Rect {
	if k.hasFoot {
		return Rect{k.FootDX, k.FootDY, k.FW, k.FH}
	}
	return Rect{-k.FW / 2, -k.FH, k.FW, k.FH}
}

func propBaseRel(name string, k *PropKind, v int) Rect {
	key := baseKey{k, v}
	if r, ok := baseCache[key]; ok {
		return r
	}
	foot := propFootRel(k)
	r := foot
	var b *pixBuf
	var ax, ay float64 // anchor inside the sprite
	if k.Native != nil {
		b = k.Native[v%len(k.Native)]
		ax, ay = k.NAX, k.NAY
	} else if k.Draw != nil {
		b = newPix(k.W+2, k.H+2)
		b.ox, b.oy = 1, 1
		k.Draw(b, v)
		ax, ay = float64(k.W)/2+1, float64(k.H)+1
	}
	if b != nil {
		top := foot.Y - 3
		x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for y := 0; y < b.h; y++ {
			ry := float64(y) - ay
			if ry < top {
				continue
			}
			for x := 0; x < b.w; x++ {
				if b.p[(y*b.w+x)*4+3] < 128 {
					continue
				}
				rx := float64(x) - ax
				x0, x1 = math.Min(x0, rx), math.Max(x1, rx+1)
				y0, y1 = math.Min(y0, ry), math.Max(y1, ry+1)
			}
		}
		if x1 > x0 {
			// join with the footprint
			x0, y0 = math.Min(x0, foot.X), math.Min(y0, foot.Y)
			x1, y1 = math.Max(x1, foot.X+foot.W), math.Max(y1, foot.Y+foot.H)
			r = Rect{x0, y0, x1 - x0, y1 - y0}
		}
	}
	baseCache[key] = r
	return r
}

func (p *Prop) base() Rect {
	r := propBaseRel(p.Name, p.K, p.Var)
	return Rect{p.X + r.X, p.Y + r.Y, r.W, r.H}
}

// instBase / instFoot give the same for a prop instance that is not built into a Level yet.
func instFoot(in PropInst, k *PropKind) Rect {
	r := propFootRel(k)
	return Rect{in.X + r.X, in.Y + r.Y, r.W, r.H}
}

func instBase(in PropInst, k *PropKind) Rect {
	r := propBaseRel(in.Kind, k, in.Var)
	return Rect{in.X + r.X, in.Y + r.Y, r.W, r.H}
}

func rectOv(a, b Rect) (float64, float64) {
	return math.Min(a.X+a.W, b.X+b.W) - math.Max(a.X, b.X), math.Min(a.Y+a.H, b.Y+b.H) - math.Max(a.Y, b.Y)
}

const (
	footTol = 1.5 // footprints may touch; more than this is standing on top
	baseTol = 3.0 // the drawn bases may rub; more than this looks like one on top of the other
	wallTol = 4.0 // furniture may lean on a wall this much
)

// pairClash: do two props stand on each other?
// exclusion: the area in front of and between the pillars of the park gate must stay clear (the road runs through it).
func gateZone(foot Rect) Rect {
	cx, cy := foot.X+foot.W/2, foot.Y+foot.H // the anchor of the gate
	return Rect{cx - 95, cy - 104, 190, 104 + 40}
}

func pairClash(an string, ak *PropKind, af, ab Rect, bn string, bk *PropKind, bf, bb Rect) bool {
	if an == "jp_gate" || bn == "jp_gate" {
		gz, of, ob, on := gateZone(af), bf, bb, bn
		if bn == "jp_gate" {
			gz, of, ob, on = gateZone(bf), af, ab, an
		}
		if on == "jp_gate" || waterOK[on] || on == "jp_jeep" && false {
			return false
		}
		if ox, oy := rectOv(of, gz); ox > 1 && oy > 1 {
			return true
		}
		if ox, oy := rectOv(ob, gz); ox > 3 && oy > 3 {
			return true
		}
		return false
	}
	hit := false
	if ox, oy := rectOv(af, bf); ox > footTol && oy > footTol {
		hit = true
	} else if ox, oy := rectOv(ab, bb); ox > baseTol && oy > baseTol {
		hit = true
	}
	if !hit {
		return false
	}
	if overlayKinds[an] || overlayKinds[bn] {
		return false
	}
	if an == "stage" && stageHosts[bn] || bn == "stage" && stageHosts[an] {
		return false
	}
	return true
}

// wallClash: does a prop stand inside a wall? Walls are structure: furniture may only lean on them.
func wallClash(name string, k *PropKind, foot, base Rect, w Wall) bool {
	tol := wallTol
	if w.Style == WallFence || w.Style == WallVoid {
		tol = 2
	}
	if w.Style == WallVoid && waterOK[name] {
		return false
	}
	hit := false
	if ox, oy := rectOv(foot, w.R); ox > tol && oy > tol {
		hit = true
	} else if w.Style == WallFence || w.Style == WallVoid {
		if ox, oy := rectOv(base, w.R); ox > baseTol && oy > baseTol {
			hit = true
		}
	}
	return hit && !wallHuggers[name] && !overlayKinds[name]
}

// groundClash: pools, roads, dance floors, helipads and the points the game needs clear.
func groundClash(d *LevelDef, name string, k *PropKind, foot Rect, at Vec) string {
	what := groundHit(d, name, foot, at)
	if what == "" || overlayKinds[name] {
		return ""
	}
	switch what {
	case "dance", "performer":
		if name == "dj" || name == "stage" || stageHosts[name] {
			return ""
		}
	case "road", "floor":
		if roadOK[name] {
			return ""
		}
	}
	return what
}

// offGround: is the foot of a prop outside the ground of the party (sea, space, outside the outline of a ship,
// island or station, or on a "water" floor)? Levels without an outline count as ground what any floor
// (except water) or any roaming area covers.
func offGround(d *LevelDef, foot Rect) bool {
	if d.Shape != nil {
		pts := [5]Vec{foot.Center(), {foot.X + 1, foot.Y + 1}, {foot.X + foot.W - 1, foot.Y + 1}, {foot.X + 1, foot.Y + foot.H - 1}, {foot.X + foot.W - 1, foot.Y + foot.H - 1}}
		for _, p := range pts {
			if !d.Shape(p.X, p.Y, -2) {
				return true
			}
		}
	}
	c := foot.Center()
	for _, f := range d.Floors {
		if (f.Kind == "water" || f.Kind == "lava") && f.R.Has(c.X, c.Y) {
			return true
		}
	}
	if d.Shape != nil || len(d.Floors) == 0 {
		return false
	}
	for _, f := range d.Floors {
		if f.Kind != "water" && f.Kind != "lava" && f.R.Grow(2).Has(c.X, c.Y) {
			return false
		}
	}
	for _, r := range d.Roam {
		if r.Grow(24).Has(c.X, c.Y) {
			return false
		}
	}
	return true
}

func groundHit(d *LevelDef, name string, foot Rect, at Vec) string {
	for _, p := range d.Pools {
		if ox, oy := rectOv(foot, p); ox > 2 && oy > 2 && !waterOK[name] {
			return "pool"
		}
	}
	if !waterOK[name] && offGround(d, foot) {
		return "offground"
	}
	for _, r := range d.Roads {
		if ox, oy := rectOv(foot, r.R); ox > 1 && oy > 1 {
			return "road"
		}
	}
	if d.Highway && d.Road.W > 0 {
		if ox, oy := rectOv(foot, d.Road); ox > 1 && oy > 1 {
			return "road"
		}
	}
	for _, f := range d.Floors {
		if spotBadFloors[f.Kind] && f.Kind != "pitch" && f.Kind != "arena" {
			if ox, oy := rectOv(foot, f.R); ox > 1 && oy > 1 {
				return "floor"
			}
		}
	}
	for _, r := range d.Dance {
		if ox, oy := rectOv(foot, r); ox > 2 && oy > 2 {
			return "dance"
		}
	}
	for _, p := range d.Pads {
		if math.Hypot(p.X-at.X, p.Y-at.Y) < 34 {
			return "pad"
		}
	}
	if foot.Grow(3).Has(d.Start.X, d.Start.Y) {
		return "start"
	}
	for _, pf := range d.Performers {
		if foot.Grow(2).Has(pf.Pos.X, pf.Pos.Y) {
			return "performer"
		}
	}
	return ""
}

// overlapIssue is one thing standing where it should not.
type overlapIssue struct {
	What string // props | wall | pool | road | floor:x | dance | pad | start | performer
	A, B string
	X, Y float64
}

func (o overlapIssue) String() string {
	return fmt.Sprintf("%s: %s / %s at (%.0f,%.0f)", o.What, o.A, o.B, o.X, o.Y)
}

// levelOverlaps lists every overlap of a built level (everything the game puts in it is there:
// the generator's props plus the phones, toilets and items added when it loads).
func levelOverlaps(l *Level) []overlapIssue {
	fitAllFootprints()
	var out []overlapIssue
	for i, a := range l.Props {
		af, ab := a.Foot, a.base()
		for j := i + 1; j < len(l.Props); j++ {
			b := l.Props[j]
			if pairClash(a.Name, a.K, af, ab, b.Name, b.K, b.Foot, b.base()) {
				out = append(out, overlapIssue{"props", a.Name, b.Name, (a.X + b.X) / 2, (a.Y + b.Y) / 2})
			}
		}
		for _, w := range l.Walls {
			if wallClash(a.Name, a.K, af, ab, w) {
				out = append(out, overlapIssue{"wall", a.Name, fmt.Sprintf("wall%d", w.Style), a.X, a.Y})
				break
			}
		}
		if what := groundClash(l.Def, a.Name, a.K, af, Vec{a.X, a.Y}); what != "" {
			out = append(out, overlapIssue{what, a.Name, "", a.X, a.Y})
		}
	}
	return out
}

// ---- resolution (generated levels only) -------------------------------------------------

type geoProp struct {
	in   PropInst
	k    *PropKind
	foot Rect
	base Rect
}

func mkGeo(in PropInst) *geoProp {
	k := propKinds[in.Kind]
	if k == nil {
		return nil
	}
	return &geoProp{in: in, k: k, foot: instFoot(in, k), base: instBase(in, k)}
}

// clashes reports whether g collides with anything already in the level (def walls / grounds / the others).
func (g *geoProp) clashes(d *LevelDef, others []*geoProp, skip *geoProp) bool {
	for _, o := range others {
		if o == g || o == skip {
			continue
		}
		if pairClash(g.in.Kind, g.k, g.foot, g.base, o.in.Kind, o.k, o.foot, o.base) {
			return true
		}
	}
	for _, w := range d.Walls {
		if wallClash(g.in.Kind, g.k, g.foot, g.base, w) {
			return true
		}
	}
	return groundClash(d, g.in.Kind, g.k, g.foot, Vec{g.in.X, g.in.Y}) != ""
}

// resolveOverlaps moves or removes the props of a generated level that stand on something.
// Props that were moved stay inside the level and on walkable ground; the ones that cannot be
// placed anywhere nearby are dropped when they are decoration (priority 1..2) and kept otherwise
// (the test reports them: the generator has to be fixed).
// skipResolve lets the tests look at the raw output of the generators.
var skipResolve bool

func resolveOverlaps(d *LevelDef) {
	if skipResolve {
		return
	}
	fitAllFootprints()
	var geos []*geoProp
	for _, in := range d.Props {
		if g := mkGeo(in); g != nil {
			geos = append(geos, g)
		}
	}
	// the most important things are placed first, so that the rest gives way to them
	order := append([]*geoProp(nil), geos...)
	sort.SliceStable(order, func(i, j int) bool { return propPriority(order[i].in.Kind) > propPriority(order[j].in.Kind) })
	var placed []*geoProp
	var out []PropInst
	for _, g := range order {
		if !g.clashes(d, placed, nil) {
			placed = append(placed, g)
			continue
		}
		if np, ok := findFreeSpot(d, g, placed); ok {
			g.in.X, g.in.Y = np.X, np.Y
			g.foot, g.base = instFoot(g.in, g.k), instBase(g.in, g.k)
			placed = append(placed, g)
			continue
		}
		spare := false // an extra bar with no room anywhere is dropped when another bar is already placed
		if g.in.Kind == "bar" {
			for _, o := range placed {
				spare = spare || o.in.Kind == "bar"
			}
		}
		if propPriority(g.in.Kind) >= 3 && !droppable[g.in.Kind] && !spare {
			placed = append(placed, g) // keep the key objects; the test flags them
		}
	}
	// keep the generator's order (draw order does not matter, but tests and seeds do)
	keep := map[*geoProp]bool{}
	for _, g := range placed {
		keep[g] = true
	}
	for _, g := range geos {
		if keep[g] {
			out = append(out, g.in)
		}
	}
	d.Props = out
}

// findFreeSpot looks for the nearest spot (rings around the original) where g stands on nothing.
func findFreeSpot(d *LevelDef, g *geoProp, placed []*geoProp) (Vec, bool) {
	orig := g.in
	probe := &geoProp{in: orig, k: g.k}
	for r := 4.0; r <= 90; r += 4 {
		n := int(2 * math.Pi * r / 6)
		if n < 8 {
			n = 8
		}
		for i := 0; i < n; i++ {
			a := 2 * math.Pi * float64(i) / float64(n)
			p := PropInst{Kind: orig.Kind, Var: orig.Var, X: math.Round(orig.X + r*math.Cos(a)), Y: math.Round(orig.Y + r*math.Sin(a))}
			if p.X < 20 || p.X > d.W-20 || p.Y < 40 || p.Y > d.H-12 {
				continue
			}
			probe.in = p
			probe.foot, probe.base = instFoot(p, g.k), instBase(p, g.k)
			if !probe.clashes(d, placed, nil) && inWalkable(d, probe.foot) {
				return Vec{p.X, p.Y}, true
			}
		}
	}
	return Vec{}, false
}

// inWalkable: the foot of a moved prop must stand on ground the level considers part of the party
// (inside a roaming area, or at least not outside the shape of a ship / station / island).
func inWalkable(d *LevelDef, foot Rect) bool {
	c := foot.Center()
	if d.Shape != nil { // ships, stations and the island: anywhere inside the outline is part of the party
		return d.Shape(c.X, c.Y, -4)
	}
	if len(d.Roam) == 0 {
		return true
	}
	for _, r := range d.Roam {
		if r.Grow(16).Has(c.X, c.Y) {
			return true
		}
	}
	return false
}
