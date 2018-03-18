package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wdevore/hardware/gpio"

	"github.com/wdevore/hardware/ftdi"
)

func main() {

	ft232h := ftdi.NewFTDI232H(0x0403, 0x06014)

	ft232h.Initialize(false)

	ft232h.Configure(false)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(ft *ftdi.FTDI232H) {
		<-c
		println("\nReceived ctrl-C, closing FTDI interface.")
		err := ft.Close()
		if err != nil {
			println("\n Failed to close FTDI interface")
			os.Exit(-1)
		}
		os.Exit(0)
	}(ft232h)

	defer ft232h.Close()

	if ft232h == nil {
		log.Fatal("Failed to create FTDI interface")
		os.Exit(-2)
	}

	log.Print("FTDI interface created and initialized")

	// Configure digital inputs and outputs using the setup function.
	// Note that pin numbers 0 to 15 map to pins D0 to D7 then C0 to C7 on the board.
	ft232h.ConfigPin(ftdi.D7, gpio.Input) // Make pin D7 a digital input.
	// log.Print(ft232h)

	ft232h.ConfigPin(ftdi.C0, gpio.Output) // Make pin C0 a digital output.
	// log.Print(ft232h)

	log.Println("IO pins configured")
	log.Println("Press Ctrl-c to quit")

	prevLevel := gpio.Low

	log.Printf("Pin C0 is: %d\n", ftdi.C0)
	log.Printf("Pin D7 is: %d\n", ftdi.D7)

	for i := 0; i < 10000; i++ {

		// Set pin C0 to a high level so the LED turns on.
		err := ft232h.OutputHigh(ftdi.C0)
		if err != nil {
			log.Printf("Error writing to Pin: %v\n", err)
		}
		// log.Print(ft232h)
		time.Sleep(time.Millisecond * 20)

		// Set pin C0 to a low level so the LED turns off.
		err = ft232h.OutputLow(ftdi.C0)
		if err != nil {
			log.Printf("Error writing to Pin: %v\n", err)
		}

		// log.Print(ft232h)
		time.Sleep(time.Millisecond * 200)

		// Read the input on pin D7 and print out if it's high or low.
		// Note: reading the pins takes upwards of 55ms so the low-time
		// will be longer, for example:
		//
		//      ----               ----
		// ____|    |_____________|    |________________
		//  L    H     read+L       H     read+L
		//
		// So we skip the second sleep above.

		level := ft232h.ReadInput(ftdi.D7)
		if level != prevLevel {
			if level == gpio.Low {
				println("Pin D7 is LOW!")
				print(ft232h.ToStringFullBinary())
			} else {
				println("Pin D7 is HIGH!")
				print(ft232h.ToStringFullBinary())
			}
		}
		prevLevel = level
	}
}
