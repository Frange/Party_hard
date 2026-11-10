package main

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// TEMPLATE NEWS of the newspaper (the small news with gaps). Each template is a joke in Spanish and English with
// slots that are filled with what the party had: an invented first name ({P}, {Q}), a famous guest who was there
// ({F}), an animal ({A}, preferably one of the party), a place of the level ({L}), an object ({O}) and small
// numbers ({N}, {M}). So one template gives many different news. Every template belongs to a context (police,
// ambulance, UFO, zombies, a calm party, food, a level...) and is only printed when the party had it.
//
// Slot forms (the same noun gives every form, with the right article, gender and number):
//
//	{A} definite "la cobaya" / "the guinea pig"    {A.un} "una cobaya" / "a guinea pig"   {A.n} "cobaya" / "guinea pig"
//	{A.pl} "cobayas" / "guinea pigs"               {A.al} "a la cobaya" / "al canguro"     {A.del} "de la cobaya" / "del canguro"
//	{A.o} "a" / "o" (Spanish adjective ending)      {P.o} the same for a person             {P.he} {P.his} {P.him} (English)
//	{^A} the same with a capital letter (start of a sentence)
//
// Selection: deterministic from the facts of the party (its seed): the same party prints the same news on every
// frame; templates printed by the last parties of the session go to the back of the line (paperSession, in memory,
// never in the save). Nothing here touches g.rng.
// ---------------------------------------------------------------------------

// paperTpl: one template news.
type paperTpl struct {
	id, ctx, head string
	es, en        string
}

// ptNoun: a noun with its gender and both plurals.
type ptNoun struct {
	es    string
	fem   bool
	en    string
	esPl  string
	enPl  string
	key   string // animals: the kind (animalKinds) it stands for
	theme string // places: the level theme ("" = any level)
}

// ptPerson: a first name (or a famous guest's name) and its gender.
type ptPerson struct {
	name string
	fem  bool
	en   string // the name in English when it differs (famous names with a translation)
}

// ptVals: the fillers of one news.
type ptVals struct {
	P, Q, F ptPerson
	A, L, O ptNoun
	N, M    int
}

// paperTplGeneral: the contexts every party has (the others only exist when the party had them, and go first).
func paperTplGeneral(ctx string) bool {
	switch ctx {
	case "food", "neigh", "sports", "weather", "odd", "close", "animals":
		return true
	}
	return false
}

// paperTplCtxOK: does the party have this context?
func paperTplCtxOK(f paperFacts, ctx string) bool {
	switch {
	case paperTplGeneral(ctx):
		return true
	case ctx == "final":
		return false // only the special edition of the ending (finalPaperData)
	case ctx == "police":
		return f.cuffed > 0 || f.units != 0 || f.jailed
	case ctx == "arrest":
		return f.cuffed > 0 || f.jailed
	case ctx == "medic":
		return f.medic
	case ctx == "calm":
		return f.kills == 0
	case ctx == "bloody":
		return f.kills >= 10
	case ctx == "two":
		return f.two
	case ctx == "animaldead":
		return f.animalsDead > 0
	case ctx == "famous":
		return len(f.famous) > 0
	case ctx == "famousdead":
		return len(f.famousDead) > 0
	case strings.HasPrefix(ctx, "rare:"):
		return f.rare[ctx[5:]]
	case strings.HasPrefix(ctx, "level:"):
		return f.theme == ctx[6:]
	}
	return false
}

func paperTplHash(seed int64, id string, salt uint64) uint64 {
	h := uint64(seed)*6364136223846793005 + 1442695040888963407 + salt*0x9e3779b97f4a7c15
	for i := 0; i < len(id); i++ {
		h = (h ^ uint64(id[i])) * 0x100000001b3
	}
	h ^= h >> 31
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 29
	return h
}

// paperTplPick: one chosen template news, already filled in both languages.
type paperTplPick struct {
	id, head, es, en string
}

