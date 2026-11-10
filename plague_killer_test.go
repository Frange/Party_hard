package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// PLAGUE_SHEET=dir go test -run TestPlagueSheet writes an enlarged sheet: idle / walk (4 frames) / stab in the 3 views.
func TestPlagueSheet(t *testing.T) {
	dir := os.Getenv("PLAGUE_SHEET")
	if dir == "" {
		t.Skip("set PLAGUE_SHEET to a directory")
	}
	os.MkdirAll(dir, 0o755)
	const z = 6
	poses := []Pose{ownPose(AnimIdle, 0)}
	for f := 0; f < animFrameCount(AnimWalk) && f < 4; f++ {
		poses = append(poses, ownPose(AnimWalk, f))
	}
	poses = append(poses, ownPose(AnimStab, 1))
	sp := killerSpecOf(plagueSlug)
	cellW, cellH := charW*z+8, charH*z+8
	big := image.NewRGBA(image.Rect(0, 0, len(poses)*cellW, 3*cellH))
	for i := 0; i < len(big.Pix); i += 4 {
		big.Pix[i], big.Pix[i+1], big.Pix[i+2], big.Pix[i+3] = 0x8a, 0x9a, 0x7a, 255
	}
	for pi, p := range poses {
		for vi, vw := range []int{vFront, vSide, vBack} {
			b := renderChar(&sp, vw, p)
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					i := (y*b.w + x) * 4
					if b.p[i+3] == 0 {
						continue
					}
					c := color.RGBA{b.p[i], b.p[i+1], b.p[i+2], 255}
					for dy := 0; dy < z; dy++ {
						for dx := 0; dx < z; dx++ {
							big.SetRGBA(pi*cellW+x*z+dx, vi*cellH+y*z+dy, c)
						}
					}
				}
			}
		}
	}
	f, err := os.Create(filepath.Join(dir, "plague"+os.Getenv("PLAGUE_TAG")+".png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, big)
	f.Close()
}

// He is a registered killer: a preset, a card of the ASESINOS tab with ES/EN names, his texts translated.
func TestPlagueRegistered(t *testing.T) {
	if !killerPresetNamed(plagueSlug) || !killerInCatalog(plagueSlug) || !killerAllowed(plagueSlug) {
		t.Fatal("the beak doctor is not a registered, allowed killer")
	}
	var info killerInfo
	for _, k := range killerCatalog {
		if k.slug == plagueSlug {
			info = k
		}
	}
	if info.es != "El Doctor Pico" || info.en != "The Beak Doctor" {
		t.Errorf("names %q / %q", info.es, info.en)
	}
	found := false
	for _, c := range albumCards() {
		if c.key == "killer:"+plagueSlug {
			found = true
			if c.kind() != cardKiller || c.roleES != "Bastón" {
				t.Errorf("card kind %d role %q", c.kind(), c.roleES)
			}
		}
	}
	if !found {
		t.Error("no card in ASESINOS")
	}
	if n := len(killerCards()); n != len(killerCatalog)+len(killerTrophy) {
		t.Errorf("%d killer cards for %d + %d", n, len(killerCatalog), len(killerTrophy))
	}
	for _, es := range []string{plagueLine, "Bastón", "Cof, cof...", "El Doctor Pico te ha recetado", "¡El Doctor Pico! No dejes que te toque."} {
		en, ok := langDataEN[es]
		if !ok || en == "" {
			t.Errorf("%q has no English", es)
		}
		if len(es) > 48 || len(en) > 48 {
			t.Errorf("%q / %q longer than 48 bytes", es, en)
		}
	}
	sp := killerSpecOf(plagueSlug)
	if sp.Name != plagueSlug || sp.Top != TopRobe {
		t.Errorf("spec %q top %d", sp.Name, sp.Top)
	}
}

// He draws in the three views and the usual animations inside the sprite box, and his layers really show.
func TestPlagueDrawsInsideTheSprite(t *testing.T) {
	sp := killerSpecOf(plagueSlug)
	twin := sp
	twin.Name = "twin"
	has := func(b *pixBuf, c color.RGBA) bool {
		for i := 0; i+3 < len(b.p); i += 4 {
			if b.p[i] == c.R && b.p[i+1] == c.G && b.p[i+2] == c.B && b.p[i+3] != 0 {
				return true
			}
		}
		return false
	}
	anims := []int{AnimIdle, AnimWalk, AnimRun, AnimStab, AnimDance3, AnimPanic}
	for _, vw := range []int{vFront, vSide, vBack} {
		for _, a := range anims {
			for f := 0; f < animFrameCount(a); f++ {
				n, edge := charimpCount(renderChar(&sp, vw, ownPose(a, f)))
				tn, tEdge := charimpCount(renderChar(&twin, vw, ownPose(a, f)))
				if edge > tEdge {
					t.Errorf("view %d anim %d frame %d: %d pixels on the edge (without layers: %d)", vw, a, f, edge, tEdge)
				}
				if n < 60 || n < tn*9/10 {
					t.Errorf("view %d anim %d frame %d: only %d pixels (without layers: %d)", vw, a, f, n, tn)
				}
			}
		}
		b := renderChar(&sp, vw, ownPose(AnimIdle, 0))
		if string(b.p) == string(renderChar(&twin, vw, ownPose(AnimIdle, 0)).p) {
			t.Errorf("view %d: the own layers are not drawn", vw)
		}
		want := []color.RGBA{pdHat, pdHood, pdBelt}
		if vw != vBack {
			want = append(want, pdBeak, pdLens)
		}
		for _, c := range want {
			if !has(b, c) {
				t.Errorf("view %d: colour %v missing", vw, c)
			}
		}
	}
	if !has(renderChar(&sp, vFront, ownPose(AnimIdle, 0)), pdCross) {
		t.Error("no cross on the chest")
	}
}

// Walking changes the picture from frame to frame in every view.
func TestPlagueWalkFramesChange(t *testing.T) {
	sp := killerSpecOf(plagueSlug)
	for _, vw := range []int{vFront, vSide, vBack} {
		seen := map[string]bool{}
		for f := 0; f < animFrameCount(AnimWalk); f++ {
			seen[string(renderChar(&sp, vw, ownPose(AnimWalk, f)).p)] = true
		}
		if len(seen) < 3 {
			t.Errorf("view %d: only %d different walking frames", vw, len(seen))
		}
	}
}

// -killer -killeras doctor_pico sends him; he is slower and tougher; a guest he reaches falls sick, coughs
// and dies a few seconds later (the doctor's kill, not the player's); he never comes to the island.
func TestPlagueComesAndMakesSick(t *testing.T) {
	g := killerGame(t, false)
	prev := dbgKillerAs
	t.Cleanup(func() { dbgKillerAs = prev })
	dbgKillerAs = plagueSlug
	if _, _, name := g.pickKiller(); name != plagueSlug {
		t.Fatalf("-killeras %s picked %s", plagueSlug, name)
	}
	if !g.spawnSlasher() {
		t.Fatal("no killer spawned")
	}
	s := g.slasher
	if !isPlague(s) || s.hp != plagueHP || s.name != plagueSlug {
		t.Fatalf("spawned %q hp %d plague %v", s.name, s.hp, s.plague != nil)
	}
	if plagueSpeed(s, 30) >= 30 || plagueSpeed(&Slasher{}, 30) != 30 {
		t.Error("he must walk slower than the others (only him)")
	}
	// put him next to a guest, in the hunt
	var v *Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" {
			v = gu
			break
		}
	}
	if v == nil {
		t.Fatal("no guest")
	}
	s.state, s.pause, s.alert = slHunt, 0, false
	s.pos = Vec{v.pos.X + 6, v.pos.Y}
	s.tgtG, s.tgtP, s.tgtD = v, false, nil
	kills0 := g.kills
	if !g.plagueTouch(s) {
		t.Fatal("the touch did not happen")
	}
	if _, ok := s.plague.sick[v]; !ok || v.out() {
		t.Fatal("the guest is not sick (or died at once)")
	}
	for i := 0; i < 30; i++ { // 1.5 s: still alive, coughing
		g.plagueTick(s, 0.05)
	}
	if v.out() || v.state != gVomit || v.icon != "Cof, cof..." {
		t.Fatalf("after 1.5 s: out %v state %d icon %q", v.out(), v.state, v.icon)
	}
	for i := 0; i < 50; i++ {
		g.plagueTick(s, 0.05)
	}
	if !v.out() {
		t.Fatal("the sick guest never fell")
	}
	if s.kills < 1 || g.kills != kills0 {
		t.Errorf("killer kills %d, player kills %d -> %d (must not count for the player)", s.kills, kills0, g.kills)
	}
	if len(s.plague.sick) != 0 {
		t.Error("the dead guest is still on the sick list")
	}
	// the island never gets him; other parties do
	lv := g.levelIdx
	g.levelIdx = specialJurassic
	if g.plagueFits(plagueSlug) || !g.plagueFits("jason") {
		t.Error("plagueFits on the island")
	}
	g.levelIdx = lv
	if !g.plagueFits(plagueSlug) {
		t.Error("he must fit an ordinary party")
	}
}
