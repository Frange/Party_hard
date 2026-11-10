package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// The first two parts of the arrival (arrival.go has the island):
//   space (0 .. arrEarthT): a little shuttle undocks from the party space station (the windows flash with the
//     disco) and flies down to the Earth, which grows until it fills the picture; re-entry glow and a white flash.
//   the Earth (arrEarthT .. arrTravel): the shuttle lands on its retro rockets next to a helicopter rental pad,
//     the players run to the yellow rented helicopter (the very same one that will burn on the island) and it
//     takes off to the left, nose first.

const (
	arrLaunch   = 0.5             // the shuttle's engine fires and it leaves the station
	arrShipLand = arrEarthT + 0.9 // the shuttle touches the ground on the Earth
	arrBoard    = 2.1             // (Earth clock) the first player is in the helicopter; the second one 0.2 s later
	arrLift     = 2.45            // (Earth clock) the helicopter lifts off
	arrEarthGY  = 452.0           // the ground of the Earth part (the skids rest here)
	arrEarthHX  = 300.0           // the helicopter on the rental pad
	arrShipEX   = 760.0           // the shuttle on the Earth
)

// arrEarthRPM: the rotor of the helicopter on the Earth (still, spinning up while they run, full when it lifts).
func arrEarthRPM(t float64) float64 {
	if t < arrEarthT || t >= arrTravel {
		return 0
	}
	return clampF((t-arrEarthT-1.4)/0.9, 0, 1)
}

// arrEarthHeli: where the helicopter of the Earth part is (anchor) at time t.
func arrEarthHeli(t float64) (x, y float64) {
	f := arrSmooth((t - arrEarthT - arrLift) / 0.75)
	return arrEarthHX - 240*f*f, arrEarthGY - 18*arrK - 540*f*f
}

// arrEarthActor: player i on the Earth: running from the shuttle to the helicopter (vis false: in the shuttle or aboard).
func arrEarthActor(i, n int, t float64) (vis bool, x, y, ph float64) {
	u := t - arrEarthT - 0.2*float64(i)
	if u < 1.1 || u >= arrBoard {
		return false, 0, 0, 0
	}
	f := (u - 1.1) / (arrBoard - 1.1)
	x0 := arrShipEX - 40 + 18*float64(i)
	return true, math.Round(x0 + (arrEarthHX+24-x0)*f), arrEarthGY + 26 + 6*float64(i), u * 11
}

// arrEarthHead: where the head of player i is seen aboard (for his joke).
func arrEarthHead(i, n int, t float64) (x, topY float64) {
	hx, hy := arrEarthHeli(t)
	x = hx - 15*arrK
	if n == 2 {
		x += float64(i*2-1) * 18
	}
	return x, hy - 6*arrK - 22
}

// ---- drawing -------------------------------------------------------------------------------------

// fr is fillRect on whole pixels (crisp edges).
func fr(cv *ebiten.Image, x, y, w, h float64, col color.RGBA) {
	x0, y0 := math.Round(x), math.Round(y)
	fillRect(cv, x0, y0, math.Max(1, math.Round(x+w)-x0), math.Max(1, math.Round(y+h)-y0), col)
}

// arrDisc: a filled ellipse that only draws the rows on the screen (the Earth gets very big).
func arrDisc(cv *ebiten.Image, cx, cy, rx, ry float64, col color.RGBA) {
	y0, y1 := math.Max(-ry, -cy-1), math.Min(ry, screenH-cy+1)
	for y := math.Floor(y0); y <= y1; y++ {
		d := y / ry
		w := rx * math.Sqrt(math.Max(0, 1-d*d))
		l, r := math.Max(-2, cx-w), math.Min(screenW+2, cx+w)
		if r > l {
			fillRect(cv, math.Round(l), math.Round(cy+y), math.Round(r-l), 1, col)
		}
	}
}

