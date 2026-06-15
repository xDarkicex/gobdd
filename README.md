# GOBDD — Zero-Allocation Binary Decision Diagrams for Go

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](./LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/xDarkicex/gobdd?style=flat-square)](https://goreportcard.com/report/github.com/xDarkicex/gobdd)
[![GoDoc](https://img.shields.io/badge/godoc-reference-00ADD8?style=flat-square&logo=go)](https://pkg.go.dev/github.com/xDarkicex/gobdd)
[![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/xDarkicex/gobdd/coverage/coverage.json)](https://github.com/xDarkicex/gobdd/actions)
[![Tests](https://img.shields.io/github/actions/workflow/status/xDarkicex/gobdd/test.yml?branch=main&style=flat-square&label=tests)](https://github.com/xDarkicex/gobdd/actions)

A pure Go implementation of Ordered Binary Decision Diagrams (OBDDs) with **zero heap allocations**. Every backing array, node table, cache bucket, and temporary slice is allocated through a custom off-heap memory allocator — the Go GC never scans BDD data.

BDDs are a canonical representation of Boolean functions. Two functions are equivalent if and only if their BDDs are structurally identical, enabling O(1) equivalence checking after construction. They underpin symbolic model checking, formal verification, SAT solving, and the modal logic engine in [github.com/xDarkicex/logic](https://github.com/xDarkicex/logic).

---

## Table of Contents

- [Why GOBDD](#why-gobdd)
- [Zero Heap Allocation](#zero-heap-allocation)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [API Overview](#api-overview)
- [Buddy Parity](#buddy-parity)
- [Architecture](#architecture)
- [Testing \& Coverage](#testing--coverage)
- [License \& Attribution](#license--attribution)

---

## Why GOBDD

There are no pure-Go BDD libraries with zero heap allocation. Existing options either wrap C libraries through CGo (Buddy, CUDD) or use standard Go allocation patterns that place BDD data structures on the garbage-collected heap. For applications that construct millions of BDD nodes — model checking, bisimulation, symbolic reachability — GC pressure becomes a dominant cost, often exceeding 40% of total runtime.

GOBDD targets that gap:

| Property | GOBDD | CGo wrappers | Hypothetical heap-BDD |
|----------|-------|-------------|----------------------|
| Pure Go | :white_check_mark: | :x: | :white_check_mark: |
| Zero GC pressure | :white_check_mark: | :white_check_mark: | :x: |
| Race-clean | :white_check_mark: | varies | varies |
| Buddy API parity | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| ARM64 aligned | :white_check_mark: | n/a | :white_check_mark: |
| No CGo build flags | :white_check_mark: | :x: | :white_check_mark: |

---

## Zero Heap Allocation

Every allocation in GOBDD uses [github.com/xDarkicex/memory](https://github.com/xDarkicex/memory) — a custom off-heap allocator providing 16-byte aligned memory with Pool, Arena, and ShardedFreeList backends.

### Why 16-byte alignment?

ARM64 (Apple Silicon, AWS Graviton) requires 128-bit atomic operations to be 16-byte aligned. Unaligned atomics trap with `SIGBUS`. The `memory` allocator guarantees every allocation meets this constraint, making GOBDD safe on ARM64 without `GOARM=7` workarounds or alignment padding hacks.

### Allocation strategy

| Allocator | Use in GOBDD |
|-----------|-------------|
| `memory.Pool` | Node table, unique table buckets, op-cache entries, variable ordering arrays, reference counts, temporary visited-tracking slices, serialization remap tables |
| `memory.Arena` | _(reserved for modal integration — frame state, timeline entries)_ |
| `memory.ShardedFreeList` | _(reserved for modal integration — high-churn tableau nodes)_ |

No `make()`, no `new()`, no `&T{}` for any slice-backed or high-frequency struct. The Go GC traces only the `BDD` manager struct itself — ~200 bytes total.

---

## Installation

```bash
go get github.com/xDarkicex/gobdd@latest
```

Requires Go 1.25+ and a dependency on `github.com/xDarkicex/memory`.

---

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/xDarkicex/gobdd"
    "github.com/xDarkicex/memory"
)

func main() {
    pool, _ := memory.NewPool(memory.DefaultConfig())
    defer pool.Reset()

    b := gobdd.New(4, pool) // 4 variables: x0, x1, x2, x3

    x0 := b.Var(0)
    x1 := b.Var(1)
    f := b.And(x0, b.Not(x1)) // x0 ∧ ¬x1

    // Count satisfying assignments
    fmt.Println(b.SatisfyCount(f)) // 4 (x0=1, x1=0, x2=*, x3=*)

    // Find one assignment
    fmt.Println(b.SatisfyOne(f)) // [true false false false]

    // Quantify
    v0only := b.ForAll(f, 0) // ∀x0.(x0∧¬x1) = false
    fmt.Println(v0only == gobdd.False)
}
```

---

## API Overview

### Constructors & Operators

| Function | Description |
|----------|-------------|
| `New(n, pool)` | Create BDD manager with `n` variables |
| `Var(v)` | BDD for variable `v` |
| `Nithvar(v)` | BDD for `¬v` |
| `Not(f)`, `And(f,g)`, `Or(f,g)` | Boolean operators |
| `Xor(f,g)`, `Implies(f,g)`, `Equiv(f,g)` | Boolean operators |
| `Nand(f,g)`, `Nor(f,g)` | Boolean operators |
| `ITE(f,g,h)` | If-then-else: `f ? g : h` |
| `Apply(f,g,op)` | Generic binary apply with operator code |
| `Restrict(f,v,val)` | Cofactor: `f[v := val]` |
| `RestrictBDD(f,c)` | BDD-encoded restriction with polarity |
| `Constrain(f,c)` | Generalized cofactor |
| `Simplify(f,d)` | Simplify under don't-care set `d` |
| `Compose(f,v,g)` | Substitute: `f[v := g]` |
| `Replace(f,pair)` | Multi-variable substitution |
| `VecCompose(f,pair)` | Compose all variables in pair |

### Quantification

| Function | Description |
|----------|-------------|
| `Exists(f,v)` / `ForAll(f,v)` | Single-variable quantification |
| `ExistsAll(f,vars)` / `ForAllVars(f,vars)` | Multi-variable (slice) |
| `ExistSet(f,set)` / `ForAllSet(f,set)` | BDD variable set quantification |
| `Unique(f,set)` | Unique existential: `∃!vars. f` |
| `AppEx(f,g,op,vars)` / `AppAll(...)` | Apply + quantify |
| `AppExBDD(f,g,op,set)` / `AppAllBDD(...)` | BDD varset variants |
| `AppUni(l,r,varset,op)` | Apply + unique quantify (fused) |

### Satisfiability & Model Extraction

| Function | Description |
|----------|-------------|
| `SatisfyOne(f)` | One satisfying assignment (`[]bool`) |
| `FullSatOne(f)` | Assignment for all variables (`[]bool`) |
| `FullSatOneBDD(f)` | Full minterm as BDD |
| `SatOne(f)` | One minterm BDD |
| `SatOneSet(f,set,pol)` | Minterm respecting variable set |
| `AllSat(f,handler)` | Enumerate all satisfying assignments |
| `SatisfyCount(f)` | Count satisfying assignments (`uint64`) |
| `SatCountDouble(f)` | Count as `float64` (overflow-safe) |
| `SatCountLn(f)` | Natural log of count |
| `SatCountSet(f,vars)` / `SatCountLnSet(...)` | Count over variable set |

### Information & Debugging

| Function | Description |
|----------|-------------|
| `Support(f)` | Variable support (`[]int32`) |
| `SupportBDD(f)` | Variable support as BDD set |
| `NodeCount()` | Total allocated nodes |
| `AnodeCount(roots)` | Distinct nodes across BDD array |
| `PathCount(f)` | Number of paths to True terminal |
| `VarProfile(f)` | Node count per variable level |
| `VarOf(f)`, `Low(f)`, `High(f)` | Node field accessors |
| `Stats()` | Comprehensive statistics |
| `PrintDot(f)` / `FprintDot(w,f)` | Graphviz DOT output |
| `PrintTable(f)` | Truth table string |
| `ToDNF(f)` / `ToCNF(f)` / `ToFormula(f)` | Formula string conversion |

### Variable Sets & Pairs

| Function | Description |
|----------|-------------|
| `MakeSet(vars)` | Build BDD variable set from slice |
| `ScanSet(f)` | Extract variables from BDD set |
| `NewPair()` | Create substitution pair |
| `Pair.Set(old, new)` | Map variable → BDD |
| `Pair.SetVar(old, new)` | Map variable → variable |
| `Pair.SetVars(olds, news, n)` | Batch variable → variable |
| `Pair.SetAll(olds, news, n)` | Batch variable → BDD |
| `Pair.Reset()` | Reset to identity mapping |

### Reordering

| Function | Description |
|----------|-------------|
| `SwapVar(v1,v2)` | Swap two adjacent variables |
| `Sift()` | Rudell's sifting (full reorder pass) |

### Serialization

| Function | Description |
|----------|-------------|
| `Save(w,f)` | Write buddy-compatible format to `io.Writer` |
| `Load(r)` | Read and rebuild from `io.Reader` |

### Reference Counting

| Function | Description |
|----------|-------------|
| `AddRef(f)` | Increment external reference count |
| `DelRef(f)` | Decrement external reference count |
| `RefCount(f)` | Query reference count |

### Lifecycle & Tuning

| Function | Description |
|----------|-------------|
| `Done()` | Release all resources |
| `IsRunning()` | Check if initialized |
| `SetVarNum(n)` / `ExtVarNum(n)` | Resize variable count |
| `SetCacheRatio(r)` | Resize operator cache |

---

## Buddy Parity

GOBDD targets full API parity with [Buddy](https://sourceforge.net/projects/buddy/) by Jorn Lind-Nielsen.

**71 functions implemented** covering the complete operational API: all Boolean operators, quantifiers (existential, universal, unique), apply-with-quantify fused operations, restriction, constraint, simplification, composition, substitution, variable sets, pairs, reordering (swap + sift), serialization (save/load), reference counting, and lifecycle management.

The remaining ~10 unimplemented functions are debug-printing variants (`bdd_printset`, `bdd_printall`) and advanced reorder methods (WIN2, WIN3, SIFTITE, auto-reorder) not required by downstream consumers.

---

## Architecture

```
gobdd/
├── bdd.go          BDD manager, unique table, ITE, op-cache
├── types.go        NodeID, OpCode distinct types
├── ops.go          Constrain, Simplify, Apply, BuildCube, PathCount
├── quant.go        ExistSet, ForAllSet, Unique, quantRec, RestrictBDD
├── sat.go          SatOne, FullSatOne, SatOneSet, AllSat
├── replace.go      Pair, Replace, VecCompose, SwapVar
├── reorder.go      Sift (Rudell), swapLevels, Stats
├── serialize.go    Save/Load (buddy-compatible format)
├── refcount.go     AddRef, DelRef, automatic remap during reorder
├── mgmt.go         Done, SetVarNum, ExtVarNum, SetCacheRatio
├── io.go           PrintDot, PrintTable, MakeSet, ScanSet, VarProfile
├── convert.go      ToDNF, ToCNF, ToFormula
└── LICENSES/
    └── buddy.txt   Original Buddy license
```

**Key design decisions:**

- **Level indirection** (`var2level` / `level2var`): Enables atomic variable swap without rebuilding the entire BDD from formula. The node table stores levels (positions), not variables — levels are swapped by updating two arrays.
- **Distinct types** (`NodeID`, `OpCode`): Go defined types prevent argument transposition (passing an operator code where a node handle is expected). The compiler catches these at build time.
- **Pool-backed unique table**: Open-addressing hash table keyed by `(level, lo, hi)` with multiplicative hashing and power-of-two sizing.
- **ITE memoization**: Operator cache avoids recomputing identical ITE subproblems, giving polynomial-time complexity on DAG-structured inputs.
- **Post-order save/load**: Serialization format matches Buddy's layout exactly, enabling cross-compatibility.

---

## Testing & Coverage

```bash
go test -v -race -cover ./...
```

- **95 tests**, all passing with `-race`
- Covers every exported function with dedicated test cases
- CI workflow (`.github/workflows/test.yml`) runs tests, race detector, and publishes a coverage badge to the `coverage` branch

---

## License & Attribution

GOBDD is a Go port of [Buddy](http://buddy.sourceforge.net/) by **Jorn Lind-Nielsen** (1996–2002). The original Buddy library is provided under a permissive MIT-style license that allows modification, redistribution, and re-licensing of derivative works.

- **GOBDD**: MIT License © 2026 xDarkicex ([LICENSE](./LICENSE))
- **Buddy**: Original license text in [LICENSES/buddy.txt](./LICENSES/buddy.txt)

This project would not exist without Jorn Lind-Nielsen's elegant BDD implementation, which has served as the reference design for symbolic model checking libraries worldwide for over two decades. Thank you, Jorn.
