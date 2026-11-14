package main

import (
	"hash/fnv"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// The three kids on wheels (Diego, César and Sam-sam) ride OUTSIDE every other party (2026-11-11, owner: "tienen que
// salir 1 de cada 2 pantallas FIJO"):
//   - WHICH parties: a fixed rule, no dice. The campaign is walked in order; a party gets them when it is at least
//     two after the last one that had them (1, 3, 5...) AND the kids have somewhere reasonable to ride there
//     (kidStreetFit: open ground connected to the party; never the space station, the pirate ship or the island). A
//     party with no room passes the turn to the next one, and the count goes on from there. The SKATE PARK always
//     has them (its bowls, skater_park.go), outside the rhythm. Today (26 parties): 1 3 5 7 9 11 (12 skate) 13 15
//     17 19 22 24 (the train station has no open ground: 21 passes to 22; the pirate ship none; the hockey rink has ground since v1.4.0 but falls right after 24). Random mode: the same answer as the
//     campaign party of the same title; a venue that is not in the campaign gets them when its title hashes odd and
//     it has room (kidStreetOnFor).
//   - WHERE: on the open ground of the level (pavements, squares, gardens, beach...: Level.Outdoor), never on a
//     road, in a pool or inside a building. Their area (kidStreet.comp) is the largest connected piece of that
//     ground the player can reach; every ride is a path INSIDE it (kidStreetPath, a search over its own cells), so
//     a kid can never wander into a house, a toilet queue or the dance floor of a club.
//   - WHAT: they ride from one spot to another, hop now and then (kidHopTick), and when they stop they do a trick
//     on the flat (an ollie, a kickflip, a 360, a tailwhip... with its name shouted: skaterTrickShout) or say one of
//     their lines (skaterLines). They run from the player as always (updateKid), also inside their area.
//   - COUNT: they arrive a few seconds after the start like every kid (addKidAt -> guestRoom, so goalMax, the
//     start room and the 2P caps are untouched) and are guests like any other: they can be killed, and a dead kid
//     never comes back (kidPlan.done). A kid of the three already in the party (a famous guest of the start) is
//     kept and the planned one is not added (kidPresent / kidDedupe).
//   - SAFETY: a kid who has not moved for kidStreetStuckT while riding is sent somewhere else; at the second time
//     he is put on the nearest cell of his area.
// In the parties without them, a guest dressed as one of the three (the random mix, a famous folder) becomes a
// plain guest, so they really are only where the rule says (debug -kidsnow / -with keep theirs).
// ---------------------------------------------------------------------------

const (
	kidStreetMinCells = 260  // cells (8x8 px) of open ground at least: ~16600 px², a pavement 960 px long and 18 wide
	kidStreetMinSpan  = 260. // and that long in x or y
	kidStreetStuckT   = 4.0  // seconds without moving while riding = stuck
	kidStreetTrickP   = 0.45 // a trick when he stops (else a line, sometimes nothing)
)

// kidStreetNo: never there, whatever the ground says (zero gravity, a ship at sea).
var kidStreetNo = map[string]bool{"station": true, "pirate": true}

// kidStreetFit: the campaign parties with room for the three outside (TestKidStreetFitMatchesTheLevels measures it
// on the built levels and fails if this list and the levels disagree).
var kidStreetFit = map[string]bool{"house": true, "beach": true, "neon": true, "ranch": true, "bull": true,
	"eiffel": true, "boat": true, "funfair": true, "stadium": true, "disco": true, "festival": true, "grave": true,
	"cinema": true, "circuit": true, "snow": true, "roof": true, "casino": true, "highway": true, "italy": true,
	"cake": true, "hockey": true, "disco70": true, "alcala": true, "guadalajara": true} // hockey: open since its two doors (v1.4.0); it falls right after the birthday, so the rhythm skips it

// kidStreetCampaign: for every campaign index, whether the three ride there (the fixed rule).
func kidStreetCampaign() []bool {
	on := make([]bool, len(campaignLevels))
	next := 0
	for i, e := range campaignLevels {
		if e.icon == "skate" { // the bowls: always, outside the rhythm (the odd parties stay odd)
			on[i] = true
			continue
		}
		if i >= next && kidStreetFit[e.icon] && !kidStreetNo[e.icon] {
			on[i], next = true, i+2
		}
	}
	return on
}

// kidStreetOnFor: whether the three come to this party (idx: the level index; def: its definition; l: the built
// level, only needed for a random venue that is not part of the campaign).
func kidStreetOnFor(idx int, def *LevelDef, l *Level) bool {
	if def == nil || idx == specialJurassic || kidStreetNo[def.Icon] {
		return false
	}
	camp := kidStreetCampaign()
	if idx >= 0 && idx < campaignN && idx < len(camp) {
		return camp[idx]
	}
	for i, e := range campaignLevels { // random mode: the campaign party of the same title decides
		if i < len(camp) && e.title != "" && e.title == def.Title {
			return camp[i]
		}
		if i < len(camp) && i < len(levelDefs) && levelDefs[i] != nil && levelDefs[i].Title != "" && levelDefs[i].Title == def.Title {
			return camp[i]
		}
	}
	h := fnv.New32a()
	h.Write([]byte(def.Title))
	if h.Sum32()%2 == 0 || l == nil {
		return false
	}
	comp, _ := kidStreetArea(l)
	return len(comp) > 0
}

// kidStreet: the area of this party (one at a time, like skateTurn).
var kidStreet struct {
	lvl  *Level
	on   bool
	comp []int  // cells of the area (cy*cw+cx)
	in   []bool // per cell of the level
	own  map[*Guest]bool
	st   map[*Guest]*kidStreetSt
	rng  *rand.Rand
}

type kidStreetSt struct {
	lastPos  Vec
	still    float64 // seconds without moving while riding
	stuckN   int
	trickEnd float64
	spinNext float64
	stopDone bool // the trick (or line) of this stop was done
}

// kidStreetSetup: called by planKids when a party loads.
func (g *Game) kidStreetSetup() {
	kidStreet.lvl, kidStreet.on, kidStreet.comp, kidStreet.in = g.lvl, false, nil, nil
	kidStreet.own, kidStreet.st = map[*Guest]bool{}, map[*Guest]*kidStreetSt{}
	kidStreet.rng = rand.New(rand.NewSource(int64(g.levelIdx)*7727 + 13))
	if g.lvl == nil || g.stealth || len(g.lvl.bowls) > 0 {
		return
	}
	if !kidStreetOnFor(g.levelIdx, g.lvl.Def, g.lvl) {
		return
	}
	comp, _ := kidStreetArea(g.lvl)
	if len(comp) == 0 {
		return
	}
	kidStreet.on, kidStreet.comp = true, comp
	kidStreet.in = make([]bool, g.lvl.cw*g.lvl.ch)
	for _, c := range comp {
		kidStreet.in[c] = true
	}
}

// kidStreetActive: the three ride outside in this party.
func (g *Game) kidStreetActive() bool {
	return kidStreet.on && kidStreet.lvl == g.lvl && g.lvl != nil
}

// kidStreetCell: a cell where a kid may ride (open ground, free for a guest, not a road, a pool or a helipad).
func kidStreetCell(l *Level, cx, cy int) bool {
	if cx < 0 || cy < 0 || cx >= l.cw || cy >= l.ch {
		return false
	}
	i := cy*l.cw + cx
	if l.blocked[i] || len(l.blockedG) == len(l.blocked) && l.blockedG[i] {
		return false
	}
	p := cellCenter(cx, cy)
	if p.X < 12 || p.Y < 12 || p.X > l.W-12 || p.Y > l.H-12 {
		return false
	}
	if !l.Outdoor(p) || l.SolidP(p.X, p.Y) || l.InPool(p) || l.badGround(p) {
		return false
	}
	kind := ""
	for _, f := range l.Def.Floors {
		if f.R.Has(p.X, p.Y) {
			kind = f.Kind
		}
	}
	switch kind {
	case "asphalt", "dash", "rails", "edge":
		return false // roads, tracks and the edge of a roof
	}
	if d := l.Def; d.CarFrom != d.CarTo && math.Abs(p.Y-d.CarFrom.Y) < 22 { // the lane of the cars
		return false
	}
	return true
}

// kidStreetArea: the largest connected piece of open ground the player can reach from the start, if it is big
// enough (else nil). span: its longest side in pixels.
func kidStreetArea(l *Level) ([]int, float64) {
	if l == nil || l.cw == 0 {
		return nil, 0
	}
	n := l.cw * l.ch
	// what the player can reach from the start (whole grid)
	reach := make([]bool, n)
	sx, sy := l.cellOf(l.NearestFree(l.Def.Start))
	if l.cellFree(sx, sy) {
		q := []int{sy*l.cw + sx}
		reach[q[0]] = true
		for len(q) > 0 {
			c := q[0]
			q = q[1:]
			cx, cy := c%l.cw, c/l.cw
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				x, y := cx+d[0], cy+d[1]
				if l.cellFree(x, y) && !reach[y*l.cw+x] {
					reach[y*l.cw+x] = true
					q = append(q, y*l.cw+x)
				}
			}
		}
	}
	ok := make([]bool, n)
	for cy := 0; cy < l.ch; cy++ {
		for cx := 0; cx < l.cw; cx++ {
			ok[cy*l.cw+cx] = reach[cy*l.cw+cx] && kidStreetCell(l, cx, cy)
		}
	}
	seen := make([]bool, n)
	var best []int
	bestSpan := 0.0
	for c := 0; c < n; c++ {
		if !ok[c] || seen[c] {
			continue
		}
		comp := []int{c}
		seen[c] = true
		x0, x1, y0, y1 := c%l.cw, c%l.cw, c/l.cw, c/l.cw
		for k := 0; k < len(comp); k++ {
			cx, cy := comp[k]%l.cw, comp[k]/l.cw
			x0, x1, y0, y1 = minI(x0, cx), maxI(x1, cx), minI(y0, cy), maxI(y1, cy)
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				x, y := cx+d[0], cy+d[1]
				if x < 0 || y < 0 || x >= l.cw || y >= l.ch {
					continue
				}
				if j := y*l.cw + x; ok[j] && !seen[j] {
					seen[j] = true
					comp = append(comp, j)
				}
			}
		}
		if len(comp) > len(best) {
			best, bestSpan = comp, math.Max(float64(x1-x0+1), float64(y1-y0+1))*navCell
		}
	}
	if len(best) < kidStreetMinCells || bestSpan < kidStreetMinSpan {
		return nil, bestSpan
	}
	return best, bestSpan
}

