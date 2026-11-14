package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"math"
	"math/rand"
	"sort"
)

// ---------------------------------------------------------------------------
// Rare set pieces that cross the party: a plane, a UFO that abducts someone,
// Superman snatching a guest, Satan striking people with lightning, God
// adding new guests, and a zombie outbreak. None of these ever raise the
// criminality rating: they are not the player's doing.
// ---------------------------------------------------------------------------

const (
	evPlane = iota
	evUFO
	evSuper
	evSatan
	evGod
	evZombie
	evSanta
	evCow
	evMeteor
	evStars
	evBat
	evRocket
	evFireworks
	evBalloon
	evBirds
	evGhost
	evRex
	evScooters // trophy rewards: the kids' own shows
	evSkate
	evPrank
	evF1 // events5.go / events6.go
	evStampede
	evParachute
	evSport
	evTheme
	evKinds
)

type Event struct {
	kind     int
	x, y     float64
	vx       float64
	t        float64
	phase    int
	n        int
	target   *Guest
	bolts    []Vec // lightning targets, for drawing
	tgt      []float64
	boltT    float64
	crew     []*Guest
	sets     []*CharSet // riders of the kids' events
	done     []bool
	sp       float64
	sp2      float64
	t2       float64
	herd     []herdRunner
	crewFrom *Guest
	boom     *f1Boom // the F1 car blowing up where it used to leave (carboom.go)
}

func (g *Game) liveGuests() []*Guest {
	var out []*Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && gu.state != gLift && gu.state != gZombie {
			out = append(out, gu)
		}
	}
	return out
}

func (g *Game) pickGuest() *Guest {
	l := g.liveGuests()
	if len(l) == 0 {
		return nil
	}
	return l[g.rng.Intn(len(l))]
}

func (g *Game) updateEvents(dt float64) {
	if g.mode != ModePlay || g.testVictim != nil {
		return
	}
	if g.time >= g.eventT || g.forceEv > 0 && g.time > 1 {
		if fastEv {
			g.eventT = g.time + 14 + g.rng.Float64()*6
		} else {
			g.eventT = g.time + rareGap(g.rng, g.time)
		}
		g.startEvent()
	}
	out := g.events[:0]
	for _, e := range g.events {
		if g.stepEvent(e, dt) {
			out = append(out, e)
		}
	}
	g.events = out
}

func (g *Game) startEvent() {
	if g.lvl != rareLvl { // a new party: the count of big set pieces starts again
		rareLvl, rareBigN = g.lvl, 0
	}
	kind := rareKind(g.rng, g.time, rareBigN, g.handSat, g.handGod, func(k int) bool { return ownedID(kidEventID[k]) })
	if fastEv {
		g.evCycle++
		kind = g.evCycle % evKinds
	}
	forced := g.forceEv > 0
	if forced {
		kind, g.forceEv = g.forceEv-1, 0
	}
	if rareBigKinds[kind] && !forced && !fastEv {
		rareBigN++ // counted when tried: a big one that cannot start here does not get a second go in this party
	}
	live := g.liveGuests()
	e := &Event{kind: kind}
	switch kind {
	case evScooters, evSkate:
		if !g.startKidRun(e) {
			return
		}
	case evPrank:
		if !g.startPrank(e) {
			return
		}
	case evF1:
		if !g.startF1(e) {
			if forced {
				println("event evF1 could not start here")
			}
			return
		}
	case evStampede:
		if !g.startStampede(e) {
			if forced {
				println("event evStampede could not start here")
			}
			return
		}
	case evParachute:
		if !g.startParachute(e) {
			if forced {
				println("event evParachute could not start here")
			}
			return
		}
	case evSport:
		if !g.startSport(e) {
			if forced {
				println("event evSport could not start here")
			}
			return
		}
	case evTheme:
		if !g.startTheme(e) {
			if forced {
				println("event evTheme could not start here")
			}
			return
		}
	case evGhost:
		if !g.startGhost(e) {
			return
		}
	case evRex:
		if !g.startRex(e) {
			return
		}
	case evFireworks:
		g.say("¡Fuegos artificiales!")
		g.snd.Play("boom")
	case evBalloon:
		e.y = 55 + g.rng.Float64()*60
		e.x, e.vx = -80, 24
		if g.rng.Intn(2) == 0 {
			e.x, e.vx = 1040, -24
		}
		g.say("Un globo aerostático sobrevuela la fiesta")
	case evBirds:
		e.y = 50 + g.rng.Float64()*110
		e.x, e.vx = -60, 120
		if g.rng.Intn(2) == 0 {
			e.x, e.vx = 1020, -120
		}
		for i := 0; i < 9; i++ {
			e.bolts = append(e.bolts, Vec{float64(i%3) * -16 * math.Copysign(1, e.vx), float64(i/3)*9 - 9 + g.rng.Float64()*6})
		}
	case evStars:
		g.say("¡Estrellas fugaces! Pide un deseo...")
	case evBat:
		e.y = 40 + g.rng.Float64()*50
		e.x, e.vx = -100, 130
		if g.rng.Intn(2) == 0 {
			e.x, e.vx = 1060, -130
		}
		g.say("¿Eso es un... murciélago gigante?")
	case evRocket:
		e.x, e.y, e.vx = 140+g.rng.Float64()*680, 620, (g.rng.Float64()-0.5)*30
		g.say("Un cohete rumbo a la Luna")
	case evSanta, evCow:
		e.y = 70 + g.rng.Float64()*50
		e.x, e.vx = -120, 80
		if g.rng.Intn(2) == 0 {
			e.x, e.vx = 1080, -80
		}
		if kind == evSanta {
			g.say("¡Jo, jo, jo! Papá Noel pasa de largo...")
		} else {
			g.say("¿Eso es una vaca volando?")
		}
	case evMeteor:
		if len(live) < 3 {
			return
		}
		v := g.pickGuest()
		e.target = v
		e.x, e.y = v.pos.X-260, -40
		e.n = 0
		g.say("¡Un meteorito se acerca!")
		g.snd.Play("alert")
	case evPlane:
		e.y = 70 + g.rng.Float64()*50
		e.x, e.vx = -90, 95
		if g.rng.Intn(2) == 0 {
			e.x, e.vx = 1050, -95
		}
	case evUFO, evSuper:
		if len(live) < 4 {
			return
		}
		e.target = g.pickGuest()
		e.x, e.vx = -70, 1
		if g.rng.Intn(2) == 0 {
			e.x, e.vx = 1030, -1
		}
		e.y = 30
		if kind == evSuper {
			g.say("¡Es un pájaro! ¡Es un avión!")
		} else {
			g.say("Un ovni sobrevuela la fiesta...")
		}
	case evSatan:
		if len(live) < 4 {
			return
		}
		e.x, e.y = 480+(g.rng.Float64()-0.5)*240, -110
		e.n = 2 + g.rng.Intn(3)
		g.handSat = true
		g.say("El cielo se vuelve rojo...")
		g.snd.Play("boom")
	case evGod:
		if g.guestRoom() < 1 { // the kill target is full (cap.go): there is nobody left to create
			return
		}
		e.x, e.y = 480+(g.rng.Float64()-0.5)*240, -110
		e.n = minI(5+g.rng.Intn(6), g.guestRoom())
		g.handGod = true
		g.say("Una luz baja del cielo...")
	case evZombie:
		if len(live) < 6 {
			return
		}
		z := g.pickGuest()
		g.makeZombie(z)
		g.say("¡Un zombi anda suelto por la fiesta!")
		g.snd.Play("alert")
		return
	}
	g.stat("events", 1)
	g.stat(fmt.Sprintf("ev_%d", kind), 1)
	g.events = append(g.events, e)
}

