package main

import (
	"math/rand"
	"testing"
)

func zombieCount(g *Game) int {
	n := 0
	for _, gu := range g.guests {
		if gu.state == gZombie && !gu.out() {
			n++
		}
	}
	return n
}

// The zombie event today: one guest turns, the bite spreads, one stab puts a zombie down and the kill count stays right.
func TestZombieEventSpreadsAndDiesOneParty(t *testing.T) {
	g := peLoad(t, 2, false)
	g.p.hidden = true // the player stays out of it
	g.events = nil
	g.forceEv = evZombie + 1
	g.startEvent()
	if zombieCount(g) != 1 {
		t.Fatalf("a zombie event must turn exactly one guest, got %d", zombieCount(g))
	}
	alive0 := g.aliveCount()
	for i := 0; i < 60*90 && zombieCount(g) < 3; i++ {
		peStep(g, 1.0/60, true)
	}
	if zombieCount(g) < 2 {
		t.Skipf("the zombie did not reach anybody in 90 s on this level (%d)", zombieCount(g))
	}
	if g.aliveCount() > alive0 {
		t.Errorf("alive count grew: %d -> %d", alive0, g.aliveCount())
	}
	n, aliveB := zombieCount(g), g.aliveCount()
	for _, gu := range g.guests {
		if gu.state == gZombie && !gu.out() {
			g.killGuest(gu, true)
			break
		}
	}
	if zombieCount(g) != n-1 {
		t.Errorf("a stab should put one zombie down: %d -> %d", n, zombieCount(g))
	}
	if g.aliveCount() != aliveB-1 {
		t.Errorf("the stabbed zombie must count as one kill: alive %d -> %d", aliveB, g.aliveCount())
	}
	for i := 0; i < 600; i++ {
		peStep(g, 1.0/60, true)
	}
}

// A zombie is no witness: one that has the player in front of him when he stabs another zombie stays a
// zombie (witness() used to alarm him into gAlarm, which "cured" him: one stab put two zombies down, the
// failure of the test above whenever the party left a zombie facing the player).
func TestZombieDoesNotWitnessTheStab(t *testing.T) {
	g := peLoad(t, 2, false)
	g.events = nil
	var zs []*Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && gu.py == nil && len(zs) < 2 {
			zs = append(zs, gu)
		}
	}
	if len(zs) < 2 {
		t.Fatal("not enough guests")
	}
	g.makeZombie(zs[0])
	g.makeZombie(zs[1])
	for _, gu := range g.guests {
		if gu != zs[0] && gu != zs[1] {
			gu.pos = Vec{gu.pos.X + 4000, gu.pos.Y} // nobody else around
		}
	}
	p := g.p
	zs[0].pos = p.pos
	zs[1].pos, zs[1].dir = Vec{p.pos.X, p.pos.Y + 24}, DirUp // looking straight at the player
	if !g.sees(zs[1].pos, zs[1].dir, p.pos, guestRange, guestDot) {
		t.Fatal("setup: the second zombie must have the player in sight")
	}
	n := zombieCount(g)
	g.killGuest(zs[0], true)
	if zs[1].state != gZombie || zombieCount(g) != n-1 {
		t.Errorf("one stab, one zombie down: %d -> %d (the one watching is in state %d)", n, zombieCount(g), zs[1].state)
	}
	if zs[1].sawKiller || g.callerGuest == zs[1] {
		t.Error("a zombie must not report the kill")
	}
}

// Two players: the zombie hunts and kills the NEAREST player, not always the first one.
func TestZombieBitesTheSecondPlayer(t *testing.T) {
	g := peLoad(t, 2, true)
	if !g.is2P() {
		t.Skip("no two players here")
	}
	g.events = nil
	var z *Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" {
			z = gu
			break
		}
	}
	g.makeZombie(z)
	for _, gu := range g.guests {
		if gu != z {
			gu.pos = Vec{gu.pos.X + 4000, gu.pos.Y} // nobody else around
		}
	}
	p1, p2 := g.first(), g.p2
	p1.pos = Vec{z.pos.X + 3000, z.pos.Y}
	p2.pos = Vec{z.pos.X + 4, z.pos.Y}
	p1.hidden, p2.hidden = false, false
	for i := 0; i < 120 && !p2.dead; i++ {
		g.updateZombie(z, 1.0/60)
		g.time += 1.0 / 60
	}
	if !p2.dead {
		t.Fatal("the zombie ignored the second player standing on him")
	}
	if p1.dead {
		t.Error("the zombie killed the far first player")
	}
}

// zombiePartyShare: the share of simulated parties of 5-8 minutes (the same rules as the game: rareGap, rareKind, the
// limit of big events) in which at least one zombie event comes up.
func zombiePartyShare(seed int64, n int) float64 {
	r := rand.New(rand.NewSource(seed))
	with := 0
	for i := 0; i < n; i++ {
		L := 300 + r.Float64()*180
		big, sat, god, z := 0, false, false, false
		for t := 45 + r.Float64()*45; t < L; t += rareGap(r, t) {
			k := rareKind(r, t, big, sat, god, func(int) bool { return false })
			sat, god = sat || k == evSatan, god || k == evGod
			if rareBigKinds[k] {
				big++
			}
			z = z || k == evZombie
		}
		if z {
			with++
		}
	}
	return float64(with) / float64(n)
}

// Zombies come in about one party of six (14-20 %), with several seeds, not more and not less.
func TestZombieOnePartyInSix(t *testing.T) {
	for _, seed := range []int64{1, 7, 42, 2024} {
		f := zombiePartyShare(seed, 30000)
		t.Logf("seed %d: parties of 5-8 min with a zombie event: %.1f %%", seed, f*100)
		if f < 0.14 || f > 0.20 {
			t.Errorf("seed %d: %.1f %% of the parties have zombies, want 14-20 %% (about 1 in 6)", seed, f*100)
		}
	}
}
