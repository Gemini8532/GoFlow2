package flow

import (
	"fmt"
	"image"
	"math"
	"sort"

	"gocv.io/x/gocv"
)

// motionVector holds the location (x, y) and velocity (u, v) of a single
// tracked feature. This is now an internal struct.
// It uses float32 to be compatible with gocv's Mat data.
type motionVector struct {
	Point    [2]float32 // x, y location
	Velocity [2]float32 // u, v velocity components
}

// --- New Mat-based Public API ---

// CleanseMotionVectorsMat applies cleansing filters to motion vectors stored in gocv.Mat matrices.
// This is the new public-facing function.
//
// Parameters:
//   - prevPtsMat: A gocv.Mat (Type CV_32FC2, Rows: N, Cols: 1) of starting points.
//   - nextPtsMat: A gocv.Mat (Type CV_32FC2, Rows: N, Cols: 1) of tracked points.
//   - statusMat: A gocv.Mat (Type CV_8UC1, Rows: N, Cols: 1) where 1=tracked, 0=lost.
//   - k: The number of nearest neighbors to use for the statistical outlier test.
//   - stdDevThreshold: The number of standard deviations from the local mean
//     to tolerate before flagging a vector as an outlier.
//   - gridCellSize: The size (in pixels) of each cell for the declustering grid.
//   - minSamplesInCell: The minimum number of vectors a declustering cell must
//     contain to produce a single median vector.
//
// Returns:
//   - (gocv.Mat, gocv.Mat): A pair of new Mat matrices (cleansedPrevPts, cleansedNextPts)
//     containing only the cleansed, declustered points.
func CleanseMotionVectorsMat(prevPtsMat, nextPtsMat, statusMat gocv.Mat, k int, stdDevThreshold float64, gridCellSize int, minSamplesInCell int) (gocv.Mat, gocv.Mat) {
	// Step 1: Convert from gocv.Mat to a native Go slice for efficient processing
	// This avoids slow CGo calls inside the nested loops of the cleansing functions.
	rawVectors := matsToMotionVectors(prevPtsMat, nextPtsMat, statusMat)
	fmt.Printf("Starting cleansing with %d raw valid vectors...\n", len(rawVectors))

	// Step 2: Run the internal cleansing logic
	cleanVectors := cleanseMotionVectorsInternal(rawVectors, k, stdDevThreshold, gridCellSize, minSamplesInCell)

	// Step 3: Convert the cleansed Go slice back to gocv.Mat matrices
	cleansedPrevMat, cleansedNextMat := motionVectorsToMats(cleanVectors)

	return cleansedPrevMat, cleansedNextMat
}

// --- Mat-to-Slice and Slice-to-Mat Helpers ---

// matsToMotionVectors converts the Gocv Mat format into a slice of motionVector structs.
// It filters out any points that were not successfully tracked (status == 0).
func matsToMotionVectors(prevPtsMat, nextPtsMat, statusMat gocv.Mat) []motionVector {
	nPoints := prevPtsMat.Rows()
	if nPoints == 0 {
		return nil
	}

	var vectors []motionVector
	for i := 0; i < nPoints; i++ {
		// Check the status Mat. 1 = tracked, 0 = lost
		if statusMat.GetUCharAt(i, 0) == 1 {
			// Read the [X, Y] float32 slice from the Mat
			// This is the version-agnostic way, avoiding image.Point2f
			p0Vec := prevPtsMat.GetVecfAt(i, 0)
			p1Vec := nextPtsMat.GetVecfAt(i, 0)

			p0 := [2]float32{p0Vec[0], p0Vec[1]}
			p1 := [2]float32{p1Vec[0], p1Vec[1]}

			velocity := [2]float32{
				p1[0] - p0[0], // u = p1.X - p0.X
				p1[1] - p0[1], // v = p1.Y - p0.Y
			}

			vectors = append(vectors, motionVector{
				Point:    p0,
				Velocity: velocity,
			})
		}
	}
	return vectors
}

