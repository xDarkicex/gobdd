package gobdd

import "github.com/xDarkicex/memory"

func (bd *BDD) Sift() {
	for v := int32(0); v < bd.varCnt; v++ {
		origLevel := bd.var2level[v]
		for pos := origLevel; pos > 0; pos-- {
			bd.swapLevels(pos - 1, pos)
		}
		bestPos := int32(0)
		bestSize := len(bd.nodes)
		for pos := int32(0); pos < bd.varCnt-1; pos++ {
			bd.swapLevels(pos, pos+1)
			if len(bd.nodes) < bestSize {
				bestSize = len(bd.nodes)
				bestPos = pos + 1
			}
		}
		for pos := bd.varCnt - 1; pos > bestPos; pos-- {
			bd.swapLevels(pos - 1, pos)
		}
	}
}

func (bd *BDD) swapLevels(l1, l2 int32) func(oldIdx int32) int32 {
	if l1 > l2 {
		l1, l2 = l2, l1
	}
	if l1+1 != l2 || l1 < 0 || l2 >= bd.varCnt {
		return func(oldIdx int32) int32 { return oldIdx }
	}
	v1 := bd.level2var[l1]
	v2 := bd.level2var[l2]
	bd.level2var[l1] = v2
	bd.level2var[l2] = v1
	bd.var2level[v1] = l2
	bd.var2level[v2] = l1
	for i := range bd.uniq.buckets {
		bd.uniq.buckets[i].ok = false
	}
	bd.uniq.size = 0
	oldNodes := bd.nodes
	newCap := len(oldNodes) + 64
	newNodes := memory.MustPoolSlice[bddNode](bd.pool, newCap)
	newNodes = newNodes[:2]
	newNodes[0] = oldNodes[0]
	newNodes[1] = oldNodes[1]
	bd.nodes = newNodes
	remap := memory.MustPoolSlice[int32](bd.pool, len(oldNodes))
	remap = remap[:len(oldNodes)]
	for i := range remap {
		remap[i] = -1
	}
	remap[0] = 0
	remap[1] = 1
	for level := bd.varCnt - 1; level >= 0; level-- {
		for oldIdx := int32(2); oldIdx < int32(len(oldNodes)); oldIdx++ {
			if oldNodes[oldIdx].level == level {
				lo := remap[oldNodes[oldIdx].lo]
				hi := remap[oldNodes[oldIdx].hi]
				newIdx := bd.addNode(level, lo, hi)
				remap[oldIdx] = newIdx
			}
		}
	}
	// Auto-remap externally referenced nodes.
	bd.applyRemap(func(oldIdx int32) int32 {
		if oldIdx >= 0 && int(oldIdx) < len(remap) {
			return remap[oldIdx]
		}
		return oldIdx
	})
	return func(oldIdx int32) int32 {
		if oldIdx >= 0 && int(oldIdx) < len(remap) {
			return remap[oldIdx]
		}
		return oldIdx
	}
}

func (bd *BDD) addNode(level, lo, hi int32) int32 {
	if lo == hi {
		return lo
	}
	idx := int32(len(bd.nodes))
	bd.nodes = append(bd.nodes, bddNode{level: level, lo: lo, hi: hi})
	bd.uniq.put(level, lo, hi, idx)
	return idx
}

func (b *BDD) Stats() Stats {
	s := Stats{NodeCount: len(b.nodes), VarCount: int(b.varCnt)}
	for i := int32(2); i < int32(len(b.nodes)); i++ {
		vari := b.level2var[b.nodes[i].level]
		if int(vari) < len(s.PerVar) {
			s.PerVar[vari]++
		}
		if b.nodes[i].lo == int32(falseIdx) {
			s.EdgesToFalse++
		}
		if b.nodes[i].hi == int32(trueIdx) {
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
