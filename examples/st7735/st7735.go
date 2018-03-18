package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/wdevore/hardware/ftdi"
	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/ftdi/devices/st7735"
	"github.com/wdevore/hardware/gpio"

	term "github.com/nsf/termbox-go"
)

var quit = false
var keyPressed rune

func main() {
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)

	terr := term.Init()
	if terr != nil {
		panic("Could not initialize termbox.")
	}

	defer term.Close()
	term.SetInputMode(term.InputEsc)

	term.Clear(term.ColorDefault, term.ColorDefault)
	printf_tb(3, 1, term.ColorWhite, term.ColorBlack, "Starting...")
	term.Flush()

	go keyScan()

	// Per Adafruit's FT232H breakout board configured as Hardware SPI:
	// https://learn.adafruit.com/adafruit-ft232h-breakout?view=all

	// D0 - Clock signal output.  This line can be configured as a clock that runs at speeds between ~450Hz to 30Mhz.
	// D1 - Serial data output.  This is for outputting a serial signal, like the MOSI line in a SPI connection.
	// D2 - Serial data input.  This is for reading a serial signal, like the MISO line in a SPI connection.
	// --> D3 - Serial select signal.  This is a chip select or chip enable signal to tell a connected device that the FT232H is ready to talk to it.

	st := st7735.NewST7735R(ftdi.D5, ftdi.D4, devices.GreenTab, devices.D128x128)

	// Note this doesn't work when termbox-go is used
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// go func() {
	// 	<-c
	// 	log.Println("\nReceived ctrl-C, quiting.")
	// 	quit = true
	// }()

	if st == nil {
		panic("Unable to create ST7735R component")
	}

	printf_tb(3, 1, term.ColorWhite, term.ColorBlack, "Initializing device...")
	term.Flush()

	err := st.Initialize(0x0403, 0x06014, 30000000, gpio.DefaultPin, devices.OrientationDefault)
	// st.SetConstantCSAssert(false)

	if err != nil {
		log.Println("Example: Failed to initialize ST7735R component")
		log.Fatal(err)
		os.Exit(-1)
	}

	defer st.Close()

	term.Clear(term.ColorDefault, term.ColorDefault)

	printf_tb(3, 4, term.ColorWhite, term.ColorBlack, "Display dimensions: %d x %d\n", st.Width, st.Height)
	printf_tb(3, 1, term.ColorWhite, term.ColorBlack, "Beginning test.")

	term.Flush()

	// nonBufferedTest(st)
	bufferedTest(st)

	printf_tb(3, 1, term.ColorWhite, term.ColorBlack, "Done.")
	term.Flush()

	// log.Println("Press 'Enter' to continue...")
	// bufio.NewReader(os.Stdin).ReadBytes('\n')
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
				printf_tb(3, 7, term.ColorWhite, term.ColorBlack, "Key: %c\n", ev.Ch)
				term.Flush()
			}
		case term.EventError:
			fmt.Printf("Term err (%v)\n", ev.Err)
		}

	}
}
func exitProg(st *st7735.ST7735R) {
	log.Println("Closing devices")
	err := st.Close()
	if err != nil {
		log.Println("\n Failed to close FTDI component")
		os.Exit(-1)
	}
	term.Close()
	os.Exit(0)
}

type colorSquare struct {
	x, y byte
	c    uint16
}

func bufferedTest(st *st7735.ST7735R) {
	paused := false
	step := false
	framePeriod := time.Duration(time.Millisecond * 17)
	ran := rand.New(rand.NewSource(99))

	w := byte(10)
	l := w / 2
	x := int(w - l)
	y := byte(w - l)
	d := 1

	const squaresTot = 255
	squares := make([]colorSquare, squaresTot)

	noiseDelay := 1000
	noiseDelayCnt := 0

	yx := int(w - l)
	yy := int(w - l)
	clearColor := devices.GREY

	for !quit {
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
				printf_tb(3, 5, term.ColorWhite, term.ColorBlack, "Paused")
			} else {
				printf_tb(3, 5, term.ColorWhite, term.ColorBlack, "      ")
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

		st.FillScreenToBuf(clearColor)

		// ------------ Render BEGIN ------------------
		t1 := time.Now()

		if noiseDelayCnt > noiseDelay {
			for i := 0; i < squaresTot; i++ {
				squares[i].x = byte(ran.Float32() * 127)
				squares[i].y = byte(ran.Float32() * 127)
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
			st.FillRectangleToBuf(squares[i].x, squares[i].y, 4, 4, squares[i].c)
		}

		st.DrawVLineToBuf(70, 10, 50, devices.CYAN)
		st.DrawHLineToBuf(45, 35, 50, devices.MAGENTA)

		st.FillRectangleToBuf(byte(x), y, w, w, devices.ORANGE)
		st.FillRectangleToBuf(byte(yx), byte(yy), w, w, devices.YELLOW)

		st.Blit()
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

		if y > st.Height {
			y = byte(w - l)
			x = int(w - l)
			yx = int(w - l)
			yy = int(w - l)
		}

		t2 := time.Now()
		elapsed := t2.Sub(t1)

		// if elapsed >= framePeriod {
		// 	sleep := framePeriod - elapsed
		// 	printf_tb(3, 8, term.ColorWhite, term.ColorBlack, "Sleeping for (%f)ms", sleep.Seconds()*1000)
		// 	term.Flush()

		// 	time.Sleep(sleep)
		// }

		// Note: Of course termbox uses cpu time too but I am not tracking it so the timing will
		// be off by a few microseconds.
		printf_tb(3, 0, term.ColorWhite, term.ColorBlack, "Frame time (%f)ms\n", elapsed.Seconds()*1000)
		term.Flush()

		// step = false
	}

}

func nonBufferedTest(st *st7735.ST7735R) {
	log.Println("Filling screen")
	t1 := time.Now()
	// st.FillScreen(devices.ORANGE)
	// st.FillScreen(devices.BLACK)
	st.FillScreen(devices.GREY)
	t2 := time.Now()

	elapsed := t2.Sub(t1)
	log.Printf("Time to fill screen (%f)s\n", elapsed.Seconds())

	log.Println("Filling rectangle")
	w := byte(10)
	l := w / 2
	x := byte(w - l)
	y := byte(w - l)
	st.FillRectangle(x, y, w, w, devices.BLUE)

	st.FillRectangle(64, 80, w, w, devices.ORANGE)

	for i := 32; i < 64; i++ {
		st.DrawPixel(byte(i), byte(i), devices.WHITE)
	}

	st.DrawFastHLine(5, 0, 10, devices.RED)

	st.DrawFastHLine(5, 127, 10, devices.BLUE)

	st.DrawPixel(0, 0, devices.WHITE)
	st.DrawPixel(0, st.Height-1, devices.WHITE)
	st.DrawPixel(st.Width-1, 0, devices.WHITE)
	st.DrawPixel(st.Width-1, st.Height-1, devices.WHITE)
}

// ---------------------------------------------------------------------------
// Termbox helpers
// ---------------------------------------------------------------------------

func print_tb(x, y int, fg, bg term.Attribute, msg string) {
	for _, c := range msg {
		term.SetCell(x, y, c, fg, bg)
		x++
	}
}

func printf_tb(x, y int, fg, bg term.Attribute, format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	print_tb(x, y, fg, bg, s)
}
