package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

// ---- helpers --------------------------------------------------------------------------------------------------------

// tplSlots: the slot names a text uses ("P", "A"...), sorted.
func tplSlots(s string) []string {
	set := map[string]bool{}
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		j := strings.IndexByte(s[i:], '}')
		if j < 0 {
			return []string{"UNCLOSED"}
		}
		tok := strings.TrimPrefix(s[i+1:i+j], "^")
		if k := strings.IndexByte(tok, '.'); k >= 0 {
			tok = tok[:k]
		}
		set[tok] = true
	}
	var out []string
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// tplFacts: facts that open the context of a template.
func tplFacts(ctx string, seed int64) paperFacts {
	f := paperFacts{level: "7", levelN: 7, kills: 4, knife: 4, seen: 1, secs: 200, seed: seed}
	switch {
	case ctx == "police":
		f.units = 1 << unitPolice
	case ctx == "arrest":
		f.cuffed = 1
	case ctx == "medic":
		f.medic = true
	case ctx == "calm":
		f.kills, f.knife = 0, 0
	case ctx == "bloody":
		f.kills, f.knife = 14, 14
	case ctx == "two":
		f.two = true
	case ctx == "animaldead":
		f.animalsDead, f.animalKinds, f.animalDeadKinds = 1, []string{"cobaya", "kangaroo"}, []string{"cobaya"}
	case ctx == "famous":
		f.famous = []ptPerson{{name: "Doctor Bacterio"}}
	case ctx == "famousdead":
		f.famousDead = []ptPerson{{name: "Doctor Bacterio"}}
	case strings.HasPrefix(ctx, "rare:"):
		f.rare = map[string]bool{ctx[5:]: true}
	case strings.HasPrefix(ctx, "level:"):
		f.theme = ctx[6:]
	}
	return f
}

// tplWorst: for every slot, the fillers with the longest texts (the worst cases for the layout).
func tplWorst(famous []ptPerson) (people, fam []ptPerson, animals, places, objects []ptNoun) {
	longP := func(l []ptPerson) []ptPerson {
		l = append([]ptPerson{}, l...)
		sort.SliceStable(l, func(i, j int) bool { return paperTextW(l[i].name) > paperTextW(l[j].name) })
		if len(l) > 2 {
			l = l[:2]
		}
		return l
	}
	longN := func(l []ptNoun) []ptNoun {
		l = append([]ptNoun{}, l...)
		w := func(n ptNoun) int {
			return maxI(paperTextW(n.es+n.esPl+"de la "), paperTextW(n.en+n.enPl+"the "))
		}
		sort.SliceStable(l, func(i, j int) bool { return w(l[i]) > w(l[j]) })
		// and the noun with the widest single word (it cannot be broken across lines)
		widest := l[0]
		for _, n := range l {
			for _, s := range []string{n.es, n.en, n.esPl, n.enPl} {
				for _, word := range strings.Fields(s) {
					for _, cur := range strings.Fields(widest.es + " " + widest.en) {
						if paperTextW(word) > paperTextW(cur) {
							widest = n
						}
					}
				}
			}
		}
		out := append([]ptNoun{}, l[:2]...)
		return append(out, widest)
	}
	return longP(ptPeople), longP(famous), longN(ptAnimals), longN(ptPlaces), longN(ptObjects)
}

// tplAllFamous: every famous guest the paper could name (the presets of the game, through paperFamousName).
func tplAllFamous() []ptPerson {
	var out []ptPerson
	probe := rand.New(rand.NewSource(1))
	seen := map[string]bool{}
	for _, f := range famousPresets {
		sp := f(probe)
		if !famousNames()[sp.Name] {
			continue
		}
		n := paperFamousName(nameFromSpec(&sp))
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		p := ptPerson{name: n, fem: sp.Female}
		if en, ok := langDataEN[n]; ok && paperFamousName(en) != "" {
			p.en = en
		}
		out = append(out, p)
	}
	return out
}

