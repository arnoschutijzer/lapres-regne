package window

import (
	"image"
	"math"
	"sort"

	"lapresregne/internal/world"
)

// screenWidth and screenHeight are the window's size in art pixels, before it is scaled up to fit:
// Hyper Light Drifter's own.
const screenWidth, screenHeight = 480, 270

// unit is how many art pixels one world unit spans across the screen.
const unit = 16.0

// depthSquash is how far down the screen the ground recedes for each unit it runs toward the
// viewer, compared with a unit across: the view looks down at the ground at a slant.
const depthSquash = 0.72

// heightSquash is how far up the screen a point rises for each unit it stands above the ground,
// compared with a unit across.
const heightSquash = 0.69

// heightScale stretches every height the window shows, so the hills stand out as they do in
// games drawn in two and a half dimensions.
const heightScale = 1.6

// lift is how many art pixels up the screen a point rises for each unit of height.
const lift = unit * heightSquash * heightScale

// ledge is how high each of the ledges the ground is shown in rises above the last, in world units.
const ledge = 1.5

// ledgeSlope is how much of the ground's own slope a ledge keeps; the rest gathers into the sheer
// face up to the next ledge.
const ledgeSlope = 0.2

// level is which ledge ground of a height lies on, counted up from the lowest.
func level(h float64) float64 { return math.Floor(h / ledge) }

// terraced is the height the window shows ground of a height at: on ledges that each slope gently,
// with a sheer face between one and the next, as cliffs show height in games drawn in two and a
// half dimensions. Whatever stands on the ground stands at this height too.
func terraced(h float64) float64 {
	base := level(h) * ledge
	return base + (h-base)*ledgeSlope
}

// project is the world's own art pixel showing a point of the world at a height: across with x,
// down the screen as it runs toward the viewer, and up as it rises. It holds whatever the camera
// shows, so painted ground can be kept.
func project(x, z, h float64) (px, py float64) {
	return x * unit, z*unit*depthSquash - h*lift
}

// camera is the point of the world the window is centred on, at the height of the ground there.
type camera struct{ x, z, h float64 }

// follow eases the camera toward the hero, standing on ground at height h, over dt seconds.
func (c *camera) follow(x, z, h, dt float64) {
	ease := math.Min(1, dt*6)
	c.x += (x - c.x) * ease
	c.z += (z - c.z) * ease
	c.h += (h - c.h) * ease
}

// origin is the world's art pixel at the screen's top left corner: a whole pixel, so sprites and
// ground keep crisp edges.
func (c camera) origin() (px, py int) {
	x, y := project(c.x, c.z, c.h)
	return int(math.Round(x)) - screenWidth/2, int(math.Round(y)) - screenHeight/2
}

// toScreen is where a point of the world at a height shows on the screen.
func (c camera) toScreen(x, z, h float64) (sx, sy float64) {
	originX, originY := c.origin()
	px, py := project(x, z, h)
	return px - float64(originX), py - float64(originY)
}

// placed is a sprite at its place on the screen: its art's top left at x, y. tree is the standing
// tree it shows, if any.
type placed struct {
	art   *image.RGBA
	x, y  int
	depth float64
	tree  *world.Tree
}

// sprites draws each sprite once at a size and keeps it.
type sprites struct {
	trees   map[treeLook]*image.RGBA
	stump   *image.RGBA
	heroes  map[heroLook]*image.RGBA
	shadows map[image.Point]*image.RGBA
}

// treeLook is what tells one tree's art from another's.
type treeLook struct {
	kind             world.TreeKind
	crownWidth, tall int
	variation        int
}

func newSprites() *sprites {
	return &sprites{trees: map[treeLook]*image.RGBA{}, stump: stumpSprite(), heroes: map[heroLook]*image.RGBA{}, shadows: map[image.Point]*image.RGBA{}}
}

// crownWidth is how many art pixels across a tree's crown spans.
func crownWidth(t world.Tree) int { return int(math.Round(t.Radius * 2 * unit * 0.8)) }

// shadow is the flat shadow an ellipse across wide pixels and down tall casts.
func (s *sprites) shadow(across, down int) *image.RGBA {
	size := image.Pt(across, down)
	if _, ok := s.shadows[size]; !ok {
		s.shadows[size] = shadowSprite(across, down)
	}
	return s.shadows[size]
}

