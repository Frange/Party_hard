package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// EVENT "juggler": a juggler throws three coloured balls in a continuous cascade (arcs computed from the time, no
// sprites). Sometimes a ball falls: he stops, blushes, and bows his head in embarrassment; the guests applaud
// anyway. The ball rolls on the floor. If a player presses E near it before he goes to get it, he picks it up and hands
// it back: a small bonus (once, 30 points) and the juggler thanks him. If not, the juggler picks it up himself, with
// shame, and goes on. Coins from the audience, applause at the end.
//
// WHEN: open-air parties of the street kind (Paris, fair, festival, beach) and birthday parties, 20 % of them, between 50
// and 200 s, up to twice. Group "street".
// ---------------------------------------------------------------------------

var jugglerIcons = map[string]bool{"eiffel": true, "funfair": true, "festival": true, "beach": true, "cake": true}

const (
	jgActLen  = 24.0
	jgBonus   = 30
	jgReach   = 30.0
	jgWaitPic = 4.2 // the ball waits this long for the player before he picks it up himself
)

var jugglerLines = pevTexts("¡Uy!", "Esto no me pasa en casa...", "¡Gracias, eres un crack!", "¡Ahora sí!", "¡Bravo!", "¡Otra, otra!", "¡Qué arte!",
	"¡Eso, ánimo!", "Qué vergüenza...", "Un momento, que se me ha escapado", "Te debo una pelota", "¡Ta-chán!")

const (
	jgJuggle = iota
	jgDropped
	jgPicking
)

// jgGiven: the level (party) in which the player already got the bonus: once per party, even if the juggler comes back.
var jgGiven *Level

type jugglerState struct {
	show    pevShow
	jphase  int
	jt0     float64
	drops   []float64 // seconds into the act when a ball falls
	didDrop int
	ball    Vec // where the fallen ball is (world)
	ballT   float64
	given   bool // the player already got his bonus
	lastCl  int
}

func (j *jugglerState) pevHas(gu *Guest) bool { return j.show.pevHas(gu) }

func init() {
	registerPartyEvent(&partyEvent{ID: "juggler", Name: "Malabarista", Group: "street",
		Prob: 0.20, AtMin: 50, AtMax: 200, Dur: 40, Cooldown: 150, MaxRuns: 2,
		Eligible: func(g *Game, t partyTraits) bool {
			return t.outdoor && jugglerIcons[t.icon] && t.roam && t.guests >= 16
		},
		Start: jugglerStart, Items: jugglerItems,
		Update: pevGuard(jugglerUpdate, func(g *Game, a *partyAct) { a.D.(*jugglerState).show.end(g) })})
}

func jugglerSpec() CharSpec {
	return CharSpec{
		Skin: hx(0xe8b890), Hair: hx(0xe04a3a), Shirt: hx(0xf0c030), Shirt2: hx(0x3a6ad0), Pants: hx(0x2a4a9a), Shoes: hx(0xd03030),
		Accent: hx(0xf0c030), HatCol: hx(0x3a6ad0), Hat: HatNone, Hair2: HairCurly, Top: TopTee, Pattern: 3, Susp: hx(0xd03030), Makeup: true, Face: 1, noOverride: true,
	}
}

func jugglerStart(g *Game, a *partyAct) bool {
	s, ok := g.pevShowStart(jugglerSpec(), jgActLen, true)
	if !ok {
		return false
	}
	r := partyDice()
	st := &jugglerState{show: *s, jphase: jgJuggle}
	st.drops = []float64{jgActLen * (0.34 + 0.12*r.Float64())}
	if r.Float64() < 0.4 {
		st.drops = append(st.drops, jgActLen*0.78)
	}
	a.D = st
	return true
}

// the balls: three colours
var jgCols = []color.RGBA{{0xe8, 0x30, 0x40, 255}, {0x30, 0x90, 0xf0, 255}, {0xf8, 0xd8, 0x30, 255}}

// jgHands: the two hands (sprite pixels from the feet).
func jgHands() (lx, rx, y float64) { return -10, 10, -31 }

// jugglerBall: position (sprite pixels from the feet, x then y up negative) of ball i at time t in the cascade.
func jugglerBall(i int, t float64) (float64, float64) {
	lx, rx, hy := jgHands()
	u := math.Mod(t/1.5+float64(i)/3, 1)
	w := math.Mod(u*2, 1)
	x0, x1 := lx, rx
	if u >= 0.5 {
		x0, x1 = rx, lx
	}
	return x0 + (x1-x0)*w, hy - 24*4*w*(1-w)
}

