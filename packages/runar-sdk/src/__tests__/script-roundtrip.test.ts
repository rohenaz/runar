/**
 * Test that Script.fromHex() → toHex() roundtrips correctly for scripts
 * containing raw ByteString state data (as used by InductiveSmartContract).
 *
 * If Script.fromHex parses raw bytes as opcodes and re-serializes differently,
 * the BIP-143 preimage scriptCode will be wrong, causing OP_CHECKSIGVERIFY
 * to fail for the OP_PUSH_TX checkPreimage.
 */
import { describe, it, expect } from 'vitest';
import { Script } from '@bsv/sdk';

describe('Script.fromHex roundtrip for inductive contract scripts', () => {
  it('roundtrips a simple script', () => {
    const hex = '51' + '52' + '93'; // OP_1 OP_2 OP_ADD
    const script = Script.fromHex(hex);
    expect(script.toHex()).toBe(hex);
  });

  it('roundtrips a script with OP_RETURN and push-data encoded state', () => {
    // codePart + OP_RETURN + push-encoded state
    const codePart = '51' + '52' + '93'; // OP_1 OP_2 OP_ADD
    const opReturn = '6a';
    const pushData33 = '21' + '02' + 'aa'.repeat(32); // 33-byte PubKey push
    const pushData8 = '08' + '6400000000000000'; // 8-byte NUM2BIN(100)
    const hex = codePart + opReturn + pushData33 + pushData8;
    const script = Script.fromHex(hex);
    expect(script.toHex()).toBe(hex);
  });

  it('roundtrips a script with raw ByteString state (zero sentinel)', () => {
    // This simulates the inductive contract state layout:
    // codePart + OP_RETURN + push-encoded(owner) + push-encoded(balance) +
    // RAW 36 bytes (genesis) + RAW 36 bytes (parent) + RAW 36 bytes (grandparent)
    const codePart = '51' + '52' + '93'; // OP_1 OP_2 OP_ADD
    const opReturn = '6a';
    const pushOwner = '21' + '02' + 'aa'.repeat(32); // 33-byte PubKey
    const pushBalance = '08' + '6400000000000000'; // 8-byte NUM2BIN
    const pushTokenId = '05' + 'deadbeef00'; // 5-byte token ID
    const rawGenesis = '00'.repeat(36); // zero sentinel
    const rawParent = '00'.repeat(36);
    const rawGrandparent = '00'.repeat(36);

    const hex = codePart + opReturn + pushOwner + pushBalance + pushTokenId +
      rawGenesis + rawParent + rawGrandparent;
    const script = Script.fromHex(hex);
    const roundtripped = script.toHex();

    // This is the critical check: raw bytes must be preserved exactly
    expect(roundtripped).toBe(hex);
  });

  it('roundtrips raw ByteString state with non-zero outpoints', () => {
    const codePart = 'ab' + '76'; // OP_CODESEPARATOR OP_DUP
    const opReturn = '6a';
    const pushOwner = '21' + '02' + 'bb'.repeat(32);
    const pushBalance = '08' + 'e803000000000000'; // NUM2BIN(1000)
    const pushTokenId = '08' + '4142434445464748'; // ABCDEFGH

    // Non-zero outpoints with bytes that could be misinterpreted as opcodes
    // First byte 0xa1 is OP_NOP (or invalid), 0x01 is "push 1 byte", etc.
    const rawGenesis = 'a1b2c3d4'.repeat(9); // 36 bytes
    const rawParent = '0102030405060708'.repeat(4) + '09101112'; // 36 bytes
    const rawGrandparent = 'ff'.repeat(36); // 36 bytes

    const hex = codePart + opReturn + pushOwner + pushBalance + pushTokenId +
      rawGenesis + rawParent + rawGrandparent;
    const script = Script.fromHex(hex);
    const roundtripped = script.toHex();

    expect(roundtripped).toBe(hex);
  });

  it('roundtrips a large script (simulating inductive contract scriptCode)', () => {
    // Build a ~70KB script to simulate inductive contract scriptCode
    // after OP_CODESEPARATOR
    const parts: string[] = [];
    // Lots of opcodes to reach ~70KB
    for (let i = 0; i < 35000; i++) {
      parts.push('76'); // OP_DUP (common filler)
    }
    // OP_RETURN + state with raw ByteString fields
    parts.push('6a');
    parts.push('21' + '02' + 'cc'.repeat(32)); // PubKey
    parts.push('08' + '0100000000000000'); // bigint
    parts.push('00'.repeat(36)); // raw ByteString
    parts.push('dd'.repeat(36)); // raw ByteString
    parts.push('ee'.repeat(36)); // raw ByteString

    const hex = parts.join('');
    const script = Script.fromHex(hex);
    const roundtripped = script.toHex();

    // For a large script, verify at least that the length is the same
    // and the hex matches
    expect(roundtripped.length).toBe(hex.length);
    expect(roundtripped).toBe(hex);
  });
});
