package main

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"unicode/utf8"
)

const paperLinesMin = 3

func paperCases() []paperFacts {
	var out []paperFacts
	for i, k := range []int{0, 2, 8, 25} {
		for _, seen := range []int{0, 2, 6} {
			f := paperFacts{level: "7", kills: k, seen: seen, knife: k, seed: int64(i*100 + seen)}
			out = append(out, f)
			g := f
			g.knife, g.poison = 0, k
			out = append(out, g)
			g.poison, g.blast, g.cuffed, g.camera, g.slasher, g.medic = 0, k, 2, true, true, true
			g.units = 1<<unitPolice | 1<<unitSWAT | 1<<unitMilitary
			g.two, g.jailed, g.challenge, g.chainMax = true, true, true, 5
			out = append(out, g)
			h := f
			h.knife, h.traps, h.poison = k/3, k/3, k-2*(k/3)
			h.units = 1 << unitFBI
			out = append(out, h)
		}
	}
	return out
}

func TestPaperStoryShapeBothLanguages(t *testing.T) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	loadLanguages()
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, f := range paperCases() {
			d := paperSample(f, false, false, 5)
			if len(d.story) < paperLinesMin {
				t.Fatalf("%s %+v: %d sentences, want >= %d", lang, f, len(d.story), paperLinesMin)
			}
			for _, l := range d.story {
				if l == "" || utf8.RuneCountInString(l) > 110 {
					t.Fatalf("%s: bad sentence %q", lang, l)
				}
			}
			if h := d.head; h == "" || strings.Contains(h, "%") || utf8.RuneCountInString(h) > 52 {
				t.Errorf("%s: bad headline %q", lang, h)
			}
		}
	}
}

func TestPaperEveryTextHasEnglish(t *testing.T) {
	loadLanguages()
	for _, f := range paperCases() {
		for _, s := range paperSentences(f) {
			if _, ok := langDataEN[s]; !ok {
				t.Errorf("no English for %q", s)
			}
		}
		if _, ok := langDataEN[paperQuote(f)]; !ok {
			t.Errorf("no English for quote %q", paperQuote(f))
		}
	}
}

func TestPaperDeterministicAndChangesWithStats(t *testing.T) {
	a := paperFacts{level: "3", kills: 12, knife: 12, seen: 0, seed: 77}
	if strings.Join(paperSentences(a), "|") != strings.Join(paperSentences(a), "|") || paperHeadline(a) != paperHeadline(a) {
		t.Fatal("same facts, different paper")
	}
	seen := map[string]bool{}
	for _, f := range paperCases() {
		seen[strings.Join(paperSentences(f), "|")] = true
	}
	if len(seen) < 20 {
		t.Errorf("only %d different stories over %d cases", len(seen), len(paperCases()))
	}
	// the method decides the opening; being seen / not seen shows up
	p := a
	p.knife, p.poison = 0, 12
	if paperSentences(a)[0] == paperSentences(p)[0] {
		t.Error("poison and knife open the same")
	}
	un := strings.Join(paperSentences(a), "|")
	b := a
	b.seen = 6
	if un == strings.Join(paperSentences(b), "|") {
		t.Error("being seen changes nothing")
	}
	if !strings.Contains(paperHeadline(a), "3") {
		t.Errorf("the headline lacks the party number: %q", paperHeadline(a))
	}
	// no real people
	for _, f := range paperCases() {
		all := strings.Join(paperSentences(f), " ") + paperHeadline(f)
		for _, bad := range []string{"Trump", "Messi", "Obama", "Putin", "Mulder", "Scully"} {
			if strings.Contains(all, bad) {
				t.Errorf("real name %q in the paper", bad)
			}
		}
	}
}

func TestPaperFactsTwoPlayers(t *testing.T) {
	g := scoreGame(t)
	g.p1 = g.p
	g.p2 = &Player{idx: 1, pos: g.p.pos, cs: g.p.cs}
	t.Cleanup(func() { g.p1, g.p2 = nil, nil })
	g.killGuest(liveGuest(g, 0), true)
	g.asPlayer(1, func() { g.killGuest(liveGuest(g, 0), true) })
	f := g.paperFactsNow()
	if f.kills != 2 || !f.two || len(paperSentences(f)) < paperLinesMin {
		t.Errorf("2P facts %+v", f)
	}
}

// ---- the page itself: sharpness, layout, columns, small news ------------------------------------------------------

