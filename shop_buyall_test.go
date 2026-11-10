package main

import (
	"os"
	"sort"
	"testing"
)

// shopBuyGame: a shop game with a frozen progress; the global lookup of purchases is restored afterwards.
func shopBuyGame(t *testing.T) *Game {
	t.Helper()
	oldF, oldO, oldC := progressFrozen, shopOwned, castSlugs
	t.Cleanup(func() { progressFrozen, shopOwned, castSlugs = oldF, oldO, oldC })
	g := freshShopGame()
	g.shopTab = shopAccess
	return g
}

func buyablePrices(g *Game) []int {
	var out []int
	for _, it := range shopCatalog[g.shopTab] {
		if shopBuyable(it) && !g.shopOwns(it) {
			out = append(out, it.price)
		}
	}
	sort.Ints(out)
	return out
}

func TestShopBuyAllTakesTheMaxCheapestFirst(t *testing.T) {
	g := shopBuyGame(t)
	pr := buyablePrices(g)
	if len(pr) < 5 {
		t.Fatalf("too few items to test: %d", len(pr))
	}
	g.progress.Wallet = pr[0] + pr[1] + pr[2] + pr[3] - 1 // one point short of the fourth
	w0 := g.progress.Wallet
	items, cost := g.shopBuyAllPlan()
	if len(items) != 3 || cost != pr[0]+pr[1]+pr[2] {
		t.Fatalf("plan = %d items, %d points; want 3 and %d", len(items), cost, pr[0]+pr[1]+pr[2])
	}
	g.shopBuyAllAsk()
	if !g.shopBuyOpen || g.shopBuyN != 3 || g.shopBuyCost != cost {
		t.Fatalf("popup: open=%v n=%d cost=%d", g.shopBuyOpen, g.shopBuyN, g.shopBuyCost)
	}
	if g.progress.Wallet != w0 {
		t.Fatal("asking must not spend anything")
	}
	if n := g.shopBuyAllDo(); n != 3 {
		t.Fatalf("bought %d, want 3", n)
	}
	if g.shopBuyOpen || g.progress.Wallet != w0-cost || g.progress.Wallet < 0 {
		t.Fatalf("wallet %d, want %d", g.progress.Wallet, w0-cost)
	}
	for _, it := range items {
		if !g.shopOwns(it) {
			t.Errorf("%s not owned", it.id)
		}
	}
	if got := len(buyablePrices(g)); got != len(pr)-3 {
		t.Errorf("left %d items, want %d", got, len(pr)-3)
	}
}

func TestShopBuyAllNeverGoesNegative(t *testing.T) {
	for _, tab := range []int{shopOutfits, shopAccess, shopPerks, shopChars} {
		for _, w := range []int{0, 4, 5, 40, 333, 2500, 40000, 1 << 30} {
			g := shopBuyGame(t)
			g.shopTab = tab
			g.progress.Wallet = w
			total := 0
			for _, it := range shopCatalog[tab] {
				if shopBuyable(it) && !g.shopOwns(it) {
					total += it.price
				}
			}
			g.shopBuyAllAsk()
			g.shopBuyAllDo()
			if g.progress.Wallet < 0 {
				t.Fatalf("tab %d wallet %d -> %d", tab, w, g.progress.Wallet)
			}
			spent := w - g.progress.Wallet
			if spent > w || spent > total {
				t.Fatalf("tab %d spent %d of %d (list %d)", tab, spent, w, total)
			}
			if w >= total && len(buyablePrices(g)) != 0 {
				t.Fatalf("tab %d: enough points but %d items left", tab, len(buyablePrices(g)))
			}
			if again, _ := g.shopBuyAllPlan(); w < total && len(again) > 0 && g.progress.Wallet >= buyablePrices(g)[0] {
				t.Fatalf("tab %d: it left something affordable", tab)
			}
		}
	}
}

func TestShopBuyAllNothingToBuy(t *testing.T) {
	g := shopBuyGame(t)
	g.progress.Wallet = 0
	g.shopBuyAllAsk()
	if g.shopBuyOpen || g.shopMsg == "" {
		t.Fatal("no points: no popup, but a message")
	}
	// trophy-locked and free items are never bought, whatever the wallet
	g = shopBuyGame(t)
	g.shopTab = shopPolice
	g.progress.Wallet = 1 << 30
	g.shopBuyAllAsk()
	if g.shopBuyOpen || len(g.progress.Owned) != 0 || g.progress.Wallet != 1<<30 {
		t.Fatal("the police tiers are locked by kills")
	}
}

func TestShopBuyAllNoPromptNoPurchaseOnNo(t *testing.T) {
	g := shopBuyGame(t)
	g.progress.Wallet = 1 << 30
	g.shopBuyAllAsk()
	if !g.shopBuyOpen {
		t.Fatal("popup expected")
	}
	g.shopBuyOpen = false // "No"
	if g.progress.Wallet != 1<<30 || len(g.progress.Owned) != 0 {
		t.Fatal("answering No must not buy")
	}
}

func TestShopBuyAllNeverWritesProgress(t *testing.T) {
	g := shopBuyGame(t)
	path := progressPath()
	before, _ := os.ReadFile(path)
	g.progress.Wallet = 1 << 30
	g.shopBuyAllAsk()
	g.shopBuyAllDo()
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("progress.json changed in a test")
	}
}

func TestShopUnboughtFirst(t *testing.T) {
	g := shopBuyGame(t)
	g.progress.Wallet = 1 << 30
	check := func() {
		t.Helper()
		items := g.shopItems()
		if len(items) != len(shopCatalog[g.shopTab]) {
			t.Fatalf("the view has %d items, the catalog %d", len(items), len(shopCatalog[g.shopTab]))
		}
		seenOwned, lastP := false, -1
		for _, it := range items {
			own := g.shopOwns(it)
			if own && !seenOwned {
				seenOwned, lastP = true, -1
			}
			if !own && seenOwned {
				t.Fatalf("%s is not bought and comes after bought ones", it.id)
			}
			if it.price < lastP {
				t.Fatalf("%s breaks the price order inside its group", it.id)
			}
			lastP = it.price
		}
	}
	check()
	first := g.shopItems()[0]
	g.shopSel = 0
	g.shopActivate()
	if !g.shopOwns(first) {
		t.Fatal("the first item was not bought")
	}
	check()
	items := g.shopItems()
	if items[len(items)-1].id != first.id && !g.shopOwns(items[len(items)-1]) {
		t.Fatal("what was just bought must go to the end")
	}
	if g.shopItems()[g.shopSel].id == first.id {
		t.Fatal("the cursor must stay on the row, now showing the next one not bought")
	}
	g.shopBuyAllAsk()
	g.shopBuyAllDo()
	check()
	// every tab keeps the rule
	for _, tab := range shopFlat {
		if tab == shopWardrobe {
			continue
		}
		g.shopTab = tab
		check()
	}
}
