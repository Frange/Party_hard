package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// PHONE ART (v1.4.0): the ringing phone of the HUD (somebody calls the police or the technicians, phonehud.go) and
// the marker over the phones of the level (draw.go drawPhoneMarkers) are a handset in the colour of its booth (red
// or yellow; blue only without booths) with a dark outline and shading, like a pixel-art phone icon: ear cup on top,
// the grip bending down, the mouth cup below and the coiled cord. The sound dashes beside it while it rings are drawn by the HUD (in the colour of who is called).
// Painted once from a pixel map (sharp at its real size), drawn at whole scales.
// ---------------------------------------------------------------------------

// phoneTint: the colours of the handset (outline, highlight, light, body, shade). The handset takes the colour of the
// booth it rings from (owner's rule): red like the red phone booths, yellow from the yellow booth of the technicians;
// blue only when the level has no booth at all.
type phoneTint struct{ K, H, L, B, D uint32 }

const (
	phoneRed = iota
	phoneYellow
	phoneBlue
	phoneTints
)

var phoneTintOf = [phoneTints]phoneTint{
	phoneRed:    {0x300a0a, 0xffc8c0, 0xf06a5a, 0xc82828, 0x8a1a1a}, // the red booth (props.go "phone")
	phoneYellow: {0x3a2a04, 0xfff4b0, 0xfce070, 0xf0c820, 0xa8800c}, // the yellow booth (techs.go "techphone")
	phoneBlue:   {0x0e1430, 0xc4dcff, 0x6aa2f0, 0x3a6cd8, 0x2244a0},
}

// phonePal: the colours of the pixel maps for one tint (the cord is always grey).
func phonePal(t int) map[byte]color.RGBA {
	p := phoneTintOf[t]
	return map[byte]color.RGBA{'K': c(p.K), 'H': c(p.H), 'L': c(p.L), 'B': c(p.B), 'D': c(p.D), 'G': c(0x8a90a0), 'g': c(0x4a4e5a)}
}

// the handset of the HUD, 16x25
var phoneHandsetMap = []string{
	"........KKKK....",
	"......KKLLLBK...",
	".....KLHHLLBBK..",
	"....KLHLLLBBBDK.",
	"....KLLLLBBBBDK.",
	".....KKKLBBBBDK.",
	"........KBBBBDK.",
	"........KLBBDDK.",
	"........KLBBDK..",
	".......KLBBBDK..",
	".......KLBBDDK..",
	"......KLBBBDK...",
	".....KLBBBDDK...",
	"..KKKLLBBBDK....",
	".KLHHLBBBDDK....",
	"KLHLLLBBBDK.....",
	"KLLLBBBBDDK.....",
	"KBBBBBBDDK......",
	".KDDDDDDK.......",
	"..KKKKKK........",
	"....KGK.........",
	"...KGgK.........",
	"....KGGK........",
	"...KgGK.........",
	"....KGK.........",
}

// the small handset of the markers over the phones, 12x5
var phoneMarkMap = []string{
	".KKKKKKKKKK.",
	"KLHHLLLLLBDK",
	"KLBKKKKKKBDK",
	"KBDK....KBDK",
	".KK......KK.",
}

func pixFromMap(rows []string, tint int) *pixBuf {
	pal := phonePal(tint)
	b := newPix(len(rows[0]), len(rows))
	for y, r := range rows {
		for x := 0; x < len(r); x++ {
			if col, ok := pal[r[x]]; ok {
				b.set(x, y, col)
			}
		}
	}
	return b
}

var phoneHandsetImg, phoneMarkImg [phoneTints]*ebiten.Image

// phoneHandset: the big handset of the HUD in a tint (built once per tint).
func phoneHandset(tint int) *ebiten.Image {
	if phoneHandsetImg[tint] == nil {
		phoneHandsetImg[tint] = pixFromMap(phoneHandsetMap, tint).img()
	}
	return phoneHandsetImg[tint]
}

// phoneMark: the small handset of the markers in a tint (built once per tint).
func phoneMark(tint int) *ebiten.Image {
	if phoneMarkImg[tint] == nil {
		phoneMarkImg[tint] = pixFromMap(phoneMarkMap, tint).img()
	}
	return phoneMarkImg[tint]
}

// phoneTintForCall: the tint of the ringing handset for a call of kind k (callingKind: 1 the red booths, 2 the
// yellow booth of the technicians); blue only when the level has no booth.
func phoneTintForCall(l *Level, k int) int {
	if k == 2 {
		return phoneYellow
	}
	if l != nil {
		for _, pr := range l.Props {
			if pr.K != nil && (pr.K.Phone || pr.Name == "phone") {
				return phoneRed
			}
		}
	}
	return phoneBlue
}

// phoneTintForProp: the tint of the handset over a booth: its own colour.
func phoneTintForProp(pr *Prop) int {
	if pr != nil && pr.K != nil && pr.K.TechPhone {
		return phoneYellow
	}
	return phoneRed
}
