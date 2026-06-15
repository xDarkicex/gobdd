package gobdd

import "github.com/xDarkicex/memory"

// ref tracking: external reference counting for automatic handle remapping
// during variable reordering.

// AddRef increments the external reference count on f.
// Nodes with positive reference counts are automatically remapped
// during SwapVar/Sift. Matches buddy bdd_addref (kernel.c:1115).
func (b *BDD) AddRef(f NodeID) NodeID {
	if f.isTerm() {
		return f
	}
	b.ensureRefCount()
	b.refCount[f]++
	return f
}

// DelRef decrements the external reference count on f.
// Matches buddy bdd_delref (kernel.c:1141).
func (b *BDD) DelRef(f NodeID) {
	if f.isTerm() {
		return
	}
	b.ensureRefCount()
	if b.refCount[f] > 0 {
		b.refCount[f]--
	}
}

// RefCount returns the external reference count for f.
func (b *BDD) RefCount(f NodeID) int32 {
	if f.isTerm() || b.refCount == nil {
		return 0
	}
	return b.refCount[f]
}

// ensureRefCount lazy-initializes the reference count slice.
func (b *BDD) ensureRefCount() {
	if b.refCount != nil {
		return
	}
	b.refCount = b.newRefCount()
}

// applyRemap updates all externally referenced nodes using a remap function.
// Called internally by swapLevels to keep user handles valid.
func (b *BDD) applyRemap(remap func(int32) int32) {
	if b.refCount == nil {
		return
	}
	// Collect old indices that have refs and their counts.
	type entry struct {
		oldIdx int32
		count  int32
	}
	var entries []entry
	for oldIdx, c := range b.refCount {
		if c > 0 {
			entries = append(entries, entry{int32(oldIdx), c})
		}
	}
	// Rebuild refCount with new indices.
	b.refCount = b.newRefCount()
	for _, e := range entries {
		newIdx := remap(e.oldIdx)
		b.refCount[newIdx] = e.count
	}
}

// resetRefCounts clears all reference counts.
// Used after GC or when the node table is rebuilt.
func (b *BDD) newRefCount() []int32 {
	rc := memory.MustPoolSlice[int32](b.pool, len(b.nodes))
	rc = rc[:len(b.nodes)]
	return rc
}

func (b *BDD) resetRefCounts() {
	if b.refCount == nil {
		return
	}
	for i := range b.refCount {
		b.refCount[i] = 0
	}
}
