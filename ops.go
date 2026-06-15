package gobdd

import (
	"math"

	"github.com/xDarkicex/memory"
)

func (b *BDD) Nithvar(v int32) NodeID { return b.Not(b.Var(v)) }
func (b *BDD) Diff(f, g NodeID) NodeID { return b.And(f, b.Not(g)) }
func (b *BDD) Less(f, g NodeID) NodeID { return b.And(b.Not(f), g) }
func (b *BDD) InvImp(f, g NodeID) NodeID { return b.Or(f, g) }

// Constrain returns the generalized cofactor of f with respect to c.
func (b *BDD) Constrain(f, c NodeID) NodeID {
	if c == trueIdx {
		return f
	}
	if c == falseIdx {
		return trueIdx
	}
	if f.isTerm() {
		return f
	}
	if f == c {
		return trueIdx
	}
	if r, ok := b.cache.get(opConstrain.cacheKey(), int32(f), int32(c)); ok {
		return NodeID(r)
	}

	nf := b.nodes[f]
	nc := b.nodes[c]

	if nc.level < nf.level {
		r := b.Constrain(f, b.cofactor(c, nc.level, true))
		b.cache.put(opConstrain.cacheKey(), int32(f), int32(c), int32(r))
		return r
	}
	if nf.level < nc.level {
		lo := b.Constrain(NodeID(nf.lo), c)
		hi := b.Constrain(NodeID(nf.hi), c)
		if lo == hi {
			b.cache.put(opConstrain.cacheKey(), int32(f), int32(c), int32(lo))
			return lo
		}
		r := b.unique(nf.level, int32(lo), int32(hi))
		b.cache.put(opConstrain.cacheKey(), int32(f), int32(c), int32(r))
		return r
	}
	if nc.lo == int32(falseIdx) {
		r := b.Constrain(NodeID(nf.hi), NodeID(nc.hi))
		b.cache.put(opConstrain.cacheKey(), int32(f), int32(c), int32(r))
		return r
	}
	if nc.hi == int32(falseIdx) {
		r := b.Constrain(NodeID(nf.lo), NodeID(nc.lo))
		b.cache.put(opConstrain.cacheKey(), int32(f), int32(c), int32(r))
		return r
	}
	lo := b.Constrain(NodeID(nf.lo), NodeID(nc.lo))
	hi := b.Constrain(NodeID(nf.hi), NodeID(nc.hi))
	if lo == hi {
		b.cache.put(opConstrain.cacheKey(), int32(f), int32(c), int32(lo))
		return lo
	}
	r := b.unique(nf.level, int32(lo), int32(hi))
	b.cache.put(opConstrain.cacheKey(), int32(f), int32(c), int32(r))
	return r
}

func (b *BDD) Simplify(f, d NodeID) NodeID {
	if d == trueIdx || f.isTerm() {
		return f
	}
	if f == d {
		return trueIdx
	}
	if d == falseIdx {
		return falseIdx
	}
	if r, ok := b.cache.get(opSimplify.cacheKey(), int32(f), int32(d)); ok {
		return NodeID(r)
	}
	nf := b.nodes[f]
	nd := b.nodes[d]
	if nd.level < nf.level {
		r := b.Simplify(f, b.cofactor(d, nd.level, true))
		b.cache.put(opSimplify.cacheKey(), int32(f), int32(d), int32(r))
		return r
	}
	if nf.level < nd.level {
		lo := b.Simplify(NodeID(nf.lo), d)
		hi := b.Simplify(NodeID(nf.hi), d)
		r := b.resolve(nf.level, lo, hi)
		b.cache.put(opSimplify.cacheKey(), int32(f), int32(d), int32(r))
		return r
	}
	lo := b.Simplify(NodeID(nf.lo), NodeID(nd.lo))
	hi := b.Simplify(NodeID(nf.hi), NodeID(nd.hi))
	r := b.resolve(nf.level, lo, hi)
	b.cache.put(opSimplify.cacheKey(), int32(f), int32(d), int32(r))
	return r
}

