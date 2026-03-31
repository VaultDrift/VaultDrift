package sync

import (
	"encoding/json"
	"testing"
)

func TestVectorClockIncrement(t *testing.T) {
	vc := NewVectorClock()

	if vc.Get("device1") != 0 {
		t.Error("initial counter should be 0")
	}

	vc.Increment("device1")
	if vc.Get("device1") != 1 {
		t.Errorf("counter should be 1, got %d", vc.Get("device1"))
	}

	vc.Increment("device1")
	if vc.Get("device1") != 2 {
		t.Errorf("counter should be 2, got %d", vc.Get("device1"))
	}
}

func TestVectorClockMerge(t *testing.T) {
	vc1 := NewVectorClock()
	vc1.Increment("device1")
	vc1.Increment("device1")

	vc2 := NewVectorClock()
	vc2.Increment("device2")
	vc2.Increment("device2")
	vc2.Increment("device2")

	vc1.Merge(vc2)

	if vc1.Get("device1") != 2 {
		t.Errorf("device1 counter should be 2, got %d", vc1.Get("device1"))
	}
	if vc1.Get("device2") != 3 {
		t.Errorf("device2 counter should be 3, got %d", vc1.Get("device2"))
	}
}

func TestVectorClockCompare(t *testing.T) {
	tests := []struct {
		name     string
		vc1      VectorClock
		vc2      VectorClock
		expected Ordering
	}{
		{
			name:     "equal clocks",
			vc1:      VectorClock{"a": 1, "b": 2},
			vc2:      VectorClock{"a": 1, "b": 2},
			expected: Equal,
		},
		{
			name:     "before - all counters less",
			vc1:      VectorClock{"a": 1, "b": 2},
			vc2:      VectorClock{"a": 2, "b": 3},
			expected: Before,
		},
		{
			name:     "after - all counters greater",
			vc1:      VectorClock{"a": 2, "b": 3},
			vc2:      VectorClock{"a": 1, "b": 2},
			expected: After,
		},
		{
			name:     "concurrent - mixed counters",
			vc1:      VectorClock{"a": 2, "b": 1},
			vc2:      VectorClock{"a": 1, "b": 2},
			expected: Concurrent,
		},
		{
			name:     "before - missing device",
			vc1:      VectorClock{"a": 1},
			vc2:      VectorClock{"a": 1, "b": 1},
			expected: Before,
		},
		{
			name:     "after - missing device in other",
			vc1:      VectorClock{"a": 1, "b": 1},
			vc2:      VectorClock{"a": 1},
			expected: After,
		},
		{
			name:     "concurrent - divergent devices",
			vc1:      VectorClock{"a": 2},
			vc2:      VectorClock{"b": 2},
			expected: Concurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.vc1.Compare(tt.vc2)
			if result != tt.expected {
				t.Errorf("Compare() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVectorClockIsConcurrent(t *testing.T) {
	vc1 := VectorClock{"a": 2, "b": 1}
	vc2 := VectorClock{"a": 1, "b": 2}

	if !vc1.IsConcurrent(vc2) {
		t.Error("clocks should be concurrent")
	}

	vc3 := VectorClock{"a": 1, "b": 1}
	vc4 := VectorClock{"a": 2, "b": 2}

	if vc3.IsConcurrent(vc4) {
		t.Error("clocks should not be concurrent")
	}
}

func TestVectorClockHappenedBefore(t *testing.T) {
	vc1 := VectorClock{"a": 1, "b": 2}
	vc2 := VectorClock{"a": 2, "b": 3}

	if !vc1.HappenedBefore(vc2) {
		t.Error("vc1 should have happened before vc2")
	}

	if vc2.HappenedBefore(vc1) {
		t.Error("vc2 should not have happened before vc1")
	}
}

func TestVectorClockCopy(t *testing.T) {
	vc1 := NewVectorClock()
	vc1.Increment("device1")
	vc1.Increment("device2")

	vc2 := vc1.Copy()

	// Modify original
	vc1.Increment("device1")

	// Copy should not be affected
	if vc2.Get("device1") != 1 {
		t.Error("copy should not be affected by changes to original")
	}

	// Verify independence
	vc2.Increment("device2")
	if vc1.Get("device2") != 1 {
		t.Error("original should not be affected by changes to copy")
	}
}

func TestVectorClockJSON(t *testing.T) {
	vc := NewVectorClock()
	vc.Increment("device1")
	vc.Increment("device1")
	vc.Increment("device2")

	// Marshal
	data, err := json.Marshal(vc)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// Unmarshal
	var vc2 VectorClock
	if err := json.Unmarshal(data, &vc2); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	// Verify
	if vc2.Get("device1") != 2 {
		t.Errorf("device1 counter should be 2, got %d", vc2.Get("device1"))
	}
	if vc2.Get("device2") != 1 {
		t.Errorf("device2 counter should be 1, got %d", vc2.Get("device2"))
	}
}

func TestVectorClockString(t *testing.T) {
	vc := VectorClock{"a": 1, "b": 2}
	s := vc.String()

	// String should contain device IDs and counters
	if s == "" {
		t.Error("String() should not return empty")
	}
}

// BenchmarkVectorClockCompare benchmarks comparison operations.
func BenchmarkVectorClockCompare(b *testing.B) {
	vc1 := VectorClock{
		"device1": 1000,
		"device2": 2000,
		"device3": 3000,
	}
	vc2 := VectorClock{
		"device1": 1001,
		"device2": 2000,
		"device3": 2999,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vc1.Compare(vc2)
	}
}

// BenchmarkVectorClockMerge benchmarks merge operations.
func BenchmarkVectorClockMerge(b *testing.B) {
	vc1 := VectorClock{
		"device1": 1000,
		"device2": 2000,
		"device3": 3000,
		"device4": 4000,
		"device5": 5000,
	}
	vc2 := VectorClock{
		"device1": 1001,
		"device2": 1999,
		"device3": 3001,
		"device4": 3999,
		"device5": 5001,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vc1.Copy().Merge(vc2)
	}
}
