package newcast

import (
	"math"
)

// Track represents a sequence of points over time.
type LTrack []Point

// Vector represents a 2D velocity vector.
type Vector struct {
	Vx, Vy float64
}

// FlowGrid is the clean, lightweight result structure.
// It contains only the final vector field data.
type FlowGrid struct {
	Width  int
	Height int
	Time   int
	Data   []Vector // Flat slice: index = (t * height + y) * width + x
}

// buildCell holds the temporary accumulation state.
// This is internal to the processor and not exported with the final grid.
type buildCell struct {
	SumVx       float64
	SumVy       float64
	TotalWeight float64
	IsSet       bool
}

// FlowProcessor manages the construction of a FlowGrid.
// It holds the temporary build data and the active set optimization.
type FlowProcessor struct {
	Grid *FlowGrid

	// buildData is a parallel array to Grid.Data used for accumulation and state tracking.
	buildData []buildCell

	// activeIndices tracks touched cells for sparse processing optimization.
	activeIndices map[int]bool
}

// NewFlowProcessor initializes the processor and the underlying result grid.
func NewFlowProcessor(width, height, time int) *FlowProcessor {
	totalCells := width * height * time
	return &FlowProcessor{
		Grid: &FlowGrid{
			Width:  width,
			Height: height,
			Time:   time,
			Data:   make([]Vector, totalCells),
		},
		buildData:     make([]buildCell, totalCells),
		activeIndices: make(map[int]bool),
	}
}

// getIndex calculates the flat index.
func (fp *FlowProcessor) getIndex(x, y, t int) int {
	return (t*fp.Grid.Height+y)*fp.Grid.Width + x
}

// getCoords recovers x, y, t from a flat index.
func (fp *FlowProcessor) getCoords(idx int) (int, int, int) {
	area := fp.Grid.Width * fp.Grid.Height
	t := idx / area
	rem := idx % area
	y := rem / fp.Grid.Width
	x := rem % fp.Grid.Width
	return x, y, t
}

// AddVector adds a vector to the build buffer using bilinear weighting.
func (fp *FlowProcessor) AddVector(t int, x, y, vx, vy float64) {
	if t < 0 || t >= fp.Grid.Time {
		return
	}

	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1

	dx := x - float64(x0)
	dy := y - float64(y0)

	w00 := (1 - dx) * (1 - dy)
	w10 := dx * (1 - dy)
	w01 := (1 - dx) * dy
	w11 := dx * dy

	fp.accumulateCell(t, x0, y0, vx, vy, w00)
	fp.accumulateCell(t, x1, y0, vx, vy, w10)
	fp.accumulateCell(t, x0, y1, vx, vy, w01)
	fp.accumulateCell(t, x1, y1, vx, vy, w11)
}

func (fp *FlowProcessor) accumulateCell(t, x, y int, vx, vy, w float64) {
	if x < 0 || x >= fp.Grid.Width || y < 0 || y >= fp.Grid.Height {
		return
	}

	idx := fp.getIndex(x, y, t)

	// Update Build Data
	fp.buildData[idx].SumVx += vx * w
	fp.buildData[idx].SumVy += vy * w
	fp.buildData[idx].TotalWeight += w

	// Mark as active for sparse processing
	fp.activeIndices[idx] = true
}

// ProcessTracks Phase 1: Accumulation.
func (fp *FlowProcessor) ProcessTracks(tracks []Track) {

	for _, ntrack := range tracks {
		track := ntrack.Points
		for t := 0; t < len(track)-1; t++ {
			if t >= fp.Grid.Time {
				break
			}
			curr := track[t]
			next := track[t+1]
			vx := next.X - curr.X
			vy := next.Y - curr.Y
			fp.AddVector(t, curr.X, curr.Y, vx, vy)
		}
	}
}

// CalculateAverages Phase 2: Normalize sums into the result Grid.
func (fp *FlowProcessor) CalculateAverages() {
	for idx := range fp.activeIndices {
		build := &fp.buildData[idx]
		if build.TotalWeight > 0.0001 {
			// Write final result to the Clean Grid
			fp.Grid.Data[idx] = Vector{
				Vx: build.SumVx / build.TotalWeight,
				Vy: build.SumVy / build.TotalWeight,
			}
			build.IsSet = true
		}
	}
}

type queueItem struct {
	idx   int
	depth int
}

// FillGaps Phase 3: Interpolation.
// Reads from Grid.Data/buildData.IsSet, Writes to Grid.Data.
func (fp *FlowProcessor) FillGaps(maxDist int) {
	if maxDist <= 0 {
		return
	}

	queue := make([]queueItem, 0, len(fp.activeIndices)*4)
	inQueue := make(map[int]bool)

	// Seed from active indices
	for idx := range fp.activeIndices {
		if fp.buildData[idx].IsSet {
			x, y, t := fp.getCoords(idx)
			fp.enqueueEmptyNeighbors(t, x, y, 1, maxDist, &queue, inQueue)
		}
	}

	head := 0
	for head < len(queue) {
		item := queue[head]
		head++

		idx := item.idx
		x, y, t := fp.getCoords(idx)

		// Check neighbors (read from Grid.Data, check IsSet via buildData)
		vx, vy, count := fp.getNeighborAverage(t, x, y)

		if count > 0 {
			// Write to Result Grid
			fp.Grid.Data[idx] = Vector{Vx: vx, Vy: vy}

			// Update Build State (so this cell can help fill others)
			fp.buildData[idx].IsSet = true
			// We don't need to update Sum/Weight or ActiveIndices here
			// because IsSet is the only flag used for the rest of FillGaps.

			if item.depth < maxDist {
				fp.enqueueEmptyNeighbors(t, x, y, item.depth+1, maxDist, &queue, inQueue)
			}
		}
	}
}

func (fp *FlowProcessor) enqueueEmptyNeighbors(t, cx, cy, depth, maxDist int, queue *[]queueItem, inQueue map[int]bool) {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := cx+dx, cy+dy

			if nx >= 0 && nx < fp.Grid.Width && ny >= 0 && ny < fp.Grid.Height {
				idx := fp.getIndex(nx, ny, t)

				// Check validity using buildData, queue uniqueness using inQueue
				if !fp.buildData[idx].IsSet && !inQueue[idx] {
					inQueue[idx] = true
					*queue = append(*queue, queueItem{idx: idx, depth: depth})
				}
			}
		}
	}
}

func (fp *FlowProcessor) getNeighborAverage(t, cx, cy int) (float64, float64, int) {
	var sumVx, sumVy float64
	var count int

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := cx+dx, cy+dy

			if nx >= 0 && nx < fp.Grid.Width && ny >= 0 && ny < fp.Grid.Height {
				idx := fp.getIndex(nx, ny, t)
				// Check validity in buildData, read value from Grid.Data
				if fp.buildData[idx].IsSet {
					vec := fp.Grid.Data[idx]
					sumVx += vec.Vx
					sumVy += vec.Vy
					count++
				}
			}
		}
	}

	if count == 0 {
		return 0, 0, 0
	}
	return sumVx / float64(count), sumVy / float64(count), count
}
