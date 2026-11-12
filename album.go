package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// GOALS (OBJETIVOS, formerly TROFEOS) = the trophy list + three card collections: PERSONAJES, VIPS and ASESINOS.
// What you earn by playing lives here; what you buy lives in the shop. Progress.Album maps a card key to bits:
// albumSeen, albumKilled, albumHelped.
//   VIPS: key "vip:<id>", 50 of them (vipRegistry); unlocked by killing him INSIDE his VIP area (vipOnKill) or helping
//         him (vipInteract, the ones who tip).
//   PERSONAJES: key = the character slug (goalchars.go: the 282 of the old shop, kids, album extras, trophy rewards);
//         unlocked by killing a guest of that kind the first time. Seeing him only shows his card as VISTO.
//   ASESINOS: key "killer:<slug>" (goalkillers.go: 21 + 4 trophy rewards); unlocked by killing that killer.
// The tabs are data (goalTabs) and the cards come from goalCardSources: a new tab (the police, for instance) is one
// entry in each plus its unlock rule in albumUnlocked. A locked card is a dark silhouette with "???". Old saves have
// no "album" field: it loads as an empty map, and what they bought in the shop still counts (charUnlocked, killerGot).
//
// Secret trophies: ordinary entries of trophyList (so nothing of the old logic changes) listed in secretTrophy.
// They show as "???" until won, are not needed for the legend trophy, and are won through the usual stats
// (a few counters are written here: see secretTick).
//
// -albumdemo N (debug, progress frozen): N=1 a few cards unlocked, N=2 every card.
// ---------------------------------------------------------------------------

const (
	albumSeen   = 1
	albumKilled = 2
	albumHelped = 4
	albumElim   = 8 // a character killed at least once but fewer than charKillsNeed times (ELIMINADO; goalchars.go)

	albumCols  = 5
	albumRows  = 3 // visible rows
	albumCardW = 126.0
	albumCardH = 104.0
	albumGapX  = 10.0
	albumGapY  = 6.0
	albumX0    = 244.0
	albumY0    = 123.0 // under the row of tabs (goalTabY)
)

// the kinds of card, one per collection tab
const (
	cardChar = iota
	cardVIP
	cardKiller
	cardCop
)

type albumCard struct {
	key      string
	nameES   string
	cardES   string // the name as written on the card (short when the full one does not fit); detail line uses nameES
	roleES   string
	roleCard string // the line under the name on the card when it differs from roleES (band members: the band)
	vip      bool
	vipIdx   int
	slug     string
	tipsVIP  bool
	roleKind int
	killer   bool   // an ASESINOS card (slug = the killer preset)
	cop      bool   // a POLICÍAS card (roleKind = the unit, goalcops.go)
	trophyID string // a reward of this trophy: it unlocks with the trophy, not by a kill
}

func (c albumCard) kind() int {
	switch {
	case c.vip:
		return cardVIP
	case c.killer:
		return cardKiller
	case c.cop:
		return cardCop
	}
	return cardChar
}

var albumList []albumCard

// albumShortName: short card names of the special characters whose full name is wider than a card.
var albumShortName = map[string]string{"taz": "Tazz", "stan_barcos": "Stann", "pantera_rosa": "Pantera Rossa", "lagarto_juancho": "Juanchi",
	"caperucita_roja": "Caperucita", "colon_almirante": "Colón", "isabel_la_catolica": "Ysabel",
	"tr_bailarina_disco": "Bailarina", "tr_rockero_leyenda": "Rockero", "tr_rey_fiesta": "Rey Fiesta"}

// cardName: what the card prints as the name.
func (c albumCard) cardName() string {
	if c.cardES != "" {
		return c.cardES
	}
	return c.nameES
}

// goalCardSources: where the cards of every collection come from (a new collection adds its function here).
var goalCardSources = []func() []albumCard{vipCards, charCards, killerCards, copCards}

