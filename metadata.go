package gridoutput

import "time"

// MetadataRecord describes one published geotiff, for the Iceberg lookup table that backs
// interpolated point lookups against the geotiff database. It is published as JSON to the
// metadata SNS topic after each geotiff upload.
type MetadataRecord struct {
	System     string
	Subsystem  string
	Grid       string
	Measure    string
	At         time.Time
	CycleTime  time.Time
	Url        string
	GeotiffKey string
}
