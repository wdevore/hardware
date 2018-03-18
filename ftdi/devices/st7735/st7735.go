package st7735

import (
	"errors"
	"log"
	"time"

	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/gpio"
	"github.com/wdevore/hardware/spi"
)

const (
	NOP     = 0x00
	SWRESET = 0x01
	RDDID   = 0x04
	RDDST   = 0x09
	SLPIN   = 0x10
	SLPOUT  = 0x11
	PTLON   = 0x12
	NORON   = 0x13
	INVOFF  = 0x20
	INVON   = 0x21
	DISPOFF = 0x28
	DISPON  = 0x29
	CASET   = 0x2A
	RASET   = 0x2B
	RAMWR   = 0x2C
	RAMRD   = 0x2E
	RAMCWR  = 0x3C // Write memory continue
	PTLAR   = 0x30
	COLMOD  = 0x3A
	MADCTL  = 0x36
	FRMCTR1 = 0xB1
	FRMCTR2 = 0xB2
	FRMCTR3 = 0xB3
	INVCTR  = 0xB4
	DISSET5 = 0xB6
	PWCTR1  = 0xC0
	PWCTR2  = 0xC1
	PWCTR3  = 0xC2
	PWCTR4  = 0xC3
	PWCTR5  = 0xC4
	VMCTR1  = 0xC5
	RDID1   = 0xDA
	RDID2   = 0xDB
	RDID3   = 0xDC
	RDID4   = 0xDD
	PWCTR6  = 0xFC
	GMCTRP1 = 0xE0
	GMCTRN1 = 0xE1
)

const (
	MadctlMY  = 0x80
	MadctlMX  = 0x40
	MadctlMV  = 0x20
	MadctlML  = 0x10
	MadctlRGB = 0x00
	MadctlBGR = 0x08
	MadctlMH  = 0x04
)

var (
	colorPush     = []byte{0x00, 0x00}
	writeBuf      = []byte{0x00}
	addWindowBuf  = []byte{0x00, 0x00, 0x00, 0x00}
	bytesPerPixel = 2
)

// ST7735 represents the TFT/LCD controller chip.
type ST7735 struct {
	// ST7735 uses the USB FTDI232 SPI object
	spi *spi.FtdiSPI

	ystart   byte
	xstart   byte
	colstart byte
	rowstart byte

	dc    gpio.Pin // Data/Command pin
	reset gpio.Pin

	tab        devices.TabColor
	dimensions devices.Dimensions

	Width  byte
	Height byte

	// In general each word is an RGB (565) of 2 bytes each.
	// High byte followed by Low byte
	// HLHLHLHLHLHL...
	// pushBuffer is the "off screen" display buffer that is blitted in one Data
	// write call. This saves a huge amount of time.
	// With a clock of 30MHz (the max for the FTDI232H device) the buffer
	// can be blitted in about 10ms as compared to non-buffered in about 1300ms!
	// 10ms allows for a framerate between 30FPS(~33ms/frame) to 60FPS(~16ms/frame).
	// Although 60FPS only leaves about 6ms for your code which is pretty tight.
	pushBuffer []byte
}

// Screen initialization commands and arguments are organized in these tables
// stored in progMem.  The table may look bulky, but that's mostly the
// formatting -- storage-wise this is hundreds of bytes more compact
// than the equivalent code.  Companion function follows.

// Commando is a simple command layout for writing to the device. It is only used
// during the initialization sequence.
type commando struct {
	Command byte
	Delayed bool
	Args    []byte
	Delay   int // 255 = 500ms delay max
}

// Initialize configures FTDI and SPI, and initializes ST7735
// Vendor/Product example would be: 0x0403, 0x06014
// A clock frequency of 0 means default to max = 30MHz
func (st *ST7735) initialize(vender, product, clockFreq int, chipSelect gpio.Pin) error {

	// Create a SPI interface from the FT232H
	st.spi = spi.NewSPI(vender, product, false)
	st.spi.DebugInit()

	if st.spi == nil {
		return errors.New("ST7735 failed to create SPI object")
	}

	if clockFreq == 0 {
		clockFreq = 30000000
	}
	log.Printf("Configuring ST7735 for a clock of (%d)MHz\n", clockFreq/1000000)

	err := st.configure(chipSelect, clockFreq)
	if err != nil {
		return err
	}

	log.Println("ST7735 configured.")

	return nil
}

