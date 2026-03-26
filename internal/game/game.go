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

// Shop layout constants (package-level so they can be shared with recruit_ui.go and NewGame).
const (
	shopCardX      = 24
	shopCardY      = 48
	shopCardW      = 528
	shopCardH      = 500
	shopInnerPad   = 16
	shopDividerOff = 260 // px from card left
	upgradeBtnH    = 50
	upgradeBtnGap  = 10

	// Tab bar sits just below the title/divider line.
	shopTabBarTop  = shopCardY + 38 // 86
	shopTabBarH    = 28
	shopContentTop = shopTabBarTop + shopTabBarH // 114
	shopSlotH      = upgradeBtnH + upgradeBtnGap  // 60

	// Derived column bounds.
	shopInnerL      = shopCardX + shopInnerPad           // 40
	shopDividerX    = shopCardX + shopDividerOff          // 284
	shopLeftInnerR  = shopDividerX - 8                   // 276
	shopRightInnerL = shopDividerX + 8                   // 292
	shopInnerR      = shopCardX + shopCardW - shopInnerPad // 536
)

// shopTab identifies which panel is shown in the shop overlay.
type shopTab int

const (
	tabCombat shopTab = iota
	tabEconomy
	tabAutomation
	tabPrestige
	tabCount
)

var tabLabels = [tabCount]string{"Combat", "Economy", "Auto", "Prestige"}

type worldArcher struct {
	idleAnim   *gifAnim
	attackAnim *gifAnim
	tierIdx    int
	x, y       float64

	pendingTargetID uint64
	hasFired        bool
}

// Game implements ebiten.Game: map, hired archers, optional recruit shop.
// combatMu guards enemies, arrows, spawn/fire timers, and playTime.
type Game struct {
	mapImg *ebiten.Image

	attackPreview *gifAnim

	archerTierRuntimes []archerTierRuntime
	shopArcherPreview  *gifAnim

	archerSpawn *spawn.ArcherSpawner
	lavaMask    *lava.LavaMask

	gold            float64
	metaUpgradeRank [metaUpgradeCount]int
	extraUpgradeRank [extraUpgradeCount]int

	enemyTypesUnlocked int

	worldArchers      []worldArcher
	selectedArcherIdx int

	combatMu    sync.RWMutex
	playTime    float64
	nextEnemyID uint64

	enemies       []worldEnemy
	arrows        []worldArrow
	enemyVariants []enemyVariant
	spawnAccum    float64
	fireAccum     float64

	goldFloats []goldFloat

	// --- Shop UI rects ---
	cardRect image.Rectangle

	// Tab bar
	activeTab shopTab
	tabRects  [tabCount]image.Rectangle
	tabHover  [tabCount]bool

	// Combat tab — LEFT
	portraitRect      image.Rectangle
	buyBtnRect        image.Rectangle
	buyHovered        bool
	archerTrainRect   image.Rectangle // upgradeBtnRects[metaArcherTier] equivalent
	archerTrainHovered bool

	// Combat tab — RIGHT (laser upgrades + extra combat upgrades)
	// [0]=metaLaserDamage, [1]=metaLaserRadius, [2]=extraArrowDamage,
	// [3]=extraArrowSpeed, [4]=extraArcherCooldown, [5]=extraLavaHeat
	combatRightRects   [6]image.Rectangle
	combatRightHovered [6]bool

	// Economy tab — LEFT ([0]=metaScavenger, [1]=metaSpawnWaveSize, [2]=enemyUnlock)
	ecoLeftRects   [3]image.Rectangle
	ecoLeftHovered [3]bool

	// Economy tab — RIGHT ([0]=extraGoldBonus, [1]=extraWaveValue, [2]=extraLootRadius)
	ecoRightRects   [3]image.Rectangle
	ecoRightHovered [3]bool

	// Automation tab — LEFT ([0]=autoScribeBuy, [1]=extraAutoScribeSpeed,
	//                         [2]=extraAutoScribeDiscount, [3]=extraAutoWave)
	autoLeftRects   [4]image.Rectangle
	autoLeftHovered [4]bool

	// Prestige tab — LEFT
	prestigeActionRect  image.Rectangle // "Prestige now" or soul-display
	prestigeActionHover bool
	prestigeConfirmRect  image.Rectangle
	prestigeConfirmHover bool

	// Prestige tab — RIGHT (soul upgrades)
	soulBtnRects   [soulUpgradeCount]image.Rectangle
	soulBtnHovered [soulUpgradeCount]bool

	// Shop toggle button (always visible)
	shopOpen        bool
	shopToggleRect  image.Rectangle
	shopToggleHover bool

	// Stats panel toggle
	statsToggleRect  image.Rectangle
	statsToggleHover bool
	statsOpen        bool

	rng *rand.Rand

	// lastFrameTime is used with time.Now for frame delta; do not use ebiten.ActualTPS for dt.
	lastFrameTime time.Time

	// Summoner laser
	laserWorldX float64
	laserWorldY float64
	laserPhase  float64

	// Speed control
	speedIndex   int
	speedBtnRects [3]image.Rectangle
	speedBtnHover [3]bool

	// GPS tracking (gold per second estimate)
	gpsAccum      float64
	gpsSample     float64
	gpsSampleTimer float64

	// Offline / notification messages
	offlineMessage      string
	offlineMessageTimer float64

	// Auto-save
	autoSaveAccum float64

	// Prestige confirm flow
	prestigeConfirmPending bool
	prestigeConfirmTimer   float64

	// Achievement toasts queue
	achievementToasts []achievementToast

	// Subsystem state
	stats      statsState
	prestige   prestigeData
	achievementsEarned []bool
	achievementMuls    achievementMuls
	autoMgr    autoManagerState
	boss       bossState
	waveCounter int
}

