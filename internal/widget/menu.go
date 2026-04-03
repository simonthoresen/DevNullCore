package widget

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"null-space/internal/domain"
)


// ─── Shortcut-key helpers ──────────────────────────────────────────────────────

// StripAmpersand returns the display label with the "&" removed,
// and the lowercase shortcut character (or 0 if none).
// e.g. "&File" → ("File", 'f'), "E&xit" → ("Exit", 'x'), "Help" → ("Help", 0)
func StripAmpersand(label string) (string, rune) {
	idx := strings.IndexByte(label, '&')
	if idx < 0 || idx >= len(label)-1 {
		return label, 0
	}
	clean := label[:idx] + label[idx+1:]
	shortcut := rune(strings.ToLower(label[idx+1 : idx+2])[0])
	return clean, shortcut
}

// RenderLabel renders a label with the shortcut character underlined.
// base is the normal style; accent highlights the shortcut char.
func RenderLabel(label string, base, accent lipgloss.Style) string {
	idx := strings.IndexByte(label, '&')
	if idx < 0 || idx >= len(label)-1 {
		return base.Render(label)
	}
	before := label[:idx]
	hotkey := label[idx+1 : idx+2]
	after := label[idx+2:]
	return base.Render(before) + accent.Render(hotkey) + base.Render(after)
}

// MenuShortcut returns the shortcut rune for a MenuDef (from its Label).
func MenuShortcut(m domain.MenuDef) rune {
	_, r := StripAmpersand(m.Label)
	return r
}

// ItemShortcut returns the shortcut rune for a MenuItemDef (from its Label).
func ItemShortcut(it domain.MenuItemDef) rune {
	_, r := StripAmpersand(it.Label)
	return r
}

// HotkeyDisplay converts a key binding string (e.g. "ctrl+c") to a display
// string (e.g. "(Ctrl+C)").
func HotkeyDisplay(key string) string {
	parts := strings.Split(key, "+")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return "(" + strings.Join(parts, "+") + ")"
}

// TruncateStyled truncates a styled string to the given visual width.
func TruncateStyled(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(text) <= width {
		return text
	}
	return ansi.Truncate(text, width, "")
}

// ─── Overlay state ─────────────────────────────────────────────────────────────

// DialogFocusZone identifies which part of the dialog has keyboard focus.
type DialogFocusZone int

const (
	DialogFocusButtons DialogFocusZone = iota // default: buttons
	DialogFocusList                           // list items (when ListItems is set)
	DialogFocusInput                          // text input (when InputPrompt is set)
)

// OverlayState holds all per-player NC overlay UI state.
type OverlayState struct {
	MenuFocused bool // F10 was pressed; action bar is focused
	MenuCursor  int  // which menu title is highlighted
	OpenMenu    int  // index of open dropdown (-1 = none)
	DropCursor  int  // focused item index in open dropdown

	Dialogs     []domain.DialogRequest
	DialogFocus int // focused button in top dialog

	// List dialog state (when top dialog has ListItems).
	DialogListCursor int // selected item index
	DialogListScroll int // scroll offset (0 = top visible)

	// Input dialog state (when top dialog has InputPrompt).
	DialogInputValue  string
	DialogInputCursor int

	// Which zone of the dialog has focus.
	DialogZone DialogFocusZone
}

// ShowDialogMsg is sent to a player's Bubble Tea program to push a dialog.
type ShowDialogMsg struct{ Dialog domain.DialogRequest }

func (o *OverlayState) HasDialog() bool { return len(o.Dialogs) > 0 }

func (o *OverlayState) TopDialog() *domain.DialogRequest {
	if len(o.Dialogs) == 0 {
		return nil
	}
	return &o.Dialogs[len(o.Dialogs)-1]
}

func (o *OverlayState) PushDialog(d domain.DialogRequest) {
	o.Dialogs = append(o.Dialogs, d)
	o.DialogFocus = 0
	o.DialogListCursor = 0
	o.DialogListScroll = 0
	o.DialogInputValue = ""
	o.DialogInputCursor = 0
	// Default focus: list if present, then input, then buttons.
	if len(d.ListItems) > 0 {
		o.DialogZone = DialogFocusList
	} else if d.InputPrompt != "" {
		o.DialogZone = DialogFocusInput
	} else {
		o.DialogZone = DialogFocusButtons
	}
}

