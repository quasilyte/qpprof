package main

import (
	"flag"

	"github.com/quasilyte/pprofutil"
	"github.com/quasilyte/stdinfo"
)

func cmdFlatWithStdlib(fs *flag.FlagSet, args []string) error {
	stdlibSet := make(map[string]struct{}, len(stdinfo.PackagesList))
	for _, pkg := range stdinfo.PackagesList {
		stdlibSet[pkg.Path] = struct{}{}
	}
	return flatAggregate(fs, args, func(s pprofutil.Symbol) bool {
		_, ok := stdlibSet[s.PkgPath]
		return ok || s.PkgPath == ""
	})
}
