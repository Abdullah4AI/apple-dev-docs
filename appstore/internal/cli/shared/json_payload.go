package shared

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// JSONPayloadKind describes the top-level JSON shape a raw payload command accepts.
type JSONPayloadKind string

const (
	JSONPayloadObject JSONPayloadKind = "object"
	JSONPayloadArray  JSONPayloadKind = "array"
	JSONPayloadAny    JSONPayloadKind = "any"
)

// ReadJSONFilePayload loads a JSON object from a file path for commands that
// accept raw payload documents.
func ReadJSONFilePayload(path string) (json.RawMessage, error) {
	return ReadJSONFilePayloadKind(path, JSONPayloadObject)
}

// ReadJSONFilePayloadKind loads a JSON payload from a file path and validates
// the requested top-level JSON shape.
func ReadJSONFilePayloadKind(path string, kind JSONPayloadKind) (json.RawMessage, error) {
	file, err := openJSONPayloadFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("payload path must be a file")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, fmt.Errorf("payload file is empty")
	}

	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	switch kind {
	case JSONPayloadObject:
		if _, ok := payload.(map[string]any); !ok {
			return nil, fmt.Errorf("payload must be a JSON object")
		}
	case JSONPayloadArray:
		if _, ok := payload.([]any); !ok {
			return nil, fmt.Errorf("payload must be a JSON array")
		}
	case JSONPayloadAny, "":
	default:
		return nil, fmt.Errorf("unsupported JSON payload kind: %s", kind)
	}

	return json.RawMessage(data), nil
}

func openJSONPayloadFile(path string) (*os.File, error) {
	file, err := OpenExistingNoFollow(path)
	if err == nil {
		return file, nil
	}

	info, statErr := os.Lstat(path)
	if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
		return nil, err
	}

	// Keep compatibility with legacy command behavior: allow symlinked payload files.
	resolvedPath, resolveErr := filepath.EvalSymlinks(path)
	if resolveErr != nil {
		return nil, resolveErr
	}

	return OpenExistingNoFollow(resolvedPath)
}
