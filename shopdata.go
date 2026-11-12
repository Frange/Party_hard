package main

import (
	"sort"
)

// Every list of the shop; the screen groups them (see shopGroups).
const (
	shopChars = iota
	shopOutfits
	shopAccess
	shopPerks
	shopAnimals
	shopKillers
	shopWeapons
	shopPolice
	shopUniforms
	shopWardrobe // not a list: the screen where each player puts on what he owns
	shopTabs
)

var shopTabNames = [shopTabs]string{"PERSONAJES", "ROPA", "ACCESORIOS", "OBJETOS", "ANIMALES", "ASESINOS", "ARMAS", "TIPOS", "UNIFORMES", "VESTUARIO"}

// shopGroups: the top level of the shop; each group has its own lists.
var shopGroups = []struct {
	name string
	tabs []int
}{
	{"JUGADOR", []int{shopOutfits, shopAccess, shopPerks}},
	{"ANIMALES", []int{shopAnimals}},
	{"ASESINOS", []int{shopWeapons}},   // the killers themselves are not sold: they are goals (goalkillers.go)
	{"VESTUARIO", []int{shopWardrobe}}, // ALWAYS the last group: drawn apart, further right and in its own colour (shop.go)
	// the police is not in the shop any more: its types are goals (goalcops.go) and its uniforms are won by escaping (copsets.go)
}

// shopFlat is every list in screen order (left/right walks through it).
// The characters (shopChars), the killers (shopKillers), the police types (shopPolice) and the uniforms (shopUniforms) are
// no longer sold: their lists only keep the rewards of the trophies and the data of the wardrobe, and are not shown as
// a shop list. They are goals now (OBJETIVOS, album.go).
var shopFlat = []int{shopOutfits, shopAccess, shopPerks, shopAnimals, shopWeapons, shopWardrobe}

func shopGroupOf(tab int) int {
	for gi, gr := range shopGroups {
		for _, t := range gr.tabs {
			if t == tab {
				return gi
			}
		}
	}
	return 0
}

type shopItem struct {
	id, name, desc string
	price          int
	slot           string // wearable slot ("" = not wearable)
	trophy         string // won with this trophy (not for sale)
	arg            string // preset slug / animal kind
}

var shopCatalog [shopTabs][]shopItem

// shopOwned is read from everywhere (police tiers, animals, killers).
var shopOwned = map[string]bool{}

func ownedID(id string) bool { return shopOwned[id] }

// wearFx: how each wearable changes the player's look (or the police's).
var wearFx = map[string]func(*CharSpec){}

// weaponHold maps a killer-weapon item to the item the killers carry.
var weaponHold = map[string]int{}

// slots, in the order they are applied on top of the base character.
var wearSlots = []string{"outfit", "skincol", "hair", "haircol", "beard", "mask", "hat", "glasses", "cape", "shoes"}

func addWear(tab int, id, name, slot string, price int, fx func(*CharSpec)) {
	wearFx[id] = fx
	desc := "Se lleva puesto en la fiesta. Enter: ponértelo / quitártelo."
	it := shopItem{id: id, name: name, desc: desc, price: price, slot: slot}
	if !shopKeepWear[id] { // left the catalogue (shopkeep.go): not for sale, but whoever owns it keeps wearing it
		shopRetired = append(shopRetired, it)
		return
	}
	shopCatalog[tab] = append(shopCatalog[tab], it)
}

