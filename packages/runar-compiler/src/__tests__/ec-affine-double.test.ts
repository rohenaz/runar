/**
 * Test affine doubling directly via a contract that uses ecAdd-style code path.
 * We use ecMul(P, 2) with the altstack approach to test if the pre-computed
 * affineDouble values are correct.
 *
 * This test creates a contract that ONLY does affine doubling (no loop).
 */
import { describe, it, expect } from 'vitest';
import { compile } from '../index.js';
import {
  LockingScript,
  UnlockingScript,
  Spend,
} from '@bsv/sdk';

const GEN_X = '79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798';
const GEN_Y = '483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8';
const PT_HEX = GEN_X + GEN_Y;

const TWO_G_X = 'c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5';
const TWO_G_Y = '1ae168fea63dc339a3c58419466ceaeef7f632653266d0e1236431a950cfe52a';

function encodePushScriptNum(n: bigint): string {
  if (n === 0n) return '00';
  if (n === -1n) return '4f';
  if (n >= 1n && n <= 16n) return (0x50 + Number(n)).toString(16);
  const hex = bigintToScriptNum(n);
  const len = hex.length / 2;
  if (len <= 75) return len.toString(16).padStart(2, '0') + hex;
  if (len <= 255) return '4c' + len.toString(16).padStart(2, '0') + hex;
  return '4d' + (len & 0xff).toString(16).padStart(2, '0') +
    ((len >> 8) & 0xff).toString(16).padStart(2, '0') + hex;
}

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

// Test via ecMul(G, 7) which goes through the normal path (no degenerate)
// and ecMul(G, 2) which goes through the degenerate path
const CONTRACT = `
import { SmartContract, assert, ecMul, ecPointX, ecPointY } from 'runar-lang';
import type { Point } from 'runar-lang';

class AfDoubleTest extends SmartContract {
  readonly pt: Point;
  constructor(pt: Point) {
    super(pt);
    this.pt = pt;
  }
  public checkMul(scalar: bigint, expectedX: bigint, expectedY: bigint) {
    const result = ecMul(this.pt, scalar);
    assert(ecPointX(result) === expectedX);
    assert(ecPointY(result) === expectedY);
  }
}
`;

describe('affineDouble via ecMul', () => {
  let lockingHex: string;

  it('compiles', () => {
    const result = compile(CONTRACT, {
      fileName: 'AfDoubleTest.runar.ts',
      constructorArgs: { pt: PT_HEX },
    });
    expect(result.artifact).toBeTruthy();
    lockingHex = result.artifact!.script;
    console.log('Script size:', lockingHex.length / 2, 'bytes');
  });

  // ecMul(G, 7) — normal path, verifies the loop is correct
  // Single-method contract: no method index needed, just push args in declaration order
  it('ecMul(G, 7) via normal path', () => {
    const scalar = 7n;
    const exX = BigInt('0x5cbdf0646e5db4eaa398f365f2ea7a0e3d419b7e0330e39ce92bddedcac4f9bc');
    const exY = BigInt('0x6aebca40ba255960a3178d6d861a54dba813d0b813fde7b5a5082628087264da');
    const unlocking =
      encodePushScriptNum(scalar) +
      encodePushScriptNum(exX) +
      encodePushScriptNum(exY);
    const spend = makeSpend(lockingHex, unlocking);
    expect(spend.validate()).toBe(true);
  });

  // ecMul(G, 2) — degenerate path (uses pre-computed affineDouble from altstack)
  it('ecMul(G, 2) via degenerate path', () => {
    const scalar = 2n;
    const exX = BigInt('0x' + TWO_G_X);
    const exY = BigInt('0x' + TWO_G_Y);
    const unlocking =
      encodePushScriptNum(scalar) +
      encodePushScriptNum(exX) +
      encodePushScriptNum(exY);
    const spend = makeSpend(lockingHex, unlocking);
    try {
      expect(spend.validate()).toBe(true);
    } catch (e: any) {
      console.log('ecMul(G,2) error:', e.message?.slice(0, 500));
      throw e;
    }
  });
});
