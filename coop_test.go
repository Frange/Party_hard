package main

import (
	"testing"
)

// two players on the entrance, ready for the cooperative gags
func coopGame(t *testing.T) *Game {
	t.Helper()
	g := twoGame(true)
	g.coopEnsure()
	g.time = 100
	g.p.pos = Vec{200, 200}
	g.p2.pos = Vec{210, 200}
	return g
}

func press(g *Game, i int) bool {
	var r bool
	g.asPlayer(i, func() { r = g.coopInteract(g.p) })
	return r
}

// with one player nothing of this file does anything
func TestCoopOnePlayerIsUntouched(t *testing.T) {
	g := twoGame(false)
	ran := 0
	g.coopTrack(0, func() { ran++ })
	if ran != 1 {
		t.Fatal("coopTrack must just run the function with one player")
	}
	g.coopUpdate(0.1)
	if g.coopInteract(g.p) || g.coopBlockStab(g.p) || g.coopHidden(g.p) || g.coopSpeedMul(g.p) != 1 {
		t.Fatal("one player: no cooperative behaviour")
	}
}

func TestCoopCarryPartner(t *testing.T) {
	g := coopGame(t)
	p1, p2 := g.p, g.p2
	// a partner on his feet is not picked up
	if press(g, 0) {
		t.Fatal("a standing partner cannot be carried")
	}
	g.coop.interAt, g.coop.handUp = [2]float64{}, [2]float64{}
	p2.knockT = 3 // knocked down
	if !press(g, 0) || g.coop.carry[0] != p2 {
		t.Fatal("E next to a knocked partner picks him up")
	}
	if p2.knockT > 0.85 {
		t.Fatalf("a carried partner wakes up quickly, knockT=%.2f", p2.knockT)
	}
	if !g.coopIsCarried(p2) || !g.coopHidden(p2) || !g.coopCarrying(p1) {
		t.Fatal("carry flags")
	}
	if g.coopSpeedMul(p1) >= 1 {
		t.Fatal("carrying slows him down")
	}
	if !g.coopBlockStab(p1) {
		t.Fatal("no stabbing while carrying the partner")
	}
	// he follows the carrier, and does not stand up on his own
	p1.pos = Vec{260, 240}
	for i := 0; i < 80; i++ {
		g.coopUpdate(0.05)
		p2.knockT -= 0.05 // his own update would do this
	}
	if dist(p2.pos, p1.pos) > 0.01 || p2.knockT <= 0 {
		t.Fatal("the carried partner stays on the shoulders")
	}
	// E puts him down: he stands up
	g.time += 5
	if !press(g, 0) || g.coop.carry[0] != nil {
		t.Fatal("E puts the partner down")
	}
	if p2.knockT != 0 || p2.dead {
		t.Fatal("after the drop he stands up alive")
	}
	if g.coop.score[0].helps != 1 {
		t.Fatalf("a carry counts as a help: %d", g.coop.score[0].helps)
	}
}

func TestCoopCarryDeadPartnerStaysDead(t *testing.T) {
	g := coopGame(t)
	g.p2.dead = true
	if !press(g, 0) || g.coop.carry[0] != g.p2 {
		t.Fatal("a dead partner can be carried")
	}
	g.coopDrop(0)
	if !g.p2.dead {
		t.Fatal("carrying never revives (the ambulance does that)")
	}
	if g.coop.score[0].helps != 0 {
		t.Fatal("dropping a body is not a help")
	}
}

func TestCoopCarrierHitDropsPartner(t *testing.T) {
	g := coopGame(t)
	g.p2.knockT = 3
	press(g, 0)
	g.p.knockT = 3 // the carrier is punched
	g.coopUpdate(0.05)
	if g.coop.carry[0] != nil {
		t.Fatal("a knocked-down carrier drops the partner")
	}
}

