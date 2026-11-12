package main

import (
	"hash/fnv"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// RETO DE HOY (today's challenge). One button at the top of the party list launches ONE party per day:
//
//   - the party is chosen by the date (hash of YYYY-MM-DD, own dice, never g.rng) among the campaign parties the
//     player has unlocked (the island is not part of it), preferring parties where the VIP carpet fits;
//   - it ALWAYS has a VIP zone (vipRoll, no 1/5, no pity): a VIP the album does not have yet when there is one;
//   - 2 or 3 modifiers picked by the date from dailyMods (below), shown on the button and in the HUD;
//   - the old RETO multipliers stay: x3 per kill, x5 to the total when completed (levelscore.go), plus dailyBonus
//     and a secret trophy the first time (trophies3.go, stat "daily");
//   - if you die or get caught (lose(); in 2P when BOTH lose) the challenge is locked until tomorrow
//     (Progress.ChallengeLock = that date); if you complete it, Progress.ChallengeDone = that date. The normal
//     game is never locked. With debug flags progressFrozen stops saveProgress, so nothing is written.
//
// today() is the clock; tests replace it.
// ---------------------------------------------------------------------------

// today: the date YYYY-MM-DD of the local clock (tests replace it).
var today = func() string { return time.Now().Format("2006-01-02") }

const (
	dailyBonus     = 2500  // points added (after the x5) when the challenge is completed
	dailyTimeLimit = 390.0 // seconds of the "Contrarreloj" modifier (6:30)
	dailySightK    = 1.35  // "Ojo de lince": everybody sees this much farther
	dailyPoliceK   = 0.4   // "Poli veloz": the patrol car comes in this fraction of the usual wait
	dailyStaminaK  = 0.5   // "Sin aliento": this fraction of the stamina
)

type dailyMod int

const (
	dmPolice   dailyMod = iota // the police arrive much sooner
	dmBlackout                 // there is a power cut for sure
	dmKiller                   // there is a killer for sure
	dmNoMap                    // no minimap
	dmTime                     // maximum time
	dmSight                    // guests, guards and officers see farther
	dmStamina                  // half the stamina
	dmN
)

var dailyModName = [dmN]string{"Poli veloz", "Apagón seguro", "Asesino seguro", "Sin minimapa", "Contrarreloj 6:30", "Ojo de lince", "Sin aliento"}

func init() {
	for k, v := range map[string]string{
		"Poli veloz": "Speedy cops", "Apagón seguro": "Sure blackout", "Asesino seguro": "Sure killer",
		"Sin minimapa": "No minimap", "Contrarreloj 6:30": "Race the clock 6:30", "Ojo de lince": "Eagle eyes",
		"Sin aliento": "Out of breath", "RETO DE HOY": "TODAY'S CHALLENGE", "RETO DE HOY: ": "TODAY'S CHALLENGE: ",
		"YA JUGADO HOY · vuelve mañana": "ALREADY PLAYED TODAY · see you tomorrow", "Completado hoy · vuelve mañana": "Done today · come back tomorrow",
		"Se acabó el tiempo": "Time is up",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}

type dailyChallenge struct {
	date  string
	level int        // index in levelDefs (a campaign party)
	mods  []dailyMod // 2 or 3
	vip   string     // the VIP of the zone
}

func (d *dailyChallenge) has(m dailyMod) bool {
	if d == nil {
		return false
	}
	for _, x := range d.mods {
		if x == m {
			return true
		}
	}
	return false
}

func (d *dailyChallenge) modNames() []string {
	var out []string
	for _, m := range d.mods {
		out = append(out, dailyModName[m])
	}
	return out
}

// dailyDice: the own dice of a date (and a salt for other uses).
func dailyDice(date string, salt int64) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(date))
	return rand.New(rand.NewSource(int64(h.Sum64()) + salt))
}

// dailyVIPRoom: does the VIP carpet fit in campaign party i? Tests replace it.
var dailyVIPRoom = func(i int) bool {
	if i < 0 || i >= len(levelDefs) {
		return true
	}
	d, _ := vipPlace(levelDefs[i], "")
	return d != nil
}

// dailyPick: the challenge of a date. unlocked = campaign parties open (1..campaignN). Always deterministic.
func dailyPick(date string, unlocked int, album map[string]int) dailyChallenge {
	n := clampI(unlocked, 1, maxI(campaignN, 1))
	r := dailyDice(date, 0)
	order := r.Perm(n)
	d := dailyChallenge{date: date, level: order[0]}
	for _, i := range order { // the first (in the order of the date) where the VIP carpet fits
		if dailyVIPRoom(i) {
			d.level = i
			break
		}
	}
	title := ""
	if d.level < len(levelDefs) {
		title = levelDefs[d.level].Title
	}
	d.vip = vipPick(r, album, title)
	k := 2 + r.Intn(2)
	for _, m := range r.Perm(int(dmN))[:k] {
		d.mods = append(d.mods, dailyMod(m))
	}
	return d
}

