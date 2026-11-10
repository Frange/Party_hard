package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// vaLand lists every land vehicle drawn as a prop sprite; all of them go through the art kit (vehart.go).
var vaLand = []string{"car", "taxi", "homercar", "wackypink", "wackymean", "wackyboulder", "van", "boxtruck", "icetruck",
	"schoolbus", "batmobile", "delorean", "monstertruck", "octane", "fennec", "dominus", "motorbike", "firetruck", "techvan", "jp_jeep"}

func vaSide(name string, v int) *pixBuf {
	k := propKinds[name]
	b := newPix(k.W+2, k.H+2)
	b.ox, b.oy = 1, 1
	k.Draw(b, v)
	return b
}

func vaVars(name string) int {
	if n := propKinds[name].Vars; n > 0 {
		return n
	}
	return 1
}

// every vehicle drawing (side, front, rear, the emergency frames and the hearse) uses the shared kit
func TestVehicleArtSharedKit(t *testing.T) {
	check := func(what string, draw func()) {
		vaKitUse.build, vaKitUse.wheel = 0, 0
		draw()
		if vaKitUse.build == 0 || vaKitUse.wheel == 0 {
			t.Errorf("%s does not use the vehicle art kit (build %d, wheels %d)", what, vaKitUse.build, vaKitUse.wheel)
		}
	}
	listed := map[string]bool{}
	for _, n := range vaLand {
		listed[n] = true
		k := propKinds[n]
		if k == nil || k.Draw == nil {
			t.Fatalf("%s: not a procedural prop", n)
		}
		for v := 0; v < vaVars(n); v++ {
			check(n+" side", func() { vaSide(n, v) })
			if d := vehViewDefs[n]; d != nil {
				for _, front := range []bool{true, false} {
					check(n+" view", func() { b := newPix(d.w+2, d.h+2); b.ox, b.oy = 1, 1; d.draw(b, v, front) })
				}
			}
		}
	}
	// any road vehicle added later must be listed here (and so drawn with the kit)
	for n, k := range propKinds {
		if k.Draw != nil && (k.Ambient > 0 || k.Category == "vehicle") && !listed[n] && !strings.Contains(n, "boat") && !strings.Contains(n, "ship") {
			t.Errorf("%s: a vehicle that is not in vaLand", n)
		}
	}
	for _, u := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
		for f := 0; f < vaBarFrames; f++ {
			check("police unit", func() { emergencyPix(0, u, f) })
		}
	}
	for f := 0; f < vaBarFrames; f++ {
		check("ambulance", func() { emergencyPix(1, 0, f) })
	}
	check("hearse", func() { b := newPix(68, 38); b.ox, b.oy = 1, 1; vaHearseSide(b) })
}

func vaBBox(b *pixBuf) (x0, y0, x1, y1 int) {
	x0, y0 = b.w, b.h
	for y := 0; y < b.h; y++ {
		for x := 0; x < b.w; x++ {
			if b.p[(y*b.w+x)*4+3] >= 200 {
				x0, y0, x1, y1 = minI(x0, x), minI(y0, y), maxI(x1, x+1), maxI(y1, y+1)
			}
		}
	}
	return
}

// the vehicles fill their box (same size, hitbox and anchors as before) and keep car-like proportions
func TestVehicleArtProportions(t *testing.T) {
	for _, n := range vaLand {
		k := propKinds[n]
		for v := 0; v < vaVars(n); v++ {
			b := vaSide(n, v)
			x0, y0, x1, y1 := vaBBox(b)
			w, h := x1-x0, y1-y0
			if float64(w) < 0.85*float64(k.W) || w > k.W+2 {
				t.Errorf("%s v%d: %d px wide in a %d box", n, v, w, k.W)
			}
			if h < 18 || h > k.H+2 {
				t.Errorf("%s v%d: %d px tall in a %d box", n, v, h, k.H)
			}
			if r := float64(w) / float64(h); r < 1.4 || r > 4.5 {
				t.Errorf("%s v%d: odd proportions %.2f", n, v, r)
			}
			// the wheels touch the ground at the same height as before (y ~ 35 for the 36+ px tall ones)
			if n != "monstertruck" && n != "motorbike" && n != "jp_jeep" && (y1 < minI(35, k.H) || y1 > 38) {
				t.Errorf("%s v%d: bottom at %d", n, v, y1)
			}
		}
	}
	// the emergency pictures are exactly as big as the car sprite they replace
	for _, kind := range []int{0, 1} {
		sb := emergencyPix(kind, 0, 0)
		if sb.w != 68 || sb.h != 38 {
			t.Errorf("emergency %d: %dx%d", kind, sb.w, sb.h)
		}
	}
}

