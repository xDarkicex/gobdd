package gobdd

import (
	"github.com/xDarkicex/memory"
)

// Done releases all resources held by the BDD.
// After Done, the BDD must not be used.
// Matches buddy bdd_done (bdd.h:231).
func (b *BDD) Done() {
	b.pool.Reset()
	b.nodes = nil
	b.var2level = nil
	b.level2var = nil
	b.uniq = nil
	b.cache = nil
	b.refCount = nil
}

// IsRunning reports whether the BDD is initialized and ready for use.
// Matches buddy bdd_isrunning (bdd.h:234).
func (b *BDD) IsRunning() bool {
	return b.nodes != nil
}

// SetVarNum changes the number of variables.
// If num > current, new variables are added with identity level mapping.
// If num < current, variables beyond num are removed.
// Matches buddy bdd_setvarnum (bdd.h:232). CC=7.
func (b *BDD) SetVarNum(num int32) {
	if num == b.varCnt {
		return
	}
	old := b.var2level
	b.var2level = memory.MustPoolSlice[int32](b.pool, int(num))
	b.var2level = b.var2level[:num]
	b.level2var = memory.MustPoolSlice[int32](b.pool, int(num))
	b.level2var = b.level2var[:num]

	n := int32(len(old))
	if num < n {
		n = num
	}
	for i := int32(0); i < n; i++ {
		b.var2level[i] = old[i]
		b.level2var[old[i]] = i
	}
	for i := n; i < num; i++ {
		b.var2level[i] = i
		b.level2var[i] = i
	}
	b.varCnt = num
}

// ExtVarNum adds n new variables and returns the index of the first new variable.
// Matches buddy bdd_extvarnum (bdd.h:233). CC=2.
func (b *BDD) ExtVarNum(n int32) int32 {
	old := b.varCnt
	b.SetVarNum(old + n)
	return old
}

// SetCacheRatio sets the operator cache size relative to the node table.
// The cache is resized to len(nodes) * ratio, clamped to [1024, 262144].
// Matches buddy bdd_setcacheratio (bddop.c:276). CC=4.
func (b *BDD) SetCacheRatio(ratio int) {
	if b.cache == nil {
		return
	}
	size := len(b.nodes) * ratio
	if size < 1024 {
		size = 1024
	}
	if size > 262144 {
		size = 262144
	}
	buckets := memory.MustPoolSlice[cacheEntry](b.pool, size)
	buckets = buckets[:size]
	mask := uint32(1)
	for mask < uint32(size) {
		mask <<= 1
	}
	// Use power-of-two size closest to requested size.
	pow2 := 1
	for pow2 < size {
		pow2 <<= 1
	}
	buckets = buckets[:pow2]
	b.cache.buckets = buckets
	b.cache.mask = uint32(pow2 - 1)
	b.cache.size = 0
}
