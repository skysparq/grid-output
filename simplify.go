package gridoutput

import "github.com/skysparq/grid-to-isobands/contour"

// Simplify returns the isoband simplification for a table's grids, or nil (no simplification)
// for a table without one.
//
// The settings scale with each grid's spacing (dx) so every table strikes the same balance of
// file size against fidelity, the one tuned by hand for the 2.5 km RTMA/URMA grid:
//
//   - Tolerance ≈ 0.2 dx. Past this, output size falls off more slowly on every grid sampled
//     (RTMA, HRRR, MRMS and GFS), so it's about the same point on each grid's
//     size-against-tolerance curve.
//   - MinRingArea ≈ 0.0016 dx², well under a cell, so it only clears slivers. Raising it saves
//     little (rings that small have few points), and at a cell or more it starts deleting real
//     features, such as small high-reflectivity cores in MRMS.
//
// gfs and gdas share a 0.25° grid whose east-west spacing shrinks toward the poles. Their
// tolerance is sized to the north-south spacing (~27.8 km), so it's a larger fraction of the
// cell at high latitudes; diminishing returns set in early on that grid anyway.
func Simplify(table string) *contour.SimplifyOptions {
	switch table {
	case `rtma`, `urma`: // 2.5 km
		return &contour.SimplifyOptions{Tolerance: 500, MinRingArea: 10_000}
	case `hrrr`: // 3 km
		return &contour.SimplifyOptions{Tolerance: 600, MinRingArea: 15_000}
	case `mrms`: // 0.01° (~1.1 km)
		return &contour.SimplifyOptions{Tolerance: 250, MinRingArea: 2_000}
	case `gfs`, `gdas`: // 0.25° (~27.8 km)
		return &contour.SimplifyOptions{Tolerance: 5_000, MinRingArea: 1_250_000}
	}
	return nil
}
