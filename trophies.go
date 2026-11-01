package main

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// Trophies: 49 goals. Each one unlocks something that cannot be bought in the
// shop (it shows up there, locked, until the trophy is won).
// ---------------------------------------------------------------------------

type trophyReward struct {
	tab      int // shop list the reward lives in
	id, name string
	slot     string // wearable slot, if any
	fx       func(*CharSpec)
	arg      string // preset slug (characters, killers)
	hold     int    // weapon
	desc     string
}

type Trophy struct {
	id, name, how string
	stat          string
	need          int
	reward        trophyReward
}

func wr(tab int, id, name, slot string, fx func(*CharSpec), desc string) trophyReward {
	return trophyReward{tab: tab, id: id, name: name, slot: slot, fx: fx, desc: desc}
}
func chr(slug, name, desc string) trophyReward {
	return trophyReward{tab: shopChars, id: "skin:" + slug, name: name, arg: slug, desc: desc}
}
func kil(slug, name, desc string) trophyReward {
	return trophyReward{tab: shopKillers, id: "killer:" + slug, name: name, arg: slug, desc: desc}
}
func wep(id, name string, hold int, desc string) trophyReward {
	return trophyReward{tab: shopWeapons, id: "kw:" + id, name: name, slot: "kweapon", hold: hold, desc: desc}
}
func evr(id, name, desc string) trophyReward {
	return trophyReward{tab: -1, id: id, name: name, desc: desc}
}
func uniR(slot, id, name string, fx func(*CharSpec), desc string) trophyReward {
	return trophyReward{tab: shopUniforms, id: id, name: name, slot: slot, fx: fx, desc: desc}
}

