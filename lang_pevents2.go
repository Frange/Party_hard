package main

// English text of the crowd events (pe_waiters.go, pe_drunks.go, pe_philosopher.go, pe_granny.go, pe_karaoke.go,
// pe_tablehide.go). TestPevTexts checks that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// waiters
		"¿Una copita?": "Care for a drink?", "¡Canapés recién hechos!": "Fresh canapés!", "¡Copas frescas!": "Fresh drinks!",
		"Con permiso...": "Excuse me...", "¡Cuidado, que quema!": "Careful, it's hot!", "¡Uy, uy, uy!": "Whoops, whoops, whoops!",
		"Se me ha caído todo...": "I dropped everything...", "¡Mi bandeja!": "My tray!", "¡Qué desastre!": "What a disaster!",
		"¡Salud!": "Cheers!", "¡Una para mí!": "One for me!", "Gracias, majo": "Thanks, mate", "Esto es lo mejor de la fiesta": "This is the best part of the party",
		"¡Ñam!": "Yum!", "¡AAH! ¡Cristales!": "AAH! Broken glass!", "¡Cuidado con el suelo!": "Mind the floor!", "¡Uy!": "Oops!", "¡Madre mía!": "Oh my!",
		// drunks
		"¡Hic!": "Hic!", "¡Ups!": "Oops!", "¡Estoy bien, estoy bien!": "I'm fine, I'm fine!", "Yo controlo... creo": "I'm in control... I think",
		"¡El suelo se ha movido!": "The floor moved!", "¿Quién ha puesto ese escalón?": "Who put that step there?",
		"No he bebido nada... mucho": "I've had nothing... much", "¡Otra ronda!": "Another round!", "Todo da vueltas, ¡qué bien!": "Everything spins, how nice!",
		"Me he tropezado con el aire": "I tripped over the air",
		// philosopher
		"¿Y si el DJ somos todos?": "What if we are all the DJ?", "La barra no existe; la sed sí": "The bar doesn't exist; thirst does",
		"Si bailo solo, ¿bailo o me caigo?": "If I dance alone, do I dance or fall?", "El hielo es agua que se ha puesto seria": "Ice is water that got serious",
		"Somos polvo de estrellas con resaca": "We are stardust with a hangover", "El vaso no está vacío: está pensando": "The glass isn't empty: it's thinking",
		"¿Quién fue primero, la fiesta o el ruido?": "Which came first, the party or the noise?", "Si nadie escucha, ¿suena la música?": "If nobody listens, does the music play?",
		"El tiempo es un círculo... como el hielo": "Time is a circle... like the ice", "Los espejos de la disco saben demasiado": "The disco mirrors know too much",
		"La vida es un canapé: se acaba rápido": "Life is a canapé: it's gone fast", "¿Y si la luna se ha ido de fiesta sin mí?": "What if the moon went partying without me?",
		"El silencio es música que no se invitó": "Silence is music that wasn't invited", "Todos somos el último invitado de alguien": "We are all somebody's last guest",
		"La resaca es el universo cobrando": "A hangover is the universe collecting", "El camarero es el verdadero filósofo": "The waiter is the real philosopher",
		"Nadie baila dos veces en el mismo suelo": "Nobody dances twice on the same floor", "Beber es pensar con más ganas": "Drinking is thinking with more feeling",
		"Hay un universo en cada burbuja": "There is a universe in every bubble", "¿Y si el baño me está esperando a mí?": "What if the toilet is waiting for me?",
		"Cada confeti fue un sueño en rosa": "Every confetti was a dream in pink", "La noche es joven; yo, un poco menos": "The night is young; I, a bit less",
		"Nunca sabrás cuántos bailamos de verdad": "You'll never know how many of us really dance", "La cuenta es lo único que nunca baila": "The bill is the only thing that never dances",
		"Ajá...": "Uh-huh...", "Qué profundo": "So deep", "Hmm, totalmente": "Hmm, totally", "Ya, ya.": "Yeah, yeah.", "Tiene razón": "He has a point", "...": "...",
		// granny
		"Psst... ¿has visto qué hace ese?": "Psst... have you seen what that one is doing?", "En mis tiempos esto no se hacía": "In my day this wasn't done",
		"Yo lo cuento todo, ¿eh?": "I tell everything, you know?", "Mmm... muy sospechoso": "Hmm... very suspicious", "Este chico no me gusta nada": "I don't like this boy at all",
		"¡Y sin abrigo! ¡Se va a constipar!": "And no coat! He'll catch a cold!", "¿Dónde vas con tanta prisa, hijo?": "Where are you off to in such a hurry, dear?",
		"Eso no se lo digo ni a mi comadre": "I wouldn't tell even my best friend", "Qué mirada más rara tiene": "What a strange look he has",
		"Apunto en mi libretita...": "I'm writing it in my little notebook...", "¡Ay, qué modernos son!": "Oh, how modern they are!",
		"Y luego dicen que no pasa nada": "And then they say nothing happens", "Esto mañana sale en el barrio": "This will be all over the block tomorrow",
		"¡Si lo ve mi hermana...!": "If my sister sees this...!", "Algo trama, yo lo huelo": "He's up to something, I can smell it",
		"¡Qué descaro, mirar a una señora!": "The nerve, staring at a lady!", "¡Ay, qué vergüenza!": "Oh, how embarrassing!", "¡Uy, qué ojos!": "Oh my, those eyes!",
		"¡Yo no miraba!": "I wasn't looking!",
		// karaoke
		"¡Ooooh, mi corazón!": "Oooh, my heart!", "¡Nanana nanana!": "Nanana nanana!", "¡Soy el rey del micro!": "I'm the king of the mic!",
		"¡Laaa, la-la-laaa!": "Laaa, la-la-laaa!", "¡Que suene, que suene!": "Play it, play it!", "¡Aaay, ay, ay, ayyy!": "Aaay, ay, ay, ayyy!",
		"¡Mis oídos!": "My ears!", "¡Que alguien pare eso!": "Somebody stop that!", "¡Auch!": "Ouch!", "¡Eso es un delito!": "That's a crime!",
		"¡Qué voz tan... valiente!": "What a... brave voice!", "¡Bravo!": "Bravo!", "¡Otra, otra!": "Encore, encore!", "¡Qué artista!": "What an artist!",
		"¡Cantas genial!": "You sing great!", "¡Se me pone la piel de gallina!": "I've got goosebumps!", "PUNTOS": "POINTS", "¡BIS!": "ENCORE!", "¡AUCH!": "OUCH!",
		// hiding under the tables
		"¡Shh!": "Shh!", "Silencio...": "Quiet...", "No hagas ruido": "Don't make a sound", "¡Que no nos vea!": "Don't let him see us!",
		"Aquí no nos ven...": "They can't see us here...", "Uf... ¿ya pasó?": "Phew... is it over?", "Creo que ya se fue": "I think he's gone", "Qué susto...": "What a scare...",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
