package world_test

import (
	"math"
	"testing"
	"time"

	"lapresregne/internal/world"
)

// pond is a small map: a meadow with a pond on its east side, an oak to the north of the start
// and a normal tree beside it.
func pond() world.Map {
	g, w := world.Grass, world.Water
	return world.Map{
		Name: "pond",
		Ground: [][]world.Ground{
			{g, g, g, g, g, w},
			{g, g, g, g, g, w},
			{g, g, g, g, w, w},
			{g, g, g, g, g, w},
		},
		Trees:    []world.PlantedTree{{Col: 1, Row: 0, Kind: world.Oak}, {Col: 2, Row: 1, Kind: world.NormalTree}},
		StartCol: 1, StartRow: 2,
	}
}

func TestTheHeroStartsOnTheMapsStartTile(t *testing.T) {
	w := world.NewOnMap(7, pond())
	x, z := w.Hero().Position()
	if x != 1.5*world.TileSize || z != 2.5*world.TileSize {
		t.Fatalf("position = %v, %v; want the centre of tile 1, 2", x, z)
	}
	if cameraX, cameraZ := w.Camera(); cameraX != x || cameraZ != z {
		t.Fatalf("camera = %v, %v; want it on the hero", cameraX, cameraZ)
	}
}

func TestTreesGrowOnlyWhereTheyArePlanted(t *testing.T) {
	w := world.NewOnMap(7, pond())
	oak, ok := w.TreeAt(1, 0)
	if !ok || oak.Kind != world.Oak {
		t.Fatalf("tree at 1, 0 = %v, %v; want the planted oak", oak.Kind, ok)
	}
	if oak.X < world.TileSize || oak.X >= 2*world.TileSize || oak.Z < 0 || oak.Z >= world.TileSize {
		t.Fatalf("oak at %v, %v; want it on its tile", oak.X, oak.Z)
	}
	if _, ok := w.TreeAt(0, 0); ok {
		t.Fatal("no tree must grow on an empty tile")
	}
	if w.TreeGrid() != world.TileSize {
		t.Fatalf("tree grid = %v, want one tree per tile", w.TreeGrid())
	}
}

func TestPlantedTreesSpreadNoWiderThanTheTilesAroundThem(t *testing.T) {
	m := pond()
	for col := range 5 {
		m.Trees = append(m.Trees, world.PlantedTree{Col: col, Row: 3, Kind: world.Oak})
	}
	w := world.NewOnMap(7, m)
	for col := range 5 {
		if tree, _ := w.TreeAt(col, 3); tree.Radius > 1.25*world.TileSize {
			t.Fatalf("oak on tile %d has a crown %v across, want it within its neighbours", col, tree.Radius)
		}
	}
}

func TestTheHeroCannotWalkOffTheMap(t *testing.T) {
	w := world.NewOnMap(7, pond())
	w.Advance(-1, 0, 5*time.Second, false)
	x, _ := w.Hero().Position()
	if x < 0 || x > world.TileSize {
		t.Fatalf("x = %v; want the hero stopped at the west edge", x)
	}
	w.Advance(0, 1, 5*time.Second, false)
	if _, z := w.Hero().Position(); z > 4*world.TileSize || z < 3*world.TileSize {
		t.Fatalf("z = %v; want the hero stopped at the south edge", z)
	}
}

func TestTheHeroSlidesAlongTheMapsEdge(t *testing.T) {
	w := world.NewOnMap(7, pond())
	w.Advance(-1, 0, 5*time.Second, false)
	_, before := w.Hero().Position()
	w.Advance(-1, -1, time.Second, false)
	if _, after := w.Hero().Position(); after >= before {
		t.Fatalf("z = %v after %v; want the hero sliding north along the edge", after, before)
	}
}

func TestWalkingToAPointOffTheMapStopsAtTheEdge(t *testing.T) {
	w := world.NewOnMap(7, pond())
	w.WalkTo(-10, 2.5*world.TileSize)
	for range 100 {
		w.Advance(0, 0, 100*time.Millisecond, false)
	}
	if x, _ := w.Hero().Position(); x < 0 {
		t.Fatalf("x = %v; want the hero still on the map", x)
	}
	w.Walk(1, 0, false)
	if x, _ := w.Hero().Position(); x <= 0 {
		t.Fatalf("x = %v; want the hero free to walk on", x)
	}
}

