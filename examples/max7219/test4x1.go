package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/wdevore/hardware/spi"
)

// -----------------------------------------------------------
// Display buffer examples for a cascade of 4 8x8 displays.
// You can buy these cascades as a single unit.
// Note: this strategy of using a buffer "hard codes" the display
// setup. Thus using the shifting features of the chained
// displays is "lost".
// -----------------------------------------------------------
var dataPacket = []byte{
	// First byte is col being written, second is row pattern
	0, 0x00000000,
	0, 0x00000000,
	0, 0x00000000,
	0, 0x00000000,
}

// A buffered shaped as: 32x8 = Row x Col
//               Row(ry)    Col(cx)
//                |           |
//                |     /-----/
//                v     v
var pixelBuf4 = [8 * 4][8]byte{}

// To initialize all 4 displays we need to:
// 1) drop CS low
// 2) write a command/data sequence 4 times
// 3) raise CS high

func initialize4(sp *spi.FtdiSPI) {
	var err error
	sp.TakeControlOfCS()

	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		err = sp.Write([]byte{modeReg, noDecode})
		if err != nil {
			panic(err)
		}
	}
	sp.DeAssertChipSelect()

	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		err = sp.Write([]byte{intensityReg, 0x01})
		if err != nil {
			log.Fatal(err)
		}
	}
	sp.DeAssertChipSelect()

	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		err = sp.Write([]byte{scanLimitReg, allColumns})
		if err != nil {
			log.Fatal(err)
		}
	}
	sp.DeAssertChipSelect()

	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		err = sp.Write([]byte{shutdownReg, normal}) // Normal operation
		if err != nil {
			log.Fatal(err)
		}
	}
	sp.DeAssertChipSelect()

	sp.ReleaseControlOfCS()
}

func clearRegisters4(sp *spi.FtdiSPI) {
	sp.TakeControlOfCS()

	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		sp.Write([]byte{digit0Reg, zero})
	}
	sp.DeAssertChipSelect()

	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		sp.Write([]byte{digit1Reg, zero})
	}
	sp.DeAssertChipSelect()
	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		sp.Write([]byte{digit2Reg, zero})
	}
	sp.DeAssertChipSelect()
	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		sp.Write([]byte{digit3Reg, zero})
	}
	sp.DeAssertChipSelect()
	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		sp.Write([]byte{digit4Reg, zero})
	}
	sp.DeAssertChipSelect()
	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		sp.Write([]byte{digit5Reg, zero})
	}
	sp.DeAssertChipSelect()
	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		sp.Write([]byte{digit6Reg, zero})
	}
	sp.DeAssertChipSelect()
	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {
		sp.Write([]byte{digit7Reg, zero})
	}
	sp.DeAssertChipSelect()

	sp.ReleaseControlOfCS()
}

