import { describe, it, expect } from 'vitest';
import { g1Add, g1Double, g1Mul, g1Neg, g1OnCurve, g1Eq, G1_INFINITY } from '../bn254/g1.js';
import { G1_X, G1_Y } from '../bn254/constants.js';
import type { G1Point } from '../types.js';

const G: G1Point = { x: G1_X, y: G1_Y };

describe('BN254 G1 curve operations', () => {
  it('generator is on curve', () => {
    expect(g1OnCurve(G)).toBe(true);
  });

  it('infinity is on curve', () => {
    expect(g1OnCurve(G1_INFINITY)).toBe(true);
  });

  it('G + O = G', () => {
    expect(g1Eq(g1Add(G, G1_INFINITY), G)).toBe(true);
    expect(g1Eq(g1Add(G1_INFINITY, G), G)).toBe(true);
  });

  it('G + (-G) = O', () => {
    const negG = g1Neg(G);
    expect(g1OnCurve(negG)).toBe(true);
    expect(g1Eq(g1Add(G, negG), G1_INFINITY)).toBe(true);
  });

  it('2*G via doubling equals G + G', () => {
    const doubled = g1Double(G);
    const added = g1Add(G, G);
    expect(g1Eq(doubled, added)).toBe(true);
    expect(g1OnCurve(doubled)).toBe(true);
  });

  it('scalar multiplication: 0 * G = O', () => {
    expect(g1Eq(g1Mul(G, 0n), G1_INFINITY)).toBe(true);
  });

  it('scalar multiplication: 1 * G = G', () => {
    expect(g1Eq(g1Mul(G, 1n), G)).toBe(true);
  });

  it('scalar multiplication: 2 * G = G + G', () => {
    expect(g1Eq(g1Mul(G, 2n), g1Add(G, G))).toBe(true);
  });

  it('scalar multiplication: 3 * G = 2*G + G', () => {
    const threeG = g1Mul(G, 3n);
    const twoGplusG = g1Add(g1Mul(G, 2n), G);
    expect(g1Eq(threeG, twoGplusG)).toBe(true);
    expect(g1OnCurve(threeG)).toBe(true);
  });
});
