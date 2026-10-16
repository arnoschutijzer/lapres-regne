package speaker

import (
	"fmt"
	"time"

	"github.com/ebitengine/oto/v3"

	"lapresregne/internal/soundtrack"
)

// Device is the default audio output, shared by the soundtrack and sound effects.
type Device struct {
	context *oto.Context
}

func Open() (*Device, error) {
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: soundtrack.SampleRate, ChannelCount: soundtrack.Channels,
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
	return &Device{context: context}, nil
}

func (d *Device) Effects() *Effects {
	return newEffects(d.context)
}

func (d *Device) Close() {
	d.context.Suspend()
}

// Playback plays a soundtrack stream on the device.
type Playback struct {
	context *oto.Context
	player  *oto.Player
	stream  *soundtrack.Stream
}

// Play takes ownership of the stream and closes it with the playback.
func (d *Device) Play(stream *soundtrack.Stream) *Playback {
	player := d.context.NewPlayer(stream)
	// Avoid half a second of decoded audio delaying navigation requests.
	player.SetBufferSize(soundtrack.SampleRate * soundtrack.Channels * 4 / 10)
	player.SetVolume(0.35)
	player.Play()
	return &Playback{context: d.context, player: player, stream: stream}
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
}
