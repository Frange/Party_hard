package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// The wardrobe: BUYING happens in the shop, WEARING happens here. Every player
// has his own saved look (progress.Equip is player 1, progress.EquipP2 is
// player 2): the base character, the clothes, a tint for the whole outfit and
// the accessories. By default player 1 is blue and player 2 is red.
//
// Perks (speed, stamina, gloves, golden wallet) are bought once and shared by
// both players: they belong to the party, not to a person.
// ---------------------------------------------------------------------------

// wardrobeNumPlayers is how many players the party has (1 or 2): the switch of the main menu
// (twoPlayerMode) or the -twop flag. A debug flag may replace this hook.
var wardrobeNumPlayers = func(g *Game) int {
	if twoPlayerMode || dbgTwoPlayers {
		return 2
	}
	return 1
}

// the look shown by the wardrobe and the shop: which player is being dressed and the row selected
var (
	wardP     int     // 0 = player 1, 1 = player 2
	wardRow   int     // selected row of the wardrobe
	wardFlash float64 // seconds left of the "new look" flash
	wardTime  float64 // spins the mannequin
)

func (g *Game) wardTwo() bool { return wardrobeNumPlayers(g) >= 2 }

// ---- the colours of each player ------------------------------------------------------

const (
	colP1 = 0x2f7bff // blue (saturated: reads on the grey floor)
	colP2 = 0xff4a4a // red
)

func playerColor(player int) color.RGBA {
	if player == 1 {
		return rgb(colP2)
	}
	return rgb(colP1)
}

// tints: the colour of the whole outfit (shirt and trousers).
var tints = []struct {
	id, name string
	col      uint32
}{
	{"azul", "Azul", 0x2c3c78}, {"rojo", "Rojo", 0xa82c2c}, {"verde", "Verde", 0x2a7a42}, {"amarillo", "Amarillo", 0xd8b828},
	{"morado", "Morado", 0x6a2a9a}, {"naranja", "Naranja", 0xe07a1c}, {"rosa", "Rosa", 0xe0508a}, {"negro", "Negro", 0x22222a}, {"blanco", "Blanco", 0xe8e8ee},
}

func tintColor(id string) (uint32, bool) {
	for _, t := range tints {
		if t.id == id {
			return t.col, true
		}
	}
	return 0, false
}

// defaultTint: what a player wears while he has chosen no outfit and no tint.
func defaultTint(player int) string {
	if player == 1 {
		return "rojo"
	}
	return "azul"
}

// paint puts the tint on the whole outfit.
func tintSpec(s *CharSpec, col uint32) {
	c := hx(col)
	s.Shirt, s.Shirt2, s.HatCol = c, c, c
	s.Pants, s.Accent = shade(c, 0.78), shade(c, 0.78)
}

// ---- the saved look of each player ---------------------------------------------------

func isWearSlot(slot string) bool {
	if slot == "base" || slot == "tint" {
		return true
	}
	for _, s := range wearSlots {
		if s == slot {
			return true
		}
	}
	return false
}

// equipOf is what a player wears in a slot. Only the clothes belong to a player: the uniforms of the
// police and the weapons of the killers are the party's, saved with player 1.
func (g *Game) equipOf(player int, slot string) string {
	if player == 1 && isWearSlot(slot) {
		if g.progress.EquipP2 == nil {
			return ""
		}
		return g.progress.EquipP2[slot]
	}
	return g.shopEquip(slot)
}

func (g *Game) setEquipOf(player int, slot, id string) {
	if player == 0 || !isWearSlot(slot) {
		g.setEquip(slot, id)
		return
	}
	if g.progress.EquipP2 == nil {
		g.progress.EquipP2 = map[string]string{}
	}
	if id == "" {
		delete(g.progress.EquipP2, slot)
		return
	}
	g.progress.EquipP2[slot] = id
}