func (o *OverlayState) PopDialog() {
	if len(o.Dialogs) > 0 {
		o.Dialogs = o.Dialogs[:len(o.Dialogs)-1]
		o.DialogFocus = 0
		o.DialogListCursor = 0
		o.DialogListScroll = 0
		o.DialogInputValue = ""
		o.DialogInputCursor = 0
		// Reset zone for the next dialog on the stack.
		if d := o.TopDialog(); d != nil {
			if len(d.ListItems) > 0 {
				o.DialogZone = DialogFocusList
			} else if d.InputPrompt != "" {
				o.DialogZone = DialogFocusInput
			} else {
				o.DialogZone = DialogFocusButtons
			}
		} else {
			o.DialogZone = DialogFocusButtons
		}
	}
}

// IsActive returns true when any overlay is intercepting input.
func (o *OverlayState) IsActive() bool {
	return o.HasDialog() || o.MenuFocused || o.OpenMenu >= 0
}

// ─── Key handling ──────────────────────────────────────────────────────────────

// HandleKey routes a key press through the overlay state machine.
// Returns true if the key was consumed and normal chrome should not process it.
func (o *OverlayState) HandleKey(key string, menus []domain.MenuDef, playerID string) bool {
	// Global hotkeys: check all menu items for matching hotkey bindings.
	for _, m := range menus {
		for _, it := range m.Items {
			if it.Hotkey != "" && it.Hotkey == key && !it.Disabled && it.Handler != nil {
				o.OpenMenu = -1
				o.MenuFocused = false
				it.Handler(playerID)
				return true
			}
		}
	}

	if o.HasDialog() {
		return o.HandleDialogKey(key)
	}
	if key == "f10" {
		if o.MenuFocused || o.OpenMenu >= 0 {
			o.MenuFocused = false
			o.OpenMenu = -1
		} else {
			o.MenuFocused = true
			o.MenuCursor = 0
			o.OpenMenu = -1
		}
		return true
	}
	// Alt+letter opens a menu by its shortcut key (e.g. Alt+F for "&File").
	if strings.HasPrefix(key, "alt+") && len(key) == 5 {
		letter := rune(key[4])
		for i, m := range menus {
			if MenuShortcut(m) == letter {
				o.MenuFocused = true
				o.MenuCursor = i
				o.OpenMenu = i
				o.DropCursor = FirstSelectable(menus[i].Items)
				return true
			}
		}
	}
	if o.OpenMenu >= 0 {
		return o.handleDropdownKey(key, menus, playerID)
	}
	if o.MenuFocused {
		return o.handleMenuBarKey(key, menus)
	}
	return false
}

func (o *OverlayState) handleMenuBarKey(key string, menus []domain.MenuDef) bool {
	n := len(menus)
	if n == 0 {
		return true
	}
	switch key {
	case "left":
		if o.MenuCursor > 0 {
			o.MenuCursor--
		} else {
			o.MenuCursor = n - 1
		}
	case "right":
		o.MenuCursor = (o.MenuCursor + 1) % n
	case "down", "enter":
		o.OpenMenu = o.MenuCursor
		o.DropCursor = FirstSelectable(menus[o.MenuCursor].Items)
	case "esc":
		o.MenuFocused = false
	default:
		// Letter key → open menu by shortcut (e.g. "f" for "&File").
		if len(key) == 1 {
			letter := rune(key[0])
			for i, m := range menus {
				if MenuShortcut(m) == letter {
					o.MenuCursor = i
					o.OpenMenu = i
					o.DropCursor = FirstSelectable(menus[i].Items)
					return true
				}
			}
		}
	}
	return true // consume all keys while menu bar is focused
}