// ---- how often (feedback: "los eventos raros casi ni los veo") ------------------------------------------------
// One rare event every 65-120 s at the start of the party, more often as it goes on (down to 70 % of that at 270 s);
// the first one at 45-90 s (loadLevel). The BIG ones (UFO, Superman, Satan, God, zombies, meteor, ghost, T-Rex) never in
// the first minute and at most rareBigMax per party: they grow with the length of the party without flooding it. A
// 5-8 minute party sees 3-6 rare events (4.5 on average, before 3.4-4.8) (pe_freq_test.go measures it). Nothing has to be unlocked except the kids'
// shows (scooters, skaters, prank), which are trophy rewards and stay so.

const (
	rareBigMax   = 2
	rareBigAfter = 60.0  // seconds of party before the first big one
	zombieExtra  = 0.007 // share of the other small kinds that turn into zombies (feedback: "no recuerdo la última vez que vi zombis")
)

var rareBigKinds = map[int]bool{evUFO: true, evSuper: true, evSatan: true, evGod: true, evZombie: true, evMeteor: true, evGhost: true, evRex: true}

var (
	rareLvl  *Level // the party rareBigN counts for
	rareBigN int
)

// rareGap: seconds to the next rare event at party time t.
func rareGap(r *rand.Rand, t float64) float64 {
	return (65 + r.Float64()*55) * math.Max(0.7, 1-t/900)
}

var rareSmall = []int{evPlane, evStars, evBat, evRocket, evSanta, evCow}

// rareKind: the kind of the next rare event (not forced). bigN: big ones already this party; sat/god: hands seen.
func rareKind(r *rand.Rand, t float64, bigN int, sat, god bool, owned func(int) bool) int {
	kind := r.Intn(evKinds)
	if kind >= evScooters && kind <= evPrank && !owned(kind) {
		kind = rareSmall[r.Intn(len(rareSmall))] // the kids' shows come only once earned
	}
	// each hand (Satan, God) shows up at most once per party
	if kind == evSatan && sat || kind == evGod && god {
		kind = rareSmall[r.Intn(len(rareSmall))]
	}
	if kind != evZombie && !rareBigKinds[kind] && r.Float64() < zombieExtra {
		kind = evZombie // a nudge so that zombies show up in about one party of six (TestZombieOnePartyInSix)
	}
	if kind == evRex && r.Float64() < 0.6 {
		kind = evPlane // the T-Rex is the rarest visitor
	}
	if rareBigKinds[kind] && (t < rareBigAfter || bigN >= rareBigMax) {
		kind = rareSmall[r.Intn(len(rareSmall))]
	}
	return kind
}

