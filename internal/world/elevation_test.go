package world_test

import (
	"math"
	"testing"
	"time"

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
			// The banks of a river's gully fall steeply to the water.
			if distance, width := w.River(x, z); distance < width+world.BankWidth+step {
				continue
			}
			rise := math.Hypot(w.Elevation(x+step, z)-w.Elevation(x, z), w.Elevation(x, z+step)-w.Elevation(x, z))
			if rise/step > 0.5 {
				t.Fatalf("slope at x %v z %v = %v, want gentle slopes no steeper than 0.5", x, z, rise/step)
			}
		}
	}
}

func TestRiversRunInGulliesAlongTheValleyFloors(t *testing.T) {
	w := world.New(7)
	found := 0
	for z := -100.0; z <= 100; z += 0.5 {
		for x := -100.0; x <= 100; x += 0.5 {
			distance, width := w.River(x, z)
			height := w.Elevation(x, z)
			switch {
			case distance < width:
				found++
				if height != 0 {
					t.Fatalf("elevation of the river at x %v z %v = %v, want the gully's floor", x, z, height)
				}
			case distance >= width+world.BankWidth && distance < width+world.BankWidth+1:
				if height < world.GullyDepth {
					t.Fatalf("elevation beside the gully at x %v z %v = %v, want at least the valley floor %v above the water", x, z, height, world.GullyDepth)
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

// gully is a map of grass on level 3 with a river on level 0 down column 2, crossed by a bridge on
// level 3 along row 1.
func gully() world.Map {
	g, w, b := world.Grass, world.Water, world.Bridge
	return world.Map{
		Name: "gully",
		Ground: [][]world.Ground{
			{g, g, w, g, g},
			{g, g, b, g, g},
			{g, g, w, g, g},
			{g, g, w, g, g},
		},
		Heights: [][]int{
			{3, 3, 0, 3, 3},
			{3, 3, 3, 3, 3},
			{3, 3, 0, 3, 3},
			{3, 3, 0, 3, 3},
		},
		StartCol: 0, StartRow: 3,
	}
}

func TestAMapsRiverLiesFlatInItsGully(t *testing.T) {
	w := world.NewOnMap(7, gully())
	bank := 3 * world.LevelHeight
	// Along the last row, away from the bridge, the banks fall from their own level to meet the
	// water at its edge.
	for z := 3 * world.TileSize; z < 4*world.TileSize; z += 0.25 {
		for x := 2 * world.TileSize; x < 3*world.TileSize; x += 0.25 {
			if height := w.Elevation(x, z); height != 0 {
				t.Fatalf("elevation of the water at x %v z %v = %v, want it flat at level 0", x, z, height)
			}
		}
		if height := w.Elevation(2*world.TileSize-0.001, z); height > 0.01 {
			t.Fatalf("elevation of the bank at the water's edge at z %v = %v, want it down at the water", z, height)
		}
		if height := w.Elevation(0.5*world.TileSize, z); height != bank {
			t.Fatalf("elevation of the field beyond the bank at z %v = %v, want %v", z, height, bank)
		}
	}
}

func TestABridgesDeckLiesFlatAboveTheWater(t *testing.T) {
	w := world.NewOnMap(7, gully())
	deck := 3 * world.LevelHeight
	for z := world.TileSize; z < 2*world.TileSize; z += 0.25 {
		for x := 2 * world.TileSize; x < 3*world.TileSize; x += 0.25 {
			if height := w.Elevation(x, z); height != deck {
				t.Fatalf("elevation of the deck at x %v z %v = %v, want it flat at %v", x, z, height, deck)
			}
		}
		// The deck meets the banks at either end on their level.
		if height := w.Elevation(2*world.TileSize-0.001, z); math.Abs(height-deck) > 0.01 {
			t.Fatalf("elevation of the bank at the deck's west end at z %v = %v, want the deck's %v", z, height, deck)
		}
	}
}

// flight is a map rising from level 0 in the west to level 4 in the east up a flight of steps on
// columns 3 to 5 of row 1, with the hero at its foot.
func flight() world.Map {
	ground := [][]world.Ground{make([]world.Ground, 9), make([]world.Ground, 9), make([]world.Ground, 9)}
	heights := [][]int{{0, 0, 0, 1, 2, 3, 4, 4, 4}, {0, 0, 0, 1, 2, 3, 4, 4, 4}, {0, 0, 0, 1, 2, 3, 4, 4, 4}}
	for col := 3; col <= 5; col++ {
		ground[1][col] = world.Steps
	}
	return world.Map{Name: "flight", Ground: ground, Heights: heights, StartCol: 1, StartRow: 1}
}

func TestStepsRiseInTreadsAndRisers(t *testing.T) {
	w := world.NewOnMap(7, flight())
	z := 1.5 * world.TileSize
	treads := map[float64]bool{}
	previous := 0.0
	for x := 3.1 * world.TileSize; x < 5.9*world.TileSize; x += 0.05 {
		height := w.Elevation(x, z)
		if height < previous {
			t.Fatalf("elevation at x %v = %v, below %v nearer the foot; want the steps only rising", x, height, previous)
		}
		previous = height
		treads[height] = true
	}
	if len(treads) < 5 || len(treads) > 20 {
		t.Fatalf("%d heights up the steps, want a flight of level treads", len(treads))
	}
}

func TestTheHeroWalksUpTheSteps(t *testing.T) {
	w := world.NewOnMap(7, flight())
	w.Advance(1, 0, 3*time.Second, false)
	if x, _ := w.Hero().Position(); x < 6*world.TileSize {
		t.Fatalf("x = %v; want the hero up the steps at the top", x)
	}
}
