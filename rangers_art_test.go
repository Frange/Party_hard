package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// rangerSlugs: the six Power Rangers of the 5-star squad (rangers_art.go), in the unit order.
var rangerSlugs = []string{"ranger_red", "ranger_blue", "ranger_yellow", "ranger_green", "ranger_pink", "ranger_black"}

func rangerSpecT(t *testing.T, slug string) CharSpec {
	for _, s := range unitSpecs().rangers {
		if s.Name == slug {
			return s
		}
	}
	t.Fatalf("%s: not in the squad", slug)
	return CharSpec{}
}

// RANGER_SHEET=out.png [RANGER_OLD=dir-with-old-folders] go test -run TestRangerArtSheet writes an
// enlarged sheet: per ranger the old pictures (front, side, back) and the new drawing (front,
// side, back, walking front and side, dancing front).
func TestRangerArtSheet(t *testing.T) {
	out := os.Getenv("RANGER_SHEET")
	if out == "" {
		t.Skip("set RANGER_SHEET")
	}
	oldDir := os.Getenv("RANGER_OLD")
	const z = 5
	cols := 11
	cellH := charH*z + 8
	big := image.NewRGBA(image.Rect(0, 0, cols*charW*z+40, len(rangerSlugs)*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x6a, 0x6a, 0x74, 255
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
	for r, slug := range rangerSlugs {
		oy := r*cellH + 4
		if oldDir != "" {
			if old, err := loadCharAsset(filepath.Join(oldDir, slug)); err == nil {
				for i := 0; i < 3; i++ {
					put(old.Views[i].outlinedSel(0.42), i*charW*z, oy)
				}
			}
		}
		sp := rangerSpecT(t, slug)
		for i, vw := range []int{vFront, vSide, vBack} {
			put(renderChar(&sp, vw, ownPose(AnimIdle, 0)), 40+(3+i)*charW*z, oy)
		}
		for i, c := range [][3]int{{vFront, AnimWalk, 1}, {vFront, AnimWalk, 3}, {vSide, AnimWalk, 1}, {vSide, AnimRun, 2}, {vFront, AnimDance0, 1}} {
			put(renderChar(&sp, c[0], ownPose(c[1], c[2])), 40+(6+i)*charW*z, oy)
		}
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}

// Every view and animation (gaits included) draws inside the sprite box (1px margin).
func TestRangerArtDrawsInBox(t *testing.T) {
	for _, slug := range rangerSlugs {
		sp := rangerSpecT(t, slug)
		if !isRangerArt(&sp) {
			t.Fatalf("%s: not drawn by rangers_art.go", slug)
		}
		for vw := 0; vw < 3; vw++ {
			for a := 0; a < animGaitEnd; a++ {
				for f := 0; f < animFrameCount(a); f++ {
					pp := ownPose(a, f)
					pp.Hold = HoldNone
					minX, minY, maxX, maxY := bbox(genChar(&sp, vw, pp))
					if maxX < 0 {
						t.Fatalf("%s: empty frame vw=%d anim=%d f=%d", slug, vw, a, f)
					}
					if minX < 1 || minY < 1 || maxX > charW-2 || maxY > charH-1 {
						t.Errorf("%s vw=%d anim=%d f=%d outside the box: %d,%d-%d,%d", slug, vw, a, f, minX, minY, maxX, maxY)
					}
				}
			}
		}
	}
}

// Walking, running and dancing change the frames in the three views, and the rig gets every part.
func TestRangerArtAnimatesAndHasParts(t *testing.T) {
	for _, slug := range rangerSlugs {
		sp := rangerSpecT(t, slug)
		for vw := 0; vw < 3; vw++ {
			for _, a := range []int{AnimWalk, AnimRun, AnimDance0} {
				if string(genChar(&sp, vw, ownPose(a, 0)).p) == string(genChar(&sp, vw, ownPose(a, 1)).p) {
					t.Errorf("%s vw %d anim %d: frames 0 and 1 are the same", slug, vw, a)
				}
			}
		}
		ca := charAssetFromSpec(sp, slug, "police", "ranger")
		for vi, v := range ca.Views {
			seen := map[uint8]bool{}
			for _, id := range v.pid {
				seen[id] = true
			}
			for id := uint8(1); id <= 6; id++ {
				if !seen[id] && !(vi == 1 && (id == 3 || id == 5)) {
					t.Errorf("%s view %d: part %d missing", slug, vi, id)
				}
			}
		}
	}
}

// The look: suit colour, black visor, white gloves/boots/belt, golden buckle; green has its golden
// shield, pink and yellow their skirt (wider at the hips than the others); the back shows no
// visor; the six are clearly different from each other.
func TestRangerArtLookAndSixDistinct(t *testing.T) {
	count := func(b *pixBuf, c color.RGBA) int {
		n := 0
		for i := 0; i+3 < len(b.p); i += 4 {
			if b.p[i] == c.R && b.p[i+1] == c.G && b.p[i+2] == c.B && b.p[i+3] != 0 {
				n++
			}
		}
		return n
	}
	fronts := map[string]*pixBuf{}
	for _, slug := range rangerSlugs {
		sp := rangerSpecT(t, slug)
		f := genChar(&sp, vFront, ownPose(AnimIdle, 0))
		fronts[slug] = f
		if n := count(f, sp.Shirt); n < 60 {
			t.Errorf("%s: only %d pixels of the suit colour", slug, n)
		}
		for name, c := range map[string]color.RGBA{"visor": rgVisor, "white": rgWhite, "gold": rgGold, "rim": rgRim} {
			if count(f, c) == 0 {
				t.Errorf("%s: front misses %s", slug, name)
			}
		}
		if count(f, rgVisor) < 14 {
			t.Errorf("%s: the visor is too small (%d px)", slug, count(f, rgVisor))
		}
		if bk := genChar(&sp, vBack, ownPose(AnimIdle, 0)); count(bk, rgVisor) != 0 {
			t.Errorf("%s: the back shows the visor", slug)
		}
		if sd := genChar(&sp, vSide, ownPose(AnimIdle, 0)); count(sd, rgVisor) == 0 {
			t.Errorf("%s: the side hides the visor", slug)
		}
	}
	if count(fronts["ranger_green"], rgGold) < 3*count(fronts["ranger_red"], rgGold) {
		t.Errorf("green: no golden shield")
	}
	skirtAt := func(slug string, x, y int) bool { // the hem of a skirt: light or dark suit colour
		sp := rangerSpecT(t, slug)
		f := fronts[slug]
		i := (y*f.w + x) * 4
		c := color.RGBA{f.p[i], f.p[i+1], f.p[i+2], 255}
		return f.p[i+3] != 0 && (c == mixc(sp.Shirt, rgWhite, 0.3) || c == shade(sp.Shirt, 0.72))
	}
	hem := 29 + 2
	for _, x := range []int{7, 16} {
		for _, s := range []string{"ranger_pink", "ranger_yellow"} {
			if !skirtAt(s, x, hem) {
				t.Errorf("%s: no skirt at x=%d", s, x)
			}
		}
		for _, s := range []string{"ranger_red", "ranger_blue", "ranger_green", "ranger_black"} {
			if skirtAt(s, x, hem) {
				t.Errorf("%s: wears a skirt at x=%d", s, x)
			}
		}
	}
	for i, a := range rangerSlugs {
		for _, c := range rangerSlugs[i+1:] {
			diff := 0
			for j := 0; j < len(fronts[a].p); j += 4 {
				if fronts[a].p[j] != fronts[c].p[j] || fronts[a].p[j+1] != fronts[c].p[j+1] || fronts[a].p[j+2] != fronts[c].p[j+2] {
					diff++
				}
			}
			if diff < 100 {
				t.Errorf("%s and %s look alike: %d different pixels", a, c, diff)
			}
		}
	}
}

// A squad of five always has five different suits, and its halos follow the suit worn.
func TestRangerSquadColours(t *testing.T) {
	g := &Game{as: &Assets{}}
	for _, s := range unitSpecs().rangers {
		s := s
		g.as.rangerCS = append(g.as.rangerCS, newCharSet(&s))
	}
	if len(g.as.rangerCS) != 6 {
		t.Fatalf("squad has %d colours, want 6", len(g.as.rangerCS))
	}
	for tm := 0.0; tm < 2; tm += 0.07 {
		seen := map[string]bool{}
		halos := map[color.RGBA]bool{}
		for k := 0; k < 5; k++ {
			c := &Cop{kind: unitRangers, alt: k + rangerSquadOff(unitRangers, tm)}
			cs := g.copCSFor(c)
			seen[cs.Spec.Name] = true
			halos[g.copHaloCol(c)] = true
		}
		if len(seen) != 5 || len(halos) != 5 {
			t.Fatalf("t=%.2f: %d suits and %d halos in a squad of five", tm, len(seen), len(halos))
		}
	}
	if rangerSquadOff(unitPolice, 1.3) != 0 {
		t.Error("the offset touches the police")
	}
}

// RANGER_EXPORT=assets-root go test -run TestRangerArtReexport writes the pictures of the six
// rangers into police/ranger_*: meta.json is kept (the new black ranger gets one like its mates).
func TestRangerArtReexport(t *testing.T) {
	root := os.Getenv("RANGER_EXPORT")
	if root == "" {
		t.Skip("set RANGER_EXPORT to an assets folder")
	}
	for _, sp := range unitSpecs().rangers {
		ca := charAssetFromSpec(sp, sp.Name, "police", "ranger")
		if old, err := loadCharAsset(filepath.Join(spritesDir(root, "police"), sp.Name)); err == nil {
			carry := ca.Meta.Carry
			ca.Meta = old.Meta
			ca.Meta.Carry = carry
		} else {
			ca.Meta.Role, ca.Meta.Edited, ca.Meta.Gender = "ranger", true, "m"
			ca.Meta.DisplayName = "Ranger Black"
		}
		if err := saveCharAsset(root, ca); err != nil {
			t.Fatal(err)
		}
	}
}

// The pictures in assets/sprites/police/ranger_* are the ones the code draws, meta.json keeps
// name, category, role and the "edited" flag the game needs to use them, and the rig walks.
func TestRangerArtExportedPictures(t *testing.T) {
	for _, sp := range unitSpecs().rangers {
		dir := filepath.Join(spritesDir("assets", "police"), sp.Name)
		ca, err := loadCharAsset(dir)
		if err != nil {
			t.Fatalf("%s: %v", sp.Name, err)
		}
		if !charimpSame(ca, charAssetFromSpec(sp, sp.Name, "police", "ranger")) {
			t.Errorf("%s: the exported pictures are not the ones the code draws", sp.Name)
		}
		var raw map[string]any
		data, _ := os.ReadFile(filepath.Join(dir, "meta.json"))
		if json.Unmarshal(data, &raw) != nil || raw["name"] != sp.Name || raw["category"] != "police" || raw["role"] != "ranger" || raw["edited"] != true {
			t.Errorf("%s: meta.json broken: %s", sp.Name, data)
		}
		rig := buildRig(ca)
		for vw := 0; vw < 3; vw++ {
			f0 := rig.render(vw, poseFor(AnimWalk, 0))
			if n, _ := charimpAPix(f0); n < 60 {
				t.Errorf("%s rig vw %d: only %d pixels", sp.Name, vw, n)
			}
			if string(rig.render(vw, poseFor(AnimWalk, 1)).p) == string(f0.p) {
				t.Errorf("%s rig vw %d: walking does not change the frames", sp.Name, vw)
			}
		}
	}
}
