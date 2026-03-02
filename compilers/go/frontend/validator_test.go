package frontend

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper: parse TypeScript source and return the ContractNode
// ---------------------------------------------------------------------------

func mustParseTS(t *testing.T, source string) *ContractNode {
	t.Helper()
	result := ParseSource([]byte(source), "test.runar.ts")
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %s", strings.Join(result.Errors, "; "))
	}
	if result.Contract == nil {
		t.Fatal("parse returned nil contract")
	}
	return result.Contract
}

// ---------------------------------------------------------------------------
// Test: Valid P2PKH contract passes validation
// ---------------------------------------------------------------------------

func TestValidate_ValidP2PKH(t *testing.T) {
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
	result := Validate(contract)

	if len(result.Errors) > 0 {
		t.Errorf("expected no validation errors, got: %s", strings.Join(result.Errors, "; "))
	}
}

// ---------------------------------------------------------------------------
// Test: Constructor missing super() call produces error
// ---------------------------------------------------------------------------

func TestValidate_ConstructorMissingSuperCall(t *testing.T) {
	// Build a ContractNode manually with a constructor that doesn't start with super()
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
				// Missing super() — jump straight to assignment
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
					ExpressionStmt{
						Expr: CallExpr{
							Callee: Identifier{Name: "assert"},
							Args: []Expression{
								BinaryExpr{
									Op:    "===",
									Left:  Identifier{Name: "val"},
									Right: PropertyAccessExpr{Property: "x"},
								},
							},
						},
					},
				},
			},
		},
	}

	result := Validate(contract)

	foundSuperError := false
	for _, e := range result.Errors {
		if strings.Contains(e, "super()") {
			foundSuperError = true
			break
		}
	}
	if !foundSuperError {
		t.Errorf("expected validation error about missing super() call, got errors: %v", result.Errors)
	}
}

// ---------------------------------------------------------------------------
// Test: Public method not ending with assert produces error
// ---------------------------------------------------------------------------

func TestValidate_PublicMethodMissingFinalAssert(t *testing.T) {
	contract := &ContractNode{
		Name:        "NoAssert",
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
					// Does NOT end with assert — just a bare expression
					ExpressionStmt{
						Expr: BinaryExpr{
							Op:    "+",
							Left:  Identifier{Name: "val"},
							Right: BigIntLiteral{Value: 1},
						},
					},
				},
			},
		},
	}

	result := Validate(contract)

	foundAssertError := false
	for _, e := range result.Errors {
		if strings.Contains(e, "assert()") {
			foundAssertError = true
			break
		}
	}
	if !foundAssertError {
		t.Errorf("expected validation error about public method not ending with assert(), got errors: %v", result.Errors)
	}
}

// ---------------------------------------------------------------------------
// Test: Direct recursion is detected
// ---------------------------------------------------------------------------

func TestValidate_DirectRecursion(t *testing.T) {
	contract := &ContractNode{
		Name:        "Recursive",
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
				Name:       "recurse",
				Visibility: "public",
				Params: []ParamNode{
					{Name: "n", Type: PrimitiveType{Name: "bigint"}},
				},
				Body: []Statement{
					// this.recurse(n - 1) — direct self-call
					ExpressionStmt{
						Expr: CallExpr{
							Callee: PropertyAccessExpr{Property: "recurse"},
							Args: []Expression{
								BinaryExpr{
									Op:    "-",
									Left:  Identifier{Name: "n"},
									Right: BigIntLiteral{Value: 1},
								},
							},
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

	result := Validate(contract)

	foundRecursionError := false
	for _, e := range result.Errors {
		if strings.Contains(e, "recursion") {
			foundRecursionError = true
			break
		}
	}
	if !foundRecursionError {
		t.Errorf("expected validation error about recursion, got errors: %v", result.Errors)
	}
}

// ---------------------------------------------------------------------------
// Test: Valid P2PKH parsed from source passes validation (integration)
// ---------------------------------------------------------------------------

func TestValidate_P2PKHFromSource(t *testing.T) {
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
	result := Validate(contract)

	if len(result.Errors) > 0 {
		t.Errorf("P2PKH should validate without errors, got: %s", strings.Join(result.Errors, "; "))
	}
	if len(result.Warnings) > 0 {
		t.Logf("validation warnings: %s", strings.Join(result.Warnings, "; "))
	}
}

// ---------------------------------------------------------------------------
// Test: StatefulSmartContract public method without trailing assert is OK
// (the compiler auto-injects it)
// ---------------------------------------------------------------------------

func TestValidate_StatefulNoFinalAssertOK(t *testing.T) {
	contract := &ContractNode{
		Name:        "Counter",
		ParentClass: "StatefulSmartContract",
		Properties: []PropertyNode{
			{Name: "count", Type: PrimitiveType{Name: "bigint"}, Readonly: false},
		},
		Constructor: MethodNode{
			Name: "constructor",
			Params: []ParamNode{
				{Name: "count", Type: PrimitiveType{Name: "bigint"}},
			},
			Body: []Statement{
				ExpressionStmt{
					Expr: CallExpr{
						Callee: Identifier{Name: "super"},
						Args:   []Expression{Identifier{Name: "count"}},
					},
				},
				AssignmentStmt{
					Target: PropertyAccessExpr{Property: "count"},
					Value:  Identifier{Name: "count"},
				},
			},
		},
		Methods: []MethodNode{
			{
				Name:       "increment",
				Visibility: "public",
				Params:     []ParamNode{},
				Body: []Statement{
					// this.count = this.count + 1 — no trailing assert
					AssignmentStmt{
						Target: PropertyAccessExpr{Property: "count"},
						Value: BinaryExpr{
							Op:    "+",
							Left:  PropertyAccessExpr{Property: "count"},
							Right: BigIntLiteral{Value: 1},
						},
					},
				},
			},
		},
	}

	result := Validate(contract)

	// StatefulSmartContract methods should NOT require a trailing assert
	for _, e := range result.Errors {
		if strings.Contains(e, "must end with an assert()") {
			t.Errorf("StatefulSmartContract public method should not require trailing assert, got error: %s", e)
		}
	}
}
