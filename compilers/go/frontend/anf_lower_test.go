package frontend

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper: parse, validate, typecheck, and lower to ANF
// ---------------------------------------------------------------------------

func mustLowerToANF(t *testing.T, source string) (*ContractNode, []string) {
	t.Helper()

	result := ParseSource([]byte(source), "test.runar.ts")
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %s", strings.Join(result.Errors, "; "))
	}
	if result.Contract == nil {
		t.Fatal("parse returned nil contract")
	}

	valResult := Validate(result.Contract)
	if len(valResult.Errors) > 0 {
		t.Fatalf("validation errors: %s", strings.Join(valResult.Errors, "; "))
	}

	tcResult := TypeCheck(result.Contract)
	if len(tcResult.Errors) > 0 {
		t.Fatalf("type check errors: %s", strings.Join(tcResult.Errors, "; "))
	}

	return result.Contract, nil
}

// ---------------------------------------------------------------------------
// Test: P2PKH produces ANF with correct property
// ---------------------------------------------------------------------------

func TestANFLower_P2PKH_Property(t *testing.T) {
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
	contract, _ := mustLowerToANF(t, source)
	program := LowerToANF(contract)

	if program.ContractName != "P2PKH" {
		t.Errorf("expected contract name P2PKH, got %s", program.ContractName)
	}

	// Check properties
	if len(program.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(program.Properties))
	}
	prop := program.Properties[0]
	if prop.Name != "pubKeyHash" {
		t.Errorf("expected property name 'pubKeyHash', got '%s'", prop.Name)
	}
	if prop.Type != "Addr" {
		t.Errorf("expected property type 'Addr', got '%s'", prop.Type)
	}
	if !prop.Readonly {
		t.Error("expected property to be readonly")
	}
}

// ---------------------------------------------------------------------------
// Test: P2PKH unlock method produces expected ANF binding kinds
// ---------------------------------------------------------------------------

func TestANFLower_P2PKH_UnlockBindings(t *testing.T) {
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
	contract, _ := mustLowerToANF(t, source)
	program := LowerToANF(contract)

	// Find the unlock method (skip constructor)
	var unlockIdx int = -1
	for i, m := range program.Methods {
		if m.Name == "unlock" {
			unlockIdx = i
			break
		}
	}
	if unlockIdx == -1 {
		t.Fatal("could not find 'unlock' method in ANF output")
	}

	method := program.Methods[unlockIdx]

	// Verify the method is public
	if !method.IsPublic {
		t.Error("expected unlock method to be public")
	}

	// Verify parameters
	if len(method.Params) != 2 {
		t.Fatalf("expected 2 params (sig, pubKey), got %d", len(method.Params))
	}
	if method.Params[0].Name != "sig" || method.Params[0].Type != "Sig" {
		t.Errorf("expected first param 'sig: Sig', got '%s: %s'", method.Params[0].Name, method.Params[0].Type)
	}
	if method.Params[1].Name != "pubKey" || method.Params[1].Type != "PubKey" {
		t.Errorf("expected second param 'pubKey: PubKey', got '%s: %s'", method.Params[1].Name, method.Params[1].Type)
	}

	// Verify the expected ANF binding kind sequence:
	// The P2PKH unlock method should produce something like:
	//   load_param (pubKey), call hash160, load_prop (pubKeyHash),
	//   bin_op ===, assert, load_param (sig), load_param (pubKey), call checkSig, assert
	//
	// The exact order may vary by implementation, but we should see these kinds.
	expectedKinds := map[string]int{
		"load_param": 0, // at least 2 (sig, pubKey — pubKey may appear twice)
		"call":       0, // at least 2 (hash160, checkSig)
		"load_prop":  0, // at least 1 (pubKeyHash)
		"bin_op":     0, // at least 1 (===)
		"assert":     0, // at least 2
	}

	for _, b := range method.Body {
		if _, ok := expectedKinds[b.Value.Kind]; ok {
			expectedKinds[b.Value.Kind]++
		}
	}

	if expectedKinds["load_param"] < 2 {
		t.Errorf("expected at least 2 load_param bindings, got %d", expectedKinds["load_param"])
	}
	if expectedKinds["call"] < 2 {
		t.Errorf("expected at least 2 call bindings (hash160, checkSig), got %d", expectedKinds["call"])
	}
	if expectedKinds["load_prop"] < 1 {
		t.Errorf("expected at least 1 load_prop binding (pubKeyHash), got %d", expectedKinds["load_prop"])
	}
	if expectedKinds["bin_op"] < 1 {
		t.Errorf("expected at least 1 bin_op binding (===), got %d", expectedKinds["bin_op"])
	}
	if expectedKinds["assert"] < 2 {
		t.Errorf("expected at least 2 assert bindings, got %d", expectedKinds["assert"])
	}

	// Also log all binding kinds for debugging
	var kinds []string
	for _, b := range method.Body {
		detail := b.Value.Kind
		switch b.Value.Kind {
		case "load_param":
			detail += "(" + b.Value.Name + ")"
		case "load_prop":
			detail += "(" + b.Value.Name + ")"
		case "call":
			detail += "(" + b.Value.Func + ")"
		case "bin_op":
			detail += "(" + b.Value.Op + ")"
		}
		kinds = append(kinds, detail)
	}
	t.Logf("unlock ANF bindings: %s", strings.Join(kinds, " -> "))
}

