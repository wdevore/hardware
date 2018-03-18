package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wdevore/hardware/ftdi"
	"github.com/wdevore/hardware/spi"
)

func main() {
	log.Println("Creating SPI device")
	spid := spi.NewSPI(0x0403, 0x06014, false)

	// Create a SPI interface from the FT232H using pin D3 as chip select.
	// Use a clock speed of 100KHz, SPI mode 0, and most significant bit first.
	spid.Configure(ftdi.D3, 100000, spi.Mode0, spi.MSBFirst)

	// Max requires an active CS so we disable constant assert so CS will toggle.
	spid.ConstantCSAssert = false

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(ft *spi.FtdiSPI) {
		<-c
		println("\nReceived ctrl-C, closing FTDI interface.")
		err := ft.Close()
		if err != nil {
			println("\n Failed to close FTDI interface")
			os.Exit(-1)
		}
		os.Exit(0)
	}(spid)

	defer spid.Close()

	clearRegisters(spid)

	// superSimple(spid)
	// enableCol2WithPattern(spid)
	// random3(spid)
	// horizontalScanBar(spid)
	// dualScanBars2(spid)
	// setPixelTest(spid)
	// matrix(spid)

	// Cascaded displays
	// setPixelPattern4(spid)
	// simpleSetPixels4(spid)
	// matrix4(spid)
	// random4(spid)
	flipflop4(spid)
}

const (
	zero byte = 0x00
)

// Registers
const (
	noOpReg byte = zero // No-Op

	// For this code the digits actually represent which column is enabled
	// You OR them to enable multple columns.
	digit0Reg byte = 0x01
	digit1Reg byte = 0x02 // For example this would enable column 2 only
	digit2Reg byte = 0x03
	digit3Reg byte = 0x04
	digit4Reg byte = 0x05
	digit5Reg byte = 0x06
	digit6Reg byte = 0x07
	digit7Reg byte = 0x08

	modeReg        byte = 0x09 // Decode Mode
	intensityReg   byte = 0x0a
	scanLimitReg   byte = 0x0b
	shutdownReg    byte = 0x0c
	displayTestReg byte = 0x0f
)

// Decode modes
const (
	noDecode           = zero
	codeBDigit0   byte = 0x01
	codeBDigit3_0 byte = 0x0f
	codeBDigit7_0 byte = 0xff
)

// Shutdown modes
const (
	shutdown byte = zero
	normal   byte = 0x01
)

// DisplayTest modes
const (
	off byte = zero
	on  byte = 0x01
)

// Intensities (a few predefined) values can range from 0 to 0x0F
const (
	iMin byte = zero
	iMed byte = 0x07
	iMax byte = 0x0f
)

// Scan limits indicates how many digits are displayed or columns enabled.
const (
	// I use the word "column" instead of digit
	noColumns  byte = zero
	allColumns byte = 0x07
)

func clearRegisters(sp *spi.FtdiSPI) {
	var err error

	err = sp.Write([]byte{modeReg, noDecode})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{intensityReg, 0x01})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{scanLimitReg, allColumns})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{shutdownReg, normal}) // Normal operation
	if err != nil {
		log.Fatal(err)
	}

	sp.Write([]byte{digit0Reg, zero})
	sp.Write([]byte{digit1Reg, zero})
	sp.Write([]byte{digit2Reg, zero})
	sp.Write([]byte{digit3Reg, zero})
	sp.Write([]byte{digit4Reg, zero})
	sp.Write([]byte{digit5Reg, zero})
	sp.Write([]byte{digit6Reg, zero})
	sp.Write([]byte{digit7Reg, zero})

}

func stringToByte(bs string) byte {
	var b byte

	// This is a "reverse" interation because the LSb is at the end of the string.
	for i := range bs {
		c := bs[len(bs)-i-1]
		if c == '1' {
			b |= 1 << uint(i)
		}
		i++
	}

	return b
}

// ------------------------------------------------------------------
// Test and examples

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

// -----------------------------------------------------------
// Display buffer examples for a cascade of 4 8x8 displays.
// You can buy these cascades as a single unit.
// Note: this strategy of using a buffer "hard codes" the display
// setup. Thus using the shifting features of the chained
// displays is "lost".
// -----------------------------------------------------------
var dataPacket = []byte{
	// First byte is col, second are rows
	0, 0,
	0, 0,
	0, 0,
	0, 0,
}