const (
	arrowPackCellSize = 1024
	arrowFlightCols   = 3
	enemyWalkFrameW   = 48
	enemyWalkFrameH   = 48
)

var tierArrowRow = []int{0, 0, 1, 2, 2, 3}

// NewGame decodes map PNG, arrow sprite sheet, enemy strips, and all archer tier GIF pairs.
func NewGame(mapPNG, arrowsPNG []byte, enemyStrips []EnemyStripPair, archerTierAssets []ArcherTierAssets) (*Game, error) {
	img, _, err := image.Decode(bytes.NewReader(mapPNG))
	if err != nil {
		return nil, err
	}

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

	// --- Rect layout ---
	// Helper to compute a slot rect in the left or right column.
	leftSlot := func(n int) image.Rectangle {
		top := shopContentTop + n*shopSlotH
		return image.Rect(shopInnerL, top, shopLeftInnerR, top+upgradeBtnH)
	}
	rightSlot := func(n int) image.Rectangle {
		top := shopContentTop + n*shopSlotH
		return image.Rect(shopRightInnerL, top, shopInnerR, top+upgradeBtnH)
	}

	// Tab bar: 4 equal-width tabs.
	const tabW = shopCardW / int(tabCount) // 132
	var tabRects [tabCount]image.Rectangle
	for i := range tabCount {
		tabRects[i] = image.Rect(
			shopCardX+int(i)*tabW, shopTabBarTop,
			shopCardX+int(i+1)*tabW, shopTabBarTop+shopTabBarH,
		)
	}

	// Portrait (combat tab left).
	const portW, portH = 120, 120
	portTop := shopContentTop + 4
	portX := shopInnerL + (shopLeftInnerR-shopInnerL-portW)/2
	portraitRect := image.Rect(portX, portTop, portX+portW, portTop+portH)

	// Buy button and archer training (combat tab left).
	const (
		nameAreaH = 46 // space for name + sub text below portrait
		btnH      = 46
		gap       = 8
	)
	buyTop := portTop + portH + nameAreaH
	archerTrainTop := buyTop + btnH + gap
	buyBtnRect := image.Rect(shopInnerL, buyTop, shopLeftInnerR, buyTop+btnH)
	archerTrainRect := image.Rect(shopInnerL, archerTrainTop, shopLeftInnerR, archerTrainTop+upgradeBtnH)

	// Combat tab right: [0]=metaLaserDamage, [1]=metaLaserRadius, [2..5]=extra combat upgrades.
	var combatRightRects [6]image.Rectangle
	for i := range 6 {
		combatRightRects[i] = rightSlot(i)
	}

	// Economy tab left: [0]=metaScavenger, [1]=metaSpawnWaveSize, [2]=enemyUnlock.
	var ecoLeftRects [3]image.Rectangle
	for i := range 3 {
		ecoLeftRects[i] = leftSlot(i)
	}

	// Economy tab right: [0]=extraGoldBonus, [1]=extraWaveValue, [2]=extraLootRadius.
	var ecoRightRects [3]image.Rectangle
	for i := range 3 {
		ecoRightRects[i] = rightSlot(i)
	}

	// Automation tab left: [0]=autoScribeBuy, [1]=extraAutoScribeSpeed,
	//                       [2]=extraAutoScribeDiscount, [3]=extraAutoWave.
	var autoLeftRects [4]image.Rectangle
	for i := range 4 {
		autoLeftRects[i] = leftSlot(i)
	}

	// Prestige tab left.
	prestigeActionRect := leftSlot(1)  // slot 0 = soul balance display, slot 1 = prestige button
	prestigeConfirmRect := leftSlot(2) // confirm button (conditional)

	// Prestige tab right: soul upgrades.
	var soulBtnRects [soulUpgradeCount]image.Rectangle
	for i := range soulUpgradeCount {
		soulBtnRects[i] = rightSlot(i)
	}

	// Shop toggle and stats toggle (always visible, top-left).
	const (
		toggleW = 112
		toggleH = 36
		toggleY = 16
	)
	shopToggleRect := image.Rect(24, toggleY, 24+toggleW, toggleY+toggleH)
	statsToggleRect := image.Rect(24+toggleW+8, toggleY, 24+toggleW+8+80, toggleY+toggleH)

	// Speed button rects (HUD top-right panel).
	const (
		hudPanelX = 1440 - 230 - 8 // 1202
		hudPanelY = 8
		hudLineH  = 16
		hudPadX   = 10
	)
	hudSpeedY := hudPanelY + 14 + 3*hudLineH + 4 // 74
	const hudBtnW, hudBtnH = 44, 22
	var speedBtnRects [3]image.Rectangle
	for i := range 3 {
		bx := hudPanelX + hudPadX + i*(hudBtnW+4)
		speedBtnRects[i] = image.Rect(bx, hudSpeedY, bx+hudBtnW, hudSpeedY+hudBtnH)
	}

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
		tabRects:           tabRects,
		portraitRect:       portraitRect,
		buyBtnRect:         buyBtnRect,
		archerTrainRect:    archerTrainRect,
		combatRightRects:   combatRightRects,
		ecoLeftRects:       ecoLeftRects,
		ecoRightRects:      ecoRightRects,
		autoLeftRects:      autoLeftRects,
		prestigeActionRect: prestigeActionRect,
		prestigeConfirmRect: prestigeConfirmRect,
		soulBtnRects:       soulBtnRects,
		shopToggleRect:     shopToggleRect,
		statsToggleRect:    statsToggleRect,
		speedBtnRects:      speedBtnRects,
		shopOpen:           false,
		selectedArcherIdx:  -1,
		rng:                rand.New(rand.NewPCG(0x49444c45, 0x4b494e47)),
		laserWorldX:        laserAX,
		laserWorldY:        laserAY,
		achievementsEarned: make([]bool, achievementCount),
		boss:               bossState{nextBossWave: bossWaveInterval},
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
	rawDt := g.frameDT()
	dt := rawDt * speedMultipliers[g.speedIndex]

	// Lifetime play time (unscaled — real time spent, not simulated time).
	g.stats.lifetimePlaySec += rawDt

	// Track peak gold for prestige starting bonus.
	if g.gold > g.prestige.peakGold {
		g.prestige.peakGold = g.gold
	}

	// Laser phase and aim use gameplay dt.
	g.laserPhase += dt * laserPulseSpeed
	g.updateLaserAim()

	// Shop preview and archer animations use raw dt (visual, not gameplay-scaled).
	if g.shopArcherPreview != nil {
		g.shopArcherPreview.update(rawDt)
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
		g.attackPreview.update(rawDt * 1.35)
		if g.attackPreview.finished {
			g.attackPreview = nil
		}
	}

	// Auto-manager (uses gameplay dt so it benefits from speed multiplier).
	g.updateAutoManager(dt)

	// Combat simulation.
	g.updateCombat(dt)

	// Achievement checks (after combat so stats counters are updated).
	g.checkAchievements()

	// GPS sampling.
	g.gpsSampleTimer += rawDt
	if g.gpsSampleTimer >= 3.0 {
		g.gpsSample = g.gpsAccum / g.gpsSampleTimer
		if g.gpsSample > g.stats.maxGoldPerSec {
			g.stats.maxGoldPerSec = g.gpsSample
		}
		g.gpsAccum = 0
		g.gpsSampleTimer = 0
	}

	// Offline message countdown (real time).
	if g.offlineMessageTimer > 0 {
		g.offlineMessageTimer -= rawDt
	}

	// Prestige confirm timeout (real time).
	if g.prestigeConfirmPending {
		g.prestigeConfirmTimer -= rawDt
		if g.prestigeConfirmTimer <= 0 {
			g.prestigeConfirmPending = false
		}
	}

	// Achievement toast queue countdown (real time).
	for len(g.achievementToasts) > 0 {
		g.achievementToasts[0].timer -= rawDt
		if g.achievementToasts[0].timer <= 0 {
			g.achievementToasts = g.achievementToasts[1:]
		} else {
			break
		}
	}

	// Auto-save (real time, every 30s). Run synchronously on the game thread to
	// avoid data races — marshal+write is fast enough not to cause frame drops.
	g.autoSaveAccum += rawDt
	if g.autoSaveAccum >= 30.0 {
		g.autoSaveAccum = 0
		_ = g.Save()
	}

	// --- Hover detection ---
	g.shopToggleHover = cursorIn(g.shopToggleRect)
	g.statsToggleHover = cursorIn(g.statsToggleRect)
	for i := range g.speedBtnRects {
		g.speedBtnHover[i] = cursorIn(g.speedBtnRects[i])
	}

	if g.shopOpen {
		for i := range g.tabRects {
			g.tabHover[i] = cursorIn(g.tabRects[i])
		}
		// Per-tab hover.
		g.buyHovered = g.activeTab == tabCombat && cursorIn(g.buyBtnRect)
		g.archerTrainHovered = g.activeTab == tabCombat && cursorIn(g.archerTrainRect)
		for i := range g.combatRightHovered {
			g.combatRightHovered[i] = g.activeTab == tabCombat && cursorIn(g.combatRightRects[i])
		}
		for i := range g.ecoLeftHovered {
			g.ecoLeftHovered[i] = g.activeTab == tabEconomy && cursorIn(g.ecoLeftRects[i])
		}
		for i := range g.ecoRightHovered {
			g.ecoRightHovered[i] = g.activeTab == tabEconomy && cursorIn(g.ecoRightRects[i])
		}
		for i := range g.autoLeftHovered {
			g.autoLeftHovered[i] = g.activeTab == tabAutomation && cursorIn(g.autoLeftRects[i])
		}
		g.prestigeActionHover = g.activeTab == tabPrestige && cursorIn(g.prestigeActionRect)
		g.prestigeConfirmHover = g.activeTab == tabPrestige && g.prestigeConfirmPending && cursorIn(g.prestigeConfirmRect)
		for i := range g.soulBtnHovered {
			g.soulBtnHovered[i] = g.activeTab == tabPrestige && cursorIn(g.soulBtnRects[i])
		}
	}

	// --- Click handling ---
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Speed buttons (always active, outside shop).
		for i := range g.speedBtnRects {
			if g.speedBtnHover[i] {
				g.speedIndex = i
			}
		}

		// Stats toggle.
		if g.statsToggleHover {
			g.statsOpen = !g.statsOpen
		}

		// Shop toggle.
		if g.shopToggleHover {
			g.shopOpen = !g.shopOpen
			if !g.shopOpen {
				g.selectedArcherIdx = -1
				g.statsOpen = false
			}
		} else if g.shopOpen {
			// Tab bar clicks.
			handled := false
			for i := range g.tabRects {
				if g.tabHover[i] {
					g.activeTab = shopTab(i)
					handled = true
					break
				}
			}
			if !handled {
				g.handleShopTabClick()
			}
		}
	}

	return nil
}

