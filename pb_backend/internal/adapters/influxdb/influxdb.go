package influxdb

import (
	"context"
	"fmt"
	"pb_backend/internal/core/domain"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/sirupsen/logrus"
)

const (
	org    = "pixelbattle"
	bucket = "canvas"
)

type InfluxDBAdapter struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	queryAPI api.QueryAPI
}

func NewInfluxDBAdapter(url, token string) *InfluxDBAdapter {
	client := influxdb2.NewClient(url, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)
	queryAPI := client.QueryAPI(org)

	return &InfluxDBAdapter{
		client:   client,
		writeAPI: writeAPI,
		queryAPI: queryAPI,
	}
}

func (a *InfluxDBAdapter) RecordPixelChange(ctx context.Context, pixel domain.Pixel) error {

	point := influxdb2.NewPoint(
		"pixel_changes",
		map[string]string{
			"xy": fmt.Sprintf("%v:%v", pixel.X, pixel.Y),
		},
		map[string]interface{}{
			"x": pixel.X,
			"y": pixel.Y,
		},
		time.Now(),
	)

	/*
		from(bucket: "canvas")
		  |> range(start: -1h)
		  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
		  |> group(columns: ["x", "y"])
		  |> map(fn: (r) => ({r with _value: 1337}))
		  |> count()
		  |> group()
		  |> sort(columns: ["y"])
	*/
	logrus.Infof("Writing point: %s", point.Time().String())

	err := a.writeAPI.WritePoint(ctx, point)
	if err != nil {
		logrus.Errorf("Write failed: %v", err)
		return fmt.Errorf("failed to write point: %w", err)
	}

	logrus.Info("Point written successfully")
	return nil
}

// GetHeatmapData returns heatmap data for the last hour
func (a *InfluxDBAdapter) GetHeatmapData(ctx context.Context) (map[string]int, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -1h)
			|> filter(fn: (r) => r._measurement == "pixel_changes")
			|> group(columns: ["x", "y"])
			|> count()
	`, bucket)

	result, err := a.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query heatmap data: %w", err)
	}
	defer result.Close()

	heatmap := make(map[string]int)
	for result.Next() {
		record := result.Record()
		x := record.ValueByKey("x").(int)
		y := record.ValueByKey("y").(int)
		count := record.ValueByKey("_value").(int64)
		key := fmt.Sprintf("%d,%d", x, y)
		heatmap[key] = int(count)
	}

	return heatmap, nil
}

// GetMostActiveZone returns the most active zone in the last hour
func (a *InfluxDBAdapter) GetMostActiveZone(ctx context.Context, resolution int) (int, int, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -1h)
			|> filter(fn: (r) => r._measurement == "pixel_changes")
			|> group(columns: ["x", "y"])
			|> count()
			|> sort(columns: ["_value"], desc: true)
			|> limit(n: 1)
	`, bucket)

	result, err := a.queryAPI.Query(ctx, query)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query most active zone: %w", err)
	}
	defer result.Close()

	if !result.Next() {
		return 0, 0, fmt.Errorf("no data found")
	}

	record := result.Record()
	x := record.ValueByKey("x").(int)
	y := record.ValueByKey("y").(int)

	// Round to the nearest zone based on resolution
	zoneX := (x / resolution) * resolution
	zoneY := (y / resolution) * resolution

	return zoneX, zoneY, nil
}

func (a *InfluxDBAdapter) Close() {
	a.client.Close()
}
