package main

import (
	"fmt"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

func newTestRand(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// PAPER_DUMP=dir writes the page (1 and 2 players, both languages) as PNG for a human look; nothing otherwise.
func TestPaperDumpForHumans(t *testing.T) {
	dir := os.Getenv("PAPER_DUMP")
	if dir == "" {
		t.Skip("PAPER_DUMP not set")
	}
	for _, lang := range []string{"es", "en"} {
		inLang(t, lang, func() {
			for _, two := range []bool{false, true} {
				f := paperFacts{level: "7", levelN: 7, kills: 14, knife: 9, poison: 3, traps: 2, seen: 2, seed: 4242, animalsDead: 2, dj: true, cuffed: 1, units: 1 << unitPolice}
				im, _ := renderPaper(paperSample(f, two, false, 9), 0)
				fh, err := os.Create(fmt.Sprintf("%s/paper_%s_%v.png", dir, lang, two))
				if err == nil {
					png.Encode(fh, im)
					fh.Close()
				}
			}
		})
	}
}
