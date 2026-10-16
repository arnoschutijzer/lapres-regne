package speaker

import (
	"math"

	"dither-cli/internal/soundtrack"
)

// voice is one instrument of the level-up jingle.
type voice int

const (
	// stab is a short bright chord that opens each phrase.
	stab voice = iota
	// knock is a low wooden drum, like an axe landing.
	knock
	// pad holds the finale's chord.
	pad
	// pluck is a soft bell twinkling over the finale.
	pluck
	// lead sings the melody of the jingle for a level that unlocks a tree.
	lead
)

// note is one sound in the jingle's score.
type note struct {
	// at is when the note starts and held how long it is held, in seconds.
	at, held float64
	// pitch is a MIDI note number: 60 is middle C.
	pitch int
	voice voice
	level float64
}

// jingleLength and jinglePeak are the length of the jingle in seconds and its loudest sample.
const jingleLength, jinglePeak = 6.0, 0.3

// levelUpScore follows Old School RuneScape's woodcutting level-up jingle in C major: two phrases of
// a stab answered by four knocks, then a held C chord under twinkling bells.
var levelUpScore = []note{
	{at: 0, held: 0.3, pitch: 67, voice: stab, level: 1},
	{at: 0, held: 0.3, pitch: 72, voice: stab, level: 0.6},
	{at: 0.30, voice: knock, level: 1.12},
	{at: 0.44, voice: knock, level: 1.6},
	{at: 0.60, voice: knock, level: 1.44},
	{at: 0.90, voice: knock, level: 1.44},
	{at: 1.20, held: 0.3, pitch: 60, voice: stab, level: 0.5},
	{at: 1.20, held: 0.3, pitch: 67, voice: stab, level: 1},
	{at: 1.50, voice: knock, level: 1.12},
	{at: 1.64, voice: knock, level: 1.6},
	{at: 1.80, voice: knock, level: 1.44},
	{at: 2.10, voice: knock, level: 1.44},
	{at: 2.38, held: 1.77, pitch: 48, voice: pad, level: 0.25},
	{at: 2.38, held: 1.77, pitch: 60, voice: pad, level: 0.18},
	{at: 2.38, held: 1.77, pitch: 67, voice: pad, level: 0.16},
	{at: 2.38, held: 1.77, pitch: 72, voice: pad, level: 0.14},
	{at: 2.40, pitch: 79, voice: pluck, level: 0.23},
	{at: 2.55, pitch: 79, voice: pluck, level: 0.2},
	{at: 2.63, pitch: 76, voice: pluck, level: 0.16},
	{at: 2.70, pitch: 84, voice: pluck, level: 0.14},
	{at: 2.75, pitch: 76, voice: pluck, level: 0.14},
	{at: 2.80, pitch: 79, voice: pluck, level: 0.16},
	{at: 2.85, pitch: 86, voice: pluck, level: 0.14},
	{at: 2.90, pitch: 84, voice: pluck, level: 0.14},
	{at: 2.95, pitch: 91, voice: pluck, level: 0.14},
	{at: 3.05, pitch: 84, voice: pluck, level: 0.18},
	{at: 3.15, pitch: 86, voice: pluck, level: 0.14},
	{at: 3.20, pitch: 79, voice: pluck, level: 0.14},
	{at: 3.25, pitch: 76, voice: pluck, level: 0.14},
	{at: 3.28, pitch: 84, voice: pluck, level: 0.14},
	{at: 3.37, pitch: 76, voice: pluck, level: 0.14},
	{at: 3.40, pitch: 79, voice: pluck, level: 0.14},
	{at: 3.45, pitch: 91, voice: pluck, level: 0.14},
	{at: 3.50, pitch: 79, voice: pluck, level: 0.27},
	{at: 3.82, pitch: 86, voice: pluck, level: 0.18},
	{at: 3.90, pitch: 79, voice: pluck, level: 0.16},
	{at: 3.97, pitch: 91, voice: pluck, level: 0.16},
	{at: 4.07, pitch: 84, voice: pluck, level: 0.18},
	{at: 4.12, pitch: 86, voice: pluck, level: 0.14},
	{at: 4.20, pitch: 86, voice: pluck, level: 0.11},
	{at: 4.30, pitch: 84, voice: pluck, level: 0.09},
	{at: 4.52, pitch: 79, voice: pluck, level: 0.05},
}

