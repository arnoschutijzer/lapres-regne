package speaker

import (
	"math"
	"testing"

	"dither-cli/internal/soundtrack"
)

// amplitude measures how strongly a frequency sounds in the first milliseconds of a mono signal.
func amplitude(samples []float32, frequency float64, milliseconds int) float64 {
	var re, im float64
	n := soundtrack.SampleRate * milliseconds / 1000
	for i := range n {
		phase := 2 * math.Pi * frequency * float64(i) / soundtrack.SampleRate
		re += float64(samples[i*soundtrack.Channels]) * math.Cos(phase)
		im += float64(samples[i*soundtrack.Channels]) * math.Sin(phase)
	}
	return math.Hypot(re, im) * 2 / float64(n)
}

func TestTheChopIsABriefBrightWoodenTok(t *testing.T) {
	samples := chopSound()
	frames := len(samples) / soundtrack.Channels
	if frames == 0 || frames > soundtrack.SampleRate/10 {
		t.Fatalf("frames = %d, want a sound shorter than a tenth of a second", frames)
	}
	if tok, thump := amplitude(samples, 1620, 12), amplitude(samples, 120, 12); tok < 4*thump {
		t.Fatalf("1620 Hz = %.3f, 120 Hz = %.3f; want a bright knock rather than a low thump", tok, thump)
	}
	peak := 0.0
	for i := 0; i < len(samples); i += soundtrack.Channels {
		if samples[i] != samples[i+1] {
			t.Fatalf("frame %d differs between channels; want a centred sound", i/2)
		}
		peak = math.Max(peak, math.Abs(float64(samples[i])))
	}
	if peak < 0.05 || peak > 0.5 {
		t.Fatalf("peak = %.2f, want an audible but quiet tok", peak)
	}
	for _, sample := range samples[len(samples)-soundtrack.SampleRate/100*soundtrack.Channels:] {
		if math.Abs(float64(sample)) > 0.01 {
			t.Fatalf("tail sample = %.3f, want the tok to fade to silence", sample)
		}
	}
}
