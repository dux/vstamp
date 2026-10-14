package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

const versionFile = ".version"

// VersionFile is .version, a YAML file. vstamp owns the top-level `version`
// and `offset` keys; every other line is left untouched on write.
type VersionFile struct {
	Version Version
	Offset  int
	raw     string
}

// readVersionFile returns nil when .version does not exist.
func readVersionFile() (*VersionFile, error) {
	data, err := os.ReadFile(versionFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseVersionFile(string(data))
}

func parseVersionFile(raw string) (*VersionFile, error) {
	f := &VersionFile{raw: raw}
	seen := map[string]bool{}
	for i, line := range strings.Split(raw, "\n") {
		key, value, _, ok := keyLine(line)
		if !ok {
			continue
		}
		if seen[key] {
			return nil, fmt.Errorf("%s:%d: duplicate %q", versionFile, i+1, key)
		}
		seen[key] = true
		switch key {
		case "version":
			v, err := ParseVersion(value)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", versionFile, i+1, err)
			}
			f.Version = v
		case "offset":
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || value != strconv.Itoa(n) {
				return nil, fmt.Errorf("%s:%d: invalid offset %q, expected a non-negative integer", versionFile, i+1, value)
			}
			f.Offset = n
		}
	}
	if !seen["version"] {
		return nil, fmt.Errorf(`%s: missing "version: X.Y.Z" line`, versionFile)
	}
	return f, nil
}

// keyLine splits a top-level `version:` or `offset:` line into its value and
// trailing comment (with the spaces before it).
func keyLine(line string) (key, value, comment string, ok bool) {
	line = strings.TrimSuffix(line, "\r")
	for _, k := range []string{"version", "offset"} {
		if rest, found := strings.CutPrefix(line, k+":"); found {
			if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
				return "", "", "", false
			}
			for i := 1; i < len(rest); i++ {
				if rest[i] == '#' && (rest[i-1] == ' ' || rest[i-1] == '\t') {
					body := strings.TrimRight(rest[:i], " \t")
					rest, comment = body, rest[len(body):]
					break
				}
			}
			return k, strings.TrimSpace(rest), comment, true
		}
	}
	return "", "", "", false
}

// render returns the file with the version line set to v. A missing file
// becomes a single version line.
func (f *VersionFile) render(v Version) string {
	if f == nil {
		return "version: " + v.String() + "\n"
	}
	lines := strings.Split(f.raw, "\n")
	for i, line := range lines {
		if key, _, comment, ok := keyLine(line); ok && key == "version" {
			cr := ""
			if strings.HasSuffix(line, "\r") {
				cr = "\r"
			}
			lines[i] = "version: " + v.String() + comment + cr
		}
	}
	return strings.Join(lines, "\n")
}

func writeVersionFile(f *VersionFile, v Version) error {
	return os.WriteFile(versionFile, []byte(f.render(v)), 0o644)
}
