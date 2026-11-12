package main

import (
	"math/rand"
	"strings"
)

// ---------------------------------------------------------------------------
// GOALS / PERSONAJES. The main characters used to be bought in the shop; now they are won by KILLING a guest of that
// kind charKillsNeed (5) times (game.go -> charKill). Progress.CharKills counts the kills; the first one marks the
// card ELIMINADO (albumElim, face shown, "n/5"), the fifth sets albumKilled = CONSEGUIDO (usable in the wardrobe).
// Old saves: albumKilled meant "killed once" and it still means "won", so nothing they had is lost and no migration
// is needed (their CharKills is empty; a won card never asks for more kills). The catalogue is data: charCatalogSlugs
// (the 282 characters the shop used to sell, in the old shop order), plus the characters of the album (kids, special
// pool, groups) and the reward characters of the trophies, which unlock with their trophy.
//
// What "unlocked" means (charUnlocked): the card has the albumKilled bit, OR the old save owns "skin:<slug>" (bought in
// the shop before the redesign, or won with a trophy). Nothing is converted on disk: old purchases keep working because
// syncShop (shop.go) still reads Progress.Owned. An unlocked character can be worn from the wardrobe ("base" row) and,
// if he is a kid, he always comes to the party (kids.go) -- the same things owning him did before.
// ---------------------------------------------------------------------------

var charCatalogSlugs = []string{
	"don_quijote", "sancho_panza", "leeloo", "korben", "conan", "he_man", "marty_mcfly", "doc_brown", "la_mascara", "taz",
	"burro_shrek", "michael_jackson", "tortuga_leonardo", "tortuga_donatello", "tortuga_raphael", "tortuga_michelangelo", "alf",
	"johnny5", "inspector_gadget", "capitan_planeta", "picard", "johnny_bravo", "pedro_picapiedra", "wilma", "pablo_marmol",
	"betty", "sam_sam", "cesar", "diego", "pitufo", "papa_pitufo", "pitufina", "lagarto_juancho", "bob_esponja", "beavis",
	"butthead", "doraemon", "nobita", "guybrush", "stan_barcos", "elaine_marley", "lechuck", "voodoo_lady", "meathook",
	"pirata_importante", "jack_rey_calabaza", "la_calaca", "capitan_calavera", "esqueleto_rockero", "esqueleto_bailarin",
	"esqueleto_ejecutivo", "wally", "mortadelo", "filemon", "zipi", "zape", "mafalda", "tintin", "capitan_haddock", "popeye",
	"carpanta", "asterix", "obelix", "panoramix", "super_lopez", "espinete", "don_pimpon", "pantera_rosa", "homer", "bart", "marge",
	"lisa", "burns", "flanders", "cartman", "kenny", "kyle", "stan", "fry", "leela", "bender", "farnsworth", "zoidberg", "peter",
	"stewie", "rick", "morty", "sonic", "tails", "knuckles", "crash", "mario", "luigi", "wario", "waluigi", "donkey_kong", "bowser",
	"kirby", "pikachu", "link", "steve_cubos", "creeper", "lara_croft", "nathan_drake", "luffy", "zoro", "nami", "usopp", "sanji",
	"chopper", "nico_robin", "franky", "brook", "ace_fuego", "goku", "vegeta", "krilin", "piccolo", "bulma", "gohan", "naruto",
	"sailor_moon", "ash_pokemon", "elvis", "freddie", "marilyn", "charlie_chaplin", "einstein", "napoleon", "cleopatra",
	"julio_cesar", "dali", "van_gogh", "frida_kahlo", "mozart", "beethoven", "shakespeare", "colon_almirante", "isabel_la_catolica",
	"gandhi", "che", "juana_de_arco", "bruce_lee", "sherlock", "poirot", "zorro", "robin_hood", "peter_pan", "capitan_garfio",
	"tarzan", "sp_cantante", "sp_guitarra", "sp_bajo", "sp_bateria", "off_cantante", "off_guitarra", "off_bajo", "off_bateria",
	"met_cantante", "met_guitarra", "met_bajo", "met_bateria", "ps_cantante", "ps_guitarra", "ps_bajo", "ps_bateria", "bt_cantante",
	"bt_guitarra", "bt_bajo", "bt_bateria", "lp_cantante", "lp_guitarra", "lp_bajo", "lp_bateria", "des_cantante", "des_guitarra",
	"des_bajo", "des_bateria", "frank_zummo", "dave_baksh", "jason_mc", "tom_delonge", "lennon", "mccartney", "harrison", "ringo",
	"deryck", "mark_hoppus", "travis_barker", "woody", "buzz", "shrek", "minion", "gru", "elsa", "blancanieves", "caperucita_roja",
	"el_nota", "borat", "austin", "zoolander", "ace_ventura", "mr_bean", "wonka", "beetlejuice", "jack_sparrow", "harry_potter",
	"hermione", "dumbledore", "gandalf", "frodo", "legolas", "aragorn", "yoda", "chewbacca", "han_solo", "luke", "leia",
	"boba_fett", "mandaloriano", "vader", "trooper", "jedi", "kirk", "spock", "redshirt", "indy", "bond", "dracula", "santa",
	"homelander", "michael_scott", "dwight", "sheldon", "barney", "dr_house", "dexter", "heisenberg", "saul", "jon_nieve", "negan",
	"miercoles", "once", "jugador_456", "guardia_rosa", "batman", "superman", "spiderman", "ironman", "thor", "hulk",
	"capitan_america", "wonder_woman", "flash", "aquaman", "catwoman", "viuda_negra", "deadpool", "lobezno", "joker", "arlequina",
	"thanos", "doctor_extrano", "bane", "blade", "terminator", "robocop", "neo", "john_wick", "rambo", "rocky", "kratos",
	"jefe_maestro", "solid_snake", "doom_slayer", "agente_47", "leon_k", "duke", "tony_s", "tommy_s", "atracador_rojo",
	"motorista_fantasma",
}

