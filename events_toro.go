package main

import (
	"image/color"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// MECHANICAL BULL (idea of the user: "un toro mecánico en el que puedan montar y caerse los invitados y que haya una
// fila de máximo 4 personas esperando para montar. Que dure 20 segundos máximo cada uno y puedan ir más luego a la
// fila para montar").
//
// THE PROP "bullride": a mechanical bull on a steel pedestal in the middle of an inflatable mattress. Only the pedestal
// is solid (16x6); the mattress is flat floor where the riders land and the others walk. It is placed by
// Level.addBullRide (called by buildLevel, vip.go) only when loadLevel decided so (toroDecision, own dice): in SOME
// parties (toroChance, one in three) of the festival, the fair, the ranch, the beach and the birthday party; never in
// STEALTH, the RETO DE HOY, the island or the random mode. The spot is chosen like the diving board and the seats:
// dry open floor far from the dance floor, the VIP zone, the start and every place of use, nothing on the mattress,
// a line of four places beside it, and every floor cell reachable before is still reachable (else it is undone).
//
// THE RIDES (a reactive party event "toro", Minor: it never takes one of the two places of the frame): now and then
// (first session 20-60 s into the party, then a pause of toroPause after each session) a few guests form a LINE of at
// most toroLineMax beside the bull (a fifth one who comes says the line is too long and goes away). The first one
// walks to the bull, climbs on and holds on while it bucks (the prop is drawn with the frame of the buck) for 6-20 s
// (never more than toroRideMax); then he is thrown onto the mattress (a funny flight, he lies dizzy with stars,
// laughs and gets up) or he lasts the whole ride and climbs down clapping. Everybody steps forward. A rider may come
// back to the end of the line later (his own pause, toroCoolMin..Max). A few others watch in front and cheer.
//
// They are plain guests all the time (pevMem, like every crowd event): they can be killed anywhere (a dead rider or
// one of the line just leaves it and the line closes up), they are witnesses, they count for every cap and goal (nobody
// is added). With an alarm, the police or a fright near the bull the session ends at once and everybody is let go.
// Every phase has its own time limit. The player never rides it (no prompt). Dice: toroDice() (never g.rng, never
// the frame's dice for the prop). -pevent toro[:fall|:hold] forces the bull in the level given (progress frozen).
// ---------------------------------------------------------------------------

const (
	bullName     = "bullride"
	toroChance   = 1.0 / 3
	toroLineMax  = 4
	toroRideMin  = 6.0
	toroRideMax  = 20.0
	toroFallProb = 0.7
	toroSlot     = 13.0 // pixels between two in the line
	toroMountT   = 0.6
	toroFlyT     = 0.6
	toroLieT     = 1.6
	toroUpT      = 1.3
	toroWinT     = 1.2
	toroWalkMax  = 14.0 // seconds to reach a place of the line or the bull, else he is let go
	toroSit      = 11.0 // how high the rider is drawn over his ground point
)

// The times of the sessions (vars: the tests shorten them).
var (
	toroSessMin  = 70.0
	toroSessVar  = 40.0
	toroPauseMin = 45.0
	toroPauseVar = 45.0
	toroCoolMin  = 25.0
	toroCoolVar  = 25.0
)

// toroIcons: the parties where a mechanical bull fits.
var toroIcons = map[string]bool{"festival": true, "funfair": true, "beach": true, "cake": true}

var (
	toroJoin   = pevTexts("¡Me toca el toro!", "¡Yo también monto!", "¡A la cola del toro!", "¡Esta vez aguanto!")
	toroFull   = pevTexts("¡Uf, qué cola! Paso", "Con cuatro ya va lleno")
	toroCheer  = pevTexts("¡Yihaa!", "¡Que se cae!", "¡Más rápido!", "¡Agárrate fuerte!", "¡Aguanta, aguanta!", "¡Dale caña al toro!")
	toroRiderL = pevTexts("¡Yihaaa!", "¡Socorro!", "¡Esto no para!", "¡Soy un vaquero!")
	toroFallL  = pevTexts("¡Ay, mi espalda!", "¡JAJAJA!", "¡Otra vez!", "¡He volado!")
	toroWin    = pevTexts("¡Lo he domado!", "¡Ni me he despeinado!", "¡Bravo!")
)

// ---- the decision and the dice --------------------------------------------------------------------------

var (
	toroRng      *rand.Rand
	toroSeed     int64 // tests: non-zero = deterministic dice
	toroDecision bool  // loadLevel sets it just before buildLevel (like vipDecision) and clears it after
	toroOff      bool  // tests: no bull anywhere
	toroForce    bool  // tests: the bull for sure where it fits
	toroInTests  bool  // tests: roll the real dice under `go test` (off by default so other tests' parties stay bare)
)

func toroDice() *rand.Rand {
	if toroRng == nil {
		sd := toroSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x70b0c0de
		}
		toroRng = rand.New(rand.NewSource(sd))
	}
	return toroRng
}