// kidStreetNearest: the cell of the area nearest to p.
func (g *Game) kidStreetNearest(p Vec) int {
	l := g.lvl
	best, bd := -1, math.MaxFloat64
	for _, c := range kidStreet.comp {
		if d := dist(cellCenter(c%l.cw, c/l.cw), p); d < bd {
			best, bd = c, d
		}
	}
	return best
}

// kidStreetPath: a path from p to the cell `to` that never leaves the area (breadth-first over its cells). If p is
// not in the area, the path starts with the way to the nearest cell of it (PathG).
func (g *Game) kidStreetPath(p Vec, to int) []Vec {
	l := g.lvl
	var lead []Vec
	cx, cy := l.cellOf(p)
	from := cy*l.cw + cx
	if cx < 0 || cy < 0 || cx >= l.cw || cy >= l.ch || !kidStreet.in[from] {
		from = g.kidStreetNearest(p)
		if from < 0 {
			return nil
		}
		lead = l.PathG(p, cellCenter(from%l.cw, from/l.cw))
	}
	prev := map[int]int{from: -1}
	q := []int{from}
	for len(q) > 0 && q[0] != to {
		c := q[0]
		q = q[1:]
		x, y := c%l.cw, c/l.cw
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= l.cw || ny >= l.ch {
				continue
			}
			j := ny*l.cw + nx
			if _, ok := prev[j]; ok || !kidStreet.in[j] {
				continue
			}
			prev[j] = c
			q = append(q, j)
		}
	}
	if _, ok := prev[to]; !ok {
		return lead
	}
	var cells []int
	for c := to; c >= 0; c = prev[c] {
		cells = append(cells, c)
	}
	// reversed, and only the corners (a straight run of cells needs no point in between)
	var out []Vec
	for k := len(cells) - 2; k >= 0; k-- {
		if k > 0 {
			a, b, c := cells[k+1], cells[k], cells[k-1]
			if b-a == c-b {
				continue
			}
		}
		out = append(out, cellCenter(cells[k]%l.cw, cells[k]/l.cw))
	}
	return append(lead, out...)
}

