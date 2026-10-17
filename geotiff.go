package gridoutput

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/airbusgeo/godal"
	"github.com/google/uuid"
	gridtoisobands "github.com/skysparq/grid-to-isobands"
)

func init() {
	godal.RegisterAll()
}

// cornerTolerance bounds how far off-axis a grid may be (as a fraction of its pixel spacing)
// before geotransformFromCorners refuses to trust the result. A grid definition's own template
// (e.g. Template30's Lambert Conformal Conic) guarantees the native grid is exactly rectilinear
// in its own projection, so any drift here means the wrong CRS was built, not real skew.
const cornerTolerance = 0.01

// EncodeGeoTiff encodes a processed model grid as a Cloud Optimized GeoTIFF in its native
// coordinate reference system (srsWKT), with no resampling of the pixel values: grid.Values is
// written through exactly as returned by IsobandsFromGrid.
//
// The geotransform is derived analytically rather than assumed: grid.Lats/Lons are always plain
// geographic degrees regardless of the source projection (see grib2-go's grid.Template0/
// Template30 Points() methods), so three corner points are reprojected into the native CRS via
// GDAL/PROJ to read off the true origin and pixel spacing. For a lat/lon grid this reprojection
// is an identity transform; for a projected grid like HRRR's Lambert Conformal Conic it's the
// exact forward projection GDAL already implements, with no interpolation involved - only the
// GeoTIFF header changes, never the data.
func EncodeGeoTiff(grid *gridtoisobands.GridValues, srsWKT, workDir string) ([]byte, error) {
	width, height := grid.SizeX, grid.SizeY

	sr, err := godal.NewSpatialRefFromWKT(srsWKT)
	if err != nil {
		return nil, fmt.Errorf(`error building geotiff: error parsing native spatial ref: %w`, err)
	}
	defer sr.Close()

	geoTransform, err := geotransformFromCorners(sr, grid.Lons, grid.Lats, width)
	if err != nil {
		return nil, fmt.Errorf(`error building geotiff: %w`, err)
	}

	values := make([]float32, len(grid.Values))
	for i, v := range grid.Values {
		values[i] = float32(v)
	}

	id := uuid.NewString()
	srcPath := filepath.Join(workDir, id+`-src.tif`)
	defer func() { _ = os.Remove(srcPath) }()

	srcDs, err := godal.Create(godal.GTiff, srcPath, 1, godal.Float32, width, height)
	if err != nil {
		return nil, fmt.Errorf(`error building geotiff: error creating source dataset: %w`, err)
	}
	if err := srcDs.Bands()[0].Write(0, 0, values, width, height); err != nil {
		return nil, fmt.Errorf(`error building geotiff: error writing source band: %w`, err)
	}
	if err := srcDs.Bands()[0].SetNoData(math.NaN()); err != nil {
		return nil, fmt.Errorf(`error building geotiff: error setting nodata: %w`, err)
	}
	if err := srcDs.SetGeoTransform(geoTransform); err != nil {
		return nil, fmt.Errorf(`error building geotiff: error setting geotransform: %w`, err)
	}
	if err := srcDs.SetSpatialRef(sr); err != nil {
		return nil, fmt.Errorf(`error building geotiff: error setting spatial ref: %w`, err)
	}
	// grib grid points are cell centers (WMO convention), not corner coordinates - tag the
	// raster accordingly rather than guessing a half-pixel corner shift.
	if err := srcDs.SetMetadata(`AREA_OR_POINT`, `Point`); err != nil {
		return nil, fmt.Errorf(`error building geotiff: error setting AREA_OR_POINT: %w`, err)
	}

	outPath := filepath.Join(workDir, id+`.tif`)
	defer func() { _ = os.Remove(outPath) }()
	cog, err := srcDs.Translate(outPath, []string{
		`-of`, `COG`,
		`-co`, `COMPRESS=DEFLATE`,
	})
	if err != nil {
		_ = srcDs.Close()
		return nil, fmt.Errorf(`error building geotiff: error translating to COG: %w`, err)
	}
	if err := cog.Close(); err != nil {
		return nil, fmt.Errorf(`error building geotiff: error closing COG dataset: %w`, err)
	}
	if err := srcDs.Close(); err != nil {
		return nil, fmt.Errorf(`error building geotiff: error closing source dataset: %w`, err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf(`error building geotiff: error reading COG output: %w`, err)
	}
	return raw, nil
}

// geotransformFromCorners reprojects the grid's [0,0], [0,1], and [1,0] points (in that native
// scan order - whatever transformers have already reordered them into) from plain geographic
// degrees into the dataset's native spatial ref, and reads off the origin and pixel spacing
// implied by them. It errors if the result isn't actually axis-aligned in that CRS, which would
// mean the wrong native spatial ref was built rather than a genuinely skewed grid.
func geotransformFromCorners(dstSR *godal.SpatialRef, lons, lats []float64, width int) ([6]float64, error) {
	var gt [6]float64
	if len(lons) < width+1 || len(lats) < width+1 {
		return gt, fmt.Errorf(`error deriving geotransform: grid is too small to have a second row`)
	}

	srcSR, err := godal.NewSpatialRefFromEPSG(4326)
	if err != nil {
		return gt, fmt.Errorf(`error deriving geotransform: error creating geographic spatial ref: %w`, err)
	}
	defer srcSR.Close()

	trn, err := godal.NewTransform(srcSR, dstSR)
	if err != nil {
		return gt, fmt.Errorf(`error deriving geotransform: error creating coordinate transform: %w`, err)
	}
	defer trn.Close()

	x := []float64{lons[0], lons[1], lons[width]}
	y := []float64{lats[0], lats[1], lats[width]}
	if err := trn.TransformEx(x, y, nil, nil); err != nil {
		return gt, fmt.Errorf(`error deriving geotransform: error reprojecting corner points: %w`, err)
	}

	originX, originY := x[0], y[0]
	pixelWidth := x[1] - x[0]
	pixelHeight := y[2] - y[0]
	rowDriftX := x[2] - x[0]
	colDriftY := y[1] - y[0]

	if math.Abs(rowDriftX) > math.Abs(pixelWidth)*cornerTolerance ||
		math.Abs(colDriftY) > math.Abs(pixelHeight)*cornerTolerance {
		return gt, fmt.Errorf(`error deriving geotransform: grid is not axis-aligned in its native spatial ref (row drift %v, col drift %v) - wrong spatial ref?`, rowDriftX, colDriftY)
	}

	gt[0], gt[1], gt[2] = originX, pixelWidth, 0
	gt[3], gt[4], gt[5] = originY, 0, pixelHeight
	return gt, nil
}

// Raster is a decoded single-band GeoTIFF: values in the same row-major scan order
// EncodeGeoTiff wrote them in, plus the georeferencing needed to place them.
type Raster struct {
	SizeX, SizeY int
	Values       []float64
	SrsWKT       string
	GeoTransform [6]float64
}

// DecodeGeoTiff reads a GeoTIFF written by EncodeGeoTiff. NaN (the nodata value) comes back as
// NaN. raw is staged as a temp file in workDir, since GDAL reads from a path.
func DecodeGeoTiff(raw []byte, workDir string) (*Raster, error) {
	path := filepath.Join(workDir, uuid.NewString()+`-read.tif`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return nil, fmt.Errorf(`error decoding geotiff: error staging file: %w`, err)
	}
	defer func() { _ = os.Remove(path) }()

	ds, err := godal.Open(path)
	if err != nil {
		return nil, fmt.Errorf(`error decoding geotiff: error opening dataset: %w`, err)
	}
	defer func() { _ = ds.Close() }()

	structure := ds.Structure()
	if structure.NBands != 1 {
		return nil, fmt.Errorf(`error decoding geotiff: expected 1 band, got %d`, structure.NBands)
	}
	width, height := structure.SizeX, structure.SizeY
	values := make([]float64, width*height)
	if err := ds.Bands()[0].Read(0, 0, values, width, height); err != nil {
		return nil, fmt.Errorf(`error decoding geotiff: error reading band: %w`, err)
	}
	gt, err := ds.GeoTransform()
	if err != nil {
		return nil, fmt.Errorf(`error decoding geotiff: error reading geotransform: %w`, err)
	}
	wkt, err := ds.SpatialRef().WKT()
	if err != nil {
		return nil, fmt.Errorf(`error decoding geotiff: error reading spatial ref: %w`, err)
	}
	return &Raster{SizeX: width, SizeY: height, Values: values, SrsWKT: wkt, GeoTransform: gt}, nil
}

// Grid rebuilds the GridValues EncodeGeoTiff was given: Values plus the plain geographic lat/lon
// of every point, recovered by inverting the geotransform EncodeGeoTiff derived from the grid's
// points (origin at point [0,0], not a pixel corner) and reprojecting into degrees. The result
// can be passed back to EncodeGeoTiff or to isoband generation like any grid from a GRIB file.
// Values is shared with the Raster, not copied.
func (r *Raster) Grid() (*gridtoisobands.GridValues, error) {
	sr, err := godal.NewSpatialRefFromWKT(r.SrsWKT)
	if err != nil {
		return nil, fmt.Errorf(`error building grid from raster: error parsing spatial ref: %w`, err)
	}
	defer sr.Close()
	geoSR, err := godal.NewSpatialRefFromEPSG(4326)
	if err != nil {
		return nil, fmt.Errorf(`error building grid from raster: error creating geographic spatial ref: %w`, err)
	}
	defer geoSR.Close()
	trn, err := godal.NewTransform(sr, geoSR)
	if err != nil {
		return nil, fmt.Errorf(`error building grid from raster: error creating coordinate transform: %w`, err)
	}
	defer trn.Close()

	gt := r.GeoTransform
	lons := make([]float64, r.SizeX*r.SizeY)
	lats := make([]float64, r.SizeX*r.SizeY)
	for row := 0; row < r.SizeY; row++ {
		for col := 0; col < r.SizeX; col++ {
			i := row*r.SizeX + col
			lons[i] = gt[0] + float64(col)*gt[1] + float64(row)*gt[2]
			lats[i] = gt[3] + float64(col)*gt[4] + float64(row)*gt[5]
		}
	}
	if err := trn.TransformEx(lons, lats, nil, nil); err != nil {
		return nil, fmt.Errorf(`error building grid from raster: error reprojecting points: %w`, err)
	}
	return &gridtoisobands.GridValues{SizeX: r.SizeX, SizeY: r.SizeY, Values: r.Values, Lats: lats, Lons: lons}, nil
}
