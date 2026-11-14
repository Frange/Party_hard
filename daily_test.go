package main

import (
	"encoding/json"
	"flag"
	"strings"
	"testing"
)

// dailyTestClock sets the date of the tests and puts everything back.
func dailyTestClock(t *testing.T, date string) {
	t.Helper()
	prev, prevFrozen := today, progressFrozen
	progressFrozen = true // nothing here may write progress.json
	today = func() string { return date }
	t.Cleanup(func() { today, progressFrozen = prev, prevFrozen })
}

func TestDailyPickDeterministicAndUnlocked(t *testing.T) {
	auditGame(t) // campaignN and the levels exist once a Game does
	prev := dailyVIPRoom
	dailyVIPRoom = func(int) bool { return true }
	t.Cleanup(func() { dailyVIPRoom = prev })
	seen := map[int]bool{}
	for d := 1; d <= 28; d++ {
		date := "2026-11-" + itoa(d/10) + itoa(d%10)
		for _, unlocked := range []int{1, 3, 8, campaignN} {
			a := dailyPick(date, unlocked, nil)
			b := dailyPick(date, unlocked, nil)
			if a.level != b.level || a.vip != b.vip || len(a.mods) != len(b.mods) {
				t.Fatalf("%s: not deterministic: %+v %+v", date, a, b)
			}
			for i := range a.mods {
				if a.mods[i] != b.mods[i] {
					t.Fatalf("%s: mods differ", date)
				}
			}
			if a.level < 0 || a.level >= unlocked {
				t.Fatalf("%s unlocked=%d: level %d is not unlocked", date, unlocked, a.level)
			}
			if len(a.mods) < 2 || len(a.mods) > 3 {
				t.Fatalf("%s: %d modifiers", date, len(a.mods))
			}
			dup := map[dailyMod]bool{}
			for _, m := range a.mods {
				if dup[m] || m < 0 || m >= dmN {
					t.Fatalf("%s: bad modifiers %v", date, a.mods)
				}
				dup[m] = true
			}
			if a.vip == "" || vipByID(a.vip) == nil {
				t.Fatalf("%s: no VIP (%q)", date, a.vip)
			}
		}
		seen[dailyPick(date, campaignN, nil).level] = true
	}
	if len(seen) < 6 {
		t.Errorf("the party barely changes from day to day: %d different in 28 days", len(seen))
	}
}

// The VIP is one the album does not have yet, when there is one.
func TestDailyVIPPrefersMissing(t *testing.T) {
	auditGame(t)
	prev := dailyVIPRoom
	dailyVIPRoom = func(int) bool { return true }
	t.Cleanup(func() { dailyVIPRoom = prev })
	album := map[string]int{}
	var missing string
	for i, d := range vipRegistry() {
		if i == 0 {
			missing = d.id
			continue
		}
		album["vip:"+d.id] = albumKilled
	}
	for d := 10; d < 30; d++ {
		if got := dailyPick("2026-12-"+itoa(d), campaignN, album); got.vip != missing {
			t.Fatalf("picked %q while only %q is missing", got.vip, missing)
		}
	}
}

// The party of the challenge always has a VIP zone (no 1/5, no pity), in real levels, 1 and 2 players.
func TestDailyAlwaysHasVIPZone(t *testing.T) {
	g := auditGame(t)
	dailyTestClock(t, "2026-11-08")
	prevTwo, prevDbg, prevVips, prevOff := twoPlayerMode, dbgTwoPlayers, vipMisses, vipOff
	t.Cleanup(func() {
		twoPlayerMode, dbgTwoPlayers, vipMisses, vipOff = prevTwo, prevDbg, prevVips, prevOff
		g.dc, g.dcArm, g.challenge = nil, nil, false
	})
	vipOff = false
	for _, two := range []bool{false, true} {
		twoPlayerMode, dbgTwoPlayers = two, two
		for _, date := range []string{"2026-11-08", "2026-11-09", "2026-11-10", "2026-11-11"} {
			today = func() string { return date }
			g.progress.Unlocked = campaignN
			g.dcKey = ""
			d := g.dailyToday()
			g.dcArm = &d
			g.loadLevel(d.level)
			if g.dc == nil || !g.challenge {
				t.Fatalf("%s: the challenge is not in force", date)
			}
			if g.lvl == nil || g.lvl.vip == nil {
				t.Fatalf("%s level %d two=%v: no VIP zone", date, d.level, two)
			}
		}
	}
	g.loadLevel(0)
	if g.dc != nil || g.challenge {
		t.Fatal("a normal load must leave the challenge")
	}
}

