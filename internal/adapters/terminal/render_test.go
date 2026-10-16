package terminal

import (
	"fmt"
	"testing"

	"dither-cli/internal/forest"
)

func BenchmarkForestFrame(b *testing.B) {
	for _, size := range [][2]int{{100, 30}, {180, 50}, {220, 90}} {
		b.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(b *testing.B) {
			r := newRenderer(forest.New(7), "L’Après-Règne")
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r.frame(size[0], size[1], float64(i%120)/30)
			}
		})
	}
}
