package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func maneSpec() CharSpec {
	return CharSpec{Skin: hx(0xe8bc94), Hair: hx(0x5a3a22), Shirt: hx(0x3a60e0), Shirt2: hx(0xf4f4f4), Pants: hx(0x2a2a30),
		Shoes: hx(0x14141a), Top: TopTee, Body: BodyNormal, Hair2: HairBaldMane}
}

func maneIsHair(c, h color.RGBA) bool {
	return c.A != 0 && (c == h || c == shade(h, 0.78) || c == shade(h, 1.25))
}

// maneTopIsSkin: the first painted pixel from the top in the middle columns is skin.
func maneTopIsSkin(b *pixBuf, skin color.RGBA) bool {
	for _, x := range []int{charCX - 1, charCX} {
		for y := 0; y < charH; y++ {
			if c := b.get(x, y); c.A != 0 {
				if c != skin && c != shade(skin, 1.22) {
					return false
				}
				break
			}
		}
	}
	return true
}

// The barber sells it: "Calvo con melena" with its English name, a price, and wearing it sets the style.
func TestBaldManeOnSale(t *testing.T) {
	buildShopCatalog()
	var it *shopItem
	for i := range shopCatalog[shopAccess] {
		if shopCatalog[shopAccess][i].id == "hair:baldmane" {
			it = &shopCatalog[shopAccess][i]
		}
	}
	if it == nil {
		t.Fatal("hair:baldmane is not on sale")
	}
	if it.name != "Calvo con melena" || it.price <= 0 || it.slot != "hair" {
		t.Fatalf("item %+v", *it)
	}
	var s CharSpec
	wearFx["hair:baldmane"](&s)
	if s.Hair2 != HairBaldMane {
		t.Fatalf("wearing it gives hair style %d", s.Hair2)
	}
	saved := uiLang
	t.Cleanup(func() { uiLang = saved })
	uiLang = "en"
	if en := tr(it.name); en == it.name || en == "" {
		t.Errorf("no English name: %q", en)
	}
}

// Bald on top, hair from the sides down to the floor in every view, and the tips move with the walk.
func TestBaldManeLook(t *testing.T) {
	sp := maneSpec()
	h := sp.Hair
	floor := maneBottom(41)
	for vw := 0; vw < 3; vw++ {
		pp := poseFor(AnimIdle, 0)
		pp.Anim = AnimIdle
		b := genChar(&sp, vw, pp)
		low, left, right := 0, 0, 0
		for y := 30; y < charH; y++ {
			for x := 0; x < charW; x++ {
				if maneIsHair(b.get(x, y), h) {
					if y >= floor-1 {
						low++
					}
					if x < charCX-2 {
						left++
					}
					if x > charCX+2 {
						right++
					}
				}
			}
		}
		if low < 2 {
			t.Errorf("view %d: the hair does not reach the floor (%d hair pixels in the last rows)", vw, low)
		}
		if vw == vFront && (left < 8 || right < 8) {
			t.Errorf("front: the mane must fall on both sides (%d left, %d right)", left, right)
		}
		// the top of the head is bald: the first painted pixel of the middle column is skin (or its shine)
		if vw != vBack && !maneTopIsSkin(b, sp.Skin) {
			t.Errorf("view %d: the top of the head is not bald", vw)
		}
		minX, _, maxX, _ := bbox(b)
		if minX < 0 || maxX > charW-1 {
			t.Errorf("view %d: outside the sprite: %d-%d", vw, minX, maxX)
		}
	}
	// with long hair (HairLong) the top IS hair: the test above really sees the difference
	long := maneSpec()
	long.Hair2 = HairLong
	pl := poseFor(AnimIdle, 0)
	if maneTopIsSkin(genChar(&long, vFront, pl), long.Skin) {
		t.Error("HairLong top is skin: the bald check proves nothing")
	}
	// the tips swing while walking: the hair of two walking frames differs
	mask := func(f int) string {
		pp := poseFor(AnimWalk, f)
		pp.Anim, pp.F = AnimWalk, f
		b := genChar(&sp, vFront, pp)
		out := make([]byte, 0, charW*12)
		for y := charH - 12; y < charH; y++ {
			for x := 0; x < charW; x++ {
				if maneIsHair(b.get(x, y), h) {
					out = append(out, '#')
				} else {
					out = append(out, '.')
				}
			}
		}
		return string(out)
	}
	if mask(0) == mask(1) && mask(1) == mask(3) {
		t.Error("the mane does not move while walking")
	}
	if os.Getenv("MANE_PNG") != "" { // a picture of the three views (x8) to look at
		const z = 8
		im := image.NewRGBA(image.Rect(0, 0, 3*charW*z, charH*z))
		for i := range im.Pix {
			im.Pix[i] = 0x50
		}
		for vw := 0; vw < 3; vw++ {
			pp := poseFor(AnimIdle, 0)
			b := genChar(&sp, vw, pp)
			for y := 0; y < charH; y++ {
				for x := 0; x < charW; x++ {
					if c := b.get(x, y); c.A != 0 {
						for dy := 0; dy < z; dy++ {
							for dx := 0; dx < z; dx++ {
								im.Set(vw*charW*z+x*z+dx, y*z+dy, c)
							}
						}
					}
				}
			}
		}
		f, err := os.Create(filepath.Join(os.Getenv("MANE_PNG"), "mane_views.png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, im)
	}
}
