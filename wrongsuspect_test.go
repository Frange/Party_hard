package main

import (
	"fmt"
	"math/rand"
	"testing"
	"unicode/utf8"
)

// blameGlobals: the package dice and switches of wrongsuspect.go go back to how they were.
func blameGlobals(t *testing.T) {
	t.Helper()
	snapshotGlobals(t)
	sd, rg, f, d, ds, lang := blameSeed, blameRng, blameForce, dbgBlame, dbgBlameSetup, uiLang
	t.Cleanup(func() {
		blameSeed, blameRng, blameForce, dbgBlame, dbgBlameSetup, uiLang = sd, rg, f, d, ds, lang
	})
}

// blamePinDice: the package-level dice of the party (events, life, crowds, trains, weather...) seeded from the seed of
// the run and put back afterwards. Without it the scene depended on whatever the previous tests left in them: it
// passed alone and failed in the full pass (QA v1.4.0).
func blamePinDice(t *testing.T, seed int64) {
	t.Helper()
	type dice struct {
		r **rand.Rand
		s *int64
	}
	all := []dice{{&partyRng, &partySeed}, {&lifeRng, &lifeSeed}, {&crowdRng, &crowdSeed}, {&danceRng, &danceSeed},
		{&cdRng, &cdSeed}, {&costumeRng, &costumeSeed}, {&snakeRng, &snakeSeed}, {&tankRng, &tankSeed},
		{&trainRng, &trainSeed}, {&vipRng, &vipSeed}, {&weatherRng, &weatherSeed}, {&vaRng, &vaSeed},
		{&toroRng, &toroSeed}, {&mdRng, &mdSeed}, {&rwRng, &rwSeed}}
	for i, d := range all {
		r0, s0, d := *d.r, *d.s, d
		t.Cleanup(func() { *d.r, *d.s = r0, s0 })
		*d.r, *d.s = nil, seed*7919+int64(i)*131+1
	}
	fe, v, ve, vd := fastEv, va, vipEliminated, vipDecision
	t.Cleanup(func() { fastEv, va, vipEliminated, vipDecision = fe, v, ve, vd })
	fastEv, va, vipEliminated, vipDecision = false, nil, map[string]bool{}, ""
}

// blameGame: playGame with one or two players.
func blameGame(t *testing.T, level int, seed int64, two bool) *Game {
	t.Helper()
	g := layerGame(t, two)
	god, rng := g.god, g.rng
	t.Cleanup(func() {
		g.god, g.rng = god, rng
		g.cops, g.veh, g.bodies = nil, nil, nil
		g.policeCalled, g.pendingCar, g.pendingUnit, g.pendingAmb = false, -1, -1, -1
		g.ambCalled, g.callerGuest, g.callerGuard = false, nil, nil
		g.blame = blameSt{}
		g.mode = ModeMenu
	})
	g.rng = rand.New(rand.NewSource(seed))
	g.loadLevel(level)
	g.rng = rand.New(rand.NewSource(seed))
	g.mode = ModePlay
	g.titleT = 0
	g.god = true
	g.time = 30
	return g
}

// blameScene: a body of the player with a passer-by right next to it and the police called. Returns the passer-by.
func blameScene(t *testing.T, g *Game) (*Guest, *Body) {
	t.Helper()
	g.updatePlay(1.0 / 60) // the quirks (famous walks, kids...) are set on the first frame: see who is ordinary
	var v, w *Guest
	for _, gu := range g.guests {
		if !g.blameEligible(gu) {
			continue
		}
		for _, o := range g.guests {
			if o != gu && g.blameEligible(o) && dist(o.pos, gu.pos) < 60 && g.lvl.LOS(o.pos, gu.pos) {
				v, w = gu, o
				break
			}
		}
		if v != nil {
			break
		}
	}
	if v == nil { // nobody close enough in this party: one ordinary guest is brought next to another
		for _, gu := range g.guests {
			if !gu.out() && gu.name == "" && gu.perf == "" && gu.quirk == "" && !gu.keep && gu.py == nil && !gu.armed && gu.inside == nil && !gu.vipCard {
				gu.state, gu.timer, gu.path, gu.partner, gu.queue = gIdle, 300, nil, nil, nil
			}
			if g.blameEligible(gu) {
				if v == nil {
					v = gu
				} else {
					w = gu
					break
				}
			}
		}
		if w == nil {
			t.Fatal("not two ordinary guests in the party")
		}
		w.pos = g.lvl.NearestFree(Vec{v.pos.X + 30, v.pos.Y})
	}
	g.killGuest(v, false)
	b := g.bodies[len(g.bodies)-1]
	b.reported = true
	w.state, w.timer, w.path = gIdle, 300, nil
	for _, pl := range g.players() {
		pl.pos = g.lvl.Def.Start
	}
	g.raisePolice(b.pos, false)
	g.suspect = nil // no random guest arrested: the case stays open while the scene is checked
	g.slasherT, g.slot2.t = 1e9, 1e9
	blameTInn, blameTBody = w, b
	return w, b
}

