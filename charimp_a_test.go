package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// charimpASlugs: the famous guests (names A-H) redrawn to look more like the characters they parody.
var charimpASlugs = []string{
	"batman", "deadpool", "capitan_america", "flash", "catwoman", "hulk", "aquaman", "he_man",
	"goku", "gohan", "bulma", "bowser", "donkey_kong", "crash", "creeper", "doraemon",
	"boba_fett", "han_solo", "chewbacca", "buzz", "gru", "gandalf", "dumbledore", "harry_potter",
	"hermione", "frodo", "elsa", "blancanieves", "bart", "homer", "bender", "beetlejuice",
	"heisenberg", "arlequina", "ash_pokemon", "et", "alien", "dracula", "burns", "flanders",
}

func charimpASpec(t *testing.T, slug string) CharSpec {
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

// CHARIMPA_SHEET=dir [CHARIMPA_TAG=antes] [CHARIMPA_ONLY=a,b] go test -run TestCharimpASheet
// writes enlarged sheets (front, side, back; idle and a walk frame).
func TestCharimpASheet(t *testing.T) {
	dir := os.Getenv("CHARIMPA_SHEET")
	if dir == "" {
		t.Skip("set CHARIMPA_SHEET to a directory to dump the sheets")
	}
	slugs := charimpASlugs
	if o := os.Getenv("CHARIMPA_ONLY"); o != "" {
		slugs = strings.Split(o, ",")
	}
	tag := os.Getenv("CHARIMPA_TAG")
	os.MkdirAll(dir, 0o755)
	const z, cols, per = 4, 4, 12
	cellW, cellH := 4*charW*z+8, charH*z+4
	for s := 0; s*per < len(slugs); s++ {
		group := slugs[s*per : minI(len(slugs), (s+1)*per)]
		rows := (len(group) + cols - 1) / cols
		big := image.NewRGBA(image.Rect(0, 0, cols*cellW, rows*cellH))
		for i := 0; i < len(big.Pix); i += 4 {
			big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x70, 0x70, 0x78, 255
		}
		for gi, slug := range group {
			sp := charimpASpec(t, slug)
			ox, oy := (gi%cols)*cellW, (gi/cols)*cellH
			for vi, vw := range []int{vFront, vSide, vBack, vFront} {
				pp := poseFor(AnimIdle, 0)
				if vi == 3 {
					pp = poseFor(AnimWalk, 1)
					pp.Anim, pp.F = AnimWalk, 1
				}
				b := renderChar(&sp, vw, pp)
				for y := 0; y < b.h; y++ {
					for x := 0; x < b.w; x++ {
						i := (y*b.w + x) * 4
						if b.p[i+3] == 0 {
							continue
						}
						c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
						for dy := 0; dy < z; dy++ {
							for dx := 0; dx < z; dx++ {
								big.SetRGBA(ox+(vi*charW+x)*z+dx, oy+y*z+dy, c)
							}
						}
					}
				}
			}
		}
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s%d.png", tag, s)))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, big)
		f.Close()
	}
}

// charimpADone: the ones redrawn in charimp_a.go (A-H, not the already renewed ones).
var charimpADone = []string{
	"batman", "deadpool", "capitan_america", "flash", "catwoman", "aquaman", "goku", "bowser",
	"donkey_kong", "creeper", "doraemon", "boba_fett", "han_solo", "buzz", "gru", "harry_potter",
	"blancanieves", "beetlejuice", "he_man", "bender",
}

func charimpAPix(b *pixBuf) (n, edge int) {
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] == 0 {
				continue
			}
			n++
			if x == 0 || y == 0 || x == b.w-1 || y == b.h-1 {
				edge++
			}
		}
	}
	return
}

