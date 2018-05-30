package surface

import (
	"math"

	"github.com/wdevore/hardware/ftdi/devices"
)

var (
	// Basic Color definitions
	//   R      G     B  = 5,6,5
	// ----- ------ -----
	BLACK uint16 = 0
	WHITE uint16 = 0

	// 0000 0000 0001 1111   == 001f
	BLUE  uint16 = 0
	RED   uint16 = 0
	GREEN uint16 = 0

	CYAN       uint16 = 0
	MAGENTA    uint16 = 0
	YELLOW     uint16 = 0
	ORANGE     uint16 = 0
	GREY       uint16 = 0
	DarkerGREY uint16 = 0
	DarkGREY   uint16 = 0
	LightGREY  uint16 = 0
)

// Surface is a byte array for rendering onto.
type Surface struct {
	Width, Height int

	// Color is the current drawing color
	ColorH byte
	ColorL byte
	color  uint16

	ClearColorH byte
	ClearColorL byte

	// In general each word is an RGB (565) of 2 bytes each.
	// High byte followed by Low byte
	// HLHLHLHLHLHL...
	// pushBuffer is the "off screen" display buffer that is blitted in one Data
	// write call. This saves a huge amount of time.
	// With a clock of 30MHz (the max for the FTDI232H device) the buffer
	// can be blitted in about 10ms as compared to non-buffered in about 1300ms!
	// 10ms allows for a framerate between 30FPS(~33ms/frame) to 60FPS(~16ms/frame).
	// Although 60FPS only leaves about 6ms for your code which is pretty tight.
	buffer     []byte
	bufferSize int
}

// NewSurface creates a new surface
func NewSurface(width, height int, colorOrder devices.ColorOrder) *Surface {
	s := new(Surface)
	s.Width = width
	s.Height = height

	bufferSize := width * height

	// A buffer of bytes
	s.buffer = make([]byte, bufferSize*2)

	s.initColors(colorOrder)

	return s
}

// Buffer provides access to the buffer. Used for blitting.
func (s *Surface) Buffer() []byte {
	return s.buffer
}

func (s *Surface) initColors(colorOrder devices.ColorOrder) {
	// 0000 0000 0001 1111   == 001f
	BLACK = 0x0000
	WHITE = 0xFFFF
	BLUE = RGBto565(0, 0, 255, colorOrder)  //0x001F
	RED = RGBto565(255, 0, 0, colorOrder)   //0xF800
	GREEN = RGBto565(0, 255, 0, colorOrder) //0x07E0

	CYAN = RGBto565(0, 255, 255, colorOrder)    //0x07FF
	MAGENTA = RGBto565(255, 0, 255, colorOrder) //0xF81F
	YELLOW = RGBto565(255, 255, 0, colorOrder)  //0xFFE0
	ORANGE = RGBto565(255, 127, 0, colorOrder)
	GREY = RGBto565(127, 127, 127, colorOrder)
	DarkerGREY = RGBto565(8, 8, 8, colorOrder)
	DarkGREY = RGBto565(32, 32, 32, colorOrder)
	LightGREY = RGBto565(200, 200, 200, colorOrder)
}

// SetColor sets the current drawing color for all subsequent calls.
func (s *Surface) SetColor(color uint16) {
	// RGB565 (16bit per pixel color)
	s.color = color
	s.ColorH = byte(color >> 8 & 0xff)
	s.ColorL = byte(color & 0xff)
}

// SetClearColor sets the clear color
func (s *Surface) SetClearColor(color uint16) {
	// RGB565 (16bit per pixel color)
	s.ClearColorH = byte(color >> 8 & 0xff)
	s.ClearColorL = byte(color & 0xff)
}

// GetPixel returns color of pixel at x,y
func (s *Surface) GetPixel(x, y int) uint16 {
	bufOff := y*s.Width + x

	return uint16(s.buffer[bufOff*2]) | uint16(s.buffer[bufOff*2+1])
}

