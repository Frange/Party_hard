package main

import (
	"math"
	"testing"
)

// useTextBox: the box ("[E] Esconderse" plus 6 px of padding each side) a prop's use text is drawn in,
// in world coordinates, as drawUseMarksFor places it.
func useTextBox(g *Game, pr *Prop, lab string) (x0, y0, x1, y1 float64) {
	top := float64(pr.K.H) * 0.62
	if pr.K.NoSolid && pr.K.H < 30 {
		top = float64(pr.K.H) + 4
	}
	w := g.textW("[E] "+lab, 1) + 12
	cy := pr.Y - top - 8 - 16
	return pr.X - w/2, cy - 6, pr.X + w/2, cy + 11 + 6
}

// The use texts of the props of every party never pile up: wherever a player stands only the nearest
// prop within reach writes its "[E] ..." (the one E acts on), so two booths or bushes side by side do not
// overlap their texts (the user's rule of margins), and the text of each one stays inside the level.
func TestPromptOnlyNearestUseTextEveryLevel(t *testing.T) {
	defs := testLevels(t)
	face := auditGame(t).face
	oldLang := uiLang
	t.Cleanup(func() { uiLang = oldLang })
	oldOverlaps := 0
	for _, lang := range []string{"es", "en"} {
		uiLang = lang
		for li, def := range defs {
			g := hideGameOn(t, def, false)
			g.face = face
			var ps []*Prop
			for _, pr := range g.lvl.Props {
				if lab, _ := g.useLabel(pr); lab != "" {
					ps = append(ps, pr)
				}
			}
			for _, pr := range ps {
				lab, _ := g.useLabel(pr)
				w := g.textW("[E] "+lab, 1)
				cx := clampUseTextX(pr.X, w)
				if g.lvl.W <= screenW && (cx-w/2 < 16 || cx+w/2 > screenW-16) {
					t.Errorf("L%d %s: the text of %s at %.0f,%.0f leaves the screen margin (x %.0f..%.0f)", li+1, def.Title, pr.Name, pr.X, pr.Y, cx-w/2, cx+w/2)
				}
				if pr.X-w/2 >= 16 && pr.X+w/2 <= screenW-16 && cx != math.Round(pr.X) {
					t.Errorf("clamp moved a text that already fits")
				}
			}
			for _, a := range ps {
				spots := []Vec{{a.X, a.Y + 8}, {a.X, a.Y - 4}}
				for _, b := range ps {
					if a != b {
						spots = append(spots, Vec{(a.X + b.X) / 2, (a.Y+b.Y)/2 + 8}, Vec{(a.X + b.X) / 2, (a.Y + b.Y) / 2})
					}
				}
				for _, s := range spots {
					g.p.pos = s
					var near []*Prop
					for _, pr := range ps {
						if distRect(s, pr.Foot) < 30 {
							near = append(near, pr)
						}
					}
					w := g.useTextProp()
					if (len(near) == 0) != (w == nil) {
						t.Fatalf("L%d at %v: %d props in reach but winner %v", li+1, s, len(near), w != nil)
					}
					if w == nil {
						continue
					}
					for _, pr := range near {
						if pr != w && distRect(s, pr.Foot) < distRect(s, w.Foot) {
							t.Fatalf("L%d at %v: the winner is not the nearest", li+1, s)
						}
					}
					// the old rule drew all of "near": did their boxes touch?
					for i := range near {
						for j := i + 1; j < len(near); j++ {
							la, _ := g.useLabel(near[i])
							lb, _ := g.useLabel(near[j])
							ax0, ay0, ax1, ay1 := useTextBox(g, near[i], la)
							bx0, by0, bx1, by1 := useTextBox(g, near[j], lb)
							if ax0 < bx1+12 && bx0 < ax1+12 && ay0 < by1+12 && by0 < ay1+12 {
								oldOverlaps++
							}
						}
					}
				}
			}
		}
	}
	if oldOverlaps == 0 {
		t.Errorf("no level has booths close enough to have piled their texts: the test lost its coverage")
	}
	t.Logf("%d spots where the old rule would have piled texts", oldOverlaps)
}
