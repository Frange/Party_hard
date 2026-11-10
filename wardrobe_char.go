package main

import (
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// VESTUARIO > PERSONAJE > "Personaje": the character picker. Confirming (or clicking) the row "Personaje" opens a
// grid over the rows: "Por defecto" (the player's own look) and every character he has won (OBJETIVOS: CONSEGUIDO
// after charKillsNeed kills, a trophy reward, or a character bought in the old shop -- old purchases that are no
// card any more, like the costumes or the killers sold back then, stay wearable). The ones still to win follow as
// "???" with the status line of their card. Choosing one saves it in the existing wardrobe slot "base" of that
// player (Progress.Equip / EquipP2: no new field) and the game draws him with that character and his gait; the
// powers, speed and everything else stay the player's. Nothing costs points.
// ---------------------------------------------------------------------------

// the picker state (wardrobe state, like wardP)
var (
	wardPick       bool // the grid is open
	wardPickSel    int
	wardPickScroll int // in rows
)

// the grid: 3 columns x 4 rows of cells inside the rows area of the wardrobe (x 244..600)
const (
	wpkX, wpkY       = 244.0, 138.0 // 12+ px under the page tabs (text ends at shopTabTextY+16)
	wpkCols, wpkRows = 3, 4
	wpkCellW         = 110.0
	wpkCellH         = 74.0
	wpkGap           = 12.0
	wpkGapY          = 12.0
	wpkBarX          = 612.0
)

// wardPickItem: one cell. id "" is the default look; locked cells keep their card for the status line.
type wardPickItem struct {
	id, name string
	locked   bool
	card     *albumCard
}

// wardPickList: default + the won characters (by name) + the old purchases that are no card + the locked ones.
func (g *Game) wardPickList() []wardPickItem {
	out := []wardPickItem{{id: "", name: "Por defecto"}}
	var got, locked []wardPickItem
	seen := map[string]bool{}
	cards := albumCards()
	for i := range cards {
		c := &cards[i]
		if c.kind() != cardChar || seen[c.slug] {
			continue
		}
		seen[c.slug] = true
		if ownedID("skin:"+c.slug) || charUnlocked(g.progress.Album, c.slug) {
			got = append(got, wardPickItem{id: c.slug, name: c.nameES, card: c})
		} else {
			locked = append(locked, wardPickItem{id: c.slug, name: "???", locked: true, card: c})
		}
	}
	for _, id := range g.progress.Owned { // bought before the redesign and no card now: still his
		if !strings.HasPrefix(id, "skin:") || seen[id[5:]] {
			continue
		}
		slug := id[5:]
		if slug == "" || presetByName(slug) == nil {
			continue
		}
		seen[slug] = true
		got = append(got, wardPickItem{id: slug, name: autoDisplayName(slug)})
	}
	sort.SliceStable(got, func(i, j int) bool { return got[i].name < got[j].name })
	out = append(out, got...)
	return append(out, locked...)
}

// wardBaseOptions: what the row "Personaje" cycles through (the picker without the locked ones).
func (g *Game) wardBaseOptions() []wardOption {
	var out []wardOption
	for _, it := range g.wardPickList() {
		if !it.locked {
			out = append(out, wardOption{it.id, it.name})
		}
	}
	return out
}

// wardBaseValid: the character a player wears can still be worn (won or owned, and it still exists).
func wardBaseValid(slug string) bool {
	return slug != "" && ownedID("skin:"+slug) && presetByName(slug) != nil
}

// wardPickIndex: the cell of what a player wears (0, the default, when he wears nothing valid).
func (g *Game) wardPickIndex(list []wardPickItem, player int) int {
	cur := g.equipOf(player, "base")
	if !wardBaseValid(cur) {
		return 0
	}
	for i, it := range list {
		if it.id == cur && !it.locked {
			return i
		}
	}
	return 0
}

func (g *Game) wardPickOpen() {
	wardPick = true
	wardPickSel = g.wardPickIndex(g.wardPickList(), wardP)
	wardPickScroll = 0
	wardPickKeepInView(len(g.wardPickList()))
	g.shopMsg = ""
}

func wardPickKeepInView(n int) {
	total := (n + wpkCols - 1) / wpkCols
	row := wardPickSel / wpkCols
	if row < wardPickScroll {
		wardPickScroll = row
	}
	if row >= wardPickScroll+wpkRows {
		wardPickScroll = row - wpkRows + 1
	}
	wardPickScroll = clampI(wardPickScroll, 0, maxI(0, total-wpkRows))
}

// wardPickCell: the rectangle of the i-th visible cell.
func wardPickCell(vis int) (x, y float64) {
	return wpkX + float64(vis%wpkCols)*(wpkCellW+wpkGap), wpkY + float64(vis/wpkCols)*(wpkCellH+wpkGapY)
}

func wardPickArea() [4]float64 {
	return [4]float64{wpkX, wpkY, wpkCols*wpkCellW + (wpkCols-1)*wpkGap, wpkRows*wpkCellH + (wpkRows-1)*wpkGapY}
}

// wardPickChoose puts the character of the cell on a player (a locked cell does nothing) and saves.
func (g *Game) wardPickChoose(player int, it wardPickItem) bool {
	if it.locked {
		return false
	}
	g.setEquipOf(player, "base", it.id)
	saveProgress(g.progress)
	g.applyShop()
	wardFlash = 0.45
	return true
}

// wardPickTry: on the row "Personaje" of the PERSONAJE page, confirming or clicking the row opens the picker.
func (g *Game) wardPickTry(slot string) bool {
	// the ATRÁS button pressed this frame (focused + A, or clicked) owns the press: the wardrobe goes back as usual
	if wardKind != wardKindPlayer || slot != "base" || wardPick || backFocus || backInject {
		return false
	}
	open := actConfirm.just()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if x >= 244 && x <= 600 && y >= 148+wardRow*25 && y <= 148+wardRow*25+23 {
			open = true
		}
	}
	if open {
		g.wardPickOpen()
		g.snd.Play("alert")
	}
	return open
}

