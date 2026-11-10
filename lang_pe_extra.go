package main

// English text of the party events pe_tvcrew.go, pe_booth.go, pe_magic.go and pe_toiletq.go (TestPevTexts checks that
// every Spanish text the events register is here and fits in 48 bytes).
func init() {
	for es, en := range map[string]string{
		// TV crew
		"En directo desde la fiesta...": "Live from the party...", "Buenas noches, hay mucho ambiente": "Good evening, great atmosphere here",
		"Les habla su reportero favorito": "Your favourite reporter speaking", "Volvemos con ustedes al estudio": "Back to you in the studio",
		"Hay buen ambiente. Y canapés": "Great vibes. And canapés", "Nos dicen que todo va con normalidad": "We are told all is perfectly normal",
		"Un momento: ¿eso es un pato?": "One moment: is that a duck?", "¡Silencio, estamos en el aire!": "Quiet, we are on air!",
		"Cámara, enfoca la tortilla": "Camera, focus on the omelette", "Seguimos sin saber quién invitó al pato": "Still no idea who invited the duck",
		"Dicen que el DJ es sospechoso": "They say the DJ is suspicious", "Fuentes bien informadas: no hay fuentes": "Well-informed sources: there are none",
		"Corten, se me ha movido el pelo": "Cut, my hair moved", "Estamos en vivo y muy nerviosos": "We are live and very nervous",
		"Esto es periodismo de barra libre": "This is open-bar journalism", "¿Algo que declarar? ¡Hola, mamá!": "Anything to declare? Hi, Mum!",
		"Se oyen sirenas. Debe ser fiesta": "Sirens heard. It must be a party", "Hay un charco muy sospechoso aquí": "There is a very suspicious puddle here",
		"Todo bajo control... casi todo": "All under control... almost all", "La policía no hace declaraciones": "The police make no statements",
		"Nos dicen que alguien se ha ido pronto": "We hear somebody left early",
		"¡Hola, mamá!":                           "Hi, Mum!", "¡Salgo en la tele!": "I'm on TV!", "¡Saludad a la cámara!": "Wave at the camera!",
		"¡Mírame, soy famoso!": "Look at me, I'm famous!", "¡Hola, abuela!": "Hi, Gran!", "¡Un selfi con la cámara!": "A selfie with the camera!",
		// magician
		"¡Abracadabra, pata de cabra!": "Abracadabra, goat's foot!", "¡Nada por aquí, nada por allá!": "Nothing here, nothing there!",
		"¡Y ahora, el gran final!": "And now, the grand finale!", "¡Presto!": "Presto!", "¡Ta-chán!": "Ta-da!",
		"¡Esto no estaba en el ensayo!": "This wasn't in the rehearsal!", "¿Dónde está el conejo?": "Where is the rabbit?",
		"¡Se me cae el sombrero!": "My hat is falling off!", "¡Elige una carta! ¡Esa!": "Pick a card! That one!",
		"¡Bravo!": "Bravo!", "¡Magia!": "Magic!", "¡Otra, otra!": "Again, again!", "¡Cómo lo hace!": "How does he do it!",
		"¡Buuu!": "Boooo!", "¡Menudo timo!": "What a rip-off!", "¡Devuelve el dinero!": "Give us our money back!", "¡Qué mal truco!": "What a bad trick!",
		"¡Tú! ¡Sígueme el truco!": "You! Follow my trick!", "¡Gran público! Toma, tu propina": "Great audience! Here, your tip",
		"¡Qué mal público!": "What a terrible audience!",
		// photo booth
		"Cuatro fotos y ni una buena": "Four photos and not one good", "Salió hasta la sombra del sombrero": "Even the hat's shadow came out",
		"La tercera foto tiene lengua": "The third photo has tongue", "Esta se la enseñaré a mis nietos": "I'll show this one to my grandkids",
		"¿Quién es el de la esquina? ¡Soy yo!": "Who is that in the corner? It's me!", "Cuatro caras, cuatro mundos": "Four faces, four worlds",
		"Me veo mejor en persona": "I look better in person", "¡Hay una mano en todas!": "There is a hand in every one!",
		"¡Sonrían!": "Say cheese!", "¡Otra más, otra más!": "One more, one more!", "¡Esa salió con ojos de pez!": "That one came out fisheye!",
		"¡Ya salen, ya salen!": "Here they come, here they come!",
		// toilet queue
		"¡No aguanto más!": "I can't hold it any longer!", "¡Date prisa, por favor!": "Hurry up, please!", "¡Esto es una emergencia!": "This is an emergency!",
		"Cruzo las piernas y rezo": "Crossing my legs and praying", "¡Tardas un mundo!": "You take forever!", "¿Quién hace tanta cola?": "Who made this queue?",
		"¡Me hago pis!": "I'm bursting!", "¡Qué mal momento!": "Terrible timing!",
		"¡Oye!": "Oi!", "¡Eh, hay cola!": "Hey, there's a queue!", "¡Al final de la cola!": "Back of the queue!",
		"¡Solo será un momento!": "It'll only be a moment!", "¡Es que me urge mucho!": "It's just very urgent!",
		"¡Gracias, qué alivio!": "Thanks, what a relief!", "Uf, ahora sí": "Phew, now we're talking",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
