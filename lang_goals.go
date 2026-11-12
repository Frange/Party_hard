package main

// English texts of OBJETIVOS (GOALS): the tabs, the hints of the locked cards and the names of the killers. Merged into
// langDataEN without overwriting anything that already exists.
func init() {
	for k, v := range map[string]string{
		"OBJETIVOS":  "GOALS",
		"PERSONAJES": "CHARACTERS",
		"ASESINOS":   "KILLERS",
		"VIPS":       "VIPS",
		"TROFEOS":    "TROPHIES",
		"TROFEO":     "TROPHY",
		"CONSEGUIDO": "GOT IT",

		"Mata a un invitado de este tipo":           "Kill a guest of this kind",
		"Se gana con un trofeo (pestaña TROFEOS)":   "Won with a trophy (TROPHIES tab)",
		"Derrota a este asesino especial":           "Beat this special killer",
		"Visto: mátalo para conseguirlo":            "Seen: kill him to unlock him",
		"Premio de trofeo":                          "Trophy prize",
		"Flechas mover   Tab: otra pestaña   ":      "Arrows move   Tab: next tab   ",
		"↑/↓ o rueda elegir   Tab: otra pestaña   ": "Up/down or wheel choose   Tab: next tab   ",
		"Cantante":   "Singer",
		"Guitarra":   "Guitar",
		"Bajo":       "Bass",
		"Batería":    "Drums",
		"Caperucita": "Caperucita",
		"Bailarina":  "Dancer",
		"Rockero":    "Rocker",
		"Rey Fiesta": "Party King",
		"Colón":      "Colón",
		"Ysabel":     "Ysabel",
		"Motosierra": "Chainsaw",
		"Garras":     "Claws",
		"Machete":    "Machete",
		"Antorcha":   "Torch",
		"Maletín":    "Briefcase",
		"Espada":     "Sword",
		"Cuchillo":   "Knife",

		// trophies that had no English text (they only showed when the list was scrolled to them)
		"Llamada al FBI": "Call the FBI",
		"Mata a 500 personas en total: llega el FBI.": "Kill 500 people in total: the FBI arrives.",
		"Policía: FBI": "Police: FBI",
		"Molder y Skully acuden con 2 estrellas.":                              "Two agents show up at 2 stars.",
		"El ejército toma la calle":                                            "The army takes the street",
		"Mata a 1000 personas en total: llegan los militares.":                 "Kill 1000 people in total: the military arrives.",
		"Policía: Ejército":                                                    "Police: Army",
		"Soldados con casco y fusiles, con 3 estrellas.":                       "Soldiers with helmets and rifles, at 3 stars.",
		"Operación SWAT":                                                       "Operation SWAT",
		"Mata a 1500 personas en total: llega el SWAT.":                        "Kill 1500 people in total: the SWAT team arrives.",
		"Policía: SWAT":                                                        "Police: SWAT",
		"Operadores que entran por la ventana, con 4 estrellas.":               "Operators who come in through the window, at 4 stars.",
		"Rangers al rescate":                                                   "Rangers to the rescue",
		"Mata a 2000 personas en total: llegan los Pawer Rangers.":             "Kill 2000 people in total: the Pawer Rangers arrive.",
		"Policía: Pawer Rangers":                                               "Police: Pawer Rangers",
		"Cinco colores y una coreografía, con 5 estrellas.":                    "Five colours and a choreography, at 5 stars.",
		"Equipo médico":                                                        "Medical team",
		"Revive 3 veces a tu compañero con la ambulancia.":                     "Revive your partner 3 times with the ambulance.",
		"Bata de médico":                                                       "Doctor's coat",
		"Blanca y con estetoscopio imaginario. Confía en mí, soy casi médico.": "White, with an imaginary stethoscope. Trust me, I'm almost a doctor.",
		"Bombero honorario":                                                    "Honorary firefighter",
		"Mata a un pirómano.":                                                  "Kill an arsonist.",
		"Casco de bombero honorario":                                           "Honorary firefighter helmet",
		"Rojo brillante. Apaga lo que quieras, menos tu ego.":                  "Bright red. Put out whatever you like, except your ego.",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, k := range killerCatalog {
		goalName(k)
	}
	for _, k := range killerTrophy {
		goalName(k.killerInfo)
	}
}

func goalName(k killerInfo) {
	if _, ok := langDataEN[k.es]; !ok {
		langDataEN[k.es] = k.en
	}
	if k.cardES != "" {
		if _, ok := langDataEN[k.cardES]; !ok {
			langDataEN[k.cardES] = k.cardEN
		}
	}
}
