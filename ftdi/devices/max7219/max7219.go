package max

import (
	"fmt"

	"github.com/wdevore/hardware/spi"
)

const (
	vender  = 0x0403
	product = 0x06014
)

const (
	zero byte = 0x00
)

// Register addresses
const (
	noOpReg byte = zero // No-Op

	// For this code the digits actually represent which column is enabled
	// You logically "OR" them to enable multple columns.
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

	// A register is broken down into address-data components each 16bits.
	addressData = 2 // bytes

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

// IMatrix provides a common API for Max7219 led matrix devices
type IMatrix interface {
	// ---------------------------------------------------------
	// Device methods
	// ---------------------------------------------------------
	Initialize() error
	Close() error
	GetWidth() int
	GetHeight() int
	ClearDevice()

	// ---------------------------------------------------------
	// Graphics
	// ---------------------------------------------------------
	SetPixel(x, y int)
	ClearPixel(x, y int)
	GetPixel(x, y int) uint8
	ChangePixel(x, y int, v uint8)
	DrawHLine(x, y, w int)
	DrawVLine(x, y, h int)
	DrawRectangle(x, y, w, h int)
	ClearDisplay()
	UpdateDisplay() error

	// ---------------------------------------------------------
	// Misc
	// ---------------------------------------------------------
	ActivateTestMode(bool) error
	CopyBuf(out [][]byte)
	PrintBuf()
}

var packet16 = []byte{0x00, 0x00}

type matrix struct {
	speed     int
	intensity byte

	spi *spi.FtdiSPI

	// A single strip of data sent.
	// For example a 4x4 cascade (i.e. 32*4 pixels) requires
	// a strip of size 128bits. This strip is sent 8 times to
	// cover the total "area".
	packet []byte // size = numberOf4x1s * pixelsPerColumn * numberOf8x8s

	numberOf8x8s int
	numberOf4x1s int // number of 4x1 cascades

	width  int
	height int

	pixelBuf [][]byte
}

// ---------------------------------------------------------
// Misc
// ---------------------------------------------------------

// PrintBuf prints internal buffer
func (m *matrix) PrintBuf() {
	for y := 0; y < m.height; y++ {
		fmt.Printf("%v\n", m.pixelBuf[y])
	}
}

func (m *matrix) CopyBuf(out [][]byte) {
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			out[y][x] = m.GetPixel(x, y)
		}
	}
}

// ---------------------------------------------------------
// Graphics
// ---------------------------------------------------------

// ClearDisplay clears the matrix buffer
func (m *matrix) ClearDisplay() {
	// Clear display
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			m.ClearPixel(x, y)
		}
	}
}

// SetPixel sets a pixel in device dimensions
func (m *matrix) SetPixel(x, y int) {
	if x < 0 || y < 0 || (x > (m.width - 1)) || (y > (m.height - 1)) {
		return
	}
	m.pixelBuf[y][x] = 1
}

// ClearPixel clears a pixel in device dimensions
func (m *matrix) ClearPixel(x, y int) {
	m.pixelBuf[y][x] = 0
}

// GetPixel returns a pixel in device dimensions
func (m *matrix) GetPixel(x, y int) uint8 {
	return m.pixelBuf[y][x]
}

// ChangePixel set a pixel to a given value
func (m *matrix) ChangePixel(x, y int, v uint8) {
	if x < 0 || y < 0 || (x > (m.width - 1)) || (y > (m.height - 1)) {
		return
	}
	m.pixelBuf[y][x] = v
}

// DrawHLine renders a horizontal line
func (m *matrix) DrawHLine(x, y, w int) {
	if x >= m.width {
		return
	}

	for ix := x; ix < x+w; ix++ {
		if ix > m.width {
			continue
		}
		m.SetPixel(ix, y)
	}
}

// DrawVLine renders a vertical line
func (m *matrix) DrawVLine(x, y, h int) {
	if y >= m.height {
		return
	}

	for iy := y; iy < y+h; iy++ {
		if iy > m.height {
			continue
		}
		m.SetPixel(x, iy)
	}
}

func (m *matrix) DrawRectangle(x, y, w, h int) {
	if y >= m.height || x >= m.width {
		return
	}

	for ry := y; ry < y+h; ry++ {
		for rx := x; rx < x+w; rx++ {
			m.SetPixel(rx, ry)
		}
	}
}
