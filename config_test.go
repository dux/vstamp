package main

import (
	"strings"
	"testing"
)

func TestParseVersionFile(t *testing.T) {
	f, err := parseVersionFile("# release info\nversion: 3.8.5  # stamped\noffset: 14\nname: fez\nextra:\n  version: 9.9.9\n  offset: x\n")
	if err != nil {
		t.Fatal(err)
	}
	if f.Version.String() != "3.8.5" || f.Offset != 14 {
		t.Fatalf("got %s offset %d", f.Version, f.Offset)
	}

	f, err = parseVersionFile("version: 0.0.1\r\n")
	if err != nil || f.Version.String() != "0.0.1" || f.Offset != 0 {
		t.Fatalf("crlf: %v %v", f, err)
	}
}

func TestParseVersionFileErrors(t *testing.T) {
	cases := map[string]string{
		"3.8.5\n":                              `missing "version: X.Y.Z" line`,
		"v380\n":                               `missing "version: X.Y.Z" line`,
		"name: x\n":                            `missing "version: X.Y.Z" line`,
		"version:3.8.5\n":                      `missing "version: X.Y.Z" line`,
		"version: \"3.8.5\"\n":                 `.version:1: invalid version`,
		"version: v3.8.5\n":                    `.version:1: invalid version`,
		"version:\n":                           `.version:1: invalid version`,
		"version: 1.0.0\nversion: 1.0.1":       `.version:2: duplicate "version"`,
		"version: 1.0.0\noffset: -1\n":         `.version:2: invalid offset "-1"`,
		"version: 1.0.0\noffset: abc\n":        `.version:2: invalid offset "abc"`,
		"version: 1.0.0\noffset: 014\n":        `.version:2: invalid offset "014"`,
		"version: 1.0.0\noffset:\n":            `.version:2: invalid offset ""`,
		"offset: 1\nversion: 1.0.0\noffset: 2": `.version:3: duplicate "offset"`,
	}
	for in, want := range cases {
		if _, err := parseVersionFile(in); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("parseVersionFile(%q) = %v, want %q", in, err, want)
		}
	}
}

func TestRenderVersionFile(t *testing.T) {
	v := Version{3, 8, 6}
	var missing *VersionFile
	if got := missing.render(v); got != "version: 3.8.6\n" {
		t.Fatalf("missing file: %q", got)
	}

	cases := map[string]string{
		"version: 3.8.5\n": "version: 3.8.6\n",
		"# header\nversion: 3.8.5   # stamped by vstamp\noffset: 14\nlist:\n  - version: 1.0.0\n": "# header\nversion: 3.8.6   # stamped by vstamp\noffset: 14\nlist:\n  - version: 1.0.0\n",
		"offset: 2\r\nversion: 3.8.5\r\n": "offset: 2\r\nversion: 3.8.6\r\n",
		"version: 3.8.5":                  "version: 3.8.6",
	}
	for in, want := range cases {
		f, err := parseVersionFile(in)
		if err != nil {
			t.Fatal(err)
		}
		if got := f.render(v); got != want {
			t.Errorf("render(%q) = %q, want %q", in, got, want)
		}
	}
}
