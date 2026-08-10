package engine

import (
	"strconv"
	"strings"
)

// VerTuple converts a version string into a slice of ints for correct
// numeric comparison (direct port of Python's _ver_tuple).
//
// Plain semver-ish strings compare component-by-component:
//
//	"1.68.2"  -> [1, 68, 2]
//	"1.68.10" -> [1, 68, 10]   (was a string-compare bug in earlier Python versions)
//
// wiserain fork versions carry a build number after a hyphen
// (e.g. "1.75.0-315"). That build number is NOT discarded — two builds
// that share the same rclone version but differ only in build number
// must still compare as different, otherwise update checks silently stop
// detecting new builds:
//
//	"1.75.0-315" -> [1, 75, 0, 315]
//	"1.75.0-306" -> [1, 75, 0, 306]
//
// Malformed input returns [0], matching the Python fallback.
func VerTuple(v string) []int {
	v = strings.TrimSpace(v)

	main := v
	buildNum := -1 // -1 means "no build suffix present"
	if idx := strings.Index(v, "-"); idx >= 0 {
		main = v[:idx]
		build := v[idx+1:]
		n, err := strconv.Atoi(build)
		if err != nil {
			n = 0
		}
		buildNum = n
	}

	parts := strings.Split(main, ".")
	out := make([]int, 0, len(parts)+1)
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return []int{0}
		}
		out = append(out, n)
	}
	if buildNum >= 0 {
		out = append(out, buildNum)
	}
	return out
}

// CompareVersions returns -1, 0, or 1 depending on whether a is less than,
// equal to, or greater than b, using VerTuple semantics. Shorter tuples are
// padded with zeros so e.g. "1.68" and "1.68.0-5" compare sensibly.
func CompareVersions(a, b string) int {
	ta, tb := VerTuple(a), VerTuple(b)
	n := len(ta)
	if len(tb) > n {
		n = len(tb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(ta) {
			x = ta[i]
		}
		if i < len(tb) {
			y = tb[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
