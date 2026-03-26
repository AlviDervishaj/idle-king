package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const saveVersion = 1

// SaveData is the complete persistent representation of a game session.
// Fields use omitempty so old saves parse cleanly when new fields are added.
// Bump saveVersion and add a migration when removing or renaming fields.
type SaveData struct {
	Version int       `json:"v"`
	SavedAt time.Time `json:"saved_at"`

	// Economy
	Gold            float64 `json:"gold"`
	TotalGoldEarned float64 `json:"total_gold_earned,omitempty"`

	// Archers: position + tier only; animations are rebuilt from assets.
	Archers []savedArcher `json:"archers,omitempty"`

	// Upgrades
	MetaUpgradeRanks  [metaUpgradeCount]int  `json:"meta_ranks"`
	ExtraUpgradeRanks [extraUpgradeCount]int `json:"extra_ranks"`
	EnemyTypesUnlocked int                  `json:"enemy_unlocked"`

	// Prestige
	PrestigeRank     int                  `json:"prestige_rank,omitempty"`
	TotalPrestiges   int                  `json:"total_prestiges,omitempty"`
	Souls            float64              `json:"souls,omitempty"`
	SoulUpgradeRanks [soulUpgradeCount]int `json:"soul_ranks,omitempty"`
	PeakGold         float64              `json:"peak_gold,omitempty"`

	// Achievements
	AchievementsEarned []bool `json:"achievements,omitempty"`

	// Statistics
	TotalKills      int64    `json:"total_kills,omitempty"`
	KillsByVariant  [4]int64 `json:"kills_by_variant,omitempty"`
	TotalWaves      int64    `json:"total_waves,omitempty"`
	TotalBossKills  int64    `json:"total_boss_kills,omitempty"`
	MaxGoldPerSec   float64  `json:"max_gold_per_sec,omitempty"`
	BestBossGold    float64  `json:"best_boss_gold,omitempty"`
	LifetimePlaySec float64  `json:"lifetime_play_sec,omitempty"`

	// Wave / boss state
	WaveCounter int `json:"wave_counter,omitempty"`

	// Income snapshot for offline calculation
	GoldPerSecSnapshot float64 `json:"gps_snapshot,omitempty"`

	// Settings
	SpeedIndex int  `json:"speed_index,omitempty"`
	ShopOpen   bool `json:"shop_open,omitempty"`
	ActiveTab  int  `json:"active_tab,omitempty"`

	// Auto-manager
	AutoMgrPurchased bool `json:"auto_mgr_purchased,omitempty"`
	AutoMgrRank      int  `json:"auto_mgr_rank,omitempty"`
}