// Configure sets up the SPI component and initializes the ST7735
func (st *ST7735) configure(chipSelect gpio.Pin, clockFreq int) error {
	log.Println("Configuring SPI")
	err := st.spi.Configure(chipSelect, clockFreq, spi.Mode0, spi.MSBFirst)

	if err != nil {
		log.Println("ST7735 Configure FAILED.")
		return err
	}

	return nil
}

func (st *ST7735) close() error {
	return st.spi.Close()
}

// commonInit setups common pin configurations
func (st *ST7735) commonInit(cmdList []commando) error {
	st.ystart = 0
	st.xstart = 0

	sp := st.spi

	// The ST7735 communicates with TFT device (aka ST7735R device) through the FTDI235H device
	// via the SPI protocol. However, the SPI protocol only accounts for, at most, 4 pins, anything
	// else needs to added manually--and controlled manually.

	// Setup extra pins for D/C and Reset. For this we need to interface with the FTDI chip
	fi := sp.GetFTDI()

	pins := []gpio.PinConfiguration{
		{Pin: st.dc, Direction: gpio.Output, Value: gpio.Z},
	}
	fi.ConfigPins(pins, true)

	// toggle RST low to reset and CS low so it'll listen to us
	sp.AssertChipSelect()

	if st.reset != gpio.NoPin {
		fi.ConfigPin(st.reset, gpio.Output)

		fi.OutputHigh(st.reset)
		time.Sleep(time.Millisecond * 500)

		fi.OutputLow(st.reset)
		time.Sleep(time.Millisecond * 500)

		fi.OutputHigh(st.reset)
		time.Sleep(time.Millisecond * 500)
	}

	if cmdList != nil {
		st.issueCommands(cmdList)
	}

	return nil
}

// ----------------------------------------------------
// Commands
// ----------------------------------------------------
func (st *ST7735) issueCommands(cmdList []commando) {
	for _, com := range cmdList {
		err := st.WriteCommand(com.Command)
		if err != nil {
			log.Printf("ST7735 issueCommands failed to write command: %v\n", err)
			return
		}

		for _, arg := range com.Args {
			st.WriteData(arg)
		}

		if com.Delayed {
			d := time.Millisecond * time.Duration(com.Delay)
			// log.Printf("ST7735 issueCommands delaying for (%d)ms\n", time.Duration(com.Delay))
			time.Sleep(d)
		}
	}
}

// ----------------------------------------------------
// Rotation
// ----------------------------------------------------

// SetRotation re-orients the display at 90 degree rotations.
// Typically this method is called last during the initialization sequence.
func (st *ST7735) SetRotation(orieo devices.RotationMode) {
	st.WriteCommand(MADCTL)

	switch orieo {
	case devices.Orientation0:
		st.WriteData(MadctlMX | MadctlMY | MadctlBGR)
		st.rowstart = 32
		st.xstart = st.colstart
		st.ystart = st.rowstart
		break
	case devices.Orientation1:
		st.WriteData(MadctlMY | MadctlMV | MadctlBGR)
		st.rowstart = 32
		st.ystart = st.colstart
		st.xstart = st.rowstart
		break
	case devices.Orientation2:
		st.WriteData(MadctlBGR)
		st.rowstart = 1
		st.xstart = st.colstart
		st.ystart = st.rowstart
		break
	case devices.Orientation3:
		st.WriteData(MadctlMX | MadctlMV | MadctlBGR)
		st.rowstart = 0
		st.ystart = st.colstart
		st.xstart = st.rowstart
		break
	}
}

// InvertDisplay inverts the display colors
func (st *ST7735) InvertDisplay(inv bool) {
	if inv {
		st.WriteCommand(INVON)
	} else {
		st.WriteCommand(INVOFF)

	}
}

// ----------------------------------------------------
// Writing
// ----------------------------------------------------

// SetConstantCSAssert sets the constant assert flag.
// If you have an the Adafruit ST7735 then you most likely
// have an micro sd card present. If you aren't using it then
// you can leave CS low for the entire time and thus save
// on bandwidth.
func (st *ST7735) SetConstantCSAssert(constant bool) {
	st.spi.ConstantCSAssert = constant
}

