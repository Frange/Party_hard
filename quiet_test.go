package main

import (
	"flag"
	"testing"
)

func TestQuietMode(t *testing.T) {
	cases := []struct {
		args []string
		env  string
		want bool
	}{
		{nil, "", false},
		{[]string{"-mute", "-lang", "es", "-title"}, "", false},
		{[]string{"-editor"}, "", false},
		{[]string{"-level", "1", "-shot", "a.png"}, "", true},
		{[]string{"-gamebench", "100"}, "", true},
		{[]string{"-twop"}, "", true},
		{[]string{"-level", "1", "-shot", "a.png", "-visible"}, "", false},
		{nil, "1", true},
		{[]string{"-visible"}, "1", false},
	}
	for _, c := range cases {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		for _, n := range []string{"mute", "title", "editor", "twop", "visible"} {
			fs.Bool(n, false, "")
		}
		for _, n := range []string{"lang", "level", "shot", "gamebench"} {
			fs.String(n, "", "")
		}
		if err := fs.Parse(c.args); err != nil {
			t.Fatal(err)
		}
		if got := quietMode(fs, c.env); got != c.want {
			t.Errorf("%v env=%q: got %v want %v", c.args, c.env, got, c.want)
		}
	}
}
