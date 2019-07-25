package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type arguments struct {
	verbose bool
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
		{"compile pattern", p.compilePattern},
		{"compile filters", p.compileFilters},
		{"execute pattern", p.executePattern},
		{"print results", p.printResults},
	}

	for _, step := range steps {
		if args.verbose {
			log.Printf("debug: starting %q step", step.name)
		}
		if err := step.fn(); err != nil {
			log.Printf("error: %s: %v", step.name, err)
			os.Exit(2)
		}
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
  phpgrep -v ~/code/php 'f(${"x:var"})' 'x=id|uid|gid'

Exit status:
  0 if something is matched
  1 if nothing is matched
  2 if error occured

For more info and examples visit https://github.com/quasilyte/phpgrep

Supported command-line flags:
`
		fmt.Fprintf(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}

	flag.BoolVar(&args.verbose, "v", false,
		`verbose mode that turns on additional debug logging`)

	flag.Parse()
}
