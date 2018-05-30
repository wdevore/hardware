package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ziutek/ftdi"
)

var quit = false

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("\nReceived ctrl-C, quiting.")
		quit = true
	}()

	d, err := ftdi.OpenFirst(0x0403, 0x6014, ftdi.ChannelA)
	if err != nil {
		log.Fatal(err)
	}

	defer d.Close()

	d.SetBitmode(0xff, ftdi.ModeBitbang)
	if err != nil {
		log.Fatal(err)
	}

	d.SetBaudrate(10000)
	if err != nil {
		log.Fatal(err)
	}

	for !quit {
		d.WriteByte(0x00 | 0x00)
		d.WriteByte(0x04 | 0x08)
		// d.WriteByte(byte(0x02))
		// d.WriteByte(byte(0x04))
	}

	// for !quit {
	// 	// log.Print("WriteByte")
	// 	for i := 0; i < 256; i++ {
	// 		d.WriteByte(byte(i))
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 	}
	// }

	// log.Print("Ok")
	// time.Sleep(time.Second)

	// buf := make([]byte, 256)
	// for i := range buf {
	// 	buf[i] = byte(i)
	// }

	// log.Print("Write")
	// _, err = d.Write(buf)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Println("Ok")

	// d.WriteByte(255)
	// if err != nil {
	// 	log.Fatal(err)
	// }

}

func exitProg(d *ftdi.Device) {
	log.Println("Closing devices")
	err := d.Close()
	if err != nil {
		log.Println("\n Failed to close FTDI component")
		os.Exit(-1)
	}
	os.Exit(0)
}
