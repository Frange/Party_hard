package main

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// The T-Rex that eats the cake after the credits is the T-Rex of the island level (the "jp_trex" sprite),
// in the poses of its meal: head down with the jaws open, bite, swallow/chew, roar (the burp); and it walks in and
// out with the walking poses. The old hand-painted green head must not come back.
func TestStingerRexIsLevelSprite(t *testing.T) {
	t.Cleanup(func() { propRec = nil })
	for _, two := range []bool{false, true} {
		g := layerGame(t, two)
		dst := ebiten.NewImage(screenW, screenH)
		g.startEndingPre(true)
		e := g.end
		e.stage = endStinger
		poses := map[string]bool{}
		for tt := 3.3; tt < 8.2; tt += 0.05 {
			e.t, e.total = tt, tt
			var rec []string
			propRec = &rec
			g.drawEnding(dst)
			propRec = nil
			n := 0
			for _, r := range rec {
				if strings.HasPrefix(r, "jp_trex:") {
					n++
					poses[r] = true
				} else {
					t.Errorf("2P=%v t=%.2f: the stinger draws another prop %q", two, tt, r)
				}
			}
			if n != 1 {
				t.Fatalf("2P=%v t=%.2f: the level's T-Rex drawn %d times (%v)", two, tt, n, rec)
			}
		}
		for _, want := range []string{"jp_trex:0", "jp_trex:5", "jp_trex:1", "jp_trex:2", "jp_trex:3", "jp_trex:10", "jp_trex:4"} {
			if !poses[want] {
				t.Errorf("2P=%v: the stinger never shows pose %s (seen %v)", two, want, poses)
			}
		}
		dst.Deallocate()
	}
}

// While it eats, the mouth of the rex is on the cake, its feet stay on the grass above the table, the whole beast is
// inside the screen (2x) with integer coordinates; it is out of the picture before and after.
func TestStingerRexPlacement(t *testing.T) {
	for tt := 0.0; tt < 8.5; tt += 0.01 {
		v, x, y, flip, ok := stingRexPose(tt)
		if !ok {
			continue
		}
		if x != math.Round(x) || y != math.Round(y) {
			t.Fatalf("t=%.2f: anchor (%v,%v) is not on whole pixels", tt, x, y)
		}
		if tt > 5.0 && tt < 6.9 {
			if flip {
				t.Fatalf("t=%.2f: it eats looking away from the cake", tt)
			}
			s := rexSkull(v)
			mx := x + s.X
			if mx < stingCakeX-20 || mx > stingCakeX+30 {
				t.Errorf("t=%.2f pose %d: the head (x=%.0f) is not over the cake (%.0f)", tt, v, mx, stingCakeX)
			}
			if feet := (y - 7) * 2; feet >= 392-12 {
				t.Errorf("t=%.2f: the feet (y=%.0f) are on the table (392)", tt, feet)
			}
			left, right := (x+rexAnchorOff(-rexArtOX, 0).X)*2, (x-rexAnchorOff(-rexArtOX, 0).X)*2
			top := (y + rexAnchorOff(0, -rexArtOY).Y) * 2
			if left < 16 || right > screenW-16 || top < 16 {
				t.Errorf("t=%.2f: the rex (%.0f..%.0f, top %.0f) leaves the margins", tt, left, right, top)
			}
		}
	}
	if _, _, _, _, ok := stingRexPose(3.0); ok {
		t.Error("the rex is there before it arrives")
	}
	if _, x, _, _, _ := stingRexPose(8.45); x+rexAnchorOff(-rexArtOX, 0).X < float64(endCW) {
		t.Errorf("the rex has not left at the end (anchor x %.0f)", x)
	}
}

// The party hat sits where the party and the prelude already put it on the same sprite.
func TestRexHatMatchesParty(t *testing.T) {
	h := rexHatSpot(rexPoseWalkA)
	if math.Abs(h.X-(-42)) > 1.5 || math.Abs(h.Y-(-80)) > 1.5 {
		t.Errorf("hat at %+v from the anchor; partyRex puts it at (-42,-80)", h)
	}
	if s := rexSkull(1); s.Y <= rexSkull(0).Y {
		t.Errorf("pose 1 must have the head down: skull %+v vs walking %+v", s, rexSkull(0))
	}
}

