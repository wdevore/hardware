package ssd1351

import (
	"errors"
	"log"
	"time"

	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/gpio"
	"github.com/wdevore/hardware/spi"
)

// Registers
const (
	SETCOLUMN      = 0x15
	SETROW         = 0x75
	WRITERAM       = 0x5C
	READRAM        = 0x5D
	SETREMAP       = 0xA0
	STARTLINE      = 0xA1
	DISPLAYOFFSET  = 0xA2
	DISPLAYALLOFF  = 0xA4
	DISPLAYALLON   = 0xA5
	NORMALDISPLAY  = 0xA6
	INVERTDISPLAY  = 0xA7
	FUNCTIONSELECT = 0xAB
	DISPLAYOFF     = 0xAE
	DISPLAYON      = 0xAF
	PRECHARGE      = 0xB1
	DISPLAYENHANCE = 0xB2
	CLOCKDIV       = 0xB3
	SETVSL         = 0xB4
	SETGPIO        = 0xB5
	PRECHARGE2     = 0xB6
	SETGRAY        = 0xB8
	USELUT         = 0xB9
	PRECHARGELEVEL = 0xBB
	VCOMH          = 0xBE
	CONTRASTABC    = 0xC1
	CONTRASTMASTER = 0xC7
	MUXRATIO       = 0xCA
	COMMANDLOCK    = 0xFD
	HORIZSCROLL    = 0x96
	STOPSCROLL     = 0x9E
	STARTSCROLL    = 0x9F

	// Timing Delays
	DELAYS_HWFILL = 3
	DELAYS_HWLINE = 1
)

var (
	colorPush     = []byte{0x00, 0x00}
	writeBuf      = []byte{0x00}
	addWindowBuf  = []byte{0x00, 0x00, 0x00, 0x00}
	bytesPerPixel = 2
)

