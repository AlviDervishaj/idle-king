package game

// speedMultipliers maps speedIndex → time dilation factor for gameplay simulation.
// UI animations, auto-save timers, and toast timers always use raw frameDT.
var speedMultipliers = [3]float64{1.0, 2.0, 4.0}

// speedLabels are the display strings for the speed buttons in the HUD.
var speedLabels = [3]string{"1x", "2x", "4x"}

// gameplayDT returns the frame delta scaled by the current speed multiplier.
// All combat, spawn, arrow movement, lava damage, and gold-float updates use this.
func (g *Game) gameplayDT() float64 {
	return g.frameDT() * speedMultipliers[g.speedIndex]
}
