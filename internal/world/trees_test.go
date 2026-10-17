package world_test

import (
	"testing"
	"time"

	"lapresregne/internal/world"
)

func TestTreeKindsGrowByRegion(t *testing.T) {
	w := world.New(7)
	counts := map[world.TreeKind]int{}
	for gz := -30; gz <= 30; gz++ {
		for gx := -30; gx <= 30; gx++ {
			tree, ok := w.TreeAt(gx, gz)
			if !ok {
				continue
			}
			counts[tree.Kind]++
			distance, width := w.River(tree.X, tree.Z)
			deep := w.Cover(tree.X, tree.Z) > 0.6
			switch {
			case distance < width+4 && tree.Kind != world.Willow:
				t.Fatalf("%v at x %.1f z %.1f grows by a river, want a willow", tree.Kind, tree.X, tree.Z)
			case distance >= width+4 && deep && tree.Kind != world.Oak && tree.Kind != world.Yew:
				t.Fatalf("%v at x %.1f z %.1f grows in deep woods, want an oak or a yew", tree.Kind, tree.X, tree.Z)
			case distance >= width+4 && !deep && tree.Kind != world.NormalTree:
				t.Fatalf("%v at x %.1f z %.1f grows in open woodland, want a normal tree", tree.Kind, tree.X, tree.Z)
			}
		}
	}
	if counts[world.Yew] == 0 || counts[world.Yew]*10 > counts[world.Oak] {
		t.Fatalf("yews = %d, oaks = %d; want yews rare among the oaks", counts[world.Yew], counts[world.Oak])
	}
	if counts[world.Willow] == 0 || counts[world.NormalTree] == 0 {
		t.Fatalf("counts = %v, want willows and normal trees too", counts)
	}
}

// besideWillow stands the hero facing seed 7's riverside willow at x 11.0, z -3.6.
func besideWillow() *world.World {
	w := world.New(7)
	for range 6 {
		w.Walk(0, -1, false)
	}
	for range 18 {
		w.Walk(1, 0, false)
	}
	return w
}

func TestAWillowRefusesALevelOneHero(t *testing.T) {
	w := besideWillow()
	if tree, _ := w.TreeInReach(); tree.Kind != world.Willow {
		t.Fatalf("tree in reach = %v, want the willow", tree.Kind)
	}
	if w.Chop() || w.Chopping() {
		t.Fatal("a level 1 hero must not chop a willow")
	}
	if message := w.Message(); message != "You need woodcutting level 30 to chop this willow." {
		t.Fatalf("message = %q, want the level requirement", message)
	}
	w.Walk(-1, 0, false)
	if message := w.Message(); message != "" {
		t.Fatalf("message = %q, want it cleared by walking away", message)
	}
}

func TestApproachingATreeAboveTheHerosLevelRefusesOnArrival(t *testing.T) {
	w := world.New(7)
	willow, _ := w.TreeAt(2, -1)
	if !w.Approach(willow) {
		t.Fatal("a willow must be approachable whatever the level")
	}
	approachUntilChopping(w)
	if w.Chopping() || w.Message() != "You need woodcutting level 30 to chop this willow." {
		t.Fatalf("chopping = %v, message = %q; want a refusal on arrival", w.Chopping(), w.Message())
	}
}

func TestADeadTreeIsChoppedForLogsLikeANormalTree(t *testing.T) {
	if world.DeadTree.String() != "dead tree" || world.DeadTree.Logs() != world.Logs {
		t.Fatalf("dead tree = %q giving %v, want a dead tree giving logs", world.DeadTree, world.DeadTree.Logs())
	}
	m := pond()
	m.Trees = []world.PlantedTree{{Col: 1, Row: 1, Kind: world.DeadTree}}
	w := world.NewOnMap(7, m)
	tree, _ := w.TreeAt(1, 1)
	w.Approach(tree)
	walkFor(w, 5*time.Second)
	if !w.Chopping() {
		t.Fatal("a level 1 hero must chop a dead tree")
	}
}

func TestADeadTreeSpreadsNoCanopy(t *testing.T) {
	m := pond()
	m.Trees = []world.PlantedTree{{Col: 1, Row: 1, Kind: world.DeadTree}}
	w := world.NewOnMap(7, m)
	open := world.NewOnMap(7, world.Map{Name: "pond", Ground: m.Ground, StartCol: 1, StartRow: 2})
	x, z := 1.5*world.TileSize, 1.5*world.TileSize
	if w.Cover(x, z) != open.Cover(x, z) {
		t.Fatalf("cover under the dead tree = %v, want the open sky's %v", w.Cover(x, z), open.Cover(x, z))
	}
}