// paperTplOrder: the templates of the party's contexts in the order of its seed; what the last parties printed goes last.
func paperTplOrder(f paperFacts, list []paperTpl) []paperTpl {
	type kv struct {
		k uint64
		t paperTpl
	}
	var ks []kv
	for _, t := range list {
		h := paperTplHash(f.seed, t.id, 1)
		if f.tplAvoid[t.id] {
			h |= 1 << 63
		} else {
			h &^= 1 << 63
		}
		ks = append(ks, kv{h, t})
	}
	sort.SliceStable(ks, func(i, j int) bool { return ks[i].k < ks[j].k })
	out := make([]paperTpl, len(ks))
	for i := range ks {
		out[i] = ks[i].t
	}
	return out
}

// paperTplPicks: the template news of this party: up to two about what happened in it (police, UFO, the level, a
// famous guest...) and two general ones, all with different section heads and never the same template twice.
func paperTplPicks(f paperFacts) []paperTplPick {
	var spec, gen []paperTpl
	for _, t := range paperTpls {
		if !paperTplCtxOK(f, t.ctx) {
			continue
		}
		if paperTplGeneral(t.ctx) {
			gen = append(gen, t)
		} else {
			spec = append(spec, t)
		}
	}
	heads := map[string]bool{}
	var out []paperTplPick
	take := func(list []paperTpl, n int) {
		got := 0
		for _, t := range paperTplOrder(f, list) {
			if got >= n {
				return
			}
			if heads[t.head] {
				continue
			}
			heads[t.head] = true
			es, en := paperTplFill(t, paperTplValues(f, t))
			out = append(out, paperTplPick{t.id, t.head, es, en})
			got++
		}
	}
	take(spec, 2)
	take(gen, 2)
	return out
}

// paperTplByID: the template of an id (nil if there is none).
func paperTplByID(id string) *paperTpl {
	for i := range paperTpls {
		if paperTpls[i].id == id {
			return &paperTpls[i]
		}
	}
	return nil
}

// paperTplIDs: the ids of the templates this party prints (the session remembers them).
func paperTplIDs(f paperFacts) []string {
	var out []string
	for _, p := range paperTplPicks(f) {
		out = append(out, p.id)
	}
	return out
}

// paperTplBriefs: the picks as small news (es and en already filled; newPaperData shows the one of the language).
func paperTplBriefs(f paperFacts) []paperBrief {
	var out []paperBrief
	for _, p := range paperTplPicks(f) {
		out = append(out, paperBrief{p.head, p.es, p.en})
	}
	return out
}

// paperIsTplBrief: a brief whose text is already written in both languages (a filled template), not a translation key.
func paperIsTplBrief(b paperBrief) bool {
	if b.en == "" {
		return false
	}
	en, ok := langDataEN[b.es]
	return !ok || en != b.en
}

// paperBriefText: the text of a small news in the language in force.
func paperBriefText(b paperBrief) string {
	if paperIsTplBrief(b) {
		if uiLang == "en" {
			return b.en
		}
		return b.es
	}
	return tr(b.es)
}

// paperTplValues: the fillers of template t in this party (deterministic from the seed and the template).
func paperTplValues(f paperFacts, t paperTpl) ptVals {
	h := func(salt uint64) int { return int(paperTplHash(f.seed, t.id, salt) >> 1) }
	var v ptVals
	v.P = ptPeople[h(2)%len(ptPeople)]
	v.Q = ptPeople[h(3)%len(ptPeople)]
	if v.Q.name == v.P.name {
		v.Q = ptPeople[(h(3)+1)%len(ptPeople)]
	}
	// the animal: one of the dead ones for the news about them, else one that was at the party, else any
	animals := ptAnimalsOf(f.animalKinds)
	if t.ctx == "animaldead" && len(f.animalDeadKinds) > 0 {
		animals = ptAnimalsOf(f.animalDeadKinds)
	}
	if len(animals) == 0 {
		animals = ptAnimals
	}
	v.A = animals[h(4)%len(animals)]
	places := ptPlacesFor(f.theme)
	v.L = places[h(5)%len(places)]
	v.O = ptObjects[h(6)%len(ptObjects)]
	fam := f.famous
	if t.ctx == "famousdead" {
		fam = f.famousDead
	}
	if len(fam) > 0 {
		v.F = fam[h(7)%len(fam)]
	} else {
		v.F = ptPerson{name: "un famoso", en: "a celebrity"}
	}
	v.N = 2 + h(8)%8
	v.M = 12 + h(9)%80
	return v
}

