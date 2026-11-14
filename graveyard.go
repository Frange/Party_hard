package main

import (
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// CEMENTERIO (v1.4.0, user): the graveyard ALWAYS gets its night of the living dead. It is part of the level, like the
// venue acts (venueacts.go): NOT a party event (the mean of events per party does not move), with its own dice
// (gyDice, never g.rng).
//
//   1. ZOMBIES: a while in, 2-3 dead rise from the graves (new bodies when the party has room, cap.go; else the guest
//      standing nearest to the grave is bitten). They are the zombies of the game (gZombie, updateZombie in events.go:
//      slow, a stab puts them down, deadly to the player on contact) but these ones EAT BRAINS: their bite kills the
//      guest (at most gyBrainCap per party; after that the bite only scares, it never turns anyone, so there is no
//      outbreak).
//   2. GHOSTS: then 2 ghosts float around the graves. They only FRIGHTEN: guests near one scream and run (gScared);
//      nobody is hurt.
//   3. CAZAFANTASMOS: a little later the ghost hunters of the game (ghostbusterSlugs, events2.go) come in, chase the
//      ghosts and zap them with their beams until they are trapped; in the confusion a beam sometimes fries a guest
//      next to the ghost (at most gyZapCap per party).
//
// Hooks: game.go (updateGraveyard, right after the venue acts) and draw.go (graveItems, with the other items).
// ---------------------------------------------------------------------------

const (
	gyBrainCap   = 5     // guests whose brains the zombies eat in one party, at most
	gyZapCap     = 3     // guests the ghost hunters fry by mistake in one party, at most
	gyZapChance  = 0.18  // per second of beam, the chance that it hits a guest next to the ghost
	gyGhostHP    = 9.0   // seconds of one beam (two beams halve it) to trap a ghost
	gyGhostLife  = 120.0 // a ghost nobody traps fades away after this long
	gyScareR     = 64.0  // a ghost frightens the guests this close
	gyGhostSpeed = 30.0  // px/s
	gyBeamR      = 120.0 // a hunter fires at a ghost this close (same reach as drawGhostBeams)
)

var (
	gyRng      *rand.Rand
	gySeed     int64 // tests: non-zero = deterministic dice
	gyInTests  bool  // tests: let it happen under go test (off by default so the other tests' parties stay as they were)
	gyTestsOff bool  // set by the tests' init (testiso_test.go): the graveyard acts are off under go test unless gyInTests
)

func gyDice() *rand.Rand {
	if gyRng == nil {
		sd := gySeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x6a7e
		}
		gyRng = rand.New(rand.NewSource(sd))
	}
	return gyRng
}

type gyGhost struct {
	x, y, t float64
	wp      Vec
	hp      float64
	caught  bool
}

type gyRun struct {
	lvl                 *Level
	started             bool
	zombieAt, ghostAt   float64 // game time each wave comes (0 = done)
	bustAt              float64
	zombies             []*Guest // the brain eaters of this party
	ghosts              []*gyGhost
	crew                []*Guest // the ghost hunters
	brains, zaps        int
	scareT, zapT, crewT float64
}

var gy *gyRun

// gyLevel: the graveyard of the campaign (and its second roll in the random mode).
func gyLevel(def *LevelDef) bool {
	return def != nil && (def.Icon == "grave" || def.Title == "CEMENTERIO")
}

func (g *Game) gyActive() *gyRun {
	if g.lvl == nil {
		return nil
	}
	if gy == nil || gy.lvl != g.lvl {
		gy = nil
		if gyLevel(g.lvl.Def) && (!gyTestsOff || gyInTests) {
			gy = &gyRun{lvl: g.lvl}
		}
	}
	return gy
}

func (g *Game) gyStart(run *gyRun) {
	run.started = true
	r := gyDice()
	run.zombieAt = g.time + 45 + r.Float64()*30
	run.ghostAt = run.zombieAt + 25 + r.Float64()*25
	if fastEv {
		run.zombieAt, run.ghostAt = g.time+3, g.time+6
	}
}

