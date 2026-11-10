package main

// English text of the life of the party (partylife.go): selfies, dance contest, pizza boy, mascot, DJ booth.
func init() {
	for es, en := range map[string]string{
		"¡PATATA!":                        "CHEESE!",
		"¡ESA SÍ!":                        "THAT ONE!",
		"¡POR FIN!":                       "FINALLY!",
		"¡SUBE ESO!":                      "TURN IT UP!",
		"¿QUÉ ES ESTO?":                   "WHAT IS THIS?",
		"¿UN VALS?":                       "A WALTZ?",
		"¡PONED OTRA!":                    "PLAY SOMETHING ELSE!",
		"El DJ está liado con los discos": "The DJ is busy with his records",
		"Uy: el DJ se equivoca de disco":  "Oops: the DJ grabbed the wrong record",
		"El DJ te hace caso y cambia la canción": "The DJ listens and changes the song",
		"¡OLÉ!":                                 "OLE!",
		"¡TOMA ESO!":                            "TAKE THAT!",
		"¡MIRAD!":                               "LOOK AT ME!",
		"¡WOO!":                                 "WOO!",
		"¡HE GANADO!":                           "I WON!",
		"¡FRAUDE!":                              "RIGGED!",
		"BUUU":                                  "BOOO",
		"¡Concurso de baile en la pista!":       "Dance contest on the floor!",
		"¡Hay ganador en el concurso de baile!": "We have a dance contest winner!",
		"¡Gracias! Toma, invita la casa":        "Thanks! Take this, it's on the house",
		"¡Buen provecho!":                       "Enjoy!",
		"Mejor llamo al cliente...":             "I'd better call the customer...",
		"Pizza para el 14B... ¿alguien?":        "Pizza for 14B... anybody?",
		"¿Es aquí el 14B?":                      "Is this 14B?",
		"Le indicas el camino y te regala una pizza": "You point the way and he gives you a pizza",
		"Repartidor":                         "Delivery guy",
		"Acaricias al gato de la fiesta":     "You pet the party cat",
		"Saludas al cangrejo de la fiesta":   "You greet the party crab",
		"Acaricias al pingüino de la fiesta": "You pet the party penguin",
		"Alimentas al pato de la fiesta":     "You feed the party duck",
		"La mascota de la fiesta ya no está": "The party mascot is gone",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
