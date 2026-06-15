// Package gobdd implements Ordered Binary Decision Diagrams (OBDDs)
// with off-heap memory via github.com/xDarkicex/memory.
//
// BDDs provide a canonical representation of Boolean functions. Two functions
// are equivalent iff their BDDs are identical — O(1) comparison after construction.
//
// Core operations (And, Or, Not) are built on the universal ITE (if-then-else)
// operator with memoization for polynomial-time complexity.
package gobdd

import (
	"math/bits"

	"github.com/xDarkicex/memory"
)

// BDD is a manager for Binary Decision Diagrams.
// All state lives in off-heap Pool-allocated slices — zero GC pressure.
type BDD struct {
	nodes   []bddNode  // Pool-backed node table
	varCnt  int32      // number of variables registered
	pool    *memory.Pool
	uniq    *uniqTable // unique table: (var,lo,hi) → node index
	cache   *opCache   // operation cache for ITE memoization
}

type bddNode struct {
	vari int32 // variable index (0..n-1), -1 for terminals
	lo   int32 // child index when var=0
	hi   int32 // child index when var=1
}

const (
	falseIdx = 0
	trueIdx  = 1
)

// New creates a BDD manager with the given number of variables.
func New(numVars int, pool *memory.Pool) *BDD {
	cap := numVars*256 + 16
	nodes := memory.MustPoolSlice[bddNode](pool, cap)
	nodes = nodes[:2]
	nodes[falseIdx] = bddNode{vari: -1, lo: -1, hi: -1}
	nodes[trueIdx] = bddNode{vari: -1, lo: -1, hi: -1}
	return &BDD{
		nodes:  nodes,
		varCnt: int32(numVars),
		pool:   pool,
		uniq:   newUniqTable(pool),
		cache:  newOpCache(pool),
	}
}

// VarCount returns the number of variables.
func (b *BDD) VarCount() int32 { return b.varCnt }

// Var returns the BDD for variable v (0-indexed).
func (b *BDD) Var(v int32) int32 {
	if v < 0 || v >= b.varCnt {
		return falseIdx
	}
	return b.unique(v, falseIdx, trueIdx)
}

// Not returns the negation of f.
func (b *BDD) Not(f int32) int32 { return b.ITE(f, falseIdx, trueIdx) }

// And returns f ∧ g.
func (b *BDD) And(f, g int32) int32 { return b.ITE(f, g, falseIdx) }

// Or returns f ∨ g.
func (b *BDD) Or(f, g int32) int32 { return b.ITE(f, trueIdx, g) }

// Implies returns f → g.
func (b *BDD) Implies(f, g int32) int32 { return b.ITE(f, g, trueIdx) }

// Xor returns f ⊕ g.
func (b *BDD) Xor(f, g int32) int32 { return b.ITE(f, b.Not(g), g) }

// Equiv returns f ↔ g.
func (b *BDD) Equiv(f, g int32) int32 { return b.ITE(f, g, b.Not(g)) }

// Nand returns ¬(f ∧ g).
func (b *BDD) Nand(f, g int32) int32 { return b.Not(b.And(f, g)) }

// Nor returns ¬(f ∨ g).
func (b *BDD) Nor(f, g int32) int32 { return b.Not(b.Or(f, g)) }

// Restrict returns f with variable v set to value.
// f[v := value] — cofactor operation. CC=3.
func (b *BDD) Restrict(f int32, v int32, value bool) int32 {
	if f < 2 {
		return f
	}
	n := b.nodes[f]
	if n.vari > v {
		return f
	}
	if n.vari == v {
		if value {
			return n.hi
		}
		return n.lo
	}
	lo := b.Restrict(n.lo, v, value)
	hi := b.Restrict(n.hi, v, value)
	return b.unique(n.vari, lo, hi)
}

// Exists returns ∃v. f — existential quantification.
// ∃v. f = f[v:=0] ∨ f[v:=1]. CC=2.
func (b *BDD) Exists(f int32, v int32) int32 {
	return b.Or(b.Restrict(f, v, false), b.Restrict(f, v, true))
}

// ExistsAll returns ∃vars. f — existentially quantify multiple variables.
// CC=2.
func (b *BDD) ExistsAll(f int32, vars []int32) int32 {
	r := f
	for _, v := range vars {
		r = b.Exists(r, v)
	}
	return r
}

// ForAll returns ∀v. f — universal quantification.
// ∀v. f = f[v:=0] ∧ f[v:=1]. CC=2.
func (b *BDD) ForAll(f int32, v int32) int32 {
	return b.And(b.Restrict(f, v, false), b.Restrict(f, v, true))
}

// ForAllVars returns ∀vars. f.
func (b *BDD) ForAllVars(f int32, vars []int32) int32 {
	r := f
	for _, v := range vars {
		r = b.ForAll(r, v)
	}
	return r
}

