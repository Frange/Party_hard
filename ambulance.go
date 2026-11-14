package main

import (
	"flag"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Ambulances and the bodies they come for (testers, v1.0.0: "the ambulance gets there and goes
// away without doing anything; the police too"). The rules the owner set (2026-11-11):
//   - A unit that gets there ALWAYS works before going. No ambulance is sent without a valid body
//     (ambFree: reported, not carried, not falling, not claimed, not reserved). Each body an
//     ambulance is sent for is RESERVED for it (Body.resv): cleaners, the stray dog and the player
//     leave it alone. If its bodies vanish on the way, the ambulance takes other free ones or turns
//     back before it gets there; a paramedic who got there and finds nothing examines the place
//     (kneeling, at least medicMinAct seconds) before he goes.
//   - The police only COVER the bodies (all of them); carrying them away is the paramedics' job,
//     40 % faster with a body on their shoulder, one at a time; 5+ bodies bring two ambulances,
//     each with a crew for its share.
//   - How they arrive: an ambulance by road, a little spaceship with a cross at the station (the
//     transit ships, vessels.go), a medical helicopter on Isla Nublada that lands on the helipad,
//     and on the pirate ship a rescue launch that comes alongside the hull: the paramedics climb a
//     rope ladder hanging from the rail (up, and down again with the body).
// ---------------------------------------------------------------------------

const (
	medicMinAct     = 6.0        // seconds a paramedic who found nothing to carry examines the place before leaving
	medicCarrySpeed = 32.0 * 1.4 // a paramedic with a body: +40 % on the old 32 px/s
	copMinStay      = 20.0       // the minimum on scene of an officer while there is a suspect
	ladderClimbT    = 1.3        // seconds up or down the rope ladder of the pirate ship
	ladderBoatY     = 446.0      // the rescue launch lies alongside the south side of the galleon (waterline ~y 442)
	ladderMaxRes    = 6          // bodies an ambulance takes over when its own ones vanished on the way
)

// -ambnow N (debug, captures): at 3 s, N guests near the start die, are found and the police and
// the ambulance are called (like any debug flag it freezes the progress: main.go debugFlagsActive).
var ambNowF = flag.Int("ambnow", 0, "debug: N bodies found and police + ambulance called at 3 s")

// ambDebugTick runs -ambnow once.
func (g *Game) ambDebugTick() {
	if ambNowF == nil || *ambNowF <= 0 || g.ambDbgDone || g.time < 3 {
		return
	}
	g.ambDbgDone = true
	n := 0
	for _, gu := range g.guests {
		if n < *ambNowF && !gu.out() && gu.perf == "" && !g.lvl.InPool(gu.pos) {
			g.killGuest(gu, false)
			n++
		}
	}
	for _, b := range g.bodies {
		b.reported, b.byPlayer = true, false
	}
	if len(g.bodies) > 0 {
		g.raisePolice(g.bodies[0].pos, false)
		g.ambCalled, g.pendingAmb = true, g.time+1
	}
}

// ladder climbing phases (Cop.climbPh)
const (
	ladNone = iota
	ladUp
	ladDeliver
	ladLeave
)

// ladderSlots: where the rescue launches lie along the hull (left of the dock and the gangplank).
var ladderSlots = []float64{236, 388}

// medicVeh: a vehicle that brings paramedics for the bodies (not the 2P revive ambulance).
func medicVeh(v *Vehicle) bool { return (v.kind == 1 || v.kind == 3) && !v.revive }

// ambLive: an ambulance still in the game and not leaving (on its way or parked).
func (g *Game) ambLive(v *Vehicle) bool {
	if v == nil || v.dead || v.state == 2 {
		return false
	}
	for _, x := range g.veh {
		if x == v {
			return true
		}
	}
	return false
}

// bodyResv: the ambulance a body is reserved for (a gone or wrecked one is forgotten).
func (g *Game) bodyResv(b *Body) *Vehicle {
	if b.resv != nil && !g.ambLive(b.resv) {
		b.resv = nil
	}
	return b.resv
}

// bodyHeld: a body somebody of the emergency services is on: reserved for an ambulance or
// claimed by a paramedic. Cleaners, the dog and the player keep their hands off it.
func (g *Game) bodyHeld(b *Body) bool {
	return g.bodyResv(b) != nil || (b.claimed != nil && b.claimed.medic)
}

// ambFree: THE predicate of a body an ambulance may be sent for.
func (g *Game) ambFree(b *Body) bool {
	return b.reported && !b.carried && b.fallT <= 0 && b.claimed == nil && !b.noReach && g.bodyResv(b) == nil
}

// ambFreeN: how many bodies an ambulance could be sent for right now.
func (g *Game) ambFreeN() int {
	n := 0
	for _, b := range g.bodies {
		if g.ambFree(b) {
			n++
		}
	}
	return n
}

// ambResvOK: a body still worth the trip of v (reserved for it and still lying there).
func ambResvOK(b *Body, v *Vehicle) bool {
	return b.resv == v && !b.carried && b.fallT <= 0
}

// ambReserved: how many bodies are still waiting for v.
func (g *Game) ambReserved(v *Vehicle) int {
	n := 0
	for _, b := range g.bodies {
		if ambResvOK(b, v) {
			n++
		}
	}
	return n
}

// ambReserve marks up to max free bodies for v and says how many it got.
func (g *Game) ambReserve(v *Vehicle, max int) int {
	n := 0
	for _, b := range g.bodies {
		if n >= max {
			break
		}
		if g.ambFree(b) {
			b.resv = v
			n++
		}
	}
	return n
}

// ambSteal: the second ambulance of a big pile takes over up to max bodies the first one has not
// started on yet (reserved, nobody walking to them).
func (g *Game) ambSteal(v *Vehicle, max int) int {
	n := 0
	for _, b := range g.bodies {
		if n >= max {
			break
		}
		if r := g.bodyResv(b); r != nil && r != v && b.claimed == nil && !b.carried && b.fallT <= 0 {
			b.resv = v
			n++
		}
	}
	return n
}

// ambWaiting: bodies reserved for some ambulance that nobody has started on.
func (g *Game) ambWaiting() int {
	n := 0
	for _, b := range g.bodies {
		if g.bodyResv(b) != nil && b.claimed == nil && !b.carried {
			n++
		}
	}
	return n
}

// ambDispatchOne sends one ambulance with the arrival of the level; nil when it could not leave
// yet (every helicopter pad busy: dispatch queued a retry).
func (g *Game) ambDispatchOne() *Vehicle {
	before := len(g.veh)
	g.dispatch(true)
	if len(g.veh) > before {
		return g.veh[len(g.veh)-1]
	}
	return nil
}

// ambSend: the pending call goes out. Never without a body to fetch: with nothing left the call is off.
func (g *Game) ambSend() {
	free := g.ambFreeN()
	if g.ambCount == 0 {
		if free == 0 {
			g.ambMaybeDone()
			return
		}
		k := 1
		if free >= ambTwinBodies || g.ambTwin {
			if free >= 2 {
				k = 2 // a big pile: two ambulances at once (never more than two)
			}
		}
		for i := 0; i < k && free > 0; i++ {
			v := g.ambDispatchOne()
			if v == nil {
				return
			}
			share := (free + (k - i) - 1) / (k - i)
			free -= g.ambReserve(v, share)
			g.ambCount++
			g.ambAt = g.time
		}
		g.ambTwin = false
		return
	}
	// the second ambulance of a big pile: the free bodies and half of what still waits for the first
	wait := g.ambWaiting()
	if free == 0 && wait < 2 {
		g.ambMaybeDone()
		return
	}
	v := g.ambDispatchOne()
	if v == nil {
		return
	}
	g.ambReserve(v, 99)
	g.ambSteal(v, wait/2)
	g.ambCount++
	g.ambAt = g.time
}

// ambMaybeDone: no ambulance and no paramedic is left on the case: the call is over.
func (g *Game) ambMaybeDone() {
	if g.pendingAmb >= 0 {
		return
	}
	for _, v := range g.veh {
		if medicVeh(v) && !v.dead && v.state != 2 {
			return
		}
	}
	for _, c := range g.cops {
		if c.medic {
			return
		}
	}
	g.ambCalled, g.ambCount, g.ambTwin = false, 0, false
	g.medicsDone()
}

// ambEnRoute: an ambulance on its way whose bodies have all gone (cleaners, dog, a hidden body...)
// takes other free ones, or turns back BEFORE it gets there. Bodies found while an ambulance is on
// the case are reserved for it (the one with the fewest), so nobody else takes them.
func (g *Game) ambEnRoute() {
	g.ambDebugTick()
	var live []*Vehicle
	for _, v := range g.veh {
		if !medicVeh(v) || !g.ambLive(v) {
			continue
		}
		if v.state == 0 || !v.spawned {
			if g.ambReserved(v) == 0 && g.ambReserve(v, ladderMaxRes) == 0 {
				v.state, v.spawned = 2, true // nothing left to fetch: it turns back on the way
				g.ambMaybeDone()
				continue
			}
			live = append(live, v)
			continue
		}
		for _, c := range g.cops {
			if c.veh == v && c.state != cLeave {
				live = append(live, v)
				break
			}
		}
	}
	if len(live) == 0 {
		return
	}
	for _, b := range g.bodies {
		if !g.ambFree(b) {
			continue
		}
		best, bn := live[0], 1<<30
		for _, v := range live {
			if n := g.ambReserved(v); n < bn {
				best, bn = v, n
			}
		}
		b.resv = best
	}
}

// ambCrewAtArrival: how many paramedics get out of v (0: nothing to fetch, it turns back).
func (g *Game) ambCrewAtArrival(v *Vehicle) int {
	r := g.ambReserved(v)
	if r == 0 {
		r = g.ambReserve(v, ladderMaxRes)
	}
	if r == 0 {
		return 0
	}
	return clampI((r+1)/2, 1, 3)
}

// medicCanTake: a body this paramedic may go for.
func (g *Game) medicCanTake(c *Cop, b *Body) bool {
	if b.carried || b.fallT > 0 {
		return false
	}
	if b.claimed != nil && b.claimed != c && b.claimed.medic {
		return false
	}
	r := g.bodyResv(b)
	return r != nil || b.reported
}

// medicNext: what a paramedic does next: the nearest body for his ambulance (then any free one);
// with nothing to carry he examines the place for a while, and only then he goes back.
func (g *Game) medicNext(c *Cop) {
	var best *Body
	bestD := 0.0
	for _, b := range g.bodies {
		if !g.medicCanTake(c, b) {
			continue
		}
		d := dist(c.pos, b.pos)
		if r := g.bodyResv(b); r != nil && r != c.veh {
			d += 400 // another ambulance's: only when nothing of his own is left
		}
		if d > 16 && len(g.lvl.Path(c.pos, b.pos)) == 0 {
			b.noReach = true // out of reach: it stays covered (no ambulance comes for it again)
			if b.resv == c.veh {
				b.resv = nil
			}
			continue
		}
		if best == nil || d < bestD {
			best, bestD = b, d
		}
	}
	if best != nil {
		if best.claimed != nil && best.claimed != c {
			g.releaseCop(best.claimed) // an officer still covering it: the paramedic takes over
		}
		best.claimed = c
		if g.bodyResv(best) == nil {
			best.resv = c.veh
		}
		c.lastSeen = best.pos
		c.target, c.state = best, cCollect
		c.path = g.lvl.Path(c.pos, best.pos)
		return
	}
	if !c.mayLeave() {
		// he got here and the body is gone: he examines the place (kneeling) before he goes
		if c.state != cExamine {
			c.state, c.timer = cExamine, 0
			c.path = nil
			if c.lastSeen != (Vec{}) && dist(c.pos, c.lastSeen) > 16 {
				c.path = g.lvl.Path(c.pos, g.lvl.NearestFree(c.lastSeen))
			}
		}
		return
	}
	c.state = cLeave
	c.path = g.lvl.Path(c.pos, g.vehDoor(c))
}

// copCoverTarget: the nearest found body nobody has covered or is covering.
func (g *Game) copCoverTarget(c *Cop) *Body {
	var best *Body
	for _, b := range g.bodies {
		if !b.reported || b.bagged || b.carried || b.fallT > 0 || b.claimed != nil {
			continue
		}
		if best == nil || dist(c.pos, b.pos) < dist(c.pos, best.pos) {
			best = b
		}
	}
	return best
}

// caseHasSuspect: somebody is wanted or held: only then the officers keep their 20 s on scene.
func (g *Game) caseHasSuspect() bool {
	if g.wanted > 0 || g.suspect != nil || jailHold {
		return true
	}
	for _, pl := range g.players() {
		if pl != nil && pl.jail.sus {
			return true
		}
	}
	return false
}

// copMayLeave: may this officer or paramedic go back now? A paramedic only after working; an officer
// at once when nobody is suspected, otherwise after the 20 s minimum (and never while somebody is held).
func (g *Game) copMayLeave(c *Cop) bool {
	if c.medic {
		return c.mayLeave()
	}
	if jailHold {
		return false
	}
	if !g.caseHasSuspect() {
		return true
	}
	return c.mayLeave()
}

// standDown: a closed case or a higher group taking over. An officer who has been on scene for
// less than copMinStay seals off the place (evidence, tape) until he has; the rest go back.
func (g *Game) standDown(c *Cop) {
	g.releaseCop(c)
	if c.onSceneT < copMinStay {
		c.state, c.wait, c.path = cCordon, 0, nil
		return
	}
	c.state = cLeave
	c.path = g.lvl.Path(c.pos, g.vehDoor(c))
}

// updateCordon: an officer who seals off the place walks slowly around it until his 20 s are done.
func (g *Game) updateCordon(c *Cop, dt float64) {
	if c.onSceneT >= copMinStay {
		g.crewLeave(c) // the crew goes back together (copcrew.go)
		return
	}
	if c.wait > 0 {
		c.wait -= dt
		return
	}
	if len(c.path) == 0 {
		a := g.rng.Float64() * 2 * math.Pi
		c.path = g.lvl.Path(c.pos, g.lvl.NearestFree(Vec{c.pos.X + math.Cos(a)*40, c.pos.Y + math.Sin(a)*30}))
		if len(c.path) == 0 {
			c.wait = 1
			return
		}
	}
	if g.copWalk(c, copSweepSpeed*0.6, dt) {
		c.wait = 1 + g.rng.Float64()*1.5
		c.dir = Dir(g.rng.Intn(4))
	}
}

// updateExamine: a paramedic with nothing to carry walks to where the body was and kneels there.
func (g *Game) updateExamine(c *Cop, dt float64) {
	if len(c.path) > 0 {
		g.copWalk(c, medicSpeed, dt)
		return
	}
	c.actT += dt
	c.dir = DirUp
	if c.mayLeave() {
		g.copNext(c)
	}
}

// ---- the island helicopter and the rescue launch of the pirate ship ---------------------------

// ambSpecialDispatch: the levels where the ambulance does not come by road (the station needs no
// special case: its transit ships already bring one, a little ship with the red cross).
func (g *Game) ambSpecialDispatch() bool {
	def := g.lvl.Def
	switch {
	case g.isIsland() && len(jpPads) > 0:
		n := 0
		for _, v := range g.veh {
			if v.kind == 3 && !v.dead && v.state != 2 {
				n++
			}
		}
		p := jpPads[0]
		tx := p.X - float64(n%2)*84 // a second one hovers down next to the pad
		g.veh = append(g.veh, &Vehicle{kind: 3, x: tx, y: -130, tx: tx, ty: p.Y, pad: -1, unit: g.unitFor(true)})
		g.snd.Play("siren")
		return true
	case pirateBay(def):
		slot := ladderSlots[0]
		for _, s := range ladderSlots {
			used := false
			for _, v := range g.veh {
				if v.ladder && !v.dead && v.state != 2 && v.tx == s {
					used = true
				}
			}
			if !used {
				slot = s
				break
			}
		}
		g.veh = append(g.veh, &Vehicle{kind: 1, x: -110, y: ladderBoatY, tx: slot, ty: ladderBoatY, ladder: true, unit: g.unitFor(true)})
		g.snd.Play("siren")
		return true
	}
	return false
}

// ladderStep moves the rescue launch: in from the left along the hull, and back out. gone: off screen.
func ladderStep(v *Vehicle, dt float64) (gone bool) {
	switch v.state {
	case 0:
		v.x += 110 * dt
		if v.x >= v.tx {
			v.x, v.state = v.tx, 1
		}
	case 2:
		v.x -= 110 * dt
		return v.x < -130
	}
	return false
}

// ladderX: where the rope ladder hangs (over the aft deck of the launch).
func ladderX(v *Vehicle) float64 { return v.tx - 22 }

// ladderDoor: the spot on deck at the top of the rope ladder.
func (g *Game) ladderDoor(v *Vehicle) Vec {
	return g.lvl.NearestFree(Vec{ladderX(v), pirBot - 10})
}

// ladderLen: how far the paramedic climbs from the deck down to the launch.
func ladderLen(v *Vehicle, door Vec) float64 { return math.Max(16, v.ty-12-door.Y) }

// ladderStart: a paramedic of a launch starts down the ladder (with the body, or to go home).
func (g *Game) ladderStart(c *Cop, ph int) bool {
	if c.veh == nil || !c.veh.ladder || c.veh.dead {
		return false
	}
	c.climbPh, c.climb = ph, ladderClimbT
	c.dir = DirDown
	return true
}

// copClimb runs a paramedic on the rope ladder; true while he is on it (nothing else happens).
func (g *Game) copClimb(c *Cop, dt float64) bool {
	if c.climbPh == ladNone {
		return false
	}
	c.moving, c.curSpd = true, 30
	c.anim += dt * stridePhase(30)
	if c.climbPh == ladDeliver {
		c.actT += dt
	}
	if c.climb -= dt; c.climb > 0 {
		return true
	}
	switch c.climbPh {
	case ladUp:
		c.climbPh = ladNone
		c.dir = DirDown
		if c.state == cReturn && c.carry == nil { // he has just handed the body over in the launch
			g.copNext(c)
		}
	case ladDeliver:
		if c.carry != nil {
			g.removeBody(c.carry)
			c.carry = nil
			c.carried++
		}
		c.climbPh, c.climb = ladUp, ladderClimbT
		c.dir = DirUp
	case ladLeave:
		c.climbPh = ladNone
		g.copBoard(c)
	}
	return true
}

// copDrawPos: where an officer is drawn (a paramedic on the rope ladder hangs below the rail).
func (g *Game) copDrawPos(c *Cop) Vec {
	if c.climbPh == ladNone || c.veh == nil {
		return c.pos
	}
	l := ladderLen(c.veh, c.pos)
	k := clampF(c.climb/ladderClimbT, 0, 1) // 1 at the start of the climb
	off := l * k                            // climbing up: from the bottom to the top
	if c.climbPh != ladUp {
		off = l * (1 - k) // going down
	}
	return Vec{c.pos.X, c.pos.Y + math.Round(off)}
}

// copBoard: an officer or paramedic is back in his vehicle; it leaves once the whole crew is aboard.
func (g *Game) copBoard(c *Cop) {
	for i, x := range g.cops {
		if x == c {
			g.cops = append(g.cops[:i], g.cops[i+1:]...)
			break
		}
	}
	crew := false
	for _, o := range g.cops {
		if o.veh == c.veh {
			crew = true
		}
	}
	if !crew && c.veh != nil {
		c.veh.state = 2
	}
	if c.medic {
		g.ambMaybeDone()
	} else if !g.policeOnCase() {
		// Do not clear the case while another type of authority is still arriving or searching.
		g.endPoliceCall()
	}
}

// drawRopeLadder: the rope ladder from the rail of the galleon down to the parked launch.
func (g *Game) drawRopeLadder(dst *ebiten.Image, v *Vehicle) {
	if !v.ladder || v.state != 1 {
		return
	}
	x := math.Round(g.wx(ladderX(v)))
	top := math.Round(g.wy(pirBot - 2))
	bot := math.Round(g.wy(v.ty - 10))
	rope := color.RGBA{0xc8, 0xa0, 0x60, 0xff}
	dark := color.RGBA{0x5a, 0x3e, 0x22, 0xff}
	fillRect(dst, x-5, top, 1, bot-top, rope)
	fillRect(dst, x+4, top, 1, bot-top, rope)
	fillRect(dst, x-4, top, 1, bot-top, dark)
	fillRect(dst, x+5, top, 1, bot-top, dark)
	for y := top + 3; y < bot; y += 5 {
		fillRect(dst, x-5, y, 10, 1, rope)
		fillRect(dst, x-5, y+1, 10, 1, dark)
	}
	fillRect(dst, x-6, top-2, 3, 3, dark) // the two knots on the rail
	fillRect(dst, x+4, top-2, 3, 3, dark)
}

// drawMedicBeacon: the flashing red/blue beacon of the ambulance ships and launches.
func (g *Game) drawMedicBeacon(dst *ebiten.Image, v *Vehicle, top float64) {
	if v.kind != 1 {
		return
	}
	col := color.RGBA{200, 30, 30, 150}
	dx := -6.0
	if int(g.time*4)%2 == 1 {
		col, dx = color.RGBA{40, 80, 230, 150}, 6
	}
	g.glowAt(dst, Vec{v.x + dx, top}, 14, col)
}
