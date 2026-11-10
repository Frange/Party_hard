package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// leaveAlive kills (silently) every guest and guard but the first n, and returns the n who stay.
func leaveAlive(g *Game, n int) []*Guest {
	var keep []*Guest
	for _, gu := range g.guests {
		if gu.out() || gu.py != nil {
			continue
		}
		if len(keep) < n && gu.perf == "" {
			keep = append(keep, gu)
			continue
		}
		gu.dead = true
	}
	for _, gd := range g.guards {
		gd.dead = true
	}
	for _, s := range g.allSlashers() {
		s.state = slGone
	}
	return keep
}

func TestLastGuestsMarkerOnlyWhenFewLeft(t *testing.T) {
	g := hudPlay(t, false)
	if ts := g.lastTargets(); len(ts) != 0 {
		t.Fatalf("%d targets with %d people left", len(ts), g.aliveCount())
	}
	for n := lastGuestsMax + 1; n >= 0; n-- {
		leaveAlive(g, n)
		ts := g.lastTargets()
		want := n
		if n > lastGuestsMax || n == 0 {
			want = 0
		}
		if g.aliveCount() != n || len(ts) != want {
			t.Errorf("%d alive: alive=%d targets=%d, want %d", n, g.aliveCount(), len(ts), want)
		}
		if n > 0 && n <= lastGuestsMax {
			for _, p := range ts { // each target is one of the people who are left
				ok := false
				for _, gu := range g.guests {
					ok = ok || (!gu.out() && gu.pos == p)
				}
				if !ok {
					t.Errorf("target %v is nobody alive", p)
				}
			}
		}
	}
}

func TestLastGuestsNotInStealthOrOtherModes(t *testing.T) {
	g := hudPlay(t, false)
	leaveAlive(g, 2)
	if len(g.lastTargets()) != 2 {
		t.Fatal("setup: expected 2 targets")
	}
	g.stealth = true
	if len(g.lastTargets()) != 0 {
		t.Error("marker in STEALTH")
	}
	g.stealth = false
	for _, m := range []Mode{ModeLevelDone, ModeLose, ModeInterlude, ModeEnding, ModeMenu} {
		g.mode = m
		if len(g.lastTargets()) != 0 {
			t.Errorf("marker in mode %v", m)
		}
	}
	g.mode = ModePlay
}

func TestLastGuestsVIPStaffDoesNotCount(t *testing.T) {
	g := hudPlay(t, false)
	leaveAlive(g, 1)
	// the Guard below has no CharSet: it must not stay in the shared game, or the next test that draws the menu
	// (TestMenuHasNoRemovedFootLines) crashes in drawActor
	prevGuards := g.guards
	t.Cleanup(func() { g.guards = prevGuards })
	g.guards = append(append([]*Guard(nil), g.guards...), &Guard{vip: 1})
	if n := len(g.lastTargets()); n != 1 {
		t.Errorf("%d targets with 1 guest and a VIP guard", n)
	}
}

// Direction and place of the chevron for a person off the screen, and none for one on the screen.
func TestLastEdgeArrow(t *testing.T) {
	cases := []struct {
		name   string
		sx, sy float64
		ang    float64
	}{
		{"right", screenW + 300, screenH / 2, 0},
		{"left", -300, screenH / 2, math.Pi},
		{"down", screenW / 2, screenH + 300, math.Pi / 2},
		{"up", screenW / 2, -300, -math.Pi / 2},
		{"down-right", screenW + 500, screenH + 500, math.Atan2(screenH/2+500, screenW/2+500)},
	}
	for _, c := range cases {
		x, y, ang, ok := lastEdgeArrow(c.sx, c.sy)
		if !ok {
			t.Errorf("%s: no chevron", c.name)
			continue
		}
		if math.Abs(ang-c.ang) > 1e-9 {
			t.Errorf("%s: angle %.3f, want %.3f", c.name, ang, c.ang)
		}
		if x < lastEdgeM || x > screenW-lastEdgeM || y < lastEdgeM || y > screenH-lastEdgeM {
			t.Errorf("%s: chevron at %.0f,%.0f is not inside the border margin", c.name, x, y)
		}
		onEdge := x == lastEdgeM || x == screenW-lastEdgeM || y == lastEdgeM || y == screenH-lastEdgeM
		if !onEdge {
			t.Errorf("%s: chevron at %.0f,%.0f does not touch the border", c.name, x, y)
		}
	}
	if _, _, _, ok := lastEdgeArrow(480, 270); ok {
		t.Error("a chevron for somebody on screen")
	}
	if _, _, _, ok := lastEdgeArrow(0, 0); ok {
		t.Error("a chevron for a corner of the screen itself")
	}
}

