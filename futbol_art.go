package main

import (
	"image/color"
	"math"
	"math/rand"
)

// ---------------------------------------------------------------------------
// Four famous footballers as party guests (owner's decision, 2026-11-10: musicians and
// footballers are shown by their real name; the other real people keep no visible name):
//   - cristiano_ronaldo: red Portugal shirt with green trim, green shorts, red socks, the 7 on
//     the back, short dark hair swept up, athletic build; he dances jumping and turning (gait "spin").
//   - lionel_messi: sky-blue and white striped Argentina shirt, black shorts, white socks, the 10
//     on the back, short brown hair and beard, shorter than the others.
//   - lamine_yamal: red Spain shirt with yellow trim, navy shorts, the 19 on the back, young and
//     slim, short dark curly hair, brown skin.
//   - kylian_mbappe: navy blue France shirt, white shorts, red socks, the 10 on the back, a very
//     short buzz cut, brown skin.
// Friendly pixel-art: the generic body plus own layers (fbHead / fbTorso / fbLeg, called by genChar
// next to the other own layers). Only the drawing changes: hitbox, stats and the one-per-party rule
// are the usual ones. Every preset uses a fixed random source of its own, so the shared random
// stream of builtinChars (export.go) is not moved for the characters after them.
// ---------------------------------------------------------------------------

const (
	fbNone = iota
	fbCR7
	fbMessi
	fbLamine
	fbMbappe
)

// footballSlugs: the four footballers, in album order.
var footballSlugs = []string{"cristiano_ronaldo", "lionel_messi", "lamine_yamal", "kylian_mbappe"}

var fbKinds = map[string]int{"cristiano_ronaldo": fbCR7, "lionel_messi": fbMessi, "lamine_yamal": fbLamine, "kylian_mbappe": fbMbappe}

func fbKind(s *CharSpec) int {
	if s == nil {
		return fbNone
	}
	return fbKinds[s.Name]
}

// footballer: a real footballer shown by his name, with his national team (a card word, translated).
type footballer struct{ name, team string }

// footballRealNames: the footballers shown by their real name (card: name / team; detail: team · Futbolista).
var footballRealNames = map[string]footballer{
	"cristiano_ronaldo": {"Cristiano Ronalda", "Portugal"},
	"lionel_messi":      {"Lionel Mesi", "Argentina"},
	"lamine_yamal":      {"Lamine Yamel", "España"},
	"kylian_mbappe":     {"Kylian Mbapé", "Francia"},
}

// footballTeam: the national team of a footballer shown by his name ("" for the others).
func footballTeam(slug string) string { return footballRealNames[slug].team }

var (
	// Portugal
	fbPorRed   = hx(0xc8182c)
	fbPorGreen = hx(0x14683a)
	fbPorGold  = hx(0xf0cc48)
	// Argentina
	fbArgWht  = hx(0xf6f8fa)
	fbArgNavy = hx(0x1a2a52)
	fbArgGold = hx(0xe8c040)
	// Spain
	fbEspYel  = hx(0xf4c82c)
	fbEspNavy = hx(0x1c2860)
	// France
	fbFraBlue = hx(0x1e3070)
	fbFraWht  = hx(0xf4f4f6)
	fbFraRed  = hx(0xd0222a)
)

func presetCR7(*rand.Rand) CharSpec {
	return def("cristiano_ronaldo", skinc(0xdcae86), hairc(0x24180f, HairShort), hatc(HatNone, 0), glass(GlassNone), beardc(BeardNone),
		tops(TopTee, 0xc8182c, 0x14683a), bottomc(BotShorts, 0x14683a), shoesc(0xf4f4f4), bodyc(BodyBig), stat(-1), carryc(HoldNone)).F(rand.New(rand.NewSource(1985)))
}

func presetMessi(*rand.Rand) CharSpec {
	return def("lionel_messi", skinc(0xf0c8a2), hairc(0x5a3a20, HairSide), hatc(HatNone, 0), glass(GlassNone), beardc(BeardNone),
		tops(TopTee, 0x74b4e4, 0xf6f8fa), bottomc(BotShorts, 0x1a1a22), shoesc(0x2a2a34), bodyc(BodyBig), stat(2), carryc(HoldNone)).F(rand.New(rand.NewSource(1987)))
}

func presetLamine(*rand.Rand) CharSpec {
	return def("lamine_yamal", skinc(0x9c6a44), hairc(0x16100c, HairShort), hatc(HatNone, 0), glass(GlassNone), beardc(BeardNone),
		tops(TopTee, 0xc41e26, 0xf4c82c), bottomc(BotShorts, 0x1c2860), shoesc(0xf0d040), bodyc(BodyBig), stat(1), carryc(HoldNone)).F(rand.New(rand.NewSource(2007)))
}

func presetMbappe(*rand.Rand) CharSpec {
	return def("kylian_mbappe", skinc(0x7c4e30), hairc(0x120e0c, HairBald), hatc(HatNone, 0), glass(GlassNone), beardc(BeardNone),
		tops(TopTee, 0x1e3070, 0xd0222a), bottomc(BotShorts, 0xf4f4f6), shoesc(0x2a2a30), bodyc(BodyBig), stat(0), carryc(HoldNone)).F(rand.New(rand.NewSource(1998)))
}

