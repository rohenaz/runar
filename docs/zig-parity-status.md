# Zig Parity Status

Legend:

- `🟩` strong / natural
- `🟨` acceptable compromise
- `🟧` works but awkward
- `🟥` real gap

## Cross-Language Matrix

| Area | TS | Go | Rust | Python | Zig | Meaning |
|---|---|---|---|---|---|---|
| Compiler correctness | 🟩 | 🟩 | 🟩 | 🟩 | 🟩 | Zig is fully green on compiler tests and conformance |
| Native frontend syntax | 🟩 | 🟨 | 🟨 | 🟨 | 🟨 | Zig parses and compiles well, but the surface is not yet fully plain-Zig natural |
| Native helper/runtime package | 🟩 | 🟩 | 🟩 | 🟩 | 🟨 | `packages/runar-zig` exists, compile-check is now honest, and hash/byte semantics are materially better, but advanced crypto is still scaffolded |
| Example inventory parity | 🟩 | 🟩 | 🟩 | 🟩 | 🟩 | Zig now has the same 21-example tree |
| Adjacent native tests | 🟩 | 🟩 | 🟩 | 🟩 | 🟩 | Zig now has tests beside the contracts |
| Real contract execution in tests | 🟩 | 🟩 | 🟩 | 🟩 | 🟨 | The direct-contract set is now materially larger, including stateful/output-heavy cases, but the full priority set is not done yet |
| Stateful contract model fit | 🟨 | 🟨 | 🟨 | 🟨 | 🟧 | Zig now has an honest explicit `StatefulContext` bridge, but the full example tree has not migrated to it yet |
| Byte/string equality fit | 🟩 | 🟩 | 🟩 | 🟩 | 🟨 | Zig now has an explicit `runar.bytesEq(...)` model and more contracts are using it, but migration is still incomplete |
| Failure/assertion model fit | 🟩 | 🟨 | 🟨 | 🟨 | 🟧 | Zig needs a more deliberate contract-failure story |
| Test quality vs mirrors | 🟩 | 🟩 | 🟩 | 🟩 | 🟧 | Mirror usage is shrinking, but many important Zig tests still do not hit the actual contract module |

## Language-Specific Boundaries

| Language | Main Native Challenge | What We Accept | What We Should Not Accept |
|---|---|---|---|
| TypeScript | DSL pressure, reference-compiler complexity | AST/reference role, stronger tooling assumptions | Hidden semantics that diverge from source language expectations |
| Go | No classes, less expression-heavy syntax | struct tags, explicit helper calls, embedded support types | unnatural OO emulation |
| Rust | Macro-heavy ergonomics | attributes/macros if explicit and type-safe | opaque macro magic that hides contract behavior |
| Python | dynamic runtime, decorators | decorators and runtime helpers | relying on dynamic behavior with weak compile guarantees |
| Zig | no inheritance, strict slice semantics, visible ownership | explicit helpers, wrappers, composition, comptime type helpers | fake inheritance, operator overloading-by-convention, hidden runtime magic |

## Where Zig Is Being Shoehorned

| Zig Area | Current State | Rating | Why |
|---|---|---|---|
| `pub const Contract = runar.StatefulSmartContract;` | treated like base-class inheritance | 🟥 | Zig does not work that way |
| `self.addOutput(...)`, `self.txPreimage` | being replaced by explicit `ctx: runar.StatefulContext` in migrated contracts | 🟧 | the new shape is honest, but the old implicit surface still exists in unmigrated examples |
| byte/content equality in contract syntax | written as if plain operators can carry DSL meaning | 🟥 | misleading to Zig users |
| `runar.Readonly(T)` | explicit type-level marker | 🟩 | this fits Zig reasonably well |
| `packages/runar-zig` as helper/runtime layer | explicit package boundary | 🟨 | good direction, validator/compile-check is now honest, but advanced crypto remains scaffolded |
| adjacent example tests | present and runnable | 🟨 | structure is right, and a few are now direct, but many assertions still target mirrors |

## Zig-Specific Decision Matrix

