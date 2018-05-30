package max

import (
	"github.com/wdevore/hardware/gpio"
	"github.com/wdevore/hardware/spi"
)

// Matrix4x4 implements a grid 4x4 led matrix array
type Matrix4x4 struct {
	matrix
}

// NewMatrix4x4 creates a 4x4 matrix driver
func NewMatrix4x4(speed int, intensity uint8) IMatrix {
	m := new(Matrix4x4)
	m.speed = speed
	m.intensity = intensity
	return m
}

// ---------------------------------------------------------
// Device methods
// ---------------------------------------------------------

// Initialize configures SPI
func (m *Matrix4x4) Initialize() error {
	m.spi = spi.NewSPI(vender, product, false)
	m.spi.EnableTrigger()

	err := m.spi.Configure(gpio.DefaultPin, m.speed, spi.Mode0, spi.MSBFirst)

	if err != nil {
		return err
	}

	// Max7219 requires an active CS so we disable constant assert so CS will toggle.
	m.spi.ConstantCSAssert = false

	// Default CS
	m.spi.DeAssertChipSelect()

	m.numberOf8x8s = 16
	m.numberOf4x1s = 4

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

	m.width = 32
	m.height = 32

	m.pixelBuf = make([][]byte, m.height)
	for i := range m.pixelBuf {
		m.pixelBuf[i] = make([]byte, m.width)
	}

	return nil
}

// Close closes SPI
func (m *Matrix4x4) Close() error {
	return m.spi.Close()
}

// GetWidth returns the devices width in pixels
func (m *Matrix4x4) GetWidth() int {
	return m.width
}

// GetHeight returns the devices height in pixels
func (m *Matrix4x4) GetHeight() int {
	return m.height
}

// ClearDevice clears the device
func (m *Matrix4x4) ClearDevice() {
	m.clearRegisters()
}

// ---------------------------------------------------------
// Misc
// ---------------------------------------------------------

// ActivateTestMode turns on all leds which bypasses any digit register values
func (m *Matrix4x4) ActivateTestMode(activate bool) error {

	packet16[0] = displayTestReg

	if activate {
		packet16[1] = on // display test mode
	} else {
		packet16[1] = off
	}

	m.spi.TakeControlOfCS()

	m.spi.AssertChipSelect()
	for n := 0; n < 16; n++ {
		err := m.spi.Write(packet16)
		if err != nil {
			return err
		}
	}
	m.spi.DeAssertChipSelect()

	m.spi.ReleaseControlOfCS()

	return nil
}

// ---------------------------------------------------------
// Graphics
// ---------------------------------------------------------

// UpdateDisplay blits the pixel buffer to device
func (m *Matrix4x4) UpdateDisplay() error {
	m.spi.TriggerPulse()

	// A packet is a stream of 128 bits = 16x8.
	//  vertical col       vertical col      vertical col     vertical col
	// c,p c,p c,p c,p - c,p c,p c,p c,p - c,p c,p c,p c,p - c,p c,p c,p c,p
	// c = column id 1->8 = 8bits
	// p = 8 bit pattern on one of the 8x8 matrices
	// Thus c,p = 16bits = 2Bytes

	// A "vertical col" = cp*4 = 2bytes*4 = 64bits = 8 Bytes

	// This means an entire packet = 256bits = 32 Bytes

	// Each blit means sending a packet for each column (8 total) = 8*32 = 2048bits = 256 bytes
	//  0  1     2  3     4  5     6  7
	// [c][p] - [c][p] - [c][p] - [c][p] -...
	//     8        16       24       32

	// sp.Write(dataPacket)
	col := byte(1)
	// bank = 31,30,29,28,27,26,25,24
	for bank := m.width - 1; bank >= m.width-8; bank-- {
		cp := 0

		for i := 0; i < 32; i++ {
			m.packet[i] = 0
		}
		// fmt.Printf("packet set: %v\n", m.packet)

		// fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
		x := bank

		for interleave := 4; interleave > 0; interleave-- {
			// Scan down a bank column in the pixel buffer.
			ry := uint8(7)

			for y := 0; y < m.height; y++ {
				m.packet[cp] = col // unfortunately col is being assigned 8 times.

				m.packet[cp+1] |= m.GetPixel(x, y) << ry
				// fmt.Printf("bank: %d, interleave: %d, x:%d, y:%d, cp:%d, ry:%d, col:%d, pack:%08b, pix:%d\n", bank, interleave, x, y, cp, ry, m.packet[cp], m.packet[cp+1], m.pixelBuf[y][x])

				// Every 8 bits we move to the next packet pattern byte
				if ry <= 0 {
					// fmt.Println("##############################")
					ry = 7
					// move to the next "p" component byte.
					cp += 2
					continue
				}
				ry--
			}
			x -= 8 // move to the next interleave column
		}
		col++

		// fmt.Printf("bank: %d, send: %v\n", bank, m.packet)
		m.spi.Write(m.packet)
	}

	m.spi.ReleaseControlOfCS()

	// time.Sleep(time.Millisecond)
	return nil
}

