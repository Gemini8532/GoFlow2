# Cleanup Execution Checklist

## Pre-Cleanup
- [ ] Commit all current changes
- [ ] Create cleanup branch: `git checkout -b cleanup/remove-old-pipeline`
- [ ] Run tests to establish baseline: `go test ./...`
- [ ] Tag current state: `git tag pre-cleanup`

## Phase 1: Update API & CLI

### Update API Server (cmd/serve/main.go)

#### Remove FlowGrid Support
- [ ] Remove `FlowGrid *newcast.FlowGrid` from `RequestData` struct (line 25)
- [ ] Remove `gridType` parameter from request struct (line 62)
- [ ] Remove entire `vectorFrameHandler` function (lines 146-236)
- [ ] Remove `/vector-frame` route registration (line 373)
- [ ] Simplify `processHandler`:
  - [ ] Remove `switch req.GridType` block (lines 112-135)
  - [ ] Always call `GenerateFlowGridFromTracks` (currently in case "average")
  - [ ] Remove FlowGrid assignment (lines 128-134)

### Update CLI Tool (cmd/cast/main.go)

#### Remove Old Filter Flags
- [ ] Remove `smoothness` flag (line 21)
- [ ] Remove `filterType` flag (line 23)
- [ ] Remove `maxAngle` flag (line 24)
- [ ] Remove `gridCellSize` flag (line 25)
- [ ] Remove `minTracksPerCell` flag (line 26)
- [ ] Remove `maxTracksPerCell` flag (line 27)
- [ ] Remove `gridType` flag (line 32)
- [ ] Keep curvefit flags: `minRSquared`, `maxRMSE`, `maxDeviation`, `maxAcceleration`

#### Update ProcessConfig Construction
- [ ] Remove old fields from config struct (lines 51-66):
  ```go
  // Remove:
  Smoothness:       *smoothness,
  FilterType:       *filterType,
  MaxAngle:         *maxAngle,
  GridCellSize:     *gridCellSize,
  MinTracksPerCell: *minTracksPerCell,
  MaxTracksPerCell: *maxTracksPerCell,
  ```

#### Update processLocally()
- [ ] Remove filter type logging (lines 80-89)
- [ ] Update to log curvefit parameters instead

#### Update processRemotely()
- [ ] Remove `gridType` parameter from function signature (line 129)
- [ ] Remove `gridType` from request body (line 151)
- [ ] Simplify output message (lines 177-186):
  - [ ] Remove switch on gridType
  - [ ] Only show curl command for `/average-flow-grid`

### Verify
- [ ] Run: `go build ./cmd/serve`
- [ ] Run: `go build ./cmd/cast`
- [ ] Test: API still starts
- [ ] Test: `/process` endpoint works
- [ ] Test: `/average-flow-grid` endpoint works
- [ ] Test: CLI tool works with simplified flags

## Phase 2: Remove Old Grid Code

### Delete Files
- [ ] `rm newcast/flow_grid.go`
- [ ] `rm newcast/flow_grid_test.go`
- [ ] `rm newcast/track_average.go`
- [ ] `rm newcast/track_average_test.go`

### Verify
- [ ] Run: `go test ./newcast`
- [ ] Should pass (these files not referenced)

## Phase 3: Simplify Processing Pipeline

### Update process.go
- [ ] Replace switch statement (lines 70-92) with curvefit-only code
- [ ] Remove unused ProcessConfig fields:
  ```go
  // Remove these from struct:
  Smoothness       float64
  FilterType       string
  MaxAngle         float64
  GridCellSize     int
  MinTracksPerCell int
  MaxTracksPerCell int
  ```

### Update Defaults
- [ ] Update default configs in `cmd/serve/main.go` (lines 78-87)
- [ ] Use recommended curvefit parameters:
  ```go
  config := newcast.ProcessConfig{
      MaxFeatures:      200,
      MinTrackLength:   6,
      MinRSquared:      0.90,
      MaxRMSE:          3.0,
      MaxDeviation:     7.0,
      MaxAcceleration:  1.5,
      BlurSigma:        0.0,
      UseFittedPoints:  false,
  }
  ```

### Delete filter.go
- [ ] `rm newcast/filter.go`

### Update Tests
- [ ] Update `curvefit_filter_test.go`:
  - [ ] Remove FilterTracksBySmoothness comparison (line 51-53)
  - [ ] Remove smoothnessTracks variable
  - [ ] Remove smoothness analysis (lines 64-66)
- [ ] Run: `go test ./newcast`
- [ ] Fix any compilation errors

### Verify
- [ ] Run: `go test ./...`
- [ ] All tests pass
- [ ] Run: `go build ./...`
- [ ] No compilation errors

## Phase 4: Cleanup

### Delete Backup Files
- [ ] `rm newcast/grid_encode.go.bkp`

### Update Documentation
- [ ] Update README.md (if exists)
- [ ] Update API documentation
- [ ] Note breaking changes

### Verify
- [ ] Run: `go test ./...`
- [ ] Run: `go build ./...`
- [ ] Manual test with flowvis frontend

## Post-Cleanup

### Code Review
- [ ] Review all changes: `git diff main`
- [ ] Check for any remaining references to removed code
- [ ] Verify test coverage maintained

### Testing
- [ ] Unit tests: `go test ./...`
- [ ] Integration test with real rainfall data
- [ ] API test with curl/postman
- [ ] Frontend test with flowvis

### Finalize
- [ ] Commit all changes: `git commit -m "Remove old pipeline code"`
- [ ] Create PR: `git push origin cleanup/remove-old-pipeline`
- [ ] Review with team
- [ ] Merge to main

## Rollback (if needed)
- [ ] `git checkout main`
- [ ] `git branch -D cleanup/remove-old-pipeline`
- [ ] Or: Cherry-pick specific commits to keep

## Success Metrics
- [ ] ✅ ~1,000 lines of code removed
- [ ] ✅ All tests passing
- [ ] ✅ API functional
- [ ] ✅ No regressions in vector field quality
- [ ] ✅ Simpler configuration
- [ ] ✅ Documentation updated

## Notes
- Keep git commits atomic (one phase per commit)
- Run tests after each phase
- Document any unexpected issues
- Backup before each major change

## Commands Reference

```bash
# Create backup branch
git checkout -b cleanup/remove-old-pipeline

# Run tests
go test ./...
go test ./newcast -v

# Build everything
go build ./...
go build ./cmd/serve

# Remove files
rm newcast/flow_grid.go newcast/flow_grid_test.go
rm newcast/track_average.go newcast/track_average_test.go
rm newcast/filter.go
rm newcast/grid_encode.go.bkp

# Check for references
grep -r "FlowGrid" .
grep -r "FilterTracksBySmoothness" .
grep -r "CalculateAveragedTracks" .
grep -r "gridType" cmd/
grep -r "filterType" cmd/
grep -r '"smoothness"' cmd/
grep -r '"density"' cmd/
grep -r '"max_angle"' cmd/

# Commit
git add -A
git commit -m "Phase X: [description]"
```
