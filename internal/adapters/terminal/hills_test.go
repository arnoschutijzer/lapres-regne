package terminal

import (
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestTheGroundShownAtEachPixelLiesUnderIt(t *testing.T) {
	w := world.New(7)
	r := newRenderer(w, "L’Après-Règne")
	c := r.render(100, 27, 0)
	lowest, highest := math.Inf(1), math.Inf(-1)
	for y := range c.height {
		for x := range c.width {
			groundX, groundZ := c.ground(float64(x), float64(y))
			height := w.Elevation(groundX, groundZ)
			lowest, highest = math.Min(lowest, height), math.Max(highest, height)
			c.lift = height
			px, py, _ := c.project(groundX, 0, groundZ)
			if math.Abs(px-float64(x)) > 0.5 || math.Abs(py-float64(y)) > 1 {
				t.Fatalf("pixel %d, %d shows the ground at x %.2f z %.2f, which projects onto %.1f, %.1f", x, y, groundX, groundZ, px, py)
			}
		}
	}
	if highest-lowest < 1 {
		t.Fatalf("the view's ground spans %.2f to %.2f, want hills in view", lowest, highest)
	}
}

func TestNearerHillsHideTheGroundBehindThem(t *testing.T) {
	r := newRenderer(world.New(7), "L’Après-Règne")
	c := r.render(100, 27, 0)
	// A ridge 6 high runs across the view 2 to 4 north of the camera, on otherwise flat ground.
	c.elevation = func(_, z float64) float64 {
		if z > c.z-4 && z < c.z-2 {
			return 6
		}
		return 0
	}
	c.highest, c.eye = 6, 0
	column := c.width / 2
	r.groundColumn(c, column, 0)
	ridge := 0
	for y := range c.height {
		_, z := c.ground(float64(column), float64(y))
		behind := z - c.z
		if behind > -4 && behind < -2 {
			ridge++
		}
		// Ground up to 6 * 0.69 / 0.72 behind the ridge lies hidden from the viewer.
		if behind <= -4 && behind > -4-6*0.69/0.72 {
			t.Fatalf("pixel row %d shows ground %.2f north of the camera, hidden behind the ridge", y, -behind)
		}
	}
	if ridge == 0 {
		t.Fatal("the ridge itself must show")
	}
}

func TestSlopesFacingTheSunAreLit(t *testing.T) {
	c := &forestCanvas{}
	// The sun shines from the north-west, onto ground rising toward the south-east.
	c.elevation = func(x, z float64) float64 { return 0.4 * (x + z) }
	sunward := c.slopeLight(0, 0, 0)
	c.elevation = func(x, z float64) float64 { return -0.4 * (x + z) }
	shaded := c.slopeLight(0, 0, 0)
	c.elevation = func(float64, float64) float64 { return 2 }
	flat := c.slopeLight(0, 0, 2)
	if !(sunward > flat && flat > shaded) || flat != 1 {
		t.Fatalf("light facing the sun %.2f, flat %.2f, facing away %.2f; want brightest facing the sun and flat ground unchanged", sunward, flat, shaded)
	}
}

func TestATreeOnAHillStandsOnItsTop(t *testing.T) {
	_, _, valley := leafBounds(world.Oak)
	r := newRenderer(world.New(7), "L’Après-Règne")
	c := &forestCanvas{width: 120, height: 120, x: 1000, scale: 3.6, elevation: func(float64, float64) float64 { return 3 }}
	c.turnTo(0)
	c.pixels = make([]forestPixel, c.width*c.height)
	for i := range c.pixels {
		c.pixels[i].depth = math.Inf(1)
	}
	c.standOn(1000, 0)
	r.drawTree(c, world.Tree{X: 1000, Height: 6, Radius: 2.4, Trunk: 0.25, Kind: world.Oak}, 0)
	lowest := 0
	for y := range c.height {
		for x := range c.width {
			if ink := c.pixels[y*c.width+x].ink; ink == leavesOf(world.Oak).shaded || ink == leavesOf(world.Oak).sunlit {
				lowest = max(lowest, y)
			}
		}
	}
	if want := float64(valley) - 3*0.69*3.6; math.Abs(float64(lowest)-want) > 1 {
		t.Fatalf("lowest leaves on the hill at row %d, in the valley at %d; want them raised to row %.0f", lowest, valley, want)
	}
}

func TestTheViewRisesAndFallsWithTheHero(t *testing.T) {
	w := world.New(7)
	// Walk up the hill west of the start, then follow the hero there.
	w.Advance(-1, 0, 2*time.Second, false)
	w.Follow(time.Second)
	x, z := w.Hero().Position()
	if w.Elevation(x, z) < 2 {
		t.Fatalf("the hero stands %.2f high at x %.1f z %.1f, want them up a hill", w.Elevation(x, z), x, z)
	}
	c := newRenderer(w, "L’Après-Règne").render(100, 27, 0)
	c.standOn(x, z)
	_, feet, _ := c.project(x, 0, z)
	if want := float64(c.height) * 0.57; math.Abs(feet-want) > 1 {
		t.Fatalf("the hero's feet are on canvas row %.1f, want them on row %.1f where the camera looks", feet, want)
	}
}
