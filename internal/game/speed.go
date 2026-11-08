package game

// speedMultipliers maps speedIndex → time dilation factor for gameplay simulation.
// UI animations, auto-save timers, and toast timers always use raw frameDT.
var speedMultipliers = [3]float64{1.0, 2.0, 4.0}

// speedLabels are the display strings for the speed buttons in the HUD.
var speedLabels = [3]string{"1x", "2x", "4x"}

// NOTE: gameplayDT is intentionally not defined here. Update() uses the pattern:
//   rawDt := g.frameDT()
//   dt     := rawDt * speedMultipliers[g.speedIndex]
// Calling frameDT() inside a helper would mutate lastFrameTime twice per frame.
