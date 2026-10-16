package main

import "math"

const treeSpacing = 4.7

type woodlandTree struct {
	x, z, height, radius, trunk float64
	variation                   float64
}

func (f *forest) cover(x, z float64) float64 {
	// Broad patches blend from grassy clearings through loose to closed canopy.
	cover := f.noise(x/24, z/24, 91)
	return math.Min(cover, 0.30+math.Hypot(x, z)*0.015)
}

func (f *forest) river(x, z float64) (distance, width float64) {
	distance = math.Inf(1)
	band := int(math.Floor((x + 30) / 110))
	for i := band - 1; i <= band+1; i++ {
		phase := f.random(i, 0, 72) * 6.28
		center := float64(i)*110 + 28 + math.Sin(z*0.038+phase)*12 + math.Sin(z*0.11+phase)*3
		if d := math.Abs(x - center); d < distance {
			distance = d
			width = 1.7 + 0.65*math.Sin(z*0.047+phase)
		}
	}
	return distance, width
}

func (f *forest) region() string {
	distance, width := f.river(f.x, f.z)
	if distance < width {
		return "river / wading"
	}
	if distance < width+1.5 {
		return "riverbank reeds"
	}
	cover := f.cover(f.x, f.z)
	if cover < 0.38 {
		return "tall-grass clearing"
	}
	if cover > 0.60 {
		return "deep woods"
	}
	return "sunlit woodland"
}

func (f *forest) treeAt(gx, gz int) (woodlandTree, bool) {
	x := (float64(gx) + 0.15 + f.random(gx, gz, 1)*0.7) * treeSpacing
	z := (float64(gz) + 0.15 + f.random(gx, gz, 2)*0.7) * treeSpacing
	cover := f.cover(x, z)
	probability := math.Min(0.97, 0.3+(cover-0.3)*1.8)
	if cover < 0.38 {
		probability = 0.025
	}
	distance, width := f.river(x, z)
	if f.random(gx, gz, 3) > probability || distance < width+1.0 || math.Hypot(x, z) < 3.5 {
		return woodlandTree{}, false
	}
	return woodlandTree{x, z, 4.5 + f.random(gx, gz, 4)*3, 2.0 + f.random(gx, gz, 5)*1.35, 0.18 + f.random(gx, gz, 6)*0.14, f.random(gx, gz, 7)}, true
}
