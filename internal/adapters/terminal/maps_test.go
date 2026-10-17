package terminal

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// lakeside is a meadow map with the hero on its west edge, a lake to their east and an oak to
// their north, so the view shows the map's edge, water and a tree.
func lakeside() *world.World {
	ground := make([][]world.Ground, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
		for col := 8; col < 14; col++ {
			ground[row][col] = world.Water
		}
	}
	return world.NewOnMap(7, world.Map{
		Name:     "Lakeside",
		Ground:   ground,
		Trees:    []world.PlantedTree{{Col: 2, Row: 12, Kind: world.Oak}},
		StartCol: 2, StartRow: 15,
	})
}

func TestAMapDrawsItsWaterAndTreesAndNothingBeyondItsEdge(t *testing.T) {
	r := newRenderer(lakeside(), "L’Après-Règne")
	c := r.render(100, 27, 0)
	if countInk(c, 5, 6) == 0 {
		t.Fatal("the lake must be drawn as water")
	}
	if countInk(c, 15, 16) == 0 {
		t.Fatal("the planted oak must be drawn with its leaves")
	}
	// Left of the hero lies the map's west edge, four world units away.
	_, z := r.world.Hero().Position()
	_, y, _ := c.project(-3, 0, z)
	for x := range 10 {
		if p := c.pixels[int(y)*c.width+x]; p.light != 0 {
			t.Fatalf("pixel %d, %d beyond the edge has light %v, want nothing drawn", x, int(y), p.light)
		}
	}
}

func TestTheStatusLineNamesTheMap(t *testing.T) {
	r := newRenderer(lakeside(), "L’Après-Règne")
	if frame := r.frame(120, 30, 0); !strings.Contains(frame, "Lakeside") || strings.Contains(frame, "seed") {
		t.Fatal("the status line must name the map instead of the seed")
	}
}

// gatehouse stands the hero on a path south of a stone wall with a floor behind it, beside a
// fence, a gate and a grave.
func gatehouse() *world.World {
	ground := make([][]world.Ground, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
	}
	for col := 10; col < 30; col++ {
		ground[10][col] = world.Wall
		for row := 4; row < 10; row++ {
			ground[row][col] = world.Floor
		}
		ground[16][col] = world.Path
	}
	ground[14][16], ground[14][17], ground[14][18] = world.Fence, world.Gate, world.Fence
	ground[14][22] = world.Grave
	ground[10][20] = world.Door
	return world.NewOnMap(7, world.Map{Name: "Gatehouse", Ground: ground, StartCol: 20, StartRow: 16})
}

func TestAMapDrawsItsBuiltGroundAndStructures(t *testing.T) {
	r := newRenderer(gatehouse(), "L’Après-Règne")
	c := r.render(100, 27, 0)
	for _, ink := range []struct {
		name string
		ink  int
	}{{"stone", stoneInk}, {"path", pathInk}, {"floor", floorInk}, {"iron", ironInk}, {"planks", plankInk}} {
		if countInk(c, ink.ink) == 0 {
			t.Errorf("no %s pixels drawn", ink.name)
		}
	}
}

func TestAWallStandsAboveTheGroundItIsBuiltOn(t *testing.T) {
	r := newRenderer(gatehouse(), "L’Après-Règne")
	c := r.render(100, 27, 0)
	// The wall's south face rises from its foot at z 21.35 up to its top.
	x, foot, _ := c.project(45, 0, 21.35)
	_, top, _ := c.project(45, wallHeight*0.9, 21.35)
	stone := 0
	for y := int(top); y < int(foot); y++ {
		if ink := c.pixels[y*c.width+int(x)].ink; ink == stoneInk || ink == shadedStoneInk {
			stone++
		}
	}
	if stone < int(foot-top)/2 {
		t.Fatalf("stone pixels = %d of %d up the wall's face, want it standing", stone, int(foot-top))
	}
}

func TestStructureInksNeverOutrankTheHeroInACell(t *testing.T) {
	if foremost(stoneInk) || foremost(pathInk) || foremost(floorInk) || foremost(ironInk) {
		t.Fatal("structures and built ground must not outrank what stands in front of them")
	}
}

