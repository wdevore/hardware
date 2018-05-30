package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wdevore/hardware/gpio"

	"github.com/wdevore/hardware/spi"
)

// Run this like:
// go run *.go

// Main dupont ribbon color pin layout:
// Green = clk = D0
// Purple = cs = D3
// Blue = DIN = MOSI = D1
// Grey = Ground
// White = Vcc = ~3.3V. Note 5V cause glitches

var quit bool

func main() {
	log.Println("Creating SPI device")
	quit = false

	spid := spi.NewSPI(0x0403, 0x06014, false)

	spid.EnableTrigger()

	// Create a SPI interface from the FT232H using pin D3 as chip select.
	// Use a clock speed of 100KHz, SPI mode 0, and most significant bit first.
	spid.Configure(gpio.DefaultPin, 2000000, spi.Mode0, spi.MSBFirst)
	// Max requires an active CS so we disable constant assert so CS will toggle.
	spid.ConstantCSAssert = false

	spid.TriggerPulse()

	spid.DeAssertChipSelect()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(ft *spi.FtdiSPI) {
		<-c
		quit = true
		println("\nReceived ctrl-C, closing FTDI interface.")
		exitProg(ft)
	}(spid)

	defer exitProg(spid)

	initializeN(spid, 16)

	// displayOnN(spid, 1, 16)
	// time.Sleep(time.Millisecond * 150)
	// displayOnN(spid, 0, 16)

	// setCornerPixelsN(spid)
	setPixelPatternN(spid)
	// setFloodFill(spid, 0)
	// displayBlinkN(spid)

	// superSimple(spid)
	// enableCol2WithPattern(spid)
	// random3(spid)
	// horizontalScanBar(spid)
	// dualScanBars2(spid)
	// setPixelTest(spid)
	// matrix(spid)

	// 4x1 Cascaded displays
	// displayOn4(spid, 1)
	// time.Sleep(time.Millisecond * 250)
	// displayOn4(spid, 0)

	// clear4(spid)

	// setPixelPattern4(spid)
	// simpleSetPixels4(spid)
	// matrix4(spid, 1000, false, 20)
	// random4(spid, 1000, 200)
	// flipflop4(spid)
}

func exitProg(sp *spi.FtdiSPI) {
	log.Println("Closing devices")
	clearRegistersN(sp, 16)

	err := sp.Close()
	if err != nil {
		log.Println("\n Failed to close FTDI component")
		os.Exit(-1)
	}
	os.Exit(0)
}

const (
	zero byte = 0x00
)

// Registers
const (
	noOpReg byte = zero // No-Op

	// For this code the digits actually represent which column is enabled
	// You OR them to enable multple columns.
	digit0Reg byte = 0x01
	digit1Reg byte = 0x02 // For example this would enable column 2 only
	digit2Reg byte = 0x03
	digit3Reg byte = 0x04
	digit4Reg byte = 0x05
	digit5Reg byte = 0x06
	digit6Reg byte = 0x07
	digit7Reg byte = 0x08

	modeReg        byte = 0x09 // Decode Mode
	intensityReg   byte = 0x0a
	scanLimitReg   byte = 0x0b
	shutdownReg    byte = 0x0c
	displayTestReg byte = 0x0f
)

// Decode modes
const (
	noDecode           = zero
	codeBDigit0   byte = 0x01
	codeBDigit3_0 byte = 0x0f
	codeBDigit7_0 byte = 0xff
)

// Shutdown modes
const (
	shutdown byte = zero
	normal   byte = 0x01
)

// DisplayTest modes
const (
	off byte = zero
	on  byte = 0x01
)

// Intensities (a few predefined) values can range from 0 to 0x0F
const (
	iMin byte = zero
	iMed byte = 0x07
	iMax byte = 0x0f
)

// Scan limits indicates how many digits are displayed or columns enabled.
const (
	// I use the word "column" instead of digit
	noColumns  byte = zero
	allColumns byte = 0x07
)

// Each 8x8 block needs to be initialized.
// Which means we need to use the Noop
func initialize2(sp *spi.FtdiSPI) {
	var err error

	err = sp.Write([]byte{modeReg})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{noDecode})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{intensityReg})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{0x01})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{scanLimitReg})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{allColumns})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{shutdownReg}) // Normal operation
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{normal}) // Normal operation
	if err != nil {
		log.Fatal(err)
	}
}

func initialize(sp *spi.FtdiSPI) {
	var err error

	err = sp.Write([]byte{modeReg, noDecode})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{intensityReg, 0x01})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{scanLimitReg, allColumns})
	if err != nil {
		log.Fatal(err)
	}

	err = sp.Write([]byte{shutdownReg, normal}) // Normal operation
	if err != nil {
		log.Fatal(err)
	}
}

func clearRegisters(sp *spi.FtdiSPI) {
	sp.Write([]byte{digit0Reg, zero})
	sp.Write([]byte{digit1Reg, zero})
	sp.Write([]byte{digit2Reg, zero})
	sp.Write([]byte{digit3Reg, zero})
	sp.Write([]byte{digit4Reg, zero})
	sp.Write([]byte{digit5Reg, zero})
	sp.Write([]byte{digit6Reg, zero})
	sp.Write([]byte{digit7Reg, zero})
}

func stringToByte(bs string) byte {
	var b byte

	// This is a "reverse" interation because the LSb is at the end of the string.
	for i := range bs {
		c := bs[len(bs)-i-1]
		if c == '1' {
			b |= 1 << uint(i)
		}
		i++
	}

	return b
}
