# CurveFit Parameter Test Results

## Quick Reference

### Recommended Configuration
```go
CurveFitConfig{
    MinRSquared:     0.90,
    MaxRMSE:         3.0,
    MaxDeviation:    7.0,
    MaxAcceleration: 1.5,
}
```

### Performance Metrics
- **Retention**: 96.1% (filters only worst 4%)
- **Quality**: R² = 0.995 (excellent)
- **Coverage**: 35.4% of grid cells
- **Smoothness**: 0.600 (good)
- **Consistency**: Perfect across datasets

### Files in this Directory

1. **SUMMARY.md** - Executive summary with recommendations
2. **ANALYSIS.md** - Detailed analysis with statistics
3. **Very_Strict.png** - Visualization (93.5% retention, highest quality)
4. **Strict.png** - Visualization (96.1% retention, **recommended**)
5. **Default.png** - Visualization (96.2% retention, current default)
6. **Moderate.png** - Visualization (97.3% retention)
7. **Relaxed.png** - Visualization (97.8% retention)
8. **Very_Relaxed.png** - Visualization (98.4% retention)
9. **Minimal.png** - Visualization (99.1% retention, lowest filtering)

### Color Coding in Visualizations
- **Green**: High quality (R² ≥ 0.9) - desired
- **Yellow-green**: Medium quality (0.8 ≤ R² < 0.9) - acceptable
- **Yellow/orange**: Lower quality (R² < 0.8) - should be rare

### How to Use

1. **View the visualizations** to see the vector fields
2. **Read SUMMARY.md** for the executive summary
3. **Read ANALYSIS.md** for detailed statistics
4. **Update your code** with the recommended configuration if desired

### Test Commands

```bash
# Run all parameter tests
cd newcast
go test -v -run TestCurveFitParameterSweep
go test -v -run TestCurveFitParameterConsistency
go test -v -run TestRecommendedConfigOnAllData
```