// paperSample: a complete page for the tests (no Game): facts, a points table and the stats of one or two players.
func paperSample(f paperFacts, two, challenge bool, day int) *paperData {
	if f.level == "" {
		f.level = "7"
	}
	f.two = two
	fin := finishBreak{base: 1210, chain: 180, clean: 60, complete: bonusComplete, unseen: bonusUnseen, speed: 40, total: 2100, wallet: 2100, challenge: challenge}
	if challenge {
		fin.killExtra, fin.finalExtra = 600, 900
	}
	pl := []paperPlayer{{kills: [kcN]int{kcKnife: 9, kcPoison: 3, kcTrap: 2}, chainMax: 6, clean: 4, seen: 3, cuff: 1}}
	if two {
		pl = append(pl, paperPlayer{kills: [kcN]int{kcKnife: 4, kcBlast: 5}, chainMax: 3, clean: 2, seen: 1,
			coop: coopStats{points: 570, hits: 2, floorT: 9, helps: 2}, title: "Rey de la pista"})
		pl[0].coop, pl[0].title = coopStats{points: 640, hits: 1, floorT: 3}, "Reina del ponche"
	}
	return newPaperData(f, fin, pl, 187, day, "ENTER: siguiente fiesta", 2)
}

func paperVariants() []*paperData {
	var out []*paperData
	for _, two := range []bool{false, true} {
		for i, f := range paperCases() {
			if i%9 != 0 {
				continue
			}
			f.seed += int64(i * 31)
			out = append(out, paperSample(f, two, i%2 == 0, 1+i%27))
		}
	}
	return out
}

func inLang(t *testing.T, lang string, fn func()) {
	old := uiLang
	t.Cleanup(func() { uiLang = old })
	loadLanguages()
	uiLang = lang
	fn()
	uiLang = old
}

func pDist(a, b color.RGBA) int {
	d := func(x, y uint8) int { return absI(int(x) - int(y)) }
	return d(a.R, b.R) + d(a.G, b.G) + d(a.B, b.B)
}

// SHARPNESS (mandatory for any text screen): every text of the page is a whole scale at whole pixels, the letters are 7 px
// tall at least, and inside each text box a pixel is EITHER exactly the ink colour OR clearly the background: no
// anti-aliased or blended in-between colours, no blur, no halo, no translucent text.
func TestPaperTextIsSharpAndWhole(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			for vi, d := range paperVariants() {
				for st := 0; st < 2; st++ {
					im, lay := renderPaper(d, st)
					if im.Rect.Dx() != screenW || im.Rect.Dy() != screenH {
						t.Fatalf("page is %v, want %dx%d", im.Rect, screenW, screenH)
					}
					for i := 3; i < len(im.Pix); i += 4 {
						if im.Pix[i] != 255 {
							t.Fatalf("translucent pixel (alpha %d) at byte %d: text and paper must be opaque", im.Pix[i], i)
						}
					}
					if len(lay.texts) < 30 {
						t.Fatalf("only %d texts on the page", len(lay.texts))
					}
					for _, tx := range lay.texts {
						if tx.scale < 1 || tx.w <= 0 || tx.h < 7*tx.scale {
							t.Fatalf("%s v%d: %q scale %d size %dx%d (letters need >= 7 px)", lang, vi, tx.s, tx.scale, tx.w, tx.h)
						}
						if tx.x < 0 || tx.y < 0 || tx.x+tx.w > screenW || tx.y+tx.h > screenH {
							t.Fatalf("%s v%d: %q leaves the page (%d,%d %dx%d)", lang, vi, tx.s, tx.x, tx.y, tx.w, tx.h)
						}
						if tx.title {
							continue
						}
						hit := 0
						for y := tx.y; y < tx.y+tx.h; y++ {
							for x := tx.x; x < tx.x+tx.w; x++ {
								p := im.RGBAAt(x, y)
								dd := pDist(p, tx.col)
								switch {
								case dd == 0:
									hit++
								case dd <= 90:
									t.Fatalf("%s v%d st%d: %q has a blended pixel %v at %d,%d (ink %v)", lang, vi, st, tx.s, p, x, y, tx.col)
								}
							}
						}
						if hit < len([]rune(tx.s))/2 {
							t.Fatalf("%s: %q: only %d ink pixels", lang, tx.s, hit)
						}
					}
				}
			}
		})
	}
}

