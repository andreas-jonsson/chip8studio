package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
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

	"github.com/sqweek/dialog"

	"github.com/skip2/go-qrcode"

	"github.com/andreas-jonsson/chip8/chip8"
	"github.com/andreas-jonsson/chip8studio/assembler"
	"github.com/andreas-jonsson/chip8studio/emulator"
	"github.com/andreas-jonsson/chip8studio/example"
)

var (
	masterWindow nucular.MasterWindow
	textEditor   = &nucular.TextEditor{Flags: nucular.EditBox}
	debugEditor  = &nucular.TextEditor{Flags: nucular.EditMultiline | nucular.EditReadOnly | nucular.EditNoCursor | nucular.EditNoHorizontalScroll}
	logEditor    = &nucular.TextEditor{Flags: nucular.EditSelectable | nucular.EditMultiline | nucular.EditClipboard | nucular.EditReadOnly}

	logBuffer bytes.Buffer
	logger    = log.New(&logBuffer, "", 0)

	chippy *chip8.System
	system *emulator.Machine

	emulatorPaused int32 = 1
	projectFile    string
	projectName    = "PONG"
)

func init() {
	assembler.Logger = logger
}

func main() {
	masterWindow = nucular.NewMasterWindowSize(0, "Chip8 Studio - "+projectFile, image.Pt(1280, 720), func(*nucular.Window) {})

	flags := nucular.WindowTitle | nucular.WindowBorder | nucular.WindowMovable | nucular.WindowScalable | nucular.WindowNonmodal | nucular.WindowNoScrollbar
	masterWindow.PopupOpen("Project - "+projectName, flags, rect.Rect{0, 0, 590, 490}, true, textWindowUpdate)
	masterWindow.PopupOpen("Output", flags, rect.Rect{0, 500, 590, 210}, true, logWindowUpdate)
	masterWindow.PopupOpen("Debug", flags, rect.Rect{600, 370, 670, 340}, true, debugWindowUpdate)
	masterWindow.PopupOpen("Emulator", flags, rect.Rect{600, 0, 670, 360}, true, emulatorWindowUpdate)

	textEditor.Flags = nucular.EditBox
	masterWindow.ActivateEditor(textEditor)

	textEditor.Paste(example.Pong)

	system = &emulator.Machine{
		CpuSpeedHz: emulator.DefaultCPUSpeed,
		Program:    assembleBinary([]byte(example.Pong)),
	}
	chippy = chip8.NewSystem(system)

	go func() {
		for {
			if step := atomic.LoadInt32(&emulatorPaused); step <= 0 {
				system.Lock()
				if err := chippy.Step(); err != nil {
					logger.Println(err)
					masterWindow.Changed()
				}

				if chippy.Invalid() {
					masterWindow.Changed()
				}
				system.Unlock()

				if step < 0 {
					atomic.StoreInt32(&emulatorPaused, 1)
				}
			}

			time.Sleep(time.Second / system.CpuSpeedHz)
		}
	}()

	masterWindow.Main()
}

func runAssembler() []byte {
	source := []byte(string(textEditor.Buffer))
	if prog := assembleBinary(source); len(prog) > 0 {
		atomic.StoreInt32(&emulatorPaused, 1)

		system.Lock()
		system.Program = prog
		system.Unlock()
		chippy.Reset()
		return prog
	}
	return nil
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

	prog, err := ioutil.ReadFile(fpName)
	if err != nil {
		log.Fatalln(err)
	}
	os.Remove(fpName)

	return prog
}

func saveSource() {
	source := []byte(string(textEditor.Buffer))
	if err := ioutil.WriteFile(projectFile, source, 644); err != nil {
		logger.Println(err)
		logEditor.Buffer = []rune(string(logBuffer.Bytes()))
		masterWindow.Changed()
	}
}

func saveAsDialog() {
	if filename, err := dialog.File().Filter("Chip8 Assembly Source", "asm").Title("Save As").Save(); err == nil {
		projectFile = filename
		projectBase := filepath.Base(projectFile)
		projectName = strings.ToUpper(strings.TrimRight(projectBase, filepath.Ext(projectBase)))
		saveSource()
	}
}

