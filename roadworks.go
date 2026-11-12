package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// ROADWORKS (idea 293 of the user: "que puedan hacer obras en la calle y estén varios obreros trabajando ahí, con unos
// conos para que se aparten un poco los coches y no les atropellen. Que estén con taladros y que haya un pequeño
// agujero en la carretera con una tubería echando agua hacia arriba").
//
// It is NOT a party event of the frame (the mean of events per party must not go up): it is an ELEMENT OF THE LEVEL,
// like the mechanical bull. loadLevel rolls it with its OWN dice (rwRoll, one party in 3.3 of the levels with the street
// of the police cars: streetOK) and buildLevel puts the zone (Level.addRoadworks) on the kerb of the road:
//   - a HOLE in the asphalt with a broken PIPE ("roadhole", not solid: people walk round and over it) that throws a jet
//     of water up (drawn here, a pure function of the time) and a puddle around it (only decoration: there is no
//     slipping system in the game);
//   - a row of orange CONES and two small BARRIERS along the edge of the lane ("roadcone", "roadbarrier": only their
//     real footprint is solid);
//   - a small tower CRANE on the pavement (crane.go) with a load hanging over the place of one of the workers.
// The spot is chosen like the bull's: away from the start, the dance floor, the places of use, the performers, the VIP
// zone and the parking of the police; every floor cell reachable before is still reachable (else it is undone).
//
// THE WORKERS: 3-4 PLAIN GUESTS of the party are dressed as workers (hard hat, orange vest, blue trousers) when it
// starts: nobody is added (caps of cap.go untouched, 1 and 2 players), they can be killed and they are witnesses. They
// stand on the pavement and drill the kerb with jackhammers (they shake, dust flies), rest, drink water and shout
// ("¡Cuidado, obras!"). Their quirk (rwQuirk) keeps the party events from taking them. If one is killed the others
// are frightened (alarm) and come back to work later (each walk has a time limit: nobody jams).
//
// THE TRAFFIC: the cars of the lane next to the works move a few pixels away from the kerb while they pass (they keep
// their lane) and a car never drives into a worker who stands in its way (it waits, like streetHold).
//
// Dice: rwDice() (never g.rng). Debug: -roadworks or -pevent obras[:crane|:drop] forces the zone in the level given
// (the progress is frozen like with every debug flag); "crane" puts the player beside the crane, "drop" drops the load.
// ---------------------------------------------------------------------------

const (
	rwChance   = 0.3
	rwQuirk    = "obrero"
	rwSwerve   = 10.0  // how far the cars of the near lane move away from the kerb
	rwSwerveIn = 90.0  // full swerve while closer than this (x) to the hole
	rwSwerveTo = 230.0 // a long ramp: two cars 70 px apart (the gap where they follow each other) differ < 8 px (same lane)
	rwWalkMax  = 22.0  // seconds to walk back to his place, else he is let go and tries later
	rwHoleName = "roadhole"
	rwConeName = "roadcone"
	rwBarName  = "roadbarrier"
)

var (
	rwRng      *rand.Rand
	rwSeed     int64 // tests: non-zero = deterministic dice
	rwDecision bool  // loadLevel sets it just before buildLevel (like toroDecision) and clears it after
	rwOff      bool  // tests: no works anywhere
	rwForce    bool  // tests: the works for sure where they fit
	rwInTests  bool  // tests: roll the real dice under `go test`
	rwFlag     = flag.Bool("roadworks", false, "debug: roadworks (workers, cones, crane) in the level given (progress frozen)")
)

var (
	rwWarn  = pevTexts("¡Cuidado, obras!", "¡Que no se acerque nadie!", "¡Por la otra acera!", "¡Ojo con la zanja!")
	rwChat  = pevTexts("¡Dale caña al taladro!", "Esta tubería no aguanta más", "¿Quién ha pedido agua?", "Uf, qué calor", "Media hora y al bocadillo", "¡Pásame la llave inglesa!")
	rwMourn = pevTexts("¡Han matado a un compañero!", "¡Se acabó la obra!", "¡Que venga el capataz!")
	rwBack  = pevTexts("Bueno... a trabajar", "Venga, que hay que acabar")
	rwDodge = pevTexts("¡Uy, por poco!")
)

func rwDice() *rand.Rand {
	if rwRng == nil {
		sd := rwSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x0b7a5c0e
		}
		rwRng = rand.New(rand.NewSource(sd))
	}
	return rwRng
}

// rwDebug: -roadworks, -pevent obras (or obras:crane / obras:drop).
func rwDebug() bool {
	return rwFlag != nil && *rwFlag || rwPeventID(dbgPartyEvent)
}

