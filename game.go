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
)

var px *ebiten.Image // 1x1 white pixel

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, h)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(px, op)
}

// Vision: everyone only sees in a cone in front of them, up to a medium distance.
const (
	guestRange = 115.0
	guestDot   = 0.3
	guardRange = 100.0
	guardDot   = 0.45
	copRange   = 130.0
	copDot     = 0.3
)

func (g *Game) sees(from Vec, dir Dir, to Vec, rng, minDot float64) bool {
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
)

type Guest struct {
	pos       Vec
	dir       Dir
	cs        *CharSet
	style     int
	perf      string
	state     gState
	next      gState
	dur       float64
	timer     float64
	anim      float64
	curSpd    float64
	moving    bool
	path      []Vec
	icon      string
	iconT     float64
	dead      bool
	gone      bool
	poison    float64
	scanT     float64
	tgt       *Guest
	tgtPlayer bool
	repath    float64
	stabT     float64
	partner   *Guest
	role      int
	drunk     int
	lookT     float64
	sawKiller bool
	danceEnd  float64
	off       float64
	lastTurn  int
	sawPos    Vec
	fun       int
	runUntil  float64
	sawSlash  bool
	reportTo  *Cop
}

func (gu *Guest) out() bool { return gu.dead || gu.gone }

func (gu *Guest) walk(speed, dt float64) bool {
	m, arr := follow(&gu.pos, &gu.dir, &gu.path, speed, dt)
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
}

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
	maxSearch float64
	wait      float64
	alt       int
	bagN      int
	kind      int // 0 police, 1 swat, 2 fbi
	shootT    float64
	stuckT    float64
	stuckPos  Vec
	stuckChk  float64
}

type Vehicle struct {
	kind    int // 0 police car, 1 ambulance, 2 police heli, 3 medical heli
	x, y    float64
	tx, ty  float64
	state   int // 0 arriving, 1 parked, 2 leaving
	spawned bool
	rot     float64
	pad     int
	alt     int
	hold    float64
	unit    int // 0 police, 1 swat, 2 fbi
	dead    bool
}

type Body struct {
	pos      Vec
	cs       *CharSet
	dir      Dir
	fallT    float64
	bagged   bool
	reported bool
	carried  bool
	claimed  *Cop
	seers    []*Guest // people who saw it but have not warned anyone yet
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
	pos      Vec
	dir      Dir
	cs       *CharSet
	style    int
	anim     float64
	moving   bool
	hidden   bool
	hideProp *Prop
	carrying *Body
	cooldown float64
	stabT    float64
	stamina  float64
	dead     bool
	knockT   float64
	danceT   float64
	running  bool
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
	playerCS       *CharSet
	copCS          *CharSet
	medicCS        *CharSet
	guardCS        []*CharSet
	specialGuardCS []*CharSet
	copAlt         []*CharSet
	swatCS         *CharSet
	fbiCS          []*CharSet
	killerCS       []*CharSet
	heliOX, heliOY float64
	animals        AnimalSprites
	bag            *ebiten.Image
	heli           *ebiten.Image
}

// ---------------------------------------------------------------------------
// Game
// ---------------------------------------------------------------------------

