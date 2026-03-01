# Integration Guide

This guide covers the full TSOP development pipeline: writing contracts, testing them, compiling to artifacts, and loading those artifacts for deployment and interaction from TypeScript, Go, or Rust.

---

## Pipeline Overview

```
 Source Code                     Artifact (JSON)                   On-Chain
 (.tsop.ts / .tsop.sol /        ┌──────────────────┐              ┌──────────────┐
  .tsop.move / .tsop.go /  ──►  │ version          │  ──►  deploy │ Locking UTXO │
  .tsop.rs)                     │ abi              │              │ (Bitcoin SV)  │
                                │ script (hex)     │  ◄──  call  └──────────────┘
       │                        │ asm              │
       │  test                  │ stateFields      │
       ▼                        └──────────────────┘
 vitest / go test /                  ▲
 cargo test                          │
                               tsop compile
                               compile() (TS)
                               CompileFromSource() (Go)
                               compile_from_source() (Rust)
```

The artifact JSON is the intermediary form that bridges compilation and deployment. It contains everything needed to construct locking scripts, build unlocking scripts, and interact with a deployed contract.

---

## Developing Contracts

TSOP contracts are classes that extend `SmartContract` (stateless) or `StatefulSmartContract` (stateful). You can write them in any of five syntax formats -- all compile to the same AST and produce identical Bitcoin Script.

| Extension | Syntax Style | Parser |
|-----------|-------------|--------|
| `.tsop.ts` | TypeScript (stable) | ts-morph |
| `.tsop.sol` | Solidity-like | Hand-written recursive descent |
| `.tsop.move` | Move-style | Hand-written recursive descent |
| `.tsop.go` | Go | `go/parser` stdlib |
| `.tsop.rs` | Rust macro DSL | Custom token parser |

A minimal stateless contract in TypeScript:

```typescript
import { SmartContract, assert, PubKey, Sig, Addr, hash160, checkSig } from 'tsop-lang';

class P2PKH extends SmartContract {
  readonly pubKeyHash: Addr;

  constructor(pubKeyHash: Addr) {
    super(pubKeyHash);
    this.pubKeyHash = pubKeyHash;
  }

  public unlock(sig: Sig, pubKey: PubKey) {
    assert(hash160(pubKey) === this.pubKeyHash);
    assert(checkSig(sig, pubKey));
  }
}
```

See the [Getting Started](./getting-started.md) guide for a detailed walkthrough and the [Language Reference](./language-reference.md) for the complete set of types, operators, and built-in functions.

---

## Testing Contracts

TSOP supports testing at multiple levels. The summary below covers the most common patterns; see the [Testing Guide](./testing-guide.md) for advanced techniques including property-based fuzzing, differential testing, and cross-compiler conformance.

### TypeScript (vitest)

`TestContract.fromSource()` compiles a contract and runs methods through the reference interpreter with mocked crypto (`checkSig` always returns true, hash functions compute real values).

```typescript
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { TestContract } from 'tsop-testing';

describe('Counter', () => {
  const source = readFileSync('Counter.tsop.ts', 'utf8');
  const counter = TestContract.fromSource(source, { count: 0n });

  it('increments', () => {
    counter.call('increment');
    expect(counter.state.count).toBe(1n);
  });
});
```

For multi-format sources, pass the filename so the parser can dispatch by extension:

```typescript
const solCounter = TestContract.fromSource(solSource, { count: 0n }, 'Counter.tsop.sol');
```

### Go (go test)

Go contracts run as native Go code using the `tsop` mock package. Add a `CompileCheck` test to verify the contract is valid TSOP.

```go
func TestCounter_Increment(t *testing.T) {
    c := &Counter{Count: 0}
    c.Increment()
    if c.Count != 1 {
        t.Errorf("expected 1, got %d", c.Count)
    }
}

func TestCounter_Compile(t *testing.T) {
    if err := tsop.CompileCheck("Counter.tsop.go"); err != nil {
        t.Fatalf("TSOP compile check failed: %v", err)
    }
}
```

Run with `go test ./...` from the contract directory.

### Rust (cargo test)

Rust contracts run as native Rust code using the `tsop::prelude` mock types. Use `#[should_panic]` for expected failures.

