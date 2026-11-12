package main

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "ola" (idea 289): THE STADIUM WAVE. In the parties with stands (stadium, bullring, hockey rink, race circuit)
// the stands now and then do the wave: the painted crowd of every "stands" prop rises with its arms up, column after
// column, from one end to the other (the top stands left to right, then the bottom ones right to left), once or twice
// round (10-20 s), and sits down again. A row of 12 to 16 real guests lines up in front of the stands (sitting, a crouch)
// and rises with the wave when it passes them (arms up, a little jump).
//
// The row is made of guests of the party (the free ones nearest to the stands), never of new people: new guests would
// join the kill target (BAJAS x/N) and could not just vanish afterwards. If there are not 12 free guests, it waits and
// tries again later (the frame's retries). If the player kills one of the row, the wave stops for a moment (they look,
// "¡Eh!") and then goes on with the others; under 6 left, it ends. Every phase has its time limit (walking 20 s, the
// frame's pevGuard at 2*Dur); the guests are let go at the end.
//
// WHEN: group "stands", Big (never with another big group), chance 12 % only in the parties with stands (waveIcons),
// 20+ guests, once a party. Capture: -pevent ola (or ola:kill).
// ---------------------------------------------------------------------------

// The row is made of any free guest of the party (named ones too: with the asset folder most guests have a name).
// waveIcons: the campaign parties whose generator puts "stands" (gen2.go, gen_venues.go).
var waveIcons = map[string]bool{"stadium": true, "bull": true, "hockey": true, "circuit": true}

var (
	waveCall  = pevTexts("¡LA OLA!", "¡Todos a la vez!", "¡Ahí viene la ola!")
	waveUp    = pevTexts("¡Uoooh!", "¡Eeeeh!", "¡Arriba!")
	waveStop  = pevTexts("¡Eh!", "¿Y ese?", "¡Se ha caído uno!")
	waveAgain = pevTexts("¡Seguimos!", "¡Otra vez!")
	waveEnd   = pevTexts("¡Otra, otra!", "¡Qué grada!")
)

const (
	waveChance   = 0.12
	waveMinRow   = 12
	waveMaxRow   = 16
	waveStep     = 15.0 // between two of the row
	waveWalkMax  = 20.0
	waveStartMin = 10.0
	waveHalfW    = 18.0 // half the width of the risen band (px)
	wavePause    = 1.2  // the stop after a kill
	waveMinLeft  = 6
)

const (
	wvWalk = iota
	wvRun
	wvSit
	wvOver
)

// waveSide: the stands along one side (top or bottom) of the party.
type waveSide struct {
	top    bool
	x0, x1 float64
	props  []*Prop
}

type waveState struct {
	row      []*pevMem
	sides    []waveSide // the order the wave goes round: [top l->r, bottom r->l]
	rowSide  int        // the side the row stands in front of
	phase    int
	t0       float64
	laps     int
	lapT     float64
	s        float64 // progress of the front, in px along the whole lap
	pauseT   float64 // game time until which the wave is stopped
	kills    int
	upSeen   map[*Guest]bool // members that rose at least once (tests)
	lapsDone int
	demoK    bool
	born     float64 // when it started (the walking limit)
}

func (s *waveState) pevHas(gu *Guest) bool { return pevHasIn(s.row, gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "ola", Name: "La ola en la grada", Group: "stands", Big: true,
		Prob: waveChance, AtMin: 60, AtMax: 280, Dur: 30, MaxRuns: 1, Prio: 1,
		Eligible: func(g *Game, t partyTraits) bool { return waveIcons[t.icon] && t.roam && t.guests >= 20 },
		Start:    waveStart, Items: waveItems,
		Update: pevGuard(waveUpdate, func(g *Game, a *partyAct) { waveEnd2(g, a.D.(*waveState)) })})
}

// waveSides: the stands of the level grouped in a top and a bottom side.
func waveSides(l *Level) []waveSide {
	var top, bot waveSide
	top.top, top.x0, bot.x0 = true, math.Inf(1), math.Inf(1)
	top.x1, bot.x1 = math.Inf(-1), math.Inf(-1)
	for _, pr := range l.Props {
		if pr.Name != "stands" || pr.gone {
			continue
		}
		sd := &bot
		if pr.Y < float64(l.Def.H)/2 {
			sd = &top
		}
		w := float64(pr.K.W) / 2
		sd.x0, sd.x1 = math.Min(sd.x0, pr.X-w), math.Max(sd.x1, pr.X+w)
		sd.props = append(sd.props, pr)
	}
	var out []waveSide
	for _, sd := range []waveSide{top, bot} {
		if len(sd.props) > 0 {
			out = append(out, sd)
		}
	}
	return out
}

