package phpgrep

import (
	"bytes"
	"fmt"
	"io"
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
	"text/template"
	"time"

	"github.com/VKCOM/noverify/src/ir"
	"github.com/VKCOM/noverify/src/ir/irconv"
	"github.com/VKCOM/noverify/src/phpdoc"
	"github.com/VKCOM/noverify/src/phpgrep"
)

type match struct {
	text             string
	matchStartOffset int
	matchLength      int

	filename string
	line     int

	data phpgrep.MatchData
}

type program struct {
	args arguments

	workers []*worker

	filters        []phpgrepFilter
	exclude        *regexp.Regexp
	outputTemplate *template.Template
	matches        int64

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
	if p.args.targets == "" {
		return fmt.Errorf("target can't be empty")
	}
	if p.args.pattern == "" {
		return fmt.Errorf("pattern can't be empty")
	}
	if p.args.format == "" {
		return fmt.Errorf("format can't be empty")
	}
	if _, err := colorizeText("", p.args.filenameColor); err != nil {
		return fmt.Errorf("color-filename: %v", err)
	}
	if _, err := colorizeText("", p.args.lineColor); err != nil {
		return fmt.Errorf("color-line: %v", err)
	}
	if _, err := colorizeText("", p.args.matchColor); err != nil {
		return fmt.Errorf("color-match: %v", err)
	}
	// If there are more than 100k results, something is wrong.
	// Most likely, a user pattern is too generic and needs adjustment.
	const maxLimit = 100000
	if p.args.limit == 0 || p.args.limit > maxLimit {
		p.args.limit = maxLimit
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
	c.CaseSensitive = p.args.caseSensitive

	m, err := c.Compile([]byte(p.args.pattern))
	if err != nil {
		return err
	}

	var filters map[string][]filterFunc
	if len(p.filters) != 0 {
		filters = make(map[string][]filterFunc)
		for _, f := range p.filters {
			filters[f.name] = append(filters[f.name], f.fn)
		}
	}

	deps := inspectFormatDeps(p.args.format)
	needMatchData := deps.capture
	needMatchLine := deps.matchLine

	p.workers = make([]*worker, p.args.workers)
	for i := range p.workers {
		p.workers[i] = &worker{
			id:            i,
			m:             m.Clone(),
			filters:       filters,
			irconv:        irconv.NewConverter(phpdoc.NewTypeParser()),
			needMatchData: needMatchData,
			needMatchLine: needMatchLine,
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

func (p *program) compileOutputFormat() error {
	format := p.args.format
	var err error
	p.outputTemplate, err = template.New("output-format").Parse(format)
	if err != nil {
		return err
	}
	return nil
}

func (p *program) printMatches() error {
	printed := uint(0)
	for _, w := range p.workers {
		for _, m := range w.matches {
			if err := printMatch(p.outputTemplate, &p.args, m); err != nil {
				return err
			}
			printed++
			if printed >= p.args.limit {
				log.Printf("results limited to %d matches", p.args.limit)
				return nil
			}
		}
	}
	log.Printf("found %d matches", printed)
	return nil
}

func (p *program) executePattern() error {
	filenameQueue := make(chan string)
	ticker := time.NewTicker(time.Second)

	var wg sync.WaitGroup
	wg.Add(len(p.workers))
	defer func() {
		close(filenameQueue)
		ticker.Stop()
		wg.Wait()
	}()

	for _, w := range p.workers {
		go func(w *worker) {
			defer wg.Done()

			for filename := range filenameQueue {
				if p.args.verbose {
					log.Printf("debug: worker#%d greps %q file", w.id, filename)
				}

				numMatches, err := w.grepFile(filename)
				if err != nil {
					log.Printf("error: execute pattern: %s: %v", filename, err)
					continue
				}
				if numMatches == 0 {
					continue
				}

				atomic.AddInt64(&p.matches, int64(numMatches))
			}
		}(w)
	}

	for _, target := range strings.Split(p.args.targets, ",") {
		target = strings.TrimSpace(target)
		if err := p.walkTarget(target, filenameQueue, ticker); err != nil {
			return err
		}
	}

	return nil
}

func (p *program) walkTarget(target string, filenameQueue chan<- string, ticker *time.Ticker) error {
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

	// TODO: use NoVerify workspace walking code?
	filesProcessed := 0
	err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		numMatches := atomic.LoadInt64(&p.matches)
		if numMatches > int64(p.args.limit) {
			return io.EOF
		}

		if p.exclude != nil {
			fullName, err := filepath.Abs(path)
			if err != nil {
				log.Printf("error: abs(%s): %v", path, err)
			}
			skip := p.exclude.MatchString(fullName)
			if skip && info.IsDir() {
				return filepath.SkipDir
			}
			if skip {
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}
		if !isPHPFile(info.Name()) {
			return nil
		}

		for {
			select {
			case filenameQueue <- path:
				filesProcessed++
				return nil
			case <-ticker.C:
				log.Printf("%d matches so far, processed %d files", numMatches, filesProcessed)
			}
		}
	})
	if err == io.EOF {
		return nil
	}

	return err
}

func mustColorizeText(s, color string) string {
	result, err := colorizeText(s, color)
	if err != nil {
		panic(err)
	}
	return result
}

var ansiColorMap = map[string]string{
	"dark-red": "31m",
	"red":      "31;1m",

	"dark-green": "32m",
	"green":      "32;1m",

	"dark-blue": "34m",
	"blue":      "34;1m",

	"dark-magenta": "35m",
	"magenta":      "35;1m",
}

func colorizeText(s, color string) (string, error) {
	switch color {
	case "", "white":
		return s, nil
	default:
		escape, ok := ansiColorMap[color]
		if !ok {
			return "", fmt.Errorf("unsupported color: %s", color)
		}
		return "\033[" + escape + s + "\033[0m", nil
	}
}

func printMatch(tmpl *template.Template, args *arguments, m match) error {
	matchText := m.text[m.matchStartOffset : m.matchStartOffset+m.matchLength]
	filename := m.filename
	if args.abs {
		abs, err := filepath.Abs(filename)
		if err != nil {
			return fmt.Errorf("abs(%q): %v", m.filename, err)
		}
		filename = abs
	}

	data := make(map[string]interface{}, 3)
	// If we captured anything, add submatches as map elements.
	if len(m.data.Capture) != 0 {
		pos := ir.GetPosition(m.data.Node)
		for _, capture := range m.data.Capture {
			// Since we don't have file contents at this point, we can't
			// do a simple contents[StartPos:EndPos].
			// But we do know that all submatches located somewhere inside m.text.
			capturePos := ir.GetPosition(capture.Node)
			width := capturePos.EndPos - capturePos.StartPos
			begin := capturePos.StartPos - pos.StartPos
			end := begin + width
			data[capture.Name] = m.text[begin:end]
		}
	}

	// Assign these after the captures so they overwrite them in case of collisions.
	data["Filename"] = filename
	data["Line"] = m.line
	data["Match"] = matchText
	data["MatchLine"] = m.text

	if !args.noColor {
		data["Filename"] = mustColorizeText(filename, args.filenameColor)
		data["Line"] = mustColorizeText(fmt.Sprint(m.line), args.lineColor)
		data["Match"] = mustColorizeText(matchText, args.matchColor)
		data["MatchLine"] = m.text[:m.matchStartOffset] + mustColorizeText(matchText, args.matchColor) + m.text[m.matchStartOffset+m.matchLength:]
	}

	if !args.multiline {
		data["Match"] = strings.ReplaceAll(data["Match"].(string), "\n", `\n`)
		data["MatchLine"] = strings.ReplaceAll(data["MatchLine"].(string), "\n", `\n`)
	}

	var buf strings.Builder
	buf.Grow(len(data["MatchLine"].(string)) * 2) // Approx
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	fmt.Println(buf.String())
	return nil
}
