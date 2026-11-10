package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// COMMON PIECES of the eight "sometimes" events of pe_*.go (conga, chairs, drinks, barbecue, bachelor party,
// mime, juggler, buskers). The frame is partyevents.go; this file only has what they share:
//
//   - pevMem: a guest an event has taken. The event "owns" him while his state is the one it put him in (idle with
//     a huge timer, dancing, ...). If anything else touches him (the song changes, a murder, a fright) he is simply
//     let go: nothing is ever forced back. At the end pevRelease leaves him idle for a second or two.
//   - pevPerf: a street artist of its own (like the pizza boy: not a guest, never counts for the level).
//   - pevCrowd: guests who stop to watch an artist and toss coins; pevCoin is the coin in the air.
//   - pixel pieces drawn through plotFn (so the tests can paint them into an image): coin, note, clap, stars,
//     digits, hats, glasses...
//
// All the dice are partyDice(); g.rng is never moved here. Every text is in pevTexts(...) with its English in
// lang_pevents.go (a test checks the lot: at most 48 characters and an English version).
// ---------------------------------------------------------------------------

// pevTextES: every Spanish text the eight events can show (filled by pevTexts at init).
var pevTextES []string

func pevTexts(ss ...string) []string {
	pevTextES = append(pevTextES, ss...)
	return ss
}

func pevPick(r *rand.Rand, ss []string) string { return ss[r.Intn(len(ss))] }

// ---- guests the events take ------------------------------------------------------------------------

type pevMem struct {
	gu   *Guest
	want gState // the state the event put him in
	ang  float64
	pt   Vec     // his place (ring point, chair, slot...)
	next float64 // game time of his next small thing (coin, line...)
	idx  int
}

// mine: he is still in the state the event gave him and is not busy with anything else.
func (m *pevMem) mine() bool {
	gu := m.gu
	if gu == nil || gu.out() || !lifeFree(gu) {
		return false
	}
	switch m.want {
	case gGoto:
		return gu.state == gGoto || gu.state == gIdle && gu.timer > 1000
	case gIdle:
		return gu.state == gIdle && gu.timer > 1000
	case gDrink:
		return gu.state == gDrink && gu.dur >= 1000
	}
	return gu.state == m.want
}

// keep re-asserts the held state (called every frame BEFORE the guests are updated).
func (m *pevMem) keep() {
	gu := m.gu
	switch m.want {
	case gIdle:
		gu.timer, gu.lookT = 9999, 5
	case gDance, gFun:
		gu.timer = 3
	}
}

// pose holds him doing any animation of the character sheet (the "fun" state 18 of the star moments shows gu.dAnim).
func (m *pevMem) pose(anim int) {
	m.hold(gFun)
	m.gu.fun, m.gu.dAnim = 18, anim
}

func (m *pevMem) hold(st gState) {
	gu := m.gu
	m.want = st
	gu.state, gu.path, gu.moving = st, nil, false
	gu.timer = 9999
	if st == gDance || st == gFun {
		gu.timer = 3
	}
	if st == gDrink {
		gu.dur = 9999
	}
}

// pevTaken: another running party event has him already.
func (g *Game) pevTaken(gu *Guest) bool {
	for _, a := range g.pev.act {
		if h, ok := a.D.(interface{ pevHas(*Guest) bool }); ok && h.pevHas(gu) {
			return true
		}
		if ts, ok := a.D.(*toastState); ok && ts.sp == gu {
			return true
		}
	}
	return false
}

// pevFreeGuest: a guest who can be taken without breaking anything he is doing.
func (g *Game) pevFreeGuest(gu *Guest) bool {
	if !lifeFree(gu) || gu.quirk != "" || gu.leader != nil || gu.armed || gu.partner != nil || g.inSelfie(gu) || g.inContest(gu) || g.pevTaken(gu) {
		return false
	}
	if starSlugs[guestSlug(gu)] || g.duet != nil && (g.duet.a == gu || g.duet.b == gu) { // they have acts of their own
		return false
	}
	switch gu.state {
	case gIdle, gGoto, gDance, gLounge:
		return gu.timer < 1000
	}
	return false
}

