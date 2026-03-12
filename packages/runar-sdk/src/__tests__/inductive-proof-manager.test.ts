/**
 * Tests for InductiveProofManager and the end-to-end inductive contract flow.
 */
import { describe, it, expect } from 'vitest';
import { InductiveProofManager, PROOF_SIZE, ZERO_PROOF } from '../inductive-proof.js';

describe('InductiveProofManager', () => {
  it('initializes with zero proof by default', () => {
    const pm = new InductiveProofManager();
    expect(pm.proof).toBe(ZERO_PROOF);
    expect(pm.proof.length).toBe(PROOF_SIZE * 2);
  });

  it('initializes with a custom proof', () => {
    const customProof = 'ab'.repeat(PROOF_SIZE);
    const pm = new InductiveProofManager(customProof);
    expect(pm.proof).toBe(customProof);
  });

  it('rejects proofs of wrong size', () => {
    const pm = new InductiveProofManager();
    expect(() => { pm.proof = 'ab'.repeat(10); }).toThrow('192 bytes');
  });

  it('generates zero proof without a generator', async () => {
    const pm = new InductiveProofManager();
    const proof = await pm.generateProof('aa'.repeat(36), 'bb'.repeat(32), {});
    expect(proof).toBe(ZERO_PROOF);
    expect(pm.hasGenerator).toBe(false);
  });

  it('uses custom generator when provided', async () => {
    const customProof = 'cc'.repeat(PROOF_SIZE);
    const generator = async () => customProof;
    const pm = new InductiveProofManager(undefined, generator);

    expect(pm.hasGenerator).toBe(true);
    const proof = await pm.generateProof('aa'.repeat(36), 'bb'.repeat(32), {});
    expect(proof).toBe(customProof);
    expect(pm.proof).toBe(customProof);
  });

  it('generator receives correct arguments', async () => {
    const genesis = 'dd'.repeat(36);
    const txid = 'ee'.repeat(32);
    const state = { count: 42n };

    let receivedArgs: unknown[] = [];
    const generator = async (g: string, prev: string, t: string, s: Record<string, unknown>) => {
      receivedArgs = [g, prev, t, s];
      return 'ff'.repeat(PROOF_SIZE);
    };

    const pm = new InductiveProofManager(undefined, generator);
    await pm.generateProof(genesis, txid, state);

    expect(receivedArgs[0]).toBe(genesis);
    expect(receivedArgs[1]).toBe(ZERO_PROOF); // previous proof was zero
    expect(receivedArgs[2]).toBe(txid);
    expect(receivedArgs[3]).toEqual(state);
  });
});
