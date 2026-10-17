package gridoutput

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/airbusgeo/godal"
	"github.com/skysparq/grib2-go/grid"
	gridtoisobands "github.com/skysparq/grid-to-isobands"
)

// realHRRRGrid mirrors grib2-go's own Template30 test fixture (a real HRRR CONUS grid
// definition), so the geotransform this derives can be checked against known values (3km cells)
// instead of an arbitrary synthetic grid.
func realHRRRGrid() grid.Template30 {
	return grid.Template30{
		EarthShape:                  6,
		Nx:                          1799,
		Ny:                          1059,
		La1:                         21138123,
		Lo1:                         237280472,
		ResolutionAndComponentFlags: 8,
		LaD:                         38500000,
		LoV:                         262500000,
		Dx:                          3000000,
		Dy:                          3000000,
		ScanningMode:                64,
		Latin1:                      38500000,
		Latin2:                      38500000,
	}
}

// realGFSGrid mirrors grib2-go's own Template0 test fixture (a real GFS 0.25deg global grid).
func realGFSGrid() grid.Template0 {
	return grid.Template0{
		EarthShape:                  6,
		PointsAlongParallel:         1440,
		PointsAlongMeridian:         721,
		Subdivisions:                -1,
		FirstLatitude:               90000000,
		FirstLongitude:              0,
		ResolutionAndComponentFlags: 48,
		LastLatitude:                -90000000,
		LastLongitude:               359750000,
		ParallelIncrement:           250000,
		MeridianIncrement:           250000,
		ScanningMode:                0,
	}
}

func TestGeotransformFromCornersLambertConformal(t *testing.T) {
	tmpl := realHRRRGrid()
	points, err := tmpl.Points()
	if err != nil {
		t.Fatal(err)
	}
	width := tmpl.XVals()

	wkt, err := tmpl.SrsWkt()
	if err != nil {
		t.Fatalf("SrsWkt: %v", err)
	}
	sr, err := godal.NewSpatialRefFromWKT(wkt)
	if err != nil {
		t.Fatal(err)
	}
	defer sr.Close()

	gt, err := geotransformFromCorners(sr, points.Lngs, points.Lats, width)
	if err != nil {
		t.Fatalf("geotransformFromCorners: %v", err)
	}

	// HRRR's native cell size is 3000m - the analytically-derived spacing should match to
	// well under a millimeter, since it's read off an exact reprojection, not estimated.
	const wantSpacing = 3000.0
	if math.Abs(math.Abs(gt[1])-wantSpacing) > 1e-6 {
		t.Errorf("pixel width = %v, want ~%v", gt[1], wantSpacing)
	}
	if math.Abs(math.Abs(gt[5])-wantSpacing) > 1e-6 {
		t.Errorf("pixel height = %v, want ~%v", gt[5], wantSpacing)
	}

	assertOriginRoundTrips(t, sr, gt, points.Lngs[0], points.Lats[0])
}

func TestGeotransformFromCornersLatLon(t *testing.T) {
	tmpl := realGFSGrid()
	points, err := tmpl.Points()
	if err != nil {
		t.Fatal(err)
	}
	width := tmpl.XVals()

	wkt, err := tmpl.SrsWkt()
	if err != nil {
		t.Fatalf("SrsWkt: %v", err)
	}
	sr, err := godal.NewSpatialRefFromWKT(wkt)
	if err != nil {
		t.Fatal(err)
	}
	defer sr.Close()

	gt, err := geotransformFromCorners(sr, points.Lngs, points.Lats, width)
	if err != nil {
		t.Fatalf("geotransformFromCorners: %v", err)
	}

	// a lat/lon grid's reprojection into its own native (geographic) CRS is an identity
	// transform, so the pixel spacing should come back as exactly the 0.25deg grid resolution.
	const wantSpacing = 0.25
	if math.Abs(gt[1]-wantSpacing) > 1e-9 {
		t.Errorf("pixel width = %v, want %v", gt[1], wantSpacing)
	}
	if math.Abs(gt[5]-(-wantSpacing)) > 1e-9 {
		t.Errorf("pixel height = %v, want %v", gt[5], -wantSpacing)
	}
	if gt[0] != points.Lngs[0] || gt[3] != points.Lats[0] {
		t.Errorf("origin = (%v, %v), want (%v, %v)", gt[0], gt[3], points.Lngs[0], points.Lats[0])
	}
}

