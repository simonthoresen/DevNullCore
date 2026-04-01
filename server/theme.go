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

// Theme defines the NC-style chrome palette as 4 depth layers, each with a
// background and foreground color. Theme authors choose whether layers share
// colors or are distinct.
//
//   Layer 0 — Desktop:  action bar, the "background" behind everything
//   Layer 1 — Menu:     dropdown menus pulled from the action bar
//   Layer 2 — Dialog:   modal dialog boxes
//   Layer 3 — Popup:    nested popups inside dialogs (confirmations, selectors)
//
// Additional colors: disabled text, drop shadow, and a highlight (selection)
// pair used for focused items, title bars, and active buttons.
type Theme struct {
	Name string `json:"name"`

	// Layer 0 — Desktop / action bar
	DesktopBg string `json:"desktopBg"`
	DesktopFg string `json:"desktopFg"`

	// Layer 1 — Menu (dropdown panels)
	MenuBg string `json:"menuBg"`
	MenuFg string `json:"menuFg"`

	// Layer 2 — Dialog (modal boxes)
	DialogBg string `json:"dialogBg"`
	DialogFg string `json:"dialogFg"`

	// Layer 3 — Popup (nested over dialogs)
	PopupBg string `json:"popupBg"`
	PopupFg string `json:"popupFg"`

	// Highlight — selected/focused items, title bars, active buttons
	HighlightBg string `json:"highlightBg"`
	HighlightFg string `json:"highlightFg"`

	// Disabled items
	DisabledFg string `json:"disabledFg"`

	// Drop shadow
	ShadowBg string `json:"shadowBg"`

	// Border characters (single-char strings; empty = use defaults)
	BorderTL  string `json:"borderTL"`  // top-left corner     (default "┌")
	BorderTR  string `json:"borderTR"`  // top-right corner    (default "┐")
	BorderBL  string `json:"borderBL"`  // bottom-left corner  (default "└")
	BorderBR  string `json:"borderBR"`  // bottom-right corner (default "┘")
	BorderH   string `json:"borderH"`   // horizontal bar      (default "─")
	BorderV   string `json:"borderV"`   // vertical bar        (default "│")
	BorderTL2 string `json:"borderTeeL"` // left tee            (default "├")
	BorderTR2 string `json:"borderTeeR"` // right tee           (default "┤")
	BarSep    string `json:"barSep"`    // action bar separator (default "│")
}

// tc converts a hex string to a color.Color, returning fallback if empty.
func tc(hex, fallback string) color.Color {
	if hex == "" {
		return lipgloss.Color(fallback)
	}
	return lipgloss.Color(hex)
}

// Layer 0 — Desktop
func (t *Theme) DesktopBgC() color.Color { return tc(t.DesktopBg, "#000080") }
func (t *Theme) DesktopFgC() color.Color { return tc(t.DesktopFg, "#AAAAAA") }

// Layer 1 — Menu
func (t *Theme) MenuBgC() color.Color { return tc(t.MenuBg, "#AAAAAA") }
func (t *Theme) MenuFgC() color.Color { return tc(t.MenuFg, "#000000") }

// Layer 2 — Dialog
func (t *Theme) DialogBgC() color.Color { return tc(t.DialogBg, "#AAAAAA") }
func (t *Theme) DialogFgC() color.Color { return tc(t.DialogFg, "#000000") }

// Layer 3 — Popup
func (t *Theme) PopupBgC() color.Color { return tc(t.PopupBg, "#AAAAAA") }
func (t *Theme) PopupFgC() color.Color { return tc(t.PopupFg, "#000000") }

// Highlight
func (t *Theme) HighlightBgC() color.Color { return tc(t.HighlightBg, "#000080") }
func (t *Theme) HighlightFgC() color.Color { return tc(t.HighlightFg, "#FFFFFF") }

// Extras
func (t *Theme) DisabledFgC() color.Color { return tc(t.DisabledFg, "#888888") }
func (t *Theme) ShadowBgC() color.Color   { return tc(t.ShadowBg, "#333333") }

// ts returns s if non-empty, otherwise fallback.
func ts(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// Border character accessors.
func (t *Theme) TL() string   { return ts(t.BorderTL, "┌") }
func (t *Theme) TR() string   { return ts(t.BorderTR, "┐") }
func (t *Theme) BL() string   { return ts(t.BorderBL, "└") }
func (t *Theme) BR() string   { return ts(t.BorderBR, "┘") }
func (t *Theme) H() string    { return ts(t.BorderH, "─") }
func (t *Theme) V() string    { return ts(t.BorderV, "│") }
func (t *Theme) TeeL() string { return ts(t.BorderTL2, "├") }
func (t *Theme) TeeR() string { return ts(t.BorderTR2, "┤") }
func (t *Theme) Sep() string  { return ts(t.BarSep, "│") }

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
