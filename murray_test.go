package main

import (
	"fmt"
	"testing"
)

// murrayCheck is an independent check of Murray's spot: outdoors, away from buildings and
// covering nothing. It returns "" when the spot is fine.
func murrayCheck(d *LevelDef, in PropInst) string {
	spr := propSpriteRect(in)
	lv := &Level{Def: d}
	for _, p := range []Vec{{spr.X, spr.Y}, {spr.X + spr.W, spr.Y}, {spr.X, spr.Y + spr.H}, {spr.X + spr.W, spr.Y + spr.H}, spr.Center(), {in.X, in.Y}} {
		if !lv.Outdoor(p) {
			return fmt.Sprintf("indoors at %v (floor %q)", p, murrayFloorAt(d, p.X, p.Y))
		}
	}
	for _, f := range d.Floors {
		if floorIndoor(f.Kind) && f.Kind != "deck" && f.R.Hits(spr.Grow(8)) {
			return fmt.Sprintf("touches a building floor %q %v", f.Kind, f.R)
		}
	}
	for _, w := range d.Walls {
		if w.R.Hits(spr) {
			return fmt.Sprintf("on a wall %v", w.R)
		}
	}
	for _, dr := range d.Doors {
		if dr.R.Grow(24).Hits(spr) {
			return fmt.Sprintf("in front of a door %v", dr.R)
		}
	}
	if dist(Vec{in.X, in.Y}, d.Start) < 60 {
		return "on the start"
	}
	for _, pf := range d.Performers {
		if spr.Grow(8).Has(pf.Pos.X, pf.Pos.Y) {
			return "on a performer"
		}
	}
	for _, r := range d.Dance {
		if r.Hits(spr) {
			return "on the dance floor"
		}
	}
	foot := instFoot(in, propKinds["murray"])
	for _, o := range d.Props {
		if o == in {
			continue
		}
		if o.Kind == "murray" {
			return "two Murrays"
		}
		k := propKinds[o.Kind]
		if k == nil {
			continue
		}
		if propSpriteRect(o).Hits(spr) {
			return fmt.Sprintf("covers / is covered by %s at %.0f,%.0f", o.Kind, o.X, o.Y)
		}
		if instFoot(o, k).Hits(foot) {
			return fmt.Sprintf("stands on the footprint of %s", o.Kind)
		}
	}
	if !inWalkable(d, foot) {
		return "off the walkable ground"
	}
	return ""
}

// TestMurrayAlwaysOutdoorsCoveringNothing: in every level where Murray stands (campaign,
// random pool and many seeds of the generated venues) he is outdoors and covers nothing.
func TestMurrayAlwaysOutdoorsCoveringNothing(t *testing.T) {
	check := func(name string, d *LevelDef) int {
		n := 0
		for _, p := range d.Props {
			if p.Kind != "murray" {
				continue
			}
			n++
			if why := murrayCheck(d, p); why != "" {
				t.Errorf("%s: Murray at %.0f,%.0f: %s", name, p.X, p.Y, why)
			}
		}
		if n > 1 {
			t.Errorf("%s: %d Murrays", name, n)
		}
		return n
	}
	found := 0
	for i, d := range builtinLevels() {
		n := check(fmt.Sprintf("level %d %s", i+1, d.Title), d)
		if i < campaignN {
			found += n
		}
	}
	if found != len(murrayLevels) {
		t.Errorf("Murray is in %d campaign parties, want %d", found, len(murrayLevels))
	}
	// the booths, items and tables added when the party loads do not cover him either
	for _, d := range builtinLevels()[:campaignN] {
		if !murrayLevels[d.Title] {
			continue
		}
		l := buildLevel(d)
		var mur *Prop
		for _, p := range l.Props {
			if p.Name == "murray" {
				mur = p
			}
		}
		if mur == nil {
			t.Errorf("%s: Murray is lost when the party loads", d.Title)
			continue
		}
		mr := propSpriteRect(PropInst{Kind: "murray", X: mur.X, Y: mur.Y, Var: mur.Var})
		for _, p := range l.Props {
			if p == mur {
				continue
			}
			if r := propSpriteRect(PropInst{Kind: p.Name, X: p.X, Y: p.Y, Var: p.Var, Scale: p.scale}); r.Hits(mr) {
				t.Errorf("%s: %s at %.0f,%.0f covers Murray", d.Title, p.Name, p.X, p.Y)
			}
		}
	}
	// generated venues with other seeds (random mode / regenerated layouts)
	placed, tried := 0, 0
	for _, e := range campaignLevels {
		if e.kind == "" || !murrayLevels[e.title] {
			continue
		}
		for s := int64(0); s < 12; s++ {
			d := genNamed(e.kind, e.seed+s*101, e.title)
			if d == nil {
				continue
			}
			addMurray(d)
			tried++
			placed += check(fmt.Sprintf("%s seed %d", e.title, e.seed+s*101), d)
		}
	}
	t.Logf("generated: Murray placed in %d of %d layouts", placed, tried)
	// a Murray put by hand in a bad spot (inside the first room floor) is moved out
	for _, d := range builtinLevels()[:campaignN] {
		if !murrayLevels[d.Title] {
			continue
		}
		for _, f := range d.Floors {
			if floorIndoor(f.Kind) && f.Kind != "deck" {
				c := f.R.Center()
				var props []PropInst
				for _, p := range d.Props {
					if p.Kind != "murray" {
						props = append(props, p)
					}
				}
				d.Props = append(props, PropInst{Kind: "murray", X: c.X, Y: c.Y})
				addMurray(d)
				check(d.Title+" (moved out)", d)
				break
			}
		}
	}
}
