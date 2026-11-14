package main

import "testing"

func shopCounts() (n map[int]int, pts map[int]int) {
	n, pts = map[int]int{}, map[int]int{}
	for tab, l := range shopCatalog {
		for _, it := range l {
			if shopBuyable(it) {
				n[tab]++
				pts[tab] += it.price
			}
		}
	}
	return
}

func TestShortShopCounts(t *testing.T) {
	buildShopCatalog()
	n, _ := shopCounts()
	want := map[int]int{shopOutfits: 41, shopAccess: 61, shopPerks: 12, shopAnimals: 12, shopWeapons: 32} // +4 wild animals, +12 weapons (2026-11-09)
	for tab, w := range want {
		if n[tab] != w {
			t.Errorf("%s: %d articles for sale, want %d", shopTabNames[tab], n[tab], w)
		}
	}
	for _, tab := range []int{shopChars, shopKillers, shopPolice, shopUniforms, shopWardrobe} {
		if n[tab] != 0 {
			t.Errorf("%s must not sell anything", shopTabNames[tab])
		}
	}
	items, total := shopTotal()
	if items != 158 || total < 400000 || total > 650000 { // shopprice.go: ~50 parties to buy everything
		t.Fatalf("%d items, %d points", items, total)
	}
	// the price order of each list is kept (cheapest first)
	for tab, l := range shopCatalog {
		prev := 0
		for _, it := range l {
			if shopBuyable(it) && it.price < prev {
				t.Fatalf("%s: %s breaks the order", shopTabNames[tab], it.id)
			}
			if shopBuyable(it) {
				prev = it.price
			}
		}
	}
}

func TestShopTabsAndGroups(t *testing.T) {
	var names []string
	for _, gr := range shopGroups {
		names = append(names, gr.name)
	}
	if len(names) != 4 || names[0] != "JUGADOR" || names[1] != "ANIMALES" || names[2] != "ASESINOS" || names[3] != "VESTUARIO" {
		t.Fatalf("groups %v", names)
	}
	for _, tab := range shopFlat {
		if tab == shopPolice || tab == shopUniforms || tab == shopChars || tab == shopKillers {
			t.Fatalf("%s is not a shop list any more", shopTabNames[tab])
		}
		found := false
		for _, gr := range shopGroups {
			for _, x := range gr.tabs {
				found = found || x == tab
			}
		}
		if !found {
			t.Fatalf("%s is in no group", shopTabNames[tab])
		}
	}
}

// every id of the keep list is a real article, and what is not kept is retired (never lost)
func TestShopKeepAndRetired(t *testing.T) {
	buildShopCatalog()
	in := map[string]bool{}
	for _, tab := range []int{shopOutfits, shopAccess} {
		for _, it := range shopCatalog[tab] {
			in[it.id] = true
			if !shopKeepWear[it.id] && shopBuyable(it) {
				t.Errorf("%s is on sale but not in the keep list", it.id)
			}
		}
	}
	for id := range shopKeepWear {
		if !in[id] {
			t.Errorf("keep list id %q does not exist", id)
		}
	}
	if len(shopRetired) < 90 || len(shopRetired) > 130 {
		t.Fatalf("%d retired articles (76+135 - 41 - 60 = 110 expected)", len(shopRetired))
	}
	for _, it := range shopRetired {
		if in[it.id] || wearFx[it.id] == nil || it.slot == "" {
			t.Fatalf("retired %s must be out of the lists and still wearable", it.id)
		}
	}
}

// a save that owns everything of the old shop keeps wearing and seeing all of it; nothing in Owned changes
func TestRetiredStayOwnedAndWearable(t *testing.T) {
	g := goalsGame(t)
	var owned []string
	for _, it := range shopRetired {
		owned = append(owned, it.id)
	}
	g.progress.Owned = append([]string(nil), owned...)
	g.progress.Equip = map[string]string{"hat": "hat:chicken", "outfit": "out:tankroja", "glasses": "gl:round"}
	g.syncShop()
	if len(g.progress.Owned) != len(owned) {
		t.Fatal("Owned must not change")
	}
	for _, id := range owned {
		if !ownedID(id) {
			t.Fatalf("%s lost", id)
		}
	}
	sp := g.playerSpecFor(0, "", "")
	base := playerSpec()
	if sp.Hat == base.Hat {
		t.Fatal("the retired hat must still dress the player")
	}
	for _, slot := range []string{"hat", "outfit", "glasses"} {
		got := false
		for _, o := range g.wardOptions(slot) {
			got = got || o.id == g.progress.Equip[slot]
		}
		if !got {
			t.Errorf("the wardrobe %s row lost %s", slot, g.progress.Equip[slot])
		}
	}
	if g.wardName(0, "hat") != "Gorro de pollo" {
		t.Fatalf("name %q", g.wardName(0, "hat"))
	}
}

// BUY ALL still buys a whole list, and owning the whole short shop completes "Liquidación total"
func TestShortShopBuyAllAndClearance(t *testing.T) {
	g := goalsGame(t)
	prevMsg, prevTab, prevSel, prevOpen := g.shopMsg, g.shopTab, g.shopSel, g.shopBuyOpen // the shared game is drawn by the audit
	t.Cleanup(func() { g.shopMsg, g.shopTab, g.shopSel, g.shopBuyOpen = prevMsg, prevTab, prevSel, prevOpen })
	g.progress.Owned = nil
	g.progress.Stats = map[string]int{}
	g.progress.Wallet = 1 << 30
	g.syncShop()
	if shopAllBought() {
		t.Fatal("nothing bought yet")
	}
	for _, tab := range []int{shopOutfits, shopAccess, shopPerks, shopAnimals, shopWeapons} {
		g.shopTab = tab
		g.shopBuyAllAsk()
		if !g.shopBuyOpen {
			t.Fatalf("%s: no popup", shopTabNames[tab])
		}
		g.shopBuyAllDo()
	}
	n, _ := shopTotal()
	if !shopAllBought() || len(g.progress.Owned) != n || g.progress.Stats["shopall"] < 1 {
		t.Fatalf("owned %d of %d, shopall=%d", len(g.progress.Owned), n, g.progress.Stats["shopall"])
	}
	for _, tr := range trophyList {
		if tr.id == "s_shopall" && tr.need != 1 {
			t.Fatal("the secret trophy asks for one 'everything bought'")
		}
	}
}