func (g *Game) updateGraveyard(dt float64) {
	if g.mode != ModePlay {
		return
	}
	run := g.gyActive()
	if run == nil {
		return
	}
	if !run.started {
		g.gyStart(run)
	}
	r := gyDice()
	if run.zombieAt > 0 && g.time >= run.zombieAt {
		run.zombieAt = 0
		g.gyRaiseZombies(run, r)
	}
	if run.ghostAt > 0 && g.time >= run.ghostAt {
		run.ghostAt = 0
		g.gySpawnGhosts(run, r)
		run.bustAt = g.time + 15 + r.Float64()*10
		if fastEv {
			run.bustAt = g.time + 4
		}
	}
	if run.bustAt > 0 && g.time >= run.bustAt {
		run.bustAt = 0
		g.gyCallCrew(run)
	}
	g.gyGhostsStep(run, r, dt)
	g.gyCrewStep(run, r, dt)
}

// gyGraves: the tombstones of the level (where the dead rise).
func (g *Game) gyGraves() []Vec {
	var out []Vec
	for _, p := range g.lvl.Props {
		if p.Name == "tombstone" {
			out = append(out, Vec{p.X, p.Y})
		}
	}
	return out
}

func (g *Game) gyRaiseZombies(run *gyRun, r *rand.Rand) {
	graves := g.gyGraves()
	if len(graves) == 0 {
		graves = []Vec{g.lvl.RandomRoam(r), g.lvl.RandomRoam(r), g.lvl.RandomRoam(r)}
	}
	n := 2 + r.Intn(2)
	for i := 0; i < n; i++ {
		at := graves[r.Intn(len(graves))]
		var z *Guest
		if g.guestRoom() > 0 {
			before := len(g.guests)
			g.addGuestAt(Vec{at.X, at.Y + 14})
			if len(g.guests) > before {
				z = g.guests[len(g.guests)-1]
			}
		}
		if z == nil { // the party is full: the dead bite whoever stands nearest to the grave
			best := 1e9
			for _, gu := range g.guests {
				if gu.out() || gu.perf != "" || gu.state == gZombie || gu.state == gLift || g.gyIsCrew(run, gu) {
					continue
				}
				if d := dist(gu.pos, at); d < best {
					best, z = d, gu
				}
			}
		}
		if z == nil {
			continue
		}
		g.makeZombie(z)
		z.icon, z.iconT = "CEREBROS...", 2.5
		run.zombies = append(run.zombies, z)
		g.burst(Vec{at.X, at.Y + 10}, 12, 40, rgb(0x5a4a30), 0.7, 2, -20) // the earth bursts open
	}
	if len(run.zombies) > 0 {
		g.say("¡Los muertos salen de sus tumbas! Quieren cerebros")
		g.snd.Play("alert")
	}
}

// gyBite: a zombie bites a guest. true = it was one of the brain eaters of the graveyard (the bite is handled here:
// it eats the brain, or only scares once the party's cap is reached); false = an ordinary zombie (it turns him).
func (g *Game) gyBite(z, o *Guest) bool {
	run := gy
	if run == nil || run.lvl != g.lvl || !gyHas(run.zombies, z) {
		return false
	}
	if run.brains < gyBrainCap && !g.gyIsCrew(run, o) {
		run.brains++
		g.killGuest(o, false)
		g.sayNews("kill", map[string]string{"actor": "Un zombi", "victim": guestLabel(o)})
		z.icon, z.iconT = "ÑAM, CEREBRO", 2
		g.checkDone()
		return true
	}
	if o.state != gScared && o.perf == "" {
		o.state, o.timer, o.path = gScared, 5+gyDice().Float64()*3, nil
		o.icon, o.iconT = "¡ZOMBI!", 2
	}
	return true
}

