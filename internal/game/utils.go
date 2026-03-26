package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

func clamp(v, lo, hi float64) float64 {
	return max(lo, min(v, hi))
}

func cursorIn(r image.Rectangle) bool {
	x, y := ebiten.CursorPosition()
	return image.Pt(x, y).In(r)
}
