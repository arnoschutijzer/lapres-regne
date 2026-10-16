package terminal

import (
	"strings"
	"testing"

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
		if c.pixels[y*c.width+int(x)].ink == stoneInk {
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
	if p := c.pixels[int(lintel)*c.width+int(x)]; p.ink != stoneInk {
		t.Fatalf("ink above the doorway = %d, want the stone lintel", p.ink)
	}
	if p := c.pixels[int(opening)*c.width+int(x)]; p.ink == stoneInk {
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
