package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// ---- POLICÍAS: the types come strictly in order by career kills (goalcops.go copTypeOpen, 2026-11-11) ----

// copReportSave: a synthetic copy of the save of the report (no personal data): 447 career kills (+63 trap, 13 poison
// stats), 11 escapes, the trophies "Mano dura" (army:tr_gala) and "Cinco estrellas" (swat:tr_plata) already won, one
// local officer killed, and two contaminated cards (army and SWAT marked killed) to prove the reconciliation.
func copReportSave() Progress {
	return Progress{
		Unlocked: 6, Wallet: 50123, CampaignRev: campaignRev, MedalRev: 1, Escapes: 11, Started: 1794381849,
		Stats: map[string]int{"kills": 447, "traps": 63, "poison": 13, "escapes": 11, "guards": 40, "stars": 5,
			"won_0": 1, "won_1": 1, "won_2": 1, "won_3": 1, "won_4": 1},
		Trophies:  map[string]bool{"t_k100": true, "t_k25": true, "t_g10": true, "t_s4": true, "t_s5": true, "t_first": true},
		Owned:     []string{"hat:tr_novato", "army:tr_gala", "cop:tr_gala", "swat:tr_plata", "cape:tr_100"},
		Album:     map[string]int{"cop:0": albumKilled, "cop:2": albumKilled, "cop:3": albumKilled, "yoda": 9, "vip:chef": 2},
		CharKills: map[string]int{"yoda": 2},
		Medals:    map[string]int{"beach": 3, "bull": 3, "house": 3, "neon": 3, "ranch": 3},
	}
}

// copCardsState: the seal of every POLICÍAS card, in order (local, FBI, army, SWAT, rangers); "" = "???".
func copCardsState(g *Game) []string {
	var out []string
	for _, c := range copCards() {
		s, _ := g.albumSeal(c)
		out = append(out, s)
	}
	return out
}

// The bug of the report: 447 kills showed army and SWAT CONSEGUIDO (their trophy uniforms opened them) before the FBI.
func TestCopUserSave447OnlyLocal(t *testing.T) {
	g := goalsGame(t)
	path := progressTempDir(t) // a temp assets/ folder: never the real progress.json
	pre := copReportSave()
	raw, err := encodeProgress(pre)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	p := loadProgress() // decode + migrate, as the game does
	g.progress = p
	g.syncShop()
	if got := copCardsState(g); fmt.Sprint(got) != fmt.Sprint([]string{"ELIMINADO", "", "", "", ""}) {
		t.Fatalf("POLICÍAS cards with 447 kills: %q, want only the local police (ELIMINADO)", got)
	}
	for _, u := range []int{unitFBI, unitMilitary, unitSWAT, unitRangers} {
		if g.copTypeOpen(u) || g.copGot(u) {
			t.Errorf("type %d open with 447 kills", u)
		}
		if _, ok := p.Album[copKey(u)]; ok {
			t.Errorf("contaminated card %s kept", copKey(u))
		}
	}
	if unitForStars(5) != unitPolice || len(policeGroups()) != 1 {
		t.Fatalf("only the local police may come: unitForStars(5)=%d groups=%v", unitForStars(5), policeGroups())
	}
	// nothing else moved: kills, trophies, medals, money, uniforms, other cards, revisions
	if p.Stats["kills"] != 447 || p.Wallet != 50123 || p.Escapes != 11 || len(p.Trophies) != len(pre.Trophies) ||
		len(p.Medals) != 5 || fmt.Sprint(p.Owned) != fmt.Sprint(pre.Owned) || p.Album["cop:0"] != albumKilled ||
		p.Album["yoda"] != 9 || p.Album["vip:chef"] != 2 || p.CharKills["yoda"] != 2 {
		t.Fatalf("the reconciliation touched other data: %+v", p)
	}
	if p.CampaignRev != campaignRev || p.MedalRev != 1 || p.CopRev != copRev || p.Unlocked != 6 {
		t.Fatalf("revisions: campaign %d medal %d cop %d unlocked %d", p.CampaignRev, p.MedalRev, p.CopRev, p.Unlocked)
	}
	// the escapes line is still there and the trophy uniforms are still worn
	if g.copEscapeLine() != "Escapes: 11 (próximo uniforme: 12)" || !ownedID("army:tr_gala") || !ownedID("swat:tr_plata") {
		t.Fatalf("escapes line %q / trophy uniforms lost", g.copEscapeLine())
	}
	// the detail window shows "???" for a closed type
	g.trophyTab = tabCops
	for i, c := range g.albumView() {
		g.albumSel = i
		d := g.goalDetailContent()
		if c.roleKind > unitPolice && (d.name != "???" || d.seal != "" || d.ok) {
			t.Errorf("detail of closed %s: name %q seal %q ok %v", c.key, d.name, d.seal, d.ok)
		}
	}
	// saved again and loaded again: idempotent
	saveProgress(g.progress)
	if q := loadProgress(); fmt.Sprint(q.Album) != fmt.Sprint(p.Album) || q.CopRev != copRev || q.Stats["kills"] != 447 {
		t.Fatal("second load changed the save")
	}
}