// motionVectorsToMats converts a cleansed slice of motionVector structs back
// into two gocv.Mat matrices (prevPts and nextPts).
func motionVectorsToMats(vectors []motionVector) (gocv.Mat, gocv.Mat) {
	nPoints := len(vectors)
	if nPoints == 0 {
		// Return empty, valid Mats
		return gocv.NewMat(), gocv.NewMat()
	}
	prevPoints := gocv.NewMatWithSize(len(vectors), 2, gocv.MatTypeCV32F)
	defer prevPoints.Close()
	nextPoints := gocv.NewMatWithSize(len(vectors), 2, gocv.MatTypeCV32F)
	defer nextPoints.Close()

	for i, v := range vectors {
		p0 := v.Point
		p1 := [2]float32{
			p0[0] + v.Velocity[0], // p1.X = p0.X + u
			p0[1] + v.Velocity[1], // p1.Y = p0.Y + v
		}
		prevPoints.SetFloatAt(i, 0, p0[0])
		prevPoints.SetFloatAt(i, 1, p0[1])
		nextPoints.SetFloatAt(i, 0, p1[0])
		nextPoints.SetFloatAt(i, 1, p1[1])
	}

	return prevPoints, nextPoints
}

// --- Internal Cleansing Logic (Updated to use [2]float32) ---

// cleanseMotionVectorsInternal applies a series of filters to a raw list of motion vectors
// to remove outliers and normalize density, similar to the Pysteps workflow.
func cleanseMotionVectorsInternal(vectors []motionVector, k int, stdDevThreshold float64, gridCellSize int, minSamplesInCell int) []motionVector {
	// Step 1: Statistical outlier rejection
	// Remove vectors that are statistically different from their k-nearest neighbors.
	inliers := detectOutliers(vectors, k, stdDevThreshold)
	fmt.Printf("  %d vectors remaining after statistical outlier detection.\n", len(inliers))

	// Step 2: Spatial declustering
	// Convert dense clusters of vectors into a single median vector on a coarse grid.
	// This further cleans outliers and normalizes vector density.
	declustered := declusterVectors(inliers, gridCellSize, minSamplesInCell)
	fmt.Printf("  %d vectors remaining after declustering.\n", len(declustered))

	return declustered
}

// detectOutliers filters a vector list using a k-Nearest Neighbors approach.
func detectOutliers(vectors []motionVector, k int, stdDevThreshold float64) []motionVector {
	var inliers []motionVector

	for i, targetVec := range vectors {
		// 1. Find distances to all other points
		type distVec struct {
			dist float64
			vec  motionVector
		}
		var neighborsWithDist []distVec

		for j, otherVec := range vectors {
			if i == j {
				continue
			}
			dist := euclideanDistance(targetVec.Point, otherVec.Point)
			neighborsWithDist = append(neighborsWithDist, distVec{dist, otherVec})
		}

		// 2. Sort by distance to find the k-nearest
		sort.Slice(neighborsWithDist, func(a, b int) bool {
			return neighborsWithDist[a].dist < neighborsWithDist[b].dist
		})

		// 3. Get the k-nearest vectors
		var kNearest []motionVector
		numNeighbors := k
		if len(neighborsWithDist) < k {
			numNeighbors = len(neighborsWithDist)
		}

		// If there are no neighbors, we can't make a judgment. Keep the vector.
		if numNeighbors == 0 {
			inliers = append(inliers, targetVec)
			continue
		}

		for n := 0; n < numNeighbors; n++ {
			kNearest = append(kNearest, neighborsWithDist[n].vec)
		}

		// 4. Calculate the velocity stats of those neighbors
		meanU, stdDevU, meanV, stdDevV := calculateVelocityStats(kNearest)

		// 5. Check if the target vector is an outlier
		isOutlier := false

		// Check U component (Velocity[0])
		if stdDevU > 0.001 { // Avoid division by zero or noise
			if math.Abs(float64(targetVec.Velocity[0])-meanU) > stdDevThreshold*stdDevU {
				isOutlier = true
			}
		}

		// Check V component (Velocity[1])
		if !isOutlier && stdDevV > 0.001 {
			if math.Abs(float64(targetVec.Velocity[1])-meanV) > stdDevThreshold*stdDevV {
				isOutlier = true
			}
		}

		// 6. Add to inliers list
		if !isOutlier {
			inliers = append(inliers, targetVec)
		}
	}
	return inliers
}

