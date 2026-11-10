package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// ---------------------------------------------------------------------------
// The end-of-level NEWSPAPER (the only screen of ModeLevelDone; it is drawn by newspaper_draw.go).
// A front page "EL DIARIO DE LA JUERGA" with a headline made from the statistics and a short news
// story in the third person, composed from sentence templates. Every choice is deterministic: the same
// facts give the same paper (paperSeed), a different party gives a different one. Nothing here is
// stored; no real names are used. Texts are Spanish keys (tr() translates them, lang_newspaper.go).
// ---------------------------------------------------------------------------

// paperFacts: everything the paper knows about the party.
type paperFacts struct {
	level                          string // "7" or "EXTRA"
	levelN                         int
	kills, clean, seen, cuffed     int
	chainMax                       int
	knife, poison, traps, blast    int
	slasher, medic, camera, jailed bool
	units                          uint
	two, challenge                 bool
	secs                           int
	seed                           int64
	animalsDead, animalsAlive      int             // animals killed in this party / still alive at the end
	island, dj                     bool            // the island level / a DJ was playing
	venue                          string          // icon of the party (cake, hockey, disco70): its own brief (venues_text.go)
	events                         map[string]bool // events of the party that took place (newspaper_events.go)
	avoid                          map[string]bool // what the last parties already printed (Spanish keys): left out when possible
	// for the template news (newspaper_tpl.go)
	theme                        string          // the kind of level (hintThemeOf; "island" for the Jurassic island)
	rare                         map[string]bool // big set pieces that started in the party (ufo, zombie, rex...)
	famous, famousDead           []ptPerson      // famous guests of the party, with the names the game shows
	animalKinds, animalDeadKinds []string        // kinds of the animals of the party (all / dead)
	tplAvoid                     map[string]bool // template ids printed by the last parties of the session
}

func (g *Game) paperFactsNow() paperFacts {
	r := &g.run
	f := paperFacts{level: "EXTRA", levelN: g.levelIdx + 1, two: g.is2P(), challenge: g.challenge, secs: int(g.endTime)}
	if g.levelIdx >= 0 && g.levelIdx < campaignN {
		f.level = fmt.Sprint(g.levelIdx + 1)
	}
	for i := 0; i < 2; i++ {
		f.kills += r.killsOf(i)
		f.clean += r.cleanKills[i]
		f.seen += r.seen[i]
		f.cuffed += r.cuffed[i]
		f.chainMax = maxI(f.chainMax, r.chainMax[i])
	}
	f.knife = r.total(kcKnife) + r.total(kcNone)
	f.poison, f.traps, f.blast = r.total(kcPoison), r.total(kcTrap), r.total(kcBlast)
	f.slasher, f.medic, f.camera, f.jailed = r.slasher, r.medic, r.camSeen > 0, r.jailed > 0
	f.units = r.units
	for _, a := range g.animals {
		if a.dead {
			f.animalsDead++
		} else {
			f.animalsAlive++
		}
	}
	f.island = g.isIsland()
	f.events = g.paperEventSet()
	if g.lvl != nil && g.lvl.Def != nil {
		f.venue = g.lvl.Def.Icon
		f.theme = hintThemeOf(g.lvl.Def)
	}
	if f.island {
		f.theme = "island"
	}
	f.rare = g.paperRareSet()
	f.famous, f.famousDead = g.paperFamousNow()
	f.animalKinds, f.animalDeadKinds = g.paperAnimalsNow()
	for _, gu := range g.guests {
		if gu.perf == "dj" {
			f.dj = true
			break
		}
	}
	f.seed = int64(g.levelIdx+3)*1000003 + int64(f.kills)*131 + int64(f.seen)*17 + int64(f.secs)
	return paperVary(f, fmt.Sprint(g.levelIdx, g.endTime, f.kills, f.seen, f.secs))
}

// ---- variety between parties --------------------------------------------------------------------------------------
// The words of the paper do not come from the game's random numbers (g.rng): the session has its own generator and a
// short memory, so the headline, the story and the small news of the last two parties are not repeated in this one.

