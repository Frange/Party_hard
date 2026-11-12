package main

// English text and newspaper briefs of the street painter and the sack race (pe_graffiti.go, pe_sackrace.go).
// TestPevTexts checks that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// street painter
		"Grafitero": "Street painter", "Esta pared pide color": "This wall needs colour", "Arte urbano, señores": "Street art, folks",
		"Un momento, que me inspiro": "Hold on, I'm inspired", "¡Lienzo libre!": "Free canvas!",
		"*fsssssh*": "*pssssssh*", "*chk chk fsssh*": "*clack clack pssh*",
		"¡Obra maestra!": "Masterpiece!", "Firmado: el artista": "Signed: the artist", "Esto vale millones": "This is worth millions",
		"¡Arte para el pueblo!": "Art for the people!",
		// sack race
		"Carrera de sacos": "Sack race", "¡Carrera de sacos!": "Sack race!", "¡Yo me apunto!": "Count me in!",
		"¡Esta la gano yo!": "This one's mine!", "¡Preparados, listos, ya!": "Ready, steady, go!",
		"¡Salto, salto, salto!": "Hop, hop, hop!", "¡Vamos, vamos!": "Come on, come on!",
		"¡Corre, saco de patatas!": "Run, you sack of spuds!", "¡Que te pillan!": "They're catching up!",
		"¡Ay, mi saco!": "Ouch, my sack!", "¡Me he tropezado!": "I tripped!", "¡Esto no vale!": "That doesn't count!",
		"¡JAJAJA!": "HAHAHA!", "¡Se ha comido el suelo!": "He kissed the floor!",
		"¡He ganado!": "I won!", "¡Campeón del saco!": "Sack champion!", "¡Soy un canguro!": "I'm a kangaroo!",
		"¡Me ha empujado!": "He pushed me!", "¡Revancha!": "Rematch!", "¡Casi!": "So close!",
		"¡Mi saco tenía un agujero!": "My sack had a hole!", "¡¿Qué ha pasado?!": "What just happened?!",
		"¡Parad la carrera!": "Stop the race!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
	registerPaperEvent("graffiti", "CULTURA", []paperTxt{
		tx("Un grafitero decoró la fachada en plena fiesta. El dueño duda entre llorar o enmarcarla.", "A street painter decorated the facade mid-party. The owner can't decide: cry or frame it."),
		tx("Aparece un graffiti nuevo en la pared. Los críticos hablan de «una mirada fresca».", "A new graffiti appears on the wall. Critics speak of \"a fresh look\"."),
		tx("«Arte para el pueblo», firmó el autor del mural. El pueblo pidió otra ronda.", "\"Art for the people\", signed the author of the mural. The people asked for another round."),
	})
	registerPaperEvent("sacos", "DEPORTES", []paperTxt{
		tx("Carrera de sacos en la fiesta: el ganador ya pide que sea deporte olímpico.", "Sack race at the party: the winner wants it to become an Olympic sport."),
		tx("Un invitado se tropezó con su propio saco y aun así cruzó la meta. Ovación.", "A guest tripped on his own sack and still crossed the line. Standing ovation."),
		tx("«Me ha empujado», dijo el segundo de la carrera de sacos. El jurado lo estudia.", "\"He pushed me\", said the runner-up of the sack race. The jury is looking into it."),
	})
}
