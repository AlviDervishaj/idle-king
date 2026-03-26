package game

import (
	"image/color"
	"math"

	"github.com/AlviDervishaj/idle-king/internal/spawn"
	"github.com/AlviDervishaj/idle-king/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	enemySpeedPxPerSec    = 36.0
	arrowSpeedPxPerSec    = 240.0
	archerFireIntervalSec = 0.82
	// enemySpawnInterval is the fixed seconds between spawn checks.
	enemySpawnInterval = 4.8
	lavaDPSPerSec      = 90.0
	arrowDamage        = 52.0
	enemyDrawScale     = 1.06
	arrowDrawScale     = 0.042
	enemyWalkFrameSec  = 0.11
	enemyDeathFrameSec = 0.12
	arrowHitRadiusPx   = 18.0
	archerMaxRangePx   = 1400.0
	archerThreatBiasPx = 110.0

	enemyHealthBarWWorld    = 28.0
	enemyHealthBarHWorld    = 4.0
	enemyHealthBarAboveFeet = 40.0
	enemyHitCenterOffsetY   = -24.0

	// maxSpawnCatchUpPerFrame bounds spawn work if a frame runs long (avoids unbounded loops).
	maxSpawnCatchUpPerFrame = 6

	maxArrowsInFlightPerEnemy = 2
	arrowArcHeightMinPx       = 18.0
	arrowArcHeightMaxPx       = 130.0
	arrowArcHeightFactor      = 0.38 // vs horizontal span
	arrowFlightDurationMinSec = 0.06
	arrowFlightDurationMaxSec = 2.8

	arrowFlightFrameSec = 0.12 // time per flight animation frame
)

type worldEnemy struct {
	x, y  float64
	vx    float64
	hp    float64
	hpMax float64

	dying      bool
	variantIdx int
	frames     []*ebiten.Image
	fIdx       int
	fAccum     float64

	id             uint64
	incomingDamage float64

	isBoss bool // boss enemies have scaled HP and drop bonus gold
}

type worldArrow struct {
	x, y float64
	// Parabolic arc from (startX, startY) to (endX, endY); arcHeight peaks at mid-flight (screen Y down).
	startX, startY float64
	endX, endY     float64
	duration       float64
	elapsed        float64
	arcHeight      float64
	drawAng        float64
	targetEnemyID  uint64
	damage         float64 // raw damage before target multipliers
	reservedDamage float64 // damage reserved on target (after target multipliers)

	flightFrames []*ebiten.Image
	flightFIdx   int
	flightFAccum float64
}

