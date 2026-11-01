package main

import "math/rand"

// ---------------------------------------------------------------------------
// A compact way to define characters: def("name", options...). Everything not set stays random,
// so the same definition produces slightly different people each time.
// ---------------------------------------------------------------------------

type po func(*CharSpec)

type presetEntry struct {
	Name string
	F    func(*rand.Rand) CharSpec
}

func def(name string, opts ...po) presetEntry {
	return presetEntry{name, func(r *rand.Rand) CharSpec {
		s := mk(r, name)
		s.Mask.A = 0
		for _, o := range opts {
			o(&s)
		}
		return s
	}}
}

func skinc(h uint32) po { return func(s *CharSpec) { s.Skin = hx(h) } }
func hairc(h uint32, style int) po {
	return func(s *CharSpec) {
		s.Hair = hx(h)
		s.Hair2 = style
	}
}
func hairs(style int) po { return func(s *CharSpec) { s.Hair2 = style } }
func tops(t int, c1, c2 uint32) po {
	return func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2 = t, hx(c1), hx(c2)
	}
}
func pantsc(c uint32) po { return func(s *CharSpec) { s.Pants = hx(c) } }
func shoesc(c uint32) po { return func(s *CharSpec) { s.Shoes = hx(c) } }
func bottomc(b int, c uint32) po {
	return func(s *CharSpec) {
		s.Bottom, s.Pants = b, hx(c)
	}
}
func hatc(h int, c uint32) po { return func(s *CharSpec) { s.Hat, s.HatCol = h, hx(c) } }
func glass(g int) po          { return func(s *CharSpec) { s.Glasses = g } }
func beardc(b int) po         { return func(s *CharSpec) { s.Beard = b } }
func bodyc(b int) po          { return func(s *CharSpec) { s.Body = b } }
func headc(h int) po          { return func(s *CharSpec) { s.Head = h } }
func stat(n int) po           { return func(s *CharSpec) { s.Stature = n } }
func carryc(k int) po         { return func(s *CharSpec) { s.Carry = k } }
func weaponc(k int) po        { return func(s *CharSpec) { s.Weapon = k } }
func maskc(c uint32, kind int) po {
	return func(s *CharSpec) {
		s.Mask, s.MaskKind = hx(c), kind
		s.Hair2, s.Beard, s.Glasses, s.Hat = HairBald, BeardNone, GlassNone, HatNone
	}
}
func capec(c uint32) po   { return func(s *CharSpec) { s.Cape = hx(c) } }
func suspc(c uint32) po   { return func(s *CharSpec) { s.Susp = hx(c) } }
func fem() po             { return func(s *CharSpec) { s.Female = true } }
func patn(n int) po       { return func(s *CharSpec) { s.Pattern = n } }
func earsn(n int) po      { return func(s *CharSpec) { s.Ears = n } }
func eyesc(e int) po      { return func(s *CharSpec) { s.Eyes = e } }
func accentc(c uint32) po { return func(s *CharSpec) { s.Accent = hx(c) } }

// colour helper: full-body costume (skin hidden by long sleeves and trousers)
func costume(c uint32) po {
	return func(s *CharSpec) {
		s.Top, s.Shirt, s.Shirt2 = TopLong, hx(c), hx(c)
		s.Bottom, s.Pants, s.Shoes = BotPants, hx(c), hx(c)
	}
}

func asFuncs(es []presetEntry) []func(*rand.Rand) CharSpec {
	var out []func(*rand.Rand) CharSpec
	for _, e := range es {
		out = append(out, e.F)
	}
	return out
}

