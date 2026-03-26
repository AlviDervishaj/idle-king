package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// ArcherTierAssets is idle + attack GIF bytes for one archer quality tier.
type ArcherTierAssets struct {
	IdleGIF, AttackGIF []byte
}

type archerTierRuntime struct {
	name         string
	idleFrames   []*ebiten.Image
	idleDelays   []int
	attackFrames []*ebiten.Image
	attackDelays []int
	attackBytes  []byte // copy for hire preview / one-shot anim
	damageMul    float64
	hireCost     int
	arrowFlightFrames []*ebiten.Image
}

const archerTierCount = 6

var ArcherTierNames = []string{
	"Common",
	"Uncommon",
	"Rare",
	"Ancient",
	"Epic",
	"Legendary",
}

// hireGoldPerTier is the gold to hire one archer of that tier (steep ramp).
var hireGoldPerTier = []int{15, 220, 950, 3500, 12000, 32000}

// damageMulPerTier scales arrowDamage for that tier (same fire cadence for everyone — this is the main DPS lever).
// Kept steep so high tiers are clearly worth the gold vs mixing low-tier squads.
var damageMulPerTier = []float64{1.0, 1.45, 1.95, 2.55, 3.25, 4.0}

func loadArcherTiers(assets []ArcherTierAssets, arrowFlightFrames [][]*ebiten.Image) ([]archerTierRuntime, error) {
	if len(assets) != archerTierCount {
		return nil, fmt.Errorf("game: want %d archer tiers, got %d", archerTierCount, len(assets))
	}
	if len(arrowFlightFrames) < archerTierCount {
		return nil, fmt.Errorf("game: not enough arrow frame sets for tiers")
	}
	out := make([]archerTierRuntime, 0, len(assets))
	for i := range assets {
		idleF, idleD, err := decodeGIF(assets[i].IdleGIF)
		if err != nil {
			return nil, fmt.Errorf("archer tier %d idle: %w", i, err)
		}
		if len(idleF) == 0 {
			return nil, fmt.Errorf("archer tier %d idle empty", i)
		}
		atkF, atkD, err := decodeGIF(assets[i].AttackGIF)
		if err != nil {
			return nil, fmt.Errorf("archer tier %d attack: %w", i, err)
		}
		if len(atkF) == 0 {
			return nil, fmt.Errorf("archer tier %d attack empty", i)
		}
		name := ArcherTierNames[i]
		hc := hireGoldPerTier[i]
		dm := damageMulPerTier[i]
		out = append(out, archerTierRuntime{
			name:              name,
			idleFrames:        idleF,
			idleDelays:        idleD,
			attackFrames:      atkF,
			attackDelays:      atkD,
			attackBytes:       append([]byte(nil), assets[i].AttackGIF...),
			damageMul:         dm,
			hireCost:          hc,
			arrowFlightFrames: arrowFlightFrames[i],
		})
	}
	return out, nil
}

func (g *Game) selectedArcherHireCost() int {
	t := g.metaUpgradeRank[metaArcherTier]
	if t < 0 || t >= len(g.archerTierRuntimes) {
		return 0
	}
	return g.archerTierRuntimes[t].hireCost
}

func (g *Game) arrowDamageForTier(tierIdx int) float64 {
	if tierIdx < 0 || tierIdx >= len(g.archerTierRuntimes) {
		return arrowDamage
	}
	return arrowDamage * g.archerTierRuntimes[tierIdx].damageMul
}
