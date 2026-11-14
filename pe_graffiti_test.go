package main

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// gfGame: campaign level idx loaded (1 or 2 players) with the street painter switched on for the test and its dice
// pinned; every global of the event is put back afterwards.
func gfGame(t *testing.T, idx int, two bool, seed int64) *Game {
	t.Helper()
	prev := [...]any{gfInTests, gfRng, gfSeed}
	prevUsed := gfUsed
	t.Cleanup(func() {
		gfInTests, gfRng, gfSeed = prev[0].(bool), prev[1].(*rand.Rand), prev[2].(int64)
		gfLv, gfUsed = gfLevel{}, prevUsed
	})
	g := peLoad(t, idx, two)
	gfInTests, gfRng, gfSeed, gfUsed = true, nil, seed*7717+int64(idx), map[int]bool{}
	gfLv = gfLevel{}
	gfLevelOf(g)
	gfLv.want, gfLv.next, gfLv.max = true, 0, 3
	return g
}

// gfExpected: the campaign parties with a free street-facing house wall (measured from the walls, see gfSegments).
var gfExpected = []string{"house", "neon", "roof", "eiffel", "disco", "cinema", "casino", "cake", "disco70", "skate"} // skate: its graffiti walls (v1.4.0, skategraffiti.go)

func TestGraffitiWallsOnlyInTheExpectedLevels(t *testing.T) {
	g := partyGame(t)
	var got []string
	for i := range campaignLevels {
		g.loadLevel(i)
		segs := gfSegments(g.lvl)
		if len(segs) == 0 {
			continue
		}
		got = append(got, g.lvl.Def.Icon)
		for _, s := range segs {
			for x := s.x0; x <= s.x1; x += 4 {
				p := Vec{x, s.bottom + gfFloorGap}
				if !g.lvl.Free(p) || g.lvl.SolidP(p.X, p.Y) || !g.lvl.Outdoor(p) {
					t.Errorf("%s: the painter's strip under the wall at %.0f,%.0f is not free outdoor floor", g.lvl.Def.Title, p.X, p.Y)
				}
			}
			if s.avail < 9 || s.r.Y < 0 {
				t.Errorf("%s: a face of %d rows at y %.0f", g.lvl.Def.Title, s.avail, s.r.Y)
			}
		}
	}
	sort.Strings(got)
	want := append([]string(nil), gfExpected...)
	sort.Strings(want)
	if len(got) < 8 || len(got) > 12 || len(got) != len(want) {
		t.Fatalf("levels with a wall for the painter: %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("levels with a wall for the painter: %v, want %v", got, want)
		}
	}
}

// gfRun steps the party until the running painter is done (or max seconds). Returns the state seen last.
func gfRun(g *Game, max float64, each func(s *gfState)) *gfState {
	var last *gfState
	for k := 0; k < int(max*30); k++ {
		peStep(g, 1.0/30, true)
		a := g.partyRunning("graffiti")
		if a == nil {
			if last != nil {
				return last
			}
			continue
		}
		last = a.D.(*gfState)
		if each != nil {
			each(last)
		}
	}
	return last
}

// gfCheckPiece: the piece is on the face of a free wall, clear of doors, props, cameras, pictures and other pieces.
func gfCheckPiece(t *testing.T, g *Game, p *gfPiece) string {
	rc := p.rect()
	var seg *gfSeg
	for i, s := range gfLv.segs {
		if rc.X >= s.x0-1 && rc.X+rc.W <= s.r.X+s.r.W && rc.Y+rc.H == s.bottom && p.art.h <= s.avail {
			seg = &gfLv.segs[i]
		}
	}
	if seg == nil {
		return "the piece is not on the face of a wall"
	}
	for _, d := range g.lvl.Def.Doors {
		if rc.Hits(sr(doorFoot(d))) {
			return "on a door"
		}
	}
	for _, pr := range g.lvl.Props {
		w, h := float64(pr.K.W), float64(pr.K.H)
		if rc.Hits(Rect{pr.X - w/2, pr.Y - h, w, h}) {
			return "behind the prop " + pr.Name
		}
	}
	for _, a := range seg.avoid {
		if rc.Hits(a) {
			return "on a picture or a window"
		}
	}
	for _, c := range g.cams {
		if rc.Grow(4).Has(c.mount.X, c.mount.Y) {
			return "on a camera"
		}
	}
	for _, q := range gfLv.pieces {
		if q != p && rc.Hits(q.rect()) {
			return "on another piece"
		}
	}
	return ""
}

