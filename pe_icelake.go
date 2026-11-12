package main

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE FIGURE SKATER (idea 290, approved by the user): in the snow and ice parties, now and then ONE guest skates on
// the ice doing tricks: glides, pirouettes, jumps, the swan, a funny fall and getting up, and a bow at the end.
//
// THE ICE: no level has a frozen lake: the ice of the level is used (the ski resort's rink, a "white" floor inside
// its boards, and the hockey rink, the "rink" floor). The skating reuses the rink's own system (ice.go: iceStep for
// the inertia, iceMove for the walls, iceTrace for the marks, iceFaceVel) and the skaters' poses (pe_skaters.go):
// nothing of it is duplicated. There is no ice that breaks anywhere, so nobody falls into the water (the user's
// extra was only "if it already exists").
//
// A reactive, Minor event "hielo" (never a place of the frame) in the group "skate" (never together with the
// skaters of the hockey rink) with its OWN dice per party: iceSkChance of the snow/hockey parties, at a moment of
// iceSkAtMin..Max, at most twice with a long pause. 20-40 s of skating. He is a plain guest (pevMem): if he is killed
// the body lies on the ice like any other (the rink's own rules), an alarm near ends it, every phase has a time
// limit. Dice: iceSkDice() (never g.rng). -pevent hielo forces it in the level given (progress frozen).
// ---------------------------------------------------------------------------

const (
	iceSkChance  = 0.30
	iceSkAtMin   = 35.0
	iceSkAtMax   = 200.0
	iceSkPause   = 150.0 // between the two shows of a party
	iceSkMinT    = 20.0
	iceSkMaxT    = 40.0
	iceSkWalkMax = 16.0
	iceSkSpeed   = 46.0
	iceSkJumpT   = 0.7
	iceSkJumpH   = 11.0
)

var iceSkIcons = map[string]bool{"snow": true, "hockey": true}

var (
	iceSkGo    = pevTexts("¡Abrid paso a la estrella!", "Mi momento sobre hielo", "¡Hoy toca exhibición!")
	iceSkJump  = pevTexts("¡Salto doble!", "¡Hop!", "¡Mirad qué altura!")
	iceSkSwan  = pevTexts("¡El cisne!", "Elegancia pura...")
	iceSkBow   = pevTexts("¡Gracias, gracias!", "¡Quiero un diez!")
	iceSkWatch = pevTexts("¡Olé!", "¡Precioso!", "¡Que se cae!")
)

var (
	iceSkRng     *rand.Rand
	iceSkSeed    int64
	iceSkInTests bool // tests: roll the real dice under go test (off by default)
	iceSkForce   bool // tests: the show comes for sure where it fits
)

func iceSkDice() *rand.Rand {
	if iceSkRng == nil {
		sd := iceSkSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x1ce5
		}
		iceSkRng = rand.New(rand.NewSource(sd))
	}
	return iceSkRng
}

type iceSkLevel struct {
	lvl   *Level
	want  bool
	at    float64
	runs  int
	tries int
}

var iceSkLv iceSkLevel

// iceSkZone: the ice of the level a figure skater may use (away from the boards and the goals); W = 0 none.
func iceSkZone(l *Level) Rect {
	if l == nil || l.Def == nil {
		return Rect{}
	}
	for _, f := range l.Def.Floors {
		if iceKind(f.Kind) && f.R.W > 160 && f.R.H > 90 {
			return Rect{f.R.X + 50, f.R.Y + 26, f.R.W - 100, f.R.H - 52}
		}
	}
	if l.Def.Icon == "snow" { // the ski resort's rink: the "white" floor inside its boards
		for _, f := range l.Def.Floors {
			if f.Kind == "white" && f.R.W > 160 && f.R.H > 90 {
				return Rect{f.R.X + 50, f.R.Y + 22, f.R.W - 100, f.R.H - 44}
			}
		}
	}
	return Rect{}
}

func iceSkFits(g *Game) bool {
	d := g.lvl.Def
	return d != nil && d.Title != "" && iceSkIcons[d.Icon] && iceSkZone(g.lvl).W > 0 && !g.stealth && g.dc == nil
}

func iceSkReset(g *Game) {
	r := iceSkDice()
	roll, at := r.Float64(), iceSkAtMin+r.Float64()*(iceSkAtMax-iceSkAtMin)
	iceSkLv = iceSkLevel{lvl: g.lvl, at: g.time + at}
	switch {
	case !iceSkFits(g):
	case pevDebugOn("hielo"):
		iceSkLv.want, iceSkLv.at = true, g.time+2
	case iceSkForce:
		iceSkLv.want = true
	case testing.Testing() && !iceSkInTests:
	default:
		iceSkLv.want = roll < iceSkChance
	}
}

