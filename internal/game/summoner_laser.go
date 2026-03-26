package game

import (
	"image/color"
	"math"

	"github.com/AlviDervishaj/idle-king/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Summoner footprint matches internal/spawn summonerTileBounds (cols 4–8, rows 0–4); laser fires from center tile.
const (
	summonerLaserCol = 6
	summonerLaserRow = 2

	laserDPSPerSec      = 75.0
	laserImpactRadiusPx = 18.0 // damage only within this radius of the mouse aim point (world px)
	laserPulseSpeed     = 3.2
)

func summonerLaserOrigin() (float64, float64) {
	x := float64(summonerLaserCol*world.TileSize + world.TileSize/2)
	y := float64((summonerLaserRow+1)*world.TileSize - 2)
	return x, y
}

func defaultSummonerLaserAim() (float64, float64) {
	_, sy := summonerLaserOrigin()
	return float64(world.MapWidthPx) * 0.72, sy
}

func screenToMapWorld(screenX, screenY, mw, mh, sw, sh int) (float64, float64) {
	scale := math.Min(float64(sw)/float64(mw), float64(sh)/float64(mh))
	dx := float64(mw) * scale
	dy := float64(mh) * scale
	ox := (float64(sw) - dx) * 0.5
	oy := (float64(sh) - dy) * 0.5
	wx := (float64(screenX) - ox) / scale
	wy := (float64(screenY) - oy) / scale
	return wx, wy
}

func (g *Game) updateLaserAim() {
	if g.mapImg == nil {
		return
	}
	mw, mh := g.mapImg.Bounds().Dx(), g.mapImg.Bounds().Dy()
	mx, my := ebiten.CursorPosition()
	wx, wy := screenToMapWorld(mx, my, mw, mh, world.ViewportWidth, world.ViewportHeight)
	g.laserWorldX = clamp(wx, 0, float64(world.MapWidthPx))
	g.laserWorldY = clamp(wy, 0, float64(world.MapHeightPx))
}

func (g *Game) applySummonerLaser(dt float64) {
	if g.lavaMask == nil {
		return
	}
	// Damage only at the mouse aim point, not along the visible beam.
	ix, iy := g.laserWorldX, g.laserWorldY
	dps := laserDPSPerSec + float64(g.metaUpgradeRank[metaLaserDamage])*15.0
	radius := laserImpactRadiusPx + float64(g.metaUpgradeRank[metaLaserRadius])*4.0
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.dying || e.hp <= 0 {
			continue
		}
		// Damage only within this radius of the mouse aim point (world px)
		if math.Hypot(e.x-ix, e.y-(iy+24)) <= radius {
			e.hp -= dps * dt
			if e.hp <= 0 {
				g.beginEnemyDeath(e, deathCausePlayerLaser)
			}
		}
	}
}

func (g *Game) drawSummonerLaser(screen *ebiten.Image, ox, oy, mapScale float64) {
	ax, ay := summonerLaserOrigin()
	bx, by := g.laserWorldX, g.laserWorldY
	x0 := float32(ox + ax*mapScale)
	y0 := float32(oy + ay*mapScale)
	x1 := float32(ox + bx*mapScale)
	y1 := float32(oy + by*mapScale)
	pulse := float32(1.0 + 0.28*math.Sin(g.laserPhase))
	outer := float32(7.0) * pulse
	inner := float32(3.0) * pulse
	vector.StrokeLine(screen, x0, y0, x1, y1, outer, color.RGBA{R: 0x20, G: 0x55, B: 0xee, A: 0x44}, false)
	vector.StrokeLine(screen, x0, y0, x1, y1, inner+2, color.RGBA{R: 0x55, G: 0xaa, B: 0xff, A: 0x88}, false)
	vector.StrokeLine(screen, x0, y0, x1, y1, inner, color.RGBA{R: 0xcc, G: 0xf5, B: 0xff, A: 0xf0}, false)

	radius := laserImpactRadiusPx + float64(g.metaUpgradeRank[metaLaserRadius])*4.0
	r := float32(radius * mapScale)
	vector.StrokeCircle(screen, x1, y1, r, 2.0, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x88}, false)
}
