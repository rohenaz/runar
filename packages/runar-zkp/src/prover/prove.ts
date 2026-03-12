/**
 * Groth16 prover — placeholder.
 *
 * A real implementation would compute the Groth16 proof from the
 * witness, proving key, and R1CS constraints.
 *
 * The actual prover would use:
 * 1. Nova folding for IVC (accumulate chain proofs incrementally)
 * 2. Groth16 compression (compress the Nova proof into a succinct proof)
 *
 * For now, this exports a mock prover that generates dummy proofs.
 */

import type { Groth16Proof, ProvingKey, G1Point, Fp } from '../types.js';
import { G1_X, G1_Y } from '../bn254/constants.js';

const G1_GEN: G1Point = { x: G1_X, y: G1_Y };

/**
 * Generate a MOCK Groth16 proof for testing.
 * NOT SECURE — this proof will not pass real verification.
 */
export function mockProve(
  _pk: ProvingKey,
  _publicInputs: Fp[],
  _witness: unknown,
): Groth16Proof {
  return {
    a: G1_GEN,
    b: {
      x: { c0: 0n, c1: 0n },
      y: { c0: 0n, c1: 0n },
      infinity: true,
    },
    c: G1_GEN,
  };
}