// pevSendTo: sends him walking to p; he stops there idle and held (timer 9999). false = no way.
func (g *Game) pevSendTo(m *pevMem, p Vec) bool {
	gu := m.gu
	path := g.lvl.PathG(gu.pos, p)
	if path == nil && dist(gu.pos, p) > 6 {
		return false
	}
	m.want = gGoto
	gu.path, gu.state, gu.next, gu.dur, gu.hurry, gu.timer, gu.moving = path, gGoto, gIdle, 9999, false, 0, false
	return true
}

// pevRelease lets go of one guest who is still ours: idle a moment, then he decides by himself.
func pevRelease(m *pevMem, r *rand.Rand) {
	gu := m.gu
	if gu == nil || gu.out() || !m.mine() {
		return
	}
	gu.state, gu.timer, gu.path, gu.moving = gIdle, 0.5+r.Float64()*2.2, nil, false
	gu.dur, gu.lookT = 0, 0
}

func pevReleaseAll(ms []*pevMem, r *rand.Rand) {
	for _, m := range ms {
		if m.mine() {
			pevRelease(m, r)
		} else {
			pevForget(m)
		}
	}
}

func pevHasIn(ms []*pevMem, gu *Guest) bool {
	for _, m := range ms {
		if m.gu == gu {
			return true
		}
	}
	return false
}

// pevPruneMems drops the ones who are not ours any more (not released: someone else has them now).
func pevPruneMems(ms []*pevMem) []*pevMem {
	out := ms[:0]
	for _, m := range ms {
		if m.mine() {
			out = append(out, m)
		} else {
			pevForget(m)
		}
	}
	return out
}

// pevForget: the event lets go of a guest that something else took (a fright, a murder). The "walk there and stay"
// length (dur 9999) is wiped so that if the game later sends him walking without setting it, he does not freeze at the end.
func pevForget(m *pevMem) {
	if gu := m.gu; gu != nil && !gu.out() && gu.dur >= 9999 && gu.state != gDrink {
		gu.dur = 0
	}
}

// ---- places ----------------------------------------------------------------------------------------

// pevClear: the spot and a ring of 8 points around it are walkable.
func (g *Game) pevClear(p Vec, rad float64) bool {
	l := g.lvl
	if !l.Free(p) || l.SolidP(p.X, p.Y) || l.badGround(p) || l.InPool(p) {
		return false
	}
	for i := 0; i < 8; i++ {
		a := float64(i) * math.Pi / 4
		q := Vec{p.X + math.Cos(a)*rad, p.Y + math.Sin(a)*rad*0.8}
		if !l.Free(q) || l.SolidP(q.X, q.Y) || l.badGround(q) || l.InPool(q) {
			return false
		}
	}
	return true
}

func (g *Game) pevPlayerFar(p Vec, d float64) bool {
	for _, pl := range g.players() {
		if pl != nil && !pl.dead && dist(pl.pos, p) < d {
			return false
		}
	}
	return true
}

// pevSpot: an open, walkable spot of the roaming area (with room all around it), away from the players.
func (g *Game) pevSpot(r *rand.Rand, rad float64, outdoor bool) (Vec, bool) {
	if len(g.lvl.Def.Roam) == 0 {
		return Vec{}, false
	}
	for i := 0; i < 90; i++ {
		p := g.lvl.RandomRoam(r)
		if outdoor && !g.lvl.Outdoor(p) {
			continue
		}
		if g.pevClear(p, rad) && g.pevPlayerFar(p, 70) {
			return p, true
		}
	}
	return Vec{}, false
}

func (g *Game) pevXY(p Vec) (int, int) {
	return int(math.Round(g.wx(p.X))), int(math.Round(g.wy(p.Y)))
}

// pevK scales a size in sprite pixels to the screen (people can be drawn smaller).
func (g *Game) pevK() func(v float64) int {
	sc := g.lifeScale()
	return func(v float64) int { return int(math.Round(v * sc)) }
}

