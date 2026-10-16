package soundtrack

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
)

const (
	SampleRate           = 44100
	Channels             = 2
	defaultFadeFrames    = 3 * SampleRate
	defaultSilenceFrames = SampleRate / 4
)

// Selection is independent of decoding and crossfading. A trigger-based selector
// can choose a song ID here without changing the playback engine.
type shuffledSongs struct {
	songs     []string
	remaining []string
	last      string
}

func (s *shuffledSongs) Next() string {
	if len(s.remaining) == 0 {
		s.remaining = append(s.remaining, s.songs...)
		rand.Shuffle(len(s.remaining), func(i, j int) { s.remaining[i], s.remaining[j] = s.remaining[j], s.remaining[i] })
		if len(s.remaining) > 1 && s.remaining[0] == s.last {
			s.remaining[0], s.remaining[1] = s.remaining[1], s.remaining[0]
		}
	}
	id := s.remaining[0]
	s.remaining = s.remaining[1:]
	s.last = id
	return id
}

type songHistory struct {
	shuffle shuffledSongs
	songs   []string
	index   int
}

func (h *songHistory) Next() string {
	if h.index+1 < len(h.songs) {
		h.index++
		return h.songs[h.index]
	}
	h.songs = append(h.songs, h.shuffle.Next())
	// Keep navigation memory bounded during long exploration sessions.
	if len(h.songs) > 64 {
		copy(h.songs, h.songs[len(h.songs)-64:])
		h.songs = h.songs[:64]
	}
	h.index = len(h.songs) - 1
	return h.songs[h.index]
}

func (h *songHistory) Previous() string {
	if len(h.songs) == 0 {
		return h.Next()
	}
	if h.index > 0 {
		h.index--
	}
	return h.songs[h.index]
}

type Track struct {
	Samples          Samples
	Frames, position int64
	Closer           io.Closer
}

func (t *Track) Close() {
	if t.Closer != nil {
		t.Closer.Close()
	}
}

type musicPhase byte

const (
	musicFadeIn musicPhase = iota
	musicPlaying
	musicFadeOut
	musicSilence
)

// Stream produces sequential fades on the audio thread, without
// timers on the rendering thread or decoding entire songs into memory.
type Stream struct {
	selectTrack                func(direction int) (*Track, error)
	current                    *Track
	fadeFrames, silenceFrames  int64
	phase                      musicPhase
	phasePosition, phaseLength int64
	gain, fadeOutGain          float64
	direction                  int
	requests                   chan int
	a                          [2048]float32
	encoded                    [8192]byte
	pending                    []byte
	err                        error
}

func newStream(selectTrack func(int) (*Track, error), fadeFrames, silenceFrames int64) (*Stream, error) {
	if fadeFrames < 1 || silenceFrames < 1 {
		return nil, fmt.Errorf("fade and silence durations must be positive")
	}
	m := &Stream{selectTrack: selectTrack, fadeFrames: fadeFrames, silenceFrames: silenceFrames, direction: 1, requests: make(chan int, 1)}
	var err error
	m.current, err = m.loadTrack(1)
	if err == nil {
		m.beginFadeIn()
	}
	return m, err
}

func (m *Stream) loadTrack(direction int) (*Track, error) {
	track, err := m.selectTrack(direction)
	if err != nil {
		return nil, err
	}
	if track.Frames < 2 {
		track.Close()
		return nil, fmt.Errorf("song must contain at least two audio frames")
	}
	return track, nil
}

// Skip is nonblocking. While fading out, the latest request chooses which song
// starts after the silence, so rapid navigation never builds up a queue.
func (m *Stream) Skip(direction int) {
	if direction == 0 {
		return
	}
	select {
	case <-m.requests:
	default:
	}
	select {
	case m.requests <- direction:
	default:
	}
}