// playerSpecFor is the character of a player dressed with what he wears; a preview item (id)
// replaces what he wears in its slot.
func (g *Game) playerSpecFor(player int, previewSlot, previewID string) *CharSpec {
	get := func(slot string) string {
		if slot == previewSlot {
			return previewID
		}
		return g.equipOf(player, slot)
	}
	spec := playerSpec()
	base := get("base")
	if base != "" && ownedID("skin:"+base) {
		if f := presetByName(base); f != nil {
			spec = f(rand.New(rand.NewSource(5)))
		}
	}
	for _, slot := range wearSlots {
		if fx := wearFx[get(slot)]; fx != nil {
			fx(&spec)
		}
	}
	// the colour of the outfit: his own choice, or blue / red while he is a plain protagonist in the base outfit
	tint := get("tint")
	if tint == "" && base == "" && (get("outfit") == "" || get("outfit") == baseOutfitID(player)) {
		tint = defaultTint(player)
	}
	if col, ok := tintColor(tint); ok {
		tintSpec(&spec, col)
	}
	return &spec
}

// equipSigOf identifies what a player wears, so the sprites are rebuilt when it changes.
func (g *Game) equipSigOf(player int) string {
	s := fmt.Sprint(player, "|")
	for _, sl := range append([]string{"base", "tint"}, wearSlots...) {
		s += g.equipOf(player, sl) + "|"
	}
	return s
}

// playerCS is the animated character of a player (0 = player 1, 1 = player 2).
// Player 1 keeps the hand drawn protagonist of the editor when there is one.
func (g *Game) playerCS(player int) *CharSet {
	if player == 0 && g.editedMainCharacter() {
		return g.as.playerCS
	}
	return g.shopCS("pl:"+g.equipSigOf(player), func() *CharSpec { return g.playerSpecFor(player, "", "") })
}

// editedMainCharacter: the user drew the protagonist by hand in the editor, so that drawing wins.
func (g *Game) editedMainCharacter() bool {
	if g.store == nil {
		return false
	}
	l := g.store.Chars["main_character"]
	return len(l) > 0 && l[0].Meta.Edited
}

// ---- the free base outfits ------------------------------------------------------------

func baseOutfitID(player int) string {
	if player == 1 {
		return "out:baser"
	}
	return "out:basea"
}

func init() {
	funnyDesc["out:basea"] = "El mono de serie del jugador 1: azul de toda la vida, sufrido y con la costura ya cansada."
	funnyDesc["out:baser"] = "El mono de serie del jugador 2: rojo pasión, con manchas que ya estaban cuando lo recogiste."
}

// addBaseOutfits puts the two free base outfits at the top of the clothes list.
func addBaseOutfits() {
	mk := func(id, name string, col uint32) shopItem {
		wearFx[id] = func(s *CharSpec) { s.Top = TopLong; tintSpec(s, col) }
		return shopItem{id: id, name: name, desc: funnyDesc[id], price: 0, slot: "outfit"}
	}
	shopCatalog[shopOutfits] = append([]shopItem{
		mk("out:basea", "Mono base azul (J1)", 0x2c3c78),
		mk("out:baser", "Mono base rojo (J2)", 0xa82c2c),
	}, shopCatalog[shopOutfits]...)
}

// ---- the wardrobe rows -----------------------------------------------------------------

type wardRowDef struct {
	slot, label string
}

var wardRows = []wardRowDef{
	{"base", "Personaje"}, {"outfit", "Ropa"}, {"tint", "Color del conjunto"}, {"hat", "Sombrero"}, {"mask", "Máscara"},
	{"cape", "Capa"}, {"shoes", "Zapatillas"}, {"glasses", "Gafas"}, {"beard", "Barba"}, {"hair", "Peinado"}, {"haircol", "Color de pelo"}, {"skincol", "Piel"},
}

// wardOption is one thing a row can show: the id that is saved and the name that is read.
type wardOption struct{ id, name string }

// wardOptions lists what a player can put in a row: only what is owned (the wardrobe never buys).
func (g *Game) wardOptions(slot string) []wardOption {
	switch slot {
	case "base":
		out := []wardOption{{"", "Protagonista"}}
		have := map[string]bool{}
		for _, it := range shopCatalog[shopChars] {
			if it.arg != "" && ownedID("skin:"+it.arg) && !have[it.arg] {
				have[it.arg] = true
				out = append(out, wardOption{it.arg, it.name})
			}
		}
		sort.SliceStable(out[1:], func(i, j int) bool { return out[1+i].name < out[1+j].name })
		return out
	case "tint":
		out := []wardOption{{"", "Automático"}}
		for _, t := range tints {
			out = append(out, wardOption{t.id, t.name})
		}
		return append(out, wardOption{"none", "Sin teñir"})
	}
	out := []wardOption{{"", "Ninguno"}}
	for _, tab := range []int{shopOutfits, shopAccess} {
		for _, it := range shopCatalog[tab] {
			if it.slot == slot && g.shopOwns(it) {
				out = append(out, wardOption{it.id, it.name})
			}
		}
	}
	return out
}

