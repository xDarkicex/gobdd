# BDD Theory

## What is a BDD?

A Binary Decision Diagram (BDD) is a directed acyclic graph (DAG) that represents a Boolean function canonically. Given a fixed variable ordering, every Boolean function has exactly one reduced, ordered BDD representation. Two functions are equivalent if and only if their BDDs are structurally identical — O(1) comparison after construction.

BDDs were introduced by Randal Bryant in 1986. They compress Boolean functions by sharing isomorphic subgraphs and eliminating redundant nodes.

## Shannon Expansion

Every Boolean function can be decomposed via Shannon expansion:

```
f(x₁, ..., xₙ) = (xᵢ ∧ f[xᵢ:=1]) ∨ (¬xᵢ ∧ f[xᵢ:=0])
```

Equivalently, using the ITE (if-then-else) operator:

```
f = ITE(xᵢ, fₕᵢ, fₗₒ)
```

where `fₕᵢ = f[xᵢ:=1]` (positive cofactor) and `fₗₒ = f[xᵢ:=0]` (negative cofactor).

A BDD node for variable `xᵢ` stores:
- `level`: the variable's position in the ordering
- `lo`: child node when `xᵢ = 0` (negative cofactor)
- `hi`: child node when `xᵢ = 1` (positive cofactor)

Terminal nodes are the constants `0` (False) and `1` (True).

## Variable Ordering

The variable ordering is critical. The same function can have O(n) nodes with a good ordering and O(2ⁿ) nodes with a bad one. Finding the optimal ordering is NP-hard, but heuristics like Rudell's sifting produce near-optimal results in practice.

**Example:** `(a ∧ b) ∨ (c ∧ d)`

Good order `a < b < c < d`: 4 variable nodes.
Bad order `a < c < b < d`: 6 variable nodes (can't share subgraphs).

## Reduction Rules

A BDD is *reduced* when no further simplifications are possible:

1. **Isomorphism elimination**: Nodes with identical `(level, lo, hi)` are merged. This is enforced by the unique table.

2. **Redundancy elimination**: Nodes where `lo == hi` are removed — the function doesn't depend on that variable at this point. In GOBDD, `unique(level, lo, hi)` returns `lo` when `lo == hi`.

## The ITE Algorithm

All Boolean operations reduce to ITE (if-then-else):

```
ITE(f, g, h) = (f ∧ g) ∨ (¬f ∧ h)
```

Standard operations expressed as ITE:

| Operation | ITE form |
|-----------|----------|
| `¬f` | `ITE(f, 0, 1)` |
| `f ∧ g` | `ITE(f, g, 0)` |
| `f ∨ g` | `ITE(f, 1, g)` |
| `f → g` | `ITE(f, g, 1)` |
| `f ⊕ g` | `ITE(f, ¬g, g)` |
| `f ↔ g` | `ITE(f, g, ¬g)` |

The ITE algorithm computes `ITE(f, g, h)` recursively:

```
ITE(f, g, h):
    if f == 1:       return g
    if f == 0:       return h
    if g == h:       return g
    if g == 1, h == 0: return f

    if cached(f, g, h): return cache[f,g,h]

    v = top variable among f, g, h
    lo = ITE(f[v:=0], g[v:=0], h[v:=0])
    hi = ITE(f[v:=1], g[v:=1], h[v:=1])

    if lo == hi: return lo
    r = make_node(v, lo, hi)
    cache[f,g,h] = r
    return r
```

With memoization, the worst-case complexity is O(|f| · |g| · |h|), but in practice it's O(|f| + |g| + |h|) for most operations.

## Quantification

### Existential quantification

```
∃v. f = f[v:=0] ∨ f[v:=1]
```

Existentially quantifying variable `v` from `f` means: "is there some value of `v` that makes `f` true?" The result is a function of the remaining variables. Computed by OR-ing the two cofactors.

### Universal quantification

```
∀v. f = f[v:=0] ∧ f[v:=1]
```

Universally quantifying variable `v` from `f` means: "for all values of `v`, is `f` true?" Computed by AND-ing the two cofactors.

### Unique quantification

```
∃!v. f = f[v:=0] ⊕ f[v:=1]
```

"Does exactly one value of `v` make `f` true?" Computed by XOR-ing the two cofactors. This is used in modal logic for deterministic transitions.

### Quantification over sets

Multiple variables can be quantified simultaneously by walking the BDD and applying the combine operator (OR for ∃, AND for ∀, XOR for ∃!) at each level in the variable set. GOBDD's `quantRec` does this in a single bottom-up pass.

## Generalized Cofactor (Constrain)

Buddy's `bdd_constrain(f, c)` computes a function that agrees with `f` whenever `c` is true, but may differ when `c` is false. Unlike restriction (which fixes specific variables), constrain allows the constraint to be an arbitrary BDD. This is used in symbolic model checking for image computation.

## Relational Product

```
RelProd(f, g, vars) = ∃vars. (f ∧ g)
```

A core operation in symbolic model checking. Given a state set `f` and a transition relation `g`, the relational product computes the set of successor states. Fusing the AND and quantification into a single pass (AppEx) avoids building the full intermediate product BDD.

## Variable Reordering

### Why reorder?

The BDD size depends critically on variable ordering. A function with `n` variables can have anywhere from O(n) to O(2ⁿ) nodes. Reordering heuristics try to find a compact ordering.

### Sifting (Rudell 1993)

For each variable:
1. Move it to the top of the ordering
2. Sift it down through all positions, tracking the configuration with fewest nodes
3. Move it back to the best position

Each swap of adjacent variables is local: only nodes at the two affected levels need to be updated. With level indirection, a swap is O(nodes at those levels).

### Level indirection

The key insight for efficient swapping: nodes store their **level** (position in ordering), not their variable. Two arrays track the mapping:

```
var2level[v] = position of variable v
level2var[l] = variable at position l
```

Swapping two adjacent levels requires:
1. Swapping the two arrays' entries
2. Rebuilding nodes at just those two levels

Without level indirection, every node in the BDD would need its variable number updated — O(N) instead of O(nodes at affected levels).

## Satisfying Assignments

### Counting

The number of satisfying assignments for function `f` over `n` variables is:

```
satcount(f) = 0                                    if f == 0
satcount(f) = 2^n                                  if f == 1
satcount(f) = satcount(lo) * 2^(gap_lo) + satcount(hi) * 2^(gap_hi)   otherwise
```

where `gap_lo` is the number of skipped variables between this node's level and its lo child's level. Variables that don't appear on a path are "don't care" — both values satisfy the path.

### Model extraction

`SatOne` walks the BDD preferring the lo (0) branch when non-zero, otherwise following hi (1). This produces one minterm (conjunction of literals) that implies the function. `AllSat` enumerates all satisfying assignments by recursively exploring both branches and handling don't-care variables.

## References

- Bryant, R.E. (1986). "Graph-Based Algorithms for Boolean Function Manipulation". IEEE Transactions on Computers.
- Rudell, R. (1993). "Dynamic Variable Ordering for Ordered Binary Decision Diagrams". ICCAD.
- Lind-Nielsen, J. (1996–2002). Buddy BDD Library. http://buddy.sourceforge.net/
