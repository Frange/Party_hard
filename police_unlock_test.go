package main

import "testing"

func freshShopGame() *Game {
	progressFrozen = true // nothing here may touch the real progress.json
	g := &Game{snd: &Sounds{muted: true}, as: &Assets{}}
	g.progress = Progress{Stats: map[string]int{}, Trophies: map[string]bool{}}
	buildShopCatalog()
	g.syncShop()
	return g
}

// the police types are not for sale any more: each one comes with a kills trophy
func TestPoliceTypesNotForSale(t *testing.T) {
	buildShopCatalog()
	n := 0
	for _, it := range shopCatalog[shopPolice] {
		if it.id == "police:0" {
			if it.price != 0 || it.trophy != "" {
				t.Fatal("the local police is always there")
			}
			continue
		}
		n++
		if it.trophy == "" || it.price < 1<<20 {
			t.Fatalf("%s must be locked behind a trophy, not priced: %+v", it.id, it)
		}
		if policeNeed(it.id) == 0 {
			t.Fatalf("%s has no kills milestone", it.id)
		}
	}
	if n != 4 {
		t.Fatalf("expected 4 unlockable tiers, got %d", n)
	}
	// the milestones go 500, 1000, 1500, 2000 in the order of the tiers
	for i := 1; i <= 4; i++ {
		if got := policeNeed("police:" + string(rune('0'+i))); got != 500*i {
			t.Fatalf("police:%d needs %d kills, want %d", i, got, 500*i)
		}
	}
	// exactly one trophy per police type and none of them is bought in the shop
	for _, it := range shopCatalog[shopPolice] {
		if it.trophy == "" {
			continue
		}
		g := freshShopGame()
		g.shopTab = shopPolice
		g.progress.Wallet = 1 << 30
		for i, x := range g.shopItems() {
			if x.id == it.id {
				g.shopSel = i
			}
		}
		g.shopActivate()
		if g.hasOwned(it.id) || g.progress.Wallet != 1<<30 {
			t.Fatalf("%s could be bought", it.id)
		}
	}
}

// reaching a milestone grants the tier, also retroactively for a save that already has the kills
func TestPoliceUnlockByKills(t *testing.T) {
	g := freshShopGame()
	g.progress.Stats["kills"] = 499
	g.trophyDirty = true
	g.checkTrophies()
	if g.hasOwned("police:1") || unitForStars(2) != unitPolice {
		t.Fatal("499 kills must not unlock the FBI")
	}
	g.progress.Stats["kills"] = 500
	g.trophyDirty = true
	g.checkTrophies()
	if !g.hasOwned("police:1") || unitForStars(2) != unitFBI {
		t.Fatal("500 kills unlock the FBI")
	}
	if g.hasOwned("police:2") {
		t.Fatal("the army needs 1000")
	}
	// a save with 2300 kills gets every tier at once
	h := freshShopGame()
	h.progress.Stats["kills"] = 2300
	h.trophyDirty = true
	h.checkTrophies()
	for i := 1; i <= 4; i++ {
		if !h.hasOwned("police:" + string(rune('0'+i))) {
			t.Fatalf("police:%d missing for 2300 kills", i)
		}
	}
	if unitForStars(5) != unitRangers {
		t.Fatal("rangers answer for 5 stars once unlocked")
	}
	shopOwned = map[string]bool{"skin:": true, "police:0": true}
}

// somebody who bought a tier with the old shop keeps it and does not get a duplicate entry
func TestPoliceOldPurchaseKept(t *testing.T) {
	g := freshShopGame()
	g.progress.Owned = []string{"police:1", "police:2"}
	g.syncShop()
	if unitForStars(3) != unitMilitary {
		t.Fatal("old purchases keep working")
	}
	g.progress.Stats["kills"] = 600
	g.trophyDirty = true
	g.checkTrophies()
	n := 0
	for _, o := range g.progress.Owned {
		if o == "police:1" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("police:1 stored %d times", n)
	}
	shopOwned = map[string]bool{"skin:": true, "police:0": true}
}