// rwPeventID: -pevent obras[:crane|:drop] is a valid id although the works are not a party event (main.go).
func rwPeventID(id string) bool { return id == "obras" || strings.HasPrefix(id, "obras:") }

func rwStage() string {
	_, st, _ := strings.Cut(dbgPartyEvent, ":")
	return st
}

// rwFits: a campaign level with the street of the police cars and ordinary traffic.
func rwFits(def *LevelDef) bool {
	return def != nil && def.Title != "" && streetOK(def) && len(def.Roam) > 0 && (def.PropScale == 0 || def.PropScale == 1) &&
		(def.CharScale == 0 || def.CharScale == 1)
}

// rwRoll: does this party get roadworks? (loadLevel; the dice are always drawn).
func (g *Game) rwRoll(def *LevelDef) bool {
	roll := rwDice().Float64()
	if rwOff || def == nil || def.Title == "" || g.stealth || g.levelIdx == specialJurassic || !rwFits(def) {
		return false
	}
	if rwDebug() {
		return true
	}
	if g.dc != nil {
		return false
	}
	if rwForce {
		return true
	}
	if testing.Testing() && !rwInTests {
		return false
	}
	return roll < rwChance
}

func init() {
	roadOK[rwConeName], roadOK[rwBarName], roadOK[rwHoleName] = true, true, true
	propKinds[rwHoleName] = &PropKind{W: 30, H: 14, FW: 16, FH: 4, NoSolid: true, SortOff: -16, Category: "decor", Draw: drawRoadHole}
	propKinds[rwConeName] = &PropKind{W: 10, H: 13, FW: 6, FH: 3, Category: "decor", Draw: drawRoadCone}
	propKinds[rwBarName] = &PropKind{W: 18, H: 12, FW: 16, FH: 3, Category: "decor", Draw: drawRoadBarrier}
}

// ---- the props ------------------------------------------------------------------------------------------

func drawRoadHole(b *pixBuf, v int) {
	// broken asphalt round a dark hole, a few loose chunks and the stub of a cut pipe sticking out
	edge, dark, mud := c(0x55555f), c(0x14141a), c(0x5a4630)
	b.ellipse(15, 9, 12, 4.5, edge)
	b.ellipse(15, 9, 10, 3.6, mud)
	b.ellipse(15, 9.5, 8, 2.8, dark)
	for _, ch := range [][2]int{{2, 7}, {26, 8}, {5, 12}, {24, 12}, {13, 3}} {
		b.rect(ch[0], ch[1], 2, 1, edge)
		b.rect(ch[0], ch[1]+1, 2, 1, c(0x2c2c34))
	}
	pipe, pipeL, pipeD := c(0x8a929e), c(0xc8d0da), c(0x4a505a)
	b.rect(14, 4, 4, 6, pipe)
	b.rect(14, 4, 1, 6, pipeL)
	b.rect(17, 4, 1, 6, pipeD)
	b.rect(13, 3, 6, 2, pipeD)
	b.rect(14, 3, 4, 1, pipeL)
}

func drawRoadCone(b *pixBuf, v int) {
	or, orD, wh := c(0xf07818), c(0xb84e0c), c(0xf4f4f0)
	b.rect(1, 11, 8, 2, c(0x202026))
	for y := 1; y <= 10; y++ {
		w := 1 + y*6/10
		x0 := 5 - w/2
		if w%2 == 0 {
			x0 = 5 - w/2
		}
		b.rect(x0, y, w, 1, or)
		b.rect(x0+w-1, y, 1, 1, orD)
	}
	b.rect(3, 5, 4, 1, wh)
	b.rect(3, 6, 4, 1, wh)
	b.rect(4, 0, 2, 1, orD)
}

func drawRoadBarrier(b *pixBuf, v int) {
	red, wh, leg := c(0xd02a2a), c(0xf4f4f0), c(0x34343c)
	b.rect(2, 4, 2, 8, leg)
	b.rect(14, 4, 2, 8, leg)
	b.rect(1, 11, 4, 1, leg)
	b.rect(13, 11, 4, 1, leg)
	b.rect(0, 2, 18, 4, wh)
	for x := 0; x < 18; x += 6 {
		b.rect(x, 2, 3, 4, red)
	}
	b.rect(0, 5, 18, 1, c(0x8a8a92))
	b.rect(8, 0, 2, 2, c(0xf0a020)) // the warning light
}

// ---- the site (a built level) -----------------------------------------------------------------------------

