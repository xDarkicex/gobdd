package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xDarkicex/gobdd"
	"github.com/xDarkicex/memory"
)

func newExampleBDD(t *testing.T) *gobdd.BDD {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	return gobdd.New(4, pool)
}

func TestBooleanOps(t *testing.T) {
	b := newExampleBDD(t)
	x0 := b.Var(0)
	x1 := b.Var(1)
	x2 := b.Var(2)
	x3 := b.Var(3)

	f := b.Or(b.And(x0, x1), b.And(x2, b.Not(x3)))

	if f == gobdd.False || f == gobdd.True {
		t.Error("formula should not be constant")
	}
	support := b.Support(f)
	if len(support) != 4 {
		t.Errorf("support should have 4 vars, got %v", support)
	}
}

func TestSatisfiability(t *testing.T) {
	b := newExampleBDD(t)
	x0 := b.Var(0)
	x1 := b.Var(1)
	x2 := b.Var(2)
	x3 := b.Var(3)
	f := b.Or(b.And(x0, x1), b.And(x2, b.Not(x3)))

	count := b.SatisfyCount(f)
	// (x0∧x1)∨(x2∧¬x3) over 4 vars: true when (x0=1,x1=1) or (x2=1,x3=0).
	if count == 0 {
		t.Error("formula should be satisfiable")
	}

	assign := b.SatisfyOne(f)
	if assign == nil || len(assign) != 4 {
		t.Error("SatisfyOne should return 4-element assignment")
	}

	// AllSat should produce `count` unique assignments.
	seen := make(map[string]bool)
	b.AllSat(f, func(a []bool) {
		seen[formatAssign(a)] = true
	})
	if len(seen) != int(count) {
		t.Logf("AllSat gave %d assignments, satcount=%d", len(seen), count)
	}
}

func TestQuantification(t *testing.T) {
	b := newExampleBDD(t)
	x0 := b.Var(0)
	x1 := b.Var(1)
	f := b.And(x0, x1)

	// ∃x0.(x0 ∧ x1) = x1
	r := b.Exists(f, 0)
	if r != x1 {
		t.Error("∃x0.(x0∧x1) should equal x1")
	}

	// ∀x0.(x0 ∧ x1) = False
	r2 := b.ForAll(f, 0)
	if r2 != gobdd.False {
		t.Error("∀x0.(x0∧x1) should be False")
	}

	// BDD varset quantification
	varset := b.MakeSet([]int32{0})
	r3 := b.ExistSet(f, varset)
	if r3 != x1 {
		t.Error("BDD varset ∃{0}.(x0∧x1) should equal x1")
	}
}

func TestVariableSets(t *testing.T) {
	b := newExampleBDD(t)
	set := b.MakeSet([]int32{0, 2, 3})
	vars := b.ScanSet(set)
	if len(vars) != 3 || vars[0] != 0 || vars[1] != 2 || vars[2] != 3 {
		t.Errorf("MakeSet/ScanSet round-trip: got %v, want [0,2,3]", vars)
	}
}

func TestSubstitution(t *testing.T) {
	b := newExampleBDD(t)
	x0 := b.Var(0)
	x1 := b.Var(1)
	x2 := b.Var(2)
	x3 := b.Var(3)
	f := b.Or(b.And(x0, x1), b.And(x2, b.Not(x3)))

	pair := b.NewPair()
	pair.Set(0, x2)
	substituted := b.Replace(f, pair)

	// f[x0:=x2] should have x1, x2, x3 in support, but not x0
	s := b.Support(substituted)
	hasX0 := false
	for _, v := range s {
		if v == 0 {
			hasX0 = true
		}
	}
	if hasX0 {
		t.Error("substituted formula should not contain x0")
	}
}

func TestSerialization(t *testing.T) {
	b := newExampleBDD(t)
	x0 := b.Var(0)
	x1 := b.Var(1)
	f := b.And(x0, x1)

	var buf bytes.Buffer
	if err := b.Save(&buf, f); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Error("serialized data should not be empty")
	}

	loaded, err := b.Load(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if b.SatisfyCount(b.Xor(f, loaded)) != 0 {
		t.Error("loaded BDD should be equivalent to original")
	}
}

func TestReordering(t *testing.T) {
	b := newExampleBDD(t)
	x0 := b.Var(0)
	x1 := b.Var(1)
	f := b.And(x0, b.Not(x1))

	before := b.SatisfyCount(f)
	remap := b.SwapVar(0, 1)
	f = remap(f)
	after := b.SatisfyCount(f)

	if before != after {
		t.Errorf("satcount changed after swap: %d → %d", before, after)
	}
}
