/**
 * TestSmartContract tests — verify the TestSmartContract wrapper correctly
 * loads compiled artifacts and executes them through the Script VM.
 *
 * These tests cover:
 *   1. Simple arithmetic (addition with assert)
 *   2. Hash lock (sha256 preimage verification)
 *   3. Multi-method dispatch (selector logic)
 *   4. Error cases (unknown method, wrong arg count)
 */

import { describe, it, expect } from 'vitest';
import { createHash } from 'crypto';
import { compile } from 'runar-compiler';
import {
  TestSmartContract,
  expectScriptSuccess,
  expectScriptFailure,
} from '../helpers.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function sha256hex(hexData: string): string {
  return createHash('sha256')
    .update(Buffer.from(hexData, 'hex'))
    .digest('hex');
}

// ---------------------------------------------------------------------------
// Contract sources
// ---------------------------------------------------------------------------

const ADDITION_SOURCE = `
class Addition extends SmartContract {
  readonly target: bigint;
  constructor(target: bigint) {
    super(target);
    this.target = target;
  }
  public verify(a: bigint, b: bigint) {
    assert(a + b === this.target);
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

const MULTI_METHOD_SOURCE = `
class TwoMethods extends SmartContract {
  readonly x: bigint;
  readonly y: bigint;
  constructor(x: bigint, y: bigint) {
    super(x, y);
    this.x = x;
    this.y = y;
  }
  public checkX(val: bigint) {
    assert(val === this.x);
  }
  public checkY(val: bigint) {
    assert(val === this.y);
  }
}
`;

// ---------------------------------------------------------------------------
// 1. Simple arithmetic contract
// ---------------------------------------------------------------------------

describe('TestSmartContract: arithmetic', () => {
  it('compiles the Addition contract successfully', () => {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    expect(result.success).toBe(true);
    expect(result.artifact).toBeDefined();
  });

  it('verify(7, 8) succeeds when target is 15', () => {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const vmResult = contract.call('verify', [7n, 8n]);
    expectScriptSuccess(vmResult);
  });

  it('verify(5, 5) fails when target is 15', () => {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const vmResult = contract.call('verify', [5n, 5n]);
    expectScriptFailure(vmResult);
  });

  it('verify(0, 15) succeeds when target is 15', () => {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const vmResult = contract.call('verify', [0n, 15n]);
    expectScriptSuccess(vmResult);
  });

  it('verify(100, -85) succeeds when target is 15', () => {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const vmResult = contract.call('verify', [100n, -85n]);
    expectScriptSuccess(vmResult);
  });
});

// ---------------------------------------------------------------------------
// 2. Hash lock contract
// ---------------------------------------------------------------------------

describe('TestSmartContract: hash lock', () => {
  // "hello" in hex is 68656c6c6f
  const preimageHex = '68656c6c6f';
  const hashHex = sha256hex(preimageHex);

  it('compiles the HashLock contract successfully', () => {
    const result = compile(HASH_LOCK_SOURCE, {
      constructorArgs: { hashValue: hashHex },
    });
    expect(result.success).toBe(true);
    expect(result.artifact).toBeDefined();
  });

  it('unlock with correct preimage succeeds', () => {
    const result = compile(HASH_LOCK_SOURCE, {
      constructorArgs: { hashValue: hashHex },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const vmResult = contract.call('unlock', [preimageHex]);
    expectScriptSuccess(vmResult);
  });

  it('unlock with wrong preimage fails', () => {
    const result = compile(HASH_LOCK_SOURCE, {
      constructorArgs: { hashValue: hashHex },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    // "wrong" in hex is 77726f6e67
    const vmResult = contract.call('unlock', ['77726f6e67']);
    expectScriptFailure(vmResult);
  });

  it('unlock with empty preimage fails', () => {
    const result = compile(HASH_LOCK_SOURCE, {
      constructorArgs: { hashValue: hashHex },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const vmResult = contract.call('unlock', ['']);
    expectScriptFailure(vmResult);
  });
});

// ---------------------------------------------------------------------------
// 3. Multi-method dispatch
// ---------------------------------------------------------------------------

describe('TestSmartContract: multi-method dispatch', () => {
  function makeContract() {
    const result = compile(MULTI_METHOD_SOURCE, {
      constructorArgs: { x: 42n, y: 99n },
    });
    expect(result.success).toBe(true);
    return TestSmartContract.fromArtifact(result.artifact!, []);
  }

  it('checkX(42) succeeds', () => {
    const contract = makeContract();
    const vmResult = contract.call('checkX', [42n]);
    expectScriptSuccess(vmResult);
  });

  it('checkX(99) fails (99 !== 42)', () => {
    const contract = makeContract();
    const vmResult = contract.call('checkX', [99n]);
    expectScriptFailure(vmResult);
  });

  it('checkY(99) succeeds', () => {
    const contract = makeContract();
    const vmResult = contract.call('checkY', [99n]);
    expectScriptSuccess(vmResult);
  });

  it('checkY(42) fails (42 !== 99)', () => {
    const contract = makeContract();
    const vmResult = contract.call('checkY', [42n]);
    expectScriptFailure(vmResult);
  });

  it('checkX(0) fails', () => {
    const contract = makeContract();
    const vmResult = contract.call('checkX', [0n]);
    expectScriptFailure(vmResult);
  });

  it('checkY(0) fails', () => {
    const contract = makeContract();
    const vmResult = contract.call('checkY', [0n]);
    expectScriptFailure(vmResult);
  });
});

// ---------------------------------------------------------------------------
// 4. Error cases
// ---------------------------------------------------------------------------

describe('TestSmartContract: error cases', () => {
  function makeContract() {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    expect(result.success).toBe(true);
    return TestSmartContract.fromArtifact(result.artifact!, []);
  }

  it('throws when calling a method that does not exist', () => {
    const contract = makeContract();
    expect(() => contract.call('nonExistent', [1n])).toThrow(
      /Method 'nonExistent' not found/,
    );
  });

  it('throws when calling with too few arguments', () => {
    const contract = makeContract();
    expect(() => contract.call('verify', [7n])).toThrow(
      /expects 2 args, got 1/,
    );
  });

  it('throws when calling with too many arguments', () => {
    const contract = makeContract();
    expect(() => contract.call('verify', [7n, 8n, 1n])).toThrow(
      /expects 2 args, got 3/,
    );
  });

  it('throws when calling with no arguments', () => {
    const contract = makeContract();
    expect(() => contract.call('verify', [])).toThrow(
      /expects 2 args, got 0/,
    );
  });
});

// ---------------------------------------------------------------------------
// 5. Metadata accessors
// ---------------------------------------------------------------------------

describe('TestSmartContract: metadata', () => {
  it('getContractName returns the contract name', () => {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    expect(contract.getContractName()).toBe('Addition');
  });

  it('getABI returns method descriptors', () => {
    const result = compile(MULTI_METHOD_SOURCE, {
      constructorArgs: { x: 42n, y: 99n },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const abi = contract.getABI();
    const publicMethods = abi.methods.filter((m) => m.isPublic);
    expect(publicMethods).toHaveLength(2);
    expect(publicMethods.map((m) => m.name).sort()).toEqual(['checkX', 'checkY']);
  });

  it('getLockingScriptHex returns a non-empty hex string', () => {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const hex = contract.getLockingScriptHex();
    expect(hex.length).toBeGreaterThan(0);
    expect(/^[0-9a-f]+$/i.test(hex)).toBe(true);
  });

  it('getLockingScript returns a Uint8Array', () => {
    const result = compile(ADDITION_SOURCE, {
      constructorArgs: { target: 15n },
    });
    const contract = TestSmartContract.fromArtifact(result.artifact!, []);
    const script = contract.getLockingScript();
    expect(script).toBeInstanceOf(Uint8Array);
    expect(script.length).toBeGreaterThan(0);
  });
});