// unlockScore is what the jingle adds for a level that unlocks a tree: a melody climbing to E and
// stepping down from G, with chords leading it home to C.
var unlockScore = []note{
	{at: 0.20, held: 0.08, pitch: 67, voice: lead, level: 0.5},
	{at: 0.30, held: 0.13, pitch: 72, voice: lead, level: 0.55},
	{at: 0.45, held: 0.13, pitch: 74, voice: lead, level: 0.55},
	{at: 0.60, held: 0.25, pitch: 76, voice: lead, level: 0.6},
	{at: 0.60, held: 0.2, pitch: 55, voice: stab, level: 0.5},
	{at: 0.60, held: 0.2, pitch: 60, voice: stab, level: 0.4},
	{at: 0.60, held: 0.2, pitch: 64, voice: stab, level: 0.4},
	{at: 0.90, held: 0.25, pitch: 72, voice: lead, level: 0.55},
	{at: 1.20, held: 0.25, pitch: 79, voice: lead, level: 0.5},
	{at: 1.50, held: 0.25, pitch: 77, voice: lead, level: 0.6},
	{at: 1.80, held: 0.25, pitch: 76, voice: lead, level: 0.55},
	{at: 1.80, held: 0.2, pitch: 55, voice: stab, level: 0.5},
	{at: 1.80, held: 0.2, pitch: 59, voice: stab, level: 0.4},
	{at: 1.80, held: 0.2, pitch: 62, voice: stab, level: 0.4},
	{at: 2.10, held: 0.25, pitch: 74, voice: lead, level: 0.6},
	{at: 2.40, held: 0.55, pitch: 72, voice: lead, level: 0.6},
	{at: 3.00, held: 0.2, pitch: 52, voice: stab, level: 0.5},
	{at: 3.00, held: 0.2, pitch: 55, voice: stab, level: 0.5},
	{at: 3.00, held: 0.2, pitch: 60, voice: stab, level: 0.4},
	{at: 3.15, held: 0.3, pitch: 72, voice: lead, level: 0.45},
	{at: 3.60, held: 0.2, pitch: 52, voice: stab, level: 0.5},
	{at: 3.60, held: 0.2, pitch: 55, voice: stab, level: 0.5},
}

// levelUpSound synthesizes the level-up jingle, with the lead melody when the level unlocks a tree.
func levelUpSound(unlocked bool) []float32 {
	frames := int(jingleLength * soundtrack.SampleRate)
	mixed := make([]float64, frames)
	score := levelUpScore
	if unlocked {
		score = append(append([]note(nil), levelUpScore...), unlockScore...)
	}
	for _, n := range score {
		n.play(mixed)
	}
	loudest := 0.0
	for _, sample := range mixed {
		loudest = math.Max(loudest, math.Abs(sample))
	}
	samples := make([]float32, frames*soundtrack.Channels)
	for frame, sample := range mixed {
		samples[frame*soundtrack.Channels] = float32(jinglePeak * sample / loudest)
		samples[frame*soundtrack.Channels+1] = float32(jinglePeak * sample / loudest)
	}
	return samples
}

// play adds the note to the mix until it has died away or the jingle ends.
func (n note) play(mixed []float64) {
	frequency := 440 * math.Pow(2, float64(n.pitch-69)/12)
	// The knock rings through a log drum's two lowest modes.
	body, overtone := newResonance(210, 6), newResonance(340, 8)
	noise := uint32(0x2545f491)
	start := int(n.at * soundtrack.SampleRate)
	for frame := start; frame < len(mixed); frame++ {
		t := float64(frame-start) / soundtrack.SampleRate
		loudness := n.envelope(t)
		if t > n.held && loudness < 1e-4 {
			return
		}
		phase := 2 * math.Pi * frequency * t
		var wave float64
		switch n.voice {
		case knock:
			noise ^= noise << 13
			noise ^= noise >> 17
			noise ^= noise << 5
			white := float64(noise)/math.MaxUint32*2 - 1
			wave = 6*body.filter(white) + 3*overtone.filter(white)
		case pluck:
			wave = math.Sin(phase) + 0.15*math.Sin(2*phase)
		default:
			// A warm tone: the fundamental with a little of its second and third harmonics.
			wave = math.Sin(phase) + 0.35*math.Sin(2*phase) + 0.12*math.Sin(3*phase)
		}
		mixed[frame] += n.level * loudness * wave
	}
}

// envelope is how loud the note sounds t seconds after it starts.
func (n note) envelope(t float64) float64 {
	attack := func(seconds float64) float64 { return math.Min(1, t/seconds) }
	// release fades a held voice out once its note ends.
	release := func(seconds float64) float64 {
		if t <= n.held {
			return 1
		}
		return math.Exp(-(t - n.held) / seconds)
	}
	switch n.voice {
	case stab:
		return attack(0.004) * (0.75*math.Exp(-t/0.07) + 0.25*math.Exp(-t/0.35)) * release(0.08)
	case knock:
		return attack(0.001) * math.Exp(-t/0.04)
	case pad:
		return attack(0.04) * (0.7 + 0.3*math.Exp(-t/0.3)) * release(0.18)
	case pluck:
		return attack(0.002) * math.Exp(-t/0.18)
	default:
		return attack(0.01) * math.Exp(-t/0.4) * release(0.06)
	}
}
