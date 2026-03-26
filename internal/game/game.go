package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/AlviDervishaj/idle-king/internal/spawn"
	"github.com/AlviDervishaj/idle-king/objects/atlas"
	"github.com/AlviDervishaj/idle-king/objects/lava"
	"github.com/AlviDervishaj/idle-king/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var letterboxFill = color.RGBA{R: 0x18, G: 0x22, B: 0x16, A: 0xff}

const mapArcherSpriteScale = 0.55

type worldArcher struct {
	idleAnim   *gifAnim
	attackAnim *gifAnim
	tierIdx    int
	x, y       float64

	pendingTargetID uint64
	hasFired        bool
}

// Game implements ebiten.Game: map, hired archers, optional recruit shop.
// combatMu guards enemies, arrows, spawn/fire timers, and playTime. Ebiten runs Update/Draw
// sequentially on one thread; the mutex still serializes access for clarity and future safety.
type Game struct {
	mapImg *ebiten.Image

	attackPreview *gifAnim

	archerTierRuntimes []archerTierRuntime
	shopArcherPreview  *gifAnim

	archerSpawn *spawn.ArcherSpawner
	lavaMask    *lava.LavaMask

	gold            float64
	metaUpgradeRank [metaUpgradeCount]int

	enemyTypesUnlocked int // spawn pool uses variants [0, enemyTypesUnlocked)

	worldArchers      []worldArcher
	selectedArcherIdx int // -1 if none; can click world archer to select for dismissal

	combatMu    sync.RWMutex
	playTime    float64
	nextEnemyID uint64

	enemies       []worldEnemy
	arrows        []worldArrow
	enemyVariants []enemyVariant
	spawnAccum    float64
	fireAccum     float64

	goldFloats []goldFloat

	cardRect     image.Rectangle
	portraitRect image.Rectangle
	buyBtnRect   image.Rectangle
	buyHovered   bool

	enemyUnlockBtnRect image.Rectangle
	unlockEnemyHovered bool

	upgradeBtnRects [metaUpgradeCount]image.Rectangle
	upgradeHovered  [metaUpgradeCount]bool

	shopOpen        bool
	shopToggleRect  image.Rectangle
	shopToggleHover bool

	rng *rand.Rand

	// lastFrameTime is used with time.Now for frame delta; do not use ebiten.ActualTPS for dt (see frameDT).
	lastFrameTime time.Time

	// Summoner laser (world px): aim follows mouse; origin is fixed at summoner platform.
	laserWorldX float64
	laserWorldY float64
	laserPhase  float64
}

const (
	arrowPackCellSize = 1024
	arrowFlightCols   = 3
	enemyWalkFrameW   = 48
	enemyWalkFrameH = 48
)

// tierArrowRow maps the 6 archer tiers to 4 arrow sprite rows in Arrows_pack.png.
// Row 0 = gray (Common/Uncommon), 1 = blue (Rare), 2 = green (Ancient/Epic), 3 = red (Legendary).
var tierArrowRow = []int{0, 0, 1, 2, 2, 3}

