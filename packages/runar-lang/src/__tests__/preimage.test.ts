// ---------------------------------------------------------------------------
// Tests for runar-lang/preimage.ts — compiler stub functions
// ---------------------------------------------------------------------------

import { describe, it, expect } from 'vitest';
import {
  checkPreimage,
  extractVersion,
  extractHashPrevouts,
  extractHashSequence,
  extractOutpoint,
  extractInputIndex,
  extractScriptCode,
  extractAmount,
  extractSequence,
  extractOutputHash,
  extractOutputs,
  extractLocktime,
  extractSigHashType,
} from '../preimage.js';

const RUNTIME_ERROR_PATTERN = /cannot be called at runtime/;

describe('preimage compiler stubs', () => {
  const dummyPreimage = '' as any;

  it('checkPreimage throws at runtime', () => {
    expect(() => checkPreimage(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractVersion throws at runtime', () => {
    expect(() => extractVersion(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractHashPrevouts throws at runtime', () => {
    expect(() => extractHashPrevouts(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractHashSequence throws at runtime', () => {
    expect(() => extractHashSequence(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractOutpoint throws at runtime', () => {
    expect(() => extractOutpoint(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractInputIndex throws at runtime', () => {
    expect(() => extractInputIndex(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractScriptCode throws at runtime', () => {
    expect(() => extractScriptCode(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractAmount throws at runtime', () => {
    expect(() => extractAmount(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractSequence throws at runtime', () => {
    expect(() => extractSequence(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractOutputHash throws at runtime', () => {
    expect(() => extractOutputHash(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractOutputs throws at runtime', () => {
    expect(() => extractOutputs(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractLocktime throws at runtime', () => {
    expect(() => extractLocktime(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('extractSigHashType throws at runtime', () => {
    expect(() => extractSigHashType(dummyPreimage)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('each function mentions "compile" in the error message', () => {
    const fns = [
      checkPreimage,
      extractVersion,
      extractHashPrevouts,
      extractHashSequence,
      extractOutpoint,
      extractInputIndex,
      extractScriptCode,
      extractAmount,
      extractSequence,
      extractOutputHash,
      extractOutputs,
      extractLocktime,
      extractSigHashType,
    ];

    for (const fn of fns) {
      try {
        (fn as any)(dummyPreimage);
        expect.unreachable(`${fn.name} should have thrown`);
      } catch (err) {
        expect((err as Error).message).toContain('compile');
      }
    }
  });
});

describe('verifyRabinSig compiler stub', () => {
  it('throws at runtime with compile message', async () => {
    const { verifyRabinSig } = await import('../oracle/rabin.js');
    const dummy = '' as any;
    expect(() => verifyRabinSig(dummy, dummy, dummy, dummy)).toThrow(
      /cannot be called at runtime/,
    );
    try {
      verifyRabinSig(dummy, dummy, dummy, dummy);
    } catch (err) {
      expect((err as Error).message).toContain('compile');
      expect((err as Error).message).toContain('verifyRabinSig');
    }
  });
});
