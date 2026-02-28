# TSOP — TypeScript-to-Bitcoin Script Compiler

## Project Overview

TSOP compiles a strict subset of TypeScript into Bitcoin SV Script. Developers write smart contracts as TypeScript classes extending `SmartContract` (stateless) or `StatefulSmartContract` (stateful), and the compiler produces Bitcoin Script locking scripts.

Three independent compiler implementations (TypeScript, Go, Rust) must produce identical output for the same input.

## Repository Structure

```
packages/
  tsop-lang/          # Language: base classes, types, builtins (developer imports)
  tsop-compiler/      # TypeScript compiler: parser → validator → typecheck → ANF → stack → emit
  tsop-ir-schema/     # Shared IR type definitions and JSON schemas
  tsop-testing/       # TestContract API, Script VM, interpreter, fuzzer
  tsop-sdk/           # Deployment SDK (providers, signers, contract interaction)
  tsop-cli/           # CLI tool
compilers/
  go/                 # Go compiler implementation
  rust/               # Rust compiler implementation
conformance/          # Cross-compiler conformance test suite
examples/             # Example contracts with tests
spec/                 # Language specification (grammar, semantics, type system)
docs/                 # User-facing documentation
```

## Build & Test

```bash
pnpm install                       # Install dependencies
pnpm run build                     # Build all packages (turbo)
npx vitest run                     # Run all TypeScript tests (packages + examples)
npx vitest run examples/           # Run example contract tests only
cd compilers/go && go test ./...   # Run Go compiler tests
cd compilers/rust && cargo test    # Run Rust compiler tests
```

## Compiler Pipeline

Each pass is a pure function in `packages/tsop-compiler/src/passes/`:

1. **01-parse.ts** — TypeScript source → TSOP AST (`ContractNode`)
2. **02-validate.ts** — Language subset constraints (no mutation of the AST)
3. **03-typecheck.ts** — Type consistency verification
4. **04-anf-lower.ts** — AST → A-Normal Form IR (flattened let-bindings)
5. **05-stack-lower.ts** — ANF → Stack IR (Bitcoin Script stack operations)
6. **06-emit.ts** — Stack IR → hex-encoded Bitcoin Script

The optimizer (`src/optimizer/constant-fold.ts`) runs between passes 4 and 5.

## Key Conventions

### AST Types Are Defined in Two Places
`packages/tsop-compiler/src/ir/tsop-ast.ts` and `packages/tsop-ir-schema/src/tsop-ast.ts` must stay in sync. Both define `ContractNode`, `PropertyNode`, `MethodNode`, etc.

### Adding a New ANF Value Kind
When adding a new ANF IR node (like `add_output`), update ALL of these:
- `packages/tsop-compiler/src/ir/anf-ir.ts` — add interface + union member
- `packages/tsop-compiler/src/passes/04-anf-lower.ts` — emit the new node
- `packages/tsop-compiler/src/passes/05-stack-lower.ts` — handle in `lowerBinding` dispatch + `collectRefs`
- `packages/tsop-compiler/src/optimizer/constant-fold.ts` — add to `foldValue`, `collectRefsInValue`, `hasSideEffects`
- `compilers/go/ir/types.go` — add fields to `ANFValue` struct
- `compilers/go/ir/loader.go` — add to `knownKinds`
- `compilers/go/codegen/stack.go` — add to `collectRefs` + `lowerBinding` dispatch
- `compilers/go/frontend/anf_lower.go` — emit the new node
- `compilers/rust/src/ir/mod.rs` — add enum variant to `ANFValue`
- `compilers/rust/src/ir/loader.rs` — add to `KNOWN_KINDS` + `kind_name`
- `compilers/rust/src/codegen/stack.rs` — add to `collect_refs` + `lower_binding` dispatch
- `compilers/rust/src/frontend/anf_lower.rs` — emit the new node

### Three Compilers Must Stay in Sync
Any language feature change must be implemented in TypeScript, Go, AND Rust. Cross-compiler tests in `packages/tsop-compiler/src/__tests__/cross-compiler.test.ts` validate consistency.

### Contract Model
- `SmartContract` — stateless, all properties `readonly`, developer writes full logic
- `StatefulSmartContract` — compiler auto-injects `checkPreimage` at method entry and state continuation at exit
- `this.addOutput(satoshis, ...values)` — multi-output intrinsic; values are positional matching mutable properties in declaration order
- `parentClass` field on `ContractNode` discriminates between the two base classes

### Testing Contracts
```typescript
import { TestContract } from 'tsop-testing';

const counter = TestContract.fromSource(source, { count: 0n });
counter.call('increment');
expect(counter.state.count).toBe(1n);
```

`TestContract` uses the interpreter (not the VM) — it tests business logic with mocked crypto (`checkSig` always true, `checkPreimage` always true). Example tests live alongside contracts in `examples/`.

### Module Resolution
pnpm workspace packages are not hoisted to root `node_modules`. The `vitest.config.ts` at root provides aliases so `examples/` tests can import `tsop-testing` by name.

## Style

- No decorators in the TSOP language — TypeScript's own keywords (`public`, `private`, `readonly`) provide all expressiveness
- One contract class per source file
- Constructor must call `super(...)` as first statement, passing all properties
- Public methods are spending entry points; private methods are inlined helpers
- `assert()` is the primary control mechanism — scripts fail if any assert is false
