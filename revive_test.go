package main

import (
	"testing"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// reviveGame is a two-player party with the paramedics' look ready.
func reviveGame() *Game {
	g := twoGame(true)
	g.face = text.NewGoXFace(bitmapfont.Face) // the font of the game (newGame): the notice box measures its texts
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

func TestReviveOncePerPlayerPerParty(t *testing.T) {
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
	if !g.reviveSpent(g.p2) || g.reviveSpent(g.p) || g.rev.job == nil {
		t.Fatal("the call must start a rescue and spend the ambulance of J2 only")
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
	// J2 spent his ambulance: a second death of J2 in the same party is final
	kill(g, g.p2, "Un asesino te ha dado caza")
	if g.canCallAmbulance(g.p) || g.reviveTarget() != nil {
		t.Fatal("only ONE revival per player and party: J2 stays dead")
	}
	if g.reviveHudLine() != "J2 ya gastó su ambulancia" {
		t.Errorf("the HUD must say J2 has no ambulance left: %q", g.reviveHudLine())
	}
	if _, ok := g.reviveNoticeBox(0); ok {
		t.Error("no notice for a partner that cannot be saved")
	}
}

func TestReviveEachPlayerHasHisOwn(t *testing.T) {
	g := reviveGame()
	if got := g.reviveHudLine(); got != "Ambulancia: J1 1 · J2 1" {
		t.Fatalf("at the start each player has one ambulance: %q", got)
	}
	kill(g, g.p, "Un asesino te ha dado caza") // J1 dies first, J2 calls
	if !g.canCallAmbulance(g.p2) {
		t.Fatal("J2 must be able to call for J1")
	}
	g.callAmbulance(g.p2, nil)
	runRescue(g)
	if g.p.dead {
		t.Fatal("J1 was not revived")
	}
	if got := g.reviveHudLine(); got != "Ambulancia: J1 0 · J2 1" {
		t.Errorf("J1 spent his, J2 keeps his: %q", got)
	}
	// later in the same party J2 dies: J1 can still bring HIM back
	g.p.knockT = 0 // he got up a while ago (runRescue only moves the rescue, not the players)
	kill(g, g.p2, "Te ha caído un piano encima")
	if !g.canCallAmbulance(g.p) {
		t.Fatal("J2 still has his own ambulance in this party")
	}
	g.callAmbulance(g.p, nil)
	runRescue(g)
	if g.p2.dead || g.rev.done != 2 {
		t.Fatalf("J2 must be revived too (done=%d)", g.rev.done)
	}
	// and now neither of them has any left
	kill(g, g.p, "Un asesino te ha dado caza")
	if g.canCallAmbulance(g.p2) {
		t.Error("J1 already had his ambulance")
	}
}

func TestReviveResetsWithANewParty(t *testing.T) {
	g := reviveGame()
	kill(g, g.p2, "Un asesino te ha dado caza")
	g.callAmbulance(g.p, nil)
	runRescue(g)
	if !g.reviveSpent(g.p2) {
		t.Fatal("J2 spent his ambulance")
	}
	old := twoPlayerMode
	twoPlayerMode = true
	defer func() { twoPlayerMode = old }()
	g.setupPlayers(g.lvl.Def) // what loadLevel does for every party
	if g.reviveSpent(g.p) || g.reviveSpent(g.p2) || g.rev.done != 0 || g.rev.job != nil {
		t.Fatal("a new party gives each player his ambulance again")
	}
	g.mode = ModePlay
	kill(g, g.p2, "Un asesino te ha dado caza")
	if !g.canCallAmbulance(g.p) {
		t.Error("after loading a party J2 can be revived again")
	}
}

// From the moment the paramedics reach the body to the partner getting up: about 10 seconds, with the two shocks.
func TestReviveTakesTenSecondsAtTheBody(t *testing.T) {
	if revWorkT < 9.5 || revWorkT > 10.5 {
		t.Fatalf("the work by the body adds up to %.1f s, wanted about 10", revWorkT)
	}
	g := reviveGame()
	kill(g, g.p2, "Te ha atropellado un coche")
	g.callAmbulance(g.p, nil)
	const dt = 1.0 / 60
	arrived, up := -1.0, -1.0
	shocks := 0
	lastPhase := -1
	for i := 0; i < 60*120 && up < 0; i++ {
		g.time += dt
		g.updateVehicles(dt)
		g.updateTwoPlayers(dt)
		j := g.rev.job
		if j == nil {
			break
		}
		if j.phase != lastPhase {
			if j.phase == rvCheck {
				arrived = g.time
			}
			if j.phase == rvShock1 || j.phase == rvShock2 {
				shocks++
				if j.phase == rvShock1 && !g.p2.dead {
					t.Fatal("the first shock must fail")
				}
			}
			lastPhase = j.phase
		}
		if arrived >= 0 && !g.p2.dead {
			up = g.time
		}
	}
	if arrived < 0 || up < 0 {
		t.Fatalf("the rescue did not finish (arrived %.1f, up %.1f)", arrived, up)
	}
	if d := up - arrived; d < 9.5 || d > 10.5 {
		t.Errorf("from the paramedics at the body to the partner up: %.2f s, wanted about 10", d)
	}
	if shocks != 2 {
		t.Errorf("two shocks (the first fails), got %d", shocks)
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
	if g.rev.spent != [2]bool{} || g.rev.job != nil || g.reviveHudH() != 14 {
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
	if !g.reviveSpent(g.p2) || g.rev.job == nil {
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
	if !g.reviveSpent(g.p2) || g.rev.job == nil {
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
