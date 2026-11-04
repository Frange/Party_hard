package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// The shop: points earned in every party are banked in a wallet that survives
// between sessions and is spent on new main characters, clothes, perks and on
// unlocking animals, killers and tiers of police.
// ---------------------------------------------------------------------------

// syncShop copies the saved purchases into the global lookup.
func (g *Game) syncShop() {
	shopOwned = map[string]bool{"skin:": true, "police:0": true}
	for i, id := range g.progress.Owned {
		if n, ok := oldShopIDs[id]; ok {
			id = n
			g.progress.Owned[i] = n
		}
		shopOwned[id] = true
	}
	for sl, id := range g.progress.Equip {
		if n, ok := oldShopIDs[id]; ok {
			g.progress.Equip[sl] = n
		}
	}
	delete(g.progress.Equip, "skin") // characters are no longer skins of the player
	castSlugs = nil
	for id := range shopOwned {
		if len(id) > 5 && id[:5] == "skin:" {
			castSlugs = append(castSlugs, id[5:])
		}
	}
	sort.Strings(castSlugs)
}

// oldShopIDs maps purchases of the first version of the shop to the new ids.
var oldShopIDs = map[string]string{
	"perk:speed": "perk:speed1", "perk:stamina": "perk:stamina1", "perk:glove": "perk:glove1",
	"hat:cap": "hat:caproja", "hat:beanie": "hat:beanieazul", "out:suit": "out:suitnegro", "out:track": "out:trackroja",
	"out:hawaiian": "out:hawaiirosa", "out:hoodie": "out:hoodgris", "out:robe": "out:robemorada", "out:armor": "out:armorgris", "out:swim": "out:swimazul",
}

func (g *Game) shopEquip(slot string) string {
	if g.progress.Equip == nil {
		return ""
	}
	return g.progress.Equip[slot]
}

// bankPoints adds the points of the party just played to the wallet (once),
// with the bonus of the golden wallet if it was bought.
func (g *Game) bankPoints() {
	if g.banked || g.points <= 0 {
		return
	}
	g.banked = true
	g.progress.Wallet += int(float64(g.points) * g.perkMoney)
	g.stat("earned", g.points)  // total points ever earned (trophies "Primeros ahorros", "Multimillonario")
	g.statMax("best", g.points) // best single party (trophies "Buena noche", "Noche de récord")
	saveProgress(g.progress)
}

// tier returns the highest owned level (1..3) of a perk family, 0 if none.
func tier(prefix string) int {
	t := 0
	for i := 1; i <= 3; i++ {
		if ownedID(fmt.Sprintf("%s%d", prefix, i)) {
			t = i
		}
	}
	return t
}

// applyShop sets the perks and dresses the player and the police with what
// was bought and equipped.
func (g *Game) applyShop() {
	g.perkSpeed = []float64{1, 1.08, 1.16, 1.25}[tier("perk:speed")]
	g.perkStamina = []float64{6, 9, 12, 16}[tier("perk:stamina")]
	g.perkCool = []float64{1, 0.8, 0.65, 0.5}[tier("perk:glove")]
	g.perkMoney = []float64{1, 1.1, 1.25, 1.5}[tier("perk:money")]
	if g.store == nil || len(g.store.Chars["main_character"]) == 0 {
		g.as.playerCS = newCharSet(g.playerSpecWith("", "", ""))
	}
	if g.store == nil || len(g.store.Chars["police"]) == 0 {
		g.dressTiers()
		fx := wearFx[g.shopEquip("cop")]
		base := copSpec()
		a := copSpec()
		a.Name, a.Female, a.Hair2, a.Hair, a.Body = "cop_f", true, HairPony, hx(0x6a3a1a), BodySlim
		b := copSpec()
		b.Name, b.Body, b.Beard, b.Stature, b.Skin = "cop_big", BodyFat, BeardMustache, -1, hx(0x9a6a48)
		if fx != nil {
			fx(&base)
			fx(&a)
			fx(&b)
		}
		g.as.copCS = newCharSet(&base)
		g.as.copAlt = []*CharSet{newCharSet(&a), newCharSet(&b)}
	}
}

