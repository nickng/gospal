package migoinfer

import (
	"strings"

	"github.com/nickng/migo/v3"
	"golang.org/x/tools/go/ssa"
)

func migoNewMem(name string) migo.Statement {
	return &migo.NewMem{Name: name}
}

func migoRead(name string) migo.Statement {
	return &migo.MemRead{Name: name}
}

func migoWrite(name string) migo.Statement {
	return &migo.MemWrite{Name: name}
}

func prefix(instr ssa.Instruction) string {
	var sb strings.Builder
	if parent := instr.Parent(); parent != nil {
		if pkg := parent.Package(); pkg != nil {
			sb.WriteString(pkg.Pkg.Path())
		} else {
			sb.WriteString("_")
		}
		sb.WriteString(".")
		sb.WriteString(parent.Name())
	} else {
		sb.WriteString("_")
	}
	sb.WriteString(".")
	return sb.String()
}
