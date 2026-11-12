package main

import (
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// WEATHER: two "sometimes" moods of some open-air parties (owner's requests 202 and 203).
//
//   - TORMENTA (storm): in a few open-air parties (stormLevels) a storm may come DURING the party, about one party
//     in 5-6 of those (stormChance), at most once, never at the start (stormAtMin..stormAtMax). It lasts
//     stormDurMin..stormDurMax seconds with a slow fade in and a soft end: fine translucent rain (snow at the ski
//     resort), little splashes, now and then a lightning flash with a far thunder, some guests run for a moment, and
//     VERY rarely (stormStrikeChance of the storms) a bolt falls on a guest: an act of nature (no stars, no crime).
//     "Que se vea bien el nivel": the shade is at most stormShadeA (about 10 %) and the drops are 1 px wide and
//     translucent, never over the players (weather_test.go measures luminance and contrast).
//   - AMANECER (dawn): in a few night open-air parties (dawnLevels), one in three (dawnChance), the night slowly turns
//     into a soft neutral dawn along the party (by time and by the share of guests already out): the night layer of
//     drawLighting gets lighter and less blue, the street lamps dim at the end. Only the drawing changes: cameras,
//     guards and everybody's sight work as always.
//
// Both are decided when the level loads with their OWN dice (weatherDice), so the rest of the party is not moved;
// never in STEALTH, in RETO DE HOY, on the island, in the random-mode extras or in any party not in the lists.
// Nothing is saved. Debug: -storm 1 (storm at ~3 s), -storm 2 (and a bolt on a guest at ~12 s), -dawn 0 (dawn in
// this party), -dawn P (dawn pinned at P %, for captures). Both freeze the progress (debugFlagsActive).
// ---------------------------------------------------------------------------

const (
	stormChance       = 0.18  // a party of stormLevels gets a storm (about 1 in 5.5)
	stormAtMin        = 70.0  // earliest second of the party
	stormAtMax        = 200.0 // latest second it may start
	stormDurMin       = 40.0  // seconds it lasts, at least
	stormDurMax       = 80.0
	stormFadeIn       = 10.0 // seconds to full rain
	stormFadeOut      = 12.0 // seconds of the soft end
	stormStrikeChance = 0.15 // storms (with lightning) that throw a bolt on a guest: about 1 party in 37 of the list
	stormShadeA       = 0.07 // the darkest the storm makes the frame (alpha of its cool shade)
	stormFlashA       = 0.18 // the brightest a lightning flash gets (alpha of its pale wash)
	stormFlashDur     = 0.32 // seconds of a flash (two quick pulses)
	stormBoltMin      = 7.0  // seconds between flashes
	stormBoltMax      = 15.0
	stormBoltTop      = 112.0 // the bolt starts below the HUD band (panels, stars, the news line): never over a text
	stormSafeR        = 48.0  // a struck guest is at least this far from every player (players are never struck)
	rainMax           = 300   // drops (or flakes) on screen at full storm
	splashMax         = 48
	dawnChance        = 0.35     // a party of dawnLevels (without storm) gets a dawn
	dawnFullS         = 300.0    // seconds of party for a full dawn (timeMidS): or earlier, by the guests already out
	dawnRate          = 1.0 / 25 // the most the dawn advances per second (no jump when many die at once)
	dawnAEnd          = 0.22     // the night layer keeps this share of its alpha at full dawn
	dawnSkyA          = 0.16     // the strongest pink-grey light on the band of sky at full dawn
)

// stormLevels: the open-air parties (by Def.Icon) where a storm makes sense: no roof over most of the floor
// (indoor floor share 0-3 %), not daytime ITALIA, not the space station, not the island (it has its own weather).
// The ski resort gets snow instead of rain (no lightning).
var stormLevels = map[string]bool{"beach": true, "eiffel": true, "funfair": true, "stadium": true, "festival": true,
	"grave": true, "circuit": true, "snow": true, "highway": true}

// dawnLevels: night open-air parties where staying until sunrise is the classic ending of the night.
var dawnLevels = map[string]bool{"beach": true, "eiffel": true, "festival": true, "roof": true}

// stormShadeCol / stormFlashCol / dawnCol: the colours of the storm shade, of the lightning wash and of the night layer
// at full dawn (a neutral grey-mauve: from cold to warm but never yellow).
var (
	stormShadeCol = rgb(0x0e1220)
	stormFlashCol = rgb(0xdce6ff)
	dawnCol       = rgb(0x2a2832)
	dawnSkyCol    = rgb(0xc8a4b4)
)

// weatherPlan: what the party will have (decided at load).
type weatherPlan struct {
	storm    bool
	snow     bool // the storm is a snowfall (ski resort)
	at, dur  float64
	strike   bool
	strikeAt float64 // seconds after the storm starts
	dawn     bool
	seed     int64 // the dice of the drops and the flashes
}

type rainDrop struct {
	x, y, v, l, end float64
	ph              float64
}

type rainSplash struct {
	x, y, t float64
}

// weatherState: the weather of the party in course (g.wth).
type weatherState struct {
	plan     weatherPlan
	r        *rand.Rand
	started  bool
	done     bool
	t        float64 // seconds since the storm started
	k        float64 // storm intensity 0..1
	drops    [rainMax]rainDrop
	splash   [splashMax]rainSplash
	nextBolt float64 // storm time of the next flash
	flashT   float64 // seconds since the last flash (< stormFlashDur = flashing)
	boltX    float64 // the bolt of the flash (screen x of the top, of the end, y of the end)
	boltX2   float64
	boltY    float64
	thunder  float64 // seconds to the thunder of the last flash (<0 = none pending)
	scared   bool    // the guests already ran for cover
	struck   bool
	dawnK    float64
	dawnPin  float64 // > 0: debug, the dawn is fixed at this progress
}

var (
	weatherRng  *rand.Rand
	weatherSeed int64 // tests: fixed seed of weatherDice (0 = the clock)
	dbgStorm    int   // -storm N: 1 storm at ~3 s, 2 also a bolt on a guest
	dbgDawn     = -1.0
)

// weatherDice: the own dice of the weather (never g.rng).
func weatherDice() *rand.Rand {
	if weatherRng == nil {
		sd := weatherSeed
		if sd == 0 {
			sd = time.Now().UnixNano() ^ 0x7e47e4
		}
		weatherRng = rand.New(rand.NewSource(sd))
	}
	return weatherRng
}

// weatherPlanFor decides the weather of a party: icon of the level, whether it is a campaign party, and whether the
// party is daytime / stealth / today's challenge. Always draws the same number of dice, so each party moves the
// sequence the same way (deterministic by the seed of the party).
func weatherPlanFor(r *rand.Rand, icon string, campaign, day, stealth, daily bool) weatherPlan {
	rs, ra, rd, rk, rka, rdw, sd := r.Float64(), r.Float64(), r.Float64(), r.Float64(), r.Float64(), r.Float64(), r.Int63()
	var p weatherPlan
	p.seed = sd
	if !campaign || stealth || daily {
		return p
	}
	if stormLevels[icon] && rs < stormChance {
		p.storm, p.snow = true, icon == "snow"
		p.at = stormAtMin + ra*(stormAtMax-stormAtMin)
		p.dur = stormDurMin + rd*(stormDurMax-stormDurMin)
		if !p.snow && rk < stormStrikeChance {
			p.strike = true
			p.strikeAt = stormFadeIn + 2 + rka*math.Max(1, p.dur-stormFadeIn-stormFadeOut-4)
		}
		return p
	}
	if dawnLevels[icon] && !day && rdw < dawnChance {
		p.dawn = true
	}
	return p
}

// weatherLevelReset is called when a level loads (after partyReset).
func (g *Game) weatherLevelReset() {
	g.wth = weatherState{thunder: -1}
	if g.lvl == nil {
		return
	}
	def := g.lvl.Def
	campaign := g.levelIdx >= 0 && g.levelIdx < campaignN
	p := weatherPlanFor(weatherDice(), def.Icon, campaign, def.Day, g.stealth, g.dc != nil)
	if dbgStorm > 0 && !g.stealth {
		p = weatherPlan{storm: true, snow: def.Icon == "snow", at: 3, dur: 60, seed: 7}
		if dbgStorm >= 2 && !p.snow {
			p.strike, p.strikeAt = true, 9
		}
	}
	if dbgDawn >= 0 && !g.stealth && !def.Day {
		p.storm, p.strike, p.dawn = false, false, true
		g.wth.dawnPin = dbgDawn / 100
	}
	g.wth.plan = p
	g.wth.r = rand.New(rand.NewSource(p.seed))
	if p.storm && !p.snow {
		weatherSounds(g.snd) // the thunder is synthesised while the level loads, not in the middle of the party
	}
}

// stormK: the storm intensity at storm time t of a storm of dur seconds (0..1, smooth at both ends).
func stormK(t, dur, fadeIn float64) float64 {
	if t <= 0 || t >= dur {
		return 0
	}
	in := smooth01(t / fadeIn)
	out := smooth01((dur - t) / stormFadeOut)
	return math.Min(in, out)
}

func (g *Game) stormFadeIn() float64 {
	if dbgStorm > 0 {
		return 4
	}
	return stormFadeIn
}

func (g *Game) updateWeather(dt float64) {
	w := &g.wth
	if g.lvl == nil || g.mode != ModePlay {
		return
	}
	if w.plan.dawn {
		g.updateDawn(dt)
	}
	if !w.plan.storm || w.done {
		return
	}
	if !w.started {
		if g.time < w.plan.at {
			return
		}
		g.startStorm()
	}
	w.t += dt
	w.k = stormK(w.t, w.plan.dur, g.stormFadeIn())
	if w.t >= w.plan.dur {
		w.done, w.k = true, 0
		if w.plan.snow {
			g.say(snowEnd[w.r.Intn(len(snowEnd))])
		} else {
			g.say(stormEnd[w.r.Intn(len(stormEnd))])
		}
		return
	}
	w.stepDrops(dt)
	if !w.scared && w.k > 0.5 {
		w.scared = true
		g.stormScare()
	}
	if w.plan.snow {
		return
	}
	// lightning: a flash now and then, the thunder a moment later
	w.flashT += dt
	if w.thunder >= 0 {
		w.thunder -= dt
		if w.thunder < 0 {
			g.snd.Play("thunder")
		}
	}
	if w.plan.strike && !w.struck && w.t >= w.plan.strikeAt {
		w.struck = true
		g.stormStrike()
	} else if w.t >= w.nextBolt && w.k > 0.6 && !g.blackoutOn() {
		g.stormFlash(-1, -1, 0.4+w.r.Float64()*1.1)
	}
}

func (g *Game) startStorm() {
	w := &g.wth
	w.started, w.t = true, 0
	w.nextBolt = g.stormFadeIn() + 1 + w.r.Float64()*4
	if dbgStorm > 0 {
		w.nextBolt = 6 // -storm: the first flash at ~9 s of the party, for the captures
	}
	w.flashT = 99
	for i := range w.drops {
		w.drops[i] = w.newDrop(true)
	}
	for i := range w.splash {
		w.splash[i].t = 99
	}
	if w.plan.snow {
		g.say(snowStart[w.r.Intn(len(snowStart))])
		return
	}
	weatherSounds(g.snd)
	g.say(stormStart[w.r.Intn(len(stormStart))])
}

// newDrop: a drop (or a flake) somewhere above the screen (anywhere on it, at the start).
func (w *weatherState) newDrop(anywhere bool) rainDrop {
	r := w.r
	d := rainDrop{x: float64(r.Intn(screenW + 120)), y: -float64(4 + r.Intn(60)), ph: r.Float64() * 6.28}
	if anywhere {
		d.y = float64(r.Intn(screenH))
	}
	if w.plan.snow {
		d.v, d.l = 38+r.Float64()*46, float64(1+r.Intn(2))
	} else {
		d.v, d.l = 420+r.Float64()*220, float64(6+r.Intn(6))
	}
	d.end = d.y + 60 + r.Float64()*float64(screenH)
	return d
}

// stepDrops moves the drops (no allocation): a drop that reaches its end leaves a little splash and starts again on top.
func (w *weatherState) stepDrops(dt float64) {
	for i := range w.drops {
		d := &w.drops[i]
		d.y += d.v * dt
		if w.plan.snow {
			d.x += (math.Sin(d.ph+d.y*0.03)*14 - 10) * dt
		} else {
			d.x -= d.v * 0.18 * dt // a little wind: the drops lean
		}
		if d.y >= d.end || d.y > screenH+12 {
			if !w.plan.snow && d.y < screenH {
				for j := range w.splash {
					if s := &w.splash[j]; s.t > 0.14 {
						s.x, s.y, s.t = math.Round(d.x), math.Round(d.end), 0
						break
					}
				}
			}
			*d = w.newDrop(false)
		}
	}
	for j := range w.splash {
		w.splash[j].t += dt
	}
}

var stormStart = []string{
	"Empieza a llover... ¡y el DJ sube el volumen!",
	"¡Tormenta! Los paraguas se ponen a bailar.",
}
var stormEnd = []string{
	"Escampa. Alguien seca los altavoces con un secador.",
	"Deja de llover. La pista huele a perro mojado.",
}
var snowStart = []string{
	"¡Nevada fuerte! Los gorros de lana suben de precio.",
	"Empieza a nevar a lo grande. El muñeco de nieve se apunta.",
}
var snowEnd = []string{
	"Deja de nevar. Alguien ya ha hecho un iglú.",
}
var stormCries = []string{"¡LLUEVE!", "¡MI PELO!", "¡A CUBIERTO!"}
var snowCries = []string{"¡QUÉ FRÍO!", "¡NIEVE!", "¡MIS PIES!"}
var stormStrikeNews = []string{
	"¡Un rayo cae en plena fiesta! Esta vez ha sido el cielo.",
	"¡Rayo! La naturaleza también se ha apuntado a la fiesta.",
}

// outdoorAt: the point stands on an open-air floor (the last floor that contains it decides, as in footprints.go).
func outdoorAt(def *LevelDef, p Vec) bool {
	out := true
	for _, f := range def.Floors {
		if f.R.Has(p.X, p.Y) {
			out = !floorIndoor(f.Kind)
		}
	}
	return out
}

// stormScare: when the rain gets heavy some guests in the open run for a moment (the running state of the blackout);
// a few shout. Never the performers, never the ones busy with something else.
func (g *Game) stormScare() {
	w := &g.wth
	cries := stormCries
	if w.plan.snow {
		cries = snowCries
	}
	n := 0
	for _, gu := range g.guests {
		if gu.out() || gu.perf != "" || gu.py != nil || !outdoorAt(g.lvl.Def, gu.pos) {
			continue
		}
		switch gu.state {
		case gIdle, gGoto, gDance, gDrink:
		default:
			continue
		}
		if w.r.Float64() >= 0.22 {
			continue
		}
		if n < 3 {
			gu.icon, gu.iconT = cries[w.r.Intn(len(cries))], 2.2
			n++
		}
		gu.state, gu.timer, gu.path = gRun, 1+w.r.Float64()*1.5, nil
		gu.runUntil = g.time + 2.5 + w.r.Float64()*2
	}
}

// stormFlash: a lightning flash; the bolt goes from the top of the screen to (x, y) (x < 0: somewhere in the upper
// part of the screen); the thunder comes after `delay` seconds.
func (g *Game) stormFlash(x, y, delay float64) {
	w := &g.wth
	w.flashT = 0
	w.nextBolt = w.t + stormBoltMin + w.r.Float64()*(stormBoltMax-stormBoltMin)
	if x < 0 {
		x = 330 + w.r.Float64()*(screenW-660) // the middle of the screen: clear of the HUD panels on both sides
		y = stormBoltTop + 60 + w.r.Float64()*90
	}
	w.boltX2, w.boltY = math.Round(x), math.Round(y)
	// the top of the bolt is right of the HUD panel (whatever its height) and below the band of the stars and the news
	w.boltX = math.Round(clampF(x+(w.r.Float64()-0.5)*120, hudX+hudW+26, screenW-30))
	w.thunder = delay
}

// stormStrike: VERY rarely, the bolt falls on a guest in the open (an act of nature: not the player's kill, no
// stars, no crime; the guest is left as a burnt skeleton like Satan's bolts).
func (g *Game) stormStrike() {
	w := &g.wth
	var cand []*Guest
	for _, gu := range g.liveGuests() {
		if gu.py != nil || gu.vipCard || !outdoorAt(g.lvl.Def, gu.pos) || !g.pevPlayerFar(gu.pos, stormSafeR) {
			continue // never next to a player: the bolt (and its flash) only ever falls on a guest
		}
		cand = append(cand, gu)
	}
	if len(cand) == 0 {
		return
	}
	v := cand[w.r.Intn(len(cand))]
	g.stormFlash(g.wx(v.pos.X), g.wy(v.pos.Y)-12, 0.05)
	g.burst(Vec{v.pos.X, v.pos.Y - 14}, 14, 90, rgb(0xfff0a0), 0.4, 2, 0)
	g.snd.Play("zap")
	credit := g.credit
	g.credit = false
	g.killCause = kcNone
	g.killGuestElec(v, false)
	g.credit = credit
	g.say(stormStrikeNews[w.r.Intn(len(stormStrikeNews))])
}

// ---- dawn --------------------------------------------------------------------------------------------------------

// dawnTarget: how far the night should have gone at party time t with this share of guests out (0..1).
func dawnTarget(t, progress float64) float64 {
	return clampF(math.Max(t/dawnFullS, progress), 0, 1)
}

func (g *Game) updateDawn(dt float64) {
	w := &g.wth
	if w.dawnPin > 0 {
		w.dawnK = clampF(w.dawnPin, 0, 1)
		return
	}
	target := dawnTarget(g.time, g.hudProgress())
	if target > w.dawnK { // only forward: the sun does not go back down
		w.dawnK = math.Min(target, w.dawnK+dawnRate*dt)
	}
}

// dawnE: the eased dawn of the party (0 = night as always).
func (g *Game) dawnE() float64 {
	if g.lvl == nil || !g.wth.plan.dawn || g.lvl.Def.Day {
		return 0
	}
	return smooth01(g.wth.dawnK)
}

// nightWash: the colour and alpha (0..255) of drawLighting's night layer of a level at dawn e (0 = the usual night,
// exactly the old formula).
func nightWash(def *LevelDef, e float64) (color.RGBA, float64) {
	t := def.Tint
	aOut := clampF(float64(t.A)*1.2, 70, 112) // night, but still playable everywhere
	base := mixc(rgb(0x14142a), color.RGBA{t.R, t.G, t.B, 255}, 0.5)
	if e <= 0 {
		return base, aOut
	}
	e = math.Min(e, 1)
	return mixc(base, dawnCol, e), aOut * (1 - (1-dawnAEnd)*e)
}

// dawnLampK: the street lamps dim in the last part of the dawn (1 = lit as always).
func (g *Game) dawnLampK() float64 {
	e := g.dawnE()
	return 1 - 0.55*smooth01((e-0.6)/0.4)
}

func scaleRGBA(c color.RGBA, k float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * k), uint8(float64(c.G) * k), uint8(float64(c.B) * k), uint8(float64(c.A) * k)}
}

