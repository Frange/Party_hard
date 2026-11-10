package main

import "testing"

// The economy (shopprice.go): buying the whole shop takes about 50 normal parties, not 4.
func TestShopEconomyFiftyParties(t *testing.T) {
	trophyFrozen(t)
	n, total := shopTotal()
	parties := shopPartiesToBuyAll()
	t.Logf("shop: %d articles, %d points; a normal party banks %d × %.2f => %.1f parties to buy everything",
		n, total, shopPartyPoints, shopMoneyAvg, parties)
	if parties < 50 || parties > 65 {
		t.Fatalf("%.1f parties to buy everything (want 50-65)", parties)
	}
	// even without the golden wallet the most a single best party (98,950 in a real save) buys is a small part
	if total < 4*98950 {
		t.Fatalf("four record parties buy the whole shop (%d)", total)
	}
	// the killers' weapons are the dear list: dearer on average than clothes and accessories
	avg := func(tab int) int {
		s, k := 0, 0
		for _, it := range shopCatalog[tab] {
			if shopBuyable(it) {
				s += it.price
				k++
			}
		}
		return s / maxI(1, k)
	}
	if avg(shopWeapons) <= avg(shopOutfits) || avg(shopWeapons) <= avg(shopAccess) {
		t.Errorf("weapons %d, clothes %d, accessories %d", avg(shopWeapons), avg(shopOutfits), avg(shopAccess))
	}
	if c := len(shopCatalog[shopWeapons]); c < 32 {
		t.Errorf("only %d weapons", c)
	}
}

// the new weapons reuse art that exists (a Hold the characters already draw) and have an English name
func TestShopNewWeaponsHaveArtAndEnglish(t *testing.T) {
	trophyFrozen(t)
	seen := map[string]bool{}
	for _, it := range shopCatalog[shopWeapons] {
		if !shopBuyable(it) {
			continue
		}
		if seen[it.name] {
			t.Errorf("weapon name twice: %s", it.name)
		}
		seen[it.name] = true
		h, ok := weaponHold[it.id]
		if !ok || h <= HoldNone || h == HoldPee {
			t.Errorf("%s: no art (%d)", it.id, h)
		}
		if wildEN(it.name) == it.name && it.name != "Machete" { // the same word in English
			t.Errorf("%s: no English name", it.name)
		}
	}
}

// VESTUARIO is the last group of the shop, drawn further right than the regular spacing, and the click lands on it
func TestShopWardrobeLastAndApart(t *testing.T) {
	last := len(shopGroups) - 1
	if shopGroups[last].tabs[0] != shopWardrobe || shopFlat[len(shopFlat)-1] != shopWardrobe {
		t.Fatal("VESTUARIO must be the last group and the last stop of left/right")
	}
	for gi := 1; gi < last; gi++ {
		if shopGroupX(gi)-shopGroupX(gi-1) != 104 {
			t.Fatalf("regular groups are not evenly spaced")
		}
	}
	if gap := shopGroupX(last) - shopGroupX(last-1); gap < 104+40 {
		t.Fatalf("VESTUARIO is not apart (%v)", gap)
	}
	if shopGroupX(last)+shopGroupW > 925 {
		t.Fatalf("VESTUARIO leaves the panel")
	}
	// left/right from the weapons goes to the wardrobe and wraps to the first list
	g := testGame()
	g.shopTab = shopWeapons
	g.shopGo(g.shopPos() + 1)
	if g.shopTab != shopWardrobe {
		t.Fatalf("right from ARMAS lands on %s", shopTabNames[g.shopTab])
	}
	g.shopGo(g.shopPos() + 1)
	if g.shopTab != shopFlat[0] {
		t.Fatalf("right from VESTUARIO lands on %s", shopTabNames[g.shopTab])
	}
}

// a click on each group's label opens that group; the gap before VESTUARIO opens nothing
func TestShopGroupClick(t *testing.T) {
	for gi := range shopGroups {
		if got := shopGroupAt(shopGroupX(gi) + 10); got != gi {
			t.Errorf("click on %s gives %d", shopGroups[gi].name, got)
		}
	}
	last := len(shopGroups) - 1
	if got := shopGroupAt(shopGroupX(last) - 20); got != -1 {
		t.Errorf("the gap before VESTUARIO opens group %d", got)
	}
}
