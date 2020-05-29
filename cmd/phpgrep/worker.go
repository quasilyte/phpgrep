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

func (w *worker) grepFile(filename string) (int, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, fmt.Errorf("read file: %v", err)
	}

	n := 0
	w.m.Find(data, func(m *phpgrep.MatchData) bool {
		n++
		w.matches = append(w.matches, match{
			text:     string(data[m.PosFrom:m.PosTo]),
			filename: filename,
			line:     m.LineFrom,
		})
		return true
	})

	return n, nil
}