// The same text laid on a flat colour has exactly two colours: a test of the canvas itself, at every scale.
func TestPaperCanvasTextHasTwoColoursOnly(t *testing.T) {
	for sc := 1; sc <= 3; sc++ {
		lay := &paperLayout{}
		c := &pcv{im: image.NewRGBA(image.Rect(0, 0, 400, 80)), lay: lay}
		c.fill(0, 0, 400, 80, paperCream)
		c.text("Ñandú Él JUERGA ¡ábc! 0123", 4, 4, sc, paperInk)
		c.bold("Bold ñ 99", 4, 40, paperRed)
		seen := map[color.RGBA]bool{}
		for y := 0; y < 80; y++ {
			for x := 0; x < 400; x++ {
				seen[c.im.RGBAAt(x, y)] = true
			}
		}
		if len(seen) != 3 {
			t.Errorf("scale %d: %d colours, want 3 (paper, ink, red): %v", sc, len(seen), seen)
		}
	}
}

func TestPaperHasTexture(t *testing.T) {
	d := paperSample(paperFacts{seed: 4}, false, false, 3)
	im, _ := renderPaper(d, 0)
	seen := map[color.RGBA]bool{}
	for y := 300; y < 340; y++ { // an area of the margin, no text
		for x := 0; x < 10; x++ {
			seen[im.RGBAAt(x, y)] = true
		}
	}
	if len(seen) < 8 {
		t.Errorf("the paper is flat: %d colours in a blank area", len(seen))
	}
	// the vertical fold is darker than the paper 30 px away; the corner is darker than the centre (vignette)
	mid := func(x, y0, y1 int) int {
		n, s := 0, 0
		for y := y0; y < y1; y++ {
			p := im.RGBAAt(x, y)
			s += int(p.R) + int(p.G) + int(p.B)
			n++
		}
		return s / n
	}
	if mid(screenW/2, 484, 488) >= mid(screenW/2-30, 484, 488) {
		t.Error("no visible vertical fold")
	}
	if mid(30, 8, 12) >= mid(130, 8, 12)+1 && mid(2, 20, 30) >= mid(40, 20, 30) {
		t.Error("no vignette at the edges")
	}
	if p := im.RGBAAt(screenW-2, 0); pDist(p, paperDesk) > 40 {
		t.Errorf("the torn corner shows %v, want the desk", p)
	}
	// deterministic
	im2, _ := renderPaper(d, 0)
	if string(im.Pix) != string(im2.Pix) {
		t.Error("the same data painted two different pages")
	}
}

func prInside(a, b prect) bool {
	return a.x >= b.x && a.y >= b.y && a.x+a.w <= b.x+b.w && a.y+a.h <= b.y+b.h
}

func prHit(a, b prect) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

func TestPaperLayoutNoOverlapFullCoverage(t *testing.T) {
	leaf := map[string]bool{"stats": true, "brief": true, "photo": true, "story": true, "quote": true, "score": true}
	body := prect{"body", pbX0, pbY0, pbX1 - pbX0, pbY1 - pbY0}
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			for vi, d := range paperVariants() {
				_, lay := renderPaper(d, 0)
				var leaves []prect
				area := 0
				for _, b := range lay.blocks {
					if !leaf[b.kind] {
						continue
					}
					if !prInside(b, prect{"", 0, 0, screenW, screenH}) || b.w <= 0 || b.h <= 0 {
						t.Fatalf("%s v%d: block %v outside the page", lang, vi, b)
					}
					leaves = append(leaves, b)
					area += b.w * b.h
				}
				for i := range leaves {
					for j := i + 1; j < len(leaves); j++ {
						if prHit(leaves[i], leaves[j]) {
							t.Fatalf("%s v%d: blocks overlap: %v and %v", lang, vi, leaves[i], leaves[j])
						}
					}
				}
				if cov := float64(area) / float64(body.w*body.h); cov < 0.9 {
					t.Errorf("%s v%d: the blocks cover %.0f%% of the body, want >= 90%%", lang, vi, cov*100)
				}
				// every text of the body sits inside ONE block (no text crosses a block border)
				for _, tx := range lay.texts {
					r := prect{"", tx.x, tx.y, tx.w, tx.h}
					if tx.y < pbY0 || tx.y >= pbY1 {
						continue
					}
					ok := false
					for _, b := range leaves {
						if prInside(r, b) {
							ok = true
						}
					}
					if !ok {
						t.Errorf("%s v%d: %q (%v) crosses a block border", lang, vi, tx.s, r)
					}
				}
				// no two texts touch
				for i := range lay.texts {
					for j := i + 1; j < len(lay.texts); j++ {
						a, b := lay.texts[i], lay.texts[j]
						if prHit(prect{"", a.x, a.y, a.w, a.h}, prect{"", b.x, b.y, b.w, b.h}) {
							t.Errorf("%s v%d: %q overlaps %q", lang, vi, a.s, b.s)
						}
					}
				}
			}
		})
	}
}

