package main

import (
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// VENUE ACTS (v1.4.0, tanda H): little scenes that belong to ONE kind of party. They are NOT party events of the frame
// (the mean of events per party does not go up, TestPartyEventsFrequencyPerParty): like the roadworks they are part of
// the level, with their own dice (vaDice, never g.rng) and their own cooldowns.
//
//   - BARCO PIRATA: duels of insults between two guests (at most vaInsultCap at once: one insults, the other answers
//     with the right comeback, and back) and sword duels (sparks, the loser gives up). The ship's guards are pirates
//     (pirate_crew.go).
//   - ESTACIÓN DE ESQUÍ: guests on skis or a board gliding around, snowball fights, guests building a snowman (it stays
//     there), and once per party at most a GIANT SNOWBALL that rolls down from the top of the map and runs over the
//     guests in its way (vaBigBall).
//   - FIESTA DE CUMPLEAÑOS: a troupe of clowns (dressed at the start) juggling, making balloon animals, squirting
//     water from the flower of the lapel at a guest, and a tiny clown car driving up and down.
//   - PISTA DE HOCKEY: hockey players (dressed at the start) passing the puck to each other and to guests, and now and
//     then a slapshot at a guest: it hurts, and some die of it (at most vaPuckKills per party).
//
// Hooks: game.go (updateVenueActs, right after the roadworks: BEFORE the guests, like every held state) and draw.go
// (venueItems, with the other items). Nobody is added: the dressed guests are plain guests of the party (caps of
// cap.go untouched), they can be killed and they are witnesses.
// ---------------------------------------------------------------------------

const (
	vaQuirk      = "venueact" // a dressed guest of the venue (clown, hockey player, skier): the party events leave him alone
	vaInsultCap  = 2          // insult duels at once on the ship
	vaPuckKills  = 3          // deaths by slapshot in one party at most
	vaPuckDeath  = 0.2        // chance that a puck that hits a guest kills him
	vaBallRadius = 18.0       // the giant snowball
	vaBallSpeed  = 95.0       // px/s down the slope
	vaWalkMax    = 14.0       // seconds to reach the place of an act, else it is dropped
)

const (
	vaInsult = iota + 1
	vaSword
	vaSnowball
	vaSnowman
	vaSki
	vaJuggle
	vaBalloon
	vaSquirt
	vaPass
	vaShot
	vaMiniDance
)

var (
	vaRng  *rand.Rand
	vaSeed int64 // tests: non-zero = deterministic dice
)

func vaDice() *rand.Rand {
	if vaRng == nil {
		sd := vaSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x5eaf00d
		}
		vaRng = rand.New(rand.NewSource(sd))
	}
	return vaRng
}

// vaProj: something flying or sliding between two points (snowball, puck, water), a function of the time.
type vaProj struct {
	on            bool
	from, to      Vec
	t0, dur, arc  float64
	kind, hitWhat int
}

func (p *vaProj) at(now float64) (Vec, float64, bool) {
	k := (now - p.t0) / p.dur
	if k < 0 || k > 1 {
		return Vec{}, 0, false
	}
	pos := Vec{p.from.X + (p.to.X-p.from.X)*k, p.from.Y + (p.to.Y-p.from.Y)*k}
	return pos, p.arc * 4 * k * (1 - k), true
}

type vaAct struct {
	kind   int
	a, b   pevMem
	b2     bool // the act has a second guest
	met    bool
	walkT0 float64
	t      float64 // time in the act proper
	until  float64
	step   int
	pair   int
	next   float64
	proj   vaProj
	spot   Vec
	victim *Guest
	done   bool
}

type vaSnowFig struct {
	pos Vec
	k   float64 // 0..1 how much of it is built
}

type vaBall struct {
	on, done bool
	at       float64 // game time it starts rolling (0 = this party has none)
	x, y     float64
	r        float64
	killed   int
}

type vaRun struct {
	lvl       *Level
	kind      string // "pirate", "snow", "cake", "hockey"
	started   bool
	acts      []*vaAct
	nextT     float64
	roles     []*Guest // the dressed guests (clowns, players, skiers)
	skiers    map[*Guest]int
	snowmen   []vaSnowFig
	ball      vaBall
	puckKills int
	car       struct {
		on     bool
		cx, cy float64
		span   float64
	}
}

var va *vaRun

// vaKindOf: which acts this level has ("" = none).
func vaKindOf(def *LevelDef) string {
	if def == nil {
		return ""
	}
	switch def.Icon {
	case "pirate", "snow", "cake", "hockey", "disco70":
		return def.Icon
	}
	if pirateBay(def) {
		return "pirate"
	}
	return ""
}

// vaActive: the run of the level being played (nil = this party has no acts).
func (g *Game) vaActive() *vaRun {
	if g.lvl == nil {
		return nil
	}
	if va == nil || va.lvl != g.lvl {
		va = nil
		if k := vaKindOf(g.lvl.Def); k != "" {
			va = &vaRun{lvl: g.lvl, kind: k, skiers: map[*Guest]int{}}
		}
	}
	return va
}

func (g *Game) vaFreeRole(gu *Guest) bool {
	if gu == nil || gu.quirk != vaQuirk || !lifeFree(gu) || gu.armed || gu.leader != nil || gu.partner != nil ||
		g.inSelfie(gu) || g.inContest(gu) || g.pevTaken(gu) || va.busy(gu) {
		return false
	}
	switch gu.state {
	case gIdle, gGoto, gDance, gLounge:
		return gu.timer < 1000
	}
	return false
}

