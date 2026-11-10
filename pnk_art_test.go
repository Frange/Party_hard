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

// pnkSlugs: the members of Sum 41, Blink-182 and The Offspring redrawn in pnk_art.go.
var pnkSlugs = []string{
	"deryck", "dave_baksh", "jason_mc", "frank_zummo",
	"mark_hoppus", "tom_delonge", "travis_barker",
	"off_cantante", "off_guitarra", "off_bajo", "off_bateria",
}

// PNK_SHEET=dir [PNK_TAG=antes] [PNK_RIG=1] go test -run TestPnkSheet writes an enlarged sheet of
// the eleven (front, side, back idle, a walk frame and a dance frame), grouped by band. With PNK_RIG
// the pictures come from the exported files animated by the rig, as the game draws them.
func TestPnkSheet(t *testing.T) {
	dir := os.Getenv("PNK_SHEET")
	if dir == "" {
		t.Skip("set PNK_SHEET")
	}
	const z = 5
	cellW, cellH := 5*charW*z+10, charH*z+6
	cols := 2
	rows := (len(pnkSlugs) + cols - 1) / cols
	big := image.NewRGBA(image.Rect(0, 0, cols*cellW, rows*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x70, 0x70, 0x78, 255
	}
	for si, slug := range pnkSlugs {
		sp := charimpASpec(t, slug)
		var rig *Rig
		if os.Getenv("PNK_RIG") != "" {
			ca, err := loadCharAsset(filepath.Join(spritesDir("assets", "special_people"), slug))
			if err != nil {
				t.Fatal(err)
			}
			rig = buildRig(ca)
		}
		ox, oy := (si%cols)*cellW, (si/cols)*cellH
		for vi, vw := range []int{vFront, vSide, vBack, vFront, vFront} {
			pp := poseFor(AnimIdle, 0)
			switch vi {
			case 3:
				pp = poseFor(AnimWalk, 1)
				pp.Anim, pp.F = AnimWalk, 1
			case 4:
				pp = poseFor(AnimDance1, 1)
				pp.Anim, pp.F = AnimDance1, 1
			}
			b := renderChar(&sp, vw, pp)
			if rig != nil {
				b = rig.render(vw, pp)
			}
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							big.SetRGBA(ox+(vi*charW+x)*z+dx, oy+y*z+dy, c)
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("pnk_%s.png", os.Getenv("PNK_TAG"))))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}

// The eleven draw in the three views inside their box, the details really show (against the
// same body without the hooks), the three views differ and walking changes the frames.
func TestPnkDrawsInsideAndWalks(t *testing.T) {
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic, AnimHop}
	for _, slug := range pnkSlugs {
		sp := charimpASpec(t, slug)
		if pnkKind(&sp) == pnkNone {
			t.Fatalf("%q: no details registered", slug)
		}
		twin := sp
		twin.Name = "twin"
		var idle [3]string
		for vw := 0; vw < 3; vw++ {
			idle[vw] = string(renderChar(&sp, vw, poseFor(AnimIdle, 0)).p)
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
						t.Errorf("%q vw %d anim %d f %d: %d pixels on the edge (plain %d)", slug, vw, a, f, edge, tEdge)
					}
					if n < 60 || n < tn*9/10 {
						t.Errorf("%q vw %d anim %d f %d: only %d pixels (plain %d)", slug, vw, a, f, n, tn)
					}
					if string(b.p) != string(tb.p) {
						differs = true
					}
				}
			}
			if !differs {
				t.Errorf("%q vw %d: the details never show", slug, vw)
			}
			w0 := charimpAWalkFrame(&sp, vw, 0)
			changed := false
			for f := 1; f < animFrameCount(AnimWalk); f++ {
				if string(charimpAWalkFrame(&sp, vw, f)) != string(w0) {
					changed = true
				}
			}
			if !changed {
				t.Errorf("%q vw %d: walking does not change the frames", slug, vw)
			}
		}
		if idle[0] == idle[1] || idle[0] == idle[2] || idle[1] == idle[2] {
			t.Errorf("%q: two views are the same picture", slug)
		}
	}
}

// pnkCount: pixels of the idle picture in rows [y0, y1) that match.
func pnkCount(t *testing.T, slug string, vw, y0, y1 int, match func(c color.RGBA) bool) int {
	sp := charimpASpec(t, slug)
	b := renderChar(&sp, vw, poseFor(AnimIdle, 0))
	n := 0
	for y := maxI(0, y0); y < minI(b.h, y1); y++ {
		for x := 0; x < b.w; x++ {
			if c := ptfAt(b, x, y); c.A > 0 && match(c) {
				n++
			}
		}
	}
	return n
}

