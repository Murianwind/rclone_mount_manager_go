package engine

import "testing"

// ── keywords ──

func givenMount(m Mount) Mount { return m }

func whenBuildingCmd(exe string, m Mount) []string { return BuildCmd(exe, m) }

func whenResolvingVolname(m Mount) string { return GetVolname(m) }

func thenCmdContains(t *testing.T, cmd []string, want string) {
	t.Helper()
	if !containsArg(cmd, want) {
		t.Errorf("expected cmd to contain %q, got %v", want, cmd)
	}
}

func thenCmdDoesNotContain(t *testing.T, cmd []string, unwanted string) {
	t.Helper()
	if containsArg(cmd, unwanted) {
		t.Errorf("did not expect cmd to contain %q, got %v", unwanted, cmd)
	}
}

func thenCmdContainsExactlyOnce(t *testing.T, cmd []string, want string) {
	t.Helper()
	if got := countArg(cmd, want); got != 1 {
		t.Errorf("expected exactly one %q, got %d occurrences in %v", want, got, cmd)
	}
}

func thenVolnameEquals(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got volname %q, want %q", got, want)
	}
}

func containsArg(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func countArg(list []string, item string) int {
	n := 0
	for _, v := range list {
		if v == item {
			n++
		}
	}
	return n
}

func TestBuildCmd(t *testing.T) {
	Scenario(t, "GIVEN a basic mount WHEN the command is built THEN it mounts remote:path to the drive", func(t *testing.T) {
		m := givenMount(Mount{Remote: "drive", Drive: "X:", RemotePath: "data"})
		cmd := whenBuildingCmd("rclone.exe", m)
		thenCmdContains(t, cmd, "mount")
		thenCmdContains(t, cmd, "drive:data")
	})

	Scenario(t, "GIVEN a mount with cache settings WHEN the command is built THEN cache flags are included", func(t *testing.T) {
		m := givenMount(Mount{Remote: "drive", Drive: "X:", CacheDir: `C:\cache`, CacheMode: "full"})
		cmd := whenBuildingCmd("rclone.exe", m)
		thenCmdContains(t, cmd, "--cache-dir")
		thenCmdContains(t, cmd, "full")
	})

	Scenario(t, "GIVEN a mount with no cache settings WHEN the command is built THEN no cache flags are added (negative case)", func(t *testing.T) {
		m := givenMount(Mount{Remote: "drive", Drive: "X:"})
		cmd := whenBuildingCmd("rclone.exe", m)
		thenCmdDoesNotContain(t, cmd, "--cache-dir")
		thenCmdDoesNotContain(t, cmd, "--vfs-cache-mode")
	})

	Scenario(t, "GIVEN extra flags in space-separated form WHEN normalized and built THEN they appear as --flag=value", func(t *testing.T) {
		m := givenMount(Mount{
			Remote:     "drive",
			Drive:      "X:",
			ExtraFlags: NormalizeFlags("--read-only; --bwlimit 10M"),
		})
		cmd := whenBuildingCmd("rclone.exe", m)
		thenCmdContains(t, cmd, "--read-only")
		thenCmdContains(t, cmd, "--bwlimit=10M")
	})

	Scenario(t, "GIVEN extra flags that redundantly set --volname WHEN the command is built THEN only the auto-resolved --volname survives (no duplicate)", func(t *testing.T) {
		m := givenMount(Mount{
			Remote: "gds", Drive: "G:", RemotePath: "GDRIVE/VIDEO",
			ExtraFlags: "--volname=GDS;--no-modtime",
		})
		cmd := whenBuildingCmd("rclone.exe", m)
		thenCmdContainsExactlyOnce(t, cmd, "--volname")
		thenCmdContains(t, cmd, "GDS")
	})
}

func TestGetVolname(t *testing.T) {
	Scenario(t, "GIVEN an explicit --volname in extra flags WHEN resolved THEN that value wins over any other rule", func(t *testing.T) {
		m := givenMount(Mount{
			Remote: "gds", RemotePath: "GDRIVE/VIDEO",
			ExtraFlags: "--buffer-size=512M;--volname=GDS;--no-modtime",
		})
		got := whenResolvingVolname(m)
		thenVolnameEquals(t, got, "GDS")
	})

	Scenario(t, "GIVEN no explicit --volname but a remote sub-path WHEN resolved THEN the last path segment is used", func(t *testing.T) {
		m := givenMount(Mount{Remote: "PLEX", RemotePath: "KODI"})
		got := whenResolvingVolname(m)
		thenVolnameEquals(t, got, "KODI")
	})

	Scenario(t, "GIVEN no --volname and no sub-path WHEN resolved THEN the remote name itself is used as a fallback", func(t *testing.T) {
		m := givenMount(Mount{Remote: "nas"})
		got := whenResolvingVolname(m)
		thenVolnameEquals(t, got, "nas")
	})
}

func TestNewMountID(t *testing.T) {
	Scenario(t, "GIVEN two separate calls WHEN new IDs are minted THEN they are non-empty and distinct", func(t *testing.T) {
		a := NewMountID()
		b := NewMountID()
		if a == "" || b == "" {
			t.Fatalf("expected non-empty IDs, got %q and %q", a, b)
		}
		if a == b {
			t.Errorf("expected two calls to produce different IDs")
		}
	})
}
