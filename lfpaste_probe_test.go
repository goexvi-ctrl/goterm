package goterm

import "testing"

// TestProbeLinefeedKeys compares govi against nvi for raw \n (linefeed, ^J)
// bytes -- what terminals that do not convert newlines to \r on paste (e.g.
// iPadSSH) send for every line break. nvi's key table (K_NL) treats \n as a
// line end in all text input; govi used to insert a literal ^J character.
func TestProbeLinefeedKeys(t *testing.T) {
	cases := []divCase{
		{"lf-paste-ins", []string{"seed line"},
			"OM1 Max =>\nM1 Max =>\nM1 Max =>\nM1 Max =>\x1b",
			"multi-line paste with \\n line endings in insert mode"},
		{"lf-colon", []string{"aaa", "bbb"},
			":s/aaa/xxx/\n",
			"colon command line executed by \\n instead of \\r"},
		{"lf-replace", []string{"abcd"},
			"llr\n",
			"r<^J> replaces the char with a line break"},
		{"lf-cmd-down", []string{"one", "two", "three"},
			"\nx",
			"^J stays the down motion in command mode"},
		{"lf-quoted", []string{"ab"},
			"i\x16\n\x1b",
			"a ^V-quoted ^J still ends the line (nvi Q_VTHIS skips K_NL)"},
	}
	for _, c := range cases {
		if runDivCase(t, 24, 80, c) {
			t.Errorf("[%s] diverged", c.name)
		}
	}
}
