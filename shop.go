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
	charSyncOwned(g.progress.Album)  // the characters unlocked by kills (goalchars.go) are "owned" too
	copSetSync(g.progress.Escapes)   // the police uniforms the escapes opened (copsets.go)
	// castSlugs: the characters bought in the old shop and the trophy rewards; they come more often. The ones
	// unlocked by kills do NOT (they would crowd out the faces still to be found).
	castSlugs = nil
	seen := map[string]bool{}
	for _, id := range g.progress.Owned {
		if len(id) > 5 && id[:5] == "skin:" && !seen[id] {
			seen[id] = true
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
	// the protagonist wears what the player 1 put on in the wardrobe (unless the user drew him by hand in the editor)
	if !g.editedMainCharacter() {
		g.as.playerCS = g.playerCS(0)
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
	return g.playerSpecFor(wardP, previewSlot, previewID)
}

// shopItems: the list of the current tab, the ones not bought yet first and the bought ones at the end (each
// group keeps the catalog order, which is by price). It is recomputed on every call, so it follows tab changes and
// purchases at once.
func (g *Game) shopItems() []shopItem { return shopUnboughtFirst(shopCatalog[g.shopTab]) }

// shopUnboughtFirst: stable partition of a list, not owned first. The input is not modified.
func shopUnboughtFirst(list []shopItem) []shopItem {
	out := make([]shopItem, 0, len(list))
	for _, it := range list {
		if !(it.price == 0 || ownedID(it.id)) {
			out = append(out, it)
		}
	}
	for _, it := range list {
		if it.price == 0 || ownedID(it.id) {
			out = append(out, it)
		}
	}
	return out
}

// shopGo moves to another list (a position in shopFlat).
func (g *Game) shopGo(i int) {
	n := len(shopFlat)
	g.shopTab, g.shopSel, g.shopScroll = shopFlat[(i%n+n)%n], 0, 0
	g.shopBuyFocus = false
}

// BUY ALL button of the lists (under the message, over the key hints; the BACK button is at the bottom left).
const (
	shopBuyX, shopBuyY = 244.0, 456.0
	shopBuyW, shopBuyH = 176.0, 20.0
)

// the Yes / No buttons of the popup
const (
	shopYesX, shopNoX = 330.0, 490.0
	shopPopBtnY       = 296.0
	shopPopBtnW       = 140.0
	shopPopBtnH       = 28.0
)

// The groups of the top row are shopGroupW wide every 104 px; VESTUARIO (always the last group) goes shopWardGap px
// further right, framed and in orange, so it is not mistaken for one more list of things to buy.
const (
	shopGroupW  = 100.0
	shopWardGap = 56.0
)

// The PUNTOS line: right edge and top, above the group row (frames at y 77+), so a long figure never touches VESTUARIO's frame.
const (
	shopPointsRight = 916.0
	shopPointsY     = 50.0 // same row as the TIENDA title: 10 px under the panel top
	shopPointsH     = 14.0 // text height at scale 1.6, rounded up
	shopGroupFrameY = 77.0
	shopTitleY      = 50.0
)

// the shop panel: it ends 12 px above the BACK button (y=500) and starts 12+ px under the top edge of the screen
const (
	shopPanX, shopPanY = 230.0, 40.0
	shopPanW, shopPanH = 700.0, 448.0
	shopTabTextY       = 109.0 // the second row of tabs: 10+ px under the frame of VESTUARIO (it ends at y=99)
	shopTabLineY       = 123.0
	shopMsgY           = 430.0 // the message of a list (the counter above it, BUY ALL under it)
	shopMsgWardY       = 456.0 // in the wardrobe there is no BUY ALL button: the bottom line
	shopRowRight       = 594.0 // right edge of the prices: 6 px of air inside the row (it ends at x=600)
)

func shopGroupX(gi int) float64 {
	x := 244 + float64(gi)*104
	if gi >= 0 && gi < len(shopGroups) && shopGroups[gi].tabs[0] == shopWardrobe {
		x += shopWardGap
	}
	return x
}

// shopGroupAt: the group under the x of a click on the top row (-1: none; the gap before VESTUARIO is nothing).
func shopGroupAt(x float64) int {
	for gi := range shopGroups {
		if gx := shopGroupX(gi); x >= gx-4 && x < gx+shopGroupW {
			return gi
		}
	}
	return -1
}

func inRect(x, y, rx, ry, rw, rh float64) bool { return x >= rx && x < rx+rw && y >= ry && y < ry+rh }

// updateShopPopup: the Yes / No question of BUY ALL (the shop below does not move).
func (g *Game) updateShopPopup() {
	if actLeft.just() || actRight.just() || actUp.just() || actDown.just() {
		g.shopBuyYes = !g.shopBuyYes
	}
	yes, no := false, false
	x, y := ebiten.CursorPosition()
	fx, fy := float64(x), float64(y)
	moved := x != g.lastMX || y != g.lastMY
	g.lastMX, g.lastMY = x, y
	if moved {
		if inRect(fx, fy, shopYesX, shopPopBtnY, shopPopBtnW, shopPopBtnH) {
			g.shopBuyYes = true
		} else if inRect(fx, fy, shopNoX, shopPopBtnY, shopPopBtnW, shopPopBtnH) {
			g.shopBuyYes = false
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		yes = inRect(fx, fy, shopYesX, shopPopBtnY, shopPopBtnW, shopPopBtnH)
		no = inRect(fx, fy, shopNoX, shopPopBtnY, shopPopBtnW, shopPopBtnH)
	}
	if actConfirm.just() {
		yes, no = g.shopBuyYes, !g.shopBuyYes
	}
	if actBack.just() {
		no = true
	}
	switch {
	case yes:
		g.shopBuyAllDo()
	case no:
		g.shopBuyOpen = false
	}
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
	g.shopMsg = ""
	if !g.shopOwns(it) {
		if it.trophy != "" {
			if g.shopTab == shopPolice {
				g.shopMsg = fmt.Sprintf("Desbloquea al llegar a %d asesinatos", policeNeed(it.id))
			} else {
				g.shopMsg = "Se consigue con un trofeo (mira OBJETIVOS)"
			}
			g.snd.Play("alert")
			return
		}
		if g.progress.Wallet < it.price {
			g.shopMsg = "No tienes puntos suficientes"
			g.snd.Play("alert")
			return
		}
		g.shopMsg = "¡Comprado: " + it.name + "!" + g.shopBuyOne(it)
		g.snd.Play("win")
		saveProgress(g.progress)
		g.applyShop()
		return
	}
	if it.slot == "" && g.shopTab == shopChars {
		g.shopMsg = "Ya lo tienes: sale más a menudo en las fiestas"
		return
	}
	if it.slot != "" {
		if g.equipOf(wardP, it.slot) == trimPrefixID(it.id, "skin:") || g.equipOf(wardP, it.slot) == it.id {
			g.setEquipOf(wardP, it.slot, "") // take it off
			g.shopMsg = "Te lo quitas"
		} else {
			g.setEquipOf(wardP, it.slot, it.id)
			g.shopMsg = "Te lo pones"
		}
		if g.wardTwo() && isWearSlot(it.slot) {
			g.shopMsg += fmt.Sprintf(" (J%d)", wardP+1)
		}
		saveProgress(g.progress)
		g.applyShop()
	}
}

// shopBuyOne: pays and adds one item (the caller checked that it is for sale and affordable). Buying is not
// dressing: it only puts the new thing on when that slot of the player was empty. It returns the note for the
// message; saving and applyShop are up to the caller.
func (g *Game) shopBuyOne(it shopItem) string {
	g.progress.Wallet -= it.price
	g.progress.Owned = append(g.progress.Owned, it.id)
	g.syncShop()
	g.stat("bought", 1)
	if shopAllBought() {
		g.stat("shopall", 1) // the secret trophy of owning everything (trophies3.go)
	}
	note := ""
	if it.slot != "" && isWearSlot(it.slot) {
		if g.equipOf(wardP, it.slot) == "" {
			g.setEquipOf(wardP, it.slot, it.id)
			note = fmt.Sprintf(" Puesto al J%d.", wardP+1)
		} else {
			note = " Póntelo en VESTUARIO."
		}
	} else if it.slot != "" {
		g.setEquip(it.slot, it.id)
	}
	return note
}

// shopBuyAllPlan: what BUY ALL would take in the current list with the points at hand: the items for sale (no
// trophy lock, not free, not owned) from the cheapest up, as many as the wallet pays. Fewer points than the
// cheapest one: nothing. The cheapest-first order is what reaches the biggest number of items.
func (g *Game) shopBuyAllPlan() (items []shopItem, cost int) {
	var cand []shopItem
	for _, it := range shopCatalog[g.shopTab] {
		if shopBuyable(it) && !g.shopOwns(it) {
			cand = append(cand, it)
		}
	}
	sort.SliceStable(cand, func(i, j int) bool { return cand[i].price < cand[j].price })
	for _, it := range cand {
		if cost+it.price > g.progress.Wallet {
			break
		}
		cost += it.price
		items = append(items, it)
	}
	return items, cost
}

// shopBuyAllAsk: the BUY ALL button was pressed: opens the Yes/No popup, or says there is nothing to buy.
func (g *Game) shopBuyAllAsk() {
	items, cost := g.shopBuyAllPlan()
	if len(items) == 0 {
		any := false
		for _, it := range shopCatalog[g.shopTab] {
			if shopBuyable(it) && !g.shopOwns(it) {
				any = true
			}
		}
		if any {
			g.shopMsg = "No tienes puntos para comprar nada de esta lista"
		} else {
			g.shopMsg = "No hay nada que comprar en esta lista"
		}
		g.snd.Play("alert")
		return
	}
	g.shopBuyOpen, g.shopBuyYes, g.shopBuyN, g.shopBuyCost = true, true, len(items), cost
}

// shopBuyAllDo: buys the plan (recomputed now, never more than the wallet pays). Returns how many were bought.
func (g *Game) shopBuyAllDo() int {
	g.shopBuyOpen = false
	items, cost := g.shopBuyAllPlan()
	if len(items) == 0 {
		return 0
	}
	for _, it := range items {
		g.shopBuyOne(it)
	}
	g.shopMsg = fmt.Sprintf("¡Comprados %d artículos por %d puntos!", len(items), cost)
	g.snd.Play("win")
	saveProgress(g.progress)
	g.applyShop()
	return len(items)
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
	if g.shopBuyOpen {
		g.updateShopPopup()
		return
	}
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
			if gi := shopGroupAt(float64(x)); gi >= 0 {
				g.shopTab, g.shopSel, g.shopScroll, g.shopBuyFocus = shopGroups[gi].tabs[0], 0, 0, false
			}
		}
		if y >= 100 && y <= 120 { // second level: the lists of the group
			gr := shopGroups[shopGroupOf(g.shopTab)]
			for i, t := range gr.tabs {
				if x >= 244+i*104 && x < 244+(i+1)*104 {
					g.shopTab, g.shopSel, g.shopScroll, g.shopBuyFocus = t, 0, 0, false
				}
			}
		}
	}
	if g.shopTab == shopWardrobe { // the wardrobe is not a list: it has its own keys
		g.updateWardrobe()
		return
	}
	if g.wardTwo() && inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		wardP = 1 - wardP // who gets what is bought and worn from the lists
	}
	if _, wy := ebiten.Wheel(); wy != 0 {
		g.shopSel = clampI(g.shopSel-int(wy)*3, 0, len(items)-1)
	}
	g.shopSel, g.shopScroll = edgeListScroll("shop", g.shopSel, g.shopScroll, shopRows, 1, len(items), [4]float64{244, 142, 356, shopRows * 28}, 6) // listscroll.go
	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		g.shopSel = clampI(g.shopSel+shopRows, 0, len(items)-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		g.shopSel = clampI(g.shopSel-shopRows, 0, len(items)-1)
	}
	// the BUY ALL button is one more stop after the last item (down) and before the first (up)
	switch {
	case g.shopBuyFocus && actDown.just():
		g.shopBuyFocus, g.shopSel = false, 0
	case g.shopBuyFocus && actUp.just():
		g.shopBuyFocus, g.shopSel = false, len(items)-1
	case actDown.just():
		if g.shopSel+1 >= len(items) {
			g.shopBuyFocus = true
		} else {
			g.shopSel++
		}
	case actUp.just():
		if g.shopSel <= 0 {
			g.shopBuyFocus = true
		} else {
			g.shopSel--
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if inRect(float64(x), float64(y), shopBuyX, shopBuyY, shopBuyW, shopBuyH) {
			g.shopBuyAllAsk()
			return
		}
	}
	if g.shopBuyFocus && actConfirm.just() && !backFocus {
		g.shopBuyAllAsk()
		return
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
	if (actConfirm.just() || click) && !g.shopBuyFocus {
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
func (g *Game) equipSig() string { return g.equipSigOf(wardP) }

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
	if g.shopBuyOpen { // the question hides the shop: opaque, so nothing shows through
		g.drawShopPopup(dst)
		return
	}
	g.drawLogo(dst, logoSideCX, 30)
	fillRect(dst, shopPanX, shopPanY, shopPanW, shopPanH, rgba(0x05070e, 228))
	fillRect(dst, shopPanX, shopPanY, shopPanW, 1, rgba(0x6a4ab0, 160))
	fillRect(dst, shopPanX, shopPanY+shopPanH-1, shopPanW, 1, rgba(0x6a4ab0, 160))
	g.drawText(dst, "TIENDA", 244, shopTitleY, rgb(0xff6a8a), 1.6)
	pts := fmt.Sprintf("PUNTOS: %d", g.progress.Wallet)
	g.drawText(dst, pts, shopPointsRight-g.textW(pts, 1.6), shopPointsY, rgb(0xf0e060), 1.6) // right-aligned: long figures grow to the left, above the group row
	curG := shopGroupOf(g.shopTab)
	for gi, gr := range shopGroups {
		gx := shopGroupX(gi)
		col := rgb(0x3fb59a)
		if gr.tabs[0] == shopWardrobe { // VESTUARIO: apart, framed and in its own colour
			edge := rgba(0xf0a040, 150)
			col = rgb(0xf0a040)
			if gi == curG {
				edge, col = rgb(0xffc060), rgb(0xffe0a0)
			}
			shopFrame(dst, gx-4, 77, shopGroupW+2, 22, rgba(0x2a1a08, 230), edge)
		}
		if gi == curG {
			if gr.tabs[0] != shopWardrobe {
				col = rgb(0xff6a8a)
			}
			fillRect(dst, gx, 98, shopGroupW, 2, col)
		}
		g.drawText(dst, gr.name, gx+2, 82, col, 1.2)
	}
	if g.shopTab == shopWardrobe { // the three pages of the wardrobe (wardrobe_kinds.go)
		g.drawWardKinds(dst)
	} else if gr := shopGroups[curG]; len(gr.tabs) > 1 {
		for i, t := range gr.tabs {
			col := rgb(0x8a86a0)
			if t == g.shopTab {
				col = rgb(0xf0e060)
				fillRect(dst, float64(244+i*104), shopTabLineY, 100, 1, col)
			}
			g.drawText(dst, shopTabNames[t], float64(246+i*104), shopTabTextY, col, 0.95)
		}
	} else {
		g.drawText(dst, shopTabNames[g.shopTab], 246, shopTabTextY, rgb(0xf0e060), 0.95)
	}
	if g.shopTab == shopWardrobe {
		g.drawWardrobe(dst)
		if g.shopMsg != "" {
			g.drawText(dst, g.shopMsg, 244, shopMsgWardY, rgb(0xf0e060), 1.3)
		}
		return
	}
	items := g.shopItems()
	if g.wardTwo() && (g.shopTab == shopOutfits || g.shopTab == shopAccess) {
		g.drawText(dst, fmt.Sprintf("Se pone el J%d (Tab: cambiar)", wardP+1), 640, 410, playerColor(wardP), 1) // under the card (it ends at y=400)
	}
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
			if g.shopTab == shopPolice {
				tag = fmt.Sprintf("%d/%d", careerKills(&g.progress), policeNeed(it.id))
			}
		}
		switch {
		case it.slot != "" && g.shopOwns(it) && (g.equipOf(wardP, it.slot) == trimPrefixID(it.id, "skin:") || g.equipOf(wardP, it.slot) == it.id):
			tag, tcol = "PUESTO", rgb(0x70e090)
		case g.shopOwns(it):
			tag, tcol = "TUYO", rgb(0x70e090)
		case g.progress.Wallet < it.price:
			tcol = rgb(0x8a6a6a)
		}
		g.drawText(dst, tag, shopRowRight-g.textW(tag, 1.3), y, tcol, 1.3)
	}
	if len(items) > shopRows {
		g.drawText(dst, fmt.Sprintf("%d/%d", g.shopSel+1, len(items)), 244, 408, rgb(0x6a6680), 1)
	}
	if g.shopSel >= 0 && g.shopSel < len(items) {
		it := items[g.shopSel]
		fillRect(dst, 640, 128, 270, 272, rgba(0x120c24, 220)) // tall enough for a description of three lines
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
		g.drawWrapped(dst, shopDescHint(it.desc), 650, 352, 250, rgb(0xb8b0d0))
		if g.shopTab == shopPolice && it.trophy != "" && !g.shopOwns(it) {
			need := policeNeed(it.id)
			have := minI(careerKills(&g.progress), need)
			g.drawText(dst, fmt.Sprintf("Desbloquea al llegar a %d asesinatos", need), 650, 296, rgb(0xf0e060), 1)
			fillRect(dst, 650, 312, 250, 8, rgb(0x2a2240))
			fillRect(dst, 650, 312, float64(250*have/maxI(1, need)), 8, rgb(0xd84a5a))
			g.drawText(dst, fmt.Sprintf("Asesinatos: %d/%d", have, need), 650, 324, rgb(0xb8b0d0), 1)
		}
	}
	if g.shopMsg != "" {
		g.drawText(dst, g.shopMsg, 244, shopMsgY, rgb(0xf0e060), 1.3)
	}
	mx, my := ebiten.CursorPosition()
	hot := g.shopBuyFocus || inRect(float64(mx), float64(my), shopBuyX, shopBuyY, shopBuyW, shopBuyH)
	bg, edge, tc := rgba(0x120c24, 230), rgba(0x6a4ab0, 220), rgb(0xf0e060)
	if hot {
		bg, edge, tc = rgba(0x2a1a4a, 250), rgb(0xff6a8a), rgb(0xffffff)
	}
	shopFrame(dst, shopBuyX, shopBuyY, shopBuyW, shopBuyH, bg, edge)
	g.drawTextC(dst, "COMPRAR TODO", shopBuyX+shopBuyW/2, shopBuyY+5, tc, 1)
}

func shopFrame(dst *ebiten.Image, x, y, w, h float64, bg, edge color.RGBA) {
	fillRect(dst, x, y, w, h, bg)
	fillRect(dst, x, y, w, 1, edge)
	fillRect(dst, x, y+h-1, w, 1, edge)
	fillRect(dst, x, y, 1, h, edge)
	fillRect(dst, x+w-1, y, 1, h, edge)
}

// drawShopPopup: "BUY ALL?" with how many items and the total cost, and Yes / No.
func (g *Game) drawShopPopup(dst *ebiten.Image) {
	fillRect(dst, 0, 0, screenW, screenH, rgb(0x05070e))
	fillRect(dst, 260, 150, 440, 210, rgb(0x120c24))
	shopFrame(dst, 260, 150, 440, 210, rgb(0x120c24), rgb(0xff6a8a))
	g.drawTextC(dst, "¿COMPRAR TODO?", 480, 166, rgb(0xff6a8a), 2)
	g.drawTextC(dst, fmt.Sprintf("Artículos: %d", g.shopBuyN), 480, 214, rgb(0xf4f0ff), 1)
	g.drawTextC(dst, fmt.Sprintf("Coste total: %d puntos", g.shopBuyCost), 480, 234, rgb(0xf0e060), 1)
	g.drawTextC(dst, fmt.Sprintf("Te quedarán: %d puntos", g.progress.Wallet-g.shopBuyCost), 480, 254, rgb(0xb8b0d0), 1)
	for _, b := range []struct {
		x     float64
		label string
		on    bool
	}{{shopYesX, "SÍ", g.shopBuyYes}, {shopNoX, "NO", !g.shopBuyYes}} {
		bg, edge, tc := rgb(0x1c1634), rgb(0x6a4ab0), rgb(0xb8b0d0)
		if b.on {
			bg, edge, tc = rgb(0x2a1a4a), rgb(0xff6a8a), rgb(0xffffff)
		}
		shopFrame(dst, b.x, shopPopBtnY, shopPopBtnW, shopPopBtnH, bg, edge)
		g.drawTextC(dst, b.label, b.x+shopPopBtnW/2, shopPopBtnY+9, tc, 1)
	}
}

// drawWrapped draws text broken into lines that fit width w.
func (g *Game) drawWrapped(dst *ebiten.Image, s string, x, y, w float64, col color.RGBA) {
	line, words := "", []string{}
	cur := ""
	s = tr(s) // translate the whole text first: lines wrapped from the Spanish and translated one by one come out half translated
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
	case "elephant", "giraffe", "kangaroo", "deer", "rabbit", "pig", "duck", "budgie", "boar", "croc", "cheetah", "lion":
		return ownedID("animal:" + n)
	case "quijote", "sancho":
		return quijoteUnlocked()
	}
	return true
}

// quijoteUnlocked: buying either of the two playable characters brings the
// pair of riders to the parties as well.
func quijoteUnlocked() bool { return ownedID("skin:don_quijote") || ownedID("skin:sancho_panza") }

// castSlugs: the special characters bought so far; they are drawn more often
// when the guests of a party are chosen.
var castSlugs []string
