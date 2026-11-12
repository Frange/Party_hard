package main

// English text and newspaper briefs of the petting, the ball thrower and the figure skater (pe_petting.go,
// pe_fetch.go, pe_icelake.go). TestPevTexts checks that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// petting
		"¿Quién es un perro bueno?": "Who's a good dog?", "¡Hola, bonito!": "Hello, cutie!", "Qué orejitas...": "Look at those ears...",
		"¡Toma mimos!": "Cuddles for you!", "Michi, michi...": "Kitty, kitty...", "Qué suave eres": "You're so soft",
		"¡Hola, gatito!": "Hi, kitty!", "¿Me dejas, minino?": "May I, puss?", "Tranquilo, caballito": "Easy, little horse",
		"¡Qué crin tan bonita!": "What a lovely mane!", "Sooo, bonito, sooo": "Whoa, boy, whoa",
		"*mueve la cola*": "*wags its tail*", "*jadea feliz*": "*pants happily*", "*rrrrrr*": "*purrrrr*", "*ronronea*": "*purrs*",
		"*relincha bajito*": "*soft neigh*", "*hiii...*": "*neeeigh...*",
		// ball thrower
		"¡Busca!": "Fetch!", "¡Ve a por ella!": "Go get it!", "¡Tráela!": "Bring it back!", "¡Allá va!": "Here it goes!",
		"¡Buen chico!": "Good boy!", "¡Muy bien!": "Well done!", "¡Qué rápido!": "So fast!", "¡Otra vez!": "Again!",
		"¡Guau!": "Woof!", "¡Guau, guau!": "Woof, woof!", "*gime*": "*whimpers*", "*espera con la pelota*": "*waits with the ball*",
		"¿Jugamos a la pelota?": "Wanna play ball?", "¡Mira lo que tengo!": "Look what I've got!",
		// figure skater
		"¡Abrid paso a la estrella!": "Make way for the star!", "Mi momento sobre hielo": "My moment on ice",
		"¡Hoy toca exhibición!": "Showtime on ice!", "¡Salto doble!": "Double jump!", "¡Hop!": "Hop!",
		"¡Mirad qué altura!": "Look how high!", "¡El cisne!": "The swan!", "Elegancia pura...": "Pure elegance...",
		"¡Gracias, gracias!": "Thank you, thank you!", "¡Quiero un diez!": "I want a perfect ten!", "¡Olé!": "Olé!",
		"¡Precioso!": "Beautiful!", "¡Que se cae!": "He's falling!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
	r := registerPaperEvent
	r("pelota", "VECINOS", []paperTxt{
		tx("Un invitado lanzó la pelota al perro cuatro veces. El perro pedía la quinta.", "A guest threw the ball for the dog four times. The dog wanted a fifth."),
		tx("Récord en la fiesta: un perro devolvió la pelota sin babearla. Casi.", "Party record: a dog brought the ball back without drool. Almost."),
		tx("El perro de la fiesta ya tiene mejor amigo: el que le tiraba la pelota.", "The party dog has a new best friend: the one who threw the ball."),
	})
	r("hielo", "DEPORTES", []paperTxt{
		tx("Un invitado bailó sobre el hielo: piruetas, un cisne y una caída de diez.", "A guest danced on the ice: spins, a swan and a perfect-ten fall."),
		tx("Exhibición sorpresa en la pista de hielo. El público pidió otra.", "Surprise show on the ice. The crowd wanted more."),
		tx("Doble salto, culetazo y reverencia: el patinaje más aplaudido de la noche.", "Double jump, pratfall and a bow: the most cheered skating of the night."),
	})
}
