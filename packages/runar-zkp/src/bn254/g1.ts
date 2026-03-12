/**
 * BN254 G1 curve operations — points on y^2 = x^3 + 3 over Fp.
 */

import { fpAdd, fpSub, fpMul, fpSqr, fpInv, fpMod, fpEq } from './field.js';
import { B } from './constants.js';
import type { G1Point, Fp } from '../types.js';

export const G1_INFINITY: G1Point = { x: 0n, y: 0n, infinity: true };

export function g1IsInfinity(p: G1Point): boolean {
  return p.infinity === true;
}

/** Check if a point is on the curve: y^2 = x^3 + 3 (mod p). */
export function g1OnCurve(p: G1Point): boolean {
  if (g1IsInfinity(p)) return true;
  const lhs = fpSqr(p.y);
  const rhs = fpAdd(fpMul(fpSqr(p.x), p.x), B);
  return fpEq(lhs, rhs);
}

/** Negate: (x, y) → (x, -y). */
export function g1Neg(p: G1Point): G1Point {
  if (g1IsInfinity(p)) return G1_INFINITY;
  return { x: p.x, y: fpSub(0n, p.y) };
}

/** Point addition (affine). */
export function g1Add(a: G1Point, b: G1Point): G1Point {
  if (g1IsInfinity(a)) return b;
  if (g1IsInfinity(b)) return a;

  const ax = fpMod(a.x), ay = fpMod(a.y);
  const bx = fpMod(b.x), by = fpMod(b.y);

  if (ax === bx) {
    if (ay === by) {
      // Point doubling
      return g1Double(a);
    }
    // a + (-a) = O
    return G1_INFINITY;
  }

  // λ = (by - ay) / (bx - ax)
  const lambda = fpMul(fpSub(by, ay), fpInv(fpSub(bx, ax)));
  // x_r = λ^2 - ax - bx
  const xr = fpSub(fpSub(fpSqr(lambda), ax), bx);
  // y_r = λ(ax - xr) - ay
  const yr = fpSub(fpMul(lambda, fpSub(ax, xr)), ay);

  return { x: xr, y: yr };
}

/** Point doubling (affine). */
export function g1Double(p: G1Point): G1Point {
  if (g1IsInfinity(p)) return G1_INFINITY;
  if (fpMod(p.y) === 0n) return G1_INFINITY;

  const px = fpMod(p.x), py = fpMod(p.y);

  // λ = 3x^2 / 2y  (a = 0 for BN254)
  const lambda = fpMul(fpMul(3n, fpSqr(px)), fpInv(fpMul(2n, py)));
  const xr = fpSub(fpSqr(lambda), fpMul(2n, px));
  const yr = fpSub(fpMul(lambda, fpSub(px, xr)), py);

  return { x: xr, y: yr };
}

/** Scalar multiplication: n * P (double-and-add). */
export function g1Mul(p: G1Point, n: bigint): G1Point {
  if (n === 0n || g1IsInfinity(p)) return G1_INFINITY;
  if (n < 0n) return g1Mul(g1Neg(p), -n);

  let result: G1Point = G1_INFINITY;
  let base = p;
  let scalar = n;

  while (scalar > 0n) {
    if (scalar & 1n) result = g1Add(result, base);
    base = g1Double(base);
    scalar >>= 1n;
  }

  return result;
}

/** Multi-scalar multiplication: Σ s_i * P_i. */
export function g1MultiMul(scalars: Fp[], points: G1Point[]): G1Point {
  let result = G1_INFINITY;
  for (let i = 0; i < scalars.length; i++) {
    result = g1Add(result, g1Mul(points[i]!, scalars[i]!));
  }
  return result;
}

export function g1Eq(a: G1Point, b: G1Point): boolean {
  if (g1IsInfinity(a) && g1IsInfinity(b)) return true;
  if (g1IsInfinity(a) || g1IsInfinity(b)) return false;
  return fpEq(a.x, b.x) && fpEq(a.y, b.y);
}