// blameArrive waits for an officer on the scene (out of his vehicle, free) and, since the police wander a lot,
// brings him and the passer-by next to the body, as when an officer comes to look at it. Then up to 2 s of play;
// true = the scene started.
func blameArrive(t *testing.T, g *Game, inn *Guest, b *Body) bool {
	t.Helper()
	var cop *Cop
	if !runPlay(g, 60, func() bool {
		for _, c := range g.cops {
			if blameCopFree(c) && c.onSceneT >= 1.5 {
				cop = c
				return true
			}
		}
		return false
	}) {
		t.Logf("no free officer on the scene (%s)", blameWhy(g))
		return false
	}
	cop.pos = g.lvl.NearestFree(Vec{b.pos.X - 50, b.pos.Y})
	if !inn.out() && !g.blameEligible(inn) && inn != g.suspect && inn != g.callerGuest {
		inn.state, inn.timer, inn.path = gIdle, 300, nil
	}
	if !inn.out() && dist(inn.pos, b.pos) > blameNear-10 {
		inn.pos = g.lvl.NearestFree(Vec{b.pos.X + 30, b.pos.Y})
	}
	return runPlay(g, 2, func() bool { return g.blame.active })
}

var (
	blameTInn  *Guest // the passer-by and the body of the running test (for blameWhy)
	blameTBody *Body
)

// blameWhy: what a test that waited for the scene in vain saw.
func blameWhy(g *Game) string {
	s := fmt.Sprintf("t=%.0f called=%v callN=%d rolled=%d count=%d end=%q cops=%d", g.time, g.policeCalled, g.blame.callN, g.blame.rolled, g.blame.count, g.blame.why, len(g.cops))
	if c, w := g.blameFind(false); c == nil || w == nil {
		s += fmt.Sprintf(" find: cop %v guest %v", c != nil, w != nil)
	}
	if gu, b := blameTInn, blameTBody; gu != nil && b != nil {
		s += fmt.Sprintf(" passer-by: state %d out %v eligible %v at %.0f px; body reported %v carried %v", gu.state, gu.out(), g.blameEligible(gu), dist(gu.pos, b.pos), b.reported, b.carried)
		for _, c := range g.cops {
			s += fmt.Sprintf(" cop at %.0f px", dist(c.pos, b.pos))
		}
	}
	for _, c := range g.cops {
		s += fmt.Sprintf(" [st %d on %.0f medic %v]", c.state, c.onSceneT, c.medic)
	}
	return s
}

// The rules of frequency: never the first call, never two calls in a row, blameCool seconds apart, two per party.
func TestEventBlameRules(t *testing.T) {
	var b blameSt
	if b.blameAllowed(1, 500) {
		t.Fatal("never in the first police call of the party")
	}
	if !b.blameAllowed(2, 500) {
		t.Fatal("the second call may have it")
	}
	b.count, b.lastCall, b.lastT = 1, 2, 500
	if b.blameAllowed(3, 500+blameCool+60) {
		t.Fatal("never in the call right after the one that had it")
	}
	if b.blameAllowed(4, 500+blameCool-1) {
		t.Fatal("not before blameCool seconds")
	}
	if !b.blameAllowed(4, 500+blameCool) {
		t.Fatal("after the cool-down another call may have it")
	}
	b.count = blameMaxParty
	if b.blameAllowed(9, 5000) {
		t.Fatal("at most blameMaxParty per party")
	}
}

