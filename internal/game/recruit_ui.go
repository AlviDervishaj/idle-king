package game

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
)

var (
	colRecruitShadow   = color.RGBA{0x08, 0x0a, 0x0f, 0x72}
	colRecruitCard     = color.RGBA{0x1a, 0x22, 0x32, 0xf2}
	colRecruitCardEdge = color.RGBA{0x6b, 0x58, 0x38, 0xff}
	colRecruitGold     = color.RGBA{0xee, 0xd4, 0x6a, 0xff}
	colRecruitText     = color.RGBA{0xee, 0xe8, 0xdc, 0xff}
	colRecruitMuted    = color.RGBA{0x8b, 0x92, 0xa8, 0xff}
	colRecruitFrame    = color.RGBA{0x0c, 0x0f, 0x16, 0xff}
	colRecruitFrameHi  = color.RGBA{0x4a, 0x3d, 0x28, 0xff}
	colRecruitBtn      = color.RGBA{0x2a, 0x5c, 0x42, 0xff}
	colRecruitBtnHi    = color.RGBA{0x36, 0x74, 0x54, 0xff}
	colRecruitBtnLo    = color.RGBA{0x1e, 0x42, 0x30, 0xff}
	colRecruitBtnDis   = color.RGBA{0x35, 0x3c, 0x48, 0xff}
	colRecruitBtnEdge  = color.RGBA{0x5a, 0x8a, 0x6a, 0xff}
	colPrestigeBtn     = color.RGBA{0x4a, 0x28, 0x6a, 0xff}
	colPrestigeBtnHi   = color.RGBA{0x62, 0x38, 0x8a, 0xff}
	colSoulBtn         = color.RGBA{0x28, 0x28, 0x62, 0xff}
	colSoulBtnHi       = color.RGBA{0x38, 0x38, 0x80, 0xff}
)

func textW(face font.Face, s string) int {
	return font.MeasureString(face, s).Ceil()
}

func baselineY(face font.Face, top int) int {
	a := face.Metrics().Ascent.Ceil()
	return top + a
}

// affordablePulse returns a 0..1 sine pulse used for the gold border on affordable buttons.
func affordablePulse(t float64) float32 {
	return float32(0.5 + 0.5*math.Sin(t*3.0))
}

// drawRecruitPanel renders the full shop overlay (card + active tab content).
func (g *Game) drawRecruitPanel(screen *ebiten.Image) {
	cx := float32(g.cardRect.Min.X)
	cy := float32(g.cardRect.Min.Y)
	cw := float32(g.cardRect.Dx())
	ch := float32(g.cardRect.Dy())

	// Card shadow + background.
	vector.FillRect(screen, cx+5, cy+7, cw, ch, colRecruitShadow, true)
	vector.FillRect(screen, cx, cy, cw, ch, colRecruitCard, true)
	vector.StrokeRect(screen, cx+1, cy+1, cw-2, ch-2, 2, colRecruitCardEdge, true)

	// Vertical divider.
	divX := float32(shopDividerX)
	vector.StrokeLine(screen, divX, cy+float32(shopTabBarTop-shopCardY), divX, cy+ch-14, 1.5, colRecruitFrameHi, true)

	innerL := shopCardX + shopInnerPad
	innerR := shopCardX + shopCardW - shopInnerPad

	// Title row.
	titleY := baselineY(bitmapfont.Face, shopCardY+14)
	text.Draw(screen, "SHOP", bitmapfont.Face, innerL, titleY, colRecruitGold)
	goldStr := formatGold(g.gold) + " gold"
	text.Draw(screen, goldStr, bitmapfont.Face, innerR-textW(bitmapfont.Face, goldStr), titleY, colRecruitGold)

	// Divider below title.
	divY := float32(shopCardY + 38)
	vector.StrokeLine(screen, float32(innerL), divY, float32(innerR), divY, 1, colRecruitFrameHi, true)

	// Tab bar.
	g.drawTabBar(screen)

	// Tab content.
	switch g.activeTab {
	case tabCombat:
		g.drawCombatTab(screen)
	case tabEconomy:
		g.drawEconomyTab(screen)
	case tabAutomation:
		g.drawAutomationTab(screen)
	case tabPrestige:
		g.drawPrestigeTab(screen)
	}
}

