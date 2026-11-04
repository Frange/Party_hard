package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// the gas trap: each player breathes his own share, so the one standing in the cloud dies and
// the one who walked away does not
func TestGasHitsEachPlayerOnHisOwn(t *testing.T) {
	g := twoGame(true)
	c := Vec{300, 300}
	g.p.pos = Vec{600, 300}  // player 1 far from the cloud
	g.p2.pos = Vec{310, 300} // player 2 inside it
	g.gas = []gasCloud{{pos: c, t: 9}}
	for f := 0; f < 60*5; f++ {
		g.time += 1.0 / 60
		g.updateTraps2(1.0 / 60)
	}
	if g.p.dead {
		t.Error("player 1 was not in the gas and must live")
	}
	if !g.p2.dead {
		t.Error("player 2 stayed 5 s in the gas and must be dead")
	}
	// and the other way round: both in the cloud, both die, each after his own 3 s
	g = twoGame(true)
	g.p.pos, g.p2.pos = Vec{300, 300}, Vec{305, 300}
	g.gas = []gasCloud{{pos: c, t: 9}}
	for f := 0; f < 60*4; f++ {
		g.updateTraps2(1.0 / 60)
	}
	if !g.p.dead || !g.p2.dead {
		t.Errorf("both breathed the gas for 4 s: p1 dead=%v p2 dead=%v", g.p.dead, g.p2.dead)
	}
}

// one player: the old exposure counter still decides (3 s)
func TestGasOnePlayerUnchanged(t *testing.T) {
	g := twoGame(false)
	g.p.pos = Vec{300, 300}
	g.gas = []gasCloud{{pos: Vec{300, 300}, t: 9}}
	for f := 0; f < 60*2; f++ {
		g.updateTraps2(1.0 / 60)
	}
	if g.p.dead {
		t.Fatal("2 s of gas must not kill yet")
	}
	for f := 0; f < 60*2; f++ {
		g.updateTraps2(1.0 / 60)
	}
	if !g.p.dead {
		t.Fatal("4 s of gas must kill")
	}
}

// the disguise: with two players the police only lose the trail when nobody is left that they
// could still be after
func TestDisguiseByPlayer(t *testing.T) {
	g := twoGame(true)
	g.wanted = 60
	g.markSuspect(g.p) // the cuffs are personal with two players (twojail.go)
	g.markSuspect(g.p2)
	g.p.item, g.p2.item = "disguise", "disguise"
	if !g.playerWantedNow(g.p) || !g.playerWantedNow(g.p2) {
		t.Fatal("both are wanted while the search is on")
	}
	g.withPlayer(g.p, g.useDisguise)
	if g.p.item != "" || !g.p.disguised {
		t.Fatal("player 1 must have changed his look")
	}
	if g.wanted <= 0 {
		t.Fatal("player 2 has not changed yet: the search goes on")
	}
	if g.playerWantedNow(g.p) {
		t.Fatal("a disguised player is not the wanted one any more")
	}
	g.withPlayer(g.p2, g.useDisguise)
	if g.wanted != 0 {
		t.Fatal("both changed: the police lose the trail")
	}
	if g.p.disguised || g.p2.disguised {
		t.Fatal("a closed case must clear the disguised flags")
	}
	// one player: as always, the disguise closes the case at once
	g = twoGame(false)
	g.wanted = 60
	g.p.item = "disguise"
	g.useDisguise()
	if g.wanted != 0 {
		t.Fatal("one player: the disguise closes the case")
	}
}

// the bubble manager keeps clear of every player, not only the first
func TestBubblesAvoidBothPlayers(t *testing.T) {
	m := &bubMgr{}
	req := func(key string, cx float64) bubReq {
		return bubReq{key: key, pri: bubGuest, score: 1000, cx: cx, y: 200, w: 60, h: 13, lines: []string{"Hola " + key}}
	}
	first := bubRect{20, 190, 40, 215} // player 1 under the first bubble
	m.extra = []bubRect{{120, 190, 140, 215}}
	var got []bubReq
	for f := 0; f < 40; f++ {
		got = m.pick([]bubReq{req("a", 30), req("b", 130)}, float64(f)/60, &first, 960)
	}
	for _, r := range got {
		x := math.Min(math.Max(r.cx-math.Round(r.w/2), 2), 960-r.w-2)
		box := bubRect{x, r.y - r.shift, x + r.w, r.y - r.shift + r.h}
		if box.hit(first) || box.hit(m.extra[0]) {
			t.Errorf("bubble %q covers a player: %+v", r.key, box)
		}
	}
	if len(got) == 0 {
		t.Fatal("the bubbles should be shown moved up, not all dropped")
	}
}