func (o *OverlayState) handleDropdownKey(key string, menus []domain.MenuDef, playerID string) bool {
	if o.OpenMenu >= len(menus) {
		return false
	}
	items := menus[o.OpenMenu].Items
	n := len(menus)
	switch key {
	case "up":
		o.DropCursor = PrevSelectable(items, o.DropCursor)
	case "down":
		o.DropCursor = NextSelectable(items, o.DropCursor)
	case "left":
		if o.MenuCursor > 0 {
			o.MenuCursor--
		} else {
			o.MenuCursor = n - 1
		}
		o.OpenMenu = o.MenuCursor
		o.DropCursor = FirstSelectable(menus[o.MenuCursor].Items)
	case "right":
		o.MenuCursor = (o.MenuCursor + 1) % n
		o.OpenMenu = o.MenuCursor
		o.DropCursor = FirstSelectable(menus[o.MenuCursor].Items)
	case "enter":
		if o.DropCursor >= 0 && o.DropCursor < len(items) {
			item := items[o.DropCursor]
			if !item.Disabled && item.Handler != nil {
				o.OpenMenu = -1
				o.MenuFocused = false
				item.Handler(playerID)
			}
		}
	case "esc":
		o.OpenMenu = -1
		// leave MenuFocused = true so user is back on the bar
	default:
		// Letter key → activate item by shortcut (e.g. "s" for "&Save").
		if len(key) == 1 {
			letter := rune(key[0])
			for i, it := range items {
				if !it.Disabled && !IsSeparator(it) && ItemShortcut(it) == letter {
					if it.Handler != nil {
						o.OpenMenu = -1
						o.MenuFocused = false
						it.Handler(playerID)
					}
					_ = i
					return true
				}
			}
		}
	}
	return true
}

func (o *OverlayState) HandleDialogKey(key string) bool {
	d := o.TopDialog()
	if d == nil {
		return false
	}
	btns := d.Buttons
	if len(btns) == 0 {
		btns = []string{"OK"}
	}
	hasList := len(d.ListItems) > 0
	hasInput := d.InputPrompt != ""

	// Tab cycles focus zones: list → input → buttons → list ...
	// When only buttons exist (no list/input), Tab cycles button focus instead.
	if key == "tab" || key == "shift+tab" {
		zones := []DialogFocusZone{}
		if hasList {
			zones = append(zones, DialogFocusList)
		}
		if hasInput {
			zones = append(zones, DialogFocusInput)
		}
		zones = append(zones, DialogFocusButtons)

		if len(zones) == 1 {
			// Only buttons — Tab cycles button focus (original behavior).
			if key == "tab" {
				o.DialogFocus = (o.DialogFocus + 1) % len(btns)
			} else {
				o.DialogFocus = (o.DialogFocus - 1 + len(btns)) % len(btns)
			}
			return true
		}
		cur := 0
		for i, z := range zones {
			if z == o.DialogZone {
				cur = i
				break
			}
		}
		if key == "tab" {
			cur = (cur + 1) % len(zones)
		} else {
			cur = (cur - 1 + len(zones)) % len(zones)
		}
		o.DialogZone = zones[cur]
		return true
	}

	// Esc always dismisses.
	if key == "esc" {
		o.fireDialogClose(d, "")
		return true
	}

	// Route to the focused zone.
	switch o.DialogZone {
	case DialogFocusList:
		return o.handleDialogListKey(key, d, btns)
	case DialogFocusInput:
		return o.handleDialogInputKey(key, d, btns)
	default:
		return o.handleDialogButtonKey(key, d, btns)
	}
}