// charKey: is this album key a character (not a VIP, not a killer)?
func charKey(key string) bool {
	return key != "" && !strings.HasPrefix(key, "vip:") && !strings.HasPrefix(key, "killer:") && !strings.HasPrefix(key, "cop:")
}

// charUnlocked: the player may use this character (killed one, bought one before, or a trophy reward).
func charUnlocked(album map[string]int, slug string) bool {
	return album[slug]&albumKilled != 0 || ownedID("skin:"+slug)
}

// charKillsNeed: kills of the same character that win him.
const charKillsNeed = 5

// charKill: the player killed a guest of this character slug. It counts the kill, marks the card ELIMINADO the first
// time and won (albumKilled) at the charKillsNeed-th; a won character just keeps counting.
func (g *Game) charKill(slug string) {
	key := albumKeyOf(slug)
	if key == "" {
		return
	}
	if g.progress.CharKills == nil {
		g.progress.CharKills = map[string]int{}
	}
	if g.progress.CharKills[key] < 1<<20 {
		g.progress.CharKills[key]++
	}
	bits := albumSeen | albumElim
	if g.progress.CharKills[key] >= charKillsNeed {
		bits |= albumKilled // won: albumMark announces the new card and opens the wardrobe
	}
	if g.progress.Album[key]&bits != bits {
		g.albumMark(key, bits) // saves
		return
	}
	saveProgress(g.progress)
}

// charKillsLeft: kills still needed to win this character (0 when won).
func (g *Game) charKillsLeft(c albumCard) int {
	if g.albumUnlocked(c) {
		return 0
	}
	return maxI(1, charKillsNeed-g.progress.CharKills[c.key])
}

// charDetailLine: the line under the grid for the selected character: name, his own description and what is left.
func (g *Game) charDetailLine(c albumCard) string {
	if !g.albumUnlocked(c) && g.albumBits(c)&(albumSeen|albumElim) == 0 {
		return "??? - " + g.charStatus(c)
	}
	return tr(c.nameES) + " - " + tr(charDesc(c.slug)) + " - " + g.charStatus(c) // charStatus: goal_detail.go
}

// charSyncOwned: copy the characters unlocked by kills into the shop's lookup, so the wardrobe, the kids and the
// riders (everything that asks ownedID("skin:...")) see them. It never touches Progress.Owned (nothing is written).
func charSyncOwned(album map[string]int) {
	for key, bits := range album {
		if charKey(key) && bits&albumKilled != 0 {
			shopOwned["skin:"+key] = true
		}
	}
}