type savedArcher struct {
	TierIdx int     `json:"t"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}

// savePath returns the OS-appropriate save file path.
func savePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("save: cache dir: %w", err)
	}
	dir = filepath.Join(dir, "idle-king")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("save: mkdir: %w", err)
	}
	return filepath.Join(dir, "save.json"), nil
}

// Save serializes the game state to disk atomically (write to .tmp, then rename).
func (g *Game) Save() error {
	sd := g.toSaveData()
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return fmt.Errorf("save: marshal: %w", err)
	}
	path, err := savePath()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("save: write: %w", err)
	}
	// Remove the destination first so os.Rename succeeds on platforms (e.g. Windows)
	// that refuse to replace an existing file atomically.
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("save: rename: %w", err)
	}
	return nil
}

// Load reads the save file and applies it to g. A missing or corrupt save is treated
// as a fresh game (returns nil). Version mismatches are migrated forward.
func (g *Game) Load() error {
	path, err := savePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil // first run — no save yet
	}
	if err != nil {
		return fmt.Errorf("save: read: %w", err)
	}
	var sd SaveData
	if err := json.Unmarshal(data, &sd); err != nil {
		// Corrupt save — start fresh rather than crashing.
		return nil
	}
	migrateForward(&sd)
	g.applySaveData(sd)
	return nil
}

// migrateForward applies schema migrations to bring old saves up to current version.
func migrateForward(sd *SaveData) {
	if sd.EnemyTypesUnlocked == 0 {
		sd.EnemyTypesUnlocked = 1
	}
	if len(sd.AchievementsEarned) < achievementCount {
		padded := make([]bool, achievementCount)
		copy(padded, sd.AchievementsEarned)
		sd.AchievementsEarned = padded
	}
}

// toSaveData extracts a SaveData snapshot. Holds combatMu.RLock while copying
// combat-guarded scalars (gold is not guarded by combatMu but lives in Game).
func (g *Game) toSaveData() SaveData {
	g.combatMu.RLock()
	pt := g.playTime
	g.combatMu.RUnlock()

	archers := make([]savedArcher, len(g.worldArchers))
	for i, a := range g.worldArchers {
		archers[i] = savedArcher{TierIdx: a.tierIdx, X: a.x, Y: a.y}
	}

	return SaveData{
		Version:            saveVersion,
		SavedAt:            time.Now(),
		Gold:               g.gold,
		TotalGoldEarned:    g.stats.totalGoldEarned,
		Archers:            archers,
		MetaUpgradeRanks:   g.metaUpgradeRank,
		ExtraUpgradeRanks:  g.extraUpgradeRank,
		EnemyTypesUnlocked: g.enemyTypesUnlocked,
		PrestigeRank:       g.prestige.rank,
		TotalPrestiges:     g.prestige.totalPrestiges,
		Souls:              g.prestige.souls,
		SoulUpgradeRanks:   g.prestige.soulRanks,
		PeakGold:           g.prestige.peakGold,
		AchievementsEarned: append([]bool(nil), g.achievementsEarned...),
		TotalKills:         g.stats.totalKills,
		KillsByVariant:     g.stats.killsByVariant,
		TotalWaves:         g.stats.totalWaves,
		TotalBossKills:     g.stats.totalBossKills,
		MaxGoldPerSec:      g.stats.maxGoldPerSec,
		BestBossGold:       g.stats.bestBossGold,
		LifetimePlaySec:    g.stats.lifetimePlaySec, // already accumulates rawDt every frame; do not add playTime
		WaveCounter:        g.waveCounter,
		GoldPerSecSnapshot: g.gpsSample,
		SpeedIndex:         g.speedIndex,
		ShopOpen:           g.shopOpen,
		ActiveTab:          int(g.activeTab),
		AutoMgrPurchased:   g.autoMgr.purchased,
		AutoMgrRank:        g.autoMgr.rank,
	}
}

// applySaveData restores mutable state from sd. Does not touch asset or struct fields.
func (g *Game) applySaveData(sd SaveData) {
	g.gold = sd.Gold
	g.stats.totalGoldEarned = sd.TotalGoldEarned
	g.metaUpgradeRank = sd.MetaUpgradeRanks
	g.extraUpgradeRank = sd.ExtraUpgradeRanks
	if sd.EnemyTypesUnlocked > 0 {
		g.enemyTypesUnlocked = sd.EnemyTypesUnlocked
	}

	// Restore archers.
	g.worldArchers = g.worldArchers[:0]
	for _, a := range sd.Archers {
		tier := a.TierIdx
		if tier < 0 || tier >= len(g.archerTierRuntimes) {
			continue
		}
		tr := &g.archerTierRuntimes[tier]
		clone := newGIFAnimShared(tr.idleFrames, tr.idleDelays, true)
		if clone != nil && len(tr.idleFrames) > 0 {
			clone.jumpToFrame(g.rng.IntN(len(tr.idleFrames)))
		}
		g.worldArchers = append(g.worldArchers, worldArcher{
			idleAnim: clone,
			tierIdx:  tier,
			x:        a.X,
			y:        a.Y,
		})
	}
	g.refreshShopArcherPreview()

	// Prestige.
	g.prestige.rank = sd.PrestigeRank
	g.prestige.totalPrestiges = sd.TotalPrestiges
	g.prestige.souls = sd.Souls
	g.prestige.soulRanks = sd.SoulUpgradeRanks
	g.prestige.peakGold = sd.PeakGold

	// Achievements.
	g.achievementsEarned = sd.AchievementsEarned
	g.rebuildAchievementMuls()

	// Statistics.
	g.stats.totalKills = sd.TotalKills
	g.stats.killsByVariant = sd.KillsByVariant
	g.stats.totalWaves = sd.TotalWaves
	g.stats.totalBossKills = sd.TotalBossKills
	g.stats.maxGoldPerSec = sd.MaxGoldPerSec
	g.stats.bestBossGold = sd.BestBossGold
	g.stats.lifetimePlaySec = sd.LifetimePlaySec

	// Wave / boss.
	g.waveCounter = sd.WaveCounter
	g.boss.nextBossWave = sd.WaveCounter + bossWaveInterval
	if g.boss.nextBossWave < bossWaveInterval {
		g.boss.nextBossWave = bossWaveInterval
	}

	// Settings.
	if sd.SpeedIndex >= 0 && sd.SpeedIndex < len(speedMultipliers) {
		g.speedIndex = sd.SpeedIndex
	}
	g.shopOpen = sd.ShopOpen
	if sd.ActiveTab >= 0 && sd.ActiveTab < int(tabCount) {
		g.activeTab = shopTab(sd.ActiveTab)
	}

	// Auto-manager.
	g.autoMgr.purchased = sd.AutoMgrPurchased
	g.autoMgr.rank = sd.AutoMgrRank

	applyOfflineProgress(g, sd)
}

// applyOfflineProgress credits gold for time elapsed since the save was made.
func applyOfflineProgress(g *Game, sd SaveData) {
	if sd.GoldPerSecSnapshot <= 0 || sd.SavedAt.IsZero() {
		return
	}
	const maxOfflineSec = 24 * 3600.0
	elapsed := time.Since(sd.SavedAt).Seconds()
	if elapsed < 5 {
		return
	}
	if elapsed > maxOfflineSec {
		elapsed = maxOfflineSec
	}
	// GoldPerSecSnapshot already incorporates prestigeGoldMul and extraGoldMul
	// (dropKillLoot applies them before feeding gpsAccum). Do not re-multiply.
	eff := g.offlineEfficiency()
	earned := sd.GoldPerSecSnapshot * elapsed * eff
	g.gold += earned
	g.stats.totalGoldEarned += earned

	// Simulate waves offline (no actual enemy spawns, just counter + boss checks).
	offlineWaves := int(elapsed / enemySpawnInterval)
	const maxOfflineWaves = 10_000
	if offlineWaves > maxOfflineWaves {
		offlineWaves = maxOfflineWaves
	}
	g.waveCounter += offlineWaves
	g.stats.totalWaves += int64(offlineWaves)
	// Advance boss schedule so nextBossWave is never behind waveCounter.
	for g.boss.nextBossWave <= g.waveCounter {
		g.boss.nextBossWave += bossWaveInterval
	}

	h := elapsed / 3600
	g.offlineMessage = fmt.Sprintf(
		"Away for %.1fh — earned %s gold (%.0f%% offline rate)",
		h, formatGold(earned), eff*100,
	)
	g.offlineMessageTimer = 8.0
}