// buildShopCatalog fills every tab (called once the presets are registered).
func buildShopCatalog() {
	for i := range shopCatalog {
		shopCatalog[i] = nil
	}
	shopRetired = nil
	c := &shopCatalog

	// ---- main characters: NOT sold any more (goalchars.go: they are unlocked by killing a guest of that kind) ----

	// ---- hats --------------------------------------------------------------
	type hatDef struct {
		id, name string
		kind     int
		cols     []uint32
		cn       []string
		price    int
	}
	basic := []string{"roja", "azul", "negra", "blanca", "verde", "rosa", "amarilla"}
	basicC := []uint32{0xd02828, 0x2a5ac8, 0x1a1a22, 0xf0f0f0, 0x2a9a3a, 0xe85a9a, 0xe8c830}
	for _, h := range []hatDef{
		{"cap", "Gorra", HatCap, basicC, basic, 120}, {"beanie", "Gorro", HatBeanie, basicC, basic, 120}, {"bucket", "Sombrero de pescador", HatBucket, basicC[:5], basic[:5], 160},
	} {
		for i, col := range h.cols {
			cc := col
			k := h.kind
			addWear(shopAccess, "hat:"+h.id+h.cn[i], h.name+" "+h.cn[i], "hat", h.price+i*10, func(s *CharSpec) { s.Hat, s.HatCol = k, hx(cc) })
		}
	}
	hat := func(id, name string, kind int, col uint32, price int) {
		addWear(shopAccess, "hat:"+id, name, "hat", price, func(s *CharSpec) { s.Hat, s.HatCol = kind, hx(col) })
	}
	hat("cowboy", "Sombrero de vaquero", HatCowboy, 0x8a5a2a, 300)
	hat("cowboyn", "Sombrero de vaquero negro", HatCowboy, 0x1a1a22, 340)
	hat("fedora", "Fedora", HatFedora, 0x3a3a46, 300)
	hat("fedorab", "Fedora blanca", HatFedora, 0xe8e8e0, 340)
	hat("bowler", "Bombín", HatBowler, 0x1a1a22, 320)
	hat("bowlerm", "Bombín marrón", HatBowler, 0x6a4a2a, 340)
	hat("sombrero", "Sombrero mexicano", HatSombrero, 0xd8a830, 350)
	hat("sombreror", "Sombrero mexicano rojo", HatSombrero, 0xc83a2a, 380)
	hat("beret", "Boina negra", HatBeret, 0x14141a, 250)
	hat("beretr", "Boina roja", HatBeret, 0xc82a2a, 260)
	hat("ushanka", "Ushanka", HatUshanka, 0x6a6a74, 280)
	hat("chef", "Gorro de cocinero", HatChef, 0xf4f4f4, 260)
	hat("tophat", "Chistera", HatTopHat, 0x14141a, 450)
	hat("tophatp", "Chistera morada", HatTopHat, 0x6a2a8a, 520)
	hat("pirate", "Sombrero pirata", HatPirate, 0x14141a, 520)
	hat("viking", "Casco vikingo", HatViking, 0xc8c8d0, 500)
	hat("horns", "Cuernos de diablo", HatHorns, 0xc02a22, 480)
	hat("wig", "Peluca blanca", HatWig, 0xf4f4f4, 420)
	hat("wizard", "Sombrero de mago", HatWizard, 0x5a2a9a, 600)
	hat("wizardb", "Sombrero de mago azul", HatWizard, 0x2a4ab8, 600)
	hat("wizardr", "Sombrero de mago rojo", HatWizard, 0xc02a2a, 600)
	hat("fez", "Fez", HatFez, 0xc02a2a, 380)
	hat("hardhat", "Casco de obra amarillo", HatHardhat, 0xe8c830, 300)
	hat("hardhato", "Casco de obra naranja", HatHardhat, 0xf08a20, 300)
	hat("partycone", "Gorro de cumpleaños", HatPartyCone, 0xe85a9a, 200)
	hat("cone", "Cono de tráfico", HatCone, 0xf08a20, 350)
	hat("turban", "Turbante blanco", HatTurban, 0xf4f4f4, 420)
	hat("turbanr", "Turbante rojo", HatTurban, 0xc02a2a, 440)
	hat("tyrol", "Sombrero tirolés", HatTyrol, 0x2a8a4a, 380)
	hat("montera", "Montera", HatMontera, 0x14141a, 500)
	hat("moto", "Casco de moto rojo", HatMoto, 0xc02a2a, 450)
	hat("motok", "Casco de moto negro", HatMoto, 0x14141a, 450)
	hat("helmet", "Casco militar", HatHelmet, 0x4a5a3a, 600)
	hat("ninja", "Capucha ninja", HatNinja, 0x14141a, 700)
	hat("ninjar", "Capucha ninja roja", HatNinja, 0xc02a2a, 740)
	hat("samurai", "Casco samurái", HatSamurai, 0x8a2a2a, 900)
	hat("crown", "Corona", HatCrown, 0xe8c040, 900)
	hat("bunny", "Orejas de conejo", HatBunny, 0xf0a0c0, 450)
	hat("bunnyb", "Orejas de conejo blancas", HatBunny, 0xf4f4f4, 450)
	hat("pumpkin", "Calabaza", HatPumpkin, 0xf08a20, 800)
	hat("chicken", "Gorro de pollo", HatChicken, 0xf8e040, 1000)
	hat("dino", "Gorro de dinosaurio", HatDino, 0x4aa04a, 1100)
	hat("shark", "Gorro de tiburón", HatShark, 0x4a7aaa, 1100)
	hat("unicorn", "Gorro de unicornio", HatUnicorn, 0xf4f4f4, 1200)
	hat("tv", "Tele en la cabeza", HatTV, 0x6a6a74, 1500)
	hat("antenna", "Antena alienígena", HatAntenna, 0x4aa04a, 900)
	hat("trooper", "Casco imperial", HatTrooper, 0xf4f4f4, 2500)
	hat("vader", "Casco oscuro", HatVader, 0x16161c, 3000)

	// ---- outfits -----------------------------------------------------------
	type colDef struct {
		n string
		c uint32
	}
	suitCols := []colDef{{"negro", 0x1a1a22}, {"blanco", 0xf0f0ee}, {"rojo", 0xb82030}, {"azul", 0x2a3a8a}, {"rosa", 0xe85a9a}, {"verde", 0x2a7a4a}, {"morado", 0x6a2a8a}, {"dorado", 0xd8b040}}
	for i, cd := range suitCols {
		cc := cd.c
		addWear(shopOutfits, "out:suit"+cd.n, "Traje "+cd.n, "outfit", 400+i*60, func(s *CharSpec) {
			s.Top, s.Shirt, s.Shirt2, s.Pants, s.Accent, s.Bottom = TopSuit, hx(cc), hx(0xf4f4f4), hx(cc), hx(0xc82a2a), BotPants
		})
	}
	hoodCols := []colDef{{"gris", 0x4a4a56}, {"rojo", 0xc02a2a}, {"azul", 0x2a4ab8}, {"verde", 0x2a8a4a}, {"negra", 0x1a1a22}, {"rosa", 0xe85a9a}, {"morada", 0x6a2a8a}, {"naranja", 0xf08a20}}
	for i, cd := range hoodCols {
		cc := cd.c
		addWear(shopOutfits, "out:hood"+cd.n, "Sudadera "+cd.n, "outfit", 200+i*20, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2 = TopHoodie, hx(cc), hx(cc) })
	}
	for i, cd := range []colDef{{"roja", 0xc82a2a}, {"azul", 0x2a4ab8}, {"verde", 0x2a8a4a}, {"negra", 0x1a1a22}, {"blanca", 0xf0f0f0}, {"amarilla", 0xe8c830}} {
		cc := cd.c
		addWear(shopOutfits, "out:track"+cd.n, "Chándal "+cd.n, "outfit", 300+i*20, func(s *CharSpec) {
			s.Top, s.Shirt, s.Shirt2, s.Pants, s.Bottom = TopLong, hx(cc), hx(cc), hx(cc), BotPants
		})
		addWear(shopOutfits, "out:tee"+cd.n, "Camiseta "+cd.n, "outfit", 150+i*10, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2 = TopTee, hx(cc), hx(cc) })
		addWear(shopOutfits, "out:tank"+cd.n, "Tirantes "+cd.n, "outfit", 160+i*10, func(s *CharSpec) {
			s.Top, s.Shirt, s.Shirt2, s.Bottom, s.Pants = TopTank, hx(cc), hx(cc), BotShorts, hx(0x3a4a6a)
		})
		addWear(shopOutfits, "out:sweat"+cd.n, "Jersey "+cd.n, "outfit", 220+i*10, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2 = TopLong, hx(cc), hx(cc) })
	}
	for i, cd := range []colDef{{"rosa", 0xe85a9a}, {"azul", 0x3aa0d0}, {"verde", 0x3aa05a}, {"naranja", 0xf08a20}} {
		cc := cd.c
		addWear(shopOutfits, "out:hawaii"+cd.n, "Camisa hawaiana "+cd.n, "outfit", 350+i*20, func(s *CharSpec) {
			s.Top, s.Shirt, s.Shirt2, s.Bottom, s.Pants = TopHawaiian, hx(cc), hx(0xf4f4f0), BotShorts, hx(0xe8d8a0)
		})
	}
	for i, cd := range []colDef{{"azul", 0x2aa0d8}, {"rojo", 0xd02828}, {"negro", 0x1a1a22}, {"amarillo", 0xe8c830}, {"verde", 0x2a9a4a}, {"rosa", 0xe85a9a}} {
		cc := cd.c
		addWear(shopOutfits, "out:swim"+cd.n, "Bañador "+cd.n, "outfit", 180+i*20, func(s *CharSpec) { s.Top, s.Shirt, s.Bottom, s.Pants = TopBare, s.Skin, BotTrunks, hx(cc) })
	}
	for i, cd := range []colDef{{"morada", 0x7a2a8a}, {"negra", 0x1a1a22}, {"roja", 0xa01828}, {"blanca", 0xf0f0ee}, {"verde", 0x2a6a3a}, {"dorada", 0xd8b040}} {
		cc := cd.c
		addWear(shopOutfits, "out:robe"+cd.n, "Túnica "+cd.n, "outfit", 500+i*40, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2 = TopRobe, hx(cc), hx(cc) })
	}
	for i, cd := range []colDef{{"gris", 0x8a8e98}, {"negra", 0x2a2a32}, {"dorada", 0xd8b040}, {"verde", 0x4a6a3a}} {
		cc := cd.c
		addWear(shopOutfits, "out:armor"+cd.n, "Armadura "+cd.n, "outfit", 1500+i*300, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2, s.Pants = TopArmor, hx(cc), hx(cc), shade(hx(cc), 0.6) })
	}
	for i, cd := range []colDef{{"amarillo", 0xe8c830}, {"naranja", 0xf08a20}, {"verde", 0x9ad030}} {
		cc := cd.c
		addWear(shopOutfits, "out:vest"+cd.n, "Chaleco reflectante "+cd.n, "outfit", 280+i*20, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2 = TopVest, hx(0x3a3a46), hx(cc) })
	}
	for i, cd := range []colDef{{"rosa", 0xe85a9a}, {"azul", 0x3a6ae0}, {"roja", 0xc82a2a}, {"negro", 0x1a1a22}, {"verde", 0x2a8a4a}} {
		cc := cd.c
		addWear(shopOutfits, "out:dress"+cd.n, "Vestido "+cd.n, "outfit", 600+i*50, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2, s.Pattern = TopDress, hx(cc), hx(cc), 0 })
	}
	addWear(shopOutfits, "out:uniform", "Uniforme azul", "outfit", 450, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants = TopUniform, hx(0x3a62c0), hx(0xf4f4f8), hx(0x1b2a5e)
	})
	addWear(shopOutfits, "out:uniformg", "Uniforme verde", "outfit", 460, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants = TopUniform, hx(0x3a6a3a), hx(0xf4f4f8), hx(0x2a3a22)
	})
	addWear(shopOutfits, "out:lab", "Bata de laboratorio", "outfit", 500, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants = TopSuit, hx(0xf4f4f4), hx(0xd0d8e8), hx(0x4a4a56)
	})
	addWear(shopOutfits, "out:pijama", "Pijama de rayas", "outfit", 350, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants, s.Pattern = TopLong, hx(0x6aa8e8), hx(0xf4f4f4), hx(0x6aa8e8), 2
	})
	addWear(shopOutfits, "out:prison", "Mono de presidiario", "outfit", 600, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants, s.Pattern = TopLong, hx(0xf08a20), hx(0x14141a), hx(0xf08a20), 2
	})
	addWear(shopOutfits, "out:zebra", "Camisa de cebra", "outfit", 450, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2, s.Pattern = TopTee, hx(0xf4f4f4), hx(0x14141a), 2 })
	addWear(shopOutfits, "out:rainbow", "Camiseta arcoíris", "outfit", 700, func(s *CharSpec) { s.Top, s.Shirt, s.Shirt2, s.Pattern = TopTee, hx(0xe85a9a), hx(0x3ad0a8), 3 })
	addWear(shopOutfits, "out:gold", "Traje dorado de gala", "outfit", 3000, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Pants, s.Accent = TopSuit, hx(0xd8b040), hx(0xf4e8a0), hx(0xd8b040), hx(0xa01020)
	})

	// ---- extras: hair, beard, colours, masks, capes, shoes, glasses ------------
	hairs := []struct {
		id, n string
		k     int
		p     int
	}{{"short", "Pelo corto", HairShort, 100}, {"long", "Pelo largo", HairLong, 150}, {"bald", "Calvo", HairBald, 100}, {"mohawk", "Cresta", HairMohawk, 250}, {"curly", "Rizos", HairCurly, 200},
		{"pompadour", "Tupé", HairPompadour, 250}, {"spiky", "Pelo de punta", HairSpiky, 200}, {"bun", "Moño", HairBun, 160}, {"bob", "Media melena", HairBob, 160}, {"pony", "Coleta", HairPony, 160},
		{"fry", "Pelo alborotado", HairFry, 220}, {"dreads", "Rastas", HairDreads, 300}, {"beehive", "Colmena", HairBeehive, 400}, {"star", "Pelo de estrella", HairStar, 500}, {"quiff", "Tupé alto", HairQuiff, 300},
		{"bowl", "Corte de tazón", HairBowl, 150}, {"buns", "Dos moños", HairBuns, 220}, {"messy", "Pelo desastre", HairMessy, 180}, {"comb", "Peinado hacia atrás", HairComb, 200}, {"homer", "Casi calvo", HairHomer, 300}}
	for _, h := range hairs {
		k := h.k
		addWear(shopAccess, "hair:"+h.id, h.n, "hair", h.p, func(s *CharSpec) { s.Hair2 = k })
	}
	for i, cd := range []colDef{{"negro", 0x14101a}, {"castaño", 0x5a3a22}, {"rubio", 0xe8c860}, {"pelirrojo", 0xc84a1a}, {"blanco", 0xf0f0f0}, {"azul", 0x3a60e0}, {"rosa", 0xf060a0}, {"verde", 0x3aa040}, {"morado", 0x8a3aca}, {"naranja", 0xf08a20}} {
		cc := cd.c
		addWear(shopAccess, "haircol:"+cd.n, "Pelo "+cd.n, "haircol", 120+i*30, func(s *CharSpec) { s.Hair = hx(cc) })
	}
	for i, b := range []struct {
		id, n string
		k     int
	}{{"stubble", "Barba de tres días", BeardStubble}, {"full", "Barba completa", BeardFull}, {"mustache", "Bigote", BeardMustache}, {"sideburns", "Patillas", BeardSideburns}, {"goatee", "Perilla", BeardGoatee}} {
		k := b.k
		addWear(shopAccess, "beard:"+b.id, b.n, "beard", 120+i*40, func(s *CharSpec) { s.Beard = k })
	}
	for i, g := range []struct {
		id, n string
		k     int
	}{{"round", "Gafas redondas", GlassRound}, {"shades", "Gafas de sol", GlassShades}, {"big", "Gafotas", GlassBig}, {"white", "Gafas blancas", GlassWhite}, {"neon", "Gafas de neón", GlassNeon}} {
		k := g.k
		addWear(shopAccess, "gl:"+g.id, g.n, "glasses", 150+i*60, func(s *CharSpec) { s.Glasses = k })
	}
	for i, cd := range []colDef{{"roja", 0xc82a2a}, {"negra", 0x1a1a24}, {"morada", 0x6a2a9a}, {"azul", 0x2a4ab8}, {"dorada", 0xe8c040}, {"verde", 0x2a8a4a}, {"blanca", 0xf0f0f0}} {
		cc := cd.c
		addWear(shopAccess, "cape:"+cd.n, "Capa "+cd.n, "cape", 400+i*60, func(s *CharSpec) { s.Cape = hx(cc) })
	}
	for i, cd := range []colDef{{"rojas", 0xd82a2a}, {"blancas", 0xf4f4f4}, {"negras", 0x14141a}, {"doradas", 0xe8c040}, {"verdes", 0x2a9a4a}, {"rosas", 0xf060a0}, {"azules", 0x2a60d0}} {
		cc := cd.c
		addWear(shopAccess, "shoe:"+cd.n, "Zapatillas "+cd.n, "shoes", 120+i*25, func(s *CharSpec) { s.Shoes = hx(cc) })
	}
	for i, cd := range []colDef{{"verde alienígena", 0x60b848}, {"azul", 0x5a8ad8}, {"rosa", 0xf0a0c0}, {"dorada", 0xe8c040}, {"gris", 0x9a9aa8}, {"naranja", 0xf0a050}, {"pálida", 0xf0e8e8}, {"morena", 0x8a5a3a}} {
		cc := cd.c
		addWear(shopAccess, "skincol:"+cd.n, "Piel "+cd.n, "skincol", 200+i*50, func(s *CharSpec) { s.Skin = hx(cc) })
	}
	for i, m := range []struct {
		id, n string
		k     int
		col   uint32
		p     int
	}{{"plain", "Máscara blanca", MaskPlain, 0xe4dccc, 500}, {"hockey", "Máscara de hockey", MaskHockey, 0xe8e4d0, 650}, {"ghost", "Máscara de fantasma", MaskGhost, 0xf2f2f2, 650},
		{"pig", "Máscara de cerdo", MaskPig, 0xc4a07c, 700}, {"clown", "Máscara de payaso", MaskClown, 0xf4f4f0, 700}, {"skull", "Calavera", MaskSkull, 0xf0f0e0, 800}} {
		k, cc := m.k, m.col
		addWear(shopAccess, "mask:"+m.id, m.n, "mask", m.p+i, func(s *CharSpec) { s.Mask, s.MaskKind = hx(cc), k })
	}

	// ---- perks (tiers) -----------------------------------------------------
	perk := func(id, name, desc string, price int) {
		c[shopPerks] = append(c[shopPerks], shopItem{id: id, name: name, desc: desc, price: price})
	}
	perk("perk:speed1", "Zapatillas veloces I", "Caminas y corres un 8 % más rápido.", 500)
	perk("perk:speed2", "Zapatillas veloces II", "Un 16 % más rápido en total.", 1500)
	perk("perk:speed3", "Zapatillas veloces III", "Un 25 % más rápido en total.", 4000)
	perk("perk:stamina1", "Pulmones de acero I", "Puedes correr 9 segundos seguidos.", 600)
	perk("perk:stamina2", "Pulmones de acero II", "Puedes correr 12 segundos seguidos.", 1800)
	perk("perk:stamina3", "Pulmones de acero III", "Puedes correr 16 segundos seguidos.", 4500)
	perk("perk:glove1", "Guantes ligeros I", "Apuñalas más rápido (recarga un 20 % menor).", 700)
	perk("perk:glove2", "Guantes ligeros II", "Recarga un 35 % menor.", 2000)
	perk("perk:glove3", "Guantes ligeros III", "Recarga un 50 % menor.", 5000)
	perk("perk:money1", "Cartera dorada I", "Ganas un 10 % más de puntos en cada fiesta.", 1200)
	perk("perk:money2", "Cartera dorada II", "Ganas un 25 % más de puntos.", 4000)
	perk("perk:money3", "Cartera dorada III", "Ganas un 50 % más de puntos.", 9000)

	// ---- animals -----------------------------------------------------------
	an := func(kind, name string, price int) {
		c[shopAnimals] = append(c[shopAnimals], shopItem{id: "animal:" + kind, name: name, desc: "Aparece en las fiestas una vez desbloqueado.", price: price, arg: kind})
	}
	an("rabbit", "Conejos", 300)
	an("duck", "Patos", 300)
	an("pig", "Cerdos", 350)
	an("budgie", "Periquitos azules", 350)
	an("deer", "Ciervos", 500)
	an("giraffe", "Jirafas", 800)
	an("kangaroo", "Canguros", 1000)
	an("elephant", "Elefantes", 1500)
	// wild ones (animals6.go): they come now and then where the level has the right ground; the big cats very rarely
	an("boar", "Jabalíes", 700)
	an("croc", "Cocodrilos", 1800)
	an("cheetah", "Guepardos", 2500)
	an("lion", "Leones", 3000)

	// ---- killers: NOT sold any more (goalkillers.go: they are unlocked by killing the killer) ----

	for _, w := range []struct {
		id, name string
		price    int
		hold     int
	}{{"knife", "Cuchillo de cocina", 300, HoldKnife}, {"baguette", "Baguette dura", 350, HoldBaguette}, {"broom", "Escoba", 400, HoldBroom}, {"plant", "Planta de interior", 420, HoldPlant}, {"beer", "Botella rota", 450, HoldBeer}, {"pizza", "Pizza gigante", 500, HoldPizza}, {"sign", "Pancarta", 550, HoldSign}, {"flag", "Bandera", 600, HoldFlag}, {"vacuum", "Aspiradora", 700, HoldVacuum}, {"megaphone", "Megáfono", 750, HoldMegaphone}, {"selfie", "Palo selfie", 800, HoldSelfie}, {"briefcase", "Maletín", 900, HoldBriefcase}, {"trophy", "Trofeo", 1000, HoldTrophy}, {"saber", "Sable láser", 2500, HoldSaber}, {"guitar", "Guitarra eléctrica", 1500, HoldGuitar}, {"sax", "Saxofón", 1300, HoldSax}, {"machete", "Machete", 500, HoldMachete}, {"torch", "Antorcha", 600, HoldTorch}, {"claw", "Garras", 700, HoldClaw}, {"chainsaw", "Motosierra", 1200, HoldChainsaw},
		// more weapons for the killers (2026-11-09): things the guests already carry, so the art is the same one
		{"cup", "Taza de café hirviendo", 650, HoldCup}, {"mic", "Micrófono de karaoke", 900, HoldMic}, {"phone", "Móvil sin batería", 950, HoldPhone},
		{"tray", "Bandeja de canapés", 1000, HoldTray}, {"camera", "Cámara con flash", 1100, HoldCamera}, {"bigcup", "Copa de litro", 1200, HoldBigCup},
		{"pompom", "Pompones de animadora", 1400, HoldPompom}, {"float", "Flotador salvavidas", 1500, HoldFloat}, {"stick", "Palo de hockey", 1600, HoldStick},
		{"balloons", "Ramo de globos", 1800, HoldBalloons}, {"board", "Tabla de socorrista", 2000, HoldBoard}, {"ext", "Extintor", 2200, HoldExt}} {
		weaponHold["kw:"+w.id] = w.hold
		c[shopWeapons] = append(c[shopWeapons], shopItem{id: "kw:" + w.id, name: w.name, desc: "Todos los asesinos llevan esta arma. Enter: usar / quitar.", price: w.price, slot: "kweapon"})
	}

	// ---- police: tiers and uniforms ----------------------------------------
	c[shopPolice] = []shopItem{
		{id: "police:0", name: "Policía local", desc: "Siempre disponible: 1 estrella.", price: 0},
		// the other types are not for sale: each one is unlocked by a trophy (kills milestones)
		{id: "police:1", name: "FBI", desc: "Molder y Skully llegan con 2 estrellas.", price: 1<<20 + 701, arg: "1", trophy: "t_cop1"},
		{id: "police:2", name: "Ejército", desc: "Soldados con 3 estrellas.", price: 1<<20 + 702, arg: "2", trophy: "t_cop2"},
		{id: "police:3", name: "SWAT", desc: "Operadores con 4 estrellas.", price: 1<<20 + 703, arg: "3", trophy: "t_cop3"},
		{id: "police:4", name: "Pawer Rangers", desc: "Llegan con 5 estrellas.", price: 1<<20 + 704, arg: "4", trophy: "t_cop4"},
	}
	// the uniforms are not sold (copsets.go): the 36 kept sets open with escapes, the other 72 are only worn by whoever
	// bought them before. Every one is a locked entry of shopUniforms (never buyable); the price argument is the old list price.
	uni := func(slot, id, name string, price int, fx func(*CharSpec)) {
		wearFx[id] = fx
		who := map[string]string{"cop": "la policía local", "fbi": "el FBI", "army": "el ejército", "swat": "el SWAT", "rangers": "los Pawer Rangers"}[slot]
		lock := "retired"
		if copSetRank(id) > 0 {
			lock = "escapes"
		}
		c[shopUniforms] = append(c[shopUniforms], shopItem{id: id, name: name, desc: "Uniforme de " + who + ". Se elige en VESTUARIO.", price: 1<<20 + 900 + len(c[shopUniforms]), slot: slot, trophy: lock})
	}
	cop := func(id, name string, price int, fx func(*CharSpec)) { uni("cop", "cop:"+id, name, price, fx) }
	cop("guardia", "Local · guardia civil", 400, func(s *CharSpec) { s.Shirt, s.Pants, s.HatCol = hx(0x3a6a3a), hx(0x2a3a22), hx(0x2a3a22) })
	cop("bobby", "Local · bobby británico", 500, func(s *CharSpec) {
		s.Hat, s.HatCol, s.Shirt, s.Pants = HatHelmet, hx(0x1a1a2a), hx(0x1a1a2a), hx(0x1a1a2a)
	})
	cop("miami", "Local · Miami", 600, func(s *CharSpec) {
		s.Shirt, s.Pants, s.Glasses, s.Hat = hx(0xe8e8f0), hx(0x8ab0e0), GlassShades, HatNone
	})
	cop("retro", "Local · retro bigotudo", 700, func(s *CharSpec) { s.Beard, s.Glasses = BeardMustache, GlassShades })
	cop("rosa", "Local · policía rosa", 800, func(s *CharSpec) { s.Shirt, s.Pants, s.HatCol = hx(0xe85a9a), hx(0xa0306a), hx(0xa0306a) })
	cop("montada", "Local · policía montada", 900, func(s *CharSpec) {
		s.Shirt, s.Pants, s.Hat, s.HatCol = hx(0xc02a2a), hx(0x1a1a2a), HatCowboy, hx(0x6a4a2a)
	})
	cop("hawaii", "Local · patrulla de playa", 600, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Bottom, s.Pants, s.Hat = TopHawaiian, hx(0xe85a9a), hx(0xf4f4f0), BotShorts, hx(0xe8d8a0), HatCap
	})
	cop("cocinero", "Local · cocinero", 700, func(s *CharSpec) {
		s.Hat, s.HatCol, s.Shirt, s.Pants = HatChef, hx(0xf4f4f4), hx(0xf4f4f4), hx(0x3a3a46)
	})
	cop("vaquero", "Local · sheriff", 800, func(s *CharSpec) {
		s.Hat, s.HatCol, s.Shirt, s.Pants, s.Beard = HatCowboy, hx(0x8a5a2a), hx(0x8a7a5a), hx(0x3a4a8a), BeardMustache
	})
	cop("ninja", "Local · policía ninja", 1100, func(s *CharSpec) {
		s.Hat, s.HatCol, s.Shirt, s.Pants = HatNinja, hx(0x14141a), hx(0x14141a), hx(0x14141a)
	})
	cop("mago", "Local · policía mago", 1200, func(s *CharSpec) {
		s.Hat, s.HatCol, s.Top, s.Shirt, s.Shirt2 = HatWizard, hx(0x5a2a9a), TopRobe, hx(0x5a2a9a), hx(0x5a2a9a)
	})
	cop("rey", "Local · policía real", 1400, func(s *CharSpec) {
		s.Hat, s.HatCol, s.Top, s.Shirt, s.Shirt2 = HatCrown, hx(0xe8c040), TopRobe, hx(0xa01830), hx(0xa01830)
	})
	cop("payaso", "Local · policías payaso", 1500, func(s *CharSpec) {
		s.Mask, s.MaskKind, s.Hat, s.HatCol = hx(0xf4f4f0), MaskClown, HatPartyCone, hx(0xe85a9a)
	})
	cop("vikingo", "Local · policías vikingos", 1700, func(s *CharSpec) { s.Hat, s.HatCol, s.Beard = HatViking, hx(0xc8c8d0), BeardFull })
	cop("pirata", "Local · policías piratas", 2000, func(s *CharSpec) { s.Hat, s.HatCol, s.Beard = HatPirate, hx(0x14141a), BeardFull })
	cop("samurai", "Local · policías samuráis", 2200, func(s *CharSpec) {
		s.Hat, s.HatCol, s.Top, s.Shirt, s.Shirt2 = HatSamurai, hx(0x8a2a2a), TopRobe, hx(0x2a2a3a), hx(0x2a2a3a)
	})
	cop("pollo", "Local · policías pollo", 2500, func(s *CharSpec) { s.Hat, s.HatCol = HatChicken, hx(0xf8e040) })
	cop("conejo", "Local · policías conejo", 2600, func(s *CharSpec) { s.Hat, s.HatCol = HatBunny, hx(0xf0a0c0) })
	cop("dino", "Local · policías dinosaurio", 3000, func(s *CharSpec) { s.Hat, s.HatCol = HatDino, hx(0x4aa04a) })
	cop("unicornio", "Local · policías unicornio", 3200, func(s *CharSpec) { s.Hat, s.HatCol = HatUnicorn, hx(0xf4f4f4) })
	cop("tiburon", "Local · policías tiburón", 3200, func(s *CharSpec) { s.Hat, s.HatCol = HatShark, hx(0x4a7aaa) })
	// more local uniforms: colour schemes and silly hats
	for i, sc := range []struct {
		n                 string
		shirt, pants, hat uint32
	}{{"azul marino", 0x1a2a5a, 0x141c3a, 0x1a2a5a}, {"negro", 0x1a1a22, 0x0e0e12, 0x1a1a22}, {"blanco", 0xf0f0f0, 0xc0c4cc, 0xf0f0f0}, {"verde militar", 0x4a5a3a, 0x2a3a22, 0x4a5a3a},
		{"gris", 0x6a6e78, 0x3a3e48, 0x6a6e78}, {"naranja", 0xf08a20, 0x8a4a10, 0xf08a20}, {"morado", 0x6a2a8a, 0x3a1a4a, 0x6a2a8a}, {"dorado", 0xd8b040, 0x6a5a20, 0xd8b040},
		{"rojo", 0xc02a2a, 0x5a1010, 0xc02a2a}, {"turquesa", 0x2aa0a0, 0x1a5a5a, 0x2aa0a0}, {"rosa pálido", 0xf0a0c0, 0xa06080, 0xf0a0c0}, {"marrón", 0x6a4a2a, 0x3a2a1a, 0x6a4a2a},
		{"amarillo", 0xe8c830, 0x8a7010, 0xe8c830}, {"celeste", 0x7ab0e0, 0x3a5a8a, 0x7ab0e0}, {"fucsia", 0xe03a8a, 0x8a1a50, 0xe03a8a}, {"lima", 0x9ad030, 0x4a6a10, 0x9ad030}} {
		sc := sc
		cop("col"+itoa(i), "Local · uniforme "+sc.n, 450+i*40, func(s *CharSpec) { s.Shirt, s.Pants, s.HatCol = hx(sc.shirt), hx(sc.pants), hx(sc.hat) })
	}
	for i, h := range []struct {
		n   string
		k   int
		col uint32
	}{{"gorra roja", HatCap, 0xd02828}, {"boina negra", HatBeret, 0x14141a}, {"chistera", HatTopHat, 0x14141a}, {"sombrero mexicano", HatSombrero, 0xd8a830}, {"fez", HatFez, 0xc02a2a},
		{"bombín", HatBowler, 0x14141a}, {"casco de obra", HatHardhat, 0xe8c830}, {"turbante", HatTurban, 0xf4f4f4}, {"montera", HatMontera, 0x14141a}, {"tirolés", HatTyrol, 0x2a8a4a},
		{"ushanka", HatUshanka, 0x6a6a74}, {"antena", HatAntenna, 0x4aa04a}, {"tele en la cabeza", HatTV, 0x6a6a74}, {"cono de tráfico", HatCone, 0xf08a20}, {"gorro de fiesta", HatPartyCone, 0xe85a9a},
		{"gorro de lana", HatBeanie, 0x2a5ac8}, {"calabaza", HatPumpkin, 0xf08a20}, {"casco de moto", HatMoto, 0xc02a2a}, {"cuernos", HatHorns, 0xc02a22}, {"peluca blanca", HatWig, 0xf4f4f4},
		{"sombrero de fieltro", HatFedora, 0x3a3a46}, {"gorro de cocinero", HatChef, 0xf4f4f4}, {"casco militar", HatHelmet, 0x4a5a3a}, {"sombrero de paja", HatBucket, 0xd8c060}} {
		h := h
		cop("hat"+itoa(i), "Local · "+h.n, 600+i*70, func(s *CharSpec) { s.Hat, s.HatCol = h.k, hx(h.col) })
	}
	fbi := func(id, name string, price int, fx func(*CharSpec)) { uni("fbi", "fbi:"+id, name, price, fx) }
	fbi("negro", "FBI · hombres de negro", 1300, func(s *CharSpec) {
		s.Shirt, s.Pants, s.Shirt2, s.Glasses = hx(0x0e0e12), hx(0x0e0e12), hx(0xf4f4f8), GlassShades
	})
	fbi("gabardina", "FBI · gabardina beige", 1500, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Hat, s.HatCol = TopRobe, hx(0x8a7a5a), hx(0x8a7a5a), HatFedora, hx(0x5a4a3a)
	})
	fbi("rosa", "FBI · agentes rosas", 1800, func(s *CharSpec) { s.Shirt, s.Pants = hx(0xe85a9a), hx(0xa0306a) })
	fbi("hawaii", "FBI · de vacaciones", 2000, func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2, s.Bottom, s.Pants = TopHawaiian, hx(0x3aa0d0), hx(0xf4f4f0), BotShorts, hx(0xe8d8a0)
	})
	fbi("alien", "FBI · expediente X", 2800, func(s *CharSpec) { s.Skin, s.Eyes = hx(0x60b848), EyesAlien })
	for i, sc := range []struct {
		n           string
		shirt, pant uint32
	}{{"traje azul", 0x2a3a6a, 0x2a3a6a}, {"traje gris", 0x6a6e78, 0x4a4e58}, {"traje marrón", 0x6a4a2a, 0x4a3a2a}, {"traje blanco", 0xf0f0f0, 0xd0d0d8}, {"traje verde", 0x2a6a3a, 0x1a4a2a}, {"traje rojo", 0xb02828, 0x6a1818}, {"traje morado", 0x6a2a8a, 0x3a1a4a}, {"traje dorado", 0xd8b040, 0x8a7020}} {
		sc := sc
		fbi("c"+itoa(i), "FBI · "+sc.n, 1400+i*120, func(s *CharSpec) { s.Shirt, s.Pants = hx(sc.shirt), hx(sc.pant) })
	}
	army := func(id, name string, price int, fx func(*CharSpec)) { uni("army", "army:"+id, name, price, fx) }
	army("desierto", "Ejército · camuflaje desierto", 2300, func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0xb8a070), hx(0xb8a070), hx(0x9a8858), hx(0xb8a070), hx(0xb8a070)
	})
	army("nieve", "Ejército · camuflaje nieve", 2400, func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0xe8eef4), hx(0xe8eef4), hx(0xc8d0d8), hx(0xe8eef4), hx(0xe8eef4)
	})
	army("boinas", "Ejército · boinas rojas", 2600, func(s *CharSpec) { s.Hat, s.HatCol = HatBeret, hx(0xc02a2a) })
	army("rosa", "Ejército · camuflaje rosa", 2800, func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0xe85a9a), hx(0xe85a9a), hx(0xa0306a), hx(0xe85a9a), hx(0xe85a9a)
	})
	for i, sc := range []struct {
		n   string
		col uint32
	}{{"camuflaje bosque", 0x4a6a3a}, {"camuflaje urbano", 0x6a6e78}, {"camuflaje azul", 0x3a5a8a}, {"camuflaje naranja", 0xd8782a}, {"camuflaje morado", 0x7a3aa8}, {"camuflaje negro", 0x1a1a22}, {"camuflaje dorado", 0xd8b040}, {"camuflaje rojo", 0xa02828}} {
		sc := sc
		army("c"+itoa(i), "Ejército · "+sc.n, 2100+i*130, func(s *CharSpec) {
			s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(sc.col), hx(sc.col), shade(hx(sc.col), 0.7), hx(sc.col), hx(sc.col)
		})
	}
	swat := func(id, name string, price int, fx func(*CharSpec)) { uni("swat", "swat:"+id, name, price, fx) }
	swat("azul", "SWAT · azul marino", 3700, func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0x1a2a5a), hx(0x1a2a5a), hx(0x141c3a), hx(0x1a2a5a), hx(0x1a2a5a)
	})
	swat("blanco", "SWAT · invierno", 3900, func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0xe8eef4), hx(0xe8eef4), hx(0xa8b0b8), hx(0xe8eef4), hx(0xe8eef4)
	})
	swat("rojo", "SWAT · rojo", 4200, func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0x8a1a1a), hx(0x8a1a1a), hx(0x3a1010), hx(0x8a1a1a), hx(0x8a1a1a)
	})
	swat("dorado", "SWAT · de gala dorado", 5500, func(s *CharSpec) {
		s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(0xd8b040), hx(0xd8b040), hx(0x6a5a20), hx(0xd8b040), hx(0xd8b040)
	})
	for i, sc := range []struct {
		n   string
		col uint32
	}{{"negro", 0x14141a}, {"verde", 0x2a4a2a}, {"gris", 0x6a6e78}, {"naranja", 0xd8782a}, {"morado", 0x5a2a8a}, {"rosa", 0xe85a9a}, {"turquesa", 0x2aa0a0}, {"marrón", 0x6a4a2a}} {
		sc := sc
		swat("c"+itoa(i), "SWAT · "+sc.n, 3600+i*180, func(s *CharSpec) {
			s.Shirt, s.Shirt2, s.Pants, s.HatCol, s.Accent = hx(sc.col), hx(sc.col), shade(hx(sc.col), 0.6), hx(sc.col), hx(sc.col)
		})
	}
	rng := func(id, name string, price int, fx func(*CharSpec)) { uni("rangers", "rangers:"+id, name, price, fx) }
	for i, sc := range []struct {
		n   string
		col uint32
	}{{"blancos", 0xf4f4f8}, {"azul marino", 0x1a2a5a}, {"plateados", 0xc8ccd8}, {"verdes", 0x2a8a3a}, {"morados", 0x7a3aa8}, {"naranjas", 0xf08a20}, {"turquesa", 0x2aa0a0}} {
		sc := sc
		rng("c"+itoa(i), "Rangers · "+sc.n, 5100+i*150, func(s *CharSpec) { s.Shirt, s.Shirt2, s.HatCol = hx(sc.col), hx(sc.col), hx(sc.col) })
	}
	rng("metal", "Rangers · metalizados", 5200, func(s *CharSpec) { s.Shirt, s.Shirt2, s.HatCol = hx(0xa8aab8), hx(0xa8aab8), hx(0xa8aab8) })
	rng("negros", "Rangers · equipo oscuro", 5600, func(s *CharSpec) { s.Shirt, s.Shirt2, s.HatCol = hx(0x1a1a22), hx(0x1a1a22), hx(0x1a1a22) })
	rng("dorados", "Rangers · edición dorada", 6500, func(s *CharSpec) { s.Shirt, s.Shirt2, s.HatCol = hx(0xe0b830), hx(0xe0b830), hx(0xe0b830) })

	addBaseOutfits() // the two free base outfits (blue for player 1, red for player 2)
	registerTrophyRewards()
	shopRescalePrices() // the single point that sets the real prices (shopprice.go)
	for t := range c {
		for i := range c[t] {
			c[t][i].desc = shopDescribe(t, c[t][i])
		}
	}
	// every tab: cheapest first (the free default stays on top)
	for t := range shopCatalog {
		sort.SliceStable(shopCatalog[t], func(i, j int) bool { return shopCatalog[t][i].price < shopCatalog[t][j].price })
	}
}
