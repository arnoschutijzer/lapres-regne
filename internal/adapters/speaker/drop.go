package speaker

import (
	"math"

	"lapresregne/internal/soundtrack"
)

// dropSound synthesizes a log landing on the ground: a dull wooden clunk through the log's low
// resonances, then a softer rebound as it settles.
func dropSound() []float32 {
	const duration, volume = 0.25, 0.28
	const rebound, reboundLevel = 0.065, 0.35
	frames := int(duration * soundtrack.SampleRate)
	// The log's hollow body rings low, with a little knock above it.
	body, bodyAgain := newResonance(240, 3), newResonance(240, 3)
	knock := newResonance(520, 4)
	// A fixed noise seed makes every drop sound the same.
	noise := uint32(0x6c8e9cf5)
	clunk := make([]float64, frames)
	for frame := range frames {
		t := float64(frame) / soundtrack.SampleRate
		noise ^= noise << 13
		noise ^= noise >> 17
		noise ^= noise << 5
		white := float64(noise)/math.MaxUint32*2 - 1
		envelope := landing(t, 0, 0.012) + reboundLevel*landing(t, rebound, 0.008)
		wood := 30*bodyAgain.filter(body.filter(white)) + 6*knock.filter(white)
		clunk[frame] = math.Tanh(envelope * wood)
	}
	samples := make([]float32, frames*soundtrack.Channels)
	for frame, sample := range clunk {
		samples[frame*soundtrack.Channels] = float32(volume * sample)
		samples[frame*soundtrack.Channels+1] = float32(volume * sample)
	}
	return samples
}
