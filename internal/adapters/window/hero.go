package window

import (
	"image"
	"image/color"
	"math"

	"lapresregne/internal/world"
)

// The hero is drawn by hand as a slender base body, with what they wear laid over it as separate
// layers, so gear can later cover or replace parts without restyling the body. Each picture is
// heroWidth pixels wide; the body's rows run from the top of the head to the soles of the feet.
// Letters are colors from heroLegend; dots are clear.
const heroWidth = 13

var heroLegend = map[byte]color.RGBA{
	'o': rgb(0x050405), // outline
	'e': rgb(0x050405), // eyes
	'H': rgb(0x2e221b), // hair in the light
	'h': rgb(0x17110e), // hair in shade
	'a': rgb(0xa07c64), // skin in the light
	'b': rgb(0x76563f), // skin
	'c': rgb(0x483226), // skin in shade
	'w': rgb(0x8a8274), // undershirt in the light, worn and grimy
	'W': rgb(0x5e584e), // undershirt
	'v': rgb(0x34302b), // undershirt in shade
	'p': rgb(0x3a3732), // shorts in the light
	'P': rgb(0x24221f), // shorts
	'q': rgb(0x121110), // shorts in shade
}

// Each facing has an upper body, from the head to the hips, and legs standing and in two strides.
type heroPose struct {
	upper, upperWorn []string
	legs, legsWorn   [3][]string
}

// The body keeps real proportions: a small head, narrow shoulders and long legs.
var (
	frontUpper = []string{
		".....ooo.....",
		"....oHHho....",
		"....oHhho....",
		"....oebeo....",
		"....oabco....",
		".....obo.....",
		"...ooabcoo...",
		".oaabbbbbcco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".obobbbbboco.",
		"..oobbbbboo..",
		"...obbbbbo...",
	}
	backUpper = []string{
		".....ooo.....",
		"....oHHho....",
		"....oHhho....",
		"....ohhho....",
		"....obhco....",
		".....obo.....",
		"...ooabcoo...",
		".oaabbbbbcco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".oaobbbbboco.",
		".obobbbbboco.",
		"..oobbbbboo..",
		"...obbbbbo...",
	}
	// The undershirt has short sleeves over the shoulders; the shorts reach the top of the legs.
	frontUpperWorn = []string{
		".............",
		".............",
		".............",
		".............",
		".............",
		".............",
		".............",
		"..wwWWWWWvv..",
		"..w.WWWWW.v..",
		"....wWWWv....",
		"....wWWWv....",
		"....wWWWv....",
		"....wWWWv....",
		"....wWWWv....",
		"....vvvvv....",
		"....pPPPq....",
		"....pPPPq....",
		"....pPPPq....",
	}
	frontLegs = [3][]string{{
		"...obbobbo...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"...oabobco...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"..oaabobcco..",
		"..ooooooooo..",
	}, {
		"...obbobbo...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"...oabobco...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"..oaabobco...",
		"..oooo.obcco.",
		".......oooo..",
	}, {
		"...obbobbo...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"...oabobco...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"...obbobco...",
		"...obbobcco..",
		".oaabo.oooo..",
		"..oooo.......",
	}}
	frontLegsWorn = [3][]string{
		{"....pP.Pq....", "....pq.Pq...."},
		{"....pP.Pq....", "....pq.Pq...."},
		{"....pP.Pq....", "....pq.Pq...."},
	}
	sideUpper = []string{
		".....ooo.....",
		"....oHHho....",
		"....oHhabo...",
		"....ohbbeo...",
		"....obbbco...",
		".....obco....",
		".....obbo....",
		"....oabbco...",
		"....oabbco...",
		"....oabbco...",
		"....oabbco...",
		"....oabbco...",
		"....oabbco...",
		"....oabbco...",
		"....oabbco...",
		"....obabco...",
		"....obbbco...",
		"....obbbco...",
	}
	sideUpperWorn = []string{
		".............",
		".............",
		".............",
		".............",
		".............",
		".............",
		".............",
		".....wWWv....",
		".....wWWv....",
		".....wWWv....",
		".....wWWv....",
		".....wWWv....",
		".....wWWv....",
		".....vvvv....",
		".....pPPq....",
		".....pPPq....",
		".....pPPq....",
		".....pPPq....",
	}
	sideLegs = [3][]string{{
		"....obbbo....",
		"....obbco....",
		"....obbco....",
		"....obbco....",
		"....oabco....",
		"....obbco....",
		"....obbco....",
		"....obbco....",
		"....obbco....",
		"....obbco....",
		"....obbbbo...",
		"....oooooo...",
	}, {
		"....obbbo....",
		"...ocbobbo...",
		"...ocbobbo...",
		"...ocbobbo...",
		"...ocb.obbo..",
		"..ocbo..obo..",
		"..ocbo..obo..",
		"..ocbo..obo..",
		"..ocbo..obo..",
		"..ocbo..obo..",
		".occbo..obbo.",
		".ooooo..oooo.",
	}, {
		"....obbbo....",
		"...obbocbo...",
		"...obbocbo...",
		"...obbocbo...",
		"...obb.ocbo..",
		"..obbo..oco..",
		"..obbo..oco..",
		"..obbo..oco..",
		"..obbo..oco..",
		"..obbo..oco..",
		".obbbo..occo.",
		".ooooo..oooo.",
	}}
	sideLegsWorn = [3][]string{
		{".....pPq....."},
		{".....pPq....."},
		{".....pPq....."},
	}
)

