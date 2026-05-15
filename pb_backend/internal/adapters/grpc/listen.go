package grpc

import (
	"context"
	"net"
)

// listen is split out so tests can stub it with bufconn.Listen.
func listen(addr string) (net.Listener, error) {
	var lc net.ListenConfig
	return lc.Listen(context.Background(), "tcp", addr)
}
