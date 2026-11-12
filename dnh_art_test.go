package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var dnhSlugs = []string{"doraemon", "nobita", "harry_potter", "hermione", "ron"}

// The five draw with their own art in the three views, inside the box (nothing on the outer
// ring), with every body part present for the rig, and walking changes the frames.
func TestDnhArtDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic, AnimCarry, AnimLaugh}
	for _, slug := range dnhSlugs {
		sp := tiaSpec(t, slug)
		if !isDnh(&sp) {
			t.Fatalf("%s: no own art", slug)
		}
		for vw := 0; vw < 3; vw++ {
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					pp := poseFor(a, f)
					pp.Anim, pp.F = a, f
					b := renderChar(&sp, vw, pp)
					n, edge := charimpAPix(b)
					if edge > 0 {
						t.Errorf("%s vw %d anim %d f %d: %d pixels on the edge", slug, vw, a, f, edge)
					}
					if n < 150 {
						t.Errorf("%s vw %d anim %d f %d: only %d pixels", slug, vw, a, f, n)
					}
				}
			}
			w0 := poseFor(AnimWalk, 0)
			w0.Anim = AnimWalk
			w1 := poseFor(AnimWalk, 2)
			w1.Anim, w1.F = AnimWalk, 2
			if string(genChar(&sp, vw, w0).p) == string(genChar(&sp, vw, w1).p) {
				t.Errorf("%s vw %d: walking does not change the frames", slug, vw)
			}
			trackParts = true
			b := genChar(&sp, vw, poseFor(AnimIdle, 0))
			trackParts = false
			seen := map[uint8]bool{}
			for i, id := range b.pid {
				if b.p[i*4+3] != 0 {
					seen[id] = true
				}
			}
			for id := uint8(PartHead); id <= PartLegR; id++ {
				if !seen[id] && !(vw == vSide && (id == PartArmL || id == PartLegL)) {
					t.Errorf("%s vw %d: part %d missing", slug, vw, id)
				}
			}
		}
	}
}

// Each one looks different from the others and from the generic body of its own spec, and the
// key colours of the reference show (Doraemon blue and red nose, Nobita yellow shirt, the red
// and gold tie on the three pupils, Hermione's brown hair, Ron's orange hair).
func TestDnhArtDistinctAndRecognisable(t *testing.T) {
	idle := poseFor(AnimIdle, 0)
	frames := map[string]string{}
	has := func(b *pixBuf, c color.RGBA) int {
		n := 0
		for i := 0; i+3 < len(b.p); i += 4 {
			if b.p[i+3] != 0 && b.p[i] == c.R && b.p[i+1] == c.G && b.p[i+2] == c.B {
				n++
			}
		}
		return n
	}
	want := map[string][]color.RGBA{
		"doraemon":     {dnBlue, dnRed, dnGold, dnWhite},
		"nobita":       {dnYellow, dnShorts, dnFrame, dnHairK},
		"harry_potter": {dnRobe, dnTieR, dnTieG, dnFrame, dnScar, dnHairK},
		"hermione":     {dnRobe, dnTieR, dnTieG, dnBrown, dnGrey, dnWhite},
		"ron":          {dnRobe, dnTieR, dnTieG, dnOrange, dnFreckle},
	}
	for _, slug := range dnhSlugs {
		sp := tiaSpec(t, slug)
		b := genChar(&sp, vFront, idle)
		frames[slug] = string(b.p)
		for _, c := range want[slug] {
			if has(b, c) == 0 {
				t.Errorf("%s: colour %v missing in the front view", slug, c)
			}
		}
		for other, f := range frames {
			if other != slug && f == frames[slug] {
				t.Errorf("%s and %s draw the same", slug, other)
			}
		}
	}
	// Doraemon smiles wide when he laughs
	sp := tiaSpec(t, "doraemon")
	laugh := poseFor(AnimLaugh, 0)
	laugh.Anim = AnimLaugh
	if has(genChar(&sp, vFront, laugh), dnTongue) == 0 || has(genChar(&sp, vFront, idle), dnTongue) != 0 {
		t.Errorf("Doraemon: the open smile only shows when he laughs")
	}
}

