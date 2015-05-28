// Copyright 2015 Matthew Collins
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"fmt"

	"github.com/thinkofdeath/steven/chat"
	"github.com/thinkofdeath/steven/render"
	"github.com/thinkofdeath/steven/resource/locale"
)

// Formatted is a drawable that draws a string.
type Formatted struct {
	baseElement
	value          chat.AnyComponent
	x, y           float64
	MaxWidth       float64
	scaleX, scaleY float64

	Width, Height float64
	Lines         int

	Text []*Text
}

// NewFormatted creates a new Formatted drawable.
func NewFormatted(val chat.AnyComponent, x, y float64) *Formatted {
	f := &Formatted{
		x: x, y: y,
		scaleX: 1, scaleY: 1,
		MaxWidth: -1,
		baseElement: baseElement{
			visible: true,
			isNew:   true,
		},
	}
	f.Update(val)
	return f
}

// NewFormattedWidth creates a new Formatted drawable with a max width.
func NewFormattedWidth(val chat.AnyComponent, x, y, width float64) *Formatted {
	f := &Formatted{
		x: x, y: y,
		scaleX: 1, scaleY: 1,
		MaxWidth: width,
		baseElement: baseElement{
			visible: true,
			isNew:   true,
		},
	}
	f.Update(val)
	return f
}

// Attach changes the location where this is attached to.
func (f *Formatted) Attach(vAttach, hAttach AttachPoint) *Formatted {
	f.vAttach, f.hAttach = vAttach, hAttach
	return f
}

func (f *Formatted) X() float64 { return f.x }
func (f *Formatted) SetX(x float64) {
	if f.x != x {
		f.x = x
		f.dirty = true
	}
}
func (f *Formatted) Y() float64 { return f.y }
func (f *Formatted) SetY(y float64) {
	if f.y != y {
		f.y = y
		f.dirty = true
	}
}
func (f *Formatted) ScaleX() float64 { return f.scaleX }
func (f *Formatted) SetScaleX(s float64) {
	if f.scaleX != s {
		f.scaleX = s
		f.dirty = true
	}
}
func (f *Formatted) ScaleY() float64 { return f.scaleY }
func (f *Formatted) SetScaleY(s float64) {
	if f.scaleY != s {
		f.scaleY = s
		f.dirty = true
	}
}

// Draw draws this to the target region.
func (f *Formatted) Draw(r Region, delta float64) {
	if f.isNew || f.isDirty() || forceDirty {
		cw, ch := f.Size()
		sx, sy := r.W/cw, r.H/ch
		f.data = f.data[:0]
		for _, t := range f.Text {
			r := getDrawRegion(t, sx, sy)
			t.SetLayer(f.layer)
			t.dirty = true
			t.Draw(r, delta)
			f.data = append(f.data, t.data...)
		}
		f.isNew = false
	}
	render.UIAddBytes(f.data)
}

// Offset returns the offset of this drawable from the attachment
// point.
func (f *Formatted) Offset() (float64, float64) {
	return f.x, f.y
}

// Size returns the size of this drawable.
func (f *Formatted) Size() (float64, float64) {
	return (f.Width + 2) * f.scaleX, f.Height * f.scaleY
}

// Remove removes the Formatted element from the draw list.
func (f *Formatted) Remove() {
	Remove(f)
}

// Update updates the component drawn by this drawable.
func (f *Formatted) Update(val chat.AnyComponent) {
	f.value = val
	f.Text = f.Text[:0]
	state := formatState{
		f: f,
	}
	state.build(val, func() chat.Color { return chat.White })
	f.Height = float64(state.lines+1) * 18
	f.Width = state.width
	f.Lines = state.lines + 1
	f.dirty = true
}
func (f *Formatted) isDirty() bool {
	if f.baseElement.isDirty() {
		return true
	}
	for _, t := range f.Text {
		if t.dirty {
			return true
		}
	}
	return false
}

func (f *Formatted) clearDirty() {
	f.dirty = false
	for _, t := range f.Text {
		t.clearDirty()
	}
}

type formatState struct {
	f      *Formatted
	lines  int
	offset float64
	width  float64
}

func (f *formatState) build(c chat.AnyComponent, color getColorFunc) {
	switch c := c.Value.(type) {
	case *chat.TextComponent:
		gc := getColor(&c.Component, color)
		f.appendText(c.Text, gc)
		for _, e := range c.Extra {
			f.build(e, gc)
		}
	case *chat.TranslateComponent:
		gc := getColor(&c.Component, color)
		for _, part := range locale.Get(c.Translate) {
			switch part := part.(type) {
			case string:
				f.appendText(part, gc)
			case int:
				if part < 0 || part >= len(c.With) {
					continue
				}
				f.build(c.With[part], gc)
			}
		}

	default:
		panic(fmt.Sprintf("unhandled component: %T", c))
	}
}

func (f *formatState) appendText(text string, color getColorFunc) {
	width := 0.0
	last := 0
	for i, r := range text {
		s := render.SizeOfCharacter(r) + 2
		if (f.f.MaxWidth > 0 && f.offset+width+s > f.f.MaxWidth) || r == '\n' {
			rr, gg, bb := colorRGB(color())
			txt := NewText(text[last:i], f.offset, float64(f.lines*18+1), rr, gg, bb)
			txt.AttachTo(f.f)
			last = i
			if r == '\n' {
				last++
			}
			f.f.Text = append(f.f.Text, txt)
			f.offset = 0
			f.lines++
			width = 0
		}
		width += s
		if f.offset+width > f.width {
			f.width = f.offset + width
		}
	}
	if last != len(text) {
		r, g, b := colorRGB(color())
		txt := NewText(text[last:], f.offset, float64(f.lines*18+1), r, g, b)
		txt.AttachTo(f.f)
		f.f.Text = append(f.f.Text, txt)
		f.offset += txt.Width + 2
		if f.offset > f.width {
			f.width = f.offset
		}
	}
}

type getColorFunc func() chat.Color

func getColor(c *chat.Component, parent getColorFunc) getColorFunc {
	return func() chat.Color {
		if c.Color != "" {
			return c.Color
		}
		if parent != nil {
			return parent()
		}
		return chat.White
	}
}

func colorRGB(c chat.Color) (r, g, b int) {
	switch c {
	case chat.Black:
		return 0, 0, 0
	case chat.DarkBlue:
		return 0, 0, 170
	case chat.DarkGreen:
		return 0, 170, 0
	case chat.DarkAqua:
		return 0, 170, 170
	case chat.DarkRed:
		return 170, 0, 0
	case chat.DarkPurple:
		return 170, 0, 170
	case chat.Gold:
		return 255, 170, 0
	case chat.Gray:
		return 170, 170, 170
	case chat.DarkGray:
		return 85, 85, 85
	case chat.Blue:
		return 85, 85, 255
	case chat.Green:
		return 85, 255, 85
	case chat.Aqua:
		return 85, 255, 255
	case chat.Red:
		return 255, 85, 85
	case chat.LightPurple:
		return 255, 85, 255
	case chat.Yellow:
		return 255, 255, 85
	case chat.White:
		return 255, 255, 255
	}
	return 255, 255, 255
}
