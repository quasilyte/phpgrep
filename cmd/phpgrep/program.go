package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	matches int64
}

func (p *program) validateFlags() error {
	if p.args.workers < 1 {
		return fmt.Errorf("workers value can't be less than 1")
	}
	if p.args.workers > 512 {
		// Users won't notice.
		p.args.workers = 512
	}
	if p.args.target == "" {
		return fmt.Errorf("target can't be empty")
	}
	if p.args.pattern == "" {
		return fmt.Errorf("pattern can't be empty")
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