func (s *waveState) lapLen() float64 {
	t := 0.0
	for _, sd := range s.sides {
		t += sd.x1 - sd.x0 + 60
	}
	return t
}

// front: the side the wave is on now and the x of its middle (side -1 = between laps).
func (s *waveState) front() (int, float64) {
	d := math.Mod(s.s, s.lapLen())
	for i, sd := range s.sides {
		L := sd.x1 - sd.x0 + 60
		if d < L {
			if i%2 == 0 { // left to right
				return i, sd.x0 - 30 + d
			}
			return i, sd.x1 + 30 - d
		}
		d -= L
	}
	return -1, 0
}

// waveUpAt: how high (0, 1 or 2) a figure at x of side i is risen now.
func (s *waveState) waveUpAt(i int, x float64) int {
	if s.phase != wvRun {
		return 0
	}
	fi, fx := s.front()
	if fi != i {
		return 0
	}
	switch d := math.Abs(x - fx); {
	case d < waveHalfW*0.55:
		return 2
	case d < waveHalfW:
		return 1
	}
	return 0
}

// waveRowPoints: the longest line of walkable points in front of the stands of a side (nil if under waveMinRow).
func (g *Game) waveRowPoints(sd waveSide) []Vec {
	l := g.lvl
	inRoam := func(p Vec) bool {
		for _, r := range l.Def.Roam {
			if p.X >= r.X && p.X <= r.X+r.W && p.Y >= r.Y && p.Y <= r.Y+r.H {
				return true
			}
		}
		return false
	}
	var best []Vec
	// first where the guests roam (they get there soon), else any walkable line in front of the stands
	for pass := 0; pass < 2 && len(best) < waveMinRow; pass++ {
		ok := func(p Vec) bool {
			return l.Free(p) && !l.SolidP(p.X, p.Y) && !waveBadGround(l, p) && !l.InPool(p) && (pass == 1 || inRoam(p))
		}
		best = nil
		for dy := 8.0; dy <= 120; dy += 4 {
			for _, pr := range sd.props {
				var y float64
				if sd.top {
					y = pr.Y + dy
				} else {
					y = pr.Y - float64(pr.K.H) - dy
				}
				var run []Vec
				miss := 0
				flush := func() {
					if len(run) > len(best) {
						best = append([]Vec(nil), run...)
					}
					run, miss = run[:0], 0
				}
				for x := sd.x0 + 8; x <= sd.x1-8; x += waveStep {
					p := Vec{x, y}
					if ok(p) && g.waveFront(sd, p) {
						run, miss = append(run, p), 0
					} else if miss++; miss > 2 { // a bench or a gap of the stands: up to two places may be skipped
						flush()
					}
				}
				flush()
			}
			if len(best) >= waveMaxRow {
				break
			}
		}
	}
	if len(best) < waveMinRow {
		return nil
	}
	if len(best) > waveMaxRow { // the middle part
		k := (len(best) - waveMaxRow) / 2
		best = best[k : k+waveMaxRow]
	}
	return best
}

// waveBadFloors: where nobody stands still for the wave (roads, tracks, rails, water, ice); the pitch and the sand of
// the ring are fine (unlike badGround, which keeps the props off them).
var waveBadFloors = map[string]bool{"asphalt": true, "dash": true, "helipad": true, "water": true, "rail": true, "rails": true,
	"edge": true, "track": true, "lava": true, "ice": true}

func waveBadGround(l *Level, p Vec) bool {
	d := l.Def
	if d.Highway && d.Road.Has(p.X, p.Y) {
		return true
	}
	for _, f := range d.Floors {
		if waveBadFloors[f.Kind] && f.R.Grow(6).Has(p.X, p.Y) {
			return true
		}
	}
	for _, rs := range d.Roads {
		if rs.R.Grow(8).Has(p.X, p.Y) {
			return true
		}
	}
	return false
}

// waveFront: p is in front of some stands of the side (within their x span).
func (g *Game) waveFront(sd waveSide, p Vec) bool {
	for _, pr := range sd.props {
		if math.Abs(p.X-pr.X) <= float64(pr.K.W)/2-4 {
			return true
		}
	}
	return false
}

