package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
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

	chippy *chip8.System
	system *emulator.Machine
)

func main() {
	masterWindow = nucular.NewMasterWindow(0, "Chip8 Studio", func(w *nucular.Window) {
		w.MenubarBegin()
		w.Row(25).Static(45, 45, 70, 70, 70)
		if w := w.Menu(label.TA("MENU", "CC"), 120, nil); w != nil {
			w.Row(25).Dynamic(1)
			if w.MenuItem(label.TA("Hide", "LC")) {
				//od.ShowMenu = false
			}
			if w.MenuItem(label.TA("About", "LC")) {
				//od.showAppAbout(w.Master())
			}
			//w.Progress(&od.Prog, 100, true)
			//w.SliderInt(0, &od.Slider, 16, 1)
			//w.CheckboxText("check", &od.Check)
		}
		if w := w.Menu(label.TA("THEME", "CC"), 180, nil); w != nil {
			w.Row(25).Dynamic(1)
			//newtheme := od.Theme
			if w.OptionText("Default Theme", true) {
				//newtheme = nstyle.DefaultTheme
			}
			if w.OptionText("White Theme", true) {
				//newtheme = nstyle.WhiteTheme
			}
			if w.OptionText("Red Theme", false) {
				//newtheme = nstyle.RedTheme
			}
			if w.OptionText("Dark Theme", false) {
				//newtheme = nstyle.DarkTheme
			}
			//if newtheme != od.Theme {
			//od.Theme = newtheme
			//	w.Master().SetStyle(nstyle.FromTheme(od.Theme, w.Master().Style().Scaling))
			//	w.Close()
			//}
		}
		//w.Progress(&od.Mprog, 100, true)
		//w.SliderInt(0, &od.Mslider, 16, 1)
		//w.CheckboxText("check", &od.Mcheck)
		w.MenubarEnd()
	})

	flags := nucular.WindowTitle | nucular.WindowBorder | nucular.WindowMovable | nucular.WindowScalable | nucular.WindowNonmodal | nucular.WindowNoScrollbar | nucular.WindowClosable
	masterWindow.PopupOpen("Source", flags, rect.Rect{0, 0, 400, 600}, true, textWindowUpdate)
	masterWindow.PopupOpen("Output", flags, rect.Rect{0, 0, 400, 200}, true, logWindowUpdate)
	masterWindow.PopupOpen("Debug", flags, rect.Rect{0, 0, 400, 400}, true, debugWindowUpdate)
	masterWindow.PopupOpen("Emulator", flags, rect.Rect{0, 0, 640, 360}, true, emulatorWindowUpdate)

	textEditor.Flags = nucular.EditBox
	masterWindow.ActivateEditor(textEditor)

	source, err := ioutil.ReadFile("../chip8/cmd/asm/tests/pong.asm")
	if err != nil {
		log.Fatalln(err)
	}

	textEditor.Paste(strings.Replace(string(source), "\r\n", "\n", -1))

	fp, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatalln(err)
	}

	var buf bytes.Buffer
	assembler.Logger = log.New(&buf, "ASM: ", 0)
	assembler.Assemble("PONG", bytes.NewReader(source), fp)

	name := fp.Name()
	fp.Close()

	logEditor.Buffer = []rune(string(buf.Bytes()))

	prog, err := ioutil.ReadFile(name)
	if err != nil {
		log.Fatalln(err)
	}
	os.Remove(name)

	system = &emulator.Machine{
		CpuSpeedHz: emulator.DefaultCPUSpeed,
		Program:    prog,
	}
	chippy = chip8.NewSystem(system)

	go func() {
		for {
			system.Lock()
			if err := chippy.Step(); err != nil {
				fmt.Println(err)
			}

			if chippy.Invalid() {
				masterWindow.Changed()
			}
			system.Unlock()

			time.Sleep(time.Second / system.CpuSpeedHz)
		}
	}()

	masterWindow.Main()
}

func textWindowUpdate(w *nucular.Window) {
	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)
	textEditor.Edit(w)
}

func logWindowUpdate(w *nucular.Window) {
	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)
	logEditor.Edit(w)
}

func debugWindowUpdate(w *nucular.Window) {
	w.Row(w.Bounds.H - 50).Static(w.Bounds.W - 15)

	system.Lock()
	var buf bytes.Buffer
	chippy.Dump(&buf, "PONG")
	system.Unlock()

	debugEditor.Buffer = []rune(buf.String())
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
