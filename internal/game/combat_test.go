package game

import (
	"testing"
)

func TestPickEnemyForArcher(t *testing.T) {
	g := &Game{
		enemies: []worldEnemy{
			{id: 1, x: 100, y: 100, hp: 100, incomingDamage: 0},
			{id: 2, x: 200, y: 100, hp: 100, incomingDamage: 0},
			{id: 3, x: 300, y: 100, hp: 100, incomingDamage: 100}, // Guaranteed kill
			{id: 4, x: 400, y: 100, hp: 100, incomingDamage: 0, dying: true},
		},
	}

	// Archer at (50, 100)
	// Enemy 1: dist 50
	// Enemy 2: dist 150
	// Enemy 3: skip (overkill)
	// Enemy 4: skip (dying)

	idx := g.pickEnemyForArcher(50, 100)
	if idx != 0 {
		t.Errorf("pickEnemyForArcher(50, 100) = %d, want 0", idx)
	}

	// Test range limit
	// archerMaxRangePx = 1400.0 (from combat.go)
	g.enemies[0].x = 2000
	idx = g.pickEnemyForArcher(50, 100)
	if idx != 1 {
		t.Errorf("pickEnemyForArcher(50, 100) after moving enemy 0 = %d, want 1", idx)
	}

	g.enemies[1].x = 2000
	idx = g.pickEnemyForArcher(50, 100)
	if idx != -1 {
		t.Errorf("pickEnemyForArcher(50, 100) with all out of range = %d, want -1", idx)
	}
}

func TestArcherTargetScore(t *testing.T) {
	ax, ay := 0.0, 0.0
	e1 := &worldEnemy{x: 100, y: 0}
	e2 := &worldEnemy{x: 110, y: 0}

	// e1 is closer and further left (more threat)
	s1 := archerTargetScore(ax, ay, e1)
	s2 := archerTargetScore(ax, ay, e2)

	if s1 >= s2 {
		t.Errorf("Score for e1 (%v) should be lower than e2 (%v)", s1, s2)
	}

	// Threat bias check: enemy slightly further but much further left
	// threat = (x / MapWidthPx) * archerThreatBiasPx
	// archerThreatBiasPx = 110.0
	// MapWidthPx = 1024 (32 * 32)

	// Actually archerTargetScore uses Hypot(e.x-ax, e.y-ay) + (e.x / MapWidthPx) * 110

	// Case: Enemy A at (100, 0), Enemy B at (95, 20)
	// Dist A = 100, Threat A = (100/1024)*110 = 10.74. Total = 110.74
	// Dist B = sqrt(95^2 + 20^2) = 97.08. Threat B = (95/1024)*110 = 10.20. Total = 107.28
	// B is better.

	// Case: Enemy A at (50, 0), Enemy B at (10, 100)
	// A: Dist 50, Threat (50/1024)*110 = 5.37. Total = 55.37
	// B: Dist 100.5, Threat (10/1024)*110 = 1.07. Total = 101.57
	// A is better.
}

func TestFindEnemyByID(t *testing.T) {
	g := &Game{
		enemies: []worldEnemy{
			{id: 10, x: 1},
			{id: 20, x: 2},
			{id: 30, x: 3},
		},
	}

	e := g.findEnemyByID(20)
	if e == nil || e.id != 20 {
		t.Errorf("findEnemyByID(20) failed")
	}

	e = g.findEnemyByID(99)
	if e != nil {
		t.Errorf("findEnemyByID(99) should be nil")
	}
}

func TestLeadLandingPoint(t *testing.T) {
	e := &worldEnemy{x: 100, y: 100, vx: -50}
	sx, sy := 0.0, 100.0

	// enemyHitCenterOffsetY = -24.0
	// arrowSpeedPxPerSec = 420.0

	px, py, dur := leadLandingPoint(e, sx, sy)

	// Expected py should be e.y + enemyHitCenterOffsetY
	if py != 100-24 {
		t.Errorf("leadLandingPoint py = %v, want %v", py, 100-24)
	}

	// Dist is roughly 100. dur = 100/420 = 0.238
	// predX = 100 + (-50 * 0.238) = 88.1
	// Second iteration will refine it.

	if px >= 100 {
		t.Errorf("leadLandingPoint px should be less than e.x due to negative vx, got %v", px)
	}

	if dur <= 0 {
		t.Errorf("leadLandingPoint duration should be positive, got %v", dur)
	}
}
