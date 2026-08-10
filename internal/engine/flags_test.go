package engine

import "testing"

// ── keywords ──
// Small, named steps so each scenario reads as a script rather than raw
// assertions — the "키워드 주도" half of the convention. These are
// intentionally trivial; the value is in the vocabulary, not the logic.

func givenRawExtraFlags(s string) string { return s }

func whenNormalized(raw string) string { return NormalizeFlags(raw) }

func thenNormalizesTo(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeFlags(t *testing.T) {
	Scenario(t, "GIVEN an empty flags string WHEN normalized THEN it stays empty", func(t *testing.T) {
		raw := givenRawExtraFlags("")
		got := whenNormalized(raw)
		thenNormalizesTo(t, got, "")
	})

	Scenario(t, "GIVEN a whitespace-only flags string WHEN normalized THEN it becomes empty", func(t *testing.T) {
		raw := givenRawExtraFlags("   ")
		got := whenNormalized(raw)
		thenNormalizesTo(t, got, "")
	})

	Scenario(t, "GIVEN flags that already have no value WHEN normalized THEN they pass through unchanged", func(t *testing.T) {
		raw := givenRawExtraFlags("--links;--fast-list")
		got := whenNormalized(raw)
		thenNormalizesTo(t, got, "--links;--fast-list")
	})

	Scenario(t, "GIVEN a space-separated flag and value WHEN normalized THEN it becomes the --flag=value form", func(t *testing.T) {
		raw := givenRawExtraFlags("--read-only; --bwlimit 10M")
		got := whenNormalized(raw)
		thenNormalizesTo(t, got, "--read-only;--bwlimit=10M")
	})

	Scenario(t, "GIVEN multiple flags packed into one semicolon-separated chunk WHEN normalized THEN each becomes its own entry", func(t *testing.T) {
		raw := givenRawExtraFlags("--no-modtime --no-checksum")
		got := whenNormalized(raw)
		thenNormalizesTo(t, got, "--no-modtime;--no-checksum")
	})

	Scenario(t, "GIVEN newline-separated flags (as pasted from a multi-line box) WHEN normalized THEN they're joined with semicolons", func(t *testing.T) {
		raw := givenRawExtraFlags("--flag value\n--flag2=value2")
		got := whenNormalized(raw)
		thenNormalizesTo(t, got, "--flag=value;--flag2=value2")
	})
}