func (g *Game) drawTabBar(screen *ebiten.Image) {
	for i, r := range g.tabRects {
		active := g.activeTab == shopTab(i)
		hovered := g.tabHover[i]
		var fill color.Color
		if active {
			fill = color.RGBA{R: 0x28, G: 0x36, B: 0x52, A: 0xff}
		} else if hovered {
			fill = color.RGBA{R: 0x22, G: 0x2c, B: 0x44, A: 0xff}
		} else {
			fill = color.RGBA{R: 0x16, G: 0x1e, B: 0x30, A: 0xff}
		}
		bx, by := float32(r.Min.X), float32(r.Min.Y)
		bw, bh := float32(r.Dx()), float32(r.Dy())
		vector.FillRect(screen, bx, by, bw, bh, fill, true)
		edgeCol := colRecruitFrameHi
		if active {
			edgeCol = colRecruitGold
		}
		vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1.5, edgeCol, true)
		label := tabLabels[i]
		lw := textW(bitmapfont.Face, label)
		lx := r.Min.X + (r.Dx()-lw)/2
		ly := baselineY(bitmapfont.Face, r.Min.Y+(r.Dy()-bitmapfont.Face.Metrics().Height.Ceil())/2)
		lc := colRecruitMuted
		if active {
			lc = colRecruitGold
		}
		text.Draw(screen, label, bitmapfont.Face, lx, ly, lc)
	}
}

// --- COMBAT TAB ---

func (g *Game) drawCombatTab(screen *ebiten.Image) {
	// Portrait frame.
	px := float32(g.portraitRect.Min.X)
	py := float32(g.portraitRect.Min.Y)
	pw := float32(g.portraitRect.Dx())
	ph := float32(g.portraitRect.Dy())
	vector.FillRect(screen, px, py, pw, ph, colRecruitFrame, true)
	vector.StrokeRect(screen, px+1, py+1, pw-2, ph-2, 1, colRecruitFrameHi, true)

	var src *ebiten.Image
	if g.attackPreview != nil {
		src = g.attackPreview.frame()
	} else if g.shopArcherPreview != nil {
		src = g.shopArcherPreview.frame()
	}
	if src != nil {
		fw, fh := src.Bounds().Dx(), src.Bounds().Dy()
		const pad = 10
		maxInner := float64(g.portraitRect.Dx() - 2*pad)
		maxH := float64(g.portraitRect.Dy() - 2*pad)
		s := maxInner / float64(fw)
		if float64(fh)*s > maxH {
			s = maxH / float64(fh)
		}
		dw := float64(fw) * s
		dh := float64(fh) * s
		px0 := float64(g.portraitRect.Min.X) + (float64(g.portraitRect.Dx())-dw)*0.5
		py0 := float64(g.portraitRect.Min.Y) + (float64(g.portraitRect.Dy())-dh)*0.5
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(px0, py0)
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(src, op)
	}

	leftColCenter := (shopInnerL + shopLeftInnerR) / 2
	nameY := baselineY(bitmapfont.Face, g.portraitRect.Max.Y+8)
	text.Draw(screen, "Archers", bitmapfont.Face, leftColCenter-textW(bitmapfont.Face, "Archers")/2, nameY, colRecruitText)

	subY := baselineY(bitmapfont.Face, g.portraitRect.Max.Y+26)
	sub := "Ranged unit"
	tIdx := g.metaUpgradeRank[metaArcherTier]
	if tIdx >= 0 && tIdx < len(g.archerTierRuntimes) {
		sub = g.archerTierRuntimes[tIdx].name + " tier"
	}
	text.Draw(screen, sub, bitmapfont.Face, leftColCenter-textW(bitmapfont.Face, sub)/2, subY, colRecruitMuted)

	g.drawRecruitBuyButton(screen)
	g.drawGenericUpgradeBtn(screen, g.archerTrainRect, g.archerTrainHovered,
		"Archer Training", archerTrainingDesc(g), g.metaUpgradeNextCost(metaArcherTier),
		g.metaUpgradeRank[metaArcherTier])

	squad := fmt.Sprintf("In squad: %d", len(g.worldArchers))
	sqY := baselineY(bitmapfont.Face, g.archerTrainRect.Max.Y+10)
	text.Draw(screen, squad, bitmapfont.Face, leftColCenter-textW(bitmapfont.Face, squad)/2, sqY, colRecruitMuted)

	// Combat right column label.
	rightL := shopRightInnerL
	titleY := baselineY(bitmapfont.Face, shopContentTop-16)
	text.Draw(screen, "COMBAT", bitmapfont.Face, rightL, titleY, colRecruitGold)

	// Laser upgrades (first two slots).
	g.drawGenericUpgradeBtn(screen, g.combatRightRects[0], g.combatRightHovered[0],
		"Beam Damage", fmt.Sprintf("Laser DPS +%d", 15*(g.metaUpgradeRank[metaLaserDamage]+1)),
		g.metaUpgradeNextCost(metaLaserDamage), g.metaUpgradeRank[metaLaserDamage])
	g.drawGenericUpgradeBtn(screen, g.combatRightRects[1], g.combatRightHovered[1],
		"Beam Radius", fmt.Sprintf("Laser area +%dpx", 4*(g.metaUpgradeRank[metaLaserRadius]+1)),
		g.metaUpgradeNextCost(metaLaserRadius), g.metaUpgradeRank[metaLaserRadius])

	// Extra combat upgrades.
	extraCombatRight := [4]int{extraArrowDamage, extraArrowSpeed, extraArcherCooldown, extraLavaHeat}
	for i, eid := range extraCombatRight {
		rank := g.extraUpgradeRank[eid]
		cost := extraUpgradeNextCost(eid, rank)
		g.drawGenericUpgradeBtn(screen, g.combatRightRects[i+2], g.combatRightHovered[i+2],
			extraUpgradeTitles[eid], extraUpgradeDesc[eid], cost, rank)
	}
}

