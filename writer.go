// Package gridoutput is the output path shared by every grid producer: isoband generation and
// upload, Cloud Optimized GeoTIFF encoding and upload, and metadata publishing. grib-to-geo uses
// it for primary products and grid-derivations for derived ones, so both write the same keys,
// rendering defaults, and metadata. It also decodes the geotiffs it writes, for consumers that
// read grids back.
package gridoutput

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/google/uuid"
	gridtoisobands "github.com/skysparq/grid-to-isobands"
	"github.com/skysparq/grid-to-isobands/contour"
)

// S3API is the subset of *s3.Client the writer uses.
type S3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// SNSAPI is the subset of *sns.Client the writer uses.
type SNSAPI interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// Writer publishes processed grids.
type Writer struct {
	S3               S3API
	SNS              SNSAPI
	Env              string // dev or prod; selects the buckets
	MetadataTopicArn string
	WorkDir          string // scratch space for GDAL temp files
}

// Output is one processed grid to publish.
type Output struct {
	Table     string
	Measure   string
	At        time.Time
	CycleTime time.Time
	// Grid is modified in place by the table's transforms: pass a copy if the values or
	// coordinates are still needed afterwards.
	Grid   *gridtoisobands.GridValues
	SrsWKT string // the grid's native spatial ref

	// Provenance, carried into the isoband properties and the metadata record.
	Source    string // the isobands' "source" property
	System    string
	Subsystem string
	GridName  string
	Url       string
	Backend   gridtoisobands.Backend
	// SimplifyOptions overrides the table's isoband simplification (see Simplify). Leave nil
	// to use the table's; an empty SimplifyOptions turns simplification off.
	SimplifyOptions *contour.SimplifyOptions
}

// Write publishes out the same way for every producer:
//
//  1. Unless the measure is GeotiffOnly, smooth the grid with the table's Transforms and upload
//     its isobands. The geotiff is written from the smoothed grid so the two agree.
//  2. Otherwise apply only the table's GeometryFixes, leaving the values unsmoothed for
//     derivations to consume.
//  3. Upload the grid as a COG at its deterministic GeotiffKey, overwriting any earlier one.
//  4. Publish a MetadataRecord for it.
func (w *Writer) Write(ctx context.Context, out Output) error {
	defaults, ok := Defaults[out.Measure]
	if !ok {
		return fmt.Errorf(`error writing %v/%v: no defaults for measure`, out.Table, out.Measure)
	}

	if defaults.GeotiffOnly {
		for _, fix := range GeometryFixes(out.Table) {
			fix(out.Grid)
		}
	} else if err := w.uploadIsobands(ctx, out, defaults); err != nil {
		return fmt.Errorf(`error writing %v/%v: %w`, out.Table, out.Measure, err)
	}

	tif, err := EncodeGeoTiff(out.Grid, out.SrsWKT, w.WorkDir)
	if err != nil {
		return fmt.Errorf(`error writing %v/%v: %w`, out.Table, out.Measure, err)
	}
	tifKey := GeotiffKey(out.Table, out.Measure, out.At, out.CycleTime)
	_, err = w.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(GeotiffBucket(w.Env)),
		Key:         aws.String(tifKey),
		Body:        bytes.NewReader(tif),
		ContentType: aws.String("image/tiff"),
	})
	if err != nil {
		return fmt.Errorf(`error writing %v/%v: error uploading geotiff: %w`, out.Table, out.Measure, err)
	}

	println(fmt.Sprintf(`INFO published %s to %s`, out.Measure, tifKey))

	err = w.publishMetadata(ctx, MetadataRecord{
		System:     out.System,
		Subsystem:  out.Subsystem,
		Grid:       out.GridName,
		Measure:    out.Measure,
		At:         out.At,
		CycleTime:  out.CycleTime,
		Url:        out.Url,
		GeotiffKey: tifKey,
	})
	if err != nil {
		return fmt.Errorf(`error writing %v/%v: %w`, out.Table, out.Measure, err)
	}
	return nil
}

// uploadIsobands contours out.Grid (applying the table's Transforms to it in place) and uploads
// the isobands.
func (w *Writer) uploadIsobands(ctx context.Context, out Output, defaults Default) error {
	simplify := out.SimplifyOptions
	if simplify == nil {
		simplify = Simplify(out.Table)
	}
	args := &gridtoisobands.IsobandArgs{
		Grid:         out.Grid,
		Preprocesses: Transforms(out.Table),
		Selections:   defaults.Selections(),
		Floor:        defaults.Floor,
		Step:         defaults.Step,
		WorkDir:      w.WorkDir,
		Backend:      out.Backend,
		Simplify:     simplify,
	}
	output, err := gridtoisobands.IsobandsFromGrid(ctx, args)
	if err != nil {
		return fmt.Errorf(`error generating isobands: %w`, err)
	}
	isobandList := featureCollectionToList(output.Isobands, map[string]any{
		"source":      out.Source,
		"at":          out.At,
		"cycle":       out.CycleTime,
		"grid":        out.GridName,
		"measure":     out.Measure,
		"source_file": out.Url,
	})
	raw, err := json.Marshal(isobandList)
	if err != nil {
		return fmt.Errorf(`error encoding isobands: %w`, err)
	}
	filename := fmt.Sprintf(`%v-%v-%v.geojson`, out.Measure, out.At.Format(`20060102T150405`), uuid.NewString())
	key := fmt.Sprintf(`%v/%v`, IsobandPrefix(out.Table), filename)
	_, err = w.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(IsobandBucket(w.Env)),
		Key:         aws.String(key),
		Body:        bytes.NewReader(raw),
		ContentType: aws.String("application/geo+json"),
	})
	if err != nil {
		return fmt.Errorf(`error uploading isobands: %w`, err)
	}

	println(fmt.Sprintf(`INFO published %s to %s`, out.Measure, filename))

	return nil
}

func (w *Writer) publishMetadata(ctx context.Context, record MetadataRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf(`error publishing metadata: %w`, err)
	}
	_, err = w.SNS.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(w.MetadataTopicArn),
		Message:  aws.String(string(raw)),
	})
	if err != nil {
		return fmt.Errorf(`error publishing metadata: %w`, err)
	}
	return nil
}

func featureCollectionToList(fc *gridtoisobands.FeatureCollection, props map[string]any) []gridtoisobands.Feature {
	features := make([]gridtoisobands.Feature, 0, len(fc.Features))
	for i, feature := range fc.Features {
		allProps := maps.Clone(props)
		maps.Copy(allProps, feature.Properties)
		allProps["index"] = i
		features = append(features, gridtoisobands.Feature{
			Type:       feature.Type,
			Geometry:   feature.Geometry,
			Properties: allProps,
		})
	}
	return features
}
