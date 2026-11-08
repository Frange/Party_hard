package main

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// COLLECTOR CARD ALBUM (idea 25) and SECRET TROPHIES (idea 47).
//
// The second page of the TROFEOS screen (and of its in-game window) is an album with one card per VIP of
// vipRegistry() and one per special character (famous guests of specialPool / starGroups, the three kids Sam-sam,
// Cesar and Diego included). Progress.Album maps a card key to bits: albumSeen, albumKilled, albumHelped.
//   VIP keys are "vip:<id>"; the rest use the character slug.
//   A VIP card is unlocked by killing him (vipOnKill) or helping him (vipInteract, the ones who tip).
//   A special card is unlocked by seeing the character on screen in a party (albumScan) or by killing him.
// A locked card is a dark silhouette with "???". Old saves have no "album" field: it loads as an empty map.
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

	albumCols  = 5
	albumRows  = 3 // visible rows
	albumCardW = 126.0
	albumCardH = 112.0
	albumGapX  = 10.0
	albumGapY  = 6.0
	albumX0    = 244.0
	albumY0    = 94.0
)

type albumCard struct {
	key      string
	nameES   string
	cardES   string // the name as written on the card (short when the full one does not fit); detail line uses nameES
	roleES   string
	vip      bool
	vipIdx   int
	slug     string
	tipsVIP  bool
	roleKind int
}

var albumList []albumCard

// albumShortName: short card names of the special characters whose full name is wider than a card.
var albumShortName = map[string]string{"taz": "Taz", "stan_barcos": "Stan", "pantera_rosa": "Pantera Rosa", "lagarto_juancho": "Juancho"}

// cardName: what the card prints as the name.
func (c albumCard) cardName() string {
	if c.cardES != "" {
		return c.cardES
	}
	return c.nameES
}

// albumCards builds the list once: 11 VIP, then the kids, then every other special character.
func albumCards() []albumCard {
	if albumList != nil {
		return albumList
	}
	out := []albumCard{}
	for i, d := range vipRegistry() {
		out = append(out, albumCard{key: "vip:" + d.id, nameES: d.es, cardES: d.cardES, roleES: d.rankES, vip: true, vipIdx: i, tipsVIP: d.behaviour == vbTip})
	}
	seen := map[string]bool{}
	probe := rand.New(rand.NewSource(1))
	add := func(slug, role string) {
		if seen[slug] {
			return
		}
		f := presetByName(slug)
		if f == nil {
			return
		}
		seen[slug] = true
		spec := f(probe)
		nm := nameFromSpec(&spec)
		out = append(out, albumCard{key: slug, nameES: nm, cardES: albumShortName[slug], roleES: role, slug: slug})
	}
	for _, k := range kidSlugs {
		add(k, "Peque travieso")
	}
	for _, s := range specialPool {
		add(s, "Invitado famoso")
	}
	for _, grp := range starGroups {
		for _, s := range grp {
			add(s, "Invitado famoso")
		}
	}
	albumList = out
	return out
}

func albumKeyOf(slug string) string {
	for _, c := range albumCards() {
		if !c.vip && c.slug == slug {
			return c.key
		}
	}
	return ""
}

func (g *Game) albumBits(c albumCard) int { return g.progress.Album[c.key] }

func (g *Game) albumUnlocked(c albumCard) bool {
	b := g.albumBits(c)
	if c.vip {
		return b&(albumKilled|albumHelped) != 0
	}
	return b != 0
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
	k := g.progress.Stats["kills"] + g.progress.Stats["traps"] + g.progress.Stats["poison"]
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
	tabAlbum    = 1
)

var (
	tabRects = [2][4]float64{{244, 62, 56, 22}, {304, 62, 46, 22}}
	tabNames = [2]string{"TROFEOS", "ÁLBUM"}
)

