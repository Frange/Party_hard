package main

import (
	"encoding/json"
	"testing"
)

func wardGame() *Game {
	g := freshShopGame()
	wardP, wardRow = 0, 0
	return g
}

func shirtOf(g *Game, player int) uint32 {
	s := g.playerSpecFor(player, "", "")
	return uint32(s.Shirt.R)<<16 | uint32(s.Shirt.G)<<8 | uint32(s.Shirt.B)
}

// player 1 is blue and player 2 is red while nobody has chosen anything
func TestWardrobeDefaultsBlueAndRed(t *testing.T) {
	g := wardGame()
	if got := shirtOf(g, 0); got != 0x2c3c78 {
		t.Fatalf("player 1 must keep the blue protagonist, got %06x", got)
	}
	if got := shirtOf(g, 1); got != 0xa82c2c {
		t.Fatalf("player 2 must be red, got %06x", got)
	}
	// and the two are really different characters once drawn
	a := genChar(g.playerSpecFor(0, "", ""), vFront, poseFor(AnimIdle, 0))
	b := genChar(g.playerSpecFor(1, "", ""), vFront, poseFor(AnimIdle, 0))
	same := true
	for i := range a.p {
		if a.p[i] != b.p[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("both players are drawn the same")
	}
}

// what a player wears is his own: player 2 never takes what player 1 puts on
func TestWardrobeEquipIsPerPlayer(t *testing.T) {
	g := wardGame()
	g.progress.Owned = append(g.progress.Owned, "hat:caproja")
	g.syncShop()
	g.setEquipOf(0, "hat", "hat:caproja")
	if g.equipOf(1, "hat") != "" {
		t.Fatal("player 2 got the hat of player 1")
	}
	if g.playerSpecFor(1, "", "").Hat != HatNone {
		t.Fatal("player 2 is wearing a hat he never chose")
	}
	if g.playerSpecFor(0, "", "").Hat == HatNone {
		t.Fatal("player 1 lost his hat")
	}
	g.setEquipOf(1, "tint", "verde")
	if g.equipOf(0, "tint") != "" || shirtOf(g, 0) != 0x2c3c78 {
		t.Fatal("the tint of player 2 changed player 1")
	}
	if shirtOf(g, 1) != 0x2a7a42 {
		t.Fatalf("player 2 tint not applied: %06x", shirtOf(g, 1))
	}
}

// an old save has no second player: it loads, player 1 is untouched and player 2 is red
func TestWardrobeOldSaveStillLoads(t *testing.T) {
	old := `{"unlocked":3,"stats":{"kills":4},"trophies":{},"wallet":50,"owned":["hat:caproja"],"equip":{"hat":"hat:caproja"}}`
	var p Progress
	if err := json.Unmarshal([]byte(old), &p); err != nil {
		t.Fatal(err)
	}
	g := freshShopGame()
	g.progress = p
	g.syncShop()
	if g.equipOf(0, "hat") != "hat:caproja" {
		t.Fatal("player 1 lost what he wore")
	}
	if g.progress.EquipP2 != nil || shirtOf(g, 1) != 0xa82c2c {
		t.Fatal("an old save must give a red player 2")
	}
	// the copy of the progress (debug mode) keeps both players apart
	g.setEquipOf(1, "tint", "rosa")
	c := cloneProgress(g.progress)
	c.EquipP2["tint"] = "negro"
	if g.progress.EquipP2["tint"] != "rosa" {
		t.Fatal("cloneProgress shares the map of player 2")
	}
	// nothing is written when player 2 is never used
	data, _ := json.Marshal(Progress{Unlocked: 1})
	if string(data) != "" && json.Valid(data) && contains(string(data), "equipP2") {
		t.Fatal("equipP2 must be omitted while empty")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// the wardrobe never buys: it only offers what is owned and does not touch the wallet
func TestWardrobeDoesNotBuy(t *testing.T) {
	g := wardGame()
	g.progress.Wallet = 777
	before := len(g.progress.Owned)
	for _, r := range wardRows {
		for _, o := range g.wardOptions(r.slot) {
			if o.id == "" || r.slot == "tint" {
				continue
			}
			if r.slot == "base" {
				if !ownedID("skin:" + o.id) {
					t.Fatalf("%s is offered but not owned", o.id)
				}
				continue
			}
			found := false
			for _, tab := range []int{shopOutfits, shopAccess} {
				for _, it := range shopCatalog[tab] {
					if it.id == o.id && g.shopOwns(it) {
						found = true
					}
				}
			}
			if !found {
				t.Fatalf("%s (%s) is offered but not owned", o.id, r.slot)
			}
		}
		for i := 0; i < 6; i++ {
			g.wardCycle(0, r.slot, 1)
		}
	}
	if g.progress.Wallet != 777 || len(g.progress.Owned) != before {
		t.Fatal("the wardrobe bought something")
	}
	// the free base outfits are there from the start
	have := map[string]bool{}
	for _, o := range g.wardOptions("outfit") {
		have[o.id] = true
	}
	if !have["out:basea"] || !have["out:baser"] {
		t.Fatal("the free blue and red base outfits are missing")
	}
}

// buying keeps what the player already wears (it is the wardrobe that dresses)
func TestShopBuyDoesNotReplaceWhatIsWorn(t *testing.T) {
	g := wardGame()
	g.progress.Wallet = 1 << 20
	g.progress.Owned = append(g.progress.Owned, "hat:caproja")
	g.syncShop()
	g.setEquipOf(0, "hat", "hat:caproja")
	g.shopTab = shopAccess
	for i, it := range g.shopItems() {
		if it.slot == "hat" && it.id != "hat:caproja" && it.price > 0 {
			g.shopSel = i
			break
		}
	}
	bought := g.shopItems()[g.shopSel].id
	g.shopActivate()
	if !g.hasOwned(bought) {
		t.Fatal("the hat was not bought")
	}
	if g.equipOf(0, "hat") != "hat:caproja" {
		t.Fatal("buying replaced the hat that was worn")
	}
	// the same purchase for an empty slot of player 2 dresses him
	wardP = 1
	g.progress.Owned = append(g.progress.Owned, "x")
	g.shopSel = 0
	for i, it := range g.shopItems() {
		if it.slot == "glasses" && !g.hasOwned(it.id) {
			g.shopSel = i
			break
		}
	}
	gl := g.shopItems()[g.shopSel].id
	g.shopActivate()
	if g.equipOf(1, "glasses") != gl || g.equipOf(0, "glasses") != "" {
		t.Fatal("what is bought for an empty slot goes to the chosen player only")
	}
	wardP = 0
}

// both players get their own animated character, and the base look of player 2 survives the shop
func TestWardrobeCharacterSets(t *testing.T) {
	g := wardGame()
	a, b := g.playerCS(0), g.playerCS(1)
	if a == nil || b == nil || a == b {
		t.Fatal("each player needs his own character set")
	}
	if g.playerCS(1) != b {
		t.Fatal("the character set must be cached until the look changes")
	}
	g.setEquipOf(1, "tint", "amarillo")
	if g.playerCS(1) == b {
		t.Fatal("changing the look must rebuild the character")
	}
	g.applyShop()
	if g.as.playerCS == nil {
		t.Fatal("player 1 has no character")
	}
}
