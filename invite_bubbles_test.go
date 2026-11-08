package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type invBox struct{ x, y, w, h float64 }

func (a invBox) hits(b invBox) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

// The balloons of the two players in the letter: only one at a time, each one for its own time only,
// and never over the paper's text, the captions, the hint or the players themselves.
func TestInviteTwoBubblesNeverCover(t *testing.T) {
	g := layerGame(t, true)
	ul := uiLang
	t.Cleanup(func() { uiLang, textRec = ul, nil })
	g.sceneReplay = true
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for phase := invArrive; phase <= invBoom; phase++ {
			d := invTalks[phase]
			// 1) the turns: no two lines of a phase overlap in time, each one is inside the cycle
			for i, a := range d.l {
				if a.to <= a.from || (d.cycle > 0 && a.to > d.cycle) {
					t.Errorf("phase %d line %d: bad window [%v,%v)", phase, i, a.from, a.to)
				}
				for j, b := range d.l {
					if i < j && a.from < b.to && b.from < a.to {
						t.Errorf("phase %d: %q and %q talk at the same time", phase, a.s, b.s)
					}
				}
			}
			// 2) each instant of the phase: geometry
			for tt := 0.0; tt < 12.0; tt += 0.1 {
				who, s := inviteTwoSpeaker(phase, tt)
				if d.cycle == 0 && tt > 11 && s != "" {
					t.Errorf("phase %d t=%.1f: %q is still on after its time", phase, tt, s)
				}
				if who < 0 {
					continue
				}
				x, y, w, h := g.inviteBubbleRect(who, s)
				b := invBox{x - 2, y - 2, w + 4, h + 4}
				if b.x < 0 || b.x+b.w > screenW || b.y+b.h > screenH {
					t.Errorf("%s phase %d t=%.1f: %q leaves the screen %v", lang, phase, tt, s, b)
				}
				if b.hits(invBox{invPX, invPY, invPW, invPH}) {
					t.Errorf("%s phase %d t=%.1f: %q covers the paper %v", lang, phase, tt, s, b)
				}
				// the players, with their hops, the run of the red one and the lean (x +-26 around 56 and 904,
				// 34 px more to the left for the red one): the balloon (tail included) never enters their column
				tb := invBox{b.x - 6, b.y, b.w + 12, b.h}
				for p, col := range []invBox{{56 - 26 - 8, 400, 60, 120}, {904 - 26 - 36, 400, 52 + 36 + 8, 120}} {
					if tb.hits(col) {
						t.Errorf("%s phase %d t=%.1f: %q covers player %d %v", lang, phase, tt, s, p, tb)
					}
				}
			}
		}
	}
	// 3) real drawing: nothing else recorded on the screen is under a balloon
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for _, c := range []struct {
			phase int
			t     float64
		}{{invWait, 1.5}, {invWait, 5}, {invOpen, 0.4}, {invOpen, 1.5}, {invRead, 1}, {invRead, 3.5}, {invRead, 6}, {invRead, 9}, {invCount, 0.5}, {invCount, 1.5}, {invBoom, 1.0}, {invBoom, 2.2}} {
			e := &endState{stage: endInvite, total: 20 + c.t}
			e.x.inv = inviteState{phase: c.phase, t: c.t}
			who, s := inviteTwoSpeaker(c.phase, c.t)
			if who < 0 {
				continue
			}
			g.end = e
			textRec = &textRecorder{}
			img := ebiten.NewImage(screenW, screenH)
			g.drawInvite(img, e)
			rec := textRec
			textRec = nil
			img.Deallocate()
			bx, by, bw, bh := g.inviteBubbleRect(who, s)
			bub := invBox{bx - 2, by - 2, bw + 4, bh + 4}
			found := 0
			for _, tx := range rec.texts {
				r := invBox{tx.x, tx.y, tx.w, tx.h}
				if tx.s == tr(s) {
					found++
					continue
				}
				if r.hits(bub) {
					t.Errorf("%s phase %d t=%.1f: the balloon %q covers the text %q", lang, c.phase, c.t, s, tx.s)
				}
			}
			if found != 1 {
				t.Errorf("%s phase %d t=%.1f: balloon %q drawn %d times (want 1)", lang, c.phase, c.t, s, found)
			}
			// the paper's own text
			sc := g.inviteScale()
			lines, _ := g.inviteLayout(sc)
			for _, l := range lines {
				r := invBox{invPX + l.x, invPY + l.y, g.textW(l.s, l.scale), 9 * l.scale}
				if r.hits(bub) {
					t.Errorf("%s: the balloon %q covers the letter line %q", lang, s, l.s)
				}
			}
		}
	}
}
