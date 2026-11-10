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

// charimpISlugs: the famous guests (names I-P) redrawn to look more like the characters they parody.
var charimpISlugs = []string{
	"indy", "inspector_gadget", "ironman", "jack_sparrow", "jedi", "jefe_maestro", "john_wick", "johnny_bravo",
	"joker", "jon_nieve", "jugador_456", "kirby", "kirk", "knuckles", "korben", "kratos", "krilin", "la_mascara",
	"pantera_rosa", "lara_croft", "leela", "leeloo", "legolas", "leia", "leon_k", "link", "lisa", "lobezno",
	"luigi", "luke", "maestro_roshi", "mafalda", "mandaloriano", "marge", "mario", "marty_mcfly", "minion",
	"mortadelo", "morty", "mr_bean", "wonder_woman", "naruto", "neo", "negan", "nobita", "once", "pablo_marmol",
	"panoramix", "papa_pitufo", "pedro_picapiedra", "peter_pan", "peter", "piccolo", "pikachu", "pitufo",
	"pitufina", "popeye", "farnsworth", "miercoles",
}

func charimpSpec(t *testing.T, slug string) CharSpec {
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

// CHARIMP_SHEET=dir [CHARIMP_TAG=antes] go test -run TestCharimpISheet writes enlarged sheets (front, side, back).
func TestCharimpISheet(t *testing.T) {
	dir := os.Getenv("CHARIMP_SHEET")
	if dir == "" {
		t.Skip("set CHARIMP_SHEET to a directory to dump the sheets")
	}
	tag := os.Getenv("CHARIMP_TAG")
	slugs := charimpISlugs
	if only := os.Getenv("CHARIMP_ONLY"); only != "" {
		slugs = strings.Split(only, ",")
	}
	os.MkdirAll(dir, 0o755)
	const z, cols, per = 4, 4, 12
	cellW, cellH := 3*charW*z+8, charH*z+4
	for s := 0; s*per < len(slugs); s++ {
		group := slugs[s*per : minI(len(slugs), (s+1)*per)]
		rows := (len(group) + cols - 1) / cols
		big := image.NewRGBA(image.Rect(0, 0, cols*cellW, rows*cellH))
		for i := 0; i < len(big.Pix); i += 4 {
			big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x70, 0x70, 0x78, 255
		}
		for gi, slug := range group {
			sp := charimpSpec(t, slug)
			ox, oy := (gi%cols)*cellW, (gi/cols)*cellH
			for vi, vw := range []int{vFront, vSide, vBack} {
				b := renderChar(&sp, vw, poseFor(AnimIdle, 0))
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

// charimpIImproved: the ones changed in this pass (spec lines in presets2/presets4/cast2/stars + charimp_i.go layers).
var charimpIImproved = []string{
	"jefe_maestro", "mandaloriano", "johnny_bravo", "jugador_456", "kratos", "leon_k", "link", "maestro_roshi",
	"negan", "once", "panoramix", "pitufo", "papa_pitufo", "pitufina", "peter_pan", "miercoles", "lobezno",
	"popeye", "naruto", "krilin", "marty_mcfly", "neo",
}

func charimpCount(b *pixBuf) (n, edge int) {
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

// Every improved guest draws in the three views and in the usual animations inside the sprite box:
// the new details never touch the outer ring (more than the same body without them does).
func TestCharimpIDrawsInsideTheSprite(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimDance7, AnimPanic, AnimHop}
	for _, slug := range charimpIImproved {
		sp := charimpSpec(t, slug)
		twin := sp
		twin.Name = "twin"
		for _, vw := range []int{vFront, vSide, vBack} {
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					n, edge := charimpCount(renderChar(&sp, vw, ownPose(a, f)))
					tn, tEdge := charimpCount(renderChar(&twin, vw, ownPose(a, f)))
					if edge > tEdge {
						t.Errorf("%s view %d anim %d frame %d: %d pixels on the edge (without details: %d)", slug, vw, a, f, edge, tEdge)
					}
					if n < 60 || n < tn*9/10 {
						t.Errorf("%s view %d anim %d frame %d: only %d pixels (without details: %d)", slug, vw, a, f, n, tn)
					}
				}
			}
		}
	}
}

// Walking changes the picture from frame to frame in every view.
func TestCharimpIWalkFramesChange(t *testing.T) {
	for _, slug := range charimpIImproved {
		sp := charimpSpec(t, slug)
		for _, vw := range []int{vFront, vSide, vBack} {
			seen := map[string]bool{}
			for f := 0; f < animFrameCount(AnimWalk); f++ {
				seen[string(renderChar(&sp, vw, ownPose(AnimWalk, f)).p)] = true
			}
			if len(seen) < 3 {
				t.Errorf("%s view %d: only %d different walking frames", slug, vw, len(seen))
			}
		}
	}
}

// The own layers really show: each one differs from the same spec drawn without them, in front and back.
func TestCharimpILayersShow(t *testing.T) {
	for slug := range ciKinds {
		sp := charimpSpec(t, slug)
		twin := sp
		twin.Name = "twin"
		for _, vw := range []int{vFront, vSide} {
			if string(renderChar(&sp, vw, ownPose(AnimIdle, 0)).p) == string(renderChar(&twin, vw, ownPose(AnimIdle, 0)).p) {
				t.Errorf("%s view %d: the own details are not drawn", slug, vw)
			}
		}
	}
}

// Redrawing them changes no name, slug or count of the album.
func TestCharimpICountsUnchanged(t *testing.T) {
	if len(charCatalogSlugs) != 282 {
		t.Errorf("%d catalogue characters, want 282", len(charCatalogSlugs))
	}
	// measured before this pass: 295 character cards, 339 entries in famousPresets
	if n := len(charCards()); n != 319 { // 295 + Maggie (simpsons_art.go) + Cell (dragonball_art.go) + Ron (dnh_art.go) + 5 Planetarios (planet_art.go) + Big Boss (ktm_art.go) + 2 Lost Prophets (bandsb_art.go) + 4 footballers (futbol_art.go) + 9 Fall Guys beans (fallguys.go)
		t.Errorf("%d character cards in the album, want 319", n)
	}
	if n := len(famousPresets); n != 363 { // 339 + Maggie + Cell + Ron + 5 Planetarios + Big Boss + 2 Lost Prophets + 4 footballers + 9 Fall Guys beans
		t.Errorf("%d famous presets, want 363", n)
	}
	for _, slug := range charimpIImproved {
		if presetByName(slug) == nil {
			t.Errorf("%s: preset lost", slug)
		}
		if sp := charimpSpec(t, slug); sp.Name != slug {
			t.Errorf("%s: name became %q", slug, sp.Name)
		}
	}
}
