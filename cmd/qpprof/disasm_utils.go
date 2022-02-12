package main

import (
	"golang.org/x/arch/x86/x86asm"
)

func numInstArgs(inst *x86asm.Inst) int {
	num := 0
	for i := 0; i < len(inst.Args); i++ {
		if inst.Args[i] == nil {
			break
		}
		num++
	}
	return num
}