// All are short (child height): the top of the head is below an ordinary guest's, and the feet
// stand on the same rows.
func TestDnhArtShort(t *testing.T) {
	plain := tiaSpec(t, "harry_potter")
	plain.Name, plain.Stature = "twin", 0
	idle := poseFor(AnimIdle, 0)
	ref := tiaTop(renderChar(&plain, vFront, idle))
	for _, slug := range dnhSlugs {
		sp := tiaSpec(t, slug)
		for vw := 0; vw < 3; vw++ {
			b := renderChar(&sp, vw, idle)
			bottom := 0
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					if b.p[(y*b.w+x)*4+3] != 0 {
						bottom = y
					}
				}
			}
			if bottom < 41 || bottom > 43 {
				t.Errorf("%s vw %d: feet end on row %d", slug, vw, bottom)
			}
		}
		if top := tiaTop(renderChar(&sp, vFront, idle)); top < ref+2 {
			t.Errorf("%s not short: top %d, ordinary guest %d", slug, top, ref)
		}
	}
}

// Rom (slug ron) is a new famous guest with his name, description (ES and EN), group and catchphrases.
func TestDnhRonRegistered(t *testing.T) {
	if presetByName("ron") == nil {
		t.Fatal("ron: no preset")
	}
	// The name carries one changed letter (owner's naming policy, 2026-11-11); the slug stays "ron".
	if displayNameOverrides["ron"] != "Rom" {
		t.Errorf("ron: display name %q", displayNameOverrides["ron"])
	}
	d, ok := charDescs["ron"]
	if !ok || d[0] == "" || d[1] == "" || langDataEN[d[0]] != d[1] {
		t.Errorf("ron: description %v", d)
	}
	found := false
	for _, g := range starGroups {
		if len(g) == 3 && g[0] == "harry_potter" && g[1] == "hermione" && g[2] == "ron" {
			found = true
		}
	}
	if !found {
		t.Error("ron: no wizard group")
	}
	for _, l := range catchphrases["ron"] {
		if len([]rune(l)) > 48 || langDataEN[l] == "" {
			t.Errorf("ron line %q: too long or not translated", l)
		}
	}
}

// DNH_SHEET=dir go test -run TestDnhArtSheet writes an enlarged sheet: front, side, back idle,
// two walk frames and the laugh, from the code and (second row) from the exported pictures.
func TestDnhArtSheet(t *testing.T) {
	dir := os.Getenv("DNH_SHEET")
	if dir == "" {
		t.Skip("set DNH_SHEET to a directory")
	}
	const z = 5
	cols := 6
	cellW, cellH := charW*z+4, charH*z+4
	big := image.NewRGBA(image.Rect(0, 0, cols*cellW, 2*len(dnhSlugs)*cellH))
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
	for si, slug := range dnhSlugs {
		sp := tiaSpec(t, slug)
		oy := 2 * si * cellH
		w1 := poseFor(AnimWalk, 1)
		w1.Anim, w1.F = AnimWalk, 1
		w3 := poseFor(AnimWalk, 3)
		w3.Anim, w3.F = AnimWalk, 3
		la := poseFor(AnimLaugh, 0)
		la.Anim = AnimLaugh
		poses := []struct {
			vw int
			p  Pose
		}{{vFront, poseFor(AnimIdle, 0)}, {vSide, poseFor(AnimIdle, 0)}, {vBack, poseFor(AnimIdle, 0)}, {vFront, w1}, {vSide, w3}, {vFront, la}}
		for i, q := range poses {
			put(renderChar(&sp, q.vw, q.p), i*cellW, oy)
		}
		if ca, err := loadCharAsset(filepath.Join(spritesDir("assets", "special_people"), slug)); err == nil {
			rig := buildRig(ca)
			for i, q := range poses {
				put(rig.render(q.vw, q.p), i*cellW, oy+cellH)
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, "dnh_sheet.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, big)
}
