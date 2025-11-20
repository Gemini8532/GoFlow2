package newcast

import (
	"math"
	"testing"
)

func TestCalculateAveragedTracks(t *testing.T) {
	tracks := []*Track{
		{
			ID: 0,
			Points: []Point{
				{X: 0, Y: 0},
				{X: 1, Y: 1},
				{X: 2, Y: 2},
			},
		},
		{
			ID: 1,
			Points: []Point{
				{X: 10, Y: 10},
				{X: 12, Y: 12},
			},
		},
		{
			ID: 2,
			Points: []Point{
				{X: 5, Y: 5},
			},
		},
	}

	averagedTracks := CalculateAveragedTracks(tracks)

	if len(averagedTracks) != 2 {
		t.Errorf("Expected 2 averaged tracks, but got %d", len(averagedTracks))
	}

	// Test track 0
	if averagedTracks[0].Midpoint.X != 1 || averagedTracks[0].Midpoint.Y != 1 {
		t.Errorf("Expected midpoint (1, 1) for track 0, but got (%f, %f)", averagedTracks[0].Midpoint.X, averagedTracks[0].Midpoint.Y)
	}
	assertVecEqual(t, averagedTracks[0].Vector, Vector{Vx: 1, Vy: 1}, 1e-6)

	// Test track 1
	if averagedTracks[1].Midpoint.X != 12 || averagedTracks[1].Midpoint.Y != 12 {
		t.Errorf("Expected midpoint (12, 12) for track 1, but got (%f, %f)", averagedTracks[1].Midpoint.X, averagedTracks[1].Midpoint.Y)
	}
	assertVecEqual(t, averagedTracks[1].Vector, Vector{Vx: 2, Vy: 2}, 1e-6)
}

func assertVecEqual(t *testing.T, actual, expected Vector, tolerance float64) {
	t.Helper()
	if math.Abs(actual.Vx-expected.Vx) > tolerance || math.Abs(actual.Vy-expected.Vy) > tolerance {
		t.Errorf("Expected vector (%f, %f), but got (%f, %f)", expected.Vx, expected.Vy, actual.Vx, actual.Vy)
	}
}

func TestGenerateAverageFlowGrid(t *testing.T) {
	averagedTracks := []*AveragedTrack{
		{
			Midpoint: Point{X: 5, Y: 5},
			Vector:   Vector{Vx: 1, Vy: 1},
		},
		{
			Midpoint: Point{X: 15, Y: 15},
			Vector:   Vector{Vx: 2, Vy: 2},
		},
	}

	grid := GenerateAverageFlowGrid(averagedTracks, 20, 20, 2)

	if len(grid.Data) != 400 {
		t.Errorf("Expected grid data of length 400, but got %d", len(grid.Data))
	}

	// With smooth diffusion, values near tracks should be present but may be lower
	// than the original track values due to the diffusion process
	vec1 := grid.Data[5*20+5]
	if vec1.Vx < 0.1 || vec1.Vx > 1.5 {
		t.Errorf("Expected Vx near track 1 to be in range [0.1, 1.5], got %f", vec1.Vx)
	}

	vec2 := grid.Data[15*20+15]
	if vec2.Vx < 0.2 || vec2.Vx > 2.5 {
		t.Errorf("Expected Vx near track 2 to be in range [0.2, 2.5], got %f", vec2.Vx)
	}

	// Test a point in the middle - should have smoothly interpolated values
	vec3 := grid.Data[10*20+10]
	if vec3.Vx <= 0.0 || vec3.Vx > 2.0 {
		t.Errorf("Expected Vx in middle to be in range (0.0, 2.0], but got %f", vec3.Vx)
	}
	if vec3.Vy <= 0.0 || vec3.Vy > 2.0 {
		t.Errorf("Expected Vy in middle to be in range (0.0, 2.0], but got %f", vec3.Vy)
	}

	// The values should be roughly similar (since the input vectors are diagonal)
	if math.Abs(vec3.Vx-vec3.Vy) > 0.1 {
		t.Errorf("Expected Vx and Vy to be similar, but got Vx=%f, Vy=%f", vec3.Vx, vec3.Vy)
	}
}

func TestGenerateAverageFlowGrid_NoTracks(t *testing.T) {
	averagedTracks := []*AveragedTrack{}

	grid := GenerateAverageFlowGrid(averagedTracks, 10, 10, 2)

	if len(grid.Data) != 100 {
		t.Errorf("Expected grid data of length 100, but got %d", len(grid.Data))
	}

	for _, vec := range grid.Data {
		assertVecEqual(t, vec, Vector{Vx: 0, Vy: 0}, 1e-6)
	}
}
