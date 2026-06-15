package gobdd

// NodeID is a handle to a BDD node. Terminal nodes are falseIdx (0) and trueIdx (1).
// Non-terminal nodes have indices >= 2.
type NodeID int32

// OpCode is a binary operator code for Apply, AppEx, AppAll, AppUni, and RelProd.
type OpCode int32

const (
	falseIdx NodeID = 0
	trueIdx  NodeID = 1
)

// Operator constants for Apply.
const (
	opAnd       OpCode = 0
	opXor       OpCode = 1
	opOr        OpCode = 2
	opNand      OpCode = 3
	opNor       OpCode = 4
	opImp       OpCode = 5
	opBiimp     OpCode = 6
	opDiff      OpCode = 7
	opLess      OpCode = 8
	opInvImp    OpCode = 9
	opConstrain OpCode = 100
	opSimplify  OpCode = 101
)

// isTerm reports whether n is a terminal node (True or False).
func (n NodeID) isTerm() bool { return int32(n) < 2 }
