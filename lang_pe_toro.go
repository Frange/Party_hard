package main

// English text and newspaper briefs of the mechanical bull and the street band (events_toro.go, pe_murga.go).
// TestPevTexts checks that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// mechanical bull
		"¡Me toca el toro!": "My turn on the bull!", "¡Yo también monto!": "I'm riding too!", "¡A la cola del toro!": "Bull line, this way!",
		"¡Esta vez aguanto!": "This time I'll last!", "¡Uf, qué cola! Paso": "Ugh, what a line! I'm out", "Con cuatro ya va lleno": "Four in line is plenty",
		"¡Yihaa!": "Yeehaw!", "¡Que se cae!": "He's falling!", "¡Más rápido!": "Faster!", "¡Agárrate fuerte!": "Hold on tight!",
		"¡Aguanta, aguanta!": "Hang on, hang on!", "¡Dale caña al toro!": "Crank up the bull!", "¡Yihaaa!": "Yeehaaaw!", "¡Socorro!": "Help!",
		"¡Esto no para!": "This thing won't stop!", "¡Soy un vaquero!": "I'm a cowboy!", "¡Ay, mi espalda!": "Ouch, my back!", "¡JAJAJA!": "HAHAHA!",
		"¡Otra vez!": "Again!", "¡He volado!": "I flew!", "¡Lo he domado!": "I tamed it!", "¡Ni me he despeinado!": "Not a hair out of place!",
		"¡Bravo!": "Bravo!",
		// street band
		"¡Que suene la murga!": "Strike up the band!", "¡Pom, pom, chis!": "Boom, boom, crash!", "¡Todos detrás de la murga!": "Everybody follow the band!",
		"¡Esa cacerola!": "Go, saucepan!", "¡Más ritmo!": "More rhythm!", "¡Yo me apunto!": "Count me in!", "¡Qué escándalo!": "What a racket!", "¡Esto es una fiesta!": "Now this is a party!",
		"¡Bravo, murga!": "Bravo, band!", "¡Hasta la próxima esquina!": "See you at the next corner!", "¡Me duelen las orejas!": "My ears hurt!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
	r := registerPaperEvent
	r("toro", "SOCIEDAD", []paperTxt{
		tx("Un toro mecánico tiró a medio invitado por los aires. El colchón aguantó.", "A mechanical bull sent half the guests flying. The mattress held up."),
		tx("Cola de cuatro para montar al toro mecánico. Récord: veinte segundos.", "A line of four for the mechanical bull. Record: twenty seconds."),
		tx("El toro mecánico ganó por goleada: los invitados, al colchón.", "The mechanical bull won by a mile: the guests, onto the mattress."),
	})
	r("murga", "CULTURA", []paperTxt{
		tx("Una murga de cubo y cacerola desfiló por la calle. Los vecinos, despiertos.", "A bucket-and-saucepan band marched down the street. The neighbours: awake."),
		tx("Una murga improvisada arrastró a media fiesta calle abajo.", "An improvised street band dragged half the party down the road."),
		tx("Kazoo, flauta de plástico y un bombo de cubo: la murga del año.", "Kazoo, plastic flute and a bucket drum: the band of the year."),
	})
}
