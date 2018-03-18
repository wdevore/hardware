package main

import (
	"log"
	"time"

	"github.com/ziutek/ftdi"
)

func main() {
	d, err := ftdi.OpenFirst(0x0403, 0x6014, ftdi.ChannelAny)
	if err != nil {
		log.Fatal(err)
	}

	defer d.Close()

	d.SetBitmode(0xff, ftdi.ModeBitbang)
	if err != nil {
		log.Fatal(err)
	}

	d.SetBaudrate(192)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("WriteByte")
	for i := 0; i < 256; i++ {
		d.WriteByte(byte(i))
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Print("Ok")
	time.Sleep(time.Second)

	buf := make([]byte, 256)
	for i := range buf {
		buf[i] = byte(i)
	}

	log.Print("Write")
	_, err = d.Write(buf)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Ok")

	d.WriteByte(255)
	if err != nil {
		log.Fatal(err)
	}

}
