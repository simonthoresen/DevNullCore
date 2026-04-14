package localcmd

import (
	"strings"

	"dev-null/internal/domain"
	"dev-null/internal/state"
	"dev-null/internal/widget"
)

// ─── Save dialog ──────────────────────────────────────────────────────────────

// SaveDialogOptions configures the Saves dialog.
type SaveDialogOptions struct {
	DataDir      string
	Overlay      *widget.OverlayState
	SelectedSave string // "gameName/saveName" of the selected entry, or ""
	CanLoad      bool   // show Load button (chrome admin only)
	CanRemove    bool   // show Remove button (console only)
	OnLoad       func(gameName, saveName string)             // called when Load is confirmed
	OnRemove     func(gameName, saveName string, cursor int) // called when Remove is confirmed
	Reload       func(cursor int)
}

// PushSaveDialog opens the Saves dialog on opts.Overlay.
func PushSaveDialog(cursor int, opts SaveDialogOptions) {
	saves := state.ListSuspends(opts.DataDir, "")
	if len(saves) == 0 {
		opts.Overlay.PushDialog(domain.DialogRequest{
			Title:   "Saves",
			Body:    "No saves found.",
			Buttons: []string{"Close"},
		})
		return
	}
	items := make([]string, len(saves))
	tags := make([]string, len(saves))
	for i, s := range saves {
		key := s.GameName + "/" + s.SaveName
		items[i] = key
		if key == opts.SelectedSave {
			tags[i] = "(●)"
		} else {
			tags[i] = "(○)"
		}
	}
	var btns []string
	if opts.CanLoad && opts.SelectedSave != "" {
		btns = append(btns, "Load")
	}
	if opts.CanRemove && opts.SelectedSave != "" {
		btns = append(btns, "Remove")
	}
	btns = append(btns, "Close")
	opts.Overlay.PushDialog(domain.DialogRequest{
		Title:     "Saves",
		ListItems: items,
		ListTags:  tags,
		Buttons:   btns,
		OnListEnter: func(idx int) {
			opts.SelectedSave = items[idx]
			opts.Overlay.PopDialog()
			PushSaveDialog(idx, opts)
		},
		OnListAction: func(btn string, idx int) {
			s := saves[idx]
			switch btn {
			case "Load":
				if opts.OnLoad != nil && opts.SelectedSave != "" {
					if s2 := saveByKey(saves, opts.SelectedSave); s2 != nil {
						opts.OnLoad(s2.GameName, s2.SaveName)
					} else {
						opts.OnLoad(s.GameName, s.SaveName)
					}
				}
			case "Remove":
				if opts.OnRemove != nil && opts.SelectedSave != "" {
					if s2 := saveByKey(saves, opts.SelectedSave); s2 != nil {
						opts.OnRemove(s2.GameName, s2.SaveName, idx)
					} else {
						opts.OnRemove(s.GameName, s.SaveName, idx)
					}
				}
			}
		},
	})
	opts.Overlay.SetTopCursor(cursor)
}

// saveByKey returns the SuspendInfo for "gameName/saveName", or nil if not found.
func saveByKey(saves []state.SuspendInfo, key string) *state.SuspendInfo {
	for i := range saves {
		if saves[i].GameName+"/"+saves[i].SaveName == key {
			return &saves[i]
		}
	}
	return nil
}

// ─── Standalone "Add..." dialogs for sub-menu use ────────────────────────────

// PushGameAddDialog opens an input dialog for adding a game by name or URL.
func PushGameAddDialog(overlay *widget.OverlayState, dataDir string, onLoad func(name string)) {
	overlay.PushDialog(domain.DialogRequest{
		Title:       "Add Game",
		Body:        "Enter a game name or URL:",
		InputPrompt: "Game",
		Buttons:     []string{"Load", "Cancel"},
		OnInputClose: func(btn, value string) {
			value = strings.TrimSpace(value)
			if btn == "Load" && value != "" && onLoad != nil {
				onLoad(value)
			}
		},
	})
}

// PushThemeAddDialog opens an input dialog for loading a theme by name.
func PushThemeAddDialog(overlay *widget.OverlayState, dataDir string, onLoad func(name string)) {
	overlay.PushDialog(domain.DialogRequest{
		Title:       "Add Theme",
		Body:        "Enter a theme name:",
		InputPrompt: "Theme",
		Buttons:     []string{"Load", "Cancel"},
		OnInputClose: func(btn, value string) {
			value = strings.TrimSpace(value)
			if btn == "Load" && value != "" && onLoad != nil {
				onLoad(value)
			}
		},
	})
}

// PushScriptAddDialog opens an input dialog for adding a plugin or shader.
func PushScriptAddDialog(overlay *widget.OverlayState, noun string, onLoad func(name string)) {
	overlay.PushDialog(domain.DialogRequest{
		Title:       "Add " + noun,
		Body:        "Enter a " + strings.ToLower(noun) + " name or URL:",
		InputPrompt: noun,
		Buttons:     []string{"Load", "Cancel"},
		OnInputClose: func(btn, value string) {
			value = strings.TrimSpace(value)
			if btn == "Load" && value != "" && onLoad != nil {
				onLoad(value)
			}
		},
	})
}

// PushSynthAddDialog opens an input dialog for adding a SoundFont.
func PushSynthAddDialog(overlay *widget.OverlayState, onLoad func(name string)) {
	overlay.PushDialog(domain.DialogRequest{
		Title:       "Add SoundFont",
		Body:        "Enter a SoundFont name:",
		InputPrompt: "SoundFont",
		Buttons:     []string{"Load", "Cancel"},
		OnInputClose: func(btn, value string) {
			value = strings.TrimSpace(value)
			if btn == "Load" && value != "" && onLoad != nil {
				onLoad(value)
			}
		},
	})
}

// PushFontAddDialog opens an input dialog for adding a Figlet font.
func PushFontAddDialog(overlay *widget.OverlayState, onLoad func(name string)) {
	overlay.PushDialog(domain.DialogRequest{
		Title:       "Add Font",
		Body:        "Enter a font name:",
		InputPrompt: "Font",
		Buttons:     []string{"Load", "Cancel"},
		OnInputClose: func(btn, value string) {
			value = strings.TrimSpace(value)
			if btn == "Load" && value != "" && onLoad != nil {
				onLoad(value)
			}
		},
	})
}
