package ra8875

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/gpio"
	"github.com/wdevore/hardware/spi"
)

// SoftRAIO8875 uses SoftSPI
type SoftRAIO8875 struct {
	RA8875Base

	spi *spi.SoftSPI

	reset gpio.Pin
}

// NewSoftRA8875 creates an un-initialized RA8875 device driver
func NewSoftRA8875(dimensions devices.Dimensions) RA8875 {
	ra := new(SoftRAIO8875)
	ra.dimensions = dimensions
	return ra
}

// NewSoftRA8875Default creates a default/typical configuration when
// using the FTDI232H GPIO USB device.
func NewSoftRA8875Default(dimensions devices.Dimensions) RA8875 {
	ra := new(SoftRAIO8875)
	ra.dimensions = dimensions

	err := ra.initialize(0x0403, 0x06014, 2000000, gpio.DefaultPin)

	if err != nil {
		panic("RA8875: Failed to default initialize.")
	}

	return ra
}

// -----------------------------------------------------------
// Control API BEGIN
// -----------------------------------------------------------

// Quit signals any waiting/polling to stop
func (ra SoftRAIO8875) Quit() {
	ra.quit = true
}

// DisplayOn turns display on or off
func (ra *SoftRAIO8875) DisplayOn(on bool) {
	if on {
		ra.writeReg(PWRR, PWRR_NORMAL|PWRR_DISPON)
	} else {
		ra.writeReg(PWRR, PWRR_NORMAL|PWRR_DISPOFF)
	}
}

// GPIOX enables TFT - display enable tied to GPIOX
func (ra *SoftRAIO8875) GPIOX(on bool) {
	if on {
		ra.writeReg(GPIOX, 1)
	} else {
		ra.writeReg(GPIOX, 0)
	}
}

// PWM1out writes a duty value to pwm1
func (ra *SoftRAIO8875) PWM1out(p uint8) {
	ra.writeReg(P1DCR, p)
}

// PWM2out writes a duty value to pwm2
func (ra *SoftRAIO8875) PWM2out(p uint8) {
	ra.writeReg(P2DCR, p)
}

// PWM1config writes a duty value to config
func (ra *SoftRAIO8875) PWM1config(on bool, clock uint8) {
	if on {
		ra.writeReg(P1CR, P1CR_ENABLE|(clock&0xF))
	} else {
		ra.writeReg(P1CR, P1CR_DISABLE|(clock&0xF))
	}
}

func (ra *SoftRAIO8875) waitPoll(regname, waitflag uint8) bool {
	/* Wait for the command to finish */
	start := time.Now()
	timeout := 1 // 1 seconds
	duration := time.Duration(timeout) * time.Second

	for time.Now().Sub(start) <= duration {
		if ra.quit {
			return false
		}

		temp, err := ra.readReg(regname)
		// log.Printf("waitPoll: %d\n", temp)
		if err != nil {
			log.Print(err)
		}

		if temp&waitflag == 1 {
			return true
		}
	}
	// return false // MEMEFIX: yeah i know, unreached! - add timeout?

	// msg := fmt.Sprintf("SoftRAIO8875: waitPoll: Timed out while polling for (%d) bytes!\n", waitflag)
	// fmt.Print(msg)
	return false
}

func (ra *SoftRAIO8875) DebugTrigPulse() {
	ra.spi.TriggerPulse()
}

// -----------------------------------------------------------
// Control API END
// -----------------------------------------------------------

// -----------------------------------------------------------
// High level graphic API BEGIN
// -----------------------------------------------------------

// FillScreen fills the screen with the spefied RGB565 color
//    RGB565 color to use when drawing the pixel
func (ra SoftRAIO8875) FillScreen(color uint16) {
	ra.DrawRectangle(0, 0, ra.Width-1, ra.Height-1, color, true)
}

// -----------------------------------------------------------
// High level graphic API END
// -----------------------------------------------------------

// -----------------------------------------------------------
// Text API BEGIN
// -----------------------------------------------------------