func TestTheStatusLineNamesTheAreaTheHeroStandsIn(t *testing.T) {
	ground := [][]world.Ground{{world.Grass, world.Grass}}
	w := world.NewOnMap(7, world.Map{Name: "Lumbridge", Ground: ground, Areas: [][]string{{"Lumbridge Castle"}}})
	if frame := newRenderer(w, "L’Après-Règne").frame(120, 30, 0); !strings.Contains(frame, "Lumbridge Castle") {
		t.Fatal("the status line must name the area the hero stands in")
	}
}

func TestADoorwayOpensUnderAStoneLintelWithItsDoorSwungBack(t *testing.T) {
	r := newRenderer(gatehouse(), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 90, 0)
	// The doorway's tile spans x 40 to 42 along the wall's south face at z 21.35.
	x, foot, _ := c.project(41, 0, 21.35)
	_, lintel, _ := c.project(41, (doorHeight+wallHeight)/2, 21.35)
	_, opening, _ := c.project(41, doorHeight/2, 21.35)
	if p := c.pixels[int(lintel)*c.width+int(x)]; p.ink != stoneInk && p.ink != shadedStoneInk {
		t.Fatalf("ink above the doorway = %d, want the stone lintel", p.ink)
	}
	if p := c.pixels[int(opening)*c.width+int(x)]; p.ink == stoneInk || p.ink == shadedStoneInk {
		t.Fatal("the doorway must open through the wall")
	}
	if foot <= opening {
		t.Fatal("the doorway's foot must lie below its middle")
	}
	if countInk(c, plankInk) == 0 {
		t.Fatal("the open door must be drawn in planks")
	}
}

func TestFencesKeepTheirColorWhenZoomedOut(t *testing.T) {
	r := newRenderer(gatehouse(), "L’Après-Règne")
	r.frame(100, 30, 0)
	fence := 0
	for _, cell := range r.cells {
		if cell.ink == plankInk {
			fence++
		}
	}
	if fence < 2 {
		t.Fatalf("cells inked as fence = %d, want the fence to show at the forest view", fence)
	}
}

// meadow is an open map of grass with the hero in its middle and nothing else on it.
func meadow() *world.World {
	ground := make([][]world.Ground, 60)
	for row := range ground {
		ground[row] = make([]world.Ground, 80)
	}
	return world.NewOnMap(7, world.Map{Name: "Meadow", Ground: ground, StartCol: 40, StartRow: 30})
}

// groundLight is the mean light of the canvas's plain grass where nothing stands on it: pixels
// still at the ground's own depth.
func groundLight(c *forestCanvas) float64 {
	total, count := 0.0, 0
	for y := range c.height {
		ground := -(float64(y) - float64(c.height)*0.57) / (0.72 * c.scale) * 0.69
		for x := range c.width {
			if p := c.pixels[y*c.width+x]; (p.ink == 0 || p.ink == 1) && p.depth == ground {
				total += p.light
				count++
			}
		}
	}
	return total / float64(count)
}

func TestTheGroundQuietsWhenZoomedOut(t *testing.T) {
	r := newRenderer(meadow(), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	near := groundLight(r.render(100, 27, 0))
	r.zoom = zoom{current: 0}
	far := groundLight(r.render(100, 27, 0))
	if far > near*0.75 {
		t.Fatalf("ground light = %.3f zoomed out, %.3f zoomed in; want the far ground clearly quieter", far, near)
	}
}

func TestADarkHaloSetsTheHeroApartFromTheGround(t *testing.T) {
	r := newRenderer(meadow(), "L’Après-Règne")
	c := r.render(100, 27, 0)
	hero := 0
	for y := 1; y < c.height-1; y++ {
		for x := 1; x < c.width-1; x++ {
			if p := c.pixels[y*c.width+x]; !foremost(p.ink) || p.light == 0 {
				continue
			}
			hero++
			for _, step := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				beside := c.pixels[(y+step[1])*c.width+x+step[0]]
				if !foremost(beside.ink) && beside.light != 0 && beside.tree == 0 {
					t.Fatalf("ground beside the hero at %d, %d is lit %.2f, want a dark halo", x+step[0], y+step[1], beside.light)
				}
			}
		}
	}
	if hero == 0 {
		t.Fatal("the hero must be drawn")
	}
}

