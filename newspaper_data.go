package main

import "fmt"

// paperPlayer: what one player did in the party (the numbers of one statistics column).
type paperPlayer struct {
	kills                       [kcN]int
	chainMax, clean, seen, cuff int
	coop                        coopStats
	title                       string // 2P: the nickname of coop.go
	streakBest                  int    // the best RACHA of clean kills (streak.go)
}

// paperPhotoCaptions: funny captions (Spanish, English) per scene; the seed picks one.
var paperPhotoCaptions = [paperPhotoScenes][][2]string{
	{{"El lugar de los hechos, con cinta y todo. Nadie vio nada.", "The scene of the crime, tape and all. Nobody saw a thing."},
		{"Imagen de archivo: la casa, minutos antes del silencio.", "File photo: the house, minutes before the silence."},
		{"La casa de la fiesta. El felpudo dice «Bienvenidos». Mentía.", "The party house. The doormat says \"Welcome\". It lied."},
		{"Un agente (a la izquierda) estudia la puerta. Estaba abierta.", "An officer (left) studies the door. It was open."},
		{"Los vecinos miraron por la ventana y luego, por otra ventana.", "The neighbours looked out of the window and then out of another."},
		{"Vista exterior. Se aprecia una lámpara que no debería estar fuera.", "Outside view. A lamp can be seen that should not be out here."}},
	{{"La pista de baile, poco antes de que alguien dejara de bailar.", "The dance floor, shortly before somebody stopped dancing."},
		{"Instantánea del último baile. Nadie recuerda la canción.", "Snapshot of the last dance. Nobody remembers the song."},
		{"La pista, minutos antes del apagón. Todos con su mejor paso.", "The floor, minutes before the blackout. Everyone at their best step."},
		{"El DJ (al fondo) jura que no tiene nada que ver con nada.", "The DJ (back) swears he has nothing to do with anything."},
		{"Fotografía del baile. Un pie sin dueño ocupa el centro.", "Photograph of the dance. A foot with no owner takes centre stage."},
		{"Los últimos invitados bailaron hasta que bailaron los forenses.", "The last guests danced until the forensics team danced."}},
	{{"Un pato inspecciona la piscina. Se niega a declarar.", "A duck inspects the pool. He refuses to comment."},
		{"Imagen exclusiva: el pato sospechoso junto a la piscina.", "Exclusive: the suspicious duck by the pool."},
		{"La piscina, antes y después. Sobre todo, mojada.", "The pool, before and after. Mostly wet."},
		{"Un flotador solitario espera a su dueño. Lleva ahí desde las diez.", "A lonely float awaits its owner. It has been there since ten."},
		{"El agua parece tranquila. El pato, no tanto.", "The water looks calm. The duck, not so much."},
		{"Un invitado, a remojo. Dice que lo tenía previsto.", "A guest, soaking. He says he had planned it."},
		{"El pato llegó con sombrero. Era el único con etiqueta.", "The duck arrived in a hat. The only one dressed for it."}},
	{{"Un invitado (centro) jura que el cono es un sombrero de diseño.", "A guest (centre) swears the cone is a designer hat."},
		{"El hombre cono dirigió el tráfico hacia el baño toda la noche.", "Cone man directed traffic to the bathroom all night."},
		{"Moda de la noche: el cono. Ayuntamiento busca uno.", "Look of the night: the cone. The council is missing one."},
		{"«Así no me pierdo», declaró el del cono. Se perdió.", "\"This way I won't get lost\", said cone man. He got lost."},
		{"Su amigo (dcha.) no ha parado de reír desde las once.", "His friend (right) has not stopped laughing since eleven."}},
	{{"El perro, con gorro de fiesta, revisa los canapés uno a uno.", "The dog, in a party hat, checks the canapés one by one."},
		{"El invitado mejor vestido de la noche tenía cuatro patas.", "The best dressed guest of the night had four legs."},
		{"El perro niega haber tocado los canapés. Tiene uno en la boca.", "The dog denies touching the canapés. One is in its mouth."},
		{"Nadie sabe de quién es el perro. Él tampoco lo dice.", "Nobody knows whose dog it is. It is not saying."},
		{"El perro pidió un taxi a las dos. Lleva el gorro puesto.", "The dog called a cab at two. Still wearing the hat."}},
	{{"La tarta, después de que alguien la confundiera con una alfombra.", "The cake, after someone mistook it for a rug."},
		{"Tarta de tres pisos, ahora de un piso. Se busca un zapato.", "Three-tier cake, now one tier. Police seek a shoe."},
		{"El pastelero, consultado, se negó a mirar la foto.", "The baker, asked for comment, refused to look at the photo."},
		{"Huella de talla 44 en la nata. La policía mide pies.", "Size 11 print in the cream. Police are measuring feet."},
		{"La vela sigue en pie. Es lo único que sigue en pie.", "The candle is still standing. The only thing that is."}},
	{{"El DJ, en plena sesión. La sesión es de siesta.", "The DJ, mid-set. The set is a nap."},
		{"El DJ pinchó un disco de cuatro horas y se fue a soñar.", "The DJ put on a four-hour record and went to dream."},
		{"«Es música ambiental», explica su representante.", "\"It is ambient music\", his manager explains."},
		{"El público pidió otra. El DJ pidió cinco minutos más.", "The crowd asked for one more. The DJ asked for five more minutes."},
		{"Los altavoces siguen sonando. El DJ, roncando a juego.", "The speakers keep playing. The DJ snores in time."}},
	{{"Un invitado descansa en el sofá. Al revés, por motivos propios.", "A guest rests on the sofa. Upside down, for his own reasons."},
		{"«Así me sube la sangre a la cabeza y pienso mejor», dice.", "\"The blood goes to my head and I think better\", he says."},
		{"El sofá, ocupado desde las once por un par de zapatos.", "The sofa, taken since eleven by a pair of shoes."},
		{"La pizza llegó. El invitado del sofá, todavía no.", "The pizza arrived. The guest on the sofa has not, yet."},
		{"Postura de yoga avanzada o cansancio. Los expertos no se ponen de acuerdo.", "Advanced yoga or exhaustion. Experts disagree."}},
}