func (g *Game) drawArrTravel(dst *ebiten.Image, a *arrivalState, cs [2]*CharSet, n int) {
	cv, t := a.canvas, a.t
	if t < arrEarthT {
		g.arrSpace(cv, t)
		dst.DrawImage(cv, nil)
		if t > 0.3 && t < 3.0 {
			al := clampF((t-0.3)/0.5, 0, 1) * (1 - clampF((t-2.4)/0.6, 0, 1))
			g.drawTextC(dst, "DE LA ESTACIÓN A LA TIERRA: VUELO DIRECTO (Y SIN FRENOS)", 480, 22, rgba(0xf4f0ff, uint8(255*al)), 2)
		}
		if t < 0.5 {
			fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*(1-t/0.5))))
		}
		if t > 3.0 { // into the air: a white flash
			fillRect(dst, 0, 0, screenW, screenH, rgba(0xffffff, uint8(255*clampF((t-3.0)/0.4, 0, 1))))
		}
		g.arrSkipHint(dst, a)
		return
	}
	g.arrEarth(cv, a, cs, n)
	dst.DrawImage(cv, nil)
	u := t - arrEarthT
	// the sign of the rental (on the screen: crisp letters)
	g.drawTextC(dst, "HELICÓPTEROS DE ALQUILER", 530, 206, rgb(0x2a2018), 2)
	g.drawTextC(dst, "Seguro no incluido. El fuego, tampoco.", 530, 234, rgb(0x8a1a14), 1)
	if u > 0.3 && u < 2.0 {
		al := clampF((u-0.3)/0.4, 0, 1) * (1 - clampF((u-1.5)/0.5, 0, 1))
		g.drawTextC(dst, "DE VUELTA EN LA TIERRA: TRANSBORDO AL HELICÓPTERO", 480, 22, rgba(0x14102a, uint8(255*al)), 2)
	}
	// the jokes of the Earth part
	for _, l := range arrLines(n) {
		if t < l.from || t >= l.to || cs[l.who] == nil || l.from >= arrTravel {
			continue
		}
		vis, x, y, _ := arrEarthActor(l.who, n, t)
		if vis {
			y -= 92
		} else {
			x, y = arrEarthHead(l.who, n, t)
			y -= 4
		}
		g.speechBubble(dst, math.Round(x), math.Round(y), l.text)
	}
	if u < 0.4 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0xffffff, uint8(255*(1-u/0.4))))
	}
	if u > 2.9 {
		fillRect(dst, 0, 0, screenW, screenH, rgba(0x000000, uint8(255*clampF((u-2.9)/0.3, 0, 1))))
	}
	g.arrSkipHint(dst, a)
}

// ---- space ----------------------------------------------------------------------------------------

// arrSpace: the station, the shuttle flying away from it and the Earth growing below.
func (g *Game) arrSpace(cv *ebiten.Image, t float64) {
	for y := 0.0; y < screenH; y += 4 {
		fillRect(cv, 0, y, screenW, 4, mixc(rgb(0x070a1c), rgb(0x02030a), y/screenH))
	}
	u := t / arrEarthT
	// a few big stars with shape (no dust of white dots)
	for i := 0; i < 9; i++ {
		x := 40 + jpHash(i, 1, 911)*880 - t*6*(0.5+jpHash(i, 3, 911))
		y := 30 + jpHash(i, 2, 911)*300
		g.drawStar(cv, math.Round(x), math.Round(y), 5+jpHash(i, 4, 911)*6, 0.8, i)
	}
	// the Earth: it grows as the shuttle comes down to it
	R := 330 + 820*u*u
	ecx, ecy := 680.0, 720.0
	arrDisc(cv, ecx, ecy, R*1.05, R*1.05, rgba(0x6ab0ff, 60)) // the air
	arrDisc(cv, ecx, ecy, R, R, rgb(0x1c5ca8))
	land := [][4]float64{{-0.42, -0.62, 0.30, 0.16}, {0.10, -0.80, 0.26, 0.10}, {0.36, -0.52, 0.22, 0.18}, {-0.12, -0.42, 0.16, 0.10}}
	for i, l := range land {
		col := rgb(0x3a9a4a)
		if i%2 == 1 {
			col = rgb(0x4aaa58)
		}
		arrDisc(cv, ecx+l[0]*R, ecy+l[1]*R, l[2]*R, l[3]*R, col)
	}
	drift := t * 0.02
	for i, cl := range [][4]float64{{-0.2, -0.72, 0.22, 0.03}, {0.3, -0.66, 0.18, 0.025}, {-0.5, -0.45, 0.16, 0.025}, {0.05, -0.5, 0.2, 0.03}} {
		arrDisc(cv, ecx+(cl[0]+drift*float64(i%2*2-1))*R, ecy+cl[1]*R, cl[2]*R, cl[3]*R, rgba(0xe8f4ff, 160))
	}
	// the station, left behind
	sx, sy, sk := 230-300*u*u, 170-220*u*u, 1-0.4*u
	g.arrStation(cv, sx, sy, sk, t)
	// the shuttle
	px, py := sx+64*sk, sy
	x, y, k := px+16*clampF(t/arrLaunch, 0, 1), py, 0.8*sk
	flame := 0.0
	if t >= arrLaunch {
		p := arrSmooth((t - arrLaunch) / (3.2 - arrLaunch))
		x = lerp(px+16, 560, p)
		y = lerp(py, 380, p) - math.Sin(p*math.Pi)*40
		k = 0.8 - 0.4*p
		flame = 1
	}
	glow := clampF((t-2.4)/0.6, 0, 1)
	arrShip(cv, x, y, k, flame, 0, glow, false, t)
}