// updateWardPick runs the picker while it is open; false lets the wardrobe handle the frame.
func (g *Game) updateWardPick() bool {
	if !wardPick {
		return false
	}
	if wardKind != wardKindPlayer || g.shopTab != shopWardrobe {
		wardPick = false
		return false
	}
	list := g.wardPickList()
	n := len(list)
	wardPickSel = clampI(wardPickSel, 0, n-1)
	if actBack.just() {
		wardPick = false
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if y >= 100 && y <= 120 { // the page tabs: close and let the wardrobe change page
			wardPick = false
			return false
		}
		for p := 0; p < g.wardPlayers(); p++ {
			if x >= 640+p*90 && x < 640+p*90+86 && y >= 126 && y <= 146 && p != wardP {
				wardP = p
				wardPickSel = g.wardPickIndex(list, wardP)
				wardFlash = 0.3
			}
		}
		for v := 0; v < wpkCols*wpkRows; v++ {
			i := wardPickScroll*wpkCols + v
			if i >= n {
				break
			}
			cx, cy := wardPickCell(v)
			if float64(x) >= cx && float64(x) < cx+wpkCellW && float64(y) >= cy && float64(y) < cy+wpkCellH {
				wardPickSel = i
				if g.wardPickChoose(wardP, list[i]) {
					g.snd.Play("alert")
				}
			}
		}
	}
	switch {
	case actRight.just():
		wardPickSel = minI(n-1, wardPickSel+1)
	case actLeft.just():
		wardPickSel = maxI(0, wardPickSel-1)
	case actDown.just():
		if wardPickSel+wpkCols < n {
			wardPickSel += wpkCols
		}
	case actUp.just():
		if wardPickSel-wpkCols >= 0 {
			wardPickSel -= wpkCols
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		wardPickSel = minI(n-1, wardPickSel+wpkCols*wpkRows)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		wardPickSel = maxI(0, wardPickSel-wpkCols*wpkRows)
	}
	if actConfirm.just() && g.wardPickChoose(wardP, list[wardPickSel]) {
		g.snd.Play("alert")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		wardPickSel = 0
		g.wardPickChoose(wardP, list[0])
	}
	if (actInteract.just() || inpututil.IsKeyJustPressed(ebiten.KeyTab)) && g.wardPlayers() > 1 {
		wardP = 1 - wardP
		wardPickSel = g.wardPickIndex(list, wardP)
		wardFlash = 0.3
	}
	wardPickKeepInView(n)
	// the wheel and the hover strips at the top and the bottom of the grid scroll it
	if _, wy := ebiten.Wheel(); wy != 0 {
		d := -1
		if wy < 0 {
			d = 1
		}
		wardPickSel, wardPickScroll = edgeApply(d, wardPickSel, wardPickScroll, wpkRows, wpkCols, n)
	}
	wardPickSel, wardPickScroll = edgeListScroll("wardpick", wardPickSel, wardPickScroll, wpkRows, wpkCols, n, wardPickArea(), 6)
	return true
}

// wardPickCS: the small animated character of a cell, built like the one the player wears (same seed).
func (g *Game) wardPickCS(it wardPickItem) *CharSet {
	if it.id == "" {
		return g.shopCS("wpk0:"+g.equipSigOf(wardP), func() *CharSpec { return g.playerSpecFor(wardP, "base", "") })
	}
	return g.shopCS("wpk:"+it.id, func() *CharSpec {
		s := presetByName(it.id)(rand.New(rand.NewSource(5)))
		return &s
	})
}

// drawWardPick: the grid (in place of the rows) and its scroll bar.
func (g *Game) drawWardPick(dst *ebiten.Image) {
	list := g.wardPickList()
	n := len(list)
	worn := [2]int{g.wardPickIndex(list, 0), -1}
	if g.wardPlayers() > 1 {
		worn[1] = g.wardPickIndex(list, 1)
	}
	for v := 0; v < wpkCols*wpkRows; v++ {
		i := wardPickScroll*wpkCols + v
		if i >= n {
			break
		}
		it := list[i]
		x, y := wardPickCell(v)
		bg, edge := rgba(0x120c24, 220), rgba(0x3a2a5a, 220)
		if i == wardPickSel {
			bg, edge = rgba(0x2a1a4a, 235), rgb(0xff6a8a)
		}
		shopFrame(dst, x, y, wpkCellW, wpkCellH, bg, edge)
		for p := 0; p < 2; p++ { // what each player wears: an inner frame in his colour (and his tag with two players)
			if worn[p] != i {
				continue
			}
			c := playerColor(p)
			in := float64(2 + p*2)
			shopFrame(dst, x+in, y+in, wpkCellW-2*in, wpkCellH-2*in, rgba(0, 0), c)
			if g.wardPlayers() > 1 {
				tag := []string{"J1", "J2"}[p]
				tx := x + 8 // J1 on the left, J2 on the right, both on the top line
				if p == 1 {
					tx = x + wpkCellW - 8 - math.Round(g.textW(tag, 1))
				}
				g.drawText(dst, tag, tx, y+6, c, 1)
			}
		}
		if it.locked {
			q := "?"
			g.drawText(dst, q, math.Round(x+wpkCellW/2-g.textW(q, 2)/2), y+14, rgb(0x5a5470), 2)
		} else if cs := g.wardPickCS(it); cs != nil {
			dir := []Dir{DirDown, DirRight, DirDown, DirLeft}[int(wardTime/1.6+float64(i)*0.37)%4]
			anim := AnimIdle
			if i == wardPickSel {
				anim = AnimDance0 + i%3
			}
			if im, flip := cs.img(anim, dir, wardTime*6); im != nil {
				w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
				op := &ebiten.DrawImageOptions{}
				if flip {
					op.GeoM.Scale(-1, 1)
					op.GeoM.Translate(w, 0)
				}
				op.GeoM.Translate(math.Round(x+wpkCellW/2-w/2), math.Round(y+wpkFeet-h))
				dst.DrawImage(im, op)
			}
		}
		col := rgb(0xf4f0ff)
		switch {
		case it.locked:
			col = rgb(0x8a86a0)
		case it.id == "":
			col = rgb(0xf0e060)
		}
		name := g.fitText(it.name, 1, wpkCellW-12)
		g.drawText(dst, name, math.Round(x+wpkCellW/2-g.textW(name, 1)/2), y+wpkNameDY, col, 1)
	}
	// the scroll bar
	total := (n + wpkCols - 1) / wpkCols
	if total > wpkRows {
		a := wardPickArea()
		fillRect(dst, wpkBarX, a[1], 4, a[3], rgba(0x2a2440, 200))
		h := math.Max(12, math.Round(a[3]*float64(wpkRows)/float64(total)))
		y := a[1] + math.Round((a[3]-h)*float64(wardPickScroll)/float64(total-wpkRows))
		fillRect(dst, wpkBarX, y, 4, h, rgb(0xff6a8a))
	}
}

// wardPickDesc: the text under the mannequin for the selected cell.
func (g *Game) wardPickDesc() string {
	list := g.wardPickList()
	it := list[clampI(wardPickSel, 0, len(list)-1)]
	switch {
	case it.locked:
		return "??? - " + g.charStatus(*it.card)
	case it.id == "":
		return tr("Por defecto") + " - " + tr("Tu aspecto de siempre, con la ropa que elijas.")
	case it.card != nil:
		return tr(it.name) + " - " + tr(charDesc(it.id))
	}
	return tr(it.name) + " - " + tr("Lo compraste en la tienda antigua: sigue siendo tuyo.")
}

// wardPickPreview: the mannequin shows the selected cell on the player being dressed (what he wears when locked).
func (g *Game) wardPickPreview() (string, func() *CharSpec) {
	list := g.wardPickList()
	it := list[clampI(wardPickSel, 0, len(list)-1)]
	id := g.equipOf(wardP, "base")
	if !it.locked {
		id = it.id
	}
	return "wdp:" + g.equipSigOf(wardP) + "|" + id, func() *CharSpec { return g.playerSpecFor(wardP, "base", id) }
}

// drawWardPickInfo: the description of the selected cell under the mannequin, at most wpkInfoLines lines.
func (g *Game) drawWardPickInfo(dst *ebiten.Image) {
	for i, l := range g.wrapLines(g.wardPickDesc(), 250, wpkInfoLines) {
		g.drawText(dst, l, 650, 396+float64(i)*14, rgb(0xb8b0d0), 1)
	}
}

const (
	wpkInfoLines = 5
	wpkFeet      = 48.0 // the sprite (44 px) stands on y+48
	wpkNameDY    = 52.0 // the name under the sprite (16 px high: 6 px left below)
)

// wrapLines breaks an (already translated) text into at most max lines of width w; the last one is cut with ".".
func (g *Game) wrapLines(s string, w float64, max int) []string {
	var lines []string
	line := ""
	for _, wd := range strings.Fields(s) {
		t := wd
		if line != "" {
			t = line + " " + wd
		}
		if g.textW(t, 1) > w && line != "" {
			lines = append(lines, line)
			line = wd
		} else {
			line = t
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) > max {
		lines = lines[:max]
		lines[max-1] = g.fitText(lines[max-1]+" …", 1, w)
	}
	for i := range lines {
		lines[i] = g.fitText(lines[i], 1, w)
	}
	return lines
}

// ---- in the game ---------------------------------------------------------------------------

// playerGait: a player who wears a character walks and dances with that character's gait (gait.go), like the
// guest would. Only the drawing changes: the moonwalker looks the other way while he slides.
func (g *Game) playerGait(p *Player, anim int, phase float64) (int, float64, Dir) {
	d := p.dir
	pl := 0
	if p != nil && p == g.p2 {
		pl = 1
	}
	base := g.equipOf(pl, "base")
	if p.cs == nil || p.cs.Spec == nil || !wardBaseValid(base) || p.cs.Spec.Name != base {
		return anim, phase, d
	}
	gait := gaitSlugs[base]
	if gait == "" {
		return anim, phase, d
	}
	anim, phase = gaitAnim(gait, anim, phase)
	if gait == "moon" && p.moving {
		switch d {
		case DirLeft:
			d = DirRight
		case DirRight:
			d = DirLeft
		}
	}
	return anim, phase, d
}

// ---- debug ---------------------------------------------------------------------------------

// debugWearChars (-wearchar slug[,slug2]): player 1 (and 2) wear these characters, in memory only (progress frozen).
func (g *Game) debugWearChars(list string) {
	for i, slug := range strings.Split(list, ",") {
		slug = strings.TrimSpace(slug)
		if i > 1 || slug == "" || presetByName(slug) == nil {
			continue
		}
		shopOwned["skin:"+slug] = true
		g.progress.Owned = append(g.progress.Owned, "skin:"+slug) // lent (progress frozen): survives syncShop
		if i == 0 {
			if g.progress.Equip == nil {
				g.progress.Equip = map[string]string{}
			}
			g.progress.Equip["base"] = slug
		} else {
			if g.progress.EquipP2 == nil {
				g.progress.EquipP2 = map[string]string{}
			}
			g.progress.EquipP2["base"] = slug
		}
	}
	g.applyShop()
}

// debugWardPick (-wardpick N, with -wardrobe): open the picker; 1 = every card won (like DEBUG > Álbum: todo
// conseguido), 2 = a few won, the rest locked. sel (>0) selects that cell.
func (g *Game) debugWardPick(n, sel int) {
	switch n {
	case 1:
		debugAlbum(&g.progress, 3)
	case 2:
		debugAlbum(&g.progress, 4)
		for _, s := range []string{"homer", "michael_jackson", "capitan_haddock", "met_cantante", "vader"} {
			g.progress.Album[s] = albumSeen | albumElim | albumKilled
		}
		g.progress.CharKills["lisa"] = 3
		g.progress.Album["lisa"] = albumSeen | albumElim
	}
	g.syncShop()
	g.applyShop()
	wardKind, wardRow = wardKindPlayer, 0
	g.wardPickOpen()
	if sel > 0 {
		wardPickSel = clampI(sel, 0, len(g.wardPickList())-1)
		wardPickKeepInView(len(g.wardPickList()))
	}
}

func init() {
	for k, v := range map[string]string{
		"Por defecto": "Default",
		"Tu aspecto de siempre, con la ropa que elijas.":        "Your usual look, with the clothes you pick.",
		"Lo compraste en la tienda antigua: sigue siendo tuyo.": "You bought it in the old shop: still yours.",
	} {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
}
