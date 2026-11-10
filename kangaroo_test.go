package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func kangFrames() []int {
	return []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3, fWarn, fKick}
}

func TestKangarooArt(t *testing.T) {
	var sigs []string
	for _, f := range kangFrames() {
		b := genKangarooFrame(f, 0)
		if b.w > 44 || b.h > 44 || b.w < 30 || b.h < 34 {
			t.Fatalf("frame %d: size %dx%d out of range", f, b.w, b.h)
		}
		// key colours of the red coat: fur, dark fur and cream pouch
		var orange, brick, cream, opaque int
		minX, maxX, maxY := b.w, 0, 0
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if b.p[(y*b.w+x)*4+3] == 0 {
					continue
				}
				opaque++
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
				switch b.get(x, y) {
				case kangarooCoats[0].body:
					orange++
				case kangarooCoats[0].dark:
					brick++
				case kangarooCoats[0].light:
					cream++
				}
			}
		}
		if orange < 120 || brick < 15 || cream < 30 {
			t.Errorf("frame %d: orange=%d brick=%d cream=%d", f, orange, brick, cream)
		}
		if (f == fStand && maxY != b.h-1) || maxX >= b.w || minX > 2 {
			t.Errorf("frame %d: feet/tail not on the box edges (minX=%d maxX=%d maxY=%d)", f, minX, maxX, maxY)
		}
		sigs = append(sigs, string(b.p))
	}
	seen := map[string]int{}
	for i, s := range sigs {
		if j, ok := seen[s]; ok {
			t.Errorf("frames %d and %d are identical", kangFrames()[j], kangFrames()[i])
		}
		seen[s] = i
	}
	for coat := 1; coat < 3; coat++ {
		if string(genKangarooFrame(fStand, coat).p) == sigs[0] {
			t.Errorf("coat %d equals coat 0", coat)
		}
	}
}

// TestKangarooDump writes a contact sheet when PH_KANG_PNG is set (debug aid).
func TestKangarooDump(t *testing.T) {
	path := os.Getenv("PH_KANG_PNG")
	if path == "" {
		t.Skip("PH_KANG_PNG not set")
	}
	const z = 6
	fr := kangFrames()
	img := image.NewRGBA(image.Rect(0, 0, 40*z*len(fr), 40*z))
	for i := range img.Pix {
		img.Pix[i] = 90
		if i%4 == 3 {
			img.Pix[i] = 255
		}
	}
	for i, f := range fr {
		b := genKangarooFrame(f, 0).outlinedSel(0.4).toRGBA()
		for y := 0; y < b.Bounds().Dy(); y++ {
			for x := 0; x < b.Bounds().Dx(); x++ {
				c := b.RGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				for dy := 0; dy < z; dy++ {
					for dx := 0; dx < z; dx++ {
						img.Set(i*40*z+x*z+dx, y*z+dy, color.RGBA{c.R, c.G, c.B, 255})
					}
				}
			}
		}
	}
	f, _ := os.Create(path)
	png.Encode(f, img)
	f.Close()
}
