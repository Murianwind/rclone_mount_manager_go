package engine

import (
	"net"
	"testing"
	"time"
)

func TestIsInternetAvailable_True(t *testing.T) {
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
}

func TestIsInternetAvailable_False(t *testing.T) {
	// Port 0 on an address with nothing listening should fail fast.
	if IsInternetAvailable("127.0.0.1", 1, 200*time.Millisecond) {
		t.Errorf("expected IsInternetAvailable to return false with nothing listening")
	}
}
