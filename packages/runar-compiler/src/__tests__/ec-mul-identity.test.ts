/**
 * Reproducer for ecMul scalar=1 failure (Go VM: "n is larger than length of array").
 * Tests whether BSV SDK JS interpreter also fails.
 */
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import { compile } from '../index.js';
import {
  LockingScript,
  UnlockingScript,
  Spend,
} from '@bsv/sdk';

const PROJECT_ROOT = resolve(import.meta.dirname, '..', '..', '..', '..');

function encodePushInt(n: number): string {
  if (n === 0) return '00';
  if (n >= 1 && n <= 16) return (0x50 + n).toString(16);
  const hex = n.toString(16).padStart(2, '0');
  return '01' + hex;
}

/** Push a script number in MINIMALDATA-compliant form. */
function encodePushScriptNum(n: bigint): string {
  // OP_0 for 0
  if (n === 0n) return '00';
  // OP_1NEGATE for -1
  if (n === -1n) return '4f';
  // OP_1 through OP_16 for 1-16
  if (n >= 1n && n <= 16n) return (0x50 + Number(n)).toString(16);
  // Otherwise encode as push data
  const hex = bigintToScriptNum(n);
  return encodePushBytes(hex);
}

function encodePushBytes(hex: string): string {
  const len = hex.length / 2;
  if (len <= 75) {
    return len.toString(16).padStart(2, '0') + hex;
  }
  if (len <= 255) {
    return '4c' + len.toString(16).padStart(2, '0') + hex;
  }
  return '4d' + (len & 0xff).toString(16).padStart(2, '0') +
    ((len >> 8) & 0xff).toString(16).padStart(2, '0') + hex;
}

// secp256k1 generator point
const GEN_X = '79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798';
const GEN_Y = '483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8';
const PT_HEX = GEN_X + GEN_Y;

// Known multiples of G for testing
const KNOWN_MULTIPLES: Record<number, { x: string; y: string }> = {
  2: {
    x: 'c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5',
    y: '1ae168fea63dc339a3c58419466ceaeef7f632653266d0e1236431a950cfe52a',
  },
  7: {
    x: '5cbdf0646e5db4eaa398f365f2ea7a0e3d419b7e0330e39ce92bddedcac4f9bc',
    y: '6aebca40ba255960a3178d6d861a54dba813d0b813fde7b5a5082628087264da',
  },
};

/** Convert a BigInt to a minimal Bitcoin Script number (LE signed magnitude) as hex. */
function bigintToScriptNum(n: bigint): string {
  if (n === 0n) return '00';
  const negative = n < 0n;
  let abs = negative ? -n : n;
  const bytes: number[] = [];
  while (abs > 0n) {
    bytes.push(Number(abs & 0xffn));
    abs >>= 8n;
  }
  if (bytes[bytes.length - 1]! & 0x80) {
    bytes.push(negative ? 0x80 : 0x00);
  } else if (negative) {
    bytes[bytes.length - 1]! |= 0x80;
  }
  return bytes.map(b => b.toString(16).padStart(2, '0')).join('');
}

function makeSpend(lockingHex: string, unlockingHex: string): Spend {
  return new Spend({
    sourceTXID: 'aa'.repeat(32),
    sourceOutputIndex: 0,
    sourceSatoshis: 1,
    lockingScript: LockingScript.fromHex(lockingHex),
    transactionVersion: 1,
    otherInputs: [],
    inputIndex: 0,
    unlockingScript: UnlockingScript.fromHex(unlockingHex),
    outputs: [{ lockingScript: LockingScript.fromHex('6a'), satoshis: 1 }],
    inputSequence: 0xffffffff,
    lockTime: 0,
  });
}

describe('ecMul scalar=1 (identity)', () => {
  let lockingHex: string;

  it('compiles ECPrimitives contract', () => {
    const source = readFileSync(
      resolve(PROJECT_ROOT, 'conformance/tests/ec-primitives/ec-primitives.runar.ts'),
      'utf-8',
    );
    const result = compile(source, {
      fileName: 'ec-primitives.runar.ts',
      constructorArgs: { pt: PT_HEX },
    });
    expect(result.artifact).toBeTruthy();
    lockingHex = result.artifact!.script;
    console.log('Locking script size:', lockingHex.length / 2, 'bytes');
  });

  // Test checkMul (method 6) with different scalars to find the boundary
  for (const [scalar, coords] of Object.entries(KNOWN_MULTIPLES)) {
    it(`checkMul with scalar=${scalar} should pass`, () => {
      const exXNum = BigInt('0x' + coords.x);
      const exYNum = BigInt('0x' + coords.y);
      const scalarN = BigInt(scalar);

      const unlocking =
        encodePushScriptNum(scalarN) +
        encodePushScriptNum(exXNum) +
        encodePushScriptNum(exYNum) +
        encodePushInt(6);

      const spend = makeSpend(lockingHex, unlocking);
      expect(spend.validate()).toBe(true);
    });
  }

  // Test checkMul with scalar=1 (should return G itself) — method 6
  it('checkMul with scalar=1 (using known expected values)', () => {
    const exXNum = BigInt('0x' + GEN_X);
    const exYNum = BigInt('0x' + GEN_Y);

    const unlocking =
      encodePushScriptNum(1n) +
      encodePushScriptNum(exXNum) +
      encodePushScriptNum(exYNum) +
      encodePushInt(6);

    const spend = makeSpend(lockingHex, unlocking);
    try {
      const valid = spend.validate();
      console.log('checkMul(1) validate():', valid);
      expect(valid).toBe(true);
    } catch (e: any) {
      console.log('checkMul(1) error:', e.message?.slice(0, 300));
      throw e;
    }
  });

  it('checkMulIdentity (scalar=1 via method 10) should pass', () => {
    const unlocking = encodePushInt(10);
    const spend = makeSpend(lockingHex, unlocking);
    try {
      const valid = spend.validate();
      console.log('checkMulIdentity validate():', valid);
      expect(valid).toBe(true);
    } catch (e: any) {
      console.log('checkMulIdentity error:', e.message?.slice(0, 300));
      throw e;
    }
  });
});
