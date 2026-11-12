package main

import (
	"os"
	"testing"
)

func scoreGame(t *testing.T) *Game {
	t.Helper()
	oldF := progressFrozen
	t.Cleanup(func() { progressFrozen = oldF })
	g, _ := fireGame(t, 3, 7)
	g.perkMoney = 1
	g.mode = ModePlay
	g.progress = Progress{Unlocked: 1}
	return g
}

func liveGuest(g *Game, n int) *Guest {
	for _, gu := range g.guests {
		if !gu.out() {
			if n == 0 {
				return gu
			}
			n--
		}
	}
	return nil
}

func TestScoreKillMultiplier(t *testing.T) {
	g := scoreGame(t)
	p0 := g.points
	g.killGuest(liveGuest(g, 0), true)
	if g.points-p0 != 30 || g.run.killExtra != 0 {
		t.Fatalf("normal kill: +%d extra %d, want +30 / 0", g.points-p0, g.run.killExtra)
	}
	g.challenge = true
	g.time += 10 // a new chain
	p0 = g.points
	g.killGuest(liveGuest(g, 0), true)
	if g.points-p0 != 30*challengeKillMult || g.run.killExtra != 30*(challengeKillMult-1) {
		t.Fatalf("RETO kill: +%d extra %d, want +90 / 60", g.points-p0, g.run.killExtra)
	}
	// a credited (trap) kill is worth 50, x3 in the challenge
	g.time += 10
	p0 = g.points
	g.run.streakN = [2]int{} // only the RETO here: the RACHA (3rd clean kill) has its own tests (streak_test.go)
	g.credit = true
	g.trapKill(liveGuest(g, 0))
	g.credit = false
	if g.points-p0 != 150 {
		t.Errorf("RETO trap kill: +%d, want +150", g.points-p0)
	}
}

func TestScoreFinishBonusesWithoutChallenge(t *testing.T) {
	g := scoreGame(t)
	g.points, g.endTime, g.time = 100, 100, 100
	g.finishScore()
	f := g.run.fin
	want := 100 + bonusComplete + bonusUnseen + bonusTimeFast
	if g.points != want || f.total != want || f.finalExtra != 0 || f.challenge {
		t.Fatalf("total %d (fin %+v), want %d with no x5", g.points, f, want)
	}
	g.finishScore() // twice changes nothing
	if g.points != want {
		t.Errorf("finishScore applied twice: %d", g.points)
	}
}

func TestScoreSeenAndSlowLoseBonuses(t *testing.T) {
	g := scoreGame(t)
	g.run.seen[0] = 1
	g.points, g.time = 100, 400
	g.finishScore()
	if g.points != 100+bonusComplete {
		t.Errorf("seen + slow: %d, want %d", g.points, 100+bonusComplete)
	}
	if speedBonus(200) != bonusTimeMid || speedBonus(10) != bonusTimeFast || speedBonus(301) != 0 {
		t.Error("speedBonus tiers are wrong")
	}
}

func TestChallengeFinishX5AndNoX5WhenLost(t *testing.T) {
	g := scoreGame(t)
	g.challenge = true
	g.points, g.time = 100, 100
	g.finishScore()
	sum := 100 + bonusComplete + bonusUnseen + bonusTimeFast
	if g.points != sum*challengeFinishMult || g.run.fin.finalExtra != sum*(challengeFinishMult-1) {
		t.Fatalf("RETO total %d, want %d", g.points, sum*challengeFinishMult)
	}
	// lost: no x5 and no bonuses, the points already scored stay
	l := scoreGame(t)
	l.challenge = true
	l.points = 270
	l.lose("prueba")
	if l.run.done || l.points != 270 {
		t.Errorf("a lost party must not get bonuses or x5 (done=%v points=%d)", l.run.done, l.points)
	}
}

func TestStatsKillCauses(t *testing.T) {
	g := scoreGame(t)
	g.killGuest(liveGuest(g, 0), true) // knife
	gu := liveGuest(g, 0)
	gu.poison = 0.01
	g.updatePoison(gu, 1) // poison
	g.credit = true
	g.trapKill(liveGuest(g, 0)) // trap
	g.killCause = kcBlast
	g.killGuest(liveGuest(g, 0), false) // explosion
	g.credit = false
	g.killGuest(liveGuest(g, 0), false) // not the player's: not counted
	r := &g.run
	if r.total(kcKnife) != 1 || r.total(kcPoison) != 1 || r.total(kcTrap) != 1 || r.total(kcBlast) != 1 {
		t.Fatalf("causes knife/poison/trap/blast = %d/%d/%d/%d, want 1 each", r.total(kcKnife), r.total(kcPoison), r.total(kcTrap), r.total(kcBlast))
	}
	if r.killsOf(0)+r.killsOf(1) != 4 {
		t.Errorf("total kills %d, want 4", r.killsOf(0)+r.killsOf(1))
	}
	if g.killCause != kcNone {
		t.Error("the cause must be consumed by the kill")
	}
}

