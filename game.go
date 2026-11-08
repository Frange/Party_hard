package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hajimehoshi/bitmapfont/v3"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Mode int

const (
	ModeMenu Mode = iota
	ModePlay
	ModeLevelDone
	ModeLose
	ModeFinal
	ModeEditor
	ModeLevelSelect
	ModeSettings
	ModeShop
	ModeConfirm
	ModeTrophies
	ModeDebug
	ModeEnding    // the story ending (ending.go)
	ModeIntro     // the intro of a new game (intro.go)
	ModeScenes    // the cinematics you can watch again (scenes.go)
	ModeInterlude // the story interludes (interlude.go)
	ModeTitle     // the title shown when the program opens (title.go)
)

// main menu rows (ModeMenu); the level grid lives in ModeLevelSelect
const (
	menuContinue = iota
	menuNewGame
	menuShop
	menuTrophies
	menuScenes
	menuSettings
	menuQuit
	menuRows
)

const menuDebug = 100   // the hidden row that the Konami code opens
const menuPlayers = 101 // 1 / 2 players switch (players.go)

var px *ebiten.Image // 1x1 white pixel

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	if textRec != nil && w >= 200 && h >= 100 {
		textRec.fills = append(textRec.fills, panelRect{x, y, w, h, c.A == 255, len(textRec.texts)})
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(px, op)
}

// Vision: everyone only sees in a cone in front of them, up to a medium distance.
const (
	guestRange = 115.0
	guestDot   = 0.5
	guardRange = 100.0
	guardDot   = 0.55
	copRange   = 130.0
	copDot     = 0.5
)

func (g *Game) sees(from Vec, dir Dir, to Vec, rng, minDot float64) bool {
	if g.suitHides(to) { // the gold VIP suit: nobody sees the wearer (vip.go)
		return false
	}
	if g.visK > 0 {
		rng *= g.visK
	}
	rng *= g.blackoutSight() // blackout.go: half as far in the dark
	d := dist(from, to)
	if d > rng {
		return false
	}
	if d > 10 {
		v := dirVec(dir)
		if (v.X*(to.X-from.X)+v.Y*(to.Y-from.Y))/d < minDot {
			return false
		}
	}
	return g.lvl.LOS(from, to)
}

// ---------------------------------------------------------------------------
// Entities
// ---------------------------------------------------------------------------

type gState int

const (
	gIdle gState = iota
	gGoto
	gDance
	gDrink
	gToilet
	gLounge
	gSwim
	gAlarm
	gToPhone
	gCall
	gFlee
	gKiller
	gChatGo
	gWaitP
	gChat
	gKiss
	gSleep
	gVomit
	gDanceWith
	gPunch
	gRun
	gPerform
	gScared
	gReport
	gFun
	gLift   // being abducted: floats up in a UFO beam
	gZombie // infected: slow, bites others into zombies
)

type Guest struct {
	vipWarns          int     // warnings the VIP guards gave him on this level (vipwarn.go)
	vipImmT, vipStun  float64 // immunity after a warning / seconds stunned with stars
	vipCard           bool    // carries a VIP invitation: a golden tag shows beside him (vip.go)
	famousOK          bool    // forced on purpose (debug -with): not a duplicate to swap
	py                *pyroSt // the arsonist (pyro.go): not counted as a guest, does not have to die
	chant             bool
	tech              bool   // on the way to the yellow phone to call the technicians
	quirk             string // special behaviour: mj, mask, taz, beavis, butthead, gb
	gait              string // movement style (gait.go): moon, robot, hips, march, hop, stagger, shy...
	gaitInit          bool
	quirkT            float64
	quirkInit         bool
	greetInit, chatty bool
	lastGreet         float64
	leader            *Guest
	keep              bool // part of a special group: never trimmed by the people limit
	nick              string
	armed             bool
	rampage           int
	origCS            *CharSet
	pos               Vec
	dir               Dir
	cs                *CharSet
	style             int
	perf              string
	state             gState
	next              gState
	dur               float64
	timer             float64
	anim              float64
	curSpd            float64
	moving            bool
	path              []Vec
	icon              string
	iconT             float64
	dead              bool
	gone              bool
	poison            float64 // poison: seconds left to live (see poison.go)
	poisonT0          float64 // ... and how many it started with
	poisonTick        float64 // time to the next symptom
	offered           bool    // being handed a poisoned drink by the player
	scanT             float64
	tgt               *Guest
	tgtPlayer         bool
	repath            float64
	stabT             float64
	partner           *Guest
	role              int
	drunk             int
	lift              float64 // height above the ground while abducted
	need              float64 // bladder/bowel pressure: at 1 they run for the toilet
	hurry             bool    // running to the toilet, not strolling
	hurryT            float64 // seconds spent running for the toilet
	rcv               bool    // about to catch the ball
	gasT              float64
	dAnim             int     // animation of a duet / star moment
	starT, starAct    float64 // when the next moment comes / how long the current one lasts
	starDone          bool
	bitten            float64 // snake venom: seconds left to live
	wcInit            bool
	wcWait            float64
	inside            *Prop // the portable toilet this guest is in
	queue             *Prop // the occupied portable toilet this guest is waiting for
	wcDone            bool  // the accident already left its mark
	lookT             float64
	sawKiller         bool
	sawP              *Player // the player he saw killing (suspect.go: alert / warned)
	warned            bool    // he has already told the police about sawP
	danceEnd          float64
	off               float64
	lastTurn          int
	sawPos            Vec
	fun               int
	runUntil          float64
	sawSlash          bool
	reportTo          *Cop
	name              string // proper display name, or "" for a generic guest
}

func (gu *Guest) out() bool { return gu.dead || gu.gone }

// paceK slows everybody who walks (player, guests, guards, police) a bit more.
var paceK = 0.86

func (gu *Guest) walk(speed, dt float64) bool {
	if gu.quirk == "kid" {
		speed *= kidSpeedK // on wheels: faster than anybody on foot
	}
	m, arr := follow(&gu.pos, &gu.dir, &gu.path, speed*paceK, dt)
	gu.moving, gu.curSpd = m, speed
	return arr
}

type gdState int

const (
	gdPatrol gdState = iota
	gdWait
	gdReact
	gdChase
	gdSearch
	gdInvestigate
	gdBeat
	gdReturn
)

type Guard struct {
	pos      Vec
	dir      Dir
	cs       *CharSet
	route    []Vec
	ri       int
	state    gdState
	path     []Vec
	timer    float64
	anim     float64
	curSpd   float64
	repath   float64
	lost     float64
	chaseT   float64
	sight    float64
	lastSeen Vec
	moving   bool
	icon     string
	iconT    float64
	scanT    float64
	home     Vec
	leash    float64
	dead     bool
	callT    float64
	special  bool
	vipGuard // VIP zone staff (vip.go); zero value = an ordinary guard
}

// Police hierarchy tiers, one per criminality star. Only one tier is ever on
// scene at a time: escalating or de-escalating sends the previous tier home.
const (
	unitPolice   = iota // 1 star: 1-2 officers
	unitFBI             // 2 stars: Mulder & Scully
	unitMilitary        // 3 stars: 3 soldiers
	unitSWAT            // 4 stars: 4 operators
	unitRangers         // 5 stars: 5 Power Rangers
)

type cState int

const (
	cCollect cState = iota
	cBag
	cBagHere
	cGoto
	cSweep
	cReturn
	cChase
	cSearch
	cTired
	cArrest
	cLeave
	cSlash
)

type Cop struct {
	pos       Vec
	dir       Dir
	state     cState
	path      []Vec
	timer     float64
	anim      float64
	curSpd    float64
	moving    bool
	icon      string
	iconT     float64
	lastSeen  Vec
	lost      float64
	repath    float64
	chaseT    float64
	maxChase  float64
	target    *Body
	carry     *Body
	veh       *Vehicle
	medic     bool
	searchT   float64
	onSceneT  float64 // seconds since the officer stepped out: nobody leaves before 20 unless relieved or the suspect is taken
	maxSearch float64
	wait      float64
	alt       int
	bagN      int
	kind      int // unitPolice/unitFBI/unitMilitary/unitSWAT/unitRangers
	shootT    float64
	stuckT    float64
	stuckPos  Vec
	stuckChk  float64
}

type Vehicle struct {
	owner   *Slasher // two players: the killer that comes out of this vehicle
	kind    int      // 0 police car, 1 ambulance, 2 police heli, 3 medical heli
	x, y    float64
	tx, ty  float64
	state   int // 0 arriving, 1 parked, 2 leaving
	spawned bool
	rot     float64
	pad     int
	alt     int
	hold    float64
	unit    int // unitPolice/unitFBI/unitMilitary/unitSWAT/unitRangers
	dead    bool
	lane    bool    // boat / shuttle set up for a lane that runs up or down the screen (airseaviews.go)
	vdir    float64 // +1 it arrives moving down the screen, -1 moving up
	revive  bool    // the ambulance a player called to bring his partner back (revive.go)
}

// HudMsg is one line in the small news queue drawn under the criminality stars.
type HudMsg struct {
	text string
	t    float64
}

type Body struct {
	pos      Vec
	cs       *CharSet
	dir      Dir
	fallT    float64
	bagged   bool
	reported bool
	informer *Guest // the one guest already going to tell the police about this body
	carried  bool
	claimed  *Cop
	seers    []*Guest // people who saw it but have not warned anyone yet
	counted  bool     // already added to the located-kills total
	label    string   // name shown when the mouse is over the body
	byPlayer bool     // only the player's own kills (and traps/poison he set) raise the criminality
	skeleton bool     // electrocuted: it lies there as a burnt skeleton (skeleton.go)
	born     float64  // g.time when it fell
}

type Particle struct {
	pos, vel   Vec
	life, max  float64
	col        color.RGBA
	size, grav float64
}

type Decal struct {
	pos  Vec
	kind int // 0 blood, 1 scorch, 2 vomit
	r    float64
}

type Player struct {
	pos        Vec
	dir        Dir
	cs         *CharSet
	style      int
	anim       float64
	moving     bool
	hidden     bool
	hideProp   *Prop
	carrying   *Body
	doses      int    // doses left in the poison bottle
	spare      string // the other pick-up kind waiting in the second slot (items.go)
	spareDoses int
	cooldown   float64
	stabT      float64
	stamina    float64
	dead       bool
	knockT     float64
	danceT     float64
	running    bool
	item       string  // what he carries: "poison" or "disguise"
	idx        int     // 0 = first player, 1 = second player (twoplayers.go)
	immT       float64 // after a punch from the other player: seconds he cannot be punched again
	msg        string  // a short cry over his head ("¡AUCH!")
	msgT       float64
	deathWhy   string // how he died (revive.go: some deaths cannot be undone by the ambulance)
	disguised  bool   // he put a disguise on since the last case was closed (twoplayers.go)
	danceSpot  Vec    // where he last danced, when and how many times in a row (each player annoys on his own)
	danceLast  float64
	danceCount int
	jail       jailState // cuffs, custody and the prison break (twojail.go)
	suitT      float64   // seconds left of the gold VIP suit: nobody can see or report him (vip.go)
	invite     bool      // he holds a stolen VIP invitation (one use, vip.go)
	passT      float64   // seconds left of the pass he showed at the VIP gate
	vipWarns   int       // warnings the VIP guards gave him on this level (vip.go: 1 thrown out, 2 beaten, 3 killed)
	vipImmT    float64   // seconds left of immunity after a warning
	stunT      float64   // seconds left of being stunned, stars over his head (vip.go)
}

type Assets struct {
	props          map[string][]PropImg
	tank           *ebiten.Image
	ufo            *ebiten.Image
	cuffs          *ebiten.Image
	cowCS          *CharSet
	floors         map[string][]*ebiten.Image
	vignette       *ebiten.Image
	glow           *ebiten.Image
	dark           *ebiten.Image // per-frame darkness layer, lights cut holes in it
	playerCS       *CharSet
	copCS          *CharSet
	medicCS        *CharSet
	fireCS         []*CharSet // firefighters painted in the editor
	techCS         []*CharSet // phone technicians painted in the editor
	guardCS        []*CharSet
	specialGuardCS []*CharSet
	copAlt         []*CharSet
	swatCS         *CharSet
	fbiCS          []*CharSet
	militaryCS     []*CharSet
	rangerCS       []*CharSet
	killerCS       []*CharSet
	heliOX, heliOY float64
	animals        AnimalSprites
	bag            *ebiten.Image
	heli           *ebiten.Image
	heliU          [6]*ebiten.Image // livery per responder kind (police..rangers) + medical at [5]
}

// ---------------------------------------------------------------------------
// Game
// ---------------------------------------------------------------------------

type Game struct {
	streakT                float64
	blk                    blackoutState // the blackout event (blackout.go)
	musicReact             float64       // 0..1: how "alert" the music sounds (musicreact.go)
	cheerT                 float64
	roadCars               []*RoadCar
	roadT                  []float64
	duet                   *duetScene
	duetT                  float64
	beams                  []beam
	floats                 []floatText
	gas                    []gasCloud
	fw                     []fwBox
	pianos                 []pianoDrop
	gasP                   float64
	gasP2                  float64 // gas exposure of player 2 (twoplayers: each player breathes his own)
	train                  *Train
	trainT                 float64
	bubbles                []worldBubble
	cheerGroup             []*Guest
	cheerGroups            int
	trails                 []fireTrail
	forceAmb               string
	tankN                  int
	zombieTint             bool
	setSel                 int // settings screen: row (actions, then "restaurar", "volver")
	setCol                 int // 0,1 keyboard slots, 2 gamepad
	capturing              bool
	forceEv                int // debug: -event N starts event N-1 right away
	events                 []*Event
	eventT                 float64
	evCycle                int
	streaked               bool
	djDead                 bool
	wreckOut               map[Vec]bool
	fireTruck              *Vehicle
	fireCrew               []*Fireman
	blaze                  fireWorld // fire and the arsonist of this party (fire.go, pyro.go)
	fireWait               float64
	dbgCut                 bool
	showCredits            bool
	powerImmune            bool
	pwEvAt                 float64
	pwFamAt                float64
	dbg                    debugState
	debugMenu              bool
	konami                 int
	dbgSel                 int
	menuMsg                string
	menuMsgT               float64
	dbgCops                bool
	dbgCamKill             bool // -camkill: at ~3 s a guest is killed in front of the first camera (camera.go)
	dbgRing                int
	dbgCheer               bool
	dbgWith                []string
	famousOwned            map[string]*Guest // who holds each famous name in this party
	famousPick             map[string]bool   // famous characters already chosen as ordinary guests
	startTarget            int
	dbgRoad                bool
	dbgRoadCars            []string
	walkBot                bool
	botE                   float64
	botT                   float64
	botDx, botDy           float64
	stuckT                 float64
	dbgSnake               bool
	dbgAlert               bool // -alertnow: pending witnesses over the players (the "!" alert)
	stealth, stealthNext   bool // STEALTH mode (stealth.go): on for this level / the next loadLevel starts it
	stealth2, stealth2Next bool // ...with two players (stealth2p.go)
	sth                    stealthState
	cams                   []*secCam // security cameras of this level (camera.go)
	dbgTrap                bool      // -trapdemo: two traps next to the start, one loading and one armed
	dbgPoison              bool      // -poisonnow: poison in the pocket and four poisoned guests next to the player
	dbgSnakeOn             bool
	kidPlan                []kidPlan
	techCaller             *Guest
	techCallT              float64
	techVan                *Vehicle
	techCrew               []*Tech
	techWait               float64
	techScanT              float64
	evManual               int     // rare events the player has set off in this party
	evManualAt             float64 // when the last one went off
	fireTested             bool
	exhibitPlan, exhibited bool
	exhibitT               float64
	names                  map[any]string
	lastMX, lastMY         int
	weapons                []*Prop
	skipSong               bool
	handSat                bool
	handGod                bool
	lib                    *musicLib
	musicMode              string
	songFile               string
	songIdx                int
	nextSong               float64
	wrecks                 []Vec
	respKind               int
	pendingUnit            int // tier a scheduled vehicle will bring; -1 when none is inbound
	stars                  int
	starsDecayT            float64
	swatKills              int
	baseN                  int
	arrivals               int
	arrivalT               float64
	elevators              []*Elevator
	sb                     startBlink // blinking floor marker at the start of a level (startblink.go)
	hud                    hudState   // animated counters of the HUD panel (hud.go)
	lh                     levelHint  // plate with the goal and a tip at the start of a party (levelhint.go)
	mm                     miniMap    // see-through map of the level, bottom right (minimap.go)
	emo                    emoteState // icons over the guests' heads (emote.go)
	slow                   slowState  // slow motion of the last kill (slowmo.go)
	lastMarks              []lastMark // what the last-guests marker drew in the last frame (lastguests.go)
	lastBuf                []Vec      // reused list of the people marked (lastguests.go)
	slasher                *Slasher
	slasherT               float64
	slasherDone            bool
	slasherN               int
	kp                     killerPlan // how often the special killer comes (killerreg.go)
	mode                   Mode
	as                     *Assets
	lvl                    *Level
	levelIdx               int
	snd                    *Sounds
	face                   text.Face
	rng                    *rand.Rand
	tick                   int
	time                   float64

	p                     *Player
	bot2T, bot2Dx, bot2Dy float64     // -walkbot with two players: the second bot
	slot2                 slasherSlot // two players: the second special killer
	coop                  coopState   // two players: the funny cooperative side (coop.go)
	p1, p2                *Player     // two players: p1 is the first one even while the game runs "as" player 2
	twoExtra              int         // two players: how many more guests may be at the party
	guests                []*Guest
	guards                []*Guard
	cops                  []*Cop
	veh                   []*Vehicle
	bodies                []*Body
	decals                []Decal
	parts                 []Particle
	amb                   []*Ambient
	animals               []*Animal
	ambT                  float64
	towT                  float64

	wanted       float64
	seenNow      bool
	policeCalled bool
	pendingCar   float64
	pendingAmb   float64
	ambCalled    bool
	rev          reviveState // the ambulance a player calls for a dead partner (revive.go)
	ambCount     int         // ambulances sent in this case (a second one comes if the bodies keep piling up)
	ambAt        float64
	points       int
	credit       bool
	titleT       float64
	callerGuest  *Guest
	callerGuard  *Guard
	suspect      *Guest
	lastKiller   lastKill // the last killer somebody saw: the police go after him (suspect.go)
	alertPos     Vec
	killPos      Vec
	murderT      float64
	shake        float64
	flash        float64

	rex                                         *JRex
	jpDinos                                     []*jpDino
	jpEggs                                      []*jpEgg
	gibs                                        []Gib
	charSc, visK                                float64 // size of the people on this level (1 = normal) and what it does to how far they see
	cam                                         Vec
	camI                                        Vec
	msgs                                        []HudMsg
	news                                        NewsBank
	newsLast                                    map[string]int
	dialogue                                    DialogueBank
	progress                                    Progress
	banked                                      bool
	challenge                                   bool      // RETO: x3 per kill, x5 to the total on completion (levelscore.go)
	run                                         runStats  // kills by cause, times seen / cuffed, final breakdown (levelscore.go)
	killCause                                   killCause // how the kill about to happen was made (levelscore.go)
	iconMode                                    bool
	perkSpeed, perkStamina, perkCool, perkMoney float64
	shopTab, shopSel, shopScroll                int
	shopMsg                                     string
	shopCache                                   map[string]*CharSet
	danceSpot                                   Vec
	danceLast                                   float64
	danceCount                                  int
	confirmT                                    float64
	confirmSel                                  int
	band                                        *Band
	bandStage                                   bool
	bandSpot                                    Vec
	bandUsed                                    []bool
	forceBand                                   int
	rexImg                                      [2]*ebiten.Image
	talkX, talkY, talkT                         float64
	talkN, talkKind                             int
	bedQuiet                                    bool // the bedroom of the ending: no party outside
	end                                         *endState
	intro                                       *introState
	inter                                       *interState
	sceneReplay                                 bool        // a cinematic is being watched again: it returns to sceneBack
	title                                       *titleState // the title that opens the program (title.go)
	introBoot                                   bool        // the intro that opens the program the first time: it ends in the main menu
	sceneBack                                   Mode
	sceneSel                                    int
	bedEmpty, bedNoTalk                         bool // the intro: he is out of bed / nobody mutters
	trophyDirty                                 bool
	toasts                                      []trophyToast
	trophySel, trophyScroll                     int
	trophyTab, albumSel, albumScroll            int // page of the TROFEOS screen (0 list, 1 album) and the album cursor (album.go)
	sec                                         secretState
	trophyOv                                    bool    // the in-game trophies window is open (the game is paused)
	helpOv                                      bool    // the controls window (help.go) is open (the game is paused)
	trophyHudA                                  float64 // 1 = solid, lower while the player walks under the counter
	tierBase                                    map[string][]CharSpec
	loseReason                                  string
	kills                                       int
	total                                       int
	endTime                                     float64
	endAt                                       float64

	store         *AssetStore
	ed            *Editor
	rigCache      map[*CharAsset]*CharSet
	pendingExport bool
	fromEditor    bool
	testVictim    *Guest
	noHeat        bool // the kill that is happening does not count as a crime (zombies)
	locatedN      int  // player kills the police have located so far (never goes down)
	starFloor     int  // highest star level reached: the next call starts there again
	killByPlayer  bool // set while a body is spawned by a player kill
	menuSel       int  // row in the level grid (ModeLevelSelect)
	mainSel       int  // row in the main menu (ModeMenu)
	selftest      int
	god           bool
	shot          string
	shotTick      int
	camFixed      bool
}

