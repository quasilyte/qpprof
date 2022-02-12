package main

import (
	"flag"
	"log"

	"github.com/cespare/subcmd"
)

const progName = "qpprof"

func main() {
	log.SetFlags(0)

	var commands []subcmd.Command

	addCommand := func(handler func(fs *flag.FlagSet, args []string) error, proto subcmd.Command) {
		doFunc := func(args []string) {
			fs := flag.NewFlagSet(progName+" "+proto.Name, flag.ExitOnError)
			if err := handler(fs, args); err != nil {
				log.Fatalf("%s %s: %v", progName, proto.Name, err)
			}
		}
		commands = append(commands, subcmd.Command{
			Name:        proto.Name,
			Description: proto.Description,
			Do:          doFunc,
		})
	}

	addCommand(cmdFlatWithRuntime, subcmd.Command{
		Name:        "flat-with-runtime",
		Description: "like pprof top, but runtime calls are folded into the caller",
	})
	addCommand(cmdFlatWithStdlib, subcmd.Command{
		Name:        "flat-with-stdlib",
		Description: "like pprof top, but stdlib calls are folded into the caller",
	})
	addCommand(cmdEnrich, subcmd.Command{
		Name:        "enrich",
		Description: "create a new CPU with extra information collected from the executable",
	})

	subcmd.Run(commands)
}
