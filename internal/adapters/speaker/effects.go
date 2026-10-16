package speaker

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/ebitengine/oto/v3"

	"lapresregne/internal/soundtrack"
)

// Effects plays short synthesized sound effects on the device.
type Effects struct {
	context *oto.Context
	chop    []byte
	// levelUp holds the level-up jingle, and unlockingLevelUp the one for a level that unlocks a tree.
	levelUp, unlockingLevelUp []byte
	// playing keeps players referenced until their sound has finished.
	playing []*oto.Player
}

func newEffects(context *oto.Context) *Effects {
	return &Effects{
		context:          context,
		chop:             encode(chopSound()),
		levelUp:          encode(levelUpSound(false)),
		unlockingLevelUp: encode(levelUpSound(true)),
	}
}

// Chop plays a woody tok as an axe lands.
func (e *Effects) Chop() {
	e.play(e.chop)
}

// LevelUp plays the level-up jingle, with its melody when the level unlocks a tree.
func (e *Effects) LevelUp(unlocked bool) {
	if unlocked {
		e.play(e.unlockingLevelUp)
	} else {
		e.play(e.levelUp)
	}
}

func (e *Effects) play(sound []byte) {
	playing := e.playing[:0]
	for _, player := range e.playing {
		if player.IsPlaying() {
			playing = append(playing, player)
		}
	}
	player := e.context.NewPlayer(bytes.NewReader(sound))
	player.Play()
	e.playing = append(playing, player)
}

// chopSound synthesizes a woody tok after Old School RuneScape's woodcutting sound: a burst of
// noise ringing through the wood's resonances near 1.2, 1.6 and 4.7 kHz over a dull body,
// slightly overdriven, held for about twelve milliseconds and answered by a faint echo.
func chopSound() []float32 {
	const duration, volume = 0.07, 0.3
	const hold, release, echoDelay, echoLevel = 0.011, 0.003, 0.03, 0.1
	frames := int(duration * soundtrack.SampleRate)
	// Two resonances in a row give the ring the steep sides of the original's.
	ring, ringAgain := newResonance(1600, 4), newResonance(1600, 4)
	knock := newResonance(1200, 3)
	sparkle := newResonance(4700, 8)
	// A fixed noise seed makes every chop sound the same.
	noise := uint32(0x9e3779b9)
	var body, dull float64
	tok := make([]float64, frames)
	for frame := range frames {
		t := float64(frame) / soundtrack.SampleRate
		envelope := math.Min(1, t/0.0004)
		if t > hold {
			envelope = math.Exp(-(t - hold) / release)
		}
		noise ^= noise << 13
		noise ^= noise >> 17
		noise ^= noise << 5
		white := float64(noise)/math.MaxUint32*2 - 1
		// Two low-pass stages keep the body dull.
		body += (white - body) * 0.15
		dull += (body - dull) * 0.15
		wood := 2.6*ringAgain.filter(ring.filter(white)) + 1.3*knock.filter(white) + 0.3*sparkle.filter(white) + 0.6*dull
		// Gentle overdrive flattens the peaks like the original's crunchy sample.
		tok[frame] = math.Tanh(1.2*envelope*wood) / math.Tanh(1.2)
	}
	samples := make([]float32, frames*soundtrack.Channels)
	echo := int(echoDelay * soundtrack.SampleRate)
	for frame := range frames {
		sample := tok[frame]
		if frame >= echo {
			sample += echoLevel * tok[frame-echo]
		}
		samples[frame*soundtrack.Channels] = float32(volume * sample)
		samples[frame*soundtrack.Channels+1] = float32(volume * sample)
	}
	return samples
}

// resonance is a band-pass filter that rings at one frequency, peaking at unity gain.
type resonance struct {
	b0, b2, a1, a2 float64
	x1, x2, y1, y2 float64
}

func newResonance(frequency, q float64) *resonance {
	w := 2 * math.Pi * frequency / soundtrack.SampleRate
	alpha := math.Sin(w) / (2 * q)
	a0 := 1 + alpha
	return &resonance{b0: alpha / a0, b2: -alpha / a0, a1: -2 * math.Cos(w) / a0, a2: (1 - alpha) / a0}
}

func (r *resonance) filter(x float64) float64 {
	y := r.b0*x + r.b2*r.x2 - r.a1*r.y1 - r.a2*r.y2
	r.x2, r.x1 = r.x1, x
	r.y2, r.y1 = r.y1, y
	return y
}

func encode(samples []float32) []byte {
	data := make([]byte, len(samples)*4)
	for i, sample := range samples {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(sample))
	}
	return data
}
