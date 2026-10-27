package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenW = 960
	screenH = 540
	wk      = 1.0 // design coordinates -> native pixels
)

var noAssets bool

func main() {
	selftest := flag.Int("selftest", 0, "run N frames of automated play and exit (debug)")
	mute := flag.Bool("mute", false, "start without sound")
	level := flag.Int("level", 0, "start level (1-4) directly, debug")
	shot := flag.String("shot", "", "save a screenshot to this PNG and exit, debug")
	shotTick := flag.Int("shottick", 120, "frame for the screenshot")
	camx := flag.Float64("camx", -1, "fixed camera x, debug")
	camy := flag.Float64("camy", -1, "fixed camera y, debug")
	rigtest := flag.String("rigtest", "", "write a rig comparison sheet to this PNG and exit, debug")
	noassets := flag.Bool("noassets", false, "ignore the assets folder and use only built-in content")
	edtab := flag.Int("edtab", 0, "editor tab to open (debug)")
	edload := flag.String("edload", "", "character/object/background to load in the editor, name (debug)")
	edmode := flag.Int("edmode", 0, "level editor mode (debug)")
	zoomf := flag.String("shotzoom", "", "x,y,w,h,zoom crop for the screenshot (debug)")
	edit := flag.Bool("editor", false, "start in the editor")
	castf := flag.String("cast", "", "write the cast contact sheet to this PNG and exit, debug")
	animf := flag.String("animsheet", "", "write the player animation sheet to this PNG and exit, debug")
	anisheet := flag.String("animalsheet", "", "write the animal frames to this PNG and exit, debug")
	check := flag.Bool("check", false, "verify level connectivity and exit, debug")
	genseed := flag.Int64("genseed", 0, "play a generated level with this seed (debug)")
	fastEvents := flag.Bool("fastevents", false, "killer and arrivals appear almost at once (debug)")
	gentest := flag.Int("gentest", 0, "generate N levels of every kind and report (debug)")
	flag.Parse()
	if *gentest < 0 {
		for _, d := range classicLevels() {
			bad := genBad(d)
			fmt.Println(d.Title, "strict-bad:", bad)
		}
		return
	}
	if *gentest > 0 {
		genTest(*gentest)
		return
	}
	if *castf != "" {
		castSheet(*castf, 4)
		return
	}
	if *anisheet != "" {
		animalSheet(*anisheet, 3)
		return
	}
	if *animf != "" {
		animSheet(*animf, 3)
		return
	}
	if *rigtest != "" {
		rigTest(*rigtest, flag.Arg(0))
		return
	}
	noAssets = *noassets
	if *check {
		checkLevels()
		return
	}

	mw, mh := ebiten.ScreenSizeInFullscreen()
	sc := math.Min(float64(mw)*0.92/screenW, float64(mh)*0.88/screenH)
	if sc < 1 {
		sc = 1
	}
	gameWinW, gameWinH = int(screenW*sc), int(screenH*sc)
	ebiten.SetWindowSize(gameWinW, gameWinH)
	ebiten.SetWindowTitle("Party Hard - Fiesta Nocturna")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	noAssets = *noassets
	if *zoomf != "" {
		fmt.Sscanf(*zoomf, "%d,%d,%d,%d,%d", &shotCrop[0], &shotCrop[1], &shotCrop[2], &shotCrop[3], &shotZoom)
	}
	g := newGame(*mute)
	if *edit || defaultEditor {
		g.openEditor()
		g.ed.tab = *edtab
		if *edload != "" {
			g.ed.debugLoad(*edload)
		}
		g.ed.lv.mode = *edmode
	}
	g.selftest = *selftest
	if *genseed != 0 {
		def := generateLevel(*genseed)
		g.baseN = len(levelDefs)
		levelDefs = append(levelDefs, def)
		*level = len(levelDefs)
	}
	fastEv = *fastEvents
	if *level > 0 {
		g.startLevel(*level - 1)
	}
	if *camx >= 0 {
		g.camFixed = true
		g.cam = Vec{*camx, *camy}
	}
	g.shot, g.shotTick = *shot, *shotTick
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}

var fastEv bool

func checkLevels() {
	defs := builtinLevels()
	if st := loadAssetStore(); st != nil && len(st.Levels) > 0 && !noAssets {
		applyObjects(st.Objs)
		defs = nil
		for _, la := range st.Levels {
			defs = append(defs, levelDefFromAsset(la))
		}
		fmt.Println("checking", len(defs), "levels from", st.Root)
	}
	for _, def := range defs {
		l := buildLevel(def)
		bad := 0
		test := func(what string, p Vec) {
			if !l.Free(p) {
				p2 := l.NearestFree(p)
				if dist(p, p2) > 24 {
					fmt.Printf("  [%s] %s %v: no free space nearby\n", def.Name, what, p)
					bad++
					return
				}
			}
			if l.Path(def.Start, p) == nil {
				fmt.Printf("  [%s] %s %v UNREACHABLE\n", def.Name, what, p)
				bad++
			}
		}
		for k, ps := range l.POIs {
			for _, p := range ps {
				test("poi "+k, p)
			}
		}
		for _, r := range def.Roam {
			test("roam", r.Center())
			test("roam-corner", Vec{r.X + 4, r.Y + 4})
		}
		for _, r := range def.Dance {
			test("dance", r.Center())
		}
		for _, pr := range l.Props {
			if pr.K.Hide || pr.K.Phone || pr.K.Trap != "" || pr.K.Poison {
				test("prop "+pr.Name, Vec{pr.X, pr.Y + 14})
			}
		}
		for _, route := range def.Guards {
			for _, p := range route {
				test("guard", p)
			}
		}
		if def.Police == "car" {
			test("carTo", Vec{def.CarTo.X, def.CarTo.Y - 24})
		}
		for _, p := range def.Pads {
			test("pad", p)
		}
		for _, p := range def.Pools {
			_ = p
		}
		fmt.Printf("%s: %d problems (props=%d solids=%d)\n", def.Title, bad, len(l.Props), len(l.solidsP))
	}
}

func genTest(n int) {
	for _, kind := range genKinds {
		ok := 0
		for i := 0; i < n; i++ {
			rng := rand.New(rand.NewSource(int64(i*131 + 7)))
			d := genByKind(rng, kind)
			if genValidate(d) {
				ok++
			}
		}
		fmt.Printf("%-9s %d/%d walkable\n", kind, ok, n)
	}
}

func genWhy(kind string, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	d := genByKind(rng, kind)
	genValidate(d)
	fmt.Println(kind, "bad after prune:", genBad(d), "props", len(d.Props))
}
