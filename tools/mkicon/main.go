// mkicon regenerates the icon of Party Hardy (window, .exe, .app) from code.
//
//	go run ./tools/mkicon            (from the project root)
//
// Writes icon/icon_{16,32,48,64,128,256,512}.png, icon/icon.ico,
// icon/AppIcon.icns and, when `rsrc` is installed
// (go install github.com/akavel/rsrc@latest), rsrc_windows_amd64.syso, which
// Go links into the Windows executables. No other dependencies. build.sh only
// copies the files; run this tool again if the logo changes.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
)

// 64x64 "art pixel" grid, drawn at 512 px (8 px per art pixel).
const grid, unit = 64, 8

var (
	bg     = color.RGBA{10, 9, 18, 255}
	ring   = color.RGBA{34, 22, 66, 255}
	shadow = color.RGBA{50, 0, 82, 255}
	cyan   = color.RGBA{99, 215, 248, 255}
)

// 10 wide x 14 high letters, strokes 3 px thick.
var hardy = map[byte][]string{
	'H': {"###....###", "###....###", "###....###", "###....###", "###....###", "###....###", "##########", "##########", "###....###", "###....###", "###....###", "###....###", "###....###", "###....###"},
	'A': {"...####...", "..######..", ".###..###.", ".###..###.", "###....###", "###....###", "###....###", "##########", "##########", "###....###", "###....###", "###....###", "###....###", "###....###"},
	'R': {"########..", "#########.", "###...###.", "###....###", "###....###", "###...###.", "#########.", "########..", "###..###..", "###...###.", "###....###", "###....###", "###....###", "###....###"},
	'D': {"########..", "#########.", "###...###.", "###....###", "###....###", "###....###", "###....###", "###....###", "###....###", "###....###", "###....###", "###...###.", "#########.", "########.."},
	'Y': {"###....###", "###....###", ".###..###.", ".###..###.", "..######..", "..######..", "...####...", "...####...", "...####...", "...####...", "...####...", "...####...", "...####...", "...####..."},
}

// 5x7 letters of PARTY.
var party = map[byte][]string{
	'P': {"####.", "#...#", "#...#", "####.", "#....", "#....", "#...."},
	'A': {".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'R': {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
	'T': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."},
	'Y': {"#...#", "#...#", ".#.#.", "..#..", "..#..", "..#..", "..#.."},
}

func lerp(a, b color.RGBA, t float64) color.RGBA {
	f := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}

func master() *image.RGBA {
	n := grid * unit
	im := image.NewRGBA(image.Rect(0, 0, n, n))
	for i := 0; i < len(im.Pix); i += 4 {
		im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = bg.R, bg.G, bg.B, 255
	}
	// faint concentric rings (hard edges, no antialiasing)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x-260), float64(y-250))
			for _, r := range []float64{100, 150, 200} {
				if math.Abs(d-r) < 2 {
					im.SetRGBA(x, y, ring)
				}
			}
		}
	}
	px := func(ax, ay int, c color.RGBA, u int) { // one art pixel of u px at (ax,ay) in units of u
		for y := 0; y < u; y++ {
			for x := 0; x < u; x++ {
				im.SetRGBA(ax*u+x, ay*u+y, c)
			}
		}
	}
	// PARTY: 6 px per art pixel, 5 letters of 5 + 1 gap = 29 wide, centred; y at 17..24 (8px units 22)
	const pu = 6
	px0, py0 := (n/pu-29)/2, 27
	for pass := 0; pass < 2; pass++ {
		for i, ch := range []byte("PARTY") {
			for r, row := range party[ch] {
				for c, v := range row {
					if v == '#' {
						ax, ay := px0+i*6+c, py0+r
						if pass == 0 {
							px(ax+1, ay+1, shadow, pu)
						} else {
							px(ax, ay, cyan, pu)
						}
					}
				}
			}
		}
	}
	// HARDY: 5 letters x 10 + 4 gaps of 1 = 54 wide (art pixels of 8 px)
	left, top := (grid-54-1)/2, 28
	pink, violet := color.RGBA{255, 40, 110, 255}, color.RGBA{145, 62, 225, 255}
	for i, ch := range []byte("HARDY") {
		for r, row := range hardy[ch] {
			for c, v := range row {
				if v == '#' {
					ax, ay := left+i*11+c, top+r
					px(ax+1, ay+1, shadow, unit) // drop shadow
					px(ax-1, ay, shadow, unit)   // outline pieces
					px(ax+1, ay, shadow, unit)
					px(ax, ay-1, shadow, unit)
					px(ax, ay+1, shadow, unit)
				}
			}
		}
	}
	for i, ch := range []byte("HARDY") {
		col := lerp(pink, violet, float64(i)/4)
		for r, row := range hardy[ch] {
			for c, v := range row {
				if v == '#' {
					px(left+i*11+c, top+r, col, unit)
				}
			}
		}
	}
	return im
}