// rwSite: the geometry of the works of a level, all derived from the hole and the crane props.
type rwSite struct {
	hole  *Prop
	crane *Prop
	sd    float64 // -1 the pavement is above the road (y smaller), +1 below
	edge  float64 // y of the edge of the asphalt
	lane  float64 // y of the lane next to the works
	cs    float64 // the crane stands on this side of the hole (+1 right, -1 left)
}

func (l *Level) rwHole() *Prop {
	if l == nil {
		return nil
	}
	for _, pr := range l.Props {
		if pr.Name == rwHoleName && !pr.gone {
			return pr
		}
	}
	return nil
}

// rwSiteOf: the works of the level (nil = none). The hole keeps the side in Var (0 = pavement above, 1 = below).
func (l *Level) rwSiteOf() *rwSite {
	h := l.rwHole()
	if h == nil {
		return nil
	}
	s := &rwSite{hole: h, sd: -1, cs: 1}
	if h.Var == 1 {
		s.sd = 1
	}
	s.edge = h.Y - 5 + s.sd*rwHoleIn
	s.lane = rwLaneY(l.Def, s.sd)
	if cr := l.craneProp(); cr != nil {
		s.crane = cr
		s.cs = 1
		if cr.X < h.X {
			s.cs = -1
		}
	}
	return s
}

const rwHoleIn = 4.0 // the hole is this far into the road from the edge

// rwLaneY: the y of the lane on that side of the centre line.
func rwLaneY(d *LevelDef, sd float64) float64 {
	if len(d.Lanes) > 0 {
		best := d.Lanes[0]
		for _, y := range d.Lanes {
			if sd < 0 && y < best || sd > 0 && y > best {
				best = y
			}
		}
		return best
	}
	lo := d.LaneOff
	if lo <= 0 {
		lo = laneOffset
	}
	return d.CarFrom.Y + sd*lo
}

// rwEdgeY: the edge of the asphalt on that side (the "asphalt" floor that holds the lane), or a guess.
func rwEdgeY(d *LevelDef, lane, sd float64) float64 {
	for _, f := range d.Floors {
		if f.Kind != "asphalt" || !f.R.Has(d.W/2, lane) {
			continue
		}
		e := f.R.Y
		if sd > 0 {
			e = f.R.Y + f.R.H
		}
		if dd := (e - lane) * sd; dd >= 10 && dd <= 24 { // a kerb too close to the lane: the guess below keeps the workers clear
			return e
		}
	}
	return lane + sd*10
}

// The places of a site around the hole at x on the edge e (sd: the pavement side, cs: the crane side).
func rwWorkerSpots(x, e, sd, cs float64) []Vec {
	return []Vec{{x - 14, e + sd*8}, {x + 14, e + sd*8}, {x + cs*40, e + sd*9}, {x - cs*40, e + sd*12}}
}
func rwLoadAt(x, e, sd, cs float64) Vec  { return Vec{x + cs*40, e + sd*9} }
func rwCraneAt(x, e, sd, cs float64) Vec { return Vec{x + cs*(40+craneJib), e + sd*5} }
func rwConeXs() []float64                { return []float64{-56, -40, -24, 24, 40, 56} }

func (s *rwSite) spots() []Vec { return rwWorkerSpots(s.hole.X, s.edge, s.sd, s.cs) }
func (s *rwSite) load() Vec    { return rwLoadAt(s.hole.X, s.edge, s.sd, s.cs) }

// addRoadworks puts the works on the kerb of a built level when loadLevel decided so. true if it placed them.
func (l *Level) addRoadworks() bool {
	d := l.Def
	if !rwDecision || rwOff || vipBuilding || !rwFits(d) || l.cw == 0 || l.rwHole() != nil {
		return false
	}
	fitAllFootprints()
	r := rand.New(rand.NewSource(rwDice().Int63()))
	reach0 := l.reachFrom(l.NearestFree(d.Start))
	var avoid []Vec
	for _, ps := range l.POIs {
		avoid = append(avoid, ps...)
	}
	for _, pf := range d.Performers {
		avoid = append(avoid, pf.Pos)
	}
	for _, sd := range []float64{-1, 1} {
		lane := rwLaneY(d, sd)
		e := rwEdgeY(d, lane, sd)
		if e < 30 || e > d.H-30 {
			continue
		}
		var xs []float64
		for x := 150.0; x <= d.W-150; x += 12 {
			xs = append(xs, x)
		}
		r.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
		tries := 0
		for _, x := range xs {
			cs := 1.0
			if r.Intn(2) == 0 {
				cs = -1
			}
			for k := 0; k < 2; k++ {
				if l.rwSpotOK(x, e, sd, cs, avoid, reach0) {
					if tries++; tries > 60 {
						break
					}
					if l.rwTry(x, e, sd, cs, reach0) {
						return true
					}
				}
				cs = -cs
			}
		}
	}
	return false
}