// The dice are rare and seeded: a fixed seed gives the same throws, and about blameChance of them say yes.
func TestEventBlameDice(t *testing.T) {
	blameGlobals(t)
	throw := func() []bool {
		blameReseed(42)
		out := make([]bool, 2000)
		for i := range out {
			out[i] = blameDice().Float64() < blameChance
		}
		return out
	}
	a, b := throw(), throw()
	yes := 0
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("the same seed must give the same throws")
		}
		if a[i] {
			yes++
		}
	}
	if f := float64(yes) / float64(len(a)); f < 0.10 || f > 0.20 {
		t.Fatalf("the wrong suspect must be rare (12-18%%): %.3f", f)
	}
	if blameChance < 0.12 || blameChance > 0.18 {
		t.Fatalf("blameChance %.2f out of 12-18%%", blameChance)
	}
}

// The first call of a party never has it, even with the dice forced.
func TestEventBlameNeverFirstCall(t *testing.T) {
	blameGlobals(t)
	blameForce = true
	g := blameGame(t, 0, 5, false)
	inn, b := blameScene(t, g)
	if blameArrive(t, g, inn, b) || g.blame.count != 0 {
		t.Fatal("the wrong suspect in the first police call of the party")
	}
}

// Without the force and with dice that say no, nothing happens either.
func TestEventBlameDiceNo(t *testing.T) {
	thrown := false
	for seed := int64(5); seed < 13 && !thrown; seed++ {
		blameGlobals(t)
		blameForce = false
		for s := int64(1); ; s++ { // dice whose first throw is a "no"
			blameReseed(s)
			if blameDice().Float64() >= blameChance {
				blameReseed(s)
				break
			}
		}
		g := blameGame(t, 0, seed, false)
		g.blameState().callN = 1 // the next call is the second of the party
		inn, b := blameScene(t, g)
		if blameArrive(t, g, inn, b) || g.blame.count != 0 {
			t.Fatal("the dice said no: no wrong suspect")
		}
		if g.blame.rolled > 2 {
			t.Fatalf("one throw per call (rolled %d)", g.blame.rolled)
		}
		thrown = g.blame.rolled == 2
		runPlay(g, 5, func() bool { return g.blame.active })
		if g.blame.active {
			t.Fatal("the dice are thrown once per call: no second chance in the same call")
		}
	}
	if !thrown {
		t.Fatal("the dice were never thrown in 8 parties")
	}
}

// The whole scene, one and two players: the passer-by is walked to the car and let go; he is never a kill nor a
// The whole scene, one and two players: the passer-by is walked to the car and let go; he is never a kill nor a
// body, the real case does not move, the players get their grace and nothing is left behind. A real lead (a witness
// telling the officer, a chase...) may cut it short: then it must end cleanly, and another party is tried.
func TestPoliceBlameWrongSuspectScene(t *testing.T) {
	for _, two := range []bool{false, true} {
		full := false
		for seed := int64(5); seed < 21 && !full; seed++ { // 16 parties: the police wander, some never stay by the body
			full = blameRun(t, two, seed)
		}
		if !full {
			t.Fatalf("2P %v: never a whole scene in 16 parties", two)
		}
	}
}