// WriteCommand writes a command via SPI protocol
func (st *ST7735) WriteCommand(command byte) error {
	// log.Printf("ST7735: WriteCommand (%02x)\n", command)
	sp := st.spi
	fi := sp.GetFTDI()

	// log.Println("ST7735: WriteCommand: toggling dc")
	fi.OutputLow(st.dc) // Low = command

	// log.Println("ST7735: WriteCommand: writing byte command")
	writeBuf[0] = command
	err := st.spi.Write(writeBuf)
	if err != nil {
		log.Println("Failed to write command.")
		return err
	}

	// log.Println("ST7735: WriteCommand: toggling cs")

	return nil
}

// WriteData writes data to the device via SPI
func (st *ST7735) WriteData(data byte) {
	// log.Printf("ST7735: WriteData: (%02x)\n", data)
	sp := st.spi
	fi := sp.GetFTDI()
	fi.OutputHigh(st.dc) // High = data

	writeBuf[0] = data
	sp.Write(writeBuf)
}

// WriteDataChunk is a slightly more efficient version of WriteData
func (st *ST7735) WriteDataChunk(data []byte) {
	// log.Printf("ST7735: WriteData: (%02x)\n", data)
	sp := st.spi
	fi := sp.GetFTDI()
	fi.OutputHigh(st.dc) // High = data

	sp.Write(data)
}

// ----------------------------------------------------
// Graphics Unbuffered
// ----------------------------------------------------

// SetAddrWindow set row and column address of where a pixel will be written.
// (aka setDrawPosition)
func (st *ST7735) SetAddrWindow(x0, y0, x1, y1 byte) {

	st.WriteCommand(CASET) // Column addr set
	addWindowBuf[1] = x0 + st.xstart
	addWindowBuf[3] = x1 + st.xstart
	st.WriteDataChunk(addWindowBuf)

	// st.WriteDataChunk([]byte{0x00, x0 + st.xstart, 0x00, x1 + st.xstart})

	// st.WriteData(0x00)
	// st.WriteData(x0 + st.xstart) // XSTART
	// st.WriteData(0x00)
	// st.WriteData(x1 + st.xstart) // XEND

	st.WriteCommand(RASET) // Row addr set
	addWindowBuf[1] = y0 + st.ystart
	addWindowBuf[3] = y1 + st.ystart
	st.WriteDataChunk(addWindowBuf)

	// st.WriteDataChunk([]byte{0x00, y0 + st.ystart, 0x00, y1 + st.ystart})

	// st.WriteData(0x00)
	// st.WriteData(y0 + st.ystart) // YSTART
	// st.WriteData(0x00)
	// st.WriteData(y1 + st.ystart) // YEND

	st.WriteCommand(RAMWR) // write to RAM
}

// PushColor writes a 16bit color value based on the current draw position.
// (aka writePixel)
func (st *ST7735) PushColor(color uint16) {
	sp := st.spi
	fi := sp.GetFTDI()

	fi.OutputHigh(st.dc)

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)
	sp.Write(colorPush)

	// Or (slow)
	// sp.Write([]byte{byte((color >> 8) & 0xff), byte(color)})

	// Or (slowest)
	// sp.WriteByte(byte(color>>8) & 0xff)
	// sp.WriteByte(byte(color))
}

// DrawPixel draws to device only.
func (st *ST7735) DrawPixel(x, y byte, color uint16) {

	if (x < 0) || (x > st.Width) || (y < 0) || (y > st.Height) {
		return
	}

	sp := st.spi
	fi := sp.GetFTDI()

	// First set "cursor"/"draw position"
	// st.SetAddrWindow(x, y, x+1, y+1)
	st.SetAddrWindow(x, y, 1, 1)

	fi.OutputHigh(st.dc)

	// Now draw.
	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)
	sp.Write(colorPush)
}

// DrawFastVLine draws a vertical line only
func (st *ST7735) DrawFastVLine(x, y, h byte, color uint16) {

	// Rudimentary clipping
	if (x > st.Width) || (y > st.Height) {
		log.Println("DrawFastVLine: Rejected x,y")
		return
	}

	if (y + h - 1) > st.Height {
		h = st.Height - y
	}

	st.SetAddrWindow(x, y, x, y+h-1)

	sp := st.spi
	fi := sp.GetFTDI()

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)

	fi.OutputHigh(st.dc)

	for h > 0 {
		sp.Write(colorPush)
		h--
	}
}

