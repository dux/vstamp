package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

// version is set by release builds: -ldflags "-X main.version=X.Y.Z".
var version string

const usage = `vstamp - keep the version in .version in sync with the git commit count

Usage:
  vstamp                 print the version from .version (dev when missing)
  vstamp tag             print the last vX.Y.Z tag reachable from HEAD (none, exit 1)
  vstamp count [flags]   set the version from the commit count on master (else main)
                         1111 commits -> 11.1.1, 380 -> 3.8.0, 5 -> 0.0.5
  vstamp --version       print vstamp version

Count flags:
  --branch <name>  count this branch instead of master/main
  --next           count the commit about to be made (+1), for scripts that commit later
  --commit         commit .version (+ package.json) as "release X.Y.Z" (implies --next)
  --amend          fold .version (+ package.json) into HEAD, keep its message
  --tag            tag vX.Y.Z (needs --commit or --amend, fails if it exists)
  --push           push branch (+ tag) atomically with --force-with-lease
  --release        same as --amend --tag --push
  --dry-run        print the change and git commands, run nothing

.version (YAML):
  version: 3.8.5   written by vstamp
  offset: 14       optional, added to the commit count
  other keys and comments are kept as they are

Before deploy:
  vstamp count --release && ./deploy
`

// errSilent exits 1 after the command already printed its result.
var errSilent = errors.New("")

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	var err error
	switch cmd {
	case "":
		err = cmdShow(stdout, stderr)
	case "tag":
		err = cmdTag(args[1:], stdout)
	case "count":
		err = cmdCount(args[1:], stdout, stderr)
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
	case "-v", "-version", "--version":
		fmt.Fprintln(stdout, toolVersion())
	default:
		err = fmt.Errorf("unknown command %q, see vstamp --help", cmd)
	}
	if err != nil {
		if err != errSilent {
			fmt.Fprintln(stderr, "vstamp:", err)
		}
		return 1
	}
	return 0
}

func toolVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}

func warnDrift(f *VersionFile, stderr io.Writer) {
	if f == nil {
		return
	}
	if tag, ok := lastTag(); ok && f.Version.Less(tag) {
		fmt.Fprintf(stderr, "warning: %s %s is behind tag v%s\n", versionFile, f.Version, tag)
	}
}

func cmdShow(stdout, stderr io.Writer) error {
	cur, err := readVersionFile()
	if err != nil {
		return err
	}
	if cur == nil {
		fmt.Fprintln(stdout, "dev")
		return nil
	}
	fmt.Fprintln(stdout, cur.Version)
	warnDrift(cur, stderr)
	return nil
}

func cmdTag(args []string, stdout io.Writer) error {
	if len(args) > 0 {
		return fmt.Errorf("unexpected argument %q", args[0])
	}
	tag, ok := lastTag()
	if !ok {
		fmt.Fprintln(stdout, "none")
		return errSilent
	}
	fmt.Fprintln(stdout, tag)
	return nil
}

func cmdCount(args []string, stdout, stderr io.Writer) error {
	fl := flag.NewFlagSet("count", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	branch := fl.String("branch", "", "")
	nextCommit := fl.Bool("next", false, "")
	commit := fl.Bool("commit", false, "")
	amend := fl.Bool("amend", false, "")
	tag := fl.Bool("tag", false, "")
	push := fl.Bool("push", false, "")
	release := fl.Bool("release", false, "")
	dryRun := fl.Bool("dry-run", false, "")
	if err := fl.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return nil
		}
		return err
	}
	if fl.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fl.Arg(0))
	}
	if *release {
		*amend, *tag, *push = true, true, true
	}
	switch {
	case *commit && *amend:
		return errors.New("--commit and --amend are exclusive")
	case *nextCommit && *amend:
		return errors.New("--next and --amend are exclusive, an amend adds no commit")
	case (*tag || *push) && !*commit && !*amend:
		return errors.New("--tag and --push need --commit or --amend")
	}

	cur, err := readVersionFile()
	if err != nil {
		return err
	}
	offset := 0
	if cur != nil {
		offset = cur.Offset
	}
	b, err := countBranch(*branch)
	if err != nil {
		return err
	}
	n, err := commitCount(b)
	if err != nil {
		return err
	}
	if *commit || *nextCommit {
		n++
	}
	next := FromCount(n + offset)
	if cur != nil && next.Less(cur.Version) {
		return fmt.Errorf(`count gives %s, below %s %s, set "offset: %d" in %s`, next, versionFile, cur.Version, MinCountAbove(cur.Version)-n, versionFile)
	}
	warnDrift(cur, stderr)

	tagName := "v" + next.String()
	if *commit || *amend {
		if head := currentBranch(); head != b {
			return fmt.Errorf("HEAD is on %q, but counting %q, switch branch or pass --branch", head, b)
		}
		staged, err := stagedFiles()
		if err != nil {
			return err
		}
		for _, f := range staged {
			if f != versionFile && f != manifestFile {
				return fmt.Errorf("%s is staged and would be committed, unstage it first", f)
			}
		}
	}
	if *tag && tagExists(tagName) {
		return fmt.Errorf("tag %s already exists", tagName)
	}
	var remote, merge string
	if *push {
		if remote, merge, err = upstream(b); err != nil {
			return err
		}
	}

	old := "none"
	if cur != nil {
		old = cur.Version.String()
	}
	manifestChanged, err := syncManifest(next, *dryRun)
	if err != nil {
		return err
	}
	if !*dryRun {
		if err := writeVersionFile(cur, next); err != nil {
			return err
		}
	}

	var steps [][]string
	if *commit || *amend {
		add := []string{"add", "--", versionFile}
		if manifestChanged {
			add = append(add, manifestFile)
		}
		steps = append(steps, add)
		if *commit {
			steps = append(steps, []string{"commit", "-m", "release " + next.String()})
		} else {
			steps = append(steps, []string{"commit", "--amend", "--no-edit"})
		}
	}
	if *tag {
		steps = append(steps, []string{"tag", tagName})
	}
	if *push {
		p := []string{"push", "--atomic", "--force-with-lease", "--force-if-includes", remote, "HEAD:" + merge}
		if *tag {
			p = append(p, "refs/tags/"+tagName)
		}
		steps = append(steps, p)
	}

	if *dryRun {
		fmt.Fprintf(stdout, "%s -> %s (dry run)\n", old, next)
	} else {
		fmt.Fprintf(stdout, "%s -> %s\n", old, next)
	}
	if manifestChanged {
		fmt.Fprintf(stdout, "  %s %s\n", manifestFile, next)
	}
	for _, s := range steps {
		fmt.Fprintln(stdout, "  "+shellLine(s))
		if *dryRun {
			continue
		}
		if _, err := git(s...); err != nil {
			if s[0] == "push" {
				return fmt.Errorf("%w\nlocal commit and tag are kept, retry with: %s", err, shellLine(s))
			}
			return err
		}
	}
	return nil
}

func shellLine(args []string) string {
	parts := []string{"git"}
	for _, a := range args {
		if strings.ContainsAny(a, " \"'") {
			a = strconv.Quote(a)
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}
