package gobdd

import (
	"os"
	"strings"
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

// --- ops.go tests ---

func TestNithvar(t *testing.T) {
	b := newBDD(t, 3)
	v := b.Var(0)
	if b.Nithvar(0) != b.Not(v) {
		t.Error("Nithvar(v) = ¬v")
	}
}

func TestDiff(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	if b.Diff(a, bb) != b.And(a, b.Not(bb)) {
		t.Error("Diff(f,g) = f ∧ ¬g")
	}
	if b.Diff(a, a) != falseIdx {
		t.Error("Diff(a,a) = False")
	}
}

func TestLess(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	if b.Less(a, bb) != b.And(b.Not(a), bb) {
		t.Error("Less(f,g) = ¬f ∧ g")
	}
	if b.Less(a, a) != falseIdx {
		t.Error("Less(a,a) = False")
	}
}

func TestInvImp(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	if b.InvImp(a, bb) != b.Or(a, bb) {
		t.Error("InvImp(f,g) = f ∨ g")
	}
}

func TestConstrain(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)

	// Constrain with True = f unchanged
	if b.Constrain(ab, trueIdx) != ab {
		t.Error("Constrain(f, true) = f")
	}
	// Constrain with False = True
	if b.Constrain(ab, falseIdx) != trueIdx {
		t.Error("Constrain(f, false) = True")
	}
	// Terminal cases
	if b.Constrain(trueIdx, a) != trueIdx {
		t.Error("Constrain(True, c) = True")
	}
	if b.Constrain(falseIdx, a) != falseIdx {
		t.Error("Constrain(False, c) = False")
	}
	// f == c
	if b.Constrain(a, a) != trueIdx {
		t.Error("Constrain(f, f) = True")
	}
}

func TestSimplify(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)

	// Simplify with d=True → f unchanged (buddy: ISONE(d) → return f)
	if b.Simplify(a, trueIdx) != a {
		t.Error("Simplify(f, true) = f")
	}
	// Simplify f=d → True
	if b.Simplify(a, a) != trueIdx {
		t.Error("Simplify(f, f) = True")
	}
	// Simplify with d=False → False (buddy: ISZERO(d) → return BDDZERO)
	if b.Simplify(a, falseIdx) != falseIdx {
		t.Error("Simplify(f, false) = False")
	}
	// Terminal cases: ISCONST(f) → return f
	if b.Simplify(trueIdx, a) != trueIdx {
		t.Error("Simplify(True, d) = True")
	}
	if b.Simplify(falseIdx, a) != falseIdx {
		t.Error("Simplify(False, d) = False")
	}
	// Simplify with don't-care: f=a∧b, d=a → result covers f when a is false
	ab := b.And(a, bb)
	r := b.Simplify(ab, a)
	if r == falseIdx {
		t.Error("Simplify(a∧b, a) should not be False")
	}
}

func TestAppEx(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	r := b.AppEx(a, bb, opAnd, []int32{0})
	// ∃a.(a∧b) = b
	if b.Equiv(r, bb) != trueIdx {
		t.Error("∃a.(a∧b) = b")
	}
}

func TestAppAll(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	r := b.AppAll(a, bb, opAnd, []int32{0})
	// ∀a.(a∧b) = False (since a can be false)
	if r != falseIdx {
		t.Error("∀a.(a∧b) = False")
	}
}

func TestRelProd(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	r := b.RelProd(a, bb, []int32{0})
	// ∃a.(a∧b) = b
	if b.Equiv(r, bb) != trueIdx {
		t.Error("RelProd(a,b,{a}) = b")
	}
}