func (l *Level) rwDry(q Vec) bool {
	return l.Free(q) && !l.SolidP(q.X, q.Y) && !l.badGround(q) && !l.InPool(q)
}

// rwSpotOK: the quick checks before building anything.
func (l *Level) rwSpotOK(x, e, sd, cs float64, avoid []Vec, reach0 []bool) bool {
	d := l.Def
	cr := rwCraneAt(x, e, sd, cs)
	lo, hi := math.Min(x-76, cr.X-20), math.Max(x+76, cr.X+20)
	if lo < 40 || hi > d.W-40 {
		return false
	}
	area := Rect{lo, e - 26, hi - lo, 38} // the pavement side and the strip of the cones
	if sd > 0 {
		area.Y = e - 12
	}
	if dist(Vec{x, e}, d.Start) < 130 || area.Grow(10).Has(d.Start.X, d.Start.Y) {
		return false
	}
	// the police park on the street: keep their place free
	if math.Abs(x-d.CarTo.X) < 100 || d.ParkX1 > d.ParkX0 && hi > d.ParkX0-60 && lo < d.ParkX1+60 {
		return false
	}
	for _, p := range rwWorkerSpots(x, e, sd, cs) {
		if !l.rwDry(p) || reach0 != nil && !reach0[l.navIdx(p)] {
			return false
		}
	}
	for _, dr := range d.Dance {
		if dr.Grow(20).Hits(area) {
			return false
		}
	}
	if z := l.vip; z != nil {
		for _, rc := range []Rect{z.in, z.trig, z.gate} {
			if rc.Grow(20).Hits(area) {
				return false
			}
		}
	}
	for _, a := range avoid {
		if area.Grow(16).Has(a.X, a.Y) {
			return false
		}
	}
	// nothing else drawn on the hole, the cones or the place of the workers
	for _, o := range l.Props {
		if pbOverlap(o.Foot.Grow(2), area) || !overlayKinds[o.Name] && pbOverlap(o.base().Grow(2), area) {
			return false
		}
	}
	return !l.propClashes(craneName, propKinds[craneName], cr, 0)
}

// rwTry builds the works at x; false (and nothing changed) if it closes a way or covers a place.
func (l *Level) rwTry(x, e, sd, cs float64, reach0 []bool) bool {
	nProps, nP, nN := len(l.Props), len(l.solidsP), len(l.solidsN)
	undo := func() {
		l.Props, l.solidsP, l.solidsN = l.Props[:nProps], l.solidsP[:nP], l.solidsN[:nN]
		l.initNav()
	}
	hv := 0
	if sd > 0 {
		hv = 1
	}
	l.addTableProp(rwHoleName, propKinds[rwHoleName], Vec{x, e - sd*rwHoleIn + 5}, hv)
	cy := e - sd*5 + 2 // the foot of the cones, inside the road
	for _, dx := range rwConeXs() {
		l.addTableProp(rwConeName, propKinds[rwConeName], Vec{x + dx, cy}, 0)
	}
	for _, dx := range []float64{-72, 72} {
		l.addTableProp(rwBarName, propKinds[rwBarName], Vec{x + dx, cy}, 0)
	}
	cv := 0
	if cs < 0 {
		cv = 1
	}
	cr := rwCraneAt(x, e, sd, cs)
	l.addTableProp(craneName, propKinds[craneName], cr, cv)
	l.initNav()
	reach1 := l.reachFrom(l.NearestFree(l.Def.Start))
	ok := func(q Vec) bool { return l.rwDry(q) && reach1[l.navIdx(q)] }
	for _, p := range rwWorkerSpots(x, e, sd, cs) {
		if !ok(p) {
			undo()
			return false
		}
	}
	if !ok(craneUseAt(cr, sd, cs)) {
		undo()
		return false
	}
	if len(reach0) == len(reach1) {
		for i := range reach0 {
			if reach0[i] && !reach1[i] && !l.blocked[i] {
				undo()
				return false
			}
		}
	}
	return true
}

// ---- the workers (at play time) ----------------------------------------------------------------------------

const (
	rwTaskDrill = iota
	rwTaskRest
	rwTaskDrink
	rwTaskTalk
)

type rwWorker struct {
	m       pevMem
	spot    Vec
	task    int
	until   float64
	away    bool
	backAt  float64
	walkT0  float64
	mourned bool
	gone    bool // not ours any more (killed, made a star...)
	dustT   float64
}