// A buffered shaped as: 32x8 = Row x Col
//               Row(ry)    Col(cx)
//                |           |
//                |     /-----/
//                v     v
var pixelBuf4 = [8 * 4][8]byte{}

// 	// This chunk is the last display where the output pins are.
//      Display columns
//   8  7  6  5  4  3  2  1
//          Row columns
//   0  1  2  3  4  5  6  7
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
//
//  2nd and 3rd chunks
//
// 	// This chunk is the first display where the input pins are.
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},
// 	{0, 0, 0, 0, 0, 0, 0, 0},

// x = col, y = row
func setPixel4(cx, ry int) {
	pixelBuf4[ry][cx] = 1
}

func unSetPixel4(cx, ry int) {
	pixelBuf4[ry][cx] = 0
}

func getPixel4(cx, ry int) byte {
	return pixelBuf4[ry][cx]
}

func clearDisplay4(sp *spi.FtdiSPI) {
	// For each 8x8 display we send 4 16bit blocks. Each block belongs
	// to one display.
	// As always the column index starts at one based on the specsheet.
	for col := byte(1); col < 9; col++ {
		dataPacket[0] = col
		dataPacket[2] = col
		dataPacket[4] = col
		dataPacket[6] = col
		sp.Write(dataPacket)
	}
}

func clearBuf4() {
	for ry := 0; ry < 8*4; ry++ {
		for cx := 0; cx < 8; cx++ {
			pixelBuf4[ry][cx] = 0
		}
	}
}

func printBuf4() {
	for ry := 0; ry < 8*4; ry++ {
		s := ""
		for cx := 0; cx < 8; cx++ {
			// fmt.Printf("ry: %d, cx: %d\n", ry, cx)
			if pixelBuf4[ry][cx] == 1 {
				s += "1"
			} else {
				s += "0"
			}
		}
		println(s)
	}
}

func blit4(sp *spi.FtdiSPI) {
	var b byte

	// We need to do something similar to clearDisplay4()
	cx := 7
	for col := byte(0); col < 8; col++ {
		dataPacket[0] = col + 1
		dataPacket[2] = col + 1
		dataPacket[4] = col + 1
		dataPacket[6] = col + 1

		// --------------------------
		// Top display
		// --------------------------
		// Now set the row data for the farthest display
		// The left most bit (0) is the top
		// The right most bit (7) is the bottom
		b = pixelBuf4[7][cx]
		b |= pixelBuf4[6][cx] << 1
		b |= pixelBuf4[5][cx] << 2
		b |= pixelBuf4[4][cx] << 3
		b |= pixelBuf4[3][cx] << 4
		b |= pixelBuf4[2][cx] << 5
		b |= pixelBuf4[1][cx] << 6
		b |= pixelBuf4[0][cx] << 7
		// fmt.Printf("B: %08b\n", b)
		dataPacket[1] = b // Nearest to *Output* pins

		b = pixelBuf4[15][cx]
		b |= pixelBuf4[14][cx] << 1
		b |= pixelBuf4[13][cx] << 2
		b |= pixelBuf4[12][cx] << 3
		b |= pixelBuf4[11][cx] << 4
		b |= pixelBuf4[10][cx] << 5
		b |= pixelBuf4[9][cx] << 6
		b |= pixelBuf4[8][cx] << 7
		dataPacket[3] = b

		b = pixelBuf4[23][cx]
		b |= pixelBuf4[22][cx] << 1
		b |= pixelBuf4[21][cx] << 2
		b |= pixelBuf4[20][cx] << 3
		b |= pixelBuf4[19][cx] << 4
		b |= pixelBuf4[18][cx] << 5
		b |= pixelBuf4[17][cx] << 6
		b |= pixelBuf4[16][cx] << 7
		dataPacket[5] = b

		// --------------------------
		// Bottom display
		// --------------------------
		b = pixelBuf4[31][cx]
		b |= pixelBuf4[30][cx] << 1
		b |= pixelBuf4[29][cx] << 2
		b |= pixelBuf4[28][cx] << 3
		b |= pixelBuf4[27][cx] << 4
		b |= pixelBuf4[26][cx] << 5
		b |= pixelBuf4[25][cx] << 6
		b |= pixelBuf4[24][cx] << 7
		dataPacket[7] = b

		sp.Write(dataPacket)

		cx--
	}
}