var trophyList = []Trophy{
	{"t_first", "Primera sangre", "Mata a una persona.", "kills", 1, wr(shopAccess, "hat:tr_novato", "Gorro de novato", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatBeanie, hx(0xa01818) }, "Un gorro rojo para quien acaba de estrenarse. Todavía huele a nuevo.")},
	{"t_k25", "Cuarto de siglo", "Mata a 25 personas en total.", "kills", 25, wr(shopOutfits, "out:tr_25", "Camiseta '25 y contando'", "outfit", func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2, s.Pattern = TopTee, hx(0x14141a), hx(0xc02a2a), 3 }, "Presume de cifras con estilo.")},
	{"t_k100", "Centenario", "Mata a 100 personas en total.", "kills", 100, wr(shopAccess, "cape:tr_100", "Capa del cazador", "cape", func(s *CharSpec) { s.Cape = hx(0x8a1a1a) }, "Granate oscuro, como los recuerdos de la fiesta.")},
	{"t_k500", "Carnicero", "Mata a 500 personas en total.", "kills", 500, wr(shopAccess, "mask:tr_500", "Máscara del verdugo", "mask", func(s *CharSpec) { s.Mask, s.MaskKind = hx(0x8a2020), MaskHockey }, "Una máscara roja que te hace parecer más simpático.")},
	{"t_k2000", "Leyenda negra", "Mata a 2000 personas en total.", "kills", 2000, chr("tr_verdugo", "El Verdugo", "Capucha, hacha y mucho aguante. Llega a la fiesta con su propia horca.")},
	{"t_w1", "Fiesta terminada", "Completa 1 fiesta.", "wins", 1, wr(shopAccess, "hat:tr_papel", "Corona de papel", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatPartyCone, hx(0xe8c040) }, "Para el rey de una fiesta que ya no tiene invitados.")},
	{"t_w5", "Fiestero", "Completa 5 fiestas.", "wins", 5, wr(shopOutfits, "out:tr_smoking", "Smoking de anfitrión", "outfit", func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants, s.Accent = TopSuit, hx(0x0e0e12), hx(0xf4f4f8), hx(0x0e0e12), hx(0xd02828)
	}, "Elegante para recibir a quien llega... y despedirlo.")},
	{"t_w15", "Anfitrión", "Completa 15 fiestas.", "wins", 15, chr("tr_rey_fiesta", "El Rey de la Fiesta", "Corona dorada, capa roja y una copa siempre llena. Dice que la fiesta es suya.")},
	{"t_w40", "Veterano", "Completa 40 fiestas.", "wins", 40, wep("katana", "Katana de leyenda", HoldMachete, "Una hoja con historia: ha cortado más fiestas que cebollas.")},
	{"t_w100", "Imbatible", "Completa 100 fiestas.", "wins", 100, wr(shopOutfits, "out:tr_campeon", "Armadura de campeón", "outfit", func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants = TopArmor, hx(0xe8c040), hx(0xe8c040), hx(0x8a6a20)
	}, "Dorada, pesada y totalmente innecesaria. Perfecta.")},
	{"t_nopolice", "Sin testigos", "Completa una fiesta sin que nadie llame a la policía.", "nopolice", 1, wr(shopAccess, "hat:tr_seda", "Capucha de seda", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatNinja, hx(0x2a2a3a) }, "Tan silenciosa que ni tú te oyes.")},
	{"t_fast", "Velocista", "Completa una fiesta en menos de 3 minutos.", "fast", 1, wr(shopAccess, "shoe:tr_turbo", "Zapatillas turbo", "shoes", func(s *CharSpec) { s.Shoes = hx(0xe8e8f0) }, "Con alas pintadas a los lados. Aún no hacen que vueles.")},
	{"t_long", "Maratón", "Completa una fiesta tras más de 10 minutos.", "long", 1, wep("honor", "Cetro de honor", HoldTrophy, "Para quien tiene paciencia. Se sostiene con las dos manos.")},
	{"t_s4", "Cuatro estrellas", "Llega a 4 estrellas de criminalidad.", "stars", 4, uniR("cop", "cop:tr_gala", "Local · uniforme de gala", func(s *CharSpec) {
		s.Shirt, s.Pants, s.HatCol, s.Accent = hx(0xf0f0f4), hx(0x1a2a5a), hx(0xf0f0f4), hx(0xe8c040)
	}, "Guantes blancos y botones dorados. Detienen igual de mal, pero con clase.")},
	{"t_s5", "Cinco estrellas", "Llega a 5 estrellas de criminalidad.", "stars", 5, uniR("swat", "swat:tr_plata", "SWAT · élite plateado", func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0xc8ccd8), hx(0xc8ccd8), hx(0x6a6e78), hx(0xc8ccd8), hx(0xc8ccd8)
	}, "El mejor uniforme para sorprender a la Interpol.")},
	{"t_ki1", "Cazador de asesinos", "Derrota a un asesino especial.", "killers", 1, kil("tr_k_sombra", "La Sombra", "Un asesino que solo se ve cuando ya es tarde. Muy discreto.")},
	{"t_ki5", "Matamonstruos", "Derrota a 5 asesinos especiales.", "killers", 5, wep("guadana", "Guadaña", HoldMachete, "Una hoja curva de la que huyen hasta los asesinos más duros.")},
	{"t_ki15", "Exterminador", "Derrota a 15 asesinos especiales.", "killers", 15, kil("tr_k_rey_calavera", "El Rey Calavera", "Corona, capa y un reino de huesos. Los demás asesinos le hacen reverencias.")},
	{"t_d20", "Pies ligeros", "Baila 20 veces.", "dance", 20, wr(shopOutfits, "out:tr_disco", "Traje disco brillante", "outfit", func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants, s.Pattern = TopSuit, hx(0xf0c030), hx(0xe83a8a), hx(0xf0c030), 3
	}, "Brilla más que la bola de la discoteca.")},
	{"t_d100", "Rey de la pista", "Baila 100 veces.", "dance", 100, chr("tr_bailarina_disco", "La Bailarina de Discoteca", "Lentejuelas, botas altas y un sentido del ritmo capaz de despertar a un muerto.")},
	{"t_h10", "Escondido", "Escóndete 10 veces.", "hide", 10, wr(shopAccess, "hat:tr_caja", "Caja de cartón sospechosa", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatHardhat, hx(0x8a6a3a) }, "Se supone que nadie sospecha de una caja que camina.")},
	{"t_p5", "Envenenador", "Mata a 5 personas con veneno.", "poison", 5, wr(shopAccess, "mask:tr_gas", "Máscara de gas", "mask", func(s *CharSpec) { s.Mask, s.MaskKind = hx(0x5a7a4a), MaskPlain }, "Para no respirar lo que sirves en el ponche.")},
	{"t_tr5", "Artificiero", "Mata a 5 personas con trampas.", "traps", 5, wr(shopOutfits, "out:tr_mono", "Mono de artificiero", "outfit", func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants = TopLong, hx(0xf08a20), hx(0xf08a20), hx(0xf08a20)
	}, "Naranja intenso, para que te vean (y salgan corriendo).")},
	{"t_a10", "Cazador", "Mata a 10 animales.", "animals", 10, wr(shopAccess, "hat:tr_safari", "Sombrero de safari", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatCowboy, hx(0xc8b070) }, "Huele a pólvora, a selva y a dignidad perdida.")},
	{"t_a50", "Gran cazador", "Mata a 50 animales.", "animals", 50, chr("tr_cazador_safari", "El Gran Cazador", "Un cazador de safari que se ha perdido hasta la fiesta. Ve animales hasta en los bocadillos.")},
	{"t_g10", "Mano dura", "Mata a 10 guardias.", "guards", 10, uniR("army", "army:tr_gala", "Ejército · de gala", func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0x2a3a22), hx(0xe8c040), hx(0x1a2a14), hx(0x2a3a22), hx(0xe8c040)
	}, "Medallas por cada uniforme que no se mancha.")},
	{"t_e5", "Testigo ocular", "Presencia 5 sucesos especiales.", "events", 5, wr(shopAccess, "haircol:tr_arco", "Pelo arcoíris", "haircol", func(s *CharSpec) { s.Hair = hx(0xe83aa8) }, "Colores para quien lo ha visto todo.")},
	{"t_ufo", "Contactos", "Presencia el paso de un ovni.", "ev_1", 1, chr("tr_marciano", "El Marciano", "Un alienígena verde con corbata. Viene en misión de paz... y a por las patatas.")},
	{"t_rex", "Era jurásica", "Presencia la llegada del T-Rex.", "ev_16", 1, chr("tr_paleontologo", "La Paleontóloga", "Con casco y pico. Sabe lo que es un dinosaurio y lo demuestra gritando.")},
	{"t_ghost", "¿A quién llamarás?", "Presencia la llegada de los Cazafantasmas.", "ev_15", 1, wep("protones", "Aspiradora de protones", HoldVacuum, "Una mochila con un tubo. No atrapa fantasmas, pero deja el suelo limpio.")},
	{"t_m10k", "Primeros ahorros", "Gana 10000 puntos en total.", "earned", 10000, wr(shopOutfits, "out:tr_rapero", "Chaqueta de rapero dorada", "outfit", func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants = TopHoodie, hx(0xe8c040), hx(0xe8c040), hx(0x1a1a22)
	}, "Con cadena imaginaria y muchas ínfulas.")},
	{"t_m100k", "Multimillonario", "Gana 100000 puntos en total.", "earned", 100000, chr("tr_multimillonario", "El Multimillonario", "Esmoquin, puro y un fajo de billetes que no es suyo. Le gusta pagar... por todo.")},
	{"t_b5", "Cliente habitual", "Haz 5 compras en la tienda.", "bought", 5, wr(shopAccess, "hat:tr_copa", "Chistera dorada", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatTopHat, hx(0xe8c040) }, "Una prueba de que en esta tienda se gasta con gusto.")},
	{"t_b30", "Compulsivo", "Haz 30 compras en la tienda.", "bought", 30, kil("tr_k_vendedor", "El Vendedor de Seguros", "Con traje barato y una sonrisa perfecta. No te dejará salir sin contratar algo.")},
	{"t_p10", "Aficionado", "Juega 10 fiestas.", "played", 10, wr(shopAccess, "shoe:tr_calle", "Zapatillas de callejero", "shoes", func(s *CharSpec) { s.Shoes = hx(0x8a5a2a) }, "Muy gastadas, pero ahora son tuyas.")},
	{"t_dead10", "Mala suerte", "Muere 10 veces.", "deaths", 10, wr(shopOutfits, "out:tr_fantasma", "Sábana de fantasma", "outfit", func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants = TopRobe, hx(0xf4f4f8), hx(0xf4f4f8), hx(0xf4f4f8)
	}, "Para que se te vea tan blanco como te sientes.")},
	{"t_world8", "Viajero", "Completa 8 fiestas distintas.", "distinct", 8, wr(shopAccess, "cape:tr_viajero", "Capa de viajero", "cape", func(s *CharSpec) { s.Cape = hx(0x2a6a8a) }, "Con polvo de ocho lugares distintos.")},
	{"t_world15", "Trotamundos", "Completa las 15 fiestas de la campaña.", "distinct", 15, uniR("rangers", "rangers:tr_oro", "Rangers · platino", func(s *CharSpec) { s.Shirt, s.Shirt2, s.HatCol = hx(0xe8e8f4), hx(0xe8e8f4), hx(0xe8e8f4) }, "Reservado a quien ha visto todo el planeta de fiestas.")},
	{"t_pts3k", "Buena noche", "Consigue 3000 puntos en una fiesta.", "best", 3000, chr("tr_rockero_leyenda", "El Rockero Legendario", "Cuero, guitarra y una melena de años de fiestas. Su solo no se puede parar.")},
	{"t_pts8k", "Noche de récord", "Consigue 8000 puntos en una fiesta.", "best", 8000, kil("tr_k_dios_oscuro", "El Dios Oscuro", "Un ser de otra dimensión que se aburre en su trono y viene a divertirse a tu costa.")},
	{"t_sam", "Cazapatinetes", "Mata a Sam-sam 5 veces (se escapa rápido: ve por la espalda).", "kill_sam_sam", 5, evr("ev:scooters", "Suceso: Desfile de patinetes", "Una fila de niños en patinete cruza la fiesta a toda velocidad y la gente se aparta. Sale de vez en cuando.")},
	{"t_cesar", "Cazatravesuras", "Mata a César 5 veces.", "kill_cesar", 5, evr("ev:prank", "Suceso: Lluvia de globos de agua", "Globos de agua caen sobre la fiesta y empapan a quien pillen debajo. Sale de vez en cuando.")},
	{"t_diego", "Cazaskaters", "Mata a Diego 5 veces.", "kill_diego", 5, evr("ev:skate", "Suceso: Exhibición de skate", "Una pandilla de skaters cruza la fiesta haciendo saltos. Sale de vez en cuando.")},
	{"t_pig", "Granjero furioso", "Mata a 15 cerdos.", "an_pig", 15, wr(shopAccess, "hat:tr_cerdo", "Gorro de cerdo", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatPig, hx(0xf0a0b4) }, "Orejas caídas y morro rosa. Dicen que te hace oler a beicon.")},
	{"t_cat", "Séptima vida", "Mata a 20 gatos.", "an_cat", 20, wr(shopAccess, "hat:tr_gato", "Gorro de gato", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatCat, hx(0x7a7a86) }, "Con orejas puntiagudas. Te cae bien todo el mundo salvo los ratones.")},
	{"t_dog", "Enemigo del perro", "Mata a 30 perros.", "an_dog", 30, wr(shopAccess, "hat:tr_perro", "Gorro de perro", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatDog, hx(0xa8703a) }, "Orejas largas y hocico negro. Mueve la cola solo en tu imaginación.")},
	{"t_bird", "Cazapájaros", "Mata a 40 pájaros, gallinas o patos.", "an_bird", 40, wr(shopAccess, "hat:tr_pajaro", "Gorro de pájaro", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatBird, hx(0x60c0e8) }, "Cresta de plumas de colores y pico naranja. Pía sin que lo notes.")},
	{"t_cut10", "Cortacables", "Corta el cable de 10 teléfonos.", "cuts", 10, wr(shopOutfits, "out:tr_tecnico", "Mono de técnico", "outfit", func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants = TopLong, hx(0xf0c820), hx(0xf08a20), hx(0x2a3a5a)
	}, "Amarillo y con muchos bolsillos. Con él nadie te pregunta qué haces junto al teléfono.")},
	{"t_tech3", "Servicio técnico", "Consigue que los técnicos arreglen 3 teléfonos que tú cortaste.", "techs", 3, wr(shopAccess, "hat:tr_casco_tec", "Casco de técnico", "hat", func(s *CharSpec) { s.Hat, s.HatCol = HatHardhat, hx(0xf0c820) }, "Un casco amarillo de la furgoneta. Todavía huele a cable quemado.")},
}

