package main

// ---------------------------------------------------------------------------
// Texts and per-party tables of ITALIA (italy.go), merged from init() so that the shared tables (genres, kinds of
// party, hints, venue lines, party events, menu colours, ambient) keep their own files untouched. Generic parody only.
// ---------------------------------------------------------------------------

func init() {
	const title = "ITALIA"
	// music (genres.go): opera country, so classical; its neighbours (highway: rock, train station: rap) differ.
	// A user mp3 named "20_..." plays here (the number of the list); an mp3 of another style is ignored (musicplan.go).
	genreOfTitle[title] = "clasica"
	partyTypeOfTitle[title] = "Pasta, amore y caos"
	hintThemes["italy"] = hintTheme{"Italia: la torre se inclina, tú no", "El anfiteatro tiene dos entradas: úsalas"}
	venueLines["italy"] = []venueLine{
		{"¡Mamma mia, qué fiestón!", "Mamma mia, what a party!"},
		{"Esa torre se inclina más que mi tío", "That tower leans more than my uncle"},
		{"El gondolero cobra hasta por mirar", "The gondolier charges just for looking"},
		{"Pizza con piña aquí es delito", "Pineapple pizza is a crime here"},
		{"Mi helado se derrite antes que yo", "My gelato melts before I do"},
		{"El anfiteatro está igual que mi piso", "The amphitheatre looks just like my flat"},
	}
	pVenue["italy"] = []paperTxt{
		tx("La torre siguió inclinada. La fiesta, en cambio, se vino abajo.", "The tower kept leaning. The party, on the other hand, collapsed."),
		tx("Un gondolero jura que nadie subió a su barca. Ni bajó.", "A gondolier swears nobody got on his boat. Or off it."),
		tx("El anfiteatro vio más espectáculo que en dos mil años.", "The amphitheatre saw more show than in two thousand years."),
		tx("La pizzería cerró antes de hora: se quedó sin clientes.", "The pizzeria closed early: it ran out of customers."),
		tx("El heladero vendió de todo menos coartadas.", "The gelato man sold everything except alibis."),
	}
	// outdoors (partyevents.go: not in partyIndoorIcons); the street acts that suit a piazza
	for _, m := range []map[string]bool{buskerIcons, jugglerIcons, mimeIcons, grannyIcons, tvIcons, congaIcons, bachelorIcons} {
		m["italy"] = true
	}
	levelIconColors["italy"] = 0xd8583a // tomato red: the menu swatch of the party
	genLabels["italy"] = "Italia"       // the editor's name of the generator
	ambientPlan = append(ambientPlan, ambientEntry{title, bgLeaves | bgSway, "crowd"})

	for k, v := range map[string]string{
		title: "ITALY",
		"Pizza, góndolas y una torre que ya se cae sola.": "Pizza, gondolas and a tower that is already falling over.",
		"Pasta, amore y caos":                             "Pasta, amore and chaos",
		"Italia: la torre se inclina, tú no":              "Italy: the tower leans, you don't",
		"El anfiteatro tiene dos entradas: úsalas":        "The amphitheatre has two ways in: use them",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, l := range venueLines["italy"] {
		if _, ok := langDataEN[l.es]; !ok {
			langDataEN[l.es] = l.en
		}
	}
	for _, p := range pVenue["italy"] {
		if _, ok := langDataEN[p.es]; !ok {
			langDataEN[p.es] = p.en
		}
	}
}