// wardCycle moves a row of a player to the next or previous option (what he owns) and returns the new name.
func (g *Game) wardCycle(player int, slot string, dir int) string {
	opts := g.wardOptions(slot)
	cur := g.equipOf(player, slot)
	i := 0
	for k, o := range opts {
		if o.id == cur {
			i = k
		}
	}
	i = (i + dir + len(opts)) % len(opts)
	g.setEquipOf(player, slot, opts[i].id)
	saveProgress(g.progress)
	g.applyShop()
	wardFlash = 0.45
	return opts[i].name
}

func (g *Game) wardName(player int, slot string) string {
	cur := g.equipOf(player, slot)
	for _, o := range g.wardOptions(slot) {
		if o.id == cur {
			return o.name
		}
	}
	return "Ninguno"
}

// ---- the screen (a list of the shop) ---------------------------------------------------

func (g *Game) wardPlayers() int {
	if g.wardTwo() {
		return 2
	}
	return 1
}

func (g *Game) updateWardrobe() {
	wardTime += 1.0 / 60
	if wardFlash > 0 {
		wardFlash -= 1.0 / 60
	}
	if wardP >= g.wardPlayers() {
		wardP = 0
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		for p := 0; p < g.wardPlayers(); p++ {
			if x >= 640+p*90 && x < 640+p*90+86 && y >= 126 && y <= 146 {
				wardP = p
			}
		}
		for i := range wardRows {
			if x >= 244 && x <= 600 && y >= 148+i*25 && y <= 148+i*25+23 {
				wardRow = i
			}
		}
	}
	if actDown.just() {
		wardRow = (wardRow + 1) % len(wardRows)
	}
	if actUp.just() {
		wardRow = (wardRow + len(wardRows) - 1) % len(wardRows)
	}
	slot := wardRows[wardRow].slot
	if actRight.just() || actConfirm.just() {
		g.wardCycle(wardP, slot, 1)
		g.snd.Play("alert")
	}
	if actLeft.just() {
		g.wardCycle(wardP, slot, -1)
		g.snd.Play("alert")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.setEquipOf(wardP, slot, "")
		saveProgress(g.progress)
		g.applyShop()
		wardFlash = 0.45
	}
	if (actInteract.just() || inpututil.IsKeyJustPressed(ebiten.KeyTab)) && g.wardPlayers() > 1 {
		wardP = 1 - wardP
		wardFlash = 0.3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.shopGo(g.shopPos() + 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		g.shopGo(g.shopPos() - 1)
	}
	if actBack.just() {
		g.mode = ModeMenu
	}
}

func (g *Game) drawWardrobe(dst *ebiten.Image) {
	// the player tabs
	for p := 0; p < g.wardPlayers(); p++ {
		col := playerColor(p)
		x := float64(640 + p*90)
		if p == wardP {
			fillRect(dst, x, 126, 86, 20, rgba(0x2a1a4a, 200))
			fillRect(dst, x, 144, 86, 2, col)
		}
		g.drawText(dst, fmt.Sprintf("JUGADOR %d", p+1), x+6, 130, col, 1)
	}
	if g.wardPlayers() == 1 {
		g.drawText(dst, "El J2 se viste cuando juegas de a 2", 640, 152, rgb(0x8a86a0), 1)
	}
	for i, r := range wardRows {
		y := 148 + float64(i)*25
		col := rgb(0x3fb59a)
		pre := "  "
		if i == wardRow {
			col, pre = rgb(0xff6a8a), "> "
			fillRect(dst, 244, y-3, 356, 23, rgba(0x2a1a4a, 160))
		}
		g.drawText(dst, pre+tr(r.label), 250, y, col, 1)
		val := g.wardName(wardP, r.slot)
		g.drawText(dst, "< "+val+" >", 596-g.textW("< "+val+" >", 1), y, rgb(0xf0e060), 1)
	}
	// the mannequin: it turns round and dances, with a flash when the look changes
	fillRect(dst, 640, 168, 270, 222, rgba(0x120c24, 220))
	col := playerColor(wardP)
	filledEllipse(dst, 775, 360, 70, 14, rgba(0x000000, 120))
	ring := rgba(colP1, 200)
	if wardP == 1 {
		ring = rgba(colP2, 200)
	}
	for a := 0.0; a < 2*math.Pi; a += 0.12 {
		fillRect(dst, math.Round(775+math.Cos(a)*52), math.Round(360+math.Sin(a)*11), 2, 1, ring)
	}
	cs := g.shopCS("wd:"+g.equipSigOf(wardP), func() *CharSpec { return g.playerSpecFor(wardP, "", "") })
	dirs := []Dir{DirDown, DirRight, DirUp, DirLeft}
	dir := dirs[int(wardTime/1.3)%len(dirs)]
	anim := AnimDance0 + (int(wardTime/5.2) % 3)
	if im, flip := cs.img(anim, dir, wardTime*6); im != nil {
		w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		if flip {
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w, 0)
		}
		op.GeoM.Scale(5, 5)
		op.GeoM.Translate(775-w*5/2, 362-h*5)
		dst.DrawImage(im, op)
	}
	if wardFlash > 0 {
		a := uint8(255 * wardFlash / 0.45)
		for i := 0; i < 10; i++ { // sparkles
			ang := float64(i)*0.63 + wardTime*3
			r := 20 + (0.45-wardFlash)*150
			sx, sy := 775+math.Cos(ang)*r, 280+math.Sin(ang)*r*0.8
			fillRect(dst, math.Round(sx), math.Round(sy), 3, 3, color.RGBA{col.R, col.G, col.B, a})
			fillRect(dst, math.Round(sx)-1, math.Round(sy)+1, 5, 1, color.RGBA{255, 255, 255, a})
			fillRect(dst, math.Round(sx)+1, math.Round(sy)-1, 1, 5, color.RGBA{255, 255, 255, a})
		}
	}
	g.drawText(dst, fmt.Sprintf("Jugador %d", wardP+1), 646, 172, col, 1.2)
	g.drawWrapped(dst, wardDesc(wardRows[wardRow].slot), 650, 396, 250, rgb(0xb8b0d0))
	g.drawText(dst, "↑/↓ fila   ←/→ cambiar   Supr quitar   "+actionHint(actInteract)+"/Tab otro jugador   RePág/AvPág lista   "+actionHint(actBack)+" volver", 244, 470, rgb(0x8a86a0), 1)
}