// kidStreetPick: a kid with nothing to do (guestPick): the next ride inside the area. False = not a street kid.
func (g *Game) kidStreetPick(gu *Guest) bool {
	if !g.kidStreetActive() || !isKid(gu) || len(kidStreet.comp) == 0 {
		return false
	}
	r := kidStreet.rng
	l := g.lvl
	gu.need = 0 // no toilet trips: he does not go inside
	to := kidStreet.comp[r.Intn(len(kidStreet.comp))]
	for k := 0; k < 8; k++ {
		c := kidStreet.comp[r.Intn(len(kidStreet.comp))]
		if dist(cellCenter(c%l.cw, c/l.cw), gu.pos) > 90 {
			to = c
			break
		}
	}
	gu.path = g.kidStreetPath(gu.pos, to)
	gu.hurry = false
	gu.state, gu.next, gu.dur = gGoto, gIdle, 0.6+r.Float64()*1.4
	if len(gu.path) == 0 {
		gu.state, gu.timer = gIdle, 1+r.Float64()
	}
	return true
}

// kidStreetTick: every frame of a street kid (updateKid): the trick when he stops and the stuck watchdog.
func (g *Game) kidStreetTick(gu *Guest, dt float64) {
	if !g.kidStreetActive() || gu.out() || g.mode != ModePlay {
		return
	}
	st := kidStreet.st[gu]
	if st == nil {
		st = &kidStreetSt{lastPos: gu.pos}
		kidStreet.st[gu] = st
	}
	r := kidStreet.rng
	// the leash: another system of the party (a group, a greeting, a star, a body...) sent him somewhere off his
	// open ground: he goes back to riding there instead
	if (gu.state == gGoto || gu.state == gIdle) && gu.timer < 1000 && !g.pevTaken(gu) && !g.kidStreetFrom(gu) {
		g.kidStreetPick(gu)
	}
	// a trick on the flat when he stops (once per stop): the hop, a spin and the name of the trick
	if gu.state == gIdle && len(gu.path) == 0 {
		if !st.stopDone && gu.icon == "" {
			st.stopDone = true
			if r.Float64() < kidStreetTrickP {
				tricks := kidStreetTricks(gu)
				trick := tricks[r.Intn(len(tricks))]
				gu.icon, gu.iconT = skaterTrickShout[trick][0], 1.6
				st.trickEnd, st.spinNext = g.time+1.2, g.time
				gu.timer = math.Max(gu.timer, 1.3) // he finishes the trick before the next ride
				if hs := kidHops[gu]; hs != nil && kidHopLvl == g.lvl {
					hs.start = g.time
				}
			} else if r.Float64() < 0.5 {
				slug := ""
				if gu.cs != nil && gu.cs.Spec != nil {
					slug = gu.cs.Spec.Name
				}
				ls := skaterLines(slug)
				gu.icon, gu.iconT = ls[r.Intn(len(ls))][0], 2
			}
		}
	} else {
		st.stopDone = false
	}
	if g.time < st.trickEnd && g.time >= st.spinNext { // the spin of the trick: round he goes
		st.spinNext = g.time + 0.15
		gu.dir = Dir((int(gu.dir) + 1) % 4)
	}
	// the watchdog: riding but not moving
	if gu.state == gGoto && len(gu.path) > 0 {
		if dist(gu.pos, st.lastPos) < 4 {
			st.still += dt
		} else {
			st.still, st.lastPos = 0, gu.pos
		}
		if st.still > kidStreetStuckT {
			st.still, st.lastPos = 0, gu.pos
			st.stuckN++
			if st.stuckN >= 2 {
				if c := g.kidStreetNearest(gu.pos); c >= 0 {
					gu.pos = cellCenter(c%g.lvl.cw, c/g.lvl.cw)
				}
				st.stuckN = 0
			}
			gu.path = nil
			g.kidStreetPick(gu)
		}
	} else if st.still > 0 {
		st.still, st.lastPos = 0, gu.pos
	}
}

