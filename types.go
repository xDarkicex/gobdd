package gobdd

// NodeID is a handle to a BDD node. Terminal nodes are False (0) and True (1).
// Non-terminal nodes have indices >= 2.
type NodeID int32

// OpCode is a binary operator code for Apply, AppEx, AppAll, AppUni, and RelProd.
type OpCode int32

// Terminal node constants.
const (
	False NodeID = 0
	True  NodeID = 1

	falseIdx = False
	trueIdx  = True
)

// Operator constants for Apply.
const (
	OpAnd       OpCode = 0
	OpXor       OpCode = 1
	OpOr        OpCode = 2
	OpNand      OpCode = 3
	OpNor       OpCode = 4
	OpImp       OpCode = 5
	OpBiimp     OpCode = 6
	OpDiff      OpCode = 7
	OpLess      OpCode = 8
	OpInvImp    OpCode = 9
	OpConstrain OpCode = 100
	OpSimplify  OpCode = 101

	opAnd       = OpAnd
	opXor       = OpXor
	opOr        = OpOr
	opNand      = OpNand
	opNor       = OpNor
	opImp       = OpImp
	opBiimp     = OpBiimp
	opDiff      = OpDiff
	opLess      = OpLess
	opInvImp    = OpInvImp
	opConstrain = OpConstrain
	opSimplify  = OpSimplify
)

// isTerm reports whether n is a terminal node (True or False).
func (n NodeID) isTerm() bool { return int32(n) < 2 }
