package ftdi

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"syscall"
	"time"

	"github.com/wdevore/hardware/gpio"

	"github.com/ziutek/ftdi"
)

const chunkSize = 65536 // bytes

// -----------------------------------------------------------------------------
// Pins
// -----------------------------------------------------------------------------
const (
	D0 gpio.Pin = iota
	D1
	D2
	D3
	D4
	D5
	D6
	D7
	C0
	C1
	C2
	C3
	C4
	C5
	C6
	C7
	// Rarely used pins. Requires EEPROM modifications.
	C8
	C9
)

// -----------------------------------------------------------------------------
// Commands
// -----------------------------------------------------------------------------
const (
	disableClockDivisor     = 0x8a
	enableAdaptiveClocking  = 0x96
	disableAdaptiveClocking = 0x97
	enable3PhaseClk         = 0x8c
	disable3PhaseClk        = 0x8d
)

var (
	commandReadHighLowBytes        = []byte{0x81, 0x83}
	commandBad                     = []byte{0xab}
	commandDisableClockDivisor     = []byte{disableClockDivisor}
	commandEnableAdaptiveClocking  = []byte{enableAdaptiveClocking}
	commandDisableAdaptiveClocking = []byte{disableAdaptiveClocking}
	commandEnable3PhaseClk         = []byte{enable3PhaseClk}
	commandDisable3PhaseClk        = []byte{disable3PhaseClk}
	commandUpdatePins              = []byte{0x80, 0, 0, 0x82, 0, 0}
	commandBasicSPIConfig          = []byte{disableClockDivisor, disableAdaptiveClocking, disable3PhaseClk}
	commandSetDivisor              = []byte{0x86, 0, 0}
)

// FTDI232H represents the Adafruit USB to GPIO breakout board.
// Adafruit part number is: P2264
//
// The board exposes all 16 io pins: D0->D7 and C0->C7
type FTDI232H struct {
	Vender  int
	Product int

	// If SleepingPoll = true then a 1 microsecond sleep occurs between each buffer
	// read while polling. Default is `False`
	SleepingPoll bool

	device *ftdi.Device

	// A 16 bit register representing the direction of each io pin.
	direction uint16
	// A 16 bit register representing the level/state of each io pin.
	level uint16

	driversUnloaded bool

	// A buffer used for reading pins configured as Input.
	chunk []byte

	// Track if the expected bytes being requested has changed value.
	// This is a simple memory allocation optimization.
	prevExpected int
	response     []byte
}

// NewFTDI232H creates and configures FTDI.
// if disableDrivers = true then you need to run as root.
// [vendor] is typically 0x0403
// There are several products, for example: 0x6014 = FT232H
func NewFTDI232H(vender, product int) *FTDI232H {
	f := new(FTDI232H)
	f.SetTarget(vender, product)
	return f
}

// Initialize optionally disables any conflicting drivers.
func (f *FTDI232H) Initialize(disableDrivers bool) error {
	f.driversUnloaded = false

	if disableDrivers {
		if !f.isRunningAsRoot() {
			log.Fatal("To disable drivers this program must be run as root.")
			return nil
		}

		err := f.disableDrivers(true)
		if err != nil {
			log.Fatal(err.Error())
			return nil
		}

		f.driversUnloaded = true
	}

	return nil
}

// SetTarget sets device identities.
func (f *FTDI232H) SetTarget(vender, product int) {
	f.Vender = vender
	f.Product = product
}

// Configure arranges default values for MPSSE.
func (f *FTDI232H) Configure(sleepingPoll bool) error {
	// We need to open the device now so we can configure various property below.
	err := f.OpenFirst()
	if err != nil {
		return err
	}

	// Change read & write buffers to maximum size
	f.device.SetReadChunkSize(chunkSize)
	f.device.SetWriteChunkSize(chunkSize)

	// Pre allocate static read buffer size.
	f.chunk = make([]byte, chunkSize)

	f.SleepingPoll = sleepingPoll

	// log.Println("FTDI232H Enabling MPSSE")
	f.EnableMPSSE()

	// log.Println("FTDI232H setting default clock, adaptive disabled, 3phase disabled")
	f.SetClock(20000000, false, false)

	log.Println("FTDI232H MPSSE syncing")
	err = f.mpsseSync(-1)

	if err != nil {
		log.Println("FTDI232H MPSSE failed to sync")
		return err
	}

	log.Println("FTDI232H MPSSE synced")

	return nil
}

