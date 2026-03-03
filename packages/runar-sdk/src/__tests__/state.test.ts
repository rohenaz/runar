import { describe, it, expect } from 'vitest';
import { serializeState, deserializeState } from '../state.js';
import type { StateField } from 'runar-ir-schema';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeFields(...defs: { name: string; type: string; index: number }[]): StateField[] {
  return defs.map(d => ({ name: d.name, type: d.type, index: d.index }));
}

// ---------------------------------------------------------------------------
// serializeState / deserializeState roundtrip
// ---------------------------------------------------------------------------

describe('serializeState / deserializeState roundtrip', () => {
  it('roundtrips a single bigint field', () => {
    const fields = makeFields({ name: 'count', type: 'bigint', index: 0 });
    const values = { count: 42n };
    const hex = serializeState(fields, values);
    const result = deserializeState(fields, hex);
    expect(result.count).toBe(42n);
  });

  it('roundtrips a zero bigint', () => {
    const fields = makeFields({ name: 'count', type: 'bigint', index: 0 });
    const values = { count: 0n };
    const hex = serializeState(fields, values);
    const result = deserializeState(fields, hex);
    expect(result.count).toBe(0n);
  });

  it('roundtrips a negative bigint', () => {
    const fields = makeFields({ name: 'count', type: 'bigint', index: 0 });
    const values = { count: -42n };
    const hex = serializeState(fields, values);
    const result = deserializeState(fields, hex);
    expect(result.count).toBe(-42n);
  });

  it('roundtrips a large bigint', () => {
    const fields = makeFields({ name: 'count', type: 'bigint', index: 0 });
    const values = { count: 1000000000000n };
    const hex = serializeState(fields, values);
    const result = deserializeState(fields, hex);
    expect(result.count).toBe(1000000000000n);
  });

  it('roundtrips multiple fields preserving order', () => {
    const fields = makeFields(
      { name: 'a', type: 'bigint', index: 0 },
      { name: 'b', type: 'bigint', index: 1 },
      { name: 'c', type: 'bigint', index: 2 },
    );
    const values = { a: 1n, b: 2n, c: 3n };
    const hex = serializeState(fields, values);
    const result = deserializeState(fields, hex);
    expect(result.a).toBe(1n);
    expect(result.b).toBe(2n);
    expect(result.c).toBe(3n);
  });
});

// ---------------------------------------------------------------------------
// Bigint state encoding/decoding
// ---------------------------------------------------------------------------

describe('bigint state encoding/decoding', () => {
  const bigintTestCases: Array<{ label: string; value: bigint }> = [
    { label: '0', value: 0n },
    { label: '1', value: 1n },
    { label: '-1', value: -1n },
    { label: '127', value: 127n },
    { label: '128', value: 128n },
    { label: '-128', value: -128n },
    { label: '255', value: 255n },
    { label: '256', value: 256n },
    { label: '-256', value: -256n },
    { label: 'large positive', value: 9999999999n },
    { label: 'large negative', value: -9999999999n },
  ];

  for (const tc of bigintTestCases) {
    it(`roundtrips bigint ${tc.label}`, () => {
      const fields = makeFields({ name: 'v', type: 'bigint', index: 0 });
      const hex = serializeState(fields, { v: tc.value });
      const result = deserializeState(fields, hex);
      expect(result.v).toBe(tc.value);
    });
  }
});

// ---------------------------------------------------------------------------
// Boolean state encoding/decoding
// ---------------------------------------------------------------------------

describe('boolean state encoding/decoding', () => {
  it('roundtrips true', () => {
    const fields = makeFields({ name: 'flag', type: 'bool', index: 0 });
    const hex = serializeState(fields, { flag: true });
    const result = deserializeState(fields, hex);
    expect(result.flag).toBe(true);
  });

  it('roundtrips false', () => {
    const fields = makeFields({ name: 'flag', type: 'bool', index: 0 });
    const hex = serializeState(fields, { flag: false });
    const result = deserializeState(fields, hex);
    expect(result.flag).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Byte string state encoding/decoding
// ---------------------------------------------------------------------------

describe('bytes state encoding/decoding', () => {
  it('roundtrips a byte string', () => {
    const fields = makeFields({ name: 'data', type: 'bytes', index: 0 });
    const hex = serializeState(fields, { data: 'aabbccdd' });
    const result = deserializeState(fields, hex);
    expect(result.data).toBe('aabbccdd');
  });

  it('roundtrips an empty byte string', () => {
    const fields = makeFields({ name: 'data', type: 'bytes', index: 0 });
    const hex = serializeState(fields, { data: '' });
    const result = deserializeState(fields, hex);
    // Empty push: decoding may return empty string
    expect(result.data).toBe('');
  });
});

// ---------------------------------------------------------------------------
// Mixed field types
// ---------------------------------------------------------------------------

describe('mixed state fields', () => {
  it('roundtrips bigint and bool fields together', () => {
    const fields = makeFields(
      { name: 'count', type: 'bigint', index: 0 },
      { name: 'active', type: 'bool', index: 1 },
    );
    const values = { count: 100n, active: true };
    const hex = serializeState(fields, values);
    const result = deserializeState(fields, hex);
    expect(result.count).toBe(100n);
    expect(result.active).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// Fix #26: decodeScriptInt negative zero edge case
// ---------------------------------------------------------------------------

describe('decodeScriptInt negative zero edge cases', () => {
  it('decodes 0x80 (negative zero, 1 byte) as 0n', () => {
    // 0x80 is negative zero in Bitcoin Script encoding.
    // The sign bit is set but the magnitude is zero.
    // Bitcoin treats negative zero as zero.
    const fields = makeFields({ name: 'v', type: 'bigint', index: 0 });
    // Manually construct hex: push 1 byte (opcode 01) + data 80
    const hex = '0180';
    const result = deserializeState(fields, hex);
    expect(result.v).toBe(0n);
  });

  it('decodes 0x0080 (negative zero, 2 bytes) as 0n', () => {
    // 0x0080 is a 2-byte representation of negative zero:
    // byte 0 = 0x00 (magnitude), byte 1 = 0x80 (sign bit set, magnitude = 0)
    const fields = makeFields({ name: 'v', type: 'bigint', index: 0 });
    // push 2 bytes (opcode 02) + data 0080
    const hex = '020080';
    const result = deserializeState(fields, hex);
    expect(result.v).toBe(0n);
  });

  it('decodes 0x0000 (multi-byte zero, no sign bit) as 0n', () => {
    // 0x0000: two zero bytes, no sign bit → still zero
    const fields = makeFields({ name: 'v', type: 'bigint', index: 0 });
    // push 2 bytes (opcode 02) + data 0000
    const hex = '020000';
    const result = deserializeState(fields, hex);
    expect(result.v).toBe(0n);
  });

  it('still correctly decodes -1 (0x81) after the fix', () => {
    // 0x81 = magnitude 1 with sign bit set → -1
    const fields = makeFields({ name: 'v', type: 'bigint', index: 0 });
    // push 1 byte (opcode 01) + data 81
    const hex = '0181';
    const result = deserializeState(fields, hex);
    expect(result.v).toBe(-1n);
  });

  it('still correctly decodes -128 (0x8080) after the fix', () => {
    // -128: magnitude bytes = [0x80], sign byte needed → [0x80, 0x80]
    const fields = makeFields({ name: 'v', type: 'bigint', index: 0 });
    // push 2 bytes (opcode 02) + data 8080
    const hex = '028080';
    const result = deserializeState(fields, hex);
    expect(result.v).toBe(-128n);
  });
});
