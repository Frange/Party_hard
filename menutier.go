package main

import "image/color"

// ---------------------------------------------------------------------------
// MENU TIER: the main menu changes colour with the progress.
//   0 normal  1 bronze (the story is finished: every campaign party won)
//   2 silver (and every trophy, the legend included)  3 gold (EVERYTHING: COMPLETADO 100 %, completion.go)
// (the legend trophy comes by itself with the others, so "every trophy" is the silver step and gold needs the rest:
// characters, VIPs, killers, cops and the whole shop.)
// Pure functions + one method that reads Progress. Used by the menu colours, the slogan, the title under the
// logo, the completion block (completion.go) and the menu music.
// ---------------------------------------------------------------------------

const (
	tierNormal = iota
	tierBronze
	tierSilver
	tierGold
)

// menuTierOf: finished = every campaign party won; trophies = every trophy; everything = completion 100 %.
func menuTierOf(finished, trophies, everything bool) int {
	switch {
	case !finished:
		return tierNormal
	case everything && trophies:
		return tierGold
	case trophies:
		return tierSilver
	}
	return tierBronze
}

// gameFinished: every campaign party was won at least once (the story is over).
func (g *Game) gameFinished() bool {
	return campaignN > 0 && g.careerLevelsDone() >= campaignN
}

// debugMenuTier: DEBUG row "Estado del menú" (0 = what the save says, 1..4 = normal/bronze/silver/gold). Never saved.
func (g *Game) menuTier() int {
	if g.dbg.tier > 0 {
		return g.dbg.tier - 1
	}
	return menuTierOf(g.gameFinished(), g.allTrophies(), g.complFull())
}

var complFullCache struct {
	n int
	v bool
}

// complFull: completion is 100 % (re-read every 30 calls: it walks the album and the shop, and the menu asks every frame).
func (g *Game) complFull() bool {
	if complFullCache.n%30 == 0 {
		complFullCache.v = complPercent(g.complParts()) >= 100
	}
	complFullCache.n++
	return complFullCache.v
}

// menuTierColors: the normal and the highlighted colour of the menu rows.
func menuTierColors(tier int) (col, hot color.RGBA) {
	switch tier {
	case tierBronze:
		return rgb(0xc88048), rgb(0xf4b880)
	case tierSilver:
		return rgb(0xb4bcc8), rgb(0xf4f8ff)
	case tierGold:
		return rgb(0xe0b030), rgb(0xfff0a0)
	}
	return rgb(0x3fb59a), rgb(0xff6a8a)
}

// menuTierTitle: the line under the logo (nothing before the story is finished); only gold is the legend.
func menuTierTitle(tier int) (s string, c color.RGBA) {
	switch tier {
	case tierBronze:
		return "★ PESADILLA DEL VECINDARIO ★", rgb(0xd89058)
	case tierSilver:
		return "★ TERROR DE LA FIESTA ★", rgb(0xd8e0ec)
	case tierGold:
		return "★ LEYENDA DE LA FIESTA ★", rgb(0xffe070)
	}
	return "", color.RGBA{}
}
