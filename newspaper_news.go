package main

import "strings"

// ---------------------------------------------------------------------------
// The words of the newspaper front page: headlines, the openings of the story, the optional sentences, the witness
// and the photo captions. Every entry is a pair (Spanish, English) written in the third person with the humour of
// a slapstick comic: clumsy cops, a suspicious duck, an ambitious omelette. Nobody real is named.
// Selection (newspaper.go) is by seed and avoids what the last parties already printed.
// ---------------------------------------------------------------------------

type paperTxt struct{ es, en string }

func tx(es, en string) paperTxt { return paperTxt{es, en} }

// pick chooses one entry of the pool by the seed of the party, leaving out what the last two parties printed (if the
// whole pool was printed, anything goes).
func (f paperFacts) pick(salt int, pool []paperTxt) string {
	cand := pool
	if len(f.avoid) > 0 {
		var fresh []paperTxt
		for _, p := range pool {
			if !f.avoid[p.es] {
				fresh = append(fresh, p)
			}
		}
		if len(fresh) > 0 {
			cand = fresh
		}
	}
	h := uint64(f.seed)*6364136223846793005 + uint64(salt+1)*1442695040888963407
	h ^= h >> 29
	h *= 0x9e3779b97f4a7c15
	h ^= h >> 31
	return cand[int(h%uint64(len(cand)))].es
}

// ---- headlines: "FIESTA %s ..." (the party number), with one %d when the line shows a count ----------------------------

var pHeadNone = []paperTxt{
	tx("FIESTA %s: CERO BAJAS Y MUCHO CANAPÉ", "PARTY %s: ZERO LOSSES, LOTS OF CANAPÉS"),
	tx("FIESTA %s: EL ASESINO SE FUE SIN TRABAJAR", "PARTY %s: THE KILLER LEFT WITHOUT WORKING"),
	tx("FIESTA %s: TODOS VIVOS Y SIN EXPLICACIÓN", "PARTY %s: ALL ALIVE, NO EXPLANATION"),
	tx("FIESTA %s: NADIE FALTÓ A LA CITA", "PARTY %s: NOBODY MISSED THE DATE"),
	tx("FIESTA %s: UNA TARDE DE LO MÁS SANA", "PARTY %s: A MOST HEALTHY EVENING"),
	tx("FIESTA %s: EL CUCHILLO SE QUEDÓ EN CASA", "PARTY %s: THE KNIFE STAYED HOME"),
	tx("FIESTA %s: LA POLI SE ABURRE, SIN CASO", "PARTY %s: COPS BORED, NO CASE"),
	tx("FIESTA %s: EL ASESINO FUE MUY EDUCADO", "PARTY %s: THE KILLER WAS VERY POLITE"),
	tx("FIESTA %s: SE SALVÓ HASTA LA TORTILLA", "PARTY %s: EVEN THE OMELETTE SURVIVED"),
	tx("FIESTA %s: DEMASIADA PAZ PARA UN SÁBADO", "PARTY %s: TOO MUCH PEACE FOR A SATURDAY"),
	tx("FIESTA %s: NI UN RASGUÑO. SOSPECHOSO", "PARTY %s: NOT A SCRATCH. SUSPICIOUS"),
	tx("FIESTA %s: SE ABURRIÓ HASTA EL ASESINO", "PARTY %s: EVEN THE KILLER GOT BORED"),
	tx("FIESTA %s: EL ASESINO SOLO VINO A BAILAR", "PARTY %s: THE KILLER ONLY CAME TO DANCE"),
	tx("FIESTA %s: UN CONO, UN PERRO Y CERO BAJAS", "PARTY %s: A CONE, A DOG AND ZERO LOSSES"),
}

var pHeadClean = []paperTxt{
	tx("FIESTA %s: LA MÁS LIMPIA DEL BARRIO", "PARTY %s: THE CLEANEST IN TOWN"),
	tx("FIESTA %s: %d AUSENTES Y NI UN TESTIGO", "PARTY %s: %d MISSING AND NOT ONE WITNESS"),
	tx("FIESTA %s: %d BAJAS SIN DEJAR HUELLA", "PARTY %s: %d DOWN, NOT A TRACE LEFT"),
	tx("FIESTA %s: EL ASESINO FUE UN FANTASMA", "PARTY %s: THE KILLER WAS A GHOST"),
	tx("FIESTA %s: %d AUSENTES Y CERO PISTAS", "PARTY %s: %d MISSING, ZERO CLUES"),
	tx("FIESTA %s: CRIMEN PERFECTO, DICE EL DJ", "PARTY %s: PERFECT CRIME, SAYS THE DJ"),
	tx("FIESTA %s: NADIE VIO NADA (%d VECES)", "PARTY %s: NOBODY SAW A THING (%d TIMES)"),
	tx("FIESTA %s: %d BAJAS EN SILENCIO TOTAL", "PARTY %s: %d DOWN IN TOTAL SILENCE"),
	tx("FIESTA %s: SIGILO NINJA, %d BAJAS", "PARTY %s: NINJA STEALTH, %d DOWN"),
	tx("FIESTA %s: LA POLI BUSCA A UN INVISIBLE", "PARTY %s: COPS HUNT AN INVISIBLE MAN"),
	tx("FIESTA %s: %d SE ESFUMARON Y NADIE LO VIO", "PARTY %s: %d VANISHED, NOBODY SAW"),
}

var pHeadBlast = []paperTxt{
	tx("FIESTA %s: ¡BOOM! %d AUSENTES EN LA PISTA", "PARTY %s: BOOM! %d MISSING ON THE FLOOR"),
	tx("FIESTA %s: LA PISTA VOLÓ CON %d INVITADOS", "PARTY %s: DANCE FLOOR BLOWN UP WITH %d GUESTS"),
	tx("FIESTA %s: PIROTECNIA NO PEDIDA, %d BAJAS", "PARTY %s: UNREQUESTED FIREWORKS, %d DOWN"),
	tx("FIESTA %s: %d INVITADOS SALEN VOLANDO", "PARTY %s: %d GUESTS GO FLYING"),
	tx("FIESTA %s: LOS VECINOS OYERON %d ¡BOOM!", "PARTY %s: NEIGHBOURS HEARD %d BOOMS"),
	tx("FIESTA %s: CONFETI EXPLOSIVO, %d BAJAS", "PARTY %s: EXPLOSIVE CONFETTI, %d DOWN"),
	tx("FIESTA %s: FUEGOS ARTIFICIALES SIN PERMISO", "PARTY %s: FIREWORKS WITHOUT A PERMIT"),
	tx("FIESTA %s: ¡LA PISTA, POR LOS AIRES! (%d)", "PARTY %s: DANCE FLOOR SKY-HIGH! (%d)"),
	tx("FIESTA %s: DEMOLICIÓN DE FIESTA, %d BAJAS", "PARTY %s: PARTY DEMOLITION, %d DOWN"),
	tx("FIESTA %s: TECHO CON VISTAS GRACIAS A %d", "PARTY %s: ROOF WITH A VIEW THANKS TO %d"),
}

var pHeadPoison = []paperTxt{
	tx("FIESTA %s: EL PONCHE SE LLEVÓ A %d", "PARTY %s: THE PUNCH TOOK %d"),
	tx("FIESTA %s: %d INVITADOS SE QUEDAN SIN POSTRE", "PARTY %s: %d GUESTS MISS DESSERT"),
	tx("FIESTA %s: EL PONCHE LLEVABA ALGO EXTRA", "PARTY %s: THE PUNCH HAD A SECRET EXTRA"),
	tx("FIESTA %s: %d BRINDIS QUE SALIERON CAROS", "PARTY %s: %d TOASTS THAT CAME DEAR"),
	tx("FIESTA %s: %d INTOXICADOS, CULPA DEL PONCHE", "PARTY %s: %d POISONED, BLAME THE PUNCH"),
	tx("FIESTA %s: LA TORTILLA NO TENÍA LA CULPA", "PARTY %s: THE OMELETTE WAS INNOCENT"),
	tx("FIESTA %s: %d CAEN TRAS EL PRIMER SORBO", "PARTY %s: %d FALL AFTER THE FIRST SIP"),
	tx("FIESTA %s: CATERING BAJO SOSPECHA, %d BAJAS", "PARTY %s: CATERING SUSPECTED, %d DOWN"),
	tx("FIESTA %s: UN SORBITO Y %d AL SUELO", "PARTY %s: ONE SIP AND %d ON THE FLOOR"),
	tx("FIESTA %s: COCTEL MORTAL, %d SIN BRINDAR", "PARTY %s: DEADLY COCKTAIL, %d UNTOASTED"),
}