func (run *vaRun) busy(gu *Guest) bool {
	for _, a := range run.acts {
		if a.a.gu == gu || a.b2 && a.b.gu == gu {
			return true
		}
	}
	return false
}

// pevHas: the party events must not take a guest of an act (pevTaken asks every running event; the acts are asked
// through vaBusy in vaCanTake instead, and the guests of an act are held: pevFreeGuest sees timer 9999).
func (g *Game) vaCanTake(gu *Guest) bool {
	return g.pevFreeGuest(gu) && !va.busy(gu)
}

// vaStart: at the start of the party: the pirate guards, the dressed troupe, the giant snowball's moment, the car.
func (g *Game) vaStart(run *vaRun) {
	run.started = true
	r := vaDice()
	if fastEv {
		println("venue acts:", run.kind, "t", int(g.time))
	}
	run.nextT = g.time + 3 + r.Float64()*4
	if fastEv {
		run.nextT = g.time + 0.5
	}
	switch run.kind {
	case "pirate":
		g.pirateGuards()
	case "cake":
		g.vaDress(run, 9, clownSpec)
		g.vaCarPlace(run, r)
	case "hockey":
		g.vaDress(run, 7, hockeyPlayerSpec)
	case "snow":
		n := g.vaDress(run, 5, skierSpec)
		for i, gu := range run.roles {
			run.skiers[gu] = i % 2 // 0 skis, 1 snowboard
		}
		_ = n
		// the giant snowball: once per party at most, two parties in three, somewhere between 1 and 3 minutes in
		if r.Float64() < 0.67 || fastEv {
			run.ball.at = g.time + 60 + r.Float64()*120
			if fastEv {
				run.ball.at = g.time + 6
			}
		}
	}
}

// vaDress turns n plain guests into the venue's troupe (their own face, the clothes of the act).
func (g *Game) vaDress(run *vaRun, n int, spec func(*rand.Rand, int) CharSpec) int {
	r := vaDice()
	got := 0
	for _, gu := range g.guests {
		if got >= n {
			break
		}
		if !g.rwCanTake(gu) {
			continue
		}
		sp := spec(r, got)
		gu.cs = newCharSet(&sp)
		gu.quirk, gu.quirkInit = vaQuirk, true
		run.roles = append(run.roles, gu)
		got++
	}
	return got
}

// clownSpec: a clown of the birthday party: a red or orange afro, white face, a red nose (the mask colour), polka
// dots, baggy trousers and big shoes.
func clownSpec(r *rand.Rand, i int) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Skin = hx(0xf8f4f0)
	s.Hair, s.Hair2 = hx(pick(r, []uint32{0xe03a2a, 0xf08a20, 0x3aa0e0, 0x50c050})), HairAfro
	s.Hat = HatNone
	if i%3 == 1 {
		s.Hat, s.HatCol = HatPartyCone, hx(pick(r, []uint32{0xf0d030, 0xe880c0, 0x40a0e0}))
	}
	s.Top, s.Pattern = TopLong, 1
	s.Shirt, s.Shirt2 = hx(pick(r, []uint32{0xf0d030, 0xe04a5a, 0x40a0e0, 0x60c070})), hx(pick(r, []uint32{0xf4f4f4, 0xe04a5a, 0x2a2a6a}))
	s.Bottom, s.Pants = BotPants, hx(pick(r, []uint32{0x3a5ad0, 0xe04a5a, 0x6a3a9a, 0x2aa060}))
	s.Shoes = hx(0xe02a2a)
	s.Glasses, s.Beard, s.Carry, s.Weapon, s.Ride = GlassNone, BeardNone, HoldNone, HoldNone, 0
	s.Face = 1
	if i%4 == 2 {
		s.Carry, s.CarryCol = HoldBalloons, hx(pick(r, []uint32{0xe04a5a, 0x40a0e0, 0xf0d030}))
	}
	return s
}

// hockeyPlayerSpec: a player of one of the two teams, helmet and stick.
func hockeyPlayerSpec(r *rand.Rand, i int) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	team := []uint32{0xc03840, 0x2c5aa8}[i%2]
	s.Top, s.Shirt, s.Shirt2, s.Pattern = TopLong, hx(team), hx(0xf4f4f8), 2
	s.Bottom, s.Pants = BotPants, hx(0x20222c)
	s.Shoes = hx(0xe8eef4)
	s.Hat, s.HatCol = HatHelmet, hx(team)
	s.Body = BodyBig
	s.Carry, s.Weapon, s.Ride, s.Glasses = HoldStick, HoldNone, 0, GlassNone
	return s
}

// skierSpec: a skier (beanie, goggles, a padded jacket) or a snowboarder (baggy and bright); the skis or the board
// are drawn under his feet while he glides (vaDrawSki).
func skierSpec(r *rand.Rand, i int) CharSpec {
	s := genericSpec(r)
	s.Name, s.DisplayName = "", ""
	s.Top, s.Shirt, s.Shirt2 = TopLong, hx(pick(r, []uint32{0xe04a3a, 0x2a8ad0, 0xf0c020, 0x30b070, 0xe060b0})), hx(0x20222c)
	s.Bottom, s.Pants = BotPants, hx(pick(r, []uint32{0x20222c, 0x3a3a5a, 0xf4f4f4}))
	s.Hat, s.HatCol = HatBeanie, hx(pick(r, []uint32{0xe04a3a, 0xf4f4f4, 0x2a2a30}))
	s.Glasses = GlassBig
	s.Carry, s.Weapon, s.Ride = HoldNone, HoldNone, 0
	return s
}

