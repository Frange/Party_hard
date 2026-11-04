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
	{"VESTUARIO", []int{shopWardrobe}},
	{"PERSONAJES", []int{shopChars}},
	{"ANIMALES", []int{shopAnimals}},
	{"ASESINOS", []int{shopKillers, shopWeapons}},
	{"POLICÍAS", []int{shopPolice, shopUniforms}},
}

// shopFlat is every list in screen order (left/right walks through it).
var shopFlat = []int{shopOutfits, shopAccess, shopPerks, shopWardrobe, shopChars, shopAnimals, shopKillers, shopWeapons, shopPolice, shopUniforms}

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
	shopCatalog[tab] = append(shopCatalog[tab], shopItem{id: id, name: name, desc: desc, price: price, slot: slot})
}

// buildShopCatalog fills every tab (called once the presets are registered).
func buildShopCatalog() {
	for i := range shopCatalog {
		shopCatalog[i] = nil
	}
	c := &shopCatalog

	// ---- main characters ---------------------------------------------------
	skins := []struct {
		slug  string
		price int
	}{
		{"don_quijote", 1500}, {"sancho_panza", 1200}, {"leeloo", 1400}, {"korben", 1500}, {"conan", 2200}, {"he_man", 2400}, {"marty_mcfly", 1800}, {"doc_brown", 1800}, {"la_mascara", 2600}, {"taz", 2000}, {"burro_shrek", 1200},
		{"michael_jackson", 3200}, {"tortuga_leonardo", 1600}, {"tortuga_donatello", 1600}, {"tortuga_raphael", 1600}, {"tortuga_michelangelo", 1600}, {"alf", 1700}, {"johnny5", 2200},
		{"inspector_gadget", 1400}, {"capitan_planeta", 1800}, {"picard", 2000}, {"johnny_bravo", 1800}, {"pedro_picapiedra", 1000}, {"wilma", 1000}, {"pablo_marmol", 1000}, {"betty", 1000},
		{"sam_sam", 900}, {"cesar", 900}, {"diego", 900}, {"pitufo", 800}, {"papa_pitufo", 900}, {"pitufina", 900}, {"lagarto_juancho", 1000}, {"bob_esponja", 1300}, {"beavis", 1300}, {"butthead", 1300}, {"doraemon", 1500}, {"nobita", 1200}, {"guybrush", 1800}, {"stan_barcos", 1400}, {"elaine_marley", 1500}, {"lechuck", 2200}, {"voodoo_lady", 1500}, {"meathook", 1200}, {"pirata_importante", 900}, {"jack_rey_calabaza", 2000}, {"la_calaca", 1700}, {"capitan_calavera", 1600}, {"esqueleto_rockero", 1400}, {"esqueleto_bailarin", 1400}, {"esqueleto_ejecutivo", 1200},
		{"hippie", 300}, {"skater", 300}, {"emo", 300}, {"punk_cresta", 350}, {"gotico", 350}, {"rapero_cadena", 400}, {"turista_rinonera", 300}, {"surfista", 350},
		{"monitor_zumba", 400}, {"payaso_hosco", 450}, {"yoga", 400}, {"hipster", 350}, {"metalero", 400}, {"rockabilly", 450}, {"fan_eurovision", 450}, {"karaoke_star", 500},
		{"samurai", 500}, {"ninja", 500}, {"pirata", 500}, {"vikingo_casco", 500}, {"mariachi", 500}, {"flamenco", 500}, {"gaitero", 500}, {"cavernicola", 500}, {"romano", 500},
		{"faraon", 550}, {"monje", 450}, {"escoces", 500}, {"sumo", 600}, {"tanguero", 500}, {"gaucho", 500}, {"carnaval_brasil", 600}, {"astronauta", 900}, {"alien_turista", 700},
		{"cocinero_furioso", 450}, {"bombero", 500}, {"medico", 500}, {"taxista", 400}, {"fontanero", 450}, {"jardinero", 400}, {"cura", 500}, {"mago_cumple", 500}, {"cientifico_loco", 700},
		{"wally", 600}, {"mortadelo", 650}, {"filemon", 650}, {"zipi", 600}, {"zape", 600}, {"mafalda", 700}, {"tintin", 700}, {"capitan_haddock", 700}, {"popeye", 700}, {"carpanta", 600},
		{"asterix", 800}, {"obelix", 800}, {"panoramix", 800}, {"super_lopez", 900}, {"espinete", 700}, {"don_pimpon", 700}, {"pantera_rosa", 1000},
		{"homer", 800}, {"bart", 800}, {"marge", 800}, {"lisa", 800}, {"burns", 900}, {"flanders", 800}, {"cartman", 850}, {"kenny", 850}, {"kyle", 850}, {"stan", 850},
		{"fry", 900}, {"leela", 900}, {"bender", 1000}, {"farnsworth", 900}, {"zoidberg", 900}, {"peter", 900}, {"stewie", 900}, {"rick", 1200}, {"morty", 1100},
		{"sonic", 900}, {"tails", 900}, {"knuckles", 900}, {"crash", 900}, {"mario", 1000}, {"luigi", 1000}, {"wario", 1100}, {"waluigi", 1100}, {"donkey_kong", 1100}, {"bowser", 1300},
		{"kirby", 1000}, {"pikachu", 1100}, {"link", 1200}, {"steve_cubos", 1000}, {"creeper", 1000}, {"lara_croft", 1500}, {"nathan_drake", 1500},
		{"luffy", 1100}, {"zoro", 1100}, {"nami", 1100}, {"usopp", 1000}, {"sanji", 1100}, {"chopper", 1000}, {"nico_robin", 1100}, {"franky", 1100}, {"brook", 1100}, {"ace_fuego", 1300},
		{"goku", 1400}, {"vegeta", 1400}, {"krilin", 1000}, {"piccolo", 1200}, {"bulma", 1000}, {"gohan", 1000}, {"naruto", 1300}, {"sailor_moon", 1200}, {"ash_pokemon", 1100},
		{"elvis", 1200}, {"freddie", 1500}, {"marilyn", 1200}, {"charlie_chaplin", 1200}, {"einstein", 1200}, {"napoleon", 1200}, {"cleopatra", 1300}, {"julio_cesar", 1200}, {"dali", 1200},
		{"van_gogh", 1200}, {"frida_kahlo", 1200}, {"mozart", 1100}, {"beethoven", 1100}, {"shakespeare", 1100}, {"colon_almirante", 1100}, {"isabel_la_catolica", 1200}, {"gandhi", 1200},
		{"che", 1200}, {"juana_de_arco", 1300}, {"bruce_lee", 1800}, {"sherlock", 1300}, {"poirot", 1200}, {"zorro", 1500}, {"robin_hood", 1300}, {"peter_pan", 1200}, {"capitan_garfio", 1300}, {"tarzan", 1300},
		{"sp_cantante", 1300}, {"sp_guitarra", 1300}, {"sp_bajo", 1300}, {"sp_bateria", 1300}, {"off_cantante", 1300}, {"off_guitarra", 1300}, {"off_bajo", 1300}, {"off_bateria", 1300}, {"met_cantante", 1500}, {"met_guitarra", 1500}, {"met_bajo", 1500}, {"met_bateria", 1500}, {"ps_cantante", 1200}, {"ps_guitarra", 1200}, {"ps_bajo", 1200}, {"ps_bateria", 1200}, {"bt_cantante", 1300}, {"bt_guitarra", 1300}, {"bt_bajo", 1300}, {"bt_bateria", 1300}, {"lp_cantante", 1300}, {"lp_guitarra", 1300}, {"lp_bajo", 1300}, {"lp_bateria", 1300}, {"des_cantante", 1200}, {"des_guitarra", 1200}, {"des_bajo", 1200}, {"des_bateria", 1200}, {"frank_zummo", 1300}, {"dave_baksh", 1300}, {"jason_mc", 1300}, {"tom_delonge", 1300}, {"lennon", 1300}, {"mccartney", 1300}, {"harrison", 1300}, {"ringo", 1300}, {"deryck", 1300}, {"mark_hoppus", 1300}, {"tom_delonge", 1300}, {"travis_barker", 1400},
		{"woody", 1200}, {"buzz", 1200}, {"shrek", 1500}, {"minion", 1300}, {"gru", 1300}, {"elsa", 1300}, {"blancanieves", 1200}, {"caperucita_roja", 1200}, {"el_nota", 1400}, {"borat", 1500}, {"austin", 1500},
		{"zoolander", 1400}, {"ace_ventura", 1400}, {"mr_bean", 1500}, {"wonka", 1500}, {"beetlejuice", 1600}, {"jack_sparrow", 1800}, {"harry_potter", 1800}, {"hermione", 1700}, {"dumbledore", 1700},
		{"gandalf", 1800}, {"frodo", 1600}, {"legolas", 1800}, {"aragorn", 1800}, {"yoda", 1800}, {"chewbacca", 1800}, {"han_solo", 2000}, {"luke", 1900}, {"leia", 1900}, {"boba_fett", 2200}, {"mandaloriano", 2600},
		{"vader", 2600}, {"trooper", 2000}, {"jedi", 2200}, {"kirk", 1500}, {"spock", 1600}, {"redshirt", 1000}, {"indy", 2000}, {"bond", 2200}, {"dracula", 1500}, {"santa", 1300},
		{"homelander", 2500}, {"michael_scott", 1800}, {"dwight", 1700}, {"sheldon", 1700}, {"barney", 1700}, {"dr_house", 1800}, {"dexter", 2000}, {"heisenberg", 2600}, {"saul", 2000},
		{"jon_nieve", 2200}, {"negan", 2400}, {"miercoles", 2000}, {"once", 1800}, {"jugador_456", 2400}, {"guardia_rosa", 2200},
		{"batman", 3000}, {"superman", 3000}, {"spiderman", 2800}, {"ironman", 3200}, {"thor", 3000}, {"hulk", 3200}, {"capitan_america", 2800}, {"wonder_woman", 2800}, {"flash", 2600}, {"aquaman", 2400},
		{"catwoman", 2600}, {"viuda_negra", 2600}, {"deadpool", 3200}, {"lobezno", 3200}, {"joker", 3000}, {"arlequina", 2800}, {"thanos", 3600}, {"doctor_extrano", 3000}, {"bane", 2800}, {"blade", 2800},
		{"terminator", 3400}, {"robocop", 3400}, {"neo", 3400}, {"john_wick", 3600}, {"rambo", 3000}, {"rocky", 2600}, {"kratos", 3800}, {"jefe_maestro", 3800}, {"solid_snake", 3600}, {"doom_slayer", 4200},
		{"agente_47", 3600}, {"leon_k", 3000}, {"duke", 3000}, {"tony_s", 3200}, {"tommy_s", 3200}, {"atracador_rojo", 2600}, {"motorista_fantasma", 4500}, {"power_ranger", 3800}, {"muerte_vacaciones", 4000},
		{"freddy", 3500}, {"jason", 3500}, {"michael", 3500}, {"ghostface", 3200}, {"pennywise", 4000}, {"demonio", 4500}, {"hombre_cubo", 2500}, {"unicornio_resaca", 3000}, {"dinosaurio_hinchable", 3000},
		{"tiburon_mascota", 2800}, {"pollo_gigante", 2800}, {"hombre_platano", 2600}, {"bebe_gigante", 3200}, {"traje_latas", 2600}, {"cabeza_tele", 3000}, {"hombre_semaforo", 2600}, {"calabaza", 2400},
		{"fantasma_sabana", 2400}, {"esqueleto_fiesta", 2400}, {"zombi_fiesta", 2600}, {"vampiro_vegano", 2600}, {"conejo_siniestro", 3200}, {"pizza_infierno", 3000},
	}
	seen := map[string]bool{}
	for _, sk := range skins {
		if seen[sk.slug] || presetByName(sk.slug) == nil {
			continue
		}
		seen[sk.slug] = true
		n := autoDisplayName(sk.slug)
		c[shopChars] = append(c[shopChars], shopItem{id: "skin:" + sk.slug, name: n, desc: n + " aparecerá en tus fiestas, y mucho más a menudo que el resto de la gente.", price: sk.price, arg: sk.slug})
	}

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

	// ---- killers -----------------------------------------------------------
	for _, k := range []struct {
		slug, name string
		price      int
	}{
		{"mimo", "El Mimo", 700}, {"caballo", "El Caballo", 600}, {"freddy", "Freddy", 800}, {"impermeable", "Impermeable amarillo", 900},
		{"pinhead", "Cabeza de Alfileres", 1000}, {"chucky", "Muñeco diabólico", 1000}, {"leatherface", "Motosierra", 1200},
		{"pennywise", "Payaso bailarín", 1500}, {"demonio", "El Demonio", 2000},
		{"carnicero", "El Carnicero", 900}, {"doctor_loco", "Doctor loco", 1100}, {"enfermera", "Enfermera asesina", 1300}, {"calabaza", "Cabeza de calabaza", 1000},
		{"espantapajaros", "Espantapájaros", 1200}, {"santa_psicopata", "Santa psicópata", 1600}, {"ninja_asesino", "Ninja asesino", 1800}, {"esqueleto", "Esqueleto", 1400},
		{"muneca", "Muñeca poseída", 1700}, {"vampiro", "Vampiro", 2200}, {"bebe_gigante", "Bebé gigante", 2600},
		{"lenador", "Leñador psicópata", 1100}, {"cirujano", "Cirujano", 1300}, {"bruja", "Bruja", 1500}, {"momia", "Momia", 1500}, {"hombre_lobo", "Hombre lobo", 2400},
		{"cocinero_asesino", "Cocinero asesino", 1000}, {"payaso_triste", "Payaso triste", 1200}, {"cazador", "Cazador", 1400}, {"buzo", "Buzo siniestro", 1800},
		{"astronauta_asesino", "Astronauta asesino", 2800}, {"gladiador", "Gladiador", 2000}, {"pirata_fantasma", "Pirata fantasma", 2200}, {"oso_fiesta", "Oso de fiesta", 2500},
		{"novia_cadaver", "Novia cadáver", 2400}, {"chef_frances", "Chef francés", 900}, {"payaso_circo", "Payaso de circo", 1000}, {"vaquero_fantasma", "Vaquero fantasma", 1500}, {"bailarin_ballet", "Bailarina de ballet", 1700}, {"taxista_loco", "Taxista loco", 800}, {"oficinista_zombi", "Oficinista zombi", 1100}, {"arbitro_asesino", "Árbitro asesino", 1200}, {"torero_maldito", "Torero maldito", 2000}, {"jardinero_loco", "Jardinero loco", 1000}, {"karateka", "Karateka siniestro", 1900}, {"domador", "Domador de leones", 1600}, {"dentista", "Dentista", 1200}, {"pescador", "Pescador", 900}, {"brujo", "Brujo oscuro", 2600}, {"caballero_negro", "Caballero negro", 3400}, {"samurai_oscuro", "Samurái oscuro", 3000}, {"marinero_maldito", "Marinero maldito", 1800}, {"disco_vampiro", "Vampiro de discoteca", 2500}, {"hombre_saco", "Hombre del saco", 1400}, {"fantasma_opera", "Fantasma de la ópera", 3300}, {"conejo_asesino", "Conejo asesino", 3000}, {"muerte", "La Muerte", 4000}, {"dj_maldito", "DJ maldito", 3200}, {"robot_asesino", "Robot asesino", 3600},
	} {
		c[shopKillers] = append(c[shopKillers], shopItem{id: "killer:" + k.slug, name: k.name, desc: "Puede aparecer en las fiestas como asesino.", price: k.price, arg: k.slug})
	}
	for _, w := range []struct {
		id, name string
		price    int
		hold     int
	}{{"knife", "Cuchillo de cocina", 300, HoldKnife}, {"baguette", "Baguette dura", 350, HoldBaguette}, {"broom", "Escoba", 400, HoldBroom}, {"plant", "Planta de interior", 420, HoldPlant}, {"beer", "Botella rota", 450, HoldBeer}, {"pizza", "Pizza gigante", 500, HoldPizza}, {"sign", "Pancarta", 550, HoldSign}, {"flag", "Bandera", 600, HoldFlag}, {"vacuum", "Aspiradora", 700, HoldVacuum}, {"megaphone", "Megáfono", 750, HoldMegaphone}, {"selfie", "Palo selfie", 800, HoldSelfie}, {"briefcase", "Maletín", 900, HoldBriefcase}, {"trophy", "Trofeo", 1000, HoldTrophy}, {"saber", "Sable láser", 2500, HoldSaber}, {"guitar", "Guitarra eléctrica", 1500, HoldGuitar}, {"sax", "Saxofón", 1300, HoldSax}, {"machete", "Machete", 500, HoldMachete}, {"torch", "Antorcha", 600, HoldTorch}, {"claw", "Garras", 700, HoldClaw}, {"chainsaw", "Motosierra", 1200, HoldChainsaw}} {
		weaponHold["kw:"+w.id] = w.hold
		c[shopWeapons] = append(c[shopWeapons], shopItem{id: "kw:" + w.id, name: w.name, desc: "Todos los asesinos llevan esta arma. Enter: usar / quitar.", price: w.price, slot: "kweapon"})
	}

	// ---- police: tiers and uniforms ----------------------------------------
	c[shopPolice] = []shopItem{
		{id: "police:0", name: "Policía local", desc: "Siempre disponible: 1 estrella.", price: 0},
		// the other types are not for sale: each one is unlocked by a trophy (kills milestones)
		{id: "police:1", name: "FBI", desc: "Mulder y Scully llegan con 2 estrellas.", price: 1<<20 + 701, arg: "1", trophy: "t_cop1"},
		{id: "police:2", name: "Ejército", desc: "Soldados con 3 estrellas.", price: 1<<20 + 702, arg: "2", trophy: "t_cop2"},
		{id: "police:3", name: "SWAT", desc: "Operadores con 4 estrellas.", price: 1<<20 + 703, arg: "3", trophy: "t_cop3"},
		{id: "police:4", name: "Power Rangers", desc: "Llegan con 5 estrellas.", price: 1<<20 + 704, arg: "4", trophy: "t_cop4"},
	}
	uni := func(slot, id, name string, price int, fx func(*CharSpec)) {
		wearFx[id] = fx
		who := map[string]string{"cop": "la policía local", "fbi": "el FBI", "army": "el ejército", "swat": "el SWAT", "rangers": "los Power Rangers"}[slot]
		c[shopUniforms] = append(c[shopUniforms], shopItem{id: id, name: name, desc: "Cambia el uniforme de " + who + ". Enter: activar / quitar.", price: price, slot: slot})
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
