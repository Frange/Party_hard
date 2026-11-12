package main

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func emoteRestore(t *testing.T) {
	t.Helper()
	on := emotesOn
	t.Cleanup(func() { emotesOn = on })
	emotesOn = true
}

// each state gives its icon
func TestEmoteByState(t *testing.T) {
	want := map[gState]emoteKind{
		gAlarm: emNone, gToPhone: emNone, gCall: emNone, gReport: emNone, gFlee: emNone, gScared: emNone, gVomit: emNone, gPunch: emAnger,
		gSleep: emZzz, gKiss: emHeart, gDanceWith: emHeart, gDance: emNote, gChat: emTalk, gWaitP: emQuestion, gIdle: emNone, gDrink: emNone,
	}
	for st, k := range want {
		if got := emoteOf(&Guest{state: st}); got != k {
			t.Errorf("state %d: emote %d, want %d", st, got, k)
		}
	}
	if emoteOf(&Guest{state: gIdle, poison: 4}) != emSkull || emoteOf(&Guest{state: gIdle, bitten: 3}) != emSkull {
		t.Error("poisoned/bitten guests wear the skull")
	}
	if emoteOf(&Guest{state: gAlarm, dead: true}) != emNone || emoteOf(&Guest{state: gAlarm, lift: 5}) != emNone {
		t.Error("dead or abducted guests wear nothing")
	}
	if emoteOf(nil) != emNone {
		t.Error("nil")
	}
}

func emoteGame(t *testing.T, two bool) *Game {
	emoteRestore(t)
	g := hudPlay(t, two)
	// the tests put bare guests (no CharSet) in the shared game: give it a real party back when they end, or the next
	// test that draws it (TestMenuMarginsMainMenu: drawActor) meets them
	prevMode := g.mode
	t.Cleanup(func() {
		g.loadLevel(0)
		g.emo.reset()
		g.mode = prevMode
	})
	g.stealth = false
	g.loadLevel(0)
	g.mode = ModePlay
	g.guests = nil
	g.emo.reset()
	return g
}

// at most 6 at once (3 calm ones), rotation gives everyone a turn, icons end by time.
func TestEmoteCapAndRotation(t *testing.T) {
	g := emoteGame(t, false)
	for i := 0; i < 30; i++ {
		g.guests = append(g.guests, &Guest{pos: Vec{80 + float64(i%10)*80, 120 + float64(i/10)*100}, state: gDance})
	}
	maxN, maxAmb := 0, 0
	seen := map[*Guest]bool{}
	for f := 0; f < 60*30; f++ {
		g.time += 1.0 / 60
		g.updateEmotes(1.0 / 60)
		n, amb := g.emo.count()
		maxN, maxAmb = maxI(maxN, n), maxI(maxAmb, amb)
		for _, s := range g.emo.slots {
			if s.gu != nil {
				seen[s.gu] = true
			}
		}
	}
	if maxN > emoteMax || maxAmb > emoteAmbientMax {
		t.Errorf("too many at once: %d (calm %d)", maxN, maxAmb)
	}
	if maxAmb < emoteAmbientMax {
		t.Errorf("never reached the calm limit (%d)", maxAmb)
	}
	if len(seen) < 20 {
		t.Errorf("only %d of 30 guests got an icon in 30 s: no rotation", len(seen))
	}
	// urgent ones come first: punching guests take the free places even with the calm ones full
	for _, gu := range g.guests[:8] {
		gu.state = gPunch
	}
	for f := 0; f < 120; f++ {
		g.time += 1.0 / 60
		g.updateEmotes(1.0 / 60)
	}
	n, amb := g.emo.count()
	if n-amb < 1 {
		t.Error("urgent states did not get an icon")
	}
	// the icons end
	for _, gu := range g.guests {
		gu.state = gIdle
	}
	for f := 0; f < 120; f++ {
		g.time += 1.0 / 60
		g.updateEmotes(1.0 / 60)
	}
	if n, _ := g.emo.count(); n != 0 {
		t.Errorf("%d icons outlived their guests' state", n)
	}
}

