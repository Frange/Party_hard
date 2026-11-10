package main

import (
	"math/rand"
	"testing"
)

func twoGame(two bool) *Game {
	g := testGame()
	g.rng = rand.New(rand.NewSource(11))
	g.snd = &Sounds{muted: true}
	def := level1()
	g.lvl = buildLevel(def)
	g.as = &Assets{}
	sp := playerSpec()
	g.as.playerCS = newCharSet(&sp)
	g.p = &Player{pos: def.Start, dir: DirUp, cs: g.as.playerCS, stamina: 6}
	old := twoPlayerMode
	twoPlayerMode = two
	defer func() { twoPlayerMode = old }()
	g.setupPlayers(def)
	g.mode = ModePlay
	return g
}

func TestOnePlayerIsUnchanged(t *testing.T) {
	g := twoGame(false)
	if g.is2P() || g.p2 != nil || len(g.players()) != 1 || g.otherPlayer(g.p) != nil {
		t.Fatal("one player: there must be no second player")
	}
	g.pickStartTarget()
	if g.twoExtra != 0 || g.guestCap() != g.guestMax() {
		t.Fatalf("one player: the cap must be the plain maximum, extra=%d", g.twoExtra)
	}
	called := 0
	g.eachSlasher(func() { called++ })
	if called != 1 {
		t.Fatal("one player: there is only one killer slot")
	}
	defer g.focus(Vec{10, 10})()
	if g.p != g.players()[0] {
		t.Fatal("focus must not change the player with one player")
	}
}

func TestTwoPlayersStartApart(t *testing.T) {
	for _, def := range testLevels(t)[:10] {
		g := twoGame(true)
		g.lvl = buildLevel(def)
		g.p = &Player{pos: def.Start, dir: DirUp, cs: g.as.playerCS, stamina: 6}
		old := twoPlayerMode
		twoPlayerMode = true
		g.setupPlayers(def)
		twoPlayerMode = old
		if !g.is2P() {
			t.Fatal("no second player")
		}
		d := dist(g.p.pos, g.p2.pos)
		if d < 20 || d > 80 {
			t.Errorf("%s: the players are %.0f px apart", def.Title, d)
		}
		for _, pl := range g.players() {
			if g.lvl.SolidP(pl.pos.X, pl.pos.Y) {
				t.Errorf("%s: a player starts inside something solid", def.Title)
			}
			if dist(pl.pos, def.Start) > 90 {
				t.Errorf("%s: a player starts far from the entrance", def.Title)
			}
		}
		if g.p2.idx != 1 || g.p.idx != 0 {
			t.Error("player indexes")
		}
	}
}

func TestAsPlayerSwitchesInputAndBack(t *testing.T) {
	g := twoGame(true)
	p1, p2 := g.p, g.p2
	g.asPlayer(1, func() {
		if g.p != p2 || inputP != 1 {
			t.Error("as player 2: g.p and the input must be player 2")
		}
	})
	if g.p != p1 || inputP != 0 {
		t.Fatal("asPlayer must put everything back")
	}
	if g.first() != p1 {
		t.Error("first()")
	}
	// nested: an NPC focus inside player 2's turn also comes back
	g.asPlayer(1, func() {
		restore := g.focus(p1.pos)
		if g.p != p1 {
			t.Error("focus should pick the nearest player")
		}
		restore()
		if g.p != p2 {
			t.Error("focus must restore player 2")
		}
	})
}

func TestPunchBetweenPlayersNeverKills(t *testing.T) {
	g := twoGame(true)
	a, b := g.p, g.p2
	a.pos, b.pos = Vec{300, 300}, Vec{312, 300}
	a.dir = DirRight
	g.stab()
	if b.dead || b.knockT < hitDownT-0.01 || b.immT < hitDownT+hitSafeT-0.01 {
		t.Fatalf("the punch must knock him down for %.0fs and protect him %.0fs more: dead=%v knock=%.2f imm=%.2f", hitDownT, hitSafeT, b.dead, b.knockT, b.immT)
	}
	if g.kills != 0 || g.mode != ModePlay {
		t.Fatal("a punch between players is not a kill")
	}
	// a second punch right away does nothing
	a.cooldown = 0
	g.stab()
	if b.knockT > hitDownT+0.01 || b.immT > hitDownT+hitSafeT+0.01 {
		t.Fatal("the protection must not be extended or restarted by another punch")
	}
	// time passes: down 3 s, then still protected until 8 s, then he can be hit again
	step := func(sec float64) {
		for i := 0; i < int(sec*60); i++ {
			g.updateTwoPlayers(1.0 / 60)
			g.asPlayer(1, func() { g.updatePlayer(1.0 / 60) })
		}
	}
	step(3.2)
	if b.knockT > 0 {
		t.Error("he should be on his feet after 3 s")
	}
	a.cooldown = 0
	g.stab()
	if b.knockT > 0 {
		t.Error("he cannot be punched again during the 5 s after getting up")
	}
	step(4.9)
	if b.immT > 0 {
		t.Errorf("the protection should be over after 8 s, left %.2f", b.immT)
	}
	a.cooldown = 0
	g.stab()
	if b.knockT <= 0 {
		t.Error("after the 5 s he can be punched again")
	}
	if a.knockT > 0 || a.immT > 0 {
		t.Error("the one who punches gets no punishment")
	}
}

