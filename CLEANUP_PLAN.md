# Code Cleanup Plan: Remove Unused Processing Paths

## Overview

The codebase has evolved to use a new processing pipeline based on:
- **curvefit.go** - Polynomial curve fitting for tracks
- **curvefit_filter.go** - Quality-based filtering using curve fit metrics
- **track_direct.go** - Direct track-to-grid processing using all points

The old processing paths using ad-hoc filtering and multi-frame flow grids are no longer needed.

## Current State Analysis

### ✅ KEEP - Modern Pipeline (In Use)

#### Core Processing
1. **newcast.go** - Core tracker implementation ✓
2. **process.go** - ProcessFilesToTracks (supports both old and new filtering) ✓
3. **curvefit.go** - Polynomial fitting for tracks ✓
4. **curvefit_filter.go** - Quality-based filtering ✓
5. **track_direct.go** - Direct grid generation from all track points ✓
6. **fast_smooth.go** - SmoothFill algorithm for gap filling ✓

#### Supporting
7. **grid_encode.go** - Frame/Vector encoding ✓
8. **types.go** - Data structures ✓
9. **visualize.go** - Track visualization ✓

### 🗑️ REMOVE - Old Pipeline (Deprecated)

#### Ad-hoc Filtering Functions
1. **filter.go** - Contains old filtering methods:
   - `FilterTracksBySmoothness()` - USED IN: process.go (lines 72, 91), test files
   - `FilterTracksByDensityAndSmoothness()` - USED IN: process.go (line 73)
   - `FilterTracksByMaxAngleChange()` - USED IN: process.go (line 75)
   - `calculateSmoothnessMetric()` - Helper for above
   - `angleBetween()` - Helper for above

#### Multi-Frame Flow Grid
2. **flow_grid.go** - OLD multi-frame grid generation:
   - `FlowGrid` type - Multi-time-step grid (USED IN: serve/main.go)
   - `FlowProcessor` - Multi-frame processor (USED IN: serve/main.go)
   - `NewFlowProcessor()` - USED IN: serve/main.go (line 130)
   - `ProcessTracks()` - USED IN: serve/main.go (line 131)
   - `CalculateAverages()` - USED IN: serve/main.go (line 132)
   - `FillGaps()` - USED IN: serve/main.go (line 133)

#### Old Averaging Approach
3. **track_average.go** - Pre-averaging tracks (midpoint-based):
   - `AveragedTrack` type - NOT USED
   - `CalculateAveragedTracks()` - NOT USED
   - `GenerateAverageFlowGrid()` - NOT USED (replaced by GenerateFlowGridFromTracks)

### 📦 Dependencies to Update

#### cmd/serve/main.go (API Server)
Currently supports TWO grid types:
- `gridType: "average"` → Uses **track_direct.go** (KEEP) ✓
- `gridType: "flow"` → Uses **flow_grid.go** (REMOVE) ✗

**Status**: Flow grid code path is still active in the API

#### cmd/cast/main.go (CLI Tool)
Currently supports:
- **Four filter types**: smoothness, density, max_angle, curvefit
- **Two grid types**: flow (multi-frame), average
- **Remote processing**: Sends gridType and filterType to server

**Status**: All old filtering options and multi-frame grid support need removal

#### ProcessConfig (process.go)
Currently supports FOUR filter types:
- `filterType: "density"` → Uses filter.go functions ✗
- `filterType: "max_angle"` → Uses filter.go functions ✗
- `filterType: "curvefit"` → Uses curvefit_filter.go ✓
- `filterType: "smoothness"` (default) → Uses filter.go functions ✗

**Status**: Only "curvefit" filtering should remain

## Removal Plan

### Phase 1: Update API (Breaking Change)

**Goal**: Remove multi-frame flow grid support from API

#### Step 1.1: Simplify serve/main.go
- Remove `FlowGrid` from `RequestData` struct
- Remove `gridType` parameter (only support average grid)
- Remove `vectorFrameHandler` (served multi-frame grids)
- Update `processHandler` to always use track_direct.go
- Keep `averageFlowGridHandler` (uses track_direct.go)

**Files to modify:**
- `cmd/serve/main.go`
- `cmd/cast/main.go`

**API Changes (serve):**
- Remove endpoint: `/vector-frame` (was used for multi-frame access)
- Keep endpoint: `/average-flow-grid` (current, uses track_direct.go)
- Simplify `/process` request (no more gridType parameter)

