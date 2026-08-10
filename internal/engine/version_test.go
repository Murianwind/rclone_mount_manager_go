package engine

import (
	"reflect"
	"testing"
)

func TestVerTuple_Malformed(t *testing.T) {
	got := VerTuple("not-a-version")
	want := []int{0}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("VerTuple(\"not-a-version\") = %v, want %v", got, want)
	}
}

func TestVerTuple_PlainVersion(t *testing.T) {
	got := VerTuple("1.68.10")
	want := []int{1, 68, 10}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("VerTuple = %v, want %v", got, want)
	}
}

func TestVerTuple_BuildSuffixKept(t *testing.T) {
	got := VerTuple("1.74.0-297")
	want := []int{1, 74, 0, 297}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("VerTuple = %v, want %v", got, want)
	}
	if CompareVersions("1.73.5", "1.74.0-297") != -1 {
		t.Errorf("expected 1.73.5 < 1.74.0-297")
	}
}

// Regression test for the real bug: same rclone version, only the build
// number changed (1.75.0-306 -> 1.75.0-315). The old logic discarded the
// build number and treated these as identical, so updates were never
// detected.
func TestVerTuple_SameVersionDifferentBuildIsDetected(t *testing.T) {
	loc := VerTuple("1.75.0-306")
	lat := VerTuple("1.75.0-315")
	if reflect.DeepEqual(loc, lat) {
		t.Fatalf("versions with different build numbers must not be equal")
	}
	if CompareVersions("1.75.0-306", "1.75.0-315") != -1 {
		t.Errorf("expected 1.75.0-306 < 1.75.0-315")
	}
}

func TestNoStringCompareBug(t *testing.T) {
	// The historical bug: "1.68.2" < "1.68.10" as strings is false,
	// but numerically 2 < 10 is true.
	if CompareVersions("1.68.2", "1.68.10") != -1 {
		t.Errorf("expected 1.68.2 < 1.68.10 numerically")
	}
}