func archerTrainingDesc(g *Game) string {
	tIdx := g.metaUpgradeRank[metaArcherTier]
	if tIdx >= archerTierCount-1 {
		return "Max tier reached"
	}
	if tIdx+1 < len(ArcherTierNames) {
		return "Unlock " + ArcherTierNames[tIdx+1]
	}
	return "Upgrade all archers"
}

// --- ECONOMY TAB ---

func (g *Game) drawEconomyTab(screen *ebiten.Image) {
	rightL := shopRightInnerL
	titleY := baselineY(bitmapfont.Face, shopContentTop-16)
	text.Draw(screen, "ECONOMY", bitmapfont.Face, rightL, titleY, colRecruitGold)
	leftL := shopInnerL
	text.Draw(screen, "CONTRACTS", bitmapfont.Face, leftL, titleY, colRecruitGold)

	// Economy left: meta upgrades + enemy unlock.
	g.drawGenericUpgradeBtn(screen, g.ecoLeftRects[0], g.ecoLeftHovered[0],
		"Scavenger Crew", "+15% kill gold",
		g.metaUpgradeNextCost(metaScavenger), g.metaUpgradeRank[metaScavenger])
	g.drawGenericUpgradeBtn(screen, g.ecoLeftRects[1], g.ecoLeftHovered[1],
		"Enemy Density", "Enemies per wave",
		g.metaUpgradeNextCost(metaSpawnWaveSize), g.metaUpgradeRank[metaSpawnWaveSize])
	g.drawEnemyUnlockButton(screen)

	// Economy right: extra economy upgrades.
	extraEcoRight := [3]int{extraGoldBonus, extraWaveValue, extraLootRadius}
	for i, eid := range extraEcoRight {
		rank := g.extraUpgradeRank[eid]
		cost := extraUpgradeNextCost(eid, rank)
		g.drawGenericUpgradeBtn(screen, g.ecoRightRects[i], g.ecoRightHovered[i],
			extraUpgradeTitles[eid], extraUpgradeDesc[eid], cost, rank)
	}
}

// --- AUTOMATION TAB ---

