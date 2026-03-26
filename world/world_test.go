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