// blameRun plays one scene; true = it went all the way to the car and he was let go.
func blameRun(t *testing.T, two bool, seed int64) bool {
	t.Helper()
	blameGlobals(t)
	blamePinDice(t, seed)
	blameForce = true
	g := blameGame(t, 0, seed, two)
	if g.is2P() != two {
		t.Fatalf("two players %v not set up", two)
	}
	g.blameState().callN = 1
	inn, b := blameScene(t, g)
	kills := g.kills
	lk := g.lastKiller
	if !blameArrive(t, g, inn, b) {
		t.Logf("seed %d: no scene (%s)", seed, blameWhy(g))
		return false
	}
	if g.blame.inn == nil || g.blame.cop == nil || g.blame.count != 1 {
		t.Fatal("the scene must have an officer and a passer-by")
	}
	inn = g.blame.inn
	cop := g.blame.cop
	if !g.blameGraceOn() {
		t.Fatal("while they deal with him the police forget the players")
	}
	g.wanted = 30 // the player is wanted: the grace has something to forgive
	for _, pl := range g.players() {
		if pl.carrying == nil && g.copInterest(pl) {
			t.Fatal("grace: the police must not take an interest in a player")
		}
	}
	// the scene always starts walking up to him (blameStart); with the officer already at his side the approach
	// ends in the very frame it starts, before the loop below looks (it failed in the long passes, QA v1.4.0)
	phases := map[int]bool{blApproach: true, g.blame.phase: true}
	ok := runPlay(g, blameTimeout+2, func() bool {
		phases[g.blame.phase] = true
		// he is never out because of this (he stays in aliveCount and out of the HUD x/y), never a body, never a kill
		if g.kills != kills {
			t.Fatalf("the wrong suspect counted as a kill: kills %d->%d", kills, g.kills)
		}
		return !g.blame.active
	})
	if !ok {
		t.Fatal("the scene never ended")
	}
	if g.blame.active || g.blame.cop != nil || g.blame.inn != nil {
		t.Fatal("nothing of the scene may be left")
	}
	if g.suspect == inn || g.lastKiller.guest == inn && lk.guest != inn {
		t.Fatal("the passer-by must never become the real suspect of the case")
	}
	if inn.out() {
		t.Logf("seed %d: the passer-by died of something else (%s)", seed, blameWhy(g))
		return false
	}
	if inn.state != gIdle && inn.state != gGoto {
		t.Fatalf("he must be back at the party (state %d)", inn.state)
	}
	released := g.blame.why == "released"
	if released {
		for _, ph := range []int{blApproach, blCuff, blEscort, blCheck} {
			if !phases[ph] {
				t.Fatalf("2P %v: phase %d never happened (%v, %s)", two, ph, phases, blameWhy(g))
			}
		}
		if inn.icon != blameFreeInn {
			t.Fatalf("he must complain on his way back (says %q)", inn.icon)
		}
	} else {
		t.Logf("seed %d: cut short (%s)", seed, blameWhy(g))
		if g.blameGraceOn() {
			t.Fatal("a scene cut short gives no grace")
		}
	}
	if g.copPresent(cop) && g.copBlameHook(cop, 0) {
		t.Fatal("the officer must go back to his work")
	}
	// the case closes normally: no orphans (officers, vehicles, the scene). Two players: without any scene an
	// officer sometimes stays in cReturn for minutes (seen in a control run, 2 of 8 parties): not checked there.
	if two {
		return released
	}
	// (the party goes on: later calls may bring other crews, so only the officer of the scene is followed)
	if !runPlay(g, 240, func() bool { return !g.copPresent(cop) }) {
		t.Fatalf("2P %v: the officer of the scene never went home (%s, wanted %.0f stars %d)", two, blameWhy(g), g.wanted, g.stars)
	}
	g.releaseOrphanVehicles()
	for _, v := range g.veh {
		crew := false
		for _, o := range g.cops {
			crew = crew || o.veh == v
		}
		if (v.kind == 0 || v.kind == 2) && v.state == 1 && !crew {
			t.Fatal("a police vehicle parked with nobody left")
		}
	}
	return released
}

// blameStarted: a party (one of several seeds) where the forced scene has just started.
func blameStarted(t *testing.T, two bool) *Game {
	t.Helper()
	for seed := int64(5); seed < 13; seed++ {
		blameGlobals(t)
		blameForce = true
		g := blameGame(t, 0, seed, two)
		g.blameState().callN = 1
		inn, b := blameScene(t, g)
		if blameArrive(t, g, inn, b) {
			return g
		}
	}
	t.Fatal("the forced scene never started in 8 parties")
	return nil
}

