package report

import (
	"encoding/json"
	"fmt"

	"github.com/Controlwatch/control-watch/pkg/model"
)

// JSON returns an indented report document terminated by one newline.
func JSON(value model.Report) ([]byte, error) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render JSON report: %w", err)
	}
	return append(encoded, '\n'), nil
}
