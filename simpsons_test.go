package main

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

var simpSlugs = []string{"homer", "marge", "bart", "lisa", "maggie"}

func simpSpec(t *testing.T, slug string) CharSpec {
	t.Helper()
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	sp := f(rand.New(rand.NewSource(1)))
	if sp.Name != slug || !isSimpson(&sp) {
		t.Fatalf("%s: preset name %q", slug, sp.Name)
	}
	return sp
}

// Every member draws in the three views and every animation (gaits included), inside the
// sprite box; Marge is the tallest, the kids keep their kid size and Maggie is the smallest.
func TestSimpsonsDrawInBox(t *testing.T) {
	top := map[string]int{}
	for _, slug := range simpSlugs {
		sp := simpSpec(t, slug)
		for vw := 0; vw < 3; vw++ {
			for a := 0; a < animGaitEnd; a++ {
				for f := 0; f < animFrameCount(a); f++ {
					pp := poseFor(a, f)
					pp.Anim, pp.F = a, f
					pp.Hold = HoldNone
					x0, y0, x1, y1 := bbox(genChar(&sp, vw, pp))
					if x1 < 0 {
						t.Fatalf("%s: empty frame vw=%d anim=%d f=%d", slug, vw, a, f)
					}
					if x0 < 1 || y0 < 1 || x1 > charW-2 || y1 > charH-1 {
						t.Errorf("%s vw=%d anim=%d f=%d outside the box: %d,%d-%d,%d", slug, vw, a, f, x0, y0, x1, y1)
					}
				}
			}
			pp := poseFor(AnimIdle, 0)
			pp.Anim = AnimIdle
			b := genChar(&sp, vw, pp)
			_, y0, _, y1 := bbox(b)
			if vw == vFront {
				top[slug] = y0
				if y1 < 41 || y1 > 42 {
					t.Errorf("%s: feet at row %d", slug, y1)
				}
			}
			// every body part is there (the game animates the exported pictures by parts)
			trackParts = true
			b = genChar(&sp, vw, pp)
			trackParts = false
			for id := uint8(1); id <= 6; id++ {
				found := false
				for i, v := range b.pid {
					if v == id && b.p[i*4+3] > 0 {
						found = true
						break
					}
				}
				if !found && !(vw == vSide && (id == 3 || id == 5)) {
					t.Errorf("%s vw=%d: part %d missing", slug, vw, id)
				}
			}
		}
	}
	if !(top["marge"] < top["homer"] && top["homer"] < top["bart"] && top["bart"] <= top["lisa"]+1 && top["lisa"] < top["maggie"]) {
		t.Errorf("heights (top rows) wrong: %v", top)
	}
	if top["maggie"]-top["homer"] < 14 {
		t.Errorf("Maggie is not a baby: %v", top)
	}
}

// Walking, running and dancing change the frames in every view, and the yellow skin is there.
func TestSimpsonsAnimate(t *testing.T) {
	for _, slug := range simpSlugs {
		sp := simpSpec(t, slug)
		for vw := 0; vw < 3; vw++ {
			for _, a := range []int{AnimWalk, AnimRun, AnimDance0, AnimStagger} {
				p0, p1 := poseFor(a, 0), poseFor(a, 1)
				p0.Anim, p1.Anim, p1.F = a, a, 1
				if string(genChar(&sp, vw, p0).p) == string(genChar(&sp, vw, p1).p) {
					t.Errorf("%s vw=%d anim=%d: frames 0 and 1 are the same", slug, vw, a)
				}
			}
			b := genChar(&sp, vw, poseFor(AnimIdle, 0))
			yellow := 0
			for i := 0; i < len(b.p); i += 4 {
				if b.p[i+3] > 0 && b.p[i] == spSkin.R && b.p[i+1] == spSkin.G && b.p[i+2] == spSkin.B {
					yellow++
				}
			}
			if yellow < 12 {
				t.Errorf("%s vw=%d: only %d yellow pixels", slug, vw, yellow)
			}
		}
	}
	if gaitOf("homer", true, 0) != "stagger" {
		t.Errorf("Homer's gait: %q", gaitOf("homer", true, 0))
	}
}