// A painter walks to the wall, paints (outline then colours) and leaves; the piece stays, finished, for the rest of the
// party; at most 3 per party, never the same design twice, never on a door, prop, picture, camera or other piece; nobody
// is held at the end and nobody is added.
func TestGraffitiPaintsAndThePieceStays(t *testing.T) {
	for _, icon := range []string{"house", "cinema", "eiffel", "disco70"} {
		idx := campaignIdx(icon)
		okAny := ""
		for seed := int64(1); seed <= 4; seed++ {
			g := gfGame(t, idx, false, seed)
			n0 := len(g.guests)
			touched := map[*Guest]bool{}
			done := 0
			msg := ""
			for round := 0; round < 3 && msg == ""; round++ {
				gfLv.next = g.time
				var painted, outlineFirst bool
				s := gfRun(g, 70, func(s *gfState) {
					touched[s.m.gu] = true
					if s.phase == gfPaint && s.piece.prog > 0.2 && s.piece.prog < 0.4 {
						outlineFirst = true
					}
					if s.piece.prog >= 1 {
						painted = true
					}
				})
				if s == nil {
					msg = "no painter started"
					break
				}
				if !painted || !outlineFirst {
					continue // a fright or the party took him: that one is half done, the next may finish
				}
				done++
				if why := gfCheckPiece(t, g, s.piece); why != "" {
					msg = icon + ": " + why
				}
			}
			if len(gfLv.pieces) > 3 {
				t.Fatalf("%s: %d pieces in one party", icon, len(gfLv.pieces))
			}
			seen := map[*gfArt]bool{}
			for _, p := range gfLv.pieces {
				if seen[p.art] {
					t.Errorf("%s: %s painted twice", icon, p.art.name)
				}
				seen[p.art] = true
			}
			// the party goes on: the pieces stay where they are
			before := len(gfLv.pieces)
			gfLv.want = false
			for k := 0; k < 30*20; k++ {
				peStep(g, 1.0/30, true)
			}
			if len(gfLv.pieces) != before {
				t.Errorf("%s: the pieces changed after the painters (%d -> %d)", icon, before, len(gfLv.pieces))
			}
			if gu := peStale(g, touched); gu != nil {
				t.Errorf("%s: a painter is still held", icon)
			}
			if len(g.guests) != n0 {
				t.Errorf("%s: the painter added guests (%d -> %d)", icon, n0, len(g.guests))
			}
			if msg == "" && done > 0 {
				t.Logf("%s, party %d: %d pieces finished, %d on the walls, t=%.0f s", icon, seed, done, len(gfLv.pieces), g.time)
				okAny = ""
				break
			}
			okAny = msg
			if msg == "" {
				okAny = "no piece was finished"
			}
			t.Logf("%s, party %d: %s", icon, seed, okAny)
		}
		if okAny != "" {
			t.Errorf("%s: %s", icon, okAny)
		}
	}
}

// Killed while painting: he drops the can (it drips), the piece stays half done, the event ends.
func TestGraffitiKilledHalfway(t *testing.T) {
	idx := campaignIdx("cinema")
	for seed := int64(1); seed <= 5; seed++ {
		g := gfGame(t, idx, false, seed)
		var killed *gfState
		var at float64
		gfRun(g, 60, func(s *gfState) {
			if killed == nil && s.phase == gfPaint && s.piece.prog > 0.5 {
				killed, at = s, s.piece.prog
				g.killGuest(s.m.gu, false)
			}
		})
		if killed == nil {
			t.Logf("party %d: the painter never got halfway", seed)
			continue
		}
		if g.partyRunning("graffiti") != nil {
			t.Fatal("the event goes on with a dead painter")
		}
		if killed.piece.prog != at || killed.piece.prog >= 1 {
			t.Errorf("the half piece went from %.2f to %.2f", at, killed.piece.prog)
		}
		found := false
		for _, p := range gfLv.pieces {
			found = found || p == killed.piece
		}
		if !found {
			t.Error("the half piece was wiped")
		}
		if len(gfLv.cans) != 1 {
			t.Errorf("%d cans on the floor, want 1", len(gfLv.cans))
		}
		return
	}
	t.Error("no painter got halfway in five parties")
}

// No painter where there is no free wall, and nothing in stealth or in the island (the frame never runs there).
func TestGraffitiNeverWithoutAWall(t *testing.T) {
	for _, icon := range []string{"beach", "festival", "pirate"} { // (the skate park has graffiti walls since v1.4.0)
		g := gfGame(t, campaignIdx(icon), false, 1)
		if len(gfLv.segs) != 0 {
			t.Errorf("%s: walls for the painter %v", icon, gfLv.segs)
		}
		for k := 0; k < 30*10; k++ {
			peStep(g, 1.0/30, true)
			if g.partyRunning("graffiti") != nil {
				t.Fatalf("%s: a painter without a wall", icon)
			}
		}
	}
}

// With two players the painter works the same (and the second player's guests do not change anything).
func TestGraffitiTwoPlayers(t *testing.T) {
	idx := campaignIdx("casino")
	for seed := int64(1); seed <= 4; seed++ {
		g := gfGame(t, idx, true, seed)
		if g.p2 == nil {
			t.Fatal("no second player")
		}
		n0 := len(g.guests)
		finished := false
		gfRun(g, 70, func(s *gfState) { finished = finished || s.piece.prog >= 1 })
		if len(g.guests) != n0 {
			t.Fatalf("guests %d -> %d", n0, len(g.guests))
		}
		if finished {
			return
		}
	}
	t.Error("with two players no piece was finished in four parties")
}

// The drawing runs (pieces, half pieces, the can, the painter's kit and cloud) without trouble.
func TestGraffitiDraws(t *testing.T) {
	g := gfGame(t, campaignIdx("house"), false, 2)
	drawn := false
	gfRun(g, 60, func(s *gfState) {
		if s.phase == gfPaint && !drawn {
			peDraw(g)
			drawn = true
		}
	})
	gfLv.cans = append(gfLv.cans, gfCan{pos: Vec{300, 400}, t0: g.time - 1, col: pevWhite})
	scr := ebiten.NewImage(screenW, screenH)
	g.drawGraffiti(scr)
	if !drawn {
		t.Error("the painter was never drawn")
	}
}
