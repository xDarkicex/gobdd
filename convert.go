package gobdd

import (
	"strings"

	"github.com/xDarkicex/memory"
)

func (b *BDD) ToFormula(f NodeID) string {
	if f == falseIdx {
		return "false"
	}
	if f == trueIdx {
		return "true"
	}
	var sb strings.Builder
	visited := memory.MustPoolSlice[bool](b.pool, len(b.nodes))
	visited = visited[:len(b.nodes)]
	b.formulaRec(f, &sb, visited)
	return sb.String()
}

func (b *BDD) formulaRec(f NodeID, sb *strings.Builder, visited []bool) {
	if visited[f] {
		return
	}
	visited[f] = true
	if f.isTerm() {
		if f == falseIdx {
			sb.WriteString("0")
		} else {
			sb.WriteString("1")
		}
		return
	}
	n := b.nodes[f]
	sb.WriteString("(x")
	sb.WriteString(itoa(int(b.level2var[n.level])))
	sb.WriteString(" ? ")
	b.formulaRec(NodeID(n.hi), sb, visited)
	sb.WriteString(" : ")
	b.formulaRec(NodeID(n.lo), sb, visited)
	sb.WriteString(")")
}

func (b *BDD) ToDNF(f NodeID) string {
	if f == falseIdx {
		return "false"
	}
	if f == trueIdx {
		return "true"
	}
	var cubes []string
	b.dnfWalk(f, nil, &cubes)
	if len(cubes) == 0 {
		return "true"
	}
	return strings.Join(cubes, " ∨ ")
}

func (b *BDD) dnfWalk(f NodeID, literals []string, cubes *[]string) {
	if f == falseIdx {
		return
	}
	if f == trueIdx {
		if len(literals) == 0 {
			*cubes = append(*cubes, "true")
			return
		}
		*cubes = append(*cubes, "("+strings.Join(literals, " ∧ ")+")")
		return
	}
	n := b.nodes[f]
	vname := "x" + itoa(int(b.level2var[n.level]))
	hiLit := memory.MustPoolSlice[string](b.pool, len(literals)+1)
	hiLit = hiLit[:len(literals)+1]
	copy(hiLit, literals)
	hiLit[len(literals)] = vname
	b.dnfWalk(NodeID(n.hi), hiLit, cubes)
	loLit := memory.MustPoolSlice[string](b.pool, len(literals)+1)
	loLit = loLit[:len(literals)+1]
	copy(loLit, literals)
	loLit[len(literals)] = "¬" + vname
	b.dnfWalk(NodeID(n.lo), loLit, cubes)
}

func (b *BDD) ToCNF(f NodeID) string {
	nf := b.Not(f)
	dnf := b.ToDNF(nf)
	if dnf == "true" {
		return "false"
	}
	if dnf == "false" {
		return "true"
	}
	return "¬(" + dnf + ")"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
