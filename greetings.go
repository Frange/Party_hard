package main

// People close to the player sometimes say hello or something silly. Not
// everybody (about a third of the guests are talkative) and not all the time.

var greetLines = []string{
	"¡Hola!", "¡Buenas!", "¿Qué tal?", "¡Menuda fiesta!", "¡Ey, tú!", "¿Tú también por aquí?", "¡Qué calor hace!", "¿Has visto la barra?",
	"Yo solo vine por el ponche.", "¡Salud!", "Me duelen los pies...", "¿Tienes fuego?", "No conozco a nadie aquí.", "¿Esa no es tu canción?", "¡Qué ruido!",
	"Tengo hambre.", "¿Bailas?", "Cuidado con el suelo, está pegajoso.", "¡Viva la fiesta!", "Esto es mejor que mi trabajo.", "Hola, ¿nos conocemos?",
	"No le digas a mi jefe que estoy aquí.", "¿Has visto mis llaves?", "¡Qué buen sitio!", "Jeje, ¿cómo va?", "Me encanta tu ropa.",
	"¿Dónde está el baño?", "Cuidado con los guardias.", "¡Qué noche!", "Yo ya no bebo... más.",
}

var catchphrases = map[string][]string{
	"homer": {"¡D'ohh!", "Mmm... rosquillas.", "¡Cerveza!"}, "bart": {"¡Ay, caramba!", "¡Cómete mis pantalones!"}, "marge": {"Mmmmh...", "Jomer, por favor..."},
	"lisa": {"Eso no es correcto.", "Soy más lista que tú."}, "burns": {"Excedente...", "Smitters, suelte a los perros."}, "flanders": {"¡Hola, vecinerino!", "¡Qué diddly bien!"},
	"cartman": {"¡Respeto mi autoridad!", "¡Me largo a casa!"}, "kenny": {"¡Mmmph mmph!", "Mph mph mph."}, "kyle": {"¡Eso no es justo!"}, "stan": {"Hostia, tío.", "Esto es una locura."},
	"fry": {"¡Cállate y toma mi dinero!", "No sé qué es eso."}, "leela": {"¡Fray, no!"}, "bender": {"¡Muerde mi brillante culo metálico!", "¡Una cerveza!"}, "farnsworth": {"¡Buenas noticias, a todos!"},
	"zoidberg": {"¡Woop woop woop!", "¿Por qué yo?"}, "peter": {"¡Jejejeje!"}, "stewie": {"¡Mundo cruel!", "Victoria será mía."},
	"rick": {"*burp* Wubba lubba dub dab.", "Nada importa, Morti."}, "morty": {"¡Oh, cielos!"},
	"wally": {"¿Me has encontrado?", "Shhh, escóndete conmigo."}, "mortadelo": {"¡Soy un armario!", "Ya está ahí el jefe..."}, "filemon": {"¡Mortadelaaaa!"},
	"asterix": {"¡Por Tutatix!"}, "obelix": {"¡Están locos estos romanos!", "¿Hay jabalí?"}, "panoramix": {"Esta poción es muy fuerte."},
	"sonic": {"¡Demasiado lento!", "¡Voy a toda velocidad!"}, "tails": {"¡Espérame, Zonic!"}, "knuckles": {"¡Esta esmeralda es mía!"},
	"mario": {"¡It's-a mee, Marrio!", "¡Mama mia!"}, "luigi": {"¡Mama mia!", "No me dejes solo."}, "wario": {"¡Wahaha!"},
	"luffy": {"¡Voy a ser el Rey de los Piratas!", "¡Tengo hambre!"}, "zoro": {"¿Dónde estoy?", "¿Por dónde era?"}, "sanji": {"¡Mi amor!", "Cocinaré para ti."}, "nami": {"¿Quién paga?", "Eso son 100.000 berries."},
	"goku": {"¡Kame jame ja!", "¡Tengo hambre!"}, "vegeta": {"¡Soy el príncipe de los Sayans!"}, "naruto": {"¡Datebayo!", "¡Voy a ser Hokaje!"},
	"batman": {"Soy Buttman.", "Soy la venganza."}, "superman": {"¡Arriba, arriba!", "¿Es un pájaro?"}, "spiderman": {"¡Con gran poder...!", "Amistoso vecino."},
	"ironman": {"Soy Ironmon.", "Jarbis, ¿qué hora es?"}, "thor": {"¡Por Odín!", "¿Otra ronda?"}, "hulk": {"¡BULK APLASTA!"}, "deadpool": {"¡Hey, tú, el que lees!", "Maximum effort."},
	"terminator": {"Volveré.", "Hasta la pista, baby."}, "robocop": {"Tienes 20 segundos para cumplir.", "Muerto o vivo."}, "neo": {"Whoa.", "No hay cuchara."},
	"john_wick": {"Yeah.", "Mi perro..."}, "yoda": {"Hazlo o no lo hagas.", "Mucho que aprender tienes."}, "vader": {"Yo soy tu madre.", "No me falta fe."}, "chewbacca": {"¡Uuurrr!"},
	"han_solo": {"Disparé yo primero.", "Tengo un mal presentimiento."}, "luke": {"Tengo un mal presentimiento."}, "kirk": {"¡Sullu, adelante!", "¡Spook!"}, "spock": {"Fascinante.", "Es ilógico."},
	"indy": {"¿Serpientes? Odio las serpientes.", "Eso va a un museo."}, "bond": {"Agitado, no mezclado.", "Blond. James Blond."}, "dracula": {"Bleh... ¡buenas noches!", "Quiero chupar tu sangre."},
	"santa": {"¡Jo, jo, jo!", "¿Has sido bueno?"}, "elvis": {"Thank you, thank you very much.", "¡Uh-huh!"}, "freddie": {"¡Ay-oh!", "Los campeones..."},
	"jack_sparrow": {"¿Dónde está el ron?", "Capitán Jack Esparrow."}, "wonka": {"Una fábrica de chocolate...", "¡Pasen, pasen!"}, "beetlejuice": {"¡Es la hora del espectáculo!", "Bettlejuice, Bettlejuice..."},
	"shrek": {"¡Salid de mi pantano!", "Los ogros son como las cebollas."}, "burro_shrek": {"¡No me abandones!", "¿Quieres waffles?"}, "gru": {"¡Minyons!", "Vaya, vaya..."}, "minion": {"¡Bello! ¡Banana!", "¡Tulaliloo ti babidi!"},
	"woody": {"¡Hay una serpiente en mi bota!"}, "buzz": {"¡Hasta el infinito y más allí!"}, "elsa": {"Suéltalo...", "¡Hace frío!"}, "harry_potter": {"¡Expecto petronum!", "¿Has visto mi varita?"}, "hermione": {"Es 'Levitosa', no 'levitosá'."},
	"gandalf": {"¡No puedes posar!", "Un mago nunca llega tarde."}, "frodo": {"No quiero este anillo."}, "legolas": {"¡Tienen un troll de las cavernas!"},
	"conan": {"¿Qué es lo mejor de la vida?"}, "he_man": {"¡Por el poder de Greyskull!", "¡Tengo la fuerza!"}, "marty_mcfly": {"¡Esto es pesado!", "¿A dónde vamos?"}, "doc_brown": {"¡Gran Scott!", "1,21 gigavatios..."},
	"la_mascara": {"¡SSSMOKIN'!", "¡Alegría, alegría!"}, "taz": {"¡Rrrrgh!"}, "michael_jackson": {"¡Hee-hee!", "¡Chamone!"}, "alf": {"¡Ja, ja, ja! ¡Gatos!", "No soy de aquí."}, "johnny5": {"¡Necesito más datos!", "¡Vivo!"},
	"inspector_gadget": {"¡Vamos, Gadjet!", "¡Sacaré un gadget!"}, "picard": {"Engage.", "Té, Earl Grey, caliente."}, "johnny_bravo": {"¡Eh, nena!", "Qué guapo soy."},
	"pedro_picapiedra": {"¡Yabba dabba duu!", "¡Willma!"}, "pitufo": {"¡La la la la la la!", "¡Pitubo pitubo!"}, "papa_pitufo": {"Los Pitubos son muy buenos."}, "bob_esponja": {"¡Estoy listo!", "¡Soy feliz!"},
	"beavis": {"¡Heh heh!", "¡Soy Cornolio!"}, "butthead": {"Huh huh.", "Esto es un rollo."}, "doraemon": {"¡Saco algo de mi bolsillo!"}, "nobita": {"¡Doramon, ayúdame!"},
	"leeloo": {"¡Multipas!"}, "korben": {"¡Multipas!"}, "pantera_rosa": {"..."}, "espinete": {"¡Hola, Don Pinpón!"}, "don_pimpon": {"¡Hola, Espinote!"},
	"lennon": {"Imagina.", "All you need is love."}, "mccartney": {"Yesterday...", "Hey Jode."},
}