func TestPaperPlayerColumns(t *testing.T) {
	for _, two := range []bool{false, true} {
		d := paperSample(paperFacts{seed: 9}, two, false, 2)
		_, lay := renderPaper(d, 0)
		var stats []prect
		rightBlocks, rightH := 0, 0
		for _, b := range lay.blocks {
			if b.kind == "stats" {
				stats = append(stats, b)
			}
			if b.x == pcRX && b.kind != "rowA" {
				rightBlocks++
				rightH += b.h
				if !two && b.kind != "brief" {
					t.Errorf("1P: the right column holds a %s block, want only small news", b.kind)
				}
			}
		}
		if len(stats) != map[bool]int{false: 1, true: 2}[two] {
			t.Fatalf("two=%v: %d stats blocks", two, len(stats))
		}
		if stats[0].x != pcLX {
			t.Errorf("J1 statistics at x=%d, want the left column (%d)", stats[0].x, pcLX)
		}
		if two {
			if stats[1].x != pcRX || stats[1].h != stats[0].h && stats[1].h < statBoxH(d.stats[1]) {
				t.Errorf("J2 statistics at x=%d, want the right column (%d)", stats[1].x, pcRX)
			}
			if len(d.stats[0].rows) != len(d.stats[1].rows) || d.stats[0].head == d.stats[1].head {
				t.Error("J1 and J2 boxes must have the same structure and different titles")
			}
		} else if rightBlocks < 3 || rightH != pbY1-pbY0 {
			t.Errorf("1P: the right column has %d small news over %d px, want it filled (%d px)", rightBlocks, rightH, pbY1-pbY0)
		}
	}
}

func TestPaperBriefsDistinctDeterministicAndSeeded(t *testing.T) {
	lists := map[string]bool{}
	for seed := int64(0); seed < 12; seed++ {
		f := paperFacts{level: "5", seed: seed * 977, kills: 8}
		d := paperSample(f, seed%2 == 0, false, 4)
		_, lay := renderPaper(d, 0)
		if len(lay.briefs) < map[bool]int{false: 5, true: 3}[seed%2 == 0] {
			t.Fatalf("seed %d: only %d small news", seed, len(lay.briefs))
		}
		set := map[string]bool{}
		for _, b := range lay.briefs {
			if set[b] {
				t.Fatalf("seed %d: the brief %q appears twice on one page", seed, b)
			}
			set[b] = true
		}
		_, lay2 := renderPaper(paperSample(f, seed%2 == 0, false, 4), 0)
		if strings.Join(lay.briefs, "|") != strings.Join(lay2.briefs, "|") {
			t.Fatalf("seed %d: not deterministic", seed)
		}
		lists[strings.Join(lay.briefs, "|")] = true
	}
	if len(lists) < 10 {
		t.Errorf("only %d different pages of small news over 12 seeds", len(lists))
	}
}