// Close shutdowns and reload any drivers
func (f *FTDI232H) Close() error {
	log.Println("FTDI232H closing device")
	err := f.device.Close()
	if err != nil {
		log.Fatal(err)
		return err
	}

	if f.driversUnloaded {
		err = f.disableDrivers(false)
		if err != nil {
			log.Fatal(err.Error())
			return err
		}
	}

	log.Println("FTDI232H device closed")

	return nil
}

func (f *FTDI232H) isRunningAsRoot() bool {
	usr, _ := user.Current()
	return usr.Uid == "0"
}

// Note: if you have authorization correctly configured via udev rules.d you don't
// need to unload the drivers--at least under linux.
func (f *FTDI232H) disableDrivers(disable bool) error {
	env := os.Environ()
	var argsFtdi []string
	var argsUsb []string
	var command string

	if runtime.GOOS == "darwin" {
		// Mac OS commands to disable FTDI driver.
		command = "kextunload"

		argsFtdi = []string{command, "-b com.apple.driver.AppleUSBFTDI"}
		argsUsb = []string{command, "/System/Library/Extensions/FTDIUSBSerialDriver.kext"}
	} else if runtime.GOOS == "linux" {
		command := "modprobe"

		if disable {
			argsFtdi = []string{command, "-r", "-q ftdi_sio"}
			argsUsb = []string{command, "-r", "-q usbserial"}
		} else {
			argsFtdi = []string{command, "-q ftdi_sio"}
			argsUsb = []string{command, "-q usbserial"}
		}
	}

	binaryPath, err := exec.LookPath(command)

	if err != nil {
		return err
	}

	execErr := syscall.Exec(binaryPath, argsFtdi, env)

	if execErr != nil {
		return execErr
	}

	execErr = syscall.Exec(binaryPath, argsUsb, env)

	if execErr != nil {
		return execErr
	}

	return nil
}

// Open opens the first device on a specific channel.
func (f *FTDI232H) Open(channel ftdi.Channel) error {
	d, err := ftdi.OpenFirst(f.Vender, f.Product, channel)
	if err != nil {
		return err
	}
	f.device = d

	return nil
}

// OpenFirst opens the first known FTDI device
func (f *FTDI232H) OpenFirst() error {
	d, err := ftdi.OpenFirst(f.Vender, f.Product, ftdi.ChannelAny)
	if err != nil {
		return err
	}
	f.device = d

	return nil
}

// ------------------------------------------------------------------------
// GPIO
// ------------------------------------------------------------------------

func (f *FTDI232H) setPin(pin gpio.Pin, mode gpio.IODirection) {
	if mode == gpio.Input {
		// Set the direction and level of the pin to 0.
		f.direction &= ^(1 << pin) & 0xffff
		f.level &= ^(1 << pin) & 0xffff
	} else {
		// Set the direction of the pin to 1.
		f.direction |= (1 << pin) & 0xffff
	}
}

// Set the specified pin HIGH.
// Note: This does NOT write the value out to the device!
// Use OutputHigh for "setting" and "writing"
// This method allows you to set multiple pins and then perform
// a single write.
func (f *FTDI232H) setHigh(pin gpio.Pin) {
	f.SetPin(pin, gpio.High)
}

// Set the specified pin LOW.
// Note: This does NOT write the value out to the device!
// Use OutputLow for "setting" and "writing".
// This method allows you to set multiple pins and then perform
// a single write.
func (f *FTDI232H) setLow(pin gpio.Pin) {
	f.SetPin(pin, gpio.Low)
}

// SetConfigPin sets the input or output mode for a specified pin.  Mode should be
// either OUT or IN. Note: This does NOT write to the device.
func (f *FTDI232H) SetConfigPin(pin gpio.Pin, mode gpio.IODirection) {
	f.setPin(pin, mode)
}

// ConfigPin sets the input or output mode for a specified pin.  Mode should be
// either OUT or IN.
func (f *FTDI232H) ConfigPin(pin gpio.Pin, mode gpio.IODirection) {
	f.setPin(pin, mode)
	f.mpsseWriteGpio()
}

