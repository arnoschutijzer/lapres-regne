package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/jfreymuth/oggvorbis"
)

//go:embed assets/*.ogg
var soundtrackFiles embed.FS

const (
	musicSampleRate = 44100
	musicChannels   = 2
	musicFadeFrames = 3 * musicSampleRate
)

var forestSongs = []string{"harmony", "newbie-melody", "autumn-voyage", "adventure", "scape-soft"}

// Selection is independent of decoding and crossfading. A trigger-based selector
// can choose a song ID here without changing the playback engine.
type shuffledSongs struct {
	remaining []string
	last      string
}

func (s *shuffledSongs) Next() string {
	if len(s.remaining) == 0 {
		s.remaining = append(s.remaining, forestSongs...)
		rand.Shuffle(len(s.remaining), func(i, j int) { s.remaining[i], s.remaining[j] = s.remaining[j], s.remaining[i] })
		if s.remaining[0] == s.last {
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
}

func openSong(id string) (*musicTrack, error) {
	data, err := soundtrackFiles.ReadFile("assets/" + id + ".ogg")
	if err != nil {
		return nil, err
	}
	decoder, err := oggvorbis.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", id, err)
	}
	if decoder.SampleRate() != musicSampleRate || decoder.Channels() != musicChannels {
		return nil, fmt.Errorf("%s must be 44.1 kHz stereo", id)
	}
	return &musicTrack{samples: decoder, frames: decoder.Length()}, nil
}

// musicStream produces sample-accurate crossfades on the audio thread, without
// timers on the rendering thread or decoding entire songs into memory.
type musicStream struct {
	selectTrack       func(direction int) (*musicTrack, error)
	current, upcoming *musicTrack
	fadeFrames        int64
	fadePosition      int64
	requests          chan int
	a, b              [2048]float32
	encoded           [8192]byte
	pending           []byte
	err               error
}

func newMusicStream(selectTrack func(int) (*musicTrack, error), fadeFrames int64) (*musicStream, error) {
	m := &musicStream{selectTrack: selectTrack, fadeFrames: fadeFrames, requests: make(chan int, 1)}
	var err error
	m.current, err = m.loadTrack(1)
	return m, err
}

func (m *musicStream) loadTrack(direction int) (*musicTrack, error) {
	track, err := m.selectTrack(direction)
	if err != nil {
		return nil, err
	}
	if m.fadeFrames <= 0 || track.frames < 2*m.fadeFrames {
		return nil, fmt.Errorf("song must be at least twice the crossfade duration")
	}
	return track, nil
}

// Skip is nonblocking; decoder changes happen on the audio thread. If a fade is
// already in progress, finish it before handling the latest navigation request.
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
	current := m.current
	fadeStart := current.frames - m.fadeFrames
	if m.upcoming == nil {
		direction := 0
		select {
		case direction = <-m.requests:
		default:
			if current.position >= fadeStart {
				direction = 1
			}
		}
		if direction != 0 {
			var err error
			m.upcoming, err = m.loadTrack(direction)
			if err != nil {
				return 0, err
			}
			m.fadePosition = 0
		}
	}
	frames := min(int64(len(m.a)/musicChannels), current.frames-current.position)
	fading := m.upcoming != nil
	if !fading {
		frames = min(frames, fadeStart-current.position)
	} else {
		frames = min(frames, m.fadeFrames-m.fadePosition)
	}
	n := int(frames) * musicChannels
	if err := readSamples(current, m.a[:n]); err != nil {
		return 0, err
	}
	if fading {
		if err := readSamples(m.upcoming, m.b[:n]); err != nil {
			return 0, err
		}
		for frame := range int(frames) {
			progress := float64(m.fadePosition+int64(frame)) / float64(m.fadeFrames)
			incoming, outgoing := math.Sincos(progress * math.Pi / 2)
			for channel := range musicChannels {
				i := frame*musicChannels + channel
				m.a[i] = float32(float64(m.a[i])*outgoing + float64(m.b[i])*incoming)
			}
		}
		m.upcoming.position += frames
		m.fadePosition += frames
	}
	current.position += frames
	if fading && m.fadePosition == m.fadeFrames {
		m.current, m.upcoming = m.upcoming, nil
	}
	return n, nil
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

func startMusic() (*musicPlayback, error) {
	history := &songHistory{}
	stream, err := newMusicStream(func(direction int) (*musicTrack, error) {
		if direction < 0 {
			return openSong(history.Previous())
		}
		return openSong(history.Next())
	}, musicFadeFrames)
	if err != nil {
		return nil, err
	}
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: musicSampleRate, ChannelCount: musicChannels,
		Format: oto.FormatFloat32LE, BufferSize: 100 * time.Millisecond,
	})
	if err != nil {
		return nil, err
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("audio device initialization timed out")
	}
	if err := context.Err(); err != nil {
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
	m.context.Suspend()
}
