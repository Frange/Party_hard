package main

import "math"

// ---------------------------------------------------------------------------
// EVENT "skaters": on the HOCKEY rink a few of the guests who already wear white skates stop wandering and SKATE:
// they glide from point to point with the rink's own inertia (iceStep / iceMove, ice.go), now and then spin a
// pirouette or take a funny fall (lying 2 s, "ouch", then back up). They never leave the ice (a box a bit inside the
// boards and away from the nets) and they never block anything: when the event ends they are let go where they
// stand, still on the ice, and the normal AI (followIce) takes over from there, so reachability is untouched.
//
// DECISION: an event of the frame (not a continuous ambience): high chance (40 %, the most the frame allows; 60 % was asked) but with a window and a pause, so
// the rink is not a permanent ballet. Between 25 and 150 s, up to twice (cooldown 100 s). Only the hockey rink.
// ---------------------------------------------------------------------------

const (
	skMin, skMax = 3, 6
	skSpeed      = 40.0
)

var skFall = pevTexts("¡Ay, mi cola!", "¡Whoaaa!", "¡PUM! Estoy bien...", "¡El hielo me odia!", "¡Cuidado!")
var skBrag = pevTexts("¡Mirad esto!", "¡Triple axel!", "¡Soy un cisne!", "¡Y sin manos!", "¡Puntuación perfecta!")

const (
	skGlide = iota
	skSpin
	skDown
	skUp
)

type skater struct {
	m    pevMem
	mode int
	t    float64
	to   Vec
}

type skateState struct {
	sk []*skater
}

func (s *skateState) pevHas(gu *Guest) bool {
	for _, k := range s.sk {
		if k.m.gu == gu {
			return true
		}
	}
	return false
}

func init() {
	registerPartyEvent(&partyEvent{ID: "skaters", Name: "Skaters", Group: "skate", Prob: 0.40, AtMin: 25, AtMax: 150, Dur: 55,
		Cooldown: 100, MaxRuns: 2,
		Eligible: func(g *Game, t partyTraits) bool { return t.icon == "hockey" },
		Start:    skStart, Update: pevGuard(skUpdate, skEnd)})
}

// skBox: where a skater may be: the ice, away from the boards and the nets.
func skBox(g *Game) Rect {
	for _, f := range g.lvl.Def.Floors {
		if f.Kind == "rink" {
			return Rect{f.R.X + 50, f.R.Y + 26, f.R.W - 100, f.R.H - 50}
		}
	}
	return Rect{}
}

func skOK(g *Game, p Vec) bool {
	b := skBox(g)
	return b.W > 0 && b.Has(p.X, p.Y) && g.lvl.OnIce(p) && !g.lvl.SolidP(p.X, p.Y)
}

func skIsSkater(gu *Guest) bool {
	return gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Shoes == hx(0xe8eef4)
}

func skStart(g *Game, a *partyAct) bool {
	var cand []*Guest
	for _, gu := range g.guests {
		if skIsSkater(gu) && g.pevFreeGuest(gu) && skOK(g, gu.pos) {
			cand = append(cand, gu)
		}
	}
	if len(cand) < skMin {
		return false
	}
	r := partyDice()
	r.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	n := skMin + r.Intn(skMax-skMin+1)
	if n > len(cand) {
		n = len(cand)
	}
	st := &skateState{}
	for _, gu := range cand[:n] {
		k := &skater{mode: skGlide}
		k.m = pevMem{gu: gu}
		k.m.pose(AnimLean)
		gu.vel = Vec{}
		k.to = skPick(g, r, gu.pos)
		st.sk = append(st.sk, k)
	}
	a.D = st
	return true
}

// skPick: the next point to skate to (inside the box, 70-190 px away).
func skPick(g *Game, r interface{ Float64() float64 }, from Vec) Vec {
	b := skBox(g)
	for i := 0; i < 20; i++ {
		ang := r.Float64() * 2 * math.Pi
		d := 70 + r.Float64()*120
		p := Vec{from.X + math.Cos(ang)*d, from.Y + math.Sin(ang)*d*0.7}
		if skOK(g, p) {
			return p
		}
	}
	return Vec{b.X + b.W/2, b.Y + b.H/2}
}

func skUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*skateState)
	r := partyDice()
	keep := st.sk[:0]
	for _, k := range st.sk {
		if k.m.mine() {
			keep = append(keep, k)
		}
	}
	st.sk = keep
	if len(st.sk) < 2 || g.time-a.t0 > a.ev.Dur {
		skEnd(g, a)
		return true
	}
	l := g.lvl
	for _, k := range st.sk {
		gu := k.m.gu
		k.m.keep()
		k.t -= dt
		wish := Vec{}
		switch k.mode {
		case skGlide:
			d := dist(gu.pos, k.to)
			if d > 12 {
				wish = Vec{(k.to.X - gu.pos.X) / d * skSpeed, (k.to.Y - gu.pos.Y) / d * skSpeed}
				gu.dAnim = AnimLean
			} else {
				switch x := r.Float64(); {
				case x < 0.30:
					k.mode, k.t = skSpin, 1.3
					gu.dAnim = AnimPirouette
					if gu.icon == "" && r.Float64() < 0.5 {
						gu.icon, gu.iconT = pevPick(r, skBrag), 1.8
					}
				case x < 0.42:
					k.mode, k.t = skDown, 2.2
					gu.dAnim = AnimFall
					gu.icon, gu.iconT = pevPick(r, skFall), 2
				default:
					k.to = skPick(g, r, gu.pos)
				}
			}
		case skSpin:
			gu.dAnim = AnimPirouette // keeps gliding on its speed, no new push
			if k.t <= 0 {
				k.mode, k.to = skGlide, skPick(g, r, gu.pos)
			}
		case skDown:
			if k.t < 1.75 {
				gu.dAnim = AnimLying
			}
			if k.t <= 0 {
				k.mode, k.t = skUp, 0.7
				gu.dAnim = AnimCrouch
			}
		case skUp:
			gu.dAnim = AnimCrouch
			if k.t <= 0 {
				k.mode, k.to = skGlide, skPick(g, r, gu.pos)
			}
		}
		// inertia of the rink; never out of the box: a skater who is about to leave it turns back
		iceStep(&gu.vel, wish, dt)
		next := Vec{gu.pos.X + gu.vel.X*dt*1.2, gu.pos.Y + gu.vel.Y*dt*1.2}
		if !skOK(g, next) {
			gu.vel = Vec{-gu.vel.X * 0.4, -gu.vel.Y * 0.4}
			k.to = skPick(g, r, gu.pos)
		}
		iceMove(l, &gu.pos, &gu.vel, dt, true)
		if !skOK(g, gu.pos) { // (a push from outside): back inside
			gu.pos = l.NearestFree(Vec{math.Max(skBox(g).X, math.Min(gu.pos.X, skBox(g).X+skBox(g).W)), math.Max(skBox(g).Y, math.Min(gu.pos.Y, skBox(g).Y+skBox(g).H))})
			gu.vel = Vec{}
		}
		if k.mode == skGlide || k.mode == skSpin {
			iceFaceVel(&gu.dir, gu.vel)
		}
		g.iceTrace(gu.pos, gu.vel)
		gu.moving = false
	}
	return false
}

func skEnd(g *Game, a *partyAct) {
	st := a.D.(*skateState)
	r := partyDice()
	for _, k := range st.sk {
		gu := k.m.gu
		if gu != nil && !gu.out() && k.m.mine() {
			pevRelease(&k.m, r)
			gu.vel = Vec{}
		}
	}
}
