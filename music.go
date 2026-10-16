package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"time"

	"github.com/ebitengine/oto/v3"
)

const (
	musicSampleRate    = 44100
	musicChannels      = 2
	musicFadeFrames    = 3 * musicSampleRate
	musicSilenceFrames = musicSampleRate / 4
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

type musicTrack struct {
	samples          interface{ Read([]float32) (int, error) }
	frames, position int64
	closer           io.Closer
}

func (t *musicTrack) Close() {
	if t.closer != nil {
		t.closer.Close()
	}
}

type musicPhase byte

const (
	musicFadeIn musicPhase = iota
	musicPlaying
	musicFadeOut
	musicSilence
)

// musicStream produces sequential fades on the audio thread, without
// timers on the rendering thread or decoding entire songs into memory.
type musicStream struct {
	selectTrack                func(direction int) (*musicTrack, error)
	current                    *musicTrack
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

func newMusicStream(selectTrack func(int) (*musicTrack, error), fadeFrames, silenceFrames int64) (*musicStream, error) {
	if fadeFrames < 1 || silenceFrames < 1 {
		return nil, fmt.Errorf("fade and silence durations must be positive")
	}
	m := &musicStream{selectTrack: selectTrack, fadeFrames: fadeFrames, silenceFrames: silenceFrames, direction: 1, requests: make(chan int, 1)}
	var err error
	m.current, err = m.loadTrack(1)
	if err == nil {
		m.beginFadeIn()
	}
	return m, err
}

func (m *musicStream) loadTrack(direction int) (*musicTrack, error) {
	track, err := m.selectTrack(direction)
	if err != nil {
		return nil, err
	}
	if track.frames < 2 {
		track.Close()
		return nil, fmt.Errorf("song must contain at least two audio frames")
	}
	return track, nil
}

// Skip is nonblocking. While fading out, the latest request chooses which song
// starts after the silence, so rapid navigation never builds up a queue.
func (m *musicStream) Skip(direction int) {
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

func readSamples(track *musicTrack, samples []float32) error {
	for len(samples) > 0 {
		n, err := track.samples.Read(samples)
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

func (m *musicStream) mix() (int, error) {
	select {
	case direction := <-m.requests:
		m.direction = direction
		if m.phase == musicPlaying || m.phase == musicFadeIn {
			m.beginFadeOut()
		}
	default:
	}
	frames := int64(len(m.a) / musicChannels)
	if m.phase == musicSilence {
		frames = min(frames, m.phaseLength-m.phasePosition)
		clear(m.a[:int(frames)*musicChannels])
	} else {
		fadeStart := m.current.frames - min(m.fadeFrames, m.current.frames/2)
		if m.phase == musicPlaying && m.current.position >= fadeStart {
			m.direction = 1
			m.beginFadeOut()
		}
		frames = min(frames, m.current.frames-m.current.position)
		if m.phase == musicPlaying {
			frames = min(frames, fadeStart-m.current.position)
		} else {
			frames = min(frames, m.phaseLength-m.phasePosition)
		}
		if err := readSamples(m.current, m.a[:int(frames)*musicChannels]); err != nil {
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
			for channel := range musicChannels {
				m.a[frame*musicChannels+channel] *= float32(gain)
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
	return int(frames) * musicChannels, nil
}

func (m *musicStream) beginFadeIn() {
	m.phase, m.phasePosition = musicFadeIn, 0
	m.phaseLength = min(m.fadeFrames, m.current.frames/2)
	m.gain = 0
}

func (m *musicStream) beginFadeOut() {
	m.phase, m.phasePosition = musicFadeOut, 0
	m.phaseLength = min(m.fadeFrames, m.current.frames-m.current.position)
	m.fadeOutGain = m.gain
}

func (m *musicStream) finishPhase() error {
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

func (m *musicStream) Read(p []byte) (int, error) {
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

type musicPlayback struct {
	context *oto.Context
	player  *oto.Player
	stream  *musicStream
}

func startMusic(directory string) (*musicPlayback, error) {
	songs, err := songsInFolder(directory)
	if err != nil || len(songs) == 0 {
		return nil, err
	}
	history := &songHistory{shuffle: shuffledSongs{songs: songs}}
	stream, err := newMusicStream(func(direction int) (*musicTrack, error) {
		if direction < 0 {
			return openSong(history.Previous())
		}
		return openSong(history.Next())
	}, musicFadeFrames, musicSilenceFrames)
	if err != nil {
		return nil, err
	}
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: musicSampleRate, ChannelCount: musicChannels,
		Format: oto.FormatFloat32LE, BufferSize: 100 * time.Millisecond,
	})
	if err != nil {
		stream.current.Close()
		return nil, err
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		stream.current.Close()
		return nil, fmt.Errorf("audio device initialization timed out")
	}
	if err := context.Err(); err != nil {
		stream.current.Close()
		return nil, err
	}
	player := context.NewPlayer(stream)
	// Avoid half a second of decoded audio delaying navigation requests.
	player.SetBufferSize(musicSampleRate * musicChannels * 4 / 10)
	player.SetVolume(0.35)
	player.Play()
	return &musicPlayback{context: context, player: player, stream: stream}, nil
}

func (m *musicPlayback) Skip(direction int) {
	m.stream.Skip(direction)
}

func (m *musicPlayback) Err() error {
	if err := m.context.Err(); err != nil {
		return err
	}
	return m.player.Err()
}

func (m *musicPlayback) Close() {
	m.player.PauseAndStopReading()
	if m.stream.current != nil {
		m.stream.current.Close()
	}
	m.context.Suspend()
}