func vaLum(c color.RGBA) float64 { return 0.3*float64(c.R) + 0.59*float64(c.G) + 0.11*float64(c.B) }

// a dark contour all around and a body that stands out from it
func TestVehicleArtContrast(t *testing.T) {
	for _, n := range vaLand {
		for v := 0; v < vaVars(n); v++ {
			b := vaSide(n, v)
			edge, ink := 0, 0
			var sum float64
			cnt := 0
			for y := 0; y < b.h; y++ {
				for x := 0; x < b.w; x++ {
					c := b.get(x-b.ox, y-b.oy)
					if c.A < 200 {
						continue
					}
					cnt++
					sum += vaLum(c)
					out := false
					for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
						if b.get(x+d[0]-b.ox, y+d[1]-b.oy).A < 200 {
							out = true
						}
					}
					if out {
						edge++
						if c == vaInk {
							ink++
						}
					}
				}
			}
			if edge == 0 || float64(ink) < 0.95*float64(edge) {
				t.Errorf("%s v%d: contour %d of %d edge pixels", n, v, ink, edge)
			}
			if avg := sum / float64(cnt); n != "batmobile" && avg-vaLum(vaInk) < 35 { // the Batmobile is black on purpose
				t.Errorf("%s v%d: body too dark against its contour (%.0f)", n, v, avg)
			}
		}
	}
}

// the light bars change from frame to frame and show red and blue
func TestEmergencyLightsAnimate(t *testing.T) {
	cases := [][2]int{{0, unitPolice}, {0, unitFBI}, {0, unitMilitary}, {0, unitSWAT}, {0, unitRangers}, {1, 0}}
	for _, cs := range cases {
		a, b := emergencyPix(cs[0], cs[1], 0), emergencyPix(cs[0], cs[1], 2)
		diff := 0
		red, blue := false, false
		for i := 0; i < len(a.p); i += 4 {
			if a.p[i] != b.p[i] || a.p[i+1] != b.p[i+1] || a.p[i+2] != b.p[i+2] {
				diff++
			}
			for _, p := range [][]byte{a.p[i : i+4], b.p[i : i+4]} {
				if p[3] > 200 && p[0] > 200 && p[1] < 120 && p[2] < 120 {
					red = true
				}
				if p[3] > 200 && p[2] > 200 && p[0] < 140 {
					blue = true
				}
			}
		}
		if diff < 6 || !red || !blue {
			t.Errorf("kind %d unit %d: frames differ in %d px, red %v blue %v", cs[0], cs[1], diff, red, blue)
		}
	}
	if sirenFrame(0) == sirenFrame(0.5/(sirenHz)) {
		t.Error("sirenFrame does not move")
	}
}

// the old per-vehicle drawing helpers are gone and every kit function is used somewhere
func TestNoOrphanVehicleDrawFuncs(t *testing.T) {
	fs := token.NewFileSet()
	files, _ := filepath.Glob("*.go")
	decl := map[string]string{}
	used := map[string]int{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fs, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(af, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Recv == nil {
					decl[x.Name.Name] = f
				}
			case *ast.Ident:
				used[x.Name]++
			}
			return true
		})
	}
	for _, old := range []string{"genCar2", "vehWheels", "vehShadow", "rlWheel", "bigWheel", "vvLamps"} {
		if _, ok := decl[old]; ok {
			t.Errorf("old vehicle helper %s still exists", old)
		}
	}
	var orphans []string
	for name, f := range decl {
		if f == "vehart.go" && used[name] < 2 {
			orphans = append(orphans, name)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Errorf("unused vehicle drawing functions: %v", orphans)
	}
}

