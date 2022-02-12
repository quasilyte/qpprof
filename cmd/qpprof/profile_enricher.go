package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/pprof/profile"
)

type profileEnricher struct {
	trackBoundcheck bool
	trackNilcheck   bool

	goarch    string
	exeFormat string

	profilePath    string
	outputPath     string
	executablePath string

	exeBytes []byte
	exeInfo  parsedExecutableInfo

	ctx codeAnalysisContext

	boundcheckFunc *profile.Function
	nilcheckFunc   *profile.Function
}

type codeAnalysisContext struct {
	exeInfo             *parsedExecutableInfo
	exeBytes            []byte
	boundcheckAddresses map[int64]struct{}
	nilcheckAddresses   map[int64]struct{}
}

type parsedExecutableInfo struct {
	// boundcheckFuncAddresses contains addresses that are related to
	// bound checking functions.
	boundcheckFuncAddresses map[int64]struct{}
}

func (pe *profileEnricher) Main() error {
	var outputFilename string
	if pe.outputPath == "" {
		outputFilename = pe.profilePath + ".enriched"
	} else {
		outputFilename = pe.outputPath
	}

	data, err := os.ReadFile(pe.profilePath)
	if err != nil {
		return fmt.Errorf("read profile: %v", err)
	}
	p, err := profile.Parse(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("parse profile: %v", err)
	}
	pe.exeBytes, err = os.ReadFile(pe.executablePath)
	if err != nil {
		return fmt.Errorf("read executable: %v", err)
	}

	if pe.trackBoundcheck {
		pe.addBoundcheckFunc(p)
	}
	if pe.trackNilcheck {
		pe.addNilcheckFunc(p)
	}

	pe.exeInfo.boundcheckFuncAddresses = make(map[int64]struct{})
	// TODO: support pe (debug/pe) format as well?
	if pe.exeFormat != "elf" {
		return fmt.Errorf("%s format is not supported", pe.exeFormat)
	}
	if err := parseELF(&pe.exeInfo, p, pe.exeBytes); err != nil {
		return err
	}

	// Analyze (mark) step. Arch-dependent.
	var analyzeFunc func(*codeAnalysisContext, int64) bool
	switch pe.goarch {
	case "amd64":
		analyzeFunc = analyzeAMD64
	default:
		return fmt.Errorf("%s arch (GOARCH) is not supported", pe.goarch)
	}
	pe.ctx.exeInfo = &pe.exeInfo
	pe.ctx.exeBytes = pe.exeBytes
	pe.ctx.boundcheckAddresses = make(map[int64]struct{})
	pe.ctx.nilcheckAddresses = make(map[int64]struct{})
	for _, sample := range p.Sample {
		if len(sample.Location) == 0 {
			continue
		}
		for _, loc := range sample.Location {
			if len(loc.Line) == 0 {
				continue
			}
			m := loc.Mapping
			addr := int64(loc.Address + m.Offset - m.Start)
			if analyzeFunc(&pe.ctx, addr) {
				continue
			}
		}
	}

	boundcheckNum := 0
	boundcheckDuration := time.Duration(0)
	nilcheckNum := 0
	nilcheckDuration := time.Duration(0)
	for _, sample := range p.Sample {
		if len(sample.Location) == 0 {
			continue
		}
		loc := sample.Location[0]
		if len(loc.Line) == 0 {
			continue
		}
		sampleValue := sample.Value[1]
		m := loc.Mapping
		addr := int64(loc.Address + m.Offset - m.Start)
		if _, ok := pe.ctx.boundcheckAddresses[addr]; ok {
			if pe.trackBoundcheck {
				pe.insertLine(loc, pe.boundcheckFunc)
			}
			boundcheckNum++
			boundcheckDuration += time.Duration(sampleValue)
			continue
		}
		if _, ok := pe.ctx.nilcheckAddresses[addr]; ok {
			if pe.trackNilcheck {
				pe.insertLine(loc, pe.nilcheckFunc)
			}
			nilcheckNum++
			nilcheckDuration += time.Duration(sampleValue)
			continue
		}
	}

	log.Printf("runtime.boundcheck: %d samples (%s)\n", boundcheckNum, boundcheckDuration)
	log.Printf("runtime.nilcheck: %d samples (%s)\n", nilcheckNum, nilcheckDuration)

	{
		f, err := os.Create(outputFilename)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := p.Write(f); err != nil {
			return err
		}
	}

	return nil
}

func (pe *profileEnricher) addBoundcheckFunc(p *profile.Profile) {
	id := len(p.Function) + 1
	fn := &profile.Function{
		ID:         uint64(id),
		Name:       "runtime.boundcheck",
		SystemName: "runtime.boundcheck",
		Filename:   "builtins.go",
		StartLine:  1,
	}
	p.Function = append(p.Function, fn)
	pe.boundcheckFunc = fn
}

func (pe *profileEnricher) addNilcheckFunc(p *profile.Profile) {
	id := len(p.Function) + 1
	fn := &profile.Function{
		ID:         uint64(id),
		Name:       "runtime.nilcheck",
		SystemName: "runtime.nilcheck",
		Filename:   "builtins.go",
		StartLine:  1,
	}
	p.Function = append(p.Function, fn)
	pe.nilcheckFunc = fn
}

func (pe *profileEnricher) insertLine(loc *profile.Location, fn *profile.Function) {
	if loc.Line[0].Function == fn {
		return
	}
	lines := make([]profile.Line, len(loc.Line)+1)
	copy(lines[1:], loc.Line)
	lines[0] = profile.Line{
		Function: fn,
		Line:     lines[1].Line,
	}
	loc.Line = lines
}

// This list can vary from one Go version to another.
// TODO: maybe we need a -go version argument and several tables?
var boundcheckFuncNames = map[string]struct{}{
	"runtime.panicIndex":        {},
	"runtime.panicIndexU":       {},
	"runtime.panicSliceAlen":    {},
	"runtime.panicSliceAlenU":   {},
	"runtime.panicSliceAcap":    {},
	"runtime.panicSliceAcapU":   {},
	"runtime.panicSliceB":       {},
	"runtime.panicSliceBU":      {},
	"runtime.panicSlice3Alen":   {},
	"runtime.panicSlice3AlenU":  {},
	"runtime.panicSlice3Acap":   {},
	"runtime.panicSlice3AcapU":  {},
	"runtime.panicSlice3B":      {},
	"runtime.panicSlice3BU":     {},
	"runtime.panicSlice3C":      {},
	"runtime.panicSlice3CU":     {},
	"runtime.panicSliceConvert": {},
}