func displayOn4(sp *spi.FtdiSPI, on uint8) {
	packetOff := []byte{displayTestReg, off}
	packetOn := []byte{displayTestReg, on}

	var err error
	sp.TakeControlOfCS()

	sp.AssertChipSelect()
	for i := 0; i < 4; i++ {

		if on == 1 {
			err = sp.Write(packetOn)
		} else {
			err = sp.Write(packetOff)
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	sp.DeAssertChipSelect()

	sp.ReleaseControlOfCS()
}

// x = col, y = row
func setPixel4(cx, ry int) {
	pixelBuf4[ry][cx] = 1
}

func unSetPixel4(cx, ry int) {
	pixelBuf4[ry][cx] = 0
}

func getPixel4(cx, ry int) byte {
	return pixelBuf4[ry][cx]
}

func clearDisplay4(sp *spi.FtdiSPI) {
	// For each 8x8 display we send 4 16bit blocks. Each block belongs
	// to one display.
	// As always the column index starts at one based on the specsheet.
	for col := byte(1); col < 9; col++ {
		dataPacket[0] = col
		dataPacket[2] = col
		dataPacket[4] = col
		dataPacket[6] = col
		sp.Write(dataPacket)
	}
}

func clearBuf4() {
	for ry := 0; ry < 8*4; ry++ {
		for cx := 0; cx < 8; cx++ {
			pixelBuf4[ry][cx] = 0
		}
	}
}

func printBuf4() {
	for ry := 0; ry < 8*4; ry++ {
		s := ""
		for cx := 0; cx < 8; cx++ {
			// fmt.Printf("ry: %d, cx: %d\n", ry, cx)
			if pixelBuf4[ry][cx] == 1 {
				s += "1"
			} else {
				s += "0"
			}
		}
		println(s)
	}
}

func clear4(sp *spi.FtdiSPI) {
	clearBuf4()

	// blit
	blit4(sp)
}

func blit4(sp *spi.FtdiSPI) {
	var b byte

	// We need to do something similar to clearDisplay4()
	cx := 7
	for col := byte(0); col < 8; col++ {
		dataPacket[0] = col + 1
		dataPacket[2] = col + 1
		dataPacket[4] = col + 1
		dataPacket[6] = col + 1

		// --------------------------
		// Top display
		// --------------------------
		// Now set the row data for the farthest display
		// The left most bit (0) is the top
		// The right most bit (7) is the bottom
		b = pixelBuf4[7][cx]
		b |= pixelBuf4[6][cx] << 1
		b |= pixelBuf4[5][cx] << 2
		b |= pixelBuf4[4][cx] << 3
		b |= pixelBuf4[3][cx] << 4
		b |= pixelBuf4[2][cx] << 5
		b |= pixelBuf4[1][cx] << 6
		b |= pixelBuf4[0][cx] << 7
		// fmt.Printf("B: %08b\n", b)
		dataPacket[1] = b // Nearest to *Output* pins

		b = pixelBuf4[15][cx]
		b |= pixelBuf4[14][cx] << 1
		b |= pixelBuf4[13][cx] << 2
		b |= pixelBuf4[12][cx] << 3
		b |= pixelBuf4[11][cx] << 4
		b |= pixelBuf4[10][cx] << 5
		b |= pixelBuf4[9][cx] << 6
		b |= pixelBuf4[8][cx] << 7
		dataPacket[3] = b

		b = pixelBuf4[23][cx]
		b |= pixelBuf4[22][cx] << 1
		b |= pixelBuf4[21][cx] << 2
		b |= pixelBuf4[20][cx] << 3
		b |= pixelBuf4[19][cx] << 4
		b |= pixelBuf4[18][cx] << 5
		b |= pixelBuf4[17][cx] << 6
		b |= pixelBuf4[16][cx] << 7
		dataPacket[5] = b

		// --------------------------
		// Bottom display
		// --------------------------
		b = pixelBuf4[31][cx]
		b |= pixelBuf4[30][cx] << 1
		b |= pixelBuf4[29][cx] << 2
		b |= pixelBuf4[28][cx] << 3
		b |= pixelBuf4[27][cx] << 4
		b |= pixelBuf4[26][cx] << 5
		b |= pixelBuf4[25][cx] << 6
		b |= pixelBuf4[24][cx] << 7
		dataPacket[7] = b

		//             top                                 bottom
		// packet = {col,pattern0,col,pattern1,col,pattern2,col,pattern3}
		sp.Write(dataPacket)

		cx--
	}
}

func setPixelPattern4(sp *spi.FtdiSPI) {

	for i := 0; i < 1; i++ {
		clearDisplay4(sp)
		clearBuf4()
		// top-left = top most display,top-most row
		setPixel4(7, 0)
		setPixel4(7, 1)
		setPixel4(7, 2)
		setPixel4(1, 0)
		setPixel4(0, 0)

		setPixel4(6, 3)
		setPixel4(6, 4)
		setPixel4(6, 5)
		setPixel4(6, 6)
		setPixel4(6, 7)

		setPixel4(5, 0)
		setPixel4(5, 2)

		// Set a few pixels on the 2nd from top display
		setPixel4(7, 8)
		setPixel4(0, 8)

		// Set a few pixels on the 3rd from top display
		setPixel4(2, 17)
		setPixel4(5, 17)

		// Set a few pixels on the bottom display
		setPixel4(4, 29)
		setPixel4(7, 29)

		// printBuf4()
		blit4(sp)
		time.Sleep(time.Millisecond * 10)
	}
}

func simpleSetPixels4(sp *spi.FtdiSPI) {

	for i := 0; i < 100000; i++ {
		clearDisplay4(sp)

		packet := []byte{
			// The right most bit is the bottom
			// The left most bit is the top
			1, stringToByte("11100000"), // Nearest to *Output* pins
			1, stringToByte("00000010"),
			1, stringToByte("00100100"),
			1, stringToByte("00001000"), // Nearest to *Input* pins
		}
		sp.Write(packet)

		time.Sleep(time.Millisecond * 10)
	}
}

func matrix4(sp *spi.FtdiSPI, loop int, scrollDirection bool, scrollSpeed int) {
	ran := rand.New(rand.NewSource(99))
	clearBuf4()

	for i := 0; i < loop; i++ {
		if quit {
			break
		}

		// Set a random pixel on row 0
		rcol := int(ran.Float32() * 8)

		if scrollDirection {
			setPixel4(rcol, 0)
		} else {
			setPixel4(rcol, 31)
		}

		blit4(sp)
		time.Sleep(time.Millisecond * time.Duration(scrollSpeed))

		if scrollDirection {
			// scroll downwards
			for ry := 8*4 - 1; ry > 0; ry-- {
				for cx := 0; cx < 8; cx++ {
					// copy row-1 into row
					pixelBuf4[ry][cx] = pixelBuf4[ry-1][cx]
				}
			}
		} else {
			// scroll up
			for ry := 0; ry < 8*4-1; ry++ {
				for cx := 0; cx < 8; cx++ {
					// copy row+1 into row
					pixelBuf4[ry][cx] = pixelBuf4[ry+1][cx]
				}
			}
		}

		// clear row 0
		if scrollDirection {
			unSetPixel4(rcol, 0)
		} else {
			unSetPixel4(rcol, 31)
		}
	}
}

func random4(sp *spi.FtdiSPI, loop int, pause int) {
	ran := rand.New(rand.NewSource(99))

	for i := 0; i < loop; i++ {
		if quit {
			break
		}

		clearBuf4()

		// Fill buff with random pattern
		for ry := 0; ry < 8*4; ry++ {
			for cx := 0; cx < 8; cx++ {
				if ran.Float32() > 0.6 {
					setPixel4(cx, ry)
				}
			}
		}

		// blit
		blit4(sp)

		// pause
		time.Sleep(time.Millisecond * time.Duration(pause))
	}

	log.Println("Done.")
}

func flipflop4(sp *spi.FtdiSPI) {
	ran := rand.New(rand.NewSource(99))
	flip := false

	for j := 0; j < 10000; j++ {
		if quit {
			break
		}

		scrollSpeed := int(ran.Float32()*50 + 5)
		pause := int(ran.Float32()*200 + 50)

		matrix4(sp, 100, flip, scrollSpeed)

		random4(sp, 20, pause)

		flip = !flip
	}
}
