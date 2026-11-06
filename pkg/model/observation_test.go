package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestObservationJSONCarriesReasonsSeparately(t *testing.T) {
	t.Parallel()

	observation := Observation{
		Classification: ClassSEP57RWA,
		Token: &TokenSurface{
			WasmHash: Measure("abc123"),
			Paused:   CannotMeasure[bool]("getter unavailable"),
			Admin:    Measure("GADMIN"),
			Roles:    map[string]Measured[[]string]{},
		},
		Reasons: map[string]string{"token.paused": "getter unavailable"},
	}

	encoded, err := json.Marshal(observation)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	jsonText := string(encoded)
	for _, fragment := range []string{
		`"classification":"SEP57_RWA"`,
		`"paused":"UNMEASURABLE"`,
		`"reasons":{"token.paused":"getter unavailable"}`,
	} {
		if !strings.Contains(jsonText, fragment) {
			t.Fatalf("Marshal() = %s, want fragment %s", jsonText, fragment)
		}
	}
	if strings.Contains(jsonText, `"Reason"`) || strings.Contains(jsonText, `"reason":"getter unavailable"`) {
		t.Fatalf("Marshal() leaked internal measured reason: %s", jsonText)
	}
}
