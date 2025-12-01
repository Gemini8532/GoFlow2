package newcast

import (
	"math"
)

func NewAverageFlowGrid(width, height int) *AverageFlowGrid {
	return &AverageFlowGrid{
		Width:  width,
		Height: height,
		Data:   make([]Vector, width*height),
	}
}

func (g *AverageFlowGrid) Set(x, y int, v Vector) {
	if x >= 0 && x < g.Width && y >= 0 && y < g.Height {
		g.Data[y*g.Width+x] = v
	}
}

func (g *AverageFlowGrid) Get(x, y int) Vector {
	if x >= 0 && x < g.Width && y >= 0 && y < g.Height {
		return g.Data[y*g.Width+x]
	}
	return Vector{}
}

// Resize creates a new AverageFlowGrid with the given dimensions and resizes the original
// vector data into it using bilinear interpolation.
func (g *AverageFlowGrid) Resize(newWidth, newHeight int) *AverageFlowGrid {
	if newWidth <= 0 || newHeight <= 0 {
		// Return an empty grid or handle error as appropriate
		return NewAverageFlowGrid(0, 0)
	}

	newGrid := NewAverageFlowGrid(newWidth, newHeight)
	xRatio := float64(g.Width-1) / float64(newWidth-1)
	yRatio := float64(g.Height-1) / float64(newHeight-1)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Find the corresponding position in the original grid
			origX := float64(x) * xRatio
			origY := float64(y) * yRatio

			// Get the integer coordinates of the surrounding pixels
			x0 := int(math.Floor(origX))
			y0 := int(math.Floor(origY))
			x1 := x0 + 1
			y1 := y0 + 1

			// Ensure coordinates are within bounds
			if x1 >= g.Width {
				x1 = g.Width - 1
			}
			if y1 >= g.Height {
				y1 = g.Height - 1
			}
			if x0 < 0 {
				x0 = 0
			}
			if y0 < 0 {
				y0 = 0
			}

			// Get the vectors of the surrounding pixels
			v00 := g.Get(x0, y0)
			v10 := g.Get(x1, y0)
			v01 := g.Get(x0, y1)
			v11 := g.Get(x1, y1)

			// Calculate interpolation weights
			dx := origX - float64(x0)
			dy := origY - float64(y0)

			// Interpolate Vx and Vy
			vx := v00.Vx*(1-dx)*(1-dy) + v10.Vx*dx*(1-dy) + v01.Vx*(1-dx)*dy + v11.Vx*dx*dy
			vy := v00.Vy*(1-dx)*(1-dy) + v10.Vy*dx*(1-dy) + v01.Vy*(1-dx)*dy + v11.Vy*dx*dy

			newGrid.Set(x, y, Vector{Vx: vx, Vy: vy})
		}
	}

	return newGrid
}