// 0 / 499 / 500 / 999 / 1000 / 1499 / 1500 / 2000 career kills: the types open one by one, FBI first.
func TestCopUnlockStrictOrderByCareerKills(t *testing.T) {
	g := goalsGame(t)
	cases := []struct {
		kills int
		open  int // highest open unit
	}{{0, unitPolice}, {499, unitPolice}, {500, unitFBI}, {999, unitFBI}, {1000, unitMilitary}, {1499, unitMilitary},
		{1500, unitSWAT}, {1999, unitSWAT}, {2000, unitRangers}}
	for _, c := range cases {
		g.progress = Progress{Stats: map[string]int{"kills": c.kills}, Trophies: map[string]bool{},
			Owned: []string{"army:tr_gala", "swat:tr_plata", "cop:tr_gala"}}
		g.syncShop()
		g.trophyDirty = true
		g.checkTrophies()
		g.syncShop()
		for _, f := range copForces {
			want := f.unit <= c.open
			if g.copTypeOpen(f.unit) != want {
				t.Errorf("%d kills: %s open=%v want %v", c.kills, f.es, !want, want)
			}
			if f.unit > unitPolice && g.copGot(f.unit) != want {
				t.Errorf("%d kills: %s card won=%v want %v", c.kills, f.es, !want, want)
			}
		}
		if unitForStars(5) != c.open || len(policeGroups()) != c.open+1 {
			t.Errorf("%d kills: unitForStars(5)=%d groups=%v, want up to %d", c.kills, unitForStars(5), policeGroups(), c.open)
		}
		// killing an officer of a closed type marks nothing; of an open type, ELIMINADO
		for _, f := range copForces {
			g.copDefeated(&Cop{kind: f.unit})
			killed := g.progress.Album[copKey(f.unit)]&albumKilled != 0
			if killed != (f.unit <= c.open) {
				t.Errorf("%d kills: defeating %s marked=%v", c.kills, f.es, killed)
			}
		}
		for _, cd := range copCards() {
			s, _ := g.albumSeal(cd)
			if (cd.roleKind <= c.open) != (s == "ELIMINADO") {
				t.Errorf("%d kills: seal of %s = %q", c.kills, cd.nameES, s)
			}
		}
	}
	// an out-of-order save (army and SWAT tiers without the FBI one, e.g. a debug copy) opens nothing above local
	g.progress = Progress{Stats: map[string]int{"kills": 10}, Trophies: map[string]bool{}, Owned: []string{"police:2", "police:3"}}
	g.syncShop()
	if g.copTypeOpen(unitMilitary) || g.copTypeOpen(unitSWAT) || unitForStars(4) != unitPolice || len(policeGroups()) != 1 {
		t.Fatal("the order FBI -> army -> SWAT cannot be skipped")
	}
}