// The head in the neighbour's window (secret ending) is the level's T-Rex too, and the dinosaur sheep's head
// comes from the same art.
func TestWindowRexAndSheepUseLevelArt(t *testing.T) {
	t.Cleanup(func() { propRec = nil })
	g := layerGame(t, false)
	c := ebiten.NewImage(endCW, endCH)
	defer c.Deallocate()
	g.startEndingPre(true)
	e := g.end
	e.trueEnd = true
	e.t = 7.2
	var rec []string
	propRec = &rec
	g.endNeighbour(c, 0, 7.2, e)
	propRec = nil
	if len(rec) != 1 || rec[0] != "jp_trex:0" {
		t.Errorf("the head in the window: props %v, want the level's T-Rex", rec)
	}
	im := rexMiniHead()
	if im == nil || im.Bounds().Dx() != 24 || im.Bounds().Dy() != 18 {
		t.Fatalf("the mini head of the dinosaur sheep: %v", im)
	}
}

// No scene paints its own T-Rex any more: the colours and shapes of the old green head are gone from the endings.
func TestNoOldRexArtInScenes(t *testing.T) {
	old := regexp.MustCompile(`0x2a4a1c|0x2a3a1c|0x3a8a30`)
	for _, f := range []string{"ending2.go", "ending2p.go", "ending.go", "endparty.go", "prelude.go", "arrival.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if m := old.Find(b); m != nil {
			t.Errorf("%s still has the colour %s of the old hand-painted T-Rex", f, m)
		}
	}
	b, _ := os.ReadFile("ending2.go")
	for _, want := range []string{"g.drawStingRex(dst, e)", "g.windowRex(", "rexMiniHead()"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("ending2.go does not call %s: the T-Rex scenes must draw the level's T-Rex", want)
		}
	}
}

// rexTopAt: the highest opaque pixel row (canvas coordinates) of the rex in pose v drawn at (x, y) in canvas column
// c, or -1 if the column is empty. It reads the very pixels buildPropImages makes for the sprite.
func rexTopAt(v int, x, y float64, flip bool, c int) int {
	b, ox, oy := rexSpritePix(v)
	px := c - int(x) + int(ox)
	if flip {
		px = int(ox) + int(x) - c - 1
	}
	if px < 0 || px >= b.w {
		return -1
	}
	for py := 0; py < b.h; py++ {
		if b.p[(py*b.w+px)*4+3] >= 128 {
			return py + int(y) - int(oy)
		}
	}
	return -1
}

// The party hat sits ON the skull (it neither floats nor sinks): the bottom row of its brim is on the outline of
// the head or right above it (distance 0 or 1 px) in every side pose of the sprite, both facings, and in every
// frame of the cake scene; and it is on the skull, not on the neck or the snout.
func TestRexHatTouchesSkull(t *testing.T) {
	check := func(tag string, v int, x, y float64, flip bool) {
		t.Helper()
		h := rexHatSpot(v)
		sx := 1.0
		if flip {
			sx = -1
		}
		hx, hy := int(math.Round(x+sx*h.X)), int(math.Round(y+h.Y))
		best := 1 << 30
		for c := hx - 4; c <= hx+3; c++ { // the brim (preHat)
			if top := rexTopAt(v, x, y, flip, c); top >= 0 {
				best = min(best, top-hy)
			}
		}
		if best < 0 || best > 1 {
			t.Errorf("%s pose %d flip=%v: the hat's brim (row %d) is %d px from the head (want 0 or 1)", tag, v, flip, hy, best)
		}
		s := rexSkull(v)
		if d := math.Hypot(sx*h.X-sx*s.X, h.Y-s.Y); d > 17 {
			t.Errorf("%s pose %d: the hat is %.1f px from the centre of the skull: not on the skull", tag, v, d)
		}
	}
	for _, v := range []int{0, 1, 2, 3, 4, 5, rexPoseChew} {
		for _, flip := range []bool{false, true} {
			check("pose", v, 200, 150, flip)
		}
	}
	for tt := 3.2; tt < 8.5; tt += 0.01 {
		if v, x, y, flip, ok := stingRexPose(tt); ok {
			check("t="+strconv.FormatFloat(tt, 'f', 2, 64), v, x, y, flip)
		}
	}
}

