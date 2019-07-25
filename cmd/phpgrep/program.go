package main

import (
	"fmt"
	"io/ioutil"
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

func (p *program) executePattern() error {
	// TODO(quasilyte): handle directory targets.
	data, err := ioutil.ReadFile(p.args.target)
	if err != nil {
		return fmt.Errorf("target not found: %v", err)
	}
	p.m.Find(data, func(m *phpgrep.MatchData) bool {
		p.matches = append(p.matches, match{
			text:     string(data[m.PosFrom:m.PosTo]),
			filename: p.args.target,
			line:     m.LineFrom,
		})
		return true
	})

	return nil
}

func (p *program) printResults() error {
	// TODO(quasilyte): add JSON output format?
	for _, m := range p.matches {
		text := m.text
		if !p.args.multiline {
			text = strings.Replace(text, "\n", `\n`, -1)
		}
		fmt.Printf("%s:%d: %s\n", m.filename, m.line, text)
	}

	return nil
}
