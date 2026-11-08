package main

import (
	"fmt"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// The end-of-level NEWSPAPER (the only screen of ModeLevelDone; it is drawn by newspaper_draw.go).
// A front page "EL DIARIO DE LA JUERGA" with a headline made from the statistics and a short news
// story in the third person, composed from sentence templates. Every choice is deterministic: the same
// facts give the same paper (paperSeed), a different party gives a different one. Nothing here is
// stored; no real names are used. Texts are Spanish keys (tr() translates them, lang_newspaper.go).
// ---------------------------------------------------------------------------

// paperFacts: everything the paper knows about the party.
type paperFacts struct {
	level                          string // "7" or "EXTRA"
	levelN                         int
	kills, clean, seen, cuffed     int
	chainMax                       int
	knife, poison, traps, blast    int
	slasher, medic, camera, jailed bool
	units                          uint
	two, challenge                 bool
	secs                           int
	seed                           int64
}

func (g *Game) paperFactsNow() paperFacts {
	r := &g.run
	f := paperFacts{level: "EXTRA", levelN: g.levelIdx + 1, two: g.is2P(), challenge: g.challenge, secs: int(g.endTime)}
	if g.levelIdx >= 0 && g.levelIdx < campaignN {
		f.level = fmt.Sprint(g.levelIdx + 1)
	}
	for i := 0; i < 2; i++ {
		f.kills += r.killsOf(i)
		f.clean += r.cleanKills[i]
		f.seen += r.seen[i]
		f.cuffed += r.cuffed[i]
		f.chainMax = maxI(f.chainMax, r.chainMax[i])
	}
	f.knife = r.total(kcKnife) + r.total(kcNone)
	f.poison, f.traps, f.blast = r.total(kcPoison), r.total(kcTrap), r.total(kcBlast)
	f.slasher, f.medic, f.camera, f.jailed = r.slasher, r.medic, r.camSeen > 0, r.jailed > 0
	f.units = r.units
	f.seed = int64(g.levelIdx+3)*1000003 + int64(f.kills)*131 + int64(f.seen)*17 + int64(f.secs)
	return f
}

// paperPick: a deterministic choice among the variants of a situation.
func paperPick(seed int64, salt int, list []string) string {
	h := uint64(seed)*6364136223846793005 + uint64(salt+1)*1442695040888963407
	h ^= h >> 29
	return list[int(h%uint64(len(list)))]
}

// dominant: the method that explains at least half of the kills ("" = a mix).
func (f paperFacts) dominant() string {
	tot := f.knife + f.poison + f.traps + f.blast
	if tot == 0 {
		return ""
	}
	for _, m := range []struct {
		n int
		s string
	}{{f.knife, "knife"}, {f.poison, "poison"}, {f.traps, "trap"}, {f.blast, "blast"}} {
		if m.n*2 >= tot {
			return m.s
		}
	}
	return ""
}

// paperHeadline: the big title with the number of the party.
func paperHeadline(f paperFacts) string {
	k := f.kills
	var forms []string
	switch {
	case f.seen == 0 && k >= 5:
		forms = []string{"FIESTA %s: LA MÁS LIMPIA DEL BARRIO", "FIESTA %s: %d AUSENTES Y NI UN TESTIGO"}
	case f.blast > 0 && f.dominant() == "blast":
		forms = []string{"FIESTA %s: ¡BOOM! %d AUSENTES EN LA PISTA", "FIESTA %s: LA PISTA VOLÓ CON %d INVITADOS"}
	case f.poison > 0 && f.dominant() == "poison":
		forms = []string{"FIESTA %s: EL PONCHE SE LLEVÓ A %d", "FIESTA %s: %d INVITADOS SE QUEDAN SIN POSTRE"}
	case k <= 3:
		forms = []string{"FIESTA %s: %d BAJAS Y CERO ESCÁNDALO", "FIESTA %s: UNA VELADA TRANQUILA, DICEN"}
	case k >= 20:
		forms = []string{"FIESTA %s: %d DESAPARECIDOS Y NI UNA PROPINA", "FIESTA %s: SE ACABARON LOS INVITADOS"}
	case f.cuffed > 0:
		forms = []string{"FIESTA %s: %d BAJAS Y LA POLI AVISADA TARDE", "FIESTA %s: SALDO %d, Y LA POLI SIN PISTAS"}
	default:
		forms = []string{"FIESTA %s — SALDO: %d DESAPARECIDOS", "FIESTA %s: %d INVITADOS NO VOLVERÁN A CASA"}
	}
	form := paperPick(f.seed, 1, forms)
	if strings.Count(form, "%d") == 0 {
		return fmt.Sprintf(form, f.level)
	}
	return fmt.Sprintf(form, f.level, k)
}

// paperSentences: the sentences of the story in priority order (the first one always exists).
func paperSentences(f paperFacts) []string {
	var open []string
	switch f.dominant() {
	case "knife":
		open = []string{"El asesino de la fiesta lo hizo a la antigua: ¡ZAS, ZAS!", "Un misterioso invitado repartió ¡ZAS! y ¡PLAF! sin pedir permiso."}
	case "poison":
		open = []string{"Según fuentes cercanas al aperitivo, el ponche llevaba un extra no pedido.", "El veneno fue el invitado más puntual: nadie lo vio llegar."}
	case "trap":
		open = []string{"Las trampas hicieron ¡CLAC! y ¡PUM! sin que nadie leyera el manual.", "El asesino de la fiesta dejó trampas por todas partes. ¡CLAC!"}
	case "blast":
		open = []string{"Un ¡BOOM! de los gordos dejó la pista sin sobremesa.", "Los vecinos oyeron ¡BOOM! y creyeron que era el DJ."}
	default:
		open = []string{"Cuchillo, veneno y trampas: un menú degustación completo.", "El asesino de la fiesta no tiene método: lo prueba todo."}
	}
	switch {
	case f.kills <= 3:
		open = append(open, "Fueron bajas discretas: casi nadie se enteró del asunto.")
	case f.kills >= 20:
		open = append(open, "Los forenses perdieron la cuenta a mitad de canapés.")
	}
	s := []string{paperPick(f.seed, 2, open)}

	var opt []string // the rest, in a stable order shuffled by the seed
	add := func(salt int, list ...string) { opt = append(opt, paperPick(f.seed, salt, list)) }
	if f.seen == 0 {
		add(10, "Nadie vio nada. Los testigos juran estar mirando el móvil.", "Ni un solo testigo: todos miraban al DJ, claro.")
	} else if f.seen >= 4 {
		add(11, "Hubo más testigos que canapés y todos corrían a chivarse.", "Testigos aseguran haberlo visto cuatro, cinco, ¡mil veces!")
	} else {
		add(12, "Testigos aseguran haberlo visto, aunque iba disfrazado de sombra.", "Un testigo lo vio, pero dice que fue sin querer.")
	}
	if f.cuffed > 0 {
		add(13, "Alguien avisó a la policía, pero el sospechoso se esfumó.", "Un testigo llamó a la poli. Casi lo pillan... y se escapó.")
	}
	if f.jailed {
		add(14, "Uno de los asesinos pasó por el calabozo; el otro, de rositas.", "Hubo detenido, hubo fianza y hubo mucha cara dura.")
	}
	if f.camera {
		add(15, "Una cámara lo grabó, pero la imagen salió fatal.", "La cámara de seguridad lo pilló, y encima salió mal peinado.")
	}
	if f.slasher {
		add(16, "Por si fuera poco, otro asesino rondaba por allí. ¡Competencia!", "Otro asesino andaba suelto. Qué fiesta más concurrida.")
	}
	switch {
	case f.units&(1<<unitMilitary) != 0:
		add(17, "El ejército desembarcó con tanques para un simple aperitivo.")
	case f.units&(1<<unitSWAT) != 0:
		add(17, "Llegó el SWAT, con megáfono, cuerda y todo.")
	case f.units&(1<<unitRangers) != 0:
		add(17, "Hasta unos rangers de colores llegaron dando saltos.")
	case f.units&(1<<unitFBI) != 0:
		add(17, "Dos agentes del FBI aparecieron muy serios y sin una pista.")
	case f.units&(1<<unitPolice) != 0:
		add(17, "Una patrulla acudió con sirena y cero resultados.", "La policía llegó tarde, como siempre. ¡NI-NO, NI-NO!")
	default:
		add(17, "Ni una sirena: la ciudad dormía a pierna suelta.")
	}
	if f.medic {
		add(18, "Las ambulancias hicieron horas extra. ¡PIII-PIII!")
	}
	if f.chainMax >= 4 {
		add(19, "Mató en cadena, ¡ZAS-ZAS-ZAS!, sin parar a respirar.", "Una racha de bajas tan seguida que ni dio tiempo al brindis.")
	}
	if f.two {
		add(20, "Los asesinos se repartieron el trabajo. Lo llaman equipo.")
	}
	if f.challenge {
		add(21, "Se rumorea que lo hizo todo por un premio extra.")
	}
	type keyed struct {
		k int
		s string
	}
	ks := make([]keyed, len(opt))
	for i, o := range opt {
		ks[i] = keyed{int(paperPick(f.seed, 100+i, []string{"0", "1", "2", "3", "4", "5", "6", "7"})[0]), o}
	}
	sort.SliceStable(ks, func(i, j int) bool { return ks[i].k < ks[j].k })
	for i := range ks {
		opt[i] = ks[i].s
	}
	s = append(s, opt...)
	return append(s, paperPick(f.seed, 30, []string{"El culpable sigue en libertad, seguramente comiendo tortilla.", "El culpable sigue suelto. Se le ofrece un canapé por su captura."}))
}

// paperQuote: the witness at the bottom of the front page.
func paperQuote(f paperFacts) string {
	return paperPick(f.seed, 40, []string{"«Yo solo vine por el aperitivo.»", "«Se lo juro, estaba en el baño.»", "«Yo no vi nada. Y tampoco me pagan.»", "«La música estaba demasiado alta para gritar.»"})
}
