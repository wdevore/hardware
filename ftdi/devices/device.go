package devices

import (
	"math"
)

/*
	The "tab"s issue:
	The TFT displays are shipped with protective films that also include a
	small tab sticker "hanging" off the film overlay. This sticker tab has a
	color: red or green.
	The Rcmd(x) commands are for specific "tab" colors. Hopefully, you didn't just
	rip it off and toss it before noting what color it is, otherwise you will have
	to guess until it works.
*/

// TabColor what color  tab sticker you "received" on your TFT device
type TabColor int

const (
	// GreenTab indicates that you "received" a Green tab sticker on it
	GreenTab TabColor = iota
	// RedTab indicates that you "received" Green tab sticker on it
	RedTab
)

/*
	Dimensions:
	The 7735 has several different width/height physical layouts:
	- 128x128
	- 128x160
	- 160x80

*/
type Dimensions int

const (
	// D128x128 = 128 by 128 pixels
	D128x128 = 0
	// D128x160 = 128 by 160 pixels
	D128x160 = 1
	// D160x80 = 160 by 80 pixels
	D160x80 = 2
)

const (
	// Min450Hz is the minimum clock speed of the FTDI232
	Min450Hz = 450
	// Max30MHz is the  maximum clock speed
	Max30MHz = 30000000
)

var (
	// Basic Color definitions
	//   R      G     B  = 5,6,5
	// ----- ------ -----

	BLACK uint16 = 0x0000
	WHITE uint16 = 0xFFFF

	// 0000 0000 0001 1111   == 001f
	BLUE  uint16 = 0x001F
	RED   uint16 = 0xF800
	GREEN uint16 = 0x07E0

	CYAN      = RGBtoRGB565(0, 255, 255) //0x07FF
	MAGENTA   = RGBtoRGB565(255, 0, 255) //0xF81F
	YELLOW    = RGBtoRGB565(255, 255, 0) //0xFFE0
	ORANGE    = RGBtoRGB565(255, 127, 0)
	GREY      = RGBtoRGB565(127, 127, 127)
	LightGREY = RGBtoRGB565(200, 200, 200)
)

const (
	// DELAY is for embedding standard delays during commands.
	DELAY   = 0x80
	delayed = true
	noDelay = false
)

// ----------------------------------------------------
// Rotation
// ----------------------------------------------------

// RotationMode controls raster drawing
type RotationMode int

const (
	Orientation0 RotationMode = iota
	Orientation1
	Orientation2
	Orientation3
	OrientationDefault
)

// RGBtoRGB565 converts an 8-bit (each) R,G,B into a 16-bit packed color
// formatted as 5-6-5.
// Each 8bits are interpolated down to 5 or 6 bits.
// 2^5 = 32 shades of Red
// 2^6 = 64 shades of Green
// 2^5 = 32 shades of Blue
func RGBtoRGB565(r, g, b int) uint16 {
	// 16 bits:
	//   R      G     B
	// 00000 000000 00000

	// Interpolate 8-8-8 values to 5-6-5 values
	lr := lerp(31.0, 255.0, float64(r)) // 8bit to 5bit
	lg := lerp(63.0, 255.0, float64(g)) // 8bit to 6bit
	lb := lerp(31.0, 255.0, float64(b)) // 8bit to 5bit

	// Shift them into their new positions.
	var c = uint16(lr) << 11
	c |= uint16(lg) << 5
	c |= uint16(lb)

	return c
}

func lerp(x1, y1, y float64) float64 {
	x := x1 / y1 * y
	return math.Round(x)
}
