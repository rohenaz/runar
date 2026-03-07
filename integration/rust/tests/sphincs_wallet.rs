//! SPHINCSWallet integration test — stateless contract with SLH-DSA-SHA2-128s verification.
//!
//! ## How It Works
//!
//! SPHINCSWallet locks funds to an SLH-DSA public key (FIPS 205, 128-bit post-quantum
//! security). Unlike WOTS+ (one-time), the same SLH-DSA keypair can sign many messages
//! because it uses a Merkle tree of WOTS+ keys internally.
//!
//! ### Constructor
//!   - pubkey: ByteString — 32-byte hex (PK.seed[16] || PK.root[16])
//!
//! ### Method: spend(msg: ByteString, sig: ByteString)
//!   - msg: the signed message (arbitrary bytes)
//!   - sig: 7,856-byte SLH-DSA-SHA2-128s signature
//!
//! ### Script Size
//!   ~188 KB — SLH-DSA verification requires computing multiple WOTS+ verifications
//!   and Merkle tree path checks.
//!
//! ### Test Approach
//!   Uses a pre-computed test vector from conformance/testdata/slhdsa-test-sig.hex
//!   with a known public key and message, avoiding the need for a full SLH-DSA
//!   signing library.

use crate::helpers::*;
use runar_lang::sdk::{DeployOptions, RunarContract, SdkValue};

/// Deterministic test public key (32 bytes hex: PK.seed || PK.root).
const SLHDSA_TEST_PK: &str = "00000000000000000000000000000000b618cb38f7f785488c9768f3a2972baf";
/// Message that was signed: "slh-dsa test vector" in hex.
const SLHDSA_TEST_MSG: &str = "736c682d647361207465737420766563746f72";

/// Load the pre-computed SLH-DSA test signature from conformance test data.
/// Returns the hex string (15,712 chars = 7,856 bytes).
fn load_test_signature() -> String {
    let manifest_dir = env!("CARGO_MANIFEST_DIR");
    let sig_path = std::path::Path::new(manifest_dir)
        .parent()
        .and_then(|p| p.parent())
        .expect("could not resolve project root")
        .join("conformance/testdata/slhdsa-test-sig.hex");
    std::fs::read_to_string(&sig_path)
        .unwrap_or_else(|e| panic!("failed to read {:?}: {}", sig_path, e))
        .trim()
        .to_string()
}

#[test]
#[ignore]
fn test_sphincs_wallet_compile() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts");
    assert_eq!(artifact.contract_name, "SPHINCSWallet");
    assert!(!artifact.script.is_empty());
}

#[test]
#[ignore]
fn test_sphincs_wallet_script_size() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts");
    let script_bytes = artifact.script.len() / 2;
    // SLH-DSA scripts are typically ~188 KB
    assert!(script_bytes > 100_000, "script too small: {} bytes", script_bytes);
    assert!(script_bytes < 500_000, "script too large: {} bytes", script_bytes);
}

#[test]
#[ignore]
fn test_sphincs_wallet_deploy() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Constructor: (pubkey: ByteString) — 32-byte hex
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(SLHDSA_TEST_PK.to_string()),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 50000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
    assert_eq!(deploy_txid.len(), 64);
}

#[test]
#[ignore]
fn test_sphincs_wallet_deploy_different_key() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    let other_pk = "aabbccdd00000000000000000000000011223344556677889900aabbccddeeff";
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(other_pk.to_string()),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 50000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
}

/// Deploy and spend with a valid SLH-DSA signature (pre-computed test vector).
///
/// The signature was generated offline with the matching private key.
/// SLH-DSA-SHA2-128s signatures are 7,856 bytes (FIPS 205 Table 2).
/// The on-chain script verifies by:
///   1. Parsing the sig into FORS trees + Hypertree layers
///   2. Computing WOTS+ public keys from signature chains
///   3. Verifying Merkle tree authentication paths
///   4. Comparing the reconstructed root against PK.root
#[test]
#[ignore]
fn test_sphincs_wallet_spend_valid_sig() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Deploy with the known test public key
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(SLHDSA_TEST_PK.to_string()),
    ]);
    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 50000,
            change_address: None,
        })
        .expect("deploy failed");

    // Load the pre-computed test signature
    let sig_hex = load_test_signature();
    assert_eq!(sig_hex.len() / 2, 7856, "SLH-DSA sig must be 7,856 bytes");

    // Spend by calling spend(msg, sig)
    let (spend_txid, _tx) = contract
        .call(
            "spend",
            &[
                SdkValue::Bytes(SLHDSA_TEST_MSG.to_string()),
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
fn test_sphincs_wallet_tampered_sig_rejected() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/sphincs-wallet/SPHINCSWallet.runar.ts");

    let mut provider = create_provider();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Deploy with the known test public key
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(SLHDSA_TEST_PK.to_string()),
    ]);
    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 50000,
            change_address: None,
        })
        .expect("deploy failed");

    // Load the pre-computed test signature and tamper byte 500
    let sig_hex = load_test_signature();
    let mut sig_bytes: Vec<u8> = (0..sig_hex.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(&sig_hex[i..i + 2], 16).unwrap())
        .collect();
    sig_bytes[500] ^= 0xFF;
    let tampered_sig_hex: String = sig_bytes.iter().map(|b| format!("{:02x}", b)).collect();

    let result = contract.call(
        "spend",
        &[
            SdkValue::Bytes(SLHDSA_TEST_MSG.to_string()),
            SdkValue::Bytes(tampered_sig_hex),
        ],
        &mut provider,
        &*signer,
        None,
    );
    assert!(result.is_err(), "spend with tampered SLH-DSA signature should be rejected");
}
