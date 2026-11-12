package main

import "testing"

// Bug (2026-11-09, Isla Nublada): "the paramedics come by boat, get off, VANISH at once and another
// ambulance comes". The T-Rex ate the officers, the call ended (police_groups watchdog) and the
// bodies went back to "not found": the paramedics arriving found nothing to pick up, got back on
// board at once and the next witness called again. Now the bodies stay found while an ambulance
// is called or working, the crew picks them up, and only one ambulance comes.
func TestAmbulanceCrewStaysAfterThePoliceAreGone(t *testing.T) {
	for _, lv := range []int{specialJurassic, 0} {
		g := playGame(t, lv, 2)
		if g.rex != nil {
			g.rex.state, g.rex.timer = jrSleep, 1e9 // no T-Rex in the way of the measurement
		}
		n := 0
		for _, gu := range g.guests {
			if !gu.out() && gu.perf == "" && n < 3 {
				g.killGuest(gu, true)
				n++
			}
		}
		orig := append([]*Body(nil), g.bodies...)
		for _, b := range g.bodies {
			b.reported = true
		}
		g.p.pos = g.lvl.Def.Start
		g.raisePolice(g.bodies[0].pos, false)
		// the officers must be out first: on the island the medical helicopter (2026-11-11) can land
		// before the police boat has tied up, and the point here is the police being gone
		if !runPlay(g, 40, func() bool {
			for _, c := range g.cops {
				if !c.medic {
					return g.ambCalled
				}
			}
			return false
		}) {
			t.Fatalf("level %d: no police and ambulance call", lv)
		}
		// every officer is killed before the ambulance gets there: the call ends
		for _, c := range append([]*Cop(nil), g.cops...) {
			if !c.medic {
				g.removeCop(c)
			}
		}
		var crew []*Cop
		born := 0.0
		if !runPlay(g, 40, func() bool {
			for _, c := range g.cops {
				if c.medic {
					crew = append(crew, c)
				}
			}
			born = g.time
			return len(crew) > 0
		}) {
			t.Fatalf("level %d: the ambulance never arrived", lv)
		}
		if g.policeCalled {
			t.Fatalf("level %d: setup: the police call should be over", lv)
		}
		ambs := map[*Vehicle]bool{}
		left, many := -1.0, false
		runPlay(g, 60, func() bool {
			rep := 0
			for _, b := range g.bodies {
				if b.reported && !b.carried {
					rep++
				}
			}
			many = many || rep >= ambTwinBodies // the >=5 bodies rule may send a second one
			for _, v := range g.veh {
				if v.kind == 1 || v.kind == 3 {
					ambs[v] = true
				}
			}
			alive := false
			for _, c := range crew {
				if containsCop(g.cops, c) {
					alive = true
				}
			}
			if !alive {
				left = g.time
				return true
			}
			return false
		})
		if left >= 0 && left-born < 8 {
			t.Fatalf("level %d: the paramedics were gone %.1fs after getting off", lv, left-born)
		}
		if len(ambs) > 1 && !many {
			t.Fatalf("level %d: %d ambulances while the first one was working", lv, len(ambs))
		}
		rem := 0
		for _, b := range orig {
			for _, x := range g.bodies {
				if x == b && !b.carried && !b.bagged {
					rem++
				}
			}
		}
		if rem == len(orig) && left >= 0 {
			st := ""
			for _, b := range g.bodies {
				st += fmtB(b)
			}
			t.Fatalf("level %d: the paramedics left (after %.1fs) without picking up any of the %d bodies: %s", lv, left-born, rem, st)
		}
	}
}

func fmtB(b *Body) string {
	r := "["
	if b.reported {
		r += "rep "
	}
	if b.claimed != nil {
		r += "claimed "
	}
	if b.fallT > 0 {
		r += "falling "
	}
	return r + "]"
}
