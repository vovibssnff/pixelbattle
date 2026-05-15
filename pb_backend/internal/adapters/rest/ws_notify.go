package rest

// WSResizeNotifier notifies WebSocket clients of canvas geometry changes (ADR-003).
type WSResizeNotifier interface {
	NotifyCanvasResize(width, height uint, payload []byte)
}
