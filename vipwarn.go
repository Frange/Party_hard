package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// THE VIP WARNING LADDER. While the VIP is alive, a VIP guard that catches a player in the zone without suit, pass
// or invitation (or after an attack on the staff) gives him a warning, counted per player and per level
// (Player.vipWarns, in memory: the players are born again with every level):
//   1st: grabs him, throws him out in front of the gate and knocks him to the floor (vipKnockTime), no damage.
//   2nd: hits him, stars over his head (vipStunTime, Player.stunT) and throws him out too.
//   3rd: kills him (the normal death: revive in 2P, game over in 1P).
// After a warning the player is immune vipImmuneTime seconds (no chained warnings) and the whole staff goes back to
// its post. Killing the VIP cancels the ladder: the old behaviour (police, stars, ordinary guard chase) returns.
// The suit and the pass avoid the punishment while they last (vipIntruder does not see him; vipPunish checks too).
// ---------------------------------------------------------------------------

// vipEscalates: the ladder is in force (a VIP zone with a live VIP).
func (g *Game) vipEscalates() bool {
	if g.lvl == nil || g.lvl.vip == nil || !g.lvl.vip.spawned {
		return false
	}
	// the main VIP (the first boss) rules: once he is dead the ladder ends, even in a mini VIP party
	for _, gd := range g.guards {
		if gd.vip == vipBoss {
			return !gd.dead
		}
	}
	return false
}

// vipStaffChase: a staff member goes after the player of the alarm and, on contact, gives the warning.
func (g *Game) vipStaffChase(gd *Guard, z *vipZone, dt float64) {
	p := z.chaseP
	if p == nil || p.dead || p.hidden || p.suitT > 0 || p.vipImmT > 0 {
		g.vipStandDown()
		return
	}
	if gd.state == gdReact {
		gd.timer -= dt
		gd.dir = faceTo(gd.pos, p.pos)
		if gd.timer <= 0 {
			gd.state, gd.repath = gdChase, 0
		}
		return
	}
	gd.chaseT += dt
	if dist(gd.pos, gd.home) > gd.leash || gd.chaseT > 9 {
		g.vipStandDown()
		return
	}
	gd.repath -= dt
	if gd.repath <= 0 {
		gd.repath = 0.25
		gd.path = g.lvl.Path(gd.pos, p.pos)
	}
	g.guardWalk(gd, 57, dt)
	if gd.moving {
		gd.anim += dt * stridePhase(gd.curSpd)
	}
	if dist(gd.pos, p.pos) < 16 {
		g.vipPunish(p, gd)
	}
}

// vipStandDown: the whole staff goes back to its post.
func (g *Game) vipStandDown() {
	for _, gd := range g.guards {
		if gd.vip != 0 && gd.vip != vipBoss && !gd.dead {
			gd.state, gd.timer, gd.path, gd.chaseT = gdWait, 0, nil, 0
		}
	}
}

// vipExitPoint: just outside the gate, a little apart for each player.
func (z *vipZone) exitPoint(idx int) Vec {
	return Vec{z.gx + float64(idx*20-10), z.in.Y + z.in.H + 36}
}

// vipPunish: one more warning for p, given by guard gd.
func (g *Game) vipPunish(p *Player, gd *Guard) {
	z := g.lvl.vip
	if z == nil || p == nil || p.dead || p.vipImmT > 0 || p.suitT > 0 || p.passT > 0 {
		g.vipStandDown()
		return
	}
	p.vipWarns++
	p.vipImmT = vipImmuneTime
	lines := [3][2]string{{"¡Fuera de aquí!", "Get out of here!"}, {"¡Te lo advertí!", "I warned you!"}, {"¡Última advertencia… era la última!", "Last warning... that was the last!"}}
	n := p.vipWarns
	if n > 3 {
		n = 3
	}
	gd.icon, gd.iconT = lines[n-1][0], 2.2
	head := Vec{p.pos.X, p.pos.Y - 14}
	if n >= 3 {
		g.snd.Play("thud")
		g.shake = 0.3
		g.vipStandDown()
		g.withPlayer(p, func() { g.die("Los guardias VIP te han matado") })
		return
	}
	if n == 1 {
		p.knockT = math.Max(p.knockT, vipKnockTime)
		g.snd.Play("thud")
		g.burst(head, 6, 40, rgb(0xd8d0c0), 0.4, 2, 120)
	} else {
		p.stunT = vipStunTime
		g.snd.Play("stab")
		g.shake = 0.25
		g.burst(head, 12, 80, rgb(0xb01818), 0.5, 2, 200)
	}
	idx := g.playerIndex(p)
	p.pos = z.exitPoint(idx)
	p.dir = DirDown
	p.carrying = nil
	g.vipStandDown()
}

// drawStunStars: three yellow stars going round the head of a stunned player.
func (g *Game) drawStunStars(dst *ebiten.Image, p *Player) {
	if p == nil || p.dead {
		return
	}
	g.drawVIPStars(dst, p.pos, p.stunT > 0)
}

// Stun stars: 5x5 px, three of them going round the head at 1.5 turns per second.
const (
	vipStarSize = 5
	vipStarRX   = 11.0
	vipStarRY   = 4.0
)

