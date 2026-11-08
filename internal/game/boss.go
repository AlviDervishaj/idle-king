package game

import (
	"github.com/AlviDervishaj/idle-king/internal/spawn"
)

const (
	bossWaveInterval   = 10   // spawn a boss every 10th wave
	bossHPMultiplier   = 25.0 // boss HP = variant base * 25 * scaleFactor
	bossGoldMultiplier = 50.0 // boss gold = base * 50 * scaleFactor
	bossSpeedMul       = 0.6  // bosses move at 60% of normal speed
)

// bossState tracks the boss wave schedule and the active boss (if any).
type bossState struct {
	nextBossWave int    // wave count at which the next boss spawns
	activeBossID uint64 // 0 if no boss is currently alive
}

// spawnBossWave spawns one boss enemy. Must be called while combatMu.Lock() is held.
func (g *Game) spawnBossWave() {
	x, y, ok := spawn.PickEnemyFeet(g.rng, g.lavaMask)
	if !ok {
		return
	}
	if len(g.enemyVariants) == 0 {
		return
	}
	pool := g.enemyTypesUnlocked
	if pool > len(g.enemyVariants) {
		pool = len(g.enemyVariants)
	}
	if pool < 1 {
		pool = 1
	}
	vi := g.rng.IntN(pool)
	v := &g.enemyVariants[vi]

	// Scale HP with wave count so later bosses are harder.
	scaleFactor := 1.0 + float64(g.waveCounter)/100.0
	hp := v.maxHP * bossHPMultiplier * scaleFactor

	g.nextEnemyID++
	e := worldEnemy{
		x: x, y: y,
		vx:         -enemySpeedPxPerSec * bossSpeedMul,
		hp:         hp,
		hpMax:      hp,
		variantIdx: vi,
		frames:     v.walkFrames,
		id:         g.nextEnemyID,
		isBoss:     true,
	}
	g.enemies = append(g.enemies, e)
	g.boss.activeBossID = e.id
}