// toroDebug: -pevent toro (or toro:fall / toro:hold).
func toroDebug() bool { return dbgPartyEvent == "toro" || strings.HasPrefix(dbgPartyEvent, "toro:") }

// toroFits: the level is one of the toro venues (campaign icon) and not a special mode.
func toroFits(def *LevelDef) bool {
	return def != nil && def.Title != "" && toroIcons[def.Icon] && len(def.Roam) > 0 && (def.PropScale == 0 || def.PropScale == 1)
}

// toroRoll: does this party get a mechanical bull? (loadLevel; the dice are always drawn).
func (g *Game) toroRoll(def *LevelDef) bool {
	roll := toroDice().Float64()
	if toroOff || def == nil || def.Title == "" || g.stealth || g.levelIdx == specialJurassic {
		return false
	}
	if toroDebug() {
		return len(def.Roam) > 0
	}
	if g.dc != nil || !toroFits(def) {
		return false
	}
	if toroForce {
		return true
	}
	if testing.Testing() && !toroInTests {
		return false
	}
	return roll < toroChance
}

// ---- the prop ---------------------------------------------------------------------------------------------

const (
	bullW    = 64
	bullH    = 44
	bullPedY = 10.0 // the bottom of the pedestal is this far above the anchor (the front edge of the mattress)
)

func init() {
	propKinds[bullName] = &PropKind{W: bullW, H: bullH, FW: 16, FH: 6, hasFoot: true, FootDX: -8, FootDY: -bullPedY - 6,
		SortOff: -bullPedY, Vars: 5, Category: "furniture", Draw: drawBullRide}
	registerPartyEvent(&partyEvent{ID: "toro", Name: "Toro mecánico", Group: "toro", Reactive: true, Minor: true,
		Prob: toroChance, AtMin: 20, AtMax: 60, Dur: 120,
		Eligible: toroDue, Start: toroStart, Items: toroItems,
		Update: pevGuard(toroUpdate, func(g *Game, a *partyAct) { toroEnd(g, a.D.(*toroState)) })})
}

