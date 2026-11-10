package main

// Three more secret trophies (same pattern as album.go: ordinary entries of trophyList listed in secretTrophy, so they
// never gate the last trophy). Their counters live in progress.Stats (omitempty maps: old saves are untouched).
//
//	s_lastone  die when only ONE guest was missing to finish the party   (stat "lastone", set in die())
//	s_noknife  complete a party without a single knife kill              (stat "noknife", set in completeLevel())
//	s_shopall  own everything the shop sells                             (checked in checkTrophies, shopAllBought)
//	s_daily    complete RETO DE HOY once                                  (stat "daily", set in dailyWin, daily.go)
func init() {
	sec := []Trophy{
		{"s_lastone", "Tan cerca, tan muerto", "Muere cuando solo te faltaba 1 invitado para acabar la fiesta.", "lastone", 1, evr("secret:lastone", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_noknife", "Yo no he sido", "Completa una fiesta sin matar con el cuchillo (veneno y trampas no cuentan).", "noknife", 1, evr("secret:noknife", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_shopall", "Liquidación total", "Compra todo lo que vende la tienda.", "shopall", 1, evr("secret:shopall", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_daily", "Deberes hechos, funeral incluido", "Completa el RETO DE HOY.", "daily", 1, evr("secret:daily", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
	}
	at := len(trophyList)
	for i, t := range trophyList {
		if t.id == legendTrophy {
			at = i
		}
	}
	rest := append([]Trophy(nil), trophyList[at:]...)
	trophyList = append(append(trophyList[:at:at], sec...), rest...)
	for _, t := range sec {
		secretTrophy[t.id] = true
	}
	for k, v := range map[string]string{
		"Tan cerca, tan muerto": "So close, so dead",
		"Muere cuando solo te faltaba 1 invitado para acabar la fiesta.": "Die when only 1 guest was left to finish the party.",
		"Yo no he sido": "It wasn't me",
		"Completa una fiesta sin matar con el cuchillo (veneno y trampas no cuentan).": "Complete a party without a single knife kill (poison and traps don't count).",
		"Liquidación total":                   "Clearance sale",
		"Compra todo lo que vende la tienda.": "Buy everything the shop sells.",
		"Deberes hechos, funeral incluido":    "Homework done, funeral included",
		"Completa el RETO DE HOY.":            "Complete TODAY'S CHALLENGE.",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

// shopBuyable: an item that is paid for with points (not the free ones, not those won with a trophy).
func shopBuyable(it shopItem) bool { return it.trophy == "" && it.price > 0 && it.price < 1<<20 }

// shopTotal: how many items the shop sells and what they cost all together.
func shopTotal() (n, points int) {
	for _, list := range shopCatalog {
		for _, it := range list {
			if shopBuyable(it) {
				n++
				points += it.price
			}
		}
	}
	return
}

// shopAllBought: every buyable item is owned.
func shopAllBought() bool {
	n := 0
	for _, list := range shopCatalog {
		for _, it := range list {
			if shopBuyable(it) {
				n++
				if !ownedID(it.id) {
					return false
				}
			}
		}
	}
	return n > 0
}