// ÑAM ÑAM and BUUURP (Spanish and English) never cover the rex, its hat or the cake: >= 12 px of air from the
// real pixels of the sprite and the hat (2x on the screen) and from the cake, >= 16 px from the edges, whole
// scales; checked on the texts the scene really draws, every frame they are on screen.
func TestStingerBangsClearOfRex(t *testing.T) {
	g := layerGame(t, false)
	pg, pe, pu := gameLang, editorLang, uiLang
	t.Cleanup(func() { gameLang, editorLang, uiLang = pg, pe, pu; textRec = nil })
	dst := ebiten.NewImage(screenW, screenH)
	defer dst.Deallocate()
	gap := func(a, b Rect) float64 {
		dx := math.Max(b.X-(a.X+a.W), a.X-(b.X+b.W))
		dy := math.Max(b.Y-(a.Y+a.H), a.Y-(b.Y+b.H))
		return math.Max(dx, dy)
	}
	cake := Rect{350, 296, 100, 96}
	for _, lang := range []string{"es", "en"} {
		gameLang, editorLang, uiLang = lang, lang, lang
		seen := map[string]bool{}
		for tt := 5.0; tt < 8.2; tt += 0.01 {
			textRec = &textRecorder{}
			g.drawStingBangs(dst, tt)
			v, x, y, flip, ok := stingRexPose(tt)
			r, ok2 := rexSpriteRect(v, x, y, flip, true)
			if !ok || !ok2 {
				t.Fatalf("t=%.2f: no rex", tt)
			}
			rex := Rect{2 * r.X, 2 * r.Y, 2 * r.W, 2 * r.H}
			for _, tx := range textRec.texts {
				seen[tx.s] = true
				box := Rect{tx.x, tx.y, tx.w + pxOff(tx.scale), tx.h + pxOff(tx.scale)}
				if tx.scale != math.Round(tx.scale) || tx.x != math.Round(tx.x) || tx.y != math.Round(tx.y) {
					t.Errorf("[%s] t=%.2f %q: scale %v at (%v,%v) is not whole", lang, tt, tx.s, tx.scale, tx.x, tx.y)
				}
				if d := gap(box, rex); d < 12 {
					t.Errorf("[%s] t=%.2f %q %+v: %.0f px from the rex+hat %+v (pose %d)", lang, tt, tx.s, box, d, rex, v)
				}
				if tt < 5.9 {
					if d := gap(box, cake); d < 12 {
						t.Errorf("[%s] t=%.2f %q %+v: %.0f px from the cake", lang, tt, tx.s, box, d)
					}
				}
				if box.X < 16 || box.Y < 16 || box.X+box.W > screenW-16 || box.Y+box.H > screenH-16 {
					t.Errorf("[%s] t=%.2f %q %+v: outside the 16 px margin", lang, tt, tx.s, box)
				}
			}
		}
		for _, want := range []string{tr("ÑAM ÑAM"), tr("BUUURP")} {
			if !seen[want] {
				t.Errorf("[%s] the scene never writes %q (seen %v)", lang, want, seen)
			}
		}
	}
}

// The scrap in the mouth (poses 2, 3, 10) is painted in the tones of the mouth only in the cake scene: the T-Rex
// of the Jurassic level keeps its red scrap.
func TestStingerRexScrapDiscreet(t *testing.T) {
	if rxScrap != c(0xb01818) {
		t.Fatalf("the level's scrap colour changed: %v", rxScrap)
	}
	for _, v := range []int{2, 3, rexPoseChew} {
		if rexScenePropImg(v) == nil {
			t.Fatalf("pose %d: the cake scene has no picture of its own", v)
		}
		if rxScrap != c(0xb01818) {
			t.Fatalf("pose %d: building the scene's picture left the scrap colour at %v", v, rxScrap)
		}
	}
	for _, v := range []int{0, 1, 4, 5} {
		if rexScenePropImg(v) != nil {
			t.Errorf("pose %d has no scrap: it must use the level's picture", v)
		}
	}
}