func (g *Game) albumClamp() {
	n := len(albumCards())
	g.albumSel = clampI(g.albumSel, 0, n-1)
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

// updateAlbumTabs: the two tabs (mouse, Tab key) and the album navigation. It returns true when the album is
// the page in view (the trophy list must not move then).
func (g *Game) updateAlbumTabs() bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for i, r := range tabRects {
			if float64(mx) >= r[0] && float64(mx) <= r[0]+r[2] && float64(my) >= r[1] && float64(my) <= r[1]+r[3] {
				g.trophyTab = i
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.trophyTab = 1 - g.trophyTab
	}
	if g.trophyTab == tabTrophies {
		if actRight.just() {
			g.trophyTab = tabAlbum
			return true
		}
		return false
	}
	n := len(albumCards())
	g.mouseMenu(func(mx, my float64) int {
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
	switch {
	case actRight.just():
		g.albumSel = minI(g.albumSel+1, n-1)
	case actLeft.just():
		if g.albumSel%albumCols == 0 {
			g.trophyTab = tabTrophies
			return false
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
	return true
}

func albumCardXY(slot int) (float64, float64) {
	return albumX0 + float64(slot%albumCols)*(albumCardW+albumGapX), albumY0 + float64(slot/albumCols)*(albumCardH+albumGapY)
}

// drawTabs: the two tabs of the TROFEOS screen.
func (g *Game) drawTabs(dst *ebiten.Image) {
	for i, r := range tabRects {
		bg, fg := rgba(0x120c24, 220), rgb(0x8a86a0)
		if g.trophyTab == i {
			bg, fg = rgba(0x3a2a6a, 235), rgb(0xf4f0ff)
		}
		fillRect(dst, r[0], r[1], r[2], r[3], bg)
		fillRect(dst, r[0], r[1]+r[3]-2, r[2], 2, rgba(0x6a4ab0, 200))
		g.drawTextC(dst, tabNames[i], r[0]+r[2]/2, r[1]+3, fg, 1)
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
		if c.vip {
			s := vipRegistry()[c.vipIdx].spec()
			return &s
		}
		s := presetByName(c.slug)(rand.New(rand.NewSource(1)))
		return &s
	})
}

// drawAlbumPanel: the card grid (inside the same panel as the trophy list).
func (g *Game) drawAlbumPanel(dst *ebiten.Image) {
	cards := albumCards()
	g.albumClamp()
	g.drawText(dst, fmt.Sprintf("%d / %d", g.albumCount(), len(cards)), 770, 66, rgb(0xf0e060), 1.6)
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
	if g.albumUnlocked(c) {
		line = tr(c.nameES) + " - " + tr(c.roleES)
	}
	g.drawText(dst, g.fitText(line, 1.2, 676), 244, 448, rgb(0xf0e060), 1.2)
	g.drawText(dst, "Flechas mover   Tab o izquierda: trofeos   "+actionHint(actBack)+" volver", 244, 472, rgb(0x8a86a0), 1)
}

func (g *Game) drawAlbumCard(dst *ebiten.Image, c albumCard, x, y float64, sel bool) {
	ok := g.albumUnlocked(c)
	bits := g.albumBits(c)
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
	const ph = 66.0
	for r := 0.0; r < ph; r += 2 {
		f := r / ph
		var col color.RGBA
		if ok {
			if c.vip {
				col = color.RGBA{uint8(70 + 90*f), uint8(40 + 60*f), uint8(90 - 40*f), 255}
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
		if !ok {
			op.ColorScale.Scale(0.06, 0.05, 0.1, 1) // dark silhouette
		}
		dst.DrawImage(im, op)
	}
	name, role := "???", "???"
	if ok {
		name, role = c.cardName(), c.roleES
	}
	ncol := rgb(0xf4f0ff)
	if !ok {
		ncol = rgb(0x6a6680)
	}
	g.drawText(dst, g.fitText(name, 1, albumCardW-8), x+4, y+ph+4, ncol, 1)
	g.drawText(dst, g.fitText(role, 1, albumCardW-8), x+4, y+ph+18, rgb(0x9a96b0), 1)
	if ok { // seal
		seal, scol := "VISTO", rgb(0x8ac0f0)
		switch {
		case bits&albumHelped != 0:
			seal, scol = "AMIGO", rgb(0x70e090)
		case bits&albumKilled != 0:
			seal, scol = "ELIMINADO", rgb(0xff6a6a)
		}
		g.drawText(dst, seal, x+4, y+ph+32, scol, 1)
	}
}

// albumDemo fills the album in memory for the screenshots (-albumdemo N).
func (g *Game) albumDemo(n int) {
	g.progress.Album = map[string]int{}
	for i, c := range albumCards() {
		bit := albumSeen
		if i%3 == 0 {
			bit = albumKilled
		}
		if c.vip {
			bit = albumKilled
			if c.tipsVIP {
				bit = albumHelped
			}
		}
		if n >= 2 || i%4 == 0 || (c.vip && i%2 == 0) {
			g.progress.Album[c.key] = bit
		}
	}
	g.trophyTab = tabAlbum
}