func (g *Game) drawAutomationTab(screen *ebiten.Image) {
	leftL := shopInnerL
	titleY := baselineY(bitmapfont.Face, shopContentTop-16)
	text.Draw(screen, "AUTOMATION", bitmapfont.Face, leftL, titleY, colRecruitGold)

	// Slot 0: Auto-Scribe buy/status.
	r := g.autoLeftRects[0]
	if !g.autoMgr.purchased {
		g.drawGenericUpgradeBtn(screen, r, g.autoLeftHovered[0],
			"Auto-Scribe", "Auto-hire archers",
			autoScribeBaseCost, 0)
	} else {
		g.drawInfoBox(screen, r,
			fmt.Sprintf("Auto-Scribe  Rank %d", g.autoMgr.rank),
			fmt.Sprintf("Hires when gold > %s", formatGold(float64(g.selectedArcherHireCost())*autoScribeThresholdMul)),
		)
	}

	// Slots 1-3: Extra automation upgrades.
	extraAutoLeft := [3]int{extraAutoScribeSpeed, extraAutoScribeDiscount, extraAutoWave}
	for i, eid := range extraAutoLeft {
		rank := g.extraUpgradeRank[eid]
		cost := extraUpgradeNextCost(eid, rank)
		hov := g.autoLeftHovered[i+1]
		if !g.autoMgr.purchased {
			g.drawInfoBox(screen, g.autoLeftRects[i+1], extraUpgradeTitles[eid], "Requires Auto-Scribe")
		} else {
			g.drawGenericUpgradeBtn(screen, g.autoLeftRects[i+1], hov,
				extraUpgradeTitles[eid], extraUpgradeDesc[eid], cost, rank)
		}
	}

	// Right side: Speed info + live stats.
	rightL := shopRightInnerL
	speedTitleY := baselineY(bitmapfont.Face, shopContentTop-16)
	text.Draw(screen, "GAME SPEED", bitmapfont.Face, rightL, speedTitleY, colRecruitGold)
	speedInfoY := baselineY(bitmapfont.Face, shopContentTop+4)
	text.Draw(screen, "Use HUD buttons ->", bitmapfont.Face, rightL, speedInfoY, colRecruitMuted)

	dpsY := baselineY(bitmapfont.Face, shopContentTop+30)
	text.Draw(screen, fmt.Sprintf("DPS: %s", formatGold(g.currentDPS())), bitmapfont.Face, rightL, dpsY, colRecruitText)
	gpsY := baselineY(bitmapfont.Face, shopContentTop+50)
	text.Draw(screen, fmt.Sprintf("Gold/sec: %s", formatGold(g.gpsSample)), bitmapfont.Face, rightL, gpsY, colRecruitText)
	killY := baselineY(bitmapfont.Face, shopContentTop+70)
	text.Draw(screen, fmt.Sprintf("Kills: %s", formatGold(float64(g.stats.totalKills))), bitmapfont.Face, rightL, killY, colRecruitMuted)
}

// --- PRESTIGE TAB ---