// SSD1351 represents the OLED ssd1351 controller chip.
type SSD1351 struct {
	// Uses the USB FTDI232 SPI object
	spi *spi.FtdiSPI

	dc    gpio.Pin // Data/Command pin
	reset gpio.Pin

	dimensions devices.Dimensions

	Width  uint8
	Height uint8

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

// NewSSD1351 creates driver
func NewSSD1351(dataCommand, reset gpio.Pin, dimensions devices.Dimensions) *SSD1351 {
	sd := new(SSD1351)

	sd.dc = dataCommand
	sd.reset = reset
	sd.dimensions = dimensions

	if dimensions == devices.D128x128 {
		sd.Width = 128
		sd.Height = 128
	} else {
		sd.Width = 128
		sd.Height = 96
	}

	pixels := int(sd.Width) * int(sd.Height)

	// A buffer of bytes
	// RRRRRGGG-GGGBBBBB RRRRRGGG-GGGBBBBB...

	sd.pushBuffer = make([]byte, pixels*bytesPerPixel)

	return sd
}

// Initialize configures FTDI and SPI, and initializes HX8357
// Vendor/Product example would be: 0x0403, 0x06014 for the FTDI chip
// A clock frequency of 0 means default to max = 30MHz
func (sd *SSD1351) Initialize(vender, product, clockFreq int, chipSelect gpio.Pin) error {

	// Create a SPI interface from the FT232H
	sd.spi = spi.NewSPI(vender, product, false)
	//sd.spi.DebugInit()

	if sd.spi == nil {
		return errors.New("SSD1351: Failed to create SPI object")
	}

	if clockFreq == 0 {
		clockFreq = devices.Max30MHz
	}
	log.Printf("SSD1351: Configuring for a clock of (%d)MHz\n", clockFreq/1000000)

	err := sd.configure(chipSelect, clockFreq)
	if err != nil {
		return err
	}

	log.Println("SSD1351: configured.")

	return nil
}

// Configure sets up the SPI component and initializes the SSD1351
func (sd *SSD1351) configure(chipSelect gpio.Pin, clockFreq int) error {
	log.Println("SSD1351: Configuring SPI")
	err := sd.spi.Configure(chipSelect, clockFreq, spi.Mode0, spi.MSBFirst)

	sd.spi.CSActiveLow = true

	if err != nil {
		log.Println("SSD1351: Configure FAILED.")
		return err
	}

	sd.commonInit()

	return nil
}

// Close turns off display and closes SPI.
func (sd *SSD1351) Close() error {
	sd.SetDisplayOn(false)
	return sd.spi.Close()
}

// commonInit setups common pin configurations
func (sd *SSD1351) commonInit() error {
	sp := sd.spi

	// The SSD1351 communicates with TFT device through the FTDI235H device
	// via the SPI protocol. However, the SPI protocol only accounts for, at most, 4 pins, anything
	// else needs to added manually--and controlled manually.

	// Setup extra pins for D/C and Reset. For this we need to interface with the FTDI chip
	fi := sp.GetFTDI()

	pins := []gpio.PinConfiguration{
		{Pin: sd.dc, Direction: gpio.Output, Value: gpio.Z},
		{Pin: sd.reset, Direction: gpio.Output, Value: gpio.Z},
	}
	fi.ConfigPins(pins, true)

	// toggle RST low to reset and CS low so it'll listen to us
	sp.AssertChipSelect()

	if sd.reset != gpio.NoPin {
		fi.ConfigPin(sd.reset, gpio.Output)

		fi.OutputHigh(sd.reset)
		time.Sleep(time.Millisecond * 500)

		fi.OutputLow(sd.reset)
		time.Sleep(time.Millisecond * 500)

		fi.OutputHigh(sd.reset)
		time.Sleep(time.Millisecond * 500)
	}

	sd.issueCommands()

	return nil
}

// ----------------------------------------------------
// Commands
// ----------------------------------------------------
func (sd *SSD1351) issueCommands2() {
	sd.WriteCommand(COMMANDLOCK) // set command lock
	sd.WriteData(0x12)
	sd.WriteCommand(COMMANDLOCK) // set command lock
	sd.WriteData(0xB1)

	// Sleep mode on
	sd.WriteCommand(DISPLAYOFF) // 0xAE

	sd.WriteCommand(CLOCKDIV) // 0xB3
	sd.WriteCommand(0xF1)     // 7:4 = Oscillator Frequency, 3:0 = CLK Div Ratio (A[3:0]+1 = 1..16)

	sd.WriteCommand(MUXRATIO)
	sd.WriteData(0x7f) // 127 or 0x7f

	sd.WriteCommand(SETREMAP)
	sd.WriteData(0x74) // default = 0x40, 0x74

	sd.WriteCommand(SETCOLUMN)
	sd.WriteData(0x00)
	sd.WriteData(0x7F)
	sd.WriteCommand(SETROW)
	sd.WriteData(0x00)
	sd.WriteData(0x7F)

	// Vertical scroll by RAM
	sd.WriteCommand(STARTLINE) // 0xA1
	if sd.dimensions == devices.D128x96 {
		sd.WriteData(96)
	} else {
		sd.WriteData(0)
	}

	// Vertical scroll by row
	sd.WriteCommand(DISPLAYOFFSET) // 0xA2
	sd.WriteData(0x0)

	sd.WriteCommand(SETGPIO)
	sd.WriteData(0x00)

	sd.WriteCommand(FUNCTIONSELECT)
	sd.WriteData(0x01) // internal (diode drop)

	sd.WriteCommand(PRECHARGE) // 0xB1 = Phase length
	sd.WriteCommand(0x32)

	sd.WriteCommand(VCOMH) // 0xBE
	sd.WriteCommand(0x05)

	sd.WriteCommand(NORMALDISPLAY) // 0xA6

	sd.WriteCommand(CONTRASTABC)
	sd.WriteData(0xC8)
	sd.WriteData(0x80)
	sd.WriteData(0xC8)

	sd.WriteCommand(CONTRASTMASTER)
	sd.WriteData(0x0F)

	sd.WriteCommand(SETVSL)
	sd.WriteData(0xA0)
	sd.WriteData(0xB5)
	sd.WriteData(0x55)

	// sd.WriteCommand(SSETPHASELENGTH)
	// sd.WriteData(0x32)

	sd.WriteCommand(PRECHARGE2)
	sd.WriteData(0x01)

	// TODO add Gamma lookup table

	// TODO Clear screen

	//writeData(0x01); // external bias

	// Sleep mode off
	sd.WriteCommand(DISPLAYON) //--turn on oled panel
}

func (sd *SSD1351) issueCommands() {
	// sd.WriteCommand(COMMANDLOCK) // set command lock
	// sd.WriteData(0x12)
	sd.WriteCommand(COMMANDLOCK) // set command lock
	sd.WriteData(0xB1)           // All commands are accessible

	sd.SetDisplayOn(false)

	// If you use 3.3V then the display has a refresh flicker. I used
	// 4.5->5V and the flicker disappears.
	sd.WriteCommand(CLOCKDIV) // 0xB3
	// freq      clk div
	// 7 6 5 4   3 2 1 0  --> typically 0xF1
	sd.WriteCommand(0xf1) // 7:4 = Oscillator Frequency, 3:0 = CLK Div Ratio (A[3:0]+1 = 1..16)

	sd.WriteCommand(MUXRATIO)
	sd.WriteData(0x7f) // 127 = 0x7f = reset value = 1/128 duty

	// Vertical scroll by row
	sd.WriteCommand(DISPLAYOFFSET) // 0xA2
	sd.WriteData(0x00)

	// Vertical scroll by RAM
	sd.WriteCommand(STARTLINE) // 0xA1
	if sd.dimensions == devices.D128x96 {
		sd.WriteData(96)
	} else {
		sd.WriteData(0x0)
	}

	// Configure automatic address increment
	sd.WriteCommand(SETREMAP)
	//        A
	//        B
	//        C
	// 01 11 0100
	//  ^    ^  ^
	//  |    |  |--- 0=Horz auto addr increment (default), 1=Vert addr inc
	//  |    |------ Reserved
	//  |
	//  |---- color depth (upper 2 bits), 00/01 = 65k colors
	//
	sd.WriteData(0x74) // default = 0x40, 0x74 or 0xb4

	sd.WriteCommand(SETGPIO)
	sd.WriteData(0x00)

	sd.WriteCommand(FUNCTIONSELECT) // 0xAB
	sd.WriteData(0x01)              // internal Vdd regulator: (diode drop)

	sd.WriteCommand(SETVSL) // Segment low voltage
	sd.WriteData(0xA0)      // Enable external VSL. 0xA2 = internal
	sd.WriteData(0xB5)
	sd.WriteData(0x55)

	sd.WriteCommand(CONTRASTABC) // Contrast current
	sd.WriteData(0xC8)           // A = 0xC8 or 0x8A
	sd.WriteData(0x80)           // B = 0x80 or 0x51
	sd.WriteData(0xC8)           // C = 0xC8 or 0x8A

	sd.WriteCommand(CONTRASTMASTER)
	sd.WriteData(0x0f) // Maximum

	// TODO add Gamma lookup table (optional)

	sd.WriteCommand(PRECHARGE) // 0xB1 = Phase length
	// A[3:0] = phase 1, A[7:4] = phase2, typically 0x32
	sd.WriteCommand(0x32)

	// sd.WriteCommand(SSETPHASELENGTH)
	// sd.WriteData(0x32)

	sd.WriteCommand(DISPLAYENHANCE)
	sd.WriteCommand(0xA4) // 0xa4
	sd.WriteCommand(0x00)
	sd.WriteCommand(0x00)

	sd.WriteCommand(PRECHARGELEVEL)
	sd.WriteCommand(0x17)

	sd.WriteCommand(PRECHARGE2) // Phase3 of driving oled pixel
	sd.WriteData(0x01)          // 0x01 or 0x08

	// Gamma values need to be specified before using this
	// otherwise the display is "washed" out. The gamma commands are: B8/B9
	// sd.WriteCommand(SETGRAY) // Phase4 of driving oled pixel
	// sd.WriteData(0x01)

	sd.WriteCommand(VCOMH) // 0xBE
	sd.WriteCommand(0x05)

	sd.WriteCommand(NORMALDISPLAY) // 0xA6 = display mode

	// sd.WriteCommand(SETCOLUMN)
	// sd.WriteData(0x00)
	// sd.WriteData(0x7F)
	// sd.WriteCommand(SETROW)
	// sd.WriteData(0x00)
	// sd.WriteData(0x7F)

	//writeData(0x01); // external bias

	sd.SetDisplayOn(true)
}

// SetDisplayOn turns on/off display
func (sd *SSD1351) SetDisplayOn(on bool) {
	if on {
		// Sleep mode off
		sd.WriteCommand(DISPLAYON) //--turn on oled panel
	} else {
		// Sleep mode on
		sd.WriteCommand(DISPLAYOFF) // 0xAE
	}
}

// SetPowerSleepOn turns on/off display and enables power save.
func (sd *SSD1351) SetPowerSleepOn(on bool) {
	if on {
		// Sleep mode on
		sd.WriteCommand(DISPLAYOFF)     // 0xAE
		sd.WriteCommand(FUNCTIONSELECT) // 0xAB
		sd.WriteData(0x0)               // internal Vdd regulator: (diode drop)
	} else {
		sd.WriteCommand(FUNCTIONSELECT) // 0xAB
		sd.WriteData(0x01)              // internal Vdd regulator: (diode drop)
		// Sleep mode off
		sd.WriteCommand(DISPLAYON) //--turn on oled panel
	}
}

// ----------------------------------------------------
// Rotation
// ----------------------------------------------------

// SetRotation re-orients the display at 90 degree rotations.
// Typically this method is called last during the initialization sequence.
func (sd *SSD1351) SetRotation(orieo devices.RotationMode) {
}

// InvertDisplay inverts the display colors
func (sd *SSD1351) InvertDisplay(inv bool) {
	if inv {
		sd.WriteCommand(INVERTDISPLAY)
	} else {
		sd.WriteCommand(NORMALDISPLAY)

	}
}

// ----------------------------------------------------
// Writing
// ----------------------------------------------------

// SetConstantCSAssert sets the constant assert flag.
// If you have an the Adafruit SSD1351 then you most likely
// have an micro sd card present. If you aren't using it then
// you can leave CS low for the entire time and thus save
// on bandwidth.
func (sd *SSD1351) SetConstantCSAssert(constant bool) {
	sd.spi.ConstantCSAssert = constant
}

// WriteCommand writes a command via SPI protocol
func (sd *SSD1351) WriteCommand(command byte) error {
	sp := sd.spi
	fi := sp.GetFTDI()

	fi.OutputLow(sd.dc) // Low = command

	writeBuf[0] = command
	err := sd.spi.Write(writeBuf)
	if err != nil {
		log.Println("Failed to write command.")
		return err
	}

	return nil
}

// WriteData writes data to the device via SPI
func (sd *SSD1351) WriteData(data byte) {
	sp := sd.spi
	fi := sp.GetFTDI()
	fi.OutputHigh(sd.dc) // High = data

	writeBuf[0] = data
	sp.Write(writeBuf)
}

// WriteDataChunk is a slightly more efficient version of WriteData
func (sd *SSD1351) WriteDataChunk(data []byte) {
	sp := sd.spi
	fi := sp.GetFTDI()
	fi.OutputHigh(sd.dc) // High = data

	sp.Write(data)
}

// ----------------------------------------------------
// Graphics Unbuffered
// ----------------------------------------------------

// SetAddrWindow sets row and column address of where pixels will be written.
// (aka setDrawPosition or Goto)
func (sd *SSD1351) SetAddrWindow(x, y, w, h uint8) {
	if (x >= sd.Width) || (y >= sd.Height) {
		return
	}

	// set x and y coordinate
	sd.WriteCommand(SETCOLUMN)
	sd.WriteData(x)
	sd.WriteData(w - 1)

	sd.WriteCommand(SETROW)
	sd.WriteData(y)
	sd.WriteData(h - 1)

	sd.WriteCommand(WRITERAM)
}

// PushColor writes a 16bit color value based on the current cursor/draw position.
// The "cursor" position is set by SetAddrWindow()
// (aka writePixel)
func (sd *SSD1351) PushColor(color uint16) {
	sp := sd.spi
	fi := sp.GetFTDI()

	fi.OutputHigh(sd.dc)

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)
	sp.Write(colorPush)
}

