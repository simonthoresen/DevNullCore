package main

import "io"

// ONLCRWriter maps bare \n → \r\n (ONLCR) in the output stream.
//
// Bubble Tea v2 sets mapNl=true on non-Windows when the output has no Fd()
// (which ssh.Session doesn't). In that mode the renderer emits bare \n for
// vertical movement, expecting the PTY to supply the \r (ONLCR). Wish's
// emulated PTY does not implement ONLCR, so we do it here.
type ONLCRWriter struct {
	w        io.Writer
	lastByte byte
}

func NewONLCRWriter(w io.Writer) *ONLCRWriter {
	return &ONLCRWriter{w: w}
}

func (o *ONLCRWriter) Write(p []byte) (int, error) {
	original := len(p)
	cleaned := o.applyONLCR(p)
	if len(cleaned) == 0 {
		return original, nil
	}
	_, err := o.w.Write(cleaned)
	if err != nil {
		return 0, err
	}
	return original, nil
}

// Read delegates to the underlying writer if it also implements io.Reader
// (which ssh.Session does).
func (o *ONLCRWriter) Read(p []byte) (int, error) {
	if r, ok := o.w.(io.Reader); ok {
		return r.Read(p)
	}
	return 0, io.EOF
}

func (o *ONLCRWriter) applyONLCR(p []byte) []byte {
	prior := o.lastByte

	prev := prior
	count := 0
	for _, b := range p {
		if b == '\n' && prev != '\r' {
			count++
		}
		prev = b
	}

	if len(p) > 0 {
		o.lastByte = p[len(p)-1]
	}

	if count == 0 {
		return p
	}

	out := make([]byte, 0, len(p)+count)
	prev = prior
	for _, b := range p {
		if b == '\n' && prev != '\r' {
			out = append(out, '\r')
		}
		out = append(out, b)
		prev = b
	}
	return out
}
