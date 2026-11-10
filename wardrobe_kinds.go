package main

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ---------------------------------------------------------------------------
// VESTUARIO has three pages (wardKind): PERSONAJE (what each player wears, wardrobe.go), ASESINOS and POLICÍAS.
//   ASESINOS: "Asesino" picks one of the killers you have unlocked (goalkillers.go), "Color de la ropa" tints his
//     clothes (saved as Equip["kdress:<slug>"], so there is no new field in the progress) and "Arma de todos" is the
//     weapon every killer carries (Equip["kweapon"], as it always was). pickKiller (killer.go) dresses the killer that
//     comes with his tint: killers drawn by hand in the editor (rigs) are left as they are.
//   POLICÍAS: one row per force; the options are the uniforms you have (the ones escaping opened, copsets.go, plus
//     what was bought or won before). The choice is Equip[<force>], which applyShop already puts on every officer.
// Only what is unlocked or owned is listed: the wardrobe never buys.
// ---------------------------------------------------------------------------

const (
	wardKindPlayer = iota
	wardKindKiller
	wardKindCop
	wardKindN
)

var wardKindNames = [wardKindN]string{"PERSONAJE", "ASESINOS", "POLICÍAS"}

// the page being shown and the killer being dressed (wardrobe state, like wardP)
var (
	wardKind   int
	wardKiller string
)

var killerWardRows = []wardRowDef{{"kwho", "Asesino"}, {"kdress", "Color de la ropa"}, {"kweapon", "Arma de todos"}}

func copWardRows() []wardRowDef {
	var out []wardRowDef
	for _, f := range copForces {
		out = append(out, wardRowDef{f.slot, f.es})
	}
	return out
}

// wardCurRows: the rows of the page in view.
func (g *Game) wardCurRows() []wardRowDef {
	switch wardKind {
	case wardKindKiller:
		return killerWardRows
	case wardKindCop:
		return copWardRows()
	}
	return wardRows
}

func (g *Game) wardGoKind(k int) {
	wardKind = (k%wardKindN + wardKindN) % wardKindN
	wardRow = 0
	wardFlash = 0.3
	if wardKind == wardKindKiller {
		g.wardKillerFix()
	}
}

// wardKillers: the killers you have unlocked, in the order of the ASESINOS cards.
func (g *Game) wardKillers() []wardOption {
	var out []wardOption
	for _, k := range killerCatalog {
		if g.killerGot(k.slug) {
			out = append(out, wardOption{k.slug, k.es})
		}
	}
	for _, k := range killerTrophy {
		if g.killerGot(k.slug) {
			out = append(out, wardOption{k.slug, k.es})
		}
	}
	return out
}

// wardKillerFix keeps wardKiller on a killer that is unlocked ("" when there is none).
func (g *Game) wardKillerFix() {
	l := g.wardKillers()
	for _, o := range l {
		if o.id == wardKiller {
			return
		}
	}
	wardKiller = ""
	if len(l) > 0 {
		wardKiller = l[0].id
	}
}

// wardGet / wardSet: the value of a row of the page in view (the new pages keep theirs in the party equipment).
func (g *Game) wardGet(player int, slot string) string {
	switch slot {
	case "kwho":
		return wardKiller
	case "kdress":
		return g.shopEquip("kdress:" + wardKiller)
	}
	return g.equipOf(player, slot)
}

func (g *Game) wardSet(player int, slot, id string) {
	switch slot {
	case "kwho":
		wardKiller = id
	case "kdress":
		if wardKiller != "" {
			g.setEquip("kdress:"+wardKiller, id)
		}
	default:
		g.setEquipOf(player, slot, id)
	}
}

