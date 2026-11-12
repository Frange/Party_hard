package main

import (
	"image/color"
	"math"
	"math/rand"
	"time"
)

// ---------------------------------------------------------------------------
// The Fall Guys as party guests: nine "beans" (judías), our own pixel-art in the style of the game
// (a tall, fat pill-shaped body with no neck, a round white face with two black eyes, flipper arms
// at the sides and short stubby feet), one per costume:
//   judia_azul (plain blue), judia_animadora (pink with a headband and a blue and white skirt),
//   judia_lima (lime green), judia_pelofuego (flaming orange hair, red jersey, pink arms),
//   judia_gallina (grey chicken suit with a red comb), judia_hamburguesa (burger with orange hair
//   and a black cape), judia_zanahoria (orange carrot with a green tuft), judia_callejera (black
//   cap, red tee and jeans) and judia_pinguino (purple penguin with a beak).
// They have their own procedural drawing (genFallBean, hooked in genChar like Crash or the Simpsons):
// the generic body cannot be a bean. Animation is only the drawing (hitbox, speed, stats and the
// one-per-party rule are the usual ones): the body waddles side to side when walking (gait "bean":
// the heavy lumbering walk, a hopping dance), the flippers follow the arms of the pose, the feet the
// legs, and the falls and deaths use the standard animations. The card says "Flan Guys" under the
// name (the game, like other franchises shown in the album). They are rare special guests: in the
// special pool (each one at most once per party, uniqueFamous) and, now and then, a gang of 2-4 of
// them arrives together (spawnBeanGang) taking famous places, never adding people to the party.
// Every preset uses a fixed random source of its own, so the shared random stream of builtinChars
// (export.go) is not moved.
// Standing rows: flames/comb/leaves up to 3 rows above the body, body 16..38 (23 rows), feet to 41.
// ---------------------------------------------------------------------------

const (
	fgNone = iota
	fgAzul
	fgAnimadora
	fgLima
	fgPelofuego
	fgGallina
	fgHamburguesa
	fgZanahoria
	fgCallejera
	fgPinguino
)

// beanSlugs: the nine beans, in album order.
var beanSlugs = []string{"judia_azul", "judia_animadora", "judia_lima", "judia_pelofuego", "judia_gallina",
	"judia_hamburguesa", "judia_zanahoria", "judia_callejera", "judia_pinguino"}

var beanKinds = map[string]int{"judia_azul": fgAzul, "judia_animadora": fgAnimadora, "judia_lima": fgLima,
	"judia_pelofuego": fgPelofuego, "judia_gallina": fgGallina, "judia_hamburguesa": fgHamburguesa,
	"judia_zanahoria": fgZanahoria, "judia_callejera": fgCallejera, "judia_pinguino": fgPinguino}

// beanGame: the line under the name on their cards (and in the detail window).
const beanGame = "Flan Guys"

func beanKind(s *CharSpec) int {
	if s == nil {
		return fgNone
	}
	return beanKinds[s.Name]
}

func isBean(s *CharSpec) bool { return beanKind(s) != fgNone }

// beanRole: "Flan Guys" for a bean, "" for anybody else (goalchars.go puts it under the name).
func beanRole(slug string) string {
	if beanKinds[slug] != fgNone {
		return beanGame
	}
	return ""
}

// fgLook: the colours of a bean: body (with light and dark), feet, flippers and flipper tips.
type fgLook struct{ body, light, dark, foot, arm, tip color.RGBA }

var (
	fgWhite   = hx(0xf8f8fa)
	fgWhiteD  = hx(0xd8d8e4)
	fgEye     = hx(0x141418)
	fgBlue    = hx(0x3a6ad8)
	fgBlueD   = hx(0x2a4aa8)
	fgPink    = hx(0xf0508c)
	fgPinkL   = hx(0xf890b8)
	fgRed     = hx(0xd8283a)
	fgRedD    = hx(0xa81c2c)
	fgOrange  = hx(0xf08a1c)
	fgOrangeL = hx(0xf8b048)
	fgOrangeD = hx(0xc0600c)
	fgYellow  = hx(0xf8d040)
	fgGreen   = hx(0x3a9a3a)
	fgGreenL  = hx(0x6ac048)
	fgGreenD  = hx(0x2a6a2a)
	fgBlack   = hx(0x26202c)
	fgBlackL  = hx(0x3c3446)
	fgBun     = hx(0xd8782c)
	fgBunL    = hx(0xec9a48)
	fgSeed    = hx(0xf8eab0)
	fgPatty   = hx(0x6a3a1c)
	fgLettuce = hx(0x58b030)
	fgJeans   = hx(0x3c4466)
	fgJeansD  = hx(0x2a3050)
	fgBeak    = hx(0xf4b020)
	fgComb    = hx(0xd82828)
)

