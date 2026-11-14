package main

// ---------------------------------------------------------------------------
// Texts and per-party tables of BARCELONA (barcelona.go), merged from init() like ALCALÁ and GUADALAJARA. The party
// keeps the internal icon "ranch" of the party it replaced (saves, medals, editor key), so every table keyed by the
// icon uses "ranch" and gets BARCELONA's content here (the old ranch entries were removed from venuelines.go,
// levelhint.go, seats.go and the newspaper nouns). The city and its places keep their real names (the same in ES and
// EN); no real person is named.
// ---------------------------------------------------------------------------

// bcnIcon: the internal icon (Def.Icon) of BARCELONA.
const bcnIcon = "ranch"

func init() {
	const title = barcelonaTitle
	// music (genres.go): a Latin beat close to the Catalan rumba; its neighbours (GLOW NEON: synthwave, PLAZA DE TOROS:
	// pasodoble) differ. A user mp3 named "4_..." plays here (musicplan.go: the campaign index + 1).
	genreOfTitle[title] = "reggaeton"
	partyTypeOfTitle[title] = "Rumba en la plaza"
	hintThemes[bcnIcon] = hintTheme{"Barcelona: el mosaico es la pista y el DJ manda", "Las terrazas esconden; las fuentes, no"}
	venueLines[bcnIcon] = []venueLine{
		{"¡Esto es Barcelona, nen!", "This is Barcelona, mate!"},
		{"He venido por la rumba y el pan con tomate", "I came for the rumba and the tomato bread"},
		{"La Sagrada Família sigue sin acabar", "The Sagrada Família is still not finished"},
		{"Cuidado con el bolso en la plaza", "Mind your bag in the square"},
		{"Este mosaico baila mejor que yo", "This mosaic dances better than me"},
		{"¿Playa o fiesta? ¡Las dos!", "Beach or party? Both!"},
		{"Las banderas ondean más que mi camisa", "The flags wave more than my shirt"},
		{"Subo al teleférico y vuelvo, ¿eh?", "I will take the cable car and be back, ok?"},
	}
	pVenue[bcnIcon] = []paperTxt{
		tx("La rumba de la plaza acabó antes que las obras de la basílica.", "The square's rumba ended before the basilica's works did."),
		tx("El teleférico siguió subiendo y bajando. Nadie quiso bajarse.", "The cable car kept going up and down. Nobody wanted to get off."),
		tx("Las fuentes de la plaza siguieron bailando solas.", "The fountains of the square went on dancing alone."),
		tx("El quiosco azul vendió todos los periódicos de la mañana.", "The blue kiosk sold every single morning paper."),
		tx("Las banderas siguieron ondeando sobre una plaza vacía.", "The flags kept waving over an empty square."),
	}
	ptPlaces = append(ptPlaces,
		ptNoun{es: "mosaico", en: "mosaic", theme: bcnIcon},
		ptNoun{es: "teleférico", en: "cable car", theme: bcnIcon},
		ptNoun{es: "quiosco", en: "kiosk", theme: bcnIcon},
	)
	// outdoors (partyevents.go: not in partyIndoorIcons); the street acts that suit a city square (the ranch's barn
	// acts, bull, poker, painter, torch patrol and the queue for the toilets, were taken out of their tables)
	for _, m := range []map[string]bool{buskerIcons, jugglerIcons, mimeIcons, tvIcons, bachelorIcons, statueIcons} {
		m[bcnIcon] = true
	}
	levelIconColors[bcnIcon] = 0x3a6ac8  // the blue of the sky and the sea: the menu swatch of the party
	genLabels["barcelona"] = "Barcelona" // the editor's name of the generator
	seatThemes[bcnIcon] = seatTheme{1, [kitN]int{3, 1, 2, 0, 0, 0}, false}
	treeThemeIcons[bcnIcon] = treeThemeEurope

	for k, v := range map[string]string{
		title: title, // a proper name: the same in English
		"Sol, mar y mosaico: la plaza baila a la catalana.": "Sun, sea and mosaic: the square dances Catalan style.",
		"Rumba en la plaza": "Square rumba",
		"Barcelona: el mosaico es la pista y el DJ manda": "Barcelona: the mosaic is the floor, the DJ rules",
		"Las terrazas esconden; las fuentes, no":          "The terraces hide you; the fountains do not",
		"Barcelona":                                       "Barcelona",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, l := range venueLines[bcnIcon] {
		if _, ok := langDataEN[l.es]; !ok {
			langDataEN[l.es] = l.en
		}
	}
	for _, p := range pVenue[bcnIcon] {
		if _, ok := langDataEN[p.es]; !ok {
			langDataEN[p.es] = p.en
		}
	}
}