func (s *sprites) tree(t world.Tree) *image.RGBA {
	if t.Stump {
		return s.stump
	}
	look := treeLook{t.Kind, crownWidth(t), int(math.Round(t.Height * unit * heightSquash)), int(t.Variation * 4)}
	if _, ok := s.trees[look]; !ok {
		s.trees[look] = treeSprite(look.kind, look.crownWidth, look.tall, float64(look.variation)/4)
	}
	return s.trees[look]
}

// heroLook is what tells one picture of the hero from another.
type heroLook struct {
	facing world.Facing
	frame  int
}

func (s *sprites) hero(facing world.Facing, frame int) *image.RGBA {
	look := heroLook{facing, frame}
	if _, ok := s.heroes[look]; !ok {
		s.heroes[look] = heroSprite(facing, frame)
	}
	return s.heroes[look]
}

// arrange places every sprite the camera shows, the farthest first, so nearer sprites cover them,
// and the shadows they cast on the ground. heroFrame is the frame of the hero's walk.
func arrange(w *world.World, view camera, art *sprites, heroFrame int) (scene, shadows []placed) {
	onScreen := func(left, top int, sprite *image.RGBA) bool {
		bounds := sprite.Bounds()
		return left <= screenWidth && top <= screenHeight && left+bounds.Dx() >= 0 && top+bounds.Dy() >= 0
	}
	// A sprite stands on the pixel at its bottom middle; a shadow lies centred a little down and
	// right of its caster's foot, away from the light.
	put := func(sprite *image.RGBA, x, z, h float64, tree *world.Tree) {
		sx, sy := view.toScreen(x, z, h)
		bounds := sprite.Bounds()
		left := int(math.Round(sx)) - bounds.Dx()/2
		top := int(math.Round(sy)) - bounds.Dy() + 1
		if onScreen(left, top, sprite) {
			scene = append(scene, placed{art: sprite, x: left, y: top, depth: z, tree: tree})
		}
	}
	cast := func(across, down int, x, z, h float64) {
		sx, sy := view.toScreen(x, z, h)
		shadow := art.shadow(across, down)
		left := int(math.Round(sx)) - across/2 + across/6
		top := int(math.Round(sy)) - down/2 + down/8
		if onScreen(left, top, shadow) {
			shadows = append(shadows, placed{art: shadow, x: left, y: top, depth: z})
		}
	}
	// Trees reach into the view from below its bottom edge, so the search runs past it.
	// Trees standing on high ground far off rise into view, so the search reaches back past the top
	// edge by the highest the ground lies.
	left, right := view.x-screenWidth/2/unit-4, view.x+screenWidth/2/unit+4
	back := view.z - (screenHeight/2+(w.Highest()-view.h+4)*lift)/(unit*depthSquash) - 2
	front := view.z + (screenHeight/2+view.h*lift)/(unit*depthSquash) + 10
	spacing := w.TreeGrid()
	for gz := int(math.Floor(back / spacing)); gz <= int(math.Ceil(front/spacing)); gz++ {
		for gx := int(math.Floor(left / spacing)); gx <= int(math.Ceil(right/spacing)); gx++ {
			if tree, ok := w.TreeAt(gx, gz); ok {
				h := terraced(w.Elevation(tree.X, tree.Z))
				put(art.tree(tree), tree.X, tree.Z, h, &tree)
				// A crown of leaves casts a broad shadow; a stump or bare limbs barely do.
				if !tree.Stump && tree.Kind != world.DeadTree {
					across := crownWidth(tree) * 4 / 5
					cast(across, across*9/20, tree.X, tree.Z, h)
				}
			}
		}
	}
	x, z := w.Hero().Position()
	h := terraced(w.ElevationOn(w.Storey(), x, z))
	put(art.hero(w.Hero().Facing(), heroFrame), x, z, h, nil)
	cast(11, 5, x, z, h)
	sort.SliceStable(scene, func(i, j int) bool { return scene[i].depth < scene[j].depth })
	return scene, shadows
}

// treeAt is the frontmost standing tree whose art covers a point of the screen.
func treeAt(scene []placed, sx, sy int) (world.Tree, bool) {
	for i := len(scene) - 1; i >= 0; i-- {
		p := scene[i]
		point := image.Pt(sx-p.x, sy-p.y)
		if p.tree == nil || p.tree.Stump || !point.In(p.art.Bounds()) {
			continue
		}
		if p.art.RGBAAt(point.X, point.Y).A != 0 {
			return *p.tree, true
		}
	}
	return world.Tree{}, false
}
