package st7735

import (
	"log"

	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/gpio"
)

var (
	// Init for 7735R, part 1 (red or green tab)
	rcmd1 = []commando{
		{Command: SWRESET, Delayed: true, //  1: Software reset, 0 args, w/delay
			Args:  nil,
			Delay: 150}, //     150 ms delay

		{Command: SLPOUT, Delayed: true, //  2: Out of sleep mode, 0 args, w/delay
			Args:  nil,
			Delay: 500}, //     500 ms delay

		{Command: FRMCTR1, Delayed: false, //  3: Frame rate ctrl - normal mode, 3 args:
			Args:  []byte{0x01, 0x2C, 0x2D}, //     Rate = fosc/(1x2+40) * (LINE+2C+2D)
			Delay: 0},

		{Command: FRMCTR2, Delayed: false, //  4: Frame rate control - idle mode, 3 args:
			Args:  []byte{0x01, 0x2C, 0x2D}, //     Rate = fosc/(1x2+40) * (LINE+2C+2D)
			Delay: 0},

		{Command: FRMCTR3, Delayed: false, //  5: Frame rate ctrl - partial mode, 6 args:
			Args: []byte{
				0x01, 0x2C, 0x2D, //     Dot inversion mode
				0x01, 0x2C, 0x2D}, //     Line inversion mode
			Delay: 0},

		{Command: INVCTR, Delayed: false, //  6: Display inversion ctrl, 1 arg, no delay:
			Args:  []byte{0x07}, //     No inversion
			Delay: 0},

		{Command: PWCTR1, Delayed: false, //  7: Power control, 3 args, no delay:
			Args: []byte{
				0xA2,
				0x02,  //     -4.6V
				0x84}, //     AUTO mode
			Delay: 0},

		{Command: PWCTR2, Delayed: false, //  8: Power control, 1 arg, no delay:
			Args:  []byte{0xC5}, //     VGH25 = 2.4C VGSEL = -10 VGH = 3 * AVDD
			Delay: 0},

		{Command: PWCTR3, Delayed: false, //  9: Power control, 2 args, no delay:
			Args: []byte{
				0x0A,  //     Opamp current small
				0x00}, //     Boost frequency
			Delay: 0},

		{Command: PWCTR4, Delayed: false, // 10: Power control, 2 args, no delay:
			Args: []byte{
				0x8A, //     BCLK/2, Opamp current small & Medium low
				0x2A},
			Delay: 0},

		{Command: PWCTR5, Delayed: false, // 11: Power control, 2 args, no delay:
			Args:  []byte{0x8A, 0xEE},
			Delay: 0},

		{Command: VMCTR1, Delayed: false, // 12: Power control, 1 arg, no delay:
			Args:  []byte{0x0E},
			Delay: 0},

		{Command: INVOFF, Delayed: false, // 13: Don't invert display, no args, no delay
			Args:  nil,
			Delay: 0},

		{Command: MADCTL, Delayed: false, // 14: Memory access control (directions), 1 arg:
			Args:  []byte{0xC8}, //     row addr/col addr, bottom to top refresh
			Delay: 0},

		{Command: COLMOD, Delayed: false, // 15: set color mode, 1 arg, no delay:
			Args:  []byte{0x05}, //     16-bit color
			Delay: 0},
	}

	// Init for 7735R, part 2 (green tab only)
	rcmd2green = []commando{
		{Command: CASET, Delayed: false, //  1: Column addr set, 4 args, no delay:
			Args: []byte{
				0x00, 0x02, //     XSTART = 0
				0x00, 0x7F + 0x02}, //     XEND = 127
			Delay: 0},

		{Command: RASET, Delayed: false, //  2: Row addr set, 4 args, no delay:
			Args: []byte{
				0x00, 0x01, //     XSTART = 0
				0x00, 0x9F + 0x01}, //     XEND = 159
			Delay: 0},
	}

	// Init for 7735R, part 2 (red tab only)
	rcmd2red = []commando{
		{Command: CASET, Delayed: false, //  1: Column addr set, 4 args, no delay:
			Args: []byte{
				0x00, 0x00, //     XSTART = 0
				0x00, 0x7F}, //     XEND = 127
			Delay: 0},

		{Command: RASET, Delayed: false, //  2: Row addr set, 4 args, no delay:
			Args: []byte{
				0x00, 0x00, //     XSTART = 0
				0x00, 0x9F}, //     XEND = 159
			Delay: 0},
	}

	// Init for 7735R, part 2 (green 1.44 tab)
	rcmd2green144 = []commando{
		{Command: CASET, Delayed: false, //  1: Column addr set, 4 args, no delay:
			Args: []byte{
				0x00, 0x00, //     XSTART = 0
				0x00, 0x7F}, //     XEND = 127
			Delay: 0},

		{Command: RASET, Delayed: false, //  2: Row addr set, 4 args, no delay:
			Args: []byte{
				0x00, 0x00, //     XSTART = 0
				0x00, 0x7F}, //     XEND = 127
			Delay: 0},
	}

	// Init for 7735R, part 2 (mini 160x80)
	rcmd2green160x80 = []commando{
		{Command: CASET, Delayed: false, //  1: Column addr set, 4 args, no delay:
			Args: []byte{
				0x00, 0x00, //     XSTART = 0
				0x00, 0x7F}, //     XEND = 79
			Delay: 0},

		{Command: RASET, Delayed: false, //  2: Row addr set, 4 args, no delay:
			Args: []byte{
				0x00, 0x00, //     XSTART = 0
				0x00, 0x9F}, //     XEND = 159
			Delay: 0},
	}

	// Init for 7735R, part 3 (red or green tab)
	rcmd3 = []commando{
		{Command: GMCTRP1, Delayed: false, //  1: Magical unicorn dust, 16 args, no delay:
			Args: []byte{
				0x02, 0x1c, 0x07, 0x12,
				0x37, 0x32, 0x29, 0x2d,
				0x29, 0x25, 0x2B, 0x39,
				0x00, 0x01, 0x03, 0x10},
			Delay: 0},

		{Command: GMCTRN1, Delayed: false, //  2: Sparkles and rainbows, 16 args, no delay:
			Args: []byte{
				0x03, 0x1d, 0x07, 0x06,
				0x2E, 0x2C, 0x29, 0x2D,
				0x2E, 0x2E, 0x37, 0x3F,
				0x00, 0x00, 0x02, 0x10},
			Delay: 0},

		{Command: NORON, Delayed: true, //  3: Normal display on, no args, w/delay
			Args:  nil,
			Delay: 10}, //     10 ms delay

		{Command: DISPON, Delayed: true, //  4: Main screen turn on, no args w/delay
			Args:  nil,
			Delay: 100}, //     100 ms delay
	}
)

