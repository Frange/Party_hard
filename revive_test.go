package main

import "testing"

// reviveGame is a two-player party with the paramedics' look ready.
func reviveGame() *Game {
	g := twoGame(true)
	sp := medicSpec()
	g.as.medicCS = newCharSet(&sp)
	g.mode = ModePlay
	return g
}

// kill makes player pl die the way g.die does, for the given reason.
func kill(g *Game, pl *Player, why string) {
	g.withPlayer(pl, func() { g.die(why) })
}

// runRescue advances the party until the rescue is over (or 120 s), recording the phases it goes through.
func runRescue(g *Game) []int {
	var seen []int
	last := -1
	for i := 0; i < 1200 && (g.rev.job != nil || i == 0); i++ {
		g.time += 0.1
		g.updateVehicles(0.1)
		g.updateTwoPlayers(0.1)
		if j := g.rev.job; j != nil && j.phase != last {
			last = j.phase
			seen = append(seen, last)
		}
	}
	return seen
}

func TestReviveOncePerParty(t *testing.T) {
	g := reviveGame()
	kill(g, g.p2, "Un asesino te ha dado caza")
	if !g.p2.dead || g.p2.deathWhy == "" {
		t.Fatal("the second player must be dead and remember why")
	}
	if g.mode != ModePlay {
		t.Fatal("one player is still alive: the party goes on")
	}
	if !g.canCallAmbulance(g.p) {
		t.Fatal("the living player must be able to call the ambulance")
	}
	if g.canCallAmbulance(g.p2) {
		t.Fatal("a dead player cannot call")
	}
	g.callAmbulance(g.p, nil)
	if !g.rev.used || g.rev.job == nil {
		t.Fatal("the call did not start a rescue")
	}
	if g.canCallAmbulance(g.p) {
		t.Fatal("nobody can call twice while the ambulance is coming")
	}
	phases := runRescue(g)
	if g.p2.dead {
		t.Fatalf("the partner was not revived, phases %v", phases)
	}
	if g.rev.done != 1 || g.p2.knockT <= 0 || g.p2.msg == "" {
		t.Error("he must sit up slowly and say something")
	}
	// the phases happen in order
	want := []int{rvWait, rvDrive, rvWalk, rvCheck, rvCharge, rvShock1, rvCharge2, rvShock2, rvUp, rvBack}
	for i, w := range want {
		if i >= len(phases) || phases[i] != w {
			t.Fatalf("phases %v, wanted %v", phases, want)
		}
	}
	if g.rev.job != nil {
		t.Error("the rescue must be over and the paramedics gone")
	}
	// the call was spent: a second death in the same party is final
	kill(g, g.p2, "Un asesino te ha dado caza")
	if g.canCallAmbulance(g.p) {
		t.Error("only ONE call per party")
	}
}

func TestReviveDelayIsReasonable(t *testing.T) {
	g := reviveGame()
	kill(g, g.p2, "Te ha atropellado un coche")
	g.callAmbulance(g.p, nil)
	if e := g.rev.job.eta; e < 6 || e > 25 {
		t.Errorf("the ambulance should take 10-15 s or so, the HUD says %.1f", e)
	}
}

func TestNoReviveAfterTRexFireOrExplosion(t *testing.T) {
	for _, why := range []string{"Te ha comido el T-Rex", "Un T-Rex te ha pisoteado", "Te has achicharrado", "Has saltado por los aires con tu propia trampa", "Un cohete de tu trampa te ha dado", "Un zombi te ha mordido"} {
		g := reviveGame()
		kill(g, g.p2, why)
		if g.reviveTarget() != nil || g.canCallAmbulance(g.p) {
			t.Errorf("%q cannot be revived", why)
		}
	}
	for _, why := range []string{"Un asesino te ha dado caza", "Te ha caído un piano encima", "Te ha atropellado un coche", "Te ha mordido una serpiente"} {
		g := reviveGame()
		kill(g, g.p2, why)
		if g.reviveTarget() == nil {
			t.Errorf("%q can be revived", why)
		}
	}
}

func TestBothDeadIsGameOver(t *testing.T) {
	g := reviveGame()
	kill(g, g.p2, "Un asesino te ha dado caza")
	g.callAmbulance(g.p, nil)
	kill(g, g.p, "Un asesino te ha dado caza")
	if g.mode != ModeLose {
		t.Error("with both players dead the party is lost, whatever the ambulance does")
	}
}

func TestReviveNeedsTwoPlayers(t *testing.T) {
	g := twoGame(false)
	g.mode = ModePlay
	g.p.dead, g.p.deathWhy = true, "x"
	if g.canCallAmbulance(g.p) || g.reviveTarget() != nil {
		t.Error("with one player there is nobody to revive")
	}
	g.updateTwoPlayers(0.1)
	if g.rev.used || g.rev.job != nil {
		t.Error("nothing may change with one player")
	}
}

func TestCutPhoneCannotCall(t *testing.T) {
	g := reviveGame()
	var ph *Prop
	for _, pr := range g.lvl.Props {
		if pr.K.Phone {
			ph = pr
			break
		}
	}
	if ph == nil {
		t.Skip("the level has no phone booth")
	}
	if lab, _ := g.useLabel(ph); lab != "Cortar cable" {
		t.Fatalf("with both alive the phone cuts the line: %q", lab)
	}
	kill(g, g.p2, "Un asesino te ha dado caza")
	if lab, _ := g.useLabel(ph); lab != "Llamar a la ambulancia" {
		t.Fatalf("with a dead partner the phone calls the ambulance: %q", lab)
	}
	ph.used = true // the line was cut
	if lab, _ := g.useLabel(ph); lab == "Llamar a la ambulancia" {
		t.Error("a cut phone cannot call")
	}
	ph.used = false
	// pressing the use key at the phone makes the call
	g.p.pos = Vec{ph.Foot.X + ph.Foot.W/2, ph.Foot.Y + ph.Foot.H + 2}
	g.interact()
	if !g.rev.used {
		t.Error("the use key at the phone must call the ambulance")
	}
	if ph.used {
		t.Error("calling the ambulance does not cut the line")
	}
}

func TestDebugTwoDeadKillsPlayer(t *testing.T) {
	old, oldTwo := dbgTwoDead, dbgTwoPlayers
	dbgTwoDead, dbgTwoPlayers = 2, true
	defer func() { dbgTwoDead, dbgTwoPlayers = old, oldTwo }()
	g := reviveGame()
	g.p = g.first()
	g.setupPlayers(g.lvl.Def)
	if !g.p2.dead {
		t.Error("-twodead 2 must start with the second player dead")
	}
	g.time = 2
	g.updateRevive(0.1)
	if !g.rev.used {
		t.Error("with -twodead the living player calls by himself")
	}
}

func TestReviveDoesNotCallThePolice(t *testing.T) {
	g := reviveGame()
	kill(g, g.p2, "Te ha atropellado un coche")
	g.callAmbulance(g.p, nil)
	runRescue(g)
	if g.policeCalled || g.pendingCar >= 0 {
		t.Error("the ambulance of the rescue must not call the police")
	}
}
