package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// ---- variety: the paper does not say the same thing party after party --------------------------------------------

// situations: the main situations of a party, to check each one has plenty of words.
func paperSituations() map[string]paperFacts {
	base := func(k int) paperFacts {
		return paperFacts{level: "7", levelN: 7, kills: k, knife: k, seen: 2, secs: 200}
	}
	none := paperFacts{level: "7", levelN: 7, secs: 200}
	clean := base(9)
	clean.seen = 0
	blast := base(9)
	blast.knife, blast.blast = 0, 9
	poison := base(9)
	poison.knife, poison.poison = 0, 9
	trap := base(9)
	trap.knife, trap.traps = 0, 9
	few := base(2)
	many := base(24)
	cops := base(9)
	cops.cuffed, cops.units = 2, 1<<unitPolice
	animals := base(9)
	animals.animalsDead = 5
	oneAnimal := base(9)
	oneAnimal.animalsDead = 1
	mixed := base(9)
	mixed.knife, mixed.poison, mixed.traps = 3, 3, 3
	alive := none
	alive.animalsAlive = 4
	return map[string]paperFacts{"none": none, "clean": clean, "blast": blast, "poison": poison, "trap": trap, "few": few,
		"many": many, "cops": cops, "animals": animals, "oneAnimal": oneAnimal, "mixed": mixed, "aliveAnimals": alive}
}

func TestPaperManyHeadlinesAndOpeningsPerSituation(t *testing.T) {
	for name, f := range paperSituations() {
		heads, opens := map[string]bool{}, map[string]bool{}
		for seed := int64(0); seed < 400; seed++ {
			g := f
			g.seed = seed*7919 + 13
			heads[paperHeadlineForm(g)] = true
			opens[paperSentences(g)[0]] = true
		}
		if len(heads) < 8 {
			t.Errorf("%s: only %d different headlines, want 8+", name, len(heads))
		}
		if len(opens) < 8 {
			t.Errorf("%s: only %d different story openings, want 8+", name, len(opens))
		}
	}
}

func TestPaperPoolsAreBigAndDistinct(t *testing.T) {
	seen := map[string]bool{}
	n := 0
	for _, pool := range paperAllPools() {
		if len(pool) < 2 {
			t.Errorf("a pool of words has %d entries", len(pool))
		}
		for _, p := range pool {
			n++
			if seen[p.es] {
				t.Errorf("duplicate Spanish text %q", p.es)
			}
			seen[p.es] = true
			if p.en == "" || p.en == p.es {
				t.Errorf("no English for %q", p.es)
			}
		}
	}
	if n < 400 {
		t.Errorf("only %d texts in the pools of the page, want 400+", n)
	}
	if len(paperBriefBank) < 130 {
		t.Errorf("only %d small news in the bank, want 130+", len(paperBriefBank))
	}
	for _, pool := range [][]paperTxt{pOpenKnife, pOpenPoison, pOpenTrap, pOpenBlast, pOpenMixed, pOpenNone, pSeen0, pSeenFew, pSeenMany, pPolice, pQuotes, pClose} {
		if len(pool) < 8 {
			t.Errorf("a main pool has only %d variants (%q)", len(pool), pool[0].es)
		}
	}
}

