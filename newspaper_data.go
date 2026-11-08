package main

import "fmt"

// paperPlayer: what one player did in the party (the numbers of one statistics column).
type paperPlayer struct {
	kills                       [kcN]int
	chainMax, clean, seen, cuff int
	coop                        coopStats
	title                       string // 2P: the nickname of coop.go
}

// paperPhotoCaptions: two funny captions (Spanish, English) per scene; the seed picks one.
var paperPhotoCaptions = [paperPhotoScenes][2][2]string{
	{{"El lugar de los hechos, con cinta y todo. Nadie vio nada.", "The scene of the crime, tape and all. Nobody saw a thing."},
		{"Imagen de archivo: la casa, minutos antes del silencio.", "File photo: the house, minutes before the silence."}},
	{{"La pista de baile, poco antes de que alguien dejara de bailar.", "The dance floor, shortly before somebody stopped dancing."},
		{"Instantánea del último baile. Nadie recuerda la canción.", "Snapshot of the last dance. Nobody remembers the song."}},
	{{"Un pato inspecciona la piscina. Se niega a declarar.", "A duck inspects the pool. He refuses to comment."},
		{"Imagen exclusiva: el pato sospechoso junto a la piscina.", "Exclusive: the suspicious duck by the pool."}},
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
	for _, sc := range paperPhotoCaptions {
		for _, cp := range sc {
			add(cp[0], cp[1])
		}
	}
	add("- un testigo anónimo", "- an anonymous witness")
}

// newPaperData composes the page from facts and numbers (no Game needed: the tests call it directly).
func newPaperData(f paperFacts, fin finishBreak, pl []paperPlayer, endTime float64, day int, hint string, other int) *paperData {
	d := &paperData{seed: f.seed, two: len(pl) > 1}
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
	d.photoVar = int((f.seed%paperPhotoScenes + paperPhotoScenes) % paperPhotoScenes)
	d.caption = tr(paperPhotoCaptions[d.photoVar][int((f.seed/3%2+2)%2)][0])
	for i, p := range pl {
		n := func(v int) string { return fmt.Sprint(v) }
		head := tr("ESTADÍSTICAS")
		if d.two {
			head = tr(fmt.Sprintf("ESTADÍSTICAS J%d", i+1))
		}
		sb := paperStatBox{head: head}
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
			sb.sub = tr(p.title)
			sb.rows = append(sb.rows,
				[2]string{tr("Puntos"), n(p.coop.points)},
				[2]string{tr("Golpes al compañero"), n(p.coop.hits)},
				[2]string{tr("Tiempo en el suelo"), fmt.Sprintf("%d s", int(p.coop.floorT+0.5))},
				[2]string{tr("Ayudas"), n(p.coop.helps + p.coop.distracts)})
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
		[2]string{tr("Cadenas"), plus(fin.chain)},
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
		d.briefs = append(d.briefs, paperBrief{head: tr(b.head), es: tr(b.es)})
	}
	d.hint, d.hint2 = tr(hint), tr("También vale ESC, clic o el botón del mando")
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
		pl[i] = paperPlayer{kills: r.kills[i], chainMax: r.chainMax[i], clean: r.cleanKills[i], seen: r.seen[i], cuff: r.cuffed[i], title: titles[i]}
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
	return newPaperData(f, fin, pl, g.endTime, day, levelDoneHint(g), other)
}