// wardKindOptions: the options of the rows of the two new pages (ok false: not one of their rows).
func (g *Game) wardKindOptions(slot string) (out []wardOption, ok bool) {
	switch slot {
	case "kwho":
		g.wardKillerFix()
		out = g.wardKillers()
		if len(out) == 0 {
			out = []wardOption{{"", "Ninguno"}}
		}
		return out, true
	case "kdress":
		out = []wardOption{{"", "Original"}}
		for _, t := range tints {
			out = append(out, wardOption{t.id, t.name})
		}
		return out, true
	case "kweapon":
		out = []wardOption{{"", "Original"}}
		for _, it := range shopCatalog[shopWeapons] {
			if it.slot == "kweapon" && g.shopOwns(it) {
				out = append(out, wardOption{it.id, it.name})
			}
		}
		return out, true
	}
	for _, f := range copForces {
		if f.slot != slot {
			continue
		}
		out = []wardOption{{"", "Original"}}
		var mine []shopItem
		for _, it := range shopCatalog[shopUniforms] {
			if it.slot == slot && g.shopOwns(it) {
				mine = append(mine, it)
			}
		}
		// the sets that escapes open come first, in the order they open; then what was bought or won before
		sort.SliceStable(mine, func(i, j int) bool {
			a, b := copSetRank(mine[i].id), copSetRank(mine[j].id)
			if (a > 0) != (b > 0) {
				return a > 0
			}
			return a < b
		})
		for _, it := range mine {
			out = append(out, wardOption{it.id, it.name})
		}
		return out, true
	}
	return nil, false
}

// killerWear dresses a killer spec with what the wardrobe chose for him (his tint).
func (g *Game) killerWear(spec *CharSpec, slug string) {
	if col, ok := tintColor(g.shopEquip("kdress:" + slug)); ok {
		tintSpec(spec, col)
	}
}

// killerDressed: the killer of the assets folder, dressed. Hand-drawn rigs keep their drawing.
func (g *Game) killerDressed(cs *CharSet) *CharSet {
	if cs == nil || cs.Spec == nil || cs.rig != nil {
		return cs
	}
	if _, ok := tintColor(g.shopEquip("kdress:" + cs.Spec.Name)); !ok {
		return cs
	}
	sp := *cs.Spec
	g.killerWear(&sp, sp.Name)
	return newCharSet(&sp)
}

// wardPreviewSpec: the mannequin of the page in view.
func (g *Game) wardPreviewSpec() (key string, mk func() *CharSpec) {
	switch wardKind {
	case wardKindKiller:
		key = "wdk:" + wardKiller + "|" + g.shopEquip("kdress:"+wardKiller) + "|" + g.shopEquip("kweapon")
		return key, func() *CharSpec {
			sp := killerSpecOf(wardKiller)
			g.killerWear(&sp, wardKiller)
			if w, ok := weaponHold[g.shopEquip("kweapon")]; ok && ownedID(g.shopEquip("kweapon")) {
				sp.Weapon = w
			}
			return &sp
		}
	case wardKindCop:
		rows := copWardRows()
		f := copForces[clampI(wardRow, 0, len(rows)-1)]
		id := g.shopEquip(f.slot)
		return "wdc:" + f.slot + "|" + id, func() *CharSpec {
			sp := copSpecOf(f.unit)
			if fx := wearFx[id]; fx != nil && ownedID(id) {
				fx(&sp)
			}
			return &sp
		}
	}
	if wardPick { // the character picker shows the selected cell (wardrobe_char.go)
		return g.wardPickPreview()
	}
	return "wd:" + g.equipSigOf(wardP), func() *CharSpec { return g.playerSpecFor(wardP, "", "") }
}

func wardKindDesc(slot string) string {
	switch slot {
	case "kwho":
		return "Los asesinos que ya has derrotado (mira OBJETIVOS)."
	case "kdress":
		return "Tiñe la ropa de este asesino cuando salga a la fiesta."
	case "kweapon":
		return "El arma que llevan todos los asesinos. Solo las que ya son tuyas."
	}
	return "Cómo viste esta policía en la fiesta. Los uniformes nuevos se ganan escapando de ella."
}

// updateWardKinds: the mouse on the tabs of the pages.
func (g *Game) updateWardKinds() {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	x, y := ebiten.CursorPosition()
	if y < 100 || y > 120 {
		return
	}
	for i := 0; i < wardKindN; i++ {
		if x >= 244+i*104 && x < 244+(i+1)*104 && i != wardKind {
			g.wardGoKind(i)
		}
	}
}

// drawWardKinds: the tabs PERSONAJE | ASESINOS | POLICÍAS under the groups of the shop.
func (g *Game) drawWardKinds(dst *ebiten.Image) {
	for i, n := range wardKindNames {
		col := rgb(0x8a86a0)
		if i == wardKind {
			col = rgb(0xf0e060)
			fillRect(dst, float64(244+i*104), shopTabLineY, 100, 1, col)
		}
		g.drawText(dst, n, float64(246+i*104), shopTabTextY, col, 0.95)
	}
}