// reward presets
func init() {
	for _, e := range []presetEntry{
		def("tr_verdugo", maskc(0x14141a, MaskPlain), tops(TopVest, 0x2a2a30, 0x2a2a30), pantsc(0x14141a), shoesc(0x14141a), bodyc(BodyGiant), stat(-3), carryc(HoldMachete)),
		def("tr_rey_fiesta", hatc(HatCrown, 0xe8c040), beardc(BeardFull), hairc(0xd8d0c0, HairLong), tops(TopRobe, 0xa01830, 0xe8c040), capec(0xa01830), carryc(HoldBeer), bodyc(BodyFat)),
		def("tr_bailarina_disco", fem(), hairc(0xe83a8a, HairBeehive), tops(TopDress, 0xf0c030, 0xe83a8a), patn(3), shoesc(0xf0c030), bodyc(BodySlim), stat(1)),
		def("tr_cazador_safari", hatc(HatCowboy, 0xc8b070), beardc(BeardMustache), tops(TopVest, 0x8a7a4a, 0xe8d8a0), pantsc(0x6a5a3a), shoesc(0x4a3a2a), bodyc(BodyBig)),
		def("tr_marciano", skinc(0x60c040), hairs(HairBald), hatc(HatAntenna, 0x60c040), glass(GlassBig), tops(TopSuit, 0x2a2a3a, 0xf4f4f8), accentc(0xe83a8a), pantsc(0x2a2a3a), headc(HeadBig), stat(1)),
		def("tr_paleontologo", fem(), hatc(HatHardhat, 0xe8c830), hairc(0x8a5a2a, HairPony), glass(GlassRound), tops(TopVest, 0xc8b070, 0xe8d8a0), pantsc(0x6a5a3a), shoesc(0x4a3a2a)),
		def("tr_multimillonario", hairc(0x3a3a3a, HairComb), beardc(BeardMustache), hatc(HatTopHat, 0x14141a), tops(TopSuit, 0x14141a, 0xf4f4f8), accentc(0xe8c040), pantsc(0x14141a), shoesc(0xe8c040), carryc(HoldBriefcase), bodyc(BodyFat)),
		def("tr_rockero_leyenda", hairc(0x3a2a1a, HairLong), beardc(BeardStubble), glass(GlassShades), tops(TopVest, 0x14141a, 0x14141a), pantsc(0x14141a), shoesc(0x14141a), carryc(HoldGuitar), bodyc(BodyBig)),
	} {
		starPresets = append(starPresets, e)
		shopOnly[e.Name] = true
	}
	for _, e := range []presetEntry{
		def("tr_k_sombra", maskc(0x0e0e12, MaskPlain), tops(TopRobe, 0x0e0e12, 0x0e0e12), weaponc(HoldKnife), stat(-2)),
		def("tr_k_rey_calavera", maskc(0xf0f0e0, MaskSkull), hatc(HatCrown, 0xe8c040), tops(TopRobe, 0x2a1a3a, 0xe8c040), capec(0x6a1030), weaponc(HoldMachete), bodyc(BodyGiant), stat(-3)),
		def("tr_k_vendedor", hairc(0x3a3a3a, HairComb), tops(TopSuit, 0x6a6e78, 0xf4f4f8), accentc(0x2a4ab8), pantsc(0x6a6e78), weaponc(HoldBriefcase), maskc(0xe8d8c0, MaskPlain)),
		def("tr_k_dios_oscuro", maskc(0x1a0a2a, MaskSkull), hatc(HatHorns, 0x6a2a8a), tops(TopRobe, 0x2a0a3a, 0x6a2a8a), capec(0x2a0a3a), weaponc(HoldTorch), bodyc(BodyGiant), stat(-4)),
	} {
		killerPresets = append(killerPresets, e)
	}
	for k, v := range map[string]string{
		"tr_k_sombra": "La Sombra", "tr_k_rey_calavera": "El Rey Calavera", "tr_k_vendedor": "El Vendedor de Seguros", "tr_k_dios_oscuro": "El Dios Oscuro",
		"tr_verdugo": "El Verdugo", "tr_rey_fiesta": "El Rey de la Fiesta", "tr_bailarina_disco": "La Bailarina de Discoteca", "tr_cazador_safari": "El Gran Cazador",
		"tr_marciano": "El Marciano", "tr_paleontologo": "La Paleontóloga", "tr_multimillonario": "El Multimillonario", "tr_rockero_leyenda": "El Rockero Legendario",
	} {
		displayNameOverrides[k] = v
	}
}

