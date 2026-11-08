package game

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// formatGold formats a gold amount compactly: 1234 → "1,234", 1234567 → "1.23M", etc.
func formatGold(n float64) string {
	switch {
	case n >= 1e12:
		return fmt.Sprintf("%.2fT", n/1e12)
	case n >= 1e9:
		return fmt.Sprintf("%.2fB", n/1e9)
	case n >= 1e6:
		return fmt.Sprintf("%.2fM", n/1e6)
	case n >= 1e3:
		// Thousands with comma separator.
		t := int(n) / 1000
		r := int(n) % 1000
		return fmt.Sprintf("%d,%03d", t, r)
	default:
		return fmt.Sprintf("%d", int(n))
	}
}

// currentDPS computes the total current damage per second across all archers + laser.
// arrowDamageForTier already incorporates achievementMuls.damageMul, so only the
// laser portion needs the multiplier applied here.
func (g *Game) currentDPS() float64 {
	interval := archerFireIntervalSec * g.extraFireIntervalMul()
	if interval < 0.01 {
		interval = 0.01
	}
	var dps float64
	for i := range g.worldArchers {
		dps += g.arrowDamageForTier(g.worldArchers[i].tierIdx) / interval
	}
	laserDmg := (laserDPSPerSec + float64(g.metaUpgradeRank[metaLaserDamage])*15.0) *
		(1.0 + g.achievementMuls.damageMul)
	dps += laserDmg
	return dps
}