// NewGame decodes map PNG, arrow sprite sheet, enemy strips, and all archer tier GIF pairs.
func NewGame(mapPNG, arrowsPNG []byte, enemyStrips []EnemyStripPair, archerTierAssets []ArcherTierAssets) (*Game, error) {
	img, _, err := image.Decode(bytes.NewReader(mapPNG))
	if err != nil {
		return nil, err
	}

	// Arrows_pack.png: 4 rows × 3 cols, 1024×1024 cells → 3 flight frames per arrow type.
	flightCells, err := atlas.SplitPNGGrid(arrowsPNG, arrowPackCellSize, arrowPackCellSize)
	if err != nil {
		return nil, fmt.Errorf("game: arrows pack: %w", err)
	}
	if len(flightCells) < 4*arrowFlightCols {
		return nil, errors.New("game: arrows pack too small")
	}

	arrowFlightByTier := make([][]*ebiten.Image, archerTierCount)
	for i := range archerTierCount {
		row := tierArrowRow[i]
		fs := row * arrowFlightCols
		arrowFlightByTier[i] = flightCells[fs : fs+arrowFlightCols]
	}

	enemyVariants, err := loadEnemyVariantsFromStrips(enemyStrips)
	if err != nil {
		return nil, err
	}
	archerTierRuntimes, err := loadArcherTiers(archerTierAssets, arrowFlightByTier)
	if err != nil {
		return nil, err
	}

	const (
		shopCardX      = 24
		shopCardY      = 48
		shopCardW      = 528
		shopCardH      = 500
		innerPad       = 16
		shopDividerOff = 260 // px from card left; left column = recruit, right = contracts
		upgradeBtnH    = 50
		upgradeBtnGap  = 10
	)
	const (
		contractsTitleY   = 48
		contractsBtnStart = 84
		summonerTitleYOff = 214 // from first contract button top
	)
	innerL := shopCardX + innerPad
	dividerX := shopCardX + shopDividerOff
	leftInnerR := dividerX - 8
	rightInnerL := dividerX + 8
	innerR := shopCardX + shopCardW - innerPad

	portraitW, portraitH := 120, 120
	px := innerL + (leftInnerR-innerL-portraitW)/2
	portraitTop := shopCardY + 56
	portraitRect := image.Rect(px, portraitTop, px+portraitW, portraitTop+portraitH)

	const (
		tierRowH = 36
		btnH     = 46
		gap      = 8
	)
	tierRowTop := portraitRect.Max.Y + 38
	buyTop := tierRowTop + tierRowH + gap
	upgradeLeftTop := buyTop + btnH + gap

	buyBtnRect := image.Rect(innerL, buyTop, leftInnerR, buyTop+btnH)

	firstRow := shopCardY + contractsBtnStart
	enemyUnlockBtnRect := image.Rect(rightInnerL, firstRow, innerR, firstRow+upgradeBtnH)
	metaStart := firstRow + upgradeBtnH + upgradeBtnGap

	var upgradeBtnRects [metaUpgradeCount]image.Rectangle
	rightIdx := 0
	for i := range upgradeBtnRects {
		if i == metaArcherTier {
			upgradeBtnRects[i] = image.Rect(innerL, upgradeLeftTop, leftInnerR, upgradeLeftTop+upgradeBtnH)
			continue
		}
		top := metaStart + rightIdx*(upgradeBtnH+upgradeBtnGap)
		if i == metaLaserDamage || i == metaLaserRadius {
			// Offset for SUMMONER title
			top += 40
		}
		upgradeBtnRects[i] = image.Rect(rightInnerL, top, innerR, top+upgradeBtnH)
		rightIdx++
	}

	const toggleW, toggleH = 112, 36
	shopToggleRect := image.Rect(24, 16, 24+toggleW, 16+toggleH)

	lav := lava.Default()
	spawner := spawn.NewArcherSpawner(lav)
	laserAX, laserAY := defaultSummonerLaserAim()

	shopPreview := newGIFAnimShared(archerTierRuntimes[0].idleFrames, archerTierRuntimes[0].idleDelays, true)

	g := &Game{
		mapImg:             ebiten.NewImageFromImage(img),
		archerTierRuntimes: archerTierRuntimes,
		shopArcherPreview:  shopPreview,
		archerSpawn:        spawner,
		lavaMask:           lav,
		enemyVariants:      enemyVariants,
		enemyTypesUnlocked: 1,
		spawnAccum:         enemySpawnInterval * 0.45,
		fireAccum:          archerFireIntervalSec,
		gold:               0,
		cardRect:           image.Rect(shopCardX, shopCardY, shopCardX+shopCardW, shopCardY+shopCardH),
		portraitRect:       portraitRect,
		buyBtnRect:         buyBtnRect,
		enemyUnlockBtnRect: enemyUnlockBtnRect,
		upgradeBtnRects:    upgradeBtnRects,
		shopToggleRect:     shopToggleRect,
		shopOpen:           false,
		selectedArcherIdx:  -1,
		rng:                rand.New(rand.NewPCG(0x49444c45, 0x4b494e47)),
		laserWorldX:        laserAX,
		laserWorldY:        laserAY,
	}
	return g, nil
}

func (g *Game) refreshShopArcherPreview() {
	t := g.metaUpgradeRank[metaArcherTier]
	if t < 0 || t >= len(g.archerTierRuntimes) {
		return
	}
	tr := &g.archerTierRuntimes[t]
	g.shopArcherPreview = newGIFAnimShared(tr.idleFrames, tr.idleDelays, true)
}