func newGame(mute bool) *Game {
	g := &Game{
		snd:      newSounds(mute),
		face:     text.NewGoXFace(bitmapfont.Face),
		rng:      rand.New(rand.NewSource(rand.Int63())),
		rigCache: map[*CharAsset]*CharSet{},
		newsLast: map[string]int{},
	}
	g.news = loadNewsBank()
	g.dialogue = loadDialogueBank()
	g.progress = loadProgress()
	g.trophyDirty = true // trophies reached with the saved stats (e.g. the police types) are granted at once
	buildShopCatalog()
	g.syncShop()
	g.perkSpeed, g.perkStamina, g.perkCool, g.perkMoney = 1, 6, 1, 1
	binds = loadBindings()
	loadPlayerSettings() // 1 or 2 players, who plays with what, the keys of player 2 (players.go)
	wb := newPix(1, 1)
	wb.p[0], wb.p[1], wb.p[2], wb.p[3] = 255, 255, 255, 255
	px = wb.img()
	// first run: write the built-in content as editable asset files
	if !noAssets {
		migrateAssets()
	}
	root := assetsRoot(false)
	if root == "" && !noAssets {
		root = assetsRoot(true)
		if root != "" {
			nc, no := exportCharacters(root), exportObjects(root)
			fmt.Println("assets: exported", nc, "characters and", no, "objects to", root)
			os.WriteFile(filepath.Join(root, "VERSION"), []byte(fmt.Sprint(assetVersion)), 0o644)
			g.pendingExport = true
		}
	}
	if !noAssets && root != "" {
		if n := refreshBakedAssets(root); n > 0 {
			fmt.Println("assets: redrew", n, "characters that had their held object painted in")
		}
		if n := processToGenerate(root); n > 0 {
			fmt.Println("togenerate: imported", n, "editable character models")
		}
	}
	if !noAssets {
		g.store = loadAssetStore()
	}
	if g.store != nil {
		applyObjects(g.store.Objs)
	}
	as := &Assets{props: buildPropImages(), floors: map[string][]*ebiten.Image{}}
	for _, k := range []string{"wood", "deck", "tile", "carpet", "grass", "pave", "asphalt", "concrete", "marble", "parquet", "sand", "dirt", "rooftop", "rooftop2", "boardwalk", "cobble", "pitch", "white", "arena", "mall", "snow", "chalk", "rails", "edge"} {
		for v := 0; v < 3; v++ {
			as.floors[k] = append(as.floors[k], stylizeFloor(k, genFloor(k, v)).img())
		}
	}
	as.glow = genGlow(96).img()
	as.vignette = genVignette().img()
	as.dark = ebiten.NewImage(screenW, screenH)
	as.bag = genBodyBag().scaled(wk).outlinedSel(0.45).img()
	as.heli = genHeli().scaled(wk).outlinedSel(0.45).img()
	as.heliOX, as.heliOY = math.Round(41*wk+1), math.Round(34*wk+1)
	liv := [6][3]uint32{{0xdcdce8, 0x2a4a9a, 0}, {0x20202c, 0x5a2a9a, 1}, {0x5a6a3a, 0x2c3a1c, 2}, {0x30323a, 0x20222a, 3}, {0xf4f4f8, 0xd02a5a, 4}, {0xf8f8fc, 0xd02828, 5}}
	for i, l := range liv {
		as.heliU[i] = genHeliPal(l[0], l[1], int(l[2])).scaled(wk).outlinedSel(0.45).img()
	}
	as.tank = genTank().scaled(1.5).outlinedSel(0.45).img()
	as.ufo = genUFO().scaled(1.5).outlinedSel(0.45).img()
	as.cuffs = genCuffs().scaled(1.5).img()
	as.animals = buildAnimalSprites()
	cow := presetCowboy(g.rng)
	as.cowCS = newCharSet(&cow)
	if g.store != nil {
		for _, oa := range g.store.Objs {
			switch oa.Meta.Name {
			case "body_bag":
				as.bag = oa.Frames[0].img()
			case "heli":
				// the exported copy of the built-in helicopter keeps its liveries; only a redrawn one replaces them
				if sameSprite(oa.Frames[0], genHeli().scaled(wk).outlinedSel(0.45)) {
					break
				}
				as.heli = oa.Frames[0].img()
				as.heliU = [6]*ebiten.Image{}
				as.heliOX, as.heliOY = float64(oa.Meta.AnchorX), float64(oa.Meta.AnchorY)
			case "tank":
				as.tank = oa.Frames[0].img()
			case "ufo":
				as.ufo = oa.Frames[0].img()
			}
		}
	}
	g.as = as
	g.setupCast()
	g.rebuildLevels()
	g.loadLevel(0)
	g.mode = ModeMenu
	g.snd.StartMusic() // music is on by default
	return g
}

func (g *Game) loadLevel(i int) {
	g.stealthBegin() // STEALTH mode on or off for this level (stealth.go)
	g.bankPoints()
	g.banked = false
	g.run = runStats{}
	g.killCause = kcNone
	g.applyShop()
	g.levelIdx = i
	g.charSc, g.visK, paceK = 1, 1, 0.86
	var def *LevelDef
	if i == specialJurassic {
		def = jurassicLevel() // the last party: not part of the campaign list
	} else {
		def = levelDefs[i]
	}
	if g.stealth {
		def = stealthDef(def) // short hedges that make corridors (stealth.go)
	}
	if def.CharScale > 0 && def.CharScale != 1 {
		g.charSc, g.visK, paceK = def.CharScale, math.Sqrt(def.CharScale), 0.86*def.CharScale
	}
	nickRNG, nickUsed = g.rng, map[string]bool{}
	g.famousOwned, g.famousPick = map[string]*Guest{}, map[string]bool{}
	g.pickStartTarget()
	vipDecision = ""
	if i != specialJurassic {
		vipDecision = g.vipRoll(def) // vipchance.go: own dice, g.rng untouched
	}
	g.lvl = buildLevel(def)
	if nd := g.planBand(def); nd != nil {
		def = nd
		g.lvl = buildLevel(def)
	}
	vipDecision = ""
	g.renderBG()
	g.setupCameras() // security cameras of this level (camera.go)
	g.time = 0
	g.p = &Player{pos: def.Start, dir: DirUp, cs: g.as.playerCS, stamina: g.perkStamina, style: g.rng.Intn(4)}
	g.setupPlayers(def) // the second player, when there are two
	g.sb.reset(g)
	g.slow = slowState{}
	g.guests, g.guards, g.cops, g.veh, g.bodies, g.decals, g.parts, g.amb, g.wrecks = nil, nil, nil, nil, nil, nil, nil, nil, nil
	g.wreckOut, g.fireTruck, g.fireCrew, g.fireWait = nil, nil, nil, 0
	g.blaze = fireWorld{}
	g.techCaller, g.techVan, g.techCrew, g.techWait, g.techScanT = nil, nil, nil, 0, 0
	g.evManual, g.evManualAt = 0, -999
	g.pwEvAt, g.pwFamAt = -999, -999
	g.ambT = 6 + g.rng.Float64()*10
	if g.lvl.Def.Highway {
		g.ambT = 0.5
	}
	g.towT = 20 + g.rng.Float64()*15
	g.spawnAnimals()
	g.spawnJurassic()
	g.wanted, g.kills = 0, 0
	g.policeCalled, g.pendingCar, g.pendingAmb, g.ambCalled, g.ambCount = false, -1, -1, false, 0
	g.points, g.titleT = 0, 4.5
	g.callerGuest, g.callerGuard, g.suspect = nil, nil, nil
	g.lastKiller = lastKill{}
	g.murderT = 80 + g.rng.Float64()*80
	g.slasher, g.slasherDone, g.slasherT, g.slasherN = nil, false, 28+g.rng.Float64()*40, 0
	g.slot2 = slasherSlot{t: 50 + g.rng.Float64()*50} // two players: the second killer comes later
	g.streakT, g.streaked = 55+g.rng.Float64()*90, false
	g.blackoutLevelReset()
	g.killerLevelReset()
	g.train, g.trainT = nil, 10+g.rng.Float64()*14
	if fastEv {
		g.trainT = 2
	}
	g.gas, g.fw, g.pianos, g.floats, g.gasP, g.gasP2 = nil, nil, nil, nil, 0, 0
	g.roadCars, g.roadT = nil, nil
	g.duet, g.duetT, g.beams = nil, 25+g.rng.Float64()*25, nil
	if fastEv {
		g.duetT = 3
	}
	g.cheerT, g.cheerGroup, g.cheerGroups = 20+g.rng.Float64()*50, nil, 0
	if g.rng.Float64() < 0.35 {
		g.cheerT = 1e9 // not every party has cheerleaders
	}
	g.trails, g.tankN = nil, 0
	g.events, g.eventT = nil, g.time+45+g.rng.Float64()*45
	g.respKind, g.swatKills = unitPolice, 0
	g.pendingUnit = -1
	g.stars, g.starsDecayT = 0, 0
	g.locatedN, g.starFloor = 0, 0
	g.handSat, g.handGod = false, false
	g.musicMode = ""
	g.djDead, g.songIdx, g.nextSong = false, 0, 70+g.rng.Float64()*40
	g.arrivals, g.arrivalT, g.elevators = 0, 40+g.rng.Float64()*30, nil
	if fastEv {
		g.slasherT, g.arrivalT = 6, 14
		g.ambT = 3
	}
	g.msgs, g.shake, g.flash = nil, 0, 0
	sayReset()
	for k := 0; k < def.Guests*14/10; k++ {
		var pos Vec
		for tries := 0; tries < 40; tries++ {
			pos = g.lvl.RandomRoam(g.rng)
			if dist(pos, def.Start) > 140 {
				break
			}
		}
		cs, name := g.newGuestCS()
		gu := &Guest{pos: pos, dir: Dir(g.rng.Intn(4)), cs: cs, name: name, style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 3, role: k % 2}
		g.guests = append(g.guests, gu)
	}
	for _, pf := range def.Performers {
		cs, name := g.newGuestCS()
		if g.store == nil || (len(g.store.Chars["people"])+len(g.store.Chars["special_people"])) == 0 {
			spec := genericSpec(g.rng)
			spec.Female = false
			if pf.Role == "dj" {
				spec.Glasses = GlassShades
				spec.Hat, spec.HatCol = HatCap, hx(0xd04080)
			}
			if pf.Role == "sing" && def.Title == "MIAMI BEACH" {
				spec = presetElvis(g.rng)
			}
			cs, name = newCharSet(&spec), nameFromSpec(&spec)
		}
		if m := g.stageMember(pf.Role); m != nil {
			if f := presetByName(m.slug); f != nil {
				spec := f(g.rng)
				cs, name = newCharSet(&spec), nameFromSpec(&spec)
			}
		}
		gu := &Guest{pos: pf.Pos, dir: pf.Dir, cs: cs, name: name, perf: pf.Role, state: gPerform, role: g.rng.Intn(2), keep: true}
		g.guests = append(g.guests, gu)
	}
	for k, route := range def.Guards {
		var hx, hy float64
		for _, p := range route {
			hx += p.X
			hy += p.Y
		}
		home := Vec{hx / float64(len(route)), hy / float64(len(route))}
		gd := &Guard{pos: route[0], cs: g.as.guardCS[k%len(g.as.guardCS)], route: route, ri: 1 % len(route), state: gdWait, timer: g.rng.Float64(), home: home, leash: 170}
		g.guards = append(g.guards, gd)
	}
	if len(g.as.specialGuardCS) > 0 && len(def.Guards) > 0 {
		route := def.Guards[g.rng.Intn(len(def.Guards))]
		sg := &Guard{pos: route[len(route)-1], cs: g.as.specialGuardCS[g.rng.Intn(len(g.as.specialGuardCS))], route: route, ri: 0, state: gdWait, timer: g.rng.Float64(), home: route[0], leash: 220, special: true}
		g.guards = append(g.guards, sg)
	}
	g.addLifeguards()
	g.spawnHippies()
	g.spawnWally()
	g.spawnStars()
	g.planKids()
	g.exhibitPlan, g.exhibited, g.exhibitT = g.rng.Float64() < 0.25, false, 35+g.rng.Float64()*70
	if fastEv {
		g.exhibitPlan, g.exhibitT = true, 8
	}
	g.scatterWeapons()
	g.pyroPlan()
	g.trimGuests()
	g.fillGuests(def.Start)
	g.uniqueFamous()
	g.spawnVIP() // VIP zone staff and the invitation carriers (vip.go)
	g.total = len(g.guests) + len(g.guards) - g.vipStaffN()
	g.hud.reset(g) // HUD panel counters start from the real numbers (hud.go)
	g.lh.reset(g)  // level hint (levelhint.go)
	g.mm.reset()   // the minimap rebuilds its walls for this level (minimap.go)
	g.emo.reset()  // guest emotes (emote.go)
	g.cam = g.camTarget()
}

// beginNewGame goes straight to the first party.
func (g *Game) beginNewGame() {
	g.startLevel(0) // the intro of the very first run already played when the program opened (firstrun.go)
}

// testStartLevel: tests replace the heavy level loading (it needs a graphics context) to follow the flow.
var testStartLevel func(g *Game, i int)

func (g *Game) startLevel(i int) {
	if testStartLevel != nil {
		testStartLevel(g, i)
		return
	}
	if g.stealth { // retry in the STEALTH mode stays in it (and with its two players)
		g.startStealthN(i, g.stealth2)
		return
	}
	g.loadLevel(i)
	g.stat("played", 1)
	saveProgress(g.progress) // from now on there is a game to continue
	g.mode = ModePlay
	g.snd.StartMusic()
	g.say(g.lvl.Def.Intro)
}

// startRandomLevel picks from the random-mode pool: everything after the
// fixed campaign block (other venues, plus re-seeded campaign kinds). It
// never gates on unlock progress.
func (g *Game) startRandomLevel() {
	n := len(levelDefs)
	if n <= campaignN {
		g.startLevel(g.rng.Intn(maxI(n, 1)))
		return
	}
	g.startLevel(campaignN + g.rng.Intn(n-campaignN))
}

// say queues a short news line under the criminality stars. Up to 2 lines
// show at once; older ones are dropped so newer news is never hidden.
func (g *Game) say(s string) { g.sayOffer(s) }

// aliveCount: guests and guards left to kill.
func (g *Game) aliveCount() int {
	n := 0
	for _, gu := range g.guests {
		if !gu.out() && gu.py == nil { // the arsonist does not have to die for the party to end
			n++
		}
	}
	for _, gd := range g.guards {
		if !gd.dead && gd.vip == 0 { // the VIP zone staff does not have to die for the party to end
			n++
		}
	}
	for _, s := range g.allSlashers() {
		if s.state == slHunt || s.state == slArrive {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (g *Game) Update() error {
	g.pickLanguage()
	g.pickLanguage()
	g.tick++
	inputP, twoInput = 0, g.is2P() && g.mode == ModePlay // two players: the keys and pads are split only while playing
	refreshPlayerDevices()                               // two players: a pad plugged in or pulled out takes effect at once
	g.updateToasts(1.0 / 60)
	g.syncMusic()
	g.updateMusicReact(1.0 / 60)
	g.updateAmbientSound() // quiet level ambience, off by default (ambient_sound.go)
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.snd.muted = !g.snd.muted
		if g.snd.muted {
			g.snd.StopMusic()
		} else {
			g.snd.StartMusic()
		}
	}
	if g.pendingExport {
		g.pendingExport = false
		if root := assetsRoot(false); root != "" {
			g.exportBuiltinLevels(root)
			g.store = loadAssetStore()
			g.rebuildLevels()
			g.loadLevel(0)
		}
	}
	if g.selftest > 0 {
		g.runSelftest()
	}
	if g.mode == ModeEditor {
		return g.updateEditor()
	}
	if g.updateHelp() { // the controls window: the whole game is paused while it is open (help.go)
		return nil
	}
	if g.updateTrophyHud() {
		return nil
	}
	switch g.mode {
	case ModeMenu:
		g.time += 1.0 / 60
		if g.menuMsgT > 0 {
			g.menuMsgT -= 1.0 / 60
		}
		g.bankPoints()
		g.checkTrophies()
		list := g.menuList()
		rows := len(list)
		g.konamiStep()
		click := g.mouseMenu(func(mx, my float64) int {
			for i := 0; i < rows; i++ {
				if y := menuRowY(i, rows); mx >= 100 && mx <= 430 && my >= y-6 && my <= y+30 {
					return i
				}
			}
			return -1
		}, &g.mainSel)
		g.mainSel = clampI(g.mainSel, 0, rows-1)
		if actDown.just() {
			g.mainSel = (g.mainSel + 1) % rows
		}
		if actUp.just() {
			g.mainSel = (g.mainSel + rows - 1) % rows
		}
		if list[g.mainSel] == menuPlayers && (actLeft.just() || actRight.just()) {
			g.togglePlayers()
		}
		refreshPlayerDevices() // pads can be plugged in or pulled out at any time
		if actConfirm.just() || click {
			switch list[g.mainSel] {
			case menuPlayers:
				g.togglePlayers()
			case menuDebug:
				g.dbgSel = 0
				g.mode = ModeDebug
			case menuContinue:
				g.menuSel = clampI(g.progress.Unlocked-1, 0, campaignN-1)
				g.mode = ModeLevelSelect
			case menuNewGame:
				if g.hasProgress() {
					g.confirmT, g.confirmSel = 0, 0
					g.mode = ModeConfirm
				} else {
					g.beginNewGame()
				}
			case menuShop:
				g.shopTab, g.shopSel, g.shopScroll, g.shopMsg = 0, 0, 0, ""
				g.mode = ModeShop
			case menuTrophies:
				g.trophySel, g.trophyScroll = 0, 0
				g.mode = ModeTrophies
			case menuScenes:
				g.sceneSel = 0
				g.mode = ModeScenes
			case menuSettings:
				g.setSel, g.setCol, g.capturing = 0, 0, false
				g.mode = ModeSettings
			case menuQuit:
				return ebiten.Termination
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
			g.openEditor()
		}
	case ModeTrophies:
		g.updateTrophies()
	case ModeEnding:
		g.updateEnding()
	case ModeIntro:
		g.updateIntro()
	case ModeTitle:
		g.updateTitle()
	case ModeInterlude:
		g.updateInterlude()
	case ModeScenes:
		g.time += 1.0 / 60
		g.updateScenes()
	case ModeDebug:
		g.time += 1.0 / 60
		g.updateDebug()
	case ModeConfirm:
		g.updateConfirm()
	case ModeShop:
		g.updateShop()
	case ModeSettings:
		g.time += 1.0 / 60
		g.updateSettings()
	case ModeLevelSelect:
		g.time += 1.0 / 60
		rows := campaignN + 1 // the campaign slots + the fixed "ALEATORIO" row
		if g.jpListed() {
			rows++ // ...and the island, once it has been unlocked
		}
		click := g.mouseMenu(func(mx, my float64) int {
			for i := 0; i < rows; i++ {
				cx, y := levelColX[i/levelRowsPerCol], 118+float64(i%levelRowsPerCol)*34
				if mx >= cx-112 && mx <= cx+112 && my >= y-8 && my <= y+24 {
					return i
				}
			}
			return -1
		}, &g.menuSel)
		if actDown.just() {
			g.menuSel = (g.menuSel + 1) % rows
		}
		if actUp.just() {
			g.menuSel = (g.menuSel + rows - 1) % rows
		}
		if actRight.just() || actLeft.just() {
			step := levelRowsPerCol
			if actLeft.just() {
				step = rows - levelRowsPerCol%rows
			}
			g.menuSel = (g.menuSel + step) % rows // jump between the columns
		}
		for i, k := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
			if i < campaignN && i < g.progress.Unlocked && inpututil.IsKeyJustPressed(k) {
				g.startLevel(i)
			}
		}
		if actConfirm.just() || click {
			if g.menuSel == campaignN {
				g.startRandomLevel()
			} else if g.menuSel == campaignN+1 && g.jpListed() {
				g.startLevel(specialJurassic)
			} else if g.menuSel < g.progress.Unlocked {
				g.startLevel(g.menuSel)
			}
		}
		if actBack.just() {
			g.mode = ModeMenu
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.startRandomLevel()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyC) { // RETO on / off (levelscore.go); the one place that switches it
			g.challenge = !g.challenge
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyG) {
			if g.baseN == 0 || g.baseN > len(levelDefs) {
				g.baseN = len(levelDefs)
			}
			def := generateLevel(g.rng.Int63())
			levelDefs = append(levelDefs[:g.baseN], def)
			g.menuSel = len(levelDefs) - 1
			g.startLevel(g.menuSel)
		}
		for _, gu := range g.guests {
			g.updateGuest(gu, 1.0/60)
		}
	case ModePlay:
		g.updatePlay(1.0 / 60 * g.slowScale()) // slow motion of the last kill (slowmo.go)
		g.slowTick()
	case ModeLevelDone:
		g.time += 1.0 / 60
		g.updateParticles(1.0 / 60)
		if actConfirm.just() || actBack.just() || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.levelDoneContinue()
		}
	case ModeFinal:
		g.time += 1.0 / 60
		if actConfirm.just() || actBack.just() || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.loadLevel(0)
			g.mode = ModeMenu
		}
	case ModeLose:
		g.time += 1.0 / 60
		g.updateParticles(1.0 / 60)
		if g.shake > 0 {
			g.shake -= 1.0 / 60
		}
		for _, gd := range g.guards {
			gd.anim += 1.0 / 60 * 10
		}
		if g.time-g.endAt > 1.0 {
			if inpututil.IsKeyJustPressed(ebiten.KeyR) || actConfirm.just() || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				g.startLevel(g.levelIdx)
			}
			if actBack.just() || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
				if g.fromEditor {
					g.fromEditor = false
					g.openEditor()
					return nil
				}
				g.loadLevel(0)
				g.mode = ModeMenu
			}
		}
	}
	return nil
}