var pHeadTrap = []paperTxt{
	tx("FIESTA %s: ¡CLAC! %d INVITADOS EN LA TRAMPA", "PARTY %s: CLACK! %d GUESTS IN THE TRAP"),
	tx("FIESTA %s: LA CASA ESTABA LLENA DE TRAMPAS", "PARTY %s: THE HOUSE WAS FULL OF TRAPS"),
	tx("FIESTA %s: NADIE LEYÓ EL CARTEL DE CUIDADO", "PARTY %s: NOBODY READ THE WARNING SIGN"),
	tx("FIESTA %s: %d PISARON DONDE NO DEBÍAN", "PARTY %s: %d STEPPED WHERE THEY SHOULD NOT"),
	tx("FIESTA %s: EL SUELO ERA UNA TRAMPA (%d)", "PARTY %s: THE FLOOR WAS A TRAP (%d)"),
	tx("FIESTA %s: PLAF, CLAC, PUM: %d BAJAS", "PARTY %s: SPLAT, CLACK, BOOM: %d DOWN"),
	tx("FIESTA %s: MANUAL DE TRAMPAS, SIN LEER", "PARTY %s: TRAP MANUAL, UNREAD"),
	tx("FIESTA %s: ASESINO DE BRICOLAJE, %d BAJAS", "PARTY %s: DIY KILLER, %d DOWN"),
	tx("FIESTA %s: TRAMPAS PARA TODOS LOS GUSTOS", "PARTY %s: TRAPS TO SUIT EVERY TASTE"),
	tx("FIESTA %s: %d INVITADOS Y UN RATONERO GIGANTE", "PARTY %s: %d GUESTS, ONE GIANT MOUSETRAP"),
}

var pHeadFew = []paperTxt{
	tx("FIESTA %s: %d BAJAS Y CERO ESCÁNDALO", "PARTY %s: %d DOWN AND ZERO SCANDAL"),
	tx("FIESTA %s: UNA VELADA TRANQUILA, DICEN", "PARTY %s: A QUIET EVENING, THEY SAY"),
	tx("FIESTA %s: SOLO %d BAJAS, MUY DISCRETO", "PARTY %s: ONLY %d DOWN, VERY DISCREET"),
	tx("FIESTA %s: %d AUSENTES Y NADIE LO NOTÓ", "PARTY %s: %d MISSING, NOBODY NOTICED"),
	tx("FIESTA %s: UN ASESINO DE POCAS PALABRAS", "PARTY %s: A KILLER OF FEW WORDS"),
	tx("FIESTA %s: %d INVITADOS SE FUERON TEMPRANO", "PARTY %s: %d GUESTS LEFT EARLY"),
	tx("FIESTA %s: PEQUEÑO SUSTO, GRAN TORTILLA", "PARTY %s: SMALL SCARE, BIG OMELETTE"),
	tx("FIESTA %s: %d SE DESPIDEN A LA FRANCESA", "PARTY %s: %d TAKE FRENCH LEAVE"),
	tx("FIESTA %s: SALDO MODESTO, %d BAJAS", "PARTY %s: MODEST TOLL, %d DOWN"),
	tx("FIESTA %s: LA FIESTA SIGUE, %d MENOS", "PARTY %s: THE PARTY GOES ON, %d FEWER"),
	tx("FIESTA %s: EL ASESINO SE LO TOMÓ CON CALMA", "PARTY %s: THE KILLER TOOK IT EASY"),
	tx("FIESTA %s: %d BAJAS Y EL DJ, DORMIDO", "PARTY %s: %d DOWN AND THE DJ ASLEEP"),
	tx("FIESTA %s: %d MENOS, PERO LA TARTA, PEOR", "PARTY %s: %d FEWER, BUT THE CAKE FARED WORSE"),
}

var pHeadMany = []paperTxt{
	tx("FIESTA %s: %d DESAPARECIDOS Y NI UNA PROPINA", "PARTY %s: %d MISSING AND NOT A SINGLE TIP"),
	tx("FIESTA %s: SE ACABARON LOS INVITADOS", "PARTY %s: THE GUESTS RAN OUT"),
	tx("FIESTA %s: %d BAJAS, RÉCORD DE LA CASA", "PARTY %s: %d DOWN, HOUSE RECORD"),
	tx("FIESTA %s: PISTA VACÍA TRAS %d BAJAS", "PARTY %s: EMPTY FLOOR AFTER %d DOWN"),
	tx("FIESTA %s: EL ASESINO NO TUVO DESCANSO", "PARTY %s: THE KILLER NEVER TOOK A BREAK"),
	tx("FIESTA %s: QUEDÓ MÁS COMIDA QUE INVITADOS", "PARTY %s: MORE FOOD THAN GUESTS LEFT"),
	tx("FIESTA %s: %d AUSENTES, GUARDARROPA LLENO", "PARTY %s: %d MISSING, CLOAKROOM FULL"),
	tx("FIESTA %s: HOSTELERÍA EN SHOCK, %d BAJAS", "PARTY %s: CATERING IN SHOCK, %d DOWN"),
	tx("FIESTA %s: HUBO FIESTA, YA NO HAY INVITADOS", "PARTY %s: THERE WAS A PARTY, NO GUESTS NOW"),
	tx("FIESTA %s: UN ASESINO CON MUCHO TRABAJO (%d)", "PARTY %s: A KILLER WITH LOTS OF WORK (%d)"),
	tx("FIESTA %s: %d BAJAS; EL PERRO SE COMIÓ LO DEMÁS", "PARTY %s: %d DOWN; THE DOG ATE THE REST"),
	tx("FIESTA %s: SOBRAN SILLAS, %d EN CONCRETO", "PARTY %s: SPARE CHAIRS, %d TO BE EXACT"),
}

var pHeadCops = []paperTxt{
	tx("FIESTA %s: %d BAJAS Y LA POLI AVISADA TARDE", "PARTY %s: %d DOWN AND THE POLICE CALLED LATE"),
	tx("FIESTA %s: SALDO %d, Y LA POLI SIN PISTAS", "PARTY %s: TOLL %d, AND COPS CLUELESS"),
	tx("FIESTA %s: LA POLI LLEGÓ CON LA SIRENA AL REVÉS", "PARTY %s: COPS CAME SIREN BACKWARDS"),
	tx("FIESTA %s: %d BAJAS Y UN POLI PERDIDO", "PARTY %s: %d DOWN AND ONE LOST COP"),
	tx("FIESTA %s: AVISARON A LA POLI, NO LLEGÓ", "PARTY %s: COPS CALLED, NEVER ARRIVED"),
	tx("FIESTA %s: SIRENAS, ESPOSAS Y NINGÚN CULPABLE", "PARTY %s: SIRENS, CUFFS, NO CULPRIT"),
	tx("FIESTA %s: LA POLICÍA PERSIGUE A UNA SOMBRA", "PARTY %s: POLICE CHASE A SHADOW"),
	tx("FIESTA %s: UN TESTIGO MARCÓ 112 SIN ÉXITO", "PARTY %s: WITNESS DIALLED 112, NO LUCK"),
	tx("FIESTA %s: %d BAJAS Y LA POLI, DE MERIENDA", "PARTY %s: %d DOWN, COPS ON A SNACK BREAK"),
	tx("FIESTA %s: TODOS A COMISARÍA MENOS EL ASESINO", "PARTY %s: ALL TO THE STATION BUT THE KILLER"),
	tx("FIESTA %s: LA POLI INTERROGA A UN PATO", "PARTY %s: POLICE QUESTION A DUCK"),
	tx("FIESTA %s: %d BAJAS; EL AGENTE PIDIÓ CANAPÉS", "PARTY %s: %d DOWN; THE OFFICER ASKED FOR CANAPÉS"),
}