// paperPhotoScene chooses the photo: a scene that suits the party (the crime scene only when there was a crime, the
// cake when there was poison or a cake party, the sleeping DJ when there was a DJ...) and, if possible, not one of the
// last two parties (the printed key "FOTO:n").
func paperPhotoScene(f paperFacts) int {
	cand := []int{sceneDance, scenePool, sceneCone, sceneDog, sceneSofa}
	if f.kills > 0 || f.cuffed > 0 {
		cand = append(cand, sceneHouse, sceneHouse)
	}
	if f.poison > 0 || f.venue == "cake" {
		cand = append(cand, sceneCake, sceneCake)
	} else {
		cand = append(cand, sceneCake)
	}
	if f.dj {
		cand = append(cand, sceneDJ, sceneDJ)
	} else {
		cand = append(cand, sceneDJ)
	}
	if f.animalsAlive > 0 || f.animalsDead > 0 {
		cand = append(cand, sceneDog, scenePool)
	}
	var fresh []int
	for _, sc := range cand {
		if !f.avoid[fmt.Sprintf("FOTO:%d", sc)] {
			fresh = append(fresh, sc)
		}
	}
	if len(fresh) > 0 {
		cand = fresh
	}
	h := uint64(f.seed)*0x9e3779b97f4a7c15 + 0x51ed27
	h ^= h >> 31
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 29
	return cand[int(h%uint64(len(cand)))]
}

// paperPhotoCaption: the caption of the scene, not one the last parties printed.
func paperPhotoCaption(f paperFacts, scene int) string {
	var pool []paperTxt
	for _, cp := range paperPhotoCaptions[scene] {
		pool = append(pool, tx(cp[0], cp[1]))
	}
	return f.pick(70, pool)
}

var paperQuoteBy = "- un testigo anónimo"

