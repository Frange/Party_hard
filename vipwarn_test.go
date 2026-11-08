package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// warnGame: a VIP level where player 1 stands inside the zone with nothing on.
func warnGame(t *testing.T, two bool) *Game {
	prevF := progressFrozen
	progressFrozen = true // the death below must never write the real progress
	t.Cleanup(func() { progressFrozen = prevF })
	g := vipGame(t, "PLAZA DE TOROS", two)
	for _, pl := range g.players() {
		pl.pos = g.lvl.Def.Start
	}
	return g
}

// warnOnce: puts p inside and ticks until a warning lands (or 10 s pass).
func warnOnce(g *Game, p *Player) bool {
	n := p.vipWarns
	p.pos = inside(g)
	for i := 0; i < 600 && p.vipWarns == n; i++ {
		p.pos = inside(g)
		vipTick(g, 0.1)
	}
	return p.vipWarns > n
}

func TestVIPHalvedTimes(t *testing.T) {
	if vipPassTime != 15 || vipShiftWindow != 3.5 {
		t.Fatalf("pass %v window %v: must be half of 30 and 7", vipPassTime, vipShiftWindow)
	}
	if suitDuration != 8 {
		t.Error("the gold suit is a general object: unchanged")
	}
}

func TestVIPWarningLadder(t *testing.T) {
	g := warnGame(t, false)
	p, z := g.p, g.lvl.vip
	if !warnOnce(g, p) || p.vipWarns != 1 {
		t.Fatal("first warning expected")
	}
	if p.knockT <= 0 || p.dead || p.stunT > 0 || z.touches(p) {
		t.Fatalf("1st: knocked down, alive, thrown out (knock %v dead %v inside %v)", p.knockT, p.dead, z.touches(p))
	}
	if !allStanding(g) {
		t.Error("the staff goes back to its post")
	}
	// the immunity: walking in again at once changes nothing
	p.knockT = 0
	p.pos = inside(g)
	vipTick(g, 2)
	if p.vipWarns != 1 {
		t.Fatal("3 s of immunity: no chained warning")
	}
	vipTick(g, 1.5)
	if !warnOnce(g, p) || p.vipWarns != 2 {
		t.Fatalf("second warning expected, got %d", p.vipWarns)
	}
	if p.stunT <= 0 || p.dead || z.touches(p) {
		t.Fatalf("2nd: stunned with stars and thrown out (stun %v)", p.stunT)
	}
	g.updatePlayerStunOnly(p)
	p.stunT, p.vipImmT = 0, 0
	if !warnOnce(g, p) || !p.dead {
		t.Fatal("3rd warning kills")
	}
}

// updatePlayerStunOnly: drawing the stunned player must not panic.
func (g *Game) updatePlayerStunOnly(p *Player) {
	img := ebiten.NewImage(screenW, screenH)
	g.drawStunStars(img, p)
}

func TestVIPWarningsPerPlayerAndLevel(t *testing.T) {
	g := warnGame(t, true)
	a, b := g.players()[0], g.players()[1]
	if !warnOnce(g, a) || a.vipWarns != 1 || b.vipWarns != 0 {
		t.Fatalf("per player: %d %d", a.vipWarns, b.vipWarns)
	}
	b.pos = g.lvl.Def.Start
	a.pos = g.lvl.Def.Start
	if !warnOnce(g, b) || b.vipWarns != 1 || a.vipWarns != 1 {
		t.Fatalf("2P second player has his own counter: %d %d", a.vipWarns, b.vipWarns)
	}
	if b.pos == a.pos {
		t.Log("exit points are spread")
	}
	// 2P third warning: the revive flow (the other one lives, the game goes on)
	a.vipImmT, b.vipImmT, a.vipWarns = 0, 0, 2
	a.pos, b.pos = g.lvl.Def.Start, g.lvl.Def.Start
	if !warnOnce(g, a) || !a.dead || g.mode == ModeLose {
		t.Fatal("2P: one dies, the other goes on")
	}
	g2 := warnGame(t, false)
	if g2.p.vipWarns != 0 {
		t.Error("a new level starts at zero")
	}
}