func init() {
	famousPresets = append(famousPresets, presetCR7, presetMessi, presetLamine, presetMbappe)
	specialPool = append(specialPool, footballSlugs...) // their cards in OBJETIVOS / PERSONAJES
	for slug, f := range footballRealNames {
		displayNameOverrides[slug] = f.name
	}
	if _, ok := gaitSlugs["cristiano_ronaldo"]; !ok {
		gaitSlugs["cristiano_ronaldo"] = "spin" // he dances jumping and turning, like his goal celebration
	}
	descs := map[string][2]string{
		"cristiano_ronaldo": {"Salta, gira y grita ¡SIIUU! hasta en el ponche", "Jumps, spins and yells SIIUU! even at the punch"},
		"lionel_messi":      {"Regatea a los camareros sin derramar nada", "Dribbles past the waiters without spilling"},
		"lamine_yamal":      {"Tan joven que pide permiso para quedarse", "So young he asks permission to stay up"},
		"kylian_mbappe":     {"Llega el primero a la barra, siempre", "Always first to the bar, every time"},
	}
	funny := map[string][2]string{
		"skin:cristiano_ronaldo": {"Ensaya el salto de celebración delante del espejo del baño.", "Rehearses his celebration jump in the bathroom mirror."},
		"skin:lionel_messi":      {"Pasa entre la gente como si fueran defensas.", "Slips through the crowd as if they were defenders."},
		"skin:lamine_yamal":      {"Hace los deberes en el descanso de la fiesta.", "Does his homework at the party's half-time."},
		"skin:kylian_mbappe":     {"Corre tanto que ya ha vuelto antes de irse.", "Runs so fast he is back before he left."},
	}
	lines := map[string][][2]string{
		"cristiano_ronaldo": {{"¡SIIUU!", "SIIUU!"}, {"¿Habéis visto mis abdominales?", "Have you seen my abs?"}, {"Agua, por favor. Nada de refrescos.", "Water, please. No fizzy drinks."}},
		"lionel_messi":      {{"Perdón, solo pasaba...", "Sorry, just passing..."}, {"¿Me pasás la pelota?", "Pass me the ball?"}},
		"lamine_yamal":      {{"Mañana tengo clase...", "I've got school tomorrow..."}, {"¡Regate por la banda!", "Dribble down the wing!"}},
		"kylian_mbappe":     {{"¡Allez!", "Allez!"}, {"¿Una carrera hasta la barra?", "Race you to the bar?"}},
	}
	add := func(es, en string) {
		if _, ok := langDataEN[es]; !ok {
			langDataEN[es] = en
		}
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
	for es, en := range map[string]string{"Futbolista": "Footballer", "Portugal": "Portugal", "Argentina": "Argentina", "España": "Spain", "Francia": "France"} {
		add(es, en)
	}
}

func fbDark(c color.RGBA, dark bool) color.RGBA {
	if dark {
		return shade(c, 0.78)
	}
	return c
}

// fbDigits: a 3x5 pixel font for the shirt numbers ('1' is one pixel wide).
var fbDigits = map[byte][]string{
	'0': {"###", "#.#", "#.#", "#.#", "###"},
	'1': {"#", "#", "#", "#", "#"},
	'7': {"###", "..#", ".#.", ".#.", ".#."},
	'9': {"###", "#.#", "###", "..#", "###"},
}

// fbNumberW: width in pixels of a shirt number (one pixel between digits).
func fbNumberW(num string) int {
	w := 0
	for i := 0; i < len(num); i++ {
		if i > 0 {
			w++
		}
		w += len(fbDigits[num[i]][0])
	}
	return w
}

// fbNumber paints the shirt number centred on x=mid (rounded left), top row y.
func fbNumber(b *pixBuf, num string, mid, y int, col color.RGBA) {
	x := mid - fbNumberW(num)/2
	for i := 0; i < len(num); i++ {
		g := fbDigits[num[i]]
		for r, row := range g {
			for c := 0; c < len(row); c++ {
				if row[c] == '#' {
					b.set(x+c, y+r, col)
				}
			}
		}
		x += len(g[0]) + 1
	}
}

// fbKit: shirt number, number colour, trim colour, socks colour.
func fbKit(k int) (string, color.RGBA, color.RGBA, color.RGBA) {
	switch k {
	case fbCR7:
		return "7", fbPorGold, fbPorGreen, fbPorRed
	case fbMessi:
		return "10", fbArgNavy, fbArgWht, fbArgWht
	case fbLamine:
		return "19", fbEspYel, fbEspYel, fbEspNavy
	case fbMbappe:
		return "10", fbFraWht, fbFraRed, fbFraRed
	}
	return "", color.RGBA{}, color.RGBA{}, color.RGBA{}
}

// fbTorso: the national shirts: stripes, collar trim, a small crest and the number on the back.
func fbTorso(b *pixBuf, s *CharSpec, vw, x0, x1, y0, y1, mid int, p Pose) {
	k := fbKind(s)
	if k == fbNone {
		return
	}
	b.cur = 2
	num, ncol, trim, _ := fbKit(k)
	front, side, back := vw == vFront, vw == vSide, vw == vBack
	if k == fbMessi { // sky-blue and white vertical stripes, over the whole shirt
		for y := y0; y <= y1; y++ {
			for x := x0 - 1; x <= x1; x++ {
				if c := b.get(x, y); c.A != 0 && int(c.B) > int(c.R)+50 && (x-x0)%2 == 1 { // the shaded sky blue
					b.set(x, y, fbArgWht)
				}
			}
		}
	}
	switch {
	case front:
		b.rect(mid-1, y0, 3, 1, trim) // the round collar
		b.set(mid, y0, s.Skin)
		crest := fbPorGold
		switch k {
		case fbMessi:
			crest = fbArgGold
		case fbLamine:
			crest = fbEspYel
		case fbMbappe:
			crest = fbFraWht
		}
		b.set(x1-2, y0+2, crest) // the crest over the heart
	case side:
		b.set(x1-1, y0, trim)
	case back:
		b.rect(x0+1, y0, x1-x0-2, 1, trim) // the back of the collar
		fbNumber(b, num, mid, y0+2, ncol)
	}
}

// fbLeg: the football socks from the knee down, with a band at the top.
func fbLeg(b *pixBuf, s *CharSpec, vw int, mx, my, ex, ey float64, legW int, dark bool) {
	k := fbKind(s)
	if k == fbNone {
		return
	}
	_, _, trim, sock := fbKit(k)
	thickLine(b, mx, my+1, ex, ey-1, legW, fbDark(sock, dark)) // the boot keeps its rows
	band := trim
	if k == fbMessi || k == fbMbappe {
		band = fbArgNavy
		if k == fbMbappe {
			band = fbFraBlue
		}
	}
	b.rect(int(math.Floor(mx)), int(math.Floor(my))+1, legW, 1, fbDark(band, dark))
}

// fbHead: Cristiano's swept-up quiff, Messi's beard, Lamine's curls, Mbappé's buzz cut.
func fbHead(b *pixBuf, s *CharSpec, vw int, p Pose, hxp, hyp, rx, ry float64, ey int) {
	k := fbKind(s)
	if k == fbNone {
		return
	}
	b.cur = 1
	ihx := int(math.Floor(hxp))
	top := int(math.Floor(hyp - ry))
	rxi := int(math.Floor(rx))
	hairL := shade(s.Hair, 1.9)
	switch k {
	case fbCR7: // the stock tuft (its tip is on row top) filled out into a short swept-up quiff, a highlight
		switch vw {
		case vFront, vBack:
			b.rect(ihx-1, top, 3, 2, s.Hair)
			b.rect(ihx-1, top-1, 2, 1, s.Hair)
			if vw == vFront {
				b.set(ihx, top, hairL)
			}
		case vSide:
			b.rect(ihx-1, top, rxi+2, 2, s.Hair)
			b.rect(ihx, top-1, rxi, 1, s.Hair)
			b.set(ihx+rxi-1, top, hairL)
		}
	case fbMessi: // a short trimmed beard on the jaw that leaves the cheeks and the eyes clear
		beard := shade(s.Hair, 0.9)
		switch vw {
		case vFront:
			b.set(ihx-rxi, ey+1, beard)
			b.set(ihx+rxi, ey+1, beard)
			b.rect(ihx-rxi, ey+2, 2*rxi+1, 1, beard)
			b.rect(ihx-rxi+1, ey+3, 2*rxi-1, 1, beard)
			b.set(ihx, ey+2, shade(s.Skin, 0.6)) // the mouth
		case vSide:
			b.rect(ihx-1, ey+1, 1, 1, beard)
			b.rect(ihx-1, ey+2, rxi+2, 2, beard)
		}
	case fbLamine: // short dark curls: the tuft rounded into a mop with lighter twists
		b.rect(ihx-rxi, top, 2*rxi+1, 2, s.Hair)
		b.rect(ihx-rxi+1, top-1, 2*rxi-1, 1, s.Hair)
		if vw != vBack {
			for x := ihx - rxi + 1; x <= ihx+rxi-1; x += 2 {
				b.set(x, top, hairL)
			}
		}
	case fbMbappe: // a buzz cut: a thin dark cap over the skull
		switch vw {
		case vFront:
			b.rect(ihx-rxi, top, 2*rxi+1, 2, s.Hair)
			b.set(ihx-rxi, top+2, s.Hair)
			b.set(ihx+rxi, top+2, s.Hair)
		case vSide:
			b.rect(ihx-rxi, top, 2*rxi+1, 2, s.Hair)
			b.rect(ihx-rxi, top+2, rxi, 2, s.Hair)
		case vBack:
			b.rect(ihx-rxi, top, 2*rxi+1, ey-top+1, s.Hair)
		}
	}
}
