package main

// English texts of the boat and balloon interludes (interlude_boat.go, interlude_balloon.go). They are merged into
// langDataEN at start-up without overwriting anything that lang_data.go already has.
var langInterludesEN = map[string]string{
	// the boat
	"Una barca, un mar y cero planes.": "A boat, a sea and zero plans.",
	"El casco tiene otra opinión.":     "The hull has other ideas.",
	"Y llega la visita.":               "And here comes a visitor.",
	"Taxi tiburón, servicio exprés.":   "Shark taxi, express service.",
	"Qué paz, qué calma...":            "So peaceful, so calm...",
	"¿Eso... es agua?":                 "Is that... water?",
	"¡Achica, achica!":                 "Bail, bail!",
	"¿Eso es una aleta?":               "Is that a fin?",
	"¡Se ha comido mi cubo!":           "It ate my bucket!",
	"¿Nos empuja? ¡Qué majo!":          "It's pushing us? How nice!",
	"¡Gracias, tiburón!":               "Thanks, shark!",
	"Con calma. Remo yo.":              "Easy. I'll row.",
	"¡¡Hay una fuga!!":                 "There's a leak!!",
	"Tú achica, yo remo.":              "You bail, I row.",
	"¡¡TIBURÓN!!":                      "SHARK!!",
	"Será un delfín.":                  "Probably a dolphin.",
	"¡Mi cubo! ¡Era nuevo!":            "My bucket! It was new!",
	"Nos empuja. Qué amable.":          "It's pushing us. How kind.",
	"¡BONK!":                           "BONK!",
	"¡ÑAM!":                            "CHOMP!",
	"¡SPLASH!":                         "SPLASH!",
	// the balloon
	"El medio de transporte más fiable.": "The most reliable way to travel.",
	"Qué altura. Qué silencio.":          "So high. So quiet.",
	"Un pájaro con opiniones.":           "A bird with opinions.",
	"Propulsión a pedorreta.":            "Raspberry-powered.",
	"Despegue con estilo.":               "Take-off in style.",
	"¡Cuerda cortada!":                   "Rope cut!",
	"Qué vistas... qué paz.":             "What a view... so peaceful.",
	"¿Eso era un pájaro?":                "Was that a bird?",
	"¡Tira peso, tira peso!":             "Drop weight, drop weight!",
	"¡Vienen árboles!":                   "Trees ahead!",
	"¡Corta ya la cuerda!":               "Cut the rope already!",
	"Ya está. Cortada.":                  "Done. Cut.",
	"Qué vistas. Qué paz.":               "What a view. So peaceful.",
	"¡¡UN PÁJARO!!":                      "A BIRD!!",
	"Tiremos lastre. Con calma.":         "Let's drop ballast. Calmly.",
	"¡¡Tira la nevera!!":                 "Throw the cooler!!",
	"Árboles. Agáchate.":                 "Trees. Duck.",
	"¡PLOP!":                             "POP!",
	"¡PSSSSS!":                           "PSSSSS!",
	"¡FUERA!":                            "OVERBOARD!",
	"¡FROOOSH!":                          "SWOOSH!",
	"¡PRRRRR!":                           "PRRRRR!",
	"¡PFRRRR!":                           "PFRRRR!",
}

func init() {
	for k, v := range langInterludesEN {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

// the balloon in the list of scenes and in the DEBUG menu
func init() {
	for k, v := range map[string]string{
		"INTERLUDIO: GLOBO A LA DERIVA": "INTERLUDE: DRIFTING BALLOON",
		"Un globo aerostático con cesta, sacos de lastre y un pinchazo muy mal momento. Baja más rápido de lo que sube.": "A hot air balloon with basket, sandbags and a puncture at the worst moment. It goes down faster than it goes up.",
		"¿ESO FLOTA?": "DOES THAT FLOAT?",
		"Más allá de la fiesta número 15 hay cielo. Y poco aire.": "Past party number 15 there is sky. And little air.",
		"Ver el interludio del globo (tras la fiesta 15)":         "Watch the balloon interlude (after party 15)",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
