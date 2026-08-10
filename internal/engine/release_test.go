package engine

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestReleaseTag_StripsVPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name": "v1.68.2"}`))
	}))
	defer srv.Close()

	got, err := FetchLatestReleaseTag(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.68.2" {
		t.Errorf("FetchLatestReleaseTag = %q, want %q", got, "1.68.2")
	}
}

func TestFetchLatestReleaseTag_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden) // e.g. GitHub API rate-limited
	}))
	defer srv.Close()

	if _, err := FetchLatestReleaseTag(srv.Client(), srv.URL); err == nil {
		t.Errorf("expected an error on non-200 status")
	}
}

func TestParseLocalRcloneVersion_WithBuildSuffix(t *testing.T) {
	output := "rclone v1.74.4-302\n- os/version: windows 10\n- os/kernel: ...\n"
	got, ok := ParseLocalRcloneVersion(output)
	if !ok {
		t.Fatalf("expected a match")
	}
	if got != "1.74.4-302" {
		t.Errorf("ParseLocalRcloneVersion = %q, want %q", got, "1.74.4-302")
	}
}

func TestParseLocalRcloneVersion_NoMatch(t *testing.T) {
	if _, ok := ParseLocalRcloneVersion("not a version string"); ok {
		t.Errorf("expected no match")
	}
}
