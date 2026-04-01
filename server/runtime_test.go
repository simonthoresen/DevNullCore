package server

import (
	"os"
	"path/filepath"
	"testing"

	"null-space/common"
)

func TestIncludeSingleFile(t *testing.T) {
	dir := t.TempDir()

	// Helper file defines a function.
	os.WriteFile(filepath.Join(dir, "helper.js"), []byte(`
		function greet(name) { return "Hello, " + name; }
	`), 0o644)

	// Main game file includes the helper and uses it.
	mainJS := filepath.Join(dir, "main.js")
	os.WriteFile(mainJS, []byte(`
		include("helper.js");
		var Game = {
			gameName: greet("World"),
			init: function(s) {}
		};
	`), 0o644)

	chatCh := make(chan common.Message, 8)
	game, err := LoadGame(mainJS, func(string) {}, chatCh)
	if err != nil {
		t.Fatalf("LoadGame: %v", err)
	}
	if game.GameName() != "Hello, World" {
		t.Errorf("got gameName=%q, want %q", game.GameName(), "Hello, World")
	}
}

func TestIncludeIdempotent(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "counter.js"), []byte(`
		var counter = (typeof counter !== 'undefined') ? counter + 1 : 1;
	`), 0o644)

	mainJS := filepath.Join(dir, "main.js")
	os.WriteFile(mainJS, []byte(`
		include("counter.js");
		include("counter.js");
		var Game = {
			gameName: "count-" + counter,
			init: function(s) {}
		};
	`), 0o644)

	chatCh := make(chan common.Message, 8)
	game, err := LoadGame(mainJS, func(string) {}, chatCh)
	if err != nil {
		t.Fatalf("LoadGame: %v", err)
	}
	// include is idempotent — counter should be 1, not 2
	if game.GameName() != "count-1" {
		t.Errorf("got gameName=%q, want %q", game.GameName(), "count-1")
	}
}

func TestNethackGameLoads(t *testing.T) {
	// Test that the actual nethack game loads through the framework.
	mainJS := filepath.Join("../dist/games/nethack", "main.js")
	if _, err := os.Stat(mainJS); err != nil {
		t.Skip("nethack game not found at", mainJS)
	}

	chatCh := make(chan common.Message, 64)
	game, err := LoadGame(mainJS, func(string) {}, chatCh)
	if err != nil {
		t.Fatalf("LoadGame nethack: %v", err)
	}
	if game.GameName() != "NetHack" {
		t.Errorf("got gameName=%q, want %q", game.GameName(), "NetHack")
	}
}

func TestIncludeRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()

	mainJS := filepath.Join(dir, "main.js")
	os.WriteFile(mainJS, []byte(`
		include("../etc/passwd");
		var Game = { init: function(s) {} };
	`), 0o644)

	chatCh := make(chan common.Message, 8)
	_, err := LoadGame(mainJS, func(string) {}, chatCh)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}
