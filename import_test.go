package main

import (
	"image/color"
	"testing"
)

// a red square on a blue-ish gradient background must come out as a red figure on transparency
func TestImportProcess(t *testing.T) {
	src := newPix(400, 600)
	for y := 0; y < src.h; y++ {
		for x := 0; x < src.w; x++ {
			i := (y*src.w + x) * 4
			src.p[i], src.p[i+1], src.p[i+2], src.p[i+3] = uint8(150+x/10), uint8(180+y/20), 230, 255
			if x > 150 && x < 250 && y > 100 && y < 500 {
				src.p[i], src.p[i+1], src.p[i+2] = 200, 30, 30
			}
		}
	}
	d := &ImportDlg{src: src, tw: 24, th: 44, tol: 40, nCols: 8, fitBox: true, bgAuto: true, removeBG: true}
	d.process()
	opaque, red := 0, 0
	for i := 0; i < d.res.w*d.res.h; i++ {
		if d.res.p[i*4+3] > 0 {
			opaque++
			if d.res.p[i*4] > 150 && d.res.p[i*4+1] < 80 {
				red++
			}
		}
	}
	if opaque == 0 || red < opaque*9/10 {
		t.Fatalf("background not removed: %d opaque, %d red", opaque, red)
	}
	// the figure is fitted to the full height and sits on the bottom edge
	if d.res.p[((d.th-1)*d.tw+d.tw/2)*4+3] == 0 || d.res.p[(d.tw/2)*4+3] == 0 {
		t.Fatal("figure does not fill the height")
	}
	_ = color.RGBA{}
}
