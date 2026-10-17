package speaker

import (
	"math"

	"lapresregne/internal/soundtrack"
)

// Moments of the door's sound, in seconds, after Old School RuneScape's: the latch clatters as
// the door starts to swing, the door rattles on its hinges, and it knocks against its frame or
// stop. Closing ends there; an opening door's hinge creaks on, sliding down in pitch until the
// door has swung right back.
const (
	doorUnlatch   = 0.06
	doorKnock     = 0.165
	doorCloseEnd  = 0.215
	doorCreak     = 0.19
	doorCreakFade = 0.37
	doorOpenEnd   = 0.48
)

// doorClatter lists the latch's clacks as the door starts to swing, each a moment in seconds after
// doorUnlatch and its strength.
var doorClatter = [...]struct{ at, strength float64 }{{0.001, 0.5}, {0.006, 0.7}, {0.011, 1}}

// creakGlide is the pitch of an opening door's creak, in hertz, at moments in seconds: held high
// as the door leaves its knock, falling fast, then sliding slowly down as it slows.
var creakGlide = [...]struct{ at, pitch float64 }{
	{0.19, 480}, {0.215, 480}, {0.23, 420}, {0.245, 361}, {0.26, 313}, {0.275, 259}, {0.29, 214},
	{0.305, 202}, {0.32, 192}, {0.335, 177}, {0.35, 166}, {0.365, 149}, {0.38, 137}, {0.395, 130},
	{0.41, 111}, {0.425, 101}, {0.44, 89}, {0.455, 81}, {0.48, 75},
}

// creakHarmonics are the strengths of the creak's harmonics, from its fundamental up.
var creakHarmonics = [...]float64{1, 0.47, 0.28, 0.19, 0.15, 0.12, 0.1, 0.08, 0.07, 0.063, 0.056, 0.05}

// doorSound synthesizes a wooden door swinging, after Old School RuneScape's: a soft low rumble
// as the hinges take the weight, a clatter of the latch, a rattle of small knocks through the
// door's wooden body as it swings, and a brighter knock as it meets its frame. An opening door's
// hinge creaks on after the knock, sliding down in pitch.
func doorSound(opening bool) []float32 {
	const volume = 0.45
	duration := doorCloseEnd
	if opening {
		duration = doorOpenEnd
	}
	frames := int(duration * soundtrack.SampleRate)
	// The door's wooden body rings low and broad wherever it is struck.
	boom, body, panel := newResonance(110, 1.5), newResonance(230, 1.5), newResonance(520, 1.5)
	// The frame takes the knock higher.
	knock, knockAgain, knockHigh := newResonance(490, 4), newResonance(700, 4), newResonance(1250, 4)
	// The hinges' rumble is low and faintly tuned.
	hum, humAgain := newResonance(160, 3), newResonance(260, 3)
	// A fixed noise seed makes every door sound the same.
	noise := uint32(0x51ed270b)
	random := func() float64 {
		noise ^= noise << 13
		noise ^= noise >> 17
		noise ^= noise << 5
		return float64(noise) / math.MaxUint32
	}
	var dull, darker, nextRattle, rattle, creakPhase float64
	nextRattle = doorUnlatch + 0.016
	door := make([]float64, frames)
	for frame := range frames {
		t := float64(frame) / soundtrack.SampleRate
		white := 2*random() - 1
		dull += (white - dull) * 0.25
		darker += (dull - darker) * 0.12

		// A soft low rumble as the hinges take the door's weight.
		rumbleLevel := math.Min(1, t/0.008) * math.Exp(-math.Max(0, t-doorUnlatch)/0.02)
		rumble := 0.5 * rumbleLevel * (hum.filter(white) + 0.6*humAgain.filter(white))

		// The latch clatters, and the door rattles on with small knocks of every strength until
		// it meets its frame.
		strike := 0.0
		for _, clack := range doorClatter {
			strike += clack.strength * landing(t, doorUnlatch+clack.at, 0.003)
		}
		if t >= nextRattle && t < doorKnock {
			rattle = 0.15 + 0.35*random()
			nextRattle = t + 0.002 + 0.006*random()
		}
		rattle *= math.Exp(-1 / (0.003 * soundtrack.SampleRate))
		strike += rattle
		wood := strike * (9*boom.filter(darker) + 7*body.filter(darker) + 6*panel.filter(darker) + 2*darker)

		// The knock as the door meets its frame.
		blow := (landing(t, doorKnock, 0.004) + 0.7*landing(t, doorKnock+0.004, 0.009)) *
			(4*knock.filter(white) + 3.4*knockAgain.filter(white) + 2*knockHigh.filter(white) + 0.8*dull)

		// An opening door's hinge creaks on, sliding down in pitch, a little rough.
		creak := 0.0
		if opening && t >= doorCreak {
			creakPhase += creakPitch(t) / soundtrack.SampleRate
			voice := 0.0
			for harmonic, strength := range creakHarmonics {
				voice += strength * math.Sin(2*math.Pi*float64(harmonic+1)*creakPhase)
			}
			level := math.Min(1, (t-doorCreak)/0.04) * math.Exp(-math.Max(0, t-doorCreakFade)/0.035)
			creak = 0.075 * level * (voice*(1+0.25*white) + 0.3*dull)
		}

		door[frame] = math.Tanh(rumble + wood + blow + creak)
	}
	muffle(door, doorBrightness)
	samples := make([]float32, frames*soundtrack.Channels)
	for frame, sample := range door {
		samples[frame*soundtrack.Channels] = float32(volume * sample)
		samples[frame*soundtrack.Channels+1] = float32(volume * sample)
	}
	return samples
}

// doorBrightness is how high the door's sound reaches, in hertz: the game's recording is sampled
// at half the usual rate and holds little above a few kilohertz, so anything higher is static.
const doorBrightness = 5000

// muffle cuts a mono signal's frequencies above a cutoff, falling away steeply through three
// low-pass stages.
func muffle(signal []float64, cutoff float64) {
	smoothing := 1 - math.Exp(-2*math.Pi*cutoff/soundtrack.SampleRate)
	for range 3 {
		previous := 0.0
		for i, sample := range signal {
			previous += (sample - previous) * smoothing
			signal[i] = previous
		}
	}
}

// creakPitch is the pitch of the creak at a moment, gliding between the moments of creakGlide.
func creakPitch(t float64) float64 {
	for i := 1; i < len(creakGlide); i++ {
		if from, to := creakGlide[i-1], creakGlide[i]; t < to.at {
			u := math.Max(0, (t-from.at)/(to.at-from.at))
			return from.pitch * math.Pow(to.pitch/from.pitch, u)
		}
	}
	return creakGlide[len(creakGlide)-1].pitch
}