// off screen guests, icons are not drawn over a player, a bubble or another emote; the switch works; 1P and 2P draw.
func TestEmoteDrawNoOverlap(t *testing.T) {
	for _, two := range []bool{false, true} {
		t.Run(fmt.Sprint("two=", two), func(t *testing.T) {
			g := emoteGame(t, two)
			img := ebiten.NewImage(screenW, screenH)
			defer img.Deallocate()
			pl := g.first()
			pl.pos = Vec{300, 300}
			near := &Guest{pos: Vec{303, 330}, state: gAlarm}      // right under the player's column
			free := &Guest{pos: Vec{600, 300}, state: gAlarm}      // free
			close2 := &Guest{pos: Vec{604, 300}, state: gPunch}    // on top of the free one's icon
			offscreen := &Guest{pos: Vec{-50, 300}, state: gAlarm} // out of the screen
			g.guests = []*Guest{near, free, close2, offscreen}
			g.emo.slots[0] = emoteSlot{gu: near, kind: emExcl, t: 0.5}
			g.emo.slots[1] = emoteSlot{gu: free, kind: emExcl, t: 0.5}
			g.emo.slots[2] = emoteSlot{gu: close2, kind: emAnger, t: 0.5}
			g.drawEmotes(img, nil)
			var drawn []*Guest
			for i := 0; i < g.emo.nDrawn; i++ {
				drawn = append(drawn, g.emo.drawn[i].gu)
			}
			has := func(gu *Guest) bool {
				for _, d := range drawn {
					if d == gu {
						return true
					}
				}
				return false
			}
			if has(near) {
				t.Error("an emote was drawn over the player")
			}
			if !has(free) {
				t.Error("the free emote was not drawn")
			}
			if has(close2) {
				t.Error("two emotes overlap")
			}
			for i := 0; i < g.emo.nDrawn; i++ {
				d := g.emo.drawn[i]
				for j := i + 1; j < g.emo.nDrawn; j++ {
					e := g.emo.drawn[j]
					if rectsHit(d.x, d.y, d.w, d.h, e.x, e.y, e.w, e.h) {
						t.Error("drawn emotes overlap")
					}
				}
			}
			// a bubble over the head hides the emote
			x, y, w, h := g.emoteRect(free, 0.5)
			bub := []bubReq{{cx: x + w/2, y: y - 2, w: 60, h: 13}}
			g.drawEmotes(img, bub)
			if g.emo.nDrawn != 0 && g.emo.drawn[0].gu == free {
				t.Error("an emote was drawn over a speech bubble")
			}
			_ = h
			// the switch
			emotesOn = false
			g.drawEmotes(img, nil)
			if g.emo.nDrawn != 0 {
				t.Error("drawn with the switch off")
			}
			g.updateEmotes(0.016)
			if n, _ := g.emo.count(); n != 0 {
				t.Error("slots kept with the switch off")
			}
			emotesOn = true
			g.mode = ModeLevelDone
			g.drawEmotes(img, nil)
			if g.emo.nDrawn != 0 {
				t.Error("drawn outside the party")
			}
		})
	}
}

// the pictures exist for every kind, and the pop is a whole step
func TestEmotePictures(t *testing.T) {
	auditGame(t) // the 1x1 pixel the pictures are painted with (px) is made by newGame: run first under -shuffle it was nil
	for k := emQuestion; k < emCount; k++ {
		im := emoteImage(k)
		if im == nil || im.Bounds().Dx() != emoteSize || im.Bounds().Dy() != emoteSize {
			t.Errorf("kind %d picture", k)
		}
		for _, row := range emoteArt[k] {
			if len(row) != 7 {
				t.Errorf("kind %d art row %q", k, row)
			}
		}
	}
	if emoteScale(0.05) != 2 || emoteScale(0.5) != 1 {
		t.Error("pop steps")
	}
}

// no witness state wears an icon (the player's "!" and the cuffs are the only "they saw you" signals), and none is handed out
func TestEmoteWitnessNone(t *testing.T) {
	emoteRestore(t)
	g := emoteGame(t, false)
	gu := &Guest{pos: Vec{600, 300}, state: gAlarm}
	g.guests = []*Guest{gu}
	for _, st := range []gState{gAlarm, gToPhone, gCall, gReport} {
		gu.state = st
		for i := 0; i < 30; i++ {
			g.updateEmotes(0.1)
		}
		if n, _ := g.emo.count(); n != 0 {
			t.Fatalf("state %d: a witness got an icon", st)
		}
	}
}