// pnkHeadTop: the first row with a skin pixel (the forehead) in the front idle picture.
func pnkHeadTop(t *testing.T, slug string) int {
	sp := charimpASpec(t, slug)
	b := renderChar(&sp, vFront, poseFor(AnimIdle, 0))
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if ptfAt(b, x, y) == sp.Skin {
				return y
			}
		}
	}
	t.Fatalf("%s: no skin", slug)
	return 0
}

func pnkNear(a, b color.RGBA, d int) bool {
	ab := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return ab(a.R, b.R) <= d && ab(a.G, b.G) <= d && ab(a.B, b.B) <= d
}

// The key colours of each look: the yellow spikes and the orange crest stand high above the
// forehead, the mohawk is tall and narrow, the check shirt has both colours, the flaming skull
// shows its yellow ring and red field, there are tattoos, drumsticks, studs and dark glasses.
func TestPnkKeyColours(t *testing.T) {
	yellow := func(c color.RGBA) bool { return c.R > 0xc0 && c.G > 0xa0 && c.B < 0x60 }
	orange := func(c color.RGBA) bool { return c.R > 0xa0 && c.G > 0x30 && c.G < 0x90 && c.B < 0x50 }
	dark := func(c color.RGBA) bool { return c.R < 0x50 && c.G < 0x50 && c.B < 0x60 }
	for _, tc := range []struct {
		slug  string
		match func(c color.RGBA) bool
		above int // how far above the forehead the hair reaches
		min   int
	}{{"jason_mc", yellow, 4, 16}, {"frank_zummo", orange, 5, 16}, {"travis_barker", dark, 5, 6}, {"off_guitarra", dark, 3, 8}} {
		top := pnkHeadTop(t, tc.slug)
		if n := pnkCount(t, tc.slug, vFront, 0, top-tc.above+1, tc.match); n == 0 {
			t.Errorf("%s: the hair does not reach %d rows above the forehead", tc.slug, tc.above)
		}
		if n := pnkCount(t, tc.slug, vFront, 0, top, tc.match); n < tc.min {
			t.Errorf("%s: only %d hair pixels above the forehead (want %d)", tc.slug, n, tc.min)
		}
	}
	// the mohawk is narrow from the front: nothing dark 3+ columns from the middle above the head
	sp := charimpASpec(t, "travis_barker")
	b := renderChar(&sp, vFront, poseFor(AnimIdle, 0))
	top := pnkHeadTop(t, "travis_barker")
	for y := 0; y < top-1; y++ {
		for x := 0; x < b.w; x++ {
			if c := ptfAt(b, x, y); c.A > 0 && (x < charCX-3 || x > charCX+3) {
				t.Errorf("travis_barker: the crest is wide at (%d,%d)", x, y)
			}
		}
	}
	// drumsticks in the hands of both drummers, beer cans for Sum 41
	// (the tan sticks against the same body without them: renderChar tones the edge pixels)
	stick := func(c color.RGBA) bool { return c.R > 0xa8 && c.G > 0x90 && c.B > 0x58 && c.B < 0xc8 }
	for _, s := range []string{"travis_barker", "off_bateria"} {
		sp := charimpASpec(t, s)
		twin := sp
		twin.Name = "twin"
		for vw := 0; vw < 3; vw++ {
			b, tb := renderChar(&sp, vw, poseFor(AnimIdle, 0)), renderChar(&twin, vw, poseFor(AnimIdle, 0))
			n := 0
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					if c := ptfAt(b, x, y); c.A > 0 && stick(c) && ptfAt(tb, x, y) != c {
						n++
					}
				}
			}
			if want := map[bool]int{true: 4, false: 6}[vw == vSide]; n < want { // in profile the far one hides
				t.Errorf("%s vw %d: no drumsticks (%d pixels)", s, vw, n)
			}
		}
	}
	for _, s := range []string{"deryck", "dave_baksh", "jason_mc", "frank_zummo"} {
		if sp := charimpASpec(t, s); sp.Carry != HoldBeer {
			t.Errorf("%s: carries %d, want a beer can", s, sp.Carry)
		}
	}
	// tattoos on the arms
	ink := func(c color.RGBA) bool { return pnkNear(c, pnkInk, 12) }
	for _, s := range []string{"tom_delonge", "travis_barker"} {
		if n := pnkCount(t, s, vFront, 0, charH, ink); n < 4 {
			t.Errorf("%s: only %d tattoo pixels", s, n)
		}
	}
	// the flaming skull: the yellow ring and the red field on the Offspring's shirts
	ring := func(c color.RGBA) bool { return pnkNear(c, pnkLogoY, 6) }
	red := func(c color.RGBA) bool { return pnkNear(c, pnkLogoR, 6) }
	for _, tc := range []struct {
		slug         string
		vw, ring, rd int
	}{{"off_bajo", vFront, 6, 2}, {"off_bajo", vBack, 6, 2}, {"off_bateria", vFront, 6, 2}, {"off_guitarra", vBack, 6, 2}, {"off_cantante", vFront, 0, 3}} {
		if n := pnkCount(t, tc.slug, tc.vw, 0, charH, ring); n < tc.ring {
			t.Errorf("%s vw %d: %d ring pixels of the skull (want %d)", tc.slug, tc.vw, n, tc.ring)
		}
		if n := pnkCount(t, tc.slug, tc.vw, 0, charH, red); n < tc.rd {
			t.Errorf("%s vw %d: %d red pixels of the skull (want %d)", tc.slug, tc.vw, n, tc.rd)
		}
	}
	// the check shirt: light and dark squares on the torso and the sleeves
	light := func(c color.RGBA) bool { return c.R > 0xb0 && c.R == c.G && c.G == c.B }
	if n := pnkCount(t, "off_guitarra", vFront, 16, 30, light); n < 10 {
		t.Errorf("off_guitarra: only %d light squares on the shirt", n)
	}
	if sp := charimpASpec(t, "off_guitarra"); sp.Glasses != GlassShades {
		t.Errorf("off_guitarra: glasses %d, want dark glasses", sp.Glasses)
	}
	// studded belts
	stud := func(c color.RGBA) bool { return pnkNear(c, pnkStud, 4) }
	for _, s := range []string{"deryck", "jason_mc", "frank_zummo"} {
		if n := pnkCount(t, s, vFront, 0, charH, stud); n < 2 {
			t.Errorf("%s: no studs on the belt", s)
		}
	}
	// the blond of the Offspring singer and the white cap of the Sum 41 singer
	if sp := charimpASpec(t, "off_cantante"); sp.Hair.R < 0xd0 || sp.Hair.B > 0xa0 || sp.Top != TopTee {
		t.Errorf("off_cantante: %+v", sp)
	}
	if sp := charimpASpec(t, "deryck"); sp.Hat != HatCap || sp.HatCol.R < 0xe0 {
		t.Errorf("deryck: no white cap")
	}
}

