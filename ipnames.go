package main

// ---------------------------------------------------------------------------
// Names of the famous characters (owner's policy, 2026-11-11). Every visible name of a real person
// (musicians, footballers, historical figures...) or of a character/brand of an existing work carries
// ONE letter changed, added or removed, so it is still recognisable but never the exact name
// ("Clara Croft", "Buttman", "Zonic", "Lionel Mesi"...). Proper names are NEVER translated: the same
// in Spanish and English (common words around them, "Cazafantasmos" or "· cantante", still are).
// The slugs (folders, progress keys, album keys) do NOT change: players have saved games.
// Tests: ipnames_test.go (TestProperNamesNotTranslated, TestNoOriginalIPNamesVisible).
// ---------------------------------------------------------------------------

// ipNames: the visible names of the characters whose name used to come from the slug itself (the
// others are set where the character is defined: stars.go, cast2.go, presets4.go, bands.go...).
var ipNames = map[string]string{
	"freddy":         "Freddi",
	"jason":          "Jasson",
	"michael":        "Michel",
	"ghostface":      "Ghostfase",
	"leatherface":    "Leatherfaze",
	"pinhead":        "Pinhed",
	"chucky":         "Chucki",
	"pennywise":      "Pennywize",
	"mulder":         "Molder",
	"scully":         "Skully",
	"burns":          "Burnes",
	"flanders":       "Flandes",
	"peter":          "Piter",
	"lois":           "Loiz",
	"stewie":         "Stewy",
	"vader":          "Vater",
	"trooper":        "Truper",
	"chewbacca":      "Chewbaca",
	"yoda":           "Yoga",
	"jedi":           "Yedi",
	"leia":           "Leya",
	"kirk":           "Kirck",
	"spock":          "Spook",
	"indy":           "Indi",
	"bond":           "Blond",
	"mario":          "Marrio",
	"luigi":          "Luiggi",
	"freddie":        "Fredie",
	"big_boss":       "Big Bos",
	"hulk":           "Bulk",
	"spiderman":      "Spaiderman",
	"batman":         "Buttman",
	"superman":       "Souperman",
	"joker":          "Poker",
	"arlequina":      "Arlekina",
	"deadpool":       "Deadpoll",
	"lobezno":        "Lobesno",
	"thor":           "Thorr",
	"ironman":        "Ironmon",
	"thanos":         "Thanoz",
	"doctor_extrano": "Doctor Estraño",
	"blade":          "Blayde",
	"bane":           "Vane",
	"terminator":     "Termimator",
	"robocop":        "Rotocop",
	"neo":            "Neon",
	"john_wick":      "John Wock",
	"rambo":          "Rombo",
	"rocky":          "Rocki",
	"jack_sparrow":   "Jack Esparrow",
	"wonka":          "Wonca",
	"beetlejuice":    "Bettlejuice",
	"shrek":          "Shreck",
	"gru":            "Grux",
	"minion":         "Minyon",
	"el_nota":        "El Notta",
	"borat":          "Barat",
	"austin":         "Austyn",
	"zoolander":      "Zoolender",
	"ace_ventura":    "Ace Bentura",
	"mr_bean":        "Mr Beans",
	"han_solo":       "Ham Solo",
	"luke":           "Lucke",
	"boba_fett":      "Boba Feta",
	"mandaloriano":   "Mandaloreano",
	"dexter":         "Dextor",
	"heisenberg":     "Heisenburg",
	"saul":           "Soul",
	"tony_s":         "Tony Z",
	"tommy_s":        "Tomy S",
	"rick":           "Rik",
	"morty":          "Morti",
	"jon_nieve":      "Jon Nieves",
	"negan":          "Megan",
	"miercoles":      "Miérkoles",
	"once":           "Onze",
	"homelander":     "Homelender",
	"michael_scott":  "Michael Scoot",
	"dwight":         "Dweight",
	"sheldon":        "Shelton",
	"barney":         "Barnie",
	"dr_house":       "Dr Hause",
	"crash":          "Krash",
	"wario":          "Vario",
	"waluigi":        "Valuigi",
	"donkey_kong":    "Donkey Kongo",
	"bowser":         "Bowzer",
	"lara_croft":     "Clara Croft",
	"nathan_drake":   "Nathan Drako",
	"jefe_maestro":   "Jefe Maeztro",
	"kratos":         "Cratos",
	"leon_k":         "Leon Q",
	"agente_47":      "Agente 48",
	"duke":           "Duko",
	"doom_slayer":    "Boom Slayer",
	"solid_snake":    "Solid Snack",
	"kirby":          "Kirbi",
	"pikachu":        "Picachu",
	"creeper":        "Kreeper",
	"popeye":         "Popey",
	"bluto":          "Bruto",
	"einstein":       "Ainstein",
	"marilyn":        "Marilin",
	"mozart":         "Mozzart",
	"beethoven":      "Beethoben",
	"shakespeare":    "Shakespere",
	"gandhi":         "Gandi",
	"sherlock":       "Sherlok",
	"zorro":          "Zurro",
	"tarzan":         "Tarzon",
	"catwoman":       "Catwuman",
	"flash":          "Flasch",
	"aquaman":        "Acuaman",
	"goku":           "Goko",
	"vegeta":         "Vejeta",
	"naruto":         "Narutto",
	"woody":          "Woodie",
	"buzz":           "Buz",
	"elsa":           "Elza",
	"link":           "Lynk",
	"gandalf":        "Gandolf",
	"frodo":          "Fredo",
	"hermione":       "Hermiona",
	"dumbledore":     "Dumbledor",
	"legolas":        "Legolaz",
	"aragorn":        "Aragon",
}

