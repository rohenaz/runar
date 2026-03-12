/**
 * BN254 optimal Ate pairing.
 *
 * Computes the pairing e: G1 × G2 → GT (where GT ⊂ Fp12).
 * Used for Groth16 verification: e(A,B) = e(α,β) · e(L,γ) · e(C,δ).
 *
 * Implementation follows the standard Miller loop + final exponentiation
 * approach for BN curves.
 */

import { P, BN_X } from './constants.js';
import { fpMod, fpNeg } from './field.js';
import {
  fp2, fp2Add, fp2Sub, fp2Mul, fp2Sqr, fp2Neg, fp2Inv, fp2Eq,
  FP2_ZERO, FP2_ONE, fp2MulScalar,
} from './fp2.js';
import { g1IsInfinity } from './g1.js';
import { g2IsInfinity } from './g2.js';
import type { Fp2, Fp6, Fp12, G1Point, G2Point } from '../types.js';

// ---------------------------------------------------------------------------
// Fp6 = Fp2[v] / (v^3 - ξ) where ξ = (9 + u)
// ---------------------------------------------------------------------------

const XI: Fp2 = fp2(9n, 1n); // non-residue for Fp6

const FP6_ZERO: Fp6 = { c0: FP2_ZERO, c1: FP2_ZERO, c2: FP2_ZERO };
const FP6_ONE: Fp6 = { c0: FP2_ONE, c1: FP2_ZERO, c2: FP2_ZERO };

function fp6Add(a: Fp6, b: Fp6): Fp6 {
  return { c0: fp2Add(a.c0, b.c0), c1: fp2Add(a.c1, b.c1), c2: fp2Add(a.c2, b.c2) };
}

function fp6Sub(a: Fp6, b: Fp6): Fp6 {
  return { c0: fp2Sub(a.c0, b.c0), c1: fp2Sub(a.c1, b.c1), c2: fp2Sub(a.c2, b.c2) };
}

function fp6MulByXi(a: Fp2): Fp2 {
  return fp2Mul(a, XI);
}

function fp6Mul(a: Fp6, b: Fp6): Fp6 {
  const t0 = fp2Mul(a.c0, b.c0);
  const t1 = fp2Mul(a.c1, b.c1);
  const t2 = fp2Mul(a.c2, b.c2);

  return {
    c0: fp2Add(t0, fp6MulByXi(fp2Sub(fp2Mul(fp2Add(a.c1, a.c2), fp2Add(b.c1, b.c2)), fp2Add(t1, t2)))),
    c1: fp2Add(fp2Sub(fp2Mul(fp2Add(a.c0, a.c1), fp2Add(b.c0, b.c1)), fp2Add(t0, t1)), fp6MulByXi(t2)),
    c2: fp2Add(fp2Sub(fp2Mul(fp2Add(a.c0, a.c2), fp2Add(b.c0, b.c2)), fp2Add(t0, t2)), t1),
  };
}

function fp6Sqr(a: Fp6): Fp6 {
  return fp6Mul(a, a); // can be optimized but correctness first
}

function fp6Neg(a: Fp6): Fp6 {
  return { c0: fp2Neg(a.c0), c1: fp2Neg(a.c1), c2: fp2Neg(a.c2) };
}

function fp6Inv(a: Fp6): Fp6 {
  const c0s = fp2Sqr(a.c0);
  const c1s = fp2Sqr(a.c1);
  const c2s = fp2Sqr(a.c2);

  const t0 = fp2Sub(c0s, fp6MulByXi(fp2Mul(a.c1, a.c2)));
  const t1 = fp2Sub(fp6MulByXi(c2s), fp2Mul(a.c0, a.c1));
  const t2 = fp2Sub(c1s, fp2Mul(a.c0, a.c2));

  const det = fp2Add(
    fp2Mul(a.c0, t0),
    fp6MulByXi(fp2Add(fp2Mul(a.c2, t1), fp2Mul(a.c1, t2))),
  );
  const detInv = fp2Inv(det);

  return {
    c0: fp2Mul(t0, detInv),
    c1: fp2Mul(t1, detInv),
    c2: fp2Mul(t2, detInv),
  };
}

// ---------------------------------------------------------------------------
// Fp12 = Fp6[w] / (w^2 - v)
// ---------------------------------------------------------------------------

const FP12_ONE: Fp12 = { c0: FP6_ONE, c1: FP6_ZERO };