// kidStreetFrom: the kid is where he should be: walking to a cell of his area, or standing on one.
func (g *Game) kidStreetFrom(gu *Guest) bool {
	l := g.lvl
	p := gu.pos
	if gu.state == gGoto && len(gu.path) > 0 {
		p = gu.path[len(gu.path)-1]
	}
	if !kidStreetInCell(l, p) {
		return false
	}
	// riding inside his area, the way there must not cut through a building either: another system (a greeting, a
	// group...) gave him a path whose end is on his ground but that crossed an indoor strip on the way (the cruise,
	// one party in a few). Only checked while he stands in his area, so the way back from outside (kidStreetPath)
	// is never refused again and again.
	if gu.state == gGoto && len(gu.path) > 0 && kidStreetInCell(l, gu.pos) {
		a := gu.pos
		for _, b := range gu.path {
			n := int(dist(a, b)/(navCell/2)) + 1
			for k := 1; k <= n; k++ {
				f := float64(k) / float64(n)
				q := Vec{a.X + (b.X-a.X)*f, a.Y + (b.Y-a.Y)*f}
				if !kidStreetInCell(l, q) && !l.Outdoor(q) {
					return false
				}
			}
			a = b
		}
	}
	return true
}

// kidStreetInCell: p is on a cell of the street kids' area.
func kidStreetInCell(l *Level, p Vec) bool {
	cx, cy := l.cellOf(p)
	return cx >= 0 && cy >= 0 && cx < l.cw && cy < l.ch && kidStreet.in[cy*l.cw+cx]
}