func (g *Game) pevPx() int { return int(math.Max(1, math.Round(g.lifeScale()))) }

// ---- the trail of a leader: the ones behind follow his exact steps --------------------------------

type pevTrail struct{ pts []Vec }

func (tr *pevTrail) add(p Vec) {
	if n := len(tr.pts); n == 0 || dist(tr.pts[n-1], p) >= 2 {
		tr.pts = append(tr.pts, p)
		if len(tr.pts) > 900 {
			tr.pts = append(tr.pts[:0], tr.pts[300:]...)
		}
	}
}

func (tr *pevTrail) length() float64 {
	s := 0.0
	for i := 1; i < len(tr.pts); i++ {
		s += dist(tr.pts[i-1], tr.pts[i])
	}
	return s
}

// prefix lays a straight tail behind the leader (opposite to `fwd`) while the points are free, so the first dancers
// have a place to stand before he has walked anywhere.
func (tr *pevTrail) prefix(from, fwd Vec, free func(Vec) bool) {
	var back []Vec
	for k := 1; k <= 90; k++ {
		p := Vec{from.X - fwd.X*float64(k)*2, from.Y - fwd.Y*float64(k)*2}
		if !free(p) {
			break
		}
		back = append(back, p)
	}
	for i := len(back) - 1; i >= 0; i-- {
		tr.pts = append([]Vec{back[i]}, tr.pts...)
	}
}

// faceVec: unit vector from a to b.
func faceVec(a, b Vec) Vec {
	d := dist(a, b)
	if d == 0 {
		return Vec{0, 1}
	}
	return Vec{(b.X - a.X) / d, (b.Y - a.Y) / d}
}

// at: the point d pixels behind the leader along his steps (the oldest one if he has not walked that far).
func (tr *pevTrail) at(d float64) Vec {
	n := len(tr.pts)
	if n == 0 {
		return Vec{}
	}
	for i := n - 1; i > 0; i-- {
		seg := dist(tr.pts[i], tr.pts[i-1])
		if seg >= d && seg > 0 {
			k := d / seg
			return Vec{tr.pts[i].X + (tr.pts[i-1].X-tr.pts[i].X)*k, tr.pts[i].Y + (tr.pts[i-1].Y-tr.pts[i].Y)*k}
		}
		d -= seg
	}
	return tr.pts[0]
}

// ---- a street artist of its own (not a guest) --------------------------------------------------------

type pevPerf struct {
	pos, entry Vec
	dir        Dir
	cs         *CharSet
	path       []Vec
	moving     bool
	anim       float64
}

func newPevPerf(spec CharSpec, entry Vec, dir Dir) *pevPerf {
	spec.noOverride = true
	return &pevPerf{pos: entry, entry: entry, dir: dir, cs: newCharSet(&spec)}
}

const pevPerfSpeed = 22.0

// step walks along path; true = arrived (or nothing to walk).
func (pf *pevPerf) step(dt float64) bool {
	m, arr := follow(&pf.pos, &pf.dir, &pf.path, pevPerfSpeed, dt)
	pf.moving = m
	if m {
		pf.anim += dt * stridePhase(pevPerfSpeed*paceK)
	}
	return arr
}

// draw: the artist in the animation given (or walking if he is on the move).
func (pf *pevPerf) draw(g *Game, dst *ebiten.Image, anim int, phase float64) {
	if pf.moving {
		anim, phase = AnimWalk, pf.anim
	}
	g.drawActor(dst, pf.cs, anim, phase, pf.pos, pf.dir, 0, false)
}

// ---- spectators and coins ------------------------------------------------------------------------

type pevCoin struct {
	from, to Vec
	t0       float64
}

const pevCoinDur = 0.55

// pevCoinAt: where the coin is at time now (world), how high above the ground (px) and whether it has landed.
func pevCoinAt(c pevCoin, now float64) (Vec, float64, bool) {
	u := (now - c.t0) / pevCoinDur
	if u >= 1 {
		return c.to, 0, true
	}
	if u < 0 {
		u = 0
	}
	return Vec{c.from.X + (c.to.X-c.from.X)*u, c.from.Y + (c.to.Y-c.from.Y)*u}, 22*math.Sin(math.Pi*u) + 8*(1-u), false
}

