package main

// ---------------------------------------------------------------------------
// The lines of the venue acts (venueacts.go), Spanish with their English in langDataEN (merged from init so that
// lang_data.go stays untouched). Every bubble <= 48 characters in both languages (tandah_test.go).
// The insult duels use the classic insult and comeback pairs of the pirate adventure games, as the owner asked
// (2026-11-12): the lines themselves, never the names of the characters.
// ---------------------------------------------------------------------------

type insultPair struct{ insult, reply, insultEN, replyEN string }

var insultPairs = []insultPair{
	{"¡Peleas como un granjero!", "¡Qué apropiado! Tú peleas como una vaca.",
		"You fight like a dairy farmer!", "How appropriate. You fight like a cow."},
	{"Una vez tuve un perro más listo que tú.", "Te habrá enseñado todo lo que sabes.",
		"I once owned a dog that was smarter than you.", "He must have taught you everything you know."},
	{"Me das ganas de vomitar.", "Me haces pensar que alguien ya lo hizo.",
		"You make me want to puke.", "You make me think somebody already did."},
	{"La gente cae a mis pies al verme llegar.", "¿Incluso antes de que huelan tu aliento?",
		"People fall at my feet when they see me coming.", "Even BEFORE they smell your breath?"},
	{"No hay palabras para lo asqueroso que eres.", "Sí que las hay, solo que nunca las aprendiste.",
		"There are no words for how disgusting you are.", "Yes there are. You just never learned them."},
	{"¡Mi pañuelo limpiará tu sangre!", "¿Al final te dieron el puesto de conserje?",
		"My handkerchief will wipe up your blood!", "So you got that job as janitor, after all."},
	{"He oído que eras un rastrero despreciable.", "Lástima que nadie haya oído hablar de TI.",
		"I've heard you were a contemptible sneak.", "Too bad no one's ever heard of YOU at all."},
	{"No eres rival para mi cerebro, pobre tonto.", "Estaría en apuros si alguna vez lo usaras.",
		"You're no match for my brains, you poor fool.", "I'd be in real trouble if you ever used them."},
}

var (
	swordStart   = pevTexts("¡En guardia, rufián!", "¡Prepárate a morder el polvo!", "¡Por mi pata de palo!")
	swordLose    = pevTexts("¡Me rindo! ¡Me rindo!", "Vale, vale... tú ganas", "¡Mi espada era de goma!")
	snowHit      = pevTexts("¡Toma bola!", "¡En toda la cara!", "¡Brrr, qué frío!", "¡Esta me la pagas!")
	snowmanLines = pevTexts("Le falta la zanahoria", "Mi muñeco tendrá sombrero", "¡El mejor muñeco de nieve!")
	juggleLines  = pevTexts("¡Tres bolas y ni una al suelo!", "¡Mirad, sin manos! Bueno, con manos", "¡Aplausos, por favor!")
	balloonLines = pevTexts("¡Un perrito de globo!", "¿Quién quiere una jirafa?", "¡Este es una espada!")
	squirtLines  = pevTexts("¡Huele mi flor!", "¡Sorpresa!")
	squirtVictim = pevTexts("¡Eh, que me mojas!", "¡Mi camisa nueva!", "¡Payaso!")
	shotLines    = pevTexts("¡Cañonazo!", "¡Toma disco!", "¡Al ángulo!")
	shotHurt     = pevTexts("¡Ay, mi diente!", "¡Que esto no es la pista!", "¡Ay, ay, ay!")
	shotOops     = pevTexts("Uy... penalti", "Eso no ha sido falta", "¡Árbitro, no he sido yo!")
)

func init() {
	for k, v := range map[string]string{
		"¡En guardia, rufián!":                              "En garde, scoundrel!",
		"¡Prepárate a morder el polvo!":                     "Prepare to bite the dust!",
		"¡Por mi pata de palo!":                             "By my wooden leg!",
		"¡Me rindo! ¡Me rindo!":                             "I give up! I give up!",
		"Vale, vale... tú ganas":                            "Fine, fine... you win",
		"¡Mi espada era de goma!":                           "My sword was made of rubber!",
		"¡Toma bola!":                                       "Snowball, take that!",
		"¡En toda la cara!":                                 "Right in the face!",
		"¡Brrr, qué frío!":                                  "Brrr, that's cold!",
		"¡Esta me la pagas!":                                "You'll pay for that one!",
		"Le falta la zanahoria":                             "It still needs the carrot",
		"Mi muñeco tendrá sombrero":                         "My snowman gets a hat",
		"¡El mejor muñeco de nieve!":                        "The best snowman ever!",
		"¡Tres bolas y ni una al suelo!":                    "Three balls and none on the floor!",
		"¡Mirad, sin manos! Bueno, con manos":               "Look, no hands! Well, with hands",
		"¡Aplausos, por favor!":                             "Applause, please!",
		"¡Un perrito de globo!":                             "A balloon doggie!",
		"¿Quién quiere una jirafa?":                         "Who wants a giraffe?",
		"¡Este es una espada!":                              "This one is a sword!",
		"¡Huele mi flor!":                                   "Smell my flower!",
		"¡Sorpresa!":                                        "Surprise!",
		"¡Eh, que me mojas!":                                "Hey, you're soaking me!",
		"¡Mi camisa nueva!":                                 "My new shirt!",
		"¡Payaso!":                                          "Clown!",
		"¡Cañonazo!":                                        "Slapshot!",
		"¡Toma disco!":                                      "Puck for you!",
		"¡Al ángulo!":                                       "Top corner!",
		"¡Ay, mi diente!":                                   "Ow, my tooth!",
		"¡Que esto no es la pista!":                         "This isn't the rink!",
		"¡Ay, ay, ay!":                                      "Ow, ow, ow!",
		"Uy... penalti":                                     "Oops... penalty",
		"Eso no ha sido falta":                              "That was no foul",
		"¡Árbitro, no he sido yo!":                          "Ref, it wasn't me!",
		"¡MEC MEC!":                                         "HONK HONK!",
		"¡Una bola de nieve gigante baja por la pista!":     "A giant snowball is rolling down the slope!",
		"La bola de nieve se ha llevado a varios invitados": "The snowball took a few guests with it",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, p := range insultPairs {
		for es, en := range map[string]string{p.insult: p.insultEN, p.reply: p.replyEN} {
			if _, ok := langDataEN[es]; !ok {
				langDataEN[es] = en
			}
		}
	}
}
