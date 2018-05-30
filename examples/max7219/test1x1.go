package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/wdevore/hardware/spi"
)

// This test is different than the test_1x1 in that it is a custom
// fit for a single 8x8 led matrix

// This toggles the display test mode causing all leds to flash on and off
func superSimple(sp *spi.FtdiSPI) {
	packetOff := []byte{displayTestReg, off}
	packetOn := []byte{displayTestReg, on}

	on := true
	var err error

	for i := 0; i < 10; i++ {
		if on {
			err = sp.Write(packetOn)
		} else {
			err = sp.Write(packetOff)
		}
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Millisecond * 100)
		on = !on
	}

	log.Println("Done.")

}

func enableCol2WithPattern(sp *spi.FtdiSPI) {
	// The right most bit is the farthest away from the max chip (or Input side)
	pattern := stringToByte("10100111")

	packet := []byte{digit1Reg, pattern}

	err := sp.Write(packet)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done.")
}

func random(sp *spi.FtdiSPI) {
	ran := rand.New(rand.NewSource(99))

	for i := 0; i < 100000; i++ {
		// Generate a column
		col := byte(ran.Float32()*8 + 1)

		// Generate a row
		row := byte(ran.Float32() * 8)

		// Random state
		var bit byte
		if ran.Float32() > 0.5 {
			bit = 1
		} else {
			bit = 0
		}

		// The right most bit is the farthest away from the max chip (or Input side)
		pattern := bit << row

		packet := []byte{col, pattern}

		err := sp.Write(packet)
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Millisecond * 10)
	}

	log.Println("Done.")
}

func random2(sp *spi.FtdiSPI) {
	ran := rand.New(rand.NewSource(99))

	for i := 0; i < 100000; i++ {
		// Generate a column
		col := byte(ran.Float32()*8 + 1)

		// Generate a row
		row := byte(ran.Float32()*256 - 1)

		packet := []byte{col, row}

		err := sp.Write(packet)
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Millisecond * 50)
	}

	log.Println("Done.")
}

func random3(sp *spi.FtdiSPI) {
	ran := rand.New(rand.NewSource(99))

	for i := 0; i < 100000; i++ {
		for col := 1; col < 9; col++ {
			// Generate a row
			row := byte(ran.Float32()*256 - 1)

			packet := []byte{byte(col), row}

			err := sp.Write(packet)
			if err != nil {
				log.Fatal(err)
			}
			// Enable this sleep to create a scan effect
			// time.Sleep(time.Millisecond * 50)
		}
		// Enable this sleep to create a computer thinking effect
		time.Sleep(time.Millisecond * 250)
	}

	log.Println("Done.")
}