type Game struct {
	streakT       float64
	streaked      bool
	djDead        bool
	songIdx       int
	nextSong      float64
	wrecks        []Vec
	respKind      int
	swatKills     int
	baseN         int
	arrivals      int
	arrivalT      float64
	elevators     []*Elevator
	slasher       *Slasher
	slasherT      float64
	slasherDone   bool
	backupCalled  bool
	pendingBackup float64
	mode          Mode
	as            *Assets
	lvl           *Level
	levelIdx      int
	snd           *Sounds
	face          text.Face
	rng           *rand.Rand
	tick          int
	time          float64

	p       *Player
	guests  []*Guest
	guards  []*Guard
	cops    []*Cop
	veh     []*Vehicle
	bodies  []*Body
	decals  []Decal
	parts   []Particle
	amb     []*Ambient
	animals []*Animal
	ambT    float64

	wanted       float64
	seenNow      bool
	policeCalled bool
	pendingCar   float64
	pendingAmb   float64
	ambCalled    bool
	points       int
	credit       bool
	titleT       float64
	callerGuest  *Guest
	callerGuard  *Guard
	suspect      *Guest
	alertPos     Vec
	killPos      Vec
	murderT      float64
	shake        float64
	flash        float64

	cam        Vec
	camI       Vec
	msg        string
	msgT       float64
	loseReason string
	kills      int
	total      int
	endTime    float64
	endAt      float64

	store         *AssetStore
	ed            *Editor
	rigCache      map[*CharAsset]*CharSet
	pendingExport bool
	fromEditor    bool
	testVictim    *Guest
	menuSel       int
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
	}
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
	if !noAssets {
		g.store = loadAssetStore()
	}
	if g.store != nil {
		applyObjects(g.store.Objs)
	}
	as := &Assets{props: buildPropImages(), floors: map[string][]*ebiten.Image{}}
	for _, k := range []string{"wood", "deck", "tile", "carpet", "grass", "pave", "asphalt", "concrete", "marble", "parquet", "sand", "dirt", "rooftop", "rooftop2", "boardwalk", "cobble", "pitch", "white", "arena", "mall"} {
		for v := 0; v < 3; v++ {
			as.floors[k] = append(as.floors[k], genFloor(k, v).img())
		}
	}
	as.glow = genGlow(96).img()
	as.vignette = genVignette().img()
	as.bag = genBodyBag().scaled(wk).outlinedSel(0.45).img()
	as.heli = genHeli().scaled(wk).outlinedSel(0.45).img()
	as.heliOX, as.heliOY = math.Round(41*wk+1), math.Round(34*wk+1)
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
				as.heli = oa.Frames[0].img()
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
	g.levelIdx = i
	def := levelDefs[i]
	g.lvl = buildLevel(def)
	g.renderBG()
	g.time = 0
	g.p = &Player{pos: def.Start, dir: DirUp, cs: g.as.playerCS, stamina: 6, style: g.rng.Intn(4)}
	g.guests, g.guards, g.cops, g.veh, g.bodies, g.decals, g.parts, g.amb, g.wrecks = nil, nil, nil, nil, nil, nil, nil, nil, nil
	g.ambT = 6 + g.rng.Float64()*10
	g.spawnAnimals()
	g.wanted, g.kills = 0, 0
	g.policeCalled, g.pendingCar, g.pendingAmb, g.ambCalled = false, -1, -1, false
	g.points, g.titleT = 0, 4.5
	g.callerGuest, g.callerGuard, g.suspect = nil, nil, nil
	g.murderT = 80 + g.rng.Float64()*80
	g.slasher, g.slasherDone, g.slasherT = nil, false, 55+g.rng.Float64()*75
	g.backupCalled, g.pendingBackup = false, -1
	g.streakT, g.streaked = 55+g.rng.Float64()*90, false
	g.respKind, g.swatKills = 0, 0
	g.djDead, g.songIdx, g.nextSong = false, 0, 70+g.rng.Float64()*40
	g.arrivals, g.arrivalT, g.elevators = 0, 40+g.rng.Float64()*30, nil
	if fastEv {
		g.slasherT, g.arrivalT = 6, 14
		g.ambT = 3
	}
	g.msg, g.msgT, g.shake, g.flash = "", 0, 0, 0
	for k := 0; k < def.Guests*14/10; k++ {
		var pos Vec
		for tries := 0; tries < 40; tries++ {
			pos = g.lvl.RandomRoam(g.rng)
			if dist(pos, def.Start) > 140 {
				break
			}
		}
		gu := &Guest{pos: pos, dir: Dir(g.rng.Intn(4)), cs: g.newGuestCS(), style: g.rng.Intn(4), off: g.rng.Float64() * 6, state: gIdle, timer: g.rng.Float64() * 3, role: k % 2}
		g.guests = append(g.guests, gu)
	}
	for _, pf := range def.Performers {
		cs := g.newGuestCS()
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
			cs = newCharSet(&spec)
		}
		gu := &Guest{pos: pf.Pos, dir: pf.Dir, cs: cs, perf: pf.Role, state: gPerform, role: g.rng.Intn(2)}
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
	g.total = len(g.guests) + len(g.guards)
	g.cam = g.camTarget()
}

func (g *Game) startLevel(i int) {
	g.loadLevel(i)
	g.mode = ModePlay
	g.snd.StartMusic()
	g.say(g.lvl.Def.Intro)
}

func (g *Game) say(s string) { g.msg, g.msgT = s, 4.5 }