// drawBullRide: v 0 at rest, 1 the head up, 2 the head down, 3 twisted left, 4 twisted right.
func drawBullRide(b *pixBuf, v int) {
	// the mattress: an inflatable pad with a red rim, a blue top with panels and a darker front side
	rim, rimD := c(0xd23a3a), c(0x8e2026)
	top, topL, side := c(0x3c6cd2), c(0x6a92e6), c(0x24448e)
	b.rect(3, 20, 58, 19, rim)
	b.rect(1, 22, 62, 15, rim)
	b.rect(5, 22, 54, 15, top)
	b.rect(3, 24, 58, 11, top)
	for x := 12; x < 56; x += 11 {
		b.rect(x, 23, 1, 13, topL)
	}
	b.rect(5, 22, 54, 1, topL)
	b.rect(3, 39, 58, 3, side)
	b.rect(1, 37, 62, 3, rimD)
	b.rect(3, 42, 58, 1, rimD)
	// the pedestal: a steel drum with a pole up to the belly
	steel, steelD, steelL := c(0xb8c0cc), c(0x5c6472), c(0xeef2f8)
	b.rect(24, 28, 16, 6, steelD)
	b.rect(24, 27, 16, 2, steel)
	b.rect(25, 27, 6, 1, steelL)
	b.rect(30, 20, 4, 8, steel)
	b.rect(30, 20, 1, 8, steelL)
	// the bull: a brown barrel body that tilts with the buck
	tilt, dx := 0.0, 0
	switch v {
	case 1:
		tilt = -2.5
	case 2:
		tilt = 2.5
	case 3:
		dx = -2
	case 4:
		dx = 2
	}
	fur, furD, belly := c(0x7a4a2a), c(0x4e2c16), c(0xa87048)
	col := func(x, y0, h int, cc color.RGBA) {
		off := int(math.Round(tilt * float64(x-32) / 14))
		b.rect(x+dx, y0+off, 1, h, cc)
	}
	for x := 17; x <= 44; x++ {
		y0, h := 10, 11
		if x < 19 || x > 42 {
			y0, h = 12, 7
		}
		col(x, y0, h, fur)
		col(x, y0+h-2, 2, furD)
		if x > 22 && x < 40 {
			col(x, y0+h-4, 2, belly)
		}
		col(x, y0, 1, belly)
	}
	// the saddle: a red pad with a white handle
	for x := 27; x <= 35; x++ {
		col(x, 8, 3, c(0xc42828))
	}
	col(33, 5, 3, c(0xf4f4f4))
	col(34, 5, 1, c(0xf4f4f4))
	// the head (right) with the horns and the nose ring
	for x := 44; x <= 52; x++ {
		y0, h := 8, 9
		if x > 50 {
			y0, h = 10, 6
		}
		col(x, y0, h, fur)
	}
	col(51, 13, 2, c(0xe8b098))
	col(52, 13, 2, c(0xe8b098))
	col(49, 15, 2, c(0xf0d040))
	col(47, 10, 1, c(0x101010))
	for _, hx := range []int{45, 50} {
		col(hx, 5, 3, c(0xf2ead8))
		col(hx-1, 4, 1, c(0xf2ead8))
		col(hx+1, 4, 1, c(0xf2ead8))
	}
	col(44, 9, 1, furD)
	// the tail (left)
	col(15, 11, 4, furD)
	col(14, 14, 3, c(0x2a1a10))
}

