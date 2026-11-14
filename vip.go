package main

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// VIP ZONES, THE GOLD SUIT, STOLEN INVITATIONS AND THE GUARD SHIFT (ideas 21, 26, 27, 30).
//
// GOLD SUIT ("traje dorado", prop "vipsuit"): picked up with the same pickupInteract as poison and clothes.
//   It is worn at once and lasts suitDuration seconds, single use. While worn NOBODY can see or report the
//   wearer: g.sees (guests, guards, cops, killers' eyes) and secCam.sees (cameras) answer false for his position, so
//   the witness chain (witness, cameraWitness, stealthSeen) never fires. Per player. The last suitWarn seconds blink.
//
// VIP ZONE: levels listed in vipPlan (by title, levels.go untouched) get a fenced red carpet found by
//   findVIPSite: a cordon of rope segments (real props, so overlap tests see them) with a gate in the south
//   side, two door guards outside, one inside, a VIP from vipRegistry and a gold suit as loot. The staff are
//   Guards with the vipGuard fields: they stand still. A player inside the zone (or in front of its gate) who
//   has no suit and no pass wakes all the staff: they chase with the ordinary guard chase (leash, 3 s to lose
//   him) and go back to their posts. Attacking staff wakes them too. The staff and the VIP do NOT count for
//   the goal of the party (aliveCount / total), but killing the VIP gives his bonus, an immediate police call
//   (cuffs unless the killer wears the suit) and reinforcements (2 stars at least).
//
// INVITATION (26): two guests carry a golden tag. Killing one takes the card (the simplest rule). Walking into the
//   zone with a card spends it (single use) and gives vipPassTime seconds of pass. Per player.
//
// GUARD SHIFT (27): every vipShiftEvery seconds the door guards yawn and look at the watch (vipYawnTime) and then
//   leave their posts for the rest of the window (vipShiftWindow in all): nobody guards the gate and the inner guard
//   ignores the player. The clock is paused while the staff is alarmed.
//
// VIP REGISTRY (50): vipRegistry() lists 50 invented VIPs (11 here, 39 in vipmore.go) (id, names, rank, behaviour, bonus, look). The plan gives a
//   different one to each level. vipEliminated is the in-memory "killed" mark for the future card album
//   (vipSnapshot); nothing is saved to disk here.
// ---------------------------------------------------------------------------

const (
	suitDuration   = 8.0
	suitWarn       = 2.0
	vipPassTime    = 15.0 // halved (was 30)
	vipShiftEvery  = 40.0
	vipYawnTime    = 1.5
	vipShiftWindow = 3.5 // the whole relief, yawning included (halved, was 7; vipShiftEvery stays 40: it is the gap between reliefs, not a time in the zone)
	vipKnockTime   = 1.5 // first warning: thrown to the floor
	vipStunTime    = 3.0 // second warning: stars over the head
	vipImmuneTime  = 3.0 // after a warning nobody chains another one
	vipCarriers    = 2
	vipSeg         = 32.0 // length of a rope segment along the cordon
	vipDepth       = 64.0 // depth of the carpet
	vipDefaultPay  = 1000
)

// roles of the staff
const (
	vipDoor  = 1
	vipBoss  = 2
	vipInner = 3
)

// vipGuard: the extra fields of a Guard that belongs to a VIP zone (embedded in Guard).
type vipGuard struct {
	vip     int // 0 = ordinary guard, vipDoor, vipBoss, vipInner
	vipID   string
	vipPost Vec
	vipAway Vec
	vipFace Dir
	vipT    float64
	vipHP   int
	vipUsed bool // the tip was given
	vipRep  float64
}

// vipBehaviour
const (
	vbStand = iota // stays where he is
	vbFlee         // runs away when he sees an intruder
	vbCall         // calls the police when somebody breaks in
	vbTip          // gives a tip if helped (E); also an easy kill
	vbTough        // takes several blows
)

type vipDef struct {
	id, es, en     string
	cardES, cardEN string // short name for the album card (the full one does not fit its width); empty: use es
	rankES, rankEN string
	behaviour      int
	bonus          int
	hp             int
	lines          [][2]string // ES, EN: talk, talk, alarm (the third one)
	spec           func() CharSpec
	tipES, tipEN   string
	elimES, elimEN string
}

func vipBase() CharSpec {
	s := baseSpec(rand.New(rand.NewSource(5)))
	s.Hair2 = HairShort
	return s
}

