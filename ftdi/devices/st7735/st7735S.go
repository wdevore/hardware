package st7735

import (
	"log"
	"time"

	"github.com/wdevore/hardware/ftdi"
	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/gpio"
)

// Scanning methods
const (
	L2R_U2D = iota // (default) The display interface is displayed , left to right, up to down
	L2R_D2U
	R2L_U2D
	R2L_D2U

	U2D_L2R
	U2D_R2L
	D2U_L2R
	D2U_R2L
)

var (
	// Init for 7735R, part 1 (red or green tab)
	initCmd = []commando{
		// {Command: SWRESET, Delayed: true, //  1: Software reset, 0 args, w/delay
		// 	Args:  nil,
		// 	Delay: 150}, //     150 ms delay

		// {Command: SLPOUT, Delayed: true, //  2: Out of sleep mode, 0 args, w/delay
		// 	Args:  nil,
		// 	Delay: 500}, //     500 ms delay

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

		// {Command: MADCTL, Delayed: false, // 14: Memory access control (directions), 1 arg:
		// 	Args:  []byte{0xC8}, //     row addr/col addr, bottom to top refresh
		// 	Delay: 0},

		{Command: GAMMA1, Delayed: false,
			Args: []byte{
				0x0f,
				0x1a,
				0x0f,
				0x18,
				0x2f,
				0x28,
				0x20,
				0x22,
				0x1f,
				0x1b,
				0x23,
				0x37,
				0x00,
				0x07,
				0x02,
				0x10,
			},
			Delay: 0},

		{Command: GAMMA2, Delayed: false,
			Args: []byte{
				0x0f,
				0x1b,
				0x0f,
				0x17,
				0x33,
				0x2c,
				0x29,
				0x2e,
				0x30,
				0x30,
				0x39,
				0x3f,
				0x00,
				0x07,
				0x03,
				0x10,
			},
			Delay: 0},

		{Command: COLMOD, Delayed: false, // set color mode, 1 arg, no delay:
			Args:  []byte{0x05}, //     16-bit color
			Delay: 0},
	}
)

// ST7735S represents the S style ST7735 display
type ST7735S struct {
	// ST7735S is-a ST7735
	ST7735

	scanningMethod int
}

