package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/quasilyte/phpgrep"
)

type match struct {
	text     string
	filename string
	line     int
}

type program struct {
	args arguments

	workers []*worker
	filters []phpgrep.Filter
	exclude *regexp.Regexp
	matches int64

	cpuProfile bytes.Buffer
}

func (p *program) validateFlags() error {
	if p.args.workers < 1 {
		return fmt.Errorf("workers value can't be less than 1")
	}
	if p.args.workers > 128 {
		// Users won't notice.
		p.args.workers = 128
	}
	if p.args.target == "" {
		return fmt.Errorf("target can't be empty")
	}
	if p.args.pattern == "" {
		return fmt.Errorf("pattern can't be empty")
	}
	return nil
}

func (p *program) startProfiling() error {
	if p.args.cpuProfile == "" {
		return nil
	}

	if err := pprof.StartCPUProfile(&p.cpuProfile); err != nil {
		return fmt.Errorf("could not start CPU profile: %v", err)
	}

	return nil
}

func (p *program) finishProfiling() error {
	if p.args.cpuProfile != "" {
		pprof.StopCPUProfile()
		err := ioutil.WriteFile(p.args.cpuProfile, p.cpuProfile.Bytes(), 0666)
		if err != nil {
			return fmt.Errorf("write CPU profile: %v", err)
		}
	}

	if p.args.memProfile != "" {
		f, err := os.Create(p.args.memProfile)
		if err != nil {
			return fmt.Errorf("create mem profile: %v", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return fmt.Errorf("write mem profile: %v", err)
		}
	}

	return nil
}

func (p *program) compileFilters() error {
	for _, s := range p.args.filters {
		f, err := compileFilter(s)
		if err != nil {
			return fmt.Errorf("compile %q filter: %v", s, err)
		}
		p.filters = append(p.filters, f)
	}

	return nil
}

func (p *program) compilePattern() error {
	var c phpgrep.Compiler
	m, err := c.Compile([]byte(p.args.pattern), p.filters...)
	if err != nil {
		return err
	}

	p.workers = make([]*worker, p.args.workers)
	for i := range p.workers {
		p.workers[i] = &worker{
			id: i,
			m:  m.Clone(),
		}
	}

	return nil
}

func (p *program) compileExcludePattern() error {
	if p.args.exclude == "" {
		return nil
	}
	var err error
	p.exclude, err = regexp.Compile(p.args.exclude)
	if err != nil {
		return fmt.Errorf("invalid exclude regexp: %v", err)
	}
	return nil
}

func (p *program) executePattern() error {
	phpExtensions := []string{
		".php",
		".php5",
		".inc",
		".phtml",
	}
	isPHPFile := func(name string) bool {
		for _, ext := range phpExtensions {
			if strings.HasSuffix(name, ext) {
				return true
			}
		}
		return false
	}

	filenameQueue := make(chan string)

	var wg sync.WaitGroup
	wg.Add(len(p.workers))
	defer func() {
		close(filenameQueue)
		wg.Wait()
	}()

	for _, w := range p.workers {
		go func(w *worker) {
			defer wg.Done()

			for filename := range filenameQueue {
				if p.args.verbose {
					log.Printf("debug: worker#%d greps %q file", w.id, filename)
				}

				matches, err := w.grepFile(filename)
				if err != nil {
					log.Printf("error: execute pattern: %s: %v", filename, err)
					continue
				}
				if len(matches) == 0 {
					continue
				}

				atomic.AddInt64(&p.matches, int64(len(matches)))
				printMatches(&p.args, matches)
			}
		}(w)
	}

	err := filepath.Walk(p.args.target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if p.exclude != nil && p.exclude.MatchString(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}
		if !isPHPFile(info.Name()) {
			return nil
		}

		filenameQueue <- path
		return nil
	})

	return err
}

func printMatches(args *arguments, matches []match) error {
	for _, m := range matches {
		text := m.text
		if !args.multiline {
			text = strings.Replace(text, "\n", `\n`, -1)
		}
		filename := m.filename
		if args.abs {
			abs, err := filepath.Abs(filename)
			if err != nil {
				return fmt.Errorf("abs(%q): %v", m.filename, err)
			}
			filename = abs
		}
		fmt.Printf("%s:%d: %s\n", filename, m.line, text)
	}

	return nil
}
