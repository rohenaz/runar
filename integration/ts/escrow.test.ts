/**
 * Escrow integration test — stateless contract with checkSig.
 *
 * Escrow is a stateless contract that locks funds and allows release or refund
 * via four methods, each requiring a signature from the appropriate party:
 *   - releaseBySeller(sig) — seller signs to release funds to buyer
 *   - releaseByArbiter(sig) — arbiter signs to release funds to buyer
 *   - refundToBuyer(sig) — buyer signs to reclaim funds
 *   - refundByArbiter(sig) — arbiter signs to refund to buyer
 *
 * The SDK's contract.call() supports auto-computed Sig params (pass null)
 * for both stateless and stateful contracts.
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

describe('Escrow', () => {
  it('should compile the Escrow contract', () => {
    const artifact = compileContract('examples/ts/escrow/Escrow.runar.ts');
    expect(artifact).toBeTruthy();
    expect(artifact.contractName).toBe('Escrow');
  });

  it('should deploy with three distinct pubkeys', async () => {
    const artifact = compileContract('examples/ts/escrow/Escrow.runar.ts');

    const provider = createProvider();
    const buyer = createWallet();
    const seller = createWallet();
    const arbiter = createWallet();

    // Fund a separate wallet to pay for the deploy transaction
    const { signer } = await createFundedWallet(provider);

    // Constructor takes (buyer: PubKey, seller: PubKey, arbiter: PubKey)
    const contract = new RunarContract(artifact, [
      buyer.pubKeyHex,
      seller.pubKeyHex,
      arbiter.pubKeyHex,
    ]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 5000 });
    expect(deployTxid).toBeTruthy();
    expect(typeof deployTxid).toBe('string');
    expect(deployTxid.length).toBe(64);
  });

  it('should deploy with the same key for multiple roles', async () => {
    const artifact = compileContract('examples/ts/escrow/Escrow.runar.ts');

    const provider = createProvider();
    const buyerAndArbiter = createWallet();
    const seller = createWallet();

    const { signer } = await createFundedWallet(provider);

    // Same key as both buyer and arbiter
    const contract = new RunarContract(artifact, [
      buyerAndArbiter.pubKeyHex,
      seller.pubKeyHex,
      buyerAndArbiter.pubKeyHex,
    ]);

    const { txid: deployTxid } = await contract.deploy(provider, signer, { satoshis: 5000 });
    expect(deployTxid).toBeTruthy();
  });

  it('should deploy and spend with releaseBySeller(sig)', async () => {
    const artifact = compileContract('examples/ts/escrow/Escrow.runar.ts');

    const provider = createProvider();
    const buyer = createWallet();
    const arbiter = createWallet();

    // The seller must be funded because releaseBySeller checks the seller's signature
    const { signer: sellerSigner, pubKeyHex: sellerPubKey } = await createFundedWallet(provider);

    // Constructor takes (buyer: PubKey, seller: PubKey, arbiter: PubKey)
    const contract = new RunarContract(artifact, [
      buyer.pubKeyHex,
      sellerPubKey,
      arbiter.pubKeyHex,
    ]);

    await contract.deploy(provider, sellerSigner, { satoshis: 5000 });

    // null Sig arg is auto-computed by the SDK from the signer
    const { txid: callTxid } = await contract.call(
      'releaseBySeller', [null], provider, sellerSigner,
    );
    expect(callTxid).toBeTruthy();
    expect(callTxid.length).toBe(64);
  });

  it('should deploy and spend with releaseByArbiter(sig)', async () => {
    const artifact = compileContract('examples/ts/escrow/Escrow.runar.ts');

    const provider = createProvider();
    const buyer = createWallet();
    const seller = createWallet();

    // The arbiter must be funded because releaseByArbiter checks the arbiter's signature
    const { signer: arbiterSigner, pubKeyHex: arbiterPubKey } = await createFundedWallet(provider);

    const contract = new RunarContract(artifact, [
      buyer.pubKeyHex,
      seller.pubKeyHex,
      arbiterPubKey,
    ]);

    await contract.deploy(provider, arbiterSigner, { satoshis: 5000 });

    const { txid: callTxid } = await contract.call(
      'releaseByArbiter', [null], provider, arbiterSigner,
    );
    expect(callTxid).toBeTruthy();
    expect(callTxid.length).toBe(64);
  });

  it('should deploy and spend with refundToBuyer(sig)', async () => {
    const artifact = compileContract('examples/ts/escrow/Escrow.runar.ts');

    const provider = createProvider();
    const seller = createWallet();
    const arbiter = createWallet();

    // The buyer must be funded because refundToBuyer checks the buyer's signature
    const { signer: buyerSigner, pubKeyHex: buyerPubKey } = await createFundedWallet(provider);

    const contract = new RunarContract(artifact, [
      buyerPubKey,
      seller.pubKeyHex,
      arbiter.pubKeyHex,
    ]);

    await contract.deploy(provider, buyerSigner, { satoshis: 5000 });

    const { txid: callTxid } = await contract.call(
      'refundToBuyer', [null], provider, buyerSigner,
    );
    expect(callTxid).toBeTruthy();
    expect(callTxid.length).toBe(64);
  });

  it('should deploy and spend with refundByArbiter(sig)', async () => {
    const artifact = compileContract('examples/ts/escrow/Escrow.runar.ts');

    const provider = createProvider();
    const buyer = createWallet();
    const seller = createWallet();

    // The arbiter must be funded because refundByArbiter checks the arbiter's signature
    const { signer: arbiterSigner, pubKeyHex: arbiterPubKey } = await createFundedWallet(provider);

    const contract = new RunarContract(artifact, [
      buyer.pubKeyHex,
      seller.pubKeyHex,
      arbiterPubKey,
    ]);

    await contract.deploy(provider, arbiterSigner, { satoshis: 5000 });

    const { txid: callTxid } = await contract.call(
      'refundByArbiter', [null], provider, arbiterSigner,
    );
    expect(callTxid).toBeTruthy();
    expect(callTxid.length).toBe(64);
  });

  it('should reject releaseBySeller with wrong signer', async () => {
    const artifact = compileContract('examples/ts/escrow/Escrow.runar.ts');

    const provider = createProvider();
    const buyer = createWallet();
    const arbiter = createWallet();

    // Deploy with seller=walletA
    const { signer: sellerSigner, pubKeyHex: sellerPubKey } = await createFundedWallet(provider);

    const contract = new RunarContract(artifact, [
      buyer.pubKeyHex,
      sellerPubKey,
      arbiter.pubKeyHex,
    ]);

    await contract.deploy(provider, sellerSigner, { satoshis: 5000 });

    // Call with wrong signer (walletB) — checkSig will fail on-chain
    const { signer: wrongSigner } = await createFundedWallet(provider);

    await expect(
      contract.call('releaseBySeller', [null], provider, wrongSigner),
    ).rejects.toThrow();
  });
});
