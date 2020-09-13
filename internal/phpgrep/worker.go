package phpgrep

import (
	"fmt"
	"io/ioutil"

	"github.com/VKCOM/noverify/src/ir"
	"github.com/VKCOM/noverify/src/ir/irconv"
	"github.com/VKCOM/noverify/src/php/parseutil"
	"github.com/VKCOM/noverify/src/phpgrep"
)

type worker struct {
	id      int
	m       *phpgrep.Matcher
	filters map[string][]filterFunc

	needMatchData bool

	irconv  *irconv.Converter
	matches []match

	data     []byte
	filename string
	n        int
}

func (w *worker) grepFile(filename string) (int, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, fmt.Errorf("read file: %v", err)
	}

	root, err := w.parseFile(data)
	if err != nil {
		return 0, err
	}

	w.data = data
	w.filename = filename
	w.n = 0
	root.Walk(w)
	return w.n, nil
}

func (w *worker) parseFile(data []byte) (*ir.Root, error) {
	root, err := parseutil.ParseFile(data)
	if err != nil {
		return nil, err
	}
	return w.irconv.ConvertRoot(root), nil
}

func (w *worker) LeaveNode(ir.Node) {}

func (w *worker) EnterNode(n ir.Node) bool {
	data, ok := w.m.Match(n)
	if ok && w.acceptMatch(data) {
		w.n++
		pos := ir.GetPosition(data.Node)
		w.matches = append(w.matches, match{
			text:     string(w.data[pos.StartPos:pos.EndPos]),
			filename: w.filename,
			line:     pos.StartLine,
			data:     w.maybeCloneData(data),
		})
	}

	return true
}

func (w *worker) acceptMatch(m phpgrep.MatchData) bool {
	if len(w.filters) == 0 {
		return true
	}

	for _, capture := range m.Capture {
		filterList, ok := w.filters[capture.Name]
		if !ok {
			continue
		}
		pos := ir.GetPosition(capture.Node)
		buf := w.data[pos.StartPos:pos.EndPos]
		for _, filter := range filterList {
			if !filter(buf) {
				return false
			}
		}
	}

	return true
}

func (w *worker) maybeCloneData(data phpgrep.MatchData) phpgrep.MatchData {
	if !w.needMatchData {
		return phpgrep.MatchData{}
	}

	capture := make([]phpgrep.CapturedNode, len(data.Capture))
	copy(capture, data.Capture)
	return phpgrep.MatchData{
		Node:    data.Node,
		Capture: capture,
	}
}
