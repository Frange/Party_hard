package main

import "math"

// clearStage keeps the DJ and the musicians visible in generated levels: any prop that would be
// drawn in front of them (further south) and hides part of them is moved sideways (speakers) or
// removed. It works on the nominal sprite box (k.W x k.H above the anchor), which is a bit larger
// than the opaque pixels, so it is slightly conservative.
func clearStage(d *LevelDef) {
	type tgt struct {
		r    Rect
		sort float64
		dj   bool
	}
	var ts []tgt
	for _, in := range d.Props {
		if in.Kind != "dj" {
			continue
		}
		if k := propKinds[in.Kind]; k != nil {
			ts = append(ts, tgt{Rect{in.X - float64(k.W)/2, in.Y - float64(k.H), float64(k.W), float64(k.H)}, in.Y + k.SortOff, true})
		}
	}
	for _, pf := range d.Performers {
		ts = append(ts, tgt{Rect{pf.Pos.X - 8, pf.Pos.Y - 30, 16, 30}, pf.Pos.Y, false})
	}
	if len(ts) == 0 {
		return
	}
	box := func(in PropInst, k *PropKind) Rect {
		sc := in.Scale
		if sc <= 0 {
			sc = 1
		}
		return Rect{in.X - float64(k.W)/2*sc, in.Y - float64(k.H)*sc, float64(k.W) * sc, float64(k.H) * sc}
	}
	covers := func(in PropInst, k *PropKind, t tgt) bool {
		if in.Y+k.SortOff <= t.sort {
			return false
		}
		return stageInter(t.r, box(in, k))/(t.r.W*t.r.H) > 0.06
	}
	out := d.Props[:0]
	for _, in := range d.Props {
		k := propKinds[in.Kind]
		if k == nil || in.Kind == "dj" || in.Kind == "truss" {
			out = append(out, in)
			continue
		}
		bad := false
		for _, t := range ts {
			if covers(in, k, t) {
				bad = true
				break
			}
		}
		if bad && in.Kind == "speaker" {
			// slide the speaker out of the way, away from the nearest target
			for _, t := range ts {
				if !covers(in, k, t) {
					continue
				}
				side := 1.0
				if in.X < t.r.X+t.r.W/2 {
					side = -1
				}
				w := float64(k.W)
				in.X = math.Round(t.r.X + t.r.W/2 + side*(t.r.W/2+w/2+6))
			}
			bad = false
			for _, t := range ts {
				if covers(in, k, t) {
					bad = true
				}
			}
		}
		if !bad {
			out = append(out, in)
		}
	}
	d.Props = out
}

func stageInter(a, b Rect) float64 {
	w, h := rectOv(a, b)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

// hidesStage reports whether a prop of kind k anchored at p would be drawn over the DJ or a musician.
func (l *Level) hidesStage(k *PropKind, p Vec) bool {
	b := Rect{p.X - float64(k.W)/2, p.Y - float64(k.H), float64(k.W), float64(k.H)}
	for _, in := range l.Def.Props {
		if in.Kind != "dj" {
			continue
		}
		if dk := propKinds["dj"]; dk != nil && p.Y+k.SortOff > in.Y+dk.SortOff {
			r := Rect{in.X - float64(dk.W)/2, in.Y - float64(dk.H), float64(dk.W), float64(dk.H)}
			if stageInter(r, b)/(r.W*r.H) > 0.06 {
				return true
			}
		}
	}
	for _, pf := range l.Def.Performers {
		if p.Y+k.SortOff <= pf.Pos.Y {
			continue
		}
		r := Rect{pf.Pos.X - 8, pf.Pos.Y - 30, 16, 30}
		if stageInter(r, b)/(r.W*r.H) > 0.06 {
			return true
		}
	}
	return false
}
