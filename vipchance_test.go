package main

import (
	"strings"
	"testing"
)

// vipChanceSetup: fresh dice, empty session marks, restored afterwards.
func vipChanceSetup(t *testing.T, seed int64) {
	t.Helper()
	vipReseed(seed)
	vipEliminated = map[string]bool{}
	t.Cleanup(func() { vipReseed(0); vipDecision = ""; vipEliminated = map[string]bool{} })
}

func vipRoomyDef(t *testing.T) *LevelDef {
	t.Helper()
	d := vipLevelDef(t, "PLAZA DE TOROS")
	if nd, _ := vipPlace(d, "king"); nd == nil {
		t.Fatal("PLAZA DE TOROS has no room")
	}
	return d
}

func TestVIPChanceIsAboutOneInFive(t *testing.T) {
	vipChanceSetup(t, 12345)
	d := vipRoomyDef(t)
	n, hit := 500, 0
	for i := 0; i < n; i++ {
		if vipDecide(vipDice(), d, nil, &vipMisses, false, false) != "" {
			hit++
		}
	}
	// 1/5 plus the pity (one extra every 9 at worst): between 15% and 26%
	if hit < n*15/100 || hit > n*26/100 {
		t.Errorf("%d zones in %d parties, want about %d", hit, n, n/5)
	}
}

func TestVIPPityAfterEightWithoutZone(t *testing.T) {
	vipChanceSetup(t, 7)
	d := vipRoomyDef(t)
	run := 0
	for i := 0; i < 400; i++ {
		if vipDecide(vipDice(), d, nil, &vipMisses, false, false) != "" {
			run = 0
		} else if run++; run > vipPityAfter {
			t.Fatalf("%d parties in a row without a zone", run)
		}
	}
	m := vipPityAfter
	if id := vipDecide(vipDice(), d, nil, &m, false, false); id == "" {
		t.Error("after the pity limit the zone is certain")
	}
	if m != 0 {
		t.Error("the counter must reset after a zone")
	}
}

func TestVIPPicksOnlyMissingWhileAnyLeft(t *testing.T) {
	vipChanceSetup(t, 99)
	album := map[string]int{}
	reg := vipRegistry()
	for i, d := range reg {
		if i%2 == 0 {
			album["vip:"+d.id] = albumKilled
		} else if i%4 == 1 {
			album["vip:"+d.id] = albumHelped | albumSeen
		} // the rest: only seen (or nothing) counts as missing
	}
	album["vip:"+reg[2].id] = albumSeen
	for i := 0; i < 300; i++ {
		for _, title := range []string{"PLAZA DE TOROS", "CASINO", "XYZ"} {
			id := vipPick(vipDice(), album, title)
			if vipGot(album, id) {
				t.Fatalf("picked %s, already got", id)
			}
		}
	}
	// session kills count too
	only := reg[0].id
	album2 := map[string]int{}
	for _, d := range reg[1:] {
		album2["vip:"+d.id] = albumKilled
	}
	for i := 0; i < 50; i++ {
		if id := vipPick(vipDice(), album2, "TORRE EIFFEL"); id != only {
			t.Fatalf("only %s is missing, got %s", only, id)
		}
	}
	vipEliminated[only] = true
	seen := map[string]bool{}
	for i := 0; i < 400; i++ {
		seen[vipPick(vipDice(), album2, "TORRE EIFFEL")] = true
	}
	if len(seen) < len(reg)-1 {
		t.Errorf("all got: expected variety, saw %d", len(seen))
	}
}

