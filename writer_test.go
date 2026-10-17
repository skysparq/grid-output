package gridoutput

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type fakeS3 struct{ puts map[string][]byte }

func (f *fakeS3) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	body, err := io.ReadAll(in.Body)
	if err != nil {
		return nil, err
	}
	f.puts[*in.Bucket+`/`+*in.Key] = body
	return &s3.PutObjectOutput{}, nil
}

type fakeSNS struct{ messages []string }

func (f *fakeSNS) Publish(_ context.Context, in *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
	f.messages = append(f.messages, *in.Message)
	return &sns.PublishOutput{}, nil
}

func newTestWriter(t *testing.T) (*Writer, *fakeS3, *fakeSNS) {
	t.Helper()
	s3c, snsc := &fakeS3{puts: map[string][]byte{}}, &fakeSNS{}
	return &Writer{S3: s3c, SNS: snsc, Env: `dev`, MetadataTopicArn: `arn:topic`, WorkDir: t.TempDir()}, s3c, snsc
}

var writeAt = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

func TestWriteWithIsobands(t *testing.T) {
	w, s3c, snsc := newTestWriter(t)
	grid, wkt := smallLambertGrid(t)
	err := w.Write(context.Background(), Output{
		Table: `rtma`, Measure: `baro-pressure-msl`, At: writeAt, CycleTime: writeAt,
		Grid: grid, SrsWKT: wkt, System: `RTMA`, Subsystem: `VARIATION-ANALYSIS`, GridName: `conus-2.5km-lcc`,
	})
	if err != nil {
		t.Fatal(err)
	}

	tifKey := `skysparq-geotiff-dev/rtma/baro-pressure-msl/20261016/20261016T120000.tif`
	if _, ok := s3c.puts[tifKey]; !ok {
		t.Errorf(`expected geotiff at %s, got %v`, tifKey, keys(s3c.puts))
	}
	isobands := 0
	for k := range s3c.puts {
		if strings.HasPrefix(k, `skysparq-ingest-dev/rtma-isobands/baro-pressure-msl-20261016T120000-`) {
			isobands++
		}
	}
	if isobands != 1 {
		t.Errorf(`expected 1 isoband upload, got %d (%v)`, isobands, keys(s3c.puts))
	}
	assertMetadata(t, snsc, `baro-pressure-msl`, `rtma/baro-pressure-msl/20261016/20261016T120000.tif`)
}

func TestWriteGeotiffOnlySkipsIsobands(t *testing.T) {
	w, s3c, snsc := newTestWriter(t)
	grid, wkt := smallLambertGrid(t)
	want := append([]float64(nil), grid.Values...)
	err := w.Write(context.Background(), Output{
		Table: `rtma`, Measure: `baro-pressure-surface`, At: writeAt, CycleTime: writeAt, Grid: grid, SrsWKT: wkt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(s3c.puts) != 1 {
		t.Fatalf(`expected only the geotiff upload, got %v`, keys(s3c.puts))
	}
	raster, err := DecodeGeoTiff(s3c.puts[`skysparq-geotiff-dev/rtma/baro-pressure-surface/20261016/20261016T120000.tif`], w.WorkDir)
	if err != nil {
		t.Fatal(err)
	}
	for i := range want {
		if !math.IsNaN(want[i]) && raster.Values[i] != float64(float32(want[i])) {
			t.Fatalf(`value %d smoothed: got %v, want %v`, i, raster.Values[i], want[i])
		}
	}
	assertMetadata(t, snsc, `baro-pressure-surface`, `rtma/baro-pressure-surface/20261016/20261016T120000.tif`)
}

func TestWriteUnknownMeasure(t *testing.T) {
	w, s3c, _ := newTestWriter(t)
	grid, wkt := smallLambertGrid(t)
	if err := w.Write(context.Background(), Output{Table: `rtma`, Measure: `no-such-measure`, Grid: grid, SrsWKT: wkt}); err == nil {
		t.Fatal(`expected an error`)
	}
	if len(s3c.puts) != 0 {
		t.Errorf(`expected no uploads, got %v`, keys(s3c.puts))
	}
}

func assertMetadata(t *testing.T, snsc *fakeSNS, measure, key string) {
	t.Helper()
	if len(snsc.messages) != 1 {
		t.Fatalf(`expected 1 metadata message, got %d`, len(snsc.messages))
	}
	var record MetadataRecord
	if err := json.Unmarshal([]byte(snsc.messages[0]), &record); err != nil {
		t.Fatal(err)
	}
	if record.Measure != measure || record.GeotiffKey != key || !record.At.Equal(writeAt) {
		t.Errorf(`unexpected metadata %+v`, record)
	}
}

func keys(m map[string][]byte) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
