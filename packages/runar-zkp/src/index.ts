/**
 * runar-zkp — ZKP (Groth16/BN254) for Rúnar Bitcoin Script contracts.
 *
 * This package provides:
 * - BN254 field and curve arithmetic (reference + Bitcoin Script codegen)
 * - Groth16 proof verification (off-chain and on-chain Script codegen)
 * - IVC circuit definitions for chain verification
 * - Mock prover/setup for testing
 */

// Types
export type {
  Fp, Fp2, Fp6, Fp12,
  G1Point, G2Point,
  Groth16Proof, VerificationKey, ProvingKey,
  SerializedProof, VerifierScript,
} from './types.js';
export { PROOF_BYTE_SIZE } from './types.js';

// BN254 field arithmetic (reference)
export {
  fpAdd, fpSub, fpMul, fpSqr, fpNeg, fpInv, fpDiv,
  fpPow, fpMod, fpEq, fpIsZero, fpToBytes, fpFromBytes, fpToHex,
} from './bn254/field.js';

// BN254 Fp2 extension
export {
  fp2, fp2Add, fp2Sub, fp2Mul, fp2Sqr, fp2Neg, fp2Conj, fp2Inv,
  fp2Eq, fp2IsZero, fp2MulScalar, FP2_ZERO, FP2_ONE,
} from './bn254/fp2.js';

// BN254 G1 operations
export {
  g1Add, g1Double, g1Mul, g1Neg, g1MultiMul,
  g1OnCurve, g1Eq, g1IsInfinity, G1_INFINITY,
} from './bn254/g1.js';

// BN254 G2 operations
export {
  g2Add, g2Double, g2Mul, g2Neg,
  g2OnCurve, g2Eq, g2IsInfinity, G2_INFINITY,
} from './bn254/g2.js';

// BN254 constants
export { P, R, B, G1_X, G1_Y, BN_X } from './bn254/constants.js';

// Pairing
export { pairing, checkPairingProduct } from './bn254/pairing.js';

// Groth16 verification (off-chain)
export { verifyGroth16 } from './groth16/verify.js';

// Groth16 Bitcoin Script codegen
export { estimateVerifierSize, generateVerifierStub } from './groth16/verify-script.js';

// Field arithmetic Bitcoin Script codegen
export {
  emitFpAdd, emitFpSub, emitFpMul, emitFpSqr, emitFpNeg, emitFpInv,
  emitFpMod, emitPushP, emitPushFp, emitInitP, emitCleanupP,
  estimateOpSizes, estimateOptimizedVerifierSize,
} from './bn254/field-script.js';

// Circuit definitions
export { defineIVCStepCircuit } from './circuit/ivc-step.js';
export { defineGenesisCircuit } from './circuit/genesis-base.js';

// Mock prover/setup (for testing)
export { mockSetup } from './prover/setup.js';
export { mockProve } from './prover/prove.js';