func TestPaperBriefBankBothLanguagesNoRealNames(t *testing.T) {
	if len(paperBriefBank) < 40 {
		t.Fatalf("%d briefs, want 40+", len(paperBriefBank))
	}
	loadLanguages()
	es := map[string]bool{}
	for _, b := range paperBriefBank {
		if es[b.es] {
			t.Errorf("duplicate brief %q", b.es)
		}
		es[b.es] = true
		if b.en == "" || b.en == b.es || langDataEN[b.es] != b.en {
			t.Errorf("no English for %q", b.es)
		}
		if _, ok := paperBriefHeads[b.head]; !ok {
			t.Errorf("head %q has no English", b.head)
		}
		if n := utf8.RuneCountInString(b.es); n > 90 {
			t.Errorf("brief too long (%d): %q", n, b.es)
		}
		for _, bad := range []string{"Trump", "Messi", "Obama", "Putin", "Biden", "Shakira", "Madonna"} {
			if strings.Contains(b.es+b.en, bad) {
				t.Errorf("real name %q", bad)
			}
		}
	}
	inLang(t, "en", func() {
		for _, f := range paperCases()[:8] {
			for _, b := range paperDerived(f, "2:05") {
				if tr(b.es) == b.es || tr(b.head) == b.head {
					t.Errorf("derived brief not translated: %q", b.es)
				}
			}
		}
		d := paperSample(paperFacts{seed: 3, kills: 5, seen: 2}, true, true, 9)
		for _, s := range []string{d.title, d.head, d.caption, d.banner, d.scoreHead, d.storyHead, d.quoteHead, d.hint, d.foot, d.earL1, d.date, d.motto} {
			if s == "" || strings.ContainsAny(s, "ÁÉÍÓÚÑ¡¿") && !strings.Contains(s, "THE") {
				t.Errorf("English page keeps Spanish text: %q", s)
			}
		}
		if d.banner != "PRESS TO CONTINUE" || d.title != "THE PARTY GAZETTE" {
			t.Errorf("English banner/title: %q / %q", d.banner, d.title)
		}
	})
	inLang(t, "es", func() {
		d := paperSample(paperFacts{seed: 3}, false, false, 9)
		if d.banner != "PULSA PARA CONTINUAR" || d.title != "EL DIARIO DE LA JUERGA" {
			t.Errorf("Spanish banner/title: %q / %q", d.banner, d.title)
		}
	})
}

func TestPaperTitleLettersAndBanner(t *testing.T) {
	for _, s := range []string{"EL DIARIO DE LA JUERGA", "THE PARTY GAZETTE"} {
		for _, r := range s {
			if r == ' ' {
				continue
			}
			if _, ok := paperGlyphCells(r); !ok {
				t.Errorf("title %q: no glyph for %q", s, r)
			}
		}
		if w := paperTitleW(s, 4); 480-w/2 < 20+150+8 {
			t.Errorf("title %q (%d px) runs into the ears", s, w)
		}
	}
	// the banner alternates between two sharp states: only its box changes, the rest of the page is identical
	d := paperSample(paperFacts{seed: 2}, false, false, 5)
	a, la := renderPaper(d, 0)
	b, _ := renderPaper(d, 1)
	if a.RGBAAt(pbBannerX+5, pbBannerY+5) == b.RGBAAt(pbBannerX+5, pbBannerY+5) {
		t.Error("the two banner states look the same")
	}
	bn := la.banner
	for y := 0; y < screenH; y++ {
		for x := 0; x < screenW; x++ {
			if x >= bn.x && x < bn.x+bn.w && y >= bn.y && y < bn.y+bn.h {
				continue
			}
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				t.Fatalf("the page differs outside the banner at %d,%d", x, y)
			}
		}
	}
	// the banner is in the lower centre and does not touch the other texts
	if bn.x+bn.w/2 != screenW/2 || bn.y < pbY1 {
		t.Errorf("banner at %v, want lower centre", bn)
	}
}

// "Press to continue" on the newspaper goes STRAIGHT on to what the old results page led to (no extra screen).
func TestPaperContinueGoesStraightOn(t *testing.T) {
	defer flowReset()
	g := flowGame(2, false)
	if g.mode != ModeLevelDone {
		t.Fatal("setup")
	}
	g.levelDoneContinue()
	if g.mode == ModeLevelDone || g.mode != ModePlay || g.levelIdx != 3 {
		t.Fatalf("after the newspaper: mode %v level %d, want the next party", g.mode, g.levelIdx)
	}
	g = flowGame(4, false) // the car interlude comes next, not another results page
	g.levelDoneContinue()
	if g.mode != ModeInterlude {
		t.Fatalf("after party 5: mode %v, want the interlude", g.mode)
	}
}

// A play-through of the Game side: the newspaper is drawn from the real game data without panicking, 1P and 2P.
func TestPaperDataFromGame(t *testing.T) {
	g := scoreGame(t)
	g.p1 = g.p
	g.p2 = &Player{idx: 1, pos: g.p.pos, cs: g.p.cs}
	t.Cleanup(func() { g.p1, g.p2 = nil, nil })
	g.killGuest(liveGuest(g, 0), true)
	g.asPlayer(1, func() { g.killGuest(liveGuest(g, 0), true) })
	d := g.paperDataNow()
	if len(d.stats) != 2 || d.stats[0].rows[0][1] != "1" || d.stats[1].rows[0][1] != "1" {
		t.Errorf("2P stats %+v", d.stats)
	}
	if len(d.score) < 6 || d.total == "" {
		t.Errorf("score table %+v", d.score)
	}
}
