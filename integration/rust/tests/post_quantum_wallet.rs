//! PostQuantumWallet integration test — stateless contract with WOTS+ verification.
//!
//! ## How It Works
//!
//! PostQuantumWallet locks funds to a Winternitz One-Time Signature (WOTS+) public key.
//! WOTS+ is a hash-based post-quantum signature scheme — its security relies only on
//! the collision resistance of SHA-256, not on any number-theoretic assumption.
//!
//! ### Constructor
//!   - pubkey: ByteString — 64-byte hex (pubSeed[32] || pkRoot[32])
//!
//! ### Method: spend(msg: ByteString, sig: ByteString)
//!   - msg: the message to verify (arbitrary bytes; hashed internally to 32 bytes)
//!   - sig: 2,144-byte WOTS+ signature (67 chains × 32 bytes each)
//!
//! ### How WOTS+ Works
//!   1. Key gen: 67 random 32-byte secret keys, each chained 15 times (W=16)
//!   2. Public key = SHA-256(all 67 chain endpoints concatenated)
//!   3. Sign: hash message to 64 base-16 digits + 3 checksum digits,
//!      chain each sk[i] forward d[i] steps
//!   4. Verify: chain each sig[i] the remaining (15 - d[i]) steps,
//!      hash all endpoints, compare to pkRoot
//!
//! ### Script Size
//!   ~10 KB — modest because WOTS+ verification is iterative SHA-256 hashing.
//!
//! ### Important Notes
//!   - "One-time": reusing the same keypair for a different message leaks key material
//!   - No Sig param — hash-based signature, not ECDSA

use crate::helpers::*;
use crate::helpers::crypto::{wots_keygen, wots_pub_key_hex, wots_sign};
use runar_lang::sdk::{DeployOptions, RunarContract, SdkValue};

#[test]
#[ignore]
fn test_post_quantum_wallet_compile() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts");
    assert_eq!(artifact.contract_name, "PostQuantumWallet");
    assert!(!artifact.script.is_empty());
}

#[test]
#[ignore]
fn test_post_quantum_wallet_script_size() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts");
    let script_bytes = artifact.script.len() / 2;
    // WOTS+ scripts are typically ~10 KB
    assert!(script_bytes > 5000, "script too small: {} bytes", script_bytes);
    assert!(script_bytes < 50000, "script too large: {} bytes", script_bytes);
}

#[test]
#[ignore]
fn test_post_quantum_wallet_deploy() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Generate WOTS+ keypair from a deterministic seed
    let mut seed = vec![0u8; 32];
    seed[0] = 0x42;
    let mut pub_seed = vec![0u8; 32];
    pub_seed[0] = 0x01;
    let kp = wots_keygen(&seed, &pub_seed);

    // Constructor: (pubkey: ByteString) — 64-byte hex (pubSeed || pkRoot)
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(wots_pub_key_hex(&kp)),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 10000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
    assert_eq!(deploy_txid.len(), 64);
}

#[test]
#[ignore]
fn test_post_quantum_wallet_deploy_different_seed() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    let mut seed = vec![0u8; 32];
    seed[0] = 0x99;
    seed[1] = 0xAB;
    let mut pub_seed = vec![0u8; 32];
    pub_seed[0] = 0x02;
    let kp = wots_keygen(&seed, &pub_seed);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(wots_pub_key_hex(&kp)),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 10000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
}

