package phpgrep

import (
	"testing"
)

func TestInspectFormatDeps(t *testing.T) {
	tests := []struct {
		format string
		deps   formatDeps
	}{
		{
			format: ``,
			deps:   formatDeps{},
		},

		{
			format: defaultFormat,
			deps:   formatDeps{matchLine: true},
		},
		{
			format: `{{.Filename}}: blah`,
			deps:   formatDeps{},
		},

		{
			format: `{{.x}}`,
			deps:   formatDeps{capture: true},
		},
		{
			format: `{{.Filename}}:{{.Line}}: {{.foo}}`,
			deps:   formatDeps{capture: true},
		},
		{
			format: `{{if .x}}hit{{else}}miss{{end}}`,
			deps:   formatDeps{capture: true},
		},
	}

	for _, test := range tests {
		have := inspectFormatDeps(test.format)
		want := test.deps
		if have.capture != want.capture {
			t.Errorf("inspect `%s`: capture=%v (want %v)",
				test.format, have.capture, want.capture)
			continue
		}
		if have.matchLine != want.matchLine {
			t.Errorf("inspect `%s`: matchLine=%v (want %v)",
				test.format, have.matchLine, want.matchLine)
			continue
		}
	}
}
