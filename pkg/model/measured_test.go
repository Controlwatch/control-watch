package model

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMeasuredJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     Measured[bool]
		wantJSON  string
		wantValue Measured[bool]
	}{
		{
			name:      "measured false remains a JSON boolean",
			value:     Measure(false),
			wantJSON:  "false",
			wantValue: Measure(false),
		},
		{
			name:      "unmeasurable becomes the stable marker",
			value:     CannotMeasure[bool]("RPC did not return the getter"),
			wantJSON:  `"UNMEASURABLE"`,
			wantValue: Measured[bool]{Unmeasurable: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if got := string(encoded); got != test.wantJSON {
				t.Fatalf("Marshal() = %s, want %s", got, test.wantJSON)
			}

			var decoded Measured[bool]
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if !reflect.DeepEqual(decoded, test.wantValue) {
				t.Fatalf("Unmarshal() = %#v, want %#v", decoded, test.wantValue)
			}
		})
	}
}

func TestMeasuredRejectsWrongValueType(t *testing.T) {
	t.Parallel()

	var measured Measured[bool]
	if err := json.Unmarshal([]byte(`"not-a-boolean"`), &measured); err == nil {
		t.Fatal("Unmarshal() error = nil, want a type error")
	}
}
