package rendertest

import (
	"null-space/internal/domain"
	"null-space/internal/state"
)

// renderScenario describes a server state to render plus the chrome player context.
type renderScenario struct {
	// name is used as the sub-test name and as the testdata directory name.
	name string
	// setup configures the CentralState before rendering. It is called once
	// and the same state is shared for both the console and chrome renders.
	setup func(st *state.CentralState)
	// playerID is the player whose chrome view is rendered. Defaults to "alice".
	playerID string
	// inActiveGame sends a GameLoadedMsg to the chrome model so it enters
	// the playing/splash/game-over rendering path.
	inActiveGame bool
	// gameName is used in GameLoadedMsg when inActiveGame is true.
	gameName string
}

// scenarios is the curated eval set. Edit this list to add, remove, or tweak
// render test cases. Run `go test ./internal/rendertest/ -update` to
// regenerate the golden files after changing state or expected layout.
var scenarios = []renderScenario{
	// ── Lobby ──────────────────────────────────────────────────────────────
	{
		name:     "lobby_empty",
		playerID: "alice",
		setup: func(st *state.CentralState) {
			st.Lock()
			defer st.Unlock()
			st.Players["alice"] = &domain.Player{
				ID: "alice", Name: "Alice", IsAdmin: true,
				TermWidth: termW, TermHeight: termH,
			}
		},
	},
	{
		name:     "lobby_two_players_two_teams",
		playerID: "alice",
		setup: func(st *state.CentralState) {
			st.Lock()
			defer st.Unlock()
			st.Players["alice"] = &domain.Player{
				ID: "alice", Name: "Alice", IsAdmin: true,
				TermWidth: termW, TermHeight: termH,
			}
			st.Players["bob"] = &domain.Player{
				ID: "bob", Name: "Bob",
				TermWidth: termW, TermHeight: termH,
			}
			st.Teams = []domain.Team{
				{Name: "Red", Color: "#ff5555", Players: []string{"alice"}},
				{Name: "Blue", Color: "#5555ff", Players: []string{"bob"}},
			}
		},
	},
	{
		name:     "lobby_chat_history",
		playerID: "alice",
		setup: func(st *state.CentralState) {
			st.Lock()
			defer st.Unlock()
			st.Players["alice"] = &domain.Player{
				ID: "alice", Name: "Alice", IsAdmin: true,
				TermWidth: termW, TermHeight: termH,
			}
			st.ChatHistory = []domain.Message{
				{Author: "", Text: "Server started."},
				{Author: "Alice", Text: "hello world"},
				{Author: "Alice", Text: "/help"},
				{Author: "", Text: "Available commands: /help /plugin /theme /shader", IsReply: true},
			}
		},
	},

	// ── Playing ────────────────────────────────────────────────────────────
	{
		name:         "playing_game",
		playerID:     "alice",
		inActiveGame: true,
		gameName:     "testgame",
		setup: func(st *state.CentralState) {
			st.Lock()
			defer st.Unlock()
			st.Players["alice"] = &domain.Player{
				ID: "alice", Name: "Alice", IsAdmin: true,
				TermWidth: termW, TermHeight: termH,
			}
			st.Players["bob"] = &domain.Player{
				ID: "bob", Name: "Bob",
				TermWidth: termW, TermHeight: termH,
			}
			st.ActiveGame = &mockGame{}
			st.GameName = "testgame"
			st.GamePhase = domain.PhasePlaying
		},
	},
	{
		name:         "playing_splash",
		playerID:     "alice",
		inActiveGame: true,
		gameName:     "testgame",
		setup: func(st *state.CentralState) {
			st.Lock()
			defer st.Unlock()
			st.Players["alice"] = &domain.Player{
				ID: "alice", Name: "Alice", IsAdmin: true,
				TermWidth: termW, TermHeight: termH,
			}
			st.ActiveGame = &mockGame{}
			st.GameName = "testgame"
			st.GamePhase = domain.PhaseSplash
		},
	},
}
