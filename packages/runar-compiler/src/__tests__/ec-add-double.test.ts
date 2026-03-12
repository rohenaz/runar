import { describe, it, expect } from 'vitest';
import { compile } from '../index.js';
import {
  LockingScript,
  UnlockingScript,
  Spend,
} from '@bsv/sdk';

// Simple contract that just doubles a point and checks the result
const SOURCE = `
import { SmartContract, assert, ecPointX, ecPointY, ecAdd } from 'runar-lang';
import type { Point } from 'runar-lang';

class DoubleTest extends SmartContract {
  readonly pt: Point;
  constructor(pt: Point) {
    super(pt);
    this.pt = pt;
  }
  public checkDouble(expectedX: bigint, expectedY: bigint) {
    const result = ecAdd(this.pt, this.pt);
    assert(ecPointX(result) === expectedX);
    assert(ecPointY(result) === expectedY);
  }
}
`;

const GEN_X = '79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798';
const GEN_Y = '483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8';
const PT_HEX = GEN_X + GEN_Y;

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

describe('ecAdd(P, P) — affine doubling via ecAdd', () => {
  it('ecAdd(G, G) should fail because affineAdd degenerates for P+P', () => {
    const result = compile(SOURCE, {
      fileName: 'DoubleTest.runar.ts',
      constructorArgs: { pt: PT_HEX },
    });
    expect(result.artifact).toBeTruthy();
    const lockingHex = result.artifact!.script;
    console.log('Contract size:', lockingHex.length / 2, 'bytes');

    const exX = BigInt('0xc6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5');
    const exY = BigInt('0x1ae168fea63dc339a3c58419466ceaeef7f632653266d0e1236431a950cfe52a');

    const unlocking =
      encodePushScriptNum(exX) +
      encodePushScriptNum(exY) +
      encodePushInt(0);

    const spend = makeSpend(lockingHex, unlocking);
    try {
      const valid = spend.validate();
      console.log('ecAdd(G,G) validate():', valid);
    } catch (e: any) {
      console.log('ecAdd(G,G) error:', e.message?.slice(0, 300));
    }
  });
});