// ---- drawing -----------------------------------------------------------------------------------------------------

var (
	rainVs [(rainMax + splashMax) * 4]ebiten.Vertex
	rainIs [(rainMax + splashMax) * 6]uint16
	rainOp ebiten.DrawTrianglesOptions
)

func init() {
	for i := 0; i < len(rainIs)/6; i++ {
		b := uint16(i * 4)
		rainIs[i*6], rainIs[i*6+1], rainIs[i*6+2] = b, b+1, b+2
		rainIs[i*6+3], rainIs[i*6+4], rainIs[i*6+5] = b+1, b+3, b+2
	}
}

// weatherRect: a screen rectangle the drops keep out of (the players).
type weatherRect struct{ x0, y0, x1, y1 float64 }

// rainQuad writes the quad n of the batch: a parallelogram from (x, y) w px wide, h tall, leaning lean px.
func rainQuad(n int, x, y, w, h, lean float64, c color.RGBA, a float64) {
	r, g, b := float32(c.R)/255, float32(c.G)/255, float32(c.B)/255
	fa := float32(a)
	v := rainVs[n*4 : n*4+4]
	pts := [4][2]float64{{x, y}, {x + w, y}, {x + lean, y + h}, {x + lean + w, y + h}}
	for i := range v {
		v[i] = ebiten.Vertex{DstX: float32(pts[i][0]), DstY: float32(pts[i][1]), SrcX: 0.5, SrcY: 0.5, ColorR: r, ColorG: g, ColorB: b, ColorA: fa}
	}
}

