package main

import (
	"errors"
	"flag"
	"fmt"
	"runtime"
	"strings"
)

func cmdEnrich(fs *flag.FlagSet, args []string) error {
	enricher := &profileEnricher{}

	var toAdd string
	allTags := []string{
		"boundcheck",
		"nilcheck",
	}

	fs.StringVar(&enricher.outputPath, "o", "", `output profile name`)
	fs.StringVar(&enricher.executablePath, "exe", "", `executable file path`)
	fs.StringVar(&enricher.goarch, "arch", runtime.GOARCH,
		`executable arch, as in GOARCH`)
	fs.StringVar(&enricher.exeFormat, "exe-format", "elf",
		`executable file format`)
	fs.StringVar(&toAdd, "add", strings.Join(allTags, ","),
		`add these to the output profile; defaults to all tags`)
	_ = fs.Parse(args)

	argv := fs.Args()
	if len(argv) != 1 {
		return errors.New("expected exactly 1 positional arg: cpu profile filename")
	}
	enricher.profilePath = argv[0]
	if enricher.executablePath == "" {
		return errors.New("-exe flag can't be empty")
	}

	for _, tag := range strings.Split(toAdd, ",") {
		tag = strings.TrimSpace(tag)
		switch tag {
		case "boundcheck":
			enricher.trackBoundcheck = true
		case "nilcheck":
			enricher.trackNilcheck = true
		default:
			return fmt.Errorf("unknown %s tag in -add", tag)
		}
	}

	return enricher.Main()
}
