package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultThemeNotNil(t *testing.T) {
	th := Default()
	if th == nil {
		t.Fatal("DefaultTheme returned nil")
	}
	if th.Name != "norton" {
		t.Errorf("expected name 'norton', got %q", th.Name)
	}
}

func TestLayerAtDepth(t *testing.T) {
	th := Default()
	tests := []struct {
		depth int
		want  *Layer
	}{
		{0, &th.Primary},
		{-1, &th.Primary},
		{1, &th.Secondary},
		{2, &th.Tertiary},
		{3, &th.Secondary},
		{4, &th.Tertiary},
		{5, &th.Secondary},
	}
	for _, tt := range tests {
		got := th.LayerAt(tt.depth)
		if got != tt.want {
			t.Errorf("LayerAt(%d): wrong layer", tt.depth)
		}
	}
}

func TestWarningLayer(t *testing.T) {
	th := Default()
	w := th.WarningLayer()
	if w != &th.Warning {
		t.Error("WarningLayer should return Warning layer")
	}
}

func TestBorderDefaults(t *testing.T) {
	th := Default()
	layer := th.LayerAt(0)
	if layer.OTL() != "╔" {
		t.Errorf("expected double-line TL, got %q", layer.OTL())
	}
	if layer.OTR() != "╗" {
		t.Errorf("expected double-line TR, got %q", layer.OTR())
	}
	if layer.OBL() != "╚" {
		t.Errorf("expected double-line BL, got %q", layer.OBL())
	}
	if layer.OBR() != "╝" {
		t.Errorf("expected double-line BR, got %q", layer.OBR())
	}
	if layer.OH() != "═" {
		t.Errorf("expected double-line H, got %q", layer.OH())
	}
	if layer.OV() != "║" {
		t.Errorf("expected double-line V, got %q", layer.OV())
	}
	if layer.IH() != "─" {
		t.Errorf("expected single-line inner H, got %q", layer.IH())
	}
	if layer.IV() != "│" {
		t.Errorf("expected single-line inner V, got %q", layer.IV())
	}
	if layer.XL() != "╟" {
		t.Errorf("expected intersection XL, got %q", layer.XL())
	}
	if layer.XR() != "╢" {
		t.Errorf("expected intersection XR, got %q", layer.XR())
	}
	if layer.XT() != "╤" {
		t.Errorf("expected intersection XT, got %q", layer.XT())
	}
	if layer.XB() != "╧" {
		t.Errorf("expected intersection XB, got %q", layer.XB())
	}
	if layer.XX() != "┼" {
		t.Errorf("expected intersection XX, got %q", layer.XX())
	}
	if layer.Sep() != "│" {
		t.Errorf("expected bar separator, got %q", layer.Sep())
	}
}

func TestBorderCustomValues(t *testing.T) {
	th := Default()
	th.Primary.OuterTL = "+"
	th.Primary.InnerH = "~"
	layer := th.LayerAt(0)
	if layer.OTL() != "+" {
		t.Errorf("expected custom TL '+', got %q", layer.OTL())
	}
	if layer.IH() != "~" {
		t.Errorf("expected custom IH '~', got %q", layer.IH())
	}
}

func TestPaletteColorAccessors(t *testing.T) {
	p := &Palette{
		Bg: "#112233", Fg: "#445566",
		Accent: "#778899", HighlightBg: "#aabbcc",
		HighlightFg: "#ddeeff", ActiveBg: "#001122",
		ActiveFg: "#334455", InputBg: "#667788",
		InputFg: "#99aabb", DisabledFg: "#ccddee",
	}
	// Just verify they don't panic and return non-nil.
	accessors := []func(){
		func() { p.BgC() },
		func() { p.FgC() },
		func() { p.AccentC() },
		func() { p.HighlightBgC() },
		func() { p.HighlightFgC() },
		func() { p.ActiveBgC() },
		func() { p.ActiveFgC() },
		func() { p.InputBgC() },
		func() { p.InputFgC() },
		func() { p.DisabledFgC() },
	}
	for i, fn := range accessors {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("accessor %d panicked: %v", i, r)
				}
			}()
			fn()
		}()
	}
}