// inKeep: a drop at (x, y), h px tall (and leaning a few px), touches one of the rects.
func inKeep(x, y, h float64, keep [2]weatherRect, nKeep int) bool {
	for j := 0; j < nKeep; j++ {
		r := keep[j]
		if x+4 >= r.x0 && x-4 <= r.x1 && y+h >= r.y0 && y <= r.y1 {
			return true
		}
	}
	return false
}

// rainBatch fills the vertex batch with the drops and splashes at intensity k, keeping out of the rects; it returns
// how many quads it wrote. No allocation (weather_test.go).
func (w *weatherState) rainBatch(k float64, keep [2]weatherRect, nKeep int) int {
	n := 0
	count := int(float64(rainMax) * k)
	var col color.RGBA
	var a float64
	if w.plan.snow {
		col, a = rgb(0xf4f8ff), 0.62*math.Min(1, k*1.4)
	} else {
		col, a = rgb(0xbcd0ec), 0.34*math.Min(1, k*1.4)
	}
	for i := 0; i < count; i++ {
		d := &w.drops[i]
		x, y := math.Round(d.x), math.Round(d.y)
		if inKeep(x, y, d.l+1, keep, nKeep) || y < -16 || y > screenH {
			continue
		}
		if w.plan.snow {
			rainQuad(n, x, y, d.l+1, d.l+1, 0, col, a)
		} else {
			rainQuad(n, x, y, 1, d.l, -math.Round(d.l*0.18), col, a)
		}
		n++
	}
	if !w.plan.snow {
		for j := range w.splash {
			s := &w.splash[j]
			if s.t > 0.14 || s.x < 0 || s.x > screenW || inKeep(s.x, s.y, 2, keep, nKeep) {
				continue
			}
			rainQuad(n, s.x-1, s.y, 3, 1, 0, col, a*0.9*(1-s.t/0.14))
			n++
		}
	}
	return n
}

