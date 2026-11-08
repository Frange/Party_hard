package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// vipLevelDef: the generated campaign level with that title.
func vipLevelDef(t *testing.T, title string) *LevelDef {
	t.Helper()
	for _, e := range campaignLevels {
		if e.title == title && e.kind != "" {
			d := genNamed(e.kind, e.seed, e.title)
			if d == nil {
				t.Fatalf("%s did not generate", title)
			}
			addMurray(d)
			return d
		}
	}
	t.Fatalf("no campaign level %q", title)
	return nil
}

// vipGame: a game on a real VIP level, the staff spawned, no guests.
func vipGame(t *testing.T, title string, two bool) *Game {
	t.Helper()
	g := twoGame(two)
	old := twoPlayerMode
	twoPlayerMode = two
	t.Cleanup(func() { twoPlayerMode = old; vipEliminated = map[string]bool{} })
	g.lvl = vipBuildWith(vipLevelDef(t, title), vipPlan[title])
	if g.lvl.vip == nil {
		t.Fatalf("%s has no VIP zone", title)
	}
	g.setupPlayers(g.lvl.Def)
	g.guests, g.guards, g.cops, g.bodies = nil, nil, nil, nil
	g.visK = 1
	g.mode = ModePlay
	g.policeCalled, g.pendingCar, g.pendingUnit = false, -1, -1
	g.spawnVIP()
	if !g.lvl.vip.spawned {
		t.Fatal("staff not spawned")
	}
	return g
}

func vipTick(g *Game, secs float64) {
	for i := 0; i < int(secs*60); i++ {
		g.time += 1.0 / 60
		g.updateSuit(1.0 / 60)
		g.updateVIP(1.0 / 60)
		for _, gd := range g.guards {
			restore := g.focus(gd.pos)
			g.updateGuard(gd, 1.0/60)
			restore()
		}
	}
}

func vipStaff(g *Game, role int) []*Guard {
	var out []*Guard
	for _, gd := range g.guards {
		if gd.vip == role {
			out = append(out, gd)
		}
	}
	return out
}

func allStanding(g *Game) bool {
	for _, gd := range g.guards {
		if gd.vip != 0 && gd.vip != vipBoss && gd.state != gdWait {
			return false
		}
	}
	return true
}

func inside(g *Game) Vec {
	z := g.lvl.vip
	return Vec{z.gx, z.in.Y + z.in.H - 12}
}

func TestVIPRegistry(t *testing.T) {
	reg := vipRegistry()
	if len(reg) < 10 {
		t.Fatalf("only %d VIPs, want 10 or more", len(reg))
	}
	forbidden := []string{"TRUMP", "BIDEN", "OBAMA", "FRANCISCO", "BENEDICTO", "FELIPE", "ELVIS", "PUTIN", "MUSK", "TYSON", "ALI", "BEZOS", "ARMSTRONG", "GAGARIN", "JUAN CARLOS", "ELIZABETH", "CHARLES"}
	ids, names := map[string]bool{}, map[string]bool{}
	for _, d := range reg {
		if d.id == "" || ids[d.id] || names[d.es] || names[d.en] {
			t.Errorf("duplicated or empty VIP %q", d.id)
		}
		ids[d.id], names[d.es], names[d.en] = true, true, true
		if d.es == "" || d.en == "" || d.rankES == "" || d.rankEN == "" || d.bonus <= 0 || d.hp < 1 || d.spec == nil || len(d.lines) < 3 {
			t.Errorf("%s: incomplete definition", d.id)
		}
		for _, f := range forbidden {
			if strings.Contains(strings.ToUpper(d.es+" "+d.en), f) {
				t.Errorf("%s: name contains %q (real name)", d.id, f)
			}
		}
		for _, ln := range d.lines {
			if len([]rune(ln[0])) > 48 || len([]rune(ln[1])) > 48 || ln[0] == "" || ln[1] == "" {
				t.Errorf("%s: line too long or empty: %q / %q", d.id, ln[0], ln[1])
			}
		}
		sp := d.spec()
		if cs := newCharSet(&sp); cs == nil || sp.Name != "" {
			t.Errorf("%s: bad spec", d.id)
		}
		if d.behaviour == vbTip && (d.tipES == "" || d.tipEN == "") {
			t.Errorf("%s: a tipping VIP needs tip texts", d.id)
		}
	}
	kinds := map[int]bool{}
	for _, d := range reg {
		kinds[d.behaviour] = true
	}
	for _, b := range []int{vbFlee, vbCall, vbTip, vbTough, vbStand} {
		if !kinds[b] {
			t.Errorf("no VIP with behaviour %d", b)
		}
	}
	if len(vipIDs()) != len(reg) || len(vipSnapshot()) < 0 {
		t.Error("registry helpers")
	}
	// every text has its English version
	for _, d := range reg {
		for _, ln := range d.lines {
			if langDataEN[ln[0]] != ln[1] {
				t.Errorf("%s: no English entry for %q", d.id, ln[0])
			}
		}
	}
}