// handleShopTabClick routes a mouse click to the active shop tab's buttons.
func (g *Game) handleShopTabClick() {
	switch g.activeTab {
	case tabCombat:
		g.handleCombatTabClick()
	case tabEconomy:
		g.handleEconomyTabClick()
	case tabAutomation:
		g.handleAutomationTabClick()
	case tabPrestige:
		g.handlePrestigeTabClick()
	}
	// Map click to select archer (only when NOT on any button).
	if !cursorIn(g.cardRect) {
		g.selectedArcherIdx = g.findArcherUnderCursor()
	}
}

func (g *Game) handleCombatTabClick() {
	// Hire archer button.
	if g.buyHovered {
		cost := g.selectedArcherHireCost()
		if cost > 0 && g.gold >= float64(cost) {
			tier := g.metaUpgradeRank[metaArcherTier]
			if tier >= 0 && tier < len(g.archerTierRuntimes) {
				g.hireOneArcher()
				// Play attack preview.
				tr := &g.archerTierRuntimes[tier]
				shot, err := newGIFAnim(tr.attackBytes, false)
				if err == nil {
					g.attackPreview = shot
				}
			}
		}
		return
	}
	// Archer training upgrade (meta).
	if g.archerTrainHovered {
		cost := g.metaUpgradeNextCost(metaArcherTier)
		if cost > 0 && g.gold >= float64(cost) {
			g.gold -= float64(cost)
			g.metaUpgradeRank[metaArcherTier]++
			g.updateAllArchersToCurrentTier()
			g.refreshShopArcherPreview()
		}
		return
	}
	// Combat right: [0]=metaLaserDamage, [1]=metaLaserRadius, [2..5]=extra upgrades.
	metaCombatRight := [2]int{metaLaserDamage, metaLaserRadius}
	for i, mid := range metaCombatRight {
		if g.combatRightHovered[i] {
			cost := g.metaUpgradeNextCost(mid)
			if cost > 0 && g.gold >= float64(cost) {
				g.gold -= float64(cost)
				g.metaUpgradeRank[mid]++
			}
			return
		}
	}
	extraCombatRight := [4]int{extraArrowDamage, extraArrowSpeed, extraArcherCooldown, extraLavaHeat}
	for i, eid := range extraCombatRight {
		if g.combatRightHovered[i+2] {
			cost := extraUpgradeNextCost(eid, g.extraUpgradeRank[eid])
			if cost > 0 && g.gold >= float64(cost) {
				g.gold -= float64(cost)
				g.extraUpgradeRank[eid]++
			}
			return
		}
	}
}