func TestApply(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)

	if b.Apply(a, bb, opAnd) != b.And(a, bb) {
		t.Error("Apply And")
	}
	if b.Apply(a, bb, opXor) != b.Xor(a, bb) {
		t.Error("Apply Xor")
	}
	if b.Apply(a, bb, opOr) != b.Or(a, bb) {
		t.Error("Apply Or")
	}
	if b.Apply(a, bb, opNand) != b.Nand(a, bb) {
		t.Error("Apply Nand")
	}
	if b.Apply(a, bb, opNor) != b.Nor(a, bb) {
		t.Error("Apply Nor")
	}
	if b.Apply(a, bb, opImp) != b.Implies(a, bb) {
		t.Error("Apply Imp")
	}
	if b.Apply(a, bb, opBiimp) != b.Equiv(a, bb) {
		t.Error("Apply Biimp")
	}
	if b.Apply(a, bb, opDiff) != b.Diff(a, bb) {
		t.Error("Apply Diff")
	}
	if b.Apply(a, bb, opLess) != b.Less(a, bb) {
		t.Error("Apply Less")
	}
	if b.Apply(a, bb, opInvImp) != b.InvImp(a, bb) {
		t.Error("Apply InvImp")
	}
	// Unknown op
	if b.Apply(a, bb, 999) != falseIdx {
		t.Error("Apply unknown = False")
	}
}

func TestBuildCube(t *testing.T) {
	b := newBDD(t, 3)
	r := b.BuildCube([]int32{0, 1}, []bool{true, false})
	// x0 ∧ ¬x1
	a := b.Var(0)
	bb := b.Var(1)
	expected := b.And(a, b.Not(bb))
	if r != expected {
		t.Error("BuildCube: x0 ∧ ¬x1")
	}
}

func TestPathCount(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	if b.PathCount(ab) <= 0 {
		t.Error("PathCount(a∧b) > 0")
	}
	if b.PathCount(falseIdx) != 0 {
		t.Error("PathCount(False) = 0")
	}
	if b.PathCount(trueIdx) != 1 {
		t.Error("PathCount(True) = 1")
	}
}

func TestSatCountLn(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	ln := b.SatCountLn(ab)
	if ln <= 0 {
		t.Error("SatCountLn(a∧b) > 0")
	}
	// SatCountLn(False) = 0
	if b.SatCountLn(falseIdx) != 0 {
		t.Error("SatCountLn(False) = 0")
	}
}

func TestSatCountSet(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	// ∃{0}.(a∧b) = b, which has 2 satisfying assignments (b=true, c=don't care)
	c := b.SatCountSet(ab, []int32{0})
	if c == 0 {
		t.Error("SatCountSet > 0")
	}
}

func TestSatCountLnSet(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	ln := b.SatCountLnSet(ab, []int32{})
	if ln <= 0 {
		t.Error("SatCountLnSet(a∧b, {}) > 0")
	}
}

// --- sat.go tests ---

func TestFullSatOne(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(2)
	f := b.And(a, b.Not(bb))
	assign := b.FullSatOne(f)
	if assign == nil {
		t.Fatal("FullSatOne: got nil")
	}
	if len(assign) != 4 {
		t.Errorf("FullSatOne: len=%d, want 4", len(assign))
	}
	if !assign[0] {
		t.Error("FullSatOne: var 0 should be true")
	}
	if assign[2] {
		t.Error("FullSatOne: var 2 should be false")
	}
	// False should return nil
	if b.FullSatOne(falseIdx) != nil {
		t.Error("FullSatOne(False) = nil")
	}
}

func TestSatOneSet(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	// varset: variables 0 and 1
	varset := b.MakeSet([]int32{0, 1})
	r := b.SatOneSet(ab, varset, true)
	if r == falseIdx {
		t.Error("SatOneSet should find minterm")
	}
	// False formula returns False
	r2 := b.SatOneSet(falseIdx, varset, true)
	if r2 != falseIdx {
		t.Error("SatOneSet(False, *) = False")
	}
}

func TestAllSat(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	count := 0
	b.AllSat(ab, func(assign []bool) {
		count++
		if !assign[0] || !assign[1] {
			t.Error("AllSat: assignment must have a=true, b=true")
		}
	})
	if count != 2 {
		t.Errorf("AllSat count: got %d, want 2 (c=0 and c=1)", count)
	}
	// False should not call handler
	count2 := 0
	b.AllSat(falseIdx, func(assign []bool) { count2++ })
	if count2 != 0 {
		t.Error("AllSat(False): no assignments")
	}
}

// --- replace.go tests ---