// --------------------------------------------------------------------
// Internal methods
// --------------------------------------------------------------------

// init each 8x8 led matrix
func (m *Matrix4x4) initRegisters() error {
	// There are 16 8x8 arranged as:
	// x x x x
	// x x x x
	// x x x x
	// x x x x
	//
	// each "x" has a column = address(1byte) + pattern(1byte)
	// thus 1 column of x's = 4 * 2bytes = 8 bytes
	// thus 1 packet = 4 * 8bytes = 32bytes = 16*2bytes

	m.packet = make([]byte, m.numberOf8x8s*2)

	var err error
	m.spi.TakeControlOfCS()

	m.spi.AssertChipSelect()
	for n := 0; n < 16; n++ {
		packet16[0] = shutdownReg
		packet16[1] = shutdown
		err = m.spi.Write(packet16) // Normal operation
		if err != nil {
			return err
		}
	}
	m.spi.DeAssertChipSelect()

	m.spi.AssertChipSelect()
	for n := 0; n < 16; n++ {
		packet16[0] = modeReg
		packet16[1] = noDecode
		err = m.spi.Write(packet16)
		if err != nil {
			return err
		}
	}
	m.spi.DeAssertChipSelect()

	m.spi.AssertChipSelect()
	for n := 0; n < 16; n++ {
		packet16[0] = intensityReg
		packet16[1] = m.intensity
		err = m.spi.Write(packet16)
		if err != nil {
			return err
		}
	}
	m.spi.DeAssertChipSelect()

	m.spi.AssertChipSelect()
	for n := 0; n < 16; n++ {
		packet16[0] = scanLimitReg
		packet16[1] = allColumns
		err = m.spi.Write(packet16)
		if err != nil {
			return err
		}
	}
	m.spi.DeAssertChipSelect()

	m.spi.AssertChipSelect()
	for n := 0; n < 16; n++ {
		packet16[0] = shutdownReg
		packet16[1] = normal
		err = m.spi.Write(packet16)
		if err != nil {
			return err
		}
	}
	m.spi.DeAssertChipSelect()

	m.spi.ReleaseControlOfCS()

	return nil
}

// Clears display by setting all the digit registers to zero in
// each 8x8 led matrix in chain.
func (m *Matrix4x4) clearRegisters() error {
	m.spi.TakeControlOfCS()

	for col := byte(1); col < 9; col++ {
		m.spi.AssertChipSelect()

		// For this column shift into each matrix
		for n := 0; n < 16; n++ {

			packet16[0] = col // set column id
			packet16[1] = 0   // zero 8 bit pattern = clear

			err := m.spi.Write(packet16)
			if err != nil {
				return err
			}
		}

		m.spi.DeAssertChipSelect()
	}

	m.spi.ReleaseControlOfCS()

	return nil
}