func (g *Game) updateGreeting(gu *Guest, dt float64) {
	if !gu.greetInit {
		gu.greetInit = true
		gu.chatty = g.rng.Float64() < 0.38 || (gu.cs != nil && gu.cs.Spec != nil && catchphrases[gu.cs.Spec.Name] != nil && g.rng.Float64() < 0.7)
	}
	if !gu.chatty || gu.icon != "" || g.time-gu.lastGreet < 28 || gu.perf != "" {
		return
	}
	p := g.p
	if p.dead || p.hidden || dist(gu.pos, p.pos) > 46 {
		return
	}
	switch gu.state {
	case gIdle, gDance, gChat, gDrink, gLounge, gDanceWith, gGoto:
	default:
		return
	}
	if g.rng.Float64() > 0.015 {
		return
	}
	line := g.freshPhrase("greet", greetLines, g.rng.Intn)
	if gu.cs != nil && gu.cs.Spec != nil {
		if l := catchphrases[gu.cs.Spec.Name]; len(l) > 0 && g.rng.Float64() < 0.75 {
			line = g.freshPhrase("catch/"+gu.cs.Spec.Name, l, g.rng.Intn)
		}
	}
	if line == "" {
		return // nothing new to say
	}
	gu.icon, gu.iconT, gu.lastGreet = line, 2.6, g.time
}
