package main

// ---------------------------------------------------------------------------
// Texts and per-party tables of ALCALÁ DE HENARES (alcala.go), merged from init() so that the shared tables (genres,
// kinds of party, hints, venue lines, newspaper, party events, menu colours, seats, trees) keep their own files
// untouched. The city and its places keep their real names (the same in ES and EN); no real person is named.
// ---------------------------------------------------------------------------

func init() {
	const title = alcalaTitle
	// music (genres.go): a Spanish town fiesta, so the pasodoble; its neighbours (TORRE EIFFEL: classical, CRUCERO
	// NOCTURNO: disco) differ. A user mp3 named "7_..." plays here (musicplan.go: the campaign index + 1).
	genreOfTitle[title] = "flamenco"
	partyTypeOfTitle[title] = "Fiesta mayor con cigüeñas"
	hintThemes["alcala"] = hintTheme{"Alcalá: las cigüeñas lo ven todo y no chivan", "El templete es la pista: el DJ manda ahí"}
	venueLines["alcala"] = []venueLine{
		{"¡Qué fiesta mayor más maja!", "What a lovely town fiesta!"},
		{"Esa cigüeña lleva aquí más que el alcalde", "That stork has been here longer than the mayor"},
		{"Me he perdido entre tanto soportal", "I got lost among all these arcades"},
		{"Pides una caña y te ponen tapa", "Order a beer and you get a free tapa"},
		{"Las cigüeñas castañean más que yo", "The storks clatter more than I do"},
		{"¿El de la estatua también baila?", "Does the statue guy dance too?"},
		{"Aquí hasta las palomas tienen carrera", "Even the pigeons here have a degree"},
	}
	pVenue["alcala"] = []paperTxt{
		tx("Las cigüeñas lo vieron todo desde el campanario. No quieren hablar.", "The storks saw it all from the belfry. They refuse to talk."),
		tx("La fiesta mayor de la plaza terminó antes que el pasodoble.", "The square's town fiesta ended before the pasodoble did."),
		tx("El templete se quedó sin público. Y sin músico.", "The bandstand ran out of audience. And of musicians."),
		tx("En los soportales sobraron tapas y faltaron clientes.", "Under the arcades the tapas were left over and the customers ran out."),
		tx("Las estatuas del banco fueron las únicas que no se movieron.", "The statues on the bench were the only ones that never moved."),
	}
	ptPlaces = append(ptPlaces,
		ptNoun{es: "plaza", fem: true, en: "square", theme: "alcala"},
		ptNoun{es: "templete", en: "bandstand", theme: "alcala"},
		ptNoun{es: "soportal", en: "arcade", theme: "alcala"},
	)
	// outdoors (partyevents.go: not in partyIndoorIcons); the street acts that suit a town square
	for _, m := range []map[string]bool{buskerIcons, jugglerIcons, mimeIcons, grannyIcons, tvIcons, congaIcons, bachelorIcons} {
		m["alcala"] = true
	}
	levelIconColors["alcala"] = 0xa8503c      // the red of the roofs: the menu swatch of the party
	genLabels["alcala"] = "Alcalá de Henares" // the editor's name of the generator
	seatThemes["alcala"] = seatTheme{1, [kitN]int{3, 1, 2, 0, 0, 0}, false}
	treeThemeIcons["alcala"] = treeThemeEurope

	for k, v := range map[string]string{
		title: title, // a proper name: the same in English
		"Cigüeñas en los tejados y fiesta mayor en la plaza.": "Storks on the roofs and a town fiesta in the square.",
		"Fiesta mayor con cigüeñas":                           "Town fiesta with storks",
		"Alcalá: las cigüeñas lo ven todo y no chivan":        "Alcalá: the storks see it all and never tell",
		"El templete es la pista: el DJ manda ahí":            "The bandstand is the dance floor: the DJ rules",
		"Alcalá de Henares":                                   "Alcalá de Henares",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, l := range venueLines["alcala"] {
		if _, ok := langDataEN[l.es]; !ok {
			langDataEN[l.es] = l.en
		}
	}
	for _, p := range pVenue["alcala"] {
		if _, ok := langDataEN[p.es]; !ok {
			langDataEN[p.es] = p.en
		}
	}
}
