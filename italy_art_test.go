package main

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// The monuments of ITALIA and PARIS read clearly: the leaning tower is tall, leans, has dark openings and stays clear
// of the party title and the intro plate at the start; the main arch of the amphitheatre has solid pillars; the
// Eiffel Tower keeps its top clear of the plate too.

// titlePlateBottom: the lowest row of the intro plate under the party title (draw.go: y 62, 17 high) plus the margin.
const titlePlateBottom = 62 + 17 + 8

func spriteTopRow(b *pixBuf) int {
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] > 0 {
				return y
			}
		}
	}
	return b.h
}

func propTopY(p PropInst) float64 {
	k := propKinds[p.Kind]
	s := renderPropSprite(p.Kind, k, p.Var)
	return p.Y - float64(k.H) - 1 + float64(spriteTopRow(s))
}

func TestPisaBigLeaningAndClearOfTitle(t *testing.T) {
	for _, seed := range []int64{campaignLevels[italyIdx].seed, 1, 2} {
		d := genNamed("italy", seed, "")
		var tw *PropInst
		lawn := false
		for i, p := range d.Props {
			if p.Kind == "pisa" {
				tw = &d.Props[i]
			}
			lawn = lawn || p.Kind == "pisalawn"
		}
		if tw == nil || !lawn {
			t.Fatalf("seed %d: tower %v, garden %v", seed, tw != nil, lawn)
		}
		if top := propTopY(*tw); top < titlePlateBottom {
			t.Errorf("seed %d: the top of the tower (y %.0f) is under the title / intro plate (needs >= %d)", seed, top, titlePlateBottom)
		}
		for _, f := range d.Floors { // the garden is drawn by its own piece: no hard rectangle of grass
			if f.Kind == "grass" {
				t.Errorf("seed %d: a rectangle of grass is back: %v", seed, f.R)
			}
		}
	}
	k := propKinds["pisa"]
	b := newPix(k.W+2, k.H+2)
	b.ox, b.oy = 1, 1
	drawPisa(b, 0)
	// tall: about 1.15x the old 207 rows of stone
	if h := pisaBaseY - (spriteTopRow(b) - 1); h < 236 {
		t.Errorf("the tower is only %d px tall", h)
	}
	// it leans: the centre of the top storeys is well to the right of the base
	mid := func(y int) float64 {
		l, r := -1, -1
		for x := 0; x < k.W; x++ {
			if c := b.get(x, y); c.A == 255 {
				if l < 0 {
					l = x
				}
				r = x
			}
		}
		return float64(l+r) / 2
	}
	if lean := mid(pisaBaseY-200) - mid(pisaBaseY-20); lean < 15 {
		t.Errorf("the tower barely leans: %.1f px", lean)
	}
	// contrast: bright marble, blue-grey shadow side and dark openings, all opaque
	bright, dark, cool := 0, 0, 0
	for y := pisaBaseY - 190; y < pisaBaseY-50; y++ {
		for x := 0; x < k.W; x++ {
			c := b.get(x, y)
			if c.A == 0 {
				continue
			}
			if c.A != 255 {
				t.Fatalf("semi-transparent pixel in the tower at %d,%d", x, y)
			}
			s := int(c.R) + int(c.G) + int(c.B)
			switch {
			case s > 3*215:
				bright++
			case s < 3*70:
				dark++
			}
			if int(c.B) > int(c.R)+12 {
				cool++
			}
		}
	}
	if bright < 1200 || dark < 500 || cool < 600 {
		t.Errorf("flat tower: %d bright, %d dark, %d bluish-shadow pixels", bright, dark, cool)
	}
}

// the main arch of the amphitheatre: opaque pillars on both sides, open (see-through) only in the passage
func TestColosGatePillarsOpaque(t *testing.T) {
	k := propKinds[colosKind(colosGate)]
	s := renderPropSprite(colosKind(colosGate), k, 0)
	ground := s.h - 6
	opaqueCols, openCols := 0, 0
	for x := 0; x < s.w; x++ {
		col := 0
		for y := ground - 20; y < ground; y++ {
			if s.get(x, y).A == 255 {
				col++
			}
		}
		switch {
		case col == 20:
			opaqueCols++
		case col == 0:
			openCols++
		}
	}
	if opaqueCols < 6 || openCols < 14 {
		t.Errorf("the gate: %d solid pillar columns, %d open columns", opaqueCols, openCols)
	}
	for _, kk := range []int{5, 7, 10, 14} { // standing walls: no half-transparent pixels
		s := renderPropSprite(colosKind(kk), propKinds[colosKind(kk)], 0)
		for i := 3; i < len(s.p); i += 4 {
			if s.p[i] != 0 && s.p[i] != 255 {
				t.Fatalf("segment %d has a half-transparent pixel", kk)
			}
		}
	}
}

