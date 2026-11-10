package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE WRONG SUSPECT (user, 2026-11-10: "a veces, que culpen a alguna persona que pasaba por al lado de un cuerpo
// creyendo que es sospechoso; sólo de vez en cuando").
//
//   - When an officer is working a scene (a reported body lying around) and an ordinary guest is walking or standing
//     within blameNear px of that body, the police MAY take him for the suspect: an officer goes for him shouting
//     "¡ALTO! ¿ES ESE EL SOSPECHOSO?", the guest protests ("¡YO NO HE SIDO!"), he is cuffed and walked to the patrol
//     car, checked at the door and RELEASED ("VALE... PUEDE IRSE" / "¡QUÉ INJUSTICIA!"): he goes back to the party.
//     He never dies, never leaves, never becomes a body: kills, the HUD x/y, the stars and the end of the party are
//     not touched (he is never gone, so aliveCount and the guest totals stay as they were).
//   - Rare: one roll per police call (blameChance), never in the first call of the party, never in the call right after
//     the one that had it, at most blameMaxParty per party and blameCool seconds of party time between two. Own dice
//     (blameRng): g.rng is not moved. Tests: blameReseed / blameForce; debug: -blamenow (dbgBlame).
//   - Comic advantage: while the police deal with him (blameGrace seconds from the shout) they do not take an interest in
//     the players (copInterest, twojail.go): an ongoing chase loses sight and turns into a search. A player carrying a
//     body is still chased. Nothing else changes: the 20 s minimum on scene (mayLeave), the stars and the medics.
//   - Safety: anything that breaks the scene (the officer dies or is sent home, a chase, the special killer, the guest
//     dies or leaves, a new level, a time-out) ends it at once and gives the guest back to the party; nothing is left
//     hanging (no claimed body, no cop or car waiting for him).
// ---------------------------------------------------------------------------

const (
	blameChance   = 0.15  // chance per (eligible) police call
	blameCool     = 180.0 // seconds of party time between two
	blameMaxParty = 2     // at most this many per party
	blameNear     = 90.0  // the guest is this close to the body
	blameCopNear  = 160.0 // ... and the officer is this close to the same body
	blameGrace    = 9.0   // seconds the police forget about the players
	blameTimeout  = 32.0  // whole scene: never longer than this
	blameCopWord  = "¡ALTO! ¿ES ESE EL SOSPECHOSO?"
	blameInnWord  = "¡YO NO HE SIDO!"
	blameFreeCop  = "VALE... PUEDE IRSE"
	blameFreeInn  = "¡QUÉ INJUSTICIA!"
)

const (
	blApproach = iota // the officer walks up to him
	blCuff            // both still: cuffs on, the guest protests
	blEscort          // walked to the patrol car
	blCheck           // checked at the car door
)

type blameSt struct {
	lvl        *Level // the party this belongs to (a new level starts it all again)
	called     bool   // g.policeCalled last frame (to count the calls)
	callN      int    // police calls of this party
	rolled     int    // the call whose dice were already thrown
	lastCall   int    // the call that had the last one
	count      int    // how many this party
	lastT      float64
	active     bool
	cop        *Cop
	inn        *Guest
	phase      int
	timer      float64 // time in the current phase
	startT     float64
	door       Vec
	graceUntil float64
	why        string  // how the last one ended (tests and the -blamenow log)
	trail      []Vec   // where the officer walked: the passer-by follows the same steps (doors, corners)
	wanted     float64 // g.wanted last frame (a rise = a real lead)
}

var (
	blameRng      *rand.Rand // own dice (never g.rng)
	blameSeed     int64      // tests: non-zero = deterministic dice
	blameForce    bool       // tests: the dice always say yes (the rules still apply)
	dbgBlame      bool       // -blamenow: a body, a passer-by and the police at once; the rules and dice are skipped
	dbgBlameSetup bool       // -blamenow: the scene still has to be set up (blameDemo)
	dbgBlameLog   bool       // -blamenow: where the scene is, on stderr (to crop the screenshots)
	blameLogT     float64
)

// blameDemo (-blamenow): two seconds in, the guest nearest the player falls, another one stands next to the body and
// the police are called about it. Debug flags freeze the progress.
func (g *Game) blameDemo() {
	if !dbgBlameSetup || g.time < 2 || g.p == nil {
		return
	}
	dbgBlameSetup = false
	var v, w *Guest
	for _, gu := range g.guests {
		if g.blameEligible(gu) && (v == nil || dist(gu.pos, g.p.pos) < dist(v.pos, g.p.pos)) {
			v = gu
		}
	}
	if v == nil {
		return
	}
	for _, gu := range g.guests {
		if gu != v && g.blameEligible(gu) && (w == nil || dist(gu.pos, v.pos) < dist(w.pos, v.pos)) {
			w = gu
		}
	}
	g.killGuest(v, false)
	b := g.bodies[len(g.bodies)-1]
	b.reported = true
	if w != nil {
		for _, dx := range []float64{30, -30, 0} {
			q := Vec{b.pos.X + dx, b.pos.Y + 24 - math.Abs(dx)*0.8}
			if g.lvl.Free(q) && g.lvl.LOS(q, b.pos) {
				w.pos = q
				break
			}
		}
		w.state, w.timer, w.path = gIdle, 60, nil
	}
	g.raisePolice(b.pos, false)
	g.suspect = nil                                         // nobody else to arrest: the officer stays on the scene
	g.p.pos = g.lvl.NearestFree(Vec{b.pos.X, b.pos.Y + 60}) // the camera looks at the scene
}

func blameDice() *rand.Rand {
	if blameRng == nil {
		sd := blameSeed
		if sd == 0 {
			sd = time.Now().UnixNano()
		}
		blameRng = rand.New(rand.NewSource(sd))
	}
	return blameRng
}

// blameReseed: tests. The dice start again with that seed.
func blameReseed(seed int64) { blameSeed, blameRng = seed, nil }

// blameAllowed: the rules of frequency for call callN at time now (dice apart).
func (b *blameSt) blameAllowed(callN int, now float64) bool {
	if callN < 2 || b.count >= blameMaxParty {
		return false // never in the first call of the party, never more than blameMaxParty
	}
	if b.count > 0 && (callN == b.lastCall+1 || now-b.lastT < blameCool) {
		return false // never twice in a row, and not too soon
	}
	return true
}

// blameState gives this party's state (cleared when a new level is loaded).
func (g *Game) blameState() *blameSt {
	if g.blame.lvl != g.lvl {
		g.blame = blameSt{lvl: g.lvl}
	}
	return &g.blame
}

// blameGraceOn: the police are busy with the wrong man and forget about the players.
func (g *Game) blameGraceOn() bool {
	return g.blame.lvl == g.lvl && g.lvl != nil && g.time < g.blame.graceUntil
}

// blameEligible: an ordinary guest going about his party (not famous, not a performer, not the real suspect...).
func (g *Game) blameEligible(gu *Guest) bool {
	if gu == nil || gu.out() || gu.py != nil || gu.armed || gu.keep || gu.perf != "" || gu.name != "" || gu.quirk != "" || gu.vipCard {
		return false
	}
	switch gu.state {
	case gIdle, gGoto, gDance, gDrink, gLounge, gChatGo, gWaitP, gChat:
	default:
		return false
	}
	if gu.poison > 0 || gu.bitten > 0 || gu.inside != nil || gu.queue != nil || gu.offered || gu.reportTo != nil || gu.vipStun > 0 || gu.lift > 0 {
		return false
	}
	if gu.sawP != nil && !gu.warned {
		return false // a witness about to warn the police
	}
	if gu == g.callerGuest || gu == g.suspect || gu == g.lastKiller.guest {
		return false
	}
	return true
}

func blameCopFree(c *Cop) bool {
	if c == nil || c.medic {
		return false
	}
	switch c.state {
	case cSweep, cGoto, cSearch, cBagHere:
		return c.carry == nil
	}
	return false
}

func (g *Game) copPresent(c *Cop) bool {
	for _, o := range g.cops {
		if o == c {
			return true
		}
	}
	return false
}

// blameFind: an officer working a body and a passer-by next to that body.
func (g *Game) blameFind(wide bool) (*Cop, *Guest) {
	near, copNear := blameNear, blameCopNear
	if wide { // -blamenow: anybody around the scene will do
		near, copNear = 260, 400
	}
	for _, b := range g.bodies {
		if !b.reported || b.carried {
			continue
		}
		var cop *Cop
		for _, c := range g.cops {
			if blameCopFree(c) && c.onSceneT >= 1.5 && dist(c.pos, b.pos) < copNear && (cop == nil || dist(c.pos, b.pos) < dist(cop.pos, b.pos)) {
				cop = c
			}
		}
		if cop == nil {
			continue
		}
		var inn *Guest
		for _, gu := range g.guests {
			if g.blameEligible(gu) && dist(gu.pos, b.pos) < near && (inn == nil || dist(gu.pos, b.pos) < dist(inn.pos, b.pos)) {
				inn = gu
			}
		}
		if inn != nil {
			return cop, inn
		}
	}
	return nil, nil
}

// updateBlame: counts the calls, throws the dice once per call and keeps an ongoing scene safe.
func (g *Game) updateBlame() {
	if g.mode != ModePlay || g.lvl == nil {
		return
	}
	g.blameDemo()
	b := g.blameState()
	if g.policeCalled && !b.called {
		b.callN++
	}
	b.called = g.policeCalled
	if b.active {
		if !g.copPresent(b.cop) || b.cop.medic || b.cop.state == cLeave || b.inn.out() || g.time-b.startT > blameTimeout {
			g.blameEnd(false, "update: cop/guest gone or time-out")
		}
		return
	}
	if !g.policeCalled || g.stealth || b.rolled == b.callN && !dbgBlame {
		return
	}
	if !dbgBlame && !b.blameAllowed(b.callN, g.time) {
		return
	}
	cop, inn := g.blameFind(dbgBlame)
	if cop == nil {
		return
	}
	b.rolled = b.callN
	if !dbgBlame && !blameForce && blameDice().Float64() >= blameChance {
		return
	}
	dbgBlame = false
	g.blameStart(cop, inn)
}

func (g *Game) blameStart(c *Cop, gu *Guest) {
	b := g.blameState()
	g.releaseCop(c) // a body he was about to bag goes back to the others
	c.state, c.wait, c.path, c.repath = cSweep, 0, nil, 0
	c.icon, c.iconT = blameCopWord, 4
	if p := gu.partner; p != nil { // a chat or a kiss is cut short: the other one goes on with his party
		if p.partner == gu {
			p.partner = nil
			if p.state == gChat || p.state == gKiss || p.state == gWaitP || p.state == gChatGo {
				p.state, p.timer = gIdle, 1
			}
		}
		gu.partner = nil
	}
	gu.state, gu.path, gu.timer = gIdle, nil, 30
	gu.icon, gu.iconT = "!?", 2
	gu.dir = faceTo(gu.pos, c.pos)
	b.active, b.cop, b.inn, b.phase, b.timer = true, c, gu, blApproach, 0
	b.startT, b.wanted = g.time, g.wanted
	b.count++
	b.lastT, b.lastCall = g.time, b.callN
	b.graceUntil = g.time + blameGrace
	g.snd.Play("alert")
}

// blameEnd gives the guest back to the party and the officer back to his work. released: the happy ending at the car.
func (g *Game) blameEnd(released bool, why string) {
	b := &g.blame
	if !b.active {
		return
	}
	b.why = why
	if dbgBlameLog {
		fmt.Fprintf(os.Stderr, "blame: frame %d end: %s\n", int(g.time*60+0.5), why)
	}
	if gu := b.inn; gu != nil && !gu.out() {
		gu.state, gu.timer, gu.path = gIdle, 1.5, nil
		if released {
			gu.icon, gu.iconT = blameFreeInn, 2.5
		} else if gu.icon == blameInnWord || gu.icon == "!?" {
			gu.icon, gu.iconT = "", 0
		}
	}
	if c := b.cop; c != nil && g.copPresent(c) && c.state == cSweep {
		c.wait, c.path = 0.5, nil
		if released {
			c.icon, c.iconT = blameFreeCop, 2.2
		}
	}
	if !released {
		b.graceUntil = 0 // the scene broke up: the police are back on the case at once
	}
	b.active, b.cop, b.inn, b.trail = false, nil, nil, nil
}

// blameToBody: the officer was just sent (cGoto) to a body lying there, not after a killer.
func blameToBody(g *Game, c *Cop) bool {
	if len(c.path) == 0 {
		return false
	}
	end := c.path[len(c.path)-1]
	for _, b := range g.bodies {
		if !b.carried && dist(b.pos, end) < 24 {
			return true
		}
	}
	return false
}

// copBlameHook: the officer of the scene (called from updateCop). true = this frame is his.
func (g *Game) copBlameHook(c *Cop, dt float64) bool {
	b := &g.blame
	if !b.active || b.cop != c || b.lvl != g.lvl {
		return false
	}
	lead := g.wanted > b.wanted+0.5 // somebody just pointed at the killer: that is no time for passers-by
	b.wanted = g.wanted
	if c.state == cGoto && !lead && blameToBody(g, c) {
		// a witness told him about a body (already reported by telling him): it can wait, he is busy
		c.state, c.path = cSweep, nil
		if b.phase == blEscort {
			c.path = g.lvl.Path(c.pos, b.door)
		}
	}
	if c.state != cSweep { // sent home, chasing, a lead on the killer, the special killer...: the scene is over
		g.blameEnd(false, fmt.Sprintf("cop state %d", c.state))
		return false
	}
	gu := b.inn
	if gu == nil || gu.out() {
		g.blameEnd(false, "guest out")
		return false
	}
	b.timer += dt
	if dbgBlameLog && g.time >= blameLogT {
		blameLogT = g.time + 0.25
		fmt.Fprintf(os.Stderr, "blame: frame %d phase %d passer-by at screen %.0f,%.0f officer %.0f,%.0f\n", int(g.time*60+0.5), b.phase, g.wx(gu.pos.X), g.wy(gu.pos.Y), g.wx(c.pos.X), g.wy(c.pos.Y))
	}
	switch b.phase {
	case blApproach:
		c.repath -= dt
		if c.repath <= 0 || len(c.path) == 0 {
			c.repath = 0.3
			c.path = g.lvl.Path(c.pos, gu.pos)
		}
		g.copWalk(c, copWalkSpeed*groupPace(c.kind), dt)
		if dist(c.pos, gu.pos) < 16 {
			b.phase, b.timer = blCuff, 0
			c.path = nil
			gu.icon, gu.iconT = blameInnWord, 2.6
			g.snd.Play("cuff")
		} else if b.timer > 9 {
			g.blameEnd(false, "not reached") // he could not reach him
			return false
		}
	case blCuff:
		c.dir = faceTo(c.pos, gu.pos)
		if b.timer >= 1.8 {
			b.phase, b.timer = blEscort, 0
			b.door = g.vehDoor(c)
			b.trail = append(b.trail[:0], c.pos)
			c.path = g.lvl.Path(c.pos, b.door)
		}
	case blEscort:
		arr := g.copWalk(c, 30, dt)
		if n := len(b.trail); n == 0 || dist(b.trail[n-1], c.pos) >= 3 {
			b.trail = append(b.trail, c.pos)
		}
		if arr || len(c.path) == 0 || dist(c.pos, b.door) < 14 || b.timer > 20 {
			b.phase, b.timer = blCheck, 0
			c.path = nil
		}
	case blCheck:
		c.dir = faceTo(c.pos, gu.pos)
		if b.timer >= 2.4 {
			g.blameEnd(true, "released")
			return true
		}
	}
	if c.moving {
		c.anim += dt * stridePhase(c.curSpd)
	}
	return true
}

// blameGuest: the wrong suspect (called from updateGuest). true = this frame is his.
func (g *Game) blameGuest(gu *Guest, dt float64) bool {
	b := &g.blame
	if !b.active || b.inn != gu || b.lvl != g.lvl {
		return false
	}
	c := b.cop
	if c == nil {
		return false
	}
	if gu.iconT > 0 {
		gu.iconT -= dt
		if gu.iconT <= 0 {
			gu.icon = ""
		}
	}
	gu.path = nil
	switch b.phase {
	case blApproach, blCuff:
		gu.dir = faceTo(gu.pos, c.pos)
	case blEscort, blCheck:
		// he walks right behind the officer, on the officer's own steps (never through a wall)
		for len(b.trail) > 0 && dist(gu.pos, b.trail[0]) < 1.5 {
			b.trail = b.trail[1:]
		}
		if len(b.trail) == 0 || dist(gu.pos, c.pos) <= 11 {
			gu.dir = faceTo(gu.pos, c.pos)
			break
		}
		to := b.trail[0]
		d := dist(gu.pos, to)
		step := math.Min(d, 45*dt)
		gu.pos = Vec{gu.pos.X + (to.X-gu.pos.X)/d*step, gu.pos.Y + (to.Y-gu.pos.Y)/d*step}
		gu.dir = faceTo(gu.pos, to)
		gu.moving, gu.curSpd = true, 30
		gu.anim += dt * stridePhase(30)
	}
	return true
}

// drawBlame: the cuffs over the wrong suspect while he is held.
func (g *Game) drawBlame(dst *ebiten.Image) {
	b := &g.blame
	if g.mode != ModePlay || !b.active || b.lvl != g.lvl || b.phase == blApproach || b.inn == nil || b.inn.out() {
		return
	}
	g.drawCuffs(dst, b.inn.pos)
}

func init() {
	for es, en := range map[string]string{
		blameCopWord: "FREEZE! IS THAT THE SUSPECT?",
		blameInnWord: "IT WASN'T ME!",
		blameFreeCop: "OK... YOU CAN GO",
		blameFreeInn: "THE INJUSTICE!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
