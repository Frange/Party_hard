package main

import (
	"encoding/json"
	"testing"
)

// goalsGame: the shared headless game with a private, frozen, empty progress; every global it touches is restored.
func goalsGame(t *testing.T) *Game {
	g := killerGame(t, false)
	prevP, prevOwned, prevCast, prevMiss := cloneProgress(g.progress), shopOwned, castSlugs, vipMisses
	t.Cleanup(func() { g.progress, shopOwned, castSlugs, vipMisses = prevP, prevOwned, prevCast, prevMiss })
	g.progress.Album, g.progress.Owned, g.progress.KillerWait = nil, nil, 0
	g.syncShop()
	return g
}

// The three collections add up: 50 VIPs, 23+4 killers (21 + the endoskeleton + the beak doctor), every character of the old shop.
func TestGoalsCatalogs(t *testing.T) {
	n := map[int]int{}
	keys := map[string]bool{}
	for _, c := range albumCards() {
		if keys[c.key] {
			t.Fatalf("duplicate card %s", c.key)
		}
		keys[c.key] = true
		n[c.kind()]++
	}
	if n[cardVIP] != 50 {
		t.Errorf("%d VIPs, want 50", n[cardVIP])
	}
	if n[cardKiller] != 27 || len(killerCatalog) != 23 || len(killerTrophy) != 4 {
		t.Errorf("killers: %d cards, %d + %d", n[cardKiller], len(killerCatalog), len(killerTrophy))
	}
	if n[cardChar] < 282 {
		t.Errorf("%d characters, want at least the 282 of the old shop", n[cardChar])
	}
	for _, s := range charCatalogSlugs {
		if presetByName(s) == nil {
			t.Errorf("%s has no preset", s)
		}
		if !keys[s] {
			t.Errorf("%s has no card", s)
		}
	}
	if len(charCatalogSlugs) != 282 {
		t.Errorf("%d catalogue characters, want 282", len(charCatalogSlugs))
	}
	// the tabs are data: one per collection, in order, each with a distinct kind
	if len(goalTabs) != tabCount || goalTabs[tabTrophies].kind != -1 {
		t.Error("tabs")
	}
}

// Every killer that left the catalogue maps to a kept one; the kept ones and the trophy ones exist as presets.
func TestKillerCatalogAndLegacy(t *testing.T) {
	for _, e := range killerPresets {
		_, trophy := killerTrophyOf(e.Name)
		switch {
		case killerInCatalog(e.Name) || trophy:
		case killerLegacy[e.Name] != "" && killerInCatalog(killerLegacy[e.Name]):
		default:
			t.Errorf("killer %s is neither kept nor mapped to a kept one", e.Name)
		}
	}
	for _, k := range killerCatalog {
		if killerSpecOf(k.slug).Name != k.slug {
			t.Errorf("%s: no preset", k.slug)
		}
		if k.es == "" || k.en == "" {
			t.Errorf("%s: unnamed", k.slug)
		}
	}
	if len(killerLegacy) != len(killerPresets)-len(killerCatalog)-len(killerTrophy) {
		t.Errorf("%d legacy mappings for %d retired killers", len(killerLegacy), len(killerPresets)-len(killerCatalog)-len(killerTrophy))
	}
}

