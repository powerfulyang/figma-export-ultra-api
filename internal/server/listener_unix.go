//go:build linux || darwin

package server

import (
	"errors"
	"net"
	"os"
	"strconv"
)

// GetListener supports systemd socket activation if LISTEN_FDS=1 for current PID.
// Otherwise it falls back to net.Listen on addr.
func GetListener(addr string) (net.Listener, error) {
	if os.Getenv("SOCKET_ACTIVATION") == "1" {
		// Simple systemd-compatible env check
		if os.Getenv("LISTEN_FDS") == "1" {
			pidStr := os.Getenv("LISTEN_PID")
			if pidStr != "" {
				if pid, _ := strconv.Atoi(pidStr); pid != os.Getpid() {
					// Not for us
				} else {
					f := os.NewFile(uintptr(3), "listener") // SD_LISTEN_FDS_START = 3
					if f != nil {
						ln, err := net.FileListener(f)
						if err == nil {
							return ln, nil
						}
					}
				}
			}
		}
		return nil, errors.New("socket activation requested but no valid LISTEN_FDS")
	}
	return net.Listen("tcp", addr)
}
