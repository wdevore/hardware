package hx8357

import (
	"fmt"
	"log"

	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/gpio"
)

var (
	// Init for 7735R, part 1 (red or green tab)
	rcmd1 = []commando{
		{Command: SWRESET, Delayed: true, //  1: Software reset, 0 args, w/delay
			Args:  nil,
			Delay: 10},

		{Command: HX8357D_SETC, Delayed: true, // set extc
			Args: []byte{
				0xFF, 0x83, 0x57},
			Delay: 300},

		{Command: SETRGB, Delayed: false, // setRGB which also enables SDO
			Args: []byte{
				0x00, //enable SDO pin! disable = 0x00 (aka 3-wire), enable = 0x80 (aka 4-wire)
				0x0, 0x06, 0x06},
			Delay: 0},

		{Command: HX8357D_SETCOM, Delayed: false,
			Args:  []byte{0x25}, // -1.52V
			Delay: 0},

		{Command: SETOSC, Delayed: false,
			Args:  []byte{0x68}, // Normal mode 70Hz, Idle mode 55 Hz
			Delay: 0},

		{Command: SETPANEL, Delayed: false, //Set Panel
			Args:  []byte{0x05}, // BGR, Gate direction swapped
			Delay: 0},

		{Command: SETPWR1, Delayed: false, //  Power control
			Args: []byte{
				0x00, // Not deep standby
				0x15, // BT
				0x1C, // VSPR
				0x1C, // VSNR
				0x83, // AP
				0xAA, // FS
			},
			Delay: 0},

		{Command: HX8357D_SETSTBA, Delayed: false,
			Args: []byte{
				0x50, // OPON normal
				0x50, // OPON idle
				0x01, // STBA
				0x3C, // STBA
				0x1E, // STBA
				0x08, // GEN
			},
			Delay: 0},

		{Command: HX8357D_SETCYC, Delayed: false,
			Args: []byte{
				0x02, // NW 0x02
				0x40, // RTN
				0x00, // DIV
				0x2A, // DUM
				0x2A, // DUM
				0x0D, // GDON
				0x78, // GDOFF
			},
			Delay: 0},

		{Command: HX8357D_SETGAMMA, Delayed: false,
			Args: []byte{
				0x02, 0x0A, 0x11, 0x1d, 0x23,
				0x35, 0x41, 0x4b, 0x4b, 0x42,
				0x3A, 0x27, 0x1B, 0x08, 0x09,
				0x03, 0x02, 0x0A, 0x11, 0x1d,
				0x23, 0x35, 0x41, 0x4b, 0x4b,
				0x42, 0x3A, 0x27, 0x1B, 0x08,
				0x09, 0x03, 0x00, 0x01,
			},
			Delay: 0},

		{Command: COLMOD, Delayed: false,
			// 1st byte = RRRRRGGG
			// 2nd byte = GGGBBBBB
			Args:  []byte{0x05}, // 16 bit, 65K-Color 1-pixel/2-transfers
			Delay: 0},

		{Command: MADCTL, Delayed: false,
			Args:  []byte{0xC0},
			Delay: 0},

		{Command: TEON, Delayed: false, // TE off
			Args:  []byte{0x00},
			Delay: 0},

		{Command: TEARLINE, Delayed: false, // tear line
			Args:  []byte{0x00, 0x02},
			Delay: 0},

		{Command: SLPOUT, Delayed: true, //Exit Sleep
			Args:  nil,
			Delay: 150},

		{Command: DISPON, Delayed: true, // display on
			Args:  nil,
			Delay: 50},
	}
)

// HX8357D represents the D style display
type HX8357D struct {
	HX8357
}

// NewHX8357D creates a variant of ST7735
func NewHX8357D(dataCommand, reset gpio.Pin, tab devices.TabColor, dimensions devices.Dimensions) *HX8357D {
	hx := new(HX8357D)

	hx.dc = dataCommand
	hx.reset = reset
	hx.tab = tab
	hx.dimensions = dimensions

	return hx
}

// Initialize configures and initializes HX8357D
// Depending on how you have physically oriented the display device the xy origin
// will be located differently.
// With the connections pins situated at the bottom orientation produces:
// 0 = fills from bottom to top
// 1 = left to right
// 2 = top to bottom
// decreasing x moves left
// decreasing y moves up
// Thus the origin is in the top-left
// 3 = right to left
func (hx *HX8357D) Initialize(vender, product, clockFreq int, chipSelect gpio.Pin, orientation devices.RotationMode) error {
	// Initialize the device
	err := hx.initialize(vender, product, clockFreq, chipSelect)
	if err != nil {
		return err
	}

	// I am not using the SD card on the device so I can leave CS low.
	// Of course if you are using a logic analyzer with an SPI decoder activated
	// the decoder won't display anything because CS is part of the SPI
	// protocol and must assert on each piece of data trafficking.
	hx.SetConstantCSAssert(true)

	log.Println("Issusing init commands")
	err = hx.commonInit(rcmd1)
	if err != nil {
		return err
	}

	switch hx.dimensions {
	case devices.D320x480:
		hx.Width = 320
		hx.Height = 480
	case devices.D480x320:
		hx.Width = 480
		hx.Height = 320

	}

	pixels := int(hx.Width) * int(hx.Height)

	// A buffer of bytes
	// RRRRRGGG-GGGBBBBB RRRRRGGG-GGGBBBBB...

	hx.pushBuffer = make([]byte, pixels*bytesPerPixel)

	hx.lineBlockSize = 5
	hx.lines = int(hx.Height) / hx.lineBlockSize // 48 * 10 = 480
	hx.chunkSize = int(hx.Width) * bytesPerPixel * hx.lines

	fmt.Printf("lineBlock %d, chunksize: %d\n", hx.lines, hx.chunkSize)
	hx.chunkBuf = make([]byte, hx.chunkSize)

	if orientation == devices.OrientationDefault {
		hx.SetRotation(devices.Orientation2)
	} else {
		hx.SetRotation(orientation)
	}

	return nil
}

// Close closes the device.
func (hx *HX8357D) Close() error {
	log.Println("HX8357D closing...")
	err := hx.close()
	if err != nil {
		log.Println("Failed to close ST7735R.")
		log.Fatal(err)
	}

	log.Println("HX8357D closed.")

	return nil
}