// ConfigPins and write out pins
func (f *FTDI232H) ConfigPins(pins []gpio.PinConfiguration, write bool) {
	for _, o := range pins {
		f.setPin(o.Pin, o.Direction)
		if o.Value != gpio.Z {
			f.SetPin(o.Pin, o.Value)
		}
	}

	if write {
		f.mpsseWriteGpio()
	}
}

// SetPin only sets the buffer pin value. It does NOT write the pin to the device.
func (f *FTDI232H) SetPin(pin gpio.Pin, value gpio.PinState) {
	if value == gpio.High {
		f.level |= (1 << pin) & 0xffff
	} else {
		f.level &= ^(1 << pin) & 0xffff
	}
}

// Output sets AND writes the specified pin to the provided high/low value.  Value should be
// either HIGH/LOW or a boolean (true = high).
func (f *FTDI232H) Output(pin gpio.Pin, value gpio.PinState) error {
	f.SetPin(pin, value)
	return f.mpsseWriteGpio()
}

// OutputHigh sets the pin High AND writes it to the device.
func (f *FTDI232H) OutputHigh(pin gpio.Pin) error {
	f.SetPin(pin, gpio.High)
	return f.mpsseWriteGpio()
}

// OutputLow sets the pin Low AND writes it to the device.
func (f *FTDI232H) OutputLow(pin gpio.Pin) error {
	f.SetPin(pin, gpio.Low)
	return f.mpsseWriteGpio()
}

// SetHigh sets the pin High ONLY.
func (f *FTDI232H) SetHigh(pin gpio.Pin) {
	f.SetPin(pin, gpio.High)
}

// SetLow sets the pin Low ONLY.
func (f *FTDI232H) SetLow(pin gpio.Pin) {
	f.SetPin(pin, gpio.Low)
}

// OutputPins takes an array of 16 States for output pins.
// Note: depending on mode some States will have no effect.
// func (f *FTDI232H) OutputPins(pins []State, write bool) {
// 	var pin Pin

// 	for _, o := range pins {
// 		f.outputPin(pin, o)
// 		pin++
// 	}

// 	if write {
// 		f.mpsseWriteGpio()
// 	}
// }

// ReadInput reads the specified pin and returns OutputHigh/true if the pin is pulled high,
// or OutputLow/false if pulled low.
func (f *FTDI232H) ReadInput(pin gpio.Pin) gpio.PinState {
	inPins := f.mpsseReadGpio()
	st := (inPins >> pin) & 0x0001
	if st == 1 {
		return gpio.High
	}
	return gpio.Low

}

// WriteByte wraps byte in a slice, then writes.
func (f *FTDI232H) WriteByte(data byte) (int, error) {
	return f.Write([]byte{data})
}

// Write writes out a byte array of size determined by the array
func (f *FTDI232H) Write(data []byte) (int, error) {
	writtenCnt, err := f.device.Write(data)

	if err != nil {
		// log.Printf("FTDI232H Write failed: %v", err)
		return 0, err
	}

	if writtenCnt != len(data) {
		msg := fmt.Sprintf("Expected to write (%d) bytes, however, only (%d) written\n", len(data), writtenCnt)
		err := errors.New(msg)
		log.Fatal(err)
		return 0, err
	}

	// log.Printf("FTDI232H Wrote (%d) bytes\n", writtenCnt)

	return writtenCnt, nil
}

// WriteLen allows writing of variable length fixed size arrays.
// Reduces memory allocations
func (f *FTDI232H) WriteLen(data []byte, length int) (int, error) {
	writtenCnt, err := f.device.WriteLen(data, length)

	if err != nil {
		// log.Printf("FTDI232H Write failed: %v", err)
		return 0, err
	}

	if writtenCnt != length {
		msg := fmt.Sprintf("Expected to write (%d) bytes, however, only (%d) written\n", len(data), writtenCnt)
		err := errors.New(msg)
		log.Fatal(err)
		return 0, err
	}

	// log.Printf("FTDI232H Wrote (%d) bytes\n", writtenCnt)

	return writtenCnt, nil
}

