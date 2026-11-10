package main

import (
	"encoding/json"
	"testing"
)

// ---- police uniforms won by escaping (copsets.go) ----

func TestEscapeFormulaAndSets(t *testing.T) {
	if escapeNeed(1) != 3 || escapeNeed(2) != 6 || escapeNeed(24) != 72 || escapeNeed(25) != 76 || escapeNeed(36) != 120 {
		t.Fatalf("formula: %d %d %d %d %d", escapeNeed(1), escapeNeed(2), escapeNeed(24), escapeNeed(25), escapeNeed(36))
	}
	for n := 2; n <= 36; n++ {
		if escapeNeed(n) <= escapeNeed(n-1) {
			t.Fatalf("set %d does not ask for more escapes than %d", n, n-1)
		}
	}
	buildShopCatalog()
	order := copSetOrder()
	if len(order) != 36 {
		t.Fatalf("%d sets, want 36", len(order))
	}
	per := map[string]int{}
	seen := map[string]bool{}
	byID := map[string]shopItem{}
	for _, it := range shopCatalog[shopUniforms] {
		byID[it.id] = it
	}
	for _, id := range order {
		if seen[id] {
			t.Fatalf("%s twice", id)
		}
		seen[id] = true
		it, ok := byID[id]
		if !ok || wearFx[id] == nil {
			t.Fatalf("%s is not a known uniform", id)
		}
		if it.trophy != "escapes" || shopBuyable(it) {
			t.Fatalf("%s must be locked behind escapes and never for sale: %+v", id, it)
		}
		per[it.slot]++
	}
	if per["cop"] != 12 || per["fbi"] != 6 || per["army"] != 6 || per["swat"] != 6 || per["rangers"] != 6 {
		t.Fatalf("sets per force: %v", per)
	}
	// the opening order is mixed: no force has three in a row at the start
	if order[0][:3] != "cop" || order[2][:3] == "cop" {
		t.Fatalf("order starts %v", order[:4])
	}
	if copSetsOpen(0) != 0 || copSetsOpen(2) != 0 || copSetsOpen(3) != 1 || copSetsOpen(5) != 1 || copSetsOpen(6) != 2 || copSetsOpen(120) != 36 || copSetsOpen(9999) != 36 {
		t.Fatal("copSetsOpen")
	}
	if need, ok := copSetNext(7); !ok || need != 9 {
		t.Fatalf("next set for 7 escapes asks %d", need)
	}
	if _, ok := copSetNext(120); ok {
		t.Fatal("nothing is left after 120")
	}
	// every old uniform that left the catalogue is still known and never for sale
	retired := 0
	for _, it := range shopCatalog[shopUniforms] {
		if it.trophy == "retired" {
			retired++
			if wearFx[it.id] == nil || shopBuyable(it) {
				t.Fatalf("%s must stay wearable and not buyable", it.id)
			}
		}
	}
	if retired != 72 {
		t.Fatalf("%d retired uniforms, want 72 (108 - 36)", retired)
	}
}

func TestEscapesCountOnlyRealEscapes(t *testing.T) {
	g := goalsGame(t)
	g.progress.Escapes = 0
	g.run.wasSus = [2]bool{}
	step := func(wanted float64) {
		g.wanted = wanted
		g.trackCuffs()
	}
	// being wanted and getting away
	step(40)
	if g.progress.Escapes != 0 {
		t.Fatal("being wanted is not escaping")
	}
	step(0)
	if g.progress.Escapes != 1 {
		t.Fatalf("the search ended: %d escapes, want 1", g.progress.Escapes)
	}
	step(0)
	if g.progress.Escapes != 1 {
		t.Fatal("one escape counts once")
	}
	// dying while wanted is not an escape
	step(40)
	g.p.dead = true
	step(0)
	g.p.dead = false
	if g.progress.Escapes != 1 {
		t.Fatal("a dead player did not escape")
	}
	// being taken (custody) is not an escape either
	step(40)
	g.p.jail.held = true
	step(0)
	g.p.jail.held = false
	if g.progress.Escapes != 1 {
		t.Fatal("a prisoner did not escape")
	}
	// a second escape
	g.run.wasSus = [2]bool{}
	step(40)
	step(0)
	if g.progress.Escapes != 2 {
		t.Fatalf("%d escapes, want 2", g.progress.Escapes)
	}
}