type pevCrowd struct {
	mem    []*pevMem
	center Vec
}

func (c *pevCrowd) pevHas(gu *Guest) bool { return pevHasIn(c.mem, gu) }

// gather sends up to n free guests from near the centre to stand around it (in front first, then the sides).
func (c *pevCrowd) gather(g *Game, r *rand.Rand, center Vec, n int, rmin, rmax float64, skip func(*Guest) bool) int {
	c.center = center
	var cand []*Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && dist(gu.pos, center) < 240 && !(skip != nil && skip(gu)) {
			cand = append(cand, gu)
		}
	}
	// the nearest ones watch
	for i := 1; i < len(cand); i++ {
		for j := i; j > 0 && dist(cand[j].pos, center) < dist(cand[j-1].pos, center); j-- {
			cand[j], cand[j-1] = cand[j-1], cand[j]
		}
	}
	ang0 := math.Pi/2 + (r.Float64()-0.5)*0.4
	k := 0
	for _, gu := range cand {
		if len(c.mem) >= n {
			break
		}
		// fan: 0, +, -, ++, -- ... around the front of the artist
		off := float64((k+1)/2) * 0.62
		if k%2 == 0 {
			off = -off
		}
		a := ang0 + off
		rad := rmin + r.Float64()*(rmax-rmin)
		pt := g.lvl.NearestFree(Vec{center.X + math.Cos(a)*rad, center.Y + math.Sin(a)*rad*0.8})
		if dist(pt, center) < rmin*0.6 || !g.lvl.clearLine(pt, pt) {
			continue
		}
		clash := false
		for _, o := range c.mem {
			if dist(o.pt, pt) < 9 {
				clash = true
			}
		}
		if clash {
			continue
		}
		m := &pevMem{gu: gu, pt: pt, next: g.time + 1.5 + r.Float64()*3}
		if !g.pevSendTo(m, pt) {
			continue
		}
		c.mem = append(c.mem, m)
		k++
	}
	return len(c.mem)
}

// update: keeps them facing the artist; tipCoin (when not nil) is where the coins fall; every `every` seconds one of
// them tosses one. The artist's reaction is the caller's business (it gets the landed coins back).
func (c *pevCrowd) update(g *Game, r *rand.Rand, tipTo *Vec, every float64, coins *[]pevCoin) {
	c.mem = pevPruneMems(c.mem)
	for _, m := range c.mem {
		gu := m.gu
		if gu.state == gIdle && gu.timer > 1000 {
			m.want = gIdle
			m.keep()
			gu.dir = faceTo(gu.pos, c.center)
			if tipTo != nil && coins != nil && g.time >= m.next {
				m.next = g.time + every*(0.6+0.8*r.Float64())
				*coins = append(*coins, pevCoin{from: Vec{gu.pos.X, gu.pos.Y - 24}, to: Vec{tipTo.X + (r.Float64()-0.5)*8, tipTo.Y + 1 + r.Float64()*3}, t0: g.time})
				if r.Float64() < 0.5 {
					g.snd.Play("spark")
				}
			}
		}
	}
}

func (c *pevCrowd) release(g *Game, r *rand.Rand) { pevReleaseAll(c.mem, r); c.mem = nil }

// settleCoins removes the landed coins and returns how many fell.
func pevSettleCoins(coins []pevCoin, now float64) ([]pevCoin, int) {
	out := coins[:0]
	n := 0
	for _, c := range coins {
		if now-c.t0 >= pevCoinDur {
			n++
		} else {
			out = append(out, c)
		}
	}
	return out, n
}