// DrawFastHLine draws a horizontal line only
func (st *ST7735) DrawFastHLine(x, y, w byte, color uint16) {

	// Rudimentary clipping
	if (x > st.Width) || (y > st.Height) {
		log.Println("DrawFastHLine: Rejected x,y")
		return
	}

	if (x + w - 1) > st.Width {
		w = st.Width - x
	}

	st.SetAddrWindow(x, y, x+w-1, y)

	sp := st.spi
	fi := sp.GetFTDI()

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)

	fi.OutputHigh(st.dc)

	for w > 0 {
		sp.Write(colorPush)
		w--
	}
}

// FillScreen fills the entire display area with "color"
func (st *ST7735) FillScreen(color uint16) {
	st.FillRectangle(0, 0, st.Width, st.Height, color)
}

// deprecated (old) FillRectangle2 fills a rectangle
// func (st *ST7735) FillRectangle2(x, y, w, h byte, color uint16) {
// 	// log.Printf("%d x %d\n", st.Width, st.Height)
// 	// rudimentary clipping (drawChar w/big text requires this)
// 	if (x > st.Width) || (y > st.Height) {
// 		// log.Println("FillRectangle Rejected rectangle x,y")
// 		return
// 	}
// 	if (x + w - 1) > st.Width {
// 		// log.Println("ST7735 FillRectangle: adjusting w")
// 		w = st.Width - x
// 	}
// 	if (y + h - 1) > st.Height {
// 		// log.Println("ST7735 FillRectangle: adjusting h")
// 		h = st.Height - y
// 	}

// 	st.SetAddrWindow(x, y, x+w-1, y+h-1)

// 	sp := st.spi
// 	fi := sp.GetFTDI()

// 	fi.OutputHigh(st.dc)

// 	pixels := int(w) * int(h) * 2
// 	// log.Printf("ST7735 FillRectangle pixels: (%d)\n", pixels)
// 	// temp := make([]byte, pixels)
// 	j := 0
// 	for y := h; y > 0; y-- {
// 		for x := w; x > 0; x-- {
// 			st.pushBuffer[j] = byte((color >> 8) & 0xff)
// 			st.pushBuffer[j+1] = byte(color & 0xff)
// 			// temp[j] = byte((color >> 8) & 0xff)
// 			// temp[j+1] = byte(color & 0xff)
// 			j += 2
// 		}
// 	}
// 	sp.WriteLen(st.pushBuffer, pixels)

// 	// for y := h; y > 0; y-- {
// 	// 	for x := w; x > 0; x-- {
// 	// 		colorPush[0] = byte((color >> 8) & 0xff)
// 	// 		colorPush[1] = byte(color & 0xff)
// 	// 		sp.Write(colorPush)
// 	// 	}
// 	// }
// }

// FillRectangle is a very slow fill.
func (st *ST7735) FillRectangle(x, y, w, h byte, color uint16) {
	// log.Printf("%d x %d\n", st.Width, st.Height)
	// rudimentary clipping (drawChar w/big text requires this)
	if (x > st.Width) || (y > st.Height) {
		// log.Println("FillRectangle Rejected rectangle x,y")
		return
	}
	if (x + w - 1) > st.Width {
		// log.Println("ST7735 FillRectangle: adjusting w")
		w = st.Width - x
	}
	if (y + h - 1) > st.Height {
		// log.Println("ST7735 FillRectangle: adjusting h")
		h = st.Height - y
	}

	st.SetAddrWindow(x, y, x+w-1, y+h-1)

	sp := st.spi
	fi := sp.GetFTDI()

	fi.OutputHigh(st.dc)

	for y := h; y > 0; y-- {
		for x := w; x > 0; x-- {
			colorPush[0] = byte((color >> 8) & 0xff)
			colorPush[1] = byte(color & 0xff)
			sp.Write(colorPush)
		}
	}
}

// ----------------------------------------------------
// Graphics Buffered (Preferred)
// ----------------------------------------------------

