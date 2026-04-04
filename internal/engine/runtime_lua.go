package engine

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"

	"null-space/internal/domain"
	"null-space/internal/render"
)

// LuaRuntime wraps a gopher-lua LState and implements domain.Game + ScriptRuntime.
type LuaRuntime struct {
	mu      sync.Mutex
	L       *lua.LState
	baseDir string
	dataDir string
	clock   domain.Clock

	commands    []domain.Command
	cachedTeams []map[string]any
	logFn       func(string)
	chatCh      chan domain.Message

	SourceFiles []SourceFile

	// game-over state set by Lua gameOver() call
	gameOverPending bool
	gameOverResults []domain.GameResult
	gameOverState   any

	// properties read from Game table
	gameNameProp  string
	teamRangeProp domain.TeamRange

	menus        []domain.MenuDef
	showDialogFn func(playerID string, d domain.DialogRequest)
}

// LoadLuaGame loads and executes a Lua game file and returns a ScriptRuntime.
// Init() is NOT called here — the server calls it after loading teams.
func LoadLuaGame(path string, logFn func(string), chatCh chan domain.Message, clock domain.Clock, dataDir string) (domain.Game, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lua game file: %w", err)
	}

	r := &LuaRuntime{
		L:       newSandboxedLState(),
		baseDir: filepath.Dir(path),
		dataDir: dataDir,
		clock:   clock,
		logFn:   logFn,
		chatCh:  chatCh,
	}

	r.SourceFiles = append(r.SourceFiles, SourceFile{
		Name:    filepath.Base(path),
		Content: string(src),
	})

	r.registerLuaGlobals()

	if err := r.L.DoString(string(src)); err != nil {
		return nil, fmt.Errorf("execute lua game script: %w", err)
	}

	if err := r.extractGameTable(); err != nil {
		return nil, fmt.Errorf("extract Game table: %w", err)
	}

	return r, nil
}

// newSandboxedLState creates a Lua VM with only safe standard libraries open.
func newSandboxedLState() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)
	return L
}

// watchdogLua sets a timeout on the Lua VM for a single call.
// Returns a cancel func that must be deferred.
func watchdogLua(L *lua.LState, method string) func() {
	ctx, cancel := context.WithTimeout(context.Background(), JSCallTimeout)
	L.SetContext(ctx)
	return func() {
		cancel()
		L.SetContext(context.Background())
		// If the VM was interrupted by timeout, clear the interrupt so future calls work.
		if err := ctx.Err(); err != nil {
			slog.Error("Lua call timed out", "method", method, "timeout", JSCallTimeout)
		}
	}
}