// rareSchedule: the rare events of a party of L seconds by the same rules (true = a big one). For the tests.
func rareSchedule(r *rand.Rand, L float64) []bool {
	var out []bool
	big, sat, god := 0, false, false
	for t := 45 + r.Float64()*45; t < L; t += rareGap(r, t) {
		k := rareKind(r, t, big, sat, god, func(int) bool { return false })
		sat, god = sat || k == evSatan, god || k == evGod
		if rareBigKinds[k] {
			big++
		}
		out = append(out, rareBigKinds[k])
	}
	return out
}

func (g *Game) makeZombie(gu *Guest) {
	if gu.state == gZombie || gu.out() {
		return
	}
	if gu.partner != nil && gu.partner.partner == gu {
		gu.partner.partner = nil
		if gu.partner.state == gChat || gu.partner.state == gKiss || gu.partner.state == gWaitP {
			gu.partner.state, gu.partner.timer = gIdle, 1
		}
	}
	gu.partner = nil
	gu.state, gu.path, gu.timer = gZombie, nil, 0
	gu.icon, gu.iconT = "UUUGH", 2
}

// stepEvent advances one event; false means it is over.
func (g *Game) stepEvent(e *Event, dt float64) bool {
	e.t += dt
	switch e.kind {
	case evScooters, evSkate:
		return g.stepKidRun(e, dt)
	case evPrank:
		return g.stepPrank(e, dt)
	case evF1:
		return g.stepF1(e, dt)
	case evStampede:
		return g.stepStampede(e, dt)
	case evParachute:
		return g.stepParachute(e, dt)
	case evSport:
		return g.stepSport(e, dt)
	case evTheme:
		return g.stepTheme(e, dt)
	case evGhost:
		return g.stepGhost(e, dt)
	case evRex:
		return g.stepRex(e, dt)
	case evPlane:
		e.x += e.vx * dt
		return e.x > -120 && e.x < 1080
	case evSanta, evCow:
		e.x += e.vx * dt
		return e.x > -160 && e.x < 1120
	case evBalloon:
		e.x += e.vx * dt
		e.y += math.Sin(e.t*0.8) * 6 * dt
		return e.x > -140 && e.x < 1100
	case evBirds:
		e.x += e.vx * dt
		return e.x > -120 && e.x < 1080
	case evFireworks:
		// rockets climb, then burst into coloured sparks
		if e.t < 8 && e.boltT <= 0 {
			e.boltT = 0.5 + g.rng.Float64()*0.6
			e.bolts = append(e.bolts, Vec{120 + g.rng.Float64()*720, 420 + g.rng.Float64()*40})
			e.tgt = append(e.tgt, 40+g.rng.Float64()*140)
		}
		e.boltT -= dt
		var nb []Vec
		var nt []float64
		for i, p := range e.bolts {
			p.Y -= 230 * dt
			if p.Y <= e.tgt[i] {
				cols := []uint32{0xff4040, 0xffd040, 0x40ff80, 0x40c0ff, 0xff60e0, 0xffffff}
				g.burst(p, 40, 110, rgb(cols[g.rng.Intn(len(cols))]), 1.4, 2, 70)
				g.snd.Play("spark")
				continue
			}
			nb = append(nb, p)
			nt = append(nt, e.tgt[i])
		}
		e.bolts, e.tgt = nb, nt
		return e.t < 8 || len(e.bolts) > 0
	case evBat:
		e.x += e.vx * dt
		e.y += math.Sin(e.t*2) * 12 * dt
		return e.x > -160 && e.x < 1120
	case evRocket:
		e.y -= (60 + e.t*70) * dt
		e.x += e.vx * dt
		// smoke that lingers behind
		e.bolts = append(e.bolts, Vec{e.x + (g.rng.Float64()-0.5)*6, e.y + 26})
		for i := range e.bolts {
			e.bolts[i].Y += 10 * dt
		}
		if len(e.bolts) > 90 {
			e.bolts = e.bolts[1:]
		}
		return e.y > -140
	case evStars:
		// a few streaks cross the sky for ~6 seconds
		if e.t < 5 && e.boltT <= 0 {
			e.boltT = 0.5 + g.rng.Float64()*0.7
			e.bolts = append(e.bolts, Vec{100 + g.rng.Float64()*900, -10 + g.rng.Float64()*120})
		}
		e.boltT -= dt
		out := e.bolts[:0]
		for _, p := range e.bolts {
			p.X -= 330 * dt
			p.Y += 150 * dt
			if p.X > -40 && p.Y < 400 {
				out = append(out, p)
			}
		}
		e.bolts = out
		return e.t < 5 || len(e.bolts) > 0
	case evMeteor:
		// falls diagonally onto where the chosen guest was standing
		tx, ty := e.target.pos.X, e.target.pos.Y
		if e.n == 0 {
			e.n = 1
			e.vx = tx
			e.boltT = ty
		}
		tx, ty = e.vx, e.boltT
		dx, dy := tx-e.x, ty-e.y
		d := math.Hypot(dx, dy)
		if d < 14 {
			g.explodeAt(tx, ty)
			for _, o := range g.guests {
				if !o.out() && dist(o.pos, Vec{tx, ty}) < 60 {
					g.killGuest(o, false)
				}
			}
			g.sayNews("kill", map[string]string{"actor": "Un meteorito", "victim": guestLabel(e.target)})
			g.checkDone()
			return false
		}
		e.x += dx / d * 300 * dt
		e.y += dy / d * 300 * dt
	case evUFO:
		return g.stepUFO(e, dt)
	case evSuper:
		return g.stepSuper(e, dt)
	case evSatan:
		switch e.phase {
		case 0:
			e.y += 120 * dt
			if e.y >= 190 {
				e.phase, e.t, e.boltT = 1, 0, 0.6
			}
		case 1:
			e.boltT -= dt
			if e.boltT <= 0 && e.n > 0 {
				e.boltT = 0.9
				e.n--
				if v := g.nearestGuest(Vec{e.x, e.y + 70}); v != nil {
					e.bolts = append(e.bolts, v.pos)
					g.snd.Play("zap")
					g.flash = 0.12
					g.burst(Vec{v.pos.X, v.pos.Y - 14}, 14, 90, rgb(0xfff0a0), 0.4, 2, 0)
					// a medium-range strike: everyone near the bolt dies with it
					vp := v.pos
					for _, o := range g.guests {
						if !o.out() && o != v && dist(o.pos, vp) < 70 {
							g.burst(Vec{o.pos.X, o.pos.Y - 14}, 8, 70, rgb(0xfff0a0), 0.35, 2, 0)
							g.killGuestElec(o, false)
						}
					}
					g.killGuestElec(v, false)
					g.sayNews("kill", map[string]string{"actor": "Satanás", "victim": guestLabel(v)})
					g.checkDone()
				}
			}
			if e.n == 0 && e.boltT <= 0 {
				e.phase = 2
			}
			if len(e.bolts) > 0 && e.t > 0.25 {
				e.t = 0
				e.bolts = e.bolts[:len(e.bolts)-1]
			}
		default:
			e.bolts = nil
			e.y -= 130 * dt
			return e.y > -130
		}
	case evGod:
		switch e.phase {
		case 0:
			e.y += 110 * dt
			if e.y >= 190 {
				e.phase, e.boltT = 1, 0
				e.t = float64(e.n)
			}
		case 1:
			e.boltT -= dt
			if e.boltT <= 0 && e.n > 0 {
				e.boltT = 0.12
				e.n--
				p := Vec{e.x + (g.rng.Float64()-0.5)*190, e.y + 70 + (g.rng.Float64()-0.5)*120}
				p = g.lvl.NearestFree(p)
				if g.guestRoom() < 1 { // the party (or its kill target, cap.go) is full: nobody is created
					e.t--
				} else {
					g.addGuestAt(p)
					g.burst(Vec{p.X, p.Y - 20}, 10, 60, rgb(0xfff0a0), 0.5, 2, -10)
				}
			}
			if e.n == 0 && e.t < 1 {
				e.phase = 2 // nobody fitted: the hands go away without a word
			} else if e.n == 0 {
				g.say("Dios ha creado " + itoa(int(e.t)) + " invitados nuevos")
				e.phase = 2
			}
		default:
			e.y -= 120 * dt
			return e.y > -130
		}
	}
	return true
}