// vipQuality: the checks every zone must pass (inside the map, cordon, no overlaps, DJ, cameras, reach).
func vipQuality(t *testing.T, title string, l *Level) {
	t.Helper()
	z := l.vip
	if z.in.X < 0 || z.in.Y < 0 || z.in.X+z.in.W > l.W || z.in.Y+z.in.H > l.H {
		t.Errorf("%s: zone out of the map", title)
	}
	if countKind(l, "vipsuit") < 1 || countKind(l, "viprope_h") < 6 || countKind(l, "viprope_v") < 2 {
		t.Errorf("%s: cordon or loot missing (suits %d)", title, countKind(l, "vipsuit"))
	}
	if iss := levelOverlaps(l); len(iss) > 0 {
		t.Errorf("%s: overlaps %v", title, iss)
	}
	if iss := djCoverIssues(l); len(iss) > 0 {
		t.Errorf("%s: DJ covered %v", title, iss)
	}
	area := z.in.Grow(14)
	for _, c := range placeCameras(l, 3) {
		if camBodyRect(c.mount).Hits(area) {
			t.Errorf("%s: a camera sits on the zone", title)
		}
	}
	reach := l.reachFrom(l.Def.Start)
	if !reach[int((z.in.Y+z.in.H+34)/navCell)*l.cw+int(z.gx/navCell)] {
		t.Errorf("%s: gate not reachable", title)
	}
	for _, d := range l.Def.Doors {
		if d.R.Hits(area) {
			t.Errorf("%s: zone on a door", title)
		}
	}
	for _, pf := range l.Def.Performers {
		if area.Has(pf.Pos.X, pf.Pos.Y) {
			t.Errorf("%s: zone on a performer", title)
		}
	}
	if area.Has(l.Def.Start.X, l.Def.Start.Y) {
		t.Errorf("%s: zone on the start", title)
	}
}

// vipBuildWith: the level of a definition with that VIP zone (what loadLevel does after the dice).
func vipBuildWith(def *LevelDef, id string) *Level {
	vipDecision = id
	defer func() { vipDecision = "" }()
	return buildLevel(def)
}

// Every level where the carpet fits (adaptive placement) gets a clean zone; the affinity table is consistent.
func TestVIPPlanPerLevel(t *testing.T) {
	used := map[string]string{}
	for title, id := range vipPlan {
		if vipByID(id) == nil {
			t.Errorf("%s: unknown VIP %q", title, id)
		}
		if o, dup := used[id]; dup {
			t.Errorf("%s and %s share the VIP %s", title, o, id)
		}
		used[id] = title
	}
	eligible := 0
	for i, d := range builtinLevels() {
		l := vipBuildWith(d, "king")
		if l.vip == nil {
			continue
		}
		eligible++
		vipQuality(t, d.Title+"#"+itoa(i), l)
	}
	t.Logf("%d levels (campaign + random pool) have room for a zone", eligible)
	if eligible < 10 {
		t.Errorf("only %d eligible levels", eligible)
	}
	// without a decision there is no zone, whatever the level
	if l := buildLevel(vipLevelDef(t, "PLAZA DE TOROS")); l.vip != nil {
		t.Error("buildLevel without a decision must not add a zone")
	}
}

func TestVIPGuardsStandStillThenWakeOnEntry(t *testing.T) {
	g := vipGame(t, "PLAZA DE TOROS", false)
	if len(vipStaff(g, vipDoor)) != 2 || len(vipStaff(g, vipInner)) != 1 || len(vipStaff(g, vipBoss)) != 1 {
		t.Fatal("2 door guards + 1 inner + the VIP expected")
	}
	g.p.pos = g.lvl.Def.Start
	posts := map[*Guard]Vec{}
	for _, gd := range g.guards {
		posts[gd] = gd.pos
	}
	vipTick(g, 3)
	for gd, p := range posts {
		if gd.pos != p || gd.state != gdWait {
			t.Fatalf("staff must stand still (%v -> %v, state %d)", p, gd.pos, gd.state)
		}
	}
	g.p.pos = inside(g)
	vipTick(g, 0.5)
	woke := 0
	for _, gd := range g.guards {
		if gd.vip != 0 && gd.vip != vipBoss && gd.state != gdWait {
			woke++
		}
	}
	if woke != 3 {
		t.Fatalf("all 3 guards must wake, %d did", woke)
	}
	// the guard chase ends when he leaves and they lose him: back to the posts
	g.p.pos = g.lvl.Def.Start
	vipTick(g, 25)
	if !allStanding(g) {
		t.Error("the guards must go back to standing")
	}
}