func TestVIPSuitAndInviteAvoidPunishment(t *testing.T) {
	g := warnGame(t, false)
	g.p.suitT = 30
	g.p.pos = inside(g)
	vipTick(g, 3)
	if g.p.vipWarns != 0 || !allStanding(g) {
		t.Fatal("suit: no punishment")
	}
	g.p.suitT = 0
	g.p.invite = true
	g.p.pos = g.lvl.vip.exitPoint(0)
	g.p.pos.Y -= 10 // the gate
	vipTick(g, 1)
	g.p.pos = inside(g)
	vipTick(g, 3)
	if g.p.vipWarns != 0 {
		t.Fatal("pass: no punishment")
	}
	// a pass that runs out inside: the ladder starts
	g.p.passT = 0
	if !warnOnce(g, g.p) {
		t.Fatal("without pass the warning comes")
	}
}

func TestVIPKillingTheVIPCancelsTheLadder(t *testing.T) {
	g := warnGame(t, false)
	boss := vipStaff(g, vipBoss)[0]
	g.p.pos = Vec{boss.pos.X + 10, boss.pos.Y}
	g.killGuard(boss)
	if g.vipEscalates() {
		t.Fatal("dead VIP: no ladder")
	}
	g.p.pos = inside(g)
	vipTick(g, 6)
	if g.p.vipWarns != 0 {
		t.Fatal("no warnings after the murder")
	}
}

func TestVIPStunnedPlayerCannotMove(t *testing.T) {
	g := warnGame(t, false)
	g.p.stunT = 1
	g.p.moving = true
	g.updatePlayer(1.0 / 60)
	if g.p.moving || g.p.stunT >= 1 {
		t.Error("a stunned player stands there and the timer runs")
	}
}

func guestInZone(g *Game) *Guest {
	gu := &Guest{cs: g.plainGuestCS(), pos: inside(g), state: gIdle}
	g.guests = append(g.guests, gu)
	return gu
}

func guestWarn(g *Game, gu *Guest) bool {
	n := gu.vipWarns
	for i := 0; i < 400 && gu.vipWarns == n && !gu.out(); i++ {
		if !gu.out() && gu.vipImmT <= 0 {
			gu.pos = inside(g)
		}
		vipTick(g, 0.1)
	}
	return gu.vipWarns > n || gu.out()
}

func TestVIPGuestLadder(t *testing.T) {
	g := warnGame(t, false)
	g.total = 1
	other := &Guest{cs: g.plainGuestCS(), pos: g.lvl.Def.Start, state: gIdle}
	g.guests = append(g.guests, other)
	gu := guestInZone(g)
	pts, stars, kills := g.points, g.stars, g.kills
	if !guestWarn(g, gu) || gu.vipWarns != 1 || gu.state != gSleep || g.lvl.vip.trig.Has(gu.pos.X, gu.pos.Y) {
		t.Fatalf("1st: knocked down and out (warns %d state %d)", gu.vipWarns, gu.state)
	}
	if other.vipWarns != 0 {
		t.Error("counter per guest")
	}
	vipTick(g, 3.5)
	gu.state = gIdle
	if !guestWarn(g, gu) || gu.vipWarns != 2 || gu.vipStun <= 0 || gu.out() {
		t.Fatalf("2nd: stunned (warns %d stun %v)", gu.vipWarns, gu.vipStun)
	}
	g.drawVIPStars(ebiten.NewImage(screenW, screenH), gu.pos, true)
	gu.vipStun, gu.vipImmT = 0, 0
	before := g.aliveCount()
	if !guestWarn(g, gu) || !gu.dead {
		t.Fatal("3rd: dies")
	}
	if g.aliveCount() != before-1 {
		t.Errorf("alive %d -> %d", before, g.aliveCount())
	}
	if g.points != pts || g.stars != stars || g.kills != kills || g.p.dead {
		t.Errorf("no points/stars/kills for the player: %d %d %d", g.points-pts, g.stars-stars, g.kills-kills)
	}
	for _, b := range g.bodies {
		if b.byPlayer {
			t.Error("body must not be the player's")
		}
	}
	if g.policeCalled {
		t.Error("no police")
	}
}

func TestVIPGuestNoLadderAfterVIPDies(t *testing.T) {
	g := warnGame(t, false)
	g.killGuard(vipStaff(g, vipBoss)[0])
	gu := guestInZone(g)
	vipTick(g, 3)
	if gu.vipWarns != 0 {
		t.Error("no guest ladder when the VIP is dead")
	}
}

func TestStunStarsAreBigEnough(t *testing.T) {
	g := &Game{}
	img := ebiten.NewImage(screenW, screenH)
	p := &Player{stunT: 1}
	g.drawStunStars(img, p) // must not panic
	if vipStarSize < 5 || vipStarRX < 10 || vipStarRX > 12 {
		t.Fatalf("stars %dpx radius %v: too small to see", vipStarSize, vipStarRX)
	}
}
