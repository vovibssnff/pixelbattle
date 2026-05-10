package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"pb_backend/internal/core/domain/crdt"
	"pb_backend/internal/metrics"
)

func writerFromPixelPayload(pixelData []byte) string {
	var aux struct {
		UserID json.RawMessage `json:"userid"`
	}
	if err := json.Unmarshal(pixelData, &aux); err != nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(aux.UserID, &s); err == nil {
		return s
	}
	var n json.Number
	if err := json.Unmarshal(aux.UserID, &n); err == nil {
		return n.String()
	}
	return ""
}

func redisLuaInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int64:
		return int(n), true
	case int:
		return n, true
	case int32:
		return int(n), true
	default:
		return 0, false
	}
}

func (r *CanvasRepository) writePixelWithOpLog(ctx context.Context, x, y uint, pixelData []byte) error {
	start := time.Now()
	hlc := crdt.Clock{Millis: uint64(time.Now().UnixMilli()), Logical: 0}
	writer := writerFromPixelPayload(pixelData)
	keys := []string{
		OpStreamKey(x, y),
		r.pixelKey(x, y),
		PixelMetaKey(x, y),
	}
	args := []interface{}{
		string(pixelData),
		writer,
		strconv.FormatUint(hlc.Millis, 10),
		strconv.FormatUint(uint64(hlc.Logical), 10),
	}
	res, err := r.rdb.Eval(ctx, pixelWriteOpLogScript, keys, args...).Result()
	if err != nil {
		metrics.ObserveDatabaseOperation("write_pixel", "redis", time.Since(start), err)
		return err
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 3 {
		metrics.ObserveDatabaseOperation("write_pixel", "redis", time.Since(start), errBadOpLogReturn)
		return errBadOpLogReturn
	}
	pathCode, ok := redisLuaInt(arr[2])
	if !ok {
		metrics.ObserveDatabaseOperation("write_pixel", "redis", time.Since(start), errBadOpLogReturn)
		return errBadOpLogReturn
	}
	path := string(crdt.ResolvedByHLCFallback)
	if pathCode == 1 {
		path = string(crdt.ResolvedByStreamID)
	}
	metrics.IncrementCRDTResolved(path)
	if pathCode == 2 {
		metrics.ObserveCRDTMergeDuration(time.Since(start))
	}
	metrics.ObserveDatabaseOperation("write_pixel", "redis", time.Since(start), nil)
	return nil
}

var errBadOpLogReturn = errors.New("redis: unexpected EVAL return from pixel op-log script")
