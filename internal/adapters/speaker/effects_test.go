package speaker

import (
	"math"
	"reflect"
	"testing"

	"lapresregne/internal/soundtrack"
	"lapresregne/internal/world"
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

// amplitudeAt measures how strongly a frequency sounds in a window of a mono signal, in seconds.
func amplitudeAt(samples []float32, frequency, from, length float64) float64 {
	start := int(from * soundtrack.SampleRate)
	return amplitude(samples[start*soundtrack.Channels:], frequency, int(length*1000))
}

func TestTheLevelUpJingleOpensWithAStabAndKnocksAndEndsOnAHeldCChord(t *testing.T) {
	samples := levelUpSound(false)
	if seconds := float64(len(samples)/soundtrack.Channels) / soundtrack.SampleRate; seconds < 5.5 || seconds > 6.5 {
		t.Fatalf("jingle lasts %.2f s, want about the original's 6 s", seconds)
	}
	// The stab sounds G4 and C5 together.
	g4, c5, between := amplitudeAt(samples, 392, 0, 0.1), amplitudeAt(samples, 523.3, 0, 0.1), amplitudeAt(samples, 460, 0, 0.1)
	if g4 < 4*between || c5 < 4*between {
		t.Fatalf("G4 = %.3f, C5 = %.3f, between = %.3f; want a G4 and C5 stab", g4, c5, between)
	}
	// A low wooden knock answers it, well below the stab's pitch.
	if knock, high := amplitudeAt(samples, 210, 0.44, 0.05), amplitudeAt(samples, 1600, 0.44, 0.05); knock < 3*high {
		t.Fatalf("210 Hz = %.3f, 1600 Hz = %.3f at 0.44 s; want a low knock", knock, high)
	}
	// The finale holds a C chord with C3 in the bass.
	if c3, d3 := amplitudeAt(samples, 130.8, 3, 0.2), amplitudeAt(samples, 146.8, 3, 0.2); c3 < 4*d3 {
		t.Fatalf("C3 = %.3f, D3 = %.3f at 3 s; want the finale's C chord", c3, d3)
	}
	peak := 0.0
	for i := 0; i < len(samples); i += soundtrack.Channels {
		if samples[i] != samples[i+1] {
			t.Fatalf("frame %d differs between channels; want a centred jingle", i/2)
		}
		peak = math.Max(peak, math.Abs(float64(samples[i])))
	}
	if peak < 0.1 || peak > 0.6 {
		t.Fatalf("peak = %.2f, want a clear jingle that never clips", peak)
	}
	for _, sample := range samples[len(samples)-soundtrack.SampleRate/10*soundtrack.Channels:] {
		if math.Abs(float64(sample)) > 0.005 {
			t.Fatalf("tail sample = %.4f, want the jingle to fade to silence", sample)
		}
	}
}

func TestTheUnlockingJingleAddsALeadMelody(t *testing.T) {
	plain, unlocking := levelUpSound(false), levelUpSound(true)
	// The lead rises to E5 over the C chord at 0.6 s, and steps down from F5 at 1.5 s.
	for _, check := range []struct{ frequency, at float64 }{{659.3, 0.62}, {698.5, 1.52}} {
		lead, without := amplitudeAt(unlocking, check.frequency, check.at, 0.1), amplitudeAt(plain, check.frequency, check.at, 0.1)
		if lead < 4*without {
			t.Fatalf("%.0f Hz at %.2f s = %.3f with the lead, %.3f without; want the lead melody", check.frequency, check.at, lead, without)
		}
	}
}

func TestAFallingTreeCreaksThenLandsWithALowThud(t *testing.T) {
	samples := fallSound()
	if seconds := float64(len(samples)/soundtrack.Channels) / soundtrack.SampleRate; seconds < 0.8 || seconds > 2 {
		t.Fatalf("fall lasts %.2f s, want a brief creak and thud", seconds)
	}
	// The trunk creaks in the wood's midrange before it lands.
	if creak, rumble := amplitudeAt(samples, 700, 0.15, 0.1), amplitudeAt(samples, 90, 0.15, 0.1); creak < 3*rumble {
		t.Fatalf("700 Hz = %.4f, 90 Hz = %.4f at 0.15 s; want a creak before the landing", creak, rumble)
	}
	// It lands with a thud far below the creak.
	if thud, creak := amplitudeAt(samples, 90, fallLanding, 0.08), amplitudeAt(samples, 700, fallLanding, 0.08); thud < 3*creak {
		t.Fatalf("90 Hz = %.4f, 700 Hz = %.4f at the landing; want a low thud", thud, creak)
	}
	peak := 0.0
	for i := 0; i < len(samples); i += soundtrack.Channels {
		if samples[i] != samples[i+1] {
			t.Fatalf("frame %d differs between channels; want a centred fall", i/2)
		}
		peak = math.Max(peak, math.Abs(float64(samples[i])))
	}
	if peak < 0.05 || peak > 0.5 {
		t.Fatalf("peak = %.2f, want an audible but quiet fall", peak)
	}
	for _, sample := range samples[len(samples)-soundtrack.SampleRate/20*soundtrack.Channels:] {
		if math.Abs(float64(sample)) > 0.005 {
			t.Fatalf("tail sample = %.4f, want the fall to fade to silence", sample)
		}
	}
}

func TestAWoodenThudPlaysTheClunk(t *testing.T) {
	if !reflect.DeepEqual(landingSound(world.WoodenThud), dropSound()) {
		t.Fatal("a wooden thud must play the dull clunk of a log landing")
	}
}

func TestADroppedLogLandsWithAShortDullClunk(t *testing.T) {
	samples := dropSound()
	if seconds := float64(len(samples)/soundtrack.Channels) / soundtrack.SampleRate; seconds < 0.1 || seconds > 0.4 {
		t.Fatalf("drop lasts %.2f s, want a short clunk", seconds)
	}
	// The clunk is duller than the axe's bright tok.
	if clunk, tok := amplitude(samples, 250, 30), amplitude(samples, 1620, 30); clunk < 3*tok {
		t.Fatalf("250 Hz = %.4f, 1620 Hz = %.4f; want a dull clunk rather than a bright tok", clunk, tok)
	}
	peak := 0.0
	for i := 0; i < len(samples); i += soundtrack.Channels {
		if samples[i] != samples[i+1] {
			t.Fatalf("frame %d differs between channels; want a centred clunk", i/2)
		}
		peak = math.Max(peak, math.Abs(float64(samples[i])))
	}
	if peak < 0.05 || peak > 0.4 {
		t.Fatalf("peak = %.2f, want an audible but quiet clunk", peak)
	}
	for _, sample := range samples[len(samples)-soundtrack.SampleRate/50*soundtrack.Channels:] {
		if math.Abs(float64(sample)) > 0.005 {
			t.Fatalf("tail sample = %.4f, want the clunk to fade to silence", sample)
		}
	}
}

// loudness is the root mean square of a mono signal over a window, in seconds.
func loudness(samples []float32, from, length float64) float64 {
	start, frames := int(from*soundtrack.SampleRate), int(length*soundtrack.SampleRate)
	total := 0.0
	for frame := start; frame < start+frames; frame++ {
		sample := float64(samples[frame*soundtrack.Channels])
		total += sample * sample
	}
	return math.Sqrt(total / float64(frames))
}

func TestTheFireworksHissUpThenPopInThrees(t *testing.T) {
	samples := fireworksSound()
	if seconds := float64(len(samples)/soundtrack.Channels) / soundtrack.SampleRate; seconds < 2.2 || seconds > 2.8 {
		t.Fatalf("fireworks last %.2f s, want about the original's 2.5 s", seconds)
	}
	// The launch hisses up before the first pop.
	if early, late := loudness(samples, 0.02, 0.05), loudness(samples, 0.15, 0.05); late < 2*early {
		t.Fatalf("hiss = %.3f early, %.3f late; want it rising toward the first pop", early, late)
	}
	// Three groups of three pops, each louder than the sizzle around it, with lulls between groups.
	for _, pop := range fireworkPops {
		if burst, sizzle := loudness(samples, pop.at+0.01, 0.03), loudness(samples, pop.at+0.1, 0.03); burst < 1.8*sizzle {
			t.Fatalf("pop at %.2f s = %.3f, sizzle after = %.3f; want a distinct pop", pop.at, burst, sizzle)
		}
	}
	if pop, lull := loudness(samples, 0.55, 0.03), loudness(samples, 0.76, 0.05); pop < 3*lull {
		t.Fatalf("pop = %.3f, lull = %.3f; want a quiet lull between groups", pop, lull)
	}
	// Each pop is a broad crackle of noise, not a tone.
	if low, high := amplitudeAt(samples, 500, 0.41, 0.03), amplitudeAt(samples, 4000, 0.41, 0.03); low > 4*high || high > 4*low {
		t.Fatalf("500 Hz = %.4f, 4 kHz = %.4f in a pop; want broadband noise", low, high)
	}
	peak := 0.0
	for i := 0; i < len(samples); i += soundtrack.Channels {
		if samples[i] != samples[i+1] {
			t.Fatalf("frame %d differs between channels; want centred fireworks", i/2)
		}
		peak = math.Max(peak, math.Abs(float64(samples[i])))
	}
	if peak < 0.1 || peak > 0.3 {
		t.Fatalf("peak = %.2f, want fireworks a little under the jingle", peak)
	}
	for _, sample := range samples[len(samples)-soundtrack.SampleRate/20*soundtrack.Channels:] {
		if math.Abs(float64(sample)) > 0.005 {
			t.Fatalf("tail sample = %.4f, want the fireworks to fade to silence", sample)
		}
	}
}
