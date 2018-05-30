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
	"github.com/wdevore/hardware/ftdi/devices/ssd1351"
	"github.com/wdevore/hardware/gpio"
)

var quit = false
var keyPressed rune

func main() {
	// log.SetFlags(0)
	// log.SetOutput(ioutil.Discard)

	log.Println("Starting...")

	ssd := ssd1351.NewSSD1351(ftdi.D5, ftdi.D4, devices.D128x128)

	// Note this doesn't work when termbox-go is used
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("\nReceived ctrl-C, quiting.")
		quit = true
	}()

	if ssd == nil {
		panic("Unable to create SSD1351 component")
	}

	log.Println("Initializing device...")

	// Max clk allowed by ssd1351 is 20MHz and I recommend an actual
	// 5V supply, breakout boards don't seem to source well enough.
	err := ssd.Initialize(0x0403, 0x06014, 20000000, gpio.DefaultPin)
	ssd.SetConstantCSAssert(true)

	if err != nil {
		log.Println("Example: Failed to initialize SSD1351 component")
		log.Fatal(err)
		os.Exit(-1)
	}

	defer exitProg(ssd)

	log.Printf("Display dimensions: %d x %d\n", ssd.Width, ssd.Height)
	log.Println("Beginning test.")

	// nonBufferedTest(ssd)
	// bufferedTest(ssd)
	bufferedThinking1(ssd)

	log.Println("Done.")

	log.Println("Press 'Enter' to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func exitProg(ssd *ssd1351.SSD1351) {
	log.Println("Closing devices")
	err := ssd.Close()
	if err != nil {
		log.Println("\n Failed to close FTDI component")
		os.Exit(-1)
	}
	os.Exit(0)
}

type colorSquare struct {
	x, y uint16
	c    uint16
}

func nonBufferedTest(ssd *ssd1351.SSD1351) {
	log.Println("Filling screen")
	// t1 := time.Now()
	// hx.FillScreen(devices.ORANGE)
	// hx.FillScreen(devices.BLACK)
	// hx.FillScreen(devices.GREY)
	// t2 := time.Now()
	// hx.FillRectangle(0, 0, hx.Width, 20, devices.ORANGE)

	// elapsed := time.Since(t1)
	// log.Printf("Time to fill screen (%f)s\n", elapsed.Seconds())

	// log.Println("Filling rectangle")
	// w := uint8(128)
	// l := w / 2
	// x := byte(w - l)
	// y := byte(w - l)
	// ssd.FillRectangle(0, 0, w, w, devices.GREY)
	r := uint8(0)
	for c := 0; c < 128; c++ {
		cc := devices.RGBtoRGB565(c, c/3, 0)
		ssd.DrawFastHLine(0, r, 128, cc)
		// fmt.Printf("%08b\n", cc)
		r++
	}

	ssd.DrawPixel(0, 0, devices.WHITE)
	ssd.DrawPixel(0, ssd.Height-1, devices.WHITE)
	ssd.DrawPixel(ssd.Width-1, ssd.Height-1, devices.WHITE)
	ssd.DrawPixel(ssd.Width-1, 0, devices.WHITE)

	// ssd.FillRectangle(50, 50, w, w, devices.BLUE)

	// for i := 32; i < 64; i++ {
	// 	st.DrawPixel(byte(i), byte(i), devices.WHITE)
	// }

	// st.DrawFastHLine(5, 127, 10, devices.BLUE)

	// st.DrawPixel(0, 0, devices.WHITE)
	// st.DrawPixel(0, st.Height-1, devices.WHITE)
	// st.DrawPixel(st.Width-1, 0, devices.WHITE)
	// st.DrawPixel(st.Width-1, st.Height-1, devices.WHITE)
}

func bufferedTest(ssd *ssd1351.SSD1351) {
	fmt.Println("Starting buffered test...")
	// hx.FillRectangle(0, 0, hx.Width, 50, devices.GREY)

	y := uint8(0)
	for {
		// ssd.FillScreenToBuf(devices.GREY)
		r := uint8(0)
		for c := 0; c < 128; c++ {
			cc := devices.RGBtoRGB565(c, c/3, 0)
			ssd.DrawHLineToBuf(0, r, 128, cc)
			// fmt.Printf("%08b\n", cc)
			r++
		}

		// ssd.FillScreenToBuf(devices.BLACK)

		ssd.FillRectangleToBuf(30, 0, 10, 10, devices.YELLOW)
		ssd.FillRectangleToBuf(70, 0, 10, 10, devices.RED)
		ssd.FillRectangleToBuf(10, 70, 10, 10, devices.BLUE)

		if quit {
			break
		}
		y += 3
		if y > 128-10-3 {
			y = 0
		}
		ssd.FillRectangleToBuf(0, y, 10, 10, devices.ORANGE)
		ssd.Blit()
	}

	// t1 := time.Now()

	// elapsed := time.Since(t1)

	// fmt.Printf("blit took: (%f)ms\n", float32(elapsed.Nanoseconds())/1000000.0)
}

func bufferedThinking1(ssd *ssd1351.SSD1351) {
	// The pixels are 8x8
	ran := rand.New(rand.NewSource(99))

	// 16x16 = %8, *16, L16
	// m := uint8(8)
	// t := uint8(16)
	// l := uint8(16)
	// q := 35
	// 8x8 = %16, *8, L8
	m := uint8(16)
	t := uint8(8)
	l := uint8(8)
	q := 100
	// 4x4 = %32, *4, L4
	// m := uint8(32)
	// t := uint8(4)
	// l := uint8(4)
	// q:=150

	for {
		ssd.FillScreenToBuf(devices.DarkGREY)

		if quit {
			break
		}

		for i := 0; i < q; i++ {
			// Generate x,y coordinates on modulus
			x := uint8(ran.Float32()*127) % m

			// Generate a row
			y := uint8(ran.Float32()*127) % m
			// fmt.Printf("%d,%d\n", x, y)
			ssd.FillRectangleToBuf(x*t, y*t, l, l, devices.ORANGE)
		}

		for c := uint8(0); c < 128+l; c += l {
			ssd.DrawVLineToBuf(c, 0, ssd.Height, devices.DarkerGREY)
		}

		for c := uint8(0); c < 128+l; c += l {
			ssd.DrawHLineToBuf(0, c, ssd.Width, devices.DarkerGREY)
		}

		time.Sleep(time.Millisecond * 300)

		ssd.Blit()
	}
}