func TestEscapesOpenUniformsAndTheyAreWorn(t *testing.T) {
	g := goalsGame(t)
	g.progress.Escapes = 2
	g.progress.Equip = map[string]string{}
	g.syncShop()
	first := copSetOrder()[0]
	if ownedID(first) {
		t.Fatal("2 escapes open nothing")
	}
	g.run.wasSus = [2]bool{}
	g.wanted = 40
	g.trackCuffs()
	g.wanted = 0
	g.trackCuffs() // the 3rd escape
	if g.progress.Escapes != 3 || !ownedID(first) || ownedID(copSetOrder()[1]) {
		t.Fatalf("escapes=%d first owned=%v second owned=%v", g.progress.Escapes, ownedID(first), ownedID(copSetOrder()[1]))
	}
	for _, o := range g.progress.Owned {
		if o == first {
			t.Fatal("the opened sets are derived from the escapes, never written to Owned")
		}
	}
	// it shows in the POLICÍAS page of the wardrobe and, once chosen, the local officers wear it
	found := false
	for _, o := range g.wardOptions("cop") {
		found = found || o.id == first
	}
	if !found {
		t.Fatal("the wardrobe must offer the uniform the escapes opened")
	}
	for _, o := range g.wardOptions("fbi") {
		if o.id != "" && copSetRank(o.id) > 0 {
			t.Fatalf("%s is not open yet", o.id)
		}
	}
	prevKind := wardKind
	t.Cleanup(func() { wardKind = prevKind })
	wardKind = wardKindCop
	g.wardCycle(0, "cop", 1)
	if g.shopEquip("cop") != first {
		t.Fatalf("the wardrobe put %q on the police", g.shopEquip("cop"))
	}
	if g.store == nil || len(g.store.Chars["police"]) == 0 {
		g.applyShop()
		if g.as.copCS == nil || g.as.copCS.Spec.Shirt != hx(0x3a6a3a) {
			t.Fatal("the local officers must wear the chosen uniform")
		}
	}
}

// somebody who bought uniforms in the old shop keeps all of them, kept or retired
func TestOldUniformsKept(t *testing.T) {
	g := goalsGame(t)
	g.progress.Escapes = 0
	g.progress.Owned = []string{"cop:hat3", "cop:guardia", "fbi:c2", "swat:azul", "out:hoodazul", "hat:chicken", "cape:blanca"}
	g.progress.Equip = map[string]string{"cop": "cop:hat3", "hat": "hat:chicken"}
	g.syncShop()
	for _, id := range g.progress.Owned {
		if !ownedID(id) {
			t.Fatalf("%s lost", id)
		}
	}
	has := func(slot, id string) bool {
		for _, o := range g.wardOptions(slot) {
			if o.id == id {
				return true
			}
		}
		return false
	}
	if !has("cop", "cop:hat3") || !has("cop", "cop:guardia") || !has("fbi", "fbi:c2") || !has("swat", "swat:azul") {
		t.Fatal("the wardrobe must list the uniforms the save already had")
	}
	if has("cop", "cop:hat4") || has("army", "army:desierto") {
		t.Fatal("what was not bought is not listed")
	}
	if !has("outfit", "out:hoodazul") || !has("hat", "hat:chicken") || !has("cape", "cape:blanca") {
		t.Fatal("clothes and accessories that left the shop stay in the wardrobe of whoever owns them")
	}
	if g.shopEquip("cop") != "cop:hat3" || wearFx["hat:chicken"] == nil || wearFx["cop:hat3"] == nil {
		t.Fatal("what was equipped still dresses")
	}
	// the kept sets are the only ones that open with escapes: a retired id is not opened by any number of them
	g.progress.Escapes = 9999
	g.syncShop()
	if !ownedID("cop:guardia") || !ownedID("rangers:dorados") {
		t.Fatal("all 36 open at 120 escapes")
	}
}

