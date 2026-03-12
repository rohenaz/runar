/**
 * Groth16 verifier → Bitcoin Script codegen.
 *
 * Generates a sequence of StackOp[] that verify a Groth16 proof on-chain.
 * The verification key is embedded as constants in the script.
 *
 * ## Architecture
 *
 * The verifier checks: e(A,B) · e(-α,β) · e(-L,γ) · e(-C,δ) = 1
 * where L = IC[0] + Σ input_i · IC[i+1].
 *
 * This is implemented as:
 * 1. IC computation (multi-scalar multiplication on G1)
 * 2. Multi-Miller loop (4 pairings computed simultaneously)
 * 3. Final exponentiation
 * 4. Check result == 1 in Fp12
 *
 * ## BSV-optimized size estimate
 *
 * With native OP_MUL/OP_MOD on arbitrary bigints, altstack P caching,
 * sparse Fp12 multiplication, and multi-Miller loop:
 *
 * | Component          | Size (KB) |
 * |--------------------|-----------|
 * | IC computation     | ~15-50    |
 * | Miller loop (4x)   | ~300-500  |
 * | Final exponentiation | ~50-80  |
 * | VK + setup         | ~1        |
 * | **Total**          | **~400-600** |
 *
 * This is comparable to existing BSV scripts (SLH-DSA verification is ~200 KB,
 * SHA-256 compression is ~23 KB × 3 = ~70 KB).
 */

import type { StackOp } from 'runar-ir-schema';
import type { VerificationKey, VerifierScript } from '../types.js';
import { estimateOptimizedVerifierSize } from '../bn254/field-script.js';

/**
 * Estimate the script size for a full Groth16 verifier.
 */
export function estimateVerifierSize(vk: VerificationKey): {
  totalBytes: number;
  totalKB: number;
  breakdown: Record<string, number>;
  feasible: boolean;
} {
  const numInputs = vk.ic.length - 1;
  const est = estimateOptimizedVerifierSize(numInputs);

  return {
    ...est,
    feasible: true, // Always feasible on BSV
  };
}

/**
 * Generate a STUB Groth16 verifier script.
 *
 * This version drops all proof/input data and pushes OP_TRUE.
 * Will be replaced with real Groth16 verifier codegen.
 */
export function generateVerifierStub(
  _vk: VerificationKey,
  numPublicInputs: number,
): VerifierScript {
  const ops: StackOp[] = [];

  // Drop proof components (8 field elements)
  for (let i = 0; i < 8; i++) {
    ops.push({ op: 'drop' } as StackOp);
  }

  // Drop public inputs
  for (let i = 0; i < numPublicInputs; i++) {
    ops.push({ op: 'drop' } as StackOp);
  }

  // Push OP_TRUE
  ops.push({ op: 'opcode', code: 'OP_TRUE' } as StackOp);

  return {
    ops,
    scriptSizeBytes: ops.length,
    opcodeCount: ops.length,
  };
}
