package gobdd

import "github.com/xDarkicex/memory"

var satPolarity bool

func (b *BDD) FullSatOne(f NodeID) []bool {
	if f == falseIdx {
		return nil
	}
	assign := memory.MustPoolSlice[bool](b.pool, int(b.varCnt))
	assign = assign[:b.varCnt]
	b.fullSatWalk(f, assign)
	return assign
}

// FullSatOneBDD returns a minterm BDD that implies f, with a variable at every level.
// Matches buddy bdd_fullsatone (bddop.c:2285). CC=4.
func (b *BDD) FullSatOneBDD(f NodeID) NodeID {
	if f == falseIdx {
		return falseIdx
	}
	res := b.fullSatOneBDDRec(f)
	// Add unconstrained variables in negated form (buddy: fills levels above root).
	support := b.Support(res)
	seen := memory.MustPoolSlice[bool](b.pool, int(b.varCnt))
	seen = seen[:b.varCnt]
	for _, v := range support {
		seen[v] = true
	}
	for v := int32(0); v < b.varCnt; v++ {
		if !seen[v] {
			res = b.And(res, b.Not(b.Var(v)))
		}
	}
	return res
}

func (b *BDD) fullSatOneBDDRec(f NodeID) NodeID {
	if f.isTerm() {
		return f
	}
	n := b.nodes[f]
	if n.lo != int32(falseIdx) {
		res := b.fullSatOneBDDRec(NodeID(n.lo))
		return b.unique(n.level, int32(res), int32(falseIdx))
	}
	res := b.fullSatOneBDDRec(NodeID(n.hi))
	return b.unique(n.level, int32(falseIdx), int32(res))
}

func (b *BDD) fullSatWalk(f NodeID, assign []bool) {
	for !f.isTerm() {
		n := b.nodes[f]
		v := b.level2var[n.level]
		if n.lo != int32(falseIdx) {
			assign[v] = false
			f = NodeID(n.lo)
		} else {
			assign[v] = true
			f = NodeID(n.hi)
		}
	}
}

func (b *BDD) SatOneSet(f, varset NodeID, pol bool) NodeID {
	if f == falseIdx {
		return falseIdx
	}
	satPolarity = pol
	return b.satOneSetRec(f, varset)
}

func (b *BDD) satOneSetRec(f, varset NodeID) NodeID {
	if f.isTerm() && varset.isTerm() {
		return f
	}
	nfVar := b.varCnt
	if !f.isTerm() {
		nfVar = b.nodes[f].level
	}
	nvVar := b.varCnt
	if !varset.isTerm() {
		nvVar = b.nodes[varset].level
	}
	if nfVar < nvVar {
		if b.nodes[f].lo == int32(falseIdx) {
			res := b.satOneSetRec(NodeID(b.nodes[f].hi), varset)
			return b.unique(b.nodes[f].level, int32(falseIdx), int32(res))
		}
		res := b.satOneSetRec(NodeID(b.nodes[f].lo), varset)
		return b.unique(b.nodes[f].level, int32(res), int32(falseIdx))
	}
	if nvVar < nfVar {
		res := b.satOneSetRec(f, NodeID(b.nodes[varset].hi))
		if satPolarity {
			return b.unique(b.nodes[varset].level, int32(falseIdx), int32(res))
		}
		return b.unique(b.nodes[varset].level, int32(res), int32(falseIdx))
	}
	if b.nodes[f].lo == int32(falseIdx) {
		res := b.satOneSetRec(NodeID(b.nodes[f].hi), NodeID(b.nodes[varset].hi))
		return b.unique(b.nodes[f].level, int32(falseIdx), int32(res))
	}
	res := b.satOneSetRec(NodeID(b.nodes[f].lo), NodeID(b.nodes[varset].hi))
	return b.unique(b.nodes[f].level, int32(res), int32(falseIdx))
}

func (b *BDD) AllSat(f NodeID, handler func(assign []bool)) {
	if f == falseIdx {
		return
	}
	profile := memory.MustPoolSlice[int8](b.pool, int(b.varCnt))
	profile = profile[:b.varCnt]
	for i := range profile {
		profile[i] = -1
	}
	b.allSatRec(f, profile, handler)
}

func (b *BDD) allSatRec(f NodeID, profile []int8, handler func([]bool)) {
	if f == falseIdx {
		return
	}
	if f == trueIdx {
		assign := memory.MustPoolSlice[bool](b.pool, int(b.varCnt))
		assign = assign[:b.varCnt]
		b.enumerateMinTerm(profile, 0, assign, handler)
		return
	}
	n := b.nodes[f]
	varIdx := b.level2var[n.level]
	if n.lo != int32(falseIdx) {
		saved := profile[varIdx]
		profile[varIdx] = 0
		b.allSatRec(NodeID(n.lo), profile, handler)
		profile[varIdx] = saved
	}
	if n.hi != int32(falseIdx) {
		saved := profile[varIdx]
		profile[varIdx] = 1
		b.allSatRec(NodeID(n.hi), profile, handler)
		profile[varIdx] = saved
	}
}

func (b *BDD) enumerateMinTerm(profile []int8, pos int32, assign []bool, handler func([]bool)) {
	if pos >= b.varCnt {
		dst := memory.MustPoolSlice[bool](b.pool, int(b.varCnt))
		dst = dst[:b.varCnt]
		copy(dst, assign)
		handler(dst)
		return
	}
	switch profile[pos] {
	case 0:
		assign[pos] = false
		b.enumerateMinTerm(profile, pos+1, assign, handler)
	case 1:
		assign[pos] = true
		b.enumerateMinTerm(profile, pos+1, assign, handler)
	default:
		assign[pos] = false
		b.enumerateMinTerm(profile, pos+1, assign, handler)
		assign[pos] = true
		b.enumerateMinTerm(profile, pos+1, assign, handler)
	}
}