// TextMode sets the display in text mode (as opposed to graphics mode)
func (ra SoftRAIO8875) TextMode() error {
	/* Set text mode */
	ra.writeCommand(MWCR0)

	temp, err := ra.readData()
	if err != nil {
		return err
	}
	temp |= MWCR0_TXTMODE // Set bit 7
	ra.writeData(temp)

	/* Select the internal (ROM) font */
	ra.writeCommand(0x21)
	temp, err = ra.readData()
	if err != nil {
		return err
	}

	t := int(temp)
	t &= ^((1 << 7) | (1 << 5)) // Clear bits 7 and 5
	ra.writeData(byte(t))

	return nil
}

// TextSetCursor set cursor location
//   x position of the cursor (in pixels, 0..1023)
//   y position of the cursor (in pixels, 0..511)
func (ra SoftRAIO8875) TextSetCursor(x, y uint16) {
	/* Set cursor location */
	ra.writeCommand(0x2A)
	ra.writeData(byte(x & 0xFF)) // lower byte
	ra.writeCommand(0x2B)
	ra.writeData(byte(x >> 8)) // upper byte
	ra.writeCommand(0x2C)
	ra.writeData(byte(y & 0xFF))
	ra.writeCommand(0x2D)
	ra.writeData(byte(y >> 8))
}

// TextColor  sets the fore and background color when rendering text
func (ra SoftRAIO8875) TextColor(foreColor, bgColor uint16) {
	/* Set Fore Color */
	ra.writeCommand(0x63)
	ra.writeData(byte((foreColor & 0xf800) >> 11))
	ra.writeCommand(0x64)
	ra.writeData(byte((foreColor & 0x07e0) >> 5))
	ra.writeCommand(0x65)
	ra.writeData(byte(foreColor & 0x001f))

	/* Set Background Color */
	ra.writeCommand(0x60)
	ra.writeData(byte((bgColor & 0xf800) >> 11))
	ra.writeCommand(0x61)
	ra.writeData(byte((bgColor & 0x07e0) >> 5))
	ra.writeCommand(0x62)
	ra.writeData(byte(bgColor & 0x001f))

	/* Clear transparency flag */
	ra.writeCommand(0x22)

	temp, err := ra.readData()
	if err != nil {
		log.Print(err)
	}

	t := int(temp)
	t &= ^(1 << 6) // Clear bit 6
	ra.writeData(byte(t))
}

// TextTransparent sets the fore color when rendering text with a transparent bg
//   an RGB565 color to use when rendering the text
func (ra SoftRAIO8875) TextTransparent(foreColor uint16) {
	/* Set Fore Color */
	ra.writeCommand(0x63)
	ra.writeData(byte((foreColor & 0xf800) >> 11))
	ra.writeCommand(0x64)
	ra.writeData(byte((foreColor & 0x07e0) >> 5))
	ra.writeCommand(0x65)
	ra.writeData(byte(foreColor & 0x001f))

	/* Set transparency flag */
	ra.writeCommand(0x22)

	temp, err := ra.readData()
	if err != nil {
		log.Print(err)
	}

	t := int(temp)
	t |= (1 << 6) // Set bit 6
	ra.writeData(byte(t))
}

// TextEnlarge sets the text enlarge settings, using one of the following values:
//   0 = 1x zoom
//   1 = 2x zoom
//   2 = 3x zoom
//   3 = 4x zoom
//   a zoom factor (0..3 for 1-4x zoom)
func (ra SoftRAIO8875) TextEnlarge(scale int) {
	if scale > 3 {
		scale = 3
	}

	/* Set font size flags */
	ra.writeCommand(0x22)

	temp, err := ra.readData()
	if err != nil {
		log.Print(err)
	}

	t := int(temp)

	t &= ^(0xF) // Clears bits 0..3
	t |= scale << 2
	t |= scale
	ra.writeData(byte(t))

	ra.textScale = scale
}

