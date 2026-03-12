import { describe, it, expect } from 'vitest';
import {
  fpAdd, fpSub, fpMul, fpSqr, fpNeg, fpInv, fpDiv,
  fpPow, fpMod, fpToBytes, fpFromBytes,
} from '../bn254/field.js';
import { P } from '../bn254/constants.js';

describe('BN254 Fp field arithmetic', () => {
  it('fpMod reduces correctly', () => {
    expect(fpMod(0n)).toBe(0n);
    expect(fpMod(1n)).toBe(1n);
    expect(fpMod(P)).toBe(0n);
    expect(fpMod(P + 1n)).toBe(1n);
    expect(fpMod(-1n)).toBe(P - 1n);
  });

  it('fpAdd wraps around p', () => {
    expect(fpAdd(P - 1n, 1n)).toBe(0n);
    expect(fpAdd(P - 1n, 2n)).toBe(1n);
    expect(fpAdd(0n, 0n)).toBe(0n);
  });

  it('fpSub handles underflow', () => {
    expect(fpSub(0n, 1n)).toBe(P - 1n);
    expect(fpSub(5n, 3n)).toBe(2n);
    expect(fpSub(3n, 5n)).toBe(P - 2n);
  });

  it('fpMul basic', () => {
    expect(fpMul(2n, 3n)).toBe(6n);
    expect(fpMul(P - 1n, P - 1n)).toBe(1n); // (-1)^2 = 1
    expect(fpMul(0n, 42n)).toBe(0n);
  });

  it('fpSqr is consistent with fpMul', () => {
    const a = 123456789n;
    expect(fpSqr(a)).toBe(fpMul(a, a));
  });

  it('fpNeg produces additive inverse', () => {
    expect(fpAdd(42n, fpNeg(42n))).toBe(0n);
    expect(fpNeg(0n)).toBe(0n);
    expect(fpNeg(1n)).toBe(P - 1n);
  });

  it('fpInv produces multiplicative inverse', () => {
    expect(fpMul(7n, fpInv(7n))).toBe(1n);
    expect(fpMul(P - 1n, fpInv(P - 1n))).toBe(1n);
  });

  it('fpInv throws on zero', () => {
    expect(() => fpInv(0n)).toThrow('division by zero');
  });

  it('fpDiv is consistent with fpMul and fpInv', () => {
    expect(fpDiv(6n, 3n)).toBe(2n);
    expect(fpDiv(1n, 7n)).toBe(fpInv(7n));
  });

  it('fpPow computes small powers', () => {
    expect(fpPow(2n, 0n)).toBe(1n);
    expect(fpPow(2n, 1n)).toBe(2n);
    expect(fpPow(2n, 10n)).toBe(1024n);
    expect(fpPow(P - 1n, 2n)).toBe(1n); // (-1)^2 = 1
  });

  it('Fermat: a^{p-1} = 1 for nonzero a', () => {
    expect(fpPow(42n, P - 1n)).toBe(1n);
  });

  it('fpToBytes and fpFromBytes roundtrip', () => {
    const values = [0n, 1n, P - 1n, 123456789012345678901234567890n];
    for (const v of values) {
      const bytes = fpToBytes(v);
      expect(bytes.length).toBe(32);
      expect(fpFromBytes(bytes)).toBe(fpMod(v));
    }
  });
});