**CLI Changes (cast):**
- Remove `-filterType` flag (smoothness, density, max_angle)
- Remove `-smoothness`, `-gridCellSize`, `-minTracksPerCell`, `-maxTracksPerCell`, `-maxAngle` flags
- Remove `-gridType` flag (always use average/direct method)
- Keep curvefit flags: `-minRSquared`, `-maxRMSE`, `-maxDeviation`, `-maxAcceleration`
- Update `processRemotely()` to not send gridType

#### Step 1.2: Update Frontend (if applicable)
- Remove code that uses `/vector-frame` endpoint
- Remove code that passes `gridType` parameter

### Phase 2: Remove Old Grid Code

**Goal**: Delete multi-frame grid generation

#### Step 2.1: Remove flow_grid.go
```bash
rm newcast/flow_grid.go
rm newcast/flow_grid_test.go
```

**Impact**: 
- Removes ~232 lines (flow_grid.go)
- Removes ~130 lines (flow_grid_test.go)

#### Step 2.2: Remove track_average.go
```bash
rm newcast/track_average.go
rm newcast/track_average_test.go
```

**Impact**:
- Removes ~191 lines (track_average.go)
- Removes ~115 lines (track_average_test.go)

### Phase 3: Simplify Process Pipeline

**Goal**: Remove ad-hoc filtering, keep only curvefit

#### Step 3.1: Update process.go
Remove support for old filter types:

```go
// OLD CODE (lines 69-92):
switch config.FilterType {
case "density":
    smoothTracks := FilterTracksBySmoothness(longTracks, config.Smoothness)
    filteredTracks = FilterTracksByDensityAndSmoothness(smoothTracks, ...)
case "max_angle":
    filteredTracks = FilterTracksByMaxAngleChange(longTracks, config.MaxAngle)
case "curvefit":
    // ... curvefit code ...
default: // "smoothness"
    filteredTracks = FilterTracksBySmoothness(longTracks, config.Smoothness)
}

// NEW CODE:
// Always use curvefit filtering
curveFitConfig := CurveFitConfig{
    MinRSquared:     config.MinRSquared,
    MaxRMSE:         config.MaxRMSE,
    MaxDeviation:    config.MaxDeviation,
    MaxAcceleration: config.MaxAcceleration,
}
// Use defaults if not specified
if curveFitConfig.MinRSquared == 0 {
    curveFitConfig = DefaultCurveFitConfig()
}
filteredTracks = FilterTracksByCurveFit(longTracks, curveFitConfig)
```

#### Step 3.2: Update ProcessConfig
Remove unused fields:

```go
// REMOVE these fields (no longer needed):
type ProcessConfig struct {
    // REMOVE:
    Smoothness       float64  // Was for FilterTracksBySmoothness
    FilterType       string   // No longer needed (always curvefit)
    MaxAngle         float64  // Was for FilterTracksByMaxAngle
    GridCellSize     int      // Was for density filtering
    MinTracksPerCell int      // Was for density filtering
    MaxTracksPerCell int      // Was for density filtering
    
    // KEEP: 
    MaxFeatures      int     // For optical flow tracking
    MinTrackLength   int     // Minimum track length
    BlurSigma        float64 // For track_direct.go smoothing
    UseFittedPoints  bool    // Whether to use polynomial-fitted points
    
    // Curve-fit filtering parameters
    MinRSquared      float64
    MaxRMSE          float64
    MaxDeviation     float64
    MaxAcceleration  float64
}
```

#### Step 3.3: Remove filter.go
```bash
rm newcast/filter.go
```

**Impact**:
- Removes ~150 lines of ad-hoc filtering code
- Removes functions: FilterTracksBySmoothness, FilterTracksByDensityAndSmoothness, FilterTracksByMaxAngleChange

#### Step 3.4: Update Tests
Tests that use old filtering will need updates:
- `curvefit_filter_test.go` - Uses FilterTracksBySmoothness for comparison
- Need to update or remove comparisons with old methods

### Phase 4: Cleanup

### Delete Backup Files
```bash
rm newcast/grid_encode.go.bkp
```

### Update Documentation Guide

### For API Users

**Before:**
```javascript
// Multi-frame grid support
POST /process
{
    "filenames": [...],
    "id": "test",
    "gridType": "flow",  // or "average"
    "config": {
        "filterType": "smoothness",
        "smoothness": 0.5,
        ...
    }
}

// Access frame at time t
GET /vector-frame?id=test&t=0
```