// box filter down to s x s
func scale(src *image.RGBA, s int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, s, s))
	sw := src.Bounds().Dx()
	if s > sw { // nearest up
		for y := 0; y < s; y++ {
			for x := 0; x < s; x++ {
				dst.SetRGBA(x, y, src.RGBAAt(x*sw/s, y*sw/s))
			}
		}
		return dst
	}
	k := float64(sw) / float64(s)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			var r, g, b, n float64
			for yy := int(float64(y) * k); yy < int(float64(y+1)*k); yy++ {
				for xx := int(float64(x) * k); xx < int(float64(x+1)*k); xx++ {
					c := src.RGBAAt(xx, yy)
					r, g, b, n = r+float64(c.R), g+float64(c.G), b+float64(c.B), n+1
				}
			}
			dst.SetRGBA(x, y, color.RGBA{uint8(r/n + .5), uint8(g/n + .5), uint8(b/n + .5), 255})
		}
	}
	return dst
}

func pngBytes(im image.Image) []byte {
	var b bytes.Buffer
	png.Encode(&b, im)
	return b.Bytes()
}

func main() {
	m := master()
	must := func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	must(os.MkdirAll("icon", 0o755))
	data := map[int][]byte{}
	for _, s := range []int{16, 32, 48, 64, 128, 256, 512, 1024} {
		data[s] = pngBytes(scale(m, s))
		if s <= 512 {
			must(os.WriteFile(filepath.Join("icon", fmt.Sprintf("icon_%d.png", s)), data[s], 0o644))
		}
	}
	// .ico with PNG entries
	var ico bytes.Buffer
	sizes := []int{16, 32, 48, 64, 128, 256}
	binary.Write(&ico, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})
	off := 6 + 16*len(sizes)
	for _, s := range sizes {
		w := byte(s)
		if s == 256 {
			w = 0
		}
		ico.Write([]byte{w, w, 0, 0})
		binary.Write(&ico, binary.LittleEndian, [2]uint16{1, 32})
		binary.Write(&ico, binary.LittleEndian, [2]uint32{uint32(len(data[s])), uint32(off)})
		off += len(data[s])
	}
	for _, s := range sizes {
		ico.Write(data[s])
	}
	must(os.WriteFile("icon/icon.ico", ico.Bytes(), 0o644))
	// .icns with PNG entries
	type ent struct {
		typ string
		s   int
	}
	var body bytes.Buffer
	for _, e := range []ent{{"icp4", 16}, {"icp5", 32}, {"icp6", 64}, {"ic07", 128}, {"ic08", 256}, {"ic09", 512}, {"ic10", 1024}} {
		body.WriteString(e.typ)
		binary.Write(&body, binary.BigEndian, uint32(8+len(data[e.s])))
		body.Write(data[e.s])
	}
	var icns bytes.Buffer
	icns.WriteString("icns")
	binary.Write(&icns, binary.BigEndian, uint32(8+body.Len()))
	icns.Write(body.Bytes())
	must(os.WriteFile("icon/AppIcon.icns", icns.Bytes(), 0o644))
	// Windows resource for the .exe (needs rsrc; optional)
	p, err := exec.LookPath("rsrc")
	if err != nil {
		if h, e := os.UserHomeDir(); e == nil {
			p, err = exec.LookPath(filepath.Join(h, "go", "bin", "rsrc"))
		}
	}
	if err == nil {
		out, err := exec.Command(p, "-ico", "icon/icon.ico", "-arch", "amd64", "-o", "rsrc_windows_amd64.syso").CombinedOutput()
		if err != nil {
			fmt.Println("rsrc:", err, string(out))
		}
	} else {
		fmt.Println("rsrc not found: rsrc_windows_amd64.syso not regenerated")
	}
	fmt.Println("icons written")
}