// ST7735R represents the R style ST7735 display
type ST7735R struct {
	// ST7735R is-a ST7735
	ST7735
}

// NewST7735R creates a variant of ST7735
func NewST7735R(dataCommand, reset gpio.Pin, tab devices.TabColor, dimensions devices.Dimensions) *ST7735R {
	st := new(ST7735R)

	st.dc = dataCommand
	st.reset = reset
	st.tab = tab
	st.dimensions = dimensions

	return st
}

// Initialize configures and initializes ST7735
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
func (st *ST7735R) Initialize(vender, product, clockFreq int, chipSelect gpio.Pin, orientation devices.RotationMode) error {
	// Initialize the ST7735 device
	err := st.initialize(vender, product, clockFreq, chipSelect)
	if err != nil {
		return err
	}

	// I am not using the SD card on the device so I can leave CS low.
	// Of course if you are using a logic analyzer with an SPI decoder activated
	// the decoder won't display anything because CS is part of the SPI
	// protocol and must assert on each piece of data trafficking.
	st.SetConstantCSAssert(true)

	// log.Println("ST7735R common init for rcmd1")
	err = st.commonInit(rcmd1)
	if err != nil {
		return err
	}

	if st.tab == devices.RedTab {
		st.colstart = 0
		st.rowstart = 0
		st.issueCommands(rcmd2red)
	} else {
		switch st.dimensions {
		case devices.D128x128:
			st.colstart = 2
			// Note: rowstart will be overridden if orientation is set.
			st.rowstart = 1 // Other code also sets this to 3.
			st.Width = 128
			st.Height = 128
			// log.Println("ST7735R issuing rcmd2green144")
			st.issueCommands(rcmd2green144)
			break
		case devices.D128x160:
			st.colstart = 2
			st.rowstart = 1
			st.Width = 128
			st.Height = 160
			st.issueCommands(rcmd2green)
			break
		case devices.D160x80:
			st.colstart = 24
			st.rowstart = 0
			st.Width = 160
			st.Height = 80
			st.issueCommands(rcmd2green160x80)
			break
		}
	}

	pixels := int(st.Width) * int(st.Height)
	// log.Printf("ST7735R offset screen buffer size: (%d) bytes\n", st.screenBufferSize)

	// A buffer of bytes
	st.pushBuffer = make([]byte, pixels*2)

	// log.Println("ST7735R issuing rcmd3")
	st.issueCommands(rcmd3)

	if orientation == devices.OrientationDefault {
		// log.Println("ST7735R setting orientation to default")
		st.SetRotation(devices.Orientation2)
	} else {
		// log.Println("ST7735R setting orientation")
		st.SetRotation(orientation)
	}

	return nil
}

// Close closes the device.
func (st *ST7735R) Close() error {
	log.Println("ST7735R closing...")
	err := st.close()
	if err != nil {
		log.Println("Failed to close ST7735R.")
		log.Fatal(err)
	}

	log.Println("ST7735R closed.")

	return nil
}
