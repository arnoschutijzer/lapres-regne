package window

import "image/color"

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
