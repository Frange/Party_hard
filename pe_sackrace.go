package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE SACK RACE (idea 301, approved: "carrera de sacos, solo en sitios espaciosos").
//
// A reactive, Minor party event "sacos" (it never takes one of the places of the frame, so the mean of planned events
// per party does not move): in SOME parties (sackPartyChance) of the open venues (beach, bullring, Paris square, Italian piazza, fair,
// festival) and only where the level has a clear course of sackCourseW x sackCourseH px of walkable outdoor floor
// with nothing solid on it (sackCourses, measured on the guests' grid), 3-5 guests line up at a start line drawn on
// the ground, pull sacks up to their waists and race in hops toward a finish line with two poles, pennants and a tape;
// a few others watch at the finish and cheer. Someone always trips (lies dizzy a moment, gets up and goes on); the
// first one breaks the tape and celebrates, the others laugh or protest. The race lasts 10-20 s (30 s at most).
//
// They are plain guests all the time (pevMem): they can be killed (a dead runner stops the race: the others freeze
// and look at the body, then they are let go and the game's own witnesses react), they count for every cap; nobody is
// added. With an alarm or the police near the race is dissolved at once. Every phase has its own time limit. Dice:
// sackDice() (never g.rng, never the frame's dice). -pevent sacos forces it (progress frozen).
// ---------------------------------------------------------------------------

const (
	sackCourseW     = 208.0
	sackCourseH     = 80.0
	sackPartyChance = 0.5
	sackLane        = 13.0
	sackMargin      = 24.0 // start and finish lines this far inside the course
	sackGatherMax   = 18.0
	sackReadyT      = 2.4
	sackRaceMax     = 22.0
	sackEndT        = 3.2
	sackStopT       = 1.2
	sackFallT       = 1.4
	sackUpT         = 0.6
	sackGuestR      = 240.0
	sackAlarmR      = 150.0
)

// sackIcons: the open, spacious venues where a sack race makes sense.
var sackIcons = map[string]bool{"beach": true, "bull": true, "eiffel": true, "funfair": true, "festival": true, "italy": true}

var (
	sackJoin  = pevTexts("¡Carrera de sacos!", "¡Yo me apunto!", "¡Esta la gano yo!")
	sackReady = pevTexts("¡Preparados, listos, ya!")
	sackCheer = pevTexts("¡Salto, salto, salto!", "¡Vamos, vamos!", "¡Corre, saco de patatas!", "¡Que te pillan!")
	sackFallL = pevTexts("¡Ay, mi saco!", "¡Me he tropezado!", "¡Esto no vale!")
	sackLaugh = pevTexts("¡JAJAJA!", "¡Se ha comido el suelo!")
	sackWinL  = pevTexts("¡He ganado!", "¡Campeón del saco!", "¡Soy un canguro!")
	sackLoseL = pevTexts("¡Me ha empujado!", "¡Revancha!", "¡Casi!", "¡Mi saco tenía un agujero!")
	sackShock = pevTexts("¡¿Qué ha pasado?!", "¡Parad la carrera!")
)

var (
	sackRng     *rand.Rand
	sackSeed    int64 // tests: non-zero = deterministic dice
	sackInTests bool  // tests: let it happen under go test
)

func sackDice() *rand.Rand {
	if sackRng == nil {
		sd := sackSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x5ac4ace
		}
		sackRng = rand.New(rand.NewSource(sd))
	}
	return sackRng
}

// ---- the course ----------------------------------------------------------------------------------------------------

