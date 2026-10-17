package gridoutput

import (
	"math"
	"slices"

	gridtoisobands "github.com/skysparq/grid-to-isobands"
	"github.com/skysparq/grid-to-isobands/transformers"
)

// GeometryFixes returns the subset of Transforms that the grid needs to produce a valid
// geotiff, as opposed to the smoothing/cleanup that only exists for the isobands' benefit.
func GeometryFixes(table string) []gridtoisobands.GridTransformer {
	if slices.Contains([]string{`gdas`, `gfs`}, table) {
		return []gridtoisobands.GridTransformer{
			gridtoisobands.SwapRightAndLeftTransformer(),
		}
	}
	return []gridtoisobands.GridTransformer{}
}

// Transforms returns the preprocessing applied to a table's grids before contouring. They are
// applied in place, so the geotiff written afterwards carries them too.
func Transforms(table string) []gridtoisobands.GridTransformer {
	if slices.Contains([]string{`gdas`, `gfs`}, table) {
		return []gridtoisobands.GridTransformer{
			gridtoisobands.SwapRightAndLeftTransformer(),
			gridtoisobands.GaussianTransformer(7, 1),
			gridtoisobands.ClipTransformer(transformers.Clip{
				Top:    20,
				Bottom: 20,
				Left:   1,
				Right:  1,
			}),
		}
	}
	if table == `mrms` {
		return []gridtoisobands.GridTransformer{
			gridtoisobands.ThresholdMaskTransformer(transformers.LessThan(-2.6), -2.6),
			gridtoisobands.BilateralTransformer(1, 2.5),
			gridtoisobands.CloseOpenTransformer(3),
			gridtoisobands.GaussianTransformer(5, 1.0),
			gridtoisobands.ThresholdMaskTransformer(transformers.LessThan(-2.5), math.NaN()),
		}
	}
	if slices.Contains([]string{`hrrr`, `rtma`, `urma`}, table) {
		return []gridtoisobands.GridTransformer{
			gridtoisobands.GaussianTransformer(7, 1),
		}
	}
	return []gridtoisobands.GridTransformer{}
}
