package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/pprof/profile"
)

func main() {
	if err := mainNoExit(); err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}

func isRuntimeFunc(symbolName string) bool {
	if strings.HasPrefix(symbolName, "runtime.") {
		return true
	}
	if !strings.Contains(symbolName, ".") {
		return true
	}
	return false
}

func findNode(s *profile.Sample, fn func(l *profile.Line) bool) *profile.Line {
	for _, loc := range s.Location {
		for i := range loc.Line {
			if fn(&loc.Line[i]) {
				return &loc.Line[i]
			}
		}
	}
	return nil
}

func mainNoExit() error {
	flagFilter := flag.String("filter", ".*", "regexp filtering function names")
	flagTopN := flag.Int("n", 100, "keep top n nodes")
	flag.Parse()

	argv := flag.Args()
	if len(argv) != 1 {
		return errors.New("expected exactly 1 positional arg: cpu profile filename")
	}
	profileFilename := flag.Args()[0]

	data, err := os.ReadFile(profileFilename)
	if err != nil {
		return err
	}
	p, err := profile.Parse(bytes.NewReader(data))
	if err != nil {
		return err
	}

	perFunc := make(map[string]int64)
	for _, s := range p.Sample {
		sampleValue := s.Value[1]
		if len(s.Location) == 0 || len(s.Location[0].Line) == 0 {
			continue
		}
		l := s.Location[0].Line[0]
		perFunc[l.Function.Name] += sampleValue
	}
	for _, s := range p.Sample {
		sampleValue := s.Value[1]
		if len(s.Location) == 0 || len(s.Location[0].Line) == 0 {
			continue
		}
		l := s.Location[0].Line[0]
		if !isRuntimeFunc(l.Function.Name) {
			continue
		}
		caller := findNode(s, func(l *profile.Line) bool {
			return !isRuntimeFunc(l.Function.Name)
		})
		if caller != nil {
			perFunc[caller.Function.Name] += sampleValue
		}
	}

	if *flagFilter != ".*" {
		filterRe, err := regexp.Compile(*flagFilter)
		if err != nil {
			return err
		}
		for k := range perFunc {
			if !filterRe.MatchString(k) {
				delete(perFunc, k)
			}
		}
	}

	type keyvalue struct {
		key string
		val int64
	}

	{
		var sorted []keyvalue
		for k, v := range perFunc {
			if isRuntimeFunc(k) {
				continue
			}
			sorted = append(sorted, keyvalue{key: k, val: v})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].val > sorted[j].val
		})
		n := *flagTopN
		if n >= len(sorted) {
			n = len(sorted)
		}
		for _, kv := range sorted[:n] {
			fmt.Printf("%12s %s\n", time.Duration(kv.val), kv.key)
		}
	}

	return nil
}
