package main

import (
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Every line of Murray on the FIN card: Spanish and English, short, no duplicates, translated by tr.
func TestFinMurrayLinesBothLanguages(t *testing.T) {
	if n := len(finMurrayLines); n < 38 {
		t.Fatalf("only %d lines, want about 40", n)
	}
	defer func(a string) { uiLang = a }(uiLang)
	seenES, seenEN := map[string]bool{}, map[string]bool{}
	for i, p := range finMurrayLines {
		es, en := p[0], p[1]
		if es == "" || en == "" || es == en {
			t.Errorf("line %d: missing translation %q / %q", i, es, en)
		}
		if len([]rune(es)) > 48 || len([]rune(en)) > 48 {
			t.Errorf("line %d too long: %q / %q", i, es, en)
		}
		if seenES[es] || seenEN[en] {
			t.Errorf("line %d duplicated: %q / %q", i, es, en)
		}
		seenES[es], seenEN[en] = true, true
		uiLang = "en"
		if got := tr(es); got != en {
			t.Errorf("tr(%q) in English = %q, want %q", es, got, en)
		}
		uiLang = "es"
		if got := tr(es); got != es {
			t.Errorf("tr(%q) in Spanish = %q", es, got)
		}
	}
}

// A new line every 7 seconds (one less than before): the slot is a pure function of the time on the card;
// every line of both cards is typed well before the next one; the menu locks stay at 20 s (secret) and
// 8 s (normal).
func TestFinMurrayEvery7Seconds(t *testing.T) {
	if finSayEvery != 7 {
		t.Fatalf("finSayEvery = %v, want 7", finSayEvery)
	}
	if finMurraySlot(0) != -1 || finMurraySlot(finSayFrom-0.01) != -1 {
		t.Fatal("Murray talks before the card is ready")
	}
	for s := 0; s < 20; s++ {
		at := finSayFrom + float64(s)*7
		if finMurraySlot(at) != s || finMurraySlot(at+6.99) != s || finMurraySlot(at+7) != s+1 {
			t.Fatalf("slot %d does not last exactly 7 s", s)
		}
	}
	if finMenuOpen(true, 19.9) || !finMenuOpen(true, 20) || finMenuOpen(false, 7.9) || !finMenuOpen(false, 8) {
		t.Error("the menu locks changed (want 20 s secret, 8 s normal)")
	}
	for _, set := range [][][2]string{finMurrayLines, finMurrayLinesNormal} {
		for _, p := range set {
			for _, l := range p {
				if d := finTypeLen(l); d > finSayEvery-2 {
					t.Errorf("%q takes %.1fs to type, the next line comes after %.0fs", l, d, finSayEvery)
				}
			}
		}
	}
}

// Murray looks at the middle of the screen on both cards: the sprite faces left (jaw forward on the left,
// cranium bulging back to the right), so it is mirrored on the left (secret) card and not on the right
// (normal) one; and finMurrayLook moves the glints (his pupils) forward and down.
func TestFinMurrayLooksAtCentre(t *testing.T) {
	k := propKinds["murray"]
	if k == nil || k.Draw == nil {
		t.Fatal("no murray prop")
	}
	draw := func(look bool) *pixBuf {
		b := newPix(k.W+2, k.H+2)
		b.ox, b.oy = 1, 1
		k.Draw(b, 0)
		if look {
			finMurrayLook(b)
		}
		return b
	}
	at := func(b *pixBuf, x, y int) [4]byte {
		i := ((y+b.oy)*b.w + x + b.ox) * 4
		return [4]byte{b.p[i], b.p[i+1], b.p[i+2], b.p[i+3]}
	}
	// which way the sprite faces: the jaw (forward) is left of the back of the cranium
	b := draw(false)
	meanX := func(y0, y1 int) float64 {
		n, sx := 0.0, 0.0
		for y := y0; y <= y1; y++ {
			for x := 0; x < k.W; x++ {
				if p := at(b, x, y); p[3] > 0 && p[0] > 150 {
					n, sx = n+1, sx+float64(x)
				}
			}
		}
		return sx / n
	}
	if meanX(23, 26) >= meanX(2, 10) {
		t.Fatal("the murray sprite no longer faces left: revisit the mirror of finMurrayGeom")
	}
	// the glints, relative to the centres of the sockets (7,12) and (15.5,12)
	glint := func(b *pixBuf) (dx, dy float64) {
		n := 0.0
		for y := 8; y <= 16; y++ {
			for x := 3; x <= 19; x++ {
				if p := at(b, x, y); p[3] > 0 && p[0] == 0xff && p[1] == 0xff && p[2] == 0xff {
					cx := 7.0
					if x >= 11 {
						cx = 15.5
					}
					dx, dy, n = dx+float64(x)+0.5-cx, dy+float64(y)+0.5-12, n+1
				}
			}
		}
		if n != 8 {
			t.Fatalf("%v glint pixels, want 2x2 in each socket", n)
		}
		return dx / n, dy / n
	}
	if dx, dy := glint(draw(true)); dx > -0.7 || dy < 0.9 {
		t.Errorf("the pupils look (%.1f, %.1f) in the sprite, want forward (left) and down", dx, dy)
	}
	for _, trueEnd := range []bool{true, false} {
		footX, _, _, _, mirror := finMurrayGeom(trueEnd)
		face := -1.0 // the sprite faces left
		if mirror {
			face = 1
		}
		if (screenW/2-footX)*face <= 0 {
			t.Errorf("trueEnd=%v: Murray at x=%.0f faces %+.0f, away from the middle", trueEnd, footX, face)
		}
	}
}

// Random order, every line once per round, never the same line twice in a row.
func TestFinMurrayNoRepeats(t *testing.T) {
	n := len(finMurrayLines)
	for _, seed := range []int64{1, 2, 3, 42, 1993, 123456789} {
		prev := -1
		for c := 0; c < 6; c++ {
			seen := map[int]bool{}
			for i := 0; i < n; i++ {
				l := finMurrayLine(seed, c*n+i)
				if l < 0 || l >= n || seen[l] {
					t.Fatalf("seed %d round %d: line %d repeated or out of range", seed, c, l)
				}
				if l == prev {
					t.Fatalf("seed %d: line %d said twice in a row", seed, l)
				}
				if finMurrayLine(seed, c*n+i) != l {
					t.Fatal("not a pure function")
				}
				seen[l], prev = true, l
			}
		}
	}
	if finMurrayLine(1, 0) == finMurrayLine(2, 0) && finMurrayLine(1, 1) == finMurrayLine(2, 1) && finMurrayLine(1, 2) == finMurrayLine(2, 2) {
		t.Error("the order does not depend on the seed")
	}
}

// The normal "¿ FINAL ?" has its own Murray: at least 25 new lines, in Spanish and English, short, none
// repeated and none taken from the secret FIN; quick enough to type; every round says each one once.
func TestFinMurrayNormalLines(t *testing.T) {
	if n := len(finMurrayLinesNormal); n < 25 {
		t.Fatalf("only %d lines for the normal ending, want >= 25", n)
	}
	prev := uiLang
	t.Cleanup(func() { uiLang = prev })
	secret := map[string]bool{}
	for _, p := range finMurrayLines {
		secret[strings.ToLower(p[0])], secret[strings.ToLower(p[1])] = true, true
	}
	seen := map[string]bool{}
	for i, p := range finMurrayLinesNormal {
		es, en := p[0], p[1]
		if es == "" || en == "" || es == en {
			t.Errorf("normal line %d: missing translation %q / %q", i, es, en)
		}
		if len([]rune(es)) > 48 || len([]rune(en)) > 48 {
			t.Errorf("normal line %d too long: %q / %q", i, es, en)
		}
		for _, l := range p {
			if secret[strings.ToLower(l)] {
				t.Errorf("normal line %d is a line of the secret FIN: %q", i, l)
			}
			if seen[strings.ToLower(l)] {
				t.Errorf("normal line %d duplicated: %q", i, l)
			}
			seen[strings.ToLower(l)] = true
			if d := finTypeLen(l); d > finSayEvery-2 {
				t.Errorf("%q takes %.1fs to type", l, d)
			}
		}
		uiLang = "en"
		if got := tr(es); got != en {
			t.Errorf("tr(%q) in English = %q, want %q", es, got, en)
		}
		uiLang = "es"
		if got := tr(es); got != es {
			t.Errorf("tr(%q) in Spanish = %q", es, got)
		}
	}
	if len(finMurraySet(false)) != len(finMurrayLinesNormal) || len(finMurraySet(true)) != len(finMurrayLines) {
		t.Error("finMurraySet does not pick the set of each ending")
	}
	n := len(finMurrayLinesNormal)
	for _, seed := range []int64{1, 7, 1993} {
		last := -1
		for c := 0; c < 4; c++ {
			got := map[int]bool{}
			for i := 0; i < n; i++ {
				l := finMurrayLineN(seed, c*n+i, n)
				if l < 0 || l >= n || got[l] || l == last {
					t.Fatalf("seed %d round %d: line %d repeated or out of range", seed, c, l)
				}
				got[l], last = true, l
			}
		}
	}
	// the longest line is fully typed before the way back to the menu appears
	longest := 0.0
	for _, p := range finMurrayLinesNormal {
		for _, l := range p {
			longest = math.Max(longest, finTypeLen(l))
		}
	}
	if finSayFrom+longest > finMenuNorm {
		t.Errorf("a line takes until %.1fs, the menu opens at %.0fs", finSayFrom+longest, finMenuNorm)
	}
}

// Geometry of the normal card, Spanish and English, with every one of his lines: Murray and his bubble
// (ink border and tail included) stay >= 16 px from the screen edges and >= 12 px from every text of the card
// ("¿ FINAL ?" with its wobble, the two lines, "Gracias por jugar", the hint); the bubble does not cover him;
// coordinates are whole pixels; and Murray is really drawn there (his bubble text is recorded).
func TestFinMurrayNormalLayout(t *testing.T) {
	prev := uiLang
	t.Cleanup(func() { uiLang = prev; textRec = nil })
	type box struct{ x0, y0, x1, y1 float64 }
	gap := func(a, b box) float64 {
		dx := math.Max(a.x0-b.x1, b.x0-a.x1)
		dy := math.Max(a.y0-b.y1, b.y0-a.y1)
		return math.Max(dx, dy)
	}
	for _, lang := range []string{"es", "en"} {
		g := layerGame(t, false)
		uiLang = lang
		g.startEndingPre(false)
		g.end.stage, g.end.t, g.end.total = endFinal, 10, 10
		textRec = &textRecorder{}
		img := ebiten.NewImage(screenW, screenH)
		g.drawEnding(img)
		img.Deallocate()
		recs := textRec.texts
		textRec = nil
		im := g.as.props["murray"][0]
		footX, footY, _, _, mirror := finMurrayGeom(false)
		if mirror || !finBubbleLeft(false) {
			t.Fatal("the normal card keeps the sprite as drawn (facing left, to the middle) with the bubble on his left")
		}
		sw, sh := float64(im.img.Bounds().Dx())*4, float64(im.img.Bounds().Dy())*4
		mur := box{footX - im.ox*4, footY - im.oy*4, footX - im.ox*4 + sw, footY - im.oy*4 + sh}
		var card []box
		bubbleSeen := false
		for _, r := range recs {
			b := box{r.x, r.y, r.x + r.w, r.y + r.h}
			if r.scale == finBubScale && b.y1 <= 136 { // the bubble's own text (above the title)
				bubbleSeen = true
				continue
			}
			if r.scale >= 7 { // "¿ FINAL ?" wobbles 2 px
				b.y0, b.y1 = b.y0-2, b.y1+2
			}
			card = append(card, b)
		}
		if !bubbleSeen {
			t.Errorf("[%s] Murray's bubble text was not drawn on the normal card", lang)
		}
		if len(card) < 4 {
			t.Fatalf("[%s] only %d texts on the card", lang, len(card))
		}
		check := func(what string, b box) {
			if b.x0 < 16 || b.y0 < 16 || b.x1 > screenW-16 || b.y1 > screenH-16 {
				t.Errorf("[%s] %s %v closer than 16 px to the edge", lang, what, b)
			}
			for _, c := range card {
				if d := gap(b, c); d < 12 {
					t.Errorf("[%s] %s %v only %.0f px from a text %v", lang, what, b, d, c)
				}
			}
		}
		check("Murray", mur)
		{ // worst case whatever the line: a full 3-line bubble as wide as the wrap
			x0, y0, x1, y1 := finBubbleBox(false, finBubWrap, 3)
			check("worst-case bubble", box{x0, y0, x1, y1})
		}
		for _, p := range finMurrayLinesNormal {
			lines := g.finBubbleLines(p[0])
			w := 0.0
			for _, l := range lines {
				w = math.Max(w, g.textW(l, finBubScale))
			}
			checkBubbleText(t, lang, p[0], lines)
			bx, by, bw, bh := finBubbleRect(false, w, len(lines))
			if bx != math.Round(bx) || by != math.Round(by) || bw != math.Round(bw) || bh != math.Round(bh) {
				t.Errorf("[%s] bubble not on whole pixels: %v %v %v %v", lang, bx, by, bw, bh)
			}
			// the border is 2 px; the tail hangs under it, towards him (right corner)
			x0, y0, x1, y1 := finBubbleBox(false, w, len(lines))
			bub := box{x0, y0, x1, y1}
			check("bubble of "+p[0], bub)
			if bub.x1 > mur.x0 && bub.y1 > mur.y0 && bub.x0 < mur.x1 && bub.y0 < mur.y1 && bx+bw > mur.x0 {
				t.Errorf("[%s] the bubble %v covers Murray %v", lang, bub, mur)
			}
		}
	}
}

// Geometry of the REAL ending card (secret FIN), Spanish and English, with every one of his lines: Murray and his bubble
// (ink border and tail included) stay >= 16 px from the screen edges and >= 12 px from every text of the card
// ("¿ FINAL ?" with its wobble, the two lines, "Gracias por jugar", the hint); the bubble does not cover him;
// coordinates are whole pixels; and Murray is really drawn there (his bubble text is recorded).
func TestFinMurrayRealLayout(t *testing.T) {
	prev := uiLang
	t.Cleanup(func() { uiLang = prev; textRec = nil })
	type box struct{ x0, y0, x1, y1 float64 }
	gap := func(a, b box) float64 {
		dx := math.Max(a.x0-b.x1, b.x0-a.x1)
		dy := math.Max(a.y0-b.y1, b.y0-a.y1)
		return math.Max(dx, dy)
	}
	for _, lang := range []string{"es", "en"} {
		g := layerGame(t, true)
		uiLang = lang
		g.startEndingPre(true)
		g.end.stage, g.end.t, g.end.total = endFinal, 10, 10
		textRec = &textRecorder{}
		img := ebiten.NewImage(screenW, screenH)
		g.drawEnding(img)
		img.Deallocate()
		recs := textRec.texts
		textRec = nil
		im := g.as.props["murray"][0]
		footX, footY, _, _, mirror := finMurrayGeom(true)
		if !mirror || finBubbleLeft(true) {
			t.Fatal("the real card mirrors the sprite (so he faces right, to the middle) with the bubble on his right")
		}
		sw, sh := float64(im.img.Bounds().Dx())*4, float64(im.img.Bounds().Dy())*4
		mur := box{footX - im.ox*4, footY - im.oy*4, footX - im.ox*4 + sw, footY - im.oy*4 + sh}
		var card []box
		bubbleSeen := false
		for _, r := range recs {
			b := box{r.x, r.y, r.x + r.w, r.y + r.h}
			if r.scale == finBubScale && b.x0 >= mur.x1 && b.y1 <= 136 { // the bubble's own text (right of him, above "FIN")
				bubbleSeen = true
				continue
			}
			if r.scale >= 7 { // "FIN" may wobble
				b.y0, b.y1 = b.y0-2, b.y1+2
			}
			card = append(card, b)
		}
		if !bubbleSeen {
			t.Errorf("[%s] Murray's bubble text was not drawn on the normal card", lang)
		}
		if len(card) < 4 {
			t.Fatalf("[%s] only %d texts on the card", lang, len(card))
		}
		check := func(what string, b box) {
			if b.x0 < 16 || b.y0 < 16 || b.x1 > screenW-16 || b.y1 > screenH-16 {
				t.Errorf("[%s] %s %v closer than 16 px to the edge", lang, what, b)
			}
			for _, c := range card {
				if d := gap(b, c); d < 12 {
					t.Errorf("[%s] %s %v only %.0f px from a text %v", lang, what, b, d, c)
				}
			}
		}
		check("Murray", mur)
		// worst case whatever the line: a full 3-line bubble as wide as the wrap
		{
			x0, y0, x1, y1 := finBubbleBox(true, finBubWrap, 3)
			check("worst-case bubble", box{x0, y0, x1, y1})
		}
		for _, p := range finMurrayLines {
			lines := g.finBubbleLines(p[0])
			w := 0.0
			for _, l := range lines {
				w = math.Max(w, g.textW(l, finBubScale))
			}
			checkBubbleText(t, lang, p[0], lines)
			bx, by, bw, bh := finBubbleRect(true, w, len(lines))
			if bx != math.Round(bx) || by != math.Round(by) || bw != math.Round(bw) || bh != math.Round(bh) {
				t.Errorf("[%s] bubble not on whole pixels: %v %v %v %v", lang, bx, by, bw, bh)
			}
			// the border is 2 px; the tail hangs under it, towards him (left corner)
			x0, y0, x1, y1 := finBubbleBox(true, w, len(lines))
			bub := box{x0, y0, x1, y1}
			check("bubble of "+p[0], bub)
			if bub.x1 > mur.x0 && bub.y1 > mur.y0 && bub.x0 < mur.x1 && bub.y0 < mur.y1 {
				t.Errorf("[%s] the bubble %v covers Murray %v", lang, bub, mur)
			}
		}
	}
}

// The way out is never announced by Murray while the menu is locked (20 s): no line of the real ending
// talks about pressing a button to leave.
func TestFinMurrayNoExitHint(t *testing.T) {
	for _, p := range finMurrayLines {
		for _, l := range p {
			low := strings.ToLower(l)
			if strings.Contains(low, "press a button") || strings.Contains(low, "pulsa un bot") {
				t.Errorf("exit hint inside a Murray line: %q", l)
			}
		}
	}
}

// checkBubbleText: a line of Murray fits his bubble whole (<= 3 lines, nothing cut with "..."), at a whole
// text scale bigger than the old 1x, with >= 6 px of paper around the ink of its letters.
func checkBubbleText(t *testing.T, lang, line string, lines []string) {
	t.Helper()
	if len(lines) > 3 || strings.Join(lines, " ") != strings.Join(strings.Fields(tr(line)), " ") {
		t.Errorf("[%s] %q does not fit the bubble whole: %q", lang, line, lines)
	}
	if finBubScale < 2 || pxScale(finBubScale) != finBubScale {
		t.Errorf("bubble text scale %v: want a whole scale >= 2 (crisp)", finBubScale)
	}
	// the font cell is 16 px at 1x: the tallest letters start 2 px into it (accents a bit higher), descenders
	// end at 14; the text is drawn at by+pad-4 and every line finBubLH lower
	_, by, _, bh := finBubbleRect(true, 100, len(lines))
	top := by + finBubPad - 4 + 2*finBubScale
	bot := by + finBubPad - 4 + float64(len(lines)-1)*finBubLH + 14*finBubScale
	if top-by < 6 || by+bh-bot < 6 {
		t.Errorf("bubble padding: %.0f px above the ink, %.0f below, want >= 6", top-by, by+bh-bot)
	}
	if finBubLH < 13*finBubScale {
		t.Errorf("line pitch %v: the lines would touch", finBubLH)
	}
}
