# CLAUDE.md — Idle King Development Guide

## Project Overview

2D idle/tower defense game. Archers auto-fire arrows at enemies crossing a lava moat; a player-aimed summoner laser adds direct damage. Kill loot funds hiring archers and meta-upgrades.

- **Language**: Go 1.25+ (currently 1.25.6)
- **Engine**: Ebitengine v2 (`github.com/hajimehoshi/ebiten/v2`)
- **Assets**: All embedded via `go:embed` in `main.go`
- **Resolution**: 1440×720 logical pixels (2:1 aspect), fixed via `Layout()`

## Quick Start

```bash
go run main.go            # run from source
go build -o idle-king .   # build binary
go test ./...             # run all tests
go vet ./...              # static analysis
```

## Architecture

```
main.go                  Entry point, go:embed, asset loading, ebiten.RunGame
internal/game/           Game struct, Update/Draw loop, combat, economy, UI, animations
internal/spawn/          Archer + enemy spawn position logic
world/                   Map constants, coordinate transforms (pure math, zero deps)
objects/atlas/           PNG tileset/strip splitting (pure decode helpers)
objects/lava/            Lava collision mask data + query methods
assets/                  Embedded images/GIFs (go:embed target)
final-background/        Rendered map PNG (go:embed target)
```

### Dependency Rules

Dependencies flow **downward only** — no circular imports.

```
main.go
  └─ internal/game
       ├─ internal/spawn
       │    ├─ objects/lava
       │    └─ world
       ├─ objects/atlas
       ├─ objects/lava
       └─ world
```

- `world/` imports nothing from this project (pure stdlib `math`).
- `objects/` packages import only `world/`.
- `internal/spawn/` imports `world/` and `objects/lava`.
- `internal/game/` may import any internal or objects package but **never** `main`.
- `main.go` orchestrates: reads embedded assets, constructs `Game`, calls `ebiten.RunGame`.

## Module Organization

Every package = **single responsibility**. When a concern outgrows its package:

1. Extract to a new package under `internal/` or `objects/`.
2. Export only the **minimum** set of types/functions callers need.
3. Keep internal types unexported. Export only what crosses the boundary.
4. Verify no circular dependencies with `go vet ./...`.

**Known debt**: `internal/game/` currently holds game state, combat, economy, UI, animations, and effects. When adding features, prefer extracting subsystems (e.g. `internal/combat/`, `internal/ui/`) over growing this package further.

## Go 1.25+ Style Rules

### Mandatory Idioms

- `for i := range n` for integer iteration (not `for i := 0; i < n; i++`).
- `any` instead of `interface{}`.
- Use `slices`, `maps`, `cmp` stdlib packages where applicable.
- Use `math/rand/v2` (not `math/rand`). Seed with explicit `rand.NewPCG`.
- Use `t.Context()` in tests when a context is needed (Go 1.24+).

### Naming

| Convention | Example | Rule |
|---|---|---|
| Constants with units | `enemySpeedPxPerSec`, `archerMaxRangePx` | Suffix with unit: `Px`, `PxPerSec`, `Sec`, `DPS` |
| Enum constants | `deathCausePlayerArrow` | `iota` enums, unexported, descriptive |
| Exported API types | `ArcherTierAssets`, `EnemyStripPair` | Only types crossing package boundaries |
| Internal structs | `worldArcher`, `worldEnemy`, `goldFloat` | Unexported, short, descriptive |

### Import Ordering

Two groups separated by a blank line:

```go
import (
    "bytes"
    "fmt"
    "math"

    "github.com/AlviDervishaj/idle-king/internal/spawn"
    "github.com/AlviDervishaj/idle-king/world"
    "github.com/hajimehoshi/ebiten/v2"
)
```

Group 1: stdlib. Group 2: all external (project internal + third-party), alphabetical.

### Error Handling

- Wrap with context: `fmt.Errorf("game: arrows pack: %w", err)`.
- `log.Fatal` only in `main.go`. Library packages return errors.
- Check all error returns. No `_` for error values except proven-safe cases.

## Game Development Patterns

### Ebitengine Lifecycle