func init() {
	add := func(k, v string) {
		if _, ok := langDataEN[k]; !ok {
			langDataEN[k] = v
		}
	}
	add("EDICIÓN Nº %s", "EDITION No. %s")
	add("Sábado por la noche", "Saturday night")
	add("PRECIO: 1 aperitivo", "PRICE: 1 appetizer")
	add("IVA: un canapé", "VAT: one canapé")
	add("Todas las noticias que caben en una página", "All the news that fits on one page")
	add("Páginas: 1 (la única)", "Pages: 1 (the only one)")
	add("NUESTRO ENVIADO ESPECIAL INFORMA", "OUR SPECIAL ENVOY REPORTS")
	add("TESTIGO PRESENCIAL", "EYEWITNESS")
	add("CUADRO DE PUNTUACIÓN", "SCORE TABLE")
	add("PULSA PARA CONTINUAR", "PRESS TO CONTINUE")
	add("Prohibida su venta a asesinos reincidentes", "Not to be sold to repeat killers")
	add("También vale ESC, clic o el botón del mando", "ESC, a click or the pad button work too")
	add("ESTADÍSTICAS J%d", "STATS P%d")
	add("J1", "P1")
	add("J2", "P2")
	for _, sc := range paperPhotoCaptions {
		for _, cp := range sc {
			add(cp[0], cp[1])
		}
	}
	add("- un testigo anónimo", "- an anonymous witness")
}

// newPaperData composes the page from facts and numbers (no Game needed: the tests call it directly).
func newPaperData(f paperFacts, fin finishBreak, pl []paperPlayer, endTime float64, day int, hint string, other int) *paperData {
	d := &paperData{seed: f.seed, kills: f.kills, two: len(pl) > 1}
	d.title = tr("EL DIARIO DE LA JUERGA")
	d.earL1, d.earL2 = tr(fmt.Sprintf("EDICIÓN Nº %s", f.level)), tr("Sábado por la noche")
	d.earR1, d.earR2 = tr("PRECIO: 1 aperitivo"), tr("IVA: un canapé")
	d.date = tr(fmt.Sprintf("Sábado, %d de noviembre", day))
	d.motto = tr("Todas las noticias que caben en una página")
	d.dateR = tr("Páginas: 1 (la única)")
	d.head = tr(paperHeadline(f))
	d.dek = fmt.Sprintf("%s %d  ·  %s %d  ·  %s %s  ·  %s %d", tr("Bajas"), f.kills, tr("Otras muertes"), other, tr("Tiempo"), lsFmtTime(endTime), tr("Te han visto"), f.seen)
	for _, s := range paperSentences(f) {
		d.story = append(d.story, tr(s))
	}
	d.storyHead, d.quoteHead = tr("NUESTRO ENVIADO ESPECIAL INFORMA"), tr("TESTIGO PRESENCIAL")
	d.quote, d.quoteBy = tr(paperQuote(f)), tr(paperQuoteBy)
	d.photoVar = paperPhotoScene(f)
	d.caption = tr(paperPhotoCaption(f, d.photoVar))
	if f.paperBoothPhoto() {
		d.photoVar, d.caption = paperSceneBooth, tr(paperBoothCaption(f))
	}
	for i, p := range pl {
		n := func(v int) string { return fmt.Sprint(v) }
		// the owner of the column: "J1" / "J2" in a dark tag (1 player: J1 too)
		sb := paperStatBox{head: tr("ESTADÍSTICAS"), tag: tr(fmt.Sprintf("J%d", i+1))}
		k := p.kills
		sb.rows = [][2]string{
			{tr("Bajas"), n(k[kcKnife] + k[kcPoison] + k[kcTrap] + k[kcBlast] + k[kcNone])},
			{tr("Cuchillo / manos"), n(k[kcKnife] + k[kcNone])},
			{tr("Veneno"), n(k[kcPoison])},
			{tr("Trampas"), n(k[kcTrap])},
			{tr("Explosiones"), n(k[kcBlast])},
			{tr("Cadena máxima"), n(p.chainMax)},
			{tr("Bajas limpias"), n(p.clean)},
			{tr("Te han visto"), n(p.seen)},
			{tr("Policía avisada"), n(p.cuff)},
			{tr("Tiempo"), lsFmtTime(endTime)},
		}
		if d.two {
			sb.rows = sb.rows[:len(sb.rows)-1] // 2P: the time is already in the dek under the headline; its row makes room for the streak line
			sb.sub = tr(p.title)
			sb.rows = append(sb.rows,
				[2]string{tr("Puntos"), n(p.coop.points)},
				[2]string{tr("Golpes al compañero"), n(p.coop.hits)},
				[2]string{tr("Tiempo en el suelo"), fmt.Sprintf("%d s", int(p.coop.floorT+0.5))},
				[2]string{tr("Ayudas"), n(p.coop.helps + p.coop.distracts)})
		}
		sb.streak = streakFor(f, p, i, endTime) // at most 2 lines: two of them still leave room for a small news
		if d.two && len(pl) == 2 {
			sb.award = awardLine(pl[0], pl[1], i) // newspaper_awards.go
		}
		d.stats = append(d.stats, sb)
	}
	plus := func(v int) string { return fmt.Sprintf("+%d", v) }
	d.scoreHead = tr("CUADRO DE PUNTUACIÓN")
	d.score = [][2]string{{tr("Puntos de la fiesta"), fmt.Sprint(fin.base)}}
	if fin.challenge {
		d.scoreTag = tr("RETO x3 · x5")
		d.score = append(d.score, [2]string{tr("Reto x3 en las bajas"), plus(fin.killExtra)})
	}
	d.score = append(d.score,
		[2]string{tr("Cadenas"), plus(fin.chain)})
	if fin.streak > 0 { // RACHA (streak.go): only when it paid something, so a party without it keeps the old table
		d.score = append(d.score, [2]string{tr(fmt.Sprintf("Racha · mejor %d", fin.streakBest)), plus(fin.streak)})
	}
	d.score = append(d.score,
		[2]string{tr("Bajas limpias"), plus(fin.clean)},
		[2]string{tr("Nivel completado"), plus(fin.complete)},
		[2]string{tr("Sin ser visto"), plus(fin.unseen)},
		[2]string{tr("Rapidez"), plus(fin.speed)})
	if fin.challenge {
		d.score = append(d.score, [2]string{tr("Reto x5 al completar"), plus(fin.finalExtra)})
	}
	d.totalK, d.total = tr("TOTAL"), fmt.Sprint(fin.total)
	d.walletK, d.wallet = tr("A la cartera"), plus(fin.wallet)
	for _, b := range paperBriefsFor(f, lsFmtTime(endTime)) {
		d.briefs = append(d.briefs, paperBrief{head: tr(b.head), es: paperBriefText(b)}) // a filled template is already in both languages
	}
	d.hint, d.hint2 = padKeyText(tr(hint)), tr("También vale ESC, clic o el botón del mando") // ENTER -> ✕ on a pad (input.go)
	d.foot = tr("Prohibida su venta a asesinos reincidentes")
	d.banner = tr("PULSA PARA CONTINUAR")
	return d
}

