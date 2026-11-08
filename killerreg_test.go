package main

import (
	"math/rand"
	"testing"
)

// killerGame: a headless party with the killer clock reset (every global restored afterwards).
func killerGame(t *testing.T, two bool) *Game {
	t.Helper()
	g := auditGame(t)
	prevTwo, prevDbg, prevK, prevFast := twoPlayerMode, dbgTwoPlayers, dbgKiller, fastEv
	t.Cleanup(func() {
		twoPlayerMode, dbgTwoPlayers, dbgKiller, fastEv = prevTwo, prevDbg, prevK, prevFast
		g.slasher, g.slot2 = nil, slasherSlot{}
	})
	dbgKiller, fastEv = false, false
	auditPlay(g, two)
	g.snd = &Sounds{muted: true}
	g.slasher, g.slot2 = nil, slasherSlot{}
	g.slasherT, g.slot2.t = 0, 0
	g.testVictim = nil
	return g
}

// runKillerTo advances the clock to t (in 1 s steps, like the party) and returns how many killers came.
func runKillerTo(g *Game, t float64) {
	for g.time < t {
		g.time++
		g.updateSlashers(1)
		for _, s := range g.allSlashers() { // each one is caught at once: the budget, not the killer, is under test
			s.state = slGone
		}
	}
}

func TestKillerChanceAboutOneInThree(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	n, hits := 300, 0
	for i := 0; i < n; i++ {
		if p := killerPlanFor(r, false); p.planned {
			hits++
			if p.at < killerAtMin || p.at > killerAtMax {
				t.Fatalf("moment %v out of range", p.at)
			}
		}
	}
	if hits < 75 || hits > 125 {
		t.Errorf("%d of %d levels have a killer, want about a third", hits, n)
	}
	if killerChance != 1.0/3 || killerLateAfter != 210 || killerSecondAfter != 300 {
		t.Errorf("constants changed: %v %v %v", killerChance, killerLateAfter, killerSecondAfter)
	}
}

func TestKillerPlanDeterministicAndNeverInStealth(t *testing.T) {
	a, b := rand.New(rand.NewSource(99)), rand.New(rand.NewSource(99))
	for i := 0; i < 50; i++ {
		if killerPlanFor(a, false) != killerPlanFor(b, false) {
			t.Fatal("the same seed gives different plans")
		}
	}
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 500; i++ {
		if killerPlanFor(r, true).planned {
			t.Fatal("a stealth level planned a killer")
		}
	}
}

func TestKillerAllowedThresholds(t *testing.T) {
	plan := killerPlan{planned: true, at: 60}
	for _, c := range []struct {
		tm   float64
		want int
	}{{0, 0}, {59, 0}, {60, 1}, {299, 1}, {300, 2}, {900, 2}} {
		if got := plan.allowed(c.tm); got != c.want {
			t.Errorf("planned: allowed(%v)=%d want %d", c.tm, got, c.want)
		}
	}
	none := killerPlan{}
	for _, c := range []struct {
		tm   float64
		want int
	}{{0, 0}, {209, 0}, {210, 1}, {299, 1}, {300, 2}} {
		if got := none.allowed(c.tm); got != c.want {
			t.Errorf("unplanned: allowed(%v)=%d want %d", c.tm, got, c.want)
		}
	}
}

func TestKillerNeverMoreThanOneBeforeFiveMinutes(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := killerGame(t, two)
		g.kp = killerPlan{planned: true, at: 40}
		runKillerTo(g, 299)
		if g.kp.spawned != 1 {
			t.Fatalf("two=%v: %d killers before 5:00, want exactly 1", two, g.kp.spawned)
		}
		runKillerTo(g, 305)
		if g.kp.spawned != 2 {
			t.Fatalf("two=%v: %d killers after 5:00, want 2", two, g.kp.spawned)
		}
		runKillerTo(g, 900)
		if g.kp.spawned != 2 {
			t.Fatalf("two=%v: %d killers in the end, never more than 2", two, g.kp.spawned)
		}
	}
}

func TestKillerSlowPlayerGetsAnAccelerator(t *testing.T) {
	g := killerGame(t, false)
	g.kp = killerPlan{} // the level had no killer planned
	runKillerTo(g, killerLateAfter-1)
	if g.kp.spawned != 0 {
		t.Fatalf("a killer came at %v s without being planned", g.time)
	}
	runKillerTo(g, killerLateAfter+2)
	if g.kp.spawned != 1 {
		t.Fatalf("no accelerator after %v s", killerLateAfter)
	}
	runKillerTo(g, 299)
	if g.kp.spawned != 1 {
		t.Fatalf("second one too early: %d", g.kp.spawned)
	}
	runKillerTo(g, 302)
	if g.kp.spawned != 2 {
		t.Fatalf("no second killer after 5:00: %d", g.kp.spawned)
	}
}