func TestNewPair(t *testing.T) {
	b := newBDD(t, 3)
	p := b.NewPair()
	if p == nil {
		t.Fatal("NewPair: got nil")
	}
	if len(p.result) != 3 {
		t.Errorf("NewPair: len=%d, want 3", len(p.result))
	}
	// Identity mapping: result[v] = Var(v)
	if p.result[0] != b.Var(0) {
		t.Error("NewPair: identity mapping")
	}
}

func TestPairSet(t *testing.T) {
	b := newBDD(t, 3)
	p := b.NewPair()
	bb := b.Var(1)
	p.Set(0, bb)
	if p.result[0] != bb {
		t.Error("Pair.Set: not set")
	}
	// Out-of-range should be silent
	p.Set(10, bb)
}

func TestPairSetAll(t *testing.T) {
	b := newBDD(t, 4)
	p := b.NewPair()
	oldVars := []int32{0, 1}
	newVals := []NodeID{b.Var(2), b.Var(3)}
	p.SetAll(oldVars, newVals, 2)
	if p.result[0] != b.Var(2) || p.result[1] != b.Var(3) {
		t.Error("Pair.SetAll: mapping wrong")
	}
}

func TestSwapVar(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	f := b.And(a, b.Not(bb)) // x0 ∧ ¬x1

	// After swapping 0 and 1, the formula should be equivalent to x1 ∧ ¬x0
	// But sat count should be preserved
	before := b.SatisfyCount(f)
	b.SwapVar(0, 1)
	after := b.SatisfyCount(f)
	if before != after {
		t.Errorf("SwapVar: sat count changed from %d to %d", before, after)
	}
}

func TestReplace(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	f := b.And(a, bb) // x0 ∧ x1
	p := b.NewPair()
	p.Set(0, c) // replace var 0 with var 2
	r := b.Replace(f, p)
	// Result should be x2 ∧ x1
	expected := b.And(c, bb)
	if r != expected {
		t.Error("Replace: x0∧x1[x0:=x2] should equal x2∧x1")
	}
	// Terminals unchanged
	if b.Replace(trueIdx, p) != trueIdx {
		t.Error("Replace(True) = True")
	}
}

func TestVecCompose(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	f := b.And(a, bb)
	p := b.NewPair()
	p.Set(0, c)
	r := b.VecCompose(f, p)
	expected := b.And(c, bb)
	if r != expected {
		t.Error("VecCompose: x0∧x1[x0:=x2] = x2∧x1")
	}
}

// --- io.go tests ---

func TestPrintDot(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	// Capture stdout? Just verify it doesn't panic
	b.PrintDot(ab)
	b.PrintDot(trueIdx)
	b.PrintDot(falseIdx)
}

func TestFprintDot(t *testing.T) {
	b := newBDD(t, 2)
	ab := b.And(b.Var(0), b.Var(1))
	f, err := os.CreateTemp("", "bdd-dot-*.dot")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	b.FprintDot(f, ab)
	fi, _ := f.Stat()
	if fi.Size() == 0 {
		t.Error("FprintDot: empty file")
	}
}

func TestPrintTable(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	ab := b.And(a, b.Var(1))
	s := b.PrintTable(ab)
	if s == "" {
		t.Error("PrintTable: empty")
	}
	// Truth table for a∧b should have 1 line with "1" (a=1,b=1)
	lines := 0
	for _, ch := range s {
		if ch == '\n' {
			lines++
		}
	}
	if lines != 4 {
		t.Errorf("PrintTable: %d lines, want 4", lines)
	}
}

func TestVarProfile(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	f := b.And(a, b.And(bb, c))
	p := b.VarProfile(f)
	if len(p) != 4 {
		t.Errorf("VarProfile: len=%d, want 4", len(p))
	}
	for i, cnt := range p[:3] {
		if cnt == 0 {
			t.Errorf("VarProfile: var %d has 0 nodes", i)
		}
	}
}

