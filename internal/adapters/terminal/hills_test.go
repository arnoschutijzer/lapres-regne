package terminal

import (
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestTheGroundShownAtEachPixelLiesUnderIt(t *testing.T) {
	w := world.New(7)
	r := newRenderer(w, NewCamera(0, 0), "L’Après-Règne")
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
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
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
		// Ground up to 6 / tan(tilt) behind the ridge lies hidden from the viewer, less the depth of
		// the row where the ridge's top edge falls.
		hidden := 6*c.tiltCos/c.tiltSin - 1/(c.tiltSin*c.scale)
		if behind <= -4 && behind > -4-hidden {
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
	r := newRenderer(world.New(7), NewCamera(0, 0), "L’Après-Règne")
	c := &forestCanvas{width: 120, height: 120, x: 1000, scale: 3.6, elevation: func(float64, float64) float64 { return 3 }}
	c.turnTo(0)
	c.tiltTo(openingTilt)
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
	x, z := w.Hero().Position()
	if w.Elevation(x, z) < 2 {
		t.Fatalf("the hero stands %.2f high at x %.1f z %.1f, want them up a hill", w.Elevation(x, z), x, z)
	}
	c := newRenderer(w, NewCamera(0, 0), "L’Après-Règne").render(100, 27, 0)
	c.standOn(x, z)
	_, feet, _ := c.project(x, 0, z)
	if want := float64(c.height) * 0.57; math.Abs(feet-want) > 1 {
		t.Fatalf("the hero's feet are on canvas row %.1f, want them on row %.1f where the camera looks", feet, want)
	}
}

// hillsideCottage is cottage's walled room of flagstones up on a rise of level 4, north of the
// hero on level 0.
func hillsideCottage() *world.World {
	m := world.Map{Name: "Hillside", Ground: make([][]world.Ground, 60), StartCol: 40, StartRow: 30}
	for row := range m.Ground {
		m.Ground[row] = make([]world.Ground, 80)
		m.Heights = append(m.Heights, make([]int, 80))
		for col := 32; col <= 48 && row >= 22 && row <= 27; col++ {
			m.Ground[row][col] = world.Floor
			if row == 22 || row == 27 || col == 32 || col == 48 {
				m.Ground[row][col] = world.Wall
			}
		}
		for col := range m.Heights[row] {
			if row <= 28 {
				m.Heights[row][col] = 4
			}
		}
	}
	m.Ground[27][40] = world.Door
	return world.NewOnMap(7, m)
}

func TestAWallOnARiseStandsOnTopOfIt(t *testing.T) {
	r := newRenderer(hillsideCottage(), NewCamera(0, 0), "L’Après-Règne")
	r.zoom = zoom{current: 3}
	c := r.render(160, 60, 0)
	// The south wall's face rises from its foot at z 55.35, on ground 4 levels up.
	c.lift = 4 * world.LevelHeight
	x, foot, _ := c.project(76, 0, 55.35)
	_, top, _ := c.project(76, wallHeight*0.9, 55.35)
	stone := 0
	for y := int(top); y < int(foot); y++ {
		if ink := c.pixels[y*c.width+int(x)].ink; ink == stoneInk || ink == shadedStoneInk {
			stone++
		}
	}
	if stone < int(foot-top)/2 {
		t.Fatalf("stone pixels = %d of %d up the wall's face, want it standing on the rise", stone, int(foot-top))
	}
}

func TestARoofOnARiseCrownsItsWalls(t *testing.T) {
	r := newRenderer(hillsideCottage(), NewCamera(0, 0), "L’Après-Règne")
	r.zoom = zoom{current: 3}
	c := r.render(160, 60, 0)
	highest := c.height
	for i, p := range c.pixels {
		if p.ink == slateInk || p.ink == shadedSlateInk {
			highest = min(highest, i/c.width)
		}
	}
	// The roof shows highest along its ridge or its far eaves, whichever the view lifts higher.
	shape := roofOver(r.world.Buildings()[0])
	c.lift = 4 * world.LevelHeight
	_, ridge, _ := c.project(80, shape.ridge(), (shape.z0+shape.z1)/2)
	_, farEaves, _ := c.project(80, eaves, shape.z0)
	if top := math.Min(ridge, farEaves); math.Abs(float64(highest)-top) > 2 {
		t.Fatalf("the roof's top is on canvas row %d, want it on the rise at row %.0f", highest, top)
	}
}
