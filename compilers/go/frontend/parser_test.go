package frontend

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test: Parse a basic P2PKH contract from TypeScript source
// ---------------------------------------------------------------------------

func TestParse_P2PKH(t *testing.T) {
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
	result := ParseSource([]byte(source), "P2PKH.runar.ts")
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %s", strings.Join(result.Errors, "; "))
	}
	if result.Contract == nil {
		t.Fatal("expected non-nil contract")
	}

	c := result.Contract
	if c.Name != "P2PKH" {
		t.Errorf("expected contract name P2PKH, got %s", c.Name)
	}
	if c.ParentClass != "SmartContract" {
		t.Errorf("expected parentClass SmartContract, got %s", c.ParentClass)
	}
	if len(c.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(c.Properties))
	}
	if c.Properties[0].Name != "pubKeyHash" {
		t.Errorf("expected property name pubKeyHash, got %s", c.Properties[0].Name)
	}
	if !c.Properties[0].Readonly {
		t.Error("expected pubKeyHash to be readonly")
	}
	if len(c.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(c.Methods))
	}
	if c.Methods[0].Name != "unlock" {
		t.Errorf("expected method name unlock, got %s", c.Methods[0].Name)
	}
	if c.Methods[0].Visibility != "public" {
		t.Errorf("expected method visibility public, got %s", c.Methods[0].Visibility)
	}
	if len(c.Methods[0].Params) != 2 {
		t.Errorf("expected 2 params on unlock, got %d", len(c.Methods[0].Params))
	}
}

// ---------------------------------------------------------------------------
// Test: Parse a stateful Counter contract
// ---------------------------------------------------------------------------

func TestParse_StatefulCounter(t *testing.T) {
	source := `
import { StatefulSmartContract } from 'runar-lang';

class Counter extends StatefulSmartContract {
  count: bigint;

  constructor(count: bigint) {
    super(count);
    this.count = count;
  }

  public increment(): void {
    this.count = this.count + 1n;
  }
}
`
	result := ParseSource([]byte(source), "Counter.runar.ts")
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %s", strings.Join(result.Errors, "; "))
	}
	if result.Contract == nil {
		t.Fatal("expected non-nil contract")
	}

	c := result.Contract
	if c.Name != "Counter" {
		t.Errorf("expected contract name Counter, got %s", c.Name)
	}
	if c.ParentClass != "StatefulSmartContract" {
		t.Errorf("expected parentClass StatefulSmartContract, got %s", c.ParentClass)
	}
	if len(c.Properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(c.Properties))
	}
	if c.Properties[0].Readonly {
		t.Error("count should not be readonly in a stateful contract")
	}
	if len(c.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(c.Methods))
	}
	if c.Methods[0].Name != "increment" {
		t.Errorf("expected method name increment, got %s", c.Methods[0].Name)
	}
}

// ---------------------------------------------------------------------------
// Test: Parse dispatches to correct parser based on file extension
// ---------------------------------------------------------------------------

func TestParseSource_DispatchesByExtension(t *testing.T) {
	// A TS source should work with .runar.ts extension
	tsSource := `
import { SmartContract, assert } from 'runar-lang';
class Minimal extends SmartContract {
  constructor() { super(); }
  public check(x: bigint): void { assert(x === 1n); }
}
`
	result := ParseSource([]byte(tsSource), "Minimal.runar.ts")
	if result.Contract == nil && len(result.Errors) == 0 {
		t.Error("expected either a contract or errors from TS parse")
	}
}

// ---------------------------------------------------------------------------
// Test: Parse with no SmartContract class produces error
// ---------------------------------------------------------------------------

func TestParse_NoContract_Error(t *testing.T) {
	source := `
class NotAContract {
  x: number;
}
`
	result := ParseSource([]byte(source), "bad.runar.ts")
	if result.Contract != nil {
		t.Error("expected nil contract for non-SmartContract class")
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors when no SmartContract found")
	}
}

// ---------------------------------------------------------------------------
// Test: Parse contract with multiple methods
// ---------------------------------------------------------------------------

func TestParse_MultipleMethods(t *testing.T) {
	source := `
import { SmartContract, assert } from 'runar-lang';

class Multi extends SmartContract {
  readonly x: bigint;

  constructor(x: bigint) {
    super(x);
    this.x = x;
  }

  public method1(a: bigint): void {
    assert(a === this.x);
  }

  public method2(b: bigint): void {
    assert(b === this.x);
  }

  private helper(c: bigint): bigint {
    return c + 1n;
  }
}
`
	result := ParseSource([]byte(source), "Multi.runar.ts")
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %s", strings.Join(result.Errors, "; "))
	}

	c := result.Contract
	if c == nil {
		t.Fatal("expected non-nil contract")
	}
	if len(c.Methods) != 3 {
		t.Fatalf("expected 3 methods, got %d", len(c.Methods))
	}

	publicCount := 0
	privateCount := 0
	for _, m := range c.Methods {
		if m.Visibility == "public" {
			publicCount++
		} else {
			privateCount++
		}
	}
	if publicCount != 2 {
		t.Errorf("expected 2 public methods, got %d", publicCount)
	}
	if privateCount != 1 {
		t.Errorf("expected 1 private method, got %d", privateCount)
	}
}

// ---------------------------------------------------------------------------
// Test: Parse constructor parameters
// ---------------------------------------------------------------------------

func TestParse_ConstructorParams(t *testing.T) {
	source := `
import { SmartContract, assert, Addr, PubKey } from 'runar-lang';

class TwoProps extends SmartContract {
  readonly addr: Addr;
  readonly key: PubKey;

  constructor(addr: Addr, key: PubKey) {
    super(addr, key);
    this.addr = addr;
    this.key = key;
  }

  public check(x: bigint): void {
    assert(x === 1n);
  }
}
`
	result := ParseSource([]byte(source), "TwoProps.runar.ts")
	if len(result.Errors) > 0 {
		t.Fatalf("parse errors: %s", strings.Join(result.Errors, "; "))
	}

	c := result.Contract
	if len(c.Constructor.Params) != 2 {
		t.Fatalf("expected 2 constructor params, got %d", len(c.Constructor.Params))
	}
	if c.Constructor.Params[0].Name != "addr" {
		t.Errorf("expected first param name=addr, got %s", c.Constructor.Params[0].Name)
	}
	if c.Constructor.Params[1].Name != "key" {
		t.Errorf("expected second param name=key, got %s", c.Constructor.Params[1].Name)
	}
}