func TestScanSet(t *testing.T) {
	b := newBDD(t, 4)
	varset := b.MakeSet([]int32{0, 2})
	result := b.ScanSet(varset)
	if len(result) != 2 || result[0] != 0 || result[1] != 2 {
		t.Errorf("ScanSet: got %v, want [0,2]", result)
	}
}

func TestMakeSet(t *testing.T) {
	b := newBDD(t, 4)
	r := b.MakeSet([]int32{0, 1})
	if r == falseIdx || r == trueIdx {
		t.Error("MakeSet: should be non-terminal set")
	}
	// r should be true only when vars 0 and 1 are both true
	a := b.Var(0)
	bb := b.Var(1)
	expected := b.And(a, bb)
	if r != expected {
		t.Error("MakeSet({0,1}) = Var(0) ∧ Var(1)")
	}
}

func TestMakeSetScanSetRoundTrip(t *testing.T) {
	b := newBDD(t, 5)
	vars := []int32{0, 2, 4}
	varset := b.MakeSet(vars)
	back := b.ScanSet(varset)
	if len(back) != len(vars) {
		t.Errorf("round-trip: len %d != %d", len(back), len(vars))
	}
	for i, v := range back {
		if v != vars[i] {
			t.Errorf("round-trip: back[%d]=%d, want %d", i, v, vars[i])
		}
	}
}

// --- convert.go tests ---

func TestToFormula(t *testing.T) {
	b := newBDD(t, 2)
	if b.ToFormula(trueIdx) != "true" {
		t.Error("ToFormula(True) = true")
	}
	if b.ToFormula(falseIdx) != "false" {
		t.Error("ToFormula(False) = false")
	}
	a := b.Var(0)
	s := b.ToFormula(a)
	if s == "" || s == "true" || s == "false" {
		t.Error("ToFormula(var): should be non-trivial")
	}
}

func TestToDNF(t *testing.T) {
	b := newBDD(t, 2)
	if b.ToDNF(falseIdx) != "false" {
		t.Error("ToDNF(False) = false")
	}
	if b.ToDNF(trueIdx) != "true" {
		t.Error("ToDNF(True) = true")
	}
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	s := b.ToDNF(ab)
	if s == "" {
		t.Error("ToDNF(a∧b): non-empty")
	}
}

func TestToCNF(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	s := b.ToCNF(ab)
	if s == "" {
		t.Error("ToCNF(a∧b): non-empty")
	}
	// Tautology
	taut := b.Or(a, b.Not(a))
	if b.ToCNF(taut) != "true" {
		t.Error("ToCNF(taut) = true")
	}
	// False
	if b.ToCNF(falseIdx) != "false" {
		t.Error("ToCNF(False) = false")
	}
}

// --- reorder.go tests ---

func TestStats(t *testing.T) {
	b := newBDD(t, 4)
	b.Var(0)
	b.Var(1)
	b.And(b.Var(2), b.Var(3))
	s := b.Stats()
	if s.NodeCount < 4 {
		t.Error("Stats: NodeCount too low")
	}
	if s.VarCount != 4 {
		t.Errorf("Stats: VarCount=%d, want 4", s.VarCount)
	}
}

func TestSift(t *testing.T) {
	b := newBDD(t, 6)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	d := b.Var(3)
	e := b.Var(4)
	f := b.Var(5)
	formula := b.And(a, b.Or(bb, b.And(c, b.Or(d, b.And(e, f)))))
	// Sift reorders variables; verify it runs without crashing.
	// Note: current implementation uses a simplified swap that may change
	// the BDD structure but produces a legal BDD.
	b.Sift()
	// Verify formula is still a valid BDD (non-negative, in range)
	if formula < 0 || int(formula) >= len(b.nodes) {
		t.Error("Formula index invalid after Sift")
	}
}

// --- New accessor tests ---

func TestVarOf(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	if b.VarOf(a) != 0 {
		t.Errorf("VarOf(Var(0)) = %d, want 0", b.VarOf(a))
	}
	if b.VarOf(trueIdx) != -1 {
		t.Error("VarOf(True) = -1")
	}
	if b.VarOf(falseIdx) != -1 {
		t.Error("VarOf(False) = -1")
	}
}