func TestGeotransformFromCornersRejectsMismatchedSRS(t *testing.T) {
	// Deliberately reproject a curvilinear Lambert Conformal grid's points as though they
	// belonged to a *different* Lambert Conformal grid (shifted meridian). The result is not
	// axis-aligned in that CRS, so this must error rather than silently write a misregistered
	// file - this is the same failure mode that would occur if a caller ever wired up the wrong
	// grid definition's SRS for a given record.
	tmpl := realHRRRGrid()
	points, err := tmpl.Points()
	if err != nil {
		t.Fatal(err)
	}

	wrong := tmpl
	wrong.LoV = tmpl.LoV - 20000000 // shift central meridian by 20 degrees
	wrongWKT, err := wrong.SrsWkt()
	if err != nil {
		t.Fatal(err)
	}
	wrongSR, err := godal.NewSpatialRefFromWKT(wrongWKT)
	if err != nil {
		t.Fatal(err)
	}
	defer wrongSR.Close()

	if _, err := geotransformFromCorners(wrongSR, points.Lngs, points.Lats, tmpl.XVals()); err == nil {
		t.Fatal("expected an error for a grid that is not axis-aligned in the given spatial ref, got nil")
	}
}

func TestGridToGeoTiffLambertConformal(t *testing.T) {
	tmpl := realHRRRGrid()
	points, err := tmpl.Points()
	if err != nil {
		t.Fatal(err)
	}
	width, height := tmpl.XVals(), tmpl.YVals()

	wkt, err := tmpl.SrsWkt()
	if err != nil {
		t.Fatal(err)
	}

	values := make([]float64, len(points.Lats))
	for i := range values {
		values[i] = float64(i % 100)
	}
	g := &gridtoisobands.GridValues{
		SizeX:  width,
		SizeY:  height,
		Lons:   points.Lngs,
		Lats:   points.Lats,
		Values: values,
	}

	workDir := t.TempDir()
	raw, err := EncodeGeoTiff(g, wkt, workDir)
	if err != nil {
		t.Fatalf("EncodeGeoTiff: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("EncodeGeoTiff returned no bytes")
	}

	// no stray temp files should be left behind in the work dir.
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected workDir to be empty after EncodeGeoTiff, found: %v", entries)
	}

	outPath := filepath.Join(workDir, "out.tif")
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	ds, err := godal.Open(outPath)
	if err != nil {
		t.Fatalf("reopening output COG: %v", err)
	}
	defer func() { _ = ds.Close() }()

	if ds.Structure().SizeX != width || ds.Structure().SizeY != height {
		t.Errorf("output size = %vx%v, want %vx%v", ds.Structure().SizeX, ds.Structure().SizeY, width, height)
	}
	sr, err := godal.NewSpatialRefFromWKT(wkt)
	if err != nil {
		t.Fatal(err)
	}
	defer sr.Close()
	dsGT, err := ds.GeoTransform()
	if err != nil {
		t.Fatal(err)
	}
	assertOriginRoundTrips(t, sr, dsGT, points.Lngs[0], points.Lats[0])
}

// assertOriginRoundTrips reprojects a geotransform's origin back to plain geographic degrees
// and checks it lands on the expected lon/lat - confirming there's no drift or misregistration
// introduced anywhere between the source points and the written geotransform.
func assertOriginRoundTrips(t *testing.T, sr *godal.SpatialRef, gt [6]float64, wantLon, wantLat float64) {
	t.Helper()
	geoSR, err := godal.NewSpatialRefFromEPSG(4326)
	if err != nil {
		t.Fatal(err)
	}
	defer geoSR.Close()
	trn, err := godal.NewTransform(sr, geoSR)
	if err != nil {
		t.Fatal(err)
	}
	defer trn.Close()

	x := []float64{gt[0]}
	y := []float64{gt[3]}
	if err := trn.TransformEx(x, y, nil, nil); err != nil {
		t.Fatal(err)
	}
	if math.Abs(x[0]-wantLon) > 0.001 || math.Abs(y[0]-wantLat) > 0.001 {
		t.Errorf("origin round-trip = (lon=%v, lat=%v), want (lon=%v, lat=%v)", x[0], y[0], wantLon, wantLat)
	}
}
