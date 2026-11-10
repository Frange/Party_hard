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

// ktmSlugs: the characters with their own layers in ktm_art.go (the killer is in killerPresets).
var ktmSlugs = []string{"kratos", "solid_snake", "big_boss", termSlug}

func ktmSpec(t *testing.T, slug string) CharSpec {
	t.Helper()
	if slug == termSlug {
		return killerSpecOf(slug)
	}
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

// KTM_SHEET=dir go test -run TestKtmSheet writes an enlarged sheet: per character, idle / walk / stab in the 3 views.
func TestKtmSheet(t *testing.T) {
	dir := os.Getenv("KTM_SHEET")
	if dir == "" {
		t.Skip("set KTM_SHEET to a directory")
	}
	os.MkdirAll(dir, 0o755)
	z, slugs := 5, ktmSlugs
	if os.Getenv("KTM_ONLY") != "" {
		z, slugs = 10, strings.Split(os.Getenv("KTM_ONLY"), ",")
	}
	poses := []Pose{poseFor(AnimIdle, 0), poseFor(AnimWalk, 1), poseFor(AnimStab, 1)}
	cellW, cellH := 3*charW*z+12, charH*z+6
	big := image.NewRGBA(image.Rect(0, 0, len(poses)*cellW, len(slugs)*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x6a, 0x8a, 0x5a, 255
	}
	for si, slug := range slugs {
		sp := ktmSpec(t, slug)
		for pi, p := range poses {
			for vi, vw := range []int{vFront, vSide, vBack} {
				b := renderChar(&sp, vw, p)
				for y := 0; y < b.h; y++ {
					for x := 0; x < b.w; x++ {
						i := (y*b.w + x) * 4
						if b.p[i+3] == 0 {
							continue
						}
						c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
						for dy := 0; dy < z; dy++ {
							for dx := 0; dx < z; dx++ {
								big.SetRGBA(pi*cellW+(vi*charW+x)*z+dx, si*cellH+y*z+dy, c)
							}
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("ktm%s.png", os.Getenv("KTM_TAG"))))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}

// They draw in the three views and the usual animations inside the sprite box (the outer ring stays
// as empty as the same body without the layers), they keep their body, and the layers really show.
func TestKtmDrawsInsideTheSprite(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimDance7, AnimPanic, AnimHop}
	for _, slug := range ktmSlugs {
		sp := ktmSpec(t, slug)
		twin := sp
		twin.Name = "twin"
		for _, vw := range []int{vFront, vSide, vBack} {
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					n, edge := charimpCount(renderChar(&sp, vw, ownPose(a, f)))
					tn, tEdge := charimpCount(renderChar(&twin, vw, ownPose(a, f)))
					if edge > tEdge {
						t.Errorf("%s view %d anim %d frame %d: %d pixels on the edge (without layers: %d)", slug, vw, a, f, edge, tEdge)
					}
					if n < 60 || n < tn*9/10 {
						t.Errorf("%s view %d anim %d frame %d: only %d pixels (without layers: %d)", slug, vw, a, f, n, tn)
					}
				}
			}
			if string(renderChar(&sp, vw, ownPose(AnimIdle, 0)).p) == string(renderChar(&twin, vw, ownPose(AnimIdle, 0)).p) {
				t.Errorf("%s view %d: the own layers are not drawn", slug, vw)
			}
		}
	}
}

// Walking changes the picture from frame to frame in every view.
func TestKtmWalkFramesChange(t *testing.T) {
	for _, slug := range ktmSlugs {
		sp := ktmSpec(t, slug)
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

// The look follows the references: key colours are really painted (Kratos's red stripe, axe blade
// and gold; Snake's blue bandana; the endoskeleton's red eyes and chrome).
func TestKtmKeyColours(t *testing.T) {
	has := func(b *pixBuf, c color.RGBA) bool {
		for i := 0; i+3 < len(b.p); i += 4 {
			if b.p[i] == c.R && b.p[i+1] == c.G && b.p[i+2] == c.B && b.p[i+3] != 0 {
				return true
			}
		}
		return false
	}
	want := map[string][]color.RGBA{
		"kratos":      {ktRed, ktBlade, ktGold, ktSkirt},
		"solid_snake": {snBand, snGlove},
		termSlug:      {enEye, enChrome, enTeeth},
	}
	if presetByName("big_boss") != nil {
		want["big_boss"] = []color.RGBA{bbBand, bbPatchE}
	}
	for slug, cols := range want {
		sp := ktmSpec(t, slug)
		b := renderChar(&sp, vFront, ownPose(AnimIdle, 0))
		for _, c := range cols {
			if !has(b, c) {
				t.Errorf("%s: colour %v missing from the front view", slug, c)
			}
		}
	}
}

// KTM_EXPORT=assets-root go test -run TestKtmExport writes the pictures of the four characters from the code
// (same method as TestCharimpReexport: an existing meta.json is kept, only the held object follows the code;
// a new folder gets the meta of exportMissingCharacters). Only these four folders are touched.
func TestKtmExport(t *testing.T) {
	root := os.Getenv("KTM_EXPORT")
	if root == "" {
		t.Skip("set KTM_EXPORT to an assets folder")
	}
	want := map[string]bool{}
	for _, s := range ktmSlugs {
		want[s] = true
	}
	n := 0
	for _, b := range builtinChars() {
		if !want[b.name] {
			continue
		}
		ca := charimpFresh(b)
		if old, err := loadCharAsset(filepath.Join(spritesDir(root, b.cat), b.name)); err == nil {
			if old.Meta.Edited {
				t.Logf("%s: edited by hand, skipped", b.name)
				continue
			}
			carry := ca.Meta.Carry
			ca.Meta = old.Meta
			ca.Meta.Carry = carry
		} else {
			spec := b.spec
			ca.Meta.ShopOnly = true
			ca.Meta.DisplayName = nameFromSpec(&spec)
			ca.Meta.Gender = "m"
		}
		if err := saveCharAsset(root, ca); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s -> %s", b.name, b.cat)
		n++
	}
	if n != len(ktmSlugs) {
		t.Errorf("exported %d of %d", n, len(ktmSlugs))
	}
}

// The exported pictures in assets/ are the ones the code draws, and the rig built from them walks in 3 views.
func TestKtmExportedPictures(t *testing.T) {
	want := map[string]bool{}
	for _, s := range ktmSlugs {
		want[s] = true
	}
	seen := 0
	for _, b := range builtinChars() {
		if !want[b.name] {
			continue
		}
		dir := filepath.Join(spritesDir("assets", b.cat), b.name)
		ca, err := loadCharAsset(dir)
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		seen++
		if !charimpSame(ca, charimpFresh(b)) {
			t.Errorf("%s: the exported pictures are not the ones the code draws", b.name)
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
	if seen != len(ktmSlugs) {
		t.Errorf("checked %d of %d", seen, len(ktmSlugs))
	}
}