func (g *Game) drawVIPStars(dst *ebiten.Image, pos Vec, on bool) {
	if !on {
		return
	}
	dark, gold, white := rgb(0x3a2a00), rgb(0xf8d820), rgb(0xffffff)
	cx, cy := math.Round(g.wx(pos.X)), math.Round(g.wy(pos.Y)-44*g.charSc)
	for i := 0; i < 3; i++ {
		a := g.time*1.5*2*math.Pi + float64(i)*2*math.Pi/3
		x, y := math.Round(cx+math.Cos(a)*vipStarRX), math.Round(cy+math.Sin(a)*vipStarRY)
		// outline: a plus 7 long under the 5-point cross, so it reads on any floor
		fillRect(dst, x-3, y-1, 7, 3, dark)
		fillRect(dst, x-1, y-3, 3, 7, dark)
		fillRect(dst, x-2, y, 5, 1, gold)
		fillRect(dst, x, y-2, 1, 5, gold)
		fillRect(dst, x-1, y-1, 3, 3, gold)
		fillRect(dst, x, y, 1, 1, white)
	}
}

// ---- ordinary guests inside the zone: the same ladder, per guest -------------------------------------------------

// vipGuestWatch: counts the immunity of the guests and sends a free guard after the first guest found in the zone.
func (g *Game) vipGuestWatch(z *vipZone, dt float64) {
	for _, gu := range g.guests {
		if gu.vipImmT > 0 {
			gu.vipImmT -= dt
		}
	}
	if !g.vipEscalates() {
		z.guestGd, z.guestTgt = nil, nil
		return
	}
	if z.guestGd != nil && (z.guestGd.dead || z.guestTgt == nil || z.guestTgt.out() || !z.inside(z.guestTgt.pos.X, z.guestTgt.pos.Y) || z.guestT > 10) {
		z.guestGd, z.guestTgt = nil, nil
	}
	if z.guestGd != nil || z.relieving() || z.alarmed {
		z.guestT += dt
		return
	}
	for _, gu := range g.guests {
		if gu.out() || gu.py != nil || gu.perf != "" || gu.vipImmT > 0 || gu.state == gLift || gu.state == gSleep || gu.vipStun > 0 || !z.inside(gu.pos.X, gu.pos.Y) {
			continue
		}
		var best *Guard
		for _, gd := range g.guards {
			if gd.vip != 0 && gd.vip != vipBoss && !gd.dead && gd.state == gdWait && (best == nil || dist(gd.pos, gu.pos) < dist(best.pos, gu.pos)) {
				best = gd
			}
		}
		if best != nil {
			z.guestGd, z.guestTgt, z.guestT = best, gu, 0
		}
		return
	}
}

// vipGuestRun: the assigned guard walks to the guest and punishes him; true while he is busy with it.
func (g *Game) vipGuestRun(gd *Guard, z *vipZone, dt float64) bool {
	gu := z.guestTgt
	if gu == nil || gu.out() {
		z.guestGd, z.guestTgt = nil, nil
		return false
	}
	gd.vipRep -= dt
	if len(gd.path) == 0 || gd.vipRep <= 0 {
		gd.path = g.lvl.Path(gd.pos, gu.pos)
		gd.vipRep = 0.4
	}
	g.guardWalk(gd, 57, dt)
	if gd.moving {
		gd.anim += dt * stridePhase(gd.curSpd)
	}
	if dist(gd.pos, gu.pos) < 16 {
		g.vipPunishGuest(gu, gd)
		z.guestGd, z.guestTgt = nil, nil
		gd.path = nil
	}
	return true
}

// vipPunishGuest: the ladder for a guest. The third warning is a death with no credit for anybody.
func (g *Game) vipPunishGuest(gu *Guest, gd *Guard) {
	z := g.lvl.vip
	gu.vipWarns++
	gu.vipImmT = vipImmuneTime
	lines := [3]string{"¡Fuera de aquí!", "¡Te lo advertí!", "¡Última advertencia… era la última!"}
	gd.icon, gd.iconT = lines[minI(gu.vipWarns, 3)-1], 2.2
	head := Vec{gu.pos.X, gu.pos.Y - 14}
	if gu.vipWarns >= 3 {
		g.snd.Play("thud")
		credit, cause := g.credit, g.killCause
		g.credit, g.killCause = false, kcNone
		g.killGuest(gu, false)
		g.credit, g.killCause = credit, cause
		return
	}
	k := 0
	for i, o := range g.guests {
		if o == gu {
			k = i
		}
	}
	if gu.vipWarns == 1 {
		gu.state, gu.timer = gSleep, vipKnockTime
		gu.icon, gu.iconT = "¡Perdone!", vipKnockTime
		g.snd.Play("thud")
		g.burst(head, 6, 40, rgb(0xd8d0c0), 0.4, 2, 120)
	} else {
		gu.state, gu.timer = gIdle, 1
		gu.vipStun = vipStunTime
		g.snd.Play("stab")
		g.burst(head, 12, 80, rgb(0xb01818), 0.5, 2, 200)
	}
	gu.path = nil
	gu.pos = Vec{z.gx + float64(k%5-2)*14, z.in.Y + z.in.H + 36}
}

func init() {
	for es, en := range map[string]string{
		"¡Fuera de aquí!":                     "Get out of here!",
		"¡Te lo advertí!":                     "I warned you!",
		"¡Última advertencia… era la última!": "Last warning... that was the last!",
		"Los guardias VIP te han matado":      "The VIP guards killed you",
		"¡Perdone!":                           "Pardon me!",
	} {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}
