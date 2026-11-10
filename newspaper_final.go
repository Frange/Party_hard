package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// The secret ending's LAST NEWSPAPER: after the T-Rex eats the cake (drawStinger) and before the scene at home
// and the FIN card. A special edition of the end-of-level paper (same engine: newspaper_canvas.go, the masthead,
// the headline, the footer banner and the wear of newspaper_wear.go), with the FAMILY PHOTO taken during the
// party (endparty.go keeps a copy of the frame just before the flash) printed in the sepia tones of the paper's
// photos, and joke news about the end of the parties. The page is composed ONCE per ending and never changes
// while it is on screen. It is skipped like the other papers (a button, a click, Enter).
// ---------------------------------------------------------------------------

// endPaper is a stage of the ending (ending.go): "-ending 2 -endstage 12" shows the newspaper on its own.
const endPaper = 12

const (
	finalPaperLen  = 60.0 // it waits for the player, but an unattended ending still reaches FIN
	finalPaperLock = 0.8  // the press that skipped the stinger does not also skip the newspaper
	fpPhotoW       = pcCW // the photo takes the whole centre column...
	fpPhotoH       = 276  // ...and nearly all its height (the caption goes under it)
)

type finalPaperState struct {
	seed  int64
	lang  string
	two   bool
	d     *paperData
	photo *image.RGBA // the party photo read back from the GPU (nil: the drawn fallback)
	took  bool        // the photo has been looked for (once per ending)
	tex   [2]*ebiten.Image
	lay   *paperLayout
}

// ---- the texts (ES -> EN) ----------------------------------------------------------------------------------

var fpTexts = [][2]string{
	{"EDICIÓN ESPECIAL", "SPECIAL EDITION"},
	{"Domingo, de mañana", "Sunday morning"},
	{"PRECIO: 1 tarta", "PRICE: 1 cake"},
	{"(ya no queda)", "(none left)"},
	{"Domingo, el día después", "Sunday, the day after"},
	{"La última edición: ya no quedan fiestas que contar", "The last edition: there are no parties left to tell"},
	{"Páginas: 1 (la última)", "Pages: 1 (the last one)"},
	{"FIN DE LAS FIESTAS", "NO MORE PARTIES"},
	{"Edición especial: la fiesta final, un T-Rex sin hambre y un barrio que por fin duerme", "Special edition: the final party, a T-Rex who is full and a neighbourhood that finally sleeps"},
	{"Foto de familia: invitados, vecinos y dinosaurios, felices. Nadie fue devorado (casi).", "Family photo: guests, neighbours and dinosaurs, all happy. Nobody got eaten (almost)."},
	{"La última fiesta terminó anoche en la isla, con todos los invitados celebrándolo con sombrero.", "The last party ended last night on the island, with every guest celebrating in a party hat."},
	{"Dinosaurios y vecinos posaron juntos para la foto de familia.", "Dinosaurs and neighbours posed together for the family photo."},
	{"Después, el T-Rex se comió la tarta y todos se fueron a dormir.", "Then the T-Rex ate the cake and everybody went to bed."},
	{"Fuentes oficiales lo confirman: no habrá más fiestas. De momento.", "Official sources confirm it: there will be no more parties. For now."},
	{"EL T-REX, EN EXCLUSIVA", "THE T-REX, EXCLUSIVE"},
	{"«¡RAWR!» (Traducción: «Gracias por la tarta, estaba buenísima»)", "\"RAWR!\" (Translation: \"Thanks for the cake, it was delicious\")"},
	{"- el T-Rex, feliz y lleno", "- the T-Rex, happy and full"},
	{"ÚLTIMA EDICIÓN", "LAST EDITION"},
	{"CRÓNICA DE LA NOCHE", "THE NIGHT, REPORTED"},
	{"Próxima edición: cuando el T-Rex haga la digestión", "Next edition: when the T-Rex has digested"},
	{"Impreso en silencio, para no despertar a los vecinos", "Printed in silence, so as not to wake the neighbours"},
}

