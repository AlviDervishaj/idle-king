package game

import (
	"fmt"
	"image/color"

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
)

func textW(face font.Face, s string) int {
	return font.MeasureString(face, s).Ceil()
}

func baselineY(face font.Face, top int) int {
	a := face.Metrics().Ascent.Ceil()
	return top + a
}

const shopDividerOff = 260

func (g *Game) drawRecruitPanel(screen *ebiten.Image) {
	cx := float32(g.cardRect.Min.X)
	cy := float32(g.cardRect.Min.Y)
	cw := float32(g.cardRect.Dx())
	ch := float32(g.cardRect.Dy())

	vector.FillRect(screen, cx+5, cy+7, cw, ch, colRecruitShadow, true)

	vector.FillRect(screen, cx, cy, cw, ch, colRecruitCard, true)
	vector.StrokeRect(screen, cx+1, cy+1, cw-2, ch-2, 2, colRecruitCardEdge, true)

	divX := float32(g.cardRect.Min.X + shopDividerOff)
	vector.StrokeLine(screen, divX, cy+36, divX, cy+ch-14, 1.5, colRecruitFrameHi, true)

	innerL := g.cardRect.Min.X + 16
	innerR := g.cardRect.Max.X - 16

	titleY := baselineY(bitmapfont.Face, g.cardRect.Min.Y+14)
	text.Draw(screen, "SHOP", bitmapfont.Face, innerL, titleY, colRecruitGold)

	goldStr := fmt.Sprintf("%d gold", int(g.gold))
	text.Draw(screen, goldStr, bitmapfont.Face, innerR-textW(bitmapfont.Face, goldStr), titleY, colRecruitGold)

	divY := float32(g.cardRect.Min.Y + 38)
	vector.StrokeLine(screen, float32(innerL), divY, float32(innerR), divY, 1, colRecruitFrameHi, true)

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

	leftColCenter := (g.cardRect.Min.X + 16 + g.cardRect.Min.X + shopDividerOff - 8) / 2

	nameY := baselineY(bitmapfont.Face, g.portraitRect.Max.Y+8)
	text.Draw(screen, "Archers", bitmapfont.Face, leftColCenter-textW(bitmapfont.Face, "Archers")/2, nameY, colRecruitText)

	subY := baselineY(bitmapfont.Face, g.portraitRect.Max.Y+26)
	sub := "Ranged unit"
	tIdx := g.metaUpgradeRank[metaArcherTier]
	if tIdx >= 0 && tIdx < len(g.archerTierRuntimes) {
		sub = g.archerTierRuntimes[tIdx].name + " training level"
	}
	text.Draw(screen, sub, bitmapfont.Face, leftColCenter-textW(bitmapfont.Face, sub)/2, subY, colRecruitMuted)

	g.drawTierSteppers(screen)
	g.drawRecruitBuyButton(screen)
	g.drawMetaUpgradeButton(screen, metaArcherTier)

	squad := fmt.Sprintf("In squad: %d", len(g.worldArchers))
	sqY := baselineY(bitmapfont.Face, g.upgradeBtnRects[metaArcherTier].Max.Y+10)
	text.Draw(screen, squad, bitmapfont.Face, leftColCenter-textW(bitmapfont.Face, squad)/2, sqY, colRecruitMuted)

	contractsY := baselineY(bitmapfont.Face, g.cardRect.Min.Y+48)
	rightL := g.cardRect.Min.X + shopDividerOff + 16
	text.Draw(screen, "CONTRACTS", bitmapfont.Face, rightL, contractsY, colRecruitGold)
	subCY := baselineY(bitmapfont.Face, g.cardRect.Min.Y+66)
	text.Draw(screen, "Roster & kill-gold bonuses", bitmapfont.Face, rightL, subCY, colRecruitMuted)

	sumY := baselineY(bitmapfont.Face, g.upgradeBtnRects[metaLaserDamage].Min.Y-34)
	text.Draw(screen, "SUMMONER", bitmapfont.Face, rightL, sumY, colRecruitGold)
	sumSubY := baselineY(bitmapfont.Face, g.upgradeBtnRects[metaLaserDamage].Min.Y-16)
	text.Draw(screen, "Tower beam upgrades", bitmapfont.Face, rightL, sumSubY, colRecruitMuted)

	g.drawEnemyUnlockButton(screen)
	for i := range g.upgradeBtnRects {
		if i == metaArcherTier {
			continue
		}
		g.drawMetaUpgradeButton(screen, i)
	}
}