// addBullRide puts the mechanical bull in a built level when loadLevel decided so. true if it placed one.
func (l *Level) addBullRide() bool {
	d := l.Def
	if !toroDecision || toroOff || vipBuilding || d == nil || d.Title == "" || l.cw == 0 || l.bullRide() != nil {
		return false
	}
	k := propKinds[bullName]
	fitAllFootprints()
	r := rand.New(rand.NewSource(toroDice().Int63()))
	var cand []Vec
	for _, rc := range d.Roam {
		for y := rc.Y + 24; y < rc.Y+rc.H-8; y += 12 {
			for x := rc.X + 36; x < rc.X+rc.W-36; x += 16 {
				cand = append(cand, Vec{math.Round(x), math.Round(y)})
			}
		}
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	reach0 := l.reachFrom(l.NearestFree(d.Start))
	var avoid []Vec
	for _, ps := range l.POIs {
		avoid = append(avoid, ps...)
	}
	for _, pf := range d.Performers {
		avoid = append(avoid, pf.Pos)
	}
	tries := 0
	for _, p := range cand {
		if !l.bullSpotOK(p, k, avoid, reach0) {
			continue
		}
		if tries++; tries > 80 {
			break
		}
		if l.tryBullRide(p, k, reach0) {
			return true
		}
	}
	return false
}

// The places around a bull (world points).
func bullMount(pr *Prop) Vec { return Vec{pr.X, pr.Y - 2} }
func bullSeat(pr *Prop) Vec  { return Vec{pr.X, pr.Y - bullPedY - 3} }
func bullLand(pr *Prop, side float64) Vec {
	return Vec{pr.X + side*24, pr.Y - 6}
}
func bullCenter(pr *Prop) Vec { return Vec{pr.X, pr.Y - 12} }

// bullQueueDirs: where the line may form (its first place and the way it grows).
var bullQueueDirs = [][2]Vec{
	{{-40, -6}, {-1, 0}}, {{40, -6}, {1, 0}}, {{-30, 10}, {-0.7, 0.7}}, {{30, 10}, {0.7, 0.7}}, {{-26, 18}, {0, 1}}, {{26, 18}, {0, 1}},
}

func bullSlot(pr *Prop, qd int, i int) Vec {
	q := bullQueueDirs[qd]
	return Vec{math.Round(pr.X + q[0].X + q[1].X*toroSlot*float64(i)), math.Round(pr.Y + q[0].Y + q[1].Y*toroSlot*float64(i))}
}

func (l *Level) bullDry(q Vec) bool {
	return l.Free(q) && !l.SolidP(q.X, q.Y) && !l.badGround(q) && !l.InPool(q)
}

// bullSpotOK: the quick checks before building anything.
func (l *Level) bullSpotOK(p Vec, k *PropKind, avoid []Vec, reach0 []bool) bool {
	d := l.Def
	if p.X < 60 || p.X > l.W-60 || p.Y < 70 || p.Y > l.H-40 || dist(p, d.Start) < 110 {
		return false
	}
	mat := Rect{p.X - 34, p.Y - 26, 68, 34}
	for y := mat.Y; y <= mat.Y+mat.H; y += navCell / 2 {
		for x := mat.X; x <= mat.X+mat.W; x += navCell / 2 {
			if !l.bullDry(Vec{x, y}) {
				return false
			}
		}
	}
	for _, dr := range d.Dance {
		if dr.Grow(30).Has(p.X, p.Y) || dr.Grow(10).Hits(mat) {
			return false
		}
	}
	if z := l.vip; z != nil {
		for _, rc := range []Rect{z.in, z.trig, z.gate} {
			if rc.Grow(30).Hits(mat) {
				return false
			}
		}
	}
	for _, a := range avoid {
		if mat.Grow(30).Has(a.X, a.Y) {
			return false
		}
	}
	// a margin all round: no other drawing on or touching the mattress, nothing standing in the bull's drawing
	spr := Rect{p.X - bullW/2, p.Y - bullH, bullW, bullH}
	for _, o := range l.Props {
		drawn := Rect{o.X - float64(o.K.W)/2, o.Y - float64(o.K.H), float64(o.K.W), float64(o.K.H)}
		if pbOverlap(o.base().Grow(2), mat) || pbOverlap(o.Foot.Grow(4), mat) || o.K.W >= 60 && o.Y > mat.Y+mat.H && pbOverlap(drawn, mat.Grow(4)) || pbOverlap(o.base(), spr) {
			return false
		}
	}
	return !l.propClashes(bullName, k, p, 0)
}

func (l *Level) tryBullRide(p Vec, k *PropKind, reach0 []bool) bool {
	nProps, nP, nN := len(l.Props), len(l.solidsP), len(l.solidsN)
	undo := func() {
		l.Props, l.solidsP, l.solidsN = l.Props[:nProps], l.solidsP[:nP], l.solidsN[:nN]
		l.initNav()
	}
	pr := l.addTableProp(bullName, k, p, 0)
	l.initNav()
	reach1 := l.reachFrom(l.NearestFree(l.Def.Start))
	ok := func(q Vec) bool { return l.bullDry(q) && reach1[l.navIdx(q)] }
	if !ok(bullMount(pr)) || !ok(bullLand(pr, -1)) || !ok(bullLand(pr, 1)) || l.bullQueueDir(pr, reach1) < 0 {
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

// bullQueueDir: the first way the line can form (all its places dry, reachable and in a straight line), -1 none.
func (l *Level) bullQueueDir(pr *Prop, reach []bool) int {
	for qd := range bullQueueDirs {
		good := true
		prev := bullSlot(pr, qd, 0)
		for i := 0; i < toroLineMax && good; i++ {
			q := bullSlot(pr, qd, i)
			good = l.bullDry(q) && (reach == nil || reach[l.navIdx(q)]) && l.clearLine(prev, q)
			prev = q
		}
		if good && l.clearLine(bullSlot(pr, qd, 0), bullMount(pr)) {
			return qd
		}
	}
	return -1
}

// bullRide: the mechanical bull of the level (nil = none).
func (l *Level) bullRide() *Prop {
	if l == nil {
		return nil
	}
	for _, pr := range l.Props {
		if pr.Name == bullName && !pr.gone {
			return pr
		}
	}
	return nil
}

// bullVar: the frame of the bull now (drawProp, draw.go): it bucks while somebody rides it.
func (g *Game) bullVar(pr *Prop) int {
	a := g.partyRunning("toro")
	if a == nil {
		return 0
	}
	s, ok := a.D.(*toroState)
	if !ok || s.rider == nil || s.prop != pr {
		return 0
	}
	switch s.rider.phase {
	case trRide:
		u := math.Min(1, s.rider.t/math.Max(1, s.rider.dur))
		rate := 3 + 5*u // it bucks faster and faster
		return 1 + int(s.rider.t*rate)%4
	case trFly:
		return 2
	}
	return 0
}

// ---- the sessions ----------------------------------------------------------------------------------------

var (
	toroLvl  *Level
	toroNext float64
	toroCool map[*Guest]float64 // game time until which a rider does not come back to the line
)

const (
	trWalk = iota // walking from the head of the line to the bull
	trMount
	trRide
	trFly
	trLie
	trUp
	trWin
	trClap
)

type toroRider struct {
	mem   *pevMem
	phase int
	t     float64 // seconds in the phase
	dur   float64 // length of the ride
	fall  bool
	side  float64
	said  float64
}

type toroState struct {
	prop    *Prop
	qd      int
	line    []*pevMem
	rider   *toroRider
	crowd   pevCrowd
	end     float64 // game time when no one else joins
	nextJ   float64
	nextM   float64
	say     float64
	fullT   float64
	rides   int
	falls   int
	joined  int
	rejoins int // riders who came back to the line
}

func (s *toroState) pevHas(gu *Guest) bool {
	return pevHasIn(s.line, gu) || s.rider != nil && s.rider.mem.gu == gu || s.crowd.pevHas(gu)
}

// toroAlarm: police, an alarm or a fright near the bull: no rides now.
func (g *Game) toroAlarm(pr *Prop) bool {
	if g.policeCalled || len(g.cops) > 0 {
		return true
	}
	c := bullCenter(pr)
	for _, gu := range g.guests {
		if gu.out() || dist(gu.pos, c) > 120 {
			continue
		}
		switch gu.state {
		case gAlarm, gFlee, gScared, gToPhone, gCall, gReport, gRun:
			return true
		}
	}
	return false
}

// toroDue: the condition of the reactive event (a bull in the level, its pause over, no alarm).
func toroDue(g *Game, t partyTraits) bool {
	if g.lvl == nil {
		return false
	}
	pr := g.lvl.bullRide()
	if pr == nil {
		return false
	}
	if toroLvl != g.lvl {
		toroLvl, toroCool = g.lvl, map[*Guest]float64{}
		toroNext = g.time + 20 + toroDice().Float64()*40
		if toroDebug() {
			toroNext = g.time + 3
		}
	}
	return g.time >= toroNext && !g.toroAlarm(pr)
}

// toroCanJoin: a free guest who may go to the line now.
func (g *Game) toroCanJoin(s *toroState, gu *Guest) bool {
	if !g.pevFreeGuest(gu) || s.pevHas(gu) || toroCool[gu] > g.time {
		return false
	}
	return dist(gu.pos, bullCenter(s.prop)) < 260
}

// toroJoinOne sends one more guest to the end of the line (a former rider first, if one is ready). true if one came.
func (g *Game) toroJoinOne(s *toroState, r *rand.Rand) bool {
	if len(s.line) >= toroLineMax {
		return false
	}
	c := bullCenter(s.prop)
	var best, back *Guest
	for _, gu := range g.guests {
		if !g.toroCanJoin(s, gu) {
			continue
		}
		if _, rode := toroCool[gu]; rode && (back == nil || dist(gu.pos, c) < dist(back.pos, c)) {
			back = gu
		}
		if best == nil || dist(gu.pos, c) < dist(best.pos, c) {
			best = gu
		}
	}
	if back != nil && r.Float64() < 0.7 {
		best = back
	}
	if best == nil {
		return false
	}
	m := &pevMem{gu: best, pt: bullSlot(s.prop, s.qd, len(s.line)), next: g.time}
	if !g.pevSendTo(m, m.pt) {
		return false
	}
	if _, rode := toroCool[best]; rode {
		s.rejoins++
	}
	s.line = append(s.line, m)
	s.joined++
	if best.icon == "" {
		best.icon, best.iconT = pevPick(r, toroJoin), 1.8
	}
	return true
}

func toroStart(g *Game, a *partyAct) bool {
	r := toroDice()
	pr := g.lvl.bullRide()
	if pr == nil {
		return false
	}
	toroNext = g.time + 6 // if it cannot start now, the condition is asked again later, not every frame
	qd := g.lvl.bullQueueDir(pr, nil)
	if qd < 0 {
		return false
	}
	s := &toroState{prop: pr, qd: qd, end: g.time + toroSessMin + r.Float64()*toroSessVar, nextJ: g.time + 4, nextM: g.time + 3}
	n := 2 + r.Intn(toroLineMax-1)
	for len(s.line) < n && g.toroJoinOne(s, r) {
	}
	if len(s.line) < 2 {
		pevReleaseAll(s.line, r)
		return false
	}
	s.crowd.gather(g, r, bullCenter(pr), 2+r.Intn(2), 36, 48, func(gu *Guest) bool { return s.pevHas(gu) })
	a.D = s
	return true
}

func toroEnd(g *Game, s *toroState) {
	r := toroDice()
	pevReleaseAll(s.line, r)
	s.line = nil
	if s.rider != nil {
		pevRelease(s.rider.mem, r)
		toroCool[s.rider.mem.gu] = g.time + toroCoolMin
		s.rider = nil
	}
	s.crowd.release(g, r)
	toroNext = g.time + toroPauseMin + r.Float64()*toroPauseVar
}

func toroUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*toroState)
	r := toroDice()
	if s.prop.gone || g.toroAlarm(s.prop) {
		toroEnd(g, s)
		return true
	}
	// the ones who are not ours any more (killed, frightened) leave the line; it closes up
	kept := s.line[:0]
	for _, m := range s.line {
		if m.mine() && !(m.gu.state == gGoto && g.time-m.next > toroWalkMax) {
			kept = append(kept, m)
		} else if m.mine() {
			pevRelease(m, r)
		} else {
			pevForget(m)
		}
	}
	s.line = kept
	s.crowd.update(g, r, nil, 0, nil)
	g.toroRide(s, r, dt)
	// the line: everybody at his place, facing the bull, now and then a shout
	c := bullCenter(s.prop)
	for i, m := range s.line {
		m.pt = bullSlot(s.prop, s.qd, i)
		gu := m.gu
		if gu.state == gGoto {
			continue
		}
		if m.want == gGoto {
			m.want = gIdle
		}
		if dist(gu.pos, m.pt) > 4 {
			if g.pevSendTo(m, m.pt) {
				m.next = g.time
			}
			continue
		}
		m.keep()
		gu.dir = faceTo(gu.pos, c)
	}
	// the head of the line goes to the bull when it is free
	if s.rider == nil && len(s.line) > 0 && g.time >= s.nextM {
		m := s.line[0]
		if m.gu.state == gIdle && m.gu.timer > 1000 && dist(m.gu.pos, m.pt) < 5 {
			s.line = s.line[1:]
			if g.pevSendTo(m, bullMount(s.prop)) {
				s.rider = &toroRider{mem: m, phase: trWalk, side: float64(r.Intn(2)*2 - 1)}
			} else {
				pevRelease(m, r)
			}
		}
	}
	// more come while the session lasts; a fifth one sees the line is full and goes away
	if g.time < s.end && g.time >= s.nextJ {
		s.nextJ = g.time + 3 + r.Float64()*4
		if len(s.line) < toroLineMax {
			g.toroJoinOne(s, r)
		} else if g.time >= s.fullT {
			for _, gu := range g.guests {
				if g.toroCanJoin(s, gu) && dist(gu.pos, c) < 140 && gu.icon == "" {
					gu.icon, gu.iconT = pevPick(r, toroFull), 2
					s.fullT = g.time + 15
					break
				}
			}
		}
	}
	// the cheering
	if s.rider != nil && s.rider.phase == trRide && g.time >= s.say {
		s.say = g.time + 1.6 + r.Float64()*1.6
		var who []*Guest
		for _, m := range s.line {
			who = append(who, m.gu)
		}
		for _, m := range s.crowd.mem {
			who = append(who, m.gu)
		}
		if len(who) > 0 {
			if gu := who[r.Intn(len(who))]; gu.icon == "" {
				gu.icon, gu.iconT = pevPick(r, toroCheer), 1.6
			}
		}
	}
	if g.time >= s.end && len(s.line) == 0 && s.rider == nil {
		toroEnd(g, s)
		return true
	}
	if g.time > s.end+60 { // safety: a session never lasts much longer than planned
		toroEnd(g, s)
		return true
	}
	return false
}

// toroRide: the rider's phases.
func (g *Game) toroRide(s *toroState, r *rand.Rand, dt float64) {
	rd := s.rider
	if rd == nil {
		return
	}
	m := rd.mem
	if !m.mine() {
		pevForget(m)
		if !m.gu.out() {
			toroCool[m.gu] = g.time + toroCoolMin
		}
		s.rider, s.nextM = nil, g.time+1.5
		return
	}
	gu := m.gu
	rd.t += dt
	next := func(ph int) { rd.phase, rd.t = ph, 0 }
	switch rd.phase {
	case trWalk:
		if gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, bullMount(s.prop)) < 5 {
			m.hold(gIdle)
			gu.dir = DirRight
			next(trMount)
		} else if rd.t > toroWalkMax {
			pevRelease(m, r)
			s.rider, s.nextM = nil, g.time+1
		}
	case trMount:
		m.keep()
		if rd.t >= toroMountT {
			rd.dur = toroRideMin + r.Float64()*(toroRideMax-toroRideMin)
			rd.fall = r.Float64() < toroFallProb
			switch dbgPartyEvent {
			case "toro:fall":
				rd.dur, rd.fall = 3, true
			case "toro:hold":
				rd.dur, rd.fall = 5, false
			}
			next(trRide)
			s.rides++
			if g.toroNear(s.prop) {
				g.snd.toroPlay("toro")
			}
		}
	case trRide:
		m.keep()
		if rd.t-rd.said > 3 && gu.icon == "" && r.Float64() < dt*0.8 {
			rd.said = rd.t
			gu.icon, gu.iconT = pevPick(r, toroRiderL), 1.5
		}
		if rd.t >= math.Min(rd.dur, toroRideMax) {
			if rd.fall {
				gu.pos = bullLand(s.prop, rd.side) // he lands there (drawn flying from the saddle)
				next(trFly)
			} else {
				next(trWin)
			}
		}
	case trFly:
		m.keep()
		if rd.t >= toroFlyT {
			s.falls++
			next(trLie)
			if g.toroNear(s.prop) {
				g.snd.Play("thud")
			}
			g.toroShout(s, r, toroCheer[1:2])
		}
	case trLie:
		m.keep()
		if rd.t >= toroLieT {
			m.pose(AnimLaugh)
			gu.icon, gu.iconT = pevPick(r, toroFallL), 1.8
			next(trUp)
		}
	case trUp:
		m.keep()
		if rd.t >= toroUpT {
			g.toroDone(s, r)
		}
	case trWin:
		m.keep()
		if rd.t >= toroWinT {
			m.pose(AnimCheer)
			gu.icon, gu.iconT = pevPick(r, toroWin), 1.8
			g.toroShout(s, r, toroWin[2:3])
			next(trClap)
		}
	case trClap:
		m.keep()
		if rd.t >= toroUpT {
			g.toroDone(s, r)
		}
	}
}

