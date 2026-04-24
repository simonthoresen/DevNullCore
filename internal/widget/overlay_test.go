package widget

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"dev-null/internal/domain"
	"dev-null/internal/engine"
)

// ─── Overlay / Menu tests ────────────────────────────────────────────────────

func TestMenuShortcut(t *testing.T) {
	_, r := StripAmpersand("&File")
	if r != 'f' {
		t.Errorf("expected 'f', got %c", r)
	}
	_, r = StripAmpersand("E&xit")
	if r != 'x' {
		t.Errorf("expected 'x', got %c", r)
	}
	_, r = StripAmpersand("NoShortcut")
	if r != 0 {
		t.Errorf("expected 0, got %c", r)
	}
}

func TestOverlayMenuNavigation(t *testing.T) {
	o := OverlayState{OpenMenu: -1}
	menus := []domain.MenuDef{
		{Label: "&File", Items: []domain.MenuItemDef{{Label: "&New"}, {Label: "&Open"}}},
		{Label: "&Edit", Items: []domain.MenuItemDef{{Label: "&Copy"}}},
	}

	// Menu activation comes from the input router (via ActionActivateMenu);
	// here we set that state directly to simulate Esc on Desktop.
	o.MenuFocused = true
	o.MenuCursor = 0
	o.OpenMenu = -1

	// Right arrow moves to Edit.
	o.HandleKey("right", menus, "")
	if o.MenuCursor != 1 {
		t.Errorf("expected cursor 1, got %d", o.MenuCursor)
	}

	// Down opens dropdown.
	o.HandleKey("down", menus, "")
	if o.OpenMenu != 1 {
		t.Errorf("expected openMenu 1, got %d", o.OpenMenu)
	}

	// Esc closes dropdown.
	o.HandleKey("esc", menus, "")
	if o.OpenMenu != -1 {
		t.Errorf("expected openMenu -1, got %d", o.OpenMenu)
	}

	// Esc again deactivates bar.
	o.HandleKey("esc", menus, "")
	if o.MenuFocused {
		t.Error("expected menuFocused=false after second Esc")
	}
}

// Ctrl/Alt menu shortcuts were removed so games can freely use the
// modifier keyspace. Menu items must be reached by navigating via Esc →
// menu bar → ampersand-letter shortcut inside the menu mode; tests for
// those flows live in TestOverlayMenuNavigation and the render golden
// scenarios.

func TestOverlayDialogStack(t *testing.T) {
	o := OverlayState{OpenMenu: -1}

	if o.HasDialog() {
		t.Error("should have no dialog initially")
	}

	o.PushDialog(domain.DialogRequest{Title: "First", Body: "A"})
	o.PushDialog(domain.DialogRequest{Title: "Second", Body: "B"})

	if !o.HasDialog() {
		t.Error("should have dialogs")
	}
	if d := o.TopDialog(); d.Title != "Second" {
		t.Errorf("expected top 'Second', got %q", d.Title)
	}

	o.PopDialog()
	if d := o.TopDialog(); d.Title != "First" {
		t.Errorf("expected top 'First', got %q", d.Title)
	}

	o.PopDialog()
	if o.HasDialog() {
		t.Error("should have no dialogs after popping both")
	}
}

// ─── About dialog click detection ────────────────────────────────────────────

func TestAboutDialogClickDetection(t *testing.T) {
	o := OverlayState{OpenMenu: -1}
	body := engine.AboutLogo()
	o.PushDialog(domain.DialogRequest{
		Title:   "About",
		Body:    body,
		Buttons: []string{"OK"},
	})

	screenW, screenH := 120, 30
	layer := testTheme().LayerAt(2)

	// Get the rendered dialog position.
	buf, renderCol, renderRow := o.RenderDialogBuf(screenW, screenH, layer)
	if buf == nil {
		t.Fatal("RenderDialogBuf returned nil buffer")
	}

	// Click inside dialog bounds should be consumed (modal).
	clickX := renderCol + buf.Width/2
	clickY := renderRow + buf.Height/2
	consumed := o.HandleDialogClick(clickX, clickY, screenW, screenH)
	if !consumed {
		t.Error("click inside dialog bounds was not consumed")
	}

	// Click outside dialog bounds should also be consumed (modal).
	consumed = o.HandleDialogClick(0, 0, screenW, screenH)
	if !consumed {
		t.Error("click outside dialog bounds was not consumed (modal)")
	}

	// Dialog can be dismissed via Enter key on the focused OK button.
	o.HandleDialogMsg(tea.KeyPressMsg{Code: -1, Text: "enter"})
	if o.HasDialog() {
		t.Error("dialog should have been dismissed after Enter")
	}
}