var vipDefs = []vipDef{
	{id: "yakuza", es: "EL PADRINO NIPÓN", en: "THE NIPPON GODFATHER", cardES: "PADRINO", cardEN: "GODFATHER", rankES: "Jefe mafioso", rankEN: "Crime boss", behaviour: vbCall, bonus: 1500, hp: 1,
		lines: [][2]string{{"Mi katana es de goma... ¡pero cuenta!", "My katana is rubber... but it counts!"}, {"¿Sushi? Solo si lo corta un amigo", "Sushi? Only if a friend slices it"}, {"¡Llamad a mis abogados! ¡Y a la poli!", "Call my lawyers! And the cops!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0xf0c8a0), hx(0x101018)
			s.Hair2, s.Glasses, s.Body = HairComb, GlassShades, BodyBig
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopRobe, hx(0x1c1c2c), hx(0xb01c2c), hx(0xe8c040), hx(0x1c1c2c)
			s.Carry, s.CarryCol = HoldSaber, hx(0xd0d0d8)
			return s
		}},
	{id: "pope", es: "SU SANTIDAD PÍO XCIX", en: "HIS HOLINESS PIUS XCIX", cardES: "PONTÍFICE", cardEN: "PONTIFF", rankES: "Pontífice", rankEN: "Pontiff", behaviour: vbTip, bonus: 2000, hp: 1,
		lines: [][2]string{{"Bendigo esta fiesta (y el minibar)", "I bless this party (and the minibar)"}, {"Mi báculo mola más que tu móvil", "My staff beats your phone"}, {"¡Santo cielo! ¡Seguridad!", "Holy heavens! Security!"}},
		tipES: "Bendición papal", tipEN: "Papal blessing",
		spec: func() CharSpec {
			s := vipBase()
			s.Skin = hx(0xf0d0b8)
			s.Hair, s.Hair2, s.Glasses, s.Hat = hx(0xc4c4c8), HairBald, GlassNone, HatNone // mitre and grey hair: ptf_art.go
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopRobe, hx(0xf6f4ee), hx(0xe8c040), hx(0xe8c040), hx(0xf6f4ee)
			s.Shoes, s.Pattern = hx(0x18181c), ptfPopeMark // long cassock with a golden stole (ptf_art.go)
			s.Carry, s.CarryCol = HoldSaber, hx(0xf0c030)
			return s
		}},
	{id: "king", es: "S.M. EL REY", en: "HIS MAJESTY THE KING", cardES: "EL REY", cardEN: "THE KING", rankES: "Monarca", rankEN: "Monarch", behaviour: vbStand, bonus: 1500, hp: 1,
		lines: [][2]string{{"Esta corona es de plástico dorado", "This crown is gold-plated plastic"}, {"Yo no abdico... de la barra libre", "I never abdicate... from the open bar"}, {"¡Guardias! ¡Que le corten la cabeza!", "Guards! Off with his head!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0xf0c8a8), hx(0xc8c8c8)
			s.Hair2, s.Beard, s.Hat = HairShort, BeardFull, HatCrown
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopRobe, hx(0xa01830), hx(0xf0e8d8), hx(0xe8c040), hx(0x6a1020)
			s.Cape = hx(0xf0ece4)
			s.Carry, s.CarryCol = HoldTrophy, hx(0xf0c030)
			return s
		}},
	{id: "president", es: "EL PRESIDENTE", en: "THE PRESIDENT", rankES: "Presidente", rankEN: "President", behaviour: vbCall, bonus: 1500, hp: 1,
		lines: [][2]string{{"Les prometo barra libre. (Mentira)", "I promise an open bar. (Lie)"}, {"¡Un selfie con mi banderita, por favor!", "A selfie with my little flag, please!"}, {"¡Esto es un ataque! ¡Llamen a la policía!", "This is an attack! Call the police!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0xf0c8a8), hx(0xb8b8c0)
			s.Hair2 = HairSide
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopSuit, hx(0x1c2a5a), hx(0xf4f4f8), hx(0xc02030), hx(0x1c2a5a)
			s.Carry = HoldFlag
			return s
		}},
	{id: "rockstar", es: "LA SUPERESTRELLA DEL ROCK", en: "THE ROCK SUPERSTAR", cardES: "SUPERESTRELLA", cardEN: "SUPERSTAR", rankES: "Estrella del rock", rankEN: "Rock star", behaviour: vbFlee, bonus: 1000, hp: 1,
		lines: [][2]string{{"¡Mi guitarra tiene más fans que tú!", "My guitar has more fans than you!"}, {"No firmo autógrafos antes del brunch", "No autographs before brunch"}, {"¡Mis fans! ¡Sacadme de aquí!", "My fans! Get me out of here!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0xf0c8a8), hx(0xf050a0)
			s.Hair2, s.Glasses = HairSpiky, GlassShades
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopVest, hx(0x101018), hx(0xe8c040), hx(0xe8c040), hx(0x202030)
			s.Carry, s.CarryCol = HoldGuitar, hx(0xd02030)
			return s
		}},
	{id: "filmstar", es: "LA ESTRELLA DE CINE", en: "THE MOVIE STAR", rankES: "Estrella de cine", rankEN: "Movie star", behaviour: vbFlee, bonus: 1200, hp: 1,
		lines: [][2]string{{"Tengo una estatuilla. Y otra. Y otra", "I have a statuette. And another. And another"}, {"Mi mejor perfil es este. Y este", "My best side is this one. And this one"}, {"¡Mis dobles! ¡Que vengan mis dobles!", "My stunt doubles! Send my stunt doubles!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0xf0c8a8), hx(0x2a1a10)
			s.Hair2 = HairPompadour
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopSuit, hx(0x14141c), hx(0xf4f4f8), hx(0xc02030), hx(0x14141c)
			s.Carry, s.CarryCol = HoldTrophy, hx(0xf0c030)
			return s
		}},
	{id: "oil", es: "EL MAGNATE PETROLERO", en: "THE OIL TYCOON", cardES: "MAGNATE", cardEN: "TYCOON", rankES: "Magnate", rankEN: "Tycoon", behaviour: vbTip, bonus: 1500, hp: 1,
		lines: [][2]string{{"Tengo yates. Muchos. Demasiados", "I own yachts. Many. Too many"}, {"¡Mi corbata vale más que tu casa!", "My tie is worth more than your house!"}, {"¡Mi dinero! ¡No toquen mi dinero!", "My money! Don't touch my money!"}},
		tipES: "Propina de magnate", tipEN: "Tycoon's tip",
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0xf0c8a8), hx(0x4a3a2a)
			s.Hair2, s.Hat, s.HatCol, s.Beard, s.Body = HairShort, HatCowboy, hx(0xe8e0c8), BeardMustache, BodyFat
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopSuit, hx(0x3a2a1a), hx(0xf4f4f8), hx(0x30a050), hx(0x3a2a1a)
			s.Carry, s.CarryCol = HoldBriefcase, hx(0x30a050)
			return s
		}},
	{id: "chef", es: "EL CHEF ESTRELLA", en: "THE STAR CHEF", rankES: "Chef estrella", rankEN: "Star chef", behaviour: vbTough, bonus: 1200, hp: 3,
		lines: [][2]string{{"¡Este gorro tiene tres estrellas!", "This hat has three stars!"}, {"¡Esa fiesta está cruda!", "This party is raw!"}, {"¡Fuera de mi cocina! ¡Seguridad!", "Out of my kitchen! Security!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0xf0c8a8), hx(0x3a2a20)
			s.Hat, s.HatCol, s.Beard, s.Body = HatChef, hx(0xf8f8f8), BeardMustache, BodyFat
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopUniform, hx(0xf8f8f8), hx(0xf8f8f8), hx(0xc02030), hx(0x30303a)
			s.Carry = HoldPizza
			return s
		}},
	{id: "boxer", es: "EL CAMPEÓN DEL MUNDO", en: "THE WORLD CHAMPION", cardES: "CAMPEÓN", cardEN: "CHAMPION", rankES: "Campeón de boxeo", rankEN: "Boxing champ", behaviour: vbTough, bonus: 1800, hp: 4,
		lines: [][2]string{{"¡Mi cinturón pesa más que tú!", "My belt weighs more than you!"}, {"Vuela como mariposa... y cobra mucho", "Float like a butterfly... and bill a lot"}, {"¡Esto lo arreglo yo a puñetazos!", "I'll sort this out with my fists!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0x8a5a3a), hx(0x101010)
			s.Hair2, s.Body = HairBald, BodyBig
			s.Top, s.Shirt, s.Bottom, s.Pants, s.Accent = TopBare, hx(0x8a5a3a), BotTrunks, hx(0xe8c040), hx(0xe8c040)
			s.Carry, s.CarryCol = HoldTrophy, hx(0xf0c030)
			return s
		}},
	{id: "sultan", es: "EL SULTÁN DEL DESIERTO", en: "THE DESERT SULTAN", cardES: "SULTÁN", cardEN: "SULTAN", rankES: "Sultán", rankEN: "Sultan", behaviour: vbStand, bonus: 1500, hp: 1,
		lines: [][2]string{{"Mi alfombra vuela. La tuya, no", "My carpet flies. Yours doesn't"}, {"Tengo un turbante para cada día", "I own a turban for every day"}, {"¡Mi guardia imperial! ¡A mí!", "My imperial guard! To me!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin, s.Hair = hx(0xc89868), hx(0x20140c)
			s.Hat, s.HatCol, s.Beard = HatTurban, hx(0xf0e8d0), BeardFull
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopRobe, hx(0x2a4aa0), hx(0xe8c040), hx(0xe8c040), hx(0x2a4aa0)
			s.Carry, s.CarryCol = HoldTrophy, hx(0xf0c030)
			return s
		}},
	{id: "astronaut", es: "EL ASTRONAUTA LEGENDARIO", en: "THE LEGENDARY ASTRONAUT", cardES: "ASTRONAUTA", cardEN: "ASTRONAUT", rankES: "Astronauta", rankEN: "Astronaut", behaviour: vbFlee, bonus: 1300, hp: 1,
		lines: [][2]string{{"He pisado la Luna. ¿Y tú?", "I walked on the Moon. And you?"}, {"Aquí no hay gravedad... ni cobertura", "No gravity here... and no signal"}, {"¡Houston, tenemos un problema!", "Houston, we have a problem!"}},
		spec: func() CharSpec {
			s := vipBase()
			s.Skin = hx(0xf0c8a8)
			s.Hat, s.HatCol = HatHelmet, hx(0xf4f4f8)
			s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants = TopUniform, hx(0xf4f4f8), hx(0xf4f4f8), hx(0xe05020), hx(0xe8e8f0)
			s.Carry = HoldFlag
			return s
		}},
}

// vipRegistry: every VIP of the game (the card album will read this).
func vipRegistry() []vipDef { return vipDefs }

func vipByID(id string) *vipDef {
	for i := range vipDefs {
		if vipDefs[i].id == id {
			return &vipDefs[i]
		}
	}
	return nil
}

// vipEliminated: ids of the VIPs killed in this session (in memory only; nothing is written to disk yet).
var vipEliminated = map[string]bool{}

// vipSnapshot: a copy of the "eliminated" marks, for the future card album.
func vipSnapshot() map[string]bool {
	out := map[string]bool{}
	for k, v := range vipEliminated {
		out[k] = v
	}
	return out
}

// vipPlan: level title -> the VIP that suits its theme (a light preference, see vipPick). The zone itself is
// random (vipRoll); any level where vipPlace finds room may get one.
var vipPlan = map[string]string{
	"PLAZA DE TOROS":        "king",
	"PARÍS":                 "pope",
	"FESTIVAL":              "rockstar",
	"PARQUE DE ATRACCIONES": "president",
	"ESTACIÓN DE ESQUÍ":     "oil",
	"CRUCERO NOCTURNO":      "yakuza",
}

// suitPlan: gold suits lying around outside the VIP zone, by title (the zone itself always has one).
var suitPlan = map[string]int{
	"HOUSE PARTY": 1, "DISCOTECA": 1, "CASINO": 1, "CRUCERO NOCTURNO": 1, "ESTADIO": 1, "FESTIVAL": 1,
	"PARQUE DE ATRACCIONES": 1, "CINE": 1, "ESTACIÓN ESPACIAL": 1,
}

// ---- props ------------------------------------------------------------------------------------------------

func init() {
	pk := propKinds
	gold, goldD, red := uint32(0xf0c030), uint32(0xa07818), uint32(0xc01830)
	pk["vipsuit"] = &PropKind{W: 20, H: 30, FW: 16, FH: 6, NoSolid: true, Item: "suit", Draw: func(b *pixBuf, v int) {
		b.ellipse(10, 28, 8, 1.8, color.RGBA{0, 0, 0, 70})
		b.rect(9, 1, 2, 4, c(goldD))
		b.rect(8, 1, 4, 1, c(goldD))
		b.rect(2, 5, 16, 3, c(gold)) // shoulders
		b.rect(3, 8, 14, 18, c(gold))
		b.rect(3, 8, 14, 1, c(0xffe880))
		b.rect(3, 8, 2, 18, c(0xd8a820)) // sleeves
		b.rect(15, 8, 2, 18, c(0xd8a820))
		for i := 0; i < 6; i++ { // lapels
			b.rect(8-i/2, 8+i, 1, 1, c(goldD))
			b.rect(11+i/2, 8+i, 1, 1, c(goldD))
		}
		b.rect(9, 8, 2, 14, c(0xf4d870))
		b.rect(8, 11, 4, 2, c(red)) // bow tie
		b.set(6, 14, c(0xffffff))   // sparkles
		b.set(14, 19, c(0xffffff))
		b.set(8, 23, c(0xffffff))
	}}
	pk["viprope_h"] = &PropKind{W: 32, H: 18, FW: 30, FH: 4, Vars: 1, Draw: func(b *pixBuf, v int) {
		for _, x := range []int{1, 28} {
			b.rect(x, 3, 3, 14, c(gold))
			b.rect(x, 3, 1, 14, c(0xffe880))
			b.rect(x-1, 15, 5, 2, c(goldD))
			b.rect(x, 1, 3, 2, c(0xffe880))
		}
		for x := 4; x < 28; x++ { // the rope sags a little
			sag := int(2.2 * math.Sin(math.Pi*float64(x-4)/24))
			b.set(x, 5+sag, c(red))
			b.set(x, 6+sag, c(0x8a1020))
		}
	}}
	pk["viprope_v"] = &PropKind{W: 8, H: 36, FW: 4, FH: 30, Vars: 1, Draw: func(b *pixBuf, v int) {
		for _, y := range []int{6, 27} {
			b.rect(2, y, 3, 8, c(gold))
			b.rect(2, y, 1, 8, c(0xffe880))
			b.rect(1, y+6, 5, 2, c(goldD))
		}
		b.rect(2, 4, 3, 2, c(0xffe880))
		b.rect(3, 14, 1, 13, c(red))
	}}
}

// ---- gold suit ---------------------------------------------------------------------------------------------

// suitOn: the player wears the gold suit.
func (g *Game) suitOn(p *Player) bool { return p != nil && p.suitT > 0 }

// suitHides: q is the position of a player who wears the suit: nobody sees him.
func (g *Game) suitHides(q Vec) bool {
	for _, p := range g.players() {
		if p != nil && p.suitT > 0 && p.pos == q {
			return true
		}
	}
	return false
}

func (g *Game) putOnSuit(p *Player) {
	p.suitT = suitDuration
	g.suitDress(p) // the gold suit is really worn: new clothes, and the handcuffs mark goes like with a disguise
	g.stat("suits", 1)
	g.snd.Play("spark")
	g.burst(Vec{p.pos.X, p.pos.Y - 16}, 24, 80, rgb(0xffd860), 0.7, 2, 0)
	g.say("¡Traje dorado puesto! Nadie puede verte")
	g.award(p.pos, 50, "traje")
}

func (g *Game) updateSuit(dt float64) {
	for _, p := range g.players() {
		if p == nil || p.suitT <= 0 {
			continue
		}
		before := p.suitT
		p.suitT -= dt
		if before > suitWarn && p.suitT <= suitWarn {
			g.say("El traje dorado pierde su brillo...")
		}
		if p.suitT <= 0 {
			p.suitT = 0
			g.say("Se acabó el traje dorado")
			g.snd.Play("alert")
		}
	}
}

// drawSuitFX: gold glow, sparkles and a time bar under the feet (blinks in the last suitWarn seconds).
func (g *Game) drawSuitFX(dst *ebiten.Image, p *Player) {
	if p == nil || p.suitT <= 0 {
		return
	}
	if p.suitT <= suitWarn && int(g.time*8)%2 == 0 {
		return
	}
	g.glowAt(dst, Vec{p.pos.X, p.pos.Y - 18}, 30, color.RGBA{255, 200, 60, 90})
	x, y := g.wx(p.pos.X), g.wy(p.pos.Y)
	for i := 0; i < 5; i++ {
		a := g.time*3 + float64(i)*2*math.Pi/5
		sx, sy := math.Round(x+math.Cos(a)*11), math.Round(y-18+math.Sin(a)*14)
		fillRect(dst, sx, sy, 2, 2, rgb(0xfff2a0))
	}
	g.drawTimeBar(dst, x, y+5, p.suitT/suitDuration, rgb(0xf0c030))
}

func (g *Game) drawTimeBar(dst *ebiten.Image, cx, y, frac float64, col color.RGBA) {
	w := 24.0
	x := math.Round(cx - w/2)
	y = math.Round(y)
	fillRect(dst, x-1, y-1, w+2, 5, rgb(0x20180a))
	fillRect(dst, x, y, math.Round(w*clampF(frac, 0, 1)), 3, col)
}

// ---- the zone -------------------------------------------------------------------------------------------------

type vipZone struct {
	id          string
	in          Rect // carpet
	trig        Rect // stepping in here wakes the staff
	gate        Rect // the opening of the cordon
	gx          float64
	posts       [2]Vec
	aways       [2]Vec
	inner, boss Vec
	loot        Vec
	spawned     bool
	cool        float64 // seconds before the staff can be alarmed again
	shiftT      float64
	guestGd     *Guard // the guard who is throwing a guest out (vipwarn.go)
	guestTgt    *Guest
	guestT      float64
	chaseP      *Player // whom the staff is after (the warning ladder)
	alarmed     bool
	called      bool
	// shapes and the mini VIP party (vipshape.go); legacy = the classic carpet with its old layout
	legacy   bool
	shape    int
	cells    []Rect   // the carpet cells (empty in the classic one: in is the carpet)
	sec      int      // vipSecLow / Normal / High
	vipSpots []Vec    // where the VIPs stand (the first is boss); empty: boss
	inners   []Vec    // where the inner guards stand; nil in the classic one: inner
	extra    []string // the other VIPs of a mini party (ids)
}

// vipRelief: the staff is changing shift (the door guards are away or yawning).
func (z *vipZone) relief() bool { return z.shiftT >= vipShiftEvery }

// vipAwayNow: the door guards have left (after the yawn).
func (z *vipZone) away() bool { return z.shiftT >= vipShiftEvery+vipYawnTime }

func (l *Level) ensureVIP() bool {
	if l.Def == nil {
		return false
	}
	if n := suitPlan[l.Def.Title]; n > 0 {
		have := countKind(l, "vipsuit")
		return l.ensureBooth("vipsuit", have+n, func(p *Prop) bool { return p.Name == "vipsuit" }, nil, 24)
	}
	return false
}

// vipOff: tests switch the zone off to look at a bare level. vipBuilding stops the recursion while it is searched.
var vipOff, vipBuilding bool

type vipPlaced struct {
	def *LevelDef
	z   *vipZone
	fp  uint64 // fingerprint of the definition it was placed on: an edited definition (editor override) is placed again
}

// vipPrint: a hash of what the zone search depends on. A definition is mutable (applyLevelOverride moves water,
// floors and props after a first build), so the pointer alone is not a valid cache key.
func vipPrint(d *LevelDef) uint64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%v|%v|%v|%v|%v|%v|%v|%v|%v", d.W, d.H, d.Start, d.Pools, d.Floors, d.Walls, d.Props, d.Dance, d.Performers)
	return h.Sum64()
}

// vipCacheKey: a placement depends on the definition and on the style (shape, size, security, party).
type vipCacheKey struct {
	def *LevelDef
	st  string
}

var vipCache = map[vipCacheKey]*vipPlaced{}

// vipDecision: the VIP chosen for the level that is being loaded ("" = no zone). loadLevel sets it (vipRoll) just
// before buildLevel and clears it after; every other buildLevel call (tests, generators, flashbacks) gets a bare level.
var vipDecision string

// buildLevel: the level of a definition; when vipDecision names a VIP it gets its zone (the definition is copied
// with the small decoration under the carpet removed and the cordon added; levels.go is never touched).
func buildLevel(def *LevelDef) *Level {
	if id := vipDecision; id != "" && !vipOff && !vipBuilding && def.Title != "" {
		if nd, z := vipPlace(def, id); nd != nil {
			l := buildLevelRaw(nd)
			zc := *z
			zc.id = id // the cache is per style, not per VIP
			zc.extra = nil
			if n := len(zc.vipSpots) - 1; n > 0 {
				zc.extra = append([]string(nil), vipStyleDecision.extra[:minI(n, len(vipStyleDecision.extra))]...)
			}
			l.vip = &zc
			l.addTables()      // cloth tables and chairs of a few indoor parties (pe_tables.go)
			l.addSeats()       // café tables, benches, deckchairs... where the guests sit down (seats.go)
			l.addDivingBoard() // a diving board at the pool (poolboard.go)
			l.addBullRide()    // the mechanical bull of some parties (events_toro.go, only when loadLevel rolled it)
			l.addRoadworks()   // roadworks on the kerb of some parties (roadworks.go, only when loadLevel rolled them)
			return l
		}
	}
	l := buildLevelRaw(def)
	l.addTables()
	l.addSeats()
	l.addDivingBoard()
	l.addBullRide()
	l.addRoadworks()
	return l
}

// vipRemovable: decoration that may be cleared from under the carpet.
func vipRemovable(name string) bool {
	k := propKinds[name]
	if k == nil || k.Trap != "" || k.Phone || k.TechPhone || k.Hide || k.Light || k.Item != "" || k.Poison || k.Ambient > 0 {
		return false
	}
	switch name {
	case "stage", "speaker", "screen", "truss", "dj", "jukebox", "bar", "buffet", "punch", "checkout", "stall", "toilet", "portapotty", "bowl":
		return false
	case "entertainer": // the decorative magician/clown of the birthday party does the magic show (pe_magic.go): never swept
		return false
	}
	return propPriority(name) <= 2
}

// vipClearCost: how many removable props the carpet and its surroundings would sweep away.
func vipClearCost(def *LevelDef, area Rect) int {
	n := 0
	for _, p := range def.Props {
		if !vipRemovable(p.Kind) {
			continue
		}
		k := propKinds[p.Kind]
		if ox, oy := rectOv(instBase(p, k), area); ox > 0 && oy > 0 {
			n++
		} else if ox, oy := rectOv(instFoot(p, k), area); ox > 0 && oy > 0 {
			n++
		}
	}
	return n
}

func vipPlace(def *LevelDef, id string) (*LevelDef, *vipZone) {
	return vipPlaceStyle(def, id, vipStyleDecision)
}

// vipPlaceStyle: the zone of that style (or a smaller/other shape, the classic sizes last) on the definition.
func vipPlaceStyle(def *LevelDef, id string, st vipStyle) (*LevelDef, *vipZone) {
	fp := vipPrint(def)
	ck := vipCacheKey{def, st.key()}
	if c, ok := vipCache[ck]; ok && c.fp == fp {
		if c.def == nil {
			return nil, nil
		}
		return c.def, c.z
	}
	vipBuilding = true
	defer func() { vipBuilding = false }()
	fitAllFootprints()
	strip := *def
	strip.Props = nil
	for _, in := range def.Props {
		if !vipRemovable(in.Kind) {
			strip.Props = append(strip.Props, in)
		}
	}
	l := buildLevelRaw(&strip)
	reach := l.reachFrom(def.Start)
	res := &vipPlaced{fp: fp}
	wantV, wantI := vipWantStaff(st)
	for _, try := range vipTries(st) {
		cells := vipShapeCells(try.shape, try.cols, try.rows, try.flip)
		if try.legacy {
			cells = vipShapeCells(vsRect, try.cols, try.rows, false)
		}
		gate := vipGateCell(cells, try.cols, try.rows)
		if gate.c < 0 {
			continue
		}
		w, depth := vipSeg*float64(try.cols), vipSeg*float64(try.rows)
		// the shaped zones need room for their people: loot + VIPs + at least one inner guard in a party
		if !try.legacy && st.partyN > 0 && len(cells) < 1+wantV+2 {
			continue
		}
		var best Rect
		bs, found := -1e9, false
		for y := 80.0; y+depth+50 < l.H; y += 8 {
			for x := 36.0; x+w+36 < l.W; x += 8 {
				in := Rect{x, y, w, depth}
				pieces := vipCordonCells(x, y, cells, gate)
				loot := Vec{in.X + in.W/2 - 8, in.Y + 0.4*in.H}
				if !try.legacy {
					loot = vipShapeLayout(x, y, cells, gate, try.cols, wantV, wantI).loot
				}
				if l.vipSiteWhyP(in, x+vipSeg*(float64(gate.c)+0.5), pieces, loot, reach) != "" {
					continue
				}
				ca := in.Grow(14)
				ca.H += 40
				if s := dist(Vec{x + w/2, y + depth/2}, def.Start) - 30*float64(vipClearCost(def, ca)); s > bs {
					best, bs, found = in, s, true
				}
			}
		}
		if !found {
			continue
		}
		in := best
		gx := in.X + vipSeg*(float64(gate.c)+0.5)
		z := &vipZone{id: id, in: in, gx: gx, legacy: try.legacy, shape: try.shape, sec: vipSecNormal}
		z.gate = Rect{gx - vipSeg/2, in.Y + in.H - 6, vipSeg, 34}
		z.trig = Rect{in.X - 4, in.Y - 4, in.W + 8, in.H + 8}
		by := in.Y + in.H
		z.posts = [2]Vec{{gx - 24, by + 16}, {gx + 24, by + 16}}
		for i, sg := range []float64{-1, 1} {
			z.aways[i] = z.posts[i]
			for _, o := range []Vec{{70, 36}, {50, 32}, {30, 30}} {
				q := Vec{gx + sg*(24+o.X-24+24), by + o.Y}
				if l.Free(q) && reach[int(q.Y/navCell)*l.cw+int(q.X/navCell)] {
					z.aways[i] = q
					break
				}
			}
		}
		z.inner = Vec{in.X + 34, in.Y + 0.72*in.H}
		z.boss = Vec{in.X + in.W - 40, in.Y + 0.47*in.H}
		z.loot = Vec{in.X + in.W/2 - 8, in.Y + 0.4*in.H}
		if !try.legacy {
			z.cells = vipCellRects(in.X, in.Y, cells)
			z.sec = st.sec
			lay := vipShapeLayout(in.X, in.Y, cells, gate, try.cols, wantV, wantI)
			z.loot, z.vipSpots, z.inners = lay.loot, lay.vips, lay.inners
			if len(z.inners) == 0 {
				z.inners = []Vec{} // low security: nobody inside (never nil: nil = the classic inner guard)
			}
			z.boss = z.vipSpots[0]
		}
		nd := *def
		nd.Props = nil
		clear := in.Grow(14)
		clear.H += 40
		for _, p := range def.Props {
			k := propKinds[p.Kind]
			if vipRemovable(p.Kind) {
				if ox, oy := rectOv(instBase(p, k), clear); ox > 0 && oy > 0 {
					continue
				}
				if ox, oy := rectOv(instFoot(p, k), clear); ox > 0 && oy > 0 {
					continue
				}
			}
			nd.Props = append(nd.Props, p)
		}
		for _, pc := range vipCordonCells(in.X, in.Y, cells, gate) {
			nd.Props = append(nd.Props, PropInst{Kind: pc.name, X: pc.x, Y: pc.y})
		}
		nd.Props = append(nd.Props, PropInst{Kind: "vipsuit", X: z.loot.X, Y: z.loot.Y})
		res.def, res.z = &nd, z
		break
	}
	vipCache[ck] = res
	if res.def == nil {
		return nil, nil
	}
	return res.def, res.z
}

type vipPiece struct {
	name string
	x, y float64
}

// vipCordon: the rope segments around a carpet of nS segments wide (a gap in the middle of the south side).
func vipCordon(in Rect, nS int) []vipPiece {
	var out []vipPiece
	gap := nS / 2
	by := in.Y + in.H + 2
	for i := 0; i < nS; i++ {
		x := in.X + vipSeg*(float64(i)+0.5)
		out = append(out, vipPiece{"viprope_h", x, in.Y})
		if i != gap {
			out = append(out, vipPiece{"viprope_h", x, by})
		}
	}
	for j := 1; j <= int(in.H/32); j++ {
		y := in.Y + 32*float64(j)
		out = append(out, vipPiece{"viprope_v", in.X - 2, y}, vipPiece{"viprope_v", in.X + in.W + 2, y})
	}
	return out
}

// vipSiteOK: can a carpet of nS segments go here? Free ground, away from props, doors, performers, the start,
// dance floors, and the gate must be reachable.
func (l *Level) vipSiteOK(in Rect, nS int, reach []bool) bool {
	return l.vipSiteWhy(in, nS, reach) == ""
}

func (l *Level) vipSiteWhy(in Rect, nS int, reach []bool) string {
	return l.vipSiteWhyP(in, in.X+vipSeg*(float64(nS/2)+0.5), vipCordon(in, nS), Vec{in.X + in.W/2 - 8, in.Y + 0.4*in.H}, reach)
}

// vipSiteWhyP: the checks of vipSiteWhy for any shape (its gate x, its cordon pieces and where its suit lies).
func (l *Level) vipSiteWhyP(in Rect, gx float64, pieces []vipPiece, loot Vec, reach []bool) string {
	area := Rect{in.X - 14, in.Y - 22, in.W + 28, in.H + 22 + 40}
	if area.X < 24 || area.Y < 70 || area.X+area.W > l.W-24 || area.Y+area.H > l.H-24 {
		return "r1"
	}
	for y := area.Y; y <= area.Y+area.H; y += 8 {
		for x := area.X; x <= area.X+area.W; x += 8 {
			p := Vec{x, y}
			if !l.Free(p) || l.badGround(p) || l.InPool(p) {
				return "r2"
			}
		}
	}
	for _, d := range l.Def.Dance {
		if d.Grow(14).Hits(area) {
			return "r3"
		}
	}
	for _, d := range l.Def.Doors {
		if d.R.Grow(26).Hits(area) {
			return "r4"
		}
	}
	c := Vec{in.X + in.W/2, in.Y + in.H/2}
	for _, pf := range l.Def.Performers {
		if dist(pf.Pos, c) < 120 {
			return "r5"
		}
	}
	if dist(l.Def.Start, c) < 160 || area.Grow(30).Has(l.Def.Start.X, l.Def.Start.Y) {
		return "r6"
	}
	for _, p := range l.Props {
		if ox, oy := rectOv(p.base(), area.Grow(4)); ox > 0 && oy > 0 {
			return "r7"
		}
		if ox, oy := rectOv(p.Foot, area.Grow(4)); ox > 0 && oy > 0 {
			return "r8"
		}
	}
	for _, w := range l.Walls {
		if ox, oy := rectOv(w.R, area.Grow(6)); ox > 0 && oy > 0 {
			return "r9"
		}
	}
	front := Vec{gx, in.Y + in.H + 34}
	if !reach[int(front.Y/navCell)*l.cw+int(front.X/navCell)] {
		return "r10"
	}
	for _, pc := range pieces {
		k := propKinds[pc.name]
		if l.propClashes(pc.name, k, Vec{pc.x, pc.y}, 0) {
			return "r11"
		}
	}
	if l.propClashes("vipsuit", propKinds["vipsuit"], loot, 0) {
		return "suit"
	}
	return ""
}

// ---- spawning ------------------------------------------------------------------------------------------------

func vipGuardSpec(r *rand.Rand, k int) CharSpec {
	s := baseSpec(r)
	s.Skin = hx(pick(r, skinTones))
	s.Hair, s.Hair2, s.Glasses, s.Body = hx(0x14101a), HairShort, GlassShades, BodyBig
	s.Top, s.Shirt, s.Shirt2, s.Accent, s.Pants, s.Shoes = TopSuit, hx(0x101018), hx(0xf4f4f8), hx(0xe8c040), hx(0x101018), hx(0x0a0a0e)
	if k%2 == 1 {
		s.Stature = 1
	}
	return s
}

func (g *Game) vipStaffN() int {
	n := 0
	for _, gd := range g.guards {
		if gd.vip != 0 {
			n++
		}
	}
	return n
}

func (g *Game) spawnVIP() {
	z := g.lvl.vip
	if z != nil {
		z.spawned = false
	}
	// the invitation carriers (also without a zone nobody carries one)
	if z == nil || g.stealth {
		return
	}
	d := vipByID(z.id)
	if d == nil { // never an empty zone: an unknown id gets a registry VIP
		d = &vipDefs[0]
		z.id = d.id
	}
	mk := func(pos Vec, role int, k int) *Guard {
		spec := vipGuardSpec(g.rng, k)
		gd := &Guard{pos: pos, dir: DirDown, cs: newCharSet(&spec), route: []Vec{pos}, state: gdWait, home: pos, leash: 190}
		gd.vipGuard = vipGuard{vip: role, vipPost: pos, vipFace: DirDown, vipHP: 1}
		g.guards = append(g.guards, gd)
		return gd
	}
	for i := 0; i < 2; i++ {
		gd := mk(z.posts[i], vipDoor, i)
		gd.vipAway = z.aways[i]
	}
	if z.inners == nil { // the classic carpet: one inner guard
		mk(z.inner, vipInner, 2).vipFace = DirRight
	}
	for i, pos := range z.inners {
		gd := mk(pos, vipInner, 2+i)
		gd.vipFace = []Dir{DirRight, DirLeft, DirDown}[i%3]
	}
	mkBoss := func(d *vipDef, pos Vec, k int) {
		spec := d.spec()
		boss := &Guard{pos: pos, dir: DirDown, cs: newCharSet(&spec), route: []Vec{pos}, state: gdWait, home: pos, leash: 190}
		boss.vipGuard = vipGuard{vip: vipBoss, vipID: d.id, vipPost: pos, vipFace: DirDown, vipHP: d.hp, vipT: 2 + float64(k)*1.7}
		g.guards = append(g.guards, boss)
	}
	mkBoss(d, z.boss, 0)
	for i, id := range z.extra { // the mini VIP party
		if i+1 >= len(z.vipSpots) {
			break
		}
		if e := vipByID(id); e != nil && e.id != d.id {
			mkBoss(e, z.vipSpots[i+1], i+1)
		}
	}
	z.spawned = true
	z.shiftT, z.alarmed, z.called, z.cool = 0, false, false, 0
	// two guests carry an invitation
	var pool []*Guest
	for _, gu := range g.guests {
		if !gu.out() && gu.perf == "" && gu.py == nil && !gu.vipCard && !z.trig.Grow(40).Has(gu.pos.X, gu.pos.Y) {
			pool = append(pool, gu)
		}
	}
	for i := 0; i < vipCarriers && len(pool) > 0; i++ {
		k := g.rng.Intn(len(pool))
		pool[k].vipCard = true
		pool = append(pool[:k], pool[k+1:]...)
	}
}

// ---- running the zone --------------------------------------------------------------------------------------

// vipIntruder: a player inside the zone (or in front of its gate) with no suit and no pass.
func (g *Game) vipIntruder(z *vipZone) *Player {
	for _, p := range g.players() {
		if p == nil || p.dead || p.hidden || p.jail.held || p.suitT > 0 || p.passT > 0 {
			continue
		}
		if p.vipImmT > 0 {
			continue
		}
		if z.touches(p) {
			return p
		}
	}
	return nil
}

// touches: the player is inside the zone or right in front of its gate.
func (z *vipZone) touches(p *Player) bool {
	return z.inside(p.pos.X, p.pos.Y) || z.gate.Has(p.pos.X, p.pos.Y)
}

func (g *Game) updateVIP(dt float64) {
	g.vipDemo()
	if g.lvl == nil || g.lvl.vip == nil || !g.lvl.vip.spawned {
		return
	}
	z := g.lvl.vip
	if z.cool > 0 {
		z.cool -= dt
	}
	// invitation: showing it at the gate spends it
	for _, p := range g.players() {
		if p == nil {
			continue
		}
		if p.vipImmT > 0 {
			p.vipImmT -= dt
		}
		if p.passT > 0 {
			p.passT -= dt
			if p.passT <= 0 {
				p.passT = 0
				g.say("Tu pase VIP ha caducado")
			}
		}
		if p.invite && !p.dead && z.touches(p) {
			p.invite, p.passT = false, vipPassTime
			g.stat("vippass", 1)
			g.snd.Play("spark")
			g.say("Enseñas la invitación: pasas a la zona VIP")
		}
	}
	g.vipGuestWatch(z, dt)
	// shift clock: stops while the staff is alarmed
	alarmed := false
	for _, gd := range g.guards {
		if gd.vip != 0 && gd.vip != vipBoss && !gd.dead && gd.state != gdWait && gd.state != gdPatrol {
			alarmed = true
		}
	}
	z.alarmed = alarmed
	if !alarmed {
		z.shiftT += dt
		if z.shiftT >= vipShiftEvery+vipShiftWindow {
			z.shiftT = 0
		}
	}
}

// vipAlarm: the whole staff goes after the intruder.
func (g *Game) vipAlarm(at Vec) {
	z := g.lvl.vip
	if z == nil || z.cool > 0 {
		return
	}
	z.cool = 3
	z.chaseP = g.nearestPlayer(at)
	i := 0
	for _, gd := range g.guards {
		if gd.vip == 0 || gd.dead {
			continue
		}
		if gd.vip == vipBoss {
			g.vipBossAlarm(gd)
			continue
		}
		switch gd.state {
		case gdWait, gdPatrol, gdReturn, gdSearch, gdInvestigate:
			gd.state, gd.timer = gdReact, 0.4+0.15*float64(i)
			gd.icon, gd.iconT = []string{"¡ALTO AHÍ!", "¡ZONA VIP!", "¡FUERA DE AQUÍ!"}[i%3], 1.6
			gd.lastSeen, gd.chaseT, gd.lost, gd.sight = at, 0, 0, 0
			gd.path = nil
			i++
		}
	}
	if i > 0 {
		g.snd.Play("alert")
		g.say("¡Zona VIP! Los guardias te han visto")
	}
}

func (g *Game) vipBossAlarm(gd *Guard) {
	d := vipByID(gd.vipID)
	if d == nil {
		return
	}
	gd.icon, gd.iconT = d.lines[2][0], 2.4
	z := g.lvl.vip
	if d.behaviour == vbCall && z != nil && !z.called && !g.dbg.noPolice {
		z.called = true
		g.raisePolice(gd.pos, false)
		g.vipReinforce()
	}
}

func (g *Game) vipReinforce() {
	g.stars = maxI(g.stars, 2)
	if g.stars > g.starFloor {
		g.starFloor = g.stars
	}
	g.escalateTo(unitForStars(g.stars))
}

func (z *vipZone) relieving() bool { return z.relief() }

// updateVIPGuard: returns true when the ordinary guard logic must be skipped this frame.
func (g *Game) updateVIPGuard(gd *Guard, dt float64) bool {
	z := g.lvl.vip
	if z == nil {
		return false
	}
	if gd.vip == vipBoss {
		g.updateVIPBoss(gd, z, dt)
		return true
	}
	if g.vipEscalates() && (gd.state == gdReact || gd.state == gdChase) {
		g.vipStaffChase(gd, z, dt)
		return true
	}
	if gd.state != gdWait && gd.state != gdPatrol {
		return false // reacting, chasing, searching, going back: the ordinary logic
	}
	gd.state = gdWait
	if z.guestGd == gd && g.vipGuestRun(gd, z, dt) {
		return true
	}
	if !z.relieving() {
		if p := g.vipIntruder(z); p != nil {
			g.vipAlarm(p.pos)
			return true
		}
	}
	want := gd.vipPost
	if gd.vip == vipDoor && z.away() {
		want = gd.vipAway
	}
	if z.relieving() && gd.icon == "" && !z.away() {
		gd.icon, gd.iconT = []string{"*BOSTEZO*", "¿Qué hora es?", "Ya casi es mi relevo"}[int(z.shiftT*7)%3], 1.4
	}
	gd.vipRep -= dt
	if dist(gd.pos, want) > 3 {
		if len(gd.path) == 0 && gd.vipRep <= 0 {
			gd.path = g.lvl.Path(gd.pos, want)
			gd.vipRep = 0.6
		}
		g.guardWalk(gd, 36, dt)
		if gd.moving {
			gd.anim += dt * stridePhase(gd.curSpd)
		}
		return true
	}
	gd.moving = false
	if want == gd.vipPost {
		gd.dir = gd.vipFace
	}
	return true
}

func (g *Game) updateVIPBoss(gd *Guard, z *vipZone, dt float64) {
	gd.state = gdWait
	gd.moving = false
	d := vipByID(gd.vipID)
	if d == nil {
		return
	}
	intr := g.vipIntruder(z)
	if intr != nil && !z.relieving() {
		g.vipAlarm(intr.pos)
	}
	if d.behaviour == vbFlee && intr != nil && !z.relieving() {
		// away from him, inside the carpet
		dx, dy := gd.pos.X-intr.pos.X, gd.pos.Y-intr.pos.Y
		n := math.Hypot(dx, dy)
		if n < 1 {
			n = 1
		}
		step := 48 * dt
		np := Vec{gd.pos.X + dx/n*step, gd.pos.Y + dy/n*step}
		box := z.in.Grow(-10)
		np.X, np.Y = clampF(np.X, box.X, box.X+box.W), clampF(np.Y, box.Y+6, box.Y+box.H)
		if !g.lvl.SolidP(np.X, np.Y) && z.inside(np.X, np.Y) {
			gd.dir = faceTo(gd.pos, np)
			gd.pos = np
			gd.moving, gd.curSpd = true, 48
			gd.anim += dt * stridePhase(48)
		}
		return
	}
	if dist(gd.pos, gd.vipPost) > 3 && intr == nil { // calm again: back to his spot
		np := Vec{gd.pos.X + (gd.vipPost.X-gd.pos.X)*math.Min(1, 3*dt), gd.pos.Y + (gd.vipPost.Y-gd.pos.Y)*math.Min(1, 3*dt)}
		if !g.lvl.SolidP(np.X, np.Y) {
			gd.pos = np
		}
		return
	}
	gd.dir = gd.vipFace
	gd.vipT -= dt
	if gd.vipT <= 0 && gd.icon == "" {
		for _, p := range g.players() {
			if p != nil && !p.dead && dist(p.pos, gd.pos) < 150 {
				gd.icon, gd.iconT = d.lines[g.rng.Intn(2)][0], 3.2
				gd.vipT = 6 + g.rng.Float64()*5
				break
			}
		}
	}
}

// vipTough: a VIP with several hit points takes a blow instead of dying.
func (g *Game) vipTough(gd *Guard) bool {
	if gd.vip != vipBoss || gd.vipHP <= 1 {
		return false
	}
	gd.vipHP--
	gd.icon, gd.iconT = "¡AUCH!", 1
	g.snd.Play("thud")
	g.burst(Vec{gd.pos.X, gd.pos.Y - 16}, 8, 50, rgb(0xb01818), 0.4, 2, 200)
	if z := g.lvl.vip; z != nil && !g.suitOn(g.p) {
		g.vipAlarm(g.p.pos)
	}
	return true
}

// vipInZone: is this spot inside the VIP area (the zone where the staff works) of the current party?
func (g *Game) vipInZone(pos Vec) bool {
	z := g.lvl.vip
	return z != nil && z.inside(pos.X, pos.Y)
}

// vipOnKill: a staff member just died by the player's hand.
func (g *Game) vipOnKill(gd *Guard) {
	if gd.vip == 0 {
		return
	}
	z := g.lvl.vip
	if z != nil && !g.suitOn(g.p) {
		z.cool = 0
		g.vipAlarm(g.p.pos)
	}
	if gd.vip != vipBoss {
		return
	}
	d := vipByID(gd.vipID)
	if d == nil {
		return
	}
	// the card (and the "eliminated" mark) only count when he dies inside the VIP area of the party
	if g.vipInZone(gd.pos) {
		vipEliminated[d.id] = true
		g.albumMark("vip:"+d.id, albumKilled)
	}
	pay := d.bonus
	if pay <= 0 {
		pay = vipDefaultPay
	}
	g.award(gd.pos, pay, "VIP ELIMINADO")
	g.say("VIP ELIMINADO: " + d.es)
	g.snd.Play("win")
	g.shake = 0.4
	g.burst(Vec{gd.pos.X, gd.pos.Y - 16}, 30, 100, rgb(0xffd860), 0.9, 2, 40)
	g.stat("vips", 1)
	if n := len(g.bodies); n > 0 {
		g.bodies[n-1].label = d.es
	}
	// the police come at once, with reinforcements; he is cuffed unless nobody could see him
	if g.suitOn(g.p) {
		g.raisePolice(gd.pos, false)
	} else {
		g.suspectSeenBy(suspPlayer, nil, true)
		g.raisePolice(gd.pos, true)
	}
	g.vipReinforce()
}

// vipInteract: help the VIP who tips (E close to him).
func (g *Game) vipInteract(p *Player) bool {
	for _, gd := range g.guards {
		if gd.vip != vipBoss || gd.dead || gd.vipUsed || dist(p.pos, gd.pos) > 30 {
			continue
		}
		d := vipByID(gd.vipID)
		if d == nil || d.behaviour != vbTip {
			continue
		}
		gd.vipUsed = true
		if g.vipInZone(gd.pos) {
			g.albumMark("vip:"+d.id, albumHelped)
		}
		g.stat("vipfriend", 1)
		gd.icon, gd.iconT = "¡Gracias, amigo!", 2.5
		g.snd.Play("spark")
		g.award(gd.pos, 300, "PROPINA")
		g.say("Ayudas al VIP: " + d.tipES)
		return true
	}
	return false
}

// vipTakeCard: a guest who carried an invitation was killed by the current player.
func (g *Game) vipTakeCard(gu *Guest) {
	gu.vipCard = false
	g.p.invite = true
	g.snd.Play("spark")
	g.say("Tienes una invitación VIP (un solo uso)")
}

// ---- drawing --------------------------------------------------------------------------------------------------

func (g *Game) drawVIPFloor(dst *ebiten.Image) {
	if g.lvl == nil || g.lvl.vip == nil {
		return
	}
	z := g.lvl.vip
	rr := func(r Rect, col color.RGBA) {
		fillRect(dst, g.wx(r.X), g.wy(r.Y), r.W*wk, r.H*wk, col)
	}
	run := Rect{z.gx - 12, z.in.Y + z.in.H - 2, 24, 30}
	rr(run, rgb(0x8a1424))
	rr(Rect{run.X, run.Y, 2, run.H}, rgb(0xe8c040))
	rr(Rect{run.X + run.W - 2, run.Y, 2, run.H}, rgb(0xe8c040))
	if !z.legacy && len(z.cells) > 0 { // round, oval, L, platform... (vipshape.go)
		g.drawVIPShapeFloor(dst, z)
		g.drawTextC(dst, "VIP", math.Round(g.wx(z.gx)), math.Round(g.wy(z.in.Y+z.in.H+10)), rgb(0xf0d060), 1)
		return
	}
	rr(z.in, rgb(0x9a1628))
	rr(Rect{z.in.X, z.in.Y, z.in.W, 3}, rgb(0x6a0c1a))
	rr(Rect{z.in.X + 3, z.in.Y + 5, z.in.W - 6, z.in.H - 8}, rgb(0xa81c30))
	for _, e := range []Rect{{z.in.X + 3, z.in.Y + 5, z.in.W - 6, 1}, {z.in.X + 3, z.in.Y + z.in.H - 4, z.in.W - 6, 1}, {z.in.X + 3, z.in.Y + 5, 1, z.in.H - 8}, {z.in.X + z.in.W - 4, z.in.Y + 5, 1, z.in.H - 8}} {
		rr(e, rgb(0xe8c040))
	}
	g.drawTextC(dst, "VIP", math.Round(g.wx(z.gx)), math.Round(g.wy(z.in.Y+z.in.H+10)), rgb(0xf0d060), 1)
}

// drawVIPOverlay: golden tags beside the invitation carriers, "help" hints, pass bars.
func (g *Game) drawVIPOverlay(dst *ebiten.Image) {
	if g.mode != ModePlay || g.lvl == nil || g.lvl.vip == nil {
		return
	}
	for _, gu := range g.guests {
		if !gu.vipCard || gu.out() {
			continue
		}
		x, y := math.Round(g.wx(gu.pos.X)), math.Round(g.wy(gu.pos.Y))
		fillRect(dst, x+7, y-17, 7, 10, rgb(0x4a3408))
		fillRect(dst, x+8, y-16, 5, 8, rgb(0xf0c030))
		fillRect(dst, x+8, y-16, 5, 1, rgb(0xfff2a0))
		fillRect(dst, x+9, y-13, 3, 1, rgb(0xa07818))
		if int(g.time*3)%2 == 0 {
			fillRect(dst, x+14, y-19, 2, 2, rgb(0xffffff))
		}
	}
	for _, p := range g.players() {
		if p == nil || p.dead || p.hidden {
			continue
		}
		x, y := g.wx(p.pos.X), g.wy(p.pos.Y)
		if p.passT > 0 {
			g.drawTimeBar(dst, x, y+11, p.passT/vipPassTime, rgb(0x60e0a0))
		} else if p.invite {
			fillRect(dst, math.Round(x)-3, math.Round(y)+6, 7, 5, rgb(0xf0c030))
		}
		for _, gd := range g.guards {
			if gd.vip == vipBoss && !gd.dead && !gd.vipUsed && dist(p.pos, gd.pos) < 30 {
				if d := vipByID(gd.vipID); d != nil && d.behaviour == vbTip {
					g.drawTextC(dst, "["+actionHintFor(actInteract, p.idx)+"] Ayudar al VIP", math.Round(g.wx(gd.pos.X)), math.Round(g.wy(gd.pos.Y))-58, rgb(0xffd860), 1)
				}
			}
		}
	}
}

// ---- debug (-vipdemo) ---------------------------------------------------------------------------------------

var dbgVIPDemo string

func (g *Game) vipDemo() {
	if dbgVIPDemo == "" || g.time < 0.5 || g.lvl == nil || g.lvl.vip == nil {
		return
	}
	mode := dbgVIPDemo
	dbgVIPDemo = ""
	z := g.lvl.vip
	front := Vec{z.gx, z.in.Y + z.in.H + 44}
	switch mode {
	case "gate":
		g.p.pos = front
	case "inside":
		g.p.pos = Vec{z.gx, z.in.Y + z.in.H - 12}
	case "suit":
		g.p.pos = Vec{z.gx, z.in.Y + z.in.H - 22}
		g.putOnSuit(g.p)
		g.p.suitT = 99
	case "relief":
		g.p.pos = front
		z.shiftT = vipShiftEvery + 3
	case "invite":
		g.p.pos = front
		g.p.invite = true
	case "stun":
		g.p.pos = front
		g.p.stunT = 999
		g.p.vipImmT = 999
	}
}

// ---- English strings (lang_data.go stays untouched) -------------------------------------------------------------

func init() {
	m := map[string]string{
		"Ponerte el traje dorado":                    "Put on the gold suit",
		"¡Traje dorado puesto! Nadie puede verte":    "Gold suit on! Nobody can see you",
		"El traje dorado pierde su brillo...":        "The gold suit is losing its shine...",
		"Se acabó el traje dorado":                   "The gold suit is spent",
		"Tu pase VIP ha caducado":                    "Your VIP pass has expired",
		"Enseñas la invitación: pasas a la zona VIP": "You show the invitation: in you go",
		"¡Zona VIP! Los guardias te han visto":       "VIP area! The guards spotted you",
		"Tienes una invitación VIP (un solo uso)":    "You got a VIP invitation (single use)",
		"ZONA VIP":             "VIP AREA",
		"Ayudar al VIP":        "Help the VIP",
		"[E] Ayudar al VIP":    "[E] Help the VIP",
		"+50 traje":            "+50 suit",
		"+300 PROPINA":         "+300 TIP",
		"¡Gracias, amigo!":     "Thanks, friend!",
		"¡AUCH!":               "OUCH!",
		"¡ALTO AHÍ!":           "HALT!",
		"¡ZONA VIP!":           "VIP AREA!",
		"¡FUERA DE AQUÍ!":      "GET OUT OF HERE!",
		"*BOSTEZO*":            "*YAWN*",
		"¿Qué hora es?":        "What time is it?",
		"Ya casi es mi relevo": "My shift is almost over",
	}
	for _, d := range vipDefs {
		m["VIP ELIMINADO: "+d.es] = "VIP ELIMINATED: " + d.en
		m["Ayudas al VIP: "+d.tipES] = "You help the VIP: " + d.tipEN
		m["+"+itoa(d.bonus)+" VIP ELIMINADO"] = "+" + itoa(d.bonus) + " VIP ELIMINATED"
		for _, ln := range d.lines {
			m[ln[0]] = ln[1]
		}
		m[d.es] = d.en
		m[d.rankES] = d.rankEN
		if d.cardES != "" {
			m[d.cardES] = d.cardEN
		}
	}
	for es, en := range m {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
}

// vipIDs: the ids of the registry, sorted (tests, album).
func vipIDs() []string {
	var out []string
	for _, d := range vipDefs {
		out = append(out, d.id)
	}
	sort.Strings(out)
	return out
}