func (r *LuaRuntime) registerLuaGlobals() {
	L := r.L

	// Pixel attribute constants
	L.SetGlobal("ATTR_NONE", lua.LNumber(render.AttrNone))
	L.SetGlobal("ATTR_BOLD", lua.LNumber(render.AttrBold))
	L.SetGlobal("ATTR_FAINT", lua.LNumber(render.AttrFaint))
	L.SetGlobal("ATTR_ITALIC", lua.LNumber(render.AttrItalic))
	L.SetGlobal("ATTR_UNDERLINE", lua.LNumber(render.AttrUnderline))
	L.SetGlobal("ATTR_REVERSE", lua.LNumber(render.AttrReverse))

	L.SetGlobal("PUA_START", lua.LNumber(render.PUAStart))
	L.SetGlobal("PUA_END", lua.LNumber(render.PUAEnd))

	L.SetGlobal("log", L.NewFunction(func(L *lua.LState) int {
		msg := L.CheckString(1)
		if r.logFn != nil {
			r.logFn(msg)
		}
		return 0
	}))

	L.SetGlobal("chat", L.NewFunction(func(L *lua.LState) int {
		msg := L.CheckString(1)
		if r.chatCh != nil {
			select {
			case r.chatCh <- domain.Message{Text: msg}:
			default:
				slog.Warn("Lua chat channel full, dropping message", "text", msg)
			}
		}
		return 0
	}))

	L.SetGlobal("chatPlayer", L.NewFunction(func(L *lua.LState) int {
		playerID := L.CheckString(1)
		msg := L.CheckString(2)
		if r.chatCh != nil {
			select {
			case r.chatCh <- domain.Message{Text: msg, IsPrivate: true, ToID: playerID}:
			default:
				slog.Warn("Lua chatPlayer channel full, dropping message")
			}
		}
		return 0
	}))

	L.SetGlobal("teams", L.NewFunction(func(L *lua.LState) int {
		tbl := goSliceToLua(L, r.cachedTeams)
		L.Push(tbl)
		return 1
	}))

	L.SetGlobal("now", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(r.clock.Now().UnixMilli()))
		return 1
	}))

	L.SetGlobal("figlet", L.NewFunction(func(L *lua.LState) int {
		text := L.CheckString(1)
		font := L.OptString(2, "")
		L.Push(lua.LString(Figlet(text, font)))
		return 1
	}))

	L.SetGlobal("gameOver", L.NewFunction(func(L *lua.LState) int {
		r.gameOverPending = true
		r.gameOverResults = nil
		r.gameOverState = nil

		// First arg: results array [{name, result}, ...]
		if L.GetTop() >= 1 {
			if tbl, ok := L.Get(1).(*lua.LTable); ok {
				tbl.ForEach(func(_, v lua.LValue) {
					if item, ok := v.(*lua.LTable); ok {
						entry := domain.GameResult{}
						if n := item.RawGetString("name"); n != lua.LNil {
							entry.Name = n.String()
						}
						if res := item.RawGetString("result"); res != lua.LNil {
							entry.Result = res.String()
						}
						r.gameOverResults = append(r.gameOverResults, entry)
					}
				})
			}
		}

		// Second arg: state to persist
		if L.GetTop() >= 2 {
			r.gameOverState = luaToGo(L.Get(2))
		}

		return 0
	}))

	L.SetGlobal("registerCommand", L.NewFunction(func(L *lua.LState) int {
		spec := L.CheckTable(1)
		name := luaTableString(spec, "name")
		desc := luaTableString(spec, "description")
		adminOnly := luaTableBool(spec, "adminOnly")
		firstArgIsPlayer := luaTableBool(spec, "firstArgIsPlayer")
		handlerFn, _ := spec.RawGetString("handler").(*lua.LFunction)

		if name == "" || handlerFn == nil {
			slog.Warn("Lua registerCommand: name and handler are required")
			return 0
		}

		capturedFn := handlerFn
		cmd := domain.Command{
			Name:             name,
			Description:      desc,
			AdminOnly:        adminOnly,
			FirstArgIsPlayer: firstArgIsPlayer,
			Handler: func(ctx domain.CommandContext, args []string) {
				r.mu.Lock()
				defer r.mu.Unlock()
				argsTbl := r.L.NewTable()
				for i, a := range args {
					r.L.RawSetInt(argsTbl, i+1, lua.LString(a))
				}
				if err := r.L.CallByParam(lua.P{Fn: capturedFn, NRet: 0, Protect: true},
					lua.LString(ctx.PlayerID),
					lua.LBool(ctx.IsAdmin),
					argsTbl,
				); err != nil {
					slog.Error("Lua command handler error", "name", name, "error", err)
					ctx.Reply(fmt.Sprintf("Command error: %v", err))
				}
			},
		}
		r.commands = append(r.commands, cmd)
		return 0
	}))

	L.SetGlobal("addMenu", L.NewFunction(func(L *lua.LState) int {
		label := L.CheckString(1)
		if label == "" {
			return 0
		}
		var items []domain.MenuItemDef
		if itemsTbl, ok := L.Get(2).(*lua.LTable); ok {
			itemsTbl.ForEach(func(_, v lua.LValue) {
				if item, ok := v.(*lua.LTable); ok {
					itemLabel := luaTableString(item, "label")
					disabled := luaTableBool(item, "disabled")
					var goHandler func(string)
					if fn, ok := item.RawGetString("onClick").(*lua.LFunction); ok {
						capturedFn := fn
						goHandler = func(playerID string) {
							r.mu.Lock()
							defer r.mu.Unlock()
							if err := r.L.CallByParam(lua.P{Fn: capturedFn, NRet: 0, Protect: true},
								lua.LString(playerID),
							); err != nil {
								slog.Error("Lua menu handler error", "error", err)
							}
						}
					}
					items = append(items, domain.MenuItemDef{
						Label:    itemLabel,
						Disabled: disabled,
						Handler:  goHandler,
					})
				}
			})
		}
		r.menus = append(r.menus, domain.MenuDef{Label: label, Items: items})
		return 0
	}))

	L.SetGlobal("messageBox", L.NewFunction(func(L *lua.LState) int {
		playerID := L.CheckString(1)
		opts := L.CheckTable(2)

		title := luaTableString(opts, "title")
		body := luaTableString(opts, "message")

		var buttons []string
		if btnTbl, ok := opts.RawGetString("buttons").(*lua.LTable); ok {
			btnTbl.ForEach(func(_, v lua.LValue) {
				buttons = append(buttons, v.String())
			})
		}

		var onClose func(string)
		if fn, ok := opts.RawGetString("onClose").(*lua.LFunction); ok {
			capturedFn := fn
			onClose = func(button string) {
				r.mu.Lock()
				defer r.mu.Unlock()
				_ = r.L.CallByParam(lua.P{Fn: capturedFn, NRet: 0, Protect: true},
					lua.LString(button),
				)
			}
		}

		d := domain.DialogRequest{
			Title:   title,
			Body:    body,
			Buttons: buttons,
			OnClose: onClose,
		}
		if r.showDialogFn != nil {
			go r.showDialogFn(playerID, d)
		}
		return 0
	}))

	// include() — load another .lua file relative to the game directory
	included := map[string]bool{}
	L.SetGlobal("include", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
			L.RaiseError("include: path traversal not allowed")
			return 0
		}
		if !strings.HasSuffix(name, ".lua") {
			name += ".lua"
		}
		absPath := filepath.Join(r.baseDir, name)
		if included[absPath] {
			return 0
		}
		included[absPath] = true
		src, err := os.ReadFile(absPath)
		if err != nil {
			L.RaiseError("include: cannot read %s: %v", name, err)
			return 0
		}
		r.SourceFiles = append(r.SourceFiles, SourceFile{Name: name, Content: string(src)})
		if err := L.DoString(string(src)); err != nil {
			L.RaiseError("include: error in %s: %v", name, err)
		}
		return 0
	}))
}