func (g *Game) stepUFO(e *Event, dt float64) bool {
	t := e.target
	switch e.phase {
	case 0:
		if t == nil || t.out() {
			e.phase = 2
			break
		}
		dx := t.pos.X - e.x
		if math.Abs(dx) < 4 {
			e.phase, e.t = 1, 0
			t.state, t.path, t.icon, t.iconT = gLift, nil, "¡AAAH!", 3
			g.snd.Play("alert")
			break
		}
		e.x += math.Copysign(math.Min(math.Abs(dx), 130*dt), dx)
	case 1:
		if t == nil || t.out() {
			e.phase = 2
			break
		}
		t.lift = math.Min(70, e.t*24)
		if e.t >= 3 {
			t.gone = true
			g.burst(Vec{t.pos.X, t.pos.Y - 70}, 16, 60, rgb(0xa0ffb0), 0.5, 2, 0)
			g.say(guestLabel(t) + " ha sido abducido por un ovni")
			g.checkDone()
			e.phase, e.t = 2, 0
		}
	default:
		e.y -= 40 * dt
		e.x += 150 * dt * math.Copysign(1, e.x-480)
		return e.t < 4 && e.x > -100 && e.x < 1060
	}
	return true
}

func (g *Game) stepSuper(e *Event, dt float64) bool {
	t := e.target
	switch e.phase {
	case 0:
		if t == nil || t.out() {
			e.phase = 2
			e.vx = math.Copysign(1, e.vx)
			break
		}
		gx, gy := t.pos.X, t.pos.Y-26
		dx, dy := gx-e.x, gy-e.y
		d := math.Hypot(dx, dy)
		if d < 10 {
			t.gone = true
			g.say(guestLabel(t) + " se ha ido volando con Souperman")
			g.burst(Vec{t.pos.X, t.pos.Y - 16}, 10, 60, rgb(0xffffff), 0.4, 2, 0)
			g.checkDone()
			e.phase = 1
			e.vx = math.Copysign(1, e.x-480+0.1)
			break
		}
		e.vx = math.Copysign(1, dx+0.001)
		sp := 170.0
		e.x += dx / d * sp * dt
		e.y += dy / d * sp * dt
	default:
		e.x += e.vx * 190 * dt
		e.y -= 70 * dt
		return e.x > -90 && e.x < 1050 && e.y > -60
	}
	return true
}