// pevGuard: the frame ends an event by force at 2*Dur+1 seconds without asking it; ours end by themselves a bit before,
// cleaning up (the guests are let go), whatever phase they are in.
func pevGuard(upd func(*Game, *partyAct, float64) bool, end func(*Game, *partyAct)) func(*Game, *partyAct, float64) bool {
	return func(g *Game, a *partyAct, dt float64) bool {
		if g.time-a.t0 > a.ev.Dur*2-1 {
			end(g, a)
			return true
		}
		return upd(g, a, dt)
	}
}

// ---- the E key on something of these events -------------------------------------------------------------

// pevInteract: a running event may answer to E (the juggler's ball, the honoree's dare).
func (g *Game) pevInteract(p *Player) bool {
	if !g.partyActive() || p == nil || p.carrying != nil || p.dead {
		return false
	}
	for _, a := range g.pev.act {
		if it, ok := a.D.(interface{ pevUse(*Game, *Player) bool }); ok && it.pevUse(g, p) {
			return true
		}
	}
	return false
}

// ---- pixel pieces (all through plotFn: the tests paint them into an image) ---------------------------

var (
	pevGold   = color.RGBA{0xf4, 0xc4, 0x30, 255}
	pevGoldDk = color.RGBA{0xb8, 0x8a, 0x10, 255}
	pevInk    = color.RGBA{0x16, 0x12, 0x1c, 255}
	pevWhite  = color.RGBA{0xf8, 0xf4, 0xea, 255}
	pevSkinW  = color.RGBA{0xf4, 0xee, 0xe6, 255}
)

// drawCoinPix: a coin of 2x2 (px 1) with a glint.
func drawCoinPix(plot plotFn, x, y, px int) {
	plot(x, y, 2*px, 2*px, pevGold)
	plot(x, y, px, px, pevWhite)
	plot(x+px, y+px, px, px, pevGoldDk)
}

// drawNotePix: a musical note of 4x7 (px 1) whose head is at (x, y+5).
func drawNotePix(plot plotFn, x, y, px int, col color.RGBA, flag bool) {
	plot(x+2*px, y, px, 6*px, col)
	plot(x, y+5*px, 3*px, 2*px, col)
	if flag {
		plot(x+3*px, y, px, px, col)
		plot(x+3*px, y+px, px, px, col)
	}
}

var pevNoteCols = []color.RGBA{{0xff, 0x6a, 0x9a, 255}, {0x6a, 0xd8, 0xff, 255}, {0xff, 0xe0, 0x60, 255}, {0x9a, 0xf0, 0x7a, 255}}

// drawNotesPix: n notes that float up and fade out above (cx, top); t in seconds.
func drawNotesPix(plot plotFn, cx, top, px int, t float64, n int) {
	for i := 0; i < n; i++ {
		ph := math.Mod(t*0.55+float64(i)/float64(n), 1)
		x := cx + int(math.Round(math.Sin(ph*5+float64(i)*2.1)*5))*px + (i-n/2)*4*px
		y := top - int(math.Round(ph*22))*px
		if ph > 0.8 && int(t*8)%2 == 0 { // blinks out
			continue
		}
		drawNotePix(plot, x, y, px, pevNoteCols[i%len(pevNoteCols)], i%2 == 0)
	}
}

// drawClapPix: two hands that open and close (the applause), over a head at (cx, y).
func drawClapPix(plot plotFn, cx, y, px int, t float64, i int) {
	gap := 2 * px
	if (int(t*6)+i)%2 == 0 {
		gap = 5 * px
	}
	plot(cx-gap-px, y, px, 2*px, clapCol)
	plot(cx+gap, y, px, 2*px, clapCol)
}

// drawStarsPix: stars that circle around a dizzy head centred on (cx, cy); t in seconds.
func drawStarsPix(plot plotFn, cx, cy, px int, t float64) {
	for i := 0; i < 3; i++ {
		a := t*4 + float64(i)*2.094
		x := cx + int(math.Round(math.Cos(a)*7))*px
		y := cy + int(math.Round(math.Sin(a)*2.5))*px
		plot(x, y-px, px, 3*px, pevGold)
		plot(x-px, y, 3*px, px, pevGold)
		plot(x, y, px, px, pevWhite)
	}
}