// registerTrophyRewards puts every reward in its shop list, locked.
func registerTrophyRewards() {
	for i, t := range trophyList {
		r := t.reward
		if r.tab < 0 {
			continue // an event, not a shop item
		}
		it := shopItem{id: r.id, name: r.name, desc: r.desc, price: 1<<20 + i, slot: r.slot, arg: r.arg, trophy: t.id}
		if r.fx != nil {
			wearFx[r.id] = r.fx
		}
		if r.hold != 0 {
			weaponHold[r.id] = r.hold
		}
		funnyDesc[r.id] = r.desc
		shopCatalog[r.tab] = append(shopCatalog[r.tab], it)
	}
	// the last trophy brings a whole golden wardrobe too
	for i, r := range legendExtras() {
		it := shopItem{id: r.id, name: r.name, desc: r.desc, price: 1<<20 + 500 + i, slot: r.slot, arg: r.arg, trophy: legendTrophy}
		if r.fx != nil {
			wearFx[r.id] = r.fx
		}
		if r.hold != 0 {
			weaponHold[r.id] = r.hold
		}
		funnyDesc[r.id] = r.desc
		shopCatalog[r.tab] = append(shopCatalog[r.tab], it)
	}
}

// ---- stats ----------------------------------------------------------------

func (g *Game) stat(k string, n int) {
	if g.progress.Stats == nil {
		g.progress.Stats = map[string]int{}
	}
	g.progress.Stats[k] += n
	g.trophyDirty = true
}

