package engine

import "null-space/internal/domain"

// ScriptRuntime extends domain.Game with lifecycle hooks used by the server that
// sit outside the public Game interface (teams cache, game-over signalling, etc.).
type ScriptRuntime interface {
	domain.Game
	SetTeamsCache(teams []map[string]any)
	SetShowDialogFn(fn func(playerID string, d domain.DialogRequest))
	IsGameOverPending() bool
	GameOverResults() []domain.GameResult
	GameOverStateExport() any
	CloseChatCh()
}
