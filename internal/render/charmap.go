package render

// CanvasCell is a sentinel rune placed in viewport cells during Canvas HD
// rendering mode. The client treats these cells as transparent, allowing
// the locally-rendered canvas to show through. Menus/dialogs that overlap
// the viewport replace these cells with real content, rendering on top.
const CanvasCell rune = '\uF8FF'

// IsCanvasCell reports whether r is the canvas transparency placeholder.
func IsCanvasCell(r rune) bool {
	return r == CanvasCell
}