```rust
#[path = "Counter.tsop.rs"]
mod contract;
use contract::*;
use tsop::prelude::*;

#[test]
fn test_increment() {
    let mut c = Counter { count: 0 };
    c.increment();
    assert_eq!(c.count, 1);
}

#[test]
fn test_compile() {
    tsop::compile_check(
        include_str!("Counter.tsop.rs"),
        "Counter.tsop.rs",
    ).unwrap();
}
```

Run with `cargo test` from the examples/rust directory.

### Script VM Testing

For full script execution testing (compiled Bitcoin Script, not interpreter), use `TestSmartContract.fromArtifact()`:

```typescript
import { TestSmartContract, expectScriptSuccess } from 'tsop-testing';
import artifact from './artifacts/P2PKH.json';

const contract = TestSmartContract.fromArtifact(artifact, [pubKeyHash]);
const result = contract.call('unlock', [sig, pubKey]);
expectScriptSuccess(result);
```

See the [Testing Guide](./testing-guide.md) for the full API.

---

## Compiling to Artifacts

The artifact is a JSON file containing the compiled locking script, the contract ABI, and metadata. This is the form you should load when deploying or interacting with a contract.

### CLI

```bash
tsop compile Counter.tsop.ts                   # Outputs artifacts/Counter.json
tsop compile Counter.tsop.ts --output ./build  # Custom output directory
tsop compile Counter.tsop.ts --ir              # Include ANF IR in artifact
tsop compile Counter.tsop.ts --asm             # Print assembly to stdout
```

### Programmatic (TypeScript)

```typescript
import { compile } from 'tsop-compiler';
import type { CompileResult, TSOPArtifact } from 'tsop-compiler';

const source = readFileSync('Counter.tsop.ts', 'utf8');
const result: CompileResult = compile(source, { fileName: 'Counter.tsop.ts' });

if (!result.success) {
  const errors = result.diagnostics
    .filter(d => d.severity === 'error')
    .map(d => d.message);
  throw new Error(`Compilation failed: ${errors.join(', ')}`);
}

const artifact: TSOPArtifact = result.artifact!;
// Write to disk, load in SDK, etc.
```

### Programmatic (Go)

The Go compiler exposes the same pipeline as a library:

```go
import main // or your import path for the Go compiler package

// From source file
artifact, err := CompileFromSource("Counter.tsop.go")

// From ANF IR JSON (generated by any compiler)
artifact, err := CompileFromIR("Counter-anf.json")

// From raw IR bytes
artifact, err := CompileFromIRBytes(irJSON)

// Serialize to JSON
jsonBytes, err := ArtifactToJSON(artifact)
```

### Programmatic (Rust)

The Rust compiler provides equivalent functions:

```rust
use tsop_compiler_rust::{compile_from_source, compile_from_ir, compile_from_source_str};
use std::path::Path;

// From source file
let artifact = compile_from_source(Path::new("Counter.tsop.rs"))?;

// From source string
let artifact = compile_from_source_str(&source, Some("Counter.tsop.rs"))?;

// From ANF IR JSON file
let artifact = compile_from_ir(Path::new("Counter-anf.json"))?;

// From ANF IR string
let artifact = compile_from_ir_str(&ir_json)?;

// Serialize with serde
let json = serde_json::to_string_pretty(&artifact).unwrap();
```

### Artifact Schema

All three compilers produce artifacts with the same JSON schema:

```json
{
  "version": "tsop-v0.1.0",
  "compilerVersion": "0.1.0",
  "contractName": "Counter",
  "abi": {
    "constructor": {
      "params": [
        { "name": "count", "type": "bigint" }
      ]
    },
    "methods": [
      { "name": "increment", "params": [...], "isPublic": true },
      { "name": "decrement", "params": [...], "isPublic": true }
    ]
  },
  "script": "5179...",
  "asm": "OP_1 OP_PICK ...",
  "stateFields": [
    { "name": "count", "type": "bigint", "index": 0 }
  ],
  "buildTimestamp": "2026-03-02T12:00:00Z"
}
```

Key fields:

| Field | Description |
|-------|-------------|
| `script` | Hex-encoded Bitcoin Script locking script |
| `asm` | Human-readable opcode assembly |
| `abi` | Constructor parameters and public method signatures |
| `stateFields` | Mutable state fields (present only for `StatefulSmartContract`) |
| `ir` | Optional ANF/Stack IR snapshots (when compiled with `--ir`) |

