package main

import "math"

// Shop prices: every price written in shopdata.go is the "list price"; the one place that turns it into the real
// price is shopRescalePrices (called once at the end of buildShopCatalog). Changing shopPriceFactor re-prices the
// whole shop. Prices only apply to what is still to be bought: whatever a save already owns (Progress.Owned) stays
// owned and the wallet is never touched.
//
// THE ECONOMY (2026-11-09: "much more expensive, buying everything must take ~50 normal parties"):
//
//	points of a normal party        shopPartyPoints = 8000 (a real save: 323,734 points in 41 parties ≈ 7,900)
//	what reaches the wallet         points × Cartera dorada (1 → 1.1 → 1.25 → 1.5); bought early, so on average
//	                                shopMoneyAvg = 1.25 over the whole journey  => ~10,000 points per party
//	list prices (shopdata.go)       clothes 23.3k + accessories 24.5k + perks 34.8k + animals 13.1k + weapons 32.4k
//	                                (the weapons count ×shopWeaponFactor: "more and more expensive")
//	real total = list × factor      ≈ 147k × 3.5 ≈ 515,000 points for the 157 articles  => ≈ 51 parties
//
// TestShopEconomyFiftyParties (shopprice_test.go) recomputes the total and the parties from the real catalogue.
const (
	shopPriceFactor  = 3.5
	shopWeaponFactor = 1.6 // the killers' weapons are dearer than the rest
	shopPartyPoints  = 8000
	shopMoneyAvg     = 1.25
)

// shopPartiesToBuyAll: how many normal parties pay for the whole shop (total / (points per party × wallet bonus)).
func shopPartiesToBuyAll() float64 {
	_, total := shopTotal()
	return float64(total) / (shopPartyPoints * shopMoneyAvg)
}

// shopScalePrice: scale and round to clean figures (multiples of 5 below 100, of 10 below 1000, of 50 below 10000,
// of 100 from there), never below 5. It is monotonic, so the relative order of two items never inverts.
func shopScalePrice(p int) int {
	v := float64(p) * shopPriceFactor
	step := 100.0
	switch {
	case v < 100:
		step = 5
	case v < 1000:
		step = 10
	case v < 10000:
		step = 50
	}
	r := int(math.Round(v/step) * step)
	if r < 5 {
		r = 5
	}
	return r
}

// shopRescalePrices applies shopScalePrice to every item paid with points (trophy locks and free items keep theirs).
func shopRescalePrices() {
	for t := range shopCatalog {
		for i := range shopCatalog[t] {
			if shopBuyable(shopCatalog[t][i]) {
				p := shopCatalog[t][i].price
				if t == shopWeapons {
					p = int(math.Round(float64(p) * shopWeaponFactor))
				}
				shopCatalog[t][i].price = shopScalePrice(p)
			}
		}
	}
}
