package window

import (
	"image/color"

	"lapresregne/internal/world"
)

// The window's palette is picked by hand, after Hyper Light Drifter's flat fields of teal, moss
// and violet, sunk into Blasphemous's darkness: every material runs through a few tones from a
// blue-black shadow to one bright accent, darkest first.
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
	voidColor = rgb(0x07060c)

	grassTones = tones{rgb(0x081215), rgb(0x0d1d20), rgb(0x132b2a), rgb(0x1b3e36), rgb(0x285643), rgb(0x427a50)}
	// bankTones are the trodden mud along a river.
	bankTones  = tones{rgb(0x120f14), rgb(0x1e1a1c), rgb(0x2c2622)}
	waterTones = tones{rgb(0x060f22), rgb(0x0b1e3a), rgb(0x123257), rgb(0x1d5379), rgb(0x6fbcc4)}
	pathTones  = tones{rgb(0x1a1216), rgb(0x2a1e1e), rgb(0x3d2b26), rgb(0x58402f)}
	stoneTones = tones{rgb(0x121019), rgb(0x1f1c29), rgb(0x2f2b3b), rgb(0x463f55), rgb(0x6c6280)}
	boardTones = tones{rgb(0x150d0e), rgb(0x2a1915), rgb(0x45291d), rgb(0x643c25)}

	// Wildflowers are the few bright accents in the grass: Blasphemous's crimson and gold, and a
	// pale violet.
	crimsonTones = tones{rgb(0x5c0f1e), rgb(0xa3243a), rgb(0xe0566a)}
	goldTones    = tones{rgb(0x6b4414), rgb(0xc8902e), rgb(0xf2cf6b)}
	paleTones    = tones{rgb(0x4a4260), rgb(0x9a90b8), rgb(0xd8d0ec)}
)

// Each kind of tree has its own leaves, from the shadow deep in its crown to the light on its top.
var canopyTones = map[world.TreeKind]tones{
	world.NormalTree: {rgb(0x06111a), rgb(0x0b1d24), rgb(0x112d2e), rgb(0x184036), rgb(0x22573f), rgb(0x357449), rgb(0x62a058)},
	world.Oak:        {rgb(0x110e0c), rgb(0x1d1a12), rgb(0x2c2915), rgb(0x423c1b), rgb(0x5c5421), rgb(0x80742b), rgb(0xbca448)},
	world.Willow:     {rgb(0x0b1516), rgb(0x142429), rgb(0x20362e), rgb(0x314a37), rgb(0x466343), rgb(0x668654), rgb(0xa2bc76)},
	world.Yew:        {rgb(0x040c15), rgb(0x071721), rgb(0x0c242b), rgb(0x123435), rgb(0x1a473f), rgb(0x28604a), rgb(0x4a855a)},
}

var (
	barkTones     = tones{rgb(0x0d090d), rgb(0x1c1317), rgb(0x31201d), rgb(0x4a3124), rgb(0x684830)}
	deadBarkTones = tones{rgb(0x14111a), rgb(0x28222e), rgb(0x413946), rgb(0x675d6e), rgb(0x968c9e)}
	// woodTones are the pale wood a cut shows.
	woodTones = tones{rgb(0x5a3e28), rgb(0x9a7046), rgb(0xd0a468)}
	// shadowColor is the flat shade a canopy or the hero casts on the ground, laid at shadowAlpha.
	shadowColor = rgb(0x02040e)
)

const shadowAlpha = 0.5