func TestVIPAllGotIsRandomAndMissingVariety(t *testing.T) {
	vipChanceSetup(t, 5)
	album := map[string]int{}
	for _, d := range vipRegistry() {
		album["vip:"+d.id] = albumKilled
	}
	seen := map[string]bool{}
	for i := 0; i < 400; i++ {
		seen[vipPick(vipDice(), album, "PLAZA DE TOROS")] = true
	}
	if len(seen) != len(vipRegistry()) {
		t.Errorf("with all 11 got every VIP should come out, saw %d", len(seen))
	}
	// none got (old save without Album): variety and the affinity shows up
	seen = map[string]bool{}
	for i := 0; i < 400; i++ {
		seen[vipPick(vipDice(), nil, "PLAZA DE TOROS")] = true
	}
	if len(seen) < 8 || !seen["king"] {
		t.Errorf("old save: saw %d VIPs", len(seen))
	}
	aff := 0
	for i := 0; i < 400; i++ {
		if vipPick(vipDice(), nil, "PLAZA DE TOROS") == "king" {
			aff++
		}
	}
	if aff < 400*40/100 {
		t.Errorf("the king in the bullring only %d/400 times", aff)
	}
	// an affine VIP already got passes to another missing one
	album = map[string]int{"vip:king": albumHelped}
	for i := 0; i < 200; i++ {
		if vipPick(vipDice(), album, "PLAZA DE TOROS") == "king" {
			t.Fatal("king is got, must not come out")
		}
	}
}

func TestVIPDecisionDeterministicAndOwnDice(t *testing.T) {
	d := vipRoomyDef(t)
	run := func() string {
		vipChanceSetup(t, 4242)
		var sb strings.Builder
		for i := 0; i < 60; i++ {
			sb.WriteString(vipDecide(vipDice(), d, nil, &vipMisses, false, false) + ",")
		}
		return sb.String()
	}
	if a, b := run(), run(); a != b {
		t.Errorf("not deterministic:\n%s\n%s", a, b)
	}
	// the roll uses vipRng only: g.rng keeps its sequence
	vipChanceSetup(t, 4242)
	g := hudPlay(t, false)
	g.rng.Seed(5)
	r1 := g.rng.Int63()
	g.rng.Seed(5)
	_ = g.vipRoll(d)
	if r2 := g.rng.Int63(); r1 != r2 {
		t.Error("vipRoll moved g.rng")
	}
}

func TestVIPNeverInStealthNorForcedDemo(t *testing.T) {
	vipChanceSetup(t, 3)
	d := vipRoomyDef(t)
	m := vipPityAfter + 5
	if id := vipDecide(vipDice(), d, nil, &m, true, false); id != "" {
		t.Error("stealth must never have a zone, even with pity")
	}
	if id := vipDecide(vipDice(), d, nil, &m, true, true); id != "" {
		t.Error("stealth must never have a zone, even forced")
	}
	for i := 0; i < 20; i++ {
		if vipDecide(vipDice(), d, nil, &vipMisses, false, true) == "" {
			t.Fatal("-vipdemo must force the zone")
		}
	}
	// a level without room never gets a zone and counts as a miss
	no := vipLevelDef(t, "ESTADIO")
	if nd, _ := vipPlace(no, "king"); nd == nil {
		m = vipPityAfter
		if vipDecide(vipDice(), no, nil, &m, false, false) != "" || m != vipPityAfter+1 {
			t.Error("a level without room: no zone and the pity keeps counting")
		}
	}
}

func TestVIPLoadLevelRollsAndHintFollowsZone(t *testing.T) {
	hintRestore(t)
	vipChanceSetup(t, 2024)
	g := hudPlay(t, false)
	g.progress.Album = nil // old save
	with, without := 0, 0
	for k := 0; k < 120; k++ {
		g.loadLevel(4) // PLAZA DE TOROS
		lines := g.levelHintLines()
		has := lines[1] == hintVIP
		if has != (g.lvl.vip != nil) {
			t.Fatalf("hint says VIP=%v but the zone is %v", has, g.lvl.vip != nil)
		}
		if g.lvl.vip != nil {
			with++
			if vipGot(nil, g.lvl.vip.id) || vipByID(g.lvl.vip.id) == nil {
				t.Fatal("bad VIP id")
			}
		} else {
			without++
		}
		if vipDecision != "" {
			t.Fatal("vipDecision must be cleared after loading")
		}
	}
	if with == 0 || without == 0 {
		t.Errorf("with %d without %d: expected both", with, without)
	}
	for k := 0; k < 40; k++ {
		g.stealthNext = true
		g.loadLevel(4)
		if !g.stealth {
			t.Fatal("stealth did not start")
		}
		if g.lvl.vip != nil {
			t.Fatal("stealth level with a VIP zone")
		}
	}
}
