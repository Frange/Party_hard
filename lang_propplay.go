package main

// English texts of the guests playing the slot machines and the roulette (propplay.go). Merged into langDataEN
// without overwriting anything.
func init() {
	for k, v := range map[string]string{
		"¡Vamos, vamos!":       "Come on, come on!",
		"¡Tres cerezas!":       "Three cherries!",
		"Una más y lo dejo":    "One more and I quit",
		"¡PREMIO!":             "JACKPOT!",
		"Esta máquina me odia": "This machine hates me",
		"¡Todo al rojo!":       "All on red!",
		"¡Al negro!":           "On black!",
		"¡El 17, el 17!":       "17, come on, 17!",
		"¡He ganado!":          "I won!",
		"Hagan juego...":       "Place your bets...",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
