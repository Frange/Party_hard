package main

import (
	"math/rand"
	"testing"
)

// every party has at least two red phones and the yellow one, apart from each other and reachable
func TestPhoneBooths(t *testing.T) {
	for _, def := range builtinLevels() {
		if len(def.Props) == 0 && def.Sky == "space" { // built when the party starts
			def = genStation(rand.New(rand.NewSource(3)))
		}
		l := buildLevel(def)
		var reds []Vec
		var yellow []Vec
		for _, p := range l.Props {
			if p.K.Phone {
				reds = append(reds, Vec{p.X, p.Y})
			}
			if p.K.TechPhone {
				yellow = append(yellow, Vec{p.X, p.Y})
			}
		}
		if len(reds) < 2 || len(yellow) != 1 {
			t.Errorf("%s: %d red, %d yellow phones", def.Title, len(reds), len(yellow))
			continue
		}
		if dist(reds[0], reds[1]) < 60 {
			t.Errorf("%s: red phones too close (%.0f)", def.Title, dist(reds[0], reds[1]))
		}
		for _, r := range reds {
			if dist(r, yellow[0]) < 40 {
				t.Errorf("%s: yellow phone on top of a red one", def.Title)
			}
		}
		if l.Path(def.Start, Vec{yellow[0].X, yellow[0].Y + 14}) == nil {
			t.Errorf("%s: yellow phone unreachable", def.Title)
		}
	}
}

func TestCampaignHasTwenty(t *testing.T) {
	builtinLevels()
	if campaignN != 21 || len(campaignLevels) != 21 {
		t.Fatalf("campaign has %d levels built of %d", campaignN, len(campaignLevels))
	}
	if campaignLevels[20].kind != "station" {
		t.Fatal("the space station must be the last party")
	}
}
