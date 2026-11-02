package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"os"
)

// rigTest writes a comparison sheet: procedural frames (top) vs rig frames (bottom).
func rigTest(path string, which string) {
	rng := rand.New(rand.NewSource(5))
	var spec CharSpec
	switch which {
	case "elvis":
		spec = presetElvis(rng)
	case "vader":
		spec = presetVader(rng)
	case "sevillana":
		spec = presetSevillana(rng)
	default:
		spec = playerSpec()
	}
	ca := charAssetFromSpec(spec, "t", "people", "")
	rig := buildRig(ca)
	kinds := []int{AnimWalk, AnimRun, AnimDance0, AnimDance1, AnimDrink, AnimTalk, AnimStab, AnimVomit, AnimFall, AnimCarry,
		AnimPoint, AnimLove, AnimLaugh, AnimPirouette, AnimDance4}
	const s = 3
	views := []int{vFront, vSide, vBack}
	img := image.NewRGBA(image.Rect(0, 0, 6*3*charW*s, len(kinds)*2*charH*s))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{60, 60, 70, 255}}, image.Point{}, draw.Src)
	for ri, k := range kinds {
		for vi, v := range views {
			for f := 0; f < animFrames[k]; f++ {
				pr := renderChar(&spec, v, poseFor(k, f)).toRGBA()
				blitRGBA(img, pr, (vi*6+f)*charW*s, ri*2*charH*s, s)
				rg := rig.render(v, poseFor(k, f)).toRGBA()
				blitRGBA(img, rg, (vi*6+f)*charW*s, (ri*2+1)*charH*s, s)
			}
		}
	}
	f, _ := os.Create(path)
	png.Encode(f, img)
	f.Close()
}

func blitRGBA(dst, src *image.RGBA, ox, oy, s int) {
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			c := src.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			for dy := 0; dy < s; dy++ {
				for dx := 0; dx < s; dx++ {
					dst.SetRGBA(ox+x*s+dx, oy+y*s+dy, c)
				}
			}
		}
	}
}

// castSheet writes a contact sheet of the whole cast (debug aid).
func castSheet(path string, zoom int) {
	rng := rand.New(rand.NewSource(3))
	var specs []*CharSpec
	p, c, g := playerSpec(), copSpec(), guardSpec(rng)
	specs = append(specs, &p, &c, &g)
	for _, f := range famousPresets {
		s := f(rng)
		specs = append(specs, &s)
	}
	for i := 0; i < 32; i++ {
		s := genericSpec(rng)
		specs = append(specs, &s)
	}
	for i := 0; i < 36; i++ {
		s := genericSpec(rng)
		s.Female = true
		decorateWoman(&s, rng)
		specs = append(specs, &s)
	}
	for i := 0; i < 18; i++ {
		s := lifeguardSpec(rng, i%2 == 1)
		specs = append(specs, &s)
	}
	cols := 18
	rows := (len(specs) + cols - 1) / cols
	img := image.NewRGBA(image.Rect(0, 0, cols*charW*zoom, rows*charH*zoom))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{60, 60, 70, 255}}, image.Point{}, draw.Src)
	for i, sp := range specs {
		fr := renderChar(sp, 0, poseFor(AnimIdle, 0)).toRGBA()
		blitRGBA(img, fr, (i%cols)*charW*zoom, (i/cols)*charH*zoom, zoom)
	}
	f, _ := os.Create(path)
	png.Encode(f, img)
	f.Close()
}

// animSheet writes the walk, run and dance frames of the player in three views (debug aid).
func animSheet(path string, zoom int) {
	p := playerSpec()
	kinds := []int{AnimWalk, AnimRun, AnimDance0, AnimDrink, AnimStab, AnimDJ, AnimPoint, AnimLove, AnimLaugh, AnimPirouette, AnimDance4}
	img := image.NewRGBA(image.Rect(0, 0, 6*3*charW*zoom, len(kinds)*charH*zoom))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{60, 60, 70, 255}}, image.Point{}, draw.Src)
	for ri, k := range kinds {
		for vi, v := range []int{vFront, vSide, vBack} {
			for f := 0; f < animFrames[k]; f++ {
				fr := renderChar(&p, v, poseFor(k, f)).toRGBA()
				blitRGBA(img, fr, (vi*6+f)*charW*zoom, ri*charH*zoom, zoom)
			}
		}
	}
	f, _ := os.Create(path)
	png.Encode(f, img)
	f.Close()
}

// animalSheet writes all animal frames (debug aid).
func animalSheet(path string, zoom int) {
	type row struct {
		kind   int
		frames []int
	}
	walk := []int{fStand, fWalk0, fWalk1, fWalk2, fWalk3}
	rows := []row{
		{animHorse, append(append([]int{}, walk...), fGraze, fWarn, fKick)},
		{animHen, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
		{animDog, append(append([]int{}, walk...), fSit)},
		{animCat, append(append([]int{}, walk...), fSit)},
		{animChick, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
		{animBull, append(append([]int{}, walk...), fGraze, fWarn, fKick)},
		{animElephant, append(append([]int{}, walk...), fGraze, fWarn)},
		{animGiraffe, append(append([]int{}, walk...), fGraze, fWarn)},
		{animKangaroo, append(append([]int{}, walk...), fWarn, fKick)},
		{animRabbit, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
		{animPig, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
		{animDuck, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
		{animBudgie, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
		{animDeer, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
		{animQuijote, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
		{animSancho, []int{fStand, fWalk0, fWalk1, fPeck0, fPeck1}},
	}
	cw, chh := 76, 76
	img := image.NewRGBA(image.Rect(0, 0, 8*cw*zoom, len(rows)*chh*zoom))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{60, 90, 60, 255}}, image.Point{}, draw.Src)
	gen := func(kind, f, coat int) *pixBuf {
		switch kind {
		case animHorse:
			return genHorseFrame(f, coat).scaled(0.7)
		case animHen:
			return genHenFrame(f, coat)
		case animDog:
			return genDogFrame(f, coat).scaled(0.62)
		case animCat:
			return genCatFrame(f, coat)
		case animChick:
			return genChickFrame(f)
		case animBull:
			return genBullFrame(f, coat)
		case animElephant:
			return genElephantFrame(f, coat)
		case animGiraffe:
			return genGiraffeFrame(f, coat)
		case animRabbit:
			return genRabbitFrame(f, coat)
		case animPig:
			return genPigFrame(f, coat)
		case animDuck:
			return genDuckFrame(f, coat)
		case animBudgie:
			return genBudgieFrame(f, coat)
		case animDeer:
			return genDeerFrame(f, coat)
		case animQuijote:
			return genRiderFrame(f, coat, false).scaled(0.8)
		case animSancho:
			return genRiderFrame(f, coat, true).scaled(0.8)
		}
		return genKangarooFrame(f, coat)
	}
	for ri, r := range rows {
		for i, f := range r.frames {
			b := gen(r.kind, f, ri%3).outlinedSel(0.4).toRGBA()
			blitRGBA(img, b, i*cw*zoom, ri*chh*zoom, zoom)
		}
	}
	f, _ := os.Create(path)
	png.Encode(f, img)
	f.Close()
}

var dbgSolids bool
