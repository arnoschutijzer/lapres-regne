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

// camera is the point of the world the window is centred on.
type camera struct{ x, z float64 }

// toScreen is where a point of the ground lies on the screen.
func (c camera) toScreen(x, z float64) (sx, sy float64) {
	return screenWidth/2 + (x-c.x)*unit, screenHeight/2 + (z-c.z)*unit*depthSquash
}

// toWorld is the point of the ground shown at a point of the screen.
func (c camera) toWorld(sx, sy float64) (x, z float64) {
	return c.x + (sx-screenWidth/2)/unit, c.z + (sy-screenHeight/2)/(unit*depthSquash)
}

// follow eases the camera toward the hero over dt seconds.
func (c *camera) follow(x, z, dt float64) {
	ease := math.Min(1, dt*6)
	c.x += (x - c.x) * ease
	c.z += (z - c.z) * ease
}

// origin is the world's art pixel at the screen's top left corner, once the camera is snapped.
func (c camera) origin() (px, py int) {
	return int(math.Round(c.x*unit)) - screenWidth/2, int(math.Round(c.z*unit*depthSquash)) - screenHeight/2
}

// snapped is the camera moved onto whole art pixels, so sprites and ground keep crisp edges.
func (c camera) snapped() camera {
	return camera{math.Round(c.x*unit) / unit, math.Round(c.z*unit*depthSquash) / (unit * depthSquash)}
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

// heightSquash is how far up the screen a point rises for each unit it stands above the ground,
// compared with a unit across.
const heightSquash = 0.69

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
	put := func(sprite *image.RGBA, x, z float64, tree *world.Tree) {
		sx, sy := view.toScreen(x, z)
		bounds := sprite.Bounds()
		left := int(math.Round(sx)) - bounds.Dx()/2
		top := int(math.Round(sy)) - bounds.Dy() + 1
		if onScreen(left, top, sprite) {
			scene = append(scene, placed{art: sprite, x: left, y: top, depth: z, tree: tree})
		}
	}
	cast := func(across, down int, x, z float64) {
		sx, sy := view.toScreen(x, z)
		shadow := art.shadow(across, down)
		left := int(math.Round(sx)) - across/2 + across/6
		top := int(math.Round(sy)) - down/2 + down/8
		if onScreen(left, top, shadow) {
			shadows = append(shadows, placed{art: shadow, x: left, y: top, depth: z})
		}
	}
	// Trees reach into the view from below its bottom edge, so the search runs past it.
	left, back := view.toWorld(-unit*4, -unit*2)
	right, front := view.toWorld(screenWidth+unit*4, screenHeight+unit*10)
	spacing := w.TreeGrid()
	for gz := int(math.Floor(back / spacing)); gz <= int(math.Ceil(front/spacing)); gz++ {
		for gx := int(math.Floor(left / spacing)); gx <= int(math.Ceil(right/spacing)); gx++ {
			if tree, ok := w.TreeAt(gx, gz); ok {
				put(art.tree(tree), tree.X, tree.Z, &tree)
				// A crown of leaves casts a broad shadow; a stump or bare limbs barely do.
				if !tree.Stump && tree.Kind != world.DeadTree {
					across := crownWidth(tree) * 4 / 5
					cast(across, across*9/20, tree.X, tree.Z)
				}
			}
		}
	}
	x, z := w.Hero().Position()
	put(art.hero(w.Hero().Facing(), heroFrame), x, z, nil)
	cast(11, 5, x, z)
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
