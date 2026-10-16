package world

import "math"

const TreeSpacing = 4.7

type Tree struct {
	X, Z, Height, Radius, Trunk float64
	Variation                   float64
	Kind                        TreeKind
	// Stump marks a felled tree.
	Stump bool
	cell  cell
}

func (w *World) Cover(x, z float64) float64 {
	// Broad patches blend from grassy clearings through loose to closed canopy.
	cover := w.Noise(x/24, z/24, 91)
	return math.Min(cover, 0.30+math.Hypot(x, z)*0.015)
}

func (w *World) River(x, z float64) (distance, width float64) {
	band := int(math.Floor((x + 30) / 110))
	channels := [3]RiverChannel{w.riverChannel(band-1, z), w.riverChannel(band, z), w.riverChannel(band+1, z)}
	return NearestRiver(channels[:], x)
}

type RiverChannel struct{ center, width float64 }

func (w *World) riverChannel(band int, z float64) RiverChannel {
	phase := w.Random(band, 0, 72) * 6.28
	return RiverChannel{
		float64(band)*110 + 28 + math.Sin(z*0.038+phase)*12 + math.Sin(z*0.11+phase)*3,
		1.7 + 0.65*math.Sin(z*0.047+phase),
	}
}

func (w *World) RiverChannels(channels []RiverChannel, z, left, right float64) []RiverChannel {
	first, last := int(math.Floor((left+30)/110))-1, int(math.Floor((right+30)/110))+1
	for band := first; band <= last; band++ {
		channels = append(channels, w.riverChannel(band, z))
	}
	return channels
}

func NearestRiver(channels []RiverChannel, x float64) (distance, width float64) {
	distance = math.Inf(1)
	for _, channel := range channels {
		if d := math.Abs(x - channel.center); d < distance {
			distance = d
			width = channel.width
		}
	}
	return distance, width
}

func (w *World) Region() string {
	distance, width := w.River(w.hero.x, w.hero.z)
	if distance < width {
		return "river / wading"
	}
	if distance < width+1.5 {
		return "riverbank reeds"
	}
	cover := w.Cover(w.hero.x, w.hero.z)
	if cover < 0.38 {
		return "tall-grass clearing"
	}
	if cover > 0.60 {
		return "deep woods"
	}
	return "sunlit woodland"
}

func (w *World) TreeAt(gx, gz int) (Tree, bool) {
	x := (float64(gx) + 0.15 + w.Random(gx, gz, 1)*0.7) * TreeSpacing
	z := (float64(gz) + 0.15 + w.Random(gx, gz, 2)*0.7) * TreeSpacing
	cover := w.Cover(x, z)
	probability := math.Min(0.97, 0.3+(cover-0.3)*1.8)
	if cover < 0.38 {
		probability = 0.025
	}
	distance, width := w.River(x, z)
	if w.Random(gx, gz, 3) > probability || distance < width+1.0 || math.Hypot(x, z) < 3.5 {
		return Tree{}, false
	}
	kind := w.treeKind(gx, gz, x, z)
	height, radius := 4.5+w.Random(gx, gz, 4)*3, 2.0+w.Random(gx, gz, 5)*1.35
	// Oaks spread low and wide, willows broaden, and yews grow tall and narrow.
	switch kind {
	case Oak:
		height, radius = height*0.85, radius*1.2
	case Willow:
		height, radius = height*0.9, radius*1.15
	case Yew:
		height, radius = height*1.15, radius*0.85
	}
	return Tree{x, z, height, radius, 0.18 + w.Random(gx, gz, 6)*0.14, w.Random(gx, gz, 7), kind, w.felled[cell{gx, gz}] > 0, cell{gx, gz}}, true
}