func TestEiffelTopClearOfTitle(t *testing.T) {
	d := genPlaza(newRand(campaignLevels[campaignIdx("eiffel")].seed), true)
	for _, p := range d.Props {
		if p.Kind == "eiffel" {
			if top := propTopY(p); top < titlePlateBottom {
				t.Errorf("the top of the Eiffel Tower (y %.0f) is under the title / intro plate (needs >= %d)", top, titlePlateBottom)
			}
			return
		}
	}
	t.Fatal("no eiffel")
}

func TestLandmarksStaySolidBehindGuests(t *testing.T) {
	for _, n := range []string{"eiffel", "pisa", "colos06", "colos12"} {
		if a, ok := landmarkFadeAmp(n); !ok || a > 0.45 {
			t.Errorf("%s: fade %.2f %v", n, a, ok)
		}
	}
	for _, n := range []string{"colosfloor", "pisalawn", "tree", "fountain"} {
		if _, ok := landmarkFadeAmp(n); ok {
			t.Errorf("%s should fade like any tall prop", n)
		}
	}
}

// ITALY_SHEET=dir writes the tower, its garden and the whole amphitheatre at 3x to look at them.
func TestItalySheet(t *testing.T) {
	dir := os.Getenv("ITALY_SHEET")
	if dir == "" {
		t.Skip("set ITALY_SHEET")
	}
	save := func(name string, s *pixBuf, z int) {
		big := newPix(s.w*z, s.h*z)
		for y := 0; y < big.h; y++ {
			for x := 0; x < big.w; x++ {
				c := s.get(x/z, y/z)
				o := (y*big.w + x) * 4
				bg := [3]byte{96, 92, 88}
				a := float64(c.A) / 255
				for i := 0; i < 3; i++ {
					big.p[o+i] = byte(math.Round(float64([]byte{c.R, c.G, c.B}[i])*a + float64(bg[i])*(1-a)))
				}
				big.p[o+3] = 255
			}
		}
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, big.toRGBA())
	}
	for _, n := range []string{"pisa", "pisalawn"} {
		save(n+".png", stylizeSprite(renderPropSprite(n, propKinds[n], 0)), 3)
	}
	// the amphitheatre assembled like in the level
	W, H := 400, 260
	cv := newPix(W, H)
	ox, oy := 200.0, 170.0
	blit := func(kind string, x, y float64) {
		k := propKinds[kind]
		s := stylizeSprite(renderPropSprite(kind, k, 0))
		x0, y0 := int(x)-k.W/2-1, int(y)-k.H-1
		for py := 0; py < s.h; py++ {
			for px := 0; px < s.w; px++ {
				c := s.get(px, py)
				if c.A > 0 {
					cv.set(x0+px, y0+py, c)
				}
			}
		}
	}
	fl := propKinds["colosfloor"]
	blit("colosfloor", ox, oy+float64(fl.H)/2)
	type pp struct {
		k string
		y float64
	}
	var ps []pp
	for _, p := range colosParts {
		ps = append(ps, pp{p.Kind, p.D.Y})
	}
	for i := 1; i < len(ps); i++ {
		for j := i; j > 0 && ps[j].y < ps[j-1].y; j-- {
			ps[j], ps[j-1] = ps[j-1], ps[j]
		}
	}
	for _, p := range ps {
		for _, q := range colosParts {
			if q.Kind == p.k {
				blit(q.Kind, ox+q.D.X, oy+q.D.Y)
			}
		}
	}
	save("colos.png", cv, 3)
	save("eiffel.png", stylizeSprite(renderPropSprite("eiffel", propKinds["eiffel"], 0)), 3)
}

// ITALIA has its own little picture in the list of parties (a leaning tower and an arch), not a flat swatch
func TestItalyListIconIsAPicture(t *testing.T) {
	if levelIconArt["italy"] == nil || campaignLevels[italyIdx].icon != "italy" {
		t.Fatal("ITALIA has no icon art")
	}
	towerMid := func(row string) float64 {
		n, s := 0, 0
		for i := 0; i < 8 && i < len(row); i++ { // the tower is on the left part
			if row[i] == 'W' || row[i] == 'w' || row[i] == 'd' {
				n++
				s += i
			}
		}
		if n == 0 {
			return -1
		}
		return float64(s) / float64(n)
	}
	kinds := map[byte]bool{}
	for r, row := range italyIconRows {
		if len(row) != 12 {
			t.Fatalf("row %d is %d wide", r, len(row))
		}
		for i := 0; i < len(row); i++ {
			if _, ok := italyIconCols[row[i]]; !ok {
				t.Fatalf("row %d: no colour for %q", r, row[i])
			}
			kinds[row[i]] = true
		}
	}
	if !kinds['W'] || !kinds['d'] || !kinds['o'] || !kinds['A'] {
		t.Error("the icon misses the tower or the arches")
	}
	if top, bot := towerMid(italyIconRows[2]), towerMid(italyIconRows[10]); top-bot < 1.5 {
		t.Errorf("the tower of the icon does not lean (%.1f -> %.1f)", bot, top)
	}
}
