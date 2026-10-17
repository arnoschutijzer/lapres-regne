package speaker

import (
	"math"

	"lapresregne/internal/soundtrack"
	"lapresregne/internal/world"
)

// The game climbs in silence, so these sounds are designed rather than matched to a recording:
// footfalls muffled on the steps of a staircase, and knocks of hands and feet on a ladder's rungs.
var (
	// stairSteps are the moments of the footfalls up or down a staircase, in seconds.
	stairSteps = [...]float64{0, 0.12, 0.24, 0.36}
	// ladderRungs are the moments of the knocks on a ladder's rungs, in seconds.
	ladderRungs = [...]float64{0, 0.17, 0.34}
)

// climbSound synthesizes climbing a staircase or ladder. On a staircase each footfall lands as a
// dull thud through the step's low resonances with a faint scuff of the sole; on a ladder each
// rung knocks twice, hand then foot, ringing higher through the wood.
func climbSound(flight world.Fixture) []float32 {
	const volume = 0.35
	moments, duration := stairSteps[:], 0.52
	if flight == world.Ladder {
		moments, duration = ladderRungs[:], 0.55
	}
	frames := int(duration * soundtrack.SampleRate)
	// The step rings low and broad under a foot; a rung rings higher and narrower.
	low, body := newResonance(110, 1.5), newResonance(260, 2)
	rung, rungAgain := newResonance(780, 5), newResonance(1150, 5)
	// A fixed noise seed makes every climb sound the same.
	noise := uint32(0x2545f491)
	random := func() float64 {
		noise ^= noise << 13
		noise ^= noise >> 17
		noise ^= noise << 5
		return float64(noise) / math.MaxUint32
	}
	var dull, darker float64
	climb := make([]float64, frames)
	for frame := range frames {
		t := float64(frame) / soundtrack.SampleRate
		white := 2*random() - 1
		dull += (white - dull) * 0.2
		darker += (dull - darker) * 0.1
		strike, scuff := 0.0, 0.0
		for i, moment := range moments {
			// Each footfall or knock lands a little softer than the one before, as the hero
			// settles into the climb.
			strength := 1 - 0.12*float64(i)
			if flight == world.Ladder {
				strike += strength * (landing(t, moment, 0.006) + 0.6*landing(t, moment+0.05, 0.006))
				continue
			}
			strike += strength * landing(t, moment, 0.014)
			scuff += strength * landing(t, moment+0.01, 0.025)
		}
		sound := strike * (8*low.filter(darker) + 5*body.filter(darker) + darker)
		if flight == world.Ladder {
			sound = strike * (3*rung.filter(white) + 2*rungAgain.filter(white) + 3*low.filter(darker) + dull)
		}
		climb[frame] = math.Tanh(sound + 0.06*scuff*dull)
	}
	muffle(climb, doorBrightness)
	samples := make([]float32, frames*soundtrack.Channels)
	for frame, sample := range climb {
		samples[frame*soundtrack.Channels] = float32(volume * sample)
		samples[frame*soundtrack.Channels+1] = float32(volume * sample)
	}
	return samples
}