func (o *OverlayState) handleDialogListKey(key string, d *domain.DialogRequest, btns []string) bool {
	n := len(d.ListItems)
	switch key {
	case "up":
		if o.DialogListCursor > 0 {
			o.DialogListCursor--
			o.ensureListVisible(d)
		}
	case "down":
		if o.DialogListCursor < n-1 {
			o.DialogListCursor++
			o.ensureListVisible(d)
		}
	case "pgup":
		o.DialogListCursor -= o.dialogListVisibleHeight(d)
		if o.DialogListCursor < 0 {
			o.DialogListCursor = 0
		}
		o.ensureListVisible(d)
	case "pgdown":
		o.DialogListCursor += o.dialogListVisibleHeight(d)
		if o.DialogListCursor >= n {
			o.DialogListCursor = n - 1
		}
		o.ensureListVisible(d)
	case "home":
		o.DialogListCursor = 0
		o.ensureListVisible(d)
	case "end":
		o.DialogListCursor = n - 1
		o.ensureListVisible(d)
	case "enter", " ":
		// Activate the focused button with the selected list item.
		o.fireDialogClose(d, btns[o.DialogFocus])
	default:
		// Left/right navigate buttons even while list is focused.
		if key == "left" && o.DialogFocus > 0 {
			o.DialogFocus--
		} else if key == "right" && o.DialogFocus < len(btns)-1 {
			o.DialogFocus++
		}
	}
	return true
}

func (o *OverlayState) handleDialogInputKey(key string, d *domain.DialogRequest, btns []string) bool {
	switch key {
	case "enter":
		o.fireDialogClose(d, btns[o.DialogFocus])
	case "backspace":
		if o.DialogInputCursor > 0 {
			o.DialogInputValue = o.DialogInputValue[:o.DialogInputCursor-1] + o.DialogInputValue[o.DialogInputCursor:]
			o.DialogInputCursor--
		}
	case "delete":
		if o.DialogInputCursor < len(o.DialogInputValue) {
			o.DialogInputValue = o.DialogInputValue[:o.DialogInputCursor] + o.DialogInputValue[o.DialogInputCursor+1:]
		}
	case "left":
		if o.DialogInputCursor > 0 {
			o.DialogInputCursor--
		}
	case "right":
		if o.DialogInputCursor < len(o.DialogInputValue) {
			o.DialogInputCursor++
		}
	case "home":
		o.DialogInputCursor = 0
	case "end":
		o.DialogInputCursor = len(o.DialogInputValue)
	default:
		// Insert printable characters.
		if len(key) == 1 && key[0] >= ' ' && key[0] < 0x7f {
			o.DialogInputValue = o.DialogInputValue[:o.DialogInputCursor] + key + o.DialogInputValue[o.DialogInputCursor:]
			o.DialogInputCursor++
		}
	}
	return true
}

func (o *OverlayState) handleDialogButtonKey(key string, d *domain.DialogRequest, btns []string) bool {
	switch key {
	case "left":
		if o.DialogFocus > 0 {
			o.DialogFocus--
		}
	case "right":
		if o.DialogFocus < len(btns)-1 {
			o.DialogFocus++
		}
	case "up":
		// Move focus back to list or input if available.
		if d.InputPrompt != "" {
			o.DialogZone = DialogFocusInput
		} else if len(d.ListItems) > 0 {
			o.DialogZone = DialogFocusList
		}
	case "enter", " ":
		o.fireDialogClose(d, btns[o.DialogFocus])
	}
	return true
}

// fireDialogClose invokes the appropriate callback and pops the dialog.
func (o *OverlayState) fireDialogClose(d *domain.DialogRequest, button string) {
	listIdx := o.DialogListCursor
	inputVal := o.DialogInputValue
	cbList := d.OnListAction
	cbInput := d.OnInputClose
	cbClose := d.OnClose
	o.PopDialog()
	switch {
	case cbList != nil && len(d.ListItems) > 0:
		cbList(button, listIdx)
	case cbInput != nil && d.InputPrompt != "":
		cbInput(button, inputVal)
	case cbClose != nil:
		cbClose(button)
	}
}

// dialogListVisibleHeight returns the number of visible list rows in the dialog.
const dialogMaxListH = 12

func (o *OverlayState) dialogListVisibleHeight(d *domain.DialogRequest) int {
	h := len(d.ListItems)
	if h > dialogMaxListH {
		h = dialogMaxListH
	}
	return h
}

