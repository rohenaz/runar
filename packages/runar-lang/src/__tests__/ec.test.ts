// ---------------------------------------------------------------------------
// Tests for runar-lang/ec.ts — secp256k1 elliptic curve constants
// ---------------------------------------------------------------------------

import { describe, it, expect } from 'vitest';
import { EC_P, EC_N, EC_G } from '../ec.js';

describe('EC_P (secp256k1 field prime)', () => {
  it('equals 2^256 - 2^32 - 977', () => {
    const expected = 2n ** 256n - 2n ** 32n - 977n;
    expect(EC_P).toBe(expected);
  });

  it('is the known secp256k1 prime hex value', () => {
    expect(EC_P).toBe(
      0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2Fn,
    );
  });

  it('is a positive 256-bit number', () => {
    expect(EC_P > 0n).toBe(true);
    expect(EC_P < 2n ** 256n).toBe(true);
    expect(EC_P >= 2n ** 255n).toBe(true);
  });
});

describe('EC_N (secp256k1 group order)', () => {
  it('equals the known secp256k1 curve order', () => {
    expect(EC_N).toBe(
      0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141n,
    );
  });

  it('is less than EC_P', () => {
    expect(EC_N < EC_P).toBe(true);
  });

  it('is a positive 256-bit number', () => {
    expect(EC_N > 0n).toBe(true);
    expect(EC_N < 2n ** 256n).toBe(true);
  });
});

describe('EC_G (secp256k1 generator point)', () => {
  it('is 128 hex characters (64 bytes)', () => {
    const g = EC_G as unknown as string;
    expect(g).toHaveLength(128);
  });

  it('contains only valid hex characters', () => {
    const g = EC_G as unknown as string;
    expect(g).toMatch(/^[0-9A-Fa-f]+$/);
  });

  it('has the correct x-coordinate (Gx)', () => {
    const g = EC_G as unknown as string;
    const gx = g.slice(0, 64);
    // Known secp256k1 generator x-coordinate
    expect(gx.toUpperCase()).toBe(
      '79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798',
    );
  });

  it('has the correct y-coordinate (Gy)', () => {
    const g = EC_G as unknown as string;
    const gy = g.slice(64, 128);
    // Known secp256k1 generator y-coordinate
    expect(gy.toUpperCase()).toBe(
      '483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8',
    );
  });

  it('Gx and Gy satisfy y^2 = x^3 + 7 (mod p)', () => {
    const g = EC_G as unknown as string;
    const gx = BigInt('0x' + g.slice(0, 64));
    const gy = BigInt('0x' + g.slice(64, 128));

    // Modular exponentiation helper
    function modPow(base: bigint, exp: bigint, mod: bigint): bigint {
      let result = 1n;
      base = ((base % mod) + mod) % mod;
      while (exp > 0n) {
        if (exp & 1n) result = (result * base) % mod;
        exp >>= 1n;
        base = (base * base) % mod;
      }
      return result;
    }

    const lhs = modPow(gy, 2n, EC_P);
    const rhs = (modPow(gx, 3n, EC_P) + 7n) % EC_P;
    expect(lhs).toBe(rhs);
  });
});
