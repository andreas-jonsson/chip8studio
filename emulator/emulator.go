package emulator

import (
	"image"
	"math/rand"
	"sync"
	"time"

	"github.com/nfnt/resize"

	"github.com/aarzilli/nucular"
)

const DefaultCPUSpeed = 500

type Machine struct {
	sync.Mutex

	Program    []byte
	Window     *nucular.Window
	CpuSpeedHz time.Duration

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
		pix[i*4] = p
		pix[i*4+1] = 0x0
		pix[i*4+2] = 0x0
		pix[i*4+3] = 0xFF
	}

	w := m.Window
	bounds := w.Bounds
	w.Row(bounds.H).Static(bounds.W)
	w.Image(resize.Resize(uint(bounds.W-15), 0, m.backBuffer, resize.NearestNeighbor).(*image.RGBA))
}
