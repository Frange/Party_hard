package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// ptfSlugs: the famous guests redrawn in ptf_art.go (the Pope is a VIP, see ptfPopeSpec).
var ptfSlugs = []string{"michael_jackson", "don_quijote", "sancho_panza"}

func ptfPopeSpec(t *testing.T) CharSpec {
	for _, d := range vipRegistry() {
		if d.id == "pope" {
			return d.spec()
		}
	}
	t.Fatal("no pope VIP")
	return CharSpec{}
}

// ptfAll: the four looks, the Pope first.
func ptfAll(t *testing.T) []CharSpec {
	out := []CharSpec{ptfPopeSpec(t)}
	for _, s := range ptfSlugs {
		out = append(out, charimpASpec(t, s))
	}
	return out
}

// PTF_SHEET=dir [PTF_TAG=antes] go test -run TestPtfSheet writes an enlarged sheet of the four
// (front, side, back idle, a walk frame and a dance frame).
func TestPtfSheet(t *testing.T) {
	dir := os.Getenv("PTF_SHEET")
	if dir == "" {
		t.Skip("set PTF_SHEET")
	}
	const z = 5
	specs := ptfAll(t)
	cellW, cellH := 5*charW*z+10, charH*z+6
	big := image.NewRGBA(image.Rect(0, 0, cellW, len(specs)*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x70, 0x70, 0x78, 255
	}
	for si := range specs {
		sp := specs[si]
		for vi, vw := range []int{vFront, vSide, vBack, vFront, vSide} {
			pp := poseFor(AnimIdle, 0)
			switch vi {
			case 3:
				pp = poseFor(AnimWalk, 1)
				pp.Anim, pp.F = AnimWalk, 1
			case 4:
				pp = gaitPoseFor(AnimMoon, 1)
				pp.Anim, pp.F = AnimMoon, 1
			}
			b := renderChar(&sp, vw, pp)
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							big.SetRGBA((vi*charW+x)*z+dx, si*cellH+y*z+dy, c)
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("ptf_%s.png", os.Getenv("PTF_TAG"))))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}

// The four draw in the three views inside their box, the details really show (against the same
// body without the hooks) and walking changes the frames.
func TestPtfDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic, AnimMoon}
	for _, sp := range ptfAll(t) {
		sp := sp
		if ptfKind(&sp) == ptfNone {
			t.Fatalf("%q: no details registered", sp.Name)
		}
		twin := sp
		twin.Name, twin.Pattern = "twin", 0
		for vw := 0; vw < 3; vw++ {
			differs := false
			for _, a := range anims {
				for f := 0; f < animFrameCount(a); f++ {
					pp := poseFor(a, f)
					if a >= AnimCount {
						pp = gaitPoseFor(a, f)
					}
					pp.Anim, pp.F = a, f
					b, tb := renderChar(&sp, vw, pp), renderChar(&twin, vw, pp)
					n, edge := charimpAPix(b)
					tn, tEdge := charimpAPix(tb)
					if edge > tEdge+2 {
						t.Errorf("%q vw %d anim %d f %d: %d pixels on the edge (plain %d)", sp.Name, vw, a, f, edge, tEdge)
					}
					if n < 60 || n < tn*9/10 {
						t.Errorf("%q vw %d anim %d f %d: only %d pixels (plain %d)", sp.Name, vw, a, f, n, tn)
					}
					if string(b.p) != string(tb.p) {
						differs = true
					}
				}
			}
			if !differs {
				t.Errorf("%q vw %d: the details never show", sp.Name, vw)
			}
			w0 := charimpAWalkFrame(&sp, vw, 0)
			changed := false
			for f := 1; f < animFrameCount(AnimWalk); f++ {
				if string(charimpAWalkFrame(&sp, vw, f)) != string(w0) {
					changed = true
				}
			}
			if !changed {
				t.Errorf("%q vw %d: walking does not change the frames", sp.Name, vw)
			}
		}
	}
}

