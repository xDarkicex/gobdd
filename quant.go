package gobdd

import "github.com/xDarkicex/memory"

type quantMode int32

const (
	quantExist  quantMode = 0
	quantForAll quantMode = 1
	quantUnique quantMode = 2
)

type varsetIndex struct {
	set    []int32
	id     int32
	last   int32
	signed bool
}

func (b *BDD) newVarsetIndex(varset NodeID, signed bool) *varsetIndex {
	vs := &varsetIndex{
		set:    memory.MustPoolSlice[int32](b.pool, int(b.varCnt)),
		id:     1,
		signed: signed,
	}
	vs.set = vs.set[:b.varCnt]
	for i := range vs.set {
		vs.set[i] = 0
	}
	if varset.isTerm() {
		return vs
	}
	if !signed {
		vs.last = b.nodes[varset].level
		for n := varset; !n.isTerm(); n = NodeID(b.nodes[n].hi) {
			vs.set[b.nodes[n].level] = vs.id
		}
	} else {
		vs.last = b.nodes[varset].level
		for n := varset; !n.isTerm(); {
			if b.nodes[n].lo == int32(falseIdx) {
				vs.set[b.nodes[n].level] = vs.id
				n = NodeID(b.nodes[n].hi)
			} else {
				vs.set[b.nodes[n].level] = -vs.id
				n = NodeID(b.nodes[n].lo)
			}
		}
	}
	return vs
}

func (vs *varsetIndex) inSet(level int32) bool {
	if level > vs.last {
		return false
	}
	if vs.signed {
		return vs.set[level] == vs.id || vs.set[level] == -vs.id
	}
	return vs.set[level] == vs.id
}

func (vs *varsetIndex) polarity(level int32) int32 {
	if level > vs.last {
		return 0
	}
	val := vs.set[level]
	if val == vs.id {
		return 1
	}
	if val == -vs.id {
		return -1
	}
	return 0
}

func (b *BDD) SatOne(f NodeID) NodeID {
	if f.isTerm() {
		return f
	}
	return b.satOneRec(f)
}

func (b *BDD) satOneRec(f NodeID) NodeID {
	if f.isTerm() {
		return f
	}
	n := b.nodes[f]
	if n.lo == int32(falseIdx) {
		res := b.satOneRec(NodeID(n.hi))
		return b.unique(n.level, int32(falseIdx), int32(res))
	}
	res := b.satOneRec(NodeID(n.lo))
	return b.unique(n.level, int32(res), int32(falseIdx))
}

func (b *BDD) RestrictBDD(f, c NodeID) NodeID {
	if c.isTerm() {
		return f
	}
	vs := b.newVarsetIndex(c, true)
	return b.restrictRec(f, vs)
}

func (b *BDD) restrictRec(f NodeID, vs *varsetIndex) NodeID {
	if f.isTerm() || b.nodes[f].level > vs.last {
		return f
	}
	level := b.nodes[f].level
	n := b.nodes[f]
	if vs.inSet(level) {
		if vs.polarity(level) > 0 {
			return b.restrictRec(NodeID(n.hi), vs)
		}
		return b.restrictRec(NodeID(n.lo), vs)
	}
	lo := b.restrictRec(NodeID(n.lo), vs)
	hi := b.restrictRec(NodeID(n.hi), vs)
	return b.unique(level, int32(lo), int32(hi))
}

func (b *BDD) ExistSet(f, varset NodeID) NodeID {
	if varset.isTerm() {
		return f
	}
	vs := b.newVarsetIndex(varset, false)
	return b.quantRec(f, quantExist, vs)
}

func (b *BDD) ForAllSet(f, varset NodeID) NodeID {
	if varset.isTerm() {
		return f
	}
	vs := b.newVarsetIndex(varset, false)
	return b.quantRec(f, quantForAll, vs)
}

func (b *BDD) quantRec(f NodeID, mode quantMode, vs *varsetIndex) NodeID {
	if f.isTerm() || b.nodes[f].level > vs.last {
		return f
	}
	level := b.nodes[f].level
	n := b.nodes[f]
	lo := b.quantRec(NodeID(n.lo), mode, vs)
	hi := b.quantRec(NodeID(n.hi), mode, vs)
	if vs.inSet(level) {
		switch mode {
		case quantExist:
			return b.Or(lo, hi)
		case quantForAll:
			return b.And(lo, hi)
		case quantUnique:
			return b.Xor(lo, hi)
		}
	}
	return b.unique(level, int32(lo), int32(hi))
}

func (b *BDD) Unique(f, varset NodeID) NodeID {
	if varset.isTerm() {
		return f
	}
	vs := b.newVarsetIndex(varset, false)
	return b.quantRec(f, quantUnique, vs)
}

func (b *BDD) AppUni(l, r NodeID, varset NodeID, op OpCode) NodeID {
	return b.Unique(b.Apply(l, r, op), varset)
}
