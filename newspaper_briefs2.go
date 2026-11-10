package main

import (
	"fmt"
	"sort"
)

// More small news for the front page (the first bank is in newspaper_briefs.go): the DJ, the food, the police, the weather,
// the traps, the cars, the animals, the oddest guests and the killer, all in the third person and with slapstick humour.
// And the "derived" ones, which are written from the numbers of the party itself.

func init() {
	for k, v := range map[string]string{"FAUNA": "WILDLIFE", "TRÁFICO": "TRAFFIC", "POLICÍA": "POLICE", "SALUD": "HEALTH", "PRENSA": "PRESS", "LOCALES": "LOCAL", "RECUENTO": "TALLY"} {
		paperBriefHeads[k] = v
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

var paperBriefBank2 = []paperBrief{
	// the DJ
	{"MÚSICA", "El DJ jura que su lista tiene mil canciones. Suenan las mismas tres.", "The DJ swears his playlist has a thousand songs. The same three play."},
	{"MÚSICA", "Un invitado pidió «música tranquila» y el DJ puso sirenas. Era un chiste.", "A guest asked for \"calm music\" and the DJ played sirens. It was a joke."},
	{"MÚSICA", "El DJ cambió el disco y se llevó un aplauso. Del disco viejo.", "The DJ changed the record and got a round of applause. From the old record."},
	{"MÚSICA", "El DJ pide un micrófono nuevo. El otro lo tiene el pato.", "The DJ asks for a new microphone. The duck has the other one."},
	{"MÚSICA", "Un cantante de baño actúa en el salón. Se pide cambio de baño.", "A bathroom singer performs in the lounge. A change of bathroom is requested."},
	{"MÚSICA", "El DJ dedicó una canción a la policía. Duraba nueve minutos.", "The DJ dedicated a song to the police. It lasted nine minutes."},
	{"MÚSICA", "La batería sonó tan fuerte que se cayó un cuadro. Y no era un cuadro.", "The drums were so loud a painting fell. And it was not a painting."},
	{"MÚSICA", "El DJ intentó bajar el volumen y subió, por error, el telón.", "The DJ tried to turn the volume down and raised, by mistake, the curtain."},
	// food
	{"GASTRONOMÍA", "La tarta llegó entera y se fue a trozos. Nadie sabe adónde.", "The cake came whole and left in pieces. Nobody knows where."},
	{"GASTRONOMÍA", "El cocinero pide que no llamen «sospechosa» a su ensaladilla.", "The cook asks that his potato salad not be called \"suspicious\"."},
	{"GASTRONOMÍA", "Una croqueta rodó por el salón y fue detenida por un agente en prácticas.", "A croquette rolled across the lounge and was stopped by a trainee officer."},
	{"GASTRONOMÍA", "El jamón desapareció a las diez. A las diez y cinco, el hueso.", "The ham vanished at ten. At five past, so did the bone."},
	{"GASTRONOMÍA", "Hay empate entre quien come la tortilla y quien la defiende.", "There is a tie between those who eat the omelette and those who defend it."},
	{"GASTRONOMÍA", "Se pide a los invitados no meter los canapés en el bolsillo del vecino.", "Guests are asked not to put canapés in the neighbour's pocket."},
	{"GASTRONOMÍA", "Receta del día: una tarta con nueve pisos. La octava se cayó.", "Recipe of the day: a nine-tier cake. The eighth fell."},
	{"GASTRONOMÍA", "Los palillos de los pinchos aparecieron en lugares inesperados.", "The toothpicks from the snacks turned up in unexpected places."},
	{"GASTRONOMÍA", "El camarero asegura que las gambas ya venían así. Nadie le cree.", "The waiter swears the prawns came like that. Nobody believes him."},
	// police
	{"POLICÍA", "Un agente se persiguió a sí mismo durante tres calles. Atrapado.", "An officer chased himself for three streets. Caught."},
	{"POLICÍA", "La poli pide ayuda a la ciudadanía y a un señor con un perro.", "The police ask citizens for help, and a man with a dog."},
	{"POLICÍA", "El inspector de guardia escribió «Caso cerrado» y se le abrió el cajón.", "The inspector on duty wrote \"Case closed\" and the drawer opened."},
	{"POLICÍA", "Un poli persiguió a un ladrón de tartas. Era su compañero.", "A cop chased a cake thief. It was his partner."},
	{"POLICÍA", "El jefe de policía llegó tarde y con el uniforme de otro.", "The police chief arrived late wearing somebody else's uniform."},
	{"POLICÍA", "Los agentes buscaron huellas en el suelo. Encontraron las suyas.", "The officers looked for prints on the floor. They found their own."},
	{"POLICÍA", "Un policía pidió el carné a un maniquí. El maniquí no lo tenía.", "A policeman asked a mannequin for ID. The mannequin did not have it."},
	{"POLICÍA", "Dos agentes se disputan una rosquilla y dejan escapar al sospechoso.", "Two officers fight over a doughnut and let the suspect go."},
	{"POLICÍA", "La esposa de un poli llamó para avisar de que la cena ya estaba.", "A cop's wife called to say dinner was ready."},
	{"POLICÍA", "El perro policía olfateó la tortilla y pidió el día libre.", "The police dog sniffed the omelette and asked for the day off."},
	// weather
	{"TIEMPO", "Chubascos de confeti al caer la noche y un pequeño tornado de servilletas.", "Confetti showers at nightfall and a small tornado of napkins."},
	{"TIEMPO", "Viento flojo del sur con rachas de música alta en el jardín.", "Light south wind with gusts of loud music in the garden."},
	{"TIEMPO", "Mañana, resaca con probabilidad de aspirina del cien por cien.", "Tomorrow, hangover with a one hundred per cent chance of aspirin."},
	{"TIEMPO", "Cielos despejados salvo en el salón, donde hay humo de canapés.", "Clear skies except in the lounge, where there is canapé smoke."},
	{"TIEMPO", "Tormenta en la cocina: un cocinero se enfada con una sartén.", "Storm in the kitchen: a cook is angry with a frying pan."},
	{"TIEMPO", "Se esperan lluvias de globos pinchados en zonas del salón.", "Showers of popped balloons are expected in parts of the lounge."},
	{"TIEMPO", "Bochorno en la pista y una nube sospechosa sobre la piscina.", "Muggy on the floor and a suspicious cloud over the pool."},
	// traps
	{"SUCESOS", "Un invitado cayó en una trampa que le habían dejado a otro. Qué mala suerte ajena.", "A guest fell into a trap meant for somebody else. Such second-hand bad luck."},
	{"SUCESOS", "Se pide al público no pisar donde dice «No pisar». Gracias.", "The public is asked not to step where it says \"Do not step\". Thank you."},
	{"SUCESOS", "Una trampa se disparó sola. La investigación apunta a la propia trampa.", "A trap went off by itself. The investigation points to the trap."},
	{"SUCESOS", "El ratonero del pasillo atrapó un zapato. Ni dueño ni pie.", "The mousetrap in the hall caught a shoe. No owner and no foot."},
	{"SUCESOS", "Aparece una trampa con lazo rosa. Se cree que era un regalo.", "A trap with a pink bow turns up. It is believed to have been a gift."},
	{"SUCESOS", "El cartel de «Cuidado, trampas» ya tiene su propia trampa.", "The sign \"Careful, traps\" now has its own trap."},
	// cars and traffic
	{"TRÁFICO", "Un coche aparcó en la piscina. El conductor dice que era un garaje.", "A car parked in the pool. The driver says it was a garage."},
	{"TRÁFICO", "Atasco de tres coches y una bicicleta frente al jardín. Pitidos por gusto.", "A jam of three cars and a bicycle outside the garden. Honking for fun."},
	{"TRÁFICO", "Un camión de reparto llegó con la tarta y se fue con las cortinas.", "A delivery truck arrived with the cake and left with the curtains."},
	{"TRÁFICO", "Un taxi esperó dos horas a un cliente que ya se había ido... de otra forma.", "A taxi waited two hours for a customer who had already left... another way."},
	{"TRÁFICO", "La grúa se llevó el coche del vecino y también al vecino.", "The tow truck took the neighbour's car and also the neighbour."},
	{"TRÁFICO", "Un coche patrulla se saltó el semáforo para llegar tarde.", "A patrol car ran the red light in order to arrive late."},
	{"TRÁFICO", "Un tren cruzó la vía y dejó a un invitado con cara de susto.", "A train crossed the tracks and left a guest wearing a scared face."},
	// animals
	{"FAUNA", "Una gallina cruzó la carretera. La policía quiere saber por qué.", "A hen crossed the road. The police want to know why."},
	{"FAUNA", "El toro del jardín se queja del trato. Dice que lo toman por un perro.", "The bull in the garden complains about his treatment. He says they take him for a dog."},
	{"FAUNA", "Una serpiente pidió asilo en el sofá y se le concedió, con reservas.", "A snake asked for asylum on the sofa and was granted it, with reservations."},
	{"FAUNA", "El loro de la casa repite todo lo que oyó. Se le busca para declarar.", "The house parrot repeats everything it heard. It is wanted for questioning."},
	{"FAUNA", "Una oveja fue vista en la azotea. Nadie sabe cómo subió ni cómo baja.", "A sheep was spotted on the roof. Nobody knows how it got up or how it will get down."},
	{"FAUNA", "El gato del vecino ha pedido un abogado y una lata de atún.", "The neighbour's cat has asked for a lawyer and a tin of tuna."},
	{"FAUNA", "Un caballo galopó por el pasillo. Su dueño lo busca en el cuarto de baño.", "A horse galloped down the hall. Its owner looks for it in the bathroom."},
	{"FAUNA", "El perro de la casa ladró a un fantasma. El fantasma dijo «guau».", "The house dog barked at a ghost. The ghost said \"woof\"."},
	{"FAUNA", "Las gallinas ponen huevos de bienvenida. De pascua, a ser posible.", "The hens lay welcome eggs. Easter ones, preferably."},
	{"FAUNA", "El T-Rex del salón pide perdón por el roce con la lámpara.", "The T-Rex in the lounge apologises for the scrape with the lamp."},
	// odd guests
	{"SOCIEDAD", "Un invitado vino disfrazado de invitado. Pasó desapercibido.", "A guest came dressed as a guest. He went unnoticed."},
	{"SOCIEDAD", "Dos invitados disfrazados de lo mismo se sientan lejos. Humillación.", "Two guests dressed as the same thing sit far apart. Humiliation."},
	{"SOCIEDAD", "Un señor con sombrero de copa asegura que es el dueño. Y de la copa.", "A man in a top hat says he is the owner. Of the hat, at least."},
	{"SOCIEDAD", "Una invitada llegó con cuatro maletas. Dice que viene de mudanza.", "A guest arrived with four suitcases. She says she is moving in."},
	{"SOCIEDAD", "Un caballero baila solo desde las ocho. Nadie se atreve a decirle nada.", "A gentleman has been dancing alone since eight. Nobody dares tell him."},
	{"SOCIEDAD", "Un invitado se quedó dormido sobre la tortilla. Fue el mejor cojín.", "A guest fell asleep on the omelette. It was the best cushion."},
	{"SOCIEDAD", "Una señora dice conocer a todos. Nadie la conoce a ella.", "A lady says she knows everybody. Nobody knows her."},
	{"SOCIEDAD", "Un mimo estuvo gritando por socorro, pero nadie entendió el gesto.", "A mime was calling for help, but nobody understood the gesture."},
	// the killer
	{"ÚLTIMA HORA", "Se busca asesino: media estatura, modales dudosos, buen apetito.", "Killer wanted: average height, dubious manners, healthy appetite."},
	{"ÚLTIMA HORA", "El asesino fue visto bailando. Eso lo hace aún más sospechoso.", "The killer was seen dancing. That makes him even more suspicious."},
	{"ÚLTIMA HORA", "Según un experto, el asesino «actúa con método». Nadie sabe cuál.", "According to an expert, the killer \"acts with method\". Nobody knows which."},
	{"ÚLTIMA HORA", "El asesino cambió de sombrero para pasar desapercibido. No coló.", "The killer changed hats to go unnoticed. It did not work."},
	{"ÚLTIMA HORA", "El asesino pide una tregua para tomar un canapé. Se le concede.", "The killer asks for a truce to have a canapé. It is granted."},
	{"ÚLTIMA HORA", "Una vecina cree haber visto al asesino en el ascensor. Era un espejo.", "A neighbour thinks she saw the killer in the lift. It was a mirror."},
	// neighbours and misc
	{"VECINOS", "El de arriba quiere dormir, el de abajo quiere bailar y el del lado, cobrar.", "The one above wants to sleep, the one below wants to dance and the one beside wants paying."},
	{"VECINOS", "Los del quinto han mandado una carta. En un globo. Ardiendo.", "The folks on the fifth floor sent a letter. By balloon. On fire."},
	{"VECINOS", "La portera pide silencio con una escoba y mucho entusiasmo.", "The caretaker asks for silence with a broom and a lot of enthusiasm."},
	{"VECINOS", "Una vecina ofrece tortilla a cambio de que la música pare. Ofertas.", "A neighbour offers omelette in exchange for the music stopping. Offers welcome."},
	{"SALUD", "El médico recomienda reposo, agua y no volver a esta fiesta.", "The doctor recommends rest, water and never coming back to this party."},
	{"SALUD", "Varios invitados sufren «ataques de risa». Es lo único que no se contagia.", "Several guests suffer \"laughing fits\". It is the only thing not catching."},
	{"SALUD", "La tirita de la fiesta, agotada. Se pide material de cocina.", "The party plaster has run out. Kitchen supplies are requested."},
	{"CULTURA", "Exposición de sombreros perdidos en el guardarropa: la entrada, un sombrero.", "Exhibition of hats lost in the cloakroom: admission, one hat."},
	{"CULTURA", "Poeta local lee su obra en el baño. Aplausos a través de la puerta.", "A local poet reads his work in the bathroom. Applause through the door."},
	{"CULTURA", "Un cuadro del salón fue robado. No era bueno, pero era grande.", "A painting was stolen from the lounge. It was not good, but it was big."},
	{"PRENSA", "Este diario ha intentado entrevistar a la tortilla. Declina comentar.", "This paper tried to interview the omelette. It declines to comment."},
	{"PRENSA", "El fotógrafo de este diario salió mal en todas las fotos. Es su estilo.", "This paper's photographer came out badly in every picture. It is his style."},
	{"PRENSA", "Erratas del diario de ayer: donde decía «asesino», léase «invitado».", "Corrections from yesterday's paper: where it said \"killer\", read \"guest\"."},
	{"PRENSA", "Lector indignado pide devolver su aperitivo. Se le devuelve el hueso.", "An indignant reader asks to return his appetizer. He is given the bone."},
	{"LOCALES", "El bar de la esquina sirve «lo de siempre». Nadie sabe qué es.", "The corner bar serves \"the usual\". Nobody knows what that is."},
	{"LOCALES", "El kiosco vendió el periódico antes de que pasara la noticia.", "The newsstand sold the paper before the news happened."},
	{"LOCALES", "Se inaugura un museo del ponche. Dos salas: la barra y la cocina.", "A punch museum opens. Two rooms: the bar and the kitchen."},
	{"DEPORTES", "Torneo de lanzamiento de croquetas: gana el que no se las come.", "Croquette throwing tournament: the winner is whoever does not eat them."},
	{"DEPORTES", "Carrera de camareros con bandeja: el ganador, sin una sola copa.", "Waiters' tray race: the winner, without a single glass."},
	{"DEPORTES", "Maratón de baile del salón: lleva doce horas y dos pies.", "Lounge dance marathon: twelve hours and two feet so far."},
}

// ---- the small news that come from the party's own numbers -------------------------------------------------------

var pBriefTime = []paperTxt{
	tx("La fiesta duró %s. Los invitados juran que se hizo larga.", "The party lasted %s. Guests swear it felt longer."),
	tx("Duración de la fiesta: %s. El reloj protesta por el esfuerzo.", "Length of the party: %s. The clock complains about the effort."),
	tx("En %s la casa pasó de fiesta a lugar de los hechos.", "In %s the house went from party to scene of the crime."),
	tx("Cronómetro oficial: %s. Los forenses piden prórroga.", "Official stopwatch: %s. The forensics team asks for extra time."),
	tx("El asesino hizo su faena en %s. Y aún le sobró tiempo para el postre.", "The killer did his job in %s. And still had time for dessert."),
	tx("Fiesta de %s de duración, con aplausos al final por compromiso.", "A party of %s, with obligatory applause at the end."),
	tx("Tiempo total: %s. Los invitados que quedan piden una silla.", "Total time: %s. The remaining guests ask for a chair."),
	tx("Con %s de música a todo volumen, el barrio recomienda tapones.", "With %s of loud music, the neighbourhood recommends earplugs."),
}

var pBriefKills = []paperTxt{
	tx("%d abrigos se quedaron sin recoger en el guardarropa.", "%d coats were left uncollected in the cloakroom."),
	tx("%d invitados no tuvieron que pagar el taxi de vuelta.", "%d guests did not need to pay for the taxi home."),
	tx("La fiesta perdió %d invitados. El catering se ahorra %d raciones.", "The party lost %d guests. The caterer saves %d servings."),
	tx("El guardarropa tiene %d abrigos sin dueño. Se regalan a quien pregunte.", "The cloakroom has %d coats without owners. Free to whoever asks."),
	tx("%d sillas vacías. Se sospecha que no se han ido por voluntad propia.", "%d empty chairs. They are suspected not to have left of their own accord."),
	tx("Faltan %d en la foto de grupo. Se hará otra, más pequeña.", "%d are missing from the group photo. A smaller one will be taken."),
	tx("%d invitados se perdieron el brindis final. Una lástima.", "%d guests missed the final toast. A pity."),
	tx("La lista de invitados baja en %d. La del catering, también.", "The guest list drops by %d. So does the catering list."),
}

var pBriefNone = []paperTxt{
	tx("Cero bajas: los forenses se fueron con la tarta y mala cara.", "Zero casualties: the forensics team left with the cake and a long face."),
	tx("Sin una sola baja, el guardarropa devolvió todos los abrigos. Qué aburrimiento.", "Without a single casualty, the cloakroom returned all the coats. How dull."),
	tx("Nadie falta. Al menos, nadie que se haya notado.", "Nobody is missing. At least, nobody who was noticed."),
	tx("El asesino se sentó a comer croquetas y se olvidó del trabajo.", "The killer sat down to eat croquettes and forgot about work."),
	tx("Hoy el cuchillo solo cortó tarta. Y mal.", "Today the knife only cut cake. And badly."),
	tx("Se esperaba crimen y llegó el aperitivo. Los vecinos, defraudados.", "A crime was expected and the appetizers came. The neighbours are disappointed."),
}

var pBriefSeen = []paperTxt{
	tx("%d testigos quedaron en el bar para contrastar versiones.", "%d witnesses met at the bar to compare versions."),
	tx("%d personas vieron algo. Cada una vio una cosa distinta.", "%d people saw something. Each saw something different."),
	tx("Los %d testigos declaran. El resto, tomando ponche.", "The %d witnesses testify. The rest are drinking punch."),
	tx("%d testigos, %d versiones y una tortilla que no se mueve.", "%d witnesses, %d versions and an omelette that will not budge."),
	tx("Con %d testigos, el caso va para largo. Y para el bar.", "With %d witnesses, the case will drag on. And head for the bar."),
	tx("%d invitados vieron al asesino, y todos prefirieron seguir bailando.", "%d guests saw the killer, and all preferred to keep dancing."),
}

var pBriefUnseen = []paperTxt{
	tx("Cero testigos: el barrio entero miraba el móvil.", "Zero witnesses: the whole neighbourhood was on their phones."),
	tx("Ni un testigo. Se ofrece una tortilla a quien recuerde algo.", "Not a witness. An omelette is offered to anyone who remembers anything."),
	tx("Sin testigos y con muchas ganas de no tener ninguno.", "No witnesses and a great wish not to have any."),
	tx("El único testigo es un loro, y repite lo que le conviene.", "The only witness is a parrot, and it repeats what suits it."),
	tx("Nadie vio al asesino. Este diario lo achaca a la iluminación.", "Nobody saw the killer. This paper blames the lighting."),
	tx("La policía interroga a las cortinas, las únicas que estaban mirando.", "The police question the curtains, the only ones that were watching."),
}

var pBriefTwo = []paperTxt{
	tx("Dos asesinos en una sola fiesta: hostelería pide cita previa.", "Two killers at one party: catering demands an appointment."),
	tx("Dos asesinos en una fiesta. El guardarropa pide una segunda percha.", "Two killers at one party. The cloakroom asks for a second hanger."),
	tx("Entre los dos asesinos hubo piques por el último invitado. Se sortearon.", "The two killers squabbled over the last guest. They drew lots."),
	tx("Dos asesinos, dos sombreros y una sola lista de tareas.", "Two killers, two hats and a single to-do list."),
	tx("Los asesinos se saludaron con un gesto de cabeza. Se respetan.", "The killers nodded to each other. They respect each other."),
}

var pBriefAnimalsDead = []paperTxt{
	tx("%d animales menos en la granja. El gallo ya no madruga.", "%d fewer animals on the farm. The rooster no longer rises early."),
	tx("Aparecieron %d animales fuera de su sitio. Y fuera del mundo.", "%d animals turned up out of place. And out of this world."),
	tx("La protectora cuenta %d bajas entre las mascotas. No está contenta.", "The animal shelter counts %d pet casualties. It is not happy."),
	tx("%d animales, ningún abogado. El asesino lo tiene fácil.", "%d animals, no lawyer. The killer has it easy."),
	tx("El veterinario atendió a %d pacientes. Ninguno dijo gracias.", "The vet saw %d patients. None said thank you."),
	tx("Las gallinas hicieron un minuto de silencio por %d compañeros.", "The hens held a minute of silence for %d companions."),
}

var pBriefAnimalsAlive = []paperTxt{
	tx("Los %d animales de la casa se salvaron. Hubo ración extra de pienso.", "All %d house animals survived. There was an extra ration of feed."),
	tx("%d animales vivos y coleando. Los que tienen cola.", "%d animals alive and kicking. The ones with tails, wagging."),
	tx("Se contaron %d animales al final. Ninguno falta, y eso es raro.", "%d animals were counted at the end. None missing, which is odd."),
	tx("Los %d animales declaran no haber visto nada. Como casi todos.", "The %d animals declare they saw nothing. Like almost everybody."),
	tx("Las mascotas, %d en total, durmieron en el sofá. El sofá no se queja.", "The pets, %d in all, slept on the sofa. The sofa does not complain."),
	tx("%d animales y ni un solo rasguño: la fauna, mejor que la prensa.", "%d animals and not a single scratch: the wildlife, better than the press."),
}

var pBriefCops = []paperTxt{
	tx("La policía llegó y se fue. Dejó una multa y un bocadillo.", "The police came and went. They left a fine and a sandwich."),
	tx("La policía interroga a todos los testigos y a un maniquí. Hay avances.", "The police question all the witnesses and a mannequin. There are developments."),
	tx("Un agente se quedó dormido en el coche patrulla. Se le despertará mañana.", "An officer fell asleep in the patrol car. He will be woken tomorrow."),
	tx("Se pide a la policía que no vuelva a llamar a la puerta con el pie.", "The police are asked not to knock on the door with their foot again."),
	tx("Los agentes pidieron refuerzos y llegaron diez. Todos con cara de susto.", "The officers asked for backup and ten arrived. All looking scared."),
	tx("La comisaría anuncia: «Estamos investigando». Es lo que ponen siempre.", "The station announces: \"We are investigating\". It is what they always say."),
}

// paperDerived: the briefs that come from the party itself (facts known to the paper). The text keeps its %d / %s: the
// layout translates and fills it in (Spanish and English share the order of the numbers).
func paperDerived(f paperFacts, timeStr string) []paperBrief {
	pk := func(salt int, pool []paperTxt) string { return f.pick(300+salt, pool) }
	out := []paperBrief{{"CRONÓMETRO", fmt.Sprintf(pk(0, pBriefTime), timeStr), ""}}
	switch {
	case f.kills == 0:
		out = append(out, paperBrief{"RECUENTO", pk(1, pBriefNone), ""})
	default:
		s := pk(1, pBriefKills)
		if formatCount(s) == 2 {
			out = append(out, paperBrief{"RECUENTO", fmt.Sprintf(s, f.kills, f.kills), ""})
		} else {
			out = append(out, paperBrief{"RECUENTO", fmt.Sprintf(s, f.kills), ""})
		}
	}
	if f.seen > 0 {
		s := pk(2, pBriefSeen)
		if formatCount(s) == 2 {
			out = append(out, paperBrief{"TESTIGOS", fmt.Sprintf(s, f.seen, f.seen), ""})
		} else {
			out = append(out, paperBrief{"TESTIGOS", fmt.Sprintf(s, f.seen), ""})
		}
	} else {
		out = append(out, paperBrief{"TESTIGOS", pk(2, pBriefUnseen), ""})
	}
	if f.two {
		out = append(out, paperBrief{"DÚO DINÁMICO", pk(3, pBriefTwo), ""})
	}
	if f.animalsDead > 0 {
		out = append(out, paperBrief{"FAUNA", fmt.Sprintf(pk(4, pBriefAnimalsDead), f.animalsDead), ""})
	} else if f.animalsAlive >= 2 {
		out = append(out, paperBrief{"FAUNA", fmt.Sprintf(pk(5, pBriefAnimalsAlive), f.animalsAlive), ""})
	}
	if f.cuffed > 0 || f.units != 0 {
		out = append(out, paperBrief{"POLICÍA", pk(6, pBriefCops), ""})
	}
	return out
}

// paperBriefsFor: every candidate, shuffled by the seed (the layout takes the ones that fit; none twice). What the last
// parties printed goes to the back of the line.
func paperBriefsFor(f paperFacts, timeStr string) []paperBrief {
	all := append(append([]paperBrief{}, paperDerived(f, timeStr)...), paperBriefBank...)
	type kv struct {
		k uint64
		b paperBrief
	}
	ks := make([]kv, len(all))
	for i, b := range all {
		h := uint64(f.seed)*6364136223846793005 + uint64(i+7)*1442695040888963407
		h ^= h >> 31
		h *= 0x9e3779b97f4a7c15
		if f.avoid[b.es] {
			h |= 1 << 63
		} else {
			h &^= 1 << 63
		}
		ks[i] = kv{h, b}
	}
	sort.SliceStable(ks, func(i, j int) bool { return ks[i].k < ks[j].k })
	out := make([]paperBrief, len(ks))
	for i := range ks {
		out[i] = ks[i].b
	}
	out = paperEventFirst(f, out) // the news about the party's events go first (newspaper_events.go)
	return paperTplInsert(f, out) // then the template news about what happened, and two general ones (newspaper_tpl.go)
}

// paperTplInsert puts the template news of the party in the list: the ones about what happened right after the
// event news, the general ones a little further down (among the shuffled bank).
func paperTplInsert(f paperFacts, list []paperBrief) []paperBrief {
	picks := paperTplPicks(f)
	if len(picks) == 0 {
		return list
	}
	nEv := len(paperEventBriefs(f))
	if nEv > len(list) {
		nEv = len(list)
	}
	var spec, gen []paperBrief
	for _, p := range picks {
		b := paperBrief{p.head, p.es, p.en}
		if t := paperTplByID(p.id); t != nil && paperTplGeneral(t.ctx) {
			gen = append(gen, b)
		} else {
			spec = append(spec, b)
		}
	}
	out := make([]paperBrief, 0, len(list)+len(picks))
	out = append(out, list[:nEv]...)
	out = append(out, spec...)
	rest := list[nEv:]
	for i, b := range rest {
		if len(gen) > 0 && (i == 1 || i == 3) {
			out = append(out, gen[0])
			gen = gen[1:]
		}
		out = append(out, b)
	}
	return append(out, gen...)
}

var paperBriefDerivedPools = [][]paperTxt{pBriefTime, pBriefKills, pBriefNone, pBriefSeen, pBriefUnseen, pBriefTwo, pBriefAnimalsDead,
	pBriefAnimalsAlive, pBriefCops}

func init() {
	add := func(k, v string) {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, pool := range paperBriefDerivedPools {
		for _, p := range pool {
			add(p.es, p.en)
		}
	}
	for _, b := range paperBriefBank2 {
		add(b.es, b.en)
	}
}
