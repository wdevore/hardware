package devices

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
	D128x128 = iota
	// D128x160 = 128 by 160 pixels
	D128x160 // Typically an ST7735S device
	D160x128 // Typically an ST7735S device
	// D160x80 = 160 by 80 pixels
	D128x96
	D160x80

	D320x480 // 320w by 480h = portrait
	D480x320 // landscape

	D480x272
	D800x480
)

const (
	// Min450Hz is the minimum clock speed of the FTDI232
	Min450Hz = 450
	// Max30MHz is the  maximum clock speed
	Max30MHz = 30000000
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

// ColorOrder :some of the devices have the SRGB pin tied either high/true (BGR) or low/false (RGB)
// If your colors are incorrect pass a different byteOrder
type ColorOrder int

const (
	RGBOrder = iota
	BGROrder
)
