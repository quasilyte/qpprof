package main

import (
	"bytes"
	"debug/elf"
	"fmt"

	"github.com/google/pprof/profile"
)

func parseELF(out *parsedExecutableInfo, p *profile.Profile, exeBytes []byte) error {
	f, err := elf.NewFile(bytes.NewReader(exeBytes))
	if err != nil {
		return fmt.Errorf("parse ELF: %v", err)
	}
	symbols, err := f.Symbols()
	if err != nil {
		return fmt.Errorf("fetch ELF symbols: %v", err)
	}

	m := p.Mapping[0]
	for _, sym := range symbols {
		if _, ok := boundcheckFuncNames[sym.Name]; !ok {
			continue
		}
		addr := sym.Value + m.Offset - m.Start
		out.boundcheckFuncAddresses[int64(addr)] = struct{}{}
	}

	return nil
}