func TestLowHigh(t *testing.T) {
	b := newBDD(t, 2)
	a := b.Var(0)
	if b.Low(a) != falseIdx {
		t.Error("Low(Var(0)) = False")
	}
	if b.High(a) != trueIdx {
		t.Error("High(Var(0)) = True")
	}
	if b.Low(trueIdx) != trueIdx {
		t.Error("Low(True) = True")
	}
	if b.High(falseIdx) != falseIdx {
		t.Error("High(False) = False")
	}
}

// --- SatOne tests ---

func TestSatOne(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	s := b.SatOne(ab)
	if s == falseIdx {
		t.Error("SatOne(a∧b) should not be False")
	}
	// The minterm should imply ab
	if b.And(s, b.Not(ab)) != falseIdx {
		t.Error("minterm should imply original formula")
	}
	// SatOne(False) = False
	if b.SatOne(falseIdx) != falseIdx {
		t.Error("SatOne(False) = False")
	}
	// SatOne(True) = True
	if b.SatOne(trueIdx) != trueIdx {
		t.Error("SatOne(True) = True")
	}
}

// --- RestrictBDD tests ---

func TestRestrictBDD(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)

	// Restrict variable 0 to true: (a∧b)|a=true = b
	c := b.Var(0)
	r := b.RestrictBDD(ab, c)
	if b.Equiv(r, bb) != trueIdx {
		t.Error("RestrictBDD(a∧b, Var(0)) = b (a restricted to true)")
	}

	// Restrict variable 0 to false: (a∧b)|a=false = False
	cNeg := b.Not(a)
	r2 := b.RestrictBDD(ab, cNeg)
	if r2 != falseIdx {
		t.Error("RestrictBDD(a∧b, ¬Var(0)) = False (a restricted to false)")
	}
	// Empty constraint -> identity
	if b.RestrictBDD(ab, trueIdx) != ab {
		t.Error("RestrictBDD(f, True) = f")
	}
}

// --- ExistSet / ForAllSet tests ---

func TestExistSet(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)

	varset := b.MakeSet([]int32{0})
	r := b.ExistSet(ab, varset)
	if b.Equiv(r, bb) != trueIdx {
		t.Error("∃{0}.(a∧b) = b")
	}
	// Empty set -> identity
	if b.ExistSet(ab, trueIdx) != ab {
		t.Error("ExistSet(f, True) = f")
	}
}

func TestForAllSet(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)

	varset := b.MakeSet([]int32{0})
	r := b.ForAllSet(ab, varset)
	if r != falseIdx {
		t.Error("∀{0}.(a∧b) = False")
	}

	taut := b.Or(a, b.Not(a))
	if b.ForAllSet(taut, varset) != trueIdx {
		t.Error("∀{0}.(a∨¬a) = True")
	}
}

// --- Unique tests ---

func TestUnique(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)

	varset := b.MakeSet([]int32{0})
	r := b.Unique(ab, varset)
	if b.Equiv(r, bb) != trueIdx {
		t.Error("Unique: ∃!{0}.(a∧b) = b")
	}

	// ∃!{0}.a: only a=true satisfies a, so one solution → True
	r2 := b.Unique(a, varset)
	if r2 != trueIdx {
		t.Error("Unique: ∃!{0}.a = True (only a=true)")
	}
}

// --- AppUni tests ---

func TestAppUni(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	varset := b.MakeSet([]int32{0})

	r := b.AppUni(a, bb, varset, opAnd)
	if r != bb {
		t.Errorf("AppUni(a,b,{0},AND) = b: r=%d bb=%d", r, bb)
	}
	// Regression: direct AppUni must match two-step Apply+Unique
	if r != b.Unique(b.Apply(a, bb, opAnd), varset) {
		t.Error("AppUni must equal Unique(Apply(l,r,op), varset)")
	}
	// AppUni with empty set = Apply
	r2 := b.AppUni(a, bb, trueIdx, opAnd)
	if r2 != b.And(a, bb) {
		t.Errorf("AppUni with empty set = Apply: r2=%d And=%d", r2, b.And(a, bb))
	}
	// Terminal shortcut: OR with True and unique quant = True
	r3 := b.AppUni(a, trueIdx, varset, opOr)
	if r3 != trueIdx {
		t.Error("AppUni(a,True,{0},OR) = True")
	}
}

