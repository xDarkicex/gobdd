# GOBDD Architecture

## Overview

GOBDD implements Ordered Binary Decision Diagrams in pure Go with zero heap allocations. The architecture follows Buddy's design while adapting to Go's type system and `xDarkicex/memory`'s off-heap allocator.

## Package Structure

```
gobdd/
├── types.go        Distinct types (NodeID, OpCode) and constants
├── bdd.go          BDD manager struct, unique table, ITE engine, op-cache
├── ops.go          Boolean operators, constraint, simplification, cube building
├── quant.go        Variable set quantification engine (exists/forall/unique)
├── sat.go          Satisfiability: model extraction, assignment enumeration
├── replace.go      Variable substitution pairs and composition
├── reorder.go      Sifting reordering with atomic level swap
├── serialize.go    Buddy-compatible save/load
├── refcount.go     External reference counting for handle remapping
├── mgmt.go         Lifecycle (Done, SetVarNum, ExtVarNum, SetCacheRatio)
├── io.go           Debug output (DOT, truth tables), variable sets
├── convert.go      Formula string rendering (DNF, CNF, ITE form)
```

## Core Data Structures

### BDD Manager

```go
type BDD struct {
    nodes     []bddNode   // contiguous node table (Pool-backed)
    varCnt    int32       // number of variables
    var2level []int32     // variable → level position
    level2var []int32     // level position → variable
    pool      *memory.Pool
    uniq      *uniqTable  // (level, lo, hi) → node index
    cache     *opCache    // ITE memoization cache
    refCount  []int32     // external reference count (lazy init)
}
```

Only the `BDD` struct itself lives on the Go heap (~200 bytes). All slices are Pool-backed, invisible to the GC.

### bddNode

```go
type bddNode struct {
    level int32  // position in variable ordering (-1 for terminals)
    lo    int32  // child index when variable = 0
    hi    int32  // child index when variable = 1
}
```

Terminals: `nodes[0]` = False (level=-1, lo=-1, hi=-1), `nodes[1]` = True (level=-1, lo=-1, hi=-1). Non-terminal nodes start at index 2.

### Distinct Types

```go
type NodeID int32   // BDD handle
type OpCode int32   // apply operator code
```

These prevent argument transposition at compile time — passing an operator code where a NodeID is expected produces a type error. The `NodeID` type also provides `isTerm()` for checking terminal status.

## Unique Table

Open-addressing hash table keyed by `(level, lo, hi)`:

- **Capacity**: 32,768 buckets (power of 2)
- **Hash function**: Multiplicative hash with bit rotation
- **Probe strategy**: Linear probing with power-of-two mask
- **Load factor**: 50% (silent drop on overflow; in practice never reached)
- **Memory**: Pool-backed `[]uniqEntry`

The unique table guarantees canonicity: every `(level, lo, hi)` triple maps to exactly one node index. When `lo == hi`, no entry is created — the redundant node is eliminated.

## ITE Engine

All Boolean operations (And, Or, Not, Xor, etc.) reduce to ITE:

```
ITE(f, g, h) = (f ∧ g) ∨ (¬f ∧ h)
```

The ITE function implements the recursive algorithm with:

1. **Terminal shortcuts**: `ITE(1,g,h) = g`, `ITE(0,g,h) = h`, `ITE(f,g,g) = g`, `ITE(f,1,0) = f`
2. **Memoization**: Operator cache avoids recomputing identical triples
3. **Redundancy elimination**: If `lo == hi` after recursion, return `lo` directly

**Complexity**: O(|f| + |g| + |h|) with memoization for typical inputs.

## Operator Cache

Separate from the unique table. Keyed by `(f, g, h)` triples for ITE memoization:

- **Capacity**: 65,536 buckets (power of 2)
- **Eviction**: Silent drop at 50% load factor
- **Scope**: Single ITE operation (not shared across operations)
- **Growth**: `SetCacheRatio(r)` resizes to `len(nodes) * r`

