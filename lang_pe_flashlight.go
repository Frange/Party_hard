package main

// English text and newspaper briefs of the torch guard (pe_flashlight.go). TestPevTexts checks that every Spanish text it
// can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		"¿Quién anda ahí?": "Who goes there?", "¿Hay alguien ahí?": "Anybody there?",
		"¡Alto ahí!": "Stop right there!", "¡Alto! ¡Seguridad!": "Freeze! Security!",
		"¡Eh! ¡Mi linterna!": "Hey! My flashlight!", "¿Y la luz? ¡Pilas!": "Lights out? Batteries!",
		"Apagar linterna":                       "Switch off flashlight",
		"Le has apagado la linterna al guardia": "You switched off the guard's flashlight",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
	registerPaperEvent("linterna", "SUCESOS", []paperTxt{
		tx("Un guardia con linterna patrulló la fiesta toda la noche. Las pilas no aguantaron.", "A guard with a flashlight patrolled the party all night. The batteries gave up."),
		tx("«¿Quién anda ahí?», preguntó el guardia de la linterna. Nadie contestó.", "\"Who goes there?\" asked the guard with the flashlight. Nobody answered."),
		tx("Alguien le apagó la linterna a un vigilante por la espalda. Sigue buscando el botón.", "Somebody switched off a watchman's flashlight from behind. He's still looking for the button."),
	})
}
