package main

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Import window: turns a big picture (a photo of a friend, a drawing...) into
// pixel art of the size of whatever is being edited. You crop what you want,
// the background can be knocked out, and the result is shrunk, reduced to a
// few colours and handed to the normal editor to be retouched by hand.
// ---------------------------------------------------------------------------

const (
	impX, impY, impW, impH = 16, 52, 720, 652 // where the source picture is shown
	impOptX                = 756
)

type ImportDlg struct {
	ed     *Editor
	kind   int // 0 character, 1 object / animal
	queue  []droppedImage
	idx    int
	total  int
	name   string
	src    *pixBuf
	srcImg *ebiten.Image
	tw, th int
	view   int // character view that receives the picture

	crop    [4]int // x0,y0,x1,y1 in source pixels (exclusive end)
	hasCrop bool
	drag    bool
	dragA   [2]int

	removeBG bool
	holes    bool // also remove same-colour pixels that are enclosed (between the arms...)
	bgAuto   bool
	bgCol    color.RGBA
	picking  bool
	tol      int
	nCols    int // 4..32; 33 = keep every colour
	fitBox   bool
	flip     bool

	res    *pixBuf
	resImg *ebiten.Image
	dirty  bool
}

// bigForTarget reports whether a picture is clearly not pixel art of the target size.
func bigForTarget(b *pixBuf, tw, th int) bool { return b.w > tw*2 || b.h > th*2 }

// viewGuess picks the character view from a file name.
func viewGuess(name string) int {
	switch {
	case strings.Contains(name, "side") || strings.Contains(name, "perfil") || strings.Contains(name, "lateral"):
		return 1
	case strings.Contains(name, "back") || strings.Contains(name, "espalda") || strings.Contains(name, "trasera"):
		return 2
	}
	return 0
}

// requestImport is the single entry for pictures arriving by button or by drag and drop.
// Small pictures go straight in; big ones open the adjust window.
func (ed *Editor) requestImport(drops []droppedImage) {
	var big []droppedImage
	var small []droppedImage
	tw, th := 0, 0
	kind := -1
	switch {
	case ed.tab == 0 && !(ed.ch.ovMode && ed.ch.ovBuf != nil):
		kind, tw, th = 0, ed.ch.ca.Views[0].w, ed.ch.ca.Views[0].h
	case ed.tab == 1:
		kind, tw, th = 1, ed.ob.frames[0].w, ed.ob.frames[0].h
	}
	for _, d := range drops {
		if d.err != "" {
			ed.say(d.err)
			continue
		}
		if kind >= 0 && bigForTarget(d.b, tw, th) {
			big = append(big, d)
		} else {
			small = append(small, d)
		}
	}
	if len(small) > 0 {
		ed.handleDrops(small)
	}
	if len(big) > 0 {
		ed.imp = newImportDlg(ed, kind, big)
	}
}

func newImportDlg(ed *Editor, kind int, drops []droppedImage) *ImportDlg {
	d := &ImportDlg{ed: ed, kind: kind, queue: drops, total: len(drops), tol: 40, nCols: 16, fitBox: true, bgAuto: true}
	d.next()
	return d
}

// next loads the next queued picture, or closes the window when there is none.
func (d *ImportDlg) next() {
	if d.idx >= len(d.queue) {
		d.ed.imp = nil
		return
	}
	it := d.queue[d.idx]
	d.idx++
	d.name = it.name
	d.src = it.b
	// absurdly large pictures are reduced first so they fit in a texture
	if d.src.w > 4096 || d.src.h > 4096 {
		k := math.Min(4096/float64(d.src.w), 4096/float64(d.src.h))
		d.src = boxScale(d.src, 0, 0, d.src.w, d.src.h, maxI(1, int(float64(d.src.w)*k)), maxI(1, int(float64(d.src.h)*k)))
	}
	d.srcImg = ebiten.NewImage(d.src.w, d.src.h)
	d.srcImg.WritePixels(d.src.p)
	if d.kind == 0 {
		d.tw, d.th = d.ed.ch.ca.Views[0].w, d.ed.ch.ca.Views[0].h
		d.view = viewGuess(d.name)
		if d.total == 1 {
			d.view = d.ed.ch.view
			if g := viewGuess(d.name); g != 0 {
				d.view = g
			}
		}
	} else {
		d.tw, d.th = d.ed.ob.frames[0].w, d.ed.ob.frames[0].h
	}
	d.hasCrop, d.drag, d.picking = false, false, false
	// a picture that already has a transparent background keeps it
	transparent := false
	for i := 3; i < len(d.src.p); i += 4 * 7 {
		if d.src.p[i] < 250 {
			transparent = true
			break
		}
	}
	d.removeBG = !transparent
	d.bgAuto = true
	d.dirty = true
}