func (g *Game) updateCombat(dt float64) {
	g.combatMu.Lock()
	defer g.combatMu.Unlock()

	if g.lavaMask == nil {
		return
	}

	g.playTime += dt

	interval := enemySpawnInterval
	g.spawnAccum += dt
	for range maxSpawnCatchUpPerFrame {
		if g.spawnAccum < interval {
			break
		}
		g.spawnAccum -= interval
		g.waveCounter++
		g.stats.totalWaves++
		// Spawn a boss every bossWaveInterval waves.
		if g.waveCounter >= g.boss.nextBossWave {
			g.spawnBossWave()
			g.boss.nextBossWave = g.waveCounter + bossWaveInterval
		}
		n := 5 + g.metaUpgradeRank[metaSpawnWaveSize]
		for range n {
			if x, y, ok := spawn.PickEnemyFeet(g.rng, g.lavaMask); ok {
				if len(g.enemyVariants) == 0 {
					continue
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
				g.nextEnemyID++
				g.enemies = append(g.enemies, worldEnemy{
					x: x, y: y, vx: -enemySpeedPxPerSec,
					hp: v.maxHP, hpMax: v.maxHP,
					variantIdx: vi,
					frames:     v.walkFrames,
					id:         g.nextEnemyID,
				})
			}
		}
	}

	next := g.enemies[:0]
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.dying {
			if len(e.frames) == 0 {
				continue
			}
			e.fAccum += dt
			for e.fAccum >= enemyDeathFrameSec {
				e.fAccum -= enemyDeathFrameSec
				e.fIdx++
			}
			if e.fIdx >= len(e.frames) {
				continue
			}
			next = append(next, *e)
			continue
		}

		e.x += e.vx * dt
		e.fAccum += dt
		if e.fAccum >= enemyWalkFrameSec {
			e.fAccum = 0
			if len(e.frames) > 0 {
				e.fIdx = (e.fIdx + 1) % len(e.frames)
			}
		}
		if g.lavaMask.IsLavaAtWorld(e.x, e.y) {
			e.hp -= lavaDPSPerSec * g.extraLavaDPSMul() * dt
		}
		row, col := world.WorldToTile(e.x, e.y)
		if !g.lavaMask.IsLava(row, col) {
			left := g.lavaMask.LeftmostLavaCol(row)
			if left < world.WidthTiles && col < left {
				e.hp = 0
			}
		}
		if e.hp <= 0 {
			g.beginEnemyDeath(e, deathCauseEnvironmental)
			if len(e.frames) > 0 {
				next = append(next, *e)
			}
			continue
		}
		next = append(next, *e)
	}
	g.enemies = next

	fireInterval := archerFireIntervalSec * g.extraFireIntervalMul()
	g.fireAccum += dt
	if g.fireAccum >= fireInterval {
		g.fireAccum = 0
		g.fireArrows()
	}
	g.updateArcherAttacks(dt)

	arrows := g.arrows[:0]
	for i := range g.arrows {
		a := &g.arrows[i]

		a.elapsed += dt
		// Advance flight animation frame.
		if len(a.flightFrames) > 0 {
			a.flightFAccum += dt
			for a.flightFAccum >= arrowFlightFrameSec {
				a.flightFAccum -= arrowFlightFrameSec
				a.flightFIdx = (a.flightFIdx + 1) % len(a.flightFrames)
			}
		}

		if a.duration <= 0 {
			arrows = append(arrows, *a)
			continue
		}
		p := a.elapsed / a.duration
		if p > 1 {
			p = 1
		}
		sin := math.Sin(math.Pi * p)
		cos := math.Cos(math.Pi * p)
		a.x = a.startX + (a.endX-a.startX)*p
		a.y = a.startY + (a.endY-a.startY)*p - a.arcHeight*sin
		// Tangent for rotation (derivative of position wrt time).
		dx := (a.endX - a.startX) / a.duration
		dy := (a.endY-a.startY)/a.duration - a.arcHeight*math.Pi*cos/a.duration
		a.drawAng = math.Atan2(dy, dx)

		offScreen := a.x < 0 || a.x > float64(world.MapWidthPx) || a.y < 0 || a.y > float64(world.MapHeightPx)
		expired := a.elapsed >= a.duration
		hit := false

		if !offScreen && !expired {
			for j := range g.enemies {
				e := &g.enemies[j]
				if e.dying || e.hp <= 0 {
					continue
				}
				// Distance check vs enemy visual "body center" (y - 24) instead of feet (y).
				if math.Hypot(a.x-e.x, a.y-(e.y+enemyHitCenterOffsetY)) <= arrowHitRadiusPx {
					dmg := a.damage
					if e.variantIdx >= 0 && e.variantIdx < len(g.enemyVariants) {
						dmg *= g.enemyVariants[e.variantIdx].arrowTakenMul
					}
					e.hp -= dmg
					if e.hp <= 0 {
						g.beginEnemyDeath(e, deathCausePlayerArrow)
					}
					hit = true
					break
				}
			}
		}

		if hit || offScreen || expired {
			// Release reserved damage from target if it still exists.
			if target := g.findEnemyByID(a.targetEnemyID); target != nil {
				target.incomingDamage -= a.reservedDamage
				if target.incomingDamage < 0 {
					target.incomingDamage = 0
				}
			}
			continue
		}
		arrows = append(arrows, *a)
	}
	g.arrows = arrows

	g.applySummonerLaser(dt)
	g.updateGoldFloats(dt)
}

