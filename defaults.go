package gridoutput

import (
	gridtoisobands "github.com/skysparq/grid-to-isobands"
	"github.com/skysparq/grid-to-isobands/transformers"
)

type Default struct {
	Step  float64
	Floor float64
	// Max, if non-zero, limits which isobands are output: values above it are folded into a
	// single top band. It only affects the isobands, never the grid saved as a geotiff.
	Max float64
	// GeotiffOnly skips isoband generation entirely, for measures that only exist as inputs to
	// downstream derivations (which consume the geotiff) and are never rendered themselves. With
	// no isobands to stay consistent with, the geotiff is also left unsmoothed.
	GeotiffOnly bool
}

// Selections returns the isoband-only transforms implied by this default.
func (d Default) Selections() []gridtoisobands.GridTransformer {
	if d.Max == 0 {
		return nil
	}
	return []gridtoisobands.GridTransformer{
		gridtoisobands.ThresholdMaskTransformer(transformers.GreaterThan(d.Max), d.Max+d.Step),
	}
}

var windSpeed = Default{Step: 10, Floor: -5}
var uvVector = Default{Step: 10, Floor: -1005}
var temperature = Default{Step: 2, Floor: -1001}
var humidity = Default{Step: 5, Floor: -2.5}
var geopotentialHeightCourse = Default{Step: 60, Floor: 0}
var geopotentialHeightFine = Default{Step: 10, Floor: -3000}
var freezeIsothermHeight = Default{Step: 150, Floor: -75}
var visibility = Default{Step: 2000, Floor: 0}
var reflectivity = Default{Step: 2.5, Floor: 3.75}
var haines = Default{Step: 1, Floor: 0.5}
var cloudCover = Default{Step: 10, Floor: -5}
var baroPressure = Default{Step: 2.5, Floor: -1.25}
var ceiling = Default{Step: 150, Floor: -75, Max: 3000}
var geotiffOnly = Default{GeotiffOnly: true}
var qpe = Default{Step: 2.5, Floor: 0.25, Max: 50}

// AirNow AQI grids hold integer AQI values with missing cells outside monitor coverage. Edges at
// x.5 put every EPA category breakpoint (50/51, 100/101, 150/151, 200/201, 300/301) on a band
// edge, with several bands per category for a smoother gradient. The -24.5 floor gives AQI 0,
// which is real clean air rather than missing data, its own band instead of a hole. Everything
// above 300 is folded into a single Hazardous band.
var aqi = Default{Step: 25, Floor: -24.5, Max: 300}

