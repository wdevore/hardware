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

	texture := surface.NewSurface(st.Width, st.Height, colorOrder)

	runIt(st, texture)

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

func runIt(st *st7735.ST7735S, texture *surface.Surface) {
	frameTimeTxt := surface.NewText(texture)
	sleepTxt := surface.NewText(texture)

	fmt.Println("Running...")
	ran := rand.New(rand.NewSource(99))

	var elapsed = time.Duration(0)
	texture.SetClearColor(surface.DarkGREY)

	mx := 16
	my := 20
	t := 8
	l := 8
	q := 150
	sleep := 200

	for !quit {
		switch key {
		case keyboard.KeyArrowUp:
			sleep += 10
			key = 0
		case keyboard.KeyArrowDown:
			sleep -= 10
			key = 0
		}

		texture.Clear()

		// ------------ Render BEGIN ------------------
		t1 := time.Now()

		texture.SetColor(surface.ORANGE)
		for i := 0; i < q; i++ {
			// Generate x,y coordinates on modulus
			x := int(ran.Float32()*127) % mx

			// Generate a row
			y := int(ran.Float32()*159) % my
			// fmt.Printf("%d,%d\n", x, y)
			texture.DrawFilledRectangle(x*t, y*t, l, l)
		}

		texture.SetColor(surface.DarkerGREY)
		for col := 0; col < st.Width; col += 8 {
			texture.DrawVLine(col, 0, st.Height)
		}
		texture.DrawVLine(st.Width-1, 0, st.Height)

		for row := 0; row < st.Height; row += 8 {
			texture.DrawHLine(0, row, st.Width)
		}
		texture.DrawHLine(0, st.Height-1, st.Width)

		txt := fmt.Sprintf("%d", sleep)
		sleepTxt.DrawText(50, int(st.Height-10), txt, surface.WHITE, surface.GREY, true)

		time.Sleep(time.Millisecond * time.Duration(sleep))
		txt = fmt.Sprintf("%3.1f", float32(elapsed)/1000000.0)
		frameTimeTxt.DrawText(5, int(st.Height-10), txt, surface.WHITE, surface.GREY, false)

		st.Blit(texture.Buffer())
		elapsed = time.Since(t1)
		// ------------ Render END ------------------
	}

}