// dressTiers puts the equipped uniforms on the FBI, army, SWAT and Rangers.
func (g *Game) dressTiers() {
	if g.tierBase == nil {
		g.tierBase = map[string][]CharSpec{}
		snap := func(k string, l []*CharSet) {
			for _, cs := range l {
				g.tierBase[k] = append(g.tierBase[k], *cs.Spec)
			}
		}
		snap("fbi", g.as.fbiCS)
		snap("army", g.as.militaryCS)
		snap("rangers", g.as.rangerCS)
		if g.as.swatCS != nil {
			g.tierBase["swat"] = []CharSpec{*g.as.swatCS.Spec}
		}
	}
	build := func(k string) []*CharSet {
		var out []*CharSet
		for _, b := range g.tierBase[k] {
			sp := b
			if fx := wearFx[g.shopEquip(k)]; fx != nil {
				fx(&sp)
			}
			out = append(out, newCharSet(&sp))
		}
		return out
	}
	if l := build("fbi"); len(l) > 0 {
		g.as.fbiCS = l
	}
	if l := build("army"); len(l) > 0 {
		g.as.militaryCS = l
	}
	if l := build("rangers"); len(l) > 0 {
		g.as.rangerCS = l
	}
	if l := build("swat"); len(l) > 0 {
		g.as.swatCS = l[0]
	}
}

// playerSpecWith: the player spec dressed with the equipped clothes; a preview
// item (id) replaces what is worn in its slot.
func (g *Game) playerSpecWith(skin, previewSlot, previewID string) *CharSpec {
	spec := playerSpec()
	if skin != "" {
		if f := presetByName(skin); f != nil {
			spec = f(rand.New(rand.NewSource(5)))
		}
	}
	for _, slot := range wearSlots {
		id := g.shopEquip(slot)
		if slot == previewSlot {
			id = previewID
		}
		if fx := wearFx[id]; fx != nil {
			fx(&spec)
		}
	}
	return &spec
}

func (g *Game) shopItems() []shopItem { return shopCatalog[g.shopTab] }

// shopGo moves to another list (a position in shopFlat).
func (g *Game) shopGo(i int) {
	n := len(shopFlat)
	g.shopTab, g.shopSel, g.shopScroll = shopFlat[(i%n+n)%n], 0, 0
}

func (g *Game) shopPos() int {
	for i, t := range shopFlat {
		if t == g.shopTab {
			return i
		}
	}
	return 0
}

func (g *Game) shopOwns(it shopItem) bool {
	if it.price == 0 {
		return true
	}
	return ownedID(it.id)
}

func (g *Game) shopActivate() {
	items := g.shopItems()
	if g.shopSel < 0 || g.shopSel >= len(items) {
		return
	}
	it := items[g.shopSel]
	if !g.shopOwns(it) {
		if it.trophy != "" {
			g.shopMsg = "Se consigue con un trofeo (mira TROFEOS)"
			g.snd.Play("alert")
			return
		}
		if g.progress.Wallet < it.price {
			g.shopMsg = "No tienes puntos suficientes"
			g.snd.Play("alert")
			return
		}
		g.progress.Wallet -= it.price
		g.progress.Owned = append(g.progress.Owned, it.id)
		g.syncShop()
		g.stat("bought", 1)
		g.shopMsg = "¡Comprado: " + it.name + "!"
		g.snd.Play("win")
		if it.slot != "" {
			g.setEquip(it.slot, it.id)
		}
		saveProgress(g.progress)
		g.applyShop()
		return
	}
	if it.slot == "" && g.shopTab == shopChars {
		g.shopMsg = "Ya lo tienes: sale más a menudo en las fiestas"
		return
	}
	if it.slot != "" {
		if g.shopEquip(it.slot) == trimPrefixID(it.id, "skin:") || g.shopEquip(it.slot) == it.id {
			g.setEquip(it.slot, "") // take it off
			g.shopMsg = "Te lo quitas"
		} else {
			g.setEquip(it.slot, it.id)
			g.shopMsg = "Te lo pones"
		}
		saveProgress(g.progress)
		g.applyShop()
	}
}

func (g *Game) setEquip(slot, id string) {
	if g.progress.Equip == nil {
		g.progress.Equip = map[string]string{}
	}
	if slot == "skin" {
		id = trimPrefixID(id, "skin:")
	}
	if id == "" {
		delete(g.progress.Equip, slot)
		return
	}
	g.progress.Equip[slot] = id
}

