package newcast

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/fxamacker/cbor/v2"
)

// Payload represents the data structure for CBOR encoding
type Payload struct {
	Width  int      `cbor:"width"`
	Height int      `cbor:"height"`
	Scale  float64  `cbor:"scale"`
	Data   []int16  `cbor:"data"`
}

// MarshalGzippedCBOR encodes the Frame to gzipped CBOR format with overflow checks
func (f *Frame) MarshalGzippedCBOR() ([]byte, error) {
	// Prepare the payload with the required data
	payload := Payload{
		Width:  f.Width,
		Height: f.Height,
		Scale:  ScaleFactor, // 100.0 as specified
		Data:   make([]int16, len(f.Data)*2), // Each vector has 2 components (Vx, Vy)
	}

	// Process each vector to convert float64 to int16 with overflow checks
	for i, vec := range f.Data {
		// Check Vx overflow
		scaledVx := vec.Vx * ScaleFactor
		if scaledVx > MaxScaledValue || scaledVx < MinScaledValue {
			return nil, fmt.Errorf("Vx overflow at index %d: value %f (scaled: %f) outside range [%f, %f]", 
				i, vec.Vx, scaledVx, MinScaledValue, MaxScaledValue)
		}

		// Check Vy overflow
		scaledVy := vec.Vy * ScaleFactor
		if scaledVy > MaxScaledValue || scaledVy < MinScaledValue {
			return nil, fmt.Errorf("Vy overflow at index %d: value %f (scaled: %f) outside range [%f, %f]", 
				i, vec.Vy, scaledVy, MinScaledValue, MaxScaledValue)
		}

		// Convert and store in payload
		payload.Data[i*2] = int16(scaledVx)
		payload.Data[i*2+1] = int16(scaledVy)
	}

	// Marshal to CBOR
	cborData, err := cbor.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to CBOR: %w", err)
	}

	// Compress with gzip
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(cborData); err != nil {
		return nil, fmt.Errorf("failed to compress with gzip: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// UnmarshalGzippedCBOR decodes gzipped CBOR data to a Frame
func (f *Frame) UnmarshalGzippedCBOR(data []byte) error {
	// Decompress the gzip data
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	// Read the decompressed CBOR data
	var cborBuf bytes.Buffer
	if _, err := cborBuf.ReadFrom(gz); err != nil {
		return fmt.Errorf("failed to read decompressed data: %w", err)
	}

	// Unmarshal CBOR to payload
	var payload Payload
	if err := cbor.Unmarshal(cborBuf.Bytes(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal CBOR: %w", err)
	}

	// Validate dimensions
	expectedLen := payload.Width * payload.Height * 2 // Each vector has 2 components
	if len(payload.Data) != expectedLen {
		return errors.New("data length doesn't match width*height*2")
	}

	// Update frame dimensions and data
	f.Width = payload.Width
	f.Height = payload.Height
	f.Data = make([]Vector, payload.Width*payload.Height)

	// Convert int16 values back to float64
	for i := 0; i < payload.Width*payload.Height; i++ {
		f.Data[i].Vx = float64(payload.Data[i*2]) / payload.Scale
		f.Data[i].Vy = float64(payload.Data[i*2+1]) / payload.Scale
	}

	return nil
}

// MarshalGzippedCBORFromAverageFlowGrid converts AverageFlowGrid to gzipped CBOR format with overflow checks
func (a *AverageFlowGrid) MarshalGzippedCBOR() ([]byte, error) {
	// Prepare the payload with the required data
	payload := Payload{
		Width:  a.Width,
		Height: a.Height,
		Scale:  ScaleFactor, // 100.0 as specified
		Data:   make([]int16, len(a.Data)*2), // Each vector has 2 components (Vx, Vy)
	}

	// Process each vector to convert float64 to int16 with overflow checks
	for i, vec := range a.Data {
		// Check Vx overflow
		scaledVx := vec.Vx * ScaleFactor
		if scaledVx > MaxScaledValue || scaledVx < MinScaledValue {
			return nil, fmt.Errorf("Vx overflow at index %d: value %f (scaled: %f) outside range [%f, %f]", 
				i, vec.Vx, scaledVx, MinScaledValue, MaxScaledValue)
		}

		// Check Vy overflow
		scaledVy := vec.Vy * ScaleFactor
		if scaledVy > MaxScaledValue || scaledVy < MinScaledValue {
			return nil, fmt.Errorf("Vy overflow at index %d: value %f (scaled: %f) outside range [%f, %f]", 
				i, vec.Vy, scaledVy, MinScaledValue, MaxScaledValue)
		}

		// Convert and store in payload
		payload.Data[i*2] = int16(scaledVx)
		payload.Data[i*2+1] = int16(scaledVy)
	}

	// Marshal to CBOR
	cborData, err := cbor.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to CBOR: %w", err)
	}

	// Compress with gzip
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(cborData); err != nil {
		return nil, fmt.Errorf("failed to compress with gzip: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// UnmarshalGzippedCBORToFrame converts gzipped CBOR data to a Frame
func UnmarshalGzippedCBORToFrame(data []byte) (*Frame, error) {
	// Decompress the gzip data
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	// Read the decompressed CBOR data
	var cborBuf bytes.Buffer
	if _, err := cborBuf.ReadFrom(gz); err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	// Unmarshal CBOR to payload
	var payload Payload
	if err := cbor.Unmarshal(cborBuf.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CBOR: %w", err)
	}

	// Validate dimensions
	expectedLen := payload.Width * payload.Height * 2 // Each vector has 2 components
	if len(payload.Data) != expectedLen {
		return nil, errors.New("data length doesn't match width*height*2")
	}

	// Create frame with the proper dimensions and data
	frame := &Frame{
		Width:  payload.Width,
		Height: payload.Height,
		Data:   make([]Vector, payload.Width*payload.Height),
	}

	// Convert int16 values back to float64
	for i := 0; i < payload.Width*payload.Height; i++ {
		frame.Data[i] = Vector{
			Vx: float64(payload.Data[i*2]) / payload.Scale,
			Vy: float64(payload.Data[i*2+1]) / payload.Scale,
		}
	}

	return frame, nil
}