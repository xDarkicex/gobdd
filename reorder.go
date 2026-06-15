package gobdd

// Sift reorders variables using Rudell's sifting algorithm.
func (bd *BDD) Sift() {
	bestCount := len(bd.nodes)
	for v := int32(0); v < bd.varCnt; v++ {
		bd.siftUp(v)
		bestPos := int32(0)
		bestCount = len(bd.nodes)
		for pos := int32(1); pos < bd.varCnt; pos++ {
			bd.siftSwap(pos-1, pos)
			if len(bd.nodes) < bestCount {
				bestCount = len(bd.nodes)
				bestPos = pos
			}
		}
		for pos := bd.varCnt - 1; pos > bestPos; pos-- {
			bd.siftSwap(pos-1, pos)
		}
	}
}

func (bd *BDD) siftUp(v int32) {
	for i := v; i > 0; i-- {
		bd.siftSwap(i-1, i)
	}
}

func (bd *BDD) siftSwap(va, vb int32) {
	if va == vb {
		return
	}
	affected := make([]int32, 0, len(bd.nodes)/4)
	for i := int32(2); i < int32(len(bd.nodes)); i++ {
		if bd.nodes[i].vari == va {
			affected = append(affected, i)
		}
	}
	for _, idx := range affected {
		n := bd.nodes[idx]
		lo := bd.cofactorSwap(n.lo, va, vb)
		hi := bd.cofactorSwap(n.hi, va, vb)
		newIdx := bd.unique(vb, lo, hi)
		if newIdx != idx {
			bd.replaceRefs(idx, newIdx)
		}
	}
}

func (bd *BDD) cofactorSwap(f, va, vb int32) int32 {
	if f < 2 {
		return f
	}
	n := bd.nodes[f]
	if n.vari > vb {
		return f
	}
	if n.vari == vb {
		return bd.unique(va, bd.cofactorSwap(n.lo, va, vb), bd.cofactorSwap(n.hi, va, vb))
	}
	if n.vari == va {
		return f
	}
	lo := bd.cofactorSwap(n.lo, va, vb)
	hi := bd.cofactorSwap(n.hi, va, vb)
	return bd.unique(n.vari, lo, hi)
}

func (bd *BDD) replaceRefs(oldIdx, newIdx int32) {
	for i := int32(2); i < int32(len(bd.nodes)); i++ {
		if bd.nodes[i].lo == oldIdx {
			bd.nodes[i].lo = newIdx
		}
		if bd.nodes[i].hi == oldIdx {
			bd.nodes[i].hi = newIdx
		}
	}
}

func (b *BDD) Stats() Stats {
	s := Stats{NodeCount: len(b.nodes), VarCount: int(b.varCnt)}
	for i := int32(2); i < int32(len(b.nodes)); i++ {
		v := b.nodes[i].vari
		if int(v) < len(s.PerVar) {
			s.PerVar[v]++
		}
		if b.nodes[i].lo == falseIdx {
			s.EdgesToFalse++
		}
		if b.nodes[i].hi == trueIdx {
			s.EdgesToTrue++
		}
	}
	return s
}

type Stats struct {
	NodeCount    int
	VarCount     int
	PerVar       [256]int
	EdgesToFalse int
	EdgesToTrue  int
}
