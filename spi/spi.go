package spi

import (
	"log"

	"github.com/wdevore/hardware/ftdi"
	"github.com/wdevore/hardware/gpio"
)

// CaptureMode controls clock polarity and phase of bit capture.  Should be a
// numeric value 0, 1, 2, or 3.  See wikipedia page for details on meaning:
// http://en.wikipedia.org/wiki/Serial_Peripheral_Interface_Bus
type CaptureMode int

// When using SPI with the FT232H the following pins will have a special meaning:
// D0 - SCK / Clock signal.  This will be the clock that tells devices when to sample and write data.
// D1 - MOSI / Data Out.  This will output data from the FT232H to the connected device.
// D2 - MISO / Data In.  This will read data from the connected device to the FT232H.
// One thing to note is that there isn't an explicit chip select / enable pin.
// You should use any of the free GPIO pins as a dedicated chip select pin
// and specify that pin when creating the SPI object.

const (
	// Mode0 captures on rising clock, propagate on falling clock, clock base = low
	Mode0 CaptureMode = iota // Typical
	// Mode1 captures of falling edge, propagate on rising clock, clock base = low
	Mode1
	// Mode2 captures on rising clock, propagate on falling clock, clock base = high
	Mode2
	// Mode3 captures on falling edge, propagage on rising clock, clock base = high
	Mode3
)

// BitOrder specifies Most or Least significant is first in the bit stream
type BitOrder int

const (
	// MSBFirst indicates MSB bit is first
	MSBFirst BitOrder = iota
	// LSBFirst indicates LSB bit is first
	LSBFirst
)

// NoChipSelectAssignment means a chip select pin isn't assigned
const NoChipSelectAssignment = 9999

var writeCommand = []byte{0, 0, 0}
var writeCommand2 = []byte{0, 0, 0, 0}

// FtdiSPI is perspective of FTDI232H
type FtdiSPI struct {
	// SPI is-a protocol facilitated by FTDI232 device
	ftdi *ftdi.FTDI232H

	// CSActiveLow is chip select active high(false) or low(true)
	CSActiveLow        bool
	chipSelect         gpio.Pin
	hardwareControlled bool
	// ConstantCSAssert controls if CS is asserted on every read/write call or
	// remains constant in an active state. For example, some devices have multiple
	// slaves which means you want to assert on every call to make sure you are
	// targeting the tft. The default = true.
	ConstantCSAssert bool

	maxSpeed int
	mode     CaptureMode
	bitOrder BitOrder

	writeClockVE int
	readClockVE  int
}

// NewSPI creates an SPI FTDI component
// A chipSelect of `NoPin` means no assignment.
func NewSPI(vender, product int, disableDrivers bool) *FtdiSPI {
	spi := new(FtdiSPI)

	spi.ConstantCSAssert = true

	spi.ftdi = new(ftdi.FTDI232H)

	spi.ftdi.SetTarget(vender, product)

	err := spi.ftdi.Initialize(disableDrivers)

	if err != nil {
		log.Fatal(err)
		return nil
	}

	return spi
}

// Configure arranges default values for SPI.
func (spi *FtdiSPI) Configure(chipSelect gpio.Pin, maxSpeed int, mode CaptureMode, bitOrder BitOrder) error {
	err := spi.ftdi.Configure(true)
	if err != nil {
		log.Println("SPI failed to configure.")
		return err
	}

	spi.CSActiveLow = true // Default for SPI protocol

	// Typically CS is controlled by hardware, however, if you configured for software-spi
	// then you would want a specific pin defined. The default hardware CS depends on
	// your device. For the FTDI232H that pin is D3.
	if chipSelect == gpio.HardwarePin {
		spi.hardwareControlled = true
	} else {
		spi.hardwareControlled = false
	}

	if chipSelect == gpio.DefaultPin {
		chipSelect = ftdi.D3
	}

	spi.chipSelect = chipSelect
	spi.maxSpeed = maxSpeed
	spi.mode = mode
	spi.bitOrder = bitOrder

	fi := spi.ftdi

	if !spi.hardwareControlled {
		// log.Printf("SPI configuring chip select on pin (%d)\n", chipSelect)
		fi.SetConfigPin(chipSelect, gpio.Output)
		fi.OutputHigh(chipSelect)
	}

	// Initialize clock, mode, and bit order.
	// log.Printf("SPI Setting clock speed to (%d)MHz\n", maxSpeed/1000000)
	spi.SetClock(maxSpeed)
	// log.Println("SPI Setting mode")
	spi.SetMode(mode)

	// log.Println("SPI Setting bit order")
	spi.SetBitOrder(bitOrder)

	return nil
}