func (g *Game) updateAllArchersToCurrentTier() {
	tier := g.metaUpgradeRank[metaArcherTier]
	if tier < 0 || tier >= len(g.archerTierRuntimes) {
		return
	}
	tr := &g.archerTierRuntimes[tier]
	for i := range g.worldArchers {
		wa := &g.worldArchers[i]
		wa.tierIdx = tier
		wa.idleAnim = newGIFAnimShared(tr.idleFrames, tr.idleDelays, true)
		if wa.idleAnim != nil {
			wa.idleAnim.jumpToFrame(g.rng.IntN(len(tr.idleFrames)))
		}
		// Clear attack anim so they don't finish an old tier animation
		wa.attackAnim = nil
	}
}

func (g *Game) findArcherUnderCursor() int {
	if g.mapImg == nil {
		return -1
	}
	mw, mh := g.mapImg.Bounds().Dx(), g.mapImg.Bounds().Dy()
	mx, my := ebiten.CursorPosition()
	wx, wy := screenToMapWorld(mx, my, mw, mh, world.ViewportWidth, world.ViewportHeight)

	best := -1
	minDist := 1e9
	const clickRadius = 24.0
	for i := range g.worldArchers {
		a := &g.worldArchers[i]
		// Vertical offset since a.y is feet; visual center is roughly y - 22.
		d := math.Hypot(wx-a.x, wy-(a.y-22))
		if d < clickRadius && d < minDist {
			minDist = d
			best = i
		}
	}
	return best
}

func (g *Game) existingArcherFeet() [][2]float64 {
	out := make([][2]float64, len(g.worldArchers))
	for i := range g.worldArchers {
		out[i][0] = g.worldArchers[i].x
		out[i][1] = g.worldArchers[i].y
	}
	return out
}

func (g *Game) Update() error {
	dt := g.frameDT()
	g.laserPhase += dt * laserPulseSpeed
	g.updateLaserAim()

	if g.shopArcherPreview != nil {
		g.shopArcherPreview.update(dt)
	}
	for i := range g.worldArchers {
		wa := &g.worldArchers[i]
		if wa.attackAnim != nil {
			wa.attackAnim.update(dt * 1.35)
			if wa.attackAnim.finished {
				wa.attackAnim = nil
			}
		} else if wa.idleAnim != nil {
			wa.idleAnim.update(dt * 1.1)
		}
	}
	if g.attackPreview != nil {
		g.attackPreview.update(dt * 1.35)
		if g.attackPreview.finished {
			g.attackPreview = nil
		}
	}

	g.updateCombat(dt)

	g.shopToggleHover = cursorIn(g.shopToggleRect)
	g.buyHovered = g.shopOpen && cursorIn(g.buyBtnRect)
	g.unlockEnemyHovered = g.shopOpen && cursorIn(g.enemyUnlockBtnRect)
	for i := range g.upgradeHovered {
		g.upgradeHovered[i] = g.shopOpen && cursorIn(g.upgradeBtnRects[i])
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.shopToggleHover {
			g.shopOpen = !g.shopOpen
			if !g.shopOpen {
				g.selectedArcherIdx = -1
			}
		} else if g.shopOpen && g.buyHovered && g.gold >= float64(g.selectedArcherHireCost()) {
			tier := g.metaUpgradeRank[metaArcherTier]
			if tier >= 0 && tier < len(g.archerTierRuntimes) {
				tr := &g.archerTierRuntimes[tier]
				cost := tr.hireCost
				wx, wy, ok := g.archerSpawn.RandomArcher(g.rng, g.existingArcherFeet())
				if ok {
					g.gold -= float64(cost)
					clone := newGIFAnimShared(tr.idleFrames, tr.idleDelays, true)
					if clone != nil {
						clone.jumpToFrame(g.rng.IntN(len(tr.idleFrames)))
					}
					g.worldArchers = append(g.worldArchers, worldArcher{
						idleAnim: clone,
						tierIdx:  tier,
						x:        wx, y: wy,
					})
					shot, err := newGIFAnim(tr.attackBytes, false)
					if err == nil {
						g.attackPreview = shot
					}
				}
			}
		} else if g.shopOpen && g.unlockEnemyHovered {
			cost := g.enemyUnlockNextCost()
			if cost > 0 && g.gold >= float64(cost) {
				g.gold -= float64(cost)
				g.enemyTypesUnlocked++
			}
		} else if g.shopOpen {
			handled := false
			for i := range g.upgradeBtnRects {
				if g.upgradeHovered[i] {
					cost := g.metaUpgradeNextCost(i)
					if cost > 0 && g.gold >= float64(cost) {
						g.gold -= float64(cost)
						g.metaUpgradeRank[i]++
						if i == metaArcherTier {
							g.updateAllArchersToCurrentTier()
							g.refreshShopArcherPreview()
						}
					}
					handled = true
					break
				}
			}
			if !handled && !cursorIn(g.cardRect) {
				g.selectedArcherIdx = g.findArcherUnderCursor()
			}
		}
	}
	return nil
}