// charCards: the PERSONAJES cards (kids first, then the catalogue, then the album extras, then the trophy rewards).
func charCards() []albumCard {
	var out []albumCard
	seen := map[string]bool{}
	probe := rand.New(rand.NewSource(1))
	add := func(slug, role, trophy string) {
		if seen[slug] {
			return
		}
		f := presetByName(slug)
		if f == nil {
			return
		}
		seen[slug] = true
		spec := f(probe)
		c := albumCard{key: slug, nameES: nameFromSpec(&spec), cardES: albumShortName[slug], roleES: role, slug: slug, trophyID: trophy}
		if band := bandOfMember(slug); band != "" { // a musician shown by his name: "James Hetfiel" / Metalica
			c.roleCard = band
		} else if team := footballTeam(slug); team != "" { // a footballer shown by his name: "Lionel Mesi" / Argentina (futbol_art.go)
			c.roleCard = team
		} else if game := beanRole(slug); game != "" { // a Fall Guys bean: "Flan Gallina" / Fall Guys (fallguys.go)
			c.roleCard = game
		} else if i := strings.Index(c.nameES, " · "); i > 0 { // a band member: "Metalica · cantante" -> Cantante / Metalica
			part := c.nameES[i+len(" · "):]
			c.cardES, c.roleCard = strings.ToUpper(part[:1])+part[1:], c.nameES[:i]
			if part == "bateria" {
				c.cardES = "Batería"
			}
		}
		out = append(out, c)
	}
	for _, k := range kidSlugs {
		add(k, "Peque travieso", "")
	}
	for _, s := range charCatalogSlugs {
		add(s, "Invitado famoso", "")
	}
	for _, s := range specialPool {
		add(s, "Invitado famoso", "")
	}
	for _, grp := range starGroups {
		for _, s := range grp {
			add(s, "Invitado famoso", "")
		}
	}
	for _, t := range trophyList { // the reward characters unlock with their trophy
		if t.reward.tab == shopChars && strings.HasPrefix(t.reward.arg, "tr_") {
			add(t.reward.arg, "Premio de trofeo", t.id)
		}
	}
	for _, r := range legendExtras() {
		if r.tab == shopChars {
			add(r.arg, "Premio de trofeo", legendTrophy)
		}
	}
	return out
}

// charVisitors: how many characters that are not unlocked yet come to every party on purpose. Most of the catalogue has
// no other way in (the shop-only ones are never random guests, and with an assets folder the random guests come from
// the files), so without this the goal "kill one of each" could not be done. They are picked at random among the
// locked ones, so every party offers new faces until the collection is complete.
const charVisitors = 4

// charLockedPool: the catalogue characters (not the kids, who arrive by themselves, nor the trophy rewards) that are
// not unlocked in this save.
func (g *Game) charLockedPool() []string {
	var out []string
	for _, c := range albumCards() {
		if c.kind() != cardChar || c.trophyID != "" || isKidSlug(c.slug) || charUnlocked(g.progress.Album, c.slug) {
			continue
		}
		out = append(out, c.slug)
	}
	return out
}

// spawnGoalVisitors puts up to charVisitors locked characters in the party, wandering on their own like the other
// named guests (keep: true). Each is built from its preset, as the characters bought in the old shop were.
func (g *Game) spawnGoalVisitors() {
	pool := g.charLockedPool()
	g.rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	n := 0
	for _, slug := range pool {
		if n >= charVisitors {
			break
		}
		f := presetByName(slug)
		if f == nil || g.guestNamed(slug) {
			continue
		}
		spec := f(g.rng)
		g.guests = append(g.guests, &Guest{pos: g.lvl.RandomRoam(g.rng), dir: Dir(g.rng.Intn(4)), cs: newCharSet(&spec), name: nameFromSpec(&spec), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 2, role: n % 2, keep: true})
		n++
	}
}

// guestNamed: is a guest with this character slug already in the party?
func (g *Game) guestNamed(slug string) bool {
	for _, gu := range g.guests {
		if gu != nil && gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Name == slug {
			return true
		}
	}
	return false
}

// wardCharacters: every character that can be worn (id = slug), unlocked or not; the wardrobe filters by ownership.
func wardCharacters() []wardOption {
	var out []wardOption
	for _, c := range albumCards() { // built once
		if c.kind() == cardChar {
			out = append(out, wardOption{c.slug, c.nameES})
		}
	}
	return out
}
