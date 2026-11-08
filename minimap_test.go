package main

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func mmRestore(t *testing.T) {
	t.Helper()
	on, kf, pf := minimapOn, mmKeyFn, mmPadFn
	t.Cleanup(func() { minimapOn, mmKeyFn, mmPadFn = on, kf, pf })
}

func mmAt(g *Game, v Vec) [4]uint8 {
	x, y := g.mm.pos(v)
	i := (y*mmW + x) * 4
	b := g.mm.buf
	return [4]uint8{b[i], b[i+1], b[i+2], b[i+3]}
}

// walls are white, every kind of character has its own colour, the panel is see-through (alpha ~0.45)
func TestMinimapColoursByKind(t *testing.T) {
	mmRestore(t)
	g := hudPlay(t, true)
	g.stealth = false
	g.loadLevel(0)
	g.mode = ModePlay
	g.guests, g.guards, g.cops, g.cams = nil, nil, nil, nil
	g.p2.pos = Vec{300, 250}
	g.first().pos = Vec{200, 250}
	// more guests than the last-guests marker covers (lastguests.go), so it is off whatever lastGuestsMax is
	g.guests = []*Guest{{pos: Vec{400, 250}}}
	for i := 0; i < lastGuestsMax+2; i++ {
		g.guests = append(g.guests, &Guest{pos: Vec{100 + float64(i)*6, 100}})
	}
	g.guards = []*Guard{{pos: Vec{500, 250}}, {pos: Vec{600, 250}}}
	g.guards[1].vip = vipBoss
	g.cops = []*Cop{{pos: Vec{700, 250}}}
	g.time = 0.1
	g.minimapFrame()
	if got := mmAt(g, Vec{400, 250}); got != mmGuest {
		t.Errorf("guest %v", got)
	}
	if got := mmAt(g, Vec{500, 250}); got != mmGuard {
		t.Errorf("guard %v", got)
	}
	if got := mmAt(g, Vec{600, 250}); got != mmVIP {
		t.Errorf("VIP %v", got)
	}
	if got := mmAt(g, Vec{700, 250}); got != mmCopA && got != mmCopB {
		t.Errorf("cop %v", got)
	}
	if got := mmAt(g, Vec{300, 250}); got != mmP2 {
		t.Errorf("player 2 %v", got)
	}
	if got := mmAt(g, Vec{200, 250}); got != mmP1 && got != mmP1b {
		t.Errorf("player 1 %v", got)
	}
	// every colour is different from every other kind
	kinds := [][4]uint8{mmGuest, mmGuard, mmVIP, mmCam, mmTrap, mmKiller, mmP2, mmP1, mmWall}
	for i := range kinds {
		for j := i + 1; j < len(kinds); j++ {
			if kinds[i] == kinds[j] {
				t.Errorf("two kinds share the colour %v", kinds[i])
			}
		}
	}
	// premultiplied: no channel above alpha (Ebitengine would draw it as light)
	for _, c := range append(kinds, mmBG, mmEdge, mmCone, mmNoise, mmCopA, mmCopB, mmP1b) {
		if c[0] > c[3] || c[1] > c[3] || c[2] > c[3] {
			t.Errorf("colour %v is not premultiplied", c)
		}
	}
	if mmBG[3] < 100 || mmBG[3] > 130 {
		t.Errorf("panel alpha %d is not ~0.45", mmBG[3])
	}
	// walls: some white pixels exist and match the solids
	white := 0
	for i := 0; i < mmW*mmH; i++ {
		if [4]uint8{g.mm.base[i*4], g.mm.base[i*4+1], g.mm.base[i*4+2], g.mm.base[i*4+3]} == mmWall {
			white++
		}
	}
	if white < 100 {
		t.Errorf("only %d white wall pixels", white)
	}
	// a solid prop of the level appears white at its centre
	checked := 0
	for _, r := range g.lvl.solidsP {
		if r.W < 14 || r.H < 14 || r.X < 0 || r.Y < 0 || r.X+r.W > g.lvl.W || r.Y+r.H > g.lvl.H {
			continue
		}
		if got := mmAt(g, Vec{r.X + r.W/2, r.Y + r.H/2}); got == mmWall {
			g.mm.buf = g.mm.base // read the base (nobody stands there)
			checked++
		}
		if checked >= 3 {
			break
		}
	}
	if checked == 0 {
		t.Error("no big solid of the level is white on the base")
	}
}