func TestCoopHighFive(t *testing.T) {
	g := coopGame(t)
	before := g.points
	if press(g, 0) {
		t.Fatal("the first press only raises the hand")
	}
	if g.coop.handUp[0] <= g.time {
		t.Fatal("the hand stays up waiting")
	}
	g.time += 0.4
	if !press(g, 1) {
		t.Fatal("the second press within a second makes the high five")
	}
	if g.points-before != fivePts {
		t.Fatalf("a high five is worth %d points, got %d", fivePts, g.points-before)
	}
	if g.coop.score[0].fives != 1 || g.coop.score[1].fives != 1 {
		t.Fatal("both count the high five")
	}
	// cooldown: no second one for 20 s
	g.time += 1
	press(g, 0)
	g.time += 0.3
	if press(g, 1) || g.points-before != fivePts {
		t.Fatal("no second high five during the cooldown")
	}
	g.time += fiveCool
	g.coop.handUp = [2]float64{}
	press(g, 0)
	g.time += 0.3
	if !press(g, 1) {
		t.Fatal("after the cooldown they can high five again")
	}
	// too late or too far
	g.time += fiveCool + 5
	g.coop.handUp = [2]float64{}
	press(g, 0)
	g.time += 2
	if press(g, 1) {
		t.Fatal("a hand that has been down for 2 s gets no high five")
	}
	g.time += 30
	g.p2.pos = Vec{400, 200}
	g.coop.handUp = [2]float64{}
	press(g, 0)
	g.time += 0.2
	if press(g, 1) {
		t.Fatal("too far for a high five")
	}
}

func TestCoopDanceDuo(t *testing.T) {
	g := coopGame(t)
	p1, p2 := g.p, g.p2
	p1.style, p2.style = 1, 3
	start := g.points
	for i := 0; i < 60; i++ { // 3 s dancing together
		p1.danceT, p2.danceT = 4, 4
		g.time += 0.05
		g.coopUpdate(0.05)
	}
	if !g.coop.duo {
		t.Fatal("two dancers close together are a duet")
	}
	if p1.style != p2.style {
		t.Fatal("a duet dances the same steps")
	}
	gain := g.points - start
	if gain < 25 || gain > 35 {
		t.Fatalf("a duet earns about 10 points a second, got %d in 3 s", gain)
	}
	if len(g.coop.fx) == 0 {
		t.Log("no hearts yet (random), fine")
	}
	for i := 0; i < 400; i++ { // a very long duet is capped
		p1.danceT, p2.danceT = 4, 4
		g.time += 0.05
		g.coopUpdate(0.05)
	}
	if g.points-start > duoPtsMax+5 {
		t.Fatalf("the duet bonus is capped at %d, got %d", duoPtsMax, g.points-start)
	}
	// far apart: no duet, no points
	g2 := coopGame(t)
	g2.p2.pos = Vec{300, 200}
	s2 := g2.points
	for i := 0; i < 40; i++ {
		g2.p.danceT, g2.p2.danceT = 4, 4
		g2.time += 0.05
		g2.coopUpdate(0.05)
	}
	if g2.coop.duo || g2.points != s2 {
		t.Fatal("two dancers far apart are not a duet")
	}
}

func TestCoopDanceBet(t *testing.T) {
	g := coopGame(t)
	g.p2.pos = Vec{320, 200}
	for i := 0; i < 20; i++ {
		g.p.danceT, g.p2.danceT = 4, 4
		g.time += 0.05
		g.coopUpdate(0.05)
	}
	if !g.coop.betOn {
		t.Fatal("two dancers far apart bet on who lasts longer")
	}
	start := g.points
	g.p.danceT = 0 // player 1 gives up
	g.p2.danceT = 2
	g.time += 0.05
	g.coopUpdate(0.05)
	if g.points-start != 15 || g.coop.score[1].points != 15 {
		t.Fatalf("the one that lasts wins 15 points: %d", g.points-start)
	}
	if g.coop.betOn {
		t.Fatal("the bet is over")
	}
}

func TestCoopCrossedTrapsKillBoth(t *testing.T) {
	g := coopGame(t)
	k := &PropKind{Trap: "explode", H: 20}
	trap := &Prop{Name: "bomb", K: k, X: 205, Y: 200, Foot: Rect{200, 196, 10, 8}}
	g.lvl.Props = append(g.lvl.Props, trap)
	g.coopOwn(trap, g.p2)
	trap.armT = 0.5
	g.triggerTrap(trap)
	if !g.p.dead || !g.p2.dead {
		t.Fatalf("a trap hurts anybody: p1 dead=%v p2 dead=%v", g.p.dead, g.p2.dead)
	}
}

