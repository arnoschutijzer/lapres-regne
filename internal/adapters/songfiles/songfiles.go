package songfiles

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/flac"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/vorbis"
	"github.com/gopxl/beep/wav"

	"dither-cli/internal/soundtrack"
)

// List returns the songs directly inside the folder. A missing folder has no songs.
func List(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var songs []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".ogg", ".mp3", ".wav", ".flac":
			songs = append(songs, filepath.Join(directory, entry.Name()))
		}
	}
	return songs, nil
}

// Decoder opens MP3, OGG/Vorbis, WAV and FLAC files by path.
type Decoder struct{}

func (Decoder) Open(path string) (*soundtrack.Track, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var decoder beep.StreamSeekCloser
	var format beep.Format
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ogg":
		decoder, format, err = vorbis.Decode(file)
	case ".mp3":
		decoder, format, err = mp3.Decode(file)
	case ".wav":
		decoder, format, err = wav.Decode(file)
	case ".flac":
		decoder, format, err = flac.Decode(file)
	default:
		err = fmt.Errorf("unsupported audio format")
	}
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	var stream beep.Streamer = decoder
	if format.SampleRate != soundtrack.SampleRate {
		stream = beep.Resample(4, format.SampleRate, soundtrack.SampleRate, stream)
	}
	frames := int64(decoder.Len()) * soundtrack.SampleRate / int64(format.SampleRate)
	return &soundtrack.Track{Samples: &decodedSamples{stream: stream}, Frames: frames, Closer: decoder}, nil
}

// Beep decodes and resamples to stereo; the audio device consumes float32 PCM.
type decodedSamples struct {
	stream beep.Streamer
	buffer [1024][2]float64
}

func (s *decodedSamples) Read(p []float32) (int, error) {
	frames := min(len(p)/soundtrack.Channels, len(s.buffer))
	n, ok := s.stream.Stream(s.buffer[:frames])
	for i, sample := range s.buffer[:n] {
		p[i*2], p[i*2+1] = float32(sample[0]), float32(sample[1])
	}
	if !ok {
		if err := s.stream.Err(); err != nil {
			return n * soundtrack.Channels, err
		}
		return n * soundtrack.Channels, io.EOF
	}
	return n * soundtrack.Channels, nil
}