// albumCards builds the list once: the VIPs, then the characters, then the killers.
func albumCards() []albumCard {
	if albumList != nil {
		return albumList
	}
	out := []albumCard{}
	for _, src := range goalCardSources {
		out = append(out, src()...)
	}
	albumList = out
	return out
}

func vipCards() []albumCard {
	var out []albumCard
	for i, d := range vipRegistry() {
		out = append(out, albumCard{key: "vip:" + d.id, nameES: d.es, cardES: d.cardES, roleES: d.rankES, vip: true, vipIdx: i, tipsVIP: d.behaviour == vbTip})
	}
	return out
}

var albumKeyIdx map[string]string

// albumKeyOf: the card key of a character (guest) slug, "" when it has no card.
func albumKeyOf(slug string) string {
	if albumKeyIdx == nil {
		albumKeyIdx = map[string]string{}
		for _, c := range albumCards() {
			if c.kind() == cardChar {
				albumKeyIdx[c.slug] = c.key
			}
		}
	}
	return albumKeyIdx[slug]
}

func (g *Game) albumBits(c albumCard) int { return g.progress.Album[c.key] }

// albumUnlocked: the card is won. A VIP needs a kill inside his area or a favour; a character or a killer needs a
// kill (or the old shop purchase / the trophy that gives it: charUnlocked, killerGot). Just seeing one is not enough.
func (g *Game) albumUnlocked(c albumCard) bool {
	b := g.albumBits(c)
	switch c.kind() {
	case cardVIP:
		return b&(albumKilled|albumHelped) != 0
	case cardKiller:
		return g.killerGot(c.slug)
	case cardCop:
		return g.copGot(c.roleKind)
	}
	return charUnlocked(g.progress.Album, c.slug)
}

// albumSeenOnly: a character seen close up, or killed fewer than charKillsNeed times, but not won yet (its card
// shows his face, marked VISTO or ELIMINADO n/5, and does not count).
func (g *Game) albumSeenOnly(c albumCard) bool {
	return c.kind() == cardChar && !g.albumUnlocked(c) && g.albumBits(c)&(albumSeen|albumElim) != 0
}

func (g *Game) albumCount() int {
	n := 0
	for _, c := range albumCards() {
		if g.albumUnlocked(c) {
			n++
		}
	}
	return n
}

// albumMark sets a bit of a card, announces a new card, feeds the secret trophy and saves.
func (g *Game) albumMark(key string, bit int) {
	if key == "" {
		return
	}
	if g.progress.Album == nil {
		g.progress.Album = map[string]int{}
	}
	old := g.progress.Album[key]
	if old&bit == bit {
		return
	}
	before := g.albumCount()
	g.progress.Album[key] = old | bit
	if charKey(key) && bit&albumKilled != 0 {
		shopOwned["skin:"+key] = true // wardrobe, kids and riders see him at once (charSyncOwned does it on load)
	}
	if g.albumCount() > before {
		g.say("Cromo nuevo en el álbum")
		g.statMax("album", g.albumCount())
	}
	saveProgress(g.progress)
}

// albumSeeR: how close a player must be to a character for the card to count as "seen".
const albumSeeR = 150.0

// albumSeenBy: a character is seen when a player stands within albumSeeR of him with no opaque wall between them
// (the camera shows the whole level, so being on screen proves nothing).
func (g *Game) albumSeenBy(pos Vec) bool {
	if g.lvl == nil {
		return false
	}
	for _, p := range g.players() {
		if p == nil {
			continue
		}
		if dist(p.pos, pos) <= albumSeeR && g.lvl.LOS(p.pos, pos) {
			return true
		}
	}
	return false
}