func (b *BDD) resolve(v int32, lo, hi NodeID) NodeID {
	if lo == hi {
		return lo
	}
	return b.unique(v, int32(lo), int32(hi))
}

func (b *BDD) AppEx(f, g NodeID, op OpCode, vars []int32) NodeID {
	return b.ExistsAll(b.Apply(f, g, op), vars)
}

func (b *BDD) AppAll(f, g NodeID, op OpCode, vars []int32) NodeID {
	return b.ForAllVars(b.Apply(f, g, op), vars)
}

func (b *BDD) RelProd(f, g NodeID, vars []int32) NodeID {
	return b.AppEx(f, g, opAnd, vars)
}

func (b *BDD) AppExBDD(f, g NodeID, op OpCode, varset NodeID) NodeID {
	return b.ExistSet(b.Apply(f, g, op), varset)
}

func (b *BDD) AppAllBDD(f, g NodeID, op OpCode, varset NodeID) NodeID {
	return b.ForAllSet(b.Apply(f, g, op), varset)
}

func (b *BDD) RelProdBDD(f, g NodeID, varset NodeID) NodeID {
	return b.AppExBDD(f, g, opAnd, varset)
}

func (b *BDD) Apply(f, g NodeID, op OpCode) NodeID {
	switch op {
	case opAnd:
		return b.And(f, g)
	case opXor:
		return b.Xor(f, g)
	case opOr:
		return b.Or(f, g)
	case opNand:
		return b.Nand(f, g)
	case opNor:
		return b.Nor(f, g)
	case opImp:
		return b.Implies(f, g)
	case opBiimp:
		return b.Equiv(f, g)
	case opDiff:
		return b.Diff(f, g)
	case opLess:
		return b.Less(f, g)
	case opInvImp:
		return b.InvImp(f, g)
	}
	return falseIdx
}

// IBuildCube builds a cube from an integer bitmask.
// The width low-order bits of value encode the polarity of vars[0..width-1]:
// bit=1 → positive literal, bit=0 → negated literal.
// The MSB corresponds to vars[0], LSB to vars[width-1].
// Matches buddy bdd_ibuildcube (bddop.c:357). CC=3.
func (b *BDD) IBuildCube(value int, width int, vars []int32) NodeID {
	r := NodeID(trueIdx)
	for z := 0; z < width; z++ {
		v := b.Var(vars[width-z-1])
		if value&1 == 0 {
			v = b.Not(v)
		}
		r = b.And(r, v)
		value >>= 1
	}
	return r
}

func (b *BDD) BuildCube(vars []int32, positive []bool) NodeID {
	r := NodeID(trueIdx)
	for i := range vars {
		v := b.Var(vars[i])
		if !positive[i] {
			v = b.Not(v)
		}
		r = b.And(r, v)
	}
	return r
}

func (b *BDD) PathCount(f NodeID) float64 {
	counts := memory.MustPoolSlice[float64](b.pool, len(b.nodes))
	counts = counts[:len(b.nodes)]
	for i := range counts {
		counts[i] = -1
	}
	return b.pathCountRec(f, counts)
}

func (b *BDD) pathCountRec(f NodeID, counts []float64) float64 {
	if f == falseIdx {
		return 0
	}
	if f == trueIdx {
		return 1
	}
	if counts[f] >= 0 {
		return counts[f]
	}
	n := b.nodes[f]
	counts[f] = b.pathCountRec(NodeID(n.lo), counts) + b.pathCountRec(NodeID(n.hi), counts)
	return counts[f]
}

func (b *BDD) SatCountLn(f NodeID) float64 {
	c := b.SatisfyCount(f)
	if c == 0 {
		return 0
	}
	return math.Log(float64(c))
}

func (b *BDD) SatCountSet(f NodeID, vars []int32) uint64 {
	return b.SatisfyCount(b.ExistsAll(f, vars))
}

func (b *BDD) SatCountLnSet(f NodeID, vars []int32) float64 {
	return b.SatCountLn(b.ExistsAll(f, vars))
}

func (c OpCode) cacheKey() int32 {
	return 1000 + int32(c)
}
