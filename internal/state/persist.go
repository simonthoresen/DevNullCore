package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadGameState reads the saved state for a game from dist/state/<gameName>.json.
// Returns nil (no error) if the file does not exist.
func LoadGameState(dataDir, gameName string) (any, error) {
	path := filepath.Join(dataDir, "state", gameName+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read game state: %w", err)
	}
	var s any
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse game state: %w", err)
	}
	return s, nil
}

// SaveGameState writes game state to dist/state/<gameName>.json.
// Creates the state/ directory if it does not exist. Does nothing if state is nil.
func SaveGameState(dataDir, gameName string, s any) error {
	if s == nil {
		return nil
	}
	dir := filepath.Join(dataDir, "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal game state: %w", err)
	}
	path := filepath.Join(dir, gameName+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write game state: %w", err)
	}
	return nil
}
