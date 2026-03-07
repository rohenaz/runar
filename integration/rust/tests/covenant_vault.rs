//! CovenantVault integration test — stateless contract with checkSig + checkPreimage.
//!
//! ## How It Works
//!
//! CovenantVault demonstrates a covenant pattern: it constrains HOW funds can be spent,
//! not just WHO can spend them. The contract checks:
//!   1. The owner's ECDSA signature (authentication via checkSig)
//!   2. The transaction preimage (via checkPreimage, which enables script-level
//!      inspection of the spending transaction)
//!   3. That the spending amount >= minAmount (covenant rule)
//!
//! ### What is checkPreimage / OP_PUSH_TX?
//!   checkPreimage verifies a BIP-143 sighash preimage against the spending transaction.
//!   This is implemented via the OP_PUSH_TX technique: the unlocking script pushes
//!   both a preimage (the raw BIP-143 serialization) and an ECDSA signature computed
//!   with private key k=1 (whose public key is the generator point G). The locking
//!   script verifies this signature against the preimage, which proves the preimage
//!   is genuine. Once verified, the script can inspect transaction fields.
//!
//! ### Constructor
//!   - owner: PubKey — the ECDSA public key that must sign to spend
//!   - recipient: Addr — the hash160 of the authorized recipient's public key
//!   - minAmount: bigint — minimum satoshis that must be sent to the recipient
//!
//! ### Method: spend(sig: Sig, amount: bigint, txPreimage: SigHashPreimage)
//!   The compiler inserts an implicit _opPushTxSig parameter before the declared params.
//!   The full unlocking script order is: <opPushTxSig> <sig> <amount> <txPreimage>
//!
//!   - sig: owner's ECDSA signature (auto-computed by SDK when SdkValue::Auto)
//!   - amount: satoshis to send to recipient (must be >= minAmount)
//!   - txPreimage: BIP-143 sighash preimage (auto-computed by SDK when SdkValue::Auto)

use crate::helpers::*;
use runar_lang::sdk::{DeployOptions, RunarContract, SdkValue};

#[test]
#[ignore]
fn test_covenant_vault_compile() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts");
    assert_eq!(artifact.contract_name, "CovenantVault");
}

#[test]
#[ignore]
fn test_covenant_vault_deploy() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts");

    let mut provider = create_provider();
    let owner = create_wallet();
    let recipient = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Constructor: (owner: PubKey, recipient: Addr, minAmount: bigint)
    // Addr is a pubKeyHash (20-byte hash160)
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner.pub_key_hex),
        SdkValue::Bytes(recipient.pub_key_hash),
        SdkValue::Int(1000),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
    assert_eq!(deploy_txid.len(), 64);
}

#[test]
#[ignore]
fn test_covenant_vault_deploy_zero_min_amount() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts");

    let mut provider = create_provider();
    let owner = create_wallet();
    let recipient = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner.pub_key_hex),
        SdkValue::Bytes(recipient.pub_key_hash),
        SdkValue::Int(0),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
}

#[test]
#[ignore]
fn test_covenant_vault_deploy_large_min_amount() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts");

    let mut provider = create_provider();
    let owner = create_wallet();
    let recipient = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner.pub_key_hex),
        SdkValue::Bytes(recipient.pub_key_hash),
        SdkValue::Int(100_000_000), // 1 BTC in satoshis
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
}

#[test]
#[ignore]
fn test_covenant_vault_deploy_same_key_owner_recipient() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts");

    let mut provider = create_provider();
    let both = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(both.pub_key_hex),
        SdkValue::Bytes(both.pub_key_hash),
        SdkValue::Int(500),
    ]);

    let (deploy_txid, _tx) = contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");
    assert!(!deploy_txid.is_empty());
}

/// Deploy and spend with valid owner signature, preimage, and amount >= minAmount.
///
/// Steps:
///   1. Create owner wallet (will be the signer — their ECDSA key must match constructor)
///   2. Deploy with (ownerPubKey, recipientPubKeyHash, minAmount=1000)
///   3. Call spend(Auto, 2000, Auto):
///      - Auto Sig → SDK auto-computes ECDSA signature from signer's private key
///      - 2000     → amount (>= minAmount of 1000)
///      - Auto SigHashPreimage → SDK auto-computes BIP-143 preimage and _opPushTxSig
///   4. The SDK builds the unlocking script: <opPushTxSig> <sig> <amount> <txPreimage>
///   5. On-chain, the script verifies checkSig(sig, owner), checkPreimage(txPreimage),
///      and asserts amount >= minAmount
#[test]
#[ignore]
fn test_covenant_vault_spend_valid() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts");

    let mut provider = create_provider();
    let recipient = create_wallet();

    // Owner must be the signer — their ECDSA key must match constructor's owner param
    let (owner_signer, owner_wallet) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner_wallet.pub_key_hex.clone()),
        SdkValue::Bytes(recipient.pub_key_hash),
        SdkValue::Int(1000), // minAmount
    ]);

    contract
        .deploy(&mut provider, &*owner_signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    // spend(sig=Auto, amount=2000, txPreimage=Auto)
    // SDK auto-computes both Sig and SigHashPreimage from the spending transaction
    let (spend_txid, _tx) = contract
        .call(
            "spend",
            &[
                SdkValue::Auto,      // ECDSA signature auto-computed from owner's key
                SdkValue::Int(2000),  // amount >= minAmount (1000)
                SdkValue::Auto,      // BIP-143 preimage auto-computed
            ],
            &mut provider,
            &*owner_signer,
            None,
        )
        .expect("spend failed");
    assert!(!spend_txid.is_empty());
    assert_eq!(spend_txid.len(), 64);
}

#[test]
#[ignore]
fn test_covenant_vault_below_min_amount_rejected() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/covenant-vault/CovenantVault.runar.ts");

    let mut provider = create_provider();
    let recipient = create_wallet();

    // Owner must be the signer
    let (owner_signer, owner_wallet) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(owner_wallet.pub_key_hex.clone()),
        SdkValue::Bytes(recipient.pub_key_hash),
        SdkValue::Int(1000), // minAmount=1000
    ]);

    contract
        .deploy(&mut provider, &*owner_signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    // spend with amount=500 (< minAmount of 1000) — should be rejected
    let result = contract.call(
        "spend",
        &[
            SdkValue::Auto,     // ECDSA signature
            SdkValue::Int(500), // amount < minAmount
            SdkValue::Auto,     // BIP-143 preimage
        ],
        &mut provider,
        &*owner_signer,
        None,
    );
    assert!(result.is_err(), "spend with amount below minAmount should be rejected");
}
