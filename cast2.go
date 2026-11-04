package main

import "math/rand"

// ---------------------------------------------------------------------------
// More cast: films, series, cartoons and TV. Some can turn up at any party,
// the "shop only" ones have to be bought first (see allowedSlug).
// ---------------------------------------------------------------------------

var castPresets2 = []presetEntry{
	// The Fifth Element
	def("leeloo", fem(), hairc(0xf08a20, HairBob), tops(TopSwim, 0xf4f4f4, 0xf4f4f4), pantsc(0xf4f4f4), suspc(0xf08a20), shoesc(0xf4f4f4), bodyc(BodySlim), stat(1)),
	def("korben", hairc(0xe8c860, HairShort), tops(TopTank, 0xf08a20, 0xf08a20), pantsc(0xa89a70), shoesc(0x3a2a1a), bodyc(BodyBig), stat(-1)),
	// Conan, He-Man
	def("conan", hairc(0x14101a, HairLong), tops(TopBare, 0xd0a070, 0xd0a070), skinc(0xd0a070), bottomc(BotShorts, 0x6a4a2a), shoesc(0x5a3a1a), bodyc(BodyGiant), stat(-3), weaponc(HoldMachete)),
	def("he_man", hairc(0xe8d060, HairLong), tops(TopBare, 0xe8b890, 0xe8b890), skinc(0xe8b890), bottomc(BotShorts, 0x6a4a2a), shoesc(0x6a3a1a), suspc(0x2a2a30), capec(0x2a8a4a), bodyc(BodyGiant), stat(-3), weaponc(HoldMachete)),
	// Back to the Future
	def("marty_mcfly", hairc(0x5a3a22, HairShort), tops(TopLong, 0xe86a20, 0xe86a20), pantsc(0x3a5a9a), shoesc(0xf4f4f4), glass(GlassShades)),
	def("doc_brown", hairc(0xf4f4f4, HairMessy), tops(TopSuit, 0xf4f4f0, 0xd8d8e0), pantsc(0x3a3a46), glass(GlassBig), bodyc(BodySlim), stat(-1)),
	// The Mask, Taz, Donkey
	def("la_mascara", skinc(0x60c040), hairc(0x14101a, HairComb), hatc(HatFedora, 0xf0d020), tops(TopSuit, 0xf0d020, 0xf0d020), pantsc(0xf0d020), accentc(0xc82a2a), shoesc(0x14141a), headc(HeadBig)),
	def("taz", skinc(0x6a4020), hairc(0x2a1a10, HairSpiky), tops(TopBare, 0x6a4020, 0x6a4020), bottomc(BotShorts, 0x6a4020), shoesc(0x6a4020), bodyc(BodyFat), headc(HeadBig), stat(3)),
	def("burro_shrek", skinc(0x9a9aa6), hairc(0x6a6a76, HairMessy), earsn(3), tops(TopLong, 0x9a9aa6, 0x9a9aa6), pantsc(0x8a8a96), shoesc(0x4a4a56), headc(HeadBig), stat(3)),
	// Michael Jackson
	def("michael_jackson", hairc(0x14101a, HairCurly), hatc(HatFedora, 0x14141a), tops(TopSuit, 0x14141a, 0xf4f4f4), pantsc(0x14141a), shoesc(0xf4f4f4), accentc(0xf4f4f4), bodyc(BodySlim), stat(-1)),
	// Ninja Turtles
	def("tortuga_leonardo", skinc(0x5ab040), hatc(HatBeanie, 0x2a5ac8), tops(TopBare, 0x5ab040, 0x5ab040), bottomc(BotShorts, 0x5ab040), shoesc(0x5ab040), capec(0x8a5a2a), bodyc(BodyBig), weaponc(HoldMachete), stat(1)),
	def("tortuga_donatello", skinc(0x5ab040), hatc(HatBeanie, 0x7a2aa8), tops(TopBare, 0x5ab040, 0x5ab040), bottomc(BotShorts, 0x5ab040), shoesc(0x5ab040), capec(0x8a5a2a), bodyc(BodyBig), carryc(HoldBroom), stat(1)),
	def("tortuga_raphael", skinc(0x5ab040), hatc(HatBeanie, 0xd02828), tops(TopBare, 0x5ab040, 0x5ab040), bottomc(BotShorts, 0x5ab040), shoesc(0x5ab040), capec(0x8a5a2a), bodyc(BodyBig), weaponc(HoldKnife), stat(1)),
	def("tortuga_michelangelo", skinc(0x5ab040), hatc(HatBeanie, 0xf08a20), tops(TopBare, 0x5ab040, 0x5ab040), bottomc(BotShorts, 0x5ab040), shoesc(0x5ab040), capec(0x8a5a2a), bodyc(BodyBig), stat(1)),
	// Ghostbusters
	def("gb_venkman", hairc(0x3a2a1a, HairComb), tops(TopLong, 0x8a8a6a, 0x8a8a6a), pantsc(0x8a8a6a), capec(0x4a4a52), shoesc(0x3a2a1a)),
	def("gb_stantz", hairc(0x5a3a22, HairShort), glass(GlassRound), tops(TopLong, 0x8a8a6a, 0x8a8a6a), pantsc(0x8a8a6a), capec(0x4a4a52), shoesc(0x3a2a1a), bodyc(BodyBig)),
	def("gb_spengler", hairc(0x2a1a10, HairBowl), glass(GlassBig), tops(TopLong, 0x8a8a6a, 0x8a8a6a), pantsc(0x8a8a6a), capec(0x4a4a52), shoesc(0x3a2a1a), bodyc(BodySlim)),
	def("gb_winston", hairs(HairShort), skinc(0x8a5a3a), hatc(HatCap, 0x8a8a6a), tops(TopLong, 0x8a8a6a, 0x8a8a6a), pantsc(0x8a8a6a), capec(0x4a4a52), shoesc(0x3a2a1a)),
	// 80s TV
	def("alf", skinc(0xb08050), hairc(0x8a6030, HairMessy), earsn(2), headc(HeadBig), tops(TopLong, 0xa87a48, 0xa87a48), pantsc(0x8a6a3a), shoesc(0x5a3a1a), stat(3)),
	def("johnny5", skinc(0xb0b4bc), hairs(HairBald), hatc(HatAntenna, 0x8a8e98), glass(GlassBig), bodyc(BodyRobot), shoesc(0x4a4e58), headc(HeadBig)),
	def("inspector_gadget", hatc(HatFedora, 0x6a7a8a), beardc(BeardStubble), tops(TopRobe, 0x7a8a9a, 0x7a8a9a), pantsc(0x4a5a6a), shoesc(0x3a3a46), bodyc(BodyBig), stat(-1), earsn(1)),
	def("capitan_planeta", skinc(0x5aa8e0), hairc(0x3aa040, HairLong), tops(TopLong, 0xf4f4f4, 0xf4f4f4), pantsc(0xf4f4f4), accentc(0xd02828), shoesc(0xd02828), capec(0xd02828), bodyc(BodyBig), stat(-1)),
	// Star Trek
	def("picard", hairs(HairBald), tops(TopUniform, 0xb82828, 0xe8c840), accentc(0x14141a), pantsc(0x14141a), shoesc(0x14141a), bodyc(BodyNormal), stat(-1)),
	// cartoons
	def("johnny_bravo", hairc(0xf0d848, HairPompadour), glass(GlassShades), tops(TopTank, 0x14141a, 0x14141a), pantsc(0x3a5a9a), shoesc(0x14141a), bodyc(BodyGiant), stat(-3)),
	def("pedro_picapiedra", hairc(0x14101a, HairShort), tops(TopTee, 0xf08a20, 0x14141a), patn(3), pantsc(0xf08a20), shoesc(0xc8a070), accentc(0x2a4ab8), bodyc(BodyFat), stat(-1)),
	def("wilma", fem(), hairc(0xe86a20, HairBun), tops(TopDress, 0xf4f4f0, 0xf4f4f0), shoesc(0xc8a070)),
	def("pablo_marmol", hairc(0xe8d060, HairShort), tops(TopTee, 0x6a4a2a, 0x6a4a2a), pantsc(0x6a4a2a), shoesc(0xc8a070), bodyc(BodyBig), stat(3)),
	def("betty", fem(), hairc(0x14101a, HairBob), tops(TopDress, 0x3a8aca, 0x3a8aca), shoesc(0xc8a070), stat(2)),
	def("pitufo", skinc(0x3a8ae0), hairs(HairBald), hatc(HatBeanie, 0xf4f4f4), tops(TopTee, 0x3a8ae0, 0x3a8ae0), pantsc(0xf4f4f4), shoesc(0xf4f4f4), headc(HeadBig), stat(4), bodyc(BodyBig)),
	def("papa_pitufo", skinc(0x3a8ae0), hatc(HatBeanie, 0xd02828), beardc(BeardFull), hairc(0xf4f4f4, HairShort), tops(TopTee, 0xd02828, 0xd02828), pantsc(0xd02828), shoesc(0xd02828), headc(HeadBig), stat(4), bodyc(BodyBig)),
	def("pitufina", fem(), skinc(0x3a8ae0), hatc(HatBeanie, 0xf4f4f4), hairc(0xe8d060, HairLong), tops(TopDress, 0xf4f4f4, 0xf4f4f4), shoesc(0xf4f4f4), headc(HeadBig), stat(4)),
	def("lagarto_juancho", skinc(0x6ab048), hairs(HairBald), hatc(HatCap, 0xd02828), tops(TopHawaiian, 0xf0d020, 0xd02828), bottomc(BotShorts, 0x3a5ab8), shoesc(0x14141a), headc(HeadBig), stat(2)),
	def("bob_esponja", skinc(0xf0e040), hairs(HairBald), glass(GlassBig), tops(TopTee, 0xf4f4f4, 0xf4f4f4), accentc(0xd02828), pantsc(0x8a6a2a), shoesc(0x14141a), bodyc(BodyBig), headc(HeadBig), stat(3)),
	def("beavis", hairc(0xe8d860, HairPompadour), tops(TopTee, 0x14141a, 0x14141a), bottomc(BotShorts, 0x3a5ab8), shoesc(0xf4f4f4), stat(0), bodyc(BodySlim)),
	def("butthead", hairc(0x4a3018, HairMessy), tops(TopTee, 0x14141a, 0x8a8a94), pantsc(0x3a5a9a), shoesc(0x14141a), bodyc(BodyBig), stat(-1)),
	def("doraemon", skinc(0x2a8ad8), hairs(HairBald), glass(GlassWhite), tops(TopTee, 0x2a8ad8, 0xf4f4f4), accentc(0xd02828), pantsc(0x2a8ad8), shoesc(0xf4f4f4), headc(HeadHuge), bodyc(BodyFat), stat(5)),
	def("nobita", hairc(0x14101a, HairShort), glass(GlassRound), tops(TopTee, 0xf0d020, 0xf0d020), bottomc(BotShorts, 0x3a5ab8), shoesc(0xf4f4f4), headc(HeadBig), stat(4)),
}

