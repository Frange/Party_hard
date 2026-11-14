package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// The DISCOTECA and the CINE are retired (campaign_retired.go): kept in the list for the saves, never played.
func TestRetiredPartiesKeptButNotPlayed(t *testing.T) {
	_ = layerGame(t, false) // levelDefs and campaignN built
	club, cine, d70 := campaignIdx("disco"), campaignIdx("cinema"), campaignIdx("disco70")
	if club != 10 || cine != 14 || campaignLevels[club].kind != "club" || campaignLevels[cine].kind != "cinema" {
		t.Fatalf("the retired parties moved: club %d, cinema %d (their index is their save key)", club, cine)
	}
	if !campaignRetired(club) || !campaignRetired(cine) || campaignRetired(d70) {
		t.Fatal("DISCOTECA and CINE must be retired, the DISCO SETENTERA must not")
	}
	if len(campaignLevels) != 28 || campaignN != 28 || campaignShown() != 26 || len(retiredParties()) != 2 {
		t.Fatalf("%d entries, %d built, %d shown", len(campaignLevels), campaignN, campaignShown())
	}
	slots := campaignSlots()
	for s, i := range slots {
		if campaignRetired(i) || campaignAtSlot(s) != i || campaignSlotOf(i) != s || campaignNumber(i) != s+1 {
			t.Errorf("slot %d -> party %d inconsistent", s, i)
		}
	}
	if campaignSlotOf(club) != -1 || campaignSlotOf(cine) != -1 || campaignAtSlot(len(slots)) != -1 {
		t.Error("a retired party has a row")
	}
	if levelDefs[slots[len(slots)-1]].Icon != "station" {
		t.Error("the last row is not the space station")
	}
	if campaignNext(club-1) != club+1 || campaignNext(cine-1) != cine+1 || campaignNext(0) != 1 {
		t.Error("the next party does not skip the retired ones")
	}
	for _, k := range medalLevels() {
		if k == "disco" || k == "cinema" {
			t.Errorf("the retired %s still gives a medal", k)
		}
	}
	// not in the random mode either
	for _, d := range levelDefs[campaignN:] {
		if d.Title == "DISCOTECA" || d.Title == "CINE" {
			t.Errorf("%s is still in the random mode", d.Title)
		}
	}
	// nor the RETO DE HOY, whatever the date
	for day := 1; day <= 60; day++ {
		dc := dailyPick(fmt.Sprintf("2026-%02d-%02d", 1+day/29, 1+day%28), 28, nil)
		if campaignRetired(dc.level) {
			t.Fatalf("the challenge of day %d is a retired party", day)
		}
	}
}