// ---------------------------------------------------------------------------
// Zombies: very slow, they bite other guests into zombies (nobody dies) and
// kill the player on contact. One stab puts a zombie down.
// ---------------------------------------------------------------------------

func (g *Game) updateZombie(gu *Guest, dt float64) {
	gu.timer -= dt
	defer g.focus(gu.pos)() // two players: the zombie hunts and bites the nearest one, not always the first
	p := g.p
	speed := 8.0
	var tp Vec
	have := false
	if !p.hidden && !p.dead && dist(gu.pos, p.pos) < 150 && g.lvl.LOS(gu.pos, p.pos) {
		tp, have = p.pos, true
		speed = 22
	} else {
		best := 200.0
		for _, o := range g.guests {
			if o.out() || o.state == gZombie || o.state == gLift {
				continue
			}
			if d := dist(gu.pos, o.pos); d < best {
				best, tp, have = d, o.pos, true
			}
		}
	}
	if !have {
		if len(gu.path) == 0 {
			gu.path = g.lvl.PathG(gu.pos, g.lvl.RandomRoam(g.rng))
		}
	} else {
		gu.repath -= dt
		if gu.repath <= 0 || len(gu.path) == 0 {
			gu.repath = 0.6
			gu.path = g.lvl.PathG(gu.pos, tp)
		}
	}
	gu.walk(speed, dt)
	if gu.timer <= 0 {
		gu.timer = 0.5
		for _, o := range g.guests {
			if o.out() || o.state == gZombie || o.state == gLift {
				continue
			}
			d := dist(gu.pos, o.pos)
			if d < 12 {
				if !g.gyBite(gu, o) { // the graveyard's zombies eat brains instead (graveyard.go)
					g.makeZombie(o)
				}
				g.snd.Play("thud")
			} else if d < 70 && o.state != gScared && o.perf == "" {
				o.state, o.timer, o.path = gScared, 5+g.rng.Float64()*3, nil
				o.icon, o.iconT = "¡ZOMBI!", 2
			}
		}
		if !p.hidden && !p.dead && dist(gu.pos, p.pos) < 12 {
			g.die("Un zombi te ha mordido")
		}
	}
}

// ---------------------------------------------------------------------------
// Drawing
// ---------------------------------------------------------------------------

func (g *Game) drawEvents(dst *ebiten.Image) {
	out := g.trails[:0]
	for _, tr := range g.trails {
		tr.t += 1.0 / 60
		if tr.t < 5 {
			a := uint8(200 * (1 - tr.t/5))
			fillRect(dst, g.wx(tr.x), g.wy(tr.y), 3, 2, color.RGBA{a, a / 2, 0, a})
			fillRect(dst, g.wx(tr.x), g.wy(tr.y)-1, 2, 1, color.RGBA{a, a * 4 / 5, 0, a})
			out = append(out, tr)
		}
	}
	g.trails = out
	for _, e := range g.events {
		x, y := g.wx(e.x), g.wy(e.y)
		switch e.kind {
		case evGhost:
			g.drawGhostBeams(dst, e)
		case evRex:
			filledEllipse(dst, g.wx(e.x), g.wy(e.y), 46, 9, color.RGBA{0, 0, 0, 70})
		case evMeteor:
			filledEllipse(dst, x, y, 7, 7, rgb(0xffb030))
			filledEllipse(dst, x, y, 4, 4, rgb(0xfff4c0))
			for i := 1.0; i < 9; i++ {
				fillRect(dst, x-i*5, y-i*5, 7-i*0.6, 7-i*0.6, rgba(0xff7020, uint8(180-i*18)))
			}
		case evPlane:
			// its shadow glides over the party; the plane itself is drawn above the lighting
			filledEllipse(dst, x, y+230, 26, 6, color.RGBA{0, 0, 0, 40})
		case evUFO:
			if e.phase == 1 && e.target != nil {
				ty := g.wy(e.target.pos.Y)
				for i := 0.0; i < 10; i++ {
					w := 6 + i*2.4
					fillRect(dst, x-w/2, y+10+(ty-y-10)*i/10, w, (ty-y)/10+1, rgba(0xa0ffb0, uint8(26+i*3)))
				}
			}
			im := g.as.ufo
			w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(math.Round(x-w), math.Round(y-h+math.Sin(e.t*3)*2))
			dst.DrawImage(im, op)
		case evSuper:
			g.drawSuper(dst, x, y, e.vx < 0, e.phase == 1)
		case evSatan:
			g.drawHand(dst, x, y, rgb(0xc02020), rgb(0x701010), false)
			for _, b := range e.bolts {
				g.drawBolt(dst, x, y+64, g.wx(b.X), g.wy(b.Y)-10)
			}
		case evGod:
			g.drawHand(dst, x, y, rgb(0xf4e6c0), rgb(0xd8c080), true)
		}
	}
}

