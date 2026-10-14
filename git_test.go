package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// isolate tests from the user's git config (hooks, signing, default branch)
func gitEnv(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"AUTHOR", "COMMITTER"} {
		t.Setenv("GIT_"+k+"_NAME", "Test")
		t.Setenv("GIT_"+k+"_EMAIL", "test@example.com")
	}
}

func sh(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepo creates a repo on branch with n empty commits and chdirs into it.
func newRepo(t *testing.T, branch string, n int) string {
	t.Helper()
	gitEnv(t)
	dir := t.TempDir()
	sh(t, dir, "init", "-q", "-b", branch)
	commits(t, dir, n)
	t.Chdir(dir)
	return dir
}

func commits(t *testing.T, dir string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		sh(t, dir, "commit", "-q", "--allow-empty", "-m", "work")
	}
}

// withRemote adds a bare origin and pushes branch with upstream tracking.
func withRemote(t *testing.T, dir, branch string) string {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "origin.git")
	sh(t, dir, "init", "-q", "--bare", "-b", branch, remote)
	sh(t, dir, "remote", "add", "origin", remote)
	sh(t, dir, "push", "-q", "-u", "origin", branch)
	return remote
}

func vstamp(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return stdout.String(), stderr.String(), code
}

func mustVstamp(t *testing.T, args ...string) string {
	t.Helper()
	out, errOut, code := vstamp(t, args...)
	if code != 0 {
		t.Fatalf("vstamp %v: exit %d\n%s%s", args, code, out, errOut)
	}
	return out
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeFile(t *testing.T, name, body string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestShowWithoutVersionFile(t *testing.T) {
	newRepo(t, "main", 1)
	if out := mustVstamp(t); out != "dev\n" {
		t.Fatalf("got %q", out)
	}
}

func TestInvalidVersionFile(t *testing.T) {
	newRepo(t, "main", 1)
	writeFile(t, ".version", "3.8.0\n")
	if _, errOut, code := vstamp(t, "count"); code != 1 || !strings.Contains(errOut, `missing "version: X.Y.Z" line`) {
		t.Fatalf("exit %d: %s", code, errOut)
	}
}

func TestCountMasterBeforeMain(t *testing.T) {
	dir := newRepo(t, "master", 3)
	sh(t, dir, "switch", "-q", "-c", "main")
	commits(t, dir, 2)
	if out := mustVstamp(t, "count"); out != "none -> 0.0.3\n" {
		t.Fatalf("got %q", out)
	}
	if got := readFile(t, ".version"); got != "version: 0.0.3\n" {
		t.Fatalf(".version = %q", got)
	}
}

func TestCountMainFallback(t *testing.T) {
	newRepo(t, "main", 12)
	mustVstamp(t, "count")
	if got := readFile(t, ".version"); got != "version: 0.1.2\n" {
		t.Fatalf(".version = %q", got)
	}
}

func TestCountBranchFlag(t *testing.T) {
	dir := newRepo(t, "master", 2)
	sh(t, dir, "switch", "-q", "-c", "develop")
	commits(t, dir, 9)
	mustVstamp(t, "count", "--branch", "develop")
	if got := readFile(t, ".version"); got != "version: 0.1.1\n" {
		t.Fatalf(".version = %q", got)
	}
	if _, errOut, code := vstamp(t, "count", "--branch", "nope"); code != 1 || !strings.Contains(errOut, `branch "nope" not found`) {
		t.Fatalf("exit %d: %s", code, errOut)
	}
}

func TestCountNoDefaultBranch(t *testing.T) {
	newRepo(t, "trunk", 1)
	if _, errOut, code := vstamp(t, "count"); code != 1 || !strings.Contains(errOut, "no master or main branch") {
		t.Fatalf("exit %d: %s", code, errOut)
	}
}

func TestCountBelowCurrentNeedsOffset(t *testing.T) {
	newRepo(t, "master", 10)
	writeFile(t, ".version", "version: 0.2.3\n")
	_, errOut, code := vstamp(t, "count")
	if code != 1 || !strings.Contains(errOut, `set "offset: 14" in .version`) {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if got := readFile(t, ".version"); got != "version: 0.2.3\n" {
		t.Fatalf(".version changed to %q", got)
	}
	writeFile(t, ".version", "# gem version\nversion: 0.2.3\noffset: 14\nowner: dux\n")
	if out := mustVstamp(t, "count"); out != "0.2.3 -> 0.2.4\n" {
		t.Fatalf("got %q", out)
	}
	if got := readFile(t, ".version"); got != "# gem version\nversion: 0.2.4\noffset: 14\nowner: dux\n" {
		t.Fatalf(".version = %q", got)
	}
}

func TestNextCountsPendingCommit(t *testing.T) {
	dir := newRepo(t, "main", 5)
	if out := mustVstamp(t, "count", "--next"); out != "none -> 0.0.6\n" {
		t.Fatalf("got %q", out)
	}
	if got := sh(t, dir, "rev-list", "--count", "main"); got != "5" {
		t.Fatalf("commit count = %s", got)
	}
	// a script committing after --next lands on the stamped count
	sh(t, dir, "add", ".version")
	sh(t, dir, "commit", "-q", "-m", "release")
	if out := mustVstamp(t, "count"); out != "0.0.6 -> 0.0.6\n" {
		t.Fatalf("got %q", out)
	}
}

func TestCommitCountsItself(t *testing.T) {
	dir := newRepo(t, "main", 5)
	writeFile(t, "package.json", "{\n  \"name\": \"x\",\n  \"version\": \"0.0.1\"\n}\n")
	sh(t, dir, "add", "package.json")
	sh(t, dir, "commit", "-q", "-m", "pkg")
	mustVstamp(t, "count", "--commit", "--tag")
	if got := readFile(t, ".version"); got != "version: 0.0.7\n" {
		t.Fatalf(".version = %q", got)
	}
	if got := sh(t, dir, "rev-list", "--count", "main"); got != "7" {
		t.Fatalf("commit count = %s", got)
	}
	if got := sh(t, dir, "log", "-1", "--format=%s"); got != "release 0.0.7" {
		t.Fatalf("message = %q", got)
	}
	if got := sh(t, dir, "rev-parse", "v0.0.7"); got != sh(t, dir, "rev-parse", "HEAD") {
		t.Fatal("tag does not point at HEAD")
	}
	if got := sh(t, dir, "show", "HEAD:package.json"); !strings.Contains(got, `"version": "0.0.7"`) {
		t.Fatalf("package.json not committed: %s", got)
	}
	if got := sh(t, dir, "status", "--porcelain"); got != "" {
		t.Fatalf("dirty tree: %s", got)
	}
}

func TestAmendKeepsCountAndMessage(t *testing.T) {
	dir := newRepo(t, "main", 4)
	sh(t, dir, "commit", "-q", "--allow-empty", "-m", "feature x")
	mustVstamp(t, "count", "--amend", "--tag")
	if got := sh(t, dir, "rev-list", "--count", "main"); got != "5" {
		t.Fatalf("commit count = %s", got)
	}
	if got := sh(t, dir, "log", "-1", "--format=%s"); got != "feature x" {
		t.Fatalf("message = %q", got)
	}
	if got := sh(t, dir, "show", "v0.0.5:.version"); got != "version: 0.0.5" {
		t.Fatalf("tagged .version = %q", got)
	}
}

func TestExistingTagAbortsBeforeWrite(t *testing.T) {
	dir := newRepo(t, "main", 5)
	sh(t, dir, "tag", "v0.0.5")
	head := sh(t, dir, "rev-parse", "HEAD")
	_, errOut, code := vstamp(t, "count", "--amend", "--tag")
	if code != 1 || !strings.Contains(errOut, "tag v0.0.5 already exists") {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if _, err := os.Stat(".version"); err == nil {
		t.Fatal(".version written")
	}
	if sh(t, dir, "rev-parse", "HEAD") != head {
		t.Fatal("HEAD moved")
	}
}

func TestForeignStagedFileAborts(t *testing.T) {
	dir := newRepo(t, "main", 2)
	writeFile(t, "other.txt", "x")
	sh(t, dir, "add", "other.txt")
	if _, errOut, code := vstamp(t, "count", "--amend"); code != 1 || !strings.Contains(errOut, "other.txt is staged") {
		t.Fatalf("exit %d: %s", code, errOut)
	}
}

func TestCommitOnOtherBranchAborts(t *testing.T) {
	dir := newRepo(t, "main", 2)
	sh(t, dir, "switch", "-q", "-c", "feature")
	if _, errOut, code := vstamp(t, "count", "--amend"); code != 1 || !strings.Contains(errOut, `but counting "main"`) {
		t.Fatalf("exit %d: %s", code, errOut)
	}
}

func TestFlagRules(t *testing.T) {
	newRepo(t, "main", 1)
	for _, args := range [][]string{
		{"count", "--commit", "--amend"},
		{"count", "--tag"},
		{"count", "--push"},
		{"count", "--offset", "5"},
		{"count", "--next", "--amend"},
		{"count", "--release", "--next"},
		{"count", "extra"},
		{"count", "--nope"},
		{"bogus"},
	} {
		if _, _, code := vstamp(t, args...); code != 1 {
			t.Errorf("vstamp %v: exit %d, want 1", args, code)
		}
	}
}

func TestTagCommandAndDrift(t *testing.T) {
	dir := newRepo(t, "main", 3)
	if out, _, code := vstamp(t, "tag"); code != 1 || out != "none\n" {
		t.Fatalf("exit %d: %q", code, out)
	}
	sh(t, dir, "tag", "v0.0.2")
	sh(t, dir, "tag", "v0.0.10")
	sh(t, dir, "tag", "v1.0.0-rc1")
	if out := mustVstamp(t, "tag"); out != "0.0.10\n" {
		t.Fatalf("got %q", out)
	}
	writeFile(t, ".version", "version: 0.0.3\n")
	out, errOut, code := vstamp(t)
	if code != 0 || out != "0.0.3\n" || !strings.Contains(errOut, "warning: .version 0.0.3 is behind tag v0.0.10") {
		t.Fatalf("exit %d: %q %q", code, out, errOut)
	}
}

func TestReleaseDryRun(t *testing.T) {
	dir := newRepo(t, "main", 3)
	withRemote(t, dir, "main")
	head := sh(t, dir, "rev-parse", "HEAD")
	out := mustVstamp(t, "count", "--release", "--dry-run")
	for _, want := range []string{
		"none -> 0.0.3 (dry run)",
		"git add -- .version",
		"git commit --amend --no-edit",
		"git tag v0.0.3",
		"git push --atomic --force-with-lease --force-if-includes origin HEAD:refs/heads/main refs/tags/v0.0.3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if _, err := os.Stat(".version"); err == nil {
		t.Fatal(".version written")
	}
	if sh(t, dir, "rev-parse", "HEAD") != head || sh(t, dir, "tag") != "" {
		t.Fatal("git state changed")
	}
}

func TestReleaseRequiresUpstream(t *testing.T) {
	newRepo(t, "main", 2)
	if _, errOut, code := vstamp(t, "count", "--release"); code != 1 || !strings.Contains(errOut, "no upstream") {
		t.Fatalf("exit %d: %s", code, errOut)
	}
}

func TestReleasePushesAmendedHeadAndTag(t *testing.T) {
	dir := newRepo(t, "main", 3)
	remote := withRemote(t, dir, "main")
	mustVstamp(t, "count", "--release")
	head := sh(t, dir, "rev-parse", "HEAD")
	if got := sh(t, remote, "rev-parse", "main"); got != head {
		t.Fatalf("remote main = %s, want %s", got, head)
	}
	if got := sh(t, remote, "rev-parse", "v0.0.3^{commit}"); got != head {
		t.Fatalf("remote tag = %s, want %s", got, head)
	}

	// next release: new local commit, amended and pushed again
	commits(t, dir, 1)
	mustVstamp(t, "count", "--release")
	if got := sh(t, remote, "show", "main:.version"); got != "version: 0.0.4" {
		t.Fatalf("remote .version = %q", got)
	}
}

func TestReleaseRejectsMovedRemote(t *testing.T) {
	// fetch=true: a background fetch refreshed origin/main, so only
	// --force-if-includes stands between us and overwriting their commit
	for _, fetch := range []bool{false, true} {
		t.Run(map[bool]string{false: "stale lease", true: "after fetch"}[fetch], func(t *testing.T) {
			dir := newRepo(t, "main", 3)
			remote := withRemote(t, dir, "main")

			other := filepath.Join(t.TempDir(), "other")
			sh(t, dir, "clone", "-q", remote, other)
			commits(t, other, 1)
			sh(t, other, "push", "-q", "origin", "main")
			theirs := sh(t, remote, "rev-parse", "main")
			if fetch {
				sh(t, dir, "fetch", "-q")
			}

			_, errOut, code := vstamp(t, "count", "--release")
			if code != 1 || !strings.Contains(errOut, "retry with: git push") {
				t.Fatalf("exit %d: %s", code, errOut)
			}
			if got := sh(t, remote, "rev-parse", "main"); got != theirs {
				t.Fatal("remote main was overwritten")
			}
			if out := sh(t, remote, "tag"); out != "" {
				t.Fatalf("tag pushed despite atomic failure: %s", out)
			}
		})
	}
}
