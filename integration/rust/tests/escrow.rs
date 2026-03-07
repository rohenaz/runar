//! Escrow integration test — stateless contract with checkSig.
//!
//! Escrow locks funds and allows release or refund via four methods, each
//! requiring a signature from the appropriate party. We verify compile + deploy.

use crate::helpers::*;
use runar_lang::sdk::{DeployOptions, RunarContract, SdkValue};

#[test]
#[ignore]
fn test_escrow_compile() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts");
    assert_eq!(artifact.contract_name, "Escrow");
}

#[test]
#[ignore]
fn test_escrow_deploy_three_pubkeys() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts");

    let mut provider = create_provider();
    let buyer = create_wallet();
    let seller = create_wallet();
    let arbiter = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Constructor: (buyer: PubKey, seller: PubKey, arbiter: PubKey)
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(buyer.pub_key_hex),
        SdkValue::Bytes(seller.pub_key_hex),
        SdkValue::Bytes(arbiter.pub_key_hex),
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
fn test_escrow_deploy_same_key_multiple_roles() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts");

    let mut provider = create_provider();
    let buyer_and_arbiter = create_wallet();
    let seller = create_wallet();
    let (signer, _wallet) = create_funded_wallet(&mut provider);

    // Same key as both buyer and arbiter
    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(buyer_and_arbiter.pub_key_hex.clone()),
        SdkValue::Bytes(seller.pub_key_hex),
        SdkValue::Bytes(buyer_and_arbiter.pub_key_hex),
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
fn test_escrow_release_by_seller() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts");

    let mut provider = create_provider();
    // Seller is also our funded signer, so the auto-computed sig matches
    let (signer, seller_wallet) = create_funded_wallet(&mut provider);
    let buyer = create_wallet();
    let arbiter = create_wallet();

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(buyer.pub_key_hex),
        SdkValue::Bytes(seller_wallet.pub_key_hex.clone()),
        SdkValue::Bytes(arbiter.pub_key_hex),
    ]);

    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    // Auto Sig is computed by the SDK from the signer (who is the seller)
    let (call_txid, _tx) = contract
        .call(
            "releaseBySeller",
            &[SdkValue::Auto],
            &mut provider,
            &*signer,
            None,
        )
        .expect("releaseBySeller failed");
    assert!(!call_txid.is_empty());
    assert_eq!(call_txid.len(), 64);
}

#[test]
#[ignore]
fn test_escrow_release_by_arbiter() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts");

    let mut provider = create_provider();
    let buyer = create_wallet();
    let seller = create_wallet();
    // Arbiter is the funded signer so the auto-computed sig matches
    let (signer, arbiter_wallet) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(buyer.pub_key_hex),
        SdkValue::Bytes(seller.pub_key_hex),
        SdkValue::Bytes(arbiter_wallet.pub_key_hex.clone()),
    ]);

    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    let (call_txid, _tx) = contract
        .call(
            "releaseByArbiter",
            &[SdkValue::Auto],
            &mut provider,
            &*signer,
            None,
        )
        .expect("releaseByArbiter failed");
    assert!(!call_txid.is_empty());
    assert_eq!(call_txid.len(), 64);
}

#[test]
#[ignore]
fn test_escrow_refund_to_buyer() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts");

    let mut provider = create_provider();
    let seller = create_wallet();
    let arbiter = create_wallet();
    // Buyer is the funded signer so the auto-computed sig matches
    let (signer, buyer_wallet) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(buyer_wallet.pub_key_hex.clone()),
        SdkValue::Bytes(seller.pub_key_hex),
        SdkValue::Bytes(arbiter.pub_key_hex),
    ]);

    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    let (call_txid, _tx) = contract
        .call(
            "refundToBuyer",
            &[SdkValue::Auto],
            &mut provider,
            &*signer,
            None,
        )
        .expect("refundToBuyer failed");
    assert!(!call_txid.is_empty());
    assert_eq!(call_txid.len(), 64);
}

#[test]
#[ignore]
fn test_escrow_refund_by_arbiter() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts");

    let mut provider = create_provider();
    let buyer = create_wallet();
    let seller = create_wallet();
    // Arbiter is the funded signer so the auto-computed sig matches
    let (signer, arbiter_wallet) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(buyer.pub_key_hex),
        SdkValue::Bytes(seller.pub_key_hex),
        SdkValue::Bytes(arbiter_wallet.pub_key_hex.clone()),
    ]);

    contract
        .deploy(&mut provider, &*signer, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    let (call_txid, _tx) = contract
        .call(
            "refundByArbiter",
            &[SdkValue::Auto],
            &mut provider,
            &*signer,
            None,
        )
        .expect("refundByArbiter failed");
    assert!(!call_txid.is_empty());
    assert_eq!(call_txid.len(), 64);
}

#[test]
#[ignore]
fn test_escrow_wrong_signer_rejected() {
    skip_if_no_node();

    let artifact = compile_contract("examples/ts/escrow/Escrow.runar.ts");

    let mut provider = create_provider();
    let buyer = create_wallet();
    let arbiter = create_wallet();
    // Deploy with seller=walletA
    let (signer_a, wallet_a) = create_funded_wallet(&mut provider);

    let mut contract = RunarContract::new(artifact, vec![
        SdkValue::Bytes(buyer.pub_key_hex),
        SdkValue::Bytes(wallet_a.pub_key_hex.clone()),
        SdkValue::Bytes(arbiter.pub_key_hex),
    ]);

    contract
        .deploy(&mut provider, &*signer_a, &DeployOptions {
            satoshis: 5000,
            change_address: None,
        })
        .expect("deploy failed");

    // Call releaseBySeller with a different signer — should be rejected
    let (signer_b, _wallet_b) = create_funded_wallet(&mut provider);
    let result = contract.call(
        "releaseBySeller",
        &[SdkValue::Auto],
        &mut provider,
        &*signer_b,
        None,
    );
    assert!(result.is_err(), "releaseBySeller with wrong signer should be rejected");
}
