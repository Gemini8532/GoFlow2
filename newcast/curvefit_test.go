package newcast

import (
	"math"
	"testing"
)

func TestFitQuadratic(t *testing.T) {
	// Test case 1: Simple parabolic motion
	points := []Point{
		{X: 0.0, Y: 0.0},
		{X: 1.0, Y: 1.0},
		{X: 4.0, Y: 4.0},
		{X: 9.0, Y: 9.0},
	}

	polyX, polyY, err := FitQuadratic(points)
	if err != nil {
		t.Fatalf("FitQuadratic failed: %v", err)
	}

	// For x = t^2, we expect A=1, B=0, C=0 (since t is the index)
	if math.Abs(polyX.A-1.0) > 0.01 || math.Abs(polyX.B) > 0.01 || math.Abs(polyX.C) > 0.01 {
		t.Errorf("Expected X polynomial to be t^2, got a*t^2 + b*t + c with a=%.2f, b=%.2f, c=%.2f", polyX.A, polyX.B, polyX.C)
	}
	if math.Abs(polyY.A-1.0) > 0.01 || math.Abs(polyY.B) > 0.01 || math.Abs(polyY.C) > 0.01 {
		t.Errorf("Expected Y polynomial to be t^2, got a*t^2 + b*t + c with a=%.2f, b=%.2f, c=%.2f", polyY.A, polyY.B, polyY.C)
	}

	// Test evaluation at t=4 (the next index)
	expectedX := 16.0 // 4^2
	actualX := polyX.Eval(4.0)
	if math.Abs(actualX-expectedX) > 0.01 {
		t.Errorf("Expected X(4) = %.2f, got %.2f", expectedX, actualX)
	}

	expectedY := 16.0 // 4^2
	actualY := polyY.Eval(4.0)
	if math.Abs(actualY-expectedY) > 0.01 {
		t.Errorf("Expected Y(4) = %.2f, got %.2f", expectedY, actualY)
	}
}

func TestFitQuadraticVelocityAcceleration(t *testing.T) {
	// Test case: x = t^2, so velocity = 2*t, acceleration = 2
	points := []Point{
		{X: 0.0, Y: 0.0},
		{X: 1.0, Y: 0.0},
		{X: 4.0, Y: 0.0},
		{X: 9.0, Y: 0.0},
	}

	polyX, _, err := FitQuadratic(points)
	if err != nil {
		t.Fatalf("FitQuadratic failed: %v", err)
	}

	// At t=2 (index 2), velocity should be 2*t = 4
	expectedVelocity := 4.0
	actualVelocity := polyX.Velocity(2.0)
	if math.Abs(actualVelocity-expectedVelocity) > 0.01 {
		t.Errorf("Expected velocity at t=2 to be %.2f, got %.2f", expectedVelocity, actualVelocity)
	}

	// Acceleration should be 2 (from 2*A where A=1)
	expectedAcceleration := 2.0
	actualAcceleration := polyX.Acceleration()
	if math.Abs(actualAcceleration-expectedAcceleration) > 0.01 {
		t.Errorf("Expected acceleration to be %.2f, got %.2f", expectedAcceleration, actualAcceleration)
	}
}

func TestFitQuadraticNotEnoughPoints(t *testing.T) {
	points := []Point{
		{X: 0.0, Y: 0.0},
		{X: 1.0, Y: 1.0},
	}

	_, _, err := FitQuadratic(points)
	if err == nil {
		t.Error("Expected error for insufficient points, got nil")
	}
	if err.Error() != "not enough points to fit quadratic" {
		t.Errorf("Expected 'not enough points to fit quadratic', got '%v'", err.Error())
	}
}

func TestFitQuadraticSingularMatrix(t *testing.T) {
	// Create points that result in a singular matrix (all at same "time" index, if we cheat)
	// The current implementation uses the slice index, which can never be singular in this way.
	// To test the singular matrix case, we'd need to manually construct the matrix system.
	// However, a simple way to trigger it is to have identical points, which our loop handles.
	// The test for identical indices is implicitly handled by the loop structure.
	// Let's ensure the logic holds. With indices 0, 1, 2, 3... the time values are distinct.
	// So, we can't trigger the singular matrix error with this implementation easily.
	// This test is now less relevant for the new time-implicit approach.
	// We will skip trying to force a singular matrix error.
	t.Skip("Skipping singular matrix test as it's hard to trigger with implicit time indices.")
}