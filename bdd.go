package gobdd

import (
	"math/bits"

	"github.com/xDarkicex/memory"
)

// BDD is a manager for Binary Decision Diagrams.
type BDD struct {
	nodes     []bddNode
	varCnt    int32
	var2level []int32
	level2var []int32
	pool      *memory.Pool
	uniq      *uniqTable
	cache     *opCache
	refCount  []int32 // external reference count per node (nil until first AddRef)
}

type bddNode struct {
	level int32
	lo    int32
	hi    int32
}

// New creates a BDD manager with the given number of variables.
func New(numVars int, pool *memory.Pool) *BDD {
	cap := numVars*256 + 16
	nodes := memory.MustPoolSlice[bddNode](pool, cap)
	nodes = nodes[:2]
	nodes[0] = bddNode{level: -1, lo: -1, hi: -1}
	nodes[1] = bddNode{level: -1, lo: -1, hi: -1}

	var2level := memory.MustPoolSlice[int32](pool, numVars)
	var2level = var2level[:numVars]
	level2var := memory.MustPoolSlice[int32](pool, numVars)
	level2var = level2var[:numVars]
	for i := 0; i < numVars; i++ {
		var2level[i] = int32(i)
		level2var[i] = int32(i)
	}

	return &BDD{
		nodes:     nodes,
		varCnt:    int32(numVars),
		var2level: var2level,
		level2var: level2var,
		pool:      pool,
		uniq:      newUniqTable(pool),
		cache:     newOpCache(pool),
	}
}

func (b *BDD) VarCount() int32 { return b.varCnt }

func (b *BDD) levelOf(v int32) int32 {
	if v < 0 || v >= b.varCnt {
		return -1
	}
	return b.var2level[v]
}

func (b *BDD) varOf(level int32) int32 {
	if level < 0 || level >= b.varCnt {
		return -1
	}
	return b.level2var[level]
}

// Accessors matching buddy bdd_var / bdd_low / bdd_high.

func (b *BDD) VarOf(f NodeID) int32 {
	if f.isTerm() {
		return -1
	}
	return b.level2var[b.nodes[f].level]
}

func (b *BDD) Low(f NodeID) NodeID {
	if f.isTerm() {
		return f
	}
	return NodeID(b.nodes[f].lo)
}

func (b *BDD) High(f NodeID) NodeID {
	if f.isTerm() {
		return f
	}
	return NodeID(b.nodes[f].hi)
}

// NodeCount returns the total number of allocated nodes.
func (b *BDD) NodeCount() int { return len(b.nodes) }

// SupportBDD returns the support of f as a BDD variable set.
// Matches buddy bdd_support (bddop.c:2053).
func (b *BDD) SupportBDD(f NodeID) NodeID {
	return b.MakeSet(b.Support(f))
}

// SatCountDouble returns the number of satisfying assignments as float64.
// Matches buddy bdd_satcount (bddop.c:2460).
func (b *BDD) SatCountDouble(f NodeID) float64 {
	return float64(b.SatisfyCount(f))
}

// AnodeCount counts distinct nodes across an array of BDDs.
// Shared nodes are counted only once. Matches buddy bdd_anodecount (bddop.c:2653). CC=4.
func (b *BDD) AnodeCount(roots []NodeID) int {
	seen := memory.MustPoolSlice[bool](b.pool, len(b.nodes))
	seen = seen[:len(b.nodes)]
	cou := 0
	for _, r := range roots {
		cou += b.countNodes(r, seen)
	}
	return cou
}

// --- Core BDD constructors ---

func (b *BDD) Var(v int32) NodeID {
	if v < 0 || v >= b.varCnt {
		return falseIdx
	}
	return b.unique(b.var2level[v], int32(falseIdx), int32(trueIdx))
}

func (b *BDD) Not(f NodeID) NodeID          { return b.ITE(f, falseIdx, trueIdx) }
func (b *BDD) And(f, g NodeID) NodeID        { return b.ITE(f, g, falseIdx) }
func (b *BDD) Or(f, g NodeID) NodeID         { return b.ITE(f, trueIdx, g) }
func (b *BDD) Implies(f, g NodeID) NodeID    { return b.ITE(f, g, trueIdx) }
func (b *BDD) Xor(f, g NodeID) NodeID        { return b.ITE(f, b.Not(g), g) }
func (b *BDD) Equiv(f, g NodeID) NodeID      { return b.ITE(f, g, b.Not(g)) }
func (b *BDD) Nand(f, g NodeID) NodeID       { return b.Not(b.And(f, g)) }
func (b *BDD) Nor(f, g NodeID) NodeID        { return b.Not(b.Or(f, g)) }

// --- Restrict, Exists, ForAll, Compose ---