var pHeadAnimalsMany = []paperTxt{
	tx("FIESTA %s: %d ANIMALES MENOS EN LA GRANJA", "PARTY %s: %d FEWER ANIMALS ON THE FARM"),
	tx("FIESTA %s: EL ZOO DE LA CASA PIDE AUXILIO", "PARTY %s: THE HOUSE ZOO CALLS FOR HELP"),
	tx("FIESTA %s: LA PROTECTORA, INDIGNADA", "PARTY %s: THE ANIMAL SHELTER IS FURIOUS"),
	tx("FIESTA %s: %d MASCOTAS Y UN ASESINO SIN CORAZÓN", "PARTY %s: %d PETS, ONE HEARTLESS KILLER"),
	tx("FIESTA %s: GALLINAS EN FUGA, %d NO LLEGARON", "PARTY %s: HENS ON THE RUN, %d NEVER MADE IT"),
	tx("FIESTA %s: REVUELO EN LA GRANJA, %d MENOS", "PARTY %s: FARM UPROAR, %d FEWER"),
	tx("FIESTA %s: NI LAS GALLINAS SE SALVARON", "PARTY %s: NOT EVEN THE HENS WERE SAFE"),
	tx("FIESTA %s: CARNICERÍA IMPROVISADA, %d ANIMALES", "PARTY %s: IMPROMPTU BUTCHER'S, %d ANIMALS"),
	tx("FIESTA %s: %d ANIMALES Y NINGÚN PERMISO", "PARTY %s: %d ANIMALS AND NO PERMIT"),
	tx("FIESTA %s: EL VETERINARIO SE TIRA DE LOS PELOS", "PARTY %s: THE VET TEARS HIS HAIR OUT"),
}

var pHeadAnimalsFew = []paperTxt{
	tx("FIESTA %s: HASTA UN ANIMAL SE LLEVÓ LO SUYO", "PARTY %s: EVEN AN ANIMAL GOT ITS SHARE"),
	tx("FIESTA %s: LA MASCOTA NO ESTABA INVITADA", "PARTY %s: THE PET WAS NOT INVITED"),
	tx("FIESTA %s: %d BAJAS Y UNA GALLINA SOSPECHOSA", "PARTY %s: %d DOWN AND ONE SUSPICIOUS HEN"),
	tx("FIESTA %s: EL PERRO NO LADRÓ, ALGO PASA", "PARTY %s: THE DOG DID NOT BARK, SOMETHING'S UP"),
	tx("FIESTA %s: LAS GALLINAS PIDEN EXPLICACIONES", "PARTY %s: HENS DEMAND EXPLANATIONS"),
	tx("FIESTA %s: INSPECCIÓN, FALTA UN ANIMAL", "PARTY %s: INSPECTION, AN ANIMAL IS MISSING"),
	tx("FIESTA %s: PIDEN CUENTAS POR UN ANIMALITO", "PARTY %s: ANSWERS DEMANDED OVER A POOR ANIMAL"),
	tx("FIESTA %s: %d BAJAS Y UN ANIMAL DE REGALO", "PARTY %s: %d DOWN AND AN ANIMAL AS A BONUS"),
}

var pHeadMixed = []paperTxt{
	tx("FIESTA %s — SALDO: %d DESAPARECIDOS", "PARTY %s — TOLL: %d MISSING"),
	tx("FIESTA %s: %d INVITADOS NO VOLVERÁN A CASA", "PARTY %s: %d GUESTS WON'T GO HOME"),
	tx("FIESTA %s: MENÚ COMPLETO, %d BAJAS", "PARTY %s: FULL MENU, %d DOWN"),
	tx("FIESTA %s: %d AUSENTES Y UN MISTERIO", "PARTY %s: %d MISSING AND A MYSTERY"),
	tx("FIESTA %s: EL ASESINO PROBÓ DE TODO UN POCO", "PARTY %s: THE KILLER TRIED A BIT OF EVERYTHING"),
	tx("FIESTA %s: %d MENOS Y LA TORTILLA INTACTA", "PARTY %s: %d FEWER, OMELETTE UNTOUCHED"),
	tx("FIESTA %s: NOCHE MOVIDA PARA %d INVITADOS", "PARTY %s: A BUSY NIGHT FOR %d GUESTS"),
	tx("FIESTA %s: LA CASA RESPIRA TRAS %d BAJAS", "PARTY %s: THE HOUSE EXHALES AFTER %d DOWN"),
	tx("FIESTA %s: UNA FIESTA DE LO MÁS MORTAL", "PARTY %s: A DEADLY GOOD PARTY"),
	tx("FIESTA %s: QUIEN LLEGÓ TARDE, SE SALVÓ", "PARTY %s: THE LATECOMERS WERE THE LUCKY ONES"),
	tx("FIESTA %s: %d SE FUERON SIN DESPEDIRSE", "PARTY %s: %d LEFT WITHOUT SAYING GOODBYE"),
	tx("FIESTA %s: LA LISTA DE INVITADOS, MÁS CORTA", "PARTY %s: THE GUEST LIST GETS SHORTER"),
	tx("FIESTA %s: %d BAJAS Y UN HOMBRE CONO", "PARTY %s: %d DOWN AND ONE CONE MAN"),
	tx("FIESTA %s: LA TARTA PISADA, %d INVITADOS TAMBIÉN", "PARTY %s: CAKE TRAMPLED, %d GUESTS TOO"),
}

// ---- the opening sentence of the story (by the main method) -------------------------------------------------------

var pOpenKnife = []paperTxt{
	tx("El asesino de la fiesta lo hizo a la antigua: ¡ZAS, ZAS!", "The party killer did it the old way: ZAS, ZAS!"),
	tx("Un misterioso invitado repartió ¡ZAS! y ¡PLAF! sin pedir permiso.", "A mysterious guest dealt out ZAS! and PLAF! without asking permission."),
	tx("El asesino sacó el cuchillo, dijo «¡ZAS!» y dejó la fiesta sin propina.", "The killer drew the knife, said \"ZAS!\" and left the party without a tip."),
	tx("Entre canapé y canapé, alguien repartió ¡ZAS! con muy mala educación.", "Between one canapé and the next, someone dealt out ZAS! with terrible manners."),
	tx("El cuchillo hizo ¡ZAS! más veces que el DJ pinchó canciones buenas.", "The knife went ZAS! more often than the DJ played good songs."),
	tx("Un invitado con cuchillo y sin invitación hizo de las suyas: ¡ZAS, ZAS!", "A guest with a knife and no invitation did his thing: ZAS, ZAS!"),
	tx("Sin guantes ni disimulo, el asesino trinchó la fiesta como un pavo.", "With no gloves and no shame, the killer carved the party like a turkey."),
	tx("El pobre cuchillo acabó la noche con más trabajo que el camarero.", "The poor knife ended the night with more work than the waiter."),
	tx("El asesino fue directo al grano, y al hígado: ¡ZAS! ¡PLAF! ¡AY!", "The killer went straight to the point, and the liver: ZAS! PLAF! OUCH!"),
	tx("Con un cuchillo y mucha cara dura, el asesino se abrió paso entre la gente.", "With one knife and a lot of nerve, the killer carved a path through the crowd."),
	tx("Alguien afilaba el cuchillo para la tarta y se pasó de entusiasmo.", "Somebody was sharpening the knife for the cake and got a little carried away."),
	tx("Los invitados oyeron ¡ZAS! y creyeron que era un nuevo paso de baile.", "Guests heard ZAS! and thought it was a new dance move."),
}

var pOpenPoison = []paperTxt{
	tx("Según fuentes cercanas al aperitivo, el ponche llevaba un extra no pedido.", "According to sources close to the appetizers, the punch had an unrequested extra."),
	tx("El veneno fue el invitado más puntual: nadie lo vio llegar.", "The poison was the most punctual guest: nobody saw it arrive."),
	tx("El ponche sabía raro, pero nadie quiso ser maleducado y repitió.", "The punch tasted odd, but nobody wanted to be rude and had seconds."),
	tx("Los canapés hicieron efecto en tiempo récord. Qué eficaz es este catering.", "The canapés worked in record time. How efficient this catering is."),
	tx("Un frasquito, una copa y mucha sangre fría: así cayeron los brindis.", "One little vial, one glass and plenty of nerve: that is how the toasts fell."),
	tx("Se sospecha que el cocinero no probó su receta, y ya se sabe por qué.", "The cook is suspected of never tasting his own recipe, and now we know why."),
	tx("Todos brindaron, pero algunos brindaron por última vez.", "Everyone drank a toast, but some drank it for the last time."),
	tx("El veneno llegó sin hacer ruido y se fue sin pagar el cubierto.", "The poison came in silently and left without paying for the meal."),
	tx("Cayeron uno a uno tras la primera copa, sin ni un «salud».", "They dropped one by one after the first glass, without even a \"cheers\"."),
	tx("Alguien echó al ponche un ingrediente secreto. Se llama «adiós».", "Somebody put a secret ingredient in the punch. It is called \"goodbye\"."),
	tx("Las copas iban llenas y los invitados, cada vez más vacíos.", "The glasses were full and the guests were getting emptier."),
	tx("El camarero jura que solo sirvió bebidas. Lo de dentro es otra historia.", "The waiter swears he only served drinks. What was inside is another story."),
}