func (g *Game) drawPrestigeTab(screen *ebiten.Image) {
	leftL := shopInnerL
	rightL := shopRightInnerL

	titleY := baselineY(bitmapfont.Face, shopContentTop-16)
	text.Draw(screen, "PRESTIGE", bitmapfont.Face, leftL, titleY, colRecruitGold)
	text.Draw(screen, "SOUL UPGRADES", bitmapfont.Face, rightL, titleY, colRecruitGold)

	// Soul balance display (left slot 0).
	soulsBox := image.Rect(shopInnerL, shopContentTop, shopLeftInnerR, shopContentTop+upgradeBtnH)
	soulsPreview := g.soulsOnPrestige()
	g.drawInfoBox(screen, soulsBox,
		fmt.Sprintf("Souls: %.0f", g.prestige.souls),
		fmt.Sprintf("Prestige %dx | Next: +%.0f Souls", g.prestige.totalPrestiges, soulsPreview),
	)

	// Prestige button (left slot 1).
	canPrestige := soulsPreview >= 1
	pBtnFill := colPrestigeBtn
	if !canPrestige {
		pBtnFill = colRecruitBtnDis
	} else if g.prestigeActionHover {
		pBtnFill = colPrestigeBtnHi
	}
	var prestigeSub string
	if !canPrestige {
		prestigeSub = "Need more gold first"
	} else {
		prestigeSub = fmt.Sprintf("+%.0f Souls", soulsPreview)
	}
	g.drawColoredBtn(screen, g.prestigeActionRect, g.prestigeActionHover && canPrestige, pBtnFill,
		"Prestige Now", prestigeSub)

	// Confirm button (left slot 2, only when pending).
	if g.prestigeConfirmPending {
		confirmFill := color.RGBA{R: 0xaa, G: 0x22, B: 0x22, A: 0xff}
		if g.prestigeConfirmHover {
			confirmFill = color.RGBA{R: 0xcc, G: 0x33, B: 0x33, A: 0xff}
		}
		g.drawColoredBtn(screen, g.prestigeConfirmRect, g.prestigeConfirmHover, confirmFill,
			"CONFIRM RESET",
			fmt.Sprintf("%.0fs to cancel", g.prestigeConfirmTimer))
	} else if canPrestige {
		g.drawInfoBox(screen, g.prestigeConfirmRect, "Click 'Prestige Now'", "then confirm to reset")
	}

	// Soul upgrades (right column).
	for i := range soulUpgradeCount {
		rank := g.prestige.soulRanks[i]
		cost := soulUpgradeNextCost(i, rank)
		canAfford := cost > 0 && g.prestige.souls >= cost
		hov := g.soulBtnHovered[i]

		fill := colSoulBtn
		if !canAfford || cost <= 0 {
			fill = colRecruitBtnDis
		} else if hov {
			fill = colSoulBtnHi
		}

		var costStr string
		switch {
		case cost <= 0:
			costStr = "MAX"
		case canAfford:
			costStr = fmt.Sprintf("%.0f Souls", cost)
		default:
			costStr = fmt.Sprintf("Need %.0f", cost)
		}
		g.drawColoredBtnFull(screen, g.soulBtnRects[i], hov && canAfford, fill,
			fmt.Sprintf("%s  Rank %d", soulUpgradeTitles[i], rank),
			soulUpgradeDesc[i],
			costStr)
	}
}

// --- STATS PANEL ---

func (g *Game) drawStatsPanel(screen *ebiten.Image) {
	const sw, sh = 600, 400
	sx := (1440 - sw) / 2
	sy := (720 - sh) / 2

	bg := color.RGBA{R: 0x0e, G: 0x14, B: 0x20, A: 0xf0}
	edge := color.RGBA{R: 0x6b, G: 0x58, B: 0x38, A: 0xff}
	vector.FillRect(screen, float32(sx), float32(sy), sw, sh, bg, true)
	vector.StrokeRect(screen, float32(sx)+1, float32(sy)+1, sw-2, sh-2, 2, edge, true)

	const padX = 20
	innerX := sx + padX
	titleY := baselineY(bitmapfont.Face, sy+16)
	text.Draw(screen, "STATISTICS", bitmapfont.Face, innerX, titleY, colRecruitGold)

	lineH := 18
	y := sy + 40
	g.drawStatLine(screen, innerX, y, "Total Kills", formatGold(float64(g.stats.totalKills)))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Total Gold Earned", formatGold(g.stats.totalGoldEarned))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Peak Gold/sec", formatGold(g.stats.maxGoldPerSec))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Total Waves", fmt.Sprintf("%d", g.stats.totalWaves))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Boss Kills", fmt.Sprintf("%d", g.stats.totalBossKills))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Best Boss Reward", formatGold(g.stats.bestBossGold)+" gold")
	y += lineH
	g.drawStatLine(screen, innerX, y, "Play Time (Run)", fmtDuration(g.playTime))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Play Time (All)", fmtDuration(g.stats.lifetimePlaySec))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Prestige Rank", fmt.Sprintf("%d (%.0f Souls)", g.prestige.rank, g.prestige.souls))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Achievements", fmt.Sprintf("%d/%d", g.achievementsEarnedCount(), achievementCount))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Current DPS", formatGold(g.currentDPS()))
	y += lineH
	g.drawStatLine(screen, innerX, y, "Gold/sec", formatGold(g.gpsSample))
}

