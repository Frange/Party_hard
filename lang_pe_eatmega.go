package main

// English text and newspaper briefs of the eating contest and the guest with the megaphone (pe_eatcontest.go,
// pe_megaphone.go). TestPevTexts checks that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// eating contest
		"CONCURSO DE COMER": "EATING CONTEST", "¡Que empiece el concurso!": "Let the contest begin!",
		"¡Preparados, listos... a comer!": "Ready, steady... chew!", "¡Hamburguesas y perritos para todos!": "Burgers and hot dogs for everyone!",
		"¡Ñam!": "Nom!", "¡Uno más!": "One more!", "¡Otro!": "Another!", "¡Esto no es nada!": "This is just the starter!",
		"Mmmf...": "Mmmf...", "¡Come, come!": "Eat, eat, eat!", "¡Vamos, vamos!": "Go, go, go!", "¡Otro perrito!": "Another hot dog!",
		"¡No puede más!": "He's about to pop!", "¡Qué máquina!": "What a machine!", "¡CAMPEÓN!": "CHAMPION!",
		"¡Que traigan más!": "Bring out the next cow!", "¡Invicto!": "Undefeated!", "¡Glup!": "Gulp!", "No puedo más...": "I can't... another bite...",
		"Me... me va a salir...": "It's... coming back up...", "¡Revancha!": "Rematch!", "¡Estaba frío!": "Mine was cold!",
		"Uf, qué lleno...": "Ugh, I'm stuffed...",
		// megaphone
		"¡ATENCIÓN: SE ACABA EL HIELO!":                "ATTENTION: WE ARE OUT OF ICE! PANIC!",
		"¡EL DJ TIENE PROHIBIDO SER PESADO!":           "THE DJ IS LEGALLY BANNED FROM BEING LAME!",
		"¡QUIEN HAYA DEJADO UN PATO EN LA PISCINA...!": "WHOEVER PUT A DUCK IN THE POOL: IT BITES!",
		"¡RECUERDEN: BEBAN AGUA!":                      "REMINDER: WATER IS ALSO A DRINK!",
		"¡SE HA PERDIDO UN ZAPATO! ¡SOLO UNO!":         "LOST: ONE SHOE. JUST THE ONE. DON'T ASK.",
		"¡PROHIBIDO BAILAR PEOR QUE YO!":               "NOBODY MAY DANCE WORSE THAN ME!",
		"¡LA CONGA SALE EN CINCO MINUTOS!":             "THE CONGA LINE DEPARTS IN FIVE MINUTES!",
		"¡EL BAÑO DE LA IZQUIERDA NO, POR FAVOR!":      "AVOID THE LEFT BATHROOM. TRUST ME.",
		"¡HAY CROQUETAS! ¡REPITO: HAY CROQUETAS!":      "SNACKS ARE OUT! THIS IS NOT A DRILL!",
		"¡QUE NADIE TOQUE EL TERMOSTATO!":              "WHOEVER TOUCHED THE THERMOSTAT: WE KNOW.",
		"¡SE BUSCA PAREJA DE BAILE PARA MI TÍO!":       "MY UNCLE NEEDS A DANCE PARTNER. URGENTLY.",
		"¡EL KARAOKE CIERRA POR SU BIEN!":              "KARAOKE IS CLOSED FOR YOUR OWN SAFETY!",
		"¡SÍ, POR FAVOR! ¡MÁS FIESTA!":                 "YES, PLEASE! MORE PARTY!",
		"¡NO DEN DE COMER AL DJ!":                      "PLEASE DO NOT FEED THE DJ!",
		"¡COCHE ROJO: TIENE LAS LUCES PUESTAS!":        "RED CAR: YOUR LIGHTS ARE ON. AND YOUR KARAOKE.",
		"¡ÚLTIMA LLAMADA PARA EL ÚLTIMO BAILE!":        "LAST CALL FOR THE LAST DANCE. AGAIN.",
		"¡APLAUSOS PARA MÍ, POR FAVOR!":                "A ROUND OF APPLAUSE FOR ME, PLEASE!",
		"¡PROBANDO, PROBANDO! ¡SE OYE!":                "TESTING, TESTING! OH WOW, I'M LOUD!",
		"¡NADIE SE VA HASTA QUE YO LO DIGA!":           "NOBODY LEAVES UNTIL I FIND MY KEYS!",
		"¡Buuu!":                                       "Booo!", "¡Bájate de ahí!": "Get off the stage!", "¡Que alguien le quite eso!": "Someone take that thing away!",
		"¡Sí, por favor!": "Yes, please!", "¡Bien dicho!": "Well said!", "¡Eso, eso!": "Hear, hear!",
		"¡Mis oídos!": "My ears!", "¡No hace falta gritar!": "No need to shout!", "¿Se me oye? ¿Sí?": "Is this thing on? Hello?",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
	r := registerPaperEvent
	r("comer", "SOCIEDAD", []paperTxt{
		tx("Concurso de comer: el ganador pidió postre. El perdedor, un cubo.", "Eating contest: the winner asked for dessert. The loser, for a bucket."),
		tx("Hamburguesas y perritos a toda velocidad: récord de la noche en la mesa larga.", "Burgers and hot dogs at full speed: record of the night at the long table."),
		tx("Cuatro valientes, un mantel de cuadros y ninguna servilleta limpia.", "Four brave eaters, one checked tablecloth and not a clean napkin left."),
	})
	r("megafono", "SOCIEDAD", []paperTxt{
		tx("Un invitado con megáfono anunció que se acababa el hielo. Cundió el pánico.", "A guest with a megaphone announced the ice was running out. Panic followed."),
		tx("Los avisos del megáfono, por orden de importancia: el hielo, el DJ y un pato.", "The megaphone notices, in order of importance: the ice, the DJ and a duck."),
		tx("Vecinos confirman que el megáfono se oyó a tres calles. Sí, por favor.", "Neighbours confirm the megaphone was heard three streets away. Yes, please."),
	})
}
