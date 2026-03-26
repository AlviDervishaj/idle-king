package game

// achievementMuls accumulates permanent multipliers from earned achievements.
// All fields are additive (gold *= 1+goldBonus, damage *= 1+damageMul, etc.).
type achievementMuls struct {
	goldBonus float64
	damageMul float64
	soulBonus float64
}

// achievementToast is a brief notification shown when an achievement is earned.
type achievementToast struct {
	title      string
	desc       string
	rewardDesc string
	timer      float64
}

// achievement defines a milestone and its optional permanent reward.
type achievement struct {
	id         int
	title      string
	desc       string
	trigger    func(g *Game) bool
	reward     func(g *Game)
	rewardDesc string
}

// allAchievements is the authoritative list. Triggers are pure reads (no locks needed if
// called after updateCombat returns, which is the required call site).
var allAchievements = []achievement{
	// Kill milestones
	{0, "First Blood", "Kill your first enemy",
		func(g *Game) bool { return g.stats.totalKills >= 1 }, nil, ""},
	{1, "Century", "Kill 100 enemies",
		func(g *Game) bool { return g.stats.totalKills >= 100 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.05 }, "+5% gold"},
	{2, "Legion", "Kill 10,000 enemies",
		func(g *Game) bool { return g.stats.totalKills >= 10_000 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.10 }, "+10% gold"},
	{3, "Endless War", "Kill 1,000,000 enemies",
		func(g *Game) bool { return g.stats.totalKills >= 1_000_000 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.20 }, "+20% gold"},

	// Gold milestones
	{4, "Pocket Change", "Earn 1,000 total gold",
		func(g *Game) bool { return g.stats.totalGoldEarned >= 1_000 }, nil, ""},
	{5, "Treasure Hoard", "Earn 1,000,000 total gold",
		func(g *Game) bool { return g.stats.totalGoldEarned >= 1_000_000 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.10 }, "+10% gold"},
	{6, "Gilded Kingdom", "Earn 1,000,000,000 total gold",
		func(g *Game) bool { return g.stats.totalGoldEarned >= 1_000_000_000 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.25 }, "+25% gold"},

	// Archer milestones
	{7, "Recruit", "Hire your first archer",
		func(g *Game) bool { return len(g.worldArchers) >= 1 }, nil, ""},
	{8, "Company", "Have 10 archers at once",
		func(g *Game) bool { return len(g.worldArchers) >= 10 },
		func(g *Game) { g.achievementMuls.damageMul += 0.05 }, "+5% damage"},
	{9, "Army", "Have 25 archers at once",
		func(g *Game) bool { return len(g.worldArchers) >= 25 },
		func(g *Game) { g.achievementMuls.damageMul += 0.10 }, "+10% damage"},
	{10, "Legendary Unit", "Upgrade archers to Legendary tier",
		func(g *Game) bool { return g.metaUpgradeRank[metaArcherTier] >= archerTierCount-1 },
		func(g *Game) { g.achievementMuls.damageMul += 0.15 }, "+15% damage"},

	// Wave milestones
	{11, "First Wave", "Survive 10 waves",
		func(g *Game) bool { return g.stats.totalWaves >= 10 }, nil, ""},
	{12, "Veteran", "Survive 100 waves",
		func(g *Game) bool { return g.stats.totalWaves >= 100 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.05 }, "+5% gold"},
	{13, "Eternal Guard", "Survive 1,000 waves",
		func(g *Game) bool { return g.stats.totalWaves >= 1_000 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.15 }, "+15% gold"},

	// Boss kills
	{14, "Boss Slayer", "Kill your first boss",
		func(g *Game) bool { return g.stats.totalBossKills >= 1 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.10 }, "+10% gold"},
	{15, "Titan Bane", "Kill 10 bosses",
		func(g *Game) bool { return g.stats.totalBossKills >= 10 },
		func(g *Game) { g.achievementMuls.damageMul += 0.10 }, "+10% damage"},

	// Prestige
	{16, "The Cycle Begins", "Prestige for the first time",
		func(g *Game) bool { return g.prestige.totalPrestiges >= 1 }, nil, ""},
	{17, "Ascendant", "Prestige 5 times",
		func(g *Game) bool { return g.prestige.totalPrestiges >= 5 },
		func(g *Game) { g.achievementMuls.soulBonus += 0.10 }, "+10% souls"},
	{18, "Eternal Cycle", "Prestige 25 times",
		func(g *Game) bool { return g.prestige.totalPrestiges >= 25 },
		func(g *Game) { g.achievementMuls.soulBonus += 0.25 }, "+25% souls"},

	// Automation & speed
	{19, "Efficiency", "Purchase the Auto-Scribe",
		func(g *Game) bool { return g.autoMgr.purchased }, nil, ""},
	{20, "Full Speed", "Enable 4x game speed",
		func(g *Game) bool { return g.speedIndex == 2 }, nil, ""},

	// Enemy variety
	{21, "Menagerie", "Unlock all enemy types",
		func(g *Game) bool { return g.enemyTypesUnlocked >= len(g.enemyVariants) }, nil, ""},

	// Income
	{22, "Cash Flow", "Reach 100 gold/sec",
		func(g *Game) bool { return g.gpsSample >= 100 }, nil, ""},
	{23, "Mint", "Reach 10,000 gold/sec",
		func(g *Game) bool { return g.gpsSample >= 10_000 },
		func(g *Game) { g.achievementMuls.goldBonus += 0.10 }, "+10% gold"},

	// Time played
	{24, "Idle King", "Play for 1 hour total",
		func(g *Game) bool { return g.stats.lifetimePlaySec >= 3_600 }, nil, ""},

	// Boss reward
	{25, "Overkill", "Earn 1,000 gold from a single boss",
		func(g *Game) bool { return g.stats.bestBossGold >= 1_000 },
		func(g *Game) { g.achievementMuls.damageMul += 0.20 }, "+20% damage"},
}

// achievementCount is the total number of achievements (used for []bool sizing).
const achievementCount = 26

// checkAchievements checks all unearned achievements and fires any that are now met.
// Must be called OUTSIDE of combatMu (triggers are pure reads of scalar counters).
func (g *Game) checkAchievements() {
	for i := range allAchievements {
		if i >= len(g.achievementsEarned) || g.achievementsEarned[i] {
			continue
		}
		if allAchievements[i].trigger(g) {
			g.achievementsEarned[i] = true
			if allAchievements[i].reward != nil {
				allAchievements[i].reward(g)
			}
			g.achievementToasts = append(g.achievementToasts, achievementToast{
				title:      allAchievements[i].title,
				desc:       allAchievements[i].desc,
				rewardDesc: allAchievements[i].rewardDesc,
				timer:      4.0,
			})
		}
	}
}

// rebuildAchievementMuls re-applies all earned rewards. Called once after loading a save.
// Rewards use += so calling them once per load is safe (idempotent via fresh zero state).
func (g *Game) rebuildAchievementMuls() {
	g.achievementMuls = achievementMuls{}
	for i, earned := range g.achievementsEarned {
		if !earned || i >= len(allAchievements) {
			continue
		}
		if allAchievements[i].reward != nil {
			allAchievements[i].reward(g)
		}
	}
}

// achievementsEarnedCount returns how many achievements have been completed.
func (g *Game) achievementsEarnedCount() int {
	n := 0
	for _, v := range g.achievementsEarned {
		if v {
			n++
		}
	}
	return n
}

