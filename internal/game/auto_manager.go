package game

import "math"

const (
	autoScribeBaseCost     = 5000  // gold to purchase Auto-Scribe for the first time
	autoScribeTickBaseSec  = 8.0   // seconds between auto-hire attempts at rank 0
	autoScribeThresholdMul = 3.0   // auto-hires when gold >= hireCost * this
)

// autoManagerState tracks the idle auto-hiring system.
type autoManagerState struct {
	purchased bool
	rank      int
	accum     float64 // time since last auto-hire attempt
}

// updateAutoManager checks whether the auto-scribe should hire another archer.
// Uses gameplayDT so it benefits from the speed multiplier at high speeds.
func (g *Game) updateAutoManager(dt float64) {
	if !g.autoMgr.purchased {
		return
	}
	speedRanks := g.extraUpgradeRank[extraAutoScribeSpeed]
	tickSec := autoScribeTickBaseSec * math.Pow(0.80, float64(g.autoMgr.rank+speedRanks))
	g.autoMgr.accum += dt
	if g.autoMgr.accum < tickSec {
		return
	}
	g.autoMgr.accum = 0

	discountRanks := g.extraUpgradeRank[extraAutoScribeDiscount]
	thresholdMul := autoScribeThresholdMul * math.Pow(0.90, float64(discountRanks))
	cost := g.selectedArcherHireCost()
	if cost <= 0 || g.gold < float64(cost)*thresholdMul {
		return
	}
	g.hireOneArcher()
}

// hireOneArcher performs a single archer hire at the current tier.
// It is the shared implementation used by both manual clicks and auto-manager.
func (g *Game) hireOneArcher() {
	tier := g.metaUpgradeRank[metaArcherTier]
	if tier < 0 || tier >= len(g.archerTierRuntimes) {
		return
	}
	tr := &g.archerTierRuntimes[tier]
	cost := tr.hireCost
	if g.gold < float64(cost) {
		return
	}
	wx, wy, ok := g.archerSpawn.RandomArcher(g.rng, g.existingArcherFeet())
	if !ok {
		return
	}
	g.gold -= float64(cost)
	clone := newGIFAnimShared(tr.idleFrames, tr.idleDelays, true)
	if clone != nil {
		clone.jumpToFrame(g.rng.IntN(len(tr.idleFrames)))
	}
	g.worldArchers = append(g.worldArchers, worldArcher{
		idleAnim: clone,
		tierIdx:  tier,
		x:        wx,
		y:        wy,
	})
}