func lastDrawAll(g *Game, img *ebiten.Image) {
	g.drawLastRings(img) // resets g.lastMarks, then the arrows add theirs
	g.drawLastArrows(img)
}

// With a person outside the visible area a big chevron is drawn on the border; with one on screen, a double ring
// on the floor and a big arrow over the head.
func TestLastGuestsDrawRingArrowAndChevron(t *testing.T) {
	g := hudPlay(t, false)
	keep := leaveAlive(g, 2)
	keep[0].pos = g.lvl.NearestFree(Vec{300, 300}) // on screen, on free ground
	keep[1].pos = Vec{screenW + 400, 200}          // far to the right, off screen
	if _, _, ang, off := lastEdgeArrow(g.lastScreen(keep[1].pos)); !off || math.Abs(ang) > 0.5 {
		t.Fatalf("the second guest should be off screen to the right (ang %.2f, off %v)", ang, off)
	}
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	for _, tm := range []float64{0.125, 0.375} { // both halves of the 2 Hz pulse
		g.time = tm
		lastDrawAll(g, img)
		var rings, arrows []lastMark
		for _, m := range g.lastMarks {
			if m.arrow {
				arrows = append(arrows, m)
			} else {
				rings = append(rings, m)
			}
		}
		if len(rings) != 1 || len(arrows) != 1 {
			t.Fatalf("%d rings and %d chevrons, want 1 and 1", len(rings), len(arrows))
		}
		r := rings[0]
		sx, sy := g.lastScreen(keep[0].pos)
		if r.x != sx || r.y != sy || r.cellsDrawn < 60 {
			t.Errorf("ring at %.0f,%.0f with %d pixels, want a real ring under the guest at %.0f,%.0f", r.x, r.y, r.cellsDrawn, sx, sy)
		}
		if r.label || r.half != lastArrowHalf || r.rw < 15 || r.rh < 8 || r.tipY >= g.wy(keep[0].pos.Y) {
			t.Errorf("arrow over the head wrong: %+v", r)
		}
		if r.rx < 0 || r.ry < 0 || r.rx+r.rw > screenW {
			t.Errorf("arrow outside the screen: %+v", r)
		}
		ex, ey, ang, _ := lastEdgeArrow(g.lastScreen(keep[1].pos))
		if arrows[0].x != ex || arrows[0].y != ey || arrows[0].ang != ang {
			t.Errorf("chevron %+v, want %.0f,%.0f angle %.2f", arrows[0], ex, ey, ang)
		}
		if arrows[0].alpha < 0.85 || r.alpha < 0.85 { // strong, not the faint marker of before
			t.Errorf("alpha chevron %.2f ring %.2f: too faint", arrows[0].alpha, r.alpha)
		}
	}
}

