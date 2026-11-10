package main

// ---------------------------------------------------------------------------
// The guest cap. There are never more guests in a party than the maximum of the
// settings (60 by default), counting everybody who is present and alive at the
// same time: the ones at the start AND the ones who arrive later (bus, taxis,
// train, themed groups, the hands of God, Ghostbusters, famous faces brought by
// a power...). Guards, police, medics, firefighters, technicians, animals, the
// special killer and the T-Rex are not guests and do not count; neither do
// bodies or the people who left (gone).
// Every route that adds guests asks guestRoom() first and brings only the ones
// that fit; enforceGuestCap() is the safety net that catches any route that forgets.
// ---------------------------------------------------------------------------

// capBreaches counts how many times the safety net had to act (the tests require 0).
var capBreaches int

// guestCap is the most guests allowed at the same time.
func (g *Game) guestCap() int { return g.guestMax() + g.twoExtra + g.crowdExtra } // two players: 20-30 more; big crowd: +30

// guestLive counts the guests who are present and alive.
func (g *Game) guestLive() int {
	n := 0
	for _, gu := range g.guests {
		if !gu.out() && gu.py == nil {
			n++
		}
	}
	return n
}

// lateJoinMin: with this many people or fewer still alive nobody else may join the party (the end is near).
const lateJoinMin = 10

// arrivalsClosed: the party is winding down. Every route that adds a guest in play (limousines, elevators,
// events with extra people, kids, famous powers) asks guestRoom, so one check closes them all. A group already
// on its way when the threshold is crossed is simply cancelled (addGuestAt finds no room).
func (g *Game) arrivalsClosed() bool {
	return g.mode == ModePlay && g.time > 3 && g.aliveCount() <= lateJoinMin
}

// ---------------------------------------------------------------------------
// The kill target cap. g.total (the "BAJAS x/TOTAL" of the HUD) is everybody the
// party ever had: the start AND every arrival, dead, alive or gone. The live cap
// above alone let it grow without end (60 at once, but the dead make room, so a
// slow party reached 116). goalMax() is the most the total may ever reach: 65 for
// one player (or the guest maximum of the settings + 5 if that is higher), plus
// the 20-30 extra people of two players. The start keeps goalStartRoom places
// free for the ones who arrive later; every arrival route asks guestRoom(), which
// now also says 0 when the target is full, so nothing inflates it. The special
// killer and the arsonist (not guests, but they join the total when they arrive
// or die) keep their place reserved.
// ---------------------------------------------------------------------------

// goalCap: the most people a one-player party may ever count as its kill target.
const goalCap = 65

// goalStartRoom: places of the target kept free at the start for the late arrivals.
const goalStartRoom = 6

// goalMax is the most the total of the party (kill target) may reach.
func (g *Game) goalMax() int { return maxI(goalCap, g.guestMax()+5) + g.twoExtra + g.crowdExtra }

// ---------------------------------------------------------------------------
// Big-crowd parties (user, 2026-11-10: "2 o 3 pantallas con más personajes; las demás siguen con el máximo").
// The campaign parties listed in bigCrowd (by Def.Icon, campaign.go) have the most free walkable ground and few
// obstacles (measured with bigcrowd_test.go TestBigCrowdLevels: free roam cells reachable from the start), so
// they take crowdMax (30) more people: kill target 65+30 = 95, live cap 60+30 = 90, start ~78-83 (the start
// keeps goalStartRoom + crowdExtra/5 places for the arrivals). The density stays the one of the fullest normal
// parties (~50 free cells per guest), so it looks full but nobody gets stuck. Everything that reads goalMax,
// guestCap or guestRoom (taxis, bus, train, God, famous powers, Fall Guys, kids...) follows it on its own.
//   - two players: the 20-30 extra of two players also apply, but the combined target never passes
//     crowdTwoTotal (110): the big-crowd extra shrinks to fit (65 + 20..30 + 15..25).
//   - RETO DE HOY and the stealth mode play these parties with the normal 65 (the daily modifiers, e.g. the
//     6:30 clock, were tuned for 65 and the challenge must be the same for everybody; stealth has its own count).
//   - the island and the random-mode rolls (no Icon) are never big-crowd.
// ---------------------------------------------------------------------------

// bigCrowd: the extra people of each big-crowd party (never more than crowdMax).
var bigCrowd = map[string]int{"funfair": 30, "festival": 30, "snow": 30}

const (
	crowdMax      = 30  // the most a big-crowd party adds to the target and to the live cap
	crowdTwoTotal = 110 // two players on a big-crowd party: the kill target never passes this
)