// updateVenueActs: game.go, right after the roadworks (before the guests: the held states are re-asserted here).
func (g *Game) updateVenueActs(dt float64) {
	if g.mode != ModePlay {
		return
	}
	run := g.vaActive()
	if run == nil {
		return
	}
	r := vaDice()
	if !run.started {
		g.vaStart(run)
	}
	out := run.acts[:0]
	for _, a := range run.acts {
		if g.vaStep(run, a, r, dt) {
			out = append(out, a)
			continue
		}
		g.vaEnd(a, r)
	}
	run.acts = out
	if g.time >= run.nextT {
		run.nextT = g.time + 5 + r.Float64()*7
		if fastEv {
			run.nextT = g.time + 1.5
		}
		g.vaSpawn(run, r)
	}
	g.vaBallStep(run, dt)
}

func (run *vaRun) count(kind int) int {
	n := 0
	for _, a := range run.acts {
		if a.kind == kind {
			n++
		}
	}
	return n
}

// vaSpawn: maybe start one more act of this party (each kind with its cap).
func (g *Game) vaSpawn(run *vaRun, r *rand.Rand) {
	type cand struct{ kind, cap int }
	var cs []cand
	switch run.kind {
	case "pirate":
		cs = []cand{{vaInsult, vaInsultCap}, {vaInsult, vaInsultCap}, {vaSword, 1}}
	case "snow":
		cs = []cand{{vaSki, 4}, {vaSnowball, 2}, {vaSnowman, 2}, {vaSnowman, 2}}
		if len(run.snowmen) >= 5 {
			cs = cs[:2]
		}
	case "cake":
		cs = []cand{{vaJuggle, 2}, {vaBalloon, 1}, {vaSquirt, 1}}
	case "hockey":
		cs = []cand{{vaPass, 2}, {vaPass, 2}, {vaShot, 1}}
	case "disco70": // guests go dancing on the small lit floors (disco70_mini.go)
		cs = []cand{{vaMiniDance, 9}}
	}
	if len(cs) == 0 {
		return
	}
	c := cs[r.Intn(len(cs))]
	if run.count(c.kind) >= c.cap {
		return
	}
	g.vaBegin(run, c.kind, r)
}

// vaNearSpot: a free place for a one-man act that the performer reaches in time: of a few random places (pevSpot),
// the one with the shortest walk, and only if it is well inside vaWalkMax (a place across the map was dropped on the
// way and the act never played, v1.4.0).
func (g *Game) vaNearSpot(r *rand.Rand, gu *Guest) (Vec, bool) {
	var best Vec
	bl, found := guestSpeed*vaWalkMax*0.7, false
	for i := 0; i < 6; i++ {
		p, ok := g.pevSpot(r, 24, false)
		if !ok {
			break
		}
		if dist(gu.pos, p) >= bl {
			continue
		}
		if n := pevPathLen(g.lvl.PathG(gu.pos, p), gu.pos); n < bl {
			best, bl, found = p, n, true
		}
	}
	return best, found
}

// vaPickPair: two guests close to each other (role guests for the troupe acts, any free guest otherwise).
func (g *Game) vaPickPair(run *vaRun, r *rand.Rand, roles bool, near float64) (*Guest, *Guest) {
	var pool []*Guest
	for _, gu := range g.guests {
		if roles && g.vaFreeRole(gu) || !roles && g.vaCanTake(gu) {
			pool = append(pool, gu)
		}
	}
	if len(pool) < 2 {
		return nil, nil
	}
	a := pool[r.Intn(len(pool))]
	var b *Guest
	bd := near
	for _, o := range pool {
		if o != a {
			if d := dist(o.pos, a.pos); d < bd {
				b, bd = o, d
			}
		}
	}
	return a, b
}