func (g *Game) drawStatLine(screen *ebiten.Image, x, y int, label, value string) {
	text.Draw(screen, label+":", bitmapfont.Face, x, baselineY(bitmapfont.Face, y), colRecruitMuted)
	text.Draw(screen, value, bitmapfont.Face, x+200, baselineY(bitmapfont.Face, y), colRecruitText)
}

func fmtDuration(sec float64) string {
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// --- SHARED DRAW HELPERS ---

// drawGenericUpgradeBtn draws a standard upgrade button with title, description, rank and cost.
func (g *Game) drawGenericUpgradeBtn(screen *ebiten.Image, r image.Rectangle, hovered bool, title, desc string, cost, rank int) {
	maxed := cost <= 0 && rank > 0
	canAfford := !maxed && cost > 0 && g.gold >= float64(cost)

	bx, by := float32(r.Min.X), float32(r.Min.Y)
	bw, bh := float32(r.Dx()), float32(r.Dy())

	var fill color.Color = colRecruitBtn
	if maxed {
		fill = color.RGBA{R: 0x22, G: 0x44, B: 0x33, A: 0xff}
	} else if !canAfford {
		fill = colRecruitBtnDis
	} else if hovered {
		fill = colRecruitBtnHi
	}
	vector.FillRect(screen, bx, by, bw, bh, fill, true)

	edge := colRecruitBtnEdge
	if maxed {
		edge = color.RGBA{R: 0x44, G: 0x88, B: 0x66, A: 0xff}
	} else if !canAfford {
		edge = colRecruitFrameHi
	}
	vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1.5, edge, true)

	if canAfford {
		pulse := affordablePulse(g.playTime)
		pEdge := color.RGBA{R: 0xee, G: 0xd4, B: 0x6a, A: uint8(pulse * 100)}
		vector.StrokeRect(screen, bx+2, by+2, bw-4, bh-4, 1.0, pEdge, true)
	}

	innerL := r.Min.X + 10
	innerR := r.Max.X - 10
	titleBaseY := baselineY(bitmapfont.Face, r.Min.Y+6)
	descBaseY := baselineY(bitmapfont.Face, r.Min.Y+24)

	tc := colRecruitText
	if !canAfford && !maxed {
		tc = colRecruitMuted
	}
	text.Draw(screen, title, bitmapfont.Face, innerL, titleBaseY, tc)
	text.Draw(screen, desc, bitmapfont.Face, innerL, descBaseY, colRecruitMuted)

	var rightStr string
	switch {
	case maxed:
		rightStr = "MAX"
	case cost <= 0:
		rightStr = "—"
	case canAfford:
		rightStr = fmt.Sprintf("%s g", formatGold(float64(cost)))
	default:
		rightStr = fmt.Sprintf("Need %s", formatGold(float64(cost)))
	}
	rc := colRecruitGold
	if !canAfford && !maxed {
		rc = colRecruitMuted
	}
	text.Draw(screen, rightStr, bitmapfont.Face, innerR-textW(bitmapfont.Face, rightStr), titleBaseY, rc)

	rankStr := fmt.Sprintf("Rank %d", rank)
	text.Draw(screen, rankStr, bitmapfont.Face, innerR-textW(bitmapfont.Face, rankStr), descBaseY, colRecruitMuted)
}

// drawInfoBox draws a non-interactive info panel showing title and description.
func (g *Game) drawInfoBox(screen *ebiten.Image, r image.Rectangle, title, desc string) {
	bx, by := float32(r.Min.X), float32(r.Min.Y)
	bw, bh := float32(r.Dx()), float32(r.Dy())
	vector.FillRect(screen, bx, by, bw, bh, colRecruitFrame, true)
	vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1, colRecruitFrameHi, true)
	innerL := r.Min.X + 10
	titleY := baselineY(bitmapfont.Face, r.Min.Y+6)
	descY := baselineY(bitmapfont.Face, r.Min.Y+24)
	text.Draw(screen, title, bitmapfont.Face, innerL, titleY, colRecruitText)
	text.Draw(screen, desc, bitmapfont.Face, innerL, descY, colRecruitMuted)
}

