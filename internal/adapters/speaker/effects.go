package speaker

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/ebitengine/oto/v3"

	"dither-cli/internal/soundtrack"
)

// Effects plays short synthesized sound effects on the device.
type Effects struct {
	context *oto.Context
	chop    []byte
	// playing keeps players referenced until their sound has finished.
	playing []*oto.Player
}

func newEffects(context *oto.Context) *Effects {
	return &Effects{context: context, chop: encode(chopSound())}
}

// Chop plays a soft wooden thunk as an axe lands.
func (e *Effects) Chop() {
	e.play(e.chop)
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

// chopSound synthesizes a muffled knock of wood: a low thump that sinks in pitch under a
// brief, filtered crack, both fading out within a few hundredths of a second.
func chopSound() []float32 {
	const duration, volume = 0.16, 0.3
	frames := int(duration * soundtrack.SampleRate)
	samples := make([]float32, frames*soundtrack.Channels)
	// A fixed noise seed makes every chop sound the same.
	noise := uint32(0x9e3779b9)
	var fast, slow, phase float64
	for frame := range frames {
		t := float64(frame) / soundtrack.SampleRate
		frequency := 90 + 70*math.Exp(-t/0.02)
		phase += 2 * math.Pi * frequency / soundtrack.SampleRate
		thump := math.Sin(phase) * math.Exp(-t/0.035)
		noise ^= noise << 13
		noise ^= noise >> 17
		noise ^= noise << 5
		white := float64(noise)/math.MaxUint32*2 - 1
		// Two low-pass filters subtracted keep the woody middle of the noise.
		fast += (white - fast) * 0.35
		slow += (fast - slow) * 0.05
		crack := (fast - slow) * math.Exp(-t/0.012)
		// A short fade-in avoids a click at the very first sample.
		attack := math.Min(1, t/0.002)
		sample := float32(volume * attack * (0.75*thump + 0.6*crack))
		samples[frame*soundtrack.Channels] = sample
		samples[frame*soundtrack.Channels+1] = sample
	}
	return samples
}

func encode(samples []float32) []byte {
	data := make([]byte, len(samples)*4)
	for i, sample := range samples {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(sample))
	}
	return data
}
