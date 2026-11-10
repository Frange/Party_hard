package main

// The small news of the newspaper front page (newspaper_draw.go): funny made-up briefs (no real people) and a few
// that come from the party itself. The bank is chosen by the seed of the party, never twice on one page.

type paperBrief struct{ head, es, en string }

var paperBriefHeads = map[string]string{
	"SUCESOS": "INCIDENTS", "SOCIEDAD": "SOCIETY", "DEPORTES": "SPORTS", "TIEMPO": "WEATHER", "HORÓSCOPO": "HOROSCOPE",
	"GASTRONOMÍA": "FOOD", "CARTELERA": "LISTINGS", "OPINIÓN": "OPINION", "ANUNCIOS": "SMALL ADS", "CULTURA": "CULTURE",
	"VECINOS": "NEIGHBOURS", "ÚLTIMA HORA": "LATEST", "MÚSICA": "MUSIC", "CRONÓMETRO": "STOPWATCH", "BALANCE": "BALANCE",
	"TESTIGOS": "WITNESSES", "ESCÁNDALO": "SCANDAL", "DÚO DINÁMICO": "DYNAMIC DUO",
}

var paperBriefBank1 = []paperBrief{
	{"SUCESOS", "Avistan un pato en la piscina. Dice que solo pasaba por allí.", "Duck spotted in the pool. Says he was only passing by."},
	{"MÚSICA", "El DJ pide perdón por el último tema. Nadie sabe cuál fue.", "The DJ apologises for the last track. Nobody knows which one."},
	{"VECINOS", "Vecino se queja del ruido: nadie le oyó por la música.", "Neighbour complains about the noise: nobody heard him over the music."},
	{"DEPORTES", "El concurso de baile acaba en empate técnico y un tobillo.", "Dance contest ends in a technical draw and one ankle."},
	{"TIEMPO", "Noche despejada con riesgo de confeti en zonas bajas.", "Clear night with a risk of confetti in low-lying areas."},
	{"HORÓSCOPO", "Libra: hoy evita los canapés misteriosos. Y las escaleras.", "Libra: today avoid mystery canapés. And stairs."},
	{"GASTRONOMÍA", "La tortilla de la fiesta, sin cebolla, desata una guerra civil.", "Party omelette without onion starts a civil war."},
	{"CARTELERA", "Estrena esta noche: «Sillas Voladoras 2». Entrada libre.", "Opening tonight: \"Flying Chairs 2\". Free entry."},
	{"OPINIÓN", "¿Quién se llevó el último trozo de pizza? Exigimos respuestas.", "Who took the last slice of pizza? We demand answers."},
	{"ANUNCIOS", "Se vende sofá, poco usado. Ligeras manchas de ponche.", "Sofa for sale, barely used. Slight punch stains."},
	{"ANUNCIOS", "Busco abrigo marrón perdido en la fiesta. Respondo por Beto.", "Seeking brown coat lost at the party. Answers to Beto."},
	{"CULTURA", "Exposición de globos pinchados: cola de cinco minutos.", "Popped balloon exhibition: a five minute queue."},
	{"SOCIEDAD", "Cinco invitados juran haber visto un fantasma. Era el DJ.", "Five guests swear they saw a ghost. It was the DJ."},
	{"SUCESOS", "Desaparece una tarta de tres pisos. Sospechosos: todos.", "Three tier cake vanishes. Suspects: everyone."},
	{"DEPORTES", "Carrera de sillas: gana un jubilado con ruedas de repuesto.", "Chair race won by a pensioner with spare wheels."},
	{"TIEMPO", "Previsión: lluvia de serpentinas a partir de medianoche.", "Forecast: paper streamer showers after midnight."},
	{"HORÓSCOPO", "Escorpio: cuidado con quien te ofrezca «solo un sorbito».", "Scorpio: beware whoever offers you \"just a sip\"."},
	{"MÚSICA", "Un invitado pide «otra de reguetón» por 40ª vez. Sin éxito.", "A guest asks for \"one more reggaeton\" for the 40th time. No luck."},
	{"VECINOS", "La señora del tercero ofrece magdalenas a cambio de silencio.", "Lady from the third floor offers muffins in exchange for silence."},
	{"ÚLTIMA HORA", "Un globo escapa de la fiesta y pide asilo en la azotea.", "A balloon flees the party and asks for asylum on the roof."},
	{"GASTRONOMÍA", "El ponche de la casa gana el premio al más sospechoso.", "House punch wins the award for most suspicious."},
	{"SOCIEDAD", "Se busca pareja de baile con dos pies izquierdos. Yo, el otro.", "Dance partner wanted with two left feet. I have the other two."},
	{"CULTURA", "Concierto sorpresa del karaoke: la sala pide un bis. De silencio.", "Surprise karaoke gig: the crowd asks for an encore. Of silence."},
	{"CARTELERA", "Sesión golfa: «El Invitado Que No Se Iba». Dura toda la noche.", "Late show: \"The Guest Who Would Not Leave\". Runs all night."},
	{"OPINIÓN", "Defiendo la tortilla con cebolla y a quien me la quite.", "I defend omelette with onion and whoever takes it away."},
	{"SUCESOS", "Una cabra aparece en el jardín. Nadie la había invitado.", "A goat turns up in the garden. Nobody had invited it."},
	{"DEPORTES", "Piscina: el waterpolo con globos queda suspendido por pinchazos.", "Pool: balloon water polo suspended due to punctures."},
	{"TIEMPO", "Ola de calor en la pista de baile: 38 grados y subiendo.", "Heat wave on the dance floor: 38 degrees and rising."},
	{"HORÓSCOPO", "Géminis: hoy conocerás a alguien. Que no sea el del cuchillo.", "Gemini: today you will meet someone. Not the one with the knife."},
	{"ANUNCIOS", "Regalo karaoke con cinco canciones. Solo suena una.", "Free karaoke machine with five songs. Only one plays."},
	{"VECINOS", "El portero asegura que nunca había visto tanta gente en el portal.", "The doorman says he never saw so many people in the lobby."},
	{"MÚSICA", "La banda se disculpa: «Era jazz». Nadie lo notó.", "The band apologises: \"It was jazz\". Nobody noticed."},
	{"ÚLTIMA HORA", "El hielo se acaba. Los refrescos, tibios. La moral, alta.", "Ice has run out. Drinks lukewarm. Morale high."},
	{"SOCIEDAD", "Un invitado se disfraza de lámpara y gana el concurso.", "A guest dresses as a lamp and wins the contest."},
	{"GASTRONOMÍA", "Receta del día: cómo servir canapés sin que nadie desaparezca.", "Recipe of the day: serving canapés without anyone vanishing."},
	{"CULTURA", "Taller de baile en grupo: se pide no pisar a quien enseña.", "Group dance workshop: please do not step on the teacher."},
	{"SUCESOS", "Se reporta un sombrero huido que no responde a su nombre.", "A runaway hat is reported. It does not answer to its name."},
	{"DEPORTES", "Pulso a brazo partido: el ganador se lleva el brazo.", "Arm wrestling: the winner takes home the arm."},
	{"TIEMPO", "Niebla en la cocina. Es el humo de las croquetas.", "Fog in the kitchen. It is the croquette smoke."},
	{"ANUNCIOS", "Cambio piano por confeti. Tratar con el que baila en el sofá.", "Swap piano for confetti. Ask the one dancing on the sofa."},
	{"OPINIÓN", "Cartas al director: «Bajad la música». Respuesta: «¿Qué?»", "Letters to the editor: \"Turn the music down\". Reply: \"What?\""},
	{"VECINOS", "Los del cuarto dicen que ellos no han sido. Nadie preguntó.", "The folks on the fourth floor say it was not them. Nobody asked."},
	{"ÚLTIMA HORA", "Baja el precio del aperitivo: ya cuesta lo que vale, nada.", "Appetizer price drops: it now costs what it is worth, nothing."},
	{"CARTELERA", "Teatro: «Siete Invitados y Una Ventana». Función agotada.", "Theatre: \"Seven Guests and One Window\". Sold out."},
	{"SOCIEDAD", "Matrimonio feliz tras conocerse en el guardarropa.", "Happy couple after meeting in the cloakroom."},
	{"MÚSICA", "Se busca triangulista urgente para orquesta de barrio.", "Urgently wanted: triangle player for neighbourhood orchestra."},
	{"GASTRONOMÍA", "Pepinillo sospechoso hallado en la ensaladilla. Se investiga.", "Suspicious pickle found in the potato salad. Under investigation."},
}

// the whole bank: this file's and the second one's (newspaper_briefs2.go)
var paperBriefBank = append(append([]paperBrief{}, paperBriefBank1...), paperBriefBank2...)

func init() {
	add := func(k, v string) {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, b := range paperBriefBank1 {
		add(b.es, b.en)
	}
	for k, v := range paperBriefHeads {
		add(k, v)
	}
}
