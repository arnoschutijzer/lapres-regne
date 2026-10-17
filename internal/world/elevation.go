package world

import "math"

// HillHeight is how high the endless forest's hills rise above its valley floors, in world units.
const HillHeight = 6.0

// GullyDepth is how far a river in the endless forest lies below the valley floor, in a gully
// whose banks fall to the water over BankWidth.
const (
	GullyDepth = 2.0
	BankWidth  = 1.5
)

// Elevation is how high the ground lies at a world position, in world units above the lowest
// ground. The endless forest rolls in gentle hills that level out into the river valleys, where
// the rivers run in gullies; a map slopes from tile to tile with its heights.
func (w *World) Elevation(x, z float64) float64 {
	if w.terrain != nil {
		return w.terrain.elevationAt(x, z)
	}
	distance, width := w.River(x, z)
	return w.hills(x, z, distance, width)
}

// ElevationAmid is the endless forest's elevation at a world position, among the river channels
// at its depth as RiverChannels finds them, so whoever has the channels at hand need not work
// them out again.
func (w *World) ElevationAmid(channels []RiverChannel, x, z float64) float64 {
	distance, width := NearestRiver(channels, x)
	return w.hills(x, z, distance, width)
}

// hills is the endless forest's elevation at a world position a distance from the nearest river,
// of a width.
func (w *World) hills(x, z, distance, width float64) float64 {
	// Broad hills carry smaller rises on their flanks.
	hills := 0.7*w.Noise(x/48, z/48, 150) + 0.3*w.Noise(x/18, z/18, 151)
	// The rivers run in gullies along the valley floors, and the hills rise away from them.
	gully := smoothstep(width, width+BankWidth, distance)
	valley := smoothstep(width+BankWidth, width+26, distance)
	return GullyDepth*gully + HillHeight*smoothstep(0.2, 0.8, hills)*valley
}

// Highest is the highest the ground can lie anywhere in the world.
func (w *World) Highest() float64 {
	if w.terrain != nil {
		return float64(w.terrain.highest) * LevelHeight
	}
	return GullyDepth + HillHeight
}

// smoothstep eases from 0 at from to 1 at to.
func smoothstep(from, to, value float64) float64 {
	u := math.Min(1, math.Max(0, (value-from)/(to-from)))
	return u * u * (3 - 2*u)
}
