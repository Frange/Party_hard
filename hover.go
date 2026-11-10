package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

var maleNames = []string{"Carlos", "Javier", "Miguel", "Pablo", "Sergio", "Álvaro", "Diego", "Hugo", "Iván", "Rubén", "Adrián", "Marcos", "Raúl", "Óscar", "Nacho", "Dani", "Luis", "Jorge", "Víctor", "Tomás", "John", "Mike", "Dave", "Tom", "Luca", "Jean", "Hans", "Kenji", "Mateo", "Bruno"}
var femaleNames = []string{"Lucía", "Marta", "Paula", "Carmen", "Laura", "Sara", "Elena", "Irene", "Nuria", "Clara", "Alba", "Noa", "Eva", "Rocío", "Silvia", "Julia", "Andrea", "Beatriz", "Lola", "Mar", "Emma", "Kate", "Anna", "Sofia", "Chloé", "Yuki", "Greta", "Vera", "Mía", "Olga"}
var surnames = []string{"García", "López", "Ruiz", "Martín", "Gómez", "Díaz", "Pérez", "Sanz", "Vega", "Gil", "Mora", "Ortiz", "Rey", "Soto", "Nieto", "Cano", "Prieto", "Lara", "Ríos", "Marín"}

var nickUsed = map[string]bool{}

// nickRNG is the running game's random source, so labels can be handed out
// from anywhere (news lines, hover tooltips).
var nickRNG *rand.Rand

// randomPersonName draws an unused funny name of the right gender; once the
// list is exhausted it falls back to a plain first name + surname.
func randomPersonName(r *rand.Rand, female bool) string {
	for try := 0; try < 60; try++ {
		fn := funnyNames[r.Intn(len(funnyNames))]
		if fn.female == female && !nickUsed[fn.name] {
			nickUsed[fn.name] = true
			return fn.name
		}
	}
	list := maleNames
	if female {
		list = femaleNames
	}
	return list[r.Intn(len(list))] + " " + surnames[r.Intn(len(surnames))]
}

// ensureNick gives every ordinary guest a name the first time one is needed.
func ensureNick(gu *Guest) string {
	if gu.name != "" {
		return gu.name
	}
	if gu.nick == "" && nickRNG != nil {
		female := gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Female
		gu.nick = randomPersonName(nickRNG, female)
	}
	return gu.nick
}

// hoverLabel returns the name of whoever the mouse is over (empty when nobody).
func (g *Game) hoverLabel(mx, my float64) string {
	wx, wy := mx-g.camI.X, my-g.camI.Y
	bd, label := 14.0, ""
	consider := func(p Vec, text func() string) {
		if wx < p.X-9 || wx > p.X+9 || wy < p.Y-42 || wy > p.Y+5 {
			return
		}
		if d := math.Hypot(wx-p.X, (wy-(p.Y-18))*0.6); d < bd {
			bd = d
			label = text()
		}
	}
	for _, gu := range g.guests {
		if gu.out() || gu.state == gLift {
			continue
		}
		gu := gu
		consider(gu.pos, func() string {
			female := gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Female
			_ = female
			gu.nick = ensureNick(gu)
			switch {
			case gu.perf == "dj":
				return "DJ " + gu.nick
			case gu.perf != "":
				return gu.nick + " (músico)"
			case gu.cs != nil && gu.cs.Spec != nil && gu.cs.Spec.Name == "lifeguard":
				return "Socorrista " + gu.nick
			}
			return gu.nick
		})
	}
	for _, gd := range g.guards {
		if gd.dead {
			continue
		}
		gd := gd
		consider(gd.pos, func() string { return "Guardia " + g.personalName(gd, gd.cs) })
	}
	for _, c := range g.cops {
		c := c
		consider(c.pos, func() string {
			var cs *CharSet
			if c.medic {
				cs = g.as.medicCS
			} else {
				cs = g.as.copCS
			}
			n := g.personalName(c, cs)
			if c.medic {
				return "Médico " + n
			}
			return []string{"Agente ", "Agente FBI ", "Soldado ", "SWAT ", "Ranger "}[clampI(c.kind, 0, 4)] + n
		})
	}
	for _, s := range g.allSlashers() {
		s := s
		if s.state != slGone {
			consider(s.pos, func() string {
				if s.name != "" {
					return s.name
				}
				return "Asesino enmascarado"
			})
		}
	}
	for _, b := range g.bodies { // the dead keep their name until they are carried off or bagged
		if b.carried || b.bagged || b.label == "" {
			continue
		}
		b := b
		consider(Vec{b.pos.X, b.pos.Y + 14}, func() string { return b.label + " (cadáver)" })
	}
	for _, an := range g.animals {
		an := an
		if !an.dead {
			consider(Vec{an.pos.X, an.pos.Y + 22}, func() string {
				if an.label != "" {
					return an.label
				}
				return animalLabel(an.kind)
			})
		}
	}
	if !g.p.dead && !g.p.hidden {
		consider(g.p.pos, func() string { return "Tú" })
	}
	return label
}

// drawHoverName shows a small label next to the mouse pointer over a person.
func (g *Game) drawHoverName(dst *ebiten.Image) {
	if g.mode != ModePlay || g.lvl == nil {
		return
	}
	cx, cy := ebiten.CursorPosition()
	if cx < 0 || cy < 0 || cx >= screenW || cy >= screenH {
		return
	}
	s := g.hoverLabel(float64(cx), float64(cy))
	if s == "" {
		return
	}
	w := g.textW(s, 1) + 10
	x, y := float64(cx)+12, float64(cy)-18
	if x+w > screenW-4 {
		x = float64(cx) - w - 8
	}
	if y < 4 {
		y = 4
	}
	fillRect(dst, x, y, w, 15, color.RGBA{14, 14, 24, 215})
	fillRect(dst, x, y, w, 1, color.RGBA{120, 200, 255, 255})
	g.drawText(dst, s, x+5, y+4, color.RGBA{245, 245, 255, 255}, 1)
}

// personalName gives each guard / officer a fixed full name that matches their sex.
func (g *Game) personalName(key any, cs *CharSet) string {
	if g.names == nil {
		g.names = map[any]string{}
	}
	if n, ok := g.names[key]; ok {
		return n
	}
	female := cs != nil && cs.Spec != nil && cs.Spec.Female
	list := maleNames
	if female {
		list = femaleNames
	}
	n := list[g.rng.Intn(len(list))] + " " + surnames[g.rng.Intn(len(surnames))]
	g.names[key] = n
	return n
}

func animalLabel(k int) string {
	if k < 0 || k >= len(animalKinds) {
		return "Animal"
	}
	if n, ok := map[string]string{"horse": "Caballo", "hen": "Gallina", "dog": "Perro", "cat": "Gato", "bull": "Toro", "elephant": "Elefante", "giraffe": "Jirafa", "kangaroo": "Canguro", "chick": "Pollito", "rabbit": "Conejo", "pig": "Cerdo", "duck": "Pato", "budgie": "Periquito", "deer": "Ciervo", "quijote": "Don Quijote y Rocinante", "sancho": "Sancho Panza y su rucio",
		"snake": "Serpiente", "frog": "Rana", "squirrel": "Ardilla", "boar": "Jabalí", "croc": "Cocodrilo", "cheetah": "Guepardo", "lion": "León", "cobaya": "Cobaya"}[animalKinds[k]]; ok {
		return n
	}
	return "Animal"
}