// sackCourses: every clear course of the level (top-left on the 8 px grid): walkable for the guests, outdoor, no road,
// no water, no solid foot. Empty = the level is not spacious enough.
func sackCourses(l *Level) []Rect {
	if l == nil || l.Def == nil || l.cw == 0 {
		return nil
	}
	cw, ch := int(sackCourseW/navCell), int(sackCourseH/navCell)
	good := make([]bool, l.cw*l.ch)
	for cy := 0; cy < l.ch; cy++ {
		for cx := 0; cx < l.cw; cx++ {
			if !l.cellFree(cx, cy) || l.blockedG != nil && l.blockedG[cy*l.cw+cx] {
				continue
			}
			p := cellCenter(cx, cy)
			good[cy*l.cw+cx] = l.Outdoor(p) && !l.badGround(p) && !l.InPool(p) && !l.SolidP(p.X, p.Y)
		}
	}
	// no prop on it either, not even a walkable one (a stage, a rug, a speaker without collision): nobody races across it
	for _, pr := range l.Props {
		f := sackPropBase(pr)
		if f.W <= 0 {
			continue
		}
		for cy := maxI(0, int(f.Y/navCell)); cy < l.ch && float64(cy)*navCell < f.Y+f.H; cy++ {
			for cx := maxI(0, int(f.X/navCell)); cx < l.cw && float64(cx)*navCell < f.X+f.W; cx++ {
				good[cy*l.cw+cx] = false
			}
		}
	}
	// nor on a dance floor (the dancers would be all over the runners)
	for _, f := range l.Def.Dance {
		f = f.Grow(8)
		for cy := maxI(0, int(f.Y/navCell)); cy < l.ch && float64(cy)*navCell < f.Y+f.H; cy++ {
			for cx := maxI(0, int(f.X/navCell)); cx < l.cw && float64(cx)*navCell < f.X+f.W; cx++ {
				good[cy*l.cw+cx] = false
			}
		}
	}
	// a summed table: a course is clear when its count of good cells is full
	sum := make([]int, (l.cw+1)*(l.ch+1))
	W := l.cw + 1
	for y := 0; y < l.ch; y++ {
		for x := 0; x < l.cw; x++ {
			v := 0
			if good[y*l.cw+x] {
				v = 1
			}
			sum[(y+1)*W+x+1] = v + sum[y*W+x+1] + sum[(y+1)*W+x] - sum[y*W+x]
		}
	}
	var out []Rect
	for y := 0; y+ch <= l.ch; y += 2 {
		for x := 0; x+cw <= l.cw; x += 2 {
			n := sum[(y+ch)*W+x+cw] - sum[y*W+x+cw] - sum[(y+ch)*W+x] + sum[y*W+x]
			if n == cw*ch {
				out = append(out, Rect{float64(x) * navCell, float64(y) * navCell, sackCourseW, sackCourseH})
			}
		}
	}
	return out
}

// sackPropBase: the ground a prop takes for the race: its foot, grown a little, and the lower half of its picture (a speaker
// standing just under the course must not poke into it).
func sackPropBase(pr *Prop) Rect {
	r := pr.Foot
	if pr.K != nil && pr.K.W > 0 && pr.K.H > 0 {
		w, h := float64(pr.K.W), float64(pr.K.H)
		b := Rect{pr.X - w/2, pr.Y - h/2, w, h / 2}
		if r.W <= 0 {
			r = b
		} else {
			x0, y0 := math.Min(r.X, b.X), math.Min(r.Y, b.Y)
			r = Rect{x0, y0, math.Max(r.X+r.W, b.X+b.W) - x0, math.Max(r.Y+r.H, b.Y+b.H) - y0}
		}
	}
	if r.W <= 0 {
		return r
	}
	return r.Grow(3)
}

type sackLevel struct {
	lvl     *Level
	courses []Rect
	want    bool
	races   int
	max     int
	next    float64
	tries   int
}

var sackLv sackLevel

func sackReset(g *Game) {
	r := sackDice()
	sackLv = sackLevel{lvl: g.lvl}
	if g.lvl.Def != nil && sackIcons[g.lvl.Def.Icon] {
		sackLv.courses = sackCourses(g.lvl)
	}
	sackLv.want = r.Float64() < sackPartyChance
	sackLv.max = 1
	if r.Float64() < 0.3 {
		sackLv.max = 2
	}
	sackLv.next = g.time + 60 + r.Float64()*150
	if pevDebugOn("sacos") {
		sackLv.want, sackLv.next, sackLv.max = true, g.time+2, 9
	}
}

