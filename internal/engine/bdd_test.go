package engine

import "testing"

// Scenario runs one BDD-style test case. name should read as a full
// GIVEN/WHEN/THEN sentence, so `go test -v` output doubles as living
// documentation of the behavior being verified — this satisfies the
// "테스트 가독성" checklist item without needing a separate spec doc.
func Scenario(t *testing.T, name string, run func(t *testing.T)) {
	t.Helper()
	t.Run(name, run)
}
