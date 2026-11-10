package main

import (
	"math/rand"
)

// ---------------------------------------------------------------------------
// GOALS / ASESINOS. The killers used to be bought in the shop (58 of them); now the catalogue is 21 of the best and
// most varied (killerCatalog) plus the 4 that trophies give (killerTrophy), and they are won by KILLING them when
// they come to a party. The other 37 presets stay in presets2.go (the editor still exports them) but they never come
// to a party any more: killerLegacy maps each one to the most similar kept killer, so a save that had bought it keeps
// that right (it counts as that kept killer being unlocked: killerGot reads Progress.Owned, nothing is rewritten).
//
// Which killer comes (pickKiller): one of those not unlocked yet; when all are unlocked, any of the allowed ones.
// When one comes (killerreg.go) is unchanged -- ~0.18 of the parties, never two in a row -- except for this: after you
// kill a killer, the next one is guaranteed within killerNextWithin parties (Progress.KillerWait; the very next party
// can never have one because of the no-back-to-back rule, so in practice it is the second).
// ---------------------------------------------------------------------------

// killerNextWithin: parties after a killer is killed by which the next (different) one has come.
const killerNextWithin = 2

type killerInfo struct {
	slug, es, en, cardES, cardEN string
}

// killerCatalog: the 23 killers of the game (21 + the endoskeleton + the beak doctor), in the order of the ASESINOS tab.
var killerCatalog = []killerInfo{
	{"michael", "Michael", "Michael", "", ""},
	{"jason", "Jason", "Jason", "", ""},
	{"ghostface", "Ghostface", "Ghostface", "", ""},
	{"leatherface", "Cara de Cuero", "Leather Mask", "", ""},
	{"freddy", "Freddy", "Freddy", "", ""},
	{"chucky", "Muñeco diabólico", "Devil doll", "Muñeco", "Doll"},
	{"mimo", "El Mimo", "The Mime", "", ""},
	{"demonio", "El Demonio", "The Demon", "", ""},
	{"enfermera", "Enfermera asesina", "Killer nurse", "Enfermera", "Nurse"},
	{"calabaza", "Cabeza de calabaza", "Pumpkin head", "Calabaza", "Pumpkin"},
	{"espantapajaros", "Espantapájaros", "Scarecrow", "", ""},
	{"santa_psicopata", "Santa psicópata", "Psycho Santa", "Santa", "Santa"},
	{"ninja_asesino", "Ninja asesino", "Killer ninja", "Ninja", "Ninja"},
	{"vampiro", "Vampiro", "Vampire", "", ""},
	{"bebe_gigante", "Bebé gigante", "Giant baby", "", ""},
	{"hombre_lobo", "Hombre lobo", "Werewolf", "", ""},
	{"muerte", "La Muerte", "Death", "", ""},
	{"conejo_asesino", "Conejo asesino", "Killer bunny", "Conejo", "Bunny"},
	{"astronauta_asesino", "Astronauta asesino", "Killer astronaut", "Astronauta", "Astronaut"},
	{"robot_asesino", "Robot asesino", "Killer robot", "Robot", "Robot"},
	{"dj_maldito", "DJ maldito", "Cursed DJ", "", ""},
	{termSlug, "El Exterminador", "The Exterminator", "Exterminador", "Exterminator"}, // terminator_killer.go
	{plagueSlug, "El Doctor Pico", "The Beak Doctor", "Doctor Pico", "Beak Doctor"},   // plague_killer.go
}

// killerTrophy: the killers that trophies give (trophies.go). They join the pool when the trophy is won and their
// card is unlocked by it. trophy = the id of the trophy.
var killerTrophy = []struct {
	killerInfo
	trophy string
}{
	{killerInfo{"tr_k_sombra", "La Sombra", "The Shadow", "", ""}, "t_ki1"},
	{killerInfo{"tr_k_rey_calavera", "El Rey Calavera", "The Skull King", "Rey Calavera", "Skull King"}, "t_ki15"},
	{killerInfo{"tr_k_vendedor", "El Vendedor de Seguros", "Insurance Salesman", "Vendedor", "Salesman"}, "t_b30"},
	{killerInfo{"tr_k_dios_oscuro", "El Dios Oscuro", "The Dark God", "Dios Oscuro", "Dark God"}, "t_pts8k"},
}

