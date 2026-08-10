package engine

import (
	"reflect"
	"testing"
)

// ── keywords ──

func givenVersionString(s string) string { return s }

func whenParsedAsTuple(v string) []int { return VerTuple(v) }

func whenCompared(a, b string) int { return CompareVersions(a, b) }

func thenTupleEquals(t *testing.T, got, want []int) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func thenLessThan(t *testing.T, cmp int) {
	t.Helper()
	if cmp != -1 {
		t.Errorf("expected the first version to compare less than the second (-1), got %d", cmp)
	}
}

func TestVerTuple(t *testing.T) {
	Scenario(t, "GIVEN a malformed version string WHEN parsed THEN it falls back to [0] (negative case)", func(t *testing.T) {
		v := givenVersionString("not-a-version")
		got := whenParsedAsTuple(v)
		thenTupleEquals(t, got, []int{0})
	})

	Scenario(t, "GIVEN a plain three-part version WHEN parsed THEN each component becomes an int", func(t *testing.T) {
		v := givenVersionString("1.68.10")
		got := whenParsedAsTuple(v)
		thenTupleEquals(t, got, []int{1, 68, 10})
	})

	Scenario(t, "GIVEN a wiserain-fork version with a build suffix WHEN parsed THEN the build number is kept, not discarded", func(t *testing.T) {
		v := givenVersionString("1.74.0-297")
		got := whenParsedAsTuple(v)
		thenTupleEquals(t, got, []int{1, 74, 0, 297})
	})
}

func TestCompareVersions(t *testing.T) {
	Scenario(t, "GIVEN two versions differing only by a build suffix WHEN compared THEN the one with the build suffix is greater", func(t *testing.T) {
		cmp := whenCompared("1.73.5", "1.74.0-297")
		thenLessThan(t, cmp)
	})

	// Regression case: same rclone version, only the build number changed
	// (1.75.0-306 -> 1.75.0-315). The old logic discarded the build number
	// and treated these as identical, so update checks never fired.
	Scenario(t, "GIVEN the same version with only the build number bumped WHEN compared THEN the newer build still wins", func(t *testing.T) {
		older := whenParsedAsTuple("1.75.0-306")
		newer := whenParsedAsTuple("1.75.0-315")
		if reflect.DeepEqual(older, newer) {
			t.Fatalf("versions with different build numbers must not be considered equal")
		}
		cmp := whenCompared("1.75.0-306", "1.75.0-315")
		thenLessThan(t, cmp)
	})

	// Regression case: the historical string-compare bug. "1.68.2" <
	// "1.68.10" as strings is false, but numerically 2 < 10 is true.
	Scenario(t, "GIVEN a version whose last component has more digits WHEN compared THEN it compares numerically, not lexically", func(t *testing.T) {
		cmp := whenCompared("1.68.2", "1.68.10")
		thenLessThan(t, cmp)
	})
}