var pOpenTrap = []paperTxt{
	tx("Las trampas hicieron ¡CLAC! y ¡PUM! sin que nadie leyera el manual.", "The traps went CLACK! and BOOM! without anyone reading the manual."),
	tx("El asesino de la fiesta dejó trampas por todas partes. ¡CLAC!", "The party killer left traps everywhere. CLACK!"),
	tx("Cada rincón tenía su trampita y cada trampita, su invitado.", "Every corner had its little trap and every little trap, its guest."),
	tx("El asesino montó las trampas con la delicadeza de quien monta un belén.", "The killer set up the traps as delicately as someone setting up a nativity scene."),
	tx("Hubo ¡PUM! en el pasillo, ¡CLAC! en la cocina y ¡AY! en el baño.", "There was a BOOM in the hall, a CLACK in the kitchen and an OUCH in the bathroom."),
	tx("Los carteles de «Cuidado» estaban, pero en una fiesta nadie lee.", "The \"Careful\" signs were there, but nobody reads at a party."),
	tx("Un ratonero gigante y varios invitados con muy mala suerte: ¡CLAC!", "One giant mousetrap and several very unlucky guests: CLACK!"),
	tx("El asesino prefirió que el suelo trabajara por él. Y el suelo trabajó.", "The killer let the floor do the work. And the floor delivered."),
	tx("Las trampas funcionaron mejor que el aire acondicionado.", "The traps worked better than the air conditioning."),
	tx("El asesino ni se manchó las manos: dejó que la casa hiciera ¡CLAC!", "The killer never even got his hands dirty: he let the house go CLACK!"),
	tx("Con algo de bricolaje y mala idea, el asesino convirtió la casa en parque temático.", "With some DIY and a bad idea, the killer turned the house into a theme park."),
}

var pOpenBlast = []paperTxt{
	tx("Un ¡BOOM! de los gordos dejó la pista sin sobremesa.", "A big BOOM! left the dance floor without an after-dinner chat."),
	tx("Los vecinos oyeron ¡BOOM! y creyeron que era el DJ.", "The neighbours heard BOOM! and thought it was the DJ."),
	tx("La pista despegó sin avisar. Primero ¡BOOM! y luego, silencio.", "The dance floor took off without warning. First BOOM! and then silence."),
	tx("Los fuegos artificiales llegaron antes de medianoche, y desde dentro.", "The fireworks arrived before midnight, and from the inside."),
	tx("Un ¡BOOM! hizo temblar la lámpara y callar al DJ. Un récord.", "A BOOM! made the lamp tremble and the DJ shut up. A record."),
	tx("Los vecinos llamaron por el ruido... por el ¡BOOM!, no por la música.", "The neighbours called about the noise... the BOOM!, not the music."),
	tx("Alguien confundió la tarta con un explosivo, o al revés. ¡BOOM!", "Someone mixed up the cake and an explosive, or the other way round. BOOM!"),
	tx("Hubo confeti, serpentinas y un ¡BOOM! que no venía en el programa.", "There was confetti, streamers and a BOOM! that was not in the programme."),
	tx("La explosión dejó el techo con vistas y a los invitados con prisa.", "The blast left the roof with a view and the guests in a hurry."),
	tx("El asesino quiso animar la fiesta y se le fue la mano. ¡BOOM!", "The killer wanted to liven up the party and overdid it. BOOM!"),
	tx("Según los bomberos, ¡BOOM! es la onomatopeya oficial del suceso.", "According to the firefighters, BOOM! is the official sound of the incident."),
}

var pOpenMixed = []paperTxt{
	tx("Cuchillo, veneno y trampas: un menú degustación completo.", "Knife, poison and traps: a complete tasting menu."),
	tx("El asesino de la fiesta no tiene método: lo prueba todo.", "The party killer has no method: he tries everything."),
	tx("Un poco de cuchillo, un poco de veneno y una pizca de trampa: receta de la casa.", "A little knife, a little poison and a pinch of trap: the house recipe."),
	tx("El asesino cambió de método más veces que el DJ de canción.", "The killer changed methods more often than the DJ changed songs."),
	tx("Ni un método repetido: el asesino se aburre con facilidad.", "Not one method repeated: the killer bores easily."),
	tx("Cada invitado cayó de una forma distinta. Hay que admirar la creatividad.", "Every guest fell a different way. You have to admire the creativity."),
	tx("Hubo de todo: cuchillo, veneno, trampas y hasta un poco de teatro.", "There was a bit of everything: knife, poison, traps and even some drama."),
	tx("Los forenses necesitan una hoja de cálculo para clasificar los métodos.", "The forensics team needs a spreadsheet to sort out the methods."),
	tx("Al asesino no le gusta la rutina: hoy cuchillo, mañana veneno.", "The killer hates routine: knife today, poison tomorrow."),
	tx("El asesino probó todos los métodos del catálogo y pidió más.", "The killer tried every method in the catalogue and asked for more."),
}

var pOpenNone = []paperTxt{
	tx("Los invitados sobrevivieron, y el asesino no sabe qué decir.", "The guests survived, and the killer does not know what to say."),
	tx("El asesino llegó a la fiesta, miró el buffet y se quedó a comer.", "The killer came to the party, looked at the buffet and stayed to eat."),
	tx("Hoy no hubo ni un ¡ZAS! ni un ¡PLAF!: el cuchillo se echó la siesta.", "Not a single ZAS! or PLAF! today: the knife took a nap."),
	tx("Nadie faltó a la cita. Ni siquiera el asesino, que se portó bien.", "Nobody missed the date. Not even the killer, who behaved."),
	tx("Fiesta sin sobresaltos: el asesino estaba más atento a la tortilla.", "A party without a scare: the killer was more interested in the omelette."),
	tx("Los forenses esperaban trabajo y se fueron con un trozo de tarta.", "The forensics team expected work and went home with a slice of cake."),
	tx("El asesino dice que hoy no tenía el día. Los invitados, encantados.", "The killer says he was not in the mood. The guests, delighted."),
	tx("La noche fue tan tranquila que la policía echó una siesta en el coche.", "The night was so quiet the police took a nap in the car."),
	tx("El único asesinato fue el del karaoke. Con el micrófono.", "The only murder was the one the karaoke suffered. With the microphone."),
	tx("Se esperaba un crimen y hubo un baile. Los cronistas piden disculpas.", "A crime was expected and a dance happened. The chroniclers apologise."),
}

var pOpenFew = []paperTxt{
	tx("Fueron bajas discretas: casi nadie se enteró del asunto.", "They were discreet casualties: hardly anyone noticed."),
	tx("Pocas bajas y mucho disimulo: el asesino va de fino.", "Few casualties and plenty of discretion: the killer has class."),
}

var pOpenManyK = []paperTxt{
	tx("Los forenses perdieron la cuenta a mitad de canapés.", "The forensics team lost count halfway through the canapés."),
	tx("La lista de bajas es tan larga que el diario la publica por entregas.", "The casualty list is so long this paper will run it in instalments."),
}

// ---- optional sentences ------------------------------------------------------------------------------------------

var pSeen0 = []paperTxt{
	tx("Nadie vio nada. Los testigos juran estar mirando el móvil.", "Nobody saw a thing. Witnesses swear they were looking at their phones."),
	tx("Ni un solo testigo: todos miraban al DJ, claro.", "Not a single witness: everyone was watching the DJ, of course."),
	tx("Cero testigos. Hasta el pato de la piscina dice que no estaba.", "Zero witnesses. Even the duck in the pool says he was not there."),
	tx("Nadie lo vio, ni siquiera el espejo del baño.", "Nobody saw him, not even the bathroom mirror."),
	tx("Los testigos estaban muy ocupados con la tortilla.", "The witnesses were very busy with the omelette."),
	tx("Se busca un testigo: el asesino pasó tan desapercibido que casi se lo pierde él.", "A witness is wanted: the killer went so unnoticed he almost missed it himself."),
	tx("Sin testigos y sin huellas: la policía dibuja un monigote en la pizarra.", "No witnesses and no prints: the police draw a stick figure on the board."),
	tx("Nadie vio nada. Los invitados estaban ocupados bailando mal.", "Nobody saw anything. The guests were busy dancing badly."),
	tx("El asesino se movió con tanto sigilo que los zapatos le pedían calma.", "The killer moved so quietly his own shoes asked him to calm down."),
	tx("Todos miraban hacia otro lado. Era un buen momento para mirar hacia otro lado.", "Everyone looked the other way. It was a good time to look the other way."),
}

