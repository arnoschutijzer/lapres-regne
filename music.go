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
	selectNext        func() (*musicTrack, error)
	current, upcoming *musicTrack
	fadeFrames        int64
	a, b              [2048]float32
	encoded           [8192]byte
	pending           []byte
	err               error
}

func newMusicStream(next func() (*musicTrack, error), fadeFrames int64) (*musicStream, error) {
	m := &musicStream{selectNext: next, fadeFrames: fadeFrames}
	var err error
	m.current, err = m.loadNext()
	return m, err
}

func (m *musicStream) loadNext() (*musicTrack, error) {
	track, err := m.selectNext()
	if err != nil {
		return nil, err
	}
	if m.fadeFrames <= 0 || track.frames < 2*m.fadeFrames {
		return nil, fmt.Errorf("song must be at least twice the crossfade duration")
	}
	return track, nil
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
	frames := min(int64(len(m.a)/musicChannels), current.frames-current.position)
	fading := current.position >= fadeStart
	if !fading {
		frames = min(frames, fadeStart-current.position)
	} else if m.upcoming == nil {
		var err error
		m.upcoming, err = m.loadNext()
		if err != nil {
			return 0, err
		}
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
			progress := float64(current.position+int64(frame)-fadeStart) / float64(m.fadeFrames)
			incoming, outgoing := math.Sincos(progress * math.Pi / 2)
			for channel := range musicChannels {
				i := frame*musicChannels + channel
				m.a[i] = float32(float64(m.a[i])*outgoing + float64(m.b[i])*incoming)
			}
		}
		m.upcoming.position += frames
	}
	current.position += frames
	if current.position == current.frames {
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
}

func startMusic() (*musicPlayback, error) {
	playlist := &shuffledSongs{}
	stream, err := newMusicStream(func() (*musicTrack, error) { return openSong(playlist.Next()) }, musicFadeFrames)
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
	player.SetVolume(0.35)
	player.Play()
	return &musicPlayback{context: context, player: player}, nil
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
