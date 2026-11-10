package main

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

func johnny5Spec() CharSpec { return presetByName("johnny5")(rand.New(rand.NewSource(1))) }

// Every view and animation (gaits included) draws something, inside the sprite box with a 1px
// margin, and taller than an ordinary guest but not a giant.
func TestJohnny5DrawsInBox(t *testing.T) {
	sp := johnny5Spec()
	if !isJohnny5(&sp) {
		t.Fatalf("preset name %q", sp.Name)
	}
	for vw := 0; vw < 3; vw++ {
		for a := 0; a < animGaitEnd; a++ {
			for f := 0; f < animFrameCount(a); f++ {
				pp := poseFor(a, f)
				pp.Anim, pp.F = a, f
				pp.Hold = HoldNone
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
	// size: compare with a plain guest standing
	pp := poseFor(AnimIdle, 0)
	pp.Anim = AnimIdle
	_, jy0, _, jy1 := bbox(genChar(&sp, vFront, pp))
	plain := presetByName("marty_mcfly")(rand.New(rand.NewSource(1)))
	_, gy0, _, gy1 := bbox(genChar(&plain, vFront, pp))
	jh, gh := jy1-jy0, gy1-gy0
	if jh <= gh || jh > gh+8 {
		t.Errorf("height %d vs plain guest %d: want a bit taller, not a giant", jh, gh)
	}
	if jy1 != gy1 && jy1 != gy1+1 {
		t.Errorf("tracks rest at row %d, guest feet at %d", jy1, gy1)
	}
}

// The tracks roll: walking frames differ in the track rows; no leg pixels (the tracks are the
// bottom and they are dark).
func TestJohnny5TracksRoll(t *testing.T) {
	sp := johnny5Spec()
	for _, a := range []int{AnimWalk, AnimRobot} {
		for _, vw := range []int{vFront, vSide} {
			p0, p1 := poseFor(a, 0), poseFor(a, 1)
			p0.Anim, p0.F, p1.Anim, p1.F = a, 0, a, 1
			b0, b1 := genChar(&sp, vw, p0), genChar(&sp, vw, p1)
			diff := 0
			for y := 35; y < charH; y++ {
				for x := 0; x < charW; x++ {
					if b0.get(x, y) != b1.get(x, y) {
						diff++
					}
				}
			}
			if diff == 0 {
				t.Errorf("anim %d vw %d: tracks do not roll", a, vw)
			}
			// the bottom row is track (black or tread), never shoes/legs
			for x := 0; x < charW; x++ {
				c := b0.get(x, 42)
				if c.A != 0 && int(c.R)+int(c.G)+int(c.B) > 3*0x50 {
					t.Errorf("anim %d vw %d: bright pixel %v at the bottom (x=%d)", a, vw, c, x)
				}
			}
		}
	}
}

// Standing with empty hands he waves: the right claw goes above the shoulders.
func TestJohnny5SaluteWhenIdle(t *testing.T) {
	sp := johnny5Spec()
	p := poseFor(AnimIdle, 0)
	p.Anim = AnimIdle
	b := genChar(&sp, vFront, p)
	found := false
	for y := 0; y < 20; y++ {
		for x := 19; x < charW; x++ {
			if b.get(x, y).A != 0 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no raised claw on the right while idle")
	}
	// the logical character does not change: same spec (hitbox comes from the game, not the art)
	// and the robot gait hooks keep working
	if sp.Body != BodyRobot || sp.Stature != 0 {
		t.Errorf("spec changed: body %v stature %d", sp.Body, sp.Stature)
	}
	if gaitOf("johnny5", true, 0) != "robot" {
		t.Errorf("gait %q", gaitOf("johnny5", true, 0))
	}
}

// PH_J5_SHEET=/path.png dumps a x5 sheet: rows = front/side/back, columns = poses.
func TestJohnny5Sheet(t *testing.T) {
	out := os.Getenv("PH_J5_SHEET")
	if out == "" {
		t.Skip("set PH_J5_SHEET to dump the sheet")
	}
	sp := johnny5Spec()
	type pf struct{ a, f int }
	cols := []pf{{AnimIdle, 0}, {AnimIdle, 1}, {AnimWalk, 0}, {AnimWalk, 2}, {AnimRobot, 1}, {AnimDance0, 1}, {AnimRobotDance, 2}, {AnimDrink, 2}, {AnimRun, 3}}
	const k = 5
	img := image.NewRGBA(image.Rect(0, 0, len(cols)*charW*k, 3*charH*k))
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			img.Set(x, y, color.RGBA{200, 200, 205, 255})
		}
	}
	for vw := 0; vw < 3; vw++ {
		for ci, c := range cols {
			pp := poseFor(c.a, c.f)
			pp.Anim, pp.F = c.a, c.f
			b := genChar(&sp, vw, pp).outlinedSel(0.42)
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
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, img)
}