// DrawPixel draws to device only.
func (sd *SSD1351) DrawPixel(x, y uint8, color uint16) {

	if (x < 0) || (x > sd.Width) || (y < 0) || (y > sd.Height) {
		return
	}

	sp := sd.spi
	fi := sp.GetFTDI()

	// First set "cursor"/"draw position"
	sd.SetAddrWindow(x, y, 1, 1)

	fi.OutputHigh(sd.dc)

	// Now draw.
	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)
	sp.Write(colorPush)
}

// DrawFastVLine draws a vertical line only
func (sd *SSD1351) DrawFastVLine(x, y, h uint8, color uint16) {

	// Rudimentary clipping
	if (x > sd.Width) || (y > sd.Height) {
		log.Println("DrawFastVLine: Rejected x,y")
		return
	}

	if (y + h) > sd.Height {
		h = sd.Height - y
	}

	sd.SetAddrWindow(x, y, x, y+h)

	sp := sd.spi
	fi := sp.GetFTDI()

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)

	fi.OutputHigh(sd.dc)

	for h > 0 {
		sp.Write(colorPush)
		h--
	}
}

// DrawFastHLine draws a horizontal line only
func (sd *SSD1351) DrawFastHLine(x, y, w uint8, color uint16) {

	// Rudimentary clipping
	if (x > sd.Width) || (y > sd.Height) {
		log.Println("DrawFastHLine: Rejected x,y")
		return
	}

	if (x + w) > sd.Width {
		w = sd.Width - x
	}

	sd.SetAddrWindow(x, y, x+w, y)

	sp := sd.spi
	fi := sp.GetFTDI()

	colorPush[0] = byte((color >> 8) & 0xff)
	colorPush[1] = byte(color & 0xff)

	fi.OutputHigh(sd.dc)

	for w > 0 {
		sp.Write(colorPush)
		w--
	}
}

