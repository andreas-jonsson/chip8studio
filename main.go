package main

import (
	"bytes"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aarzilli/nucular"
	"github.com/aarzilli/nucular/label"
	"github.com/aarzilli/nucular/rect"
	"github.com/andreas-jonsson/chip8/chip8"
	"github.com/andreas-jonsson/chip8studio/assembler"
	"github.com/andreas-jonsson/chip8studio/emulator"
)

var (
	masterWindow nucular.MasterWindow
	textEditor   = &nucular.TextEditor{Flags: nucular.EditBox}
	debugEditor  = &nucular.TextEditor{Flags: nucular.EditMultiline | nucular.EditReadOnly}
	logEditor    = &nucular.TextEditor{Flags: nucular.EditSelectable | nucular.EditMultiline | nucular.EditClipboard | nucular.EditReadOnly}

	logBuffer bytes.Buffer

	chippy *chip8.System
	system *emulator.Machine

	emulatorPaused int32 = 1
	projectName    string
)

func init() {
	assembler.Logger = log.New(&logBuffer, "ASM: ", 0)
}

func main() {
	const projectFile = "../chip8/cmd/asm/tests/pong.asm"
	projectBase := filepath.Base(projectFile)
	projectName = strings.ToUpper(strings.TrimRight(projectBase, filepath.Ext(projectBase)))

	masterWindow = nucular.NewMasterWindowSize(0, "Chip8 Studio - "+projectFile, image.Pt(1280, 720), func(*nucular.Window) {})

	flags := nucular.WindowTitle | nucular.WindowBorder | nucular.WindowMovable | nucular.WindowScalable | nucular.WindowNonmodal | nucular.WindowNoScrollbar
	masterWindow.PopupOpen(projectName, flags, rect.Rect{0, 0, 400, 600}, true, textWindowUpdate)
	masterWindow.PopupOpen("Output", flags, rect.Rect{0, 0, 400, 200}, true, logWindowUpdate)
	masterWindow.PopupOpen("Debug", flags, rect.Rect{0, 0, 400, 400}, true, debugWindowUpdate)
	masterWindow.PopupOpen("Emulator", flags, rect.Rect{0, 0, 640, 360}, true, emulatorWindowUpdate)

	textEditor.Flags = nucular.EditBox
	masterWindow.ActivateEditor(textEditor)

	source, err := ioutil.ReadFile(projectFile)
	if err != nil {
		log.Fatalln(err)
	}

	textEditor.Paste(strings.Replace(string(source), "\r\n", "\n", -1))

	system = &emulator.Machine{
		CpuSpeedHz: emulator.DefaultCPUSpeed,
		Program:    assembleBinary(source),
	}
	chippy = chip8.NewSystem(system)

	go func() {
		for {
			if atomic.LoadInt32(&emulatorPaused) == 0 {
				system.Lock()
				if err := chippy.Step(); err != nil {
					fmt.Println(err)
				}

				if chippy.Invalid() {
					masterWindow.Changed()
				}
				system.Unlock()
			}

			time.Sleep(time.Second / system.CpuSpeedHz)
		}
	}()

	masterWindow.Main()
}

func runAssembler() {
	source := []byte(string(textEditor.Buffer))
	if prog := assembleBinary(source); len(prog) > 0 {
		atomic.StoreInt32(&emulatorPaused, 1)

		system.Lock()
		system.Program = prog
		system.Unlock()
		chippy.Reset()
	}
}

func assembleBinary(source []byte) []byte {
	fp, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatalln(err)
	}

	_, errs := assembler.Assemble(projectName, bytes.NewReader(source), fp)
	if len(errs) > 0 {
		return nil
	}

	fpName := fp.Name()
	fp.Close()

	logEditor.Buffer = []rune(string(logBuffer.Bytes()))

	prog, err := ioutil.ReadFile(fpName)
	if err != nil {
		log.Fatalln(err)
	}
	os.Remove(fpName)

	return prog
}

func textWindowUpdate(w *nucular.Window) {
	w.MenubarBegin()
	w.Row(20).Static(50, 50)

	if w := w.Menu(label.TA("Project", "CC"), 120, nil); w != nil {
		w.Row(25).Dynamic(1)
		if w.MenuItem(label.TA("New", "LC")) {
		}
		if w.MenuItem(label.TA("Open", "LC")) {
		}
		if w.MenuItem(label.TA("Save", "LC")) {
		}
	}
	if w := w.Menu(label.TA("Build", "CC"), 120, nil); w != nil {
		w.Row(25).Dynamic(1)
		if w.MenuItem(label.TA("Assemble", "LC")) {
			runAssembler()
		}
		if w.MenuItem(label.TA("Bundle", "LC")) {
		}
	}
	w.MenubarEnd()

	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)
	textEditor.Edit(w)
}

func logWindowUpdate(w *nucular.Window) {
	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)
	logEditor.Edit(w)
}

func debugWindowUpdate(w *nucular.Window) {
	w.Row(25).Static(60, 60, 60)

	if atomic.LoadInt32(&emulatorPaused) != 0 {
		if w.ButtonText("Run") {
			atomic.StoreInt32(&emulatorPaused, 0)
		}
		if w.ButtonText("Step") {
			atomic.StoreInt32(&emulatorPaused, 0)
		}
	} else {
		if w.ButtonText("Pause") {
			atomic.StoreInt32(&emulatorPaused, 1)
		}
	}

	didReset := false
	if w.ButtonText("Reset") {
		system.Lock()
		chippy.Reset()
		system.Unlock()
		atomic.StoreInt32(&emulatorPaused, 1)
		didReset = true
	}

	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)

	if debugEditor.Buffer == nil || didReset || atomic.LoadInt32(&emulatorPaused) == 0 {
		system.Lock()
		var buf bytes.Buffer
		chippy.Dump(&buf, projectName)
		system.Unlock()

		debugEditor.Buffer = []rune(buf.String())
	}
	debugEditor.Edit(w)
}

func emulatorWindowUpdate(w *nucular.Window) {
	if system.Window == nil {
		system.Window = w
	}

	system.Lock()
	chippy.Invalidate()
	chippy.Refresh()
	system.Unlock()
}