type rwRun struct {
	lvl     *Level
	site    *rwSite
	started bool
	ws      []*rwWorker
	chatT   float64
	swerve  map[*Ambient]float64 // the lane y of each car that is being moved aside
	staged  bool
	placed  bool // debug: the player was put beside the crane
	crane   craneState
}

var (
	rw       *rwRun
	rwNoneAt *Level // the last level looked at that has no works (not searched again every frame)
)

// rwActive: the run of the level being played (nil = no works).
func (g *Game) rwActive() *rwRun {
	if g.lvl == nil || g.lvl == rwNoneAt {
		return nil
	}
	if rw == nil || rw.lvl != g.lvl {
		rw = nil
		if s := g.lvl.rwSiteOf(); s != nil {
			rw = &rwRun{lvl: g.lvl, site: s, swerve: map[*Ambient]float64{}}
		} else {
			rwNoneAt = g.lvl
		}
	}
	return rw
}

// rwWorkerSpec: a worker's clothes on top of a generic guest (his own face and hair).
func rwWorkerSpec(r *rand.Rand) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Top, s.Bottom = TopVest, BotPants
	s.Shirt, s.Shirt2 = hx(0xf07a1a), hx(0xe8e070)
	s.Pants, s.Shoes = hx(0x2e3e62), hx(0x5a3a1e)
	s.Hat, s.HatCol = HatHardhat, hx(0xf0c820)
	s.Carry, s.Weapon, s.Cape, s.Mask, s.MaskKind = 0, 0, color.RGBA{}, color.RGBA{}, 0
	s.Ride, s.Glasses = 0, GlassNone
	return s
}

// rwCanTake: a plain guest who can become a worker without breaking anything.
func (g *Game) rwCanTake(gu *Guest) bool {
	if !lifeFree(gu) || gu.name != "" || gu.keep || gu.quirk != "" || gu.armed || gu.leader != nil || gu.partner != nil ||
		gu.famousOK || gu.vipCard || gu.cs == nil || g.inSelfie(gu) || g.inContest(gu) || g.pevTaken(gu) {
		return false
	}
	if starSlugs[guestSlug(gu)] {
		return false
	}
	if sp := gu.cs.Spec; sp != nil && sp.Name != "" && (quirkOf(sp.Name) != "" || isKidSlug(sp.Name) || presetByName(sp.Name) != nil) {
		return false // a kid, a famous one or a character with his own quirk (not set yet on the first frame)
	}
	switch gu.state {
	case gIdle, gGoto, gDance, gLounge:
		return gu.timer < 1000
	}
	return false
}

// rwStart dresses 3-4 guests as workers and puts them at their places (at the start of the party: nobody sees it).
func (g *Game) rwStart(run *rwRun) {
	run.started = true
	r := rwDice()
	spots := run.site.spots()
	n := 3 + r.Intn(2)
	h := Vec{run.site.hole.X, run.site.edge}
	var cand []*Guest
	for _, gu := range g.guests {
		if g.rwCanTake(gu) {
			cand = append(cand, gu)
		}
	}
	sortGuestsBy(cand, func(gu *Guest) float64 { return dist(gu.pos, h) })
	for i := 0; i < n && i < len(cand); i++ {
		gu := cand[i]
		spec := rwWorkerSpec(r)
		gu.cs = newCharSet(&spec)
		gu.quirk, gu.quirkInit = rwQuirk, true
		w := &rwWorker{m: pevMem{gu: gu}, spot: spots[i]}
		if g.time < 2 {
			gu.pos = spots[i]
			g.rwSetTask(run, w, r, i%2)
		} else if !g.pevSendTo(&w.m, spots[i]) {
			w.away, w.backAt = true, g.time+3
		} else {
			w.walkT0 = g.time
		}
		run.ws = append(run.ws, w)
	}
	run.chatT = g.time + 2 + r.Float64()*3
	if rwDebug() && !testing.Testing() && run.site.crane != nil { // captures: where to look
		L := run.site.load()
		fmt.Printf("roadworks: hole %.0f,%.0f load %.0f,%.0f crane %.0f,%.0f workers %d\n", run.site.hole.X, run.site.hole.Y, L.X, L.Y,
			run.site.crane.X, run.site.crane.Y, len(run.ws))
	}
}

func (s *rwSite) faceRoad() Dir {
	if s.sd < 0 {
		return DirDown
	}
	return DirUp
}

