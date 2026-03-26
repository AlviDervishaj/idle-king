package game

import (
	"bytes"
	"image/gif"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func decodeGIF(data []byte) (frames []*ebiten.Image, delays []int, err error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	frames = make([]*ebiten.Image, len(g.Image))
	for i, im := range g.Image {
		frames[i] = ebiten.NewImageFromImage(im)
	}
	delays = g.Delay
	if len(delays) != len(frames) {
		delays = make([]int, len(frames))
		for i := range delays {
			delays[i] = 10
		}
	}
	return frames, delays, nil
}

type gifAnim struct {
	frames   []*ebiten.Image
	delays   []int
	idx      int
	accum    time.Duration
	loop     bool
	finished bool
}

func newGIFAnim(data []byte, loop bool) (*gifAnim, error) {
	frames, delays, err := decodeGIF(data)
	if err != nil {
		return nil, err
	}
	return &gifAnim{frames: frames, delays: delays, loop: loop}, nil
}

func newGIFAnimShared(frames []*ebiten.Image, delays []int, loop bool) *gifAnim {
	if len(frames) == 0 {
		return nil
	}
	d := delays
	if len(d) != len(frames) {
		d = make([]int, len(frames))
		for i := range d {
			d[i] = 10
		}
	}
	return &gifAnim{frames: frames, delays: d, loop: loop}
}

func (a *gifAnim) update(dt float64) {
	if a == nil || len(a.frames) == 0 || a.finished {
		return
	}
	a.accum += time.Duration(dt * float64(time.Second))
	delay := time.Duration(a.delays[a.idx]) * 10 * time.Millisecond
	if delay == 0 {
		delay = 100 * time.Millisecond
	}
	for a.accum >= delay {
		a.accum -= delay
		if a.idx+1 >= len(a.frames) {
			if a.loop {
				a.idx = 0
			} else {
				a.finished = true
				return
			}
		} else {
			a.idx++
		}
		delay = time.Duration(a.delays[a.idx]) * 10 * time.Millisecond
		if delay == 0 {
			delay = 100 * time.Millisecond
		}
	}
}

func (a *gifAnim) frame() *ebiten.Image {
	if a == nil || len(a.frames) == 0 {
		return nil
	}
	return a.frames[a.idx]
}

func (a *gifAnim) jumpToFrame(i int) {
	if a == nil || len(a.frames) == 0 {
		return
	}
	a.idx = i % len(a.frames)
	a.accum = 0
	a.finished = false
}