// ---- the bank of templates ------------------------------------------------------------------------------------------

// TestPaperTplBankNewDistinctBothLanguages: at least 100 new template news (not counting the special edition), all
// different from each other and from the news that already existed, with Spanish and English, the same slots in both,
// a known context and a translated section head.
func TestPaperTplBankNewDistinctBothLanguages(t *testing.T) {
	loadLanguages()
	old := map[string]bool{}
	for _, b := range paperBriefBank {
		old[b.es], old[b.en] = true, true
	}
	for _, b := range fpBriefs {
		old[b.es], old[b.en] = true, true
	}
	for _, p := range paperEventPools {
		for _, x := range p.pool {
			old[x.es], old[x.en] = true, true
		}
	}
	for _, pool := range append(paperAllPools(), paperBriefDerivedPools...) {
		for _, x := range pool {
			old[x.es], old[x.en] = true, true
		}
	}
	ids, es, en := map[string]bool{}, map[string]bool{}, map[string]bool{}
	ctxs := map[string]int{}
	nonFinal := 0
	for _, tp := range paperTpls {
		if tp.id == "" || ids[tp.id] {
			t.Errorf("template id %q empty or repeated", tp.id)
		}
		ids[tp.id] = true
		// the joke without its fillers must be new (no repeated template, no copy of an old news)
		if es[tp.es] || en[tp.en] || old[tp.es] || old[tp.en] {
			t.Errorf("%s: repeated text", tp.id)
		}
		es[tp.es], en[tp.en] = true, true
		if tp.es == "" || tp.en == "" || tp.es == tp.en {
			t.Errorf("%s: Spanish/English missing", tp.id)
		}
		if a, b := tplSlots(tp.es), tplSlots(tp.en); strings.Join(a, ",") != strings.Join(b, ",") {
			t.Errorf("%s: slots ES %v, EN %v", tp.id, a, b)
		}
		if _, ok := paperBriefHeads[tp.head]; !ok || tr(tp.head) == "" {
			t.Errorf("%s: head %q has no English", tp.id, tp.head)
		}
		inLang(t, "en", func() {
			if tr(tp.head) == tp.head && paperBriefHeads[tp.head] != tp.head {
				t.Errorf("%s: head %q is not translated", tp.id, tp.head)
			}
		})
		known := paperTplGeneral(tp.ctx) || tp.ctx == "final"
		if !known {
			known = paperTplCtxOK(tplFacts(tp.ctx, 1), tp.ctx) && !paperTplCtxOK(paperFacts{kills: 4}, tp.ctx)
		}
		if !known {
			t.Errorf("%s: context %q is not known (or is always on)", tp.id, tp.ctx)
		}
		if strings.HasPrefix(tp.ctx, "rare:") {
			if _, ok := paperRareKinds[tp.ctx[5:]]; !ok {
				t.Errorf("%s: no big set piece %q", tp.id, tp.ctx)
			}
		}
		if strings.HasPrefix(tp.ctx, "level:") {
			th := tp.ctx[6:]
			if _, ok := hintThemes[th]; !ok && th != "island" {
				t.Errorf("%s: no level theme %q", tp.id, th)
			}
		}
		ctxs[tp.ctx]++
		if tp.ctx != "final" {
			nonFinal++
		}
	}
	if nonFinal < 100 {
		t.Errorf("%d new template news for the level paper, want 100+", nonFinal)
	}
	if ctxs["final"] < 3 {
		t.Errorf("%d template news for the last paper, want 3+", ctxs["final"])
	}
	for _, c := range []string{"police", "arrest", "medic", "calm", "food", "animals", "neigh", "famous", "close", "sports", "weather", "odd",
		"rare:ufo", "rare:zombie", "rare:super", "rare:satan", "rare:god", "rare:rex", "rare:meteor", "rare:ghost", "rare:plane", "rare:cow",
		"level:beach", "level:station", "level:island", "level:italy", "level:eiffel"} {
		if ctxs[c] == 0 {
			t.Errorf("no template news for %q", c)
		}
	}
	t.Logf("%d template news (%d for the level paper) in %d contexts", len(paperTpls), nonFinal, len(ctxs))
}

