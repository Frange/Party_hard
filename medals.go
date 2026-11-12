package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// MEDALS (idea 210). Every campaign party gives a BRONZE, SILVER or GOLD medal by the POINTS of the party when it is
// won (the final total of the newspaper: kills, chains, clean kills and the completion / unseen / speed bonuses).
// The owner asked for EASY medals ("tantos trofeos para niños"): bronze is practically winning the party, silver a
// normal party and gold a casual player who plays with gusto, no stealth, combos or rush needed.
//
// ONE function decides the numbers: medalThresholds(goal), from the kill target of the party (BAJAS x/N, cap.go). They
// are fractions of medalReasonableMax(goal), the typical total of an EXPERT (every victim by the player, chains of 8,
// rarely seen), fitted to the simulation of medals_calib_test.go that drives the real scoring code:
//
//	reasonable max R(goal) = goal*medalPtsPerKill + medalFlat
//	bronze = 16 % of R, silver = 30 % of R, gold = 45 % of R (each one rounded down to 50)
//
// With 65 (normal party): 1050 / 1950 / 2950; with 95 (funfair, festival, ski): 1500 / 2850 / 4250. If the target
// of a party changes (cap.go, bigCrowd, a new party, the guest settings) the numbers follow by themselves. Measured
// (300 parties per style, fixed seed): a normal winner ("media": no chains, seen 20-60 %) gets GOLD in ~88 % of the
// parties, a clumsy winner ("torpe") gold ~13 %, silver or better ~85 %, bronze always.
//
// Which parties: the campaign list (campaign.go) - the parties that exist at that moment. NOT the secret island /
// Jurassic park (just for fun), NOT the random mode, NOT the RETO DE HOY (its x3/x5 belongs to the challenge), NOT the
// stealth mode (it has its own count and ending, it never reaches completeLevel). TWO PLAYERS: the party total (both
// together) against the thresholds of the real target of that party (it is 20-30 people bigger, so the numbers are
// too); the party list shows the one-player numbers, the hint plate at the start the exact ones.
//
// Save: Progress.Medals, by the party's ICON (stable when a party is inserted), only the best one; old saves have
// none. The trophy "Bañado en oro" (medalTrophy) wants gold in every party of the list at that moment.
// ---------------------------------------------------------------------------

const (
	medalNone = iota
	medalBronze
	medalSilver
	medalGold
)

const (
	medalTrophy = "t_goldall"  // gold in every campaign party (the island does not count)
	medalStat   = "goldmedals" // how many golds right now (shown as x/N in OBJETIVOS)
	// the expert's typical party (medals_calib_test.go): points per victim of the target, plus the flat bonuses
	medalPtsPerKill = 97
	medalFlat       = 300
	// the medals as per mil of the reasonable max
	medalBronzePM = 160
	medalSilverPM = 300
	medalGoldPM   = 450
)

// medalReasonableMax: the typical total of an expert in a party with that kill target.
func medalReasonableMax(goal int) int {
	if goal < 1 {
		goal = 1
	}
	return goal*medalPtsPerKill + medalFlat
}

// medalThresholds: the points of bronze, silver and gold for a party with that kill target (the ONLY place).
func medalThresholds(goal int) [3]int {
	r := medalReasonableMax(goal)
	at := func(pm int) int { return maxI(50, r*pm/1000/50*50) }
	b, s, g := at(medalBronzePM), at(medalSilverPM), at(medalGoldPM)
	s = maxI(s, b+50)
	g = maxI(g, s+50)
	return [3]int{b, s, g}
}

// medalFor: the medal those points win.
func medalFor(points, goal int) int {
	th := medalThresholds(goal)
	for m := medalGold; m >= medalBronze; m-- {
		if points >= th[m-1] {
			return m
		}
	}
	return medalNone
}

// medalLevels: the icons of the parties that give a medal (the campaign list as it is now; never the island).
func medalLevels() []string {
	var out []string
	for _, e := range campaignLevels {
		if e.icon != "" && e.icon != "jurassic" {
			out = append(out, e.icon)
		}
	}
	return out
}

func medalLevel(icon string) bool {
	for _, k := range medalLevels() {
		if k == icon {
			return true
		}
	}
	return false
}

// medalGoalFor: the kill target of a party for ONE player (the party list): the normal cap or the guest maximum of
// the settings + 5, plus the big-crowd extra of that party (cap.go goalMax / crowdBonusFor).
func (g *Game) medalGoalFor(def *LevelDef) int {
	if def == nil {
		return goalCap
	}
	return maxI(goalCap, g.guestMax()+5) + clampI(bigCrowd[def.Icon], 0, crowdMax)
}

