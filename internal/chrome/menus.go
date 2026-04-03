package chrome

import (
	"fmt"
	"path/filepath"
	"strings"

	"null-space/internal/domain"
	"null-space/internal/engine"
)

// allMenus returns the full ordered list of menus for the NC action bar:
// the framework "File" menu followed by any game-registered menus.
// invalidateMenuCache forces the next cachedMenus() call to rebuild.
func (m *Model) invalidateMenuCache() {
	m.menuCache = nil
}

// cachedMenus returns the menu tree, rebuilding only when the active game has changed.
func (m *Model) cachedMenus() []domain.MenuDef {
	m.api.State().RLock()
	game := m.api.State().ActiveGame
	m.api.State().RUnlock()

	if m.menuCache != nil && m.menuCacheGame == game {
		return m.menuCache
	}

	fileItems := []domain.MenuItemDef{
		{Label: "&Resume Game...", Handler: func(_ string) { m.showResumeGameDialog() }},
		{Label: "---"},
		{Label: "&Themes...", Handler: func(_ string) { m.showPlayerListDialog("Themes", "themes", ".json") }},
		{Label: "&Plugins...", Handler: func(_ string) { m.showPlayerListDialog("Plugins", "plugins", ".js") }},
		{Label: "&Shaders...", Handler: func(_ string) { m.showShaderDialog() }},
		{Label: "---"},
	}
	if m.IsLocal {
		fileItems = append(fileItems, domain.MenuItemDef{
			Label: "&Quit",
			Handler: func(_ string) {
				// Ctrl+C is the reliable quit path in local mode.
			},
		})
	} else {
		fileItems = append(fileItems, domain.MenuItemDef{
			Label: "&Disconnect",
			Handler: func(playerID string) {
				go m.api.KickPlayer(playerID)
			},
		})
	}
	menus := []domain.MenuDef{{Label: "&File", Items: fileItems}}
	if game != nil {
		menus = append(menus, game.Menus()...)
	}
	menus = append(menus, domain.MenuDef{
		Label: "&Help",
		Items: []domain.MenuItemDef{
			{Label: "&About...", Handler: func(_ string) {
				m.overlay.PushDialog(domain.DialogRequest{
					Title:   "About",
					Body:    engine.AboutLogo(),
					Buttons: []string{"OK"},
				})
			}},
		},
	})
	m.menuCache = menus
	m.menuCacheGame = game
	return menus
}

func (m *Model) showResumeGameDialog() {
	saves := m.api.ListSuspends()
	if len(saves) == 0 {
		m.overlay.PushDialog(domain.DialogRequest{
			Title:   "Resume Game",
			Body:    "No suspended games found.",
			Buttons: []string{"OK"},
		})
		return
	}

	teamCount := m.api.State().TeamCount()

	var lines []string
	var buttons []string
	for i, s := range saves {
		if i >= 9 {
			break // limit to 9 saves in the dialog
		}
		teamNote := ""
		if s.TeamCount != teamCount {
			teamNote = fmt.Sprintf("  (lobby has %d teams)", teamCount)
		}
		lines = append(lines, fmt.Sprintf("  %d. %s/%s  (%d teams, %s)%s",
			i+1, s.GameName, s.SaveName, s.TeamCount, s.SavedAt.Format(domain.TimeFormatShort), teamNote))
		buttons = append(buttons, fmt.Sprintf("%d", i+1))
	}
	buttons = append(buttons, "Cancel")

	body := strings.Join(lines, "\n")

	// Capture saves slice for the OnClose callback.
	capturedSaves := saves
	m.overlay.PushDialog(domain.DialogRequest{
		Title:   "Resume Game",
		Body:    body,
		Buttons: buttons,
		OnClose: func(button string) {
			if button == "Cancel" || button == "" {
				return
			}
			idx := 0
			fmt.Sscanf(button, "%d", &idx)
			if idx < 1 || idx > len(capturedSaves) {
				return
			}
			s := capturedSaves[idx-1]
			if err := m.api.ResumeGame(s.GameName, s.SaveName); err != nil {
				m.overlay.PushDialog(domain.DialogRequest{
					Title:   "Resume Failed",
					Body:    err.Error(),
					Buttons: []string{"OK"},
				})
			}
		},
	})
}