| Topic | Current Approach | Problem | Better Zig-Shaped Direction | Status |
|---|---|---|---|---|
| Contract base model | pseudo-inheritance via `pub const Contract = ...` | implies hidden fields/methods | explicit composition or explicit helper surface | Open |
| Stateful helpers | explicit `ctx: runar.StatefulContext` on migrated contracts | still mixed with older implicit `self.*` examples | keep the explicit context model and migrate the remaining stateful tree to it | Partial |
| Bytes equality | compiler-level semantics on plain operators | misleading in plain Zig | explicit helper such as `runar.bytesEq(a, b)` or a wrapped bytes type with honest API | Open |
| Readonly fields | `runar.Readonly(T)` | acceptable, but needs consistent examples/docs | keep, refine, and document as the canonical explicit marker | Good |
| Contract failure model | mostly piggybacks on `runar.assert` | not yet clearly designed for Zig-native tests | explicit, documented assertion/failure behavior for `zig test` | Partial |
| Ownership/allocation | hidden in helper calls | can become noisy or surprising | keep high-level helpers but make ownership rules explicit and boring | Partial |
| Example tests | many mirror implementations | proves less than real contract execution | migrate priority examples to direct contract tests as semantics become natural | Open |

## Current Wave Progress

- Removed the Zig constructor-validation suppression from `packages/runar-zig`; compile-check now fails honestly and the validator understands Zig `init` assignment semantics directly.
- Upgraded several core helper semantics in `packages/runar-zig`, including `ripemd160`, `hash160`, signed-magnitude `num2bin` / `bin2num`, `checkMultiSig`, `sha256Compress`, `sha256Finalize`, and single-block `blake3` helpers.
- Converted three example suites from mirror-only behavior tests to partial real-contract execution:
  - `escrow`
  - `stateful-counter`
  - `property-initializers`
- Added a dedicated Zig `assert_probe` executable so negative-path contract assertions can be tested honestly from the example suite without pretending panics are catchable in-process.
- Added an explicit `runar.bytesEq(...)` path in both Zig frontends and `packages/runar-zig`, then used it to migrate the first byte-comparison contracts away from misleading plain `==`.
- Added an explicit `runar.StatefulContext` bridge in `packages/runar-zig` and both Zig compiler pipelines. The compiler now erases that source-level context from the ABI while native Zig tests can seed it with real preimages and inspect real outputs.
- Migrated `auction` to the explicit context model and converted it to direct real-contract tests, including probe-backed negative assertions.
- Migrated `token-ft` to the explicit context model, explicit `runar.bytesEq(...)`, and real output-capture tests.
- Migrated `token-nft` to the explicit context model and direct output-capture tests.
- Migrated the `join` / `move` half of `tic-tac-toe` to the real contract path and replaced dishonest pubkey equality with `runar.bytesEq(...)` there.
- The remaining direct-execution blockers are now clearer:
  - the remaining stateful/output-heavy examples that still rely on the old implicit `self.*` surface
  - byte-content equality expressed as plain `==` in unmigrated contracts
  - advanced crypto helpers that are still only runtime scaffolding
- Advanced crypto helpers are still not honest enough:
  - Rabin verification is still scaffolded
  - WOTS / SLH-DSA verification is still scaffolded
  - EC point helpers are still toy arithmetic

## Target Model

What “good” should look like for Zig:

1. A Zig developer can open a `.runar.zig` contract and it reads like honest Zig.
2. The runtime surface the contract depends on is explicit in the code, not implied by fake inheritance.
3. If a construct has special contract semantics, the syntax should signal that clearly.
4. Adjacent Zig tests should mostly exercise the real contract modules, not mirrors.
5. `packages/runar-zig` should feel like a coherent native helper/runtime package, not a grab bag of compiler accommodations.

## What Is Already Strong

