package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wdevore/hardware/spi"
)

func main() {
	basicTest()
}

func basicTest() {
	var err error

	log.Println("Creating Soft SPI device")
	soft := spi.NewSoftSPI(0x0403, 0x06014, false)

	defer soft.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(soft *spi.SoftSPI) {
		<-c
		println("\nReceived ctrl-C, closing FTDI interface.")
		err := soft.Close()
		if err != nil {
			println("\n Failed to close FTDI interface")
			os.Exit(-1)
		}
		os.Exit(0)
	}(soft)

	err = soft.Configure(100000, spi.MSBFirst)
	if err != nil {
		panic(err)
	}

	// fmt.Printf("%08b\n", 0x80)
	soft.Write(0x7b)
	fmt.Println("writing done")

	var pin6 byte
	pin6, err = soft.Read()
	fmt.Printf("pin6: %08b\n", pin6)
}
