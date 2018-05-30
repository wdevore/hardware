package spi

// A software emulation of the SPI protocol using BitBang mode.
// Note: this is extremely slow, for example, the fastest a pin will
// toggle is 12KHz on the FTDI232H!

import (
	"fmt"
	"log"
	"time"

	"github.com/wdevore/hardware/ftdi"
	"github.com/wdevore/hardware/gpio"
)

// When using SPI with the FT232H the following pins will have a special meaning:
// D0 - SCK / Clock signal.  This will be the clock that tells devices when to sample and write data.
// D1 - MOSI / Data Out.  This will output data from the FT232H to the connected device.
// D2 - MISO / Data In.  This will read data from the connected device to the FT232H.
// One thing to note is that there isn't an explicit chip select / enable pin.
// You should use any of the free GPIO pins as a dedicated chip select pin
// and specify that pin when creating the SPI object.

const (
	// MSBFirst indicates MSB bit is first
	SoftMSBFirst BitOrder = iota
	// LSBFirst indicates LSB bit is first
	SoftLSBFirst
)

// SoftSPI is an emulation
type SoftSPI struct {
	// SPI is-a protocol facilitated by FTDI232 device
	ftdi *ftdi.FTDI232H

	//
	//   m m         t
	// c i o   r     r
	// l s s c s     i
	// k o i s t     g
	// | | | | |     |
	// 0 1 2 3 4 5 6 7

	clk  gpio.Pin // Output = D0
	miso gpio.Pin // Input  = D1 = Data out
	mosi gpio.Pin // Output = D2 = Data in
	cs   gpio.Pin // Output = D3
	rst  gpio.Pin // Output = D4
	trig gpio.Pin // Output = D7

	pins byte

	// CSActiveLow is chip select active high(false) or low(true)
	CSActiveLow bool

	// ConstantCSAssert controls if CS is asserted on every read/write call or
	// remains constant in an active state. For example, some devices have multiple
	// slaves which means you want to assert on every call to make sure you are
	// targeting the tft. The default = true.
	ConstantCSAssert bool

	bitOrder BitOrder
}

// NewSoftSPI creates an SPI FTDI component
func NewSoftSPI(vender, product int, disableDrivers bool) *SoftSPI {
	spi := new(SoftSPI)

	spi.ConstantCSAssert = false

	spi.ftdi = new(ftdi.FTDI232H)

	spi.ftdi.SetTarget(vender, product)

	err := spi.ftdi.Initialize(disableDrivers)

	if err != nil {
		log.Fatal(err)
		return nil
	}

	return spi
}

// Configure sets up pins and various stuff
func (sopi *SoftSPI) Configure(maxSpeed int, bitOrder BitOrder) error {
	err := sopi.ftdi.SoftConfigure(false)
	if err != nil {
		log.Println("SPI failed to configure.")
		return err
	}

	sopi.CSActiveLow = true // Default for SPI protocol

	sopi.clk = ftdi.D0
	sopi.miso = ftdi.D1
	sopi.mosi = ftdi.D2
	sopi.cs = ftdi.D3
	sopi.rst = ftdi.D4
	sopi.trig = ftdi.D7

	sopi.bitOrder = bitOrder

	pins := []gpio.PinConfiguration{
		{Pin: sopi.clk, Direction: gpio.Output, Value: gpio.Low},
		{Pin: sopi.miso, Direction: gpio.Input, Value: gpio.Z},
		{Pin: sopi.mosi, Direction: gpio.Output, Value: gpio.Low},
		{Pin: sopi.cs, Direction: gpio.Output, Value: gpio.High},
		{Pin: sopi.rst, Direction: gpio.Output, Value: gpio.High},
		{Pin: sopi.trig, Direction: gpio.Output, Value: gpio.Low},
	}
	// In Bitbang mode the directions are ignored. You can read all the
	// pins at once regardless of direction.
	sopi.ConfigPins(pins)

	// Initialize clock, mode, and bit order.
	// log.Printf("SPI Setting clock speed to (%d)MHz\n", maxSpeed/1000000)
	sopi.SetClock(maxSpeed)

	// log.Println("SPI Setting bit order")
	sopi.SetBitOrder(bitOrder)

	fmt.Printf("Default pin values: %08b\n", sopi.pins)
	sopi.ftdi.WriteByte(sopi.pins)

	// Give time for the GPIO pins to stablize.
	time.Sleep(time.Millisecond)

	return nil
}

// Close closes the FTDI232 device
func (sopi *SoftSPI) Close() error {
	log.Println("Soft SPI closing FTDI device")
	err := sopi.ftdi.Close()

	if err != nil {
		return err
	}

	log.Println("FTDI closed")
	return nil
}

// SetClock sets the speed of the SPI clock in hertz.  Note that not all speeds
// are supported and a lower speed might be chosen by the hardware.
func (sopi *SoftSPI) SetClock(hz int) {
	sopi.ftdi.SetClock(hz, false, false)
}

// SetBitOrder sets the order of bits to be read/written over serial lines.  Should be
// either MSBFIRST for most-significant first, or LSBFIRST for
// least-signifcant first.
func (sopi *SoftSPI) SetBitOrder(order BitOrder) {
	sopi.bitOrder = order
}

func (sopi *SoftSPI) ConfigPins(pins []gpio.PinConfiguration) {
	for _, o := range pins {
		if o.Value != gpio.Z {
			sopi.setPin(o.Pin, o.Value)
		}
	}
}