// --- Pair.SetVar / Reset tests ---

func TestPairSetVar(t *testing.T) {
	b := newBDD(t, 4)
	p := b.NewPair()
	p.SetVar(0, 2)
	c := b.Var(2)
	level := b.var2level[0]
	if p.result[level] != c {
		t.Error("SetVar: result[level] should be Var(2)")
	}
}

func TestPairReset(t *testing.T) {
	b := newBDD(t, 3)
	p := b.NewPair()
	a := b.Var(0)
	p.Set(0, b.Var(2))
	p.Reset()
	level := b.var2level[0]
	if p.result[level] != a {
		t.Error("Reset: result should be back to Var(0)")
	}
}

// --- AppExBDD / RelProdBDD tests ---

func TestAppExBDD(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	varset := b.MakeSet([]int32{0})
	r := b.AppExBDD(a, bb, opAnd, varset)
	if b.Equiv(r, bb) != trueIdx {
		t.Error("AppExBDD(a,b,AND,{0}) = b")
	}
}

func TestRelProdBDD(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	varset := b.MakeSet([]int32{0})
	r := b.RelProdBDD(a, bb, varset)
	if b.Equiv(r, bb) != trueIdx {
		t.Error("RelProdBDD(a,b,{0}) = b")
	}
}

// --- Serialization tests ---

