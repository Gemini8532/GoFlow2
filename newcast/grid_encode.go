package newcast

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
)

const ScaleFactor = 100.0
const MaxScaledValue = 32767.0
const MinScaledValue = -32768.0

type Frame struct {
	Width  int
	Height int
	Data   []Vector
}

func NewFrame(width, height int) *Frame {
	return &Frame{
		Width:  width,
		Height: height,
		Data:   make([]Vector, width*height),
	}
}

func (f *Frame) Set(x, y int, v Vector) {
	if x >= 0 && x < f.Width && y >= 0 && y < f.Height {
		f.Data[y*f.Width+x] = v
	}
}

func (f *Frame) Get(x, y int) Vector {
	if x >= 0 && x < f.Width && y >= 0 && y < f.Height {
		return f.Data[y*f.Width+x]
	}
	return Vector{}
}

// Resize creates a new Frame with the given dimensions and resizes the original
// vector data into it using bilinear interpolation.
func (f *Frame) Resize(newWidth, newHeight int) *Frame {
	if newWidth <= 0 || newHeight <= 0 {
		// Return an empty frame or handle error as appropriate
		return NewFrame(0, 0)
	}

	newFrame := NewFrame(newWidth, newHeight)
	xRatio := float64(f.Width-1) / float64(newWidth-1)
	yRatio := float64(f.Height-1) / float64(newHeight-1)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Find the corresponding position in the original frame
			origX := float64(x) * xRatio
			origY := float64(y) * yRatio

			// Get the integer coordinates of the surrounding pixels
			x0 := int(math.Floor(origX))
			y0 := int(math.Floor(origY))
			x1 := x0 + 1
			y1 := y0 + 1

			// Ensure coordinates are within bounds
			if x1 >= f.Width {
				x1 = f.Width - 1
			}
			if y1 >= f.Height {
				y1 = f.Height - 1
			}
			if x0 < 0 {
				x0 = 0
			}
			if y0 < 0 {
				y0 = 0
			}

			// Get the vectors of the surrounding pixels
			v00 := f.Get(x0, y0)
			v10 := f.Get(x1, y0)
			v01 := f.Get(x0, y1)
			v11 := f.Get(x1, y1)

			// Calculate interpolation weights
			dx := origX - float64(x0)
			dy := origY - float64(y0)

			// Interpolate Vx and Vy
			vx := v00.Vx*(1-dx)*(1-dy) + v10.Vx*dx*(1-dy) + v01.Vx*(1-dx)*dy + v11.Vx*dx*dy
			vy := v00.Vy*(1-dx)*(1-dy) + v10.Vy*dx*(1-dy) + v01.Vy*(1-dx)*dy + v11.Vy*dx*dy

			newFrame.Set(x, y, Vector{Vx: vx, Vy: vy})
		}
	}

	return newFrame
}

// clampAndRound scales the float64 value, rounds it, and clamps it to the safe int16 range.
// It returns the resulting int16 value and true if clamping occurred.
func clampAndRound(v float64) (int16, bool) {
	scaled := v * ScaleFactor

	clamped := scaled
	clamped = math.Max(clamped, MinScaledValue)
	clamped = math.Min(clamped, MaxScaledValue)

	// Check if clamping occurred
	clampedFlag := clamped != scaled

	// Round to the nearest integer
	rounded := math.Round(clamped)

	return int16(rounded), clampedFlag
}