var fgLooks = map[int]fgLook{
	fgAzul:        {fgBlue, hx(0x5a8ef0), fgBlueD, hx(0x24408c), fgBlue, fgBlue},
	fgAnimadora:   {fgPink, fgPinkL, hx(0xc03068), fgBlueD, fgPink, fgWhite},
	fgLima:        {hx(0xa8d828), hx(0xc8f048), hx(0x6aa018), hx(0x4a8a18), hx(0xa8d828), hx(0xa8d828)},
	fgPelofuego:   {fgPink, fgPinkL, hx(0xc03068), hx(0xc03068), fgPink, fgPinkL},
	fgGallina:     {hx(0xc8ccd8), hx(0xe8eaf0), hx(0x8c90a0), fgYellow, hx(0xb4b8c6), hx(0xe8eaf0)},
	fgHamburguesa: {fgBlack, fgBlackL, hx(0x160e14), hx(0x1a1218), fgBlack, fgBlackL},
	fgZanahoria:   {fgOrange, fgOrangeL, fgOrangeD, fgGreenD, fgOrange, fgYellow},
	fgCallejera:   {fgRed, hx(0xf04858), fgRedD, hx(0x2a2a30), fgRed, hx(0xf0e0d0)},
	fgPinguino:    {hx(0x8a2a8c), hx(0xb04ab0), hx(0x5a1a5c), hx(0xf0a020), hx(0x8a2a8c), hx(0xb04ab0)},
}

// fgFaceRow: first row of the white face (it sits lower under hoods and hats).
var fgFaceRow = map[int]int{fgPelofuego: 4, fgGallina: 7, fgHamburguesa: 5, fgCallejera: 4, fgPinguino: 6}

const fgRows = 23 // rows of the body

func beanPreset(slug string, body, foot uint32, seed int64) func(*rand.Rand) CharSpec {
	return func(*rand.Rand) CharSpec {
		return def(slug, skinc(body), hairc(body, HairBald), hatc(HatNone, 0), glass(GlassNone), beardc(BeardNone),
			tops(TopTee, body, body), bottomc(BotPants, body), shoesc(foot), bodyc(BodyFat), stat(4), carryc(HoldNone)).F(rand.New(rand.NewSource(seed)))
	}
}

// beanNames: the names, the same in Spanish and English (proper names are never translated; owner, 2026-11-11).
var beanNames = map[string][2]string{
	"judia_azul":        {"Flan Azul", "Flan Azul"},
	"judia_animadora":   {"Flan Animador", "Flan Animador"},
	"judia_lima":        {"Flan Lima", "Flan Lima"},
	"judia_pelofuego":   {"Flan Pelofuego", "Flan Pelofuego"},
	"judia_gallina":     {"Flan Gallina", "Flan Gallina"},
	"judia_hamburguesa": {"Flan Hamburguesa", "Flan Hamburguesa"},
	"judia_zanahoria":   {"Flan Zanahoria", "Flan Zanahoria"},
	"judia_callejera":   {"Flan Callejero", "Flan Callejero"},
	"judia_pinguino":    {"Flan Pingüino", "Flan Pingüino"},
}

