package gobdd

import "strings"

func (b *BDD) ToFormula(f int32) string {
	if f == falseIdx {
		return "false"
	}
	if f == trueIdx {
		return "true"
	}
	var sb strings.Builder
	b.formulaRec(f, &sb, make(map[int32]bool))
	return sb.String()
}

func (b *BDD) formulaRec(f int32, sb *strings.Builder, visited map[int32]bool) {
	if visited[f] {
		return
	}
	visited[f] = true
	if f < 2 {
		if f == falseIdx {
			sb.WriteString("0")
		} else {
			sb.WriteString("1")
		}
		return
	}
	n := b.nodes[f]
	sb.WriteString("(x")
	sb.WriteString(itoa(int(n.vari)))
	sb.WriteString(" ? ")
	b.formulaRec(n.hi, sb, visited)
	sb.WriteString(" : ")
	b.formulaRec(n.lo, sb, visited)
	sb.WriteString(")")
}

func (b *BDD) ToDNF(f int32) string {
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

func (b *BDD) dnfWalk(f int32, literals []string, cubes *[]string) {
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
	hiLit := make([]string, len(literals)+1)
	copy(hiLit, literals)
	hiLit[len(literals)] = "x" + itoa(int(n.vari))
	b.dnfWalk(n.hi, hiLit, cubes)
	loLit := make([]string, len(literals)+1)
	copy(loLit, literals)
	loLit[len(literals)] = "¬x" + itoa(int(n.vari))
	b.dnfWalk(n.lo, loLit, cubes)
}

func (b *BDD) ToCNF(f int32) string {
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