func TestPaletteColorFallbacks(t *testing.T) {
	p := &Palette{} // all empty — should use fallbacks
	// These should not panic.
	p.BgC()
	p.FgC()
	p.AccentC()
	p.InputBgC()
	p.DisabledFgC()
}

func TestPaletteStyleBuilders(t *testing.T) {
	p := &Palette{Bg: "#000000", Fg: "#ffffff", Accent: "#ff0000",
		HighlightBg: "#0000ff", HighlightFg: "#ffffff",
		ActiveBg: "#00ff00", ActiveFg: "#000000",
		InputBg: "#111111", InputFg: "#eeeeee",
		DisabledFg: "#888888"}

	// Verify style builders don't panic and produce non-empty renders.
	styles := []func() string{
		func() string { return p.BaseStyle().Render("x") },
		func() string { return p.AccentStyle().Render("x") },
		func() string { return p.HighlightStyle().Render("x") },
		func() string { return p.ActiveStyle().Render("x") },
		func() string { return p.DisabledStyle().Render("x") },
		func() string { return p.InputStyle().Render("x") },
	}
	for i, fn := range styles {
		s := fn()
		if len(s) == 0 {
			t.Errorf("style %d rendered empty", i)
		}
	}
}

func TestShadowStyle(t *testing.T) {
	th := Default()
	s := th.ShadowStyle().Render("x")
	if len(s) == 0 {
		t.Error("shadow style rendered empty")
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.json")
	os.WriteFile(path, []byte(`{
		"name": "custom",
		"primary": {"bg": "#111", "fg": "#eee"},
		"secondary": {"bg": "#222", "fg": "#ddd"},
		"tertiary": {"bg": "#333", "fg": "#ccc"},
		"warning": {"bg": "#f00", "fg": "#fff"}
	}`), 0o644)

	th, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if th.Name != "custom" {
		t.Fatalf("expected 'custom', got %q", th.Name)
	}
	if th.Primary.Bg != "#111" {
		t.Fatalf("expected primary bg '#111', got %q", th.Primary.Bg)
	}
}

func TestLoadInfersName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mytheme.json")
	os.WriteFile(path, []byte(`{"primary": {"bg": "#000"}}`), 0o644)

	th, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if th.Name != "mytheme" {
		t.Fatalf("expected inferred name 'mytheme', got %q", th.Name)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte(`{not json}`), 0o644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/theme.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadWithPerLayerBorders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bordered.json")
	os.WriteFile(path, []byte(`{
		"primary": {"bg": "#000", "outerTL": "┌", "outerH": "─"},
		"secondary": {"bg": "#111", "outerTL": "╔", "outerH": "═"}
	}`), 0o644)

	th, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if th.Primary.OTL() != "┌" {
		t.Fatalf("expected primary TL '┌', got %q", th.Primary.OTL())
	}
	if th.Primary.OH() != "─" {
		t.Fatalf("expected primary H '─', got %q", th.Primary.OH())
	}
	if th.Secondary.OTL() != "╔" {
		t.Fatalf("expected secondary TL '╔', got %q", th.Secondary.OTL())
	}
}

func TestListThemes(t *testing.T) {
	dir := t.TempDir()
	themesDir := filepath.Join(dir, "themes")
	os.MkdirAll(themesDir, 0o755)
	os.WriteFile(filepath.Join(themesDir, "dark.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(themesDir, "light.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(themesDir, "readme.txt"), []byte(""), 0o644)
	os.MkdirAll(filepath.Join(themesDir, "subdir"), 0o755)

	names := ListThemes(dir)
	if len(names) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(names), names)
	}
}

func TestListThemesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	names := ListThemes(dir)
	if names != nil {
		t.Fatalf("expected nil, got %v", names)
	}
}