// vaBegin starts an act: the guests walk to their places first.
func (g *Game) vaBegin(run *vaRun, kind int, r *rand.Rand) {
	a := &vaAct{kind: kind, walkT0: g.time}
	var ga, gb *Guest
	gap := 26.0
	switch kind {
	case vaInsult, vaSword:
		ga, gb = g.vaPickPair(run, r, false, 260)
		a.pair = r.Intn(len(insultPairs))
	case vaSnowball:
		ga, gb = g.vaPickPair(run, r, false, 300)
		gap = 60
	case vaPass:
		ga, gb = g.vaPickPair(run, r, true, 400)
		gap = 90
	case vaSnowman, vaJuggle, vaBalloon, vaSki:
		var pool []*Guest
		for _, gu := range g.guests {
			ok := g.vaFreeRole(gu)
			if kind == vaSnowman {
				ok = g.vaCanTake(gu)
			}
			if kind == vaSki {
				_, sk := run.skiers[gu]
				ok = ok && sk
			}
			if ok {
				pool = append(pool, gu)
			}
		}
		if len(pool) == 0 {
			return
		}
		ga = pool[r.Intn(len(pool))]
	case vaSquirt, vaShot:
		// a troupe member and a plain guest close to him
		for _, gu := range g.guests {
			if g.vaFreeRole(gu) && (ga == nil || r.Intn(3) == 0) {
				ga = gu
			}
		}
		if ga == nil {
			return
		}
		bd := 200.0
		for _, gu := range g.guests {
			if gu != ga && g.vaCanTake(gu) {
				if d := dist(gu.pos, ga.pos); d < bd {
					gb, bd = gu, d
				}
			}
		}
		gap = 44
		if kind == vaShot {
			gap = 80
		}
	}
	if kind == vaMiniDance {
		var pool []*Guest
		for _, gu := range g.guests {
			if g.vaCanTake(gu) {
				pool = append(pool, gu)
			}
		}
		if len(pool) == 0 {
			return
		}
		ga = pool[r.Intn(len(pool))]
		m := discoMinis[r.Intn(len(discoMinis))].R
		p := Vec{m.X + 8 + r.Float64()*(m.W-16), m.Y + 10 + r.Float64()*(m.H-14)}
		if !g.lvl.Free(p) {
			return
		}
		a.a.gu, a.spot = ga, p
		if !g.pevSendTo(&a.a, p) {
			return
		}
		run.acts = append(run.acts, a)
		return
	}
	if ga == nil {
		return
	}
	a.a.gu = ga
	if kind == vaSki {
		a.until = g.time + 25 + r.Float64()*25
		a.met = true
		if !g.vaGlide(a, r) {
			return
		}
		run.acts = append(run.acts, a)
		return
	}
	if kind == vaSnowman || kind == vaJuggle || kind == vaBalloon {
		spot, ok := g.vaNearSpot(r, ga)
		if !ok || !g.pevSendTo(&a.a, spot) {
			return
		}
		a.spot = spot
		run.acts = append(run.acts, a)
		return
	}
	if gb == nil {
		return
	}
	a.b.gu, a.b2 = gb, true
	// the place: near the two of them if there is room (a short walk), else any open spot of the party
	rad := math.Min(gap/2+14, 38)
	mid, ok := Vec{}, false
	c := Vec{(ga.pos.X + gb.pos.X) / 2, (ga.pos.Y + gb.pos.Y) / 2}
	for i := 0; i < 24 && !ok; i++ {
		p := Vec{c.X + (r.Float64()*2-1)*float64(10+i*6), c.Y + (r.Float64()*2-1)*float64(8+i*4)}
		if g.pevClear(p, rad) && g.pevPlayerFar(p, 70) {
			mid, ok = p, true
		}
	}
	if !ok {
		if mid, ok = g.pevSpot(r, rad, false); !ok {
			return
		}
	}
	pa, pb := Vec{mid.X - gap/2, mid.Y}, Vec{mid.X + gap/2, mid.Y}
	if !g.lvl.Free(pa) || !g.lvl.Free(pb) {
		return
	}
	if !g.pevSendTo(&a.a, pa) {
		return
	}
	if !g.pevSendTo(&a.b, pb) {
		pevRelease(&a.a, r)
		return
	}
	a.spot = mid
	run.acts = append(run.acts, a)
}

// vaGlide sends a skier to another open place, quickly.
func (g *Game) vaGlide(a *vaAct, r *rand.Rand) bool {
	for i := 0; i < 12; i++ {
		p := g.lvl.RandomRoam(r)
		if dist(p, a.a.gu.pos) < 120 || !g.lvl.Outdoor(p) {
			continue
		}
		if g.pevSendTo(&a.a, p) {
			a.a.gu.hurry, a.a.gu.hurryT = true, g.time
			a.walkT0 = g.time
			return true
		}
	}
	return false
}

func vaArrived(m *pevMem) bool {
	gu := m.gu
	return m.want == gGoto && (gu.state == gIdle && gu.timer > 1000 || gu.path == nil && gu.state != gGoto)
}

