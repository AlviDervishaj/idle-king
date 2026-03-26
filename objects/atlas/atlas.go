// Package atlas splits PNG tilesets into cells (pure decode helpers).
package atlas

import (
	"bytes"
	"errors"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

// SplitPNGGrid decodes PNG bytes and returns each cell as a sub-image (row-major).
func SplitPNGGrid(data []byte, cellW, cellH int) ([]*ebiten.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	eimg := ebiten.NewImageFromImage(img)
	b := eimg.Bounds()
	cols := b.Dx() / cellW
	rows := b.Dy() / cellH
	if cols <= 0 || rows <= 0 {
		return nil, errors.New("atlas: image smaller than cell size")
	}
	out := make([]*ebiten.Image, 0, cols*rows)
	for ry := range rows {
		for rx := range cols {
			r := image.Rect(b.Min.X+rx*cellW, b.Min.Y+ry*cellH, b.Min.X+(rx+1)*cellW, b.Min.Y+(ry+1)*cellH)
			out = append(out, eimg.SubImage(r).(*ebiten.Image))
		}
	}
	return out, nil
}

// SplitHorizontalStrip splits a single-row strip (e.g. walk cycle) into fixed-size frames.
func SplitHorizontalStrip(data []byte, frameW, frameH int) ([]*ebiten.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	eimg := ebiten.NewImageFromImage(img)
	b := eimg.Bounds()
	if b.Dy() < frameH {
		return nil, errors.New("atlas: strip height smaller than frame")
	}
	n := b.Dx() / frameW
	out := make([]*ebiten.Image, 0, n)
	for i := range n {
		r := image.Rect(b.Min.X+i*frameW, b.Min.Y, b.Min.X+(i+1)*frameW, b.Min.Y+frameH)
		out = append(out, eimg.SubImage(r).(*ebiten.Image))
	}
	return out, nil
}
