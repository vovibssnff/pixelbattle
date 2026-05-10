package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestIncrementCRDTResolved(t *testing.T) {
	before := testutil.ToFloat64(crdtResolvedTotal.WithLabelValues("stream_id"))
	IncrementCRDTResolved("stream_id")
	after := testutil.ToFloat64(crdtResolvedTotal.WithLabelValues("stream_id"))
	if got := after - before; got != 1 {
		t.Fatalf("stream_id delta %g", got)
	}
}

func TestObserveDatabaseOperationScopedIncrementsCounter(t *testing.T) {
	lv := []string{"op", "redis", "success", "shard-a", "gw-1"}
	before := testutil.ToFloat64(databaseOperationTotal.WithLabelValues(lv[0], lv[1], lv[2], lv[3], lv[4]))
	ObserveDatabaseOperationScoped("op", "redis", "shard-a", "gw-1", time.Millisecond, nil)
	after := testutil.ToFloat64(databaseOperationTotal.WithLabelValues(lv[0], lv[1], lv[2], lv[3], lv[4]))
	if after-before != 1 {
		t.Fatalf("counter delta %g", after-before)
	}
}

func TestObserveDatabaseOperationDefaultsShardLabels(t *testing.T) {
	before := testutil.ToFloat64(databaseOperationTotal.WithLabelValues("x", "redis", "success", MonolithShardID, MonolithInstanceID))
	ObserveDatabaseOperation("x", "redis", time.Millisecond, nil)
	after := testutil.ToFloat64(databaseOperationTotal.WithLabelValues("x", "redis", "success", MonolithShardID, MonolithInstanceID))
	if after-before != 1 {
		t.Fatalf("expected monolith labels, delta %g", after-before)
	}
}