// Simulated session: many parties in a row with the same result. The words keep changing and the headline never repeats
// the one of the party before.
func TestPaperSessionDoesNotRepeatParties(t *testing.T) {
	old := paperSession
	t.Cleanup(func() { paperSession = old })
	paperSession = &paperSessionT{rng: newTestRand(5)}
	f := paperFacts{level: "7", levelN: 7, kills: 9, knife: 9, seen: 2, cuffed: 1, units: 1 << unitPolice, secs: 200}
	heads, opens, briefs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	prev := ""
	var prevStory, prevBriefs string
	for i := 0; i < 40; i++ {
		g := paperVary(f, "party-"+string(rune('A'+i)))
		h := paperHeadlineForm(g)
		if h == prev {
			t.Fatalf("party %d: the same headline as the party before: %q", i, h)
		}
		prev = h
		heads[h] = true
		st := paperSentences(g)
		opens[st[0]] = true
		if strings.Join(st, "|") == prevStory {
			t.Fatalf("party %d: the same story as the party before", i)
		}
		prevStory = strings.Join(st, "|")
		var bs []string
		for _, b := range paperBriefsFor(g, "3:00")[:10] {
			bs = append(bs, b.es)
			briefs[b.es] = true
		}
		if strings.Join(bs, "|") == prevBriefs {
			t.Fatalf("party %d: the same small news as the party before", i)
		}
		prevBriefs = strings.Join(bs, "|")
		// the same party printed twice (every frame) gives the same words
		again := paperVary(f, "party-"+string(rune('A'+i)))
		if paperHeadlineForm(again) != h || strings.Join(paperSentences(again), "|") != prevStory {
			t.Fatalf("party %d: the page changed between two frames", i)
		}
	}
	if len(heads) < 10 || len(opens) < 8 || len(briefs) < 40 {
		t.Errorf("40 parties gave %d headlines, %d openings and %d small news; want 10+, 8+ and 40+", len(heads), len(opens), len(briefs))
	}
}

// Nothing printed overflows: every headline fits the page width, every word fits its column and every sentence is short,
// in both languages and for the party numbers with the most characters.
func TestPaperTextsFitTheirColumns(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			for name, f := range paperSituations() {
				for _, lvl := range []string{"7", "21", "EXTRA"} {
					for seed := int64(0); seed < 60; seed++ {
						g := f
						g.level, g.seed = lvl, seed*104729+3
						d := paperSample(g, seed%2 == 0, false, 9)
						if n := utf8.RuneCountInString(d.head); n > 52 || paperTextW(d.head)*2 > 920 {
							t.Errorf("%s/%s %s: headline too long (%d): %q", lang, name, lvl, n, d.head)
						}
						for _, s := range d.story {
							if utf8.RuneCountInString(s) > 110 {
								t.Errorf("%s: sentence too long: %q", lang, s)
							}
							checkWords(t, s, pcCW-pbPhotoW-12-10)
						}
						checkWords(t, d.quote, pcCW-pbPhotoW-12-10)
						if len(wrapPx(d.caption, pbPhotoW)) > 3 {
							t.Errorf("%s: caption over 3 lines: %q", lang, d.caption)
						}
						for _, b := range d.briefs {
							if utf8.RuneCountInString(b.es) > 100 {
								t.Errorf("%s: brief too long: %q", lang, b.es)
							}
							checkWords(t, b.es, pcLW-10)
							checkWords(t, b.head, pcLW-10)
						}
					}
				}
			}
			// the whole bank and every pool, word by word
			for _, b := range paperBriefBank {
				checkWords(t, tr(b.es), pcLW-10)
			}
			for _, pool := range paperAllPools() {
				for _, p := range pool {
					checkWords(t, tr(p.es), pcCW-pbPhotoW-12-10)
				}
			}
		})
	}
}

// checkWords: no single word is wider than the column (wrapPx could not break it and it would run out of the box).
func checkWords(t *testing.T, s string, w int) {
	t.Helper()
	for _, word := range splitWords(s) {
		if paperTextW(word) > w {
			t.Errorf("the word %q of %q is wider than its column (%d px)", word, s, w)
		}
	}
}

// the printed small news never repeat on a page and fit their column (several lines, but inside the box)
func TestPaperBriefsFitInTheirBoxes(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			for seed := int64(0); seed < 40; seed++ {
				d := paperSample(paperFacts{seed: seed * 31, kills: int(seed % 12), seen: int(seed % 4), animalsDead: int(seed % 3)}, seed%2 == 0, false, 3)
				_, lay := renderPaper(d, 0)
				for _, b := range lay.blocks {
					if b.kind != "brief" {
						continue
					}
					if b.x < 0 || b.y < pbY0 || b.x+b.w > screenW || b.y+b.h > pbY1 {
						t.Fatalf("%s: a small news box leaves the body: %+v", lang, b)
					}
				}
				for _, tx := range lay.texts {
					if tx.x+tx.w > screenW-12 {
						t.Fatalf("%s: %q runs into the right margin (%d)", lang, tx.s, tx.x+tx.w)
					}
				}
			}
		})
	}
}

