package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wdevore/hardware/ftdi"
	"github.com/wdevore/hardware/gpio"
	"github.com/wdevore/hardware/spi"
)

func main() {
	log.Println("Creating SPI device")
	spid := spi.NewSPI(0x0403, 0x06014, false)

	// Create a SPI interface from the FT232H using pin 8 (C0) as chip select.
	// Use a clock speed of 5mhz, SPI mode 0, and most significant bit first.
	// spid.Configure(ftdi.C0, 5000000, ftdi.Mode0, ftdi.MSBFirst)

	spid.Configure(ftdi.D3, 100000, spi.Mode0, spi.MSBFirst)

	fi := spid.GetFTDI()
	fi.SetConfigPin(ftdi.D5, gpio.Output) //DC
	fi.SetConfigPin(ftdi.D4, gpio.Output) // Reset

	fi.OutputHigh(ftdi.D5)
	fi.OutputHigh(ftdi.D4)

	// spid.ConfigPin(ftdi.D6, gpio.Output) // ST7735R backlight pin
	// spid.OutputLow(ftdi.D6)              // Turn off light

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(ft *spi.FtdiSPI) {
		<-c
		println("\nReceived ctrl-C, closing FTDI interface.")
		err := ft.Close()
		if err != nil {
			println("\n Failed to close FTDI interface")
			os.Exit(-1)
		}
		os.Exit(0)
	}(spid)

	defer spid.Close()
	const (
		MADCTL = 0x36
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

	// # Write three bytes (0x01, 0x02, 0x03) out using the SPI Hardware feature of the FT232H.
	log.Println("Writing bytes")
	data1 := []byte{MADCTL}                          // Command
	data2 := []byte{MadctlMX | MadctlMY | MadctlBGR} // Data
	// data3 := []byte{0xff}

	// Write command
	fi.OutputLow(ftdi.D5) // Low = command
	// spid.OutputLow(ftdi.D3) // chip select

	log.Println("WriteCommand: writing byte command")
	err := spid.Write(data1)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("WriteCommand: toggling cs")
	// spid.OutputHigh(ftdi.D3)

	// Write data
	fi.OutputHigh(ftdi.D5) // High = data
	// spid.OutputLow(ftdi.D3)

	err = spid.Write(data2)
	if err != nil {
		log.Fatal(err)
	}

	// spid.OutputHigh(ftdi.D3)

	for i := 0; i < 1000; i++ {
		// spid.OutputHigh(ftdi.D4)
		// log.Println("Writing")
		// spid.Write(data1)
		// time.Sleep(time.Millisecond * 10)

		fi.OutputLow(ftdi.D5)
		err := spid.Write(data1)
		if err != nil {
			log.Printf("Write failed: %v\n", err)
			break
		}
		fi.OutputHigh(ftdi.D5)
		err = spid.Write(data2)
		if err != nil {
			log.Printf("Write failed: %v\n", err)
			break
		}
		// _, err = fi.Write(data3)
		// if err != nil {
		// 	log.Printf("Write failed: %v\n", err)
		// 	break
		// }
		// spid.OutputLow(ftdi.D4)
		time.Sleep(time.Millisecond * 10)
	}

	log.Println("Done.")
}
