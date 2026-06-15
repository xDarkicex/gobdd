package gobdd

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func newBDD(t *testing.T, vars int) *BDD {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Reset)
	return New(vars, pool)
}

func TestNew(t *testing.T) {
	b := newBDD(t, 4)
	if b.NodeCount() != 2 {
		t.Errorf("new BDD: got %d nodes, want 2 (False+True)", b.NodeCount())
	}
}

func TestVar(t *testing.T) {
	b := newBDD(t, 4)
	v0 := b.Var(0)
	if v0 < 2 {
		t.Error("variable node should be >= 2")
	}
}

func TestNot(t *testing.T) {
	b := newBDD(t, 2)
	v := b.Var(0)
	n := b.Not(v)
	// n should be different from v but not False
	if n == v || n == falseIdx {
		t.Errorf("Not(var) = %d, want a different non-terminal", n)
	}
}

func TestAnd(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	r := b.And(a, bb)
	if r == falseIdx {
		t.Error("And(a,b) should not be False")
	}
	// And(a, Not(a)) should be False
	fa := b.And(a, b.Not(a))
	if fa != falseIdx {
		t.Error("And(a, ¬a) should be False")
	}
}

func TestOr(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	r := b.Or(a, bb)
	if r == falseIdx {
		t.Error("Or(a,b) should not be False")
	}
	// Or(a, Not(a)) should be True
	ta := b.Or(a, b.Not(a))
	if ta != trueIdx {
		t.Error("Or(a, ¬a) should be True")
	}
}

func TestImplies(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	// a → a should be True
	if b.Implies(a, a) != trueIdx {
		t.Error("a → a should be True")
	}
	// a → b should not be True (not a tautology)
	if b.Implies(a, bb) == trueIdx {
		t.Error("a → b should NOT be True")
	}
}

func TestXor(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	// a xor a should be False
	if b.Xor(a, a) != falseIdx {
		t.Error("a xor a should be False")
	}
	// a xor b should not be False
	if b.Xor(a, bb) == falseIdx {
		t.Error("a xor b should NOT be False")
	}
}

func TestEquiv(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	if b.Equiv(a, a) != trueIdx {
		t.Error("a ↔ a should be True")
	}
	if b.Equiv(a, bb) == trueIdx {
		t.Error("a ↔ b should NOT be True")
	}
}

func TestITE(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	// ITE(true, b, c) = b
	if b.ITE(trueIdx, bb, c) != bb {
		t.Error("ITE(true, b, c) should be b")
	}
	// ITE(false, b, c) = c
	if b.ITE(falseIdx, bb, c) != c {
		t.Error("ITE(false, b, c) should be c")
	}
	// ITE(a, b, b) = b
	if b.ITE(a, bb, bb) != bb {
		t.Error("ITE(a, b, b) should be b")
	}
}

func TestSatisfyOne(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	// a ∧ b ∧ c — only {true, true, true}
	f := b.And(a, b.And(bb, c))
	assign := b.SatisfyOne(f)
	if assign == nil {
		t.Fatal("expected assignment")
	}
	if len(assign) != 3 {
		t.Fatalf("expected 3 vars, got %d", len(assign))
	}
	if !assign[0] || !assign[1] || !assign[2] {
		t.Errorf("expected all true, got %v", assign)
	}
	// False should return nil
	if b.SatisfyOne(falseIdx) != nil {
		t.Error("SatisfyOne(False) should return nil")
	}
}

func TestSatisfyOneOr(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	// a ∨ b — many satisfying assignments
	f := b.Or(a, bb)
	assign := b.SatisfyOne(f)
	if assign == nil {
		t.Fatal("expected assignment")
	}
	if len(assign) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(assign))
	}
	// At least one must be true
	if !assign[0] && !assign[1] {
		t.Errorf("expected at least one true, got %v", assign)
	}
}

func TestNodeCount(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	d := b.Var(3)
	_ = b.And(a, b.Or(bb, b.And(c, d)))
	if b.NodeCount() < 2 {
		t.Error("NodeCount should be >= 2 after building formulas")
	}
}

func TestUniqueTable(t *testing.T) {
	b := newBDD(t, 2)
	a1 := b.Var(0)
	a2 := b.Var(0)
	// Same variable should return same node
	if a1 != a2 {
		t.Error("Var(0) called twice should return same node")
	}
}

func TestOpCache(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	// Same operation should be cached
	r1 := b.And(a, bb)
	r2 := b.And(a, bb)
	if r1 != r2 {
		t.Error("And(a,b) called twice should return same result (cached)")
	}
}

func TestComplexFormula(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	d := b.Var(3)
	// (a ∧ b) ∨ (c ∧ d)
	f := b.Or(b.And(a, bb), b.And(c, d))
	if f == falseIdx || f == trueIdx {
		t.Error("(a∧b)∨(c∧d) should not be constant")
	}
}

func TestLargeVarCount(t *testing.T) {
	b := newBDD(t, 32)
	a := b.Var(0)
	z := b.Var(31)
	f := b.And(a, z)
	if f == falseIdx || f == trueIdx {
		t.Error("a∧z with 32 vars should not be constant")
	}
}

func TestDeepNesting(t *testing.T) {
	b := newBDD(t, 5)
	f := b.Var(0)
	for i := int32(1); i < 5; i++ {
		f = b.And(f, b.Var(i))
	}
	if f == falseIdx {
		t.Error("conjunction of 5 vars should not be False")
	}
}
