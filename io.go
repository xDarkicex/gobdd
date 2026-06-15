package gobdd

import (
	"fmt"
	"os"
	"strings"

	"github.com/xDarkicex/memory"
)

func (b *BDD) PrintDot(f NodeID) { b.FprintDot(os.Stdout, f) }

func (b *BDD) FprintDot(file *os.File, f NodeID) {
	var sb strings.Builder
	sb.WriteString("digraph BDD {\n")
	sb.WriteString("  node [shape=box];\n")
	sb.WriteString(fmt.Sprintf("  %d [label=\"True\" shape=box style=filled fillcolor=lightgreen];\n", trueIdx))
	sb.WriteString(fmt.Sprintf("  %d [label=\"False\" shape=box style=filled fillcolor=lightcoral];\n", falseIdx))
	visited := memory.MustPoolSlice[bool](b.pool, len(b.nodes))
	visited = visited[:len(b.nodes)]
	b.dotWalk(f, &sb, visited)
	sb.WriteString("}\n")
	file.WriteString(sb.String())
}

func (b *BDD) dotWalk(f NodeID, sb *strings.Builder, visited []bool) {
	if f.isTerm() || visited[f] {
		return
	}
	visited[f] = true
	n := b.nodes[f]
	sb.WriteString(fmt.Sprintf("  %d [label=\"x%d\"];\n", f, b.level2var[n.level]))
	sb.WriteString(fmt.Sprintf("  %d -> %d [style=dashed label=\"0\"];\n", f, n.lo))
	sb.WriteString(fmt.Sprintf("  %d -> %d [style=solid label=\"1\"];\n", f, n.hi))
	b.dotWalk(NodeID(n.lo), sb, visited)
	b.dotWalk(NodeID(n.hi), sb, visited)
}

func (b *BDD) PrintTable(f NodeID) string {
	var sb strings.Builder
	total := 1 << uint(b.varCnt)
	for i := 0; i < total; i++ {
		val := NodeID(trueIdx)
		for v := int32(0); v < b.varCnt; v++ {
			bit := (i >> uint(v)) & 1
			term := b.Var(v)
			if bit == 0 {
				term = b.Not(term)
			}
			val = b.And(val, term)
		}
		result := b.And(f, val)
		for v := b.varCnt - 1; v >= 0; v-- {
			sb.WriteByte(byte('0' + byte((i>>uint(v))&1)))
		}
		sb.WriteString(" | ")
		if result == falseIdx {
			sb.WriteString("0\n")
		} else {
			sb.WriteString("1\n")
		}
	}
	return sb.String()
}

func (b *BDD) VarProfile(f NodeID) []int32 {
	profile := memory.MustPoolSlice[int32](b.pool, int(b.varCnt))
	profile = profile[:b.varCnt]
	visited := memory.MustPoolSlice[bool](b.pool, len(b.nodes))
	visited = visited[:len(b.nodes)]
	b.varProfileRec(f, profile, visited)
	return profile
}

func (b *BDD) varProfileRec(f NodeID, profile []int32, visited []bool) {
	if f.isTerm() || visited[f] {
		return
	}
	visited[f] = true
	profile[b.level2var[b.nodes[f].level]]++
	b.varProfileRec(NodeID(b.nodes[f].lo), profile, visited)
	b.varProfileRec(NodeID(b.nodes[f].hi), profile, visited)
}

func (b *BDD) ScanSet(f NodeID) []int32 {
	if f.isTerm() {
		return nil
	}
	var result []int32
	for n := f; !n.isTerm(); n = NodeID(b.nodes[n].hi) {
		result = append(result, b.level2var[b.nodes[n].level])
	}
	return result
}

func (b *BDD) MakeSet(vars []int32) NodeID {
	r := NodeID(trueIdx)
	for i := len(vars) - 1; i >= 0; i-- {
		r = b.And(r, b.Var(vars[i]))
	}
	return r
}
