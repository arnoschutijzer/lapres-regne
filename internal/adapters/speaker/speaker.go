package speaker

import (
	"fmt"
	"time"

	"github.com/ebitengine/oto/v3"

	"dither-cli/internal/soundtrack"
)

// Playback plays a soundtrack stream on the default audio device.
type Playback struct {
	context *oto.Context
	player  *oto.Player
	stream  *soundtrack.Stream
}

// Start takes ownership of the stream and closes it when starting fails.
func Start(stream *soundtrack.Stream) (*Playback, error) {
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: soundtrack.SampleRate, ChannelCount: soundtrack.Channels,
		Format: oto.FormatFloat32LE, BufferSize: 100 * time.Millisecond,
	})
	if err != nil {
		stream.Close()
		return nil, err
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		stream.Close()
		return nil, fmt.Errorf("audio device initialization timed out")
	}
	if err := context.Err(); err != nil {
		stream.Close()
		return nil, err
	}
	player := context.NewPlayer(stream)
	// Avoid half a second of decoded audio delaying navigation requests.
	player.SetBufferSize(soundtrack.SampleRate * soundtrack.Channels * 4 / 10)
	player.SetVolume(0.35)
	player.Play()
	return &Playback{context: context, player: player, stream: stream}, nil
}

func (m *Playback) Skip(direction int) {
	m.stream.Skip(direction)
}

func (m *Playback) Err() error {
	if err := m.context.Err(); err != nil {
		return err
	}
	return m.player.Err()
}

func (m *Playback) Close() {
	m.player.PauseAndStopReading()
	m.stream.Close()
	m.context.Suspend()
}
