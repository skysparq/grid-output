package gridoutput

import (
	"testing"
	"time"
)

func TestGeotiffKey(t *testing.T) {
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	cycle := at.Add(-6 * time.Hour)
	cases := []struct {
		table, measure, want string
	}{
		{`rtma`, `baro-pressure-msl`, `rtma/baro-pressure-msl/20261016/20261016T120000.tif`},
		{`mrms`, `precip-1h`, `mrms/precip-1h/20261016/20261016T120000.tif`},
		{`hrrr`, `temperature-500hpa`, `hrrr/temperature-500hpa/20261016/C20261016T060000-20261016T120000.tif`},
	}
	for _, c := range cases {
		if got := GeotiffKey(c.table, c.measure, at, cycle); got != c.want {
			t.Errorf(`%s: got %s, want %s`, c.table, got, c.want)
		}
	}
}

func TestBuckets(t *testing.T) {
	if got := GeotiffBucket(`dev`); got != `skysparq-geotiff-dev` {
		t.Errorf(`geotiff bucket = %s`, got)
	}
	if got := IsobandBucket(`prod`); got != `skysparq-ingest-prod` {
		t.Errorf(`isoband bucket = %s`, got)
	}
	if got := IsobandPrefix(`rtma`); got != `rtma-isobands` {
		t.Errorf(`isoband prefix = %s`, got)
	}
}
