package main

// Generated lines for the Monkey Island cast (monkey.go). Short catchphrases and original jokes in the same spirit.

var monkeyCatch = map[string][]string{
	"guybrush":          {"¡Soy Guybrush Threepwood, y quiero ser pirata!", "Soy un pirata.", "Mmm... no llego.", "Mmm... no.", "Mmm... no puedo.", "No puedo usar eso con eso.", "Aguanto la respiración diez minutos.", "¡Peleas como un granjero!", "¿Un pollo de goma con una polea?", "¡Mira detrás de ti! ¡Un mono de tres cabezas!", "Esto no se abre. Qué raro.", "¿Alguien tiene un pirata de repuesto?", "Tengo una idea. Es mala, pero es una idea."},
	"stan_barcos":       {"¡Un barco de segunda mano! ¡Casi nuevo!", "¡Cuenta conmigo, amigo!", "Este chollo no se repite. ¡Compra ya!", "¡Solo lo usó una abuelita pirata!", "Te hago un precio de locura.", "¿Qué hace falta para que te lleves este barco?", "¡Lo tengo en tu talla!", "¡Ven, siéntate! Hablemos de barcos.", "Garantía de tres días... o menos."},
	"elaine_marley":     {"Guybrush Threepwood, ¿otra vez tú?", "Soy la gobernadora, ¿recuerdas?", "No me hables de LeChuck.", "Guybrush, ¡eres un desastre!", "Gobernadora Marley, a su servicio.", "Dime que no has roto nada.", "Basta de piratas por hoy.", "¡Esta isla necesita orden!"},
	"lechuck":           {"¡Argg!", "¡Maldito seas, Guybrush!", "¡Guybrush Threepwood!", "¡Elaine será mía!", "¡Ja, ja, ja, ja!", "Soy un fantasma, ¿qué esperabas?", "¡Te voy a despedazar! Luego, si eso.", "¡El vudú es mi aliado!", "¡Mi barco es el más aterrador!", "¡Todos pagaréis por esto!"},
	"voodoo_lady":       {"Ah, ¡el muchacho!", "Cuidado con el vudú.", "Pronto lo entenderás.", "¿Quieres que te lea el futuro?", "Mmm... veo piratas. Muchos.", "¡Cuidado con LeChuck!", "Necesitas un amuleto... o dos.", "Todo tiene su precio."},
	"meathook":          {"¡Mi loro se llama Marcel!", "¡Los ganchos no se lavan solos!", "Yo tenía manos... creo.", "¡Arr! Cuidado con mis ganchos.", "Rascarse con ganchos duele.", "¡Aplaudir es un reto!"},
	"pirata_importante": {"Somos piratas importantes.", "Hablas con un pirata importante.", "¡Arr! ¿Dónde está el grog?", "El grog se acaba. Casi nunca.", "Los piratas de verdad no hacen cola."},
}

var murrayLines = []string{
	"¡Soy Murray, el demonio más temido!",
	"¡Arrodillaos ante mí, mortales!",
	"¡Tembladme! ¡Soy invencible!",
	"¡Me clavaron en una estaca! ¡Qué insulto!",
	"¡Sacadme de aquí, plebeyos!",
	"¡Os maldigo a todos! ¡A todos!",
	"¡Cuando me libere, os vais a enterar!",
	"¡Soy el amo del mundo! ¿Hola?",
	"¿Alguien me escucha? ¡Soy Murray!",
	"¡Ignorarme es un error fatal!",
	"¡Conquistaré el Caribe! Mañana.",
	"¡Mi poder es infinito! Mis brazos, no.",
	"No tengo cuerpo, pero tengo ideas.",
	"¡Soy un demonio muy peligroso!",
	"¡Cuéntame un chiste, mortal!",
	"Estar clavado es muy incómodo.",
	"¡Una calavera tiene derechos!",
	"¡Guybrush! ¡Te odio, Guybrush!",
	"Dadme una pierna y os destruyo.",
	"¡Mi grito helará vuestra sangre!",
	"¡Cuidado, que muerdo!",
	"¿Tenéis pizza? Tengo hambre de almas.",
	"No me ignoréis, ¡soy el amo!",
	"¡Venid a mí, mis fieles!... ¿Fieles?",
	"¡Ya verán cuando me crezca el cuerpo!",
	"Un demonio también necesita compañía.",
	"¡Qué fiesta más aburrida! ¡Maldita sea!",
	"¡Moriréis todos! Algún día. Seguro.",
	"¡Soy más temible que un loro!",
	"Soy Murray. Mi nombre os hará llorar.",
}

