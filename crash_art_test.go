package main

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

func crashSpec() CharSpec { return presetByName("crash")(rand.New(rand.NewSource(1))) }

// Every view and animation (gaits included) draws something inside the sprite box (1px margin).
func TestCrashArtDrawsInBox(t *testing.T) {
	sp := crashSpec()
	if !isCrash(&sp) {
		t.Fatalf("preset name %q", sp.Name)
	}
	for vw := 0; vw < 3; vw++ {
		for a := 0; a < animGaitEnd; a++ {
			for f := 0; f < animFrameCount(a); f++ {
				pp := poseFor(a, f)
				pp.Anim, pp.F = a, f
				pp.Hold = HoldNone // held objects may reach out of the box, as with everybody
				minX, minY, maxX, maxY := bbox(genChar(&sp, vw, pp))
				if maxX < 0 {
					t.Fatalf("empty frame vw=%d anim=%d f=%d", vw, a, f)
				}
				if minX < 1 || minY < 1 || maxX > charW-2 || maxY > charH-1 {
					t.Errorf("vw=%d anim=%d f=%d outside the box: %d,%d-%d,%d", vw, a, f, minX, minY, maxX, maxY)
				}
			}
		}
	}
}

// His own look: orange fur, green eyes and a dark nose in front, blue jeans, lilac trainers with
// white soles, brown gloves; the snout sticks out forward in profile; the back shows no face.
func TestCrashArtLook(t *testing.T) {
	sp := crashSpec()
	pp := poseFor(AnimIdle, 0)
	pp.Anim = AnimIdle
	has := func(b *pixBuf, c color.RGBA) bool {
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if b.get(x, y) == c {
					return true
				}
			}
		}
		return false
	}
	front, side, back := genChar(&sp, vFront, pp), genChar(&sp, vSide, pp), genChar(&sp, vBack, pp)
	for _, c := range []color.RGBA{crFur, crCream, crIris, crNose, crBrow, crJeans, crBand, crShoe, crSole, crGlove, crCrest, crEarIn} {
		if !has(front, c) {
			t.Errorf("front misses colour %v", c)
		}
	}
	for _, c := range []color.RGBA{crIris, crNose, crCream} {
		if has(back, c) {
			t.Errorf("back shows face colour %v", c)
		}
	}
	if !has(back, crCrest) || !has(back, crShoe) {
		t.Error("back misses the crest or the trainers")
	}
	// the nose is at the front tip of the snout, beyond the skull
	nx := -1
	for y := 0; y < 20; y++ {
		for x := 0; x < charW; x++ {
			if side.get(x, y) == crNose && x > nx {
				nx = x
			}
		}
	}
	if nx < 18 {
		t.Errorf("profile snout too short: nose at x=%d", nx)
	}
	// shorter than an ordinary guest's body but wider at the shoulders: torso row 21 spans 12 px
	n := 0
	for x := 0; x < charW; x++ {
		if front.get(x, 21).A != 0 {
			n++
		}
	}
	if n < 12 {
		t.Errorf("shoulders only %d px wide", n)
	}
}

// Walking, running and dancing change the frames in the three views, and the parts for the
// rig (head, torso, both arms and legs) are all tagged.
func TestCrashArtAnimatesAndHasParts(t *testing.T) {
	sp := crashSpec()
	for vw := 0; vw < 3; vw++ {
		for _, a := range []int{AnimWalk, AnimRun, AnimDance0, AnimHop} {
			p0, p1 := poseFor(a, 0), poseFor(a, 1)
			p0.Anim, p0.F, p1.Anim, p1.F = a, 0, a, 1
			if string(genChar(&sp, vw, p0).p) == string(genChar(&sp, vw, p1).p) {
				t.Errorf("vw %d anim %d: frames 0 and 1 are the same", vw, a)
			}
		}
	}
	ca := charAssetFromSpec(sp, "crash", "special_people", "")
	for vi, v := range ca.Views {
		seen := map[uint8]bool{}
		for _, id := range v.pid {
			seen[id] = true
		}
		for id := uint8(1); id <= 6; id++ {
			if !seen[id] && !(vi == 1 && (id == 3 || id == 5)) {
				t.Errorf("view %d: part %d missing", vi, id)
			}
		}
	}
	if gaitOf("crash", true, 0) != "spin" {
		t.Errorf("gait %q", gaitOf("crash", true, 0))
	}
}

// PH_CRASH_SHEET=/path.png dumps a x6 sheet: rows = front/side/back, columns = poses, and the
// last column an ordinary guest for scale.
func TestCrashArtSheet(t *testing.T) {
	out := os.Getenv("PH_CRASH_SHEET")
	if out == "" {
		t.Skip("set PH_CRASH_SHEET to dump the sheet")
	}
	sp := crashSpec()
	plain := presetByName("marty_mcfly")(rand.New(rand.NewSource(1)))
	type pf struct{ a, f int }
	cols := []pf{{AnimIdle, 0}, {AnimWalk, 0}, {AnimWalk, 2}, {AnimRun, 1}, {AnimRun, 3}, {AnimDance0, 1}, {AnimHop, 2}, {AnimLaugh, 1}, {AnimIdle, -1}}
	const k = 6
	img := image.NewRGBA(image.Rect(0, 0, len(cols)*charW*k, 3*charH*k))
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			img.Set(x, y, color.RGBA{120, 150, 120, 255})
		}
	}
	for vw := 0; vw < 3; vw++ {
		for ci, c := range cols {
			who := &sp
			f := c.f
			if f < 0 {
				who, f = &plain, 0
			}
			pp := poseFor(c.a, f)
			pp.Anim, pp.F = c.a, f
			b := genChar(who, vw, pp).outlinedSel(0.42)
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					col := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < k; dy++ {
						for dx := 0; dx < k; dx++ {
							img.Set((ci*charW+x)*k+dx, (vw*charH+y)*k+dy, col)
						}
					}
				}
			}
		}
	}
	fo, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer fo.Close()
	png.Encode(fo, img)
}