// SetPixel using current drawing color
func (s *Surface) SetPixel(x, y int) {
	if (x < 0) || (x > s.Width-1) || (y < 0) || (y > s.Height-1) {
		return
	}

	// Calculate memory location based on screen width and height
	bufOff := y*s.Width + x

	s.buffer[bufOff*2] = s.ColorH   // High
	s.buffer[bufOff*2+1] = s.ColorL // Low
}

// SetPixelWithColor draws a pixel
func (s *Surface) SetPixelWithColor(x, y int, color uint16) {
	if (x < 0) || (x > s.Width-1) || (y < 0) || (y > s.Height-1) {
		return
	}

	s.SetColor(color)

	// RGB565 (16bit per pixel color)
	// Calculate memory location based on screen width and height
	bufOff := y*s.Width + x

	s.buffer[bufOff*2] = s.ColorH   // High
	s.buffer[bufOff*2+1] = s.ColorL // Low
}

// DrawVLine draws a vertical line
func (s *Surface) DrawVLine(x, y, h int) {
	// TODO add cohen-sutherland clipping
	if (x < 0) || (x > s.Width) || (y < 0) || (y > s.Height) {
		return
	}

	if (y + h) > s.Height {
		return
	}

	for iy := y; iy < y+h; iy++ {
		bufOff := iy*s.Width + x

		s.buffer[bufOff*2] = s.ColorH   // High
		s.buffer[bufOff*2+1] = s.ColorL // Low
	}
}

// DrawHLine draws a horizontal line
func (s *Surface) DrawHLine(x, y, w int) {
	if (x < 0) || (x > s.Width) || (y < 0) || (y > s.Height) {
		return
	}

	if (x + w) > s.Width {
		return
	}

	for ix := x; ix < x+w; ix++ {
		bufOff := y*s.Width + ix

		s.buffer[bufOff*2] = s.ColorH   // High
		s.buffer[bufOff*2+1] = s.ColorL // Low
	}
}

// DrawFilledRectangle fills a rectangle
func (s *Surface) DrawFilledRectangle(x, y, w, h int) {
	if (x < 0) || (x > s.Width-1) || (y < 0) || (y > s.Height-1) {
		return
	}

	sw := x + w
	sh := y + h
	if (sw > s.Width) || (sh > s.Height) {
		return
	}

	j := 0

	// fillLoop:
	for sy := y; sy < sh; sy++ {
		for sx := x; sx < sw; sx++ {
			j = sy*s.Width + sx
			// if j > s.bufferSize {
			// 	break fillLoop
			// }
			s.buffer[j*2] = s.ColorH   // High
			s.buffer[j*2+1] = s.ColorL // Low
		}
	}
}

// Clear clears buffer to current clear color
func (s *Surface) Clear() {
	s.ColorH = s.ClearColorH
	s.ColorL = s.ClearColorL
	s.DrawFilledRectangle(0, 0, s.Width, s.Height)
}

// RGBto565 converts an 8-bit (each) R,G,B into a 16-bit packed color
// formatted as 5-6-5.
// Each 8bits are interpolated down to 5 or 6 bits.
// 2^5 = 32 shades of Red
// 2^6 = 64 shades of Green
// 2^5 = 32 shades of Blue
//
func RGBto565(r, g, b int, colorOrder devices.ColorOrder) uint16 {
	// 16 bits:
	//   R      G     B
	// 00000 000000 00000

	var lr, lg, lb float64

	// Interpolate 8-8-8 values to 5-6-5 values
	if colorOrder == devices.BGROrder {
		lr = lerp(31.0, 255.0, float64(b)) // 8bit to 5bit
		lg = lerp(63.0, 255.0, float64(g)) // 8bit to 6bit
		lb = lerp(31.0, 255.0, float64(r)) // 8bit to 5bit
	} else {
		lr = lerp(31.0, 255.0, float64(r)) // 8bit to 5bit
		lg = lerp(63.0, 255.0, float64(g)) // 8bit to 6bit
		lb = lerp(31.0, 255.0, float64(b)) // 8bit to 5bit
	}

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
