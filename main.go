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
	"os"
	"runtime/pprof"
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
	level := flag.Int("level", 0, "start level N (1-based, 1..number of levels) directly, debug; the campaign is 1..26 (25 BARCO PIRATA, 26 ESTACIÓN ESPACIAL), then the random-mode pool")
	findF := flag.String("edfind", "", "search text preset in the editor lists (debug)")
	impF := flag.String("edimport", "", "picture to import in the editor on start (debug)")
	impAcc := flag.Bool("edaccept", false, "accept the imported picture straight away (debug)")
	impSave := flag.String("edsave", "", "after -edaccept, save it with this name (debug)")
	gbenchF := flag.Int("gamebench", 0, "game: run this many frames, print ms per frame of Update and Draw and quit (debug)")
	benchF := flag.Int("edbench", 0, "editor: run this many frames, print ms per frame and quit (debug)")
	trophyOvF := flag.Bool("trophyov", false, "open the trophies window at once (debug)")
	helpOvF := flag.Bool("helpov", false, "open the controls help window at once (debug)")
	padHintsF := flag.Bool("padhints", false, "the written hints name the gamepad buttons, as if a pad were in use (debug)")
	profF := flag.String("cpuprofile", "", "write a CPU profile to this file (debug)")
	jurF := flag.Bool("jurassic", false, "start the island of the last party directly (debug)")
	debugEd := flag.Int("edorig", 0, "editor: debug mode on, open the n-th (1..21) original scenario (debug)")
	scenesF := flag.Int("scenes", 0, "open the scenes list; 1 pretends the first ending was seen, 2 pretends everything was (debug)")
	interF := flag.Int("interlude", 0, "play a story interlude directly: 5 (car), 10 (boat) or 20 (rocket) (debug)")
	endStageF := flag.Int("endstage", 0, "with -ending: start at this stage (1 walk, 2 door, 3 bed, 4 credits, 5 stinger, 6 final, 7 arrival, 8 prelude of the ending, 9 party, 10 the LETTER prelude of the space station on its own) (debug)")
	endAtF := flag.Float64("endat", 0, "with -endstage: seconds into the stage (debug)")
	jpUnlF := flag.Bool("jpunlocked", false, "pretend the island is unlocked and open the list of parties (debug)")
	jpArrF := flag.Bool("jparrival", false, "play the helicopter arrival on the island (after the letter) and then the island (debug; with -players 2 for two)")
	invAutoF := flag.Float64("inviteauto", 0, "the invitation presses continue by itself at this second (debug)")
	firstRunF := flag.Bool("firstrun", false, "pretend it is the first time: empty progress in memory, nothing is saved (debug)")
	introF := flag.Bool("intro", false, "play the intro directly (debug)")
	titleF := flag.Bool("title", false, "show the title screen even with debug flags (debug)")
	noTitleF := flag.Bool("notitle", false, "do not show the title screen when the program opens (debug)")
	endF := flag.Int("ending", 0, "play the ending directly: 1 without every trophy, 2 with them, 3 the invitation (debug)")
	vlaneF := flag.Bool("vlane", false, "boat / space levels: the vehicles arrive down a vertical lane (debug)")
	demoMove := flag.String("eddemo", "", "editor debug: with -edorig, move the first 'pool' or 'helipad' of the original by (+60,+40)")
	objTool := flag.Int("edobjtool", 0, "editor debug: Objetos tool (0 place, 1 move, 2 zone)")
	ptTool := flag.Int("edpttool", 0, "editor debug: Puntos tool")
	bowlKidsF := flag.Bool("bowlkids", false, "the kids go to the skate bowls at once and all the time (debug)")
	shot := flag.String("shot", "", "save a screenshot to this PNG and exit, debug")
	shotTick := flag.Int("shottick", 120, "frame for the screenshot")
	camx := flag.Float64("camx", -1, "fixed camera x, debug")
	camy := flag.Float64("camy", -1, "fixed camera y, debug")
	rigtest := flag.String("rigtest", "", "write a rig comparison sheet to this PNG and exit, debug")
	noassets := flag.Bool("noassets", false, "ignore the assets folder and use only built-in content")
	edtab := flag.Int("edtab", 0, "editor tab to open (debug)")
	pevF := flag.String("pevent", "", "force the party event with this id (see -pevents), debug")
	pevsF := flag.Bool("pevents", false, "list the party events (id, chance, window) and exit")
	eventF := flag.Int("event", 0, "start special event N (1 plane,2 ufo,3 souperman,4 satan,5 god,6 zombie), debug")
	settingsF := flag.Bool("settings", false, "open the settings screen (debug)")
	menuDbgF := flag.Bool("menudebug", false, "main menu with the DEBUG row unlocked (debug, with -shot)")
	mainSelF := flag.Int("mainsel", -1, "main menu: start with this row selected (debug, with -shot)")
	playersF := flag.Int("players", 0, "1 or 2: the players switch of the main menu, without saving it (debug)")
	setSelF := flag.Int("setsel", 0, "settings screen: the selected row (debug)")
	ambF := flag.String("ambient", "", "send this ambient vehicle down the road at once (debug)")
	iconF := flag.Bool("iconmode", false, "draw only the logo on a dark background (used to make the app icon)")
	bandF := flag.Int("band", 0, "always play band N (1-11) in the party, debug")
	cutF := flag.Bool("cutphones", false, "cut every red phone a few seconds after the start (debug)")
	kidsF := flag.Bool("kidsnow", false, "the three kids arrive a few seconds after the start (debug)")
	ringF := flag.Int("ring", 0, "1 red / 2 yellow: a guest rings a phone at second 2 (debug)")
	pyroF := flag.Bool("pyronow", false, "the arsonist walks into the party a moment after the start (debug)")
	zapF := flag.Bool("zapnow", false, "electrocute a few guests next to the player a moment after the start (debug)")
	hideDemoF := flag.String("hidedemo", "", "x,y,in|out|busy: the player stands at x,y, goes into the nearest hiding place at second 1 and (out, busy: the partner on the spot) comes out at second 2 (debug)")
	pauseF := flag.Bool("pausenow", false, "open the PAUSE window (Esc) a moment after the party starts (debug)")
	cheerF := flag.Bool("cheernow", false, "the cheerleaders show up at once (debug)")
	vipDemoF := flag.String("vipdemo", "", "VIP zone demo: gate | inside | suit | relief | invite | stun (debug)")
	poisonF := flag.Bool("poisonnow", false, "poison in the pocket and four poisoned guests next to the player (debug)")
	trapDemoF := flag.Bool("trapdemo", false, "the two traps nearest to the start: one half loaded (circle), one armed (LED and radius), held still (debug)")
	snakeF := flag.Bool("snakenow", false, "a snake right next to the start (debug)")
	flag.StringVar(&dbgWild, "wildnow", "", "a wild animal next to the start: frog, squirrel, boar, croc, cheetah, lion, cobaya, capibara (debug)")
	withF := flag.String("with", "", "comma-separated character slugs that replace the first guests (debug)")
	twoDeadF := flag.Int("twodead", 0, "with two players: player 1 or 2 starts dead and the other calls the ambulance; 3|4 nobody calls, 5|6 the living one waits by a phone (debug)")
	twoCuffF := flag.Bool("twocuff", false, "two players: the police are called at 3 s and both are suspects (debug)")
	twoJailF := flag.Int("twojail", 0, "two players: player 1 is caught at 3 s (1) or has 5 s left of the minute in custody (2) (debug)")
	lastGuestsF := flag.Int("lastguests", -1, "half a second in, only N people stay alive: 1-3 shows the last-guests marker, 0 ends the party to see the slow motion of the last kill (debug, progress frozen)")
	noHintF := flag.Bool("nohint", false, "no level hint plate at the start of a party (debug)")
	minimapF := flag.Bool("minimap", false, "start with the minimap on (TAB or Start switches it) (debug, progress frozen)")
	noEmoteF := flag.Bool("noemotes", false, "no emote icons over the guests (debug)")
	killerF := flag.Bool("killer", false, "force a special killer at ~6 s in every party (debug, progress frozen)")
	killerAsF := flag.String("killeras", "", "with -killer: which killer comes, by slug, e.g. el_exterminador (debug)")
	grenadeF := flag.Int("grenade", 0, "SMOKE GRENADE: 1 two grenades in every party (one next to the start), 2 also one in the hand thrown at the biggest group at ~2 s (smoke.go, debug, progress frozen)")
	blackoutF := flag.Bool("blackout", false, "force the BLACKOUT event at ~3 s of the party, 10 s long (debug, progress frozen)")
	stormF := flag.Int("storm", 0, "STORM of the open-air parties: 1 rain (snow at the ski resort) from ~3 s with a flash at ~9 s, 2 also a bolt on a guest at ~12 s (weather.go, debug, progress frozen)")
	dawnF := flag.Int("dawn", -1, "DAWN in a night party: 0 slowly along the party, 1-100 fixed at that percent (weather.go, debug, progress frozen)")
	blameF := flag.Bool("blamenow", false, "at ~2 s a guest is killed next to another one and the police are called: the officer takes the passer-by for the suspect (wrongsuspect.go, debug)")
	camKillF := flag.Bool("camkill", false, "in a level with a camera (3, 5-ish: house party, disco, cinema, casino, station): at ~3 s a guest is killed in front of the camera, to see the red LED/cone and WANTED (debug)")
	stealthBoxF := flag.Bool("stealthbox", false, "with -stealth: start with the cardboard box already on (debug)")
	copMemF := flag.Bool("copmemory", false, "DEBUG option on: a scared witness who meets an officer or a guard tells on you (suspect.go) (debug)")
	stealthF := flag.Bool("stealth", false, "play the STEALTH mode (win a party unseen, cardboard box on E), with -level N for the party (debug)")
	stealth2F := flag.Bool("stealth2p", false, "play the STEALTH mode with TWO players: 1 decoy (noise on F), 2 sneaks past; with -level N for the party (debug)")
	challengeF := flag.Bool("challenge", false, "launch RETO DE HOY (today's challenge: VIP + modifiers, x3 per kill, x5 on completion) ignoring the daily lock; -level N picks the party (debug)")
	levelDoneDemoF := flag.Int("leveldonedemo", 0, "show the end-of-level newspaper at once with sample data; 1 or 2 players (with -players 2) (debug)")
	alertNowF := flag.Bool("alertnow", false, "a pending witness over each player so the \"!\" alert can be seen (debug)")
	coopDemoF := flag.Int("coopdemo", 0, "two players: 1 scoreboard, 2 dance duet and marked trap, 3 carrying the partner (debug)")
	twopF := flag.Bool("twop", false, "start every party with two players, without saving the choice (debug)")
	solidsF := flag.Bool("solids", false, "draw every solid rectangle in red (debug)")
	walkF := flag.Bool("walkbot", false, "the player walks about at random and logs where he gets stuck (debug)")
	roadCarsF := flag.String("roadcars", "", "with -testroad: park these vehicles (comma separated) on the roads (debug)")
	roadF := flag.Bool("testroad", false, "add four one-lane roads to the level (debug)")
	langF := flag.String("lang", "", "game language for this run: es | en (debug)")
	copsF := flag.Bool("copsnow", false, "call the police a few seconds after the start (debug)")
	peleaDieF := flag.Bool("peleadie", false, "with -pevent pelea: the fight always ends with a death (debug, progress frozen)")
	trexKillF := flag.Bool("trexkill", false, "Isla Nublada: the Pawer Rangers are sent at the first call and always bring the T-Rex down (debug, progress frozen)")
	unlockedF := flag.Int("unlocked", 0, "play as if the first N police groups were unlocked: 1 police, 2 +FBI, 3 +military, 4 +SWAT, 5 +Pawer Rangers (debug, progress frozen, nothing saved)")
	credF := flag.Bool("credits", false, "open the credits (debug)")
	allTrF := flag.Bool("alltrophies", false, "pretend every trophy is won, without saving (debug)")
	dbgF := flag.Bool("debugscreen", false, "open the DEBUG screen (debug)")
	immF := flag.Bool("immune", false, "start with the immunity halo on (debug)")
	albumDemoF := flag.Int("albumdemo", 0, "open the OBJETIVOS screen on the PERSONAJES page with example cards: 1 a few unlocked, 2 all (debug, progress frozen, nothing saved)")
	goalSelF := flag.Int("goalsel", 0, "with -albumdemo: the card selected in the OBJETIVOS tab (the page scrolls to it) (debug)")
	goalDetailF := flag.String("goaldetail", "", "open the detail window of OBJETIVOS on tab:index (negative index = from the end; with -albumdemo for example cards) (debug)")
	goalTabF := flag.Int("goaltab", 0, "the OBJETIVOS tab to show: 0 trophies, 1 characters, 2 VIPs, 3 killers, 4 police; with -albumdemo example cards, alone the save as it is (debug, nothing saved)")
	trophyF := flag.Bool("trophyscreen", false, "open the trophies screen (debug)")
	shopF := flag.Bool("shopscreen", false, "open the shop screen (debug)")
	wardPageF := flag.Int("wardpage", 0, "with -wardrobe: the page of the wardrobe, 0 character, 1 killers, 2 police (lends some killers and 24 escapes) (debug)")
	escapesF := flag.Int("escapes", -1, "with -albumdemo or -wardrobe: the escapes the save has, in memory only (debug)")
	wardF := flag.Int("wardrobe", 0, "open the wardrobe with 1 or 2 players, with some clothes lent (debug, progress frozen)")
	wardPickF := flag.Int("wardpick", 0, "with -wardrobe: open the character picker, 1 every character won (as DEBUG Album: all won), 2 a few won (debug)")
	wardPickSelF := flag.Int("wardpicksel", 0, "with -wardpick: the cell selected in the character picker (debug)")
	wearCharF := flag.String("wearchar", "", "player 1 (and 2) wear these characters: slug[,slug2], in memory only (debug, progress frozen)")
	levelSel := flag.Bool("levelselect", false, "open the level select screen (debug)")
	medalDemoF := flag.Bool("medaldemo", false, "every party unlocked with sample medals (bronze, silver, gold in turn), nothing saved (debug)")
	medalToastF := flag.Int("medaltoast", 0, "in the party (-level N): the points jump to bronze (1), silver (2) or gold (3) to show the medal toasts, nothing saved (debug)")
	streakDemoF := flag.Int("streakdemo", 0, "in the party: each player starts a RACHA of N clean kills (streak.go), nothing saved (debug)")
	edload := flag.String("edload", "", "character/object/background to load in the editor, name (debug)")
	edcarry := flag.String("edcarry", "", "held object to give the character in the editor (debug)")
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
	flag.Bool("visible", false, "force a normal visible window even with debug flags (quiet mode off)")
	flag.Parse()
	quiet := quietMode(flag.CommandLine, os.Getenv("PH_QUIET"))
	if debugFlagsActive(flag.CommandLine) {
		progressFrozen = true // any debug/capture/benchmark flag: freeze the progress BEFORE any level or save
	}
	if *pevsF || (*pevF != "" && partyByID(*pevF) == nil && !rwPeventID(*pevF)) { // obras[:stage]: roadworks.go
		fmt.Print(partyList())
		os.Exit(0)
	}
	if *bowlKidsF {
		bowlRateDiv = 0.5
	}
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
	sc = wholeWindowScale(sc)
	gameWinW, gameWinH = int(screenW*sc), int(screenH*sc)
	ebiten.SetWindowSize(gameWinW, gameWinH)
	ebiten.SetWindowTitle(gameWindowTitle())
	if im, err := png.Decode(bytes.NewReader(iconPNG)); err == nil {
		ebiten.SetWindowIcon([]image.Image{im})
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	noAssets = *noassets
	if *zoomf != "" {
		fmt.Sscanf(*zoomf, "%d,%d,%d,%d,%d", &shotCrop[0], &shotCrop[1], &shotCrop[2], &shotCrop[3], &shotZoom)
	}
	loadLanguages()
	if *langF == "en" || *langF == "es" {
		gameLang, editorLang = *langF, *langF
	}
	g := newGame(*mute)
	twoPlayerMode = resolvePlayers(*playersF, twoPlayerMode) // before any startLevel; never saved
	if *twoDeadF > 0 {
		dbgTwoDead, dbgTwoPlayers, twoPlayerMode = *twoDeadF, true, true
	}
	if *twoCuffF || *twoJailF > 0 {
		dbgTwoPlayers, twoPlayerMode = true, true
		dbgTwoCuff, dbgTwoJail = *twoCuffF, *twoJailF
	}
	if *twopF {
		dbgTwoPlayers, twoPlayerMode = true, true // the choice is not saved: settings are only written when the menu switch is used
	}
	if *coopDemoF > 0 {
		dbgTwoPlayers, twoPlayerMode, coopDemo = true, true, *coopDemoF
	}
	if *wearCharF != "" { // before any startLevel: the players are dressed when the party starts
		progressFrozen = true
		g.debugWearChars(*wearCharF)
	}
	if *edit || defaultEditor {
		g.openEditor()
		g.ed.tab = *edtab
		if *edload != "" {
			g.ed.debugLoad(*edload)
		}
		g.ed.lv.mode = *edmode
		g.ed.benchN = *benchF
		g.ed.lv.objTool, g.ed.lv.ptTool = *objTool, *ptTool
		if *debugEd > 0 {
			g.ed.debug = true
			g.ed.pendingOrig = *debugEd
			g.ed.demoMove = *demoMove
		}
		if *profF != "" {
			if pf, err := os.Create(*profF); err == nil {
				pprof.StartCPUProfile(pf)
				benchStop = pprof.StopCPUProfile
			}
		}
		if *impF != "" {
			if b, err := loadImageAny(*impF); err == nil {
				g.ed.requestImport([]droppedImage{{name: strings.ToLower(*impF), b: b}})
				if *impAcc && g.ed.imp != nil {
					g.ed.imp.process()
					g.ed.imp.accept()
					if *impSave != "" {
						g.ed.ch.name, g.ed.ob.name = *impSave, *impSave
						g.ed.save()
					}
				}
			}
		}
		g.ed.ch.find, g.ed.ob.find, g.ed.lv.find = *findF, *findF, *findF
		if *edcarry != "" {
			g.ed.ch.ca.Meta.Carry = *edcarry
			g.ed.ch.dirty = true
		}
		g.ed.ch.mode = minI(*edmode, 1)
		g.ed.ob.mode = minI(*edmode, 1)
	}
	g.selftest = *selftest
	if *genseed != 0 {
		def := generateLevel(*genseed)
		g.baseN = len(levelDefs)
		levelDefs = append(levelDefs, def)
		*level = len(levelDefs)
	}
	fastEv = *fastEvents
	dbgPartyEvent = *pevF
	g.forceBand = *bandF // before the party starts: planBand reads it in loadLevel (it was set after, -band did nothing)
	*level = demoLevel(*level, *levelDoneDemoF, *stealthF || *stealth2F)
	if *level > 0 && !*stealthF && !*stealth2F {
		g.startLevel(*level - 1)
	}
	g.dbg.copMemory = *copMemF
	g.dbgCamKill = *camKillF
	dbgBlame, dbgBlameSetup, dbgBlameLog = *blameF, *blameF, *blameF
	dbgLastGuests = *lastGuestsF
	if *noHintF {
		levelHintOn = false
	}
	if *minimapF {
		minimapOn = true
	}
	if *noEmoteF {
		emotesOn = false
	}
	if *killerAsF != "" {
		dbgKillerAs, *killerF = *killerAsF, true // terminator_killer.go
	}
	if *killerF {
		dbgKiller = true
		g.killerLevelReset()
	}
	if *blackoutF {
		dbgBlackout = true
		g.blackoutLevelReset()
	}
	if *grenadeF > 0 {
		dbgGrenade = *grenadeF
		g.smokeLevelReset()
	}
	if *stormF > 0 || *dawnF >= 0 {
		dbgStorm, dbgDawn = *stormF, float64(*dawnF)
		g.weatherLevelReset()
	}
	if *stealthF || *stealth2F {
		progressFrozen = true
		if *stealth2F {
			g.startStealth2P(maxI(*level-1, 0))
		} else {
			g.startStealth(maxI(*level-1, 0))
		}
		g.sth.box = *stealthBoxF
	}
	if *scenesF > 0 {
		progressFrozen = true
		g.progress.Stats = map[string]int{"seen_end1": 1}
		if *scenesF > 1 {
			g.progress.Stats = debugAllSeen()
		}
		g.sceneSel = *edtab
		g.mode = ModeScenes
	}
	if *introF {
		progressFrozen = true
		g.startIntro()
	}
	if *interF > 0 {
		progressFrozen = true
		if *interF >= 20 {
			g.startInterlude(interRocket, len(campaignLevels)-1)
		} else if *interF >= 15 {
			g.startInterlude(interBalloon, 15)
		} else if *interF >= 10 {
			g.startInterlude(interBoat, 10)
		} else {
			g.startInterlude(interCar, 5)
		}
	}
	if *endStageF == endStageLetter && *endF == 0 {
		*endF = 3 // -endstage 10 alone: the letter prelude of the station, no other flag needed
	}
	if *endF > 0 {
		progressFrozen = true
		switch *endF {
		case 1:
			g.startEndingPre(false)
		case 2:
			g.startEndingPre(true)
		default:
			g.startInvitePre()
		}
		if *endStageF == endStageLetter && g.end != nil { // the prelude of the letter (space station), whatever the ending
			g.enterPrelude(preLetter)
			g.end.t, g.end.total = *endAtF, *endAtF
		} else if *endStageF > 0 && g.end != nil {
			g.end.stage, g.end.t = *endStageF, *endAtF
			g.end.total = *endAtF
			g.end.heroX = 980 - *endAtF*870/endLenFor(endWalk, g.end.trueEnd)
		}
	}
	invAutoAt = *invAutoF
	if *jpUnlF {
		progressFrozen = true
		if g.progress.Stats == nil {
			g.progress.Stats = map[string]int{}
		}
		g.progress.Stats["jp_unlocked"] = 1
		g.progress.Unlocked = campaignN
		g.mode = ModeLevelSelect
		g.menuSel = campaignN + 1
	}
	if *jurF {
		progressFrozen = true
		g.startLevel(specialJurassic)
	}
	if *jpArrF {
		progressFrozen = true
		g.startArrival()
	}
	if *vlaneF && g.lvl != nil && g.lvl.Def.Transit != "" {
		d := g.lvl.Def
		d.CarFrom, d.CarTo, d.ParkX0, d.ParkX1 = Vec{935, -60}, Vec{935, 420}, 0, 0
	}
	g.forceEv = *eventF
	g.forceAmb = *ambF
	if *menuDbgF {
		g.debugMenu = true
	}
	if *mainSelF >= 0 {
		g.mainSel = *mainSelF
	}
	if *settingsF || *credF {
		g.mode = ModeSettings
		g.showCredits = *credF
		g.setSel = *setSelF
		dbgSetSel = *setSelF > 0
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
	dbgPoliceGroups = *unlockedF // police_groups.go
	dbgRexKill = *trexKillF      // rexfight.go
	dbgPeleaDie = *peleaDieF     // pe_pelea.go
	g.dbgCheer = *cheerF
	debugZap = *zapF
	hideDemo = *hideDemoF // hideexit.go
	pauseNow = *pauseF
	pyroNow = *pyroF
	g.dbgRoad = *roadF
	if *roadCarsF != "" {
		g.dbgRoadCars = strings.Split(*roadCarsF, ",")
	}
	g.walkBot = *walkF
	dbgSolids = *solidsF
	if *withF != "" {
		g.dbgWith = strings.Split(*withF, ",")
	}
	g.dbgSnake = *snakeF
	dbgVIPDemo = *vipDemoF
	g.dbgPoison = *poisonF
	g.dbgTrap = *trapDemoF
	g.dbgAlert = *alertNowF
	if *challengeF { // RETO DE HOY (forced, ignores the lock; -level N picks the party); progress frozen by debugFlagsActive
		lv := -1
		if *level > 0 {
			lv = *level - 1
		}
		g.startDaily(true, lv)
	}
	levelDoneDemo = *levelDoneDemoF
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
	if *albumDemoF > 0 {
		progressFrozen = true
		g.mode = ModeTrophies
		g.albumDemo(*albumDemoF)
		if *goalTabF > 0 {
			g.goTab(*goalTabF)
		}
		g.albumSel = *goalSelF
		g.albumClamp()
	} else if *goalTabF > 0 { // the OBJETIVOS tab with the save as it is (progress frozen by debugFlagsActive)
		progressFrozen = true
		g.mode = ModeTrophies
		g.goTab(*goalTabF)
		g.albumSel = *goalSelF
		g.albumClamp()
	}
	if *goalDetailF != "" { // the detail window of OBJETIVOS (goal_detail.go)
		progressFrozen = true
		g.goalDetailDebug(*goalDetailF)
	}
	if *shopF {
		g.mode = ModeShop
		g.progress.Wallet = 5000
		g.shopTab = *edtab
		if g.shopTab == shopChars || g.shopTab == shopKillers || g.shopTab == shopPolice || g.shopTab == shopUniforms { // not sold any more (they are goals)
			g.shopTab = shopFlat[0]
		}
	}
	if *wardF > 0 {
		progressFrozen = true
		n := *wardF
		if n == 3 { // two players, with the second one on the screen
			n, wardP = 2, 1
		}
		wardrobeNumPlayers = func(*Game) int { return n }
		g.progress.Owned = append(g.progress.Owned, "hat:caproja", "cape:roja", "shoe:rojas", "out:teeroja", "out:teeverde", "glasses:none")
		g.syncShop()
		g.progress.Equip = map[string]string{}
		g.progress.EquipP2 = nil
		if n > 1 {
			g.progress.EquipP2 = map[string]string{"hat": "hat:caproja", "cape": "cape:roja"}
		}
		if *wardPageF > 0 { // lend some unlocked killers and escapes (memory only: progress is frozen)
			g.progress.Album = map[string]int{"killer:michael": albumKilled, "killer:jason": albumKilled, "killer:freddy": albumKilled}
			g.progress.Escapes = 24
			g.progress.Equip["kdress:jason"] = "rojo"
			g.syncShop()
			wardKind, wardKiller = *wardPageF%wardKindN, "jason"
		}
		g.applyShop()
		g.mode, g.shopTab = ModeShop, shopWardrobe
		if *wardPickF > 0 { // the character picker (wardrobe_char.go)
			g.debugWardPick(*wardPickF, *wardPickSelF)
		}
	}
	if *escapesF >= 0 && (*albumDemoF > 0 || *wardF > 0) {
		g.progress.Escapes = *escapesF
		g.syncShop()
	}
	if *levelSel {
		g.mode = ModeLevelSelect
	}
	// medals.go / streak.go (progress frozen by debugFlagsActive)
	medalToastDemo, streakDemo = *medalToastF, *streakDemoF
	if *medalDemoF { // medals.go: the party list with medals (progress frozen by debugFlagsActive)
		progressFrozen = true
		g.medalDemo()
	}
	if *camx >= 0 {
		g.camFixed = true
		g.cam = Vec{*camx, *camy}
	}
	g.shot, g.shotTick = *shot, *shotTick
	g.trophyOv = *trophyOvF
	g.helpOv = *helpOvF
	dbgPadHints = *padHintsF
	if *shot != "" || *benchF > 0 || *gbenchF > 0 {
		progressFrozen = true // screenshots and benchmarks never touch the real progress
	}
	if *firstRunF {
		progressFrozen = true
		g.progress = Progress{Unlocked: 1}
		g.syncShop()
		g.applyShop()
	}
	// the very first time (nothing saved, intro not seen): the program opens with the intro
	// the title opens the program; any debug flag (a level, a shot, -intro...) skips it, except -title itself
	if g.mode == ModeMenu && (*titleF || (!*noTitleF && plainStart())) {
		g.startTitle()
	} else if g.mode == ModeMenu && !noAssets && g.firstRun() {
		g.bootIntro()
	}
	var rg ebiten.Game = g
	if *gbenchF > 0 {
		rg = &benchGame{Game: g, n: *gbenchF}
	}
	var runErr error
	if quiet {
		quietSetup()
		runErr = ebiten.RunGameWithOptions(&quietGame{Game: rg}, quietOptions())
	} else {
		runErr = ebiten.RunGame(rg)
	}
	if err := runErr; err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}

var fastEv bool

func checkLevels() {
	defs := builtinLevels()
	if st := loadAssetStore(); st != nil && len(st.Levels) > 0 && !noAssets {
		applyObjects(st.Objs)
		custom, ov := splitLevels(st.Levels)
		levelOverrides = ov
		defs = builtinLevels()
		if len(custom) > 0 {
			defs = nil
			for _, la := range custom {
				defs = append(defs, levelDefFromAsset(la))
			}
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

// plainStart: the program was started the way a player does, with no debug flag (only mute, language and noassets).
func plainStart() bool { return plainStartFor(flag.CommandLine) }

func plainStartFor(fs *flag.FlagSet) bool {
	plain := true
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "mute", "lang", "noassets", "title", "notitle":
		default:
			plain = false
		}
	})
	return plain
}

// resolvePlayers: the effective 2-player switch. -players 1 / 2 forces it; any other value keeps the saved one.
// demoLevel: -leveldonedemo needs a party in progress to show the end-of-level screen over; without -level N it
// would wait on the title / the beds cinematic, so it starts level 1 (the screen then comes up at once).
func demoLevel(level, demo int, stealth bool) int {
	if demo > 0 && level == 0 && !stealth {
		return 1
	}
	return level
}

func resolvePlayers(flagVal int, saved bool) bool {
	if flagVal == 1 || flagVal == 2 {
		return flagVal == 2
	}
	return saved
}

// debugFlagsActive: some debug, capture or benchmark flag was given (anything but the normal-use ones:
// -mute, -lang, -noassets, -title, -notitle, -editor). Then the real progress must never be written.
func debugFlagsActive(fs *flag.FlagSet) bool {
	active := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "mute", "lang", "noassets", "title", "notitle", "editor", "visible":
		default:
			active = true
		}
	})
	return active
}

// debugAllSeen: the "seen" marks of a player who has watched every cinematic (-scenes 2).
func debugAllSeen() map[string]int {
	m := map[string]int{}
	for _, k := range []string{"seen_end1", "seen_end2", "seen_invite", "seen_interlude5", "seen_interlude10", "seen_interlude15", "seen_interlude20", "seen_intro", "seen_arrival"} {
		m[k] = 1
	}
	return m
}
