/**
 * End-to-end OP_PUSH_TX verification using BSV SDK's Spend.validate().
 * Builds a locking script with OP_CODESEPARATOR + checkPreimage,
 * computes the preimage/signature, and verifies via Spend.
 *
 * Tests both small and large (>65KB) scripts to catch varint encoding
 * or Script roundtrip issues specific to inductive contract sizes.
 */
import { describe, it, expect } from 'vitest';
import {
  Transaction,
  Script,
  LockingScript,
  UnlockingScript,
  Spend,
} from '@bsv/sdk';
import { computeOpPushTx } from '../oppushtx.js';

// Compressed secp256k1 generator point G (OP_PUSH_TX public key)
const G_HEX =
  '0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798';

function encodePushData(hex: string): string {
  const len = hex.length / 2;
  if (len === 0) return '00'; // OP_0
  if (len <= 75) return len.toString(16).padStart(2, '0') + hex;
  if (len <= 0xff) return '4c' + len.toString(16).padStart(2, '0') + hex;
  if (len <= 0xffff) {
    const lo = (len & 0xff).toString(16).padStart(2, '0');
    const hi = ((len >> 8) & 0xff).toString(16).padStart(2, '0');
    return '4d' + lo + hi + hex;
  }
  const b0 = (len & 0xff).toString(16).padStart(2, '0');
  const b1 = ((len >> 8) & 0xff).toString(16).padStart(2, '0');
  const b2 = ((len >> 16) & 0xff).toString(16).padStart(2, '0');
  const b3 = ((len >> 24) & 0xff).toString(16).padStart(2, '0');
  return '4e' + b0 + b1 + b2 + b3 + hex;
}

function encodeVarInt(n: number): string {
  if (n < 253) return n.toString(16).padStart(2, '0');
  if (n < 65536) {
    return 'fd' + (n & 0xff).toString(16).padStart(2, '0') +
      ((n >> 8) & 0xff).toString(16).padStart(2, '0');
  }
  return 'fe' + (n & 0xff).toString(16).padStart(2, '0') +
    ((n >> 8) & 0xff).toString(16).padStart(2, '0') +
    ((n >> 16) & 0xff).toString(16).padStart(2, '0') +
    ((n >> 24) & 0xff).toString(16).padStart(2, '0');
}

function toLittleEndian64(n: number): string {
  const buf = BigInt(n).toString(16).padStart(16, '0');
  return buf.match(/.{2}/g)!.reverse().join('');
}

/**
 * Build a minimal locking script that does:
 *   [filler opcodes]
 *   OP_CODESEPARATOR
 *   [preimage on stack] [sig on stack]
 *   <G>
 *   OP_CHECKSIGVERIFY
 *   OP_TRUE
 *
 * The unlocking script pushes: <sig> <preimage>
 */
function buildCheckPreimageLockingScript(
  fillerBeforeCodeSep: number,
  fillerAfterCodeSep: number = 0,
): {
  lockingHex: string;
  codeSepOffset: number;
} {
  // Filler before OP_CODESEPARATOR: OP_1 OP_DROP repeated
  const preFillerPairs = Math.floor(fillerBeforeCodeSep / 2);
  let preFiller = '';
  for (let i = 0; i < preFillerPairs; i++) {
    preFiller += '5175'; // OP_1 OP_DROP
  }
  if (fillerBeforeCodeSep % 2 === 1) {
    preFiller += '61'; // OP_NOP
  }

  const codeSepOffset = preFiller.length / 2;

  // After OP_CODESEPARATOR:
  // Stack: [..., sig, preimage]  (sig pushed first, preimage on top)
  // We need: preimage sig G CHECKSIGVERIFY TRUE
  //
  // Stack ops:
  //   OP_SWAP          — swap preimage and sig: now sig on top, preimage below
  //   <push G>         — push generator point
  //   OP_CHECKSIGVERIFY — verify sig against sighash using pubkey G
  //   OP_DROP          — drop the preimage
  //   [filler: OP_1 OP_DROP repeated]  — dead code to inflate scriptCode
  //   OP_TRUE          — push 1 for script success

  // Post-checkPreimage filler (after CHECKSIGVERIFY, so it just needs to not fail)
  let postFiller = '';
  const postFillerPairs = Math.floor(fillerAfterCodeSep / 2);
  for (let i = 0; i < postFillerPairs; i++) {
    postFiller += '5175'; // OP_1 OP_DROP
  }
  if (fillerAfterCodeSep % 2 === 1) {
    postFiller += '61'; // OP_NOP
  }

  const body =
    'ab' +                    // OP_CODESEPARATOR
    '7c' +                    // OP_SWAP (bring sig to top)
    encodePushData(G_HEX) +  // push G (33 bytes)
    'ad' +                    // OP_CHECKSIGVERIFY
    '75' +                    // OP_DROP (drop preimage)
    postFiller +              // filler after verify (inflates scriptCode)
    '51';                     // OP_TRUE

  return {
    lockingHex: preFiller + body,
    codeSepOffset,
  };
}

