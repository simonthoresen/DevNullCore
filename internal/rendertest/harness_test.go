// Package rendertest provides golden-file render tests for the server console and
// player chrome views. Run with -update to regenerate expected outputs:
//
//	go test ./internal/rendertest/ -update
package rendertest

import (
	"flag"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"null-space/internal/chrome"
	"null-space/internal/console"
	"null-space/internal/domain"
	"null-space/internal/render"
	"null-space/internal/state"
)

var update = flag.Bool("update", false, "regenerate golden files instead of comparing")

// fixedTime is the deterministic wall-clock value used across all render tests.
var fixedTime = time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)

// ─── Mock console API ────────────────────────────────────────────────────────

type mockConsoleAPI struct {
	st     *state.CentralState
	clock  *domain.MockClock
	chatCh chan domain.Message
	slogCh chan console.SlogLine
}

func newMockConsoleAPI(st *state.CentralState) *mockConsoleAPI {
	return &mockConsoleAPI{
		st:     st,
		clock:  &domain.MockClock{T: fixedTime},
		chatCh: make(chan domain.Message),
		slogCh: make(chan console.SlogLine),
	}
}

func (a *mockConsoleAPI) State() *state.CentralState                                      { return a.st }
func (a *mockConsoleAPI) Clock() domain.Clock                                              { return a.clock }
func (a *mockConsoleAPI) DataDir() string                                                  { return "" }
func (a *mockConsoleAPI) Uptime() string                                                   { return "0s" }
func (a *mockConsoleAPI) BroadcastChat(msg domain.Message)                                 {}
func (a *mockConsoleAPI) ChatCh() <-chan domain.Message                                    { return a.chatCh }
func (a *mockConsoleAPI) SlogCh() <-chan console.SlogLine                                  { return a.slogCh }
func (a *mockConsoleAPI) TabCandidates(string, []string) (string, []string)               { return "", nil }
func (a *mockConsoleAPI) DispatchCommand(string, domain.CommandContext)                    {}
func (a *mockConsoleAPI) SetConsoleWidth(int)                                              {}

// ─── Mock chrome API ─────────────────────────────────────────────────────────

type mockChromeAPI struct {
	st    *state.CentralState
	clock *domain.MockClock
}

func newMockChromeAPI(st *state.CentralState) *mockChromeAPI {
	return &mockChromeAPI{
		st:    st,
		clock: &domain.MockClock{T: fixedTime},
	}
}

func (a *mockChromeAPI) State() *state.CentralState                                      { return a.st }
func (a *mockChromeAPI) Clock() domain.Clock                                              { return a.clock }
func (a *mockChromeAPI) DataDir() string                                                  { return "" }
func (a *mockChromeAPI) Uptime() string                                                   { return "0s" }
func (a *mockChromeAPI) BroadcastChat(domain.Message)                                     {}
func (a *mockChromeAPI) BroadcastMsg(tea.Msg)                                             {}
func (a *mockChromeAPI) SendToPlayer(string, tea.Msg)                                     {}
func (a *mockChromeAPI) ServerLog(string)                                                 {}
func (a *mockChromeAPI) TabCandidates(string, []string) (string, []string)               { return "", nil }
func (a *mockChromeAPI) DispatchCommand(string, domain.CommandContext)                    {}
func (a *mockChromeAPI) StartGame()                                                       {}
func (a *mockChromeAPI) AcknowledgeGameOver(string)                                       {}
func (a *mockChromeAPI) SuspendGame(string) error                                         { return nil }
func (a *mockChromeAPI) ResumeGame(string, string) error                                  { return nil }
func (a *mockChromeAPI) ListSuspends() []state.SuspendInfo                                { return nil }
func (a *mockChromeAPI) KickPlayer(string) error                                          { return nil }

// ─── Mock game ───────────────────────────────────────────────────────────────

// mockGame is a minimal domain.Game that renders a fixed ASCII frame so that
// render tests don't depend on a real JS runtime.
type mockGame struct{}

func (g *mockGame) GameName() string                      { return "Test Game" }
func (g *mockGame) TeamRange() domain.TeamRange           { return domain.TeamRange{} }
func (g *mockGame) Init(any)                              {}
func (g *mockGame) Start()                                {}
func (g *mockGame) Update(float64)                        {}
func (g *mockGame) OnPlayerLeave(string)                  {}
func (g *mockGame) OnInput(string, string)                {}
func (g *mockGame) StatusBar(string) string               { return "" }
func (g *mockGame) CommandBar(string) string              { return "" }
func (g *mockGame) Commands() []domain.Command            { return nil }
func (g *mockGame) Menus() []domain.MenuDef               { return nil }
func (g *mockGame) CharMap() *render.CharMapDef           { return nil }
func (g *mockGame) RenderCanvas(string, int, int) []byte  { return nil }
func (g *mockGame) RenderCanvasImage(string, int, int) *image.RGBA { return nil }
func (g *mockGame) HasCanvasMode() bool                   { return false }
func (g *mockGame) Unload()                               {}
func (g *mockGame) State() any                            { return nil }
func (g *mockGame) SetState(any)                          {}
func (g *mockGame) GameSource() []domain.GameSourceFile   { return nil }
func (g *mockGame) GameAssets() []domain.GameAsset        { return nil }
func (g *mockGame) Layout(string, int, int) *domain.WidgetNode { return nil }

