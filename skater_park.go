package main

import "math"

// ---------------------------------------------------------------------------
// The skate park and the three kids on wheels (Diego, César and Sam-sam):
//   - in a party with bowls (the SKATE PARK) the three always come, just after it starts (planKids): they
//     arrive like every kid (guestRoom permitting, so the caps and the kill target are untouched; if there is
//     no room yet they try again a moment later). A kid who dies or leaves never comes back.
//   - every 8-20 s one of them (not the one who went last, unless the others are busy for long) rides to the
//     nearest free bowl, rides in, does a big trick up the wall (a handplant, a 540, a backflip...) and a few
//     more, and rolls out (skateTurnFor / kidBowl, skatebowl.go). Two may ride at once, each in his own bowl.
//     Nobody gets stuck: the trip to the bowl has a timeout.
//   - round any party, rolling about, a kid hops now and then (an ollie or a bunny hop) and sometimes shouts
//     one of his jokes (kidHopTick, drawn by kidHopZ).
// ---------------------------------------------------------------------------

const (
	skateTurnMin   = 8.0  // seconds between two visits to the bowls (the park's turn)...
	skateTurnVar   = 12.0 // ...plus up to this many
	skateTurnGrace = 12.0 // the last one may go again if the others have been busy this long after his turn was due
	kidHopDur      = 0.45 // a hop on the board
	kidHopH        = 7.0  // its height (px)
)

var skateTurn struct {
	lvl        *Level
	next       float64
	last       *Guest
	visits     int // visits started in this party (for the tests)
	lastSlug   string
	repeatSlug int  // visits that went to the same kid as the one before while another could go (tests)
	grace      bool // this turn went to the last one again: the others were busy for too long
	graceUsed  int
}

// skateTurnSync: a new party starts its own turn.
func (g *Game) skateTurnSync() {
	if skateTurn.lvl != g.lvl {
		skateTurn.lvl, skateTurn.last = g.lvl, nil
		skateTurn.next = g.time + 3 + g.rng.Float64()*5
		skateTurn.visits, skateTurn.lastSlug, skateTurn.repeatSlug, skateTurn.graceUsed = 0, "", 0, 0
	}
}

// skateTurnFor: whether it is gu's turn to go to a bowl now (the caller starts his visit).
func (g *Game) skateTurnFor(gu *Guest) bool {
	if g.lvl == nil || len(g.lvl.bowls) == 0 || !isKid(gu) {
		return false
	}
	g.skateTurnSync()
	if g.time < skateTurn.next { // (a visit may still be on: the next kid takes another bowl)
		return false
	}
	skateTurn.grace = false
	if gu == skateTurn.last { // somebody else, if somebody else is about
		for _, o := range g.guests {
			if o != gu && isKid(o) && !o.out() {
				if g.time < skateTurn.next+skateTurnGrace {
					return false
				}
				skateTurn.grace = true // the others have been busy (running, hiding...) for a while: he goes again
			}
		}
	}
	return true
}

func (g *Game) skateTurnStarted(gu *Guest) {
	g.skateTurnSync()
	skateTurn.last = gu
	skateTurn.next = g.time + skateTurnMin + g.rng.Float64()*skateTurnVar // counted from this visit's start
	skateTurn.visits++
	slug := ""
	if gu.cs != nil && gu.cs.Spec != nil {
		slug = gu.cs.Spec.Name
	}
	others := false // another of the three still in the party
	for _, o := range g.guests {
		others = others || (o != gu && isKid(o) && !o.out())
	}
	if slug == skateTurn.lastSlug && !skateTurn.grace && others {
		skateTurn.repeatSlug++
	}
	if skateTurn.grace {
		skateTurn.graceUsed++
	}
	skateTurn.lastSlug = slug
}

// skateTurnDone: a visit is over (or could not start): the next one stays 8-20 s after the last start, and
// never less than 2 s from now.
func (g *Game) skateTurnDone() {
	g.skateTurnSync()
	skateTurn.next = math.Max(skateTurn.next, g.time+2)
}

// ---- hops round the party --------------------------------------------------------------------------

type kidHopSt struct{ next, start float64 }

var (
	kidHops   = map[*Guest]*kidHopSt{}
	kidHopLvl *Level
)

// kidHopTick: now and then, rolling about, a kid hops on his board (and sometimes says something).
func (g *Game) kidHopTick(gu *Guest) {
	if kidHopLvl != g.lvl {
		kidHops, kidHopLvl = map[*Guest]*kidHopSt{}, g.lvl
	}
	st := kidHops[gu]
	if st == nil {
		st = &kidHopSt{next: g.time + 2 + g.rng.Float64()*5, start: -10}
		kidHops[gu] = st
	}
	if g.mode != ModePlay || gu.out() || !gu.moving || kidSessions[gu] != nil || g.time < st.next {
		return
	}
	st.start, st.next = g.time, g.time+3+g.rng.Float64()*6
	if gu.icon == "" && g.rng.Float64() < 0.3 {
		slug := ""
		if gu.cs != nil && gu.cs.Spec != nil {
			slug = gu.cs.Spec.Name
		}
		if g.rng.Float64() < 0.5 {
			gu.icon = skaterTrickShout[trOllie][0]
			if slug != "diego" {
				gu.icon = skaterTrickShout[trNoHander][0]
			}
		} else {
			l := skaterLines(slug)
			gu.icon = l[g.rng.Intn(len(l))][0]
		}
		gu.iconT = 1.6
	}
}

// kidHopZ: how high a kid is off the ground in a hop right now (whole pixels; 0 most of the time).
func (g *Game) kidHopZ(gu *Guest) float64 {
	st := kidHops[gu]
	if st == nil || kidHopLvl != g.lvl {
		return 0
	}
	u := (g.time - st.start) / kidHopDur
	if u <= 0 || u >= 1 {
		return 0
	}
	return math.Round(kidHopH * math.Sin(math.Pi*u))
}