func wardDesc(slot string) string {
	switch slot {
	case "base":
		return "Con quién sales a la fiesta. Sólo los que has comprado en PERSONAJES."
	case "tint":
		return "Tiñe todo el conjunto. Automático: azul para el J1 y rojo para el J2."
	case "outfit":
		return "Aquí no se compra nada: sólo se eligen las prendas que ya son tuyas."
	}
	return "Elige entre lo que ya tienes. Lo que falte, se compra en la tienda."
}

// ---- the marker of a player --------------------------------------------------------------

// drawPlayerMarker paints the ring and the tag of a player under his feet and over his head, in
// his colour (blue the first, red the second), so that two players who dress the same are not mixed up.
func (g *Game) drawPlayerMarker(dst *ebiten.Image, pos Vec, player int) {
	cx, cy := g.wx(pos.X), g.wy(pos.Y)-1
	sc := g.charSc
	if sc <= 0 {
		sc = 1
	}
	col := playerColor(player)
	for a := 0.0; a < 2*math.Pi; a += 0.2 {
		fillRect(dst, math.Round(cx+math.Cos(a)*14*sc), math.Round(cy+math.Sin(a)*5*sc), 1, 1, col)
	}
	tag := fmt.Sprintf("P%d", player+1)
	w := math.Round(g.textW(tag, 1))
	y := cy - 46*sc
	fillRect(dst, math.Round(cx-w/2)-2, y-1, w+4, 10, color.RGBA{col.R / 3, col.G / 3, col.B / 3, 220})
	g.drawText(dst, tag, math.Round(cx-w/2), y, col, 1)
}
