package frontend

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test: Valid P2PKH passes type check
// ---------------------------------------------------------------------------

func TestTypeCheck_ValidP2PKH(t *testing.T) {
	source := `
import { SmartContract, assert, PubKey, Sig, Addr, hash160, checkSig } from 'runar-lang';

class P2PKH extends SmartContract {
  readonly pubKeyHash: Addr;

  constructor(pubKeyHash: Addr) {
    super(pubKeyHash);
    this.pubKeyHash = pubKeyHash;
  }

  public unlock(sig: Sig, pubKey: PubKey): void {
    assert(hash160(pubKey) === this.pubKeyHash);
    assert(checkSig(sig, pubKey));
  }
}
`
	contract := mustParseTS(t, source)

	// Validate first (prerequisite for type check)
	valResult := Validate(contract)
	if len(valResult.Errors) > 0 {
		t.Fatalf("validation failed: %s", strings.Join(valResult.Errors, "; "))
	}

	tcResult := TypeCheck(contract)
	if len(tcResult.Errors) > 0 {
		t.Errorf("expected no type check errors for P2PKH, got: %s", strings.Join(tcResult.Errors, "; "))
	}
	if tcResult.Contract == nil {
		t.Error("expected non-nil contract in type check result")
	}
}

// ---------------------------------------------------------------------------
// Test: Unknown function call (Math.floor) produces error
// ---------------------------------------------------------------------------

func TestTypeCheck_UnknownFunction_MathFloor(t *testing.T) {
	// Build an AST that calls Math.floor — this should be rejected
	contract := &ContractNode{
		Name:        "Bad",
		ParentClass: "SmartContract",
		Properties: []PropertyNode{
			{Name: "x", Type: PrimitiveType{Name: "bigint"}, Readonly: true},
		},
		Constructor: MethodNode{
			Name: "constructor",
			Params: []ParamNode{
				{Name: "x", Type: PrimitiveType{Name: "bigint"}},
			},
			Body: []Statement{
				ExpressionStmt{
					Expr: CallExpr{
						Callee: Identifier{Name: "super"},
						Args:   []Expression{Identifier{Name: "x"}},
					},
				},
				AssignmentStmt{
					Target: PropertyAccessExpr{Property: "x"},
					Value:  Identifier{Name: "x"},
				},
			},
		},
		Methods: []MethodNode{
			{
				Name:       "check",
				Visibility: "public",
				Params: []ParamNode{
					{Name: "val", Type: PrimitiveType{Name: "bigint"}},
				},
				Body: []Statement{
					// const result = Math.floor(val) — should be rejected
					VariableDeclStmt{
						Name: "result",
						Init: CallExpr{
							Callee: MemberExpr{
								Object:   Identifier{Name: "Math"},
								Property: "floor",
							},
							Args: []Expression{Identifier{Name: "val"}},
						},
					},
					ExpressionStmt{
						Expr: CallExpr{
							Callee: Identifier{Name: "assert"},
							Args: []Expression{
								BinaryExpr{
									Op:    "===",
									Left:  Identifier{Name: "result"},
									Right: PropertyAccessExpr{Property: "x"},
								},
							},
						},
					},
				},
			},
		},
	}

	tcResult := TypeCheck(contract)

	foundUnknownError := false
	for _, e := range tcResult.Errors {
		if strings.Contains(e, "unknown function") || strings.Contains(e, "Math.floor") {
			foundUnknownError = true
			break
		}
	}
	if !foundUnknownError {
		t.Errorf("expected type check error about unknown function 'Math.floor', got errors: %v", tcResult.Errors)
	}
}

// ---------------------------------------------------------------------------
// Test: Unknown function call (console.log) produces error
// ---------------------------------------------------------------------------