func (d *ImportDlg) cropRect() (x0, y0, x1, y1 int) {
	if !d.hasCrop {
		return 0, 0, d.src.w, d.src.h
	}
	return d.crop[0], d.crop[1], d.crop[2], d.crop[3]
}

// ---- image processing -------------------------------------------------------------

// boxScale resamples the rectangle (x0,y0)-(x1,y1) of src to dw x dh, averaging colours
// weighted by their alpha (so edges of cut-outs stay clean).
func boxScale(src *pixBuf, x0, y0, x1, y1, dw, dh int) *pixBuf {
	out := newPix(dw, dh)
	cw, ch := x1-x0, y1-y0
	for dy := 0; dy < dh; dy++ {
		sy0 := y0 + dy*ch/dh
		sy1 := maxI(sy0+1, y0+(dy+1)*ch/dh)
		for dx := 0; dx < dw; dx++ {
			sx0 := x0 + dx*cw/dw
			sx1 := maxI(sx0+1, x0+(dx+1)*cw/dw)
			var r, g, b, a, n float64
			for sy := sy0; sy < sy1 && sy < y1; sy++ {
				for sx := sx0; sx < sx1 && sx < x1; sx++ {
					i := (sy*src.w + sx) * 4
					al := float64(src.p[i+3])
					r += float64(src.p[i]) * al
					g += float64(src.p[i+1]) * al
					b += float64(src.p[i+2]) * al
					a += al
					n++
				}
			}
			if n == 0 || a == 0 {
				continue
			}
			o := (dy*dw + dx) * 4
			out.p[o], out.p[o+1], out.p[o+2], out.p[o+3] = uint8(r/a), uint8(g/a), uint8(b/a), uint8(a/n)
		}
	}
	return out
}

