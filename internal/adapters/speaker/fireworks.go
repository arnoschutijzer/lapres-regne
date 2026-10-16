package speaker

import (
	"math"

	"lapresregne/internal/soundtrack"
)

// firework is one pop of the level-up fireworks: when it bursts, in seconds, and how loud it is
// next to the loudest.
type firework struct{ at, level float64 }

// fireworkPops follow Old School RuneScape's level-up fireworks: three groups of three pops, 0.15 s
// apart within a group and 0.6 s apart between groups, each group a little quieter.
var fireworkPops = []firework{
	{0.25, 1}, {0.40, 1}, {0.55, 0.95},
	{0.855, 0.95}, {1.0, 0.95}, {1.15, 0.9},
	{1.45, 0.75}, {1.60, 0.75}, {1.75, 0.72},
}

// fireworksSound synthesizes the level-up fireworks: a launch hissing up, then the pops, each a
// burst of broad noise over a sizzle that lingers between pops, darkening as it fades after the last.
func fireworksSound() []float32 {
	const duration, peak = 2.5, 0.22
	frames := int(duration * soundtrack.SampleRate)
	// The original stops near 11 kHz, so the noise is gently rounded off above 12 kHz; an aging
	// sizzle loses its highs below 2.5 kHz.
	brightCutoff := 1 - math.Exp(-2*math.Pi*12000/soundtrack.SampleRate)
	darkCutoff := 1 - math.Exp(-2*math.Pi*2500/soundtrack.SampleRate)
	// darkBoost makes up for the level the darker noise loses.
	const darkBoost = 2.2
	// A fixed noise seed makes every display sound the same.
	noise := uint32(0x6c8e9cf5)
	var bright, dark float64
	fireworks := make([]float64, frames)
	loudest := 0.0
	for frame := range frames {
		t := float64(frame) / soundtrack.SampleRate
		noise ^= noise << 13
		noise ^= noise >> 17
		noise ^= noise << 5
		white := float64(noise)/math.MaxUint32*2 - 1
		bright += (white - bright) * brightCutoff
		dark += (white - dark) * darkCutoff

		// The launch hisses up over a fifth of a second and breaks off just before the first pop.
		hiss := 0.0
		if t < 0.235 {
			hiss = 0.6 * math.Pow(t/0.21, 1.5) * math.Min(1, (0.235-t)/0.01)
		}
		pops, freshSizzle, agedSizzle := 0.0, 0.0, 0.0
		for _, pop := range fireworkPops {
			since := t - pop.at
			pops += pop.level * popEnvelope(since)
			sizzle := pop.level * sizzleEnvelope(since)
			fresh := math.Exp(-max(0, since) / 0.25)
			freshSizzle += sizzle * fresh
			agedSizzle += sizzle * (1 - fresh)
		}
		fireworks[frame] = (hiss+pops+freshSizzle)*bright + darkBoost*agedSizzle*dark
		loudest = math.Max(loudest, math.Abs(fireworks[frame]))
	}
	samples := make([]float32, frames*soundtrack.Channels)
	for frame, sample := range fireworks {
		// The last fiftieth of a second fades out, so the sound never ends on a click.
		fade := math.Min(1, (duration-float64(frame)/soundtrack.SampleRate)/0.02)
		level := float32(peak / loudest * sample * fade)
		samples[frame*soundtrack.Channels] = level
		samples[frame*soundtrack.Channels+1] = level
	}
	return samples
}

// popEnvelope shapes a pop some seconds after it bursts: an 8 ms attack, a 45 ms hold and a quick decay.
func popEnvelope(since float64) float64 {
	switch {
	case since < 0:
		return 0
	case since < 0.008:
		return since / 0.008
	case since < 0.053:
		return 1
	}
	return math.Exp(-(since - 0.053) / 0.02)
}

// sizzleEnvelope shapes the crackle a pop leaves behind, lingering for a few tenths of a second.
func sizzleEnvelope(since float64) float64 {
	if since < 0 {
		return 0
	}
	return 0.45 * math.Min(1, since/0.02) * math.Exp(-since/0.17)
}
