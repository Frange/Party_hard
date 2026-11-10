package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// heroSlugs: the six superheroes with their own drawing (hero_art.go).
var heroSlugs = []string{"superman", "batman", "hulk", "capitan_america", "lobezno", "deadpool"}

func heroSpec(t *testing.T, slug string) CharSpec {
	f := presetByName(slug)
	if f == nil {
		t.Fatalf("%s: no preset", slug)
	}
	return f(newRand(5))
}

// HERO_SHEET=out.png [HERO_OLD=dir-with-old-folders] go test -run TestHeroArtSheet writes an enlarged
// sheet: per hero the old exported pictures (front, side, back) and the new drawing (front, side,
// back, two walking frames front and side).
func TestHeroArtSheet(t *testing.T) {
	out := os.Getenv("HERO_SHEET")
	if out == "" {
		t.Skip("set HERO_SHEET")
	}
	oldDir := os.Getenv("HERO_OLD")
	const z = 5
	cols := 10
	cellH := charH*z + 8
	big := image.NewRGBA(image.Rect(0, 0, cols*charW*z+40, len(heroSlugs)*cellH))
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
	for r, slug := range heroSlugs {
		oy := r*cellH + 4
		if oldDir != "" {
			if old, err := loadCharAsset(filepath.Join(oldDir, slug)); err == nil {
				for i := 0; i < 3; i++ {
					put(old.Views[i].outlinedSel(0.42), i*charW*z, oy)
				}
			}
		}
		sp := heroSpec(t, slug)
		for i, vw := range []int{vFront, vSide, vBack} {
			put(renderChar(&sp, vw, ownPose(AnimIdle, 0)), 40+(3+i)*charW*z, oy)
		}
		for i, c := range [][2]int{{vFront, 1}, {vFront, 3}, {vSide, 1}, {vSide, 3}} {
			put(renderChar(&sp, c[0], ownPose(AnimWalk, c[1])), 40+(6+i)*charW*z, oy)
		}
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}

// Every view and animation (gaits included) draws inside the sprite box (1px margin), with the
// held object left out (it may reach out of the box, as with everybody).
func TestHeroArtDrawsInBox(t *testing.T) {
	for _, slug := range heroSlugs {
		sp := heroSpec(t, slug)
		if !isHeroArt(&sp) {
			t.Fatalf("%s: not drawn by hero_art.go (name %q)", slug, sp.Name)
		}
		for vw := 0; vw < 3; vw++ {
			for a := 0; a < animGaitEnd; a++ {
				for f := 0; f < animFrameCount(a); f++ {
					pp := ownPose(a, f)
					pp.Hold = HoldNone
					sp2 := sp
					sp2.Weapon, sp2.Carry = 0, 0
					minX, minY, maxX, maxY := bbox(genChar(&sp2, vw, pp))
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
func TestHeroArtAnimatesAndHasParts(t *testing.T) {
	for _, slug := range heroSlugs {
		sp := heroSpec(t, slug)
		for vw := 0; vw < 3; vw++ {
			for _, a := range []int{AnimWalk, AnimRun, AnimDance0} {
				if string(genChar(&sp, vw, ownPose(a, 0)).p) == string(genChar(&sp, vw, ownPose(a, 1)).p) {
					t.Errorf("%s vw %d anim %d: frames 0 and 1 are the same", slug, vw, a)
				}
			}
		}
		ca := charAssetFromSpec(sp, slug, "special_people", "")
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

// Each hero shows its own colours, the six are clearly different from each other, and Hulk is
// the widest of them.
func TestHeroArtLookAndHulkWidest(t *testing.T) {
	has := func(b *pixBuf, c color.RGBA) bool {
		for i := 0; i+3 < len(b.p); i += 4 {
			if b.p[i] == c.R && b.p[i+1] == c.G && b.p[i+2] == c.B && b.p[i+3] != 0 {
				return true
			}
		}
		return false
	}
	want := map[string][]color.RGBA{
		"superman":        {hrSupBlue, hrRed, hrYellow, hrBlack},
		"batman":          {hrBatGrey, hrBlack, hrYellow, hrWhite, hrSkin},
		"hulk":            {hrHulk, hrHulkD, hrPurple, hrHulkHr},
		"capitan_america": {hrCapBlue, hrWhite, hrRed, hrYellow, hrPink},
		"lobezno":         {hrLogY, hrLogB, hrGold, hrSilver},
		"deadpool":        {hrDPRed, hrBlack, hrWhite, hrHilt},
	}
	fronts := map[string]*pixBuf{}
	width := map[string]int{}
	for _, slug := range heroSlugs {
		sp := heroSpec(t, slug)
		f := genChar(&sp, vFront, ownPose(AnimIdle, 0))
		fronts[slug] = f
		for _, c := range want[slug] {
			if !has(f, c) {
				t.Errorf("%s: front misses colour %v", slug, c)
			}
		}
		minX, _, maxX, _ := bbox(f)
		width[slug] = maxX - minX + 1
		if bk := genChar(&sp, vBack, ownPose(AnimIdle, 0)); has(bk, hrSkin) && slug != "superman" {
			t.Errorf("%s: the back shows the face", slug)
		}
	}
	for i, a := range heroSlugs {
		for _, c := range heroSlugs[i+1:] {
			diff := 0
			for j := 0; j < len(fronts[a].p); j += 4 {
				if fronts[a].p[j] != fronts[c].p[j] || fronts[a].p[j+1] != fronts[c].p[j+1] || fronts[a].p[j+2] != fronts[c].p[j+2] {
					diff++
				}
			}
			if diff < 120 {
				t.Errorf("%s and %s look alike: %d different pixels", a, c, diff)
			}
		}
		if a != "hulk" && width[a] >= width["hulk"] {
			t.Errorf("%s (%d px) is as wide as Hulk (%d px)", a, width[a], width["hulk"])
		}
	}
	if width["hulk"] < 20 {
		t.Errorf("Hulk only %d px wide", width["hulk"])
	}
}
