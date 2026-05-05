package service

import "sync/atomic"

// canvasFrozen is a process-wide atomic toggled by POST /api/admin/canvas/freeze.
// When set, the WebSocket pixel placement path rejects placements with reason "frozen"
// and increments ws_errors_total{kind="frozen"} (plan §12.1). The flag is intentionally
// in-memory: a restart resets it to false, which is the safe default for an event.
var canvasFrozen atomic.Bool

// SetCanvasFrozen toggles read-only mode. Used by the admin handler.
func SetCanvasFrozen(frozen bool) {
	canvasFrozen.Store(frozen)
}

// IsCanvasFrozen returns true when placements should be rejected.
func IsCanvasFrozen() bool {
	return canvasFrozen.Load()
}