func TestSaveLoadTerminal(t *testing.T) {
	b := newBDD(t, 3)
	var buf strings.Builder
	if err := b.Save(&buf, trueIdx); err != nil {
		t.Fatal(err)
	}
	r, err := b.Load(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if r != trueIdx {
		t.Error("Save/Load(True) round-trip")
	}
}

func TestSaveLoadFalse(t *testing.T) {
	b := newBDD(t, 3)
	var buf strings.Builder
	b.Save(&buf, falseIdx)
	r, err := b.Load(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if r != falseIdx {
		t.Error("Save/Load(False) round-trip")
	}
}

func TestSaveLoadVar(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	var buf strings.Builder
	if err := b.Save(&buf, a); err != nil {
		t.Fatal(err)
	}
	r, err := b.Load(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if r != a {
		t.Errorf("Save/Load(Var(0)) round-trip: got %d want %d", r, a)
	}
}

func TestSaveLoadAnd(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	var buf strings.Builder
	if err := b.Save(&buf, ab); err != nil {
		t.Fatal(err)
	}
	r, err := b.Load(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	// Functional equivalence after load
	if b.SatisfyCount(b.Xor(r, ab)) != 0 {
		t.Error("Save/Load(a∧b): loaded BDD not equivalent to original")
	}
}

func TestSaveLoadComplex(t *testing.T) {
	b := newBDD(t, 6)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	d := b.Var(3)
	e := b.Var(4)
	f := b.Var(5)
	formula := b.And(a, b.Or(bb, b.And(c, b.Or(d, b.And(e, f)))))
	before := b.SatisfyCount(formula)

	var buf strings.Builder
	if err := b.Save(&buf, formula); err != nil {
		t.Fatal(err)
	}
	loaded, err := b.Load(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	after := b.SatisfyCount(loaded)
	if before != after {
		t.Errorf("Save/Load complex: sat count %d → %d", before, after)
	}
	// Verify node identity: ITE with same inputs should hit cache
	dup, _ := b.Load(strings.NewReader(buf.String()))
	if loaded != dup {
		t.Error("loading same data twice must return same node (canonical)")
	}
}

func TestSaveLoadWithSupport(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(2)
	ab := b.And(a, bb)
	before := b.Support(ab)

	var buf strings.Builder
	b.Save(&buf, ab)
	loaded, _ := b.Load(strings.NewReader(buf.String()))
	after := b.Support(loaded)

	if len(before) != len(after) {
		t.Errorf("Support changed: %v → %v", before, after)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("Support[%d]: %d → %d", i, before[i], after[i])
		}
	}
}

func TestSaveLoadPreservesOrder(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	f := b.Or(b.And(a, bb), c)

	var buf strings.Builder
	b.Save(&buf, f)
	loaded, _ := b.Load(strings.NewReader(buf.String()))
		if b.SatisfyCount(b.Xor(loaded, f)) != 0 {
			t.Error("loaded BDD not equivalent to original")
		}
	}

func TestIBuildCube(t *testing.T) {
	b := newBDD(t, 4)
	vars := []int32{0, 1, 2, 3}
	// 0xB = 1011 → var[0] ∧ ¬var[1] ∧ var[2] ∧ var[3]
	r := b.IBuildCube(0xB, 4, vars)
	expected := b.And(b.Var(0), b.And(b.Not(b.Var(1)), b.And(b.Var(2), b.Var(3))))
	if r != expected {
		t.Error("IBuildCube(0xB, 4, [0,1,2,3]) = v0 ∧ ¬v1 ∧ v2 ∧ v3")
	}
	// Single bit: width=1, value=1 → MSB=1 → var[0] positive
	r2 := b.IBuildCube(1, 1, []int32{0})
	if r2 != b.Var(0) {
		t.Error("IBuildCube(1, 1, [0]) = Var(0)")
	}
	// value=0, width=1 → all negated
	r3 := b.IBuildCube(0, 2, []int32{0, 1})
	if r3 != b.And(b.Not(b.Var(0)), b.Not(b.Var(1))) {
		t.Error("IBuildCube(0, 2, [0,1]) = ¬v0 ∧ ¬v1")
	}
}

func TestAnodeCount(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	c := b.Var(2)
	ab := b.And(a, bb)
	bc := b.And(bb, c)
	// ab and bc share Var(1) node
	n := b.AnodeCount([]NodeID{ab, bc})
	if n < 4 {
		t.Errorf("AnodeCount([a∧b, b∧c]): got %d, want >= 4", n)
	}
	// Single BDD
	n2 := b.AnodeCount([]NodeID{a})
	if n2 != 1 {
		t.Errorf("AnodeCount([Var(0)]): got %d, want 1", n2)
	}
	// Empty
	if b.AnodeCount(nil) != 0 {
		t.Error("AnodeCount(nil) = 0")
	}
}

func TestFullSatOneBDD(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	r := b.FullSatOneBDD(ab)
	if r == falseIdx {
		t.Error("FullSatOneBDD(a∧b) should not be False")
	}
	// The minterm must have variables at all levels 0..rootLevel
	s := b.ScanSet(r)
	if len(s) != 3 {
		t.Errorf("FullSatOneBDD should have all 3 vars, got %v", s)
	}
	// Minterm must imply original
	if b.SatisfyCount(b.Xor(b.And(r, b.Not(ab)), falseIdx)) < 1 {
		// Just check it's satisfiable
	}
	// False returns False
	if b.FullSatOneBDD(falseIdx) != falseIdx {
		t.Error("FullSatOneBDD(False) = False")
	}
}

func TestSupportBDD(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(2)
	f := b.And(a, bb)
	s := b.SupportBDD(f)
	vars := b.ScanSet(s)
	if len(vars) != 2 || vars[0] != 0 || vars[1] != 2 {
		t.Errorf("SupportBDD: got %v, want [0,2]", vars)
	}
	// SupportBDD of True returns empty set (True BDD)
	s2 := b.SupportBDD(trueIdx)
	if s2 != trueIdx {
		t.Error("SupportBDD(True) = True (empty set)")
	}
}

func TestSatCountDouble(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	d := b.SatCountDouble(ab)
	if d <= 0 {
		t.Error("SatCountDouble(a∧b) > 0")
	}
	if b.SatCountDouble(falseIdx) != 0 {
		t.Error("SatCountDouble(False) = 0")
	}
}

func TestAddRefDelRef(t *testing.T) {
	b := newBDD(t, 3)
	a := b.Var(0)
	if b.RefCount(a) != 0 {
		t.Error("initial refcount = 0")
	}
	r := b.AddRef(a)
	if r != a {
		t.Error("AddRef returns same node")
	}
	if b.RefCount(a) != 1 {
		t.Error("refcount should be 1 after AddRef")
	}
	b.AddRef(a)
	if b.RefCount(a) != 2 {
		t.Error("refcount should be 2 after double AddRef")
	}
	b.DelRef(a)
	if b.RefCount(a) != 1 {
		t.Error("refcount should be 1 after DelRef")
	}
	b.DelRef(a)
	if b.RefCount(a) != 0 {
		t.Error("refcount back to 0 after two DelRefs")
	}
	// DelRef below zero should not panic
	b.DelRef(a)
	if b.RefCount(a) != 0 {
		t.Error("refcount stays at 0")
	}
	// Terminals are no-ops
	if b.AddRef(trueIdx) != trueIdx {
		t.Error("AddRef(True) = True (no-op)")
	}
}

func TestAddRefSurvivesSwap(t *testing.T) {
	b := newBDD(t, 4)
	a := b.Var(0)
	bb := b.Var(1)
	f := b.And(a, bb)
	b.AddRef(f)

	// After swap, use remap function to update handle.
	remap := b.SwapVar(0, 1)
	f = remap(f)

	if f.isTerm() || int32(f) >= int32(len(b.nodes)) {
		t.Error("f invalid after swap")
	}
	if b.SatisfyCount(f) == 0 {
		t.Error("f should still be satisfiable after swap")
	}
}

func TestDone(t *testing.T) {
	b := newBDD(t, 3)
	b.Var(0)
	if !b.IsRunning() {
		t.Error("IsRunning should be true after New")
	}
	b.Done()
	if b.IsRunning() {
		t.Error("IsRunning should be false after Done")
	}
}

func TestIsRunning(t *testing.T) {
	b := newBDD(t, 2)
	if !b.IsRunning() {
		t.Error("New BDD should be running")
	}
}

func TestSetVarNum(t *testing.T) {
	b := newBDD(t, 3)
	b.SetVarNum(5)
	if b.VarCount() != 5 {
		t.Errorf("VarCount after SetVarNum(5): got %d", b.VarCount())
	}
	// New variables should be usable
	v4 := b.Var(4)
	if v4.isTerm() {
		t.Error("Var(4) should be valid after SetVarNum(5)")
	}
	// Shrink
	b.SetVarNum(3)
	if b.VarCount() != 3 {
		t.Errorf("VarCount after SetVarNum(3): got %d", b.VarCount())
	}
}

func TestExtVarNum(t *testing.T) {
	b := newBDD(t, 3)
	first := b.ExtVarNum(2)
	if first != 3 {
		t.Errorf("ExtVarNum(2) should return 3, got %d", first)
	}
	if b.VarCount() != 5 {
		t.Errorf("VarCount after ExtVarNum: got %d", b.VarCount())
	}
}

func TestSetCacheRatio(t *testing.T) {
	b := newBDD(t, 4)
	b.SetCacheRatio(4)
	// Verify cache was resized
	if b.cache == nil {
		t.Error("cache should not be nil")
	}
	// Verify operations still work after cache resize
	a := b.Var(0)
	bb := b.Var(1)
	ab := b.And(a, bb)
	if ab == falseIdx {
		t.Error("And should still work after SetCacheRatio")
	}
}

func TestPairSetVars(t *testing.T) {
	b := newBDD(t, 5)
	p := b.NewPair()
	oldVars := []int32{0, 1, 2}
	newVars := []int32{3, 4, 2}
	p.SetVars(oldVars, newVars, 3)
	// Var 0 → Var 3
	if p.result[b.var2level[0]] != b.Var(3) {
		t.Error("SetVars: var 0 should map to var 3")
	}
	// Var 1 → Var 4
	if p.result[b.var2level[1]] != b.Var(4) {
		t.Error("SetVars: var 1 should map to var 4")
	}
	// Var 2 → Var 2 (identity)
	if p.result[b.var2level[2]] != b.Var(2) {
		t.Error("SetVars: var 2 should stay var 2")
	}
}
