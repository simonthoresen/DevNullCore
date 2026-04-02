package theme

import (
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
	l0 := th.LayerAt(0)
	l1 := th.LayerAt(1)
	l2 := th.LayerAt(2)
	l3 := th.LayerAt(3)
	l4 := th.LayerAt(4)

	if l0 != &th.Primary {
		t.Error("depth 0 should be Primary")
	}
	if l1 != &th.Secondary {
		t.Error("depth 1 should be Secondary")
	}
	if l2 != &th.Tertiary {
		t.Error("depth 2 should be Tertiary")
	}
	if l3 != &th.Secondary {
		t.Error("depth 3 should be Secondary (alternating)")
	}
	if l4 != &th.Tertiary {
		t.Error("depth 4 should be Tertiary (alternating)")
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
	if layer.IH() != "─" {
		t.Errorf("expected single-line inner H, got %q", layer.IH())
	}
	if layer.XL() != "╟" {
		t.Errorf("expected double-single intersection, got %q", layer.XL())
	}
}
