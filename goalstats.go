package main

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// OBJETIVOS / CARRERA: the totals of the whole career of the player, as the sixth page of OBJETIVOS (next to
// TROFEOS, PERSONAJES, VIPS, ASESINOS and POLICÍAS; no new menu). It only READS the counters that Progress already
// keeps (Stats: played, wins, kills, traps, poison, deaths, earned, best, chain, stars, animals, won_N; Escapes; the
// album), plus ONE new counter, Stats["secs"] (seconds played), which is added to the same map: old saves simply
// have none (0) and nothing is lost or migrated.
//
// Why a tab and not a line: the tab row has room (it ends at x ~ 700, well before the counter at x 770), the
// trophy page is already full (list + detail + hints) and a page of 17 numbers needs its own air.
// ---------------------------------------------------------------------------

const careerSecsKey = "secs"

var careerFrac float64 // seconds not yet added to the counter (session only)

// careerTick: called every frame of a party that runs; adds whole seconds to Stats["secs"].
func (g *Game) careerTick(dt float64) {
	if g.mode != ModePlay || g.lvl == nil {
		return
	}
	careerFrac += dt
	if careerFrac < 1 {
		return
	}
	n := int(careerFrac)
	careerFrac -= float64(n)
	if g.progress.Stats == nil {
		g.progress.Stats = map[string]int{}
	}
	g.progress.Stats[careerSecsKey] += n // (not g.stat: it would wake the trophy check every second)
}

// careerKills: THE one count of the career kills (2026-11-11, owner), read by CARRERA, the kills trophies and the
// police unlocks (FBI 500 / army 1000 / SWAT 1500 / rangers 2000), the shop and the trophy cards. Stats["kills"] holds
// the player's own kills INCLUDING the poisoned ones (poison.go adds "kills" and "poison" for the same death);
// Stats["traps"] and Stats["poison"] are details of HOW, not extra bodies. CARRERA used to add kills + traps + poison
// (523 in a real save whose police progress said 447): now every screen shows the same number, the one the trophies
// and the police unlocks have always used, so no milestone moves and nothing already won can close again. Only the
// reading changes: the save keeps the same keys (no migration).
func careerKills(p *Progress) int {
	if p == nil {
		return 0
	}
	return p.Stats["kills"]
}

// progressStat: the value of a trophy counter; "kills" is the career count (careerKills), the rest is the raw Stats.
func progressStat(p *Progress, key string) int {
	if p == nil {
		return 0
	}
	if key == "kills" {
		return careerKills(p)
	}
	return p.Stats[key]
}

func careerTime(secs int) string {
	if secs >= 3600 {
		return fmt.Sprintf("%d h %02d min", secs/3600, secs/60%60)
	}
	return fmt.Sprintf("%d min %02d s", secs/60, secs%60)
}

// careerLevelsDone: campaign parties finished at least once.
func (g *Game) careerLevelsDone() int {
	n := 0
	for i := 0; i < campaignN; i++ {
		if campaignRetired(i) { // a retired party (campaign_retired.go) keeps its won_N but no longer counts
			continue
		}
		if g.progress.Stats[fmt.Sprintf("won_%d", i)] > 0 {
			n++
		}
	}
	return n
}

// careerGot: cards won / total of one collection.
func (g *Game) careerGot(kind int) (int, int) {
	got, all := 0, 0
	for _, c := range albumCards() {
		if c.kind() != kind {
			continue
		}
		all++
		if g.albumUnlocked(c) {
			got++
		}
	}
	return got, all
}

func careerOf(a, b int) string { return fmt.Sprintf("%d / %d", a, b) }

// careerRows: the two columns of the page (label in Spanish, tr() gives the English one; value already written).
func (g *Game) careerRows() (left, right [][2]string) {
	st := g.progress.Stats
	kills := careerKills(&g.progress)
	left = [][2]string{
		{"Partidas jugadas", fmt.Sprint(st["played"])},
		{"Niveles completados", careerOf(g.careerLevelsDone(), campaignShown())},
		{"Fiestas ganadas", fmt.Sprint(st["wins"])},
		{"Bajas totales", fmt.Sprint(kills)},
		{careerCuffsLabel, fmt.Sprint(g.progress.Escapes)},
		{"Tiempo jugado", careerTime(st[careerSecsKey])},
		{"Puntos ganados", fmt.Sprint(st["earned"])},
		{"Mejor fiesta", fmt.Sprint(st["best"])},
		{"Mejor cadena", fmt.Sprint(st["chain"])},
	}
	ch, chAll := g.careerGot(cardChar)
	vp, vpAll := g.careerGot(cardVIP)
	ks, ksAll := g.careerGot(cardKiller)
	cp, cpAll := g.careerGot(cardCop)
	right = [][2]string{
		{"Trofeos", careerOf(g.trophyCount(), len(trophyList))},
		{"Personajes", careerOf(ch, chAll)},
		{"VIPs conocidos", careerOf(vp, vpAll)},
		{"Asesinos vencidos", careerOf(ks, ksAll)},
		{"Policías vencidos", careerOf(cp, cpAll)},
		{"Mejor racha de estrellas", fmt.Sprint(st["stars"])},
		{"Animales caídos", fmt.Sprint(st["animals"])},
		{"Veces que has caído", fmt.Sprint(st["deaths"])},
	}
	return
}

const (
	careerCuffsLabel = "Esposas evitadas"
	careerCuffsGap   = 12.0 // between the label and the symbol
	careerCuffsDY    = 4.0  // the 14 px symbol centred in the 32 px stripe (label at +6)
)

const (
	careerY0    = 128.0
	careerPitch = 34.0
	careerLX0   = 250.0 // left column: label here, value right-aligned at careerLX1
	careerLX1   = 600.0
	careerRX0   = 650.0
	careerRX1   = 910.0
)

func (g *Game) drawCareer(dst *ebiten.Image) {
	g.drawText(dst, careerOf(g.trophyCount(), len(trophyList)), 770, 66, rgb(0xf0e060), 1.6)
	left, right := g.careerRows()
	col := func(rows [][2]string, x0, x1 float64) {
		for i, r := range rows {
			y := careerY0 + float64(i)*careerPitch
			if i%2 == 0 {
				fillRect(dst, x0-6, y-5, x1-x0+12, careerPitch-2, rgba(0x120c24, 150))
			}
			g.drawText(dst, r[0], x0, y+6, rgb(0xb8b0d0), 1)
			if r[0] == careerCuffsLabel { // the handcuffs symbol after the label (cuffs_art.go)
				drawCuffsIcon(dst, cuffsSmall, x0+math.Round(g.textW(tr(r[0]), 1))+careerCuffsGap, y+careerCuffsDY, 1)
			}
			v := tr(r[1])
			// the value at the same whole scale and on the same line as its label (it used to be twice as big)
			g.drawText(dst, v, math.Round(x1-g.textW(v, 1)), y+6, rgb(0xf0e060), 1)
		}
	}
	col(left, careerLX0, careerLX1)
	col(right, careerRX0, careerRX1)
	g.drawText(dst, careerSinceLine(g.progress.Started), 244, 452, rgb(0x8a86a0), 1)
}