func colDist(p []uint8, i int, c color.RGBA) float64 {
	dr, dg, db := float64(p[i])-float64(c.R), float64(p[i+1])-float64(c.G), float64(p[i+2])-float64(c.B)
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// borderColor is the most common colour along the edge of the picture: the background.
func borderColor(b *pixBuf) color.RGBA {
	type acc struct{ r, g, b, n int }
	buckets := map[int]*acc{}
	add := func(x, y int) {
		i := (y*b.w + x) * 4
		if b.p[i+3] < 128 {
			return
		}
		k := int(b.p[i]>>4)<<8 | int(b.p[i+1]>>4)<<4 | int(b.p[i+2]>>4)
		a := buckets[k]
		if a == nil {
			a = &acc{}
			buckets[k] = a
		}
		a.r += int(b.p[i])
		a.g += int(b.p[i+1])
		a.b += int(b.p[i+2])
		a.n++
	}
	for x := 0; x < b.w; x++ {
		add(x, 0)
		add(x, b.h-1)
	}
	for y := 0; y < b.h; y++ {
		add(0, y)
		add(b.w-1, y)
	}
	var best *acc
	for _, a := range buckets {
		if best == nil || a.n > best.n {
			best = a
		}
	}
	if best == nil {
		return color.RGBA{255, 255, 255, 255}
	}
	return color.RGBA{uint8(best.r / best.n), uint8(best.g / best.n), uint8(best.b / best.n), 255}
}

// knockOut makes the background transparent: pixels close to bg that touch the edge
// (or, with holes, every pixel close to bg).
func knockOut(b *pixBuf, bg color.RGBA, tol float64, holes bool) {
	near := func(x, y int) bool {
		i := (y*b.w + x) * 4
		return b.p[i+3] > 0 && colDist(b.p, i, bg) <= tol
	}
	if holes {
		for y := 0; y < b.h; y++ {
			for x := 0; x < b.w; x++ {
				if near(x, y) {
					b.p[(y*b.w+x)*4+3] = 0
				}
			}
		}
		return
	}
	// flood from the edge: a pixel joins the background when it looks like the pixel it is
	// reached from (so soft gradients and shadows are followed) and is not too far from the
	// main background colour (so the flood does not leak into the figure)
	step, limit := tol*0.45, tol*4
	seen := make([]bool, b.w*b.h)
	var st [][2]int
	for x := 0; x < b.w; x++ {
		for _, y := range []int{0, b.h - 1} {
			if near(x, y) && !seen[y*b.w+x] {
				seen[y*b.w+x] = true
				st = append(st, [2]int{x, y})
			}
		}
	}
	for y := 0; y < b.h; y++ {
		for _, x := range []int{0, b.w - 1} {
			if near(x, y) && !seen[y*b.w+x] {
				seen[y*b.w+x] = true
				st = append(st, [2]int{x, y})
			}
		}
	}
	for len(st) > 0 {
		p := st[len(st)-1]
		st = st[:len(st)-1]
		pi := (p[1]*b.w + p[0]) * 4
		pc := color.RGBA{b.p[pi], b.p[pi+1], b.p[pi+2], 255}
		b.p[pi+3] = 0 // the colour stays readable for its neighbours: only alpha changes
		for _, dd := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			x, y := p[0]+dd[0], p[1]+dd[1]
			if x < 0 || y < 0 || x >= b.w || y >= b.h || seen[y*b.w+x] {
				continue
			}
			i := (y*b.w + x) * 4
			if b.p[i+3] == 0 || colDist(b.p, i, pc) > step || colDist(b.p, i, bg) > limit {
				continue
			}
			seen[y*b.w+x] = true
			st = append(st, [2]int{x, y})
		}
	}
}