// stealth shows only walls and players; the blackout adds noise
func TestMinimapStealthAndBlackout(t *testing.T) {
	mmRestore(t)
	g := hudPlay(t, false)
	g.stealth = false
	g.loadLevel(0)
	g.mode = ModePlay
	// more guests than the "last guests" marker covers, so no blinking extra dots appear whatever lastGuestsMax is
	g.guests = nil
	for i := 0; i < lastGuestsMax+3; i++ {
		g.guests = append(g.guests, &Guest{pos: Vec{100 + float64(i)*6, 100}})
	}
	g.guards = []*Guard{{pos: Vec{500, 250}}}
	g.cops, g.cams = nil, nil
	g.first().pos = Vec{200, 250}
	g.stealth = true
	g.minimapFrame()
	g.stealth = false
	if g.mm.dots != 1 {
		t.Errorf("stealth: %d dots, only the player expected", g.mm.dots)
	}
	g.minimapFrame()
	if want := 1 + len(g.guests) + 1; g.mm.dots != want {
		t.Errorf("normal: %d dots, want %d", g.mm.dots, want)
	}
	prev := g.blk
	t.Cleanup(func() { g.blk = prev })
	g.blk.t = 5
	g.time = 0.3 // int(0.3*8)%3 = 2: characters visible
	g.minimapFrame()
	noise := 0
	for i := 0; i < mmW*mmH; i++ {
		if [4]uint8{g.mm.buf[i*4], g.mm.buf[i*4+1], g.mm.buf[i*4+2], g.mm.buf[i*4+3]} == mmNoise {
			noise++
		}
	}
	if noise == 0 {
		t.Error("no noise during the blackout")
	}
}

// The walls are drawn once per level, the frame allocates nothing of its own, and the panel is cached per level.
func TestMinimapCachedNoAllocs(t *testing.T) {
	mmRestore(t)
	g := hudPlay(t, false)
	g.stealth = false
	g.loadLevel(0)
	g.mode = ModePlay
	minimapOn = true
	b0 := g.mm.built
	g.minimapFrame()
	b := g.mm.built
	if b != b0+1 {
		t.Fatalf("base built %d times", b-b0)
	}
	allocs := testing.AllocsPerRun(50, func() { g.minimapFrame() })
	if allocs > 0 {
		t.Errorf("minimapFrame allocates %.1f per frame", allocs)
	}
	if g.mm.built != b {
		t.Errorf("the base was rebuilt (%d)", g.mm.built)
	}
	img := ebiten.NewImage(screenW, screenH)
	defer img.Deallocate()
	base := testing.AllocsPerRun(50, func() { g.mm.img.WritePixels(g.mm.buf); img.DrawImage(g.mm.img, &mmOp) })
	got := testing.AllocsPerRun(50, func() { g.drawMinimap(img) })
	if got > base+1 {
		t.Errorf("drawMinimap allocates %.1f, a bare upload+draw %.1f", got, base)
	}
	g.loadLevel(1)
	g.mode = ModePlay
	g.minimapFrame()
	if g.mm.built != b+1 {
		t.Errorf("a new level must rebuild once (built=%d)", g.mm.built)
	}
}

// TAB and Start switch it, only in the session, off by default; it sits in the corner clear of the rest; 1P and 2P draw.
func TestMinimapToggleAndPlace(t *testing.T) {
	mmRestore(t)
	if minimapOn {
		t.Error("the minimap must be off by default")
	}
	for _, two := range []bool{false, true} {
		t.Run(fmt.Sprint("two=", two), func(t *testing.T) {
			mmRestore(t)
			g := hudPlay(t, two)
			g.stealth = false
			g.loadLevel(0)
			g.mode = ModePlay
			minimapOn = false
			tab, start := false, false
			mmKeyFn = func(k ebiten.Key) bool { return k == ebiten.KeyTab && tab }
			mmPadFn = func() bool { return start }
			img := ebiten.NewImage(screenW, screenH)
			defer img.Deallocate()
			g.drawMinimap(img)
			if g.mm.drawn {
				t.Fatal("drawn while off")
			}
			tab = true
			g.updateMinimapInput()
			tab = false
			if !minimapOn {
				t.Fatal("TAB did not turn it on")
			}
			g.drawMinimap(img)
			if !g.mm.drawn {
				t.Fatal("not drawn when on")
			}
			start = true
			g.updateMinimapInput()
			start = false
			if minimapOn {
				t.Fatal("Start did not turn it off")
			}
			minimapOn = true
			g.mode = ModeLevelDone
			g.drawMinimap(img)
			if g.mm.drawn {
				t.Fatal("drawn outside the party")
			}
			x, y, w, h := mmRect()
			if x+w > screenW || y+h > screenH {
				t.Fatal("outside the screen")
			}
			for n, o := range map[string][4]float64{
				"help": {helpBtnX, helpBtnY, helpBtnW, helpBtnH}, "phone": {880, 40, 80, 80},
				"hud": {hudX, hudY, hudW, 90}, "strip": {200, screenH - 70, 560, 70},
			} {
				if rectsHit(x, y, w, h, o[0], o[1], o[2], o[3]) {
					t.Errorf("the minimap touches %s", n)
				}
			}
		})
	}
}

// The help window lists the minimap key.
func TestMinimapInHelp(t *testing.T) {
	loadLanguages()
	if _, ok := langDataEN[helpMinimapText]; !ok {
		t.Error("no English text for the help row")
	}
}