// Compose returns f[v := g] — substitute variable v with BDD g.
// f composed with g for variable v. CC=4.
func (b *BDD) Compose(f int32, v int32, g int32) int32 {
	if f < 2 {
		return f
	}
	n := b.nodes[f]
	if n.vari > v {
		return f
	}
	if n.vari == v {
		return b.ITE(g, n.hi, n.lo)
	}
	lo := b.Compose(n.lo, v, g)
	hi := b.Compose(n.hi, v, g)
	return b.unique(n.vari, lo, hi)
}

// Support returns the set of variables that f depends on.
// CC=3.
func (b *BDD) Support(f int32) []int32 {
	seen := make([]bool, b.varCnt)
	b.supportWalk(f, seen)
	var result []int32
	for i, s := range seen {
		if s {
			result = append(result, int32(i))
		}
	}
	return result
}

func (b *BDD) supportWalk(f int32, seen []bool) {
	if f < 2 {
		return
	}
	n := b.nodes[f]
	seen[n.vari] = true
	b.supportWalk(n.lo, seen)
	b.supportWalk(n.hi, seen)
}

// SatisfyOne returns one satisfying assignment, or nil if f is False.
func (b *BDD) SatisfyOne(f int32) []bool {
	if f == falseIdx {
		return nil
	}
	assign := make([]bool, b.varCnt)
	b.satWalk(f, assign)
	return assign
}

func (b *BDD) satWalk(f int32, assign []bool) {
	for f >= 2 {
		n := b.nodes[f]
		if n.lo != falseIdx {
			assign[n.vari] = false
			f = n.lo
		} else {
			assign[n.vari] = true
			f = n.hi
		}
	}
}

// SatisfyCount returns the number of satisfying assignments.
// Uses dynamic programming on the BDD structure. CC=5.
func (b *BDD) SatisfyCount(f int32) uint64 {
	counts := memory.MustPoolSlice[uint64](b.pool, len(b.nodes))
	counts = counts[:len(b.nodes)]
	for i := range counts {
		counts[i] = ^uint64(0) // sentinel for "not computed"
	}
	return b.satCount(f, counts)
}

func (b *BDD) satCount(f int32, counts []uint64) uint64 {
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
	lo := b.satCount(n.lo, counts)
	hi := b.satCount(n.hi, counts)
	// Each path skips the vars between this level and the child
	loShift := lo >> 1
	hiShift := hi >> 1
	if n.lo < 2 || b.nodes[n.lo].vari != n.vari+1 {
		loShift = lo >> 1
	} else {
		loShift = lo
	}
	if n.hi < 2 || b.nodes[n.hi].vari != n.vari+1 {
		hiShift = hi >> 1
	} else {
		hiShift = hi
	}
	counts[f] = loShift + hiShift
	return counts[f]
}

// NodeCount returns the total number of nodes.
func (b *BDD) NodeCount() int { return len(b.nodes) }

// ITE is the universal if-then-else: if f then g else h.
// O(|f|·|g|·|h|) worst-case, O(|f|+|g|+|h|) with memoization. CC=7.
func (b *BDD) ITE(f, g, h int32) int32 {
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

	if r, ok := b.cache.get(f, g, h); ok {
		return r
	}

	v := b.topVar(f, g, h)
	lo := b.ITE(b.cofactor(f, v, false), b.cofactor(g, v, false), b.cofactor(h, v, false))
	hi := b.ITE(b.cofactor(f, v, true), b.cofactor(g, v, true), b.cofactor(h, v, true))

	if lo == hi {
		b.cache.put(f, g, h, lo)
		return lo
	}

	r := b.unique(v, lo, hi)
	b.cache.put(f, g, h, r)
	return r
}

func (b *BDD) topVar(f, g, h int32) int32 {
	v := b.varCnt
	if f >= 2 && b.nodes[f].vari < v {
		v = b.nodes[f].vari
	}
	if g >= 2 && b.nodes[g].vari < v {
		v = b.nodes[g].vari
	}
	if h >= 2 && b.nodes[h].vari < v {
		v = b.nodes[h].vari
	}
	return v
}

func (b *BDD) cofactor(f int32, v int32, value bool) int32 {
	if f < 2 {
		return f
	}
	nf := b.nodes[f]
	if nf.vari > v {
		return f
	}
	if nf.vari == v {
		if value {
			return nf.hi
		}
		return nf.lo
	}
	return f
}

func (b *BDD) unique(v, lo, hi int32) int32 {
	if lo == hi {
		return lo
	}
	if idx, ok := b.uniq.get(v, lo, hi); ok {
		return idx
	}
	idx := int32(len(b.nodes))
	b.nodes = append(b.nodes, bddNode{vari: v, lo: lo, hi: hi})
	b.uniq.put(v, lo, hi, idx)
	return idx
}

// --- Unique table (Pool-backed open addressing) ---

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

// --- Operation cache (ITE memoization) ---

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
