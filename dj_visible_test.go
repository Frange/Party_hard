package main

import (
	"fmt"
	"testing"
)

// The DJ and the musicians must never be hidden by a prop that is drawn after them (a taller
// sprite further south). The drawn rectangle is the box of the opaque pixels of the sprite.

type djKey struct {
	k *PropKind
	v int
}

var drawnCache = map[djKey]Rect{}

// propDrawnRel: bounding box of the opaque pixels of the whole sprite, relative to the anchor.
func propDrawnRel(k *PropKind, v int) Rect {
	key := djKey{k, v}
	if r, ok := drawnCache[key]; ok {
		return r
	}
	r := Rect{-float64(k.W) / 2, -float64(k.H), float64(k.W), float64(k.H)}
	var b *pixBuf
	var ax, ay float64
	if k.Native != nil {
		b = k.Native[v%len(k.Native)]
		ax, ay = k.NAX, k.NAY
	} else if k.Draw != nil {
		b = newPix(k.W+2, k.H+2)
		b.ox, b.oy = 1, 1
		k.Draw(b, v)
		ax, ay = float64(k.W)/2+1, float64(k.H)+1
	}
	if b != nil {
		x0, y0, x1, y1 := 1e9, 1e9, -1e9, -1e9
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if b.p[(y*b.w+x)*4+3] < 128 {
					continue
				}
				fx, fy := float64(x)-ax, float64(y)-ay
				if fx < x0 {
					x0 = fx
				}
				if fx+1 > x1 {
					x1 = fx + 1
				}
				if fy < y0 {
					y0 = fy
				}
				if fy+1 > y1 {
					y1 = fy + 1
				}
			}
		}
		if x1 > x0 {
			r = Rect{x0, y0, x1 - x0, y1 - y0}
		}
	}
	drawnCache[key] = r
	return r
}

func interArea(a, b Rect) float64 {
	w, h := rectOv(a, b)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

// djCoverExceptions: kinds allowed to be drawn over the DJ area, with the reason.
var djCoverExceptions = map[string]string{
	"dj":    "the DJ's own desk is drawn in front of him on purpose",
	"truss": "lighting rig hanging above the stage: thin bars, overlay",
}

const djMaxCover = 0.10

// djGenSeeds: seeds checked per generated kind (seed = s*131).
const djGenSeeds = 24

// djCoverIssues lists the props drawn after the DJ / performers that cover more than djMaxCover of them.
func djCoverIssues(l *Level) []string {
	var out []string
	type tgt struct {
		name string
		r    Rect
		sort float64
	}
	var ts []tgt
	for _, p := range l.Props {
		if p.Name == "dj" {
			rr := propDrawnRel(p.K, p.Var)
			ts = append(ts, tgt{"dj", Rect{p.X + rr.X, p.Y + rr.Y, rr.W, rr.H}, p.Sort})
		}
	}
	for _, pf := range l.Def.Performers {
		ts = append(ts, tgt{"perf-" + pf.Role, Rect{pf.Pos.X - 8, pf.Pos.Y - 30, 16, 30}, pf.Pos.Y})
	}
	for _, t := range ts {
		area := t.r.W * t.r.H
		for _, p := range l.Props {
			if p.gone || djCoverExceptions[p.Name] != "" || p.Sort <= t.sort || p.K == nil {
				continue
			}
			rr := propDrawnRel(p.K, p.Var)
			sc := p.sc()
			pr := Rect{p.X + rr.X*sc, p.Y + rr.Y*sc, rr.W * sc, rr.H * sc}
			if c := interArea(t.r, pr) / area; c > djMaxCover {
				out = append(out, fmt.Sprintf("%s at (%.0f,%.0f) covered %.0f%% by %s at (%.0f,%.0f)", t.name, t.r.X+t.r.W/2, t.r.Y+t.r.H, c*100, p.Name, p.X, p.Y))
			}
		}
	}
	return out
}

func TestDJAndBandVisible(t *testing.T) {
	snapshotPropKinds(t)
	if st := loadAssetStore(); st != nil {
		applyObjects(st.Objs)
	}
	fitAllFootprints()
	defs := map[string]*LevelDef{}
	for i, d := range builtinLevels() {
		defs[fmt.Sprintf("built%02d-%s", i, d.Title)] = d
	}
	for i, d := range classicLevels() {
		defs[fmt.Sprintf("classic%02d-%s", i, d.Title)] = d
	}
	defs["jurassic"] = jurassicLevel()
	for _, k := range ovlKinds {
		for s := int64(1); s <= djGenSeeds; s++ {
			if d := genNamed(k, s*131, ""); d != nil {
				defs[fmt.Sprintf("%s-s%d", k, s)] = d
			}
		}
	}
	// Every level is strict, generated ones included: the generators keep the stage clear (genclear.go).
	bad := 0
	for n, d := range defs {
		for _, is := range djCoverIssues(buildLevel(d)) {
			bad++
			t.Errorf("%s: %s", n, is)
		}
	}
	if bad == 0 && len(defs) < 10 {
		t.Fatalf("too few levels checked: %d", len(defs))
	}
}
