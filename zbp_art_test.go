package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var zbpSlugs = []string{"zipi", "zape", "pantera_rosa", "mr_bean"}

// The four draw with their own art in the three views and the usual animations, inside the box
// (no more on the outer ring than the ordinary body holding the same thing), with every body part for the rig, and walking changes the frames.
func TestZbpArtDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimDance7, AnimPanic, AnimHop, AnimCarry, AnimWave}
	for _, slug := range zbpSlugs {
		sp := tiaSpec(t, slug)
		if !isZbp(&sp) {
			t.Fatalf("%s: no own art", slug)
		}
		twin := sp // the ordinary body with the same held object (Mr Bean's briefcase)
		twin.Name = "twin"
		for vw := 0; vw < 3; vw++ {
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					b := renderChar(&sp, vw, ownPose(a, f))
					n, edge := charimpAPix(b)
					_, tEdge := charimpAPix(renderChar(&twin, vw, ownPose(a, f)))
					if edge > tEdge {
						t.Errorf("%s vw %d anim %d f %d: %d pixels on the edge (ordinary body: %d)", slug, vw, a, f, edge, tEdge)
					}
					if n < 120 {
						t.Errorf("%s vw %d anim %d f %d: only %d pixels", slug, vw, a, f, n)
					}
				}
			}
			seen := map[string]bool{}
			for f := 0; f < animFrameCount(AnimWalk); f++ {
				seen[string(genChar(&sp, vw, ownPose(AnimWalk, f)).p)] = true
			}
			if len(seen) < 3 {
				t.Errorf("%s vw %d: only %d different walking frames", slug, vw, len(seen))
			}
			trackParts = true
			b := genChar(&sp, vw, ownPose(AnimIdle, 0))
			trackParts = false
			got := map[uint8]bool{}
			for i, id := range b.pid {
				if b.p[i*4+3] != 0 {
					got[id] = true
				}
			}
			for id := uint8(PartHead); id <= PartLegR; id++ {
				if !got[id] && !(vw == vSide && (id == PartArmL || id == PartLegL)) {
					t.Errorf("%s vw %d: part %d missing", slug, vw, id)
				}
			}
		}
	}
}

// The twins are children (well below an ordinary guest), the Panther is the tallest and Mr Bean
// is taller than an ordinary guest; all stand on the same row.
func TestZbpArtStature(t *testing.T) {
	plain := tiaSpec(t, "mr_bean")
	plain.Name, plain.Stature = "twin", 0
	idle := ownPose(AnimIdle, 0)
	tops := map[string]int{"plain": tiaTop(renderChar(&plain, vFront, idle))}
	for _, slug := range zbpSlugs {
		sp := tiaSpec(t, slug)
		for vw := 0; vw < 3; vw++ {
			b := renderChar(&sp, vw, idle)
			bottom := 0
			for i := 3; i < len(b.p); i += 4 {
				if b.p[i] != 0 {
					bottom = (i / 4) / b.w
				}
			}
			if bottom != 42 {
				t.Errorf("%s vw %d: feet end on row %d", slug, vw, bottom)
			}
		}
		tops[slug] = tiaTop(renderChar(&sp, vFront, idle))
	}
	if tops["zipi"] < tops["plain"]+3 || tops["zape"] < tops["plain"]+3 {
		t.Errorf("the twins are not children: %v", tops)
	}
	if tops["pantera_rosa"] > tops["plain"]-4 || tops["pantera_rosa"] >= tops["mr_bean"] {
		t.Errorf("the Panther is not the tallest: %v", tops)
	}
	if tops["mr_bean"] > tops["plain"] {
		t.Errorf("Mr Bean is shorter than an ordinary guest: %v", tops)
	}
	t.Logf("tops: %v", tops)
}

// zbpColours: how many pixels of the drawing are near a colour.
func zbpColours(b *pixBuf, c color.RGBA) int { return zbpColoursAbove(b, c, b.h) }

// zbpColoursAbove: the same, only in the rows above maxY (the head of the twins).
func zbpColoursAbove(b *pixBuf, c color.RGBA, maxY int) int {
	n := 0
	for i := 0; i < len(b.p) && i/4/b.w < maxY; i += 4 {
		if b.p[i+3] == 0 {
			continue
		}
		d := absI(int(b.p[i])-int(c.R)) + absI(int(b.p[i+1])-int(c.G)) + absI(int(b.p[i+2])-int(c.B))
		if d < 40 {
			n++
		}
	}
	return n
}

