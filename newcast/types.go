package newcast

// Vector represents a 2D velocity vector.
type Vector struct {
	Vx, Vy float64
}

// AverageFlowGrid represents a single grid of averaged vectors.
type AverageFlowGrid struct {
	Width  int
	Height int
	Data   []Vector
}