// partyMedalOn: the party being played gives a medal.
func (g *Game) partyMedalOn() bool {
	return g.lvl != nil && g.lvl.Def != nil && g.levelIdx >= 0 && g.levelIdx < campaignN && g.dc == nil && !g.challenge &&
		!g.stealth && medalLevel(g.lvl.Def.Icon)
}

// partyMedalGoal: the target the medal of this party is measured against (the real one: 2P and settings included).
func (g *Game) partyMedalGoal() int { return g.goalMax() }

func (g *Game) medalOf(icon string) int { return clampI(g.progress.Medals[icon], medalNone, medalGold) }

// goldCount: golds among the parties of the list now.
func (g *Game) goldCount() int {
	n := 0
	for _, k := range medalLevels() {
		if g.medalOf(k) == medalGold {
			n++
		}
	}
	return n
}

// awardMedal: the party was won (completeLevel, after finishScore). Remembers the medal of the newspaper and keeps
// the best one in the save.
func (g *Game) awardMedal() {
	g.run.medal, g.run.medalNew, g.run.medalOn = medalNone, false, false
	if !g.partyMedalOn() {
		return
	}
	g.run.medalOn = true
	g.run.medalGoal = g.partyMedalGoal()
	m := medalFor(g.points, g.run.medalGoal)
	g.run.medal = m
	icon := g.lvl.Def.Icon
	if m > g.medalOf(icon) {
		if g.progress.Medals == nil {
			g.progress.Medals = map[string]int{}
		}
		g.progress.Medals[icon] = m
		g.run.medalNew = true
		g.medalSync()
	}
}

// medalLive: every frame of the party (trackCuffs). When the points of the party (both players together in 2P, as
// awardMedal counts them) reach bronze, silver or gold, a toast says so, like a trophy: the same queue (g.toasts),
// so it never overlaps a trophy toast, and the same sound. One toast per medal and party (run.medalSaid, reset with
// the party); if several medals are reached at once, one toast each, in order. Only in the parties that give a medal
// (partyMedalOn: not the RETO, the stealth mode, the random mode or the island) and only while playing: the bonuses
// of the end of the party show their medal in the newspaper. Nothing is saved here: awardMedal keeps the medal of a
// WON party in progress.json, as before.
func (g *Game) medalLive() {
	r := &g.run
	if r.done || g.mode != ModePlay || !g.partyMedalOn() {
		return
	}
	if medalToastDemo > 0 && g.time > 0.5 { // -medaltoast N (progress frozen): the points jump to that medal
		g.points = maxI(g.points, medalThresholds(g.partyMedalGoal())[clampI(medalToastDemo, 1, 3)-1])
		medalToastDemo = 0
	}
	m := medalFor(g.points, g.partyMedalGoal())
	if m <= r.medalSaid {
		return
	}
	for r.medalSaid < m {
		r.medalSaid++
		g.toasts = append(g.toasts, trophyToast{medal: r.medalSaid, medalGoal: g.partyMedalGoal()})
	}
	if g.snd != nil {
		g.snd.Play("win")
	}
}

// medalToastDemo: the debug flag -medaltoast N (1 bronze, 2 silver, 3 gold).
var medalToastDemo int

// medalToastLines: the three lines of the medal toast (Spanish keys, tr() runs in drawText).
func medalToastLines(m, goal int) (head, name, foot string) {
	th := medalThresholds(goal)
	head = "¡Medalla conseguida!"
	switch m {
	case medalGold:
		return head, "¡Medalla de oro!", "Gana la fiesta para quedártela"
	case medalSilver:
		return head, "¡Medalla de plata!", fmt.Sprintf("Siguiente: oro a %d puntos", th[2])
	}
	return head, "¡Medalla de bronce!", fmt.Sprintf("Siguiente: plata a %d puntos", th[1])
}

// medalSync: the x/N of the trophy follows the list of parties and the golds of the save.
func (g *Game) medalSync() {
	for i := range trophyList {
		if trophyList[i].id == medalTrophy {
			trophyList[i].need = len(medalLevels())
		}
	}
	if g.progress.Stats == nil {
		g.progress.Stats = map[string]int{}
	}
	n := g.goldCount()
	g.statMax("goldmedals", n)           // = medalStat (TestTrophyStatsAreWritten looks for the writer)
	if g.progress.Stats[medalStat] > n { // a party inserted later (no gold yet) lowers it
		g.progress.Stats[medalStat] = n
	}
	g.trophyDirty = true
}

// medalAllGold: gold in every party of the list (the trophy).
func (g *Game) medalAllGold() bool {
	n := len(medalLevels())
	return n > 0 && g.goldCount() >= n
}

// ---- the menu tier that never goes down (menutier.go) --------------------------------------------------------------
//
// The new trophy raises the total by one, so whoever had every trophy / COMPLETADO 100 % in v1.0.0 would lose the
// silver / gold menu until winning gold everywhere. Progress.MenuTier keeps the best tier ever reached; a save from
// before the medals (MedalRev 0) gets it computed once as v1.0.0 would (without the new trophy). Nothing achieved is
// taken away; the new trophy still shows as missing in OBJETIVOS and under the 100 % ("Nuevo: 1 trofeo").