func TestCoopDistractCop(t *testing.T) {
	g := coopGame(t)
	c := &Cop{pos: Vec{225, 200}, state: cChase}
	g.cops = append(g.cops, c)
	// the cop is not chasing: nothing to distract
	c.state = cSweep
	if press(g, 0) {
		t.Fatal("only a chasing cop can be distracted")
	}
	c.state = cChase
	g.coop.interAt, g.coop.handUp = [2]float64{}, [2]float64{}
	if !press(g, 0) {
		t.Fatal("E next to a chasing cop distracts him")
	}
	if !g.coopCopDistracted(c, 0.1) {
		t.Fatal("the cop stands still")
	}
	g.time += distractFor + 0.5
	if g.coopCopDistracted(c, 0.1) {
		t.Fatal("after 3 s he goes on")
	}
	// the cooldown: 15 s
	g.time += 1
	if press(g, 0) {
		t.Fatal("no new distraction during the cooldown")
	}
	g.time += distractCD
	if !press(g, 0) {
		t.Fatal("after the cooldown it works again")
	}
	if g.coop.score[0].distracts != 2 {
		t.Fatalf("distractions are counted: %d", g.coop.score[0].distracts)
	}
	// too far
	g.time += 20
	c.pos = Vec{500, 500}
	delete(g.coop.distract, c)
	if press(g, 0) {
		t.Fatal("the officer is too far")
	}
}

func TestCoopStatsPerPlayer(t *testing.T) {
	g := coopGame(t)
	g.coopTrack(0, func() { g.points += 30; g.kills++ })
	g.coopTrack(1, func() { g.points += 50; g.kills += 2 })
	if g.coop.score[0].points != 30 || g.coop.score[0].kills != 1 || g.coop.score[1].points != 50 || g.coop.score[1].kills != 2 {
		t.Fatalf("score: %+v", g.coop.score)
	}
	// a punch to the partner counts as a hit for the one who gave it
	g.coopTrack(0, func() { g.hitPlayer(g.p, g.p2) })
	if g.coop.score[0].hits != 1 || g.coop.score[1].hits != 0 {
		t.Fatalf("hits: %+v", g.coop.score)
	}
	// time on the floor
	for i := 0; i < 20; i++ {
		g.coopUpdate(0.1)
	}
	if g.coop.score[1].floorT < 1.5 {
		t.Fatalf("the knocked-down partner's floor time: %.2f", g.coop.score[1].floorT)
	}
	if g.coop.score[0].floorT != 0 {
		t.Fatal("the one on his feet has no floor time")
	}
}

func TestCoopTitlesAreDifferent(t *testing.T) {
	a := coopStats{points: 500, kills: 12, hits: 0, danceT: 2}
	b := coopStats{points: 200, kills: 3, hits: 4, floorT: 12, danceT: 30}
	ti := coopTitleFor(a, b)
	if ti[0] == "" || ti[1] == "" || ti[0] == ti[1] {
		t.Fatalf("each player has his own title: %v", ti)
	}
	if ti[0] != "Carnicero con delicadeza" && ti[0] != "Anfitrión del año" {
		t.Errorf("the one with more kills and points: %q", ti[0])
	}
	// nothing at all: still two different titles
	z := coopTitleFor(coopStats{}, coopStats{})
	if z[0] == "" || z[1] == "" || z[0] == z[1] {
		t.Fatalf("empty party titles: %v", z)
	}
	// symmetry: swapping the players swaps the titles
	sw := coopTitleFor(b, a)
	if sw[0] != ti[1] || sw[1] != ti[0] {
		t.Fatalf("titles must follow the player: %v vs %v", ti, sw)
	}
	// the helper gets the guardian angel
	h := coopTitleFor(coopStats{helps: 3, points: 100, kills: 1}, coopStats{points: 100, kills: 1})
	if h[0] != "Ángel de la guarda" {
		t.Errorf("the helper: %q", h[0])
	}
}