// ---------------------------------------------------------------------------
// Horror killers: at most one per level, they arrive in a macabre car and kill people.
// ---------------------------------------------------------------------------
var killerPresets = []presetEntry{
	def("freddy", skinc(0x8a4a3a), hairs(HairBald), hatc(HatFedora, 0x6a4a2a), tops(TopLong, 0x8a2020, 0x2a6a2a), patn(2), pantsc(0x4a3a2a), shoesc(0x3a2a1a), weaponc(HoldClaw)),
	def("jason", maskc(0xe8e4d0, MaskHockey), tops(TopTee, 0x5a5a4a, 0x5a5a4a), pantsc(0x34342a), bodyc(BodyBig), stat(-2), weaponc(HoldMachete)),
	def("michael", maskc(0xe4dccc, MaskPlain), tops(TopLong, 0x2a3a5a, 0x2a3a5a), pantsc(0x2a3a5a), bodyc(BodyBig), stat(-1), weaponc(HoldKnife)),
	def("ghostface", maskc(0xf2f2f2, MaskGhost), tops(TopRobe, 0x14141a, 0x14141a), weaponc(HoldKnife), stat(-1)),
	def("leatherface", maskc(0xc4a07c, MaskPig), tops(TopLong, 0x5a4a3a, 0x5a4a3a), pantsc(0x3a3028), bodyc(BodyGiant), stat(-2), weaponc(HoldChainsaw)),
	def("pinhead", skinc(0xd8d0d0), hairs(HairBald), tops(TopRobe, 0x14141a, 0x14141a), weaponc(HoldClaw), stat(-1)),
	def("chucky", hairc(0xb04a1a, HairMessy), headc(HeadBig), tops(TopLong, 0x2a8a4a, 0xd04040), patn(2), pantsc(0x2a4ab8), suspc(0x2a4ab8), bodyc(BodySlim), stat(7), weaponc(HoldKnife)),
	def("pennywise", skinc(0xf2f2f2), hairc(0xd83030, HairCurly), maskc(0xf4f4f0, MaskClown), tops(TopRobe, 0xc8c8c0, 0xc8c8c0), carryc(HoldBalloons), weaponc(HoldClaw), stat(-1)),
	def("demonio", skinc(0xc02a22), hairs(HairBald), hatc(HatHorns, 0), tops(TopBare, 0xc02a22, 0xc02a22), pantsc(0x2a0a0a), shoesc(0x14100c), capec(0x5a0a0a), weaponc(HoldTorch), bodyc(BodyBig), stat(-2)),
	def("mimo", maskc(0xf4f4f0, MaskClown), hatc(HatBeret, 0x14141a), tops(TopLong, 0xf4f4f4, 0x14141a), patn(2), pantsc(0x14141a), weaponc(HoldKnife)),
	def("caballo", maskc(0x8a5a30, MaskPig), tops(TopLong, 0x2a2a30, 0x2a2a30), pantsc(0x1a1a20), bodyc(BodyBig), stat(-1), weaponc(HoldMachete)),
	def("carnicero", beardc(BeardFull), hairs(HairBald), tops(TopLong, 0xe8e0d0, 0xa01818), pantsc(0x4a3a2a), bodyc(BodyBig), stat(-1), weaponc(HoldMachete)),
	def("doctor_loco", maskc(0xe8f0f4, MaskPlain), tops(TopSuit, 0xf4f4f4, 0xd0d8e8), pantsc(0x3a4a5a), hairc(0x8a8a90, HairMessy), weaponc(HoldKnife), stat(-1)),
	def("enfermera", fem(), hatc(HatMedic, 0xf4f4f4), maskc(0xe8f0f4, MaskPlain), tops(TopDress, 0xf4f4f4, 0xd02828), patn(3), weaponc(HoldKnife)),
	def("calabaza", hatc(HatPumpkin, 0xf08a20), tops(TopRobe, 0x2a1a14, 0x2a1a14), weaponc(HoldMachete), stat(-1)),
	def("espantapajaros", hatc(HatSombrero, 0xd8b860), maskc(0xc8a860, MaskPlain), tops(TopLong, 0x8a6a3a, 0x5a7a3a), patn(3), pantsc(0x6a5a3a), weaponc(HoldClaw), bodyc(BodySlim), stat(-2)),
	def("santa_psicopata", hatc(HatBeanie, 0xd82a2a), beardc(BeardFull), hairc(0xf4f4f4, HairShort), tops(TopLong, 0xd82a2a, 0xd82a2a), pantsc(0xd82a2a), shoesc(0x14101a), bodyc(BodyFat), weaponc(HoldMachete)),
	def("ninja_asesino", hatc(HatNinja, 0x14141a), tops(TopLong, 0x14141a, 0x14141a), pantsc(0x14141a), weaponc(HoldKnife), bodyc(BodySlim), stat(-1)),
	def("esqueleto", maskc(0xf0f0e0, MaskSkull), hatc(HatTopHat, 0x14141a), tops(TopSuit, 0x1a1a22, 0xf4f4f4), pantsc(0x1a1a22), weaponc(HoldMachete), bodyc(BodySlim), stat(-2)),
	def("muneca", fem(), maskc(0xf4d8c8, MaskClown), hairc(0xe8c860, HairPony), tops(TopDress, 0xe85a9a, 0xe85a9a), weaponc(HoldKnife), stat(5), headc(HeadBig)),
	def("vampiro", skinc(0xece4e4), hairc(0x14101a, HairComb), tops(TopSuit, 0x14101a, 0xf4f4f8), accentc(0xa01020), pantsc(0x14101a), capec(0x7a1020), weaponc(HoldClaw), stat(-1)),
	def("bebe_gigante", hatc(HatBeanie, 0xf8d8e8), tops(TopTee, 0xf8f0f0, 0xf8f0f0), bottomc(BotTrunks, 0xf8f0f0), bodyc(BodyGiant), stat(-3), headc(HeadHuge), weaponc(HoldMachete)),
	def("lenador", beardc(BeardFull), hairc(0x5a3a22, HairShort), hatc(HatBeanie, 0xc02a2a), tops(TopLong, 0xc02a2a, 0x14141a), patn(3), pantsc(0x3a4a6a), bodyc(BodyBig), weaponc(HoldChainsaw), stat(-2)),
	def("cirujano", maskc(0x9ad0c0, MaskPlain), hatc(HatBeanie, 0x9ad0c0), tops(TopLong, 0x9ad0c0, 0x9ad0c0), pantsc(0x7ab0a0), weaponc(HoldKnife)),
	def("bruja", fem(), hatc(HatWizard, 0x14141a), skinc(0x8ac070), hairc(0x2a2a2a, HairLong), tops(TopRobe, 0x2a1a3a, 0x2a1a3a), weaponc(HoldClaw), stat(-1)),
	def("momia", skinc(0xe8dcc0), maskc(0xe8dcc0, MaskPlain), tops(TopRobe, 0xe8dcc0, 0xc8b898), weaponc(HoldClaw), stat(-1)),
	def("hombre_lobo", skinc(0x6a4a2a), hairc(0x4a3018, HairMessy), beardc(BeardFull), tops(TopTank, 0x6a4a2a, 0x6a4a2a), pantsc(0x4a3a2a), bodyc(BodyGiant), weaponc(HoldClaw), stat(-3)),
	def("cocinero_asesino", hatc(HatChef, 0xf4f4f4), beardc(BeardMustache), tops(TopLong, 0xf4f4f4, 0xa01818), pantsc(0x3a3a46), bodyc(BodyBig), weaponc(HoldMachete)),
	def("payaso_triste", maskc(0xf4f4f0, MaskClown), hatc(HatBowler, 0x3a3a8a), tops(TopSuit, 0x6a6a9a, 0xf4f4f4), pantsc(0x6a6a9a), weaponc(HoldKnife), bodyc(BodySlim), stat(-1)),
	def("cazador", hatc(HatCowboy, 0x5a4a2a), beardc(BeardFull), tops(TopVest, 0x5a6a3a, 0xe8d8a0), pantsc(0x4a4a2a), weaponc(HoldMachete), bodyc(BodyBig)),
	def("buzo", maskc(0x2a2a32, MaskPlain), hatc(HatHardhat, 0xf0c820), tops(TopLong, 0x1a1a22, 0x1a1a22), pantsc(0x1a1a22), shoesc(0xf0c820), weaponc(HoldKnife), bodyc(BodyBig)),
	def("astronauta_asesino", maskc(0xf4f4f8, MaskPlain), tops(TopLong, 0xf4f4f8, 0xf4f4f8), pantsc(0xf4f4f8), shoesc(0x8a8a94), weaponc(HoldMachete), bodyc(BodyBig), stat(-1)),
	def("gladiador", hatc(HatHelmet, 0xc8a040), tops(TopArmor, 0xc8a040, 0xc8a040), pantsc(0x8a2a2a), shoesc(0x6a4a2a), weaponc(HoldMachete), bodyc(BodyGiant), stat(-2)),
	def("pirata_fantasma", skinc(0xb8d8d0), hatc(HatPirate, 0x14141a), beardc(BeardFull), hairc(0xe8f0f0, HairLong), tops(TopLong, 0x5a7a8a, 0x5a7a8a), pantsc(0x3a4a5a), weaponc(HoldClaw)),
	def("oso_fiesta", maskc(0x7a4a2a, MaskPig), tops(TopLong, 0x7a4a2a, 0x7a4a2a), pantsc(0x7a4a2a), bodyc(BodyGiant), weaponc(HoldClaw), stat(-3)),
	def("novia_cadaver", fem(), skinc(0xd0e0e0), maskc(0xe8f0f0, MaskPlain), hairc(0x14141a, HairLong), tops(TopDress, 0xf4f4f8, 0xf4f4f8), patn(3), weaponc(HoldKnife)),
	def("conejo_asesino", hatc(HatBunny, 0xf4f4f4), maskc(0xf4f4f4, MaskPig), tops(TopLong, 0xf4f4f4, 0xf4f4f4), pantsc(0xf4f4f4), weaponc(HoldMachete), bodyc(BodyBig), stat(-1)),
	def("muerte", maskc(0xf0f0e0, MaskSkull), hatc(HatNinja, 0x14141a), tops(TopRobe, 0x14141a, 0x14141a), capec(0x14141a), weaponc(HoldMachete), stat(-3)),
	def("dj_maldito", maskc(0x2a2a32, MaskSkull), hatc(HatTV, 0x6a6a74), tops(TopHoodie, 0x14141a, 0x14141a), pantsc(0x14141a), weaponc(HoldClaw), glass(GlassNeon)),
	def("robot_asesino", skinc(0x8a8e98), hairs(HairBald), glass(GlassNeon), hatc(HatAntenna, 0x8a8e98), tops(TopArmor, 0x8a8e98, 0x8a8e98), pantsc(0x5a5e68), bodyc(BodyRobot), weaponc(HoldChainsaw), stat(-2)),
	def("chef_frances", hatc(HatChef, 0xf4f4f4), beardc(BeardMustache), tops(TopLong, 0xf4f4f4, 0xf4f4f4), pantsc(0x3a3a46), weaponc(HoldBaguette), bodyc(BodyBig)),
	def("payaso_circo", maskc(0xf4f4f0, MaskClown), hatc(HatPartyCone, 0xe85a9a), tops(TopHawaiian, 0xe85a9a, 0x3ad0a8), pantsc(0x3a6ae0), shoesc(0xf0c030), weaponc(HoldKnife), bodyc(BodyFat)),
	def("vaquero_fantasma", skinc(0xb8d0d0), hatc(HatCowboy, 0x4a3a2a), beardc(BeardStubble), tops(TopVest, 0x6a5a3a, 0xe8d8a0), pantsc(0x3a3a4a), weaponc(HoldMachete)),
	def("bailarin_ballet", maskc(0xe8e0e8, MaskPlain), tops(TopTank, 0xf4f4f8, 0xf4f4f8), bottomc(BotSkirt, 0xf4a0c0), shoesc(0xf4a0c0), weaponc(HoldKnife), bodyc(BodySlim)),
	def("taxista_loco", hatc(HatCap, 0xe8c830), beardc(BeardStubble), tops(TopLong, 0xe8c830, 0x14141a), patn(2), pantsc(0x14141a), weaponc(HoldClaw)),
	def("oficinista_zombi", skinc(0x8ab088), hairc(0x2a2a2a, HairComb), tops(TopSuit, 0x4a4a56, 0xe8e8f0), accentc(0xc82a2a), pantsc(0x4a4a56), weaponc(HoldBriefcase)),
	def("arbitro_asesino", hatc(HatCap, 0x14141a), tops(TopTee, 0x14141a, 0xf4f4f4), patn(2), pantsc(0x14141a), weaponc(HoldKnife), bodyc(BodyBig)),
	def("torero_maldito", hatc(HatMontera, 0x14141a), tops(TopVest, 0xe8c040, 0xc82a2a), pantsc(0xe8c040), shoesc(0xf4f4f4), weaponc(HoldSaber), bodyc(BodySlim)),
	def("jardinero_loco", hatc(HatBucket, 0x8a7a4a), beardc(BeardFull), tops(TopVest, 0x4a7a3a, 0xe8d8a0), pantsc(0x3a4a3a), weaponc(HoldPlant), bodyc(BodyBig)),
	def("karateka", maskc(0x14141a, MaskPlain), tops(TopRobe, 0xf4f4f4, 0xf4f4f4), accentc(0x14141a), weaponc(HoldClaw), stat(-1)),
	def("domador", hatc(HatTopHat, 0xc82a2a), beardc(BeardMustache), tops(TopSuit, 0xc82a2a, 0xf4f4f4), pantsc(0x14141a), shoesc(0x14141a), weaponc(HoldFlag)),
	def("dentista", maskc(0xe8f4f4, MaskPlain), tops(TopLong, 0x9ad0e0, 0x9ad0e0), pantsc(0x7ab0c0), weaponc(HoldKnife)),
	def("pescador", hatc(HatBucket, 0xe8c830), beardc(BeardFull), tops(TopRobe, 0xe8c830, 0xe8c830), pantsc(0x3a4a5a), weaponc(HoldMachete)),
	def("brujo", hatc(HatWizard, 0x2a1a3a), beardc(BeardFull), hairc(0x6a6a74, HairLong), tops(TopRobe, 0x2a1a3a, 0x2a1a3a), weaponc(HoldTorch)),
	def("caballero_negro", hatc(HatHelmet, 0x14141a), tops(TopArmor, 0x1a1a22, 0x1a1a22), pantsc(0x14141a), shoesc(0x14141a), weaponc(HoldSaber), bodyc(BodyGiant), stat(-2)),
	def("samurai_oscuro", hatc(HatSamurai, 0x2a1a1a), tops(TopRobe, 0x3a1a1a, 0x3a1a1a), pantsc(0x14141a), weaponc(HoldMachete), bodyc(BodyBig)),
	def("marinero_maldito", skinc(0xa8c0b8), hatc(HatCap, 0xf4f4f4), tops(TopTee, 0xf4f4f4, 0x2a4ab8), patn(2), pantsc(0xf4f4f4), weaponc(HoldFlag)),
	def("disco_vampiro", skinc(0xece4e4), hairc(0x14101a, HairBeehive), tops(TopSuit, 0xe83a8a, 0xf4f4f8), pantsc(0xe83a8a), capec(0x6a1030), weaponc(HoldClaw)),
	def("hombre_saco", maskc(0xb89a60, MaskPlain), tops(TopRobe, 0x6a5a3a, 0x6a5a3a), weaponc(HoldClaw), stat(-1)),
	def("fantasma_opera", maskc(0xf4f4f4, MaskPlain), hatc(HatTopHat, 0x14141a), tops(TopSuit, 0x14141a, 0xf4f4f4), capec(0x14141a), pantsc(0x14141a), weaponc(HoldGuitar)),
	def("impermeable", hatc(HatBucket, 0xf0c820), tops(TopRobe, 0xf0c820, 0xf0c820), maskc(0x2a2a30, MaskSkull), weaponc(HoldMachete), stat(-1)),
}

