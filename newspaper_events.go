package main

import (
	"sort"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// THE NEWSPAPER TELLS WHAT HAPPENED AT THE PARTY: every "sometimes" event that took place leaves a mark (pevState.ran,
// and g.stat("pev_<id>") for the statistics) and the front page prints up to three small news about them, each one chosen
// among 3 variants by the seed of the party (and leaving out what the last parties printed). The ones of partylife.go
// (selfies, dance contest, pizza boy) are read from g.life. Nothing here touches g.rng.
//
// Another event can add its own: registerPaperEvent("id", pool) and make it start through the frame (partyNoteRan).
// A photo-booth strip may take the place of the photo of the page (paperSceneBooth).
// ---------------------------------------------------------------------------

const paperSceneBooth = 99 // the photo of the page when the booth was used (a scene outside paperPhotoScenes)

type paperEventPool struct {
	head string
	pool []paperTxt
}

var paperEventPools = map[string]paperEventPool{}

// registerPaperEvent: the small news (3 variants is the rule) of one event id. Their English goes to the table.
func registerPaperEvent(id, head string, pool []paperTxt) {
	paperEventPools[id] = paperEventPool{head, pool}
	for _, p := range pool {
		if _, ok := langDataEN[p.es]; !ok {
			langDataEN[p.es] = p.en
		}
	}
}

// partyNoteRan: the frame calls it when an event starts (once per run).
func (g *Game) partyNoteRan(e *partyEvent) {
	if g.pev.ran == nil {
		g.pev.ran = map[string]bool{}
	}
	g.pev.ran[e.ID] = true
	g.stat("pev_"+e.ID, 1)
}

// paperEventSet: what happened in this party (the ids of the events).
func (g *Game) paperEventSet() map[string]bool {
	m := map[string]bool{}
	for id := range g.pev.ran {
		m[id] = true
	}
	ls := &g.life
	if ls.ready {
		if ls.selfieT > 18+selfieJit {
			m["selfie"] = true
		}
		// what really started (the flags), never "the planned time has passed": g.time keeps running while the paper is
		// on screen, and comparing with it made the news appear in the middle of the page (the page kept changing)
		if ls.contestRan {
			m["contest"] = true
		}
		if ls.pizzaRan || ls.pizzaDead {
			m["pizza"] = true
			if ls.pizzaDead {
				m["pizzadead"] = true
				delete(m, "pizza")
			}
		}
	}
	// the fireworks of THIS party count from the moment they are armed (completeLevel), not when they are shown: the
	// first frame of the paper must already be the final page
	if g.fwx.shown || (g.fwx.armed && g.fwx.endT == g.endTime) {
		m["fireworks"] = true
	}
	return m
}

func init() {
	r := registerPaperEvent
	r("conga", "SOCIEDAD", []paperTxt{
		tx("Una conga recorrió la fiesta y se llevó por delante a media sala.", "A conga line crossed the party and took half the room with it."),
		tx("Nadie sabe quién empezó la conga. Nadie sabe cómo parar la conga.", "Nobody knows who started the conga. Nobody knows how to stop it."),
		tx("La conga dio dos vueltas y media. El pie izquierdo pide el divorcio.", "The conga did two and a half laps. The left foot is asking for a divorce."),
	})
	r("chairs", "DEPORTES", []paperTxt{
		tx("Juego de las sillas: ganó quien más rápido se sentó. Hay quejas.", "Musical chairs: the fastest to sit won. There are complaints."),
		tx("Silla menos, empujón más. El juego de las sillas acabó en protesta.", "One chair less, one shove more. Musical chairs ended in protest."),
		tx("La última silla tuvo dos dueños y un solo trasero. Se estudia el caso.", "The last chair had two owners and one backside. The case is studied."),
	})
	r("drinks", "DEPORTES", []paperTxt{
		tx("Concurso de tragos en la barra: ganó el que veía triple.", "Drinking contest at the bar: the one seeing triple won."),
		tx("Duelo de vasos: un campeón, un mareado y un camarero agotado.", "Glass duel: one champion, one dizzy loser and one worn-out bartender."),
		tx("La barra vivió su noche de gloria. El suelo, también: se movía.", "The bar had its night of glory. So did the floor: it was moving."),
	})
	r("bbq", "GASTRONOMÍA", []paperTxt{
		tx("La barbacoa se encendió, la barbacoa se lució y el chuletón, no.", "The barbecue lit up and showed off. The steak did not make it."),
		tx("Llamarada en la parrilla. El cocinero jura que estaba al punto.", "Flames on the grill. The cook swears it was medium."),
		tx("Olor a brasa en toda la fiesta. Los vecinos preguntan si hay sitio.", "Smell of charcoal all over the party. The neighbours ask if there is room."),
	})
	r("bachelor", "SOCIEDAD", []paperTxt{
		tx("Una despedida de soltero cantó, brindó y se perdió. En ese orden.", "A bachelor party sang, toasted and got lost. In that order."),
		tx("El homenajeado de la despedida aún no sabe cómo acabó con ese sombrero.", "The guest of honour still does not know how he ended up in that hat."),
		tx("Despedida a todo trapo: banda, pancarta y retos. Se salvó el novio.", "Send-off in full swing: sash, banner and dares. The groom survived."),
	})
	r("mime", "CULTURA", []paperTxt{
		tx("Un mimo atrapado en una caja invisible pidió ayuda sin decir una palabra.", "A mime trapped in an invisible box asked for help without a word."),
		tx("El mimo cosechó aplausos y monedas. Sigue sin poder salir de la caja.", "The mime won applause and coins. He still cannot leave the box."),
		tx("Un artista mudo conmovió al público. Más tarde confesó que tenía ganas de hablar.", "A silent artist moved the audience. Later he admitted he wanted to talk."),
	})
	r("juggler", "CULTURA", []paperTxt{
		tx("Un malabarista dejó caer una bola y el público, la mandíbula.", "A juggler dropped a ball and the crowd dropped its jaw."),
		tx("Tres bolas, dos manos y una sola cara de susto. Gran número.", "Three balls, two hands and a single scared face. Great act."),
		tx("El malabarista asegura que lo de la bola fue 'parte del número'.", "The juggler says the ball thing was 'part of the act'."),
	})
	r("busker", "MÚSICA", []paperTxt{
		tx("Músicos callejeros tocaron en plena fiesta. Recaudaron cuatro monedas.", "Street musicians played in the middle of the party. They raised four coins."),
		tx("Un dúo de músicos hizo bailar a tres invitados y llorar a un perro.", "A duo of musicians made three guests dance and one dog cry."),
		tx("La banda improvisada pidió una canción y nadie se la sabía. Triunfó.", "The improvised band asked for a song and nobody knew it. A triumph."),
	})
	r("toast", "SOCIEDAD", []paperTxt{
		tx("El anfitrión brindó subido a una caja. La caja pide que no se repita.", "The host gave a toast standing on a crate. The crate asks for no repeat."),
		tx("Brindis general: copas en alto y cero ideas de por qué se brindaba.", "General toast: glasses up and no idea what for."),
		tx("El discurso del anfitrión fue corto, emotivo y con un hipo a mitad.", "The host's speech was short, moving and had a hiccup in the middle."),
	})
	r("selfie", "SOCIEDAD", []paperTxt{
		tx("Los invitados se hicieron tantos selfies que la fiesta salió borrosa.", "Guests took so many selfies that the party came out blurry."),
		tx("Un selfie de grupo salió perfecto, salvo por quien estaba al fondo.", "A group selfie came out perfect, except for the one in the background."),
		tx("Se pide a los invitados que miren a la fiesta y no solo al móvil.", "Guests are asked to look at the party and not only at the phone."),
	})
	r("contest", "DEPORTES", []paperTxt{
		tx("El concurso de baile acabó con un ganador y varias rodillas lesionadas.", "The dance contest ended with one winner and several injured knees."),
		tx("Pista llena en el concurso de baile. El jurado miraba el reloj.", "Full floor at the dance contest. The jury kept checking the clock."),
		tx("La ganadora del baile dedica el trofeo a sus zapatos. Y a nadie más.", "The dance winner dedicates the trophy to her shoes. And nobody else."),
	})
	r("pizza", "ÚLTIMA HORA", []paperTxt{
		tx("Llegó un pizzero a la dirección equivocada. Se quedó a la fiesta.", "A pizza boy arrived at the wrong address. He stayed for the party."),
		tx("Pizza entregada a un invitado que no la había pedido. Nadie protestó.", "Pizza delivered to a guest who had not ordered it. Nobody complained."),
		tx("El repartidor sigue buscando al cliente. La pizza, ya no existe.", "The delivery man is still looking for the customer. The pizza no longer exists."),
	})
	r("pizzadead", "SUCESOS", []paperTxt{
		tx("Se busca repartidor desaparecido. Última vez visto con una pizza.", "Missing delivery man sought. Last seen carrying a pizza."),
		tx("La pizzería denuncia la desaparición de su repartidor. Y de la pizza.", "The pizzeria reports its delivery man missing. And the pizza."),
		tx("Repartidor desaparecido en plena entrega: la masa madre está inquieta.", "Delivery man missing mid-delivery: the sourdough is worried."),
	})
	r("abuela_dead", "SUCESOS", []paperTxt{
		tx("Tragedia en la fiesta: la abuela más cotilla ya no podrá contarlo.", "Tragedy at the party: the nosiest granny will not be able to tell it."),
		tx("Muere la vecina que lo sabía todo. El barrio pierde su telediario.", "The neighbour who knew everything is dead. The block loses its evening news."),
		tx("Conmoción: la abuela del chisme se fue sin acabar la historia.", "Shock: the gossiping granny left without finishing the story."),
	})
	r("tvcrew", "ÚLTIMA HORA", []paperTxt{
		tx("Según la televisión local, la fiesta 'transcurre con total normalidad'.", "According to local TV, the party 'is going on as normal'."),
		tx("Un equipo de televisión grabó la fiesta en directo. Salió un invitado saludando.", "A TV crew filmed the party live. A guest waving came out."),
		tx("La tele local pidió declaraciones. El micrófono acabó en la ensaladilla.", "Local TV asked for statements. The microphone ended up in the potato salad."),
	})
	r("booth", "CARTELERA", []paperTxt{
		tx("La cabina de fotos hizo el agosto: cuatro caras por tira y ninguna seria.", "The photo booth made a killing: four faces per strip and none serious."),
		tx("Una tira de la cabina de fotos circula por la fiesta. Hay lengua en la tercera.", "A photo booth strip is going around the party. There is tongue in the third."),
		tx("La cortina de la cabina de fotos aguantó tres flashes y una discusión.", "The photo booth curtain survived three flashes and one argument."),
	})
	r("magician", "CULTURA", []paperTxt{
		tx("El mago sacó un conejo de la chistera. El conejo pidió una explicación.", "The magician pulled a rabbit from the hat. The rabbit asked for an explanation."),
		tx("Un truco de cartas salió volando por los aires. Aún se buscan la sota y el rey.", "A card trick flew into the air. The jack and the king are still missing."),
		tx("El mago aseguró que el humo era 'parte del número'. Los bomberos opinan otra cosa.", "The magician said the smoke was 'part of the act'. The fire brigade disagrees."),
	})
	r("toiletq", "VECINOS", []paperTxt{
		tx("Larga cola ante el único baño: se aguantó con dignidad, cruzando las piernas.", "Long queue for the only toilet: held with dignity, legs crossed."),
		tx("Un colado en la cola del baño provocó un '¡Oye!' que se oyó en la calle.", "A queue-jumper at the toilet caused an 'Oi!' heard from the street."),
		tx("El baño vivió su noche más concurrida. La puerta pide un descanso.", "The toilet had its busiest night. The door asks for a break."),
	})
	r("fireworks", "ÚLTIMA HORA", []paperTxt{
		tx("Fuegos artificiales para cerrar la fiesta. El cielo, encantado; el perro, no.", "Fireworks to close the party. The sky delighted; the dog, not so much."),
		tx("El cielo se llenó de colores al terminar la fiesta. Nadie miró hacia abajo.", "The sky filled with colour at the end of the party. Nobody looked down."),
		tx("Cohetes y aplausos al final de la noche. Los vecinos, por una vez, aplaudieron.", "Rockets and applause at the end of the night. The neighbours, for once, clapped."),
	})
}

// paperEventBriefs: up to three small news about what happened, in an order that depends on the seed.
func paperEventBriefs(f paperFacts) []paperBrief {
	if len(f.events) == 0 {
		return nil
	}
	var ids []string
	for id := range f.events {
		if _, ok := paperEventPools[id]; ok && f.events[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	sort.SliceStable(ids, func(i, j int) bool { return paperEventHash(f.seed, ids[i]) < paperEventHash(f.seed, ids[j]) })
	if len(ids) > 3 {
		ids = ids[:3]
	}
	var out []paperBrief
	for i, id := range ids {
		p := paperEventPools[id]
		out = append(out, paperBrief{head: p.head, es: f.pick(700+i*7+len(id), p.pool)})
	}
	return out
}

func paperEventHash(seed int64, id string) uint64 {
	h := uint64(seed)*6364136223846793005 + 1442695040888963407
	for i := 0; i < len(id); i++ {
		h = (h ^ uint64(id[i])) * 0x100000001b3
	}
	h ^= h >> 29
	return h * 0x9e3779b97f4a7c15
}

// paperEventFirst puts the briefs about the party's events at the head of the shuffled list (none twice).
func paperEventFirst(f paperFacts, list []paperBrief) []paperBrief {
	ev := paperEventBriefs(f)
	if len(ev) == 0 {
		return list
	}
	seen := map[string]bool{}
	out := make([]paperBrief, 0, len(list)+len(ev))
	for _, b := range ev {
		seen[b.es] = true
		out = append(out, b)
	}
	for _, b := range list {
		if !seen[b.es] {
			out = append(out, b)
		}
	}
	return out
}

// paperBoothPhoto: the photo of the page is the booth strip (the booth was used and the seed says so: about half the times).
func (f paperFacts) paperBoothPhoto() bool { return f.events["booth"] && (f.seed>>5)&1 == 0 }

var paperBoothCaps = []paperTxt{
	tx("Tira de la cabina de fotos: cuatro caras y ninguna de perfil.", "Photo booth strip: four faces and none in profile."),
	tx("Cuatro fotos, cuatro muecas. La tercera la firma el hipo.", "Four photos, four grimaces. The third one is signed by the hiccup."),
	tx("La cabina lo vio todo. Y lo imprimió en cuatro tiras.", "The booth saw it all. And printed it in four strips."),
}

func init() {
	for _, p := range paperBoothCaps {
		if _, ok := langDataEN[p.es]; !ok {
			langDataEN[p.es] = p.en
		}
	}
}

func paperBoothCaption(f paperFacts) string { return f.pick(60, paperBoothCaps) }

// boothStrip draws the photo of the page: a strip of four sepia pictures of silly faces on a table.
func (c *pcv) boothStrip(x, y, w, h int, seed int64) {
	t := photoTone
	c.dfill(x, y, w, h, 0.3, t[1], t[2]) // the table (dithered: 1 px stripes looked like a printing fault)
	fw, fh := 38, 50
	gap := (w - 4*fw) / 5
	for i := 0; i < 4; i++ {
		fx := x + gap + i*(fw+gap)
		fy := y + (h-fh)/2 - 2 + (i%2)*3
		c.fill(fx-2, fy-2, fw+4, fh+4, t[3])
		c.fill(fx, fy, fw, fh, t[2])
		cx, cy := fx+fw/2, fy+fh/2+4
		c.ellipse(cx, cy, 11, 13, t[0])
		c.ellipse(cx, cy, 9, 11, t[3])
		switch (i + int(seed&3)) % 4 {
		case 0: // plain smile
			c.fill(cx-5, cy-4, 2, 2, t[0])
			c.fill(cx+3, cy-4, 2, 2, t[0])
			c.fill(cx-4, cy+4, 8, 1, t[0])
		case 1: // tongue out
			c.fill(cx-5, cy-4, 2, 2, t[0])
			c.fill(cx+3, cy-4, 2, 2, t[0])
			c.fill(cx-4, cy+4, 8, 1, t[0])
			c.fill(cx-1, cy+5, 3, 4, t[1])
		case 2: // crossed eyes
			c.fill(cx-2, cy-4, 2, 2, t[0])
			c.fill(cx+1, cy-4, 2, 2, t[0])
			c.fill(cx-3, cy+5, 6, 2, t[0])
		default: // a hat and open mouth
			c.fill(cx-12, cy-14, 24, 3, t[0])
			c.fill(cx-7, cy-22, 14, 9, t[0])
			c.fill(cx-5, cy-4, 2, 2, t[0])
			c.fill(cx+3, cy-4, 2, 2, t[0])
			c.ellipse(cx, cy+6, 3, 3, t[0])
		}
		c.fill(cx-12, fy+fh-8, 24, 8, t[0]) // shoulders
	}
}

// paperEventTextOK: for the tests, every small news about an event fits the brief rule (<= 90 characters).
func paperEventTextOK(s string) bool { return utf8.RuneCountInString(s) <= 90 }