func (g *Game) statMax(k string, n int) {
	if g.progress.Stats == nil {
		g.progress.Stats = map[string]int{}
	}
	if n > g.progress.Stats[k] {
		g.progress.Stats[k] = n
		g.trophyDirty = true
	}
}

// checkTrophies grants any trophy whose goal has been reached.
func (g *Game) checkTrophies() {
	if !g.trophyDirty {
		return
	}
	g.trophyDirty = false
	for _, t := range trophyList {
		reached := g.progress.Stats[t.stat] >= t.need
		if t.id == legendTrophy {
			reached = g.othersDone()
		}
		if g.progress.Trophies[t.id] || !reached {
			continue
		}
		if g.progress.Trophies == nil {
			g.progress.Trophies = map[string]bool{}
		}
		g.progress.Trophies[t.id] = true
		g.progress.Owned = append(g.progress.Owned, t.reward.id)
		if t.id == legendTrophy {
			for _, r := range legendExtras() {
				g.progress.Owned = append(g.progress.Owned, r.id)
			}
		}
		g.syncShop()
		g.applyShop()
		g.toasts = append(g.toasts, trophyToast{name: t.name, reward: t.reward.name})
		g.snd.Play("win")
		saveProgress(g.progress)
	}
}

// trophyToast is the pop-up that slides in from the top when a trophy is won.
type trophyToast struct {
	name, reward string
	t            float64
}