// Killing a guest of a kind unlocks the character; seeing him does not; old purchases keep working.
func TestCharUnlockByKillAndMigration(t *testing.T) {
	g := goalsGame(t)
	card := func(slug string) albumCard {
		for _, c := range albumCards() {
			if c.key == slug {
				return c
			}
		}
		t.Fatalf("no card %s", slug)
		return albumCard{}
	}
	g.albumMark("homer", albumSeen)
	if g.albumUnlocked(card("homer")) || !g.albumSeenOnly(card("homer")) {
		t.Fatal("seen is not unlocked")
	}
	if ownedID("skin:homer") {
		t.Fatal("not owned yet")
	}
	g.albumMark("homer", albumKilled)
	if !g.albumUnlocked(card("homer")) || !ownedID("skin:homer") {
		t.Fatal("a kill unlocks the character and the wardrobe sees him")
	}
	// the kills are remembered after a reload (syncShop reads the album)
	g.progress.Album = map[string]int{"bart": albumKilled}
	g.syncShop()
	if !ownedID("skin:bart") || ownedID("skin:homer") {
		t.Fatal("syncShop must rebuild the lookup from the album")
	}
	for _, c := range castSlugs {
		if c == "bart" {
			t.Fatal("a character unlocked by a kill must not be pushed into the party more often")
		}
	}
	// an old save that BOUGHT a character keeps it, with no album at all
	g.progress.Album = nil
	g.progress.Owned = []string{"skin:lisa", "killer:pinhead"}
	g.syncShop()
	if !g.albumUnlocked(card("lisa")) {
		t.Fatal("a character bought before the redesign must stay unlocked")
	}
	found := false
	for _, o := range g.wardOptions("base") {
		found = found || o.id == "lisa"
	}
	if !found {
		t.Error("the wardrobe must offer the unlocked character")
	}
	// ...and the bought killer that left the catalogue counts as the kept one it maps to
	if !g.killerGot("freddy") || g.killerGot("jason") {
		t.Fatal("killer:pinhead must unlock freddy (and nothing else)")
	}
}

// A character of a trophy (La Leyenda...) unlocks with the trophy, not by a kill, and is not in the visitors.
func TestCharTrophyRewardsAndVisitors(t *testing.T) {
	g := goalsGame(t)
	var tro []string
	for _, c := range albumCards() {
		if c.kind() == cardChar && c.trophyID != "" {
			tro = append(tro, c.slug)
		}
	}
	if len(tro) < 9 {
		t.Fatalf("only %d trophy characters", len(tro))
	}
	for _, s := range g.charLockedPool() {
		for _, x := range tro {
			if s == x {
				t.Fatalf("%s is a trophy reward: not a visitor", s)
			}
		}
		if isKidSlug(s) {
			t.Fatalf("%s arrives by himself", s)
		}
	}
	before := len(g.guests)
	g.spawnGoalVisitors()
	if added := len(g.guests) - before; added != charVisitors {
		t.Fatalf("%d visitors, want %d", added, charVisitors)
	}
	seen := map[string]bool{}
	for _, gu := range g.guests[before:] {
		n := gu.cs.Spec.Name
		if seen[n] || charUnlocked(g.progress.Album, n) {
			t.Fatalf("visitor %q repeated or already unlocked", n)
		}
		seen[n] = true
	}
	// with everything unlocked nobody comes
	g.progress.Album = map[string]int{}
	for _, c := range albumCards() {
		if c.kind() == cardChar {
			g.progress.Album[c.key] = albumKilled
		}
	}
	before = len(g.guests)
	g.spawnGoalVisitors()
	if len(g.guests) != before {
		t.Fatal("all unlocked: no visitors")
	}
}

// The killer that comes is one the player does not have; with all of them, any.
func TestKillerPickPrefersMissing(t *testing.T) {
	g := goalsGame(t)
	g.progress.Album = map[string]int{}
	for _, k := range killerCatalog[1:] {
		g.progress.Album["killer:"+k.slug] = albumKilled
	}
	for i := 0; i < 40; i++ {
		if _, _, name := g.pickKiller(); name != killerCatalog[0].slug {
			t.Fatalf("picked %s, the only one missing is %s", name, killerCatalog[0].slug)
		}
	}
	g.progress.Album["killer:"+killerCatalog[0].slug] = albumKilled
	seen := map[string]bool{}
	for i := 0; i < 600; i++ {
		_, _, name := g.pickKiller()
		if !killerInCatalog(name) {
			t.Fatalf("%s is not allowed", name)
		}
		seen[name] = true
	}
	if len(seen) < 15 {
		t.Errorf("all unlocked: only %d different killers in 600 picks", len(seen))
	}
}

