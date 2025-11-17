package main

import (
	"example/goflow/sgfilter"
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// CreatePlots creates a plot with clean circle, noisy points, and filtered results
// Each line now includes small circles of the same color as the line
func CreatePlots(clean, original, filtered []sgfilter.Point2D, filterName string) error {
	// Create a new plot
	p := plot.New()

	p.Title.Text = "Smoothing Comparison"
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"

	// Convert data to plotter.XYs format
	cleanXYs := make(plotter.XYs, len(clean))
	origXYs := make(plotter.XYs, len(original))
	filteredXYs := make(plotter.XYs, len(filtered))

	for i, pt := range clean {
		cleanXYs[i].X = pt.X
		cleanXYs[i].Y = pt.Y
	}
	for i, pt := range original {
		origXYs[i].X = pt.X
		origXYs[i].Y = pt.Y
	}
	for i, pt := range filtered {
		filteredXYs[i].X = pt.X
		filteredXYs[i].Y = pt.Y
	}

	// Create line plots with labels for legend
	cleanLine, err := plotter.NewLine(cleanXYs)
	if err != nil {
		return err
	}
	cleanLine.Color = plotutil.Color(4) // Different color for clean circle
	cleanLine.Width = vg.Points(1)
	cleanLine.Dashes = []vg.Length{vg.Points(5), vg.Points(5)} // Dashed line for clean circle
	
	// Create scatter plot for clean points with same color as line
	cleanScatter, err := plotter.NewScatter(cleanXYs)
	if err != nil {
		return err
	}
	cleanScatter.Color = cleanLine.Color
	cleanScatter.GlyphStyle.Radius = vg.Points(2)
	p.Legend.Add("Clean Circle", cleanLine)

	origLine, err := plotter.NewLine(origXYs)
	if err != nil {
		return err
	}
	origLine.Color = plotutil.Color(0)
	origLine.Width = vg.Points(1)
	
	// Create scatter plot for original points with same color as line
	origScatter, err := plotter.NewScatter(origXYs)
	if err != nil {
		return err
	}
	origScatter.Color = origLine.Color
	origScatter.GlyphStyle.Radius = vg.Points(2)
	p.Legend.Add("Noisy Points", origLine)

	filteredLine, err := plotter.NewLine(filteredXYs)
	if err != nil {
		return err
	}
	filteredLine.Color = plotutil.Color(1)
	filteredLine.Width = vg.Points(2)
	
	// Create scatter plot for filtered points with same color as line
	filteredScatter, err := plotter.NewScatter(filteredXYs)
	if err != nil {
		return err
	}
	filteredScatter.Color = filteredLine.Color
	filteredScatter.GlyphStyle.Radius = vg.Points(2)
	p.Legend.Add(filterName, filteredLine)

	p.Add(cleanLine, cleanScatter, origLine, origScatter, filteredLine, filteredScatter)
	p.Legend.Top = true

	// Determine the maximum range to set consistent axis scales for comparison
	// Find the maximum absolute value across all datasets to determine appropriate scale
	maxAbs := 0.0
	for _, dataset := range [][]sgfilter.Point2D{clean, original, filtered} {
		for _, pt := range dataset {
			if abs := math.Abs(pt.X); abs > maxAbs {
				maxAbs = abs
			}
			if abs := math.Abs(pt.Y); abs > maxAbs {
				maxAbs = abs
			}
		}
	}
	
	// Add a margin to accommodate potential noise and ensure full visibility
	margin := 1.0
	scale := math.Ceil(maxAbs + margin)
	
	p.X.Min = -scale
	p.X.Max = scale
	p.Y.Min = -scale
	p.Y.Max = scale

	// Save plot to file
	return p.Save(8*vg.Inch, 6*vg.Inch, "sgfilter_comparison.png")
}