// arrStation: the party space station seen from outside: a module with flashing disco windows, a ring, a truss and
// two pairs of solar panels.
func (g *Game) arrStation(cv *ebiten.Image, x, y, k, t float64) {
	fr(cv, x-160*k, y-3*k, 320*k, 6*k, rgb(0x8890a0)) // the truss
	for _, side := range []float64{-1, 1} {
		px := x + side*118*k - 38*k
		for _, py := range []float64{y - 52*k, y + 12*k} {
			fr(cv, px, py, 76*k, 40*k, rgb(0x1a2f6a))
			for q := 1.0; q < 4; q++ {
				fr(cv, px+q*19*k, py, 1, 40*k, rgb(0x4a6ac0))
			}
			fr(cv, px, py+20*k, 76*k, 1, rgb(0x4a6ac0))
		}
		fr(cv, px+36*k, y-12*k, 4*k, 24*k, rgb(0x6a7080))
	}
	ringAt(cv, math.Round(x), math.Round(y), math.Round(58*k), math.Max(2, math.Round(5*k)), rgb(0x9aa0ae))
	filledEllipse(cv, math.Round(x), math.Round(y), math.Round(40*k), math.Round(26*k), rgb(0xc8ccd8))
	fr(cv, x-40*k, y-10*k, 80*k, 20*k, rgb(0xb0b6c4))
	for i := 0; i < 5; i++ { // the party is still going on in there
		col := rgb(partyColors[(i+int(t*6))%len(partyColors)])
		fr(cv, x-30*k+float64(i)*13*k, y-5*k, 8*k, 8*k, col)
	}
	fr(cv, x+40*k, y-6*k, 18*k, 12*k, rgb(0x8890a0)) // the docking port
	if int(t*2)%2 == 0 {
		fillDisc(cv, math.Round(x), math.Round(y-28*k), math.Max(2, math.Round(3*k)), rgb(0xff4040))
	}
}

// arrShipImgs: the shuttle of the arrival, cached per size step (see arrShipImg).
var arrShipImgs = map[int]*ebiten.Image{}

// arrShipImg: the shuttle (vessels.go genShip, the same drawer as the ships of the space station:
// white with a red stripe) at scale k, rounded to 1/20 steps and shrunk with the dominant-colour
// scaler, so it stays crisp pixel art at every size. Returns the image and the scale it was made at.
func arrShipImg(k float64) (*ebiten.Image, float64) {
	q := int(math.Round(k * 20))
	if q < 4 {
		q = 4
	}
	if im := arrShipImgs[q]; im != nil {
		return im, float64(q) / 20
	}
	pb := genShip(rgb(0xe4e8ee), rgb(0xd04030), false, crPlain).outlinedSel(0.42)
	if q != 20 {
		pb = pb.scaled(float64(q) / 20)
	}
	im := pb.img()
	arrShipImgs[q] = im
	return im, float64(q) / 20
}

