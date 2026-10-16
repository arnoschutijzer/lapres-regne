package speaker

import (
	"math"

	"lapresregne/internal/soundtrack"
)

// fallLanding is when the falling tree hits the ground, in seconds.
const fallLanding = 0.55

// fallSound synthesizes a tree coming down: the trunk creaks as its last fibres tear, the crown
// rushes through the air, and the tree lands with a low thud and a smaller bounce.
func fallSound() []float32 {
	const duration, volume = 1.4, 0.32
	frames := int(duration * soundtrack.SampleRate)
	// Two resonances give the creak the hollow voice of a trunk.
	trunk, trunkAgain := newResonance(700, 6), newResonance(1150, 5)
	// A wide band high up is the leaves rushing.
	leaves := newResonance(2800, 0.8)
	// Two low resonances give the landing its weight.
	ground, groundAgain := newResonance(90, 2), newResonance(170, 2.5)
	// A fixed noise seed makes every fall sound the same.
	noise := uint32(0x2545f491)
	// phase counts through the creak's fibre ticks.
	phase := 0.0
	fall := make([]float64, frames)
	for frame := range frames {
		t := float64(frame) / soundtrack.SampleRate
		noise ^= noise << 13
		noise ^= noise >> 17
		noise ^= noise << 5
		white := float64(noise)/math.MaxUint32*2 - 1

		// The creak ticks ever slower as the trunk gives way, swelling and then cut off by the landing.
		rate := 55 - 30*t/fallLanding
		phase += rate / soundtrack.SampleRate
		tick := 0.0
		if phase >= 1 {
			phase--
			tick = 1 + 0.5*white
		}
		creakLevel := 0.0
		if t < fallLanding {
			creakLevel = math.Sin(math.Pi*t/fallLanding) * math.Min(1, (fallLanding-t)/0.05)
		}
		creak := 4 * creakLevel * trunkAgain.filter(trunk.filter(tick))

		// The crown rushes through the air on the way down and settles after landing.
		rushLevel := 0.0
		if t > 0.25 {
			rushLevel = math.Min(1, (t-0.25)/0.3) * math.Exp(-math.Max(0, t-fallLanding)/0.18)
		}
		rush := 0.35 * rushLevel * leaves.filter(white)

		// The landing thuds and bounces once, both dying away.
		thudLevel := landing(t, fallLanding, 0.16) + 0.4*landing(t, fallLanding+0.17, 0.1)
		thud := thudLevel * (5*ground.filter(white) + 2*groundAgain.filter(white))

		fall[frame] = math.Tanh(creak + rush + thud)
	}
	samples := make([]float32, frames*soundtrack.Channels)
	for frame, sample := range fall {
		samples[frame*soundtrack.Channels] = float32(volume * sample)
		samples[frame*soundtrack.Channels+1] = float32(volume * sample)
	}
	return samples
}

// landing is the envelope of an impact at a moment: an instant attack decaying over a time constant.
func landing(t, at, decay float64) float64 {
	if t < at {
		return 0
	}
	return math.Min(1, (t-at)/0.002) * math.Exp(-(t-at)/decay)
}