// Saves: a frontier on a retired party moves to the next one; wins in them are kept (but no longer counted); nothing
// else of the save changes; the migration is idempotent; an old plain JSON save of v1.0.0 loads.
func TestMigrateRev4RetiredParties(t *testing.T) {
	cases := []struct {
		name       string
		json       string
		unlocked   int
		keep       []string
		gone, wins int
	}{
		// v1.0.0 tester copy (b_some.json): 9 parties won, the DISCOTECA was next
		{"club next (v1.0.0)", `{"unlocked":10,"stats":{"distinct":9,"kills":900,"played":77,"won_0":1,"won_1":1,"won_2":1,"won_3":1,"won_4":1,"won_5":1,"won_6":1,"won_7":1,"won_8":1},"trophies":{"first_blood":true},"wallet":4321,"owned":["hat:pirate"],"album":{"elvis":3},"campaignRev":2,"started":1700000000}`,
			12, []string{"won_0", "won_5", "won_7", "won_9"}, 0, 9}, // (rev 5: won_6..won_8 -> won_7..won_9, ALCALÁ opens)
		{"cinema next (v1.3.0)", `{"unlocked":14,"stats":{"won_0":1,"won_9":1,"won_12":1},"campaignRev":3}`, 17, []string{"won_10", "won_13"}, 0, 2}, // (rev 6: GUADALAJARA opens too)
		{"both won", `{"unlocked":17,"stats":{"won_9":2,"won_13":1,"won_15":1},"medals":{"disco":3,"cinema":2},"campaignRev":3}`, 19, []string{"won_10", "won_14", "won_17"}, 0, 1},
		{"new game", `{"unlocked":1,"campaignRev":3}`, 1, nil, 0, 0},
		{"no stats, rev 0, festival open", `{"unlocked":11}`, 12, nil, 0, 0},
		{"everything", `{"unlocked":26,"stats":{"won_25":1},"campaignRev":3}`, 28, []string{"won_27"}, 0, 1},
	}
	for _, c := range cases {
		dir := t.TempDir()
		path := filepath.Join(dir, "progress.json")
		if err := os.WriteFile(path, []byte(c.json), 0o644); err != nil {
			t.Fatal(err)
		}
		p, found := readProgressFile(path)
		if !found {
			t.Fatalf("%s: the old save does not load", c.name)
		}
		before := cloneProgress(p)
		migrateProgress(&p)
		if p.CampaignRev != campaignRev || campaignRev != 6 {
			t.Errorf("%s: rev %d", c.name, p.CampaignRev)
		}
		if p.Unlocked != c.unlocked {
			t.Errorf("%s: unlocked %d, want %d", c.name, p.Unlocked, c.unlocked)
		}
		for _, k := range c.keep {
			if p.Stats[k] == 0 || p.Stats[k] != before.Stats[k] && before.CampaignRev >= 5 {
				t.Errorf("%s: %s lost (%d -> %d)", c.name, k, before.Stats[k], p.Stats[k])
			}
		}
		if before.Wallet != p.Wallet || len(before.Album) != len(p.Album) || len(before.Trophies) != len(p.Trophies) ||
			before.Medals["disco"] != p.Medals["disco"] || before.Medals["cinema"] != p.Medals["cinema"] {
			t.Errorf("%s: the migration touched something else", c.name)
		}
		g := &Game{progress: p}
		if n := g.careerLevelsDone(); campaignN == 28 && n != c.wins {
			t.Errorf("%s: %d parties counted as won, want %d (retired ones do not count)", c.name, n, c.wins)
		}
		again := cloneProgress(p)
		migrateProgress(&again)
		if again.Unlocked != p.Unlocked || fmt.Sprint(again.Stats) != fmt.Sprint(p.Stats) {
			t.Errorf("%s: not idempotent", c.name)
		}
	}
}

// Winning the party before a retired one opens the party after it; the interludes go by the row in the list.
func TestRetiredSkippedByUnlockAndInterludes(t *testing.T) {
	g := layerGame(t, false)
	prevP, prevIdx, prevFrozen := g.progress, g.levelIdx, progressFrozen
	t.Cleanup(func() { g.progress, g.levelIdx, progressFrozen = prevP, prevIdx, prevFrozen })
	progressFrozen = true
	g.progress = Progress{Unlocked: 10, Stats: map[string]int{}}
	g.levelIdx = 9 // the stadium, the frontier
	g.unlockNext()
	if g.progress.Unlocked != 12 {
		t.Errorf("after the stadium: unlocked %d, want 12 (the festival, the club skipped)", g.progress.Unlocked)
	}
	g.progress.Unlocked, g.levelIdx = 14, 13 // the graveyard
	g.unlockNext()
	if g.progress.Unlocked != 16 {
		t.Errorf("after the graveyard: unlocked %d, want 16 (GUADALAJARA, the cinema skipped)", g.progress.Unlocked)
	}
	if g.campaignMenuSel() != campaignSlotOf(15) {
		t.Errorf("CONTINUAR puts the cursor on row %d, want the circuit's %d", g.campaignMenuSel(), campaignSlotOf(15))
	}
	g.progress.Stats = map[string]int{} // no interlude seen yet
	for slot, want := range map[int]int{4: interCar, 9: interBoat, 14: interBalloon} {
		if k, ok := g.interludeAfter(campaignAtSlot(slot)); !ok || k != want {
			t.Errorf("no interlude after the party of row %d", slot+1)
		}
	}
	for _, i := range retiredParties() {
		if _, ok := g.interludeAfter(i); ok {
			t.Errorf("an interlude after the retired party %d", i)
		}
	}
}
