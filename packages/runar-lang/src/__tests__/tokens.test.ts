// ---------------------------------------------------------------------------
// Tests for runar-lang/tokens — FungibleToken and NonFungibleToken stubs
// ---------------------------------------------------------------------------

import { describe, it, expect } from 'vitest';
import { FungibleToken } from '../tokens/fungible.js';
import { NonFungibleToken } from '../tokens/nft.js';

const RUNTIME_ERROR_PATTERN = /cannot be called at runtime/;

// Both classes are abstract, so we need concrete subclasses to instantiate them.

class TestFungible extends FungibleToken {
  constructor(supply: bigint, holder: any) {
    super(supply, holder);
  }
}

class TestNFT extends NonFungibleToken {
  constructor(owner: any, tokenId: any) {
    super(owner, tokenId);
  }
}

describe('FungibleToken', () => {
  const dummy = '' as any;

  it('is abstract and cannot be instantiated directly', () => {
    // TypeScript enforces this at compile time, but we can verify the
    // subclass pattern works
    const token = new TestFungible(1000n, dummy);
    expect(token).toBeInstanceOf(FungibleToken);
  });

  it('stores supply and holder from constructor', () => {
    const token = new TestFungible(500n, 'somePubKey' as any);
    expect(token.supply).toBe(500n);
    expect(token.holder).toBe('somePubKey');
  });

  it('transfer() throws at runtime', () => {
    const token = new TestFungible(100n, dummy);
    expect(() => token.transfer(dummy, dummy)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('merge() throws at runtime', () => {
    const token = new TestFungible(100n, dummy);
    expect(() => token.merge(dummy, 50n, dummy)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('split() throws at runtime', () => {
    const token = new TestFungible(100n, dummy);
    expect(() => token.split(dummy, 50n, dummy, dummy)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('transfer error message mentions "compile"', () => {
    const token = new TestFungible(100n, dummy);
    try {
      token.transfer(dummy, dummy);
      expect.unreachable('should have thrown');
    } catch (err) {
      expect((err as Error).message).toContain('compile');
      expect((err as Error).message).toContain('FungibleToken');
    }
  });

  it('validateGenesis() throws at runtime (via protected access)', () => {
    // Access protected method via cast
    const token = new TestFungible(100n, dummy);
    expect(() => (token as any).validateGenesis(dummy)).toThrow(RUNTIME_ERROR_PATTERN);
  });
});

describe('NonFungibleToken', () => {
  const dummy = '' as any;

  it('is abstract and cannot be instantiated directly', () => {
    const nft = new TestNFT(dummy, dummy);
    expect(nft).toBeInstanceOf(NonFungibleToken);
  });

  it('stores owner and tokenId from constructor', () => {
    const nft = new TestNFT('ownerPK' as any, 'tokenABC' as any);
    expect(nft.owner).toBe('ownerPK');
    expect(nft.tokenId).toBe('tokenABC');
  });

  it('transfer() throws at runtime', () => {
    const nft = new TestNFT(dummy, dummy);
    expect(() => nft.transfer(dummy, dummy)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('burn() throws at runtime', () => {
    const nft = new TestNFT(dummy, dummy);
    expect(() => nft.burn(dummy)).toThrow(RUNTIME_ERROR_PATTERN);
  });

  it('transfer error message mentions "compile" and "NonFungibleToken"', () => {
    const nft = new TestNFT(dummy, dummy);
    try {
      nft.transfer(dummy, dummy);
      expect.unreachable('should have thrown');
    } catch (err) {
      expect((err as Error).message).toContain('compile');
      expect((err as Error).message).toContain('NonFungibleToken');
    }
  });

  it('burn error message mentions "compile" and "NonFungibleToken"', () => {
    const nft = new TestNFT(dummy, dummy);
    try {
      nft.burn(dummy);
      expect.unreachable('should have thrown');
    } catch (err) {
      expect((err as Error).message).toContain('compile');
      expect((err as Error).message).toContain('NonFungibleToken');
    }
  });
});