// TextWrite renders some text on the screen when in text mode
func (ra SoftRAIO8875) TextWrite(text string) {
	ra.writeCommand(MRWC)
	for _, ch := range text {
		// log.Printf("%d, %d\n", pos, ch)
		ra.writeData(byte(ch))
		time.Sleep(time.Millisecond)
	}
}

// -----------------------------------------------------------
// Text API END
// -----------------------------------------------------------

// -----------------------------------------------------------
// Graphic API BEGIN
// -----------------------------------------------------------

// DrawRectangle fills the screen with the spefied RGB565 color
//    RGB565 color to use when drawing the pixel
func (ra SoftRAIO8875) DrawRectangle(x, y, w, h, color uint16, filled bool) {
	/* Set X */
	ra.writeCommand(0x91)
	ra.writeData(byte(x & 0xff)) // lower
	ra.writeCommand(0x92)
	ra.writeData(byte(x >> 8)) // upper

	/* Set Y */
	ra.writeCommand(0x93)
	ra.writeData(byte(y & 0xff))
	ra.writeCommand(0x94)
	ra.writeData(byte(y >> 8))

	/* Set X1 */
	ra.writeCommand(0x95)
	ra.writeData(byte(w & 0xff))
	ra.writeCommand(0x96)
	ra.writeData(byte(w >> 8))

	/* Set Y1 */
	ra.writeCommand(0x97)
	ra.writeData(byte(h & 0xff))
	ra.writeCommand(0x98)
	ra.writeData(byte(h >> 8))

	/* Set Color */
	ra.writeCommand(0x63)
	ra.writeData(byte((color & 0xf800) >> 11))
	ra.writeCommand(0x64)
	ra.writeData(byte((color & 0x07e0) >> 5))
	ra.writeCommand(0x65)
	ra.writeData(byte(color & 0x001f))

	/* Draw! */
	ra.writeCommand(DCR)
	if filled {
		ra.writeData(0xB0)
	} else {
		ra.writeData(0x90)
	}

	/* Wait for the command to finish */
	ra.waitPoll(DCR, DCR_LINESQUTRI_STATUS)
}

// -----------------------------------------------------------
// Graphic API END
// -----------------------------------------------------------

// Initialize configures FTDI and SPI, and initializes RA8875
// Vendor/Product example would be: 0x0403, 0x06014 for the FTDI chip
// A clock frequency of 0 means default to max = 30MHz
func (ra *SoftRAIO8875) initialize(vender, product, clockFreq int, chipSelect gpio.Pin) error {
	// Create a SPI interface from the FT232H
	ra.spi = spi.NewSoftSPI(0x0403, 0x06014, false)

	ra.spi.ConstantCSAssert = false

	if ra.spi == nil {
		return errors.New("RA8875: Failed to create SPI object")
	}

	if clockFreq == 0 {
		clockFreq = devices.Max30MHz
	}
	log.Printf("RA8875: Configuring for a clock of (%d)MHz\n", clockFreq/1000000)

	err := ra.configure(clockFreq)
	if err != nil {
		return err
	}

	return nil
}

// Configure sets up the SPI component and initializes the RA8875
func (ra *SoftRAIO8875) configure(clockFreq int) error {
	log.Println("SoftRAIO8875: Configuring SPI")
	err := ra.spi.Configure(clockFreq, spi.MSBFirst)

	if err != nil {
		log.Println("RA8875: Configure FAILED.")
		return err
	}

	log.Println("RA8875: initReset")
	err = ra.initReset()
	if err != nil {
		log.Println("RA8875: Configure FAILED.")
		return err
	}

	log.Println("RA8875: initDriver")
	ra.initDriver()

	return nil
}

// Close closes the SPI object
func (ra *SoftRAIO8875) Close() error {
	return ra.spi.Close()
}

// Init setups common pin configurations and resets
func (ra *SoftRAIO8875) initReset() error {
	sp := ra.spi

	// toggle RST low to reset and CS low so it'll listen to us
	sp.AssertChipSelect()

	sp.SetReset(false)
	time.Sleep(time.Millisecond * 100)

	sp.SetReset(true)
	time.Sleep(time.Millisecond * 100)

	sp.DeAssertChipSelect()

	ra.Reset()

	return nil
}

