package main

import "math/rand"

// ---------------------------------------------------------------------------
// The "star cast": groups of well known characters and bands that are put into
// every party on purpose, so the crowd is not only random people.
// ---------------------------------------------------------------------------

var starPresets = []presetEntry{
	// Cervantes
	def("don_quijote", hatc(HatHelmet, 0xc8c8d0), beardc(BeardGoatee), hairc(0x8a8a90, HairShort), tops(TopArmor, 0xa8acb8, 0xa8acb8), pantsc(0x6a6a74), shoesc(0x4a3a2a), bodyc(BodySlim), stat(-3), carryc(HoldFlag)),
	def("sancho_panza", hatc(HatCap, 0x6a4a2a), beardc(BeardStubble), hairc(0x2a2018, HairShort), tops(TopVest, 0x8a6a3a, 0xe8d8b0), pantsc(0x6a5a3a), shoesc(0x4a3a2a), bodyc(BodyFat), stat(3)),
	// Sonic
	def("tails", skinc(0xf8e0c0), hairc(0xf0a030, HairSpiky), headc(HeadBig), tops(TopLong, 0xf0a030, 0xf0a030), pantsc(0xf0a030), shoesc(0xd82a2a), bodyc(BodyNormal), stat(3)),
	def("knuckles", skinc(0xf0c8a0), hairc(0xd02020, HairSpiky), headc(HeadBig), tops(TopLong, 0xd02020, 0xd02020), pantsc(0xd02020), shoesc(0x2a9a3a), bodyc(BodyBig), stat(1)),
	// Dragon Ball
	def("krilin", hairs(HairBald), tops(TopLong, 0xf08a20, 0xf08a20), pantsc(0xf08a20), shoesc(0x2a50c0), stat(4), headc(HeadBig)),
	def("piccolo", skinc(0x60b848), hatc(HatTurban, 0xf4f4f4), tops(TopLong, 0x7a2aa8, 0x7a2aa8), pantsc(0x7a2aa8), capec(0xf4f4f4), shoesc(0x6a3a1a), bodyc(BodyBig), stat(-2)),
	def("bulma", fem(), hairc(0x40c0d0, HairBob), tops(TopDress, 0xf070b0, 0xf4f4f4), shoesc(0xf4f4f4)),
	def("maestro_roshi", hairs(HairBald), glass(GlassShades), beardc(BeardFull), tops(TopHawaiian, 0x3aa0d0, 0xf4f4f4), bottomc(BotShorts, 0x3aa0d0), hatc(HatNone, 0), carryc(HoldNone), stat(3)),
	def("gohan", hairc(0x14101a, HairSpiky), tops(TopLong, 0x7a2aa8, 0x7a2aa8), pantsc(0x7a2aa8), shoesc(0x2a50c0), stat(4), headc(HeadBig)),
	// Spanish TV
	def("pantera_rosa", skinc(0xf080b0), hairc(0xf080b0, HairBald), earsn(2), headc(HeadBig), tops(TopBare, 0xf080b0, 0xf080b0), bottomc(BotShorts, 0xf080b0), shoesc(0xf080b0), bodyc(BodyNormal), stat(1)),
	def("espinete", skinc(0xf0c090), hairc(0x7a4a2a, HairSpiky), headc(HeadBig), tops(TopTee, 0xf0e040, 0xf0e040), bottomc(BotShorts, 0x3a6ac0), shoesc(0xd82a2a), stat(4)),
	def("don_pimpon", hatc(HatBowler, 0x3a2a1a), beardc(BeardMustache), hairc(0x3a2a1a, HairShort), tops(TopSuit, 0xc83a2a, 0xf4e8a0), accentc(0x2a2a8a), pantsc(0x3a2a1a), bodyc(BodyBig), stat(-1)),
	// The Beatles
	def("lennon", hairc(0x5a3a22, HairLong), glass(GlassRound), tops(TopSuit, 0xf4f4f0, 0xf4f4f0), pantsc(0xf4f4f0), accentc(0xf4f4f0), shoesc(0xf4f4f0)),
	def("mccartney", hairc(0x3a2414, HairBowl), tops(TopSuit, 0x1a1a22, 0xf4f4f4), pantsc(0x1a1a22), accentc(0x1a1a22), shoesc(0x14141a)),
	def("harrison", hairc(0x1a1410, HairBowl), tops(TopSuit, 0x3a3a46, 0xf4f4f4), pantsc(0x3a3a46), accentc(0x3a3a46), shoesc(0x14141a)),
	def("ringo", hairc(0x3a2414, HairBowl), beardc(BeardStubble), tops(TopSuit, 0x1a1a22, 0xf4f4f4), pantsc(0x1a1a22), accentc(0x1a1a22), shoesc(0x14141a)),
	// Sum 41
	def("deryck", hairc(0xb89a50, HairSpiky), tops(TopTank, 0x14141a, 0x14141a), pantsc(0x2a2a30), shoesc(0xf4f4f4), suspc(0xd02828)),
	def("dave_baksh", hairc(0x14101a, HairMessy), hatc(HatBeanie, 0x14141a), tops(TopTee, 0xd02828, 0xd02828), pantsc(0x2a2a30), shoesc(0x14141a)),
	def("jason_mc", hairc(0x3a2414, HairShort), tops(TopTee, 0x2a2a30, 0xf4f4f4), patn(2), pantsc(0x3a4a6a), shoesc(0xf4f4f4)),
	def("frank_zummo", hairc(0x14101a, HairMohawk), tops(TopTank, 0xf4f4f4, 0xf4f4f4), pantsc(0x14141a), shoesc(0x14141a)),
	// Blink-182
	def("mark_hoppus", hairc(0x14101a, HairShort), tops(TopTee, 0x14141a, 0x14141a), pantsc(0x3a4a6a), shoesc(0xf4f4f4), beardc(BeardStubble)),
	def("tom_delonge", hairc(0x3a2414, HairMessy), hatc(HatCap, 0x14141a), tops(TopTee, 0xe8e8f0, 0x14141a), patn(2), bottomc(BotShorts, 0x6a5a3a), shoesc(0xf4f4f4)),
	def("travis_barker", hairs(HairBald), tops(TopBare, 0xe0b890, 0xe0b890), pantsc(0x14141a), shoesc(0x14141a), skinc(0xe0b890), beardc(BeardGoatee), bodyc(BodyBig)),
}