- Compiler correctness and conformance are green.
- The Zig frontend now has explicit readonly syntax.
- `packages/runar-zig` now validates Zig constructors honestly instead of suppressing a known error.
- `packages/runar-zig` now has materially better hash/byte/number helper semantics for the simpler examples.
- The full 21-contract Zig example tree exists with adjacent tests.
- `escrow`, `stateful-counter`, `property-initializers`, `auction`, `token-ft`, and `token-nft` now execute the real contract module for their positive-path tests.
- `stateful-counter` and `property-initializers` now also cover their negative assertion paths through a dedicated subprocess probe instead of mirrors.
- `p2pkh`, `p2blake3pkh`, `blake3`, `sha256-compress`, and `sha256-finalize` now use explicit `runar.bytesEq(...)` and execute real contract paths in their Zig tests, including real negative assertion probes.
- `auction`, `token-ft`, and `token-nft` now prove the explicit `StatefulContext` direction in real tests instead of mirrors.
- `tic-tac-toe` now has real-contract coverage for join/move and their negative assertion rules, but its output-heavy win/tie/cancel methods are still pending migration.
- The public docs now describe the Zig package and example runner accurately.

## What Still Needs Real Improvement

### 1. Plain-Zig Executability

This is the biggest remaining gap.

Today some `.runar.zig` contracts compile through the Rúnar frontend correctly, but are not naturally executable as ordinary Zig modules because they rely on Rúnar-level semantics that Zig itself does not provide directly.

Representative issues:

- unmigrated stateful examples still using the old implicit helper surface
- contract-style assertions/failure expectations
- advanced crypto helpers that are still runtime scaffolding rather than real semantics
- the remaining example tree still has a mix of old `==` byte comparisons and newer explicit `runar.bytesEq(...)`

### 2. Stateful Surface Design

We need one clear answer for how stateful contracts access runtime features in Zig.

The chosen direction is now clearer:

- explicit helper methods that receive `ctx: runar.StatefulContext`

The unacceptable shape is:

- pretending Zig has class inheritance and injected members

### 3. Direct Example Testing

The test tree shape is now correct, but the quality bar is not met until the important examples are mostly testing the real contracts.

Priority contracts to move next:

- `tic-tac-toe`
- `ec-demo`
- `post-quantum-wallet`
- `sphincs-wallet`

## Priority Ladder

| Priority | Work Item | Why |
|---|---|---|
| P0 | finish migrating the remaining stateful/output-heavy examples to `StatefulContext` | the model is now chosen; migration is the main remaining semantic cleanup |
| P0 | settle the honest bytes/content equality API | this blocks natural direct execution |
| P1 | finish the output-heavy half of `tic-tac-toe` | best remaining stress test for the stateful design |
| P2 | convert EC/PQ examples to real-contract tests | important, but dependent on cleaner core semantics |
| P2 | reduce mirror-test usage across the tree | cleanup after the core model is right |

## Acceptance Criteria

We should consider Zig “one of the best implementations” only when most of these are true:

- `packages/runar-zig` is the obvious native way to support Zig contracts and tests
- stateful Zig contracts use an explicit, understandable runtime access model
- byte/content comparisons are expressed honestly
- at least the priority example set runs as real contract tests under `zig test`
- mirrors are the exception, not the default
- docs can explain the Zig model without caveats that sound like compiler escape hatches

## Phased Roadmap

### Phase 1: Make It Honest

- finalize runtime/stateful access design
- finalize byte/content equality design
- remove the most misleading pseudo-inheritance assumptions

### Phase 2: Make It Direct

- convert the priority examples to real-contract tests
- keep compile-check coverage, but stop leaning on mirrors for core confidence

### Phase 3: Make It Excellent

- refine ergonomics
- tighten failure/assertion behavior
- improve advanced EC/PQ example testing quality
- reduce any leftover runtime awkwardness in `packages/runar-zig`

## Bottom Line

| Question | Answer |
|---|---|
| Is the compiler work good enough? | Yes |
| Is the Zig package/test surface useful already? | Yes |
| Is it fully natural and up to the same standard as Go/Rust/Python? | No |
| What most needs improvement? | direct execution model for `.runar.zig` contracts, especially stateful/runtime semantics |
| What should we protect no matter what? | no fake inheritance, no dishonest operator semantics, no mirror-heavy tests as the final state |

## Guiding Standard

If a Zig developer reads the code, it should look like honest Zig with explicit Rúnar helpers, not Zig-shaped syntax carrying hidden compiler-only meanings.
