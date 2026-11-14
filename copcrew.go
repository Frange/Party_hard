package main

import "math"

// ---------------------------------------------------------------------------
// CREWS (2026-11-12, owner's feedback). Every police type comes as a crew of its vehicle: the local police 1 or 2
// (by the bodies found), the FBI always 2, the army 3, SWAT 4 and the Pawer Rangers 5 (groupStat, police_groups.go).
// None of them carries bodies: they cover the ones found and search the place (the ambulance and the cleaners take
// the bodies away).
//
//   - A crew goes back COMPLETE or not at all: an officer who is done searching keeps sweeping until every mate of
//     his vehicle is done too (crewReady), and then they all walk back together (crewLeave). Only a dead mate (his
//     body stays on the floor) makes the crew smaller.
//   - Any crew that goes copIdleLeave seconds without seeing a body still uncovered or the suspect goes back too,
//     whatever its type (the FBI that "never gives up" included), until somebody calls again.
//   - Army and SWAT carry their machine guns in their hands (copGunCS) and may use them, at most ONCE among all of
//     them in a call (copBrutal, killer.go); after that shot everybody reviews the place for copGunReview seconds and
//     the whole response goes back until the next call.
// ---------------------------------------------------------------------------

const (
	copIdleLeave = 25.0 // seconds without a body uncovered or the suspect in sight: the crew leaves
	copGunReview = 6.0  // seconds of looking around after the one burst of a call
	copGunChance = 0.3  // chance that a ready army / SWAT officer really opens fire when his moment comes
)

// copCrew: the officers of c's vehicle still on scene (c included). An officer whose vehicle was wrecked is a crew
// of his own.
func (g *Game) copCrew(c *Cop) []*Cop {
	if c == nil || c.veh == nil || c.medic {
		return []*Cop{c}
	}
	out := make([]*Cop, 0, 5)
	for _, o := range g.cops {
		if o.veh == c.veh && !o.medic {
			out = append(out, o)
		}
	}
	return out
}

// copDone: this officer has finished his part (or is already on his way back).
func (g *Game) copDone(o *Cop) bool {
	switch o.state {
	case cLeave:
		return true
	case cSweep, cSearch, cTired, cCordon:
		return o.searchT >= o.maxSearch && g.copMayLeave(o)
	}
	return false
}

// crewReady: every officer of c's crew is done, so they may all go back.
func (g *Game) crewReady(c *Cop) bool {
	if !g.copMayLeave(c) {
		return false
	}
	for _, o := range g.copCrew(c) {
		if o != c && !g.copDone(o) {
			return false
		}
	}
	return true
}

// crewLeave: the whole crew of c walks back to its vehicle (the mates that are still working too).
func (g *Game) crewLeave(c *Cop) {
	for _, o := range g.copCrew(c) {
		if o.state == cLeave || o.climb > 0 {
			continue
		}
		g.releaseCop(o)
		o.state = cLeave
		o.path = g.lvl.Path(o.pos, g.vehDoor(o))
	}
}

// copIdleTick: the idle clock of an officer (updateCop, every frame). It starts again whenever he sees the suspect
// or a body nobody has covered, or while he is covering one, chasing or arresting. When the whole crew has been
// idle for copIdleLeave seconds, it goes back. True when the crew was sent away now.
func (g *Game) copIdleTick(c *Cop, dt float64, sees bool) bool {
	if c.medic || c.state == cLeave {
		return false
	}
	switch c.state {
	case cChase, cArrest, cBagHere, cSlash:
		c.idleT = 0
		return false
	}
	if sees {
		c.idleT = 0
		return false
	}
	c.idleT += dt
	if c.idleT < copIdleLeave {
		return false
	}
	// only now the costly look around for an uncovered body (once a second)
	if math.Mod(c.idleT, 1) < dt {
		for _, b := range g.bodies {
			if !b.bagged && !b.carried && g.sees(c.pos, c.dir, b.pos, copRange, copDot) {
				c.idleT = 0
				return false
			}
		}
	}
	if jailHold {
		return false
	}
	for _, o := range g.copCrew(c) {
		if o != c && o.state != cLeave && o.idleT < copIdleLeave {
			return false
		}
	}
	g.crewLeave(c)
	return true
}

// copGunReviewAll: the burst of a call has been fired: everybody on scene looks around a little and then the whole
// response goes back (until the next call).
func (g *Game) copGunReviewAll() {
	for _, o := range g.cops {
		if o.medic || o.state == cLeave {
			continue
		}
		o.maxSearch = math.Min(o.maxSearch, o.searchT+copGunReview)
		o.idleT = math.Max(o.idleT, copIdleLeave-copGunReview)
	}
}

// copGunUnit: the types that carry a machine gun (army and SWAT).
func copGunUnit(kind int) bool { return kind == unitMilitary || kind == unitSWAT }

// copGunCache: the look of a soldier / SWAT officer with his machine gun, made once per look.
var copGunCache = map[*CharSet]*CharSet{}

// copGunCS: cs with a machine gun in the hands (army and SWAT). A drawn rig, a costume or a look that already holds
// something is left as it is.
func copGunCS(cs *CharSet, kind int) *CharSet {
	if cs == nil || !copGunUnit(kind) || cs.rig != nil || cs.Spec == nil || cs.Spec.Skin.A == 0 || cs.Spec.Carry != 0 || cs.Spec.Weapon != 0 {
		return cs
	}
	if a, ok := copGunCache[cs]; ok {
		return a
	}
	if len(copGunCache) > 64 { // the looks of an old session: start again
		copGunCache = map[*CharSet]*CharSet{}
	}
	sp := *cs.Spec
	sp.Carry = HoldRifle
	a := newCharSet(&sp)
	copGunCache[cs] = a
	return a
}
