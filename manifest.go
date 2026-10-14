package main

import (
	"errors"
	"io/fs"
	"os"
)

const manifestFile = "package.json"

// syncManifest rewrites the top-level "version" of package.json in place,
// leaving formatting and key order alone. Reports whether the file changed.
func syncManifest(v Version, dryRun bool) (bool, error) {
	data, err := os.ReadFile(manifestFile)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	out, ok := setManifestVersion(data, v.String())
	if !ok || string(out) == string(data) {
		return false, nil
	}
	if dryRun {
		return true, nil
	}
	return true, os.WriteFile(manifestFile, out, 0o644)
}

func setManifestVersion(data []byte, version string) ([]byte, bool) {
	start, end, ok := versionSpan(data)
	if !ok {
		return data, false
	}
	out := make([]byte, 0, len(data)+len(version))
	out = append(out, data[:start]...)
	out = append(out, version...)
	return append(out, data[end:]...), true
}

// versionSpan finds the contents of the string value of the top-level
// "version" key, so nested objects (engines, dependencies) are never touched.
func versionSpan(data []byte) (int, int, bool) {
	depth, key := 0, false
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '{', '[':
			depth++
			key = depth == 1 && data[i] == '{'
		case '}', ']':
			depth--
			key = false
		case ',':
			key = depth == 1
		case '"':
			j := stringEnd(data, i)
			if j < 0 {
				return 0, 0, false
			}
			if key && string(data[i+1:j]) == "version" {
				k := skipSpace(data, j+1)
				if k < len(data) && data[k] == ':' {
					k = skipSpace(data, k+1)
					if k < len(data) && data[k] == '"' {
						if e := stringEnd(data, k); e > 0 {
							return k + 1, e, true
						}
					}
				}
			}
			key = false
			i = j
		}
	}
	return 0, 0, false
}

// stringEnd returns the index of the closing quote of the string opened at i.
func stringEnd(data []byte, i int) int {
	for j := i + 1; j < len(data); j++ {
		switch data[j] {
		case '\\':
			j++
		case '"':
			return j
		}
	}
	return -1
}

func skipSpace(data []byte, i int) int {
	for i < len(data) && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
		i++
	}
	return i
}