var pSeenFew = []paperTxt{
	tx("Testigos aseguran haberlo visto, aunque iba disfrazado de sombra.", "Witnesses claim to have seen him, though he was dressed as a shadow."),
	tx("Un testigo lo vio, pero dice que fue sin querer.", "A witness saw him, but says it was by accident."),
	tx("Una señora con bolso lo vio todo y lo contó a medias.", "A lady with a handbag saw it all and told half of it."),
	tx("Un par de invitados lo vieron, pero creyeron que era parte del baile.", "A couple of guests saw him but thought it was part of the dance."),
	tx("Un testigo dice que era alto. Otro, que era bajo. Un tercero, que era un abrigo.", "One witness says he was tall. Another, short. A third, that it was a coat."),
	tx("El único testigo se lo contó al camarero, que se lo contó a la tortilla.", "The only witness told the waiter, who told the omelette."),
	tx("Alguien lo vio, pero estaba en mitad de un baile y no se atrevió a parar.", "Somebody saw him, but was mid-dance and did not dare stop."),
	tx("Hubo testigos, sí, pero de esos que vieron la tarta, no al asesino.", "There were witnesses, yes, but the kind who saw the cake, not the killer."),
	tx("Un testigo ofrece una descripción: «Un señor... con cara de señor».", "A witness offers a description: \"A man... with a man's face\"."),
	tx("Lo vieron de espaldas, de perfil y a lo lejos. En resumen, no lo vieron.", "He was seen from behind, in profile and from afar. In short, not seen."),
}

var pSeenMany = []paperTxt{
	tx("Hubo más testigos que canapés y todos corrían a chivarse.", "There were more witnesses than canapés and all of them ran to tell."),
	tx("Testigos aseguran haberlo visto cuatro, cinco, ¡mil veces!", "Witnesses claim to have seen him four, five, a thousand times!"),
	tx("Medio salón lo vio y el otro medio lo oyó, pero nadie levantó la ceja.", "Half the room saw him and the other half heard him, but nobody raised an eyebrow."),
	tx("Con tanto testigo, la policía pidió un número para esperar turno.", "With so many witnesses, the police asked for a number to wait their turn."),
	tx("Lo vio hasta el gato del vecino, y ya es decir.", "Even the neighbour's cat saw him, and that says a lot."),
	tx("Los testigos no se ponen de acuerdo, pero todos lo vieron. Y quieren cobrar.", "The witnesses disagree, but they all saw him. And want to be paid."),
	tx("Todos lo vieron y todos corrieron: una buena fiesta, si hubiera sido de atletismo.", "Everyone saw him and everyone ran: a fine party, if it had been an athletics meet."),
	tx("Con tanta gente mirando, el asesino saludó a la cámara. Qué cara.", "With so many people watching, the killer waved at the camera. What nerve."),
	tx("El asesino fue visto por tantos que ya casi firma autógrafos.", "The killer was seen by so many that he almost signs autographs."),
	tx("Se formó cola para declarar. Se sirvieron aperitivos mientras tanto.", "A queue formed to testify. Appetizers were served in the meantime."),
}

var pCuffed = []paperTxt{
	tx("Alguien avisó a la policía, pero el sospechoso se esfumó.", "Somebody called the police, but the suspect vanished."),
	tx("Un testigo llamó a la poli. Casi lo pillan... y se escapó.", "A witness called the cops. They almost caught him... and he escaped."),
	tx("Las esposas llegaron tarde y se quedaron sin sospechoso.", "The handcuffs came late and were left without a suspect."),
	tx("Los agentes persiguieron al asesino... hasta que se les cayó el casco.", "The officers chased the killer... until one dropped his helmet."),
	tx("Un agente grita «¡Alto!» y el asesino, muy educado, se paró cinco segundos.", "An officer shouted \"Halt!\" and the killer, very politely, stopped for five seconds."),
	tx("La policía casi lo detiene; solo le faltó el sospechoso.", "The police almost arrested him; all they lacked was the suspect."),
	tx("Se avisó a la poli con tanta calma que llegó con un pastel.", "The police were called so calmly that they arrived with a pie."),
	tx("Un poli confundió al DJ con el asesino. El DJ se lo tomó bien.", "A cop mistook the DJ for the killer. The DJ took it well."),
}

var pJailed = []paperTxt{
	tx("Uno de los asesinos pasó por el calabozo; el otro, de rositas.", "One of the killers spent time in the cell; the other got off scot-free."),
	tx("Hubo detenido, hubo fianza y hubo mucha cara dura.", "There was an arrest, there was bail and there was plenty of nerve."),
	tx("El detenido pidió su llamada y llamó al DJ, para variar.", "The detainee asked for his call and rang the DJ, for a change."),
	tx("Del calabozo salió con un nuevo amigo: el poli de guardia.", "He left the cell with a new friend: the guard on duty."),
	tx("La fianza se pagó en canapés. El juez aceptó dos de tortilla.", "Bail was paid in canapés. The judge accepted two omelette ones."),
	tx("El compañero del detenido aprovechó para seguir trabajando. Qué profesional.", "The detainee's partner kept on working in the meantime. What a professional."),
}

var pCamera = []paperTxt{
	tx("Una cámara lo grabó, pero la imagen salió fatal.", "A camera filmed him, but the picture came out terrible."),
	tx("La cámara de seguridad lo pilló, y encima salió mal peinado.", "The security camera caught him, and he came out with bad hair too."),
	tx("La cámara grabó todo, aunque el vigilante estaba viendo otro canal.", "The camera recorded everything, though the guard was watching another channel."),
	tx("Las imágenes de la cámara son clarísimas: se ve un dedo. Solo un dedo.", "The camera footage is crystal clear: you can see a finger. Just a finger."),
	tx("Una cámara lo grabó de frente. Por desgracia, estaba apuntando al techo.", "A camera filmed him head-on. Sadly, it was pointing at the ceiling."),
	tx("La cámara lo grabó y la policía lo vio... en cámara lenta, tres días después.", "The camera recorded him and the police watched it... in slow motion, three days later."),
	tx("El vídeo es oro: sale el asesino, la tortilla y un pato.", "The video is gold: it shows the killer, the omelette and a duck."),
	tx("La cámara lo grabó y el asesino saludó. Ahora tiene seguidores.", "The camera recorded him and the killer waved. Now he has followers."),
}

var pSlasher = []paperTxt{
	tx("Por si fuera poco, otro asesino rondaba por allí. ¡Competencia!", "As if that were not enough, another killer was around. Competition!"),
	tx("Otro asesino andaba suelto. Qué fiesta más concurrida.", "Another killer was on the loose. What a crowded party."),
	tx("Un segundo asesino, con máscara y mala educación, dio la sorpresa.", "A second killer, with a mask and bad manners, was a surprise."),
	tx("Dos asesinos en la misma fiesta y ninguno cedió el paso.", "Two killers at the same party and neither gave way."),
	tx("El asesino de la máscara y el otro se miraron mal. Era territorio ocupado.", "The masked killer and the other one glared at each other. It was occupied territory."),
	tx("El de la máscara llegó sin avisar y se hizo con la mitad del trabajo.", "The masked one arrived without warning and took half the work."),
	tx("Un asesino extra apareció por el pasillo. Se rumorea que trae su propia tarta.", "An extra killer appeared in the hall. It is rumoured he brings his own cake."),
	tx("Con dos asesinos sueltos, el guardarropa ya no daba abasto.", "With two killers on the loose, the cloakroom could no longer cope."),
}

var pMilitary = []paperTxt{
	tx("El ejército desembarcó con tanques para un simple aperitivo.", "The army landed with tanks for a simple appetizer."),
	tx("Los tanques aparcaron en el jardín y aplastaron el parterre.", "The tanks parked in the garden and flattened the flower bed."),
	tx("El ejército llegó con tanques, banda de música y ninguna pista.", "The army arrived with tanks, a brass band and no clues."),
	tx("Un general pidió una tortilla y rodeó la casa con diez tanques.", "A general asked for an omelette and surrounded the house with ten tanks."),
	tx("Los soldados formaron en el jardín y saludaron al seto, que no contestó.", "The soldiers lined up in the garden and saluted the hedge, which did not reply."),
	tx("Un tanque se confundió de calle y acabó en la pista de baile.", "A tank took a wrong turn and ended up on the dance floor."),
}

