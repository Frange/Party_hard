package main

// English text of the v1.4.0 venue elements: the race cars of the CIRCUITO (racetrack.go) and the night of the living
// dead of the CEMENTERIO (graveyard.go). "Cazafantasmos" is a proper name: the same in both languages.
func init() {
	for es, en := range map[string]string{
		"¡COCHE!":      "CAR!",
		"CEREBROS...":  "BRAAAINS...",
		"ÑAM, CEREBRO": "YUM, BRAIN",
		"Un zombi":     "A zombie",
		"¡Los muertos salen de sus tumbas! Quieren cerebros": "The dead rise from their graves! They want brains",
		"¡Fantasmas en el cementerio! Todo el mundo grita":   "Ghosts in the graveyard! Everybody screams",
		"¡AAAAH!":       "AAAAH!",
		"¡UN FANTASMA!": "A GHOST!",
		"¡QUE ME COGE!": "IT'S GOT ME!",
		"¡Llegan los Cazafantasmos! A por los fantasmas": "The Cazafantasmos are here! Get those ghosts",
		"¡Fantasma atrapado!":                            "Ghost trapped!",
		"¡UY! ESE NO ERA":                                "OOPS! WRONG ONE",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
