/**
 * FungibleToken integration test — stateful contract with addOutput (SDK Deploy path).
 *
 * FungibleToken is a StatefulSmartContract with properties:
 *   - owner: PubKey (mutable)
 *   - balance: bigint (mutable)
 *   - tokenId: ByteString (readonly)
 *
 * Methods: transfer(sig, to, amount, outputSatoshis), send(sig, to, outputSatoshis),
 *          merge(sig, totalBalance, outputSatoshis)
 *
 * All methods require a Sig parameter (checkSig), so spending requires raw transaction
 * construction. We test compile + deploy via the SDK. Full spending tests are covered
 * by the Go integration suite (token_ft_test.go).
 */

import { describe, it, expect } from 'vitest';
import { compileContract } from './helpers/compile.js';
import { RunarContract, RPCProvider } from 'runar-sdk';
import { createFundedWallet, createWallet } from './helpers/wallet.js';

function createProvider() {
  return new RPCProvider('http://localhost:18332', 'regtest', 'regtest', {
    autoMine: true,
    network: 'testnet',
  });
}

describe('FungibleToken', () => {
  it('should compile the FungibleToken contract', () => {
    const artifact = compileContract('examples/ts/token-ft/FungibleTokenExample.runar.ts');
    expect(artifact).toBeTruthy();
    expect(artifact.contractName).toBe('FungibleToken');
  });

  it('should deploy with owner and initial balance', async () => {
    const artifact = compileContract('examples/ts/token-ft/FungibleTokenExample.runar.ts');

    const provider = createProvider();
    const owner = createWallet();
    const { signer } = await createFundedWallet(provider);

    // tokenId is a ByteString (hex-encoded)
    const tokenIdHex = Buffer.from('TEST-TOKEN-001').toString('hex');

    // Constructor: (owner: PubKey, balance: bigint, tokenId: ByteString)
    const contract = new RunarContract(artifact, [
      owner.pubKeyHex,
      1000n,
      tokenIdHex,
    ]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 5000 });
    expect(deployTxid).toBeTruthy();
    expect(typeof deployTxid).toBe('string');
    expect(deployTxid.length).toBe(64);
  });

  it('should deploy with zero initial balance', async () => {
    const artifact = compileContract('examples/ts/token-ft/FungibleTokenExample.runar.ts');

    const provider = createProvider();
    const owner = createWallet();
    const { signer } = await createFundedWallet(provider);

    const tokenIdHex = Buffer.from('ZERO-BAL-TOKEN').toString('hex');

    const contract = new RunarContract(artifact, [
      owner.pubKeyHex,
      0n,
      tokenIdHex,
    ]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 5000 });
    expect(deployTxid).toBeTruthy();
  });

  it('should deploy with large balance', async () => {
    const artifact = compileContract('examples/ts/token-ft/FungibleTokenExample.runar.ts');

    const provider = createProvider();
    const owner = createWallet();
    const { signer } = await createFundedWallet(provider);

    const tokenIdHex = Buffer.from('BIG-TOKEN').toString('hex');

    const contract = new RunarContract(artifact, [
      owner.pubKeyHex,
      21000000_00000000n, // 21 million * 10^8 (satoshi-scale)
      tokenIdHex,
    ]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 5000 });
    expect(deployTxid).toBeTruthy();
  });

  it('should send entire balance to a recipient', async () => {
    const artifact = compileContract('examples/ts/token-ft/FungibleTokenExample.runar.ts');

    const provider = createProvider();
    const { signer, pubKeyHex } = await createFundedWallet(provider);
    const recipient = createWallet();

    const tokenIdHex = Buffer.from('SEND-TOKEN').toString('hex');

    // Constructor: (owner: PubKey, balance: bigint, tokenId: ByteString)
    const contract = new RunarContract(artifact, [
      pubKeyHex,
      1000n,
      tokenIdHex,
    ]);

    await contract.deploy(provider, signer, { satoshis: 5000 });

    // send(sig, to, outputSatoshis) — null Sig is auto-computed from the signer
    // send uses addOutput: the on-chain script expects output with owner=to, balance=1000
    // Pass newState so the SDK builds the correct continuation output
    const { txid: callTxid } = await contract.call(
      'send', [null, recipient.pubKeyHex, 5000n], provider, signer,
      { newState: { owner: recipient.pubKeyHex } },
    );
    expect(callTxid).toBeTruthy();
    expect(callTxid.length).toBe(64);
  });

  it('should reject send with wrong signer', async () => {
    const artifact = compileContract('examples/ts/token-ft/FungibleTokenExample.runar.ts');

    const provider = createProvider();
    // Deploy with owner=walletA
    const { signer: ownerSigner, pubKeyHex: ownerPubKey } = await createFundedWallet(provider);
    const recipient = createWallet();

    const tokenIdHex = Buffer.from('REJECT-SEND-TOKEN').toString('hex');

    const contract = new RunarContract(artifact, [
      ownerPubKey,
      1000n,
      tokenIdHex,
    ]);

    await contract.deploy(provider, ownerSigner, { satoshis: 5000 });

    // Call send with walletB — checkSig will fail on-chain
    const { signer: wrongSigner } = await createFundedWallet(provider);

    await expect(
      contract.call(
        'send', [null, recipient.pubKeyHex, 5000n], provider, wrongSigner,
        { newState: { owner: recipient.pubKeyHex } },
      ),
    ).rejects.toThrow();
  });
});
