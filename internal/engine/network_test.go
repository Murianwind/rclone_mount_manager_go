package engine

import (
	"net"
	"testing"
	"time"
)

func TestIsInternetAvailable(t *testing.T) {
	Scenario(t, "GIVEN a reachable host WHEN connectivity is checked THEN it reports available", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to start test listener: %v", err)
		}
		defer ln.Close()
		go func() {
			for {
				c, err := ln.Accept()
				if err != nil {
					return
				}
				c.Close()
			}
		}()

		addr := ln.Addr().(*net.TCPAddr)
		if !IsInternetAvailable("127.0.0.1", addr.Port, time.Second) {
			t.Errorf("expected IsInternetAvailable to return true against a live listener")
		}
	})

	// Negative case: this is what the network-monitor's "disconnected"
	// branch (unmount everything) depends on being detected correctly.
	Scenario(t, "GIVEN nothing listening on the target port (network disconnected) WHEN connectivity is checked THEN it reports unavailable", func(t *testing.T) {
		if IsInternetAvailable("127.0.0.1", 1, 200*time.Millisecond) {
			t.Errorf("expected IsInternetAvailable to return false with nothing listening")
		}
	})
}
