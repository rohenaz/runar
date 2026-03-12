/**
 * BN254 base field Fp arithmetic — reference implementation.
 *
 * All operations are mod p where p is the BN254 base field prime.
 * Uses native bigint arithmetic. This module serves as the ground truth
 * for testing the Bitcoin Script codegen.
 */

import { P, P_MINUS_2 } from './constants.js';
import type { Fp } from '../types.js';

/** Reduce to [0, p). */
export function fpMod(a: bigint): Fp {
  const r = a % P;
  return r < 0n ? r + P : r;
}

/** Addition: (a + b) mod p */
export function fpAdd(a: Fp, b: Fp): Fp {
  return fpMod(a + b);
}

/** Subtraction: (a - b) mod p */
export function fpSub(a: Fp, b: Fp): Fp {
  return fpMod(a - b);
}

/** Multiplication: (a * b) mod p */
export function fpMul(a: Fp, b: Fp): Fp {
  return fpMod(a * b);
}

/** Negation: (-a) mod p */
export function fpNeg(a: Fp): Fp {
  return a === 0n ? 0n : P - a;
}

/** Squaring: a^2 mod p */
export function fpSqr(a: Fp): Fp {
  return fpMod(a * a);
}

/** Modular exponentiation: a^exp mod p (binary method). */
export function fpPow(base: Fp, exp: bigint): Fp {
  if (exp === 0n) return 1n;
  let result = 1n;
  let b = fpMod(base);
  let e = exp;
  while (e > 0n) {
    if (e & 1n) result = fpMul(result, b);
    b = fpSqr(b);
    e >>= 1n;
  }
  return result;
}

/** Modular inverse: a^{-1} mod p (Fermat's little theorem). */
export function fpInv(a: Fp): Fp {
  if (a === 0n) throw new Error('fpInv: division by zero');
  return fpPow(a, P_MINUS_2);
}

/** Division: a / b mod p */
export function fpDiv(a: Fp, b: Fp): Fp {
  return fpMul(a, fpInv(b));
}

/** Check equality. */
export function fpEq(a: Fp, b: Fp): boolean {
  return fpMod(a) === fpMod(b);
}

/** Check if a is zero. */
export function fpIsZero(a: Fp): boolean {
  return fpMod(a) === 0n;
}

/** Convert bigint to 32-byte big-endian Uint8Array. */
export function fpToBytes(a: Fp): Uint8Array {
  const bytes = new Uint8Array(32);
  let v = fpMod(a);
  for (let i = 31; i >= 0; i--) {
    bytes[i] = Number(v & 0xffn);
    v >>= 8n;
  }
  return bytes;
}

/** Convert 32-byte big-endian Uint8Array to Fp. */
export function fpFromBytes(bytes: Uint8Array): Fp {
  let result = 0n;
  for (let i = 0; i < 32; i++) {
    result = (result << 8n) | BigInt(bytes[i]!);
  }
  return fpMod(result);
}

/** Encode as hex string (64 chars, zero-padded). */
export function fpToHex(a: Fp): string {
  return fpMod(a).toString(16).padStart(64, '0');
}
