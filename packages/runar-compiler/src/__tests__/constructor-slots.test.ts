import { describe, it, expect } from 'vitest';
import { compile } from '../index.js';

// ---------------------------------------------------------------------------
// Contract sources
// ---------------------------------------------------------------------------

const P2PKH_SOURCE = `
class P2PKH extends SmartContract {
  readonly pk: PubKey;

  constructor(pk: PubKey) {
    super(pk);
    this.pk = pk;
  }

  public unlock(sig: Sig) {
    assert(checkSig(sig, this.pk));
  }
}
`;

const ESCROW_SOURCE = `
class Escrow extends SmartContract {
  readonly buyer: PubKey;
  readonly seller: PubKey;
  readonly arbiter: PubKey;

  constructor(buyer: PubKey, seller: PubKey, arbiter: PubKey) {
    super(buyer, seller, arbiter);
    this.buyer = buyer;
    this.seller = seller;
    this.arbiter = arbiter;
  }

  public release(sig: Sig) {
    assert(checkSig(sig, this.buyer));
  }

  public refund(sig: Sig) {
    assert(checkSig(sig, this.seller));
  }
}
`;

const HASH_LOCK_SOURCE = `
class HashLock extends SmartContract {
  readonly hashValue: Sha256;

  constructor(hashValue: Sha256) {
    super(hashValue);
    this.hashValue = hashValue;
  }

  public unlock(preimage: ByteString) {
    assert(sha256(preimage) === this.hashValue);
  }
}
`;

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('constructorSlots in compiled artifact', () => {
  it('P2PKH artifact has exactly one constructorSlot for pk', () => {
    const result = compile(P2PKH_SOURCE);
    expect(result.success).toBe(true);

    const slots = result.artifact!.constructorSlots;
    expect(slots).toBeDefined();
    expect(slots).toHaveLength(1);
    expect(slots![0]!.paramIndex).toBe(0);
    // byteOffset should point to a valid position in the script hex
    const hexOffset = slots![0]!.byteOffset * 2;
    expect(hexOffset).toBeGreaterThanOrEqual(0);
    expect(hexOffset).toBeLessThan(result.artifact!.script.length);
    // The byte at that offset should be OP_0 (00)
    expect(result.artifact!.script.slice(hexOffset, hexOffset + 2)).toBe('00');
  });

  it('Escrow artifact has constructorSlots for referenced properties', () => {
    const result = compile(ESCROW_SOURCE);
    expect(result.success).toBe(true);

    const slots = result.artifact!.constructorSlots;
    expect(slots).toBeDefined();
    // Only buyer (paramIndex 0) and seller (paramIndex 1) are referenced
    // in the method bodies — arbiter is declared but unused, so no placeholder.
    expect(slots!.length).toBeGreaterThanOrEqual(2);

    const paramIndices = new Set(slots!.map((s: { paramIndex: number }) => s.paramIndex));
    expect(paramIndices.has(0)).toBe(true); // buyer (used in release)
    expect(paramIndices.has(1)).toBe(true); // seller (used in refund)

    // All byte offsets should point to OP_0 in the script
    for (const slot of slots!) {
      const hexOffset = slot.byteOffset * 2;
      expect(result.artifact!.script.slice(hexOffset, hexOffset + 2)).toBe('00');
    }
  });

  it('HashLock artifact has constructorSlot for hashValue', () => {
    const result = compile(HASH_LOCK_SOURCE);
    expect(result.success).toBe(true);

    const slots = result.artifact!.constructorSlots;
    expect(slots).toBeDefined();
    expect(slots!.length).toBeGreaterThanOrEqual(1);
    expect(slots![0]!.paramIndex).toBe(0);
  });

  it('baked constructor args produce no constructorSlots', () => {
    const result = compile(P2PKH_SOURCE, {
      constructorArgs: { pk: '02' + 'ab'.repeat(32) },
    });
    expect(result.success).toBe(true);

    const slots = result.artifact!.constructorSlots;
    // When args are baked in, there should be no placeholder slots
    expect(slots === undefined || slots.length === 0).toBe(true);
  });

  it('constructorSlot byteOffsets are distinct', () => {
    const result = compile(ESCROW_SOURCE);
    expect(result.success).toBe(true);

    const slots = result.artifact!.constructorSlots;
    expect(slots).toBeDefined();

    const offsets = slots!.map(s => s.byteOffset);
    const uniqueOffsets = new Set(offsets);
    expect(uniqueOffsets.size).toBe(offsets.length);
  });
});