func (g *Game) drawPlane(dst *ebiten.Image, x, y float64, left bool, t float64) {
	s := 1.0
	if left {
		s = -1
	}
	b := func(dx, dy, w, h float64, c color.RGBA) {
		dx, dy, w, h = dx*1.8, dy*1.8, w*1.8, h*1.8
		if s < 0 {
			dx = -dx - w
		}
		fillRect(dst, x+dx, y+dy, w, h, c)
	}
	white, grey := rgb(0xdfe3ea), rgb(0x8c93a2)
	b(-24, -3, 48, 6, white)
	b(14, -3, 12, 4, grey)
	b(-6, -14, 8, 11, grey)
	b(-6, 3, 8, 11, grey)
	b(-26, -8, 6, 6, white)
	b(-22, -4, 3, 2, rgb(0x4aa0e0))
	if int(t*2)%2 == 0 {
		b(-7, -15, 2, 2, rgb(0xff3030))
		b(-7, 13, 2, 2, rgb(0x30ff60))
	}
}

func (g *Game) drawSuper(dst *ebiten.Image, x, y float64, left, carrying bool) {
	s := 1.0
	if left {
		s = -1
	}
	b := func(dx, dy, w, h float64, c color.RGBA) {
		if s < 0 {
			dx = -dx - w
		}
		fillRect(dst, x+dx, y+dy, w, h, c)
	}
	b(-30, -4, 22, 8, rgb(0xc02828)) // cape
	b(-8, -3, 18, 7, rgb(0x2850c0))  // body
	b(10, -3, 8, 4, rgb(0x2850c0))   // legs forward
	b(16, 1, 4, 3, rgb(0xc02828))    // boots
	b(-10, -2, 4, 5, rgb(0xc02828))  // belt
	b(8, -8, 7, 7, rgb(0xf0c8a0))    // head
	b(8, -9, 7, 3, rgb(0x181820))    // hair
	b(0, -2, 3, 3, rgb(0xf0d030))    // emblem
	if carrying {
		b(-2, 4, 6, 8, rgb(0xd0d0d8))
	}
}

func (g *Game) drawHand(dst *ebiten.Image, x, y float64, col, dark color.RGBA, halo bool) {
	if halo {
		g.glowAt(dst, Vec{x - g.camI.X, y - g.camI.Y + 40}, 90, color.RGBA{200, 180, 90, 90})
	}
	fillRect(dst, x-13, y-260, 26, 260, col)
	fillRect(dst, x-13, y-260, 4, 260, dark)
	fillRect(dst, x-26, y, 52, 34, col)
	fillRect(dst, x-26, y+30, 52, 4, dark)
	for i := 0.0; i < 4; i++ {
		fillRect(dst, x-26+i*13.5, y+34, 11, 30-math.Abs(i-1.5)*5, col)
		fillRect(dst, x-26+i*13.5+9, y+34, 2, 28-math.Abs(i-1.5)*5, dark)
	}
	fillRect(dst, x-38, y+6, 14, 12, col)
}

func (g *Game) drawBolt(dst *ebiten.Image, x0, y0, x1, y1 float64) {
	n := 9
	px, py := x0, y0
	for i := 1; i <= n; i++ {
		t := float64(i) / float64(n)
		nx := x0 + (x1-x0)*t + math.Sin(g.time*40+float64(i)*7)*9*(1-t)
		ny := y0 + (y1-y0)*t
		fillRect(dst, math.Min(px, nx)-1, py, math.Abs(nx-px)+3, ny-py+1, rgb(0xfff8c0))
		fillRect(dst, math.Min(px, nx)-3, py, math.Abs(nx-px)+7, ny-py+1, rgba(0x80b0ff, 60))
		px, py = nx, ny
	}
	g.glowAt(dst, Vec{x1 - g.camI.X, y1 - g.camI.Y}, 40, color.RGBA{120, 150, 230, 140})
}

// nearestGuest picks a random one of the few guests closest to a point, so the
// hands of Satan and God act around the middle of the screen where they reach.
func (g *Game) nearestGuest(p Vec) *Guest {
	l := g.liveGuests()
	sort.Slice(l, func(i, j int) bool { return dist(l[i].pos, p) < dist(l[j].pos, p) })
	if len(l) == 0 {
		return nil
	}
	return l[g.rng.Intn(minI(len(l), 4))]
}