func (g *Game) iceSkOK(p Vec) bool {
	z := iceSkZone(g.lvl)
	return z.W > 0 && z.Has(p.X, p.Y) && g.lvl.Free(p) && !g.lvl.solidNPC(p.X, p.Y)
}

func iceSkDue(g *Game, t partyTraits) bool {
	if g.lvl == nil {
		return false
	}
	if iceSkLv.lvl != g.lvl {
		iceSkReset(g)
	}
	if !iceSkLv.want || iceSkLv.runs >= 2 || g.time < iceSkLv.at || g.partyRunning("skaters") != nil {
		return false
	}
	z := iceSkZone(g.lvl)
	return !g.pevAlarmNear(Vec{z.X + z.W/2, z.Y + z.H/2}, 200)
}

const (
	isWalk = iota // to the ice
	isGlide
	isSpin
	isJump
	isSwan
	isFall
	isLie
	isUp
	isBow
)

type iceSkState struct {
	m      pevMem
	phase  int
	t      float64
	skate  float64 // seconds on the ice
	dur    float64
	to     Vec
	entry  Vec
	fell   bool
	jumps  int
	spins  int
	swans  int
	falls  int
	onIce  float64 // tests: the longest stretch he stayed inside the zone
	outIce bool    // tests: he was ever outside the zone while skating
	why    string  // tests: why it ended ("" = the bow)
}

func (s *iceSkState) pevHas(gu *Guest) bool { return s.m.gu == gu }

func init() {
	registerPartyEvent(&partyEvent{ID: "hielo", Name: "Patinador sobre hielo", Group: "skate", Reactive: true, Minor: true,
		Prob: iceSkChance, AtMin: iceSkAtMin, AtMax: iceSkAtMax, Dur: 32,
		Eligible: iceSkDue, Start: iceSkStart,
		Update: pevGuard(iceSkUpdate, func(g *Game, a *partyAct) { iceSkEnd(g, a.D.(*iceSkState)) })})
}

// iceSkPick: the next point to glide to (inside the zone, 60-160 px away).
func (g *Game) iceSkPick(r *rand.Rand, from Vec) Vec {
	z := iceSkZone(g.lvl)
	for i := 0; i < 24; i++ {
		ang := r.Float64() * 2 * math.Pi
		d := 60 + r.Float64()*100
		p := Vec{from.X + math.Cos(ang)*d, from.Y + math.Sin(ang)*d*0.6}
		if g.iceSkOK(p) {
			return p
		}
	}
	return g.lvl.NearestFree(Vec{z.X + z.W/2, z.Y + z.H/2})
}

func iceSkStart(g *Game, a *partyAct) bool {
	r := iceSkDice()
	iceSkLv.at = g.time + 6
	if iceSkLv.tries++; iceSkLv.tries > 15 {
		iceSkLv.want = false
		return false
	}
	z := iceSkZone(g.lvl)
	c := Vec{z.X + z.W/2, z.Y + z.H/2}
	// a guest who already wears skates and is on the ice first; else the nearest free one who can walk onto it
	var best *Guest
	for _, gu := range g.guests {
		if g.pevFreeGuest(gu) && skIsSkater(gu) && g.iceSkOK(gu.pos) {
			best = gu
			break
		}
	}
	if best == nil {
		for _, gu := range g.guests {
			if g.pevFreeGuest(gu) && (best == nil || dist(gu.pos, c) < dist(best.pos, c)) {
				best = gu
			}
		}
	}
	if best == nil || dist(best.pos, c) > 420 && !pevDebugOn("hielo") {
		return false
	}
	entry := Vec{math.Max(z.X+8, math.Min(best.pos.X, z.X+z.W-8)), math.Max(z.Y+8, math.Min(best.pos.Y, z.Y+z.H-8))}
	entry = g.lvl.NearestFree(entry)
	if !g.iceSkOK(entry) {
		entry = g.lvl.NearestFree(c)
		if !g.iceSkOK(entry) {
			return false
		}
	}
	s := &iceSkState{entry: entry, dur: iceSkMinT + r.Float64()*(iceSkMaxT-iceSkMinT)}
	s.m.gu = best
	if g.iceSkOK(best.pos) {
		s.m.pose(AnimLean)
		best.vel = Vec{}
		s.phase, s.to = isGlide, g.iceSkPick(r, best.pos)
	} else if !g.pevSendTo(&s.m, entry) {
		return false
	}
	if best.icon == "" {
		best.icon, best.iconT = pevPick(r, iceSkGo), 2
	}
	iceSkLv.runs++
	iceSkLv.at = g.time + iceSkPause*(0.8+0.4*r.Float64())
	a.D = s
	if pevDebugOn("hielo") {
		fmt.Printf("PEVDEMO hielo zone %.0f %.0f %.0f %.0f\n", z.X, z.Y, z.W, z.H)
	}
	return true
}