// GetFTDI returns the FTDI component
func (spi *FtdiSPI) GetFTDI() *ftdi.FTDI232H {
	return spi.ftdi
}

// Close closes the FTDI232 device
func (spi *FtdiSPI) Close() error {
	log.Println("SPI closing FTDI device")
	err := spi.ftdi.Close()

	if err != nil {
		return err
	}

	log.Println("SPI FTDI closed")
	return nil
}

// NewSPIDefaults creates an SPI component with default settings.
func NewSPIDefaults(vender, product int, disableDrivers bool) *FtdiSPI {
	spi := NewSPI(vender, product, disableDrivers)
	spi.Configure(NoChipSelectAssignment, 1000000, Mode0, MSBFirst)
	return spi
}

// SetClock sets the speed of the SPI clock in hertz.  Note that not all speeds
// are supported and a lower speed might be chosen by the hardware.
func (spi *FtdiSPI) SetClock(hz int) {
	spi.ftdi.SetClock(hz, false, false)
}

// SetMode sets SPI mode which controls clock polarity and phase.  Should be a
// numeric value 0, 1, 2, or 3.  See wikipedia page for details on meaning:
// http://en.wikipedia.org/wiki/Serial_Peripheral_Interface_Bus
func (spi *FtdiSPI) SetMode(mode CaptureMode) {
	var clockBase gpio.PinState

	switch mode {
	case Mode0:
		spi.writeClockVE = 1
		spi.readClockVE = 0
		// Clock base is low.
		clockBase = gpio.Low
		break
	case Mode1:
		spi.writeClockVE = 0
		spi.readClockVE = 1
		clockBase = gpio.Low
		break
	case Mode2:
		spi.writeClockVE = 1
		spi.readClockVE = 0
		clockBase = gpio.High
		break
	case Mode3:
		spi.writeClockVE = 0
		spi.readClockVE = 1
		clockBase = gpio.High
		break
	}

	pins := []gpio.PinConfiguration{
		{Pin: 0, Direction: gpio.Output, Value: clockBase}, // Set clock as output and start at it base value
		{Pin: 1, Direction: gpio.Output, Value: gpio.Z},
		{Pin: 2, Direction: gpio.Input, Value: gpio.Z},
	}
	spi.ftdi.ConfigPins(pins, true)
}

// SetBitOrder sets the order of bits to be read/written over serial lines.  Should be
// either MSBFIRST for most-significant first, or LSBFIRST for
// least-signifcant first.
func (spi *FtdiSPI) SetBitOrder(order BitOrder) {
	spi.bitOrder = order
}

// Write writes the specified array of bytes out on the MOSI line.
// This is a Half-duplex SPI write.
func (spi *FtdiSPI) Write(data []byte) error {
	// Build command to write SPI data.
	writeCommand[0] = 0x10 | (byte(spi.bitOrder) << 3) | byte(spi.writeClockVE)
	// logger.debug('SPI write with command {0:2X}.'.format(command))

	// Compute length low and high bytes.
	// NOTE: Must actually send length minus one because the MPSSE engine
	// considers `0` a length of 1 and 0xffff a length of 65536
	length := uint16(len(data) - 1)
	writeCommand[1] = byte(length & 0xff)
	writeCommand[2] = byte((length >> 8) & 0xff)

	if !spi.ConstantCSAssert {
		spi.AssertChipSelect()
	}

	// Send command and length.
	_, err := spi.ftdi.Write(writeCommand)

	if err != nil {
		return err
	}

	// Send data.
	_, err = spi.ftdi.Write(data)
	if err != nil {
		return err
	}

	if !spi.ConstantCSAssert {
		spi.DeAssertChipSelect()
	}

	return nil
}

