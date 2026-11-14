package main

// ---------------------------------------------------------------------------
// Texts and per-party tables of GUADALAJARA (guadalajara.go), merged from init() so that the shared tables (genres,
// kinds of party, hints, venue lines, newspaper, party events, menu colours, seats, trees) keep their own files
// untouched. The city, its places and its shops keep their real names (the same in ES and EN: "Pastelería Mariam",
// "Miel de la Alcarria"); no real person is named.
// ---------------------------------------------------------------------------

func init() {
	const title = guadalajaraTitle
	// music (genres.go): a Spanish city fiesta, so the pasodoble; its neighbours (CEMENTERIO: metal, CIRCUITO:
	// electronic) differ. A user mp3 named "16_..." plays here (musicplan.go: the campaign index + 1).
	genreOfTitle[title] = "flamenco"
	partyTypeOfTitle[title] = "Verbena con miel y templete"
	hintThemes["guadalajara"] = hintTheme{"Guadalajara: el toro del monte no chiva", "La fuente y el templete reúnen a la gente"}
	venueLines["guadalajara"] = []venueLine{
		{"¡Qué verbena más rica en la plaza!", "What a sweet street party in the square!"},
		{"He venido por la miel de la Alcarria", "I came for the honey of the Alcarria"},
		{"¿Alguien ha probado la Pastelería Mariam?", "Has anyone tried the Pastelería Mariam?"},
		{"El toro del monte no baila ni a la de tres", "The bull on the hill won't dance at all"},
		{"La fuente salpica más que yo bailando", "The fountain splashes more than my dancing"},
		{"Bajo la parra se está de lujo", "Under the vines it's simply perfect"},
		{"En el quiosco ya no queda ni un periódico", "The kiosk is out of newspapers"},
	}
	pVenue["guadalajara"] = []paperTxt{
		tx("El toro del monte lo vio todo. Sigue sin moverse.", "The bull on the hill saw it all. It still won't move."),
		tx("La fuente de la plaza corrió toda la noche. Los invitados, no.", "The fountain of the square ran all night. The guests did not."),
		tx("En el puesto de miel sobraron tarros y faltaron clientes.", "At the honey stall the jars were left over and the customers ran out."),
		tx("El templete se quedó sin público. Y sin DJ.", "The bandstand ran out of audience. And of DJ."),
		tx("La Pastelería Mariam abrió con más dulces que clientes.", "The Pastelería Mariam opened with more cakes than customers."),
	}
	ptPlaces = append(ptPlaces,
		ptNoun{es: "fuente", fem: true, en: "fountain", theme: "guadalajara"},
		ptNoun{es: "pérgola", fem: true, en: "pergola", theme: "guadalajara"},
	)
	// outdoors (partyevents.go: not in partyIndoorIcons); the street acts that suit a town square
	for _, m := range []map[string]bool{buskerIcons, jugglerIcons, mimeIcons, grannyIcons, tvIcons, congaIcons, bachelorIcons} {
		m["guadalajara"] = true
	}
	levelIconColors["guadalajara"] = 0x6a3a9a // the purple of the stadium and the kiosk: the menu swatch of the party
	genLabels["guadalajara"] = "Guadalajara"  // the editor's name of the generator
	seatThemes["guadalajara"] = seatTheme{1, [kitN]int{3, 1, 2, 0, 0, 0}, false}
	treeThemeIcons["guadalajara"] = treeThemeEurope

	for k, v := range map[string]string{
		title: title, // a proper name: the same in English
		"Fuente, templete y miel de la Alcarria: fiesta en la plaza.": "Fountain, bandstand and Alcarria honey: a party in the square.",
		"Verbena con miel y templete":                                 "Street party with honey and bandstand",
		"Guadalajara: el toro del monte no chiva":                     "Guadalajara: the bull on the hill never tells",
		"La fuente y el templete reúnen a la gente":                   "The fountain and the bandstand draw the crowd",
		"Guadalajara": "Guadalajara",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, l := range venueLines["guadalajara"] {
		if _, ok := langDataEN[l.es]; !ok {
			langDataEN[l.es] = l.en
		}
	}
	for _, p := range pVenue["guadalajara"] {
		if _, ok := langDataEN[p.es]; !ok {
			langDataEN[p.es] = p.en
		}
	}
}
