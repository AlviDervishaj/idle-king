package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/AlviDervishaj/idle-king/internal/game"
	"github.com/AlviDervishaj/idle-king/world"
	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets final-background
var assetsFS embed.FS

func readAsset(path string) []byte {
	b, err := assetsFS.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func main() {
	// Note: asset naming is inconsistent (Attak1 vs. Attack1 vs. Attack)
	archerAttackSuffixes := []string{"Attak1", "Attack1", "Attack1", "Attack", "Attack1", "Attack1"}

	archerTiers := make([]game.ArcherTierAssets, len(game.ArcherTierNames))
	for i := range game.ArcherTierNames {
		tierNum := i + 1
		name := game.ArcherTierNames[i]
		suffix := archerAttackSuffixes[i]

		idlePath := fmt.Sprintf("assets/human-archer/%d_%s/[%s] Idle_MiniArcher.gif", tierNum, name, name)
		attackPath := fmt.Sprintf("assets/human-archer/%d_%s/[%s] %s_MiniArcher.gif", tierNum, name, name, suffix)

		archerTiers[i] = game.ArcherTierAssets{
			IdleGIF:   readAsset(idlePath),
			AttackGIF: readAsset(attackPath),
		}
	}

	enemyStrips := make([]game.EnemyStripPair, 4)
	for i := range 4 {
		num := i + 1
		enemyStrips[i] = game.EnemyStripPair{
			WalkPNG:  readAsset(fmt.Sprintf("assets/free-field-enemies-pixel-art-for-tower-defense/%d/S_Walk.png", num)),
			DeathPNG: readAsset(fmt.Sprintf("assets/free-field-enemies-pixel-art-for-tower-defense/%d/S_Death.png", num)),
		}
	}

	g, err := game.NewGame(
		readAsset("final-background/idle-king-map.png"),
		readAsset("assets/Arrows_pack.png"),
		enemyStrips,
		archerTiers,
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := g.Load(); err != nil {
		log.Printf("idle-king: no save file or load error: %v", err)
	}

	ebiten.SetWindowSize(world.ViewportWidth, world.ViewportHeight)
	ebiten.SetWindowTitle("Idle King")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetTPS(60)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}

	if err := g.Save(); err != nil {
		log.Printf("idle-king: save on exit failed: %v", err)
	}
}