const toastLife = 5.0

func (g *Game) updateToasts(dt float64) {
	if len(g.toasts) == 0 {
		return
	}
	g.toasts[0].t += dt
	if g.toasts[0].t > toastLife {
		g.toasts = g.toasts[1:]
	}
}

func (g *Game) drawToast(dst *ebiten.Image) {
	if len(g.toasts) == 0 {
		return
	}
	tt := g.toasts[0]
	const w, h = 400.0, 64.0
	slide := 1.0
	switch {
	case tt.t < 0.45:
		slide = tt.t / 0.45
	case tt.t > toastLife-0.45:
		slide = (toastLife - tt.t) / 0.45
	}
	slide = slide * slide * (3 - 2*slide) // ease in/out
	x := screenW/2 - w/2
	y := -h - 6 + (h+22)*slide
	// dark rounded pill with a soft edge
	fillRect(dst, x-2, y-2, w+4, h+4, rgba(0xffffff, 40))
	fillRect(dst, x+8, y, w-16, h, rgb(0x14161e))
	fillRect(dst, x, y+8, w, h-16, rgb(0x14161e))
	filledEllipse(dst, x+10, y+10, 10, 10, rgb(0x14161e))
	filledEllipse(dst, x+w-10, y+10, 10, 10, rgb(0x14161e))
	filledEllipse(dst, x+10, y+h-10, 10, 10, rgb(0x14161e))
	filledEllipse(dst, x+w-10, y+h-10, 10, 10, rgb(0x14161e))
	// the gold cup in a circle
	cx, cy := x+36, y+h/2
	filledEllipse(dst, cx, cy, 24, 24, rgb(0x2a2c3a))
	filledEllipse(dst, cx, cy, 20, 20, rgb(0xe8c040))
	cup := rgb(0x6a4a10)
	fillRect(dst, cx-9, cy-11, 18, 10, cup)
	filledEllipse(dst, cx, cy-1, 9, 8, cup)
	fillRect(dst, cx-2, cy+5, 4, 5, cup)
	fillRect(dst, cx-7, cy+9, 14, 3, cup)
	fillRect(dst, cx-13, cy-10, 4, 6, cup)
	fillRect(dst, cx+9, cy-10, 4, 6, cup)
	g.drawText(dst, "¡Trofeo desbloqueado!", x+72, y+9, rgb(0x9aa0b4), 1)
	g.drawText(dst, tt.name, x+72, y+24, rgb(0xf4f6ff), 1.6)
	g.drawText(dst, "Premio: "+tt.reward, x+72, y+46, rgb(0xe8c040), 1)
}