// crowdBonusFor: how many more people this party takes (0 for every normal party).
func (g *Game) crowdBonusFor(def *LevelDef) int {
	if def == nil || g.stealth || g.dc != nil {
		return 0
	}
	return clampI(bigCrowd[def.Icon], 0, crowdMax)
}

// crowdExtraFit: the big-crowd extra once the two players extra (g.twoExtra) is known.
func (g *Game) crowdExtraFit(n int) int {
	if n <= 0 {
		return 0
	}
	if g.twoExtra > 0 {
		n = minI(n, maxI(0, crowdTwoTotal-goalCap-g.twoExtra))
	}
	return n
}

// startRoom: places of the target kept free at the start (more in a big-crowd party: more arrivals).
func (g *Game) startRoom() int { return goalStartRoom + g.crowdExtra/5 }

// goalReserve: places kept for the special killer(s) and the arsonist, who join the total later.
func (g *Game) goalReserve() int {
	n := 0
	slots := []*Slasher{g.slasher}
	if g.wantTwoPlayers() {
		slots = append(slots, g.slot2.s)
	}
	for _, s := range slots {
		if s == nil || s.state == slArrive {
			n++
		}
	}
	for _, gu := range g.guests {
		if gu.py != nil && gu.dead {
			return n // the arsonist already died: he is in the total
		}
	}
	return n + 1
}

// goalLeft: how many more guests may still join the kill target of this party. Before the level
// has its total (g.total is set to 0 while it loads) the start routines are not limited here.
func (g *Game) goalLeft() int {
	if g.total <= 0 {
		return 1 << 20
	}
	if r := g.goalMax() - g.total - g.goalReserve(); r > 0 {
		return r
	}
	return 0
}

// trimStartTotal: the start never takes more than goalMax-goalStartRoom of the target; plain guests
// (no name, no group, no performer, no VIP invitation) are dropped first, newest first.
func (g *Game) trimStartTotal() {
	limit := g.goalMax() - g.startRoom()
	for len(g.guests)+len(g.guards)-g.vipStaffN() > limit {
		cut := -1
		for i := len(g.guests) - 1; i >= 0; i-- {
			if gu := g.guests[i]; !gu.keep && gu.perf == "" && gu.name == "" && !gu.vipCard && gu.py == nil && !gu.out() {
				cut = i
				break
			}
		}
		if cut < 0 {
			return
		}
		g.guests = append(g.guests[:cut], g.guests[cut+1:]...)
	}
}

// guestRoom is how many more guests fit right now (at once, and in the kill target).
func (g *Game) guestRoom() int {
	if g.arrivalsClosed() {
		return 0
	}
	if r := minI(g.guestCap()-g.guestLive(), g.goalLeft()); r > 0 {
		return r
	}
	return 0
}

// enforceGuestCap removes the newest excess guests. Reaching it means some route did not check
// the room (capBreaches goes up so the tests notice); players should never see it act.
func (g *Game) enforceGuestCap() {
	for g.guestLive() > g.guestCap() {
		cut := -1
		for i := len(g.guests) - 1; i >= 0; i-- {
			if gu := g.guests[i]; !gu.out() && gu.perf == "" && gu.py == nil {
				cut = i
				break
			}
		}
		if cut < 0 {
			return
		}
		g.guests[cut].gone = true
		capBreaches++
	}
}

// ---------------------------------------------------------------------------
// The animal cap. A level has between animalMin and animalMax animals alive on
// the map and never more than bullMax bulls (a bullring or any other level).
// animalMin is the design target of the generators (not forced: a level that has
// none gets none); animalMax and bullMax are strict and enforced in ONE place,
// capAnimals(), at the end of spawnAnimals(). Everything that is a creature in
// g.animals counts (pets, farm animals, critters, riders, snake, Wally's dog).
// The T-Rex, the dinosaurs and monsters of the island level and the guests'
// costumes are not in g.animals and do not count.
// ---------------------------------------------------------------------------

const (
	animalMin = 2
	animalMax = 8
	bullMax   = 1
)

// capAnimals trims g.animals to animalMax and bullMax. The ones the level itself defines come first
// in the list, so the scattered extras (critters, riders, snake, dog) are the first to go.
func (g *Game) capAnimals() {
	bulls := 0
	out := g.animals[:0]
	for _, a := range g.animals {
		if a.kind == animBull {
			if bulls >= bullMax {
				continue
			}
			bulls++
		}
		out = append(out, a)
	}
	if len(out) > animalMax {
		out = out[:animalMax]
	}
	g.animals = out
}