// medalLegacyInit runs once the game is loaded (newGame).
func (g *Game) medalLegacyInit() {
	if g.progress.MedalRev == 0 {
		g.progress.MenuTier = maxI(g.progress.MenuTier, g.v100MenuTier())
		g.progress.MedalRev = 1
	}
	g.medalSync()
}

// v100MenuTier: the menu tier v1.0.0 gave this (already migrated) save, with the rules of v1.0.0: 25 parties (every
// campaign party but the BARCO PIRATA, new in v1.1.0 at pirateIdx) and 70 trophies (every one but medalTrophy).
// The album and the shop are the same as in v1.0.0. Whatever the new party and the new trophy add counts only from
// now on (under the kept 100 % it shows as "Nuevo:", completion.go).
func (g *Game) v100MenuTier() int {
	won, all := 0, 0
	for i := 0; i < campaignN; i++ {
		if i == pirateIdx {
			continue
		}
		all++
		if g.progress.Stats[fmt.Sprintf("won_%d", i)] > 0 {
			won++
		}
	}
	finished := all > 0 && won >= all
	allOld := g.progress.Trophies[legendTrophy] && g.othersDoneExcept(medalTrophy)
	parts := g.complParts()
	for i := range parts {
		switch parts[i].one {
		case "1 fiesta":
			parts[i].got, parts[i].all = won, all
		case "1 trofeo":
			parts[i].all = maxI(0, parts[i].all-1)
			if g.progress.Trophies[medalTrophy] {
				parts[i].got--
			}
		}
	}
	return menuTierOf(finished, allOld, complPercent(parts) >= 100)
}

// othersDoneExcept: othersDone (powers.go) without one trophy.
func (g *Game) othersDoneExcept(skip string) bool {
	for _, t := range trophyList {
		if t.id != legendTrophy && t.id != skip && !secretTrophy[t.id] && !g.progress.Trophies[t.id] {
			return false
		}
	}
	return true
}

// medalDemo: the debug flag -medaldemo (progress frozen by the caller): every party open, medals in turn.
func (g *Game) medalDemo() {
	g.progress = cloneProgress(g.progress)
	g.progress.Unlocked = maxI(g.progress.Unlocked, campaignN)
	g.progress.Medals = map[string]int{}
	for i, k := range medalLevels() {
		g.progress.Medals[k] = 1 + i%3
	}
	g.medalSync()
}

// ---- texts and drawing -----------------------------------------------------------------------------------------

func medalName(m int) string {
	switch m {
	case medalGold:
		return "ORO"
	case medalSilver:
		return "PLATA"
	case medalBronze:
		return "BRONCE"
	}
	return ""
}

// medalLine: the short line with the points needed (party list and hint plate).
func medalLine(goal int) string {
	th := medalThresholds(goal)
	return fmt.Sprintf("Oro %d · Plata %d · Bronce %d", th[2], th[1], th[0])
}

// medalHintLine: the third line of the hint plate at the start of a party that gives a medal ("" otherwise).
func (g *Game) medalHintLine() string {
	if !g.partyMedalOn() {
		return ""
	}
	return "Medallas: " + medalLine(g.partyMedalGoal())
}

// medal colours: rim, face, light
func medalCols(m int) (rim, face, hi color.RGBA) {
	switch m {
	case medalGold:
		return rgb(0x8a5a10), rgb(0xf0c030), rgb(0xfff0a0)
	case medalSilver:
		return rgb(0x5a6070), rgb(0xc8d0dc), rgb(0xffffff)
	}
	return rgb(0x5a2e10), rgb(0xc88048), rgb(0xf4c090)
}

const medalIconW, medalIconH = 10.0, 14.0

// levelLineCX: the centre of the points line of the party list, moved in so it keeps 16 px from the panel (230-930).
func levelLineCX(cx, w float64) float64 {
	return float64(int(clampF(cx, 230+16+w/2+1, 930-16-w/2-1)))
}

// levelMedalX: the medal of a row of the party list sits 8 px right of its centred name (label width w).
func levelMedalX(cx, w float64) float64 { return float64(int(cx + w/2 + 8)) }

// drawMedal paints a small pixel medal (ribbon + disc) with its top left corner at x, y: medalIconW x medalIconH.
func drawMedal(dst *ebiten.Image, x, y float64, m int) { drawMedalS(dst, x, y, m, 1) }

