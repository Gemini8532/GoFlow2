package newcast

import (
	"gocv.io/x/gocv"
	"math"
	"testing"
	"time"
)

func TestFitQuadratic(t *testing.T) {
	// Test case 1: Simple parabolic motion
	t0 := time.Now()
	points := []Point{
		{Time: t0, Vec: gocv.Point2f{X: 0, Y: 0}},
		{Time: t0.Add(1 * time.Second), Vec: gocv.Point2f{X: 1, Y: 1}},
		{Time: t0.Add(2 * time.Second), Vec: gocv.Point2f{X: 4, Y: 4}},
		{Time: t0.Add(3 * time.Second), Vec: gocv.Point2f{X: 9, Y: 9}},
	}

	polyX, polyY, err := FitQuadratic(points)
	if err != nil {
		t.Fatalf("FitQuadratic failed: %v", err)
	}

	// For x = t^2, we expect A=1, B=0, C=0
	// For y = t^2, we expect A=1, B=0, C=0
	if math.Abs(polyX.A-1.0) > 0.01 || math.Abs(polyX.B) > 0.01 || math.Abs(polyX.C) > 0.01 {
		t.Errorf("Expected X polynomial to be t^2, got a*t^2 + b*t + c with a=%.2f, b=%.2f, c=%.2f", polyX.A, polyX.B, polyX.C)
	}
	if math.Abs(polyY.A-1.0) > 0.01 || math.Abs(polyY.B) > 0.01 || math.Abs(polyY.C) > 0.01 {
		t.Errorf("Expected Y polynomial to be t^2, got a*t^2 + b*t + c with a=%.2f, b=%.2f, c=%.2f", polyY.A, polyY.B, polyY.C)
	}

	// Test evaluation
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
	t0 := time.Now()
	points := []Point{
		{Time: t0, Vec: gocv.Point2f{X: 0, Y: 0}},
		{Time: t0.Add(1 * time.Second), Vec: gocv.Point2f{X: 1, Y: 0}},
		{Time: t0.Add(2 * time.Second), Vec: gocv.Point2f{X: 4, Y: 0}},
		{Time: t0.Add(3 * time.Second), Vec: gocv.Point2f{X: 9, Y: 0}},
	}

	polyX, _, err := FitQuadratic(points)
	if err != nil {
		t.Fatalf("FitQuadratic failed: %v", err)
	}

	// At t=2, velocity should be 2*t = 4
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
	t0 := time.Now()
	points := []Point{
		{Time: t0, Vec: gocv.Point2f{X: 0, Y: 0}},
		{Time: t0.Add(1 * time.Second), Vec: gocv.Point2f{X: 1, Y: 1}},
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
	// Create points that result in a singular matrix (all at same time)
	t0 := time.Now()
	points := []Point{
		{Time: t0, Vec: gocv.Point2f{X: 0, Y: 0}},
		{Time: t0, Vec: gocv.Point2f{X: 1, Y: 1}},  // Same time as above
		{Time: t0, Vec: gocv.Point2f{X: 2, Y: 2}},  // Same time as above
		{Time: t0, Vec: gocv.Point2f{X: 3, Y: 3}},  // Same time as above
	}

	_, _, err := FitQuadratic(points)
	if err == nil {
		t.Error("Expected error for singular matrix, got nil")
	}
	if err.Error() != "failed to fit curve (singular matrix)" {
		t.Errorf("Expected 'failed to fit curve (singular matrix)', got '%v'", err.Error())
	}
}