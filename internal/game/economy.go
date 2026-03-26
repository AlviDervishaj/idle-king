package game

// deathCause records how an enemy died for loot rules (environmental = no loot).
type deathCause uint8

const (
	deathCauseEnvironmental deathCause = iota // lava or breach
	deathCausePlayerArrow
	deathCausePlayerLaser
)

const (
	scavengerBonusPerRank = 0.15 // 15% gold multiplier per Scavenger rank

	enemyUnlockBaseCost = 42
	enemyUnlockStep     = 28 // added for each roster tier already unlocked
)

const (
	metaScavenger = iota
	metaSpawnWaveSize
	metaArcherTier
	metaLaserDamage
	metaLaserRadius
	metaUpgradeCount
)

var metaUpgradeTitles = []string{
	"Scavenger crew",
	"Enemy Density",
	"Archer Training",
	"Beam damage",
	"Beam radius",
}

var metaUpgradeDesc = []string{
	"+15% kill loot",
	"Enemies per wave",
	"Upgrade all units",
	"Laser DPS",
	"Laser area",
}

func metaUpgradeBaseCost(id int) int {
	switch id {
	case metaScavenger:
		return 32
	case metaSpawnWaveSize:
		return 45
	case metaArcherTier:
		return 380
	case metaLaserDamage:
		return 65
	case metaLaserRadius:
		return 72
	default:
		return 32
	}
}

func (g *Game) metaUpgradeNextCost(id int) int {
	if id < 0 || id >= metaUpgradeCount {
		return 0
	}
	r := g.metaUpgradeRank[id]
	if id == metaArcherTier {
		if r >= archerTierCount-1 {
			return 0
		}
		// Archer training is a major milestone: steep multiplier.
		costs := []int{380, 1850, 7200, 24000, 85000}
		if r < len(costs) {
			return costs[r]
		}
		return 0
	}
	base := metaUpgradeBaseCost(id)
	return base + r*22
}

// dropKillLoot adds kill gold (with all bonuses) and returns the amount granted.
// goldMul is an extra multiplier applied on top of all standard bonuses (e.g. boss
// scaling). Pass 1.0 for normal enemies. Also feeds GPS accumulator and lifetime
// stats. Called from beginEnemyDeath while combatMu.Lock() is held.
func (g *Game) dropKillLoot(vi int, goldMul float64) float64 {
	base := 1.0
	if vi >= 0 && vi < len(g.enemyVariants) {
		base = g.enemyVariants[vi].baseGoldReward
	}
	n := base *
		(1.0 + scavengerBonusPerRank*float64(g.metaUpgradeRank[metaScavenger])) *
		g.extraGoldMul() *
		g.prestigeGoldMul() *
		goldMul
	if n < 0.1 {
		n = 0.1
	}
	g.gold += n
	g.stats.totalGoldEarned += n
	g.gpsAccum += n
	return n
}

// enemyUnlockNextCost returns gold to add the next enemy type to the spawn pool, or 0 if all are unlocked.
func (g *Game) enemyUnlockNextCost() int {
	n := len(g.enemyVariants)
	if n <= 1 || g.enemyTypesUnlocked >= n {
		return 0
	}
	return enemyUnlockBaseCost + (g.enemyTypesUnlocked-1)*enemyUnlockStep
}