// drawMedalS: drawMedal with whole pixels of s x s (the toast uses 3: 30 x 42).
func drawMedalS(dst *ebiten.Image, x, y float64, m int, s float64) {
	if m == medalNone {
		return
	}
	rim, face, hi := medalCols(m)
	x, y = float64(int(x)), float64(int(y))
	r := func(dx, dy, w, h float64, c color.RGBA) { fillRect(dst, x+dx*s, y+dy*s, w*s, h*s, c) }
	r(1, 0, 3, 5, rgb(0xd02a3a)) // ribbon
	r(6, 0, 3, 5, rgb(0x2a5ad0))
	r(2, 4, 6, 10, rim) // the disc: an octagon of 10 x 10
	r(1, 5, 8, 8, rim)
	r(0, 6, 10, 6, rim)
	r(2, 5, 6, 8, face)
	r(1, 6, 8, 6, face)
	r(3, 6, 2, 2, hi)
	r(4, 8, 2, 3, rim) // a little "1"
}

// paperMedal: the medal in the head of the points table of the newspaper (only parties that give one).
func (g *Game) paperMedal(d *paperData) {
	r := &g.run
	if !r.medalOn || !r.done {
		return
	}
	if r.medal == medalNone {
		d.scoreTag = tr(fmt.Sprintf("SIN MEDALLA · bronce %d", medalThresholds(r.medalGoal)[0]))
		return
	}
	d.medal = r.medal
	if r.medalNew {
		d.scoreTag = tr(medalName(r.medal)) + " · " + tr("¡NUEVA MEDALLA!")
		return
	}
	d.scoreTag = tr("MEDALLA DE " + medalName(r.medal))
}

// medal on the newspaper canvas: 9 x 12, ribbon + disc, with its top left corner at x, y.
func (c *pcv) medal(x, y, m int) {
	rim, face, hi := medalCols(m)
	if c.lay != nil {
		c.lay.medals = append(c.lay.medals, prect{"medal", x, y, 9, 12})
	}
	c.fill(x+1, y, 2, 4, color.RGBA{0xb0, 0x20, 0x20, 255})
	c.fill(x+6, y, 2, 4, color.RGBA{0x20, 0x40, 0xa0, 255})
	c.fill(x+2, y+3, 5, 9, rim)
	c.fill(x, y+5, 9, 5, rim)
	c.fill(x+1, y+4, 7, 7, rim)
	c.fill(x+2, y+5, 5, 5, face)
	c.fill(x+1, y+6, 7, 3, face)
	c.fill(x+3, y+5, 1, 2, hi)
}

func init() {
	medal := Trophy{medalTrophy, "Bañado en oro", "Consigue la medalla de ORO en todas las fiestas de la campaña (la Isla Nublada no cuenta: allí se va a jugar).",
		medalStat, len(medalLevels()), evr("medal:goldall", "Vitrina de oro", "Tus medallas de oro brillan en la lista de fiestas. La gloria no se come, pero luce mucho.")}
	at := len(trophyList)
	for i, t := range trophyList {
		if t.id == legendTrophy {
			at = i
		}
	}
	rest := append([]Trophy(nil), trophyList[at:]...)
	trophyList = append(append(trophyList[:at:at], medal), rest...)
	for k, v := range map[string]string{
		"Bañado en oro": "Dipped in gold",
		"Consigue la medalla de ORO en todas las fiestas de la campaña (la Isla Nublada no cuenta: allí se va a jugar).": "Get the GOLD medal in every party of the campaign (Isla Nublada does not count: that one is just for fun).",
		"Vitrina de oro": "Golden showcase",
		"Tus medallas de oro brillan en la lista de fiestas. La gloria no se come, pero luce mucho.": "Your gold medals shine on the party list. You can't eat glory, but it looks great.",
		"Oro %d · Plata %d · Bronce %d":           "Gold %d · Silver %d · Bronze %d",
		"Medallas: Oro %d · Plata %d · Bronce %d": "Medals: Gold %d · Silver %d · Bronze %d",
		"ORO":                     "GOLD",
		"PLATA":                   "SILVER",
		"BRONCE":                  "BRONZE",
		"MEDALLA DE ORO":          "GOLD MEDAL",
		"MEDALLA DE PLATA":        "SILVER MEDAL",
		"MEDALLA DE BRONCE":       "BRONZE MEDAL",
		"¡NUEVA MEDALLA!":         "NEW MEDAL!",
		"SIN MEDALLA · bronce %d": "NO MEDAL · bronze %d",
		"Nuevo:":                  "New:",
		// the toast during the party (medalLive)
		"¡Medalla conseguida!":           "Medal earned!",
		"¡Medalla de bronce!":            "Bronze medal!",
		"¡Medalla de plata!":             "Silver medal!",
		"¡Medalla de oro!":               "Gold medal!",
		"Siguiente: plata a %d puntos":   "Next: silver at %d points",
		"Siguiente: oro a %d puntos":     "Next: gold at %d points",
		"Gana la fiesta para quedártela": "Win the party to keep it",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