// Reset performs a SW-based reset of the RA8875
func (ra *SoftRAIO8875) Reset() error {
	err := ra.writeCommand(PWRR)
	if err != nil {
		log.Println("RA8875: Failed to write PWRR.")
		return err
	}
	ra.writeData(PWRR_SOFTRESET)
	if err != nil {
		log.Println("RA8875: Failed to write PWRR_SOFTRESET.")
		return err
	}
	ra.writeData(PWRR_NORMAL)
	if err != nil {
		log.Println("RA8875: Failed to write PWRR_NORMAL.")
		return err
	}
	time.Sleep(time.Millisecond)
	return nil
}

// Initialise the PLL
func (ra *SoftRAIO8875) pLLinit() {
	if ra.dimensions == devices.D480x272 {
		ra.writeReg(PLLC1, PLLC1_PLLDIV1+10)
		time.Sleep(time.Millisecond)
		ra.writeReg(PLLC2, PLLC2_DIV4)
		time.Sleep(time.Millisecond)
	} else /* ra.dimensions == D800x480)*/ {
		ra.writeReg(PLLC1, PLLC1_PLLDIV1+10)
		time.Sleep(time.Millisecond)
		ra.writeReg(PLLC2, PLLC2_DIV4)
		time.Sleep(time.Millisecond)
	}
}

func (ra *SoftRAIO8875) initDriver() {
	log.Println("RA8875: init PLL")
	ra.pLLinit()

	ra.writeReg(SYSR, SYSR_16BPP|SYSR_MCU8)

	/* Timing values */
	var pixclk uint8
	var hsync_start uint8
	var hsync_pw uint8
	var hsync_finetune uint8
	var hsync_nondisp uint8
	var vsync_pw uint8
	var vsync_nondisp uint16
	var vsync_start uint16

	/* Set the correct values for the display being used */
	if ra.dimensions == devices.D480x272 {
		pixclk = PCSR_PDATL | PCSR_4CLK
		hsync_nondisp = 10
		hsync_start = 8
		hsync_pw = 48
		hsync_finetune = 0
		vsync_nondisp = 3
		vsync_start = 8
		vsync_pw = 10
	} else /* ra.dimensions == D800x480)*/ {
		log.Println("RA8875: initDriver: for 800x480")
		pixclk = PCSR_PDATL | PCSR_2CLK
		hsync_nondisp = 26
		hsync_start = 32
		hsync_pw = 96
		hsync_finetune = 0
		vsync_nondisp = 32
		vsync_start = 23
		vsync_pw = 2
	}

	log.Println("RA8875: initDriver: setting pixclk")
	ra.writeReg(PCSR, pixclk)
	time.Sleep(time.Millisecond)

	log.Println("RA8875: initDriver: setting dimensions")
	if ra.dimensions == devices.D480x272 {
		ra.Width = 480
		ra.Height = 272
	} else {
		ra.Width = 800
		ra.Height = 480
	}

	/* Horizontal settings registers */
	log.Println("RA8875: initDriver: setting Horizontal")
	ra.writeReg(HDWR, uint8(ra.Width/8)-1) // H width: (HDWR + 1) * 8 = 480
	ra.writeReg(HNDFTR, HNDFTR_DE_HIGH+hsync_finetune)
	ra.writeReg(HNDR, (hsync_nondisp-hsync_finetune-2)/8) // H non-display: HNDR * 8 + HNDFTR + 2 = 10
	ra.writeReg(HSTR, hsync_start/8-1)                    // Hsync start: (HSTR + 1)*8
	ra.writeReg(HPWR, HPWR_LOW+(hsync_pw/8-1))            // HSync pulse width = (HPWR+1) * 8

	/* Vertical settings registers */
	log.Println("RA8875: initDriver: setting Vertical")
	ra.writeReg(VDHR0, uint8(ra.Height-1)&0xFF)
	ra.writeReg(VDHR1, uint8((ra.Height-1)>>8))
	ra.writeReg(VNDR0, uint8(vsync_nondisp-1)) // V non-display period = VNDR + 1
	ra.writeReg(VNDR1, uint8(vsync_nondisp>>8))
	ra.writeReg(VSTR0, uint8(vsync_start-1)) // Vsync start position = VSTR + 1
	ra.writeReg(VSTR1, uint8(vsync_start>>8))
	ra.writeReg(VPWR, VPWR_LOW+vsync_pw-1) // Vsync pulse width = VPWR + 1

	/* Set active window X */
	log.Println("RA8875: initDriver: active window X")
	ra.writeReg(HSAW0, 0) // horizontal start point
	ra.writeReg(HSAW1, 0)
	ra.writeReg(HEAW0, uint8((ra.Width-1)&0xFF)) // horizontal end point
	ra.writeReg(HEAW1, uint8((ra.Width-1)>>8))

	/* Set active window Y */
	log.Println("RA8875: initDriver: active window Y")
	ra.writeReg(VSAW0, 0) // vertical start point
	ra.writeReg(VSAW1, 0)
	ra.writeReg(VEAW0, uint8((ra.Height-1)&0xFF)) // horizontal end point
	ra.writeReg(VEAW1, uint8((ra.Height-1)>>8))

	/* ToDo: Setup touch panel? */

	/* Clear the entire window */
	log.Println("RA8875: initDriver: clear window")
	ra.writeReg(MCLR, MCLR_START|MCLR_FULL)
	time.Sleep(time.Millisecond * 500)
}

