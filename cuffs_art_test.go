package main

import (
	"os"
	"strings"
	"testing"
)

func cuffLum(r, g, b uint8) float64 { return 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b) }

// Every place that shows the handcuffs goes through drawCuffsIcon / cuffsPix; no other drawing of them remains.
func TestCuffsSymbolSharedByAllUses(t *testing.T) {
	for _, f := range []string{"draw.go", "hudicons.go", "goalstats.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(src), "drawCuffsIcon(") {
			t.Errorf("%s does not use the shared drawCuffsIcon", f)
		}
	}
	for _, f := range []string{"game.go", "draw.go", "hudicons.go"} {
		src, _ := os.ReadFile(f)
		if strings.Contains(string(src), "genCuffs") || strings.Contains(string(src), "as.cuffs") {
			t.Errorf("%s still has the old handcuffs drawing", f)
		}
	}
}

// Both sizes stay readable: dark outline all around, steel clearly lighter than the outline, two open rings, the
// lime clasps, the blue chain and the gold tip; and the proportions of a pair of cuffs (a bit wider than tall).
func TestCuffsSymbolReadable(t *testing.T) {
	for _, tier := range []int{cuffsTiny, cuffsSmall} {
		b := cuffsPix(tier)
		w, h := cuffsSize(tier)
		if b.w != w || b.h != h {
			t.Fatalf("tier %d: sprite %dx%d, size says %dx%d", tier, b.w, b.h, w, h)
		}
		if r := float64(w) / float64(h); r < 1.05 || r > 1.45 {
			t.Errorf("tier %d: proportions %.2f", tier, r)
		}
		if w < 12 || h > 16 {
			t.Errorf("tier %d: %dx%d is outside 12..16", tier, w, h)
		}
		var navy, steel, lime, blue, gold, opaque int
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				c := b.get(x, y)
				if c.A == 0 {
					continue
				}
				opaque++
				switch {
				case c == cuffNavy:
					navy++
				case c == cuffLime:
					lime++
				case c == cuffGold:
					gold++
				case c == cuffChain[0] || c == cuffChain[1]:
					blue++
				default:
					steel++
					if cuffLum(c.R, c.G, c.B)-cuffLum(cuffNavy.R, cuffNavy.G, cuffNavy.B) < 60 {
						t.Errorf("tier %d: steel pixel (%d,%d) has little contrast against the outline", tier, x, y)
					}
				}
			}
		}
		if navy*100 < opaque*25 || navy*100 > opaque*65 {
			t.Errorf("tier %d: outline is %d of %d pixels", tier, navy, opaque)
		}
		if steel < 10 || lime < 2 || blue < 3 || gold < 1 {
			t.Errorf("tier %d: parts missing steel=%d lime=%d chain=%d gold=%d", tier, steel, lime, blue, gold)
		}
		// the two holes stay open (transparent, enclosed): probe the centre of each ring
		cy := h - cuffsGeoms[tier].d + (cuffsGeoms[tier].d-1)/2
		for _, cx := range []int{(cuffsGeoms[tier].d - 1) / 2, 3 * (cuffsGeoms[tier].d - 1) / 2} {
			if b.get(cx, cy).A != 0 && b.get(cx+1, cy).A != 0 {
				t.Errorf("tier %d: ring at %d,%d is not open", tier, cx, cy)
			}
		}
		// every outer border pixel row/column touching the edge of an opaque pixel is outlined: no steel on the canvas edge
		for x := 0; x < w; x++ {
			for _, y := range []int{0, h - 1} {
				if c := b.get(x, y); c.A != 0 && c != cuffNavy && c != cuffGold && c != cuffChain[0] && c != cuffChain[1] {
					t.Errorf("tier %d: steel on the canvas edge at %d,%d", tier, x, y)
				}
			}
		}
	}
}

// The HUD icon box and the career symbol keep the margins: the strip height fits the TIME row.
func TestCuffsHUDStripFits(t *testing.T) {
	if hudIconAdv(hudCuffs) < 12 || hudIconH(hudCuffs) > 11 {
		t.Errorf("hud cuffs box %vx%v", hudIconAdv(hudCuffs), hudIconH(hudCuffs))
	}
	_, h := cuffsSize(cuffsSmall)
	if pad := (careerPitch - 2 - float64(h)) / 2; pad < 6 {
		t.Errorf("career symbol padding %v < 6", pad)
	}
	if careerCuffsDY-(-5) < 6 || careerPitch-2-5-careerCuffsDY-float64(h) < 6 {
		t.Errorf("career symbol not 6 px inside its stripe")
	}
}
