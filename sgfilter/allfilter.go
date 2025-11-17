package sgfilter

import (
	"fmt"
	"math"
)

// CatmullRomSpline interpolates smoothly through points
// Returns interpolated points and velocities
// numInterpolated: how many points between each original point (e.g., 1 = same density)
func CatmullRomSpline(points []Point2D, dt float64, tension float64) (smoothed []Point2D, velocities []Point2D) {
	n := len(points)
	if n < 4 {
		return points, make([]Point2D, n)
	}

	smoothed = make([]Point2D, n)
	velocities = make([]Point2D, n)

	// Apply Catmull-Rom smoothing by using it to create a smoother path through the points
	// For each point i, we'll calculate a weighted average using the Catmull-Rom approach
	for i := 0; i < n; i++ {
		// Get the 4 control points for Catmull-Rom spline
		p0, p1, p2, p3 := getCatmullRomPoints(points, i)

		// Use t=0.5 to get a balanced point between p1 and p2 that smooths
		t := 0.5
		pos, vel := evaluateCatmullRom(p0, p1, p2, p3, t, tension)

		// Use the Catmull-Rom point as the smoothed value
		smoothed[i] = pos
		velocities[i] = Point2D{X: vel.X / dt, Y: vel.Y / dt}
	}

	return smoothed, velocities
}

func getCatmullRomPoints(points []Point2D, i int) (p0, p1, p2, p3 Point2D) {
	n := len(points)

	// Handle boundaries by extrapolating
	if i == 0 {
		p0 = Point2D{
			X: 2*points[0].X - points[1].X,
			Y: 2*points[0].Y - points[1].Y,
		}
		p1 = points[0]
		p2 = points[1]
		p3 = points[2]
	} else if i == n-1 {
		p0 = points[n-3]
		p1 = points[n-2]
		p2 = points[n-1]
		p3 = Point2D{
			X: 2*points[n-1].X - points[n-2].X,
			Y: 2*points[n-1].Y - points[n-2].Y,
		}
	} else if i == n-2 {
		p0 = points[i-1]
		p1 = points[i]
		p2 = points[i+1]
		p3 = Point2D{
			X: 2*points[i+1].X - points[i].X,
			Y: 2*points[i+1].Y - points[i].Y,
		}
	} else {
		p0 = points[i-1]
		p1 = points[i]
		p2 = points[i+1]
		p3 = points[i+2]
	}

	return p0, p1, p2, p3
}

func evaluateCatmullRom(p0, p1, p2, p3 Point2D, t, tension float64) (pos, vel Point2D) {
	// Catmull-Rom basis matrix (with tension parameter)
	// tension = 0.5 is standard Catmull-Rom
	// tension = 0.0 gives sharper curves
	// tension = 1.0 gives smoother curves

	t2 := t * t
	t3 := t2 * t

	// Position coefficients
	c0 := -tension*t3 + 2*tension*t2 - tension*t
	c1 := (2-tension)*t3 + (tension-3)*t2 + 1
	c2 := (tension-2)*t3 + (3-2*tension)*t2 + tension*t
	c3 := tension*t3 - tension*t2

	pos.X = c0*p0.X + c1*p1.X + c2*p2.X + c3*p3.X
	pos.Y = c0*p0.Y + c1*p1.Y + c2*p2.Y + c3*p3.Y

	// Velocity coefficients (derivative of position)
	d0 := -3*tension*t2 + 4*tension*t - tension
	d1 := 3*(2-tension)*t2 + 2*(tension-3)*t
	d2 := 3*(tension-2)*t2 + 2*(3-2*tension)*t + tension
	d3 := 3*tension*t2 - 2*tension*t

	vel.X = d0*p0.X + d1*p1.X + d2*p2.X + d3*p3.X
	vel.Y = d0*p0.Y + d1*p1.Y + d2*p2.Y + d3*p3.Y

	return pos, vel
}

// Kalman filter - better for tracking with motion model
type KalmanFilter2D struct {
	// State: [x, y, vx, vy]
	State [4]float64
	// Covariance matrix (simplified, diagonal)
	P [4]float64
	// Process noise
	Q float64
	// Measurement noise
	R float64
	// Time step
	dt float64
}

func NewKalmanFilter2D(initialPos Point2D, dt, processNoise, measurementNoise float64) *KalmanFilter2D {
	return &KalmanFilter2D{
		State: [4]float64{initialPos.X, initialPos.Y, 0, 0},
		P:     [4]float64{1, 1, 1, 1},
		Q:     processNoise,
		R:     measurementNoise,
		dt:    dt,
	}
}

func (kf *KalmanFilter2D) Update(measurement Point2D) (pos, vel Point2D) {
	dt := kf.dt

	// Predict step
	// x = x + vx*dt
	// y = y + vy*dt
	// vx = vx (constant velocity model)
	// vy = vy
	kf.State[0] += kf.State[2] * dt
	kf.State[1] += kf.State[3] * dt

	// Update covariance
	for i := range kf.P {
		kf.P[i] += kf.Q
	}

	// Update step (measurement update for position only)
	// Kalman gain for x and y positions
	kx := kf.P[0] / (kf.P[0] + kf.R)
	ky := kf.P[1] / (kf.P[1] + kf.R)

	// Innovation (measurement - prediction)
	innX := measurement.X - kf.State[0]
	innY := measurement.Y - kf.State[1]

	// Update state
	kf.State[0] += kx * innX
	kf.State[1] += ky * innY

	// Update velocities based on position correction
	kf.State[2] += (kx * innX) / dt
	kf.State[3] += (ky * innY) / dt

	// Update covariance
	kf.P[0] *= (1 - kx)
	kf.P[1] *= (1 - ky)
	kf.P[2] *= 0.99 // Slight decay to prevent over-confidence
	kf.P[3] *= 0.99

	return Point2D{X: kf.State[0], Y: kf.State[1]},
		Point2D{X: kf.State[2], Y: kf.State[3]}
}