// vaStep: false when the act is over (or broke off).
func (g *Game) vaStep(run *vaRun, a *vaAct, r *rand.Rand, dt float64) bool {
	if !a.a.mine() || a.b2 && !a.b.mine() {
		return false
	}
	if a.kind == vaSki {
		if g.time > a.until {
			return false
		}
		if vaArrived(&a.a) {
			if !g.vaGlide(a, r) {
				return false
			}
		}
		return true
	}
	if !a.met {
		ok := vaArrived(&a.a) && (!a.b2 || vaArrived(&a.b))
		if !ok {
			if g.time-a.walkT0 > vaWalkMax {
				return false
			}
			return true
		}
		a.met = true
		a.t, a.next = 0, g.time
		if a.b2 {
			a.a.gu.dir, a.b.gu.dir = faceTo(a.a.gu.pos, a.b.gu.pos), faceTo(a.b.gu.pos, a.a.gu.pos)
			a.a.pose(AnimIdle)
			a.b.pose(AnimIdle)
		} else {
			a.a.pose(AnimIdle)
		}
		a.until = g.time + map[int]float64{vaInsult: 99, vaSword: 6 + r.Float64()*3, vaSnowball: 9 + r.Float64()*5,
			vaSnowman: 13, vaJuggle: 10 + r.Float64()*6, vaBalloon: 7, vaSquirt: 5, vaPass: 10 + r.Float64()*6, vaShot: 4, vaMiniDance: 15 + r.Float64()*12}[a.kind]
		if a.kind == vaMiniDance {
			a.pair = AnimDance0 + r.Intn(AnimDance11-AnimDance0+1)
		}
	}
	a.a.keep()
	if a.b2 {
		a.b.keep()
	}
	a.t += dt
	switch a.kind {
	case vaInsult:
		return g.vaInsultStep(a)
	case vaSword:
		if g.time >= a.next {
			a.next = g.time + 0.32
			a.step++
			anims := []int{AnimStab, AnimPunch}
			a.a.gu.dAnim, a.b.gu.dAnim = anims[a.step%2], anims[(a.step+1)%2]
			mid := Vec{(a.a.gu.pos.X + a.b.gu.pos.X) / 2, a.a.gu.pos.Y - 22}
			g.vaSparks(mid, r, rgb(0xfff0a0), 5)
			if a.step == 2 {
				a.a.gu.icon, a.a.gu.iconT = pevPick(r, swordStart), 2.2
			}
		}
		if g.time >= a.until {
			loser, winner := a.b.gu, a.a.gu
			if r.Intn(2) == 0 {
				loser, winner = winner, loser
			}
			loser.icon, loser.iconT = pevPick(r, swordLose), 2.4
			winner.dAnim = AnimLaugh
			return false
		}
	case vaSnowball:
		g.vaThrowStep(a, r, 1.0)
		if g.time >= a.until {
			return false
		}
	case vaPass:
		g.vaThrowStep(a, r, 0.7)
		if g.time >= a.until {
			return false
		}
	case vaShot:
		return g.vaShotStep(run, a, r)
	case vaMiniDance:
		a.a.gu.dAnim = a.pair
		if g.time >= a.until {
			return false
		}
	case vaSnowman:
		a.a.gu.dAnim = []int{AnimIdle, AnimPunch}[int(a.t*1.5)%2]
		k := math.Min(1, a.t/12)
		if a.step == 0 {
			run.snowmen = append(run.snowmen, vaSnowFig{pos: Vec{a.spot.X + 16, a.spot.Y}})
			a.step = 1
			a.a.gu.dir = DirRight
			a.a.gu.icon, a.a.gu.iconT = pevPick(r, snowmanLines), 2.4
		}
		run.snowmen[len(run.snowmen)-1].k = math.Max(run.snowmen[len(run.snowmen)-1].k, k)
		if g.time >= a.until {
			run.snowmen[len(run.snowmen)-1].k = 1
			a.a.gu.dAnim = AnimLaugh
			return false
		}
	case vaJuggle:
		a.a.gu.dAnim = AnimIdle
		if a.step == 0 && a.t > 1 {
			a.step = 1
			a.a.gu.icon, a.a.gu.iconT = pevPick(r, juggleLines), 2.4
		}
		if g.time >= a.until {
			return false
		}
	case vaBalloon:
		a.a.gu.dAnim = []int{AnimTalk, AnimWave}[int(a.t)%2]
		if a.step == 0 && a.t > 1.5 {
			a.step = 1
			a.a.gu.icon, a.a.gu.iconT = pevPick(r, balloonLines), 2.6
		}
		if g.time >= a.until {
			return false
		}
	case vaSquirt:
		a.a.gu.dAnim = AnimPoint
		if a.step == 0 && a.t > 0.8 {
			a.step = 1
			a.proj = vaProj{on: true, from: Vec{a.a.gu.pos.X, a.a.gu.pos.Y - 24}, to: Vec{a.b.gu.pos.X, a.b.gu.pos.Y - 22}, t0: g.time, dur: 1.6, kind: 2}
			a.a.gu.icon, a.a.gu.iconT = pevPick(r, squirtLines), 2
		}
		if a.step == 1 && g.time > a.proj.t0+0.6 {
			a.step = 2
			a.b.gu.dAnim = AnimPanic
			a.b.gu.icon, a.b.gu.iconT = pevPick(r, squirtVictim), 2.2
		}
		if g.time >= a.until {
			return false
		}
	}
	return true
}

// vaInsultStep: the duel of insults: A insults, B gives the right comeback, then B insults and A answers (two rounds).
func (g *Game) vaInsultStep(a *vaAct) bool {
	if g.time < a.next {
		return true
	}
	if a.step >= 4 {
		// the one who answered last wins the round: he laughs, the other one walks off
		a.a.gu.dAnim, a.b.gu.dAnim = AnimLaugh, AnimPanic
		return false
	}
	p := insultPairs[(a.pair+a.step/2)%len(insultPairs)]
	sp, ot := a.a.gu, a.b.gu
	if (a.step/2)%2 == 1 {
		sp, ot = ot, sp
	}
	line := p.insult
	if a.step%2 == 1 {
		sp, ot = ot, sp
		line = p.reply
	}
	sp.icon, sp.iconT = line, 2.7
	sp.dAnim, ot.dAnim = AnimPoint, AnimTalk
	if a.step%2 == 1 {
		sp.dAnim = AnimLaugh
	}
	a.step++
	a.next = g.time + 2.9
	return true
}

