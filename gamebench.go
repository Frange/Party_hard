package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// benchGame wraps the game to time it (debug: -gamebench N): it runs N frames, prints the
// average and the worst cost of Update and Draw per frame and quits. The draw time is the CPU
// time that Ebiten needs to queue the frame, not the time of the graphics card.
type benchGame struct {
	*Game
	n, f       int
	upd, drw   time.Duration
	updW, drwW time.Duration
}

func (b *benchGame) Update() error {
	t := time.Now()
	err := b.Game.Update()
	d := time.Since(t)
	b.upd += d
	if d > b.updW && b.f > 60 { // the first frames build the level and the sounds: not counted as worst
		b.updW = d
	}
	return err
}

func (b *benchGame) Draw(screen *ebiten.Image) {
	t := time.Now()
	b.Game.Draw(screen)
	d := time.Since(t)
	b.drw += d
	if d > b.drwW && b.f > 60 {
		b.drwW = d
	}
	b.f++
	if b.f >= b.n {
		ms := func(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
		fmt.Printf("game bench: %d frames, update %.2f ms (worst %.2f), draw %.2f ms (worst %.2f) per frame, guests %d, players %d\n",
			b.f, ms(b.upd)/float64(b.f), ms(b.updW), ms(b.drw)/float64(b.f), ms(b.drwW), len(b.Game.guests), len(b.Game.players()))
		os.Exit(0)
	}
}
