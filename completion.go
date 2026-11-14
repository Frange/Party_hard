package main

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// COMPLETADO: the percentage of the game finished and what is still missing, top right of the main menu (under the
// trophy counter). It only READS counters that already exist (careerLevelsDone, careerGot, trophyCount, shopOwned);
// nothing new is saved.
// ---------------------------------------------------------------------------

// complPart: one thing to complete. one / many are the Spanish nouns for "n left" (tr() gives the English ones).
type complPart struct {
	one, many string
	got, all  int
}

// complParts: the parts of the game that count, read from the real data.
func (g *Game) complParts() []complPart {
	ch, chAll := g.careerGot(cardChar)
	vp, vpAll := g.careerGot(cardVIP)
	ks, ksAll := g.careerGot(cardKiller)
	cp, cpAll := g.careerGot(cardCop)
	shop, shopAll := 0, 0
	for _, list := range shopCatalog {
		for _, it := range list {
			if !shopBuyable(it) { // the same items as the trophy "toda la tienda" (trophies3.go)
				continue
			}
			shopAll++
			if ownedID(it.id) {
				shop++
			}
		}
	}
	return []complPart{
		{"1 fiesta", "%d fiestas", g.careerLevelsDone(), campaignShown()},
		{"1 personaje", "%d personajes", ch, chAll},
		{"1 VIP", "%d VIPs", vp, vpAll},
		{"1 asesino", "%d asesinos", ks, ksAll},
		{"1 policía", "%d policías", cp, cpAll},
		{"1 trofeo", "%d trofeos", g.trophyCount(), len(trophyList)},
		{"1 de tienda", "%d de tienda", shop, shopAll},
	}
}

// complPercent: the mean of the fractions of every part (each one weighs the same), rounded DOWN so that 100 only
// shows when every part is complete.
func complPercent(parts []complPart) int {
	sum, n := 0, 0
	done := true
	for _, p := range parts {
		if p.all <= 0 {
			continue
		}
		n++
		got := clampI(p.got, 0, p.all)
		sum += got * 10000 / p.all
		if got < p.all {
			done = false
		}
	}
	if n == 0 {
		return 0
	}
	if done {
		return 100
	}
	return minI(99, sum/n/100)
}

// complMissing: the things left, the biggest gap first, at most max of them ("14 personajes").
func complMissing(parts []complPart, max int) []string {
	var miss []complPart
	for _, p := range parts {
		if p.all > p.got {
			miss = append(miss, complPart{p.one, p.many, p.got, p.all})
		}
	}
	sort.SliceStable(miss, func(i, j int) bool { return miss[i].all-miss[i].got > miss[j].all-miss[j].got })
	var out []string
	for _, p := range miss {
		if len(out) == max {
			break
		}
		if n := p.all - p.got; n == 1 {
			out = append(out, tr(p.one))
		} else {
			out = append(out, tr(fmt.Sprintf(p.many, n)))
		}
	}
	return out
}

const (
	complRight = 944.0 // right edge of the dark box: 16 px from the screen edge
	complPad   = 8.0   // air between the text and the box (left and right)
	complY     = 55.0  // first text line; the box starts 3 px higher, 12 px under the trophy counter (it ends at y=40)
	complPitch = 14.0
	complMaxW  = 220.0 // widest text line: the box then starts at x=708, 12 px right of the bedroom window frame (ends at x=696)
)

// complLines: the lines of the block, already in the language in force (the first one is the title).
func (g *Game) complLines(parts []complPart) []string {
	pc := complPercent(parts)
	if pc >= 100 {
		return []string{"100 %"}
	}
	lines := []string{tr(fmt.Sprintf("COMPLETADO %d%%", pc))}
	head := tr("Faltan:")
	if g.progress.MenuTier >= tierGold { // the 100 % was reached once: it stays, what came later is "new" (medals.go)
		lines[0], head = "100 %", tr("Nuevo:")
	}
	cur := head
	for _, m := range complMissing(parts, 4) {
		next := cur + " " + m
		if cur != head {
			next = cur + " · " + m
		}
		if cur != head && g.textW(next, 1) > complMaxW && len(lines) < 3 {
			lines = append(lines, cur)
			cur = m
			continue
		}
		cur = next
	}
	return append(lines, cur)
}

var complCache struct {
	frame int
	parts []complPart
}

func (g *Game) drawCompletion(dst *ebiten.Image) {
	complCache.frame++
	if complCache.parts == nil || complCache.frame%30 == 0 {
		complCache.parts = g.complParts()
	}
	lines := g.complLines(complCache.parts)
	w := 0.0
	for _, l := range lines {
		w = maxF(w, g.textW(l, 1))
	}
	h := float64(len(lines)) * complPitch
	fillRect(dst, complRight-w-2*complPad, complY-3, w+2*complPad, h+4, rgba(0x05070e, 170)) // complPad of air on each side of the text
	for i, l := range lines {
		col := rgb(0xb8b0d0)
		if i == 0 {
			col = rgb(0xf0e060)
		}
		g.drawText(dst, l, complRight-complPad-g.textW(l, 1), complY+float64(i)*complPitch, col, 1)
	}
}
