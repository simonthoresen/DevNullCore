package engine

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"

	"null-space/internal/domain"
	"null-space/internal/network"
)

// LuaPlugin wraps a gopher-lua LState for a per-player plugin.
// The plugin exports a Plugin table with an onMessage(author, text, isSystem) hook.
type LuaPlugin struct {
	mu   sync.Mutex
	L    *lua.LState
	name string
}

// LoadLuaPlugin reads and executes a Lua plugin file, extracting Plugin.onMessage.
func LoadLuaPlugin(path string, clock domain.Clock) (*LuaPlugin, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lua plugin file: %w", err)
	}

	p := &LuaPlugin{
		L:    newSandboxedLState(),
		name: strings.TrimSuffix(filepath.Base(path), ".lua"),
	}

	p.L.SetGlobal("log", p.L.NewFunction(func(L *lua.LState) int {
		slog.Info("plugin log", "plugin", p.name, "msg", L.CheckString(1))
		return 0
	}))
	p.L.SetGlobal("now", p.L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(clock.Now().UnixMilli()))
		return 1
	}))

	if err := p.L.DoString(string(src)); err != nil {
		return nil, fmt.Errorf("execute lua plugin script: %w", err)
	}

	pluginLV := p.L.GetGlobal("Plugin")
	if pluginLV == lua.LNil {
		return nil, fmt.Errorf("lua plugin must export a global Plugin table")
	}
	if _, ok := pluginLV.(*lua.LTable); !ok {
		return nil, fmt.Errorf("Plugin must be a table")
	}
	pluginTbl := pluginLV.(*lua.LTable)
	if _, ok := pluginTbl.RawGetString("onMessage").(*lua.LFunction); !ok {
		return nil, fmt.Errorf("Plugin.onMessage is required and must be a function")
	}

	return p, nil
}

// OnMessage calls Plugin.onMessage(author, text, isSystem).
// Returns a non-empty string if the plugin wants to inject input.
func (p *LuaPlugin) OnMessage(author, text string, isSystem bool) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	pluginTbl, ok := p.L.GetGlobal("Plugin").(*lua.LTable)
	if !ok {
		return ""
	}
	fn, ok := pluginTbl.RawGetString("onMessage").(*lua.LFunction)
	if !ok {
		return ""
	}

	cancel := watchdogLua(p.L, "Plugin.onMessage")
	defer cancel()

	if err := p.L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true},
		lua.LString(author),
		lua.LString(text),
		lua.LBool(isSystem),
	); err != nil {
		slog.Error("lua plugin onMessage error", "plugin", p.name, "error", err)
		return ""
	}

	ret := p.L.Get(-1)
	p.L.Pop(1)
	if s, ok := ret.(lua.LString); ok {
		return string(s)
	}
	return ""
}

func (p *LuaPlugin) Name() string { return p.name }

func (p *LuaPlugin) Unload() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.L.Close()
}

// ResolvePluginPathLua resolves a Lua plugin name to a local file path.
func ResolvePluginPathLua(nameOrURL, dataDir string) (name, path string, err error) {
	if network.IsURL(nameOrURL) {
		cacheDir := filepath.Join(dataDir, "plugins", ".cache")
		local, localErr := network.DownloadToCache(nameOrURL, cacheDir)
		if localErr != nil {
			return "", "", fmt.Errorf("download plugin: %w", localErr)
		}
		return strings.TrimSuffix(filepath.Base(local), ".lua"), local, nil
	}
	return nameOrURL, filepath.Join(dataDir, "plugins", nameOrURL+".lua"), nil
}
