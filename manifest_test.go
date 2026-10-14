package main

import "testing"

func TestSetManifestVersion(t *testing.T) {
	in := `{
  "name": "version",
  "description": "has \"version\": \"9.9.9\" inside",
  "engines": { "version": "1.0.0" },
  "files": ["version"],
  "version" : "0.1.0",
  "dependencies": {"version": "^2.0.0"}
}
`
	want := `{
  "name": "version",
  "description": "has \"version\": \"9.9.9\" inside",
  "engines": { "version": "1.0.0" },
  "files": ["version"],
  "version" : "3.8.5",
  "dependencies": {"version": "^2.0.0"}
}
`
	out, ok := setManifestVersion([]byte(in), "3.8.5")
	if !ok || string(out) != want {
		t.Fatalf("got ok=%v\n%s", ok, out)
	}
}

func TestSetManifestVersionMissing(t *testing.T) {
	in := `{"name": "x", "engines": {"version": "1.0.0"}}`
	if out, ok := setManifestVersion([]byte(in), "3.8.5"); ok || string(out) != in {
		t.Fatalf("changed manifest without top-level version: %s", out)
	}
}