---

## Loading Artifacts in TypeScript

The `tsop-sdk` package provides `TSOPContract` for deploying and interacting with compiled contracts.

### Setup

```bash
pnpm add tsop-sdk tsop-compiler tsop-lang
```

### Deploy and Call

```typescript
import { readFileSync } from 'node:fs';
import { compile } from 'tsop-compiler';
import { TSOPContract, LocalSigner } from 'tsop-sdk';
import type { TSOPArtifact } from 'tsop-ir-schema';

// 1. Compile (or load a pre-compiled artifact JSON)
const source = readFileSync('P2PKH.tsop.ts', 'utf8');
const result = compile(source, { fileName: 'P2PKH.tsop.ts' });
const artifact: TSOPArtifact = result.artifact!;

// Or load from a saved artifact file:
// const artifact = JSON.parse(readFileSync('artifacts/P2PKH.json', 'utf8'));

// 2. Instantiate with constructor arguments
const contract = new TSOPContract(artifact, [pubKeyHash]);

// 3. Get the locking script (hex)
const lockingScript = contract.getLockingScript();

// 4. Deploy to blockchain
const { txid } = await contract.deploy(provider, signer, { satoshis: 10_000 });

// 5. Call a method (spend the UTXO)
const result = await contract.call('unlock', [sig, pubKey], provider, signer);

// 6. Reconnect to an existing deployed contract
const existing = await TSOPContract.fromTxId(artifact, txid, 0, provider);
console.log(existing.state); // Extract state from the locking script
```

### State Management (Stateful Contracts)

```typescript
import { serializeState, deserializeState } from 'tsop-sdk';

// Serialize state to hex (for embedding in locking scripts)
const stateHex = serializeState(artifact.stateFields!, { count: 42n });

// Deserialize state from a locking script
const state = deserializeState(artifact.stateFields!, stateHex);
```

### Production Integration with @bsv/sdk

For production deployments, use `@bsv/sdk` directly with the artifact's locking script:

```typescript
import { PrivateKey, Transaction, P2PKH, ARC, LockingScript } from '@bsv/sdk';

// Load the compiled locking script into @bsv/sdk
const lockingScript = LockingScript.fromHex(contract.getLockingScript());

// Build a deployment transaction
const deployTx = new Transaction();
deployTx.addInput({
  sourceTransaction: fundingTx,
  sourceOutputIndex: 0,
  unlockingScriptTemplate: new P2PKH().unlock(privKey),
});
deployTx.addOutput({ lockingScript, satoshis: 10_000 });
deployTx.addOutput({ lockingScript: new P2PKH().lock(privKey.toAddress()), change: true });
await deployTx.fee();
await deployTx.sign();

// Broadcast via ARC
const broadcaster = new ARC('https://arc.taal.com');
const result = await deployTx.broadcast(broadcaster);

// Spend the contract
const spendTx = new Transaction();
spendTx.addInput({
  sourceTransaction: deployTx,
  sourceOutputIndex: 0,
  unlockingScript: UnlockingScript.fromHex(
    contract.buildUnlockingScript('unlock', [sigHex, pubKeyHex])
  ),
});
```

---

## Loading Artifacts in Go

The Go compiler can both produce and consume artifacts. To load a pre-compiled artifact JSON file:

```go
package main

import (
    "encoding/json"
    "os"
)

// Artifact mirrors the TSOP artifact schema (same struct as in compiler.go)
type Artifact struct {
    Version         string       `json:"version"`
    CompilerVersion string       `json:"compilerVersion"`
    ContractName    string       `json:"contractName"`
    ABI             ABI          `json:"abi"`
    Script          string       `json:"script"`
    ASM             string       `json:"asm"`
    StateFields     []StateField `json:"stateFields,omitempty"`
    BuildTimestamp  string       `json:"buildTimestamp"`
}

func LoadArtifact(path string) (*Artifact, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var artifact Artifact
    if err := json.Unmarshal(data, &artifact); err != nil {
        return nil, err
    }
    return &artifact, nil
}
```

To compile and produce an artifact directly from Go:

```go
// Full pipeline: source file → artifact
artifact, err := CompileFromSource("Counter.tsop.go")
if err != nil {
    log.Fatal(err)
}

// Or from ANF IR JSON (portable across compilers)
artifact, err := CompileFromIR("Counter-anf.json")

// Access the compiled script
fmt.Println("Script hex:", artifact.Script)
fmt.Println("Contract:", artifact.ContractName)
for _, m := range artifact.ABI.Methods {
    if m.IsPublic {
        fmt.Printf("  %s(%v)\n", m.Name, m.Params)
    }
}

// Serialize back to JSON
jsonBytes, _ := ArtifactToJSON(artifact)
os.WriteFile("Counter.json", jsonBytes, 0644)
```

The `artifact.Script` hex string is the locking script ready to be embedded in a Bitcoin SV transaction using your Go BSV library of choice.

---

## Loading Artifacts in Rust

The Rust compiler uses serde for JSON serialization. To load a pre-compiled artifact:

```rust
use serde::{Deserialize, Serialize};
use std::fs;

// TSOPArtifact mirrors the shared schema (same struct as in artifact.rs)
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct TSOPArtifact {
    version: String,
    compiler_version: String,
    contract_name: String,
    abi: ABI,
    script: String,
    asm: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    state_fields: Option<Vec<StateField>>,
    build_timestamp: String,
}

fn load_artifact(path: &str) -> Result<TSOPArtifact, Box<dyn std::error::Error>> {
    let data = fs::read_to_string(path)?;
    let artifact: TSOPArtifact = serde_json::from_str(&data)?;
    Ok(artifact)
}
```

To compile and produce an artifact directly from Rust:

```rust
use tsop_compiler_rust::{compile_from_source, compile_from_source_str};
use std::path::Path;

// Full pipeline: source file → artifact
let artifact = compile_from_source(Path::new("Counter.tsop.rs"))
    .expect("compilation failed");

// Or from a source string
let source = std::fs::read_to_string("Counter.tsop.rs").unwrap();
let artifact = compile_from_source_str(&source, Some("Counter.tsop.rs"))
    .expect("compilation failed");

// Access the compiled script
println!("Script hex: {}", artifact.script);
println!("Contract: {}", artifact.contract_name);

// Serialize back to JSON
let json = serde_json::to_string_pretty(&artifact).unwrap();
std::fs::write("Counter.json", json).unwrap();
```

The `artifact.script` hex string is the locking script for embedding in a Bitcoin SV transaction using your Rust BSV library of choice.

---

## Cross-Compiler Workflow

All three compilers produce byte-identical Bitcoin Script for the same contract source. This means you can:

1. **Write** a contract in any supported format (`.tsop.ts`, `.tsop.sol`, `.tsop.go`, etc.)
2. **Test** using the language-native test runner (vitest, go test, cargo test)
3. **Compile** with any of the three compilers
4. **Load** the artifact from any language

The ANF IR (A-Normal Form Intermediate Representation) serves as the portable interchange format. You can generate ANF IR from one compiler and feed it to another:

```bash
# Generate ANF IR with the TypeScript compiler
tsop compile Counter.tsop.ts --ir

# Feed the ANF IR to the Go compiler backend
tsop-go --ir Counter-anf.json --output Counter.json

# Or to the Rust compiler backend
tsop-rust --ir Counter-anf.json --output Counter.json
```

This produces identical `script` hex in all three cases. The cross-compiler conformance test suite (`conformance/`) validates this guarantee by comparing SHA-256 hashes of the output across all compilers.

---

## Recommended Workflow

1. **Development**: Write contracts in your preferred format with full IDE support. TypeScript (`.tsop.ts`) has the most mature tooling.

2. **Testing**: Run language-native tests for fast iteration on business logic. Add `CompileCheck` / `compile_check` tests to catch TSOP language errors the host compiler would miss.

3. **Compilation**: Use the CLI (`tsop compile`) or the programmatic API to produce artifact JSON files. Commit artifacts to version control or store them as build outputs.

4. **Deployment**: Load the artifact in your application using `TSOPContract` (TypeScript), `json.Unmarshal` (Go), or `serde_json::from_str` (Rust). Use the `script` field as the locking script for transaction construction.

5. **Interaction**: Use the ABI from the artifact to build unlocking scripts for spending. For stateful contracts, use `stateFields` to serialize/deserialize on-chain state.