var pevDigits = [10][5]uint8{
	{7, 5, 5, 5, 7}, {2, 6, 2, 2, 7}, {7, 1, 7, 4, 7}, {7, 1, 7, 1, 7}, {5, 5, 7, 1, 1},
	{7, 4, 7, 1, 7}, {7, 4, 7, 5, 7}, {7, 1, 1, 1, 1}, {7, 5, 7, 5, 7}, {7, 5, 7, 1, 7},
}

// drawDigitPix: a digit of 3x5 (px 1) with its top-left corner at (x, y), with a dark shadow so it reads on anything.
func drawDigitPix(plot plotFn, x, y, px, d int, col color.RGBA) {
	if d < 0 || d > 9 {
		return
	}
	for r := 0; r < 5; r++ {
		for c := 0; c < 3; c++ {
			if pevDigits[d][r]>>(2-c)&1 == 1 {
				plot(x+(c+1)*px, y+(r+1)*px, px, px, pevInk)
			}
		}
	}
	for r := 0; r < 5; r++ {
		for c := 0; c < 3; c++ {
			if pevDigits[d][r]>>(2-c)&1 == 1 {
				plot(x+c*px, y+r*px, px, px, col)
			}
		}
	}
}

// drawLinePix: a staircase line of w x w pixels.
func drawLinePix(plot plotFn, x0, y0, x1, y1, w int, col color.RGBA) {
	steps := maxI(absI(x1-x0), absI(y1-y0))
	for i := 0; i <= steps; i++ {
		x := x0 + (x1-x0)*i/maxI(steps, 1)
		y := y0 + (y1-y0)*i/maxI(steps, 1)
		plot(x, y, w, w, col)
	}
}

// drawChefHatPix: a tall white cook's hat whose brim sits at (cx, top+...) ; top = y of the head's top.
func drawChefHatPix(plot plotFn, cx, top, px int) {
	plot(cx-3*px, top-2*px, 7*px, 3*px, pevWhite)
	plot(cx-4*px, top-5*px, 3*px, 3*px, pevWhite)
	plot(cx-px, top-6*px, 3*px, 4*px, pevWhite)
	plot(cx+2*px, top-5*px, 3*px, 3*px, pevWhite)
	plot(cx-3*px, top, 7*px, px, color.RGBA{0xc8, 0xc4, 0xbc, 255})
}

// drawApronPix: a white apron over the chest and belly of a figure whose feet are at (cx, cy).
func drawApronPix(plot plotFn, cx, cy, px int, k func(float64) int) {
	top := cy - k(25)
	h := k(14)
	plot(cx-2*px, top, 5*px, h, pevWhite)
	plot(cx-3*px, top+k(5), 7*px, h-k(5), pevWhite)
	plot(cx-px, top+k(4), px, k(4), color.RGBA{0xd8, 0x40, 0x30, 255}) // a pocket stain
	plot(cx-3*px, top+h-px, 7*px, px, color.RGBA{0xc8, 0xc4, 0xbc, 255})
}

func drawStarGlassesPix(plot plotFn, cx, ey, px int, col color.RGBA) {
	plot(cx-5*px, ey, 4*px, 2*px, pevInk)
	plot(cx+px, ey, 4*px, 2*px, pevInk)
	plot(cx-px, ey, 2*px, px, pevInk)
	plot(cx-4*px, ey, 2*px, px, col)
	plot(cx+2*px, ey, 2*px, px, col)
	plot(cx-3*px, ey-px, px, px, col)
	plot(cx+3*px, ey-px, px, px, col)
}

// ---- a show: an artist walks in, performs for a while with an audience that tosses coins, and walks out -------

const (
	showEnter = iota
	showAct
	showLeave
)

type pevShow struct {
	pf      *pevPerf
	crowd   pevCrowd
	phase   int
	t0      float64
	spot    Vec
	tip     Vec // where the coins fall
	coins   []pevCoin
	tips    int
	actLen  float64
	started bool
}

