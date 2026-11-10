package main

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The last newspaper comes right after the T-Rex eats the cake, before the scene at home and FIN; never in
// the normal ending; skipping it goes on (it does not jump to FIN).
func TestEndingFinalPaperOrder(t *testing.T) {
	if nextEndStage(endStinger, true) != endPaper || nextEndStage(endPaper, true) != endHome || nextEndStage(endHome, true) != endFinal {
		t.Fatal("secret ending: stinger -> newspaper -> home -> FIN")
	}
	for st := endInvite; st <= endHome; st++ {
		if nextEndStage(st, false) == endPaper && st != endStinger {
			t.Errorf("stage %d leads to the newspaper", st)
		}
	}
	for _, two := range []bool{false, true} {
		g := preGame(t, false)
		twoPlayerMode = two
		g.startEndingPre(true)
		g.end.stage, g.end.t = endStinger, endLenFor(endStinger, true)
		g.updateEnding()
		if g.end.stage != endPaper || g.end.skipLock < 0.5 {
			t.Fatalf("2P=%v: after the stinger comes stage %d (lock %.2f)", two, g.end.stage, g.end.skipLock)
		}
		g.end.t = endStageLen(g.end) // what a press does
		g.updateEnding()
		if g.end.stage != endHome {
			t.Fatalf("2P=%v: skipping the newspaper went to stage %d", two, g.end.stage)
		}
	}
	if l := endLenFor(endPaper, true); l < 20 {
		t.Errorf("the newspaper lasts only %.0fs: it must leave time to read it", l)
	}
}

// The FIN card of the secret ending keeps the player for 20 s (Murray talks): no hint and no way out, by Esc
// nor by Enter; after that the usual "back to the menu" is shown and works. The normal "¿ FINAL ?" (Murray
// talks there too) keeps the player a short 8 s, the same way.
func TestFinMenuLockedFor20Seconds(t *testing.T) {
	if finMenuOpen(true, 0) || finMenuOpen(true, 12) || finMenuOpen(true, 19.99) || !finMenuOpen(true, 20) || !finMenuOpen(true, 25) {
		t.Fatal("finMenuOpen: closed before 20 s, open from 20 s")
	}
	if finMenuOpen(false, 3) || finMenuOpen(false, 7.99) || !finMenuOpen(false, 8) || !finMenuOpen(false, 12) {
		t.Fatal("the normal ending: closed before 8 s, open from 8 s")
	}
	pi, pf := backInject, backFocus
	t.Cleanup(func() { backInject, backFocus = pi, pf })
	for _, c := range []struct {
		secret bool
		at     float64
		leave  bool
	}{{true, 3, false}, {true, 12, false}, {true, 19.5, false}, {true, 25, true}, {false, 3, false}, {false, 7.5, false}, {false, 8.5, true}} {
		g := layerGame(t, false)
		g.startEndingPre(c.secret)
		g.end.stage, g.end.t, g.end.total = endFinal, c.at, c.at
		textRec = &textRecorder{}
		img := ebiten.NewImage(screenW, screenH)
		g.drawEnding(img)
		img.Deallocate()
		hint := false
		for _, r := range textRec.texts { // the hint line at the bottom (Murray's bubble may say "salir" too)
			if r.y >= 480 && (strings.Contains(r.s, "menú") || strings.Contains(r.s, "salir") || strings.Contains(r.s, "saltar")) {
				hint = true
			}
		}
		textRec = nil
		if hint != c.leave {
			t.Errorf("t=%.1f: the way back shown=%v, want %v", c.at, hint, c.leave)
		}
		backInject = true
		g.updateEnding()
		backInject = false
		if left := g.end == nil; left != c.leave {
			t.Errorf("t=%.1f: Esc left the card=%v, want %v", c.at, left, c.leave)
		}
	}
}