// albumScan: every half second, a named special character close to a player and in his line of sight is "seen".
func (g *Game) albumScan(dt float64) {
	g.sec.scanT -= dt
	if g.sec.scanT > 0 || g.mode != ModePlay {
		return
	}
	g.sec.scanT = 0.5
	for _, gu := range g.guests {
		if gu == nil || gu.dead || gu.cs == nil || gu.cs.Spec == nil || gu.cs.Spec.Name == "" {
			continue
		}
		if !g.albumSeenBy(gu.pos) {
			continue
		}
		if k := albumKeyOf(gu.cs.Spec.Name); k != "" {
			g.albumMark(k, albumSeen)
		}
	}
}

// ---- secret trophies -------------------------------------------------------------------------------------

type secretState struct {
	scanT     float64
	blk       bool
	blkKills  int
	dance     [2]bool
	trapT     float64
	trapN     int
	trapGiven bool
}

var secretTrophy = map[string]bool{}

func init() {
	sec := []Trophy{
		{"s_vipdance", "Baile de gala", "Baila a menos de 40 px de un VIP.", "vipdance", 1, evr("secret:vipdance", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_vip3", "Cazador de coronas", "Elimina a 3 VIP en total.", "vips", 3, evr("secret:vip3", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_vipfriend", "Amigo de los poderosos", "Ayuda a un VIP que da propina (E).", "vipfriend", 1, evr("secret:vipfriend", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_vippass", "Con invitación, por favor", "Entra en una zona VIP con una invitación robada.", "vippass", 1, evr("secret:vippass", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_suit3", "Brillo dorado", "Usa el traje dorado 3 veces.", "suits", 3, evr("secret:suit3", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_mine", "Mina perfecta", "Mata a 3 personas con una sola trampa.", "minex", 1, evr("secret:mine", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_chain3", "Cadena x3", "Encadena 3 muertes seguidas.", "chain", 3, evr("secret:chain3", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_shop75", "Tarjeta al rojo", "Haz 75 compras en la tienda.", "bought", 75, evr("secret:shop75", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_dark", "Luces fuera, manos quietas", "Aguanta un apagón sin matar a nadie.", "blackpeace", 1, evr("secret:dark", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
		{"s_album15", "Cromos y más cromos", "Consigue 15 cromos del álbum.", "album", 15, evr("secret:album15", "Medalla secreta", "Un honor discreto. No lo cuentes.")},
	}
	at := len(trophyList)
	for i, t := range trophyList {
		if t.id == legendTrophy {
			at = i
		}
	}
	rest := append([]Trophy(nil), trophyList[at:]...)
	trophyList = append(append(trophyList[:at:at], sec...), rest...)
	for _, t := range sec {
		secretTrophy[t.id] = true
	}
}

// secretHidden: a secret trophy not won yet shows as "???".
func (g *Game) secretHidden(t Trophy) bool { return secretTrophy[t.id] && !g.trophyDone(t) }

// secretTick runs every frame of a party: the scan for the album and the counters of the secret trophies.
func (g *Game) secretTick(dt float64) {
	g.albumScan(dt)
	s := &g.sec
	// dancing next to a VIP
	for i, p := range g.players() {
		if p == nil || i > 1 {
			continue
		}
		if p.danceT <= 0 {
			s.dance[i] = false
			continue
		}
		if s.dance[i] {
			continue
		}
		for _, gd := range g.guards {
			if gd.vip == vipBoss && !gd.dead && dist(p.pos, gd.pos) < 40 {
				s.dance[i] = true
				g.stat("vipdance", 1)
				break
			}
		}
	}
	// a blackout survived without killing anybody
	k := careerKills(&g.progress) + g.progress.Stats["traps"] // a trap death in the dark also breaks it
	if g.blk.t > 0 {
		if !s.blk {
			s.blk, s.blkKills = true, k
		} else if k > s.blkKills {
			s.blk = false // somebody died in the dark
		}
	} else if s.blk {
		s.blk = false
		if g.blk.done && g.mode == ModePlay {
			g.stat("blackpeace", 1)
		}
	}
}

// secretTrapKill: three kills within a short burst of a trap count as one perfect mine.
func (g *Game) secretTrapKill() {
	s := &g.sec
	if g.time-s.trapT <= 0.6 {
		s.trapN++
	} else {
		s.trapN, s.trapGiven = 1, false
	}
	s.trapT = g.time
	if s.trapN >= 3 && !s.trapGiven {
		s.trapGiven = true
		g.stat("minex", 1)
	}
}

// ---- the screen ---------------------------------------------------------------------------------------------

const (
	tabTrophies = 0
	tabChars    = 1 // PERSONAJES
	tabVIPs     = 2
	tabKillers  = 3
	tabCops     = 4
	tabCareer   = 5 // CARRERA: the totals of the whole career (goalstats.go)
	tabCount    = 6
)

// goalTab: one page of OBJETIVOS. kind is the card kind it lists (-1 = the list of trophies).
type goalTab struct {
	name string // Spanish label; tr() gives the English one
	kind int
}

var goalTabs = [tabCount]goalTab{{"TROFEOS", -1}, {"PERSONAJES", cardChar}, {"VIPS", cardVIP}, {"ASESINOS", cardKiller}, {"POLICÍAS", cardCop}, {"CARRERA", -2}}

// the row of tabs sits under the title and the toast column (the toast covers y 16-80), above the list
const (
	goalTabY = 91.0
	goalTabH = 20.0
)

// goalTabRects: x, y, w, h of every tab, laid out from the width of its label in the current language.
func (g *Game) goalTabRects() [tabCount][4]float64 {
	var out [tabCount][4]float64
	x := 244.0 // the title's column (and the cards')
	for i, t := range goalTabs {
		w := math.Ceil(g.textW(t.name, 1)) + 14
		out[i] = [4]float64{x, goalTabY, w, goalTabH}
		x += w + 6
	}
	return out
}

// albumView: the cards of the page in view (none on the trophy list).
func (g *Game) albumView() []albumCard {
	all := albumCards()
	k := goalTabs[clampI(g.trophyTab, 0, tabCount-1)].kind
	out := make([]albumCard, 0, len(all))
	if k < 0 {
		return out
	}
	// the order: first the cards never met (???), then the ones met or killed but not won yet, the won ones last;
	// stable inside every group (catalogue order)
	var grp [4][]albumCard
	for _, c := range all {
		if c.kind() == k {
			gi := g.albumGroup(c)
			grp[gi] = append(grp[gi], c)
		}
	}
	for _, l := range grp {
		out = append(out, l...)
	}
	return out
}

// albumGroup: 0 = never met (???), 1 = seen only (VISTO), 2 = killed but fewer times than needed (ELIMINADO), 3 = won.
func (g *Game) albumGroup(c albumCard) int {
	switch {
	case g.albumUnlocked(c):
		return 3
	case g.albumBits(c)&albumElim != 0:
		return 2
	case g.albumBits(c) != 0:
		return 1
	}
	return 0
}

func (g *Game) albumClamp() {
	n := len(g.albumView())
	g.albumSel = clampI(g.albumSel, 0, maxI(n-1, 0))
	row := g.albumSel / albumCols
	if row < g.albumScroll {
		g.albumScroll = row
	}
	if row >= g.albumScroll+albumRows {
		g.albumScroll = row - albumRows + 1
	}
	maxScroll := maxI(0, (n+albumCols-1)/albumCols-albumRows)
	g.albumScroll = clampI(g.albumScroll, 0, maxScroll)
}

func (g *Game) goTab(i int) {
	g.trophyTab = clampI(i, 0, tabCount-1)
	g.albumSel, g.albumScroll = 0, 0
}

// updateAlbumTabs: the tabs (mouse, Tab key) and the card navigation. It returns true when a card page is the
// page in view (the trophy list must not move then).
func (g *Game) updateAlbumTabs() bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for i, r := range g.goalTabRects() {
			if float64(mx) >= r[0] && float64(mx) <= r[0]+r[2] && float64(my) >= r[1] && float64(my) <= r[1]+r[3] {
				if g.trophyTab != i {
					g.goTab(i)
				}
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.goTab((g.trophyTab + tabCount - 1) % tabCount)
		} else {
			g.goTab((g.trophyTab + 1) % tabCount)
		}
	}
	if g.trophyTab == tabTrophies {
		if actRight.just() {
			g.goTab(tabChars)
			return true
		}
		return false
	}
	view := g.albumView()
	n := len(view)
	clicked := g.mouseMenu(func(mx, my float64) int {
		for i := 0; i < albumCols*albumRows; i++ {
			idx := (g.albumScroll+i/albumCols)*albumCols + i%albumCols
			x, y := albumCardXY(i)
			if idx < n && mx >= x && mx <= x+albumCardW && my >= y && my <= y+albumCardH {
				return idx
			}
		}
		return -1
	}, &g.albumSel)
	if _, wy := ebiten.Wheel(); wy != 0 {
		g.albumSel = clampI(g.albumSel-int(wy)*albumCols, 0, n-1)
	}
	// the mouse resting on the top or bottom strip of the grid scrolls it (listscroll.go)
	g.albumSel, g.albumScroll = edgeListScroll("goals", g.albumSel, g.albumScroll, albumRows, albumCols, n,
		[4]float64{albumX0, albumY0, albumCols*(albumCardW+albumGapX) - albumGapX, albumRows*(albumCardH+albumGapY) - albumGapY}, 2.5)
	switch {
	case actRight.just():
		if g.albumSel >= n-1 { // past the last card: the next tab
			if g.trophyTab < tabCount-1 {
				g.goTab(g.trophyTab + 1)
			}
		} else {
			g.albumSel = minI(g.albumSel+1, n-1)
		}
	case actLeft.just():
		if g.albumSel%albumCols == 0 {
			g.goTab(g.trophyTab - 1)
			return g.trophyTab != tabTrophies
		}
		g.albumSel--
	case actDown.just():
		g.albumSel = minI(g.albumSel+albumCols, n-1)
	case actUp.just():
		g.albumSel = maxI(g.albumSel-albumCols, 0)
	case inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
		g.albumSel = minI(g.albumSel+albumCols*albumRows, n-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
		g.albumSel = maxI(g.albumSel-albumCols*albumRows, 0)
	}
	g.albumClamp()
	if n > 0 && (clicked || goalDetailAsk()) { // the detail window of the card (goal_detail.go)
		g.openGoalDetail()
	}
	return true
}

func albumCardXY(slot int) (float64, float64) {
	return albumX0 + float64(slot%albumCols)*(albumCardW+albumGapX), albumY0 + float64(slot/albumCols)*(albumCardH+albumGapY)
}

// drawTabs: the title and the row of tabs of the OBJETIVOS screen.
func (g *Game) drawTabs(dst *ebiten.Image) {
	g.drawText(dst, "OBJETIVOS", 244, 66, rgb(0xff6a8a), 1.6)
	for i, r := range g.goalTabRects() {
		bg, fg := rgba(0x120c24, 220), rgb(0x8a86a0)
		if g.trophyTab == i {
			bg, fg = rgba(0x3a2a6a, 235), rgb(0xf4f0ff)
		}
		fillRect(dst, r[0], r[1], r[2], r[3], bg)
		fillRect(dst, r[0], r[1]+r[3]-2, r[2], 2, rgba(0x6a4ab0, 200))
		g.drawText(dst, goalTabs[i].name, r[0]+7, r[1]+6, fg, 1)
	}
}

func (g *Game) fitText(s string, scale, w float64) string {
	s = tr(s)
	if g.textW(s, scale) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 1 {
		r = r[:len(r)-1]
		if g.textW(string(r)+".", scale) <= w {
			break
		}
	}
	return string(r) + "."
}

func (g *Game) albumCS(c albumCard) *CharSet {
	return g.shopCS("album:"+c.key, func() *CharSpec {
		switch c.kind() {
		case cardVIP:
			s := vipRegistry()[c.vipIdx].spec()
			return &s
		case cardKiller:
			s := killerSpecOf(c.slug)
			return &s
		case cardCop:
			s := copSpecOf(c.roleKind)
			return &s
		}
		s := presetByName(c.slug)(rand.New(rand.NewSource(1)))
		return &s
	})
}

// drawAlbumPanel: the card grid (inside the same panel as the trophy list).
func (g *Game) drawAlbumPanel(dst *ebiten.Image) {
	cards := g.albumView()
	g.albumClamp()
	got := 0
	for _, c := range cards {
		if g.albumUnlocked(c) {
			got++
		}
	}
	g.drawText(dst, fmt.Sprintf("%d / %d", got, len(cards)), 770, 66, rgb(0xf0e060), 1.6)
	if goalTabs[clampI(g.trophyTab, 0, tabCount-1)].kind == cardCop { // the escapes that open the uniforms (copsets.go)
		line := tr(g.copEscapeLine())
		g.drawText(dst, line, 914-math.Ceil(g.textW(line, 1)), goalTabY+6, rgb(0x70e0c0), 1)
	}
	for i := 0; i < albumCols*albumRows; i++ {
		idx := (g.albumScroll+i/albumCols)*albumCols + i%albumCols
		if idx >= len(cards) {
			break
		}
		x, y := albumCardXY(i)
		g.drawAlbumCard(dst, cards[idx], x, y, idx == g.albumSel)
	}
	// scroll bar
	totalRows := (len(cards) + albumCols - 1) / albumCols
	if totalRows > albumRows {
		trackH := albumRows*(albumCardH+albumGapY) - albumGapY
		fillRect(dst, 920, albumY0, 4, trackH, rgba(0x2a2440, 200))
		th := trackH * float64(albumRows) / float64(totalRows)
		ty := albumY0 + (trackH-th)*float64(g.albumScroll)/float64(totalRows-albumRows)
		fillRect(dst, 920, ty, 4, th, rgb(0x8a6ad0))
	}
	// the selected card, in words
	c := cards[clampI(g.albumSel, 0, len(cards)-1)]
	line := "???"
	switch {
	case c.kind() == cardChar:
		line = g.charDetailLine(c) // own description + kills left (goalchars.go)
	case g.albumUnlocked(c):
		line = tr(c.nameES) + " - " + tr(c.roleES)
	case g.albumSeenOnly(c):
		line = tr(c.nameES) + " - " + tr("Visto: mátalo para conseguirlo")
	default:
		line = "??? - " + tr(c.lockedHint())
	}
	g.drawText(dst, g.fitText(line, 1.2, 676), 244, 459, rgb(0xf0e060), 1.2)
}

// lockedHint: how to win a locked card (Spanish; the caller translates it).
func (c albumCard) lockedHint() string {
	switch {
	case c.vip:
		return "Elimínalo dentro de su zona VIP"
	case c.trophyID != "":
		return "Se gana con un trofeo (pestaña TROFEOS)"
	case c.killer:
		return "Derrota a este asesino especial"
	case c.cop:
		return copLockedHint(c.roleKind)
	}
	return "Mata a un invitado de este tipo"
}

func (g *Game) drawAlbumCard(dst *ebiten.Image, c albumCard, x, y float64, sel bool) {
	ok := g.albumUnlocked(c)
	seen := g.albumSeenOnly(c) // seen close up, not killed yet: the face shows, the card is not won
	frame := rgb(0x3a3450)
	if ok {
		frame = rgb(0xc8a030)
		if c.vip {
			frame = rgb(0xe8c040)
		}
	}
	if sel {
		frame = rgb(0xff6a8a)
	}
	fillRect(dst, x-2, y-2, albumCardW+4, albumCardH+4, frame)
	fillRect(dst, x, y, albumCardW, albumCardH, rgb(0x120c24))
	// portrait box: foil gradient (gold/violet for the unlocked ones, dull for the locked ones)
	const ph = 62.0
	for r := 0.0; r < ph; r += 2 {
		f := r / ph
		var col color.RGBA
		if ok {
			if c.vip {
				col = color.RGBA{uint8(70 + 90*f), uint8(40 + 60*f), uint8(90 - 40*f), 255}
			} else if c.cop {
				col = color.RGBA{uint8(30 + 20*f), uint8(50 + 50*f), uint8(100 + 70*f), 255}
			} else if c.killer {
				col = color.RGBA{uint8(70 + 60*f), uint8(26 + 20*f), uint8(34 + 10*f), 255}
			} else {
				col = color.RGBA{uint8(40 + 30*f), uint8(60 + 70*f), uint8(110 + 40*f), 255}
			}
		} else {
			col = color.RGBA{uint8(24 + 10*f), uint8(22 + 8*f), uint8(34 + 12*f), 255}
		}
		fillRect(dst, x, y+r, albumCardW, 2, col)
	}
	if ok { // a diagonal shine
		for k := 0.0; k < ph; k += 6 {
			fillRect(dst, x+albumCardW-30-k, y+k, 8, 2, rgba(0xffffff, 40))
		}
	}
	if im, flip := g.albumCS(c).img(AnimIdle, DirDown, 0); im != nil {
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		s := 1.0
		for (s+1)*h <= ph-4 && s < 4 {
			s++
		}
		op := &ebiten.DrawImageOptions{}
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(float64(int(x+albumCardW/2-w*s/2)), float64(int(y+ph-2-h*s)))
		if !ok && !seen {
			op.ColorScale.Scale(0.06, 0.05, 0.1, 1) // dark silhouette
		}
		dst.DrawImage(im, op)
	}
	name, role := "???", "???"
	if ok || seen {
		name, role = c.cardName(), c.roleES
		if c.roleCard != "" {
			role = c.roleCard
		}
	}
	ncol := rgb(0xf4f0ff)
	if !ok {
		ncol = rgb(0x6a6680)
		if seen {
			ncol = rgb(0xb8b4d0)
		}
	}
	g.drawText(dst, g.fitText(name, 1, albumCardW-8), x+4, y+ph+3, ncol, 1)
	g.drawText(dst, g.fitText(role, 1, albumCardW-8), x+4, y+ph+15, rgb(0x9a96b0), 1)
	if seal, scol := g.albumSeal(c); seal != "" { // the stamp (goal_detail.go, shared with the detail window)
		g.drawText(dst, seal, x+4, y+ph+27, scol, 1)
	}
}

// albumDemo fills the album in memory for the screenshots (-albumdemo N).
func (g *Game) albumDemo(n int) {
	g.progress.Album = map[string]int{}
	for i, c := range albumCards() {
		bit := albumSeen
		if i%3 == 0 {
			bit = albumKilled
		} else if i%3 == 1 && !c.vip && !c.killer && !c.cop { // killed a few times, not won yet (ELIMINADO n/5)
			bit = albumSeen | albumElim
			if g.progress.CharKills == nil {
				g.progress.CharKills = map[string]int{}
			}
			g.progress.CharKills[c.key] = 1 + i%(charKillsNeed-1)
		}
		if c.vip {
			bit = albumKilled
			if c.tipsVIP {
				bit = albumHelped
			}
		}
		if c.killer || c.cop {
			bit = albumKilled
		}
		if n >= 2 || i%4 == 0 || (c.vip && i%2 == 0) {
			g.progress.Album[c.key] = bit
		}
	}
	g.trophyTab = tabChars
}