// TestPaperTplWorstCasesFit: every template, filled with the longest names, animals, places and objects (and the
// longest famous names of the game), in Spanish and English: no slot left, short enough, no word wider than the
// narrowest column, few lines; and the worst one of each template, printed on a real page, stays inside its box with
// its padding, sharp (scale 1, whole pixels) and clear of the other blocks.
func TestPaperTplWorstCasesFit(t *testing.T) {
	pw := paperNoWear
	t.Cleanup(func() { paperNoWear = pw })
	paperNoWear = true
	loadLanguages()
	people, fam, animals, places, objects := tplWorst(tplAllFamous())
	if len(fam) == 0 {
		t.Fatal("no famous name to test")
	}
	const colW = pcLW - 10
	for _, tp := range paperTpls {
		worst, longest, wide := map[string]string{}, map[string]string{}, map[string]bool{}
		for _, P := range people {
			for _, F := range fam {
				for _, A := range animals {
					for _, L := range places {
						for _, O := range objects {
							v := ptVals{P: P, Q: people[len(people)-1], F: F, A: A, L: L, O: O, N: 9, M: 88}
							es, en := paperTplFill(tp, v)
							for lang, s := range map[string]string{"es": es, "en": en} {
								if strings.ContainsAny(s, "{}") {
									t.Fatalf("%s %s: slot left: %q", tp.id, lang, s)
								}
								if utf8.RuneCountInString(s) > utf8.RuneCountInString(longest[lang]) {
									longest[lang] = s
								}
								if paperTextW(s) > paperTextW(worst[lang]) {
									worst[lang] = s
								}
								for _, w := range splitWords(s) {
									if paperTextW(w) > colW {
										wide[lang+" "+w] = true
									}
								}
							}
						}
					}
				}
			}
		}
		for w := range wide {
			t.Errorf("%s: the word %q is wider than the column", tp.id, w)
		}
		for _, lang := range []string{"es", "en"} {
			if n := utf8.RuneCountInString(longest[lang]); n > 100 {
				t.Errorf("%s %s: %d characters: %q", tp.id, lang, n, longest[lang])
			}
			for _, s := range []string{worst[lang], longest[lang]} {
				if n := len(wrapPx(s, colW)); n > 6 {
					t.Errorf("%s %s: %d lines in a small column: %q", tp.id, lang, n, s)
				}
			}
			tplCheckOnPage(t, tp.id+" "+lang, tp.head, worst[lang])
		}
	}
}

