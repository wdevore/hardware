package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wdevore/hardware/ftdi/devices/max7219"
)

// Tests a single 8x8 led matrix

var quit bool

func main() {
	quit = false

	matrix := max.NewMatrix1x1(200000, 1)

	if matrix == nil {
		panic("Could not create matrix")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(m max.IMatrix) {
		<-c
		quit = true
		log.Println("\nReceived ctrl-C, closing matrix.")
	}(matrix)

	defer exitProg(matrix)

	err := matrix.Initialize()

	if err != nil {
		panic(err)
	}

	matrix.ClearDevice()

	// testActivateTestMode(matrix)
	// testDisplayBlink(matrix)
	// testCornerPixels(matrix)
	// testCross(matrix)
	// testVerticalScanBar(matrix)
	// testMatrix(matrix)
	testThinking(matrix)
}

func exitProg(m max.IMatrix) {
	log.Println("Closing devices")
	err := m.Close()
	if err != nil {
		log.Println("\n Failed to close matrix")
		os.Exit(-1)
	}
	os.Exit(0)
}

func testThinking(m max.IMatrix) {
	ran := rand.New(rand.NewSource(99))

	for {
		if quit {
			break
		}

		// Generate a column
		col := int(ran.Float32() * 8)

		// Generate a row
		row := int(ran.Float32() * 8)

		// Random state
		var bit uint8
		if ran.Float32() > 0.5 {
			bit = 1
		} else {
			bit = 0
		}

		m.ChangePixel(col, row, bit)

		m.UpdateDisplay()

		time.Sleep(time.Millisecond * 10)
	}

	log.Println("Done.")
}

func testMatrix(m max.IMatrix) {
	ran := rand.New(rand.NewSource(99))
	m.ClearDisplay()

	for {
		if quit {
			break
		}

		// Set a random pixel on row 0
		rcol := int(ran.Float32() * 8)

		m.SetPixel(rcol, 0)

		m.UpdateDisplay()

		time.Sleep(time.Millisecond * 50)

		// scroll downwards
		for row := 7; row > 0; row-- {
			// copy row-1 into row
			for col := 0; col < 8; col++ {
				m.ChangePixel(col, row, m.GetPixel(col, row-1))
			}
		}

		// clear row 0
		m.ClearPixel(rcol, 0)
	}
}

func testVerticalScanBar(m max.IMatrix) {
	d := 1
	c := 0

	for {
		if quit {
			break
		}

		m.ClearDisplay()
		if c > 7 {
			c = 6
			d = -1
		} else if c < 0 {
			c = 1
			d = 1
		}

		m.DrawVLine(c, 0, m.GetHeight())
		// m.PrintBuf()

		c += d

		m.UpdateDisplay()

		time.Sleep(time.Millisecond * 50)
	}

	log.Println("Done.")
}

func testCross(m max.IMatrix) {
	m.ClearDisplay()

	// [0 0 1 0 0 0 1 0]
	// [0 0 1 0 0 0 1 0]
	// [1 1 1 1 1 1 1 1]
	// [0 0 1 0 0 0 1 0]
	// [0 0 1 0 0 0 1 0]
	// [0 0 1 0 0 0 1 0]
	// [1 1 1 1 1 1 1 1]
	// [0 0 1 0 0 0 1 0]

	m.DrawHLine(0, 2, m.GetWidth())
	m.DrawHLine(0, m.GetHeight()-2, m.GetWidth())

	m.DrawVLine(2, 0, m.GetHeight())
	m.DrawVLine(m.GetWidth()-2, 0, m.GetHeight())

	// m.PrintBuf()
	m.UpdateDisplay()

	fmt.Println("Done.")
}

func testActivateTestMode(m max.IMatrix) {
	m.ActivateTestMode(false)
}

func testDisplayBlink(m max.IMatrix) {
	blink := true
	for {
		if quit {
			break
		}
		m.ActivateTestMode(blink)
		blink = !blink
		time.Sleep(time.Millisecond * 200)
	}
}

func testCornerPixels(m max.IMatrix) {
	m.ClearDisplay()

	m.SetPixel(0, 0)
	m.SetPixel(m.GetWidth()-1, 0)
	m.SetPixel(0, m.GetHeight()-1)
	m.SetPixel(m.GetWidth()-1, m.GetHeight()-1)

	m.UpdateDisplay()

	fmt.Println("Done.")
}
