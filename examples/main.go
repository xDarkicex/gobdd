// Example usage of the GOBDD library.
//
// This program demonstrates core BDD operations: building formulas,
// Boolean operations, quantification, satisfiability, model extraction,
// variable sets, substitution, serialization, and reordering.
package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xDarkicex/gobdd"
	"github.com/xDarkicex/memory"
)

func main() {
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		panic(err)
	}
	defer pool.Reset()

	b := gobdd.New(4, pool) // 4 variables: x0, x1, x2, x3

	// ── Boolean operations ───────────────────────────────────────────

	x0 := b.Var(0)
	x1 := b.Var(1)
	x2 := b.Var(2)
	x3 := b.Var(3)

	// (x0 ∧ x1) ∨ (x2 ∧ ¬x3)
	f := b.Or(
		b.And(x0, x1),
		b.And(x2, b.Not(x3)),
	)

	fmt.Println("── Formula ──")
	fmt.Printf("(x0 ∧ x1) ∨ (x2 ∧ ¬x3) = %s\n", b.ToDNF(f))
	fmt.Printf("Nodes: %d\n", b.NodeCount())
	fmt.Printf("Support: %v\n", b.Support(f))

	// ── Satisfiability ───────────────────────────────────────────────

	fmt.Println("\n── Satisfiability ──")
	fmt.Printf("Satisfying assignments: %d\n", b.SatisfyCount(f))
	fmt.Printf("One assignment: %v\n", formatAssign(b.SatisfyOne(f)))
	fmt.Printf("Full assignment: %v\n", formatAssign(b.FullSatOne(f)))

	// Enumerate all
	var models []string
	b.AllSat(f, func(assign []bool) {
		models = append(models, formatAssign(assign))
	})
	fmt.Printf("All models (%d):\n", len(models))
	for _, m := range models {
		fmt.Printf("  %s\n", m)
	}

	// ── Quantification ───────────────────────────────────────────────

	fmt.Println("\n── Quantification ──")
	existsX0 := b.Exists(f, 0)
	fmt.Printf("∃x0. f : support=%v count=%d\n", b.Support(existsX0), b.SatisfyCount(existsX0))

	forAllX0 := b.ForAll(f, 0)
	fmt.Printf("∀x0. f : %s\n", b.ToFormula(forAllX0))

	// Quantify over a set
	varset := b.MakeSet([]int32{0, 1})
	existsSet := b.ExistSet(f, varset)
	fmt.Printf("∃{0,1}. f : support=%v\n", b.Support(existsSet))

	// ── Variable sets ────────────────────────────────────────────────

	fmt.Println("\n── Variable Sets ──")
	set := b.MakeSet([]int32{0, 2, 3})
	vars := b.ScanSet(set)
	fmt.Printf("MakeSet({0,2,3}) → scan: %v\n", vars)

	// ── Substitution ─────────────────────────────────────────────────

	fmt.Println("\n── Substitution ──")
	pair := b.NewPair()
	pair.Set(0, x2)          // x0 := x2
	pair.Set(1, b.Not(x3))   // x1 := ¬x3
	substituted := b.Replace(f, pair)
	fmt.Printf("f[x0:=x2, x1:=¬x3] = %s\n", b.ToDNF(substituted))

	// ── Serialization ────────────────────────────────────────────────

	fmt.Println("\n── Serialization ──")
	var buf bytes.Buffer
	if err := b.Save(&buf, f); err != nil {
		panic(err)
	}
	loaded, err := b.Load(strings.NewReader(buf.String()))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Round-trip: %d bytes, equivalent=%v\n",
		buf.Len(), b.SatisfyCount(b.Xor(f, loaded)) == 0)

	// ── Reordering ───────────────────────────────────────────────────

	fmt.Println("\n── Reordering ──")
	before := b.NodeCount()
	remap := b.SwapVar(0, 1)
	f = remap(f)
	after := b.NodeCount()
	fmt.Printf("SwapVar(0,1): nodes %d → %d, count=%d\n",
		before, after, b.SatisfyCount(f))
}

func formatAssign(a []bool) string {
	if a == nil {
		return "<none>"
	}
	s := make([]string, len(a))
	for i, v := range a {
		if v {
			s[i] = "1"
		} else {
			s[i] = "0"
		}
	}
	return strings.Join(s, "")
}
