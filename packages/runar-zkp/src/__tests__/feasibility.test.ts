import { describe, it, expect } from 'vitest';
import { estimateOpSizes, estimateOptimizedVerifierSize } from '../bn254/field-script.js';
import { estimateVerifierSize, generateVerifierStub } from '../groth16/verify-script.js';
import { mockSetup } from '../prover/setup.js';

describe('Groth16 in BSV Script feasibility', () => {
  it('estimates Fp operation sizes with altstack P optimization', () => {
    const sizes = estimateOpSizes();

    // With altstack P caching, operations should be very compact
    expect(sizes.fpAdd!.bytes).toBe(5);   // ADD(1) + getP(3) + MOD(1)
    expect(sizes.fpMul!.bytes).toBe(5);   // MUL(1) + getP(3) + MOD(1)
    expect(sizes.fpSqr!.bytes).toBe(6);   // DUP(1) + fpMul(5)
    expect(sizes.fpSub!.bytes).toBe(10);  // getP(3) + ROT(1) + SUB(1) + ADD(1) + getP(3) + MOD(1)

    // fpInv: ~254 squarings + ~127 multiplications
    expect(sizes.fpInv!.bytes).toBeGreaterThan(1500);
    expect(sizes.fpInv!.bytes).toBeLessThan(3000);

    console.log('Optimized Fp operation sizes (bytes):');
    for (const [name, { bytes }] of Object.entries(sizes)) {
      console.log(`  ${name}: ${bytes} bytes`);
    }
  });

  it('estimates optimized Groth16 verifier size under 1 MB', () => {
    const est = estimateOptimizedVerifierSize(1);

    // With BSV native bigint ops + altstack P + sparse Fp12 + multi-Miller,
    // the verifier should be well under 1 MB
    expect(est.totalKB).toBeLessThan(1024); // under 1 MB
    expect(est.totalKB).toBeGreaterThan(50); // sanity: at least 50 KB

    console.log(`\nOptimized Groth16 verifier (1 public input): ${est.totalKB} KB`);
    console.log('Breakdown:');
    for (const [name, bytes] of Object.entries(est.breakdown)) {
      if (bytes > 100) {
        console.log(`  ${name}: ${(bytes / 1024).toFixed(1)} KB`);
      } else {
        console.log(`  ${name}: ${bytes} bytes`);
      }
    }
  });

  it('estimates size scales linearly with public inputs', () => {
    const est1 = estimateOptimizedVerifierSize(1);
    const est3 = estimateOptimizedVerifierSize(3);

    // More public inputs = more IC computation, but Miller loop stays the same
    expect(est3.totalBytes).toBeGreaterThan(est1.totalBytes);
    // The difference should be modest (just IC scalar muls)
    const diff = est3.totalBytes - est1.totalBytes;
    expect(diff).toBeLessThan(est1.totalBytes); // less than doubling

    console.log(`\n1 input: ${est1.totalKB} KB, 3 inputs: ${est3.totalKB} KB, delta: ${Math.ceil(diff/1024)} KB`);
  });

  it('compares to existing BSV scripts', () => {
    const est = estimateOptimizedVerifierSize(1);

    // SLH-DSA verification: ~188 KB
    // SHA-256 compression × 3: ~70 KB
    // Schnorr ZKP: ~877 KB
    // Our target: same order of magnitude
    console.log(`\nScript size comparison:`);
    console.log(`  SHA-256 compress ×3:  ~70 KB`);
    console.log(`  SLH-DSA verify:       ~188 KB`);
    console.log(`  Schnorr ZKP:          ~877 KB`);
    console.log(`  Groth16 verify (est): ~${est.totalKB} KB`);

    // Should be in the same ballpark as existing large scripts
    expect(est.totalKB).toBeLessThan(2048); // under 2 MB
  });

  it('generates a verifier stub', () => {
    const { vk } = mockSetup(1);
    const stub = generateVerifierStub(vk, 1);

    expect(stub.ops.length).toBeGreaterThan(0);
    // 8 proof drops + 1 input drop + 1 OP_TRUE = 10 ops
    expect(stub.opcodeCount).toBe(10);
  });

  it('estimateVerifierSize via VK object', () => {
    const { vk } = mockSetup(1);
    const est = estimateVerifierSize(vk);
    expect(est.feasible).toBe(true);
    expect(est.totalKB).toBeGreaterThan(0);
  });
});
