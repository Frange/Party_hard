package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// limoOpaqueBox: bounding box of the car itself (the soft shadow is translucent).
func limoOpaqueBox(b *pixBuf) image.Rectangle {
	r := image.Rectangle{Min: image.Pt(b.w, b.h)}
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] == 255 {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return r
}

func lumaAt(b *pixBuf, x, y int) (l float64, r, g, bl int) {
	i := (y*b.w + x) * 4
	r, g, bl = int(b.p[i]), int(b.p[i+1]), int(b.p[i+2])
	return 0.3*float64(r) + 0.59*float64(g) + 0.11*float64(bl), r, g, bl
}

// The limousine is a very long, low car: at least 4.5 times longer than tall,
// both paints, and it sits on its anchor (tyres on v.y, nose a little in front of v.x).
func TestLimoProportion(t *testing.T) {
	for _, silver := range []bool{false, true} {
		b := genLimo(silver)
		box := limoOpaqueBox(b)
		ratio := float64(box.Dx()) / float64(box.Dy())
		if ratio < 4.5 || ratio > 5.3 {
			t.Fatalf("silver=%v: limo is %dx%d (ratio %.2f), want 4.5-5.3", silver, box.Dx(), box.Dy(), ratio)
		}
		if box.Dx() < 130 {
			t.Fatalf("silver=%v: limo only %d px long", silver, box.Dx())
		}
		if box.Max.Y-1 < limoBase-1 || box.Max.Y-1 > limoBase {
			t.Fatalf("silver=%v: tyres end at row %d, anchor row %d", silver, box.Max.Y-1, limoBase)
		}
		if box.Min.X > limoFront {
			t.Fatalf("silver=%v: the nose (%d) is behind the anchor %d", silver, box.Min.X, limoFront)
		}
		// dark outline all round: the first opaque pixel of the middle row is near-black
		if l, _, _, _ := lumaAt(b, box.Min.X, 22); l > 40 {
			t.Fatalf("silver=%v: no dark outline at the nose (luma %.0f)", silver, l)
		}
	}
}

// The side windows are blue and stand out against the paint, on navy and on silver.
func TestLimoWindowContrast(t *testing.T) {
	for _, silver := range []bool{false, true} {
		b := genLimo(silver)
		for _, wx := range []int{46, 68, 90, 110} { // one pixel inside each side window, clear of the glint
			wl, r, _, bl := lumaAt(b, wx, 12)
			if bl < r+25 {
				t.Fatalf("silver=%v: window at x=%d is not blue (%d,%d)", silver, wx, r, bl)
			}
			bodyL, _, _, _ := lumaAt(b, wx, 22)
			if d := wl - bodyL; d < 25 && d > -25 {
				t.Fatalf("silver=%v: window x=%d luma %.0f vs body %.0f: too little contrast", silver, wx, wl, bodyL)
			}
		}
		// four door cuts, darker than the body next to them
		for _, sx := range limoSeamX[1:4] {
			sl, _, _, _ := lumaAt(b, sx, 17)
			nl, _, _, _ := lumaAt(b, sx+2, 17)
			if sl >= nl {
				t.Fatalf("silver=%v: door cut at x=%d not darker (%.0f vs %.0f)", silver, sx, sl, nl)
			}
		}
	}
	navy, _, _, _ := lumaAt(genLimo(false), 30, 22)
	silver, _, _, _ := lumaAt(genLimo(true), 30, 22)
	if silver-navy < 60 {
		t.Fatalf("the two paints look alike (navy %.0f, silver %.0f)", navy, silver)
	}
}

// Passengers step out of the side doors, which lie between the two axles.
func TestLimoDoorsBetweenAxles(t *testing.T) {
	for k := 0; k < 6; k++ {
		x := limoDoorWorld(k) + float64(limoFront)
		if x <= float64(limoWheelX[0]+6) || x >= float64(limoWheelX[1]-4) {
			t.Fatalf("door %d at column %.0f is not between the axles %v", k, x, limoWheelX)
		}
	}
	v := &Vehicle{kind: 6, x: 500}
	if c := limoCentre(v); c < 500 || c > 500+float64(limoTail-limoFront) {
		t.Fatalf("limo centre %.0f outside its body", c)
	}
}

// Only limo.go paints a limousine: no other file builds one from tinted car sprites.
func TestLimoSingleDrawFunction(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	call := regexp.MustCompile(`\b(genLimo|genLimoBody|limoImage|drawLimoAt)\(`)
	drawCalls := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "limo.go" {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		s := string(src)
		if call.MatchString(s) {
			t.Errorf("%s paints the limousine itself; use g.drawLimo (limo.go)", f)
		}
		if strings.Contains(s, "func (g *Game) drawLimo(") {
			t.Errorf("%s defines another drawLimo", f)
		}
		drawCalls += strings.Count(s, "g.drawLimo(")
	}
	if drawCalls != 1 {
		t.Fatalf("g.drawLimo is called %d times outside limo.go, want exactly the vehicle loop", drawCalls)
	}
	if dir := os.Getenv("PH_LIMO_PNG"); dir != "" {
		for i, silver := range []bool{false, true} {
			b := genLimo(silver).scaled(6)
			f, err := os.Create(filepath.Join(dir, []string{"limo_navy.png", "limo_silver.png"}[i]))
			if err != nil {
				t.Fatal(err)
			}
			png.Encode(f, b.toRGBA())
			f.Close()
		}
	}
}