// Every redrawn guest draws in the three views inside its box (no more pixels on the outer ring
// than the same body without the details), its details really show, and walking changes the frames.
func TestCharimpADrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic}
	for _, slug := range charimpADone {
		sp := charimpASpec(t, slug)
		if caKind(&sp) == caNone {
			t.Fatalf("%s: no details registered", slug)
		}
		twin := sp
		twin.Name = "twin"
		for vw := 0; vw < 3; vw++ {
			differs := false
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					pp := poseFor(a, f)
					pp.Anim, pp.F = a, f
					b := renderChar(&sp, vw, pp)
					tb := renderChar(&twin, vw, pp)
					n, edge := charimpAPix(b)
					tn, tEdge := charimpAPix(tb)
					if edge > tEdge {
						t.Errorf("%s vw %d anim %d f %d: %d pixels on the edge (plain body %d)", slug, vw, a, f, edge, tEdge)
					}
					if n < 60 || n < tn*9/10 {
						t.Errorf("%s vw %d anim %d f %d: only %d pixels (plain body %d)", slug, vw, a, f, n, tn)
					}
					if string(b.p) != string(tb.p) {
						differs = true
					}
				}
			}
			if !differs && vw != vBack { // some keep their back plain (a cape covers it)
				t.Errorf("%s vw %d: the details never show", slug, vw)
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

func charimpAWalkFrame(sp *CharSpec, vw, f int) []uint8 {
	pp := poseFor(AnimWalk, f)
	pp.Anim, pp.F = AnimWalk, f
	return renderChar(sp, vw, pp).p
}

// The redraw keeps every name: same famous characters, same count (318 before the change).
func TestCharimpAKeepsTheCast(t *testing.T) {
	fn := famousNames()
	for _, slug := range charimpADone {
		if !fn[slug] {
			t.Errorf("%s is no longer a famous guest", slug)
		}
	}
	if len(fn) != 342 { // 318 + Maggie (simpsons_art.go) + Cell (dragonball_art.go) + Ron (dnh_art.go) + 5 Planetarios (planet_art.go) + Big Boss (ktm_art.go) + 2 Lost Prophets (bandsb_art.go) + 4 footballers (futbol_art.go) + 9 Fall Guys beans (fallguys.go)
		t.Errorf("famous characters: %d, want 342", len(fn))
	}
}

// CHARIMPA_EXPORT=assets-root go test -run TestCharimpAExport redraws the exported pictures of the
// redrawn guests (the game draws the famous guests of special_people from these files). Edited ones
// are left alone; name, gender and shop flags are kept.
func TestCharimpAExport(t *testing.T) {
	root := os.Getenv("CHARIMPA_EXPORT")
	if root == "" {
		t.Skip("set CHARIMPA_EXPORT to an assets folder")
	}
	want := map[string]bool{}
	for _, s := range charimpADone {
		want[s] = true
	}
	n := 0
	for _, b := range builtinChars() {
		if b.cat != "special_people" || !want[b.name] {
			continue
		}
		old, err := loadCharAsset(filepath.Join(spritesDir(root, b.cat), b.name))
		if err != nil || old.Meta.Edited {
			t.Logf("%s: skipped (%v)", b.name, err)
			continue
		}
		spec := b.spec
		spec.Ride = 0
		ca := charAssetFromSpec(spec, b.name, b.cat, b.role)
		ca.Meta.DisplayName, ca.Meta.Gender, ca.Meta.ShopOnly = old.Meta.DisplayName, old.Meta.Gender, old.Meta.ShopOnly
		if err := saveCharAsset(root, ca); err != nil {
			t.Fatal(err)
		}
		n++
	}
	t.Logf("redrawn: %d", n)
	if n != len(charimpADone) {
		t.Errorf("redrawn %d of %d", n, len(charimpADone))
	}
}

// The game draws the famous guests from assets/sprites/special_people: their pictures must be the
// redrawn ones, and the animated rig built from them must walk in the three views.
func TestCharimpAExportedPictures(t *testing.T) {
	want := map[string]bool{}
	for _, s := range charimpADone {
		want[s] = true
	}
	seen := 0
	for _, b := range builtinChars() {
		if b.cat != "special_people" || !want[b.name] {
			continue
		}
		ca, err := loadCharAsset(filepath.Join(spritesDir("assets", b.cat), b.name))
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		if ca.Meta.Edited {
			continue
		}
		seen++
		spec := b.spec
		spec.Ride = 0
		fresh := charAssetFromSpec(spec, b.name, b.cat, b.role)
		for i := range ca.Views {
			if string(ca.Views[i].p) != string(fresh.Views[i].p) {
				t.Errorf("%s view %d: the exported picture is not the redrawn one", b.name, i)
			}
		}
		rig := buildRig(ca)
		for vw := 0; vw < 3; vw++ {
			f0 := rig.render(vw, poseFor(AnimWalk, 0))
			if n, _ := charimpAPix(f0); n < 60 {
				t.Errorf("%s rig vw %d: only %d pixels", b.name, vw, n)
			}
			if string(rig.render(vw, poseFor(AnimWalk, 1)).p) == string(f0.p) {
				t.Errorf("%s rig vw %d: walking does not change the frames", b.name, vw)
			}
		}
	}
	if seen != len(charimpADone) {
		t.Errorf("checked %d of %d", seen, len(charimpADone))
	}
}

// CHARIMPA_OLD=dir-with-old-character-folders CHARIMPA_SHEET=out go test -run TestCharimpABeforeAfter
// writes enlarged before -> after sheets (front, side, back of the exported pictures).
func TestCharimpABeforeAfter(t *testing.T) {
	oldDir, out := os.Getenv("CHARIMPA_OLD"), os.Getenv("CHARIMPA_SHEET")
	if oldDir == "" || out == "" {
		t.Skip("set CHARIMPA_OLD and CHARIMPA_SHEET")
	}
	const z, per, cols = 3, 10, 2
	cellW, cellH := 7*charW*z+16, charH*z+6
	for s := 0; s*per < len(charimpADone); s++ {
		group := charimpADone[s*per : minI(len(charimpADone), (s+1)*per)]
		rows := (len(group) + cols - 1) / cols
		big := image.NewRGBA(image.Rect(0, 0, cols*cellW, rows*cellH))
		for i := 0; i < len(big.Pix); i += 4 {
			big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x70, 0x70, 0x78, 255
		}
		put := func(b *pixBuf, ox, oy int) {
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							big.SetRGBA(ox+x*z+dx, oy+y*z+dy, c)
						}
					}
				}
			}
		}
		for gi, slug := range group {
			ox, oy := (gi%cols)*cellW, (gi/cols)*cellH
			old, err := loadCharAsset(filepath.Join(oldDir, slug))
			if err != nil {
				t.Fatal(err)
			}
			now, err := loadCharAsset(filepath.Join(spritesDir("assets", "special_people"), slug))
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 3; i++ {
				put(old.Views[i].outlinedSel(0.42), ox+i*charW*z, oy)
				put(now.Views[i].outlinedSel(0.42), ox+(4+i)*charW*z, oy)
			}
			// an arrow between before and after
			ay := oy + charH*z/2
			for x := ox + 3*charW*z + 10; x < ox+4*charW*z-10; x++ {
				big.SetRGBA(x, ay, color.RGBA{240, 240, 240, 255})
			}
			for k := 0; k < 6; k++ {
				big.SetRGBA(ox+4*charW*z-10-k, ay-k, color.RGBA{240, 240, 240, 255})
				big.SetRGBA(ox+4*charW*z-10-k, ay+k, color.RGBA{240, 240, 240, 255})
			}
		}
		f, err := os.Create(filepath.Join(out, fmt.Sprintf("antes_despues_%d.png", s)))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, big)
		f.Close()
	}
}
