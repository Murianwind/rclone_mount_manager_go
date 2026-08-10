package engine

import (
	"fmt"
	"net"
	"time"
)

// IsInternetAvailable checks connectivity by attempting a TCP connection
// to host:port (defaults to Google DNS, 8.8.8.8:53, in the Python version).
// Mirrors is_internet_available().
func IsInternetAvailable(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
