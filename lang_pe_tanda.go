package main

// English text of the suitcase, the human statue, the cook on fire, the choir, the street vendor and the drone
// (pe_suitcase.go, pe_statue.go, pe_cookfire.go, pe_choir.go, pe_vendor.go, pe_drone.go, pe_treats.go).
// TestPevTexts checks that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// suitcase
		"Abrir maleta": "Open suitcase", "Vuelvo enseguida...": "Back in a minute...", "¿Y mi maleta?": "Where's my suitcase?",
		"¡Aquí estaba!": "There it is!", "¡Mi maleta! ¿Quién la ha abierto?": "My suitcase! Who opened it?",
		"Maleta: ¡un fajo de puntos!": "Suitcase: a wad of points!", "Maleta: ¡un pato de goma! Cuac": "Suitcase: a rubber duck! Quack",
		"Maleta: ¡confeti para todos!": "Suitcase: confetti for everyone!", "Maleta: ¡billetes por el aire!": "Suitcase: banknotes in the air!",
		"Maleta: ¡una tarta entera!": "Suitcase: a whole cake!", "Maleta: ¡un gorro de fiesta!": "Suitcase: a party hat!",
		"Maleta: ¡algo útil dentro!": "Suitcase: something useful inside!", "Maleta: solo calcetines sucios...": "Suitcase: just dirty socks...",
		"¡JAJAJA!": "HAHAHA!", "¡Cuac!": "Quack!", "¡Billetes!": "Money!", "¡Tarta!": "Cake!", "¡Hala!": "Whoa!",
		// statue
		"¡AAAH! ¡Se ha movido!": "AAAH! It moved!", "¡Qué susto!": "What a fright!", "¡Está vivo!": "It's alive!", "¡Mamá!": "Mommy!",
		"¡Una estatua viva!": "A living statue!", "¡Qué quieto está!": "He's so still!",
		// cook on fire
		"¡Hoy cocino yo!": "I'm cooking today!", "Un chorrito de aceite...": "A splash of oil...", "¡Uy! ¡EL ACEITE!": "Oops! THE OIL!",
		"¡FUEGO EN LA COCINA!": "FIRE IN THE KITCHEN!", "¡Llamad a los bomberos!": "Call the fire brigade!", "¡Mi sartén!": "My frying pan!",
		"¡Gracias, bomberos!": "Thank you, firefighters!", "Yo solo quería freír un huevo": "I only wanted to fry an egg",
		// choir
		"¡Todos juntos!": "All together now!", "¡La la laaa!": "La la laaa!", "¡Oh, oh, oooh!": "Oh, oh, oooh!",
		"¡Otra vez el estribillo!": "The chorus again!", "¡Se acabó el concierto!": "The concert is over!",
		"¡Que alguien llame a un médico!": "Somebody call a doctor!", "¡Desafinaba, pero tanto no!": "He sang off key, but not THAT bad!",
		"¡Corred!": "Run!", "¡Nunca más canto!": "I'm never singing again!",
		// street vendor
		"¡Churros con wifi!": "Churros with wifi!", "¡Helado de croqueta!": "Croquette ice cream!", "¡Globos que no vuelan!": "Balloons that don't fly!",
		"¡Limonada de ayer, como nueva!": "Yesterday's lemonade, good as new!", "¡Palomitas cuadradas!": "Square popcorn!",
		"¡Pipas sin cáscara y sin pipa!": "Seeds with no shell and no seed!", "¡Bocatas invisibles, muy ligeros!": "Invisible sandwiches, very light!",
		"¡Gorros para el confeti!": "Hats for the confetti!", "¡Gracias! ¡Que aproveche!": "Thanks! Enjoy!", "¡Me voy, que me multan!": "I'm off before I get fined!",
		"¡Vuelva pronto!": "Come back soon!", "Comprar (gratis)": "Buy (free)",
		"Churro turbo: ¡corres más unos segundos!": "Turbo churro: you run faster for a bit!", "Gorro de fiesta: ¡qué elegancia!": "Party hat: so elegant!",
		"Un globo: ¡que no se te escape!": "A balloon: don't let it fly away!", "Limonada: ¡aguante a tope!": "Lemonade: stamina to the max!",
		// drone
		"¡Mira, un dron!": "Look, a drone!", "¿Nos está grabando?": "Is it filming us?", "¡Saluda, que salimos en la tele!": "Wave, we're on TV!",
		"¡Qué lucecitas!": "Such pretty lights!", "¿Eso es un ovni?": "Is that a UFO?", "¡Que no se caiga encima!": "Don't let it fall on us!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}