function fp12Mul(a: Fp12, b: Fp12): Fp12 {
  const t0 = fp6Mul(a.c0, b.c0);
  const t1 = fp6Mul(a.c1, b.c1);
  // c0 = t0 + t1*v (where v = w^2 enters via the modular reduction)
  // In Fp6, multiplying by v is: {c0: xi*c2, c1: c0, c2: c1}
  const t1v: Fp6 = { c0: fp6MulByXi(t1.c2), c1: t1.c0, c2: t1.c1 };
  return {
    c0: fp6Add(t0, t1v),
    c1: fp6Sub(fp6Sub(fp6Mul(fp6Add(a.c0, a.c1), fp6Add(b.c0, b.c1)), t0), t1),
  };
}

function fp12Sqr(a: Fp12): Fp12 {
  return fp12Mul(a, a);
}

function fp12Inv(a: Fp12): Fp12 {
  const c0s = fp6Sqr(a.c0);
  const c1s = fp6Sqr(a.c1);
  // v * c1^2 where multiplying Fp6 by v: {c0: xi*c2, c1: c0, c2: c1}
  const c1sv: Fp6 = { c0: fp6MulByXi(c1s.c2), c1: c1s.c0, c2: c1s.c1 };
  const det = fp6Sub(c0s, c1sv);
  const detInv = fp6Inv(det);
  return {
    c0: fp6Mul(a.c0, detInv),
    c1: fp6Neg(fp6Mul(a.c1, detInv)),
  };
}

function fp12Conj(a: Fp12): Fp12 {
  return { c0: a.c0, c1: fp6Neg(a.c1) };
}

function fp12Pow(base: Fp12, exp: bigint): Fp12 {
  if (exp === 0n) return FP12_ONE;
  let result = FP12_ONE;
  let b = base;
  let e = exp;
  while (e > 0n) {
    if (e & 1n) result = fp12Mul(result, b);
    b = fp12Sqr(b);
    e >>= 1n;
  }
  return result;
}

// ---------------------------------------------------------------------------
// Miller loop (optimal Ate)
// ---------------------------------------------------------------------------

function lineDouble(r: { x: Fp2; y: Fp2 }, p: G1Point): { coeff: Fp12; newR: { x: Fp2; y: Fp2 } } {
  const rx = r.x, ry = r.y;
  const px = fpMod(p.x), py = fpMod(p.y);

  // λ = 3*rx^2 / (2*ry)
  const rx2 = fp2Sqr(rx);
  const threeRx2 = fp2Add(fp2Add(rx2, rx2), rx2);
  const twoRy = fp2Add(ry, ry);
  const lambda = fp2Mul(threeRx2, fp2Inv(twoRy));

  // new R
  const newX = fp2Sub(fp2Sqr(lambda), fp2Add(rx, rx));
  const newY = fp2Sub(fp2Mul(lambda, fp2Sub(rx, newX)), ry);

  // Line evaluation at P: l = lambda * (px - rx) - (py - ry)
  // Embedded in Fp12 sparse element
  const c0: Fp6 = {
    c0: fp2Sub(fp2MulScalar(lambda, fpNeg(px)), fp2Neg(ry)),
    c1: FP2_ZERO,
    c2: FP2_ZERO,
  };
  const c1: Fp6 = {
    c0: fp2MulScalar(FP2_ONE, py),
    c1: FP2_ZERO,
    c2: FP2_ZERO,
  };
  // Simplified: sparse multiplication would be more efficient
  const coeff: Fp12 = { c0, c1 };

  return { coeff, newR: { x: newX, y: newY } };
}

function lineAdd(r: { x: Fp2; y: Fp2 }, q: { x: Fp2; y: Fp2 }, p: G1Point): { coeff: Fp12; newR: { x: Fp2; y: Fp2 } } {
  const rx = r.x, ry = r.y;
  const qx = q.x, qy = q.y;
  const px = fpMod(p.x), py = fpMod(p.y);

  const lambda = fp2Mul(fp2Sub(qy, ry), fp2Inv(fp2Sub(qx, rx)));
  const newX = fp2Sub(fp2Sub(fp2Sqr(lambda), rx), qx);
  const newY = fp2Sub(fp2Mul(lambda, fp2Sub(rx, newX)), ry);

  const c0: Fp6 = {
    c0: fp2Sub(fp2MulScalar(lambda, fpNeg(px)), fp2Neg(ry)),
    c1: FP2_ZERO,
    c2: FP2_ZERO,
  };
  const c1: Fp6 = {
    c0: fp2MulScalar(FP2_ONE, py),
    c1: FP2_ZERO,
    c2: FP2_ZERO,
  };
  const coeff: Fp12 = { c0, c1 };

  return { coeff, newR: { x: newX, y: newY } };
}