// drawColoredBtn draws a two-row button with a custom fill color (title + sub text).
func (g *Game) drawColoredBtn(screen *ebiten.Image, r image.Rectangle, hovered bool, fill color.Color, label, sub string) {
	bx, by := float32(r.Min.X), float32(r.Min.Y)
	bw, bh := float32(r.Dx()), float32(r.Dy())
	vector.FillRect(screen, bx, by, bw, bh, fill, true)
	edge := color.RGBA{R: 0x88, G: 0x66, B: 0xaa, A: 0xff}
	vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1.5, edge, true)
	if hovered {
		vector.StrokeRect(screen, bx+2, by+2, bw-4, bh-4, 1.0,
			color.RGBA{R: 0xcc, G: 0xaa, B: 0xff, A: 0x80}, true)
	}
	innerL := r.Min.X + 10
	titleY := baselineY(bitmapfont.Face, r.Min.Y+6)
	subY := baselineY(bitmapfont.Face, r.Min.Y+24)
	text.Draw(screen, label, bitmapfont.Face, innerL, titleY, colRecruitText)
	text.Draw(screen, sub, bitmapfont.Face, innerL, subY, colRecruitMuted)
}

// drawColoredBtnFull draws a button with title, description and right-aligned cost string.
func (g *Game) drawColoredBtnFull(screen *ebiten.Image, r image.Rectangle, hovered bool, fill color.Color, title, desc, costStr string) {
	bx, by := float32(r.Min.X), float32(r.Min.Y)
	bw, bh := float32(r.Dx()), float32(r.Dy())
	vector.FillRect(screen, bx, by, bw, bh, fill, true)
	edge := color.RGBA{R: 0x44, G: 0x44, B: 0x88, A: 0xff}
	vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1.5, edge, true)
	if hovered {
		vector.StrokeRect(screen, bx+2, by+2, bw-4, bh-4, 1.0,
			color.RGBA{R: 0x88, G: 0x88, B: 0xff, A: 0x80}, true)
	}
	innerL := r.Min.X + 10
	innerR := r.Max.X - 10
	titleY := baselineY(bitmapfont.Face, r.Min.Y+6)
	descY := baselineY(bitmapfont.Face, r.Min.Y+24)
	text.Draw(screen, title, bitmapfont.Face, innerL, titleY, colRecruitText)
	text.Draw(screen, desc, bitmapfont.Face, innerL, descY, colRecruitMuted)
	text.Draw(screen, costStr, bitmapfont.Face, innerR-textW(bitmapfont.Face, costStr), titleY, colRecruitGold)
}

// drawRecruitBuyButton draws the hire-archer button on the combat tab left.
func (g *Game) drawRecruitBuyButton(screen *ebiten.Image) {
	r := g.buyBtnRect
	bx := float32(r.Min.X)
	by := float32(r.Min.Y)
	bw := float32(r.Dx())
	bh := float32(r.Dy())

	cost := g.selectedArcherHireCost()
	canAfford := cost > 0 && g.gold >= float64(cost)
	pressed := g.buyHovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && canAfford

	var fill color.Color = colRecruitBtn
	if !canAfford {
		fill = colRecruitBtnDis
	} else if pressed {
		fill = colRecruitBtnLo
	} else if g.buyHovered {
		fill = colRecruitBtnHi
	}

	vector.FillRect(screen, bx, by, bw, bh, fill, true)
	edge := colRecruitBtnEdge
	if !canAfford {
		edge = colRecruitFrameHi
	}
	vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1.5, edge, true)

	if canAfford {
		pulse := affordablePulse(g.playTime)
		pEdge := color.RGBA{R: 0xee, G: 0xd4, B: 0x6a, A: uint8(pulse * 120)}
		vector.StrokeRect(screen, bx+2, by+2, bw-4, bh-4, 1.5, pEdge, true)
	}

	innerL := r.Min.X + 12
	innerR := r.Max.X - 12
	btnTop := r.Min.Y + (r.Dy()-16)/2
	base := baselineY(bitmapfont.Face, btnTop)

	left := "Hire archer"
	var right string
	if cost <= 0 {
		right = "—"
	} else if canAfford {
		right = fmt.Sprintf("%s gold", formatGold(float64(cost)))
	} else {
		right = fmt.Sprintf("Need %s", formatGold(float64(cost)))
	}
	lc := colRecruitText
	rc := colRecruitGold
	if !canAfford {
		lc = colRecruitMuted
		rc = colRecruitMuted
	}
	text.Draw(screen, left, bitmapfont.Face, innerL, base, lc)
	text.Draw(screen, right, bitmapfont.Face, innerR-textW(bitmapfont.Face, right), base, rc)
}

