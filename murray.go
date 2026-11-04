package main

import "math/rand"

// ---------------------------------------------------------------------------
// Murray, the evil skull on a stake: he sits in a few parties and never stops
// talking, as in Monkey Island 3. He is decoration (not a guest: he does not
// count for finishing the party and nobody can kill him); his lines go through
// the bubble manager like everybody else's.
// ---------------------------------------------------------------------------

type murrayT struct {
	next, until float64
	line        string
	last        int
}

var (
	murrays   = map[*Prop]*murrayT{}
	murrayLvl *Level
	murrayRng = rand.New(rand.NewSource(1993))
)

// murrayTalk advances Murray's chatter and returns the frame to draw (0 shut, 1 or 2 talking).
func (g *Game) murrayTalk(pr *Prop) int {
	if murrayLvl != g.lvl {
		murrays, murrayLvl = map[*Prop]*murrayT{}, g.lvl
	}
	st := murrays[pr]
	if st == nil {
		st = &murrayT{next: g.time + 1 + murrayRng.Float64()*2, last: -1}
		murrays[pr] = st
	}
	if g.time >= st.next && g.time >= st.until {
		l := g.freshPhrase("murray", murrayLines, murrayRng.Intn)
		if l == "" { // he has said everything lately: a short silence
			st.next = g.time + 2
		} else {
			st.line = l
			st.until = g.time + 2.2 + float64(len([]rune(st.line)))/20
			st.next = st.until + 1 + murrayRng.Float64()*2.5 // he talks all the time
		}
	}
	if g.time >= st.until {
		return 0
	}
	if x, y := g.wx(pr.X), g.wy(pr.Y); x > -10 && x < float64(screenW)+10 && y > 0 && y < float64(screenH)+60 {
		g.bubble(pr, bubGuest, Vec{pr.X, pr.Y}, st.line, false, -2)
	}
	return 1 + int(g.time*8)%2
}

// murrayLevels: the parties where he is stuck on his stake.
var murrayLevels = map[string]bool{"MIAMI BEACH": true, "CRUCERO NOCTURNO": true, "CEMENTERIO": true, "BULL'S EYE": true}

// addMurray puts the stake on firm ground, on a spot where it stands on nothing and away from the start.
func addMurray(d *LevelDef) {
	if d == nil || !murrayLevels[d.Title] {
		return
	}
	for _, p := range d.Props {
		if p.Kind == "murray" {
			return
		}
	}
	fitAllFootprints()
	var geos []*geoProp
	for _, in := range d.Props {
		if g := mkGeo(in); g != nil {
			geos = append(geos, g)
		}
	}
	var spots []Vec
	for _, r := range d.Roam {
		for _, f := range [][2]float64{{0.85, 0.8}, {0.15, 0.8}, {0.85, 0.2}, {0.15, 0.2}, {0.5, 0.85}, {0.5, 0.5}} {
			spots = append(spots, Vec{r.X + r.W*f[0], r.Y + r.H*f[1]})
		}
	}
	for _, sp := range spots {
		if dist(sp, d.Start) < 90 || sp.X < 40 || sp.X > d.W-40 || sp.Y < 70 || sp.Y > d.H-30 {
			continue
		}
		in := PropInst{Kind: "murray", X: float64(int(sp.X)), Y: float64(int(sp.Y))}
		g := mkGeo(in)
		if g == nil || g.clashes(d, geos, nil) || !inWalkable(d, g.foot) {
			continue
		}
		d.Props = append(d.Props, in)
		return
	}
}
