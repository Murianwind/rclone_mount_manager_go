package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// RcloneReleaseAPI / AppReleaseAPI mirror the two GitHub "latest release"
// endpoints polled by _check_versions_async(): rclone's wiserain fork, and
// this app's own repo.
const (
	RcloneReleaseAPI = "https://api.github.com/repos/wiserain/rclone/releases/latest"
	AppReleaseAPI    = "https://api.github.com/repos/Murianwind/rclone_mount_manager_go/releases/latest"
)

// githubRelease is the subset of the GitHub "latest release" response this
// package needs.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// FetchLatestReleaseTag calls a GitHub "releases/latest"-shaped API URL
// and returns the tag name with any leading "v" stripped (e.g. "v1.68.2"
// -> "1.68.2"). client may be nil to use http.DefaultClient.
func FetchLatestReleaseTag(client *http.Client, apiURL string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, apiURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", err
	}
	return strings.TrimPrefix(rel.TagName, "v"), nil
}

var rcloneVersionOutputRe = regexp.MustCompile(`rclone v([\d.\-]+)`)

// ParseLocalRcloneVersion extracts the version string from `rclone
// version`'s stdout (e.g. "rclone v1.74.4-302\n- os/version: ..." ->
// "1.74.4-302"). The hyphenated build suffix used by the wiserain fork is
// kept, matching the comment in _check_versions_async: capturing the full
// string keeps the displayed version aligned with the GitHub tag, while
// VerTuple() is what actually strips/compares the build number.
func ParseLocalRcloneVersion(rcloneVersionOutput string) (version string, ok bool) {
	m := rcloneVersionOutputRe.FindStringSubmatch(rcloneVersionOutput)
	if m == nil {
		return "", false
	}
	return m[1], true
}
