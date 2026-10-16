package speaker

import (
	"math"
	"testing"

	"dither-cli/internal/soundtrack"
)

func TestTheChopIsAShortQuietThunkThatFadesOut(t *testing.T) {
	samples := chopSound()
	frames := len(samples) / soundtrack.Channels
	if frames == 0 || frames > soundtrack.SampleRate/4 {
		t.Fatalf("frames = %d, want a sound shorter than a quarter second", frames)
	}
	peak := 0.0
	for i := 0; i < len(samples); i += soundtrack.Channels {
		if samples[i] != samples[i+1] {
			t.Fatalf("frame %d differs between channels; want a centred sound", i/2)
		}
		peak = math.Max(peak, math.Abs(float64(samples[i])))
	}
	if peak < 0.05 || peak > 0.5 {
		t.Fatalf("peak = %.2f, want an audible but quiet thunk", peak)
	}
	for _, sample := range samples[len(samples)-soundtrack.SampleRate/100*soundtrack.Channels:] {
		if math.Abs(float64(sample)) > 0.01 {
			t.Fatalf("tail sample = %.3f, want the thunk to fade to silence", sample)
		}
	}
}