**After:**
```javascript
// Single averaged grid only
POST /process
{
    "filenames": [...],
    "id": "test",
    "config": {
        // Curvefit parameters (optional, uses defaults if omitted)
        "minRSquared": 0.90,
        "maxRMSE": 3.0,
        "maxDeviation": 7.0,
        "maxAcceleration": 1.5,
        ...
    }
}

// Access the averaged flow grid
GET /average-flow-grid?id=test
```

### For CLI Tool Users

**Before:**
```bash
# Old filtering options
./cast -filterType smoothness -smoothness 0.5 file1.png file2.png
./cast -filterType density -gridCellSize 64 -minTracksPerCell 2 file1.png file2.png
./cast -filterType max_angle -maxAngle 0.8 file1.png file2.png

# Multi-frame grid support
./cast -gridType flow -serverURL http://localhost:9093 file1.png file2.png
```

**After:**
```bash
# Curvefit filtering only (with defaults)
./cast file1.png file2.png

# Custom curvefit parameters
./cast -minRSquared 0.90 -maxRMSE 3.0 -maxDeviation 7.0 file1.png file2.png

# Remote processing (always uses average grid)
./cast -serverURL http://localhost:9093 file1.png file2.png
```

### For Code Users

**Before:**
```go
config := ProcessConfig{
    FilterType: "smoothness",
    Smoothness: 0.5,
    GridCellSize: 64,
    MinTracksPerCell: 2,
    MaxTracksPerCell: 5,
}
tracks := ProcessFilesToTracks(files, config)
```

**After:**
```go
config := ProcessConfig{
    MinRSquared:     0.90,
    MaxRMSE:         3.0,
    MaxDeviation:    7.0,
    MaxAcceleration: 1.5,
    // Or omit to use defaults from DefaultCurveFitConfig()
}
tracks := ProcessFilesToTracks(files, config)
```

## Testing Strategy

### Before Removal
1. Run all existing tests to establish baseline
2. Verify current API endpoints work

### During Migration
1. **Phase 1**: Update serve/main.go, test API still works
2. **Phase 2**: Remove flow_grid.go, verify tests still pass
3. **Phase 3**: Update process.go, run all tests
4. **Phase 4**: Cleanup, final test run

### After Removal
1. Run full test suite: `go test ./...`
2. Test API manually with flowvis frontend
3. Verify test coverage hasn't decreased significantly

## Rollback Plan

If issues are discovered:
1. All changes are in version control (git)
2. Can revert specific commits
3. Old code can be temporarily reinstated if needed

## Benefits of Cleanup

### Code Quality
- **Remove ~700+ lines** of unused code
- **Simpler architecture**: One processing path instead of four
- **Easier maintenance**: Fewer code paths to test

### Performance
- No overhead from unused filter methods
- Clearer optimization targets

### Documentation
- Simpler API to document
- Clearer user-facing configuration

## Risks & Mitigation

### Risk 1: Breaking Existing API Clients
**Mitigation**: 
- Version the API (v1 → v2)
- Provide migration guide
- Deprecated but functional v1 endpoint for transition period

### Risk 2: Loss of Functionality
**Mitigation**:
- Multi-frame grids aren't used in practice
- Curvefit filtering is superior to ad-hoc methods
- Keep removed code in git history if needed

### Risk 3: Test Coverage Decrease
**Mitigation**:
- Maintain curvefit parameter tests (newly added)
- Keep integration tests
- Update unit tests to use new filtering

## Timeline

- **Day 1**: Phase 1 (Update API) - 2-3 hours
- **Day 1**: Phase 2 (Remove grid code) - 1 hour
- **Day 2**: Phase 3 (Simplify pipeline) - 2-3 hours
- **Day 2**: Phase 4 (Cleanup) - 30 minutes
- **Day 2-3**: Testing and verification - 2-3 hours

**Total Estimate**: 1-2 days of focused work

## Approval Checklist

- [ ] Review plan with team
- [ ] Confirm no external dependencies on `/vector-frame` endpoint
- [ ] Confirm no external dependencies on old filter types
- [ ] Backup current state in git
- [ ] Create feature branch for cleanup
- [ ] Schedule testing window

## Success Criteria

- [ ] All tests pass
- [ ] API endpoints work with flowvis frontend
- [ ] Code is simpler (measured by line count)
- [ ] No regressions in vector field quality
- [ ] Documentation is updated
