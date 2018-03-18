package main

import (
	"log"

	"github.com/ziutek/ftdi"
)

// You can find the vender and product using:
// >lsusb
var (
	vender  = 0x0403
	product = 0x6014
)

// Mode is in or out
type Mode int

// State is io value (true or false)
type State bool

const (
	gpioOut    Mode  = 0
	gpioIn     Mode  = 1
	outputHigh State = true
	outputLow  State = false
)

func main() {
	d, err := ftdi.OpenFirst(vender, product, ftdi.ChannelAny)
	if err != nil {
		log.Fatal(err)
	}

	defer d.Close()

	log.Print("Enabling MPSSE mode")
	d.SetBitmode(0xff, ftdi.ModeMPSSE)
	if err != nil {
		log.Fatal(err)
	}

	// checkErr(d.SetBaudrate(192))

	// log.Print("WriteByte")
	// for i := 0; i < 256; i++ {
	// 	checkErr(d.WriteByte(byte(i)))
	// }

	// log.Print("Ok")
	// time.Sleep(time.Second)

	// buf := make([]byte, 256)
	// for i := range buf {
	// 	buf[i] = byte(i)
	// }

	// log.Print("Write")
	// _, err = d.Write(buf)
	// checkErr(err)

	// log.Println("Ok")

	// checkErr(d.WriteByte(255))

}