// shopOnly: these characters only turn up once they have been bought.
var shopOnly = map[string]bool{
	"leeloo": true, "korben": true, "conan": true, "he_man": true, "marty_mcfly": true, "doc_brown": true, "la_mascara": true, "taz": true,
	"michael_jackson": true, "tortuga_leonardo": true, "tortuga_donatello": true, "tortuga_raphael": true, "tortuga_michelangelo": true,
	"alf": true, "johnny5": true, "picard": true, "capitan_planeta": true, "johnny_bravo": true,
}

// allowedSlug: may this character appear without being bought?
func allowedSlug(slug string) bool { return !shopOnly[slug] || ownedID("skin:"+slug) }

func init() {
	famousPresets = append(famousPresets, asFuncs(castPresets2)...)
	for k, v := range map[string]string{
		"leeloo": "Leeloo", "korben": "Korben Dallas", "conan": "Conan el Bárbaro", "he_man": "He-Man", "marty_mcfly": "Marty McFly", "doc_brown": "Dr. Emmett Brown",
		"la_mascara": "La Máscara", "taz": "El Diablo de Tasmania", "burro_shrek": "Burro", "michael_jackson": "Michael Jackson",
		"tortuga_leonardo": "Leonardo", "tortuga_donatello": "Donatello", "tortuga_raphael": "Rafael", "tortuga_michelangelo": "Miguel Ángel",
		"gb_venkman": "Cazafantasmas Peter", "gb_stantz": "Cazafantasmas Ray", "gb_spengler": "Cazafantasmas Egon", "gb_winston": "Cazafantasmas Winston",
		"alf": "ALF", "johnny5": "Cortocircuito", "inspector_gadget": "Inspector Gadget", "capitan_planeta": "Capitán Planeta", "picard": "Capitán Picard",
		"johnny_bravo": "Johnny Bravo", "pedro_picapiedra": "Pedro Picapiedra", "wilma": "Wilma", "pablo_marmol": "Pablo Mármol", "betty": "Betty",
		"pitufo": "Pitufo", "papa_pitufo": "Papá Pitufo", "pitufina": "Pitufina", "lagarto_juancho": "Lagarto Juancho", "bob_esponja": "Bob Esponja",
		"beavis": "Beavis", "butthead": "Butt-Head", "doraemon": "Doraemon", "nobita": "Nobita",
	} {
		displayNameOverrides[k] = v
	}
	starGroups = append(starGroups,
		[]string{"tortuga_leonardo", "tortuga_donatello", "tortuga_raphael", "tortuga_michelangelo"},
		[]string{"pedro_picapiedra", "wilma", "pablo_marmol", "betty"},
		[]string{"papa_pitufo", "pitufo", "pitufo", "pitufina"},
		[]string{"beavis", "butthead"},
		[]string{"doraemon", "nobita"},
		[]string{"marty_mcfly", "doc_brown"},
		[]string{"leeloo", "korben"},
	)
}