// the exported car and motorbike pictures (what the game shows) are the current procedural ones
func TestVehicleExportsMatchProcedural(t *testing.T) {
	for _, n := range []string{"car", "motorbike"} {
		k := propKinds[n]
		if k == nil || k.Draw == nil {
			t.Fatalf("%s: built-in kind missing", n)
		}
		dir := spriteDirOf("assets", "objetos", n)
		if os.Getenv("PH_REGEN_VEHICLE_EXPORTS") == "1" {
			for v := 0; v < vaVars(n); v++ {
				if err := savePNG(filepath.Join(dir, "frame"+itoa(v)+".png"), renderPropSprite(n, k, v)); err != nil {
					t.Fatal(err)
				}
			}
		}
		oa, err := loadObjAsset(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(oa.Frames) != vaVars(n) {
			t.Fatalf("%s: %d exported frames, %d variants", n, len(oa.Frames), vaVars(n))
		}
		for v, f := range oa.Frames {
			want := renderPropSprite(n, k, v)
			if f.w != want.w || f.h != want.h || string(f.p) != string(want.p) {
				t.Errorf("%s frame %d differs from the procedural drawing (run with PH_REGEN_VEHICLE_EXPORTS=1)", n, v)
			}
		}
	}
}

// PH_VEH_SHEET=path writes every vehicle (side, front, rear, emergency frames) on one big sheet, 3x.
func TestVehicleArtSheet(t *testing.T) {
	out := os.Getenv("PH_VEH_SHEET")
	if out == "" {
		return
	}
	var pics []*pixBuf
	for _, n := range vaLand {
		for v := 0; v < vaVars(n); v++ {
			pics = append(pics, vaSide(n, v))
			if d := vehViewDefs[n]; d != nil {
				for _, front := range []bool{true, false} {
					b := newPix(d.w+2, d.h+2)
					b.ox, b.oy = 1, 1
					d.draw(b, v, front)
					pics = append(pics, b)
				}
			}
		}
	}
	for _, u := range []int{unitPolice, unitFBI, unitMilitary, unitSWAT, unitRangers} {
		for f := 0; f < vaBarFrames; f++ {
			pics = append(pics, emergencyPix(0, u, f))
		}
	}
	for f := 0; f < vaBarFrames; f++ {
		pics = append(pics, emergencyPix(1, 0, f))
	}
	hb := newPix(68, 38)
	hb.ox, hb.oy = 1, 1
	vaHearseSide(hb)
	pics = append(pics, hb)
	const W, sc, pageH = 520, 3, 250
	page, x, y, rowH := 0, 4, 4, 0
	sheet := newPix(W*sc, pageH*sc)
	flush := func() {
		if err := savePNG(strings.TrimSuffix(out, ".png")+"_"+itoa(page)+".png", sheet); err != nil {
			t.Fatal(err)
		}
		page++
		sheet = newPix(W*sc, pageH*sc)
	}
	for _, p := range pics {
		if x+p.w+4 > W {
			x, y, rowH = 4, y+rowH+6, 0
		}
		if y+p.h+4 > pageH {
			flush()
			x, y, rowH = 4, 4, 0
		}
		for i := 0; i < len(sheet.p) && x == 4 && y == 4; i += 4 {
			sheet.p[i], sheet.p[i+1], sheet.p[i+2], sheet.p[i+3] = 0x9a, 0xa4, 0xb4, 255
		}
		for yy := 0; yy < p.h; yy++ {
			for xx := 0; xx < p.w; xx++ {
				j := (yy*p.w + xx) * 4
				if p.p[j+3] < 128 {
					continue
				}
				sheet.rect((x+xx)*sc, (y+yy)*sc, sc, sc, color.RGBA{p.p[j], p.p[j+1], p.p[j+2], 255})
			}
		}
		x += p.w + 6
		rowH = maxI(rowH, p.h)
	}
	flush()
}