// drawEventsTop draws what flies above everything, after the lighting pass.
func (g *Game) drawEventsTop(dst *ebiten.Image) {
	defer g.drawBubbles(dst)
	if dbgSolids && g.lvl != nil {
		for _, r := range g.lvl.solidsP {
			fillRect(dst, g.wx(r.X), g.wy(r.Y), r.W, r.H, color.RGBA{255, 0, 0, 90})
		}
	}
	defer g.drawFloats(dst)
	defer g.drawUseMarks(dst)
	g.drawTraps2(dst)
	g.drawBeams(dst)
	for _, e := range g.events {
		x, y := g.wx(e.x), g.wy(e.y)
		switch e.kind {
		case evScooters, evSkate:
			g.drawKidRun(dst, e)
		case evPrank:
			g.drawPrank(dst, e)
		case evF1:
			g.drawF1(dst, e)
		case evStampede:
			g.drawStampede(dst, e)
		case evParachute:
			g.drawParachute(dst, e)
		case evSport:
			g.drawSport(dst, e)
		case evGhost:
			g.drawGhost(dst, e)
		case evRex:
			g.drawRex(dst, e)
		case evPlane:
			g.drawPlane(dst, x, y, e.vx < 0, e.t)
		case evSanta:
			g.drawSanta(dst, x, y, e.vx < 0, e.t)
		case evCow:
			g.drawCow(dst, x, y, e.vx < 0, e.t)
		case evBat:
			g.drawBatwing(dst, x, y, e.vx < 0, e.t)
		case evBalloon:
			g.drawBalloon(dst, x, y, e.t)
		case evBirds:
			for i, b := range e.bolts {
				bx, by := x+b.X, y+b.Y+math.Sin(e.t*9+float64(i))*2
				flap := math.Sin(e.t*14 + float64(i)*1.3)
				col := color.RGBA{20, 20, 28, 230}
				fillRect(dst, bx-4, by-flap*3, 4, 2, col)
				fillRect(dst, bx, by-flap*3, 4, 2, col)
				fillRect(dst, bx-1, by, 3, 2, col)
			}
		case evFireworks:
			for _, p := range e.bolts {
				px, py := g.wx(p.X), g.wy(p.Y)
				for i := 0.0; i < 8; i++ {
					a := uint8(220 - i*28)
					fillRect(dst, px, py+i*3, 2, 3, color.RGBA{a, a * 3 / 4, a / 3, a})
				}
				g.glowAt(dst, Vec{px - g.camI.X, py - g.camI.Y}, 14, color.RGBA{255, 200, 120, 120})
			}
		case evRocket:
			g.drawRocket(dst, e, x, y)
		case evStars:
			for _, p := range e.bolts {
				px, py := g.wx(p.X), g.wy(p.Y)
				for i := 0.0; i < 14; i++ {
					a := uint8(230 - i*16)
					fillRect(dst, px+i*5, py-i*2.3, 3, 2, color.RGBA{a, a, uint8(float64(a) * 0.8), a})
				}
				g.glowAt(dst, Vec{px - g.camI.X, py - g.camI.Y}, 16, color.RGBA{255, 250, 210, 140})
			}
		}
	}
}

func (g *Game) drawSanta(dst *ebiten.Image, x, y float64, left bool, t float64) {
	s := 1.0
	if left {
		s = -1
	}
	b := func(dx, dy, w, h float64, c color.RGBA) {
		if s < 0 {
			dx = -dx - w
		}
		fillRect(dst, x+dx, y+dy, w, h, c)
	}
	for i := 0.0; i < 3; i++ { // reindeer
		ox := 30 + i*22
		bob := math.Sin(t*8+i) * 2
		b(ox, bob-2, 14, 7, rgb(0x8a5a30))
		b(ox+12, bob-6, 5, 5, rgb(0x8a5a30))
		b(ox+15, bob-9, 1, 4, rgb(0x5a3a20))
		if i == 2 {
			b(ox+16, bob-4, 2, 2, rgb(0xe03030))
		}
		b(ox+2, bob+5, 2, 5, rgb(0x5a3a20))
		b(ox+10, bob+5, 2, 5, rgb(0x5a3a20))
		b(ox-8, bob, 8, 1, rgb(0xc8c0a0))
	}
	b(-18, -4, 40, 9, rgb(0xc02020)) // sleigh
	b(-18, -4, 40, 2, rgb(0xe8c040))
	b(-22, 5, 48, 2, rgb(0xe8c040))
	b(-4, -14, 9, 10, rgb(0xd02828)) // Santa
	b(-3, -9, 7, 3, rgb(0xf4f4f4))
	b(-18, -12, 10, 8, rgb(0x2a8a40)) // sack
}

func (g *Game) drawCow(dst *ebiten.Image, x, y float64, left bool, t float64) {
	s := 1.0
	if left {
		s = -1
	}
	b := func(dx, dy, w, h float64, c color.RGBA) {
		if s < 0 {
			dx = -dx - w
		}
		fillRect(dst, x+dx, y+dy+math.Sin(t*3)*3, w, h, c)
	}
	b(-14, -6, 28, 13, rgb(0xf4f0e8))
	b(-8, -6, 8, 6, rgb(0x2a2a30))
	b(4, 0, 7, 6, rgb(0x2a2a30))
	b(12, -9, 9, 9, rgb(0xf4f0e8))
	b(18, -3, 4, 4, rgb(0xe8a0a0))
	b(13, -11, 2, 3, rgb(0x2a2a30))
	for i := 0.0; i < 4; i++ {
		b(-12+i*7, 7, 3, 6+math.Sin(t*10+i)*2, rgb(0xd8d0c8))
	}
	b(-18, -4, 5, 2, rgb(0xd8d0c8)) // tail
}