// the rare-event key and the legend powers are for the whole party: a press from the second
// player's pad fires them just like one from the first
func TestPartyWideActionsEitherPlayer(t *testing.T) {
	playersTestSetup(t)
	fakePads(2)
	oldJust := padJustFn
	defer func() { padJustFn = oldJust }()
	twoInput = true
	playerDev = [2]int{0, 1}
	b2 := binds2[actEvent].Pad
	padJustFn = func(id ebiten.GamepadID, b ebiten.StandardGamepadButton) bool { return id == 11 && b == b2 }
	if actEvent.justFor(0) {
		t.Fatal("player 1's pad did not press anything")
	}
	if !actEvent.justEither() {
		t.Fatal("player 2 pressed the event key: it must fire")
	}
	twoInput = false
	if actEvent.justEither() != actEvent.just() {
		t.Fatal("one player: justEither is just()")
	}
}

// the use mark says the key of each player (his own binding)
func TestActionHintPerPlayer(t *testing.T) {
	playersTestSetup(t)
	twoInput = true
	playerDev = [2]int{-1, -1}
	h1, h2 := actionHintFor(actInteract, 0), actionHintFor(actInteract, 1)
	if h1 == h2 {
		t.Fatalf("both players show the same key %q", h1)
	}
	if h1 != actionHint(actInteract) {
		t.Fatalf("player 1 hint %q != %q", h1, actionHint(actInteract))
	}
	twoInput = false
	if actionHintFor(actInteract, 1) != actionHint(actInteract) {
		t.Fatal("one player: the plain hint")
	}
}

// each player annoys the dancers on his own: player 2 dancing does not count for player 1
func TestDanceCountsPerPlayer(t *testing.T) {
	g := twoGame(true)
	g.p.pos, g.p2.pos = Vec{300, 300}, Vec{302, 300}
	for i := 0; i < 3; i++ {
		g.withPlayer(g.p, g.danceRoll)
		g.withPlayer(g.p2, g.danceRoll)
	}
	if g.p.danceCount != 3 || g.p2.danceCount != 3 {
		t.Fatalf("each player has danced 3 times: %d %d", g.p.danceCount, g.p2.danceCount)
	}
}

// two players using the things of the party at the same time: each hides in his own bush,
// picks up his own item and cuts his own phone, without touching the other one's state
func TestTwoPlayersUseThingsAtOnce(t *testing.T) {
	g := twoGame(true)
	mk := func(name string, x, y float64) *Prop {
		k := propKinds[name]
		return &Prop{Name: name, K: k, X: x, Y: y, Foot: Rect{x - float64(k.FW)/2, y - float64(k.FH), float64(k.FW), float64(k.FH)}}
	}
	b1, b2 := mk("bush", 200, 300), mk("bush", 260, 300)
	phone := mk("phone", 330, 300)
	bottle := mk("poisonbottle", 400, 300)
	g.lvl.Props = append(g.lvl.Props, b1, b2, phone, bottle)
	g.p.pos, g.p2.pos = Vec{200, 301}, Vec{260, 301}
	g.interact() // player 1 hides
	g.asPlayer(1, g.interact)
	if !g.p.hidden || !g.p2.hidden || g.p.hideProp != b1 || g.p2.hideProp != b2 {
		t.Fatalf("each one hides in his own bush: %v %v", g.p.hidden, g.p2.hidden)
	}
	g.asPlayer(1, g.interact) // player 2 comes out: player 1 stays hidden
	if g.p2.hidden || !g.p.hidden {
		t.Fatal("player 2 left his bush and player 1 must stay inside his")
	}
	g.interact()
	if g.p.hidden {
		t.Fatal("player 1 leaves his bush")
	}
	// the phone: player 2 cuts it, player 1 is not credited with anything
	g.p.pos, g.p2.pos = Vec{600, 300}, Vec{330, 301}
	g.asPlayer(1, g.interact)
	if !phone.used {
		t.Fatal("player 2 must have cut the cable")
	}
	// the item: player 1 takes the poison, player 2 has none
	g.p.pos, g.p2.pos = Vec{400, 301}, Vec{600, 300}
	g.interact()
	if g.p.item != "poison" || g.p2.item != "" {
		t.Fatalf("the poison is player 1's: %q %q", g.p.item, g.p2.item)
	}
}

// both players in the pool: each one swims on his own (slower in the water, nothing shared)
func TestTwoPlayersInThePool(t *testing.T) {
	g := twoGame(true)
	g.lvl = buildLevel(level2())
	g.lvl.Def.Swim = true
	var in Vec
	for _, r := range g.lvl.Def.Pools {
		in = Vec{r.X + r.W/2, r.Y + r.H/2}
		break
	}
	if in == (Vec{}) {
		t.Skip("no pool")
	}
	g.p.pos, g.p2.pos = in, Vec{in.X + 14, in.Y}
	if !g.lvl.InPool(g.p.pos) || !g.lvl.InPool(g.p2.pos) {
		t.Fatal("both are inside the pool")
	}
	a, b := g.p.pos, g.p2.pos
	g.p.moving, g.p2.moving = false, false
	g.updatePlayer(1.0 / 60)
	g.asPlayer(1, func() { g.updatePlayer(1.0 / 60) })
	if g.p.dead || g.p2.dead || dist(g.p.pos, a) > 3 || dist(g.p2.pos, b) > 3 {
		t.Fatal("two players standing in the water must not move or die")
	}
}