// PollRead reads an expected number of bytes by polling for them.
// [timeout] is specified in seconds. If [timeout] == -1 then timeout = 10 seconds
func (f *FTDI232H) PollRead(expected int, timeout int64) ([]byte, error) {
	// Function to continuously poll reads on the FTDI device until an
	// expected number of bytes are returned.  Will throw a timeout error if no
	// data is received within the specified number of timeout seconds.  Returns
	// the read data as a string if successful, otherwise raises an execption.
	start := time.Now()
	if timeout < 0 {
		timeout = 3 // 3 seconds
	}
	duration := time.Duration(timeout) * time.Second
	// fmt.Printf("FTDI232H pollRead polling for (%d)s ...\n", duration/time.Second)

	iResp := 0

	// Reallocate if the expected buffer size is changing.
	if f.prevExpected != expected {
		// Start with an empty response buffer.
		f.response = make([]byte, expected)
		f.prevExpected = expected
	}

	// Loop calling read until the response chunk buffer is full or a timeout occurs.
	for time.Now().Sub(start) <= duration {
		// log.Println("FTDI232H reading device")
		bytesRead, err := f.device.Read(f.chunk)
		if err != nil {
			return nil, err
		}

		// The response buffer is of fixed size. We copy bytes until the
		// response buffer is filled or we copied the chunk.
		// Copy as long as we haven't exceeded the response buffer or we haven't
		// copied all the chunk bytes.
		if bytesRead > 0 {
			// fmt.Printf("FTDI232H BytesRead (%d): ", bytesRead)
			for iChunk := 0; iChunk < bytesRead; iChunk++ {
				// fmt.Printf("%#x,", f.chunk[iChunk])
				if iResp >= expected {
					break
				}
				// Copy/append another byte into response
				f.response[iResp] = f.chunk[iChunk]
				iResp++
			}
		}

		if iResp >= expected {
			// fmt.Printf("FTDI232H Got (%d) bytes.\n", iResp)
			// We received all the expected bytes within the duration.
			return f.response, nil
		}

		if f.SleepingPoll {
			// log.Println("FTDI232H pollRead sleeping...")
			time.Sleep(time.Millisecond)
		}
	}

	msg := fmt.Sprintf("Timedout while polling for (%d) bytes!\n", expected)

	return nil, errors.New(msg)
}

// ------------------------------------------------------------------------
// MPSSE
// ------------------------------------------------------------------------

// EnableMPSSE enables MPSSE mode
func (f *FTDI232H) EnableMPSSE() {
	err := f.device.SetBitmode(0xff, ftdi.ModeMPSSE)
	if err != nil {
		log.Fatal(err)
	}
}

func (f *FTDI232H) mpsseGpio() {
	// Update command to change the MPSSE GPIO state to the current directions
	// and levels.

	// lower 8 bits
	commandUpdatePins[1] = byte(f.level & 0xff)     // levelLow
	commandUpdatePins[2] = byte(f.direction & 0xff) // dirLow

	// upper 8 bits
	commandUpdatePins[4] = byte((f.level >> 8) & 0xff)     // levelHigh
	commandUpdatePins[5] = byte((f.direction >> 8) & 0xff) // dirHigh
}

// Write the current MPSSE GPIO state to the FT232H chip.
func (f *FTDI232H) mpsseWriteGpio() error {
	f.mpsseGpio()
	_, err := f.Write(commandUpdatePins)
	return err
}

// if [maxRetries] < 0 then default to 10.
func (f *FTDI232H) mpsseSync(maxRetries int) error {
	// Synchronize buffers with MPSSE by sending bad opcode and reading expected
	// error response.  Should be called once after enabling MPSSE.

	// Send a bad/unknown command (0xab), then read buffer until bad command
	// response is found.
	// log.Println("FTDI232H mpsseSync writing bad command")
	_, err := f.Write(commandBad)

	if err != nil {
		return err
	}

	if maxRetries < 0 {
		maxRetries = 10
	}

	// Keep reading until bad command response (0xfa 0xab) is returned.
	// Fail if too many read attempts are made to prevent sticking in a loop.
	tries := 0
	sync := false

	for !sync {
		data, err := f.PollRead(2, -1)
		if err != nil {
			log.Println("FTDI232H mpsseSync pollRead failed.")
			return err
		}

		if data[0] == 0xfa && data[1] == 0xab {
			// log.Println("FTDI232H mpsseSync finaly sunk")
			sync = true
		} else {
			tries++
			log.Printf("FTDI232H mpsseSync trying again: %d", tries)

			if tries >= maxRetries {
				return errors.New("could not synchronize with FT232H")
			}
		}
	}

	return nil
}