// quantize reduces the opaque pixels of b to n colours (k-means, a few passes).
func quantize(b *pixBuf, n int) {
	var idx []int
	for i := 0; i < b.w*b.h; i++ {
		if b.p[i*4+3] > 0 {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 || n >= len(idx) {
		return
	}
	type vec [3]float64
	px := func(i int) vec { return vec{float64(b.p[i*4]), float64(b.p[i*4+1]), float64(b.p[i*4+2])} }
	dist := func(a, c vec) float64 {
		return (a[0]-c[0])*(a[0]-c[0]) + (a[1]-c[1])*(a[1]-c[1]) + (a[2]-c[2])*(a[2]-c[2])
	}
	// start from the colour nearest the mean, then the farthest from the chosen ones
	var mean vec
	for _, i := range idx {
		p := px(i)
		mean[0] += p[0]
		mean[1] += p[1]
		mean[2] += p[2]
	}
	for k := range mean {
		mean[k] /= float64(len(idx))
	}
	best := idx[0]
	for _, i := range idx {
		if dist(px(i), mean) < dist(px(best), mean) {
			best = i
		}
	}
	cent := []vec{px(best)}
	for len(cent) < n {
		far, fd := idx[0], -1.0
		for _, i := range idx {
			m := math.MaxFloat64
			for _, c := range cent {
				m = math.Min(m, dist(px(i), c))
			}
			if m > fd {
				far, fd = i, m
			}
		}
		if fd <= 0 {
			break
		}
		cent = append(cent, px(far))
	}
	assign := make([]int, len(idx))
	for it := 0; it < 6; it++ {
		sum := make([]vec, len(cent))
		cnt := make([]float64, len(cent))
		for j, i := range idx {
			p := px(i)
			bi := 0
			for c := range cent {
				if dist(p, cent[c]) < dist(p, cent[bi]) {
					bi = c
				}
			}
			assign[j] = bi
			sum[bi][0] += p[0]
			sum[bi][1] += p[1]
			sum[bi][2] += p[2]
			cnt[bi]++
		}
		for c := range cent {
			if cnt[c] > 0 {
				cent[c] = vec{sum[c][0] / cnt[c], sum[c][1] / cnt[c], sum[c][2] / cnt[c]}
			}
		}
	}
	for j, i := range idx {
		c := cent[assign[j]]
		b.p[i*4], b.p[i*4+1], b.p[i*4+2], b.p[i*4+3] = uint8(c[0]), uint8(c[1]), uint8(c[2]), 255
	}
}

// process builds the pixel-art result from the crop and the options.
func (d *ImportDlg) process() {
	x0, y0, x1, y1 := d.cropRect()
	cw, ch := x1-x0, y1-y0
	tw, th := d.tw, d.th
	// work at a few times the final size so the cut-out is smooth, but never beyond the crop itself
	k := math.Min(float64(tw*4)/float64(cw), float64(th*4)/float64(ch))
	if k > 1 {
		k = 1
	}
	ww, wh := maxI(1, int(float64(cw)*k)), maxI(1, int(float64(ch)*k))
	wb := boxScale(d.src, x0, y0, x1, y1, ww, wh)
	if d.removeBG {
		if d.bgAuto {
			d.bgCol = borderColor(wb)
		}
		knockOut(wb, d.bgCol, float64(d.tol), d.holes)
	}
	bx0, by0, bx1, by1 := 0, 0, ww, wh
	if d.fitBox {
		minx, miny, maxx, maxy := ww, wh, -1, -1
		for y := 0; y < wh; y++ {
			for x := 0; x < ww; x++ {
				if wb.p[(y*ww+x)*4+3] > 128 {
					minx, miny = minI(minx, x), minI(miny, y)
					maxx, maxy = maxI(maxx, x), maxI(maxy, y)
				}
			}
		}
		if maxx >= 0 {
			bx0, by0, bx1, by1 = minx, miny, maxx+1, maxy+1
		}
	}
	bw, bh := bx1-bx0, by1-by0
	s := math.Min(float64(tw)/float64(bw), float64(th)/float64(bh))
	dw, dh := clampI(int(math.Round(float64(bw)*s)), 1, tw), clampI(int(math.Round(float64(bh)*s)), 1, th)
	sub := boxScale(wb, bx0, by0, bx1, by1, dw, dh)
	out := newPix(tw, th)
	ox, oy := (tw-dw)/2, th-dh
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sx := x
			if d.flip {
				sx = dw - 1 - x
			}
			i := (y*dw + sx) * 4
			if sub.p[i+3] < 128 {
				continue
			}
			o := ((oy+y)*tw + ox + x) * 4
			out.p[o], out.p[o+1], out.p[o+2], out.p[o+3] = sub.p[i], sub.p[i+1], sub.p[i+2], 255
		}
	}
	if d.nCols <= 32 {
		quantize(out, d.nCols)
	}
	d.res = out
	if d.resImg == nil || d.resImg.Bounds().Dx() != tw || d.resImg.Bounds().Dy() != th {
		d.resImg = ebiten.NewImage(tw, th)
	}
	d.resImg.WritePixels(out.p)
	d.dirty = false
}

// ---- window -------------------------------------------------------------------------

func (d *ImportDlg) accept() {
	if d.res == nil {
		return
	}
	name := []string{"front", "side", "back"}[clampI(d.view, 0, 2)]
	drop := []droppedImage{{name: name, b: d.res}}
	if d.kind == 0 {
		d.ed.ch.importDrops(drop)
	} else {
		d.ed.ob.importDrops(drop)
	}
	d.next()
}

