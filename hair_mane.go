package main

import (
	"image/color"
	"math"
)

// HairBaldMane ("Calvo con melena", the barber's shop): bald on top, with long hair from the sides and the back of the
// head that falls all the way down to the floor. The head (chars.go) paints the bald scalp and the hair around the ears
// and the nape; drawMane paints the long part: two curtains behind the body seen from the front, one at the back seen
// from the side, and a whole sheet over the back seen from behind. The tips swing with the pose (sway, lean and frame),
// so it moves when the character walks or dances.

// maneTip: how far (px) the tips of the hair swing for this pose.
func maneTip(p Pose) float64 {
	return p.Sway*1.2 - p.Lean*0.8 + 0.8*math.Sin(float64(p.F)*math.Pi/2)
}

// maneBottom: the last row of the hair: the floor, under the feet.
func maneBottom(footRow float64) int {
	return minI(int(footRow)+1, charH-1)
}

func drawMane(b *pixBuf, vw int, p Pose, hxp, hyp, rx, footRow float64, col color.RGBA) {
	dark, light := shade(col, 0.78), shade(col, 1.25)
	y0 := int(math.Floor(hyp))
	y1 := maneBottom(footRow)
	if y1 <= y0 {
		return
	}
	tip := maneTip(p)
	strand := func(y, x0, w int, f float64) {
		for i := 0; i < w; i++ {
			c := col
			switch {
			case i == 0 || i == w-1:
				c = dark
			case (i+int(f*7))%3 == 0:
				c = light // a shiny streak along the hair
			}
			if y == y1 && (x0+i)%2 == 1 { // ragged tips
				continue
			}
			b.set(x0+i, y, c)
		}
	}
	for y := y0; y <= y1; y++ {
		f := float64(y-y0) / float64(y1-y0)
		swing := tip * f * f
		switch vw {
		case vFront: // two curtains at both sides, flaring a little towards the floor
			flare := 2.2 * f
			xl := int(math.Floor(hxp - rx - 1.5 - flare + swing))
			xr := int(math.Floor(hxp + rx - 0.5 + flare + swing))
			w := 2
			if f > 0.55 {
				w = 3
			}
			strand(y, xl-(w-2), w, f)
			strand(y, xr, w, f)
		case vBack: // a sheet that covers the back and spreads out at the bottom
			half := rx + 0.5 + 2.5*f
			x0 := int(math.Floor(hxp - half + swing))
			x1 := int(math.Floor(hxp + half + swing))
			strand(y, x0, x1-x0+1, f)
		default: // seen from the side: it hangs down the back (the left of the sprite) and trails behind
			x0 := int(math.Floor(hxp - rx - 0.5 - 1.5*f + swing))
			w := 3
			if f > 0.5 {
				w = 4
			}
			strand(y, x0, w, f)
		}
	}
}

// maneScalp: the parts of the head that keep hair with HairBaldMane (the sides, the ears and the nape); the top is bald.
func maneScalp(vw int, dx, dy, rx, ry float64) bool {
	switch vw {
	case vBack:
		return dy > -ry*0.1
	case vSide:
		return dx < -rx*0.15 && dy > -ry*0.2
	}
	return math.Abs(dx) > rx*0.7 && dy > -ry*0.15
}