var monkeyDuets = []duetDef{
	{"guybrush", "lechuck", []duetStep{ds(1, "¡Maldito seas, Guybrush!", AnimPoint), ds(0, "Hola, LeChuck. ¿Cuánto tiempo?", AnimWave), ds(1, "¡Argg! ¡Te voy a destruir!", AnimPunch, 0xff4020), ds(0, "¿Después de la fiesta, vale?", AnimTalk)}},
	{"guybrush", "elaine_marley", []duetStep{ds(1, "Guybrush, ¿qué has roto ahora?", AnimTalk), ds(0, "Nada. Bueno, casi nada.", AnimPanic), ds(1, "Soy la gobernadora, ¿recuerdas?", AnimPoint), ds(0, "¡Eres la mejor!", AnimLove)}},
	{"stan_barcos", "guybrush", []duetStep{ds(0, "¡Amigo! ¿Buscas un barco?", AnimWave), ds(1, "Yo... mmm... no puedo.", AnimTalk), ds(0, "¡Te lo dejo a precio de risa!", AnimLaugh), ds(1, "Soy un pirata. Solo miro.", AnimTalk)}},
	{"elaine_marley", "lechuck", []duetStep{ds(1, "¡Elaine será mía!", AnimPoint), ds(0, "Ni en tus sueños, LeChuck.", AnimPunch), ds(1, "¡Argg!", AnimPanic)}},
	{"voodoo_lady", "guybrush", []duetStep{ds(0, "Ah, ¡el muchacho!", AnimWave), ds(1, "Necesito consejo...", AnimTalk), ds(0, "Todo tiene su precio.", AnimPoint), ds(1, "Mmm... no llevo dinero.", AnimPanic)}},
	{"meathook", "guybrush", []duetStep{ds(0, "¡Mi loro se llama Marcel!", AnimWave), ds(1, "Qué... bonito nombre.", AnimTalk), ds(0, "¡Choca esos... ganchos!", AnimLaugh)}},
}

var monkeyNames = map[string]string{
	"guybrush":          "Guybrush Threepwood",
	"stan_barcos":       "Stan, vendedor de barcos",
	"elaine_marley":     "Gobernadora Marley",
	"lechuck":           "LeChuck",
	"voodoo_lady":       "La Dama Vudú",
	"meathook":          "Gancho",
	"pirata_importante": "Pirata Importante",
}

var monkeyDesc = map[string]string{
	"skin:guybrush":          "Un aspirante a pirata con coleta rubia que puede aguantar la respiración diez minutos. Y poco más.",
	"skin:stan_barcos":       "Vendedor de barcos de segunda mano con chaqueta a cuadros y brazos que no paran. Te venderá hasta tu propio barco.",
	"skin:elaine_marley":     "La gobernadora de la isla. Tiene mucha paciencia... y se le está acabando.",
	"skin:lechuck":           "El pirata fantasma verde de barba negra. Aterrador y rencoroso, pero llega siempre tarde.",
	"skin:voodoo_lady":       "Una señora muy misteriosa que lee el futuro. Nunca da un consejo gratis.",
	"skin:meathook":          "Un pirata con ganchos en lugar de manos y un loro llamado Marcel. Aplaudir se le da fatal.",
	"skin:pirata_importante": "Uno de esos piratas muy importantes del bar. Nadie sabe qué hacen, pero son importantes.",
}