// ---- the event -----------------------------------------------------------------------------------------------------

func init() {
	registerPartyEvent(&partyEvent{ID: "sacos", Name: "Carrera de sacos", Group: "sacos", Reactive: true, Minor: true,
		Prob: sackPartyChance, AtMin: 60, AtMax: 900, Dur: 30,
		Eligible: sackDue, Start: sackStart, Items: sackItems,
		Update: pevGuard(sackUpdate, func(g *Game, a *partyAct) { sackEnd(g, a.D.(*sackState)) })})
}

const (
	sakGather = iota
	sakReady
	sakRace
	sakEnd
	sakStop
)

type sackRunner struct {
	m       pevMem
	x, y    float64
	speed   float64
	hop     float64 // 0..1 phase of the hop
	period  float64
	fallAt  float64 // x distance run when he trips (< 0 = no planned fall)
	falls   int
	downT   float64 // > 0: lying (then getting up)
	done    bool
	place   int
	arrived bool
}

type sackState struct {
	course    Rect
	dir       float64
	startX    float64
	finishX   float64
	run       []*sackRunner
	fans      []*pevMem
	phase     int
	t         float64
	cheerNext float64
	winner    *sackRunner
	finished  int
	body      Vec
	// for the tests
	fellN int
}

func (s *sackState) pevHas(gu *Guest) bool {
	for _, r := range s.run {
		if r.m.gu == gu {
			return true
		}
	}
	return pevHasIn(s.fans, gu)
}

func (s *sackState) laneY(i, n int) float64 {
	return math.Round(s.course.Y + s.course.H/2 + (float64(i)-float64(n-1)/2)*sackLane)
}

func sackDue(g *Game, t partyTraits) bool {
	if g.lvl == nil {
		return false
	}
	if testing.Testing() && !sackInTests && !pevDebugOn("sacos") {
		return false
	}
	if sackLv.lvl != g.lvl {
		sackReset(g)
	}
	lv := &sackLv
	return lv.want && len(lv.courses) > 0 && lv.races < lv.max && lv.tries < 10 && g.time >= lv.next
}

func sackStart(g *Game, a *partyAct) bool {
	r := sackDice()
	lv := &sackLv
	lv.next = g.time + 8
	lv.tries++
	// a course away from the players (they would be in the way), tried in a random order
	var course Rect
	found := false
	for _, i := range r.Perm(len(lv.courses)) {
		c := lv.courses[i]
		mid := Vec{c.X + c.W/2, c.Y + c.H/2}
		if g.pevPlayerFar(mid, 60) && !g.sackAlarmNear(mid, sackAlarmR) {
			course, found = c, true
			break
		}
	}
	if !found {
		return false
	}
	s := &sackState{course: course, dir: 1}
	if r.Intn(2) == 0 {
		s.dir = -1
	}
	s.startX = course.X + sackMargin
	s.finishX = course.X + course.W - sackMargin
	if s.dir < 0 {
		s.startX, s.finishX = s.finishX, s.startX
	}
	mid := Vec{course.X + course.W/2, course.Y + course.H/2}
	rad := sackGuestR
	if pevDebugOn("sacos") {
		rad = 1e9
	}
	var cand []*Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && !isKid(gu) && dist(gu.pos, mid) < rad {
			cand = append(cand, gu)
		}
	}
	if len(cand) < 3 {
		return false
	}
	// the nearest ones run, the next ones watch
	for i := 1; i < len(cand); i++ {
		for j := i; j > 0 && dist(cand[j].pos, mid) < dist(cand[j-1].pos, mid); j-- {
			cand[j], cand[j-1] = cand[j-1], cand[j]
		}
	}
	n := 3 + r.Intn(3)
	if n > len(cand) {
		n = len(cand)
	}
	fall := r.Intn(n) // somebody always trips
	for i := 0; i < n; i++ {
		ru := &sackRunner{x: s.startX, y: s.laneY(i, n), speed: 16 + r.Float64()*7, period: 0.42 + r.Float64()*0.1, fallAt: -1}
		if i == fall {
			ru.fallAt = 40 + r.Float64()*100
		}
		ru.m.gu = cand[i]
		if !g.pevSendTo(&ru.m, Vec{ru.x, ru.y}) {
			continue
		}
		s.run = append(s.run, ru)
		if i == 0 && cand[i].icon == "" {
			cand[i].icon, cand[i].iconT = pevPick(r, sackJoin), 2
		}
	}
	if len(s.run) < 3 {
		sackRelease(s, r)
		return false
	}
	// the fans: at the finish, above and below the lanes
	top, bot := course.Y+8, course.Y+course.H-6
	for i := n; i < len(cand) && len(s.fans) < 4; i++ {
		j := len(s.fans)
		y := top
		if j%2 == 1 {
			y = bot
		}
		p := Vec{math.Round(s.finishX - s.dir*(8+float64(j/2)*20)), y}
		if !g.lvl.Free(p) || g.lvl.SolidP(p.X, p.Y) {
			continue
		}
		m := &pevMem{gu: cand[i], pt: p}
		if g.pevSendTo(m, p) {
			s.fans = append(s.fans, m)
		}
	}
	a.D = s
	lv.races++
	if pevDebugOn("sacos") {
		fmt.Printf("PEVDEMO sacos %d runners at %.0f %.0f\n", len(s.run), course.X, course.Y)
	}
	return true
}