// ptAnimalsOf: the animals of the table that stand for these kinds (in the order of the table, no repeats).
func ptAnimalsOf(kinds []string) []ptNoun {
	set := map[string]bool{}
	for _, k := range kinds {
		set[k] = true
	}
	var out []ptNoun
	for _, a := range ptAnimals {
		if set[a.key] {
			out = append(out, a)
		}
	}
	return out
}

// ptPlacesFor: the places of a level theme plus the ones of any party.
func ptPlacesFor(theme string) []ptNoun {
	var out []ptNoun
	for _, p := range ptPlaces {
		if p.theme == "" || p.theme == theme {
			out = append(out, p)
		}
	}
	return out
}

// paperTplFill fills a template in both languages.
func paperTplFill(t paperTpl, v ptVals) (es, en string) {
	return ptExpand(t.es, "es", &v), ptExpand(t.en, "en", &v)
}

// ptExpand replaces every {slot.form} of s.
func ptExpand(s, lang string, v *ptVals) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '{' {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := strings.IndexByte(s[i:], '}')
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		tok := s[i+1 : i+j]
		caps := strings.HasPrefix(tok, "^")
		out := v.slot(strings.TrimPrefix(tok, "^"), lang)
		if caps {
			out = ptUpperFirst(out)
		}
		b.WriteString(out)
		i += j + 1
	}
	return b.String()
}

func ptUpperFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// slot: the text of one {slot.form}; an unknown slot stays visible ("{?x}": the tests catch it).
func (v *ptVals) slot(tok, lang string) string {
	name, form := tok, ""
	if i := strings.IndexByte(tok, '.'); i >= 0 {
		name, form = tok[:i], tok[i+1:]
	}
	switch name {
	case "P":
		return v.P.form(form, lang)
	case "Q":
		return v.Q.form(form, lang)
	case "F":
		return v.F.form(form, lang)
	case "A":
		return v.A.form(form, lang)
	case "L":
		return v.L.form(form, lang)
	case "O":
		return v.O.form(form, lang)
	case "N":
		return strconv.Itoa(v.N)
	case "M":
		return strconv.Itoa(v.M)
	}
	return "{?" + tok + "}"
}

func (p ptPerson) form(form, lang string) string {
	switch form {
	case "":
		if lang == "en" && p.en != "" {
			return p.en
		}
		return p.name
	case "o":
		return pick2(p.fem, "a", "o")
	case "el":
		return pick2(p.fem, "la", "el")
	case "lo":
		return pick2(p.fem, "la", "lo")
	case "un":
		return pick2(p.fem, "una", "un")
	case "he":
		return pick2(p.fem, "she", "he")
	case "his":
		return pick2(p.fem, "her", "his")
	case "him":
		return pick2(p.fem, "her", "him")
	}
	return "{?" + form + "}"
}

func pick2(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}

func (n ptNoun) form(form, lang string) string {
	if lang == "en" {
		switch form {
		case "":
			return "the " + n.en
		case "un":
			return enArticle(n.en) + " " + n.en
		case "n":
			return n.en
		case "pl":
			return n.enPl
		}
		return "{?" + form + "}"
	}
	switch form {
	case "":
		return pick2(n.fem, "la ", "el ") + n.es
	case "un":
		return pick2(n.fem, "una ", "un ") + n.es
	case "n":
		return n.es
	case "pl":
		return n.esPl
	case "los":
		return pick2(n.fem, "las ", "los ") + n.esPl
	case "al":
		return pick2(n.fem, "a la ", "al ") + n.es
	case "del":
		return pick2(n.fem, "de la ", "del ") + n.es
	case "o":
		return pick2(n.fem, "a", "o")
	case "este":
		return pick2(n.fem, "esta ", "este ") + n.es
	}
	return "{?" + form + "}"
}

// enArticle: "a" or "an".
func enArticle(w string) string {
	if w != "" && strings.ContainsRune("aeiouAEIOU", rune(w[0])) {
		return "an"
	}
	return "a"
}

// ---- what the party had, for the templates (read from the game by paperFactsNow) ------------------------------------