// Blit writes the contents of the displayBuffer directly to the display as fast as it can!
// At time of writing could be improved for speed by going directly to SPI interface
// itself but for now fast enough doing it this way for most needs
func (st *ST7735) Blit() {
	// writes the contents of the buffer directly to the display as fast as it can!
	// At time of writing could be improved for speed by going directly to SPI interface
	// itself but for now fast enough doing it this way for most needs

	// Because we are only ever writing the screen buffer we do not need to set
	// a screen start location as by default after a reset it will be 0,0 and after
	// a full buffer write it will reset back to this. If you decide to extend this
	// code by doing partial buffer writes then you will need to set the start location
	// to 0,0 using the following line (uncomment if needed)
	//  st.SetAddrWindow(0,0,1,1);
	// st.spi.TriggerPulse()

	sp := st.spi
	fi := sp.GetFTDI()

	st.SetAddrWindow(0, 0, st.Width-1, st.Height-1)

	fi.OutputHigh(st.dc)

	sp.Write(st.pushBuffer)
}

// DrawPixelToBuf draws to screen buffer only. You will need to eventually
// call Blit() to see anything.
func (st *ST7735) DrawPixelToBuf(x, y byte, color uint16) {

	if (x < 0) || (x > st.Width) || (y < 0) || (y > st.Height) {
		return
	}

	// write to screen buffer memory instead, this is quick and dirty, presumes always using
	// RGB565 (16bit per pixel colour)
	// Calculate memory location based on screen width and height
	bufOff := int(y)*int(st.Height) + int(x)
	st.pushBuffer[bufOff*2] = byte(color >> 8 & 0xff) // High
	st.pushBuffer[bufOff*2+1] = byte(color & 0xff)    // Low
}

// DrawVLineToBuf draws a vertical line
func (st *ST7735) DrawVLineToBuf(x, y, h byte, color uint16) {

	// Rudimentary clipping
	if (x > st.Width) || (y > st.Height) {
		return
	}

	msb := byte((color >> 8) & 0xff) // High
	lsb := byte(color & 0xff)        // Low

	for iy := y; iy < y+h; iy++ {
		bufOff := int(iy)*int(st.Height) + int(x)

		st.pushBuffer[bufOff*2] = msb
		st.pushBuffer[bufOff*2+1] = lsb
	}
}

// DrawHLineToBuf draws a horizontal line
func (st *ST7735) DrawHLineToBuf(x, y, w byte, color uint16) {

	// Rudimentary clipping
	if (x > st.Width) || (y > st.Height) {
		return
	}

	msb := byte((color >> 8) & 0xff) // High
	lsb := byte(color & 0xff)        // Low

	for ix := x; ix < x+w; ix++ {
		bufOff := int(y)*int(st.Height) + int(ix)

		st.pushBuffer[bufOff*2] = msb
		st.pushBuffer[bufOff*2+1] = lsb
	}
}

// FillScreenToBuf fills the entire display area with "color"
func (st *ST7735) FillScreenToBuf(color uint16) {
	st.FillRectangleToBuf(0, 0, st.Width, st.Height, color)
}

// FillRectangleToBuf fills a rectangle in the screen buffer
func (st *ST7735) FillRectangleToBuf(x, y, w, h byte, color uint16) {
	// log.Printf("%d x %d\n", st.Width, st.Height)
	// rudimentary clipping (drawChar w/big text requires this)
	if (x > st.Width) || (y > st.Height) {
		// log.Println("FillRectangle Rejected rectangle x,y")
		return
	}

	sw := int(x + w)
	sh := int(y + h)
	if (sw > int(st.Width)) || (sh > int(st.Height)) {
		// log.Println("FillRectangle Rejected rectangle x,y")
		return
	}

	msb := byte((color >> 8) & 0xff)
	lsb := byte(color & 0xff)

	j := 0
	ih := int(st.Height)
	pl := len(st.pushBuffer)

	// log.Printf("%d,%d\n", sw, sh)
fillLoop:
	for sy := int(y); sy < sh; sy++ {
		for sx := int(x); sx < sw; sx++ {
			j = int(sy)*ih + int(sx)
			if j > pl {
				break fillLoop
			}
			st.pushBuffer[j*2] = msb
			st.pushBuffer[j*2+1] = lsb
		}
	}
}