// an old progress.json (no escapes, no police cards) loads with everything it had
func TestOldProgressNothingLost(t *testing.T) {
	old := `{"unlocked":22,"stats":{"kills":340,"played":50},"trophies":{"t_first":true},"wallet":1234,
	 "owned":["skin:lisa","killer:pinhead","police:1","cop:vikingo","fbi:negro","out:suitazul","hat:chicken","kw:knife","animal:duck","perk:speed2"],
	 "equip":{"hat":"hat:chicken","cop":"cop:vikingo","kweapon":"kw:knife"}}`
	var p Progress
	if err := json.Unmarshal([]byte(old), &p); err != nil {
		t.Fatal(err)
	}
	migrateProgress(&p)
	if p.Escapes != 0 || p.Wallet != 1234 || len(p.Owned) != 10 || p.Equip["cop"] != "cop:vikingo" {
		t.Fatalf("%+v", p)
	}
	out, _ := json.Marshal(p)
	var back map[string]any
	json.Unmarshal(out, &back)
	if _, ok := back["escapes"]; ok {
		t.Fatal("escapes must be omitted while it is 0 (omitempty)")
	}
	g := goalsGame(t)
	g.progress = p
	g.syncShop()
	for _, id := range p.Owned {
		if !ownedID(id) {
			t.Fatalf("%s lost", id)
		}
	}
	// the card of the type the save had unlocked: the FBI by the old tier, the local police by the uniform it bought
	if !g.copGot(unitFBI) || !g.copGot(unitPolice) || g.copGot(unitSWAT) {
		t.Fatal("police cards of an old save")
	}
	var found bool
	for _, o := range g.wardOptions("cop") {
		found = found || o.id == "cop:vikingo"
	}
	if !found {
		t.Fatal("the bought uniform is listed")
	}
}

// ---- POLICÍAS cards of OBJETIVOS (goalcops.go) ----

func TestCopCards(t *testing.T) {
	g := goalsGame(t)
	var cards []albumCard
	for _, c := range albumCards() {
		if c.kind() == cardCop {
			cards = append(cards, c)
		}
	}
	if len(cards) != 5 || len(goalTabs) != tabCount || tabCops < 0 || tabCops >= tabCount ||
		goalTabs[tabCops].kind != cardCop || goalTabs[tabCops].name != "POLICÍAS" {
		t.Fatalf("%d police cards, tabs %v", len(cards), goalTabs)
	}
	for _, c := range cards {
		if g.albumUnlocked(c) {
			t.Fatalf("%s unlocked in an empty save", c.key)
		}
		if charKey(c.key) {
			t.Fatalf("%s must not count as a character (shopOwned skin:)", c.key)
		}
	}
	// killing an officer of a type unlocks that card, not the others
	g.copDefeated(&Cop{kind: unitSWAT})
	if !g.copGot(unitSWAT) || g.copGot(unitFBI) || g.copGot(unitPolice) {
		t.Fatal("killing a SWAT officer unlocks the SWAT card only")
	}
	if ownedID("skin:cop:3") {
		t.Fatal("a police card is not a character")
	}
	g.copDefeated(&Cop{kind: unitRangers, medic: true})
	if g.copGot(unitRangers) {
		t.Fatal("medics are not a police type")
	}
	// the milestone that brings the type also marks it
	g.progress.Owned = []string{"police:2"}
	g.syncShop()
	if !g.copGot(unitMilitary) {
		t.Fatal("the army milestone unlocks the army card")
	}
	// hints
	if h := cards[0].lockedHint(); h == "" {
		t.Fatal("hint")
	}
	if cards[1].lockedHint() == cards[2].lockedHint() {
		t.Fatal("each tier states its own milestone")
	}
}

func TestPoliceStabAndTrapMarkTheCard(t *testing.T) {
	g := goalsGame(t)
	for _, kind := range []int{unitPolice, unitFBI} {
		c := &Cop{kind: kind, pos: g.p.pos}
		g.cops = append(g.cops, c)
		if kind == unitPolice {
			g.stabCop(c)
		} else {
			g.trapCop(c)
		}
	}
	if !g.copGot(unitPolice) || !g.copGot(unitFBI) {
		t.Fatal("a stab and a trap on officers mark their cards")
	}
}