func clampF(v, lo, hi float64) float64 {
	if hi < lo {
		return lo
	}
	return math.Max(lo, math.Min(hi, v))
}

func (g *Game) camTarget() Vec {
	return Vec{0, 0}
}

func (g *Game) runSelftest() {
	if g.tick == 1 && g.mode != ModePlay {
		g.startLevel(g.levelIdx)
	}
	if g.tick == 1 {
		g.god = true
	}
	if g.tick >= g.selftest {
		gs := []int{}
		for _, gd := range g.guards {
			gs = append(gs, int(gd.state))
		}
		cs := []int{}
		for _, c := range g.cops {
			cs = append(cs, int(c.state))
		}
		st := map[gState]int{}
		for _, gu := range g.guests {
			if !gu.out() {
				st[gu.state]++
			}
		}
		if g.testVictim != nil {
			fmt.Println("horse kick test: victim dead =", g.testVictim.dead)
		}
		if s := g.slasher; s != nil {
			fmt.Printf("slasher: %s state=%d hp=%d chainsaw=%v\n", s.name, s.state, s.hp, s.chainsaw)
		}
		fmt.Printf("guests=%d total=%d arrivals=%d veh=%d\n", len(g.guests), g.total, g.arrivals, len(g.veh))
		lastMsg := ""
		if n := len(g.msgs); n > 0 {
			lastMsg = g.msgs[n-1].text
		}
		kinds := make([]int, len(g.cops))
		for i, c := range g.cops {
			kinds[i] = c.kind
		}
		fmt.Printf("selftest ok: level=%d time=%.1f left=%d cops=%v kinds=%v cars=%d guards=%v wanted=%.1f stars=%d respKind=%d pendingUnit=%d called=%v mode=%d bodies=%d states=%v msg=%q\n",
			g.levelIdx, g.time, g.aliveCount(), cs, kinds, len(g.veh), gs, g.wanted, g.stars, g.respKind, g.pendingUnit, g.policeCalled, g.mode, len(g.bodies), st, lastMsg)
		os.Exit(0)
	}
	if g.mode == ModePlay && g.tick%260 == 0 && g.aliveCount() > 8 {
		for _, gu := range g.guests {
			if !gu.out() && gu.state != gSwim {
				g.p.pos = g.lvl.NearestFree(Vec{gu.pos.X, gu.pos.Y + 14})
				g.stab()
				break
			}
		}
	}
	if g.mode == ModePlay && g.tick%500 == 100 {
		g.p.danceT = 4
		g.danceRoll()
	}
	if g.mode == ModePlay && g.tick == 1500 {
		g.murderT = 0
	}
	if g.mode == ModePlay && g.tick == 300 {
		for _, a := range g.animals {
			if a.kind == animHorse {
				for _, gu := range g.guests {
					if !gu.out() && gu.perf == "" {
						gu.pos = g.lvl.NearestFree(Vec{a.pos.X - a.face*16, a.pos.Y})
						gu.state, gu.timer, gu.path = gIdle, 9999, nil
						g.testVictim = gu
						break
					}
				}
				break
			}
		}
	}
	if g.mode == ModeLose || g.mode == ModeLevelDone {
		g.startLevel(g.levelIdx)
		g.god = true
	}
}

func (g *Game) updatePlay(dt float64) {
	g.time += dt
	for i := range g.msgs {
		g.msgs[i].t -= dt
	}
	for len(g.msgs) > 0 && g.msgs[0].t <= 0 {
		g.msgs = g.msgs[1:]
	}
	g.sayPromote()
	if g.shake > 0 {
		g.shake -= dt
	}
	if g.flash > 0 {
		g.flash -= dt
	}
	if actBack.just() {
		if g.fromEditor {
			g.fromEditor = false
			g.openEditor()
			return
		}
		g.loadLevel(0)
		g.mode = ModeMenu
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.startLevel(g.levelIdx)
		return
	}
	g.coopTrack(0, func() { g.updatePlayer(dt) })
	if g.is2P() {
		g.updateTwoPlayers(dt)
		g.asPlayer(1, func() { g.coopTrack(1, func() { g.updatePlayer(dt) }) })
		g.coopUpdate(dt)
	}
	g.trackCuffs()
	g.levelDoneDemoSetup()
	g.sb.update(g, dt)
	g.updateHUD(dt)
	g.updateLevelHint(dt)
	g.updateMinimapInput()
	g.updateEmotes(dt)
	g.seenNow = false
	g.updateMurder(dt)
	g.updateSlashers(dt)
	g.updateArrivals(dt)
	g.updateStreaker()
	g.updateBlackout(dt)
	g.updateFootprints(dt) // prints on sand, snow, mud (footprints.go)
	g.secretTick(dt)
	g.updateBubbles(dt)
	g.updateFloats(dt)
	g.updateSuit(dt)
	g.updateVIP(dt)
	g.updateTraps2(dt)
	g.updateDuets(dt)
	g.updateBeams(dt)
	g.eachPlayer(func(pl *Player) { // each player uses his own disguise when he is the wanted one
		if pl.item == "disguise" && g.playerWantedNow(pl) && !pl.dead {
			g.useDisguise()
		}
	})
	if len(g.dbgWith) > 0 && g.time > 0.3 {
		var pool []*Guest
		for _, gu := range g.guests {
			if gu.perf == "" && !gu.out() {
				pool = append(pool, gu)
			}
		}
		for i, sl := range g.dbgWith {
			if f := presetByName(sl); f != nil && i < len(pool) {
				spec := f(g.rng)
				gu := pool[i]
				gu.famousOK = true
				gu.cs, gu.name = newCharSet(&spec), nameFromSpec(&spec)
				gu.quirkInit, gu.gaitInit = false, false
				gu.pos = g.lvl.NearestFree(Vec{g.p.pos.X + float64(i)*30 - 60, g.p.pos.Y - 70})
				gu.state, gu.path = gIdle, nil
			}
		}
		g.dbgWith = nil
	}
	if g.dbgRoad && g.time > 0.2 {
		g.dbgRoad = false
		d := g.lvl.Def
		d.Roads = []RoadSeg{{R(40, 120, 700, 36), roadRight}, {R(40, 200, 700, 36), roadLeft}, {R(800, 30, 36, 460), roadDown}, {R(860, 30, 36, 460), roadUp}}
		d.roadsDrawn = false
		g.renderBG()
		// debug: -roadcars a,b,c parks those vehicles along the test roads (slowly), to look at them
		for i, n := range g.dbgRoadCars {
			si := i % 4
			s := d.Roads[si]
			start, lat := roadStart(s)
			slot := float64(i / 4)
			g.roadCars = append(g.roadCars, &RoadCar{seg: si, pos: start + roadSign(s)*(60+slot*120+float64(si)*20), lat: lat, v0: 1, name: n, length: 40})
		}
	}
	if g.dbgTrap {
		g.trapDemoTick()
	}
	if g.dbgPoison && g.time > 0.5 {
		g.dbgPoison = false
		g.p.item, g.p.doses = "poison", poisonDoses
		for i, gu := range g.guests {
			if i >= 4 {
				break
			}
			gu.pos = g.lvl.NearestFree(Vec{g.p.pos.X + float64(i-1)*26, g.p.pos.Y + 34})
			gu.state, gu.path, gu.timer, gu.perf = gIdle, nil, 3, ""
			gu.poison = 7 + float64(i)*5
			gu.poisonT0, gu.poisonTick = gu.poison, 1
		}
	}
	if g.dbgAlert { // -alertnow: pinned witnesses (2 for player 1, 1 for player 2) so the "!" stays on
		for i, gu := range g.guests {
			ps := g.players()
			if i >= 3 || len(ps) == 0 {
				break
			}
			p := ps[0]
			if i == 2 && len(ps) > 1 {
				p = ps[1]
			} else if i == 2 {
				break
			}
			gu.sawKiller, gu.sawP, gu.warned = true, p, false
			gu.state = gCall
		}
	}
	if g.dbgSnake && g.time > 0.3 {
		g.scatterSnake()
		g.dbgSnake = false
		g.dbgSnakeOn = true
	}
	g.debugZapStep()
	g.lastGuestsDemo() // -lastguests N (lastguests.go)
	if g.dbgCheer && g.time > 1 {
		g.dbgCheer, g.cheerT = false, 0
	}
	g.updateCheer()
	g.updateTrains(dt)
	g.updateExhibitionists()
	g.updateEvents(1.0 / 60)
	for _, gu := range g.guests {
		restore := g.focus(gu.pos) // two players: each guest reacts to the one nearest to him
		g.updateGuest(gu, dt)
		restore()
	}
	for _, gd := range g.guards {
		restore := g.focus(gd.pos)
		g.updateGuard(gd, dt)
		restore()
	}
	for _, c := range append([]*Cop(nil), g.cops...) {
		g.updateCop(c, dt)
	}
	if g.mode != ModePlay {
		return
	}
	for _, b := range g.bodies {
		if b.fallT > 0 {
			b.fallT -= dt
		}
	}
	if g.titleT > 0 {
		g.titleT -= dt
	}
	g.updateProps(dt)
	g.uniqueFamous() // nobody famous twice in the same party, whoever added them
	g.enforceGuestCap()
	g.checkTrophies()
	g.maybeAmbulance()
	g.updateStars(dt)
	g.updateSuspect()
	g.updateCameras(dt) // security cameras: sweep and LED (camera.go)
	g.updateStealth(dt) // STEALTH mode: who sees the player, alert phases (stealth.go)
	g.updateJail(dt)
	if !g.seenNow && g.wanted > 0 {
		g.wanted -= dt
		if !g.lawActive() {
			g.wanted = 0 // nobody is left on the scene looking for you: the cuffs come off
		}
		if g.wanted <= 0 {
			g.clearSuspects() // two players: the search is over for both
		}
	}
	g.updateVehicles(dt)
	g.updateFire(dt)
	g.updatePyroWorld(dt)
	g.updateTech(dt)
	g.updateKidArrivals()
	if g.dbgCops && g.time > 4 {
		g.dbgCops = false
		g.raisePolice(g.p.pos, true)
	}
	if g.dbgCamKill && g.time > 3 {
		g.dbgCamKill = false
		g.debugCamKill()
	}
	if g.dbgRing > 0 && g.time > 2 {
		for _, gu := range g.guests {
			if !gu.out() {
				gu.state, gu.timer, gu.tech = gCall, 30, g.dbgRing == 2
				break
			}
		}
		g.dbgRing = 0
	}
	g.updateAmbient(dt)
	g.updateRoads(dt)
	g.updateAnimals(dt)
	g.updateJurassic(dt)
	g.watchAllStuck(dt)
	g.updateParticles(dt)
	if !g.camFixed {
		t := g.camTarget()
		g.cam.X += (t.X - g.cam.X) * 0.12
		g.cam.Y += (t.Y - g.cam.Y) * 0.12
	}
}

func (g *Game) burst(pos Vec, n int, speed float64, col color.RGBA, life, size, grav float64) {
	for i := 0; i < n; i++ {
		a := g.rng.Float64() * 2 * math.Pi
		s := speed * (0.3 + g.rng.Float64()*0.7)
		l := life * (0.6 + g.rng.Float64()*0.4)
		g.parts = append(g.parts, Particle{pos: pos, vel: Vec{math.Cos(a) * s, math.Sin(a)*s - speed*0.3}, life: l, max: l, col: col, size: size, grav: grav})
	}
}

func (g *Game) updateParticles(dt float64) {
	out := g.parts[:0]
	for _, p := range g.parts {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.vel.Y += p.grav * dt
		p.pos.X += p.vel.X * dt
		p.pos.Y += p.vel.Y * dt
		out = append(out, p)
	}
	g.parts = out
}

// --- player -----------------------------------------------------------------

func (g *Game) updatePlayer(dt float64) {
	p := g.p
	if p.dead {
		p.moving = false
		return
	}
	if p.jail.held { // in the hands of the police (two players): he cannot do anything
		p.moving = false
		return
	}
	if p.cooldown > 0 {
		p.cooldown -= dt
	}
	if p.stabT > 0 {
		p.stabT -= dt
	}
	if p.knockT > 0 {
		p.knockT -= dt
		p.moving = false
		return
	}
	if p.stunT > 0 { // stunned by the VIP guards: stands there seeing stars
		p.stunT -= dt
		p.moving = false
		return
	}
	if actInteract.just() || (inputP == 0 && g.botPressE(dt)) {
		g.interact()
	}
	if p.hidden {
		p.moving = false
		return
	}
	var dx, dy float64
	if actLeft.held() {
		dx--
	}
	if actRight.held() {
		dx++
	}
	if actUp.held() {
		dy--
	}
	if actDown.held() {
		dy++
	}
	if dx == 0 && dy == 0 {
		dx, dy = padAxis()
	}
	if g.walkBot {
		if inputP == 0 {
			dx, dy = g.botMove(dt)
		} else {
			dx, dy = g.botMove2(dt) // two players: the second one walks about at random too
		}
	}
	p.moving = dx != 0 || dy != 0
	if inputP == 0 { // the event key and the legend powers are for the whole party: either player may use them
		if actEvent.justEither() {
			g.manualEvent()
		}
		g.updatePowers()
	}
	whistled := actDance.just() && g.stealthWhistle(p) // STEALTH 2P: the dance key makes the decoy noise (stealth2p.go)
	if actDance.just() && !whistled && p.carrying == nil && p.danceT <= 0 {
		p.danceT = 4.5
		g.say("¡A bailar!")
		g.stat("dance", 1)
		g.danceRoll()
	}
	if p.danceT > 0 {
		if p.moving {
			p.danceT = 0
		} else {
			p.danceT -= dt
			p.anim += dt * 5
			return
		}
	}
	if g.lvl.SolidP(p.pos.X, p.pos.Y) {
		// somehow inside a solid (a teleport, a prop that appeared): never leave the player stuck
		g.lvl.unstick(p)
	}
	speed := 46.0
	p.running = false
	running := actRun.held()
	if p.moving && running && p.stamina > 0 {
		speed = 74
		p.running = true
		p.stamina -= dt
	} else if p.stamina < g.perkStamina {
		p.stamina += dt * 0.9
	}
	speed *= g.perkSpeed
	speed *= p.speedMul() // the ball and chain of the one who escaped (twojail.go)
	if g.dbg.speed {
		speed *= 2
	}
	if p.carrying != nil {
		speed *= 0.75
	}
	speed *= g.stealthSpeedMul(p) // the cardboard box is slow (stealth.go)
	speed *= g.coopSpeedMul(p)    // two players: the partner on the shoulders slows him down
	if g.lvl.InPool(p.pos) {
		speed *= 0.6
		p.running = false
	}
	for _, r := range g.lvl.Def.Slow {
		if r.Has(p.pos.X, p.pos.Y) {
			speed *= 0.55
			break
		}
	}
	if p.moving {
		l := math.Hypot(dx, dy)
		dx, dy = dx/l, dy/l
		if math.Abs(dx) > math.Abs(dy) {
			if dx < 0 {
				p.dir = DirLeft
			} else {
				p.dir = DirRight
			}
		} else if dy < 0 {
			p.dir = DirUp
		} else {
			p.dir = DirDown
		}
		before := p.pos
		nx := p.pos.X + dx*speed*paceK*dt
		if !g.lvl.SolidP(nx, p.pos.Y) {
			p.pos.X = nx
		}
		ny := p.pos.Y + dy*speed*paceK*dt
		if !g.lvl.SolidP(p.pos.X, ny) {
			p.pos.Y = ny
		}
		p.anim += dt * stridePhase(speed)
		g.watchStuck(before, dx, dy, dt)
	}
	if actAttack.just() && !g.stealthNoAttack() {
		g.stab()
	}
}

// danceRoll: nearby people react to the player dancing.
func (g *Game) danceRoll() {
	p := g.p
	// dancing again and again in the same place ends up annoying somebody
	if dist(p.pos, p.danceSpot) < 70 && g.time-p.danceLast < 90 && p.danceCount > 0 {
		p.danceCount++
	} else {
		p.danceCount = 1
	}
	p.danceSpot, p.danceLast = p.pos, g.time
	if p.danceCount >= 4 {
		var best *Guest
		for _, gu := range g.guests {
			if !gu.out() && gu.perf == "" && gu.state != gPunch && dist(gu.pos, p.pos) < 170 && (best == nil || dist(gu.pos, p.pos) < dist(best.pos, p.pos)) {
				best = gu
			}
		}
		if best != nil {
			p.danceCount = 0
			best.icon, best.iconT = g.line(guestCategory(best), "dance_annoyed"), 2
			best.state, best.timer = gPunch, 12
		}
	}
	for _, gu := range g.guests {
		if gu.out() || dist(gu.pos, p.pos) > 95 || !g.lvl.LOS(gu.pos, p.pos) {
			continue
		}
		switch gu.state {
		case gIdle, gGoto, gDance, gDrink, gLounge, gToilet, gChatGo, gSwim:
		default:
			continue
		}
		r := g.rng.Float64()
		switch {
		case r < 0.40:
			side := 1.0
			if g.rng.Intn(2) == 0 {
				side = -1
			}
			t := Vec{p.pos.X + side*(22+g.rng.Float64()*10), p.pos.Y + (g.rng.Float64()-0.5)*14}
			gu.path = g.lvl.PathG(gu.pos, t)
			gu.state, gu.danceEnd = gDanceWith, g.time+p.danceT+2+g.rng.Float64()*3
			gu.icon, gu.iconT = g.line(guestCategory(gu), "dance_join"), 1.5
		case r < 0.55:
		case r < 0.97:
			// somebody who would rather not watch walks off to another zone
			gu.icon, gu.iconT = g.line(guestCategory(gu), "dance_leave"), 1.6
			far := g.lvl.RandomRoam(g.rng)
			for k := 0; k < 6; k++ {
				if c := g.lvl.RandomRoam(g.rng); dist(c, p.pos) > dist(far, p.pos) {
					far = c
				}
			}
			gu.path = g.lvl.PathG(gu.pos, far)
			gu.state, gu.next, gu.dur = gGoto, gIdle, 1
		default:
			gu.icon, gu.iconT = g.line(guestCategory(gu), "dance_annoyed"), 2
			gu.state, gu.timer = gPunch, 12
		}
	}
}

func (g *Game) stab() {
	p := g.p
	if p.cooldown > 0 || p.hidden || p.dead || p.knockT > 0 || p.stunT > 0 {
		return
	}
	if p.carrying != nil {
		g.say("Suelta el cuerpo primero (E)")
		return
	}
	if g.coopBlockStab(p) {
		return
	}
	p.cooldown = 0.45 * g.perkCool
	p.stabT = 0.18
	var bestG *Guest
	var bestD *Guard
	bd := 26.0
	for _, gu := range g.guests {
		if gu.out() {
			continue
		}
		if d := dist(p.pos, gu.pos); d < bd {
			bestG, bestD, bd = gu, nil, d
		}
	}
	for _, gd := range g.guards {
		if gd.dead {
			continue
		}
		if d := dist(p.pos, gd.pos); d < bd {
			bestG, bestD, bd = nil, gd, d
		}
	}
	if o := g.playerInReach(p, math.Min(bd, 26)); o != nil { // the other player: a punch, never a kill
		g.hitPlayer(p, o)
		return
	}
	if g.hitAnySlasher(p, math.Min(bd, 28)) {
		return
	}
	if g.canKillCops() {
		if c := g.nearestCop(p.pos, math.Min(bd, 28)); c != nil {
			g.stabCop(c)
			return
		}
	}
	if an, ad := g.nearAnimal(p.pos, math.Min(bd, 24)); an != nil && ad < bd {
		g.killAnimal(an)
		return
	}
	switch {
	case bestG != nil && (bestG.state == gZombie || bestG.armed):
		// putting down a zombie is not a crime: no witnesses, no heat
		g.credit, g.noHeat = true, true
		g.killGuest(bestG, false)
		g.credit, g.noHeat = false, false
	case bestG != nil:
		g.killGuest(bestG, true)
	case bestD != nil:
		g.killGuard(bestD)
	}
}

