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
	"null-space/internal/render"
)

// LuaShader wraps a gopher-lua LState for a per-player post-processing shader.
type LuaShader struct {
	mu   sync.Mutex
	L    *lua.LState
	name string
}

// LoadLuaShader reads and executes a Lua shader file, extracting Shader.process.
func LoadLuaShader(path string, clock domain.Clock) (*LuaShader, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lua shader file: %w", err)
	}

	s := &LuaShader{
		L:    newSandboxedLState(),
		name: strings.TrimSuffix(filepath.Base(path), ".lua"),
	}

	s.L.SetGlobal("log", s.L.NewFunction(func(L *lua.LState) int {
		slog.Info("shader log", "shader", s.name, "msg", L.CheckString(1))
		return 0
	}))
	s.L.SetGlobal("now", s.L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(clock.Now().UnixMilli()))
		return 1
	}))

	// Pixel attribute constants
	s.L.SetGlobal("ATTR_NONE", lua.LNumber(render.AttrNone))
	s.L.SetGlobal("ATTR_BOLD", lua.LNumber(render.AttrBold))
	s.L.SetGlobal("ATTR_FAINT", lua.LNumber(render.AttrFaint))
	s.L.SetGlobal("ATTR_ITALIC", lua.LNumber(render.AttrItalic))
	s.L.SetGlobal("ATTR_UNDERLINE", lua.LNumber(render.AttrUnderline))
	s.L.SetGlobal("ATTR_REVERSE", lua.LNumber(render.AttrReverse))

	if err := s.L.DoString(string(src)); err != nil {
		return nil, fmt.Errorf("execute lua shader script: %w", err)
	}

	shaderLV := s.L.GetGlobal("Shader")
	if shaderLV == lua.LNil {
		return nil, fmt.Errorf("lua shader must export a global Shader table")
	}
	shaderTbl, ok := shaderLV.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("Shader must be a table")
	}
	if _, ok := shaderTbl.RawGetString("process").(*lua.LFunction); !ok {
		return nil, fmt.Errorf("Shader.process is required and must be a function")
	}

	// Call Shader.init() if defined
	if initFn, ok := shaderTbl.RawGetString("init").(*lua.LFunction); ok {
		if err := s.L.CallByParam(lua.P{Fn: initFn, NRet: 0, Protect: true}); err != nil {
			return nil, fmt.Errorf("lua shader init error: %w", err)
		}
	}

	return s, nil
}

// Process implements domain.Shader by calling Shader.process(buf, elapsed).
func (s *LuaShader) Process(buf *render.ImageBuffer, elapsed float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	shaderTbl, ok := s.L.GetGlobal("Shader").(*lua.LTable)
	if !ok {
		return
	}
	fn, ok := shaderTbl.RawGetString("process").(*lua.LFunction)
	if !ok {
		return
	}

	cancel := watchdogLua(s.L, "Shader.process")
	defer cancel()

	luaBuf := s.newLuaShaderBuf(buf)
	if err := s.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true},
		luaBuf,
		lua.LNumber(elapsed),
	); err != nil {
		slog.Error("lua shader process error", "shader", s.name, "error", err)
	}
}

// newLuaShaderBuf exposes the ImageBuffer to a Lua shader.
// Provides: buf.width, buf.height, buf:get(x,y), buf:set(x,y,ch,fg,bg,attr).
func (s *LuaShader) newLuaShaderBuf(buf *render.ImageBuffer) *lua.LTable {
	L := s.L
	tbl := L.NewTable()
	L.SetField(tbl, "width", lua.LNumber(buf.Width))
	L.SetField(tbl, "height", lua.LNumber(buf.Height))

	L.SetField(tbl, "get", L.NewFunction(func(L *lua.LState) int {
		x := int(L.CheckNumber(2))
		y := int(L.CheckNumber(3))
		if x < 0 || x >= buf.Width || y < 0 || y >= buf.Height {
			L.Push(lua.LNil)
			return 1
		}
		pixel := buf.Pixels[y*buf.Width+x]
		result := L.NewTable()
		L.SetField(result, "ch", lua.LString(string(pixel.Char)))
		L.SetField(result, "fg", lua.LString(ColorToHex(pixel.Fg)))
		L.SetField(result, "bg", lua.LString(ColorToHex(pixel.Bg)))
		L.SetField(result, "attr", lua.LNumber(pixel.Attr))
		L.Push(result)
		return 1
	}))

	L.SetField(tbl, "set", L.NewFunction(func(L *lua.LState) int {
		x := int(L.CheckNumber(2))
		y := int(L.CheckNumber(3))
		ch := L.CheckString(4)
		fg := parseLuaColor(L.Get(5))
		bg := parseLuaColor(L.Get(6))
		attr := parseLuaAttr(L.Get(7))
		if len(ch) > 0 {
			buf.SetChar(x, y, []rune(ch)[0], fg, bg, attr)
		}
		return 0
	}))

	return tbl
}

func (s *LuaShader) Name() string { return s.name }

func (s *LuaShader) Unload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	shaderTbl, ok := s.L.GetGlobal("Shader").(*lua.LTable)
	if ok {
		if fn, ok := shaderTbl.RawGetString("unload").(*lua.LFunction); ok {
			_ = s.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true})
		}
	}
	s.L.Close()
}

// ResolveShaderPathLua resolves a Lua shader name to a local file path.
func ResolveShaderPathLua(nameOrURL, dataDir string) (name, path string, err error) {
	if network.IsURL(nameOrURL) {
		cacheDir := filepath.Join(dataDir, "shaders", ".cache")
		local, localErr := network.DownloadToCache(nameOrURL, cacheDir)
		if localErr != nil {
			return "", "", fmt.Errorf("download shader: %w", localErr)
		}
		return strings.TrimSuffix(filepath.Base(local), ".lua"), local, nil
	}
	return nameOrURL, filepath.Join(dataDir, "shaders", nameOrURL+".lua"), nil
}
