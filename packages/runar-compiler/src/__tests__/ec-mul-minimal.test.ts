/**
 * Minimal ecMul test — verifies ecMul codegen via BSV SDK interpreter.
 * Args are pushed in declaration order (first param first → bottom of stack).
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

function encodePushInt(n: number): string {
  if (n === 0) return '00';
  if (n >= 1 && n <= 16) return (0x50 + n).toString(16);
  const hex = n.toString(16).padStart(2, '0');
  return '01' + hex;
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

// Multi-method contract for proper method dispatch
const MULTI_METHOD_MUL = `
import { SmartContract, assert, ecMul, ecPointX, ecPointY } from 'runar-lang';
import type { Point } from 'runar-lang';

class EcMulTest extends SmartContract {
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
  public checkIdentity() {
    const result = ecMul(this.pt, 1n);
    assert(ecPointX(result) === ecPointX(this.pt));
    assert(ecPointY(result) === ecPointY(this.pt));
  }
}
`;

describe('ecMul script execution', () => {
  let lockingHex: string;

  it('compiles contract', () => {
    const result = compile(MULTI_METHOD_MUL, {
      fileName: 'EcMulTest.runar.ts',
      constructorArgs: { pt: PT_HEX },
    });
    expect(result.artifact).toBeTruthy();
    lockingHex = result.artifact!.script;
    console.log('Contract size:', lockingHex.length / 2, 'bytes');
  });

  // Args pushed in DECLARATION order: scalar, expectedX, expectedY, then method index
  it('ecMul(G, 7) via checkMul (method 0)', () => {
    const scalar = 7n;
    const exX = BigInt('0x5cbdf0646e5db4eaa398f365f2ea7a0e3d419b7e0330e39ce92bddedcac4f9bc');
    const exY = BigInt('0x6aebca40ba255960a3178d6d861a54dba813d0b813fde7b5a5082628087264da');

    // Push in declaration order: scalar, exX, exY, then method index
    const unlocking =
      encodePushScriptNum(scalar) +
      encodePushScriptNum(exX) +
      encodePushScriptNum(exY) +
      encodePushInt(0); // method 0

    const spend = makeSpend(lockingHex, unlocking);
    try {
      expect(spend.validate()).toBe(true);
    } catch (e: any) {
      console.log('ecMul(G,7) error:', e.message?.slice(0, 500));
      throw e;
    }
  });

  it('ecMul(G, 2) via checkMul (method 0)', () => {
    const scalar = 2n;
    const exX = BigInt('0xc6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5');
    const exY = BigInt('0x1ae168fea63dc339a3c58419466ceaeef7f632653266d0e1236431a950cfe52a');

    const unlocking =
      encodePushScriptNum(scalar) +
      encodePushScriptNum(exX) +
      encodePushScriptNum(exY) +
      encodePushInt(0);

    const spend = makeSpend(lockingHex, unlocking);
    try {
      expect(spend.validate()).toBe(true);
    } catch (e: any) {
      console.log('ecMul(G,2) error:', e.message?.slice(0, 500));
      throw e;
    }
  });

  it('ecMul(G, 1) via checkMul (method 0)', () => {
    const scalar = 1n;
    const exX = BigInt('0x' + GEN_X);
    const exY = BigInt('0x' + GEN_Y);

    const unlocking =
      encodePushScriptNum(scalar) +
      encodePushScriptNum(exX) +
      encodePushScriptNum(exY) +
      encodePushInt(0);

    const spend = makeSpend(lockingHex, unlocking);
    try {
      expect(spend.validate()).toBe(true);
    } catch (e: any) {
      console.log('ecMul(G,1) error:', e.message?.slice(0, 500));
      throw e;
    }
  });

  it('checkIdentity (method 1) — ecMul(G, 1) compared to G', () => {
    const unlocking = encodePushInt(1); // method 1

    const spend = makeSpend(lockingHex, unlocking);
    try {
      expect(spend.validate()).toBe(true);
    } catch (e: any) {
      console.log('checkIdentity error:', e.message?.slice(0, 500));
      throw e;
    }
  });
});
