package main

import (
	"fmt"
	"strings"
	"testing"
)

// paperTextDump: every visible text of a page (headline, story, quote, caption, photo, small news, stats), one string.
func paperTextDump(d *paperData) string {
	var sb strings.Builder
	fmt.Fprintln(&sb, d.head, "|", d.dek, "|", d.photoVar, "|", d.caption, "|", d.quote)
	for _, s := range d.story {
		fmt.Fprintln(&sb, "S:", s)
	}
	for _, b := range d.briefs {
		fmt.Fprintln(&sb, "B:", b.head, b.es)
	}
	fmt.Fprintln(&sb, d.stats, d.score, d.total)
	return sb.String()
}

// TestPaperStableWhileShown: the paper of an ended party must not change while it is on screen (the bug: the
// headline and the news kept changing). The party ends, then the newspaper screen runs for many ticks.
func TestPaperStableWhileShown(t *testing.T) {
	for _, two := range []bool{false, true} {
		for _, challenge := range []bool{false, true} {
			t.Run(fmt.Sprint("two=", two, "/challenge=", challenge), func(t *testing.T) {
				old := paperSession
				t.Cleanup(func() { paperSession = old })
				paperSession = &paperSessionT{rng: newTestRand(3)}
				g := scoreGame(t)
				g.challenge = challenge
				if two {
					g.p1 = g.p
					g.p2 = &Player{idx: 1, pos: g.p.pos, cs: g.p.cs}
					t.Cleanup(func() { g.p1, g.p2 = nil, nil })
				}
				for i := 0; i < 4; i++ {
					if gu := liveGuest(g, 0); gu != nil {
						g.killGuest(gu, true)
					}
				}
				g.time = 30
				g.life.ready = true
				g.life.contestAt, g.life.pizzaAt = 40, 45 // planned for later: they never happened in this party
				g.completeLevel()
				if g.mode != ModeLevelDone {
					g.mode = ModeLevelDone
				}
				g.fireworksArm(true) // this party ends with fireworks (shown on the first frames of the screen)
				paperC = paperCache{}
				first := paperTextDump(g.paperDataNow())
				frozen := paperDataKey(g.paperFrozen())
				_, lay0 := renderPaper(g.paperFrozen(), 0)
				for tick := 0; tick < 3*60*60; tick++ { // three minutes of newspaper on screen
					g.time += 1.0 / 60
					g.tick++
					g.updateParticles(1.0 / 60)
					g.fireworksUpdate(1.0 / 60)
					if tick%7 != 0 {
						continue
					}
					if now := paperTextDump(g.paperDataNow()); now != first {
						t.Fatalf("tick %d: the page changed\nBEFORE:\n%s\nNOW:\n%s", tick, first, now)
					}
					if paperDataKey(g.paperFrozen()) != frozen {
						t.Fatalf("tick %d: the frozen page changed", tick)
					}
				}
				f := g.paperFactsNow()
				if f.events["contest"] || f.events["pizza"] || !f.events["fireworks"] {
					t.Errorf("events of the paper: %v (contest/pizza never started; fireworks armed)", f.events)
				}
				_, lay1 := renderPaper(g.paperFrozen(), 0)
				if fmt.Sprint(lay0.texts) != fmt.Sprint(lay1.texts) {
					t.Errorf("the drawn texts changed after three minutes")
				}
			})
		}
	}
}

// TestPaperSameStateSamePage: rebuilding the page from the same facts gives the same page (no hidden dice).
func TestPaperSameStateSamePage(t *testing.T) {
	for _, f := range paperCases() {
		for _, two := range []bool{false, true} {
			a := paperTextDump(paperSample(f, two, two, 9))
			for i := 0; i < 20; i++ {
				if b := paperTextDump(paperSample(f, two, two, 9)); b != a {
					t.Fatalf("same facts, different page:\n%s\n---\n%s", a, b)
				}
			}
		}
	}
}