// properNameSlugs: the characters whose visible name is a proper name (kept identical in English).
var properNameSlugs = []string{
	"freddy", "jason", "michael", "ghostface", "leatherface", "pinhead", "chucky", "pennywise", "mulder", "scully",
	"elvis", "et", "homer", "bart", "marge", "lisa", "burns", "flanders", "fry", "leela", "bender", "farnsworth",
	"zoidberg", "peter", "lois", "stewie", "cartman", "kenny", "kyle", "stan", "vader", "trooper", "chewbacca", "yoda",
	"jedi", "leia", "kirk", "spock", "indy", "bond", "mario", "luigi", "freddie", "lp_ritmo", "lp_corista", "leeloo",
	"korben", "conan", "he_man", "marty_mcfly", "doc_brown", "la_mascara", "michael_jackson", "tortuga_leonardo",
	"tortuga_donatello", "tortuga_raphael", "tortuga_michelangelo", "alf", "johnny5", "inspector_gadget", "capitan_planeta",
	"picard", "johnny_bravo", "pedro_picapiedra", "wilma", "pablo_marmol", "betty", "pitufo", "papa_pitufo", "pitufina",
	"lagarto_juancho", "bob_esponja", "beavis", "butthead", "doraemon", "nobita", "guybrush", "lechuck", "ron", "cell",
	"judia_azul", "judia_animadora", "judia_lima", "judia_pelofuego", "judia_gallina", "judia_hamburguesa",
	"judia_zanahoria", "judia_callejera", "judia_pinguino", "cristiano_ronaldo", "lionel_messi", "lamine_yamal",
	"kylian_mbappe", "big_boss", "hulk", "spiderman", "batman", "superman", "joker", "arlequina", "deadpool", "lobezno",
	"thor", "ironman", "thanos", "doctor_extrano", "blade", "bane", "terminator", "robocop", "neo", "john_wick", "rambo",
	"rocky", "jack_sparrow", "wonka", "beetlejuice", "shrek", "gru", "minion", "el_nota", "borat", "austin", "zoolander",
	"ace_ventura", "mr_bean", "napoleon_d", "han_solo", "luke", "boba_fett", "mandaloriano", "dexter", "heisenberg",
	"saul", "tony_s", "tommy_s", "rick", "morty", "jon_nieve", "negan", "miercoles", "once", "homelander",
	"michael_scott", "dwight", "sheldon", "barney", "dr_house", "sonic", "crash", "wario", "waluigi", "donkey_kong",
	"bowser", "lara_croft", "nathan_drake", "jefe_maestro", "kratos", "leon_k", "duke", "doom_slayer", "solid_snake",
	"kirby", "pikachu", "creeper", "wally", "odlaw", "mago_blancabarba", "mortadelo", "filemon", "super_lopez", "zipi",
	"zape", "rompetechos", "carpanta", "mafalda", "tintin", "capitan_haddock", "popeye", "bluto", "luffy", "zoro", "nami",
	"usopp", "sanji", "chopper", "nico_robin", "franky", "brook", "ace_fuego", "asterix", "obelix", "panoramix",
	"abraracurcix", "julio_cesar", "cleopatra", "napoleon", "einstein", "van_gogh", "frida_kahlo", "dali",
	"charlie_chaplin", "marilyn", "mozart", "beethoven", "shakespeare", "colon_almirante", "isabel_la_catolica", "gandhi",
	"che", "sherlock", "poirot", "zorro", "peter_pan", "capitan_garfio", "tarzan", "jane_selva", "juana_de_arco",
	"bruce_lee", "wonder_woman", "catwoman", "flash", "capitan_america", "viuda_negra", "aquaman", "goku", "vegeta",
	"naruto", "sailor_moon", "ash_pokemon", "woody", "buzz", "elsa", "link", "gandalf", "frodo", "harry_potter",
	"hermione", "dumbledore", "legolas", "aragorn", "maggie", "tails", "knuckles", "krilin", "piccolo", "bulma",
	"maestro_roshi", "gohan", "pantera_rosa", "espinete", "don_pimpon", "lennon", "mccartney", "harrison", "ringo",
	"deryck", "dave_baksh", "jason_mc", "frank_zummo", "mark_hoppus", "tom_delonge", "travis_barker", "sp_cantante",
	"sp_bajo", "sp_bateria", "off_cantante", "off_guitarra", "met_cantante", "met_guitarra", "met_bajo", "met_bateria",
	"bt_cantante", "bt_guitarra", "bt_bajo", "lp_guitarra", "lp_bajo", "des_cantante", "don_quijote", "sancho_panza",
	"robin_hood", "blancanieves", "caperucita_roja", "dracula", "wenda", "steve_cubos", "sam_sam", "cesar", "diego",
}

func init() {
	for k, v := range ipNames {
		displayNameOverrides[k] = v
	}
}

// properNames: every proper name shown in the game (characters, bands and the Flan Guys line).
func properNames() []string {
	var out []string
	for _, s := range properNameSlugs {
		out = append(out, autoDisplayName(s))
	}
	for _, b := range bandList {
		out = append(out, b.name)
	}
	for _, d := range f1Drivers { // the F1 event (events5.go)
		out = append(out, d.name)
	}
	return append(out, beanGame, rangersName, murreyName)
}

// rangersName: the 5-star police squad (shop card, goals, news); murreyName: the talking skull on his stake
// (murray.go, fin_murray.go). Proper names with one letter changed, the same in Spanish and English; the slugs
// and keys ("rangers", "police:4", "murray") stay.
const (
	rangersName = "Pawer Rangers"
	murreyName  = "Murrey"
)

// keepProperNames makes the English table leave the proper names as they are (an exact entry wins
// over the word-by-word replacements: "Capitán Planetta" never becomes "Captain Planetta").
func keepProperNames() {
	for _, n := range properNames() {
		langDataEN[n] = n
	}
}