## Level Indirection

Variables and levels are decoupled:

```
var2level[v] → the level (position) where variable v is evaluated
level2var[l] → the variable evaluated at level l
```

In the default identity mapping: `var2level[i] = i`, `level2var[i] = i`.

When two adjacent levels `l` and `l+1` are swapped:
1. `level2var[l]` and `level2var[l+1]` exchange values
2. `var2level` is updated to match
3. The entire node table is rebuilt bottom-up: nodes at the same level keep their children (remapped to new indices), but the variable they branch on changes

This rebuild invalidates existing NodeID handles — the returned remap function translates old handles to new ones. Reference-counted handles (via `AddRef`) are automatically remapped internally.

## Variable Set Quantification

Quantifiers operate on BDD variable sets — chains of nodes along hi edges:

```
VarSet({0, 2, 3}) → node(v=0, lo=0, hi=node(v=2, lo=0, hi=node(v=3, lo=0, hi=1)))
```

The `varsetIndex` helper marks which levels are in the set using a Pool-backed `[]int32` array. The `quantRec` function walks the BDD bottom-up, applying the combine operator (Or/And/Xor) at levels in the set.

**RestrictBDD** uses a signed variant where positive markers indicate restriction to true and negative markers restriction to false.

## Sifting Reorder

### swapLevels(l1, l2)

1. Swap `level2var[l1]` and `level2var[l2]`, update `var2level`
2. Clear the unique table (all entries invalidated by level change)
3. Save old node table, allocate fresh one (terminals 0 and 1 preserved)
4. Allocate remap table: `remap[oldIdx] = newIdx`
5. Rebuild bottom-up (highest level → lowest level), so children are remapped before parents
6. Apply remap to reference-counted external roots
7. Return remap closure for caller's handles

### Sift()

For each variable:
1. Move to top (repeated swapLevels upward)
2. Sift down through all positions, tracking node count at each
3. Move back to best position

Complexity: O(n²) worst case, but each `swapLevels` is O(nodes at two levels), typically small.

## Serialization Format

Buddy-compatible text format:

```
N V              # node count, variable count (0 0 for terminals)
v0 v1 ... vV-1   # var2level for each variable (space separated)
id var lo hi     # repeated N times, children before parents
```

**Save**: DFS post-order traversal with visited-tracking `[]bool` (Pool-backed). Count reachable nodes first, then write header, ordering, and node entries.

**Load**: Read header, ordering, then nodes. Each node is rebuilt via `ITE(Var(var), hi, lo)` using a Pool-backed open-addressing hash table to remap old file IDs to new NodeIDs.

## Memory Allocation Strategy

| Allocator | User | Allocation Pattern |
|-----------|------|-------------------|
| `memory.Pool` | Node table | Grow-only, initial capacity = `numVars * 256 + 16` |
| `memory.Pool` | Unique table buckets | Fixed 32,768 entries |
| `memory.Pool` | Op-cache buckets | Fixed 65,536 entries |
| `memory.Pool` | var2level / level2var | Fixed to varCnt |
| `memory.Pool` | RefCount slice | Lazy init, sized to len(nodes) |
| `memory.Pool` | Temp slices | Visited tracking (countNodes, dotWalk, formulaRec, varProfileRec, Support, SatisfyOne, AllSat, FullSatOne), save rec, serialization remap |
| `memory.Pool` | loadHash | Serialization load remap (power-of-2 open addressing) |

All slices MUST be resized after `MustPoolSlice`: `slice = slice[:n]` — Pool returns capacity-only allocation with length 0.

## Concurrency

GOBDD is not currently safe for concurrent use. The BDD manager maintains mutable internal state (node table, unique table, caches, pools). Concurrent operations would require external synchronization. The `xDarkicex/memory` package provides `ShardedFreeList` for concurrent allocation patterns, reserved for future concurrent tableau expansion in the modal package.
