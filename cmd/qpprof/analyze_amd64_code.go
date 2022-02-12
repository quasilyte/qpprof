package main

import (
	"golang.org/x/arch/x86/x86asm"
)

func analyzeAMD64(ctx *codeAnalysisContext, addr int64) bool {
	var a amd64analyzer
	return a.markBoundcheck(ctx, addr) || a.markNilcheck(ctx, addr)
}

type amd64analyzer struct{}

func (a *amd64analyzer) markBoundcheck(ctx *codeAnalysisContext, addr int64) bool {
	//   TEST | CMP
	//   JUMP x
	// x:
	//   [...MOV] args
	//   CALL panicindexfunc

	code := ctx.exeBytes
	cmp, err := x86asm.Decode(code[addr:], 64)
	if err != nil {
		return false
	}
	if cmp.Op != x86asm.CMP && cmp.Op != x86asm.TEST {
		return false
	}
	jmp, err := x86asm.Decode(code[addr+int64(cmp.Len):], 64)
	if err != nil {
		return false
	}
	jumpFrom := int64(addr) + int64(cmp.Len)
	var jumpTo int64
	switch jmp.Op {
	case x86asm.JBE, x86asm.JB, x86asm.JA:
		rel, ok := jmp.Args[0].(x86asm.Rel)
		if !ok {
			return false
		}
		jumpTo = jumpFrom + int64(rel) + int64(jmp.Len)
	default:
		return false
	}
	pos := jumpTo
	var call x86asm.Inst
	// Some args could be already in the appropriate registers,
	// but in the general case there could be a few MOV-like
	// instructions to place args for the panic-like call.
	// But eventually we need to reach the CALL instruction.
	for i := 0; i < 5; i++ {
		inst, err := x86asm.Decode(code[pos:], 64)
		if err != nil {
			return false
		}
		if inst.Op == x86asm.CALL {
			call = inst
			break
		}
		switch inst.Op {
		case x86asm.NOP:
			// OK, could be the padding.
		case x86asm.MOV, x86asm.LEA:
			// OK, probably moving args.
		case x86asm.XOR:
			// See if it's XOR reg, reg which is basically MOV $0.
			reg1, ok1 := inst.Args[0].(x86asm.Reg)
			reg2, ok2 := inst.Args[1].(x86asm.Reg)
			if !ok1 || !ok2 || reg1 != reg2 {
				return false
			}
		default:
			return false
		}
		pos += int64(inst.Len)
	}
	if call.Op != x86asm.CALL {
		return false
	}
	rel, ok := call.Args[0].(x86asm.Rel)
	if !ok {
		return false
	}
	if _, ok := ctx.exeInfo.boundcheckFuncAddresses[pos+int64(rel)+int64(call.Len)]; !ok {
		return false
	}

	ctx.boundcheckAddresses[addr] = struct{}{}
	ctx.boundcheckAddresses[addr+int64(cmp.Len)] = struct{}{}

	return false
}

func (a *amd64analyzer) markNilcheck(ctx *codeAnalysisContext, addr int64) bool {
	// TESTB AX, (reg)

	code := ctx.exeBytes
	inst, err := x86asm.Decode(code[addr:], 64)
	if err != nil {
		return false
	}
	if inst.Op != x86asm.TEST || a.numInstArgs(&inst) != 2 {
		return false
	}
	arg1, ok := inst.Args[0].(x86asm.Mem)
	if !ok || arg1.Disp != 0 || arg1.Index != 0 {
		return false
	}
	arg2, ok := inst.Args[1].(x86asm.Reg)
	if !ok || arg2 != x86asm.AL {
		return false
	}

	ctx.nilcheckAddresses[addr] = struct{}{}

	return true
}

func (a *amd64analyzer) numInstArgs(inst *x86asm.Inst) int {
	num := 0
	for i := 0; i < len(inst.Args); i++ {
		if inst.Args[i] == nil {
			break
		}
		num++
	}
	return num
}