var heroPoses = map[world.Facing]heroPose{
	world.FacingDown:  {frontUpper, frontUpperWorn, frontLegs, frontLegsWorn},
	world.FacingUp:    {backUpper, frontUpperWorn, frontLegs, frontLegsWorn},
	world.FacingRight: {sideUpper, sideUpperWorn, sideLegs, sideLegsWorn},
}

// heroStride is how far the hero walks for each stride of their walk, in world units: one step of
// each foot.
const heroStride = 1.1

// heroSprite draws the hero facing one way in one frame of their walk: 0 standing, then 1 and 2
// for each foot forward. Their feet are at the bottom centre. Facing left mirrors facing right.
func heroSprite(facing world.Facing, frame int) *image.RGBA {
	pose := heroPoses[facing]
	if facing == world.FacingLeft {
		pose = heroPoses[world.FacingRight]
	}
	legs := pose.legs[frame]
	sprite := image.NewRGBA(image.Rect(0, 0, heroWidth, len(pose.upper)+len(legs)))
	none := func(int) color.RGBA { return color.RGBA{} }
	lay(sprite, pose.upper, 0, 0, none, heroLegend)
	lay(sprite, legs, 0, len(pose.upper), none, heroLegend)
	lay(sprite, pose.upperWorn, 0, 0, none, heroLegend)
	lay(sprite, pose.legsWorn[frame], 0, len(pose.upper), none, heroLegend)
	if facing == world.FacingLeft {
		return mirrored(sprite)
	}
	return sprite
}

// mirrored flips a sprite left to right.
func mirrored(sprite *image.RGBA) *image.RGBA {
	bounds := sprite.Bounds()
	flipped := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			flipped.SetRGBA(bounds.Max.X-1-x+bounds.Min.X, y, sprite.RGBAAt(x, y))
		}
	}
	return flipped
}

// gait follows how far the hero has walked, to pick the frame of their walk.
type gait struct {
	lastX, lastZ float64
	walked       float64
	moving       bool
	started      bool
}

// follow notes where the hero stands now.
func (g *gait) follow(x, z float64) {
	if g.started {
		distance := (x-g.lastX)*(x-g.lastX) + (z-g.lastZ)*(z-g.lastZ)
		g.moving = distance > 1e-8
		if g.moving {
			g.walked += math.Sqrt(distance)
		}
	}
	g.lastX, g.lastZ, g.started = x, z, true
}

// frame is the frame of the walk to draw: standing while still, else each foot forward in turn.
func (g *gait) frame() int {
	if !g.moving {
		return 0
	}
	if int(g.walked/(heroStride/2))%2 == 0 {
		return 1
	}
	return 2
}