// arrShip: the little shuttle, pointing right, centred on (x, y); k is its scale (1 = 96 px long). back: the
// main engine; down: the retro rockets under it; glow: the heat of the re-entry; legs: landing legs out.
func arrShip(cv *ebiten.Image, x, y, k, back, down, glow float64, legs bool, t float64) {
	im, k := arrShipImg(k)
	if glow > 0 {
		filledEllipse(cv, math.Round(x+14*k), math.Round(y), math.Round(62*k), math.Round(30*k), rgba(0xff7a20, uint8(120*glow)))
		filledEllipse(cv, math.Round(x+30*k), math.Round(y), math.Round(36*k), math.Round(22*k), rgba(0xffd060, uint8(140*glow)))
	}
	if back > 0 { // the cyan jet of the main engine, flickering
		fl := math.Round((26 + 10*math.Abs(math.Sin(t*40))) * k * back)
		fr(cv, x-47*k-fl, y-5*k, fl, 10*k, rgb(0x2ab8e8))
		fr(cv, x-47*k-fl*0.75, y-3*k, fl*0.75, 6*k, rgb(0x7ae8ff))
		fr(cv, x-47*k-fl*0.4, y-1*k, fl*0.4, 2*k, rgb(0xe8fcff))
	}
	if down > 0 {
		fl := (22 + 10*math.Abs(math.Sin(t*37))) * k * down
		for _, dx := range []float64{-26, 18} {
			fr(cv, x+dx*k, y+9*k, 10*k, fl, rgb(0xf0a030))
			fr(cv, x+dx*k+3*k, y+9*k, 4*k, fl*0.6, rgb(0xfff0a0))
		}
	}
	if legs {
		for _, dx := range []float64{-30, 26} {
			fr(cv, x+dx*k, y+8*k, 3*k, 18*k, rgb(0x6a7080))
			fr(cv, x+dx*k-4*k, y+24*k, 11*k, 3*k, rgb(0x4a5060))
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(math.Round(x-48*k), math.Round(y-22*k))
	cv.DrawImage(im, op)
}

// ---- the Earth ------------------------------------------------------------------------------------

func (g *Game) arrEarth(cv *ebiten.Image, a *arrivalState, cs [2]*CharSet, n int) {
	t := a.t
	u := t - arrEarthT
	for y := 0.0; y < 300; y += 4 { // the sky
		fillRect(cv, 0, y, screenW, 4, mixc(rgb(0x6ab4ea), rgb(0xd8eef8), y/300))
	}
	filledEllipse(cv, 160, 90, 60, 14, rgba(0xffffff, 200)) // two clouds
	filledEllipse(cv, 820, 130, 70, 16, rgba(0xffffff, 190))
	fillRect(cv, 0, 300, screenW, 52, rgb(0x2f8cc0)) // the sea, far away
	for i := 0; i < 14; i++ {
		x := math.Mod(jpHash(i, 1, 913)*screenW+u*12, screenW)
		fr(cv, x, 306+jpHash(i, 2, 913)*40, 18, 2, rgba(0x9ad8f0, 170))
	}
	for y := 352.0; y < screenH; y += 4 { // the concrete of the heliport
		fillRect(cv, 0, y, screenW, 4, mixc(rgb(0xb4b2a8), rgb(0x8e8c84), (y-352)/188))
	}
	fillRect(cv, 0, 352, screenW, 3, rgb(0x6e6c66))
	for x := 20.0; x < screenW; x += 90 { // the joints of the slabs
		fillRect(cv, x, 356, 2, screenH-356, rgba(0x6e6c66, 120))
	}
	// the two pads
	filledEllipse(cv, arrEarthHX, arrEarthGY+6, 150, 34, rgb(0x6f7378))
	filledEllipse(cv, arrShipEX, arrEarthGY+6, 100, 24, rgb(0x6f7378))
	// the sign of the rental: a board on two posts (its letters are drawn on the screen)
	fillRect(cv, 404, 252, 6, 100, rgb(0x5a4a3a))
	fillRect(cv, 650, 252, 6, 100, rgb(0x5a4a3a))
	fillRect(cv, 376, 194, 308, 62, rgb(0x3a2a1c))
	fillRect(cv, 380, 198, 300, 54, rgb(0xf2e2b0))
	// the shuttle comes down on its retro rockets and stays there
	sy := arrEarthGY - 52 // its feet (27 px below the centre at scale 1) on the ground at scale 2
	down := 0.0
	if u < 0.9 {
		sy = lerp(-80, arrEarthGY-52, arrSmooth(u/0.9))
		down = 1
	}
	if u >= 0.9 && u < 1.3 { // the dust of the landing
		f := (u - 0.9) / 0.4
		for k := 0; k < 6; k++ {
			side := float64(k%2*2 - 1)
			filledEllipse(cv, arrShipEX+side*(40+f*(50+float64(k)*12)), arrEarthGY+4-f*8, 20+f*24, 10+f*8, rgba(0xc8c0a8, uint8(170*(1-f))))
		}
	}
	arrShip(cv, arrShipEX, sy, 2, 0, down, 0, u > 0.6, t) // whole scale: crisp pixels, as big as the helicopter
	// the helicopter (the very same yellow one), with whoever is aboard
	side, _ := arrHeliImgs()
	hx, hy := arrEarthHeli(t)
	alt := arrEarthGY - 18*arrK - hy
	filledEllipse(cv, hx+alt*0.3, arrEarthGY+4, math.Max(20, 40*arrK-10-alt*0.25), 30, rgba(0x000000, 80))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(arrK, arrK)
	op.GeoM.Translate(math.Round(hx-42*arrK), math.Round(hy-35*arrK))
	cv.DrawImage(side, op)
	for i := 0; i < n; i++ {
		if vis, _, _, _ := arrEarthActor(i, n, t); vis || u < 1.5 {
			continue
		}
		x, top := arrEarthHead(i, n, t)
		g.arrHead(cv, cs[i], x, top, DirLeft, sceneTint(i, cs[0], cs[i]))
	}
	arrRotor(cv, hx-arrK, hy-25*arrK, arrEarthRPM(t), a.rot)
	if arrEarthRPM(t) > 0.1 {
		fillRect(cv, hx+50*arrK, hy-10*arrK-14, 3, 8+28*math.Abs(math.Cos(a.rot*1.7)), rgba(0x22222a, 170))
	}
	// the players run from the shuttle to the helicopter
	for i := n - 1; i >= 0; i-- {
		vis, x, y, ph := arrEarthActor(i, n, t)
		if !vis || cs[i] == nil {
			continue
		}
		g.sceneChar(cv, cs[i], AnimRun, DirLeft, ph, x, y, 2, 1, 0, 0, sceneTint(i, cs[0], cs[i]))
	}
}
