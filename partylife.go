package main

import (
	"image/color"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// LIFE OF THE PARTY: five small things that make the party feel alive. None of them is a guest and none of them
// counts to finish the level. All the dice are our own (lifeDice): g.rng is not moved, so the rest of the party is
// the same as before.
//
//  1. THE DJ CHANGES THE SONG (lifeSongChanged, djBoothUse): when the song changes, some guests (not all) walk to
//     the dance floor and dance for a while. The player can change the song with E next to the booth; sometimes
//     the DJ gets the record wrong and a waltz plays (an existing synthesised genre, no new audio).
//  2. SELFIES: from time to time an idle guest, alone or with the group around him, takes a selfie (arm out,
//     flash, a short line). Only an overlay is drawn: the guest keeps his state, gait and pick.
//  3. DANCE CONTEST: rarely, a group dances in a circle on the floor, one wins and celebrates. No points.
//  4. WRONG PIZZA BOY: rare (about 1 party in 6). He walks in looking for another address. E helps him and he
//     gives a pizza (full stamina and a few points). If he is killed it is a murder like a guard's and
//     lifeState.pizzaDead remembers it for the newspaper. He leaves the way he came.
//  5. LEVEL MASCOT: a harmless animal of its own (cat, crab, penguin or duck, by kind of level). It is NOT in
//     g.animals (so animalMax and bullMax are not touched) and it is not a guest. E pets it (small points once);
//     if it is alive when the party is completed there is a bonus (lifeFinish); if it dies, nothing.
// ---------------------------------------------------------------------------

const (
	selfieEvery     = 16.0 // seconds between two attempts (plus lifeSelfieJit)
	selfieJit       = 24.0
	selfieChance    = 0.7
	selfieDur       = 2.6  // seconds of a selfie
	selfieFlashAt   = 1.9  // the flash goes off this far into it
	selfieFlashLen  = 0.16 //
	selfieGroupP    = 0.5  // share of selfies taken as a group
	selfieGroupNear = 38.0 // who joins the group
	selfieMax       = 2    // at the same time

	contestChance = 0.30 // parties with a dance contest
	contestMinN   = 4
	contestMaxN   = 7
	contestGather = 14.0 // seconds to reach the circle
	contestDance  = 9.0
	contestWin    = 4.5

	waveShare    = 0.35 // share of the free guests who go to dance when the song changes
	waveMax      = 12
	waveGap      = 6.0  // two waves are at least this far apart
	djCoolTime   = 25.0 // the DJ lets the player change the song once in a while
	djOopsChance = 0.30 // the DJ gets the record wrong

	pizzaChance   = 1.0 / 6 // parties with a pizza boy who got the wrong address
	pizzaStamina  = 40      // points for helping him
	pizzaWaitMax  = 42.0
	pizzaSpeed    = 26.0
	mascotPet     = 15  // points for petting the mascot (once)
	mascotBonus   = 100 // bonus if it is alive when the party is completed
	mascotSpeed   = 13.0
	mascotReach   = 30.0
	pizzaReach    = 30.0
	lifeStabReach = 24.0
)

var (
	lifeRng  *rand.Rand
	lifeSeed int64 // tests: non-zero = deterministic dice
)

func lifeDice() *rand.Rand {
	if lifeRng == nil {
		sd := lifeSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		lifeRng = rand.New(rand.NewSource(sd))
	}
	return lifeRng
}

type selfie struct {
	gs []*Guest // gs[0] holds the phone
	t0 float64
}

type contestPhase int

const (
	ctGather contestPhase = iota
	ctDance
	ctWin
)

type danceContest struct {
	gs     []*Guest
	pts    []Vec
	center Vec
	phase  contestPhase
	t0     float64
	winner *Guest
	nextIc float64
}

const (
	pzEnter = iota
	pzAsk
	pzGift
	pzLeave
)

type pizzaBoy struct {
	pos, entry Vec
	dir        Dir
	cs         *CharSet
	path       []Vec
	state      int
	t0, timer  float64
	anim       float64
	moving     bool
	moveAt     float64
}

type mascot struct {
	kind   string // cat, crab, penguin, duck
	pos    Vec
	dir    Dir
	path   []Vec
	timer  float64
	moving bool
	petted bool
	dead   bool
	anim   float64
}

type lifeState struct {
	ready      bool
	selfieT    float64
	selfies    []selfie
	contest    *danceContest
	contestAt  float64 // game time when the contest starts (0 = none this party)
	pizza      *pizzaBoy
	pizzaAt    float64 // game time when he comes in (0 = none this party)
	pizzaDead  bool    // killed by the player (the newspaper may say "delivery man missing")
	contestRan bool    // the contest really started this party (the newspaper reads it; contestAt goes back to 0)
	pizzaRan   bool    // the pizza boy really came in this party (the newspaper reads it; pizzaAt goes back to 0)
	pizzaHelp  bool
	mascot     *mascot
	mascotDead bool
	djCool     float64
	waveAt     float64
	oops       bool           // the song that plays now is the DJ's mistake
	oopsNew    float64        // game time of the last mistake still waiting for the music to change
	evBase     map[string]int // the counters "ev_<kind>" of the statistics when the party started (the newspaper
	// compares them to know which big set pieces happened in this party: paperRareSet; read only, nothing is saved)
}

// lifeReset: new party. Rolls the rare things with the own dice.
func (g *Game) lifeReset() {
	r := lifeDice()
	g.life = lifeState{ready: true, waveAt: -999, oopsNew: -999, evBase: map[string]int{}}
	for k, v := range g.progress.Stats {
		if strings.HasPrefix(k, "ev_") {
			g.life.evBase[k] = v
		}
	}
	if g.lvl == nil || g.stealth || g.levelIdx == specialJurassic {
		return
	}
	ls := &g.life
	ls.selfieT = 18 + r.Float64()*selfieJit
	d := g.lvl.Def
	if len(d.Dance) > 0 && r.Float64() < contestChance {
		ls.contestAt = 60 + r.Float64()*140
	}
	if len(d.Roam) > 0 && r.Float64() < pizzaChance {
		ls.pizzaAt = 35 + r.Float64()*120
	}
	g.mascotSpawn(r)
}

func (g *Game) lifeActive() bool {
	return g.mode == ModePlay && g.lvl != nil && !g.stealth && g.levelIdx != specialJurassic
}

func (g *Game) updateLife(dt float64) {
	if !g.lifeActive() {
		return
	}
	if !g.life.ready {
		g.lifeReset()
	}
	g.lifeSelfies(dt)
	g.lifeContest(dt)
	g.lifePizza(dt)
	g.lifeMascot(dt)
}

// lifeFree: a guest that can be sent somewhere without breaking anything he is doing.
func lifeFree(gu *Guest) bool {
	return gu != nil && !gu.out() && gu.perf == "" && gu.py == nil && gu.inside == nil && gu.queue == nil &&
		gu.poison <= 0 && gu.bitten <= 0 && gu.vipStun <= 0 && gu.lift <= 0 && !gu.tech
}

// ---------------------------------------------------------------------------
// 1. THE SONG CHANGES -> SOME GUESTS GO DANCING
// ---------------------------------------------------------------------------

// lifeSongChanged: the music changed (the timer of the DJ, the end of a file or the player at the booth).
func (g *Game) lifeSongChanged() {
	if !g.lifeActive() {
		return
	}
	if !g.life.ready {
		g.lifeReset()
	}
	g.danceWave(lifeDice())
}

// danceWave sends a share of the free guests to the dance floor to dance for a while. It returns how many went.
func (g *Game) danceWave(r *rand.Rand) int {
	ls := &g.life
	d := g.lvl.Def
	if len(d.Dance) == 0 || g.time-ls.waveAt < waveGap {
		return 0
	}
	ls.waveAt = g.time
	n := 0
	for _, gu := range g.guests {
		if n >= waveMax {
			break
		}
		if !lifeFree(gu) || g.inContest(gu) || gu.state == gIdle && gu.timer > 1000 { // 9999 = held by a party event
			continue
		}
		switch gu.state {
		case gIdle, gGoto, gLounge:
		default:
			continue
		}
		if gu.state == gGoto && gu.next == gDance {
			continue
		}
		if r.Float64() >= waveShare {
			continue
		}
		target := g.lvl.RandomIn(r, d.Dance[r.Intn(len(d.Dance))])
		path := g.lvl.PathG(gu.pos, target)
		if path == nil {
			continue
		}
		gu.path, gu.state, gu.next, gu.dur = path, gGoto, gDance, 8+r.Float64()*8
		gu.hurry = false
		n++
	}
	return n
}

// djBoothUse: the player pressed E next to the booth at b.
func (g *Game) djBoothUse(b Vec) {
	ls := &g.life
	r := lifeDice()
	if g.djDead { // nobody stops him: the old behaviour (the song changes) plus the dancers
		g.skipSong = true
		g.snd.Play("spark")
		g.say("Cambias la canción en la cabina del DJ")
		g.lifeSongChanged()
		return
	}
	if g.time < ls.djCool {
		g.say("El DJ está liado con los discos")
		return
	}
	ls.djCool = g.time + djCoolTime
	oops := r.Float64() < djOopsChance
	g.skipSong = true
	g.snd.Play("spark")
	if oops {
		ls.oopsNew = g.time
		g.say("Uy: el DJ se equivoca de disco")
	} else {
		g.say("El DJ te hace caso y cambia la canción")
	}
	// the ones close to the booth react
	lines := []string{"¡ESA SÍ!", "¡POR FIN!", "¡SUBE ESO!"}
	if oops {
		lines = []string{"¿QUÉ ES ESTO?", "¿UN VALS?", "¡PONED OTRA!"}
	}
	k := 0
	for _, gu := range g.guests {
		if k >= 3 {
			break
		}
		if lifeFree(gu) && gu.icon == "" && dist(gu.pos, b) < 110 {
			gu.icon, gu.iconT = lines[k%len(lines)], 2.2
			k++
		}
	}
	g.lifeSongChanged()
}

// lifeOops: the synthesised song that plays now is the DJ's mistake (a waltz in the middle of the party).
// It is called by syncMusic when the song has just changed on request (skip) or not (timer).
func (g *Game) lifeOopsOnChange(skip bool) {
	ls := &g.life
	ls.oops = skip && g.time-ls.oopsNew < 1.5
	ls.oopsNew = -999
}

func lifeOopsSpec() songSpec {
	return songSpec{style: "clasica", bpm: genreSongs["clasica"].bpm, variant: 0, seed: 4242, bars: 8}
}

// ---------------------------------------------------------------------------
// 2. SELFIES
// ---------------------------------------------------------------------------

func (g *Game) lifeSelfies(dt float64) {
	ls := &g.life
	r := lifeDice()
	// running ones
	keep := ls.selfies[:0]
	for _, s := range ls.selfies {
		ok := g.time-s.t0 < selfieDur
		for _, gu := range s.gs {
			if gu.out() || gu.state != gIdle || gu.moving {
				ok = false
			}
		}
		if ok {
			if t := g.time - s.t0; t >= selfieFlashAt && t < selfieFlashAt+dt*1.5 && s.gs[0].icon == "" && r.Float64() < 0.6 {
				s.gs[0].icon, s.gs[0].iconT = "¡PATATA!", 1.2
			}
			keep = append(keep, s)
		}
	}
	ls.selfies = keep
	if g.time < ls.selfieT || len(ls.selfies) >= selfieMax {
		return
	}
	ls.selfieT = g.time + selfieEvery + r.Float64()*selfieJit
	if r.Float64() >= selfieChance {
		return
	}
	var cand []*Guest
	for _, gu := range g.guests {
		if lifeFree(gu) && gu.state == gIdle && !gu.moving && gu.quirk == "" && gu.timer > 0 && !g.inSelfie(gu) && !g.inContest(gu) {
			cand = append(cand, gu)
		}
	}
	if len(cand) == 0 {
		return
	}
	lead := cand[r.Intn(len(cand))]
	s := selfie{gs: []*Guest{lead}, t0: g.time}
	if r.Float64() < selfieGroupP {
		for _, gu := range cand {
			if gu != lead && len(s.gs) < 4 && dist(gu.pos, lead.pos) < selfieGroupNear {
				s.gs = append(s.gs, gu)
			}
		}
	}
	for _, gu := range s.gs {
		gu.dir = DirDown
		if gu.timer < selfieDur+0.4 {
			gu.timer = selfieDur + 0.4
		}
	}
	ls.selfies = append(ls.selfies, s)
}

func (g *Game) inSelfie(gu *Guest) bool {
	for _, s := range g.life.selfies {
		for _, m := range s.gs {
			if m == gu {
				return true
			}
		}
	}
	return false
}

func (g *Game) inContest(gu *Guest) bool {
	if c := g.life.contest; c != nil {
		for _, m := range c.gs {
			if m == gu {
				return true
			}
		}
	}
	return false
}

// plotFn paints a w*h rectangle of pixels: the game draws with fillRect, the tests into an image.
type plotFn func(x, y, w, h int, c color.RGBA)

func (g *Game) plotter(dst *ebiten.Image) plotFn {
	return func(x, y, w, h int, c color.RGBA) { fillRect(dst, float64(x), float64(y), float64(w), float64(h), c) }
}

// selfieArmAt returns the arm (shoulder and hand, screen pixels) and the phone rectangle for a selfie at time t.
func selfiePose(cx, cy int, sc, t float64) (sx, sy, hx0, hy0 int, out float64) {
	k := math.Min(1, t/0.4)
	if t > selfieDur-0.3 {
		k = math.Max(0, (selfieDur-t)/0.3)
	}
	px := func(v float64) int { return int(math.Round(v * sc)) }
	sx, sy = cx+px(3), cy-px(23)
	hx0, hy0 = sx+int(math.Round(float64(px(6))*k)), sy-int(math.Round(float64(px(11))*k))
	return sx, sy, hx0, hy0, k
}

// drawSelfiePix paints the raised arm, the phone and, when it is the moment, the flash.
func drawSelfiePix(plot plotFn, cx, cy int, sc, t float64, skin color.RGBA) {
	sx, sy, hx0, hy0, k := selfiePose(cx, cy, sc, t)
	if k <= 0 {
		return
	}
	// the arm: a 1 px (2 px when big) staircase from the shoulder to the hand
	w := 1
	if sc >= 1.5 {
		w = 2
	}
	steps := maxI(absI(hx0-sx), absI(hy0-sy))
	for i := 0; i <= steps; i++ {
		x := sx + (hx0-sx)*i/maxI(steps, 1)
		y := sy + (hy0-sy)*i/maxI(steps, 1)
		plot(x, y, w, w, skin)
	}
	// the phone: dark, with a light lens
	plot(hx0-1, hy0-4, 3, 5, rgb(0x16161e))
	plot(hx0, hy0-3, 1, 1, rgb(0x8ad0f0))
	if t >= selfieFlashAt && t < selfieFlashAt+selfieFlashLen {
		fc, hl := rgb(0xfffbe0), rgb(0xffffff)
		fx, fy := hx0, hy0-3
		plot(fx-5, fy, 11, 1, fc)
		plot(fx, fy-5, 1, 11, fc)
		plot(fx-1, fy-1, 3, 3, hl)
		plot(fx-3, fy-3, 1, 1, fc)
		plot(fx+3, fy-3, 1, 1, fc)
		plot(fx-3, fy+3, 1, 1, fc)
		plot(fx+3, fy+3, 1, 1, fc)
	}
}

func (g *Game) lifeScale() float64 {
	if g.charSc > 0 {
		return g.charSc
	}
	return 1
}

// ---------------------------------------------------------------------------
// 3. DANCE CONTEST
// ---------------------------------------------------------------------------

func (g *Game) lifeContest(dt float64) {
	ls := &g.life
	r := lifeDice()
	c := ls.contest
	if c == nil {
		if ls.contestAt > 0 && g.time >= ls.contestAt {
			ls.contestAt, ls.contestRan = 0, true
			g.contestStart(r)
		}
		return
	}
	// who is still there
	keep := c.gs[:0]
	var pts []Vec
	for i, gu := range c.gs {
		if gu.out() || !lifeFree(gu) {
			continue
		}
		switch gu.state {
		case gGoto, gDance, gIdle:
		default:
			continue
		}
		keep = append(keep, gu)
		pts = append(pts, c.pts[i])
	}
	c.gs, c.pts = keep, pts
	if len(c.gs) < 3 {
		g.contestEnd()
		return
	}
	age := g.time - c.t0
	switch c.phase {
	case ctGather:
		arrived := 0
		for _, gu := range c.gs {
			if gu.state == gDance {
				arrived++
			}
		}
		if arrived == len(c.gs) || age > contestGather {
			c.phase, c.t0, c.nextIc = ctDance, g.time, g.time+1
			for _, gu := range c.gs {
				if gu.state != gDance {
					gu.path, gu.state, gu.timer = nil, gDance, 3
				}
			}
			g.say("¡Concurso de baile en la pista!")
		}
	case ctDance:
		for _, gu := range c.gs {
			gu.state, gu.timer, gu.dir = gDance, 3, faceTo(gu.pos, c.center)
		}
		if g.time >= c.nextIc {
			c.nextIc = g.time + 1.6
			lines := []string{"¡OLÉ!", "¡TOMA ESO!", "¡MIRAD!", "¡WOO!"}
			gu := c.gs[r.Intn(len(c.gs))]
			if gu.icon == "" {
				gu.icon, gu.iconT = lines[r.Intn(len(lines))], 1.4
			}
		}
		if age >= contestDance {
			c.phase, c.t0 = ctWin, g.time
			c.winner = c.gs[r.Intn(len(c.gs))]
			c.winner.icon, c.winner.iconT = "¡HE GANADO!", 3
			k := 0
			for _, gu := range c.gs {
				if gu != c.winner && k < 2 {
					gu.icon, gu.iconT = []string{"¡FRAUDE!", "BUUU"}[k], 2.4
					k++
				}
			}
			g.say("¡Hay ganador en el concurso de baile!")
		}
	case ctWin:
		for _, gu := range c.gs {
			gu.state, gu.timer = gDance, 3
		}
		if age >= contestWin {
			g.contestEnd()
		}
	}
}

func (g *Game) contestEnd() {
	c := g.life.contest
	g.life.contest = nil
	if c == nil {
		return
	}
	r := lifeDice()
	for _, gu := range c.gs {
		if !gu.out() && gu.state == gDance {
			gu.state, gu.timer, gu.path = gIdle, 1+r.Float64()*2, nil
		}
	}
}

func (g *Game) contestStart(r *rand.Rand) bool {
	d := g.lvl.Def
	if len(d.Dance) == 0 {
		return false
	}
	fl := d.Dance[0]
	for _, f := range d.Dance {
		if f.W*f.H > fl.W*fl.H {
			fl = f
		}
	}
	center := Vec{fl.X + fl.W/2, fl.Y + fl.H/2}
	var cand []*Guest
	for _, gu := range g.guests {
		if !lifeFree(gu) || gu.quirk != "" || g.inSelfie(gu) {
			continue
		}
		switch gu.state {
		case gIdle, gGoto, gDance, gLounge:
			cand = append(cand, gu)
		}
	}
	if len(cand) < contestMinN {
		return false
	}
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	n := contestMinN + r.Intn(contestMaxN-contestMinN+1)
	c := &danceContest{center: center, t0: g.time}
	rad := math.Max(14, math.Min(24, math.Min(fl.W, fl.H)/2-6))
	for _, gu := range cand {
		if len(c.gs) >= n {
			break
		}
		a := float64(len(c.gs)) / float64(n) * 2 * math.Pi
		pt := g.lvl.NearestFree(Vec{center.X + math.Cos(a)*rad, center.Y + math.Sin(a)*rad*0.7})
		path := g.lvl.PathG(gu.pos, pt)
		if path == nil && dist(gu.pos, pt) > 6 {
			continue
		}
		gu.path, gu.state, gu.next, gu.dur, gu.hurry = path, gGoto, gDance, 60, false
		c.gs, c.pts = append(c.gs, gu), append(c.pts, pt)
	}
	if len(c.gs) < contestMinN-1 {
		for _, gu := range c.gs { // not enough: let them go
			gu.state, gu.timer, gu.path = gIdle, 1, nil
		}
		return false
	}
	g.life.contest = c
	return true
}

// drawCrown: a little gold crown over the winner.
func drawCrownPix(plot plotFn, cx, top int) {
	gold, dark := rgb(0xf4c430), rgb(0xb88a10)
	plot(cx-3, top+2, 7, 2, gold)
	plot(cx-3, top, 1, 2, gold)
	plot(cx, top-1, 1, 3, gold)
	plot(cx+3, top, 1, 2, gold)
	plot(cx-3, top+3, 7, 1, dark)
	plot(cx, top+1, 1, 1, rgb(0xd03030))
}

// ---------------------------------------------------------------------------
// 4. THE WRONG PIZZA BOY
// ---------------------------------------------------------------------------

func pizzaSpec() CharSpec {
	return CharSpec{
		Skin: hx(0xe0b090), Hair: hx(0x3a2a20), Shirt: hx(0xc82820), Shirt2: hx(0xc82820), Pants: hx(0x2a2a44), Shoes: hx(0x202020),
		Accent: hx(0xf4f0e0), HatCol: hx(0xc82820), Hat: HatCap, Top: TopTee, noOverride: true,
	}
}

func (g *Game) lifePizza(dt float64) {
	ls := &g.life
	r := lifeDice()
	if ls.pizza == nil {
		if ls.pizzaAt > 0 && g.time >= ls.pizzaAt {
			ls.pizzaAt, ls.pizzaRan = 0, true
			g.pizzaEnter(r)
		}
		return
	}
	e := ls.pizza
	e.moving = false
	age := g.time - e.t0
	switch e.state {
	case pzEnter:
		m, arr := follow(&e.pos, &e.dir, &e.path, pizzaSpeed, dt)
		e.moving = m
		if arr {
			e.state, e.timer, e.moveAt = pzAsk, 0, g.time+7
		}
	case pzAsk:
		if len(e.path) > 0 {
			m, _ := follow(&e.pos, &e.dir, &e.path, pizzaSpeed, dt)
			e.moving = m
		} else if g.time >= e.moveAt {
			e.moveAt = g.time + 7 + r.Float64()*4
			e.path = g.lvl.PathG(e.pos, g.lvl.RandomRoam(r))
		}
		if age > pizzaWaitMax {
			g.pizzaLeave()
		}
	case pzGift:
		e.timer -= dt
		if e.timer <= 0 {
			g.pizzaLeave()
		}
	case pzLeave:
		m, arr := follow(&e.pos, &e.dir, &e.path, pizzaSpeed*1.15, dt)
		e.moving = m
		if arr {
			ls.pizza = nil
		}
	}
	if e.moving {
		e.anim += dt * 7
	}
}

func (g *Game) pizzaLeave() {
	e := g.life.pizza
	e.state = pzLeave
	e.path = g.lvl.PathG(e.pos, e.entry)
	if e.path == nil {
		g.life.pizza = nil
	}
}

func (g *Game) pizzaEnter(r *rand.Rand) bool {
	// he comes in at the roaming point farthest from the player
	var entry Vec
	best := -1.0
	for i := 0; i < 8; i++ {
		p := g.lvl.RandomRoam(r)
		if d := dist(p, g.p.pos); d > best {
			best, entry = d, p
		}
	}
	entry = g.lvl.NearestFree(entry)
	dest := g.lvl.RandomRoam(r)
	path := g.lvl.PathG(entry, dest)
	if path == nil {
		return false
	}
	sp := pizzaSpec()
	g.life.pizza = &pizzaBoy{pos: entry, entry: entry, dir: DirDown, cs: newCharSet(&sp), path: path, state: pzEnter, t0: g.time}
	return true
}

// pizzaLine is what he says now (the bubble over him).
func pizzaLine(e *pizzaBoy, helped bool) string {
	switch e.state {
	case pzGift:
		return "¡Gracias! Toma, invita la casa"
	case pzLeave:
		if helped {
			return "¡Buen provecho!"
		}
		return "Mejor llamo al cliente..."
	case pzAsk:
		if int(e.anim/9)%2 == 0 || e.moving {
			return "Pizza para el 14B... ¿alguien?"
		}
		return "¿Es aquí el 14B?"
	}
	return "¿Es aquí el 14B?"
}

// pizzaHelp: the player (p) helps him. Full stamina and a few points.
func (g *Game) pizzaHelp(p *Player) {
	e := g.life.pizza
	g.life.pizzaHelp = true
	e.state, e.timer, e.path = pzGift, 2.6, nil
	e.dir = faceTo(e.pos, p.pos)
	p.stamina = math.Max(p.stamina, g.perkStamina)
	g.award(e.pos, pizzaStamina, "")
	g.snd.Play("spark")
	g.say("Le indicas el camino y te regala una pizza")
}

func (g *Game) pizzaKill() {
	e := g.life.pizza
	if e == nil {
		return
	}
	g.life.pizza, g.life.pizzaDead = nil, true
	g.killByPlayer = true
	g.spawnBody(e.pos, e.cs, e.dir)
	g.bodies[len(g.bodies)-1].label = "Repartidor"
	g.killByPlayer = false
	g.stat("pizza_boy", 1)
	g.snd.Play("stab")
	g.kills++
	g.countKill(kcKnife, false)
	g.points += g.killVal(20)
	g.witness()
	g.checkDone()
}

// pizza box carried in front of the chest (screen pixels).
func drawPizzaBoxPix(plot plotFn, cx, cy int, dir Dir, sc float64) {
	k := func(v int) int { return int(math.Round(float64(v) * sc)) }
	x := cx - k(5)
	switch dir {
	case DirLeft:
		x = cx - k(10)
	case DirRight:
		x = cx + k(1)
	}
	y := cy - k(17)
	w := k(10)
	plot(x, y, w, k(3), rgb(0xf0e8d0))
	plot(x, y, w, 1, rgb(0xfffaf0))
	plot(x, y+k(3)-1, w, 1, rgb(0xa89870))
	plot(x+k(1), y+k(1), k(3), 1, rgb(0xc82820))
	plot(x+k(6), y+k(1), k(3), 1, rgb(0xc82820))
	plot(x, y-k(2), w, k(2), rgb(0xd8c8a0)) // the lid, tilted up
	plot(x, y-k(2), w, 1, rgb(0xf0e8d0))
}

// ---------------------------------------------------------------------------
// 5. THE MASCOT
// ---------------------------------------------------------------------------

// mascotKindFor: the mascot of a kind of level, by the title.
func mascotKindFor(title string) string {
	t := strings.ToUpper(title)
	has := func(ws ...string) bool {
		for _, w := range ws {
			if strings.Contains(t, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("PLAYA", "MIAMI", "CRUCERO", "YACHT", "YATE", "ISLA", "BEACH"):
		return "crab"
	case has("ESQUÍ", "ESQUI", "SKI", "NIEVE", "SNOW", "FRÍO"):
		return "penguin"
	case has("PARQUE", "PLAZA EUROPEA", "FESTIVAL", "CIRCUITO", "ESTADIO", "SKATE", "PISCINA", "JARDÍN"):
		return "duck"
	}
	return "cat"
}

func (g *Game) mascotSpawn(r *rand.Rand) {
	if len(g.lvl.Def.Roam) == 0 {
		return
	}
	var pos Vec
	for i := 0; i < 12; i++ {
		pos = g.lvl.NearestFree(g.lvl.RandomRoam(r))
		if g.p == nil || dist(pos, g.p.pos) > 70 {
			break
		}
	}
	g.life.mascot = &mascot{kind: mascotKindFor(g.lvl.Def.Title), pos: pos, dir: DirRight, timer: 2 + r.Float64()*3}
}

func (g *Game) lifeMascot(dt float64) {
	m := g.life.mascot
	if m == nil || m.dead {
		return
	}
	r := lifeDice()
	m.moving = false
	if len(m.path) > 0 {
		mv, _ := follow(&m.pos, &m.dir, &m.path, mascotSpeed, dt)
		m.moving = mv
	} else {
		m.timer -= dt
		if m.timer <= 0 {
			m.timer = 3 + r.Float64()*6
			if r.Float64() < 0.7 {
				// a short stroll near where it is
				t := Vec{m.pos.X + (r.Float64()*2-1)*60, m.pos.Y + (r.Float64()*2-1)*40}
				if g.lvl.Free(t) {
					m.path = g.lvl.PathG(m.pos, t)
				}
			}
		}
	}
	if m.moving {
		m.anim += dt * 8
	}
}

var mascotName = map[string]string{
	"cat": "gato", "crab": "cangrejo", "penguin": "pingüino", "duck": "pato",
}

var mascotPetLine = map[string]string{
	"cat":     "Acaricias al gato de la fiesta",
	"crab":    "Saludas al cangrejo de la fiesta",
	"penguin": "Acaricias al pingüino de la fiesta",
	"duck":    "Alimentas al pato de la fiesta",
}

func (g *Game) mascotKill() {
	m := g.life.mascot
	if m == nil || m.dead {
		return
	}
	m.dead = true
	g.life.mascotDead = true
	g.decals = append(g.decals, Decal{pos: m.pos, kind: 0, r: 4})
	g.burst(Vec{m.pos.X, m.pos.Y - 4}, 8, 50, rgb(0xb01818), 0.5, 2, 200)
	g.snd.Play("stab")
	g.say("La mascota de la fiesta ya no está")
}

// lifeFinish: the party is completed. A mascot that is alive is a bonus (called before finishScore, so it is in the base).
func (g *Game) lifeFinish() int {
	m := g.life.mascot
	if m == nil || m.dead || g.stealth {
		return 0
	}
	g.points += mascotBonus
	return mascotBonus
}

// lifeInteract: E next to the pizza boy or the mascot.
func (g *Game) lifeInteract(p *Player) bool {
	if !g.lifeActive() || p.carrying != nil || p.dead {
		return false
	}
	ls := &g.life
	if e := ls.pizza; e != nil && (e.state == pzEnter || e.state == pzAsk) && dist(p.pos, e.pos) < pizzaReach {
		g.pizzaHelp(p)
		return true
	}
	if m := ls.mascot; m != nil && !m.dead && dist(p.pos, m.pos) < mascotReach {
		if !m.petted {
			m.petted = true
			g.award(m.pos, mascotPet, "")
			g.say(mascotPetLine[m.kind])
		} else {
			g.say(mascotPetLine[m.kind])
		}
		return true
	}
	return false
}

// lifeStab: the player's blow reaches the pizza boy or the mascot (the nearest, within maxD).
func (g *Game) lifeStab(p *Player, maxD float64) bool {
	if !g.lifeActive() {
		return false
	}
	ls := &g.life
	dp, dm := math.Inf(1), math.Inf(1)
	if ls.pizza != nil {
		dp = dist(p.pos, ls.pizza.pos)
	}
	if ls.mascot != nil && !ls.mascot.dead {
		dm = dist(p.pos, ls.mascot.pos)
	}
	lim := math.Min(maxD, lifeStabReach)
	switch {
	case dp <= lim && dp <= dm:
		g.pizzaKill()
		return true
	case dm <= lim:
		g.mascotKill()
		return true
	}
	return false
}

// ---- mascot sprites (pixel rows, drawn facing right; '.' is empty) ----

var mascotRows = map[string][]string{
	"cat": {
		".........a.a",
		".........bbb",
		"..bbbbb.bwbb",
		".bbbbbbbbbbn",
		"tbbbbbbbbbb.",
		"tbbbbbbbbb..",
		"t.bb..bb....",
		"..dd..dd....",
	},
	"crab": {
		"rr..e.e..rr",
		"r...e.e...r",
		".rrrrrrrrr.",
		"..rrrrrrr..",
		".l.l.l.l.l.",
	},
	"penguin": {
		"..kkk..",
		".kkkkk.",
		".kwkwk.",
		"..ooo..",
		".kwwwk.",
		"kkwwwkk",
		"kkwwwkk",
		".kwwwk.",
		"..o.o..",
	},
	"duck": {
		".....yy..",
		"....yeyoo",
		"....yyyo.",
		"w..yyyy..",
		"wyyyyyyy.",
		".yyyyyyy.",
		"..yyyyy..",
		"...o.o...",
	},
}

var mascotPal = map[string]map[byte]color.RGBA{
	"cat":     {'a': rgb(0x9a5a20), 'b': rgb(0xd88a3a), 't': rgb(0xb06a28), 'w': rgb(0x203020), 'n': rgb(0xe8a0a0), 'd': rgb(0xf4e8d0)},
	"crab":    {'r': rgb(0xd83a28), 'e': rgb(0x181818), 'l': rgb(0xa02418)},
	"penguin": {'k': rgb(0x20242c), 'w': rgb(0xf4f4f0), 'o': rgb(0xf0a020)},
	"duck":    {'y': rgb(0xf4d430), 'e': rgb(0x181818), 'o': rgb(0xf08a20), 'w': rgb(0xf4f0c0)},
}

// drawMascotPix paints the mascot with its feet at (cx, cy). flip mirrors it (looking left). hop lifts it a pixel.
func drawMascotPix(plot plotFn, kind string, cx, cy int, flip bool, hop int, px int) {
	rows := mascotRows[kind]
	pal := mascotPal[kind]
	if len(rows) == 0 {
		return
	}
	w := len(rows[0])
	x0 := cx - w*px/2
	y0 := cy - len(rows)*px - hop*px
	// a flat shadow
	plot(x0+px, cy-1, (w-2)*px, 1, color.RGBA{0, 0, 0, 70})
	for j, row := range rows {
		for i := 0; i < len(row); i++ {
			ch := row[i]
			c, ok := pal[ch]
			if !ok {
				continue
			}
			ii := i
			if flip {
				ii = w - 1 - i
			}
			plot(x0+ii*px, y0+j*px, px, px, c)
		}
	}
}

// ---------------------------------------------------------------------------
// drawing: everything goes into the y-sorted list of the world
// ---------------------------------------------------------------------------

func (g *Game) lifeItems(dst *ebiten.Image, items []drawItem) []drawItem {
	if !g.lifeActive() {
		return items
	}
	ls := &g.life
	plot := g.plotter(dst)
	sc := g.lifeScale()
	for _, s := range ls.selfies {
		s := s
		lead := s.gs[0]
		items = append(items, drawItem{lead.pos.Y + 0.1, func() {
			drawSelfiePix(plot, int(math.Round(g.wx(lead.pos.X))), int(math.Round(g.wy(lead.pos.Y))), sc, g.time-s.t0, lead.cs.Spec.Skin)
		}})
	}
	if c := ls.contest; c != nil && c.phase == ctWin && c.winner != nil && !c.winner.out() {
		w := c.winner
		items = append(items, drawItem{w.pos.Y + 0.1, func() {
			bob := int(math.Floor(math.Mod(g.time*5, 2)))
			drawCrownPix(plot, int(math.Round(g.wx(w.pos.X))), int(math.Round(g.wy(w.pos.Y)-47*sc))-bob)
		}})
	}
	if e := ls.pizza; e != nil {
		items = append(items, drawItem{e.pos.Y, func() {
			anim, ph := AnimIdle, 0.0
			if e.moving {
				anim, ph = AnimWalk, e.anim
			}
			g.drawActor(dst, e.cs, anim, ph, e.pos, e.dir, 0, false)
			if e.state != pzLeave || !g.life.pizzaHelp {
				drawPizzaBoxPix(plot, int(math.Round(g.wx(e.pos.X))), int(math.Round(g.wy(e.pos.Y))), e.dir, sc)
			}
			g.bubble(e, bubGuest, e.pos, pizzaLine(e, g.life.pizzaHelp), false, 0)
		}})
	}
	if m := ls.mascot; m != nil && !m.dead {
		items = append(items, drawItem{m.pos.Y, func() {
			px := int(math.Max(1, math.Round(sc)))
			hop := 0
			if m.moving {
				hop = int(math.Floor(math.Mod(m.anim, 2)))
			}
			drawMascotPix(plot, m.kind, int(math.Round(g.wx(m.pos.X))), int(math.Round(g.wy(m.pos.Y))), m.dir == DirLeft, hop, px)
		}})
	}
	return items
}