func TestKillerNotInStealth(t *testing.T) {
	g := killerGame(t, false)
	g.stealth = true
	t.Cleanup(func() { g.stealth = false })
	g.killerLevelReset()
	if g.kp.planned {
		t.Fatal("planned in stealth")
	}
	runKillerTo(g, 600)
	if g.kp.spawned != 0 || g.slasher != nil {
		t.Fatalf("stealth got %d killers", g.kp.spawned)
	}
}

func TestKillerFlagForcesOne(t *testing.T) {
	found := false
	for _, n := range mainFlagNames(t) {
		found = found || n == "killer"
	}
	if !found {
		t.Fatal("-killer flag missing")
	}
	g := killerGame(t, false)
	dbgKiller = true
	g.killerLevelReset()
	if !g.kp.planned || g.kp.at != killerForcedAt {
		t.Fatalf("plan %+v", g.kp)
	}
	runKillerTo(g, 20)
	if g.kp.spawned != 1 {
		t.Fatalf("forced killer did not come: %d", g.kp.spawned)
	}
}

func TestKillerLevelResetNoPanicAndKeepsDice(t *testing.T) {
	g := killerGame(t, false)
	before := g.rng.Int63()
	g.rng = rand.New(rand.NewSource(5))
	want := rand.New(rand.NewSource(5)).Int63()
	for i := 0; i < 50; i++ {
		g.killerLevelReset()
	}
	if got := g.rng.Int63(); got != want {
		t.Errorf("killerLevelReset moved g.rng (%d != %d, before %d)", got, want, before)
	}
}

// ---- the clock in the HUD ----

func TestHUDClockFormat(t *testing.T) {
	for _, c := range []struct {
		t    float64
		want string
	}{{0, "0:00"}, {59.9, "0:59"}, {61, "1:01"}, {600, "10:00"}, {725.4, "12:05"}, {6249, "104:09"}, {-3, "0:00"}} {
		if got := hudClockText(c.t); got != c.want {
			t.Errorf("hudClockText(%v)=%q want %q", c.t, got, c.want)
		}
	}
}

func TestHUDClockColorByBonusThreshold(t *testing.T) {
	green, amber, grey := rgb(0x60e090), rgb(0xffc040), rgb(0x9a94b0)
	for _, c := range []struct {
		t    float64
		want [3]uint8
	}{{0, rgbArr(green)}, {179, rgbArr(green)}, {timeFastS + 0.1, rgbArr(rgb(0xffffff))}, {timeFastS + 0.3, rgbArr(amber)}, {timeFastS + 5, rgbArr(amber)}, {299, rgbArr(amber)}, {timeMidS + 0.1, rgbArr(rgb(0xffffff))}, {timeMidS + 2, rgbArr(grey)}, {900, rgbArr(grey)}} {
		if got := rgbArr(hudClockColor(c.t)); got != c.want {
			t.Errorf("colour at %v = %v want %v", c.t, got, c.want)
		}
	}
}

func rgbArr(c interface{ RGBA() (r, g, b, a uint32) }) [3]uint8 {
	r, g, b, _ := c.RGBA()
	return [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}

func TestHUDClockShownAndPausedTimeDoesNotCount(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		g.time = 161
		rec := hudRecord(g)
		found := false
		for _, tx := range rec.texts {
			found = found || tx.s == "2:41"
		}
		if !found {
			t.Errorf("two=%v: the HUD does not write 2:41", two)
		}
		g.hud.reset(g)
	}
}

// The help window pauses the game (Update stops before updatePlay): the level time stands still.
func TestHUDClockPausedByHelpDoesNotCount(t *testing.T) {
	g := helpGame(t, false)
	helpHW(t)
	g.time = 100
	g.helpOv = true
	for i := 0; i < 30; i++ {
		if !g.updateHelp() { // true = the rest of Update (updatePlay, the only place the time runs) is skipped
			t.Fatal("the open help window does not pause the frame")
		}
	}
	if g.time != 100 {
		t.Errorf("the time moved to %v with the help open", g.time)
	}
}
