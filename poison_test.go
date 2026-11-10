package main

import (
	"math/rand"
	"testing"
)

func poisonGame() *Game {
	g := testGame()
	g.rng = rand.New(rand.NewSource(7))
	g.snd = &Sounds{muted: true}
	g.lvl = buildLevel(level1())
	g.p = &Player{pos: Vec{200, 300}, stamina: 6}
	return g
}

func poisonGuest(g *Game, at Vec) *Guest {
	spec := genericSpec(g.rng)
	gu := &Guest{pos: at, dir: DirDown, cs: newCharSet(&spec), state: gIdle, name: "Test"}
	g.guests = append(g.guests, gu)
	g.total++
	return gu
}

// a bottle holds 5 doses: five guests can be offered a drink, then it is gone
func TestPoisonDoses(t *testing.T) {
	g := poisonGame()
	g.p.item, g.p.doses = "poison", poisonDoses
	for i := 0; i < 6; i++ {
		poisonGuest(g, Vec{g.p.pos.X + 10, g.p.pos.Y + 6})
	}
	for n := 1; n <= poisonDoses; n++ {
		gu := g.poisonTarget()
		if gu == nil {
			t.Fatalf("no target for dose %d", n)
		}
		g.offerPoison(gu)
		if !gu.offered || gu.state != gDrink {
			t.Fatalf("dose %d: the guest is not drinking", n)
		}
		if g.p.doses != poisonDoses-n {
			t.Fatalf("doses left %d, want %d", g.p.doses, poisonDoses-n)
		}
	}
	if g.p.item != "" || g.poisonTarget() != nil {
		t.Error("an empty bottle must disappear")
	}
	if dosesText(3) != "Veneno: 3 dosis" || dosesText(0) != "Se acabó el veneno" {
		t.Error("dose message")
	}
	// the punch takes two doses
	g.p.item, g.p.doses = "poison", poisonDoses
	g.useDoses(poisonBowlDoses)
	if g.p.doses != poisonDoses-poisonBowlDoses {
		t.Error("the punch bowl takes two doses")
	}
	g.useDoses(10)
	if g.p.item != "" {
		t.Error("the bottle empties when the doses run out")
	}
}

// dizzy, then sick, then dead: in that order, and nobody dies before the last stage
func TestPoisonStagesInOrder(t *testing.T) {
	g := poisonGame()
	gu := poisonGuest(g, Vec{300, 300})
	g.startPoison(gu)
	if gu.poison < poisonMin || gu.poison > poisonMin+poisonSpread {
		t.Fatalf("time to live out of range: %.1f", gu.poison)
	}
	if poisonStage(gu) != 1 {
		t.Fatal("it must start dizzy")
	}
	seen := map[int]bool{}
	last := 0
	vomits, died := 0, false
	for i := 0; i < 30*60 && !died; i++ {
		prev := gu.state
		died = g.updatePoison(gu, 1.0/30)
		if gu.state == gVomit && prev != gVomit {
			vomits++
		}
		if !died {
			if gu.dead {
				t.Fatal("dead before the end")
			}
			s := poisonStage(gu)
			if s < last {
				t.Fatalf("stage went back from %d to %d", last, s)
			}
			last = s
			seen[s] = true
		}
	}
	if !died || !gu.dead {
		t.Fatal("the guest never died")
	}
	if !seen[1] || !seen[2] || !seen[3] {
		t.Errorf("stages seen: %v", seen)
	}
	if vomits == 0 {
		t.Error("no vomiting before the end")
	}
	if g.progress.Stats["poison"] != 1 || g.progress.Stats["kills"] != 1 || g.kills != 1 {
		t.Errorf("the death must count: poison %d kills %d g.kills %d", g.progress.Stats["poison"], g.progress.Stats["kills"], g.kills)
	}
	if len(g.bodies) != 1 || g.bodies[0].byPlayer {
		t.Error("the body must be left, but not marked as the player's work (nobody knows who poisoned it)")
	}
}

// a poisoned guest does not go and tell the police about their own state
func TestPoisonedDoNotCall(t *testing.T) {
	g := poisonGame()
	gu := poisonGuest(g, Vec{300, 300})
	g.startPoison(gu)
	gu.state = gToPhone
	g.updatePoison(gu, 0.05)
	if gu.state == gToPhone || gu.state == gReport || gu.state == gCall {
		t.Error("a poisoned guest went to call the police")
	}
}

// the drink works from the offered glass, and from a poisoned bowl; both start the countdown
func TestPoisonedDrinkStartsCountdown(t *testing.T) {
	g := poisonGame()
	g.mode = ModeMenu // no scans, no police
	g.p.item, g.p.doses = "poison", poisonDoses
	gu := poisonGuest(g, Vec{g.p.pos.X + 10, g.p.pos.Y + 6})
	g.offerPoison(gu)
	for i := 0; i < 90 && gu.poison <= 0; i++ {
		g.updateGuest(gu, 1.0/30)
	}
	if gu.poison <= 0 || gu.offered {
		t.Fatal("the offered drink did not poison the guest")
	}
	if gu.dead {
		t.Fatal("and certainly not at once")
	}
	// a poisoned bowl
	var bowl *Prop
	for _, pr := range g.lvl.Props {
		if pr.K.Poison {
			bowl = pr
		}
	}
	if bowl == nil {
		t.Skip("level 1 has no bowl in this build")
	}
	bowl.armT = 40
	o := poisonGuest(g, Vec{bowl.Foot.X + bowl.Foot.W/2, bowl.Foot.Y + bowl.Foot.H/2})
	o.state, o.timer, o.dur = gDrink, 0.05, 1
	for i := 0; i < 5 && o.poison <= 0; i++ {
		g.updateGuest(o, 1.0/30)
	}
	if o.poison <= 0 {
		t.Error("the poisoned punch did not poison the drinker")
	}
}

// the sick look like it: retching while vomiting, and the dizzy lean on nothing when they stop
func TestPoisonedPoses(t *testing.T) {
	g := poisonGame()
	gu := poisonGuest(g, Vec{300, 300})
	g.startPoison(gu)
	if a, _ := g.guestPose(gu); a != AnimTired {
		t.Errorf("a dizzy guest standing still should look tired, got anim %d", a)
	}
	gu.state = gVomit
	if a, _ := g.guestPose(gu); a != AnimVomit {
		t.Errorf("vomiting should use the vomit animation, got %d", a)
	}
	gu.moving, gu.state = true, gGoto
	if poisonedStagger(gu, 1.3) == 0 && poisonedStagger(gu, 2.1) == 0 {
		t.Error("a dizzy guest on the move must stagger")
	}
	h := poisonGuest(g, Vec{320, 300})
	h.moving = true
	if poisonedStagger(h, 1.3) != 0 {
		t.Error("healthy guests do not stagger")
	}
}