// toroDone: the rider goes back to the party; he may come back to the line after his pause.
func (g *Game) toroDone(s *toroState, r *rand.Rand) {
	m := s.rider.mem
	pevRelease(m, r)
	toroCool[m.gu] = g.time + toroCoolMin + r.Float64()*toroCoolVar
	s.rider, s.nextM = nil, g.time+0.8
}

// toroShout: one of the watchers or of the line says something.
func (g *Game) toroShout(s *toroState, r *rand.Rand, lines []string) {
	ms := append(append([]*pevMem(nil), s.crowd.mem...), s.line...)
	if len(ms) == 0 {
		return
	}
	if gu := ms[r.Intn(len(ms))].gu; gu.icon == "" {
		gu.icon, gu.iconT = pevPick(r, lines), 1.6
	}
}

func (g *Game) toroNear(pr *Prop) bool { return !g.pevPlayerFar(bullCenter(pr), 220) }

// ---- drawing -----------------------------------------------------------------------------------------------

// drawGuest: the rider climbing, on the bull, in the air or lying on the mattress (the frame's hook, pe_fx.go).
func (s *toroState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	rd := s.rider
	if rd == nil || rd.mem.gu != gu || !rd.mem.mine() {
		return false
	}
	sc := g.lifeScale()
	if sc <= 0 {
		sc = 1
	}
	pr := s.prop
	seatX := pr.X
	switch g.bullVar(pr) {
	case 3:
		seatX -= 2
	case 4:
		seatX += 2
	}
	at, lift := bullMount(pr), toroSit
	anim, phase, dir := AnimCheer, g.time*5, DirRight
	switch rd.phase {
	case trWalk, trUp, trClap:
		return false
	case trMount:
		u := math.Min(1, rd.t/toroMountT)
		lift = toroSit * u
		anim, phase = AnimWalk, rd.t*8
	case trRide:
		at.X = seatX
		switch g.bullVar(pr) {
		case 1:
			lift += 2
		case 2:
			lift -= 1
		}
		if int(rd.t*1.3)%2 == 1 {
			anim = AnimPanic
		}
	case trWin:
		at.X = seatX
		anim, phase = AnimWave, rd.t*6
	case trFly:
		u := math.Min(1, rd.t/toroFlyT)
		land := bullLand(pr, rd.side)
		from := bullMount(pr)
		at = Vec{from.X + (land.X-from.X)*u, from.Y + (land.Y-from.Y)*u}
		lift = toroSit*(1-u) + 22*4*u*(1-u)
		anim, phase = AnimFall, u*3
		if rd.side < 0 {
			dir = DirLeft
		}
	case trLie:
		g.drawLying(dst, gu.cs, false, Vec{math.Round(gu.pos.X), math.Round(gu.pos.Y)})
		cx, cy := g.pevXY(gu.pos)
		drawStarsPix(g.plotter(dst), cx, cy-g.pevK()(12), g.pevPx(), g.time)
		return true
	}
	g.drawActor(dst, gu.cs, anim, phase, Vec{math.Round(at.X), math.Round(at.Y)}, dir, -math.Round(lift)/sc, false)
	return true
}