// walledMeadow is meadow's open grass with an east to west wall through row 10 and a north to
// south wall down column 25.
func walledMeadow() *world.World {
	ground := make([][]world.Ground, 60)
	for row := range ground {
		ground[row] = make([]world.Ground, 80)
	}
	for col := 30; col < 50; col++ {
		ground[25][col] = world.Wall
	}
	for row := 27; row < 36; row++ {
		ground[row][45] = world.Wall
	}
	return world.NewOnMap(7, world.Map{Name: "Walled meadow", Ground: ground, StartCol: 40, StartRow: 30})
}

// lightAt is the light of the canvas pixel showing the ground at x, z, rendered up close.
func lightAt(w *world.World, x, z float64) float64 {
	r := newRenderer(w, "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 60, 0)
	px, py, _ := c.project(x, 0, z)
	return c.pixels[int(py)*c.width+int(px)].light
}

func TestWallsCastShadowsAwayFromTheSun(t *testing.T) {
	// The wall along row 25 stands from z 50.65 to 51.35; the sun in the north-west throws its
	// shadow a little to the south.
	open, shaded := lightAt(meadow(), 80.5, 51.8), lightAt(walledMeadow(), 80.5, 51.8)
	if shaded > open*0.6 {
		t.Fatalf("light south of the wall = %.2f, open ground = %.2f; want the wall's shadow", shaded, open)
	}
}

func TestTheGroundDarkensAtTheFootOfAWall(t *testing.T) {
	// The wall down column 45 stands from x 90.65 to 91.35; its west side faces the sun.
	open, foot := lightAt(meadow(), 90.45, 63), lightAt(walledMeadow(), 90.45, 63)
	if foot > open*0.9 || foot < open*0.5 {
		t.Fatalf("light at the wall's sunlit foot = %.2f, open ground = %.2f; want it a little darker", foot, open)
	}
}

// inkAt is the ink of the canvas pixel showing the point x, y, z, rendered up close.
func inkAt(w *world.World, x, y, z float64) int {
	r := newRenderer(w, "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 60, 0)
	px, py, _ := c.project(x, y, z)
	return c.pixels[int(py)*c.width+int(px)].ink
}

func TestShadowsDeepenTheGroundsColor(t *testing.T) {
	if ink := inkAt(walledMeadow(), 80.5, 0, 51.8); ink != shadowGrassInk {
		t.Fatalf("ink in the wall's shadow = %d, want the shaded grass", ink)
	}
}

func TestAWallsFacesTurnedFromTheSunAreDarkerStone(t *testing.T) {
	// The wall along row 25: its south face looks away from the sun, its top up at it.
	if ink := inkAt(walledMeadow(), 80.5, 1.5, 51.35); ink != shadedStoneInk {
		t.Fatalf("ink on the wall's south face = %d, want the shaded stone", ink)
	}
	if ink := inkAt(walledMeadow(), 80.5, wallHeight+copingHeight, 51); ink != stoneInk {
		t.Fatalf("ink on the wall's top = %d, want the sunlit stone", ink)
	}
}

// pondside is meadow's open grass with a pond across rows 34 to 39, just south of the hero.
func pondside() *world.World {
	ground := make([][]world.Ground, 60)
	for row := range ground {
		ground[row] = make([]world.Ground, 80)
	}
	for row := 34; row < 40; row++ {
		for col := 20; col < 60; col++ {
			ground[row][col] = world.Water
		}
	}
	return world.NewOnMap(7, world.Map{Name: "Pondside", Ground: ground, StartCol: 40, StartRow: 30})
}

func TestMeadowsBloomWithWildflowersEvenFromAfar(t *testing.T) {
	c := newRenderer(meadow(), "L’Après-Règne").render(160, 60, 0)
	if countInk(c, flowerInks[:]...) == 0 {
		t.Fatal("the far view of open grass must show specks of wildflowers")
	}
}

func TestStonesBushesAndFlowersDotTheGrassUpClose(t *testing.T) {
	r := newRenderer(meadow(), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 60, 0)
	for name, inks := range map[string][]int{"stones": {pebbleInk}, "bushes": {bushInk}, "flowers": flowerInks[:]} {
		if countInk(c, inks...) == 0 {
			t.Errorf("no %s drawn on the grass up close", name)
		}
	}
}

func TestWaterIsEdgedWithMuddyBanksAndReeds(t *testing.T) {
	r := newRenderer(pondside(), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 60, 0)
	if countInk(c, bankInk) == 0 {
		t.Fatal("the pond must have a muddy bank")
	}
	if countInk(c, reedInk) == 0 {
		t.Fatal("reeds must grow along the pond")
	}
}

// cottage is meadow's open grass with a walled room of flagstones north of the hero, its door in
// the middle of its south wall.
func cottage() *world.World {
	return roofedCottage(world.Slate)
}

// roofedCottage is the cottage under a roof of the given roofing.
func roofedCottage(roofing world.Roofing) *world.World {
	ground := make([][]world.Ground, 60)
	roofs := make([][]world.Roofing, 60)
	for row := range ground {
		ground[row] = make([]world.Ground, 80)
		roofs[row] = make([]world.Roofing, 80)
	}
	for row := 22; row <= 27; row++ {
		for col := 32; col <= 48; col++ {
			ground[row][col] = world.Floor
			roofs[row][col] = roofing
			if row == 22 || row == 27 || col == 32 || col == 48 {
				ground[row][col] = world.Wall
			}
		}
	}
	ground[27][40] = world.Door
	return world.NewOnMap(7, world.Map{Name: "Cottage", Ground: ground, Roofs: roofs, StartCol: 40, StartRow: 30})
}

func TestMossCreepsUpTheFootOfWalls(t *testing.T) {
	r := newRenderer(walledMeadow(), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	if countInk(r.render(160, 60, 0), mossInk) == 0 {
		t.Fatal("walls must weather with moss near the ground")
	}
}

func TestOnlyTheWallsOfBuildingsHaveWindows(t *testing.T) {
	render := func(w *world.World) *forestCanvas {
		r := newRenderer(w, "L’Après-Règne")
		r.zoom = zoom{current: 6}
		return r.render(160, 60, 0)
	}
	if countInk(render(cottage()), glassInk) == 0 {
		t.Fatal("the cottage's walls must have windows")
	}
	if countInk(render(walledMeadow()), glassInk) != 0 {
		t.Fatal("walls across open fields must have no windows")
	}
}

// keep is meadow's open grass with a battlemented wall along row 25, and a room floored with boards
// beyond it.
func keep() *world.World {
	ground := make([][]world.Ground, 60)
	for row := range ground {
		ground[row] = make([]world.Ground, 80)
	}
	for col := 30; col < 50; col++ {
		ground[25][col] = world.Battlement
		for row := 20; row < 25; row++ {
			ground[row][col] = world.Boards
		}
	}
	return world.NewOnMap(7, world.Map{Name: "Keep", Ground: ground, StartCol: 40, StartRow: 30})
}

func TestBattlementsRiseInMerlonsAboveTheWall(t *testing.T) {
	r := newRenderer(keep(), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 60, 0)
	// The wall's skyline steps up over each merlon and down between them.
	left, _, _ := c.project(76, 0, 51)
	right, _, _ := c.project(86, 0, 51)
	highest, lowest := c.height, 0
	for px := int(left); px < int(right); px++ {
		for py := range c.height {
			if ink := c.pixels[py*c.width+px].ink; ink == stoneInk || ink == shadedStoneInk {
				highest, lowest = min(highest, py), max(lowest, py)
				break
			}
		}
	}
	if step := int(merlonHeight * 0.69 * c.scale); lowest-highest < step*3/4 {
		t.Fatalf("the wall's top varies by %d pixels, want merlons rising about %d above the gaps", lowest-highest, step)
	}
}

func TestBoardedFloorsAreDrawnInWood(t *testing.T) {
	r := newRenderer(keep(), "L’Après-Règne")
	r.zoom = zoom{current: 4}
	if countInk(r.render(160, 60, 0), boardsInk) == 0 {
		t.Fatal("the boarded floor must be drawn in wood")
	}
}

// roofInks are the inks a roof is drawn in.
var roofInks = []int{thatchInk, shadedThatchInk, slateInk, shadedSlateInk, clayInk, shadedClayInk}

func TestBuildingsAreRoofedFromOutside(t *testing.T) {
	r := newRenderer(cottage(), "L’Après-Règne")
	r.zoom = zoom{current: 3}
	c := r.render(160, 60, 0)
	if countInk(c, slateInk, shadedSlateInk) == 0 {
		t.Fatal("the stone-floored cottage must be roofed in slate")
	}
	// The roof's ridge stands well above the walls, over the middle of the cottage.
	x, y, _ := c.project(80, wallHeight+copingHeight+1.5, 50)
	if ink := c.pixels[int(y)*c.width+int(x)].ink; ink != slateInk && ink != shadedSlateInk {
		t.Fatalf("ink at the roof's ridge = %d, want slate", ink)
	}
}

func TestEachRoofingIsDrawnInItsOwnColour(t *testing.T) {
	for _, roofing := range []struct {
		kind world.Roofing
		inks []int
	}{
		{world.Slate, []int{slateInk, shadedSlateInk}},
		{world.Thatch, []int{thatchInk, shadedThatchInk}},
		{world.ClayTiles, []int{clayInk, shadedClayInk}},
	} {
		r := newRenderer(roofedCottage(roofing.kind), "L’Après-Règne")
		r.zoom = zoom{current: 3}
		c := r.render(160, 60, 0)
		if countInk(c, roofing.inks...) == 0 {
			t.Errorf("roofing %d drew no pixels in inks %v", roofing.kind, roofing.inks)
		}
	}
}

func TestRoofsHideWhileTheHeroIsIndoors(t *testing.T) {
	w := cottage()
	w.WalkTo(80, 50)
	for range 200 {
		w.Advance(0, 0, 50*time.Millisecond, false)
	}
	r := newRenderer(w, "L’Après-Règne")
	r.zoom = zoom{current: 3}
	if !w.Indoors() || countInk(r.render(160, 60, 0), roofInks...) != 0 {
		t.Fatal("no roof must hide the hero inside the cottage")
	}
}

// behindCottage walks the hero to the tile just north of the cottage, behind its back wall and roof
// as seen from the south.
func behindCottage(t *testing.T) *world.World {
	w := cottage()
	w.WalkTo(81, 43)
	for range 400 {
		w.Advance(0, 0, 50*time.Millisecond, false)
	}
	if x, z := w.Hero().Position(); math.Hypot(x-81, z-43) > 0.1 {
		t.Fatalf("hero stands at %.1f, %.1f, want behind the cottage", x, z)
	}
	return w
}

// heroPixels counts the lit pixels of the hero.
func heroPixels(c *forestCanvas) int {
	hero := 0
	for _, p := range c.pixels {
		if foremost(p.ink) && p.light > 0 {
			hero++
		}
	}
	return hero
}

func TestTheHeroShowsThroughABuildingInFrontOfThem(t *testing.T) {
	r := newRenderer(behindCottage(t), "L’Après-Règne")
	r.zoom = zoom{current: 3}
	c := r.render(160, 60, 0)
	if heroPixels(c) == 0 {
		t.Fatal("the hero behind the cottage must show through its wall and roof")
	}
	if countInk(c, roofInks...) == 0 {
		t.Fatal("the cottage must stay roofed while the hero stands outside it")
	}
}

func TestTheBuildingInFrontOfTheHeroEasesOpenAroundThem(t *testing.T) {
	w := behindCottage(t)
	building := append([]int{stoneInk, shadedStoneInk}, roofInks...)
	settled := newRenderer(w, "L’Après-Règne")
	settled.zoom = zoom{current: 3}
	open := countInk(settled.render(160, 60, 2.1), building...)
	// A renderer that last showed the hero in the open has the clearing shut as they step behind.
	easing := newRenderer(w, "L’Après-Règne")
	easing.zoom = zoom{current: 3}
	easing.revealing = true
	easing.lastSeconds = 1
	if opening := countInk(easing.render(160, 60, 1.05), building...); opening <= open {
		t.Fatalf("%d pixels of the cottage a moment after the hero steps behind it, want more than the %d once it has opened", opening, open)
	}
	for frame := range 60 {
		easing.render(160, 60, 1.05+float64(frame+1)/60)
	}
	if opened := countInk(easing.render(160, 60, 2.1), building...); opened != open {
		t.Fatalf("%d pixels of the cottage a second after the hero steps behind it, want the %d once it has opened", opened, open)
	}
}

// besideTableAndWall stands the hero in the open, with a table on the tile in front of them and a
// wall running north and south just past their right side, neither hiding them.
func besideTableAndWall() *world.World {
	const cols, rows = 40, 30
	ground := make([][]world.Ground, rows)
	edges := world.Edges{
		West: make([][]world.Ground, rows), North: make([][]world.Ground, rows),
		Rising: make([][]world.Ground, rows), Falling: make([][]world.Ground, rows),
	}
	for row := range rows {
		ground[row] = make([]world.Ground, cols)
		edges.West[row], edges.North[row] = make([]world.Ground, cols), make([]world.Ground, cols)
		edges.Rising[row], edges.Falling[row] = make([]world.Ground, cols), make([]world.Ground, cols)
	}
	edges.West[17][21], edges.West[18][21] = world.Wall, world.Wall
	return world.NewOnMap(7, world.Map{
		Name: "Yard", Ground: ground, Edges: edges, StartCol: 20, StartRow: 16,
		Fixtures: []world.PlacedFixture{{Col: 20, Row: 17, Kind: world.Table}},
	})
}

func TestTheHeroInTheOpenLeavesTheFurnitureInFrontOfThemDrawn(t *testing.T) {
	_, c := closeView(besideTableAndWall())
	// The hero stands at 41, 33; the table's top lies a tile in front of them, at z 35.
	if found := inkNear(c, 41, tableHeight, 35, plankInk, shadedOf(plankInk)); found < 10 {
		t.Fatalf("%d of 25 pixels of planks on the table in front of the hero, want it drawn", found)
	}
}

func TestTheHeroInTheOpenLeavesTheWallBesideThemStanding(t *testing.T) {
	_, c := closeView(besideTableAndWall())
	// The wall runs along x 42, in front of the hero and just past their right side.
	if found := inkNear(c, 42, 1.5, 35, stoneInks...); found < 10 {
		t.Fatalf("%d of 25 pixels of stone on the wall beside the hero, want it standing", found)
	}
}

// riverCrossing stands the hero on a bridge carrying a path east over a river.
func riverCrossing() *world.World {
	ground := make([][]world.Ground, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
		for col := 16; col < 24; col++ {
			ground[row][col] = world.Water
		}
	}
	for col := 14; col < 26; col++ {
		ground[15][col] = world.Bridge
	}
	return world.NewOnMap(7, world.Map{Name: "Crossing", Ground: ground, StartCol: 20, StartRow: 15})
}

func TestABridgeIsDeckedInBoardsAndRailedOverTheWater(t *testing.T) {
	r := newRenderer(riverCrossing(), "L’Après-Règne")
	c := r.render(100, 27, 0)
	if countInk(c, boardsInk) == 0 {
		t.Error("the bridge's deck must be drawn in boards")
	}
	if countInk(c, plankInk) == 0 {
		t.Error("the bridge must be railed in planks")
	}
}

// hall is a stone-floored hall with the hero in its middle and fixtures of a kind standing on
// the tile two north of them.
func hall(kind world.Fixture) *world.World {
	ground := make([][]world.Ground, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
		for col := range ground[row] {
			ground[row][col] = world.Floor
		}
	}
	return world.NewOnMap(7, world.Map{
		Name: "Hall", Ground: ground, StartCol: 20, StartRow: 15,
		Fixtures: []world.PlacedFixture{{Col: 20, Row: 13, Kind: kind}},
	})
}

// inkAbove is the ink drawn where a point at height y above the middle of the tile two north of
// the hero shows on the canvas.
func inkAbove(c *forestCanvas, y float64) int {
	x, screenY, _ := c.project(20.5*world.TileSize, y, 13.5*world.TileSize)
	return c.pixels[int(screenY)*c.width+int(x)].ink
}

func TestATableStandsOnItsTileInWood(t *testing.T) {
	r := newRenderer(hall(world.Table), "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 90, 0)
	if ink := inkAbove(c, tableHeight); ink != plankInk {
		t.Fatalf("ink on the table's top = %d, want its wooden planks", ink)
	}
}

func TestEachFixtureShowsItsTellingColour(t *testing.T) {
	for _, fixture := range []struct {
		kind world.Fixture
		inks []int
	}{
		{world.Chair, []int{plankInk}},
		{world.Throne, []int{blueClothInk}},
		{world.Armour, []int{ironInk}},
		{world.Range, []int{emberInk, flameInk}},
		{world.Barrel, []int{plankInk}},
		{world.Crate, []int{plankInk}},
		{world.Sacks, []int{burlapInk}},
		{world.Shelves, []int{plankInk}},
		{world.Staircase, []int{stoneInk, shadedStoneInk}},
		{world.Ladder, []int{plankInk}},
		{world.Torch, []int{flameInk}},
		{world.Candles, []int{waxInk}},
		{world.Standard, []int{blueClothInk}},
		{world.Altar, []int{whiteClothInk}},
		{world.Pew, []int{plankInk}},
		{world.Organ, []int{ironInk}},
		{world.Coffin, []int{plankInk}},
		{world.Fountain, []int{waterInk}},
		{world.Statue, []int{stoneInk, shadedStoneInk}},
		{world.Well, []int{stoneInk, shadedStoneInk}},
		{world.Haystack, []int{thatchInk, shadedThatchInk}},
		{world.Signpost, []int{plankInk}},
		{world.Cart, []int{plankInk}},
		{world.Trough, []int{waterInk}},
		{world.Scarecrow, []int{burlapInk, redClothInk}},
		{world.Crop, []int{leafInk, shadedLeafInk}},
		{world.Anvil, []int{ironInk}},
		{world.Counter, []int{plankInk}},
		{world.Bed, []int{redClothInk}},
	} {
		r := newRenderer(hall(fixture.kind), "L’Après-Règne")
		r.zoom = zoom{current: 6}
		c := r.render(160, 90, 0)
		if drawn := countInk(c, fixture.inks...); drawn < 15 {
			t.Errorf("fixture %d drew %d pixels in inks %v, want it standing in them", fixture.kind, drawn, fixture.inks)
		}
	}
}

func TestAPewFacesAlongTheChurchAcrossItsRow(t *testing.T) {
	ground := [][]world.Ground{{world.Wall, world.Floor, world.Floor, world.Floor}}
	w := world.NewOnMap(7, world.Map{Name: "Nave", Ground: ground, Fixtures: []world.PlacedFixture{
		{Col: 1, Row: 0, Kind: world.Pew}, {Col: 2, Row: 0, Kind: world.Pew},
	}})
	r := newRenderer(w, "L’Après-Règne")
	r.render(100, 27, 0)
	if back := backOf(r.built.at(1, 0)); back != 0 {
		t.Fatalf("pew beside the west wall backs onto side %d, want north, facing along the nave", back)
	}
}

func TestAFountainOfFourTilesRisesOnceFromTheirMiddle(t *testing.T) {
	ground := make([][]world.Ground, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
	}
	var fountain []world.PlacedFixture
	for _, tile := range [4][2]int{{20, 12}, {21, 12}, {20, 13}, {21, 13}} {
		fountain = append(fountain, world.PlacedFixture{Col: tile[0], Row: tile[1], Kind: world.Fountain})
	}
	w := world.NewOnMap(7, world.Map{Name: "Courtyard", Ground: ground, StartCol: 20, StartRow: 16, Fixtures: fountain})
	r := newRenderer(w, "L’Après-Règne")
	r.zoom = zoom{current: 6}
	c := r.render(160, 90, 0)
	// The pedestal stands where the four tiles meet, its front showing below the bowl it lifts.
	x, y, _ := c.project(21*world.TileSize, 0.7, 13*world.TileSize+0.3)
	stone := 0
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if ink := c.pixels[(int(y)+dy)*c.width+int(x)+dx].ink; ink == stoneInk || ink == shadedStoneInk {
				stone++
			}
		}
	}
	if stone < 10 {
		t.Fatalf("stone pixels where the four tiles meet = %d of 25, want the fountain's pedestal", stone)
	}
}

// stepped is a meadow crossed by a barrier of a kind stepping diagonally from tile to tile, the
// hero standing beside it.
func stepped(barrier world.Ground) *world.World {
	ground := make([][]world.Ground, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
	}
	for step := range 12 {
		ground[8+step][14+step] = barrier
	}
	return world.NewOnMap(7, world.Map{Name: "Stepped", Ground: ground, StartCol: 22, StartRow: 13})
}

// inkNear counts the pixels in one of the inks within two pixels of where a point shows.
func inkNear(c *forestCanvas, x, y, z float64, inks ...int) int {
	px, py, _ := c.project(x, y, z)
	found := 0
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			if slices.Contains(inks, c.pixels[(int(py)+dy)*c.width+int(px)+dx].ink) {
				found++
			}
		}
	}
	return found
}