// The typewriter of Murray's bubble: about 30 letters a second with small pauses at spaces and punctuation; the
// jaw opens and shuts at ~7 Hz while letters come out, rests in the pauses and is shut when the line is complete.
func TestFinMurrayTypewriterAndJaw(t *testing.T) {
	s := "Hola, mundo. Esto es una frase de prueba!"
	n := len([]rune(s))
	if v, o := finTypeAt(-1, s); v != 0 || o {
		t.Fatal("nothing before the line starts")
	}
	prev, flips, lastOpen := 0, 0, false
	for tt := 0.0; tt < 4; tt += 1.0 / 120 {
		v, o := finTypeAt(tt, s)
		if v < prev || v > n {
			t.Fatalf("t=%.2f: %d visible after %d", tt, v, prev)
		}
		if v == n && o {
			t.Fatalf("t=%.2f: the jaw moves after the line is complete", tt)
		}
		if tt < 1 && o != lastOpen {
			flips++
		}
		prev, lastOpen = v, o
	}
	if prev != n {
		t.Fatal("the line never completes")
	}
	if flips < 8 || flips > 20 { // ~7 Hz: 14 changes a second, fewer with the pauses
		t.Errorf("the jaw changed %d times in the first second, want a talking rhythm", flips)
	}
	// speed: about 30 letters a second
	if l := finTypeLen(s); l < float64(n)/30 || l > float64(n)/30+1.2 {
		t.Errorf("the line takes %.2fs for %d letters", l, n)
	}
	if v, _ := finTypeAt(0.5, strings.Repeat("a", 40)); v < 14 || v > 17 {
		t.Errorf("after 0.5 s %d letters", v)
	}
	// the pause after the comma: the jaw is shut a moment
	comma := strings.IndexRune(s, ',')
	at := 0.0
	for tt := 0.0; tt < 2; tt += 0.001 {
		if v, _ := finTypeAt(tt, s); v > comma {
			at = tt
			break
		}
	}
	if _, o := finTypeAt(at+0.12, s); o {
		t.Error("the jaw should rest in the pause after the comma")
	}
	// every line of Murray, in both languages, is fully typed well before the next one (>= 2 complete lines in 20 s)
	for _, p := range finMurrayLines {
		for _, l := range p {
			if d := finTypeLen(l); d > finSayEvery-2 {
				t.Errorf("%q takes %.1fs to type", l, d)
			}
		}
	}
	done := 0
	for slot := 0; ; slot++ {
		end := finSayFrom + float64(slot)*finSayEvery + finTypeLen(finMurrayLines[0][0])
		if end > finMenuAfter {
			break
		}
		done++
	}
	if done < 2 {
		t.Errorf("only %d full lines before the menu appears", done)
	}
}