/**
 * Build a raw transaction hex with one input and one output.
 */
function buildRawTx(
  prevTxid: string,
  prevVout: number,
  unlockScript: string,
  outputScript: string,
  outputSatoshis: number,
): string {
  let tx = '01000000'; // version

  // 1 input
  tx += '01';
  tx += prevTxid.match(/.{2}/g)!.reverse().join('');
  tx += prevVout.toString(16).padStart(8, '0').match(/.{2}/g)!.reverse().join('');
  // scriptSig
  const unlockLen = unlockScript.length / 2;
  tx += encodeVarInt(unlockLen);
  tx += unlockScript;
  tx += 'ffffffff'; // sequence

  // 1 output
  tx += '01';
  tx += toLittleEndian64(outputSatoshis);
  const outLen = outputScript.length / 2;
  tx += encodeVarInt(outLen);
  tx += outputScript;

  tx += '00000000'; // locktime
  return tx;
}

describe('OP_PUSH_TX via Spend.validate()', () => {
  it('validates checkPreimage for a small script', () => {
    const prevTxid = 'aa'.repeat(32);
    const prevVout = 0;
    const prevSatoshis = 100000;
    const outputScript = '76a914' + 'bb'.repeat(20) + '88ac';
    const outputSatoshis = 90000;

    const { lockingHex, codeSepOffset } = buildCheckPreimageLockingScript(10);

    // Build a placeholder TX (empty unlock) to compute the preimage
    const placeholderTx = buildRawTx(
      prevTxid, prevVout, '', outputScript, outputSatoshis,
    );

    const { sigHex, preimageHex } = computeOpPushTx(
      placeholderTx, 0, lockingHex, prevSatoshis, codeSepOffset,
    );

    // Build the unlocking script: <sig> <preimage>
    // (sig pushed first onto stack, then preimage on top)
    const unlockScript = encodePushData(sigHex) + encodePushData(preimageHex);

    // Rebuild TX with the real unlock
    const finalTx = buildRawTx(
      prevTxid, prevVout, unlockScript, outputScript, outputSatoshis,
    );

    // Verify with Spend
    const tx = Transaction.fromHex(finalTx);
    const spend = new Spend({
      sourceTXID: prevTxid,
      sourceOutputIndex: prevVout,
      sourceSatoshis: prevSatoshis,
      lockingScript: LockingScript.fromHex(lockingHex),
      transactionVersion: tx.version,
      otherInputs: [],
      inputIndex: 0,
      unlockingScript: UnlockingScript.fromHex(unlockScript),
      outputs: tx.outputs.map((o) => ({
        lockingScript: o.lockingScript,
        satoshis: o.satoshis ?? 0,
      })),
      inputSequence: tx.inputs[0]!.sequence,
      lockTime: tx.lockTime,
    });

    const valid = spend.validate();
    expect(valid).toBe(true);
  });

  it('validates checkPreimage for a large script (>65KB, simulating inductive)', () => {
    const prevTxid = 'dd'.repeat(32);
    const prevVout = 0;
    const prevSatoshis = 500000;
    const outputScript = '76a914' + 'ee'.repeat(20) + '88ac';
    const outputSatoshis = 490000;

    // Use 70000 bytes of filler AFTER OP_CODESEPARATOR to push scriptCode > 65535
    const { lockingHex, codeSepOffset } = buildCheckPreimageLockingScript(10, 70000);

    // Verify the scriptCode after OP_CODESEPARATOR is > 65535 bytes
    const scriptCodeAfterCodeSep = lockingHex.slice((codeSepOffset + 1) * 2);
    expect(scriptCodeAfterCodeSep.length / 2).toBeGreaterThan(65535);

    // Build a placeholder TX
    const placeholderTx = buildRawTx(
      prevTxid, prevVout, '', outputScript, outputSatoshis,
    );

    const { sigHex, preimageHex } = computeOpPushTx(
      placeholderTx, 0, lockingHex, prevSatoshis, codeSepOffset,
    );

    // Build unlocking script
    const unlockScript = encodePushData(sigHex) + encodePushData(preimageHex);

    // Rebuild TX with the real unlock
    const finalTx = buildRawTx(
      prevTxid, prevVout, unlockScript, outputScript, outputSatoshis,
    );

    // Verify with Spend
    const tx = Transaction.fromHex(finalTx);
    const spend = new Spend({
      sourceTXID: prevTxid,
      sourceOutputIndex: prevVout,
      sourceSatoshis: prevSatoshis,
      lockingScript: LockingScript.fromHex(lockingHex),
      transactionVersion: tx.version,
      otherInputs: [],
      inputIndex: 0,
      unlockingScript: UnlockingScript.fromHex(unlockScript),
      outputs: tx.outputs.map((o) => ({
        lockingScript: o.lockingScript,
        satoshis: o.satoshis ?? 0,
      })),
      inputSequence: tx.inputs[0]!.sequence,
      lockTime: tx.lockTime,
    });

    const valid = spend.validate();
    expect(valid).toBe(true);
  });
});