func jugglerUpdate(g *Game, a *partyAct, dt float64) bool {
	st := a.D.(*jugglerState)
	r := partyDice()
	prev := st.show.phase
	done := g.pevShowStep(&st.show, dt, 3+r.Intn(2), 4.2, nil)
	s := &st.show
	if done {
		s.end(g)
		return true
	}
	if prev == showEnter && s.phase == showAct {
		st.jt0 = g.time
	}
	if s.phase != showAct {
		return false
	}
	age := g.time - s.t0
	switch st.jphase {
	case jgJuggle:
		if st.didDrop < len(st.drops) && age >= st.drops[st.didDrop] {
			st.didDrop++
			st.jphase, st.jt0 = jgDropped, g.time
			st.ballT = g.time
			side := 1.0
			if r.Float64() < 0.5 {
				side = -1
			}
			st.ball = Vec{s.pf.pos.X + side*(14+r.Float64()*6), s.pf.pos.Y + 4 + r.Float64()*3}
			if !g.lvl.Free(st.ball) || g.lvl.SolidP(st.ball.X, st.ball.Y) {
				st.ball = Vec{s.pf.pos.X, s.pf.pos.Y + 8}
			}
			s.pf.dir = DirDown
			bubbleOnce(g, s.pf, jugglerLines[0])
			s.cheer(r, jugglerLines[6:8])
		}
	case jgDropped:
		if g.time-st.jt0 > jgWaitPic {
			st.jphase, st.jt0 = jgPicking, g.time
			s.pf.dir = faceTo(s.pf.pos, st.ball)
			s.crowdCheer(r)
		}
	case jgPicking:
		if g.time-st.jt0 > 1.6 {
			st.jphase = jgJuggle
			st.ballT = 0
		}
	}
	return false
}

func bubbleOnce(g *Game, pf *pevPerf, txt string) { g.bubble(pf, bubGuest, pf.pos, txt, false, 0) }

func (s *pevShow) crowdCheer(r *rand.Rand) {
	for _, m := range s.crowd.mem {
		if m.gu.icon == "" && r.Float64() < 0.5 {
			m.gu.icon, m.gu.iconT = jugglerLines[4+r.Intn(3)], 1.8
		}
	}
}

// pevUse: E near the fallen ball: the player picks it up and gives it back (bonus, once).
func (j *jugglerState) pevUse(g *Game, p *Player) bool {
	if j.show.phase != showAct || j.jphase != jgDropped || dist(p.pos, j.ball) > jgReach {
		return false
	}
	r := partyDice()
	j.jphase, j.jt0 = jgPicking, g.time-0.4
	j.show.pf.dir = faceTo(j.show.pf.pos, p.pos)
	if !j.given && jgGiven != g.lvl {
		j.given, jgGiven = true, g.lvl
		g.award(j.ball, jgBonus, "")
	}
	g.bubble(j.show.pf, bubGuest, j.show.pf.pos, jugglerLines[2], false, 0)
	j.show.crowdCheer(r)
	g.snd.Play("spark")
	return true
}

// drawBallPix: a ball of 4x4 with a highlight.
func drawBallPix(plot plotFn, x, y, px int, c color.RGBA) {
	plot(x-px, y-2*px, 3*px, 4*px, c)
	plot(x-2*px, y-px, 5*px, 2*px, c)
	plot(x-px, y-px, px, px, pevWhite)
}

// drawBlushPix: red cheeks on a head whose face centre is at (cx, cy).
func drawBlushPix(plot plotFn, cx, cy, px int) {
	red := color.RGBA{0xf0, 0x50, 0x60, 255}
	plot(cx-4*px, cy, 2*px, px, red)
	plot(cx+3*px, cy, 2*px, px, red)
}

func jugglerItems(g *Game, dst *ebiten.Image, a *partyAct, items []drawItem) []drawItem {
	st := a.D.(*jugglerState)
	s := &st.show
	plot := g.plotter(dst)
	k := g.pevK()
	px := g.pevPx()
	pf := s.pf
	acting := s.phase == showAct
	items = append(items, drawItem{pf.pos.Y, func() {
		anim, ph := AnimIdle, g.time*1.3
		switch {
		case acting && st.jphase == jgJuggle:
			anim, ph = AnimPanic, g.time*2.5 // both arms up: the balls fly between the hands
		case acting && st.jphase == jgPicking:
			anim, ph = AnimCrouch, g.time*2
		case acting:
			anim, ph = AnimTired, g.time*2
		}
		pf.draw(g, dst, anim, ph)
	}})
	if acting {
		items = append(items, drawItem{pf.pos.Y + 0.2, func() {
			cx, cy := g.pevXY(pf.pos)
			t := g.time - s.t0
			lx, rx, hy := jgHands()
			switch st.jphase {
			case jgJuggle:
				for i := 0; i < 3; i++ {
					bx, by := jugglerBall(i, t)
					drawBallPix(plot, cx+k(bx), cy+k(by), px, jgCols[i])
				}
			default:
				// two balls in the hands, the third on the floor (or in the air if he has just let it go)
				drawBallPix(plot, cx+k(lx), cy+k(hy), px, jgCols[0])
				drawBallPix(plot, cx+k(rx), cy+k(hy), px, jgCols[1])
				if st.jphase == jgDropped {
					drawBlushPix(plot, cx, cy-k(34), px)
					fall := math.Min(1, (g.time-st.ballT)/0.6)
					bx, by := g.pevXY(st.ball)
					h := (1 - fall) * float64(k(34))
					if fall > 0.99 { // a little bounce while it rolls
						h = math.Abs(math.Sin((g.time-st.ballT-0.6)*7)) * float64(k(4)) * math.Max(0, 1-(g.time-st.ballT-0.6)/1.2)
					}
					drawBallPix(plot, bx, by-int(h), px, jgCols[2])
				}
			}
		}})
	}
	items = showCoinItems(g, dst, s, items)
	items = showClapItems(g, dst, s, acting && st.jphase == jgJuggle && math.Mod(g.time-s.t0, 8) > 6.6, items)
	return items
}
