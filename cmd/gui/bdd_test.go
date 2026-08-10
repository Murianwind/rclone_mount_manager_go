package main

import "testing"

// Scenario runs one BDD-style test case. name should read as a full
// GIVEN/WHEN/THEN sentence — same convention as internal/engine's tests.
func Scenario(t *testing.T, name string, run func(t *testing.T)) {
	t.Helper()
	t.Run(name, run)
}