// The small news of the newspaper (newspaper_events.go) for these events: three variants each, ES and EN.
func init() {
	r := registerPaperEvent
	r("suitcase", "SUCESOS", []paperTxt{
		tx("Una maleta olvidada en plena fiesta resultó traer premio. El dueño, no.", "A suitcase left in the middle of the party came with a prize. The owner did not."),
		tx("Objetos perdidos: una maleta, un pato de goma y la dignidad de su dueño.", "Lost and found: a suitcase, a rubber duck and its owner's dignity."),
		tx("Alguien abrió una maleta ajena. Dentro había más fiesta.", "Somebody opened a stranger's suitcase. Inside there was more party."),
	})
	r("statue", "CULTURA", []paperTxt{
		tx("Una estatua humana pasó la noche inmóvil. Bueno, casi toda.", "A human statue spent the night perfectly still. Well, almost all of it."),
		tx("La estatua plateada asustó a medio público con un solo gesto.", "The silver statue scared half the audience with a single move."),
		tx("Arte callejero: un invitado pintado de metal posó durante horas.", "Street art: a guest painted like metal posed for hours."),
	})
	r("cookfire", "SUCESOS", []paperTxt{
		tx("Un chorrito de aceite de más y la cocina acabó en manos de los bomberos.", "One splash of oil too many and the kitchen ended up in the firefighters' hands."),
		tx("El cocinero solo quería freír un huevo. Los bomberos llegaron antes que el huevo.", "The cook only wanted to fry an egg. The firefighters arrived before the egg."),
		tx("Llamarada en los fogones: la fiesta se quedó sin cena y con olor a humo.", "Flare-up at the stove: the party lost its dinner and gained a smell of smoke."),
	})
	r("choir", "MÚSICA", []paperTxt{
		tx("Un coro espontáneo cantó abrazado y desafinado. Fue precioso.", "A spontaneous choir sang arm in arm and off key. It was beautiful."),
		tx("Seis invitados, un estribillo y ni una sola nota en su sitio.", "Six guests, one chorus and not a single note in its place."),
		tx("El coro improvisado se balanceó tanto que mareó a los de la primera fila.", "The improvised choir swayed so much the front row got dizzy."),
	})
	r("vendor", "GASTRONOMÍA", []paperTxt{
		tx("Un vendedor ambulante ofreció churros con wifi. Nadie encontró la contraseña.", "A street vendor sold churros with wifi. Nobody found the password."),
		tx("Pasó un carrito de helados de croqueta. Hubo valientes.", "A cart selling croquette ice cream went by. There were brave ones."),
		tx("El vendedor regaló gorros y globos y se marchó antes de la multa.", "The vendor gave away hats and balloons and left before the fine."),
	})
	r("drone", "ÚLTIMA HORA", []paperTxt{
		tx("Un dron con lucecitas sobrevoló la fiesta. Nadie sabe de quién era.", "A drone with little lights flew over the party. Nobody knows whose it was."),
		tx("Medio público miró al cielo por un dron. El otro medio, a su móvil.", "Half the crowd looked at the sky for a drone. The other half, at their phones."),
		tx("Un dron de colores dio una vuelta y se fue. Se sospecha de un vecino.", "A coloured drone did a lap and left. A neighbour is suspected."),
	})
}
