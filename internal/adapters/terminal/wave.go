package terminal

import "math"

// wave approximates math.Sin to within 0.002 at a fraction of its cost, for the textures and
// shimmer computed for every pixel.
func wave(x float64) float64 {
	// The phase within one turn, from -0.5 to 0.5.
	t := x * (0.5 / math.Pi)
	t -= math.Floor(t + 0.5)
	// A parabola through the sine's zeros and peaks, then sharpened toward the true curve.
	y := 8*t - 16*t*math.Abs(t)
	return y + 0.225*(y*math.Abs(y)-y)
}