// ensureListVisible adjusts scroll so the cursor is visible.
func (o *OverlayState) ensureListVisible(d *domain.DialogRequest) {
	visH := o.dialogListVisibleHeight(d)
	if o.DialogListCursor < o.DialogListScroll {
		o.DialogListScroll = o.DialogListCursor
	}
	if o.DialogListCursor >= o.DialogListScroll+visH {
		o.DialogListScroll = o.DialogListCursor - visH + 1
	}
}

// ─── Mouse handling ───────────────────────────────────────────────────────────

// HandleClick processes a left mouse click at screen position (x, y).
// ncBarRow is the screen row of the action bar. screenW/screenH are for dialog centering.
// Returns true if the click was consumed by the overlay.
func (o *OverlayState) HandleClick(x, y, ncBarRow, screenW, screenH int, menus []domain.MenuDef, playerID string) bool {
	// Priority 1: dialog (topmost overlay)
	if o.HasDialog() {
		return o.HandleDialogClick(x, y, screenW, screenH)
	}

	// Priority 2: open dropdown
	if o.OpenMenu >= 0 && o.OpenMenu < len(menus) {
		if o.handleDropdownClick(x, y, ncBarRow, menus, playerID) {
			return true
		}
		// Click outside dropdown — close it
		o.OpenMenu = -1
		o.MenuFocused = false
		// Fall through to check if click was on the bar itself
	}

	// Priority 3: action bar row
	if y == ncBarRow && len(menus) > 0 {
		pos := MenuBarPositions(menus)
		for i, m := range menus {
			clean, _ := StripAmpersand(m.Label)
			startX := pos[i]
			endX := startX + len(clean) + 2 // " label "
			if x >= startX && x < endX {
				o.MenuFocused = true
				o.MenuCursor = i
				o.OpenMenu = i
				o.DropCursor = FirstSelectable(menus[i].Items)
				return true
			}
		}
		// Click on bar but not on a menu title — just activate the bar
		o.MenuFocused = true
		return true
	}

	return false
}

func (o *OverlayState) handleDropdownClick(x, y, ncBarRow int, menus []domain.MenuDef, playerID string) bool {
	items := menus[o.OpenMenu].Items
	if len(items) == 0 {
		return false
	}

	// Dropdown position
	pos := MenuBarPositions(menus)
	ddCol := 0
	if o.OpenMenu < len(pos) {
		ddCol = pos[o.OpenMenu]
	}
	ddRow := ncBarRow + 1 // dropdown starts one row below bar

	// Calculate dropdown dimensions
	maxLW := 0
	for _, it := range items {
		if !IsSeparator(it) {
			clean, _ := StripAmpersand(it.Label)
			if len(clean) > maxLW {
				maxLW = len(clean)
			}
		}
	}
	innerW := maxLW + 2
	if innerW < 14 {
		innerW = 14
	}
	boxW := innerW + 2 // borders

	// Check if click is inside the dropdown box
	relX := x - ddCol
	relY := y - ddRow
	if relX < 0 || relX >= boxW || relY < 0 {
		return false
	}

	// Count rendered lines: top border + items (separators count as 1 line each) + bottom border
	lineIdx := 0
	for i, it := range items {
		lineIdx++ // each item/separator is one line (after top border)
		if relY == lineIdx && !IsSeparator(it) && !it.Disabled {
			if it.Handler != nil {
				o.DropCursor = i
				o.OpenMenu = -1
				o.MenuFocused = false
				it.Handler(playerID)
			}
			return true
		}
	}
	return relY <= lineIdx+1 // consumed if inside box area
}

