// Package spawn resolves valid archer foot positions on the idle-king map.
package spawn

import (
	"math"
	"math/rand/v2"

	"github.com/AlviDervishaj/idle-king/objects/lava"
	"github.com/AlviDervishaj/idle-king/world"
)

const (
	minSeparationPx    = 28.0
	mapEdgeMarginTiles = 2
	centerBiasPx       = 200.0
)

// statueTiles are (col, row) where decoration GID 305 sits (idle-king-map.tmx "decoration" layer).
var statueTiles = []struct{ Col, Row int}{
	{13, 0}, {21, 2}, {28, 3}, {19, 6}, {13, 10}, {31, 12}, {26, 14}, {13, 15},
}

// summonerTileBounds is the inclusive tile rectangle of non-zero Summoner layer tiles, expanded by 1 tile.
var summonerTileBounds = struct{ Col0, Row0, Col1, Row1 int }{4, 0, 8, 4}

type tileCoord struct {
	col, row int
}

// ArcherSpawner precomputes weighted candidate tiles for a given lava mask.
type ArcherSpawner struct {
	lava           *lava.LavaMask
	candidates     []tileCoord
	weights        []float64
	weightSum      float64
}

// NewArcherSpawner builds spawn data for the given lava mask (typically lava.Default()).
func NewArcherSpawner(l *lava.LavaMask) *ArcherSpawner {
	s := &ArcherSpawner{lava: l}
	s.candidates, s.weights = s.computeCandidates()
	for _, w := range s.weights {
		s.weightSum += w
	}
	return s
}

func mapCenterPx() (float64, float64) {
	return float64(world.MapWidthPx) / 2, float64(world.MapHeightPx) / 2
}

func spawnWeightTowardCenter(wx, wy float64) float64 {
	cx, cy := mapCenterPx()
	dx, dy := wx-cx, wy-cy
	d2 := dx*dx + dy*dy
	return 1.0 / (1.0+d2/(centerBiasPx*centerBiasPx))
}

func (s *ArcherSpawner) blockedMask() [][]bool {
	b := make([][]bool, world.HeightTiles)
	for r := range b {
		b[r] = make([]bool, world.WidthTiles)
	}
	for row := summonerTileBounds.Row0; row <= summonerTileBounds.Row1 && row < world.HeightTiles; row++ {
		for col := summonerTileBounds.Col0; col <= summonerTileBounds.Col1 && col < world.WidthTiles; col++ {
			if world.InBounds(row, col) {
				b[row][col] = true
			}
		}
	}
	for _, st := range statueTiles {
		for dr := -1; dr <= 1; dr++ {
			for dc := -1; dc <= 1; dc++ {
				row, col := st.Row+dr, st.Col+dc
				if world.InBounds(row, col) {
					b[row][col] = true
				}
			}
		}
	}
	return b
}

func inMapEdgeMargin(col, row int) bool {
	return col >= mapEdgeMarginTiles &&
		col < world.WidthTiles-mapEdgeMarginTiles &&
		row >= mapEdgeMarginTiles &&
		row < world.HeightTiles-mapEdgeMarginTiles
}

func (s *ArcherSpawner) computeCandidates() ([]tileCoord, []float64) {
	if s.lava == nil {
		return nil, nil
	}
	blocked := s.blockedMask()
	var coords []tileCoord
	var weights []float64
	for row := range world.HeightTiles {
		lavaStart := s.lava.LeftmostLavaCol(row)
		for col := 0; col < lavaStart; col++ {
			if !inMapEdgeMargin(col, row) {
				continue
			}
			if s.lava.IsLava(row, col) || blocked[row][col] {
				continue
			}
			wx, wy := archerFeetWorld(col, row)
			coords = append(coords, tileCoord{col: col, row: row})
			weights = append(weights, spawnWeightTowardCenter(wx, wy))
		}
	}
	return coords, weights
}

// archerFeetWorld returns world pixel position (x = horizontal center, y = feet near tile bottom).
func archerFeetWorld(col, row int) (x, y float64) {
	x = float64(col*world.TileSize + world.TileSize/2)
	y = float64((row+1)*world.TileSize - 2)
	return x, y
}

func feetConflict(wx, wy float64, existing [][2]float64) bool {
	for _, p := range existing {
		dx, dy := wx-p[0], wy-p[1]
		if math.Hypot(dx, dy) < minSeparationPx {
			return true
		}
	}
	return false
}

// RandomArcher picks a spawn west of lava, biased toward map center, respecting spacing from existing feet.
func (s *ArcherSpawner) RandomArcher(rng *rand.Rand, existingFeet [][2]float64) (x, y float64, ok bool) {
	if len(s.candidates) == 0 {
		return 0, 0, false
	}
	var validIdx []int
	for i := range s.candidates {
		t := s.candidates[i]
		wx, wy := archerFeetWorld(t.col, t.row)
		if feetConflict(wx, wy, existingFeet) {
			continue
		}
		validIdx = append(validIdx, i)
	}
	if len(validIdx) == 0 {
		return 0, 0, false
	}
	var sum float64
	for _, i := range validIdx {
		sum += s.weights[i]
	}
	if sum <= 0 {
		i := validIdx[rng.IntN(len(validIdx))]
		wx, wy := archerFeetWorld(s.candidates[i].col, s.candidates[i].row)
		return wx, wy, true
	}
	r := rng.Float64() * sum
	for _, i := range validIdx {
		r -= s.weights[i]
		if r <= 0 {
			wx, wy := archerFeetWorld(s.candidates[i].col, s.candidates[i].row)
			return wx, wy, true
		}
	}
	last := validIdx[len(validIdx)-1]
	wx, wy := archerFeetWorld(s.candidates[last].col, s.candidates[last].row)
	return wx, wy, true
}

// CandidateCount returns how many precomputed spawn tiles exist (for tests).
func (s *ArcherSpawner) CandidateCount() int { return len(s.candidates) }
