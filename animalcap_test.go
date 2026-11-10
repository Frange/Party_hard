package main

import "testing"

func animalCounts(g *Game) (total, bulls int) {
	for _, a := range g.animals {
		total++
		if a.kind == animBull {
			bulls++
		}
	}
	return
}

// every level (campaign, classics and 10 seeds per generator) ends with animalMax animals or fewer and one bull at most
func TestAnimalCapSweep(t *testing.T) {
	defs := testLevels(t)
	for _, kind := range genKinds {
		for sd := int64(1); sd <= 10; sd++ {
			if d := genNamed(kind, sd*977, ""); d != nil {
				defs = append(defs, d)
			}
		}
	}
	if animalMin > animalMax || bullMax != 1 || animalMax != 8 {
		t.Fatalf("constants changed: %d %d %d", animalMin, animalMax, bullMax)
	}
	for i, def := range defs {
		for rep := int64(0); rep < 3; rep++ {
			g := capGame(def, int64(i)*7+rep)
			g.spawnAnimals()
			n, b := animalCounts(g)
			if n > animalMax || b > bullMax {
				t.Errorf("%s: %d animals, %d bulls", def.Title, n, b)
			}
		}
	}
}

// the net catches everything: a level crammed with animals and three bulls
func TestAnimalCapTrims(t *testing.T) {
	g := testGame()
	for i := 0; i < 20; i++ {
		k := animHen
		if i%4 == 0 {
			k = animBull
		}
		g.animals = append(g.animals, &Animal{kind: k})
	}
	g.capAnimals()
	n, b := animalCounts(g)
	if n != animalMax || b != bullMax {
		t.Errorf("got %d animals, %d bulls", n, b)
	}
	g.animals = g.animals[:2]
	g.capAnimals()
	if len(g.animals) != 2 {
		t.Errorf("a quiet level keeps its animals")
	}
}