// SetClock sets the clock speed.
// [adaptive] has a typical default of `false`, [threePhase] is typically `false`
// [clock] is specified in Hertzs (Hz)
// Set the clock speed of the MPSSE engine.  Can be any value from 450hz
// to 30mhz and will pick that speed or the closest speed below it.
func (f *FTDI232H) SetClock(clock int, adaptive, threePhase bool) {

	// ----------------------------------------------------------
	// Could issue each command on a separate "write"
	// Disable clock divisor by 5 to enable faster speeds on FT232H.
	// f.write(commandDisableClockDivisor)

	// Turn on/off adaptive clocking.
	// if adaptive {
	// 	f.write(commandEnableAdaptiveClocking)
	// } else {
	// 	f.write(commandDisableAdaptiveClocking)
	// }

	// Turn on/off three phase clock (needed for I2C).
	// Also adjust the frequency for three-phase clocking as specified in section 2.2.4
	// of this document:
	// http://www.ftdichip.com/Support/Documents/AppNotes/AN_255_USB%20to%20I2C%20Example%20using%20the%20FT232H%20and%20FT201X%20devices.pdf
	// if threePhase {
	// 	f.write(commandEnable3PhaseClk)
	// } else {
	// 	f.write(commandDisable3PhaseClk)
	// }
	// ----------------------------------------------------------
	// Or
	// We issue all commands with one write as done below:
	f.Write(commandBasicSPIConfig)

	// Compute divisor for requested clock.
	// Use equation from section 3.8.1 of:
	//  http://www.ftdichip.com/Support/Documents/AppNotes/AN_108_Command_Processor_for_MPSSE_and_MCU_Host_Bus_Emulation_Modes.pdf
	// Note equation is using 60mhz master clock instead of 12mhz.
	divisor := int(math.Ceil((30000000.0-float64(clock))/float64(clock))) & 0xffff
	if threePhase {
		divisor = int(float64(divisor) * float64(2.0/3.0))
		// logger.debug('Setting clockspeed with divisor value {0}'.format(divisor))
	}

	// Send command to set divisor from low and high byte values.
	commandSetDivisor[1] = byte(divisor & 0xff)        // low byte
	commandSetDivisor[2] = byte((divisor >> 8) & 0xff) // high byte

	f.Write(commandSetDivisor)
}

func (f *FTDI232H) mpsseReadGpio() gpio.Pins {
	// Read both GPIO bus states and return a 16 bit value with their state.
	// D0-D7 are the lower 8 bits and C0-C7 are the upper 8 bits.

	// Send command to read low byte and high byte.
	f.Write(commandReadHighLowBytes)

	// Wait for 2 byte response.
	data, err := f.PollRead(2, -1)
	if err != nil {
		log.Fatal(err)
	}

	// Assemble response into 16 bit value.
	lowByte := uint16(data[0])
	highByte := uint16(data[1]) << 8

	// logger.debug('Read MPSSE GPIO low byte = {0:02X} and high byte = {1:02X}'.format(low_byte, high_byte))

	return gpio.Pins(highByte | lowByte)
}

func (f FTDI232H) String() string {
	s := "\n"
	s += "          111111\n"
	s += "0123456789012345\n"
	s += fmt.Sprintf("%s : Direction\n%s : Level\n, Response: %v", uint16ToBinaryString(f.direction), uint16ToBinaryString(f.level), f.response)
	return s
}

// ToStringFullBinary returns a full report of the component
func (f *FTDI232H) ToStringFullBinary() string {
	s := "\n"
	s += "          111111\n"
	s += "0123456789012345\n"
	s += fmt.Sprintf("%s : Direction\n%s : Level\n", uint16ToBinaryString(f.direction), uint16ToBinaryString(f.level))
	s += fmt.Sprintf("Response:\n 76543210 76543210\n[")
	for _, o := range f.response {
		s += fmt.Sprintf("%s,", byteToBinaryString(o))
	}
	s += fmt.Sprintf("] : [%v]\n", f.response)

	return s
}

// ------------------------------------------------------------------------
// Simple helper utilities
// ------------------------------------------------------------------------
func byteToBinaryString(v byte) string {
	s := fmt.Sprintf("%08b", v)
	return s
}

func uint16ToBinaryString(v uint16) string {
	s := fmt.Sprintf("%016b", v)
	return s
}
