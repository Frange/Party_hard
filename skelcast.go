package main

// ---------------------------------------------------------------------------
// The skeleton crowd: six bony guests who turn up on their own like the other
// famous faces (once per party), can be bought in the shop and have their own
// catchphrases and a few duets. Names are tributes, not trademarks.
// ---------------------------------------------------------------------------

var castSkel = []presetEntry{
	def("jack_rey_calabaza", maskc(0xf0f0e4, MaskSkull), skinc(0xe8e4d2), tops(TopSuit, 0x14141a, 0xf4f4f4), patn(2), accentc(0xd02828), pantsc(0x14141a), shoesc(0x14141a), bodyc(BodySlim), headc(HeadBig), stat(2)),
	def("la_calaca", fem(), maskc(0xf6f0e0, MaskSkull), skinc(0xe8e4d2), hairc(0x14141a, HairLong), tops(TopDress, 0xc83a8a, 0xf0c030), patn(3), shoesc(0xf0c030), bodyc(BodySlim), stat(1)),
	def("capitan_calavera", maskc(0xe8e4d4, MaskSkull), skinc(0xe8e4d2), hatc(HatPirate, 0x14141a), tops(TopLong, 0x8a1a1a, 0xe8e0c8), patn(2), pantsc(0x2a1a1a), shoesc(0x14141a), capec(0x3a1a1a), carryc(HoldSaber), bodyc(BodyBig), stat(-1)),
	def("esqueleto_rockero", maskc(0xf0f0e0, MaskSkull), skinc(0xe8e4d2), hairc(0x14141a, HairSpiky), glass(GlassShades), tops(TopVest, 0x14141a, 0x14141a), pantsc(0x2a2a34), shoesc(0x14141a), carryc(HoldGuitar), bodyc(BodySlim)),
	def("esqueleto_bailarin", maskc(0xf4f4e8, MaskSkull), skinc(0xe8e4d2), hatc(HatTopHat, 0xe8c040), tops(TopSuit, 0xf0c030, 0xe83a8a), patn(3), pantsc(0xf0c030), shoesc(0xe83a8a), bodyc(BodySlim), stat(1)),
	def("esqueleto_ejecutivo", maskc(0xe8e8dc, MaskSkull), skinc(0xe8e4d2), tops(TopSuit, 0x2a2a34, 0xf4f4f8), accentc(0xa01818), pantsc(0x2a2a34), shoesc(0x14141a), carryc(HoldBriefcase), bodyc(BodyNormal)),
}

var skelSlugs = []string{"jack_rey_calabaza", "la_calaca", "capitan_calavera", "esqueleto_rockero", "esqueleto_bailarin", "esqueleto_ejecutivo"}

var skelQuirk = map[string]string{
	"jack_rey_calabaza": "sk_jack", "la_calaca": "sk_calaca", "capitan_calavera": "sk_capitan",
	"esqueleto_rockero": "sk_rockero", "esqueleto_bailarin": "sk_bailarin", "esqueleto_ejecutivo": "sk_ejecutivo",
}

var skelCatch = map[string][]string{
	"jack_rey_calabaza":   {"¡Soy el Rey de Halloween!", "Este año me disfrazo de Navidad.", "Las calabazas me adoran.", "¡Qué pesadilla de fiesta!", "No tengo corazón, pero sí ritmo.", "¿Alguien tiene hilo y aguja?", "Todo me da miedo menos yo."},
	"la_calaca":           {"¡Que viva el Día de Muertos!", "Traigo flores y mala idea.", "Estoy en los huesos, ¿y tú?", "¡Baila conmigo, mortal!", "Me caigo de risa. Literalmente.", "¿Una calaverita de azúcar?", "Hasta la muerte... ¡y después!"},
	"capitan_calavera":    {"¡Arr! No tengo ni ron ni carne.", "¡Todos a bordo de la fiesta!", "Mi barco se hundió. Yo no.", "¡Navegamos con huesos y valor!", "Tengo hueso para rato.", "¿Dónde dejé el tesoro? Ah, no tengo bolsillos.", "¡Yo nunca pierdo la cabeza!"},
	"esqueleto_rockero":   {"¡Rock hasta los huesos!", "Mi solo es mortal.", "Me gusta el metal. Y el calcio.", "¡Esta va por los muertos!", "Toco con los dedos... de hueso.", "¡Más volumen, que no oigo!", "No tengo oídos. ¡Más alto!"},
	"esqueleto_bailarin":  {"¡Hoy muevo el esqueleto de verdad!", "Bailo hasta desmontarme.", "Siento el ritmo en la médula.", "¡Esta pista es mía!", "No tengo caderas, ¡pero las muevo!", "¡Música! ¡Se me enfrían los huesos!", "Me falta carne, no ritmo."},
	"esqueleto_ejecutivo": {"Tengo una reunión... eterna.", "Trabajo hasta los huesos.", "Mi jefe me dejó en los huesos.", "¿Podemos hacerlo por email?", "Estoy en los huesos. Presupuesto.", "Necesito vacaciones. De la tumba.", "Mi cuenta está en números muertos."},
}

