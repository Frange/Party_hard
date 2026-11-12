package main

// English of the poker table, the explosive cake, the photographer, the painter on his ladder and the sleeping guard
// (pe_poker.go, trapcake.go, pe_photo.go, pe_painter.go, pe_sleepguard.go). TestPevTexts checks the lot (48 bytes).
func init() {
	for es, en := range map[string]string{
		// poker
		"Reparto yo": "I'll deal", "Hagan sus apuestas": "Place your bets", "Nueva mano": "New hand", "Cortad, que barajo": "Cut, I'm shuffling",
		"Subo diez": "Raise ten", "Lo veo": "I'll call", "Paso": "Check", "Voy con todo": "All in", "Me retiro": "I fold", "Cinco más": "Five more",
		"Llevo una mano de escándalo": "I've got a monster hand", "Ni pestañeo": "Not even blinking", "Yo no me tiro faroles": "I never bluff",
		"¡Va de farol!": "He's bluffing!", "No me lo creo": "I don't buy it", "Ese tic te delata": "That twitch gives you away",
		"¡Escalera de color!": "Straight flush!", "¡Full de reyes!": "Full house, kings!", "¡Póker de ases!": "Four aces!", "¡Toma ya!": "Oh yeah!",
		"¡Otra vez!": "Again?!", "Las cartas me odian": "The cards hate me", "¿Quién ha barajado?": "Who shuffled this?", "Qué suerte tienes...": "Lucky you...",
		"¡Me habéis desplumado!": "You've cleaned me out!", "¡Me largo, tramposos!": "I'm out, you cheats!", "¡Esto está amañado!": "This game is rigged!",
		"Buena partida": "Good game", "La próxima, revancha": "Rematch next time", "Me voy con las ganancias": "I'm off with my winnings",
		"¡Se acabó la partida!": "Game over, folks!", "¡Dejad las cartas!": "Drop the cards!",
		// cake
		"¡NATA!": "CREAM!", "¡Me ha entrado en el ojo!": "It's in my eye!", "¡Mi traje!": "My suit!", "¡Ups!": "Oops!", "¡Resbala!": "Slippery!",
		"¡Qué tarta más mala!": "What a terrible cake!", "¡La tarta ha explotado!": "The cake has exploded!",
		"Te ha explotado una tarta en la cara": "A cake blew up in your face",
		// photographer
		"¡Una foto, por favor!": "A photo, please!", "¡Juntaos un poco!": "Squeeze in a bit!", "¡Para el álbum!": "For the album!", "¡Mirad aquí!": "Look over here!",
		"¡Patata!": "Cheese!", "¡Decid patata!": "Say cheese!", "¡Sonreíd!": "Smile!",
		"¿Salgo bien?": "Do I look good?", "¡Mi lado bueno!": "My good side!", "¡Patataaa!": "Cheeeese!",
		"¡Preciosos!": "Gorgeous!", "¡Qué fotón!": "What a shot!", "Esta sale en portada": "This one's front page", "¡Flash!": "Flash!",
		// painter
		"Un poco más a la izquierda...": "A bit more to the left...", "Esta pared quedará preciosa": "This wall will look lovely",
		"¡Cuidado con el bote!": "Mind the paint pot!", "Pintura fresca, ¿eh?": "Wet paint, OK?", "No me mováis la escalera": "Don't wobble my ladder",
		"¡Uaaah!": "Whoaaa!", "¡La escaleraaa!": "My ladderrr!", "Mañana la segunda mano": "Second coat tomorrow",
		"Listo. Ni una gota fuera": "Done. Not a drop spilled", "¡El pintor se ha caído!": "The painter fell!", "¡Qué golpe!": "What a crash!",
		"Tirar la escalera": "Knock the ladder over", "Está muy alto: tira la escalera": "He's too high: knock the ladder over",
		// sleeping guard
		"¡Eh!": "Hey!", "¡Eh! ¿Quién anda ahí?": "Hey! Who goes there?", "¡No estaba durmiendo!": "I wasn't sleeping!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