func (s *pevShow) pevHas(gu *Guest) bool { return s.crowd.pevHas(gu) }

// pevShowStart: the artist appears at the farthest roaming point and walks to a clear spot. false = no room now.
func (g *Game) pevShowStart(spec CharSpec, actLen float64, outdoor bool) (*pevShow, bool) {
	r := partyDice()
	spot, ok := g.pevSpot(r, 22, outdoor)
	if !ok {
		return nil, false
	}
	// he walks in from a roaming point 90-220 px away (about ten seconds of walking), away from the players
	var entry Vec
	var path []Vec
	for i := 0; i < 40 && path == nil; i++ {
		p := g.lvl.NearestFree(g.lvl.RandomRoam(r))
		if d := dist(p, spot); d < 90 || d > 220 || !g.pevPlayerFar(p, 60) {
			continue
		}
		if pt := g.lvl.PathG(p, spot); pt != nil {
			entry, path = p, pt
		}
	}
	if path == nil {
		return nil, false
	}
	s := &pevShow{pf: newPevPerf(spec, entry, DirDown), spot: spot, tip: Vec{spot.X + 12, spot.Y + 4}, actLen: actLen, t0: g.time}
	s.pf.path = path
	return s, true
}

// pevShowStep runs the three phases. onAct is called once when the artist reaches his spot (crowd: how many watchers);
// land gets the number of coins that fell this frame. true = the show is over.
func (g *Game) pevShowStep(s *pevShow, dt float64, watchers int, every float64, land func(n int)) bool {
	r := partyDice()
	age := g.time - s.t0
	switch s.phase {
	case showEnter:
		if s.pf.step(dt) {
			s.phase, s.t0, s.pf.dir = showAct, g.time, DirDown
			s.crowd.gather(g, r, s.pf.pos, watchers, 26, 40, nil)
		} else if age > 22 {
			return true
		}
	case showAct:
		s.crowd.update(g, r, &s.tip, every, &s.coins)
		var n int
		s.coins, n = pevSettleCoins(s.coins, g.time)
		if n > 0 {
			s.tips += n
			if land != nil {
				land(n)
			}
		}
		if age >= s.actLen {
			s.phase, s.t0 = showLeave, g.time
			s.crowd.release(g, r)
			s.coins = nil
			s.pf.path = g.lvl.PathG(s.pf.pos, s.pf.entry)
			if s.pf.path == nil {
				return true
			}
		}
	case showLeave:
		if s.pf.step(dt) || age > 22 {
			return true
		}
	}
	return false
}

func (s *pevShow) end(g *Game) { s.crowd.release(g, partyDice()) }

// showItems: the coins in the air, in the y-sorted list.
func showCoinItems(g *Game, dst *ebiten.Image, s *pevShow, items []drawItem) []drawItem {
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	for _, c := range s.coins {
		c := c
		items = append(items, drawItem{c.to.Y + 6, func() {
			p, h, landed := pevCoinAt(c, g.time)
			if landed {
				return
			}
			x, y := g.pevXY(p)
			drawCoinPix(plot, x, y-k(h), px)
		}})
	}
	return items
}

// showClapItems: the audience applauds over their heads while `on`.
func showClapItems(g *Game, dst *ebiten.Image, s *pevShow, on bool, items []drawItem) []drawItem {
	if !on {
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

// drawCoinPilePix: the coins that fell on the ground (n up to 8) in a little heap at (x, y).
func drawCoinPilePix(plot plotFn, x, y, px, n int) {
	if n > 8 {
		n = 8
	}
	for i := 0; i < n; i++ {
		drawCoinPix(plot, x+(i%4)*3*px-5*px, y-(i/4)*px*2, px)
	}
}

// applause: a few of the audience clap and shout.
func (s *pevShow) cheer(r *rand.Rand, lines []string) {
	for _, m := range s.crowd.mem {
		if m.gu.icon == "" && r.Float64() < 0.5 {
			m.gu.icon, m.gu.iconT = pevPick(r, lines), 1.8
		}
	}
}