type paperSessionT struct {
	rng    *rand.Rand
	id     string          // the party the page on screen belongs to
	salt   int64           // its random salt (the same every frame of the same party)
	avoid  map[string]bool // what the previous parties printed
	recent [2][]string     // the printed keys of the last two parties
	// template news (newspaper_tpl.go): the ids printed by the last three parties (memory only, never saved)
	tplAvoid  map[string]bool
	tplRecent [3][]string
}

var paperSession = &paperSessionT{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}

// paperVary gives the facts of a NEW party a fresh salt and the memory of the last ones; the same party keeps its words.
func paperVary(f paperFacts, id string) paperFacts {
	s := paperSession
	if s.id != id {
		s.id, s.salt = id, s.rng.Int63()
		s.avoid = map[string]bool{}
		for _, r := range s.recent {
			for _, k := range r {
				s.avoid[k] = true
			}
		}
		s.tplAvoid = map[string]bool{}
		for _, r := range s.tplRecent {
			for _, k := range r {
				s.tplAvoid[k] = true
			}
		}
		f.seed ^= s.salt >> 8
		f.avoid, f.tplAvoid = s.avoid, s.tplAvoid
		s.recent[1], s.recent[0] = s.recent[0], paperPrinted(f)
		s.tplRecent[2], s.tplRecent[1], s.tplRecent[0] = s.tplRecent[1], s.tplRecent[0], paperTplIDs(f)
		return f
	}
	f.seed ^= s.salt >> 8
	f.avoid, f.tplAvoid = s.avoid, s.tplAvoid
	return f
}

// paperPrinted: the keys (Spanish) of what the page of these facts prints, to be remembered.
func paperPrinted(f paperFacts) []string {
	out := []string{paperHeadlineForm(f)}
	out = append(out, paperSentences(f)...)
	out = append(out, paperQuote(f))
	sc := paperPhotoScene(f)
	out = append(out, fmt.Sprintf("FOTO:%d", sc), paperPhotoCaption(f, sc))
	for i, b := range paperBriefsFor(f, "0:00") {
		if i >= 14 {
			break
		}
		out = append(out, b.es)
	}
	return out
}

// paperPick: a deterministic choice among the variants of a situation.
func paperPick(seed int64, salt int, list []string) string {
	h := uint64(seed)*6364136223846793005 + uint64(salt+1)*1442695040888963407
	h ^= h >> 29
	return list[int(h%uint64(len(list)))]
}

// dominant: the method that explains at least half of the kills ("" = a mix).
func (f paperFacts) dominant() string {
	tot := f.knife + f.poison + f.traps + f.blast
	if tot == 0 {
		return ""
	}
	for _, m := range []struct {
		n int
		s string
	}{{f.knife, "knife"}, {f.poison, "poison"}, {f.traps, "trap"}, {f.blast, "blast"}} {
		if m.n*2 >= tot {
			return m.s
		}
	}
	return ""
}

// paperHeadlineForm: the template of the big title ("FIESTA %s: ..."), from the main situation of the party plus the
// general pool. paperHeadline fills in the number of the party and the count.
func paperHeadlineForm(f paperFacts) string {
	k := f.kills
	var pool []paperTxt
	switch {
	case k == 0:
		pool = pHeadNone
	case f.animalsDead >= 3:
		pool = pHeadAnimalsMany
	case f.seen == 0 && k >= 5:
		pool = pHeadClean
	case f.blast > 0 && f.dominant() == "blast":
		pool = pHeadBlast
	case f.poison > 0 && f.dominant() == "poison":
		pool = pHeadPoison
	case f.traps > 0 && f.dominant() == "trap":
		pool = pHeadTrap
	case k >= 20:
		pool = pHeadMany
	case f.cuffed > 0:
		pool = pHeadCops
	case k <= 3:
		pool = pHeadFew
	case f.animalsDead >= 1:
		pool = pHeadAnimalsFew
	default:
		pool = pHeadMixed
	}
	all := append([]paperTxt{}, pool...)
	if k > 0 { // the general pool gives a second chance of variety
		all = append(all, pHeadMixed...)
	}
	return f.pick(1, all)
}