// drawWeather paints the dawn sky, the storm (shade, drops, flash, bolt) over the lit world and below the HUD.
func (g *Game) drawWeather(dst *ebiten.Image) {
	w := &g.wth
	if g.lvl == nil || (g.mode != ModePlay && g.mode != ModeLose) {
		return
	}
	if e := g.dawnE(); e > 0 {
		g.drawDawnSky(dst, e)
	}
	if w.k <= 0 {
		return
	}
	fillRect(dst, 0, 0, screenW, screenH, premul(stormShadeCol, stormShadeA*w.k))
	var keep [2]weatherRect
	nk := 0
	for _, p := range g.players() {
		if p == nil || nk >= len(keep) {
			continue
		}
		x, y := g.wx(p.pos.X), g.wy(p.pos.Y)
		keep[nk] = weatherRect{x - 12, y - 40, x + 12, y + 4}
		nk++
	}
	if n := w.rainBatch(w.k, keep, nk); n > 0 {
		dst.DrawTriangles(rainVs[:n*4], rainIs[:n*6], px, &rainOp)
	}
	if f := stormFlashK(w.flashT); f > 0 {
		fillRect(dst, 0, 0, screenW, screenH, premul(stormFlashCol, stormFlashA*f))
		if w.flashT < 0.2 && w.boltY > stormBoltTop+16 {
			g.drawBolt(dst, w.boltX, stormBoltTop, w.boltX2, w.boltY)
		}
	}
}