func (b *BDD) Restrict(f NodeID, v int32, value bool) NodeID {
	vl := b.levelOf(v)
	if f.isTerm() {
		return f
	}
	n := b.nodes[f]
	if n.level > vl {
		return f
	}
	if n.level == vl {
		if value {
			return NodeID(n.hi)
		}
		return NodeID(n.lo)
	}
	lo := b.Restrict(NodeID(n.lo), v, value)
	hi := b.Restrict(NodeID(n.hi), v, value)
	return b.unique(n.level, int32(lo), int32(hi))
}

func (b *BDD) Exists(f NodeID, v int32) NodeID {
	return b.Or(b.Restrict(f, v, false), b.Restrict(f, v, true))
}

func (b *BDD) ExistsAll(f NodeID, vars []int32) NodeID {
	r := f
	for _, v := range vars {
		r = b.Exists(r, v)
	}
	return r
}

func (b *BDD) ForAll(f NodeID, v int32) NodeID {
	return b.And(b.Restrict(f, v, false), b.Restrict(f, v, true))
}

func (b *BDD) ForAllVars(f NodeID, vars []int32) NodeID {
	r := f
	for _, v := range vars {
		r = b.ForAll(r, v)
	}
	return r
}

func (b *BDD) Compose(f NodeID, v int32, g NodeID) NodeID {
	vl := b.levelOf(v)
	if f.isTerm() {
		return f
	}
	n := b.nodes[f]
	if n.level > vl {
		return f
	}
	if n.level == vl {
		return b.ITE(g, NodeID(n.hi), NodeID(n.lo))
	}
	lo := b.Compose(NodeID(n.lo), v, g)
	hi := b.Compose(NodeID(n.hi), v, g)
	return b.unique(n.level, int32(lo), int32(hi))
}

// --- Support, Satisfy ---

func (b *BDD) Support(f NodeID) []int32 {
	seen := memory.MustPoolSlice[bool](b.pool, int(b.varCnt))
	seen = seen[:b.varCnt]
	b.supportWalk(f, seen)
	var result []int32
	for i, s := range seen {
		if s {
			result = append(result, b.level2var[int32(i)])
		}
	}
	return result
}

func (b *BDD) supportWalk(f NodeID, seen []bool) {
	if f.isTerm() {
		return
	}
	n := b.nodes[f]
	seen[n.level] = true
	b.supportWalk(NodeID(n.lo), seen)
	b.supportWalk(NodeID(n.hi), seen)
}

func (b *BDD) SatisfyOne(f NodeID) []bool {
	if f == falseIdx {
		return nil
	}
	assign := memory.MustPoolSlice[bool](b.pool, int(b.varCnt))
	assign = assign[:b.varCnt]
	b.satWalk(f, assign)
	return assign
}

