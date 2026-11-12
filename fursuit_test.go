package main

import (
	"math/rand"
	"testing"
)

// fursuitBases: plain heads (no hat or big hairdo): the ones the suits are for.
func fursuitBases(t *testing.T) []*CharSet {
	t.Helper()
	var out []*CharSet
	for _, b := range costumeBases(t) {
		if !tallHead(b) {
			out = append(out, b)
		}
	}
	if len(out) < 3 {
		t.Fatalf("only %d plain heads", len(out))
	}
	return out
}

// The person shows: in the suit, the face of the guest (eyes, skin) is left exactly as it was, inside a hood of fur;
// the suit covers the body and the hood the top of the head. Front view, every suit, several guests.
func TestFursuitFaceShowsInTheHood(t *testing.T) {
	for _, base := range fursuitBases(t) {
		raw := baseFrame(base, AnimIdle, vFront, 0)
		for _, mk := range fursuitKinds {
			li := costumeLookIdx(mk.id)
			if li == 0 || !fursuitLook(uint8(li)) {
				t.Fatalf("%s is not a costume look", mk.id)
			}
			cs := dressedCharSet(base, dressKey{uint8(li), 0})
			got := cs.dressFrame(raw, AnimIdle, vFront, 0)
			d, ok := cs.dressGeomFor(raw, AnimIdle, vFront, 0)
			if !ok {
				t.Fatal("no head found")
			}
			same, face := 0, 0
			for y := d.eyeY() - 1; y <= d.hy+d.hh-1; y++ {
				for x := d.l(); x <= d.r(); x++ {
					i := (y*raw.w + x) * 4
					if raw.p[i+3] == 0 {
						continue
					}
					face++
					if raw.p[i] == got.p[i] && raw.p[i+1] == got.p[i+1] && raw.p[i+2] == got.p[i+2] && raw.p[i+3] == got.p[i+3] {
						same++
					}
				}
			}
			if face < 10 || float64(same) < 0.8*float64(face) {
				t.Errorf("%s on %s: only %d of %d face pixels are the guest's own", mk.id, base.Spec.Name, same, face)
			}
			// the hood: fur over the head, and the suit: fur over the body
			fur := hx(mk.fur)
			hood, body := 0, 0
			for y := 0; y < got.h; y++ {
				for x := 0; x < got.w; x++ {
					i := (y*got.w + x) * 4
					if got.p[i+3] == 0 || got.p[i] != fur.R || got.p[i+1] != fur.G || got.p[i+2] != fur.B {
						continue
					}
					if y < d.eyeY()-1 {
						hood++
					} else if y > d.ty {
						body++
					}
				}
			}
			if hood < 6 || body < 20 {
				t.Errorf("%s on %s: hood %d and suit %d fur pixels", mk.id, base.Spec.Name, hood, body)
			}
		}
	}
}

// The suits are never picked by the themed parties, and the guests in them walk slower (and only them).
func TestFursuitLooksApartAndSlower(t *testing.T) {
	for kind, ids := range costumeLevels {
		for _, id := range ids {
			if fursuitLook(uint8(costumeLookIdx(id))) {
				t.Errorf("theme %s uses the suit %s", kind, id)
			}
		}
	}
	sp := genericSpec(rand.New(rand.NewSource(5)))
	sp.Name = ""
	plain := &Guest{cs: newCharSet(&sp)}
	suit := &Guest{cs: dressedCharSet(plain.cs, dressKey{uint8(costumeLookIdx("oso")), 0})}
	themed := &Guest{cs: dressedCharSet(plain.cs, dressKey{uint8(costumeLookIdx("paja")), 0})}
	if fursuitSlow(plain) || fursuitSlow(themed) || !fursuitSlow(suit) {
		t.Fatal("only the suits slow a guest down")
	}
	run := func(gu *Guest) float64 {
		gu.pos, gu.path = Vec{0, 0}, []Vec{{500, 0}}
		gu.walk(guestSpeed, 1)
		return gu.pos.X
	}
	a, b := run(plain), run(suit)
	if b >= a || b < a*0.7 {
		t.Errorf("a suit walks %.1f px where a plain guest walks %.1f", b, a)
	}
}