func textWindowUpdate(w *nucular.Window) {
	w.MenubarBegin()
	w.Row(20).Static(50, 50, 50)

	if w := w.Menu(label.TA("File", "CC"), 120, nil); w != nil {
		w.Row(25).Dynamic(1)
		if w.MenuItem(label.TA("New", "LC")) {
			projectName = "UNTITLED"
			projectFile = ""
			textEditor.Buffer = nil
			runAssembler()
		}
		if w.MenuItem(label.TA("Open", "LC")) {
			if filename, err := dialog.File().Filter("Chip8 Assembly Source", "asm").Load(); err == nil {
				if source, err := ioutil.ReadFile(filename); err == nil {
					textEditor.Buffer = []rune(strings.Replace(string(source), "\r\n", "\n", -1))
					runAssembler()
					masterWindow.Changed()
				}
			}
		}
		if w.MenuItem(label.TA("Save", "LC")) {
			if projectFile == "" {
				saveAsDialog()
			} else {
				saveSource()
			}
		}
		if w.MenuItem(label.TA("Save As", "LC")) {
			saveAsDialog()
		}
		if w.MenuItem(label.TA("Exit", "LC")) {
			if dialog.Message("%s", "Do you want to exit? All unsaved data will be lost!").Title("Are you sure?").YesNo() {
				os.Exit(0)
			}
		}
	}
	if w := w.Menu(label.TA("Build", "CC"), 120, nil); w != nil {
		w.Row(25).Dynamic(1)
		if w.MenuItem(label.TA("Assemble", "LC")) {
			runAssembler()
		}
		if w.MenuItem(label.TA("Bundle (Binary)", "LC")) {
			if prog := runAssembler(); prog != nil {
				if filename, err := dialog.File().Filter("Chip8 Binary", "ch8").Title("Save As").Save(); err == nil {
					ioutil.WriteFile(filename, prog, 644)
				}
			}
		}
		if w.MenuItem(label.TA("Bundle (QR-Code)", "LC")) {
			if prog := runAssembler(); prog != nil {
				const imageSize = 512

				if data, err := qrcode.Encode(string(prog), qrcode.High, imageSize); err == nil {
					img, err := png.Decode(bytes.NewReader(data))
					if err != nil {
						log.Fatalln(err)
					}

					rgbaImage := image.NewRGBA(image.Rect(0, 0, imageSize, imageSize))
					draw.Draw(rgbaImage, img.Bounds(), img, image.ZP, draw.Over)

					masterWindow.PopupOpen("QR-Code", nucular.WindowTitle|nucular.WindowBorder|nucular.WindowMovable|nucular.WindowNoScrollbar|nucular.WindowClosable, rect.Rect{0, 0, imageSize + 15, imageSize + 35}, true, func(w *nucular.Window) {
						w.Row(imageSize).Static(imageSize)
						if w.Button(label.I(rgbaImage), false) {
							if filename, err := dialog.File().Filter("QR-Code", "png").Title("Write QR-Code").Save(); err == nil {
								if fp, err := os.Create(filename); err == nil {
									png.Encode(fp, img)
									fp.Close()
								}
							}
						}
					})
				}
			}
		}
	}
	w.MenubarEnd()

	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)
	textEditor.Edit(w)
}

func logWindowUpdate(w *nucular.Window) {
	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)
	logEditor.Buffer = []rune(string(logBuffer.Bytes()))
	logEditor.Edit(w)
}

func debugWindowUpdate(w *nucular.Window) {
	w.Row(25).Static(60, 60, 60)

	updateDebugWindow := false
	if atomic.LoadInt32(&emulatorPaused) != 0 {
		if w.ButtonText("Run") {
			atomic.StoreInt32(&emulatorPaused, 0)
			logger.Println("Run")
		}
		if w.ButtonText("Step") {
			atomic.StoreInt32(&emulatorPaused, -1)
			logger.Println("Step")
			updateDebugWindow = true

			for atomic.LoadInt32(&emulatorPaused) <= 0 {
				time.Sleep(time.Millisecond)
			}
		}
	} else {
		if w.ButtonText("Pause") {
			atomic.StoreInt32(&emulatorPaused, 1)
			logger.Println("Pause")
		}
	}

	if w.ButtonText("Reset") {
		system.Lock()
		chippy.Reset()
		system.Unlock()
		atomic.StoreInt32(&emulatorPaused, 1)
		logger.Println("Reset")
		updateDebugWindow = true
	}

	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)

	if debugEditor.Buffer == nil || updateDebugWindow || atomic.LoadInt32(&emulatorPaused) == 0 {
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
