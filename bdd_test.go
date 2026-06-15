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
		t.Errorf("got %d nodes, want 2", b.NodeCount())
	}
}

func TestVar(t *testing.T) {
	b := newBDD(t, 4)
	v0 := b.Var(0)
	if v0 != b.Var(0) {
		t.Error("Var(0) should return same node")
	}
	if v0 < 2 {
		t.Error("variable node should be >= 2")
	}
	if b.Var(-1) != falseIdx {
		t.Error("Var(-1) = False")
	}
	if b.Var(5) != falseIdx {
		t.Error("Var(5) = False")
	}
}

func TestNot(t *testing.T) {
	b := newBDD(t, 2)
	v := b.Var(0)
	n := b.Not(v)
	if n == v || n == falseIdx || n == trueIdx {
		t.Error("Not(var) should be non-terminal, different from var")
	}
	if b.Not(b.Not(v)) != v {
		t.Error("¬¬v = v")
	}
}

func TestAnd(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	if b.And(a, bb) == falseIdx {
		t.Error("a∧b ≠ False")
	}
	if b.And(a, b.Not(a)) != falseIdx {
		t.Error("a∧¬a = False")
	}
	if b.And(a, trueIdx) != a {
		t.Error("a∧true = a")
	}
}

func TestOr(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	if b.Or(a, b.Not(a)) != trueIdx {
		t.Error("a∨¬a = True")
	}
	if b.Or(a, falseIdx) != a {
		t.Error("a∨false = a")
	}
}

func TestImplies(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	if b.Implies(a, a) != trueIdx {
		t.Error("a→a = True")
	}
	if b.Implies(a, bb) == trueIdx {
		t.Error("a→b ≠ True")
	}
}

func TestXor(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	if b.Xor(a, a) != falseIdx {
		t.Error("a⊕a = False")
	}
	if b.Xor(a, b.Not(a)) != trueIdx {
		t.Error("a⊕¬a = True")
	}
}

func TestEquiv(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	if b.Equiv(a, a) != trueIdx {
		t.Error("a↔a = True")
	}
	if b.Equiv(a, b.Not(a)) != falseIdx {
		t.Error("a↔¬a = False")
	}
}

func TestNand(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	if b.Nand(a, a) != b.Not(a) {
		t.Error("a nand a = ¬a")
	}
	if b.Nand(a, bb) == falseIdx {
		t.Error("a nand b ≠ False")
	}
}

func TestNor(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	if b.Nor(a, a) != b.Not(a) {
		t.Error("a nor a = ¬a")
	}
}

func TestITE(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	if b.ITE(trueIdx, bb, c) != bb {
		t.Error("ITE(true,b,c) = b")
	}
	if b.ITE(falseIdx, bb, c) != c {
		t.Error("ITE(false,b,c) = c")
	}
	if b.ITE(a, trueIdx, falseIdx) != a {
		t.Error("ITE(a,true,false) = a")
	}
}

func TestRestrict(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	if b.Restrict(a, 0, true) != trueIdx {
		t.Error("a|a=true = True")
	}
	if b.Restrict(a, 0, false) != falseIdx {
		t.Error("a|a=false = False")
	}
	ab := b.And(a, bb)
	if b.Restrict(ab, 0, true) != bb {
		t.Error("(a∧b)|a=true = b")
	}
}

func TestExists(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	if b.Exists(a, 0) != trueIdx {
		t.Error("∃a.a = True")
	}
}

func TestExistsAll(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	f := b.And(a, b.And(bb, c))
	r := b.ExistsAll(f, []int32{0, 1})
	if r == falseIdx || r == trueIdx {
		t.Error("∃{a,b}.(a∧b∧c) should be c, not constant")
	}
}

func TestForAll(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	if b.ForAll(a, 0) != falseIdx {
		t.Error("∀a.a = False")
	}
	taut := b.Or(a, b.Not(a))
	if b.ForAll(taut, 0) != trueIdx {
		t.Error("∀a.(a∨¬a) = True")
	}
}

func TestForAllVars(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	taut := b.Or(a, b.Not(a))
	if b.ForAllVars(taut, []int32{0}) != trueIdx {
		t.Error("∀{a}.(a∨¬a) = True")
	}
}

func TestCompose(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	if b.Compose(a, 0, bb) != bb {
		t.Error("a[a:=b] = b")
	}
	ac := b.And(a, c)
	bc := b.And(bb, c)
	if b.Compose(ac, 0, bb) != bc {
		t.Error("(a∧c)[a:=b] = b∧c")
	}
}

func TestSupport(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(2)
	f := b.And(a, bb)
	s := b.Support(f)
	if len(s) != 2 || s[0] != 0 || s[1] != 2 {
		t.Errorf("Support(a∧b@var2): got %v, want [0,2]", s)
	}
	if len(b.Support(trueIdx)) != 0 {
		t.Error("Support(True) = []")
	}
}

func TestSatisfyOne(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	f := b.And(a, b.And(bb, c))
	assign := b.SatisfyOne(f)
	if assign == nil || !assign[0] || !assign[1] || !assign[2] {
		t.Error("a∧b∧c: all true")
	}
	if b.SatisfyOne(falseIdx) != nil {
		t.Error("SatisfyOne(False) = nil")
	}
}

func TestSatisfyCount(t *testing.T) {
	b := newBDD(t, 3)
	if b.SatisfyCount(trueIdx) == 0 {
		t.Error("True count > 0")
	}
	if b.SatisfyCount(falseIdx) != 0 {
		t.Error("False count = 0")
	}
}

func TestComplexFormula(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	d := b.Var(3)
	f := b.Or(b.And(a, bb), b.And(c, d))
	if f == falseIdx || f == trueIdx {
		t.Error("(a∧b)∨(c∧d) not constant")
	}
}

func TestLargeVarCount(t *testing.T) {
	b := newBDD(t, 64)
	a := b.Var(0)
	z := b.Var(63)
	f := b.And(a, z)
	if f == falseIdx || f == trueIdx {
		t.Error("a∧z with 64 vars not constant")
	}
}

func TestDeepNesting(t *testing.T) {
	b := newBDD(t, 10)
	f := b.Var(0)
	for i := int32(1); i < 10; i++ {
		f = b.And(f, b.Var(i))
	}
	if f == falseIdx {
		t.Error("∧ of 10 vars ≠ False")
	}
}

func TestNodeCountGrowing(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	d := b.Var(3)
	_ = b.And(a, b.Or(bb, b.And(c, d)))
	if b.NodeCount() < 6 {
		t.Error("complex formula should create multiple nodes")
	}
}

func TestCacheHit(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	r1 := b.And(a, bb)
	r2 := b.And(a, bb)
	if r1 != r2 {
		t.Error("cached result differs")
	}
}

func TestVarCount(t *testing.T) {
	b := newBDD(t, 16)
	if b.VarCount() != 16 {
		t.Errorf("VarCount: got %d, want 16", b.VarCount())
	}
}