func (g *Game) spawnBody(pos Vec, cs *CharSet, dir Dir) {
	nb := &Body{pos: pos, cs: cs, dir: dir, fallT: 0.45, byPlayer: (g.killByPlayer || g.credit) && !g.noHeat, born: g.time}
	g.bodies = append(g.bodies, nb)
	if elecDeathNext {
		elecDeathNext = false
		g.makeSkeletonBody(nb)
		return
	}
	g.decals = append(g.decals, Decal{pos: pos, kind: 0, r: 9 + g.rng.Float64()*4})
	g.burst(Vec{pos.X, pos.Y - 14}, 18, 80, rgb(0xb01818), 0.7, 2, 220)
}

func (g *Game) killGuest(gu *Guest, byPlayer bool) {
	if gu.out() {
		return
	}
	cause := g.killCause
	g.killCause = kcNone
	if byPlayer || g.credit { // a kill of the player (his hands, his poison, his traps): the statistics
		g.countKill(cause, true)
	}
	gu.dead = true
	gu.icon = ""
	if gu.py != nil { // the arsonist: he was not part of the guest count, now he is a body like any other
		g.total++
		if byPlayer {
			g.stat("pyros", 1)
		}
	}
	if gu.perf == "dj" && !g.djDead {
		g.djDead = true
		g.songIdx = g.rng.Intn(5)
		g.nextSong = g.time + 55 + g.rng.Float64()*30
		g.sayNews("dj_dead", nil)
		g.say("Ve a la cabina del DJ (E) para cambiar la canción")
	}
	g.killByPlayer = byPlayer
	if byPlayer {
		g.stat("kills", 1)
		if gu.cs != nil && gu.cs.Spec != nil && isKidSlug(gu.cs.Spec.Name) {
			g.stat("kill_"+gu.cs.Spec.Name, 1)
		}
		if gu.cs != nil && gu.cs.Spec != nil {
			g.albumMark(albumKeyOf(gu.cs.Spec.Name), albumKilled)
		}
	}
	g.spawnBody(gu.pos, gu.cs, gu.dir)
	g.bodies[len(g.bodies)-1].label = guestLabel(gu)
	g.killByPlayer = false
	if byPlayer && gu.vipCard {
		g.vipTakeCard(gu) // his VIP invitation is yours (vip.go)
	}
	if gu.partner != nil && gu.partner.partner == gu {
		gu.partner.partner = nil
		if gu.partner.state == gChat || gu.partner.state == gKiss || gu.partner.state == gWaitP {
			gu.partner.state, gu.partner.timer = gIdle, 1
		}
	}
	if byPlayer {
		g.snd.Play("stab")
		g.kills++
		g.points += g.killVal(30)
		g.witness()
		g.sayNews("kill_player", map[string]string{"victim": guestLabel(gu)})
	} else {
		g.snd.Play("thud")
		if g.credit {
			g.award(gu.pos, g.killVal(50), "")
		}
	}
	g.checkDone()
}

func (g *Game) killGuard(gd *Guard) {
	if gd.dead {
		return
	}
	if g.vipTough(gd) { // a tough VIP takes several blows (vip.go)
		return
	}
	gd.dead = true
	gd.icon = ""
	g.stat("guards", 1)
	g.killByPlayer = true
	g.spawnBody(gd.pos, gd.cs, gd.dir)
	g.bodies[len(g.bodies)-1].label = "Guardia " + g.personalName(gd, gd.cs)
	g.killByPlayer = false
	g.snd.Play("stab")
	g.kills++
	g.countKill(kcKnife, false)
	g.points += g.killVal(50)
	g.witness()
	g.vipOnKill(gd)
	g.checkDone()
}

func (g *Game) checkDone() {
	if g.aliveCount() == 0 && g.mode == ModePlay && g.stealth {
		g.stealthWin() // STEALTH mode: its own ending, nothing is banked or unlocked
		return
	}
	if g.aliveCount() == 0 && g.mode == ModePlay {
		if !g.slowBegin() { // the very last kill of the party: a moment of slow motion first (slowmo.go)
			g.completeLevel()
		}
	}
}

// completeLevel: the party is over (everybody is out): bonuses, bank, unlock and the next screen.
func (g *Game) completeLevel() {
	if g.mode == ModePlay {
		g.endTime = g.time
		g.snd.Play("win")
		g.finishScore() // completion bonuses and the x5 of the RETO (levelscore.go), before banking
		g.bankPoints()
		g.unlockNext()
		g.stat("wins", 1)
		if !g.policeCalled {
			g.stat("nopolice", 1)
		}
		if g.time < 180 {
			g.stat("fast", 1)
		}
		if g.time > 600 {
			g.stat("long", 1)
		}
		if g.levelIdx < campaignN && g.progress.Stats[fmt.Sprintf("won_%d", g.levelIdx)] == 0 {
			g.stat(fmt.Sprintf("won_%d", g.levelIdx), 1)
			g.stat("distinct", 1)
		}
		switch {
		case g.levelIdx == specialJurassic: // the last party is over: the secret ending (the first time)
			if g.seen("seen_end2") && debugSnap == nil {
				g.mode = ModeLevelDone // played again from the list: no new ending, it can be watched in SCENES
			} else {
				g.startEndingPre(true)
			}
		case campaignN > 0 && g.levelIdx == campaignN-1: // the 21st party closes the story
			if g.allTrophies() {
				g.startInvitePre() // ...unless you own every trophy: one more party, on the island
			} else {
				g.startEndingPre(false)
			}
		case g.levelIdx+1 < len(levelDefs):
			g.mode = ModeLevelDone
		default:
			g.mode = ModeFinal
		}
	}
}

func (g *Game) lose(reason string) {
	if g.mode != ModePlay || g.invulnerable() {
		return
	}
	g.mode = ModeLose
	g.stat("deaths", 1)
	g.bankPoints()
	g.loseReason = reason
	g.endTime = g.time
	g.endAt = g.time
	g.snd.Play("lose")
	g.shake = 0.4
}

func (g *Game) die(reason string) {
	if g.invulnerable() {
		return
	}
	g.p.dead = true
	g.p.deathWhy = reason
	g.p.hidden = false
	g.p.carrying = nil
	g.burst(Vec{g.p.pos.X, g.p.pos.Y - 14}, 22, 90, rgb(0xb01818), 0.8, 2, 220)
	g.decals = append(g.decals, Decal{pos: g.p.pos, kind: 0, r: 11})
	if g.is2P() && !g.allPlayersDead() { // two players: the other one goes on
		g.say(g.playerName(g.p) + " ha caído... " + g.playerName(g.otherPlayer(g.p)) + " sigue")
		return
	}
	g.lose(reason)
}

func (g *Game) canWatch(gu *Guest) bool {
	switch gu.state {
	case gSleep, gKiss, gVomit, gAlarm, gToPhone, gCall, gFlee, gRun, gScared, gReport:
		return false
	}
	return !gu.out()
}

// witness: whoever has the player in front of them at the moment of the kill notices.
func (g *Game) witness() {
	p := g.p
	p.jail.killT = g.time + 0.001 // the last one to kill (twojail.go: who the police want if nobody saw it)
	var seers []*Guest
	for _, gu := range g.guests {
		if g.canWatch(gu) && g.sees(gu.pos, gu.dir, p.pos, guestRange, guestDot) {
			gu.sawKiller = true
			gu.sawPos = p.pos
			gu.sawP, gu.warned = p, false
			seers = append(seers, gu)
		}
	}
	direct := g.copSeesAt(p.pos) // an officer saw it himself: the police know at once
	saw := len(seers) > 0 || direct
	for _, gd := range g.guards {
		if !gd.dead && g.sees(gd.pos, gd.dir, p.pos, guardRange*1.4, guardDot) {
			saw = true
		}
	}
	if saw {
		g.run.seen[g.playerIndex(p)]++      // statistics: somebody saw this kill
		g.run.cleanAt[g.playerIndex(p)] = 0 // ...so it is not a clean kill
	}
	if saw {
		// the last one seen killing is the one they want; only an officer's sighting puts the cuffs
		// on now, a guest's sighting is an ALERT until he warns the police (suspect.go)
		g.suspectSeenBy(suspPlayer, nil, direct)
	}
	if len(seers) > 0 && len(g.bodies) > 0 {
		last := g.bodies[len(g.bodies)-1]
		for _, gu := range seers {
			g.knowBody(gu, last)
		}
	}
	if len(seers) > 0 {
		g.killPos = p.pos
		sort.Slice(seers, func(i, j int) bool { return dist(seers[i].pos, p.pos) < dist(seers[j].pos, p.pos) })
		for i, gu := range seers {
			if i == 0 && g.callerAvailable() {
				g.becomeCaller(gu)
			} else {
				g.alarm(gu, "!?")
			}
		}
	}
	for _, gd := range g.guards {
		if !gd.dead && g.sees(gd.pos, gd.dir, p.pos, guardRange*1.4, guardDot) {
			gd.sight = 99
		}
	}
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave && g.sees(c.pos, c.dir, p.pos, copRange, copDot) {
			g.wanted = math.Max(g.wanted, 45)
		}
	}
	g.cameraWitness() // a security camera with the killer in its cone is a direct witness (camera.go)
}

func (g *Game) alarm(gu *Guest, txt string) {
	if kidRiding(gu) {
		return // busy with his tricks in the bowl: he warns nobody and runs from nobody
	}
	if gu.perf != "" {
		gu.icon, gu.iconT = txt, 1.4
		return
	}
	if gu.state == gAlarm || gu.state == gToPhone || gu.state == gCall || gu.state == gFlee || gu.state == gScared || gu.state == gReport {
		return
	}
	if gu.partner != nil && gu.partner.partner == gu {
		gu.partner.partner = nil
		if gu.partner.state == gChat || gu.partner.state == gKiss || gu.partner.state == gWaitP {
			gu.partner.state, gu.partner.timer = gIdle, 1
		}
	}
	gu.partner = nil
	gu.state = gAlarm
	gu.timer = 1.0 + g.rng.Float64()*1.3
	gu.icon, gu.iconT = txt, 1.4
	gu.path = nil
	// no beep here: with a crowd (zombies, the T-Rex, a fight) the guests warn each other non stop
}

func (g *Game) callerAvailable() bool {
	// once the police have been called (even if still on their way) nobody calls again
	if g.policeCalled || g.pendingCar >= 0 || g.pendingUnit >= 0 {
		return false
	}
	if gu := g.callerGuest; gu != nil && !gu.out() {
		return false
	}
	if gd := g.callerGuard; gd != nil && !gd.dead {
		return false
	}
	return true
}

func (g *Game) becomeCaller(gu *Guest) {
	if gu.perf != "" {
		g.alarm(gu, "!?")
		return
	}
	g.callerGuest = gu
	g.claimBodies(gu)
	g.alarm(gu, "!?")
}

// reporterActive: this guest is on their way to tell the police.
func (g *Game) reporterActive(gu *Guest) bool {
	if gu == nil || gu.out() {
		return false
	}
	if gu.tech {
		return false
	}
	switch gu.state {
	case gReport, gToPhone, gCall:
		return true
	}
	return gu == g.callerGuest
}

// reporters counts guests currently going to warn the police (max 2 at a time).
func (g *Game) reporters() int {
	n := 0
	for _, gu := range g.guests {
		if g.reporterActive(gu) {
			n++
		}
	}
	return n
}

// claimBodies: the bodies this guest knows are now "being reported" by them; nobody else needs to go.
func (g *Game) claimBodies(gu *Guest) {
	for _, b := range g.bodies {
		if !b.reported && !b.carried && g.knows(gu, b) && (b.informer == nil || !g.reporterActive(b.informer)) {
			b.informer = gu
		}
	}
}

// --- interaction -------------------------------------------------------------

type candidate struct {
	d  float64
	fn func()
}