/// Deploy and spend with a valid WOTS+ signature.
///
/// Steps:
///   1. Generate WOTS+ keypair from deterministic seed
///   2. Deploy contract with the public key
///   3. Sign a message with the WOTS+ secret key (2,144-byte signature)
///   4. Call spend(msg, sig) — the on-chain script verifies by:
///      - Hashing the message to 32 bytes
///      - Extracting 64 base-16 digits + 3 checksum digits
///      - Chaining each sig[i] forward (15 - d[i]) times
///      - Hashing all 67 endpoints, comparing to pkRoot
#[test]
#[ignore]
fn test_post_quantum_wallet_spend_valid_sig() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Generate WOTS+ keypair
    let mut seed = vec![0u8; 32];
    seed[0] = 0x42;
    let mut pub_seed = vec![0u8; 32];
    pub_seed[0] = 0x01;
    let kp = wots_keygen(&seed, &pub_seed);

    // Deploy
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(wots_pub_key_hex(&kp)),
    ]);
    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 10000,
            change_address: None,
        })
        .expect("deploy failed");

    // Sign a message
    let msg = b"spend this UTXO";
    let sig = wots_sign(msg, &kp.sk, &kp.pub_seed);

    // WOTS+ signature: 67 chains × 32 bytes = 2,144 bytes
    assert_eq!(sig.len(), 2144);

    // Convert to hex for SDK
    let msg_hex: String = msg.iter().map(|b| format!("{:02x}", b)).collect();
    let sig_hex: String = sig.iter().map(|b| format!("{:02x}", b)).collect();

    // Spend by calling spend(msg, sig)
    let (spend_txid, _tx) = contract
        .call(
            "spend",
            &[
                SdkValue::Bytes(msg_hex),
                SdkValue::Bytes(sig_hex),
            ],
            &mut provider,
            &*signer,
            None,
        )
        .expect("spend failed");
    assert!(!spend_txid.is_empty());
    assert_eq!(spend_txid.len(), 64);
}

#[test]
#[ignore]
fn test_post_quantum_wallet_tampered_sig_rejected() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Generate WOTS+ keypair
    let mut seed = vec![0u8; 32];
    seed[0] = 0x42;
    let mut pub_seed = vec![0u8; 32];
    pub_seed[0] = 0x01;
    let kp = wots_keygen(&seed, &pub_seed);

    // Deploy
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(wots_pub_key_hex(&kp)),
    ]);
    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 10000,
            change_address: None,
        })
        .expect("deploy failed");

    // Sign a message, then tamper with the signature
    let msg = b"spend this UTXO";
    let mut sig = wots_sign(msg, &kp.sk, &kp.pub_seed);
    sig[100] ^= 0xFF; // tamper byte 100

    let msg_hex: String = msg.iter().map(|b| format!("{:02x}", b)).collect();
    let sig_hex: String = sig.iter().map(|b| format!("{:02x}", b)).collect();

    let result = contract.call(
        "spend",
        &[
            SdkValue::Bytes(msg_hex),
            SdkValue::Bytes(sig_hex),
        ],
        &mut provider,
        &*signer,
        None,
    );
    assert!(result.is_err(), "spend with tampered WOTS+ signature should be rejected");
}

#[test]
#[ignore]
fn test_post_quantum_wallet_wrong_message_rejected() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/post-quantum-wallet/PostQuantumWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Generate WOTS+ keypair
    let mut seed = vec![0u8; 32];
    seed[0] = 0x42;
    let mut pub_seed = vec![0u8; 32];
    pub_seed[0] = 0x01;
    let kp = wots_keygen(&seed, &pub_seed);

    // Deploy
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(wots_pub_key_hex(&kp)),
    ]);
    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 10000,
            change_address: None,
        })
        .expect("deploy failed");

    // Sign "original message" but call spend with "different message"
    let original_msg = b"original message";
    let sig = wots_sign(original_msg, &kp.sk, &kp.pub_seed);

    let different_msg = b"different message";
    let msg_hex: String = different_msg.iter().map(|b| format!("{:02x}", b)).collect();
    let sig_hex: String = sig.iter().map(|b| format!("{:02x}", b)).collect();

    let result = contract.call(
        "spend",
        &[
            SdkValue::Bytes(msg_hex),
            SdkValue::Bytes(sig_hex),
        ],
        &mut provider,
        &*signer,
        None,
    );
    assert!(result.is_err(), "spend with wrong message should be rejected");
}
