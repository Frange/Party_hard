package main

import "testing"

// Rule (2026-11-10): ANY body a guest or a guard sees is reported, whoever killed it ("a corpse nobody can account
// for is a police matter"). Before, guestScan and the guards' scan skipped every body that was not the player's, so
// a fight, a car, an explosion... left corpses next to people walking past and nobody called. Only the player's own
// kills raise the stars or make him a suspect.

type deathKind struct {
	name string
	kill func(g *Game, v *Guest)
}

var otherDeaths = []deathKind{
	{"pelea", func(g *Game, v *Guest) { peleaKill(g, v) }}, // also the "tiroteo" between guests
	{"atropello/tanque/animal/asesino", func(g *Game, v *Guest) { g.killGuest(v, false) }},
	{"veneno ajeno", func(g *Game, v *Guest) {
		g.credit, g.killCause = true, kcPoison
		g.killGuest(v, false)
		g.credit = false
	}},
	{"explosion/trampa", func(g *Game, v *Guest) { g.killCause = kcBlast; g.killGuest(v, false) }},
	{"electrocutado", func(g *Game, v *Guest) { g.killGuestElec(v, false) }},
	{"sin ruido (noHeat)", func(g *Game, v *Guest) {
		g.noHeat = true
		g.killGuest(v, false)
		g.noHeat = false
	}},
}

func TestAnyBodyIsReported(t *testing.T) {
	levels := []int{0, 4, 11}
	for _, d := range otherDeaths {
		called := 0
		for _, lv := range levels {
			g := playGame(t, lv, int64(lv+3))
			v := victimInView(g)
			if v == nil {
				t.Fatalf("level %d: no victim", lv)
			}
			d.kill(g, v)
			b := g.bodies[len(g.bodies)-1]
			if b.byPlayer {
				t.Fatalf("%s: the body must not be the player's", d.name)
			}
			g.p.pos = g.lvl.Def.Start
			if runPlay(g, 45, func() bool { return g.policeCalled }) {
				called++
			} else {
				t.Logf("%s, level %d: nobody called (seers %d)", d.name, lv, len(b.seers))
			}
			// the player did nothing: no stars, no cuffs, nobody after him
			runPlay(g, 5, func() bool { return false })
			if g.stars != 0 || g.playerIsSuspect() || g.wanted > 0 {
				t.Fatalf("%s: stars %d suspect %v wanted %.0f for a death that is not the player's", d.name, g.stars, g.playerIsSuspect(), g.wanted)
			}
		}
		if called < len(levels)-1 {
			t.Fatalf("%s: only %d of %d bodies brought the police", d.name, called, len(levels))
		}
	}
}

// the debug option "nobody calls the police" still holds
func TestAnyBodyNoPoliceDebug(t *testing.T) {
	g := playGame(t, 0, 5)
	old := g.dbg.noPolice
	g.dbg.noPolice = true
	t.Cleanup(func() { g.dbg.noPolice = old })
	peleaKill(g, victimInView(g))
	g.p.pos = g.lvl.Def.Start
	if runPlay(g, 45, func() bool { return g.policeCalled }) {
		t.Fatal("with the police switched off nobody must come")
	}
}

// the player's kills still raise the stars; somebody else's do not
func TestOnlyPlayerKillsGiveStars(t *testing.T) {
	g := playGame(t, 0, 6)
	for i := 0; i < 3; i++ {
		peleaKill(g, victimInView(g))
	}
	for _, b := range g.bodies {
		b.reported = true
	}
	g.raisePolice(g.bodies[0].pos, false)
	g.updateStars(0.1)
	if g.stars != 0 || g.playerHeat() {
		t.Fatalf("other people's deaths gave %d stars", g.stars)
	}
	g.killGuest(victimInView(g), true)
	g.bodies[len(g.bodies)-1].reported = true
	g.updateStars(0.1)
	if g.stars < 1 {
		t.Fatal("a located kill of the player gives stars")
	}
}

// the police groups count every death of the party
func TestPoliceGroupsCountEveryDeath(t *testing.T) {
	withGroups(t, 2)
	g := playGame(t, 0, 7)
	g.kills = 0
	// policeGroupFor: the k-th death is in the FBI's stretch when (k-1)*2 >= total, so the first one is
	// ceil(total/2)+1 (total/2+1 fell one short whenever the party had an odd number of guests)
	need := (g.total+1)/2 + 1
	t.Logf("party of %d: the FBI from death %d", g.total, need)
	n := 0
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && n < need-1 {
			peleaKill(g, gu)
			n++
		}
	}
	if n != need-1 || g.groupUnitNow() != unitPolice {
		t.Fatalf("%d of %d guests dead: still the local police's stretch", g.partyDeaths(), g.total)
	}
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" {
			peleaKill(g, gu)
			break
		}
	}
	if g.groupUnitNow() != unitFBI {
		t.Fatalf("%d of %d guests dead (none by the player): the FBI's stretch", g.partyDeaths(), g.total)
	}
}