func (g *Game) trophyDone(t Trophy) bool { return g.progress.Trophies[t.id] }

func (g *Game) trophyCount() int {
	n := 0
	for _, t := range trophyList {
		if g.trophyDone(t) {
			n++
		}
	}
	return n
}

// ---- screen ---------------------------------------------------------------

const trophyRows = 9

func (g *Game) updateTrophies() {
	g.time += 1.0 / 60
	n := len(trophyList)
	click := g.mouseMenu(func(mx, my float64) int {
		for i := 0; i < trophyRows; i++ {
			if mx >= 244 && mx <= 600 && my >= 120+float64(i)*30 && my <= 120+float64(i)*30+28 {
				return g.trophyScroll + i
			}
		}
		return -1
	}, &g.trophySel)
	_ = click
	if _, wy := ebiten.Wheel(); wy != 0 {
		g.trophySel = clampI(g.trophySel-int(wy)*3, 0, n-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.trophySel = clampI(g.trophySel+trophyRows, 0, n-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		g.trophySel = clampI(g.trophySel-trophyRows, 0, n-1)
	}
	if actDown.just() {
		g.trophySel = (g.trophySel + 1) % n
	}
	if actUp.just() {
		g.trophySel = (g.trophySel + n - 1) % n
	}
	if g.trophySel < g.trophyScroll {
		g.trophyScroll = g.trophySel
	}
	if g.trophySel >= g.trophyScroll+trophyRows {
		g.trophyScroll = g.trophySel - trophyRows + 1
	}
	if actBack.just() {
		g.mode = ModeMenu
	}
}

func (g *Game) drawTrophies(dst *ebiten.Image) {
	g.drawBedroom(dst)
	g.drawLogo(dst, 160, 30)
	fillRect(dst, 230, 56, 700, 440, rgba(0x05070e, 228))
	fillRect(dst, 230, 56, 700, 1, rgba(0x6a4ab0, 160))
	fillRect(dst, 230, 495, 700, 1, rgba(0x6a4ab0, 160))
	g.drawText(dst, "TROFEOS", 244, 66, rgb(0xff6a8a), 1.6)
	g.drawText(dst, fmt.Sprintf("%d / %d", g.trophyCount(), len(trophyList)), 770, 66, rgb(0xf0e060), 1.6)
	for i := 0; i < trophyRows; i++ {
		idx := g.trophyScroll + i
		if idx >= len(trophyList) {
			break
		}
		t := trophyList[idx]
		y := 120 + float64(i)*30
		col := rgb(0x3fb59a)
		pre := "  "
		if idx == g.trophySel {
			col, pre = rgb(0xff6a8a), "> "
			fillRect(dst, 244, y-3, 356, 26, rgba(0x2a1a4a, 160))
		}
		done := g.trophyDone(t)
		if !done && idx != g.trophySel {
			col = rgb(0x6a7280)
		}
		g.drawText(dst, pre+t.name, 250, y, col, 1.3)
		tag, tc := "", rgb(0x6a6680)
		if done {
			tag, tc = "CONSEGUIDO", rgb(0x70e090)
		} else if t.need > 1 {
			tag = fmt.Sprintf("%d/%d", minI(g.progress.Stats[t.stat], t.need), t.need)
		}
		g.drawText(dst, tag, 600-g.textW(tag, 1.1), y+2, tc, 1.1)
	}
	g.drawText(dst, fmt.Sprintf("%d/%d", g.trophySel+1, len(trophyList)), 244, 396, rgb(0x6a6680), 1)
	t := trophyList[g.trophySel]
	fillRect(dst, 640, 100, 270, 330, rgba(0x120c24, 220))
	cup := rgb(0x5a5668)
	if g.trophyDone(t) {
		cup = rgb(0xe8c040)
	}
	// a little cup
	filledEllipse(dst, 775, 150, 26, 22, cup)
	fillRect(dst, 749, 126, 52, 24, cup)
	fillRect(dst, 770, 168, 10, 16, cup)
	fillRect(dst, 756, 184, 38, 8, cup)
	fillRect(dst, 740, 132, 8, 14, cup)
	fillRect(dst, 802, 132, 8, 14, cup)
	g.drawTextC(dst, t.name, 775, 204, rgb(0xf4f0ff), 1.3)
	g.drawWrapped(dst, "Cómo conseguirlo: "+t.how, 650, 232, 250, rgb(0xb8b0d0))
	status := "Premio (aún bloqueado): "
	scol := rgb(0xf0e060)
	if g.trophyDone(t) {
		status, scol = "Premio desbloqueado: ", rgb(0x70e090)
	}
	g.drawWrapped(dst, status+t.reward.name+". "+t.reward.desc, 650, 300, 250, scol)
	g.drawText(dst, "Los premios se usan desde la TIENDA, en su lista (marcados con TROFEO).", 244, 440, rgb(0x8a86a0), 1)
	g.drawText(dst, "↑/↓ o rueda elegir   "+actionHint(actBack)+" volver", 244, 470, rgb(0x8a86a0), 1)
}

var _ = strings.ToLower
