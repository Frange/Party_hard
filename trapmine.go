package main

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Traps work like mines. E on a trap prop (speaker, stove, barbecue, fuse box, gas tank...) starts
// a LOAD of trapArmTime seconds (a clock-like circle fills over it); while it loads nothing happens.
// When the circle is full the trap is ARMED and stays so FOR EVER (no expiry, no way to disarm it):
// it goes off the moment somebody comes within trapDetectR of it: anybody: a guest (even one on the phone), a guard, a cop, a
// soldier, a medic, an animal, a special killer or a player.
// NO exceptions when it goes off: any player (the one who armed it, his partner) who comes within the
// radius of an ARMED trap sets it off and suffers the same effect as anybody (real death/damage).
// The only rule is about ARMING, not about affecting: when the load ends and a player (J1 or J2) is
// still inside the radius, the trap waits (armWait) and becomes armed when nobody of the team is
// inside, so arming one never kills you by mistake and nobody has an advantage. In two players each
// one owns his traps (coopOwn; it only matters for the colour and the credit). The effect, the bodies and who gets the credit (g.credit) are the ones
// triggerTrap always had. The traps are level props: nothing here is placed by a generator.
// ---------------------------------------------------------------------------

const (
	trapArmTime   = 2.5  // seconds a trap takes to load before it is armed
	trapDetectR   = 32.0 // detection radius of an armed trap (world pixels)
	trapChargeCol = 0xffa040
)

// trapCentre is the point the detection radius is measured from.
func trapCentre(pr *Prop) Vec {
	return Vec{pr.Foot.X + pr.Foot.W/2, pr.Foot.Y + pr.Foot.H/2}
}

// armTrap starts loading a trap for player p.
func (g *Game) armTrap(pr *Prop, p *Player) {
	pr.armT, pr.armed, pr.by, pr.armWait = trapArmTime, false, p, false
	g.coopOwn(pr, p)
	if pr.K.Trap == "cake" {
		g.cakeArmed(pr) // the candles fizz (trapcake.go)
	}
}

// trapProp: a prop that is a trap (not the poisoned bowl).
func trapProp(pr *Prop) bool { return pr != nil && pr.K != nil && pr.K.Trap != "" && !pr.K.Poison }

// updateTrapMine runs one tick of a trap that is loading or armed.
func (g *Game) updateTrapMine(pr *Prop, dt float64) {
	if pr.used || pr.gone {
		return
	}
	if pr.armT > 0 {
		pr.armT -= dt
		if int(pr.armT*8)%2 == 0 {
			g.burst(Vec{pr.X, pr.Y - 16}, 1, 30, rgb(0xffd040), 0.3, 1, 60)
		}
		if pr.armT <= 0 {
			pr.armT, pr.armWait = 0, true
		}
		return
	}
	if pr.armWait { // loaded: it only arms when nobody of the team is inside the radius
		if g.playerNear(pr) {
			return
		}
		pr.armWait, pr.armed, pr.armedAt = false, true, g.time
		g.snd.Play("spark")
		return
	}
	if !pr.armed {
		return
	}
	if g.trapDetects(pr) || cakeGoesOff(pr) { // the cake does not wait for anybody: it blows up once armed (trapcake.go)
		pr.armed = false
		trapSetOff = g.trapSetters(pr)
		g.triggerTrap(pr)
		trapSetOff = nil
	}
}

// trapSetOff: the guests who set off the trap going off right now (inside trapDetectR). The traps whose effect comes
// a moment later (gas, fireworks, the piano) always catch them: a fast guest (the kids on wheels, the skaters) used
// to walk through the gas or out from under the piano untouched (2026-11-12).
var trapSetOff []*Guest

// trapSetters: the living guests inside the detection radius of pr.
func (g *Game) trapSetters(pr *Prop) []*Guest {
	c := trapCentre(pr)
	var out []*Guest
	for _, gu := range g.guests {
		if !gu.out() && dist(gu.pos, c) < trapDetectR {
			out = append(out, gu)
		}
	}
	return out
}

// trapDetects: somebody is within trapDetectR of the armed trap.
func (g *Game) trapDetects(pr *Prop) bool {
	c := trapCentre(pr)
	near := func(v Vec) bool { return dist(v, c) < trapDetectR }
	for _, gu := range g.guests {
		if !gu.out() && near(gu.pos) {
			return true
		}
	}
	for _, gd := range g.guards {
		if !gd.dead && near(gd.pos) {
			return true
		}
	}
	for _, cp := range g.cops {
		if near(cp.pos) { // police, soldiers and medics alike
			return true
		}
	}
	for _, a := range g.animals {
		if !a.dead && near(a.pos) {
			return true
		}
	}
	for _, s := range g.allSlashers() {
		if s.state != slGone && near(s.pos) {
			return true
		}
	}
	return g.playerNear(pr)
}

