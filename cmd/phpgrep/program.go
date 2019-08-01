package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/quasilyte/phpgrep"
)

type match struct {
	text     string
	filename string
	line     int
}

type program struct {
	args arguments

	m       *phpgrep.Matcher
	filters []phpgrep.Filter

	matches []match
}

func (p *program) validateFlags() error {
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
	p.m = m
	return nil
}

func (p *program) grepFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read target: %v", err)
	}

	p.m.Find(data, func(m *phpgrep.MatchData) bool {
		p.matches = append(p.matches, match{
			text:     string(data[m.PosFrom:m.PosTo]),
			filename: filename,
			line:     m.LineFrom,
		})
		return true
	})

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

	return filepath.Walk(p.args.target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		if !isPHPFile(info.Name()) {
			return nil
		}

		if p.args.verbose {
			log.Printf("debug: grep %q file", path)
		}
		return p.grepFile(path)
	})
}

func (p *program) printResults() error {
	// TODO(quasilyte): add JSON output format?
	for _, m := range p.matches {
		text := m.text
		if !p.args.multiline {
			text = strings.Replace(text, "\n", `\n`, -1)
		}
		filename := m.filename
		if p.args.abs {
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