func TestVIPSuitLetsYouIn(t *testing.T) {
	g := vipGame(t, "PLAZA DE TOROS", false)
	g.p.pos = inside(g)
	g.p.suitT = 5
	vipTick(g, 1.5)
	if !allStanding(g) {
		t.Fatal("with the suit nobody wakes")
	}
	vipTick(g, 5) // the suit ends inside the zone
	if g.p.suitT != 0 {
		t.Fatalf("suit not spent: %v", g.p.suitT)
	}
	woke := false
	for _, gd := range g.guards {
		if gd.vip != 0 && gd.vip != vipBoss && gd.state != gdWait {
			woke = true
		}
	}
	if !woke && g.p.vipWarns == 0 { // they woke and already gave the first warning (vipwarn.go)
		t.Error("once the suit ends inside, the guards wake")
	}
}

func TestVIPAttackWakesTheStaff(t *testing.T) {
	g := vipGame(t, "FESTIVAL", false)
	g.p.pos = g.lvl.Def.Start
	door := vipStaff(g, vipDoor)[0]
	g.p.pos = Vec{door.pos.X + 10, door.pos.Y}
	g.killGuard(door)
	for _, gd := range g.guards {
		if gd.vip == vipInner && gd.state == gdWait {
			t.Error("the inner guard must react to the murder")
		}
	}
	if g.aliveCount() != 0 {
		t.Errorf("staff must not count for the goal: alive %d", g.aliveCount())
	}
}

func TestSuitMakesYouInvisibleToEveryone(t *testing.T) {
	g := suspectGame()
	g.visK = 1
	g.mode = ModePlay
	g.p.pos = Vec{200, 300}
	gu := suspectGuest(g, Vec{230, 300}, DirLeft)
	c := suspectCop(g, Vec{240, 300}, DirLeft)
	g.cams = []*secCam{{mount: Vec{200, 250}, eye: Vec{200, 256}, base: 1.5708, ang: 1.5708, still: true}}
	if !g.sees(gu.pos, gu.dir, g.p.pos, guestRange, guestDot) || !g.sees(c.pos, c.dir, g.p.pos, copRange, copDot) || !g.cameraSees(g.p.pos) {
		t.Fatal("setup: without the suit they all see him")
	}
	g.p.suitT = suitDuration
	if g.sees(gu.pos, gu.dir, g.p.pos, guestRange, guestDot) || g.sees(c.pos, c.dir, g.p.pos, copRange, copDot) || g.cameraSees(g.p.pos) {
		t.Fatal("the suit must hide him from guests, cops and cameras")
	}
	g.policeCalled = false
	g.witness()
	g.cameraWitness()
	if g.policeCalled || g.wanted != 0 || g.playerIsSuspect() || gu.sawKiller {
		t.Error("a kill with the suit on must not be witnessed")
	}
	// a guard standing in front of him does not react either
	gd := &Guard{pos: Vec{225, 300}, dir: DirLeft, cs: gu.cs, route: []Vec{{225, 300}}, state: gdWait, home: Vec{225, 300}, leash: 100}
	g.guards = []*Guard{gd}
	for i := 0; i < 180; i++ {
		gd.dir = DirLeft
		g.updateGuard(gd, 1.0/60)
	}
	if gd.state == gdReact || gd.state == gdChase {
		t.Error("the guard must not notice the suit wearer")
	}
	g.p.suitT = 0
	for i := 0; i < 180; i++ {
		gd.dir = DirLeft
		g.updateGuard(gd, 1.0/60)
	}
	if gd.state != gdReact && gd.state != gdChase && gd.state != gdBeat {
		t.Errorf("without the suit the same guard notices him (state %d sight %.2f sees %v)", gd.state, gd.sight, g.sees(gd.pos, gd.dir, g.p.pos, guardRange, guardDot))
	}
}

