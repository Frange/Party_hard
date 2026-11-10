package main

// English texts of the BUY ALL button of the shop (shop.go). Merged without overwriting anything.
func init() {
	for k, v := range map[string]string{
		"COMPRAR TODO":           "BUY ALL",
		"¿COMPRAR TODO?":         "BUY EVERYTHING?",
		"Artículos: %d":          "Items: %d",
		"Coste total: %d puntos": "Total cost: %d points",
		"Te quedarán: %d puntos": "You will have left: %d points",
		"SÍ":                     "YES",
		"NO":                     "NO",
		"¡Comprados %d artículos por %d puntos!":           "Bought %d items for %d points!",
		"No tienes puntos para comprar nada de esta lista": "Not enough points to buy anything from this list",
		"No hay nada que comprar en esta lista":            "Nothing to buy in this list",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
