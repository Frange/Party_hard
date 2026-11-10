package main

import "math/rand"

// ---------------------------------------------------------------------------
// The "star cast": groups of well known characters and bands that are put into
// every party on purpose, so the crowd is not only random people.
// ---------------------------------------------------------------------------

var starPresets = []presetEntry{
	// Cervantes (hats, beards, lance, shield and waistcoat in ptf_art.go)
	def("don_quijote", skinc(0xe8c09a), hatc(HatNone, 0), hairc(0xb4b4b8, HairShort), tops(TopArmor, 0x8c98ac, 0x8c98ac), pantsc(0x7a8aa4), shoesc(0x5a3a20), bodyc(BodySlim), stat(-3)),
	def("sancho_panza", skinc(0xe0a880), hatc(HatNone, 0), hairc(0x6a4426, HairShort), tops(TopLong, 0xf0ead8, 0xf0ead8), pantsc(0x7a5a34), shoesc(0xc8b080), bodyc(BodyFat), stat(3)),
	// Sonic
	def("tails", skinc(0xf08a1c), hairc(0xf08a1c, HairBald), headc(HeadHuge), tops(TopLong, 0xf08a1c, 0xf08a1c), pantsc(0xf08a1c), shoesc(0xd82a2a), stat(3)),
	def("knuckles", skinc(0xf0c8a0), hairc(0xd02020, HairSpiky), headc(HeadBig), tops(TopLong, 0xd02020, 0xd02020), pantsc(0xd02020), shoesc(0x2a9a3a), bodyc(BodyBig), stat(1)),
	// Dragon Ball
	def("krilin", skinc(0xf4c8a0), hairs(HairBald), tops(TopLong, 0xf08a20, 0xf08a20), pantsc(0xf08a20), shoesc(0x2a50c0), stat(4), headc(HeadBig)),
	def("piccolo", skinc(0x60b848), hatc(HatTurban, 0xf4f4f4), tops(TopTank, 0x7a2aa8, 0x7a2aa8), pantsc(0x7a2aa8), capec(0xf4f4f4), shoesc(0x6a3a1a), bodyc(BodyBig), stat(-2)),
	def("bulma", fem(), skinc(0xf8d8b8), hairc(0x40c0d0, HairBob), tops(TopDress, 0xf070b0, 0xf4f4f4), shoesc(0xf4f4f4)),
	def("maestro_roshi", skinc(0xf0c8a0), hairc(0xf4f4f0, HairBald), glass(GlassShades), beardc(BeardFull), tops(TopHawaiian, 0x3aa0d0, 0xf4f4f4), bottomc(BotShorts, 0x3aa0d0), shoesc(0x8a5a2a), hatc(HatNone, 0), carryc(HoldNone), stat(3)),
	def("gohan", skinc(0xf4c8a0), hairc(0x14101a, HairSpiky), tops(TopLong, 0x7a2aa8, 0x7a2aa8), pantsc(0x7a2aa8), shoesc(0x6a3a1a), stat(4), headc(HeadBig)),
	// Spanish TV
	def("pantera_rosa", skinc(0xf080b0), hairc(0xf080b0, HairBald), earsn(2), headc(HeadBig), tops(TopBare, 0xf080b0, 0xf080b0), bottomc(BotShorts, 0xf080b0), shoesc(0xf080b0), bodyc(BodyNormal), stat(1)),
	def("espinete", skinc(0xf0c090), hairc(0x7a4a2a, HairSpiky), headc(HeadBig), tops(TopTee, 0xf0e040, 0xf0e040), bottomc(BotShorts, 0x3a6ac0), shoesc(0xd82a2a), stat(4)),
	def("don_pimpon", hatc(HatBowler, 0x3a2a1a), beardc(BeardMustache), hairc(0x3a2a1a, HairShort), tops(TopSuit, 0xc83a2a, 0xf4e8a0), accentc(0x2a2a8a), pantsc(0x3a2a1a), bodyc(BodyBig), stat(-1)),
	// The Beatles
	def("lennon", hairc(0x5a3a22, HairLong), glass(GlassRound), tops(TopSuit, 0xf4f4f0, 0xf4f4f0), pantsc(0xf4f4f0), accentc(0xf4f4f0), shoesc(0xf4f4f0)),
	def("mccartney", hairc(0x3a2414, HairBowl), tops(TopSuit, 0x1a1a22, 0xf4f4f4), pantsc(0x1a1a22), accentc(0x1a1a22), shoesc(0x14141a)),
	def("harrison", hairc(0x1a1410, HairBowl), tops(TopSuit, 0x3a3a46, 0xf4f4f4), pantsc(0x3a3a46), accentc(0x3a3a46), shoesc(0x14141a)),
	def("ringo", hairc(0x3a2414, HairBowl), beardc(BeardStubble), tops(TopSuit, 0x1a1a22, 0xf4f4f4), pantsc(0x1a1a22), accentc(0x1a1a22), shoesc(0x14141a)),
	// Sum 41 (style archetypes, the prints, spikes, belts and cans are drawn in pnk_art.go)
	def("deryck", skinc(0xf0c8a8), hairc(0x5a3a22, HairShort), hatc(HatCap, 0xf4f4f4), tops(TopTee, 0x16161c, 0x16161c), pantsc(0x3a5a8a), shoesc(0x14141a), carryc(HoldBeer)),
	def("dave_baksh", skinc(0x9a6440), hairc(0x14101a, HairShort), beardc(BeardGoatee), tops(TopTee, 0x16161c, 0x16161c), pantsc(0x4a4a52), shoesc(0x14141a), carryc(HoldBeer)),
	def("jason_mc", skinc(0xf0c8a8), hairc(0xf4d81c, HairShort), tops(TopTee, 0x16161c, 0x16161c), pantsc(0x6a6a4a), shoesc(0x2a2a30), carryc(HoldBeer)),
	def("frank_zummo", skinc(0xf4d0b4), hairc(0xe8501a, HairShort), tops(TopTee, 0xf4f4f4, 0xf4f4f4), pantsc(0x16161c), shoesc(0x14141a), carryc(HoldBeer)),
	// Blink-182 (skate tee, tattoos, mohawk and drumsticks in pnk_art.go)
	def("mark_hoppus", skinc(0xf0c8a8), hairc(0x14101a, HairShort), tops(TopTee, 0x3a4a6a, 0x3a4a6a), pantsc(0x2a3a5a), shoesc(0xf4f4f4), beardc(BeardStubble)),
	def("tom_delonge", skinc(0xecc4a0), hairc(0x3a2414, HairShort), tops(TopTee, 0xa82830, 0xa82830), bottomc(BotShorts, 0x6a5a3a), shoesc(0x14141a)),
	def("travis_barker", skinc(0xe0b890), hairc(0x18181e, HairBald), tops(TopTank, 0x16161c, 0x16161c), pantsc(0x14141a), shoesc(0x14141a), bodyc(BodyBig)),
}