// ---------------------------------------------------------------------------
// Fictional characters from films, series, comics and games (special_people)
// ---------------------------------------------------------------------------
var fictionalPresets = []presetEntry{
	// comics and superheroes
	def("hulk", skinc(0x5ab040), hairc(0x14101a, HairShort), tops(TopBare, 0x5ab040, 0x5ab040), bottomc(BotShorts, 0x7a3aa8), shoesc(0x5ab040), bodyc(BodyGiant), stat(-3), headc(HeadNormal)),
	def("spiderman", maskc(0xc82a2a, MaskPlain), tops(TopLong, 0xc82a2a, 0x2a4ab8), pantsc(0x2a4ab8), shoesc(0xc82a2a), patn(3), bodyc(BodySlim)),
	def("batman", hatc(HatNinja, 0x1a1a24), tops(TopLong, 0x2a2a34, 0xf0c030), pantsc(0x2a2a34), shoesc(0x14141a), capec(0x1a1a24), stat(-1)),
	def("superman", hairc(0x14101a, HairComb), tops(TopLong, 0x2a50c0, 0xf0c030), pantsc(0x2a50c0), shoesc(0xc82a2a), capec(0xc82a2a), bodyc(BodyBig), stat(-2)),
	def("joker", skinc(0xf2f2f2), hairc(0x3aa040, HairMessy), tops(TopSuit, 0x6a2a8a, 0xf0e060), accentc(0xf0e060), pantsc(0x4a1a6a), beardc(BeardMustache)),
	def("arlequina", fem(), skinc(0xf4f0f0), hairc(0x3a60e0, HairBuns), tops(TopTank, 0xc82a2a, 0x14141a), bottomc(BotShorts, 0x14141a), shoesc(0xc82a2a)),
	def("deadpool", maskc(0xc02020, MaskPlain), tops(TopLong, 0xc02020, 0x14141a), pantsc(0xc02020), shoesc(0x14141a), weaponc(HoldMachete), carryc(HoldMachete)),
	def("lobezno", hairc(0x14101a, HairSpiky), tops(TopTank, 0xf0c820, 0xf0c820), pantsc(0x2a4ab8), beardc(BeardStubble), weaponc(HoldClaw), carryc(HoldClaw), stat(1), bodyc(BodyBig)),
	def("thor", hairc(0xe8d070, HairLong), tops(TopArmor, 0xa8b0c0, 0xc82a2a), pantsc(0x3a3a46), capec(0xc82a2a), bodyc(BodyBig), stat(-2), beardc(BeardStubble)),
	def("ironman", hatc(HatHelmet, 0xc02020), tops(TopArmor, 0xc02a2a, 0xf0c030), pantsc(0xc02a2a), shoesc(0xf0c030), bodyc(BodyBig)),
	def("thanos", skinc(0x8a5ab0), headc(HeadBig), hairs(HairBald), tops(TopArmor, 0xd8a820, 0x5a3a8a), pantsc(0x5a3a8a), bodyc(BodyGiant), stat(-3)),
	def("doctor_extrano", hairc(0x2a2020, HairComb), beardc(BeardGoatee), tops(TopRobe, 0x2a50a0, 0xc82a2a), capec(0xc82a2a)),
	def("motorista_fantasma", maskc(0xf0f0e8, MaskSkull), hairc(0xff8020, HairSpiky), tops(TopLong, 0x1a1a20, 0x1a1a20), pantsc(0x1a1a20), bodyc(BodyBig)),
	def("blade", glass(GlassShades), hairs(HairShort), tops(TopRobe, 0x1a1a20, 0x1a1a20), pantsc(0x1a1a20), weaponc(HoldMachete)),
	def("bane", maskc(0x5a5a64, MaskPlain), tops(TopVest, 0x4a3a2a, 0x4a3a2a), pantsc(0x3a3a2a), bodyc(BodyGiant), stat(-3)),
	// films
	def("terminator", glass(GlassShades), hairs(HairShort), tops(TopLong, 0x1a1a20, 0x1a1a20), pantsc(0x1a1a20), bodyc(BodyBig), stat(-2), beardc(BeardStubble)),
	def("robocop", hatc(HatHelmet, 0xb4bccc), skinc(0xd0b090), glass(GlassNeon), tops(TopArmor, 0x9aa4b8, 0x2a3040), accentc(0x2a3040), pantsc(0x4a5468), shoesc(0x2a2e3a), bodyc(BodyBig), stat(-1), carryc(HoldNone), weaponc(HoldNone)),
	def("neo", glass(GlassShades), hairs(HairShort), tops(TopRobe, 0x101014, 0x101014)),
	def("john_wick", hairc(0x1a1a20, HairLong), beardc(BeardStubble), tops(TopSuit, 0x16161c, 0xf4f4f8), pantsc(0x16161c), accentc(0x16161c)),
	def("rambo", hairc(0x1a1a20, HairLong), tops(TopBare, 0xc89a6e, 0xc89a6e), pantsc(0x4a5a3a), bodyc(BodyBig), stat(-2), hatc(HatNinja, 0xc82a2a)),
	def("rocky", hairc(0x1a1a20, HairShort), tops(TopBare, 0xc89a6e, 0xc89a6e), bottomc(BotTrunks, 0xc82a2a), shoesc(0xc82a2a), bodyc(BodyBig)),
	def("jack_sparrow", hatc(HatPirate, 0x3a2a1a), hairc(0x1a1210, HairDreads), beardc(BeardGoatee), tops(TopLong, 0x8a2a2a, 0xe8e0c8), pantsc(0x3a2a1a), shoesc(0x2a1a10), weaponc(HoldMachete)),
	def("wonka", hatc(HatTopHat, 0x6a2a8a), hairc(0xa0602a, HairBob), tops(TopSuit, 0x6a2a8a, 0xf0e060), accentc(0x3a8a3a), pantsc(0x4a1a6a)),
	def("beetlejuice", skinc(0xe8e4e0), hairc(0xe8f0e0, HairMessy), tops(TopSuit, 0x14141a, 0xf4f4f4), patn(2), pantsc(0x14141a), accentc(0x2a8a2a)),
	def("shrek", skinc(0x88b848), hairs(HairBald), earsn(2), tops(TopVest, 0x7a5a3a, 0xe8e0c8), pantsc(0x5a4a2a), bodyc(BodyFat), stat(-1)),
	def("gru", skinc(0xf0d0b8), hairs(HairBald), headc(HeadBig), tops(TopLong, 0x2a2a34, 0x2a2a34), pantsc(0x2a2a34), accentc(0x8a8a96), bodyc(BodySlim), stat(1)),
	def("minion", skinc(0xf8d838), hairs(HairBald), glass(GlassBig), tops(TopLong, 0xf8d838, 0xf8d838), pantsc(0x3a5ab8), suspc(0x3a5ab8), bodyc(BodyFat), headc(HeadBig), stat(6)),
	def("el_nota", beardc(BeardFull), glass(GlassShades), tops(TopRobe, 0x8a6a4a, 0x8a6a4a), shoesc(0xa89a7a), hairc(0x6a4a2a, HairLong)),
	def("borat", hairc(0x1a1a20, HairComb), beardc(BeardMustache), tops(TopSuit, 0x6a6a6e, 0xf4f4f4), pantsc(0x6a6a6e)),
	def("austin", hairc(0x5a3a1a, HairShort), glass(GlassBig), tops(TopSuit, 0x2a50c0, 0xf0f0e0), pantsc(0x2a50c0), accentc(0xc82a2a)),
	def("zoolander", hairc(0x1a1a20, HairComb), tops(TopLong, 0x2a2a34, 0x2a2a34), pantsc(0x2a2a34)),
	def("ace_ventura", hairc(0x1a1a20, HairPompadour), tops(TopHawaiian, 0xe04a8a, 0xf8f8f0), patn(1), bottomc(BotShorts, 0xf0e0b0), shoesc(0xe8e8e8)),
	def("mr_bean", hairc(0x3a2a1a, HairComb), tops(TopSuit, 0x6a4a2a, 0xf0f0e8), accentc(0xc82a2a), pantsc(0x4a3a2a), carryc(HoldBriefcase)),
	def("napoleon_d", hairc(0xd06a20, HairCurly), glass(GlassBig), tops(TopTee, 0xe8e0d0, 0xe8e0d0), pantsc(0x3a5ab8), bodyc(BodySlim)),
	def("han_solo", hairc(0x5a3a1a, HairShort), tops(TopVest, 0x14141a, 0xf0f0e8), pantsc(0x2a2a34), shoesc(0x14141a), stat(-1)),
	def("luke", hairc(0xd0b060, HairShort), tops(TopRobe, 0xe8e4d8, 0xe8e4d8), weaponc(HoldSaber), carryc(HoldSaber)),
	def("boba_fett", hatc(HatHelmet, 0x4a8a4a), tops(TopArmor, 0x4a8a4a, 0xc89a40), pantsc(0x4a8a4a), bodyc(BodyBig)),
	def("mandaloriano", hatc(HatHelmet, 0xc4ccd8), tops(TopArmor, 0xa8b0c0, 0x4a3a2a), pantsc(0x4a3a2a), capec(0x4a3a2a), bodyc(BodyBig)),
	// series
	def("dexter", hairc(0x3a2a1a, HairShort), tops(TopTee, 0x2a60c8, 0x2a60c8), pantsc(0x2a2a30), glass(GlassRound)),
	def("heisenberg", hatc(HatFedora, 0x14141a), glass(GlassRound), beardc(BeardGoatee), tops(TopLong, 0x2a2a30, 0x2a2a30), pantsc(0x2a2a30), hairs(HairBald)),
	def("saul", hairc(0x3a2a1a, HairComb), tops(TopSuit, 0xc89a2a, 0xe04a8a), accentc(0x3a8a3a), pantsc(0xc89a2a), patn(2)),
	def("tony_s", hairc(0x2a2020, HairShort), tops(TopRobe, 0x6a6a74, 0x6a6a74), bodyc(BodyBig), stat(-1), beardc(BeardStubble)),
	def("tommy_s", hatc(HatCap, 0x4a4a52), hairc(0x2a2020, HairShort), tops(TopRobe, 0x3a3a46, 0x3a3a46), pantsc(0x3a3a46), beardc(BeardStubble)),
	def("atracador_rojo", maskc(0xe8d0b0, MaskPlain), tops(TopLong, 0xc82a2a, 0xc82a2a), pantsc(0xc82a2a), shoesc(0x14141a)),
	def("rick", hairc(0x8ac8e8, HairSpiky), tops(TopLong, 0xf0f0f0, 0xf0f0f0), pantsc(0x8a7a5a), glass(GlassNone), eyesc(EyesNormal), stat(-1)),
	def("morty", hairc(0x6a4a2a, HairShort), tops(TopTee, 0xf0e050, 0xf0e050), pantsc(0x3a5ab8), headc(HeadBig), stat(3), bodyc(BodySlim)),
	def("jon_nieve", hairc(0x2a2020, HairCurly), tops(TopLong, 0x1a1a20, 0x1a1a20), pantsc(0x1a1a20), capec(0x14141a), beardc(BeardStubble), weaponc(HoldMachete), carryc(HoldMachete)),
	def("negan", hairc(0x3a3a3a, HairShort), beardc(BeardFull), tops(TopLong, 0x1a1a20, 0x1a1a20), pantsc(0x2a2a30), carryc(HoldMachete), accentc(0xc82a2a)),
	def("miercoles", fem(), hairc(0x14101a, HairBuns), skinc(0xf0e4e0), tops(TopDress, 0x14141a, 0xf4f4f4), shoesc(0x14141a), stat(2), bodyc(BodySlim)),
	def("once", fem(), hairs(HairBob), hairc(0x5a3a2a, HairBob), tops(TopDress, 0xf090b0, 0xf090b0), shoesc(0xe8e8e8), stat(2)),
	def("homelander", hairc(0xe8d070, HairComb), tops(TopLong, 0x2a3a8a, 0xc82a2a), pantsc(0x2a3a8a), capec(0xc82a2a), bodyc(BodyBig), stat(-2)),
	def("michael_scott", hairc(0x3a2a1a, HairComb), tops(TopSuit, 0x2a2a34, 0xf4f4f4), accentc(0x2a2a8a), carryc(HoldCup)),
	def("dwight", hairc(0x5a3a1a, HairComb), glass(GlassRound), tops(TopLong, 0xc8a82a, 0xc8a82a), pantsc(0x3a2a1a), accentc(0x3a2a1a)),
	def("sheldon", hairc(0x3a2a1a, HairShort), tops(TopTee, 0xe04a4a, 0x14141a), patn(2), pantsc(0x8a7a5a)),
	def("barney", hairc(0x3a2a1a, HairComb), tops(TopSuit, 0x1a1a24, 0xf4f4f8), accentc(0x2a60c8), pantsc(0x1a1a24)),
	def("dr_house", hairc(0x6a6a6e, HairShort), beardc(BeardStubble), tops(TopSuit, 0x3a3a46, 0x5a8ad8), pantsc(0x3a3a46), carryc(HoldBroom)),
	// games
	def("sonic", skinc(0xf0d0a8), hairc(0x2a60d0, HairSpiky), headc(HeadBig), tops(TopLong, 0x2a60d0, 0x2a60d0), pantsc(0x2a60d0), shoesc(0xd82a2a), bodyc(BodyNormal), stat(2)),
	def("crash", skinc(0xe8802a), hairc(0xd82a2a, HairSpiky), tops(TopBare, 0xe8802a, 0xe8802a), bottomc(BotShorts, 0x2a50c0), shoesc(0xf4f4f0), headc(HeadBig), bodyc(BodySlim), stat(1)),
	def("wario", hatc(HatCap, 0xf0d020), beardc(BeardMustache), tops(TopLong, 0xf0d020, 0xf0d020), pantsc(0x7a3aa8), suspc(0x7a3aa8), bodyc(BodyFat), stat(2)),
	def("waluigi", hatc(HatCap, 0x6a2a8a), beardc(BeardMustache), tops(TopLong, 0x6a2a8a, 0x6a2a8a), pantsc(0x2a2a40), bodyc(BodySlim), stat(-3)),
	def("donkey_kong", skinc(0x6a4020), hairs(HairBald), headc(HeadBig), tops(TopBare, 0x6a4020, 0xc82a2a), bottomc(BotShorts, 0x6a4020), shoesc(0x6a4020), bodyc(BodyGiant), stat(-2)),
	def("bowser", skinc(0x4a8a30), hairc(0xd84a20, HairSpiky), hatc(HatHorns, 0), tops(TopArmor, 0xd8a820, 0x2a8a2a), pantsc(0x4a8a30), bodyc(BodyGiant), stat(-2)),
	def("lara_croft", fem(), hairc(0x5a3a1a, HairPony), tops(TopTank, 0x3aa0a0, 0x3aa0a0), bottomc(BotShorts, 0x4a3a2a), shoesc(0x3a2a1a), carryc(HoldNone)),
	def("nathan_drake", hairc(0x4a3a2a, HairShort), beardc(BeardStubble), tops(TopLong, 0xe8e0c8, 0xe8e0c8), pantsc(0x5a4a3a)),
	def("jefe_maestro", hatc(HatHelmet, 0x4a8a50), tops(TopArmor, 0x4a8a50, 0xd8a820), pantsc(0x4a8a50), bodyc(BodyBig), stat(-1)),
	def("kratos", skinc(0xe8e4dc), hairs(HairBald), beardc(BeardFull), hairc(0x6a2a1a, HairBald), tops(TopBare, 0xe8e4dc, 0xc82a2a), pantsc(0x5a3a2a), bodyc(BodyGiant), stat(-2)),
	def("leon_k", hairc(0x8a6a3a, HairLong), tops(TopRobe, 0x4a3a2a, 0x4a3a2a), pantsc(0x3a3a46)),
	def("agente_47", hairs(HairBald), tops(TopSuit, 0x16161c, 0xf4f4f8), accentc(0xc82a2a), pantsc(0x16161c), glass(GlassNone)),
	def("duke", hairc(0xe8d070, HairComb), glass(GlassShades), tops(TopTank, 0xf0f0f0, 0xf0f0f0), pantsc(0x4a5a3a), bodyc(BodyBig), stat(-2)),
	def("doom_slayer", hatc(HatHelmet, 0x4a8a4a), tops(TopArmor, 0x4a8a4a, 0x2a4a2a), pantsc(0x4a8a4a), bodyc(BodyGiant), stat(-2)),
	def("solid_snake", hatc(HatNinja, 0xc82a2a), skinc(0xe8c8a0), tops(TopLong, 0x4a5a6a, 0x4a5a6a), pantsc(0x4a5a6a), beardc(BeardStubble)),
	def("kirby", skinc(0xf4a0c0), hairs(HairBald), headc(HeadHuge), tops(TopLong, 0xf4a0c0, 0xf4a0c0), pantsc(0xf4a0c0), shoesc(0xc82a2a), bodyc(BodyFat), stat(7)),
	def("pikachu", skinc(0xf8d838), hatc(HatBunny, 0xf8d838), tops(TopLong, 0xf8d838, 0xf8d838), pantsc(0xf8d838), shoesc(0xf8d838), headc(HeadBig), bodyc(BodySlim), stat(4)),
	def("steve_cubos", skinc(0xc89a7a), hairc(0x4a3a2a, HairShort), tops(TopTee, 0x3ab0b0, 0x3ab0b0), pantsc(0x3a3aa0), shoesc(0x6a6a74), bodyc(BodyBig)),
	def("creeper", skinc(0x58b040), hairs(HairBald), headc(HeadBig), tops(TopLong, 0x58b040, 0x2a7a2a), patn(3), pantsc(0x58b040), shoesc(0x2a7a2a), bodyc(BodySlim), stat(1)),
}

