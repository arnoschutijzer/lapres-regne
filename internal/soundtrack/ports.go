package soundtrack

// Samples yields interleaved stereo float32 samples at SampleRate.
type Samples interface {
	Read(p []float32) (int, error)
}

// Decoder opens a song by its ID for streaming.
type Decoder interface {
	Open(song string) (*Track, error)
}

// Player is the running soundtrack as the game controls it.
type Player interface {
	Skip(direction int)
	Err() error
}
