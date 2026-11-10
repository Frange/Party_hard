package main

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

var ovlKinds = []string{"house", "club", "pub", "cinema", "mall", "market", "casino", "station", "graveyard", "ski", "plaza", "paris",
	"bullring", "stadium", "festival", "funfair", "roof", "yacht", "skatepark", "circuit", "highway", "trainstation", "beach", "birthday", "hockey", "disco70"}

// every scenario the game can put on screen: the campaign, the random pool, the island and a spread of seeds of every kind
func allGeneratedLevels() map[string]*LevelDef {
	out := map[string]*LevelDef{}
	for i, d := range builtinLevels() {
		out[fmt.Sprintf("built%02d-%s", i, d.Title)] = d
	}
	out["jurassic"] = jurassicLevel()
	for _, k := range ovlKinds {
		for s := int64(1); s <= 3; s++ {
			if d := genNamed(k, s*131, ""); d != nil {
				out[fmt.Sprintf("%s-s%d", k, s)] = d
			}
		}
	}
	return out
}

func TestOverlapReport(t *testing.T) {
	skipResolve = os.Getenv("PH_RAW") != ""
	defer func() { skipResolve = false }()
	if os.Getenv("PH_OVL") == "" {
		t.Skip()
	}
	levels := allGeneratedLevels()
	var names []string
	for n := range levels {
		names = append(names, n)
	}
	sort.Strings(names)
	byKind := map[string]int{}
	total, bad := 0, 0
	for _, n := range names {
		l := buildLevel(levels[n])
		iss := levelOverlaps(l)
		if len(iss) > 0 {
			bad++
		}
		total += len(iss)
		for _, is := range iss {
			a, b := is.A, is.B
			if a > b {
				a, b = b, a
			}
			byKind[is.What+" "+a+"/"+b]++
		}
		if os.Getenv("PH_OVL") == "full" {
			for _, is := range iss {
				fmt.Printf("%s  %s\n", n, is)
			}
		} else if len(iss) > 0 {
			fmt.Printf("%-28s %d\n", n, len(iss))
		}
	}
	fmt.Println("levels:", len(names), "with overlaps:", bad, "total:", total)
	var ks []string
	for k := range byKind {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return byKind[ks[i]] > byKind[ks[j]] })
	for _, k := range ks {
		fmt.Printf("%4d  %s\n", byKind[k], k)
	}
}

// ---- regression tests ----------------------------------------------------------------

// no object of any generated scenario stands on another object, on a wall, in the water, on a road, on a
// dance floor, on the points the game needs clear, or outside the ground of the party
func TestNoObjectsOnTopOfEachOther(t *testing.T) {
	levels := allGeneratedLevels()
	if len(levels) < 90 {
		t.Fatalf("only %d scenarios generated", len(levels))
	}
	var names []string
	for n := range levels {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		for _, is := range levelOverlaps(buildLevel(levels[n])) {
			t.Errorf("%s: %s", n, is)
		}
	}
}

// the island: nothing stands in the gate or in front of it, the parked jeeps and everything else are on land
func TestJurassicGateAndGround(t *testing.T) {
	d := jurassicLevel()
	l := buildLevel(d)
	fitAllFootprints()
	var gate *Prop
	for _, p := range l.Props {
		if p.Name == "jp_gate" {
			gate = p
		}
	}
	if gate == nil {
		t.Fatal("no gate")
	}
	zone := gateZone(gate.Foot)
	for _, p := range l.Props {
		if p == gate || waterOK[p.Name] {
			continue
		}
		if ox, oy := rectOv(p.Foot, zone); ox > 1 && oy > 1 {
			t.Errorf("%s stands in the gate (%.0f,%.0f)", p.Name, p.X, p.Y)
		}
		if ox, oy := rectOv(p.base(), zone); ox > 3 && oy > 3 {
			t.Errorf("%s is drawn on the gate (%.0f,%.0f)", p.Name, p.X, p.Y)
		}
		if p.Name == "jp_jeep" || p.Name == "speaker" || p.Name == "dj" {
			f := p.Foot
			for _, c := range []Vec{{f.X, f.Y}, {f.X + f.W, f.Y}, {f.X, f.Y + f.H}, {f.X + f.W, f.Y + f.H}} {
				if !d.Shape(c.X, c.Y, 0) {
					t.Errorf("%s at (%.0f,%.0f) is over the sea", p.Name, p.X, p.Y)
					break
				}
			}
		}
	}
}