func trimPrefixID(s, p string) string {
	if len(s) >= len(p) && s[:len(p)] == p {
		return s[len(p):]
	}
	return s
}

const shopRows = 9

func (g *Game) updateShop() {
	g.time += 1.0 / 60
	items := g.shopItems()
	click := g.mouseMenu(func(mx, my float64) int {
		for i := 0; i < shopRows; i++ {
			if mx >= 244 && mx <= 600 && my >= 142+float64(i)*28 && my <= 142+float64(i)*28+26 {
				return g.shopScroll + i
			}
		}
		return -1
	}, &g.shopSel)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if y >= 78 && y <= 98 { // top level: the groups
			for gi := range shopGroups {
				if x >= 244+gi*104 && x < 244+(gi+1)*104 {
					g.shopTab, g.shopSel, g.shopScroll = shopGroups[gi].tabs[0], 0, 0
				}
			}
		}
		if y >= 100 && y <= 120 { // second level: the lists of the group
			gr := shopGroups[shopGroupOf(g.shopTab)]
			for i, t := range gr.tabs {
				if x >= 244+i*104 && x < 244+(i+1)*104 {
					g.shopTab, g.shopSel, g.shopScroll = t, 0, 0
				}
			}
		}
	}
	if _, wy := ebiten.Wheel(); wy != 0 {
		g.shopSel = clampI(g.shopSel-int(wy)*3, 0, len(items)-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.shopSel = clampI(g.shopSel+shopRows, 0, len(items)-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		g.shopSel = clampI(g.shopSel-shopRows, 0, len(items)-1)
	}
	if actDown.just() {
		g.shopSel = (g.shopSel + 1) % len(items)
	}
	if actUp.just() {
		g.shopSel = (g.shopSel + len(items) - 1) % len(items)
	}
	if actRight.just() {
		g.shopGo(g.shopPos() + 1)
	}
	if actLeft.just() {
		g.shopGo(g.shopPos() - 1)
	}
	items = g.shopItems()
	if g.shopSel >= len(items) {
		g.shopSel = len(items) - 1
	}
	if g.shopSel < g.shopScroll {
		g.shopScroll = g.shopSel
	}
	if g.shopSel >= g.shopScroll+shopRows {
		g.shopScroll = g.shopSel - shopRows + 1
	}
	if actConfirm.just() || click {
		g.shopActivate()
	}
	if actBack.just() {
		g.mode = ModeMenu
	}
}

// shopPreview returns the sprite for the selected item (nil: none).
func (g *Game) shopPreview(it shopItem) *ebiten.Image {
	sig := g.equipSig()
	switch g.shopTab {
	case shopChars:
		return g.shopCS("skin:"+it.arg, func() *CharSpec {
			if f := presetByName(it.arg); f != nil {
				sp := f(rand.New(rand.NewSource(5)))
				return &sp
			}
			sp := playerSpec()
			return &sp
		}).frameImg(AnimIdle, vFront, 0)
	case shopOutfits, shopAccess:
		return g.shopCS("cl:"+it.id+sig, func() *CharSpec { return g.playerSpecWith("", it.slot, it.id) }).frameImg(AnimIdle, vFront, 0)
	case shopKillers:
		return g.shopCS("k:"+it.arg, func() *CharSpec {
			for _, e := range killerPresets {
				if e.Name == it.arg {
					s := e.F(rand.New(rand.NewSource(3)))
					return &s
				}
			}
			s := playerSpec()
			return &s
		}).frameImg(AnimIdle, vFront, 0)
	case shopPolice:
		var cs *CharSet
		switch it.id {
		case "police:0":
			cs = g.as.copCS
		case "police:1":
			if len(g.as.fbiCS) > 0 {
				cs = g.as.fbiCS[0]
			}
		case "police:2":
			if len(g.as.militaryCS) > 0 {
				cs = g.as.militaryCS[0]
			}
		case "police:3":
			cs = g.as.swatCS
		case "police:4":
			if len(g.as.rangerCS) > 0 {
				cs = g.as.rangerCS[0]
			}
		}
		if cs != nil {
			return cs.frameImg(AnimIdle, vFront, 0)
		}
	case shopUniforms:
		kind := map[string]string{"cop": "cop", "fbi": "fbi", "army": "army", "swat": "swat", "rangers": "rangers"}[it.slot]
		return g.shopCS("uni:"+it.id, func() *CharSpec {
			sp := copSpec()
			if kind != "cop" {
				g.dressTiers()
				if b := g.tierBase[kind]; len(b) > 0 {
					sp = b[0]
				}
			}
			if fx := wearFx[it.id]; fx != nil {
				fx(&sp)
			}
			return &sp
		}).frameImg(AnimIdle, vFront, 0)
	case shopWeapons:
		return g.shopCS("kw:"+it.id, func() *CharSpec {
			for _, e := range killerPresets {
				if e.Name == "michael" {
					sp := e.F(rand.New(rand.NewSource(3)))
					sp.Weapon = weaponHold[it.id]
					return &sp
				}
			}
			sp := playerSpec()
			return &sp
		}).frameImg(AnimIdle, vFront, 0)
	case shopAnimals:
		k := animalKindIndex(it.arg)
		if im := g.as.animals[animalKey(k, 0, fStand)]; im != nil {
			return im
		}
	}
	return nil
}

// equipSig identifies what is worn, so previews are rebuilt when it changes.
func (g *Game) equipSig() string {
	s := ""
	for _, sl := range wearSlots {
		s += g.shopEquip(sl) + "|"
	}
	return s + g.shopEquip("skin")
}

func (g *Game) shopCS(key string, mk func() *CharSpec) *CharSet {
	if g.shopCache == nil {
		g.shopCache = map[string]*CharSet{}
	}
	if cs, ok := g.shopCache[key]; ok {
		return cs
	}
	cs := newCharSet(mk())
	g.shopCache[key] = cs
	return cs
}

func (g *Game) drawShop(dst *ebiten.Image) {
	if g.iconMode {
		dst.Fill(rgb(0x0a0912))
		g.drawLogo(dst, 160, 60)
		return
	}
	g.drawBedroom(dst)
	g.drawLogo(dst, 160, 30)
	fillRect(dst, 230, 56, 700, 440, rgba(0x05070e, 228))
	fillRect(dst, 230, 56, 700, 1, rgba(0x6a4ab0, 160))
	fillRect(dst, 230, 495, 700, 1, rgba(0x6a4ab0, 160))
	g.drawText(dst, "TIENDA", 244, 62, rgb(0xff6a8a), 1.6)
	g.drawText(dst, fmt.Sprintf("PUNTOS: %d", g.progress.Wallet), 700, 62, rgb(0xf0e060), 1.6)
	curG := shopGroupOf(g.shopTab)
	for gi, gr := range shopGroups {
		col := rgb(0x3fb59a)
		if gi == curG {
			col = rgb(0xff6a8a)
			fillRect(dst, float64(244+gi*104), 98, 100, 2, col)
		}
		g.drawText(dst, gr.name, float64(246+gi*104), 82, col, 1.2)
	}
	if gr := shopGroups[curG]; len(gr.tabs) > 1 {
		for i, t := range gr.tabs {
			col := rgb(0x8a86a0)
			if t == g.shopTab {
				col = rgb(0xf0e060)
				fillRect(dst, float64(244+i*104), 118, 100, 1, col)
			}
			g.drawText(dst, shopTabNames[t], float64(246+i*104), 104, col, 0.95)
		}
	} else {
		g.drawText(dst, shopTabNames[g.shopTab], 246, 104, rgb(0xf0e060), 0.95)
	}
	items := g.shopItems()
	for i := 0; i < shopRows; i++ {
		idx := g.shopScroll + i
		if idx >= len(items) {
			break
		}
		it := items[idx]
		y := 146 + float64(i)*28
		col := rgb(0x3fb59a)
		pre := "  "
		if idx == g.shopSel {
			col, pre = rgb(0xff6a8a), "> "
			fillRect(dst, 244, y-3, 356, 25, rgba(0x2a1a4a, 160))
		}
		g.drawText(dst, pre+it.name, 250, y, col, 1.3)
		tag, tcol := fmt.Sprintf("%d", it.price), rgb(0xf0e060)
		if it.trophy != "" {
			tag, tcol = "TROFEO", rgb(0x8a6aa8)
		}
		switch {
		case it.slot != "" && g.shopOwns(it) && (g.shopEquip(it.slot) == trimPrefixID(it.id, "skin:") || g.shopEquip(it.slot) == it.id):
			tag, tcol = "PUESTO", rgb(0x70e090)
		case g.shopOwns(it):
			tag, tcol = "TUYO", rgb(0x70e090)
		case g.progress.Wallet < it.price:
			tcol = rgb(0x8a6a6a)
		}
		g.drawText(dst, tag, 600-g.textW(tag, 1.3), y, tcol, 1.3)
	}
	if len(items) > shopRows {
		g.drawText(dst, fmt.Sprintf("%d/%d", g.shopSel+1, len(items)), 244, 408, rgb(0x6a6680), 1)
	}
	if g.shopSel >= 0 && g.shopSel < len(items) {
		it := items[g.shopSel]
		fillRect(dst, 640, 128, 270, 262, rgba(0x120c24, 220))
		if im := g.shopPreview(it); im != nil {
			w, h := float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
			sc := 4.0
			if g.shopTab == shopAnimals {
				sc = 2.5
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(sc, sc)
			op.GeoM.Translate(775-w*sc/2, 262-h*sc/2)
			dst.DrawImage(im, op)
		}
		if g.shopTab == shopPerks {
			filledEllipse(dst, 775, 252, 46, 46, rgb(0xe8c040))
			filledEllipse(dst, 775, 252, 38, 38, rgb(0xb88a20))
			filledEllipse(dst, 775, 252, 32, 32, rgb(0xe8c040))
			n := []string{"", "I", "II", "III"}[1]
			if l := len(it.id); l > 0 {
				n = []string{"", "I", "II", "III"}[clampI(int(it.id[l-1]-'0'), 1, 3)]
			}
			g.drawTextC(dst, n, 775, 238, rgb(0x6a4a10), 3)
		}
		g.drawTextC(dst, it.name, 775, 134, rgb(0xf4f0ff), 1.2)
		g.drawWrapped(dst, it.desc, 650, 352, 250, rgb(0xb8b0d0))
	}
	if g.shopMsg != "" {
		g.drawText(dst, g.shopMsg, 244, 428, rgb(0xf0e060), 1.3)
	}
	g.drawText(dst, "←/→ lista   ↑/↓ o rueda elegir   "+actionHint(actConfirm)+" comprar / ponerse   "+actionHint(actBack)+" volver", 244, 470, rgb(0x8a86a0), 1)
}

// drawWrapped draws text broken into lines that fit width w.
func (g *Game) drawWrapped(dst *ebiten.Image, s string, x, y, w float64, col color.RGBA) {
	line, words := "", []string{}
	cur := ""
	for _, r := range s + " " {
		if r == ' ' {
			words = append(words, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	for _, wd := range words {
		t := wd
		if line != "" {
			t = line + " " + wd
		}
		if g.textW(t, 1) > w && line != "" {
			g.drawText(dst, line, x, y, col, 1)
			y += 14
			line = wd
		} else {
			line = t
		}
	}
	if line != "" {
		g.drawText(dst, line, x, y, col, 1)
	}
}

// animalAllowed: free animals are always there; the rest must be bought.
func animalAllowed(kind int) bool {
	if kind < 0 || kind >= len(animalKinds) {
		return true
	}
	switch n := animalKinds[kind]; n {
	case "elephant", "giraffe", "kangaroo", "deer", "rabbit", "pig", "duck", "budgie":
		return ownedID("animal:" + n)
	case "quijote", "sancho":
		return quijoteUnlocked()
	}
	return true
}

// killerAllowed: Michael, Ghostface and Jason are free, the rest are bought.
func killerAllowed(name string) bool {
	switch name {
	case "michael", "ghostface", "jason":
		return true
	}
	return ownedID("killer:" + name)
}

// quijoteUnlocked: buying either of the two playable characters brings the
// pair of riders to the parties as well.
func quijoteUnlocked() bool { return ownedID("skin:don_quijote") || ownedID("skin:sancho_panza") }

// castSlugs: the special characters bought so far; they are drawn more often
// when the guests of a party are chosen.
var castSlugs []string
