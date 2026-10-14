package main

import (
	"fmt"
	"regexp"
	"strconv"
)

type Version struct {
	Major, Minor, Patch int
}

var versionRe = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

func ParseVersion(s string) (Version, error) {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("invalid version %q, expected X.Y.Z", s)
	}
	n := make([]int, 3)
	for i := range n {
		v, err := strconv.Atoi(m[i+1])
		if err != nil {
			return Version{}, fmt.Errorf("invalid version %q: %w", s, err)
		}
		n[i] = v
	}
	return Version{n[0], n[1], n[2]}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Less(o Version) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	return v.Patch < o.Patch
}

// FromCount maps a commit count to a version: the last two digits are minor
// and patch, the rest is major (1111 -> 11.1.1, 380 -> 3.8.0, 5 -> 0.0.5).
func FromCount(n int) Version {
	return Version{n / 100, n / 10 % 10, n % 10}
}

// MinCountAbove is the smallest count whose FromCount is greater than v.
func MinCountAbove(v Version) int {
	switch {
	case v.Minor > 9:
		return (v.Major + 1) * 100
	case v.Patch > 9:
		return v.Major*100 + (v.Minor+1)*10
	default:
		return v.Major*100 + v.Minor*10 + v.Patch + 1
	}
}