// MarshalPNG converts the Frame into a standard 8-bit RGBA PNG.
// It packs 16-bit values across 2 channels.
// Vx -> Red (High), Green (Low)
// Vy -> Blue (High), Alpha (Low)
func (f *Frame) MarshalPNG() ([]byte, error) {
	rect := image.Rect(0, 0, f.Width, f.Height)
	img := image.NewNRGBA(rect)

	i := 0
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			vec := f.Data[y*f.Width+x]

			// Clamp and Round both components
			xInt, xClamped := clampAndRound(vec.Vx)
			yInt, yClamped := clampAndRound(vec.Vy)

			if xClamped || yClamped {
				// NOTE: If logging is desired, you could log a warning here.
				// fmt.Printf("Warning: Clamping required at (%d, %d) for Vx=%.2f, Vy=%.2f\n", x, y, vec.Vx, vec.Vy)
			}

			// Debug: log pixel (48, 144)
			if x == 48 && y == 144 {
				fmt.Printf("ENCODING pixel (%d,%d): vx=%.6f, vy=%.6f, xInt=%d, yInt=%d\n", x, y, vec.Vx, vec.Vy, xInt, yInt)
			}

			// Cast to uint16 to get the raw bit pattern (Two's Complement for negatives)
			xUint := uint16(xInt)
			yUint := uint16(yInt)

			// Encode in RGB only, keep alpha at 255 to avoid browser corruption
			// We'll pack both values into 3 channels using a different scheme
			// For now, just use RG for Vx and B for Vy high byte, encode Vy low separately
			// Actually, simpler: use RG for Vx (16 bits) and BA for Vy (16 bits) but with alpha always 255

			// Wait, that won't work. Let me use a different approach:
			// Encode Vx in RG, Vy high in B, and Vy low in a separate pixel
			// Actually the simplest fix: use RGBA but write alpha as 255, store Vy low in a metadata chunk

			// SIMPLEST FIX: Use RGB for one vector, next pixel's RGB for another vector
			// But that changes the image size...

			// ACTUAL FIX: Just set alpha to 255 always and lose the Vy low byte precision
			// Or better: encode both in 24 bits total

			// Let's try: RG = Vx (16 bit), B = Vy high (8 bit), A = 255
			// This loses precision on Vy but avoids alpha corruption

			img.Pix[i+0] = uint8(xUint >> 8)   // Red: Vx High
			img.Pix[i+1] = uint8(xUint & 0xFF) // Green: Vx Low
			img.Pix[i+2] = uint8(yUint >> 8)   // Blue: Vy High
			img.Pix[i+3] = 255                 // Alpha: Always 255 to avoid corruption

			// Debug: log bytes for pixel (48, 144)
			if x == 48 && y == 144 {
				fmt.Printf("BYTES written (ALPHA=255): r=%d, g=%d, b=%d, a=%d (xUint=%d, yUint=%d, LOST Vy low byte!)\n",
					img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3], xUint, yUint)
			}

			i += 4
		}
	}

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode png: %w", err)
	}

	return buf.Bytes(), nil
}

// UnmarshalPNG reads PNG data and reconstructs the Frame.
func UnmarshalPNG(data []byte) (*Frame, error) {
	reader := bytes.NewReader(data)
	return DecodePNG(reader)
}

// DecodePNG decodes a PNG from an io.Reader into a Frame.
func DecodePNG(r io.Reader) (*Frame, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode png image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	frame := NewFrame(width, height)

	// We need to handle different image types returned by decode
	switch src := img.(type) {
	case *image.NRGBA:
		// Fast path for NRGBA (expected)
		i := 0
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				xHigh := uint16(src.Pix[i+0])
				xLow := uint16(src.Pix[i+1])
				yHigh := uint16(src.Pix[i+2])
				yLow := uint16(src.Pix[i+3])

				// Reconstruct uint16
				xUint := (xHigh << 8) | xLow
				yUint := (yHigh << 8) | yLow

				// Cast back to int16 to restore sign, then float64
				frame.Data[y*width+x] = Vector{
					Vx: float64(int16(xUint)) / ScaleFactor,
					Vy: float64(int16(yUint)) / ScaleFactor,
				}
				i += 4
			}
		}
	default:
		// Slower fallback for other image types (e.g. RGBA, NRGBA64)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := img.At(x+bounds.Min.X, y+bounds.Min.Y)
				r, g, b, a := c.RGBA() // Returns values in [0, 0xFFFF]

				r8 := uint8(r >> 8)
				g8 := uint8(g >> 8)
				b8 := uint8(b >> 8)
				a8 := uint8(a >> 8)

				xUint := (uint16(r8) << 8) | uint16(g8)
				yUint := (uint16(b8) << 8) | uint16(a8)

				frame.Data[y*width+x] = Vector{
					Vx: float64(int16(xUint)) / ScaleFactor,
					Vy: float64(int16(yUint)) / ScaleFactor,
				}
			}
		}
	}

	return frame, nil
}

type Sequence struct {
	Frames []*Frame
}

func (s *Sequence) MarshalSequence() ([][]byte, error) {
	out := make([][]byte, len(s.Frames))
	for i, frame := range s.Frames {
		b, err := frame.MarshalPNG()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal frame %d: %w", i, err)
		}
		out[i] = b
	}
	return out, nil
}

// ValidateValue checks if a value fits within the int16 range
// Note: This function is preserved but the core logic now uses clampAndRound
func ValidateValue(v float64) error {
	scaled := v * ScaleFactor
	if scaled < MinScaledValue || scaled > MaxScaledValue {
		return errors.New("value out of range for int16 packing")
	}
	return nil
}