func TestSuitPickupSingleUseAndExpiry(t *testing.T) {
	for _, two := range []bool{false, true} {
		g, pr := pickupSetup(t, two, "vipsuit")
		g.p.pos = Vec{200, 200}
		if two {
			g.otherPlayer(g.p).pos = Vec{300, 260}
		}
		g.interact()
		if g.p.suitT != suitDuration || !pr.gone || g.p.item != "" {
			t.Fatalf("two=%v: suit not worn (t=%v gone=%v item=%q)", two, g.p.suitT, pr.gone, g.p.item)
		}
		if two {
			if other := g.otherPlayer(g.p); other.suitT != 0 || g.suitHides(other.pos) {
				t.Error("the suit is per player")
			}
		}
		if canTake(g.p, "suit") {
			t.Error("cannot take a second suit while wearing one")
		}
		vipTick(g, suitDuration+0.2)
		if g.p.suitT != 0 || g.suitHides(g.p.pos) {
			t.Error("the suit must expire")
		}
	}
}

func TestVIPInvitationOneUsePerPlayer(t *testing.T) {
	for _, two := range []bool{false, true} {
		g := vipGame(t, "PLAZA DE TOROS", two)
		z := g.lvl.vip
		carrier := &Guest{cs: g.plainGuestCS(), pos: Vec{50, 50}, vipCard: true}
		g.guests = append(g.guests, carrier)
		for _, pl := range g.players() {
			pl.pos = g.lvl.Def.Start
		}
		g.killGuest(carrier, true)
		if !g.p.invite || carrier.vipCard {
			t.Fatal("killing the carrier gives the invitation")
		}
		g.p.pos = Vec{z.gx, z.in.Y + z.in.H + 4}
		vipTick(g, 1)
		if g.p.invite || g.p.passT <= 0 || !allStanding(g) {
			t.Fatalf("two=%v: the invitation must be spent at the gate without waking anybody (invite=%v pass=%v)", two, g.p.invite, g.p.passT)
		}
		g.p.passT = 0 // pass expired
		vipTick(g, 0.5)
		if allStanding(g) {
			t.Error("second time, without a card, the guards wake")
		}
		if two {
			o := g.otherPlayer(g.p)
			if o.invite || o.passT != 0 {
				t.Error("the pass is for the player who took it")
			}
		}
	}
}

func TestVIPGuardShiftOpensAWindow(t *testing.T) {
	g := vipGame(t, "PLAZA DE TOROS", false)
	z := g.lvl.vip
	g.p.pos = g.lvl.Def.Start
	vipTick(g, vipShiftEvery+vipYawnTime-0.3)
	if !z.relief() {
		t.Fatal("the shift must have started")
	}
	yawn := false
	for _, gd := range vipStaff(g, vipDoor) {
		if gd.icon != "" {
			yawn = true
		}
	}
	if !yawn {
		t.Error("the guards must yawn / look at the watch")
	}
	vipTick(g, 1.0)
	for _, gd := range vipStaff(g, vipDoor) {
		if dist(gd.pos, gd.vipPost) < 20 {
			t.Errorf("the door guard must have left his post (dist %.0f)", dist(gd.pos, gd.vipPost))
		}
	}
	g.p.pos = inside(g)
	vipTick(g, 0.5)
	if !allStanding(g) {
		t.Fatal("during the relief you can slip in")
	}
	g.p.pos = g.lvl.Def.Start
	vipTick(g, 8)
	if z.relief() {
		t.Error("the relief must end")
	}
	for _, gd := range vipStaff(g, vipDoor) {
		if dist(gd.pos, gd.vipPost) > 6 {
			t.Errorf("the new guard must be at the post (dist %.0f)", dist(gd.pos, gd.vipPost))
		}
	}
	g.p.pos = inside(g)
	vipTick(g, 0.5)
	if allStanding(g) {
		t.Error("after the relief the gate is guarded again")
	}
}