func init() {
	famousPresets = append(famousPresets, asFuncs(starPresets)...)
	for k, v := range map[string]string{
		"don_quijote": "Don Quijote", "sancho_panza": "Sancho Panza", "tails": "Tails", "knuckles": "Knuckles", "sonic": "Sonic", "krilin": "Krilin", "piccolo": "Piccolo", "bulma": "Bulma", "maestro_roshi": "Maestro Roshi", "gohan": "Gohan",
		"pantera_rosa": "La Pantera Rosa", "espinete": "Espinete", "don_pimpon": "Don Pimpón",
		"lennon": "John Lennon", "mccartney": "Paul McCartney", "harrison": "George Harrison", "ringo": "Ringo Starr",
		"deryck": "Deryck (Sum 41)", "dave_baksh": "Dave (Sum 41)", "jason_mc": "Jason (Sum 41)", "frank_zummo": "Frank (Sum 41)",
		"mark_hoppus": "Mark (Blink-182)", "tom_delonge": "Tom (Blink-182)", "travis_barker": "Travis (Blink-182)",
		"homer": "Homer", "bart": "Bart", "marge": "Marge", "lisa": "Lisa", "cartman": "Cartman", "kenny": "Kenny", "kyle": "Kyle", "stan": "Stan",
		"fry": "Fry", "leela": "Leela", "bender": "Bender", "farnsworth": "Profesor Farnsworth", "zoidberg": "Zoidberg", "elvis": "Elvis",
	} {
		displayNameOverrides[k] = v
	}
}

var starCache map[string]func(*rand.Rand) CharSpec

// presetByName finds a character preset by its internal name.
func presetByName(name string) func(*rand.Rand) CharSpec {
	if starCache == nil {
		starCache = map[string]func(*rand.Rand) CharSpec{}
		probe := rand.New(rand.NewSource(1))
		for _, f := range famousPresets {
			if n := f(probe).Name; n != "" {
				if _, dup := starCache[n]; !dup {
					starCache[n] = f
				}
			}
		}
	}
	return starCache[name]
}

var starGroups = [][]string{
	{"homer", "marge", "bart", "lisa"},
	{"fry", "leela", "bender", "farnsworth", "zoidberg"},
	{"luffy", "zoro", "nami", "usopp", "sanji", "chopper"},
	{"cartman", "kenny", "kyle", "stan"},
	{"sonic", "tails", "knuckles"},
	{"goku", "vegeta", "krilin", "piccolo", "bulma", "gohan"},
	{"mortadelo", "filemon", "zipi", "zape"},
	{"pantera_rosa", "espinete", "don_pimpon"},
	{"wally", "wenda", "mago_blancabarba", "odlaw"},
}

// spawnStars puts a few famous groups and one band into the party.
func (g *Game) spawnStars() {
	order := g.rng.Perm(len(starGroups))
	n := 3 + g.rng.Intn(3)
	for _, gi := range order[:n] {
		var leader *Guest
		base := g.lvl.RandomRoam(g.rng)
		for i, slug := range starGroups[gi] {
			f := presetByName(slug)
			if f == nil || !allowedSlug(slug) {
				continue
			}
			if g.rng.Intn(8) == 0 && i > 1 { // sometimes a member is missing
				continue
			}
			spec := f(g.rng)
			pos := g.lvl.NearestFree(Vec{base.X + float64(i%3)*18 - 18, base.Y + float64(i/3)*16})
			gu := &Guest{pos: pos, dir: Dir(g.rng.Intn(4)), cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 2, role: i % 2, keep: true}
			if leader == nil {
				leader = gu
			} else {
				gu.leader = leader
			}
			g.guests = append(g.guests, gu)
		}
	}
	// a few more famous faces wandering around on their own
	pool := append([]string(nil), specialPool...)
	g.rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	nSpecial := 5 + g.rng.Intn(4)
	for _, slug := range pool {
		if nSpecial == 0 {
			break
		}
		if f := presetByName(slug); f != nil && allowedSlug(slug) {
			spec := f(g.rng)
			g.guests = append(g.guests, &Guest{pos: g.lvl.RandomRoam(g.rng), dir: Dir(g.rng.Intn(4)), cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 2, role: nSpecial % 2, keep: true})
			nSpecial--
		}
	}
	// special characters the player bought turn up on their own as well
	if len(castSlugs) > 0 {
		for k := 0; k < 3; k++ {
			slug := castSlugs[g.rng.Intn(len(castSlugs))]
			if f := presetByName(slug); f != nil {
				spec := f(g.rng)
				gu := &Guest{pos: g.lvl.RandomRoam(g.rng), dir: Dir(g.rng.Intn(4)), cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 2, role: k % 2, keep: true}
				g.guests = append(g.guests, gu)
			}
		}
	}
	g.spawnBand()
}
