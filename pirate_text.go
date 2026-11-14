package main

// ---------------------------------------------------------------------------
// Texts and per-party tables of the BARCO PIRATA (pirate.go), merged from init() so that the shared tables (genres,
// kinds of party, hints, venue lines, newspaper, party events, menu colours, ambient) keep their own files untouched.
// Generic parody only: no film, no famous ship, no real person.
// ---------------------------------------------------------------------------

func init() {
	const title = pirateTitle
	// music (genres.go): a big adventure score; its neighbours (70s disco: disco, space station: electro) differ.
	// A user mp3 named "25_..." plays here (the number of the list); an mp3 of another style is ignored (musicplan.go).
	genreOfTitle[title] = "epico"
	partyTypeOfTitle[title] = "Abordaje con ron"
	hintThemes["pirate"] = hintTheme{"Barco pirata: que nadie quede a flote", "Barriles y cofres: buenos escondites"}
	venueLines["pirate"] = []venueLine{
		{"¡Al abordaje... de la barra!", "Board the... bar!"},
		{"Este ron sabe a madera y a remordimiento", "This rum tastes of wood and regret"},
		{"El loro sabe más secretos que yo", "The parrot knows more secrets than I do"},
		{"¿Quién ha dejado el cañón cargado?", "Who left the cannon loaded?"},
		{"Mi mapa del tesoro era un menú", "My treasure map was a menu"},
		{"Me mareo y aún no hemos zarpado", "I'm seasick and we haven't even sailed"},
	}
	pVenue["pirate"] = []paperTxt{
		tx("El barco seguía anclado. La fiesta, en cambio, se fue a pique.", "The ship stayed at anchor. The party, on the other hand, sank."),
		tx("El loro del capitán se niega a declarar sin su abogado.", "The captain's parrot refuses to talk without his lawyer."),
		tx("Faltaba ron en las bodegas y sobraban preguntas en el muelle.", "The hold ran out of rum and the dock was full of questions."),
		tx("Nadie caminó por la plancha: todos se fueron antes.", "Nobody walked the plank: everyone left before that."),
		tx("El cofre del tesoro apareció vacío. Los invitados, también.", "The treasure chest turned up empty. So did the guests."),
	}
	ptPlaces = append(ptPlaces,
		ptNoun{es: "cubierta", fem: true, en: "deck", theme: "pirate"},
		ptNoun{es: "bodega", fem: true, en: "hold", theme: "pirate"},
		ptNoun{es: "pasarela", fem: true, en: "gangplank", theme: "pirate"},
	)
	// outdoors (partyevents.go: not in partyIndoorIcons); the acts that suit a party on deck. The dog act stays
	// ashore like on the cruise (pe_dog.go).
	for _, m := range []map[string]bool{congaIcons, bachelorIcons, jugglerIcons, tvIcons, magicIcons} {
		m["pirate"] = true
	}
	dogNoIcons["pirate"] = true
	levelIconColors["pirate"] = 0x8a3a3a // rum red: the menu swatch of the party
	genLabels["pirate"] = "Barco pirata" // the editor's name of the generator
	// no entry in ambientPlan (it keeps 10-14 levels, ambient_test.go): since v1.4.0 the bay animates on its own
	// (pirate_sea.go: wave crests, a dolphin, a shark's fin);
	// the sky animates on its own (seasky.go: the ship on the horizon, the passing pirate ship)

	for k, v := range map[string]string{
		title: "PIRATE SHIP",
		"Ron, cañones y un capitán que no sabe nadar.": "Rum, cannons and a captain who can't swim.",
		"Abordaje con ron":                      "Boarding party with rum",
		"Barco pirata: que nadie quede a flote": "Pirate ship: let nobody stay afloat",
		"Barriles y cofres: buenos escondites":  "Barrels and chests: fine hiding spots",
		"Coger el tesoro":                       "Grab the treasure",
		"tesoro":                                "treasure",
		"¡Un tesoro! Doblones para el bolsillo": "Treasure! Doubloons for your pocket",
		"Barco pirata":                          "Pirate ship",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, l := range venueLines["pirate"] {
		if _, ok := langDataEN[l.es]; !ok {
			langDataEN[l.es] = l.en
		}
	}
	for _, p := range pVenue["pirate"] {
		if _, ok := langDataEN[p.es]; !ok {
			langDataEN[p.es] = p.en
		}
	}
}
