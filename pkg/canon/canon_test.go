package canon

import (
	"testing"
)

func TestMarshalOrdersNestedObjectKeys(t *testing.T) {
	t.Parallel()

	value := map[string]any{
		"z": 1,
		"a": map[string]any{"y": 2, "x": 1},
		"m": []any{map[string]any{"b": 2, "a": 1}, 3},
	}
	encoded, err := Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	const want = `{"a":{"x":1,"y":2},"m":[{"a":1,"b":2},3],"z":1}`
	if got := string(encoded); got != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}

func TestHashIgnoresMapInsertionOrder(t *testing.T) {
	t.Parallel()

	left := map[string]any{"alpha": 1, "beta": 2}
	right := map[string]any{"beta": 2, "alpha": 1}
	leftHash, err := Hash(left)
	if err != nil {
		t.Fatalf("Hash(left) error = %v", err)
	}
	rightHash, err := Hash(right)
	if err != nil {
		t.Fatalf("Hash(right) error = %v", err)
	}
	if leftHash != rightHash {
		t.Fatalf("hashes differ: %s != %s", leftHash, rightHash)
	}
}
