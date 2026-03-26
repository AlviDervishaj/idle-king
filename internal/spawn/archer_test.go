package spawn

import (
	"testing"

	"github.com/AlviDervishaj/idle-king/objects/lava"
)

func TestArcherCandidateCount(t *testing.T) {
	s := NewArcherSpawner(lava.Default())
	if s.CandidateCount() < 50 {
		t.Fatalf("expected many spawn tiles west of lava, got %d", s.CandidateCount())
	}
}
