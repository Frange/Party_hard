package main

import "math/rand"

// ---------------------------------------------------------------------------
// More cast: comics, anime, pirates, Gauls, history, fantasy and underwear.
// ---------------------------------------------------------------------------

var morePresets = []presetEntry{
	// Where's Wally?
	def("wally", hatc(HatBeanie, 0xd02828), hairc(0x1e1810, HairShort), tops(TopTee, 0xd8342c, 0xf6f0e0), pantsc(0x4a6ab0), shoesc(0x6a4a2a), stat(1)),
	def("wenda", fem(), hatc(HatBeanie, 0xd02828), glass(GlassRound), hairc(0x3a2414, HairBob), tops(TopTee, 0xd02828, 0xf4f4f4), patn(2), bottomc(BotSkirt, 0x3a5a9a), shoesc(0x6a4a2a)),
	def("odlaw", hatc(HatBowler, 0x14141a), glass(GlassShades), beardc(BeardMustache), hairc(0x14141a, HairShort), tops(TopTee, 0x14141a, 0xe8d020), patn(2), pantsc(0x14141a), shoesc(0x14141a), stat(1)),
	def("mago_blancabarba", hatc(HatWizard, 0xd02828), beardc(BeardFull), hairc(0xf4f4f4, HairLong), tops(TopRobe, 0xd02828, 0xf4f4f4), capec(0xd02828), stat(2)),
	// Spanish comics
	def("mortadelo", hairs(HairBald), glass(GlassRound), beardc(BeardMustache), tops(TopSuit, 0x3a4a2a, 0xe8e0c0), accentc(0xc82a2a), pantsc(0x3a4a2a), bodyc(BodySlim), stat(-3)),
	def("filemon", hairs(HairBald), beardc(BeardMustache), tops(TopSuit, 0x2a2a34, 0xf4f4f4), accentc(0xc82a2a), pantsc(0x2a2a34), bodyc(BodyBig), stat(1)),
	def("super_lopez", hairc(0x2a2a30, HairComb), tops(TopLong, 0x2a50c0, 0x2a50c0), pantsc(0x2a50c0), capec(0xc82a2a), shoesc(0xc82a2a), bodyc(BodyBig), glass(GlassRound)),
	def("zipi", hairc(0xf0c848, HairShort), tops(TopTee, 0xc82a2a, 0xc82a2a), pantsc(0x3a4a8a), stat(3), headc(HeadBig)),
	def("zape", hairc(0x16121a, HairBowl), tops(TopTee, 0x3a8a4a, 0x3a8a4a), pantsc(0x5a4a3a), stat(3), headc(HeadBig)),
	def("rompetechos", hairs(HairBald), glass(GlassBig), beardc(BeardMustache), tops(TopSuit, 0x6a5a3a, 0xf0f0e0), pantsc(0x6a5a3a), bodyc(BodySlim), stat(-1)),
	def("carpanta", hatc(HatBowler, 0x4a3a2a), beardc(BeardStubble), tops(TopLong, 0x8a7a5a, 0x8a7a5a), pantsc(0x4a4a3a), bodyc(BodySlim)),
	def("mafalda", fem(), hairc(0x14101a, HairBob), tops(TopDress, 0xd83030, 0xd83030), shoesc(0x14141a), stat(5), headc(HeadBig)),
	def("tintin", hairc(0xd8a030, HairQuiff), tops(TopLong, 0x3a5ab8, 0x3a5ab8), bottomc(BotShorts, 0xd8c890), shoesc(0x6a4a2a), stat(2)),
	def("capitan_haddock", hatc(HatCap, 0x1a2a5a), beardc(BeardFull), hairc(0x2a2020, HairShort), tops(TopLong, 0x1a2a5a, 0x1a2a5a), pantsc(0x1a2a5a), bodyc(BodyBig)),
	def("popeye", hatc(HatCap, 0xf4f4f8), beardc(BeardStubble), tops(TopTee, 0x1a1a22, 0xc82a2a), pantsc(0x2a4ab8), bodyc(BodyBig), skinc(0xe8b48a)),
	def("bluto", beardc(BeardFull), hairc(0x14101a, HairShort), tops(TopTee, 0x2a2a34, 0x2a2a34), pantsc(0x8a2a2a), bodyc(BodyGiant), stat(-2)),
	// One Piece
	def("luffy", hatc(HatCowboy, 0xe8c458), hairc(0x2a1a14, HairMessy), tops(TopVest, 0xe0442a, 0xf0d040), bottomc(BotShorts, 0x2a68b4), accentc(0xf0c030), shoesc(0x7a5030), bodyc(BodySlim), headc(HeadBig)),
	def("zoro", hairc(0x58b04a, HairSpiky), tops(TopVest, 0xd8c8a0, 0xd8c8a0), pantsc(0x242838), accentc(0x3a8a3a), shoesc(0x14141a), bodyc(BodyBig), stat(-1)),
	def("nami", fem(), hairc(0xf07a28, HairPony), tops(TopTee, 0x3a8ae0, 0xf4f4f4), patn(2), bottomc(BotSkirt, 0xf0802a), shoesc(0x8a5a2a)),
	def("usopp", hairc(0x14101a, HairCurly), glass(GlassBig), tops(TopLong, 0xc89a40, 0xc89a40), pantsc(0xc89a40), suspc(0x3a5ab8), bodyc(BodySlim), stat(-1)),
	def("sanji", hairc(0xf2cc50, HairBob), beardc(BeardNone), tops(TopSuit, 0x1c2038, 0x3a8ad0), pantsc(0x1c2038), accentc(0x3a8ad0), bodyc(BodySlim), stat(-1)),
	def("chopper", skinc(0x9a6a44), hatc(HatBeanie, 0xf47aa0), tops(TopBare, 0x9a6a44, 0x9a6a44), bottomc(BotShorts, 0x4a60b0), stat(6), headc(HeadBig)),
	def("nico_robin", fem(), hairc(0x14101a, HairLong), tops(TopLong, 0x7a3aa8, 0x7a3aa8), pantsc(0x4a2a6a), stat(-1), bodyc(BodySlim)),
	def("franky", hairc(0x40c0f0, HairPompadour), glass(GlassShades), tops(TopHawaiian, 0xf04040, 0xf8f8f0), patn(1), bottomc(BotTrunks, 0x2a60c8), bodyc(BodyGiant), stat(-2), skinc(0xe8b48a)),
	def("brook", maskc(0xf0f0e0, MaskSkull), tops(TopSuit, 0x14141a, 0xf4f4f4), pantsc(0x14141a), bodyc(BodySlim), stat(-3)),
	def("ace_fuego", hairc(0x14101a, HairShort), hatc(HatCowboy, 0xe8802a), tops(TopBare, 0xe8b48a, 0xe8b48a), bottomc(BotShorts, 0x2a50c0), skinc(0xe8b48a), bodyc(BodyBig)),
	// Asterix
	def("asterix", hairc(0xe8c040, HairShort), tops(TopTee, 0x2c2e60, 0x2c2e60), pantsc(0xd8452a), shoesc(0x6a4a2a), stat(2)),
	def("obelix", hairc(0xd05020, HairShort), tops(TopBare, 0xe8b48a, 0xe8b48a), skinc(0xe8b48a), pantsc(0x62a8e0), bodyc(BodyGiant), stat(-2)),
	def("panoramix", hatc(HatNone, 0xf0f0f0), beardc(BeardFull), hairc(0xf4f4f4, HairLong), tops(TopRobe, 0xf4f4f4, 0xf4f4f4), capec(0xc82828)),
	def("abraracurcix", hatc(HatHorns, 0), hairc(0xd05020, HairShort), beardc(BeardFull), tops(TopVest, 0x8a5a2a, 0x8a5a2a), pantsc(0x3a5ab8), bodyc(BodyFat)),
	def("legionario_romano", hatc(HatHelmet, 0xc8a040), tops(TopArmor, 0xb82828, 0xc8a040), pantsc(0xb82828), shoesc(0x6a4a2a), bodyc(BodyBig)),
	def("julio_cesar", hatc(HatCrown, 0x3a8a3a), hairs(HairBald), tops(TopRobe, 0xf0ece0, 0xf0ece0), capec(0xb01830), shoesc(0x8a5a2a)),
	def("cleopatra", fem(), hairc(0x14101a, HairBob), hatc(HatCrown, 0xe8c040), tops(TopDress, 0xf4f0e0, 0xe8c040), patn(3), shoesc(0xe8c040)),
	// history, art, science
	def("napoleon", hatc(HatBowler, 0x1a2a5a), tops(TopUniform, 0x2a3a8a, 0xf4f4f4), accentc(0xe8c040), pantsc(0xf4f4f4), shoesc(0x14141a), stat(2)),
	def("einstein", hairc(0xf4f4f4, HairMessy), beardc(BeardMustache), tops(TopLong, 0x6a6a74, 0x6a6a74), pantsc(0x4a4a52)),
	def("van_gogh", hairc(0xd05a20, HairShort), beardc(BeardFull), hatc(HatBucket, 0xe8c040), tops(TopLong, 0x3a5a9a, 0x3a5a9a), pantsc(0x6a5a3a)),
	def("frida_kahlo", fem(), hairc(0x14101a, HairBuns), tops(TopDress, 0x3aa0a0, 0xe8c040), patn(3), shoesc(0x8a2a2a)),
	def("dali", hairc(0x14101a, HairComb), beardc(BeardMustache), tops(TopSuit, 0x2a2a34, 0xf4f4f4), accentc(0xc82a2a), pantsc(0x2a2a34)),
	def("charlie_chaplin", hatc(HatBowler, 0x14141a), beardc(BeardMustache), tops(TopSuit, 0x14141a, 0xf4f4f4), pantsc(0x14141a), shoesc(0x14141a), stat(2), bodyc(BodySlim)),
	def("marilyn", fem(), hairc(0xf0d878, HairCurly), tops(TopDress, 0xf4f4f0, 0xf4f4f0), shoesc(0xd8b040), bodyc(BodyNormal)),
	def("mozart", hatc(HatWig, 0xf4f4f4), tops(TopSuit, 0x3a5a9a, 0xf4f4f0), accentc(0xe8c040), pantsc(0xf4f4f0), shoesc(0x14141a)),
	def("beethoven", hairc(0xb8b8c0, HairMessy), tops(TopLong, 0x3a2a2a, 0x3a2a2a), pantsc(0x2a2a2a)),
	def("shakespeare", hairc(0x5a4a3a, HairComb), beardc(BeardGoatee), tops(TopLong, 0x8a2a2a, 0x8a2a2a), pantsc(0x2a2a2a)),
	def("colon_almirante", hatc(HatBowler, 0x6a2a2a), hairc(0xb8b8c0, HairLong), tops(TopRobe, 0xb02828, 0xe8c040), shoesc(0x3a2a1a), beardc(BeardStubble)),
	def("isabel_la_catolica", fem(), hatc(HatCrown, 0xe8c040), hairc(0x8a5a2a, HairBun), tops(TopDress, 0x7a1a3a, 0xe8c040), patn(3)),
	def("gandhi", hairs(HairBald), glass(GlassRound), tops(TopRobe, 0xf4f0e8, 0xf4f0e8), skinc(0xc98a5e), bodyc(BodySlim)),
	def("che", hatc(HatBeret, 0x14141a), hairc(0x2a2020, HairLong), beardc(BeardStubble), tops(TopLong, 0x4a5a3a, 0x4a5a3a), pantsc(0x4a5a3a)),
	def("sherlock", hatc(HatNone, 0x8a7a5a), tops(TopRobe, 0x6a5a3a, 0x6a5a3a), pantsc(0x3a3a3a), hairc(0x2a2020, HairShort), carryc(HoldNone), bodyc(BodySlim), stat(-1)),
	def("poirot", hatc(HatBowler, 0x14141a), beardc(BeardMustache), tops(TopSuit, 0x2a2a34, 0xf4f4f0), pantsc(0x2a2a34), stat(3), bodyc(BodyBig)),
	def("zorro", hatc(HatSombrero, 0x14141a), skinc(0xe0b088), hairc(0x14101a, HairShort), beardc(BeardMustache), tops(TopRobe, 0x14141a, 0x14141a), capec(0x14141a)),
	def("robin_hood", hatc(HatFedora, 0x2a8a3a), hairc(0x8a5a2a, HairShort), beardc(BeardStubble), tops(TopTee, 0x3a9a4a, 0x3a9a4a), pantsc(0x6a4a2a), shoesc(0x4a3a2a)),
	def("peter_pan", hatc(HatNone, 0x2a8a3a), hairc(0xd05a20, HairShort), tops(TopTee, 0x3a9a4a, 0x3a9a4a), pantsc(0x4a7a3a), shoesc(0x4a3a2a), stat(1), bodyc(BodySlim)),
	def("capitan_garfio", hatc(HatPirate, 0xb02828), hairc(0x14101a, HairLong), beardc(BeardMustache), tops(TopLong, 0xb02828, 0xf4f4f0), pantsc(0x2a2a34), shoesc(0x14141a), stat(-1), carryc(HoldClaw)),
	def("tarzan", hairc(0x3a2a1a, HairLong), tops(TopBare, 0xc89a6e, 0xc89a6e), bottomc(BotTrunks, 0x8a5a2a), skinc(0xc89a6e), bodyc(BodyBig), shoesc(0xc89a6e)),
	def("jane_selva", fem(), hairc(0x5a3a1a, HairPony), tops(TopTank, 0xc89a40, 0xc89a40), bottomc(BotShorts, 0x8a5a2a), shoesc(0x6a4a2a)),
	def("juana_de_arco", fem(), hatc(HatHelmet, 0xc8c8d0), hairc(0x8a5a2a, HairBob), tops(TopArmor, 0xc8c8d0, 0x3a5ab8), pantsc(0x3a5ab8)),
	def("bruce_lee", hairc(0x14101a, HairShort), tops(TopLong, 0xf0d020, 0xf0d020), pantsc(0xf0d020), accentc(0x14141a), shoesc(0x14141a), bodyc(BodySlim), skinc(0xf0d0a0)),
	// superheroes and anime
	def("wonder_woman", fem(), hatc(HatCrown, 0xe8c040), hairc(0x14101a, HairLong), tops(TopSwim, 0xc82828, 0xe8c040), bottomc(BotShorts, 0x2a3a9a), shoesc(0xc82828), capec(0xc82828), stat(-1)),
	def("catwoman", fem(), skinc(0xf0d0c0), maskc(0x14141a, MaskPlain), tops(TopLong, 0x14141a, 0x14141a), pantsc(0x14141a), shoesc(0x14141a), bodyc(BodySlim)),
	def("flash", skinc(0xe8c0a0), maskc(0xc82828, MaskPlain), tops(TopLong, 0xc82828, 0xe8c040), pantsc(0xc82828), shoesc(0xe8c040), bodyc(BodyNormal), stat(-1)),        // bolt emblem and ear wings: charimp_a.go
	def("capitan_america", skinc(0xe8c0a0), maskc(0x2a4ab0, MaskPlain), tops(TopLong, 0x2a4ab0, 0x2a4ab0), pantsc(0x2a4ab0), shoesc(0xc82828), bodyc(BodyBig), stat(-1)), // star, stripes, shield: charimp_a.go
	def("viuda_negra", fem(), hairc(0xc83a22, HairPony), tops(TopLong, 0x14141a, 0x14141a), pantsc(0x14141a), shoesc(0x14141a), bodyc(BodySlim)),
	def("aquaman", hairc(0xe8c040, HairSide), beardc(BeardStubble), tops(TopLong, 0xe07a1a, 0xa85a10), patn(3), pantsc(0x2a8a4a), shoesc(0x2a8a4a), bodyc(BodyGiant), stat(-2)), // orange scales, green legs
	def("goku", skinc(0xf4c8a0), hairc(0x14101a, HairSpiky), tops(TopLong, 0xf08020, 0x2a50c0), pantsc(0xf08020), accentc(0x2a50c0), shoesc(0x2a50c0), bodyc(BodyBig)),
	def("vegeta", skinc(0xf0c098), hairc(0x14101a, HairShort), tops(TopArmor, 0x1e2a78, 0xf4f4f4), pantsc(0x1e2a78), shoesc(0xf4f4f4), stat(1), bodyc(BodyBig)),
	def("naruto", hairc(0xf0d030, HairSpiky), tops(TopHoodie, 0xf08020, 0x2a2a34), pantsc(0xf08020), shoesc(0x2a3a8a), stat(1)),
	def("sailor_moon", fem(), hairc(0xf0d878, HairBuns), tops(TopTee, 0xf4f4f8, 0xf4f4f8), patn(0), bottomc(BotSkirt, 0x2a50c0), shoesc(0xc82828), bodyc(BodySlim)),
	def("ash_pokemon", hatc(HatCap, 0xc82828), hairc(0x14101a, HairShort), tops(TopHoodie, 0x2a50c0, 0xf4f4f4), pantsc(0x3a4a6a), shoesc(0xc82828), stat(1)),
	// Disney, games, fantasy
	def("woody", hatc(HatCowboy, 0x8a5428), hairc(0x5a3a1a, HairShort), tops(TopLong, 0xe8c040, 0xd8642a), patn(3), pantsc(0x3a5ab8), shoesc(0x6a3a1a), stat(1)),
	def("buzz", skinc(0xf0c8a8), hairc(0x7a4ab0, HairShort), tops(TopLong, 0xf4f4f8, 0x7a3aa8), pantsc(0xf4f4f8), bodyc(BodyBig), stat(-1), shoesc(0x3aa040)),
	def("elsa", fem(), hairc(0xf4f0dc, HairLong), tops(TopDress, 0x8ac8f0, 0xf4f4ff), patn(3), capec(0xa8d8f8), shoesc(0x8ac8f0), bodyc(BodySlim), stat(-1)),
	def("blancanieves", fem(), skinc(0xf6dcd0), hairc(0x14101a, HairBob), tops(TopDress, 0xf0d040, 0xf0d040), shoesc(0xc82828)), // blue bodice, bow: charimp_a.go
	def("caperucita_roja", fem(), hairc(0x5a3a1a, HairPony), tops(TopDress, 0xf4f4f0, 0xc82828), capec(0xc82828), shoesc(0x4a2a1a), stat(1), carryc(HoldPlant)),
	def("link", hatc(HatNone, 0x3a9a4a), hairc(0xe8c870, HairShort), tops(TopLong, 0x3a9a4a, 0x3a9a4a), pantsc(0xe8e0c0), shoesc(0x6a4a2a), stat(1)),
	def("gandalf", hatc(HatWizard, 0x8a8a96), beardc(BeardFull), hairc(0xc8c8d0, HairLong), tops(TopRobe, 0x8a8a96, 0x8a8a96), bodyc(BodyBig), stat(-2)),
	def("frodo", hairc(0x5a3a1a, HairCurly), tops(TopVest, 0x4a6a3a, 0xf4f0e0), pantsc(0x6a5a3a), capec(0x3a6a3a), stat(5), shoesc(0xc89a6e)),
	def("harry_potter", skinc(0xf0c8a8), glass(GlassRound), hairc(0x14101a, HairMessy), tops(TopRobe, 0x14141a, 0x14141a), accentc(0xc82828), stat(1)),
	def("hermione", fem(), hairc(0x5a3a1a, HairCurly), tops(TopRobe, 0x14141a, 0x14141a), accentc(0xc82828), stat(1)),
	def("dumbledore", hatc(HatWizard, 0x5a2a8a), beardc(BeardFull), hairc(0xf4f4f4, HairLong), tops(TopRobe, 0x5a2a8a, 0x5a2a8a), glass(GlassRound), stat(-2)),
	def("jugador_456", tops(TopLong, 0x1a7a5a, 0xf4f4f0), pantsc(0x1a7a5a), shoesc(0xf4f4f4), hairc(0x1a1a20, HairShort), stat(0)),
	def("guardia_rosa", maskc(0x14141a, MaskPlain), tops(TopLong, 0xf060a0, 0xf060a0), pantsc(0xf060a0), shoesc(0x14141a)),
	def("legolas", hairc(0xf0e0a0, HairLong), tops(TopRobe, 0x4a7a4a, 0x4a7a4a), pantsc(0x5a4a3a), bodyc(BodySlim), stat(-3)),
	def("aragorn", hairc(0x2a2020, HairLong), beardc(BeardStubble), tops(TopRobe, 0x3a3a46, 0x3a3a46), pantsc(0x3a3a2a), capec(0x3a4a3a), bodyc(BodyBig), stat(-1)),
}