func (g *Game) interact() {
	p := g.p
	if g.pickupInteract(p) { // poison and clothes can ALWAYS be picked up (items.go), before anything else
		return
	}
	if g.coopInteract(p) { // two players: pick up / put down the partner, distract a cop, high five
		return
	}
	if g.stealthInteract(p) { // STEALTH mode: put on / take off the cardboard box
		return
	}
	if p.hidden {
		p.hidden = false
		pr := p.hideProp
		p.pos = g.lvl.NearestFree(Vec{pr.X, pr.Y + 16})
		return
	}
	var cs []candidate
	for _, pr := range g.lvl.Props {
		pr := pr
		d := distRect(p.pos, pr.Foot)
		if d > 16 {
			continue
		}
		k := pr.K
		switch {
		case k.Hide:
			if p.carrying != nil {
				cs = append(cs, candidate{d, func() { g.dispose(false) }})
			} else {
				cs = append(cs, candidate{d, func() {
					p.hidden, p.hideProp = true, pr
					g.stat("hide", 1)
					p.pos = Vec{pr.X, pr.Y - 4}
					g.say("Escondido. Pulsa E para salir")
				}})
			}
		case k.Phone && !pr.used && g.canCallAmbulance(p):
			cs = append(cs, candidate{d - 6, func() { g.callAmbulance(p, pr) }})
		case k.Phone && !pr.used && p.carrying == nil:
			cs = append(cs, candidate{d, func() {
				pr.used = true
				g.stat("cuts", 1)
				g.snd.Play("spark")
				g.say("Has cortado el cable del teléfono")
				g.award(Vec{pr.X, pr.Y}, 25, "cable")
			}})
		case k.Poison && p.carrying == nil:
			cs = append(cs, candidate{d, func() {
				if pr.armT > 0 {
					g.say("La bebida ya está envenenada")
					return
				}
				if p.item != "poison" {
					g.say("Necesitas un frasco de veneno: búscalo por la fiesta")
					return
				}
				g.useDoses(poisonBowlDoses)
				g.award(Vec{pr.X, pr.Y}, 15, "veneno")
				pr.armT = 40
				g.coopOwn(pr, p)
				g.snd.Play("splash")
				g.say("Has envenenado el ponche")
				g.say(dosesText(p.doses))
			}})
		case (pr.Name == "closet" || pr.Name == "wardrobe") && p.carrying == nil:
			cs = append(cs, candidate{d, func() {
				if g.wanted > 0 {
					g.shakeOffPolice(p)
					g.say("Te has cambiado de ropa. Pierden tu pista")
					g.snd.Play("spark")
				} else {
					g.say("Un armario: puedes cambiarte aquí si te buscan")
				}
			}})
		case k.Trap != "" && !pr.used && pr.armT == 0 && !pr.armed && !pr.armWait && p.carrying == nil:
			cs = append(cs, candidate{d, func() {
				g.armTrap(pr, p)
				g.say("Trampa activada. ¡Aléjate!")
			}})
		}
	}
	// the DJ booth: once the DJ is dead anybody can change the song
	if p.carrying == nil {
		var booth []Vec
		for _, pr := range g.lvl.Props {
			if pr.Name == "dj" {
				booth = append(booth, Vec{pr.X, pr.Y})
			}
		}
		for _, pf := range g.lvl.Def.Performers {
			if pf.Role == "dj" {
				booth = append(booth, pf.Pos)
			}
		}
		for _, b := range booth {
			if d := dist(p.pos, b); d < 44 {
				cs = append(cs, candidate{d - 6, func() {
					if !g.djDead {
						g.say("El DJ no te deja tocar sus discos")
						return
					}
					g.skipSong = true
					g.snd.Play("spark")
					g.say("Cambias la canción en la cabina del DJ")
				}})
				break
			}
		}
	}
	if p.carrying == nil {
		for _, w := range g.lvl.Def.Windows {
			if d := distRect(p.pos, w); d < 14 {
				w := w
				cs = append(cs, candidate{d - 2, func() { g.jumpWindow(w) }})
			}
		}
	}
	if p.carrying != nil {
		for _, pool := range g.lvl.Pools {
			if d := distRect(p.pos, pool); d < 14 {
				cs = append(cs, candidate{d, func() { g.dispose(true) }})
			}
		}
	} else {
		for _, b := range g.bodies {
			if d := dist(p.pos, b.pos); d < 22 && !b.carried && b.fallT <= 0 {
				b := b
				cs = append(cs, candidate{d - 4, func() {
					b.carried = true
					p.carrying = b
					g.say("Cargas el cuerpo. Escóndelo en un contenedor, arbusto, armario o la piscina")
				}})
			}
		}
	}
	if gu := g.poisonTarget(); gu != nil {
		gu := gu
		cs = append(cs, candidate{dist(gu.pos, p.pos) - 3, func() { g.offerPoison(gu) }})
	}
	if len(cs) == 0 {
		if p.carrying != nil {
			p.carrying.carried = false
			p.carrying.pos = p.pos
			p.carrying = nil
			g.say("Has soltado el cuerpo")
		}
		return
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
	cs[0].fn()
}

func (g *Game) dispose(pool bool) {
	p := g.p
	b := p.carrying
	for i, x := range g.bodies {
		if x == b {
			g.bodies = append(g.bodies[:i], g.bodies[i+1:]...)
			break
		}
	}
	p.carrying = nil
	g.award(p.pos, 20, "cadáver")
	if pool {
		g.snd.Play("splash")
		g.burst(Vec{p.pos.X, p.pos.Y}, 12, 50, rgb(0xa0d8f8), 0.5, 2, 160)
	} else {
		g.snd.Play("thud")
	}
	g.say("Cuerpo escondido")
}

// --- props / traps -----------------------------------------------------------

func (g *Game) updateProps(dt float64) {
	for _, pr := range g.lvl.Props {
		if pr.K.Poison {
			if pr.armT > 0 {
				pr.armT -= dt
			}
			continue
		}
		if pr.armT > 0 || pr.armed || pr.armWait {
			g.updateTrapMine(pr, dt) // loads, then waits like a mine (trapmine.go)
		}
	}
}

func (g *Game) triggerTrap(pr *Prop) {
	pr.used = true
	g.coopTrapFired(pr)
	g.credit = true
	defer func() { g.credit = false }()
	centre := Vec{pr.X, pr.Y - 8}
	switch pr.K.Trap {
	case "gas", "fireworks", "piano":
		g.credit = false
		g.triggerNewTrap(pr)
	case "explode":
		g.snd.Play("boom")
		g.shake = 0.6
		g.flash = 0.25
		g.burst(centre, 40, 150, rgb(0xff8020), 0.8, 3, 40)
		g.burst(centre, 24, 100, rgb(0xffe060), 0.6, 2, 20)
		g.burst(centre, 16, 60, rgb(0x303038), 1.2, 3, -30)
		g.decals = append(g.decals, Decal{pos: Vec{pr.X, pr.Y}, kind: 1, r: 22})
		for _, gu := range g.guests {
			if !gu.out() && dist(gu.pos, centre) < 64 {
				g.killCause = kcBlast
				g.killGuest(gu, false)
				g.stat("traps", 1)
			}
		}
		for _, gd := range g.guards {
			if !gd.dead && dist(gd.pos, centre) < 64 {
				gd.dead = true
				g.countKill(kcBlast, false)
				g.points += g.killVal(50)
				g.spawnBody(gd.pos, gd.cs, gd.dir)
				g.bodies[len(g.bodies)-1].label = "Guardia " + g.personalName(gd, gd.cs)
			}
		}
		g.trapCops(centre, 64)
		g.eachPlayer(func(p *Player) {
			if !p.dead && dist(p.pos, centre) < 58 {
				g.die("Has saltado por los aires con tu propia trampa")
			}
		})
		for _, gu := range g.guests {
			if !gu.out() && dist(gu.pos, centre) < 220 {
				g.alarm(gu, "!?")
			}
		}
		g.checkDone()
	case "electric":
		g.snd.Play("zap")
		g.flash = 0.2
		for _, pool := range g.lvl.Pools {
			for i := 0; i < 40; i++ {
				g.burst(Vec{pool.X + g.rng.Float64()*pool.W, pool.Y + g.rng.Float64()*pool.H}, 1, 40, rgb(0x90e0ff), 0.4, 2, 0)
			}
		}
		for _, gu := range g.guests {
			if !gu.out() && g.lvl.InPool(gu.pos) {
				g.burst(Vec{gu.pos.X, gu.pos.Y - 12}, 10, 70, rgb(0xd0f4ff), 0.4, 2, 0)
				g.killCause = kcTrap
				g.killGuestElec(gu, false)
				g.sayNews("electrocuted", map[string]string{"victim": guestLabel(gu)})
				g.stat("traps", 1)
			}
		}
		g.checkDone()
	}
}

// --- guests ------------------------------------------------------------------

func follow(pos *Vec, dir *Dir, path *[]Vec, speed, dt float64) (moved, arrived bool) {
	if len(*path) == 0 {
		return false, true
	}
	t := (*path)[0]
	dx, dy := t.X-pos.X, t.Y-pos.Y
	d := math.Hypot(dx, dy)
	step := speed * dt
	if d <= step {
		*pos = t
		*path = (*path)[1:]
		return true, len(*path) == 0
	}
	pos.X += dx / d * step
	pos.Y += dy / d * step
	if math.Abs(dx) > math.Abs(dy)*1.3 {
		if dx < 0 {
			*dir = DirLeft
		} else {
			*dir = DirRight
		}
	} else if math.Abs(dy) > math.Abs(dx)*1.3 {
		if dy < 0 {
			*dir = DirUp
		} else {
			*dir = DirDown
		}
	}
	return true, false
}

func faceTo(from, to Vec) Dir {
	dx, dy := to.X-from.X, to.Y-from.Y
	if math.Abs(dx) > math.Abs(dy) {
		if dx > 0 {
			return DirRight
		}
		return DirLeft
	}
	if dy > 0 {
		return DirDown
	}
	return DirUp
}

func (g *Game) poiRandom(kind string) (Vec, bool) {
	l := g.lvl.POIs[kind]
	if len(l) == 0 {
		return Vec{}, false
	}
	return l[g.rng.Intn(len(l))], true
}

func (g *Game) findPartner(gu *Guest) *Guest {
	var best *Guest
	bd := 260.0
	for _, o := range g.guests {
		if o == gu || o.out() || (o.state != gIdle && o.state != gLounge) {
			continue
		}
		d := dist(o.pos, gu.pos) + g.rng.Float64()*80
		if d < bd {
			best, bd = o, d
		}
	}
	return best
}

func (g *Game) guestPick(gu *Guest) {
	d := g.lvl.Def
	if l := gu.leader; l != nil && !l.out() && gu.need < 1 && g.rng.Float64() < 0.8 {
		// the group sticks together behind its leader
		gu.path = g.lvl.PathG(gu.pos, g.lvl.NearestFree(Vec{l.pos.X + g.rng.Float64()*50 - 25, l.pos.Y + g.rng.Float64()*30 - 10}))
		gu.state, gu.next, gu.dur = gGoto, gIdle, 2+g.rng.Float64()*2
		return
	}
	if gu.need >= 1 {
		// "I really have to go": run to the toilet, or relieve themselves where
		// they stand when the venue has none
		gu.need = 0
		gu.hurry, gu.hurryT = true, 0
		gu.icon, gu.iconT = []string{"WC!", "¡NO AGUANTO!", "¡BAÑO!"}[g.rng.Intn(3)], 2.5
		if target, ok := g.poiRandom("toilet"); ok {
			gu.path = g.lvl.PathG(gu.pos, target)
			gu.state, gu.next, gu.dur = gGoto, gToilet, 6+g.rng.Float64()*4
			return
		}
		g.accident(gu, g.rng.Float64() < 0.25)
		return
	}
	if gu.drunk >= 3 {
		gu.drunk = 0
		if g.rng.Float64() < 0.5 {
			gu.state, gu.timer = gVomit, 2.4
			gu.icon, gu.iconT = "BLEH", 2.4
		} else {
			gu.state, gu.timer = gSleep, 16+g.rng.Float64()*14
		}
		return
	}
	r := g.rng.Float64()
	var target Vec
	next := gIdle
	dur := 2 + g.rng.Float64()*5
	ok := false
	switch {
	case r < 0.28 && len(d.Dance) > 0:
		target, ok = g.lvl.RandomIn(g.rng, d.Dance[g.rng.Intn(len(d.Dance))]), true
		next, dur = gDance, 7+g.rng.Float64()*10
	case r < 0.42:
		if target, ok = g.poiRandom("drink"); ok {
			next, dur = gDrink, 3
		}
	case r < 0.58 || (r < 0.62):
		if p := g.findPartner(gu); p != nil {
			kind := gChat
			dur = 7 + g.rng.Float64()*6
			if r >= 0.58 {
				kind, dur = gKiss, 6+g.rng.Float64()*4
			}
			gu.partner, gu.role = p, 0
			p.partner, p.role = gu, 1
			p.state, p.timer, p.path = gWaitP, 30, nil
			gu.state, gu.next, gu.dur, gu.timer, gu.repath = gChatGo, kind, dur, 30, 0
			return
		}
	case r < 0.67:
		if target, ok = g.poiRandom("toilet"); ok {
			next, dur = gToilet, 5
		}
	case r < 0.74:
		if target, ok = g.poiRandom("lounge"); ok {
			next, dur = gLounge, 8
		}
	case r < 0.84 && d.Swim && len(g.lvl.Pools) > 0:
		target, ok = g.lvl.RandomIn(g.rng, g.lvl.Pools[g.rng.Intn(len(g.lvl.Pools))].Grow(-8)), true
		next, dur = gSwim, 8+g.rng.Float64()*12
	case d.Highway && r >= 0.62 && r < 0.70:
		// the motorway is the dance floor
		target, ok = g.lvl.RandomIn(g.rng, d.Road.Grow(-14)), true
		next, dur = gDance, 6+g.rng.Float64()*8
	case r >= 0.90:
		g.guestFun(gu)
		return
	case r >= 0.76 && r < 0.80 && d.Police == "car":
		// cross the street to the other pavement; most people look first
		cy := math.Min(d.CarFrom.Y+g.laneOff()+10, d.H-12)
		if g.rng.Float64() < 0.8 && g.trafficNear(Vec{gu.pos.X, cy}) {
			break
		}
		if path := g.lvl.PathG(gu.pos, Vec{gu.pos.X + g.rng.Float64()*80 - 40, cy}); path != nil {
			gu.path = path
			gu.state, gu.next, gu.dur = gGoto, gIdle, 3
			return
		}
	case r >= 0.80:
		// some people just stand around, calm, watching the party
		gu.state, gu.timer, gu.path = gIdle, 14+g.rng.Float64()*26, nil
		return
	}
	if !ok {
		target = g.lvl.RandomRoam(g.rng)
	}
	gu.path = g.lvl.PathG(gu.pos, target)
	gu.state, gu.next, gu.dur = gGoto, next, dur
}

const guestSpeed = 21.0

func (g *Game) updateGuest(gu *Guest, dt float64) {
	if !gu.quirkInit {
		gu.quirkInit = true
		if gu.cs != nil && gu.cs.Spec != nil {
			gu.quirk = quirkOf(gu.cs.Spec.Name)
		}
	}
	g.gaitInit(gu)
	if gu.quirk != "" {
		g.updateQuirk(gu, dt)
	}
	g.updateGreeting(gu, dt)
	g.updateStar(gu, dt)
	if gu.out() {
		return
	}
	if g.lvl != nil && len(g.lvl.bowls) > 0 && g.bowlGuest(gu, dt) {
		return
	}
	gu.moving = false
	if gu.vipStun > 0 { // stunned by the VIP guards: stands there seeing stars
		gu.vipStun -= dt
		return
	}
	if gu.py != nil {
		g.updatePyro(gu, dt)
		return
	}
	if gu.bitten > 0 {
		g.venom(gu, dt)
		if gu.out() {
			return
		}
	}
	if gu.state != gToilet && (gu.wcInit || gu.inside != nil || gu.queue != nil) {
		gu.wcInit, gu.inside, gu.queue = false, nil, nil
	}
	if gu.stabT > 0 {
		gu.stabT -= dt
	}
	if gu.iconT > 0 {
		gu.iconT -= dt
		if gu.iconT <= 0 {
			gu.icon = ""
		}
	}
	if gu.poison > 0 && g.updatePoison(gu, dt) {
		return
	}
	g.maybePickWeapon(gu)
	if g.mode == ModePlay && gu.iconT <= 0 && gu.perf == "" && (gu.state == gIdle || gu.state == gDance || gu.state == gLounge || gu.state == gGoto) && g.rng.Float64() < dt*0.003 {
		gu.icon, gu.iconT = g.chatterLine(gu), 2.8
	}
	if gu.chant && gu.icon == "" && g.rng.Float64() < 0.0015 && !gu.out() {
		gu.icon, gu.iconT = g.freshPhrase("chants", chants, g.rng.Intn), 2.5
	}
	if gu.state != gSleep && gu.perf == "" {
		gu.need += dt * (0.006 + 0.004*gu.off/6)
	}
	if g.mode == ModePlay && gu.state != gZombie && gu.state != gLift {
		gu.scanT -= dt
		if gu.scanT <= 0 {
			gu.scanT = 0.3
			g.guestScan(gu)
		}
	}
	switch gu.state {
	case gIdle:
		gu.timer -= dt
		gu.lookT -= dt
		if gu.lookT <= 0 {
			gu.lookT = 1.5 + g.rng.Float64()*3
			if g.rng.Float64() < 0.5 {
				gu.dir = Dir(g.rng.Intn(4))
			}
			if gu.icon == "" && !g.p.hidden && !g.p.dead && dist(gu.pos, g.p.pos) < 40 && g.rng.Float64() < 0.3 {
				gu.icon, gu.iconT = g.line(guestCategory(gu), "greet"), 1.3
			}
		}
		if gu.timer <= 0 {
			g.guestPick(gu)
		}
	case gGoto:
		spd := guestSpeed
		if gu.hurry {
			spd = guestRunSpeed
			gu.hurryT += dt
			if gu.hurryT > 2.5 && g.rng.Float64() < dt*0.05 {
				g.accident(gu, g.rng.Float64() < 0.4)
				break
			}
		}
		if gu.walk(spd, dt) {
			gu.hurry = false
			gu.state, gu.timer = gu.next, gu.dur
			switch gu.state {
			case gDance:
				gu.dir = DirDown
			case gDrink, gToilet, gLounge:
				gu.dir = DirUp
			case gFun:
				gu.dir = DirDown
			}
		}
	case gChatGo:
		p := gu.partner
		if p == nil || p.out() || p.partner != gu || p.state != gWaitP {
			gu.partner = nil
			gu.state, gu.timer = gIdle, 1
			break
		}
		gu.timer -= dt
		gu.repath -= dt
		if gu.repath <= 0 {
			gu.repath = 0.6
			gu.path = g.lvl.PathG(gu.pos, p.pos)
		}
		gu.walk(guestSpeed, dt)
		if dist(gu.pos, p.pos) < 24 || gu.timer <= 0 {
			if gu.timer <= 0 {
				gu.state, gu.timer = gIdle, 1
				p.state, p.timer = gIdle, 1
				break
			}
			gu.state, gu.timer = gu.next, gu.dur
			p.state, p.timer = gu.next, gu.dur
			gu.dir, p.dir = faceTo(gu.pos, p.pos), faceTo(p.pos, gu.pos)
			gu.path = nil
		}
	case gWaitP:
		gu.timer -= dt
		if gu.timer <= 0 || gu.partner == nil || gu.partner.out() {
			gu.state, gu.timer = gIdle, 1
		}
	case gChat:
		gu.timer -= dt
		if turn := int(g.time / 2.2); turn != gu.lastTurn {
			gu.lastTurn = turn
			if turn%2 == gu.role && g.rng.Float64() < 0.3 {
				ph := []string{"...", "HAHA", "BLAH", "REALLY?", "LOL", "NO WAY", "COOL"}
				gu.icon, gu.iconT = ph[g.rng.Intn(len(ph))], 1.8
			}
		}
		if gu.timer <= 0 {
			gu.state, gu.timer = gIdle, 1
		}
	case gKiss:
		gu.timer -= dt
		if gu.role == 0 && gu.icon == "" && g.rng.Float64() < 0.004 {
			gu.icon, gu.iconT = "<3", 1.2
		}
		if gu.timer <= 0 {
			gu.state, gu.timer = gIdle, 2
		}
	case gDance:
		gu.anim += dt * 5
		gu.timer -= dt
		if gu.timer <= 0 {
			gu.state, gu.timer = gIdle, 0
		}
	case gDanceWith:
		if len(gu.path) > 0 {
			gu.walk(guestSpeed*1.6, dt)
		} else {
			gu.anim += dt * 5
			gu.dir = faceTo(gu.pos, g.p.pos)
			if g.time > gu.danceEnd {
				gu.state, gu.timer = gIdle, 1
			}
		}
	case gSwim:
		gu.anim += dt * 2
		gu.timer -= dt
		if gu.timer <= 0 {
			gu.state, gu.timer = gIdle, 0
		}
	case gDrink:
		gu.timer -= dt
		if gu.timer <= 0 {
			g.snd.Play("glug")
			gu.drunk++
			poisoned := false
			if gu.offered {
				g.startPoison(gu)
				poisoned = true
			}
			for _, pr := range g.lvl.Props {
				if pr.K.Poison && pr.armT > 0 && distRect(gu.pos, pr.Foot) < 30 && gu.poison <= 0 {
					g.startPoison(gu)
					poisoned = true
				}
			}
			if !poisoned {
				gu.icon, gu.iconT = g.line(guestCategory(gu), "drink"), 1.3
			}
			gu.state, gu.timer = gIdle, 1.5
			// too much to drink: they feel it
			if gu.drunk >= 2 && g.rng.Float64() < 0.3 {
				gu.drunk = 0
				gu.state, gu.timer = gVomit, 2.4
				gu.icon, gu.iconT = "BLEH", 2.4
			}
			gu.need += 0.18
		}
	case gToilet:
		g.updateToilet(gu, dt)
	case gLounge:
		gu.timer -= dt
		if gu.timer <= 0 {
			gu.state, gu.timer = gIdle, 0
		}
	case gVomit:
		gu.timer -= dt
		if gu.timer <= 0 {
			g.decals = append(g.decals, Decal{pos: Vec{gu.pos.X, gu.pos.Y + 4}, kind: 2, r: 8})
			if gu.poison <= 0 && g.rng.Float64() < 0.6 {
				gu.state, gu.timer = gSleep, 12+g.rng.Float64()*12
			} else {
				gu.state, gu.timer = gIdle, 2
			}
		}
	case gSleep:
		gu.timer -= dt
		if gu.icon == "" {
			gu.icon, gu.iconT = "ZZZ", 1.6
		}
		if gu.timer <= 0 {
			gu.state, gu.timer = gIdle, 2
			gu.icon, gu.iconT = "???", 1.5
		}
	case gAlarm:
		gu.timer -= dt
		if gu.timer <= 0 {
			if g.callerGuest == gu {
				g.guestReact(gu)
			} else if c := g.nearCop(gu.pos); c != nil && g.reporters() < 2 && (gu.sawKiller || gu.sawSlash || g.knowsAny(gu)) {
				g.claimBodies(gu)
				gu.state, gu.reportTo, gu.repath = gReport, c, 0
				gu.timer = 14
				gu.icon, gu.iconT = "!!", 1.2
			} else {
				g.scare(gu)
			}
		}
	case gToPhone:
		if gu.tech {
			if gu.walk(guestRunSpeed, dt) {
				gu.state, gu.timer = gCall, 2.5
				gu.dir = DirUp
				gu.icon, gu.iconT = "¡Teléfono roto!", 3
			}
			break
		}
		if gu.walk(guestRunSpeed, dt) {
			if g.phoneSpot(gu.pos) == nil {
				g.guestFlee(gu)
			} else {
				gu.state, gu.timer = gCall, 2.5
				gu.dir = DirUp
				gu.icon, gu.iconT = g.line(guestCategory(gu), "calling_police"), 3
			}
		}
	case gFlee:
		if gu.walk(guestRunSpeed, dt) {
			// every line is cut: there is nobody to call
			if g.callerGuest == gu {
				g.callerGuest = nil
			}
			g.scare(gu)
			gu.icon, gu.iconT = "¡Sin línea!", 2
		}
	case gCall:
		gu.timer -= dt
		if gu.timer <= 0 {
			if gu.tech {
				gu.tech = false
				if g.techCaller == gu {
					g.techCaller = nil
					if len(g.cutPhones()) > 0 && !g.techBusy() {
						g.dispatchTech()
					}
				}
				gu.state, gu.timer = gIdle, 3
				break
			}
			if g.callerGuest == gu {
				g.callerGuest = nil
				g.guestWarned(gu) // the warning is given: the alert becomes the cuffs
				g.callPolice(gu, gu.sawKiller)
			}
			gu.state, gu.timer = gIdle, 3
		}
	case gKiller:
		g.updateKiller(gu, dt)
	case gPerform:
		gu.anim += dt * 6
	case gPunch:
		gu.timer -= dt
		gu.repath -= dt
		if gu.repath <= 0 {
			gu.repath = 0.3
			gu.path = g.lvl.PathG(gu.pos, g.p.pos)
		}
		gu.walk(56, dt)
		if gu.timer <= 0 || g.p.hidden {
			gu.state, gu.timer = gIdle, 2
		} else if dist(gu.pos, g.p.pos) < 14 {
			g.p.knockT = 4
			g.p.danceT = 0
			g.snd.Play("thud")
			g.burst(Vec{g.p.pos.X, g.p.pos.Y - 20}, 8, 60, rgb(0xffe060), 0.4, 2, 100)
			gu.stabT = 0.25
			gu.state, gu.timer = gIdle, 4
			gu.icon, gu.iconT = "...", 2
		}
	case gFun:
		g.updateFun(gu, dt)
	case gZombie:
		g.updateZombie(gu, dt)
	case gLift:
		gu.moving = false
	case gScared:
		gu.timer -= dt
		if len(gu.path) == 0 {
			gu.path = g.lvl.PathG(gu.pos, g.farRoam(gu.pos))
		}
		gu.walk(guestRunSpeed, dt)
		if gu.timer <= 0 {
			gu.state, gu.timer, gu.path = gIdle, 2+g.rng.Float64()*2, nil
		}
	case gReport:
		g.updateReport(gu, dt)
	case gRun:
		gu.timer -= dt
		if gu.runUntil > 0 && g.time > gu.runUntil {
			gu.runUntil = 0
			gu.state, gu.timer, gu.path = gIdle, 2, nil
			break
		}
		if len(gu.path) == 0 || gu.timer <= 0 {
			gu.path = g.lvl.PathG(gu.pos, g.lvl.RandomRoam(g.rng))
			gu.timer = 3
		}
		gu.walk(57, dt)
	}
	if gu.moving {
		gu.anim += dt * stridePhase(gu.curSpd)
	}
}

// phoneSpot returns the closest working phone (nil if none).
func (g *Game) phoneSpot(from Vec) *Prop {
	var best *Prop
	for _, pr := range g.lvl.Props {
		if pr.K.Phone && !pr.used {
			if best == nil || dist(from, Vec{pr.X, pr.Y}) < dist(from, Vec{best.X, best.Y}) {
				best = pr
			}
		}
	}
	return best
}

func (g *Game) guestReact(gu *Guest) {
	if pr := g.phoneSpot(gu.pos); pr != nil {
		gu.path = g.lvl.PathG(gu.pos, Vec{pr.X, pr.Y + 14})
		gu.state = gToPhone
		return
	}
	g.guestFlee(gu)
}

func (g *Game) guestFlee(gu *Guest) {
	var t Vec
	for i := 0; i < 20; i++ {
		t = g.lvl.RandomRoam(g.rng)
		if dist(t, gu.pos) > 150 {
			break
		}
	}
	gu.path = g.lvl.PathG(gu.pos, t)
	gu.state = gFlee
}

func (g *Game) guestScan(gu *Guest) {
	if !g.canWatch(gu) || gu.state == gKiller || gu.state == gPunch {
		return
	}
	for _, b := range g.bodies {
		if b.reported || b.carried || !b.byPlayer {
			continue
		}
		if b.informer != nil && b.informer != gu && g.reporterActive(b.informer) {
			continue // someone is already going to warn about this one
		}
		if g.sees(gu.pos, gu.dir, b.pos, guestRange, guestDot) {
			known := g.knows(gu, b)
			g.knowBody(gu, b)
			if g.callerAvailable() {
				if g.alertPos == (Vec{}) {
					g.alertPos = b.pos
				}
				g.becomeCaller(gu)
			} else if !known {
				g.alarm(gu, g.line(guestCategory(gu), "saw_body"))
			}
			return
		}
	}
	for _, o := range g.guests {
		if o.armed && o != gu && !o.out() && g.sees(gu.pos, gu.dir, o.pos, guestRange, guestDot) {
			g.alarm(gu, "!?")
			break
		}
	}
	p := g.p
	if p.carrying != nil && !p.hidden && g.sees(gu.pos, gu.dir, p.pos, guestRange, guestDot) {
		gu.sawKiller = true
		gu.sawP, gu.warned = p, false
		if g.callerAvailable() {
			g.becomeCaller(gu)
		} else {
			g.alarm(gu, "!?")
		}
	}
}

// People do not kill each other by themselves any more: only a guest who picks up a weapon does.
const randomMurders = false

// --- rare murders among guests -------------------------------------------------

func (g *Game) updateMurder(dt float64) {
	if !randomMurders {
		return
	}
	g.murderT -= dt
	if g.murderT > 0 {
		return
	}
	g.murderT = 90 + g.rng.Float64()*90
	var cands []*Guest
	for _, gu := range g.guests {
		if !gu.out() && (gu.state == gIdle || gu.state == gGoto || gu.state == gDance || gu.state == gDrink || gu.state == gLounge) {
			cands = append(cands, gu)
		}
	}
	if len(cands) < 3 {
		return
	}
	k := cands[g.rng.Intn(len(cands))]
	k.tgt, k.tgtPlayer = nil, false
	if g.rng.Float64() < 0.04 && !g.p.hidden && !g.p.dead {
		k.tgtPlayer = true
		k.icon, k.iconT = "!!!", 6
		g.snd.Play("alert")
	} else {
		for tries := 0; tries < 10; tries++ {
			v := cands[g.rng.Intn(len(cands))]
			if v != k {
				k.tgt = v
				break
			}
		}
		if k.tgt == nil {
			return
		}
	}
	k.state, k.timer, k.repath = gKiller, 40, 0
}

func (g *Game) updateKiller(gu *Guest, dt float64) {
	var tp Vec
	switch {
	case gu.tgtPlayer && !g.p.dead && !g.p.hidden:
		tp = g.p.pos
	case !gu.tgtPlayer && gu.tgt != nil && !gu.tgt.out():
		tp = gu.tgt.pos
	default:
		if gu.armed {
			g.disarm(gu)
			return
		}
		gu.state, gu.timer = gIdle, 2
		return
	}
	gu.timer -= dt
	if gu.timer <= 0 {
		if gu.armed {
			g.disarm(gu)
			return
		}
		gu.state, gu.timer = gIdle, 2
		return
	}
	gu.repath -= dt
	if gu.repath <= 0 {
		gu.repath = 0.4
		gu.path = g.lvl.PathG(gu.pos, tp)
	}
	speed := 32.0
	if gu.tgtPlayer {
		speed = 57
	}
	gu.walk(speed, dt)
	if dist(gu.pos, tp) < 12 {
		gu.stabT = 0.25
		g.snd.Play("stab")
		if gu.tgtPlayer {
			g.die(guestLabel(gu) + " te ha apuñalado por la espalda")
			return
		}
		v := gu.tgt
		g.killGuest(v, false)
		g.sayNews("kill", map[string]string{"actor": guestLabel(gu), "victim": guestLabel(v)})
		saw := g.copSeesAt(v.pos)
		for _, o := range g.guests {
			if o != gu && g.canWatch(o) && g.sees(o.pos, o.dir, v.pos, guestRange, guestDot) {
				saw = true
				if len(g.bodies) > 0 {
					g.knowBody(o, g.bodies[len(g.bodies)-1])
				}
				if g.callerAvailable() {
					g.becomeCaller(o)
				} else {
					g.alarm(o, "!?")
				}
			}
		}
		if saw && gu.armed {
			g.suspectSeen(suspGuest, gu) // seen killing: the cuffs are for him now
		}
		if gu.armed && gu.rampage > 1 {
			gu.rampage--
			if g.rampageTarget(gu) {
				return
			}
		}
		if gu.armed {
			g.disarm(gu)
			return
		}
		gu.state, gu.timer = gIdle, 3
	}
}

// --- security guards -----------------------------------------------------------

func (g *Game) updateGuard(gd *Guard, dt float64) {
	if gd.dead {
		return
	}
	gd.moving = false
	if gd.iconT > 0 {
		gd.iconT -= dt
		if gd.iconT <= 0 {
			gd.icon = ""
		}
	}
	if gd.vip != 0 && g.updateVIPGuard(gd, dt) { // VIP staff: stands still until somebody walks in (vip.go)
		return
	}
	p := g.p
	if gd.vip == 0 && gd.state != gdBeat && gd.state != gdChase && gd.state != gdReact {
		rng := guardRange
		if gd.special {
			rng *= 1.3
		}
		if g.wanted > 0 || p.carrying != nil {
			rng *= 1.4
		}
		if !p.hidden && !p.dead && !g.stealthBoxed(p) && g.sees(gd.pos, gd.dir, p.pos, rng, guardDot) {
			rate := 1.0
			if dist(gd.pos, p.pos) < 45 {
				rate = 2.5
			}
			if g.wanted > 0 || p.carrying != nil {
				rate *= 2
			}
			gd.sight += rate * dt
			g.seenNow = g.seenNow || false
			if gd.sight > 0.4 && gd.icon == "" {
				gd.icon, gd.iconT = "HUH?", 0.3
			}
		} else if gd.sight > 0 {
			gd.sight = math.Max(0, gd.sight-0.8*dt)
		}
		if gd.sight >= 1.6 {
			gd.state, gd.timer = gdReact, 0.4
			gd.icon, gd.iconT = "HEY!", 1.5
			gd.lastSeen = p.pos
			gd.chaseT, gd.lost = 0, 0
			g.snd.Play("alert")
		}
	}
	// bodies
	gd.scanT -= dt
	if gd.scanT <= 0 && gd.state != gdBeat {
		gd.scanT = 0.4
		for _, b := range g.bodies {
			if !b.reported && !b.carried && b.byPlayer && g.sees(gd.pos, gd.dir, b.pos, guardRange, guardDot) {
				if g.callerAvailable() {
					g.callerGuard = gd
					gd.callT = 3
					gd.icon, gd.iconT = "?!", 2
					g.alertPos = b.pos
				}
				if gd.state == gdPatrol || gd.state == gdWait {
					gd.state = gdInvestigate
					gd.path = g.lvl.Path(gd.pos, b.pos)
				}
				break
			}
		}
	}
	if gd.callT > 0 {
		gd.callT -= dt
		if gd.callT <= 0 && g.callerGuard == gd {
			g.callerGuard = nil
			g.callPoliceGuard(gd)
		}
	}
	switch gd.state {
	case gdWait:
		gd.timer -= dt
		if gd.timer <= 0 {
			gd.path = g.lvl.Path(gd.pos, gd.route[gd.ri])
			gd.ri = (gd.ri + 1) % len(gd.route)
			gd.state = gdPatrol
		}
	case gdPatrol:
		arr := g.guardWalk(gd, 21, dt)
		if arr {
			gd.state, gd.timer = gdWait, 1.5+g.rng.Float64()*2
			gd.dir = Dir(g.rng.Intn(4))
		}
	case gdReturn:
		if g.guardWalk(gd, 31, dt) {
			gd.state, gd.timer = gdWait, 0.5
		}
	case gdInvestigate:
		if g.guardWalk(gd, 29, dt) {
			gd.state, gd.timer = gdWait, 2
		}
	case gdReact:
		gd.timer -= dt
		gd.dir = faceTo(gd.pos, p.pos)
		if gd.timer <= 0 {
			gd.state = gdChase
			gd.repath = 0
		}
	case gdChase:
		sees := !p.hidden && !p.dead && g.sees(gd.pos, gd.dir, p.pos, 150, -0.3)
		gd.chaseT += dt
		if sees {
			gd.lastSeen = p.pos
			gd.lost = 0
			g.seenNow = true
		} else {
			gd.lost += dt
		}
		// limits: leash around the post, endurance, and losing the target
		if dist(gd.pos, gd.home) > gd.leash || gd.chaseT > 9 || gd.lost > 3 {
			gd.state, gd.sight = gdSearch, 0
			gd.timer = 3
			gd.path = g.lvl.Path(gd.pos, gd.lastSeen)
			gd.icon, gd.iconT = "...", 1.5
			break
		}
		gd.repath -= dt
		if gd.repath <= 0 {
			gd.repath = 0.25
			if sees {
				gd.path = g.lvl.Path(gd.pos, p.pos)
			}
		}
		chase := 57.0
		if gd.special {
			chase = 94
		}
		g.guardWalk(gd, chase, dt)
		if sees && dist(gd.pos, p.pos) < 14 {
			gd.state = gdBeat
			g.snd.Play("thud")
			g.die("Los guardias de seguridad te han dado una paliza")
		}
	case gdSearch:
		arr := g.guardWalk(gd, 28, dt)
		if arr {
			gd.timer -= dt
			if gd.timer <= 0 {
				gd.state = gdReturn
				gd.path = g.lvl.Path(gd.pos, gd.route[gd.ri])
			}
		}
	case gdBeat:
		gd.anim += dt * 10
		if g.tick%6 == 0 {
			g.burst(Vec{p.pos.X, p.pos.Y - 8}, 3, 40, rgb(0xb01818), 0.4, 2, 200)
		}
	}
	g.guardSeparate(gd, dt)
	if gd.moving {
		gd.anim += dt * stridePhase(gd.curSpd)
	}
}

// guardSeparate: the same gentle anti-clumping nudge as copSeparate, so a
// pair of guards does not stand fused together.
func (g *Game) guardSeparate(gd *Guard, dt float64) {
	if gd.dead {
		return
	}
	const minDist, push = 13.0, 36.0
	var fx, fy float64
	for _, o := range g.guards {
		if o == gd || o.dead {
			continue
		}
		dx, dy := gd.pos.X-o.pos.X, gd.pos.Y-o.pos.Y
		if d := math.Hypot(dx, dy); d > 0 && d < minDist {
			k := (minDist - d) / minDist
			fx += dx / d * k
			fy += dy / d * k
		}
	}
	gd.pos.X += fx * push * dt
	gd.pos.Y += fy * push * dt
}

func (g *Game) guardWalk(gd *Guard, speed, dt float64) bool {
	m, arr := follow(&gd.pos, &gd.dir, &gd.path, speed*paceK, dt)
	gd.moving, gd.curSpd = m, speed
	return arr
}

// --- police ------------------------------------------------------------------

// knowBody: a guest has seen a body that nobody has been warned about yet.
func (g *Game) knowBody(gu *Guest, b *Body) {
	for _, s := range b.seers {
		if s == gu {
			return
		}
	}
	b.seers = append(b.seers, gu)
}

func (g *Game) knows(gu *Guest, b *Body) bool {
	for _, s := range b.seers {
		if s == gu {
			return true
		}
	}
	return false
}

// knowsAny: does this guest carry news of a body nobody has been told about?
func (g *Game) knowsAny(gu *Guest) bool {
	for _, b := range g.bodies {
		if !b.reported && !b.carried && g.knows(gu, b) && (b.informer == nil || b.informer == gu || !g.reporterActive(b.informer)) {
			return true
		}
	}
	return false
}

// deliver: the warning reaches the police (phone call or a cop in person): the bodies the guest saw are now detected.
func (g *Game) deliver(gu *Guest) {
	for _, b := range g.bodies {
		if !b.reported && !b.carried && g.knows(gu, b) {
			b.reported = true
			b.seers = nil
			if g.alertPos == (Vec{}) {
				g.alertPos = b.pos
			}
		}
	}
}

func (g *Game) callPolice(gu *Guest, informed bool) {
	if g.policeCalled {
		return
	}
	g.deliver(gu)
	g.raisePolice(gu.pos, informed)
}

// callPoliceGuard: a security guard phones in the bodies he has seen around him.
func (g *Game) callPoliceGuard(gd *Guard) {
	if g.policeCalled {
		return
	}
	for _, b := range g.bodies {
		if !b.reported && !b.carried && dist(b.pos, gd.pos) < 150 && g.lvl.LOS(gd.pos, b.pos) {
			b.reported, b.seers = true, nil
			if g.alertPos == (Vec{}) {
				g.alertPos = b.pos
			}
		}
	}
	g.raisePolice(gd.pos, false)
}

// playerHeat: is any of the trouble the player's doing? Fights and murders
// between other people never touch the criminality rating.
func (g *Game) playerHeat() bool {
	if g.wanted > 0 {
		return true
	}
	for _, b := range g.bodies {
		if b.byPlayer && b.reported && !b.carried {
			return true
		}
	}
	return false
}

func (g *Game) raisePolice(at Vec, informed bool) {
	if g.dbg.noPolice {
		return
	}
	if g.policeCalled {
		// Another witness may refresh the search, but it must not summon a
		// second, overlapping authority.
		if informed {
			g.markPlayerWanted(32)
			g.killPos = at
		}
		return
	}
	g.policeCalled = true
	g.snd.Play("ring")
	g.respKind, g.swatKills = unitPolice, 0
	g.stars, g.starsDecayT = 0, 0
	if g.playerHeat() || informed {
		// the rating never falls: a new call starts at the highest level reached
		g.stars = maxI(1, g.starFloor)
		g.respKind = unitForStars(g.stars)
		if g.stars > g.starFloor {
			g.starFloor = g.stars
		}
	}
	if g.alertPos == (Vec{}) {
		g.alertPos = at
	}
	// (a witness who has NOT warned yet no longer counts as "informed": see suspect.go)
	if informed {
		if !g.playerCleared() {
			g.wanted = 70
			g.ensureSuspect()
		}
	} else if g.lastKiller.kind == suspNone && g.rng.Float64() < 0.45 {
		var live []*Guest
		for _, gu := range g.guests {
			if !gu.out() {
				live = append(live, gu)
			}
		}
		if len(live) > 3 {
			g.suspect = live[g.rng.Intn(len(live))]
		}
	}
	g.pendingCar = g.time + 6
	g.sayNews("police_call", nil)
}

// changeResponse enforces a single responder type on scene at a time: every
// existing officer of a different tier returns to their car before the new
// tier takes over the investigation.
func (g *Game) changeResponse(kind int) {
	if kind == g.respKind {
		return
	}
	for _, c := range g.cops {
		if c.medic || c.kind == kind || c.state == cLeave {
			continue
		}
		g.releaseCop(c)
		c.state = cLeave
		c.path = g.lvl.Path(c.pos, g.vehDoor(c))
	}
	for _, v := range g.veh {
		if v.unit == kind || (v.kind != 0 && v.kind != 2) {
			continue
		}
		// a vehicle with its crew still out leaves by itself once they are aboard
		crew := false
		for _, c := range g.cops {
			if c.veh == v {
				crew = true
			}
		}
		if !crew {
			v.state = 2
		}
	}
	g.respKind = kind
}

// closeCase: the suspect has been taken, so the player's guilt disappears,
// every authority walks back to its vehicle and the stars drop to zero.
func (g *Game) closeCase() {
	for _, pl := range g.players() {
		pl.disguised = false
		pl.jail.sus = false // the case is closed: nobody is a suspect any more
	}
	g.wanted, g.seenNow = 0, false
	g.pendingCar, g.pendingUnit = -1, -1
	g.suspect = nil
	g.lastKiller = lastKill{}
	g.stars, g.starsDecayT = 0, 0
	for _, b := range g.bodies {
		b.byPlayer = false
	}
	for _, o := range g.cops {
		if o.medic || o.state == cLeave {
			continue
		}
		g.releaseCop(o)
		o.state = cLeave
		o.path = g.lvl.Path(o.pos, g.vehDoor(o))
	}
	for _, v := range g.veh {
		if (v.kind == 0 || v.kind == 2) && v.state == 0 && !v.spawned {
			v.state = 2
		}
	}
}

// mayLeave: the minimum stay on scene (20s) has passed.
func (c *Cop) mayLeave() bool { return c.medic || (c.onSceneT >= 20 && !jailHold) }

func (g *Game) vehDoor(c *Cop) Vec {
	v := c.veh
	if v == nil {
		return c.pos
	}
	if v.isHeli() {
		return g.lvl.NearestFree(Vec{v.tx, v.ty})
	}
	return g.lvl.NearestFree(Vec{v.x, v.y - 24})
}

func (g *Game) freePad() int {
	for i := range g.lvl.Def.Pads {
		used := false
		for _, v := range g.veh {
			if v.isHeli() && v.pad == i && v.state != 2 {
				used = true
			}
		}
		if !used {
			return i
		}
	}
	return -1
}

func (g *Game) dispatch(medic bool) {
	def := g.lvl.Def
	if def.Police == "heli" {
		pad := g.freePad()
		if pad < 0 {
			if medic {
				g.pendingAmb = g.time + 3
			} else {
				g.pendingCar = g.time + 3
			}
			return
		}
		p := def.Pads[pad]
		k := 2
		if medic {
			k = 3
		}
		g.veh = append(g.veh, &Vehicle{kind: k, x: p.X, y: -130, tx: p.X, ty: p.Y, pad: pad, unit: g.unitFor(medic)})
		g.snd.Play("siren")
		return
	}
	k := 0
	if medic {
		k = 1
	}
	n := 0
	for _, v := range g.veh {
		if v.state != 2 {
			n++
		}
	}
	tx := def.CarTo.X - float64(n)*100
	if def.ParkX1 > def.ParkX0 {
		// a custom parking range: stop inside it, spaced out, on the side the cars arrive from
		if def.CarFrom.X > def.CarTo.X {
			tx = math.Max(def.ParkX0, def.ParkX1-float64(n)*100)
		} else {
			tx = math.Min(def.ParkX1, def.ParkX0+float64(n)*100)
		}
	}
	// emergency vehicles drive in the lane matching their direction
	ly := def.CarFrom.Y + g.laneOff()
	if def.CarFrom.X > def.CarTo.X {
		ly = def.CarFrom.Y - g.laneOff()
	}
	g.veh = append(g.veh, &Vehicle{kind: k, x: def.CarFrom.X + 40, y: ly, tx: tx, ty: ly, unit: g.unitFor(medic)})
	g.snd.Play("siren")
}

// maybeAmbulance: with many reported bodies one cop is not enough, so a paramedic comes too.
func (g *Game) maybeAmbulance() {
	if g.pendingAmb >= 0 && g.time >= g.pendingAmb {
		g.pendingAmb = -1
		g.ambCount++
		g.ambAt = g.time
		g.dispatch(true)
	}
	if g.pendingAmb >= 0 || !g.policeCalled {
		return
	}
	n := 0
	for _, b := range g.bodies {
		if b.reported && !b.carried {
			n++
		}
	}
	if g.ambCalled {
		// not enough hands: a second ambulance for a big pile of bodies
		if g.ambCount == 1 && n >= 5 && g.time > g.ambAt+25 {
			g.pendingAmb = g.time + 5
			g.sayNews("ambulance_call", nil)
		}
		return
	}
	need := n >= 3
	if n >= 1 {
		for _, c := range g.cops {
			if !c.medic && (c.bagN >= 2 || c.state == cLeave) {
				need = true
			}
		}
	}
	if need {
		g.ambCalled = true
		g.pendingAmb = g.time + 7
		g.sayNews("ambulance_call", nil)
	}
}

func (g *Game) updateVehicles(dt float64) {
	def := g.lvl.Def
	if g.pendingCar >= 0 && g.time >= g.pendingCar {
		g.pendingCar = -1
		g.dispatch(false)
	}
	out := g.veh[:0]
	for _, v := range g.veh {
		heli := v.isHeli()
		if heli {
			v.rot += dt * 30
		}
		if !heli {
			laneRemap(def, v)
		}
		switch v.state {
		case 0:
			if heli {
				v.y += 100 * dt
				if v.y >= v.ty {
					v.y, v.state = v.ty, 1
				}
			} else if v.lane {
				laneStep(def, v, dt)
			} else {
				v.x -= 130 * dt
				if v.x <= v.tx {
					v.x, v.state = v.tx, 1
				}
			}
		case 1:
			if v.kind == 9 {
				if !v.spawned {
					v.spawned = true
					g.kidFromVehicle(v)
					v.hold = 2.2
				}
				if v.hold -= dt; v.hold <= 0 {
					v.state = 2
				}
				break
			}
			if !v.spawned && v.kind == 6 {
				v.spawned = true
				g.limoArrives(v)
				v.hold = 2.2
			}
			if v.kind == 6 && v.hold > 0 {
				v.hold -= dt
				if v.hold <= 0 {
					v.state = 2
				}
				break
			}
			if v.kind == 7 {
				if !v.spawned {
					v.spawned = true
					g.firemenArrive(v)
				}
				break
			}
			if v.kind == 8 {
				if !v.spawned {
					v.spawned = true
					g.techsArrive(v)
				}
				break
			}
			if v.revive { // the ambulance a player called for his dead partner (revive.go)
				if !v.spawned {
					v.spawned = true
					g.reviveArrives(v)
				}
				break
			}
			if !v.spawned && (v.kind == 4 || v.kind == 5) {
				v.spawned = true
				g.slasherArrives(v)
				v.state = 2
			}
			if !v.spawned {
				v.spawned = true
				medic := v.kind == 1 || v.kind == 3
				var pos Vec
				if heli {
					pos = g.lvl.NearestFree(Vec{v.tx + 14, v.ty + 22})
				} else {
					pos = g.lvl.NearestFree(Vec{v.x - 12, v.y - 24})
				}
				n := 1 + g.rng.Intn(2)
				switch v.unit {
				case unitFBI:
					n = 2
				case unitMilitary:
					n = 3
				case unitSWAT:
					n = 4
				case unitRangers:
					n = 5
				}
				if medic {
					cnt := 0
					for _, b := range g.bodies {
						if b.reported && !b.carried {
							cnt++
						}
					}
					n = clampI((cnt+1)/2, 1, 3)
				}
				if !medic && v.unit != g.respKind {
					g.changeResponse(v.unit)
					if g.pendingUnit == v.unit {
						g.pendingUnit = -1
					}
				}
				for k := 0; k < n; k++ {
					p2 := g.lvl.NearestFree(Vec{pos.X + float64(k)*10, pos.Y - float64(k%2)*8})
					cop := &Cop{pos: p2, dir: DirDown, veh: v, medic: medic, alt: k, kind: v.unit, maxChase: 3 + g.rng.Float64()*11}
					// Every authority type gives up within forty seconds.
					cop.maxSearch = 27 + g.rng.Float64()*11
					if v.unit > 0 {
						cop.maxSearch = 34 + g.rng.Float64()*6
						cop.shootT = 8 + g.rng.Float64()*4
					}
					if g.wanted > 0 {
						cop.maxSearch += 18
					}
					g.cops = append(g.cops, cop)
					if medic {
						g.copNext(cop)
					} else if k == 0 && g.hasAlertPos() {
						// Only the first officer of the crew heads straight for
						// the tip; the rest fan out on their own patrol instead
						// of all marching to the same spot.
						cop.state = cGoto
						cop.path = g.lvl.Path(cop.pos, g.alertPos)
					} else {
						cop.state, cop.wait = cSweep, float64(k)*0.4
					}
				}
			}
		case 2:
			if heli {
				v.y -= 100 * dt
				if v.y < -140 {
					continue
				}
			} else if v.lane {
				if laneStep(def, v, dt) {
					continue
				}
			} else {
				v.x += 130 * dt
				if v.x > def.CarFrom.X+80 {
					continue
				}
			}
		}
		out = append(out, v)
	}
	var alive []*Vehicle
	for _, v := range out {
		if !v.dead {
			alive = append(alive, v)
		}
	}
	g.veh = alive
}

// copNext decides what an officer or paramedic does when it has nothing urgent.
// Paramedics collect and carry bodies. Police first search for the killer; when the search is
// over they carry the remaining bagged bodies to their vehicle, then leave.
func (g *Game) copNext(c *Cop) {
	if !c.medic && c.searchT < c.maxSearch {
		c.state, c.wait, c.path = cSweep, 0, nil
		return
	}
	var best *Body
	canCarry := c.medic || c.kind > 0 || (c.bagN < 2 && !g.ambCalled)
	for _, b := range g.bodies {
		if !canCarry || b.carried || !b.reported || b.fallT > 0 || (b.claimed != nil && b.claimed != c) {
			continue
		}
		if best == nil || dist(c.pos, b.pos) < dist(c.pos, best.pos) {
			best = b
		}
	}
	if best != nil {
		best.claimed = c
		if !c.medic {
			c.bagN++
		}
		c.target, c.state = best, cCollect
		c.path = g.lvl.Path(c.pos, best.pos)
		return
	}
	if !c.medic {
		if s := g.suspect; s != nil && !s.out() {
			c.state, c.repath = cArrest, 0
			s.state, s.timer, s.path = gRun, 3, nil
			s.icon, s.iconT = "!?", 1.5
			return
		}
	}
	if !c.mayLeave() {
		c.state, c.wait, c.path = cSweep, 0, nil
		return
	}
	c.state = cLeave
	c.path = g.lvl.Path(c.pos, g.vehDoor(c))
}

func (g *Game) removeBody(b *Body) {
	for i, x := range g.bodies {
		if x == b {
			g.bodies = append(g.bodies[:i], g.bodies[i+1:]...)
			return
		}
	}
}

func (g *Game) copWalk(c *Cop, speed, dt float64) bool {
	m, arr := follow(&c.pos, &c.dir, &c.path, speed*paceK, dt)
	c.moving, c.curSpd = m, speed
	return arr
}

func (g *Game) releaseCop(c *Cop) {
	if c.target != nil {
		c.target.claimed = nil
		c.target = nil
	}
	if c.carry != nil {
		c.carry.carried = false
		c.carry.pos = c.pos
		c.carry.claimed = nil
		c.carry = nil
	}
}

var copPhrases = []string{"I'M IN POSITION", "ROGER THAT...", "NEED BACKUP!", "WHAT A MESS...", "COPY THAT", "SECURE THE AREA!"}

const (
	copWalkSpeed  = 52.0 // police walk faster than a walking player
	copSweepSpeed = 46.0
	copChaseSpeed = 78.0 // a bit faster than a running player: escapable
	medicSpeed    = 43.0
)

// sweepTarget: where the police look next. First around where the killer was last seen.
func (g *Game) sweepTarget(c *Cop) Vec {
	// each officer takes a different part of the map: pick the candidate that is
	// farthest from where the other officers are, or already heading
	far := func(p Vec) float64 {
		m := 1e9
		for _, o := range g.cops {
			if o == c || o.medic || o.state == cLeave {
				continue
			}
			if d := dist(p, o.pos); d < m {
				m = d
			}
			if n := len(o.path); n > 0 {
				if d := dist(p, o.path[n-1]); d < m {
					m = d
				}
			}
		}
		return m
	}
	best, bestD := Vec{}, -1.0
	for i := 0; i < 12; i++ {
		p := g.lvl.RandomRoam(g.rng)
		if g.wanted > 0 && c.searchT < 25 && g.killPos != (Vec{}) && dist(p, g.killPos) > 190 && i < 8 {
			continue
		}
		if d := far(p); d > bestD {
			best, bestD = p, d
			if d > 160 {
				break
			}
		}
	}
	if bestD < 0 {
		return g.lvl.RandomRoam(g.rng)
	}
	return best
}

func (g *Game) updateCop(c *Cop, dt float64) {
	c.moving = false
	if c.state != cLeave {
		c.onSceneT += dt
	}
	if c.iconT > 0 {
		c.iconT -= dt
		if c.iconT <= 0 {
			c.icon = ""
		}
	}
	g.copUnstick(c, dt)
	if g.coopCopDistracted(c, dt) { // two players: the partner is distracting this officer
		return
	}
	if g.copSlashHook(c, dt) {
		return
	}
	g.copBrutal(c, dt)
	p := g.copTarget(c) // two players: the suspect this officer goes after (twojail.go)
	// police notice bodies lying around; while hunting the killer they only deal with the ones right next to them
	bodyRange := copRange
	if g.wanted > 0 {
		bodyRange = 75
	}
	if c.state != cLeave {
		for _, b := range g.bodies {
			if !b.reported && !b.carried && !b.bagged && g.sees(c.pos, c.dir, b.pos, copRange, copDot) {
				b.reported = true
				if c.icon == "" {
					c.icon, c.iconT = copPhrases[g.rng.Intn(len(copPhrases))], 2
				}
			}
		}
	}
	if !c.medic && (c.state == cSweep || c.state == cGoto) {
		for _, b := range g.bodies {
			if (c.bagN < 2 || c.kind > 0) && !b.bagged && !b.carried && b.fallT <= 0 && b.claimed == nil && g.sees(c.pos, c.dir, b.pos, bodyRange, copDot) {
				b.reported, b.claimed = true, c
				c.target, c.state = b, cBagHere
				c.path = g.lvl.Path(c.pos, b.pos)
				break
			}
		}
	}
	chasing := c.state == cChase
	sees := false
	// A cop already sent home (a lower tier stood down by an escalation, or
	// one whose shift ended) is done: nothing brings it back into the case.
	if !c.medic && c.state != cLeave {
		rng, dot := copRange, copDot
		if chasing {
			rng, dot = 150, -0.3
		}
		sees = !p.hidden && !p.dead && g.copInterest(p) && g.sees(c.pos, c.dir, p.pos, rng, dot)
	}
	if sees && c.state != cTired {
		if !chasing {
			c.icon, c.iconT = g.line(copCategory(c.kind), "halt"), 1.8
			g.snd.Play("alert")
			c.chaseT = 0
			g.releaseCop(c)
		}
		c.state = cChase
		c.lastSeen = p.pos
		g.killPos = p.pos
		c.lost = 0
		g.seenNow = true
		if p.carrying != nil && g.playerCleared() {
			g.withPlayer(p, func() { g.suspectSeen(suspPlayer, nil) }) // a body on his shoulder is as good as seeing him kill
		}
		if g.wanted < 20 && !g.playerCleared() {
			g.wanted = 20
		}
		g.markSuspect(p)
	}
	if !c.medic && c.state != cLeave && c.state != cChase && c.state != cTired {
		// The FBI never gives up investigating on its own: it keeps searching
		// until the criminality drops, or a higher tier is called in.
		if c.kind != unitFBI || g.stars < 2 {
			c.searchT += dt
		}
	}
	switch c.state {
	case cGoto:
		if g.copWalk(c, copWalkSpeed, dt) || len(c.path) == 0 {
			c.state, c.wait = cSweep, 1
		}
	case cSweep:
		if c.wait > 0 {
			c.wait -= dt
			if int(c.wait*3)%2 == 0 && g.rng.Float64() < 0.05 {
				c.dir = Dir(g.rng.Intn(4))
			}
			break
		}
		if len(c.path) == 0 {
			c.path = g.lvl.Path(c.pos, g.sweepTarget(c))
		}
		if g.copWalk(c, copSweepSpeed, dt) {
			c.wait = 0.7 + g.rng.Float64()*1.3
			if c.kind == unitRangers {
				// Power Rangers never stand still between patrol legs.
				c.wait = 0
			}
			c.dir = Dir(g.rng.Intn(4))
		}
		if c.searchT >= c.maxSearch {
			g.copNext(c)
		}
	case cBagHere:
		b := c.target
		if b == nil || b.bagged || b.carried {
			g.releaseCop(c)
			g.copNext(c)
			break
		}
		if dist(c.pos, b.pos) < 16 {
			c.moving = false
			c.timer += dt
			if c.timer >= 1.6 {
				c.timer = 0
				b.bagged = true
				c.bagN++
				b.claimed = nil
				c.target = nil
				c.icon, c.iconT = copPhrases[g.rng.Intn(len(copPhrases))], 2
				c.state, c.wait = cSweep, 0.5
			}
		} else {
			g.copWalk(c, copWalkSpeed, dt)
			if len(c.path) == 0 {
				c.path = g.lvl.Path(c.pos, b.pos)
			}
		}
	case cChase:
		c.chaseT += dt
		if !sees {
			c.lost += dt
			if c.lost > 1.6 {
				c.state, c.timer = cSearch, 5
				c.path = g.lvl.Path(c.pos, c.lastSeen)
				break
			}
		}
		if c.chaseT > c.maxChase {
			c.state, c.timer = cTired, 5
			c.icon, c.iconT = g.line(copCategory(c.kind), "tired"), 4
			break
		}
		c.repath -= dt
		if c.repath <= 0 {
			c.repath = 0.25
			if sees {
				c.path = g.lvl.Path(c.pos, p.pos)
			}
		}
		g.copWalk(c, copChaseSpeed, dt)
		if sees && dist(c.pos, p.pos) < 13 {
			g.copCatch(c, p)
		}
	case cTired:
		c.timer -= dt
		g.copWalk(c, 15, dt)
		if c.timer <= 0 && !c.mayLeave() {
			c.timer = 1
		} else if c.timer <= 0 {
			// gives up: back to the vehicle and away
			g.wanted = math.Min(g.wanted, 10)
			g.releaseCop(c)
			c.state = cLeave
			c.path = g.lvl.Path(c.pos, g.vehDoor(c))
		}
	case cSearch:
		arr := g.copWalk(c, copWalkSpeed, dt)
		if arr {
			c.timer -= dt
			if c.timer <= 0 {
				g.copNext(c)
			}
		}
	case cCollect:
		b := c.target
		if b == nil || b.carried {
			g.copNext(c)
			break
		}
		speed := copWalkSpeed
		if c.medic {
			speed = medicSpeed
		}
		g.copWalk(c, speed, dt)
		if dist(c.pos, b.pos) < 16 {
			if b.bagged {
				b.carried = true
				c.carry, c.target = b, nil
				c.state = cReturn
				c.path = g.lvl.Path(c.pos, g.vehDoor(c))
			} else {
				c.state, c.timer = cBag, 1.6
				c.path = nil
			}
		} else if len(c.path) == 0 {
			c.path = g.lvl.Path(c.pos, b.pos)
		}
	case cBag:
		b := c.target
		if b == nil || b.carried {
			g.copNext(c)
			break
		}
		c.timer -= dt
		if c.timer <= 0 {
			b.bagged = true
			b.carried = true
			c.carry, c.target = b, nil
			c.state = cReturn
			c.path = g.lvl.Path(c.pos, g.vehDoor(c))
		}
	case cReturn:
		speed := 28.0
		if c.medic {
			speed = 32
		}
		if g.copWalk(c, speed, dt) {
			if c.carry != nil {
				g.removeBody(c.carry)
				c.carry = nil
			}
			g.copNext(c)
		}
	case cArrest:
		s := g.suspect
		if s == nil || s.out() {
			g.copNext(c)
			break
		}
		c.repath -= dt
		if c.repath <= 0 {
			c.repath = 0.3
			c.path = g.lvl.Path(c.pos, s.pos)
		}
		g.copWalk(c, 59, dt)
		if dist(c.pos, s.pos) < 14 {
			s.gone = true
			g.suspect = nil
			g.sayNews("suspect_taken", map[string]string{"victim": guestLabel(s)})
			// the suspect is taken: the player's guilt goes away and all the
			// authorities go home
			g.closeCase()
			g.checkDone()
		}
	case cLeave:
		if g.copWalk(c, copWalkSpeed, dt) {
			// back in the vehicle: it leaves once every crew member is aboard
			for i, x := range g.cops {
				if x == c {
					g.cops = append(g.cops[:i], g.cops[i+1:]...)
					break
				}
			}
			crew := false
			for _, o := range g.cops {
				if o.veh == c.veh {
					crew = true
				}
			}
			if !crew && c.veh != nil {
				c.veh.state = 2
			}
			if c.medic {
				left := false
				for _, o := range g.cops {
					if o.medic {
						left = true
					}
				}
				if !left {
					g.ambCalled, g.ambCount = false, 0
				}
			} else {
				// Do not clear the case while another type of authority is still
				// arriving or searching.
				active := false
				for _, o := range g.cops {
					if !o.medic && o.state != cLeave {
						active = true
					}
				}
				for _, v := range g.veh {
					if (v.kind == 0 || v.kind == 2) && v.state != 2 {
						active = true
					}
				}
				if !active {
					g.policeCalled = false
					g.suspect = nil
					g.wanted = 0
					g.clearSuspects()
					g.stars, g.starsDecayT, g.respKind, g.pendingUnit = 0, 0, unitPolice, -1
					for _, b := range g.bodies {
						if !b.carried && !b.bagged {
							b.reported, b.claimed = false, nil
						}
					}
				}
			}
		}
	}
	g.copSeparate(c, dt)
	if c.moving {
		c.anim += dt * stridePhase(c.curSpd)
	}
}

// copSeparate gently pushes officers apart when they end up standing on top
// of each other, so a crew reads as several people rather than one blob.
func (g *Game) copSeparate(c *Cop, dt float64) {
	if c.state == cLeave || c.state == cArrest || c.state == cSlash {
		return
	}
	const minDist, push = 13.0, 36.0
	var fx, fy float64
	for _, o := range g.cops {
		if o == c {
			continue
		}
		dx, dy := c.pos.X-o.pos.X, c.pos.Y-o.pos.Y
		if d := math.Hypot(dx, dy); d > 0 && d < minDist {
			k := (minDist - d) / minDist
			fx += dx / d * k
			fy += dy / d * k
		}
	}
	c.pos.X += fx * push * dt
	c.pos.Y += fy * push * dt
}

// stridePhase: animation frames per second for a walking or running speed (px/s),
// so feet do not slide on the floor.
func stridePhase(speed float64) float64 {
	if speed > 42 {
		return speed * wk / 5.0
	}
	return speed * wk / 3.4
}

// ---------------------------------------------------------------------------
// Ambient jokes: cars, tanks, horses and UFOs crossing the scene now and then
// ---------------------------------------------------------------------------

const (
	ambCar = iota
	ambBike
	ambTank
	ambHorse
	ambUFO
	ambObj
	ambTow
	ambMeteor
)

type Ambient struct {
	kind   int
	x, y   float64
	vx     float64
	col    int
	phase  float64
	name   string
	dead   bool
	loaded bool    // tow truck: already picked up a wreck
	v0     float64 // cruising speed (signed); vx is what traffic currently allows
	length float64
	stopX  float64 // where a special vehicle does its stunt
	wait   float64
	done   bool
	t2     float64
	hop    float64 // visual lift (stunts); the lane is unchanged
}

// towLane: true for the lower, rightward lane, false for the upper, leftward one.
// laneOffset: how far each one-way lane sits from the dashed center line
// (def.CarFrom.Y, which is drawn right on top of it). Positive Y is down on
// screen, so the lower lane (right-moving) is CarFrom.Y+laneOffset and the
// upper lane (left-moving) is CarFrom.Y-laneOffset — on opposite sides of
// the dashed line, not both on the same side of it.
const laneOffset = 18.0 // lanes one car height (36px) apart

// laneOff is the lane distance for the current level (custom backgrounds set it from the road width).
func (g *Game) laneOff() float64 {
	if o := g.lvl.Def.LaneOff; o > 0 {
		return o
	}
	return laneOffset
}

func (g *Game) towLane(y float64) bool {
	return y > g.lvl.Def.CarFrom.Y
}

// maybeTow sends a tow truck down whichever one-way lane still has a wreck
// and does not already have one working it (at most 1 truck per lane).
func (g *Game) maybeTow(dt float64) {
	def := g.lvl.Def
	if def.Police != "car" || len(g.wrecks) == 0 {
		return
	}
	g.towT -= dt
	if g.towT > 0 {
		return
	}
	g.towT = 10 + g.rng.Float64()*8
	for _, lane := range [2]bool{true, false} {
		haveWreck := false
		wy := 0.0
		for _, w := range g.wrecks {
			if g.towLane(w.Y) == lane {
				haveWreck, wy = true, w.Y
				break
			}
		}
		if !haveWreck {
			continue
		}
		busy := false
		for _, a := range g.amb {
			if a.kind == ambTow && !a.dead && g.towLane(a.y) == lane {
				busy = true
				break
			}
		}
		if busy {
			continue
		}
		a := &Ambient{kind: ambTow, col: g.rng.Intn(4)}
		// it drives along the very lane of the wreck (levels with custom lane widths or several
		// lanes would otherwise have it pass beside the wreck without ever picking it up)
		if lane {
			a.y, a.x, a.vx = wy, -100, 70
		} else {
			a.y, a.x, a.vx = wy, 1060, -70
		}
		g.amb = append(g.amb, a)
	}
}

func (g *Game) updateAmbient(dt float64) {
	if g.lvl.Def.Transit != "" {
		return // open sea and outer space have no street traffic
	}
	if g.forceAmb != "" {
		// debug: -ambient NAME sends that vehicle down the right-moving lane at once
		def := g.lvl.Def
		a := &Ambient{kind: ambObj, name: g.forceAmb, y: def.CarFrom.Y + g.laneOff(), x: -60, vx: propKinds[g.forceAmb].Ambient, phase: 1}
		a.v0, a.length, a.stopX = a.vx, ambLength(a), 500
		g.amb = append(g.amb, a)
		g.forceAmb = ""
	}
	g.ambT -= dt
	if g.ambT <= 0 {
		def := g.lvl.Def
		if def.Highway {
			g.ambT = 0.5 + g.rng.Float64()*0.9
		} else if def.FastTraffic {
			g.ambT = 3 + g.rng.Float64()*3
		} else {
			g.ambT = 9 + g.rng.Float64()*14
		}
		a := &Ambient{col: g.rng.Intn(4), phase: g.rng.Float64() * 6}
		speed := 100.0
		right := g.rng.Intn(2) == 0
		switch def.Traffic {
		case "right":
			right = true
		case "left":
			right = false
		}
		if def.Police == "car" {
			// A tank is a very occasional set piece, not routine traffic.
			r := g.rng.Float64()
			switch {
			case r < 0.035:
				a.kind = ambTank
			case r < 0.20:
				a.kind = ambHorse
			case r < 0.45:
				a.kind = ambBike
			default:
				a.kind = ambCar
			}
			if fastEv {
				a.kind = ambTank
			}
			// a tank is rare: at most twice per party
			if a.kind == ambTank {
				if g.tankN >= 2 {
					a.kind = ambCar
				} else {
					g.tankN++
				}
			}
			var custom []string
			for n, k := range propKinds {
				if k.Ambient > 0 {
					custom = append(custom, n)
				}
			}
			sort.Strings(custom)
			if len(custom) > 0 && !fastEv && g.rng.Float64() < 0.4 {
				a.kind, a.name = ambObj, custom[g.rng.Intn(len(custom))]
			}
			// Two one-way lanes, one on each side of the dashed center line:
			// below it always drives right, above it always drives left.
			if right {
				a.y = def.CarFrom.Y + laneOffset
			} else {
				a.y = def.CarFrom.Y - laneOffset
			}
			if len(def.Lanes) > 0 {
				a.y = def.Lanes[g.rng.Intn(len(def.Lanes))]
				right = a.y > def.CarFrom.Y
			}
			switch a.kind {
			case ambObj:
				speed = propKinds[a.name].Ambient
			case ambBike:
				speed = 140
			case ambTank:
				speed = 55
			case ambHorse:
				speed = 85
			}
		} else if def.Sky == "space" {
			a.kind = ambMeteor
			a.y = 8 + g.rng.Float64()*30
			speed = 220
		} else {
			a.kind = ambUFO
			a.y = 14 + g.rng.Float64()*26
			speed = 45
		}
		if right {
			a.x, a.vx = -100, speed
		} else {
			a.x, a.vx = 1060, -speed
		}
		a.v0 = a.vx
		a.length = ambLength(a)
		a.stopX = 280 + g.rng.Float64()*400
		if a.kind == ambCar || a.kind == ambBike || a.kind == ambObj || a.kind == ambTank || a.kind == ambHorse {
			// never spawn on top of the car that just entered this lane
			for _, o := range g.amb {
				if !o.dead && math.Abs(o.y-a.y) < 8 && math.Abs(o.x-a.x) < 110 {
					a = nil
					break
				}
			}
		}
		if a != nil {
			g.amb = append(g.amb, a)
		} else {
			g.ambT = 2
			if def.Highway {
				g.ambT = 0.25
			}
		}
	}
	g.maybeTow(dt) // (it was never called before: the tow trucks never showed up)
	out := g.amb[:0]
	for _, a := range g.amb {
		g.driveAmbient(a, dt)
		a.x += a.vx * dt
		a.phase += dt
		if a.kind == ambTank {
			g.tankCrush(a)
		}
		if a.kind == ambCar || a.kind == ambBike || a.kind == ambObj {
			g.runOver(a)
		}
		if a.kind == ambTow && !a.loaded {
			for i, w := range g.wrecks {
				if math.Abs(w.X-a.x) < 10 && math.Abs(w.Y-a.y) < 6 {
					g.wrecks = append(g.wrecks[:i], g.wrecks[i+1:]...)
					a.loaded = true
					break
				}
			}
		}
		if a.x < -140 || a.x > 1100 || a.dead {
			continue
		}
		out = append(out, a)
	}
	g.amb = out
}

// native-resolution sprites for the ambient jokes
func genTank() *pixBuf {
	b := newPix(56, 30)
	b.ox, b.oy = 1, 1
	b.rect(2, 16, 46, 9, c(0x2a3a22))
	for x := 4; x < 46; x += 6 {
		b.ellipse(float64(x)+2, 21, 3, 3, c(0x14180e))
		b.ellipse(float64(x)+2, 21, 1.4, 1.4, c(0x6a7a58))
	}
	b.rect(4, 9, 40, 9, c(0x4a6a3a))
	b.rect(4, 9, 40, 2, c(0x6a8a52))
	b.rect(4, 16, 40, 2, c(0x3a5a2c))
	b.rect(14, 2, 22, 8, c(0x4a6a3a))
	b.rect(14, 2, 22, 2, c(0x6a8a52))
	b.rect(-4, 5, 20, 3, c(0x3a5a2c))
	b.rect(-4, 5, 20, 1, c(0x6a8a52))
	b.rect(24, 5, 3, 3, c(0xf4f4f0))
	b.rect(23, 6, 5, 1, c(0xf4f4f0))
	b.rect(38, 3, 1, 5, c(0x2a3a22))
	return b
}

func genUFO() *pixBuf {
	b := newPix(34, 18)
	b.ox, b.oy = 1, 1
	b.ellipse(16, 9, 16, 4.5, c(0x9aa4b8))
	b.ellipse(16, 8, 14, 3, c(0xc8d0e0))
	b.ellipse(16, 5, 7, 5, c(0x70e0c0))
	b.ellipse(15, 4, 4, 3, c(0xb0f4e0))
	for i := 0; i < 5; i++ {
		b.rect(4+i*6, 11, 2, 2, c([]uint32{0xff4090, 0xf0e040, 0x40e0ff, 0xff4090, 0xf0e040}[i]))
	}
	b.rect(10, 13, 12, 2, c(0x6a7488))
	return b
}

func genCuffs() *pixBuf {
	b := newPix(14, 8)
	b.ellipse(3.5, 4, 3, 3, c(0xc8ccd8))
	b.ellipse(3.5, 4, 1.6, 1.6, color0())
	b.ellipse(10.5, 4, 3, 3, c(0xc8ccd8))
	b.ellipse(10.5, 4, 1.6, 1.6, color0())
	b.rect(6, 3, 2, 2, c(0x8a8e9c))
	return b
}

func color0() color.RGBA { return color.RGBA{} }

// testLevel plays a level from the editor; Escape returns to the editor.
func (g *Game) testLevel(name string) {
	g.store = loadAssetStore()
	if g.store != nil {
		applyObjects(g.store.Objs)
	}
	g.as.props = buildPropImages()
	g.rigCache = map[*CharAsset]*CharSet{}
	g.setupCast()
	g.rebuildLevels()
	idx := g.levelIndexByName(name)
	g.fromEditor = true
	ebiten.SetWindowSize(gameWinW, gameWinH)
	ebiten.SetWindowTitle("Party Hardy - probando escenario (Esc vuelve al editor)")
	g.startLevel(idx)
}

func (g *Game) hasAlertPos() bool { return g.alertPos != (Vec{}) }

// assetVersion is bumped when the built-in content changes in a way that needs a fresh export.
const assetVersion = 8

// migrateAssets moves an outdated assets folder aside (nothing is deleted) so that fresh defaults are exported.
func migrateAssets() {
	root := assetsRoot(false)
	if root == "" {
		return
	}
	data, _ := os.ReadFile(filepath.Join(root, "VERSION"))
	v := 0
	fmt.Sscan(string(data), &v)
	if v >= assetVersion {
		return
	}
	backup := fmt.Sprintf("%s_backup_v%d_%d", root, v, time.Now().Unix())
	if err := os.Rename(root, backup); err == nil {
		fmt.Println("assets: old content moved to", backup)
	}
}

// jumpWindow hops the player to the opposite side of a window along its thin axis.
func (g *Game) jumpWindow(w Rect) {
	p := g.p
	c := w.Center()
	var target Vec
	if w.W >= w.H { // horizontal window: cross vertically
		off := w.H/2 + 12
		if p.pos.Y < c.Y {
			target = Vec{p.pos.X, c.Y + off}
		} else {
			target = Vec{p.pos.X, c.Y - off}
		}
	} else {
		off := w.W/2 + 12
		if p.pos.X < c.X {
			target = Vec{c.X + off, p.pos.Y}
		} else {
			target = Vec{c.X - off, p.pos.Y}
		}
	}
	p.pos = g.lvl.NearestFree(target)
	g.burst(p.pos, 6, 30, color.RGBA{200, 220, 230, 255}, 0.4, 2, 0)
	g.say("¡Saltas por la ventana!")
}

// trafficNear reports whether a vehicle is close enough to the point to hit someone crossing there.
func (g *Game) trafficNear(p Vec) bool {
	for _, a := range g.amb {
		if !a.dead && math.Abs(a.x-p.X) < 170 {
			return true
		}
	}
	return false
}

// runOver: vehicles on the road kill whoever is standing in their lane. People who cross
// without looking (and the player) can be hit; it never raises the criminality.
func (g *Game) runOver(a *Ambient) {
	for _, gu := range g.guests {
		if gu.out() {
			continue
		}
		if math.Abs(gu.pos.X-a.x) < 26 && math.Abs(gu.pos.Y-a.y) < 11 {
			g.killGuest(gu, false)
			g.sayNews("kill", map[string]string{"actor": "Un coche", "victim": guestLabel(gu)})
			g.snd.Play("thud")
		}
	}
	g.eachPlayer(func(p *Player) {
		if !p.dead && !p.hidden && math.Abs(p.pos.X-a.x) < 24 && math.Abs(p.pos.Y-a.y) < 10 {
			g.die("Te ha atropellado un coche")
		}
	})
}

func ambLength(a *Ambient) float64 {
	switch a.kind {
	case ambBike:
		return 30
	case ambTank:
		return 76
	case ambHorse:
		return 60
	case ambObj:
		if k := propKinds[a.name]; k != nil {
			return float64(k.W)
		}
	}
	return 66
}

// roadKind: things that drive along a lane and must queue behind each other.
func roadKind(k int) bool {
	return k == ambCar || k == ambBike || k == ambObj || k == ambTank || k == ambHorse
}

// driveAmbient sets this vehicle's speed for the frame: it follows the traffic
// ahead of it in its lane (cars never drive through each other), and the few
// special vehicles do their stunts.
func (g *Game) driveAmbient(a *Ambient, dt float64) {
	if !roadKind(a.kind) || a.v0 == 0 {
		return
	}
	if a.kind == ambTank || a.kind == ambObj && a.name == "monstertruck" {
		// the tank (and the monster truck) ignores everything and drives through all traffic
		a.vx = a.v0
		g.specialStunt(a, dt, new(float64))
		return
	}
	dir := math.Copysign(1, a.v0)
	allowed := math.Abs(a.v0)
	gapTo := func(x, length float64) float64 { return (x-a.x)*dir - (length+a.length)/2 }
	for _, o := range g.amb {
		// a vehicle that is stopped (unloading, parked) is simply driven through
		if o == a || o.dead || !roadKind(o.kind) || o.v0 == 0 || math.Abs(o.vx) < 1 || math.Abs(o.y-a.y) > 8 || math.Copysign(1, o.v0) != dir {
			continue
		}
		if gp := gapTo(o.x, o.length); gp > -4 && gp < 70 {
			if gp < 14 {
				allowed = 0
			} else {
				allowed = math.Min(allowed, math.Max(math.Abs(o.vx), 0)*0.95)
			}
		}
	}
	if a.wait > 0 {
		a.wait -= dt
		allowed = 0
	}
	g.specialStunt(a, dt, &allowed)
	a.vx = dir * allowed
}

// monsterCrush: the truck drives over the cars in its lane and over whoever is
// in the way, leaving the flattened cars burning for the tow trucks.
func (g *Game) monsterCrush(a *Ambient, dt float64) {
	if a.phase > 0 && int(a.phase*3)%9 == 0 && g.rng.Float64() < dt*8 {
		g.burst(Vec{a.x - math.Copysign(30, a.v0), a.y}, 1, 30, color.RGBA{60, 60, 66, 220}, 0.5, 3, -20)
	}
	a.hop = math.Max(0, a.hop-dt*40)
	for _, o := range g.amb {
		if o == a || o.dead || !(o.kind == ambCar || o.kind == ambBike || o.kind == ambObj && o.name != "monstertruck") {
			continue
		}
		if math.Abs(o.y-a.y) < 10 && math.Abs(o.x-a.x) < 46 {
			o.dead = true
			a.hop = 14
			g.snd.Play("thud")
			g.explodeAt(o.x, o.y-6)
			g.wrecks = append(g.wrecks, Vec{o.x, o.y})
			g.say("¡El monster truck aplasta un coche!")
		}
	}
}

// boostCar: the Rocket League cars leave a trail of boost and now and then
// jump (the flip is only the hop; there is no rotation).
func (g *Game) boostCar(a *Ambient, dt float64) {
	col := rgb(0x4ab0ff)
	if a.col%2 == 1 {
		col = rgb(0xffa030)
	}
	rear := -math.Copysign(30, a.v0)
	if g.rng.Float64() < dt*40 {
		g.burst(Vec{a.x + rear, a.y - 7}, 1, 60, col, 0.3, 3, 0)
	}
	if !a.done && (a.x-a.stopX)*math.Copysign(1, a.v0) >= 0 {
		a.done = true
		a.t2 = 0.001
		g.bubbleAt(Vec{a.x, a.y - 30}, []string{"¡QUÉ SAVE!", "¡WHAT A SAVE!", "¡PAPA VOLÓ!", "¡CENTRANDO!", "¡NICE SHOT!"}[g.rng.Intn(5)], 1.4)
	}
	if a.t2 > 0 {
		a.t2 += dt
		a.hop = 52 * math.Sin(math.Min(1, a.t2/1.1)*math.Pi)
		if a.t2 > 1.1 {
			a.t2, a.hop = 0, 0
		}
	}
}

func (g *Game) specialStunt(a *Ambient, dt float64, allowed *float64) {
	if a.kind != ambObj {
		return
	}
	dir := math.Copysign(1, a.v0)
	passed := (a.x-a.stopX)*dir >= 0
	switch a.name {
	case "schoolbus":
		if passed && !a.done {
			a.done = true
			a.wait = 4
			n := 4 + g.rng.Intn(5)
			side := -1.0 // the pavement is on the far side of the road from the lanes' centre line
			if a.y < g.lvl.Def.CarFrom.Y {
				side = -1
			} else {
				side = 1
			}
			for i := 0; i < n; i++ {
				g.addGuestAt(Vec{a.x + float64(i)*10 - 30, a.y + side*34 + side*float64(i%2)*8})
			}
			g.say("El autobús escolar trae más invitados: +" + itoa(n))
			g.snd.Play("alert")
		}
	case "taxi":
		if passed && !a.done {
			a.done = true
			if g.rng.Float64() < 0.65 && g.guestRoom() >= 1 {
				a.wait = 3
				n := 1 + g.rng.Intn(3)
				side := 1.0
				if a.y < g.lvl.Def.CarFrom.Y {
					side = -1
				}
				for i := 0; i < n; i++ {
					g.addGuestAt(Vec{a.x + float64(i)*11 - 12, a.y + side*34 + side*float64(i%2)*8})
				}
				g.say("Un taxi deja " + itoa(n) + " invitado(s) en la puerta")
			}
		}
	case "monstertruck":
		g.monsterCrush(a, dt)
	case "octane", "fennec", "dominus":
		g.boostCar(a, dt)
	case "delorean":
		if passed && !a.done {
			a.done = true
			a.t2 = 0.001
			g.say("¡88 millas por hora!")
		}
		if a.t2 > 0 {
			a.t2 += dt
			if a.col%2 == 0 {
				// full speed, leaving two lines of fire where the wheels were, then gone in a flash
				*allowed = math.Abs(a.v0) + a.t2*420
				g.fireTrail(a, dt)
				if *allowed > 760 || a.x < -20 || a.x > 980 {
					g.burst(Vec{a.x, a.y - 10}, 24, 120, rgb(0xffffff), 0.5, 3, 0)
					g.burst(Vec{a.x, a.y - 10}, 12, 80, rgb(0x60b0ff), 0.6, 2, 0)
					g.snd.Play("zap")
					a.dead = true
				}
			} else {
				// takes off: the wheels fold and it climbs into the sky, with fire underneath
				*allowed = math.Abs(a.v0) * 0.9
				a.y -= 20 * a.t2 * dt * 3
				g.fireTrail(a, dt)
				if a.y < g.lvl.Def.CarFrom.Y-170 {
					g.burst(Vec{a.x, a.y}, 20, 100, rgb(0xffffff), 0.5, 3, 0)
					a.dead = true
				}
			}
		}
	}
}

type fireTrail struct {
	x, y float64
	t    float64
}

// fireTrail leaves two short flames at the car's wheels; purely visual.
func (g *Game) fireTrail(a *Ambient, dt float64) {
	for _, off := range [2]float64{-14, 12} {
		g.trails = append(g.trails, fireTrail{a.x + off*math.Copysign(1, -a.v0)*-1, a.y - 2, 0})
	}
	g.burst(Vec{a.x, a.y - 2}, 2, 50, rgb(0xff8020), 0.4, 2, 0)
}

// mouseMenu lets the mouse pick menu rows: moving over a row selects it, a left click confirms.
// pick maps a pointer position to a row (or -1). It returns true on a click over a row.
func (g *Game) mouseMenu(pick func(mx, my float64) int, sel *int) bool {
	x, y := ebiten.CursorPosition()
	moved := x != g.lastMX || y != g.lastMY
	g.lastMX, g.lastMY = x, y
	i := pick(float64(x), float64(y))
	if i < 0 {
		return false
	}
	if moved {
		*sel = i
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		*sel = i
		return true
	}
	return false
}

// lawActive: is any police officer or unit still on the scene or on the way?
func (g *Game) lawActive() bool {
	if !g.policeCalled || g.pendingCar >= 0 || g.pendingUnit >= 0 {
		return true
	}
	for _, c := range g.cops {
		if !c.medic && c.state != cLeave {
			return true
		}
	}
	for _, v := range g.veh {
		if (v.kind == 0 || v.kind == 2) && v.state != 2 {
			return true
		}
	}
	return false
}

// otherCuffed: somebody else is being handcuffed right now (the killer, or a
// suspect being taken), so the player's own cuffs mark must not show too.
func (g *Game) otherCuffed() bool {
	if g.slasher != nil && g.slasher.state == slCuffed {
		return true
	}
	for _, c := range g.cops {
		if c.state == cArrest {
			return true
		}
	}
	return false
}

// trimGuests enforces the people limit at the start of a party: ordinary
// guests are dropped at random; specials (performers, groups, lifeguards) stay.
func (g *Game) trimGuests() {
	limit := g.startTarget
	for len(g.guests) > limit {
		var cand []int
		for i, gu := range g.guests {
			if !gu.keep {
				cand = append(cand, i)
			}
		}
		if len(cand) == 0 {
			return
		}
		i := cand[g.rng.Intn(len(cand))]
		g.guests = append(g.guests[:i], g.guests[i+1:]...)
	}
	// the groups and special guests of the start never take the party over the cap either
	for g.guestLive() > g.guestCap() {
		cut := -1
		for i := len(g.guests) - 1; i >= 0; i-- {
			if !g.guests[i].out() && g.guests[i].perf == "" {
				cut = i
				break
			}
		}
		if cut < 0 {
			break
		}
		g.guests = append(g.guests[:cut], g.guests[cut+1:]...)
	}
}

// hasProgress: is there anything a new game would erase?
func (g *Game) hasProgress() bool {
	p := g.progress
	return p.Unlocked > 1 || p.Wallet > 0 || len(p.Owned) > 0 || len(p.Equip) > 0
}

// updateConfirm: the "erase everything?" popup. For five seconds only the
// safe button works.
func (g *Game) updateConfirm() {
	g.time += 1.0 / 60
	g.confirmT += 1.0 / 60
	ready := g.confirmT >= 5
	click := g.mouseMenu(func(mx, my float64) int {
		if my >= 330 && my <= 372 {
			if mx >= 250 && mx <= 480 {
				return 0
			}
			if mx >= 500 && mx <= 710 {
				return 1
			}
		}
		return -1
	}, &g.confirmSel)
	if !ready {
		g.confirmSel = 0
	}
	if actRight.just() || actLeft.just() {
		if ready {
			g.confirmSel = 1 - g.confirmSel
		}
	}
	if actBack.just() {
		g.mode = ModeMenu
		return
	}
	if actConfirm.just() || click {
		switch {
		case g.confirmSel == 0:
			g.mode = ModeMenu
		case ready:
			g.wipeProgress() // everything, the cinematics seen too: the intro plays again and the menu has no CONTINUAR
		}
	}
}

func (g *Game) drawConfirm(dst *ebiten.Image) {
	g.drawMenu(dst)
	fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, 170))
	fillRect(dst, 200, 140, 560, 260, rgb(0x1a0a10))
	fillRect(dst, 200, 140, 560, 3, rgb(0xff3030))
	fillRect(dst, 200, 397, 560, 3, rgb(0xff3030))
	g.drawTextC(dst, "¡CUIDADO!", 480, 160, rgb(0xff5050), 3)
	g.drawTextC(dst, "Vas a borrar todo el progreso:", 480, 218, rgb(0xf4f0ff), 1.5)
	g.drawTextC(dst, "fiestas desbloqueadas, puntos y compras.", 480, 244, rgb(0xf4f0ff), 1.5)
	g.drawTextC(dst, "¿Estás seguro de que quieres borrar todo?", 480, 284, rgb(0xf0e060), 1.5)
	ready := g.confirmT >= 5
	btn := func(x, w float64, label string, on, sel bool, col color.RGBA) {
		bg := rgba(0x2a2a34, 255)
		if on {
			bg = shade(col, 0.45)
		}
		fillRect(dst, x, 330, w, 42, bg)
		if sel {
			fillRect(dst, x-3, 327, w+6, 3, rgb(0xffffff))
			fillRect(dst, x-3, 372, w+6, 3, rgb(0xffffff))
		}
		tc := rgb(0x6a6a76)
		if on {
			tc = rgb(0xf4f4f4)
		}
		g.drawTextC(dst, label, x+w/2, 340, tc, 1.4)
	}
	btn(250, 230, "NO, ME HE EQUIVOCADO", true, g.confirmSel == 0, rgb(0x2aa060))
	label := "BORRA TODO"
	if !ready {
		label = fmt.Sprintf("BORRA TODO (%d)", int(math.Ceil(5-g.confirmT)))
	}
	btn(500, 210, label, ready, g.confirmSel == 1 && ready, rgb(0xd03030))
	g.drawTextC(dst, "Esc: volver", 480, 378, rgb(0x8a86a0), 1)
}
