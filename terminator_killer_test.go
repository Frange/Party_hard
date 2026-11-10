package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The endoskeleton is one of the killers of the catalogue: a card in ASESINOS (weapon line "Rifle"), a built-in
// preset with a rifle, and exported pictures in special_enemies with the rifle as the held object.
func TestTermRegistered(t *testing.T) {
	if !killerInCatalog(termSlug) || !killerAllowed(termSlug) || !killerPresetNamed(termSlug) {
		t.Fatal("el_exterminador is not a catalogue killer")
	}
	found := false
	for _, c := range killerCards() {
		if c.slug == termSlug {
			found = true
			if c.roleES != "Rifle" || c.nameES != "El Exterminador" {
				t.Errorf("card %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("no ASESINOS card")
	}
	if sp := killerSpecOf(termSlug); sp.Weapon != HoldRifle || sp.Name != termSlug {
		t.Errorf("spec weapon %d name %q", sp.Weapon, sp.Name)
	}
	if slasherActorName(&Slasher{name: termSlug}) != "El Exterminador" {
		t.Errorf("news name %q", slasherActorName(&Slasher{name: termSlug}))
	}
	if tr("El Exterminador") == "El Exterminador" && langDataEN["El Exterminador"] == "" {
		t.Error("no English name")
	}
	for _, k := range []string{termLine, "Rifle", "El Exterminador te ha disparado", "¡El Exterminador! Corre o escóndete."} {
		if langDataEN[k] == "" {
			t.Errorf("%q has no English", k)
		}
		if len([]rune(k)) > 48 {
			t.Errorf("%q is too long for a bubble", k)
		}
	}
	data, err := os.ReadFile(filepath.Join(spritesDir("assets", "special_enemies"), termSlug, "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	ca, err := loadCharAsset(filepath.Join(spritesDir("assets", "special_enemies"), termSlug))
	if err != nil || ca.Meta.Carry != "rifle" || ca.Meta.Role != "killer" {
		t.Fatalf("asset %v: %s", err, data)
	}
	// the rifle is drawn by the rig too (a held object of the animation, not painted in the pictures)
	rig := buildRig(ca)
	if rig.carry != HoldRifle {
		t.Errorf("rig carry %d", rig.carry)
	}
}

// He never comes to the secret island; elsewhere he may.
func TestTermNotOnTheIsland(t *testing.T) {
	g := &Game{}
	g.levelIdx = specialJurassic
	if g.termFits(termSlug) || !g.termFits("jason") {
		t.Fatal("island rule")
	}
	g.levelIdx = 3
	if !g.termFits(termSlug) {
		t.Fatal("he must be allowed in a party")
	}
}

// -killeras el_exterminador makes him come; he is tougher (termHP) and slower than the others.
func TestTermForcedComesToughAndSlow(t *testing.T) {
	g := killerGame(t, false)
	prev := dbgKillerAs
	t.Cleanup(func() { dbgKillerAs = prev })
	found := false
	for _, n := range mainFlagNames(t) {
		found = found || n == "killeras"
	}
	if !found {
		t.Fatal("-killeras flag missing")
	}
	dbgKillerAs, dbgKiller = termSlug, true
	g.killerLevelReset()
	g.time = killerForcedAt + 1
	g.updateSlashers(0.1)
	s := g.slasher
	if s == nil || s.name != termSlug || !isTerm(s) {
		t.Fatalf("forced killer: %+v", s)
	}
	if s.hp != termHP || s.hp <= 3 || s.chainsaw {
		t.Errorf("hp %d chainsaw %v", s.hp, s.chainsaw)
	}
	if v := termSpeed(s, 27); v >= 27 || v <= 0 {
		t.Errorf("speed %v", v)
	}
	if v := termSpeed(&Slasher{}, 27); v != 27 {
		t.Errorf("another killer changed speed: %v", v)
	}
}

// He aims at a guest in front of him and shoots him dead from afar; the shot needs the aiming time.
func TestTermShootsAGuest(t *testing.T) {
	g := killerGame(t, false)
	s := &Slasher{name: termSlug, state: slHunt, cs: g.as.playerCS}
	g.termSpawn(s)
	g.slasher = s
	var gu *Guest
	for _, o := range g.guests {
		if o.out() || o.perf != "" {
			continue
		}
		for _, dx := range []float64{50, -50} {
			at := Vec{o.pos.X + dx, o.pos.Y}
			if g.lvl.Free(at) && g.lvl.LOS(at, o.pos) {
				s.pos, gu = at, o
				s.dir = DirLeft
				if dx < 0 {
					s.dir = DirRight
				}
				break
			}
		}
		if gu != nil {
			break
		}
	}
	if gu == nil {
		t.Skip("no free spot beside a guest in this level")
	}
	s.tgtG = gu
	s.term.reload = 0
	if g.termShoot(s, termAim/2) || gu.out() {
		t.Fatal("he shot before aiming")
	}
	shot := false
	for i := 0; i < 20 && !shot; i++ {
		shot = g.termShoot(s, 0.1)
	}
	if !shot || !gu.out() || s.kills != 1 {
		t.Fatalf("shot=%v dead=%v kills=%d", shot, gu.out(), s.kills)
	}
	if s.term.reload <= 0 || g.termShoot(s, 0.1) {
		t.Error("no reload after a shot")
	}
}

// Killing him takes termHP stabs, counts as a killer beaten and unlocks his card.
func TestTermDiesAndCounts(t *testing.T) {
	g := killerGame(t, false)
	s := &Slasher{name: termSlug, state: slHunt, cs: g.as.playerCS, pos: g.p.pos}
	g.termSpawn(s)
	g.slasher = s
	before := g.progress.Stats["killers"]
	for i := 0; i < termHP; i++ {
		if s.state == slGone {
			t.Fatalf("down after %d stabs", i)
		}
		s.pause = 0
		g.hitSlasher()
	}
	if s.state != slGone {
		t.Fatal("still standing after termHP stabs")
	}
	if g.progress.Album["killer:"+termSlug]&albumKilled == 0 {
		t.Error("card not unlocked")
	}
	if g.progress.Stats["killers"] != before+1 {
		t.Errorf("killers beaten %d -> %d", before, g.progress.Stats["killers"])
	}
}
