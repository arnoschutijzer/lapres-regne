package window

import (
	"image/color"
	"math"

	"lapresregne/internal/world"
)

// The window's palette is picked by hand after Blasphemous: umber, olive and ochre sunk into
// near-black, cold murky water, and crimson as the only bright accent. Every material runs through
// a few tones, darkest first, and the step from one tone to the next is broken by an ordered
// dither, as in the terminal's braille forest, so every surface carries grit.
type tones []color.RGBA

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
}

// at is a material's tone by index, held within its tones.
func (t tones) at(index int) color.RGBA {
	return t[min(len(t)-1, max(0, index))]
}

var (
	// voidColor is whatever lies beyond the world.
	voidColor = rgb(0x040405)

	// floorTones are the bare forest floor: dark earth and rot.
	floorTones = tones{rgb(0x060607), rgb(0x0e0d0d), rgb(0x181512), rgb(0x231e17), rgb(0x302919), rgb(0x433820)}
	// mossTones are the moss and dying grass that patch the floor.
	mossTones = tones{rgb(0x080b09), rgb(0x10150f), rgb(0x192014), rgb(0x242d18), rgb(0x33401c), rgb(0x4a5424)}
	// bankTones are the trodden mud along a river.
	bankTones  = tones{rgb(0x070708), rgb(0x0f0e0e), rgb(0x1a1715), rgb(0x26211c)}
	waterTones = tones{rgb(0x030506), rgb(0x070c0f), rgb(0x0d161a), rgb(0x162429), rgb(0x24383a), rgb(0x5f7470)}
	pathTones  = tones{rgb(0x0c0a0a), rgb(0x1a1512), rgb(0x2a2219), rgb(0x3c3022), rgb(0x52422c)}
	stoneTones = tones{rgb(0x09090b), rgb(0x141417), rgb(0x222125), rgb(0x333137), rgb(0x4a4650), rgb(0x6c6670)}
	boardTones = tones{rgb(0x0b0807), rgb(0x1c130e), rgb(0x2e1f15), rgb(0x44301e)}

	// Crimson is the forest's one bright accent: a few blood-red flowers in the dark.
	crimsonTones = tones{rgb(0x3a0810), rgb(0x76121e), rgb(0xae2430)}
	// litterTones are fallen leaves rotting on the floor.
	litterTones = tones{rgb(0x24120c), rgb(0x3e1f10), rgb(0x5c3218)}
	// ashTones and emberTones are the specks drifting in the air.
	ashTones   = tones{rgb(0x2a2826), rgb(0x6a6660), rgb(0xa8a298)}
	emberTones = tones{rgb(0x4a1608), rgb(0xa8481a), rgb(0xe8963a)}
)

// bayer holds the thresholds of a 4 x 4 ordered dither, the terminal forest's own.
var bayer = [4][4]float64{
	{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5},
}

// dithered is a material's color at a value counted in its tones, such as 2.4 for most of the way
// from its third tone to its fourth: the fraction picks between the two through the ordered dither
// at the pixel x, y.
func (t tones) dithered(value float64, x, y int) color.RGBA {
	threshold := (bayer[y&3][x&3] + 0.5) / 16
	return t.at(int(math.Floor(value + 1 - threshold)))
}

// Each kind of tree has its own leaves, from the shadow deep in its crown to the light on its top.
var canopyTones = map[world.TreeKind]tones{
	world.NormalTree: {rgb(0x040605), rgb(0x0a0e0a), rgb(0x121a10), rgb(0x1c2714), rgb(0x283618), rgb(0x3a4a1c), rgb(0x5e6a2c)},
	world.Oak:        {rgb(0x070504), rgb(0x120d08), rgb(0x1e160c), rgb(0x2c2010), rgb(0x3e2e14), rgb(0x56421a), rgb(0x7e6428)},
	world.Willow:     {rgb(0x060806), rgb(0x10140e), rgb(0x1a2016), rgb(0x272e1e), rgb(0x373f26), rgb(0x4c5530), rgb(0x707a48)},
	world.Yew:        {rgb(0x020404), rgb(0x060b0a), rgb(0x0b1311), rgb(0x111c18), rgb(0x182820), rgb(0x223626), rgb(0x384e34)},
}

var (
	barkTones     = tones{rgb(0x040303), rgb(0x0d0a09), rgb(0x18120f), rgb(0x261d17), rgb(0x382a20), rgb(0x4e3c2c)}
	deadBarkTones = tones{rgb(0x080707), rgb(0x141312), rgb(0x22201e), rgb(0x35322e), rgb(0x4e4a44), rgb(0x76706a)}
	// woodTones are the pale wood a cut shows.
	woodTones = tones{rgb(0x3e2a1a), rgb(0x6e5032), rgb(0x9c7a4c)}
	// shadowColor is the shade a canopy or the hero casts on the ground, laid at shadowAlpha.
	shadowColor = rgb(0x010102)
)

const shadowAlpha = 0.6
