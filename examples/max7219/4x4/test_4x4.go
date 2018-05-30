package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wdevore/hardware/ftdi/devices/max7219"
)

// Tests a grid of chained 8x8 led matrices arranged
// in a 4x4 grid resulting in 32x32 pixel dimension.

// Pin wiring:
// FTDI232H             4x1 cascade
// D0                   CLK
// D1                   DIN
// D3                   CS
//                      Vcc = 3.3
//                      Grnd = ground

var quit bool

func main() {
	quit = false

	matrix := max.NewMatrix4x4(4000000, 1)

	if matrix == nil {
		panic("Could not create matrix")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(m max.IMatrix) {
		<-c
		quit = true
		log.Println("\nReceived ctrl-C, closing matrix.")
	}(matrix)

	defer exitProg(matrix)

	err := matrix.Initialize()

	if err != nil {
		panic(err)
	}

	matrix.ClearDevice()

	// testActivateTestMode(matrix)
	// testDisplayBlink(matrix)
	// testDisplayFlash(matrix)
	// testCornerPixels(matrix)
	// testCross(matrix)
	// testVerticalScanBar(matrix)
	// testHorizontalScanBar(matrix)
	// testMatrix(matrix)
	// testThinking(matrix)
	testGravityBounce(matrix)
}

func exitProg(m max.IMatrix) {
	log.Println("Closing devices")
	m.ActivateTestMode(false)
	m.ClearDevice()

	err := m.Close()
	if err != nil {
		log.Println("\n Failed to close matrix")
		os.Exit(-1)
	}
	os.Exit(0)
}

func testThinking(m max.IMatrix) {
	ran := rand.New(rand.NewSource(99))

	for {
		if quit {
			break
		}

		// Generate a column
		col := int(ran.Float32() * float32(m.GetWidth()))

		// Generate a row
		row := int(ran.Float32() * float32(m.GetHeight()))

		// Random state
		var bit uint8
		if ran.Float32() > 0.9 {
			bit = 1
		} else {
			bit = 0
		}

		m.ChangePixel(col, row, bit)

		m.UpdateDisplay()

		// time.Sleep(time.Millisecond * 10)
	}

	log.Println("Done.")
}

func testMatrix(m max.IMatrix) {
	ran := rand.New(rand.NewSource(99))
	m.ClearDisplay()

	for {
		if quit {
			break
		}

		// Set a random pixel on row 0
		rcol := int(ran.Float32() * float32(m.GetWidth()))

		m.SetPixel(rcol, 0)

		m.UpdateDisplay()

		time.Sleep(time.Millisecond * 35)

		// scroll downwards
		for row := m.GetHeight() - 1; row > 0; row-- {
			// copy row-1 into row
			for col := 0; col <= m.GetWidth()-1; col++ {
				m.ChangePixel(col, row, m.GetPixel(col, row-1))
			}
		}

		// clear row 0
		m.ClearPixel(rcol, 0)
	}
}

func testVerticalScanBar(m max.IMatrix) {
	d := 1
	c := 0

	for {
		if quit {
			break
		}

		m.ClearDisplay()
		if c > m.GetWidth()-1 {
			c = m.GetWidth() - 2
			d = -1
		} else if c < 0 {
			c = 1
			d = 1
		}

		m.DrawVLine(c, 0, m.GetHeight())
		// m.PrintBuf()

		c += d

		m.UpdateDisplay()

		time.Sleep(time.Millisecond * 20)
	}

	log.Println("Done.")
}

func testHorizontalScanBar(m max.IMatrix) {
	d := 1
	c := 0

	for {
		if quit {
			break
		}

		m.ClearDisplay()
		if c > m.GetHeight()-1 {
			c = m.GetHeight() - 2
			d = -1
		} else if c < 0 {
			c = 1
			d = 1
		}

		m.DrawHLine(0, c, m.GetWidth())
		// m.PrintBuf()

		c += d

		m.UpdateDisplay()

		time.Sleep(time.Millisecond * 20)
	}

	log.Println("Done.")
}

func testCross(m max.IMatrix) {
	m.ClearDisplay()

	// [0 0 1 0 0 0 1 0]
	// [0 0 1 0 0 0 1 0]
	// [1 1 1 1 1 1 1 1]
	// [0 0 1 0 0 0 1 0]
	// [0 0 1 0 0 0 1 0]
	// [0 0 1 0 0 0 1 0]
	// [1 1 1 1 1 1 1 1]
	// [0 0 1 0 0 0 1 0]

	m.DrawHLine(0, 2, m.GetWidth())
	m.DrawHLine(0, m.GetHeight()-2, m.GetWidth())

	m.DrawVLine(2, 0, m.GetHeight())
	m.DrawVLine(m.GetWidth()-2, 0, m.GetHeight())

	// m.PrintBuf()
	m.UpdateDisplay()

	fmt.Println("Done.")
}

func testActivateTestMode(m max.IMatrix) {
	m.ActivateTestMode(true)
	time.Sleep(time.Second)
	m.ActivateTestMode(false)
}

// This test blinks the display by flood filling rather than
// using the testmode register
func testDisplayFlash(m max.IMatrix) {
	blink := true
	for {
		if quit {
			break
		}

		if blink {
			m.DrawRectangle(0, 0, m.GetWidth(), m.GetHeight())
		} else {
			m.ClearDisplay()
		}
		blink = !blink

		m.UpdateDisplay()

		time.Sleep(time.Millisecond * 10)
	}
}

func testDisplayBlink(m max.IMatrix) {
	blink := true
	for {
		if quit {
			break
		}
		m.ActivateTestMode(blink)
		blink = !blink
		time.Sleep(time.Millisecond * 200)
	}
}

func testCornerPixels(m max.IMatrix) {
	for {
		if quit {
			break
		}
		m.ClearDisplay()

		m.SetPixel(0, 0)
		m.SetPixel(m.GetWidth()-1, 0)
		m.SetPixel(0, m.GetHeight()-1)
		m.SetPixel(m.GetWidth()-1, m.GetHeight()-1)

		// m.PrintBuf()
		m.UpdateDisplay()
	}
	fmt.Println("Done.")
}

// ------------------------------------------------------------------
// Gravity bounce
// ------------------------------------------------------------------
// The pixel data just before gravity is applied
var originalBuf [][]byte

const (
	thinkingPeriod = 3 // seconds
	gravityPeriod  = 8
	wigglePeriod   = 1.0
	gravity        = 0.0001 // acceleration
)

// This test does thinking for a certain period.
// Then it applies gravity to each pixel where each
// pixel bounces of the bottom for a fixed period.
// All pixels then move back to their original postion
// using an easing-bounce.
// Then thinking resumes.
// Repeat.
func testGravityBounce(m max.IMatrix) {
	originalBuf = make([][]byte, m.GetHeight())
	for i := range originalBuf {
		originalBuf[i] = make([]byte, m.GetWidth())
	}

	for {
		if quit {
			break
		}

		thinkFor(m, thinkingPeriod)

		// Track the original pixel positions.

		runGravity(m)
		runEasing(m)

	}

	fmt.Println("Done.")
}

type vector struct {
	ox, oy int

	x int
	y float64
	v float64 // velocity
	e float64 // energy loss
}

var pixels []*vector

func runGravity(m max.IMatrix) {
	ran := rand.New(rand.NewSource(666))

	pixels = []*vector{}

	// Capture all pixels into a collection of velocity vectors
	for y := 0; y < m.GetHeight(); y++ {
		for x := 0; x < m.GetWidth(); x++ {
			if m.GetPixel(x, y) == 1 {
				v := new(vector)
				v.ox = x
				v.oy = y
				v.x = x
				v.y = float64(y)
				v.v = -0.05 //* ran.Float64() //(ran.Float64() / (v.y + 2.0)) + 0.001
				v.e = 1.0
				pixels = append(pixels, v)
			}
		}
	}

	start := time.Now()

	duration := time.Duration(gravityPeriod) * time.Second

	// Apply gravity
	for time.Now().Sub(start) <= duration {
		if quit {
			break
		}
		m.ClearDisplay()

		// update velocity
		for _, p := range pixels {
			p.v += gravity
			if p.v > 1.0 {
				p.v = 1.0
			}
			p.y += p.v
			// fmt.Printf("v: %f\n", p.v)

			if p.y > float64(m.GetHeight()) {
				p.y = float64(m.GetHeight() - 1)
				p.v = -(ran.Float64() / 1.0) * 0.08 / p.e
				p.e += 0.3
			}
			m.SetPixel(p.x, int(p.y))
		}

		m.UpdateDisplay()
	}
}

func runEasing(m max.IMatrix) {
	for t := 0.0; t < 1.0; t += 0.02 {
		if quit {
			break
		}
		m.ClearDisplay()

		// Move them back into their original positions.
		j := 0
		for _, p := range pixels {
			change := float64(p.oy) - p.y

			cy := cubicEaseIn(t, p.y, change, wigglePeriod)

			m.SetPixel(p.x, int(cy))
			j++
		}

		m.UpdateDisplay()
		time.Sleep(time.Millisecond * 10)
	}
}

/// Easing equation function for a cubic (t^3) easing in:
/// accelerating from zero velocity.
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func cubicEaseIn(t, b, c, d float64) float64 {
	t1 := t / d
	return c*t1*t*t + b
}

/// Easing equation function for a cubic (t^3) easing out:
/// decelerating from zero velocity.
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func cubicEaseOut(t, b, c, d float64) float64 {
	t1 := t/d - 1
	return c*(t1*t*t+1) + b
}

/// Easing equation function for a quintic (t^5) easing out:
/// decelerating from zero velocity.
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func quintEaseOut(t, b, c, d float64) float64 {
	t1 := t/d - 1
	return c*(t1*t*t*t*t+1) + b
}

/// Easing equation function for a back (overshooting cubic easing: (s+1)*t^3 - s*t^2) easing out:
/// decelerating from zero velocity.
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func backEaseOut(t, b, c, d float64) float64 {
	t1 := t/d - 1.0
	return c*(t1*t*((1.70158+1.0)*t+1.70158)+1.0) + b
}

/// <summary>
/// Easing equation function for a back (overshooting cubic easing: (s+1)*t^3 - s*t^2) easing in:
/// accelerating from zero velocity.
/// </summary>
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func backEaseIn(t, b, c, d float64) float64 {
	t1 := t / d
	return c*t1*t*((1.70158+1)*t-1.70158) + b
}

/// Easing equation function for an elastic (exponentially decaying sine wave) easing in:
/// accelerating from zero velocity.
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func elasticEaseIn(t, b, c, d float64) float64 {
	t1 := t / d
	if t1 == 1.0 {
		return b + c
	}

	p := d * 0.3
	s := p / 4

	t2 := t - 1
	return -(c * math.Pow(2, 10*t2) * math.Sin((t*d-s)*(2*math.Pi)/p)) + b
}

/// Easing equation function for an elastic (exponentially decaying sine wave) easing in/out:
/// acceleration until halfway, then deceleration.
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func elasticEaseInOut(t, b, c, d float64) float64 {
	t1 := t / d / 2
	if t1 == 2 {
		return b + c
	}

	p := d * (0.3 * 1.5)
	s := p / 4

	t2 := t - 1
	if t < 1 {
		return -.5*(c*math.Pow(2, 10*t2)*math.Sin((t*d-s)*(2*math.Pi)/p)) + b
	}

	return c*math.Pow(2, -10*t2)*math.Sin((t*d-s)*(2*math.Pi)/p)*0.5 + c + b
}

/// Easing equation function for a sinusoidal (sin(t)) easing in:
/// accelerating from zero velocity.
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func sineEaseIn(t, b, c, d float64) float64 {
	return -c*math.Cos(t/d*(math.Pi/2)) + c + b
}

/// Easing equation function for a sinusoidal (sin(t)) easing out:
/// decelerating from zero velocity.
/// <param name="t">Current time in seconds.</param>
/// <param name="b">Starting value.</param>
/// <param name="c">Final value.</param>
/// <param name="d">Duration of animation.</param>
/// <returns>The correct value.</returns>
func sineEaseOut(t, b, c, d float64) float64 {
	return c*math.Sin(t/d*(math.Pi/2)) + b
}

func thinkFor(m max.IMatrix, timeFor int64) {
	ran := rand.New(rand.NewSource(99))

	start := time.Now()

	duration := time.Duration(timeFor) * time.Second

	for time.Now().Sub(start) <= duration {
		if quit {
			break
		}

		// Generate a column
		col := int(ran.Float32()*(float32(m.GetWidth())) + 0.5)

		// Generate a row
		row := int(ran.Float32() * float32(m.GetHeight()))

		// Random state
		var bit uint8
		if ran.Float32() > 0.9 {
			bit = 1
		} else {
			bit = 0
		}

		m.ChangePixel(col, row, bit)

		m.UpdateDisplay()
	}
}
