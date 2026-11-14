package main

import "math"

// ---------------------------------------------------------------------------
// SEE-THROUGH WITH ANTICIPATION (v1.4.0). Tall things (tree tops, roofs, tents, booths...) turn see-through while
// somebody is behind them (draw.go). Before, they only started once the person was already hidden and stayed fairly
// solid; now they start fading a little BEFORE (fadeAheadMargin px round the "behind" zone, more the closer he is), go
// further (fadeAmp) and the change is a soft fade (fadeRate per frame towards the target).
// ---------------------------------------------------------------------------

const (
	fadeAmp         = 0.7  // how see-through a tall thing gets (was 0.55: 1 = invisible)
	fadeAheadMargin = 30.0 // px round the hidden zone where it already starts to fade
	fadeRate        = 0.12 // share of the way to the target per frame (was 0.2: a softer fade)
)

// behindK: 0 = nobody near, 1 = somebody hidden behind the sprite; in between while he is getting there (within
// fadeAheadMargin of the zone where the sprite covers him).
func behindK(pr *Prop, pi PropImg, p Vec) float64 {
	w, h := float64(pi.img.Bounds().Dx()), float64(pi.img.Bounds().Dy())
	return behindKRect(pr.X-pi.ox, pr.Y-pi.oy, w, h, pr.Sort, p)
}

// behindKRect: the same on a sprite rectangle (x0, y0, w, h) whose sort line is sortY (pure: tested).
func behindKRect(x0, y0, w, h, sortY float64, p Vec) float64 {
	// the zone where a person (feet at p, about 22 px tall over them) is covered: inside the sprite's columns, between its
	// top and its sort line
	xl, xr := x0+2, x0+w-2
	yt, yb := y0, math.Min(sortY-1, y0+h+22)
	if xr <= xl || yb <= yt {
		return 0
	}
	dx := math.Max(0, math.Max(xl-p.X, p.X-xr))
	dy := math.Max(0, math.Max(yt-p.Y, p.Y-yb))
	if dx == 0 && dy == 0 && p.Y < sortY-1 {
		return 1
	}
	d := math.Hypot(dx, dy)
	if p.Y >= sortY-1 && d == 0 {
		d = p.Y - (sortY - 1) + 0.5
	}
	return math.Max(0, 1-d/fadeAheadMargin)
}

// fadeTarget: how see-through a tall prop should be now (the strongest of everybody near it). With playersOnly only the
// players count (the monuments, landmarkfade.go).
func (g *Game) fadeTarget(pr *Prop, pi PropImg, playersOnly bool) float64 {
	best := 0.0
	see := func(p Vec) {
		if best < 1 {
			best = math.Max(best, behindK(pr, pi, p))
		}
	}
	for _, p := range g.players() {
		if p != nil && !p.dead && !p.hidden {
			see(p.pos)
		}
	}
	if playersOnly {
		return best
	}
	for _, gu := range g.guests {
		if !gu.out() {
			see(gu.pos)
		}
	}
	for _, gd := range g.guards {
		if !gd.dead {
			see(gd.pos)
		}
	}
	for _, c := range g.cops {
		see(c.pos)
	}
	return best
}