func waveStart(g *Game, a *partyAct) bool {
	r := partyDice()
	sides := waveSides(g.lvl)
	if len(sides) == 0 {
		return false
	}
	s := &waveState{sides: sides, t0: g.time, born: g.time, laps: 1 + r.Intn(2), upSeen: map[*Guest]bool{}}
	s.lapT = 8 + r.Float64()*2
	if s.laps == 1 {
		s.lapT = 11 + r.Float64()*3
	}
	// the side with the longest row of free points (the top one first when they tie)
	var pts []Vec
	for i, sd := range sides {
		if p := g.waveRowPoints(sd); len(p) > len(pts) {
			pts, s.rowSide = p, i
		}
	}
	if pts == nil {
		return false
	}
	mid := pts[len(pts)/2]
	var cand []*Guest
	for _, gu := range g.guests {
		if danceFree(g, gu) && !fursuitSlow(gu) && g.pevPlayerFar(gu.pos, 30) {
			cand = append(cand, gu)
		}
	}
	near := func(p Vec) float64 { // distance to the nearest place of the row
		d := math.Inf(1)
		for _, q := range pts {
			d = math.Min(d, dist(p, q))
		}
		return d
	}
	sort.SliceStable(cand, func(i, j int) bool { return near(cand[i].pos) < near(cand[j].pos) })
	if len(cand) > len(pts) {
		cand = cand[:len(pts)]
	}
	if len(cand) < waveMinRow {
		return false
	}
	// left to right: the guests by x go to the points by x (nobody crosses the row)
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].pos.X < cand[j].pos.X })
	k0 := (len(pts) - len(cand)) / 2
	for i, gu := range cand {
		m := &pevMem{gu: gu, pt: pts[k0+i], idx: i}
		if g.pevSendTo(m, m.pt) {
			s.row = append(s.row, m)
		}
	}
	if len(s.row) < waveMinRow {
		pevReleaseAll(s.row, r)
		return false
	}
	a.D = s
	if pevDemo("ola") != "" {
		pevDemoLog("ola", mid)
	}
	return true
}

// waveFace: the row looks at the field (away from its stands).
func (s *waveState) waveFace() Dir {
	if s.sides[s.rowSide].top {
		return DirDown
	}
	return DirUp
}

func waveUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*waveState)
	r := partyDice()
	age := g.time - s.t0
	if s.phase == wvOver {
		return true
	}
	// a member killed: the wave stops a moment and goes on; one taken by something else simply leaves the row
	out := s.row[:0]
	killed := 0
	for _, m := range s.row {
		if m.mine() {
			out = append(out, m)
			continue
		}
		if m.gu.dead {
			killed++
		}
		pevForget(m)
	}
	s.row = out
	if killed > 0 {
		s.kills += killed
		if s.phase == wvRun {
			s.pauseT = g.time + wavePause
			for _, m := range s.row {
				if m.want == gGoto {
					continue
				}
				m.pose(AnimCrouch)
				if m.gu.icon == "" && r.Float64() < 0.3 {
					m.gu.icon, m.gu.iconT = pevPick(r, waveStop), 1.2
				}
			}
		}
	}
	if len(s.row) < waveMinLeft && s.phase != wvSit {
		waveEnd2(g, s)
		return true
	}
	switch s.phase {
	case wvWalk:
		arrived := 0
		for _, m := range s.row {
			if peleaArrived(m) {
				arrived++
			}
		}
		// it starts when they are all there, or after waveStartMin with 10 seated (the late ones join as they arrive)
		if arrived == len(s.row) || age > waveStartMin && arrived >= waveMinRow-2 {
			for _, m := range s.row {
				if peleaArrived(m) {
					m.pose(AnimCrouch)
					m.gu.dir = s.waveFace()
				}
			}
			s.phase, s.t0 = wvRun, g.time
			mid := s.row[len(s.row)/2].gu
			mid.icon, mid.iconT = pevPick(r, waveCall), 2
		} else if age > waveWalkMax {
			waveEnd2(g, s)
			return true
		}
	case wvRun:
		if g.time >= s.pauseT {
			if s.pauseT > 0 && g.time-s.pauseT < dt*1.5 && len(s.row) > 0 && s.row[0].gu.icon == "" {
				s.row[0].gu.icon, s.row[0].gu.iconT = pevPick(r, waveAgain), 1.2
			}
			fi, fx := s.front()
			if !(pevDemo("ola") == "top" && fi == s.rowSide && len(s.row) > 0 && fx >= s.row[len(s.row)/2].gu.pos.X) {
				s.s += dt * s.lapLen() / s.lapT // (capture ola:top: the front stays over the middle of the row)
			}
		}
		if s.s >= s.lapLen()*float64(s.laps) {
			s.lapsDone = s.laps
			s.phase, s.t0 = wvSit, g.time
			for _, m := range s.row {
				if m.want == gGoto {
					continue
				}
				m.pose(AnimCheer)
				if m.gu.icon == "" && r.Float64() < 0.25 {
					m.gu.icon, m.gu.iconT = pevPick(r, waveEnd), 1.4
				}
			}
			break
		}
		s.lapsDone = int(s.s / s.lapLen())
		in := s.row[:0]
		for _, m := range s.row {
			if m.want == gGoto { // still walking to his place: he sits when he gets there, or gives up
				switch {
				case peleaArrived(m):
					m.pose(AnimCrouch)
				case g.time-s.born > waveWalkMax:
					pevRelease(m, r)
					continue
				default:
					in = append(in, m)
					continue
				}
			}
			in = append(in, m)
			m.keep()
			m.gu.dir = s.waveFace()
			up := s.waveUpAt(s.rowSide, m.gu.pos.X) > 0 && g.time >= s.pauseT
			if up && m.gu.dAnim != AnimCheer {
				m.pose(AnimCheer)
				s.upSeen[m.gu] = true
				if m.gu.icon == "" && r.Float64() < 0.08 {
					m.gu.icon, m.gu.iconT = pevPick(r, waveUp), 1
				}
			} else if !up && m.gu.dAnim != AnimCrouch {
				m.pose(AnimCrouch)
			}
		}
		s.row = in
		if pevDemo("ola") == "kill" && !s.demoK && age > 4 && len(s.row) > 0 {
			s.demoK = true
			peleaKill(g, s.row[len(s.row)/3].gu) // capture: one of the row dies (not by the player)
		}
	case wvSit:
		for _, m := range s.row {
			m.keep()
		}
		if age > 1.6 {
			waveEnd2(g, s)
			return true
		}
	}
	return false
}

