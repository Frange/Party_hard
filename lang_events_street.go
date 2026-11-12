package main

// English of the street events, the mascots, the dance battle and the party countdown (events_street.go, mascot.go,
// pe_dancebattle.go, countdown.go). TestPevTexts checks that every Spanish text of pevTexts is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// the car out of fuel
		"¡Vaya, sin gasolina!": "Oh great, out of gas!", "¿Pero qué le pasa ahora?": "What's wrong with it now?",
		"¡Me he quedado tirado!":       "I'm stranded!",
		"¡Humo! Eso no es buena señal": "Smoke! That's not a good sign", "¿Y ahora qué hago?": "And now what do I do?",
		"¿Alguien me echa una mano?": "Can somebody give me a hand?", "Venga, bonito, arranca...": "Come on, baby, start...",
		"Uf... cómo pesa": "Ugh... it's so heavy", "¡Un poco más!": "Just a bit more!",
		"¡Empujando, que es gerundio!": "Push it, push it real good!",
		"¡Arrancó!":                    "It started!", "¡Por fin!": "Finally!", "¡Perdón por el atasco!": "Sorry about the jam!",
		"¡PIII!": "HONK!", "¡PIII, PIII!": "HONK, HONK!", "¡Venga ya!": "Oh come on!", "¡Mueve eso!": "Move that thing!",
		"Bueno, esperamos...": "Well, we'll wait...", "Sin prisa, eh": "No rush, eh",
		// the taxi
		"¡Taxi!": "Taxi!", "Me voy, que mañana curro": "I'm off, work tomorrow", "Yo me piro": "I'm out of here",
		"¡Esperadme!": "Wait for me!", "Nos vamos a otra fiesta": "We're off to another party",
		"Un taxi se lleva a unos invitados": "A taxi takes some guests away",
		"Llega un taxi con más invitados":   "A taxi brings more guests",
		// the mascots
		"¡Abrazos gratis!": "Free hugs!", "¡Qué calor hace aquí dentro!": "It's so hot in here!",
		"¡Baila conmigo!": "Dance with me!", "¡Foto con la mascota!": "Photo with the mascot!",
		"No veo nada por los ojos": "I can't see a thing in here",
		"¡Grrr... de broma!":       "Grrr... just kidding!", "¡Muuu!": "Mooo!", "¡Cuac, cuac!": "Quack, quack!",
		"¡Raaawr!": "Raaawr!", "¡Mascota!": "Mascot!",
		// the dance battle
		"¡Te reto a un duelo de baile!": "I challenge you to a dance-off!", "¿Bailas o qué?": "You dancing or what?",
		"¡Esta pista es mía!": "This dance floor is mine!", "¡A ver qué sabes hacer!": "Let's see what you've got!",
		"¿Eso es todo?": "Is that all?", "¡Mira y aprende!": "Watch and learn!", "¡Toma ya!": "Oh yeah!",
		"¡Supera esto!": "Beat that!", "Mi abuela baila mejor": "My grandma dances better",
		"¡Uooo!": "Woooo!", "¡Baila, baila!": "Dance, dance!", "¡Duelo!": "Dance-off!", "¡Qué nivel!": "What skills!",
		"¡Otra, otra!": "Encore!", "¡Soy el rey de la pista!": "I'm the king of the floor!", "¡Gané!": "I won!",
		"¡Campeón!": "Champion!", "Quiero la revancha": "I want a rematch", "Hoy no era mi día": "Not my day",
		"Me ha dado flato": "I got a stitch",
		// the countdown
		"LA FIESTA ACABA EN %d:%02d":                     "THE PARTY ENDS IN %d:%02d",
		"¡FIESTA CERRADA! Llega la policía":              "PARTY'S OVER! The police are here",
		"¡Hora de cerrar! La policía disuelve la fiesta": "Closing time! The police break up the party",
		"¡Hola!": "Hi!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