var Defaults = map[string]Default{
	`wind-u-100hpa`:              uvVector,
	`wind-v-100hpa`:              uvVector,
	`wind-speed-100hpa`:          windSpeed,
	`temperature-100hpa`:         temperature,
	`humidity-100hpa`:            humidity,
	`dew-point-100hpa`:           temperature,
	`geopotential-height-100hpa`: geopotentialHeightCourse,

	`wind-u-150hpa`:              uvVector,
	`wind-v-150hpa`:              uvVector,
	`wind-speed-150hpa`:          windSpeed,
	`temperature-150hpa`:         temperature,
	`humidity-150hpa`:            humidity,
	`dew-point-150hpa`:           temperature,
	`geopotential-height-150hpa`: geopotentialHeightCourse,

	`wind-u-200hpa`:              uvVector,
	`wind-v-200hpa`:              uvVector,
	`wind-speed-200hpa`:          windSpeed,
	`temperature-200hpa`:         temperature,
	`humidity-200hpa`:            humidity,
	`dew-point-200hpa`:           temperature,
	`geopotential-height-200hpa`: geopotentialHeightCourse,

	`wind-u-250hpa`:              uvVector,
	`wind-v-250hpa`:              uvVector,
	`wind-speed-250hpa`:          windSpeed,
	`temperature-250hpa`:         temperature,
	`humidity-250hpa`:            humidity,
	`dew-point-250hpa`:           temperature,
	`geopotential-height-250hpa`: geopotentialHeightCourse,

	`wind-u-300hpa`:              uvVector,
	`wind-v-300hpa`:              uvVector,
	`wind-speed-300hpa`:          windSpeed,
	`temperature-300hpa`:         temperature,
	`humidity-300hpa`:            humidity,
	`dew-point-300hpa`:           temperature,
	`geopotential-height-300hpa`: geopotentialHeightCourse,

	`wind-u-350hpa`:              uvVector,
	`wind-v-350hpa`:              uvVector,
	`wind-speed-350hpa`:          windSpeed,
	`temperature-350hpa`:         temperature,
	`humidity-350hpa`:            humidity,
	`dew-point-350hpa`:           temperature,
	`geopotential-height-350hpa`: geopotentialHeightCourse,

	`wind-u-400hpa`:              uvVector,
	`wind-v-400hpa`:              uvVector,
	`wind-speed-400hpa`:          windSpeed,
	`temperature-400hpa`:         temperature,
	`humidity-400hpa`:            humidity,
	`dew-point-400hpa`:           temperature,
	`geopotential-height-400hpa`: geopotentialHeightCourse,

	`wind-u-450hpa`:              uvVector,
	`wind-v-450hpa`:              uvVector,
	`wind-speed-450hpa`:          windSpeed,
	`temperature-450hpa`:         temperature,
	`humidity-450hpa`:            humidity,
	`dew-point-450hpa`:           temperature,
	`geopotential-height-450hpa`: geopotentialHeightCourse,

	`wind-u-500hpa`:              uvVector,
	`wind-v-500hpa`:              uvVector,
	`wind-speed-500hpa`:          windSpeed,
	`temperature-500hpa`:         temperature,
	`humidity-500hpa`:            humidity,
	`dew-point-500hpa`:           temperature,
	`geopotential-height-500hpa`: geopotentialHeightFine,

	`wind-u-550hpa`:              uvVector,
	`wind-v-550hpa`:              uvVector,
	`wind-speed-550hpa`:          windSpeed,
	`temperature-550hpa`:         temperature,
	`humidity-550hpa`:            humidity,
	`dew-point-550hpa`:           temperature,
	`geopotential-height-550hpa`: geopotentialHeightFine,

	`wind-u-600hpa`:              uvVector,
	`wind-v-600hpa`:              uvVector,
	`wind-speed-600hpa`:          windSpeed,
	`temperature-600hpa`:         temperature,
	`humidity-600hpa`:            humidity,
	`dew-point-600hpa`:           temperature,
	`geopotential-height-600hpa`: geopotentialHeightFine,

	`wind-u-650hpa`:              uvVector,
	`wind-v-650hpa`:              uvVector,
	`wind-speed-650hpa`:          windSpeed,
	`temperature-650hpa`:         temperature,
	`humidity-650hpa`:            humidity,
	`dew-point-650hpa`:           temperature,
	`geopotential-height-650hpa`: geopotentialHeightFine,

	`wind-u-700hpa`:              uvVector,
	`wind-v-700hpa`:              uvVector,
	`wind-speed-700hpa`:          windSpeed,
	`temperature-700hpa`:         temperature,
	`humidity-700hpa`:            humidity,
	`dew-point-700hpa`:           temperature,
	`geopotential-height-700hpa`: geopotentialHeightFine,

	`wind-u-750hpa`:              uvVector,
	`wind-v-750hpa`:              uvVector,
	`wind-speed-750hpa`:          windSpeed,
	`temperature-750hpa`:         temperature,
	`humidity-750hpa`:            humidity,
	`dew-point-750hpa`:           temperature,
	`geopotential-height-750hpa`: geopotentialHeightFine,

	`wind-u-800hpa`:              uvVector,
	`wind-v-800hpa`:              uvVector,
	`wind-speed-800hpa`:          windSpeed,
	`temperature-800hpa`:         temperature,
	`humidity-800hpa`:            humidity,
	`dew-point-800hpa`:           temperature,
	`geopotential-height-800hpa`: geopotentialHeightFine,

	`wind-u-850hpa`:              uvVector,
	`wind-v-850hpa`:              uvVector,
	`wind-speed-850hpa`:          windSpeed,
	`temperature-850hpa`:         temperature,
	`humidity-850hpa`:            humidity,
	`dew-point-850hpa`:           temperature,
	`geopotential-height-850hpa`: geopotentialHeightFine,

	`wind-u-900hpa`:              uvVector,
	`wind-v-900hpa`:              uvVector,
	`wind-speed-900hpa`:          windSpeed,
	`temperature-900hpa`:         temperature,
	`humidity-900hpa`:            humidity,
	`dew-point-900hpa`:           temperature,
	`geopotential-height-900hpa`: geopotentialHeightFine,

	`wind-u-925hpa`:              uvVector,
	`wind-v-925hpa`:              uvVector,
	`wind-speed-925hpa`:          windSpeed,
	`temperature-925hpa`:         temperature,
	`humidity-925hpa`:            humidity,
	`dew-point-925hpa`:           temperature,
	`geopotential-height-925hpa`: geopotentialHeightFine,

	`wind-u-950hpa`:              uvVector,
	`wind-v-950hpa`:              uvVector,
	`wind-speed-950hpa`:          windSpeed,
	`temperature-950hpa`:         temperature,
	`humidity-950hpa`:            humidity,
	`dew-point-950hpa`:           temperature,
	`geopotential-height-950hpa`: geopotentialHeightFine,

	`wind-u-975hpa`:              uvVector,
	`wind-v-975hpa`:              uvVector,
	`wind-speed-975hpa`:          windSpeed,
	`temperature-975hpa`:         temperature,
	`humidity-975hpa`:            humidity,
	`dew-point-975hpa`:           temperature,
	`geopotential-height-975hpa`: geopotentialHeightFine,

	`wind-u-1000hpa`:              uvVector,
	`wind-v-1000hpa`:              uvVector,
	`wind-speed-1000hpa`:          windSpeed,
	`temperature-1000hpa`:         temperature,
	`humidity-1000hpa`:            humidity,
	`dew-point-1000hpa`:           temperature,
	`geopotential-height-1000hpa`: geopotentialHeightFine,

	`geopotential-height-freeze-isotherm`: freezeIsothermHeight,
	`humidity-freeze-isotherm`:            humidity,
	`dew-point-freeze-isotherm`:           temperature,

	`humidity-surface`:             humidity,
	`wind-gust-surface`:            windSpeed,
	`wind-chill-surface`:           temperature,
	`dew-point-surface`:            temperature,
	`apparent-temperature-surface`: temperature,
	`visibility-surface`:           visibility,
	`temperature-surface`:          temperature,
	`wind-speed-surface`:           windSpeed,
	`wind-u-surface`:               uvVector,
	`wind-v-surface`:               uvVector,

	`haines-index`:      haines,
	`cloud-cover-atmos`: cloudCover,
	`baro-pressure-msl`: baroPressure,

	`baro-pressure-surface`: geotiffOnly,
	`terrain-height`:        geotiffOnly,

	`base-reflectivity`:      reflectivity,
	`composite-reflectivity`: reflectivity,

	`ceiling`: ceiling,

	`qpe-1hr`: qpe,

	`aqi-ozone`:    aqi,
	`aqi-pm25`:     aqi,
	`aqi-pm10`:     aqi,
	`aqi-combined`: aqi,
}