// killerLegacy: the 37 killers that left the catalogue -> the kept killer that is most like each one.
var killerLegacy = map[string]string{
	"pinhead": "freddy", "pennywise": "mimo", "impermeable": "michael", "caballo": "conejo_asesino", "carnicero": "leatherface",
	"doctor_loco": "enfermera", "lenador": "leatherface", "cirujano": "enfermera", "bruja": "vampiro", "momia": "vampiro",
	"cocinero_asesino": "leatherface", "payaso_triste": "mimo", "cazador": "jason", "buzo": "astronauta_asesino",
	"gladiador": "ninja_asesino", "pirata_fantasma": "ghostface", "oso_fiesta": "bebe_gigante", "novia_cadaver": "enfermera",
	"chef_frances": "leatherface", "payaso_circo": "mimo", "vaquero_fantasma": "ghostface", "bailarin_ballet": "mimo",
	"taxista_loco": "michael", "oficinista_zombi": "robot_asesino", "arbitro_asesino": "michael", "torero_maldito": "ninja_asesino",
	"jardinero_loco": "leatherface", "karateka": "ninja_asesino", "domador": "santa_psicopata", "dentista": "enfermera",
	"pescador": "jason", "brujo": "muerte", "caballero_negro": "muerte", "samurai_oscuro": "ninja_asesino",
	"marinero_maldito": "ghostface", "disco_vampiro": "vampiro", "hombre_saco": "espantapajaros", "fantasma_opera": "ghostface",
	"esqueleto": "muerte", "muneca": "chucky",
}

func killerInCatalog(slug string) bool {
	for _, k := range killerCatalog {
		if k.slug == slug {
			return true
		}
	}
	return false
}

func killerTrophyOf(slug string) (trophy string, ok bool) {
	for _, k := range killerTrophy {
		if k.slug == slug {
			return k.trophy, true
		}
	}
	return "", false
}

// killerAllowed: may this killer come to a party? The 21 always; a trophy killer once the trophy gave it.
func killerAllowed(name string) bool {
	if killerInCatalog(name) {
		return true
	}
	if _, ok := killerTrophyOf(name); ok {
		return ownedID("killer:" + name)
	}
	return false
}

// killerOwnedBefore: the save bought this killer, or one that maps to it, in the shop before the redesign.
func killerOwnedBefore(slug string) bool {
	if ownedID("killer:" + slug) {
		return true
	}
	for old, to := range killerLegacy {
		if to == slug && ownedID("killer:"+old) {
			return true
		}
	}
	return false
}

// killerGot: the killer is unlocked (killed one, bought one before, or a trophy reward).
func (g *Game) killerGot(slug string) bool {
	if g.progress.Album["killer:"+slug]&albumKilled != 0 {
		return true
	}
	if _, ok := killerTrophyOf(slug); ok {
		return ownedID("killer:" + slug)
	}
	return killerOwnedBefore(slug)
}

// killerMissing: how many of the 21 killers that come to every save are not unlocked yet.
func (g *Game) killerMissing() int {
	n := 0
	for _, k := range killerCatalog {
		if !g.killerGot(k.slug) {
			n++
		}
	}
	return n
}

// killerSpecOf: the look of a killer preset (cards, previews).
func killerSpecOf(slug string) CharSpec {
	for _, e := range killerPresets {
		if e.Name == slug {
			return e.F(rand.New(rand.NewSource(3)))
		}
	}
	return playerSpec()
}

// killerWeaponRole: the line under the name on a card.
func killerWeaponRole(w int) string {
	switch w {
	case HoldChainsaw:
		return "Motosierra"
	case HoldClaw:
		return "Garras"
	case HoldMachete:
		return "Machete"
	case HoldTorch:
		return "Antorcha"
	case HoldBriefcase:
		return "Maletín"
	case HoldSaber:
		return "Espada"
	case HoldRifle:
		return "Rifle"
	}
	return "Cuchillo"
}