func (g *Game) drawEnemyUnlockButton(screen *ebiten.Image) {
	r := g.ecoLeftRects[2]
	bx := float32(r.Min.X)
	by := float32(r.Min.Y)
	bw := float32(r.Dx())
	bh := float32(r.Dy())

	cost := g.enemyUnlockNextCost()
	maxed := cost <= 0
	canAfford := !maxed && g.gold >= float64(cost)
	hovered := g.ecoLeftHovered[2]
	pressed := hovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && canAfford

	var fill color.Color = colRecruitBtn
	if maxed || !canAfford {
		fill = colRecruitBtnDis
	} else if pressed {
		fill = colRecruitBtnLo
	} else if hovered {
		fill = colRecruitBtnHi
	}

	vector.FillRect(screen, bx, by, bw, bh, fill, true)
	edge := colRecruitBtnEdge
	if !canAfford {
		edge = colRecruitFrameHi
	}
	vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1.5, edge, true)

	innerL := r.Min.X + 16
	innerR := r.Max.X - 14
	base := baselineY(bitmapfont.Face, r.Min.Y+6)

	total := len(g.enemyVariants)
	if total < 1 {
		total = 1
	}
	left := fmt.Sprintf("Enemy Roster  %d/%d", g.enemyTypesUnlocked, total)
	text.Draw(screen, left, bitmapfont.Face, innerL, base, colRecruitText)

	subY := baselineY(bitmapfont.Face, r.Min.Y+24)
	desc := "Add another spawn type"
	if maxed {
		desc = "All enemy types unlocked"
	}
	text.Draw(screen, desc, bitmapfont.Face, innerL, subY, colRecruitMuted)

	var right string
	switch {
	case maxed:
		right = "—"
	case canAfford:
		right = fmt.Sprintf("%d gold", cost)
	default:
		right = fmt.Sprintf("Need %d", cost)
	}
	rc := colRecruitGold
	if !canAfford {
		rc = colRecruitMuted
	}
	text.Draw(screen, right, bitmapfont.Face, innerR-textW(bitmapfont.Face, right), base, rc)
}

// drawShopToggle renders the Shop and Stats toggle buttons (always visible).
func (g *Game) drawShopToggle(screen *ebiten.Image) {
	g.drawToggleButton(screen, g.shopToggleRect, g.shopToggleHover, g.shopOpen, "Shop")
	g.drawToggleButton(screen, g.statsToggleRect, g.statsToggleHover, g.statsOpen, "Stats")
}

// drawToggleButton renders a toggle button with active state highlight.
func (g *Game) drawToggleButton(screen *ebiten.Image, r image.Rectangle, hovered, active bool, label string) {
	bx, by := float32(r.Min.X), float32(r.Min.Y)
	bw, bh := float32(r.Dx()), float32(r.Dy())

	var fill color.Color = color.RGBA{R: 0x1a, G: 0x22, B: 0x32, A: 0xe8}
	if active {
		fill = color.RGBA{R: 0x28, G: 0x3e, B: 0x5a, A: 0xf0}
	} else if hovered {
		fill = color.RGBA{R: 0x22, G: 0x2c, B: 0x44, A: 0xe8}
	}
	vector.FillRect(screen, bx, by, bw, bh, fill, true)

	edgeCol := colRecruitFrameHi
	if active {
		edgeCol = colRecruitGold
	}
	vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1.5, edgeCol, true)

	lw := textW(bitmapfont.Face, label)
	lx := r.Min.X + (r.Dx()-lw)/2
	ly := baselineY(bitmapfont.Face, r.Min.Y+(r.Dy()-bitmapfont.Face.Metrics().Height.Ceil())/2)
	lc := colRecruitMuted
	if active || hovered {
		lc = colRecruitGold
	}
	text.Draw(screen, label, bitmapfont.Face, lx, ly, lc)
}
