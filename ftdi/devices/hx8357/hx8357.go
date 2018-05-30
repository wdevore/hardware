package hx8357

import (
	"errors"
	"log"
	"time"

	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/gpio"
	"github.com/wdevore/hardware/spi"
)

const (
	HX8357_D      = 0xD
	HX8357_B      = 0xB
	TFTWIDTH      = 320
	TFTHEIGHT     = 480
	NOP           = 0x00
	SWRESET       = 0x01
	RDDID         = 0x04
	RDDST         = 0x09
	RDPOWMODE     = 0x0A
	RDMADCTL      = 0x0B
	RDCOLMOD      = 0x0C
	RDDIM         = 0x0D
	RDDSDR        = 0x0F
	SLPIN         = 0x10
	SLPOUT        = 0x11
	HX8357B_PTLON = 0x12
	HX8357B_NORON = 0x13
	INVOFF        = 0x20
	INVON         = 0x21
	DISPOFF       = 0x28
	DISPON        = 0x29
	CASET         = 0x2A
	PASET         = 0x2B
	RAMWR         = 0x2C
	RAMRD         = 0x2E

	HX8357B_PTLAR             = 0x30
	TEON                      = 0x35
	TEARLINE                  = 0x44
	MADCTL                    = 0x36
	COLMOD                    = 0x3A
	SETOSC                    = 0xB0
	SETPWR1                   = 0xB1
	HX8357B_SETDISPLAY        = 0xB2
	SETRGB                    = 0xB3
	HX8357D_SETCOM            = 0xB6
	HX8357B_SETDISPMODE       = 0xB4
	HX8357D_SETCYC            = 0xB4
	HX8357B_SETOTP            = 0xB7
	HX8357D_SETC              = 0xB9
	HX8357B_SET_PANEL_DRIVING = 0xC0
	HX8357D_SETSTBA           = 0xC0
	HX8357B_SETDGC            = 0xC1
	HX8357B_SETID             = 0xC3
	HX8357B_SETDDB            = 0xC4
	HX8357B_SETDISPLAYFRAME   = 0xC5
	HX8357B_GAMMASET          = 0xC8
	HX8357B_SETCABC           = 0xC9
	SETPANEL                  = 0xCC

	HX8357B_SETPOWER        = 0xD0
	HX8357B_SETVCOM         = 0xD1
	HX8357B_SETPWRNORMAL    = 0xD2
	HX8357B_RDID1           = 0xDA
	HX8357B_RDID2           = 0xDB
	HX8357B_RDID3           = 0xDC
	HX8357B_RDID4           = 0xDD
	HX8357D_SETGAMMA        = 0xE0
	HX8357B_SETGAMMA        = 0xC8
	HX8357B_SETPANELRELATED = 0xE9
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

// HX8357 represents the TFT/LCD controller chip.
type HX8357 struct {
	// Uses the USB FTDI232 SPI object
	spi *spi.FtdiSPI

	dc    gpio.Pin // Data/Command pin
	reset gpio.Pin

	tab        devices.TabColor
	dimensions devices.Dimensions

	Width  uint16
	Height uint16

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

	lineBlockSize int
	lines         int
	chunkSize     int

	// fmt.Printf("lineBlock %d, chunksize: %d\n", lines, chunkSize)
	chunkBuf []byte
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

// Initialize configures FTDI and SPI, and initializes HX8357
// Vendor/Product example would be: 0x0403, 0x06014 for the FTDI chip
// A clock frequency of 0 means default to max = 30MHz
func (hx *HX8357) initialize(vender, product, clockFreq int, chipSelect gpio.Pin) error {

	// Create a SPI interface from the FT232H
	hx.spi = spi.NewSPI(vender, product, false)
	//hx.spi.DebugInit()

	if hx.spi == nil {
		return errors.New("HX8357: Failed to create SPI object")
	}

	if clockFreq == 0 {
		clockFreq = devices.Max30MHz
	}
	log.Printf("HX8357: Configuring for a clock of (%d)MHz\n", clockFreq/1000000)

	err := hx.configure(chipSelect, clockFreq)
	if err != nil {
		return err
	}

	log.Println("HX8357: configured.")

	return nil
}

// Configure sets up the SPI component and initializes the ST7735
func (hx *HX8357) configure(chipSelect gpio.Pin, clockFreq int) error {
	log.Println("HX8357: Configuring SPI")
	err := hx.spi.Configure(chipSelect, clockFreq, spi.Mode0, spi.MSBFirst)

	if err != nil {
		log.Println("HX8357: Configure FAILED.")
		return err
	}

	return nil
}

func (hx *HX8357) close() error {
	return hx.spi.Close()
}

// commonInit setups common pin configurations
func (hx *HX8357) commonInit(cmdList []commando) error {
	sp := hx.spi

	// The HX8357 communicates with TFT device (aka HX8357D device) through the FTDI235H device
	// via the SPI protocol. However, the SPI protocol only accounts for, at most, 4 pins, anything
	// else needs to added manually--and controlled manually.

	// Setup extra pins for D/C and Reset. For this we need to interface with the FTDI chip
	fi := sp.GetFTDI()

	pins := []gpio.PinConfiguration{
		{Pin: hx.dc, Direction: gpio.Output, Value: gpio.Z},
	}
	fi.ConfigPins(pins, true)

	// toggle RST low to reset and CS low so it'll listen to us
	sp.AssertChipSelect()

	if hx.reset != gpio.NoPin {
		fi.ConfigPin(hx.reset, gpio.Output)

		fi.OutputHigh(hx.reset)
		time.Sleep(time.Millisecond * 100)

		fi.OutputLow(hx.reset)
		time.Sleep(time.Millisecond * 100)

		fi.OutputHigh(hx.reset)
		time.Sleep(time.Millisecond * 150)
	}

	if cmdList != nil {
		hx.issueCommands(cmdList)
	}

	return nil
}

// ----------------------------------------------------
// Commands
// ----------------------------------------------------
func (hx *HX8357) issueCommands(cmdList []commando) {
	for _, com := range cmdList {
		err := hx.WriteCommand(com.Command)
		if err != nil {
			log.Printf("HX8357: issueCommands failed to write command: %v\n", err)
			return
		}

		for _, arg := range com.Args {
			hx.WriteData(arg)
		}

		if com.Delayed {
			d := time.Millisecond * time.Duration(com.Delay)
			// log.Printf("HX8357 issueCommands delaying for (%d)ms\n", time.Duration(com.Delay))
			time.Sleep(d)
		}
	}
}

// ----------------------------------------------------
// Rotation
// ----------------------------------------------------

// SetRotation re-orients the display at 90 degree rotations.
// Typically this method is called last during the initialization sequence.
func (hx *HX8357) SetRotation(orieo devices.RotationMode) {
	hx.WriteCommand(MADCTL)

	switch orieo {
	case devices.Orientation0:
		hx.WriteData(MadctlMX | MadctlMY | MadctlRGB)
		break
	case devices.Orientation1:
		hx.WriteData(MadctlMY | MadctlMV | MadctlRGB)
		break
	case devices.Orientation2:
		hx.WriteData(MadctlRGB)
		break
	case devices.Orientation3:
		hx.WriteData(MadctlMX | MadctlMV | MadctlRGB)
		break
	}
}

// InvertDisplay inverts the display colors
func (hx *HX8357) InvertDisplay(inv bool) {
	if inv {
		hx.WriteCommand(INVON)
	} else {
		hx.WriteCommand(INVOFF)

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
func (hx *HX8357) SetConstantCSAssert(constant bool) {
	hx.spi.ConstantCSAssert = constant
}

// WriteCommand writes a command via SPI protocol
func (hx *HX8357) WriteCommand(command byte) error {
	sp := hx.spi
	fi := sp.GetFTDI()

	fi.OutputLow(hx.dc) // Low = command

	writeBuf[0] = command
	err := hx.spi.Write(writeBuf)
	if err != nil {
		log.Println("Failed to write command.")
		return err
	}

	return nil
}

// WriteData writes data to the device via SPI
func (hx *HX8357) WriteData(data byte) {
	sp := hx.spi
	fi := sp.GetFTDI()
	fi.OutputHigh(hx.dc) // High = data

	writeBuf[0] = data
	sp.Write(writeBuf)
}

// WriteDataChunk is a slightly more efficient version of WriteData
func (hx *HX8357) WriteDataChunk(data []byte) {
	sp := hx.spi
	fi := sp.GetFTDI()
	fi.OutputHigh(hx.dc) // High = data

	sp.Write(data)
}

// ----------------------------------------------------
// Graphics Unbuffered
// ----------------------------------------------------

// SetAddrWindow set row and column address of where pixels will be written.
// (aka setDrawPosition)
func (hx *HX8357) SetAddrWindow(x, y, w, h uint16) {
	xa := (uint32(x) << 16) | uint32(x+w-1)
	ya := (uint32(y) << 16) | uint32(y+h-1)

	hx.WriteCommand(CASET) // Column addr set
	// -- -- -- --
	addWindowBuf[0] = byte((xa >> 24) & 0xff)
	addWindowBuf[1] = byte((xa >> 16) & 0xff)
	addWindowBuf[2] = byte((xa >> 8) & 0xff)
	addWindowBuf[3] = byte(xa & 0xff)
	hx.WriteDataChunk(addWindowBuf)

	hx.WriteCommand(PASET) // Row addr set
	addWindowBuf[0] = byte((ya >> 24) & 0xff)
	addWindowBuf[1] = byte((ya >> 16) & 0xff)
	addWindowBuf[2] = byte((ya >> 8) & 0xff)
	addWindowBuf[3] = byte(ya & 0xff)
	hx.WriteDataChunk(addWindowBuf)

	hx.WriteCommand(RAMWR) // write to RAM
}

// PushColor writes a 16bit color value based on the current cursor/draw position.
// The "cursor" position is set by SetAddrWindow()
// (aka writePixel)
func (hx *HX8357) PushColor(color uint16) {
	sp := hx.spi
	fi := sp.GetFTDI()

	fi.OutputHigh(hx.dc)

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)
	sp.Write(colorPush)
}

// DrawPixel draws to device only.
func (hx *HX8357) DrawPixel(x, y uint16, color uint16) {

	if (x < 0) || (x > hx.Width) || (y < 0) || (y > hx.Height) {
		return
	}

	sp := hx.spi
	fi := sp.GetFTDI()

	// First set "cursor"/"draw position"
	hx.SetAddrWindow(x, y, 1, 1)

	fi.OutputHigh(hx.dc)

	// Now draw.
	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)
	sp.Write(colorPush)
}

// DrawFastVLine draws a vertical line only
func (hx *HX8357) DrawFastVLine(x, y, h uint16, color uint16) {

	// Rudimentary clipping
	if (x > hx.Width) || (y > hx.Height) {
		log.Println("DrawFastVLine: Rejected x,y")
		return
	}

	if (y + h) > hx.Height {
		h = hx.Height - y
	}

	hx.SetAddrWindow(x, y, x, y+h)

	sp := hx.spi
	fi := sp.GetFTDI()

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)

	fi.OutputHigh(hx.dc)

	for h > 0 {
		sp.Write(colorPush)
		h--
	}
}

// DrawFastHLine draws a horizontal line only
func (hx *HX8357) DrawFastHLine(x, y, w uint16, color uint16) {

	// Rudimentary clipping
	if (x > hx.Width) || (y > hx.Height) {
		log.Println("DrawFastHLine: Rejected x,y")
		return
	}

	if (x + w) > hx.Width {
		w = hx.Width - x
	}

	hx.SetAddrWindow(x, y, x+w, y)

	sp := hx.spi
	fi := sp.GetFTDI()

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)

	fi.OutputHigh(hx.dc)

	for w > 0 {
		sp.Write(colorPush)
		w--
	}
}

