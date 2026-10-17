package window

// hash is a stable pseudo-random value from 0 to 1 for a pixel of a sprite.
func hash(x, y int, salt uint64) float64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ salt*0x165667B19E3779F9
	h ^= h >> 29
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 32
	return float64(h&0xFFFFFF) / 0xFFFFFF
}
