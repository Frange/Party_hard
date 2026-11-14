package main

// ---------------------------------------------------------------------------
// Texts of PARÍS (paris.go; it was TORRE EIFFEL). The tables keyed by the icon "eiffel" keep their entries (venue
// lines, hint, seats, costumes, party events); here only the new lines about the city's street life are added. The
// title is the name of the city ("PARIS" in English, its own name); no real person is named.
// ---------------------------------------------------------------------------

func init() {
	venueLines["eiffel"] = append(venueLines["eiffel"],
		venueLine{"Este acordeón me persigue desde el metro", "This accordion followed me from the metro"},
		venueLine{"El mimo no me contesta. Qué maleducado", "The mime will not answer me. So rude"},
		venueLine{"Una baguette bajo el brazo y a bailar", "A baguette under my arm and off I dance"},
		venueLine{"Me he mareado en el tiovivo", "The carousel made me dizzy"},
		venueLine{"He comprado un libro viejo en el muelle", "I bought an old book on the quay"},
	)
	pVenue["eiffel"] = append(pVenue["eiffel"],
		tx("El acordeonista siguió tocando cuando ya no quedaba nadie.", "The accordion player kept playing when nobody was left."),
		tx("El mimo lo vio todo. Como siempre, no dijo nada.", "The mime saw it all. As usual, he said nothing."),
		tx("La panadería de la plaza se quedó con todas las baguettes.", "The bakery on the square was left with all its baguettes."),
	)
	ptPlaces = append(ptPlaces,
		ptNoun{es: "boca de metro", fem: true, en: "metro entrance", theme: "eiffel"},
		ptNoun{es: "tiovivo", en: "carousel", theme: "eiffel"},
	)
	for k, v := range map[string]string{
		parisTitle: "PARIS", // the city's own name in English
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, l := range venueLines["eiffel"] {
		if _, ok := langDataEN[l.es]; !ok {
			langDataEN[l.es] = l.en
		}
	}
	for _, p := range pVenue["eiffel"] {
		if _, ok := langDataEN[p.es]; !ok {
			langDataEN[p.es] = p.en
		}
	}
}