// The eleven always draw the same pictures (so the exported ones are checked by
// TestCharimpReexportedPictures), keep their slugs, are listed in charimpRenewed, dance the pogo
// when they are punks, and are named after the band and the role, not after anyone.
func TestPnkFixedExportedAndNamed(t *testing.T) {
	renewed := map[string]bool{}
	for _, s := range charimpRenewed {
		renewed[s] = true
	}
	found := map[string]bool{}
	for _, b := range builtinChars() {
		for _, s := range pnkSlugs {
			if b.name == s {
				found[s] = true
				if b.cat != "special_people" || !charimpFixed(b) {
					t.Errorf("%s: cat %s fixed %v", s, b.cat, charimpFixed(b))
				}
			}
		}
	}
	for _, s := range pnkSlugs {
		if !found[s] || presetByName(s) == nil {
			t.Errorf("%s: not a built-in famous guest", s)
		}
		if !renewed[s] {
			t.Errorf("%s: not in charimpRenewed", s)
		}
		name := displayNameOverrides[s]
		band := map[byte]string{'d': "Sum 41", 'j': "Sum 41", 'f': "Sum 41", 'm': "Blink-182", 't': "Blink-182", 'o': "The Offspring"}[s[0]]
		if m, named := bandRealNames[s]; named { // a musician shown by his real name (bands_names_test.go)
			if name != m.name || bandOfMember(s) != band {
				t.Errorf("%s: display name %q band %q, want %q of %q", s, name, bandOfMember(s), m.name, band)
			}
		} else if !strings.HasPrefix(name, band+" · ") {
			t.Errorf("%s: display name %q, want %q + role", s, name, band)
		}
		if _, ok := charDescs[s]; !ok {
			t.Errorf("%s: no description", s)
		}
	}
	for _, s := range []string{"jason_mc", "frank_zummo", "travis_barker", "off_guitarra"} {
		if got, _ := gaitAnim(gaitSlugs[s], AnimDance2, 0.3); got != AnimHop {
			t.Errorf("%s: dances %d, want the pogo", s, got)
		}
		if got, _ := gaitAnim(gaitSlugs[s], AnimWalk, 0.3); got != AnimWalk {
			t.Errorf("%s: the pogo changed the walk (%d)", s, got)
		}
	}
	for _, bd := range bandList {
		switch bd.slug {
		case "sum41", "blink182", "offspring":
			for _, m := range bd.members {
				if pnkSlugKind[m.slug] == pnkNone {
					t.Errorf("%s: member %s not redrawn", bd.slug, m.slug)
				}
			}
		}
	}
}
