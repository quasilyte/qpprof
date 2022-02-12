package main

import (
	"flag"

	"github.com/quasilyte/pprofutil"
)

func cmdFlatWithRuntime(fs *flag.FlagSet, args []string) error {
	return flatAggregate(fs, args, func(s pprofutil.Symbol) bool {
		return s.PkgPath == "runtime" || s.PkgName == ""
	})
}