// aliveCount: guests and guards left to kill.
func (g *Game) aliveCount() int {
	n := 0
	for _, gu := range g.guests {
		if !gu.out() {
			n++
		}
	}
	for _, gd := range g.guards {
		if !gd.dead {
			n++
		}
	}
	if s := g.slasher; s != nil && (s.state == slHunt || s.state == slArrive) {
		n++
	}
	return n
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (g *Game) Update() error {
	g.tick++
	g.syncMusic()
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
	switch g.mode {
	case ModeMenu:
		g.time += 1.0 / 60
		n := len(levelDefs)
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.menuSel = (g.menuSel + 1) % n
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.menuSel = (g.menuSel + n - 1) % n
		}
		for i, k := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
			if i < n && inpututil.IsKeyJustPressed(k) {
				g.startLevel(i)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.startLevel(g.menuSel)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.startLevel(g.rng.Intn(n))
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
		if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
			g.openEditor()
		}
		for _, gu := range g.guests {
			g.updateGuest(gu, 1.0/60)
		}
		if !g.camFixed {
			c := g.lvl.Def
			g.cam = Vec{0, 0}
			_ = c
		}
	case ModePlay:
		g.updatePlay(1.0 / 60)
	case ModeLevelDone:
		g.time += 1.0 / 60
		g.updateParticles(1.0 / 60)
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.startLevel(g.levelIdx + 1)
		}
	case ModeFinal:
		g.time += 1.0 / 60
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
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
			if inpututil.IsKeyJustPressed(ebiten.KeyR) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
				g.startLevel(g.levelIdx)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
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
		fmt.Printf("selftest ok: level=%d time=%.1f left=%d cops=%v cars=%d guards=%v wanted=%.1f called=%v mode=%d bodies=%d states=%v msg=%q\n",
			g.levelIdx, g.time, g.aliveCount(), cs, len(g.veh), gs, g.wanted, g.policeCalled, g.mode, len(g.bodies), st, g.msg)
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
	if g.msgT > 0 {
		g.msgT -= dt
	}
	if g.shake > 0 {
		g.shake -= dt
	}
	if g.flash > 0 {
		g.flash -= dt
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
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
	g.updatePlayer(dt)
	g.seenNow = false
	g.updateMurder(dt)
	g.updateSlasher(dt)
	g.updateArrivals(dt)
	g.updateStreaker()
	for _, gu := range g.guests {
		g.updateGuest(gu, dt)
	}
	for _, gd := range g.guards {
		g.updateGuard(gd, dt)
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
	g.maybeAmbulance()
	g.maybeBackup()
	if !g.seenNow && g.wanted > 0 {
		g.wanted -= dt
	}
	g.updateVehicles(dt)
	g.updateAmbient(dt)
	g.updateAnimals(dt)
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
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.interact()
	}
	if p.hidden {
		p.moving = false
		return
	}
	var dx, dy float64
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		dx--
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		dx++
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		dy--
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		dy++
	}
	p.moving = dx != 0 || dy != 0
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && p.carrying == nil && p.danceT <= 0 {
		p.danceT = 4.5
		g.say("¡A bailar!")
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
	speed := 66.0
	p.running = false
	running := ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)
	if p.moving && running && p.stamina > 0 {
		speed = 106
		p.running = true
		p.stamina -= dt
	} else if p.stamina < 6 {
		p.stamina += dt * 0.9
	}
	if p.carrying != nil {
		speed *= 0.75
	}
	if g.lvl.InPool(p.pos) {
		speed *= 0.6
		p.running = false
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
		nx := p.pos.X + dx*speed*dt
		if !g.lvl.SolidP(nx, p.pos.Y) {
			p.pos.X = nx
		}
		ny := p.pos.Y + dy*speed*dt
		if !g.lvl.SolidP(p.pos.X, ny) {
			p.pos.Y = ny
		}
		p.anim += dt * stridePhase(speed)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		g.stab()
	}
}

// danceRoll: nearby people react to the player dancing.
func (g *Game) danceRoll() {
	p := g.p
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
		case r < 0.55:
			side := 1.0
			if g.rng.Intn(2) == 0 {
				side = -1
			}
			t := Vec{p.pos.X + side*(22+g.rng.Float64()*10), p.pos.Y + (g.rng.Float64()-0.5)*14}
			gu.path = g.lvl.Path(gu.pos, t)
			gu.state, gu.danceEnd = gDanceWith, g.time+p.danceT+2+g.rng.Float64()*3
			gu.icon, gu.iconT = "WOO!", 1.5
		case r < 0.80:
		case r < 0.97:
			gu.icon, gu.iconT = "NAH", 1.6
			gu.path = g.lvl.Path(gu.pos, g.lvl.RandomRoam(g.rng))
			gu.state, gu.next, gu.dur = gGoto, gIdle, 1
		default:
			gu.icon, gu.iconT = "HEY!", 2
			gu.state, gu.timer = gPunch, 12
		}
	}
}