func sackRelease(s *sackState, r *rand.Rand) {
	for _, ru := range s.run {
		if ru.m.mine() {
			pevRelease(&ru.m, r)
		} else {
			pevForget(&ru.m)
		}
	}
	pevReleaseAll(s.fans, r)
}

func sackEnd(g *Game, s *sackState) {
	if s == nil {
		return
	}
	r := sackDice()
	sackRelease(s, r)
	sackLv.next = g.time + 120 + r.Float64()*90
	if pevDebugOn("sacos") {
		sackLv.next = g.time + 5
	}
}

// sackHopZ: how high a runner is in the air (whole pixels).
func (ru *sackRunner) hopZ() float64 {
	if ru.downT > 0 {
		return 0
	}
	return math.Round(math.Sin(math.Pi*ru.hop) * 5)
}

func sackUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*sackState)
	r := sackDice()
	s.t += dt
	mid := Vec{s.course.X + s.course.W/2, s.course.Y + s.course.H/2}
	// a dead runner or fan stops everything: they freeze and look at the body, then they are let go
	if s.phase != sakStop {
		for _, ru := range s.run {
			if ru.m.gu.dead {
				s.phase, s.t, s.body = sakStop, 0, ru.m.gu.pos
			}
		}
		for _, m := range s.fans {
			if m.gu.dead {
				s.phase, s.t, s.body = sakStop, 0, m.gu.pos
			}
		}
		if s.phase == sakStop {
			said := false
			for _, ru := range s.run {
				if ru.m.mine() {
					ru.m.hold(gIdle)
					ru.downT = 0
					ru.m.gu.dir = faceTo(ru.m.gu.pos, s.body)
					if !said && ru.m.gu.icon == "" {
						ru.m.gu.icon, ru.m.gu.iconT, said = pevPick(r, sackShock), 1.6, true
					}
				}
			}
			for _, m := range s.fans {
				if m.mine() {
					m.gu.dir = faceTo(m.gu.pos, s.body)
				}
			}
			return false
		}
	}
	if s.phase == sakStop {
		for _, ru := range s.run {
			if ru.m.mine() {
				ru.m.keep()
			}
		}
		if s.t >= sackStopT {
			sackEnd(g, s)
			return true
		}
		return false
	}
	if g.sackAlarmNear(mid, sackAlarmR) {
		sackEnd(g, s)
		return true
	}
	// runners who are not ours any more (a fright) leave the race
	alive := s.run[:0]
	for _, ru := range s.run {
		if ru.m.mine() {
			alive = append(alive, ru)
		} else {
			pevForget(&ru.m)
		}
	}
	s.run = alive
	s.fans = pevPruneMems(s.fans)
	if len(s.run) < 2 && s.phase != sakEnd || len(s.run) == 0 {
		sackEnd(g, s)
		return true
	}
	switch s.phase {
	case sakGather:
		all := true
		for _, ru := range s.run {
			gu := ru.m.gu
			if !ru.arrived && gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, Vec{ru.x, ru.y}) < 6 {
				ru.arrived = true
				ru.m.hold(gIdle)
			}
			if !ru.arrived {
				all = false
			}
		}
		if all || s.t > sackGatherMax {
			kept := s.run[:0]
			for _, ru := range s.run {
				if ru.arrived {
					kept = append(kept, ru)
				} else {
					pevRelease(&ru.m, r)
				}
			}
			s.run = kept
			if len(s.run) < 3 && !(pevDebugOn("sacos") && len(s.run) >= 2) {
				sackEnd(g, s)
				return true
			}
			s.phase, s.t = sakReady, 0
			ann := s.run[len(s.run)/2].m.gu
			if len(s.fans) > 0 && s.fans[0].mine() {
				ann = s.fans[0].gu
			}
			ann.icon, ann.iconT = sackReady[0], 2.2
		}
	case sakReady:
		for _, ru := range s.run {
			ru.m.keep()
			ru.m.gu.pos = Vec{ru.x, ru.y}
			ru.m.gu.dir = sackDir(s.dir)
		}
		for _, m := range s.fans {
			m.gu.dir = faceTo(m.gu.pos, Vec{s.startX, mid.Y})
		}
		if s.t >= sackReadyT {
			s.phase, s.t = sakRace, 0
			s.cheerNext = 0.8
		}
	case sakRace:
		for _, ru := range s.run {
			ru.m.keep()
			gu := ru.m.gu
			if ru.done {
				ru.hop = 0
				gu.pos = Vec{ru.x, ru.y}
				continue
			}
			if ru.downT > 0 {
				ru.downT -= dt
				gu.pos = Vec{ru.x, ru.y}
				continue
			}
			ru.hop += dt / ru.period
			if ru.hop >= 1 {
				ru.hop -= 1
				ran := math.Abs(ru.x - s.startX)
				// a trip: the planned one, or now and then another (twice at most)
				if ru.falls < 2 && (ru.fallAt >= 0 && ran >= ru.fallAt || r.Float64() < 0.02) {
					ru.falls++
					ru.fallAt = -1
					ru.downT = sackFallT + sackUpT
					ru.hop = 0
					s.fellN++
					if gu.icon == "" {
						gu.icon, gu.iconT = pevPick(r, sackFallL), 1.4
					}
					if len(s.fans) > 0 && s.fans[r.Intn(len(s.fans))].gu.icon == "" {
						f := s.fans[r.Intn(len(s.fans))].gu
						if f.icon == "" {
							f.icon, f.iconT = pevPick(r, sackLaugh), 1.4
						}
					}
					continue
				}
			}
			if ru.hop > 0.1 && ru.hop < 0.9 {
				ru.x += s.dir * ru.speed * dt / 0.8
			}
			if (ru.x-s.finishX)*s.dir >= 0 {
				ru.x = s.finishX + s.dir*float64(4+s.finished*3)
				ru.done = true
				s.finished++
				ru.place = s.finished
				if s.winner == nil {
					s.winner = ru
				}
			}
			gu.pos = Vec{math.Round(ru.x), ru.y}
			gu.dir = sackDir(s.dir)
		}
		if s.t >= s.cheerNext {
			s.cheerNext = s.t + 1.3 + r.Float64()*0.8
			if len(s.fans) > 0 {
				f := s.fans[r.Intn(len(s.fans))].gu
				if f.icon == "" {
					f.icon, f.iconT = pevPick(r, sackCheer), 1.3
				}
			}
		}
		if s.finished == len(s.run) || s.winner != nil && s.t > sackRaceMax-6 || s.t > sackRaceMax {
			s.phase, s.t = sakEnd, 0
			if s.winner == nil { // nobody arrived in time: the one ahead wins
				for _, ru := range s.run {
					if s.winner == nil || math.Abs(ru.x-s.startX) > math.Abs(s.winner.x-s.startX) {
						s.winner = ru
					}
				}
			}
			w := s.winner.m.gu
			w.icon, w.iconT = pevPick(r, sackWinL), 2.2
			for _, ru := range s.run {
				ru.downT = 0
				if ru == s.winner {
					continue
				}
				if ru.m.gu.icon == "" && r.Intn(2) == 0 {
					ru.m.gu.icon, ru.m.gu.iconT = pevPick(r, sackLoseL), 2
				}
			}
		}
	case sakEnd:
		for _, ru := range s.run {
			ru.m.keep()
			gu := ru.m.gu
			if ru == s.winner {
				ru.hop = math.Mod(s.t*2.2, 1) // he jumps for joy
				gu.dir = DirDown
			} else {
				ru.hop = 0
				gu.dir = faceTo(gu.pos, s.winner.m.gu.pos)
			}
			gu.pos = Vec{math.Round(ru.x), ru.y}
		}
		if s.t >= sackEndT {
			sackEnd(g, s)
			return true
		}
	}
	return false
}

