package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/google/pprof/profile"
	"github.com/quasilyte/pprofutil"
)

func flatAggregate(fs *flag.FlagSet, args []string, fold func(pprofutil.Symbol) bool) error {
	agg := &flatAggregator{
		foldPredicate: fold,
	}

	fs.UintVar(&agg.n, "n", 100,
		`show top n nodes`)
	fs.StringVar(&agg.filter, "filter", ".*",
		`regexp used to filter symbols that will be shown`)
	_ = fs.Parse(args)

	argv := fs.Args()
	if len(argv) != 1 {
		return errors.New("expected exactly 1 positional arg: cpu profile filename")
	}
	agg.profilePath = argv[0]

	return agg.Main()
}

type flatAggregator struct {
	foldPredicate func(pprofutil.Symbol) bool
	filter        string
	profilePath   string

	n uint
}

func (agg *flatAggregator) Main() error {
	var filterRe *regexp.Regexp
	if agg.filter != ".*" {
		re, err := regexp.Compile(agg.filter)
		if err != nil {
			return fmt.Errorf("bad filter regexp: %v", err)
		}
		filterRe = re
	}

	data, err := os.ReadFile(agg.profilePath)
	if err != nil {
		return fmt.Errorf("read profile: %v", err)
	}
	p, err := profile.Parse(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("parse profile: %v", err)
	}

	// Collect all flat values.
	perFunc := make(map[string]int64)
	for _, s := range p.Sample {
		sampleValue := s.Value[1]
		if len(s.Location) == 0 || len(s.Location[0].Line) == 0 {
			continue
		}
		l := s.Location[0].Line[0]
		perFunc[l.Function.Name] += sampleValue
	}

	// Add folded values to the first non-fold caller.
	for _, s := range p.Sample {
		sampleValue := s.Value[1]
		if len(s.Location) == 0 || len(s.Location[0].Line) == 0 {
			continue
		}
		l := s.Location[0].Line[0]
		sym := pprofutil.ParseFuncName(l.Function.Name)
		if !agg.foldPredicate(sym) {
			continue
		}
		caller := agg.findNode(s, func(l *profile.Line) bool {
			sym := pprofutil.ParseFuncName(l.Function.Name)
			return !agg.foldPredicate(sym)
		})
		if caller != nil {
			perFunc[caller.Function.Name] += sampleValue
		}
	}

	if filterRe != nil {
		for k := range perFunc {
			if !filterRe.MatchString(k) {
				delete(perFunc, k)
			}
		}
	}

	type keyKalue struct {
		key string
		val int64
	}
	{
		var sorted []keyKalue
		for k, v := range perFunc {
			if agg.foldPredicate(pprofutil.ParseFuncName(k)) {
				continue
			}
			sorted = append(sorted, keyKalue{key: k, val: v})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].val > sorted[j].val
		})
		n := int(agg.n)
		if n >= len(sorted) {
			n = len(sorted)
		}
		for i, kv := range sorted[:n] {
			fmt.Printf("%3d | %-7s %s\n", i+1, time.Duration(kv.val), kv.key)
		}
	}

	return nil
}

func (agg *flatAggregator) findNode(s *profile.Sample, pred func(*profile.Line) bool) *profile.Line {
	for _, loc := range s.Location {
		for i := range loc.Line {
			if pred(&loc.Line[i]) {
				return &loc.Line[i]
			}
		}
	}
	return nil
}