// tplCheckOnPage prints one small news first on a page and checks its box.
func tplCheckOnPage(t *testing.T, name, head, text string) {
	t.Helper()
	d := paperSample(paperFacts{seed: 5, kills: 3, knife: 3}, false, false, 9)
	d.briefs = append([]paperBrief{{head: head, es: text}}, d.briefs...)
	_, lay := renderPaper(d, 0)
	var box *prect
	for i, b := range lay.blocks {
		if b.kind == "brief" && b.x == pcLX {
			box = &lay.blocks[i]
			break
		}
	}
	if box == nil || len(lay.briefs) == 0 || lay.briefs[0] != text {
		t.Fatalf("%s: the news was not printed first: %q", name, text)
	}
	var mine []ptext
	for _, tx := range lay.texts {
		if prInside(prect{"", tx.x, tx.y, tx.w, tx.h}, *box) {
			mine = append(mine, tx)
		}
	}
	if len(mine) < 2 {
		t.Fatalf("%s: %d texts in the box", name, len(mine))
	}
	for _, tx := range mine {
		if tx.scale != 1 {
			t.Errorf("%s: %q at scale %d (blurry)", name, tx.s, tx.scale)
		}
		if tx.x-box.x < 5 || box.x+box.w-(tx.x+tx.w) < 5 || tx.y-box.y < 6 || box.y+box.h-(tx.y+tx.h) < 6 {
			t.Errorf("%s: %q too close to the edge of its box %+v (%d,%d %dx%d)", name, tx.s, *box, tx.x, tx.y, tx.w, tx.h)
		}
		// the column rule on the right is 5 px further: >= 6 px of paper between the text and any line
		if pcLX+pcLW+5-(tx.x+tx.w) < 6 {
			t.Errorf("%s: %q touches the column rule", name, tx.s)
		}
	}
	// >= 12 px from the texts of the next block
	bottom := 0
	for _, tx := range mine {
		bottom = maxI(bottom, tx.y+tx.h)
	}
	for _, tx := range lay.texts {
		if tx.x >= pcLX && tx.x < pcLX+pcLW && tx.y >= box.y+box.h && tx.y-bottom < 12 {
			t.Errorf("%s: the next block's %q is %d px under the news", name, tx.s, tx.y-bottom)
		}
	}
	// no block overlaps another, nothing leaves the body
	for i, a := range lay.blocks {
		if a.kind == "masthead" || a.kind == "headline" || a.kind == "dek" || a.kind == "footer" {
			continue
		}
		if a.y < pbY0 || a.y+a.h > pbY1 {
			t.Errorf("%s: block %+v leaves the body", name, a)
		}
		for _, b := range lay.blocks[i+1:] {
			if b.kind != "rowA" && a.kind != "rowA" && prHit(a, b) {
				t.Errorf("%s: blocks overlap %+v %+v", name, a, b)
			}
		}
	}
}

// ---- choice ---------------------------------------------------------------------------------------------------------

// TestPaperTplEveryContextGetsPrinted: every template can reach the page when its context happens (over some seeds),
// the picks are deterministic, never the same template or section head twice, and the ones about what happened go
// first.
func TestPaperTplEveryContextGetsPrinted(t *testing.T) {
	loadLanguages()
	reached := map[string]bool{}
	for _, tp := range paperTpls {
		if tp.ctx == "final" {
			continue
		}
		for seed := int64(0); seed < 400 && !reached[tp.id]; seed++ {
			f := tplFacts(tp.ctx, seed*7919+11)
			picks := paperTplPicks(f)
			again := paperTplPicks(f)
			if fmt.Sprint(picks) != fmt.Sprint(again) {
				t.Fatalf("%s: the same facts pick different news", tp.ctx)
			}
			ids, heads := map[string]bool{}, map[string]bool{}
			for _, p := range picks {
				if ids[p.id] || heads[p.head] {
					t.Fatalf("seed %d: %q or head %q twice in %v", seed, p.id, p.head, picks)
				}
				ids[p.id], heads[p.head] = true, true
				reached[p.id] = true
			}
			if !paperTplGeneral(tp.ctx) && (len(picks) == 0 || paperTplGeneral(paperTplByID(picks[0].id).ctx)) {
				t.Fatalf("%s seed %d: the news about what happened does not go first: %v", tp.ctx, seed, picks)
			}
		}
		if !reached[tp.id] {
			t.Errorf("template %q (%s) never printed", tp.id, tp.ctx)
		}
	}
	// a party without anything special prints only general ones; a UFO party prints a UFO news
	for _, p := range paperTplPicks(paperFacts{seed: 3, kills: 4}) {
		if !paperTplGeneral(paperTplByID(p.id).ctx) {
			t.Errorf("plain party printed %q", p.id)
		}
	}
	f := tplFacts("rare:ufo", 99)
	if p := paperTplPicks(f); len(p) == 0 || paperTplByID(p[0].id).ctx != "rare:ufo" {
		t.Errorf("UFO party: %v", p)
	}
	// the facts of the party fill the gaps: the famous guest and the animals of the party
	f = tplFacts("famous", 5)
	f.animalKinds = []string{"kangaroo"}
	for seed := int64(0); seed < 60; seed++ {
		f.seed = seed
		for _, p := range paperTplPicks(f) {
			tp := paperTplByID(p.id)
			if strings.Contains(tp.es, "{F") && !strings.Contains(p.es, "Doctor Bacterio") {
				t.Errorf("%s: the famous guest is not in %q", p.id, p.es)
			}
			if strings.Contains(tp.es, "{A") && !strings.Contains(tp.es, "{A.") && tp.ctx != "final" && !strings.Contains(p.es, "canguro") {
				t.Errorf("%s: the party's animal is not in %q", p.id, p.es)
			}
		}
	}
}

