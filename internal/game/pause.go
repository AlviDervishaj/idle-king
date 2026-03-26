package game

import (
	"image"
	"image/color"

	bitmapfont "github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ErrQuit is returned from Update() when the player chooses "Quit to Desktop".
// main.go checks for this sentinel so it can call Save() before the process exits.
var ErrQuit = errQuit{}

type errQuit struct{}

func (errQuit) Error() string { return "quit" }

// pauseMenuState holds the interactive rects and hover flags for the pause overlay.
type pauseMenuState struct {
	resumeRect  image.Rectangle
	resumeHover bool

	speedRects [3]image.Rectangle
	speedHover [3]bool

	statsRect  image.Rectangle
	statsHover bool

	quitRect  image.Rectangle
	quitHover bool
}

// Pause panel geometry — centred in the 1440×720 logical viewport.
const (
	pmW = 400
	pmH = 400
	pmX = (1440 - pmW) / 2 // 520
	pmY = (720 - pmH) / 2  // 160
)

// drawPauseMenu renders the full-screen dim + the centred pause panel.
// Must be the very last call in Draw() so it covers all other UI.
func (g *Game) drawPauseMenu(screen *ebiten.Image) {
	face := bitmapfont.Face

	// ── full-screen dim ───────────────────────────────────────────────────────
	vector.FillRect(screen, 0, 0, 1440, 720,
		color.RGBA{0x00, 0x00, 0x00, 0xb2}, false)

	// ── drop shadow ───────────────────────────────────────────────────────────
	vector.FillRect(screen,
		float32(pmX+5), float32(pmY+7), float32(pmW), float32(pmH),
		color.RGBA{0x00, 0x00, 0x00, 0xa8}, false)

	// ── panel ─────────────────────────────────────────────────────────────────
	vector.FillRect(screen,
		float32(pmX), float32(pmY), float32(pmW), float32(pmH),
		color.RGBA{0x12, 0x18, 0x26, 0xf8}, false)
	vector.StrokeRect(screen,
		float32(pmX+1), float32(pmY+1), float32(pmW-2), float32(pmH-2),
		2, colRecruitCardEdge, false)

	// ── title ─────────────────────────────────────────────────────────────────
	const titleStr = "PAUSED"
	tw := textW(face, titleStr)
	text.Draw(screen, titleStr, face,
		pmX+(pmW-tw)/2, baselineY(face, pmY+12), colRecruitGold)

	vector.StrokeLine(screen,
		float32(pmX+16), float32(pmY+38),
		float32(pmX+pmW-16), float32(pmY+38),
		1, colRecruitCardEdge, false)

	// ── layout constants ──────────────────────────────────────────────────────
	const (
		btnW = 300
		btnH = 36
		btnX = pmX + (pmW-btnW)/2
	)
	curY := pmY + 50

	// 1) Resume ────────────────────────────────────────────────────────────────
	g.pm.resumeRect = image.Rect(btnX, curY, btnX+btnW, curY+btnH)
	pmDrawBtn(screen, g.pm.resumeRect, g.pm.resumeHover, "Resume Game",
		colRecruitBtn, colRecruitBtnHi)
	curY += btnH + 10

	// 2) Speed selector ────────────────────────────────────────────────────────
	speedLabelStr := "Speed:"
	text.Draw(screen, speedLabelStr, face,
		btnX, baselineY(face, curY+6), colRecruitMuted)

	const sBtnW, sBtnH = 54, 28
	sBtnX0 := btnX + textW(face, speedLabelStr) + 14
	for i := range 3 {
		sr := image.Rect(
			sBtnX0+i*(sBtnW+6), curY,
			sBtnX0+i*(sBtnW+6)+sBtnW, curY+sBtnH,
		)
		g.pm.speedRects[i] = sr
		active := g.speedIndex == i
		var bg, edge color.RGBA
		switch {
		case active:
			bg = color.RGBA{0xcc, 0x99, 0x22, 0xff}
			edge = color.RGBA{0xff, 0xdd, 0x44, 0xff}
		case g.pm.speedHover[i]:
			bg = color.RGBA{0x33, 0x44, 0x60, 0xff}
			edge = color.RGBA{0x88, 0x88, 0xaa, 0xff}
		default:
			bg = color.RGBA{0x22, 0x2e, 0x44, 0xff}
			edge = color.RGBA{0x44, 0x44, 0x66, 0x88}
		}
		vector.FillRect(screen,
			float32(sr.Min.X), float32(sr.Min.Y), float32(sr.Dx()), float32(sr.Dy()),
			bg, false)
		vector.StrokeRect(screen,
			float32(sr.Min.X), float32(sr.Min.Y), float32(sr.Dx()), float32(sr.Dy()),
			1, edge, false)
		lw := textW(face, speedLabels[i])
		lx := sr.Min.X + (sr.Dx()-lw)/2
		lc := colRecruitText
		if active {
			lc = color.RGBA{0x11, 0x11, 0x00, 0xff}
		}
		text.Draw(screen, speedLabels[i], face, lx, baselineY(face, sr.Min.Y+5), lc)
	}
	curY += sBtnH + 14

	// 3) Statistics toggle ─────────────────────────────────────────────────────
	statsLbl := "Statistics"
	if g.shopOpen && g.statsOpen {
		statsLbl = "Statistics  (currently open)"
	}
	g.pm.statsRect = image.Rect(btnX, curY, btnX+btnW, curY+btnH)
	pmDrawBtn(screen, g.pm.statsRect, g.pm.statsHover, statsLbl,
		color.RGBA{0x1e, 0x3a, 0x5a, 0xff}, color.RGBA{0x28, 0x4e, 0x74, 0xff})
	curY += btnH + 14

	// ── divider ────────────────────────────────────────────────────────────────
	vector.StrokeLine(screen,
		float32(pmX+16), float32(curY),
		float32(pmX+pmW-16), float32(curY),
		1, color.RGBA{0x55, 0x44, 0x33, 0x88}, false)
	curY += 14

	// 4) Quit to Desktop ───────────────────────────────────────────────────────
	g.pm.quitRect = image.Rect(btnX, curY, btnX+btnW, curY+btnH)
	pmDrawBtn(screen, g.pm.quitRect, g.pm.quitHover, "Quit to Desktop",
		color.RGBA{0x5a, 0x1e, 0x1e, 0xff}, color.RGBA{0x7a, 0x28, 0x28, 0xff})
	curY += btnH + 14

	// ── auto-save / offline description ───────────────────────────────────────
	// Explains the persistent save + offline-progress feature.
	noteLines := []string{
		"Progress is auto-saved every 30s and on quit.",
		"Gold keeps accumulating while you're away —",
		"come back anytime, nothing is ever lost!",
	}
	noteCol := color.RGBA{0x77, 0x8a, 0x9a, 0xff}
	for _, line := range noteLines {
		lw := textW(face, line)
		lx := pmX + (pmW-lw)/2
		text.Draw(screen, line, face, lx, baselineY(face, curY), noteCol)
		curY += 14
	}
}

// pmDrawBtn renders a single button inside the pause panel using the project's
// standard style: shadow + fill + border + centred label.
func pmDrawBtn(screen *ebiten.Image, r image.Rectangle, hovered bool,
	label string, bgNormal, bgHover color.RGBA) {

	bg := bgNormal
	if hovered {
		bg = bgHover
	}
	// shadow
	vector.FillRect(screen,
		float32(r.Min.X+2), float32(r.Min.Y+3), float32(r.Dx()), float32(r.Dy()),
		color.RGBA{0, 0, 0, 0x60}, false)
	// body
	vector.FillRect(screen,
		float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()),
		bg, false)
	// border
	edgeA := uint8(0x88)
	if hovered {
		edgeA = 0xff
	}
	vector.StrokeRect(screen,
		float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()),
		1, color.RGBA{0xcc, 0xaa, 0x44, edgeA}, false)
	// centred label
	lw := textW(bitmapfont.Face, label)
	lx := r.Min.X + (r.Dx()-lw)/2
	text.Draw(screen, label, bitmapfont.Face, lx,
		baselineY(bitmapfont.Face, r.Min.Y+10), colRecruitText)
}