// Maggie is a new famous guest of the family group with her texts; the others keep their names.
func TestSimpsonsCast(t *testing.T) {
	fn := famousNames()
	for _, slug := range simpSlugs {
		if !fn[slug] {
			t.Errorf("%s is not a famous guest", slug)
		}
	}
	found := false
	for _, g := range starGroups {
		if len(g) > 0 && g[0] == "homer" {
			found = len(g) == 5 && g[4] == "maggie"
		}
	}
	if !found {
		t.Errorf("Maggie is not in the family group")
	}
	for slug, want := range map[string]string{"homer": "Homer", "marge": "Marge", "bart": "Bart", "lisa": "Lisa", "maggie": "Maggie"} {
		sp := simpSpec(t, slug)
		if n := nameFromSpec(&sp); n != want {
			t.Errorf("%s: name %q", slug, n)
		}
	}
	if d, ok := charDescs["maggie"]; !ok || d[0] == "" || d[1] == "" {
		t.Errorf("Maggie has no description ES/EN")
	}
}

// PH_SIMP_SHEET=/path.png dumps a x5 sheet: one block per member, rows = front/side/back.
func TestSimpsonsSheet(t *testing.T) {
	out := os.Getenv("PH_SIMP_SHEET")
	if out == "" {
		t.Skip("set PH_SIMP_SHEET to dump the sheet")
	}
	type pf struct{ a, f int }
	cols := []pf{{AnimIdle, 0}, {AnimWalk, 0}, {AnimWalk, 2}, {AnimRun, 1}, {AnimDance0, 1}, {AnimStagger, 2}}
	k := 5
	if os.Getenv("PH_SIMP_IDLE") != "" { // only the standing pose, bigger
		cols, k = cols[:1], 10
	}
	bw := len(cols) * charW
	img := image.NewRGBA(image.Rect(0, 0, len(simpSlugs)*bw*k, 3*charH*k))
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			c := color.RGBA{200, 200, 205, 255}
			if (x/(bw*k))%2 == 1 {
				c = color.RGBA{180, 186, 196, 255}
			}
			img.Set(x, y, c)
		}
	}
	for si, slug := range simpSlugs {
		sp := simpSpec(t, slug)
		for vw := 0; vw < 3; vw++ {
			for ci, c := range cols {
				pp := poseFor(c.a, c.f)
				pp.Anim, pp.F = c.a, c.f
				b := renderChar(&sp, vw, pp)
				for y := 0; y < b.h; y++ {
					for x := 0; x < b.w; x++ {
						i := (y*b.w + x) * 4
						if b.p[i+3] == 0 {
							continue
						}
						col := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
						for dy := 0; dy < k; dy++ {
							for dx := 0; dx < k; dx++ {
								img.Set((si*bw+ci*charW+x)*k+dx, (vw*charH+y)*k+dy, col)
							}
						}
					}
				}
			}
		}
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, img)
}

// SIMP_EXPORT=assets-root go test -run TestSimpsonsExport writes the pictures of the family from
// the code (same method as TestCharimpReexport; Maggie's folder is created). meta.json is kept.
func TestSimpsonsExport(t *testing.T) {
	root := os.Getenv("SIMP_EXPORT")
	if root == "" {
		t.Skip("set SIMP_EXPORT to an assets folder")
	}
	want := map[string]bool{}
	for _, s := range simpSlugs {
		want[s] = true
	}
	n := 0
	for _, b := range builtinChars() {
		if b.cat != "special_people" || !want[b.name] {
			continue
		}
		ca := charimpFresh(b)
		if old, err := loadCharAsset(filepath.Join(spritesDir(root, b.cat), b.name)); err == nil {
			if old.Meta.Edited {
				t.Logf("%s: edited, left alone", b.name)
				continue
			}
			carry := ca.Meta.Carry
			ca.Meta = old.Meta
			ca.Meta.Carry = carry
		}
		if err := saveCharAsset(root, ca); err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != len(simpSlugs) {
		t.Errorf("exported %d of %d", n, len(simpSlugs))
	}
}
