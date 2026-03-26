// Package world holds fixed map grid dimensions and pure coordinate helpers.
package world

import "math"

// Viewport is the fixed 2:1 logical resolution (width = 2 × height).
const (
	ViewportWidth  = 1440
	ViewportHeight = 720
)

const (
	WidthTiles  = 32
	HeightTiles = 16
	TileSize    = 32
)

const (
	MapWidthPx  = WidthTiles * TileSize
	MapHeightPx = HeightTiles * TileSize
)

// InBounds reports whether (row, col) lies inside the tile grid.
func InBounds(row, col int) bool {
	return row >= 0 && row < HeightTiles && col >= 0 && col < WidthTiles
}

// WorldToTile converts world pixel coordinates (top-left origin) to tile indices (row, col).
func WorldToTile(px, py float64) (row, col int) {
	col = int(math.Floor(px / float64(TileSize)))
	row = int(math.Floor(py / float64(TileSize)))
	return row, col
}