var pSWAT = []paperTxt{
	tx("Llegó el SWAT, con megáfono, cuerda y todo.", "The SWAT team came, with megaphone, rope and all."),
	tx("El SWAT entró por el tejado y salió por la puerta, muy digno.", "SWAT came in through the roof and went out by the door, quite dignified."),
	tx("El SWAT bajó rapelando y se enredó con la lámpara.", "SWAT came down on ropes and got tangled in the lamp."),
	tx("El SWAT rodeó la casa dos veces; la tercera, tomó aperitivo.", "SWAT surrounded the house twice; the third time, they had appetizers."),
	tx("Los del SWAT derribaron la puerta, que estaba abierta.", "The SWAT men broke down the door, which was open."),
	tx("Un agente del SWAT pidió «silencio» por el megáfono. La música no oyó.", "A SWAT officer asked for \"silence\" through the megaphone. The music did not hear."),
}

var pRangers = []paperTxt{
	tx("Hasta unos rangers de colores llegaron dando saltos.", "Even some colourful rangers arrived, leaping about."),
	tx("Los rangers aterrizaron en pose y no atraparon a nadie, pero posaron bien.", "The rangers landed in pose and caught nobody, but they posed well."),
	tx("Cinco rangers de colores gritaron a la vez. Ninguno dijo lo mismo.", "Five colourful rangers shouted at once. None said the same thing."),
	tx("Los rangers hicieron una coreografía y se fueron sin detener a nadie.", "The rangers did a choreography and left without arresting anyone."),
	tx("Los rangers rojos y los azules discutieron quién llegaba primero.", "The red rangers and the blue ones argued over who arrived first."),
	tx("Los rangers llegaron en moto, en monopatín y en una mini nevera.", "The rangers came by motorbike, skateboard and a mini fridge."),
}

var pFBI = []paperTxt{
	tx("Dos agentes del FBI aparecieron muy serios y sin una pista.", "Two FBI agents showed up very serious and without a clue."),
	tx("Los agentes del FBI se pusieron gafas oscuras. Era de noche, pero da igual.", "The FBI agents put on dark glasses. It was night, but never mind."),
	tx("El FBI llegó, miró el buffet y declaró «caso complejo».", "The FBI arrived, looked at the buffet and declared \"complex case\"."),
	tx("Dos del FBI interrogaron a una planta. La planta se acogió a la quinta.", "Two FBI men questioned a plant. The plant pleaded the fifth."),
	tx("El FBI abrió un expediente de tres páginas. Dos eran de la tortilla.", "The FBI opened a three-page file. Two pages were about the omelette."),
	tx("Los del FBI buscaron huellas en la tarta y dejaron las suyas.", "The FBI men looked for prints on the cake and left their own."),
}

var pPolice = []paperTxt{
	tx("Una patrulla acudió con sirena y cero resultados.", "A patrol arrived with a siren and zero results."),
	tx("La policía llegó tarde, como siempre. ¡NI-NO, NI-NO!", "The police arrived late, as always. WEE-WOO, WEE-WOO!"),
	tx("La policía llegó con la sirena puesta y la linterna apagada.", "The police arrived with the siren on and the torch off."),
	tx("Un agente interrogó a una lámpara. La lámpara no vio nada.", "An officer questioned a lamp. The lamp saw nothing."),
	tx("Los agentes acordonaron el jardín, y también al jardinero.", "The officers cordoned off the garden, and the gardener too."),
	tx("Un poli resbaló en el vestíbulo y dio el parte desde el suelo.", "A cop slipped in the hall and filed his report from the floor."),
	tx("La patrulla aparcó mal y recibió una multa de la propia policía.", "The patrol parked badly and was fined by the police themselves."),
	tx("Dos agentes tomaron declaración a una tortilla. Se negó a hablar.", "Two officers took a statement from an omelette. It refused to talk."),
	tx("Un poli pidió refuerzos y llegó otro poli. Los refuerzos eran eso.", "A cop called for backup and another cop arrived. That was the backup."),
	tx("La poli se llevó a un pato para interrogarlo. Salió bajo fianza.", "The cops took a duck in for questioning. He got out on bail."),
}

var pNoPolice = []paperTxt{
	tx("Ni una sirena: la ciudad dormía a pierna suelta.", "Not a siren: the city was sleeping like a log."),
	tx("La policía no apareció. Dicen que estaban buscando aparcamiento.", "The police never showed up. They say they were looking for parking."),
	tx("Nadie llamó a la poli. Todos pensaron que ya lo habría hecho otro.", "Nobody called the police. Everybody thought somebody else would."),
	tx("Ni un coche patrulla: estaban de reunión sobre aparcamientos.", "Not a patrol car: they were at a meeting about parking."),
	tx("La comisaría no cogió el teléfono. Era la hora del bocadillo.", "The police station did not pick up the phone. It was sandwich time."),
	tx("Sin policía, sin sirenas y sin quejas: el barrio se hizo el sueco.", "No police, no sirens and no complaints: the neighbourhood played dumb."),
	tx("La patrulla del barrio pasó de largo, silbando una melodía.", "The neighbourhood patrol drove past, whistling a tune."),
	tx("Ni un agente en tres calles. Se sospecha que están de huelga de zapatos.", "Not an officer in three streets. They are rumoured to be on a shoe strike."),
}

var pMedic = []paperTxt{
	tx("Las ambulancias hicieron horas extra. ¡PIII-PIII!", "The ambulances worked overtime. BEEP-BEEP!"),
	tx("Las ambulancias llegaron y se fueron vacías, con cara de pena.", "The ambulances came and left empty, looking sad."),
	tx("Los sanitarios llegaron con camilla y se llevaron un canapé.", "The paramedics arrived with a stretcher and took away a canapé."),
	tx("Una ambulancia se perdió y apareció en otra fiesta. Fue la que más se divirtió.", "An ambulance got lost and turned up at another party. It had the most fun."),
	tx("Los sanitarios llegaron corriendo y se quedaron sin aliento. Qué ironía.", "The paramedics ran in and ran out of breath. What irony."),
	tx("Se pidió una ambulancia y vinieron dos, por si acaso.", "One ambulance was requested and two came, just in case."),
	tx("El médico de guardia dice que lo suyo es la tortilla, no los accidentes.", "The doctor on call says his thing is omelette, not accidents."),
	tx("Un sanitario tomó el pulso a un maniquí y declaró: «Hay esperanza».", "A paramedic took a mannequin's pulse and declared, \"There is hope\"."),
}

var pChain = []paperTxt{
	tx("Mató en cadena, ¡ZAS-ZAS-ZAS!, sin parar a respirar.", "Killed in a chain, ZAS-ZAS-ZAS!, without stopping to breathe."),
	tx("Una racha de bajas tan seguida que ni dio tiempo al brindis.", "A streak of casualties so rapid there was no time for a toast."),
	tx("Una, dos, tres... el asesino perdió la cuenta, y se puso a contar otra vez.", "One, two, three... the killer lost count and started counting again."),
	tx("Cayeron como fichas de dominó, con el mismo ruido y menos gracia.", "They fell like dominoes, with the same noise and less grace."),
	tx("El asesino encadenó tantas bajas que pidió un descanso a mitad.", "The killer chained so many that he asked for a break halfway."),
	tx("Fue una racha, un maratón y casi un récord de la casa.", "It was a streak, a marathon and almost a house record."),
	tx("Los testigos cuentan que no hubo tiempo ni de pestañear. Ni de gritar.", "Witnesses say there was no time to blink. Or scream."),
	tx("Las bajas fueron seguidas, sin pausa y con mucha técnica.", "The hits came one after another, without pause and with plenty of technique."),
}

var pTwo = []paperTxt{
	tx("Los asesinos se repartieron el trabajo. Lo llaman equipo.", "The killers shared the work. They call it teamwork."),
	tx("Dos asesinos, un solo plan y cero comunicación. Funcionó.", "Two killers, one plan and zero communication. It worked."),
	tx("Entre los dos asesinos se pasaron el cuchillo como quien pasa la sal.", "Between them, the two killers passed the knife like someone passing the salt."),
	tx("Los dos asesinos discutieron quién había hecho más. Ganó el DJ, por abstención.", "The two killers argued over who had done more. The DJ won, by abstention."),
	tx("Dos asesinos y una lista de invitados cada vez más corta.", "Two killers and an ever shorter guest list."),
	tx("A cuatro manos, la fiesta se quedó vacía mucho antes del postre.", "With four hands at work, the party was empty well before dessert."),
	tx("Los dos asesinos se chocaron en el pasillo y se pidieron perdón. Educados.", "The two killers bumped in the hall and said sorry. Polite."),
	tx("Hubo reparto de tareas: uno al cuchillo y el otro a hacer ruido.", "Tasks were shared: one on the knife and the other on making noise."),
}