func iceSkEnd(g *Game, s *iceSkState) {
	if s == nil {
		return
	}
	gu := s.m.gu
	if s.m.mine() {
		pevRelease(&s.m, iceSkDice())
		if gu != nil && !g.lvl.OnIce(gu.pos) {
			gu.vel = Vec{}
		}
	} else {
		pevForget(&s.m)
	}
}

// iceSkTrick: what he does when he reaches a point.
func (g *Game) iceSkTrick(s *iceSkState, r *rand.Rand) {
	gu := s.m.gu
	next := func(ph int) { s.phase, s.t = ph, 0 }
	x := r.Float64()
	if !s.fell && s.skate > s.dur*0.6 {
		x = 0.9 // the funny fall comes once in every show
	}
	switch {
	case x < 0.28:
		next(isSpin)
		gu.dAnim = AnimPirouette
		s.spins++
		if gu.icon == "" && r.Float64() < 0.5 {
			gu.icon, gu.iconT = pevPick(r, skBrag), 1.6
		}
	case x < 0.5:
		next(isJump)
		gu.dAnim = AnimCheer
		s.jumps++
		if gu.icon == "" {
			gu.icon, gu.iconT = pevPick(r, iceSkJump), 1.4
		}
	case x < 0.68:
		next(isSwan)
		gu.dAnim = AnimLean
		s.swans++
		s.to = g.iceSkPick(r, gu.pos)
		if gu.icon == "" {
			gu.icon, gu.iconT = pevPick(r, iceSkSwan), 1.8
		}
	case x >= 0.88 && !s.fell:
		next(isFall)
		gu.dAnim = AnimFall
		s.fell = true
		s.falls++
		gu.icon, gu.iconT = pevPick(r, skFall), 2
	default:
		s.to = g.iceSkPick(r, gu.pos)
	}
}