// paperHeadline: the big title with the number of the party.
func paperHeadline(f paperFacts) string {
	form := paperHeadlineForm(f)
	n := f.kills
	for _, t := range pHeadAnimalsMany {
		if t.es == form {
			n = f.animalsDead
		}
	}
	if formatCount(form) == 0 {
		return fmt.Sprintf(form, f.level)
	}
	return fmt.Sprintf(form, f.level, n)
}

// paperSentences: the sentences of the story in priority order (the first one always exists, the last one closes).
func paperSentences(f paperFacts) []string {
	var open []paperTxt
	switch {
	case f.kills == 0:
		open = pOpenNone
	case f.dominant() == "knife":
		open = pOpenKnife
	case f.dominant() == "poison":
		open = pOpenPoison
	case f.dominant() == "trap":
		open = pOpenTrap
	case f.dominant() == "blast":
		open = pOpenBlast
	default:
		open = pOpenMixed
	}
	open = append([]paperTxt{}, open...)
	switch {
	case f.kills >= 1 && f.kills <= 3:
		open = append(open, pOpenFew...)
	case f.kills >= 20:
		open = append(open, pOpenManyK...)
	}
	s := []string{f.pick(2, open)}

	// the most newsworthy first (animals), then the rest in an order shuffled by the seed
	switch {
	case f.animalsDead >= 4:
		s = append(s, f.pick(50, pAnimalsMany))
	case f.animalsDead >= 1:
		s = append(s, f.pick(51, pAnimalsDead))
	case f.animalsAlive >= 2:
		s = append(s, f.pick(52, pAnimalsAlive))
	}
	if f.island {
		s = append(s, f.pick(53, pIsland))
	}
	if pool := pVenue[f.venue]; len(pool) > 0 { // the cake, the rink, the mirror ball (venues_text.go)
		s = append(s, f.pick(54, pool))
	}
	var opt []string
	add := func(salt int, pool []paperTxt) { opt = append(opt, f.pick(salt, pool)) }
	switch {
	case f.seen == 0:
		add(10, pSeen0)
	case f.seen >= 4:
		add(11, pSeenMany)
	default:
		add(12, pSeenFew)
	}
	if f.cuffed > 0 {
		add(13, pCuffed)
	}
	if f.jailed {
		add(14, pJailed)
	}
	if f.camera {
		add(15, pCamera)
	}
	if f.slasher {
		add(16, pSlasher)
	}
	switch {
	case f.units&(1<<unitMilitary) != 0:
		add(17, pMilitary)
	case f.units&(1<<unitSWAT) != 0:
		add(17, pSWAT)
	case f.units&(1<<unitRangers) != 0:
		add(17, pRangers)
	case f.units&(1<<unitFBI) != 0:
		add(17, pFBI)
	case f.units&(1<<unitPolice) != 0:
		add(17, pPolice)
	default:
		add(17, pNoPolice)
	}
	if f.medic {
		add(18, pMedic)
	}
	if f.chainMax >= 4 {
		add(19, pChain)
	}
	if f.two {
		add(20, pTwo)
	}
	if f.challenge {
		add(21, pChallenge)
	}
	if f.dj {
		add(22, pDJ)
	}
	switch {
	case f.secs > 0 && f.secs <= 100:
		add(23, pFast)
	case f.secs >= 420:
		add(24, pSlow)
	}
	switch {
	case f.level == "EXTRA":
		add(25, pExtra)
	case f.levelN >= 15:
		add(26, pLevelLate)
	case f.levelN >= 1 && f.levelN <= 5:
		add(27, pLevelEarly)
	}
	type keyed struct {
		k uint64
		s string
	}
	ks := make([]keyed, len(opt))
	for i, o := range opt {
		h := uint64(f.seed)*6364136223846793005 + uint64(i+100)*1442695040888963407
		h ^= h >> 31
		ks[i] = keyed{h * 0x9e3779b97f4a7c15, o}
	}
	sort.SliceStable(ks, func(i, j int) bool { return ks[i].k < ks[j].k })
	for _, k := range ks {
		s = append(s, k.s)
	}
	return append(s, f.pick(30, pClose))
}

// paperQuote: the witness at the bottom of the front page.
func paperQuote(f paperFacts) string { return f.pick(40, pQuotes) }
