package main

// ---------------------------------------------------------------------------
// The first time: with no game saved (a fresh install, or after "BORRA TODO")
// the program opens with the intro and then leaves you in the main menu WITHOUT
// the CONTINUAR row. NUEVO JUEGO goes straight to the first party (the intro
// is not repeated) and from then on CONTINUAR is there.
// ---------------------------------------------------------------------------

// hasSave: is there a game to continue? Having started the first party counts
// ("played"), and so does anything that was earned or bought.
func (g *Game) hasSave() bool {
	p := g.progress
	return p.Unlocked > 1 || p.Stats["played"] > 0 || p.Wallet > 0 || len(p.Owned) > 0 || len(p.Equip) > 0
}

// firstRun: nothing saved and the intro not seen yet.
func (g *Game) firstRun() bool {
	return !g.hasSave() && !g.seen("seen_intro")
}

// bootIntro plays the intro as the opening of the program; it ends in the main menu.
func (g *Game) bootIntro() {
	g.startIntro()
	g.introBoot = true
}

// leaveBootIntro: back to the main menu, on NUEVO JUEGO.
func (g *Game) leaveBootIntro() {
	g.introBoot = false
	g.mode = ModeMenu
	g.mainSel = 0
	for i, id := range g.menuList() {
		if id == menuNewGame {
			g.mainSel = i
		}
	}
	g.snd.StartMusic()
}

// wipeProgress erases everything (the "BORRA TODO" popup), the cinematics seen too, and starts over
// with the intro, as on the very first run.
func (g *Game) wipeProgress() {
	g.progress = Progress{Unlocked: 1, GuestMin: g.progress.GuestMin, GuestMax: g.progress.GuestMax}
	saveProgress(g.progress)
	g.syncShop()
	g.shopCache, g.tierBase = nil, nil
	g.applyShop()
	g.bootIntro()
}
