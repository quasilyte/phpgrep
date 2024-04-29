package phpgrep

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

const defaultFormat = `{{.Filename}}:{{.Line}}: {{.MatchLine}}`

type arguments struct {
	replace       bool
	verbose       bool
	multiline     bool
	abs           bool
	caseSensitive bool
	noColor       bool
	strictSyntax  bool

	limit uint

	cpuProfile string
	memProfile string

	phpFileExt     string
	phpFileExtList []string

	targets        string
	pattern        string
	filters        []string
	exclude        string
	format         string
	excludeResults string

	progressMode string

	filenameColor string
	lineColor     string
	matchColor    string

	workers int // TODO: make a uint flag and don't check for <0?
}

func Main() (int, error) {
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
		{"compile exclude results", p.compileExcludeResults},
		{"compile exclude pattern", p.compileExcludePattern},
		{"compile pattern", p.compilePattern},
		{"compile output format", p.compileOutputFormat},
		{"execute pattern", p.executePattern},
		{"print matches", p.printMatches},
		{"replace matches", p.replaceMatches},
		{"finish profiling", p.finishProfiling},
	}

	for _, step := range steps {
		if args.verbose {
			log.Printf("debug: starting %q step", step.name)
		}
		if err := step.fn(); err != nil {
			return exitError, fmt.Errorf("%s: %v", step.name, err)
		}
	}

	if p.matches == 0 {
		return exitNotMatched, nil
	}
	return exitMatched, nil
}

func parseFlags(args *arguments) {
	flag.Usage = func() {
		const usage = `Usage: phpgrep [flags...] targets pattern [filters...]
Where:
  flags are command-line arguments that are listed in -help (see below)
  targets is a comma-separated list of file or directory names to search in
  pattern is a string that describes what is being matched
  filters are optional arguments bound to the pattern

Examples:
  # Find f calls with a single varible argument.
  phpgrep file.php 'f(${"var"})'

  # Like the previous example, but searches inside entire
  # directory recursively and variable names are restricted
  # to $id, $uid and $gid.
  # Also uses -v flag that makes phpgrep output more info.
  phpgrep -v ~/code/php 'f(${"x:var"})' 'x=id,uid,gid'

  # Run phpgrep on 2 folders (recursively).
  phpgrep dir1,dir2 '"some string"'

  # Print only matches, without locations.
  phpgrep -format '{{.Match}}' file.php 'pattern'

  # Print only assignments right-hand side.
  phpgrep -format '{{.rhs}}' file.php '$_ = $rhs'

  # Ignore vendored source code inside project.
  phpgrep --exclude '/vendor/' project/ 'pattern'

Custom output formatting is possible via the -format flag template.
  {{.Filename}}  match containing file name
  {{.Line}}      line number where the match started
  {{.MatchLine}} a source code line that contains the match
  {{.Match}}     an entire match string
  {{.x}}         $x submatch string (can be any submatch name)

The output colors can be configured with "--color-<name>" flags.
Use --no-color to disable the output coloring.

Exit status:
  0 if something is matched
  1 if nothing is matched
  2 if error occurred

For more info and examples visit https://github.com/quasilyte/phpgrep

Supported command-line flags:
`
		fmt.Fprint(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}

	flag.BoolVar(&args.replace, "i", false,
		`replace matches with --format result in-place`)
	flag.BoolVar(&args.verbose, "v", false,
		`verbose mode: turn on additional debug logging`)
	flag.BoolVar(&args.multiline, "m", false,
		`multiline mode: print matches without escaping newlines to \n`)
	flag.BoolVar(&args.caseSensitive, "case-sensitive", false,
		`do a strict case matching, so F() and f() are considered to be distinct`)
	flag.BoolVar(&args.strictSyntax, "strict-syntax", false,
		`disable syntax normalizations, so 'array()' and '[]' are not considered to be identical, and so on`)
	flag.BoolVar(&args.noColor, "no-color", false,
		`disable the colored output`)
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
	flag.StringVar(&args.excludeResults, "exclude-results", "",
		`exclude the results listed in the file`)
	flag.StringVar(&args.phpFileExt, "php-ext", "php,php5,inc,phtml",
		`a comma-separated list of extensions to scan`)

	flag.StringVar(&args.progressMode, "progress", "update",
		`progress printing mode: "update", "append" or "none"`)

	flag.StringVar(&args.filenameColor, "color-filename", envVarOrDefault("PHPGREP_COLOR_FILENAME", "dark-magenta"),
		`{{.Filename}} text color`)
	flag.StringVar(&args.lineColor, "color-line", envVarOrDefault("PHPGREP_COLOR_LINE", "dark-green"),
		`{{.Line}} text color`)
	flag.StringVar(&args.matchColor, "color-match", envVarOrDefault("PHPGREP_COLOR_MATCH", "dark-red"),
		`{{.Match}} text color`)

	flag.StringVar(&args.format, "format", defaultFormat,
		`specify an alternate format for the output, using the syntax Go templates`)

	flag.Parse()

	argv := flag.Args()
	if len(argv) != 0 {
		args.targets = argv[0]
	}
	if len(argv) >= 2 {
		args.pattern = argv[1]
	}
	if len(argv) > 2 {
		args.filters = argv[2:]
	}
	if args.verbose {
		args.progressMode = "append"
	}

	if args.verbose {
		log.Printf("debug: targets: %s", args.targets)
		log.Printf("debug: pattern: %s", args.pattern)
		log.Printf("debug: filters: %#v", args.filters)
	}
}

func envVarOrDefault(envKey, defaultValue string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultValue
}