var pChallenge = []paperTxt{
	tx("Se rumorea que lo hizo todo por un premio extra.", "It is rumoured he did it all for an extra prize."),
	tx("El asesino iba a por el reto, y el reto, a por él.", "The killer went for the challenge, and the challenge went for him."),
	tx("Dicen que apostó triple y quiso cobrar triple. Qué listo.", "They say he bet triple and wanted to collect triple. How clever."),
	tx("Hubo un reto de por medio, y el asesino no es de los que se rajan.", "There was a challenge involved, and the killer is not one to back out."),
	tx("Quien apuesta fuerte juega fuerte: el reto salió a pedir de boca.", "Whoever bets big plays big: the challenge went swimmingly."),
	tx("Los apostantes del barrio miraron el marcador y se rascaron la cartera.", "The neighbourhood punters looked at the scoreboard and scratched their wallets."),
}

var pAnimalsDead = []paperTxt{
	tx("Los animales de la casa tampoco salieron ilesos. Qué falta de respeto.", "The house animals did not come out unharmed either. What disrespect."),
	tx("Hasta la gallina pagó el pato, aunque el pato estaba ausente.", "Even the hen paid the price, though the duck was absent."),
	tx("Las mascotas no estaban en la lista, pero cayeron igualmente.", "The pets were not on the list, but fell anyway."),
	tx("La protectora de animales ya redacta una carta... de varias páginas.", "The animal shelter is already drafting a letter... several pages long."),
	tx("Hubo víctimas con plumas, con pelo y con cara de no entender nada.", "There were victims with feathers, with fur and with a face of utter confusion."),
	tx("Un animal del jardín fue de los primeros en irse. Se le echará de menos.", "A garden animal was among the first to go. It will be missed."),
	tx("Hasta los animales miraron al asesino con mala cara. Los que quedaban.", "Even the animals gave the killer dirty looks. The ones that were left."),
	tx("El veterinario, que pasaba por allí, prefirió cambiar de acera.", "The vet, who was passing by, preferred to cross the street."),
	tx("La granja está de luto, y la sopa de gallina, de mal gusto.", "The farm is in mourning, and the chicken soup in poor taste."),
	tx("Los animales no habían hecho nada, salvo existir. Y eso, al parecer, bastó.", "The animals had done nothing but exist. And that, it seems, was enough."),
}

var pAnimalsMany = []paperTxt{
	tx("Fue una masacre animal: ni un cencerro quedó sonando.", "It was an animal massacre: not a single cowbell was left ringing."),
	tx("Los pocos animales que escaparon piden asilo en casa del vecino.", "The few animals that escaped are asking for asylum at the neighbour's."),
	tx("La granja entera quedó en silencio. Hasta los grillos pidieron explicaciones.", "The whole farm fell silent. Even the crickets demanded explanations."),
	tx("Se llamó a la protectora, a la perrera y a un señor con un gorro de pastor.", "The shelter, the pound and a man in a shepherd's hat were called."),
	tx("La lista de bajas parece un cuento infantil: gallinas, patos y un perro.", "The casualty list reads like a children's story: hens, ducks and a dog."),
	tx("Las gallinas que quedaron se escondieron bajo el sofá, y no son de esconderse.", "The remaining hens hid under the sofa, and they are not the hiding type."),
	tx("Los animales llevan crespón negro y los patos, un gorro de luto.", "The animals wear black armbands and the ducks wear a mourning hat."),
	tx("Un pastor de ovejas ofreció su ayuda. Fue rechazada por lo evidente.", "A shepherd offered his help. It was declined, for obvious reasons."),
}

var pAnimalsAlive = []paperTxt{
	tx("Los animales de la casa salieron ilesos y con cara de haberlo visto todo.", "The house animals came out unharmed, looking like they had seen it all."),
	tx("Ni un pelo ni una pluma fuera de su sitio: los animales, a salvo.", "Not a hair or a feather out of place: the animals, safe."),
	tx("Hubo un perro que no ladró y una gallina que no puso un huevo. Sospechoso.", "There was a dog that did not bark and a hen that did not lay an egg. Suspicious."),
	tx("Los animales respetaron el toque de queda y la tortilla, también.", "The animals respected the curfew, and so did the omelette."),
	tx("Una gallina declaró que nunca había visto una fiesta tan movida.", "A hen stated she had never seen such a lively party."),
	tx("Las mascotas se salvaron. El asesino debe de ser de los que quieren a los animales.", "The pets were spared. The killer must be one of those animal lovers."),
	tx("Los animales cuentan que se escondieron bajo la mesa. La mesa lo confirma.", "The animals say they hid under the table. The table confirms it."),
	tx("El perro de la casa dormía tan a gusto que se perdió la noticia.", "The house dog slept so soundly he missed the news."),
	tx("Hasta el toro permaneció tranquilo. Dijo que estaba de vacaciones.", "Even the bull stayed calm. He said he was on holiday."),
	tx("Las serpientes del jardín se portaron mejor que algunos invitados.", "The garden snakes behaved better than some of the guests."),
}

var pIsland = []paperTxt{
	tx("Un tiranosaurio con sombrero hizo de anfitrión y repitió de todo.", "A tyrannosaur in a hat played host and asked for seconds."),
	tx("En la isla, el dinosaurio más grande probó los canapés. Y a quien los traía.", "On the island, the biggest dinosaur tried the canapés. And whoever brought them."),
	tx("Los dinosaurios llegaron sin invitación, pero al menos trajeron apetito.", "The dinosaurs came uninvited, but at least they brought an appetite."),
	tx("El T-Rex rugió y el DJ bajó el volumen, por si acaso.", "The T-Rex roared and the DJ turned the volume down, just in case."),
	tx("Un T-Rex se comió la tarta y a dos invitados. Dice que no sabía que eran para otros.", "A T-Rex ate the cake and two guests. He says he did not know they were for others."),
	tx("Los velociraptors pidieron mesa para cinco. No tenían reserva.", "The velociraptors asked for a table for five. They had no reservation."),
}

var pDJ = []paperTxt{
	tx("El DJ siguió pinchando como si nada. Es lo que se llama vocación.", "The DJ kept spinning as if nothing happened. That is called vocation."),
	tx("El DJ dijo que no oyó nada, y era cierto, con tanto bajo.", "The DJ said he heard nothing, and it was true, with all that bass."),
	tx("El DJ cambió de canción justo a tiempo. O tarde. Depende de a quién se le pregunte.", "The DJ changed the song just in time. Or late. Depends who you ask."),
	tx("El DJ puso otra de reguetón y acabó con lo poco que quedaba de calma.", "The DJ put on more reggaeton and finished off what little calm remained."),
	tx("Cuando el DJ paró la música, todos creyeron que había pasado algo. Era un descanso.", "When the DJ stopped the music, everybody thought something had happened. It was a break."),
	tx("El DJ ofreció una canción como coartada. Duraba seis minutos.", "The DJ offered a song as an coartada. It lasted six minutes."),
	tx("Sospechoso número uno: el DJ, por poner siempre la misma canción.", "Suspect number one: the DJ, for always playing the same song."),
	tx("El DJ no vio nada, no oyó nada y, tras investigar, no pinchó nada.", "The DJ saw nothing, heard nothing and, after investigation, played nothing."),
}

var pFast = []paperTxt{
	tx("Todo ocurrió tan rápido que los invitados aún estaban saludando.", "It all happened so fast the guests were still saying hello."),
	tx("La fiesta duró lo que tarda en enfriarse una croqueta.", "The party lasted as long as it takes a croquette to cool."),
	tx("Fue un visto y no visto: a media canción, ya estaba todo hecho.", "It was over in a flash: halfway through a song, it was all done."),
	tx("El asesino llegó, hizo lo suyo y todavía quedaba tortilla caliente.", "The killer arrived, did his thing and the omelette was still warm."),
	tx("Duró tan poco que el DJ aún no había encontrado su lista.", "It lasted so little that the DJ had not even found his playlist."),
	tx("Rapidez de manual: el asesino dejó a los forenses sin tiempo de sentarse.", "Textbook speed: the killer left the forensics team no time to sit down."),
}