func (b *BDD) satWalk(f NodeID, assign []bool) {
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

func (b *BDD) SatisfyCount(f NodeID) uint64 {
	counts := memory.MustPoolSlice[uint64](b.pool, len(b.nodes))
	counts = counts[:len(b.nodes)]
	for i := range counts {
		counts[i] = ^uint64(0)
	}
	return b.satCount(f, counts)
}

func (b *BDD) satCount(f NodeID, counts []uint64) uint64 {
	if f == falseIdx {
		return 0
	}
	if f == trueIdx {
		return 1 << uint(b.varCnt)
	}
	if counts[f] != ^uint64(0) {
		return counts[f]
	}
	n := b.nodes[f]
	lo := b.satCount(NodeID(n.lo), counts)
	hi := b.satCount(NodeID(n.hi), counts)
	loShift := lo >> 1
	if n.lo >= 2 && b.nodes[n.lo].level == n.level+1 {
		loShift = lo
	}
	hiShift := hi >> 1
	if n.hi >= 2 && b.nodes[n.hi].level == n.level+1 {
		hiShift = hi
	}
	counts[f] = loShift + hiShift
	return counts[f]
}

// --- ITE: universal if-then-else ---

func (b *BDD) ITE(f, g, h NodeID) NodeID {
	if f == trueIdx {
		return g
	}
	if f == falseIdx {
		return h
	}
	if g == h {
		return g
	}
	if g == trueIdx && h == falseIdx {
		return f
	}
	if r, ok := b.cache.get(int32(f), int32(g), int32(h)); ok {
		return NodeID(r)
	}

	v := b.topVar(f, g, h)
	lo := b.ITE(b.cofactor(f, v, false), b.cofactor(g, v, false), b.cofactor(h, v, false))
	hi := b.ITE(b.cofactor(f, v, true), b.cofactor(g, v, true), b.cofactor(h, v, true))

	if lo == hi {
		b.cache.put(int32(f), int32(g), int32(h), int32(lo))
		return lo
	}
	r := b.unique(v, int32(lo), int32(hi))
	b.cache.put(int32(f), int32(g), int32(h), int32(r))
	return r
}

func (b *BDD) topVar(f, g, h NodeID) int32 {
	v := b.varCnt
	if !f.isTerm() && b.nodes[f].level < v {
		v = b.nodes[f].level
	}
	if !g.isTerm() && b.nodes[g].level < v {
		v = b.nodes[g].level
	}
	if !h.isTerm() && b.nodes[h].level < v {
		v = b.nodes[h].level
	}
	return v
}

func (b *BDD) cofactor(f NodeID, v int32, value bool) NodeID {
	if f.isTerm() {
		return f
	}
	nf := b.nodes[f]
	if nf.level > v {
		return f
	}
	if nf.level == v {
		if value {
			return NodeID(nf.hi)
		}
		return NodeID(nf.lo)
	}
	return f
}

func (b *BDD) unique(v, lo, hi int32) NodeID {
	if lo == hi {
		return NodeID(lo)
	}
	if idx, ok := b.uniq.get(v, lo, hi); ok {
		return NodeID(idx)
	}
	idx := int32(len(b.nodes))
	b.nodes = append(b.nodes, bddNode{level: v, lo: lo, hi: hi})
	b.uniq.put(v, lo, hi, idx)
	return NodeID(idx)
}

// --- Unique table ---

type uniqTable struct {
	buckets []uniqEntry
	mask    uint32
	size    int
}

type uniqEntry struct {
	v  int32
	lo int32
	hi int32
	id int32
	ok bool
}

func newUniqTable(pool *memory.Pool) *uniqTable {
	cap := 32768
	buckets := memory.MustPoolSlice[uniqEntry](pool, cap)
	buckets = buckets[:cap]
	return &uniqTable{buckets: buckets, mask: uint32(cap - 1)}
}

func (u *uniqTable) get(v, lo, hi int32) (int32, bool) {
	h := uniqHash(v, lo, hi)
	idx := h & u.mask
	for {
		e := &u.buckets[idx]
		if !e.ok {
			return 0, false
		}
		if e.v == v && e.lo == lo && e.hi == hi {
			return e.id, true
		}
		idx = (idx + 1) & u.mask
	}
}

func (u *uniqTable) put(v, lo, hi, id int32) {
	if u.size >= len(u.buckets)/2 {
		return
	}
	h := uniqHash(v, lo, hi)
	idx := h & u.mask
	for u.buckets[idx].ok {
		idx = (idx + 1) & u.mask
	}
	u.buckets[idx] = uniqEntry{v: v, lo: lo, hi: hi, id: id, ok: true}
	u.size++
}

func uniqHash(v, lo, hi int32) uint32 {
	h := uint32(v) * 0x9e3779b9
	h ^= uint32(lo) * 0x85ebca6b
	h ^= uint32(hi) * 0xc2b2ae35
	h ^= bits.RotateLeft32(uint32(v)^uint32(lo), 13)
	return h
}

// --- Operation cache ---

type opCache struct {
	buckets []cacheEntry
	mask    uint32
	size    int
}

type cacheEntry struct {
	f  int32
	g  int32
	h  int32
	r  int32
	ok bool
}

func newOpCache(pool *memory.Pool) *opCache {
	cap := 65536
	buckets := memory.MustPoolSlice[cacheEntry](pool, cap)
	buckets = buckets[:cap]
	return &opCache{buckets: buckets, mask: uint32(cap - 1)}
}

func (c *opCache) get(f, g, h int32) (int32, bool) {
	hash := cacheHash(f, g, h)
	idx := hash & c.mask
	for {
		e := &c.buckets[idx]
		if !e.ok {
			return 0, false
		}
		if e.f == f && e.g == g && e.h == h {
			return e.r, true
		}
		idx = (idx + 1) & c.mask
	}
}

func (c *opCache) put(f, g, h, r int32) {
	if c.size >= len(c.buckets)/2 {
		return
	}
	hash := cacheHash(f, g, h)
	idx := hash & c.mask
	for c.buckets[idx].ok {
		idx = (idx + 1) & c.mask
	}
	c.buckets[idx] = cacheEntry{f: f, g: g, h: h, r: r, ok: true}
	c.size++
}

func cacheHash(f, g, h int32) uint32 {
	hv := uint32(f) * 0x9e3779b9
	hv ^= uint32(g) * 0x85ebca6b
	hv ^= uint32(h) * 0xc2b2ae35
	hv ^= bits.RotateLeft32(uint32(f)^uint32(g)^uint32(h), 17)
	return hv
}
