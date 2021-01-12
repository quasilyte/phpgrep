package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEnd2End(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	phpgrepBin := filepath.Join(wd, "phpgrep.exe")

	out, err := exec.Command("go", "build", "-race", "-o", phpgrepBin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build phpgrep: %v: %s", err, out)
	}

	type patternTest struct {
		pattern string
		filters []string
		matches []string
		exclude string
		targets string
	}
	tests := []struct {
		name  string
		tests []patternTest
	}{
		{
			name: "multi-target",
			tests: []patternTest{
				{
					targets: `f1.php,f2.php`,
					pattern: `var_dump(${"*"})`,
					matches: []string{
						"f1.php:3: var_dump('1')",
						"f2.php:3: var_dump('2')",
					},
				},
			},
		},

		{
			name: "exclude",
			tests: []patternTest{
				{
					pattern: `"str"`,
					matches: []string{
						`file.php:3: "str"`,
						"file.php:4: 'str'",
						"file.php:7: 'str'",
						`file.php:8: 'str'`,
						"file.php:12: 'str'",
						`file.php:13: "str"`,
					},
					exclude: `/vendor`,
				},
				{
					pattern: `'str'`,
					matches: []string{
						`file.php:3: "str"`,
						"file.php:4: 'str'",
						"file.php:7: 'str'",
						`file.php:8: 'str'`,
						"file.php:12: 'str'",
						`file.php:13: "str"`,
					},
					exclude: `.*vendor.*`,
				},
			},
		},

		{
			name: "filter",
			tests: []patternTest{
				// Test '=' for consts.
				{
					pattern: `$_(${"x:const"})`,
					filters: []string{`x=FOO`},
					matches: []string{"file.php:16: var_dump(FOO)"},
				},
				{
					pattern: `$_(${"x:const"})`,
					filters: []string{`x=BAR`},
					matches: []string{"file.php:17: var_dump(BAR)"},
				},
				{
					pattern: `$_(${"x:const"})`,
					filters: []string{`x=FOO,C::BAZ`},
					matches: []string{
						"file.php:16: var_dump(FOO)",
						"file.php:18: var_dump(C::BAZ)",
					},
				},
				{
					pattern: `$_(${"const"})`,
					matches: []string{
						"file.php:16: var_dump(FOO)",
						"file.php:17: var_dump(BAR)",
						"file.php:18: var_dump(C::BAZ)",
					},
				},

				// Test '=' for vars.
				{
					pattern: `$_(${"x:var"})`,
					filters: []string{`x=$uid`},
					matches: []string{"file.php:9: var_dump($uid)"},
				},
				{
					pattern: `$_(${"x:var"})`,
					filters: []string{`x=$pid`},
					matches: []string{"file.php:10: var_dump($pid)"},
				},
				{
					pattern: `$_(${"x:var"})`,
					filters: []string{`x=$pid,$uid`},
					matches: []string{
						"file.php:9: var_dump($uid)",
						"file.php:10: var_dump($pid)",
					},
				},
				{
					pattern: `$_(${"var"})`,
					matches: []string{
						"file.php:9: var_dump($uid)",
						"file.php:10: var_dump($pid)",
					},
				},

				// Test '=' for strings.
				{
					pattern: `define($name, $_)`,
					filters: []string{`name="FOO"`},
					matches: []string{`file.php:3: define("FOO", 1)`},
				},
				{
					pattern: `define($name, $_)`,
					filters: []string{`name='FOO'`},
					matches: []string{`file.php:4: define('FOO', 2)`},
				},
				{
					pattern: `define($name, $_)`,
					filters: []string{`name='FOO','BAR'`},
					matches: []string{
						`file.php:4: define('FOO', 2)`,
						`file.php:5: define('BAR', 3)`,
					},
				},

				// Test `~` for strings.
				{
					pattern: `define($name, $_)`,
					filters: []string{`name~"FOO"`},
					matches: []string{`file.php:3: define("FOO", 1)`},
				},
				{
					pattern: `define($name, $_)`,
					filters: []string{`name~^"FOO"$`},
					matches: []string{`file.php:3: define("FOO", 1)`},
				},
				{
					pattern: `define($name, $_)`,
					filters: []string{`name~^."FOO"$`},
				},
				{
					pattern: `define($name, $_)`,
					filters: []string{`name~^.."FOO".$`},
				},

				// Test '=' for ints.
				{
					pattern: `define($_, $v)`,
					filters: []string{`v=2`},
					matches: []string{
						`file.php:4: define('FOO', 2)`,
					},
				},
				{
					pattern: `define($_, $v)`,
					filters: []string{`v=1,2`},
					matches: []string{
						`file.php:3: define("FOO", 1)`,
						`file.php:4: define('FOO', 2)`,
					},
				},
				{
					pattern: `define($_, ${"v:int"})`,
					filters: []string{`v=2`},
					matches: []string{
						`file.php:4: define('FOO', 2)`,
					},
				},
				{
					pattern: `define($_, ${"v:int"})`,
					filters: []string{`v=1,2`},
					matches: []string{
						`file.php:3: define("FOO", 1)`,
						`file.php:4: define('FOO', 2)`,
					},
				},

				// Test '!=' for ints.
				{
					pattern: `define($_, $v)`,
					filters: []string{`v!=2`},
					matches: []string{
						`file.php:3: define("FOO", 1)`,
						`file.php:5: define('BAR', 3)`,
					},
				},
				{
					pattern: `define($_, $v)`,
					filters: []string{`v!=3,2`},
					matches: []string{
						`file.php:3: define("FOO", 1)`,
					},
				},
				{
					pattern: `define($_, ${"v:int"})`,
					filters: []string{`v!=2`},
					matches: []string{
						`file.php:3: define("FOO", 1)`,
						`file.php:5: define('BAR', 3)`,
					},
				},
				{
					pattern: `define($_, ${"v:int"})`,
					filters: []string{`v!=3,2`},
					matches: []string{
						`file.php:3: define("FOO", 1)`,
					},
				},
			},
		},
	}

	for _, test := range tests {
		testName := test.name
		patternTests := test.tests
		t.Run(test.name, func(t *testing.T) {
			target := filepath.Join("testdata", testName)
			if err := os.Chdir(target); err != nil {
				t.Fatalf("chdir to test: %v", err)
			}

			for _, test := range patternTests {
				var phpgrepArgs []string
				if test.exclude != "" {
					phpgrepArgs = append(phpgrepArgs, "--exclude", test.exclude)
				}
				targets := "."
				if test.targets != "" {
					targets = test.targets
				}
				phpgrepArgs = append(phpgrepArgs, targets, test.pattern)
				phpgrepArgs = append(phpgrepArgs, test.filters...)
				out, err := exec.Command(phpgrepBin, phpgrepArgs...).CombinedOutput()
				if err != nil {
					if getExitCode(err) == 1 && len(test.matches) == 0 {
						// OK: exit code 1 means "no matches".
						return
					}
					t.Fatalf("run phpgrep: %v: %s", err, out)
				}
				have := strings.Split(strings.TrimSpace(string(out)), "\n")
				want := test.matches
				want = append(want, fmt.Sprintf("found %d matches", len(test.matches)))
				if diff := cmp.Diff(want, have); diff != "" {
					t.Errorf("output mismatch (+have -want):\n%s", diff)
				}
			}

			if err := os.Chdir(wd); err != nil {
				t.Fatalf("chdir back: %v", err)
			}
		})
	}
}

func getExitCode(err error) int {
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return -1
}