// specialPool: characters that turn up on their own, a few in every party.
var specialPool = []string{
	"yoda", "vader", "indy", "et", "capitan_america", "terminator", "robocop", "batman", "superman", "spiderman", "hulk", "goku", "mario", "sonic", "shrek", "minion",
	"jack_sparrow", "elvis", "freddie", "homer", "bart", "bender", "fry", "cartman", "luffy", "mortadelo", "filemon", "asterix", "obelix", "wally", "pantera_rosa", "espinete",
	"conan", "he_man", "marty_mcfly", "doc_brown", "la_mascara", "taz", "burro_shrek", "michael_jackson", "alf", "johnny5", "inspector_gadget", "capitan_planeta", "picard",
	"johnny_bravo", "lagarto_juancho", "bob_esponja", "doraemon", "nobita", "pitufo", "papa_pitufo", "pitufina", "pedro_picapiedra", "wilma", "beavis", "butthead", "kirk", "spock",
}

// quirkOf: characters with their own mad behaviour.
func quirkOf(slug string) string {
	if q, ok := monkeyQuirk[slug]; ok {
		return q
	}
	if q, ok := skelQuirk[slug]; ok {
		return q
	}
	switch slug {
	case "michael_jackson":
		return "mj"
	case "la_mascara":
		return "mask"
	case "taz":
		return "taz"
	case "beavis":
		return "beavis"
	case "butthead":
		return "butthead"
	case "sam_sam", "cesar", "diego":
		return "kid"
	}
	return ""
}

