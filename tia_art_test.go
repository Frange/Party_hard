package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var tiaArtSlugs = []string{"mortadelo", "filemon", "bender"}

func tiaSpec(t *testing.T, slug string) CharSpec {
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

// tiaTop: first row with a pixel (the top of the head).
func tiaTop(b *pixBuf) int {
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] != 0 {
				return y
			}
		}
	}
	return b.h
}

// The three draw with their own art in the three views, inside the box (nothing on the outer
// ring), with every body part present for the rig, and walking changes the frames.
func TestTiaArtDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic, AnimCarry}
	for _, slug := range tiaArtSlugs {
		sp := tiaSpec(t, slug)
		if !isTia(&sp) && !isBenderArt(&sp) {
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
					if n < 120 {
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

// Mortadelo stands well above an ordinary guest and Filemon below; the feet are on the same row.
func TestTiaArtStature(t *testing.T) {
	plain := tiaSpec(t, "mortadelo")
	plain.Name, plain.Stature = "twin", 0
	idle := poseFor(AnimIdle, 0)
	tops := map[string]int{"plain": tiaTop(renderChar(&plain, vFront, idle))}
	for _, slug := range []string{"mortadelo", "filemon"} {
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
			if bottom < 41 || bottom > 42 {
				t.Errorf("%s vw %d: feet end on row %d", slug, vw, bottom)
			}
		}
		tops[slug] = tiaTop(renderChar(&sp, vFront, idle))
	}
	if tops["mortadelo"] > tops["plain"]-4 {
		t.Errorf("Mortadelo not tall enough: top %d, ordinary guest %d", tops["mortadelo"], tops["plain"])
	}
	if tops["filemon"] < tops["plain"]+3 {
		t.Errorf("Filemon not short enough: top %d, ordinary guest %d", tops["filemon"], tops["plain"])
	}
	t.Logf("tops: %v", tops)
}

// TIA_SHEET=dir [TIA_TAG=x] go test -run TestTiaArtSheet writes an enlarged sheet of the three
// (front, side, back idle and two walk frames), from the code and from the exported pictures.
func TestTiaArtSheet(t *testing.T) {
	dir := os.Getenv("TIA_SHEET")
	if dir == "" {
		t.Skip("set TIA_SHEET to a directory")
	}
	const z = 5
	cols := 6
	cellW, cellH := charW*z+4, charH*z+4
	big := image.NewRGBA(image.Rect(0, 0, cols*cellW, len(tiaArtSlugs)*cellH))
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
	for row, slug := range tiaArtSlugs {
		sp := tiaSpec(t, slug)
		var frames []*pixBuf
		if from := os.Getenv("TIA_FROM"); from != "" { // the exported pictures (front, side, back)
			ca, err := loadCharAsset(filepath.Join(spritesDir(from, "special_people"), slug))
			if err != nil {
				t.Fatal(err)
			}
			for i, v := range ca.Views {
				put(v, i*cellW, row*cellH)
			}
			continue
		}
		for _, vw := range []int{vFront, vSide, vBack} {
			frames = append(frames, renderChar(&sp, vw, poseFor(AnimIdle, 0)))
		}
		for _, f := range []int{1, 3} {
			pp := poseFor(AnimWalk, f)
			pp.Anim, pp.F = AnimWalk, f
			frames = append(frames, renderChar(&sp, vSide, pp))
		}
		pp := poseFor(AnimDance3, 1)
		pp.Anim, pp.F = AnimDance3, 1
		frames = append(frames, renderChar(&sp, vFront, pp))
		for i, b := range frames {
			put(b, i*cellW, row*cellH)
		}
	}
	os.MkdirAll(dir, 0o755)
	f, err := os.Create(filepath.Join(dir, os.Getenv("TIA_TAG")+"tia_sheet.png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}