// FillScreen fills the entire display area with "color"
func (sd *SSD1351) FillScreen(color uint16) {
	sd.FillRectangle(0, 0, sd.Width, sd.Height, color)
}

// FillRectangle is a very slow fill.
// Filling the entire screen takes 12+ seconds!
func (sd *SSD1351) FillRectangle(x, y, w, h uint8, color uint16) {
	// log.Printf("%d x %d\n", st.Width, st.Height)
	// rudimentary clipping (drawChar w/big text requires this)
	if (x > sd.Width) || (y > sd.Height) {
		// log.Println("FillRectangle Rejected rectangle x,y")
		return
	}
	if (x + w) > sd.Width {
		// log.Println("SSD1351 FillRectangle: adjusting w")
		w = sd.Width - x
	}
	if (y + h) > sd.Height {
		// log.Println("SSD1351 FillRectangle: adjusting h")
		h = sd.Height - y
	}

	sd.SetAddrWindow(x, y, w, h)

	sp := sd.spi
	fi := sp.GetFTDI()

	fi.OutputHigh(sd.dc)

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
func (sd *SSD1351) Blit() {
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

	sp := sd.spi
	fi := sp.GetFTDI()

	sd.SetAddrWindow(0, 0, sd.Width, sd.Height)

	fi.OutputHigh(sd.dc)

	sp.Write(sd.pushBuffer)
}

// DrawPixelToBuf draws to screen buffer only. You will need to eventually
// call Blit() to see anything.
func (sd *SSD1351) DrawPixelToBuf(x, y uint8, color uint16) {

	if x >= sd.Width {
		x = sd.Width - 1
	}

	if y >= sd.Height {
		y = sd.Height - 1
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
	bufOff := int(y)*int(sd.Width) + int(x)
	sd.pushBuffer[bufOff*2] = byte(color >> 8 & 0xff) // High
	sd.pushBuffer[bufOff*2+1] = byte(color & 0xff)    // Low
}

// DrawVLineToBuf draws a vertical line
func (sd *SSD1351) DrawVLineToBuf(x, y, h uint8, color uint16) {

	// Rudimentary clipping
	if x >= sd.Width {
		x = sd.Width - 1
	}

	if y >= sd.Height {
		y = sd.Height - 1
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
		bufOff := int(iy)*int(sd.Width) + int(x)

		sd.pushBuffer[bufOff*2] = msb
		sd.pushBuffer[bufOff*2+1] = lsb
	}
}

// DrawHLineToBuf draws a horizontal line
func (sd *SSD1351) DrawHLineToBuf(x, y, w uint8, color uint16) {

	// Rudimentary clipping
	if x >= sd.Width {
		x = sd.Width - 1
	}

	if y >= sd.Height {
		y = sd.Height - 1
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
		bufOff := int(y)*int(sd.Width) + int(ix)

		sd.pushBuffer[bufOff*2] = msb
		sd.pushBuffer[bufOff*2+1] = lsb
	}
}

// FillScreenToBuf fills the entire display area with "color"
func (sd *SSD1351) FillScreenToBuf(color uint16) {
	sd.FillRectangleToBuf(0, 0, sd.Width, sd.Height, color)
	// fmt.Printf("color: %16b\n", color)
	// sd.FillRectangleToBuf(0, 0, sd.Width, 2, color)
}

// FillRectangleToBuf fills a rectangle in the screen buffer
func (sd *SSD1351) FillRectangleToBuf(x, y, w, h uint8, color uint16) {
	// log.Printf("%d x %d\n", st.Width, st.Height)
	// rudimentary clipping (drawChar w/big text requires this)
	if x >= sd.Width {
		x = sd.Width - 1
	}

	if y >= sd.Height {
		y = sd.Height - 1
	}

	if x < 0 {
		x = 0
	}

	if y < 0 {
		y = 0
	}

	sw := int(x + w)
	sh := int(y + h)
	if sw > int(sd.Width) {
		sw = int(sd.Width) - 1
	}

	if sh > int(sd.Height) {
		sh = int(sd.Height) - 1
	}

	msb := byte((color >> 8) & 0xff)
	lsb := byte(color & 0xff)

	j := 0
	iw := int(sd.Width)
	pl := len(sd.pushBuffer)

fillLoop:
	for sy := int(y); sy < sh; sy++ {
		for sx := int(x); sx < sw; sx++ {
			j = (sy*iw + sx) * bytesPerPixel
			// fmt.Printf("j: %d, sx:%d, sy:%d\n", j, sx, sy)
			if j >= pl {
				// fmt.Printf("j: %d\n", j)
				break fillLoop
			}
			sd.pushBuffer[j] = msb
			j++
			if j >= pl {
				// fmt.Printf("j2: %d\n", j)
				break fillLoop
			}
			sd.pushBuffer[j] = lsb
		}
	}
}
