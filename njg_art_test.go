package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// njgSlugs: Neo and the Joker, redrawn in njg_art.go.
var njgSlugs = []string{"neo", "joker"}

// NJG_SHEET=dir [NJG_OLD=dir-with-old-folders] [NJG_ONLY=a,b] go test -run TestNjgSheet writes an
// enlarged sheet: the old exported pictures (if NJG_OLD), then the code today (front, side, back,
// two walk frames in profile and a front walk frame).
func TestNjgSheet(t *testing.T) {
	out := os.Getenv("NJG_SHEET")
	if out == "" {
		t.Skip("set NJG_SHEET")
	}
	slugs := append([]string{}, njgSlugs...)
	slugs = append(slugs, "gb_venkman", "gb_stantz", "gb_spengler", "gb_winston")
	if o := os.Getenv("NJG_ONLY"); o != "" {
		slugs = strings.Split(o, ",")
	}
	oldDir := os.Getenv("NJG_OLD")
	const z = 5
	cols := 6
	if oldDir != "" {
		cols += 3
	}
	cellH := charH*z + 6
	big := image.NewRGBA(image.Rect(0, 0, cols*charW*z+10, len(slugs)*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x78, 0x80, 0x78, 255
	}
	put := func(b *pixBuf, col, row int) {
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				i := (y*b.w + x) * 4
				if b.p[i+3] == 0 {
					continue
				}
				c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
				for dy := 0; dy < z; dy++ {
					for dx := 0; dx < z; dx++ {
						big.SetRGBA(col*charW*z+x*z+dx, row*cellH+y*z+dy, c)
					}
				}
			}
		}
	}
	for r, slug := range slugs {
		col := 0
		if oldDir != "" {
			if a, err := loadCharAsset(filepath.Join(oldDir, slug)); err == nil {
				for i := 0; i < 3; i++ {
					put(a.Views[i].outlinedSel(0.42), i, r)
				}
			}
			col = 3
		}
		sp := charimpASpec(t, slug)
		for i, vw := range []int{vFront, vSide, vBack} {
			put(renderChar(&sp, vw, poseFor(AnimIdle, 0)), col+i, r)
		}
		for i, c := range []struct{ vw, f int }{{vSide, 1}, {vSide, 3}, {vFront, 1}} {
			pp := poseFor(AnimWalk, c.f)
			pp.Anim, pp.F = AnimWalk, c.f
			put(renderChar(&sp, c.vw, pp), col+3+i, r)
		}
	}
	f, err := os.Create(filepath.Join(out, "njg_sheet.png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}

// njgAllSlugs: the six drawn in njg_art.go.
var njgAllSlugs = []string{"neo", "joker", "gb_venkman", "gb_stantz", "gb_spengler", "gb_winston"}

// The six draw in the three views inside their box, their own drawing really shows (against the
// same spec under another name) and walking changes the frames.
func TestNjgDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic, AnimLaugh}
	for _, slug := range njgAllSlugs {
		sp := charimpASpec(t, slug)
		if !isNjg(&sp) {
			t.Fatalf("%s: not drawn by njg_art.go", slug)
		}
		twin := sp
		twin.Name = "twin"
		for vw := 0; vw < 3; vw++ {
			differs := false
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					pp := poseFor(a, f)
					pp.Anim, pp.F = a, f
					b, tb := renderChar(&sp, vw, pp), renderChar(&twin, vw, pp)
					n, edge := charimpAPix(b)
					_, tEdge := charimpAPix(tb)
					if edge > tEdge+2 {
						t.Errorf("%s vw %d anim %d f %d: %d pixels on the edge (plain %d)", slug, vw, a, f, edge, tEdge)
					}
					if n < 100 { // slim Egon in profile is about 115
						t.Errorf("%s vw %d anim %d f %d: only %d pixels", slug, vw, a, f, n)
					}
					if string(b.p) != string(tb.p) {
						differs = true
					}
				}
			}
			if !differs {
				t.Errorf("%s vw %d: the own drawing never shows", slug, vw)
			}
			w0 := charimpAWalkFrame(&sp, vw, 0)
			changed := false
			for f := 1; f < animFrameCount(AnimWalk); f++ {
				if string(charimpAWalkFrame(&sp, vw, f)) != string(w0) {
					changed = true
				}
			}
			if !changed {
				t.Errorf("%s vw %d: walking does not change the frames", slug, vw)
			}
		}
	}
}

func njgCount(b *pixBuf, y0, y1 int, ok func(c color.RGBA) bool) int {
	n := 0
	for y := y0; y < y1; y++ {
		for x := 0; x < b.w; x++ {
			i := (y*b.w + x) * 4
			if b.p[i+3] > 0 && ok(color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}) {
				n++
			}
		}
	}
	return n
}

func njgDark(c color.RGBA) bool { return c.R < 0x40 && c.G < 0x40 && c.B < 0x48 }