func (g *Game) drawEnemyUnlockButton(screen *ebiten.Image) {
	r := g.enemyUnlockBtnRect
	if r.Empty() {
		return
	}
	bx := float32(r.Min.X)
	by := float32(r.Min.Y)
	bw := float32(r.Dx())
	bh := float32(r.Dy())

	cost := g.enemyUnlockNextCost()
	maxed := cost <= 0
	canAfford := !maxed && g.gold >= float64(cost)
	hovered := g.unlockEnemyHovered
	pressed := hovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && canAfford

	var fill color.Color = colRecruitBtn
	if maxed {
		fill = colRecruitBtnDis
	} else if !canAfford {
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
	base := baselineY(bitmapfont.Face, r.Min.Y+4+8)

	total := len(g.enemyVariants)
	if total < 1 {
		total = 1
	}
	title := "Enemy roster"
	left := fmt.Sprintf("%s  %d/%d", title, g.enemyTypesUnlocked, total)
	text.Draw(screen, left, bitmapfont.Face, innerL, base, colRecruitText)

	subY := baselineY(bitmapfont.Face, r.Min.Y+4+26)
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

func (g *Game) drawMetaUpgradeButton(screen *ebiten.Image, id int) {
	r := g.upgradeBtnRects[id]
	if r.Empty() {
		return
	}
	bx := float32(r.Min.X)
	by := float32(r.Min.Y)
	bw := float32(r.Dx())
	bh := float32(r.Dy())

	cost := g.metaUpgradeNextCost(id)
	canAfford := cost > 0 && g.gold >= float64(cost)
	hovered := g.upgradeHovered[id]
	pressed := hovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && canAfford

	var fill color.Color = colRecruitBtn
	if !canAfford {
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
	base := baselineY(bitmapfont.Face, r.Min.Y+4+8)

	title := metaUpgradeTitles[id]
	if id >= len(metaUpgradeTitles) {
		title = "?"
	}
	desc := metaUpgradeDesc[id]
	if id >= len(metaUpgradeDesc) {
		desc = ""
	}

	rank := g.metaUpgradeRank[id]
	left := fmt.Sprintf("%s  Rank %d", title, rank)
	text.Draw(screen, left, bitmapfont.Face, innerL, base, colRecruitText)

	subY := baselineY(bitmapfont.Face, r.Min.Y+4+26)
	text.Draw(screen, desc, bitmapfont.Face, innerL, subY, colRecruitMuted)

	var right string
	if canAfford {
		right = fmt.Sprintf("%d gold", cost)
	} else {
		right = fmt.Sprintf("Need %d", cost)
	}
	rc := colRecruitGold
	if !canAfford {
		rc = colRecruitMuted
	}
	text.Draw(screen, right, bitmapfont.Face, innerR-textW(bitmapfont.Face, right), base, rc)
}

func (g *Game) drawTierSteppers(screen *ebiten.Image) {
	// Steppers removed: archer tier is now a global meta-upgrade.
}

func (g *Game) drawRecruitBuyButton(screen *ebiten.Image) {
	bx := float32(g.buyBtnRect.Min.X)
	by := float32(g.buyBtnRect.Min.Y)
	bw := float32(g.buyBtnRect.Dx())
	bh := float32(g.buyBtnRect.Dy())

	cost := g.selectedArcherHireCost()
	canAfford := cost > 0 && g.gold >= float64(cost)
	pressed := g.buyHovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && canAfford && cost > 0

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

	innerL := g.buyBtnRect.Min.X + 12
	innerR := g.buyBtnRect.Max.X - 12
	btnTop := g.buyBtnRect.Min.Y + (g.buyBtnRect.Dy()-16)/2
	base := baselineY(bitmapfont.Face, btnTop)

	left := "Hire archer"
	var right string
	if cost <= 0 {
		right = "—"
	} else if canAfford {
		right = fmt.Sprintf("%d gold", cost)
	} else {
		right = fmt.Sprintf("Need %d gold", cost)
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

func (g *Game) drawShopToggle(screen *ebiten.Image) {
	bx := float32(g.shopToggleRect.Min.X)
	by := float32(g.shopToggleRect.Min.Y)
	bw := float32(g.shopToggleRect.Dx())
	bh := float32(g.shopToggleRect.Dy())

	pressed := g.shopToggleHover && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	const a = uint8(0.8 * 255)
	fill := color.RGBA{R: 0x1a, G: 0x22, B: 0x32, A: a}
	if pressed {
		fill = color.RGBA{R: 0x0c, G: 0x0f, B: 0x16, A: a}
	} else if g.shopToggleHover {
		fill = color.RGBA{R: 0x24, G: 0x2e, B: 0x42, A: a}
	}
	vector.FillRect(screen, bx, by, bw, bh, fill, true)
	vector.StrokeRect(screen, bx+1, by+1, bw-2, bh-2, 1.5, colRecruitGold, true)

	label := "Shop"
	base := baselineY(bitmapfont.Face, g.shopToggleRect.Min.Y+(g.shopToggleRect.Dy()-16)/2)
	cx := g.shopToggleRect.Min.X + g.shopToggleRect.Dx()/2
	text.Draw(screen, label, bitmapfont.Face, cx-textW(bitmapfont.Face, label)/2, base, colRecruitGold)
}