func TestDailyLockAndUnlock(t *testing.T) {
	g := auditGame(t)
	dailyTestClock(t, "2026-11-08")
	prev := g.progress
	t.Cleanup(func() { g.progress = prev; g.dc, g.dcArm, g.challenge = nil, nil, false; g.mode = ModeMenu })
	g.progress = Progress{Unlocked: 1}
	if g.dailyState() != dailyOpen {
		t.Fatal("a fresh save must be open")
	}
	// lose inside the challenge: locked today, open tomorrow; the normal game is not affected
	auditPlay(g, false)
	g.dc, g.challenge = &dailyChallenge{date: today(), mods: []dailyMod{dmNoMap}}, true
	g.lose("prueba")
	if g.progress.ChallengeLock != "2026-11-08" || g.dailyState() != dailyLost {
		t.Fatalf("not locked after losing: %+v", g.progress)
	}
	if g.startDaily(false, -1) {
		t.Fatal("a locked challenge started")
	}
	today = func() string { return "2026-11-09" }
	if g.dailyState() != dailyOpen {
		t.Fatal("still locked the next day")
	}
	// a normal party (no challenge) that is lost does not lock anything
	g.progress = Progress{Unlocked: 1}
	auditPlay(g, false)
	g.lose("prueba")
	if g.progress.ChallengeLock != "" {
		t.Fatal("losing a normal party locked the challenge")
	}
	// completed: done today, not again until tomorrow
	auditPlay(g, false)
	g.dc, g.challenge = &dailyChallenge{date: today(), mods: []dailyMod{dmNoMap}}, true
	g.dailyWin()
	if g.dailyState() != dailyDone || g.progress.Stats["daily"] != 1 {
		t.Fatalf("not done: %+v", g.progress)
	}
	if g.startDaily(false, -1) {
		t.Fatal("a completed challenge started again")
	}
	today = func() string { return "2026-11-10" }
	if g.dailyState() != dailyOpen {
		t.Fatal("still done the next day")
	}
}

// In 2 players the challenge is lost only when BOTH lose (die() calls lose() only then).
func TestDailyTwoPlayersLocksOnlyWhenBothLose(t *testing.T) {
	g := auditGame(t)
	dailyTestClock(t, "2026-11-08")
	prev := g.progress
	t.Cleanup(func() { g.progress = prev; g.dc, g.challenge = nil, false; g.mode = ModeMenu })
	g.progress = Progress{Unlocked: 1}
	auditPlay(g, true)
	g.dc, g.challenge = &dailyChallenge{date: today()}, true
	g.die("uno")
	if g.progress.ChallengeLock != "" || g.mode != ModePlay {
		t.Fatalf("locked with one player still alive (mode %v)", g.mode)
	}
	g.asPlayer(1, func() { g.die("dos") })
	if g.progress.ChallengeLock != "2026-11-08" {
		t.Fatalf("not locked when both lost: %+v", g.progress)
	}
}

func TestDailyTimeLimitLoses(t *testing.T) {
	g := auditGame(t)
	dailyTestClock(t, "2026-11-08")
	prev := g.progress
	t.Cleanup(func() { g.progress = prev; g.dc, g.challenge = nil, false; g.mode = ModeMenu })
	g.progress = Progress{Unlocked: 1}
	auditPlay(g, false)
	g.dc, g.challenge = &dailyChallenge{date: today(), mods: []dailyMod{dmTime}}, true
	g.time = dailyTimeLimit - 1
	if g.dailyTick() || g.mode != ModePlay {
		t.Fatal("ended too soon")
	}
	g.time = dailyTimeLimit + 1
	if !g.dailyTick() || g.mode != ModeLose || g.progress.ChallengeLock == "" {
		t.Fatalf("the clock did not end the challenge (mode %v)", g.mode)
	}
}

// The score keeps x3 kills / x5 completion and adds the daily bonus only inside the challenge.
func TestDailyScoreBonus(t *testing.T) {
	g := auditGame(t)
	dailyTestClock(t, "2026-11-08")
	t.Cleanup(func() { g.dc, g.challenge = nil, false })
	auditPlay(g, false)
	g.challenge, g.endTime = true, 100
	g.points = 1000
	plain := g.previewFinish()
	g.dc = &dailyChallenge{date: today()}
	daily := g.previewFinish()
	if daily.total != plain.total+dailyBonus {
		t.Fatalf("total %d, want %d + %d", daily.total, plain.total, dailyBonus)
	}
	if plain.total != (1000+plain.complete+plain.unseen+plain.speed)*challengeFinishMult {
		t.Fatalf("x5 broken: %+v", plain)
	}
}

