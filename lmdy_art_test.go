package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

var lmdySlugs = []string{"lara_croft", "marty_mcfly", "doc_brown", "yoda"}

func lmdySpec(t *testing.T, slug string) CharSpec {
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

func lmdyBox(b *pixBuf) (x0, y0, x1, y1 int) {
	x0, y0, x1, y1 = 99, 99, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				x0, y0, x1, y1 = minI(x0, x), minI(y0, y), maxI(x1, x), maxI(y1, y)
			}
		}
	}
	return
}

// The four draw with their own art in the three views and the usual animations, inside the box
// (nothing on the outer ring once outlined), with every body part present for the rig, and
// walking changes the frames.
func TestLmdyArtDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimDance7, AnimPanic, AnimHop, AnimCarry, AnimWave}
	for _, slug := range lmdySlugs {
		sp := lmdySpec(t, slug)
		if !isLmdy(&sp) {
			t.Fatalf("%s: no own art", slug)
		}
		for vw := 0; vw < 3; vw++ {
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					b := renderChar(&sp, vw, ownPose(a, f))
					n, edge := charimpAPix(b)
					if edge > 0 {
						t.Errorf("%s vw %d anim %d f %d: %d pixels on the edge", slug, vw, a, f, edge)
					}
					if n < 120 {
						t.Errorf("%s vw %d anim %d f %d: only %d pixels", slug, vw, a, f, n)
					}
				}
			}
			seen := map[string]bool{}
			for f := 0; f < animFrameCount(AnimWalk); f++ {
				seen[string(renderChar(&sp, vw, ownPose(AnimWalk, f)).p)] = true
			}
			if len(seen) < 3 {
				t.Errorf("%s vw %d: only %d different walking frames", slug, vw, len(seen))
			}
			trackParts = true
			b := genChar(&sp, vw, poseFor(AnimIdle, 0))
			trackParts = false
			parts := map[uint8]int{}
			for i, id := range b.pid {
				if b.p[i*4+3] != 0 {
					parts[id]++
				}
			}
			for id := uint8(PartHead); id <= PartLegR; id++ {
				if parts[id] == 0 && !(vw == vSide && (id == PartArmL || id == PartLegL)) {
					t.Errorf("%s vw %d: part %d missing", slug, vw, id)
				}
			}
			if parts[0] != 0 {
				t.Errorf("%s vw %d: %d pixels with no part", slug, vw, parts[0])
			}
		}
	}
}

// lmdyCount: pixels of the idle front view close to a colour.
func lmdyCount(b *pixBuf, c color.RGBA) int {
	n := 0
	for i := 0; i+3 < len(b.p); i += 4 {
		if b.p[i+3] == 0 {
			continue
		}
		d := abs(int(b.p[i])-int(c.R)) + abs(int(b.p[i+1])-int(c.G)) + abs(int(b.p[i+2])-int(c.B))
		if d < 30 {
			n++
		}
	}
	return n
}

// Each one shows its signature colours (turquoise top and brown shorts, red vest and jeans,
// yellow coat and white hair, green skin and cream robe), the four look different from each
// other and from an ordinary guest, Yoda is much shorter, and the feet rest on the usual row.
func TestLmdyArtLooksLikeThem(t *testing.T) {
	idle := ownPose(AnimIdle, 0)
	want := map[string][][2]any{
		"lara_croft":  {{laTop, 12}, {laShort, 8}, {laBoot, 6}, {laHair, 10}, {laGun, 2}},
		"marty_mcfly": {{maVest, 20}, {maJean, 12}, {maShoe, 3}, {maChkK, 3}, {maHair, 10}},
		"doc_brown":   {{dcCoat, 30}, {dcHair, 25}, {dcShirt, 6}, {dcPant, 6}, {dcBox, 2}},
		"yoda":        {{yoSkin, 40}, {yoRobe, 30}, {yoColl, 6}, {yoStick, 6}, {yoEye, 4}},
	}
	plain := lmdySpec(t, "lara_croft")
	plain.Name = "twin"
	pb := genChar(&plain, vFront, idle)
	_, plainTop, _, plainBottom := lmdyBox(pb)
	seen := map[string]string{string(pb.p): "plain"}
	for _, slug := range lmdySlugs {
		sp := lmdySpec(t, slug)
		b := genChar(&sp, vFront, idle)
		for _, w := range want[slug] {
			if n := lmdyCount(b, w[0].(color.RGBA)); n < w[1].(int) {
				t.Errorf("%s: colour %v only %d pixels, want %d", slug, w[0], n, w[1])
			}
		}
		if o, dup := seen[string(b.p)]; dup {
			t.Errorf("%s looks the same as %s", slug, o)
		}
		seen[string(b.p)] = slug
		for vw := 0; vw < 3; vw++ {
			_, top, _, bottom := lmdyBox(genChar(&sp, vw, idle))
			if bottom < plainBottom-1 || bottom > plainBottom+1 {
				t.Errorf("%s vw %d: feet end on row %d, ordinary guest %d", slug, vw, bottom, plainBottom)
			}
			if slug == "yoda" && top < plainTop+8 {
				t.Errorf("yoda vw %d: top %d, not short enough (ordinary guest %d)", vw, top, plainTop)
			}
			if slug != "yoda" && (top < plainTop-5 || top > plainTop+3) {
				t.Errorf("%s vw %d: top %d, ordinary guest %d", slug, vw, top, plainTop)
			}
		}
	}
}

// LMDY_SHEET=path.png go test -run TestLmdyArtSheet writes an enlarged sheet: one row per
// character, front/side/back standing then walking frames, and an ordinary guest for scale.
func TestLmdyArtSheet(t *testing.T) {
	out := os.Getenv("LMDY_SHEET")
	if out == "" {
		t.Skip("set LMDY_SHEET to dump the sheet")
	}
	type cell struct{ vw, a, f int }
	cols := []cell{{vFront, AnimIdle, 0}, {vSide, AnimIdle, 0}, {vBack, AnimIdle, 0}, {vFront, AnimWalk, 1}, {vSide, AnimWalk, 1}, {vSide, AnimWalk, 3}, {vFront, AnimDance3, 1}, {vFront, AnimLaugh, 0}}
	const z = 6
	rows := append([]string{}, lmdySlugs...)
	rows = append(rows, "twin")
	img := image.NewRGBA(image.Rect(0, 0, len(cols)*charW*z, len(rows)*charH*z))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 0x78, 0x8c, 0x78, 255
	}
	for ri, slug := range rows {
		var sp CharSpec
		if slug == "twin" {
			sp = lmdySpec(t, "marty_mcfly")
			sp.Name = "twin"
		} else {
			sp = lmdySpec(t, slug)
		}
		for ci, c := range cols {
			b := renderChar(&sp, c.vw, ownPose(c.a, c.f))
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					col := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							img.SetRGBA((ci*charW+x)*z+dx, (ri*charH+y)*z+dy, col)
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
