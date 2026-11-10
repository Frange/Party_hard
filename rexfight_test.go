package main

import (
	"math/rand"
	"testing"
)

// the guns of the military and the SWAT weaken the T-Rex but never kill it; the police and the
// FBI do nothing; only the Power Rangers who won their roll take it to 0
func TestRexDamageRules(t *testing.T) {
	for _, u := range []int{unitMilitary, unitSWAT} {
		hp := rexMaxHP
		for i := 0; i < 1000; i++ {
			hp = rexDamage(hp, u, true)
		}
		if hp != rexFloorHP || hp <= 0 {
			t.Fatalf("unit %d left it at %.0f, want the floor %.0f", u, hp, rexFloorHP)
		}
	}
	for _, u := range []int{unitPolice, unitFBI} {
		if rexDamage(rexMaxHP, u, true) != rexMaxHP {
			t.Fatalf("unit %d must not hurt it", u)
		}
	}
	hp := rexMaxHP
	for i := 0; i < 1000; i++ {
		hp = rexDamage(hp, unitRangers, false)
	}
	if hp != rexFloorHP {
		t.Fatal("the Power Rangers who lost their roll cannot kill it")
	}
	hits := 0
	for hp = rexMaxHP; hp > 0; hits++ {
		hp = rexDamage(hp, unitRangers, true)
	}
	if hits < 10 {
		t.Fatalf("it has a lot of life: %d blows were enough", hits)
	}
	if !(rexSlow(rexFloorHP) < 1 && rexSlow(rexFloorHP) >= rexSlowMin && rexSlow(rexMaxHP) == 1) {
		t.Fatal("weakened means a little slower")
	}
}

// the roll is about one in three, and both outcomes happen
func TestRexRangersRoll(t *testing.T) {
	old := dbgRexKill
	dbgRexKill = false
	defer func() { dbgRexKill = old }()
	wins := 0
	for s := int64(1); s <= 400; s++ {
		g := &Game{rng: rand.New(rand.NewSource(s))}
		if g.rexRangersRoll(&JRex{}) {
			wins++
		}
	}
	if wins < 100 || wins > 180 {
		t.Fatalf("the Power Rangers won %d of 400 fights, want ~35%%", wins)
	}
	g := &Game{rng: rand.New(rand.NewSource(1))}
	r := &JRex{}
	first := g.rexRangersRoll(r)
	for i := 0; i < 20; i++ {
		if g.rexRangersRoll(r) != first {
			t.Fatal("the fight is decided once per T-Rex")
		}
	}
}

func islandGame(t *testing.T, seed int64) *Game {
	t.Helper()
	g := playGame(t, specialJurassic, seed)
	if g.rex == nil {
		t.Fatal("no T-Rex on the island")
	}
	return g
}

// a real fight: five Power Rangers next to the T-Rex. Sometimes they bring it down, sometimes it eats some.
func TestRexRangersFight(t *testing.T) {
	killed, eaten, fights := 0, 0, 0
	for s := int64(1); s <= 16 && (killed == 0 || eaten == 0); s++ {
		g := islandGame(t, s)
		r := g.rex
		r.state, r.timer, r.roarIn = jrWander, 0, 1e9
		r.rolled, r.rangersWin = true, s%3 == 0 // both outcomes on purpose: the roll itself is tested above
		var crew []*Cop
		for k := 0; k < 5; k++ {
			c := &Cop{pos: g.lvl.NearestFree(Vec{r.pos.X - 60 + float64(k)*25, r.pos.Y + 30}), kind: unitRangers, state: cSweep, alt: k, maxSearch: 1e9, onSceneT: 30}
			g.cops = append(g.cops, c)
			crew = append(crew, c)
		}
		fights++
		runPlay(g, 60, func() bool { return g.rex == nil })
		if g.rex == nil {
			killed++
			if !r.rangersWin {
				t.Fatal("the T-Rex died although the Power Rangers lost their roll")
			}
		} else if g.rex.hp <= 0 {
			t.Fatal("a T-Rex with no life still standing")
		}
		for _, c := range crew {
			if !containsCop(g.cops, c) {
				eaten++
				break
			}
		}
	}
	if killed == 0 || eaten == 0 {
		t.Fatalf("in %d fights: T-Rex killed %d times, rangers eaten in %d; both must happen", fights, killed, eaten)
	}
}