// paperDataNow: the page of the party that has just ended.
func (g *Game) paperDataNow() *paperData {
	f := g.paperFactsNow()
	fin := g.run.fin
	if !g.run.done {
		fin = g.previewFinish()
	}
	r := &g.run
	n := 1
	if g.is2P() {
		n = 2
		g.coopEnsure()
	}
	pl := make([]paperPlayer, n)
	var titles [2]string
	if n == 2 {
		titles = coopTitleFor(g.coop.score[0], g.coop.score[1])
	}
	for i := range pl {
		pl[i] = paperPlayer{kills: r.kills[i], chainMax: r.chainMax[i], clean: r.cleanKills[i], seen: r.seen[i], cuff: r.cuffed[i], title: titles[i], streakBest: r.streakBest[i]}
		if n == 2 {
			pl[i].coop = g.coop.score[i]
		}
	}
	other := g.total - g.aliveCount() - (r.killsOf(0) + r.killsOf(1) - r.nonGuest)
	if other < 0 {
		other = 0
	}
	day := 1 + (g.levelIdx*3)%28
	if day < 1 {
		day = 1
	}
	d := newPaperData(f, fin, pl, g.endTime, day, levelDoneHint(g), other)
	g.paperMedal(d) // the medal of a campaign party, in the head of the points table (medals.go)
	return d
}
