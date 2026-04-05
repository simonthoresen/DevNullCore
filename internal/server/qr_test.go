package server

import (
	"testing"
)

const wantInvite = `$env:NS='ABCDEFGH';irm https://raw.githubusercontent.com/simonthoresen/null-space/main/join.ps1|iex`

const wantQR = "" +
	"                         \n" +
	"                         \n" +
	"  ▛▀▀▌ ▚▟▙▝▞▟▟▗▐▝▞ ▛▀▀▌  \n" +
	"  ▌█▌▌▝▖▚▖▙█▗▘ ▚▝▞▘▌█▌▌  \n" +
	"  ▌▀▘▌▟█▛▙▞▚▝▝▐▐▙▙▖▌▀▘▌  \n" +
	"  ▀▀▀▘▚▚▚▙▙▘▙▘▘▙▚▙▌▀▀▀▘  \n" +
	"  ▚▐▞▘▛▙▌▙▘▖▟▝▗▚█▝▐▞ ▗▖  \n" +
	"  ▜▙▄▀▖▜▄▚▞▐█▄▄▙▞▟ ▀▝▌   \n" +
	"  ▜▖▐▀▞▞▘▌▜▖▐▖▞▖▐▐█▀ ▐▘  \n" +
	"  ▜█▚▚ ▟▀▝▟▚▚▟▟▄▜ ▗▖▟▀▌  \n" +
	"   ▗▀▀▝▄█▀▀▝▝▚▙▖▟ ▜▝  ▖  \n" +
	"  ▛▝ ▚▙ ▄▚▚▌▞▖▖▌▙▙▄▘▐▐▌  \n" +
	"  ▌▝▀▚▜ ▛▚▐▛▖ ▟▌▄▝▐▛ ▙▌  \n" +
	"  ▞▌▀▘▝▛▙ ▌▐▚▖▄▄ ▛ ▀▘▖▖  \n" +
	"  ▘▘▖▀▙▗▝█▜▄▀ ▘▐▘▗▐▀▗▗▘  \n" +
	"  ▀█▚▚▌▟▛█▐▗█▟▚▌▐▌▜▖▞▝▌  \n" +
	"  ▚▀▝▚▞▗▚▜▀▚▀▜▙ █▞▚▖▐▗▖  \n" +
	"  ▘█▛▀▖▟▙█▄█▙▄ ▖▚▚▟▝▖▐▘  \n" +
	"  ▘▝ ▘█▞▞▛▖▝▐ ▀▀▖▐▛▀▌▛▖  \n" +
	"  ▛▀▀▌▞▞▐▞▜▐▚▄█▖▛ ▌▘▙█   \n" +
	"  ▌█▌▌▞▟▛▙▖▐▄▀▛▝▜▚▀▛▙▟▘  \n" +
	"  ▌▀▘▌ ▘ ▄▖▘▟▀▐▙▙▀█▚▛▐▘  \n" +
	"  ▀▀▀▘▀ ▝ ▝ ▀▘▀   ▀▝▝▝   \n" +
	"                         \n" +
	"                         \n"

func TestRenderQR(t *testing.T) {
	got, err := renderQR(wantInvite)
	if err != nil {
		t.Fatalf("renderQR: %v", err)
	}
	if got != wantQR {
		t.Errorf("QR mismatch\ngot:\n%s\nwant:\n%s", got, wantQR)
	}
}

func TestInviteString(t *testing.T) {
	const wantPrefix = "$env:NS='"
	const wantSuffix = "irm https://raw.githubusercontent.com/simonthoresen/null-space/main/join.ps1|iex"

	if len(wantInvite) == 0 {
		t.Fatal("invite string is empty")
	}
	if wantInvite[:len(wantPrefix)] != wantPrefix {
		t.Errorf("invite does not start with %q: %q", wantPrefix, wantInvite)
	}
	if wantInvite[len(wantInvite)-len(wantSuffix):] != wantSuffix {
		t.Errorf("invite does not end with %q: %q", wantSuffix, wantInvite)
	}
}
