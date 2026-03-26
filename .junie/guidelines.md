# Idle King Development Guidelines

This document provides project-specific information for developers working on the Idle King game.

## 1. Build and Configuration

### Environment Requirements
- **Go Version**: 1.25.6 or higher.
- **Dependencies**: The project relies on [Ebitengine](https://ebitengine.org/) (v2).

### Running the Project
To run the game directly from source:
```bash
go run main.go
```

### Building the Project
To create a binary:
```bash
go build -o idle-king main.go
```

### Asset Management
- Assets (images, GIFs) are embedded into the binary using the `go:embed` directive in `main.go`.
- If adding new assets:
    1. Place the asset in the `assets/` directory.
    2. Add a `//go:embed` directive in `main.go`.
    3. Update the `game.NewGame` call if the asset needs to be passed to the game engine.

### Logical Resolution
The game uses a fixed logical resolution:
- **Width**: 1440
- **Height**: 720
- **Aspect Ratio**: 2:1

## 2. Testing

### Running Tests
All tests can be executed using the standard Go test tool:
```bash
go test ./...
```

### Adding New Tests
- Create a file with the `_test.go` suffix in the same package as the code being tested.
- Use the standard `testing` package.
- For tests requiring a context, use `t.Context()` (available since Go 1.24).

### Test Example
Below is a simple test demonstration for the `world` package, which handles coordinate conversions and bounds checking:

```go
package world

import "testing"

func TestInBounds(t *testing.T) {
	tests := []struct {
		row, col int
		want     bool
	}{
		{0, 0, true},
		{HeightTiles - 1, WidthTiles - 1, true},
		{-1, 0, false},
		{HeightTiles, 0, false},
	}

	for _, tt := range tests {
		if got := InBounds(tt.row, tt.col); got != tt.want {
			t.Errorf("InBounds(%d, %d) = %v, want %v", tt.row, tt.col, got, tt.want)
		}
	}
}
```

## 3. Additional Development Information

### Code Style and Idioms
- **Modern Go**: Strictly follow Go 1.25+ idioms.
    - Use `for i := range n` for simple iteration.
    - Use `any` instead of `interface{}`.
    - Utilize `slices`, `maps`, and `cmp` packages from the standard library.
- **Naming**: Use clear, descriptive names. Follow standard Go naming conventions (PascalCase for exported, camelCase for unexported).
- **No Magic Numbers**: All dimensions, timings, and gameplay constants must be defined as named constants (e.g., in `world/world.go` or within the relevant package).

### Architecture
- **`internal/game`**: Contains the core `Game` struct and the `Update`/`Draw` loop logic.
- **`internal/spawn`**: Handles logic for finding valid spawn positions for archers and enemies.
- **`world`**: Provides map dimensions and pure functions for coordinate transformations.
- **`objects`**: Contains logic and data for specific map elements like `lava`.

### Frame Delta Timing
Do not rely on `ebiten.ActualTPS()` for movement calculations, as it can be unstable. Instead, use `time.Since()` to calculate the actual elapsed time between frames:

```go
func (g *Game) frameDT() float64 {
	now := time.Now()
	dt := now.Sub(g.lastFrameTime).Seconds()
	g.lastFrameTime = now
	return dt
}
```

### Mutex Usage
The `Game` struct uses `sync.RWMutex` (`combatMu`) to protect gameplay state (enemies, arrows, timers). Although Ebitengine runs `Update` and `Draw` sequentially on a single thread by default, using a mutex ensures clarity and safety for future concurrent operations.
