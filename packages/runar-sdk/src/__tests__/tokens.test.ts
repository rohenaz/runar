import { describe, it, expect, vi } from 'vitest';
import { TokenWallet } from '../tokens.js';
import { MockProvider } from '../providers/mock.js';
import { serializeState } from '../state.js';
import type { RunarArtifact, StateField } from 'runar-ir-schema';
import type { TransactionData, UTXO } from '../types.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeArtifact(
  overrides?: Partial<RunarArtifact>,
): RunarArtifact {
  return {
    version: 'runar-v0.1.0',
    compilerVersion: '0.1.0',
    contractName: 'FungibleToken',
    abi: {
      constructor: {
        params: [
          { name: 'supply', type: 'bigint' },
          { name: 'holder', type: 'PubKey' },
        ],
      },
      methods: [
        {
          name: 'transfer',
          params: [
            { name: 'sig', type: 'Sig' },
            { name: 'to', type: 'Addr' },
          ],
          isPublic: true,
        },
        {
          name: 'merge',
          params: [
            { name: 'sig', type: 'Sig' },
            { name: 'otherSupply', type: 'bigint' },
            { name: 'otherHolder', type: 'PubKey' },
          ],
          isPublic: true,
        },
      ],
    },
    script: '76a988ac',
    asm: 'OP_DUP OP_HASH160 OP_EQUALVERIFY OP_CHECKSIG',
    buildTimestamp: '2026-03-02T00:00:00.000Z',
    stateFields: [
      { name: 'supply', type: 'bigint', index: 0 },
      { name: 'holder', type: 'PubKey', index: 1 },
    ],
    ...overrides,
  };
}

const FAKE_TXID = 'aa'.repeat(32);

function makeTx(
  txid: string,
  outputs: Array<{ satoshis: number; script: string }>,
): TransactionData {
  return {
    txid,
    version: 1,
    inputs: [{ txid: '00'.repeat(32), outputIndex: 0, script: '', sequence: 0xffffffff }],
    outputs,
    locktime: 0,
  };
}

function makeTokenUtxo(
  txid: string,
  codeHex: string,
  supply: bigint,
  holder: string,
  satoshis = 10_000,
  outputIndex = 0,
): { utxo: UTXO; tx: TransactionData } {
  const stateFields: StateField[] = [
    { name: 'supply', type: 'bigint', index: 0 },
    { name: 'holder', type: 'PubKey', index: 1 },
  ];
  const stateHex = serializeState(stateFields, { supply, holder });
  const fullScript = codeHex + '6a' + stateHex; // code + OP_RETURN + state

  const tx = makeTx(txid, [{ satoshis, script: fullScript }]);
  const utxo: UTXO = { txid, outputIndex, satoshis, script: fullScript };
  return { utxo, tx };
}

const DUMMY_HOLDER = '02' + 'aa'.repeat(32);

// ---------------------------------------------------------------------------
// Mocked signer
// ---------------------------------------------------------------------------