func init() {
	famousPresets = append(famousPresets, asFuncs(starPresets)...)
	for k, v := range map[string]string{
		"don_quijote": "Don Quijote", "sancho_panza": "Sancho Panza", "tails": "Tails", "knuckles": "Knuckles", "sonic": "Sonic", "krilin": "Krilin", "piccolo": "Piccolo", "bulma": "Bulma", "maestro_roshi": "Maestro Roshi", "gohan": "Gohan",
		"pantera_rosa": "La Pantera Rosa", "espinete": "Espinete", "don_pimpon": "Don Pimpón",
		"lennon": "John Lennon", "mccartney": "Paul McCartney", "harrison": "George Harrison", "ringo": "Ringo Starr",
		"deryck": bandMemberName("deryck", "Sum 41", "cantante"), "dave_baksh": bandMemberName("dave_baksh", "Sum 41", "guitarra"),
		"jason_mc": bandMemberName("jason_mc", "Sum 41", "bajo"), "frank_zummo": bandMemberName("frank_zummo", "Sum 41", "batería"),
		"mark_hoppus": bandMemberName("mark_hoppus", "Blink-182", "bajo"), "tom_delonge": bandMemberName("tom_delonge", "Blink-182", "guitarra"),
		"travis_barker": bandMemberName("travis_barker", "Blink-182", "batería"), "elvis": "Elvis",
		"homer": "Homer", "bart": "Bart", "marge": "Marge", "lisa": "Lisa", "cartman": "Cartman", "kenny": "Kenny", "kyle": "Kyle", "stan": "Stan",
		"fry": "Fry", "leela": "Leela", "bender": "Bender", "farnsworth": "Profesor Farnsworth", "zoidberg": "Zoidberg",
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
		// the reward characters of the trophies are registered after the merge above (init order), so look at the
		// star list too: without this tr_verdugo, tr_rey_fiesta... could never be built (cards, wardrobe, visits)
		for _, e := range starPresets {
			if _, dup := starCache[e.Name]; !dup && e.Name != "" {
				starCache[e.Name] = e.F
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
	nSpecial = maxI(0, nSpecial-g.spawnBeanGang()) // a Fall Guys gang takes famous places (fallguys.go)
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
	g.spawnGoalVisitors()
	g.spawnBand()
}
