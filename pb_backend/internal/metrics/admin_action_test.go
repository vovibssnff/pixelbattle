package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestIncrementAdminActionIncrementsLabeledSeries(t *testing.T) {
	before := testutil.ToFloat64(adminActionTotal.WithLabelValues("ban", "ok"))
	IncrementAdminAction("ban", "ok")
	after := testutil.ToFloat64(adminActionTotal.WithLabelValues("ban", "ok"))
	if got := after - before; got != 1 {
		t.Fatalf("expected +1 on ban/ok, got delta %g (before %g after %g)", got, before, after)
	}
}