func (g *Game) stab() {
	p := g.p
	if p.cooldown > 0 || p.hidden || p.dead || p.knockT > 0 {
		return
	}
	if p.carrying != nil {
		g.say("Suelta el cuerpo primero (E)")
		return
	}
	p.cooldown = 0.45
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
	if s := g.slasher; s != nil && s.state == slHunt && dist(p.pos, s.pos) < math.Min(bd, 28) {
		g.hitSlasher()
		return
	}
	switch {
	case bestG != nil:
		g.killGuest(bestG, true)
	case bestD != nil:
		g.killGuard(bestD)
	}
}

func (g *Game) spawnBody(pos Vec, cs *CharSet, dir Dir) {
	g.bodies = append(g.bodies, &Body{pos: pos, cs: cs, dir: dir, fallT: 0.45})
	g.decals = append(g.decals, Decal{pos: pos, kind: 0, r: 9 + g.rng.Float64()*4})
	g.burst(Vec{pos.X, pos.Y - 14}, 18, 80, rgb(0xb01818), 0.7, 2, 220)
}

func (g *Game) killGuest(gu *Guest, byPlayer bool) {
	if gu.out() {
		return
	}
	gu.dead = true
	gu.icon = ""
	if gu.perf == "dj" && !g.djDead {
		g.djDead = true
		g.songIdx = g.rng.Intn(5)
		g.nextSong = g.time + 55 + g.rng.Float64()*30
		g.say("¡Han matado al DJ! Ahora suena punk rock")
	}
	g.spawnBody(gu.pos, gu.cs, gu.dir)
	if gu.partner != nil && gu.partner.partner == gu {
		gu.partner.partner = nil
		if gu.partner.state == gChat || gu.partner.state == gKiss || gu.partner.state == gWaitP {
			gu.partner.state, gu.partner.timer = gIdle, 1
		}
	}
	if byPlayer {
		g.snd.Play("stab")
		g.kills++
		g.points += 30
		g.witness()
	} else {
		g.snd.Play("thud")
		if g.credit {
			g.points += 50
		}
	}
	g.checkDone()
}

func (g *Game) killGuard(gd *Guard) {
	if gd.dead {
		return
	}
	gd.dead = true
	gd.icon = ""
	g.spawnBody(gd.pos, gd.cs, gd.dir)
	g.snd.Play("stab")
	g.kills++
	g.points += 50
	g.witness()
	g.checkDone()
}

func (g *Game) checkDone() {
	if g.aliveCount() == 0 && g.mode == ModePlay {
		g.endTime = g.time
		g.snd.Play("win")
		if g.levelIdx+1 < len(levelDefs) {
			g.mode = ModeLevelDone
		} else {
			g.mode = ModeFinal
		}
	}
}

func (g *Game) lose(reason string) {
	if g.mode != ModePlay || g.god {
		return
	}
	g.mode = ModeLose
	g.loseReason = reason
	g.endTime = g.time
	g.endAt = g.time
	g.snd.Play("lose")
	g.shake = 0.4
}

func (g *Game) die(reason string) {
	if g.god {
		return
	}
	g.p.dead = true
	g.p.hidden = false
	g.burst(Vec{g.p.pos.X, g.p.pos.Y - 14}, 22, 90, rgb(0xb01818), 0.8, 2, 220)
	g.decals = append(g.decals, Decal{pos: g.p.pos, kind: 0, r: 11})
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
	var seers []*Guest
	for _, gu := range g.guests {
		if g.canWatch(gu) && g.sees(gu.pos, gu.dir, p.pos, guestRange, guestDot) {
			gu.sawKiller = true
			gu.sawPos = p.pos
			seers = append(seers, gu)
		}
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
		if !gd.dead && g.sees(gd.pos, gd.dir, p.pos, guardRange*1.4, -0.1) {
			gd.sight = 99
		}
	}
	for _, c := range g.cops {
		if g.sees(c.pos, c.dir, p.pos, copRange, copDot) {
			g.wanted = math.Max(g.wanted, 45)
		}
	}
}

func (g *Game) alarm(gu *Guest, txt string) {
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
	g.snd.Play("alert")
}

