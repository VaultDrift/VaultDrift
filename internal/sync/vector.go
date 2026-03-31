// Package sync provides the delta sync protocol with Merkle trees and vector clocks.
package sync

import (
	"encoding/json"
	"fmt"
)

// VectorClock tracks the logical time for each device in the system.
// It maps device IDs to monotonic counters.
type VectorClock map[string]uint64

// Ordering represents the relationship between two vector clocks.
type Ordering int

const (
	// Before means clock1 happened before clock2.
	Before Ordering = -1
	// Equal means clock1 and clock2 represent the same state.
	Equal Ordering = 0
	// After means clock1 happened after clock2.
	After Ordering = 1
	// Concurrent means clock1 and clock2 are concurrent (conflict).
	Concurrent Ordering = 2
)

// NewVectorClock creates a new empty vector clock.
func NewVectorClock() VectorClock {
	return make(VectorClock)
}

// Increment increments the counter for the given device.
func (v VectorClock) Increment(deviceID string) {
	v[deviceID]++
}

// Get returns the counter for a device, or 0 if not present.
func (v VectorClock) Get(deviceID string) uint64 {
	return v[deviceID]
}

// Merge combines another vector clock into this one, taking the maximum
// value for each device.
func (v VectorClock) Merge(other VectorClock) {
	for device, counter := range other {
		if counter > v[device] {
			v[device] = counter
		}
	}
}

// Compare determines the ordering between two vector clocks.
func (v VectorClock) Compare(other VectorClock) Ordering {
	superset := false
	subset := false

	// Check all devices in this clock
	for device, counter := range v {
		otherCounter := other[device]
		if counter > otherCounter {
			superset = true
		} else if counter < otherCounter {
			subset = true
		}
	}

	// Check devices only in other clock
	for device, counter := range other {
		if _, exists := v[device]; !exists && counter > 0 {
			subset = true
		}
	}

	switch {
	case !superset && !subset:
		return Equal
	case superset && !subset:
		return After
	case !superset && subset:
		return Before
	default:
		return Concurrent
	}
}

// IsConcurrent returns true if this clock is concurrent with another.
func (v VectorClock) IsConcurrent(other VectorClock) bool {
	return v.Compare(other) == Concurrent
}

// HappenedBefore returns true if this clock happened before another.
func (v VectorClock) HappenedBefore(other VectorClock) bool {
	return v.Compare(other) == Before
}

// Copy creates a deep copy of the vector clock.
func (v VectorClock) Copy() VectorClock {
	copy := make(VectorClock, len(v))
	for device, counter := range v {
		copy[device] = counter
	}
	return copy
}

// MarshalJSON implements json.Marshaler.
func (v VectorClock) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]uint64(v))
}

// UnmarshalJSON implements json.Unmarshaler.
func (v *VectorClock) UnmarshalJSON(data []byte) error {
	var m map[string]uint64
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*v = m
	return nil
}

// String returns a string representation of the vector clock.
func (v VectorClock) String() string {
	return fmt.Sprintf("%v", map[string]uint64(v))
}
