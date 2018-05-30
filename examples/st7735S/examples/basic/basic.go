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
var key keyboard.Key
var lightOn = true
var colorOrder devices.ColorOrder = devices.RGBOrder

func main() {

	err := keyboard.Open()
	if err != nil {
		panic(err)
	}

	defer keyboard.Close()

	go keyScan()

	// log.SetFlags(0)
	// log.SetOutput(ioutil.Discard)

	st := st7735.NewST7735S(ftdi.D4, ftdi.D5, devices.GreenTab, devices.D160x128)

	if st == nil {
		keyboard.Close()
		panic("Unable to create ST7735S component")
	}

	fmt.Println("Initializing device...")

	err = st.Initialize(0x0403, 0x06014, 0, gpio.DefaultPin, devices.OrientationDefault, colorOrder)
	st.SetConstantCSAssert(true)

	if err != nil {
		keyboard.Close()
		log.Println("Example: Failed to initialize ST7735R component")
		log.Fatal(err)
		os.Exit(-1)
	}

	defer st.Close()

	fmt.Printf("Display dimensions: %d x %d\n", st.Width, st.Height)

	st.DisplayOn(true)

	// nonBufferedTest(st)
	bufferedTest(st)
	// drawText(0, 0, "", devices.WHITE, st)

	fmt.Println("Press 'Escape' to exit...")
	// bufio.NewReader(os.Stdin).ReadBytes('\n')

	for !quit {
		time.Sleep(time.Millisecond * 200)
	}

	st.LightOn(false)
	st.DisplayOn(false)
}

// A Goroutine so that the waiting doesn't block on the main rendering thread.
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

func exitProg(st *st7735.ST7735S) {
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

func bufferedTest(st *st7735.ST7735S) {
	texture := surface.NewSurface(st.Width, st.Height, colorOrder)
	frameTimeTxt := surface.NewText(texture)

	fmt.Println("Starting Buffered test")
	// paused := false
	// step := false
	// framePeriod := time.Duration(time.Millisecond * 17)
	ran := rand.New(rand.NewSource(99))

	w := int(10)
	l := w / 2
	x := int(w - l)
	y := int(w - l)
	d := 1

	const squaresTot = 255
	squares := make([]colorSquare, squaresTot)

	noiseDelay := 500
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
			lightOn = !lightOn
			st.LightOn(lightOn)
			keyPressed = 0

			// case 'p':
			// 	paused = !paused
			// 	if paused {
			// 	} else {
			// 	}
			// 	term.Flush()
			// 	keyPressed = 0
			// case 's':
			// 	step = true
			// 	keyPressed = 0
		}

		switch key {
		case keyboard.KeyArrowLeft:
			fmt.Println("left arrow")
			key = 0
		case keyboard.KeyArrowRight:
			fmt.Println("right arrow")
			key = 0
		}

		// if paused {
		// 	if !step {
		// 		time.Sleep(framePeriod)
		// 		continue
		// 	} else {
		// 		step = false
		// 	}
		// }

		texture.Clear()

		// ------------ Render BEGIN ------------------
		t1 := time.Now()

		if noiseDelayCnt > noiseDelay {
			for i := 0; i < squaresTot; i++ {
				squares[i].x = int(ran.Float32() * float32(st.Width-1))
				squares[i].y = int(ran.Float32() * float32(st.Height-1))
				squares[i].c = surface.RGBto565(int(ran.Float32()*255), int(ran.Float32()*255), int(ran.Float32()*255), devices.BGROrder)

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
			texture.DrawFilledRectangle(squares[i].x, squares[i].y, 8, 8)
		}

		texture.SetColor(surface.CYAN)
		texture.DrawVLine(70, 10, 50)
		texture.SetColor(surface.MAGENTA)
		texture.DrawHLine(45, 35, 50)

		texture.SetColor(surface.ORANGE)
		texture.DrawFilledRectangle(x, y, w, w)

		texture.SetColor(surface.YELLOW)
		texture.DrawFilledRectangle(yx, yy, w, w)

		texture.SetPixelWithColor(0, 0, surface.WHITE)
		texture.SetPixelWithColor(0, st.Height-1, surface.WHITE)
		texture.SetPixelWithColor(st.Width-1, 0, surface.WHITE)
		texture.SetPixelWithColor(st.Width-1, st.Height-1, surface.WHITE)

		txt := fmt.Sprintf("%03.1f", float32(elapsed)/1000000.0)
		frameTimeTxt.DrawText(5, int(st.Height-10), txt, surface.WHITE, surface.GREY, false)

		st.Blit(texture.Buffer())
		elapsed = time.Since(t1)
		// fmt.Printf("blit: %f\n", float32(elapsed)/1000000.0)
		// ------------ Render END ------------------

		x += d

		if x > int(st.Width-w) {
			d = -1
			y += 12
			x = int(st.Width - w)
		} else if x < 0 {
			d = 1
			y += 3
		}

		yy += d
		if yy > int(st.Width-w) {
			d = -1
			y += 12
			yy = int(st.Width - w)
		} else if yy < 0 {
			d = 1
			y += 3
		}

		if y > st.Height {
			y = w - l
			x = w - l
			yx = w - l
			yy = w - l
		}

		// if elapsed >= framePeriod {
		// 	sleep := framePeriod - elapsed
		// 	time.Sleep(sleep)
		// }

		// step = false
	}

}

func nonBufferedTest(st *st7735.ST7735S) {
	fmt.Println("Starting NON buffered test")
	t1 := time.Now()
	st.FillScreen(surface.ORANGE)
	// st.FillScreen(devices.BLACK)
	// st.FillScreen(devices.GREY)
	t2 := time.Now()

	elapsed := t2.Sub(t1)
	fmt.Printf("Time to fill screen (%f)s\n", elapsed.Seconds())

	fmt.Println("Filling rectangle")
	w := byte(10)
	l := w / 2
	x := byte(w - l)
	y := byte(w - l)
	st.FillRectangle(x, y, w, w, surface.BLUE)

	st.FillRectangle(64, 80, w, w, surface.GREEN)

	st.FillRectangle(118, 140, w, w, surface.MAGENTA)

	st.FillRectangle(50, 140, w, w, surface.CYAN)

	// for i := 32; i < 64; i++ {
	// 	st.DrawPixel(byte(i), byte(i), devices.WHITE)
	// }

	// st.DrawFastHLine(5, 0, 10, devices.RED)

	// st.DrawFastHLine(5, 127, 10, devices.BLUE)

	st.DrawPixel(0, 0, surface.WHITE)
	st.DrawPixel(0, byte(st.Height-1), surface.RED)
	st.DrawPixel(byte(st.Width-1), 0, surface.GREEN)
	st.DrawPixel(byte(st.Width-1), byte(st.Height-1), surface.BLUE)

	fmt.Println("Done.")
}
