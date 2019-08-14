package main

import (
	"fmt"
	"io/ioutil"

	"github.com/quasilyte/phpgrep"
)

type worker struct {
	id      int
	m       *phpgrep.Matcher
	matches []match
}

func (w *worker) grepFile(filename string) ([]match, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read target: %v", err)
	}

	w.matches = w.matches[:0]
	w.m.Find(data, func(m *phpgrep.MatchData) bool {
		w.matches = append(w.matches, match{
			text:     string(data[m.PosFrom:m.PosTo]),
			filename: filename,
			line:     m.LineFrom,
		})
		return true
	})

	return w.matches, nil
}