// sackAlarmNear: like pevAlarmNear, but the kids who run about playing (state gRun) are not an alarm.
func (g *Game) sackAlarmNear(p Vec, r float64) bool {
	for _, c := range g.cops {
		if c != nil && dist(c.pos, p) < r+80 {
			return true
		}
	}
	for _, gu := range g.guests {
		if gu.out() || isKid(gu) || dist(gu.pos, p) > r {
			continue
		}
		switch gu.state {
		case gAlarm, gFlee, gScared, gToPhone, gCall, gReport, gRun:
			return true
		}
	}
	return false
}

func sackDir(d float64) Dir {
	if d < 0 {
		return DirLeft
	}
	return DirRight
}

// ---- drawing -------------------------------------------------------------------------------------------------------

var (
	sackBurlap = color.RGBA{0xc8, 0x9a, 0x58, 255}
	sackSeam   = color.RGBA{0x8a, 0x62, 0x30, 255}
	sackTie    = color.RGBA{0x5a, 0x3a, 0x1a, 255}
	sackChalk  = color.RGBA{0xf0, 0xee, 0xe4, 255}
	sackPole   = color.RGBA{0xe8, 0xe8, 0xf0, 255}
	sackTape   = color.RGBA{0xe8, 0x30, 0x40, 255}
)

