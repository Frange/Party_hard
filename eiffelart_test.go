package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func eifSprite() *pixBuf {
	k := propKinds["eiffel"]
	return renderPropSprite("eiffel", k, 0)
}

func TestEiffelArtShape(t *testing.T) {
	k := &PropKind{W: eifW, H: eifH}
	b := newPix(k.W+2, k.H+2)
	b.ox, b.oy = 1, 1
	drawEiffel(b, 0)
	if b.w != 132 || b.h != 226 {
		t.Fatalf("size %dx%d", b.w, b.h)
	}
	// the arch: centre of the lower part is see-through
	for y := 196; y < eifGround; y++ {
		half := 2
		if y >= 200 {
			half = 14
		}
		if y >= 208 {
			half = 25
		}
		for x := 65 - half; x <= 65+half; x++ {
			if b.get(x, y).A != 0 {
				t.Fatalf("arch blocked at %d,%d", x, y)
			}
		}
	}
	// sky between the legs under the second platform
	if b.get(65, 165).A != 0 {
		t.Error("no gap between the legs")
	}
	// both legs present on the ground rows, antenna on top
	for _, x := range []int{65 - 50, 65 + 50} {
		if b.get(x, eifGround-1).A == 0 {
			t.Errorf("leg missing at x=%d", x)
		}
	}
	if b.get(65, 2).A == 0 {
		t.Error("no antenna")
	}
	// the iron is dark and warm-grey, the windows are lit
	dark, lit, n := 0, 0, 0
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			c := b.get(x, y)
			if c.A == 0 {
				continue
			}
			n++
			if int(c.R)+int(c.G)+int(c.B) < 3*150 {
				dark++
			}
			if c.R > 230 && c.G > 200 && c.B < 180 {
				lit++
			}
		}
	}
	if n < 1500 || dark*10 < n*9 {
		t.Errorf("iron: %d opaque, %d dark", n, dark)
	}
	if lit < 4 {
		t.Errorf("no lit windows (%d)", lit)
	}
}

func TestEiffelRenderedKeepsArchOpen(t *testing.T) {
	s := eifSprite()
	cx := s.w / 2
	for y := s.h - 28; y < s.h-4; y++ {
		if s.get(cx, y).A != 0 {
			t.Fatalf("rendered arch blocked at y=%d", y)
		}
	}
}

// EIFFEL_EXPORT=assets-root go test -run TestEiffelExport writes only the eiffel folder (meta kept, anchors/foot updated).
func TestEiffelExport(t *testing.T) {
	root := os.Getenv("EIFFEL_EXPORT")
	if root == "" {
		t.Skip("set EIFFEL_EXPORT to an assets folder")
	}
	k := propKinds["eiffel"]
	oa := &ObjAsset{}
	ax, ay := int((float64(k.W)/2+1)*wk), int((float64(k.H)+1)*wk)
	fw, fh := int(k.FW*wk+0.5), int(k.FH*wk+0.5)
	oa.Meta = ObjMeta{Name: "eiffel", Category: "furniture", Solid: false, AnchorX: ax, AnchorY: ay, FootW: fw, FootH: fh, FootX: ax - fw/2, FootY: ay - fh}
	oa.Frames = []*pixBuf{eifSprite()}
	if err := saveObjAsset(root, oa); err != nil {
		t.Fatal(err)
	}
}

func TestEiffelExportedMatches(t *testing.T) {
	oa, err := loadObjAsset(spriteDirOf("assets", "objetos", "eiffel"))
	if err != nil {
		t.Skip(err)
	}
	if oa.Meta.Solid || oa.Meta.FootW != 84 || oa.Meta.FootH != 14 {
		t.Errorf("meta %+v", oa.Meta)
	}
	f := oa.Frames[0]
	if f.w != 132 || f.h != 226 {
		t.Fatalf("exported size %dx%d", f.w, f.h)
	}
	for y := f.h - 28; y < f.h-4; y++ {
		if f.get(f.w/2, y).A != 0 {
			t.Fatalf("exported arch blocked at y=%d", y)
		}
	}
}

// EIFFEL_SHEET=file.png writes the isolated prop on a sky colour, 4x, to look at it.
func TestEiffelSheet(t *testing.T) {
	out := os.Getenv("EIFFEL_SHEET")
	if out == "" {
		t.Skip("set EIFFEL_SHEET")
	}
	s := eifSprite()
	im := s.toRGBA()
	const z = 4
	big := newPix(s.w*z, s.h*z)
	for y := 0; y < big.h; y++ {
		for x := 0; x < big.w; x++ {
			sc := im.RGBAAt(x/z, y/z)
			o := (y*big.w + x) * 4
			if sc.A == 0 {
				big.p[o], big.p[o+1], big.p[o+2] = 120, 170, 220
				if (y / z) > s.h-30 {
					big.p[o], big.p[o+1], big.p[o+2] = 80, 130, 60
				}
			} else {
				big.p[o], big.p[o+1], big.p[o+2] = sc.R, sc.G, sc.B
			}
			big.p[o+3] = 255
		}
	}
	f, _ := os.Create(filepath.Clean(out))
	defer f.Close()
	png.Encode(f, big.toRGBA())
}