func iceSkUpdate(g *Game, a *partyAct, dt float64) bool {
	s := a.D.(*iceSkState)
	r := iceSkDice()
	if !s.m.mine() {
		s.why = "lost"
		iceSkEnd(g, s)
		return true
	}
	gu := s.m.gu
	z := iceSkZone(g.lvl)
	if g.pevAlarmNear(gu.pos, 130) {
		s.why = "alarm"
		iceSkEnd(g, s)
		return true
	}
	s.t += dt
	next := func(ph int) { s.phase, s.t = ph, 0 }
	if s.phase == isWalk {
		if gu.state == gIdle && gu.timer > 1000 && dist(gu.pos, s.entry) < 6 {
			s.m.pose(AnimLean)
			gu.vel = Vec{}
			s.to = g.iceSkPick(r, gu.pos)
			next(isGlide)
		} else if s.t > iceSkWalkMax {
			s.why = "walk"
			iceSkEnd(g, s)
			return true
		}
		return false
	}
	s.m.keep()
	s.skate += dt
	if g.iceSkOK(gu.pos) {
		s.onIce += dt
	} else {
		s.outIce = true
	}
	wish := Vec{}
	push := func(sp float64) {
		if d := dist(gu.pos, s.to); d > 10 {
			wish = Vec{(s.to.X - gu.pos.X) / d * sp, (s.to.Y - gu.pos.Y) / d * sp}
		}
	}
	switch s.phase {
	case isGlide:
		gu.dAnim = AnimLean
		push(iceSkSpeed)
		if dist(gu.pos, s.to) <= 12 {
			if s.skate >= s.dur {
				next(isBow)
				gu.dAnim = AnimWave
				gu.vel = Vec{}
				gu.icon, gu.iconT = pevPick(r, iceSkBow), 2
				g.iceSkCheer(s, r)
			} else {
				g.iceSkTrick(s, r)
			}
		} else if s.t > 9 { // never stuck going to one point
			s.to, s.t = g.iceSkPick(r, gu.pos), 0
		}
	case isSpin:
		gu.dAnim = AnimPirouette
		if s.t >= 1.4 {
			s.to = g.iceSkPick(r, gu.pos)
			next(isGlide)
		}
	case isJump:
		gu.dAnim = AnimCheer
		if s.t >= iceSkJumpT {
			s.to = g.iceSkPick(r, gu.pos)
			next(isGlide)
			if r.Float64() < 0.5 {
				g.iceSkCheer(s, r)
			}
		}
	case isSwan:
		gu.dAnim = AnimLean
		push(iceSkSpeed * 0.8)
		if s.t >= 2.2 || dist(gu.pos, s.to) <= 12 {
			s.to = g.iceSkPick(r, gu.pos)
			next(isGlide)
		}
	case isFall:
		gu.dAnim = AnimFall
		if s.t >= 0.45 {
			next(isLie)
			if !g.pevPlayerFar(gu.pos, 220) {
				g.snd.Play("thud")
			}
		}
	case isLie:
		gu.dAnim = AnimLying
		if s.t >= 1.8 {
			gu.dAnim = AnimCrouch
			next(isUp)
		}
	case isUp:
		gu.dAnim = AnimCrouch
		if s.t >= 0.8 {
			s.to = g.iceSkPick(r, gu.pos)
			next(isGlide)
		}
	case isBow:
		gu.dAnim = AnimWave
		gu.vel = Vec{}
		if s.t >= 1.6 {
			iceSkEnd(g, s)
			return true
		}
	}
	if s.phase != isBow {
		iceStep(&gu.vel, wish, dt)
		ahead := Vec{gu.pos.X + gu.vel.X*dt*1.2, gu.pos.Y + gu.vel.Y*dt*1.2}
		if !g.iceSkOK(ahead) { // about to leave the ice: he turns back
			gu.vel = Vec{-gu.vel.X * 0.4, -gu.vel.Y * 0.4}
			if s.phase == isGlide || s.phase == isSwan {
				s.to = g.iceSkPick(r, gu.pos)
			}
		}
		iceMove(g.lvl, &gu.pos, &gu.vel, dt, true)
		if !g.iceSkOK(gu.pos) { // (a push from outside): back inside
			gu.pos = g.lvl.NearestFree(Vec{math.Max(z.X+4, math.Min(gu.pos.X, z.X+z.W-4)), math.Max(z.Y+4, math.Min(gu.pos.Y, z.Y+z.H-4))})
			gu.vel = Vec{}
		}
		if s.phase == isGlide || s.phase == isSwan || s.phase == isSpin {
			iceFaceVel(&gu.dir, gu.vel)
		}
		g.iceTrace(gu.pos, gu.vel)
	}
	gu.moving = false
	return false
}

// iceSkCheer: somebody near the ice claps or shouts.
func (g *Game) iceSkCheer(s *iceSkState, r *rand.Rand) {
	gu := s.m.gu
	for _, o := range g.guests {
		if o != gu && !o.out() && o.icon == "" && dist(o.pos, gu.pos) < 140 && lifeFree(o) {
			o.icon, o.iconT = pevPick(r, iceSkWatch), 1.5
			return
		}
	}
}

// drawGuest: the skater in the air during a jump and lying (dizzy) after the fall (the frame's hook, pe_fx.go).
func (s *iceSkState) drawGuest(g *Game, dst *ebiten.Image, gu *Guest, pos Vec) bool {
	if s.m.gu != gu || !s.m.mine() {
		return false
	}
	switch s.phase {
	case isJump:
		sc := g.lifeScale()
		if sc <= 0 {
			sc = 1
		}
		u := math.Min(1, s.t/iceSkJumpT)
		lift := iceSkJumpH * 4 * u * (1 - u)
		x, y := math.Round(pos.X), math.Round(pos.Y)
		g.drawActor(dst, gu.cs, AnimPirouette, s.t*6, Vec{x, y}, gu.dir, -math.Round(lift)/sc, false)
		return true
	case isLie:
		g.drawLying(dst, gu.cs, false, Vec{math.Round(pos.X), math.Round(pos.Y)})
		cx, cy := g.pevXY(gu.pos)
		drawStarsPix(g.plotter(dst), cx, cy-g.pevK()(12), g.pevPx(), g.time)
		return true
	}
	return false
}
