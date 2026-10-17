package gridoutput

import (
	"fmt"
	"slices"
	"time"
)

// GeotiffBucket is the bucket holding every table's geotiffs.
func GeotiffBucket(env string) string {
	return fmt.Sprintf(`skysparq-geotiff-%v`, env)
}

// IsobandBucket is the bucket isobands are uploaded to.
func IsobandBucket(env string) string {
	return fmt.Sprintf(`skysparq-ingest-%v`, env)
}

// IsobandPrefix is the key prefix for a table's isobands within IsobandBucket.
func IsobandPrefix(table string) string {
	return fmt.Sprintf(`%v-isobands`, table)
}

// GeotiffKey is the key of a grid's geotiff within GeotiffBucket.
//
// Keyed by table/measure/day/time rather than a unique-per-upload name: the geotiff bucket is
// a queryable-by-path store of the latest grid for each table+measure+time, not an archive,
// so a later run for the same time is expected to overwrite an earlier one rather than
// accumulate alongside it. Forecast tables additionally key by cycle time: overlapping
// forecast cycles can validate at the same "at" time with materially different values, so
// without the cycle in the key a later cycle's run would silently overwrite an earlier
// cycle's distinct forecast for that same valid time. Actuals (e.g. mrms) have no such
// overlap - cycle time and valid time are the same thing - so their key is left as-is.
//
// Because the key is deterministic, anything that knows a grid's table, measure, and times can
// find its geotiff without searching.
func GeotiffKey(table, measure string, at, cycleTime time.Time) string {
	filename := fmt.Sprintf(`%v.tif`, at.Format(`20060102T150405`))
	if IsForecastTable(table) {
		filename = fmt.Sprintf(`C%v-%v.tif`, cycleTime.Format(`20060102T150405`), at.Format(`20060102T150405`))
	}
	return fmt.Sprintf(`%v/%v/%v/%v`, table, measure, at.Format(`20060102`), filename)
}

// IsForecastTable reports whether table produces forecast data, where the same valid ("at")
// time can legitimately be reached by more than one model cycle. Actuals-style tables (mrms,
// actuals) have no such overlap, since their cycle time and valid time are the same instant.
func IsForecastTable(table string) bool {
	return slices.Contains([]string{`gfs`, `hrrr`}, table)
}