func (g *mockGame) Render(buf *render.ImageBuffer, _ string, x, y, w, h int) {
	// Draw a simple bordered box with fixed content.
	if w < 4 || h < 3 {
		return
	}
	buf.WriteString(x, y, "[ Test Game Output ]", nil, nil, 0)
	for row := 1; row < h-1; row++ {
		buf.WriteString(x, y+row, strings.Repeat(".", w), nil, nil, 0)
	}
	buf.WriteString(x, y+h-1, "[ game over: press enter ]", nil, nil, 0)
}

func (g *mockGame) RenderSplash(_ *render.ImageBuffer, _ string, _, _, _, _ int) bool {
	return false // use framework default
}

func (g *mockGame) RenderGameOver(_ *render.ImageBuffer, _ string, _, _, _, _ int, _ []domain.GameResult) bool {
	return false // use framework default
}

// ─── Golden file helpers ─────────────────────────────────────────────────────

// goldenPath returns the path for a golden file given scenario and variant names.
func goldenPath(scenario, file string) string {
	return filepath.Join("testdata", "renders", scenario, file+".txt")
}

// checkOrUpdate either writes the golden file (when -update is set) or
// asserts that the current output matches it.
func checkOrUpdate(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		t.Logf("updated %s", path)
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file missing: %s\n  run with -update to generate it", path)
	}
	want := string(raw)
	if got != want {
		t.Errorf("render mismatch for %s\n--- got ---\n%s\n--- want ---\n%s",
			path, got, want)
	}
}

// stripRender strips all ANSI/OSC escape codes and returns the plain text.
func stripRender(s string) string {
	return ansi.Strip(s)
}

// renderConsole creates a console model with the given API, sends a window-size
// message, and returns the ANSI-stripped render content.
func renderConsole(api console.ServerAPI, profile colorprofile.Profile, w, h int) string {
	m := console.NewModel(api, func() {}, profile)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return stripRender(m2.View().Content)
}

// renderChrome creates a chrome model for the given player, applies variant
// settings, optionally marks it as active in-game, then returns the
// ANSI-stripped render content.
func renderChrome(
	api chrome.ServerAPI,
	playerID string,
	inActiveGame bool,
	gameName string,
	variant chromeVariant,
	profile colorprofile.Profile,
	w, h int,
) string {
	m := chrome.NewModel(api, playerID)
	m.IsEnhancedClient = variant.isEnhancedClient
	m.IsTerminalClient = variant.isTerminalClient
	m.ColorProfile = profile

	m2, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	if inActiveGame {
		m2, _ = m2.Update(domain.GameLoadedMsg{Name: gameName})
	}
	return stripRender(m2.View().Content)
}

// ─── Chrome variant definitions ──────────────────────────────────────────────

// chromeVariant describes an execution context for the player chrome.
type chromeVariant struct {
	// name is used as the sub-test name.
	name string
	// label is a comment shown in the golden file header.
	label            string
	isEnhancedClient bool
	isTerminalClient bool
}

// chromeVariants lists the four execution contexts the developer cares about,
// in order:
//
//	a) server --local (plain SSH pipe to local terminal)
//	b) server + plain ssh client (SSH pipe to remote terminal)
//	c) client --local (enhanced, terminal-mode client process)
//	d) server + client.exe (enhanced graphical client)
var chromeVariants = []chromeVariant{
	{
		name:             "server_local",
		label:            "a) server --local (SSH pipe to local terminal)",
		isEnhancedClient: false,
		isTerminalClient: false,
	},
	{
		name:             "server_ssh",
		label:            "b) server + plain SSH client (SSH pipe to remote terminal)",
		isEnhancedClient: false,
		isTerminalClient: false,
	},
	{
		name:             "client_local",
		label:            "c) client --local (enhanced terminal-mode client)",
		isEnhancedClient: true,
		isTerminalClient: true,
	},
	{
		name:             "client_remote",
		label:            "d) server + client.exe (enhanced graphical client)",
		isEnhancedClient: true,
		isTerminalClient: false,
	},
}

// colorVariants defines the two terminal color modes tested per variant.
// "ascii" uses the NoTTY profile (no escape codes at all).
// "ansi"  uses the TrueColor profile then strips codes — exercises the full
// ANSI serialization path while still yielding plain-text for comparison.
type colorVariantDef struct {
	name    string
	profile colorprofile.Profile
}

var colorVariants = []colorVariantDef{
	{name: "ascii", profile: colorprofile.NoTTY},
	{name: "ansi", profile: colorprofile.TrueColor},
}

// termW and termH are the fixed terminal dimensions used in all render tests.
const termW, termH = 80, 24