func (d *ImportDlg) update(u *UI) {
	if d.dirty {
		d.process()
	}
	u.rect(0, 0, edW, edH, rgb(0x0a0a10))
	title := "IMPORTAR IMAGEN"
	if d.total > 1 {
		title += fmt.Sprintf("  (%d/%d)", d.idx, d.total)
	}
	u.heading(impX, 18, edW-2*impX, title)
	if u.esc {
		d.next()
		return
	}

	// ---- source picture with the crop rectangle
	u.round(impX, impY, impW, impH, rgb(0x101018))
	u.roundFrame(impX, impY, impW, impH, uiLine)
	k := math.Min(float64(impW-8)/float64(d.src.w), float64(impH-8)/float64(d.src.h))
	dw, dh := int(float64(d.src.w)*k), int(float64(d.src.h)*k)
	ix, iy := impX+(impW-dw)/2, impY+(impH-dh)/2
	srcImg := d.srcImg
	u.add(func(dst *ebiten.Image) {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(float64(ix), float64(iy))
		dst.DrawImage(srcImg, op)
	})
	toSrc := func(mx, my int) (int, int) {
		return clampI(int(float64(mx-ix)/k), 0, d.src.w), clampI(int(float64(my-iy)/k), 0, d.src.h)
	}
	overImg := u.in(ix, iy, dw, dh)
	if overImg && u.pressed && u.menu.id == "" {
		sx, sy := toSrc(u.mx, u.my)
		if d.picking {
			d.pickBG(sx, sy)
		} else {
			d.drag, d.dragA = true, [2]int{sx, sy}
		}
	}
	if d.drag {
		sx, sy := toSrc(u.mx, u.my)
		a := d.dragA
		d.crop = [4]int{minI(a[0], sx), minI(a[1], sy), maxI(a[0], sx), maxI(a[1], sy)}
		d.hasCrop = d.crop[2]-d.crop[0] > 3 && d.crop[3]-d.crop[1] > 3
		if !u.down {
			d.drag = false
			d.dirty = true
		}
	}
	x0, y0, x1, y1 := d.cropRect()
	rx, ry := ix+int(float64(x0)*k), iy+int(float64(y0)*k)
	rw, rh := int(float64(x1-x0)*k), int(float64(y1-y0)*k)
	if d.hasCrop {
		shade := color.RGBA{0, 0, 0, 150}
		u.add(func(dst *ebiten.Image) {
			fillRect(dst, float64(ix), float64(iy), float64(dw), float64(ry-iy), shade)
			fillRect(dst, float64(ix), float64(ry+rh), float64(dw), float64(iy+dh-ry-rh), shade)
			fillRect(dst, float64(ix), float64(ry), float64(rx-ix), float64(rh), shade)
			fillRect(dst, float64(rx+rw), float64(ry), float64(ix+dw-rx-rw), float64(rh), shade)
		})
		u.frame(rx, ry, rw, rh, uiAccent)
	}
	if d.picking {
		u.text(impX+8, impY+impH-18, "Haz clic en el color que quieres quitar", uiAccent)
	} else {
		u.text(impX+8, impY+impH-18, "Arrastra sobre la imagen para recortar lo que quieras", uiDim)
	}

	// ---- options
	x, w := impOptX, edW-impOptX-16
	y := 52
	what := "Para un personaje"
	if d.kind == 1 {
		what = "Para un objeto o animal"
	}
	u.text(x, y, fmt.Sprintf("%d x %d", d.src.w, d.src.h), uiDim)
	u.text(x+90, y, what, uiDim)
	y += 24
	u.heading(x, y, w, "RECORTE")
	y += 22
	cx0, cy0, cx1, cy1 := d.cropRect()
	u.text(x, y+4, fmt.Sprintf("%d x %d", cx1-cx0, cy1-cy0), uiText)
	if u.iconBtn(x+90, y, w-90, 26, "image", "Imagen completa", !d.hasCrop, "Quitar el recorte") {
		d.hasCrop = false
		d.dirty = true
	}
	y += 38
	u.heading(x, y, w, "TAMAÑO EN PÍXELES")
	y += 22
	ch := false
	if u.stepper(x, y, "Ancho", &d.tw, 8, 200, 1) {
		ch = true
	}
	if u.stepper(x+w/2, y, "Alto", &d.th, 8, 200, 1) {
		ch = true
	}
	if ch {
		d.dirty = true
	}
	y += 26
	if d.kind == 0 {
		if d.tw != d.ed.ch.ca.Views[0].w || d.th != d.ed.ch.ca.Views[0].h {
			u.text(x, y, "Cambiar el tamaño vacía las demás vistas", uiRed)
		}
		y += 18
		u.heading(x, y, w, "VISTA DE LA IMAGEN")
		y += 22
		if i := u.seg(x, y, w, 28, []segItem{{"person", "Frente"}, {"person", "Perfil"}, {"person", "Espalda"}}, d.view); i >= 0 {
			d.view = i
		}
		y += 38
	} else {
		y += 8
	}
	u.heading(x, y, w, "FONDO")
	y += 22
	if u.toggle(x, y, 150, "Quitar el fondo", &d.removeBG) {
		d.dirty = true
	}
	if d.removeBG {
		u.rect(x+158, y, 20, 20, d.bgCol)
		u.frame(x+158, y, 20, 20, uiLine)
		if u.button(x+184, y, w-184, 20, "Elegir de la imagen", d.picking) {
			d.picking = !d.picking
		}
		y += 28
		if d.picking {
			d.bgAuto = false
		}
		if u.slider("imptol", x, y, w-110, &d.tol, 0, 150, "Tolerancia") {
			d.dirty = true
		}
		y += 22
		if u.toggle(x, y, w, "Quitar también los huecos (entre brazos, piernas...)", &d.holes) {
			d.dirty = true
		}
		y += 28
	} else {
		y += 28
	}
	u.heading(x, y, w, "PIXELADO")
	y += 22
	if u.toggle(x, y, w, "Ajustar al contorno (llenar todo el tamaño)", &d.fitBox) {
		d.dirty = true
	}
	y += 26
	lab := "Colores"
	if d.nCols > 32 {
		lab = "Todos los colores"
	}
	if u.slider("impcols", x, y, w-130, &d.nCols, 4, 33, lab) {
		d.dirty = true
	}
	y += 22
	if u.toggle(x, y, w, "Voltear (mirar al otro lado)", &d.flip) {
		d.dirty = true
	}
	y += 32

	// ---- preview
	u.heading(x, y, w, "RESULTADO")
	y += 22
	if d.res != nil {
		z := maxI(1, minI((edH-y-60)/d.th, (w-110)/d.tw))
		pw, ph := d.tw*z, d.th*z
		bg := canvasBGs[clampI(cvBG, 0, len(canvasBGs)-1)]
		u.rect(x, y, pw, ph, rgb(bg.a))
		if bg.a != bg.b {
			u.add(func(dst *ebiten.Image) {
				for yy := 0; yy < d.th; yy++ {
					for xx := 0; xx < d.tw; xx++ {
						if (xx+yy)%2 == 0 {
							fillRect(dst, float64(x+xx*z), float64(y+yy*z), float64(z), float64(z), rgb(bg.b))
						}
					}
				}
			})
		}
		img := d.resImg
		u.add(func(dst *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(z), float64(z))
			op.GeoM.Translate(float64(x), float64(y))
			dst.DrawImage(img, op)
		})
		u.frame(x-1, y-1, pw+2, ph+2, uiLine)
		u.text(x+pw+14, y, "Tamaño real:", uiDim)
		u.rect(x+pw+14, y+20, d.tw, d.th, rgb(bg.a))
		u.add(func(dst *ebiten.Image) {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x+pw+14), float64(y+20))
			dst.DrawImage(img, op)
		})
		u.text(x+pw+14, y+ph-32, "Luego lo retocas", uiDim)
		u.text(x+pw+14, y+ph-16, "en el editor", uiDim)
	}
	by := edH - 46
	if u.iconBtn(x, by, w/2-4, 36, "check", "Aceptar", false, "Pasar el resultado al editor (Intro)") || (u.enter && u.focus == "") {
		d.accept()
		return
	}
	if u.iconBtn(x+w/2+4, by, w/2-4, 36, "back", "Cancelar", false, "Esc") {
		d.next()
	}
}

// pickBG takes the background colour from the clicked point of the picture.
func (d *ImportDlg) pickBG(sx, sy int) {
	var r, g, b, n int
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x, y := clampI(sx+dx, 0, d.src.w-1), clampI(sy+dy, 0, d.src.h-1)
			i := (y*d.src.w + x) * 4
			r += int(d.src.p[i])
			g += int(d.src.p[i+1])
			b += int(d.src.p[i+2])
			n++
		}
	}
	d.bgCol = color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), 255}
	d.bgAuto = false
	d.removeBG = true
	d.picking = false
	d.dirty = true
}

var _ = sort.Ints
