package world_test

import (
	"math"
	"testing"

	"lapresregne/internal/world"
)

func TestTheEndlessForestRisesAndFallsInRollingHills(t *testing.T) {
	w := world.New(7)
	lowest, highest := math.Inf(1), math.Inf(-1)
	for z := -200.0; z <= 200; z += 2 {
		for x := -200.0; x <= 200; x += 2 {
			height := w.Elevation(x, z)
			if height < 0 || height > w.Highest() {
				t.Fatalf("elevation at x %v z %v = %v, want it between 0 and %v", x, z, height, w.Highest())
			}
			lowest, highest = math.Min(lowest, height), math.Max(highest, height)
		}
	}
	if highest-lowest < w.Highest()/2 {
		t.Fatalf("elevations span %v to %v, want hills rising over half of %v", lowest, highest, w.Highest())
	}
}

func TestTheHillsSlopeGently(t *testing.T) {
	w := world.New(7)
	const step = 0.25
	for z := -100.0; z <= 100; z += 1.3 {
		for x := -100.0; x <= 100; x += 1.3 {
			rise := math.Hypot(w.Elevation(x+step, z)-w.Elevation(x, z), w.Elevation(x, z+step)-w.Elevation(x, z))
			if rise/step > 0.5 {
				t.Fatalf("slope at x %v z %v = %v, want gentle slopes no steeper than 0.5", x, z, rise/step)
			}
		}
	}
}

func TestRiversRunAlongTheValleyFloors(t *testing.T) {
	w := world.New(7)
	found := 0
	for z := -100.0; z <= 100; z += 0.5 {
		for x := -100.0; x <= 100; x += 0.5 {
			if distance, width := w.River(x, z); distance < width+1 {
				found++
				if height := w.Elevation(x, z); height != 0 {
					t.Fatalf("elevation of the river or its bank at x %v z %v = %v, want the valley floor", x, z, height)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("no river found to test")
	}
}

func TestAMapWithoutHeightsLiesFlat(t *testing.T) {
	w := world.NewOnMap(7, pond())
	for z := 0.0; z < 4*world.TileSize; z += 0.5 {
		for x := 0.0; x < 6*world.TileSize; x += 0.5 {
			if height := w.Elevation(x, z); height != 0 {
				t.Fatalf("elevation at x %v z %v = %v, want a flat map", x, z, height)
			}
		}
	}
}

func TestElevationAmidSampledRiversMatchesElevation(t *testing.T) {
	w := world.New(7)
	for z := -60.0; z <= 60; z += 1.7 {
		channels := w.RiverChannels(nil, z, -60, 60)
		for x := -60.0; x <= 60; x += 1.3 {
			if amid, height := w.ElevationAmid(channels, x, z), w.Elevation(x, z); amid != height {
				t.Fatalf("elevation amid the rivers at x %v z %v = %v, want %v", x, z, amid, height)
			}
		}
	}
}

func TestAMapRisesAndFallsWithItsHeights(t *testing.T) {
	m := pond()
	m.Heights = [][]int{
		{0, 0, 2, 2},
		{0, 0, 2, 2},
	}
	w := world.NewOnMap(7, m)
	centre := func(col, row int) (float64, float64) {
		return (float64(col) + 0.5) * world.TileSize, (float64(row) + 0.5) * world.TileSize
	}
	if height := w.Elevation(centre(2, 0)); height != 2*world.LevelHeight {
		t.Fatalf("elevation at the centre of a level 2 tile = %v, want %v", height, 2*world.LevelHeight)
	}
	if height := w.Elevation(centre(1, 1)); height != 0 {
		t.Fatalf("elevation at the centre of a level 0 tile = %v, want 0", height)
	}
	if height := w.Elevation(2*world.TileSize, 0.5*world.TileSize); height != world.LevelHeight {
		t.Fatalf("elevation between a level 0 and a level 2 tile = %v, want it halfway at %v", height, world.LevelHeight)
	}
	// Tiles the heights leave out lie at level 0.
	if height := w.Elevation(centre(1, 3)); height != 0 {
		t.Fatalf("elevation of a tile without a height = %v, want 0", height)
	}
	if w.Highest() != 2*world.LevelHeight {
		t.Fatalf("highest = %v, want the highest tile's %v", w.Highest(), 2*world.LevelHeight)
	}
}