// rwSetTask: what he does now at his place (the 4th one is the foreman: he talks and points more than he drills).
func (g *Game) rwSetTask(run *rwRun, w *rwWorker, r *rand.Rand, task int) {
	gu := w.m.gu
	w.task = task
	switch task {
	case rwTaskDrill:
		w.m.pose(AnimIdle) // arms down on the handles; the shake is drawn (worksJitter)
		gu.dir = run.site.faceRoad()
		w.until = g.time + 5 + r.Float64()*6
	case rwTaskRest:
		w.m.hold(gIdle)
		gu.dir = []Dir{DirLeft, DirRight, run.site.faceRoad()}[r.Intn(3)]
		w.until = g.time + 2 + r.Float64()*3
	case rwTaskDrink:
		w.m.pose(AnimDrink)
		w.until = g.time + 2.2 + r.Float64()*1.4
	case rwTaskTalk:
		w.m.pose([]int{AnimTalk, AnimPoint, AnimTired}[r.Intn(3)])
		w.until = g.time + 2.5 + r.Float64()*2.5
	}
}

func (g *Game) rwNextTask(run *rwRun, w *rwWorker, i int, r *rand.Rand) {
	if i == 3 { // the foreman
		g.rwSetTask(run, w, r, []int{rwTaskTalk, rwTaskRest, rwTaskDrink, rwTaskTalk}[r.Intn(4)])
		return
	}
	switch w.task {
	case rwTaskDrill:
		if r.Float64() < 0.3 {
			g.rwSetTask(run, w, r, rwTaskDrink)
		} else {
			g.rwSetTask(run, w, r, rwTaskRest)
		}
	default:
		g.rwSetTask(run, w, r, rwTaskDrill)
	}
}

// rwFreeAgain: a worker who ran away is calm again and can go back to work.
func (g *Game) rwFreeAgain(gu *Guest) bool {
	if !lifeFree(gu) || gu.quirk != rwQuirk || gu.armed || gu.leader != nil || g.inSelfie(gu) || g.inContest(gu) || g.pevTaken(gu) {
		return false
	}
	switch gu.state {
	case gIdle, gGoto, gDance, gLounge:
		return gu.timer < 1000
	}
	return false
}

// updateWorks: the workers, the traffic by the works and the crane (game.go, right after the party events and
// BEFORE the guests: the held states are re-asserted here).
func (g *Game) updateWorks(dt float64) {
	if g.mode != ModePlay {
		return
	}
	run := g.rwActive()
	if run == nil {
		return
	}
	r := rwDice()
	if !run.started {
		g.rwStart(run)
	}
	g.rwStageDebug(run)
	for i, w := range run.ws {
		if w.gone {
			continue
		}
		gu := w.m.gu
		if gu.out() {
			w.gone = true
			if gu.dead && !w.mourned {
				w.mourned = true
				g.rwMourn(run, w, r)
			}
			continue
		}
		if gu.quirk != rwQuirk { // made a star, a kid... by another system: he is not a worker any more
			w.gone = true
			pevForget(&w.m)
			continue
		}
		if w.away {
			if g.time >= w.backAt && g.rwFreeAgain(gu) {
				if g.pevSendTo(&w.m, w.spot) {
					w.away, w.walkT0 = false, g.time
					if r.Float64() < 0.5 && gu.icon == "" {
						gu.icon, gu.iconT = pevPick(r, rwBack), 2
					}
				} else {
					w.backAt = g.time + 4
				}
			}
			continue
		}
		if !w.m.mine() {
			pevForget(&w.m)
			w.away, w.backAt = true, g.time+8+r.Float64()*8
			continue
		}
		if w.m.want == gGoto {
			if gu.state == gIdle && gu.timer > 1000 || dist(gu.pos, w.spot) < 3 {
				g.rwSetTask(run, w, r, rwTaskDrill)
			} else if g.time-w.walkT0 > rwWalkMax {
				pevRelease(&w.m, r)
				w.away, w.backAt = true, g.time+6+r.Float64()*6
			}
			continue
		}
		w.m.keep()
		if w.task == rwTaskDrill {
			gu.dir = run.site.faceRoad()
			if g.time >= w.dustT && len(g.parts) < 500 {
				w.dustT = g.time + 0.09
				bit := rwBit(run.site, gu)
				l := 0.35 + r.Float64()*0.35
				g.parts = append(g.parts, Particle{pos: bit, vel: Vec{(r.Float64() - 0.5) * 30, -10 - r.Float64()*18}, life: l, max: l,
					col: color.RGBA{0xb4, 0xa4, 0x86, 255}, size: 2, grav: 30})
			}
		}
		if g.time >= w.until {
			g.rwNextTask(run, w, i, r)
		}
	}
	// now and then one of them shouts (a warning when a player is close)
	if g.time >= run.chatT {
		run.chatT = g.time + 6 + r.Float64()*8
		var pool []*rwWorker
		for _, w := range run.ws {
			if !w.gone && !w.away && w.m.mine() && w.m.gu.icon == "" {
				pool = append(pool, w)
			}
		}
		if len(pool) > 0 {
			w := pool[r.Intn(len(pool))]
			lines := rwChat
			if !g.pevPlayerFar(Vec{run.site.hole.X, run.site.edge}, 90) {
				lines = rwWarn
			}
			w.m.gu.icon, w.m.gu.iconT = pevPick(r, lines), 2.2
		}
	}
	g.rwTraffic(run)
	g.craneUpdate(run, dt)
}