func ptfAt(b *pixBuf, x, y int) color.RGBA {
	i := (y*b.w + x) * 4
	return color.RGBA{b.p[i], b.p[i+1], b.p[i+2], b.p[i+3]}
}

func ptfWhitish(c color.RGBA) bool {
	return c.A > 0 && c.R > 0xb0 && c.G > 0xb0 && c.B > 0xa8 && int(c.R)-int(c.B) < 24
}
func ptfGoldish(c color.RGBA) bool {
	return c.A > 0 && c.R > 160 && c.G > 120 && c.B < 120
}

// The Pope's new look: not the old one (pointed wizard hat, round glasses, plain robe), a white
// cassock that reaches the shoes, a golden stripe down the middle, a mitre with a golden cross
// above the head, and the same id, texts and stats.
func TestPtfPopeLook(t *testing.T) {
	sp := ptfPopeSpec(t)
	old := vipBase()
	old.Skin = hx(0xf0d0b8)
	old.Hair2, old.Glasses, old.Hat, old.HatCol = HairBald, GlassRound, HatWizard, hx(0xf4f0e0)
	old.Top, old.Shirt, old.Shirt2, old.Accent, old.Pants = TopRobe, hx(0xf4f2ea), hx(0xe8c040), hx(0xe8c040), hx(0xf4f2ea)
	old.Carry, old.CarryCol = HoldSaber, hx(0xf0c030)
	for vw := 0; vw < 3; vw++ {
		if string(renderChar(&sp, vw, poseFor(AnimIdle, 0)).p) == string(renderChar(&old, vw, poseFor(AnimIdle, 0)).p) {
			t.Errorf("vw %d: the Pope still looks like before", vw)
		}
	}
	if sp.Name != "" || sp.Hat == HatWizard || sp.Glasses != GlassNone || sp.Top != TopRobe {
		t.Errorf("spec: %+v", sp)
	}
	b := renderChar(&sp, vFront, poseFor(AnimIdle, 0))
	// the cassock: every row between the hips and the shoes is white beside the golden stripe
	hem, gold := 0, 0
	for y := 27; y < 41; y++ {
		if ptfWhitish(ptfAt(b, charCX-2, y)) && ptfWhitish(ptfAt(b, charCX+2, y)) {
			hem++
		}
		for x := charCX - 1; x <= charCX+1; x++ {
			if ptfGoldish(ptfAt(b, x, y)) {
				gold++
				break
			}
		}
	}
	if hem < 12 {
		t.Errorf("the cassock is not long: %d white rows of 14 down to the shoes", hem)
	}
	if gold < 10 {
		t.Errorf("no golden stripe down the cassock: %d rows", gold)
	}
	// the mitre: white with a golden cross above the eyes, higher than the old hat line
	mitreGold, mitreWhite := 0, 0
	for y := 2; y < 12; y++ {
		for x := charCX - 3; x <= charCX+3; x++ {
			c := ptfAt(b, x, y)
			if ptfGoldish(c) {
				mitreGold++
			} else if ptfWhitish(c) {
				mitreWhite++
			}
		}
	}
	if mitreGold < 4 || mitreWhite < 8 {
		t.Errorf("mitre: %d gold and %d white pixels", mitreGold, mitreWhite)
	}
	for _, d := range vipRegistry() {
		if d.id == "pope" && (d.es != "SU SANTIDAD PÍO XCIX" || d.bonus != 2000 || d.hp != 1 || d.behaviour != vbTip) {
			t.Errorf("the Pope changed his texts or stats: %+v", d)
		}
	}
}