var sackFlagCols = []color.RGBA{{0xff, 0x50, 0x60, 255}, {0xff, 0xd0, 0x40, 255}, {0x40, 0xc0, 0xff, 255}, {0x60, 0xe0, 0x70, 255}}

// drawSackPix: the sack pulled up to the waist of a figure whose feet are at (cx, cy) and who is z pixels in the air.
func drawSackPix(plot plotFn, cx, cy, px, z int, k func(float64) int) {
	top := cy - k(14) - z
	h := k(14)
	plot(cx-5*px, top, 11*px, h, sackBurlap)
	plot(cx-6*px, top+px, px, h-2*px, sackSeam)
	plot(cx+6*px, top+px, px, h-2*px, sackSeam)
	plot(cx-5*px, top+h-px, 11*px, px, sackSeam)
	plot(cx-5*px, top, 11*px, px, sackTie) // the rim, held in his fists
	for y := top + 3*px; y < top+h-2*px; y += 3 * px {
		plot(cx-3*px, y, 2*px, px, sackSeam)
		plot(cx+2*px, y+px, 2*px, px, sackSeam)
	}
	plot(cx-6*px, top-px, 2*px, 2*px, pevSkinW) // the fists
	plot(cx+5*px, top-px, 2*px, 2*px, pevSkinW)
}

