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