// paperRareKinds: the big set pieces of events.go the templates know, by the name used in the contexts ("rare:ufo").
var paperRareKinds = map[string]int{"plane": evPlane, "ufo": evUFO, "super": evSuper, "satan": evSatan, "god": evGod,
	"zombie": evZombie, "santa": evSanta, "cow": evCow, "meteor": evMeteor, "ghost": evGhost, "rex": evRex}

// paperRareSet: the big set pieces that started in this party (the counters of the statistics minus the ones they had
// when the party started: partylife.go keeps that snapshot; nothing is written).
func (g *Game) paperRareSet() map[string]bool {
	m := map[string]bool{}
	for name, k := range paperRareKinds {
		key := "ev_" + strconv.Itoa(k)
		if g.progress.Stats[key] > g.life.evBase[key] {
			m[name] = true
		}
	}
	return m
}

// paperBannedNames: real people who are never named in the paper (the owner's policy: only musicians and
// footballers keep their real name; the rest of the real people have no visible name).
var paperBannedNames = []string{"Trump", "Obama", "Putin", "Biden", "Shakira", "Madonna", "Mulder", "Scully"}

// paperFamousName: the name of a famous guest as the paper prints it: the name the game shows, or "" when it does
// not suit a sentence (a band fallback "Banda · rol", too long, or a forbidden real name).
func paperFamousName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "·") || utf8.RuneCountInString(name) > 20 {
		return ""
	}
	for _, w := range strings.Fields(name) {
		if paperTextW(w) > 150 {
			return ""
		}
	}
	for _, bad := range paperBannedNames {
		if strings.Contains(name, bad) {
			return ""
		}
	}
	return name
}

// paperFamousNow: the famous guests of the party (alive / dead), sorted by name, from the names the game shows.
func (g *Game) paperFamousNow() (alive, dead []ptPerson) {
	seen := map[string]bool{}
	for _, gu := range g.guests {
		if gu == nil || gu.cs == nil || gu.cs.Spec == nil || !g.isFamous(gu.cs.Spec.Name) {
			continue
		}
		n := paperFamousName(gu.name)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		p := ptPerson{name: n, fem: gu.cs.Spec.Female}
		if en, ok := langDataEN[n]; ok && en != n && paperFamousName(en) != "" {
			p.en = en
		}
		if gu.dead {
			dead = append(dead, p)
		} else {
			alive = append(alive, p)
		}
	}
	by := func(l []ptPerson) { sort.Slice(l, func(i, j int) bool { return l[i].name < l[j].name }) }
	by(alive)
	by(dead)
	return alive, dead
}

// paperAnimalsNow: the kinds of the animals of the party (alive / dead) and the mascot, sorted.
func (g *Game) paperAnimalsNow() (all, dead []string) {
	a, d := map[string]bool{}, map[string]bool{}
	for _, an := range g.animals {
		if an == nil || an.kind < 0 || an.kind >= len(animalKinds) {
			continue
		}
		k := animalKinds[an.kind]
		a[k] = true
		if an.dead {
			d[k] = true
		}
	}
	if m := g.life.mascot; m != nil {
		a[m.kind] = true
		if m.dead {
			d[m.kind] = true
		}
	}
	keys := func(m map[string]bool) []string {
		var out []string
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	return keys(a), keys(d)
}

// ---- the special edition of the ending ----------------------------------------------------------------------------

// paperTplFinalBriefs: two template news for the last newspaper (the seed of the ending fills them), in the language
// in force.
func paperTplFinalBriefs(seed int64) []paperBrief {
	f := paperFacts{seed: seed, theme: "island"}
	var list []paperTpl
	for _, t := range paperTpls {
		if t.ctx == "final" {
			list = append(list, t)
		}
	}
	var out []paperBrief
	heads := map[string]bool{}
	for _, t := range paperTplOrder(f, list) {
		if len(out) >= 2 {
			break
		}
		if heads[t.head] {
			continue
		}
		heads[t.head] = true
		es, en := paperTplFill(t, paperTplValues(f, t))
		txt := es
		if uiLang == "en" {
			txt = en
		}
		out = append(out, paperBrief{head: tr(t.head), es: txt})
	}
	return out
}