// TestPaperTplSessionNoRepeat: parties in a row with the same result never print a template news of the last party;
// the same party (every frame) prints the same ones.
func TestPaperTplSessionNoRepeat(t *testing.T) {
	old := paperSession
	t.Cleanup(func() { paperSession = old })
	paperSession = &paperSessionT{rng: newTestRand(9)}
	base := tplFacts("police", 0)
	base.rare = map[string]bool{"ufo": true}
	var prev []string
	all := map[string]bool{}
	for i := 0; i < 30; i++ {
		id := fmt.Sprint("party-", i)
		f := paperVary(base, id)
		ids := paperTplIDs(f)
		if len(ids) < 3 {
			t.Fatalf("party %d: only %d template news", i, len(ids))
		}
		for _, x := range ids {
			for _, p := range prev {
				if x == p {
					t.Errorf("party %d repeats %q of the party before", i, x)
				}
			}
			all[x] = true
		}
		if again := paperTplIDs(paperVary(base, id)); strings.Join(again, ",") != strings.Join(ids, ",") {
			t.Fatalf("party %d: the page changed between two frames: %v / %v", i, ids, again)
		}
		prev = ids
	}
	if len(all) < 30 {
		t.Errorf("30 parties printed only %d different template news", len(all))
	}
}

// ---- language and names ---------------------------------------------------------------------------------------------

// TestPaperTplPrintedInTheLanguage: the filled news reach the page in the language in force (they are not translation
// keys), and the bank news are still translated.
func TestPaperTplPrintedInTheLanguage(t *testing.T) {
	f := tplFacts("rare:ghost", 21)
	picks := paperTplPicks(f)
	if len(picks) < 3 {
		t.Fatalf("picks: %v", picks)
	}
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			d := paperSample(f, false, false, 4)
			got := map[string]bool{}
			for _, b := range d.briefs {
				got[b.es] = true
			}
			for _, p := range picks {
				want := map[string]string{"es": p.es, "en": p.en}[lang]
				if !got[want] {
					t.Errorf("%s: %q missing from the page", lang, want)
				}
				if !got[want] || want == "" {
					continue
				}
				if lang == "en" && strings.ContainsAny(want, "¡¿«»") {
					t.Errorf("en: Spanish marks in %q", want)
				}
			}
			for _, b := range paperBriefBank[:5] {
				if paperIsTplBrief(b) {
					t.Errorf("bank news taken for a template: %q", b.es)
				}
			}
		})
	}
}

// TestPaperTplNoForbiddenRealNames: the templates name nobody real, and a famous guest is only printed with the name
// the game shows (musicians and footballers with their real name; the rest with their parody name).
func TestPaperTplNoForbiddenRealNames(t *testing.T) {
	var real []string
	for _, m := range bandRealNames {
		real = append(real, m.name)
	}
	for _, f := range footballRealNames {
		real = append(real, f.name)
	}
	for _, tp := range paperTpls {
		for _, bad := range append(append([]string{}, paperBannedNames...), real...) {
			if strings.Contains(tp.es+tp.en, bad) {
				t.Errorf("%s: real name %q written in the template", tp.id, bad)
			}
		}
	}
	for _, s := range []string{"Donald Trump", "Barack Obama", "Metalica · cantante", "Un nombre larguísimo que no cabe nunca"} {
		if paperFamousName(s) != "" {
			t.Errorf("paperFamousName(%q) must be empty", s)
		}
	}
	allowed := map[string]bool{}
	for _, r := range real {
		allowed[r] = true
	}
	names := tplAllFamous()
	if len(names) < 20 {
		t.Fatalf("only %d famous names", len(names))
	}
	for _, p := range names {
		for _, bad := range paperBannedNames {
			if strings.Contains(p.name+p.en, bad) {
				t.Errorf("famous %q prints a forbidden real name", p.name)
			}
		}
	}
	for _, p := range ptPeople {
		if allowed[p.name] {
			t.Errorf("first name %q is a real person's name", p.name)
		}
	}
}