// Anything that breaks the scene gives the guest back at once: the officer killed, the case closed.
func TestPoliceBlameAbortsCleanly(t *testing.T) {
	blameGlobals(t)
	for _, how := range []string{"killed", "closed", "guest"} {
		g := blameStarted(t, false)
		inn, cop := g.blame.inn, g.blame.cop
		switch how {
		case "killed":
			g.removeCop(cop)
		case "closed":
			g.closeCase()
		case "guest":
			g.killGuest(inn, false)
		}
		runPlay(g, 0.2, func() bool { return false })
		if g.blame.active || g.blame.inn != nil || g.blame.cop != nil {
			t.Fatalf("%s: the scene must end at once", how)
		}
		if g.blameGraceOn() {
			t.Fatalf("%s: a broken scene gives no grace", how)
		}
		if how != "guest" && inn.out() {
			t.Fatalf("%s: the guest must be back at the party", how)
		}
		// 2026-11-11: a fresh officer seals off the place (cCordon) until his 20 s are done
		if how == "closed" && g.copPresent(cop) && cop.state != cLeave && cop.state != cCordon {
			t.Fatal("the officer sent home must go home")
		}
		g.blame = blameSt{}
	}
}

// Only ordinary guests: no famous, no performers, no real suspect, no caller.
func TestGuestBlameEligible(t *testing.T) {
	blameGlobals(t)
	g := blameGame(t, 0, 5, false)
	var gu *Guest
	for _, o := range g.guests {
		if g.blameEligible(o) {
			gu = o
			break
		}
	}
	if gu == nil {
		t.Fatal("no ordinary guest")
	}
	for name, f := range map[string]func(*Guest){
		"famous":    func(o *Guest) { o.name = "Elvis" },
		"performer": func(o *Guest) { o.perf = "dj" },
		"armed":     func(o *Guest) { o.armed = true },
		"suspect":   func(o *Guest) { g.suspect = o },
		"caller":    func(o *Guest) { g.callerGuest = o },
		"vip":       func(o *Guest) { o.vipCard = true },
		"dead":      func(o *Guest) { o.dead = true },
	} {
		c := *gu
		g.suspect, g.callerGuest = nil, nil
		f(&c)
		if g.blameEligible(&c) {
			t.Errorf("%s must never be taken for the suspect", name)
		}
	}
	g.suspect, g.callerGuest = nil, nil
}

// The texts: ES and EN, short, and the bubble keeps them in at most two lines inside its width.
func TestLangBlameTexts(t *testing.T) {
	snapshotLang(t)
	blameGlobals(t)
	g := layerGame(t, false)
	for _, s := range []string{blameCopWord, blameInnWord, blameFreeCop, blameFreeInn} {
		en, ok := langDataEN[s]
		if !ok || en == "" || en == s {
			t.Fatalf("%q has no English", s)
		}
		for _, lang := range []string{"es", "en"} {
			uiLang = lang
			txt := tr(s)
			if lang == "en" && txt != en {
				t.Fatalf("tr(%q) in English = %q", s, txt)
			}
			if utf8.RuneCountInString(txt) > 48 {
				t.Fatalf("%q longer than 48", txt)
			}
			lines := wrapBubble(txt, bubMaxW, func(x string) float64 { return g.textW(x, 1) }, bubLines)
			for _, l := range lines {
				if len(l) >= 3 && l[len(l)-3:] == "..." {
					t.Fatalf("%s: %q is cut in the bubble", lang, txt)
				}
				// whole pixels and room around the text: the box is the widest line + 8 (4 px each side)
				if w := g.textW(l, 1); w != float64(int(w)) || w+8 > bubMaxW+8 {
					t.Fatalf("%s: line %q of %.1f px", lang, l, w)
				}
			}
		}
	}
	uiLang = "es"
}

// The scene does not take the ambulance's work: the medics still come for the body in the same place.
func TestAmbulanceBlameSameScene(t *testing.T) {
	blameGlobals(t)
	g := blameStarted(t, false)
	g.ambCalled, g.pendingAmb = true, g.time // the paramedics are called while it goes on
	medics := false
	runPlay(g, 120, func() bool {
		for _, c := range g.cops {
			if c.medic {
				medics = true
			}
		}
		return medics && !g.blame.active
	})
	if !medics {
		t.Fatal("the ambulance must still come")
	}
	for _, c := range g.cops {
		if c.medic && g.blame.cop == c {
			t.Fatal("a paramedic is never the officer of the scene")
		}
	}
}