const (
	dailyOpen = iota // can be played now
	dailyLost        // died or caught today
	dailyDone        // completed today
)

// dailyState: what the button offers right now.
func (g *Game) dailyState() int {
	t := today()
	switch {
	case g.dbg.dailyFree: // DEBUG: always open (debugmode.go); nothing is saved
		return dailyOpen
	case g.progress.ChallengeDone == t:
		return dailyDone
	case g.progress.ChallengeLock == t:
		return dailyLost
	}
	return dailyOpen
}

// dailyToday: today's challenge (cached: the VIP check builds levels).
func (g *Game) dailyToday() dailyChallenge {
	key := today() + "|" + itoa(g.progress.Unlocked)
	for _, d := range vipRegistry() {
		if vipGot(g.progress.Album, d.id) {
			key += d.id
		}
	}
	if g.dcKey != key {
		g.dcKey, g.dcToday = key, dailyPick(today(), g.progress.Unlocked, g.progress.Album)
	}
	return g.dcToday
}

// startDaily launches today's challenge. force (debug flag -challenge) ignores the lock; lvl >= 0 forces the party.
func (g *Game) startDaily(force bool, lvl int) bool {
	if !force && g.dailyState() != dailyOpen {
		return false
	}
	d := g.dailyToday()
	if lvl >= 0 && lvl < len(levelDefs) {
		d.level = lvl
	}
	g.dcArm = &d
	if !force && !progressFrozen {
		g.dailyFail() // entering spends the day's only attempt: leaving (pause, Esc, closing) does not give it back
	}
	g.startLevel(d.level)
	return true
}

// dailyLoad runs at the start of loadLevel: the challenge armed by startDaily is the one in force; any other load
// (menu, retry, next party) leaves it. The RETO multipliers follow it, but a plain g.challenge (tests) is untouched.
func (g *Game) dailyLoad() {
	was := g.dc != nil
	g.dc, g.dcArm = g.dcArm, nil
	switch {
	case g.dc != nil:
		g.challenge = true
	case was:
		g.challenge = false
	}
}

// dailyApplyStats: the modifiers that change numbers before the player exists (after applyShop).
func (g *Game) dailyApplyStats() {
	if g.dc.has(dmStamina) {
		g.perkStamina = maxF(3, g.perkStamina*dailyStaminaK)
	}
}

// dailyApplyEvents: the modifiers that plan events (after blackoutLevelReset and killerLevelReset).
func (g *Game) dailyApplyEvents() {
	if g.dc == nil || g.stealth {
		return
	}
	r := dailyDice(g.dc.date, 77)
	if g.dc.has(dmBlackout) {
		g.blk = blackoutState{planned: true, at: 40 + r.Float64()*60, dur: blackoutMin + r.Float64()*(blackoutMax-blackoutMin)}
	}
	if g.dc.has(dmKiller) && killerRegulation {
		g.kp = killerPlan{planned: true, at: killerAtMin + r.Float64()*30}
	}
}

// dailyIntro: the news line at the start of the party.
func (g *Game) dailyIntro() {
	if g.dc != nil {
		g.say(tr("RETO DE HOY: ") + strings.Join(trAll(g.dc.modNames()), " · "))
	}
}

func trAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = tr(s)
	}
	return out
}

// dailyDelay: the wait of the patrol car (seconds) with the modifier "Poli veloz".
func (g *Game) dailyDelay(sec float64) float64 {
	if g.dc.has(dmPolice) {
		return sec * dailyPoliceK
	}
	return sec
}

// dailySight: the factor of every sight range.
func (g *Game) dailySight() float64 {
	if g.dc.has(dmSight) {
		return dailySightK
	}
	return 1
}

// dailyTimeLeft: seconds left with the time modifier (ok=false: no such modifier).
func (g *Game) dailyTimeLeft() (float64, bool) {
	if !g.dc.has(dmTime) {
		return 0, false
	}
	return maxF(0, dailyTimeLimit-g.time), true
}