func TestTheHeroWadesThroughWaterTiles(t *testing.T) {
	w := world.NewOnMap(7, pond())
	w.WalkTo(4.5*world.TileSize, 2.5*world.TileSize)
	for range 100 {
		w.Advance(0, 0, 100*time.Millisecond, false)
	}
	if region := w.Region(); region != "river / wading" {
		t.Fatalf("region = %q, want the hero wading", region)
	}
	before, _ := w.Hero().Position()
	w.Advance(1, 0, 100*time.Millisecond, false)
	if after, _ := w.Hero().Position(); math.Abs(after-before-0.225) > 1e-9 {
		t.Fatalf("waded %v, want half of 0.45", after-before)
	}
}

func TestAPlantedTreeCanBeChopped(t *testing.T) {
	w := world.NewOnMap(7, pond())
	tree, ok := w.TreeAt(2, 1)
	if !ok || !w.Approach(tree) {
		t.Fatal("the planted tree must be approachable")
	}
	approachUntilChopping(w)
	if !w.Chopping() {
		t.Fatal("the hero must chop the planted tree on arrival")
	}
}

func TestResumingOnAMapReturnsTheHeroOnlyToTheSameMap(t *testing.T) {
	kept := world.NewOnMap(7, pond())
	kept.Advance(1, 0, 200*time.Millisecond, false)
	progress := &memoryProgress{}
	if err := kept.Keep(progress, quitAt); err != nil {
		t.Fatal(err)
	}
	if progress.snapshot.Map != "pond" {
		t.Fatalf("saved map = %q, want pond", progress.snapshot.Map)
	}
	w, err := world.ResumeOn(pond(), 7, progress, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	keptX, keptZ := kept.Hero().Position()
	if x, z := w.Hero().Position(); x != keptX || z != keptZ {
		t.Fatalf("position = %v, %v; want %v, %v", x, z, keptX, keptZ)
	}
	other := pond()
	other.Name = "another pond"
	w, _ = world.ResumeOn(other, 7, progress, quitAt)
	if x, z := w.Hero().Position(); x != 1.5*world.TileSize || z != 2.5*world.TileSize {
		t.Fatalf("position = %v, %v; want the other map's start", x, z)
	}
	w, _ = world.Resume(7, progress, quitAt)
	if x, z := w.Hero().Position(); x != 0 || z != 0 {
		t.Fatalf("position = %v, %v; want the endless forest's start", x, z)
	}
}

// courtyard is a walled yard: a stone floor inside walls with a gap to the south, a path leading
// out, and a fence, a gate and a grave beside it.
func courtyard() world.Map {
	g, p, f, w := world.Grass, world.Path, world.Floor, world.Wall
	return world.Map{
		Name: "courtyard",
		Ground: [][]world.Ground{
			{w, w, w, w, w, g, world.Fence, world.Gate, world.Fence},
			{w, f, f, f, w, g, g, g, g},
			{w, f, f, f, w, g, world.Grave, g, g},
			{w, w, f, w, w, g, g, g, g},
			{g, g, p, g, g, g, g, g, g},
		},
		StartCol: 2, StartRow: 1,
	}
}

func TestOnlyStructuresBlockTheWay(t *testing.T) {
	for ground, blocks := range map[world.Ground]bool{
		world.Grass: false, world.Water: false, world.Path: false, world.Floor: false, world.Boards: false, world.Door: false,
		world.Wall: true, world.Battlement: true, world.Fence: true, world.Gate: true, world.Grave: true,
	} {
		if ground.Blocks() != blocks {
			t.Errorf("ground %v blocks = %v, want %v", ground, ground.Blocks(), blocks)
		}
	}
}

func TestWallsStopTheHero(t *testing.T) {
	w := world.NewOnMap(7, courtyard())
	w.Advance(-1, 0, 2*time.Second, false)
	if x, _ := w.Hero().Position(); x < world.TileSize || x > 1.2*world.TileSize {
		t.Fatalf("x = %v; want the hero stopped against the west wall", x)
	}
	w.Advance(0, -1, 2*time.Second, false)
	if _, z := w.Hero().Position(); z < world.TileSize {
		t.Fatalf("z = %v; want the hero stopped against the north wall", z)
	}
}

func TestTheHeroWalksOutThroughADoorway(t *testing.T) {
	m := courtyard()
	m.Ground[3][2] = world.Door
	w := world.NewOnMap(7, m)
	w.Advance(0, 1, 2*time.Second, false)
	if _, z := w.Hero().Position(); z < 4*world.TileSize {
		t.Fatalf("z = %v; want the hero out through the door onto the path", z)
	}
}

func TestTheHeroWalksOutThroughAGapInTheWall(t *testing.T) {
	w := world.NewOnMap(7, courtyard())
	w.Advance(0, 1, 2*time.Second, false)
	if _, z := w.Hero().Position(); z < 4*world.TileSize {
		t.Fatalf("z = %v; want the hero out on the path", z)
	}
}

func TestFencesGatesAndGravesStopTheHero(t *testing.T) {
	m := courtyard()
	for _, start := range [][2]int{{6, 1}, {7, 1}, {6, 3}} {
		m.StartCol, m.StartRow = start[0], start[1]
		w := world.NewOnMap(7, m)
		w.Advance(0, -1, 2*time.Second, false)
		if _, z := w.Hero().Position(); z < float64(start[1])*world.TileSize {
			t.Fatalf("from tile %v z = %v; want the hero stopped by what stands north", start, z)
		}
	}
}

// walkFor advances the world in small steps for a while without any keys held.
func walkFor(w *world.World, duration time.Duration) {
	for elapsed := time.Duration(0); elapsed < duration; elapsed += 50 * time.Millisecond {
		w.Advance(0, 0, 50*time.Millisecond, false)
	}
}

func TestWalkingToAPointBeyondAWallGoesAroundThroughTheGap(t *testing.T) {
	w := world.NewOnMap(7, courtyard())
	w.WalkTo(11, 3)
	walkFor(w, 10*time.Second)
	if x, z := w.Hero().Position(); x != 11 || z != 3 {
		t.Fatalf("position = %v, %v; want the hero round the wall at 11, 3", x, z)
	}
}

func TestWalkingToAWallStopsOnTheNearestTileBesideIt(t *testing.T) {
	w := world.NewOnMap(7, courtyard())
	w.WalkTo(1, 5)
	walkFor(w, 5*time.Second)
	if x, z := w.Hero().Position(); math.Hypot(x-3, z-5) > 0.01 {
		t.Fatalf("position = %v, %v; want the hero on the floor beside the wall, at 3, 5", x, z)
	}
}

func TestApproachingATreeBeyondAWallGoesAroundToChopIt(t *testing.T) {
	m := courtyard()
	m.Trees = []world.PlantedTree{{Col: 6, Row: 1, Kind: world.NormalTree}}
	w := world.NewOnMap(7, m)
	tree, _ := w.TreeAt(6, 1)
	w.Approach(tree)
	walkFor(w, 10*time.Second)
	if !w.Chopping() {
		x, z := w.Hero().Position()
		t.Fatalf("hero at %v, %v is not chopping; want them round the wall and at the tree", x, z)
	}
}

func TestTheRegionNamesTheAreaTheHeroStandsIn(t *testing.T) {
	m := courtyard()
	m.Areas = [][]string{
		{"", "", "", "", "", "", "", "", ""},
		{"", "Keep", "Keep", "Keep"},
	}
	w := world.NewOnMap(7, m)
	if region := w.Region(); region != "Keep" {
		t.Fatalf("region = %q, want Keep", region)
	}
	w.WalkTo(11, 3)
	walkFor(w, 10*time.Second)
	if region := w.Region(); region != "tall-grass clearing" {
		t.Fatalf("region = %q outside any area, want the lay of the land", region)
	}
}

func TestASavedPlaceTheMapNowBlocksSendsTheHeroToTheStart(t *testing.T) {
	kept := world.NewOnMap(7, courtyard())
	kept.WalkTo(11, 3)
	walkFor(kept, 10*time.Second)
	progress := &memoryProgress{}
	kept.Keep(progress, quitAt)
	rebuilt := courtyard()
	rebuilt.Ground[1][5] = world.Fence
	w, err := world.ResumeOn(rebuilt, 7, progress, quitAt)
	if err != nil {
		t.Fatal(err)
	}
	if x, z := w.Hero().Position(); x != 5 || z != 3 {
		t.Fatalf("position = %v, %v; want the start, as a fence now stands where the hero was kept", x, z)
	}
}