var pSlow = []paperTxt{
	tx("La velada fue larga, tanto que la tortilla se enfrió dos veces.", "The evening was long, so long the omelette got cold twice."),
	tx("El asesino se tomó su tiempo: la fiesta duró más que el telediario.", "The killer took his time: the party lasted longer than the news."),
	tx("Fue una noche eterna, con intermedio y todo.", "It was an endless night, with an intermission and everything."),
	tx("Hubo tiempo para todo: bailar, brindar y despedirse para siempre.", "There was time for everything: dancing, toasting and saying goodbye forever."),
	tx("Se hizo tan larga que algunos invitados cenaron dos veces.", "It dragged on so long that some guests had dinner twice."),
	tx("Cuando por fin acabó, el DJ ya había repetido la lista tres veces.", "When it finally ended, the DJ had repeated the playlist three times."),
}

var pLevelEarly = []paperTxt{
	tx("Los más veteranos comentan que es una fiesta de principiantes.", "Seasoned folk say this is a beginners' party."),
	tx("Aún era temprano en la temporada, y el asesino ya estaba calentando.", "It was early in the season, and the killer was already warming up."),
	tx("Primeras fiestas del año: el asesino todavía aprende dónde está la cocina.", "The first parties of the year: the killer is still learning where the kitchen is."),
	tx("Fiesta de calentamiento. El plato fuerte llega más tarde.", "A warm-up party. The main course comes later."),
	tx("Todo apunta a que esto es solo el aperitivo de una larga temporada.", "Everything suggests this is only the appetizer of a long season."),
	tx("El asesino va cogiendo confianza: la próxima fiesta, más.", "The killer is gaining confidence: next party, more."),
}

var pLevelLate = []paperTxt{
	tx("A estas alturas, el asesino ya tiene tarjeta de cliente en el guardarropa.", "By now the killer has a loyalty card at the cloakroom."),
	tx("Los más veteranos dicen que esta fiesta fue de las difíciles, y son pocos.", "The veterans say this party was a tough one, and there are few of them."),
	tx("El asesino lleva tantas fiestas encima que ya conoce a la mitad de los canapés.", "The killer has been to so many parties he knows half the canapés."),
	tx("Fiesta de nivel alto: hasta los porteros pidieron cambiar de turno.", "A high-level party: even the doormen asked to switch shifts."),
	tx("Con tanta experiencia, el asesino ya tacha invitados de memoria.", "With so much experience, the killer crosses off guests from memory."),
	tx("Los vecinos ya no se asoman cuando oyen música: saben cómo acaba.", "The neighbours no longer look out when they hear music: they know how it ends."),
}

var pExtra = []paperTxt{
	tx("Fiesta fuera de calendario: nadie sabe quién la convocó.", "A party out of the calendar: nobody knows who called it."),
	tx("Fiesta extra, de esas que no vienen en la agenda ni en la lista de la compra.", "An extra party, the kind that is neither in the diary nor on the shopping list."),
	tx("Se trata de una fiesta sorpresa. La mayoría de los invitados, también.", "It was a surprise party. Most of the guests were surprised too."),
	tx("Las fiestas de última hora son las mejores: poca gente cuerda y mucho que contar.", "Last-minute parties are the best: few sane guests and plenty to tell."),
	tx("Fiesta extra, con extras: más ruido, más canapés y menos invitados.", "An extra party, with extras: more noise, more canapés and fewer guests."),
	tx("Fiesta improvisada, resultado improvisado.", "An improvised party, an improvised result."),
}

var pClose = []paperTxt{
	tx("El culpable sigue en libertad, seguramente comiendo tortilla.", "The culprit remains at large, probably eating omelette."),
	tx("El culpable sigue suelto. Se le ofrece un canapé por su captura.", "The culprit is still loose. A canapé is offered for his capture."),
	tx("La policía promete investigar, en cuanto encuentre la comisaría.", "The police promise to investigate, as soon as they find the station."),
	tx("El caso queda abierto, igual que el buffet.", "The case remains open, just like the buffet."),
	tx("Se ruega a los vecinos que, si ven algo raro, miren hacia otro lado.", "Neighbours are asked to look the other way if they see anything odd."),
	tx("El asesino ya ha reservado mesa para la próxima fiesta.", "The killer has already booked a table for the next party."),
	tx("Mañana, la policía hará una declaración que nadie entenderá.", "Tomorrow the police will issue a statement nobody will understand."),
	tx("El asesino, preguntado por este diario, dijo: «Yo solo pasaba por allí».", "The killer, asked by this paper, said: \"I was only passing by\"."),
	tx("Los invitados que quedan juran no volver a una fiesta sin escolta.", "The guests who remain swear never to attend a party without an escort."),
	tx("La investigación continuará... en cuanto acabe el postre.", "The investigation will continue... as soon as dessert is over."),
	tx("Este diario seguirá informando, si el kiosco no cierra antes.", "This paper will keep you informed, if the newsstand does not close first."),
	tx("El culpable podría ser cualquiera. Incluso el lector. Que se ande con ojo.", "The culprit could be anyone. Even the reader. Keep an eye out."),
}

// the witness quote at the foot of the story
var pQuotes = []paperTxt{
	tx("«Yo solo vine por el aperitivo.»", "\"I only came for the appetizers.\""),
	tx("«Se lo juro, estaba en el baño.»", "\"I swear, I was in the bathroom.\""),
	tx("«Yo no vi nada. Y tampoco me pagan.»", "\"I saw nothing. And I am not paid either.\""),
	tx("«La música estaba demasiado alta para gritar.»", "\"The music was too loud to scream.\""),
	tx("«Pensé que era parte del espectáculo.»", "\"I thought it was part of the show.\""),
	tx("«Yo estaba pidiendo otra ronda, agente.»", "\"I was ordering another round, officer.\""),
	tx("«El pato lo vio todo, pero no habla.»", "\"The duck saw everything, but he won't talk.\""),
	tx("«Vi una sombra y me cambié de canapé.»", "\"I saw a shadow and switched canapés.\""),
	tx("«Oí ¡ZAS! y creí que era el karaoke.»", "\"I heard ZAS! and thought it was karaoke.\""),
	tx("«Me dijeron barra libre. No dijeron de qué.»", "\"They said open bar. They did not say open to what.\""),
	tx("«Me escondí tras la tortilla. Funcionó.»", "\"I hid behind the omelette. It worked.\""),
	tx("«Estaba bailando y no miré. Lo siento.»", "\"I was dancing and did not look. Sorry.\""),
	tx("«Era alto, o bajo, o con sombrero. Algo así.»", "\"He was tall, or short, or wearing a hat. Something like that.\""),
	tx("«El DJ puso otra canción y me distraje.»", "\"The DJ put on another song and I got distracted.\""),
	tx("«A mí me invitaron por error. Y sigo aquí.»", "\"I was invited by mistake. And I am still here.\""),
	tx("«Yo vi un guante... o un calcetín. No sé.»", "\"I saw a glove... or a sock. I do not know.\""),
	tx("«El perro llevaba gorro. Yo no, y me miraban a mí.»", "\"The dog wore a hat. I did not, and they stared at me.\""),
	tx("«Pisé la tarta, sí. Pero con mucho cariño.»", "\"I stepped on the cake, yes. But very gently.\""),
	tx("«El DJ roncaba a ritmo. Mejor que la música.»", "\"The DJ snored in time. Better than the music.\""),
}

func init() {
	add := func(k, v string) {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, pool := range paperAllPools() {
		for _, p := range pool {
			add(p.es, p.en)
		}
	}
}

// paperAllPools: every pool of words (for the translation table and the tests).
func paperAllPools() [][]paperTxt {
	return [][]paperTxt{pHeadNone, pHeadClean, pHeadBlast, pHeadPoison, pHeadTrap, pHeadFew, pHeadMany, pHeadCops,
		pHeadAnimalsMany, pHeadAnimalsFew, pHeadMixed, pOpenKnife, pOpenPoison, pOpenTrap, pOpenBlast, pOpenMixed, pOpenNone, pOpenFew, pOpenManyK,
		pSeen0, pSeenFew, pSeenMany, pCuffed, pJailed, pCamera, pSlasher, pMilitary, pSWAT, pRangers, pFBI, pPolice,
		pNoPolice, pMedic, pChain, pTwo, pChallenge, pAnimalsDead, pAnimalsMany, pAnimalsAlive, pIsland, pDJ, pFast,
		pSlow, pLevelEarly, pLevelLate, pExtra, pClose, pQuotes}
}

// formatCount: how many %d a headline template takes.
func formatCount(s string) int { return strings.Count(s, "%d") }
