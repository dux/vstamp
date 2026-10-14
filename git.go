package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func refExists(ref string) bool {
	_, err := git("rev-parse", "-q", "--verify", ref)
	return err == nil
}

// countBranch picks the branch to count: the given name, else master, else main.
func countBranch(name string) (string, error) {
	if name != "" {
		if !refExists("refs/heads/" + name) {
			return "", fmt.Errorf("branch %q not found", name)
		}
		return name, nil
	}
	for _, b := range []string{"master", "main"} {
		if refExists("refs/heads/" + b) {
			return b, nil
		}
	}
	return "", errors.New("no master or main branch, pass --branch <name>")
}

func commitCount(branch string) (int, error) {
	out, err := git("rev-list", "--count", "refs/heads/"+branch)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

func currentBranch() string {
	out, _ := git("symbolic-ref", "-q", "--short", "HEAD")
	return out
}

// lastTag is the highest vX.Y.Z tag reachable from HEAD.
func lastTag() (Version, bool) {
	out, err := git("tag", "--merged", "HEAD", "--list", "v*", "--sort=-v:refname")
	if err != nil {
		return Version{}, false
	}
	for _, tag := range strings.Fields(out) {
		if v, err := ParseVersion(strings.TrimPrefix(tag, "v")); err == nil {
			return v, true
		}
	}
	return Version{}, false
}

func tagExists(tag string) bool {
	return refExists("refs/tags/" + tag)
}

func stagedFiles() ([]string, error) {
	out, err := git("diff", "--cached", "--name-only")
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

// upstream returns the remote and remote ref the branch tracks.
func upstream(branch string) (string, string, error) {
	remote, _ := git("config", "branch."+branch+".remote")
	merge, _ := git("config", "branch."+branch+".merge")
	if remote == "" || merge == "" {
		return "", "", fmt.Errorf("branch %q has no upstream, run: git push -u origin %s", branch, branch)
	}
	return remote, merge, nil
}
