package main

import (
	"image"
	"testing"
)

func renderPaperBoth(d *paperData) (base, worn *image.RGBA, lay *paperLayout) {
	paperNoWear = true
	base, _ = renderPaper(d, 0)
	paperNoWear = false
	worn, lay = renderPaper(d, 0)
	return
}

// The second layer of wear (rips, stains, blood) never touches a letter and leaves the text boxes almost untouched: a
// background pixel under a stain keeps most of its contrast with the ink, and only a small share of the text area changes.
func TestPaperWearKeepsTextReadable(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			for vi, d := range paperVariants() {
				for _, k := range []int{0, 30} {
					d.kills = k
					base, worn, lay := renderPaperBoth(d)
					total, changed := 0, 0
					for _, tx := range lay.texts {
						for y := tx.y; y < tx.y+tx.h; y++ {
							for x := tx.x; x < tx.x+tx.w; x++ {
								b, w := base.RGBAAt(x, y), worn.RGBAAt(x, y)
								total++
								if b == w {
									continue
								}
								changed++
								if tx.title || tx.dark {
									t.Fatalf("%s v%d: the wear touched %q at %d,%d", lang, vi, tx.s, x, y)
								}
								if b == tx.col {
									t.Fatalf("%s v%d: a letter of %q changed at %d,%d", lang, vi, tx.s, x, y)
								}
								if o, n := pinkDist(b, tx.col), pinkDist(w, tx.col); n < 140 || n*100 < o*paperWearKeep-100 {
									t.Fatalf("%s v%d: %q loses contrast at %d,%d: %d -> %d", lang, vi, tx.s, x, y, o, n)
								}
							}
						}
					}
					if changed*100 > total*4 {
						t.Errorf("%s v%d kills %d: the wear changes %d of %d text pixels (> 4%%)", lang, vi, k, changed, total)
					}
				}
			}
		})
	}
}

// The rips are big (the desk shows 12 px or more into the page), the empty bottom strip is stained, blood appears when
// there were victims, everything depends on the seed and the same seed paints the same page.
func TestPaperWearBigRipsStainsBloodBySeed(t *testing.T) {
	var masks []map[int]bool
	bloodPages := 0
	for seed := int64(1); seed <= 8; seed++ {
		d := paperSample(paperFacts{seed: seed * 7919, kills: 9}, seed%2 == 0, false, 3)
		base, worn, _ := renderPaperBoth(d)
		again, _ := renderPaper(d, 0)
		if string(again.Pix) != string(worn.Pix) {
			t.Fatalf("seed %d: the same page painted twice differs", seed)
		}
		m := map[int]bool{}
		bottom, red := 0, 0
		for y := 0; y < screenH; y++ {
			for x := 0; x < screenW; x++ {
				b, w := base.RGBAAt(x, y), worn.RGBAAt(x, y)
				if b == w {
					continue
				}
				m[y*screenW+x] = true
				if y >= 488 && pDist(w, paperDesk) >= 40 {
					bottom++
				}
				if int(w.R) > int(w.G)+60 && int(w.R) > int(w.B)+50 && int(w.G) < 90 {
					red++
				}
			}
		}
		masks = append(masks, m)
		if bottom < 400 {
			t.Errorf("seed %d: only %d stained pixels in the bottom strip", seed, bottom)
		}
		if red >= 20 {
			bloodPages++
		}
		// the deepest rip on each long edge
		deep := func(e int) int {
			best := 0
			long := screenW
			if e >= 2 {
				long = screenH
			}
			for a := 0; a < long; a++ {
				dd := 0
				for dd < 60 {
					x, y := ripXY(e, a, dd)
					if pDist(worn.RGBAAt(x, y), paperDesk) >= 40 {
						break
					}
					dd++
				}
				if dd > best {
					best = dd
				}
			}
			return best
		}
		if dt, db := deep(0), deep(1); dt < 12 || db < 12 {
			t.Errorf("seed %d: rips too small: top %d px, bottom %d px", seed, dt, db)
		}
	}
	if bloodPages < 6 {
		t.Errorf("blood on only %d of 8 pages with victims", bloodPages)
	}
	for i := 1; i < len(masks); i++ {
		same := 0
		for p := range masks[i] {
			if masks[0][p] {
				same++
			}
		}
		if same*2 > len(masks[i]) {
			t.Errorf("seed %d wears the page like seed 1 (%d of %d pixels in common)", i+1, same, len(masks[i]))
		}
	}
}