// ---- the owner of each statistics column ----------------------------------------------------------------------------

func TestPaperStatsColumnsShowJ1J2(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			want := map[string]string{"es": "J", "en": "P"}[lang]
			for _, two := range []bool{false, true} {
				d := paperSample(paperFacts{seed: 21, kills: 6}, two, false, 3)
				n := map[bool]int{false: 1, true: 2}[two]
				if len(d.stats) != n {
					t.Fatalf("two=%v: %d stat boxes", two, len(d.stats))
				}
				im, lay := renderPaper(d, 0)
				for i := 0; i < n; i++ {
					tag := want + string(rune('1'+i))
					if d.stats[i].tag != tag {
						t.Errorf("%s two=%v: column %d is tagged %q, want %q", lang, two, i, d.stats[i].tag, tag)
					}
					found := false
					for _, tx := range lay.texts {
						if tx.s != tag || !tx.dark {
							continue
						}
						found = true
						// the tag sits on the dark label, inside its own column, and every pixel is ink or paper colour
						col := pcLX
						if i == 1 {
							col = pcRX
						}
						if tx.x < col || tx.x+tx.w > col+pcLW || tx.y < pbY0 || tx.y+tx.h > pbY0+statHeadH+2 && i == 0 {
							t.Errorf("%s: tag %q is outside its column header (%+v)", lang, tag, tx)
						}
						if p := im.RGBAAt(tx.x-2, tx.y+4); pDist(p, paperInk) != 0 {
							t.Errorf("%s: tag %q is not on a dark label (%v)", lang, tag, p)
						}
					}
					if !found {
						t.Errorf("%s two=%v: the tag %q is not on the page", lang, two, tag)
					}
				}
			}
		})
	}
}

// ---- the paper looks used ------------------------------------------------------------------------------------------

func TestPaperLooksWorn(t *testing.T) {
	d := paperSample(paperFacts{seed: 12}, false, false, 3)
	im, _ := renderPaper(d, 0)
	// the cigarette burn: brown and black pixels in the top-right margin, and a hole to the desk
	dark, brown, hole := 0, 0, 0
	for y := 0; y < 22; y++ {
		for x := 880; x < 950; x++ {
			p := im.RGBAAt(x, y)
			switch {
			case pDist(p, paperDesk) < 40:
				hole++
			case int(p.R)+int(p.G)+int(p.B) < 200:
				dark++
			case int(p.R) > int(p.B)+60 && int(p.R) < 200:
				brown++
			}
		}
	}
	if hole < 12 || dark < 30 || brown < 60 {
		t.Errorf("cigarette burn missing: hole %d, charred %d, brown %d pixels", hole, dark, brown)
	}
	// the top edge is irregular: the desk shows through at several different depths
	depths := map[int]bool{}
	for x := 0; x < 940; x++ {
		dd := 0
		for dd < 14 && pDist(im.RGBAAt(x, dd), paperDesk) < 40 {
			dd++
		}
		depths[dd] = true
	}
	if len(depths) < 5 {
		t.Errorf("the top edge is too straight: %d different depths", len(depths))
	}
	// stains and creases change the page: a page with them differs from one without in many pixels of the margins
	left := map[uint32]bool{}
	for y := 0; y < screenH; y++ {
		for x := 0; x < 13; x++ {
			p := im.RGBAAt(x, y)
			left[uint32(p.R)<<16|uint32(p.G)<<8|uint32(p.B)] = true
		}
	}
	if len(left) < 120 {
		t.Errorf("the left margin has only %d colours: no stains?", len(left))
	}
	// the reading text is not covered: the sharpness test (TestPaperTextIsSharpAndWhole) checks every box; here a quick look
	// at the story area: its background is light
	sum, n := 0, 0
	for y := 250; y < 300; y++ {
		for x := 520; x < 560; x++ {
			p := im.RGBAAt(x, y)
			sum += int(p.R) + int(p.G) + int(p.B)
			n++
		}
	}
	if sum/n < 400 {
		t.Errorf("the middle of the page is too dark: %d", sum/n)
	}
}
