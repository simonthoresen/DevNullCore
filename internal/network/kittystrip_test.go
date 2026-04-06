//go:build !windows

package network

import (
	"bytes"
	"testing"
)

// TestKittyStripWriterONLCR verifies that bare \n is mapped to \r\n (ONLCR),
// while \r\n is left unchanged and Kitty sequences are still stripped.
//
// Without the fix (applyONLCR removed), Bubble Tea's mapNl mode emits bare \n
// over SSH, the cursor moves down without returning to column 0, and characters
// on subsequent rows land shifted right — accumulating as rendering artifacts.
func TestKittyStripWriterONLCR(t *testing.T) {
	tests := []struct {
		name string
		// writes are applied as separate Write calls to test cross-call state.
		writes [][]byte
		want   []byte
	}{
		{
			name:   "bare newline maps to CR LF",
			writes: [][]byte{[]byte("abc\ndef")},
			want:   []byte("abc\r\ndef"),
		},
		{
			name:   "existing CR LF is not doubled",
			writes: [][]byte{[]byte("abc\r\ndef")},
			want:   []byte("abc\r\ndef"),
		},
		{
			name:   "multiple bare newlines",
			writes: [][]byte{[]byte("a\nb\nc")},
			want:   []byte("a\r\nb\r\nc"),
		},
		{
			name:   "CR LF split across two writes — no double CR",
			writes: [][]byte{[]byte("abc\r"), []byte("\ndef")},
			want:   []byte("abc\r\ndef"),
		},
		{
			name:   "bare newline at start of second write",
			writes: [][]byte{[]byte("abc"), []byte("\ndef")},
			want:   []byte("abc\r\ndef"),
		},
		{
			name:   "kitty sequences stripped alongside ONLCR",
			writes: [][]byte{[]byte("\x1b[>1uabc\ndef\x1b[<u")},
			want:   []byte("abc\r\ndef"),
		},
		{
			name:   "no newlines — unchanged",
			writes: [][]byte{[]byte("hello world")},
			want:   []byte("hello world"),
		},
		{
			name:   "empty write",
			writes: [][]byte{{}},
			want:   []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewKittyStripWriter(&buf)
			for _, p := range tt.writes {
				n, err := w.Write(p)
				if err != nil {
					t.Fatalf("Write() error: %v", err)
				}
				// Write must report the original length, not the expanded length.
				if n != len(p) {
					t.Fatalf("Write() returned n=%d, want %d", n, len(p))
				}
			}
			if got := buf.Bytes(); !bytes.Equal(got, tt.want) {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}