// the military never kill it, even in a long fight
func TestRexMilitaryOnlyWeaken(t *testing.T) {
	g := islandGame(t, 3)
	r := g.rex
	r.state, r.timer, r.roarIn = jrWander, 0, 1e9
	for k := 0; k < 3; k++ {
		g.cops = append(g.cops, &Cop{pos: g.lvl.NearestFree(Vec{r.pos.X - 120 + float64(k)*30, r.pos.Y + 40}), kind: unitMilitary, state: cSweep, maxSearch: 1e9, onSceneT: 30})
	}
	min := rexMaxHP
	runPlay(g, 60, func() bool {
		if g.rex == nil {
			return true
		}
		if g.rex.hp < min {
			min = g.rex.hp
		}
		return false
	})
	if g.rex == nil || min < rexFloorHP {
		t.Fatalf("the military killed it or took it below the floor (min %.0f)", min)
	}
}

// what the T-Rex kills is reported: the police come (the local police first) and the player gets no stars
func TestRexKillsBringThePolice(t *testing.T) {
	called := 0
	for s := int64(1); s <= 4; s++ {
		g := islandGame(t, s)
		r := g.rex
		r.state, r.timer, r.roarIn = jrWander, 0, 1e9
		v := victimInView(g)
		if v == nil {
			continue
		}
		r.pos = Vec{v.pos.X + 40, v.pos.Y + 18}
		r.face = -1
		g.jrEats(rexPrey{id: v, at: func() Vec { return v.pos }, cs: v.cs, alive: func() bool { return !v.out() }, kill: func() { g.killGuest(v, false) }})
		if len(g.bodies) == 0 || !g.bodies[len(g.bodies)-1].byRex {
			t.Fatal("the T-Rex's victim must be marked")
		}
		r.state, r.pos = jrWander, Vec{790, 256}
		g.p.pos = g.lvl.Def.Start
		if runPlay(g, 45, func() bool { return g.policeCalled }) {
			called++
			if g.stars != 0 {
				t.Fatalf("a T-Rex death gave the player %d stars", g.stars)
			}
			if g.respKind != unitPolice {
				t.Fatal("the local police come first")
			}
		} else {
			b := g.bodies[len(g.bodies)-1]
			t.Logf("seed %d: no call: seers %d reported %v carried %v caller %v st %v ph %v", s, len(b.seers), b.reported, b.carried, g.callerGuest != nil, func() int {
				if g.callerGuest == nil {
					return -1
				}
				return int(g.callerGuest.state)
			}(), g.phoneSpot(b.pos) != nil)
		}
	}
	if called < 3 {
		t.Fatalf("only %d of 4 T-Rex deaths brought the police", called)
	}
}

// on the island an officer who has done his search and bagged two bodies still stays for the rest
func TestIslandPoliceCoverEveryBody(t *testing.T) {
	g := islandGame(t, 1)
	g.bodies = nil
	for i := 0; i < 3; i++ {
		g.bodies = append(g.bodies, &Body{pos: g.lvl.RandomRoam(g.rng), reported: true, byRex: true})
	}
	c := &Cop{pos: g.lvl.Def.Start, kind: unitPolice, searchT: 99, maxSearch: 30, bagN: 2, onSceneT: 99}
	g.cops = []*Cop{c}
	g.ambCalled = true
	g.copNext(c)
	if c.state != cCollect || c.target == nil {
		t.Fatalf("with bodies uncovered the officer must not leave (state %d)", c.state)
	}
	c.target.claimed, c.target = nil, nil
	g.bodies = nil // all picked up
	g.copNext(c)
	if c.state != cLeave {
		t.Fatalf("everything picked up: he may go (state %d)", c.state)
	}
}
