package server

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme defines the color palette for the NC-style chrome (action bar,
// dropdowns, dialogs, and their shadows).
type Theme struct {
	Name string `json:"name"`

	// Action bar
	BarBg       string `json:"barBg"`
	BarFg       string `json:"barFg"`
	BarActiveBg string `json:"barActiveBg"`
	BarActiveFg string `json:"barActiveFg"`

	// Boxes (dropdowns, dialogs)
	BoxBg      string `json:"boxBg"`
	BoxFg      string `json:"boxFg"`
	BoxTitleBg string `json:"boxTitleBg"`
	BoxTitleFg string `json:"boxTitleFg"`

	// Active button / highlighted item
	BtnActiveBg string `json:"btnActiveBg"`
	BtnActiveFg string `json:"btnActiveFg"`

	// Disabled items
	DisabledFg string `json:"disabledFg"`

	// Drop shadow
	ShadowBg string `json:"shadowBg"`
}

// tc converts a hex string to a color.Color, returning fallback if empty.
func tc(hex, fallback string) color.Color {
	if hex == "" {
		return lipgloss.Color(fallback)
	}
	return lipgloss.Color(hex)
}

func (t *Theme) BarBgC() color.Color       { return tc(t.BarBg, "#000080") }
func (t *Theme) BarFgC() color.Color       { return tc(t.BarFg, "#AAAAAA") }
func (t *Theme) BarActiveBgC() color.Color { return tc(t.BarActiveBg, "#AAAAAA") }
func (t *Theme) BarActiveFgC() color.Color { return tc(t.BarActiveFg, "#000080") }
func (t *Theme) BoxBgC() color.Color       { return tc(t.BoxBg, "#AAAAAA") }
func (t *Theme) BoxFgC() color.Color       { return tc(t.BoxFg, "#000000") }
func (t *Theme) BoxTitleBgC() color.Color  { return tc(t.BoxTitleBg, "#000080") }
func (t *Theme) BoxTitleFgC() color.Color  { return tc(t.BoxTitleFg, "#FFFFFF") }
func (t *Theme) BtnActiveBgC() color.Color { return tc(t.BtnActiveBg, "#000080") }
func (t *Theme) BtnActiveFgC() color.Color { return tc(t.BtnActiveFg, "#FFFFFF") }
func (t *Theme) DisabledFgC() color.Color  { return tc(t.DisabledFg, "#888888") }
func (t *Theme) ShadowBgC() color.Color    { return tc(t.ShadowBg, "#333333") }

// LoadTheme reads a theme JSON file and returns the parsed Theme.
func LoadTheme(path string) (*Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse theme %s: %w", path, err)
	}
	if t.Name == "" {
		t.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	}
	return &t, nil
}

// DefaultTheme returns the built-in norton theme (all fields empty = use defaults).
func DefaultTheme() *Theme {
	return &Theme{Name: "norton"}
}

// ListThemes returns the names of available theme files in the themes directory.
func ListThemes(dataDir string) []string {
	dir := filepath.Join(dataDir, "themes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return names
}