// A v1.0.0 / v1.1.x save keeps every legit card: the reconciliation only clears cards of closed types.
func TestCopReconcileOldSaves(t *testing.T) {
	// v1.1.x: 1200 kills (FBI + army open, trophies t_cop1/t_cop2 and police:1/2 owned), army killed, SWAT marked by error
	p := Progress{Unlocked: 26, CampaignRev: 3, MedalRev: 1, MenuTier: 2, Wallet: 7,
		Stats:    map[string]int{"kills": 1200},
		Trophies: map[string]bool{"t_cop1": true, "t_cop2": true},
		Owned:    []string{"police:1", "police:2", "swat:tr_plata"},
		Album:    map[string]int{"cop:0": albumKilled, "cop:1": albumKilled, "cop:2": albumKilled, "cop:3": albumKilled, "cop:4": albumKilled}}
	migrateProgress(&p)
	for k, want := range map[string]bool{"cop:0": true, "cop:1": true, "cop:2": true, "cop:3": false, "cop:4": false} {
		if _, ok := p.Album[k]; ok != want {
			t.Errorf("v1.1.x %s kept=%v want %v", k, ok, want)
		}
	}
	if p.MenuTier != 2 || p.MedalRev != 1 || p.CampaignRev != 3 || p.Wallet != 7 || p.Stats["kills"] != 1200 || p.CopRev != copRev {
		t.Fatalf("v1.1.x save changed: %+v", p)
	}
	// v1.0.0 (CampaignRev 2): the trophies alone keep the cards (the kills stat of old saves may be lower)
	q := Progress{Unlocked: 25, CampaignRev: 2, Stats: map[string]int{"kills": 30},
		Trophies: map[string]bool{"t_cop1": true, "t_cop2": true, "t_cop3": true, "t_cop4": true},
		Album:    map[string]int{"cop:1": albumKilled, "cop:4": albumKilled}}
	migrateProgress(&q)
	if q.Album["cop:1"] == 0 || q.Album["cop:4"] == 0 || q.CampaignRev != campaignRev {
		t.Fatalf("v1.0.0 save lost legit cards: %+v", q.Album)
	}
	// a save already reconciled is not touched again (CopRev 1), and a brand new save is fine
	r := Progress{CopRev: copRev, Album: map[string]int{"cop:3": albumKilled}}
	migrateProgress(&r)
	if r.Album["cop:3"] == 0 {
		t.Fatal("CopRev 1: the reconciliation runs once")
	}
	var n Progress
	migrateProgress(&n)
	if n.CopRev != copRev || n.Album != nil {
		t.Fatalf("new save: %+v", n)
	}
	// omitempty: an untouched old save does not grow a copRev field until it is written by this version
	b, _ := json.Marshal(Progress{})
	if bytes.Contains(b, []byte("copRev")) {
		t.Fatal("copRev must be omitempty")
	}
}

// DEBUG: nothing counts. The Konami session gives the real progress back untouched (kills, escapes, cards,
// uniforms), and a frozen run (any debug flag: debugFlagsActive, TestDebugFlagsFreezeProgress) writes nothing.
func TestCopDebugCountsNothing(t *testing.T) {
	snapshotGlobals(t)
	g := goalsGame(t)
	path := progressTempDir(t)
	base := copReportSave()
	base.Stats["kills"] = 600 // FBI open, so a kill of an FBI agent would count outside the debug mode
	base.Owned = append(base.Owned, "police:1")
	base.Trophies["t_cop1"] = true
	base.CopRev = copRev
	delete(base.Album, "cop:2") // a clean save: only the local card
	delete(base.Album, "cop:3")
	saveProgress(base)
	before, _ := os.ReadFile(path)
	g.progress = loadProgress()
	g.syncShop()
	g.progress.Equip = map[string]string{} // debugEnd gives back an empty map for a nil one (same content)
	want, _ := json.Marshal(g.progress)

	play := func() {
		for i := 0; i < 900; i++ { // career kills well past every milestone
			g.stat("kills", 1)
		}
		g.trophyDirty = true
		g.checkTrophies()
		for i := 0; i < 30; i++ {
			g.cuffEscaped()
		}
		g.copDefeated(&Cop{kind: unitFBI})
		g.copDefeated(&Cop{kind: unitSWAT})
		saveProgress(g.progress)
	}
	// 1) the DEBUG menu (Konami): a copy, frozen, restored by "Salir del DEBUG"
	progressFrozen, debugSnap = false, nil
	g.debugApply()
	play()
	g.debugEnd()
	g.syncShop()
	if got, _ := json.Marshal(g.progress); string(got) != string(want) {
		t.Fatalf("DEBUG session leaked into the progress:\n%s\n%s", got, want)
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Fatal("DEBUG session wrote the save")
	}
	// 2) a run with debug flags (-level, -wildnow, -trexkill...): main.go freezes the progress before anything
	progressFrozen = true
	play()
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Fatal("a debug-flag run wrote the save")
	}
	progressFrozen = false
	if q := loadProgress(); q.Stats["kills"] != 600 || q.Escapes != 11 || q.Album["cop:1"] != 0 || q.Album["cop:3"] != 0 {
		t.Fatalf("the save on disk counted debug play: kills %d escapes %d album %v", q.Stats["kills"], q.Escapes, q.Album)
	}
}