// dailyTick: the time limit (called from updatePlay). True when it ended the party.
func (g *Game) dailyTick() bool {
	if left, ok := g.dailyTimeLeft(); ok && left <= 0 && g.mode == ModePlay {
		g.lose("Se acabó el tiempo")
		return g.mode != ModePlay
	}
	return false
}

// dailyFail: attempt spent (entered, died or caught): locked until tomorrow.
func (g *Game) dailyFail() {
	g.progress.ChallengeLock = today()
	saveProgress(g.progress)
}

// dailyWin: completed: done for today, secret trophy.
func (g *Game) dailyWin() {
	g.progress.ChallengeDone = today()
	g.stat("daily", 1)
	saveProgress(g.progress)
}

// dailyButtonLines: the texts of the button: the title, then the party on its own line and the modifiers
// on theirs (wrapped to the inner width, at most dailyModLines lines), and whether it can be launched.
func (g *Game) dailyButtonLines() (label string, lines []string, open bool) {
	label = tr("RETO DE HOY")
	switch g.dailyState() {
	case dailyLost:
		return label, []string{tr("YA JUGADO HOY · vuelve mañana")}, false
	case dailyDone:
		return label, []string{tr("Completado hoy · vuelve mañana")}, false
	}
	return label, g.dailyLinesFor(g.dailyToday()), true
}

// dailyLinesFor: the party on its own line, the modifiers under it (wrapped to the inner width of the button).
func (g *Game) dailyLinesFor(d dailyChallenge) (lines []string) {
	if d.level < len(levelDefs) {
		lines = append(lines, tr(levelDefs[d.level].Title))
	}
	mods := strings.Join(trAll(d.modNames()), " · ")
	return append(lines, g.wrapText(mods, 1, dailyBtnW-2*dailyBtnPad)...)
}

// drawDailyButton: the single RETO DE HOY button on top of the party list. Whole-pixel boxes and scale-1 text,
// with an inner margin: the title and the party on the first line, the modifiers on the second.
func (g *Game) drawDailyButton(dst *ebiten.Image, sel bool) {
	label, lines, open := g.dailyButtonLines()
	g.drawDailyBox(dst, sel, label, lines, open)
}

// drawDailyBox: the box and its texts (split from drawDailyButton so the tests can feed every party and modifier).
func (g *Game) drawDailyBox(dst *ebiten.Image, sel bool, label string, lines []string, open bool) {
	x, y := dailyBtnX, dailyBtnY
	bg, edge, tc, sc := rgba(0x2a1608, 235), rgb(0xb06a20), rgb(0xff9030), rgb(0xe0c8a0)
	if sel {
		bg, edge, tc = rgba(0x4a2610, 245), rgb(0xffb040), rgb(0xffd060)
	}
	if !open {
		bg, edge, tc, sc = rgba(0x14141c, 235), rgb(0x3a4652), rgb(0x6f7b8a), rgb(0x6f7b8a)
		if sel {
			edge = rgb(0x6f7b8a)
		}
	}
	fillRect(dst, x, y, dailyBtnW, dailyBtnH, bg)
	fillRect(dst, x, y, dailyBtnW, 1, edge)
	fillRect(dst, x, y+dailyBtnH-1, dailyBtnW, 1, edge)
	fillRect(dst, x, y, 1, dailyBtnH, edge)
	fillRect(dst, x+dailyBtnW-1, y, 1, dailyBtnH, edge)
	pre := "  "
	if sel {
		pre = "> "
	}
	cx := x + dailyBtnW/2
	if open && len(lines) > 0 {
		// the title and the party share the first row (the list of 26 parties needs the height); the modifiers
		// go under them
		head, party := pre+label, lines[0]
		hw, pw := g.textW(head, 1), g.textW(party, 1)
		x0 := math.Round(cx - (hw+dailyHeadGap+pw)/2)
		g.drawText(dst, head, x0, y+dailyTextY0, tc, 1)
		drawChallengeFlag(dst, x0-12, y+dailyTextY0, tc)
		g.drawText(dst, party, x0+hw+dailyHeadGap, y+dailyTextY0, sc, 1)
		for i, l := range lines[1:] {
			g.drawTextC(dst, l, cx, y+dailyTextY0+dailyLineDY*float64(i+1), sc, 1)
		}
		return
	}
	lw := g.textW(pre+label, 1)
	g.drawTextC(dst, pre+label, cx, y+dailyTextY0, tc, 1)
	drawChallengeFlag(dst, math.Round(cx-lw/2)-12, y+dailyTextY0, tc)
	for i, l := range lines {
		g.drawTextC(dst, l, cx, y+dailyTextY0+dailyLineDY*float64(i+1), sc, 1)
	}
}