// Killing a killer unlocks his card and guarantees the next different one within 2 parties, never back to back.
func TestKillerGuaranteeAfterKill(t *testing.T) {
	g := goalsGame(t)
	g.killerDefeated("mimo")
	if !g.albumUnlocked(albumCard{key: "killer:mimo", slug: "mimo", killer: true}) || g.progress.KillerWait != killerNextWithin {
		t.Fatalf("kill: card=%v wait=%d", g.progress.Album, g.progress.KillerWait)
	}
	// the party where it happened had a killer (planned): the next one can never have one...
	g.kp = killerPlan{planned: true, at: 30}
	g.killerLevelReset()
	if g.kp.planned {
		t.Fatal("the very next party cannot have a killer (never two in a row)")
	}
	if g.progress.KillerWait != 1 {
		t.Fatalf("wait = %d after one party, want 1", g.progress.KillerWait)
	}
	// ...and the second one has it for sure, whatever the dice say
	g.killerLevelReset()
	if !g.kp.planned || g.kp.at < killerAtMin || g.kp.at > killerAtMax {
		t.Fatalf("the second party after the kill must have a killer: %+v", g.kp)
	}
	g.killerArrived()
	if g.progress.KillerWait != 0 {
		t.Fatal("the guarantee is paid when he arrives")
	}
	// a stealth party in between does not count and does not break the wait
	g.progress.KillerWait = 2
	g.kp = killerPlan{}
	g.stealth = true
	g.killerLevelReset()
	g.stealth = false
	if g.progress.KillerWait != 2 || g.kp.planned {
		t.Fatal("stealth parties are skipped")
	}
}

// With every killer unlocked there is nothing to guarantee: the base rate rules.
func TestKillerNoGuaranteeWhenAllGot(t *testing.T) {
	g := goalsGame(t)
	g.progress.Album = map[string]int{}
	for _, k := range killerCatalog {
		g.progress.Album["killer:"+k.slug] = albumKilled
	}
	g.killerDefeated("mimo")
	if g.progress.KillerWait != 0 {
		t.Fatalf("wait %d with all killers unlocked", g.progress.KillerWait)
	}
}

// The rate of parties with a killer is the old one (~0.18, never back to back) when no killer is ever killed.
func TestKillerBaseRateUnchangedByTheGuarantee(t *testing.T) {
	g := goalsGame(t)
	g.progress.KillerWait = 0
	hits, prev := 0, false
	for i := 0; i < 4000; i++ {
		g.kp = killerPlan{planned: prev}
		g.killerLevelReset()
		if g.kp.planned && prev {
			t.Fatal("two in a row")
		}
		if g.kp.planned {
			hits++
		}
		prev = g.kp.planned
	}
	if rate := float64(hits) / 4000; rate < 1.0/7 || rate > 1.0/4.5 {
		t.Errorf("base rate %.3f, want about 0.18", rate)
	}
}

// The new field is optional in the save: an old file loads, and it is not written when 0.
func TestProgressKillerWaitIsBackwardCompatible(t *testing.T) {
	var p Progress
	if err := json.Unmarshal([]byte(`{"unlocked":4,"stats":{"kills":9},"owned":["skin:homer","killer:pinhead"]}`), &p); err != nil || p.KillerWait != 0 {
		t.Fatalf("old save: %v wait=%d", err, p.KillerWait)
	}
	b, _ := json.Marshal(Progress{Unlocked: 1})
	if containsStr(string(b), "killerWait") {
		t.Error("KillerWait must be omitted when 0")
	}
	p.KillerWait = 2
	c := cloneProgress(p)
	if c.KillerWait != 2 {
		t.Error("clone loses KillerWait")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// The menu says OBJETIVOS / GOALS.
func TestMenuSaysGoals(t *testing.T) {
	if menuLabel(menuTrophies) != "OBJETIVOS" {
		t.Errorf("menu label %q", menuLabel(menuTrophies))
	}
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	loadLanguages()
	uiLang = "en"
	if tr("OBJETIVOS") != "GOALS" || tr("PERSONAJES") != "CHARACTERS" || tr("ASESINOS") != "KILLERS" {
		t.Error("English tabs")
	}
}