func (r *LuaRuntime) extractGameTable() error {
	gameLV := r.L.GetGlobal("Game")
	if gameLV == lua.LNil {
		return fmt.Errorf("script must define a global 'Game' table")
	}
	gameTbl, ok := gameLV.(*lua.LTable)
	if !ok {
		return fmt.Errorf("'Game' must be a table")
	}

	// init is mandatory
	if _, ok := gameTbl.RawGetString("init").(*lua.LFunction); !ok {
		return fmt.Errorf("Game must define an init(savedState) function")
	}

	// Read scalar properties
	r.gameNameProp = luaTableString(gameTbl, "gameName")

	if trTbl, ok := gameTbl.RawGetString("teamRange").(*lua.LTable); ok {
		if min := trTbl.RawGetString("min"); min != lua.LNil {
			r.teamRangeProp.Min = int(lua.LVAsNumber(min))
		}
		if max := trTbl.RawGetString("max"); max != lua.LNil {
			r.teamRangeProp.Max = int(lua.LVAsNumber(max))
		}
	}

	return nil
}

// --- domain.Game implementation ---

func (r *LuaRuntime) Init(savedState any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	defer watchdogLua(r.L, "init")()
	fn := r.luaGameFn("init")
	if fn == nil {
		return
	}
	arg := goToLua(r.L, savedState)
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, arg); err != nil {
		slog.Error("Lua init error", "error", err)
	}

	// Re-read splashScreen — init() may have set it dynamically.
	// (Lua games don't have a splash screen hook; they use the default.)
}

func (r *LuaRuntime) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("start")
	if fn == nil {
		return
	}
	defer watchdogLua(r.L, "start")()
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}); err != nil {
		slog.Error("Lua start error", "error", err)
	}
}

