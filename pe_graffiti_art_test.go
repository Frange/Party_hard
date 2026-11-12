package main

import (
	"image/color"
	"image/png"
	"os"
	"testing"
)

// Every design of the graffiti library is a real little picture: >= 4 colours besides the outline, a dark outline all
// round, not empty, not noise (most pixels belong to one connected shape, few colour changes per pixel), it fits its box.
func TestGraffitiDesignsQuality(t *testing.T) {
	if len(gfLib) < 10 || len(gfLib) > 16 {
		t.Fatalf("%d designs, want 10-16", len(gfLib))
	}
	names := map[string]bool{}
	for _, a := range gfLib {
		if names[a.name] {
			t.Errorf("%s twice", a.name)
		}
		names[a.name] = true
		if a.w > gfMaxW || a.h > gfMaxH || a.w < 13 || a.h < 8 {
			t.Errorf("%s: %dx%d does not fit the box %dx%d", a.name, a.w, a.h, gfMaxW, gfMaxH)
		}
		cols := map[color.RGBA]int{}
		opaque, ink := 0, 0
		for _, c := range a.pix {
			if c.A == 0 {
				continue
			}
			opaque++
			if c == gfInk {
				ink++
				continue
			}
			cols[c]++
		}
		if len(cols) < 4 || len(cols) > 7 {
			t.Errorf("%s: %d colours besides the outline, want 4-6", a.name, len(cols))
		}
		fill := float64(opaque) / float64(a.w*a.h)
		if fill < 0.30 || fill > 0.97 {
			t.Errorf("%s: %.0f %% of the box painted", a.name, fill*100)
		}
		if ink == 0 || float64(ink)/float64(opaque) > 0.75 {
			t.Errorf("%s: outline %d of %d pixels", a.name, ink, opaque)
		}
		// the outline: no coloured pixel touches bare wall (4 neighbours) or the box edge
		for y := 0; y < a.h; y++ {
			for x := 0; x < a.w; x++ {
				c := a.at(x, y)
				if c.A == 0 || c == gfInk {
					continue
				}
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					xx, yy := x+d[0], y+d[1]
					if xx < 0 || yy < 0 || xx >= a.w || yy >= a.h || a.at(xx, yy).A == 0 {
						t.Errorf("%s: colour at %d,%d without outline", a.name, x, y)
					}
				}
			}
		}
		// not noise: the biggest connected shape holds most of the paint, and colours change rarely along a row
		seen := make([]bool, len(a.pix))
		best := 0
		for i := range a.pix {
			if seen[i] || a.pix[i].A == 0 {
				continue
			}
			n, stack := 0, []int{i}
			seen[i] = true
			for len(stack) > 0 {
				j := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				n++
				x, y := j%a.w, j/a.w
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					xx, yy := x+d[0], y+d[1]
					if xx < 0 || yy < 0 || xx >= a.w || yy >= a.h {
						continue
					}
					k := yy*a.w + xx
					if !seen[k] && a.pix[k].A != 0 {
						seen[k] = true
						stack = append(stack, k)
					}
				}
			}
			if n > best {
				best = n
			}
		}
		if float64(best)/float64(opaque) < 0.7 {
			t.Errorf("%s: the main shape has only %d of %d pixels (scattered)", a.name, best, opaque)
		}
		changes := 0
		for y := 0; y < a.h; y++ {
			for x := 1; x < a.w; x++ {
				if a.at(x, y) != a.at(x-1, y) {
					changes++
				}
			}
		}
		if r := float64(changes) / float64(a.w*a.h); r > 0.6 {
			t.Errorf("%s: %.2f colour changes per pixel (noise)", a.name, r)
		}
	}
}

// GRAFFITI_SHEET=dir writes the contact sheet (every design at x3) for the review of the user.
func TestGraffitiContactSheet(t *testing.T) {
	dir := os.Getenv("GRAFFITI_SHEET")
	if dir == "" {
		t.Skip("set GRAFFITI_SHEET=dir to write the contact sheet")
	}
	f, err := os.Create(dir + "/graffiti_sheet_x3.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, gfContactSheet(3)); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("GRAFFITI_BIG") != "" { // a bigger copy, for a close look
		g, err := os.Create(dir + "/graffiti_sheet_x8.png")
		if err != nil {
			t.Fatal(err)
		}
		defer g.Close()
		if err := png.Encode(g, gfContactSheet(8)); err != nil {
			t.Fatal(err)
		}
	}
}
