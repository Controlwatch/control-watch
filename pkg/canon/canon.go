// Package canon provides deterministic JSON encoding and SHA-256 hashing for
// ControlWatch snapshots and their semantic observations.
package canon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// Marshal returns compact canonical JSON. Object keys are ordered
// lexicographically, array order is preserved, and custom JSON marshalers are
// honored before canonicalization.
func Marshal(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal source JSON: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("decode source JSON: %w", err)
	}

	var canonical bytes.Buffer
	if err := writeValue(&canonical, normalized); err != nil {
		return nil, err
	}
	return canonical.Bytes(), nil
}

// Hash returns the lowercase hexadecimal SHA-256 digest of canonical JSON.
func Hash(value any) (string, error) {
	canonical, err := Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func writeValue(destination *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		destination.WriteString("null")
	case bool:
		if typed {
			destination.WriteString("true")
		} else {
			destination.WriteString("false")
		}
	case string:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Errorf("marshal string: %w", err)
		}
		destination.Write(encoded)
	case json.Number:
		destination.WriteString(typed.String())
	case []any:
		destination.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				destination.WriteByte(',')
			}
			if err := writeValue(destination, item); err != nil {
				return err
			}
		}
		destination.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		destination.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				destination.WriteByte(',')
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return fmt.Errorf("marshal object key: %w", err)
			}
			destination.Write(encodedKey)
			destination.WriteByte(':')
			if err := writeValue(destination, typed[key]); err != nil {
				return err
			}
		}
		destination.WriteByte('}')
	default:
		return fmt.Errorf("unsupported normalized JSON type %T", value)
	}
	return nil
}
