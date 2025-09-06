// Package server provides network listener functionality
package server

import (
	"net"
)

// GetListener returns a listener for the given addr.
// Default implementation dials the addr normally.
func GetListener(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