func (g *Game) callerAvailable() bool {
	if g.policeCalled && g.copOnScene() {
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
	g.alarm(gu, "!?")
}

// --- interaction -------------------------------------------------------------

type candidate struct {
	d  float64
	fn func()
}

func (g *Game) interact() {
	p := g.p
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
					p.pos = Vec{pr.X, pr.Y - 4}
					g.say("Escondido. Pulsa E para salir")
				}})
			}
		case k.Phone && !pr.used && p.carrying == nil:
			cs = append(cs, candidate{d, func() {
				pr.used = true
				g.snd.Play("spark")
				g.say("Has cortado el cable del teléfono")
			}})
		case k.Poison && p.carrying == nil:
			cs = append(cs, candidate{d, func() {
				if pr.armT > 0 {
					g.say("El ponche ya está envenenado")
					return
				}
				pr.armT = 40
				g.snd.Play("splash")
				g.say("Has envenenado el ponche")
			}})
		case k.Trap != "" && !pr.used && pr.armT == 0 && p.carrying == nil:
			cs = append(cs, candidate{d, func() {
				pr.armT = 3
				if k.Trap == "electric" {
					pr.armT = 1.5
				}
				g.say("Trampa activada. ¡Aléjate!")
			}})
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
		if pr.armT > 0 {
			pr.armT -= dt
			if int(pr.armT*8)%2 == 0 {
				g.burst(Vec{pr.X, pr.Y - 16}, 1, 30, rgb(0xffd040), 0.3, 1, 60)
			}
			if pr.armT <= 0 {
				pr.armT = 0
				g.triggerTrap(pr)
			}
		}
	}
}

func (g *Game) triggerTrap(pr *Prop) {
	pr.used = true
	g.credit = true
	defer func() { g.credit = false }()
	centre := Vec{pr.X, pr.Y - 8}
	switch pr.K.Trap {
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
				g.killGuest(gu, false)
			}
		}
		for _, gd := range g.guards {
			if !gd.dead && dist(gd.pos, centre) < 64 {
				gd.dead = true
				g.points += 50
				g.spawnBody(gd.pos, gd.cs, gd.dir)
			}
		}
		if !g.p.dead && dist(g.p.pos, centre) < 58 {
			g.die("Has saltado por los aires con tu propia trampa")
		}
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
				g.killGuest(gu, false)
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
	case r >= 0.90:
		g.guestFun(gu)
		return
	}
	if !ok {
		target = g.lvl.RandomRoam(g.rng)
	}
	gu.path = g.lvl.Path(gu.pos, target)
	gu.state, gu.next, gu.dur = gGoto, next, dur
}

const guestSpeed = 26.0

