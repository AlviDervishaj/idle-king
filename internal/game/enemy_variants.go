package game

import (
	"errors"
	"fmt"

	"github.com/AlviDervishaj/idle-king/objects/atlas"
	"github.com/hajimehoshi/ebiten/v2"
)

// EnemyStripPair is PNG bytes for one enemy type (e.g. S_Walk.png + S_Death.png from a pack folder).
type EnemyStripPair struct {
	WalkPNG, DeathPNG []byte
}

type enemyVariant struct {
	walkFrames  []*ebiten.Image
	deathFrames []*ebiten.Image
	maxHP       float64
	// arrowTakenMul scales incoming archer arrow damage (later packs are armored vs arrows).
	arrowTakenMul float64
	// baseGoldReward is the amount of gold granted on death (before contract modifiers).
	baseGoldReward float64
	// mirrorX: pack 1 S_Walk faces east; -scale X faces them west (toward the breach).
	// Higher pack strips are already authored facing west, so mirroring would show them backward.
	mirrorX bool
}

// Baseline HP for pack index 0; higher packs use multipliers from enemyHPMultiplierForPackIndex.
const enemyMaxHPBase = 92.0

func enemyHPMultiplierForPackIndex(i int) float64 {
	// Stronger enemies from higher-numbered asset folders (extend when adding pack 5+).
	m := []float64{1.0, 1.22, 1.48, 1.78, 2.08}
	if i < 0 {
		return 1
	}
	if i >= len(m) {
		return m[len(m)-1]
	}
	return m[i]
}

func arrowResistForPackIndex(i int) float64 {
	// Multiplier on arrow damage taken (not laser/lava). Pack 0 baseline; tougher enemies shrug arrows.
	m := []float64{1.0, 0.78, 0.68, 0.58}
	if i < 0 {
		return 1
	}
	if i >= len(m) {
		return m[len(m)-1]
	}
	return m[i]
}

func goldRewardForPackIndex(i int) float64 {
	// Tier 1 (0): 1.0, Tier 2 (1): 1.3, Tier 3 (2): 1.6, Tier 4 (3): 1.9.
	m := []float64{1.0, 1.3, 1.6, 1.9}
	if i < 0 {
		return 1.0
	}
	if i >= len(m) {
		return m[len(m)-1]
	}
	return m[i]
}

func loadEnemyVariantsFromStrips(pairs []EnemyStripPair) ([]enemyVariant, error) {
	if len(pairs) == 0 {
		return nil, errors.New("game: no enemy strip pairs")
	}
	out := make([]enemyVariant, 0, len(pairs))
	for i := range pairs {
		walk, err := atlas.SplitHorizontalStrip(pairs[i].WalkPNG, enemyWalkFrameW, enemyWalkFrameH)
		if err != nil {
			return nil, fmt.Errorf("enemy walk strip %d: %w", i, err)
		}
		if len(walk) == 0 {
			return nil, fmt.Errorf("enemy walk strip %d empty", i)
		}
		death, err := atlas.SplitHorizontalStrip(pairs[i].DeathPNG, enemyWalkFrameW, enemyWalkFrameH)
		if err != nil {
			return nil, fmt.Errorf("enemy death strip %d: %w", i, err)
		}
		mul := enemyHPMultiplierForPackIndex(i)
		out = append(out, enemyVariant{
			walkFrames:     walk,
			deathFrames:    death,
			maxHP:          enemyMaxHPBase * mul,
			arrowTakenMul:  arrowResistForPackIndex(i),
			baseGoldReward: goldRewardForPackIndex(i),
			mirrorX:        i == 0,
		})
	}
	return out, nil
}