// frameDT returns real elapsed time since the last Update, capped to avoid huge jumps when
// the window loses focus or the OS stalls the process (Ebiten's ActualTPS is not valid for dt).
func (g *Game) frameDT() float64 {
	const (
		defaultStep = 1.0 / 60.0
		maxStep     = 0.12 // ~8.3 Hz minimum; prevents "catch-up" explosions
	)
	now := time.Now()
	if g.lastFrameTime.IsZero() {
		g.lastFrameTime = now
		return defaultStep
	}
	dt := now.Sub(g.lastFrameTime).Seconds()
	g.lastFrameTime = now
	if dt > maxStep {
		dt = maxStep
	}
	if dt < 0 {
		dt = 0
	}
	return dt
}

func (g *Game) Draw(screen *ebiten.Image) {
	b := screen.Bounds()
	sw, sh := b.Dx(), b.Dy()
	mw, mh := g.mapImg.Bounds().Dx(), g.mapImg.Bounds().Dy()
	if mw <= 0 || mh <= 0 || sw <= 0 || sh <= 0 {
		return
	}

	screen.Fill(letterboxFill)

	scale := math.Min(float64(sw)/float64(mw), float64(sh)/float64(mh))
	dx := float64(mw) * scale
	dy := float64(mh) * scale
	ox := (float64(sw) - dx) * 0.5
	oy := (float64(sh) - dy) * 0.5

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(ox, oy)
	op.GeoM.Scale(scale, scale)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(g.mapImg, op)

	g.drawMapArchers(screen, ox, oy, scale)
	g.drawCombat(screen, ox, oy, scale)
	g.drawSummonerLaser(screen, ox, oy, scale)

	if g.shopOpen {
		g.drawRecruitPanel(screen)
	}
	g.drawShopToggle(screen)
}

func (g *Game) drawCombat(screen *ebiten.Image, ox, oy, mapScale float64) {
	g.combatMu.RLock()
	defer g.combatMu.RUnlock()
	g.drawEnemies(screen, ox, oy, mapScale)
	g.drawEnemyHealthBars(screen, ox, oy, mapScale)
	g.drawArrows(screen, ox, oy, mapScale)
	drawGoldFloats(screen, ox, oy, mapScale, g.goldFloats)
}

func (g *Game) drawMapArchers(screen *ebiten.Image, ox, oy, scale float64) {
	s := scale * mapArcherSpriteScale
	for i := range g.worldArchers {
		a := &g.worldArchers[i]
		var fr *ebiten.Image
		if a.attackAnim != nil && !a.attackAnim.finished {
			fr = a.attackAnim.frame()
		} else if a.idleAnim != nil {
			fr = a.idleAnim.frame()
		}
		if fr == nil {
			continue
		}
		fw, fh := fr.Bounds().Dx(), fr.Bounds().Dy()
		sx := ox + a.x*scale
		sy := oy + a.y*scale
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(fw)/2, -float64(fh))
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(sx, sy)
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(fr, op)

		if g.shopOpen {
			targetIdx := g.selectedArcherIdx
			if targetIdx < 0 && len(g.worldArchers) > 0 {
				targetIdx = len(g.worldArchers) - 1
			}
			if i == targetIdx {
				r := float32(18 * s)
				vector.StrokeCircle(screen, float32(sx), float32(sy-float64(fh)*s*0.5), r, 2.0, color.RGBA{R: 0xee, G: 0xd4, B: 0x6a, A: 0xd0}, false)
			}
		}
	}
}

// Layout implements ebiten.Game.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return world.ViewportWidth, world.ViewportHeight
}
