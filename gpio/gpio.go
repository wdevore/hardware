package gpio

// Pins represents both A and B io banks
type Pins uint16

// Pin represents D0-D7 or C0-C7
type Pin uint

const (
	// NoPin indicates no pin is to be used
	NoPin Pin = 100001
	// DefaultPin indicates that a default pin be used.
	DefaultPin Pin = iota
	// HardwarePin indicates that the device will control the pin instead of software
	HardwarePin
)

// PinConfiguration specifies a pin configuration
type PinConfiguration struct {
	Pin       Pin
	Direction IODirection
	Value     PinState
}

// IODirection represents the pin mode and is: 0 = Out, 1 = In
type IODirection int

const (
	// Output configures pin as an output
	Output IODirection = 0
	// Input configures pin as an input
	Input IODirection = 1
)

// PinState represents the io value high/low/Z
type PinState int

const (
	// Low indicates a pin value to Low/Off
	Low PinState = 0
	// High indicates a pin value to High/On
	High PinState = 1
	// Z means pin is undefined or don't care
	Z PinState = 2
)