func (g *Game) findEnemyByID(id uint64) *worldEnemy {
	for i := range g.enemies {
		if g.enemies[i].id == id {
			return &g.enemies[i]
		}
	}
	return nil
}

func (g *Game) fireArrows() {
	if len(g.worldArchers) == 0 {
		return
	}
	// tickPending tracks damage already committed this fire round per enemy ID so
	// subsequent archers avoid piling onto the same target.
	tickPending := make(map[uint64]float64, len(g.enemies))
	for i := range g.worldArchers {
		a := &g.worldArchers[i]
		if a.idleAnim == nil || a.attackAnim != nil {
			continue
		}
		idx := g.pickEnemyForArcher(a.x, a.y, tickPending)
		if idx < 0 {
			continue
		}
		e := &g.enemies[idx]
		if e.dying {
			continue
		}
		// Reserve this archer's expected damage so the next archer accounts for it.
		dmg := g.arrowDamageForTier(a.tierIdx)
		if e.variantIdx >= 0 && e.variantIdx < len(g.enemyVariants) {
			dmg *= g.enemyVariants[e.variantIdx].arrowTakenMul
		}
		tickPending[e.id] += dmg
		// Archer starts animation and targets the enemy
		g.triggerArcherAttack(i)
		a.pendingTargetID = e.id
		a.hasFired = false
	}
}

func (g *Game) updateArcherAttacks(dt float64) {
	for i := range g.worldArchers {
		a := &g.worldArchers[i]
		if a.attackAnim == nil || a.hasFired {
			continue
		}
		// Spawn arrow at a specific frame (roughly halfway)
		if a.attackAnim.idx >= len(a.attackAnim.frames)/2 {
			a.hasFired = true
			e := g.findEnemyByID(a.pendingTargetID)
			if e == nil || e.dying || e.hp <= 0 {
				// Original target is gone at the moment of release; try to pick a new one
				// so the archer doesn't waste the entire animation/cooldown.
				idx := g.pickEnemyForArcher(a.x, a.y, nil)
				if idx >= 0 {
					e = &g.enemies[idx]
				}
			}
			if e == nil || e.dying || e.hp <= 0 {
				continue
			}
			sx := a.x + 20
			sy := a.y - 10
			if math.Hypot(e.x-sx, e.y-sy) < 8 {
				continue
			}
			arrowSpd := arrowSpeedPxPerSec * g.extraArrowSpeedMul()
			endX, endY, duration := leadLandingPoint(e, sx, sy, arrowSpd)
			horiz := math.Abs(endX - sx)
			arcH := horiz * arrowArcHeightFactor
			if arcH < arrowArcHeightMinPx {
				arcH = arrowArcHeightMinPx
			}
			if arcH > arrowArcHeightMaxPx {
				arcH = arrowArcHeightMaxPx
			}
			dx0 := (endX - sx) / duration
			dy0 := (endY-sy)/duration - arcH*math.Pi/duration
			initAng := math.Atan2(dy0, dx0)

			rawDmg := g.arrowDamageForTier(a.tierIdx)
			reservedDmg := rawDmg
			if e.variantIdx >= 0 && e.variantIdx < len(g.enemyVariants) {
				reservedDmg *= g.enemyVariants[e.variantIdx].arrowTakenMul
			}
			e.incomingDamage += reservedDmg

			tr := &g.archerTierRuntimes[a.tierIdx]
			g.arrows = append(g.arrows, worldArrow{
				x: sx, y: sy,
				startX: sx, startY: sy,
				endX: endX, endY: endY,
				duration:       duration,
				elapsed:        0,
				arcHeight:      arcH,
				drawAng:        initAng,
				targetEnemyID:  e.id,
				damage:         rawDmg,
				reservedDamage: reservedDmg,
				flightFrames:   tr.arrowFlightFrames,
			})
		}
	}
}

