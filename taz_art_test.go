package main

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func tazSpec() CharSpec { return presetByName("taz")(rand.New(rand.NewSource(1))) }

func tazHas(b *pixBuf, c color.RGBA) bool {
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.get(x, y) == c {
				return true
			}
		}
	}
	return false
}

// tazWhirlFrames: the pirouette frames the game shows from assets/sprites/special_people/taz/anim
// (the rig cannot turn the body into a whirlwind, so they are exported as overrides).
func tazWhirlFrames() map[string]*pixBuf {
	sp := tazSpec()
	out := map[string]*pixBuf{}
	for vw := 0; vw < 3; vw++ {
		for f := 0; f < animFrameCount(AnimPirouette); f++ {
			pp := poseFor(AnimPirouette, f)
			pp.Anim, pp.F = AnimPirouette, f
			out[overrideKey(AnimPirouette, vw, f)] = renderChar(&sp, vw, pp)
		}
	}
	return out
}

// Every view and animation (gaits and the whirlwind included) draws inside the sprite box.
func TestTazArtDrawsInBox(t *testing.T) {
	sp := tazSpec()
	if !isTaz(&sp) {
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
}

// His look: brown fur, cream face and belly, pink inside the ears, white eyes and teeth, dark-brown
// nose, the red tongue; big (like the other large guests); the back shows no face; the three
// views differ; the jaws stick out in profile.
func TestTazArtLook(t *testing.T) {
	sp := tazSpec()
	pp := poseFor(AnimIdle, 0)
	pp.Anim = AnimIdle
	front, side, back := genChar(&sp, vFront, pp), genChar(&sp, vSide, pp), genChar(&sp, vBack, pp)
	for _, c := range []color.RGBA{tzFur, tzBeige, tzPink, tzEyeW, tzPupil, tzNose, tzMouth, tzTooth, tzTongue, tzClaw} {
		if !tazHas(front, c) {
			t.Errorf("front misses colour %v", c)
		}
	}
	for _, c := range []color.RGBA{tzFur, tzBeige, tzNose, tzTooth, tzTongue, tzEyeW} {
		if !tazHas(side, c) {
			t.Errorf("side misses colour %v", c)
		}
	}
	for _, c := range []color.RGBA{tzBeige, tzPink, tzEyeW, tzNose, tzTongue, tzTooth} {
		if tazHas(back, c) {
			t.Errorf("back shows face colour %v", c)
		}
	}
	if string(front.p) == string(back.p) || string(front.p) == string(side.p) || string(side.p) == string(back.p) {
		t.Error("two views are the same")
	}
	// the tongue is big: at least 30 tongue pixels in front, hanging below the mouth
	n, low := 0, 0
	for y := 0; y < front.h; y++ {
		for x := 0; x < front.w; x++ {
			if c := front.get(x, y); c == tzTongue || c == tzTongL || c == tzTongD {
				n++
				if y >= 22 {
					low++
				}
			}
		}
	}
	if n < 30 || low < 4 {
		t.Errorf("tongue: %d px, %d below the jaw", n, low)
	}
	// the nose sits at the front tip of the jaws in profile
	nx := -1
	for y := 0; y < 20; y++ {
		for x := 0; x < charW; x++ {
			if side.get(x, y) == tzNose && x > nx {
				nx = x
			}
		}
	}
	if nx < 19 {
		t.Errorf("profile jaws too short: nose at x=%d", nx)
	}
	// a large guest: tall and wide (as tall as the old fat big-headed taz)
	_, minY, _, maxY := bbox(front)
	minX, _, maxX, _ := bbox(front)
	if maxY-minY < 36 || maxX-minX < 17 {
		t.Errorf("too small: %d x %d", maxX-minX+1, maxY-minY+1)
	}
}

// Walking and running change the frames; the parts of the rig are all tagged; he walks heavily
// and dances as a whirlwind; the whirlwind is brown, turns and differs from the standing pose.
func TestTazArtAnimatesAndWhirls(t *testing.T) {
	sp := tazSpec()
	for vw := 0; vw < 3; vw++ {
		for _, a := range []int{AnimWalk, AnimRun, AnimLumber, AnimPirouette} {
			p0, p1 := poseFor(a, 0), poseFor(a, 1)
			p0.Anim, p0.F, p1.Anim, p1.F = a, 0, a, 1
			if string(genChar(&sp, vw, p0).p) == string(genChar(&sp, vw, p1).p) {
				t.Errorf("vw %d anim %d: frames 0 and 1 are the same", vw, a)
			}
		}
	}
	ca := charAssetFromSpec(sp, "taz", "special_people", "")
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
	if g := gaitOf("taz", true, 0); g != "twister" {
		t.Errorf("gait %q", g)
	}
	if a, _ := gaitAnim("twister", AnimDance1, 0); a != AnimPirouette {
		t.Errorf("dance -> %d", a)
	}
	if a, _ := gaitAnim("twister", AnimWalk, 0); a != AnimLumber {
		t.Errorf("walk -> %d", a)
	}
	pw := poseFor(AnimPirouette, 2)
	pw.Anim, pw.F = AnimPirouette, 2
	w := genChar(&sp, vFront, pw)
	if tazHas(w, tzPink) || tazHas(w, tzEyeW) || !tazHas(w, tzFur) || !tazHas(w, tzFurD) || !tazHas(w, tzTongue) {
		t.Error("the whirlwind is not a brown tornado with the tongue flying by")
	}
}

// The other famous guests do not change: Crash is still Crash and the old generic look is not
// borrowed by anybody else.
func TestTazArtOnlyTaz(t *testing.T) {
	for _, n := range []string{"crash", "la_mascara", "michael_jackson", "burro_shrek"} {
		f := presetByName(n)
		if f == nil {
			t.Fatalf("no preset %s", n)
		}
		sp := f(rand.New(rand.NewSource(1)))
		if isTaz(&sp) {
			t.Errorf("%s drawn as taz", n)
		}
		if tazHas(genChar(&sp, vFront, poseFor(AnimIdle, 0)), tzTongue) {
			t.Errorf("%s has the taz tongue", n)
		}
	}
}

// The exported whirlwind frames in assets are the ones the code draws (TAZ_ANIM_EXPORT=assets
// writes them again, with the three standing pictures).
func TestTazWhirlExported(t *testing.T) {
	dir := filepath.Join(spritesDir("assets", "special_people"), "taz", "anim")
	frames := tazWhirlFrames()
	if root := os.Getenv("TAZ_ANIM_EXPORT"); root != "" {
		dir = filepath.Join(spritesDir(root, "special_people"), "taz", "anim")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for k, b := range frames {
			if err := savePNG(filepath.Join(dir, k+".png"), b); err != nil {
				t.Fatal(err)
			}
		}
		// and the three standing pictures (meta.json is kept)
		for _, bc := range builtinChars() {
			if bc.name != "taz" || bc.cat != "special_people" {
				continue
			}
			ca := charimpFresh(bc)
			if old, err := loadCharAsset(filepath.Join(spritesDir(root, bc.cat), bc.name)); err == nil {
				carry := ca.Meta.Carry
				ca.Meta = old.Meta
				ca.Meta.Carry = carry
			}
			if err := saveCharAsset(root, ca); err != nil {
				t.Fatal(err)
			}
		}
	}
	for k, b := range frames {
		got, err := loadPNG(filepath.Join(dir, k+".png"))
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if got.w != b.w || got.h != b.h || string(got.p) != string(b.p) {
			t.Errorf("%s: the exported frame is not the one the code draws", k)
		}
	}
	// and the game picks them up for the pirouette
	ca, err := loadCharAsset(filepath.Join(spritesDir("assets", "special_people"), "taz"))
	if err != nil {
		t.Fatal(err)
	}
	cs := newRigCharSet(ca)
	if fr := cs.rigFrame(AnimPirouette, vFront, 1); !tazHas(fr, tzFurD) || tazHas(fr, tzPink) {
		t.Error("the game does not show the whirlwind for the pirouette")
	}
}

// PH_TAZ_SHEET=/path.png dumps a x6 sheet: rows = front/side/back, columns = poses, the last
// column an ordinary guest for scale.
func TestTazArtSheet(t *testing.T) {
	out := os.Getenv("PH_TAZ_SHEET")
	if out == "" {
		t.Skip("set PH_TAZ_SHEET to dump the sheet")
	}
	sp := tazSpec()
	plain := presetByName("marty_mcfly")(rand.New(rand.NewSource(1)))
	type pf struct{ a, f int }
	cols := []pf{{AnimIdle, 0}, {AnimLumber, 0}, {AnimLumber, 3}, {AnimRun, 1}, {AnimRun, 3}, {AnimPanic, 1}, {AnimPirouette, 0}, {AnimPirouette, 3}, {AnimIdle, -1}}
	const k = 6
	img := image.NewRGBA(image.Rect(0, 0, len(cols)*charW*k, 3*charH*k))
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			img.Set(x, y, color.RGBA{200, 200, 196, 255})
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