// NewST7735S creates a variant of ST7735 128x160 (Green tab) TFT
func NewST7735S(dataCommand, reset gpio.Pin, tab devices.TabColor, dimensions devices.Dimensions) *ST7735S {
	st := new(ST7735S)

	st.dc = dataCommand
	st.reset = reset
	st.tab = tab
	st.dimensions = dimensions

	st.scanningMethod = L2R_U2D
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
func (st *ST7735S) Initialize(vender, product, clockFreq int, chipSelect gpio.Pin, orientation devices.RotationMode, colorOrder devices.ColorOrder) error {
	// Initialize the ST7735 device
	err := st.initialize(vender, product, clockFreq, chipSelect, colorOrder)
	if err != nil {
		return err
	}

	// I am not using the SD card on the device so I can leave CS low.
	// Of course if you are using a logic analyzer with an SPI decoder activated
	// the decoder won't display anything because CS is part of the SPI
	// protocol and must assert on each piece of data trafficking.
	st.SetConstantCSAssert(true)

	log.Println("ST7735S common init")
	err = st.commonInit(initCmd)
	if err != nil {
		return err
	}

	switch st.dimensions {
	case devices.D128x160:
		st.Width = 160
		st.Height = 128
		break
	case devices.D160x128:
		st.Width = 128
		st.Height = 160
		break
	}

	time.Sleep(time.Millisecond * 200)

	err = st.WriteCommand(SLPOUT)
	if err != nil {
		log.Printf("ST7735S failed to write SLPOUT: %v\n", err)
		return err
	}

	time.Sleep(time.Millisecond * 120)

	// st.DisplayOn(true)

	pixels := int(st.Width) * int(st.Height)
	// log.Printf("ST7735S offset screen buffer size: (%d) bytes\n", st.screenBufferSize)

	// A buffer of bytes
	st.pushBuffer = make([]byte, pixels*2)

	if orientation == devices.OrientationDefault {
		// log.Println("ST7735S setting orientation to default")
		st.SetRotation(devices.Orientation2)
	} else {
		// log.Println("ST7735S setting orientation")
		st.SetRotation(orientation)
	}

	st.EnableBacklightControl(ftdi.D7)

	return nil
}

// Not used. Use SetRotation instead.
// func (st *ST7735S) setGramScanWay() {

// 	// Gets the scan direction of GRAM
// 	MemoryAccessReg_Data := byte(0) //0x36

// 	switch st.scanningMethod {
// 	case L2R_U2D:
// 		MemoryAccessReg_Data = 0X00 | 0x00 //x Scan direction | y Scan direction
// 		break
// 	case L2R_D2U:
// 		MemoryAccessReg_Data = 0x00 | 0x80 //0xC8 | 0X10
// 		break
// 	case R2L_U2D: //	0X4
// 		MemoryAccessReg_Data = 0x40 | 0x00
// 		break
// 	case R2L_D2U: //	0XC
// 		MemoryAccessReg_Data = 0x40 | 0x80
// 		break
// 	case U2D_L2R: //0X2
// 		MemoryAccessReg_Data = 0X00 | 0X00 | 0x20
// 		break
// 	case U2D_R2L: //0X6
// 		MemoryAccessReg_Data = 0x00 | 0X40 | 0x20
// 		break
// 	case D2U_L2R: //0XA
// 		MemoryAccessReg_Data = 0x80 | 0x00 | 0x20
// 		break
// 	case D2U_R2L: //0XE
// 		MemoryAccessReg_Data = 0x40 | 0x80 | 0x20
// 		break
// 	}

// 	st.colstart = 2
// 	st.rowstart = 1

// 	// set (MemoryAccessReg_Data & 0x10) != 1
// 	if (MemoryAccessReg_Data & 0x10) != 1 {
// 	} else {
// 		st.rowstart += 2
// 	}

// 	log.Printf("ST7735S X,Y adjustment: %d,%d\n", st.colstart, st.rowstart)

// 	// Set the read / write scan direction of the frame memory
// 	err := st.WriteCommand(MADCTL) //MX, MY, RGB mode
// 	if err != nil {
// 		log.Printf("ST7735S failed to write scan direction: %v\n", err)
// 		return
// 	}

// 	// Note: some tft displays have the SRGB pin physically tied to High or Low
// 	// which means setting the mode here is meaningless. My tft has it tide to High
// 	// which forces the color order to BGR regardless of what the code below
// 	// does.
// 	RGBMode := byte(0x00)

// 	// f7 - 11110111
// 	//  & - 00001000  <-- BGR
// 	MemoryAccessReg_Data = RGBMode // & 0xf7

// 	log.Printf("ST7735S MADCTL: %08b\n", MemoryAccessReg_Data)

// 	st.WriteData(MemoryAccessReg_Data) //RGB color filter panel
// }

// DisplayOn turns the display on or off
func (st *ST7735S) DisplayOn(on bool) error {
	mode := byte(DISPON)
	if !on {
		mode = DISPOFF
	}
	err := st.WriteCommand(mode)
	if err != nil {
		log.Printf("ST7735S failed to write DISPON: %v\n", err)
		return err
	}

	return nil
}

// LightOn turns the backlight on or off
func (st *ST7735S) LightOn(on bool) error {
	st.BacklightOn(on)
	return nil
}

// Close closes the device.
func (st *ST7735S) Close() error {
	log.Println("ST7735S closing...")
	err := st.close()
	if err != nil {
		log.Println("Failed to close ST7735S.")
		log.Fatal(err)
	}

	log.Println("ST7735S closed.")

	return nil
}