| Method | Purpose | Notes |
|---|---|---|
| `Update()` | Game logic: input, combat, animation, economy | Called at 60 TPS |
| `Draw(screen)` | Render: map, archers, enemies, arrows, UI, laser | Same goroutine, after `Update` |
| `Layout(w,h)` | Returns fixed `(1440, 720)` | Pure, no state mutation |

### Frame Delta Timing

**Never use `ebiten.ActualTPS()` for movement.** Use `frameDT()`:

```go
func (g *Game) frameDT() float64 {
    now := time.Now()
    if g.lastFrameTime.IsZero() {
        g.lastFrameTime = now
        return 1.0 / 60.0 // defaultStep
    }
    dt := now.Sub(g.lastFrameTime).Seconds()
    g.lastFrameTime = now
    if dt > 0.12 { // maxStep — prevents catch-up explosions
        dt = 0.12
    }
    return dt
}
```

All movement/timers multiply by `dt`: `e.x += e.vx * dt`.

### Coordinate Systems

- **World pixels**: origin top-left of map image (1024×512 = 32×16 tiles at 32px each).
- **Screen pixels**: 1440×720 viewport (map scaled/centered with letterboxing).
- Convert with `screenToMapWorld()` and the `ox, oy, scale` triple in `Draw`.
- Tile lookup: `world.WorldToTile(px, py)` → `(row, col)`.
- **Feet convention**: entity `(x, y)` is bottom-center. Sprites draw with `Translate(-fw/2, -fh)`.

### Mutex Convention

`combatMu sync.RWMutex` guards: `enemies`, `arrows`, `spawnAccum`, `fireAccum`, `playTime`, `goldFloats`, `nextEnemyID`.

```go
// Write path (Update):
g.combatMu.Lock()
defer g.combatMu.Unlock()

// Read path (Draw):
g.combatMu.RLock()
defer g.combatMu.RUnlock()
```

## Idempotency Rules

Operations must be **safely repeatable** without unintended side effects:

1. **Pure asset decoding** — `decodeGIF`, `atlas.SplitPNGGrid` are pure functions. Same input → same output. No mutable global state.
2. **Spawn guards** — `PickEnemyFeet` and `RandomArcher` return `(x, y, ok)`. Callers check `ok`; failure is a no-op.
3. **Death re-entry guard** — `beginEnemyDeath` checks `if e.dying { return }`. Safe to call twice.
4. **Gold float cap** — `spawnGoldFloat` evicts oldest at `maxGoldFloats`. Repeated spawns degrade gracefully.
5. **Bounded catch-up** — timer accumulators use subtraction loops bounded by `maxSpawnCatchUpPerFrame`. No unbounded loops after long frames.
6. **Slice reuse** — combat update reuses slices with `enemies[:0]` pattern. No allocation, no stale references.

**Principle**: guard entry, bound iteration, degrade gracefully.

## Asset Management

1. Place files in `assets/` (or `final-background/` for map renders).
2. Single `//go:embed assets final-background` directive in `main.go` captures everything.
3. `readAsset(path)` panics on missing file (fail fast at startup).
4. Pass raw `[]byte` to `game.NewGame()` which decodes into Ebitengine images.
5. Path convention: `assets/<category>/<tier_or_index>/<filename>`.

**Note**: archer attack GIF names are inconsistent across tiers (`Attak1`, `Attack1`, `Attack`). The `archerAttackSuffixes` slice in `main.go` handles this. Normalize names when adding new assets.

## Testing

- **Table-driven tests** as the default pattern with anonymous struct slices.
- Test files: `_test.go` suffix, same package (white-box testing).
- Run: `go test ./...` — no external test dependencies.
- `Game` struct can be partially constructed for unit tests (populate only fields the test touches).
- **Highest-value targets**: pure functions (`archerTargetScore`, `leadLandingPoint`, `InBounds`, `WorldToTile`).

## Constants — No Magic Numbers

- All gameplay values (speeds, intervals, damage, costs) are named `const` at the top of the relevant file.
- Always include units in the name: `arrowSpeedPxPerSec`, `enemyDeathFrameSec`, `laserImpactRadiusPx`.
- Tuning tables (HP multipliers, gold rewards, hire costs) use package-level `var` slices indexed by tier.
- RNG seeds use hex literals with mnemonic values: `rand.NewPCG(0x49444c45, 0x4b494e47)` — ASCII for "IDLE", "KING".