// killerCards: the ASESINOS cards (the 21, then the trophy ones).
func killerCards() []albumCard {
	var out []albumCard
	add := func(k killerInfo, trophy string) {
		sp := killerSpecOf(k.slug)
		role := killerWeaponRole(sp.Weapon)
		if r := plagueRole(k.slug); r != "" {
			role = r // his staff (plague_killer.go)
		}
		if trophy != "" {
			role = "Premio de trofeo"
		}
		out = append(out, albumCard{key: "killer:" + k.slug, nameES: k.es, cardES: k.cardES, roleES: role, slug: k.slug, killer: true, trophyID: trophy})
	}
	for _, k := range killerCatalog {
		add(k, "")
	}
	for _, k := range killerTrophy {
		add(k.killerInfo, k.trophy)
	}
	return out
}

// killerChoose: which preset comes. missing = the allowed killers not unlocked yet (empty: all are unlocked).
func (g *Game) killerPool() (all, missing []presetEntry) {
	for _, k := range killerPresets {
		if !killerAllowed(k.Name) || !g.termFits(k.Name) || !g.plagueFits(k.Name) {
			continue
		}
		all = append(all, k)
		if !g.killerGot(k.Name) {
			missing = append(missing, k)
		}
	}
	return
}

// killerDefeated: a killer died by the player's hand or a trap. It unlocks his card and starts the guarantee of the
// next one (only while some are still missing).
func (g *Game) killerDefeated(name string) {
	if killerInCatalog(name) {
		g.albumMark("killer:"+name, albumKilled)
	}
	if g.killerMissing() > 0 {
		g.progress.KillerWait = killerNextWithin
		saveProgress(g.progress)
	}
}

// killerWaitStep runs when a party loads (not STEALTH): it counts the parties since the last killer was killed and
// reports whether this party must have the next killer. prev = the previous party had one (never two in a row, so the
// wait holds at 1 until a party without a previous killer comes).
func (g *Game) killerWaitStep(prev bool) bool {
	w := g.progress.KillerWait
	switch {
	case w <= 0:
		return false
	case w > 1:
		g.progress.KillerWait = w - 1
		saveProgress(g.progress)
		return false
	}
	return !prev
}

// killerArrived: a killer stepped out of his vehicle: the guarantee is paid.
func (g *Game) killerArrived() {
	if g.progress.KillerWait != 0 {
		g.progress.KillerWait = 0
		saveProgress(g.progress)
	}
}

// killerPresetNamed: is this the name of one of the built-in killers (the 58 presets)? Anything else in the assets
// folder is a killer the user drew himself, and it is never filtered.
func killerPresetNamed(name string) bool {
	for _, e := range killerPresets {
		if e.Name == name {
			return true
		}
	}
	return false
}

// pickAssetKiller: the same rule as the built-in pool, over the killers of the assets folder (the exported copies of
// the built-in ones included: they follow the catalogue too). nil = none usable, the caller falls back to the presets.
func (g *Game) pickAssetKiller() *CharSet {
	var all, missing []*CharSet
	for _, cs := range g.as.killerCS {
		if cs == nil || cs.Spec == nil {
			continue
		}
		n := cs.Spec.Name
		if killerPresetNamed(n) && !killerAllowed(n) {
			continue // a killer that left the catalogue
		}
		if !g.termFits(n) || !g.plagueFits(n) {
			continue // the endoskeleton never comes to the island (terminator_killer.go)
		}
		all = append(all, cs)
		if killerInCatalog(n) && !g.killerGot(n) {
			missing = append(missing, cs)
		}
	}
	pool := all
	if len(missing) > 0 {
		pool = missing
	}
	if len(pool) == 0 {
		return nil
	}
	return pool[killerDice().Intn(len(pool))]
}