// stormFlashK: the brightness of a lightning flash s seconds after it (two quick pulses, 0 when over).
func stormFlashK(s float64) float64 {
	switch {
	case s < 0 || s >= stormFlashDur:
		return 0
	case s < 0.07:
		return 1
	case s < 0.14:
		return 0.25
	case s < 0.21:
		return 0.7
	}
	return 0.7 * (1 - (s-0.21)/(stormFlashDur-0.21))
}

// drawDawnSky: at dawn the band of sky above the floors (if the level has one) gets a soft pink-grey light.
func (g *Game) drawDawnSky(dst *ebiten.Image, e float64) {
	top := 1e9
	for _, f := range g.lvl.Def.Floors {
		top = math.Min(top, f.R.Y)
	}
	if top < 8 || top > screenH/2 {
		return
	}
	h := g.wy(top)
	steps := 4.0
	for i := 0.0; i < steps; i++ {
		a := dawnSkyA * e * (1 - i/steps)
		fillRect(dst, 0, math.Round(h*i/steps), screenW, math.Round(h/steps)+1, premul(dawnSkyCol, a))
	}
}

// weatherSounds adds the thunder to the effects (built once, the same synthesis as the rest of audio.go).
func weatherSounds(s *Sounds) {
	if s == nil || s.sfx == nil {
		return
	}
	if _, ok := s.sfx["thunder"]; ok {
		return
	}
	s.sfx["thunder"] = mix(
		lowpass(noise(2.4, 0.9, 1.5), 48),
		shift(lowpass(noise(1.6, 0.6, 2.4), 30), 0.22),
		tone(44, 1.8, 0.45, 2.2, 0),
	)
}
