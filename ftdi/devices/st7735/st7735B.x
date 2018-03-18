package st7735

/*


NOTE: THIS ISN'T COMPLETE. I DON'T HAVE A B TYPE DISPLAY.



*/

import (
	"github.com/wdevore/hardware/gpio"
)

var (
	// Bcmd Initialization commands for 7735B screens
	bcmd = []commando{
		{SWRESET, delayed, //  1: Software reset, no args, w/delay
			nil,
			50}, //     50 ms delay

		{SLPOUT, delayed, //  2: Out of sleep mode, no args, w/delay
			nil,
			500}, //     500 ms delay

		{COLMOD, delayed, //  3: Set color mode, 1 arg + delay:
			[]byte{0x05}, //     16-bit color
			10},          //     10 ms delay

		{FRMCTR1, delayed, //  4: Frame rate control, 3 args + delay:
			[]byte{
				0x00,  //     fastest refresh
				0x06,  //     6 lines front porch
				0x03}, //     3 lines back porch
			10}, //     10 ms delay

		{MADCTL, noDelay, //  5: Memory access ctrl (directions), 1 arg:
			[]byte{0x08}, //     Row addr/col addr, bottom to top refresh
			0},

		{DISSET5, noDelay, //  6: Display settings #5, 2 args, no delay:
			[]byte{0x15, //     1 clk cycle nonoverlap, 2 cycle gate
				//     rise, 3 cycle osc equalize
				0x02},
			0}, //     Fix on VTL

		{INVCTR, noDelay, //  7: Display inversion control, 1 arg:
			[]byte{0x0},
			0}, //     Line inversion

		{PWCTR1, delayed, //  8: Power control, 2 args + delay:
			[]byte{0x02, //     GVDD = 4.7V
				0x70}, //     1.0uA
			10}, //     10 ms delay

		{PWCTR2, noDelay, //  9: Power control, 1 arg, no delay:
			[]byte{0x05}, //     VGH = 14.7V, VGL = -7.35V
			0},

		{PWCTR3, noDelay, // 10: Power control, 2 args, no delay:
			[]byte{0x01, //     Opamp current small
				0x02}, //     Boost frequency
			0},

		{VMCTR1, delayed, // 11: Power control, 2 args + delay:
			[]byte{0x3C, //     VCOMH = 4V
				0x38}, //     VCOML = -1.1V
			10}, //     10 ms delay

		{PWCTR6, noDelay, // 12: Power control, 2 args, no delay:
			[]byte{0x11, 0x15},
			0},

		{GMCTRP1, noDelay, // 13: Magical unicorn dust, 16 args, no delay:
			[]byte{
				0x09, 0x16, 0x09, 0x20, //     (seriously though, not sure what
				0x21, 0x1B, 0x13, 0x19, //      these config values represent)
				0x17, 0x15, 0x1E, 0x2B,
				0x04, 0x05, 0x02, 0x0E},
			0},

		{GMCTRN1, delayed, // 14: Sparkles and rainbows, 16 args + delay:
			[]byte{
				0x0B, 0x14, 0x08, 0x1E, //     (ditto)
				0x22, 0x1D, 0x18, 0x1E,
				0x1B, 0x1A, 0x24, 0x2B,
				0x06, 0x06, 0x02, 0x0F},
			10}, //     10 ms delay

		{CASET, noDelay, // 15: Column addr set, 4 args, no delay:
			[]byte{
				0x00, 0x02, //     XSTART = 2
				0x00, 0x81}, //     XEND = 129
			0},

		{RASET, noDelay, // 16: Row addr set, 4 args, no delay:
			[]byte{
				0x00, 0x02, //     XSTART = 1
				0x00, 0x81}, //     XEND = 160
			0},

		{NORON, delayed, // 17: Normal display on, no args, w/delay
			nil,
			10}, //     10 ms delay

		{DISPON, delayed, // 18: Main screen turn on, no args, w/delay
			nil,
			500}, //     500 ms delay
	}
)

// ST7735B represents the B style ST7735 display
type ST7735B struct {
	// ST7735B is-a ST7735
	ST7735
}

// NewST7735B creates a variant of ST7735
func NewST7735B(dataCommand, reset gpio.Pin, tab TabColor) *ST7735B {
	st := new(ST7735B)

	st.dc = dataCommand
	st.reset = reset
	st.tab = tab

	return st
}

// Initialize configures FTDI and SPI, and initializes ST7735
func (st *ST7735B) Initialize() error {
	err := st.configure()
	if err != nil {
		return err
	}

	st.commonInit(bcmd)
	// commandList(Rcmd2red)
	// commandList(Rcmd3)
	// setRotation(1)

	return nil
}

// Configure sets up the SPI component and initializes the ST7735
func (st *ST7735B) configure() error {

	// // Use a clock speed of 30mhz (Max for FT232H), SPI mode 0, and most significant bit first.
	// err := st.spi.Configure(st.chipSelect, 30000000, ftdi.Mode0, ftdi.MSBFirst)

	// if err != nil {
	// 	return err
	// }

	return nil
}

func (st *ST7735B) commonInit(cmdList []commando) error {
	st.ystart = 0
	st.xstart = 0
	st.colstart = 0
	st.rowstart = 0

	// sp := st.spi

	// pins := []gpio.PinConfiguration{
	// 	{Pin: st.dc, Direction: gpio.Output, Value: gpio.Z},
	// 	{Pin: st.chipSelect, Direction: gpio.Output, Value: gpio.Z},
	// }
	// sp.ConfigPins(pins, true)

	// // toggle RST low to reset and CS low so it'll listen to us
	// sp.OutputLow(st.chipSelect)

	// if st.reset != ftdi.NoPin {
	// 	sp.ConfigPin(st.reset, gpio.Output)
	// 	sp.Output(st.reset, gpio.High)
	// 	time.Sleep(time.Millisecond * 500)
	// 	sp.Output(st.reset, gpio.Low)
	// 	time.Sleep(time.Millisecond * 500)
	// 	sp.Output(st.reset, gpio.High)
	// 	time.Sleep(time.Millisecond * 500)
	// }

	// if cmdList != nil {
	// 	st.issueCommands(cmdList)
	// }

	return nil
}