// The ring is double (gold + pale, dark outline) and the two rings swap colours with the 2 Hz pulse.
func TestLastRingIsDoubleAndPulses(t *testing.T) {
	seen := map[int]bool{}
	for d := 0.0; d < 1.2; d += 0.01 {
		seen[lastRingClass(d, lcGold, lcPale)] = true
	}
	for _, c := range []int{lcNone, lcDark, lcGold, lcPale} {
		if !seen[c] {
			t.Errorf("class %d never appears", c)
		}
	}
	g := hudPlay(t, false)
	if g.time = 0.125; !g.lastPhase() {
		t.Error("first half of the pulse expected")
	}
	if g.time = 0.375; g.lastPhase() {
		t.Error("second half of the pulse expected")
	}
	g.time = 0
	g.time = 0.05
	b0 := g.lastBob()
	got := map[float64]bool{}
	for i := 0; i < 120; i++ {
		g.time = float64(i) / 120
		got[g.lastBob()] = true
	}
	for b := range got {
		if b < 0 || b > 3 || b != math.Round(b) {
			t.Errorf("bob %.2f not a whole 0..3", b)
		}
	}
	if len(got) < 3 {
		t.Errorf("the arrow only takes %d heights (%v), start %.0f", len(got), got, b0)
	}
}

// "ÚLTIMO" / "LAST" only when ONE person is left, with a bigger arrow, whole-number text scale, in both languages.
func TestLastLabelOnlyWithOne(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		g := hudPlay(t, false)
		gameLang, editorLang = lang, lang
		prevUI := uiLang
		t.Cleanup(func() { uiLang = prevUI })
		uiLang = lang
		img := ebiten.NewImage(screenW, screenH)
		defer img.Deallocate()
		for n := 3; n >= 1; n-- {
			keep := leaveAlive(g, n)
			for i, gu := range keep {
				gu.pos = g.lvl.NearestFree(Vec{200 + float64(i)*200, 320})
			}
			textRec = &textRecorder{}
			lastDrawAll(g, img)
			rec := textRec
			textRec = nil
			var labels []textRect
			for _, tx := range rec.texts {
				if tx.s == "ÚLTIMO" || tx.s == "LAST" {
					labels = append(labels, tx)
				}
			}
			if n == 1 {
				want := map[string]string{"es": "ÚLTIMO", "en": "LAST"}[lang]
				if len(labels) != 1 || tr(labels[0].s) != want && labels[0].s != want {
					t.Fatalf("%s n=1: labels %+v, want one %q", lang, labels, want)
				}
				if labels[0].scale != math.Floor(labels[0].scale) {
					t.Errorf("label scale %.2f is not whole", labels[0].scale)
				}
				m := g.lastMarks[0]
				if !m.label || m.half != lastArrowOne {
					t.Errorf("one left: %+v, want the big arrow with label", m)
				}
				if labels[0].y < m.ry-2 || labels[0].y > m.tipY {
					t.Errorf("label at y=%.0f outside the arrow column %.0f..%.0f", labels[0].y, m.ry, m.tipY)
				}
			} else if len(labels) != 0 {
				t.Errorf("%s n=%d: a label with several people left", lang, n)
			}
		}
	}
}

// The marker has priority over the emotes: an emote that would cover the arrow is not drawn.
func TestLastArrowBeatsEmote(t *testing.T) {
	g := emoteGame(t, false)
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	g.first().pos = Vec{80, 450}
	gu := &Guest{pos: g.lvl.NearestFree(Vec{600, 300}), state: gAlarm}
	g.guests = []*Guest{gu}
	g.guards, g.cops = nil, nil
	for _, s := range g.allSlashers() {
		s.state = slGone
	}
	g.emo.slots[0] = emoteSlot{gu: gu, kind: emExcl, t: 0.5}
	g.drawEmotes(img, nil)
	if g.emo.nDrawn != 1 {
		t.Skip("setup: the emote is not drawn even without the marker")
	}
	g.drawLastRings(img) // 1 alive: marker on
	if len(g.lastMarks) != 1 {
		t.Fatalf("no marker (%d alive)", g.aliveCount())
	}
	g.drawEmotes(img, nil)
	if g.emo.nDrawn != 0 {
		t.Error("the emote was drawn over the last-guest arrow")
	}
}