func (g *Game) triggerArcherAttack(archerIdx int) {
	if archerIdx < 0 || archerIdx >= len(g.worldArchers) {
		return
	}
	a := &g.worldArchers[archerIdx]
	t := a.tierIdx
	if t < 0 || t >= len(g.archerTierRuntimes) {
		return
	}
	tr := &g.archerTierRuntimes[t]
	if len(tr.attackFrames) == 0 {
		return
	}
	a.attackAnim = newGIFAnimShared(tr.attackFrames, tr.attackDelays, false)
	if a.attackAnim != nil {
		a.attackAnim.jumpToFrame(0)
	}
}

// pickEnemyForArcher chooses an in-range enemy to target, prioritizing those closer
// to the goal (left side) and avoiding overkill. tickPending tracks damage already
// committed to each enemy this fire round so archers spread across targets.
func (g *Game) pickEnemyForArcher(ax, ay float64, tickPending map[uint64]float64) int {
	best := -1
	first := true
	var bestScore float64
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.dying || e.hp <= 0 {
			continue
		}
		// Skip if in-flight + this-tick committed damage will already kill it.
		if e.hp-e.incomingDamage-tickPending[e.id] <= 0 {
			continue
		}
		if math.Hypot(e.x-ax, e.y-ay) > archerMaxRangePx {
			continue
		}
		sc := archerTargetScore(ax, ay, e, tickPending[e.id])
		if first || sc < bestScore {
			first = false
			best = i
			bestScore = sc
		}
	}
	return best
}

func leadLandingPoint(e *worldEnemy, sx, sy, arrowSpeed float64) (predX, predY, duration float64) {
	predX, predY = e.x, e.y+enemyHitCenterOffsetY
	var dist float64
	for range 2 {
		dx := predX - sx
		dy := predY - sy
		dist = math.Hypot(dx, dy)
		if dist < 4 {
			dist = 4
		}
		duration = dist / arrowSpeed
		if duration < arrowFlightDurationMinSec {
			duration = arrowFlightDurationMinSec
		}
		if duration > arrowFlightDurationMaxSec {
			duration = arrowFlightDurationMaxSec
		}
		predX = e.x + e.vx*duration
		predY = e.y + enemyHitCenterOffsetY
	}
	predX = clamp(predX, 4, float64(world.MapWidthPx)-4)
	predY = clamp(predY, 4, float64(world.MapHeightPx)-4)
	return predX, predY, duration
}

func (g *Game) beginEnemyDeath(e *worldEnemy, cause deathCause) {
	if e.dying {
		return
	}
	playerKill := cause == deathCausePlayerArrow || cause == deathCausePlayerLaser
	if playerKill {
		// Stats tracking.
		g.stats.totalKills++
		if e.variantIdx >= 0 && int(e.variantIdx) < len(g.stats.killsByVariant) {
			g.stats.killsByVariant[e.variantIdx]++
		}

		goldMul := 1.0
		if e.isBoss {
			scaleFactor := 1.0 + float64(g.waveCounter)/200.0
			goldMul = bossGoldMultiplier * scaleFactor
			g.stats.totalBossKills++
			g.boss.activeBossID = 0
		}
		n := g.dropKillLoot(e.variantIdx, goldMul)
		if e.isBoss && n > g.stats.bestBossGold {
			g.stats.bestBossGold = n
		}
		if n > 0 {
			g.spawnGoldFloat(e.x, e.y-enemyHealthBarAboveFeet-10, n)
		}
	}
	var deathFrames []*ebiten.Image
	if e.variantIdx >= 0 && e.variantIdx < len(g.enemyVariants) {
		deathFrames = g.enemyVariants[e.variantIdx].deathFrames
	}
	if len(deathFrames) == 0 {
		e.dying = true
		e.vx = 0
		e.hp = 0
		e.frames = nil
		return
	}
	e.dying = true
	e.vx = 0
	e.hp = 0
	e.frames = deathFrames
	e.fIdx = 0
	e.fAccum = 0
}