// vaThrowStep: a snowball (arc) or a puck (flat, quicker) goes from one to the other and back.
func (g *Game) vaThrowStep(a *vaAct, r *rand.Rand, dur float64) {
	if a.proj.on && g.time < a.proj.t0+a.proj.dur {
		return
	}
	if a.proj.on { // it arrived: a splash of snow, or the other one stops the puck
		to := a.proj.to
		if a.kind == vaSnowball {
			g.vaSparks(to, r, rgb(0xf4f8ff), 7)
			hit := a.b.gu
			if a.step%2 == 0 {
				hit = a.a.gu
			}
			if r.Intn(3) == 0 && hit.icon == "" {
				hit.icon, hit.iconT = pevPick(r, snowHit), 1.8
			}
		}
		a.proj.on = false
		a.next = g.time + 0.4 + r.Float64()*0.8
		return
	}
	if g.time < a.next {
		return
	}
	a.step++
	from, to := a.a.gu, a.b.gu
	if a.step%2 == 0 {
		from, to = to, from
	}
	from.dAnim, to.dAnim = AnimPunch, AnimIdle
	arc := 0.0
	kind := 1
	h := 6.0
	if a.kind == vaSnowball {
		arc, kind, h = 18, 0, 20
	}
	a.proj = vaProj{on: true, from: Vec{from.pos.X, from.pos.Y - h}, to: Vec{to.pos.X, to.pos.Y - h}, t0: g.time, dur: dur, arc: arc, kind: kind}
}

// vaShotStep: a slapshot at a guest: the puck flies, hits him; he may die of it.
func (g *Game) vaShotStep(run *vaRun, a *vaAct, r *rand.Rand) bool {
	if a.step == 0 {
		a.step = 1
		a.a.gu.dAnim = AnimPunch
		a.a.gu.icon, a.a.gu.iconT = pevPick(r, shotLines), 1.8
		a.proj = vaProj{on: true, from: Vec{a.a.gu.pos.X, a.a.gu.pos.Y - 4}, to: Vec{a.b.gu.pos.X, a.b.gu.pos.Y - 10}, t0: g.time + 0.4, dur: 0.35, kind: 1}
		return true
	}
	if a.step == 1 && g.time >= a.proj.t0+a.proj.dur {
		a.step = 2
		a.proj.on = false
		v := a.b.gu
		g.vaSparks(Vec{v.pos.X, v.pos.Y - 16}, r, rgb(0xb01818), 6)
		if run.puckKills < vaPuckKills && r.Float64() < vaPuckDeath {
			run.puckKills++
			pevForget(&a.b)
			g.killGuest(v, false)
			a.a.gu.icon, a.a.gu.iconT = pevPick(r, shotOops), 2.4
			return false
		}
		a.b.gu.dAnim = AnimPanic
		a.b.gu.icon, a.b.gu.iconT = pevPick(r, shotHurt), 2.2
		a.until = g.time + 1.6
	}
	return g.time < a.until || a.step < 2
}

func (g *Game) vaEnd(a *vaAct, r *rand.Rand) {
	for _, m := range []*pevMem{&a.a, &a.b} {
		if m.gu == nil {
			continue
		}
		if m.mine() {
			pevRelease(m, r)
		} else {
			pevForget(m)
		}
	}
}

func (g *Game) vaSparks(p Vec, r *rand.Rand, col color.RGBA, n int) {
	if len(g.parts) > 500 {
		return
	}
	for i := 0; i < n; i++ {
		an := r.Float64() * 2 * math.Pi
		s := 20 + r.Float64()*40
		l := 0.25 + r.Float64()*0.3
		g.parts = append(g.parts, Particle{pos: p, vel: Vec{math.Cos(an) * s, math.Sin(an)*s - 12}, life: l, max: l, col: col, size: 2, grav: 60})
	}
}

// ---- the clown car ---------------------------------------------------------------------------------------------

// vaCarPlace finds a straight open strip where the clown car drives up and down (not solid: a gag on wheels).
func (g *Game) vaCarPlace(run *vaRun, r *rand.Rand) {
	for i := 0; i < 80; i++ {
		p := g.lvl.RandomRoam(r)
		ok := true
		for dx := -110.0; dx <= 110 && ok; dx += 6 {
			for _, dy := range []float64{0, -10} {
				if !g.lvl.Free(Vec{p.X + dx, p.Y + dy}) {
					ok = false
					break
				}
			}
		}
		if ok && dist(p, g.lvl.Def.Start) > 60 {
			run.car.on, run.car.cx, run.car.cy, run.car.span = true, p.X, p.Y, 90
			return
		}
	}
}

func (run *vaRun) carAt(t float64) (Vec, float64) {
	s := math.Sin(t * 0.35)
	return Vec{run.car.cx + run.car.span*s, run.car.cy}, math.Cos(t * 0.35)
}

// ---- the giant snowball ----------------------------------------------------------------------------------------

func (g *Game) vaBallStep(run *vaRun, dt float64) {
	b := &run.ball
	if run.kind != "snow" || b.at == 0 || b.done {
		return
	}
	if !b.on {
		if g.time < b.at {
			return
		}
		// it comes down through the most crowded column of the slope
		best, bx := -1, 480.0
		for x := 80.0; x <= g.lvl.Def.W-80; x += 40 {
			n := 0
			for _, gu := range g.guests {
				if !gu.out() && math.Abs(gu.pos.X-x) < vaBallRadius+6 {
					n++
				}
			}
			if n > best {
				best, bx = n, x
			}
		}
		b.on, b.x, b.y, b.r = true, bx, -vaBallRadius, 12
		if fastEv {
			println("venue acts: giant snowball at x", int(bx), "t", int(g.time))
		}
		g.say("¡Una bola de nieve gigante baja por la pista!")
		g.snd.Play("alert")
		return
	}
	b.y += vaBallSpeed * dt
	b.r = math.Min(vaBallRadius+8, b.r+dt*3)
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" {
			continue
		}
		if math.Abs(gu.pos.X-b.x) < b.r+3 && math.Abs(gu.pos.Y-b.y) < b.r*0.6+3 {
			g.killGuest(gu, false)
			b.killed++
		}
	}
	if b.y > g.lvl.Def.H+b.r*2 {
		b.on, b.done = false, true
		if b.killed > 0 {
			g.say("La bola de nieve se ha llevado a varios invitados")
		}
	}
}

