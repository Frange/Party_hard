package main

import (
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Tests of the diving board (poolboard.go): the prop only where there are pools, never blocking anything, and the
// "sometimes" jump with its splash.

// pbLevels: the campaign levels with pools (the island is not in the campaign list).
func pbLevels(t *testing.T) []int {
	t.Helper()
	var out []int
	for i := range campaignLevels {
		if i < len(levelDefs) && len(levelDefs[i].Pools) > 0 {
			out = append(out, i)
		}
	}
	if len(out) < 2 {
		t.Fatalf("expected the two campaign parties with a pool, got %v", out)
	}
	return out
}

func TestPoolBoardOnlyWherePoolsAndNeverBlocks(t *testing.T) {
	partyGame(t)
	prevVip := vipDecision
	t.Cleanup(func() { vipDecision, divingOff = prevVip, false })
	vipDecision = ""
	pools := map[int]bool{}
	for _, i := range pbLevels(t) {
		pools[i] = true
	}
	for i, d := range levelDefs {
		if d == nil || d.Title == "" {
			continue
		}
		divingOff = false
		l := buildLevel(d)
		b := l.divingBoard()
		if len(d.Pools) == 0 || d.PropScale > 0 && d.PropScale != 1 {
			if b != nil {
				t.Errorf("level %d %s: a diving board without a pool", i, d.Title)
			}
			continue
		}
		if b == nil {
			if pools[i] {
				t.Errorf("level %d %s: a campaign pool without its diving board", i, d.Title)
			}
			continue
		}
		// the pedestal on dry ground, the plank and the landing over the water
		for _, q := range []Vec{{b.Foot.X, b.Foot.Y}, {b.Foot.X + b.Foot.W, b.Foot.Y + b.Foot.H}, {b.X, b.Y - 1}} {
			if l.InPool(q) {
				t.Errorf("level %d: the pedestal stands in the water at %v", i, q)
			}
		}
		if b.Foot.W != 12 || b.Foot.H != 7 {
			t.Errorf("level %d: the solid part must be only the pedestal, got %v", i, b.Foot)
		}
		for u := 0.0; u <= 1; u += 0.1 {
			if q := pbBoardAt(b, u); !l.InPool(q) || !l.Free(q) || l.SolidP(q.X, q.Y) {
				t.Errorf("level %d: plank point %v is not open water", i, q)
			}
		}
		if land := pbLandAt(b); !l.InPool(land) || !l.Free(land) {
			t.Errorf("level %d: he would land out of the water at %v", i, land)
		}
		reach := l.reachFrom(d.Start)
		for _, q := range []Vec{pbStandAt(b), pbQueueAt(b), pbLandAt(b)} {
			if !reach[l.navIdx(q)] || !l.Free(q) {
				t.Errorf("level %d: %v cannot be reached", i, q)
			}
		}
		// nothing that could be reached before is cut off
		divingOff = true
		bare := buildLevel(d)
		r0 := bare.reachFrom(d.Start)
		lost := 0
		for k := range r0 {
			if r0[k] && !reach[k] && !l.blocked[k] {
				lost++
			}
		}
		if lost > 0 {
			t.Errorf("level %d: the board cuts off %d walkable cells", i, lost)
		}
		t.Logf("level %d %s: board at %.0f,%.0f var %d", i, d.Title, b.X, b.Y, b.Var)
	}
}

func TestPoolBoardSpriteIsMirroredAndCrisp(t *testing.T) {
	k := propKinds[pbName]
	a, b := newPix(k.W, k.H), newPix(k.W, k.H)
	k.Draw(a, 0)
	k.Draw(b, 1)
	opaque := 0
	for y := 0; y < k.H; y++ {
		for x := 0; x < k.W; x++ {
			ca, cb := a.get(x, y), b.get(k.W-1-x, y)
			if ca != cb {
				t.Fatalf("var 1 is not the mirror of var 0 at %d,%d", x, y)
			}
			if ca.A != 0 && ca.A != 255 {
				t.Fatalf("half-transparent pixel at %d,%d", x, y)
			}
			if ca.A == 255 {
				opaque++
			}
		}
	}
	if opaque < 200 {
		t.Errorf("the board is almost empty (%d pixels)", opaque)
	}
	// the pedestal is the middle of the sprite: the default footprint (bottom centre) is on it
	if c := a.get(k.W/2, k.H-2); c.A != 255 {
		t.Error("no pedestal under the anchor")
	}
}

func TestPoolBoardEventOnlyInPoolParties(t *testing.T) {
	g := partyGame(t)
	e := partyByID("poolboard")
	if e == nil {
		t.Fatal("not registered")
	}
	in := peEligible(g, "poolboard")
	want := pbLevels(t)
	if len(in) != len(want) {
		t.Errorf("eligible %v, want the pool parties %v", in, want)
	}
	if e.Prob > 0.40 || e.Cooldown < 120 || e.Big {
		t.Errorf("it must be occasional, with a long cooldown: %+v", e)
	}
}

// pbRun: runs the jump until it ends; it checks every frame that nobody of the event is inside a solid thing and that
// there are never more than two jumpers.
func pbRun(t *testing.T, g *Game, frames int, each func(s *poolBoardState)) *poolBoardState {
	t.Helper()
	a := g.partyRunning("poolboard")
	if a == nil {
		t.Fatal("not running")
	}
	s := a.D.(*poolBoardState)
	scr := ebiten.NewImage(screenW, screenH)
	if len(s.js) < 1 || len(s.js) > 2 {
		t.Fatalf("%d jumpers", len(s.js))
	}
	for f := 0; f < frames && g.partyRunning("poolboard") != nil; f++ {
		peStep(g, 1.0/30, true)
		for _, j := range s.js {
			gu := j.mem.gu
			if !gu.out() && (g.lvl.SolidP(gu.pos.X, gu.pos.Y) || !g.lvl.Free(gu.pos)) {
				t.Fatalf("jumper inside something at %v (phase %d)", gu.pos, j.phase)
			}
			if f%5 == 0 {
				p := gu.pos
				g.pevGuestDraw(scr, gu, &p)
			}
		}
		if f%10 == 0 {
			peDraw(g)
		}
		if each != nil {
			each(s)
		}
	}
	return s
}

func TestPoolBoardJumpSplashAndSwim(t *testing.T) {
	peRetry(t, "poolboard", false, func(g *Game) string {
		touched := map[*Guest]bool{}
		s := pbRun(t, g, 30*80, func(s *poolBoardState) { peTrack(g, touched) })
		if s.splashes == 0 {
			return "nobody jumped"
		}
		for _, j := range s.js {
			gu := j.mem.gu
			if j.phase != pbDone {
				return "a jumper never finished"
			}
			if gu.out() {
				continue
			}
			if !g.lvl.InPool(gu.pos) && gu.state == gSwim {
				return "he swims out of the water"
			}
		}
		if g.partyRunning("poolboard") != nil {
			return "the event never ended"
		}
		if gu := peStale(g, touched); gu != nil {
			return "a guest is still held"
		}
		return ""
	})
}

func TestPoolBoardStabbedOnThePlank(t *testing.T) {
	peRetry(t, "poolboard", false, func(g *Game) string {
		var victim *Guest
		dead := false
		pbRun(t, g, 30*80, func(s *poolBoardState) {
			if dead {
				return
			}
			for _, j := range s.js {
				if j.phase == pbPlank && j.t > 0.5 {
					victim = j.mem.gu
					g.killGuest(victim, true)
					dead = true
				}
			}
		})
		if !dead {
			return "nobody walked the plank"
		}
		if !victim.dead {
			return "not dead"
		}
		var body *Body
		for _, b := range g.bodies {
			if dist(b.pos, victim.pos) < 2 {
				body = b
			}
		}
		if body == nil {
			return "no body"
		}
		if !g.lvl.InPool(body.pos) {
			return "the body is not in the water"
		}
		if g.partyRunning("poolboard") != nil {
			return "the event did not end"
		}
		return ""
	})
}

func TestPoolBoardDoesNotMoveGameRngAndTwoPlayers(t *testing.T) {
	g := peStart(t, "poolboard", false)
	g.rng = rand.New(rand.NewSource(99))
	for s := 0; s < 30*40; s++ {
		peStep(g, 1.0/30, false)
		if s%30 == 0 {
			peDraw(g)
		}
	}
	if got, want := g.rng.Int63(), rand.New(rand.NewSource(99)).Int63(); got != want {
		t.Error("poolboard moved g.rng")
	}
	g2 := peStart(t, "poolboard", true)
	if !g2.is2P() {
		t.Fatal("not two players")
	}
	pbRun(t, g2, 30*80, nil)
}
