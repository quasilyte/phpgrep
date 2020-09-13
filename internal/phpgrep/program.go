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
	text     string
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
	if p.args.target == "" {
		return fmt.Errorf("target can't be empty")
	}
	if p.args.pattern == "" {
		return fmt.Errorf("pattern can't be empty")
	}
	if p.args.format == "" {
		return fmt.Errorf("format can't be empty")
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

	// TODO: this is very a pessimistic way of checking.
	needMatchData := p.args.format != defaultFormat

	p.workers = make([]*worker, p.args.workers)
	for i := range p.workers {
		p.workers[i] = &worker{
			id:            i,
			m:             m.Clone(),
			filters:       filters,
			irconv:        irconv.NewConverter(phpdoc.NewTypeParser()),
			needMatchData: needMatchData,
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

	// TODO: use NoVerify workspace walking code?
	filesProcessed := 0
	err := filepath.Walk(p.args.target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		numMatches := atomic.LoadInt64(&p.matches)
		if numMatches > int64(p.args.limit) {
			return io.EOF
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

func printMatch(tmpl *template.Template, args *arguments, m match) error {
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

	data := map[string]interface{}{
		"Filename": filename,
		"Line":     m.line,
		"Match":    text,
	}

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

	var buf strings.Builder
	buf.Grow(len(text) * 2) // Approx
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	fmt.Println(buf.String())
	return nil
}
