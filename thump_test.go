package main

import (
	"math"
	"testing"
)

func thumpLeft(b []byte) []float64 {
	out := make([]float64, len(b)/4)
	for i := range out {
		out[i] = float64(int16(uint16(b[i*4])|uint16(b[i*4+1])<<8)) / 32000
	}
	return out
}

// the loop is made of whole beats, is not silent and has its kick drum at the start of every beat
func TestThumpLoop(t *testing.T) {
	pcm := buildThump()
	if len(pcm) != thumpBeatSamples*thumpBeats*4 {
		t.Fatalf("loop length %d", len(pcm))
	}
	l := thumpLeft(pcm)
	peak := 0.0
	for _, v := range l {
		peak = math.Max(peak, math.Abs(v))
	}
	if peak < 0.5 {
		t.Fatalf("the loop is almost silent (peak %.2f)", peak)
	}
	for b := 0; b < thumpBeats; b++ {
		start := b * thumpBeatSamples
		best, at := 0.0, 0
		for i := 0; i < thumpBeatSamples*4/10; i++ { // before the off-beat bass
			if v := math.Abs(l[start+i]); v > best {
				best, at = v, i
			}
		}
		if at > int(0.012*sampleRate) {
			t.Errorf("beat %d: the kick peaks %d samples after the beat, not at its start", b, at)
		}
		// and it has died down before the next one
		tail := 0.0
		for i := thumpBeatSamples * 8 / 10; i < thumpBeatSamples*95/100; i++ {
			tail = math.Max(tail, math.Abs(l[start+i]))
		}
		if tail > best*0.5 {
			t.Errorf("beat %d: no clear kick (peak %.2f, tail %.2f)", b, best, tail)
		}
	}
}

// the picture and the sound share one clock: the pulse peaks exactly (latency apart) on every kick
func TestThumpClockMatchesSound(t *testing.T) {
	for b := 0; b < 40; b++ {
		audioT := float64(b*thumpBeatSamples) / sampleRate // the sample where the kick starts
		visT := audioT + thumpLatency                      // when the picture hits
		if n := beatN(visT + 1e-6); n != b {
			t.Fatalf("beat %d reads as %d", b, n)
		}
		if k := thumpKick(visT + 1e-6); k < 0.99 {
			t.Fatalf("beat %d: the picture pulse is %.2f on the kick", b, k)
		}
		if k := thumpKick(visT - 0.002); b > 0 && k > 0.2 {
			t.Fatalf("beat %d: the pulse is already %.2f before the kick", b, k)
		}
		if k := thumpKick(visT + thumpBeat*0.5); k > 0.05 {
			t.Fatalf("beat %d: the pulse has not died by the off beat (%.2f)", b, k)
		}
	}
	if math.Abs(thumpBeat*thumpBeats*sampleRate-float64(thumpBeatSamples*thumpBeats)) > 1e-6 {
		t.Error("the loop is not a whole number of samples")
	}
}

// muted: nothing starts and nothing breaks
func TestThumpMute(t *testing.T) {
	s := &Sounds{muted: true, tracks: map[string][]byte{}, sfx: map[string][]byte{}}
	s.StartThump()
	s.SetThumpVolume(1)
	if s.thump != nil {
		t.Fatal("a muted game started the loop")
	}
	s.StopThump()
	if s.thumpWant {
		t.Fatal("stop did not clear it")
	}
}

// how loud the party is in each scene: always 0..1, loud before the cloud pops, cut at the closed door
func TestThumpVolumes(t *testing.T) {
	for tt := 0.0; tt < introLen; tt += 0.05 {
		if v := introThumpVol(tt); v < 0 || v > 1 {
			t.Fatalf("intro volume %.2f at %.2f", v, tt)
		}
	}
	if !(introThumpVol(2) < introThumpVol(10) && introThumpVol(10) < introThumpVol(inBurst-0.1)) {
		t.Error("the bass should swell while he cannot sleep")
	}
	if introThumpVol(inBurst+0.2) > 0.3 {
		t.Error("it should drop when the cloud pops")
	}
	if introThumpVol(inWalk1+1.0) < 0.8 {
		t.Error("the open door should let the party in")
	}
	if introThumpVol(inWalk1+2.6) > 0.15 || introThumpVol(inCard) != 0 {
		t.Error("the closed door should cut it off")
	}
	for tt := 0.0; tt < interLen; tt += 0.05 {
		for _, v := range []float64{interBaseVol(tt, 5), interMemVol(tt, 5)} {
			if v < 0 || v > 1 {
				t.Fatalf("interlude volume %.2f at %.2f", v, tt)
			}
		}
	}
	street, memory := interMemVol(1+interSlot*(interMem+0.2), 5), interMemVol(1+interSlot*0.3, 5)
	if street != 0 || memory < 0.3 {
		t.Errorf("the memory layer must be silent in the street and clear in a memory (%.2f vs %.2f)", street, memory)
	}
	if interBaseVol(itFadeOut, 5) > 0.001 || interMemVol(itFadeOut, 5) > 0.001 {
		t.Error("the interlude music must be gone when it ends")
	}
}