// ---- drawing ---------------------------------------------------------------------------------------------------

// venueItems: projectiles, snowmen, skis, juggling balls, water, the clown car and the giant snowball (draw.go).
func (g *Game) venueItems(dst *ebiten.Image, items []drawItem) []drawItem {
	run := g.vaActive()
	if run == nil {
		return items
	}
	for i := range run.snowmen {
		sm := run.snowmen[i]
		items = append(items, drawItem{sm.pos.Y, func() { g.vaDrawSnowman(dst, sm) }})
	}
	for gu, kind := range run.skiers {
		if gu.out() || gu.quirk != vaQuirk || !gu.moving {
			continue
		}
		gu, kind := gu, kind
		items = append(items, drawItem{gu.pos.Y + 0.25, func() { g.vaDrawSki(dst, gu, kind) }})
	}
	for _, a := range run.acts {
		a := a
		if a.proj.on {
			if p, _, ok := a.proj.at(g.time); ok {
				items = append(items, drawItem{p.Y + 30, func() { g.vaDrawProj(dst, a) }})
			}
		}
		if a.kind == vaJuggle && a.met {
			items = append(items, drawItem{a.a.gu.pos.Y + 0.3, func() { g.vaDrawJuggle(dst, a) }})
		}
	}
	if run.car.on {
		p, _ := run.carAt(g.time)
		items = append(items, drawItem{p.Y, func() { g.vaDrawCar(dst, run) }})
	}
	if run.ball.on {
		items = append(items, drawItem{run.ball.y + run.ball.r, func() { g.vaDrawBall(dst, &run.ball) }})
	}
	return items
}

func (g *Game) vaDrawProj(dst *ebiten.Image, a *vaAct) {
	p, h, ok := a.proj.at(g.time)
	if !ok {
		return
	}
	x, y := math.Round(g.wx(p.X)), math.Round(g.wy(p.Y-h))
	switch a.proj.kind {
	case 0: // snowball
		fillRect(dst, x-1, y-1, 3, 3, rgb(0xf4f8ff))
		fillRect(dst, x+1, y+1, 1, 1, rgb(0xb8c8e0))
	case 1: // puck
		fillRect(dst, x-2, y, 4, 2, rgb(0x101014))
		fillRect(dst, x-6, y+1, 3, 1, rgba(0xe8f0ff, 120))
	case 2: // the jet of the squirting flower: a string of drops along the way
		k := (g.time - a.proj.t0) / a.proj.dur
		for i := 0; i < 10; i++ {
			f := math.Min(k, 1) * float64(i) / 10
			q := Vec{a.proj.from.X + (a.proj.to.X-a.proj.from.X)*f, a.proj.from.Y + (a.proj.to.Y-a.proj.from.Y)*f - 10*4*f*(1-f)}
			fillRect(dst, math.Round(g.wx(q.X)), math.Round(g.wy(q.Y)), 2, 2, rgb(0x7cc4f4))
		}
	}
}

func (g *Game) vaDrawSnowman(dst *ebiten.Image, sm vaSnowFig) {
	k := g.pevK()
	x, y := math.Round(g.wx(sm.pos.X)), math.Round(g.wy(sm.pos.Y))
	snow, sh := rgb(0xf4f8ff), rgb(0xc8d4e8)
	b := math.Min(1, sm.k*2.2)
	if b <= 0 {
		return
	}
	r1 := float64(k(7)) * b
	filledEllipse(dst, x, y-r1, r1+1, r1, sh)
	filledEllipse(dst, x-1, y-r1-1, r1, r1-1, snow)
	if sm.k > 0.5 {
		r2 := float64(k(5)) * math.Min(1, (sm.k-0.45)*3)
		cy := y - 2*r1 - r2 + 2
		filledEllipse(dst, x, cy, r2+1, r2, sh)
		filledEllipse(dst, x-1, cy-1, r2, r2-1, snow)
		if sm.k > 0.8 {
			r3 := float64(k(4))
			hy := cy - r2 - r3 + 2
			filledEllipse(dst, x, hy, r3, r3, snow)
			fillRect(dst, x-2, hy-1, 1, 1, rgb(0x14141a))
			fillRect(dst, x+1, hy-1, 1, 1, rgb(0x14141a))
			fillRect(dst, x+1, hy+1, 3, 1, rgb(0xf08a20)) // the carrot
			if sm.k >= 1 {
				fillRect(dst, x-3, hy-r3-1, 7, 1, rgb(0x14141a)) // the hat
				fillRect(dst, x-2, hy-r3-4, 5, 3, rgb(0x14141a))
				fillRect(dst, x-r2-3, cy-1, 3, 1, rgb(0x6a4a2a)) // the twig arms
				fillRect(dst, x+r2+1, cy-2, 3, 1, rgb(0x6a4a2a))
			}
		}
	}
}

