package main

// English text of the eight "sometimes" party events (pe_*.go): conga, chairs, drinks, barbecue, bachelor party,
// mime, juggler and buskers. TestPevTexts checks that every Spanish text the events can show is here and fits (48).
func init() {
	for es, en := range map[string]string{
		// conga
		"¡UNO, DOS, TRES... CONGA!": "ONE, TWO, THREE... CONGA!", "¡Conga, conga, conga!": "Conga, conga, conga!", "¡Que no pare la conga!": "Keep the conga going!",
		"¡Me apunto!": "I'm in!", "¡Hacedme sitio!": "Make room for me!", "¡Conga!": "Conga!", "¡Más rápido!": "Faster!",
		"¡Uf, qué sed!": "Phew, I'm thirsty!", "¡Mis pies!": "My feet!", "¡Otra vuelta!": "One more lap!",
		// chairs
		"¡Juego de las sillas en la pista!": "Musical chairs on the floor!", "¡STOP!": "STOP!", "¡Había una silla menos!": "There was one chair less!",
		"¡Eso es trampa!": "That's cheating!", "¡Me han empujado!": "I was pushed!", "¡Esa silla era mía!": "That chair was mine!",
		"¡Yo llegué primero!": "I got there first!", "¡No vale, me ataba el zapato!": "No fair, I was tying my shoe!", "¡Mía!": "Mine!",
		"¡Aquí!": "Here!", "¡Uf, por poco!": "Phew, that was close!", "¡Soy el rey de las sillas!": "I'm the king of chairs!", "¡HE GANADO!": "I WON!",
		// drinks
		"¡Concurso de tragos en la barra!": "Drinking contest at the bar!", "¡Bebe, bebe, bebe!": "Drink, drink, drink!", "¡Tú puedes!": "You can do it!",
		"¡Dale, dale, dale!": "Go, go, go!", "¡Otro, otro!": "Another one!", "¡Soy el rey de la barra!": "I'm the king of the bar!",
		"Uuuy... el suelo se mueve": "Whoa... the floor is moving", "¡Hic!": "Hic!", "¡Camarero, otro!": "Waiter, another one!",
		"¡Increíble!": "Incredible!", "Yo ya veo doble...": "I'm seeing double already...",
		// barbecue
		"¡Barbacoa en marcha!": "Barbecue is on!", "¡AAAH! ¡MI CHULETÓN!": "AAAH! MY STEAK!", "¿Qué huele a quemado?": "What's that burning smell?",
		"¡Se quema, se quema!": "It's burning, it's burning!", "¡Cuidado, que sale fuego!": "Watch out, flames!",
		"Estaba al punto... por fuera.": "It was medium... on the outside.", "¡Qué bien huele!": "It smells so good!", "¡Yo quiero una!": "I want one!",
		"¡Ya casi está!": "Almost ready!", "Un poco crujiente, nada más.": "A bit crispy, that's all.", "¡Apártense, soy profesional!": "Stand back, I'm a pro!",
		// bachelor party
		"¡Que se besen, que se besen!": "Kiss, kiss, kiss!", "¡Hoy se casa el amigo!": "Our friend is getting married!", "¡Que viva el novio!": "Long live the groom!",
		"¡Que viva la novia!": "Long live the bride!", "¡Yo soy la que más baila!": "I'm the best dancer here!", "¡Esta noche no hay vuelta atrás!": "No turning back tonight!",
		"¡Cantad conmigo!": "Sing with me!", "¡Despedida, despedida!": "Party time, party time!", "¡CHIN CHIN!": "CHEERS!", "¡Por la despedida!": "To the send-off!",
		"¡Por los nervios!": "To the nerves!", "¡Que sea un gran sí!": "May it be a big yes!",
		"¡Reto: baila como un pato!": "Dare: dance like a duck!", "¡Reto: cuenta un chiste malo!": "Dare: tell a bad joke!",
		"¡Reto: habla en rima un minuto!": "Dare: talk in rhyme for a minute!", "¡Reto: imita a un camarero!": "Dare: act like a waiter!",
		"¡Reto: un cumplido al primero que veas!": "Dare: compliment the first person you see!", "¡Reto: abraza a un desconocido!": "Dare: hug a stranger!",
		"¡Reto: grita \"patata\" tres veces!": "Dare: shout \"potato\" three times!",
		"¡Que lo haga!":                       "Make him do it!", "¡Reto aceptado!": "Challenge accepted!", "¡Cobarde!": "Chicken!", "¡Eso, eso!": "Yes, yes!",
		"¡Despedida de soltero en marcha!": "Stag party is on the move!", "¡Despedida de soltera en marcha!": "Hen party is on the move!",
		// mime
		"?!": "?!", "...": "...", "!!": "!!", "¿?": "?", ":)": ":)", "¡Qué arte!": "What art!", "¡Qué buen mimo!": "What a great mime!",
		// juggler
		"¡Uy!": "Oops!", "Esto no me pasa en casa...": "This never happens at home...", "¡Gracias, eres un crack!": "Thanks, you're a star!",
		"¡Ahora sí!": "Now we're talking!", "¡Bravo!": "Bravo!", "¡Otra, otra!": "Encore, encore!", "¡Eso, ánimo!": "Come on, cheer up!",
		"Qué vergüenza...": "How embarrassing...", "Un momento, que se me ha escapado": "One moment, it got away from me", "Te debo una pelota": "I owe you a ball",
		"¡Ta-chán!": "Ta-da!",
		// buskers
		"¡Qué bien toca!": "He plays so well!", "¡Toca otra!": "Play another one!", "¡Eso es música!": "Now that's music!", "¡Qué voz!": "What a voice!",
		"Una moneda para el artista": "A coin for the artist", "Me ha puesto la piel de gallina": "It gave me goosebumps",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
