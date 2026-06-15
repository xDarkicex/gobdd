// Package gobdd implements Reduced Ordered Binary Decision Diagrams (ROBDDs)
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
// All state lives in off-heap Pool-allocated slices.
type BDD struct {
	nodes   []bddNode // Pool-backed node table
	varCnt  int32     // number of variables registered
	pool    *memory.Pool
	uniq    *uniqTable // unique table: (var,lo,hi) → node index
	cache   *opCache   // operation cache: (op,a,b) → result index
}

type bddNode struct {
	vari int32 // variable index (0..n-1), -1 for terminals
	lo   int32 // child index when var=0
	hi   int32 // child index when var=1
}

const (
	falseIdx = 0 // terminal False node index
	trueIdx  = 1 // terminal True node index
)

// New creates a BDD manager with the given number of variables.
// All internal tables are Pool-backed — zero GC pressure.
func New(numVars int, pool *memory.Pool) *BDD {
	cap := numVars*256 + 16
	nodes := memory.MustPoolSlice[bddNode](pool, cap)
	nodes = nodes[:2] // reserve 0=False, 1=True
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

// ITE is the universal if-then-else: if f then g else h.
// This is the core BDD operation — all Boolean ops reduce to ITE.
// O(|f|·|g|·|h|) worst-case, O(|f|+|g|+|h|) with memoization. CC=7.
func (b *BDD) ITE(f, g, h int32) int32 {
	// Terminal cases
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

	// Check cache
	if r, ok := b.cache.get(0, f, g, h); ok {
		return r
	}

	// Pick top variable
	v := b.topVar(f, g, h)

	// Recurse: ITE(f, g, h) = (v ? ITE(f_hi, g_hi, h_hi) : ITE(f_lo, g_lo, h_lo))
	lo := b.ITE(b.cofactor(f, v, false), b.cofactor(g, v, false), b.cofactor(h, v, false))
	hi := b.ITE(b.cofactor(f, v, true), b.cofactor(g, v, true), b.cofactor(h, v, true))

	if lo == hi {
		b.cache.put(0, f, g, h, lo)
		return lo
	}

	r := b.unique(v, lo, hi)
	b.cache.put(0, f, g, h, r)
	return r
}

// topVar returns the highest (smallest index) variable among non-terminal nodes.
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

// cofactor returns f restricted to var=v with the given value.
func (b *BDD) cofactor(f int32, v int32, value bool) int32 {
	if f < 2 {
		return f
	}
	nf := b.nodes[f]
	if nf.vari > v {
		return f // variable doesn't appear in f
	}
	if nf.vari == v {
		if value {
			return nf.hi
		}
		return nf.lo
	}
	return f
}

// unique returns the canonical node for (var, lo, hi), reusing existing or creating new.
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

// SatisfyOne returns one satisfying assignment for f, or nil if f is False.
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

// NodeCount returns the number of nodes in the BDD (including terminals).
func (b *BDD) NodeCount() int { return len(b.nodes) }

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
	cap := 16384
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
		return // grow not implemented for v1 — rely on initial capacity
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

func (c *opCache) get(_ int32, f, g, h int32) (int32, bool) {
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

func (c *opCache) put(_ int32, f, g, h, r int32) {
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
