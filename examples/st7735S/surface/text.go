package surface

// Text renders text
type Text struct {
	Text    string
	surface *Surface
}

// NewText creates a new text object
func NewText(surface *Surface) *Text {
	t := new(Text)
	t.surface = surface
	return t
}

// DrawCharColRow draws a character indexed by col/row.
func (t *Text) DrawCharColRow(x, y int, col, row int, color uint16) {
	cw := 8
	yi := y

	t.surface.SetColor(color)

	ci := (row * 16) + col
	for i := ci * cw; i < ci*cw+cw; i++ {
		xi := x
		cl := ConsoleFont8x8[i]
		// fmt.Printf("i: %d\n", i)
		// For each bit we set a pixel based on color
		for p := 0; p < cw; p++ {
			// fmt.Printf("%d, %d, %08b\n", xi, yi, cl)
			bit := (cl << byte(p)) & 0x80
			if bit > 0 {
				t.surface.SetPixel(xi, yi)
			}
			xi++
		}

		yi++
		// fmt.Printf("%08b (%d)\n", st7735S.ConsoleFont8x8[i], i)
	}
}

// DrawChar draws a character at x,y
func (t *Text) DrawChar(x, y int, ch int, foreColor, backColor uint16, transparentBack bool) {
	cw := 8
	yi := y

	for i := ch * cw; i < ch*cw+cw; i++ {
		xi := x
		cl := ConsoleFont8x8[i]
		// fmt.Printf("i: %d\n", i)
		// For each bit we set a pixel based on color
		for p := 0; p < cw; p++ {
			// fmt.Printf("%d, %d, %08b\n", xi, yi, cl)
			bit := (cl << byte(p)) & 0x80
			if bit > 0 {
				t.surface.SetPixelWithColor(xi, yi, foreColor)
			} else {
				if !transparentBack {
					t.surface.SetPixelWithColor(xi, yi, backColor)
				}
			}
			xi++
		}

		yi++
		// fmt.Printf("%08b (%d)\n", st7735S.ConsoleFont8x8[i], i)
	}
}

// DrawText draws text at x,y
func (t *Text) DrawText(x, y int, text string, foreColor, backColor uint16, transparentBack bool) {
	// Render each character
	for _, ch := range text {
		t.DrawChar(x, y, int(ch), foreColor, backColor, transparentBack)
		x += 8
	}
}