func init() {
	seeds := []int64{2020, 2021, 2022, 2023, 2024, 2025, 2026, 2027, 2028}
	bodies := []uint32{0x3a6ad8, 0xf0508c, 0xa8d828, 0xf0508c, 0xc8ccd8, 0x26202c, 0xf08a1c, 0xd8283a, 0x8a2a8c}
	feet := []uint32{0x24408c, 0x2a4aa8, 0x4a8a18, 0xc03068, 0xf8d040, 0x1a1218, 0x2a6a2a, 0x2a2a30, 0xf0a020}
	for i, slug := range beanSlugs {
		famousPresets = append(famousPresets, beanPreset(slug, bodies[i], feet[i], seeds[i]))
	}
	specialPool = append(specialPool, beanSlugs...) // their cards in OBJETIVOS / PERSONAJES, and rare single visits
	gaitDefs["bean"] = gaitDef{walk: AnimLumber, dance: AnimHop}
	for _, slug := range beanSlugs {
		if _, ok := gaitSlugs[slug]; !ok {
			gaitSlugs[slug] = "bean" // they waddle side to side and dance hopping
		}
	}
	descs := map[string][2]string{
		"judia_azul":        {"Se cae, se levanta y se vuelve a caer", "Falls, gets up and falls down again"},
		"judia_animadora":   {"Anima a todos aunque vaya el último", "Cheers everyone on, even in last place"},
		"judia_lima":        {"Verde fosforito: imposible perderlo de vista", "Neon green: impossible to lose sight of"},
		"judia_pelofuego":   {"Lleva el pelo en llamas y ni se inmuta", "Its hair is on fire and it doesn't care"},
		"judia_gallina":     {"Disfrazado de gallina, huye de todo", "Dressed as a chicken, runs from everything"},
		"judia_hamburguesa": {"Huele a parrilla y a victoria", "Smells of barbecue and victory"},
		"judia_zanahoria":   {"Viste de verdura y su madre está orgullosa", "Dresses as a vegetable; its mum is proud"},
		"judia_callejera":   {"Gorra, vaqueros y cero equilibrio", "Cap, jeans and zero balance"},
		"judia_pinguino":    {"Viene del frío y resbala en todas partes", "Came from the cold and slips everywhere"},
	}
	funny := map[string][2]string{
		"skin:judia_azul":        {"Tropieza con su propia sombra y lo llama estrategia.", "Trips over its own shadow and calls it strategy."},
		"skin:judia_animadora":   {"Hace la ola él solo en mitad del baile.", "Does the wave all alone in the middle of the dance."},
		"skin:judia_lima":        {"Brilla tanto que lo usan de lámpara en el jardín.", "Glows so much they use it as a garden lamp."},
		"skin:judia_pelofuego":   {"Enciende las velas de la tarta sin mechero.", "Lights the birthday candles without a lighter."},
		"skin:judia_gallina":     {"Cacarea cuando alguien saca el móvil.", "Clucks whenever someone takes out a phone."},
		"skin:judia_hamburguesa": {"Nadie sabe si viene a la fiesta o es el catering.", "Nobody knows if it's a guest or the catering."},
		"skin:judia_zanahoria":   {"Los conejos de la fiesta lo siguen a todas partes.", "The party rabbits follow it everywhere."},
		"skin:judia_callejera":   {"Se pone la gorra hacia atrás para correr más.", "Wears the cap backwards to run faster."},
		"skin:judia_pinguino":    {"Pide el ponche con mucho, mucho hielo.", "Asks for the punch with lots and lots of ice."},
	}
	lines := map[string][][2]string{
		"judia_azul":        {{"¡Uooooh!", "Whoooa!"}, {"¿Dónde está la corona?", "Where's the crown?"}},
		"judia_animadora":   {{"¡Vamos, vamos, flanes!", "Go, go, flans!"}, {"¡Dame una F!", "Give me an F!"}},
		"judia_lima":        {{"¡Que no me empujen!", "Stop pushing me!"}, {"¡Hoy sí que gano!", "Today I win for sure!"}},
		"judia_pelofuego":   {{"¡Estoy que ardo!", "I'm on fire!"}, {"¿Alguien huele a quemado?", "Anyone smell burning?"}},
		"judia_gallina":     {{"¡Coc, coc, cocoooc!", "Bawk, bawk, bawk!"}, {"¡Yo no he puesto ese huevo!", "I didn't lay that egg!"}},
		"judia_hamburguesa": {{"¿Con queso o sin queso?", "With or without cheese?"}, {"¡No me pongáis kétchup!", "No ketchup on me!"}},
		"judia_zanahoria":   {{"¡Soy muy bueno para la vista!", "I'm very good for your eyes!"}, {"¡Ni se te ocurra pelarme!", "Don't you dare peel me!"}},
		"judia_callejera":   {{"¡Me piro, vampiro!", "Later, alligator!"}, {"¿Una carrera de obstáculos?", "Obstacle race, anyone?"}},
		"judia_pinguino":    {{"¡Qué calor hace aquí!", "It's so hot in here!"}, {"¡Me resbalo! ¡Me resbalooo!", "I'm slipping! Slippiiing!"}},
	}
	add := func(es, en string) {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
	}
	for slug, n := range beanNames {
		displayNameOverrides[slug] = n[0]
		add(n[0], n[1])
	}
	for slug, d := range descs {
		if _, ok := charDescs[slug]; !ok {
			charDescs[slug] = d
		}
		add(d[0], d[1])
	}
	for k, d := range funny {
		if _, ok := funnyDesc[k]; !ok {
			funnyDesc[k] = d[0]
		}
		add(d[0], d[1])
	}
	for slug, ls := range lines {
		if _, ok := catchphrases[slug]; ok {
			continue
		}
		for _, l := range ls {
			catchphrases[slug] = append(catchphrases[slug], l[0])
			add(l[0], l[1])
		}
	}
	add(beanGame, beanGame)
}