func gyHas(list []*Guest, gu *Guest) bool {
	for _, x := range list {
		if x == gu {
			return true
		}
	}
	return false
}

func (g *Game) gyIsCrew(run *gyRun, gu *Guest) bool {
	return gyHas(run.crew, gu)
}

func (g *Game) gySpawnGhosts(run *gyRun, r *rand.Rand) {
	graves := g.gyGraves()
	for i := 0; i < 2; i++ {
		p := g.lvl.RandomRoam(r)
		if len(graves) > 0 {
			p = graves[r.Intn(len(graves))]
		}
		run.ghosts = append(run.ghosts, &gyGhost{x: p.X, y: p.Y, wp: g.lvl.RandomRoam(r), hp: gyGhostHP, t: r.Float64() * 6})
	}
	g.say("¡Fantasmas en el cementerio! Todo el mundo grita")
	g.snd.Play("alert")
}

func (g *Game) gyGhostsStep(run *gyRun, r *rand.Rand, dt float64) {
	run.scareT -= dt
	scare := run.scareT <= 0
	if scare {
		run.scareT = 0.5
	}
	out := run.ghosts[:0]
	for _, gh := range run.ghosts {
		gh.t += dt
		if gh.caught || gh.t > gyGhostLife {
			continue
		}
		d := dist(Vec{gh.x, gh.y}, gh.wp)
		if d < 6 {
			gh.wp = g.lvl.RandomRoam(r)
		} else {
			gh.x += (gh.wp.X - gh.x) / d * gyGhostSpeed * dt
			gh.y += (gh.wp.Y - gh.y) / d * gyGhostSpeed * dt
		}
		if scare {
			g.gyScare(run, gh, r)
		}
		out = append(out, gh)
	}
	run.ghosts = out
}

// gyScare: the guests near the ghost scream and run away. Nobody is hurt.
func (g *Game) gyScare(run *gyRun, gh *gyGhost, r *rand.Rand) {
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" || gu.state == gZombie || gu.state == gLift || gu.state == gScared ||
			g.gyIsCrew(run, gu) || g.pevTaken(gu) || dist(gu.pos, Vec{gh.x, gh.y}) > gyScareR {
			continue
		}
		gu.state, gu.timer, gu.path = gScared, 4+r.Float64()*3, nil
		gu.icon, gu.iconT = gyScreams[r.Intn(len(gyScreams))], 2
	}
}

var gyScreams = []string{"¡AAAAH!", "¡UN FANTASMA!", "¡MAMÁ!", "¡QUE ME COGE!"}

// gyCallCrew: the four ghost hunters come in (the ones already at the party join them).
func (g *Game) gyCallCrew(run *gyRun) {
	if len(run.ghosts) == 0 {
		return
	}
	base := g.lvl.RandomRoam(g.rng)
	for i, slug := range ghostbusterSlugs {
		var gu *Guest
		for _, o := range g.guests { // a famous face is only once per party (famous.go): reuse him
			if !o.out() && guestSlug(o) == slug {
				gu = o
				break
			}
		}
		if gu == nil && g.guestRoom() > 0 {
			f := presetByName(slug)
			if f == nil {
				continue
			}
			spec := f(g.rng)
			gu = &Guest{pos: g.lvl.NearestFree(Vec{base.X + float64(i)*14 - 20, base.Y + float64(i%2)*10}), dir: DirDown,
				cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6,
				state: gIdle, timer: 1, role: i % 2, keep: true}
			g.guests = append(g.guests, gu)
			g.total++
		}
		if gu != nil {
			run.crew = append(run.crew, gu)
		}
	}
	if len(run.crew) > 0 {
		g.say("¡Llegan los Cazafantasmos! A por los fantasmas")
		g.snd.Play("alert")
	}
}

