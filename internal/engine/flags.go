package engine

import (
	"regexp"
	"strings"
)

// splitChunkRe finds every occurrence of "--" inside a chunk, used to break
// up strings like "--flag1 --flag2 --flag3 value" into individual flags.
var flagStartRe = regexp.MustCompile(`--`)

// flagWithSpaceRe matches "--flag value" (space-separated, no "=").
var flagWithSpaceRe = regexp.MustCompile(`^(--[\w-]+)\s+(\S+)$`)

// NormalizeFlags normalizes a raw extra-flags string into the canonical
// semicolon-separated, "--flag=value" form used for storage.
//
// Accepted input forms (all of these normalize the same way):
//
//	--flag=value;--flag2 value2
//	--flag value\n--flag2=value2
//	--flag1 --flag2 --flag3 value
//
// Output form: --flag=value;--flag2=value2;--flag3
//   - flags are separated by semicolons (safe even if a value contains spaces)
//   - space-separated "flag value" pairs are rewritten as "flag=value"
//   - flags with no value are kept as-is
//
// This is a direct behavioral port of the Python normalize_flags().
func NormalizeFlags(extra string) string {
	if strings.TrimSpace(extra) == "" {
		return ""
	}

	var flags []string

	for _, chunk := range regexp.MustCompile(`[;\n]+`).Split(extra, -1) {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}

		for _, part := range splitOnFlagBoundaries(chunk) {
			part = strings.TrimSpace(part)
			if part == "" || !strings.HasPrefix(part, "--") {
				continue
			}
			if strings.Contains(part, "=") {
				flags = append(flags, part)
				continue
			}
			if m := flagWithSpaceRe.FindStringSubmatch(part); m != nil {
				flags = append(flags, m[1]+"="+m[2])
			} else {
				flags = append(flags, part)
			}
		}
	}

	return strings.Join(flags, ";")
}

// splitOnFlagBoundaries splits chunk at every "--" boundary, keeping the
// "--" with the text that follows it (equivalent to Python's
// re.split(r'(?=--)', chunk)).
func splitOnFlagBoundaries(chunk string) []string {
	locs := flagStartRe.FindAllStringIndex(chunk, -1)
	if len(locs) == 0 {
		return []string{chunk}
	}

	var parts []string
	if locs[0][0] > 0 {
		parts = append(parts, chunk[:locs[0][0]])
	}
	for i, loc := range locs {
		start := loc[0]
		end := len(chunk)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		parts = append(parts, chunk[start:end])
	}
	return parts
}