// An old progress.json without the new fields loads fine; the new ones are omitted when empty.
func TestDailyProgressBackwardCompatible(t *testing.T) {
	var p Progress
	if err := json.Unmarshal([]byte(`{"unlocked":5,"stats":{"played":3},"wallet":10}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.Unlocked != 5 || p.ChallengeLock != "" || p.ChallengeDone != "" {
		t.Fatalf("%+v", p)
	}
	b, _ := json.Marshal(p)
	if s := string(b); strings.Contains(s, "challengeLock") || strings.Contains(s, "challengeDone") {
		t.Fatalf("empty fields are written: %s", s)
	}
	p.ChallengeLock = "2026-11-08"
	b, _ = json.Marshal(p)
	var q Progress
	json.Unmarshal(b, &q)
	if q.ChallengeLock != "2026-11-08" {
		t.Fatal("the lock is not saved")
	}
}

// -challenge is a debug flag (it freezes the progress) and the shop total is ~125k.
func TestDailyFlagIsDebug(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Bool("challenge", false, "")
	if err := fs.Parse([]string{"-challenge"}); err != nil {
		t.Fatal(err)
	}
	if !debugFlagsActive(fs) {
		t.Fatal("-challenge must count as a debug flag")
	}
}

func TestShopPricesScaled(t *testing.T) {
	trophyFrozen(t)
	n, total := shopTotal()
	// the short shop (shopkeep.go): 41 clothes + 60 accessories + 12 perks + 12 animals + 32 weapons, ~515k points
	// (about 50 parties, shopprice.go); the characters, the killers, the police types and the uniforms are goals, not articles
	if n != 158 || total < 400000 || total > 650000 {
		t.Fatalf("shop: %d items, %d points (want 158 items, 400000-650000)", n, total)
	}
	for _, list := range shopCatalog {
		prev := 0
		for _, it := range list {
			if !shopBuyable(it) {
				continue
			}
			if it.price < 5 || it.price%5 != 0 {
				t.Errorf("%s costs %d", it.id, it.price)
			}
			if it.price < prev {
				t.Errorf("%s breaks the order", it.id)
			}
			prev = it.price
		}
	}
	for _, p := range []int{5, 100, 250, 1100, 9000} {
		if shopScalePrice(p) < 5 || shopScalePrice(p+1) < shopScalePrice(p) {
			t.Errorf("shopScalePrice not monotonic at %d", p)
		}
	}
}

func TestDailyEnterSpendsAttempt(t *testing.T) {
	g := auditGame(t)
	dailyTestClock(t, "2026-11-08")
	prev, prevFrozen := g.progress, progressFrozen
	t.Cleanup(func() {
		g.progress, progressFrozen = prev, prevFrozen
		g.dc, g.dcArm, g.challenge = nil, nil, false
		g.mode = ModeMenu
	})
	// old save without the field works and is open
	var old Progress
	if err := json.Unmarshal([]byte(`{"unlocked":1}`), &old); err != nil {
		t.Fatal(err)
	}
	g.progress = old
	progressFrozen = true // never write the real save
	// debug/frozen: entering marks nothing
	if !g.startDaily(false, -1) || g.progress.ChallengeLock != "" {
		t.Fatalf("frozen entry must not mark: %+v", g.progress)
	}
	g.progress = old
	if !g.startDaily(true, -1) || g.progress.ChallengeLock != "" {
		t.Fatal("forced entry must not mark")
	}
	// normal entry: marked at once
	g.progress = old
	progressFrozen = false
	noAssetsPrev := noAssets
	noAssets = true // saveProgress becomes a no-op
	t.Cleanup(func() { noAssets = noAssetsPrev })
	if !g.startDaily(false, -1) {
		t.Fatal("did not start")
	}
	if g.progress.ChallengeLock != "2026-11-08" || g.dailyState() != dailyLost {
		t.Fatalf("entry did not spend the attempt: %+v", g.progress)
	}
	// leaving to the menu does not free it
	g.loadLevel(0)
	g.mode = ModeMenu
	if g.dailyState() != dailyLost || g.startDaily(false, -1) {
		t.Fatal("leaving freed the attempt")
	}
	if l, _, open := g.dailyButtonLines(); open || l == "" {
		t.Fatal("button must be closed")
	}
	today = func() string { return "2026-11-09" }
	if g.dailyState() != dailyOpen {
		t.Fatal("not available the next day")
	}
}
