package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenW = 960
	screenH = 540
	wk      = 1.0 // design coordinates -> native pixels
)

var noAssets bool

//go:embed icon/icon_256.png
var iconPNG []byte

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
	eventF := flag.Int("event", 0, "start special event N (1 plane,2 ufo,3 superman,4 satan,5 god,6 zombie), debug")
	settingsF := flag.Bool("settings", false, "open the settings screen (debug)")
	ambF := flag.String("ambient", "", "send this ambient vehicle down the road at once (debug)")
	iconF := flag.Bool("iconmode", false, "draw only the logo on a dark background (used to make the app icon)")
	bandF := flag.Int("band", 0, "always play band N (1-11) in the party, debug")
	cutF := flag.Bool("cutphones", false, "cut every red phone a few seconds after the start (debug)")
	kidsF := flag.Bool("kidsnow", false, "the three kids arrive a few seconds after the start (debug)")
	ringF := flag.Int("ring", 0, "1 red / 2 yellow: a guest rings a phone at second 2 (debug)")
	cheerF := flag.Bool("cheernow", false, "the cheerleaders show up at once (debug)")
	snakeF := flag.Bool("snakenow", false, "a snake right next to the start (debug)")
	withF := flag.String("with", "", "comma-separated character slugs that replace the first guests (debug)")
	copsF := flag.Bool("copsnow", false, "call the police a few seconds after the start (debug)")
	credF := flag.Bool("credits", false, "open the credits (debug)")
	allTrF := flag.Bool("alltrophies", false, "pretend every trophy is won, without saving (debug)")
	dbgF := flag.Bool("debugscreen", false, "open the DEBUG screen (debug)")
	immF := flag.Bool("immune", false, "start with the immunity halo on (debug)")
	trophyF := flag.Bool("trophyscreen", false, "open the trophies screen (debug)")
	shopF := flag.Bool("shopscreen", false, "open the shop screen (debug)")
	levelSel := flag.Bool("levelselect", false, "open the level select screen (debug)")
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
	expM := flag.String("exportmusic", "", "render a batch of mp3 songs into this folder (needs ffmpeg) and exit")
	batchF := flag.Int("musicbatch", 2, "batch number for -exportmusic (a new number gives new songs)")
	flag.Parse()
	if *expM != "" {
		exportMusic(*expM, *batchF)
		return
	}
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
	if im, err := png.Decode(bytes.NewReader(iconPNG)); err == nil {
		ebiten.SetWindowIcon([]image.Image{im})
	}
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
	g.forceEv = *eventF
	g.forceAmb = *ambF
	if *settingsF || *credF {
		g.mode = ModeSettings
		g.showCredits = *credF
	}
	g.iconMode = *iconF
	g.dbgCut = *cutF
	techLog = *cutF
	if *allTrF {
		progressFrozen = true
		g.progress = cloneProgress(g.progress)
		for _, t := range trophyList {
			g.progress.Trophies[t.id] = true
		}
	}
	if *dbgF {
		g.debugMenu, g.mode = true, ModeDebug
	}
	if *immF {
		progressFrozen = true
		g.dbg.immune = true
	}
	g.dbgCops = *copsF
	g.dbgCheer = *cheerF
	if *withF != "" {
		g.dbgWith = strings.Split(*withF, ",")
	}
	g.dbgSnake = *snakeF
	g.dbgRing = *ringF
	if *kidsF {
		g.kidPlan = []kidPlan{{"sam_sam", 3, false}, {"cesar", 6, false}, {"diego", 9, false}}
	}
	g.forceBand = *bandF
	if *trophyF {
		g.mode = ModeTrophies
		g.trophySel = *edtab
		g.progress.Trophies = map[string]bool{"t_first": true}
		g.toasts = []trophyToast{{name: "Primera sangre", reward: "Gorro de novato", t: 0.2}}
	}
	if *shopF {
		g.mode = ModeShop
		g.progress.Wallet = 5000
		g.shopTab = *edtab
	}
	if *levelSel {
		g.mode = ModeLevelSelect
	}
	if *camx >= 0 {
		g.camFixed = true
		g.cam = Vec{*camx, *camy}
	}
	g.shot, g.shotTick = *shot, *shotTick
	if *shot != "" {
		progressFrozen = true // screenshots never touch the real progress
	}
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
