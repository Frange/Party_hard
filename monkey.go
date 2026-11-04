package main

// ---------------------------------------------------------------------------
// A tribute to the pirates of Monkey Island: Guybrush Threepwood, Stan, the
// governor Elaine Marley, LeChuck, the Voodoo Lady, Meathook and an Important
// Pirate, plus Murray, the evil talking skull on a stake (see murray.go).
// The lines are short catchphrases and original jokes in the same spirit
// (monkey_lines.go).
// ---------------------------------------------------------------------------

var castMonkey = []presetEntry{
	def("guybrush", hairc(0xe8d060, HairPony), tops(TopSuit, 0x2a4a9a, 0xf4f4f0), pantsc(0x3a2a1a), shoesc(0x2a1a10), skinc(0xf2c8a0), carryc(HoldMachete)),
	def("stan_barcos", hairc(0x2a1a10, HairComb), tops(TopLong, 0x8a5a2a, 0xe8d8a0), patn(3), pantsc(0x5a3a1a), shoesc(0x2a1a10), accentc(0xd02828), bodyc(BodyBig), stat(-1)),
	def("elaine_marley", fem(), hairc(0xc8501e, HairLong), tops(TopDress, 0x2a8a4a, 0x2a8a4a), shoesc(0x5a3a1a), bodyc(BodySlim), stat(1)),
	def("lechuck", skinc(0xa8c8b8), hairc(0xe0581a, HairShort), beardc(BeardFull), hatc(HatPirate, 0x3a1a1a), tops(TopSuit, 0x7a1a1a, 0x2a2a30), pantsc(0x1a1a22), shoesc(0x14141a), capec(0x4a1a1a), bodyc(BodyBig), stat(-2)),
	def("voodoo_lady", fem(), skinc(0x8a5a3a), hatc(HatTurban, 0x6a2a8a), tops(TopRobe, 0x7a2a8a, 0xe8c040), accentc(0xe8c040), bodyc(BodyBig), stat(-1)),
	def("meathook", hairc(0x3a2a1a, HairMessy), beardc(BeardFull), tops(TopTank, 0xe8e0c8, 0x2a3a6a), patn(2), pantsc(0x5a2a2a), shoesc(0x2a1a10), carryc(HoldClaw), bodyc(BodyBig)),
	def("pirata_importante", hairc(0x1a1210, HairMessy), beardc(BeardFull), hatc(HatPirate, 0x2a1a2a), tops(TopLong, 0x8a2a2a, 0xe8e0c8), patn(2), pantsc(0x3a2a1a), shoesc(0x14141a), weaponc(HoldKnife), bodyc(BodyBig)),
}

// monkeyQuirk: they shout their own lines now and then (see updateQuirk).
var monkeyQuirk = map[string]string{
	"guybrush": "mi_guybrush", "stan_barcos": "mi_stan", "elaine_marley": "mi_elaine", "lechuck": "mi_lechuck",
	"voodoo_lady": "mi_voodoo", "meathook": "mi_meathook", "pirata_importante": "mi_pirate",
}

func init() {
	famousPresets = append(famousPresets, asFuncs(castMonkey)...)
	for k, v := range monkeyNames {
		displayNameOverrides[k] = v
	}
	for k, v := range monkeyDesc {
		funnyDesc[k] = v
	}
	for k, v := range monkeyCatch {
		catchphrases[k] = v
		quirkLines[monkeyQuirk[k]] = v
	}
	quirkAnims["mi_guybrush"] = []int{AnimTalk, AnimPoint, AnimWave, AnimTalk}
	quirkAnims["mi_stan"] = []int{AnimWave, AnimTalk, AnimPoint, AnimWave}
	quirkAnims["mi_elaine"] = []int{AnimTalk, AnimPoint, AnimTalk}
	quirkAnims["mi_lechuck"] = []int{AnimLaugh, AnimPoint, AnimLaugh, AnimTalk}
	quirkAnims["mi_voodoo"] = []int{AnimTalk, AnimPoint}
	quirkAnims["mi_meathook"] = []int{AnimLaugh, AnimTalk}
	duetDefs = append(duetDefs, monkeyDuets...)
	for _, s := range []string{"guybrush", "stan_barcos", "elaine_marley", "lechuck", "voodoo_lady", "meathook", "pirata_importante"} {
		specialPool = append(specialPool, s)
	}
	starGroups = append(starGroups, []string{"guybrush", "elaine_marley", "lechuck"})
}