func (g *Game) updateGuest(gu *Guest, dt float64) {
	if gu.out() {
		return
	}
	gu.moving = false
	if gu.stabT > 0 {
		gu.stabT -= dt
	}
	if gu.iconT > 0 {
		gu.iconT -= dt
		if gu.iconT <= 0 {
			gu.icon = ""
		}
	}
	if gu.poison > 0 {
		gu.poison -= dt
		if gu.poison <= 0 {
			g.credit = true
			g.killGuest(gu, false)
			g.credit = false
			return
		}
	}
	if g.mode == ModePlay {
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
		}
		if gu.timer <= 0 {
			g.guestPick(gu)
		}
	case gGoto:
		if gu.walk(guestSpeed, dt) {
			gu.state, gu.timer = gu.next, gu.dur
			switch gu.state {
			case gDance:
				gu.dir = DirDown
			case gDrink, gToilet, gLounge:
				gu.dir = DirUp
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
			gu.path = g.lvl.Path(gu.pos, p.pos)
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
			for _, pr := range g.lvl.Props {
				if pr.K.Poison && pr.armT > 0 && distRect(gu.pos, pr.Foot) < 30 && gu.poison <= 0 {
					gu.poison = 4 + g.rng.Float64()*5
				}
			}
			gu.state, gu.timer = gIdle, 1.5
		}
	case gToilet, gLounge:
		gu.timer -= dt
		if gu.timer <= 0 {
			gu.state, gu.timer = gIdle, 0
		}
	case gVomit:
		gu.timer -= dt
		if gu.timer <= 0 {
			g.decals = append(g.decals, Decal{pos: Vec{gu.pos.X, gu.pos.Y + 4}, kind: 2, r: 8})
			if g.rng.Float64() < 0.6 {
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
			} else if c := g.nearCop(gu.pos); c != nil && (gu.sawKiller || gu.sawSlash || g.knowsAny(gu)) {
				gu.state, gu.reportTo, gu.repath = gReport, c, 0
				gu.timer = 14
				gu.icon, gu.iconT = "!!", 1.2
			} else {
				g.scare(gu)
			}
		}
	case gToPhone:
		if gu.walk(guestRunSpeed, dt) {
			if g.phoneSpot(gu.pos) == nil {
				g.guestFlee(gu)
			} else {
				gu.state, gu.timer = gCall, 2.5
				gu.dir = DirUp
				gu.icon, gu.iconT = "911", 3
			}
		}
	case gFlee:
		if gu.walk(guestRunSpeed, dt) {
			gu.state, gu.timer = gCall, 2.5
			gu.icon, gu.iconT = "911", 3
		}
	case gCall:
		gu.timer -= dt
		if gu.timer <= 0 {
			if g.callerGuest == gu {
				g.callerGuest = nil
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
			gu.path = g.lvl.Path(gu.pos, g.p.pos)
		}
		gu.walk(80, dt)
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
	case gScared:
		gu.timer -= dt
		if len(gu.path) == 0 {
			gu.path = g.lvl.Path(gu.pos, g.farRoam(gu.pos))
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
			gu.path = g.lvl.Path(gu.pos, g.lvl.RandomRoam(g.rng))
			gu.timer = 3
		}
		gu.walk(82, dt)
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
		gu.path = g.lvl.Path(gu.pos, Vec{pr.X, pr.Y + 14})
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
	gu.path = g.lvl.Path(gu.pos, t)
	gu.state = gFlee
}

func (g *Game) guestScan(gu *Guest) {
	if !g.canWatch(gu) || gu.state == gKiller || gu.state == gPunch {
		return
	}
	for _, b := range g.bodies {
		if b.reported || b.carried {
			continue
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
				g.alarm(gu, "?!")
			}
			return
		}
	}
	p := g.p
	if p.carrying != nil && !p.hidden && g.sees(gu.pos, gu.dir, p.pos, guestRange, guestDot) {
		gu.sawKiller = true
		if g.callerAvailable() {
			g.becomeCaller(gu)
		} else {
			g.alarm(gu, "!?")
		}
	}
}

// --- rare murders among guests -------------------------------------------------

func (g *Game) updateMurder(dt float64) {
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
		gu.state, gu.timer = gIdle, 2
		return
	}
	gu.timer -= dt
	if gu.timer <= 0 {
		gu.state, gu.timer = gIdle, 2
		return
	}
	gu.repath -= dt
	if gu.repath <= 0 {
		gu.repath = 0.4
		gu.path = g.lvl.Path(gu.pos, tp)
	}
	speed := 46.0
	if gu.tgtPlayer {
		speed = 82
	}
	gu.walk(speed, dt)
	if dist(gu.pos, tp) < 12 {
		gu.stabT = 0.25
		g.snd.Play("stab")
		if gu.tgtPlayer {
			g.die("Un invitado psicópata te ha apuñalado por la espalda")
			return
		}
		v := gu.tgt
		g.killGuest(v, false)
		for _, o := range g.guests {
			if o != gu && g.canWatch(o) && g.sees(o.pos, o.dir, v.pos, guestRange, guestDot) {
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
	p := g.p
	if gd.state != gdBeat && gd.state != gdChase && gd.state != gdReact {
		rng := guardRange
		if gd.special {
			rng *= 1.3
		}
		if g.wanted > 0 || p.carrying != nil {
			rng *= 1.4
		}
		if !p.hidden && !p.dead && g.sees(gd.pos, gd.dir, p.pos, rng, guardDot) {
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
			if !b.reported && !b.carried && g.sees(gd.pos, gd.dir, b.pos, guardRange, guardDot) {
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
		arr := g.guardWalk(gd, 30, dt)
		if arr {
			gd.state, gd.timer = gdWait, 1.5+g.rng.Float64()*2
			gd.dir = Dir(g.rng.Intn(4))
		}
	case gdReturn:
		if g.guardWalk(gd, 44, dt) {
			gd.state, gd.timer = gdWait, 0.5
		}
	case gdInvestigate:
		if g.guardWalk(gd, 42, dt) {
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
		chase := 82.0
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
		arr := g.guardWalk(gd, 40, dt)
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
	if gd.moving {
		gd.anim += dt * stridePhase(gd.curSpd)
	}
}

func (g *Game) guardWalk(gd *Guard, speed, dt float64) bool {
	m, arr := follow(&gd.pos, &gd.dir, &gd.path, speed, dt)
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
		if !b.reported && !b.carried && g.knows(gu, b) {
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
	g.deliver(gu)
	g.raisePolice(gu.pos, informed)
}

// callPoliceGuard: a security guard phones in the bodies he has seen around him.
func (g *Game) callPoliceGuard(gd *Guard) {
	for _, b := range g.bodies {
		if !b.reported && !b.carried && dist(b.pos, gd.pos) < 150 {
			b.reported, b.seers = true, nil
			if g.alertPos == (Vec{}) {
				g.alertPos = b.pos
			}
		}
	}
	g.raisePolice(gd.pos, false)
}

func (g *Game) raisePolice(at Vec, informed bool) {
	if g.policeCalled {
		return
	}
	g.policeCalled = true
	g.snd.Play("ring")
	g.respKind, g.swatKills = 0, 0
	if r := g.rng.Float64(); r < 0.06 {
		g.respKind = 1
	} else if r < 0.12 {
		g.respKind = 2
	}
	if g.alertPos == (Vec{}) {
		g.alertPos = at
	}
	if !informed {
		for _, gu := range g.guests {
			if !gu.out() && gu.sawKiller {
				informed = true
			}
		}
	}
	if informed {
		g.wanted = 70
	} else if g.rng.Float64() < 0.45 {
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
	g.say("¡Alguien ha llamado a la policía!")
}

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
	g.veh = append(g.veh, &Vehicle{kind: k, x: def.CarFrom.X + 40, y: def.CarFrom.Y, tx: def.CarTo.X - float64(n)*100, ty: def.CarFrom.Y, unit: g.unitFor(medic)})
	g.snd.Play("siren")
}

// maybeAmbulance: with many reported bodies one cop is not enough, so a paramedic comes too.
func (g *Game) maybeAmbulance() {
	if g.pendingAmb >= 0 && g.time >= g.pendingAmb {
		g.pendingAmb = -1
		g.dispatch(true)
	}
	if g.ambCalled || g.pendingAmb >= 0 || !g.policeCalled {
		return
	}
	n := 0
	for _, b := range g.bodies {
		if b.reported && !b.carried {
			n++
		}
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
		g.say("Llega una ambulancia a por los cuerpos")
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
		switch v.state {
		case 0:
			if heli {
				v.y += 100 * dt
				if v.y >= v.ty {
					v.y, v.state = v.ty, 1
				}
			} else {
				v.x -= 130 * dt
				if v.x <= v.tx {
					v.x, v.state = v.tx, 1
				}
			}
		case 1:
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
				n := 1
				switch v.unit {
				case 1:
					n = 4
				case 2:
					n = 2
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
				for k := 0; k < n; k++ {
					p2 := g.lvl.NearestFree(Vec{pos.X + float64(k)*10, pos.Y - float64(k%2)*8})
					cop := &Cop{pos: p2, dir: DirDown, veh: v, medic: medic, alt: v.alt, kind: v.unit, maxChase: 3 + g.rng.Float64()*11}
					cop.maxSearch = 32 + g.rng.Float64()*30
					if v.unit > 0 {
						cop.maxSearch += 14
						cop.alt = k
						cop.shootT = 8 + g.rng.Float64()*4
					}
					if g.wanted > 0 {
						cop.maxSearch += 18
					}
					g.cops = append(g.cops, cop)
					if medic {
						g.copNext(cop)
					} else {
						cop.state = cGoto
						if g.hasAlertPos() {
							cop.path = g.lvl.Path(cop.pos, g.alertPos)
						}
					}
				}
			}
		case 2:
			if heli {
				v.y -= 100 * dt
				if v.y < -140 {
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
	m, arr := follow(&c.pos, &c.dir, &c.path, speed, dt)
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
	copWalkSpeed  = 74.0 // police walk faster than a walking player
	copSweepSpeed = 66.0
	copChaseSpeed = 112.0 // a bit faster than a running player: escapable
	medicSpeed    = 62.0
)

// sweepTarget: where the police look next. First around where the killer was last seen.
func (g *Game) sweepTarget(c *Cop) Vec {
	if g.wanted > 0 && c.searchT < 25 && g.killPos != (Vec{}) {
		for i := 0; i < 14; i++ {
			p := g.lvl.RandomRoam(g.rng)
			if dist(p, g.killPos) < 190 {
				return p
			}
		}
	}
	return g.lvl.RandomRoam(g.rng)
}

func (g *Game) updateCop(c *Cop, dt float64) {
	c.moving = false
	if c.iconT > 0 {
		c.iconT -= dt
		if c.iconT <= 0 {
			c.icon = ""
		}
	}
	g.copUnstick(c, dt)
	if g.copSlashHook(c, dt) {
		return
	}
	g.copBrutal(c, dt)
	p := g.p
	// police notice bodies lying around; while hunting the killer they only deal with the ones right next to them
	bodyRange := copRange
	if g.wanted > 0 {
		bodyRange = 75
	}
	for _, b := range g.bodies {
		if !b.reported && !b.carried && !b.bagged && g.sees(c.pos, c.dir, b.pos, copRange, copDot) {
			b.reported = true
			if c.icon == "" {
				c.icon, c.iconT = copPhrases[g.rng.Intn(len(copPhrases))], 2
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
	if !c.medic {
		rng, dot := copRange, copDot
		if chasing {
			rng, dot = 150, -0.3
		}
		sees = !p.hidden && !p.dead && (g.wanted > 0 || p.carrying != nil) && g.sees(c.pos, c.dir, p.pos, rng, dot)
	}
	if sees && c.state != cTired {
		if !chasing {
			c.icon, c.iconT = "YOU'RE UNDER ARREST!", 1.8
			g.snd.Play("alert")
			c.chaseT = 0
			g.releaseCop(c)
		}
		c.state = cChase
		c.lastSeen = p.pos
		g.killPos = p.pos
		c.lost = 0
		g.seenNow = true
		if g.wanted < 20 {
			g.wanted = 20
		}
	}
	if !c.medic && c.state != cLeave && c.state != cChase && c.state != cTired {
		c.searchT += dt
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
			c.icon, c.iconT = "PUFF... PUFF...", 4
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
			g.die("¡La policía te ha atrapado!")
		}
	case cTired:
		c.timer -= dt
		g.copWalk(c, 22, dt)
		if c.timer <= 0 {
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
		speed := 40.0
		if c.medic {
			speed = 46
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
		g.copWalk(c, 84, dt)
		if dist(c.pos, s.pos) < 14 {
			s.gone = true
			g.suspect = nil
			g.say("La policía se lleva a un sospechoso")
			g.checkDone()
			g.copNext(c)
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
					g.ambCalled = false
				}
			} else {
				g.policeCalled = false
				g.suspect = nil
				g.wanted = 0
				for _, b := range g.bodies {
					if !b.carried && !b.bagged {
						b.reported, b.claimed = false, nil
					}
				}
			}
		}
	}
	if c.moving {
		c.anim += dt * stridePhase(c.curSpd)
	}
}

// stridePhase: animation frames per second for a walking or running speed (px/s),
// so feet do not slide on the floor.
func stridePhase(speed float64) float64 {
	if speed > 60 {
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
)

type Ambient struct {
	kind  int
	x, y  float64
	vx    float64
	col   int
	phase float64
	name  string
	dead  bool
}

func (g *Game) updateAmbient(dt float64) {
	g.ambT -= dt
	if g.ambT <= 0 {
		g.ambT = 16 + g.rng.Float64()*26
		def := g.lvl.Def
		a := &Ambient{col: g.rng.Intn(4), phase: g.rng.Float64() * 6}
		speed := 100.0
		if def.Police == "car" {
			a.kind = []int{ambCar, ambCar, ambBike, ambTank, ambHorse}[g.rng.Intn(5)]
			if fastEv {
				a.kind = ambTank
			}
			var custom []string
			for n, k := range propKinds {
				if k.Ambient > 0 {
					custom = append(custom, n)
				}
			}
			sort.Strings(custom)
			if len(custom) > 0 && !fastEv && g.rng.Float64() < 0.6 {
				a.kind, a.name = ambObj, custom[g.rng.Intn(len(custom))]
			}
			a.y = def.CarFrom.Y - 12 + g.rng.Float64()*14
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
		} else {
			a.kind = ambUFO
			a.y = 14 + g.rng.Float64()*26
			speed = 45
		}
		if g.rng.Intn(2) == 0 {
			a.x, a.vx = -100, speed
		} else {
			a.x, a.vx = 1060, -speed
		}
		g.amb = append(g.amb, a)
	}
	out := g.amb[:0]
	for _, a := range g.amb {
		a.x += a.vx * dt
		a.phase += dt
		if a.kind == ambTank {
			g.tankCrush(a)
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
	idx := 0
	if g.store != nil {
		for i, la := range g.store.Levels {
			if la.Meta.Name == name {
				idx = i
			}
		}
	}
	g.fromEditor = true
	ebiten.SetWindowSize(gameWinW, gameWinH)
	ebiten.SetWindowTitle("Party Hard - probando escenario (Esc vuelve al editor)")
	g.startLevel(idx)
}

func (g *Game) hasAlertPos() bool { return g.alertPos != (Vec{}) }

// assetVersion is bumped when the built-in content changes in a way that needs a fresh export.
const assetVersion = 7

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
