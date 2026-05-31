package rest

// WSRuntimeNotifier notifies WebSocket clients of runtime admin changes.
type WSRuntimeNotifier interface {
	NotifyCanvasResize(width, height uint, payload []byte)
	NotifyPixelCooldown(seconds int, payload []byte)
}
