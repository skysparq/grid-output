package gridoutput

import (
	"math"
	"os"
	"testing"

	gridtoisobands "github.com/skysparq/grid-to-isobands"
)

// smallLambertGrid is a 60x40 corner of the HRRR grid: the same projection, small enough to
// contour quickly.
func smallLambertGrid(t *testing.T) (*gridtoisobands.GridValues, string) {
	t.Helper()
	tmpl := realHRRRGrid()
	tmpl.Nx, tmpl.Ny = 60, 40
	points, err := tmpl.Points()
	if err != nil {
		t.Fatal(err)
	}
	wkt, err := tmpl.SrsWkt()
	if err != nil {
		t.Fatal(err)
	}
	values := make([]float64, len(points.Lats))
	for i := range values {
		values[i] = 1000 + float64(i%60)/4 // a gentle east-west gradient, in hPa
	}
	values[5] = math.NaN()
	return &gridtoisobands.GridValues{
		SizeX:  tmpl.XVals(),
		SizeY:  tmpl.YVals(),
		Lons:   points.Lngs,
		Lats:   points.Lats,
		Values: values,
	}, wkt
}

func TestDecodeGeoTiffRoundTrips(t *testing.T) {
	grid, wkt := smallLambertGrid(t)
	workDir := t.TempDir()
	raw, err := EncodeGeoTiff(grid, wkt, workDir)
	if err != nil {
		t.Fatal(err)
	}

	raster, err := DecodeGeoTiff(raw, workDir)
	if err != nil {
		t.Fatal(err)
	}
	if raster.SizeX != grid.SizeX || raster.SizeY != grid.SizeY {
		t.Fatalf(`size = %vx%v, want %vx%v`, raster.SizeX, raster.SizeY, grid.SizeX, grid.SizeY)
	}
	for i, want := range grid.Values {
		got := raster.Values[i]
		if math.IsNaN(want) {
			if !math.IsNaN(got) {
				t.Errorf(`value %d = %v, want NaN`, i, got)
			}
			continue
		}
		if got != float64(float32(want)) { // stored as float32
			t.Errorf(`value %d = %v, want %v`, i, got, want)
		}
	}

	decoded, err := raster.Grid()
	if err != nil {
		t.Fatal(err)
	}
	for i := range grid.Lats {
		if math.Abs(decoded.Lats[i]-grid.Lats[i]) > 1e-6 || math.Abs(decoded.Lons[i]-grid.Lons[i]) > 1e-6 {
			t.Fatalf(`point %d = (lon=%v, lat=%v), want (lon=%v, lat=%v)`, i, decoded.Lons[i], decoded.Lats[i], grid.Lons[i], grid.Lats[i])
		}
	}

	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf(`expected workDir to be empty after decoding, found: %v`, entries)
	}
}

func TestDecodeGeoTiffRejectsGarbage(t *testing.T) {
	if _, err := DecodeGeoTiff([]byte(`not a tiff`), t.TempDir()); err == nil {
		t.Fatal(`expected an error`)
	}
}
