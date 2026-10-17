package window

import (
	"image"
	"image/color"
	"math"
	"sort"

	"lapresregne/internal/world"
)

// screenWidth and screenHeight are the window's size in art pixels, before it is scaled up to fit.
const screenWidth, screenHeight = 640, 360

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

// snapped is the camera moved onto whole art pixels, so sprites and ground keep crisp edges.
func (c camera) snapped() camera {
	return camera{math.Round(c.x*unit) / unit, math.Round(c.z*unit*depthSquash) / (unit * depthSquash)}
}

// background is the color of whatever lies beyond the world.
var background = color.RGBA{8, 6, 14, 255}

// groundColor is the color of the ground at a world point. px and py are the world's own art pixel
// there, so the texture stays put on the ground while the camera moves.
func groundColor(w *world.World, x, z float64, px, py int) color.RGBA {
	// Tufts a few pixels across give the ground a calm texture rather than static.
	texture := (hash(px/6, py/4, 5) - 0.5) * 0.16
	if w.Mapped() {
		ground, ok := w.GroundAt(x, z)
		if !ok {
			return background
		}
		switch ground {
		case world.Water:
			return ramp(waterMaterial, 0.5+texture)
		case world.Path:
			return ramp(pathMaterial, 0.55+texture)
		case world.Floor, world.Door:
			return ramp(stoneMaterial, 0.5+texture)
		case world.Boards:
			return ramp(boardsMaterial, 0.5+texture)
		}
		return ramp(grassMaterial, 0.6-w.Cover(x, z)*0.4+texture)
	}
	if distance, width := w.River(x, z); distance < width {
		// Ripples run across the river in thin bands.
		ripple := 0.1 * math.Sin(float64(py)*0.9+float64(px)*0.15)
		return ramp(waterMaterial, 0.45+ripple+texture*0.5)
	}
	// Grass darkens under the canopy and in broad, soft patches.
	return ramp(grassMaterial, 0.5-w.Cover(x, z)*0.45+(w.Noise(x/6, z/6, 3)-0.5)*0.3+texture)
}

// paintGround fills an image of the screen with the ground the camera shows. Each 2 x 2 block of
// art pixels shows one sample of the ground.
func paintGround(ground *image.RGBA, w *world.World, view camera) {
	originX := int(math.Round(view.x*unit)) - screenWidth/2
	originY := int(math.Round(view.z*unit*depthSquash)) - screenHeight/2
	for sy := 0; sy < screenHeight; sy += 2 {
		for sx := 0; sx < screenWidth; sx += 2 {
			x, z := view.toWorld(float64(sx)+1, float64(sy)+1)
			c := groundColor(w, x, z, originX+sx, originY+sy)
			for dy := range 2 {
				for dx := range 2 {
					ground.SetRGBA(sx+dx, sy+dy, c)
				}
			}
		}
	}
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
	trees  map[treeLook]*image.RGBA
	stumps map[int]*image.RGBA
	heroes map[world.Facing]*image.RGBA
}

// treeLook is what tells one tree's art from another's.
type treeLook struct {
	kind             world.TreeKind
	crownWidth, tall int
	variation        int
}

func newSprites() *sprites {
	return &sprites{trees: map[treeLook]*image.RGBA{}, stumps: map[int]*image.RGBA{}, heroes: map[world.Facing]*image.RGBA{}}
}

func (s *sprites) tree(t world.Tree) *image.RGBA {
	if t.Stump {
		width := max(6, int(math.Round(t.Trunk*2*unit*1.6)))
		if _, ok := s.stumps[width]; !ok {
			s.stumps[width] = stumpSprite(width)
		}
		return s.stumps[width]
	}
	look := treeLook{t.Kind, int(math.Round(t.Radius * 2 * unit)), int(math.Round(t.Height * unit)), int(t.Variation * 4)}
	if _, ok := s.trees[look]; !ok {
		s.trees[look] = treeSprite(look.kind, look.crownWidth, look.tall, float64(look.variation)/4)
	}
	return s.trees[look]
}

func (s *sprites) hero(facing world.Facing) *image.RGBA {
	if _, ok := s.heroes[facing]; !ok {
		s.heroes[facing] = heroSprite(facing, unit)
	}
	return s.heroes[facing]
}

// arrange places every sprite the camera shows, the farthest first, so nearer sprites cover them.
func arrange(w *world.World, view camera, art *sprites) []placed {
	var scene []placed
	put := func(sprite *image.RGBA, x, z float64, tree *world.Tree) {
		sx, sy := view.toScreen(x, z)
		bounds := sprite.Bounds()
		left := int(math.Round(sx)) - bounds.Dx()/2
		top := int(math.Round(sy)) - bounds.Dy() + 1
		if left > screenWidth || top > screenHeight || left+bounds.Dx() < 0 || top+bounds.Dy() < 0 {
			return
		}
		scene = append(scene, placed{art: sprite, x: left, y: top, depth: z, tree: tree})
	}
	// Trees reach into the view from below its bottom edge, so the search runs past it.
	left, back := view.toWorld(-unit*4, -unit*2)
	right, front := view.toWorld(screenWidth+unit*4, screenHeight+unit*10)
	spacing := w.TreeGrid()
	for gz := int(math.Floor(back / spacing)); gz <= int(math.Ceil(front/spacing)); gz++ {
		for gx := int(math.Floor(left / spacing)); gx <= int(math.Ceil(right/spacing)); gx++ {
			if tree, ok := w.TreeAt(gx, gz); ok {
				put(art.tree(tree), tree.X, tree.Z, &tree)
			}
		}
	}
	x, z := w.Hero().Position()
	put(art.hero(w.Hero().Facing()), x, z, nil)
	sort.SliceStable(scene, func(i, j int) bool { return scene[i].depth < scene[j].depth })
	return scene
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