var quirkLines = map[string][]string{
	"mj":       {"¡Hee-hee!", "¡Shamone!", "¡Auuu!", "¡Moonwalk!"},
	"mask":     {"¡SSSMOKIN'!", "¡Fiuuu!", "¡Me gusta!", "¡Estoy de lujo!", "¡Alegría!"},
	"taz":      {"¡Grrrr!", "¡Aaarrgh!", "¡Rrrr!"},
	"beavis":   {"¡Heh heh!", "¡Soy Cornholio!", "¡Necesito papel higiénico!", "¡Fuego! ¡Fuego!", "¡Heh heh, dijo 'mojar'!"},
	"butthead": {"Huh huh... mola.", "Huh huh huh, dijo 'culo'.", "Eres idiota, Beavis.", "Esto es un rollo.", "Huh huh, qué asco."},
}

var quirkAnims = map[string][]int{
	"mj":       {AnimDance10, AnimPirouette, AnimLean, AnimDance10, AnimDance4, AnimLean},
	"mask":     {AnimDance3, AnimPirouette, AnimLaugh, AnimDance7, AnimPanic, AnimDance9, AnimLove, AnimDance5},
	"taz":      {AnimPirouette, AnimPirouette, AnimPanic},
	"beavis":   {AnimLaugh, AnimDance6, AnimLaugh},
	"butthead": {AnimLaugh, AnimTalk},
}

// quirkPose is the animation of a guest with a quirk while standing around.
func (g *Game) quirkPose(gu *Guest) (int, float64, bool) {
	list := quirkAnims[gu.quirk]
	if len(list) == 0 {
		return 0, 0, false
	}
	slot := int(g.time/2.2 + gu.off)
	a := list[slot%len(list)]
	speed := 7.0
	if a == AnimPirouette {
		speed = 10
	}
	return a, g.time*speed + gu.off, true
}

// updateQuirk: things they shout, and the Lean needs him to face sideways.
func (g *Game) updateQuirk(gu *Guest, dt float64) {
	if gu.quirk == "kid" {
		g.updateKid(gu)
	}
	gu.quirkT -= dt
	if gu.quirkT <= 0 {
		gu.quirkT = 4 + g.rng.Float64()*6
		if l := quirkLines[gu.quirk]; len(l) > 0 && gu.icon == "" && !gu.moving {
			gu.icon, gu.iconT = l[g.rng.Intn(len(l))], 2.2
		}
	}
	if gu.quirk == "mj" && !gu.moving {
		if list := quirkAnims[gu.quirk]; list[int(g.time/2.2+gu.off)%len(list)] == AnimLean {
			gu.dir = DirRight
		}
	}
}

func _unusedRand(r *rand.Rand) {}
