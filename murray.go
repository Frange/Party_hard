package main

import (
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Murray, the evil skull on a stake: he sits in a few parties and never stops
// talking, as in Monkey Island 3. He is decoration (not a guest: he does not
// count for finishing the party and nobody can kill him); his lines go through
// the bubble manager like everybody else's.
// ---------------------------------------------------------------------------

type murrayT struct {
	next, until float64
	line        string
	last        int
}

var (
	murrays   = map[*Prop]*murrayT{}
	murrayLvl *Level
	murrayRng = rand.New(rand.NewSource(1993))
)

// murrayTalk advances Murray's chatter and returns the frame to draw (0 shut, 1 or 2 talking).
func (g *Game) murrayTalk(pr *Prop) int {
	if murrayLvl != g.lvl {
		murrays, murrayLvl = map[*Prop]*murrayT{}, g.lvl
	}
	st := murrays[pr]
	if st == nil {
		st = &murrayT{next: g.time + 1 + murrayRng.Float64()*2, last: -1}
		murrays[pr] = st
	}
	if g.time >= st.next && g.time >= st.until {
		l := g.freshPhrase("murray", murrayLines, murrayRng.Intn)
		if l == "" { // he has said everything lately: a short silence
			st.next = g.time + 2
		} else {
			st.line = l
			st.until = g.time + 2.2 + float64(len([]rune(st.line)))/20
			st.next = st.until + 1 + murrayRng.Float64()*2.5 // he talks all the time
		}
	}
	if g.time >= st.until {
		return 0
	}
	if x, y := g.wx(pr.X), g.wy(pr.Y); x > -10 && x < float64(screenW)+10 && y > 0 && y < float64(screenH)+60 {
		g.bubble(pr, bubGuest, Vec{pr.X, pr.Y}, st.line, false, -2)
	}
	return 1 + int(g.time*8)%2
}

// murrayLevels: the parties where he is stuck on his stake.
var murrayLevels = map[string]bool{"MIAMI BEACH": true, "CRUCERO NOCTURNO": true, "CEMENTERIO": true} // (v1.4.0: the ranch became BARCELONA, a city square)

// addMurray puts the skull outdoors, on open ground where it covers nothing (see murraySpotOK).
// A Murray already in the level that breaks the rule is moved; a level with no valid
// outdoor spot simply has no Murray.
func addMurray(d *LevelDef) {
	if d == nil || !murrayLevels[d.Title] {
		return
	}
	fitAllFootprints()
	var keep []PropInst
	had := false
	for _, p := range d.Props {
		if p.Kind != "murray" {
			keep = append(keep, p)
			continue
		}
		if !had && murraySpotOK(d, p) {
			had = true
			keep = append(keep, p)
		}
	}
	d.Props = keep
	if had {
		return
	}
	if in, ok := murrayPlace(d); ok {
		d.Props = append(d.Props, in)
	}
}

// murrayPlace scans the walkable ground of the level for valid spots and picks the one
// with the most room around it, preferring the edge of the map (out of the way).
func murrayPlace(d *LevelDef) (PropInst, bool) {
	best, bestScore, found := PropInst{}, math.Inf(-1), false
	for y := 60.0; y < d.H-20; y += 6 {
		for x := 30.0; x < d.W-30; x += 6 {
			in := PropInst{Kind: "murray", X: x, Y: y}
			if !murraySpotOK(d, in) {
				continue
			}
			sc := math.Min(murrayClearance(d, in), 48) - 0.04*math.Min(math.Min(x, d.W-x), math.Min(y, d.H-y))
			if sc > bestScore {
				best, bestScore, found = in, sc, true
			}
		}
	}
	return best, found
}

// propSpriteRect is the area a placed prop draws on (bottom-centre anchor).
func propSpriteRect(in PropInst) Rect {
	k := propKinds[in.Kind]
	if k == nil {
		return Rect{in.X, in.Y, 0, 0}
	}
	sc := in.Scale
	if sc == 0 {
		sc = 1
	}
	if len(k.Native) > 0 {
		b := k.Native[in.Var%len(k.Native)]
		return Rect{in.X - k.NAX*sc, in.Y - k.NAY*sc, float64(b.w) * sc, float64(b.h) * sc}
	}
	return Rect{in.X - float64(k.W)/2*sc, in.Y - float64(k.H)*sc, float64(k.W) * sc, float64(k.H) * sc}
}

// murrayHUD: screen areas covered by the HUD (score panel, title and level line, trophies, hint box)
// when the whole party fits on the screen.
var murrayHUD = []Rect{{0, 0, 270, 100}, {250, 0, 460, 85}, {800, 0, 160, 40}, {320, 420, 320, 120}}

// murrayPathFloors: open ground that is still a path (sidewalks, streets, boardwalks): not for Murray.
var murrayPathFloors = map[string]bool{"pave": true, "cobble": true, "asphalt": true, "dash": true, "boardwalk": true, "edge": true, "rails": true, "rail": true, "track": true, "pitch": true, "arena": true, "helipad": true}

// murrayFloorAt: the topmost floor kind under p.
func murrayFloorAt(d *LevelDef, x, y float64) string {
	kind := ""
	for _, f := range d.Floors {
		if f.R.Has(x, y) {
			kind = f.Kind
		}
	}
	return kind
}

// murraySpotOK is the placement rule: the whole sprite stands outdoors (never inside a building,
// a roofed area or near one), off paths, roads and pads, and it covers nothing: no prop (drawn
// area or footprint), wall, door, start, performer, dance floor, pool or animal spawn.
func murraySpotOK(d *LevelDef, in PropInst) bool {
	g := mkGeo(in)
	if g == nil {
		return false
	}
	spr := propSpriteRect(in)
	if spr.X < 8 || spr.Y < 8 || spr.X+spr.W > d.W-8 || spr.Y+spr.H > d.H-8 {
		return false
	}
	for _, r := range murrayHUD { // never under the score panel, the title or the hints
		if r.Hits(spr.Grow(6)) {
			return false
		}
	}
	lv := &Level{Def: d}
	area := spr.Grow(6)
	for yy := area.Y; yy <= area.Y+area.H; yy += area.H / 4 {
		for xx := area.X; xx <= area.X+area.W; xx += area.W / 4 {
			if !lv.Outdoor(Vec{xx, yy}) || murrayPathFloors[murrayFloorAt(d, xx, yy)] {
				return false
			}
			if d.Shape != nil && !d.Shape(xx, yy, -4) {
				return false
			}
		}
	}
	// keep clear of buildings: no room floor near the sprite
	near := spr.Grow(24)
	for _, f := range d.Floors {
		if (floorIndoor(f.Kind) && f.Kind != "deck" || f.Kind == "water" || f.Kind == "lava" || spotBadFloors[f.Kind]) && f.R.Hits(near) {
			return false
		}
	}
	for _, w := range d.Walls {
		if w.R.Hits(spr.Grow(10)) {
			return false
		}
	}
	for _, w := range d.Windows {
		if w.Hits(spr.Grow(10)) {
			return false
		}
	}
	for _, dr := range d.Doors {
		if dr.R.Grow(36).Hits(spr) {
			return false
		}
	}
	for _, r := range d.Roads {
		if r.R.Grow(10).Hits(spr) {
			return false
		}
	}
	if d.Highway && d.Road.Grow(10).Hits(spr) {
		return false
	}
	for _, r := range d.Pools {
		if r.Grow(16).Hits(spr) {
			return false
		}
	}
	for _, r := range d.Dance {
		if r.Grow(16).Hits(spr) {
			return false
		}
	}
	for _, pd := range d.Pads {
		if math.Hypot(pd.X-in.X, pd.Y-in.Y) < 70 {
			return false
		}
	}
	c := spr.Center()
	if dist(c, d.Start) < 90 || dist(Vec{in.X, in.Y}, d.Start) < 90 {
		return false
	}
	if (d.CarFrom != Vec{} || d.CarTo != Vec{}) && segDist(Vec{in.X, in.Y}, d.CarFrom, d.CarTo) < 40 {
		return false
	}
	for _, pf := range d.Performers {
		if spr.Grow(24).Has(pf.Pos.X, pf.Pos.Y) {
			return false
		}
	}
	for _, a := range d.Animals {
		if spr.Grow(20).Has(a.Pos.X, a.Pos.Y) {
			return false
		}
	}
	for _, o := range d.Props {
		if o.Kind == "murray" && o.X == in.X && o.Y == in.Y {
			continue
		}
		og := mkGeo(o)
		if og == nil {
			continue
		}
		if propSpriteRect(o).Hits(spr.Grow(4)) || og.foot.Grow(12).Hits(g.foot) {
			return false
		}
	}
	return inWalkable(d, g.foot) && !offGround(d, g.foot) && groundClash(d, in.Kind, g.k, g.foot, Vec{in.X, in.Y}) == ""
}

// murrayClearance: how much free room there is around the skull (to the nearest prop, wall or door).
func murrayClearance(d *LevelDef, in PropInst) float64 {
	spr := propSpriteRect(in)
	best := 200.0
	gap := func(r Rect) {
		dx := math.Max(0, math.Max(r.X-(spr.X+spr.W), spr.X-(r.X+r.W)))
		dy := math.Max(0, math.Max(r.Y-(spr.Y+spr.H), spr.Y-(r.Y+r.H)))
		best = math.Min(best, math.Hypot(dx, dy))
	}
	for _, o := range d.Props {
		if o.Kind != "murray" {
			gap(propSpriteRect(o))
		}
	}
	for _, w := range d.Walls {
		gap(w.R)
	}
	for _, dr := range d.Doors {
		gap(dr.R)
	}
	return best
}

// segDist: distance from p to the segment a-b.
func segDist(p, a, b Vec) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/l2))
	}
	return math.Hypot(p.X-a.X-t*dx, p.Y-a.Y-t*dy)
}

// murrayHidden: would a sprite drawn on r cover Murray (or be covered by him)?
func murrayHidden(d *LevelDef, r Rect) bool {
	for _, in := range d.Props {
		if in.Kind == "murray" && propSpriteRect(in).Grow(4).Hits(r) {
			return true
		}
	}
	return false
}