func (g *Game) vaDrawSki(dst *ebiten.Image, gu *Guest, kind int) {
	k := g.pevK()
	x, y := math.Round(g.wx(gu.pos.X)), math.Round(g.wy(gu.pos.Y))
	if kind == 1 { // the snowboard, across the feet
		w := float64(k(14))
		fillRect(dst, x-w/2, y, w, 2, rgb(0x2a8ad0))
		fillRect(dst, x-w/2, y, w, 1, rgb(0xf0c020))
		return
	}
	l := float64(k(16)) // two skis, pointing where he goes
	switch gu.dir {
	case DirLeft, DirRight:
		off := l * 0.3
		if gu.dir == DirLeft {
			off = -off
		}
		fillRect(dst, x-l/2+off, y-1, l, 1, rgb(0xd83a2a))
		fillRect(dst, x-l/2+off+2, y+1, l, 1, rgb(0xd83a2a))
	default:
		fillRect(dst, x-3, y-l/2+2, 1, l, rgb(0xd83a2a))
		fillRect(dst, x+2, y-l/2+2, 1, l, rgb(0xd83a2a))
	}
}

func (g *Game) vaDrawJuggle(dst *ebiten.Image, a *vaAct) {
	k := g.pevK()
	gu := a.a.gu
	x, top := math.Round(g.wx(gu.pos.X)), math.Round(g.wy(gu.pos.Y))-float64(k(44))
	cols := []uint32{0xe04a5a, 0xf0d030, 0x40a0e0}
	for i := 0; i < 3; i++ {
		ph := math.Mod(g.time*1.6+float64(i)/3, 1)
		bx := x + math.Round(float64(k(7))*math.Cos(ph*2*math.Pi))
		by := top - math.Round(float64(k(10))*math.Sin(ph*math.Pi))
		fillRect(dst, bx-1, by-1, 3, 3, rgb(cols[i]))
	}
}

func (g *Game) vaDrawCar(dst *ebiten.Image, run *vaRun) {
	p, v := run.carAt(g.time)
	k := g.pevK()
	x, y := math.Round(g.wx(p.X)), math.Round(g.wy(p.Y))
	w, h := float64(k(26)), float64(k(10))
	dir := 1.0
	if v < 0 {
		dir = -1
	}
	filledEllipse(dst, x, y, w/2+2, 2, color.RGBA{0, 0, 0, 70})
	fillRect(dst, x-w/2, y-h-3, w, h, rgb(0xe8402a)) // the body
	fillRect(dst, x-w/2, y-h-3, w, 2, rgb(0xff7050))
	fillRect(dst, x+dir*(w/2-3)-1, y-h+1, 3, 3, rgb(0xf0e060)) // the headlamp
	for i, c := range []uint32{0x3aa0e0, 0x50c050, 0xf08a20} { // three clowns squeezed in (afro and red nose)
		cx := x - w/2 + 5 + float64(i)*8
		filledEllipse(dst, cx, y-h-6, 4, 3, rgb(c))
		fillRect(dst, cx-1, y-h-5, 3, 3, rgb(0xf8f4f0))
		fillRect(dst, cx+dir, y-h-4, 1, 1, rgb(0xe02a2a))
	}
	for _, wx := range []float64{x - w/2 + 5, x + w/2 - 5} { // the wheels, turning
		filledEllipse(dst, wx, y-2, 3, 3, rgb(0x14141a))
		a := g.time * 6 * dir
		fillRect(dst, math.Round(wx+math.Cos(a)*2), math.Round(y-2+math.Sin(a)*2), 1, 1, rgb(0xd0d0d8))
	}
	if math.Mod(g.time, 6) < 0.8 { // honk
		g.drawTextC(dst, "¡MEC MEC!", x, y-h-20, rgb(0xfff0a0), 1)
	}
}

func (g *Game) vaDrawBall(dst *ebiten.Image, b *vaBall) {
	x, y := math.Round(g.wx(b.x)), math.Round(g.wy(b.y))
	r := b.r * g.lifeScale()
	filledEllipse(dst, x+2, y+r*0.8, r, r*0.35, color.RGBA{0, 0, 0, 60})
	filledEllipse(dst, x, y, r+2, r+2, rgb(0x4a5670)) // a dark outline: it must stand out on the snow
	filledEllipse(dst, x, y, r+1, r+1, rgb(0x9aaac8))
	filledEllipse(dst, x-1, y-1, r-1, r-1, rgb(0xf4f8ff))
	filledEllipse(dst, x-r*0.35, y-r*0.35, r*0.3, r*0.25, rgb(0xffffff))
	a := b.y / 9 // it rolls: a few lumps go round
	for i := 0; i < 4; i++ {
		an := a + float64(i)*math.Pi/2
		fillRect(dst, math.Round(x+math.Cos(an)*r*0.6), math.Round(y+math.Sin(an)*r*0.6), 3, 2, rgb(0xc8d4e8))
	}
	for i := 0; i < 6; i++ { // the powder it throws up behind
		fillRect(dst, math.Round(x-r+float64(i)*r/3), math.Round(y-r-2-float64((i*7+int(b.y))%5)), 2, 2, rgba(0xf4f8ff, 160))
	}
}