func waveEnd2(g *Game, s *waveState) {
	if s.phase == wvOver {
		return
	}
	pevReleaseAll(s.row, partyDice())
	s.phase = wvOver
}

// drawGuest: a member of the row jumps a little while he is up.
func (s *waveState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if s.phase != wvRun || !pevHasIn(s.row, gu) || gu.state != gFun || gu.dAnim != AnimCheer {
		return false
	}
	anim, phase := g.guestPose(gu)
	g.drawActor(dst, gu.cs, anim, phase, pos, gu.dir, -2, false)
	return true
}

// the painted crowd of the stands prop (props3.go): rows t = 0..3 at y = 10 + 9t, a figure every 8 px.
var (
	waveSkins  = []uint32{0xe8b48a, 0xc98a5e, 0x8a5a3a, 0xf7d9c0}
	waveShirts = []uint32{0xe04a5a, 0x40a0e0, 0xf0c030, 0x60c070, 0xf4f4f0, 0x8a2aa8}
)

// waveTone: the muted palette of the prop sprites (stylizeSprite).
func waveTone(hex uint32) color.RGBA {
	c := rgb(hex)
	k := func(v uint8) uint8 { return uint8(float64(v) * 0.86) }
	return color.RGBA{k(c.R), k(c.G), k(c.B), 255}
}

// waveItems: the risen figures over the painted crowd of every stands prop.
func waveItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	s := a.D.(*waveState)
	if s.phase != wvRun || g.time < s.pauseT {
		return items
	}
	plot := g.plotter(dst)
	for i, sd := range s.sides {
		for _, pr := range sd.props {
			pr, i := pr, i
			if _, fx := s.front(); math.Abs(fx-pr.X) > float64(pr.K.W)/2+waveHalfW {
				continue
			}
			items = append(items, drawItem{pr.Y + 0.5, func() { drawWaveStands(g, plot, s, i, pr) }})
		}
	}
	return items
}

func drawWaveStands(g *Game, plot plotFn, s *waveState, side int, pr *Prop) {
	left := pr.X - float64(pr.K.W)/2
	top := pr.Y - float64(pr.K.H)
	ox, oy := int(math.Round(g.wx(left))), int(math.Round(g.wy(top)))
	for t := 0; t < 4; t++ {
		y := 10 + t*9
		for x := 3 + (t*5)%6; x < 206; x += 8 {
			up := s.waveUpAt(side, left+float64(x)+2)
			if up == 0 {
				continue
			}
			skin := waveTone(waveSkins[(x/8+t)%4])
			shirt := waveTone(waveShirts[(x/8*3+t*2+pr.Var)%6])
			h := 2*up + up/2 // risen 2 or 5 px
			px, py := ox+x, oy+y
			plot(px-1, py+1-h, 6, 6, shirt) // the body, higher
			plot(px, py-2-h, 4, 3, skin)    // the head
			if up == 2 {                    // the arms up
				plot(px-2, py-5-h, 1, 5, skin)
				plot(px+5, py-5-h, 1, 5, skin)
			}
		}
	}
}