// Michael's stage outfit: white socks above black shoes, a white glove, and the dance slides
// backwards (the moonwalk replaces the dances of his style).
func TestPtfMichaelSocksGloveAndDance(t *testing.T) {
	sp := charimpASpec(t, "michael_jackson")
	b := renderChar(&sp, vFront, poseFor(AnimIdle, 0))
	socks := 0
	for y := 36; y < 40; y++ { // above the shoes (rows 40-41)
		for x := 0; x < charW; x++ {
			if c := ptfAt(b, x, y); c.A > 0 && c.R > 160 && c.R == c.G && c.G == c.B {
				socks++
			}
		}
	}
	if socks < 4 {
		t.Errorf("no white socks: %d pixels", socks)
	}
	if sp.Shoes.R > 60 || sp.Hat != HatFedora || sp.Name != "michael_jackson" {
		t.Errorf("spec: %+v", sp)
	}
	for _, a := range []int{AnimDance0, AnimDance1, AnimDance2, AnimDance3} {
		if got, _ := gaitAnim(gaitSlugs["michael_jackson"], a, 0.3); got != AnimMoon {
			t.Errorf("dance %d: %d, want the moonwalk", a, got)
		}
	}
}

// The three famous guests always draw the same pictures (so the exported ones are checked by
// TestCharimpReexportedPictures) and keep their slugs.
func TestPtfFixedAndExported(t *testing.T) {
	for _, b := range builtinChars() {
		for _, s := range ptfSlugs {
			if b.name == s && (b.cat != "special_people" || !charimpFixed(b)) {
				t.Errorf("%s: cat %s fixed %v", s, b.cat, charimpFixed(b))
			}
		}
	}
	for _, s := range ptfSlugs {
		if presetByName(s) == nil {
			t.Errorf("%s: no preset", s)
		}
	}
}

// PTF_OLD=dir-with-the-old-folders PTF_SHEET=out go test -run TestPtfBeforeAfter writes an enlarged
// before -> after sheet: the Pope (old spec / new spec) and the three exported famous guests.
func TestPtfBeforeAfter(t *testing.T) {
	oldDir, out := os.Getenv("PTF_OLD"), os.Getenv("PTF_SHEET")
	if oldDir == "" || out == "" {
		t.Skip("set PTF_OLD and PTF_SHEET")
	}
	const z = 4
	cellW, cellH := 7*charW*z+16, charH*z+6
	big := image.NewRGBA(image.Rect(0, 0, cellW, 4*cellH))
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
	pope := ptfPopeSpec(t)
	old := vipBase()
	old.Skin = hx(0xf0d0b8)
	old.Hair2, old.Glasses, old.Hat, old.HatCol = HairBald, GlassRound, HatWizard, hx(0xf4f0e0)
	old.Top, old.Shirt, old.Shirt2, old.Accent, old.Pants = TopRobe, hx(0xf4f2ea), hx(0xe8c040), hx(0xe8c040), hx(0xf4f2ea)
	old.Carry, old.CarryCol = HoldSaber, hx(0xf0c030)
	for i, vw := range []int{vFront, vSide, vBack} {
		put(renderChar(&old, vw, poseFor(AnimIdle, 0)), i*charW*z, 0)
		put(renderChar(&pope, vw, poseFor(AnimIdle, 0)), (4+i)*charW*z, 0)
	}
	for si, slug := range ptfSlugs {
		oy := (si + 1) * cellH
		a, err := loadCharAsset(filepath.Join(oldDir, slug))
		if err != nil {
			t.Fatal(err)
		}
		n, err := loadCharAsset(filepath.Join(spritesDir("assets", "special_people"), slug))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			put(a.Views[i].outlinedSel(0.42), i*charW*z, oy)
			put(n.Views[i].outlinedSel(0.42), (4+i)*charW*z, oy)
		}
	}
	for r := 0; r < 4; r++ { // an arrow between before and after
		ay := r*cellH + charH*z/2
		for x := 3*charW*z + 10; x < 4*charW*z-10; x++ {
			big.SetRGBA(x, ay, color.RGBA{240, 240, 240, 255})
		}
		for k := 0; k < 6; k++ {
			big.SetRGBA(4*charW*z-10-k, ay-k, color.RGBA{240, 240, 240, 255})
			big.SetRGBA(4*charW*z-10-k, ay+k, color.RGBA{240, 240, 240, 255})
		}
	}
	f, err := os.Create(filepath.Join(out, "ptf_antes_despues.png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}
