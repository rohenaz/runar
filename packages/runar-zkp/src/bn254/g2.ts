/**
 * BN254 G2 curve operations — points on the twist curve over Fp2.
 *
 * The twist curve is y^2 = x^3 + b' where b' = 3 / (9 + u).
 */

import { fp2Add, fp2Sub, fp2Mul, fp2Sqr, fp2Inv, fp2Eq, fp2IsZero, fp2, FP2_ZERO } from './fp2.js';
import { B_TWIST_C0, B_TWIST_C1 } from './constants.js';
import type { G2Point, Fp2 } from '../types.js';

const B_TWIST: Fp2 = { c0: B_TWIST_C0, c1: B_TWIST_C1 };

export const G2_INFINITY: G2Point = { x: FP2_ZERO, y: FP2_ZERO, infinity: true };

export function g2IsInfinity(p: G2Point): boolean {
  return p.infinity === true;
}

/** Check if a point is on the twist curve: y^2 = x^3 + b' (over Fp2). */
export function g2OnCurve(p: G2Point): boolean {
  if (g2IsInfinity(p)) return true;
  const lhs = fp2Sqr(p.y);
  const rhs = fp2Add(fp2Mul(fp2Sqr(p.x), p.x), B_TWIST);
  return fp2Eq(lhs, rhs);
}

export function g2Neg(p: G2Point): G2Point {
  if (g2IsInfinity(p)) return G2_INFINITY;
  return { x: p.x, y: fp2Sub(FP2_ZERO, p.y) };
}

export function g2Add(a: G2Point, b: G2Point): G2Point {
  if (g2IsInfinity(a)) return b;
  if (g2IsInfinity(b)) return a;

  if (fp2Eq(a.x, b.x)) {
    if (fp2Eq(a.y, b.y)) return g2Double(a);
    return G2_INFINITY;
  }

  const lambda = fp2Mul(fp2Sub(b.y, a.y), fp2Inv(fp2Sub(b.x, a.x)));
  const xr = fp2Sub(fp2Sub(fp2Sqr(lambda), a.x), b.x);
  const yr = fp2Sub(fp2Mul(lambda, fp2Sub(a.x, xr)), a.y);

  return { x: xr, y: yr };
}

export function g2Double(p: G2Point): G2Point {
  if (g2IsInfinity(p)) return G2_INFINITY;
  if (fp2IsZero(p.y)) return G2_INFINITY;

  // λ = 3x^2 / 2y  (a = 0 for BN254 twist)
  const three_x2 = fp2Mul(fp2(3n, 0n), fp2Sqr(p.x));
  const two_y = fp2Add(p.y, p.y);
  const lambda = fp2Mul(three_x2, fp2Inv(two_y));
  const xr = fp2Sub(fp2Sqr(lambda), fp2Add(p.x, p.x));
  const yr = fp2Sub(fp2Mul(lambda, fp2Sub(p.x, xr)), p.y);

  return { x: xr, y: yr };
}

export function g2Mul(p: G2Point, n: bigint): G2Point {
  if (n === 0n || g2IsInfinity(p)) return G2_INFINITY;
  if (n < 0n) return g2Mul(g2Neg(p), -n);

  let result: G2Point = G2_INFINITY;
  let base = p;
  let scalar = n;

  while (scalar > 0n) {
    if (scalar & 1n) result = g2Add(result, base);
    base = g2Double(base);
    scalar >>= 1n;
  }

  return result;
}

export function g2Eq(a: G2Point, b: G2Point): boolean {
  if (g2IsInfinity(a) && g2IsInfinity(b)) return true;
  if (g2IsInfinity(a) || g2IsInfinity(b)) return false;
  return fp2Eq(a.x, b.x) && fp2Eq(a.y, b.y);
}