// rwMourn: a worker was killed: the others near him take fright (the usual alarm of a guest) and come back later.
func (g *Game) rwMourn(run *rwRun, dead *rwWorker, r *rand.Rand) {
	said := false
	for _, w := range run.ws {
		if w == dead || w.gone || w.m.gu.out() {
			continue
		}
		gu := w.m.gu
		if w.m.mine() {
			pevRelease(&w.m, r)
		}
		w.away, w.backAt = true, g.time+14+r.Float64()*10
		if dist(gu.pos, dead.m.gu.pos) < 170 {
			txt := "!?"
			if !said {
				txt, said = pevPick(r, rwMourn), true
			}
			g.alarm(gu, txt)
		}
	}
}

// rwBit: where the jackhammer hits the ground.
func rwBit(s *rwSite, gu *Guest) Vec {
	return Vec{gu.pos.X + 3, gu.pos.Y - s.sd*4}
}

// rwIsWorker: a guest dressed as a worker of the current works.
func rwIsWorker(gu *Guest) bool { return gu != nil && gu.quirk == rwQuirk }

// rwDrilling: he is at his place drilling now.
func (run *rwRun) drilling(gu *Guest) bool {
	for _, w := range run.ws {
		if w.m.gu == gu {
			return !w.gone && !w.away && w.m.want == gFun && w.task == rwTaskDrill && w.m.mine()
		}
	}
	return false
}

// worksJitter: the shake of a drilling worker (pevGuestDraw, pe_fx.go): 1 px up and down, whole pixels.
func (g *Game) worksJitter(gu *Guest) float64 {
	if rw == nil || gu.quirk != rwQuirk || rw.lvl != g.lvl || !rw.drilling(gu) {
		return 0
	}
	return float64(int(g.time*26+gu.off*7) % 2)
}

// rwTraffic: the cars of the lane by the works move aside while they pass; none drives into a worker.
func (g *Game) rwTraffic(run *rwRun) {
	s := run.site
	in := map[*Ambient]bool{}
	for _, a := range g.amb {
		in[a] = true
		if a.dead || !roadKind(a.kind) || a.v0 == 0 {
			continue
		}
		base, held := run.swerve[a]
		if !held {
			base = a.y
		}
		if math.Abs(base-s.lane) > 4 {
			continue
		}
		dx := math.Abs(a.x - s.hole.X)
		f := 0.0
		switch {
		case dx < rwSwerveIn:
			f = 1
		case dx < rwSwerveTo:
			f = 1 - (dx-rwSwerveIn)/(rwSwerveTo-rwSwerveIn)
		}
		if f <= 0 {
			if held {
				a.y = base
				delete(run.swerve, a)
			}
			continue
		}
		run.swerve[a] = base
		a.y = base - s.sd*math.Round(rwSwerve*f)
	}
	g.rwGuard(run)
	for a := range run.swerve {
		if !in[a] {
			delete(run.swerve, a)
		}
	}
	if rwDebug() && g.ambT > 2.5 && len(run.swerve) == 0 {
		g.ambT = 2.5 // captures: more cars by the works
	}
}

// rwGuard: a worker in the way of a vehicle (he ran into the road): the car waits for him and, if it is too close or it
// is one that stops for nobody (the tank, the monster truck...), he jumps back onto the pavement.
func (g *Game) rwGuard(run *rwRun) {
	s := run.site
	for _, w := range run.ws {
		gu := w.m.gu
		if w.gone || gu.out() || gu.lift > 0 {
			continue
		}
		for _, a := range g.amb {
			if a.dead || !(roadKind(a.kind) || a.kind == ambTow) || a.v0 == 0 || math.Abs(gu.pos.Y-a.y) >= 15 {
				continue
			}
			dir := math.Copysign(1, a.v0)
			ahead := (gu.pos.X - a.x) * dir
			if ahead < -a.length/2-8 || ahead > a.length/2+70 {
				continue
			}
			a.wait = math.Max(a.wait, 0.2)
			if ahead < a.length/2+34 || a.kind == ambTank || a.name == "monstertruck" || a.name == "delorean" {
				to := Vec{gu.pos.X, s.edge + s.sd*9}
				if !g.lvl.rwDry(to) {
					to = w.spot // his own place is always open
				}
				gu.pos = to
				gu.path = nil
				if gu.icon == "" {
					gu.icon, gu.iconT = rwDodge[0], 1.4
				}
			}
		}
	}
}