// ---------------------------------------------------------------------------
// Party archetypes (people): generic types, no real people
// ---------------------------------------------------------------------------
var archetypePresets = []presetEntry{
	def("cani", hatc(HatCap, 0x2a60c8), tops(TopHoodie, 0x2a60c8, 0xf4f4f4), pantsc(0x2a60c8), shoesc(0xf4f4f4), suspc(0xf0c820)),
	def("ravero_fluor", glass(GlassNeon), hairc(0xff40a0, HairSpiky), tops(TopTank, 0x80ff40, 0x80ff40), pantsc(0xff40a0), shoesc(0x40e0ff)),
	def("techno_vikingo", hatc(HatViking, 0x9aa0a8), beardc(BeardFull), hairc(0xd8a050, HairLong), tops(TopVest, 0x6a4a2a, 0x6a4a2a), pantsc(0x4a3a2a), bodyc(BodyBig)),
	def("dj_frustrado", hatc(HatCap, 0x2a2a34), tops(TopTee, 0x1a1a20, 0xf04040), glass(GlassShades), bodyc(BodySlim)),
	def("breakdancer", hatc(HatBeanie, 0xe04a4a), tops(TopHoodie, 0x2a8a4a, 0xf4f4f4), pantsc(0x2a2a34), shoesc(0xf4f4f4)),
	def("festivalero", hatc(HatBucket, 0xf0c820), tops(TopTank, 0xf060a0, 0xf060a0), bottomc(BotShorts, 0x3aa0e0), accentc(0xf0c820), suspc(0xf0c820)),
	def("hippie", hairs(HairLong), tops(TopTee, 0xf060a0, 0x40c0f0), patn(3), pantsc(0x3a8a5a), shoesc(0x8a5a2a), beardc(BeardFull)),
	def("rastafari", hairs(HairDreads), hatc(HatBeanie, 0x3aa040), tops(TopTee, 0xf0d020, 0xc82a2a), patn(2), pantsc(0x3aa040), skinc(0x6b4630)),
	def("despedida_novio", hatc(HatPartyCone, 0xf060a0), tops(TopHawaiian, 0x40c0f0, 0xf8f8f0), patn(1), bottomc(BotShorts, 0xf0e0b0), carryc(HoldBeer)),
	def("borracho", hairs(HairMessy), tops(TopTee, 0xc8b88a, 0xc8b88a), pantsc(0x4a3a2a), carryc(HoldBeer), beardc(BeardStubble)),
	def("segurata", glass(GlassShades), tops(TopSuit, 0x14141a, 0xf4f4f4), accentc(0x14141a), pantsc(0x14141a), bodyc(BodyBig), stat(-2), hairs(HairBald)),
	def("relaciones_publicas", tops(TopSuit, 0x2a2a3a, 0xf4f4f4), accentc(0xd04040), pantsc(0x2a2a3a), carryc(HoldSign), hairs(HairComb)),
	def("camarero_hosco", tops(TopSuit, 0x1a1a22, 0xf4f4f8), accentc(0x14141a), pantsc(0x1a1a22), carryc(HoldTray), beardc(BeardStubble)),
	def("quiere_reggaeton", hatc(HatCap, 0x14141a), glass(GlassShades), tops(TopTank, 0xf4f4f4, 0xf4f4f4), pantsc(0x3a5ab8), suspc(0xf0c820)),
	def("metalero", hairc(0x14101a, HairLong), tops(TopTank, 0x14141a, 0xc0c0c8), pantsc(0x14141a), shoesc(0x14141a), suspc(0xb0b0b8), beardc(BeardFull)),
	def("punk_cresta", hairc(0xd82a60, HairMohawk), tops(TopVest, 0x16161c, 0x16161c), pantsc(0x9a1a1a), patn(2), shoesc(0x14141a), suspc(0xc0c0c8)),
	def("gotico", skinc(0xf0e8ec), hairc(0x0c0a10, HairLong), tops(TopRobe, 0x101014, 0x101014), shoesc(0x101014)),
	def("emo", hairc(0x14101a, HairBob), tops(TopTee, 0x14141a, 0xe0508a), patn(2), pantsc(0x1a1a22), shoesc(0xf4f4f4), bodyc(BodySlim)),
	def("skater", hatc(HatCap, 0x3a3a46), tops(TopTee, 0xf0f0e8, 0xf0f0e8), pantsc(0x4a6a8a), shoesc(0xd04040), bodyc(BodySlim)),
	def("rapero_cadena", hatc(HatCap, 0x1a1a20), tops(TopTank, 0xf4f4f4, 0xf4f4f4), pantsc(0x2a2a34), suspc(0xf0c820), bodyc(BodyBig), skinc(0x6b4630)),
	def("trapero", glass(GlassNeon), hairc(0xf040c0, HairMohawk), tops(TopHoodie, 0xe8f060, 0x40e0e0), pantsc(0x2a2a34), shoesc(0xf4f4f4)),
	def("reggaetonero", glass(GlassWhite), hatc(HatCap, 0xf4f4f4), tops(TopTee, 0x14141a, 0x14141a), pantsc(0xe8e8e0), suspc(0xf0c820)),
	def("kpop_fan", fem(), hairc(0xf080c0, HairBob), tops(TopTee, 0xf0b0e0, 0x80c0f0), patn(1), bottomc(BotSkirt, 0xf0f0f8), carryc(HoldBalloons)),
	def("rockabilly", hairc(0x14101a, HairPompadour), tops(TopTee, 0xf4f4f4, 0xf4f4f4), pantsc(0x3a5a8a), shoesc(0x14141a), accentc(0xc82a2a)),
	def("hipster", hatc(HatBeanie, 0x4a4a52), beardc(BeardFull), glass(GlassRound), tops(TopLong, 0xa83a2a, 0x2a2a2a), patn(2), pantsc(0x2a3a5a)),
	def("indie", hairs(HairMessy), glass(GlassRound), tops(TopLong, 0x8a8a6a, 0x8a8a6a), pantsc(0x3a3a46), bodyc(BodySlim)),
	def("fan_eurovision", carryc(HoldFlag), tops(TopTee, 0xc0c4d0, 0xf0c820), patn(1), pantsc(0xc0c4d0), glass(GlassNeon)),
	def("karaoke_star", carryc(HoldMic), tops(TopSuit, 0x8a2aa8, 0xf0e060), pantsc(0x8a2aa8), hairs(HairPompadour)),
	def("saxofonista", carryc(HoldSax), hatc(HatFedora, 0x2a2a34), tops(TopLong, 0x3a3a46, 0x3a3a46), pantsc(0x2a2a34), glass(GlassShades)),
	def("imitador_rey", hairc(0x14101a, HairPompadour), glass(GlassShades), beardc(BeardSideburns), tops(TopSuit, 0xf6f6fa, 0xe8c840), accentc(0xe8c840), pantsc(0xf6f6fa), carryc(HoldMic)),
	def("aleman_sandalias", hatc(HatBucket, 0xc8b88a), tops(TopTee, 0xf0f0e0, 0xf0f0e0), bottomc(BotShorts, 0xb8a878), shoesc(0x8a5a2a), skinc(0xf7d9c0), beardc(BeardMustache), bodyc(BodyBig)),
	def("britanico_cangrejo", skinc(0xe05a40), tops(TopTank, 0xf4f4f4, 0xf4f4f4), bottomc(BotShorts, 0x3a5ab8), carryc(HoldBeer), bodyc(BodyBig)),
	def("turista_rinonera", hatc(HatBucket, 0xe8e0c0), tops(TopHawaiian, 0xe0a030, 0xf8f8f0), patn(1), bottomc(BotShorts, 0xc8b888), carryc(HoldCamera), glass(GlassShades), suspc(0x3a3a46)),
	def("mochilero", hatc(HatBucket, 0x6a8a4a), tops(TopTee, 0xe8e0d0, 0xe8e0d0), bottomc(BotShorts, 0x6a5a3a), capec(0x3a6a3a), hairs(HairMessy), beardc(BeardFull)),
	def("surfista", hairc(0xe8d070, HairLong), tops(TopBare, 0xe8b48a, 0xe8b48a), bottomc(BotTrunks, 0x2ab0e0), skinc(0xe8b48a), bodyc(BodyBig), stat(-1)),
	def("socorrista", hatc(HatCap, 0xc82a2a), tops(TopTank, 0xf0d020, 0xf0d020), bottomc(BotShorts, 0xc82a2a), bodyc(BodyBig), stat(-1), glass(GlassShades)),
	def("monitor_zumba", fem(), hairc(0xf06080, HairPony), tops(TopTank, 0x40f0c0, 0x40f0c0), pantsc(0xf040c0), bodyc(BodyNormal), shoesc(0xf4f4f4)),
	def("yoga", fem(), hairs(HairBun), tops(TopTank, 0xa0d0b0, 0xa0d0b0), pantsc(0x4a4a8a), bodyc(BodySlim)),
	def("vendedor_cocos", hatc(HatBucket, 0xd8c070), tops(TopHawaiian, 0x40a050, 0xf8f8f0), patn(1), bottomc(BotShorts, 0xe8e0c0), carryc(HoldPlant)),
	def("padre_hawaiano", tops(TopHawaiian, 0xe05a4a, 0xf8f8f0), patn(1), bottomc(BotShorts, 0x5a8ab0), shoesc(0x8a5a2a), beardc(BeardMustache), bodyc(BodyFat)),
	def("palo_selfie", carryc(HoldSelfie), tops(TopTee, 0xf06080, 0xf06080), glass(GlassShades), hairs(HairComb)),
	def("oficinista", tops(TopSuit, 0x3a3a4a, 0xf4f4f4), accentc(0x2a60c8), pantsc(0x3a3a4a), carryc(HoldBriefcase), hairs(HairComb)),
	def("jefe_insoportable", tops(TopSuit, 0x1a1a24, 0xf4f4f4), accentc(0xc82a2a), pantsc(0x1a1a24), bodyc(BodyBig), glass(GlassNone), hairc(0x5a5a5e, HairComb)),
	def("becario", tops(TopTee, 0xf4f4f4, 0xf4f4f4), pantsc(0x4a5a7a), carryc(HoldBigCup), bodyc(BodySlim), glass(GlassRound)),
	def("programador", tops(TopHoodie, 0x2a2a34, 0x2a2a34), glass(GlassRound), hairs(HairMessy), pantsc(0x3a3a46), bodyc(BodySlim)),
	def("guru_motivacional", carryc(HoldMic), tops(TopSuit, 0x2a2a34, 0xf4f4f4), accentc(0xe8c840), pantsc(0x2a2a34), hairs(HairComb)),
	def("vendedor_coches", tops(TopSuit, 0xc8b878, 0xf4f4f4), patn(3), accentc(0xd82a2a), pantsc(0xc8b878), hairs(HairComb), beardc(BeardMustache)),
	def("abogado", tops(TopSuit, 0x16161c, 0xf4f4f8), accentc(0x8a2a2a), pantsc(0x16161c), carryc(HoldBriefcase), glass(GlassRound)),
	def("cientifico_loco", tops(TopLong, 0xf4f4f4, 0xf4f4f4), pantsc(0x3a3a46), hairc(0xf4f4f4, HairMessy), glass(GlassBig), stat(1)),
	def("medico", tops(TopLong, 0xf4f4f4, 0xf4f4f4), pantsc(0x3a8ab0), suspc(0x3a3a46), hairs(HairShort)),
	def("bombero", hatc(HatHardhat, 0xc82a2a), tops(TopVest, 0xf0c820, 0xe0e0e0), pantsc(0x2a2a30), bodyc(BodyBig), shoesc(0x14141a)),
	def("repartidor_pizza", hatc(HatCap, 0xc82a2a), tops(TopTee, 0xc82a2a, 0xf4f4f4), pantsc(0x2a2a34), carryc(HoldPizza)),
	def("taxista", hatc(HatCap, 0x2a2a34), tops(TopVest, 0xf0d020, 0xf0d020), pantsc(0x2a2a34), beardc(BeardMustache)),
	def("cocinero_furioso", hatc(HatChef, 0xf4f4f8), tops(TopLong, 0xf4f4f8, 0xf4f4f8), pantsc(0x4a4a52), carryc(HoldMachete), beardc(BeardMustache), bodyc(BodyBig)),
	def("fontanero", hatc(HatCap, 0x3a5ab8), tops(TopLong, 0xd0d0d8, 0xd0d0d8), pantsc(0x3a5ab8), suspc(0x3a5ab8), shoesc(0x3a2a1a), beardc(BeardMustache)),
	def("jardinero", hatc(HatBucket, 0xd8c070), tops(TopTee, 0xe8e0c8, 0xe8e0c8), pantsc(0x4a6a3a), suspc(0x4a6a3a), carryc(HoldBroom)),
	def("cura", hairs(HairBald), tops(TopRobe, 0x14141a, 0xf4f4f4), shoesc(0x14141a)),
	def("mago_cumple", hatc(HatWizard, 0x4a2a8a), tops(TopRobe, 0x4a2a8a, 0xf0e060), beardc(BeardFull), hairc(0xe8e8e8, HairLong), carryc(HoldBalloons)),
	def("payaso_hosco", maskc(0xf4f4f0, MaskClown), hairc(0xf06020, HairCurly), tops(TopLong, 0xe0508a, 0x3a8ae0), patn(1), pantsc(0xe0508a), carryc(HoldBalloons), bodyc(BodyBig)),
	def("inspector_hacienda", tops(TopSuit, 0x6a6a74, 0xf4f4f4), accentc(0x3a3a46), pantsc(0x6a6a74), carryc(HoldBriefcase), glass(GlassRound), hairc(0x6a6a6e, HairComb)),
	def("notario", tops(TopSuit, 0x2a2a3a, 0xf4f4f4), accentc(0x8a2a2a), pantsc(0x2a2a3a), glass(GlassRound), hairc(0xd8d8d8, HairComb), beardc(BeardMustache)),
	def("exterminador", maskc(0x6a6a74, MaskPlain), tops(TopLong, 0xf0d020, 0xf0d020), pantsc(0xf0d020), shoesc(0x3a3a3a), carryc(HoldVacuum)),
	def("detective_gabardina", hatc(HatFedora, 0x6a5a4a), tops(TopRobe, 0xb89a6a, 0xb89a6a), beardc(BeardStubble), hairs(HairShort)),
	def("benidorm", tops(TopTank, 0xf4f4f4, 0xf4f4f4), bottomc(BotShorts, 0xe8e0c8), shoesc(0x8a5a2a), bodyc(BodyFat), skinc(0xd9a07a), beardc(BeardMustache), hairs(HairBald)),
	def("samurai", hatc(HatSamurai, 0x2a2a34), tops(TopRobe, 0x8a2a2a, 0x14141a), carryc(HoldMachete), hairs(HairBun), beardc(BeardMustache)),
	def("ninja", hatc(HatNinja, 0x14141a), tops(TopLong, 0x14141a, 0x14141a), pantsc(0x14141a), shoesc(0x14141a), carryc(HoldMachete), skinc(0xf0d8b0)),
	def("sumo", hairs(HairBun), tops(TopBare, 0xf0c8a0, 0xf0c8a0), bottomc(BotTrunks, 0xe8e0d0), skinc(0xf0c8a0), bodyc(BodyGiant), stat(-1)),
	def("vikingo_casco", hatc(HatViking, 0xa8a8b0), beardc(BeardFull), hairc(0xc8602a, HairLong), tops(TopVest, 0x7a5a3a, 0x7a5a3a), pantsc(0x5a4a3a), bodyc(BodyBig)),
	def("escoces", hatc(HatBeret, 0x2a3a8a), tops(TopLong, 0xf4f4f0, 0xf4f4f0), bottomc(BotSkirt, 0x2a6a3a), patn(2), beardc(BeardFull), hairc(0xc8602a, HairShort)),
	def("duende_irlandes", hatc(HatTopHat, 0x2a8a3a), tops(TopSuit, 0x2a8a3a, 0xf0f0e0), pantsc(0x2a8a3a), beardc(BeardFull), hairc(0xd05a20, HairShort), stat(3)),
	def("mariachi", hatc(HatSombrero, 0x14141a), tops(TopSuit, 0x14141a, 0xe8e8e8), accentc(0xc0c0c8), pantsc(0x14141a), beardc(BeardMustache)),
	def("tanguero", hatc(HatFedora, 0x14141a), tops(TopSuit, 0x14141a, 0xf4f4f4), accentc(0xc82a2a), pantsc(0x14141a), hairs(HairComb)),
	def("gaucho", hatc(HatCowboy, 0x4a3a2a), tops(TopRobe, 0x8a5a3a, 0xc8a050), pantsc(0xe8e0c8), beardc(BeardMustache)),
	def("carnaval_brasil", hatc(HatCrown, 0xff40a0), tops(TopSwim, 0xff40a0, 0xf0d020), bottomc(BotTrunks, 0xff40a0), fem(), hairs(HairLong), skinc(0xc98a5e)),
	def("bailarin_samba", hatc(HatBunny, 0xf0d020), tops(TopSwim, 0x40d0a0, 0xf0d020), bottomc(BotTrunks, 0x40d0a0), hairs(HairBun), fem()),
	def("turista_australiano", hatc(HatCowboy, 0xb89a60), tops(TopLong, 0xc8b888, 0xc8b888), bottomc(BotShorts, 0xb8a878), shoesc(0x5a3a1a), beardc(BeardStubble)),
	def("astronauta", hatc(HatHelmet, 0xf0f0f4), tops(TopArmor, 0xf0f0f4, 0xc82a2a), pantsc(0xf0f0f4), shoesc(0xc0c4cc), bodyc(BodyBig)),
	def("alien_turista", skinc(0x6ad04a), headc(HeadBig), eyesc(EyesAlien), hairs(HairBald), tops(TopHawaiian, 0xe05a4a, 0xf8f8f0), patn(1), bottomc(BotShorts, 0xf0e0b0), carryc(HoldCamera), bodyc(BodySlim)),
	def("portero_hockey", maskc(0xf0f0e8, MaskHockey), tops(TopLong, 0x2a4ab8, 0xf4f4f4), patn(2), pantsc(0x2a4ab8), bodyc(BodyBig)),
	def("faraon", hatc(HatCrown, 0xe8c040), tops(TopBare, 0xe8c040, 0xe8c040), bottomc(BotSkirt, 0xf4f4f0), skinc(0xc89a6e), glass(GlassShades), beardc(BeardGoatee)),
	def("pirata", hatc(HatPirate, 0x14141a), tops(TopLong, 0xe0d8c0, 0x8a2a2a), pantsc(0x3a2a1a), beardc(BeardFull), carryc(HoldMachete)),
	def("monje", hairs(HairBald), tops(TopRobe, 0x8a5a2a, 0x8a5a2a), shoesc(0x6a4a2a), bodyc(BodyBig)),
	def("rey_medieval", hatc(HatCrown, 0xf0c030), tops(TopRobe, 0xa02a2a, 0xf4f4f0), beardc(BeardFull), hairc(0xc8c8c8, HairLong), bodyc(BodyBig), carryc(HoldTrophy)),
	def("cavernicola", hairs(HairMessy), tops(TopTank, 0xb8802a, 0x5a3a1a), patn(3), bottomc(BotShorts, 0xb8802a), beardc(BeardFull), carryc(HoldBroom), bodyc(BodyBig), shoesc(0x6a4a2a)),
	def("romano", tops(TopRobe, 0xf4f4f0, 0xc82a2a), shoesc(0xe8302a), hairs(HairShort), hatc(HatCrown, 0x3a8a3a)),
	def("caballero_moto", hatc(HatMoto, 0x2a2a34), tops(TopArmor, 0x9aa2b0, 0x4a5262), pantsc(0x4a5262), bodyc(BodyBig)),
	def("napoleonico", hatc(HatPirate, 0x14141a), tops(TopSuit, 0x2a3a8a, 0xf4f4f0), accentc(0xe8c040), pantsc(0xf4f4f0), hairs(HairShort)),
	def("robot_victoriano", bodyc(BodyRobot), hatc(HatTopHat, 0x8a929e)),
	def("noble_frances", hatc(HatWig, 0xf0f0f0), tops(TopSuit, 0xe880b0, 0xf4f4f4), accentc(0xe8c040), pantsc(0xe880b0), beardc(BeardMustache)),
	def("monje_dj", hairs(HairBald), tops(TopRobe, 0x3a2a1a, 0x3a2a1a), glass(GlassNeon), carryc(HoldMic)),
	def("campesino_antorcha", hatc(HatBucket, 0xc8b070), tops(TopTee, 0xd8d0b8, 0xd8d0b8), pantsc(0x6a5a3a), carryc(HoldTorch), beardc(BeardStubble)),
	def("dinosaurio_hinchable", costume(0x4aa840), hatc(HatDino, 0x4aa840), bodyc(BodyGiant), stat(-2)),
	def("tiburon_mascota", costume(0x7a8aa0), hatc(HatShark, 0x7a8aa0), bodyc(BodyFat)),
	def("pollo_gigante", costume(0xf6f4ee), hatc(HatChicken, 0xf6f4ee), bodyc(BodyGiant), shoesc(0xf0b020), stat(-2)),
	def("hombre_platano", costume(0xf8d838), hairs(HairBald), bodyc(BodyFat), stat(-2)),
	def("unicornio_resaca", costume(0xf4eef8), hatc(HatUnicorn, 0xf4eef8), bodyc(BodyBig)),
	def("conejo_siniestro", costume(0xf4f4f4), hatc(HatBunny, 0xf4f4f4), carryc(HoldMachete), bodyc(BodyBig), stat(-1)),
	def("muneco_nieve", costume(0xf4f8ff), hatc(HatTopHat, 0x14141a), bodyc(BodyGiant), stat(-2), accentc(0xe86a20)),
	def("power_ranger", maskc(0xc82a2a, MaskPlain), tops(TopLong, 0xc82a2a, 0xf4f4f4), pantsc(0xc82a2a), shoesc(0xf4f4f4), bodyc(BodyNormal)),
	def("superheroe_saldo", capec(0x6a6a74), tops(TopTank, 0x3a3a46, 0xf0d020), pantsc(0xc82a2a), glass(GlassShades), bodyc(BodyFat), stat(2)),
	def("hombre_cubo", costume(0x7a8290), hatc(HatBucket, 0x7a8290), bodyc(BodyGiant)),
	def("bebe_gigante", skinc(0xf4c9a4), hairs(HairBald), headc(HeadBig), tops(TopBare, 0xf4c9a4, 0xf4c9a4), bottomc(BotTrunks, 0xf4f4f4), bodyc(BodyFat), stat(3), shoesc(0xf4f4f4)),
	def("conspiranoico", hatc(HatHelmet, 0xc8ccd4), glass(GlassRound), tops(TopHoodie, 0x4a4a52, 0x4a4a52), pantsc(0x3a3a46), beardc(BeardFull)),
	def("terraplanista", carryc(HoldSign), tops(TopTee, 0xe0e0e0, 0x2a2a30), pantsc(0x4a4a52), beardc(BeardStubble)),
	def("coach_masculinidad", glass(GlassShades), tops(TopTank, 0x14141a, 0x14141a), pantsc(0x2a2a34), bodyc(BodyBig), stat(-1), skinc(0xe8b48a)),
	def("guru_espiritual", tops(TopRobe, 0xf4f0e8, 0xf4f0e8), beardc(BeardFull), hairc(0xf4f4f4, HairLong), carryc(HoldBroom)),
	def("guru_cripto", tops(TopSuit, 0x2a2a34, 0xf4f4f8), accentc(0xf0c030), pantsc(0x2a2a34), carryc(HoldTrophy), glass(GlassShades)),
	def("fanatico_ovnis", hatc(HatHelmet, 0xc8ccd4), tops(TopTee, 0x3a8a4a, 0x3a8a4a), pantsc(0x4a4a52), glass(GlassBig), bodyc(BodySlim)),
	def("gamer_encerrado", skinc(0xf8e8dc), glass(GlassRound), tops(TopHoodie, 0x4a2a8a, 0x4a2a8a), pantsc(0x2a2a34), hairs(HairMessy), bodyc(BodyFat), carryc(HoldNone)),
	def("otaku", hairc(0x4a8ae8, HairSpiky), tops(TopTee, 0xf4f4f4, 0xe04a8a), patn(1), pantsc(0x2a2a34), carryc(HoldMachete), glass(GlassRound), bodyc(BodySlim)),
	def("abuelo_pijama", hairs(HairBald), tops(TopLong, 0x6a8ab0, 0xf4f4f4), patn(2), pantsc(0x6a8ab0), shoesc(0xc88a8a), bodyc(BodySlim), skinc(0xf0d0b8), stat(1)),
	def("abuela_breakdance", fem(), hairc(0xe8e8ec, HairBun), tops(TopDress, 0x8a6aa8, 0x8a6aa8), shoesc(0xf4f4f4), stat(2), bodyc(BodySlim)),
	def("vecino_bata", tops(TopRobe, 0xc88a9a, 0xc88a9a), shoesc(0x8a5a5a), hairs(HairMessy), beardc(BeardStubble), hatc(HatBeanie, 0x6a6aa8)),
	def("senor_basura", tops(TopTee, 0xd8d0c0, 0xd8d0c0), bottomc(BotShorts, 0x6a6a74), shoesc(0x8a5a2a), carryc(HoldBriefcase), beardc(BeardStubble)),
	def("tecnico_fibra", hatc(HatHardhat, 0xf0a020), tops(TopVest, 0xf08020, 0xe0e0e0), pantsc(0x3a3a46), carryc(HoldBriefcase)),
	def("megafonista", carryc(HoldMegaphone), tops(TopTee, 0xf0d020, 0xf0d020), pantsc(0x3a3a46), hatc(HatCap, 0xc82a2a), bodyc(BodyBig)),
	def("chandal_supremo", tops(TopHoodie, 0x14141a, 0xf0d020), pantsc(0x14141a), shoesc(0xf0d020), hatc(HatCap, 0x14141a), suspc(0xf0d020), glass(GlassShades)),
	def("muerte_vacaciones", maskc(0xf0f0e8, MaskSkull), tops(TopHawaiian, 0xe04a8a, 0x2a2a2a), patn(1), bottomc(BotShorts, 0x14141a), hatc(HatBucket, 0xf8f8f0), carryc(HoldBroom)),
	def("ninja_borracho", hatc(HatNinja, 0x2a2a34), tops(TopLong, 0x2a2a34, 0x2a2a34), pantsc(0x2a2a34), carryc(HoldBeer), skinc(0xf0d8b0)),
	def("abuelo_raver", hairs(HairBald), glass(GlassNeon), tops(TopHoodie, 0x40f080, 0xf040c0), pantsc(0x2a2a34), skinc(0xf0d0b8), accentc(0xf040c0), stat(1)),
	def("hombre_globos", carryc(HoldBalloons), tops(TopTee, 0xe04a4a, 0xf4f4f4), pantsc(0x3a5ab8), hairs(HairCurly)),
	def("vampiro_vegano", skinc(0xf0e8ec), hairc(0x14101a, HairComb), tops(TopSuit, 0x14101a, 0xf4f4f8), accentc(0x3a8a3a), pantsc(0x14101a), capec(0x6a1020), carryc(HoldPlant)),
	def("turista_interdim", skinc(0x8a5ad0), headc(HeadBig), eyesc(EyesAlien), hairs(HairBald), tops(TopHawaiian, 0x40e0c0, 0xf8f8f0), patn(1), bottomc(BotShorts, 0xf0e0b0), glass(GlassNone), bodyc(BodySlim)),
	def("policia_fiesta", hatc(HatPolice, 0x1b2a5e), tops(TopHawaiian, 0xf08020, 0xf8f8f0), patn(1), bottomc(BotShorts, 0x1b2a5e), carryc(HoldBeer), glass(GlassShades)),
	def("ultimo_superviviente", hatc(HatCap, 0x4a4a3a), tops(TopLong, 0x5a5a4a, 0x5a5a4a), pantsc(0x3a3a2a), carryc(HoldTorch), beardc(BeardFull), bodyc(BodyBig), glass(GlassShades)),
	def("hombre_aspiradora", carryc(HoldVacuum), tops(TopTee, 0x6a8ab0, 0x6a8ab0), pantsc(0x4a4a52), hatc(HatBeanie, 0xc82a2a)),
	def("hombre_semaforo", hatc(HatTV, 0x30303a), tops(TopTee, 0xf0d020, 0x2a2a30), pantsc(0x2a2a30), patn(2)),
	def("cabeza_tele", hatc(HatTV, 0x30303a), tops(TopSuit, 0x2a2a34, 0xf4f4f8), accentc(0xc82a2a), pantsc(0x2a2a34)),
	def("hombre_planta", carryc(HoldPlant), tops(TopTee, 0x5a8a4a, 0x5a8a4a), pantsc(0x4a5a3a), hatc(HatBucket, 0x5a8a4a)),
	def("senor_trofeo", carryc(HoldTrophy), tops(TopSuit, 0x3a2a1a, 0xf4f4f0), accentc(0xe8c040), pantsc(0x3a2a1a), bodyc(BodyBig)),
	def("cartel_se_vende", carryc(HoldSign), tops(TopTee, 0xd8d0c0, 0xd8d0c0), pantsc(0x5a4a3a), hairs(HairShort)),
	def("traje_latas", costume(0xb8bcc8), hatc(HatBucket, 0xb8bcc8), patn(1), bodyc(BodyBig), stat(-1)),
	def("pizza_infierno", hatc(HatHorns, 0), tops(TopTee, 0xc82a2a, 0xf4f4f4), pantsc(0x2a0a0a), carryc(HoldPizza), skinc(0xc0302a)),
	def("fan_ovni_foil", hatc(HatHelmet, 0xd0d4dc), tops(TopTee, 0xf0f0f0, 0x3aa040), pantsc(0x3a3a46), glass(GlassBig)),
	def("moderno_bigote", hatc(HatFedora, 0x3a3a46), beardc(BeardMustache), glass(GlassRound), tops(TopLong, 0xa84a3a, 0x2a2a2a), patn(2), pantsc(0x2a3a5a), hairs(HairComb)),
	def("jugador_pelota", hatc(HatCap, 0xc82a2a), tops(TopTee, 0xc82a2a, 0xf4f4f4), patn(2), bottomc(BotShorts, 0xf4f4f4), shoesc(0xf4f4f4)),
	def("gaitero", hatc(HatBeret, 0x2a6a3a), tops(TopLong, 0xf0f0e0, 0xf0f0e0), bottomc(BotSkirt, 0x8a2a2a), patn(2), beardc(BeardMustache), carryc(HoldBroom)),
	def("flamenco", hatc(HatFedora, 0x14141a), tops(TopLong, 0xf4f4f4, 0xc82a2a), pantsc(0x14141a), hairs(HairComb), beardc(BeardMustache)),
	def("chino_con_paraguas", hatc(HatBucket, 0xd8c070), tops(TopRobe, 0xe0a030, 0xc82a2a), skinc(0xf0d890), hairs(HairPony), beardc(BeardMustache)),
	def("fan_anime", fem(), hairc(0xf080c0, HairBuns), tops(TopDress, 0x6a8ae8, 0xf4f4f4), skinc(0xf8e0d0), stat(1), bodyc(BodySlim)),
	def("chica_neon", fem(), hairc(0x40f0e0, HairBob), tops(TopTank, 0xf040c0, 0x40f0e0), bottomc(BotSkirt, 0x6a2aa8), glass(GlassNeon)),
	def("senora_peluqueria", fem(), hairc(0xd8d0e0, HairCurly), tops(TopDress, 0x8ac8a8, 0x8ac8a8), bodyc(BodyBig), glass(GlassRound)),
	def("campista", hatc(HatBucket, 0x6a8a4a), tops(TopVest, 0x6a8a4a, 0xe8d8a0), bottomc(BotShorts, 0x8a7a4a), carryc(HoldTorch), glass(GlassRound)),
	def("esqueleto_fiesta", maskc(0xf0f0e8, MaskSkull), tops(TopLong, 0x14141a, 0xf0f0e8), patn(2), pantsc(0x14141a), shoesc(0x14141a)),
	def("calabaza", hatc(HatPumpkin, 0xe8802a), tops(TopLong, 0x4a2a1a, 0x4a2a1a), pantsc(0x2a1a10), bodyc(BodyBig)),
	def("fantasma_sabana", maskc(0xf4f4f4, MaskGhost), tops(TopRobe, 0xf4f4f4, 0xf4f4f4)),
	def("zombi_fiesta", skinc(0x8ab088), hairs(HairMessy), hairc(0x3a3a2a, HairMessy), tops(TopLong, 0x6a5a4a, 0x6a5a4a), pantsc(0x4a4a3a), beardc(BeardStubble)),
	def("bailarin_poseido", skinc(0xd8e8d8), hairs(HairMessy), eyesc(EyesAlien), tops(TopTank, 0xf4f4f4, 0xf4f4f4), pantsc(0x2a2a34), bodyc(BodySlim)),
}

func init() {
	famousPresets = append(famousPresets, asFuncs(fictionalPresets)...)
}

// holdName gives the meta.json name of a held item (for exported characters).
func holdName(kind int) string {
	for n, k := range carryNames {
		if k == kind {
			return n
		}
	}
	return ""
}