// fursuitGame: a party with the suits forced in (the event starts a few seconds in).
func fursuitGame(t *testing.T, idx int, two bool, seed int64) *Game {
	t.Helper()
	prev := [...]any{fursuitRng, fursuitSeed, fursuitLvl, fursuitStarted, fursuitWant, fursuitDone}
	t.Cleanup(func() {
		fursuitRng, fursuitSeed, fursuitLvl, fursuitStarted = prev[0].(*rand.Rand), prev[1].(int64), prev[2].(*Level), prev[3].(int)
		fursuitWant, fursuitDone = prev[4].(bool), prev[5].(bool)
	})
	fursuitRng, fursuitSeed = rand.New(rand.NewSource(seed)), seed
	peLoadSeed = seed
	g := peLoad(t, idx, two)
	peLoadSeed = 0
	fursuitLvl = nil          // peLoad keeps the suits out of the other tests' parties: this one rolls them
	dbgPartyEvent = "mascota" // forced: the roll of the party says yes, 3 s in
	g.pev = pevState{ready: true, traits: partyTraitsOf(g.lvl.Def, g.levelIdx)}
	for s := 0; s < 30*10 && g.partyRunning("mascota") == nil; s++ {
		peStep(g, 1.0/30, true)
	}
	return g
}

// In a party with the suits, 1-3 guests wear them for the whole party; they dance now and then by themselves (their own
// step is drawn); nobody is added (caps and goals untouched); killing one gives the player a few extra points and a
// burst of confetti and fur; a death that is not the player's gives nothing. 1 and 2 players.
func TestFursuitPartyDanceAndBonus(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := fursuitGame(t, 0, two, 3)
		a := g.partyRunning("mascota")
		if a == nil {
			t.Fatal("the suits did not come")
		}
		s := a.D.(*fursuitState)
		if n := len(s.ms); n < fursuitMin || n > fursuitMax {
			t.Fatalf("%d guests in a suit", n)
		}
		total, live := g.total, g.guestLive()
		danced := false
		for i := 0; i < 30*60; i++ {
			peStep(g, 1.0/30, true)
			if i%20 == 0 {
				peDraw(g)
			}
			for _, m := range s.ms {
				if m.gu.state == gDance {
					danced = true
				}
			}
		}
		if g.total != total || g.guestLive() != live {
			t.Errorf("the suits changed the party: total %d->%d live %d->%d", total, g.total, live, g.guestLive())
		}
		if !danced {
			t.Error("no mascot danced in a minute")
		}
		if g.partyRunning("mascota") == nil {
			t.Fatal("the suits left before the end of the party")
		}
		// the player kills one: bonus; somebody else kills another: nothing
		m0 := s.ms[0]
		pts := g.points
		g.killGuest(m0.gu, true)
		peStep(g, 1.0/30, false)
		if !m0.paid || len(s.fx) == 0 {
			t.Fatal("no burst for the dead mascot")
		}
		if g.points < pts+fursuitBonus {
			t.Errorf("points %d -> %d: no bonus for the mascot", pts, g.points)
		}
		if len(s.ms) > 1 {
			m1 := s.ms[1]
			pts = g.points
			peleaKill(g, m1.gu)
			peStep(g, 1.0/30, false)
			if g.points != pts {
				t.Errorf("a death that is not the player's gave %d points", g.points-pts)
			}
		}
		for i := 0; i < 60; i++ {
			peStep(g, 1.0/30, false)
			if i%10 == 0 {
				peDraw(g)
			}
		}
		if len(s.fx) != 0 {
			t.Error("the burst never ends")
		}
	}
}

// Only some parties: about one in four of the ones that fit, by its own dice (g.rng and the frame's dice untouched).
func TestFursuitSometimes(t *testing.T) {
	g := partyGame(t)
	prev := [...]any{fursuitRng, fursuitLvl, dbgPartyEvent, fursuitWant, g.lvl, g.time}
	t.Cleanup(func() {
		fursuitRng, fursuitLvl, dbgPartyEvent, fursuitWant = prev[0].(*rand.Rand), prev[1].(*Level), prev[2].(string), prev[3].(bool)
		g.lvl, g.time = prev[4].(*Level), prev[5].(float64)
	})
	dbgPartyEvent = ""
	fursuitRng = rand.New(rand.NewSource(42))
	tr := partyTraitsOf(g.lvl.Def, g.levelIdx)
	if !fursuitFits(tr) {
		t.Fatal("level 0 does not fit the suits")
	}
	yes, n := 0, 4000
	pr := partyRng
	for i := 0; i < n; i++ {
		g.lvl = &Level{Def: g.lvl.Def, W: g.lvl.W, H: g.lvl.H} // a new party
		g.time = 0
		fursuitDue(g, tr)
		if fursuitWant {
			yes++
		}
	}
	if partyRng != pr {
		t.Error("the frame's dice were replaced")
	}
	f := float64(yes) / float64(n)
	if f < 0.20 || f > 0.30 {
		t.Errorf("suits in %.0f %% of the parties, want about 25 %%", f*100)
	}
	if fursuitFits(partyTraits{}) {
		t.Error("the suits fit an unknown level")
	}
}