// calculateVelocityStats computes the mean and sample standard deviation
func calculateVelocityStats(vectors []motionVector) (meanU, stdDevU, meanV, stdDevV float64) {
	n := float64(len(vectors))
	if n == 0 {
		return 0, 0, 0, 0
	}

	var sumU, sumV float64
	for _, v := range vectors {
		sumU += float64(v.Velocity[0]) // U
		sumV += float64(v.Velocity[1]) // V
	}
	meanU = sumU / n
	meanV = sumV / n

	if n == 1 {
		// Cannot calculate std dev with one sample
		return meanU, 0, meanV, 0
	}

	var sumSqDiffU, sumSqDiffV float64
	for _, v := range vectors {
		sumSqDiffU += math.Pow(float64(v.Velocity[0])-meanU, 2)
		sumSqDiffV += math.Pow(float64(v.Velocity[1])-meanV, 2)
	}

	// Use (n-1) for sample standard deviation
	stdDevU = math.Sqrt(sumSqDiffU / (n - 1))
	stdDevV = math.Sqrt(sumSqDiffV / (n - 1))

	return meanU, stdDevU, meanV, stdDevV
}

// euclideanDistance calculates the spatial distance between two points.
func euclideanDistance(p1, p2 [2]float32) float64 {
	dx := p1[0] - p2[0]
	dy := p1[1] - p2[1]
	return math.Sqrt(float64(dx*dx + dy*dy))
}

// --- Filter 2: Spatial Declustering (Grid Median) ---

// declusterVectors takes a list of vectors and replaces dense clusters
func declusterVectors(vectors []motionVector, gridCellSize int, minSamplesInCell int) []motionVector {
	// 1. Bin vectors into grid cells
	// map[grid_cell_coord] -> list_of_vectors_in_that_cell
	grid := make(map[image.Point][]motionVector)

	if gridCellSize <= 0 {
		gridCellSize = 1 // Avoid division by zero
	}

	for _, v := range vectors {
		cellX := int(math.Floor(float64(v.Point[0]) / float64(gridCellSize))) // Point[0] is X
		cellY := int(math.Floor(float64(v.Point[1]) / float64(gridCellSize))) // Point[1] is Y
		cell := image.Point{X: cellX, Y: cellY}
		grid[cell] = append(grid[cell], v)
	}

	// 2. Iterate over cells, calculate median, and create new vector list
	var declusteredVectors []motionVector

	for cell, vectorsInCell := range grid {
		// 3. Only keep cells with enough samples
		if len(vectorsInCell) >= minSamplesInCell {
			// 4. Calculate the median velocity for the cell
			medianVelocity := calculateMedianVelocity(vectorsInCell)

			// 5. Create a new vector at the center of the grid cell
			newVec := motionVector{
				Point: [2]float32{
					float32(cell.X*gridCellSize) + float32(gridCellSize)/2.0, // X
					float32(cell.Y*gridCellSize) + float32(gridCellSize)/2.0, // Y
				},
				Velocity: medianVelocity,
			}
			declusteredVectors = append(declusteredVectors, newVec)
		}
	}

	return declusteredVectors
}

// calculateMedianVelocity finds the median U and median V for a slice of vectors.
func calculateMedianVelocity(vectors []motionVector) [2]float32 {
	n := len(vectors)
	if n == 0 {
		return [2]float32{0, 0}
	}

	uValues := make([]float64, n)
	vValues := make([]float64, n)
	for i, v := range vectors {
		uValues[i] = float64(v.Velocity[0]) // U
		vValues[i] = float64(v.Velocity[1]) // V
	}

	sort.Float64s(uValues)
	sort.Float64s(vValues)

	medianU := findMedian(uValues)
	medianV := findMedian(vValues)

	return [2]float32{float32(medianU), float32(medianV)}
}

// findMedian is a helper to find the median of a *sorted* slice of float64s.
func findMedian(sortedValues []float64) float64 {
	n := len(sortedValues)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		// Odd number of elements
		return sortedValues[n/2]
	}
	// Even number of elements
	mid1 := sortedValues[n/2-1]
	mid2 := sortedValues[n/2]
	return (mid1 + mid2) / 2.0
}
