/**
 * BN254 quadratic extension field Fp2 = Fp[u] / (u^2 + 1).
 *
 * Elements are pairs (c0, c1) representing c0 + c1 * u where u^2 = -1.
 */

import { fpAdd, fpSub, fpMul, fpNeg, fpInv, fpMod, fpSqr } from './field.js';
import type { Fp, Fp2 } from '../types.js';

export function fp2(c0: Fp, c1: Fp): Fp2 {
  return { c0: fpMod(c0), c1: fpMod(c1) };
}

export const FP2_ZERO: Fp2 = { c0: 0n, c1: 0n };
export const FP2_ONE: Fp2 = { c0: 1n, c1: 0n };

export function fp2Add(a: Fp2, b: Fp2): Fp2 {
  return { c0: fpAdd(a.c0, b.c0), c1: fpAdd(a.c1, b.c1) };
}

export function fp2Sub(a: Fp2, b: Fp2): Fp2 {
  return { c0: fpSub(a.c0, b.c0), c1: fpSub(a.c1, b.c1) };
}

/** Multiply: (a0 + a1*u)(b0 + b1*u) = (a0*b0 - a1*b1) + (a0*b1 + a1*b0)*u */
export function fp2Mul(a: Fp2, b: Fp2): Fp2 {
  const t0 = fpMul(a.c0, b.c0);
  const t1 = fpMul(a.c1, b.c1);
  return {
    c0: fpSub(t0, t1),
    c1: fpSub(fpMul(fpAdd(a.c0, a.c1), fpAdd(b.c0, b.c1)), fpAdd(t0, t1)),
  };
}

/** Square: (a0 + a1*u)^2 = (a0^2 - a1^2) + 2*a0*a1*u */
export function fp2Sqr(a: Fp2): Fp2 {
  const t0 = fpMul(a.c0, a.c1);
  return {
    c0: fpMul(fpAdd(a.c0, a.c1), fpSub(a.c0, a.c1)),
    c1: fpAdd(t0, t0),
  };
}

export function fp2Neg(a: Fp2): Fp2 {
  return { c0: fpNeg(a.c0), c1: fpNeg(a.c1) };
}

/** Conjugate: (a0, a1) → (a0, -a1). Since u^2 = -1, conj(a) = a0 - a1*u. */
export function fp2Conj(a: Fp2): Fp2 {
  return { c0: a.c0, c1: fpNeg(a.c1) };
}

/** Inverse: 1 / (a0 + a1*u) = (a0 - a1*u) / (a0^2 + a1^2) */
export function fp2Inv(a: Fp2): Fp2 {
  const norm = fpAdd(fpSqr(a.c0), fpSqr(a.c1)); // a0^2 + a1^2
  const inv = fpInv(norm);
  return { c0: fpMul(a.c0, inv), c1: fpMul(fpNeg(a.c1), inv) };
}

export function fp2Eq(a: Fp2, b: Fp2): boolean {
  return fpMod(a.c0) === fpMod(b.c0) && fpMod(a.c1) === fpMod(b.c1);
}

export function fp2IsZero(a: Fp2): boolean {
  return fpMod(a.c0) === 0n && fpMod(a.c1) === 0n;
}

/** Multiply by scalar in Fp. */
export function fp2MulScalar(a: Fp2, s: Fp): Fp2 {
  return { c0: fpMul(a.c0, s), c1: fpMul(a.c1, s) };
}