func TestOnePlayerDiesTheOtherGoesOn(t *testing.T) {
	g := twoGame(true)
	g.asPlayer(0, func() { g.die("test") })
	if !g.p.dead || g.mode != ModePlay || g.livePlayers() != 1 {
		t.Fatal("when one dies the party goes on")
	}
	g.asPlayer(1, func() { g.die("test") })
	if g.mode != ModeLose {
		t.Fatal("the party is lost when both are dead")
	}
	// the dead one cannot move or attack
	g2 := twoGame(true)
	g2.asPlayer(1, func() { g2.die("x") })
	before := g2.p2.pos
	g2.asPlayer(1, func() { g2.updatePlayer(1) })
	if g2.p2.pos != before {
		t.Fatal("a dead player must stay where he fell")
	}
}

func TestTwoPlayersMoreGuests(t *testing.T) {
	old := twoPlayerMode
	twoPlayerMode = true
	defer func() { twoPlayerMode = old }()
	g := twoGame(true)
	for i := 0; i < 20; i++ {
		g.pickStartTarget()
		if g.twoExtra < 20 || g.twoExtra > 30 {
			t.Fatalf("extra guests %d, want 20..30", g.twoExtra)
		}
		if g.startTarget < 45+20 || g.startTarget > 60+30 {
			t.Fatalf("start target %d", g.startTarget)
		}
		if g.guestCap() != g.guestMax()+g.twoExtra {
			t.Fatal("the cap must grow with the extra guests")
		}
	}
}

func TestFocusIsTheNearestPlayer(t *testing.T) {
	g := twoGame(true)
	g.p.pos, g.p2.pos = Vec{100, 300}, Vec{700, 300}
	if g.nearestPlayer(Vec{150, 300}) != g.p || g.nearestPlayer(Vec{650, 300}) != g.p2 {
		t.Fatal("nearest player")
	}
	g.p2.dead = true
	if g.nearestPlayer(Vec{690, 300}) != g.p {
		t.Fatal("a dead player does not attract the NPCs")
	}
}

func TestSecondKillerSlot(t *testing.T) {
	g := twoGame(true)
	cs, _, _ := g.pickKiller()
	mk := func(x float64) *Slasher {
		return &Slasher{cs: cs, state: slHunt, hp: 3, pos: Vec{x, 300}, dir: DirDown}
	}
	g.slasher = mk(200)
	g.slot2.s = mk(500)
	if len(g.allSlashers()) != 2 {
		t.Fatal("two players: two killers can be at the party")
	}
	n := 0
	g.eachSlasher(func() {
		n++
		if g.slasher == nil {
			t.Error("both slots must be seen as g.slasher in turn")
		}
	})
	if n != 2 || g.slasher == nil || g.slot2.s == nil || g.slasher.pos.X != 200 || g.slot2.s.pos.X != 500 {
		t.Fatal("the slots must be put back in place")
	}
	// a stab at the second killer hits him, not the first
	g.p.pos = Vec{490, 300}
	hp1, hp2 := g.slasher.hp, g.slot2.s.hp
	if !g.hitAnySlasher(g.p, 28) {
		t.Fatal("the second killer is within reach")
	}
	if g.slasher.hp != hp1 || g.slot2.s.hp != hp2-1 {
		t.Fatalf("wrong killer was hit: %d %d", g.slasher.hp, g.slot2.s.hp)
	}
}

// a second killer only exists with two players: with one the old limit stays
func TestOneKillerSlotWithOnePlayer(t *testing.T) {
	g := twoGame(false)
	g.slot2.s = &Slasher{state: slHunt}
	n := 0
	g.eachSlasher(func() { n++ })
	if n != 1 {
		t.Fatal("one player: the second slot must never run")
	}
}

// two bots walking at random for a while: neither of the players stays stuck for more than ~5 s
func TestTwoBotsDoNotStayStuck(t *testing.T) {
	defs := testLevels(t)
	if len(defs) > 8 {
		defs = defs[:8]
	}
	for _, def := range defs {
		g := twoGame(true)
		g.lvl = buildLevel(def)
		g.p = &Player{pos: def.Start, dir: DirUp, cs: g.as.playerCS, stamina: 6}
		old := twoPlayerMode
		twoPlayerMode = true
		g.setupPlayers(def)
		twoPlayerMode = old
		g.walkBot = true
		stuckRecs = map[any]*stuckRec{}
		last := []Vec{g.p.pos, g.p2.pos}
		still := []float64{0, 0}
		const dt = 1.0 / 60
		for f := 0; f < 60*90; f++ {
			g.time += dt
			g.updateTwoPlayers(dt)
			g.updatePlayer(dt)
			g.asPlayer(1, func() { g.updatePlayer(dt) })
			g.watchAllStuck(dt)
			for i, pl := range g.players() {
				if dist(pl.pos, last[i]) > 0.6 || pl.knockT > 0 || !pl.moving {
					last[i], still[i] = pl.pos, 0
					continue
				}
				if still[i] += dt; still[i] > stuckLimit+0.6 {
					t.Fatalf("%s: player %d stuck for %.1f s at %.0f,%.0f", def.Title, i+1, still[i], pl.pos.X, pl.pos.Y)
				}
			}
		}
	}
}
