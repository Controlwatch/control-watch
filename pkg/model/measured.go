package model

import (
	"bytes"
	"encoding/json"
)

// UnmeasurableMarker is the stable JSON representation of an unreadable value.
const UnmeasurableMarker = "UNMEASURABLE"

// Measured wraps a value that may be unreadable. Its JSON representation is
// either the value itself or the string "UNMEASURABLE". The reason is carried
// separately in Observation.Reasons.
type Measured[T any] struct {
	Value        T
	Unmeasurable bool
	Reason       string
}

// Measure constructs a readable measured value.
func Measure[T any](value T) Measured[T] {
	return Measured[T]{Value: value}
}

// CannotMeasure constructs an unreadable value with its internal reason.
func CannotMeasure[T any](reason string) Measured[T] {
	return Measured[T]{Unmeasurable: true, Reason: reason}
}

// MarshalJSON emits the value directly or the stable unreadable marker.
func (m Measured[T]) MarshalJSON() ([]byte, error) {
	if m.Unmeasurable {
		return json.Marshal(UnmeasurableMarker)
	}
	return json.Marshal(m.Value)
}

// UnmarshalJSON restores either a measured value or the unreadable marker.
// Reasons are restored separately from Observation.Reasons.
func (m *Measured[T]) UnmarshalJSON(data []byte) error {
	marker, err := json.Marshal(UnmeasurableMarker)
	if err != nil {
		return err
	}
	if bytes.Equal(bytes.TrimSpace(data), marker) {
		var zero T
		*m = Measured[T]{Value: zero, Unmeasurable: true}
		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*m = Measured[T]{Value: value}
	return nil
}