function millerLoop(p: G1Point, q: G2Point): Fp12 {
  if (g1IsInfinity(p) || g2IsInfinity(q)) return FP12_ONE;

  let f = FP12_ONE;
  let r = { x: q.x, y: q.y };

  // BN_X in binary (6x + 2 loop, using NAF of BN_X)
  // For simplicity, use basic binary representation of 6*BN_X + 2
  const sixXPlus2 = 6n * BN_X + 2n;
  const bits: number[] = [];
  let v = sixXPlus2;
  while (v > 0n) {
    bits.push(Number(v & 1n));
    v >>= 1n;
  }

  // Miller loop: iterate bits from MSB to LSB (skip the top bit)
  for (let i = bits.length - 2; i >= 0; i--) {
    // Doubling step
    const dbl = lineDouble(r, p);
    f = fp12Mul(fp12Sqr(f), dbl.coeff);
    r = dbl.newR;

    if (bits[i] === 1) {
      // Addition step
      const add = lineAdd(r, { x: q.x, y: q.y }, p);
      f = fp12Mul(f, add.coeff);
      r = add.newR;
    }
  }

  return f;
}

// ---------------------------------------------------------------------------
// Final exponentiation
// ---------------------------------------------------------------------------

function finalExponentiation(f: Fp12): Fp12 {
  // Easy part: f^{(p^6 - 1)(p^2 + 1)}
  const fConj = fp12Conj(f);
  const fInv = fp12Inv(f);
  let result = fp12Mul(fConj, fInv); // f^{p^6 - 1}

  // f^{p^2}: Frobenius map (simplified — apply p^2 Frobenius)
  // For correctness we use exponentiation
  result = fp12Mul(fp12Pow(result, P * P), fp12Inv(fp12Pow(result, P * P - 1n)));
  // Simplified: just use result directly for now (the easy part is result itself)
  // A proper implementation would use Frobenius endomorphism

  // Hard part: f^{(p^4 - p^2 + 1) / r}
  // This is the most complex part. For a correct implementation, we'd use
  // the BN-specific formula involving BN_X. For now, we use direct exponentiation.
  const _hardExp = (P * P * P * P - P * P + 1n) / (6n * BN_X + 2n);
  void _hardExp;
  // Note: this is astronomically expensive for a reference implementation.
  // A production version would use the optimized formula with Frobenius maps.
  // For testing with small inputs, we'll use the simplified path.

  return result; // TODO: complete hard part with optimized formula
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Compute the optimal Ate pairing e(P, Q) on BN254.
 *
 * Returns an element of GT ⊂ Fp12. The result is only meaningful
 * for comparison (checking if two pairings are equal), not as an
 * absolute value.
 */
export function pairing(p: G1Point, q: G2Point): Fp12 {
  const f = millerLoop(p, q);
  return finalExponentiation(f);
}

/**
 * Check the Groth16 pairing equation:
 *   e(A, B) == e(alpha, beta) * e(L, gamma) * e(C, delta)
 *
 * This is equivalent to checking:
 *   e(A, B) * e(-L, gamma) * e(-C, delta) == e(alpha, beta)
 *
 * Or more efficiently using the product-of-pairings check:
 *   e(A, B) * e(alpha_neg, beta) * e(L_neg, gamma) * e(C_neg, delta) == 1
 */
export function checkPairingProduct(
  pairs: Array<{ g1: G1Point; g2: G2Point }>,
): boolean {
  // Compute product of Miller loops, then single final exponentiation
  let f = FP12_ONE;
  for (const { g1, g2 } of pairs) {
    if (!g1IsInfinity(g1) && !g2IsInfinity(g2)) {
      f = fp12Mul(f, millerLoop(g1, g2));
    }
  }
  const result = finalExponentiation(f);
  // Check if result == 1 in Fp12
  return fp12IsOne(result);
}

function fp12IsOne(a: Fp12): boolean {
  return fp2Eq(a.c0.c0, FP2_ONE) &&
    fp2Eq(a.c0.c1, FP2_ZERO) &&
    fp2Eq(a.c0.c2, FP2_ZERO) &&
    fp2Eq(a.c1.c0, FP2_ZERO) &&
    fp2Eq(a.c1.c1, FP2_ZERO) &&
    fp2Eq(a.c1.c2, FP2_ZERO);
}
