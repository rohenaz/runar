/**
 * PostQuantumWallet integration test — stateless contract with WOTS+ verification.
 *
 * ## How It Works
 *
 * PostQuantumWallet locks funds to a Winternitz One-Time Signature (WOTS+) public key.
 * WOTS+ is a hash-based post-quantum signature scheme — its security relies only on
 * the collision resistance of SHA-256, not on any number-theoretic assumption.
 *
 * ### Constructor
 *   - pubkey: ByteString — 64-byte hex (pubSeed[32] || pkRoot[32])
 *     - pubSeed: randomness used in hash chain key derivation
 *     - pkRoot: SHA-256 hash of all 67 public key chain endpoints
 *
 * ### Method: spend(msg: ByteString, sig: ByteString)
 *   - msg: the message to verify (arbitrary bytes; hashed internally to 32 bytes)
 *   - sig: 2,144-byte WOTS+ signature (67 chains × 32 bytes each)
 *
 * ### How WOTS+ Works
 *   1. Key generation: create 67 random 32-byte secret keys (sk[0..66])
 *   2. For each sk[i], compute a hash chain of length 15 (W=16):
 *      pk[i] = H^15(sk[i]) (apply SHA-256 fifteen times)
 *   3. Public key = SHA-256(pk[0] || pk[1] || ... || pk[66])
 *   4. To sign: hash the message to get 64 base-16 digits + 3 checksum digits.
 *      For digit d[i], output sig[i] = H^d[i](sk[i]) — i.e., chain sk[i] forward d steps
 *   5. Verifier chains each sig[i] the remaining (15 - d[i]) steps and checks
 *      the result matches the public key
 *
 * ### Script Size
 *   ~10 KB — modest because WOTS+ verification is just iterative SHA-256 hashing.
 *
 * ### Important Notes
 *   - "One-time" means each UTXO can only be spent once with a given keypair.
 *     Reusing the same keypair for a different message leaks secret key material.
 *   - No Sig param — this is a hash-based signature, not ECDSA
 */

import { describe, it, expect } from 'vitest';
import { compileContract } from './helpers/compile.js';
import { RunarContract, RPCProvider } from 'runar-sdk';
import { createFundedWallet } from './helpers/wallet.js';
import { wotsKeygen, wotsSign, wotsPubKeyHex } from './helpers/crypto.js';

function createProvider() {
  return new RPCProvider('http://localhost:18332', 'regtest', 'regtest', {
    autoMine: true,
    network: 'testnet',
  });
}

