package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wdevore/hardware/ftdi"
	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/ftdi/devices/hx8357"
	"github.com/wdevore/hardware/gpio"

	term "github.com/nsf/termbox-go"
)

var quit = false
var keyPressed rune

func main() {
	// log.SetFlags(0)
	// log.SetOutput(ioutil.Discard)

	// terr := term.Init()
	// if terr != nil {
	// 	panic("Could not initialize termbox.")
	// }

	// defer term.Close()
	// term.SetInputMode(term.InputEsc)

	// term.Clear(term.ColorDefault, term.ColorDefault)
	// printf_tb(3, 1, term.ColorWhite, term.ColorBlack, "Starting...")
	log.Println("Starting...")
	// term.Flush()

	go keyScan()

	// Per Adafruit's FT232H breakout board configured as Hardware SPI:
	// https://learn.adafruit.com/adafruit-ft232h-breakout?view=all

	// D0 - Clock signal output.  This line can be configured as a clock that runs at speeds between ~450Hz to 30Mhz.
	// D1 - Serial data output.  This is for outputting a serial signal, like the MOSI line in a SPI connection.
	// D2 - Serial data input.  This is for reading a serial signal, like the MISO line in a SPI connection.
	// --> D3 - Serial select signal.  This is a chip select or chip enable signal to tell a connected device that the FT232H is ready to talk to it.

	hx := hx8357.NewHX8357D(ftdi.D5, ftdi.D4, devices.GreenTab, devices.D320x480)

	// Note this doesn't work when termbox-go is used
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("\nReceived ctrl-C, quiting.")
		quit = true
	}()

	if hx == nil {
		panic("Unable to create ST7735R component")
	}

	// printf_tb(3, 1, term.ColorWhite, term.ColorBlack, "Initializing device...")
	log.Println("Initializing device...")
	// term.Flush()

	// err := hx.Initialize(0x0403, 0x06014, 1000000, gpio.DefaultPin, devices.Orientation0)
	err := hx.Initialize(0x0403, 0x06014, 0, gpio.DefaultPin, devices.Orientation0)
	// st.SetConstantCSAssert(false)

	if err != nil {
		log.Println("Example: Failed to initialize ST7735R component")
		log.Fatal(err)
		os.Exit(-1)
	}

	defer hx.Close()

	// term.Clear(term.ColorDefault, term.ColorDefault)

	// printf_tb(3, 4, term.ColorWhite, term.ColorBlack, "Display dimensions: %d x %d\n", hx.Width, hx.Height)
	// printf_tb(3, 1, term.ColorWhite, term.ColorBlack, "Beginning test.")
	// term.Flush()
	log.Printf("Display dimensions: %d x %d\n", hx.Width, hx.Height)
	log.Println("Beginning test.")

	// nonBufferedTest(hx)
	bufferedTest(hx)
	log.Println("Done.")

	// printf_tb(3, 1, term.ColorWhite, term.ColorBlack, "Done.")
	// term.Flush()

	// printf_tb(3, 15, term.ColorWhite, term.ColorBlack, "Press 'Enter' to continue...")
	// term.Flush()
	log.Println("Press 'Enter' to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// A Goroutine so that the pollEvent doesn't block on the main rendering thread.
func keyScan() {
keyPressListenerLoop:
	for {
		switch ev := term.PollEvent(); ev.Type {
		case term.EventKey:
			switch ev.Key {
			case term.KeyEsc:
				term.Sync()
				quit = true
				break keyPressListenerLoop
			// case term.KeySpace:
			// 	term.Sync()
			default:
				term.Sync()
				keyPressed = ev.Ch
				// printfTb(3, 7, term.ColorWhite, term.ColorBlack, "Key: %c\n", ev.Ch)
				term.Flush()
			}
		case term.EventError:
			fmt.Printf("Term err (%v)\n", ev.Err)
		}

	}
}
func exitProg(hx *hx8357.HX8357D) {
	log.Println("Closing devices")
	err := hx.Close()
	if err != nil {
		log.Println("\n Failed to close FTDI component")
		os.Exit(-1)
	}
	// term.Close()
	os.Exit(0)
}

type colorSquare struct {
	x, y uint16
	c    uint16
}

func bufferedTest(hx *hx8357.HX8357D) {
	// hx.FillRectangle(0, 0, hx.Width, 50, devices.GREY)

	// hx.FillScreenToBuf(devices.GREY)
	hx.FillScreenToBuf(devices.BLACK)

	// the action area
	hx.FillRectangleToBuf(0, 0, 200, 160, devices.LightGREY)

	hx.FillRectangleToBuf(200, 0, 120, 160, devices.ORANGE)
	hx.FillRectangleToBuf(0, 160, 320, 480-160, devices.GREY)

	t1 := time.Now()

	hx.Blit()

	elapsed := time.Since(t1)

	fmt.Printf("blit took: (%f)ms\n", float32(elapsed.Nanoseconds())/1000000.0)
}

func bufferedTest_Basic(hx *hx8357.HX8357D) {
	// hx.FillRectangle(0, 0, hx.Width, 50, devices.GREY)
	log.Println("Filled area")

	// hx.FillScreenToBuf(devices.GREY)
	hx.FillScreenToBuf(devices.BLACK)

	hx.FillRectangleToBuf(0, 0, 10, 10, devices.ORANGE)
	hx.FillRectangleToBuf(30, 0, 10, 10, devices.YELLOW)
	hx.FillRectangleToBuf(200, 0, 10, 10, devices.RED)
	hx.FillRectangleToBuf(10, 200, 10, 10, devices.BLUE)

	hx.DrawVLineToBuf(20, 0, 10, devices.GREEN)

	hx.DrawPixelToBuf(0, 0, devices.WHITE)
	hx.DrawPixelToBuf(hx.Width, 0, devices.WHITE)
	hx.DrawPixelToBuf(0, hx.Height, devices.WHITE)
	hx.DrawPixelToBuf(hx.Width, hx.Height, devices.WHITE)

	t1 := time.Now()

	hx.Blit()

	elapsed := time.Since(t1)

	fmt.Printf("blit took: (%f)ms\n", float32(elapsed.Nanoseconds())/1000000.0)
}

func bufferedTest2(hx *hx8357.HX8357D) {
	// hx.FillScreen(devices.GREY)
	hx.FillRectangle(0, 0, hx.Width, 20, devices.GREY)

	paused := false
	step := false
	framePeriod := time.Duration(time.Millisecond * 17)
	ran := rand.New(rand.NewSource(99))

	w := uint16(10)
	l := w / 2
	x := int(w - l)
	y := int(w - l)
	d := 1

	const squaresTot = 255
	squares := make([]colorSquare, squaresTot)

	noiseDelay := 1000
	noiseDelayCnt := 0

	yx := int(w - l)
	yy := int(w - l)
	clearColor := devices.GREY

	for !quit {
		quit = true
		switch keyPressed {
		case '1':
			clearColor = devices.GREY
			keyPressed = 0
		case '2':
			clearColor = devices.LightGREY
			keyPressed = 0
		case '3':
			clearColor = devices.ORANGE
			keyPressed = 0
		case '4':
			clearColor = devices.WHITE
			keyPressed = 0
		case 'p':
			paused = !paused
			if paused {
				// printfTb(3, 5, term.ColorWhite, term.ColorBlack, "Paused")
			} else {
				// printfTb(3, 5, term.ColorWhite, term.ColorBlack, "      ")
			}
			term.Flush()
			keyPressed = 0
		case 's':
			step = true
			keyPressed = 0
		}

		if paused {
			if !step {
				time.Sleep(framePeriod)
				continue
			} else {
				step = false
			}
		}

		hx.FillScreenToBuf(clearColor)

		// ------------ Render BEGIN ------------------
		// t1 := time.Now()

		if noiseDelayCnt > noiseDelay {
			for i := 0; i < squaresTot; i++ {
				squares[i].x = uint16(ran.Float32() * 127)
				squares[i].y = uint16(ran.Float32() * 127)
				squares[i].c = devices.RGBtoRGB565(int(ran.Float32()*255), int(ran.Float32()*255), int(ran.Float32()*255))

				// c := devices.RGBtoRGB565(int(ran.Float32()*255), int(ran.Float32()*255), int(ran.Float32()*255))
				// ix := int(ran.Float32() * 127)
				// iy := int(ran.Float32() * 127)
				// st.FillRectangleToBuf(byte(ix), byte(iy), 4, 4, c)
			}
			noiseDelayCnt = 0
		} else {
			noiseDelayCnt += 10
		}

		for i := 0; i < squaresTot; i++ {
			hx.FillRectangleToBuf(squares[i].x, squares[i].y, 4, 4, squares[i].c)
		}

		hx.DrawVLineToBuf(70, 10, 50, devices.CYAN)
		hx.DrawHLineToBuf(45, 35, 50, devices.MAGENTA)

		hx.FillRectangleToBuf(uint16(x), uint16(y), w, w, devices.ORANGE)
		hx.FillRectangleToBuf(uint16(yx), uint16(yy), w, w, devices.YELLOW)

		hx.Blit()
		// ------------ Render END ------------------

		x += d

		if x > int(128-w) {
			d = -1
			y += 12
			x = int(128 - w)
		} else if x < 0 {
			d = 1
			y += 3
		}

		yy += d
		if yy > int(128-w) {
			d = -1
			y += 12
			yy = int(128 - w)
		} else if yy < 0 {
			d = 1
			y += 3
		}

		if y > int(hx.Height) {
			y = int(w - l)
			x = int(w - l)
			yx = int(w - l)
			yy = int(w - l)
		}

		// t2 := time.Now()
		// elapsed := t2.Sub(t1)

		// if elapsed >= framePeriod {
		// 	sleep := framePeriod - elapsed
		// 	printf_tb(3, 8, term.ColorWhite, term.ColorBlack, "Sleeping for (%f)ms", sleep.Seconds()*1000)
		// 	term.Flush()

		// 	time.Sleep(sleep)
		// }

		// Note: Of course termbox uses cpu time too but I am not tracking it so the timing will
		// be off by a few microseconds.
		// printfTb(3, 0, term.ColorWhite, term.ColorBlack, "Frame time (%f)ms\n", elapsed.Seconds()*1000)
		// term.Flush()

		// step = false
	}

	log.Println("done.")
}

func nonBufferedTest(hx *hx8357.HX8357D) {
	// hx.FillScreen(devices.GREY)
	l := uint16(5)

	x := uint16(0)
	y := uint16(0)
	px := uint16(0)
	py := uint16(0)
	d := int(l)
	hd := int(l)

	for {
		if quit {
			break
		}

		x = uint16(int(x) + d)
		if x >= hx.Width-l {
			d = -int(l)
			x = hx.Width - l
			y = uint16(int(y) + hd)
		} else if x <= 0 {
			d = int(l)
			x = l
			y = uint16(int(y) + hd)
		}

		if y > hx.Height-l {
			hd = -int(l)
			y = hx.Height - l
		} else if y <= 0 {
			hd = int(l)
			y = l
		}

		time.Sleep(time.Millisecond * 50)
		hx.FillRectangle(px, py, l, l, devices.GREY)
		hx.FillRectangle(x, y, l, l, devices.ORANGE)
		px = x
		py = y
	}
}

func nonBufferedTest2(hx *hx8357.HX8357D) {
	// hx.FillScreen(devices.BLACK)
	ran := rand.New(rand.NewSource(99))

	for i := 0; i < 10000; i++ {
		if quit {
			break
		}
		x := uint16(ran.Float32() * float32(hx.Width))
		y := uint16(ran.Float32() * float32(hx.Height))
		c := devices.RGBtoRGB565(int(ran.Float32()*255), int(ran.Float32()*255), int(ran.Float32()*255))

		hx.FillRectangle(x, y, 5, 5, c)
	}
}

func nonBufferedTest3(hx *hx8357.HX8357D) {
	// printf_tb(3, 10, term.ColorWhite, term.ColorBlack, "Filling screen")
	log.Println("Filling screen")
	t1 := time.Now()
	// hx.FillScreen(devices.ORANGE)
	// hx.FillScreen(devices.BLACK)
	// hx.FillScreen(devices.GREY)
	// t2 := time.Now()
	// hx.FillRectangle(0, 0, hx.Width, 20, devices.ORANGE)

	elapsed := time.Since(t1)
	// printf_tb(3, 11, term.ColorWhite, term.ColorBlack, "Frame time (%f)ms\n", elapsed.Seconds()*1000)
	log.Printf("Time to fill screen (%f)s\n", elapsed.Seconds())

	// log.Println("Filling rectangle")
	w := uint16(128)
	// l := w / 2
	// x := byte(w - l)
	// y := byte(w - l)
	hx.FillRectangle(0, 0, w, w, devices.BLUE)

	hx.DrawPixel(0, 0, devices.WHITE)

	hx.DrawPixel(0, hx.Height-1, devices.WHITE)
	hx.DrawPixel(hx.Width-1, hx.Height-1, devices.WHITE)
	hx.DrawPixel(hx.Width-1, 0, devices.WHITE)

	// hx.FillRectangle(50, 50, w, w, devices.BLUE)

	// for i := 32; i < 64; i++ {
	// 	st.DrawPixel(byte(i), byte(i), devices.WHITE)
	// }

	// st.DrawFastHLine(5, 0, 10, devices.RED)

	// st.DrawFastHLine(5, 127, 10, devices.BLUE)

	// st.DrawPixel(0, 0, devices.WHITE)
	// st.DrawPixel(0, st.Height-1, devices.WHITE)
	// st.DrawPixel(st.Width-1, 0, devices.WHITE)
	// st.DrawPixel(st.Width-1, st.Height-1, devices.WHITE)
}

// ---------------------------------------------------------------------------
// Termbox helpers
// ---------------------------------------------------------------------------

func printTb(x, y int, fg, bg term.Attribute, msg string) {
	for _, c := range msg {
		term.SetCell(x, y, c, fg, bg)
		x++
	}
}

func printfTb(x, y int, fg, bg term.Attribute, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	printTb(x, y, fg, bg, s)
}