// Neo: a black coat down to the ankles, wide below the hips, whose tails swing behind him in
// profile while he walks; the small black shades and pale skin.
func TestNjgNeoCoat(t *testing.T) {
	sp := charimpASpec(t, "neo")
	b := genChar(&sp, vFront, poseFor(AnimIdle, 0))
	for y := 27; y <= 37; y++ {
		if w := njgCount(b, y, y+1, njgDark); w < 6 {
			t.Errorf("row %d: the coat is only %d pixels wide", y, w)
		}
	}
	if n := njgCount(b, 3, 13, func(c color.RGBA) bool { return c.R > 0xe0 && c.G > 0xc8 && c.B > 0xb8 }); n < 15 {
		t.Errorf("pale face: %d pixels", n)
	}
	if b.get(9, 7).A == 0 || !njgDark(b.get(9, 7)) || !njgDark(b.get(14, 7)) {
		t.Errorf("no sunglasses: %v %v", b.get(9, 7), b.get(14, 7))
	}
	// in profile the tails trail further back while walking than standing
	back := func(p Pose) int {
		b := genChar(&sp, vSide, p)
		for x := 0; x < b.w; x++ {
			for y := 30; y < 39; y++ {
				if b.get(x, y).A > 0 {
					return x
				}
			}
		}
		return 99
	}
	idle := back(poseFor(AnimIdle, 0))
	moved := false
	for f := 0; f < animFrameCount(AnimWalk); f++ {
		pp := poseFor(AnimWalk, f)
		pp.Anim, pp.F = AnimWalk, f
		if back(pp) < idle {
			moved = true
		}
	}
	if !moved {
		t.Errorf("the coat tails never fly behind while walking (idle back edge %d)", idle)
	}
}

// Joker: green hair, white face with the red grin and teeth, purple suit with the yellow
// waistcoat and green shirt, white gloves.
func TestNjgJokerLook(t *testing.T) {
	sp := charimpASpec(t, "joker")
	b := genChar(&sp, vFront, poseFor(AnimIdle, 0))
	checks := []struct {
		what   string
		y0, y1 int
		min    int
		ok     func(c color.RGBA) bool
	}{
		{"green hair", 2, 9, 12, func(c color.RGBA) bool { return c.G > 0x80 && c.R < 0x80 && c.B < 0x70 }},
		{"red grin", 8, 13, 6, func(c color.RGBA) bool { return c.R > 0xb0 && c.G < 0x40 && c.B < 0x50 }},
		{"teeth", 9, 12, 3, func(c color.RGBA) bool { return c.R > 0xf0 && c.G > 0xf0 && c.B > 0xe0 }},
		{"yellow-green eyes", 6, 9, 2, func(c color.RGBA) bool { return c.R > 0xb0 && c.G > 0xb0 && c.B < 0x60 }},
		{"purple suit", 14, 40, 60, func(c color.RGBA) bool { return c.B > c.G+0x30 && c.R > c.G+0x20 }},
		{"yellow waistcoat", 14, 24, 10, func(c color.RGBA) bool { return c.R > 0xd0 && c.G > 0xa0 && c.B < 0x50 }},
		{"white gloves and shoe tops", 22, 44, 6, func(c color.RGBA) bool { return c.R > 0xe8 && c.G > 0xe8 && c.B > 0xe0 }},
	}
	for _, c := range checks {
		if n := njgCount(b, c.y0, c.y1, c.ok); n < c.min {
			t.Errorf("%s: %d pixels (want %d)", c.what, n, c.min)
		}
	}
}

// Ghostbusters: no grey cape any more, a brown proton pack on the back with its cable, a grey
// belt and black boots; their specs (and so names, hair and stats) are the same as before.
func TestNjgGhostbusters(t *testing.T) {
	brown := func(c color.RGBA) bool { return c.R > c.B+0x20 && c.G > c.B+0x10 && c.R < 0xa0 && c.R > 0x50 }
	for _, slug := range njgAllSlugs[2:] {
		sp := charimpASpec(t, slug)
		if sp.Cape != hx(0x4a4a52) || sp.Shoes != hx(0x3a2a1a) || sp.Top != TopLong {
			t.Errorf("%s: the spec changed: %+v", slug, sp)
		}
		bk := genChar(&sp, vBack, poseFor(AnimIdle, 0))
		if n := njgCount(bk, 10, 30, brown); n < 25 {
			t.Errorf("%s: proton pack on the back: %d brown pixels", slug, n)
		}
		fr := genChar(&sp, vFront, poseFor(AnimIdle, 0))
		if n := njgCount(fr, 38, 44, func(c color.RGBA) bool { return c.R < 0x20 && c.G < 0x20 && c.B < 0x24 }); n < 4 {
			t.Errorf("%s: black boots: %d pixels", slug, n)
		}
		if n := njgCount(fr, 20, 34, func(c color.RGBA) bool { return c == gbBelt || c == gbBeltL }); n < 3 {
			t.Errorf("%s: grey belt: %d pixels", slug, n)
		}
		if n := njgCount(fr, 0, 44, func(c color.RGBA) bool { return c == gbLogoR }); n < 1 {
			t.Errorf("%s: no logo patch", slug)
		}
		sd := genChar(&sp, vSide, poseFor(AnimIdle, 0))
		if n := njgCount(sd, 0, 44, func(c color.RGBA) bool { return c == hx(0x4a4a52) }); n > 0 {
			t.Errorf("%s: the grey cape still shows (%d pixels)", slug, n)
		}
		if n := njgCount(sd, 10, 30, brown); n < 20 {
			t.Errorf("%s: pack in profile: %d brown pixels", slug, n)
		}
	}
}