// drawHUD renders the always-visible overlay in the top-right corner of the screen.
// It shows gold, income rate, DPS, wave counter, kill count, and speed controls.
func (g *Game) drawHUD(screen *ebiten.Image) {
	const (
		panelW = 230
		panelH = 108
		panelX = 1440 - panelW - 8
		panelY = 8
		padX   = 10
		lineH  = 16
	)

	bg := color.RGBA{R: 0x10, G: 0x15, B: 0x1e, A: 0xd8}
	edge := color.RGBA{R: 0x6b, G: 0x58, B: 0x38, A: 0xff}
	vector.FillRect(screen, panelX, panelY, panelW, panelH, bg, false)
	vector.StrokeRect(screen, panelX+1, panelY+1, panelW-2, panelH-2, 1.5, edge, false)

	goldColor := color.RGBA{R: 0xee, G: 0xd4, B: 0x6a, A: 0xff}
	textColor := color.RGBA{R: 0xee, G: 0xe8, B: 0xdc, A: 0xff}
	mutedColor := color.RGBA{R: 0x8b, G: 0x92, B: 0xa8, A: 0xff}

	x := panelX + padX

	// Row 1: Gold
	y1 := panelY + 14
	goldStr := formatGold(g.gold) + " gold"
	text.Draw(screen, goldStr, bitmapfont.Face, x, y1, goldColor)

	// Row 2: Income + DPS
	y2 := y1 + lineH
	gpsStr := "+" + formatGold(g.gpsSample) + "/sec"
	dpsStr := formatGold(g.currentDPS()) + " DPS"
	text.Draw(screen, gpsStr, bitmapfont.Face, x, y2, textColor)
	text.Draw(screen, dpsStr, bitmapfont.Face, panelX+panelW-padX-textW(bitmapfont.Face, dpsStr), y2, textColor)

	// Row 3: Kills + Wave
	y3 := y2 + lineH
	killStr := formatGold(float64(g.stats.totalKills)) + " kills"
	var waveStr string
	nextBoss := g.boss.nextBossWave - g.waveCounter
	if nextBoss <= 3 && nextBoss > 0 {
		waveStr = fmt.Sprintf("Wave %d — BOSS in %d!", g.waveCounter, nextBoss)
	} else if g.boss.activeBossID != 0 {
		waveStr = fmt.Sprintf("Wave %d — BOSS!", g.waveCounter)
	} else {
		waveStr = fmt.Sprintf("Wave %d", g.waveCounter)
	}
	text.Draw(screen, killStr, bitmapfont.Face, x, y3, mutedColor)
	text.Draw(screen, waveStr, bitmapfont.Face, panelX+panelW-padX-textW(bitmapfont.Face, waveStr), y3, mutedColor)

	// Speed buttons row
	y4 := y3 + lineH + 4
	btnW := 44
	btnH := 22
	for i, label := range speedLabels {
		bx := panelX + padX + i*(btnW+4)
		by := y4
		active := g.speedIndex == i
		var fill color.Color
		if active {
			fill = color.RGBA{R: 0x2a, G: 0x5c, B: 0x42, A: 0xff}
		} else if g.speedBtnHover[i] {
			fill = color.RGBA{R: 0x24, G: 0x2e, B: 0x42, A: 0xff}
		} else {
			fill = color.RGBA{R: 0x18, G: 0x20, B: 0x2e, A: 0xff}
		}
		edgeCol := edge
		if active {
			edgeCol = color.RGBA{R: 0x5a, G: 0x8a, B: 0x6a, A: 0xff}
		}
		vector.FillRect(screen, float32(bx), float32(by), float32(btnW), float32(btnH), fill, false)
		vector.StrokeRect(screen, float32(bx)+1, float32(by)+1, float32(btnW)-2, float32(btnH)-2, 1.5, edgeCol, false)
		lc := mutedColor
		if active {
			lc = goldColor
		}
		lw := textW(bitmapfont.Face, label)
		text.Draw(screen, label, bitmapfont.Face, bx+(btnW-lw)/2, by+14, lc)
		// Store rect for Update() hover/click detection.
		g.speedBtnRects[i] = image.Rect(bx, by, bx+btnW, by+btnH)
	}

	// Offline message banner.
	if g.offlineMessageTimer > 0 && g.offlineMessage != "" {
		alpha := float32(1.0)
		if g.offlineMessageTimer < 1.5 {
			alpha = float32(g.offlineMessageTimer / 1.5)
		}
		bannerY := 6
		bannerH := 28
		bx, bw := 200, 1040
		bannerFill := color.RGBA{R: 0x0c, G: 0x14, B: 0x24, A: uint8(200 * alpha)}
		vector.FillRect(screen, float32(bx), float32(bannerY), float32(bw), float32(bannerH), bannerFill, false)
		tw := textW(bitmapfont.Face, g.offlineMessage)
		tx := bx + (bw-tw)/2
		ty := bannerY + 18
		tc := color.RGBA{R: 0xee, G: 0xd4, B: 0x6a, A: uint8(255 * alpha)}
		text.Draw(screen, g.offlineMessage, bitmapfont.Face, tx, ty, tc)
	}

	// Achievement toast (bottom-center of screen).
	if len(g.achievementToasts) > 0 {
		toast := &g.achievementToasts[0]
		alpha := float32(1.0)
		if toast.timer < 1.0 {
			alpha = float32(toast.timer)
		}
		const tw, th = 340, 52
		tx := (1440 - tw) / 2
		ty := 720 - th - 12
		toastFill := color.RGBA{R: 0x1a, G: 0x22, B: 0x32, A: uint8(230 * alpha)}
		toastEdge := color.RGBA{R: 0xee, G: 0xd4, B: 0x6a, A: uint8(200 * alpha)}
		vector.FillRect(screen, float32(tx), float32(ty), tw, th, toastFill, false)
		vector.StrokeRect(screen, float32(tx)+1, float32(ty)+1, tw-2, th-2, 1.5, toastEdge, false)
		titleCol := color.RGBA{R: 0xee, G: 0xd4, B: 0x6a, A: uint8(255 * alpha)}
		descCol := color.RGBA{R: 0xee, G: 0xe8, B: 0xdc, A: uint8(200 * alpha)}
		titleStr := "Achievement: " + toast.title
		text.Draw(screen, titleStr, bitmapfont.Face, tx+10, ty+16, titleCol)
		sub := toast.desc
		if toast.rewardDesc != "" {
			sub += "  (" + toast.rewardDesc + ")"
		}
		text.Draw(screen, sub, bitmapfont.Face, tx+10, ty+32, descCol)
	}
}