// archerTargetScore returns lower-is-better ordering: geometric distance plus bias so enemies
// further left (closer to breaching past lava) are prioritized when distances are similar.
// pendingDmg is damage already committed to this enemy this fire round; it adds a spread
// penalty so archers naturally distribute across targets instead of piling on one.
func archerTargetScore(ax, ay float64, e *worldEnemy, pendingDmg float64) float64 {
	d := math.Hypot(e.x-ax, e.y-ay)
	// e.x / MapWidthPx: 0 = west (urgent), 1 = east — add cost for eastern targets.
	threat := (e.x / float64(world.MapWidthPx)) * archerThreatBiasPx
	// Penalise already-targeted enemies to encourage spread (0.4px penalty per HP committed).
	spreadPenalty := pendingDmg * 0.4
	return d + threat + spreadPenalty
}

func (g *Game) drawEnemyHealthBars(screen *ebiten.Image, ox, oy, mapScale float64) {
	barW := float32(enemyHealthBarWWorld * mapScale)
	barH := float32(enemyHealthBarHWorld * mapScale)
	bg := color.RGBA{R: 0x18, G: 0x18, B: 0x1a, A: 0xee}
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.dying || e.hp <= 0 {
			continue
		}
		denom := e.hpMax
		if denom <= 0 {
			denom = enemyMaxHPBase
		}
		ratio := e.hp / denom
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		sx := ox + e.x*mapScale
		sy := oy + (e.y-enemyHealthBarAboveFeet)*mapScale
		bx := float32(sx) - barW*0.5
		by := float32(sy)
		vector.FillRect(screen, bx, by, barW, barH, bg, false)
		fillW := barW * float32(ratio)
		if fillW < 0.5 {
			continue
		}
		var fg color.RGBA
		if e.isBoss {
			// Bosses use a purple/gold health bar for visibility.
			fg = color.RGBA{R: 0xcc, G: 0x44, B: 0xee, A: 0xf5}
		} else {
			rr := uint8(255 * (1 - ratio))
			gg := uint8(255 * ratio)
			fg = color.RGBA{R: rr, G: gg, B: 0x28, A: 0xf5}
		}
		vector.FillRect(screen, bx, by, fillW, barH, fg, false)
	}
}

func (g *Game) drawEnemies(screen *ebiten.Image, ox, oy, mapScale float64) {
	s := mapScale * enemyDrawScale
	for i := range g.enemies {
		e := &g.enemies[i]
		if len(e.frames) == 0 {
			continue
		}
		n := len(e.frames)
		var fr *ebiten.Image
		if e.dying {
			if e.fIdx < 0 || e.fIdx >= n {
				continue
			}
			fr = e.frames[e.fIdx]
		} else {
			fr = e.frames[e.fIdx%n]
		}
		if fr == nil {
			continue
		}
		fw, fh := fr.Bounds().Dx(), fr.Bounds().Dy()
		sx := ox + e.x*mapScale
		sy := oy + e.y*mapScale
		sxScale := s
		if e.variantIdx >= 0 && e.variantIdx < len(g.enemyVariants) && g.enemyVariants[e.variantIdx].mirrorX {
			sxScale = -s
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(fw)/2, -float64(fh))
		op.GeoM.Scale(sxScale, s)
		op.GeoM.Translate(sx, sy)
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(fr, op)

	}
}

func (g *Game) drawArrows(screen *ebiten.Image, ox, oy, mapScale float64) {
	s := mapScale * arrowDrawScale
	for i := range g.arrows {
		a := &g.arrows[i]
		if len(a.flightFrames) == 0 {
			continue
		}
		fr := a.flightFrames[a.flightFIdx%len(a.flightFrames)]
		if fr == nil {
			continue
		}
		fw, fh := fr.Bounds().Dx(), fr.Bounds().Dy()
		sx := ox + a.x*mapScale
		sy := oy + a.y*mapScale
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(fw)/2, -float64(fh)/2)
		op.GeoM.Scale(s, s)
		op.GeoM.Rotate(a.drawAng)
		op.GeoM.Translate(sx, sy)
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(fr, op)
	}
}