// The four are told apart: every pair differs in every view, Zipi is blond and Zape dark in all
// three views, the twins wear the red jumper with yellow trims and blue socks, the Panther is pink
// with a red nose and yellow eyes, Mr Bean wears tweed with a red tie in front.
func TestZbpArtDistinct(t *testing.T) {
	idle := ownPose(AnimIdle, 0)
	draw := func(slug string, vw int) *pixBuf { sp := tiaSpec(t, slug); return genChar(&sp, vw, idle) }
	for vw := 0; vw < 3; vw++ {
		for i, a := range zbpSlugs {
			for _, c := range zbpSlugs[i+1:] {
				if string(draw(a, vw).p) == string(draw(c, vw).p) {
					t.Errorf("%s and %s look the same in view %d", a, c, vw)
				}
			}
		}
		zi, za := draw("zipi", vw), draw("zape", vw)
		const neck = 26 // the heads end above the jumper
		ziB, zaB := zbpColoursAbove(zi, zbBlond, neck), zbpColoursAbove(za, zbBlond, neck)
		ziK, zaK := zbpColoursAbove(zi, zbBlack, neck), zbpColoursAbove(za, zbBlack, neck)
		if ziB < 30 || zaB > 0 || zaK < 30 || ziK > 6 {
			t.Errorf("view %d: Zipi blond %d / black %d px, Zape blond %d / black %d px", vw, ziB, ziK, zaB, zaK)
		}
		for _, k := range []string{"zipi", "zape"} {
			b := draw(k, vw)
			j := zbpColours(b, zbJumper) + zbpColours(b, zbJumperL) + zbpColours(b, zbJumperD)
			tr := zbpColours(b, zbTrim) - zbpColoursAbove(b, zbTrim, neck) // the blond hair is near the trim colour
			so, sh := zbpColours(b, zbSock)+zbpColours(b, zbSockL), zbpColours(b, zbShoe)+zbpColours(b, zbShoeD)
			if j < 20 || tr < 4 || so < 2 || sh < 4 {
				t.Errorf("%s view %d: jumper %d, trims %d, socks %d, shoes %d", k, vw, j, tr, so, sh)
			}
		}
		pr := draw("pantera_rosa", vw)
		if zbpColours(pr, zbPink) < 80 {
			t.Errorf("panther view %d: only %d pink pixels", vw, zbpColours(pr, zbPink))
		}
		mb := draw("mr_bean", vw)
		tw, tr := zbpColours(mb, zbTweed)+zbpColours(mb, zbTweedL)+zbpColours(mb, zbTweedD), zbpColours(mb, zbTrou)+zbpColours(mb, zbTrouL)
		if tw < 40 || tr < 20 {
			t.Errorf("Mr Bean view %d: tweed %d, trousers %d", vw, tw, tr)
		}
	}
	pr, mb := draw("pantera_rosa", vFront), draw("mr_bean", vFront)
	if zbpColours(pr, zbNose) < 2 || zbpColours(pr, zbYellow) < 2 {
		t.Error("the Panther has no red nose or yellow eyes")
	}
	if zbpColours(mb, zbTie) < 4 || zbpColours(mb, zbPin) < 1 || zbpColours(mb, zbPatch) < 2 {
		t.Error("Mr Bean has no red tie, pin or elbow patches")
	}
	// Zipi hides his hands behind his back when he stands: no skin under the shoulders in front
	zi := tiaSpec(t, "zipi")
	trackParts = true
	b := genChar(&zi, vFront, idle)
	trackParts = false
	hands := 0
	for i, id := range b.pid {
		if (id == PartArmL || id == PartArmR) && b.p[i*4+3] != 0 && zbpColours(&pixBuf{w: 1, h: 1, p: b.p[i*4 : i*4+4]}, zbSkin) == 1 {
			hands++
		}
	}
	if hands != 0 {
		t.Errorf("Zipi shows %d hand pixels in front while standing", hands)
	}
}

// ZBP_SHEET=dir [ZBP_FROM=assets-root] go test -run TestZbpArtSheet writes an enlarged sheet of the
// four (front, side, back idle, two walk frames side, a dance front), or the exported pictures.
func TestZbpArtSheet(t *testing.T) {
	dir := os.Getenv("ZBP_SHEET")
	if dir == "" {
		t.Skip("set ZBP_SHEET to a directory")
	}
	const z = 6
	cols := 7
	cellW, cellH := charW*z+4, charH*z+4
	big := image.NewRGBA(image.Rect(0, 0, cols*cellW, len(zbpSlugs)*cellH))
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
	from := os.Getenv("ZBP_FROM")
	for row, slug := range zbpSlugs {
		sp := tiaSpec(t, slug)
		if from != "" {
			ca, err := loadCharAsset(filepath.Join(spritesDir(from, "special_people"), slug))
			if err != nil {
				t.Fatal(err)
			}
			for i, v := range ca.Views {
				put(v, i*cellW, row*cellH)
			}
			continue
		}
		var frames []*pixBuf
		for _, vw := range []int{vFront, vSide, vBack} {
			frames = append(frames, renderChar(&sp, vw, ownPose(AnimIdle, 0)))
		}
		frames = append(frames, renderChar(&sp, vSide, ownPose(AnimWalk, 1)), renderChar(&sp, vSide, ownPose(AnimWalk, 3)),
			renderChar(&sp, vFront, ownPose(AnimWalk, 1)), renderChar(&sp, vFront, ownPose(AnimDance3, 1)))
		for i, b := range frames {
			put(b, i*cellW, row*cellH)
		}
	}
	os.MkdirAll(dir, 0o755)
	f, err := os.Create(filepath.Join(dir, os.Getenv("ZBP_TAG")+"zbp_sheet.png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}