func (m *Model) showPlayerListDialog(title, subdir, ext string) {
	dir := filepath.Join(m.api.DataDir(), subdir)
	items := engine.ListDir(dir, ext)
	body := "(empty)"
	if len(items) > 0 {
		var lines []string
		for _, name := range items {
			lines = append(lines, "  "+name)
		}
		body = strings.Join(lines, "\n")
	}
	m.overlay.PushDialog(domain.DialogRequest{
		Title:   title,
		Body:    body,
		Buttons: []string{"Close"},
	})
}

func (m *Model) showShaderDialog() {
	m.pushShaderDialog()
}

func (m *Model) pushShaderDialog() {
	available := engine.ListDir(filepath.Join(m.api.DataDir(), "shaders"), ".js")
	loadedSet := make(map[string]bool)
	for _, n := range m.shaderNames {
		loadedSet[n] = true
	}

	// Build flat list: active shaders first (in order), then unloaded available ones.
	var items []string
	var tags []string
	for i, name := range m.shaderNames {
		items = append(items, fmt.Sprintf("%d. %s", i+1, name))
		tags = append(tags, "[active]")
	}
	for _, name := range available {
		if !loadedSet[name] {
			items = append(items, name)
			tags = append(tags, "")
		}
	}
	if len(items) == 0 {
		m.overlay.PushDialog(domain.DialogRequest{
			Title:   "Shaders",
			Body:    "No shaders found in shaders/",
			Buttons: []string{"Add", "Close"},
			OnClose: func(button string) {
				if button == "Add" {
					m.showShaderAddDialog()
				}
			},
		})
		return
	}

	activeCount := len(m.shaderNames)
	m.overlay.PushDialog(domain.DialogRequest{
		Title:    "Shaders",
		ListItems: items,
		ListTags:  tags,
		Buttons:  []string{"Add", "Remove", "Up", "Down", "Close"},
		OnListAction: func(button string, idx int) {
			switch button {
			case "Add":
				m.showShaderAddDialog()
			case "Remove":
				if idx < activeCount {
					name := m.shaderNames[idx]
					m.showShaderRemoveConfirm(name)
				} else {
					m.overlay.PushDialog(domain.DialogRequest{
						Title:   "Remove",
						Body:    "Only active shaders can be removed.",
						Buttons: []string{"OK"},
						OnClose: func(_ string) { m.pushShaderDialog() },
					})
				}
			case "Up":
				if idx > 0 && idx < activeCount {
					m.moveShader(m.shaderNames[idx], -1)
					m.pushShaderDialog()
				}
			case "Down":
				if idx >= 0 && idx < activeCount-1 {
					m.moveShader(m.shaderNames[idx], +1)
					m.pushShaderDialog()
				}
			case "Close", "":
				// done
			}
		},
	})
}

func (m *Model) showShaderAddDialog() {
	m.overlay.PushDialog(domain.DialogRequest{
		Title:        "Add Shader",
		Body:         "Enter a shader name or URL:",
		InputPrompt:  "Shader",
		Buttons:      []string{"Load", "Cancel"},
		OnInputClose: func(button, value string) {
			if button == "Load" && strings.TrimSpace(value) != "" {
				m.handleShaderCommand("/shader load " + strings.TrimSpace(value))
			}
			m.pushShaderDialog()
		},
	})
}

func (m *Model) showShaderRemoveConfirm(name string) {
	m.overlay.PushDialog(domain.DialogRequest{
		Title:   "Confirm Remove",
		Body:    fmt.Sprintf("Remove shader '%s'?", name),
		Buttons: []string{"Remove", "Cancel"},
		OnClose: func(button string) {
			if button == "Remove" {
				m.handleShaderCommand("/shader unload " + name)
			}
			m.pushShaderDialog()
		},
	})
}
