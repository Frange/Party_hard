package main

import "math/rand"

// ---------------------------------------------------------------------------
// Famous characters (Sonic, Shrek, Guybrush, Homer...) appear at most once per
// party. Every route that adds a guest (start of the level, arrivals, events,
// powers, groups) may pick whoever it likes; uniqueFamous() then swaps any
// second copy for an ordinary guest, so no route can repeat one.
// ---------------------------------------------------------------------------

var famousNameSet map[string]bool

// famousNames: the presets of famousPresets that are real characters: not the
// generic archetypes and not the kids (they have their own rules).
func famousNames() map[string]bool {
	if famousNameSet != nil {
		return famousNameSet
	}
	skip := map[string]bool{}
	for _, a := range archetypePresets {
		skip[a.Name] = true
	}
	for _, k := range kidSlugs {
		skip[k] = true
	}
	famousNameSet = map[string]bool{}
	probe := rand.New(rand.NewSource(1))
	for _, f := range famousPresets {
		if n := f(probe).Name; n != "" && !skip[n] {
			famousNameSet[n] = true
		}
	}
	return famousNameSet
}

// isFamous: is this character name one of the one-per-party famous ones?
func (g *Game) isFamous(name string) bool {
	if name == "" || isKidSlug(name) {
		return false
	}
	if famousNames()[name] {
		return true
	}
	if g.store != nil {
		for _, ca := range g.store.Chars["special_people"] {
			if ca.Meta.Name == name {
				return true
			}
		}
	}
	return false
}

// plainGuestCS is an ordinary, unnamed guest: from the people folder or the generator.
func (g *Game) plainGuestCS() *CharSet {
	if g.store != nil {
		var people []*CharAsset
		for _, ca := range g.store.Chars["people"] {
			if !ca.Meta.ShopOnly {
				people = append(people, ca)
			}
		}
		if len(people) > 0 {
			return g.rigFor(people[g.rng.Intn(len(people))])
		}
	}
	spec := randomGuestSpec(g.rng)
	return newCharSet(&spec)
}

// uniqueFamous swaps the second and later copies of a famous character for ordinary guests.
// Performers (the band) own their name first; then guests in list order.
func (g *Game) uniqueFamous() {
	if g.famousOwned == nil {
		g.famousOwned = map[string]*Guest{}
	}
	name := func(gu *Guest) string {
		if gu.cs == nil || gu.cs.Spec == nil || !g.isFamous(gu.cs.Spec.Name) {
			return ""
		}
		return gu.cs.Spec.Name
	}
	for _, gu := range g.guests {
		if gu.perf != "" {
			if n := name(gu); n != "" && g.famousOwned[n] == nil {
				g.famousOwned[n] = gu
			}
		}
	}
	for _, gu := range g.guests {
		n := name(gu)
		if n == "" || gu.perf != "" {
			continue
		}
		if own := g.famousOwned[n]; own == nil {
			g.famousOwned[n] = gu // first one in the party keeps the name
			continue
		} else if own == gu || gu.famousOK {
			continue
		}
		gu.cs, gu.name = g.plainGuestCS(), ""
		gu.quirk, gu.quirkInit = "", false
	}
}

// ---- how many people a party starts with ---------------------------------------

// pickStartTarget draws the number of guests this party starts with, between the
// minimum and the maximum of the settings (40 to 60 by default).
func (g *Game) pickStartTarget() {
	lo, hi := g.maxStart(), g.guestMax()
	g.startTarget = lo
	if hi > lo {
		g.startTarget = lo + g.rng.Intn(hi-lo+1)
	}
}

// fillGuests tops the party up with ordinary guests until it has the target number.
func (g *Game) fillGuests(start Vec) {
	for len(g.guests) < g.startTarget {
		var pos Vec
		for tries := 0; tries < 40; tries++ {
			pos = g.lvl.RandomRoam(g.rng)
			if dist(pos, start) > 140 {
				break
			}
		}
		gu := &Guest{pos: pos, dir: Dir(g.rng.Intn(4)), cs: g.plainGuestCS(), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 3, role: len(g.guests) % 2}
		g.guests = append(g.guests, gu)
	}
}
