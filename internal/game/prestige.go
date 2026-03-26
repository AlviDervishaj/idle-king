package game

import (
	"fmt"
	"math"
)

// Soul upgrades are purchased with Souls (prestige currency) and persist across resets.
const (
	soulUpgradeGoldMul    = iota // +10% gold mul per rank; cost: 1 → 3 → 9 Souls
	soulUpgradeOffline           // +10% offline efficiency per rank (base 50%)
	soulUpgradeStartGold         // start each run with bonus gold (10% of last run peak per rank)
	soulUpgradeSoulBonus         // +10% souls earned per prestige per rank
	soulUpgradeAutoUnlock        // auto-manager available from run start (rank 1 = unlocked)
	soulUpgradeCount      = 5
)

var soulUpgradeTitles = [soulUpgradeCount]string{
	"Soul Tithe",
	"Spirit Rest",
	"Haunted Coffers",
	"Soul Resonance",
	"Phantom Scribe",
}

var soulUpgradeDesc = [soulUpgradeCount]string{
	"+10% gold multiplier",
	"+10% offline income",
	"Bonus starting gold",
	"+10% soul gain",
	"Auto-hire from start",
}

// soulUpgradeNextCost returns the Soul cost for the next rank, or 0 if maxed.
// Exponential: cost_n = 1 * 3^rank (1 → 3 → 9 → 27 ...).
func soulUpgradeNextCost(id, rank int) float64 {
	if id < 0 || id >= soulUpgradeCount {
		return 0
	}
	const maxRank = 10
	if rank >= maxRank {
		return 0
	}
	if id == soulUpgradeAutoUnlock && rank >= 1 {
		return 0 // one-time purchase
	}
	base := []float64{1, 1, 2, 1, 3}[id]
	if rank == 0 {
		return base
	}
	return base * math.Pow(3, float64(rank))
}

// prestigeData is the persistent prestige layer (survives all run resets).
type prestigeData struct {
	rank           int
	totalPrestiges int
	souls          float64
	soulRanks      [soulUpgradeCount]int
	// peakGold is the highest gold seen in the last run (for soulUpgradeStartGold).
	peakGold float64
}

// soulsOnPrestige calculates how many Souls the player earns for prestiging now.
// Formula: floor(sqrt(totalGoldEarned / 1000)). Rewards throughput, not hoarding.
func (g *Game) soulsOnPrestige() float64 {
	const soulThreshold = 1000.0
	bonus := 1.0 + float64(g.prestige.soulRanks[soulUpgradeSoulBonus])*0.10 + g.achievementMuls.soulBonus
	raw := math.Floor(math.Sqrt(g.stats.totalGoldEarned/soulThreshold)) * bonus
	return raw
}

// prestigeGoldMul returns the permanent gold multiplier from prestige rank and soul upgrades.
func (g *Game) prestigeGoldMul() float64 {
	return 1.0 +
		float64(g.prestige.rank)*0.15 +
		float64(g.prestige.soulRanks[soulUpgradeGoldMul])*0.10
}

// offlineEfficiency returns the fraction of online income granted for offline time.
func (g *Game) offlineEfficiency() float64 {
	const base = 0.5
	bonus := float64(g.prestige.soulRanks[soulUpgradeOffline]) * 0.10
	eff := base + bonus
	if eff > 0.9 {
		eff = 0.9
	}
	return eff
}

// startingGoldBonus returns the gold the player begins the run with (from soul upgrade).
func (g *Game) startingGoldBonus() float64 {
	if g.prestige.soulRanks[soulUpgradeStartGold] == 0 {
		return 0
	}
	return g.prestige.peakGold * float64(g.prestige.soulRanks[soulUpgradeStartGold]) * 0.10
}

// doPrestige performs a prestige reset. Caller must NOT hold combatMu.
func (g *Game) doPrestige() {
	souls := g.soulsOnPrestige()
	if souls < 1 {
		return
	}

	// Update prestige-layer state (persists).
	g.prestige.rank++
	g.prestige.totalPrestiges++
	g.prestige.souls += souls

	// Reset per-run economy.
	g.gold = g.startingGoldBonus()
	g.stats.totalGoldEarned = 0
	g.prestige.peakGold = 0

	// Reset upgrades (run-scoped upgrades only; prestige upgrades persist).
	g.metaUpgradeRank = [metaUpgradeCount]int{}
	// Keep prestige-category extra upgrades (extraSoulMultiplier, extraOfflineBonus, extraStartingGold
	// are not in extraUpgradeCount, so all extra upgrades are run-scoped and can be zeroed).
	g.extraUpgradeRank = [extraUpgradeCount]int{}

	// Reset archers and enemy roster.
	g.worldArchers = g.worldArchers[:0]
	g.enemyTypesUnlocked = 1

	// Reset auto-manager unless soul upgrade grants it.
	if g.prestige.soulRanks[soulUpgradeAutoUnlock] == 0 {
		g.autoMgr.purchased = false
		g.autoMgr.rank = 0
	}
	g.autoMgr.accum = 0

	// Reset run-scoped wave counter and boss state.
	// stats.totalWaves is a lifetime counter and must NOT be reset here.
	g.waveCounter = 0
	g.boss.nextBossWave = bossWaveInterval
	g.boss.activeBossID = 0

	// Reset combat state (requires lock).
	g.combatMu.Lock()
	g.enemies = g.enemies[:0]
	g.arrows = g.arrows[:0]
	g.spawnAccum = enemySpawnInterval * 0.45
	g.fireAccum = 0
	g.playTime = 0
	g.combatMu.Unlock()

	// Update shop preview to first tier.
	g.refreshShopArcherPreview()

	// Show notification.
	g.offlineMessage = formatPrestigeMsg(g.prestige.rank, souls)
	g.offlineMessageTimer = 7.0
	g.prestigeConfirmPending = false

	_ = g.Save()
}

func formatPrestigeMsg(rank int, souls float64) string {
	return fmt.Sprintf("Prestige %d! Earned %.0f Souls — run restarted.", rank, souls)
}