func TestStatsPerPlayerTwoPlayers(t *testing.T) {
	g := scoreGame(t)
	g.p1 = g.p
	g.p2 = &Player{idx: 1, pos: g.p.pos, cs: g.p.cs}
	g.killGuest(liveGuest(g, 0), true) // player 1
	g.asPlayer(1, func() {
		g.killGuest(liveGuest(g, 0), true)
		g.killGuest(liveGuest(g, 0), true)
	})
	if g.run.killsOf(0) != 1 || g.run.killsOf(1) != 2 {
		t.Fatalf("kills J1/J2 = %d/%d, want 1/2", g.run.killsOf(0), g.run.killsOf(1))
	}
	g.p1, g.p2 = nil, nil
}

func TestStatsCuffsCountedOncePerMark(t *testing.T) {
	g := scoreGame(t)
	g.wanted = 30
	g.trackCuffs()
	g.trackCuffs()
	g.wanted = 0
	g.trackCuffs()
	g.wanted = 30
	g.trackCuffs()
	if g.run.cuffed[0] != 2 {
		t.Errorf("cuffed %d, want 2 (two separate marks)", g.run.cuffed[0])
	}
}

func TestLevelDoneFrozenProgressAndNoPanicDraw(t *testing.T) {
	g := scoreGame(t)
	progressFrozen = true
	real := "assets/progress.json"
	before, errB := os.Stat(real)
	g.challenge = true
	g.points, g.time = 500, 120
	g.finishScore()
	g.bankPoints()
	if g.progress.Wallet != g.run.fin.total {
		t.Errorf("wallet %d, want %d", g.progress.Wallet, g.run.fin.total)
	}
	after, errA := os.Stat(real)
	if (errB == nil) != (errA == nil) || (errB == nil && !before.ModTime().Equal(after.ModTime())) {
		t.Error("the real progress.json changed with progressFrozen on")
	}
}

func TestScoreChainMultiplier(t *testing.T) {
	g := scoreGame(t)
	g.time = 10
	var got []int
	for i := 0; i < 3; i++ {
		p0 := g.points
		g.run.streakN = [2]int{} // only the chain here: the RACHA has its own tests (streak_test.go)
		g.killGuest(liveGuest(g, 0), true)
		got = append(got, g.points-p0)
		g.time += 2 // inside the window
	}
	if got[0] != 30 || got[1] != 36 || got[2] != 42 {
		t.Fatalf("chain values %v, want [30 36 42]", got)
	}
	if g.run.chainMax[0] != 3 || g.chainNow(0) != 3 {
		t.Errorf("chain max/now %d/%d, want 3/3", g.run.chainMax[0], g.chainNow(0))
	}
	g.time += chainWindow + 1 // too late: the chain restarts
	g.run.streakN = [2]int{}  // only the chain here (streak_test.go)
	p0 := g.points
	g.killGuest(liveGuest(g, 0), true)
	if g.points-p0 != 30 || g.chainNow(0) != 1 {
		t.Errorf("after the window: +%d chain %d, want +30 / 1", g.points-p0, g.chainNow(0))
	}
	// cap
	g.run.chainN[0], g.run.chainLast[0] = 50, g.time
	g.run.streakN = [2]int{} // only the chain here (streak_test.go)
	p0 = g.points
	g.killGuest(liveGuest(g, 0), true)
	if g.points-p0 != 90 {
		t.Errorf("capped chain: +%d, want +90 (x3)", g.points-p0)
	}
	// RETO multiplies the chain value too
	h := scoreGame(t)
	h.challenge = true
	h.run.chainN[0], h.run.chainLast[0] = 3, h.time
	p0 = h.points
	h.killGuest(liveGuest(h, 0), true) // chain 4: x1.6 -> 48 -> x3
	if h.points-p0 != 144 {
		t.Errorf("RETO + chain: +%d, want +144", h.points-p0)
	}
}

func TestScoreCleanKill(t *testing.T) {
	g := scoreGame(t)
	g.time = 5
	g.killGuest(liveGuest(g, 0), false) // not his
	g.credit = true
	g.trapKill(liveGuest(g, 0)) // unseen trap kill
	g.credit = false
	p0 := g.points
	g.tickClean(false)
	if g.points != p0 {
		t.Fatal("paid before the delay")
	}
	g.time += cleanDelay + 0.1
	g.tickClean(false)
	if g.points-p0 != bonusClean || g.run.cleanKills[0] != 1 {
		t.Errorf("clean: +%d kills %d, want +%d / 1", g.points-p0, g.run.cleanKills[0], bonusClean)
	}
	// seen: a witness cancels it
	g.run.cleanAt[0] = g.time + 1
	g.run.seen[0] = 0
	g.run.cleanAt[0] = 0 // what witness() does when somebody saw
	p0 = g.points
	g.time += 5
	g.tickClean(false)
	if g.points != p0 {
		t.Error("a seen kill must not be clean")
	}
	// RETO: x3, and the end of the party pays what is pending
	h := scoreGame(t)
	h.challenge = true
	h.run.cleanAt[0] = h.time + 100
	h.finishScore()
	if h.run.cleanKills[0] != 1 || h.run.fin.clean != bonusClean {
		t.Errorf("end of party: clean kills %d fin.clean %d", h.run.cleanKills[0], h.run.fin.clean)
	}
}