// FillScreen fills the entire display area with "color"
func (hx *HX8357) FillScreen(color uint16) {
	hx.FillRectangle(0, 0, hx.Width, hx.Height, color)
}

// FillRectangle is a very slow fill.
// Filling the entire screen takes 12+ seconds!
func (hx *HX8357) FillRectangle(x, y, w, h uint16, color uint16) {
	// log.Printf("%d x %d\n", st.Width, st.Height)
	// rudimentary clipping (drawChar w/big text requires this)
	if (x > hx.Width) || (y > hx.Height) {
		// log.Println("FillRectangle Rejected rectangle x,y")
		return
	}
	if (x + w) > hx.Width {
		// log.Println("ST7735 FillRectangle: adjusting w")
		w = hx.Width - x
	}
	if (y + h) > hx.Height {
		// log.Println("ST7735 FillRectangle: adjusting h")
		h = hx.Height - y
	}

	hx.SetAddrWindow(x, y, w, h)

	sp := hx.spi
	fi := sp.GetFTDI()

	fi.OutputHigh(hx.dc)

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
// The max that can be blitted at any one time is 65541 -headers = 65536
// Trying to copy more than 65K results in a "failed to write command".
//
// Thus in order to get the frameperiod down we need to only work sub blocks
//
// For example, the Action area could be:
// 200w x 160h = 32000, *2 = 64000
// OR
// 180w x 180h = 32400, *2 = 64800
// Or for higher frame rates perhaps:
// 128w x 128h = 16384 = *2 = 32768

// And the score or overlay areas would be the remaining areas:
//
//           320 width
// |--------------------------|
// |                 |--------|
// |    Action       |--------|
// |                 |-over---|
// |                 |-lay A--|
// |     area        |--------|
// |                 |--------|
// |--------------------------|  480 height
// |--------------------------|
// |--------------------------|
// |--------------------------|
// |--------------------------|
// |--------------------------|
// |--------- overlay B ------|
// |--------------------------|

// Blit writes line chunks so it is a bit faster than Blit3
func (hx *HX8357) Blit() {
	sp := hx.spi
	fi := sp.GetFTDI()

	chI := 0
	lI := 0
	// Each chunk is N "lines" of pushBuffer data
	for lineChunk := 0; lineChunk < hx.lineBlockSize; lineChunk++ {
		// fmt.Printf("%d, %d, %d\n", lineChunk, chI, lI)
		// copy all bytes from pB to cB
		for i := 0; i < hx.chunkSize; i++ {
			hx.chunkBuf[i] = hx.pushBuffer[chI]
			chI++
		}

		// Tell the controller chip that we are sending a
		// block/rectangle of data that starts at 0, lw and
		// the block is width x 10 dimensions. The chip will
		// then autoincrement the cursor.
		hx.SetAddrWindow(0, uint16(lI), hx.Width, uint16(hx.lines))

		fi.OutputHigh(hx.dc)
		sp.Write(hx.chunkBuf)
		lI += hx.lines
	}
}

// Blit3 is a bit faster in that it writes 1 horizontal line chunk
// at a time which is certainly faster than 1 pixel at a time.
func (hx *HX8357) Blit3() {
	sp := hx.spi
	fi := sp.GetFTDI()

	chunkSize := int(hx.Width) * bytesPerPixel
	var chunkBuf = make([]byte, chunkSize)

	is := 0
	ie := chunkSize

	for i := uint16(0); i < hx.Height; i++ {
		hx.SetAddrWindow(0, i, hx.Width, 1)

		fi.OutputHigh(hx.dc)

		// log.Printf("i: %d, %d, %d", i, is, ie)
		ji := 0
		for ii := int(is); ii < is+chunkSize; ii++ {
			chunkBuf[ji] = hx.pushBuffer[ii]
			ji++
		}
		// chunkBuf = hx.pushBuffer[is:ie]
		// fmt.Printf("%v\n", chunkBuf)
		// println("------------")
		sp.Write(chunkBuf)
		// fi.OutputLow(hx.dc)

		is = ie
		ie += chunkSize
	}
	// sp.Write(hx.pushBuffer)
}

// Blit2 writes the contents of the displayBuffer directly to the display as fast as it can!
// At time of writing could be improved for speed by going directly to SPI interface
// itself but for now fast enough doing it this way for most needs
func (hx *HX8357) Blit2() {
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

	sp := hx.spi
	fi := sp.GetFTDI()

	hx.SetAddrWindow(0, 0, hx.Width, hx.Height)

	fi.OutputHigh(hx.dc)

	sp.Write(hx.pushBuffer)
}

// DrawPixelToBuf draws to screen buffer only. You will need to eventually
// call Blit() to see anything.
func (hx *HX8357) DrawPixelToBuf(x, y uint16, color uint16) {

	if x >= hx.Width {
		x = hx.Width - 1
	}

	if y >= hx.Height {
		y = hx.Height - 1
	}

	if x < 0 {
		x = 0
	}

	if y < 0 {
		y = 0
	}

	// write to screen buffer memory instead, this is quick and dirty, presumes always using
	// RGB565 (16bit per pixel colour)
	// Calculate memory location based on screen width and height
	bufOff := int(y)*int(hx.Width) + int(x)
	hx.pushBuffer[bufOff*2] = byte(color >> 8 & 0xff) // High
	hx.pushBuffer[bufOff*2+1] = byte(color & 0xff)    // Low
}

// DrawVLineToBuf draws a vertical line
func (hx *HX8357) DrawVLineToBuf(x, y, h uint16, color uint16) {

	// Rudimentary clipping
	if x >= hx.Width {
		x = hx.Width - 1
	}

	if y >= hx.Height {
		y = hx.Height - 1
	}

	if x < 0 {
		x = 0
	}

	if y < 0 {
		y = 0
	}

	msb := byte((color >> 8) & 0xff) // High
	lsb := byte(color & 0xff)        // Low

	for iy := y; iy < y+h; iy++ {
		bufOff := int(iy)*int(hx.Width) + int(x)

		hx.pushBuffer[bufOff*2] = msb
		hx.pushBuffer[bufOff*2+1] = lsb
	}
}

// DrawHLineToBuf draws a horizontal line
func (hx *HX8357) DrawHLineToBuf(x, y, w uint16, color uint16) {

	// Rudimentary clipping
	if x >= hx.Width {
		x = hx.Width - 1
	}

	if y >= hx.Height {
		y = hx.Height - 1
	}

	if x < 0 {
		x = 0
	}

	if y < 0 {
		y = 0
	}

	msb := byte((color >> 8) & 0xff) // High
	lsb := byte(color & 0xff)        // Low

	for ix := x; ix < x+w; ix++ {
		bufOff := int(y)*int(hx.Width) + int(ix)

		hx.pushBuffer[bufOff*2] = msb
		hx.pushBuffer[bufOff*2+1] = lsb
	}
}

// FillScreenToBuf fills the entire display area with "color"
func (hx *HX8357) FillScreenToBuf(color uint16) {
	hx.FillRectangleToBuf(0, 0, hx.Width, hx.Height, color)
	// fmt.Printf("color: %16b\n", color)
	// hx.FillRectangleToBuf(0, 0, hx.Width, 2, color)
}

// FillRectangleToBuf fills a rectangle in the screen buffer
func (hx *HX8357) FillRectangleToBuf(x, y, w, h uint16, color uint16) {
	// log.Printf("%d x %d\n", st.Width, st.Height)
	// rudimentary clipping (drawChar w/big text requires this)
	if x >= hx.Width {
		x = hx.Width - 1
	}

	if y >= hx.Height {
		y = hx.Height - 1
	}

	if x < 0 {
		x = 0
	}

	if y < 0 {
		y = 0
	}

	sw := int(x + w)
	sh := int(y + h)
	if sw > int(hx.Width) {
		sw = int(hx.Width) - 1
	}

	if sh > int(hx.Height) {
		sh = int(hx.Height) - 1
	}

	msb := byte((color >> 8) & 0xff)
	lsb := byte(color & 0xff)

	j := 0
	iw := int(hx.Width)
	pl := len(hx.pushBuffer)

fillLoop:
	for sy := int(y); sy < sh; sy++ {
		for sx := int(x); sx < sw; sx++ {
			j = (sy*iw + sx) * bytesPerPixel
			// fmt.Printf("j: %d, sx:%d, sy:%d\n", j, sx, sy)
			if j >= pl {
				// fmt.Printf("j: %d\n", j)
				break fillLoop
			}
			hx.pushBuffer[j] = msb
			j++
			if j >= pl {
				// fmt.Printf("j2: %d\n", j)
				break fillLoop
			}
			hx.pushBuffer[j] = lsb
		}
	}
}