function makeMockSigner() {
  return {
    getAddress: vi.fn().mockResolvedValue('mocked-address'),
    getPublicKey: vi.fn().mockResolvedValue(DUMMY_HOLDER),
    sign: vi.fn().mockResolvedValue('00'.repeat(72)),
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('TokenWallet', () => {
  // -------------------------------------------------------------------------
  // getBalance
  // -------------------------------------------------------------------------

  describe('getBalance', () => {
    it('returns 0n when no UTXOs exist', async () => {
      const provider = new MockProvider();
      const signer = makeMockSigner();
      const artifact = makeArtifact();
      const wallet = new TokenWallet(artifact, provider, signer);

      const balance = await wallet.getBalance();
      expect(balance).toBe(0n);
    });

    it('sums supply across multiple UTXOs', async () => {
      const provider = new MockProvider();
      const signer = makeMockSigner();
      const artifact = makeArtifact();

      // Create two token UTXOs with different supplies
      const txid1 = 'aa'.repeat(32);
      const txid2 = 'bb'.repeat(32);
      const { utxo: utxo1, tx: tx1 } = makeTokenUtxo(txid1, '76a988ac', 100n, DUMMY_HOLDER);
      const { utxo: utxo2, tx: tx2 } = makeTokenUtxo(txid2, '76a988ac', 250n, DUMMY_HOLDER);

      provider.addTransaction(tx1);
      provider.addTransaction(tx2);
      provider.addUtxo('mocked-address', utxo1);
      provider.addUtxo('mocked-address', utxo2);

      const wallet = new TokenWallet(artifact, provider, signer);
      const balance = await wallet.getBalance();
      expect(balance).toBe(350n);
    });
  });

  // -------------------------------------------------------------------------
  // getUtxos — script prefix filtering
  // -------------------------------------------------------------------------

  describe('getUtxos', () => {
    it('filters UTXOs by script prefix matching artifact.script', async () => {
      const provider = new MockProvider();
      const signer = makeMockSigner();
      const artifact = makeArtifact({ script: '76a988ac' });

      // One matching UTXO, one non-matching
      const matching: UTXO = {
        txid: FAKE_TXID,
        outputIndex: 0,
        satoshis: 10_000,
        script: '76a988ac' + '6a' + '0100', // starts with artifact.script
      };
      const nonMatching: UTXO = {
        txid: 'bb'.repeat(32),
        outputIndex: 0,
        satoshis: 5_000,
        script: 'deadbeef6a0100', // different prefix
      };

      provider.addUtxo('mocked-address', matching);
      provider.addUtxo('mocked-address', nonMatching);

      const wallet = new TokenWallet(artifact, provider, signer);
      const utxos = await wallet.getUtxos();
      expect(utxos).toHaveLength(1);
      expect(utxos[0]!.txid).toBe(FAKE_TXID);
    });

    it('includes all UTXOs when script is missing', async () => {
      const provider = new MockProvider();
      const signer = makeMockSigner();

      const utxo1: UTXO = { txid: FAKE_TXID, outputIndex: 0, satoshis: 10_000, script: '' };
      const utxo2: UTXO = { txid: 'bb'.repeat(32), outputIndex: 0, satoshis: 5_000, script: '' };
      provider.addUtxo('mocked-address', utxo1);
      provider.addUtxo('mocked-address', utxo2);

      // Artifact with empty script — can't filter
      const artifact = makeArtifact({ script: '' });
      const wallet = new TokenWallet(artifact, provider, signer);
      const utxos = await wallet.getUtxos();
      expect(utxos).toHaveLength(2);
    });
  });

  // -------------------------------------------------------------------------
  // transfer — error paths
  // -------------------------------------------------------------------------

  describe('transfer', () => {
    it('throws when no UTXOs exist', async () => {
      const provider = new MockProvider();
      const signer = makeMockSigner();
      const artifact = makeArtifact();
      const wallet = new TokenWallet(artifact, provider, signer);

      await expect(wallet.transfer('recipient-address', 100n)).rejects.toThrow(
        'no token UTXOs found',
      );
    });

    it('throws when no UTXO has sufficient balance', async () => {
      const provider = new MockProvider();
      const signer = makeMockSigner();
      const artifact = makeArtifact();

      const { utxo, tx } = makeTokenUtxo(FAKE_TXID, '76a988ac', 50n, DUMMY_HOLDER);
      provider.addTransaction(tx);
      provider.addUtxo('mocked-address', utxo);

      const wallet = new TokenWallet(artifact, provider, signer);
      await expect(wallet.transfer('recipient-address', 100n)).rejects.toThrow(
        'insufficient token balance',
      );
    });
  });

  // -------------------------------------------------------------------------
  // merge — error paths
  // -------------------------------------------------------------------------

  describe('merge', () => {
    it('throws when fewer than 2 UTXOs exist', async () => {
      const provider = new MockProvider();
      const signer = makeMockSigner();
      const artifact = makeArtifact();

      // Only one UTXO
      const { utxo, tx } = makeTokenUtxo(FAKE_TXID, '76a988ac', 100n, DUMMY_HOLDER);
      provider.addTransaction(tx);
      provider.addUtxo('mocked-address', utxo);

      const wallet = new TokenWallet(artifact, provider, signer);
      await expect(wallet.merge()).rejects.toThrow('need at least 2 UTXOs to merge');
    });

    it('throws when no UTXOs exist', async () => {
      const provider = new MockProvider();
      const signer = makeMockSigner();
      const artifact = makeArtifact();
      const wallet = new TokenWallet(artifact, provider, signer);

      await expect(wallet.merge()).rejects.toThrow('need at least 2 UTXOs to merge');
    });
  });
});