// drawGuest (pe_fx.go hook): the runners are drawn by the race: hopping, in their sacks, or lying after a trip.
func (s *sackState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if s.phase == sakGather || s.phase == sakStop {
		return false
	}
	var ru *sackRunner
	for _, x := range s.run {
		if x.m.gu == gu {
			ru = x
		}
	}
	if ru == nil {
		return false
	}
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	cx, cy := g.pevXY(pos)
	if ru.downT > sackUpT && s.phase == sakRace {
		g.drawLying(dst, gu.cs, false, pos)
		plot(cx-k(12), cy-k(5), k(9), k(5), sackBurlap) // the sack round his legs
		plot(cx-k(12), cy-k(5), px, k(5), sackTie)
		drawStarsPix(plot, cx, cy-k(12), px, g.time)
		return true
	}
	anim := AnimIdle
	if s.phase == sakEnd && ru != s.winner {
		anim = AnimLaugh
	}
	if s.phase == sakEnd && ru == s.winner {
		anim = AnimWave
	}
	z := int(ru.hopZ())
	g.drawActor(dst, gu.cs, anim, g.time*4, pos, gu.dir, -float64(z), false)
	drawSackPix(plot, cx, cy, px, k(float64(z)), k)
	return true
}

func sackItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s, ok := a.D.(*sackState)
	if !ok || s.phase == sakStop {
		return items
	}
	plot := g.plotter(dst)
	n := len(s.run)
	if n == 0 {
		return items
	}
	y0 := s.laneY(0, n) - sackLane/2 - 2
	y1 := s.laneY(n-1, n) + sackLane/2 + 2
	broken := s.winner != nil
	// the lines on the ground: under everybody of the course
	items = append(items, drawItem{s.course.Y - 1, func() {
		sx, fx := int(g.wx(s.startX)), int(g.wx(s.finishX))
		ya, yb := int(g.wy(y0)), int(g.wy(y1))
		for y := ya; y < yb; y += 4 {
			plot(sx-1, y, 2, 2, sackChalk)
		}
		for y := ya; y < yb; y += 2 { // a chequered finish
			c := sackChalk
			if (y/2)%2 == 0 {
				c = pevInk
			}
			plot(fx-2, y, 2, 2, c)
			if c == pevInk {
				c = sackChalk
			} else {
				c = pevInk
			}
			plot(fx, y, 2, 2, c)
		}
	}})
	// the poles with their pennants, and the tape between them (broken by the winner)
	pole := func(y float64, top bool) {
		x, yy := int(g.wx(s.finishX)), int(g.wy(y))
		plot(x-1, yy-18, 2, 18, sackPole)
		plot(x-1, yy-19, 2, 1, pevInk)
		for i := 0; i < 3; i++ {
			c := sackFlagCols[(i+boolI(top))%len(sackFlagCols)]
			fx := x + 1 + i*4
			if s.dir > 0 {
				fx = x - 4 - i*4
			}
			plot(fx, yy-18+i, 3, 3, c)
			plot(fx+1, yy-15+i, 1, 1, c)
		}
	}
	items = append(items, drawItem{y0 - 0.5, func() {
		pole(y0, true)
		x := int(g.wx(s.finishX))
		ya, yb := int(g.wy(y0)), int(g.wy(y1))
		if !broken {
			plot(x-1, ya-12, 2, yb-ya, sackTape)
		} else {
			plot(x-1, ya-12, 2, 6, sackTape)
		}
	}})
	items = append(items, drawItem{y1 + 0.5, func() {
		pole(y1, false)
		if broken {
			x, yb := int(g.wx(s.finishX)), int(g.wy(y1))
			plot(x-1, yb-18, 2, 6, sackTape)
		}
	}})
	// the fans clap
	if s.phase == sakRace || s.phase == sakEnd {
		for i, m := range s.fans {
			m, i := m, i
			if !m.mine() {
				continue
			}
			items = append(items, drawItem{m.gu.pos.Y + 0.1, func() {
				cx, cy := g.pevXY(m.gu.pos)
				drawClapPix(plot, cx, cy-g.pevK()(36), g.pevPx(), g.time, i)
			}})
		}
	}
	return items
}