// The minimap dot of a last guest is 2x2 and blinks gold / white.
func TestLastMinimapBlink(t *testing.T) {
	mmRestore(t)
	g := hudPlay(t, false)
	keep := leaveAlive(g, 1)
	keep[0].pos = g.lvl.NearestFree(Vec{300, 300})
	g.first().pos = Vec{80, 450}
	var cols [2][4]uint8
	for i, tm := range []float64{0.0, 1.0 / 6} {
		g.time = tm
		g.minimapFrame()
		cols[i] = mmAt(g, keep[0].pos)
		x, y := g.mm.pos(keep[0].pos)
		for _, d := range [][2]int{{1, 0}, {0, 1}, {1, 1}} {
			i2 := ((y+d[1])*mmW + x + d[0]) * 4
			if [4]uint8{g.mm.buf[i2], g.mm.buf[i2+1], g.mm.buf[i2+2], g.mm.buf[i2+3]} != cols[i] {
				t.Errorf("the dot is not 2x2 (t=%.2f)", tm)
			}
		}
	}
	if cols[0] == cols[1] || cols[0] != mmLastB || cols[1] != mmLastA {
		t.Errorf("no gold/white blink: %v", cols)
	}
}

// No allocations of its own per frame: the cost is that of the plain draw calls.
func TestLastMarkerAllocs(t *testing.T) {
	g := hudPlay(t, false)
	keep := leaveAlive(g, 3)
	for i, gu := range keep {
		gu.pos = g.lvl.NearestFree(Vec{200 + float64(i)*200, 320})
	}
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	lastDrawAll(g, img)
	one := testing.AllocsPerRun(30, func() { fillRect(img, 10, 10, 5, 1, lastGold) })
	tg := g.lastTargets()
	if len(tg) != 3 {
		t.Fatalf("%d targets", len(tg))
	}
	got := testing.AllocsPerRun(30, func() { g.time += 0.0167; lastDrawAll(g, img) })
	// rings: ~45 runs per ring (dark/gold/pale rows) and ~20 rects of arrow per person; nothing but DrawImage allocates
	if limit := one * 3 * 200; got > limit {
		t.Errorf("%.0f allocs per frame, limit %.0f (a fillRect costs %.0f)", got, limit, one)
	}
	t.Logf("allocs per frame %.0f, one fillRect %.0f", got, one)
	if a := testing.AllocsPerRun(30, func() { g.lastTargets() }); a != 0 {
		t.Errorf("lastTargets allocates %.0f", a)
	}
}

// It must not collide with the HUD panel, and nothing panics with 1 or 2 players.
func TestLastMarkerClearOfHUDAndNoPanic(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := hudPlay(t, two)
		keep := leaveAlive(g, 1)
		keep[0].pos = g.lvl.NearestFree(Vec{300, 300})
		img := ebiten.NewImage(screenW, screenH)
		defer img.Deallocate()
		rec := hudRecord(g)
		lastDrawAll(g, img)
		m := g.lastMarks[0]
		for _, p := range rec.panels {
			if rectsHit(m.rx, m.ry, m.rw, m.rh, p.x, p.y, p.w, p.h) {
				t.Errorf("two=%v: marker %+v over the HUD panel %+v", two, m, p)
			}
		}
		for _, pos := range []Vec{{5, 5}, {screenW - 5, 5}, {-300, 300}, {5000, 5000}} { // corners and far away
			keep[0].pos = pos
			lastDrawAll(g, img)
		}
	}
}

func TestLastGuestsDemoFlagKeepsN(t *testing.T) {
	g := hudPlay(t, false)
	prev := dbgLastGuests
	t.Cleanup(func() { dbgLastGuests = prev })
	dbgLastGuests = 2
	g.time = 1
	g.lastGuestsDemo()
	if g.aliveCount() != 2 || dbgLastGuests != -1 {
		t.Errorf("alive %d (want 2), flag %d (want -1: once)", g.aliveCount(), dbgLastGuests)
	}
}
