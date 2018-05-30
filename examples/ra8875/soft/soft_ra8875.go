package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wdevore/hardware/ftdi/devices"
	"github.com/wdevore/hardware/ftdi/devices/ra8875"
)

var quit = false
var keyPressed rune

func main() {
	log.Println("Starting...")

	// Per Adafruit's FT232H breakout board configured as Hardware SPI:
	// https://learn.adafruit.com/adafruit-ft232h-breakout?view=all

	ra := ra8875.NewSoftRA8875Default(devices.D800x480)

	// Note this doesn't work when termbox-go is used
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func(ra ra8875.RA8875) {
		<-c
		log.Println("\nReceived ctrl-C, quiting.")
		ra.Quit()
		exitProg(ra)
	}(ra)

	if ra == nil {
		panic("Unable to create RA8875 component")
	}

	defer ra.Close()

	// log.Printf("Display dimensions: %d x %d\n", ra.Width, ra.Height)
	log.Println("Beginning test.")

	helloWorld(ra)

	log.Println("Press 'Enter' to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func exitProg(ra ra8875.RA8875) {
	log.Println("Closing devices")
	err := ra.Close()
	if err != nil {
		log.Println("\n Failed to close FTDI component")
		os.Exit(-1)
	}
	os.Exit(0)
}

func helloWorld(ra ra8875.RA8875) {
	// var err error

	fmt.Println("Turning on display")
	ra.DisplayOn(true)
	// ra.DebugTrigPulse()
	// ra.DebugTrigPulse()

	// fmt.Println("GPIOX to true")
	// // ra.DebugTrigPulse()
	// // ra.DebugTrigPulse()
	// // ra.DebugTrigPulse()
	// // ra.DebugTrigPulse()
	ra.GPIOX(true) // Enable TFT - display enable tied to GPIOX

	// fmt.Println("PWM1config to PWM_CLK_DIV1024")
	ra.PWM1config(true, ra8875.PWM_CLK_DIV1024) // PWM output for backlight

	// fmt.Println("PWM1config 255")
	ra.PWM1out(255)

	// fmt.Println("Fillscreen to RED")
	// // ra.DebugTrigPulse()

	ra.FillScreen(devices.RED)

	/* Switch to text mode */
	// fmt.Println("switching to text mode")
	// err = ra.TextMode()
	// if err != nil {
	// 	log.Println(err)
	// 	exitProg(ra)
	// }
	// // /* Set the cursor location (in pixels) */
	// fmt.Println("Set cursor pos")
	// ra.TextSetCursor(10, 10)

	// fmt.Println("Set text color")
	// ra.TextColor(devices.ORANGE, devices.GREY)

	// fmt.Println("writing text")
	// ra.TextWrite("Hello World!")
}