// WriteLen writes the specified array of bytes out on the MOSI line.
// Allows writing of variable length arrays of fixed size
// This is a Half-duplex SPI write.
func (spi *FtdiSPI) WriteLen(data []byte, length int) error {
	// Build command to write SPI data.
	writeCommand[0] = 0x10 | (byte(spi.bitOrder) << 3) | byte(spi.writeClockVE)
	// logger.debug('SPI write with command {0:2X}.'.format(command))

	// Compute length low and high bytes.
	// NOTE: Must actually send length minus one because the MPSSE engine
	// considers `0` a length of 1 and 0xffff a length of 65536
	dlength := uint16(length - 1)
	writeCommand[1] = byte(dlength & 0xff)
	writeCommand[2] = byte((dlength >> 8) & 0xff)

	if !spi.ConstantCSAssert {
		spi.AssertChipSelect()
	}

	// Send command and length.
	_, err := spi.ftdi.Write(writeCommand)

	if err != nil {
		return err
	}

	// Send data.
	_, err = spi.ftdi.WriteLen(data, length)
	if err != nil {
		return err
	}

	if !spi.ConstantCSAssert {
		spi.DeAssertChipSelect()
	}

	return nil
}

// Half-duplex SPI read.  The specified length of bytes will be clocked
// in the MISO line and returned as a bytearray object.
func (spi *FtdiSPI) Read(length int) ([]byte, error) {
	// Build command to read SPI data.
	writeCommand2[0] = 0x20 | (byte(spi.bitOrder) << 3) | (byte(spi.readClockVE) << 2)
	// logger.debug('SPI read with command {0:2X}.'.format(command))

	// Compute length low and high bytes.
	// NOTE: Must actually send length minus one because the MPSSE engine
	// considers 0 a length of 1 and 0xffff a length of 65536
	writeCommand2[1] = byte((length - 1) & 0xff)
	writeCommand2[2] = byte(((length - 1) >> 8) & 0xff)
	writeCommand2[3] = 0x87

	if !spi.ConstantCSAssert {
		spi.AssertChipSelect()
	}

	// Send command and length.
	spi.Write(writeCommand2)

	if !spi.ConstantCSAssert {
		spi.DeAssertChipSelect()
	}

	// Read response bytes.
	response, err := spi.ftdi.PollRead(length, -1)

	return response, err
}

// Transfer is a Full-duplex SPI read and write.  The specified array of bytes will be
// clocked out the MOSI line, while simultaneously bytes will be read from
// the MISO line.  Read bytes will be returned as a bytearray object.
func (spi *FtdiSPI) Transfer(data []byte) ([]byte, error) {
	// Build command to read and write SPI data.
	writeCommand[0] = 0x30 | (byte(spi.bitOrder) << 3) | byte(spi.readClockVE<<2) | byte(spi.writeClockVE)
	// logger.debug('SPI transfer with command {0:2X}.'.format(command))
	// Compute length low and high bytes.
	// NOTE: Must actually send length minus one because the MPSSE engine
	// considers 0 a length of 1 and 0xffff a length of 65536

	length := len(data)
	writeCommand[1] = byte((length - 1) & 0xff)
	writeCommand[2] = byte(((length - 1) >> 8) & 0xff)

	// Send command and length.
	if !spi.ConstantCSAssert {
		spi.AssertChipSelect()
	}

	spi.Write(writeCommand)
	spi.Write(data)
	spi.ftdi.WriteByte(0x87)

	if !spi.ConstantCSAssert {
		spi.DeAssertChipSelect()
	}

	// Read response bytes.
	response, err := spi.ftdi.PollRead(length, -1)

	return response, err
}

// AssertChipSelect will toggle chip select low or high depending on Active configuration
func (spi *FtdiSPI) AssertChipSelect() {
	if spi.chipSelect != gpio.NoPin {
		// log.Println("SPI asserting chip select")
		if spi.CSActiveLow {
			spi.ftdi.OutputLow(spi.chipSelect)
		} else {
			spi.ftdi.OutputHigh(spi.chipSelect)
		}
	}
}

// DeAssertChipSelect will toggle chip select low or high depending on Active configuration
func (spi *FtdiSPI) DeAssertChipSelect() {
	if spi.chipSelect != gpio.NoPin {
		// log.Println("SPI DE-asserting chip select")
		if spi.CSActiveLow {
			spi.ftdi.OutputHigh(spi.chipSelect)
		} else {
			spi.ftdi.OutputLow(spi.chipSelect)
		}
	}
}

// ----------------------------------------------------------------------------------
// Debug stuff
// ----------------------------------------------------------------------------------

// DebugInit configure debugging
func (spi *FtdiSPI) DebugInit() {
	spi.ftdi.SetConfigPin(ftdi.D7, gpio.Output) // Trigger
}

// TriggerPulse generate a timed pulse for various tools, ex Logic analyser.
func (spi *FtdiSPI) TriggerPulse() {
	spi.ftdi.OutputHigh(ftdi.D7)
	spi.ftdi.OutputLow(ftdi.D7)
}