func init() {
	famousPresets = append(famousPresets, asFuncs(morePresets)...)
	// underwear parties: a few of each
	famousPresets = append(famousPresets, presetUnderwearM, presetUnderwearM, presetUnderwearF, presetUnderwearF, presetUnderwearF)
	for k, v := range map[string]string{
		"mortadelo": "Mortadelo", "filemon": "Filemón", "super_lopez": "Super López", "zipi": "Zipi", "zape": "Zape",
		"rompetechos": "Rompetechos", "carpanta": "Carpanta", "mafalda": "Mafalda", "tintin": "Tintín", "capitan_haddock": "Capitán Haddock",
		"luffy": "Luffy", "zoro": "Zoro", "nami": "Nami", "usopp": "Usopp", "sanji": "Sanji", "chopper": "Chopper", "nico_robin": "Nico Robin", "franky": "Franky", "brook": "Brook", "ace_fuego": "Ace",
		"asterix": "Astérix", "obelix": "Obélix", "panoramix": "Panóramix", "abraracurcix": "Abraracúrcix", "legionario_romano": "Legionario romano", "julio_cesar": "Julio César", "cleopatra": "Cleopatra",
		"napoleon": "Napoleón", "van_gogh": "Van Gogh", "frida_kahlo": "Frida Kahlo", "dali": "Dalí", "charlie_chaplin": "Charlie Chaplin", "colon_almirante": "Cristóbal Colón", "isabel_la_catolica": "Isabel la Católica",
		"che": "Che Guevara", "poirot": "Hércules Poirot", "peter_pan": "Peter Pan", "capitan_garfio": "Capitán Garfio", "jane_selva": "Jane", "juana_de_arco": "Juana de Arco", "bruce_lee": "Bruce Lee",
		"wonder_woman": "Mujer Maravilla", "capitan_america": "Capitán América", "viuda_negra": "Viuda Negra", "ash_pokemon": "Ash", "caperucita_roja": "Caperucita Roja", "blancanieves": "Blancanieves",
		"jugador_456": "Jugador 456", "guardia_rosa": "Guardia rosa", "harry_potter": "Harry Potter", "sailor_moon": "Sailor Moon", "super_lopez2": "Super López",
		"asesino": "Asesino", "wally": "Wally", "wenda": "Walinda", "odlaw": "Odlaw", "mago_blancabarba": "Mago Barbablanca",
	} {
		displayNameOverrides[k] = v
	}
}

var _ = rand.New
