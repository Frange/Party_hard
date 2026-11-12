package main

// English text of the mosh pit, the slow dance and the stadium wave (pe_pogo.go, pe_parejas.go, pe_ola.go).
// TestPevTexts checks that every Spanish text they can show is here and fits (48 bytes).
func init() {
	for es, en := range map[string]string{
		// pogo
		"¡POGO, POGO!": "MOSH, MOSH!", "¡Que empiece el pogo!": "Let the mosh pit begin!", "¡A saltar todos!": "Everybody jump!",
		"¡Oi, oi, oi!": "Oi, oi, oi!", "¡Más fuerte!": "Harder!", "¡Hey, hey!": "Hey, hey!", "¡Arriba esas manos!": "Hands up!",
		"¡Pogo!": "Mosh!", "¡Ay, mi espalda!": "Ouch, my back!", "¡Hombre al suelo!": "Man down!", "¡Arriba, colega!": "Up you get, mate!",
		"¡Qué locos!": "They're nuts!", "Yo no me meto ahí": "I'm not going in there", "¡Dale!": "Go on!", "¡Uooo!": "Woooo!",
		"¡Eh! ¿Qué ha pasado?": "Hey! What happened?", "¡Abrid el círculo!": "Open the circle!", "¡Para, para!": "Stop, stop!",
		// slow dance
		"¡Ahora una lenta!": "Now a slow one!", "¡Las parejas a la pista!": "Couples to the floor!",
		"Esta va por los enamorados": "This one's for the lovebirds", "Qué bien bailas": "You dance so well", "No me pises...": "Don't step on me...",
		"Te quiero": "I love you", "¿Bailamos otra?": "One more dance?", "Mmm...": "Mmm...", "¡Qué bonito!": "How lovely!",
		"Ay, el amor...": "Ah, love...", "¡Que se besen!": "Kiss, kiss!", "Yo no tengo pareja...": "I've got no one to dance with...",
		"¿Cariño?": "Honey?", "¿Dónde te has ido?": "Where did you go?", "¡NOOOO!": "NOOOO!", "Snif... ¡no me dejes!": "Sniff... don't leave me!",
		"¡Mi amor!": "My love!",
		// stadium wave
		"¡LA OLA!": "THE WAVE!", "¡Todos a la vez!": "All together now!", "¡Ahí viene la ola!": "Here comes the wave!",
		"¡Uoooh!": "Whoooa!", "¡Eeeeh!": "Heeeey!", "¡Arriba!": "Up!", "¡Eh!": "Hey!", "¿Y ese?": "What about him?",
		"¡Se ha caído uno!": "Someone went down!", "¡Seguimos!": "Keep it going!", "¡Otra vez!": "Again!", "¡Otra, otra!": "Again, again!",
		"¡Qué grada!": "What a crowd!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
