package game

import "math"

// Extra upgrades on top of the 5 meta-upgrades. Costs use exponential scaling.
const (
	// Economy category
	extraGoldBonus  = iota // +10% all gold per rank
	extraWaveValue         // higher-value variants spawn more (+5% weight per rank)
	extraLootRadius        // +5% bonus gold per rank (cosmetic radius increase)

	// Combat category
	extraArrowDamage    // +8% arrow damage per rank, stacks with tier multiplier
	extraArrowSpeed     // +15% arrow flight speed per rank
	extraArcherCooldown // -5% fire interval per rank (capped at -50%)
	extraLavaHeat       // +20% lava DPS per rank

	// Automation category (requires auto-manager purchase)
	extraAutoScribeSpeed    // auto-hire ticks 20% faster per rank
	extraAutoScribeDiscount // auto-hire gold threshold drops 10% per rank
	extraAutoWave           // auto-trigger boss waves when gold > bossWaveCostFactor

	extraUpgradeCount = 10
)

// extraUpgradeTitles are display names for the shop UI.
var extraUpgradeTitles = [extraUpgradeCount]string{
	"Gold Multiplier",
	"Wave Value",
	"Loot Radius",
	"Arrow Power",
	"Arrow Speed",
	"Archer Haste",
	"Lava Wrath",
	"Scribe Speed",
	"Scribe Discount",
	"Auto Wave",
}

// extraUpgradeDesc are the effect descriptions shown in the shop.
var extraUpgradeDesc = [extraUpgradeCount]string{
	"+10% kill gold",
	"+5% high-tier spawns",
	"+5% bonus gold",
	"+8% arrow damage",
	"+15% arrow speed",
	"-5% fire interval",
	"+20% lava DPS",
	"-20% hire interval",
	"-10% gold threshold",
	"Auto-trigger bosses",
}

// extraUpgradeBaseCost is the cost for rank 0→1 of each extra upgrade.
var extraUpgradeBaseCost = [extraUpgradeCount]int{
	85,  // extraGoldBonus
	120, // extraWaveValue
	60,  // extraLootRadius
	200, // extraArrowDamage
	180, // extraArrowSpeed
	300, // extraArcherCooldown
	150, // extraLavaHeat
	500, // extraAutoScribeSpeed
	450, // extraAutoScribeDiscount
	800, // extraAutoWave
}

// extraUpgradeNextCost returns the gold cost to buy the next rank, or 0 if maxed.
// Uses exponential scaling: cost_n = base * 1.55^rank.
func extraUpgradeNextCost(id, rank int) int {
	if id < 0 || id >= extraUpgradeCount {
		return 0
	}
	const maxRank = 20
	if rank >= maxRank {
		return 0
	}
	base := float64(extraUpgradeBaseCost[id])
	if rank == 0 {
		return int(base)
	}
	return int(base * math.Pow(1.55, float64(rank)))
}

// extraArrowDamageMul returns the combined arrow damage multiplier from extra upgrades.
func (g *Game) extraArrowDamageMul() float64 {
	return 1.0 + float64(g.extraUpgradeRank[extraArrowDamage])*0.08
}

// extraArrowSpeedMul returns the arrow flight speed multiplier from extra upgrades.
func (g *Game) extraArrowSpeedMul() float64 {
	return 1.0 + float64(g.extraUpgradeRank[extraArrowSpeed])*0.15
}

// extraFireIntervalMul returns the archer fire interval multiplier (< 1 = faster firing).
// Capped so the interval never drops below 50% of base.
func (g *Game) extraFireIntervalMul() float64 {
	reduction := float64(g.extraUpgradeRank[extraArcherCooldown]) * 0.05
	if reduction > 0.5 {
		reduction = 0.5
	}
	return 1.0 - reduction
}

// extraLavaDPSMul returns the lava damage multiplier from extra upgrades.
func (g *Game) extraLavaDPSMul() float64 {
	return 1.0 + float64(g.extraUpgradeRank[extraLavaHeat])*0.20
}

// extraGoldMul returns the additive gold bonus from extra economy upgrades (and achievements).
func (g *Game) extraGoldMul() float64 {
	return 1.0 +
		float64(g.extraUpgradeRank[extraGoldBonus])*0.10 +
		float64(g.extraUpgradeRank[extraLootRadius])*0.05 +
		g.achievementMuls.goldBonus
}