// ---- from the game --------------------------------------------------------------------------------------------------

// TestPaperTplFromGame: a real party: a UFO that came during it, a famous guest and the party's animals reach the facts,
// and the page prints a UFO news; a UFO of an earlier party does not count.
func TestPaperTplFromGame(t *testing.T) {
	old := paperSession
	t.Cleanup(func() { paperSession = old })
	paperSession = &paperSessionT{rng: newTestRand(4)}
	g := scoreGame(t)
	g.progress.Stats = map[string]int{fmt.Sprint("ev_", evUFO): 3} // UFOs of earlier parties
	g.lifeReset()
	if f := g.paperFactsNow(); f.rare["ufo"] {
		t.Fatal("a UFO of an earlier party counted")
	}
	g.stat(fmt.Sprint("ev_", evUFO), 1) // this party's
	var famous string
	for _, p := range tplAllFamous() {
		if !strings.ContainsAny(p.name, "áéíóúñÁÉÍÓÚÑ") {
			famous = p.name
			break
		}
	}
	probe := rand.New(rand.NewSource(1))
	for _, fp := range famousPresets {
		sp := fp(probe)
		if nameFromSpec(&sp) == famous {
			g.guests = append(g.guests, &Guest{cs: newCharSet(&sp), name: famous})
			break
		}
	}
	f := g.paperFactsNow()
	if !f.rare["ufo"] || f.rare["zombie"] {
		t.Errorf("rare set pieces: %v", f.rare)
	}
	if len(f.famous) == 0 || f.famous[0].name != famous {
		t.Errorf("famous of the party: %v (want %q)", f.famous, famous)
	}
	if f.theme == "" {
		t.Errorf("no level theme")
	}
	want := ""
	for _, p := range paperTplPicks(f) {
		if paperTplByID(p.id).ctx == "rare:ufo" {
			want = paperBriefText(paperBrief{p.head, p.es, p.en})
		}
	}
	found := false
	for _, b := range g.paperDataNow().briefs {
		if want != "" && b.es == want {
			found = true
		}
	}
	if !found {
		t.Errorf("the UFO party printed no UFO news")
	}
}

// TestPaperTplFinalEdition: the last newspaper prints two template news of the farewell, in the language in force, the
// same ones for the same ending.
func TestPaperTplFinalEdition(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			a, b := finalPaperData(42, false), finalPaperData(42, false)
			if fmt.Sprint(a.briefs) != fmt.Sprint(b.briefs) {
				t.Fatal("the last paper changes")
			}
			fin := paperTplFinalBriefs(42)
			if len(fin) != 2 || a.briefs[0] != fin[0] || a.briefs[1] != fin[1] {
				t.Fatalf("final template news: %v", fin)
			}
			for _, x := range fin {
				if strings.Contains(x.es, "{") || (lang == "en" && strings.ContainsAny(x.es+x.head, "«»¡¿")) {
					t.Errorf("%s: %q", lang, x.es)
				}
			}
			_, lay := renderFinalPaper(a, nil, 0)
			printed := strings.Join(lay.briefs, "|")
			if !strings.Contains(printed, fin[0].es) {
				t.Errorf("%s: the farewell news is not on the page", lang)
			}
		})
	}
}