var skelNames = map[string]string{
	"jack_rey_calabaza":   "Jack, Rey Calabaza",
	"la_calaca":           "La Calaca",
	"capitan_calavera":    "Capitán Calavera",
	"esqueleto_rockero":   "Esqueleto Rockero",
	"esqueleto_bailarin":  "Esqueleto Bailarín",
	"esqueleto_ejecutivo": "Esqueleto Ejecutivo",
}

var skelDesc = map[string]string{
	"skin:jack_rey_calabaza":   "Un esqueleto alto con traje de rayas que gobierna Halloween. Dice que el año que viene se encarga de la Navidad.",
	"skin:la_calaca":           "Una esqueleto elegante de Día de Muertos con vestido de flores. Siempre llega primera a la fiesta... y se queda.",
	"skin:capitan_calavera":    "Un pirata en los huesos con sable y tricornio. Perdió el barco, el ron y el cuerpo, pero conserva el estilo.",
	"skin:esqueleto_rockero":   "Rockero de huesos y gafas de sol que toca la guitarra a todo volumen. Que no oiga nada es culpa de las orejas que no tiene.",
	"skin:esqueleto_bailarin":  "Un esqueleto con traje disco dorado que baila hasta desmontarse. Sus huesos suenan al ritmo del bajo.",
	"skin:esqueleto_ejecutivo": "Un oficinista en los huesos con maletín y corbata. Lleva siglos esperando una reunión que acabe.",
}

var skelDuets = []duetDef{
	{"jack_rey_calabaza", "la_calaca", []duetStep{ds(0, "¿Bailamos? Estoy en los huesos.", AnimTalk), ds(1, "Pensaba que no me lo pedirías.", AnimLove), ds(0, "Cuidado, que me desmonto.", AnimLaugh), ds(1, "¡Qué ganas de tus huesos!", AnimDance4)}},
	{"esqueleto_rockero", "esqueleto_bailarin", []duetStep{ds(0, "¡Un solo para ti!", AnimWave), ds(1, "¡Toca más fuerte!", AnimDance6), ds(0, "Voy a perder un dedo.", AnimPanic), ds(1, "Tranquilo, tienes más.", AnimLaugh)}},
	{"esqueleto_ejecutivo", "capitan_calavera", []duetStep{ds(0, "¿Tiene cita para el tesoro?", AnimTalk), ds(1, "¡Arr! Mi cita fue en 1700.", AnimPoint), ds(0, "Le agendo para el lunes.", AnimTalk), ds(1, "¿Y el ron lo pagas tú?", AnimLaugh)}},
}

func init() {
	famousPresets = append(famousPresets, asFuncs(castSkel)...)
	for k, v := range skelNames {
		displayNameOverrides[k] = v
	}
	for k, v := range skelDesc {
		funnyDesc[k] = v
	}
	for k, v := range skelCatch {
		catchphrases[k] = v
		quirkLines[skelQuirk[k]] = v
	}
	quirkAnims["sk_jack"] = []int{AnimTalk, AnimPoint, AnimLaugh, AnimTalk}
	quirkAnims["sk_calaca"] = []int{AnimDance4, AnimTalk, AnimLove, AnimDance2}
	quirkAnims["sk_capitan"] = []int{AnimLaugh, AnimPoint, AnimTalk}
	quirkAnims["sk_rockero"] = []int{AnimGuitar, AnimDance0, AnimGuitar, AnimDance6}
	quirkAnims["sk_bailarin"] = []int{AnimDance3, AnimPirouette, AnimDance7, AnimDance9}
	quirkAnims["sk_ejecutivo"] = []int{AnimTalk, AnimCall, AnimTalk}
	duetDefs = append(duetDefs, skelDuets...)
	specialPool = append(specialPool, skelSlugs...)
	starGroups = append(starGroups, []string{"esqueleto_rockero", "esqueleto_bailarin", "la_calaca"})
}
