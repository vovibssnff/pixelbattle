package grpc

import "net"

// listen is split out so tests can stub it with bufconn.Listen.
func listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