// toroItems: the watchers clap while somebody rides.
func toroItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s, ok := a.D.(*toroState)
	if !ok || s.rider == nil || s.rider.phase != trRide {
		return items
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for _, m := range s.crowd.mem {
		gu := m.gu
		if gu.state != gIdle || gu.timer < 1000 {
			continue
		}
		items = append(items, drawItem{gu.pos.Y + 0.3, func() {
			cx, cy := g.pevXY(gu.pos)
			drawClapPix(plot, cx, cy-k(34), px, g.time, int(math.Abs(gu.pos.X)))
		}})
	}
	return items
}

// ---- sound (synthetic, quiet, once per ride) ---------------------------------------------------------------

// toroPlay: the own short sounds of the bull and the street band, built the first time they are needed.
func (s *Sounds) toroPlay(name string) {
	if s == nil || s.sfx == nil {
		return
	}
	if _, ok := s.sfx[name]; !ok {
		switch name {
		case "toro": // the motor starts: a low hum that rises, with a clank
			s.sfx[name] = mix(tone(70, 0.5, 0.25, 3, 1), shift(tone(110, 0.35, 0.18, 5, 1), 0.15), lowpass(noise(0.08, 0.25, 30), 3))
		case "murga": // the bucket drum and the pot: bom, bom, clink
			s.sfx[name] = mix(tone(62, 0.22, 0.32, 14, 0), shift(tone(62, 0.22, 0.28, 14, 0), 0.32),
				shift(noise(0.06, 0.12, 40), 0.62), shift(tone(1600, 0.08, 0.06, 30, 0), 0.62))
		default:
			return
		}
	}
	s.Play(name)
}
