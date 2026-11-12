package main

import (
	"fmt"
	"strings"
)

// Coming out of a hiding place (bush, closet, dumpster, stall, portable toilet... every prop with
// Hide). The player comes out EXACTLY where he stood when he went in (Player.hideFrom). Only when
// that spot is taken now (the partner stands on it, something solid is there) does he come out at
// the closest free spot he could WALK to from it: the search floods the ground the player can
// really walk on, so it never jumps a wall into a house or onto the other side of a fence.
//
// Before, the exit was "the nearest free cell 16 px below the prop": with a bush on the roof edge of
// a building (party 1, top right) that cell was inside the house, and the player popped out in it.

const (
	hideExitStep   = 2.0  // flood grid (px)
	hideExitRadius = 96.0 // how far from the entry spot the flood may look
	hideExitBusy   = 7.0  // another player closer than this to a spot takes it
)

// hideEnter: the player goes into the hiding place pr (game.go interact).
func (g *Game) hideEnter(p *Player, pr *Prop) {
	p.hideFrom, p.hideFromOK = p.pos, true
	p.hidden, p.hideProp = true, pr
	p.pos = Vec{pr.X, pr.Y - 4}
}

// hideLeave: the player comes out of his hiding place, at the spot he went in from.
func (g *Game) hideLeave(p *Player) {
	p.pos = g.hideExitPos(p)
	p.hidden = false
	p.hideFromOK = false
}

// hideExitPos: where the hidden player p comes out.
func (g *Game) hideExitPos(p *Player) Vec {
	want := p.hideFrom
	if !p.hideFromOK { // hidden by some other way (old code paths, tests): in front of the prop
		pr := p.hideProp
		if pr == nil {
			return g.lvl.NearestFree(p.pos)
		}
		want = Vec{pr.X, pr.Foot.Y + pr.Foot.H + 5}
	}
	return g.hideFreeNear(p, want)
}

// hideSpotTaken: another (visible, standing) player is on q.
func (g *Game) hideSpotTaken(p *Player, q Vec) bool {
	for _, o := range g.players() {
		if o == nil || o == p || o.hidden || o.dead {
			continue
		}
		if dist(o.pos, q) < hideExitBusy {
			return true
		}
	}
	return false
}

// hideFreeNear: want itself when the player fits there and nobody stands on it; otherwise the
// nearest such spot by walking distance from want (breadth-first flood on a fine grid where a step
// is only allowed onto ground the player's collision box fits on). When want itself is covered by a
// solid (something was put there), the flood may cross that solid, but never a wall, until it
// reaches open ground again; from there on only walkable ground counts.
func (g *Game) hideFreeNear(p *Player, want Vec) Vec {
	l := g.lvl
	if !l.SolidP(want.X, want.Y) && !g.hideSpotTaken(p, want) {
		return want
	}
	n := int(hideExitRadius/hideExitStep)*2 + 1
	c := n / 2
	at := func(i, j int) Vec {
		return Vec{want.X + float64(i-c)*hideExitStep, want.Y + float64(j-c)*hideExitStep}
	}
	// 0 = not seen, 1 = seen while still crossing the covering solid, 2 = seen on open ground
	seen := make([]uint8, n*n)
	type node struct{ i, j int }
	startOpen := !l.SolidP(want.X, want.Y)
	queue := []node{{c, c}}
	if startOpen {
		seen[c*n+c] = 2
	} else {
		seen[c*n+c] = 1
	}
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for h := 0; h < len(queue); h++ {
		cur := queue[h]
		state := seen[cur.j*n+cur.i]
		for _, d := range dirs {
			ni, nj := cur.i+d[0], cur.j+d[1]
			if ni < 0 || nj < 0 || ni >= n || nj >= n || seen[nj*n+ni] != 0 {
				continue
			}
			q := at(ni, nj)
			if q.X < 0 || q.Y < 0 || q.X > l.W || q.Y > l.H {
				continue
			}
			if !l.SolidP(q.X, q.Y) {
				if !g.hideSpotTaken(p, q) {
					return q
				}
				seen[nj*n+ni] = 2
				queue = append(queue, node{ni, nj})
				continue
			}
			if state == 1 && !l.hideWallAt(q) { // still digging out of what covers the entry spot
				seen[nj*n+ni] = 1
				queue = append(queue, node{ni, nj})
			}
		}
	}
	return l.NearestFree(want) // nothing walkable close by (should never happen)
}

// hideWallAt: the player's foot box at q touches a wall or the edge of the world.
func (l *Level) hideWallAt(q Vec) bool {
	box := Rect{q.X - 4, q.Y - 3, 8, 5}
	for _, w := range l.Walls {
		if box.Hits(wallFoot(w)) {
			return true
		}
	}
	return q.X < 0 || q.Y < 0 || q.X > l.W || q.Y > l.H
}

// hideDemo (-hidedemo x,y,in|out|busy): for screenshots of this file's rule. The player is put at
// x,y; at second 1 he presses E (the real interact: he goes into the nearest hiding place); with
// "out" he presses E again at second 2, with "busy" the partner (two players) first steps onto the
// spot he went in from.
var (
	hideDemo     string
	hideDemoDone bool
	hideDemoFrom Vec
)

func (g *Game) hideDemoStep() {
	if hideDemo == "" || g.mode != ModePlay || g.p == nil {
		return
	}
	var x, y float64
	var mode string
	if n, _ := fmt.Sscanf(strings.ReplaceAll(hideDemo, ",", " "), "%g %g %s", &x, &y, &mode); n < 2 {
		hideDemo = ""
		return
	}
	switch {
	case g.time < 1:
		g.p.pos = Vec{x, y}
		if o := g.otherPlayer(g.p); o != nil && mode != "busy" {
			o.pos = g.lvl.NearestFree(Vec{x + 40, y})
		}
	case !g.p.hidden && g.time < 2 && !hideDemoDone:
		hideDemoDone, hideDemoFrom = true, g.p.pos
		g.interact()
	case g.p.hidden && g.time >= 2 && (mode == "out" || mode == "busy"):
		if o := g.otherPlayer(g.p); o != nil && mode == "busy" {
			o.pos = hideDemoFrom
		}
		g.interact()
		println(fmt.Sprintf("HIDEDEMO went in from %.0f,%.0f, came out at %.0f,%.0f", hideDemoFrom.X, hideDemoFrom.Y, g.p.pos.X, g.p.pos.Y))
		hideDemo = ""
	}
}
