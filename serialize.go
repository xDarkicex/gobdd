package gobdd

import (
	"fmt"
	"io"

	"github.com/xDarkicex/memory"
)

// Save writes a BDD to w in buddy-compatible format.
// Terminals are written as "0 0 <value>\n".
// Non-terminals write the node count, variable ordering, then each node
// in DFS post-order (children before parent).
// CC=5.
func (b *BDD) Save(w io.Writer, f NodeID) error {
	if f.isTerm() {
		_, err := fmt.Fprintf(w, "0 0 %d\n", int32(f))
		return err
	}

	// Count reachable nodes.
	seen := memory.MustPoolSlice[bool](b.pool, len(b.nodes))
	seen = seen[:len(b.nodes)]
	n := b.countNodes(f, seen)
	if n == 0 {
		return fmt.Errorf("no nodes reachable")
	}

	// Write header: nodeCount varCount
	if _, err := fmt.Fprintf(w, "%d %d\n", n, b.varCnt); err != nil {
		return err
	}

	// Write variable ordering (var2level for each variable).
	for i := int32(0); i < b.varCnt; i++ {
		if _, err := fmt.Fprintf(w, "%d", b.var2level[i]); err != nil {
			return err
		}
		if i+1 < b.varCnt {
			if _, err := fmt.Fprint(w, " "); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}

	// Clear marks for the actual save walk.
	for i := range seen {
		seen[i] = false
	}
	return b.saveRec(w, f, seen)
}

// countNodes counts reachable non-terminal nodes. CC=3.
func (b *BDD) countNodes(f NodeID, seen []bool) int {
	if f.isTerm() || seen[f] {
		return 0
	}
	seen[f] = true
	return 1 + b.countNodes(NodeID(b.nodes[f].lo), seen) + b.countNodes(NodeID(b.nodes[f].hi), seen)
}

// saveRec writes nodes in post-order. CC=3.
func (b *BDD) saveRec(w io.Writer, f NodeID, seen []bool) error {
	if f.isTerm() || seen[f] {
		return nil
	}
	seen[f] = true
	n := b.nodes[f]
	if err := b.saveRec(w, NodeID(n.lo), seen); err != nil {
		return err
	}
	if err := b.saveRec(w, NodeID(n.hi), seen); err != nil {
		return err
	}
	vari := b.level2var[n.level]
	_, err := fmt.Fprintf(w, "%d %d %d %d\n", f, vari, n.lo, n.hi)
	return err
}

// Load reads a BDD from r in buddy-compatible format.
// Returns the root NodeID. The BDD's variable ordering is updated.
// CC=8.
func (b *BDD) Load(r io.Reader) (NodeID, error) {
	var nodeCount, varCount int32
	if _, err := fmt.Fscan(r, &nodeCount, &varCount); err != nil {
		return falseIdx, fmt.Errorf("header: %w", err)
	}

	// Terminal case.
	if nodeCount == 0 && varCount == 0 {
		var val int32
		if _, err := fmt.Fscan(r, &val); err != nil {
			return falseIdx, fmt.Errorf("terminal: %w", err)
		}
		return NodeID(val), nil
	}

	// Read variable ordering.
	if varCount > b.varCnt {
		return falseIdx, fmt.Errorf("file has %d vars, BDD has %d", varCount, b.varCnt)
	}
	savedOrder := memory.MustPoolSlice[int32](b.pool, int(varCount))
	savedOrder = savedOrder[:varCount]
	for i := int32(0); i < varCount; i++ {
		if _, err := fmt.Fscan(r, &savedOrder[i]); err != nil {
			return falseIdx, fmt.Errorf("order[%d]: %w", i, err)
		}
	}
	b.applyOrder(savedOrder, varCount)
	return b.loadNodes(r, nodeCount)
}

// applyOrder updates var2level/level2var from a saved ordering. CC=2.
func (b *BDD) applyOrder(order []int32, count int32) {
	for v := int32(0); v < count; v++ {
		b.var2level[v] = order[v]
		b.level2var[order[v]] = v
	}
}

// loadNodes reads nodeCount nodes and rebuilds the BDD. CC=5.
func (b *BDD) loadNodes(r io.Reader, nodeCount int32) (NodeID, error) {
	lh := newLoadHash(b.pool, int(nodeCount))
	var root NodeID

	for i := int32(0); i < nodeCount; i++ {
		var oldID, vari, lo, hi int32
		if _, err := fmt.Fscan(r, &oldID, &vari, &lo, &hi); err != nil {
			return falseIdx, fmt.Errorf("node %d: %w", i, err)
		}
		if lo >= 2 {
			newLo, ok := lh.get(lo)
			if !ok {
				return falseIdx, fmt.Errorf("lo=%d not found", lo)
			}
			lo = int32(newLo)
		}
		if hi >= 2 {
			newHi, ok := lh.get(hi)
			if !ok {
				return falseIdx, fmt.Errorf("hi=%d not found", hi)
			}
			hi = int32(newHi)
		}
		newNode := b.ITE(b.Var(vari), NodeID(hi), NodeID(lo))
		lh.add(oldID, newNode)
		root = newNode
	}
	return root, nil
}


type loadHash struct {
	table []lhEntry
	mask  int32
}

type lhEntry struct {
	key  int32
	data NodeID
}

func newLoadHash(pool *memory.Pool, size int) *loadHash {
	cap := 1
	for cap < size*2 {
		cap <<= 1
	}
	table := memory.MustPoolSlice[lhEntry](pool, cap)
	table = table[:cap]
	for i := range table {
		table[i].key = -1
	}
	return &loadHash{table: table, mask: int32(cap - 1)}
}

func (lh *loadHash) get(key int32) (NodeID, bool) {
	idx := key & lh.mask
	for {
		e := &lh.table[idx]
		if e.key == -1 {
			return 0, false
		}
		if e.key == key {
			return e.data, true
		}
		idx = (idx + 1) & lh.mask
	}
}

func (lh *loadHash) add(key int32, data NodeID) {
	idx := key & lh.mask
	for lh.table[idx].key != -1 {
		idx = (idx + 1) & lh.mask
	}
	lh.table[idx] = lhEntry{key: key, data: data}
}
