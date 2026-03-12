/**
 * Groth16 trusted setup — placeholder.
 *
 * A real implementation would generate the proving key and verification key
 * from the R1CS circuit using a trusted setup ceremony (powers of tau +
 * circuit-specific phase 2).
 *
 * For now, this exports placeholder types and a mock setup function.
 */

import type { VerificationKey, ProvingKey, G1Point, G2Point } from '../types.js';
import { G1_X, G1_Y, G2_X_C0, G2_X_C1, G2_Y_C0, G2_Y_C1 } from '../bn254/constants.js';

const G1_GEN: G1Point = { x: G1_X, y: G1_Y };
const G2_GEN: G2Point = {
  x: { c0: G2_X_C0, c1: G2_X_C1 },
  y: { c0: G2_Y_C0, c1: G2_Y_C1 },
};

/**
 * Generate a MOCK verification key for testing.
 * NOT SECURE — do not use in production.
 */
export function mockSetup(numPublicInputs: number): { pk: ProvingKey; vk: VerificationKey } {
  const ic: G1Point[] = [];
  for (let i = 0; i <= numPublicInputs; i++) {
    ic.push(G1_GEN); // placeholder — real setup would use random points
  }

  const vk: VerificationKey = {
    alpha: G1_GEN,
    beta: G2_GEN,
    gamma: G2_GEN,
    delta: G2_GEN,
    ic,
  };

  return { pk: { vk }, vk };
}