func (r *LuaRuntime) Update(dt float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("update")
	if fn == nil {
		return
	}
	defer watchdogLua(r.L, "update")()
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, lua.LNumber(dt)); err != nil {
		slog.Error("Lua update error", "error", err)
	}
}

func (r *LuaRuntime) OnPlayerLeave(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("onPlayerLeave")
	if fn == nil {
		return
	}
	defer watchdogLua(r.L, "onPlayerLeave")()
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, lua.LString(playerID)); err != nil {
		slog.Error("Lua onPlayerLeave error", "error", err)
	}
}

func (r *LuaRuntime) OnInput(playerID, key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("onInput")
	if fn == nil {
		return
	}
	defer watchdogLua(r.L, "onInput")()
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true},
		lua.LString(playerID), lua.LString(key),
	); err != nil {
		slog.Error("Lua onInput error", "error", err)
	}
}

func (r *LuaRuntime) Render(buf *render.ImageBuffer, playerID string, x, y, width, height int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("render")
	if fn == nil {
		return
	}
	defer watchdogLua(r.L, "render")()
	luaBuf := r.newLuaImageBuffer(buf, x, y, width, height)
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true},
		luaBuf,
		lua.LString(playerID),
		lua.LNumber(x), lua.LNumber(y),
		lua.LNumber(width), lua.LNumber(height),
	); err != nil {
		slog.Error("Lua render error", "error", err)
	}
}

func (r *LuaRuntime) RenderSplash(buf *render.ImageBuffer, playerID string, x, y, width, height int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("renderSplash")
	if fn == nil {
		return false
	}
	defer watchdogLua(r.L, "renderSplash")()
	luaBuf := r.newLuaImageBuffer(buf, x, y, width, height)
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true},
		luaBuf,
		lua.LString(playerID),
		lua.LNumber(x), lua.LNumber(y),
		lua.LNumber(width), lua.LNumber(height),
	); err != nil {
		slog.Error("Lua renderSplash error", "error", err)
		return false
	}
	return true
}

func (r *LuaRuntime) RenderGameOver(buf *render.ImageBuffer, playerID string, x, y, width, height int, results []domain.GameResult) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("renderGameOver")
	if fn == nil {
		return false
	}
	defer watchdogLua(r.L, "renderGameOver")()
	luaBuf := r.newLuaImageBuffer(buf, x, y, width, height)
	resultsTbl := r.L.NewTable()
	for i, res := range results {
		entry := r.L.NewTable()
		r.L.SetField(entry, "name", lua.LString(res.Name))
		r.L.SetField(entry, "result", lua.LString(res.Result))
		r.L.RawSetInt(resultsTbl, i+1, entry)
	}
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true},
		luaBuf,
		lua.LString(playerID),
		lua.LNumber(x), lua.LNumber(y),
		lua.LNumber(width), lua.LNumber(height),
		resultsTbl,
	); err != nil {
		slog.Error("Lua renderGameOver error", "error", err)
		return false
	}
	return true
}

func (r *LuaRuntime) Layout(_ string, _, _ int) *domain.WidgetNode {
	return nil // NC widget tree not supported in Lua
}

func (r *LuaRuntime) StatusBar(playerID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("statusBar")
	if fn == nil {
		return ""
	}
	defer watchdogLua(r.L, "statusBar")()
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, lua.LString(playerID)); err != nil {
		slog.Error("Lua statusBar error", "error", err)
		return ""
	}
	ret := r.L.Get(-1)
	r.L.Pop(1)
	return ret.String()
}

func (r *LuaRuntime) CommandBar(playerID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn := r.luaGameFn("commandBar")
	if fn == nil {
		return ""
	}
	defer watchdogLua(r.L, "commandBar")()
	if err := r.L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, lua.LString(playerID)); err != nil {
		slog.Error("Lua commandBar error", "error", err)
		return ""
	}
	ret := r.L.Get(-1)
	r.L.Pop(1)
	return ret.String()
}

func (r *LuaRuntime) Commands() []domain.Command {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]domain.Command, len(r.commands))
	copy(result, r.commands)
	return result
}

func (r *LuaRuntime) Menus() []domain.MenuDef {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.menus
}

