package main

// More lines for the speech bubbles: the same situations as the dialogue bank
// plus "chatter", the random one-liners people drop while standing around.

var jokeLines = DialogueBank{
	"people": {
		"dance_join":     {"¡Que no pare la música!", "Esta me la sé", "¡Hoy no trabajo mañana!", "¡A mover el esqueleto!", "¡Dejadme sitio, que vengo fuerte!", "Bailo fatal, pero con ganas"},
		"dance_leave":    {"Me duelen hasta las pestañas", "Voy a por otra copa", "Tengo que mirar el móvil", "Se me ha dormido el pie", "Voy a hidratarme (con ron)"},
		"dance_annoyed":  {"¿Tú de qué vas?", "¡Mis zapatos nuevos!", "¡Que esto es un baile, no un rodeo!", "¡Respeta mi espacio vital!", "¡Pisotón de campeonato!"},
		"greet":          {"¡Cuánto tiempo!", "¿Nos conocemos?", "¡Tú eres el del grupo de WhatsApp!", "¡Qué pintas!", "¡Hombre, tú por aquí!", "Hola, soy el primo de alguien"},
		"scared":         {"¡MAMÁÁÁ!", "¡Yo no he sido!", "¡Dejadme, que soy muy joven!", "¡Que viene, que viene!", "¡Corre, Forrest, corre!", "¡Ay, mi madre!", "¡Esto no estaba en el cartel!"},
		"saw_body":       {"¿Eso es kétchup? ...No.", "Ese ya no se levanta", "¡Se ha pasado con la fiesta!", "¡Hay uno tirado y no es de la borrachera!", "¿Alguien ha pedido una ambulancia?", "Ok, eso no es normal"},
		"drink":          {"¡Por los que no están!", "Que me lo apunten", "Esto no es agua, ¿verdad?", "Sólo una... (la décima)", "Cheers, chin-chin, kanpai", "Sabe a gloria y a gasolina"},
		"dance_with":     {"¡Eres el rey de la pista!", "¡Pegamos bien!", "Baila como si nadie mirase", "¡Qué arte, qué arte!", "Ni Shakira, oye"},
		"calling_police": {"¿Policía? Hay un loco suelto", "¡Alo, alo, emergencias!", "¡Que manden a todos!", "Sí, vengan, hay un problema... gordo", "¡Diga que traigan café!"},
		"accused":        {"¡Era ese, lo juro!", "¡Lo he visto con el cuchillo!", "¡Mira qué cara de asesino tiene!", "¡Ese tiene pinta rara!", "¡Detengan al sospechoso... del abrigo!"},
		"chatter": {
			"Creo que he perdido el móvil... o me lo han robado",
			"Me han dicho que habría pizza", "Mi ex está aquí. Mi ex y su nuevo ex", "Yo vine solo a por el hielo",
			"¿Alguien ha visto mis llaves?", "Esta canción me recuerda a mi divorcio", "He bailado tanto que ya no siento las piernas",
			"Dicen que hay un asesino, pero yo vengo por el catering", "¿Hay wifi?", "Mi madre cree que estoy en la biblioteca",
			"¡Qué buena está la fiesta!", "Me voy a casa. (No me voy a casa)", "¿Esto es una fiesta o un casting?",
			"He perdido un zapato y no sé dónde", "¿Quién ha puesto reguetón?", "Le he dicho a mi jefe que estaba enfermo",
			"Una vez bailé con un famoso. Era su doble", "Lo bueno de beber agua es que lo hacemos todos... ah no", "No me juzgues, es mi cuarta croqueta",
			"Hoy me siento imparable", "Si hay barra libre, yo me quedo a vivir", "Tengo mucho sueño pero mucho más ron",
			"¿A que no sabes qué me ha pasado hoy?", "¡Hay que repetir esto cada semana!", "¿Alguien tiene un cargador?",
			"Mi horóscopo decía «cuidado con los extraños»", "He leído que reír alarga la vida", "No tengo ni idea de quién cumple años",
			"Mi perro baila mejor que tú", "Esta es la mejor fiesta de mi vida (la anterior también)", "Qué raro, huele a quemado",
			"Me han invitado por error y me quedo", "Voy a hacer un brindis... pero luego", "¡Fotos, fotos! ¡Sonríe!",
			"Si me ves llorar, es por la cebolla del chiringuito", "El que no baila, que se tire a la piscina", "A mí me gustan las fiestas con final feliz",
			"Mañana empiezo la dieta", "Mi madre me dijo que no hablara con extraños; aquí estamos", "Hoy cuenta como cardio",
			"Cada vez que cierro los ojos, veo cócteles", "¿Vendrá la policía? Qué ilusión", "Te juro que no soy borracho, soy simpático",
		},
		"chatter_house":   {"Qué bonita la casa. ¿Es de alquiler?", "El baño está libre, creo", "¿Quién ha roto el jarrón? Yo no", "Este sofá es increíblemente cómodo"},
		"chatter_beach":   {"¡Agua! ¡Agua fría!", "¿Quién me echa crema en la espalda?", "He perdido el bañador. En serio", "Creo que me ha picado una medusa. O una avispa"},
		"chatter_highway": {"¡Cuidado, que vienen coches!", "A mí los camiones me ponen", "Cruzar es para valientes", "Yo cruzo con los ojos cerrados", "¿Cuánto tarda en pasar el siguiente?"},
		"chatter_station": {"¿Se oye algo en el espacio? ...No", "Esta fiesta está fuera de este mundo", "¡Gravedad cero, copas llenas!", "¿Alguien ha visto a mi alien?"},
		"chatter_boat":    {"Creo que me estoy mareando", "¡Yo sólo hago de grumete!", "¿Hay chalecos salvavidas? ...Para mí no", "Capitán, ¿hacia dónde navegamos?"},
		"chatter_casino":  {"¡Rojo, rojo, rojo!", "Me juego hasta la dignidad", "Esta es mi noche de suerte", "He perdido la camisa. Y los pantalones"},
		"chatter_snow":    {"¡Mis dedos! ¡Ya no tengo dedos!", "Este chocolate caliente es tóxico", "¿Quién ha pisado mi esquí?", "Me quedé congelado, literalmente"},
		"chatter_grave":   {"No me lo estoy pasando de muerte", "Hay mucha gente por aquí... ya sabéis", "¿Se mueve esa lápida?", "Yo no creo en fantasmas. Nada. Nada de nada"},
		"chatter_funfair": {"¡Quiero otra vuelta!", "Creo que he vomitado en la noria", "Me han estafado en la tómbola", "Vamos a la montaña rusa... otra vez"},
		"chatter_stadium": {"¡Gooool!", "Árbitro, ¡eres un...!", "¡Eso era penalti!", "¡Olé, olé, olé!"},
		"chatter_circuit": {"¡Vroom, vroom!", "Yo corro más que esos", "¡Pit stop de cervezas!", "Me han adelantado... a pie"},
		"chatter_cinema":  {"¡Shhh! ¡Que quiero ver la peli!", "¿Quién ha traído palomitas con ajo?", "Spoiler: el mayordomo", "Yo vine por el aire acondicionado"},
		"chatter_roof":    {"No mires hacia abajo", "Qué vistas... qué vértigo", "¿Y si saltamos? (Es broma)", "Esto no es seguro, ¿verdad?"},
	},
	"special_people": {
		"chatter": {
			"Firmo autógrafos... o servilletas", "Soy famoso, ¿no me reconoces?", "No, no hago fotos. Bueno, una", "Mi agente me ha dicho que viniera",
			"¡Que no me toquen el peinado!", "Mi equipo está en la puerta", "Hoy me siento con mucho personaje", "¡Esto sale en la prensa mañana!",
		},
	},
	"police": {
		"understood": {"RECIBIDO, CAMBIO", "YA VOY, YA VOY", "UNA SIRENA Y ME TENÉIS"},
		"halt":       {"¡POR LA AUTORIDAD QUE ME CONFIERE EL CHALECO!", "¡ESO ES UN DELITO... CREO!", "¡PÁRESE O LE PONGO UNA MULTA!"},
		"arrest":     {"¡TIENE DERECHO A UNA LLAMADA!", "¡Y A UN ABOGADO, SI TIENE!", "¡CAZADO!"},
		"tired":      {"¿HAY ALGÚN DONUT POR AQUÍ?", "ME DUELEN LOS PIES", "ESTO NO ESTABA EN LA OPOSICIÓN"},
	},
	"fbi": {
		"understood": {"LA VERDAD ESTÁ AHÍ FUERA", "CONFIRMADO. NO LO CONFIRMEN", "EL EXPEDIENTE ESTÁ ABIERTO"},
		"halt":       {"¡FBI! ¡SOLTÉ LA COPA!", "¡TENGO UNA PLACA Y UNA ACTITUD!", "¡QUIETO O LLAMO A MULDER!"},
		"arrest":     {"SOSPECHOSO EN EL SACO", "LE LEO SUS DERECHOS... TODOS", "CASO CERRADO. OTRO MÁS"},
		"tired":      {"¿ALGUIEN HA VISTO A SCULLY?", "NECESITO CAFÉ DE LABORATORIO", "ESTO ES UN X-FILE"},
	},
	"power_rangers": {
		"understood": {"¡AL ZORD, EQUIPO!", "¡TRANSFORMACIÓN LISTA!", "¡CONTACTO!"},
		"halt":       {"¡ES HORA DE MORFAR... Y DE MORDERTE!", "¡FIESTA TERMINADA!", "¡NO ES UN JUEGO! ...BUENO, UN POCO!"},
		"arrest":     {"¡VICTORIA PARA LOS RANGERS!", "¡UN VILLANO MENOS EN LA CIUDAD!", "¡A LA CELDA DEL ZORD!"},
		"tired":      {"¡NECESITO RECARGAR EL MORFIN!", "¡NO ERA TAN FÁCIL EN LA TELE!"},
	},
	"killers": {
		"pain":   {"¡AAAY!", "¡Eso me ha dolido!", "HNNNG"},
		"caught": {"Estaba a punto de cenar...", "¡Qué mala suerte!", "Lo que hay que ver..."},
	},
}

func mergeJokes(bank DialogueBank) {
	for cat, sits := range jokeLines {
		if bank[cat] == nil {
			bank[cat] = map[string][]string{}
		}
		for sit, lines := range sits {
			bank[cat][sit] = append(bank[cat][sit], lines...)
		}
	}
}

// chatterLine: something funny to say while standing around; sometimes it is
// about the place.
func (g *Game) chatterLine(gu *Guest) string {
	if g.rng.Float64() < 0.3 {
		key := "chatter_" + g.lvl.Def.Icon
		if lines := g.dialogue["people"][key]; len(lines) > 0 {
			return lines[g.rng.Intn(len(lines))]
		}
	}
	if gu.name != "" && g.rng.Float64() < 0.5 {
		return g.line("special_people", "chatter")
	}
	return g.line("people", "chatter")
}