func TestTypeCheck_UnknownFunction_ConsoleLog(t *testing.T) {
	contract := &ContractNode{
		Name:        "Bad",
		ParentClass: "SmartContract",
		Properties:  []PropertyNode{},
		Constructor: MethodNode{
			Name:   "constructor",
			Params: []ParamNode{},
			Body: []Statement{
				ExpressionStmt{
					Expr: CallExpr{
						Callee: Identifier{Name: "super"},
						Args:   nil,
					},
				},
			},
		},
		Methods: []MethodNode{
			{
				Name:       "check",
				Visibility: "public",
				Params: []ParamNode{
					{Name: "val", Type: PrimitiveType{Name: "bigint"}},
				},
				Body: []Statement{
					// console.log(val)
					ExpressionStmt{
						Expr: CallExpr{
							Callee: MemberExpr{
								Object:   Identifier{Name: "console"},
								Property: "log",
							},
							Args: []Expression{Identifier{Name: "val"}},
						},
					},
					ExpressionStmt{
						Expr: CallExpr{
							Callee: Identifier{Name: "assert"},
							Args:   []Expression{BoolLiteral{Value: true}},
						},
					},
				},
			},
		},
	}

	tcResult := TypeCheck(contract)

	foundError := false
	for _, e := range tcResult.Errors {
		if strings.Contains(e, "unknown function") || strings.Contains(e, "console.log") {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("expected type check error about unknown function 'console.log', got errors: %v", tcResult.Errors)
	}
}

// ---------------------------------------------------------------------------
// Test: Type mismatch in binary arithmetic operator
// ---------------------------------------------------------------------------

func TestTypeCheck_TypeMismatch_ArithmeticOnBoolean(t *testing.T) {
	contract := &ContractNode{
		Name:        "Mismatch",
		ParentClass: "SmartContract",
		Properties:  []PropertyNode{},
		Constructor: MethodNode{
			Name:   "constructor",
			Params: []ParamNode{},
			Body: []Statement{
				ExpressionStmt{
					Expr: CallExpr{
						Callee: Identifier{Name: "super"},
						Args:   nil,
					},
				},
			},
		},
		Methods: []MethodNode{
			{
				Name:       "check",
				Visibility: "public",
				Params: []ParamNode{
					{Name: "flag", Type: PrimitiveType{Name: "boolean"}},
				},
				Body: []Statement{
					// const result = flag + 1n — boolean + bigint should error
					VariableDeclStmt{
						Name: "result",
						Init: BinaryExpr{
							Op:    "+",
							Left:  Identifier{Name: "flag"},
							Right: BigIntLiteral{Value: 1},
						},
					},
					ExpressionStmt{
						Expr: CallExpr{
							Callee: Identifier{Name: "assert"},
							Args: []Expression{
								BinaryExpr{
									Op:    "===",
									Left:  Identifier{Name: "result"},
									Right: BigIntLiteral{Value: 2},
								},
							},
						},
					},
				},
			},
		},
	}

	tcResult := TypeCheck(contract)

	foundTypeError := false
	for _, e := range tcResult.Errors {
		if strings.Contains(e, "must be bigint") || strings.Contains(e, "boolean") {
			foundTypeError = true
			break
		}
	}
	if !foundTypeError {
		t.Errorf("expected type check error about type mismatch (boolean used in arithmetic), got errors: %v", tcResult.Errors)
	}
}

// ---------------------------------------------------------------------------
// Test: Arithmetic contract passes type check
// ---------------------------------------------------------------------------

func TestTypeCheck_ValidArithmetic(t *testing.T) {
	source := `
import { SmartContract, assert } from 'runar-lang';

class Arithmetic extends SmartContract {
  readonly target: bigint;

  constructor(target: bigint) {
    super(target);
    this.target = target;
  }

  public verify(a: bigint, b: bigint): void {
    const sum: bigint = a + b;
    const diff: bigint = a - b;
    const prod: bigint = a * b;
    const quot: bigint = a / b;
    const result: bigint = sum + diff + prod + quot;
    assert(result === this.target);
  }
}
`
	contract := mustParseTS(t, source)

	valResult := Validate(contract)
	if len(valResult.Errors) > 0 {
		t.Fatalf("validation failed: %s", strings.Join(valResult.Errors, "; "))
	}

	tcResult := TypeCheck(contract)
	if len(tcResult.Errors) > 0 {
		t.Errorf("expected no type check errors for Arithmetic, got: %s", strings.Join(tcResult.Errors, "; "))
	}
}

// ---------------------------------------------------------------------------
// Test: Boolean logic contract passes type check
// ---------------------------------------------------------------------------

func TestTypeCheck_ValidBooleanLogic(t *testing.T) {
	source := `
import { SmartContract, assert } from 'runar-lang';

class BoolLogic extends SmartContract {
  readonly threshold: bigint;

  constructor(threshold: bigint) {
    super(threshold);
    this.threshold = threshold;
  }

  public verify(a: bigint, b: bigint, flag: boolean): void {
    const aAbove: boolean = a > this.threshold;
    const bAbove: boolean = b > this.threshold;
    const bothAbove: boolean = aAbove && bAbove;
    const eitherAbove: boolean = aAbove || bAbove;
    const notFlag: boolean = !flag;
    assert(bothAbove || (eitherAbove && notFlag));
  }
}
`
	contract := mustParseTS(t, source)

	valResult := Validate(contract)
	if len(valResult.Errors) > 0 {
		t.Fatalf("validation failed: %s", strings.Join(valResult.Errors, "; "))
	}

	tcResult := TypeCheck(contract)
	if len(tcResult.Errors) > 0 {
		t.Errorf("expected no type check errors for BoolLogic, got: %s", strings.Join(tcResult.Errors, "; "))
	}
}

// ---------------------------------------------------------------------------
// Test: Subtype compatibility (PubKey assignable to ByteString parameter)
// ---------------------------------------------------------------------------

func TestTypeCheck_SubtypeCompatibility(t *testing.T) {
	source := `
import { SmartContract, assert, PubKey, sha256 } from 'runar-lang';

class HashCheck extends SmartContract {
  readonly expectedHash: Sha256;

  constructor(expectedHash: Sha256) {
    super(expectedHash);
    this.expectedHash = expectedHash;
  }

  public verify(pubKey: PubKey): void {
    assert(sha256(pubKey) === this.expectedHash);
  }
}
`
	contract := mustParseTS(t, source)

	// PubKey should be assignable to ByteString (sha256's parameter type)
	tcResult := TypeCheck(contract)
	// Filter out errors that are NOT about subtype/argument type issues
	for _, e := range tcResult.Errors {
		if strings.Contains(e, "argument") && strings.Contains(e, "PubKey") {
			t.Errorf("PubKey should be assignable to ByteString, but got error: %s", e)
		}
	}
}
