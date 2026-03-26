package spawn

import (
	"math/rand/v2"

	"github.com/AlviDervishaj/idle-king/objects/lava"
	"github.com/AlviDervishaj/idle-king/world"
)

// PickEnemyFeet chooses a spawn on the east (right) side of the map, strictly to the right of
// the lava band for that row, on non-lava grass.
func PickEnemyFeet(rng *rand.Rand, m *lava.LavaMask) (x, y float64, ok bool) {
	if m == nil {
		return 0, 0, false
	}
	for range 96 {
		row := 1 + rng.IntN(world.HeightTiles-2)
		rl := m.RightmostLavaCol(row)
		if rl < 0 {
			continue
		}
		// Columns strictly east of lava with a 3-tile gap: rl+4 .. WidthTiles-1
		minCol := rl + 4
		if minCol >= world.WidthTiles {
			continue
		}
		col := minCol + rng.IntN(world.WidthTiles-minCol)
		if m.IsLava(row, col) {
			continue
		}
		x = float64(col*world.TileSize + world.TileSize/2)
		y = float64((row+1)*world.TileSize - 2)
		return x, y, true
	}
	return 0, 0, false
}
