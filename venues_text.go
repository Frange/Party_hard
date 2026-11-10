package main

// ---------------------------------------------------------------------------
// Texts and per-party tables of the three parties that come before the space station (campaign.go): the BIRTHDAY
// party ("cake"), the HOCKEY rink ("hockey") and the 70s DISCO ("disco70"). They are merged from init() so that the
// shared tables (genres, kinds of party, hints, venue lines) keep their own files untouched. Generic parody only.
// ---------------------------------------------------------------------------

func init() {
	// music (genres.go): the neighbours of each party in the campaign never repeat its style
	for t, id := range map[string]string{"FIESTA DE CUMPLEAÑOS": "pop", "PISTA DE HOCKEY": "rock", "DISCO SETENTERA": "disco"} {
		genreOfTitle[t] = id
	}
	// the silly name of each party (partytypes.go)
	for t, s := range map[string]string{
		"FIESTA DE CUMPLEAÑOS": "Soplavelas sin control",
		"PISTA DE HOCKEY":      "Prórroga sin árbitro",
		"DISCO SETENTERA":      "Fiebre de pata de elefante",
	} {
		partyTypeOfTitle[t] = s
	}
	// indoors for the party events (partyevents.go): the living room, the rink and the disco hall
	for _, id := range []string{"cake", "hockey", "disco70"} {
		partyIndoorIcons[id] = true
	}
	// the goal and the tip of the title card (levelhint.go)
	hintThemes["cake"] = hintTheme{"Cumpleaños: que nadie sople las velas", "Entre globos y regalos hay rincones ciegos"}
	hintThemes["hockey"] = hintTheme{"Hockey: la pista es tuya, el hielo no", "En el hielo se patina: frena antes de la banda"}
	hintThemes["disco70"] = hintTheme{"Disco setentera: aquí el que baila, cae", "Las baldosas brillan: mézclate con la gente"}
	// what the guests say about the place (venuelines.go): 6 lines each
	venueLines["cake"] = []venueLine{
		{"¡Que sople, que sople la vela grande!", "Let them blow out the big candle!"},
		{"Yo vine solo por la tarta. Y por la tarta", "I only came for the cake. And the cake"},
		{"Este gorro me aprieta, pero me veo genial", "This hat pinches, but I look great"},
		{"¿Alguien ha visto el regalo del cumpleañero?", "Has anyone seen the birthday gift?"},
		{"Los globos ya van por el segundo pinchazo", "The balloons are on their second pop"},
		{"Cumplo treinta y tantos... otra vez", "I turn thirty-something... again"},
	}
	venueLines["hockey"] = []venueLine{
		{"¡Ese árbitro necesita gafas!", "That referee needs glasses!"},
		{"Patino fatal, pero con mucha dignidad", "I skate badly, but with great dignity"},
		{"Mi trasero y el hielo ya son amigos", "My backside and the ice are friends now"},
		{"¡Pasa el disco! ¿Qué disco? ¿Dónde?", "Pass the puck! What puck? Where?"},
		{"Vine por la pelea, no por el partido", "I came for the fight, not the game"},
		{"Hace un frío que pela y nadie se baja", "It is freezing and nobody gets off the ice"},
	}
	venueLines["disco70"] = []venueLine{
		{"¡Esta pata de elefante no me deja bailar!", "These bell-bottoms will not let me dance!"},
		{"Mi afro y yo mandamos en esta pista", "My afro and I rule this floor"},
		{"La bola de espejos me guiña un ojo", "The mirror ball is winking at me"},
		{"¿Esa baldosa dibuja una cara o es el ponche?", "Is that tile a face, or is it the punch?"},
		{"Sábado noche y mis patillas piden pista", "Saturday night and my sideburns want a floor"},
		{"Esa lámpara de lava me está mirando", "That lava lamp is staring at me"},
	}
	// English
	for k, v := range map[string]string{
		"FIESTA DE CUMPLEAÑOS": "BIRTHDAY PARTY",
		"PISTA DE HOCKEY":      "HOCKEY RINK",
		"DISCO SETENTERA":      "70s DISCO",

		"Tarta gigante, globos y un invitado de más: ¡a soplar las velas!": "Giant cake, balloons and one guest too many: blow out the candles!",
		"Hielo, palos y una grada que pide pelea. Hoy el disco eres tú.":   "Ice, sticks and a crowd that wants a fight. Today you are the puck.",
		"Pantalones de campana, patillas y una pista que se ilumina sola.": "Bell-bottoms, sideburns and a floor that lights itself up.",

		"Soplavelas sin control":     "Candle-blowing mayhem",
		"Prórroga sin árbitro":       "Overtime without a referee",
		"Fiebre de pata de elefante": "Bell-bottom fever",

		"Cumpleaños: que nadie sople las velas":          "Birthday: let nobody blow out the candles",
		"Entre globos y regalos hay rincones ciegos":     "Balloons and presents make blind corners",
		"Hockey: la pista es tuya, el hielo no":          "Hockey: the rink is yours, the ice is not",
		"En el hielo se patina: frena antes de la banda": "You skate on ice: brake before the boards",
		"Disco setentera: aquí el que baila, cae":        "70s disco: whoever dances, falls",
		"Las baldosas brillan: mézclate con la gente":    "The tiles glow: blend in with the crowd",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, id := range []string{"cake", "hockey", "disco70"} {
		for _, l := range venueLines[id] {
			if _, ok := langDataEN[l.es]; !ok {
				langDataEN[l.es] = l.en
			}
		}
	}
}

// pVenue: one brief of the newspaper that only the parties of the three new venues can print (newspaper.go).
var pVenue = map[string][]paperTxt{
	"cake": {
		tx("La tarta gigante duró once minutos. La fiesta, bastante menos.", "The giant cake lasted eleven minutes. The party, quite a bit less."),
		tx("El cumpleañero no sopló las velas: se las apagaron por él.", "The birthday guest never blew out the candles: someone did it for him."),
		tx("El mago del cumpleaños dice que el truco era hacer desaparecer invitados.", "The birthday magician says the trick was making guests disappear."),
		tx("Los globos sobrevivieron. Los gorros de cono, no tanto.", "The balloons survived. The cone hats, not so much."),
		tx("Entre los regalos había una tarta y una sorpresa bastante peor.", "Among the presents there was a cake and a much worse surprise."),
	},
	"hockey": {
		tx("El partido se suspendió por una pelea. Nadie sabe quién empezó, todos saben quién acabó.", "The match was stopped by a brawl. Nobody knows who started it, everybody knows who finished it."),
		tx("Un invitado patinó diez metros sin querer. Dice que fue su mejor actuación.", "A guest skated ten meters by accident. He says it was his best performance."),
		tx("El árbitro no vio nada. Para eso tiene el silbato.", "The referee saw nothing. That is what the whistle is for."),
		tx("La máquina de alisar el hielo quedó aparcada, sin culpa y con testigos.", "The ice resurfacer stayed parked, blameless and with witnesses."),
		tx("Los jugadores pidieron prórroga, pero la fiesta ya había terminado.", "The players asked for overtime, but the party was already over."),
	},
	"disco70": {
		tx("La bola de espejos lo vio todo. Los espejos no mienten, pero tampoco declaran.", "The mirror ball saw everything. Mirrors do not lie, but they do not testify either."),
		tx("Las baldosas dibujaron un corazón justo cuando ocurrió lo peor.", "The floor tiles drew a heart right when the worst happened."),
		tx("Un afro enorme escondió a tres testigos. Se investiga el peinado.", "A huge afro hid three witnesses. The hairdo is under investigation."),
		tx("Los pantalones de campana complicaron la huida de más de un invitado.", "Bell-bottoms made the escape harder for more than one guest."),
		tx("La lámpara de lava no se inmutó, como siempre.", "The lava lamp did not flinch, as always."),
	},
}

func init() {
	for _, pool := range pVenue {
		for _, p := range pool {
			if _, ok := langDataEN[p.es]; !ok {
				langDataEN[p.es] = p.en
			}
		}
	}
}
