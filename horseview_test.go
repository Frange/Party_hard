package main

import "testing"

// opaque bounding box of a pixBuf
func pixBBox(b *pixBuf) (w, h int) {
	x0, y0, x1, y1 := b.w, b.h, -1, -1
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] > 40 {
				if x < x0 {
					x0 = x
				}
				if x > x1 {
					x1 = x
				}
				if y < y0 {
					y0 = y
				}
				if y > y1 {
					y1 = y
				}
			}
		}
	}
	return x1 - x0 + 1, y1 - y0 + 1
}

// The horse seen from behind must not look bigger than seen head on.
func TestHorseBackNotBiggerThanFront(t *testing.T) {
	for coat := 0; coat < 3; coat++ {
		f, k := newPix(32, 58), newPix(32, 58)
		horseView(f, coat, true, false)
		horseView(k, coat, false, false)
		fw, fh := pixBBox(f)
		kw, kh := pixBBox(k)
		if float64(kw) > float64(fw)*1.15 {
			t.Errorf("coat %d: back width %d > front %d +15%%", coat, kw, fw)
		}
		if float64(kh) > float64(fh)*1.15 {
			t.Errorf("coat %d: back height %d > front %d +15%%", coat, kh, fh)
		}
		t.Logf("coat %d front %dx%d back %dx%d", coat, fw, fh, kw, kh)
	}
	f, k := bullFreeBuf(true, 0), bullFreeBuf(false, 0)
	fw, _ := pixBBox(f)
	kw, _ := pixBBox(k)
	if float64(kw) > float64(fw)*1.15 {
		t.Errorf("bull back width %d > front %d +15%%", kw, fw)
	}
	t.Logf("bull front w%d back w%d", fw, kw)
}

// Seen head on the horse's head must be narrow and long: no more than ~40% of the chest width.
func TestHorseFrontHeadNarrow(t *testing.T) {
	for coat := 0; coat < 3; coat++ {
		b := newPix(32, 58)
		horseView(b, coat, true, false)
		rowW := func(y int) int {
			n := 0
			for x := 0; x < b.w; x++ {
				if b.p[(y*b.w+x)*4+3] > 40 {
					n++
				}
			}
			return n
		}
		head := 0
		for y := 25; y <= 36; y++ { // face rows: neck is not counted, so use the blaze/head columns only
			w := 0
			for x := 0; x < b.w; x++ {
				if b.p[(y*b.w+x)*4+3] > 40 && x >= 11 && x <= 19 {
					w++
				}
			}
			if w > head {
				head = w
			}
		}
		chest := rowW(38)
		neckTop := rowW(21)
		if neckTop > 10 {
			t.Errorf("coat %d: poll/ears row %d px wide, want <=10", coat, neckTop)
		}
		if float64(neckTop) > 0.4*float64(chest)+0.5 {
			t.Errorf("coat %d: head/poll %d px vs chest %d px (>40%%)", coat, neckTop, chest)
		}
		t.Logf("coat %d poll %d chest %d headcols %d", coat, neckTop, chest, head)
	}
}
