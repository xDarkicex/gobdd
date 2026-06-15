package gobdd

import "github.com/xDarkicex/memory"

type Pair struct {
	result []NodeID
	last   int32
	id     int32
	bdd    *BDD
}

func (b *BDD) NewPair() *Pair {
	result := memory.MustPoolSlice[NodeID](b.pool, int(b.varCnt))
	result = result[:b.varCnt]
	for i := range result {
		result[i] = b.Var(int32(i))
	}
	return &Pair{result: result, bdd: b, last: -1}
}

func (p *Pair) Set(oldVar int32, newVal NodeID) {
	if oldVar >= 0 && int(oldVar) < len(p.result) {
		level := p.bdd.var2level[oldVar]
		p.result[level] = newVal
		if level > p.last {
			p.last = level
		}
	}
}

func (p *Pair) SetVar(oldVar, newVar int32) { p.Set(oldVar, p.bdd.Var(newVar)) }


// SetVars sets multiple variable-to-variable mappings.
// oldVars and newVars are variable indices.
// Matches buddy bdd_setpairs (pairs.c:252).
func (p *Pair) SetVars(oldVars, newVars []int32, n int) {
	for i := 0; i < n; i++ {
		p.SetVar(oldVars[i], newVars[i])
	}
}

func (p *Pair) Reset() {
	for i := int32(0); i < p.bdd.varCnt; i++ {
		p.result[i] = p.bdd.Var(i)
	}
	p.last = 0
}

func (p *Pair) SetAll(oldVars []int32, newVals []NodeID, n int) {
	for i := 0; i < n; i++ {
		p.Set(oldVars[i], newVals[i])
	}
}

func (b *BDD) SwapVar(v1, v2 int32) func(oldIdx NodeID) NodeID {
	if v1 > v2 {
		v1, v2 = v2, v1
	}
	remap := b.swapLevels(b.var2level[v1], b.var2level[v2])
	return func(oldIdx NodeID) NodeID { return NodeID(remap(int32(oldIdx))) }
}

func (b *BDD) Replace(f NodeID, p *Pair) NodeID {
	if f.isTerm() {
		return f
	}
	n := b.nodes[f]
	lo := b.Replace(NodeID(n.lo), p)
	hi := b.Replace(NodeID(n.hi), p)
	varIdx := b.level2var[n.level]
	newVar := p.result[varIdx]
	if newVar != b.Var(varIdx) {
		return b.Compose(b.unique(n.level, int32(lo), int32(hi)), varIdx, newVar)
	}
	if lo != NodeID(n.lo) || hi != NodeID(n.hi) {
		return b.unique(n.level, int32(lo), int32(hi))
	}
	return f
}

func (b *BDD) VecCompose(f NodeID, p *Pair) NodeID {
	r := f
	for v := int32(0); v < b.varCnt; v++ {
		if p.result[v] != b.Var(v) {
			r = b.Compose(r, v, p.result[v])
		}
	}
	return r
}