// gyTarget: the ghost the hunter goes for (the nearest one), nil = none left.
func (run *gyRun) target(p Vec) *gyGhost {
	var best *gyGhost
	bd := 1e9
	for _, gh := range run.ghosts {
		if d := dist(p, Vec{gh.x, gh.y}); !gh.caught && d < bd {
			best, bd = gh, d
		}
	}
	return best
}

func (g *Game) gyCrewStep(run *gyRun, r *rand.Rand, dt float64) {
	if len(run.crew) == 0 {
		return
	}
	run.crewT -= dt
	walk := run.crewT <= 0
	if walk {
		run.crewT = 0.6
	}
	run.zapT -= dt
	zapNow := run.zapT <= 0
	if zapNow {
		run.zapT = 1
	}
	for i, gu := range run.crew {
		if gu.out() || gu.state == gFlee || gu.state == gRun || gu.state == gZombie {
			continue
		}
		gh := run.target(gu.pos)
		if gh == nil {
			continue
		}
		d := dist(gu.pos, Vec{gh.x, gh.y})
		if walk {
			if d > 80 {
				tgt := Vec{gh.x + float64(i-2)*22, gh.y + 40}
				gu.path = g.lvl.PathG(gu.pos, g.lvl.NearestFree(tgt))
				gu.state, gu.next, gu.dur, gu.hurry = gGoto, gIdle, 0.5, true
			} else {
				gu.state, gu.timer = gIdle, 0.7
			}
		}
		if d > gyBeamR {
			continue
		}
		gh.hp -= dt
		if gh.hp <= 0 && !gh.caught {
			gh.caught = true
			g.burst(Vec{gh.x, gh.y - 20}, 30, 100, rgb(0x90ffb0), 0.8, 2, 0)
			g.say("¡Fantasma atrapado!")
			g.snd.Play("win")
		}
		if zapNow && run.zaps < gyZapCap && r.Float64() < gyZapChance {
			g.gyZap(run, gu, gh)
		}
	}
	if run.target(Vec{}) == nil && len(run.ghosts) == 0 { // every ghost trapped: the hunters stay for the party
		for _, gu := range run.crew {
			if !gu.out() {
				gu.hurry = false
			}
		}
		run.crew = nil
	}
}

// gyZap: by mistake, the beam fries a guest standing next to the ghost (or on the way to it).
func (g *Game) gyZap(run *gyRun, hunter *Guest, gh *gyGhost) {
	if run.zaps >= gyZapCap {
		return
	}
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" || gu.state == gLift || g.gyIsCrew(run, gu) {
			continue
		}
		if dist(gu.pos, Vec{gh.x, gh.y}) < 40 || segDist(gu.pos, hunter.pos, Vec{gh.x, gh.y}) < 8 {
			run.zaps++
			g.killGuest(gu, false)
			g.sayNews("kill", map[string]string{"actor": guestLabel(hunter), "victim": guestLabel(gu)})
			g.bubbleAt(Vec{hunter.pos.X, hunter.pos.Y - 30}, "¡UY! ESE NO ERA", 1.6)
			g.burst(Vec{gu.pos.X, gu.pos.Y - 12}, 14, 80, rgb(0xfff0a0), 0.5, 2, 0)
			g.snd.Play("zap")
			g.checkDone()
			return
		}
	}
}

// graveItems: the ghosts and the beams, sorted with the rest of the party.
func (g *Game) graveItems(dst *ebiten.Image, items []drawItem) []drawItem {
	run := gy
	if run == nil || run.lvl != g.lvl {
		return items
	}
	for _, gh := range run.ghosts {
		e := &Event{x: gh.x, y: gh.y, t: gh.t}
		for _, gu := range run.crew {
			if !gu.out() && run.target(gu.pos) == gh {
				e.crew = append(e.crew, gu)
			}
		}
		items = append(items, drawItem{gh.y + 40, func() {
			g.drawGhostBeams(dst, e)
			g.drawGhost(dst, e)
		}})
	}
	return items
}
