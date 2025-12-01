package newcast

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

const (
	ScaleFactor    = 100.0
	MaxScaledValue = 32767.0
	MinScaledValue = -32768.0
)

// Payload represents the data structure for CBOR encoding
type Payload struct {
	Width  int     `cbor:"width"`
	Height int     `cbor:"height"`
	Scale  float64 `cbor:"scale"`
	Data   []int16 `cbor:"data"`
}

// MarshalGzippedCBOR encodes the AverageFlowGrid to gzipped CBOR format with overflow checks
func (g *AverageFlowGrid) MarshalGzippedCBOR() ([]byte, error) {
	// Prepare the payload with the required data
	payload := Payload{
		Width:  g.Width,
		Height: g.Height,
		Scale:  ScaleFactor, // 100.0 as specified
		Data:   make([]int16, len(g.Data)*2), // Each vector has 2 components (Vx, Vy)
	}

	// Process each vector to convert float64 to int16 with overflow checks
	for i, vec := range g.Data {
		// Check Vx overflow
		scaledVx := vec.Vx * ScaleFactor
		if scaledVx > MaxScaledValue {
			scaledVx = MaxScaledValue
		} else if scaledVx < MinScaledValue {
			scaledVx = MinScaledValue
		}

		// Check Vy overflow
		scaledVy := vec.Vy * ScaleFactor
		if scaledVy > MaxScaledValue {
			scaledVy = MaxScaledValue
		} else if scaledVy < MinScaledValue {
			scaledVy = MinScaledValue
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

// UnmarshalGzippedCBOR decodes gzipped CBOR data to an AverageFlowGrid
func UnmarshalGzippedCBOR(data []byte) (*AverageFlowGrid, error) {
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

	// Create grid with the proper dimensions and data
	grid := &AverageFlowGrid{
		Width:  payload.Width,
		Height: payload.Height,
		Data:   make([]Vector, payload.Width*payload.Height),
	}

	// Convert int16 values back to float64
	for i := 0; i < payload.Width*payload.Height; i++ {
		grid.Data[i] = Vector{
			Vx: float64(payload.Data[i*2]) / payload.Scale,
			Vy: float64(payload.Data[i*2+1]) / payload.Scale,
		}
	}

	return grid, nil
}