func (g *Game) drawBatwing(dst *ebiten.Image, x, y float64, left bool, t float64) {
	s := 1.0
	if left {
		s = -1
	}
	b := func(dx, dy, w, h float64, c color.RGBA) {
		dx, dy, w, h = dx*1.8, dy*1.8, w*1.8, h*1.8
		if s < 0 {
			dx = -dx - w
		}
		fillRect(dst, x+dx, y+dy, w, h, c)
	}
	blk, dk := rgb(0x3c3c52), rgb(0x78789a)
	b(-20, -2, 40, 5, blk) // fuselage
	b(14, -1, 12, 3, dk)   // nose
	b(-8, -10, 14, 8, blk) // upper wing
	b(-8, 3, 14, 8, blk)   // lower wing
	b(-22, -7, 6, 5, blk)  // tail fin
	b(-22, 4, 6, 5, blk)
	b(-3, -3, 10, 3, rgb(0x7ab0d8)) // canopy
	b(-8, -10, 2, 3, dk)
	b(-8, 8, 2, 3, dk)
	b(-27, 0, 5, 2, rgba(0xffb030, uint8(150+int(math.Sin(t*30)*70)))) // thruster
	// bat-signal glow under it
	g.glowAt(dst, Vec{x - g.camI.X, y - g.camI.Y + 20}, 36, color.RGBA{240, 220, 90, 40})
}

// rocketImg: the cached rocket sprite (vessels.go genRocket).
var rocketImg *ebiten.Image

func (g *Game) drawRocket(dst *ebiten.Image, e *Event, x, y float64) {
	for i, p := range e.bolts {
		a := uint8(8 + i*i/180)
		r := 4 + float64(len(e.bolts)-i)*0.1
		filledEllipse(dst, g.wx(p.X), g.wy(p.Y), r, r, rgba(0xb8bcc8, a))
	}
	x, y = math.Round(x), math.Round(y)
	ny := y - 44 + rocketH - 3 // the mouth of the nozzles
	// the flame of each engine: red, orange, yellow and a white-hot core, flickering, and its sparks
	for k, ex := range []float64{x - 5, x + 4} {
		fl := math.Round(18 + math.Sin(e.t*40+float64(k)*2)*5)
		for _, ly := range []struct {
			w, l float64
			col  uint32
		}{{4, fl, 0xe0381a}, {3, fl * 0.8, 0xff8a20}, {2, fl * 0.55, 0xffd040}, {1, fl * 0.3, 0xfff4c0}} {
			for r := 0.0; r < ly.l; r++ {
				hw := math.Max(1, math.Round(ly.w*(1-r/ly.l*0.6)))
				fillRect(dst, ex-hw+1, ny+r, 2*hw, 1, rgb(ly.col))
			}
		}
		for i := 0; i < 5; i++ {
			fi := float64(i)
			py := ny + fl + math.Mod(e.t*90+fi*11, 26)
			px := ex + math.Round(math.Sin(fi*2.3+e.t*5)*3)
			col := rgb(0xffa020)
			if i%2 == 1 {
				col = rgb(0xe0381a)
			}
			fillRect(dst, px, math.Round(py), 2, 2, col)
		}
	}
	if rocketImg == nil {
		rocketImg = genRocket().img()
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-rocketW/2, y-44)
	dst.DrawImage(rocketImg, op)
	g.glowAt(dst, Vec{x - g.camI.X, y - g.camI.Y + 24}, 50, color.RGBA{255, 160, 50, 110})
}

func (g *Game) drawBalloon(dst *ebiten.Image, x, y, t float64) {
	cols := []color.RGBA{rgb(0xe03030), rgb(0xf0d030), rgb(0x3060e0), rgb(0xf4f4f4), rgb(0x30b050)}
	for i, cc := range cols {
		w := 10.0 - math.Abs(float64(i)-2)*2.4
		filledEllipse(dst, x+float64(i-2)*5.5, y-6+math.Abs(float64(i-2))*2, w/1.5+5, 17-math.Abs(float64(i-2))*3.4, cc)
	}
	fillRect(dst, x-9, y+12, 1, 10, rgb(0x6a4a2a))
	fillRect(dst, x+8, y+12, 1, 10, rgb(0x6a4a2a))
	fillRect(dst, x-8, y+22, 17, 7, rgb(0x8a5a2e))
	fillRect(dst, x-8, y+22, 17, 1, rgb(0xb88a5c))
	g.glowAt(dst, Vec{x - g.camI.X, y - g.camI.Y + 22}, 16, color.RGBA{255, 180, 80, 70})
}