// The special edition: headline, at least 7 news (report, the T-Rex, 5+ small ones; heads never repeated), every text in both languages and
// inside the page, the blocks do not overlap, the text is sharp; the same page with the real photo or the drawn one.
func TestFinalPaperPage(t *testing.T) {
	pw := paperNoWear
	t.Cleanup(func() { paperNoWear = pw })
	paperNoWear = true
	// a fake party frame: dark sky, bright figures
	fake := image.NewRGBA(image.Rect(0, 0, endCW, endCH))
	for y := 0; y < endCH; y++ {
		for x := 0; x < endCW; x++ {
			c := color.RGBA{20, 20, 60, 255}
			if y > 180 && (x/20)%2 == 0 {
				c = color.RGBA{240, 200, 120, 255}
			}
			fake.SetRGBA(x, y, c)
		}
	}
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			for _, two := range []bool{false, true} {
				for _, ph := range []*image.RGBA{nil, fake} {
					d := finalPaperData(42, two)
					if lang == "en" && d.head == "FIN DE LAS FIESTAS" {
						t.Fatal("headline not translated")
					}
					im, lay := renderFinalPaper(d, ph, 0)
					if len(lay.briefs)+2 < 7 { // the report and the T-Rex's words are news too
						t.Errorf("%s 2P=%v: only %d small news on the page", lang, two, len(lay.briefs))
					}
					heads := map[string]bool{}
					for _, h := range lay.heads {
						if heads[h] {
							t.Errorf("head %q twice", h)
						}
						heads[h] = true
					}
					found := false
					for _, tx := range lay.texts {
						if tx.s == d.head && tx.scale >= 2 {
							found = true
						}
						if tx.x < 0 || tx.y < 0 || tx.x+tx.w > screenW || tx.y+tx.h > screenH || tx.h < 7*tx.scale {
							t.Errorf("%s: %q leaves the page or is too small", lang, tx.s)
						}
					}
					if !found {
						t.Errorf("%s: no big headline", lang)
					}
					for i := range lay.texts {
						for j := i + 1; j < len(lay.texts); j++ {
							a, b := lay.texts[i], lay.texts[j]
							if prHit(prect{"", a.x, a.y, a.w, a.h}, prect{"", b.x, b.y, b.w, b.h}) {
								t.Errorf("%s: %q overlaps %q", lang, a.s, b.s)
							}
						}
					}
					var blocks []prect
					for _, b := range lay.blocks {
						if b.kind == "brief" || b.kind == "photo" || b.kind == "story" || b.kind == "quote" {
							blocks = append(blocks, b)
						}
					}
					for i := range blocks {
						for j := i + 1; j < len(blocks); j++ {
							if prHit(blocks[i], blocks[j]) {
								t.Errorf("blocks overlap: %v %v", blocks[i], blocks[j])
							}
						}
					}
					for _, tx := range lay.texts { // body texts stay in one block (no text crosses a column)
						if tx.y < pbY0 || tx.y >= pbY1 {
							continue
						}
						ok := false
						for _, b := range blocks {
							if prInside(prect{"", tx.x, tx.y, tx.w, tx.h}, b) {
								ok = true
							}
						}
						if !ok {
							t.Errorf("%s: %q crosses a block border", lang, tx.s)
						}
					}
					// the photo: only the four tones of the paper's photos inside its frame
					for y := pbY0 + 2; y < pbY0+fpPhotoH; y++ {
						for x := pcCX + 1; x < pcCX+fpPhotoW-1; x++ {
							p := im.RGBAAt(x, y)
							if p != photoTone[0] && p != photoTone[1] && p != photoTone[2] && p != photoTone[3] {
								t.Fatalf("photo pixel %v at %d,%d is not a sepia tone", p, x, y)
							}
						}
					}
				}
			}
		})
	}
	// the texts of the edition: every one in both languages, no real names
	inLang(t, "en", func() {
		for _, p := range fpTexts {
			if tr(p[0]) != p[1] {
				t.Errorf("tr(%q) = %q", p[0], tr(p[0]))
			}
		}
		for _, b := range fpBriefs {
			if tr(b.es) != b.en || tr(b.head) == b.head {
				t.Errorf("brief %q not translated", b.head)
			}
		}
	})
	if n := len(fpTexts) + 2*len(fpBriefs); n < 24 {
		t.Errorf("only %d new texts", n)
	}
}

// The real photo is used when there is one; the page is composed once and does not change while it is shown.
func TestFinalPaperStableAndUsesThePhoto(t *testing.T) {
	pw := paperNoWear
	t.Cleanup(func() { paperNoWear = pw })
	paperNoWear = true
	a, _ := renderFinalPaper(finalPaperData(7, false), nil, 0)
	ph := image.NewRGBA(image.Rect(0, 0, endCW, endCH))
	for i := range ph.Pix {
		ph.Pix[i] = 255
		if (i/4)%endCW < endCW/2 && i%4 != 3 {
			ph.Pix[i] = 0
		}
	}
	b, _ := renderFinalPaper(finalPaperData(7, false), ph, 0)
	diff := 0
	for y := pbY0 + 2; y < pbY0+fpPhotoH; y++ {
		for x := pcCX + 1; x < pcCX+fpPhotoW-1; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				diff++
			}
		}
	}
	if diff < 1000 {
		t.Error("the photo of the party is not printed on the page")
	}
	// half black, half white: the left of the printed photo is dark, the right light
	if l, r := b.RGBAAt(pcCX+40, pbY0+100), b.RGBAAt(pcCX+fpPhotoW-40, pbY0+100); l.R >= r.R {
		t.Errorf("the photo is not the picture: left %v right %v", l, r)
	}
	g := layerGame(t, false)
	g.startEndingPre(true)
	g.end.stage, g.end.t = endPaper, 1
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	g.time = 0
	g.drawEnding(img)
	d0 := g.end.x.fpaper.d
	k0 := paperTextDump(d0)
	for k := 1; k < 6; k++ {
		g.time = float64(k) * 0.37
		g.end.t += 0.37
		g.drawEnding(img)
	}
	if g.end.x.fpaper.d != d0 || paperTextDump(g.end.x.fpaper.d) != k0 {
		t.Error("the newspaper changed while it was on screen")
	}
}
