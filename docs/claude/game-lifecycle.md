# Game Lifecycle, State & Suspend/Resume

## Game Lifecycle
```
LOBBY (teams + chat) -> SPLASH -> PLAYING -> GAME OVER -> LOBBY
                                   |
                               SUSPENDED -> LOBBY (game still in memory)
                                   ^
                                RESUME (warm or cold)
```
1. **Lobby**: Players configure teams, chat. Admin loads game with `/game load <name>`.
2. **Load**: Framework snapshots teams for the game (lobby teams stay independent), loads saved state, calls `init(savedState)`. `teams()` returns game teams.
3. **Splash**: Shows game splash screen (custom or default with game name). Admin presses Enter to start, or auto-starts after 10s.
4. **Splash->Playing**: Framework calls `start()`. Game sets up its playing state.
5. **Reconnect**: If a player disconnects mid-game and reconnects with the same name, they rejoin the game automatically.
5. **Playing**: Normal game mode. Game calls `gameOver(results, state)` when done.
4. **Game Over**: Framework renders ranked results screen. All players press Enter or 15s auto-transition.
5. Back to **Lobby** -- game unloaded, teams preserved for next round.
6. **Suspend** (optional): Admin runs `/game suspend [saveName]`. Framework calls `Game.suspend()` to get session state, persists it to `dist/state/saves/<gameName>/<saveName>.json`, transitions to lobby. Runtime stays alive for warm resume.
7. **Resume**: Admin runs `/game resume <gameName/saveName>` or uses File -> Resume Game menu. **Warm resume** (runtime alive): calls `Game.resume(nil)`, goes straight to Playing. **Cold resume** (server restarted): loads game fresh, calls `init(globalState)` + `start()` + `resume(sessionState)`, skips splash.

Late joiners see the lobby and can chat but don't join the active game. Lobby teams are independent from game teams -- players can freely organize for the next round while a game is running.

## Suspend/Resume

Games opt in to suspend/resume by setting `canSuspend: true` on the `Game` object. Suspend saves are independent of global game state (high scores via `gameOver(results, state)`) -- you can have multiple suspended sessions of the same game.

**JS hooks** (all optional, require `canSuspend: true`):
- `suspend()` -- called on `/game suspend`. Returns session state to persist. Game should pause internal logic.
- `resume(sessionState)` -- called on resume. `sessionState` is `null` for warm resume (runtime still alive), or the saved state for cold resume.

**Save files**: `dist/state/saves/<gameName>/<saveName>.json` -- contains team snapshot, disconnected player map, and game session state. Deleted after successful resume.

**Commands**:
- `/game suspend [saveName]` -- admin only. Auto-generates timestamp name if omitted.
- `/game resume <gameName/saveName>` -- admin only. Tab-completes against saved sessions. No args lists available saves.
- File -> Resume Game menu -- shows saves in a dialog with team count validation.

## Teams

Players manage teams in the lobby panel (right side, fixed 32 chars). New players start **unassigned** (shown under "Unassigned" at the top of the team list). Tab switches focus between chat and team panel. Navigation in team panel:
- **Down** from unassigned -> join first team (or create one if none exist)
- **Down** from a team -> move to team below
- **Down** from last team -> create new "Team \<your name\>" (blocked if you're the sole member, to avoid drop/recreate churn)
- **Up** from a team -> move to team above
- **Up** from first team -> become unassigned
- **Enter** (first player in team) -> rename team
- **Left/Right** (first player in team) -> cycle team color

New teams default to "Team \<creator name\>" and the first unused palette color. Games can declare `teamRange: {min, max}` to enforce valid team counts. Games access teams via the `teams()` global, which returns `[{name, color, players: [{id, name}, ...]}, ...]`. Game teams are a snapshot taken at load time -- lobby teams remain editable during a game. Unassigned players are excluded from the game snapshot.

## Game State (`Game.state`)

All mutable game data must live on `Game.state`. The framework reads this property for:
- **Suspend/resume:** `Game.state` is serialized to JSON on suspend and restored via `SetState()` on cold resume. No special suspend/resume hooks needed -- the framework handles it.
- **Client-side state replication:** (future) enhanced clients receive state deltas and render locally.

Games still persist cross-session data (high scores) via `gameOver(results, persistState)`. The `persistState` argument saves to `dist/state/<gamename>.json` and is received in `init(savedState)` on the next load. `Game.state` is session-scoped -- it lives only during gameplay.

## Central Clock (`internal/domain/clock.go`)

The framework provides a central `Clock` interface (`Now() time.Time`) used for all time-related operations. Games access it via the `now()` JS global (epoch milliseconds). In tests, inject a `MockClock` to control time. `Update(dt)` receives the real elapsed seconds between ticks.

## Game Over

Games call `gameOver(results, state)` where `results` is an array of `{ name, result }` in ranked order and `state` is an optional object to persist for the next run. The framework renders the game-over screen -- games don't need to provide their own. `name` is the display name (player or team). `result` is a freeform string (e.g. `"4200 pts"`, `"1st"`, `"DNF"`). Both arguments are optional. State is received via `config.savedState` in `init()` on the next load.