// dialogLayout computes the inner width, total dimensions, and position
// for the current top dialog. Mirrors the logic in RenderDialog.
func (o *OverlayState) dialogLayout(screenW, screenH int) (innerW, totalW, totalH, col, row int) {
	d := o.TopDialog()
	if d == nil {
		return
	}
	btns := d.Buttons
	if len(btns) == 0 {
		btns = []string{"OK"}
	}
	hasList := len(d.ListItems) > 0
	hasInput := d.InputPrompt != ""

	bodyLines := strings.Split(d.Body, "\n")
	maxBodyW := 0
	for _, l := range bodyLines {
		w := ansi.StringWidth(l)
		if w > maxBodyW {
			maxBodyW = w
		}
	}
	maxListW := 0
	if hasList {
		for i, item := range d.ListItems {
			w := ansi.StringWidth(item) + 5
			if i < len(d.ListTags) && d.ListTags[i] != "" {
				w += 2 + ansi.StringWidth(d.ListTags[i])
			}
			if w > maxListW {
				maxListW = w
			}
		}
	}
	inputLabelW := 0
	if hasInput {
		inputLabelW = ansi.StringWidth(d.InputPrompt) + 2
	}
	btnW := 0
	for _, b := range btns {
		btnW += len(b) + 4
	}
	btnW += (len(btns) - 1) * 2

	innerW = maxBodyW + 2
	if maxListW+2 > innerW {
		innerW = maxListW + 2
	}
	if inputLabelW+12 > innerW {
		innerW = inputLabelW + 12
	}
	if ansi.StringWidth(d.Title)+2 > innerW {
		innerW = ansi.StringWidth(d.Title) + 2
	}
	if btnW+2 > innerW {
		innerW = btnW + 2
	}
	if innerW < 22 {
		innerW = 22
	}

	contentRows := len(bodyLines)
	if hasList {
		contentRows = o.dialogListVisibleHeight(d)
	}

	totalLines := 3 + contentRows + 2 + 1 // top+title+sep + content + sep+buttons + bottom
	if hasInput {
		totalLines += 2
	}
	totalW = innerW + 2
	totalH = totalLines
	col = max(0, (screenW-totalW)/2)
	row = max(2, (screenH-totalH)/2)
	return
}

func (o *OverlayState) HandleDialogClick(x, y, screenW, screenH int) bool {
	d := o.TopDialog()
	if d == nil {
		return false
	}
	btns := d.Buttons
	if len(btns) == 0 {
		btns = []string{"OK"}
	}
	hasList := len(d.ListItems) > 0

	innerW, totalW, totalH, col, row := o.dialogLayout(screenW, screenH)

	relX := x - col
	relY := y - row

	// Click on a list item?
	if hasList {
		contentRows := o.dialogListVisibleHeight(d)
		listStartRow := 3 // after top + title + sep
		listEndRow := listStartRow + contentRows
		if relY >= listStartRow && relY < listEndRow && relX >= 0 && relX < totalW {
			clickedIdx := o.DialogListScroll + (relY - listStartRow)
			if clickedIdx >= 0 && clickedIdx < len(d.ListItems) {
				o.DialogListCursor = clickedIdx
				o.DialogZone = DialogFocusList
			}
			return true
		}
	}

	// Click on a button?
	btnW := 0
	for _, b := range btns {
		btnW += len(b) + 4
	}
	btnW += (len(btns) - 1) * 2
	btnRowY := row + totalH - 2
	if y == btnRowY {
		lpad := max(0, (innerW-btnW)/2)
		bx := col + 1 + lpad
		for i, b := range btns {
			bw := len(b) + 4
			if x >= bx && x < bx+bw {
				o.fireDialogClose(d, btns[i])
				return true
			}
			bx += bw + 2
		}
	}

	return relX >= 0 && relX < totalW+1 && relY >= 0 && relY < totalH+1
}

// ─── Selectable-item helpers ───────────────────────────────────────────────────

func IsSeparator(item domain.MenuItemDef) bool {
	return strings.TrimLeft(item.Label, "-") == ""
}

func FirstSelectable(items []domain.MenuItemDef) int {
	for i, it := range items {
		if !IsSeparator(it) && !it.Disabled {
			return i
		}
	}
	return 0
}

func NextSelectable(items []domain.MenuItemDef, cur int) int {
	for i := cur + 1; i < len(items); i++ {
		if !IsSeparator(items[i]) && !items[i].Disabled {
			return i
		}
	}
	return cur
}

func PrevSelectable(items []domain.MenuItemDef, cur int) int {
	for i := cur - 1; i >= 0; i-- {
		if !IsSeparator(items[i]) && !items[i].Disabled {
			return i
		}
	}
	return cur
}