// ----------------------------------------------------------
// Wrappers: Low level
// ----------------------------------------------------------

func (ra *SoftRAIO8875) writeReg(reg, val uint8) {
	// fmt.Printf("writeReg: command: %x\n", reg)
	ra.writeCommand(reg)
	// fmt.Printf("writeData: val: %x\n", val)
	ra.writeData(val)
}

func (ra *SoftRAIO8875) readReg(reg uint8) (uint8, error) {
	ra.writeCommand(reg)
	return ra.readData()
}

// WriteData writes data to the device via SPI
func (ra *SoftRAIO8875) writeData(data byte) error {
	sp := ra.spi

	fmt.Printf("writeData: DATAWRITE: %x\n", DATAWRITE)
	if !sp.ConstantCSAssert {
		sp.AssertChipSelect()
	}

	_, err := sp.Transfer(DATAWRITE)

	if err != nil {
		return err
	}

	fmt.Printf("writeData: data: %x\n", data)
	_, err = sp.Transfer(data)

	if !sp.ConstantCSAssert {
		sp.DeAssertChipSelect()
	}

	if err != nil {
		return err
	}

	return nil
}

func (ra *SoftRAIO8875) readData() (uint8, error) {
	sp := ra.spi
	if !sp.ConstantCSAssert {
		sp.AssertChipSelect()
	}

	_, err := sp.Transfer(DATAREAD)

	x, err := sp.Transfer(0)

	if !sp.ConstantCSAssert {
		sp.DeAssertChipSelect()
	}

	if err != nil {
		return 0, err
	}

	return uint8(x), err
}

// WriteCommand writes a command via SPI protocol
func (ra *SoftRAIO8875) writeCommand(command byte) error {
	sp := ra.spi
	if !sp.ConstantCSAssert {
		sp.AssertChipSelect()
	}

	// ra.DebugTrigPulse()
	// ra.DebugTrigPulse()
	// ra.DebugTrigPulse()

	fmt.Printf("writeCommand: CMDWRITE: %x\n", CMDWRITE)
	_, err := sp.Transfer(CMDWRITE)
	if err != nil {
		log.Println(err)
	}
	// ra.DebugTrigPulse()

	fmt.Printf("writeCommand: command: %x\n", command)
	_, err = sp.Transfer(command)
	// ra.DebugTrigPulse()
	// ra.DebugTrigPulse()

	if !sp.ConstantCSAssert {
		sp.DeAssertChipSelect()
	}

	return nil
}
