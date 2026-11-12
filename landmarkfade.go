package main

import "strings"

// The big monuments (the Eiffel Tower, the leaning tower, the walls of the amphitheatre, the masts of the pirate ship)
// are almost always standing in front of somebody, so the usual "tall things turn see-through while somebody is behind them" made
// them look like ghosts. They stay solid while guests walk behind them and only turn a little see-through while a
// PLAYER is behind (so you never lose yourself). landmarkFadeAmp gives how much (0 = never).
func landmarkFadeAmp(name string) (float64, bool) {
	switch {
	case name == "eiffel":
		return 0.4, true
	case name == "pisa":
		return 0.4, true
	case strings.HasPrefix(name, "colos") && name != "colosfloor":
		return 0.4, true
	case name == "pmast": // the masts of the BARCO PIRATA (pirate_props.go)
		return 0.5, true
	}
	return 0, false
}

// italyMonument: the pieces of ITALIA that are the party itself: no VIP carpet or overlap fix may remove them.
func italyMonument(name string) bool {
	return name == "pisa" || name == "pisalawn" || strings.HasPrefix(name, "colos")
}

// playerBehind: like actorBehind, but only the players count.
func (g *Game) playerBehind(pr *Prop, pi PropImg) bool {
	w, h := float64(pi.img.Bounds().Dx()), float64(pi.img.Bounds().Dy())
	x0, y0 := pr.X-pi.ox, pr.Y-pi.oy
	for _, p := range g.players() {
		if p == nil || p.dead || p.hidden {
			continue
		}
		if p.pos.Y >= pr.Sort-1 || p.pos.Y < y0 {
			continue
		}
		if p.pos.X > x0+2 && p.pos.X < x0+w-2 && p.pos.Y-22 < y0+h {
			return true
		}
	}
	return false
}