// fpBriefs: the joke news of the special edition. Every head is different (a page never repeats one).
var fpBriefs = []paperBrief{
	{"DINOSAURIOS", "El T-Rex se zampa la tarta de cinco pisos de un bocado. Su veredicto: «Un poco seca».",
		"The T-Rex wolfs down the five-tier cake in one bite. His verdict: \"A bit dry.\""},
	{"BAILE", "Tres raptores ganan el concurso de baile. El jurado no se atrevió a votar a nadie más.",
		"Three raptors win the dance contest. The jury did not dare vote for anyone else."},
	{"RUIDO", "Silencio absoluto en el barrio por primera vez en años. Los sismógrafos se aburren.",
		"Total silence in the neighbourhood for the first time in years. Seismographs are bored."},
	{"VECINDARIO", "Los vecinos, felices: duermen ocho horas seguidas y ya no recordaban cómo se hacía.",
		"The neighbours are happy: they slept eight hours straight and had forgotten how."},
	{"SUCESOS", "Desaparece una tarta. El sospechoso tiene glaseado en los dientes. Dientes de un palmo.",
		"A cake goes missing. The suspect has icing on his teeth. Foot-long teeth."},
	{"TRÁFICO AÉREO", "El helicóptero de alquiler, quemado. La empresa pregunta si alguien guardó el ticket.",
		"The rented helicopter, burnt to a crisp. The company asks if anyone kept the receipt."},
	{"ECONOMÍA", "El alquiler de este mes sigue sin pagar. El casero vio al T-Rex y ya no lo reclama.",
		"This month's rent is still unpaid. The landlord saw the T-Rex and stopped asking."},
	{"SOCIEDAD", "Se acabaron las fiestas. El ayuntamiento barrerá el confeti antes de 2031, promete.",
		"The parties are over. The council promises to sweep up the confetti before 2031."},
	{"ENTREVISTA", "«Nunca había sido tan feliz», dice el T-Rex, que hoy ha comido algo que no gritaba.",
		"\"Never been happier,\" says the T-Rex, who today ate something that did not scream."},
	{"POLICÍA", "La policía cierra el caso: «Nadie oyó nada. Literalmente: ya no queda ruido».",
		"Police close the case: \"Nobody heard a thing. Literally: there is no noise left.\""},
	{"EL TIEMPO", "Cielo despejado, chubascos de confeti residual y algún rugido aislado al norte.",
		"Clear skies, showers of leftover confetti and the odd scattered roar up north."},
	{"ANUNCIOS", "Se vende bola de discoteca, poco uso. Abstenerse dinosaurios: ya rompieron tres.",
		"Disco ball for sale, barely used. No dinosaurs please: they broke three already."},
	{"DEPORTES", "La conga más larga de la historia acaba tras cuatro kilómetros. El cerdo llegó primero.",
		"The longest conga in history ends after four kilometres. The pig came first."},
}

