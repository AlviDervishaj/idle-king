package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

const (
	goldFloatLifetimeSec  = 1.05
	goldFloatRisePxPerSec = 32.0 // world space (negative Y = up on screen)
	maxGoldFloats         = 36
)

type goldFloat struct {
	x, y   float64
	t      float64
	amount float64
}

func (g *Game) spawnGoldFloat(wx, wy float64, amount float64) {
	if amount <= 0 {
		return
	}
	if len(g.goldFloats) >= maxGoldFloats {
		g.goldFloats = g.goldFloats[1:]
	}
	g.goldFloats = append(g.goldFloats, goldFloat{x: wx, y: wy, t: 0, amount: amount})
}

func (g *Game) updateGoldFloats(dt float64) {
	out := g.goldFloats[:0]
	for i := range g.goldFloats {
		f := &g.goldFloats[i]
		f.t += dt
		f.y -= goldFloatRisePxPerSec * dt
		if f.t < goldFloatLifetimeSec {
			out = append(out, *f)
		}
	}
	g.goldFloats = out
}

func goldFloatAlpha(t float64) float64 {
	p := t / goldFloatLifetimeSec
	if p < 0.62 {
		return 1
	}
	if p >= 1 {
		return 0
	}
	return 1 - (p-0.62)/(1-0.62)
}

func drawGoldFloats(screen *ebiten.Image, ox, oy, mapScale float64, floats []goldFloat) {
	for i := range floats {
		f := &floats[i]
		label := fmt.Sprintf("+%d", int(f.amount))
		if f.amount > 0 && f.amount < 10 {
			// If it has decimals, show them.
			if f.amount != float64(int(f.amount)) {
				label = fmt.Sprintf("+%.1f", f.amount)
			}
		}
		tw := textW(bitmapfont.Face, label)
		sx := ox + f.x*mapScale
		sy := oy + f.y*mapScale
		ix := int(sx) - tw/2
		iy := int(sy)
		a := goldFloatAlpha(f.t)
		if a <= 0.02 {
			continue
		}
		ai := uint8(255 * a)
		if ai == 0 {
			continue
		}
		shadow := color.RGBA{R: 0x08, G: 0x0a, B: 0x0f, A: ai * 3 / 4}
		gold := color.RGBA{R: 0xee, G: 0xd4, B: 0x6a, A: ai}
		text.Draw(screen, label, bitmapfont.Face, ix+1, iy+1, shadow)
		text.Draw(screen, label, bitmapfont.Face, ix, iy, gold)
	}
}