// kidStreetTricks: the tricks he can do on the flat (Diego on his board, the other two on scooters).
func kidStreetTricks(gu *Guest) []int {
	if gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Name == "diego" {
		return []int{trOllie, trKickflip, trHeelflip, trSpin180, trSpin360}
	}
	return []int{trOllie, trTailwhip, trBarspin, trNoHander, trSpin180}
}

// kidStreetFleeTo: where a street kid runs from the player (far from him, inside the area).
func (g *Game) kidStreetFleeTo(gu *Guest) ([]Vec, bool) {
	if !g.kidStreetActive() || len(kidStreet.comp) == 0 {
		return nil, false
	}
	l := g.lvl
	r := kidStreet.rng
	best, bd := -1, -1.0
	for i := 0; i < 12; i++ {
		c := kidStreet.comp[r.Intn(len(kidStreet.comp))]
		t := cellCenter(c%l.cw, c/l.cw)
		if dist(t, gu.pos) < 80 {
			continue
		}
		if d := dist(t, g.p.pos) - 0.3*dist(t, gu.pos); d > bd {
			best, bd = c, d
		}
	}
	if best < 0 {
		return nil, true
	}
	return g.kidStreetPath(gu.pos, best), true
}

// kidStreetArrivePos: where a street kid rides in: the cell of the area nearest to the usual pavement entry.
func (g *Game) kidStreetArrivePos(entry Vec) (Vec, bool) {
	if !g.kidStreetActive() {
		return entry, false
	}
	c := g.kidStreetNearest(entry)
	if c < 0 {
		return entry, false
	}
	return cellCenter(c%g.lvl.cw, c/g.lvl.cw), true
}

// kidStreetOwn: this kid was brought by the rule (or by a debug switch): never turned into a plain guest.
func kidStreetOwn(gu *Guest) {
	if kidStreet.own == nil {
		kidStreet.own = map[*Guest]bool{}
	}
	kidStreet.own[gu] = true
}