func readSamples(track *Track, samples []float32) error {
	for len(samples) > 0 {
		n, err := track.Samples.Read(samples)
		samples = samples[n:]
		if len(samples) == 0 {
			return nil
		}
		if err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func (m *Stream) mix() (int, error) {
	select {
	case direction := <-m.requests:
		m.direction = direction
		if m.phase == musicPlaying || m.phase == musicFadeIn {
			m.beginFadeOut()
		}
	default:
	}
	frames := int64(len(m.a) / Channels)
	if m.phase == musicSilence {
		frames = min(frames, m.phaseLength-m.phasePosition)
		clear(m.a[:int(frames)*Channels])
	} else {
		fadeStart := m.current.Frames - min(m.fadeFrames, m.current.Frames/2)
		if m.phase == musicPlaying && m.current.position >= fadeStart {
			m.direction = 1
			m.beginFadeOut()
		}
		frames = min(frames, m.current.Frames-m.current.position)
		if m.phase == musicPlaying {
			frames = min(frames, fadeStart-m.current.position)
		} else {
			frames = min(frames, m.phaseLength-m.phasePosition)
		}
		if err := readSamples(m.current, m.a[:int(frames)*Channels]); err != nil {
			return 0, err
		}
		for frame := range int(frames) {
			gain := 1.0
			if m.phase != musicPlaying {
				progress := float64(m.phasePosition+int64(frame)+1) / float64(m.phaseLength)
				if m.phase == musicFadeIn {
					gain = math.Sin(progress * math.Pi / 2)
				} else if progress >= 1 {
					gain = 0
				} else {
					gain = m.fadeOutGain * math.Cos(progress*math.Pi/2)
				}
			}
			m.gain = gain
			for channel := range Channels {
				m.a[frame*Channels+channel] *= float32(gain)
			}
		}
		m.current.position += frames
	}
	if m.phase != musicPlaying {
		m.phasePosition += frames
		if m.phasePosition == m.phaseLength {
			if err := m.finishPhase(); err != nil {
				return 0, err
			}
		}
	}
	return int(frames) * Channels, nil
}

func (m *Stream) beginFadeIn() {
	m.phase, m.phasePosition = musicFadeIn, 0
	m.phaseLength = min(m.fadeFrames, m.current.Frames/2)
	m.gain = 0
}

func (m *Stream) beginFadeOut() {
	m.phase, m.phasePosition = musicFadeOut, 0
	m.phaseLength = min(m.fadeFrames, m.current.Frames-m.current.position)
	m.fadeOutGain = m.gain
}

func (m *Stream) finishPhase() error {
	switch m.phase {
	case musicFadeIn:
		m.phase = musicPlaying
	case musicFadeOut:
		m.current.Close()
		m.current = nil
		m.phase, m.phasePosition, m.phaseLength = musicSilence, 0, m.silenceFrames
	case musicSilence:
		var err error
		m.current, err = m.loadTrack(m.direction)
		if err != nil {
			return err
		}
		m.beginFadeIn()
	}
	return nil
}

func (m *Stream) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(m.pending) == 0 {
		if m.err != nil {
			return 0, m.err
		}
		n, err := m.mix()
		m.err = err
		if err != nil {
			return 0, err
		}
		for i, sample := range m.a[:n] {
			binary.LittleEndian.PutUint32(m.encoded[i*4:], math.Float32bits(sample))
		}
		m.pending = m.encoded[:n*4]
	}
	n := copy(p, m.pending)
	m.pending = m.pending[n:]
	return n, nil
}

// Shuffle streams the songs in shuffled order, opening each through the decoder.
func Shuffle(songs []string, decoder Decoder) (*Stream, error) {
	history := &songHistory{shuffle: shuffledSongs{songs: songs}}
	return newStream(func(direction int) (*Track, error) {
		if direction < 0 {
			return decoder.Open(history.Previous())
		}
		return decoder.Open(history.Next())
	}, defaultFadeFrames, defaultSilenceFrames)
}

// Close releases the current song. Call it only after the audio device stops reading.
func (m *Stream) Close() {
	if m.current != nil {
		m.current.Close()
	}
}