func TestWallsAndFencesSteppingDiagonallyJoinAcrossTheCorners(t *testing.T) {
	for _, barrier := range []struct {
		ground world.Ground
		height float64
		inks   []int
	}{
		{world.Wall, 1.5, []int{stoneInk, shadedStoneInk, mossInk}},
		{world.Battlement, 1.5, []int{stoneInk, shadedStoneInk, mossInk}},
		{world.Fence, 0.9, []int{plankInk}},
	} {
		r := newRenderer(stepped(barrier.ground), "L’Après-Règne")
		r.zoom = zoom{current: 6}
		c := r.render(160, 90, 0)
		// The corner where the steps on tiles 20, 14 and 21, 15 meet.
		if found := inkNear(c, 21*world.TileSize, barrier.height, 15*world.TileSize, barrier.inks...); found < 5 {
			t.Errorf("%v: %d of 25 pixels at the corner between two steps drawn, want it joined across", barrier.ground, found)
		}
	}
}

func TestASignpostOfFourTilesStandsOnceWhereTheyMeet(t *testing.T) {
	var signpost []world.PlacedFixture
	for _, tile := range [4][2]int{{20, 12}, {21, 12}, {20, 13}, {21, 13}} {
		signpost = append(signpost, world.PlacedFixture{Col: tile[0], Row: tile[1], Kind: world.Signpost})
	}
	w := world.NewOnMap(7, world.Map{Name: "Crossroads", Ground: make2D(30, 40), StartCol: 20, StartRow: 16, Fixtures: signpost})
	_, c := closeView(w)
	if found := inkNear(c, 21*world.TileSize, 1.0, 13*world.TileSize, plankInk); found < 5 {
		t.Fatalf("%d of 25 pixels of the post where the four tiles meet, want the signpost standing there", found)
	}
	if found := inkNear(c, 20.5*world.TileSize, 1.0, 13.5*world.TileSize, plankInk); found > 0 {
		t.Fatalf("%d pixels of a post in the middle of one of its tiles, want one signpost, not four", found)
	}
}

func TestStepsAreDrawnInStone(t *testing.T) {
	ground := make([][]world.Ground, 30)
	heights := make([][]int, 30)
	for row := range ground {
		ground[row] = make([]world.Ground, 40)
		heights[row] = make([]int, 40)
		for col := range heights[row] {
			heights[row][col] = min(4, max(0, col-17))
		}
	}
	for row := 14; row <= 16; row++ {
		for col := 18; col <= 20; col++ {
			ground[row][col] = world.Steps
		}
	}
	w := world.NewOnMap(7, world.Map{Name: "Stair", Ground: ground, Heights: heights, StartCol: 15, StartRow: 15})
	_, c := closeView(w)
	c.lift = 0
	if found := inkNear(c, 39, w.Elevation(39, 31), 31, floorInk, shadowFloorInk); found < 10 {
		t.Fatalf("%d of 25 pixels of stone on the steps, want them drawn in stone", found)
	}
}
