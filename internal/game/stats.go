package game

// statsState tracks lifetime and per-run statistics. All fields that need persisting
// are mirrored into SaveData. Fields updated inside updateCombat are written while
// combatMu.Lock() is already held, so no additional locking is needed.
type statsState struct {
	totalKills      int64
	killsByVariant  [4]int64
	totalGoldEarned float64
	totalWaves      int64
	totalBossKills  int64
	maxGoldPerSec   float64
	bestBossGold    float64
	lifetimePlaySec float64
}
