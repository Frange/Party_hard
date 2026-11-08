package main

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// the ghost look: translucent at the head, melting towards the tail, a wavier tail than head
func TestGhostLookNumbers(t *testing.T) {
	base := 0.7
	if a := ghostAlphaAt(0, base); a < ghostAlphaMin || a > ghostAlphaMax+0.1 {
		t.Fatalf("a ghost head must be translucent (%.2f..%.2f), got %.2f", ghostAlphaMin, ghostAlphaMax, a)
	}
	prev := 2.0
	for f := 0.0; f <= 1.0; f += 0.05 {
		a := ghostAlphaAt(f, base)
		if a > prev+1e-9 {
			t.Fatalf("the ghost must only melt downwards: alpha grows at %.2f", f)
		}
		prev = a
	}
	if ghostAlphaAt(0.95, base) > 0.15*base {
		t.Fatalf("no feet: the bottom of the body must be almost gone, got %.2f", ghostAlphaAt(0.95, base))
	}
	var head, tail float64
	for i := 0; i < 200; i++ {
		tt := float64(i) * 0.05
		if v := abs64(ghostWave(0.1, tt, 0)); v > head {
			head = v
		}
		if v := abs64(ghostWave(0.95, tt, 0)); v > tail {
			tail = v
		}
	}
	if tail < 2*head {
		t.Fatalf("the tail must ripple more than the head (%.2f vs %.2f)", tail, head)
	}
}

func abs64(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// layerGame: the full headless game of the audit (fonts, glow, white pixel), set for one or two players.
func layerGame(t *testing.T, two bool) *Game {
	g := auditGame(t)
	pm, pd := twoPlayerMode, dbgTwoPlayers
	t.Cleanup(func() { twoPlayerMode, dbgTwoPlayers = pm, pd })
	auditReset(g)
	twoPlayerMode, dbgTwoPlayers = two, two
	if g.as.playerCS == nil {
		spec := presetByName("guybrush")(rand.New(rand.NewSource(3)))
		spec.noOverride = true
		g.as.playerCS = newCharSet(&spec)
	}
	return g
}

// Every scene paints sky, buildings and ground BEFORE the people. Whatever is drawn in front of them must say
// so with the layer prefix "front:". Here the street of both endings, one and two players, every stage
// and several moments (the neighbour's house was drawn over the ghosts and the hero).
func TestScenesLayerOrder(t *testing.T) {
	defer func() { twoPlayerMode, dbgTwoPlayers, layerRec = false, false, nil }()
	for _, two := range []bool{false, true} {
		for _, secret := range []bool{false, true} {
			for _, stage := range []int{endWalk, endDoor} {
				for _, tt := range []float64{0.2, 1.5, 3.2, 6.5, 10} {
					g := layerGame(t, two)
					e := &endState{stage: stage, trueEnd: secret, heroX: 600, t: tt, total: tt + 20}
					g.end = e
					var rec []string
					layerRec = &rec
					c := ebiten.NewImage(endCW, endCH)
					g.endWorld(c, 300, tt, e)
					c.Deallocate()
					layerRec = nil
					name := strings.Join([]string{map[bool]string{false: "1P", true: "2P"}[two], map[bool]string{false: "normal", true: "secret"}[secret]}, "/")
					firstActor := -1
					for i, l := range rec {
						if strings.HasPrefix(l, "actor:") && firstActor < 0 {
							firstActor = i
						}
						if strings.HasPrefix(l, "bg:") && firstActor >= 0 {
							t.Errorf("%s stage %d t=%.1f: %q is drawn AFTER an actor (%q): %v", name, stage, tt, l, rec[firstActor], rec)
						}
					}
					if firstActor < 0 {
						t.Errorf("%s stage %d t=%.1f: no actor was drawn at all: %v", name, stage, tt, rec)
					}
					if stage == endDoor && tt < 3 {
						has := false
						for _, l := range rec {
							has = has || l == "bg:neighbour"
						}
						if !has {
							t.Errorf("%s door t=%.1f: the neighbour's house is not drawn: %v", name, tt, rec)
						}
					}
					if secret && stage == endWalk && tt > 3 {
						ghosts := 0
						for _, l := range rec {
							if l == "actor:ghost" {
								ghosts++
							}
						}
						if ghosts < 5 {
							t.Errorf("%s secret walk t=%.1f: the conga must be drawn with drawGhostActor, got %d ghosts", name, tt, ghosts)
						}
					}
				}
			}
		}
	}
}

// the "¡UUUH!" of a ghost at the screen edge must never be cut off
func TestGhostWailStaysOnScreen(t *testing.T) {
	for _, w := range []float64{30, 45, 70} { // any plausible width of the bubble
		for x := -30.0; x <= endCW+30; x++ {
			l := ghostWailX(x, w, endCW)
			if l < 0 || l+w > endCW {
				t.Fatalf("wail at ghost x=%.0f spans %.0f..%.0f, outside 0..%d", x, l, l+w, endCW)
			}
		}
	}
}

// -scenes 2 must pretend that every scene was seen (none locked)
func TestScenesDebugSeesAll(t *testing.T) {
	old := progressFrozen
	progressFrozen = true
	defer func() { progressFrozen = old }()
	m := debugAllSeen()
	for i, s := range sceneList {
		if m[s.key] == 0 {
			t.Fatalf("scene %d (%s) is not marked as seen by -scenes 2", i, s.key)
		}
	}
}