func KalmanSmooth(points []Point2D, dt, processNoise, measurementNoise float64) (smoothed []Point2D, velocities []Point2D) {
	n := len(points)
	if n == 0 {
		return nil, nil
	}

	smoothed = make([]Point2D, n)
	velocities = make([]Point2D, n)

	kf := NewKalmanFilter2D(points[0], dt, processNoise, measurementNoise)

	for i, p := range points {
		smoothed[i], velocities[i] = kf.Update(p)
	}

	return smoothed, velocities
}

// Bilateral filter - edge-preserving smoothing
// Better at preserving sharp curves
func BilateralSmooth(points []Point2D, spatialSigma, rangeSigma float64, windowSize int) []Point2D {
	n := len(points)
	smoothed := make([]Point2D, n)
	halfWindow := windowSize / 2

	for i := 0; i < n; i++ {
		var sumX, sumY, sumWeight float64

		for j := -halfWindow; j <= halfWindow; j++ {
			idx := i + j
			if idx < 0 || idx >= n {
				continue
			}

			// Spatial weight (distance in index space)
			spatialDist := float64(j)
			spatialWeight := math.Exp(-spatialDist * spatialDist / (2 * spatialSigma * spatialSigma))

			// Range weight (distance in value space)
			dx := points[idx].X - points[i].X
			dy := points[idx].Y - points[i].Y
			rangeDist := math.Sqrt(dx*dx + dy*dy)
			rangeWeight := math.Exp(-rangeDist * rangeDist / (2 * rangeSigma * rangeSigma))

			weight := spatialWeight * rangeWeight
			sumX += weight * points[idx].X
			sumY += weight * points[idx].Y
			sumWeight += weight
		}

		if sumWeight > 0 {
			smoothed[i] = Point2D{X: sumX / sumWeight, Y: sumY / sumWeight}
		} else {
			smoothed[i] = points[i]
		}
	}

	return smoothed
}

// CatmullRomFilter applies Catmull-Rom spline smoothing and returns velocities
func CatmullRomFilter(points []Point2D, dt float64) (smoothed []Point2D, velocities []Point2D) {
	// Use the parameters from the original main function: tension = 0.5
	tension := 1.0
	return CatmullRomSpline(points, dt, tension)
}

// KalmanFilter applies Kalman smoothing and returns velocities
func KalmanFilter(points []Point2D, dt float64) (smoothed []Point2D, velocities []Point2D) {
	// Use the parameters from the original main function: processNoise = 0.1, measurementNoise = 0.5
	return KalmanSmooth(points, dt, 0.1, 0.5)
}

// BilateralFilter applies bilateral smoothing and returns velocities
func BilateralFilter(points []Point2D, dt float64) ([]Point2D, []Point2D) {
	// Use improved parameters for better smoothing:
	spatialSigma := 4.0 // Increased to consider spatial neighbors more strongly
	rangeSigma := 2.0   // Increased to allow more range variation before decreasing weights
	windowSize := 7     // Increased window size for more neighbors
	smoothed := BilateralSmooth(points, spatialSigma, rangeSigma, windowSize)
	// Since BilateralSmooth doesn't return velocities, we compute them separately
	velocities := ComputeVelocities(smoothed, dt)
	return smoothed, velocities
}

func main() {
	// Create a curved path (circle segment) with noise
	points := make([]Point2D, 20)
	for i := range points {
		angle := float64(i) * math.Pi / 10.0 // 0 to π
		radius := 10.0
		noise := (math.Sin(float64(i)*2.5) * 0.3) // Simulated optical flow noise

		points[i] = Point2D{
			X: radius*math.Cos(angle) + noise,
			Y: radius*math.Sin(angle) + noise*0.8,
		}
	}

	dt := 0.1

	fmt.Println("Original points (curved path with noise):")
	for i, p := range points {
		fmt.Printf("%d: (%.3f, %.3f)\n", i, p.X, p.Y)
	}

	// Method 1: Catmull-Rom Spline (good for curves)
	fmt.Println("\n--- Catmull-Rom Spline ---")
	smoothed1, vel1 := CatmullRomSpline(points, dt, 0.5)
	fmt.Println("Smoothed points:")
	for i := range smoothed1[:5] {
		fmt.Printf("%d: (%.3f, %.3f), vel: (%.3f, %.3f)\n",
			i, smoothed1[i].X, smoothed1[i].Y, vel1[i].X, vel1[i].Y)
	}

	// Method 2: Kalman Filter (assumes motion model)
	fmt.Println("\n--- Kalman Filter ---")
	smoothed2, vel2 := KalmanSmooth(points, dt, 0.1, 0.5)
	fmt.Println("Smoothed points:")
	for i := range smoothed2[:5] {
		fmt.Printf("%d: (%.3f, %.3f), vel: (%.3f, %.3f)\n",
			i, smoothed2[i].X, smoothed2[i].Y, vel2[i].X, vel2[i].Y)
	}

	// Method 3: Bilateral Filter (edge-preserving)
	fmt.Println("\n--- Bilateral Filter ---")
	smoothed3 := BilateralSmooth(points, 2.0, 1.0, 5)
	fmt.Println("Smoothed points:")
	for i := range smoothed3[:5] {
		fmt.Printf("%d: (%.3f, %.3f)\n", i, smoothed3[i].X, smoothed3[i].Y)
	}
}
