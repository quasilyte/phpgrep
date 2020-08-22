package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
)

const (
	exitMatched    = 0
	exitNotMatched = 1
	exitError      = 2
)

type arguments struct {
	verbose   bool
	multiline bool
	abs       bool

	limit uint

	cpuProfile string
	memProfile string

	target  string
	pattern string
	filters []string
	exclude string
	format  string

	workers int
}

func main() {
	log.SetFlags(0)

	var args arguments
	parseFlags(&args)

	p := &program{
		args: args,
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{"validate flags", p.validateFlags},
		{"start profiling", p.startProfiling},
		{"compile filters", p.compileFilters},
		{"compile pattern", p.compilePattern},
		{"compile exclude pattern", p.compileExcludePattern},
		{"compile output format", p.compileOutputFormat},
		{"execute pattern", p.executePattern},
		{"print matches", p.printMatches},
		{"finish profiling", p.finishProfiling},
	}

	for _, step := range steps {
		if args.verbose {
			log.Printf("debug: starting %q step", step.name)
		}
		if err := step.fn(); err != nil {
			log.Printf("error: %s: %v", step.name, err)
			os.Exit(exitError)
		}
	}

	if p.matches == 0 {
		os.Exit(exitNotMatched)
	} else {
		os.Exit(exitMatched)
	}
}

func parseFlags(args *arguments) {
	flag.Usage = func() {
		const usage = `Usage: phpgrep [flags...] target pattern [filters...]
Where:
  flags are command-line flags that are listed in -help (see below)
  target is a file or directory name where search is performed
  pattern is a string that describes what is being matched
  filters are optional arguments bound to the pattern

Examples:
  # Find f calls with a single varible argument.
  phpgrep file.php 'f(${"var"})'
  # Like previous example, but searches inside entire
  # directory recursively and variable names are restricted
  # to $id, $uid and $gid.
  # Also uses -v flag that makes phpgrep output more info.
  phpgrep -v ~/code/php 'f(${"x:var"})' 'x=id,uid,gid'
  # Print only matches, without locations.
  phpgrep -format '{{.Match}}' file.php 'pattern'

Exit status:
  0 if something is matched
  1 if nothing is matched
  2 if error occurred

Custom output formatting is possible via the -format flag template.
Refer to the package documentation to see all available options.

For more info and examples visit https://github.com/quasilyte/phpgrep

Supported command-line flags:
`
		fmt.Fprint(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}

	flag.BoolVar(&args.verbose, "v", false,
		`verbose mode: turn on additional debug logging`)
	flag.BoolVar(&args.multiline, "m", false,
		`multiline mode: print matches without escaping newlines to \n`)
	flag.BoolVar(&args.abs, "abs", false,
		`print absolute filenames in the output`)
	flag.UintVar(&args.limit, "limit", 1000,
		`stop after this many match results, 0 for unlimited`)
	flag.IntVar(&args.workers, "workers", runtime.NumCPU(),
		`set the number of concurrent workers`)
	flag.StringVar(&args.memProfile, "memprofile", "",
		`write memory profile to the specified file`)
	flag.StringVar(&args.cpuProfile, "cpuprofile", "",
		`write CPU profile to the specified file`)
	flag.StringVar(&args.exclude, "exclude", "",
		`exclude files or directories by regexp pattern`)
	flag.StringVar(&args.format, "format", `{{.Filename}}:{{.Line}}: {{.Match}}`,
		`specify an alternate format for the output, using the syntax Go templates`)

	flag.Parse()

	argv := flag.Args()
	if len(argv) >= 1 {
		args.target = argv[0]
	}
	if len(argv) >= 2 {
		args.pattern = argv[1]
	}
	if len(argv) > 2 {
		args.filters = argv[2:]
	}
}
