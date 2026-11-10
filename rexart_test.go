package main

import (
	"image"
	"image/png"
	"os"
	"testing"
)

// Every pose of the T-Rex paints something, the poses differ from each other (walk frames, front, back, meal) and
// nothing is drawn outside the buffer edge. REX_DUMP=dir writes a contact sheet for a human look.
func TestRexArtPosesDrawAndDiffer(t *testing.T) {
	k := propKinds["jp_trex"]
	if k == nil || k.Vars != rexPoses {
		t.Fatalf("jp_trex has %v variants, want %d", k, rexPoses)
	}
	var sprites []*pixBuf
	for v := 0; v < rexPoses; v++ {
		b := renderPropSprite("jp_trex", k, v)
		n := 0
		for i := 3; i < len(b.p); i += 4 {
			if b.p[i] != 0 {
				n++
			}
		}
		if n < 1500 {
			t.Errorf("pose %d paints only %d pixels", v, n)
		}
		for x := 0; x < b.w; x++ { // the sprite keeps off the very edge of the buffer
			if b.p[(0*b.w+x)*4+3] != 0 || b.p[((b.h-1)*b.w+x)*4+3] != 0 {
				t.Errorf("pose %d touches the top or bottom edge", v)
				break
			}
		}
		sprites = append(sprites, b)
	}
	for i := range sprites {
		for j := i + 1; j < len(sprites); j++ {
			if string(sprites[i].p) == string(sprites[j].p) {
				t.Errorf("poses %d and %d are identical", i, j)
			}
		}
	}
	if dir := os.Getenv("REX_DUMP"); dir != "" {
		sheet := image.NewRGBA(image.Rect(0, 0, 4*164, 3*96))
		for i, b := range sprites {
			im := b.toRGBA()
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					sheet.SetRGBA((i%4)*164+x, (i/4)*96+y, im.RGBAAt(x, y))
				}
			}
		}
		f, err := os.Create(dir + "/rex.png")
		if err == nil {
			png.Encode(f, sheet)
			f.Close()
		}
		for si, set := range [][]int{{0, 4}, {6, 8}, {1, 10}} { // 3x on a green floor
			z := image.NewRGBA(image.Rect(0, 0, 2*164*3, 96*3))
			for i := 0; i < z.Rect.Dx()*z.Rect.Dy(); i++ {
				z.Pix[i*4], z.Pix[i*4+1], z.Pix[i*4+2], z.Pix[i*4+3] = 110, 150, 80, 255
			}
			for n, v := range set {
				im := sprites[v].toRGBA()
				for y := 0; y < im.Rect.Dy()*3; y++ {
					for x := 0; x < im.Rect.Dx()*3; x++ {
						if p := im.RGBAAt(x/3, y/3); p.A != 0 {
							z.SetRGBA(n*164*3+x, y, p)
						}
					}
				}
			}
			if f, err := os.Create(dir + "/rex_z" + string(rune('0'+si)) + ".png"); err == nil {
				png.Encode(f, z)
				f.Close()
			}
		}
	}
	// the party event's T-Rex uses the same art
	if a, b := genRexFrame(0), genRexFrame(1); string(a.p) == string(b.p) {
		t.Error("the event T-Rex does not step")
	}
}
