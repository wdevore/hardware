package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/wdevore/hardware/examples/st7735S/surface"
	"github.com/wdevore/hardware/ftdi"
	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/ftdi/devices/st7735"
	"github.com/wdevore/hardware/gpio"
)

var quit = false
var keyPressed rune
var colorOrder devices.ColorOrder = devices.RGBOrder
var key keyboard.Key

func main() {
	// log.SetFlags(0)
	// log.SetOutput(ioutil.Discard)

	err := keyboard.Open()
	if err != nil {
		panic(err)
	}

	defer keyboard.Close()

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

	err = st.Initialize(0x0403, 0x06014, 0, gpio.DefaultPin, devices.OrientationDefault, colorOrder)
	st.SetConstantCSAssert(true)

	if err != nil {
		log.Println("Example: Failed to initialize ST7735R component")
		log.Fatal(err)
		os.Exit(-1)
	}

	defer st.Close()

	// nonBufferedTest(st)
	bufferedTest(st)

	fmt.Println("Press 'Escape' to exit...")
	// bufio.NewReader(os.Stdin).ReadBytes('\n')

	for !quit {
		time.Sleep(time.Millisecond * 200)
	}

	// st.LightOn(false)
	// st.DisplayOn(false)
}

// A Goroutine so that the pollEvent doesn't block on the main rendering thread.
func keyScan() {
	var err error

	for {
		keyPressed, key, err = keyboard.GetKey()
		// fmt.Printf("key: %d (%016b), keyPressed: %d\n", key, key, keyPressed)
		if err != nil {
			quit = true
			fmt.Println(err)
			break
		}

		if key == keyboard.KeyEsc {
			// fmt.Println("Escape detected")
			quit = true
			break
		}
	}
	keyboard.Close()
}

func exitProg(st *st7735.ST7735R) {
	log.Println("Closing devices")
	err := st.Close()
	if err != nil {
		log.Println("\n Failed to close FTDI component")
		os.Exit(-1)
	}
	os.Exit(0)
}

type colorSquare struct {
	x, y int
	c    uint16
}

func bufferedTest(st *st7735.ST7735R) {
	texture := surface.NewSurface(st.Width, st.Height, colorOrder)
	frameTimeTxt := surface.NewText(texture)

	paused := false
	step := false
	framePeriod := time.Duration(time.Millisecond * 17)
	ran := rand.New(rand.NewSource(99))

	w := int(10)
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
	var elapsed = time.Duration(0)

	for !quit {
		switch keyPressed {
		case 49: // 1
			texture.SetClearColor(surface.BLACK)
			keyPressed = 0
		case 50:
			texture.SetClearColor(surface.LightGREY)
			keyPressed = 0
		case 51:
			texture.SetClearColor(surface.ORANGE)
			keyPressed = 0
		case 52:
			texture.SetClearColor(surface.WHITE)
			keyPressed = 0
		case 108:
			// lightOn = !lightOn
			// st.LightOn(lightOn)
			keyPressed = 0
			// case 'p':
			// paused = !paused
			// if paused {
			// 	printf_tb(3, 5, term.ColorWhite, term.ColorBlack, "Paused")
			// } else {
			// 	printf_tb(3, 5, term.ColorWhite, term.ColorBlack, "      ")
			// }
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

		texture.Clear()

		// ------------ Render BEGIN ------------------
		t1 := time.Now()

		if noiseDelayCnt > noiseDelay {
			for i := 0; i < squaresTot; i++ {
				squares[i].x = int(ran.Float32() * 127)
				squares[i].y = int(ran.Float32() * 127)
				squares[i].c = surface.RGBto565(int(ran.Float32()*255), int(ran.Float32()*255), int(ran.Float32()*255), colorOrder)

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
			texture.SetColor(squares[i].c)
			texture.DrawFilledRectangle(squares[i].x, squares[i].y, 4, 4)
		}

		texture.SetColor(surface.CYAN)
		texture.DrawVLine(70, 10, 50)
		texture.SetColor(surface.MAGENTA)
		texture.DrawHLine(45, 35, 50)

		texture.SetColor(surface.ORANGE)
		texture.DrawFilledRectangle(x, y, w, w)

		texture.SetColor(surface.YELLOW)
		texture.DrawFilledRectangle(yx, yy, w, w)

		txt := fmt.Sprintf("%03.1f", float32(elapsed)/1000000.0)
		frameTimeTxt.DrawText(5, int(st.Height-10), txt, surface.WHITE, surface.GREY)

		st.Blit(texture.Buffer())
		elapsed = time.Since(t1)
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
			y = int(w - l)
			x = int(w - l)
			yx = int(w - l)
			yy = int(w - l)
		}

		// if elapsed >= framePeriod {
		// 	sleep := framePeriod - elapsed
		// 	printf_tb(3, 8, term.ColorWhite, term.ColorBlack, "Sleeping for (%f)ms", sleep.Seconds()*1000)
		// 	term.Flush()

		// 	time.Sleep(sleep)
		// }

		// step = false
	}

}

func nonBufferedTest(st *st7735.ST7735R) {
	log.Println("Filling screen")
	t1 := time.Now()
	// st.FillScreen(devices.ORANGE)
	// st.FillScreen(devices.BLACK)
	st.FillScreen(surface.GREY)
	t2 := time.Now()

	elapsed := t2.Sub(t1)
	log.Printf("Time to fill screen (%f)s\n", elapsed.Seconds())

	log.Println("Filling rectangle")
	w := byte(10)
	l := w / 2
	x := byte(w - l)
	y := byte(w - l)
	st.FillRectangle(x, y, w, w, surface.BLUE)

	st.FillRectangle(64, 80, w, w, surface.ORANGE)

	for i := 32; i < 64; i++ {
		st.DrawPixel(byte(i), byte(i), surface.WHITE)
	}

	st.DrawFastHLine(5, 0, 10, surface.RED)

	st.DrawFastHLine(5, 127, 10, surface.BLUE)

	st.DrawPixel(0, 0, surface.WHITE)
	st.DrawPixel(0, byte(st.Height-1), surface.WHITE)
	st.DrawPixel(byte(st.Width-1), 0, surface.WHITE)
	st.DrawPixel(byte(st.Width-1), byte(st.Height-1), surface.WHITE)
}