func TestCoopGagsHaveCooldowns(t *testing.T) {
	g := coopGame(t)
	// twins
	g.p.cs = g.p2.cs
	g.coopUpdate(0.1)
	if g.coop.twinNext <= g.time {
		t.Fatal("the twins gag goes on cooldown")
	}
	first := g.p.msg
	g.p.msg, g.p.msgT = "", 0
	g.time += 5
	g.coopUpdate(0.1)
	if g.p.msg == first && first != "" {
		t.Fatal("twins must not repeat within 90 s")
	}
	// "that is mine": both next to the same item
	pr := &Prop{Name: "poisonbottle", K: &PropKind{Item: "poison", H: 10}, X: 205, Y: 200, Foot: Rect{203, 198, 4, 4}}
	g.lvl.Props = append(g.lvl.Props, pr)
	g.p.msg, g.p2.msg = "", ""
	g.time += 100
	g.coop.sayNext = [2]float64{}
	g.coopUpdate(0.1)
	if g.p.msg != "¡Eso es mío!" {
		t.Fatalf("both reach for the same item: %q", g.p.msg)
	}
	if g.coop.mineNext <= g.time {
		t.Fatal("the gag has a cooldown")
	}
	// the greeting after being apart
	g2 := coopGame(t)
	g2.p2.pos = Vec{600, 200}
	for i := 0; i < 200; i++ {
		g2.time += 0.1
		g2.coopUpdate(0.1)
	}
	g2.p2.pos = Vec{225, 200}
	g2.coopUpdate(0.1)
	if g2.coop.greetNext <= g2.time {
		t.Fatal("meeting again after a while is greeted")
	}
}

func TestCoopSayIsRateLimited(t *testing.T) {
	g := coopGame(t)
	g.coopSay(g.p, "uno")
	g.coopSay(g.p, "dos")
	if g.p.msg != "uno" {
		t.Fatalf("a player does not talk twice within 2.5 s: %q", g.p.msg)
	}
	g.coopSayNow(g.p, "tres") // an answer ignores the wait
	if g.p.msg != "tres" {
		t.Fatal("coopSayNow answers at once")
	}
}

// the texts of this file fit a short cry
func TestCoopPhrasesAreShort(t *testing.T) {
	g := coopGame(t)
	lists := [][]string{
		{"¡Chócala!", "¡Choca esos cinco!", "¡Aquí arriba!", "¡Equipazo!", "¡Toma ya!", "¡Así se hace!", "¡Los mejores!"},
		{"¿Qué hora es, agente?", "Oiga, agente...", "¿Sabe usted dónde...?", "¡Mire, un pájaro!", "¿Eh? ¿Las cuatro?", "Mmm... ¿Qué?", "¿Un pájaro?", "Un momento..."},
		{"¡Pesas!", "¡Ay, mi espalda!", "¡Arriba!", "Ya te tengo", "¡Y tú, huelas!", "¿Adónde me llevas?", "Qué vistas...", "¡Bájame!"},
		{"¡A bailar juntos!", "¡Un dúo!", "¡Vamos allá!", "¡Ese es mi ritmo!", "¡Dale!", "¡Tú sígueme!", "¡Qué dúo!", "¡Aplausos!", "¡Bis!"},
		{"¡Eh! ¡Tu trampa!", "¡Ups, mi trampa!", "¡Era para otro!", "¡Perdona!", "¡Eso es mío!", "¡Lo vi yo primero!", "¡Es mío!", "¡Gemelos!", "¡Mismo estilo!", "¡Qué elegancia!"},
		{"¡Colega!", "¡Tú por aquí!", "¿Dónde estabas?", "¡Por fin!", "¡Te buscaba!", "¡Hola!", "¡Gané la apuesta!", "¡Me dio un tirón!", "Tengo agujetas", "¡Yo bailo más!", "¡Mírame!", "¡Ni de broma!", "¡Qué va!"},
		{"¡Me las pagarás!", "¡Eso es trampa!", "¡Te vi!", "¡Ya verás!", "¡Gracias, colega!", "Me debes una", "Mi héroe", "¡Estoy bien!"},
	}
	_ = g
	for _, l := range lists {
		for _, s := range l {
			if n := len([]rune(s)); n > 24 {
				t.Errorf("%q has %d characters: too long for a cry over a head", s, n)
			}
		}
	}
	for _, c := range coopTitles {
		if len([]rune(c.name)) > 30 {
			t.Errorf("title %q is too long", c.name)
		}
	}
}