// ---------------------------------------------------------------------------
// Test: P2PKH unlock specific binding details
// ---------------------------------------------------------------------------

func TestANFLower_P2PKH_BindingDetails(t *testing.T) {
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
	contract, _ := mustLowerToANF(t, source)
	program := LowerToANF(contract)

	var unlockIdx int = -1
	for i, m := range program.Methods {
		if m.Name == "unlock" {
			unlockIdx = i
			break
		}
	}
	if unlockIdx == -1 {
		t.Fatal("could not find 'unlock' method")
	}

	method := program.Methods[unlockIdx]

	// Check that we have a call to hash160
	foundHash160 := false
	for _, b := range method.Body {
		if b.Value.Kind == "call" && b.Value.Func == "hash160" {
			foundHash160 = true
			if len(b.Value.Args) != 1 {
				t.Errorf("hash160 should have 1 arg, got %d", len(b.Value.Args))
			}
			break
		}
	}
	if !foundHash160 {
		t.Error("expected a call to hash160 in unlock method bindings")
	}

	// Check that we have a call to checkSig
	foundCheckSig := false
	for _, b := range method.Body {
		if b.Value.Kind == "call" && b.Value.Func == "checkSig" {
			foundCheckSig = true
			if len(b.Value.Args) != 2 {
				t.Errorf("checkSig should have 2 args, got %d", len(b.Value.Args))
			}
			break
		}
	}
	if !foundCheckSig {
		t.Error("expected a call to checkSig in unlock method bindings")
	}

	// Check that we have a bin_op === with result_type "bytes" (because
	// hash160 returns a byte type and pubKeyHash is Addr, also a byte type)
	foundEqOp := false
	for _, b := range method.Body {
		if b.Value.Kind == "bin_op" && b.Value.Op == "===" {
			foundEqOp = true
			if b.Value.ResultType != "bytes" {
				t.Errorf("expected bin_op === to have ResultType='bytes' (byte-typed equality), got '%s'", b.Value.ResultType)
			}
			break
		}
	}
	if !foundEqOp {
		t.Error("expected a bin_op === in unlock method bindings")
	}
}

// ---------------------------------------------------------------------------
// Test: Constructor is lowered as a method
// ---------------------------------------------------------------------------

func TestANFLower_ConstructorIncluded(t *testing.T) {
	source := `
import { SmartContract, assert } from 'runar-lang';

class Simple extends SmartContract {
  readonly x: bigint;

  constructor(x: bigint) {
    super(x);
    this.x = x;
  }

  public check(val: bigint): void {
    assert(val === this.x);
  }
}
`
	contract, _ := mustLowerToANF(t, source)
	program := LowerToANF(contract)

	// The constructor should appear as the first method in the ANF output
	if len(program.Methods) < 2 {
		t.Fatalf("expected at least 2 methods (constructor + check), got %d", len(program.Methods))
	}

	ctor := program.Methods[0]
	if ctor.Name != "constructor" {
		t.Errorf("expected first method to be 'constructor', got '%s'", ctor.Name)
	}
	if ctor.IsPublic {
		t.Error("constructor should not be public")
	}
}

// ---------------------------------------------------------------------------
// Test: Arithmetic produces correct ANF
// ---------------------------------------------------------------------------

func TestANFLower_Arithmetic(t *testing.T) {
	source := `
import { SmartContract, assert } from 'runar-lang';

class ArithTest extends SmartContract {
  readonly target: bigint;

  constructor(target: bigint) {
    super(target);
    this.target = target;
  }

  public verify(a: bigint, b: bigint): void {
    assert(a + b === this.target);
  }
}
`
	contract, _ := mustLowerToANF(t, source)
	program := LowerToANF(contract)

	// Find the verify method
	var verifyIdx int = -1
	for i, m := range program.Methods {
		if m.Name == "verify" {
			verifyIdx = i
			break
		}
	}
	if verifyIdx == -1 {
		t.Fatal("could not find 'verify' method")
	}

	method := program.Methods[verifyIdx]

	// Should have a bin_op + for a + b
	foundAdd := false
	for _, b := range method.Body {
		if b.Value.Kind == "bin_op" && b.Value.Op == "+" {
			foundAdd = true
			break
		}
	}
	if !foundAdd {
		t.Error("expected bin_op + in verify method for 'a + b'")
	}

	// Should have a bin_op === for equality check
	foundEq := false
	for _, b := range method.Body {
		if b.Value.Kind == "bin_op" && b.Value.Op == "===" {
			foundEq = true
			break
		}
	}
	if !foundEq {
		t.Error("expected bin_op === in verify method")
	}
}
