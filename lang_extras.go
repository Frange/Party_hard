package main

// English text of the second batch of "sometimes" things: the stray dog (pe_dog.go), the skaters (pe_skaters.go),
// the stashes (stash.go), the memory of the guests (memory.go), the CARRERA page of OBJETIVOS (goalstats.go) and the
// help line of the uses of E (help.go). TestPevTexts checks that every Spanish text registered with pevTexts is here
// and fits (48). The prizes of the newspaper carry their English in newspaper_awards.go.
func init() {
	for es, en := range map[string]string{
		// stray dog
		"¡Guau!": "Woof!", "¡Guau, guau!": "Woof, woof!", "¡Arf, arf!": "Arf, arf!", "Grrr... ¡mío!": "Grrr... mine!", "¡Rrrguau!": "Rrrwoof!",
		"¡Guau! ¿Qué es eso?": "Woof! What is that?", "*sniff, sniff*": "*sniff, sniff*", "*olfatea*": "*sniffs around*", "*sniff* ¡Guau!": "*sniff* Woof!",
		"El perro callejero ya no ladrará más": "The stray dog will not bark any more",
		// skaters
		"¡Ay, mi cola!": "Ouch, my tailbone!", "¡Whoaaa!": "Whoaaa!", "¡PUM! Estoy bien...": "BANG! I'm fine...", "¡El hielo me odia!": "The ice hates me!",
		"¡Cuidado!": "Watch out!", "¡Mirad esto!": "Look at this!", "¡Triple axel!": "Triple axel!", "¡Soy un cisne!": "I'm a swan!",
		"¡Y sin manos!": "And no hands!", "¡Puntuación perfecta!": "Perfect score!",
		// stashes
		"Un billete de 20 arrugado. ¡A la caja!": "A crumpled 20 note. Cha-ching!", "Una pizza fría. Aun así, te la comes": "A cold pizza. You eat it anyway",
		"Una llave oxidada. No abre nada... aún": "A rusty key. It opens nothing... yet", "¡Calzoncillos con globos! Ni preguntes": "Underpants with balloons! Don't ask",
		"Un peluche con cara de pocos amigos": "A plush toy with an unfriendly face", "Una dentadura postiza. Te sonríe": "False teeth. They smile at you",
		"Una navaja... de plástico. Da más pena": "A knife... made of plastic. Sadder than scary", "¡Una ficha dorada de casino!": "A golden casino token!",
		// memory of the guests
		"¡Tú otra vez!": "You again!", "Yo a ti te conozco...": "I know you...", "¿No eras tú el de la otra vez?": "Weren't you the one from last time?",
		"Ese es el que me dio el susto.": "That's the one who scared me.", "Mira quién ha vuelto.": "Look who's back.", "Esta vez no me pillas.": "You won't catch me this time.",
		"Me suenas de una pesadilla.": "You sound like a nightmare.", "Tú eres el de las malas pulgas.": "You're the grumpy one.",
		"¡Madre mía, otra vez este!": "Oh dear, this one again!", "Te voy a vigilar, ¿eh?": "I'm watching you, OK?", "Esta vez sí he visto algo.": "This time I did see something.",
		"Aquí viene el problemático.": "Here comes trouble.", "No me mires así, te recuerdo.": "Don't look at me like that, I remember you.",
		"Me debes una disculpa y un sofá.": "You owe me an apology and a sofa.", "¡Eh, tú! Sí, tú. Te vi.": "Hey, you! Yes, you. I saw you.",
		"Cuidado con ese, ya sé cómo es.": "Careful with that one, I know the type.", "Qué cara más familiar y poco amistosa.": "What a familiar and unfriendly face.",
		"Esta vez me quedo junto a la salida.": "This time I'm staying by the exit.", "¿Otra fiesta, otro lío? Claro.": "Another party, another mess? Of course.",
		"No te acerques a mi vaso, por si acaso.": "Stay away from my cup, just in case.", "Te tengo fichado. Con cariño.": "I've got you on file. Fondly.",
		"Sabía que volverías. Qué miedo.": "I knew you'd be back. How scary.", "¿Vienes a por la revancha?": "Are you here for a rematch?",
		"Yo no olvido una cara. Ni una cuchilla.": "I never forget a face. Or a blade.",
		// OBJETIVOS / CARRERA
		"CARRERA": "CAREER", "Partidas jugadas": "Parties played", "Niveles completados": "Levels completed", "Fiestas ganadas": "Parties won",
		"Bajas totales": "Total kills", "Esposas evitadas": "Cuffs dodged", "Tiempo jugado": "Time played", "Puntos ganados": "Points earned",
		"Mejor fiesta": "Best party", "Mejor cadena": "Best chain", "Trofeos": "Trophies", "Personajes": "Characters", "VIPs conocidos": "VIPs met",
		"Asesinos vencidos": "Killers beaten", "Policías vencidos": "Cops beaten", "Mejor racha de estrellas": "Most stars at once",
		"Animales caídos": "Animals down", "Veces que has caído": "Times you fell",
		"Toda tu carrera: se cuenta desde la primera fiesta.": "Your whole career: counted since the first party.",
		"Toda tu carrera: desde el %d/%d/%d, %02d:%02d.":      "Your whole career: since %d/%d/%d, %02d:%02d.",
		"Tab: otra pestaña   ":                                "Tab: another page   ",
		// help
		helpUseNear: "Near: pizza boy, mascot, DJ (song), waiter (drink), magician (tip), stashes",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