// playerNear: a living, visible player (J1 or J2, whoever armed it) is within trapDetectR.
func (g *Game) playerNear(pr *Prop) bool {
	c := trapCentre(pr)
	for _, p := range g.players() {
		if p != nil && !p.dead && !p.hidden && dist(p.pos, c) < trapDetectR {
			return true
		}
	}
	return false
}

// drawTrapState paints the load circle or the armed LED + faint radius over a trap prop.
func (g *Game) drawTrapState(dst *ebiten.Image, pr *Prop, x, y float64) {
	if !trapProp(pr) || pr.used || pr.gone {
		return
	}
	cy := y - float64(pr.K.H) - 12
	ly := y - float64(pr.K.H) + 5 // the LED sits on the top of the prop
	switch {
	case pr.armT > 0:
		f := 1 - pr.armT/trapArmTime
		drawChargeArc(dst, math.Round(x), math.Round(cy), 7, f, rgb(trapChargeCol))
	case pr.armWait:
		// loaded, waiting for the team to leave the radius: a full circle
		drawChargeArc(dst, math.Round(x), math.Round(cy), 7, 1, rgb(trapChargeCol))
	case pr.armed:
		fc := trapCentre(pr)
		gx, gy := g.wx(fc.X), g.wy(fc.Y)
		ringAt(dst, gx, gy, trapDetectR*wk, 1, color.RGBA{76, 24, 16, 76})
		if int((g.time-pr.armedAt)*2.5)%2 == 0 {
			fillRect(dst, math.Round(x)-1, ly-1, 3, 3, rgb(0xff2828))
			fillRect(dst, math.Round(x)-2, ly-2, 5, 5, color.RGBA{60, 8, 8, 60})
		} else {
			fillRect(dst, math.Round(x)-1, ly-1, 3, 3, rgb(0x602020))
		}
	}
}

// drawChargeArc: a clock-like circle at (cx, cy) with radius r, filled clockwise from the top up to f (0..1).
func drawChargeArc(dst *ebiten.Image, cx, cy, r, f float64, col color.RGBA) {
	f = math.Max(0, math.Min(1, f))
	ringAt(dst, cx, cy, r+1, 1, color.RGBA{16, 10, 6, 150}) // dark backing
	const steps = 72
	for i := 0; i < steps; i++ {
		a := float64(i) / steps
		ang := a*2*math.Pi - math.Pi/2
		px, py := cx+math.Cos(ang)*r, cy+math.Sin(ang)*r
		c := color.RGBA{col.R / 4, col.G / 4, col.B / 4, 140}
		if a < f {
			c = col
		}
		fillRect(dst, math.Round(px), math.Round(py), 1, 1, c)
		if a < f {
			fillRect(dst, math.Round(cx+math.Cos(ang)*(r-1)), math.Round(cy+math.Sin(ang)*(r-1)), 1, 1, c)
		}
	}
}

// trapDemoTick (-trapdemo): holds the two traps nearest to the player in front of the camera, one half
// loaded (the circle) and the other armed (LED and radius). Guests are kept away so nothing goes off.
func (g *Game) trapDemoTick() {
	var ts []*Prop
	for _, pr := range g.lvl.Props {
		if trapProp(pr) && !pr.used && !pr.gone {
			ts = append(ts, pr)
		}
	}
	sort.Slice(ts, func(i, j int) bool {
		return dist(g.p.pos, trapCentre(ts[i])) < dist(g.p.pos, trapCentre(ts[j]))
	})
	if len(ts) >= 1 {
		ts[0].armT, ts[0].armed, ts[0].armWait, ts[0].by = trapArmTime*0.6, false, false, g.p
	}
	if len(ts) >= 2 {
		if !ts[1].armed {
			ts[1].armed, ts[1].armedAt, ts[1].by = true, g.time, g.p
		}
		ts[1].armT = 0
	}
	for _, gu := range g.guests {
		for _, pr := range ts[:min(2, len(ts))] {
			if dist(gu.pos, trapCentre(pr)) < trapDetectR+30 {
				gu.pos = Vec{gu.pos.X + 90, gu.pos.Y + 90}
			}
		}
	}
}