// ---- drawing -----------------------------------------------------------------------------------------------

// worksItems: the puddle, the water jet, the jackhammers and the crane's load (draw.go, with the other items).
func (g *Game) worksItems(dst *ebiten.Image, items []drawItem) []drawItem {
	run := g.rwActive()
	if run == nil {
		return items
	}
	s := run.site
	h := s.hole
	items = append(items, drawItem{h.Y - 22, func() { g.rwDrawPuddle(dst, s) }})
	items = append(items, drawItem{h.Y - 3, func() { g.rwDrawJet(dst, s) }})
	for _, w := range run.ws {
		gu := w.m.gu
		if !run.drilling(gu) {
			continue
		}
		items = append(items, drawItem{gu.pos.Y + 0.3 - (s.sd+1)*0.3, func() { g.rwDrawHammer(dst, s, gu) }})
	}
	return g.craneItems(dst, run, items)
}

func (g *Game) rwDrawPuddle(dst *ebiten.Image, s *rwSite) {
	cx, cy := g.wx(s.hole.X), g.wy(s.hole.Y-4)
	filledEllipse(dst, cx, cy, 27, 7, rgba(0x2a5a8a, 110))
	filledEllipse(dst, cx, cy, 20, 5, rgba(0x4a8ac0, 120))
	// rings where the drops land
	for i := 0; i < 3; i++ {
		ph := math.Mod(g.time*0.9+float64(i)*0.33, 1)
		rx := math.Round(3 + ph*9)
		x := cx + float64([]int{-12, 9, -2}[i])
		y := cy + float64([]int{-1, 2, 3}[i])
		a := uint8(150 * (1 - ph))
		fillRect(dst, x-rx, y, rx*2, 1, rgba(0xc8ecff, a))
	}
}

// rwDrawJet: the jet of water from the broken pipe and the drops falling round it (a function of the time).
func (g *Game) rwDrawJet(dst *ebiten.Image, s *rwSite) {
	t := g.time
	px, py := g.wx(s.hole.X+1), g.wy(s.hole.Y-10)
	hgt := math.Round(24 + 3*math.Sin(t*7) + 2*math.Sin(t*13))
	sway := math.Round(math.Sin(t * 2.3))
	water, core, spray := rgb(0x7cc4f4), rgb(0xe4f6ff), rgb(0xb4e2ff)
	fillRect(dst, px-1+sway, py-hgt, 3, hgt, water)
	fillRect(dst, px+sway, py-hgt, 1, hgt, core)
	// the crown of the jet
	for i := 0; i < 6; i++ {
		a := float64(i)/5*math.Pi + math.Sin(t*5+float64(i))*0.2
		fillRect(dst, px+sway+math.Round(math.Cos(a)*4), py-hgt+math.Round(-math.Sin(a)*2), 1, 1, spray)
	}
	// drops falling on both sides
	for i := 0; i < 12; i++ {
		ph := math.Mod(t*1.25+float64(i)*0.083, 1)
		side := float64(i%2*2 - 1)
		vx := side * (8 + float64(i*5%9))
		x := px + sway + math.Round(vx*ph*1.4)
		y := py - hgt + math.Round(-6*ph+(hgt+6)*ph*ph*1.2)
		if y > py+4 {
			continue
		}
		fillRect(dst, x, y, 1, 2, water)
	}
}

// rwDrawHammer: the jackhammer in the hands of a drilling worker, shaking.
func (g *Game) rwDrawHammer(dst *ebiten.Image, s *rwSite, gu *Guest) {
	k := g.pevK()
	j := float64(int(g.time*30+gu.off*5) % 2)
	x := g.wx(gu.pos.X+3) + j - 1
	bottom := g.wy(gu.pos.Y - s.sd*4)
	top := bottom - float64(k(17))
	body, bodyD, handle := rgb(0x2c2c34), rgb(0x60606c), rgb(0xd8b020)
	fillRect(dst, x-float64(k(4)), top, float64(k(9)), float64(k(2)), handle)
	fillRect(dst, x-1, top+float64(k(2)), 3, float64(k(10)), body)
	fillRect(dst, x+1, top+float64(k(2)), 1, float64(k(10)), bodyD)
	fillRect(dst, x, top+float64(k(12)), 1, bottom-(top+float64(k(12))), rgb(0xb8c0cc))
}