// Write sends a byte-bit sequence out the MOSI pin.
// This is a Half-duplex SPI write.
func (sopi *SoftSPI) Write(data byte) error {
	// fmt.Printf("Writing: %x, %08b\n", data, data)

	// Send bits 7..0 on MOSI pin
	for i := 0; i < 8; i++ {
		bit := data & 0x80
		if bit > 0 {
			// fmt.Printf("High-")
			sopi.setHigh(sopi.mosi)
			sopi.ftdi.WriteByte(sopi.pins)
		} else {
			// fmt.Printf("Low-")
			sopi.setLow(sopi.mosi)
			sopi.ftdi.WriteByte(sopi.pins)
		}

		// Pulse clk-pin to indicate bit value should be sampled/read.
		sopi.setLow(sopi.clk)
		sopi.ftdi.WriteByte(sopi.pins)
		sopi.setHigh(sopi.clk)
		sopi.ftdi.WriteByte(sopi.pins)

		data = data << 1
		// fmt.Printf("data: %x, %08b, %d\n", data, data, bit)
	}
	// if err != nil {
	// 	return err
	// }
	sopi.setLow(sopi.clk)
	sopi.ftdi.WriteByte(sopi.pins)

	return nil
}

// Half-duplex SPI read. The specified length of bytes will be clocked
// in the MISO line and returned as a bytearray object.
func (sopi *SoftSPI) Read() (byte, error) {

	// Read response bytes.
	response, err := sopi.ftdi.PollRead(1, -1)
	fmt.Printf("Soft read: %v\n", response)

	return response[0], err
}

// IsPinHigh returns true is pin is High and false for Low.
func (sopi *SoftSPI) IsPinHigh(pin gpio.Pin) bool {
	return sopi.pins&(1<<pin) == 1
}

// TogglePin toggles pin to the opposite state.
func (sopi *SoftSPI) TogglePin(pin gpio.Pin) {
	// Capture original state
	if sopi.IsPinHigh(pin) {
		sopi.setLow(pin)
	} else {
		sopi.setHigh(pin)
	}

	sopi.ftdi.WriteByte(sopi.pins)
}

// PulsePin toggles the pin and leaves it in its original state.
func (sopi *SoftSPI) PulsePin(pin gpio.Pin) {
	// Capture original state
	sopi.TogglePin(pin)
	sopi.TogglePin(pin)
}

func (sopi *SoftSPI) SetReset(state bool) {
	if state {
		sopi.setHigh(sopi.rst)
	} else {
		sopi.setLow(sopi.rst)
	}
	sopi.ftdi.WriteByte(sopi.pins)
}

// Transfer is a Full-duplex SPI read and write.  The specified array of bytes will be
// clocked out the MOSI line, while simultaneously bytes will be read from
// the MISO line.  Read bytes will be returned as a bytearray object.
// transferCommand could be a value of 0x30 for most devices.
func (sopi *SoftSPI) Transfer(data byte) (byte, error) {
	var err error
	var response byte

	// sopi.Write(data)
	for i := 0; i < 8; i++ {
		bit := data & 0x80
		if bit > 0 {
			// fmt.Printf("High-")
			sopi.setHigh(sopi.mosi)
			sopi.ftdi.WriteByte(sopi.pins)
		} else {
			// fmt.Printf("Low-")
			sopi.setLow(sopi.mosi)
			sopi.ftdi.WriteByte(sopi.pins)
		}

		// Pulse clk-pin to indicate bit value should be sampled/read.
		sopi.setHigh(sopi.clk)
		sopi.ftdi.WriteByte(sopi.pins)

		response, err = sopi.ftdi.PinsRead()
		if err != nil {
			return 0, err
		}

		sopi.setLow(sopi.clk)
		sopi.ftdi.WriteByte(sopi.pins)

		data = data << 1
		// fmt.Printf("data: %x, %08b, %d\n", data, data, bit)
	}

	// Read response bytes.
	// response, err := sopi.ftdi.PollRead(1, -1)

	// if err != nil {
	// 	log.Printf("SPI: Transfer pollread failed on data (%v)\n", data)
	// 	log.Println(err)
	// }

	return response, nil
}

// AssertChipSelect will toggle chip select low or high depending on Active configuration
func (sopi *SoftSPI) AssertChipSelect() {
	// log.Println("SPI asserting chip select")
	if sopi.CSActiveLow {
		sopi.setLow(sopi.cs)
	} else {
		sopi.setHigh(sopi.cs)
	}
	sopi.ftdi.WriteByte(sopi.pins)
}

// DeAssertChipSelect will toggle chip select low or high depending on Active configuration
func (sopi *SoftSPI) DeAssertChipSelect() {
	// log.Println("SPI DE-asserting chip select")
	if sopi.CSActiveLow {
		sopi.setHigh(sopi.cs)
	} else {
		sopi.setLow(sopi.cs)
	}
	sopi.ftdi.WriteByte(sopi.pins)
}

func (sopi *SoftSPI) setPin(pin gpio.Pin, state gpio.PinState) {
	if state == 1 {
		sopi.setHigh(pin)
	} else {
		sopi.setLow(pin)
	}
}

func (sopi *SoftSPI) setLow(pin gpio.Pin) {
	sopi.pins &= ^(1 << pin) & 0xff
}

func (sopi *SoftSPI) setHigh(pin gpio.Pin) {
	sopi.pins |= (1 << pin) & 0xff
}

// ----------------------------------------------------------------------------------
// Debug stuff
// ----------------------------------------------------------------------------------

// TriggerPulse generate a timed pulse for various tools, ex Logic analyser.
func (sopi *SoftSPI) TriggerPulse() {
	// log.Println("SPI: Triggering pulse")
	sopi.TogglePin(sopi.trig)
}
