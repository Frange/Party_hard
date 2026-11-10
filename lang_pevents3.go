package main

// English text of the fist fight and the shooting between guests (pe_pelea.go, pe_tiroteo.go). TestPevTexts checks
// that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// fight
		"¿Qué miras?": "What are you looking at?", "¡Repite eso!": "Say that again!", "¡Esa era MI copa!": "That was MY drink!",
		"¡Me has pisado!": "You stepped on me!", "¡Tú a mí no me hablas así!": "Don't you talk to me like that!",
		"¡Fuera de mi pista!": "Get off my dance floor!", "¿Tienes algún problema?": "Got a problem?", "¡Has empezado tú!": "You started it!",
		"¡Toma!": "Take that!", "¡Ay!": "Ouch!", "¡Pum!": "Pow!", "¡Au!": "Ow!", "¡Zas!": "Whack!",
		"¡Pelea, pelea!": "Fight, fight!", "¡Uuuh!": "Ooooh!", "¡Separadlos!": "Break it up!", "¡Dale!": "Hit him!",
		"¡Que alguien haga algo!": "Somebody do something!",
		"Vale, vale... ya está":   "OK, OK... that's it", "Bueno, perdona": "Well, sorry", "¿Amigos?": "Friends?",
		"Ya me había cansado": "I was getting tired anyway", "¡K.O.!": "K.O.!", "Veo pajaritos...": "I'm seeing birdies...",
		"¡Se le ha ido la mano!": "That got way out of hand!", "Una pelea ha acabado muy mal": "A fight has ended very badly",
		"¡Paz!": "Peace!", "Gracias por separarnos": "Thanks for pulling us apart",
		"¡AAAH!": "AAAH!", "¡CORRE!": "RUN!", "¡Al suelo!": "Get down!",
		// shooting
		"¡Tú otra vez!": "You again!", "¡Esta fiesta es pequeña para los dos!": "This party ain't big enough for us two!",
		"¡Quieto ahí!": "Freeze!", "¡Me debes dinero!": "You owe me money!",
		"¡Que lleva pistola!": "He's got a gun!", "¡Todos al suelo!": "Everybody down!", "¡Llamad a la policía!": "Call the police!",
		"¡Tiroteo en la fiesta!": "Shoot-out at the party!", "Alguien ha caído en el tiroteo": "Somebody went down in the shoot-out",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
