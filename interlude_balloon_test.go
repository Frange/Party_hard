package main

import (
	"image/color"
	"math"
	"os"
	"regexp"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// the passengers stay inside the basket (in the coordinates of the rig, which is what moves and tilts)
func TestBalloonPassengersStayInBasket(t *testing.T) {
	for _, two := range []bool{false, true} {
		for tt := 0.0; tt <= balloonLen; tt += 0.05 {
			var xs [2]float64
			for k := 0; k < 2; k++ {
				x, y, _, _, _, _, _ := balloonPose(tt, k, two)
				xs[k] = x
				const m = 8.0
				if x < bnRX-bnBW/2+m || x > bnRX+bnBW/2-m || y < bnRY-bnBH || y > bnRY+4 {
					t.Fatalf("2P=%v t=%.2f: passenger %d at (%.0f,%.0f) outside the basket", two, tt, k, x, y)
				}
			}
			if two && math.Abs(xs[0]-xs[1]) < 30 {
				t.Fatalf("t=%.2f: both on the same seat", tt)
			}
		}
	}
}

func TestBalloonTimelineAndStory(t *testing.T) {
	if !(bnCut < bnPeak && bnPeak < bnPeck && bnPeck < bnFall0 && bnFall0 < bnThrow0 && bnThrow0 < bnThrow1 && bnThrow1 < bnThrow2 &&
		bnThrow2 < bnTrees && bnTrees < bnZip && bnZip < bnFade && bnFade < balloonLen) {
		t.Fatal("the balloon timeline is out of order")
	}
	if balloonLen > 20 || balloonLen < 10 {
		t.Fatalf("the balloon lasts %.1f s", balloonLen)
	}
	if len(interludePlaces(interBalloon)) != 0 {
		t.Error("the balloon has its own scene: no street memories")
	}
	// it climbs, then falls, and is flat at the end; it is never under the field
	if balloonAlt(0) != 0 || balloonAlt(bnPeak) < 400 || balloonAlt(bnTrees) > 120 {
		t.Errorf("altitude: %.0f %.0f %.0f", balloonAlt(0), balloonAlt(bnPeak), balloonAlt(bnTrees))
	}
	for tt := 0.0; tt <= balloonLen; tt += 0.05 {
		if balloonAlt(tt) < 0 {
			t.Fatalf("t=%.2f: under the ground", tt)
		}
		if d := balloonDeflate(tt); d < 0 || d > 1 {
			t.Fatalf("t=%.2f: deflate %v", tt, d)
		}
	}
	if balloonDeflate(bnPeck-0.1) != 0 || balloonDeflate(bnZip) < 0.99 {
		t.Error("it deflates only after the peck and is flat when it flies off")
	}
	// it is still on the screen when the fade to black starts (the basket centre is at 480+tx) and has left it by the end
	if tx, _, _ := balloonRig(bnFade); 480+tx > screenW-40 {
		t.Errorf("the balloon is already gone when the fade starts (tx %.0f)", tx)
	}
	if tx, _, _ := balloonRig(balloonLen - 0.2); tx < 600 { // the whole basket (200 wide) is beyond the right edge
		t.Errorf("the balloon has not left the screen at the end (tx %.0f)", tx)
	}
	// everything they throw has a different thrower in 2P and the same in 1P
	if balloonThrower(0, false) != balloonThrower(1, false) || balloonThrower(0, true) == balloonThrower(1, true) {
		t.Error("throwers")
	}
	// varied actions
	for k := 0; k < 2; k++ {
		seen := map[int]bool{}
		for _, tt := range []float64{0.5, bnCut, bnCut + 1, bnPeck + 0.1, bnTrees + 0.1, bnZip + 1} {
			_, _, a, _, _, _, _ := balloonPose(tt, k, true)
			seen[a] = true
		}
		if len(seen) < 3 {
			t.Fatalf("passenger %d (2P): only %d different actions", k, len(seen))
		}
	}
}

func TestBalloonInterludeDraws(t *testing.T) {
	g := auditGame(t)
	oldPx := px
	if px == nil {
		px = ebiten.NewImage(1, 1)
		px.Fill(color.White)
	}
	t.Cleanup(func() { px = oldPx; g.inter = nil })
	for _, two := range []bool{false, true} {
		in := &interState{kind: interBalloon, two: two, did: map[string]bool{}}
		g.inter = in
		dst := ebiten.NewImage(screenW, screenH)
		for tt := 0.0; tt < balloonLen; tt += 0.3 {
			in.t = tt
			g.drawInterlude(dst)
		}
	}
}

// seeing the balloon is saved even with the progress frozen (as in a debug session): in memory, never on disk
func TestBalloonSeenWithFrozenProgress(t *testing.T) {
	prev := progressFrozen
	progressFrozen = true
	t.Cleanup(func() { progressFrozen = prev })
	g := &Game{}
	g.progress = Progress{Stats: map[string]int{}}
	if _, ok := g.interludeAfter(campaignAtSlot(14)); !ok { // the 15th row of the list (campaign_retired.go)
		t.Fatal("after party 15 there must be the balloon")
	}
	g.markSeen("seen_interlude15")
	if !g.seen("seen_interlude15") {
		t.Fatal("seen_interlude15 is not remembered")
	}
	if _, ok := g.interludeAfter(14); ok {
		t.Fatal("the balloon shows again")
	}
	for _, i := range []int{13, 15, 16} {
		if _, ok := g.interludeAfter(i); ok {
			t.Errorf("level %d has an interlude", i)
		}
	}
}

// every phrase of the boat and balloon interludes (captions, bubbles, comic words) is short and has its English
func TestInterludeNewTextsTranslated(t *testing.T) {
	re := regexp.MustCompile(`(?:say\(\d, |line = |sceneBang\([^"]*)"([^"\\]+)"`)
	for _, f := range []string{"interlude_boat.go", "interlude_balloon.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, m := range re.FindAllStringSubmatch(string(src), -1) {
			n++
			s := m[1]
			if utf8.RuneCountInString(s) > 48 {
				t.Errorf("%s: %q is longer than 48 characters", f, s)
			}
			if _, ok := langDataEN[s]; !ok {
				t.Errorf("%s: no English for %q", f, s)
			}
		}
		if n < 12 {
			t.Errorf("%s: only %d phrases found: is the pattern out of date?", f, n)
		}
	}
}