func setPixelPattern4(sp *spi.FtdiSPI) {

	for i := 0; i < 1; i++ {
		clearDisplay4(sp)
		clearBuf4()
		// top-left = top most display,top-most row
		setPixel4(7, 0)
		setPixel4(7, 1)
		setPixel4(7, 2)
		setPixel4(1, 0)
		setPixel4(0, 0)

		setPixel4(6, 3)
		setPixel4(6, 4)
		setPixel4(6, 5)
		setPixel4(6, 6)
		setPixel4(6, 7)

		setPixel4(5, 0)
		setPixel4(5, 2)

		// Set a few pixels on the 2nd from top display
		setPixel4(7, 8)
		setPixel4(0, 8)

		// Set a few pixels on the 3rd from top display
		setPixel4(2, 17)
		setPixel4(5, 17)

		// Set a few pixels on the bottom display
		setPixel4(4, 29)
		setPixel4(7, 29)

		// printBuf4()
		blit4(sp)
		time.Sleep(time.Millisecond * 10)
	}
}

func simpleSetPixels4(sp *spi.FtdiSPI) {

	for i := 0; i < 100000; i++ {
		clearDisplay4(sp)

		packet := []byte{
			// The right most bit is the bottom
			// The left most bit is the top
			1, stringToByte("11100000"), // Nearest to *Output* pins
			1, stringToByte("00000010"),
			1, stringToByte("00100100"),
			1, stringToByte("00001000"), // Nearest to *Input* pins
		}
		sp.Write(packet)

		time.Sleep(time.Millisecond * 10)
	}
}

func matrix4(sp *spi.FtdiSPI, loop int, scrollDirection bool, scrollSpeed int) {
	ran := rand.New(rand.NewSource(99))
	clearBuf4()

	for i := 0; i < loop; i++ {
		// Set a random pixel on row 0
		rcol := int(ran.Float32() * 8)

		if scrollDirection {
			setPixel4(rcol, 0)
		} else {
			setPixel4(rcol, 31)
		}

		blit4(sp)
		time.Sleep(time.Millisecond * time.Duration(scrollSpeed))

		if scrollDirection {
			// scroll downwards
			for ry := 8*4 - 1; ry > 0; ry-- {
				for cx := 0; cx < 8; cx++ {
					// copy row-1 into row
					pixelBuf4[ry][cx] = pixelBuf4[ry-1][cx]
				}
			}
		} else {
			// scroll up
			for ry := 0; ry < 8*4-1; ry++ {
				for cx := 0; cx < 8; cx++ {
					// copy row+1 into row
					pixelBuf4[ry][cx] = pixelBuf4[ry+1][cx]
				}
			}
		}

		// clear row 0
		if scrollDirection {
			unSetPixel4(rcol, 0)
		} else {
			unSetPixel4(rcol, 31)
		}
	}
}

func random4(sp *spi.FtdiSPI, loop int, pause int) {
	ran := rand.New(rand.NewSource(99))

	for i := 0; i < loop; i++ {
		clearBuf4()

		// Fill buff with random pattern
		for ry := 0; ry < 8*4; ry++ {
			for cx := 0; cx < 8; cx++ {
				if ran.Float32() > 0.6 {
					setPixel4(cx, ry)
				}
			}
		}

		// blit
		blit4(sp)

		// pause
		time.Sleep(time.Millisecond * time.Duration(pause))
	}

	log.Println("Done.")
}

func flipflop4(sp *spi.FtdiSPI) {
	ran := rand.New(rand.NewSource(99))
	flip := false

	for j := 0; j < 10000; j++ {
		scrollSpeed := int(ran.Float32()*50 + 5)
		pause := int(ran.Float32()*200 + 50)

		matrix4(sp, 100, flip, scrollSpeed)

		random4(sp, 20, pause)

		flip = !flip
	}
}
