package main

// English texts of the storm of the open-air parties (weather.go). Merged into langDataEN without overwriting
// anything that already exists.
func init() {
	for k, v := range map[string]string{
		"Empieza a llover... ¡y el DJ sube el volumen!":              "It starts to rain... and the DJ turns it up!",
		"¡Tormenta! Los paraguas se ponen a bailar.":                 "Storm! The umbrellas start dancing.",
		"Escampa. Alguien seca los altavoces con un secador.":        "The rain stops. Someone dries the speakers with a hairdryer.",
		"Deja de llover. La pista huele a perro mojado.":             "The rain stops. The dance floor smells of wet dog.",
		"¡Nevada fuerte! Los gorros de lana suben de precio.":        "Heavy snow! Woolly hats just went up in price.",
		"Empieza a nevar a lo grande. El muñeco de nieve se apunta.": "Big snow coming down. The snowman joins the party.",
		"Deja de nevar. Alguien ya ha hecho un iglú.":                "The snow stops. Someone has already built an igloo.",
		"¡LLUEVE!":     "RAIN!",
		"¡MI PELO!":    "MY HAIR!",
		"¡A CUBIERTO!": "TAKE COVER!",
		"¡QUÉ FRÍO!":   "SO COLD!",
		"¡NIEVE!":      "SNOW!",
		"¡MIS PIES!":   "MY FEET!",
		"¡Un rayo cae en plena fiesta! Esta vez ha sido el cielo.": "Lightning strikes the party! This time it was the sky.",
		"¡Rayo! La naturaleza también se ha apuntado a la fiesta.": "Lightning! Nature has joined the party too.",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