func (r *LuaRuntime) GameName() string     { return r.gameNameProp }
func (r *LuaRuntime) TeamRange() domain.TeamRange { return r.teamRangeProp }
func (r *LuaRuntime) CharMap() *render.CharMapDef { return nil }
func (r *LuaRuntime) HasCanvasMode() bool  { return false }

func (r *LuaRuntime) RenderCanvas(_ string, _, _ int) []byte       { return nil }
func (r *LuaRuntime) RenderCanvasImage(_ string, _, _ int) *image.RGBA { return nil }

func (r *LuaRuntime) Unload() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.L.Close()
}

func (r *LuaRuntime) State() any {
	r.mu.Lock()
	defer r.mu.Unlock()
	gameLV := r.L.GetGlobal("Game")
	gameTbl, ok := gameLV.(*lua.LTable)
	if !ok {
		return nil
	}
	sv := gameTbl.RawGetString("state")
	if sv == lua.LNil {
		return nil
	}
	return luaToGo(sv)
}

func (r *LuaRuntime) SetState(state any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	gameLV := r.L.GetGlobal("Game")
	gameTbl, ok := gameLV.(*lua.LTable)
	if !ok {
		return
	}
	r.L.SetField(gameTbl, "state", goToLua(r.L, state))
}

func (r *LuaRuntime) GameSource() []domain.GameSourceFile {
	result := make([]domain.GameSourceFile, len(r.SourceFiles))
	for i, sf := range r.SourceFiles {
		result[i] = domain.GameSourceFile{Name: sf.Name, Content: sf.Content}
	}
	return result
}

// --- ScriptRuntime extension methods ---

func (r *LuaRuntime) SetTeamsCache(teams []map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cachedTeams = teams
}

func (r *LuaRuntime) SetShowDialogFn(fn func(playerID string, d domain.DialogRequest)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.showDialogFn = fn
}

func (r *LuaRuntime) IsGameOverPending() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.gameOverPending {
		return false
	}
	r.gameOverPending = false
	return true
}

func (r *LuaRuntime) GameOverResults() []domain.GameResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gameOverResults
}

func (r *LuaRuntime) GameOverStateExport() any {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gameOverState
}

func (r *LuaRuntime) CloseChatCh() {
	if r.chatCh != nil {
		close(r.chatCh)
	}
}

// --- helpers ---

// luaGameFn retrieves a function from the Game table by name.
// Must be called with r.mu held.
func (r *LuaRuntime) luaGameFn(name string) *lua.LFunction {
	gameLV := r.L.GetGlobal("Game")
	gameTbl, ok := gameLV.(*lua.LTable)
	if !ok {
		return nil
	}
	fn, ok := gameTbl.RawGetString(name).(*lua.LFunction)
	if !ok {
		return nil
	}
	return fn
}

// newLuaImageBuffer creates a Lua table wrapping an ImageBuffer region.
// Lua games call buf:setChar(x, y, ch, fg, bg, attr), buf:writeString(...), buf:fill(...).
func (r *LuaRuntime) newLuaImageBuffer(buf *render.ImageBuffer, ox, oy, w, h int) *lua.LTable {
	L := r.L
	tbl := L.NewTable()
	L.SetField(tbl, "width", lua.LNumber(w))
	L.SetField(tbl, "height", lua.LNumber(h))

	L.SetField(tbl, "setChar", L.NewFunction(func(L *lua.LState) int {
		x := int(L.CheckNumber(2)) // 1 = self (tbl), 2 = x, ...
		y := int(L.CheckNumber(3))
		ch := L.CheckString(4)
		fg := parseLuaColor(L.Get(5))
		bg := parseLuaColor(L.Get(6))
		attr := parseLuaAttr(L.Get(7))
		if len(ch) > 0 {
			buf.SetChar(ox+x, oy+y, []rune(ch)[0], fg, bg, attr)
		}
		return 0
	}))

	L.SetField(tbl, "writeString", L.NewFunction(func(L *lua.LState) int {
		x := int(L.CheckNumber(2))
		y := int(L.CheckNumber(3))
		text := L.CheckString(4)
		fg := parseLuaColor(L.Get(5))
		bg := parseLuaColor(L.Get(6))
		attr := parseLuaAttr(L.Get(7))
		buf.WriteString(ox+x, oy+y, text, fg, bg, attr)
		return 0
	}))

	L.SetField(tbl, "fill", L.NewFunction(func(L *lua.LState) int {
		x := int(L.CheckNumber(2))
		y := int(L.CheckNumber(3))
		fw := int(L.CheckNumber(4))
		fh := int(L.CheckNumber(5))
		ch := L.CheckString(6)
		fg := parseLuaColor(L.Get(7))
		bg := parseLuaColor(L.Get(8))
		attr := parseLuaAttr(L.Get(9))
		fillCh := ' '
		if len(ch) > 0 {
			fillCh = []rune(ch)[0]
		}
		buf.Fill(ox+x, oy+y, fw, fh, fillCh, fg, bg, attr)
		return 0
	}))

	return tbl
}

