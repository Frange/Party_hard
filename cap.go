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
func (g *Game) guestCap() int { return g.guestMax() + g.twoExtra } // two players: 20-30 more guests

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

// guestRoom is how many more guests fit right now.
func (g *Game) guestRoom() int {
	if r := g.guestCap() - g.guestLive(); r > 0 {
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
