/**
 * SPHINCSWallet integration test — stateless contract with SLH-DSA-SHA2-128s verification.
 *
 * ## How It Works
 *
 * SPHINCSWallet locks funds to an SLH-DSA public key (FIPS 205, 128-bit post-quantum
 * security level). Unlike WOTS+ (which is one-time), the same SLH-DSA keypair can
 * sign many messages because it uses a Merkle tree of WOTS+ keys internally.
 *
 * ### Constructor
 *   - pubkey: ByteString — 32-byte hex (PK.seed[16] || PK.root[16])
 *
 * ### Method: spend(msg: ByteString, sig: ByteString)
 *   - msg: the signed message (arbitrary bytes)
 *   - sig: 7,856-byte SLH-DSA-SHA2-128s signature
 *   The contract verifies the SLH-DSA signature on-chain using ~188 KB of Bitcoin Script.
 *
 * ### Script Size
 *   ~188 KB — SLH-DSA verification requires computing multiple WOTS+ verifications
 *   and Merkle tree path checks within the Bitcoin Script VM.
 *
 * ### Test Approach
 *   We use a pre-computed test vector from conformance/testdata/slhdsa-test-sig.hex
 *   with a known public key and message, avoiding the need for a full SLH-DSA
 *   signing library. The same test vector is used by the Go integration tests.
 */

import { describe, it, expect } from 'vitest';
import { compileContract } from './helpers/compile.js';
import { RunarContract, RPCProvider } from 'runar-sdk';
import { createFundedWallet } from './helpers/wallet.js';
import { readFileSync } from 'fs';
import { resolve } from 'path';

function createProvider() {
  return new RPCProvider('http://localhost:18332', 'regtest', 'regtest', {
    autoMine: true,
    network: 'testnet',
  });
}

// Deterministic test public key (32 bytes hex: PK.seed || PK.root)
const SLHDSA_TEST_PK = '00000000000000000000000000000000b618cb38f7f785488c9768f3a2972baf';
// Message that was signed: "slh-dsa test vector" in hex
const SLHDSA_TEST_MSG = '736c682d647361207465737420766563746f72';

/**
 * Load the pre-computed SLH-DSA test signature from the conformance test data.
 * The signature is 7,856 bytes (15,712 hex chars) generated offline with
 * a known keypair for deterministic testing.
 */
function loadTestSignature(): string {
  const sigPath = resolve(__dirname, '../../conformance/testdata/slhdsa-test-sig.hex');
  return readFileSync(sigPath, 'utf-8').trim();
}

describe('SPHINCSWallet', () => {
  it('should compile the contract', () => {
    const artifact = compileContract('examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts');
    expect(artifact).toBeTruthy();
    expect(artifact.contractName).toBe('SPHINCSWallet');
    expect(artifact.script.length).toBeGreaterThan(0);
  });

  it('should produce a very large script (~188 KB)', () => {
    const artifact = compileContract('examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts');
    const scriptBytes = artifact.script.length / 2;
    // SLH-DSA scripts are typically ~188 KB
    expect(scriptBytes).toBeGreaterThan(100000);
    expect(scriptBytes).toBeLessThan(500000);
  });

  it('should deploy with an SLH-DSA public key', async () => {
    const artifact = compileContract('examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    // Constructor: (pubkey: ByteString) — 32-byte hex
    const contract = new RunarContract(artifact, [SLHDSA_TEST_PK]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 50000 });
    expect(deployTxid).toBeTruthy();
    expect(typeof deployTxid).toBe('string');
    expect(deployTxid.length).toBe(64);
  });

  it('should deploy with a different public key', async () => {
    const artifact = compileContract('examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    // Different test pubkey
    const otherPK = 'aabbccdd00000000000000000000000011223344556677889900aabbccddeeff';
    const contract = new RunarContract(artifact, [otherPK]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 50000 });
    expect(deployTxid).toBeTruthy();
  });

  it('should deploy and spend with a valid SLH-DSA signature', async () => {
    const artifact = compileContract('examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    // --- Step 1: Deploy with the test public key ---
    const contract = new RunarContract(artifact, [SLHDSA_TEST_PK]);
    await contract.deploy(provider, signer, { satoshis: 50000 });

    // --- Step 2: Load the pre-computed test signature ---
    // The signature was generated offline with the matching private key.
    // SLH-DSA-SHA2-128s signatures are 7,856 bytes (FIPS 205 Table 2).
    const sigHex = loadTestSignature();
    expect(sigHex.length / 2).toBe(7856);

    // --- Step 3: Call spend(msg, sig) to unlock the UTXO ---
    // The contract's on-chain script verifies the SLH-DSA signature against
    // the public key embedded in the locking script. This involves:
    //   1. Parsing the signature into FORS trees + Hypertree layers
    //   2. Computing WOTS+ public keys from signature chains
    //   3. Verifying Merkle tree authentication paths
    //   4. Comparing the reconstructed root against PK.root
    const { txid: spendTxid } = await contract.call(
      'spend',
      [SLHDSA_TEST_MSG, sigHex],
      provider,
      signer,
    );
    expect(spendTxid).toBeTruthy();
    expect(spendTxid.length).toBe(64);
  });

  it('should reject spend with tampered signature', async () => {
    const artifact = compileContract('examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    const contract = new RunarContract(artifact, [SLHDSA_TEST_PK]);
    await contract.deploy(provider, signer, { satoshis: 50000 });

    // Load the valid signature and tamper byte 500 with XOR 0xFF
    const sigHex = loadTestSignature();
    const sigBuf = Buffer.from(sigHex, 'hex');
    sigBuf[500] ^= 0xFF;
    const tamperedSigHex = sigBuf.toString('hex');

    await expect(
      contract.call(
        'spend',
        [SLHDSA_TEST_MSG, tamperedSigHex],
        provider,
        signer,
      ),
    ).rejects.toThrow();
  });
});