func displayOn(sp *spi.FtdiSPI, on uint8) {
	packetOff := []byte{displayTestReg, off}
	packetOn := []byte{displayTestReg, on}

	var err error

	if on == 1 {
		err = sp.Write(packetOn)
	} else {
		err = sp.Write(packetOff)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func clearDisplay(sp *spi.FtdiSPI) {
	// Clear display
	for c := 1; c < 9; c++ {
		packet := []byte{byte(c), 0x00}
		sp.Write(packet)
	}
}

func horizontalScanBar(sp *spi.FtdiSPI) {

	d := 1
	col := 1

	for i := 0; i < 100000; i++ {
		clearDisplay(sp)

		packet := []byte{byte(col), 0xff}

		err := sp.Write(packet)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Millisecond * 50)

		col += d
		if col > 8 {
			d = -1
			col = 0x07
		}

		if col < 1 {
			d = 1
			col = 0x02
		}
	}

	log.Println("Done.")
}

func verticalScanBar(sp *spi.FtdiSPI) {

	d := 1
	row := 0

	for i := 0; i < 100000; i++ {
		clearDisplay(sp)

		for c := 1; c < 9; c++ {
			packet := []byte{byte(c), 1 << byte(row)}

			err := sp.Write(packet)
			if err != nil {
				log.Fatal(err)
			}
		}

		time.Sleep(time.Millisecond * 50)

		row += d
		if row > 7 {
			d = -1
			row = 0x06
		}

		if row < 0 {
			d = 1
			row = 0x01
		}
	}

	log.Println("Done.")
}

// -----------------------------------------------------------
// Display buffer examples for an 8x8 display
// -----------------------------------------------------------
func setPixelTest(sp *spi.FtdiSPI) {
	clearDisplay(sp)

	clearBuf()
	// drawHorizontalLine(3, 2, 4)
	// drawVerticalLine(3, 2, 3)
	setPixel(0, 0)
	// setPixel(1, 0)
	// setPixel(7, 0)
	// setPixel(0, 7)
	// setPixel(7, 7)

	blit(sp)
}

var hod = float64(1.0)
var hvd = float64(1.0)

func dualScanBars(sp *spi.FtdiSPI) {
	vd := hvd
	vr := float64(0.0)

	hod = float64(0.7)
	hd := hod
	hr := float64(0.0)

	for i := 0; i < 100000; i++ {
		clearBuf()

		// Draw a frame for the vertical bar
		vd, vr = animateVerticalScan(vd, vr)

		// Draw a frame for the horizontal bar
		hd, hr = animateHorizontalScan(hd, hr)

		blit(sp)
		time.Sleep(time.Millisecond * 50)
	}
}

func dualScanBars2(sp *spi.FtdiSPI) {
	vd := hvd
	vr := float64(0.0)

	hd := hod
	hr := float64(0.0)
	alternate := false
	j := 0
	for i := 0; i < 100000; i++ {
		clearBuf()

		if alternate {
			// Draw a frame for the vertical bar
			vd, vr = animateVerticalScan(vd, vr)
		} else {
			// Draw a frame for the horizontal bar
			hd, hr = animateHorizontalScan(hd, hr)
		}

		j++
		if j > 8 {
			alternate = !alternate
			j = 0
		}
		blit(sp)
		time.Sleep(time.Millisecond * 50)
	}
}

func matrix(sp *spi.FtdiSPI) {
	ran := rand.New(rand.NewSource(99))
	clearBuf()

	for i := 0; i < 100000; i++ {
		// Set a random pixel on row 0
		rcol := int(ran.Float32() * 8)

		setPixel(rcol, 0)

		blit(sp)
		time.Sleep(time.Millisecond * 50)

		// scroll downwards
		for row := 7; row > 0; row-- {
			// copy row-1 into row
			for col := 0; col < 8; col++ {
				pixelBuf[row][col] = pixelBuf[row-1][col]
			}
		}

		// clear row 0
		unSetPixel(rcol, 0)
	}
}

func animateVerticalScan(currentDirection float64, row float64) (float64, float64) {
	drawHorizontalLine(0, int(row), 8)

	row += currentDirection

	if row > 7 {
		currentDirection = float64(-hvd)
		row = 6
	}

	if row < 0 {
		currentDirection = float64(hvd)
		row = 1
	}

	return currentDirection, row
}

func animateHorizontalScan(currentDirection float64, col float64) (float64, float64) {
	drawVerticalLine(0, int(col), 8)

	col += currentDirection

	if col > 7.5 {
		currentDirection = float64(-hod)
		col = 7
	}

	if col < 0 {
		currentDirection = float64(hod)
		col = 1
	}

	return currentDirection, col
}

// -----------------------------------------------------------
// Display buffer methods
// -----------------------------------------------------------
// 8x8 = 64 pixels
// var pixelBuf = make([]byte, 64)
// Or
var pixelBuf = [][]byte{
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 0, 0},
}

func clearBuf() {
	for cx := 0; cx < 8; cx++ {
		for ry := 0; ry < 8; ry++ {
			pixelBuf[cx][ry] = 0
		}
	}
}

func printBuf() {
	for cx := 0; cx < 8; cx++ {
		s := ""
		for ry := 0; ry < 8; ry++ {
			if pixelBuf[cx][ry] == 1 {
				s += "1"
			} else {
				s += "0"
			}
		}
		println(s)
	}
}

func blit(sp *spi.FtdiSPI) {
	var b byte

	for cx := 0; cx < 8; cx++ {
		// Get row pixels
		j := 0
		for ry := 0; ry < 8; ry++ {
			b |= pixelBuf[ry][cx] << uint(j)
			j++
		}
		// fmt.Printf("%08b\n", b)
		packet := []byte{byte(cx + 1), b}
		b = 0
		sp.Write(packet)
	}
}

func setPixel(x, y int) {
	// origin is on the top left.
	pixelBuf[y][x] = 1
}

func unSetPixel(x, y int) {
	// origin is on the top left.
	pixelBuf[y][x] = 0
}

func drawVerticalLine(ystart, col, height int) {
	for y := ystart; y < ystart+height; y++ {
		setPixel(col, y)
	}
}

func drawHorizontalLine(xstart, row, width int) {
	for x := xstart; x < xstart+width; x++ {
		setPixel(x, row)
	}
}
