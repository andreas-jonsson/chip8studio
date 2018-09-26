/*
Copyright (C) 2018 Andreas T Jonsson

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package emulator

import (
	"image"
	"image/color/palette"
	"math/rand"
	"sync"
	"time"

	"github.com/nfnt/resize"

	"golang.org/x/mobile/event/key"

	"github.com/aarzilli/nucular"
)

const DefaultCPUSpeed = 500

type Machine struct {
	sync.Mutex

	Program    []byte
	Window     *nucular.Window
	CpuSpeedHz time.Duration
	Event      *key.Event

	backBuffer *image.RGBA
}

func (m *Machine) Load(memory []byte) {
	copy(memory, m.Program)
}

func (m *Machine) Rand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (m *Machine) BeginTone() {
}

func (m *Machine) EndTone() {
}

func (m *Machine) Key(code int) bool {
	if m.Event != nil {
		switch key.Code(m.Event.Code) {
		case key.Code0, key.CodeKeypad0:
			return code == 0
		case key.Code1, key.CodeKeypad1:
			return code == 1
		case key.Code2, key.CodeKeypad2:
			return code == 2
		case key.Code3, key.CodeKeypad3:
			return code == 3
		case key.Code4, key.CodeKeypad4:
			return code == 4
		case key.Code5, key.CodeKeypad5:
			return code == 5
		case key.Code6, key.CodeKeypad6:
			return code == 6
		case key.Code7, key.CodeKeypad7:
			return code == 7
		case key.Code8, key.CodeKeypad8:
			return code == 8
		case key.Code9, key.CodeKeypad9:
			return code == 9
		case key.CodeA:
			return code == 0xA
		case key.CodeB:
			return code == 0xB
		case key.CodeC:
			return code == 0xC
		case key.CodeD:
			return code == 0xD
		case key.CodeE:
			return code == 0xE
		case key.CodeF:
			return code == 0xF
		}
	}

	return false
}
func (m *Machine) SetCPUFrequency(freq int) {
	m.CpuSpeedHz = time.Duration(freq)
}

func (m *Machine) ResizeVideo(width int) {
}

func (m *Machine) Draw(video []byte) {
	if m.backBuffer == nil {
		m.backBuffer = image.NewRGBA(image.Rect(0, 0, 64, 32))
	}

	pix := m.backBuffer.Pix
	for i, p := range video {
		r, g, b, _ := palette.Plan9[p].RGBA()
		pix[i*4] = byte(r)
		pix[i*4+1] = byte(g)
		pix[i*4+2] = byte(b)
		pix[i*4+3] = 0xFF
	}

	w := m.Window
	bounds := w.Bounds
	w.Row(bounds.H).Static(bounds.W)
	w.Image(resize.Resize(uint(bounds.W-15), 0, m.backBuffer, resize.NearestNeighbor).(*image.RGBA))
}