func init() {
	add := func(k, v string) {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	for _, p := range fpTexts {
		add(p[0], p[1])
	}
	heads := map[string]string{"DINOSAURIOS": "DINOSAURS", "BAILE": "DANCE", "RUIDO": "NOISE", "VECINDARIO": "NEIGHBOURHOOD",
		"SUCESOS": "INCIDENTS", "TRÁFICO AÉREO": "AIR TRAFFIC", "ECONOMÍA": "ECONOMY", "SOCIEDAD": "SOCIETY",
		"ENTREVISTA": "INTERVIEW", "POLICÍA": "POLICE", "EL TIEMPO": "WEATHER", "ANUNCIOS": "SMALL ADS", "DEPORTES": "SPORTS"}
	for _, b := range fpBriefs {
		add(b.head, heads[b.head])
		add(b.es, b.en)
	}
}

// finalPaperData: the special edition, translated. The seed shuffles which news go first (a different mix each
// time the ending is seen); the page is then frozen.
func finalPaperData(seed int64, two bool) *paperData {
	d := &paperData{seed: seed, two: two}
	d.title = tr("EL DIARIO DE LA JUERGA")
	d.earL1, d.earL2 = tr("EDICIÓN ESPECIAL"), tr("Domingo, de mañana")
	d.earR1, d.earR2 = tr("PRECIO: 1 tarta"), tr("(ya no queda)")
	d.date = tr("Domingo, el día después")
	d.motto = tr("La última edición: ya no quedan fiestas que contar")
	d.dateR = tr("Páginas: 1 (la última)")
	d.head = tr("FIN DE LAS FIESTAS")
	d.dek = tr("Edición especial: la fiesta final, un T-Rex sin hambre y un barrio que por fin duerme")
	d.caption = tr("Foto de familia: invitados, vecinos y dinosaurios, felices. Nadie fue devorado (casi).")
	d.storyHead = tr("CRÓNICA DE LA NOCHE")
	for _, s := range []string{
		"La última fiesta terminó anoche en la isla, con todos los invitados celebrándolo con sombrero.",
		"Dinosaurios y vecinos posaron juntos para la foto de familia.",
		"Después, el T-Rex se comió la tarta y todos se fueron a dormir.",
		"Fuentes oficiales lo confirman: no habrá más fiestas. De momento.",
	} {
		d.story = append(d.story, tr(s))
	}
	d.quoteHead, d.quote, d.quoteBy = tr("EL T-REX, EN EXCLUSIVA"), tr("«¡RAWR!» (Traducción: «Gracias por la tarta, estaba buenísima»)"), tr("- el T-Rex, feliz y lleno")
	d.briefs = paperTplFinalBriefs(seed) // two template news of the farewell first (newspaper_tpl.go), then the joke news
	for _, i := range rand.New(rand.NewSource(seed)).Perm(len(fpBriefs)) {
		b := fpBriefs[i]
		d.briefs = append(d.briefs, paperBrief{head: tr(b.head), es: tr(b.es)})
	}
	d.hint, d.hint2 = tr("ÚLTIMA EDICIÓN"), tr("Próxima edición: cuando el T-Rex haga la digestión")
	d.foot = tr("Impreso en silencio, para no despertar a los vecinos")
	d.banner = tr("PULSA PARA CONTINUAR")
	return d
}

// ---- the page ----------------------------------------------------------------------------------------------

// renderFinalPaper paints the special edition (photo: the party picture, nil for the drawn fallback).
func renderFinalPaper(d *paperData, photo *image.RGBA, state int) (*image.RGBA, *paperLayout) {
	lay := &paperLayout{}
	c := newPaperCanvas(d.seed, lay)
	r := &prender{c: c, d: d, used: make([]bool, len(d.briefs))}
	r.masthead()
	r.headline()
	r.finalBody(photo)
	r.footer(state)
	if !paperNoWear {
		c.wear(d.seed, 0) // worn, stained and torn like every paper, but no blood: nobody died at this party
	}
	return c.im, lay
}

// finalBody: left, the report and small news; centre, the big family photo and its caption; right, the T-Rex's
// words and more small news. The columns are tiled by blocks (like the level paper).
func (r *prender) finalBody(photo *image.RGBA) {
	c, d := r.c, r.d
	c.fill(pcLX+pcLW+5, pbY0, 1, pbY1-pbY0, paperInk)
	c.fill(pcCX+pcCW+4, pbY0, 1, pbY1-pbY0, paperInk)
	H := pbY1 - pbY0
	storyL := wrapPx(joinSent(d.story), pcLW-10)
	story := pbox{"story", 4 + pRowH + len(storyL)*pRowH + 5, func(x, y, w, h int) {
		r.c.bold(d.storyHead, x+5, y+4, paperRed)
		for k, l := range storyL {
			r.c.text(l, x+5, y+4+pRowH+k*pRowH, 1, paperInk)
		}
	}}
	quoteL := wrapPx(d.quote, pcRW-10)
	quote := pbox{"quote", 4 + pRowH + len(quoteL)*pRowH + pRowH + 5, func(x, y, w, h int) {
		c := r.c
		c.shade(x, y, w, h, -9)
		c.bold(d.quoteHead, x+5, y+4, paperInk)
		for k, l := range quoteL {
			c.text(l, x+5, y+4+pRowH+k*pRowH, 1, paperRed)
		}
		c.textR(d.quoteBy, x+w-6, y+4+pRowH+len(quoteL)*pRowH, 1, paperInk)
	}}
	left := append([]pbox{story}, r.takeBriefs(pcLW, H-story.h)...)
	right := append([]pbox{quote}, r.takeBriefs(pcRW, H-quote.h)...)
	photoBox := pbox{"photo", H, func(x, y, w, h int) {
		c.partyPhoto(x, y+1, fpPhotoW, fpPhotoH, photo, d.two, d.seed)
		for k, l := range wrapPx(d.caption, fpPhotoW) {
			c.text(l, x, y+1+fpPhotoH+6+k*pRowH, 1, paperInk)
		}
	}}
	r.stack(pcLX, pcLW, pbY0, H, left)
	r.stack(pcCX, pcCW, pbY0, H, []pbox{photoBox})
	r.stack(pcRX, pcRW, pbY0, H, right)
}

func joinSent(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}

// partyPhoto prints the family photo of the party in the four sepia tones of the paper's photos: the picture is
// fitted to the width (the top of the sky is cropped), its levels stretched (it was taken at night), then grain
// and an ordered dither, so it reads like the other photos of the paper. No picture: a drawn one.
func (c *pcv) partyPhoto(x, y, w, h int, src *image.RGBA, two bool, seed int64) {
	c.clip = image.Rect(x, y, x+w, y+h)
	if src == nil {
		c.partyPhotoDrawn(x, y, w, h, two, seed)
	} else {
		sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
		s := float64(sw) / float64(w)
		yOff := math.Max(0, float64(sh)-float64(h)*s)
		lum := func(xx, yy int) float64 {
			sx := minI(sw-1, int((float64(xx-x)+0.5)*s))
			sy := minI(sh-1, int(yOff+(float64(yy-y)+0.5)*s))
			p := src.Pix[sy*src.Stride+sx*4:]
			return (0.3*float64(p[0]) + 0.59*float64(p[1]) + 0.11*float64(p[2])) / 255
		}
		var hist [256]int // auto levels: the darkest 3 % and the brightest 2 % clip
		for yy := y; yy < y+h; yy++ {
			for xx := x; xx < x+w; xx++ {
				hist[int(lum(xx, yy)*255)]++
			}
		}
		lo, hi, acc, n := 0, 255, 0, w*h
		for i := 0; i < 256; i++ {
			if acc += hist[i]; acc > n*3/100 {
				lo = i
				break
			}
		}
		acc = 0
		for i := 255; i >= 0; i-- {
			if acc += hist[i]; acc > n*2/100 {
				hi = i
				break
			}
		}
		if hi-lo < 24 {
			hi = minI(255, lo+24)
		}
		for yy := y; yy < y+h; yy++ {
			for xx := x; xx < x+w; xx++ {
				l := clampF((lum(xx, yy)*255-float64(lo))/float64(hi-lo), 0, 1)
				l = math.Pow(l, 0.8)
				l += (float64(paperHash(xx, yy, uint64(seed)+99)&15) - 7.5) / 120 // the grain of the print
				v := clampF(l, 0, 0.999) * 3
				k := int(v)
				col := photoTone[k]
				if dither(v-float64(k), xx, yy) {
					col = photoTone[k+1]
				}
				c.put(xx, yy, col)
			}
		}
	}
	c.clip = image.Rectangle{}
	c.frame(x, y, w, h, paperInk)
	c.frame(x-1, y-1, w+2, h+2, paperShade)
}

// partyPhotoDrawn: the fallback picture (no frame to read, e.g. in the tests): the island at night, bunting,
// fireworks, the T-Rex in a party hat and a row of guests with their arms up.
func (c *pcv) partyPhotoDrawn(x, y, w, h int, two bool, seed int64) {
	t := photoTone
	gy := y + h*2/3
	for yy := y; yy < gy; yy++ {
		c.dfill(x, yy, w, 1, 0.15+0.5*float64(yy-y)/float64(gy-y), t[0], t[1])
	}
	c.dfill(x, gy, w, y+h-gy, 0.4, t[1], t[2])
	for k := 0; k < 4; k++ { // fireworks
		fx, fy := x+40+k*w/4+int(paperHash(k, 1, uint64(seed))%30), y+24+int(paperHash(k, 2, uint64(seed))%30)
		for a := 0; a < 10; a++ {
			ang := float64(a) * math.Pi / 5
			for r := 4; r < 16; r += 3 {
				c.fill(fx+int(math.Cos(ang)*float64(r)), fy+int(math.Sin(ang)*float64(r)), 2, 2, t[3])
			}
		}
	}
	for xx := x; xx < x+w; xx += 14 { // bunting
		c.tri(xx, y+6, xx+10, y+6, xx+5, y+14, t[2])
	}
	// the T-Rex, with a party hat
	rx := x + w - 150
	c.ellipse(rx+60, gy-40, 46, 30, t[0])
	c.ellipse(rx+100, gy-86, 28, 18, t[0])
	c.fill(rx+40, gy-20, 14, 30, t[0])
	c.fill(rx+76, gy-20, 14, 30, t[0])
	c.tri(rx+18, gy-50, rx+18, gy-30, rx-30, gy-10, t[0])
	c.fill(rx+108, gy-90, 4, 4, t[3])
	c.tri(rx+92, gy-102, rx+108, gy-102, rx+100, gy-126, t[2])
	n := 12
	if two {
		n = 13
	}
	for i := 0; i < n; i++ {
		c.figureC(x+24+i*(w-190)/n, gy+40+int(paperHash(i, 3, uint64(seed))%10), i%3 == 0, true, t[0])
	}
}

// ---- the Game side -----------------------------------------------------------------------------------------

// finalPaperPhoto: the family photo taken during the party (endparty.go keeps it). If the party was not seen
// (skipped, or "-endstage 12"), the party is drawn once off screen at the moment of the photo. The frame is read
// back from the GPU; when that is impossible (tests) it is nil and the page draws its fallback.
func (g *Game) finalPaperPhoto(e *endState) *image.RGBA {
	p := &e.x.party
	if p.photo == nil {
		tmp := ebiten.NewImage(screenW, screenH)
		rec, lr, t, tot, st := textRec, layerRec, e.t, e.total, e.stage
		textRec, layerRec = nil, nil // nothing of the off-screen party is on the page
		e.stage, e.t = endParty, partyFlashAt-0.1
		g.drawParty(tmp, e)
		textRec, layerRec, e.t, e.total, e.stage = rec, lr, t, tot, st
		tmp.Deallocate()
	}
	if p.photo == nil {
		return nil
	}
	b := p.photo.Bounds()
	im := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	ok := func() (ok bool) {
		defer func() {
			if recover() != nil {
				ok = false
			}
		}()
		p.photo.ReadPixels(im.Pix)
		return true
	}()
	if !ok {
		return nil
	}
	for i := 3; i < len(im.Pix); i += 4 { // an empty read (nothing drawn yet) is no photo
		if im.Pix[i] != 0 {
			return im
		}
	}
	return nil
}

// finalPaperFrozen: the page of this ending, composed once (only a change of language composes it again).
func (g *Game) finalPaperFrozen(e *endState) *finalPaperState {
	fp := &e.x.fpaper
	if !fp.took {
		fp.took = true
		fp.photo = g.finalPaperPhoto(e)
		fp.seed = time.Now().UnixNano()
	}
	if two := g.scene2P(); fp.d == nil || fp.lang != uiLang || fp.two != two {
		fp.freeTex()
		fp.lang, fp.two = uiLang, two
		fp.d = finalPaperData(fp.seed, two)
	}
	return fp
}

func (fp *finalPaperState) freeTex() {
	for i, t := range fp.tex {
		if t != nil {
			t.Deallocate()
			fp.tex[i] = nil
		}
	}
	fp.lay = nil
}

func (g *Game) drawFinalPaper(dst *ebiten.Image, e *endState) {
	fp := g.finalPaperFrozen(e)
	st := int(g.time*4) & 1 // the banner blinks like the one of the level paper; the words never change
	if fp.tex[st] == nil {
		im, lay := renderFinalPaper(fp.d, fp.photo, st)
		fp.tex[st] = ebiten.NewImageFromImage(im)
		if fp.lay == nil {
			fp.lay = lay
		}
	}
	dst.Fill(color.RGBA{0, 0, 0, 255})
	dst.DrawImage(fp.tex[st], nil)
	if textRec != nil {
		textRec.fills = append(textRec.fills, panelRect{0, 0, screenW, screenH, true, len(textRec.texts)})
		for _, t := range fp.lay.texts {
			textRec.texts = append(textRec.texts, textRect{"drawText", t.s, float64(t.x), float64(t.y), float64(t.w), float64(t.h), float64(t.scale), screenW, screenH, dst.Bounds(), false})
		}
	}
}