// luaTableString reads a string field from a Lua table, returning "" if absent.
func luaTableString(tbl *lua.LTable, key string) string {
	v := tbl.RawGetString(key)
	if v == lua.LNil {
		return ""
	}
	return v.String()
}

// luaTableBool reads a bool field from a Lua table, returning false if absent.
func luaTableBool(tbl *lua.LTable, key string) bool {
	v := tbl.RawGetString(key)
	if b, ok := v.(lua.LBool); ok {
		return bool(b)
	}
	return false
}

// goToLua converts a Go value to a Lua value.
func goToLua(L *lua.LState, v any) lua.LValue {
	if v == nil {
		return lua.LNil
	}
	switch val := v.(type) {
	case bool:
		return lua.LBool(val)
	case int:
		return lua.LNumber(float64(val))
	case int64:
		return lua.LNumber(float64(val))
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case []any:
		tbl := L.NewTable()
		for i, item := range val {
			L.RawSetInt(tbl, i+1, goToLua(L, item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, item := range val {
			L.SetField(tbl, k, goToLua(L, item))
		}
		return tbl
	default:
		return lua.LNil
	}
}

// goSliceToLua converts []map[string]any (teams cache) to a Lua table.
func goSliceToLua(L *lua.LState, slice []map[string]any) *lua.LTable {
	tbl := L.NewTable()
	for i, m := range slice {
		L.RawSetInt(tbl, i+1, goToLua(L, m))
	}
	return tbl
}

// luaToGo converts a Lua value to a Go value suitable for JSON serialisation.
func luaToGo(v lua.LValue) any {
	switch val := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(val)
	case lua.LNumber:
		return float64(val)
	case lua.LString:
		return string(val)
	case *lua.LTable:
		return luaTableToGo(val)
	default:
		return nil
	}
}

// luaTableToGo converts a Lua table: array-like → []any, otherwise → map[string]any.
func luaTableToGo(tbl *lua.LTable) any {
	n := tbl.MaxN()
	if n > 0 {
		result := make([]any, n)
		for i := 1; i <= n; i++ {
			result[i-1] = luaToGo(tbl.RawGetInt(i))
		}
		return result
	}
	result := make(map[string]any)
	tbl.ForEach(func(k, v lua.LValue) {
		if ks, ok := k.(lua.LString); ok {
			result[string(ks)] = luaToGo(v)
		}
	})
	return result
}

// parseLuaColor converts a Lua value to a color.Color.
// LNil → nil, "#RRGGBB" hex string → color.RGBA.
func parseLuaColor(v lua.LValue) color.Color {
	if v == lua.LNil {
		return nil
	}
	s := v.String()
	if len(s) == 7 && s[0] == '#' {
		r := hexByte(s[1], s[2])
		g := hexByte(s[3], s[4])
		b := hexByte(s[5], s[6])
		return color.RGBA{R: r, G: g, B: b, A: 255}
	}
	return nil
}

// parseLuaAttr converts a Lua value to a PixelAttr.
func parseLuaAttr(v lua.LValue) render.PixelAttr {
	if v == lua.LNil {
		return render.AttrNone
	}
	return render.PixelAttr(lua.LVAsNumber(v))
}
