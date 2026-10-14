package main

import "testing"

func TestFromCount(t *testing.T) {
	cases := map[int]string{0: "0.0.0", 5: "0.0.5", 42: "0.4.2", 380: "3.8.0", 1111: "11.1.1", 12345: "123.4.5"}
	for n, want := range cases {
		if got := FromCount(n).String(); got != want {
			t.Errorf("FromCount(%d) = %s, want %s", n, got, want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	for _, s := range []string{"0.0.0", "1.2.3", "11.1.1", "1.10.0"} {
		v, err := ParseVersion(s)
		if err != nil || v.String() != s {
			t.Errorf("ParseVersion(%q) = %v, %v", s, v, err)
		}
	}
	for _, s := range []string{"", "v1.2.3", "v380", "1.2", "1.2.3.4", "01.2.3", "1.2.3-rc1", " 1.2.3", "dev"} {
		if _, err := ParseVersion(s); err == nil {
			t.Errorf("ParseVersion(%q) accepted", s)
		}
	}
}

func TestLess(t *testing.T) {
	ordered := []string{"0.0.1", "0.1.0", "0.9.9", "1.0.0", "1.2.3", "1.10.0", "2.0.0"}
	for i := 0; i < len(ordered)-1; i++ {
		a, _ := ParseVersion(ordered[i])
		b, _ := ParseVersion(ordered[i+1])
		if !a.Less(b) || b.Less(a) || a.Less(a) {
			t.Errorf("order broken between %s and %s", a, b)
		}
	}
}

func TestMinCountAbove(t *testing.T) {
	for _, s := range []string{"0.0.0", "0.2.3", "3.9.8", "3.9.9", "1.10.0", "1.2.15", "1.9.12", "0.0.9"} {
		v, _ := ParseVersion(s)
		want := 0
		for !v.Less(FromCount(want)) {
			want++
		}
		if got := MinCountAbove(v); got != want {
			t.Errorf("MinCountAbove(%s) = %d, want %d", s, got, want)
		}
	}
}
