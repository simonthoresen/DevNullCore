package domain

import (
	"testing"
)

func TestHashNil(t *testing.T) {
	var n *WidgetNode
	if h := n.Hash(); h != 0 {
		t.Fatalf("expected 0 for nil node, got %d", h)
	}
}

func TestHashGameviewIsZero(t *testing.T) {
	n := &WidgetNode{Type: "gameview"}
	if h := n.Hash(); h != 0 {
		t.Fatalf("expected 0 for gameview, got %d", h)
	}
}

func TestHashEmptyTypeIsZero(t *testing.T) {
	n := &WidgetNode{Type: ""}
	if h := n.Hash(); h != 0 {
		t.Fatalf("expected 0 for empty type (fallback to gameview), got %d", h)
	}
}

func TestHashInteractiveIsZero(t *testing.T) {
	n := &WidgetNode{Type: "button", Action: "click"}
	if h := n.Hash(); h != 0 {
		t.Fatalf("expected 0 for interactive node, got %d", h)
	}
}

func TestHashFocusableIsZero(t *testing.T) {
	n := &WidgetNode{Type: "gameview", IsFocusable: true}
	if h := n.Hash(); h != 0 {
		t.Fatalf("expected 0 for focusable node, got %d", h)
	}
}

func TestHashLabelIsDeterministic(t *testing.T) {
	n := &WidgetNode{Type: "label", Text: "hello"}
	h1 := n.Hash()
	h2 := n.Hash()
	if h1 == 0 {
		t.Fatal("expected non-zero hash for label")
	}
	if h1 != h2 {
		t.Fatal("expected deterministic hash")
	}
}

func TestHashDifferentContentDiffers(t *testing.T) {
	n1 := &WidgetNode{Type: "label", Text: "hello"}
	n2 := &WidgetNode{Type: "label", Text: "world"}
	if n1.Hash() == n2.Hash() {
		t.Fatal("expected different hashes for different content")
	}
}

func TestHashGameviewChildPropagatesZero(t *testing.T) {
	parent := &WidgetNode{
		Type: "hsplit",
		Children: []*WidgetNode{
			{Type: "label", Text: "ok"},
			{Type: "gameview"},
		},
	}
	if h := parent.Hash(); h != 0 {
		t.Fatalf("expected 0 to propagate from gameview child, got %d", h)
	}
}

func TestHashPureStaticTreeIsNonZero(t *testing.T) {
	tree := &WidgetNode{
		Type: "vsplit",
		Children: []*WidgetNode{
			{Type: "label", Text: "Status"},
			{Type: "table", Rows: [][]string{{"A", "1"}, {"B", "2"}}},
		},
	}
	if h := tree.Hash(); h == 0 {
		t.Fatal("expected non-zero hash for fully static tree")
	}
}

func TestHashTableRowsAffectHash(t *testing.T) {
	n1 := &WidgetNode{Type: "table", Rows: [][]string{{"A", "1"}}}
	n2 := &WidgetNode{Type: "table", Rows: [][]string{{"A", "2"}}}
	if n1.Hash() == n2.Hash() {
		t.Fatal("expected different hashes for different table rows")
	}
}

func TestHashDimensionsAffectHash(t *testing.T) {
	n1 := &WidgetNode{Type: "panel", Width: 10, Height: 20}
	n2 := &WidgetNode{Type: "panel", Width: 10, Height: 30}
	if n1.Hash() == n2.Hash() {
		t.Fatal("expected different hashes for different dimensions")
	}
}
