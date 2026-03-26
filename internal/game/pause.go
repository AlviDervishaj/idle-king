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

	quitRect  image.Rectangle
	quitHover bool
}

// Pause panel geometry — centred in the 1440×720 logical viewport.
const (
	pmW = 360
	pmH = 260
	pmX = (1440 - pmW) / 2 // 540
	pmY = (720 - pmH) / 2  // 230
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
		pmX+(pmW-tw)/2, baselineY(face, pmY+14), colRecruitGold)

	vector.StrokeLine(screen,
		float32(pmX+16), float32(pmY+40),
		float32(pmX+pmW-16), float32(pmY+40),
		1, colRecruitCardEdge, false)

	// ── buttons ───────────────────────────────────────────────────────────────
	const (
		btnW = 280
		btnH = 38
		btnX = pmX + (pmW-btnW)/2
	)
	curY := pmY + 54

	// Resume ──────────────────────────────────────────────────────────────────
	g.pm.resumeRect = image.Rect(btnX, curY, btnX+btnW, curY+btnH)
	pmDrawBtn(screen, g.pm.resumeRect, g.pm.resumeHover, "Resume Game",
		colRecruitBtn, colRecruitBtnHi)
	curY += btnH + 16

	// Divider ─────────────────────────────────────────────────────────────────
	vector.StrokeLine(screen,
		float32(pmX+16), float32(curY),
		float32(pmX+pmW-16), float32(curY),
		1, color.RGBA{0x55, 0x44, 0x33, 0x88}, false)
	curY += 16

	// Quit to Desktop ─────────────────────────────────────────────────────────
	g.pm.quitRect = image.Rect(btnX, curY, btnX+btnW, curY+btnH)
	pmDrawBtn(screen, g.pm.quitRect, g.pm.quitHover, "Quit to Desktop",
		color.RGBA{0x5a, 0x1e, 0x1e, 0xff}, color.RGBA{0x7a, 0x28, 0x28, 0xff})
	curY += btnH + 16

	// ── save / offline note ───────────────────────────────────────────────────
	noteLines := []string{
		"Progress saves every 30s and on quit.",
		"Gold accumulates offline — nothing is lost!",
	}
	noteCol := color.RGBA{0x77, 0x8a, 0x9a, 0xff}
	for _, line := range noteLines {
		lw := textW(face, line)
		lx := pmX + (pmW-lw)/2
		text.Draw(screen, line, face, lx, baselineY(face, curY), noteCol)
		curY += 14
	}
}

// pmDrawBtn renders a single button inside the pause panel:
// drop-shadow + fill + border + centred label.
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
		baselineY(bitmapfont.Face, r.Min.Y+11), colRecruitText)
}