describe('PostQuantumWallet', () => {
  it('should compile the contract', () => {
    const artifact = compileContract('examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts');
    expect(artifact).toBeTruthy();
    expect(artifact.contractName).toBe('PostQuantumWallet');
    expect(artifact.script.length).toBeGreaterThan(0);
  });

  it('should produce a script of approximately 10 KB', () => {
    const artifact = compileContract('examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts');
    const scriptBytes = artifact.script.length / 2;
    // WOTS+ scripts are typically ~10 KB
    expect(scriptBytes).toBeGreaterThan(5000);
    expect(scriptBytes).toBeLessThan(50000);
  });

  it('should deploy with a WOTS+ public key', async () => {
    const artifact = compileContract('examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    // Generate WOTS+ keypair from a deterministic seed
    const seed = Buffer.alloc(32);
    seed[0] = 0x42;
    const pubSeed = Buffer.alloc(32);
    pubSeed[0] = 0x01;
    const kp = wotsKeygen(seed, pubSeed);

    // Constructor: (pubkey: ByteString) — 64-byte hex (pubSeed || pkRoot)
    const contract = new RunarContract(artifact, [wotsPubKeyHex(kp)]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 10000 });
    expect(deployTxid).toBeTruthy();
    expect(typeof deployTxid).toBe('string');
    expect(deployTxid.length).toBe(64);
  });

  it('should deploy with a different seed', async () => {
    const artifact = compileContract('examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    // Different seed produces a different public key
    const seed = Buffer.alloc(32);
    seed[0] = 0x99;
    seed[1] = 0xAB;
    const pubSeed = Buffer.alloc(32);
    pubSeed[0] = 0x02;
    const kp = wotsKeygen(seed, pubSeed);

    const contract = new RunarContract(artifact, [wotsPubKeyHex(kp)]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 10000 });
    expect(deployTxid).toBeTruthy();
  });

  it('should deploy and spend with a valid WOTS+ signature', async () => {
    const artifact = compileContract('examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    // --- Step 1: Generate WOTS+ keypair ---
    const seed = Buffer.alloc(32);
    seed[0] = 0x42;
    const pubSeed = Buffer.alloc(32);
    pubSeed[0] = 0x01;
    const kp = wotsKeygen(seed, pubSeed);

    // --- Step 2: Deploy the contract ---
    const contract = new RunarContract(artifact, [wotsPubKeyHex(kp)]);
    await contract.deploy(provider, signer, { satoshis: 10000 });

    // --- Step 3: Sign a message with the WOTS+ secret key ---
    // The message is arbitrary — it gets SHA-256 hashed internally.
    // Each nibble of the hash determines how far to chain each secret key.
    const msg = Buffer.from('spend this UTXO');
    const sig = wotsSign(msg, kp.sk, kp.pubSeed);

    // WOTS+ signature is 67 chains × 32 bytes = 2,144 bytes
    expect(sig.length).toBe(2144);

    // --- Step 4: Call spend(msg, sig) to unlock the UTXO ---
    // The on-chain script:
    //   1. Hashes the message to 32 bytes (SHA-256)
    //   2. Extracts 64 base-16 digits + 3 checksum digits
    //   3. For each digit d[i], chains sig[i] forward (15 - d[i]) times
    //   4. Hashes all 67 chain endpoints together
    //   5. Asserts the result matches pkRoot from the constructor
    const { txid: spendTxid } = await contract.call(
      'spend',
      [msg.toString('hex'), sig.toString('hex')],
      provider,
      signer,
    );
    expect(spendTxid).toBeTruthy();
    expect(spendTxid.length).toBe(64);
  });

  it('should reject spend with tampered signature', async () => {
    const artifact = compileContract('examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    // Generate WOTS+ keypair
    const seed = Buffer.alloc(32);
    seed[0] = 0x42;
    const pubSeed = Buffer.alloc(32);
    pubSeed[0] = 0x01;
    const kp = wotsKeygen(seed, pubSeed);

    const contract = new RunarContract(artifact, [wotsPubKeyHex(kp)]);
    await contract.deploy(provider, signer, { satoshis: 10000 });

    // Sign and then tamper with the signature
    const msg = Buffer.from('spend this UTXO');
    const sig = wotsSign(msg, kp.sk, kp.pubSeed);

    // Tamper: XOR byte 100 with 0xFF
    const tamperedSig = Buffer.from(sig);
    tamperedSig[100] ^= 0xFF;

    await expect(
      contract.call(
        'spend',
        [msg.toString('hex'), tamperedSig.toString('hex')],
        provider,
        signer,
      ),
    ).rejects.toThrow();
  });

  it('should reject spend with wrong message', async () => {
    const artifact = compileContract('examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts');

    const provider = createProvider();
    const { signer } = await createFundedWallet(provider);

    // Generate WOTS+ keypair
    const seed = Buffer.alloc(32);
    seed[0] = 0x42;
    const pubSeed = Buffer.alloc(32);
    pubSeed[0] = 0x01;
    const kp = wotsKeygen(seed, pubSeed);

    const contract = new RunarContract(artifact, [wotsPubKeyHex(kp)]);
    await contract.deploy(provider, signer, { satoshis: 10000 });

    // Sign "original message" but call with "different message"
    const originalMsg = Buffer.from('original message');
    const sig = wotsSign(originalMsg, kp.sk, kp.pubSeed);

    const differentMsg = Buffer.from('different message');

    await expect(
      contract.call(
        'spend',
        [differentMsg.toString('hex'), sig.toString('hex')],
        provider,
        signer,
      ),
    ).rejects.toThrow();
  });
});
