package main

// English text of the amateur superhero (pe_hero.go). TestPevTexts checks that every Spanish text it can show is here
// and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		"¡Voy a salvar a alguien!": "I'll save someone!", "¡Esto es trabajo para mí!": "This is a job for me!",
		"¡Tranquilos, ya llego yo!": "Relax, I'm on my way!", "¡Al rescate!": "To the rescue!",
		"¡Puedo volaaar!": "I can flyyy!", "¡Más alto, más lejos!": "Higher, farther!", "¡Allá voooy!": "Here I gooo!",
		"¡Estoy bien! ¡ESTOY BIEN!": "I'm fine! I'M FINE!", "Mi capa... mi preciosa capa": "My cape... my precious cape",
		"Mañana vuelo mejor": "I'll fly better tomorrow", "El suelo me ha atacado": "The floor attacked me",
		"¡Misión cumplida!": "Mission accomplished!", "¡De nada, ciudadanos!": "You're welcome, citizens!",
		"¡Otra vida salvada!": "Another life saved!", "¡PLAF!": "SPLAT!", "¿Alguien se ahogaba? ¿No?": "Was someone drowning? No?",
		"¡El agua está a salvo!": "The water is safe now!", "¿Ese va en calzoncillos?": "Is he wearing his undies outside?",
		"¿Eso es una sábana?": "Is that a bed sheet?", "¡Qué valiente... o qué loco!": "So brave... or so crazy!",
		"¡Uy! Hoy no vuelo": "Oops! No flying today",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
