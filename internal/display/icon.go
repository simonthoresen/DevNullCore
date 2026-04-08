package display

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"

	"github.com/hajimehoshi/ebiten/v2"
)

// SetWindowIconFromICO parses a Windows ICO file and sets the Ebitengine
// window icon to the largest image it contains.
func SetWindowIconFromICO(data []byte) error {
	imgs, err := ParseICO(data)
	if err != nil {
		return err
	}
	if len(imgs) == 0 {
		return fmt.Errorf("ico: no images found")
	}
	ebiten.SetWindowIcon(imgs)
	return nil
}

// ParseICO extracts all images from a Windows ICO file.
// Supports both PNG-compressed and 32-bit BMP entries.
func ParseICO(data []byte) ([]image.Image, error) {
	if len(data) < 6 {
		return nil, fmt.Errorf("ico: too short")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if len(data) < 6+count*16 {
		return nil, fmt.Errorf("ico: truncated directory")
	}

	var imgs []image.Image
	for i := range count {
		off := 6 + i*16
		size := int(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		dataOff := int(binary.LittleEndian.Uint32(data[off+12 : off+16]))
		if dataOff+size > len(data) {
			return nil, fmt.Errorf("ico: entry %d out of bounds", i)
		}
		entry := data[dataOff : dataOff+size]

		var img image.Image
		var err error
		if len(entry) >= 8 && string(entry[:8]) == "\x89PNG\r\n\x1a\n" {
			img, err = png.Decode(io.NewSectionReader(
				readerAt(data), int64(dataOff), int64(size)))
		} else {
			img, err = decodeBMPEntry(entry)
		}
		if err != nil {
			return nil, fmt.Errorf("ico: entry %d: %w", i, err)
		}
		imgs = append(imgs, img)
	}
	return imgs, nil
}

// decodeBMPEntry decodes a 32-bit BGRA BMP entry from an ICO file.
func decodeBMPEntry(data []byte) (image.Image, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("bmp: header too short")
	}
	width := int(binary.LittleEndian.Uint32(data[4:8]))
	// ICO BMP height is doubled (includes AND mask).
	height := int(binary.LittleEndian.Uint32(data[8:12])) / 2
	bpp := int(binary.LittleEndian.Uint16(data[14:16]))
	if bpp != 32 {
		return nil, fmt.Errorf("bmp: unsupported bpp %d (need 32)", bpp)
	}

	pixelStart := 40 // BITMAPINFOHEADER size
	stride := width * 4
	needed := pixelStart + height*stride
	if len(data) < needed {
		return nil, fmt.Errorf("bmp: truncated pixel data")
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		// BMP rows are bottom-up.
		srcRow := data[pixelStart+(height-1-y)*stride:]
		for x := range width {
			b := srcRow[x*4+0]
			g := srcRow[x*4+1]
			r := srcRow[x*4+2]
			a := srcRow[x*4+3]
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img, nil
}

type readerAt []byte

func (r readerAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r)) {
		return 0, io.EOF
	}
	n := copy(p, r[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