// ---- the gang ----------------------------------------------------------------------------------

// beanGangChance: one party in this many gets a gang of 2-4 beans arriving together.
const beanGangChance = 5

var (
	fgRng   *rand.Rand
	fgForce bool // tests: the gang always comes
)

// fgDice: the gang's own generator for the whole session (the game's dice are not moved).
func fgDice() *rand.Rand {
	if fgRng == nil {
		fgRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return fgRng
}

// spawnBeanGang: now and then 2-4 different beans arrive together, the first one leading the others,
// and each one only if he is not at the party yet. It returns how many came, so spawnStars (stars.go)
// takes them from the famous places of the party instead of adding people.
func (g *Game) spawnBeanGang() int {
	if g.lvl == nil {
		return 0
	}
	d := fgDice()
	if !fgForce && d.Intn(beanGangChance) != 0 {
		return 0
	}
	var pool []string
	for _, s := range beanSlugs {
		if presetByName(s) != nil && allowedSlug(s) && !g.guestNamed(s) {
			pool = append(pool, s)
		}
	}
	d.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	n := minI(len(pool), 2+d.Intn(3))
	if n < 2 {
		return 0
	}
	base := g.lvl.RandomRoam(d)
	var leader *Guest
	for i, slug := range pool[:n] {
		spec := presetByName(slug)(d)
		pos := g.lvl.NearestFree(Vec{base.X + float64(i%2)*18 - 9, base.Y + float64(i/2)*16})
		gu := &Guest{pos: pos, dir: Dir(d.Intn(4)), cs: newCharSet(&spec), name: nameFromSpec(&spec), style: d.Intn(4), off: d.Float64() * 6, state: gIdle, timer: d.Float64() * 2, role: i % 2, keep: true}
		if leader == nil {
			leader = gu
		} else {
			gu.leader = leader
		}
		g.guests = append(g.guests, gu)
	}
	return n
}

// ---- the drawing -------------------------------------------------------------------------------

// fgHalf: half width of a body row (front and back are 12 px wide, the profile 10).
func fgHalf(vw, r int) int {
	if vw == vSide {
		switch r {
		case 0:
			return 3
		case 1, fgRows - 1:
			return 4
		}
		return 5
	}
	switch r {
	case 0:
		return 4
	case 1, fgRows - 1:
		return 5
	}
	return 6
}

// fgCostume: the colour of a body pixel (row r, column x of a row spanning x0..x1) for a costume.
func fgCostume(k, vw, r, x, x0, x1 int, lk fgLook) color.RGBA {
	col := lk.body
	switch {
	case r == 0:
		col = lk.light
	case x == x1 || r == fgRows-1:
		col = lk.dark
	case x == x0+1 && r >= 2 && r <= 10 && vw != vBack:
		col = lk.light
	}
	front := vw == vFront
	side := vw == vSide
	edge := x == x0 || x == x1
	switch k {
	case fgAnimadora: // headband, and a white and blue pleated skirt
		switch {
		case r == 2:
			col = fgWhite
		case r == 3:
			col = fgPinkL
		case r == 16 || r == 17:
			col = fgWhite
		case r == 18:
			col = fgBlue
		case r == 19:
			col = fgBlue
			if x%2 == 0 {
				col = fgWhite
			}
		}
	case fgPelofuego: // orange hair framing the face, red jersey
		hair := r <= 3 || (r <= 9 && ((front && (x <= x0+1 || x >= x1-1)) || (side && x <= x0+4))) || (vw == vBack && r <= 10)
		switch {
		case hair:
			col = fgOrange
			if r == 0 || (r <= 2 && x%3 == 0) {
				col = fgOrangeL
			}
			if x == x1 && r > 0 {
				col = fgOrangeD
			}
		case r >= 11 && r <= 20 && (side || (x > x0 && x < x1)):
			col = fgRed
			if x == x1-1 || r == 20 {
				col = fgRedD
			}
		}
	case fgGallina: // orange feathered bottom
		if r >= 17 && r <= 21 {
			col = fgOrange
			if r == 17 && x%2 == 0 {
				col = fgOrangeL
			}
			if x == x1 || r == 21 {
				col = fgOrangeD
			}
		}
	case fgHamburguesa: // bun, orange hair, lettuce, patty, bottom bun; the black cape around
		cape := (front && edge) || (side && x <= x0+2) || vw == vBack
		switch {
		case r <= 4:
			col = fgBun
			if r <= 1 {
				col = fgBunL
			}
			if (r == 1 && (x == 9 || x == 14)) || (r == 2 && (x == 11 || x == 16)) || (r == 3 && x == 8) {
				col = fgSeed
			}
			if r == 4 && front && x <= x0+2 {
				col = fgComb // a slice of tomato peeking out
			}
		case cape:
			// the cape keeps the body colours
		case r <= 11:
			col = fgOrange // the hair around the face
			if x == x1 || x == x1-1 {
				col = fgOrangeD
			}
		case r <= 13:
			col = fgLettuce
			if (x+r)%2 == 0 {
				col = fgGreenL
			}
		case r <= 15:
			col = fgPatty
		case r <= 18:
			col = fgBun
			if r == 18 {
				col = fgOrangeD
			}
		}
	case fgZanahoria: // ridges across the root and yellow-green bands at the bottom
		switch {
		case (r == 12 || r == 18) && (x <= x0+2 || x >= x1-2) && !edge:
			col = fgOrangeD
		case r == 15 && x >= x0+3 && x <= x0+5:
			col = fgOrangeD
		case r == 19 || r == 21:
			col = fgYellow
		case r == 20:
			col = fgGreenL
		}
	case fgCallejera: // black cap, dark hair round the face, red tee, belt and jeans
		switch {
		case r <= 3:
			col = fgBlack
			if r == 1 && x == x0+2 {
				col = fgBlackL
			}
		case r <= 10 && ((front && (x <= x0+1 || x >= x1-1)) || (side && x <= x0+4) || vw == vBack):
			col = fgBlack
			if r == 4 && (x == x0+1 || (side && x == x0+4)) {
				col = fgOrange // a lock of ginger hair under the cap
			}
		case r == 17:
			col = fgBlack // the belt
		case r >= 18:
			col = fgJeans
			if x == x1 || r == fgRows-1 {
				col = fgJeansD
			}
			if r >= 20 && x == (x0+x1)/2 && !side {
				col = fgJeansD // between the legs
			}
		}
	case fgPinguino: // pink belly in front
		if r >= 14 && r <= 20 && ((front && x >= x0+2 && x <= x1-2 && !(r == 14 && (x == x0+2 || x == x1-2))) || (side && x >= x1-2 && x < x1)) {
			col = fgPink
			if r == 20 {
				col = hx(0xc03068)
			}
		}
	}
	return col
}

func genFallBean(s *CharSpec, vw int, p Pose) *pixBuf {
	b := newPix(charW, charH)
	if trackParts {
		b.pid = make([]uint8, charW*charH)
	}
	k := beanKind(s)
	lk := fgLooks[k]
	rnd := func(f float64) int { return int(math.Floor(f + 0.5)) }
	bob := clampI(rnd(p.Bob), -1, 3)
	if p.Sit {
		bob = maxI(bob, 3)
	}
	tilt := clampF(p.Sway*0.6, -2, 2)
	if vw == vSide {
		tilt = clampF(p.Lean*0.5, -1, 2)
	}
	switch p.Anim { // the waddle: the body rocks from one side to the other
	case AnimWalk, AnimRun, AnimLumber, AnimStagger, AnimHop:
		if vw != vSide {
			tilt += []float64{0, 1.4, 0, -1.4}[(p.F%4+4)%4]
		}
	}
	tilt = clampF(tilt, -2, 2)
	top := 16 + bob
	// the rock bends the body in three bands, so the face (rows <= 13) always moves in one piece
	sh := func(r int) int {
		switch {
		case r <= 13:
			return rnd(tilt)
		case r <= 18:
			return rnd(tilt * 0.5)
		}
		return 0
	}
	lift := func(l Limb) int {
		if p.Sit {
			return 0
		}
		return clampI(rnd((10-l.Y)*0.5), 0, 2)
	}

	// ---- feet: short stubs under the body ----
	leg := func(cx float64, l Limb, sign float64, id uint8, dark bool) {
		b.cur = id
		col := lk.foot
		if dark {
			col = shade(col, 0.74)
		}
		foot := 41 - lift(l)
		var x int
		if vw == vSide {
			x = int(math.Floor(cx)) + clampI(rnd(l.Z*0.5), -2, 2)
		} else {
			x = int(math.Floor(cx)) + int(sign)*clampI(rnd((l.X-1.5)*0.6), -1, 1)
		}
		if p.Sit { // sitting: the feet stick out under the body
			foot = top + fgRows
			if vw == vSide {
				x += 3
			}
		}
		y0 := top + fgRows - 3
		foot = maxI(foot, top+fgRows)
		b.rect(x, y0, 3, foot-y0+1, col)
		b.rect(x, foot, 3, 1, shade(col, 0.78))
	}

	// ---- the body, the face and what sticks out of the costume ----
	body := func() {
		for r := 0; r < fgRows; r++ {
			b.cur = PartTorso
			if r <= 10 {
				b.cur = PartHead
			}
			hw := fgHalf(vw, r)
			x0, x1 := 12-hw, 11+hw
			for x := x0; x <= x1; x++ {
				b.set(x+sh(r), top+r, fgCostume(k, vw, r, x, x0, x1, lk))
			}
			if k == fgAnimadora && r >= 17 && r <= 19 { // the skirt flares out
				b.set(x0-1+sh(r), top+r, fgWhite)
				b.set(x1+1+sh(r), top+r, fgBlue)
			}
		}
		put := func(r, x int, c color.RGBA) {
			b.cur = PartHead
			if r > 10 {
				b.cur = PartTorso
			}
			b.set(x+sh(maxI(r, 0)), top+r, c)
		}
		// the face
		fy := 3
		if v, ok := fgFaceRow[k]; ok {
			fy = v
		}
		open := p.Open || p.Anim == AnimLaugh || p.Anim == AnimCheer
		switch vw {
		case vFront:
			for r := 0; r < 7; r++ {
				for x := 8; x <= 15; x++ {
					if (r == 0 || r == 6) && (x == 8 || x == 15) {
						continue
					}
					c := fgWhite
					if r == 6 || x == 15 {
						c = fgWhiteD
					}
					put(fy+r, x, c)
				}
			}
			for _, x := range []int{10, 13} {
				put(fy+2, x, fgEye)
				put(fy+3, x, fgEye)
			}
			if open {
				put(fy+5, 11, fgEye)
				put(fy+5, 12, fgEye)
			}
		case vSide:
			for r := 0; r < 7; r++ {
				for x := 13; x <= 16; x++ {
					if (r == 0 || r == 6) && x == 13 {
						continue
					}
					c := fgWhite
					if r == 6 {
						c = fgWhiteD
					}
					put(fy+r, x, c)
				}
			}
			put(fy+2, 15, fgEye)
			put(fy+3, 15, fgEye)
			if open {
				put(fy+5, 15, fgEye)
			}
		}
		// what sticks out
		switch k {
		case fgPelofuego: // flames of hair
			for i, x := range []int{8, 10, 12, 14} {
				hgt := []int{2, 3, 3, 2}[i]
				if vw == vSide {
					x = []int{8, 10, 12, 14}[i] - 1
				}
				for r := 1; r <= hgt; r++ {
					c := fgOrange
					if r == hgt {
						c = fgYellow
					}
					put(-r, x, c)
					put(-r, x+1, fgOrangeL)
				}
			}
		case fgGallina: // red comb on top; the chicken's eyes, beak and wattle above the face
			for _, c := range [][2]int{{-1, 10}, {-1, 11}, {-1, 12}, {-1, 13}, {-2, 11}, {-2, 12}, {-3, 12}, {-2, 10}, {-3, 10}} {
				put(c[0], c[1], fgComb)
			}
			switch vw {
			case vFront:
				put(3, 9, fgEye)
				put(3, 14, fgEye)
				put(4, 11, fgBeak)
				put(4, 12, fgBeak)
				put(5, 12, fgComb)
			case vSide:
				put(3, 15, fgEye)
				put(4, 17, fgBeak)
				put(4, 18, fgBeak)
				put(5, 16, fgComb)
			}
		case fgZanahoria: // the green tuft
			for _, c := range [][2]int{{-1, 11}, {-1, 12}, {-2, 10}, {-2, 11}, {-2, 12}, {-2, 13}, {-3, 10}, {-3, 13}} {
				col := fgGreen
				if c[0] == -3 {
					col = fgGreenL
				}
				put(c[0], c[1], col)
			}
		case fgCallejera: // the peak of the cap
			switch vw {
			case vFront:
				for x := 7; x <= 18; x++ {
					put(3, x, fgBlackL)
				}
			case vSide:
				for x := 16; x <= 19; x++ {
					put(3, x, fgBlackL)
				}
			}
		case fgPinguino: // the penguin's eyes and beak above the face
			switch vw {
			case vFront:
				for _, x := range []int{9, 10, 13, 14} {
					put(1, x, fgWhite)
					put(2, x, fgWhite)
				}
				put(2, 10, fgEye)
				put(2, 13, fgEye)
				put(3, 11, fgBeak)
				put(3, 12, fgBeak)
				put(4, 11, shade(fgBeak, 0.8))
				put(4, 12, shade(fgBeak, 0.8))
			case vSide:
				put(1, 14, fgWhite)
				put(2, 14, fgWhite)
				put(2, 15, fgWhite)
				put(1, 15, fgWhite)
				put(2, 15, fgEye)
				put(3, 17, fgBeak)
				put(4, 17, shade(fgBeak, 0.8))
			}
		}
	}

	// ---- flippers ----
	arm := func(sx float64, l Limb, sign float64, id uint8, dark bool) (int, int) {
		b.cur = id
		sx += float64(sh(9))
		sy := float64(top + 9)
		var tx, ty float64
		if vw == vSide {
			tx, ty = l.Z, l.Y
		} else {
			tx, ty = sign*(l.X-0.7)*0.8, l.Y+l.Z*0.1
		}
		n := math.Hypot(tx, ty)
		if n < 0.1 {
			tx, ty, n = 0, 1, 1
		}
		ex := clampF(sx+tx/n*7, 1, 20)
		ey := math.Max(sy+ty/n*7, 2)
		col, tip := lk.arm, lk.tip
		if dark {
			col, tip = shade(col, 0.74), shade(tip, 0.74)
		}
		// a flat flipper: 3x2 stamps along the line, so it keeps its thickness at any angle
		steps := 14
		for i := 0; i <= steps; i++ {
			f := float64(i) / float64(steps)
			c := col
			switch {
			case f >= 0.75:
				c = tip
			case f >= 0.35:
				c = shade(col, 0.9)
			}
			x, y := int(math.Floor(sx+(ex-sx)*f)), int(math.Floor(sy+(ey-sy)*f))
			b.rect(x, y-1, 3, 2, c)
		}
		return int(math.Floor(ex)) + 1, int(math.Floor(ey))
	}
	held := func(hx0, hy0 int, id uint8) {
		b.cur = id
		kind := p.Hold
		if kind == HoldNone {
			kind = s.Carry
		}
		if s.Weapon != 0 && (kind == HoldKnife || kind == HoldNone) {
			kind = s.Weapon
		}
		if kind != HoldNone {
			drawHeld(b, kind, s.CarryCol, vw, float64(hx0), float64(hy0))
		}
	}

	switch vw {
	case vSide:
		arm(10, p.LA, 1, PartArmL, true)
		leg(10, p.LF, 1, PartLegL, true)
		leg(11, p.RF, 1, PartLegR, false)
		body()
		hx0, hy0 := arm(10, p.RA, 1, PartArmR, false)
		held(hx0, hy0, PartArmR)
	case vBack:
		leg(8, p.RF, -1, PartLegL, false) // seen from behind his right foot is on the screen left
		leg(13, p.LF, 1, PartLegR, false)
		body()
		arm(18, p.LA, 1, PartArmR, false)
		hx0, hy0 := arm(3, p.RA, -1, PartArmL, false)
		held(hx0, hy0, PartArmL)
	default:
		leg(8, p.LF, -1, PartLegL, false)
		leg(13, p.RF, 1, PartLegR, false)
		body()
		arm(3, p.LA, -1, PartArmL, false)
		hx0, hy0 := arm(18, p.RA, 1, PartArmR, false)
		held(hx0, hy0, PartArmR)
	}
	return b
}