func TestDialogClickMultiButton(t *testing.T) {
	o := OverlayState{OpenMenu: -1}
	var clicked string
	o.PushDialog(domain.DialogRequest{
		Title:   "Confirm",
		Body:    "Are you sure?",
		Buttons: []string{"Yes", "No", "Cancel"},
		OnClose: func(btn string) { clicked = btn },
	})

	screenW, screenH := 80, 24
	layer := testTheme().LayerAt(2)

	// Render to verify dialog is present.
	buf, _, _ := o.RenderDialogBuf(screenW, screenH, layer)
	if buf == nil {
		t.Fatal("RenderDialogBuf returned nil buffer")
	}

	// Click inside dialog is consumed (modal behavior).
	consumed := o.HandleDialogClick(40, 12, screenW, screenH)
	if !consumed {
		t.Error("click inside dialog should be consumed")
	}

	// Use keyboard to navigate to "No" and press it.
	// First button (Yes) is focused by default; Tab moves to No.
	o.HandleDialogMsg(tea.KeyPressMsg{Code: -1, Text: "tab"})
	o.HandleDialogMsg(tea.KeyPressMsg{Code: -1, Text: "enter"})
	if o.HasDialog() {
		t.Error("dialog should be dismissed after pressing No")
	}
	if clicked != "No" {
		t.Errorf("expected OnClose('No'), got %q", clicked)
	}
}

func TestDialogClickButton(t *testing.T) {
	o := OverlayState{OpenMenu: -1}
	var clicked string
	o.PushDialog(domain.DialogRequest{
		Title:   "Confirm",
		Body:    "Click a button",
		Buttons: []string{"OK", "Cancel"},
		OnClose: func(btn string) { clicked = btn },
	})

	screenW, screenH := 80, 24
	layer := testTheme().LayerAt(2)

	// Render to populate layout caches.
	buf, renderCol, renderRow := o.RenderDialogBuf(screenW, screenH, layer)
	if buf == nil {
		t.Fatal("RenderDialogBuf returned nil buffer")
	}

	// Find the "[ OK ]" text in the rendered buffer to get button coordinates.
	found := false
	for row := 0; row < buf.Height; row++ {
		var line strings.Builder
		for col := 0; col < buf.Width; col++ {
			line.WriteRune(buf.Pixels[row*buf.Width+col].Char)
		}
		if idx := strings.Index(line.String(), "[ OK ]"); idx >= 0 {
			// Click the center of the OK button in screen coordinates.
			clickX := renderCol + idx + 3 // center of "[ OK ]"
			clickY := renderRow + row
			consumed := o.HandleDialogClick(clickX, clickY, screenW, screenH)
			if !consumed {
				t.Error("click on button should be consumed")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("could not find '[ OK ]' text in rendered dialog buffer")
	}

	if o.HasDialog() {
		t.Error("dialog should be dismissed after clicking OK button")
	}
	if clicked != "OK" {
		t.Errorf("expected OnClose('OK'), got %q", clicked)
	}
}

func TestAboutDialogKeyDismiss(t *testing.T) {
	o := OverlayState{OpenMenu: -1}
	body := engine.AboutLogo()
	o.PushDialog(domain.DialogRequest{
		Title:   "About",
		Body:    body,
		Buttons: []string{"OK"},
	})

	if !o.HasDialog() {
		t.Fatal("dialog should be open")
	}

	// Press Enter to close (activates the focused OK button).
	consumed, _ := o.HandleDialogMsg(tea.KeyPressMsg{Code: -1, Text: "enter"})
	if !consumed {
		t.Error("enter should be consumed by dialog")
	}
	if o.HasDialog() {
		t.Error("dialog should be dismissed after Enter")
	}
}
