package max

import (
	"github.com/wdevore/hardware/gpio"
	"github.com/wdevore/hardware/spi"
)

// Matrix1x1 implements a 1x1 led matrix
type Matrix1x1 struct {
	matrix
}

// NewMatrix1x1 creates a 1x1 matrix driver
func NewMatrix1x1(speed int, intensity uint8) IMatrix {
	m := new(Matrix1x1)
	m.speed = speed
	m.intensity = intensity
	return m
}

// ---------------------------------------------------------
// Device methods
// ---------------------------------------------------------

// Initialize configures SPI
func (m *Matrix1x1) Initialize() error {
	m.spi = spi.NewSPI(vender, product, false)

	err := m.spi.Configure(gpio.DefaultPin, m.speed, spi.Mode0, spi.MSBFirst)

	if err != nil {
		return err
	}

	// Max7219 requires an active CS so we disable constant assert so CS will toggle.
	m.spi.ConstantCSAssert = false

	// Default CS
	m.spi.DeAssertChipSelect()

	m.numberOf8x8s = 1

	err = m.initRegisters()
	if err != nil {
		return err
	}

	err = m.clearRegisters()
	if err != nil {
		return err
	}

	// Pixel buffer configuration
	// Origin is at the top-left corner. Typically farther away from
	// input edge of device.

	m.width = 8
	m.height = 8

	m.pixelBuf = make([][]byte, m.height)
	for i := range m.pixelBuf {
		m.pixelBuf[i] = make([]byte, m.width)
	}

	return nil
}

// Close closes SPI
func (m *Matrix1x1) Close() error {
	return m.spi.Close()
}

// GetWidth returns the devices width in pixels
func (m *Matrix1x1) GetWidth() int {
	return m.width
}

// GetHeight returns the devices height in pixels
func (m *Matrix1x1) GetHeight() int {
	return m.height
}

// ClearDevice clears the device
func (m *Matrix1x1) ClearDevice() {
	m.clearRegisters()
}

// ---------------------------------------------------------
// Misc
// ---------------------------------------------------------

// ActivateTestMode turns on all leds which bypasses any digit register values
func (m *Matrix1x1) ActivateTestMode(activate bool) error {
	m.packet[0] = displayTestReg
	if activate {
		m.packet[1] = on
	} else {
		m.packet[1] = off
	}

	err := m.spi.Write(m.packet)
	if err != nil {
		return err
	}

	return nil
}

// ---------------------------------------------------------
// Graphics
// ---------------------------------------------------------

// ClearDisplay clears the matrix buffer
func (m *Matrix1x1) ClearDisplay() {
	// Clear display
	for y := 0; y < m.height; y++ {
		for x := 0; x < m.width; x++ {
			m.ClearPixel(x, y)
		}
	}
}

// UpdateDisplay blits the pixel buffer to device
func (m *Matrix1x1) UpdateDisplay() error {
	var b byte

	// Turns on upper-left corner pixel
	// m.packet[0] = 8    // column register address
	// m.packet[1] = 0x80 // row pattern
	// m.spi.Write(m.packet)
	// return nil

	// We can only write columns not rows because the registers
	// expect column addresses followed by an 8bit row pattern.

	// cols: left --> 8 7 6 5 4 3 2 1 <-- right  == ry
	// pattern: left most bit is higher or row 0 in buf
	for col := byte(1); col < 9; col++ {
		// Get row of pixels
		j := 7
		x := 8 - col
		for ry := 0; ry < m.height; ry++ {
			b |= m.pixelBuf[ry][x] << uint(j)
			j--
		}
		// fmt.Printf("%08b, col: %d\n", b, col)

		// Writing a packet means we are writing a column
		// which runs left/right
		m.packet[0] = col // column register address
		m.packet[1] = b   // row pattern
		b = 0

		err := m.spi.Write(m.packet)
		if err != nil {
			return err
		}
	}

	return nil
}

// --------------------------------------------------------------------
// Internal methods
// --------------------------------------------------------------------

func (m *Matrix1x1) initRegisters() error {
	m.packet = make([]byte, m.numberOf8x8s*addressData)

	var err error
	m.spi.TakeControlOfCS()

	m.spi.AssertChipSelect()
	packet16[0] = shutdownReg
	packet16[1] = shutdown
	err = m.spi.Write(packet16) // Normal operation
	if err != nil {
		return err
	}
	m.spi.DeAssertChipSelect()

	m.spi.AssertChipSelect()
	packet16[0] = modeReg
	packet16[1] = noDecode
	err = m.spi.Write(packet16)
	if err != nil {
		return err
	}
	m.spi.DeAssertChipSelect()

	m.spi.AssertChipSelect()
	packet16[0] = intensityReg
	packet16[1] = m.intensity
	err = m.spi.Write(packet16)
	if err != nil {
		return err
	}
	m.spi.DeAssertChipSelect()

	m.spi.AssertChipSelect()
	packet16[0] = scanLimitReg
	packet16[1] = allColumns
	err = m.spi.Write(packet16)
	if err != nil {
		return err
	}
	m.spi.DeAssertChipSelect()

	m.spi.AssertChipSelect()
	packet16[0] = shutdownReg
	packet16[1] = normal
	err = m.spi.Write(packet16)
	if err != nil {
		return err
	}
	m.spi.DeAssertChipSelect()

	m.spi.ReleaseControlOfCS()

	return nil
}

// Clears display by setting all the digit registers to zero.
func (m *Matrix1x1) clearRegisters() error {
	m.spi.TakeControlOfCS()

	for col := byte(1); col < 9; col++ {
		m.spi.AssertChipSelect()

		packet16[0] = col  // set column id
		packet16[1] = zero // zero 8 bit pattern = clear

		err := m.spi.Write(packet16)

		if err != nil {
			return err
		}

		m.spi.DeAssertChipSelect()
	}

	m.spi.ReleaseControlOfCS()

	return nil
}