func (g *Game) handleEconomyTabClick() {
	// Economy left: [0]=metaScavenger, [1]=metaSpawnWaveSize, [2]=enemyUnlock.
	metaEcoLeft := [2]int{metaScavenger, metaSpawnWaveSize}
	for i, mid := range metaEcoLeft {
		if g.ecoLeftHovered[i] {
			cost := g.metaUpgradeNextCost(mid)
			if cost > 0 && g.gold >= float64(cost) {
				g.gold -= float64(cost)
				g.metaUpgradeRank[mid]++
			}
			return
		}
	}
	// Enemy unlock.
	if g.ecoLeftHovered[2] {
		cost := g.enemyUnlockNextCost()
		if cost > 0 && g.gold >= float64(cost) {
			g.gold -= float64(cost)
			g.enemyTypesUnlocked++
		}
		return
	}
	// Economy right: extra economy upgrades.
	extraEcoRight := [3]int{extraGoldBonus, extraWaveValue, extraLootRadius}
	for i, eid := range extraEcoRight {
		if g.ecoRightHovered[i] {
			cost := extraUpgradeNextCost(eid, g.extraUpgradeRank[eid])
			if cost > 0 && g.gold >= float64(cost) {
				g.gold -= float64(cost)
				g.extraUpgradeRank[eid]++
			}
			return
		}
	}
}

