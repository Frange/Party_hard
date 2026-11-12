package main

// English text of the diving board (poolboard.go) and the cleaning crew (cleaners.go). TestPevTexts checks that every
// Spanish text the events can show is here and fits (48).
func init() {
	for es, en := range map[string]string{
		// diving board
		"¡Mirad cómo salto!": "Watch me jump!", "¡Allá voy!": "Here I go!", "¡Bomba va!": "Cannonball!",
		"¡Al agua, patos!": "Into the water, ducks!", "¡Que me grabe alguien!": "Somebody film this!",
		"¡Un diez!": "A perfect ten!", "¡Uuuuh!": "Ooooh!", "¡Otra, otra!": "Again, again!",
		"¡Menudo planchazo!": "What a belly flop!", "¡Toma ya!": "Oh yeah!", "¡Qué estilo!": "What style!",
		// cleaning crew
		"Equipo de limpieza, ¡paso!": "Cleaning crew, coming through!", "¡Venimos a limpiar!": "We came to clean!",
		"Aquí no ha pasado nada": "Nothing happened here", "Otro más para la bolsa": "Another one for the bag",
		"Esto sale con lejía": "Bleach gets this out", "¡Cuidado, suelo mojado!": "Careful, wet floor!",
		"Siempre igual en estas fiestas": "Same thing every party", "Cobro por horas, eh": "I charge by the hour, you know",
		"Llega el equipo de limpieza": "The cleaning crew arrives",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