func TestVIPKillBonusPoliceAndGoal(t *testing.T) {
	g := vipGame(t, "PARQUE DE ATRACCIONES", false)
	boss := vipStaff(g, vipBoss)[0]
	d := vipByID(boss.vipID)
	g.p.pos = Vec{boss.pos.X + 8, boss.pos.Y}
	before, alive, total := g.points, g.aliveCount(), g.total
	g.killGuard(boss)
	if !boss.dead || g.points-before < d.bonus {
		t.Errorf("bonus missing: +%d, want >= %d", g.points-before, d.bonus)
	}
	if !g.policeCalled || g.stars < 2 || !vipEliminated[d.id] || !g.playerIsSuspect() {
		t.Errorf("police=%v stars=%d eliminated=%v suspect=%v", g.policeCalled, g.stars, vipEliminated[d.id], g.playerIsSuspect())
	}
	if g.aliveCount() != alive || g.total != total {
		t.Error("the VIP does not count for the goal")
	}
	if !vipSnapshot()[d.id] {
		t.Error("snapshot must export the mark")
	}
	// with the suit nobody cuffs him, but the police still come
	g2 := vipGame(t, "PARQUE DE ATRACCIONES", false)
	g2.p.suitT = 5
	b2 := vipStaff(g2, vipBoss)[0]
	g2.p.pos = Vec{b2.pos.X + 8, b2.pos.Y}
	g2.killGuard(b2)
	if !g2.policeCalled || g2.playerIsSuspect() {
		t.Errorf("suit: police=%v suspect=%v", g2.policeCalled, g2.playerIsSuspect())
	}
}

func TestVIPBehaviours(t *testing.T) {
	// tough: several blows
	g := vipGame(t, "TORRE EIFFEL", false)
	boss := vipStaff(g, vipBoss)[0]
	boss.vipID = "boxer"
	boss.vipHP = vipByID("boxer").hp
	g.p.pos = Vec{boss.pos.X + 8, boss.pos.Y}
	hits := 0
	for !boss.dead && hits < 10 {
		g.killGuard(boss)
		hits++
	}
	if hits != vipByID("boxer").hp {
		t.Errorf("tough VIP died after %d blows, want %d", hits, vipByID("boxer").hp)
	}
	// flee: runs away from the intruder
	g = vipGame(t, "FESTIVAL", false)
	boss = vipStaff(g, vipBoss)[0]
	start := boss.pos
	g.p.pos = Vec{boss.pos.X - 18, boss.pos.Y}
	if !g.lvl.vip.trig.Has(g.p.pos.X, g.p.pos.Y) {
		t.Fatal("setup: intruder must be in the zone")
	}
	vipTick(g, 0.4)
	if boss.pos.X <= start.X {
		t.Errorf("the VIP must run away from the intruder (%v -> %v)", start, boss.pos)
	}
	// call: the police come when someone breaks in
	g = vipGame(t, "PLAZA DE TOROS", false)
	vipStaff(g, vipBoss)[0].vipID = "yakuza"
	g.p.pos = inside(g)
	vipTick(g, 0.5)
	if !g.policeCalled || g.stars < 2 {
		t.Error("a VIP who calls for help brings the police and reinforcements")
	}
	// tip: help instead of killing
	g = vipGame(t, "TORRE EIFFEL", false)
	boss = vipStaff(g, vipBoss)[0]
	g.p.suitT = 10
	g.p.pos = Vec{boss.pos.X + 10, boss.pos.Y}
	before := g.points
	if !g.vipInteract(g.p) || g.points-before != 300 || g.vipInteract(g.p) {
		t.Errorf("a tip once: +%d", g.points-before)
	}
}

func TestVIPLoadDrawAndTwoPlayers(t *testing.T) {
	vipForce = true
	t.Cleanup(func() { vipForce = false })
	for _, two := range []bool{false, true} {
		g := layerGame(t, two)
		idx := -1
		for i, d := range levelDefs {
			if d.Title == "PLAZA DE TOROS" {
				idx = i
			}
		}
		if idx < 0 {
			t.Fatal("no CASINO")
		}
		g.loadLevel(idx)
		g.lvl.graded = true
		g.titleT = 0
		g.mode = ModePlay
		z := g.lvl.vip
		if z == nil || !z.spawned || g.vipStaffN() != 4 {
			t.Fatalf("two=%v: zone not spawned", two)
		}
		cards := 0
		for _, gu := range g.guests {
			if gu.vipCard {
				cards++
			}
		}
		if cards != vipCarriers {
			t.Errorf("%d invitation carriers, want %d", cards, vipCarriers)
		}
		if g.total != len(g.guests)+len(g.guards)-4 {
			t.Error("total must leave the staff out")
		}
		for _, pl := range g.players() {
			pl.pos = Vec{z.gx, z.in.Y + z.in.H - 10}
		}
		g.p.suitT, g.p.invite = 5, true
		img := ebiten.NewImage(screenW, screenH)
		for i := 0; i < 90; i++ {
			g.time += 1.0 / 60
			g.updateVIP(1.0 / 60)
			g.updateSuit(1.0 / 60)
			if i%30 == 0 {
				g.Draw(img)
			}
		}
		img.Deallocate()
	}
}