func (g *Game) handleAutomationTabClick() {
	// Auto-scribe buy (slot 0).
	if g.autoLeftHovered[0] {
		if !g.autoMgr.purchased && g.gold >= autoScribeBaseCost {
			g.gold -= autoScribeBaseCost
			g.autoMgr.purchased = true
		}
		return
	}
	// Extra automation upgrades.
	extraAutoLeft := [3]int{extraAutoScribeSpeed, extraAutoScribeDiscount, extraAutoWave}
	for i, eid := range extraAutoLeft {
		if g.autoLeftHovered[i+1] {
			if !g.autoMgr.purchased {
				return // requires purchase first
			}
			cost := extraUpgradeNextCost(eid, g.extraUpgradeRank[eid])
			if cost > 0 && g.gold >= float64(cost) {
				g.gold -= float64(cost)
				g.extraUpgradeRank[eid]++
			}
			return
		}
	}
}

func (g *Game) handlePrestigeTabClick() {
	// Prestige confirm flow.
	if g.prestigeConfirmHover {
		g.doPrestige()
		return
	}
	if g.prestigeActionHover {
		souls := g.soulsOnPrestige()
		if souls >= 1 {
			g.prestigeConfirmPending = true
			g.prestigeConfirmTimer = 3.0
		}
		return
	}
	// Soul upgrades.
	for i := range g.soulBtnRects {
		if g.soulBtnHovered[i] {
			cost := soulUpgradeNextCost(i, g.prestige.soulRanks[i])
			if cost > 0 && g.prestige.souls >= cost {
				g.prestige.souls -= cost
				g.prestige.soulRanks[i]++
			}
			return
		}
	}
}

// frameDT returns real elapsed time since the last Update, capped to avoid huge jumps.
func (g *Game) frameDT() float64 {
	const (
		defaultStep = 1.0 / 60.0
		maxStep     = 0.12
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
		if g.statsOpen {
			g.drawStatsPanel(screen)
		}
	}
	g.drawShopToggle(screen)
	g.drawHUD(screen)
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
